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
// If forceXclip is true, xclip is used regardless of other available backends.
// The returned string identifies the selected backend.
func NewReaderWriter(forceXclip bool) (Reader, Writer, string, error) {
	if forceXclip {
		if !availableFn("xclip") {
			return nil, nil, "", errors.New("xclip forced but not found")
		}
		x := XclipClipboard{}
		return x, x, "x11 (xclip, forced)", nil
	}

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
	case availableFn("pbpaste"):
		if availableFn("osascript") {
			o := DarwinOsascriptClipboard{}
			return o, o, "darwin (osascript)", nil
		}
		d := DarwinClipboard{}
		return d, d, "darwin (pbpaste)", nil
	default:
		return nil, nil, "", ErrNoClipAvailable
	}
}

var availableFn = available

func available(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}
