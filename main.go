package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	application := NewApp()

	// Configure platform-specific options
	windowsOptions := configureWindowsOptions()
	macOptions := configureMacOptions()
	linuxOptions := configureLinuxOptions()

	// Create application with options
	err := wails.Run(&options.App{
		Title:            "Application Updater",
		Width:            1024,
		Height:           768,
		MinWidth:         800,
		MinHeight:        600,
		MaxWidth:         1920,
		MaxHeight:        1080,
		DisableResize:    false,
		Fullscreen:       false,
		AlwaysOnTop:      false,
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Windows:                  windowsOptions,
		Mac:                      macOptions,
		Linux:                    linuxOptions,
		OnDomReady:               application.DomReady,
		OnShutdown:               application.Shutdown,
		OnBeforeClose:            application.BeforeClose,
		EnableDefaultContextMenu: false,
		Bind: []interface{}{
			application,
		},
	})

	if err != nil {
		log.Printf("Application failed to start: %v\n", err)
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// configureWindowsOptions returns Windows-specific options
func configureWindowsOptions() *windows.Options {
	return &windows.Options{
		WebviewIsTransparent:              false,
		WindowIsTranslucent:               false,
		DisableWindowIcon:                 false,
		DisableFramelessWindowDecorations: false,
		WebviewUserDataPath:               "",
		WebviewBrowserPath:                "",
		Theme:                             windows.SystemDefault,
	}
}

// configureMacOptions returns Mac-specific options
func configureMacOptions() *mac.Options {
	return &mac.Options{
		TitleBar: &mac.TitleBar{
			TitlebarAppearsTransparent: false,
			HideTitle:                  false,
			HideTitleBar:               false,
			FullSizeContent:            false,
			UseToolbar:                 false,
			HideToolbarSeparator:       true,
		},
		WebviewIsTransparent: false,
		WindowIsTranslucent:  false,
		About: &mac.AboutInfo{
			Title:   "Application Updater",
			Message: "© 2023 Sophon. All rights reserved.",
		},
	}
}

// configureLinuxOptions returns Linux-specific options
func configureLinuxOptions() *linux.Options {
	return &linux.Options{
		WindowIsTranslucent: false,
		WebviewGpuPolicy:    linux.WebviewGpuPolicyAlways,
	}
}
