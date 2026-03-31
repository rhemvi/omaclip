// Package clipboard manages clipboard history and monitors the system clipboard for new entries.
package clipboard

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Reader abstracts clipboard reading so different implementations can be swapped in.
type Reader interface {
	GetText() (string, error)
}

// Writer abstracts clipboard writing so different implementations can be swapped in.
type Writer interface {
	SetText(text string) error
}

// Monitor polls the system clipboard and maintains an in-memory history.
type Monitor struct {
	mu           sync.RWMutex
	history      []ClipboardEntry
	maxHistory   int
	pollInterval time.Duration
	lastSeen     string
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

// CopyItem writes the entry with the given ID back to the system clipboard.
func (m *Monitor) CopyItem(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, entry := range m.history {
		if entry.ID == id {
			if err := m.writer.SetText(entry.Content); err != nil {
				return err
			}
			m.lastSeen = entry.Content
			return nil
		}
	}
	return fmt.Errorf("entry %s not found", id)
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
			if err != nil || text == "" || text == m.lastSeen {
				continue
			}
			m.lastSeen = text
			m.addEntry(text)
		}
	}
}

// addEntry appends a new entry to history, trimming to maxHistory, then notifies the callback.
func (m *Monitor) addEntry(content string) {
	m.mu.Lock()

	entry := ClipboardEntry{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Content:   content,
		Timestamp: time.Now(),
	}

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
