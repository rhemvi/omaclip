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
	ThemeColorPath string `conf:""`
	Clipboard      struct {
		MaxHistory   int           `conf:"default:50"`
		PollInterval time.Duration `conf:"default:500ms"`
	}
	conf.Version
}

func main() {
	log := logger.New()

	if err := run(log); err != nil {
		log.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg := appConfig{
		ThemeColorPath: filepath.Join(os.Getenv("HOME"), ".config/omarchy/current/theme/colors.toml"),
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

	cfgStr, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("building config string: %w", err)
	}
	log.Info("application configuration", "config", cfgStr)

	application := app.NewApp(log, app.Config{
		MaxHistory:     cfg.Clipboard.MaxHistory,
		ThemeColorPath: cfg.ThemeColorPath,
		PollInterval:   cfg.Clipboard.PollInterval,
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
