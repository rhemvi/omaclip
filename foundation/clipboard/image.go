package clipboard

import (
	"path/filepath"
	"slices"
	"strings"
)

// preferredImageTypes lists clipboard image MIME types in order of preference.
var preferredImageTypes = []string{
	"image/png",
	"image/jpeg",
	"image/bmp",
	"image/tiff",
	"image/webp",
}

var imageFileExtensions = []string{
	".png", ".jpg", ".jpeg",
	".gif", ".bmp", ".tiff", ".tif",
	".webp", ".avif", ".heic", ".heif",
}

func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return slices.Contains(imageFileExtensions, ext)
}

type clipboardTypes struct {
	hasText     bool
	hasImage    bool
	hasFileList bool
	imageTypes  []string
}

func parseClipboardTypes(raw string) clipboardTypes {
	var ct clipboardTypes
	for t := range strings.SplitSeq(strings.TrimSpace(raw), "\n") {
		t = strings.TrimSpace(t)
		switch {
		case t == "text/plain" || strings.HasPrefix(t, "text/plain;") || t == "STRING" || t == "UTF8_STRING":
			ct.hasText = true
		case strings.HasPrefix(t, "image/"):
			ct.hasImage = true
			ct.imageTypes = append(ct.imageTypes, t)
		case t == "text/uri-list" || t == "x-special/gnome-copied-files":
			ct.hasFileList = true
		}
	}
	return ct
}

// bestImageType returns the preferred image MIME type from those available on
// the clipboard, falling back to "image/png" if none of the preferred types match.
func (ct clipboardTypes) bestImageType() string {
	for _, pref := range preferredImageTypes {
		if slices.Contains(ct.imageTypes, pref) {
			return pref
		}
	}
	if len(ct.imageTypes) > 0 {
		return ct.imageTypes[0]
	}
	return "image/png"
}
