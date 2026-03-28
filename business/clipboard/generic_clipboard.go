package clipboard

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// GenericClipboard reads and writes the system clipboard via the Wails runtime.
type GenericClipboard struct {
	Ctx context.Context
}

// GetText returns the current clipboard contents using the Wails runtime.
func (g GenericClipboard) GetText() (string, error) {
	return runtime.ClipboardGetText(g.Ctx)
}

// SetText writes text to the clipboard using the Wails runtime.
func (g GenericClipboard) SetText(text string) error {
	return runtime.ClipboardSetText(g.Ctx, text)
}
