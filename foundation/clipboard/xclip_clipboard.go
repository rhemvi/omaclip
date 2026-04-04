package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// XclipClipboard reads and writes the system clipboard via xclip (X11).
type XclipClipboard struct{}

// GetText returns the current clipboard text using xclip. Returns empty if the clipboard only contains non-text types.
func (x XclipClipboard) GetText() (string, error) {
	typesCmd := exec.Command("xclip", "-selection", "clipboard", "-t", "TARGETS", "-o")
	typesOut, err := typesCmd.Output()
	if err != nil {
		return "", fmt.Errorf("xclip TARGETS: %w", err)
	}

	hasText := false
	for _, t := range strings.Split(strings.TrimSpace(string(typesOut)), "\n") {
		t = strings.TrimSpace(t)
		if t == "text/plain" || t == "STRING" || t == "UTF8_STRING" {
			hasText = true
			break
		}
	}
	if !hasText {
		return "", nil
	}

	cmd := exec.Command("xclip", "-selection", "clipboard", "-o")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("xclip: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// GetImage returns PNG image bytes from the clipboard if the clipboard contains an image without a text representation.
func (x XclipClipboard) GetImage() ([]byte, error) {
	typesCmd := exec.Command("xclip", "-selection", "clipboard", "-t", "TARGETS", "-o")
	typesOut, err := typesCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("xclip TARGETS: %w", err)
	}

	types := strings.Split(strings.TrimSpace(string(typesOut)), "\n")
	hasImage := false
	for _, t := range types {
		t = strings.TrimSpace(t)
		if t == "text/plain" || t == "STRING" || t == "UTF8_STRING" {
			return nil, nil
		}
		if t == "image/png" {
			hasImage = true
		}
	}

	if !hasImage {
		return nil, nil
	}

	imgCmd := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-o")
	imgData, err := imgCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("xclip image/png: %w", err)
	}
	return imgData, nil
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
