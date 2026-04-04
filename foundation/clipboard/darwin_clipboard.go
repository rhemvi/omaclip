package clipboard

import (
	"fmt"
	"os/exec"
	"strings"
)

// DarwinClipboard reads and writes the system clipboard via pbpaste/pbcopy (macOS).
// pbpaste/pbcopy are text-only — they do not support image clipboard operations.
type DarwinClipboard struct{}

// GetText returns the current clipboard contents using pbpaste.
func (d DarwinClipboard) GetText() (string, error) {
	cmd := exec.Command("pbpaste")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pbpaste: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// GetImage always returns nil — pbpaste does not support image clipboard operations.
func (d DarwinClipboard) GetImage() ([]byte, error) {
	return nil, nil
}

// SetText writes text to the clipboard using pbcopy.
func (d DarwinClipboard) SetText(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pbcopy: %w", err)
	}
	return nil
}

// SetImage always returns an error — pbcopy does not support image clipboard operations.
func (d DarwinClipboard) SetImage(pngData []byte) error {
	return ErrNotImplemented
}
