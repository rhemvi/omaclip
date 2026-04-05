// Package app is the Wails bind target that wires together all business packages.
package app

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
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
	DisableRemoteClipboards      bool
}

// App is the Wails bind target. It owns startup/shutdown and delegates to business packages.
type App struct {
	ctx             context.Context
	log             *slog.Logger
	cfg             Config
	monitor         *clipboard.Monitor
	colors          theme.ThemeColors
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
		passphraseStore: &passphrase.Store{},
	}
}

// Startup is called by Wails when the application starts.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.log.Info("starting application")

	reader, writer, err := osclip.NewReaderWriter()
	if err != nil {
		a.log.Error("clipboard unavailable", "error", err)
		os.Exit(1)
	}
	a.monitor = clipboard.NewMonitor(a.log, reader, writer, a.cfg.MaxHistory, a.cfg.PollInterval)

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

	if err := validatePassphraseFromConfig(a.cfg.ConfigPath, a.passphraseStore); err != nil {
		a.log.Error(err.Error())
		os.Exit(1)
	}

	if a.passphraseStore.Get() != "" && !a.cfg.DisableRemoteClipboards {
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

// CopyRemoteImage writes base64-encoded PNG data to the system clipboard as an image.
func (a *App) CopyRemoteImage(imageDataBase64 string) error {
	return a.monitor.CopyImage(imageDataBase64)
}

// GetTheme returns the currently loaded theme colors.
func (a *App) GetTheme() theme.ThemeColors {
	return a.colors
}

// RemoteClipboardsEnabled reports whether remote clipboard sync is enabled.
func (a *App) RemoteClipboardsEnabled() bool {
	return !a.cfg.DisableRemoteClipboards
}

// GetConfigPath returns the path to the configuration file.
func (a *App) GetConfigPath() string {
	return a.cfg.ConfigPath
}

// NeedsPassphrase reports whether a passphrase has not yet been configured.
func (a *App) NeedsPassphrase() bool {
	if a.cfg.DisableRemoteClipboards {
		return false
	}
	return a.passphraseStore.Get() == ""
}

// SubmitPassphrase validates, saves the passphrase provided by the user, and starts networking.
func (a *App) SubmitPassphrase(p string) error {
	if err := passphrase.Validate(p); err != nil {
		return err
	}
	if err := fconfig.Save(a.cfg.ConfigPath, fconfig.Config{Passphrase: p}); err != nil {
		return err
	}
	a.passphraseStore.Set(p)
	a.startNetworking()
	return nil
}

// validatePassphraseFromConfig checks if a passphrase is already set in the config file, and if so validates it and sets it in the store.
func validatePassphraseFromConfig(configPath string, store *passphrase.Store) error {
	cfg, err := fconfig.Load(configPath)
	if err != nil || cfg.Passphrase == "" {
		return nil
	}
	if err := passphrase.Validate(cfg.Passphrase); err != nil {
		return fmt.Errorf("invalid passphrase in config file — fix or delete %s and restart: %w", configPath, err)
	}
	store.Set(cfg.Passphrase)
	return nil
}

// startNetworking initialises the TLS sync server, mDNS discovery, and peer fetcher.
// It is called at startup when a passphrase is already configured, or on first SubmitPassphrase.
func (a *App) startNetworking() {
	caTLSCert, caCert, err := tlscert.GenerateCA(a.passphraseStore.KeyBytes())
	if err != nil {
		a.log.Error("failed to generate CA cert", "error", err)
		return
	}

	leafCert, err := tlscert.GenerateLeaf(caCert, caTLSCert.PrivateKey)
	if err != nil {
		a.log.Error("failed to generate leaf cert", "error", err)
		return
	}

	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)

	a.syncServer = bsync.New(a.log, leafCert)
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

	a.peerFetcher = peersclipsync.New(a.log, a.discoverer, a.cfg.RemoteClipboardsPollInterval, a.passphraseStore, caPool)
	a.peerFetcher.OnUpdate = func() {
		runtime.EventsEmit(a.ctx, "remote:updated")
	}
	a.peerFetcher.Start(a.ctx)
}

func areWeRunningInOmarchy(themeColorPath string) bool {
	_, err := os.Stat(themeColorPath)
	return err == nil
}
