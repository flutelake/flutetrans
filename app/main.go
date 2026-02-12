package main

import (
	"context"
	"embed"

	"app/internal/services"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()
	connectionService := services.NewConnectionService()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "FluteTrans",
		Width:     1200,
		Height:    800,
		MinWidth:  960,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			connectionService.Startup(ctx)
		},
		Bind: []interface{}{
			app,
			connectionService,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
