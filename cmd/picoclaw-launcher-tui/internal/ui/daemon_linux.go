//go:build linux
// +build linux

package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/sipeed/picoclaw/cmd/picoclaw-launcher-tui/internal/daemon"
	"github.com/sipeed/picoclaw/web/backend/launcherconfig"
)

func (s *appState) daemonMenu() tview.Primitive {
	menu := NewMenu("Daemon Manager", s.buildDaemonMenuItems())
	menu.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			s.pop()
			return nil
		}
		return event
	})
	return menu
}

func refreshDaemonMenuFromState(menu *Menu, s *appState) {
	menu.applyItems(s.buildDaemonMenuItems())
}

func (s *appState) buildDaemonMenuItems() []MenuItem {
	status, statusErr := daemon.GetStatus()
	cfg, cfgErr := s.loadLauncherConfig()

	publicDesc := "Failed to load launcher config"
	if cfgErr == nil {
		if cfg.Public {
			publicDesc = "Current: enabled (restart required to apply changes)"
		} else {
			publicDesc = "Current: disabled (restart required to apply changes)"
		}
	}

	restartDesc := "Apply latest public setting to daemon"
	if s.daemonRestartRequired {
		restartDesc = "Restart required to apply public setting"
	}

	statusText := "Status unavailable"
	if statusErr == nil {
		enabledText := "disabled"
		if status.Enabled {
			enabledText = "enabled"
		}
		activeText := "inactive"
		if status.Active {
			activeText = "active"
		}
		statusText = fmt.Sprintf("enabled=%s, active=%s", enabledText, activeText)
	}

	enableDesc := "systemctl --user enable --now picoclaw-launcher.service"
	if shouldBlockEnableDaemon(s.daemonRestartRequired) {
		enableDesc = "Restart daemon first to apply pending public change"
	}

	items := []MenuItem{
		{
			Label:       "Service Status",
			Description: statusText,
			Action: func() {
				if statusErr != nil {
					s.showMessage("Status unavailable", statusErr.Error())
					return
				}
				s.showMessage("Daemon Status", statusText)
			},
		},
		{
			Label:       "Toggle Public",
			Description: publicDesc,
			Action: func() {
				s.toggleDaemonPublic()
				refreshMainMenuIfPresent(s)
				if menu, ok := s.menus["daemon"]; ok {
					refreshDaemonMenuFromState(menu, s)
				}
			},
			Disabled: cfgErr != nil,
		},
		{
			Label:       "Enable Daemon",
			Description: enableDesc,
			Action: func() {
				s.enableDaemon()
				if menu, ok := s.menus["daemon"]; ok {
					refreshDaemonMenuFromState(menu, s)
				}
			},
			Disabled: cfgErr != nil || shouldBlockEnableDaemon(s.daemonRestartRequired),
		},
		{
			Label:       "Disable Daemon",
			Description: "systemctl --user disable --now picoclaw-launcher.service",
			Action: func() {
				s.disableDaemon()
				if menu, ok := s.menus["daemon"]; ok {
					refreshDaemonMenuFromState(menu, s)
				}
			},
		},
		{
			Label:       "Restart Daemon",
			Description: restartDesc,
			Action: func() {
				s.restartDaemon()
				if menu, ok := s.menus["daemon"]; ok {
					refreshDaemonMenuFromState(menu, s)
				}
			},
			Disabled: cfgErr != nil,
		},
		{
			Label:       "View Daemon Logs",
			Description: "journalctl --user -u picoclaw-launcher.service -n 100 --no-pager",
			Action: func() {
				s.showMessage(
					"Daemon Logs",
					"Run: journalctl --user -u picoclaw-launcher.service -n 100 --no-pager",
				)
			},
		},
		{
			Label:       "Back",
			Description: "Return to main menu",
			Action: func() {
				s.pop()
			},
		},
	}
	return items
}

func (s *appState) launcherConfigPath() string {
	return launcherconfig.PathForAppConfig(s.configPath)
}

func (s *appState) loadLauncherConfig() (launcherconfig.Config, error) {
	return launcherconfig.Load(s.launcherConfigPath(), launcherconfig.Default())
}

func (s *appState) saveLauncherConfig(cfg launcherconfig.Config) error {
	return launcherconfig.Save(s.launcherConfigPath(), cfg)
}

func (s *appState) toggleDaemonPublic() {
	cfg, err := s.loadLauncherConfig()
	if err != nil {
		s.showMessage("Failed to load launcher config", err.Error())
		return
	}
	cfg.Public = !cfg.Public
	if err := s.saveLauncherConfig(cfg); err != nil {
		s.showMessage("Failed to save launcher config", err.Error())
		return
	}
	s.daemonRestartRequired = true
	if cfg.Public {
		s.showMessage("Public enabled", "Public mode enabled. Restart daemon to apply.")
		return
	}
	s.showMessage("Public disabled", "Public mode disabled. Restart daemon to apply.")
}

func (s *appState) enableDaemon() {
	if shouldBlockEnableDaemon(s.daemonRestartRequired) {
		s.showMessage("Restart required", "Restart daemon first to apply pending public change")
		return
	}
	cfg, err := s.loadLauncherConfig()
	if err != nil {
		s.showMessage("Failed to load launcher config", err.Error())
		return
	}
	if err := daemon.Enable(s.configPath, cfg.Public); err != nil {
		s.showMessage("Enable daemon failed", err.Error())
		return
	}
	s.daemonRestartRequired = false
	s.showMessage("Daemon enabled", "User service is enabled and started")
}

func (s *appState) disableDaemon() {
	if err := daemon.Disable(); err != nil {
		s.showMessage("Disable daemon failed", err.Error())
		return
	}
	s.showMessage("Daemon disabled", "User service is disabled and stopped")
}

func (s *appState) restartDaemon() {
	cfg, err := s.loadLauncherConfig()
	if err != nil {
		s.showMessage("Failed to load launcher config", err.Error())
		return
	}
	if err := daemon.Restart(s.configPath, cfg.Public); err != nil {
		s.showMessage("Restart daemon failed", err.Error())
		return
	}
	s.daemonRestartRequired = false
	s.showMessage("Daemon restarted", "User service restarted with latest settings")
}
