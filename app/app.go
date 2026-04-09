// Package app is the Wails bind target that wires together all business packages.
package app

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/rhemvi/omaclip/app/handlers"
	"github.com/rhemvi/omaclip/business/clipboard"
	"github.com/rhemvi/omaclip/business/passphrase"
	"github.com/rhemvi/omaclip/business/peersclipsync"
	bsync "github.com/rhemvi/omaclip/business/sync"
	"github.com/rhemvi/omaclip/business/theme"
	osclip "github.com/rhemvi/omaclip/foundation/clipboard"
	fconfig "github.com/rhemvi/omaclip/foundation/config"
	fmdns "github.com/rhemvi/omaclip/foundation/mdns"
	"github.com/rhemvi/omaclip/foundation/tlscert"

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
	PeersMDNSInterface           string
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

	reader, writer, backend, err := osclip.NewReaderWriter()
	if err != nil {
		a.log.Error("clipboard unavailable", "error", err)
		os.Exit(1)
	}
	a.log.Info("clipboard backend selected", "backend", backend)
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
		a.log.Error(fmt.Sprintf("invalid passphrase in config file — fix or delete %s and restart: %s", a.cfg.ConfigPath, err))
		os.Exit(1)
	}

	if a.passphraseStore.Get() != "" && !a.cfg.DisableRemoteClipboards {
		if err := a.startNetworking(); err != nil {
			a.log.Error("failed to start networking", "error", err)
			if errors.Is(err, fmdns.ErrInterfaceNotFound) {
				a.log.Error("the requested network interface is not available, please pass a valid network interface or skip the flag to auto discover")
				os.Exit(1)
			}
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

// CopyRemoteImage writes base64-encoded image data from a remote peer to the system clipboard.
func (a *App) CopyRemoteImage(imageDataBase64 string, mimeType string) error {
	return a.monitor.CopyImage(imageDataBase64, mimeType)
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
	if err := a.startNetworking(); err != nil {
		a.log.Error("failed to start networking", "error", err)
	}
	return nil
}

// validatePassphraseFromConfig checks if a passphrase is already set in the config file, and if so validates it and sets it in the store.
func validatePassphraseFromConfig(configPath string, store *passphrase.Store) error {
	cfg, err := fconfig.Load(configPath)
	if err != nil || cfg.Passphrase == "" {
		return nil
	}
	if err := passphrase.Validate(cfg.Passphrase); err != nil {
		return err
	}
	store.Set(cfg.Passphrase)
	return nil
}

// startNetworking initialises the TLS sync server, mDNS discovery, and peer fetcher.
// It is called at startup when a passphrase is already configured, or on first SubmitPassphrase.
func (a *App) startNetworking() error {
	caTLSCert, caCert, err := tlscert.GenerateCA(a.passphraseStore.KeyBytes())
	if err != nil {
		return fmt.Errorf("failed to generate CA cert: %w", err)
	}

	leafCert, err := tlscert.GenerateLeaf(caCert, caTLSCert.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to generate leaf cert %w", err)
	}

	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)

	a.syncServer = bsync.New(a.log, leafCert)
	registerRoutes(a.syncServer, &handlers.ClipboardHandler{
		Monitor:         a.monitor,
		MaxHistory:      a.cfg.RemoteClipboardsMaxHistory,
		PassphraseStore: a.passphraseStore,
	})
	if err := a.syncServer.Start(); err != nil {
		return fmt.Errorf("failed to start sync https server: %w", err)
	}


	host, _ := os.Hostname()
	discoverer, err := fmdns.New(a.log, a.cfg.PeersPollInterval, host, a.passphraseStore, a.cfg.PeersMDNSInterface)
	if err != nil {
		return fmt.Errorf("failed to start mDNS discoverer: %w", err)
	}
	a.discoverer = discoverer
	if err := a.discoverer.Register(a.syncServer.Port()); err != nil {
		return fmt.Errorf("failed to register mDNS service to network %w", err)
	}
	a.discoverer.Start(a.ctx)

	a.peerFetcher = peersclipsync.New(
		a.log,
		a.discoverer,
		a.cfg.RemoteClipboardsPollInterval,
		a.passphraseStore,
		caPool,
	)
	a.peerFetcher.OnUpdate = func() {
		runtime.EventsEmit(a.ctx, "remote:updated")
	}
	a.peerFetcher.Start(a.ctx)
	return nil
}

func areWeRunningInOmarchy(themeColorPath string) bool {
	_, err := os.Stat(themeColorPath)
	return err == nil
}
