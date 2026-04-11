package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// XclipClipboard reads and writes the system clipboard via xclip (X11).
type XclipClipboard struct{}

// GetText returns the current clipboard text using xclip. Returns empty if the clipboard only contains non-text types.
func (x XclipClipboard) GetText(ctx context.Context) (string, error) {
	types := x.clipboardTypes(ctx)

	if !types.hasText {
		return "", nil
	}

	// If this is a copied file, skip the text (just the filename/URI).
	if types.hasFileList {
		return "", nil
	}

	cmd := exec.CommandContext(ctx, "xclip", "-selection", "clipboard", "-o")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("xclip: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// GetImage returns image bytes from the clipboard. It reads the original file
// when a file URI is present (file manager copy), otherwise reads the best
// available image type (preferring PNG).
func (x XclipClipboard) GetImage(ctx context.Context) ([]byte, error) {
	types := x.clipboardTypes(ctx)

	// If a file URI is present and points to an image, read it directly.
	if types.hasFileList {
		if path := xclipFileImagePath(ctx); path != "" {
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

	imgType := types.bestImageType()
	imgCmd := exec.CommandContext(ctx, "xclip", "-selection", "clipboard", "-t", imgType, "-o")
	imgData, err := imgCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("xclip %s: %w", imgType, err)
	}
	return imgData, nil
}

func (x XclipClipboard) clipboardTypes(ctx context.Context) clipboardTypes {
	typesCmd := exec.CommandContext(ctx, "xclip", "-selection", "clipboard", "-t", "TARGETS", "-o")
	typesOut, err := typesCmd.Output()
	if err != nil {
		return clipboardTypes{}
	}
	return parseClipboardTypes(string(typesOut))
}

// xclipFileImagePath reads text/uri-list from the clipboard and returns the
// local file path if it points to a single image file.
func xclipFileImagePath(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "xclip", "-selection", "clipboard", "-t", "text/uri-list", "-o")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
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
func (x XclipClipboard) SetText(ctx context.Context, text string) error {
	cmd := exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xclip: %w", err)
	}
	return nil
}

// SetImage writes image data to the clipboard using xclip.
func (x XclipClipboard) SetImage(ctx context.Context, data []byte, mimeType string) error {
	cmd := exec.CommandContext(ctx, "xclip", "-selection", "clipboard", "-t", mimeType)
	cmd.Stdin = bytes.NewReader(data)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xclip image: %w", err)
	}
	return nil
}
