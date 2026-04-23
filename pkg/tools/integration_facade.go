package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sipeed/picoclaw/pkg/audio/tts"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/skills"
	integrationtools "github.com/sipeed/picoclaw/pkg/tools/integration"
)

type (
	SendCallbackWithContext  = integrationtools.SendCallbackWithContext
	ReactionCallback         = integrationtools.ReactionCallback
	MCPManager               = integrationtools.MCPManager
	MCPTool                  = integrationtools.MCPTool
	FindSkillsTool           = integrationtools.FindSkillsTool
	InstallSkillTool         = integrationtools.InstallSkillTool
	MessageTool              = integrationtools.MessageTool
	ReactionTool             = integrationtools.ReactionTool
	SendTTSTool              = integrationtools.SendTTSTool
	APIKeyPool               = integrationtools.APIKeyPool
	APIKeyIterator           = integrationtools.APIKeyIterator
	SearchProvider           = integrationtools.SearchProvider
	SearchResultItem         = integrationtools.SearchResultItem
	BraveSearchProvider      = integrationtools.BraveSearchProvider
	TavilySearchProvider     = integrationtools.TavilySearchProvider
	SogouSearchProvider      = integrationtools.SogouSearchProvider
	DuckDuckGoSearchProvider = integrationtools.DuckDuckGoSearchProvider
	PerplexitySearchProvider = integrationtools.PerplexitySearchProvider
	SearXNGSearchProvider    = integrationtools.SearXNGSearchProvider
	GLMSearchProvider        = integrationtools.GLMSearchProvider
	BaiduSearchProvider      = integrationtools.BaiduSearchProvider
	WebSearchTool            = integrationtools.WebSearchTool
	WebSearchToolOptions     = integrationtools.WebSearchToolOptions
	WebFetchTool             = integrationtools.WebFetchTool
)

func NewMCPTool(manager MCPManager, serverName string, tool *mcp.Tool) *MCPTool {
	return integrationtools.NewMCPTool(manager, serverName, tool)
}

func NewFindSkillsTool(registryMgr *skills.RegistryManager, cache *skills.SearchCache) *FindSkillsTool {
	return integrationtools.NewFindSkillsTool(registryMgr, cache)
}

func NewInstallSkillTool(registryMgr *skills.RegistryManager, workspace string) *InstallSkillTool {
	return integrationtools.NewInstallSkillTool(registryMgr, workspace)
}

func NewMessageTool() *MessageTool {
	return integrationtools.NewMessageTool()
}

func NewReactionTool() *ReactionTool {
	return integrationtools.NewReactionTool()
}

func NewSendTTSTool(provider tts.TTSProvider, store media.MediaStore) *SendTTSTool {
	return integrationtools.NewSendTTSTool(provider, store)
}

func NewAPIKeyPool(keys []string) *APIKeyPool {
	return integrationtools.NewAPIKeyPool(keys)
}

func SetPreferredWebSearchLanguage(lang string) {
	integrationtools.SetPreferredWebSearchLanguage(lang)
}

func GetPreferredWebSearchLanguage() string {
	return integrationtools.GetPreferredWebSearchLanguage()
}

func WebSearchToolOptionsFromConfig(cfg *config.Config) WebSearchToolOptions {
	return integrationtools.WebSearchToolOptionsFromConfig(cfg)
}

func WebSearchProviderReady(opts WebSearchToolOptions, name string) bool {
	return integrationtools.WebSearchProviderReady(opts, name)
}

func ResolveWebSearchProviderName(opts WebSearchToolOptions, query string) (string, error) {
	return integrationtools.ResolveWebSearchProviderName(opts, query)
}

func NewWebSearchTool(opts WebSearchToolOptions) (*WebSearchTool, error) {
	return integrationtools.NewWebSearchTool(opts)
}

func NewWebFetchTool(maxChars int, format string, fetchLimitBytes int64) (*WebFetchTool, error) {
	return integrationtools.NewWebFetchTool(maxChars, format, fetchLimitBytes)
}

func NewWebFetchToolWithProxy(
	maxChars int,
	proxy string,
	format string,
	fetchLimitBytes int64,
	privateHostWhitelist []string,
) (*WebFetchTool, error) {
	return integrationtools.NewWebFetchToolWithProxy(maxChars, proxy, format, fetchLimitBytes, privateHostWhitelist)
}

func NewWebFetchToolWithConfig(
	maxChars int,
	proxy string,
	format string,
	fetchLimitBytes int64,
	privateHostWhitelist []string,
) (*WebFetchTool, error) {
	return integrationtools.NewWebFetchToolWithConfig(maxChars, proxy, format, fetchLimitBytes, privateHostWhitelist)
}
