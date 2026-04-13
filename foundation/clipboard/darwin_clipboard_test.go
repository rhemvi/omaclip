package clipboard

import "testing"

func TestTypeSize(t *testing.T) {
	info := "«class PNGf», 10845271, «class utf8», 42, JPEG picture, 1522509, «class furl», 100"

	tests := []struct {
		name     string
		typeName string
		want     int64
	}{
		{"PNGf", "PNGf", 10845271},
		{"JPEG", "JPEG picture", 1522509},
		{"utf8", "utf8", 42},
		{"furl", "furl", 100},
		{"not found", "TIFF", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := typeSize(info, tt.typeName); got != tt.want {
				t.Errorf("typeSize(%q) = %d, want %d", tt.typeName, got, tt.want)
			}
		})
	}
}

func TestTypeSize_EmptyInfo(t *testing.T) {
	if got := typeSize("", "PNGf"); got != 0 {
		t.Errorf("typeSize on empty = %d, want 0", got)
	}
}

func TestTypeSize_NoNumber(t *testing.T) {
	if got := typeSize("«class PNGf», , JPEG", "PNGf"); got != 0 {
		t.Errorf("typeSize with missing number = %d, want 0", got)
	}
}
