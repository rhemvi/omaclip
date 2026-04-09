package clipboard

import (
	"errors"
	"os/exec"
)

var (
	ErrNoClipAvailable = errors.New("no supported clipboard binary found (tried: wl-paste, xclip, xsel, osascript+pbpaste)")
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
	SetImage(data []byte, mimeType string) error
}

// NewReaderWriter returns the first available clipboard reader and writer by probing known binaries in order:
// wl-paste (Wayland) → xclip (X11) → xsel (X11) → osascript+pbpaste (macOS).
// The returned string identifies the selected backend.
func NewReaderWriter() (Reader, Writer, string, error) {
	switch {
	case availableFn("wl-paste"):
		w := WaylandClipboard{}
		return w, w, "wayland (wl-paste)", nil
	case availableFn("xclip"):
		x := XclipClipboard{}
		return x, x, "x11 (xclip)", nil
	case availableFn("xsel"):
		s := XselClipboard{}
		return s, s, "x11 (xsel)", nil
	case availableFn("osascript") && availableFn("pbpaste"):
		o := DarwinClipboard{}
		return o, o, "darwin (osascript+pbpaste)", nil
	default:
		return nil, nil, "", ErrNoClipAvailable
	}
}

var availableFn = available

func available(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}
