// Package app is the Wails bind target that wires together all business packages.
package app

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"clipmaster/business/clipboard"
	"clipmaster/business/theme"
	osclip "clipmaster/foundation/clipboard"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Config holds all configurable values for the application.
type Config struct {
	MaxHistory     int
	ThemeColorPath string
	PollInterval   time.Duration
}

// App is the Wails bind target. It owns startup/shutdown and delegates to business packages.
type App struct {
	ctx        context.Context
	log        *slog.Logger
	cfg        Config
	monitor    *clipboard.Monitor
	colors     theme.ThemeColors
	useWayland bool
}

// NewApp creates an App with the provided configuration.
func NewApp(log *slog.Logger, cfg Config) *App {
	return &App{
		cfg:        cfg,
		log:        log,
		useWayland: isWaylandAvailable(),
	}
}

// Startup is called by Wails when the application starts.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.log.Info("starting application")

	generic := clipboard.GenericClipboard{Ctx: ctx}
	var reader clipboard.Reader
	if a.useWayland {
		reader = osclip.WaylandClipboard{}
	} else {
		reader = generic
	}
	a.monitor = clipboard.NewMonitor(reader, generic, a.cfg.MaxHistory, a.cfg.PollInterval)

	if areWeRunningInOmarchy(a.cfg.ThemeColorPath) {
		colors, err := theme.Load(a.cfg.ThemeColorPath)
		if err != nil {
			a.log.Warn("could not load theme", "error", err)
		} else {
			a.colors = colors
			runtime.EventsEmit(ctx, "theme:loaded", colors)
		}

		w := theme.NewWatcher(a.cfg.ThemeColorPath, func(c theme.ThemeColors) {
			a.colors = c
			runtime.EventsEmit(ctx, "theme:loaded", c)
		})
		if err := w.Start(ctx); err != nil {
			a.log.Warn("could not watch theme file", "error", err)
		}
	}

	a.monitor.OnNewEntry = func(entry clipboard.ClipboardEntry) {
		runtime.EventsEmit(ctx, "clipboard:new", entry)
	}
	a.monitor.Start(ctx)
}

// Shutdown is called by Wails when the application is closing.
func (a *App) Shutdown(ctx context.Context) {
	a.log.Info("shutting down application")
	a.monitor.Stop()
}

// GetHistory returns all clipboard entries in reverse-chronological order.
func (a *App) GetHistory() []clipboard.ClipboardEntry {
	return a.monitor.GetHistory()
}

// CopyItem writes the entry with the given ID back to the system clipboard.
func (a *App) CopyItem(id string) error {
	return a.monitor.CopyItem(id)
}

// GetTheme returns the currently loaded theme colors.
func (a *App) GetTheme() theme.ThemeColors {
	return a.colors
}

func areWeRunningInOmarchy(themeColorPath string) bool {
	_, err := os.Stat(themeColorPath)
	return err == nil
}

func isWaylandAvailable() bool {
	_, err := exec.LookPath("wl-paste")
	return err == nil
}
