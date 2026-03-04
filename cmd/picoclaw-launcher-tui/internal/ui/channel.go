package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	picoclawconfig "github.com/sipeed/picoclaw/pkg/config"
)

func makeChannelMenu(s *appState) *tview.List {
	menu := tview.NewList()
	menu.SetBorder(true).SetTitle("Channels").SetTitleAlign(tview.AlignCenter)
	menu.ShowSecondaryText(true)
	menu.SetWrapAround(true)
	menu.SetHighlightFullLine(true)

	refreshChannelMenuFromState(menu, s)
	return menu
}

func refreshChannelMenuFromState(menu *tview.List, s *appState) {
	menu.Clear()

	items := []MenuItem{
		channelItem(
			"WhatsApp",
			"Configure WhatsApp Bridge",
			s.config.Channels.WhatsApp.Enabled,
			func() { s.push("whatsapp_form", s.whatsappForm()) },
		),
		{Label: "Back", Description: "Return to main menu", Action: func() { s.pop() }},
	}

	for _, item := range items {
		menu.AddItem(item.Label, item.Description, 0, item.Action)
	}

	menu.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			s.pop()
			return nil
		}
		return event
	})
}

func (s *appState) whatsappForm() tview.Primitive {
	cfg := &s.config.Channels.WhatsApp
	form := baseChannelForm("WhatsApp", cfg.Enabled, s.makeChannelOnEnabled(&cfg.Enabled))
	form.AddInputField("Bridge URL", cfg.BridgeURL, 128, nil, func(text string) {
		cfg.BridgeURL = strings.TrimSpace(text)
	})
	addAllowFromField(form, &cfg.AllowFrom)
	return wrapWithBack(form, s)
}

func (s *appState) makeChannelOnEnabled(enabledPtr *bool) func(bool) {
	return func(v bool) {
		*enabledPtr = v
		s.dirty = true
		refreshMainMenuIfPresent(s)
	}
}

func addAllowFromField(form *tview.Form, allowFrom *picoclawconfig.FlexibleStringSlice) {
	form.AddInputField("Allow From", strings.Join(*allowFrom, ","), 128, nil, func(text string) {
		*allowFrom = splitCSV(text)
	})
}

func baseChannelForm(title string, enabled bool, onEnabled func(bool)) *tview.Form {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(fmt.Sprintf("Channel: %s", title))
	form.SetButtonBackgroundColor(tcell.NewRGBColor(80, 250, 123))
	form.SetButtonTextColor(tcell.NewRGBColor(12, 13, 22))
	form.AddCheckbox("Enabled", enabled, func(checked bool) {
		onEnabled(checked)
	})
	return form
}

func wrapWithBack(form *tview.Form, s *appState) tview.Primitive {
	form.AddButton("Back", func() {
		s.pop()
	})
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			s.pop()
			return nil
		}
		return event
	})
	return form
}

func splitCSV(input string) picoclawconfig.FlexibleStringSlice {
	parts := strings.Split(strings.TrimSpace(input), ",")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		cleaned = append(cleaned, value)
	}
	return cleaned
}

func addIntField(form *tview.Form, label string, value int, onChange func(int)) {
	form.AddInputField(label, fmt.Sprintf("%d", value), 16, nil, func(text string) {
		var parsed int
		if _, err := fmt.Sscanf(strings.TrimSpace(text), "%d", &parsed); err == nil {
			onChange(parsed)
		}
	})
}

func addInt64Field(form *tview.Form, label string, value int64, onChange func(int64)) {
	form.AddInputField(label, fmt.Sprintf("%d", value), 16, nil, func(text string) {
		var parsed int64
		if _, err := fmt.Sscanf(strings.TrimSpace(text), "%d", &parsed); err == nil {
			onChange(parsed)
		}
	})
}

func channelItem(label, description string, enabled bool, action MenuAction) MenuItem {
	item := MenuItem{
		Label:       label,
		Description: description,
		Action:      action,
	}
	if !enabled {
		color := tcell.ColorGray
		item.MainColor = &color
	}
	return item
}
