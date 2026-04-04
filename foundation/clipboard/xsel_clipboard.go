package clipboard

import (
	"fmt"
	"os/exec"
	"strings"
)

// XselClipboard reads and writes the system clipboard via xsel (X11).
// xsel is text-only — it does not support image clipboard operations or MIME type listing.
type XselClipboard struct{}

// GetText returns the current clipboard contents using xsel.
func (x XselClipboard) GetText() (string, error) {
	cmd := exec.Command("xsel", "--clipboard", "--output")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("xsel: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// GetImage always returns nil — xsel does not support image clipboard operations.
func (x XselClipboard) GetImage() ([]byte, error) {
	return nil, nil
}

// SetText writes text to the clipboard using xsel.
func (x XselClipboard) SetText(text string) error {
	cmd := exec.Command("xsel", "--clipboard", "--input")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xsel: %w", err)
	}
	return nil
}

// SetImage always returns an error — xsel does not support image clipboard operations.
func (x XselClipboard) SetImage(pngData []byte) error {
	return ErrNotImplemented
}
