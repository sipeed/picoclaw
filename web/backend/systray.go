//go:build (!darwin && !freebsd) || cgo

package main

import (
	_ "embed"
	"fmt"

	"fyne.io/systray"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/web/backend/utils"
)

func runTray() {
	systray.Run(onReady, shutdownApp)
}

// onReady is called when the system tray is ready
func onReady() {
	// Set icon and tooltip
	systray.SetIcon(getIcon())
	systray.SetTooltip(fmt.Sprintf(T(AppTooltip), appName))

	// Create menu items
	mOpen := systray.AddMenuItem(T(MenuOpen), T(MenuOpenTooltip))
	mAbout := systray.AddMenuItem(T(MenuAbout), T(MenuAboutTooltip))

	// Add version info under About menu
	mVersion := mAbout.AddSubMenuItem(fmt.Sprintf(T(MenuVersion), appVersion), T(MenuVersionTooltip))
	mVersion.Disable()
	mRepo := mAbout.AddSubMenuItem(T(MenuGitHub), "")
	mDocs := mAbout.AddSubMenuItem(T(MenuDocs), "")

	systray.AddSeparator()

	// Add restart option
	mRestart := systray.AddMenuItem(T(MenuRestart), T(MenuRestartTooltip))

	systray.AddSeparator()

	// Quit option
	mQuit := systray.AddMenuItem(T(MenuQuit), T(MenuQuitTooltip))

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				if err := openBrowser(); err != nil {
					logger.Errorf("Failed to open browser: %v", err)
				}

			case <-mVersion.ClickedCh:
				// Version info - do nothing, just shows current version

			case <-mRepo.ClickedCh:
				if err := utils.OpenBrowser("https://github.com/sipeed/picoclaw"); err != nil {
					logger.Errorf("Failed to open GitHub: %v", err)
				}

			case <-mDocs.ClickedCh:
				if err := utils.OpenBrowser(T(DocUrl)); err != nil {
					logger.Errorf("Failed to open docs: %v", err)
				}

			case <-mRestart.ClickedCh:
				fmt.Println("Restart request received...")
				if apiHandler != nil {
					if pid, err := apiHandler.RestartGateway(); err != nil {
						logger.Errorf("Failed to restart gateway: %v", err)
					} else {
						logger.Infof("Gateway restarted (PID: %d)", pid)
					}
				}

			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()

	if !*noBrowser {
		// Auto-open browser after systray is ready (if not disabled)
		// Check no-browser flag via environment or pass as parameter if needed
		if err := openBrowser(); err != nil {
			logger.Errorf("Warning: Failed to auto-open browser: %v", err)
		}
	}
}

// getIcon returns the system tray icon
func getIcon() []byte {
	return iconData
}
