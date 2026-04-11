package clipboard

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DarwinClipboard reads and writes the macOS clipboard using pbpaste/pbcopy for text
// and osascript for image operations.
type DarwinClipboard struct{}

// GetText returns the current clipboard text using pbpaste. Returns empty if the clipboard only contains non-text types.
func (d DarwinClipboard) GetText(ctx context.Context) (string, error) {
	info, err := d.clipboardInfo(ctx)
	if err != nil {
		return "", err
	}

	if !containsType(info, "public.utf8-plain-text") && !containsType(info, "«class utf8»") {
		return "", nil
	}

	if containsType(info, "«class furl»") || containsType(info, "public.file-url") {
		path := d.fileURL(ctx)
		if path != "" && isImageFile(path) {
			return "", nil
		}
	}

	cmd := exec.CommandContext(ctx, "pbpaste")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pbpaste: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// GetImage returns image bytes from the clipboard. It prefers reading the
// original file when a file URL is present (Finder copy), falling back to
// JPEG clipboard data, then PNG.
func (d DarwinClipboard) GetImage(ctx context.Context) ([]byte, error) {
	info, err := d.clipboardInfo(ctx)
	if err != nil {
		return nil, err
	}

	if containsType(info, "«class furl»") || containsType(info, "public.file-url") {
		path := d.fileURL(ctx)
		if path != "" && isImageFile(path) {
			data, err := os.ReadFile(path)
			if err == nil {
				return data, nil
			}
		}
	}

	if containsType(info, "JPEG picture") {
		data, err := d.readClipboardAs(ctx, "JPEG picture", "omaclip-read-*.jpg")
		if err == nil && len(data) > 0 {
			return data, nil
		}
	}

	if !containsType(info, "PNGf") {
		return nil, nil
	}

	return d.readClipboardAs(ctx, "«class PNGf»", "omaclip-read-*.png")
}

// SetText writes text to the clipboard using pbcopy.
func (d DarwinClipboard) SetText(ctx context.Context, text string) error {
	cmd := exec.CommandContext(ctx, "pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pbcopy: %w", err)
	}
	return nil
}

// SetImage writes image data to the clipboard using osascript via a temporary file.
func (d DarwinClipboard) SetImage(ctx context.Context, data []byte, mimeType string) error {
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
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("osascript set clipboard image: %w", err)
	}
	return nil
}

func (d DarwinClipboard) clipboardInfo(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "osascript", "-e", "clipboard info")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("osascript clipboard info: %w", err)
	}
	return string(out), nil
}

func (d DarwinClipboard) fileURL(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "osascript", "-e", `POSIX path of (the clipboard as «class furl»)`)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (d DarwinClipboard) readClipboardAs(ctx context.Context, clipClass, tmpPattern string) ([]byte, error) {
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

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
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
