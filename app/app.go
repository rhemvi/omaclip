// Package app exposes the Wails service that wires together all business packages.
package app

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/rhemvi/omaclip/app/handlers"
	"github.com/rhemvi/omaclip/business/clipboard"
	"github.com/rhemvi/omaclip/business/copyhook"
	"github.com/rhemvi/omaclip/business/passphrase"
	"github.com/rhemvi/omaclip/business/peersclipsync"
	bsync "github.com/rhemvi/omaclip/business/sync"
	"github.com/rhemvi/omaclip/business/theme"
	osclip "github.com/rhemvi/omaclip/foundation/clipboard"
	fconfig "github.com/rhemvi/omaclip/foundation/config"
	fmdns "github.com/rhemvi/omaclip/foundation/mdns"
	"github.com/rhemvi/omaclip/foundation/peers"
	"github.com/rhemvi/omaclip/foundation/tlscert"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	ErrSpecifiedThemeFileNotFound = errors.New("specified Omarchy theme config file was not found")
	ErrDefaultThemeFilesNotFound  = errors.New("no config file was found in the default Omarchy 3 and 4 locations")
)

// Config holds all configurable values for the application.
type Config struct {
	MaxHistory                   int
	MaxPinned                    int
	MaxPngImageMB                int
	MaxNonPngImageMB             int
	ThemeColorPath               string
	ConfigPath                   string
	CopyHook                     string
	PollInterval                 time.Duration
	RemoteClipboardsPollInterval time.Duration
	RemoteClipboardsMaxHistory   int
	PeersPollInterval            time.Duration
	PeersMDNSInterface           string
	DisableRemoteClipboards      bool
	ManualPeersList              []string
	SyncServerPort               int
}

// peersProvider is satisfied by any type that can return a list of peers.
type peersProvider interface {
	Peers() []fmdns.Peer
}

// App is the Wails service. It owns startup/shutdown and delegates to business packages.
type App struct {
	wailsApp        *application.App
	ctx             context.Context
	log             *slog.Logger
	cfg             Config
	monitor         *clipboard.Monitor
	copyHook        copyhook.Runner
	colors          theme.ThemeColors
	syncServer      *bsync.Server
	discoverer      *fmdns.Discoverer
	peerFetcher     *peersclipsync.Fetcher
	passphraseStore *passphrase.Store
}

// NewApp creates an App with the provided configuration.
func NewApp(wailsApp *application.App, log *slog.Logger, cfg Config) *App {
	return &App{
		wailsApp:        wailsApp,
		cfg:             cfg,
		log:             log,
		copyHook:        copyhook.NewRunner(log, cfg.CopyHook),
		passphraseStore: &passphrase.Store{},
	}
}

// ServiceStartup initialises the clipboard monitor and optional peer networking.
func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.ctx = ctx
	a.log.Info("starting application")

	reader, writer, backend, err := osclip.NewReaderWriter(a.cfg.MaxPngImageMB, a.cfg.MaxNonPngImageMB)
	if err != nil {
		return fmt.Errorf("clipboard unavailable: %w", err)
	}
	a.log.Info("clipboard backend selected", "backend", backend)
	a.monitor = clipboard.NewMonitor(a.log, reader, writer, a.cfg.MaxHistory, a.cfg.MaxPngImageMB, a.cfg.MaxNonPngImageMB, a.cfg.PollInterval)
	a.monitor.SetRemoteMaxHistory(a.cfg.RemoteClipboardsMaxHistory)

	themeColorPath, err := areWeRunningInOmarchy(a.cfg.ThemeColorPath)
	if err == nil {
		colors, err := theme.Load(themeColorPath)
		if err != nil {
			a.log.Warn("could not load theme", "error", err)
		} else {
			a.colors = colors
			a.wailsApp.Event.Emit("theme:loaded", colors)
		}

		w := theme.NewWatcher(themeColorPath, func(c theme.ThemeColors) {
			a.colors = c
			a.wailsApp.Event.Emit("theme:loaded", c)
		})
		if err := w.Start(ctx); err != nil {
			a.log.Warn("could not watch theme file", "error", err)
		}
	} else {
		a.log.Warn(
			"no usable Omarchy theme file was found; using the built-in Tokyo Night theme and disabling the theme watcher",
			"error", err,
		)
	}

	if err := validatePassphraseFromConfig(a.cfg.ConfigPath, a.passphraseStore); err != nil {
		return fmt.Errorf(
			"invalid passphrase in config file — fix or delete %s and restart: %s",
			a.cfg.ConfigPath,
			err,
		)
	}

	if a.passphraseStore.Get() != "" && !a.cfg.DisableRemoteClipboards {
		if err := a.startNetworking(); err != nil {
			a.log.Error("failed to start networking", "error", err)
			if errors.Is(err, fmdns.ErrInterfaceNotFound) {
				return fmt.Errorf("the requested network interface is not available; pass a valid network interface or skip the flag to auto discover: %w", err)
			}
		}
	}

	a.monitor.OnNewEntry = func(entry clipboard.ClipboardEntry) {
		a.wailsApp.Event.Emit("clipboard:new", entry)
	}
	a.monitor.Start(ctx)
	return nil
}

// ServiceShutdown releases background resources when the application closes.
func (a *App) ServiceShutdown() error {
	a.log.Info("shutting down application")
	if a.monitor != nil {
		a.monitor.Stop()
	}
	if a.syncServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		a.syncServer.Shutdown(ctx)
	}
	if a.discoverer != nil {
		a.discoverer.Shutdown()
	}
	return nil
}

// GetHistory returns all clipboard entries in reverse-chronological order.
func (a *App) GetHistory() []clipboard.ClipboardEntry {
	return a.monitor.GetHistory()
}

// GetMaxPinned returns the maximum number of clipboard items that can be pinned.
func (a *App) GetMaxPinned() int {
	return a.cfg.MaxPinned
}

// SetPinnedIDs tells the monitor which entry IDs are currently pinned so they survive history trimming.
func (a *App) SetPinnedIDs(ids []string) {
	a.monitor.SetPinnedIDs(ids)
}

// CopyItem writes the entry with the given ID back to the system clipboard and triggers the copy hook.
func (a *App) CopyItem(id string) error {
	if err := a.monitor.CopyItem(id); err != nil {
		return err
	}
	a.copyHook.Trigger()
	return nil
}

// GetRemoteClipboards returns clipboard entries from all discovered peers.
func (a *App) GetRemoteClipboards() []peersclipsync.PeerClipboard {
	if a.peerFetcher == nil {
		return nil
	}
	return a.peerFetcher.GetAll()
}

// CopyRemoteItem writes the given text directly to the system clipboard and triggers the copy hook.
func (a *App) CopyRemoteItem(content string) error {
	if err := a.monitor.CopyText(content); err != nil {
		return err
	}
	a.copyHook.Trigger()
	return nil
}

// CopyRemoteImage writes base64-encoded image data from a remote peer to the system clipboard and triggers the copy hook.
func (a *App) CopyRemoteImage(imageDataBase64 string, mimeType string) error {
	if err := a.monitor.CopyImage(imageDataBase64, mimeType); err != nil {
		return err
	}
	a.copyHook.Trigger()
	return nil
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

	a.syncServer = bsync.New(a.log, leafCert, a.cfg.SyncServerPort)
	registerRoutes(a.syncServer, &handlers.ClipboardHandler{
		Monitor:         a.monitor,
		PassphraseStore: a.passphraseStore,
	})
	if err := a.syncServer.Start(); err != nil {
		return fmt.Errorf("failed to start sync https server: %w", err)
	}

	var provider peersProvider
	if len(a.cfg.ManualPeersList) > 0 {
		a.log.Info("using manual peers list", "peers", a.cfg.ManualPeersList)
		provider = peers.New(a.cfg.ManualPeersList)
	} else {
		host, _ := os.Hostname()
		a.discoverer, err = fmdns.New(
			a.log,
			a.cfg.PeersPollInterval,
			host,
			a.passphraseStore,
			a.cfg.PeersMDNSInterface,
		)
		if err != nil {
			return fmt.Errorf("failed to start mDNS discoverer: %w", err)
		}
		if err := a.discoverer.Register(a.syncServer.Port()); err != nil {
			return fmt.Errorf("failed to register mDNS service to network %w", err)
		}
		a.discoverer.Start(a.ctx)
		provider = a.discoverer
	}
	a.peerFetcher = peersclipsync.New(
		a.log,
		provider,
		a.cfg.RemoteClipboardsPollInterval,
		a.passphraseStore,
		caPool,
	)
	a.peerFetcher.OnUpdate = func() {
		a.wailsApp.Event.Emit("remote:updated")
	}
	a.peerFetcher.Start(a.ctx)
	return nil
}

func areWeRunningInOmarchy(configuredPath string) (string, error) {
	if configuredPath != "" {
		if _, err := os.Stat(configuredPath); err != nil {
			return "", fmt.Errorf("%w: %q: %w", ErrSpecifiedThemeFileNotFound, configuredPath, err)
		}
		return configuredPath, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory for Omarchy theme discovery: %w", err)
	}

	paths := []string{
		filepath.Join(home, ".local/state/omarchy/current/theme/colors.toml"),
		filepath.Join(home, ".config/omarchy/current/theme/colors.toml"),
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", ErrDefaultThemeFilesNotFound
}
