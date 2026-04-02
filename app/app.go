// Package app is the Wails bind target that wires together all business packages.
package app

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"clipmaster/app/handlers"
	"clipmaster/business/clipboard"
	"clipmaster/business/passphrase"
	"clipmaster/business/peersclipsync"
	bsync "clipmaster/business/sync"
	"clipmaster/business/theme"
	osclip "clipmaster/foundation/clipboard"
	fconfig "clipmaster/foundation/config"
	fmdns "clipmaster/foundation/mdns"
	"clipmaster/foundation/tlscert"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Config holds all configurable values for the application.
type Config struct {
	MaxHistory                   int
	ThemeColorPath               string
	ConfigPath                   string
	PollInterval                 time.Duration
	RemoteClipboardsPollInterval time.Duration
	RemoteClipboardsMaxHistory   int
	PeersPollInterval            time.Duration
}

// App is the Wails bind target. It owns startup/shutdown and delegates to business packages.
type App struct {
	ctx         context.Context
	log         *slog.Logger
	cfg         Config
	monitor     *clipboard.Monitor
	colors      theme.ThemeColors
	useWayland  bool
	syncServer      *bsync.Server
	discoverer      *fmdns.Discoverer
	peerFetcher     *peersclipsync.Fetcher
	passphraseStore *passphrase.Store
}

// NewApp creates an App with the provided configuration.
func NewApp(log *slog.Logger, cfg Config) *App {
	return &App{
		cfg:             cfg,
		log:             log,
		useWayland:      isWaylandAvailable(),
		passphraseStore: &passphrase.Store{},
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

	if cfg, err := fconfig.Load(a.cfg.ConfigPath); err == nil {
		a.passphraseStore.Set(cfg.Passphrase)
	}

	if a.passphraseStore.Get() != "" {
		a.startNetworking()
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
	if a.syncServer != nil {
		a.syncServer.Shutdown(ctx)
	}
	if a.discoverer != nil {
		a.discoverer.Shutdown()
	}
}

// GetHistory returns all clipboard entries in reverse-chronological order.
func (a *App) GetHistory() []clipboard.ClipboardEntry {
	return a.monitor.GetHistory()
}

// CopyItem writes the entry with the given ID back to the system clipboard.
func (a *App) CopyItem(id string) error {
	return a.monitor.CopyItem(id)
}

// GetRemoteClipboards returns clipboard entries from all discovered peers.
func (a *App) GetRemoteClipboards() []peersclipsync.PeerClipboard {
	if a.peerFetcher == nil {
		return nil
	}
	return a.peerFetcher.GetAll()
}

// CopyRemoteItem writes the given text directly to the system clipboard.
func (a *App) CopyRemoteItem(content string) error {
	return a.monitor.CopyText(content)
}

// GetTheme returns the currently loaded theme colors.
func (a *App) GetTheme() theme.ThemeColors {
	return a.colors
}

// NeedsPassphrase reports whether a passphrase has not yet been configured.
func (a *App) NeedsPassphrase() bool {
	cfg, err := fconfig.Load(a.cfg.ConfigPath)
	return err != nil || cfg.Passphrase == ""
}

// SubmitPassphrase saves the passphrase provided by the user and starts networking.
func (a *App) SubmitPassphrase(passphrase string) error {
	if err := fconfig.Save(a.cfg.ConfigPath, fconfig.Config{Passphrase: passphrase}); err != nil {
		return err
	}
	a.passphraseStore.Set(passphrase)
	a.startNetworking()
	return nil
}

// startNetworking initialises the TLS sync server, mDNS discovery, and peer fetcher.
// It is called at startup when a passphrase is already configured, or on first SubmitPassphrase.
func (a *App) startNetworking() {
	cert, err := tlscert.Generate()
	if err != nil {
		a.log.Error("failed to generate TLS cert", "error", err)
		return
	}

	a.syncServer = bsync.New(a.log, cert)
	registerRoutes(a.syncServer, &handlers.ClipboardHandler{
		Monitor:         a.monitor,
		MaxHistory:      a.cfg.RemoteClipboardsMaxHistory,
		PassphraseStore: a.passphraseStore,
	})
	if err := a.syncServer.Start(a.ctx); err != nil {
		a.log.Warn("sync server failed to start", "error", err)
		return
	}

	host, _ := os.Hostname()
	a.discoverer = fmdns.New(a.log, a.cfg.PeersPollInterval, host, a.passphraseStore)
	if err := a.discoverer.Register(a.syncServer.Port()); err != nil {
		a.log.Warn("mDNS registration failed", "error", err)
	}
	a.discoverer.Start(a.ctx)

	a.peerFetcher = peersclipsync.New(a.log, a.discoverer, a.cfg.RemoteClipboardsPollInterval, a.passphraseStore)
	a.peerFetcher.OnUpdate = func() {
		runtime.EventsEmit(a.ctx, "remote:updated")
	}
	a.peerFetcher.Start(a.ctx)
}

func areWeRunningInOmarchy(themeColorPath string) bool {
	_, err := os.Stat(themeColorPath)
	return err == nil
}

func isWaylandAvailable() bool {
	_, err := exec.LookPath("wl-paste")
	return err == nil
}
