package app

import (
	"clipmaster/app/handlers"
	bsync "clipmaster/business/sync"
)

func registerRoutes(s *bsync.Server, h *handlers.ClipboardHandler) {
	auth := handlers.RequirePassphrase
	s.Handle("GET /api/clipboard", auth(h.PassphraseStore, h.GetClipboard))
	s.Handle("GET /api/clipboard/{id}/image", auth(h.PassphraseStore, h.GetClipboardImage))
}
