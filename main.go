package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"strings"

	"nanite/cli"
	"nanite/config"
	"nanite/vault"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Intercept CLI subcommands before launching the Wails GUI.
	// A bare word first argument (no leading "-") is treated as a subcommand.
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		cfg := config.LoadOrDefault()
		ctx := context.Background()
		mgr, err := vault.NewManager(ctx, cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "nanite: vault error:", err)
			os.Exit(1)
		}
		defer mgr.CloseAll()
		cli.Run(os.Args[1:], cfg, mgr)
		os.Exit(0)
	}

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:         "NANITE",
		Width:         800,
		Height:        830,
		MinWidth:      800,
		MinHeight:     830,
		DisableResize: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// BackgroundColour: &options.RGBA{R: 31, G: 41, B: 55, A: 255},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Debug: options.Debug{
			OpenInspectorOnStartup: os.Getenv("DEBUG") == "1",
		},
		Frameless: false,
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  true,
				HideTitleBar:               false,
				FullSizeContent:            true,
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
