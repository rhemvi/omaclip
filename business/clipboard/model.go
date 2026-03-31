// Package clipboard manages clipboard history and monitors the system clipboard for new entries.
package clipboard

import "time"

// ClipboardEntry represents a single item captured from the clipboard.
type ClipboardEntry struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
