package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// FreeRideTool adapts the FreeRide logic (from clawhub/free-ride) for PicoClaw.
// It manages OpenRouter's free models and configures them as fallbacks.
type FreeRideTool struct {
	configPath string
	reloadFunc func() error
}

func NewFreeRideTool(configPath string, reloadFunc func() error) *FreeRideTool {
	return &FreeRideTool{
		configPath: configPath,
		reloadFunc: reloadFunc,
	}
}

func (t *FreeRideTool) Name() string {
	return "freeride"
}

func (t *FreeRideTool) Description() string {
	return "FreeRide gives you unlimited free AI in PicoClaw by automatically managing OpenRouter's free models. " +
		"Use 'auto' to configure best model + fallbacks, or 'list' to see available free models."
}

func (t *FreeRideTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"enum":        []string{"auto", "list", "status", "settimeout"},
				"description": "The command to run: 'auto' (configures models), 'list' (shows free models), 'status' (checks current setup), 'settimeout' (sets request timeout)",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "For 'list', how many models to show. For 'auto', how many fallbacks to configure.",
				"default":     5,
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "For 'settimeout', the request timeout in seconds (default 300)",
				"default":     300,
			},
		},
		"required": []string{"command"},
	}
}

type openRouterModel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextLength int    `json:"context_length"`
	Pricing       struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
	} `json:"pricing"`
	SupportedParameters []string `json:"supported_parameters"`
	Created             int64    `json:"created"`
}

func (t *FreeRideTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	cmd, _ := args["command"].(string)
	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	timeout := 300
	switch v := args["timeout"].(type) {
	case float64:
		timeout = int(v)
	case int:
		timeout = v
	}

	switch cmd {
	case "list":
		return t.handleList(ctx, limit)
	case "auto":
		return t.handleAuto(ctx, limit)
	case "status":
		return t.handleStatus()
	case "settimeout":
		return t.handleSetTimeout(timeout)
	default:
		return ErrorResult(fmt.Sprintf("unknown command: %s", cmd))
	}
}

func (t *FreeRideTool) fetchFreeModels(ctx context.Context) ([]openRouterModel, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenRouter API returned status %d", resp.StatusCode)
	}

	var wrapper struct {
		Data []openRouterModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, err
	}

	var freeModels []openRouterModel
	for _, m := range wrapper.Data {
		// Only consider free models
		if m.Pricing.Prompt == "0" || m.Pricing.Prompt == "0.0" || m.Pricing.Prompt == "0.00" {
			// CRITICAL: PeakClaw requires tool support for its steering logic.
			// Filter out models that don't explicitly support function calling.
			hasTools := false
			for _, p := range m.SupportedParameters {
				if p == "tools" {
					hasTools = true
					break
				}
			}
			if !hasTools {
				continue
			}

			// Blacklist known tool-blind models with inaccurate metadata
			lowerID := strings.ToLower(m.ID)
			if strings.Contains(lowerID, "lyria") || strings.Contains(lowerID, "liquid") {
				continue
			}

			freeModels = append(freeModels, m)
		}
	}

	// Rank models
	sort.Slice(freeModels, func(i, j int) bool {
		return scoreModel(freeModels[i]) > scoreModel(freeModels[j])
	})

	return freeModels, nil
}

func scoreModel(m openRouterModel) float64 {
	score := 0.0

	// Context length (40%) - normalize against 128k
	ctxScore := float64(m.ContextLength) / 128000.0
	if ctxScore > 1.0 {
		ctxScore = 1.0
	}
	score += ctxScore * 0.4

	// Capabilities (30%) - tools, vision, prompt caching, etc.
	capabilityScore := 0.0
	for _, p := range m.SupportedParameters {
		if p == "tools" {
			capabilityScore += 0.5
		}
		if p == "response_format" {
			capabilityScore += 0.5
		}
	}
	if capabilityScore > 1.0 {
		capabilityScore = 1.0
	}
	score += capabilityScore * 0.3

	// Recency (20%) - newer is better
	// Normalize against 2 years ago
	twoYearsAgo := time.Now().AddDate(-2, 0, 0).Unix()
	now := time.Now().Unix()
	if m.Created > twoYearsAgo {
		recencyScore := float64(m.Created-twoYearsAgo) / float64(now-twoYearsAgo)
		score += recencyScore * 0.2
	}

	// Provider Trust (10%) - hardcoded list of trusted names
	trustNames := []string{
		"google",
		"meta",
		"nvidia",
		"mistral",
		"anthropic",
		"openai",
		"microsoft",
		"qwen",
		"deepseek",
	}
	for _, name := range trustNames {
		if strings.Contains(strings.ToLower(m.ID), name) {
			score += 0.1
			break
		}
	}

	return score
}

func (t *FreeRideTool) handleList(ctx context.Context, limit int) *ToolResult {
	models, err := t.fetchFreeModels(ctx)
	if err != nil {
		return ErrorResult(fmt.Errorf("failed to fetch models: %w", err).Error())
	}

	if len(models) == 0 {
		return SilentResult("No free models found on OpenRouter.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d free models on OpenRouter (ranked by quality):\n\n", len(models)))
	for i, m := range models {
		if i >= limit {
			break
		}
		sb.WriteString(fmt.Sprintf("%d. **%s** (%s)\n", i+1, m.Name, m.ID))
		sb.WriteString(fmt.Sprintf("   Context: %d tokens | Score: %.2f\n", m.ContextLength, scoreModel(m)))
		sb.WriteString(fmt.Sprintf("   Parameters: %s\n\n", strings.Join(m.SupportedParameters, ", ")))
	}

	return UserResult(sb.String())
}

func (t *FreeRideTool) handleAuto(ctx context.Context, limit int) *ToolResult {
	models, err := t.fetchFreeModels(ctx)
	if err != nil {
		return ErrorResult(fmt.Errorf("failed to fetch models: %w", err).Error())
	}

	if len(models) == 0 {
		return ErrorResult("No free models found on OpenRouter.")
	}

	cfgObj, err := config.LoadConfig(t.configPath)
	if err != nil {
		return ErrorResult(fmt.Errorf("failed to load config: %w", err).Error())
	}

	// 1. Add models to ModelList if not present
	// 2. Collect all valid free models for fallbacks (new AND existing)
	var fallbackModels []string
	for i, m := range models {
		if i >= limit {
			break
		}
		modelName := strings.ReplaceAll(m.ID, "/", "-")
		if !modelExists(cfgObj, modelName) {
			mc := &config.ModelConfig{
				ModelName: modelName,
				Model:     m.ID,
				Protocol:  "openrouter",
				Enabled:   true,
			}
			mc.SetAPIKey("env://OPENROUTER_API_KEY")
			cfgObj.ModelList = append(cfgObj.ModelList, mc)
		}
		fallbackModels = append(fallbackModels, modelName)
	}

	// 2. Set fallbacks for the default agent
	if len(fallbackModels) > 0 {
		// Update AgentDefaults fallbacks
		cfgObj.Agents.Defaults.ModelFallbacks = append(cfgObj.Agents.Defaults.ModelFallbacks, fallbackModels...)
		// Deduplicate fallbacks
		cfgObj.Agents.Defaults.ModelFallbacks = uniqueStrings(cfgObj.Agents.Defaults.ModelFallbacks)

		if err := config.SaveConfig(t.configPath, cfgObj); err != nil {
			return ErrorResult(fmt.Errorf("failed to save config: %w", err).Error())
		}

		msg := fmt.Sprintf(
			"Success! Added %d free models as fallbacks: %s.\n",
			len(fallbackModels),
			strings.Join(fallbackModels, ", "),
		)
		msg += "Re-loading configuration to apply changes..."

		if t.reloadFunc != nil {
			if err := t.reloadFunc(); err != nil {
				return ErrorResult(fmt.Sprintf("%s\nFailed to reload: %v", msg, err))
			}
		}

		return UserResult(msg)
	}

	return UserResult(
		"No new free models to add. Your configuration is up to date.",
	)
}

func (t *FreeRideTool) handleStatus() *ToolResult {
	cfgObj, err := config.LoadConfig(t.configPath)
	if err != nil {
		return ErrorResult(fmt.Errorf("failed to load config: %w", err).Error())
	}

	var sb strings.Builder
	sb.WriteString("FreeRide Status:\n")
	sb.WriteString(fmt.Sprintf("- Primary Model: %s\n", cfgObj.Agents.Defaults.GetModelName()))
	sb.WriteString(fmt.Sprintf("- Fallback Models: %s\n", strings.Join(cfgObj.Agents.Defaults.ModelFallbacks, ", ")))

	// Check for OpenRouter models in fallbacks
	openRouterCount := 0
	for _, fb := range cfgObj.Agents.Defaults.ModelFallbacks {
		if strings.Contains(strings.ToLower(fb), "openrouter") || isKnownOpenRouterAlias(cfgObj, fb) {
			openRouterCount++
		}
	}
	sb.WriteString(fmt.Sprintf("- Managed Free Models: %d\n", openRouterCount))

	return UserResult(sb.String())
}

func (t *FreeRideTool) handleSetTimeout(timeoutSeconds int) *ToolResult {
	if timeoutSeconds < 30 {
		return ErrorResult("timeout must be at least 30 seconds")
	}

	cfgObj, err := config.LoadConfig(t.configPath)
	if err != nil {
		return ErrorResult(fmt.Errorf("failed to load config: %w", err).Error())
	}

	updated := 0
	for _, mc := range cfgObj.ModelList {
		// Only update OpenRouter models (free models)
		protocol := strings.ToLower(mc.Protocol)
		if protocol == "openrouter" {
			mc.RequestTimeout = timeoutSeconds
			updated++
		}
	}

	if updated == 0 {
		return ErrorResult("no OpenRouter models found in config. Run 'freeride auto' first.")
	}

	if err := config.SaveConfig(t.configPath, cfgObj); err != nil {
		return ErrorResult(fmt.Errorf("failed to save config: %w", err).Error())
	}

	msg := fmt.Sprintf("Set request timeout to %d seconds for %d OpenRouter models.\n", timeoutSeconds, updated)
	msg += "Re-loading configuration to apply changes..."

	if t.reloadFunc != nil {
		if err := t.reloadFunc(); err != nil {
			return ErrorResult(fmt.Sprintf("%s\nFailed to reload: %v", msg, err))
		}
	}

	return UserResult(msg)
}

func modelExists(cfg *config.Config, modelName string) bool {
	for _, m := range cfg.ModelList {
		if m.ModelName == modelName {
			return true
		}
	}
	return false
}

func isKnownOpenRouterAlias(cfg *config.Config, modelName string) bool {
	for _, m := range cfg.ModelList {
		if m.ModelName == modelName {
			if strings.HasPrefix(m.Model, "openrouter/") {
				return true
			}
			if strings.ToLower(m.Protocol) == "openrouter" {
				return true
			}
		}
	}
	return false
}

func uniqueStrings(input []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range input {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
