package main

import (
	_ "embed"

	"github.com/energye/systray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed icon.ico
var appIcon []byte

// setupTray initializes the system tray icon and menu.
// Uses energye/systray which works alongside Wails without thread conflicts.
func (a *App) setupTray() {
	go systray.Run(func() {
		// onReady
		systray.SetIcon(appIcon)
		systray.SetTitle("PicoClaw")
		systray.SetTooltip("PicoClaw Launcher")

		// Left click → show window
		systray.SetOnClick(func(menu systray.IMenu) {
			wailsRuntime.WindowShow(a.ctx)
		})

		// Right click → show context menu
		systray.SetOnRClick(func(menu systray.IMenu) {
			menu.ShowMenu()
		})

		mShow := systray.AddMenuItem("Show Window", "Show the launcher window")
		mShow.Click(func() {
			wailsRuntime.WindowShow(a.ctx)
		})

		systray.AddSeparator()

		mStart := systray.AddMenuItem("Start Gateway", "Start PicoClaw gateway service")
		mStart.Click(func() {
			a.StartGateway()
		})

		mStop := systray.AddMenuItem("Stop Gateway", "Stop PicoClaw gateway service")
		mStop.Click(func() {
			a.StopGateway()
		})

		systray.AddSeparator()

		mQuit := systray.AddMenuItem("Exit", "Quit PicoClaw Launcher")
		mQuit.Click(func() {
			a.StopGateway()
			a.forceQuit = true
			systray.Quit()
			wailsRuntime.Quit(a.ctx)
		})
	}, func() {
		// onExit - cleanup
	})
}
