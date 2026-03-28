package theme

import (
	"context"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors a colors.toml file for changes and invokes a callback with the new colors.
type Watcher struct {
	path    string
	onChange func(ThemeColors)
	watcher *fsnotify.Watcher
}

// NewWatcher creates a Watcher that monitors the given colors.toml path.
func NewWatcher(path string, onChange func(ThemeColors)) *Watcher {
	return &Watcher{
		path:    path,
		onChange: onChange,
	}
}

// Start begins watching for file changes. It blocks until the context is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.watcher = fsw

	themeDir := filepath.Dir(w.path)
	parentDir := filepath.Dir(themeDir)

	if err := fsw.Add(parentDir); err != nil {
		fsw.Close()
		return err
	}
	fsw.Add(themeDir)

	go w.loop(ctx, themeDir)
	return nil
}

func (w *Watcher) loop(ctx context.Context, themeDir string) {
	defer w.watcher.Close()

	var debounce *time.Timer
	target := filepath.Base(w.path)
	themeDirName := filepath.Base(themeDir)

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			name := filepath.Base(event.Name)

			if name == themeDirName && event.Op&(fsnotify.Create|fsnotify.Rename) != 0 {
				w.watcher.Add(themeDir)
			} else if name != target {
				continue
			} else if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(200*time.Millisecond, func() {
				colors, err := Load(w.path)
				if err != nil {
					return
				}
				w.onChange(colors)
			})
		case _, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
		}
	}
}
