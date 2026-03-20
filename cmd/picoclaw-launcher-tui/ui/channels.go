// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (a *App) newChannelsPage() tview.Primitive {
	list := tview.NewList()
	list.SetBorder(true).
		SetTitle(" [#00f0ff::b] COMMUNICATION CHANNELS ").
		SetTitleColor(tcell.NewHexColor(0x00f0ff)).
		SetBorderColor(tcell.NewHexColor(0x00f0ff))
	list.SetMainTextColor(tcell.NewHexColor(0xe0e0e0))
	list.SetSecondaryTextColor(tcell.NewHexColor(0x808080))
	list.SetSelectedStyle(
		tcell.StyleDefault.Background(tcell.NewHexColor(0xff00ff)).Foreground(tcell.NewHexColor(0x050510)),
	)
	list.SetHighlightFullLine(true)
	list.SetBackgroundColor(tcell.NewHexColor(0x050510))

	rebuild := func() {
		sel := list.GetCurrentItem()
		list.Clear()

		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		configPath := filepath.Join(home, ".picoclaw", "config.json")

		var cfg map[string]any
		if data, err := os.ReadFile(configPath); err == nil {
			_ = json.Unmarshal(data, &cfg)
		}

		if chRaw, ok := cfg["channels"].(map[string]any); ok {
			for name, ch := range chRaw {
				chMap, ok := ch.(map[string]any)
				enabled := "disabled"
				if ok {
					if e, ok := chMap["enabled"].(bool); ok && e {
						enabled = "enabled"
					}
				}
				list.AddItem(name, fmt.Sprintf("Status: %s", enabled), 0, func() {
					a.showChannelEditForm(configPath, name, chMap)
				})
			}
		}

		if sel >= 0 && sel < list.GetItemCount() {
			list.SetCurrentItem(sel)
		}
	}
	rebuild()

	a.pageRefreshFns["channels"] = rebuild

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			return a.goBack()
		}
		return event
	})

	return a.buildShell("channels", list, " [#ff00ff]Enter:[-] edit  [#ff2a2a]ESC:[-] back ")
}

func (a *App) showChannelEditForm(configPath, channelName string, existing map[string]any) {
	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(" [::b]EDIT CHANNEL ").
		SetTitleColor(tcell.NewHexColor(0x39ff14)).
		SetBorderColor(tcell.NewHexColor(0x00f0ff))
	form.SetBackgroundColor(tcell.NewHexColor(0x1a1a2e))
	form.SetFieldBackgroundColor(tcell.NewHexColor(0x050510))
	form.SetFieldTextColor(tcell.NewHexColor(0x00f0ff))
	form.SetLabelColor(tcell.NewHexColor(0xe0e0e0))
	form.SetButtonBackgroundColor(tcell.NewHexColor(0xff00ff))
	form.SetButtonTextColor(tcell.NewHexColor(0xffffff))

	fields := make(map[string]*tview.InputField)
	var nameField *tview.InputField

	if channelName == "" {
		nameField = tview.NewInputField().
			SetLabel("Channel Name").
			SetText("").
			SetFieldWidth(28)
		form.AddFormItem(nameField)
	}

	for k, v := range existing {
		if reflect.ValueOf(v).Kind() == reflect.Map || reflect.ValueOf(v).Kind() == reflect.Slice {
			continue
		}
		valStr := fmt.Sprintf("%v", v)
		field := tview.NewInputField().
			SetLabel(k).
			SetText(valStr).
			SetFieldWidth(28)
		form.AddFormItem(field)
		fields[k] = field
	}

	form.AddButton("SAVE", func() {
		var cfg map[string]any
		if data, err := os.ReadFile(configPath); err == nil {
			if err := json.Unmarshal(data, &cfg); err != nil {
				cfg = make(map[string]any)
			}
		} else {
			cfg = make(map[string]any)
		}

		if _, ok := cfg["channels"]; !ok {
			cfg["channels"] = make(map[string]any)
		}
		channels, ok := cfg["channels"].(map[string]any)
		if !ok {
			channels = make(map[string]any)
			cfg["channels"] = channels
		}

		finalName := channelName
		if channelName == "" {
			if nameField == nil || nameField.GetText() == "" {
				a.showError("Channel name is required")
				return
			}
			finalName = nameField.GetText()
		}

		updated := make(map[string]any)
		if existing != nil {
			for k, v := range existing {
				updated[k] = v
			}
		}
		for k, field := range fields {
			val := field.GetText()
			if val == "true" {
				updated[k] = true
			} else if val == "false" {
				updated[k] = false
			} else if num, err := strconv.Atoi(val); err == nil {
				updated[k] = num
			} else {
				updated[k] = val
			}
		}

		if channelName != "" && finalName != channelName {
			delete(channels, channelName)
		}
		channels[finalName] = updated

		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			a.showError(fmt.Sprintf("Failed to save config: %v", err))
			return
		}
		if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
			a.showError(fmt.Sprintf("Failed to create config directory: %v", err))
			return
		}
		if err := os.WriteFile(configPath, data, 0o600); err != nil {
			a.showError(fmt.Sprintf("Failed to write config: %v", err))
			return
		}

		a.hideModal("channel-edit")
		a.goBack()
	})

	form.AddButton("CANCEL", func() {
		a.hideModal("channel-edit")
	})

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			a.hideModal("channel-edit")
			return nil
		}
		return event
	})

	a.showModal("channel-edit", centeredForm(form, 4, 20))
}
