// Package clipboard provides OS-level clipboard access.
package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
)

// WaylandClipboard reads and writes the system clipboard via wl-paste/wl-copy.
type WaylandClipboard struct{}

// GetText returns the current clipboard contents using wl-paste.
func (w WaylandClipboard) GetText() (string, error) {
	cmd := exec.Command("wl-paste", "--no-newline")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("wl-paste: %w", err)
	}
	return string(out), nil
}

// SetText writes text to the clipboard using wl-copy.
func (w WaylandClipboard) SetText(text string) error {
	cmd := exec.Command("wl-copy", text)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wl-copy: %w: %s", err, stderr.String())
	}
	return nil
}
