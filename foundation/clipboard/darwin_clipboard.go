package clipboard

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DarwinClipboard reads and writes the macOS clipboard using pbpaste/pbcopy for text
// and osascript for image operations.
type DarwinClipboard struct{}

// GetText returns the current clipboard text using pbpaste. Returns empty if the clipboard only contains non-text types.
func (d DarwinClipboard) GetText() (string, error) {
	info, err := d.clipboardInfo()
	if err != nil {
		return "", err
	}

	if !containsType(info, "public.utf8-plain-text") && !containsType(info, "«class utf8»") {
		return "", nil
	}

	if containsType(info, "«class furl»") || containsType(info, "public.file-url") {
		path := d.fileURL()
		if path != "" && isImageFile(path) {
			return "", nil
		}
	}

	cmd := exec.Command("pbpaste")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pbpaste: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// GetImage returns image bytes from the clipboard. It prefers reading the
// original file when a file URL is present (Finder copy), falling back to
// JPEG clipboard data, then PNG.
func (d DarwinClipboard) GetImage() ([]byte, error) {
	info, err := d.clipboardInfo()
	if err != nil {
		return nil, err
	}

	if containsType(info, "«class furl»") || containsType(info, "public.file-url") {
		path := d.fileURL()
		if path != "" && isImageFile(path) {
			data, err := os.ReadFile(path)
			if err == nil {
				return data, nil
			}
		}
	}

	if containsType(info, "JPEG picture") {
		data, err := d.readClipboardAs("JPEG picture", "omaclip-read-*.jpg")
		if err == nil && len(data) > 0 {
			return data, nil
		}
	}

	if !containsType(info, "PNGf") {
		return nil, nil
	}

	return d.readClipboardAs("«class PNGf»", "omaclip-read-*.png")
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

// SetImage writes image data to the clipboard using osascript via a temporary file.
func (d DarwinClipboard) SetImage(data []byte, mimeType string) error {
	f, err := os.CreateTemp("", "omaclip-write-*.png")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if _, err := f.Write(data); err != nil {
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

func (d DarwinClipboard) clipboardInfo() (string, error) {
	cmd := exec.Command("osascript", "-e", "clipboard info")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("osascript clipboard info: %w", err)
	}
	return string(out), nil
}

func (d DarwinClipboard) fileURL() string {
	cmd := exec.Command("osascript", "-e", `POSIX path of (the clipboard as «class furl»)`)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (d DarwinClipboard) readClipboardAs(clipClass, tmpPattern string) ([]byte, error) {
	f, err := os.CreateTemp("", tmpPattern)
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	script := fmt.Sprintf(
		`set filePath to (POSIX file %q)
set fileRef to open for access filePath with write permission
write (the clipboard as %s) to fileRef
close access fileRef`, f.Name(), clipClass)

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

func containsType(info, typeName string) bool {
	return strings.Contains(info, typeName)
}
