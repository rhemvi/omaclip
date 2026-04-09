// Package clipboard manages clipboard history and monitors the system clipboard for new entries.
package clipboard

import "time"

// ClipboardEntry represents a single item captured from the clipboard.
type ClipboardEntry struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	ContentType string    `json:"contentType"`
	ImageData     string    `json:"imageData,omitempty"`
	ImageMimeType string    `json:"imageMimeType,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}
