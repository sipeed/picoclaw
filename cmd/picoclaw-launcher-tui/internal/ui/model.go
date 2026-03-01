package ui

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	picoclawconfig "github.com/sipeed/picoclaw/pkg/config"
)

func (s *appState) modelMenu() tview.Primitive {
	items := make([]MenuItem, 0, 2+len(s.config.ModelList))
	items = append(items,
		MenuItem{Label: "Back", Description: "Return to main menu", Action: func() { s.pop() }},
		MenuItem{
			Label:       "Add model",
			Description: "Append a new model entry",
			Action: func() {
				s.addModel(
					picoclawconfig.ModelConfig{ModelName: "new-model", Model: "openai/gpt-5.2"},
				)
				s.push(
					fmt.Sprintf("model-%d", len(s.config.ModelList)-1),
					s.modelForm(len(s.config.ModelList)-1),
				)
			},
		},
	)
	currentModel := strings.TrimSpace(s.config.Agents.Defaults.Model)
	for i := range s.config.ModelList {
		index := i
		model := s.config.ModelList[i]
		isValid := isModelValid(model)
		desc := model.APIBase
		if desc == "" {
			desc = model.AuthMethod
		}
		if desc == "" {
			desc = "api_key required"
		}
		label := fmt.Sprintf("%s (%s)", model.ModelName, model.Model)
		if model.ModelName == currentModel && currentModel != "" {
			label = "* " + label
		}
		isSelected := model.ModelName == currentModel && currentModel != ""
		items = append(items, MenuItem{
			Label:       label,
			Description: desc,
			MainColor:   modelStatusColor(isValid, isSelected),
			Action: func() {
				s.push(fmt.Sprintf("model-%d", index), s.modelForm(index))
			},
		})
	}

	menu := NewMenu("Models", items)
	menu.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			s.pop()
			return nil
		}
		if event.Rune() == 'q' {
			s.pop()
			return nil
		}
		if event.Rune() == ' ' {
			row, _ := menu.GetSelection()
			if row > 0 && row <= len(s.config.ModelList) {
				model := s.config.ModelList[row-1]
				if !isModelValid(model) {
					s.showMessage(
						"Invalid model",
						"Select a model with api_key or oauth auth_method",
					)
					return nil
				}
				s.config.Agents.Defaults.Model = model.ModelName
				s.dirty = true
				refreshModelMenu(menu, s.config.Agents.Defaults.Model, s.config.ModelList)
				refreshMainMenuIfPresent(s)
			}
			return nil
		}
		return event
	})
	return menu
}

func (s *appState) modelForm(index int) tview.Primitive {
	model := &s.config.ModelList[index]
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(fmt.Sprintf("Model: %s", model.ModelName))
	form.SetButtonBackgroundColor(tcell.NewRGBColor(80, 250, 123))
	form.SetButtonTextColor(tcell.NewRGBColor(12, 13, 22))

	addInput(form, "Model Name", model.ModelName, func(value string) {
		model.ModelName = value
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["model"]; ok {
			refreshModelMenuFromState(menu, s)
		}
	})
	addInput(form, "Model", model.Model, func(value string) {
		model.Model = value
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["model"]; ok {
			refreshModelMenuFromState(menu, s)
		}
	})
	addInput(form, "API Base", model.APIBase, func(value string) {
		model.APIBase = value
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["model"]; ok {
			refreshModelMenuFromState(menu, s)
		}
	})
	addInput(form, "API Key", model.APIKey, func(value string) {
		model.APIKey = value
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["model"]; ok {
			refreshModelMenuFromState(menu, s)
		}
	})
	addInput(form, "Proxy", model.Proxy, func(value string) {
		model.Proxy = value
	})
	addInput(form, "Auth Method", model.AuthMethod, func(value string) {
		model.AuthMethod = value
		s.dirty = true
		refreshMainMenuIfPresent(s)
		if menu, ok := s.menus["model"]; ok {
			refreshModelMenuFromState(menu, s)
		}
	})
	addInput(form, "Connect Mode", model.ConnectMode, func(value string) {
		model.ConnectMode = value
	})
	addInput(form, "Workspace", model.Workspace, func(value string) {
		model.Workspace = value
	})
	addInput(form, "Max Tokens Field", model.MaxTokensField, func(value string) {
		model.MaxTokensField = value
	})
	addIntInput(form, "RPM", model.RPM, func(value int) {
		model.RPM = value
	})
	addIntInput(form, "Request Timeout", model.RequestTimeout, func(value int) {
		model.RequestTimeout = value
	})

	form.AddButton("Delete", func() {
		s.deleteModel(index)
	})
	form.AddButton("Test", func() {
		s.testModel(model)
	})
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

func addInput(form *tview.Form, label, value string, onChange func(string)) {
	form.AddInputField(label, value, 128, nil, func(text string) {
		onChange(strings.TrimSpace(text))
	})
}

func addIntInput(form *tview.Form, label string, value int, onChange func(int)) {
	form.AddInputField(label, fmt.Sprintf("%d", value), 16, nil, func(text string) {
		var parsed int
		if _, err := fmt.Sscanf(strings.TrimSpace(text), "%d", &parsed); err == nil {
			onChange(parsed)
		}
	})
}

func (s *appState) addModel(model picoclawconfig.ModelConfig) {
	s.config.ModelList = append(s.config.ModelList, model)
}

func (s *appState) deleteModel(index int) {
	if index < 0 || index >= len(s.config.ModelList) {
		return
	}
	s.config.ModelList = append(s.config.ModelList[:index], s.config.ModelList[index+1:]...)
	s.pop()
}

func modelStatusColor(valid bool, selected bool) *tcell.Color {
	if valid {
		color := tview.Styles.PrimaryTextColor
		return &color
	}
	color := tcell.ColorGray
	return &color
}

func refreshModelMenu(menu *Menu, currentModel string, models []picoclawconfig.ModelConfig) {
	for i, model := range models {
		row := i + 1
		label := fmt.Sprintf("%s (%s)", model.ModelName, model.Model)
		isValid := isModelValid(model)
		if model.ModelName == currentModel && currentModel != "" {
			label = "* " + label
		}
		cell := menu.GetCell(row, 0)
		if cell != nil {
			cell.SetText(label)
			isSelected := model.ModelName == currentModel && currentModel != ""
			color := modelStatusColor(isValid, isSelected)
			if color != nil {
				cell.SetTextColor(*color)
			}
		}
	}
}

func refreshModelMenuFromState(menu *Menu, s *appState) {
	items := make([]MenuItem, 0, 2+len(s.config.ModelList))
	items = append(items,
		MenuItem{Label: "Back", Description: "Return to main menu", Action: func() { s.pop() }},
		MenuItem{
			Label:       "Add model",
			Description: "Append a new model entry",
			Action: func() {
				s.addModel(
					picoclawconfig.ModelConfig{ModelName: "new-model", Model: "openai/gpt-5.2"},
				)
				s.push(
					fmt.Sprintf("model-%d", len(s.config.ModelList)-1),
					s.modelForm(len(s.config.ModelList)-1),
				)
			},
		},
	)
	currentModel := strings.TrimSpace(s.config.Agents.Defaults.Model)
	for i := range s.config.ModelList {
		index := i
		model := s.config.ModelList[i]
		isValid := isModelValid(model)
		desc := model.APIBase
		if desc == "" {
			desc = model.AuthMethod
		}
		if desc == "" {
			desc = "api_key required"
		}
		label := fmt.Sprintf("%s (%s)", model.ModelName, model.Model)
		if model.ModelName == currentModel && currentModel != "" {
			label = "* " + label
		}
		isSelected := model.ModelName == currentModel && currentModel != ""
		items = append(items, MenuItem{
			Label:       label,
			Description: desc,
			MainColor:   modelStatusColor(isValid, isSelected),
			Action: func() {
				s.push(fmt.Sprintf("model-%d", index), s.modelForm(index))
			},
		})
	}
	menu.applyItems(items)
}

func isModelValid(model picoclawconfig.ModelConfig) bool {
	hasKey := strings.TrimSpace(model.APIKey) != "" ||
		strings.TrimSpace(model.AuthMethod) == "oauth"
	hasModel := strings.TrimSpace(model.Model) != ""
	return hasKey && hasModel
}

func (s *appState) testModel(model *picoclawconfig.ModelConfig) {
	if model == nil {
		return
	}
	if strings.TrimSpace(model.APIKey) == "" {
		s.showMessage("Missing API Key", "Set api_key before testing")
		return
	}
	base := strings.TrimSpace(model.APIBase)
	if base == "" {
		s.showMessage("Missing API Base", "Set api_base before testing")
		return
	}
	modelID := strings.TrimSpace(model.Model)
	if modelID == "" {
		s.showMessage("Missing Model", "Set model before testing")
		return
	}
	if !strings.HasPrefix(modelID, "openai/") {
		s.showMessage("Unsupported model", "Only openai/* models are supported for test")
		return
	}
	modelName := strings.TrimPrefix(modelID, "openai/")
	endpoint := strings.TrimRight(base, "/") + "/chat/completions"

	payload := fmt.Sprintf(
		`{"model":"%s","messages":[{"role":"user","content":"ping"}],"max_tokens":1}`,
		modelName,
	)
	client := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequest("POST", endpoint, strings.NewReader(payload))
	if err != nil {
		s.showMessage("Test failed", err.Error())
		return
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(model.APIKey))

	resp, err := client.Do(request)
	if err != nil {
		s.showMessage("Test failed", err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.showMessage("Test OK", resp.Status)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	s.showMessage(
		"Test failed",
		fmt.Sprintf("%s: %s", resp.Status, strings.TrimSpace(string(body))),
	)
}
