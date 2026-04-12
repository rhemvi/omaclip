package clipboard

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rhemvi/omaclip/foundation/imagefilereader"
)

// DarwinClipboard reads and writes the macOS clipboard using pbpaste/pbcopy for text
// and osascript for image operations.
type DarwinClipboard struct {
	imgReader imagefilereader.Reader
}

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
		return "", nil
	}

	cmd := exec.CommandContext(ctx, "pbpaste")
	maxText := max(d.imgReader.MaxPngBytes(), d.imgReader.MaxNonPngBytes())
	out, err := readCommandOutput(cmd, maxText)
	if err != nil {
		return "", fmt.Errorf("pbpaste: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// GetImage returns image bytes from the clipboard. It prefers reading the
// original file when a file URL is present (Finder copy), falling back to
// PNG clipboard data, then JPEG.
func (d DarwinClipboard) GetImage(ctx context.Context) ([]byte, error) {
	info, err := d.clipboardInfo(ctx)
	if err != nil {
		return nil, err
	}

	if containsType(info, "«class furl»") || containsType(info, "public.file-url") {
		path := d.fileURL(ctx)
		if path != "" {
			if imagefilereader.IsImage(path) {
				return d.imgReader.ReadImageFile(path)
			}
			return nil, nil
		}
	}

	if containsType(info, "PNGf") {
		data, err := d.readClipboardAs(ctx, "«class PNGf»", "omaclip-read-*.png")
		if err != nil {
			return nil, fmt.Errorf("read PNGf: %w", err)
		}
		if len(data) > 0 {
			return data, nil
		}
	}

	if containsType(info, "JPEG picture") {
		data, err := d.readClipboardAs(ctx, "JPEG picture", "omaclip-read-*.jpg")
		if err != nil {
			return nil, fmt.Errorf("read JPEG: %w", err)
		}
		if len(data) > 0 {
			return data, nil
		}
	}

	return nil, nil
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

	info, statErr := os.Stat(f.Name())
	if statErr != nil {
		return nil, fmt.Errorf("stat temp file: %w", statErr)
	}

	maxBytes := d.imgReader.MaxNonPngBytes()
	if strings.HasSuffix(strings.ToLower(tmpPattern), ".png") {
		maxBytes = d.imgReader.MaxPngBytes()
	}

	if info.Size() > maxBytes {
		return nil, fmt.Errorf(
			"%w: clipboard image is %.2f MB, limit is %d MB",
			errOutputTooLarge, float64(info.Size())/(1024*1024), maxBytes/(1024*1024),
		)
	}

	data, err := os.ReadFile(f.Name())
	if err != nil {
		return nil, fmt.Errorf("reading temp file: %w", err)
	}
	return data, nil
}

// Watch polls the macOS pasteboard change count and sends a signal on notify each time it changes. The poll loop runs in a background goroutine until ctx is cancelled.
func (d DarwinClipboard) Watch(ctx context.Context, notify chan<- struct{}) error {
	count, err := d.changeCount(ctx)
	if err != nil {
		return fmt.Errorf("initial pasteboard change count: %w", err)
	}

	go func() {
		defer close(notify)
		lastCount := count
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				count, err := d.changeCount(ctx)
				if err != nil {
					continue
				}
				if count != lastCount {
					lastCount = count
					select {
					case notify <- struct{}{}:
					default:
					}
				}
			}
		}
	}()

	return nil
}

func (d DarwinClipboard) changeCount(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", "-e",
		`ObjC.import("AppKit"); $.NSPasteboard.generalPasteboard.changeCount`)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("osascript change count: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func containsType(info, typeName string) bool {
	return strings.Contains(info, typeName)
}
