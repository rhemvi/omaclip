// Package clipboard provides OS-level clipboard access.
package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// WaylandClipboard reads and writes the system clipboard via wl-paste/wl-copy.
type WaylandClipboard struct{}

// GetText returns the current clipboard contents using wl-paste. Returns empty if the clipboard only contains non-text types (e.g. image).
func (w WaylandClipboard) GetText() (string, error) {
	typesCmd := exec.Command("wl-paste", "--list-types")
	typesOut, err := typesCmd.Output()
	if err != nil {
		return "", fmt.Errorf("wl-paste --list-types: %w", err)
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

	cmd := exec.Command("wl-paste", "--no-newline")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("wl-paste: %w", err)
	}
	return string(out), nil
}

// GetImage returns PNG image bytes from the clipboard if the clipboard contains an image without a text representation.
func (w WaylandClipboard) GetImage() ([]byte, error) {
	cmd := exec.Command("wl-paste", "--list-types")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("wl-paste --list-types: %w", err)
	}

	types := strings.Split(strings.TrimSpace(string(out)), "\n")
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

	imgCmd := exec.Command("wl-paste", "--type", "image/png")
	imgData, err := imgCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("wl-paste image/png: %w", err)
	}
	return imgData, nil
}

// SetText writes text to the clipboard using wl-copy.
func (w WaylandClipboard) SetText(text string) error {
	cmd := exec.Command("wl-copy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wl-copy: %w", err)
	}
	return nil
}

// SetImage writes PNG image data to the clipboard using wl-copy.
func (w WaylandClipboard) SetImage(pngData []byte) error {
	cmd := exec.Command("wl-copy", "--type", "image/png")
	cmd.Stdin = bytes.NewReader(pngData)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wl-copy image: %w", err)
	}
	return nil
}
