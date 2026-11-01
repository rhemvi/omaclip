// Package main is the entry point for the omaclip desktop application.
package main

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/rhemvi/omaclip/app"
	"github.com/rhemvi/omaclip/foundation/logger"
	"github.com/rhemvi/omaclip/foundation/vcs"

	"github.com/ardanlabs/conf/v3"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

const appVersion = "0.10.0"

type appConfig struct {
	ThemeColorPath string `conf:"help:full path to an Omarchy theme colors.toml file; when unset Omarchy 4 and 3 locations are detected automatically"`
	ConfigPath     string `conf:"help:path to the omaclip config file (default: $HOME/.config/omaclip/config.json)"`
	CopyHook       string `conf:"help:shell command to run after an item is copied from Omaclip"`
	Debug          bool   `conf:"default:false,help:enable debug log level"`
	Clipboard      struct {
		MaxHistory       int           `conf:"default:50,help:maximum number of clipboard entries to keep in history"`
		MaxPinned        int           `conf:"default:3,help:maximum number of clipboard items that can be pinned at the top of the list"`
		MaxPngImageMB    int           `conf:"default:5,help:maximum PNG image size in megabytes to accept into clipboard history"`
		MaxNonPngImageMB int           `conf:"default:2,help:maximum non-PNG image size in megabytes to accept into clipboard history (e.g. JPEG from file managers or macOS clipboard)"`
		PollInterval     time.Duration `conf:"default:2s,help:in case we fallback to polling how often to poll the system clipboard"`
	}
	RemoteClipboards struct {
		MaxHistory   int           `conf:"default:5,help:maximum number of local clipboard entries to transmit to remote peers"`
		PollInterval time.Duration `conf:"default:2s,help:how often to fetch clipboard entries from remote peers"`
		Disable      bool          `conf:"default:false,help:disable fetching clipboards from remote peers"`
	}
	Peers struct {
		PollInterval  time.Duration `conf:"default:2s,help:how often to browse for peers on the local network via mDNS"`
		MDNSInterface string        `conf:"help:bind mDNS to a specific network interface (e.g. wlan0) instead of all interfaces"`
		List          []string      `conf:"help:manually specify peers [name@]ip:port (e.g. dell@192.168.1.53:36742 or 192.168.1.53:36742) will disable mDNS peer discovery if set"`
	}
	SyncServer struct {
		Port int `conf:"default:0,help:port the clipboard sync HTTPS server will listen on; zero uses an OS-assigned port"`
	}
	conf.Version
}

func main() {
	if err := run(); err != nil {
		logger.New(slog.LevelInfo).Error("application error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := appConfig{
		ConfigPath: filepath.Join(os.Getenv("HOME"), ".config/omaclip/config.json"),
	}
	cfg.Build = vcs.Version(appVersion)

	help, err := conf.Parse("OMACLIP", &cfg)
	if errors.Is(err, conf.ErrHelpWanted) || errors.Is(err, conf.ErrVersionWanted) {
		fmt.Println(help)
		return nil
	}
	if err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	minLevel := slog.LevelInfo
	if cfg.Debug {
		minLevel = slog.LevelDebug
	}
	log := logger.New(minLevel)

	cfgStr, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("building config string: %w", err)
	}
	log.Info("application configuration", "config", cfgStr)

	wailsApp := application.New(application.Options{
		Name:        "omaclip",
		Description: "A clipboard manager with local network sync",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	service := app.NewApp(wailsApp, log, app.Config{
		MaxHistory:                   cfg.Clipboard.MaxHistory,
		MaxPinned:                    cfg.Clipboard.MaxPinned,
		MaxPngImageMB:                cfg.Clipboard.MaxPngImageMB,
		MaxNonPngImageMB:             cfg.Clipboard.MaxNonPngImageMB,
		ThemeColorPath:               cfg.ThemeColorPath,
		ConfigPath:                   cfg.ConfigPath,
		CopyHook:                     cfg.CopyHook,
		PollInterval:                 cfg.Clipboard.PollInterval,
		RemoteClipboardsPollInterval: cfg.RemoteClipboards.PollInterval,
		RemoteClipboardsMaxHistory:   cfg.RemoteClipboards.MaxHistory,
		PeersPollInterval:            cfg.Peers.PollInterval,
		PeersMDNSInterface:           cfg.Peers.MDNSInterface,
		DisableRemoteClipboards:      cfg.RemoteClipboards.Disable,
		ManualPeersList:              cfg.Peers.List,
		SyncServerPort:               cfg.SyncServer.Port,
	})

	wailsApp.RegisterService(application.NewService(service))
	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "omaclip",
		Width:            1024,
		Height:           768,
		BackgroundColour: application.NewRGB(26, 27, 38),
		URL:              "/",
	})

	if err := wailsApp.Run(); err != nil {
		return err
	}

	return nil
}
