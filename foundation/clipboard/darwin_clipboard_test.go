package clipboard

import "testing"

func TestParseClipboardTypeSize(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   int64
	}{
		{"PNGf", "«class PNGf», 10845271\n", 10845271},
		{"JPEG", "JPEG picture, 1522509\n", 1522509},
		{"utf8", "«class utf8», 42\n", 42},
		{"furl", "«class furl», 100\n", 100},
		{"empty output", "", 0},
		{"no comma", "«class PNGf»", 0},
		{"no number", "«class PNGf», \n", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTypeSize(tt.output)
			if got != tt.want {
				t.Errorf("parseTypeSize(%q) = %d, want %d", tt.output, got, tt.want)
			}
		})
	}
}
