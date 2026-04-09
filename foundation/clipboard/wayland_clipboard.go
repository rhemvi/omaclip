// Package clipboard provides OS-level clipboard access.
package clipboard

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
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

	types := parseClipboardTypes(string(typesOut))

	if !types.hasText {
		return "", nil
	}

	// If this is a copied image file, skip the text (just the filename/URI).
	if types.hasFileList {
		if path := wlPasteFileImagePath(); path != "" {
			return "", nil
		}
	}

	cmd := exec.Command("wl-paste", "--no-newline")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("wl-paste: %w", err)
	}
	return string(out), nil
}

// GetImage returns image bytes from the clipboard. It reads the original file
// when a file URI is present (file manager copy), otherwise reads image/png.
func (w WaylandClipboard) GetImage() ([]byte, error) {
	cmd := exec.Command("wl-paste", "--list-types")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("wl-paste --list-types: %w", err)
	}

	types := parseClipboardTypes(string(out))

	// If a file URI is present and points to an image, read it directly.
	if types.hasFileList {
		if path := wlPasteFileImagePath(); path != "" {
			data, err := os.ReadFile(path)
			if err == nil {
				return data, nil
			}
		}
	}

	// For non-file image data (e.g. screenshot, copy from browser), skip if text is also present.
	if types.hasText {
		return nil, nil
	}

	if !types.hasImage {
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

// Watch starts wl-paste --watch and sends a signal on notify each time the clipboard changes. Returns an error if the process fails to start. The scan loop runs in a background goroutine until ctx is cancelled.
func (w WaylandClipboard) Watch(ctx context.Context, notify chan<- struct{}) error {
	cmd := exec.CommandContext(ctx, "wl-paste", "--watch", "echo")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("wl-paste --watch stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("wl-paste --watch start: %w", err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case notify <- struct{}{}:
			default:
			}
		}
		_ = cmd.Wait()
		close(notify)
	}()

	return nil
}


// wlPasteFileImagePath reads text/uri-list from the clipboard and returns the
// local file path if it points to a single image file.
func wlPasteFileImagePath() string {
	cmd := exec.Command("wl-paste", "--type", "text/uri-list")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
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

// SetImage writes image data to the clipboard using wl-copy.
func (w WaylandClipboard) SetImage(data []byte, mimeType string) error {
	cmd := exec.Command("wl-copy", "--type", mimeType)
	cmd.Stdin = bytes.NewReader(data)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wl-copy image: %w", err)
	}
	return nil
}
