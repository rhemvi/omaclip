// Package clipboard manages clipboard history and monitors the system clipboard for new entries.
package clipboard

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

const maxImageBytes = 5 * 1024 * 1024

// Reader abstracts clipboard reading so different implementations can be swapped in.
type Reader interface {
	GetText() (string, error)
	GetImage() ([]byte, error)
}

// Writer abstracts clipboard writing so different implementations can be swapped in.
type Writer interface {
	SetText(text string) error
	SetImage(pngData []byte) error
}

// Monitor polls the system clipboard and maintains an in-memory history.
type Monitor struct {
	mu           sync.RWMutex
	history      []ClipboardEntry
	maxHistory   int
	pollInterval time.Duration
	lastSeen     string
	lastSeenHash string
	cancel       context.CancelFunc
	reader       Reader
	writer       Writer
	OnNewEntry   func(ClipboardEntry)
}

// NewMonitor creates a Monitor with the given reader, writer, capacity, and poll interval.
func NewMonitor(reader Reader, writer Writer, maxHistory int, pollInterval time.Duration) *Monitor {
	return &Monitor{
		reader:       reader,
		writer:       writer,
		maxHistory:   maxHistory,
		pollInterval: pollInterval,
	}
}

// Start begins polling the clipboard in a background goroutine.
func (m *Monitor) Start(ctx context.Context) {
	ctx, m.cancel = context.WithCancel(ctx)
	go m.poll(ctx)
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

// CopyItem writes the entry with the given ID back to the system clipboard.
func (m *Monitor) CopyItem(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, entry := range m.history {
		if entry.ID == id {
			if entry.ContentType == "image" {
				imgBytes, err := base64.StdEncoding.DecodeString(entry.ImageData)
				if err != nil {
					return fmt.Errorf("decoding image data: %w", err)
				}
				if err := m.writer.SetImage(imgBytes); err != nil {
					return err
				}
				m.lastSeenHash = sha256Hex(imgBytes)
				m.lastSeen = ""
				return nil
			}
			if err := m.writer.SetText(entry.Content); err != nil {
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
	if err := m.writer.SetText(text); err != nil {
		return err
	}
	m.lastSeen = text
	return nil
}

// CopyImage writes base64-encoded PNG data to the system clipboard without adding it to history.
func (m *Monitor) CopyImage(imageDataBase64 string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	imgBytes, err := base64.StdEncoding.DecodeString(imageDataBase64)
	if err != nil {
		return fmt.Errorf("decoding image data: %w", err)
	}
	if err := m.writer.SetImage(imgBytes); err != nil {
		return err
	}
	m.lastSeenHash = sha256Hex(imgBytes)
	m.lastSeen = ""
	return nil
}

// poll runs the clipboard polling loop until ctx is cancelled.
func (m *Monitor) poll(ctx context.Context) {
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			text, err := m.reader.GetText()
			if err == nil && text != "" && text != m.lastSeen {
				m.lastSeen = text
				m.lastSeenHash = ""
				m.addEntry(ClipboardEntry{
					ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
					Content:     text,
					ContentType: "text",
					Timestamp:   time.Now(),
				})
				continue
			}

			imgData, err := m.reader.GetImage()
			if err != nil || len(imgData) == 0 || len(imgData) > maxImageBytes {
				continue
			}
			hash := sha256Hex(imgData)
			if hash == m.lastSeenHash {
				continue
			}
			m.lastSeenHash = hash
			m.lastSeen = ""
			m.addEntry(ClipboardEntry{
				ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
				ContentType: "image",
				ImageData:   base64.StdEncoding.EncodeToString(imgData),
				Timestamp:   time.Now(),
			})
		}
	}
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
