package tools

import "github.com/sipeed/picoclaw/pkg/config"

// RegisterFeishuTools registers Feishu-related local tools that do not require
// a live channel instance or remote API injection.
func RegisterFeishuTools(registry *ToolRegistry, cfg *config.Config) {
	RegisterFeishuToolsWithClient(registry, cfg, nil)
}

// RegisterFeishuToolsWithClient registers local Feishu tools and, when provided,
// also registers remote query tools backed by an injected client.
func RegisterFeishuToolsWithClient(registry *ToolRegistry, cfg *config.Config, client FeishuRemoteClient) {
	if registry == nil || cfg == nil {
		return
	}
	if cfg.Tools.IsToolEnabled("feishu_parse") {
		registry.Register(NewFeishuParseTool())
	}
	if client != nil && cfg.Tools.IsToolEnabled("feishu_remote") {
		registry.Register(NewFeishuRemoteTool(client))
	}
}
