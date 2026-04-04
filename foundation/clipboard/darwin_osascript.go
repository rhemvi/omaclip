package clipboard

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DarwinOsascriptClipboard reads and writes the macOS clipboard using pbpaste/pbcopy for text
// and osascript for image operations.
type DarwinOsascriptClipboard struct{}

// GetText returns the current clipboard text using pbpaste. Returns empty if the clipboard only contains non-text types.
func (d DarwinOsascriptClipboard) GetText() (string, error) {
	info, err := d.clipboardInfo()
	if err != nil {
		return "", err
	}

	if !containsType(info, "public.utf8-plain-text") {
		return "", nil
	}

	cmd := exec.Command("pbpaste")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pbpaste: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// GetImage returns PNG image bytes from the clipboard if it contains an image without a text representation.
func (d DarwinOsascriptClipboard) GetImage() ([]byte, error) {
	info, err := d.clipboardInfo()
	if err != nil {
		return nil, err
	}

	if containsType(info, "public.utf8-plain-text") {
		return nil, nil
	}
	if !containsType(info, "PNGf") {
		return nil, nil
	}

	f, err := os.CreateTemp("", "clipmaster-read-*.png")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	script := fmt.Sprintf(
		`set filePath to (POSIX file %q)
set fileRef to open for access filePath with write permission
write (the clipboard as «class PNGf») to fileRef
close access fileRef`, f.Name())

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("osascript read clipboard image: %w", err)
	}

	data, err := os.ReadFile(f.Name())
	if err != nil {
		return nil, fmt.Errorf("reading temp file: %w", err)
	}
	return data, nil
}

// SetText writes text to the clipboard using pbcopy.
func (d DarwinOsascriptClipboard) SetText(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pbcopy: %w", err)
	}
	return nil
}

// SetImage writes PNG image data to the clipboard using osascript via a temporary file.
func (d DarwinOsascriptClipboard) SetImage(pngData []byte) error {
	f, err := os.CreateTemp("", "clipmaster-write-*.png")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if _, err := f.Write(pngData); err != nil {
		f.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	f.Close()

	script := fmt.Sprintf(`set the clipboard to (read (POSIX file %q) as «class PNGf»)`, f.Name())
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("osascript set clipboard image: %w", err)
	}
	return nil
}

func (d DarwinOsascriptClipboard) clipboardInfo() (string, error) {
	cmd := exec.Command("osascript", "-e", "clipboard info")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("osascript clipboard info: %w", err)
	}
	return string(out), nil
}

func containsType(info, typeName string) bool {
	return strings.Contains(info, typeName)
}
