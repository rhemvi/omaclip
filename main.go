package main

import (
	"embed"

	"clipmaster/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	a := app.NewApp()

	err := wails.Run(&options.App{
		Title:  "clipmaster",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 26, G: 27, B: 38, A: 1},
		OnStartup:        a.Startup,
		OnShutdown:       a.Shutdown,
		Bind: []any{
			a,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
