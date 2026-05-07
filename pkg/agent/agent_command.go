// PicoClaw - Ultra-lightweight personal AI agent

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func (al *AgentLoop) handleCommand(
	ctx context.Context,
	msg bus.InboundMessage,
	agent *AgentInstance,
	opts *processOptions,
) (string, bool) {
	normalizeProcessOptionsInPlace(opts)

	if !commands.HasCommandPrefix(msg.Content) {
		return "", false
	}

	if matched, handled, reply := al.applyExplicitSkillCommand(msg.Content, agent, opts); matched {
		return reply, handled
	}

	if al.cmdRegistry == nil {
		return "", false
	}

	rt := al.buildCommandsRuntime(ctx, agent, opts)
	executor := commands.NewExecutor(al.cmdRegistry, rt)

	var commandReply string
	result := executor.Execute(ctx, commands.Request{
		Channel:  msg.Channel,
		ChatID:   msg.ChatID,
		SenderID: msg.SenderID,
		Text:     msg.Content,
		Reply: func(text string) error {
			commandReply = text
			return nil
		},
	})

	switch result.Outcome {
	case commands.OutcomeHandled:
		if result.Err != nil {
			return mapCommandError(result), true
		}
		if commandReply != "" {
			return commandReply, true
		}
		return "", true
	default: // OutcomePassthrough — let the message fall through to LLM
		return "", false
	}
}

func (al *AgentLoop) applyExplicitSkillCommand(
	raw string,
	agent *AgentInstance,
	opts *processOptions,
) (matched bool, handled bool, reply string) {
	normalizeProcessOptionsInPlace(opts)

	cmdName, ok := commands.CommandName(raw)
	if !ok || cmdName != "use" {
		return false, false, ""
	}

	if agent == nil || agent.ContextBuilder == nil {
		return true, true, commandsUnavailableSkillMessage()
	}

	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) < 2 {
		return true, true, buildUseCommandHelp(agent)
	}

	arg := strings.TrimSpace(parts[1])
	if strings.EqualFold(arg, "clear") || strings.EqualFold(arg, "off") {
		if opts != nil {
			al.clearPendingSkills(opts.Dispatch.SessionKey)
		}
		return true, true, "Cleared pending skill override."
	}

	skillName, ok := agent.ContextBuilder.ResolveSkillName(arg)
	if !ok {
		return true, true, fmt.Sprintf("Unknown skill: %s\nUse /list skills to see installed skills.", arg)
	}

	if len(parts) < 3 {
		if opts == nil || strings.TrimSpace(opts.Dispatch.SessionKey) == "" {
			return true, true, commandsUnavailableSkillMessage()
		}
		al.setPendingSkills(opts.Dispatch.SessionKey, []string{skillName})
		return true, true, fmt.Sprintf(
			"Skill %q is armed for your next message. Send your next prompt normally, or use /use clear to cancel.",
			skillName,
		)
	}

	message := strings.TrimSpace(strings.Join(parts[2:], " "))
	if message == "" {
		return true, true, buildUseCommandHelp(agent)
	}

	if opts != nil {
		opts.ForcedSkills = append(opts.ForcedSkills, skillName)
		opts.Dispatch.UserMessage = message
		opts.UserMessage = message
	}

	return true, false, ""
}

func (al *AgentLoop) buildCommandsRuntime(
	ctx context.Context,
	agent *AgentInstance,
	opts *processOptions,
) *commands.Runtime {
	normalizeProcessOptionsInPlace(opts)

	registry := al.GetRegistry()
	cfg := al.GetConfig()
	rt := &commands.Runtime{
		Config:          cfg,
		ListAgentIDs:    registry.ListAgentIDs,
		ListDefinitions: al.cmdRegistry.Definitions,
		ListMCPServers: func(ctx context.Context) []commands.MCPServerInfo {
			if cfg == nil {
				return nil
			}

			if len(cfg.Tools.MCP.Servers) == 0 {
				return nil
			}

			if err := al.ensureMCPInitialized(ctx); err != nil {
				logger.WarnCF("agent", "Failed to refresh MCP status for command",
					map[string]any{
						"error": err.Error(),
					})
			}

			connected := make(map[string]int)
			if manager := al.mcp.getManager(); manager != nil {
				for serverName, conn := range manager.GetServers() {
					connected[serverName] = len(conn.Tools)
				}
			}

			servers := make([]commands.MCPServerInfo, 0, len(cfg.Tools.MCP.Servers))
			for serverName, serverCfg := range cfg.Tools.MCP.Servers {
				toolCount, isConnected := connected[serverName]
				servers = append(servers, commands.MCPServerInfo{
					Name:      serverName,
					Enabled:   serverCfg.Enabled,
					Deferred:  serverIsDeferred(cfg.Tools.MCP.Discovery.Enabled, serverCfg),
					Connected: isConnected,
					ToolCount: toolCount,
				})
			}

			sort.Slice(servers, func(i, j int) bool {
				return strings.ToLower(servers[i].Name) < strings.ToLower(servers[j].Name)
			})

			return servers
		},
		ListMCPTools: func(ctx context.Context, serverName string) ([]commands.MCPToolInfo, error) {
			if cfg == nil {
				return nil, fmt.Errorf("command unavailable: config not loaded")
			}

			serverName = strings.TrimSpace(serverName)
			if serverName == "" {
				return nil, fmt.Errorf("server name is required")
			}

			resolvedName := ""
			var serverCfg config.MCPServerConfig
			for name, candidate := range cfg.Tools.MCP.Servers {
				if strings.EqualFold(name, serverName) {
					resolvedName = name
					serverCfg = candidate
					break
				}
			}
			if resolvedName == "" {
				return nil, fmt.Errorf("MCP server '%s' is not configured", serverName)
			}
			if !serverCfg.Enabled {
				return nil, fmt.Errorf("MCP server '%s' is configured but disabled", resolvedName)
			}
			if !cfg.Tools.IsToolEnabled("mcp") {
				return nil, fmt.Errorf("MCP integration is disabled")
			}

			if err := al.ensureMCPInitialized(ctx); err != nil {
				logger.WarnCF("agent", "Failed to initialize MCP runtime for command",
					map[string]any{
						"server": resolvedName,
						"error":  err.Error(),
					})
			}

			manager := al.mcp.getManager()
			if manager == nil {
				return nil, fmt.Errorf("MCP server '%s' is configured but not connected", resolvedName)
			}

			conn, ok := manager.GetServer(resolvedName)
			if !ok {
				return nil, fmt.Errorf("MCP server '%s' is configured but not connected", resolvedName)
			}

			toolInfos := make([]commands.MCPToolInfo, 0, len(conn.Tools))
			for _, tool := range conn.Tools {
				if tool == nil {
					continue
				}
				name := strings.TrimSpace(tool.Name)
				if name == "" {
					continue
				}

				description := strings.TrimSpace(tool.Description)
				if description == "" {
					description = fmt.Sprintf("MCP tool from %s server", resolvedName)
				}

				toolInfos = append(toolInfos, commands.MCPToolInfo{
					Name:        name,
					Description: description,
					Parameters:  summarizeMCPToolParameters(tool.InputSchema),
				})
			}
			sort.Slice(toolInfos, func(i, j int) bool {
				return toolInfos[i].Name < toolInfos[j].Name
			})
			return toolInfos, nil
		},
		GetEnabledChannels: func() []string {
			if al.channelManager == nil {
				return nil
			}
			return al.channelManager.GetEnabledChannels()
		},
		GetActiveTurn: func() any {
			info := al.GetActiveTurn()
			if info == nil {
				return nil
			}
			return info
		},
		SwitchChannel: func(value string) error {
			if al.channelManager == nil {
				return fmt.Errorf("channel manager not initialized")
			}
			if _, exists := al.channelManager.GetChannel(value); !exists && value != "cli" {
				return fmt.Errorf("channel '%s' not found or not enabled", value)
			}
			return nil
		},
	}
	rt.StopActiveTurn = func() (commands.StopResult, error) {
		if opts == nil {
			return commands.StopResult{}, fmt.Errorf("process options not available")
		}
		return al.stopActiveTurnForSession(opts.Dispatch.SessionKey)
	}
	if agent != nil && agent.ContextBuilder != nil {
		rt.ListSkillNames = agent.ContextBuilder.ListSkillNames
	}
	rt.ReloadConfig = func() error {
		if al.reloadFunc == nil {
			return fmt.Errorf("reload not configured")
		}
		return al.reloadFunc()
	}
	if agent != nil {
		if agent.ContextBuilder != nil {
			rt.ListSkillNames = agent.ContextBuilder.ListSkillNames
		}
		rt.GetModelInfo = func() (string, string) {
			return agent.Model, resolvedCandidateProvider(agent.Candidates, cfg.Agents.Defaults.Provider)
		}
		rt.SwitchModel = func(value string) (string, error) {
			value = strings.TrimSpace(value)
			modelCfg, err := resolvedModelConfig(cfg, value, agent.Workspace)
			if err != nil {
				return "", err
			}

			nextProvider, _, err := providers.CreateProviderFromConfig(modelCfg)
			if err != nil {
				return "", fmt.Errorf("failed to initialize model %q: %w", value, err)
			}

			nextCandidates := resolveModelCandidates(cfg, cfg.Agents.Defaults.Provider, value, agent.Fallbacks)
			if len(nextCandidates) == 0 {
				return "", fmt.Errorf("model %q did not resolve to any provider candidates", value)
			}

			oldModel := agent.Model
			oldProvider := agent.Provider
			agent.Model = value
			agent.Provider = nextProvider
			agent.Candidates = nextCandidates
			agent.ThinkingLevel = parseThinkingLevel(modelCfg.ThinkingLevel)

			if oldProvider != nil && oldProvider != nextProvider {
				if stateful, ok := oldProvider.(providers.StatefulProvider); ok {
					stateful.Close()
				}
			}
			return oldModel, nil
		}

		rt.ClearHistory = func() error {
			if opts == nil {
				return fmt.Errorf("process options not available")
			}
			return al.contextManager.Clear(ctx, opts.SessionKey)
		}

		rt.ResetSession = func(clear bool) (string, error) {
			if opts == nil {
				return "", fmt.Errorf("process options not available")
			}
			routeSessionKey := strings.TrimSpace(opts.Dispatch.RouteSessionKey)
			if routeSessionKey == "" {
				return "", fmt.Errorf("route session key not available")
			}
			if clear {
				return "", al.clearSessionOverride(routeSessionKey)
			}

			nextSessionKey := buildResetSessionKey(agent.ID, routeSessionKey)
			if nextSessionKey == "" {
				return "", fmt.Errorf("failed to allocate reset session key")
			}
			if err := al.setSessionOverride(routeSessionKey, nextSessionKey); err != nil {
				return "", err
			}
			return nextSessionKey, nil
		}

		rt.AskSideQuestion = func(ctx context.Context, question string) (string, error) {
			return al.askSideQuestion(ctx, agent, opts, question)
		}

		rt.GetContextStats = func() *commands.ContextStats {
			if opts == nil || agent.Sessions == nil {
				return nil
			}
			usage := computeContextUsage(agent, opts.SessionKey)
			if usage == nil {
				return nil
			}
			history := agent.Sessions.GetHistory(opts.SessionKey)
			return &commands.ContextStats{
				UsedTokens:       usage.UsedTokens,
				TotalTokens:      usage.TotalTokens,
				CompressAtTokens: usage.CompressAtTokens,
				UsedPercent:      usage.UsedPercent,
				MessageCount:     len(history),
			}
		}
	}
	return rt
}

func summarizeMCPToolParameters(schema any) []commands.MCPToolParameterInfo {
	schemaMap := normalizeMCPSchema(schema)
	properties, ok := schemaMap["properties"].(map[string]any)
	if !ok || len(properties) == 0 {
		return nil
	}

	required := make(map[string]struct{})
	switch raw := schemaMap["required"].(type) {
	case []string:
		for _, name := range raw {
			required[name] = struct{}{}
		}
	case []any:
		for _, value := range raw {
			name, ok := value.(string)
			if ok {
				required[name] = struct{}{}
			}
		}
	}

	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)

	params := make([]commands.MCPToolParameterInfo, 0, len(names))
	for _, name := range names {
		param := commands.MCPToolParameterInfo{Name: name}
		if propMap, ok := properties[name].(map[string]any); ok {
			if typeName, ok := propMap["type"].(string); ok {
				param.Type = strings.TrimSpace(typeName)
			}
			if desc, ok := propMap["description"].(string); ok {
				param.Description = strings.TrimSpace(desc)
			}
		}
		_, param.Required = required[name]
		params = append(params, param)
	}
	return params
}

func normalizeMCPSchema(schema any) map[string]any {
	if schema == nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		}
	}

	if schemaMap, ok := schema.(map[string]any); ok {
		return schemaMap
	}

	var jsonData []byte
	switch raw := schema.(type) {
	case json.RawMessage:
		jsonData = raw
	case []byte:
		jsonData = raw
	}

	if jsonData == nil {
		var err error
		jsonData, err = json.Marshal(schema)
		if err != nil {
			return map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []string{},
			}
		}
	}

	var result map[string]any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		}
	}

	return result
}

func (al *AgentLoop) setPendingSkills(sessionKey string, skillNames []string) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" || len(skillNames) == 0 {
		return
	}

	filtered := make([]string, 0, len(skillNames))
	for _, name := range skillNames {
		name = strings.TrimSpace(name)
		if name != "" {
			filtered = append(filtered, name)
		}
	}
	if len(filtered) == 0 {
		return
	}

	al.pendingSkills.Store(sessionKey, filtered)
}

func (al *AgentLoop) takePendingSkills(sessionKey string) []string {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return nil
	}

	value, ok := al.pendingSkills.LoadAndDelete(sessionKey)
	if !ok {
		return nil
	}

	skills, ok := value.([]string)
	if !ok {
		return nil
	}

	return append([]string(nil), skills...)
}

func (al *AgentLoop) clearPendingSkills(sessionKey string) {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return
	}
	al.pendingSkills.Delete(sessionKey)
}
