//go:build !linux
// +build !linux

package ui

import "github.com/rivo/tview"

func (s *appState) daemonMenu() tview.Primitive {
	menu := NewMenu("Daemon Manager", []MenuItem{
		{
			Label:       "Unsupported",
			Description: "Linux only",
			Action: func() {
				s.showMessage("Unsupported", "Daemon manager is only available on Linux")
			},
		},
		{
			Label:       "Back",
			Description: "Return to main menu",
			Action: func() {
				s.pop()
			},
		},
	})
	return menu
}

func refreshDaemonMenuFromState(menu *Menu, s *appState) {
	menu.applyItems([]MenuItem{
		{
			Label:       "Unsupported",
			Description: "Linux only",
			Action: func() {
				s.showMessage("Unsupported", "Daemon manager is only available on Linux")
			},
		},
		{
			Label:       "Back",
			Description: "Return to main menu",
			Action: func() {
				s.pop()
			},
		},
	})
}
