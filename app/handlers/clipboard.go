// Package handlers contains HTTP handlers for the sync server.
package handlers

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"clipmaster/business/clipboard"
	"clipmaster/business/passphrase"
)

// ClipboardHandler holds dependencies for all HTTP handlers.
type ClipboardHandler struct {
	Monitor         *clipboard.Monitor
	MaxHistory      int
	PassphraseStore *passphrase.Store
}

// RequirePassphrase returns middleware that validates the X-Clipmaster-Pass header.
func RequirePassphrase(store *passphrase.Store, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Clipmaster-Pass")), []byte(store.Hash())) != 1 {
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

	stripped := make([]clipboard.ClipboardEntry, len(entries))
	for i, e := range entries {
		stripped[i] = e
		stripped[i].ImageData = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stripped) //nolint:errcheck
}

// GetClipboardImage returns the raw PNG bytes for a specific clipboard entry.
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

	w.Header().Set("Content-Type", "image/png")
	w.Write(imgBytes) //nolint:errcheck
}
