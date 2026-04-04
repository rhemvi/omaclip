package clipboard

import (
	"errors"
	"os/exec"
)

var (
	ErrNoClipAvailable = errors.New("no supported clipboard binary found (tried: wl-paste, xclip, xsel, pbpaste)")
	ErrNotImplemented  = errors.New("not implemented")
)

// Reader abstracts clipboard reading across platform backends.
type Reader interface {
	GetText() (string, error)
	GetImage() ([]byte, error)
}

// Writer abstracts clipboard writing across platform backends.
type Writer interface {
	SetText(text string) error
	SetImage(pngData []byte) error
}

// NewReaderWriter returns the first available clipboard reader and writer by probing known binaries in order:
// wl-paste (Wayland) → xclip (X11) → xsel (X11) → pbpaste (macOS).
func NewReaderWriter() (Reader, Writer, error) {
	switch {
	case available("wl-paste"):
		w := WaylandClipboard{}
		return w, w, nil
	case available("xclip"):
		x := XclipClipboard{}
		return x, x, nil
	case available("xsel"):
		s := XselClipboard{}
		return s, s, nil
	case available("pbpaste"):
		if available("osascript") {
			o := DarwinOsascriptClipboard{}
			return o, o, nil
		}
		d := DarwinClipboard{}
		return d, d, nil
	default:
		return nil, nil, ErrNoClipAvailable
	}
}

func available(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}
