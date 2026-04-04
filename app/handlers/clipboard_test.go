package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"clipmaster/business/passphrase"
)

func TestRequirePassphrase_Unauthorized(t *testing.T) {
	store := &passphrase.Store{}
	store.Set("correctpassphrase")

	dummy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequirePassphrase(store, dummy)

	tests := []struct {
		name       string
		headerVal  string
		wantStatus int
	}{
		{"missing header", "", http.StatusUnauthorized},
		{"wrong passphrase", "wronghash", http.StatusUnauthorized},
		{"correct hash", store.Hash(), http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/clipboard", nil)
			if tt.headerVal != "" {
				req.Header.Set("X-Clipmaster-Pass", tt.headerVal)
			}
			rec := httptest.NewRecorder()
			handler(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
