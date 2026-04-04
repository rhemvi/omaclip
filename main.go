// Package main is the entry point for the clipmaster desktop application.
package main

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"clipmaster/app"
	"clipmaster/foundation/logger"
	"clipmaster/foundation/vcs"

	"github.com/ardanlabs/conf/v3"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

type appConfig struct {
	ThemeColorPath string `conf:"help:fullpath to the Omarchy theme colors.toml file (default: $HOME/.config/omarchy/current/theme/colors.toml)"`
	ConfigPath     string `conf:"help:path to the clipmaster config file (default: $HOME/.config/clipmaster/config.json)"`
	Debug          bool   `conf:"default:false,help:enable debug log level"`
	Clipboard      struct {
		MaxHistory   int           `conf:"default:50,help:maximum number of clipboard entries to keep in history"`
		PollInterval time.Duration `conf:"default:500ms,help:how often to poll the system clipboard for changes"`
	}
	RemoteClipboards struct {
		MaxHistory   int           `conf:"default:5,help:maximum number of local clipboard entries to transmit to remote peers"`
		PollInterval time.Duration `conf:"default:1s,help:how often to fetch clipboard entries from remote peers"`
		Disable      bool          `conf:"default:false,help:disable fetching clipboards from remote peers"`
	}
	Peers struct {
		PollInterval time.Duration `conf:"default:2s,help:how often to browse for peers on the local network via mDNS"`
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
		ThemeColorPath: filepath.Join(os.Getenv("HOME"), ".config/omarchy/current/theme/colors.toml"),
		ConfigPath:     filepath.Join(os.Getenv("HOME"), ".config/clipmaster/config.json"),
	}
	cfg.Build = vcs.Version()

	help, err := conf.Parse("CLIPMASTER", &cfg)
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

	application := app.NewApp(log, app.Config{
		MaxHistory:                   cfg.Clipboard.MaxHistory,
		ThemeColorPath:               cfg.ThemeColorPath,
		ConfigPath:                   cfg.ConfigPath,
		PollInterval:                 cfg.Clipboard.PollInterval,
		RemoteClipboardsPollInterval: cfg.RemoteClipboards.PollInterval,
		RemoteClipboardsMaxHistory:   cfg.RemoteClipboards.MaxHistory,
		PeersPollInterval:            cfg.Peers.PollInterval,
		DisableRemoteClipboards:      cfg.RemoteClipboards.Disable,
	})

	if err := wails.Run(&options.App{
		Title:  "clipmaster",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 26, G: 27, B: 38, A: 1},
		OnStartup:        application.Startup,
		OnShutdown:       application.Shutdown,
		Bind: []any{
			application,
		},
	}); err != nil {
		return err
	}

	return nil
}
