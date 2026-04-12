package clipboard

import (
	"context"
	"errors"
	"os/exec"

	"github.com/rhemvi/omaclip/foundation/imagefilereader"
)

var (
	ErrNoClipAvailable = errors.New("no supported clipboard binary found (tried: wl-paste, xclip, xsel, osascript+pbpaste)")
	ErrNotImplemented  = errors.New("not implemented")
)

// Reader abstracts clipboard reading across platform backends.
type Reader interface {
	GetText(ctx context.Context) (string, error)
	GetImage(ctx context.Context) ([]byte, error)
}

// Writer abstracts clipboard writing across platform backends.
type Writer interface {
	SetText(ctx context.Context, text string) error
	SetImage(ctx context.Context, data []byte, mimeType string) error
}

// NewReaderWriter returns the first available clipboard reader and writer by probing known binaries in order:
// wl-paste (Wayland) → xclip (X11) → xsel (X11) → osascript+pbpaste (macOS).
// The returned string identifies the selected backend.
func NewReaderWriter(maxPngImageMB, maxNonPngImageMB int) (Reader, Writer, string, error) {
	imgReader := imagefilereader.NewReader(maxPngImageMB, maxNonPngImageMB)

	switch {
	case availableFn("wl-paste"):
		w := WaylandClipboard{imgReader: imgReader}
		return w, w, "wayland (wl-paste)", nil
	case availableFn("xclip"):
		x := XclipClipboard{imgReader: imgReader}
		return x, x, "x11 (xclip)", nil
	case availableFn("xsel"):
		maxText := max(imgReader.MaxPngBytes(), imgReader.MaxNonPngBytes())
		s := XselClipboard{maxTextBytes: maxText}
		return s, s, "x11 (xsel)", nil
	case availableFn("osascript") && availableFn("pbpaste"):
		o := DarwinClipboard{imgReader: imgReader}
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
