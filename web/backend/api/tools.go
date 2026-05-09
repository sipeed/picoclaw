package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"runtime"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	picotools "github.com/sipeed/picoclaw/pkg/tools"
)

type toolCatalogEntry struct {
	Name        string
	Description string
	Category    string
	ConfigKey   string
}

type toolSupportItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	ConfigKey   string `json:"config_key"`
	Status      string `json:"status"`
	ReasonCode  string `json:"reason_code,omitempty"`
}

type toolSupportResponse struct {
	Tools []toolSupportItem `json:"tools"`
}

type toolStateRequest struct {
	Enabled bool `json:"enabled"`
}

type webSearchProviderOption struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Configured   bool   `json:"configured"`
	Current      bool   `json:"current"`
	RequiresAuth bool   `json:"requires_auth"`
}

type webSearchProviderConfig struct {
	Enabled    bool     `json:"enabled"`
	MaxResults int      `json:"max_results"`
	BaseURL    string   `json:"base_url,omitempty"`
	APIKey     string   `json:"api_key,omitempty"`
	APIKeys    []string `json:"api_keys,omitempty"`
	APIKeySet  bool     `json:"api_key_set,omitempty"`
}

type webSearchConfigResponse struct {
	Provider       string                             `json:"provider"`
	CurrentService string                             `json:"current_service"`
	PreferNative   bool                               `json:"prefer_native"`
	Proxy          string                             `json:"proxy,omitempty"`
	Providers      []webSearchProviderOption          `json:"providers"`
	Settings       map[string]webSearchProviderConfig `json:"settings"`
}

type webSearchConfigRequest struct {
	Provider     string                             `json:"provider"`
	PreferNative bool                               `json:"prefer_native"`
	Proxy        string                             `json:"proxy"`
	Settings     map[string]webSearchProviderConfig `json:"settings"`
}

var toolCatalog = []toolCatalogEntry{
	{
		Name:        "read_file",
		Description: "Read file content from the workspace or explicitly allowed paths.",
		Category:    "filesystem",
		ConfigKey:   "read_file",
	},
	{
		Name:        "write_file",
		Description: "Create or overwrite files within the writable workspace scope.",
		Category:    "filesystem",
		ConfigKey:   "write_file",
	},
	{
		Name:        "list_dir",
		Description: "Inspect directories and enumerate files available to the agent.",
		Category:    "filesystem",
		ConfigKey:   "list_dir",
	},
	{
		Name:        "edit_file",
		Description: "Apply targeted edits to existing files without rewriting everything.",
		Category:    "filesystem",
		ConfigKey:   "edit_file",
	},
	{
		Name:        "append_file",
		Description: "Append content to the end of an existing file.",
		Category:    "filesystem",
		ConfigKey:   "append_file",
	},
	{
		Name:        "exec",
		Description: "Run shell commands inside the configured workspace sandbox.",
		Category:    "filesystem",
		ConfigKey:   "exec",
	},
	{
		Name:        "cron",
		Description: "Schedule one-time or recurring reminders, jobs, and shell commands.",
		Category:    "automation",
		ConfigKey:   "cron",
	},
	{
		Name:        "web_search",
		Description: "Search the web using the configured providers.",
		Category:    "web",
		ConfigKey:   "web",
	},
	{
		Name:        "web_fetch",
		Description: "Fetch and summarize the contents of a webpage.",
		Category:    "web",
		ConfigKey:   "web_fetch",
	},
	{
		Name:        "message",
		Description: "Send a follow-up message back to the active user or chat.",
		Category:    "communication",
		ConfigKey:   "message",
	},
	{
		Name:        "send_file",
		Description: "Send an outbound file or media attachment to the active chat.",
		Category:    "communication",
		ConfigKey:   "send_file",
	},
	{
		Name:        "find_skills",
		Description: "Search external skill registries for installable skills.",
		Category:    "skills",
		ConfigKey:   "find_skills",
	},
	{
		Name:        "install_skill",
		Description: "Install a skill into the current workspace from a registry.",
		Category:    "skills",
		ConfigKey:   "install_skill",
	},
	{
		Name:        "spawn",
		Description: "Launch a background subagent for long-running or delegated work.",
		Category:    "agents",
		ConfigKey:   "spawn",
	},
	{
		Name:        "spawn_status",
		Description: "Query the status of spawned subagents.",
		Category:    "agents",
		ConfigKey:   "spawn_status",
	},
	{
		Name:        "i2c",
		Description: "Interact with I2C hardware devices exposed on the host.",
		Category:    "hardware",
		ConfigKey:   "i2c",
	},
	{
		Name:        "spi",
		Description: "Interact with SPI hardware devices exposed on the host.",
		Category:    "hardware",
		ConfigKey:   "spi",
	},
	{
		Name:        "serial",
		Description: "Interact with serial ports exposed on the host.",
		Category:    "hardware",
		ConfigKey:   "serial",
	},
	{
		Name:        "tool_search_tool_regex",
		Description: "Discover hidden MCP tools by regex search when tool discovery is enabled.",
		Category:    "discovery",
		ConfigKey:   "mcp.discovery.use_regex",
	},
	{
		Name:        "tool_search_tool_bm25",
		Description: "Discover hidden MCP tools by semantic ranking when tool discovery is enabled.",
		Category:    "discovery",
		ConfigKey:   "mcp.discovery.use_bm25",
	},
	{
		Name:        "request_permission",
		Description: "Request user permission for accessing paths outside workspace.",
		Category:    "system",
		ConfigKey:   "request_permission",
	},
}

func (h *Handler) registerToolRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tools", h.handleListTools)
	mux.HandleFunc("PUT /api/tools/{name}/state", h.handleUpdateToolState)
	mux.HandleFunc("GET /api/tools/web-search-config", h.handleGetWebSearchConfig)
	mux.HandleFunc("PUT /api/tools/web-search-config", h.handleUpdateWebSearchConfig)
	mux.HandleFunc("POST /api/permission/grant", h.handleGrantPermission)
}

func (h *Handler) handleListTools(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toolSupportResponse{
		Tools: buildToolSupport(cfg),
	})
}

func (h *Handler) handleUpdateToolState(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	var req toolStateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if err := applyToolState(cfg, r.PathValue("name"), req.Enabled); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func buildToolSupport(cfg *config.Config) []toolSupportItem {
	items := make([]toolSupportItem, 0, len(toolCatalog))
	for _, entry := range toolCatalog {
		status := "disabled"
		reasonCode := ""

		switch entry.Name {
		case "find_skills", "install_skill":
			if cfg.Tools.IsToolEnabled(entry.ConfigKey) {
				if cfg.Tools.IsToolEnabled("skills") {
					status = "enabled"
				} else {
					status = "blocked"
					reasonCode = "requires_skills"
				}
			}
		case "spawn", "spawn_status":
			if cfg.Tools.IsToolEnabled(entry.ConfigKey) {
				if cfg.Tools.IsToolEnabled("subagent") {
					status = "enabled"
				} else {
					status = "blocked"
					reasonCode = "requires_subagent"
				}
			}
		case "tool_search_tool_regex":
			status, reasonCode = resolveDiscoveryToolSupport(cfg, cfg.Tools.MCP.Discovery.UseRegex)
		case "tool_search_tool_bm25":
			status, reasonCode = resolveDiscoveryToolSupport(cfg, cfg.Tools.MCP.Discovery.UseBM25)
		case "web_search":
			status, reasonCode = resolveWebSearchToolSupport(cfg)
		case "i2c", "spi":
			status, reasonCode = resolveHardwareToolSupport(cfg.Tools.IsToolEnabled(entry.ConfigKey))
		case "serial":
			status, reasonCode = resolveSerialToolSupport(cfg.Tools.IsToolEnabled(entry.ConfigKey))
		default:
			if cfg.Tools.IsToolEnabled(entry.ConfigKey) {
				status = "enabled"
			}
		}

		items = append(items, toolSupportItem{
			Name:        entry.Name,
			Description: entry.Description,
			Category:    entry.Category,
			ConfigKey:   entry.ConfigKey,
			Status:      status,
			ReasonCode:  reasonCode,
		})
	}
	return items
}

func resolveHardwareToolSupport(enabled bool) (string, string) {
	if !enabled {
		return "disabled", ""
	}
	if runtime.GOOS != "linux" {
		return "blocked", "requires_linux"
	}
	return "enabled", ""
}

func resolveSerialToolSupport(enabled bool) (string, string) {
	if !enabled {
		return "disabled", ""
	}
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
		return "enabled", ""
	default:
		return "blocked", "requires_serial_platform"
	}
}

func resolveDiscoveryToolSupport(cfg *config.Config, methodEnabled bool) (string, string) {
	if !cfg.Tools.IsToolEnabled("mcp") {
		return "disabled", ""
	}
	if !cfg.Tools.MCP.Discovery.Enabled {
		return "blocked", "requires_mcp_discovery"
	}
	if !methodEnabled {
		return "disabled", ""
	}
	return "enabled", ""
}

func resolveWebSearchToolSupport(cfg *config.Config) (string, string) {
	if !cfg.Tools.IsToolEnabled("web") {
		return "disabled", ""
	}
	return "enabled", ""
}

func applyToolState(cfg *config.Config, toolName string, enabled bool) error {
	switch toolName {
	case "read_file":
		cfg.Tools.ReadFile.Enabled = enabled
	case "write_file":
		cfg.Tools.WriteFile.Enabled = enabled
	case "list_dir":
		cfg.Tools.ListDir.Enabled = enabled
	case "edit_file":
		cfg.Tools.EditFile.Enabled = enabled
	case "append_file":
		cfg.Tools.AppendFile.Enabled = enabled
	case "exec":
		cfg.Tools.Exec.Enabled = enabled
	case "cron":
		cfg.Tools.Cron.Enabled = enabled
	case "web_search":
		cfg.Tools.Web.Enabled = enabled
	case "web_fetch":
		cfg.Tools.WebFetch.Enabled = enabled
	case "message":
		cfg.Tools.Message.Enabled = enabled
	case "send_file":
		cfg.Tools.SendFile.Enabled = enabled
	case "find_skills":
		cfg.Tools.FindSkills.Enabled = enabled
		if enabled {
			cfg.Tools.Skills.Enabled = true
		}
	case "install_skill":
		cfg.Tools.InstallSkill.Enabled = enabled
		if enabled {
			cfg.Tools.Skills.Enabled = true
		}
	case "spawn":
		cfg.Tools.Spawn.Enabled = enabled
		if enabled {
			cfg.Tools.Subagent.Enabled = true
		}
	case "spawn_status":
		cfg.Tools.SpawnStatus.Enabled = enabled
		if enabled {
			cfg.Tools.Spawn.Enabled = true
			cfg.Tools.Subagent.Enabled = true
		}
	case "i2c":
		cfg.Tools.I2C.Enabled = enabled
	case "spi":
		cfg.Tools.SPI.Enabled = enabled
	case "serial":
		cfg.Tools.Serial.Enabled = enabled
	case "tool_search_tool_regex":
		cfg.Tools.MCP.Discovery.UseRegex = enabled
		if enabled {
			cfg.Tools.MCP.Enabled = true
			cfg.Tools.MCP.Discovery.Enabled = true
		}
	case "tool_search_tool_bm25":
		cfg.Tools.MCP.Discovery.UseBM25 = enabled
		if enabled {
			cfg.Tools.MCP.Enabled = true
			cfg.Tools.MCP.Discovery.Enabled = true
		}
	default:
		return fmt.Errorf("tool %q cannot be updated", toolName)
	}
	return nil
}

func (h *Handler) handleGetWebSearchConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(buildWebSearchConfigResponse(cfg)); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *Handler) handleUpdateWebSearchConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	var req webSearchConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	provider := normalizeWebSearchProvider(req.Provider)
	if provider == "" {
		http.Error(w, "invalid web search provider", http.StatusBadRequest)
		return
	}

	cfg.Tools.Web.Provider = provider
	cfg.Tools.Web.PreferNative = req.PreferNative
	cfg.Tools.Web.Proxy = strings.TrimSpace(req.Proxy)

	if settings, ok := req.Settings["sogou"]; ok {
		cfg.Tools.Web.Sogou.Enabled = settings.Enabled
		cfg.Tools.Web.Sogou.MaxResults = settings.MaxResults
	}
	if settings, ok := req.Settings["duckduckgo"]; ok {
		cfg.Tools.Web.DuckDuckGo.Enabled = settings.Enabled
		cfg.Tools.Web.DuckDuckGo.MaxResults = settings.MaxResults
	}
	if settings, ok := req.Settings["brave"]; ok {
		cfg.Tools.Web.Brave.Enabled = settings.Enabled
		cfg.Tools.Web.Brave.MaxResults = settings.MaxResults
		if keys, ok := normalizeWebSearchAPIKeys(settings.APIKeys, settings.APIKey); ok {
			cfg.Tools.Web.Brave.SetAPIKeys(keys)
		}
	}
	if settings, ok := req.Settings["tavily"]; ok {
		cfg.Tools.Web.Tavily.Enabled = settings.Enabled
		cfg.Tools.Web.Tavily.MaxResults = settings.MaxResults
		cfg.Tools.Web.Tavily.BaseURL = strings.TrimSpace(settings.BaseURL)
		if keys, ok := normalizeWebSearchAPIKeys(settings.APIKeys, settings.APIKey); ok {
			cfg.Tools.Web.Tavily.SetAPIKeys(keys)
		}
	}
	if settings, ok := req.Settings["perplexity"]; ok {
		cfg.Tools.Web.Perplexity.Enabled = settings.Enabled
		cfg.Tools.Web.Perplexity.MaxResults = settings.MaxResults
		if keys, ok := normalizeWebSearchAPIKeys(settings.APIKeys, settings.APIKey); ok {
			cfg.Tools.Web.Perplexity.APIKeys = config.SimpleSecureStrings(keys...)
		}
	}
	if settings, ok := req.Settings["searxng"]; ok {
		cfg.Tools.Web.SearXNG.Enabled = settings.Enabled
		cfg.Tools.Web.SearXNG.MaxResults = settings.MaxResults
		cfg.Tools.Web.SearXNG.BaseURL = strings.TrimSpace(settings.BaseURL)
	}
	if settings, ok := req.Settings["glm_search"]; ok {
		cfg.Tools.Web.GLMSearch.Enabled = settings.Enabled
		cfg.Tools.Web.GLMSearch.MaxResults = settings.MaxResults
		cfg.Tools.Web.GLMSearch.BaseURL = strings.TrimSpace(settings.BaseURL)
		if key := strings.TrimSpace(settings.APIKey); key != "" {
			cfg.Tools.Web.GLMSearch.APIKey = *config.NewSecureString(key)
		}
	}
	if settings, ok := req.Settings["baidu_search"]; ok {
		cfg.Tools.Web.BaiduSearch.Enabled = settings.Enabled
		cfg.Tools.Web.BaiduSearch.MaxResults = settings.MaxResults
		cfg.Tools.Web.BaiduSearch.BaseURL = strings.TrimSpace(settings.BaseURL)
		if key := strings.TrimSpace(settings.APIKey); key != "" {
			cfg.Tools.Web.BaiduSearch.APIKey = *config.NewSecureString(key)
		}
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(buildWebSearchConfigResponse(cfg)); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func normalizeWebSearchProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "auto":
		return "auto"
	case "sogou", "brave", "tavily", "duckduckgo", "perplexity", "searxng", "glm_search", "baidu_search":
		return strings.ToLower(strings.TrimSpace(provider))
	default:
		return ""
	}
}

func normalizeWebSearchAPIKeys(apiKeys []string, apiKey string) ([]string, bool) {
	if apiKeys != nil {
		keys := make([]string, 0, len(apiKeys))
		seen := make(map[string]struct{}, len(apiKeys))
		for _, key := range apiKeys {
			trimmed := strings.TrimSpace(key)
			if trimmed == "" {
				continue
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			keys = append(keys, trimmed)
		}
		return keys, true
	}

	if trimmed := strings.TrimSpace(apiKey); trimmed != "" {
		return []string{trimmed}, true
	}

	return nil, false
}

func buildWebSearchConfigResponse(cfg *config.Config) webSearchConfigResponse {
	opts := picotools.WebSearchToolOptionsFromConfig(cfg)
	current := resolveCurrentWebSearchProvider(cfg)
	settings := map[string]webSearchProviderConfig{
		"sogou": {
			Enabled:    cfg.Tools.Web.Sogou.Enabled,
			MaxResults: cfg.Tools.Web.Sogou.MaxResults,
		},
		"duckduckgo": {
			Enabled:    cfg.Tools.Web.DuckDuckGo.Enabled,
			MaxResults: cfg.Tools.Web.DuckDuckGo.MaxResults,
		},
		"brave": {
			Enabled:    cfg.Tools.Web.Brave.Enabled,
			MaxResults: cfg.Tools.Web.Brave.MaxResults,
			APIKeySet:  len(cfg.Tools.Web.Brave.APIKeys.Values()) > 0,
		},
		"tavily": {
			Enabled:    cfg.Tools.Web.Tavily.Enabled,
			MaxResults: cfg.Tools.Web.Tavily.MaxResults,
			BaseURL:    cfg.Tools.Web.Tavily.BaseURL,
			APIKeySet:  len(cfg.Tools.Web.Tavily.APIKeys.Values()) > 0,
		},
		"perplexity": {
			Enabled:    cfg.Tools.Web.Perplexity.Enabled,
			MaxResults: cfg.Tools.Web.Perplexity.MaxResults,
			APIKeySet:  len(cfg.Tools.Web.Perplexity.APIKeys.Values()) > 0,
		},
		"searxng": {
			Enabled:    cfg.Tools.Web.SearXNG.Enabled,
			MaxResults: cfg.Tools.Web.SearXNG.MaxResults,
			BaseURL:    cfg.Tools.Web.SearXNG.BaseURL,
		},
		"glm_search": {
			Enabled:    cfg.Tools.Web.GLMSearch.Enabled,
			MaxResults: cfg.Tools.Web.GLMSearch.MaxResults,
			BaseURL:    cfg.Tools.Web.GLMSearch.BaseURL,
			APIKeySet:  cfg.Tools.Web.GLMSearch.APIKey.String() != "",
		},
		"baidu_search": {
			Enabled:    cfg.Tools.Web.BaiduSearch.Enabled,
			MaxResults: cfg.Tools.Web.BaiduSearch.MaxResults,
			BaseURL:    cfg.Tools.Web.BaiduSearch.BaseURL,
			APIKeySet:  cfg.Tools.Web.BaiduSearch.APIKey.String() != "",
		},
	}

	providers := []webSearchProviderOption{
		{
			ID:         "auto",
			Label:      "Auto",
			Configured: current != "",
			Current: cfg.Tools.Web.Provider == "" ||
				cfg.Tools.Web.Provider == "auto",
		},
		{
			ID:         "sogou",
			Label:      "Sogou",
			Configured: picotools.WebSearchProviderReady(opts, "sogou"),
			Current:    current == "sogou",
		},
		{
			ID:         "duckduckgo",
			Label:      "DuckDuckGo",
			Configured: picotools.WebSearchProviderReady(opts, "duckduckgo"),
			Current:    current == "duckduckgo",
		},
		{
			ID:           "brave",
			Label:        "Brave Search",
			Configured:   picotools.WebSearchProviderReady(opts, "brave"),
			Current:      current == "brave",
			RequiresAuth: true,
		},
		{
			ID:           "tavily",
			Label:        "Tavily",
			Configured:   picotools.WebSearchProviderReady(opts, "tavily"),
			Current:      current == "tavily",
			RequiresAuth: true,
		},
		{
			ID:           "perplexity",
			Label:        "Perplexity",
			Configured:   picotools.WebSearchProviderReady(opts, "perplexity"),
			Current:      current == "perplexity",
			RequiresAuth: true,
		},
		{
			ID:         "searxng",
			Label:      "SearXNG",
			Configured: picotools.WebSearchProviderReady(opts, "searxng"),
			Current:    current == "searxng",
		},
		{
			ID:           "glm_search",
			Label:        "GLM Search",
			Configured:   picotools.WebSearchProviderReady(opts, "glm_search"),
			Current:      current == "glm_search",
			RequiresAuth: true,
		},
		{
			ID:           "baidu_search",
			Label:        "Baidu Search",
			Configured:   picotools.WebSearchProviderReady(opts, "baidu_search"),
			Current:      current == "baidu_search",
			RequiresAuth: true,
		},
	}

	provider := cfg.Tools.Web.Provider
	if provider == "" {
		provider = "auto"
	}

	return webSearchConfigResponse{
		Provider:       provider,
		CurrentService: current,
		PreferNative:   cfg.Tools.Web.PreferNative,
		Proxy:          cfg.Tools.Web.Proxy,
		Providers:      providers,
		Settings:       settings,
	}
}

func resolveCurrentWebSearchProvider(cfg *config.Config) string {
	if cfg == nil || !cfg.Tools.IsToolEnabled("web") {
		return ""
	}
	selected, err := picotools.ResolveWebSearchProviderName(picotools.WebSearchToolOptionsFromConfig(cfg), "")
	if err != nil {
		return ""
	}
	return selected
}

// handleGrantPermission proxies permission grant requests to the gateway's internal endpoint.
// Frontend sends POST /api/permission/grant with { "path": "...", "duration": "once"|"session" }
// We add the default agent ID ("main") and forward to the gateway's /internal/permission/grant.
//
//	POST /api/permission/grant
func (h *Handler) handleGrantPermission(w http.ResponseWriter, r *http.Request) {
	if !h.gatewayAvailableForProxy() {
		logger.Warnf("Gateway not available for permission grant proxy")
		http.Error(w, "Gateway not available", http.StatusServiceUnavailable)
		return
	}

	// Parse incoming request body from frontend
	var req struct {
		Path     string `json:"path"`
		Duration string `json:"duration"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Path == "" || req.Duration == "" {
		http.Error(w, "path and duration are required", http.StatusBadRequest)
		return
	}
	if req.Duration != "once" && req.Duration != "session" {
		http.Error(w, "duration must be 'once' or 'session'", http.StatusBadRequest)
		return
	}

	// Get gateway auth token from pid data
	gateway.mu.Lock()
	pidData := gateway.pidData
	gateway.mu.Unlock()

	if pidData == nil || pidData.Token == "" {
		logger.Warnf("Gateway auth token not available for permission grant")
		http.Error(w, "Gateway auth token not available", http.StatusServiceUnavailable)
		return
	}
	authToken := pidData.Token

	// Prepare request body for gateway's internal endpoint (requires agent_id)
	grantReq := struct {
		AgentID  string `json:"agent_id"`
		Path     string `json:"path"`
		Duration string `json:"duration"`
	}{
		AgentID:  "main", // Default to main agent
		Path:     req.Path,
		Duration: req.Duration,
	}

	// Create reverse proxy to gateway's internal permission grant endpoint
	target := h.gatewayProxyURL()
	target.Path = "/internal/permission/grant"

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			// Inject auth token for gateway health server
			pr.Out.Header.Set("Authorization", "Bearer "+authToken)
			// Replace request body with the modified payload that includes agent_id
			bodyBytes, _ := json.Marshal(grantReq)
			pr.Out.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			pr.Out.ContentLength = int64(len(bodyBytes))
			pr.Out.Header.Set("Content-Type", "application/json")
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Errorf("Failed to proxy permission grant request: %v", err)
			http.Error(w, "Gateway unavailable: "+err.Error(), http.StatusBadGateway)
		},
	}

	proxy.ServeHTTP(w, r)
}
