package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	configstore "jane/cmd/picoclaw-launcher-tui/internal/config"
	picoclawconfig "jane/pkg/config"
)

type appState struct {
	app         *tview.Application
	pages       *tview.Pages
	stack       []string
	config      *picoclawconfig.Config
	configPath  string
	gatewayCmd  *exec.Cmd
	menus       map[string]*Menu
	original    []byte
	hasOriginal bool
	backupPath  string
	dirty       bool
	logPath     string
}

func Run() error {
	applyStyles()
	cfg, err := configstore.Load()
	if err != nil {
		return err
	}
	path, err := configstore.ConfigPath()
	if err != nil {
		return err
	}

	if cfg == nil {
		cfg = picoclawconfig.DefaultConfig()
	}

	originalData, hasOriginal := loadOriginalConfig(path)
	backupPath := path + ".bak"
	if hasOriginal {
		_ = writeBackupConfig(backupPath, originalData)
	}

	logPath := filepath.Join(filepath.Dir(path), "gateway.log")
	state := &appState{
		app:         tview.NewApplication(),
		pages:       tview.NewPages(),
		config:      cfg,
		configPath:  path,
		menus:       map[string]*Menu{},
		original:    originalData,
		hasOriginal: hasOriginal,
		backupPath:  backupPath,
		logPath:     logPath,
	}

	state.push("main", state.mainMenu())

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(bannerView(), 6, 0, false)
	root.AddItem(state.pages, 0, 1, true)
	root.AddItem(footerView(), 1, 0, false)

	if err := state.app.SetRoot(root, true).EnableMouse(false).Run(); err != nil {
		return err
	}
	return nil
}

func (s *appState) push(name string, primitive tview.Primitive) {
	s.pages.AddPage(name, primitive, true, true)
	s.stack = append(s.stack, name)
	s.pages.SwitchToPage(name)
	if menu, ok := primitive.(*Menu); ok {
		s.menus[name] = menu
	}
}

func (s *appState) pop() {
	if len(s.stack) == 0 {
		return
	}
	last := s.stack[len(s.stack)-1]
	s.pages.RemovePage(last)
	s.stack = s.stack[:len(s.stack)-1]
	if len(s.stack) == 0 {
		s.app.Stop()
		return
	}
	current := s.stack[len(s.stack)-1]
	s.pages.SwitchToPage(current)
	if menu, ok := s.menus[current]; ok {
		s.refreshMenu(current, menu)
	}
}

func (s *appState) mainMenu() tview.Primitive {
	menu := NewMenu("Menu", nil)
	refreshMainMenu(menu, s)
	menu.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			s.requestExit()
			return nil
		}

		return event
	})

	return menu
}

func (s *appState) refreshMenu(name string, menu *Menu) {
	switch name {
	case "main":
		refreshMainMenu(menu, s)
	case "model":
		refreshModelMenuFromState(menu, s)
	case "channel":
		refreshChannelMenuFromState(menu, s)
	}
}

func (s *appState) countChannels() (enabled int, total int) {
	c := s.config.Channels
	entries := []bool{
		c.Telegram.Enabled,
		c.Discord.Enabled,
		c.QQ.Enabled,
		c.MaixCam.Enabled,
		c.WhatsApp.Enabled,
		c.DingTalk.Enabled,
		c.Slack.Enabled,
		c.Matrix.Enabled,
		c.LINE.Enabled,
		c.OneBot.Enabled,
	}
	total = len(entries)
	for _, v := range entries {
		if v {
			enabled++
		}
	}
	return enabled, total
}

func refreshMainMenuIfPresent(s *appState) {
	if menu, ok := s.menus["main"]; ok {
		refreshMainMenu(menu, s)
	}
}

func refreshMainMenu(menu *Menu, s *appState) {
	selectedModel := s.selectedModelName()
	modelReady := selectedModel != ""
	channelReady := s.hasEnabledChannel()
	enabledCount, totalChannels := s.countChannels()
	gatewayRunning := s.gatewayCmd != nil || s.isGatewayRunning()

	gatewayLabel := "Start Gateway"
	gatewayDescription := "Launch gateway for channels"
	if gatewayRunning {
		gatewayLabel = "Stop Gateway"
		gatewayDescription = "Gateway running"
	}

	items := []MenuItem{
		{
			Label:       rootModelLabel(selectedModel),
			Description: rootModelDescription(),
			Action: func() {
				s.push("model", s.modelMenu())
			},
			MainColor: func() *tcell.Color {
				if modelReady {
					return nil
				}
				color := tcell.ColorGray
				return &color
			}(),
		},
		{
			Label:       rootChannelLabel(channelReady),
			Description: fmt.Sprintf("%d/%d enabled", enabledCount, totalChannels),
			Action: func() {
				s.push("channel", s.channelMenu())
			},
			MainColor: func() *tcell.Color {
				if channelReady {
					return nil
				}
				color := tcell.ColorGray
				return &color
			}(),
		},
		{
			Label:       "Start Talk",
			Description: "Open picoclaw agent in terminal",
			Action: func() {
				s.requestStartTalk()
			},
			Disabled: !modelReady,
		},
		{
			Label:       gatewayLabel,
			Description: gatewayDescription,
			Action: func() {
				if gatewayRunning {
					s.stopGateway()
				} else {
					s.requestStartGateway()
				}
				refreshMainMenu(menu, s)
			},
			Disabled: !gatewayRunning && (!modelReady || !channelReady),
		},
		{
			Label:       "View Gateway Log",
			Description: "Open gateway.log",
			Action: func() {
				s.viewGatewayLog()
			},
		},
		{
			Label:       "Exit",
			Description: "Exit the TUI",
			Action: func() {
				s.requestExit()
			},
		},
	}
	menu.applyItems(items)
}

func (s *appState) applyChangesValidated() bool {
	if err := s.config.ValidateModelList(); err != nil {
		s.showMessage("Validation failed", err.Error())
		return false
	}
	if err := s.validateAgentModel(); err != nil {
		s.showMessage("Validation failed", err.Error())
		return false
	}
	if err := configstore.Save(s.config); err != nil {
		s.showMessage("Save failed", err.Error())
		return false
	}
	if data, err := os.ReadFile(s.configPath); err == nil {
		s.original = data
		s.hasOriginal = true
		_ = writeBackupConfig(s.backupPath, data)
	}
	return true
}

func (s *appState) requestExit() {
	if s.dirty {
		s.confirmApplyOrDiscard(func() {
			s.app.Stop()
		}, func() {
			s.discardChanges()
			s.app.Stop()
		})
		return
	}
	s.app.Stop()
}

func (s *appState) requestStartTalk() {
	if s.dirty {
		s.confirmApplyOrDiscard(func() {
			s.startTalk()
		}, func() {
			s.startTalk()
		})
		return
	}
	s.startTalk()
}

func (s *appState) requestStartGateway() {
	if s.dirty {
		s.confirmApplyOrDiscard(func() {
			s.startGateway()
		}, func() {
			s.startGateway()
		})
		return
	}
	s.startGateway()
}

func (s *appState) viewGatewayLog() {
	data, err := os.ReadFile(s.logPath)
	if err != nil {
		s.showMessage("Log not found", "gateway.log not found")
		return
	}
	text := tview.NewTextView()
	text.SetBorder(true).SetTitle("Gateway Log")
	text.SetText(string(data))
	text.SetDoneFunc(func(key tcell.Key) {
		s.pages.RemovePage("log")
	})
	text.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			s.pages.RemovePage("log")
			return nil
		}
		return event
	})
	s.pages.AddPage("log", text, true, true)
}

func (s *appState) selectedModelName() string {
	modelName := strings.TrimSpace(s.config.Agents.Defaults.Model)
	if modelName == "" {
		return ""
	}
	if !s.isActiveModelValid() {
		return ""
	}
	return modelName
}

func rootModelLabel(selected string) string {
	if selected == "" {
		return "Model (None)"
	}
	return "Model (" + selected + ")"
}

func rootModelDescription() string {
	return "Using SPACE to choose your model"
}

func rootChannelLabel(valid bool) string {
	if !valid {
		return "Channel (no channel enabled)"
	}
	return "Channel"
}

func (s *appState) startTalk() {
	if !s.isActiveModelValid() {
		s.showMessage("Model required", "Select a valid model before starting talk")
		return
	}
	if !s.applyChangesValidated() {
		return
	}
	s.app.Suspend(func() {
		cmd := exec.Command("picoclaw", "agent")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	})
}

func (s *appState) startGateway() {
	if !s.isActiveModelValid() {
		s.showMessage("Model required", "Select a valid model before starting gateway")
		return
	}
	if !s.hasEnabledChannel() {
		s.showMessage("Channel required", "Enable at least one channel before starting gateway")
		return
	}
	if !s.applyChangesValidated() {
		return
	}
	_ = stopGatewayProcess()
	cmd := exec.Command("picoclaw", "gateway")
	logFile, err := os.OpenFile(s.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		s.showMessage("Gateway failed", err.Error())
		return
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		s.showMessage("Gateway failed", err.Error())
		_ = logFile.Close()
		return
	}
	_ = logFile.Close()
	s.gatewayCmd = cmd
}

func (s *appState) stopGateway() {
	_ = stopGatewayProcess()
	if s.gatewayCmd != nil && s.gatewayCmd.Process != nil {
		_ = s.gatewayCmd.Process.Kill()
	}
	s.gatewayCmd = nil
}

func (s *appState) isGatewayRunning() bool {
	return isGatewayProcessRunning()
}

func (s *appState) validateAgentModel() error {
	modelName := strings.TrimSpace(s.config.Agents.Defaults.Model)
	if modelName == "" {
		return nil
	}
	_, err := s.config.GetModelConfig(modelName)
	return err
}

func (s *appState) isActiveModelValid() bool {
	modelName := strings.TrimSpace(s.config.Agents.Defaults.Model)
	if modelName == "" {
		return false
	}
	cfg, err := s.config.GetModelConfig(modelName)
	if err != nil {
		return false
	}
	hasKey := strings.TrimSpace(cfg.APIKey) != "" || strings.TrimSpace(cfg.AuthMethod) == "oauth"
	hasModel := strings.TrimSpace(cfg.Model) != ""
	return hasKey && hasModel
}

func (s *appState) hasEnabledChannel() bool {
	c := s.config.Channels
	return c.Telegram.Enabled || c.Discord.Enabled || c.QQ.Enabled || c.MaixCam.Enabled ||
		c.WhatsApp.Enabled || c.DingTalk.Enabled || c.Slack.Enabled ||
		c.Matrix.Enabled || c.LINE.Enabled || c.OneBot.Enabled
}

func (s *appState) confirmApplyOrDiscard(onApply func(), onDiscard func()) {
	if s.pages.HasPage("apply") {
		return
	}
	modal := tview.NewModal().
		SetText("Apply changes or discard before continuing?").
		AddButtons([]string{"Cancel", "Discard", "Apply"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			s.pages.RemovePage("apply")
			switch buttonLabel {
			case "Discard":
				s.discardChanges()
				if onDiscard != nil {
					onDiscard()
				}
			case "Apply":
				if s.applyChangesValidated() {
					s.dirty = false
					if onApply != nil {
						onApply()
					}
				}
			}
		})
	modal.SetBorder(true)
	s.pages.AddPage("apply", modal, true, true)
}

func (s *appState) discardChanges() {
	if s.hasOriginal {
		_ = writeOriginalConfig(s.configPath, s.original)
	} else {
		_ = os.Remove(s.configPath)
	}
	_ = os.Remove(s.backupPath)
	if cfg, err := configstore.Load(); err == nil && cfg != nil {
		s.config = cfg
	}
	s.dirty = false
	refreshMainMenuIfPresent(s)
}

func (s *appState) showMessage(title, message string) {
	if s.pages.HasPage("message") {
		return
	}
	modal := tview.NewModal().
		SetText(strings.TrimSpace(message)).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			s.pages.RemovePage("message")
		})
	modal.SetTitle(title).SetBorder(true)
	modal.SetBackgroundColor(tview.Styles.ContrastBackgroundColor)
	modal.SetTextColor(tview.Styles.PrimaryTextColor)
	modal.SetButtonBackgroundColor(tcell.NewRGBColor(112, 102, 255))
	modal.SetButtonTextColor(tview.Styles.PrimaryTextColor)
	s.pages.AddPage("message", modal, true, true)
}

func loadOriginalConfig(path string) ([]byte, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false
		}
		return nil, false
	}
	return data, true
}

func writeOriginalConfig(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

func writeBackupConfig(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}
