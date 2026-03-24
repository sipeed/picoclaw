// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	tuicfg "github.com/sipeed/picoclaw/cmd/picoclaw-launcher-tui/config"
)

type modelsAPIResponse struct {
	Data []modelEntry `json:"data"`
}

type modelEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (a *App) newModelsPage(schemeName, userName, baseURL string) tview.Primitive {
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(0, 0)
	table.SetBorder(true).
		SetTitle(fmt.Sprintf(" [#00f0ff::b] MODELS · %s / %s ", schemeName, userName)).
		SetTitleColor(tcell.NewHexColor(0x00f0ff)).
		SetBorderColor(tcell.NewHexColor(0x00f0ff))
	table.SetSelectedStyle(
		tcell.StyleDefault.Background(tcell.NewHexColor(0xff00ff)).Foreground(tcell.NewHexColor(0xffffff)),
	)
	table.SetBackgroundColor(tcell.NewHexColor(0x050510))

	var modelIDs []string

	status := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText("[#ffff00]FETCHING MODELS...[-]")
	status.SetBackgroundColor(tcell.NewHexColor(0x050510))

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(status, 1, 0, false).
		AddItem(table, 0, 1, false)

	apiKey := a.resolveKey(schemeName, userName)

	go func() {
		var entries []modelEntry
		var err error
		if apiKey == "" {
			err = fmt.Errorf("key is required")
		} else {
			entries, err = fetchModels(baseURL, apiKey)
		}

		a.modelCacheMu.Lock()
		if a.modelCache == nil {
			a.modelCache = make(map[string][]modelEntry)
		}
		if err == nil && len(entries) > 0 {
			a.modelCache[cacheKey(schemeName, userName)] = entries
		} else {
			a.modelCache[cacheKey(schemeName, userName)] = nil
		}
		a.modelCacheMu.Unlock()

		a.tapp.QueueUpdateDraw(func() {
			if err != nil {
				status.SetText(fmt.Sprintf("[#ff2a2a]ERROR: %s[-]", err.Error()))
				table.SetCell(0, 0, tview.NewTableCell(" (failed to load models)"))
				a.tapp.SetFocus(table)
				return
			}
			if len(entries) == 0 {
				status.SetText("[#ff2a2a]NO MODELS RETURNED[-]")
				table.SetCell(0, 0, tview.NewTableCell(" (no models available)"))
				a.tapp.SetFocus(table)
				return
			}

			status.SetText(fmt.Sprintf("[#39ff14]%d MODEL(S) LOADED[-]", len(entries)))
			for i, m := range entries {
				modelIDs = append(modelIDs, m.ID)
				table.SetCell(i, 0,
					tview.NewTableCell(fmt.Sprintf("%3d", i+1)).
						SetAlign(tview.AlignRight).
						SetTextColor(tcell.NewHexColor(0x808080)).
						SetSelectable(false),
				)
				table.SetCell(i, 1,
					tview.NewTableCell(" "+m.ID).
						SetAlign(tview.AlignLeft).
						SetExpansion(1).
						SetTextColor(tcell.NewHexColor(0xe0e0e0)),
				)
			}
			a.tapp.SetFocus(table)
		})
	}()

	table.SetSelectedFunc(func(row, _ int) {
		if row < 0 || row >= len(modelIDs) {
			return
		}
		a.cfg.Provider.Current = tuicfg.ProviderCurrent{
			Scheme: schemeName,
			User:   userName,
			Model:  modelIDs[row],
		}
		a.save()

		// Trigger model selected callback if set
		if a.OnModelSelected != nil && a.cfg.Model.Type == "provider" {
			scheme := a.cfg.Provider.SchemeByName(schemeName)
			if scheme == nil {
				a.goBack()
				return
			}
			var user tuicfg.User
			for _, u := range a.cfg.Provider.Users {
				if u.Scheme == schemeName && u.Name == userName {
					user = u
					break
				}
			}
			a.OnModelSelected(*scheme, user, modelIDs[row])
		}

		a.goBack()
	})

	return a.buildShell("models", flex, " [#39ff14]Enter:[-] select  [#ff00ff]ESC:[-] back ")
}

func (a *App) resolveKey(schemeName, userName string) string {
	for _, u := range a.cfg.Provider.Users {
		if u.Scheme == schemeName && u.Name == userName {
			return u.Key
		}
	}
	return ""
}

func fetchModels(baseURL, apiKey string) ([]modelEntry, error) {
	url := strings.TrimRight(baseURL, "/") + "/models"

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result modelsAPIResponse
	if err := json.Unmarshal(body, &result); err == nil && len(result.Data) > 0 {
		return result.Data, nil
	}

	var arr []modelEntry
	if err := json.Unmarshal(body, &arr); err == nil {
		return arr, nil
	}

	return nil, fmt.Errorf(
		"decode response: unrecognized shape: %s",
		strings.TrimSpace(string(body[:min(len(body), 256)])),
	)
}
