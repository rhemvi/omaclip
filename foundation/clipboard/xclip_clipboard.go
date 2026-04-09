package clipboard

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// XclipClipboard reads and writes the system clipboard via xclip (X11).
type XclipClipboard struct{}

// GetText returns the current clipboard text using xclip. Returns empty if the clipboard only contains non-text types.
func (x XclipClipboard) GetText() (string, error) {
	types := x.clipboardTypes()

	if !types.hasText {
		return "", nil
	}

	// If this is a copied image file, skip the text (just the filename/URI).
	if types.hasFileList {
		if path := xclipFileImagePath(); path != "" {
			return "", nil
		}
	}

	cmd := exec.Command("xclip", "-selection", "clipboard", "-o")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("xclip: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// GetImage returns image bytes from the clipboard. It reads the original file
// when a file URI is present (file manager copy), otherwise reads image/png.
func (x XclipClipboard) GetImage() ([]byte, error) {
	types := x.clipboardTypes()

	// If a file URI is present and points to an image, read it directly.
	if types.hasFileList {
		if path := xclipFileImagePath(); path != "" {
			data, err := os.ReadFile(path)
			if err == nil {
				return data, nil
			}
		}
	}

	// For non-file image data, skip if text is also present.
	if types.hasText {
		return nil, nil
	}

	if !types.hasImage {
		return nil, nil
	}

	imgCmd := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-o")
	imgData, err := imgCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("xclip image/png: %w", err)
	}
	return imgData, nil
}

func (x XclipClipboard) clipboardTypes() clipboardTypes {
	typesCmd := exec.Command("xclip", "-selection", "clipboard", "-t", "TARGETS", "-o")
	typesOut, err := typesCmd.Output()
	if err != nil {
		return clipboardTypes{}
	}
	return parseClipboardTypes(string(typesOut))
}

// xclipFileImagePath reads text/uri-list from the clipboard and returns the
// local file path if it points to a single image file.
func xclipFileImagePath() string {
	cmd := exec.Command("xclip", "-selection", "clipboard", "-t", "text/uri-list", "-o")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		u, err := url.Parse(line)
		if err != nil || u.Scheme != "file" {
			continue
		}
		path := u.Path
		if isImageFile(path) {
			return path
		}
	}
	return ""
}

// SetText writes text to the clipboard using xclip.
func (x XclipClipboard) SetText(text string) error {
	cmd := exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xclip: %w", err)
	}
	return nil
}

// SetImage writes PNG image data to the clipboard using xclip.
func (x XclipClipboard) SetImage(pngData []byte) error {
	cmd := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png")
	cmd.Stdin = bytes.NewReader(pngData)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xclip image: %w", err)
	}
	return nil
}
