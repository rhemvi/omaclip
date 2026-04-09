package clipboard

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
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
// wl-paste (Wayland) → xclip (X11) → xsel (X11) → osascript (macOS) → pbpaste (macOS).
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
	case availableFn("osascript"):
		o := DarwinOsascriptClipboard{}
		return o, o, "darwin (osascript)", nil
	case availableFn("pbpaste"):
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

// imageFileExtensions lists extensions recognized as image files when detecting
// file copies from a file manager.
var imageFileExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".bmp": true, ".tiff": true, ".tif": true,
	".webp": true, ".avif": true, ".heic": true, ".heif": true,
}

func isImageFile(path string) bool {
	return imageFileExtensions[strings.ToLower(filepath.Ext(path))]
}
