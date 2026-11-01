package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/rhemvi/omaclip/business/clipboard"
)

type hookTestWriter struct {
	err error
}

func (w *hookTestWriter) SetText(context.Context, string) error {
	return w.err
}

func (w *hookTestWriter) SetImage(context.Context, []byte, string) error {
	return w.err
}

func TestCopyRemoteItemTriggersHookAfterClipboardWrite(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "triggered")
	app := newHookTestApp(&hookTestWriter{}, "printf triggered > "+strconv.Quote(marker))

	if err := app.CopyRemoteItem("copied text"); err != nil {
		t.Fatalf("copy remote item: %v", err)
	}

	waitForHookTest(t, func() bool {
		data, err := os.ReadFile(marker)
		return err == nil && string(data) == "triggered"
	})
}

func TestCopyRemoteItemDoesNotTriggerHookWhenClipboardWriteFails(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "triggered")
	writeErr := errors.New("write failed")
	app := newHookTestApp(&hookTestWriter{err: writeErr}, "printf triggered > "+strconv.Quote(marker))

	if err := app.CopyRemoteItem("copied text"); !errors.Is(err, writeErr) {
		t.Fatalf("copy remote item error = %v, want %v", err, writeErr)
	}

	time.Sleep(100 * time.Millisecond)
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("hook marker exists after failed clipboard write: %v", err)
	}
}

func newHookTestApp(writer clipboard.Writer, hook string) *App {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := NewApp(nil, log, Config{CopyHook: hook})
	app.monitor = clipboard.NewMonitor(log, nil, writer, 1, 1, 1, time.Second)
	return app
}

func waitForHookTest(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("hook did not run before timeout")
}
