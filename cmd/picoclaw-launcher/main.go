// PicoClaw Launcher - Desktop GUI for PicoClaw AI Agent
//
// A Wails v2 desktop application that provides:
//   - Service status monitoring and control
//   - AI chat interface
//   - Configuration editor
//
// Usage:
//
//	wails build -o picoclaw-launcher.exe
//	./picoclaw-launcher [config.json]

package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend
var assets embed.FS

func defaultConfigPath() string {
	if p := os.Getenv("PICOCLAW_CONFIG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", "config.json")
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "PicoClaw Launcher - Desktop GUI for PicoClaw AI Agent\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [config.json]\n", os.Args[0])
	}
	flag.Parse()

	configPath := defaultConfigPath()
	if flag.NArg() > 0 {
		configPath = flag.Arg(0)
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to resolve config path: %v\n", err)
		os.Exit(1)
	}

	app := NewApp(absPath)

	err = wails.Run(&options.App{
		Title:     "PicoClaw Launcher",
		Width:     960,
		Height:    640,
		MinWidth:  720,
		MinHeight: 480,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			if app.forceQuit {
				return false // allow quit
			}
			// Hide to tray instead of quitting
			wailsRuntime.WindowHide(ctx)
			return true
		},
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: false,
			WebviewUserDataPath:               "",
			Theme:                             windows.SystemDefault,
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
