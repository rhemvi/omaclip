package clipboard

import (
	"path/filepath"
	"slices"
	"strings"
)

var imageFileExtensions = []string{
	".png", ".jpg", ".jpeg",
	".gif", ".bmp", ".tiff", ".tif",
	".webp", ".avif", ".heic", ".heif",
}

func isImageFile(path string) bool {
	ext := filepath.Ext(path)
	return slices.Contains(imageFileExtensions, ext)
}

type clipboardTypes struct {
	hasText     bool
	hasImage    bool
	hasFileList bool
}

func parseClipboardTypes(raw string) clipboardTypes {
	var ct clipboardTypes
	for _, t := range strings.Split(strings.TrimSpace(raw), "\n") {
		t = strings.TrimSpace(t)
		switch {
		case t == "text/plain" || t == "STRING" || t == "UTF8_STRING":
			ct.hasText = true
		case strings.HasPrefix(t, "image/"):
			ct.hasImage = true
		case t == "text/uri-list" || t == "x-special/gnome-copied-files":
			ct.hasFileList = true
		}
	}
	return ct
}
