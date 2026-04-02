// Package handlers contains HTTP handlers for the sync server.
package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"

	"clipmaster/business/clipboard"
	"clipmaster/business/passphrase"
)

// ClipboardHandler holds dependencies for all HTTP handlers.
type ClipboardHandler struct {
	Monitor        *clipboard.Monitor
	MaxHistory     int
	PassphraseStore *passphrase.Store
}

// GetClipboard returns the last N clipboard entries as JSON.
// Returns 401 if the X-Clipmaster-Pass header is missing or incorrect.
func (h *ClipboardHandler) GetClipboard(w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Clipmaster-Pass")), []byte(h.PassphraseStore.Get())) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	all := h.Monitor.GetHistory()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(all[:min(h.MaxHistory, len(all))]) //nolint:errcheck
}
