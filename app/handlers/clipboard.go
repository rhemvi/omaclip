// Package handlers contains HTTP handlers for the sync server.
package handlers

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/rhemvi/omaclip/business/clipboard"
	"github.com/rhemvi/omaclip/business/passphrase"
)

// ClipboardHandler holds dependencies for all HTTP handlers.
type ClipboardHandler struct {
	Monitor         *clipboard.Monitor
	MaxHistory      int
	PassphraseStore *passphrase.Store
}

// RequirePassphrase returns middleware that validates the X-Omaclip-Pass header.
func RequirePassphrase(store *passphrase.Store, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Omaclip-Pass")), []byte(store.Hash())) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// GetClipboard returns the last N clipboard entries as JSON.
// Image entries are included but without their ImageData payload.
func (h *ClipboardHandler) GetClipboard(w http.ResponseWriter, r *http.Request) {
	all := h.Monitor.GetHistory()
	limit := min(h.MaxHistory, len(all))
	entries := all[:limit]

	stripped := make([]clipboard.ClipboardEntry, 0, len(entries))
	for _, e := range entries {
		if e.ContentType == "image-rejected" {
			continue
		}
		// Will fetch images with dedicated endpoint, so strip data from this response to save bandwidth
		e.ImageData = ""
		stripped = append(stripped, e)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stripped) //nolint:errcheck
}

// GetClipboardImage returns the raw image bytes for a specific clipboard entry.
func (h *ClipboardHandler) GetClipboardImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	entry, ok := h.Monitor.GetEntry(id)
	if !ok || entry.ContentType != "image" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	imgBytes, err := base64.StdEncoding.DecodeString(entry.ImageData)
	if err != nil {
		http.Error(w, "corrupt image data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", entry.ImageMimeType)
	w.Write(imgBytes) //nolint:errcheck
}
