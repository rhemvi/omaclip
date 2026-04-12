// Package imagefilereader reads image files from disk with pre-read size checking.
package imagefilereader

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// ErrImageTooLarge is returned when a file exceeds the configured size limit.
var ErrImageTooLarge = errors.New("image file too large")

var imageFileExtensions = []string{
	".png", ".jpg", ".jpeg",
	".gif", ".bmp", ".tiff", ".tif",
	".webp", ".avif", ".heic", ".heif",
}

// Reader reads image files from disk with size limit enforcement.
type Reader struct {
	maxPngBytes    int64
	maxNonPngBytes int64
}

// NewReader creates a Reader with the given size limits in megabytes.
func NewReader(maxPngImageMB, maxNonPngImageMB int) Reader {
	return Reader{
		maxPngBytes:    int64(maxPngImageMB) * 1024 * 1024,
		maxNonPngBytes: int64(maxNonPngImageMB) * 1024 * 1024,
	}
}

// MaxPngBytes returns the configured maximum size in bytes for PNG images.
func (r Reader) MaxPngBytes() int64 { return r.maxPngBytes }

// MaxNonPngBytes returns the configured maximum size in bytes for non-PNG images.
func (r Reader) MaxNonPngBytes() int64 { return r.maxNonPngBytes }

// IsImage reports whether path has a known image file extension.
func IsImage(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return slices.Contains(imageFileExtensions, ext)
}

// ReadImageFile checks the file size via os.Stat before reading. PNG files use
// the PNG limit; all other image extensions use the non-PNG limit. Returns
// ErrImageTooLarge (wrapped with detail) if the file exceeds the limit.
func (r Reader) ReadImageFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat image file: %w", err)
	}

	maxBytes := r.maxNonPngBytes
	if strings.ToLower(filepath.Ext(path)) == ".png" {
		maxBytes = r.maxPngBytes
	}

	if info.Size() > maxBytes {
		return nil, fmt.Errorf(
			"%w: %s is %.2f MB, limit is %d MB",
			ErrImageTooLarge,
			filepath.Base(path),
			float64(info.Size())/(1024*1024),
			maxBytes/(1024*1024),
		)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read image file: %w", err)
	}
	return data, nil
}
