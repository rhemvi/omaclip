package clipboard

import (
	"testing"
)

func mockAvailable(bins ...string) func(string) bool {
	set := make(map[string]bool)
	for _, b := range bins {
		set[b] = true
	}
	return func(bin string) bool {
		return set[bin]
	}
}

func TestNewReaderWriter(t *testing.T) {
	tests := []struct {
		name        string
		available   []string
		wantBackend string
		wantErr     bool
	}{
		{
			name:        "wayland selected when wl-paste available",
			available:   []string{"wl-paste"},
			wantBackend: "wayland (wl-paste)",
		},
		{
			name:        "wayland preferred over xclip",
			available:   []string{"wl-paste", "xclip"},
			wantBackend: "wayland (wl-paste)",
		},
		{
			name:        "xclip selected when no wl-paste",
			available:   []string{"xclip"},
			wantBackend: "x11 (xclip)",
		},
		{
			name:        "xsel selected when no wl-paste or xclip",
			available:   []string{"xsel"},
			wantBackend: "x11 (xsel)",
		},
		{
			name:        "darwin selected when osascript and pbpaste available",
			available:   []string{"osascript", "pbpaste"},
			wantBackend: "darwin (osascript+pbpaste)",
		},
		{
			name:    "error when only osascript available",
			available: []string{"osascript"},
			wantErr: true,
		},
		{
			name:    "error when only pbpaste available",
			available: []string{"pbpaste"},
			wantErr: true,
		},
		{
			name:    "error when nothing available",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := availableFn
			availableFn = mockAvailable(tt.available...)
			t.Cleanup(func() { availableFn = original })

			_, _, backend, err := NewReaderWriter()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if backend != tt.wantBackend {
				t.Errorf("backend = %q, want %q", backend, tt.wantBackend)
			}
		})
	}
}
