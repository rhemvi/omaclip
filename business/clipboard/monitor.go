// Package clipboard manages clipboard history and monitors the system clipboard for new entries.
package clipboard

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"net/http"
	"sync"
	"time"

	_ "image/gif"
	_ "image/jpeg"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// ErrNoConversion is returned by toPNG when the image format is not supported for PNG conversion.
var ErrNoConversion = errors.New("image format not supported for PNG conversion")

// Reader abstracts clipboard reading so different implementations can be swapped in.
type Reader interface {
	GetText(ctx context.Context) (string, error)
	GetImage(ctx context.Context) ([]byte, error)
}

// Writer abstracts clipboard writing so different implementations can be swapped in.
type Writer interface {
	SetText(ctx context.Context, text string) error
	SetImage(ctx context.Context, data []byte, mimeType string) error
}

// cmdTimeout is the maximum time allowed for a single clipboard read or write operation.
const cmdTimeout = 2 * time.Second

// Watcher is optionally implemented by clipboard backends that support event-driven change notifications instead of polling.
type Watcher interface {
	Watch(ctx context.Context, notify chan<- struct{}) error
}

// Monitor polls the system clipboard and maintains an in-memory history.
type Monitor struct {
	mu               sync.RWMutex
	log              *slog.Logger
	history          []ClipboardEntry
	maxHistory       int
	maxPngImageMB    int
	maxNonPngImageMB int
	pollInterval     time.Duration
	lastSeen         string
	lastSeenHash     string
	cancel           context.CancelFunc
	reader           Reader
	writer           Writer
	OnNewEntry       func(ClipboardEntry)
}

// NewMonitor creates a Monitor with the given reader, writer, capacity, and poll interval.
func NewMonitor(
	log *slog.Logger,
	reader Reader,
	writer Writer,
	maxHistory int,
	maxPngImageMB int,
	maxNonPngImageMB int,
	pollInterval time.Duration,
) *Monitor {
	return &Monitor{
		log:              log,
		reader:           reader,
		writer:           writer,
		maxHistory:       maxHistory,
		maxPngImageMB:    maxPngImageMB,
		maxNonPngImageMB: maxNonPngImageMB,
		pollInterval:     pollInterval,
	}
}

// Start begins monitoring the clipboard in a background goroutine. If the reader implements Watcher and watching succeeds, event-driven watching is used; otherwise falls back to polling.
func (m *Monitor) Start(ctx context.Context) {
	ctx, m.cancel = context.WithCancel(ctx)

	var watchCh chan struct{}
	if watcher, ok := m.reader.(Watcher); ok {
		notify := make(chan struct{}, 1)
		if err := watcher.Watch(ctx, notify); err != nil {
			m.log.Warn("clipboard watcher failed, falling back to polling", "error", err)
		} else {
			watchCh = notify
		}
	} else {
		m.log.Warn("clipboard watch is not supported with this driver, falling back to polling")
	}

	go m.poll(ctx, watchCh)
}

// Stop halts the polling goroutine.
func (m *Monitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
}

// GetHistory returns all clipboard entries in reverse-chronological order.
func (m *Monitor) GetHistory() []ClipboardEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ClipboardEntry, len(m.history))
	for i, entry := range m.history {
		result[len(m.history)-1-i] = entry
	}
	return result
}

// GetEntry returns a single entry by ID.
func (m *Monitor) GetEntry(id string) (ClipboardEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, entry := range m.history {
		if entry.ID == id {
			return entry, true
		}
	}
	return ClipboardEntry{}, false
}

// CopyImage writes base64-encoded image data to the system clipboard without adding it to history.
func (m *Monitor) CopyImage(imageDataBase64 string, mimeType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	imgBytes, err := base64.StdEncoding.DecodeString(imageDataBase64)
	if err != nil {
		return fmt.Errorf("decoding image data: %w", err)
	}
	outBytes, convErr := toPNG(m.log, imgBytes)
	outMime := "image/png"
	if errors.Is(convErr, ErrNoConversion) {
		outMime = mimeType
	}
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	if err := m.writer.SetImage(ctx, outBytes, outMime); err != nil {
		return err
	}
	m.lastSeenHash = sha256Hex(outBytes)
	m.lastSeen = ""
	return nil
}

// CopyItem writes the entry with the given ID back to the system clipboard.
func (m *Monitor) CopyItem(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	for _, entry := range m.history {
		if entry.ID == id {
			if entry.ContentType == "image" {
				imgBytes, err := base64.StdEncoding.DecodeString(entry.ImageData)
				if err != nil {
					return fmt.Errorf("decoding image data: %w", err)
				}
				outBytes, convErr := toPNG(m.log, imgBytes)
				outMime := "image/png"
				if errors.Is(convErr, ErrNoConversion) {
					outMime = entry.ImageMimeType
				}
				if err := m.writer.SetImage(ctx, outBytes, outMime); err != nil {
					return err
				}
				m.lastSeenHash = sha256Hex(outBytes)
				m.lastSeen = ""
				return nil
			}
			if err := m.writer.SetText(ctx, entry.Content); err != nil {
				return err
			}
			m.lastSeen = entry.Content
			m.lastSeenHash = ""
			return nil
		}
	}
	return fmt.Errorf("entry %s not found", id)
}

// CopyText writes arbitrary text to the system clipboard without adding it to history.
func (m *Monitor) CopyText(text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	if err := m.writer.SetText(ctx, text); err != nil {
		return err
	}
	m.lastSeen = text
	return nil
}

// poll runs the clipboard monitoring loop until ctx is cancelled. If watchCh is non-nil, clipboard reads are triggered by watch events instead of a ticker. Falls back to polling if the watch channel is closed.
func (m *Monitor) poll(ctx context.Context, watchCh <-chan struct{}) {
	var tickerCh <-chan time.Time
	var ticker *time.Ticker
	if watchCh == nil {
		ticker = time.NewTicker(m.pollInterval)
		tickerCh = ticker.C
		defer ticker.Stop()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-tickerCh:
			m.readClipboard(ctx)
		case _, ok := <-watchCh:
			if !ok {
				m.log.Warn("clipboard watcher stopped, falling back to polling")
				watchCh = nil
				ticker = time.NewTicker(m.pollInterval)
				tickerCh = ticker.C
				defer ticker.Stop()
				continue
			}
			m.readClipboard(ctx)
		}
	}
}

// readClipboard checks the system clipboard for new text or image content and adds it to history.
func (m *Monitor) readClipboard(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, cmdTimeout)
	defer cancel()

	text, err := m.reader.GetText(ctx)
	textChanged := err == nil && text != "" && text != m.lastSeen

	imgData, imgErr := m.reader.GetImage(ctx)
	var imgHash string
	if imgErr == nil && len(imgData) > 0 {
		imgHash = sha256Hex(imgData)
	}
	imgChanged := imgHash != "" && imgHash != m.lastSeenHash

	if textChanged {
		m.lastSeen = text
		m.lastSeenHash = ""
		m.addEntry(ClipboardEntry{
			ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
			Content:     text,
			ContentType: "text",
			Timestamp:   time.Now(),
		})
	}

	if imgChanged {
		m.lastSeenHash = imgHash
		if !textChanged {
			m.lastSeen = ""
		}
		mimeType := http.DetectContentType(imgData)
		limitName := "max_png_image_mb"
		maxMB := m.maxPngImageMB
		if mimeType != "image/png" {
			limitName = "max_non_png_image_mb"
			maxMB = m.maxNonPngImageMB
		}
		sizeMB := fmt.Sprintf("%.2f", float64(len(imgData))/1024/1024)
		if len(imgData) > maxMB*1024*1024 {
			m.log.Warn(
				"image rejected: exceeds size limit",
				"size_mb", sizeMB,
				"limit", limitName,
				"limit_mb", maxMB,
				"mime_type", mimeType,
			)
			m.addEntry(ClipboardEntry{
				ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
				ContentType: "image-rejected",
				Content:     fmt.Sprintf("Image rejected: %s MB exceeds %d MB limit (%s)", sizeMB, maxMB, mimeType),
				Timestamp:   time.Now(),
			})
		} else {
			m.addEntry(ClipboardEntry{
				ID:            fmt.Sprintf("%d", time.Now().UnixNano()),
				ContentType:   "image",
				ImageData:     base64.StdEncoding.EncodeToString(imgData),
				ImageMimeType: mimeType,
				Timestamp:     time.Now(),
			})
		}
	}
}

// toPNG decodes image bytes of any supported format and re-encodes them as PNG for clipboard compatibility.
// If the data is already PNG, it is returned as-is with a nil error.
// Returns ErrNoConversion if the format is not supported or encoding fails; the original bytes are returned unchanged.
func toPNG(log *slog.Logger, data []byte) ([]byte, error) {
	sizeMB := fmt.Sprintf("%.2f", float64(len(data))/1024/1024)
	if http.DetectContentType(data) == "image/png" {
		log.Info("copy out: image already PNG, skipping conversion", "size_mb", sizeMB)
		return data, nil
	}
	start := time.Now()
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Warn("copy out: image format not supported for PNG conversion, using original data",
			"mime_type", http.DetectContentType(data),
			"error", err,
		)
		return data, ErrNoConversion
	}
	var buf bytes.Buffer
	enc := &png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, img); err != nil {
		log.Warn("copy out: failed to encode image as PNG, using original data", "error", err)
		return data, ErrNoConversion
	}
	log.Info("copy out: converted image to PNG for clipboard compatibility",
		"original_format", format,
		"original_mb", sizeMB,
		"png_mb", fmt.Sprintf("%.2f", float64(buf.Len())/1024/1024),
		"duration", time.Since(start),
	)
	return buf.Bytes(), nil
}

// addEntry appends a new entry to history, trimming to maxHistory, then notifies the callback.
func (m *Monitor) addEntry(entry ClipboardEntry) {
	m.mu.Lock()

	m.history = append(m.history, entry)
	if len(m.history) > m.maxHistory {
		m.history = m.history[len(m.history)-m.maxHistory:]
	}

	cb := m.OnNewEntry
	m.mu.Unlock()

	if cb != nil {
		cb(entry)
	}
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
