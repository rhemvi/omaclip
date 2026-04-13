package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"testing"

	"github.com/rhemvi/omaclip/foundation/imagefilereader"
)

// mockReader implements Reader for testing.
type mockReader struct {
	text    string
	textErr error
	img     []byte
	imgErr  error
}

func (r *mockReader) GetText(_ context.Context) (string, error) { return r.text, r.textErr }
func (r *mockReader) GetImage(_ context.Context) ([]byte, error) { return r.img, r.imgErr }

// makePNG generates a minimal valid 1x1 PNG image.
func makePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// newTestMonitor creates a Monitor wired to a mockReader with sensible defaults.
func newTestMonitor(r *mockReader) *Monitor {
	return NewMonitor(
		slog.Default(),
		r,
		nil, // writer not needed for readClipboard
		50,
		5,
		2,
		0, // pollInterval unused in direct calls
	)
}

func TestReadClipboard_NewText(t *testing.T) {
	r := &mockReader{text: "hello"}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 1 {
		t.Fatalf("got %d entries, want 1", len(h))
	}
	if h[0].ContentType != "text" || h[0].Content != "hello" {
		t.Errorf("got entry %+v, want text=hello", h[0])
	}
}

func TestReadClipboard_DuplicateTextIgnored(t *testing.T) {
	r := &mockReader{text: "hello"}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())
	m.readClipboard(context.Background())

	if got := len(m.GetHistory()); got != 1 {
		t.Errorf("got %d entries, want 1 (duplicate should be ignored)", got)
	}
}

func TestReadClipboard_EmptyTextIgnored(t *testing.T) {
	r := &mockReader{text: ""}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	if got := len(m.GetHistory()); got != 0 {
		t.Errorf("got %d entries, want 0", got)
	}
}

func TestReadClipboard_TextErrorIgnored(t *testing.T) {
	r := &mockReader{textErr: fmt.Errorf("read error"), img: makePNG(t)}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 1 {
		t.Fatalf("got %d entries, want 1", len(h))
	}
	if h[0].ContentType != "image" {
		t.Errorf("got contentType=%s, want image", h[0].ContentType)
	}
}

func TestReadClipboard_NewImage(t *testing.T) {
	pngData := makePNG(t)
	r := &mockReader{img: pngData}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 1 {
		t.Fatalf("got %d entries, want 1", len(h))
	}
	if h[0].ContentType != "image" {
		t.Errorf("got contentType=%s, want image", h[0].ContentType)
	}
	if h[0].ImageMimeType != "image/png" {
		t.Errorf("got mime=%s, want image/png", h[0].ImageMimeType)
	}
}

func TestReadClipboard_DuplicateImageIgnored(t *testing.T) {
	pngData := makePNG(t)
	r := &mockReader{img: pngData}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())
	m.readClipboard(context.Background())

	if got := len(m.GetHistory()); got != 1 {
		t.Errorf("got %d entries, want 1 (duplicate should be ignored)", got)
	}
}

func TestReadClipboard_ImageTooLarge_AddedOnce(t *testing.T) {
	tooLargeErr := fmt.Errorf("%w: photo.jpg is 10.00 MB, limit is 2 MB", imagefilereader.ErrImageTooLarge)
	r := &mockReader{imgErr: tooLargeErr}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())
	m.readClipboard(context.Background())
	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 1 {
		t.Fatalf("got %d entries, want 1 (repeated rejection should be deduped)", len(h))
	}
	if h[0].ContentType != "image-rejected" {
		t.Errorf("got contentType=%s, want image-rejected", h[0].ContentType)
	}
}

func TestReadClipboard_ImageTooLarge_DifferentFileNotDeduped(t *testing.T) {
	r := &mockReader{
		imgErr: fmt.Errorf("%w: photo1.jpg is 10.00 MB, limit is 2 MB", imagefilereader.ErrImageTooLarge),
	}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	r.imgErr = fmt.Errorf("%w: photo2.jpg is 8.00 MB, limit is 2 MB", imagefilereader.ErrImageTooLarge)
	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 2 {
		t.Fatalf("got %d entries, want 2 (different rejections should both appear)", len(h))
	}
}

func TestReadClipboard_ImageExceedsSizeLimit_Rejected(t *testing.T) {
	// Create image data that exceeds the 2 MB non-PNG limit.
	// Use non-PNG bytes so http.DetectContentType won't return image/png.
	bigData := make([]byte, 3*1024*1024)
	bigData[0] = 0xFF // not a PNG header
	r := &mockReader{img: bigData}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 1 {
		t.Fatalf("got %d entries, want 1", len(h))
	}
	if h[0].ContentType != "image-rejected" {
		t.Errorf("got contentType=%s, want image-rejected", h[0].ContentType)
	}
}

func TestReadClipboard_ImageExceedsSizeLimit_DuplicateIgnored(t *testing.T) {
	bigData := make([]byte, 3*1024*1024)
	bigData[0] = 0xFF
	r := &mockReader{img: bigData}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())
	m.readClipboard(context.Background())

	if got := len(m.GetHistory()); got != 1 {
		t.Errorf("got %d entries, want 1 (duplicate should be ignored)", got)
	}
}

func TestReadClipboard_TextAndImage(t *testing.T) {
	pngData := makePNG(t)
	r := &mockReader{text: "hello", img: pngData}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 2 {
		t.Fatalf("got %d entries, want 2", len(h))
	}
	// History is reverse-chronological: image first, then text.
	if h[0].ContentType != "image" {
		t.Errorf("entry[0] got contentType=%s, want image", h[0].ContentType)
	}
	if h[1].ContentType != "text" {
		t.Errorf("entry[1] got contentType=%s, want text", h[1].ContentType)
	}
}

func TestReadClipboard_TextAfterImage(t *testing.T) {
	pngData := makePNG(t)
	r := &mockReader{img: pngData}
	m := newTestMonitor(r)

	m.readClipboard(context.Background())

	r.img = nil
	r.text = "new text"
	m.readClipboard(context.Background())

	h := m.GetHistory()
	if len(h) != 2 {
		t.Fatalf("got %d entries, want 2", len(h))
	}
	if h[0].ContentType != "text" || h[0].Content != "new text" {
		t.Errorf("entry[0] got %+v, want text=new text", h[0])
	}
}

func TestReadClipboard_OnNewEntryCallback(t *testing.T) {
	r := &mockReader{text: "hello"}
	m := newTestMonitor(r)

	var called int
	m.OnNewEntry = func(_ ClipboardEntry) { called++ }

	m.readClipboard(context.Background())

	if called != 1 {
		t.Errorf("OnNewEntry called %d times, want 1", called)
	}
}

func TestReadClipboard_MaxHistoryTrimmed(t *testing.T) {
	r := &mockReader{}
	m := newTestMonitor(r)
	m.maxHistory = 3

	for i := range 5 {
		r.text = fmt.Sprintf("text-%d", i)
		m.readClipboard(context.Background())
	}

	h := m.GetHistory()
	if len(h) != 3 {
		t.Fatalf("got %d entries, want 3", len(h))
	}
	if h[0].Content != "text-4" {
		t.Errorf("newest entry got %s, want text-4", h[0].Content)
	}
	if h[2].Content != "text-2" {
		t.Errorf("oldest entry got %s, want text-2", h[2].Content)
	}
}
