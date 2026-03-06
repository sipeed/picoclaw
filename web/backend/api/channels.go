package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// registerChannelRoutes binds channel management endpoints to the ServeMux.
func (h *Handler) registerChannelRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/channels", h.handleListChannels)
	mux.HandleFunc("PUT /api/channels/{name}", h.handleUpdateChannel)
	mux.HandleFunc("PATCH /api/channels/{name}/toggle", h.handleToggleChannel)
}

// channelMeta holds static metadata for each supported channel.
type channelMeta struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

var channelRegistry = []channelMeta{
	{Name: "telegram", DisplayName: "Telegram"},
	{Name: "discord", DisplayName: "Discord"},
	{Name: "slack", DisplayName: "Slack"},
	{Name: "feishu", DisplayName: "Feishu"},
	{Name: "dingtalk", DisplayName: "DingTalk"},
	{Name: "line", DisplayName: "LINE"},
	{Name: "qq", DisplayName: "QQ"},
	{Name: "onebot", DisplayName: "OneBot"},
	{Name: "wecom", DisplayName: "WeCom"},
	{Name: "wecom_app", DisplayName: "WeCom App"},
	{Name: "wecom_aibot", DisplayName: "WeCom AI Bot"},
	{Name: "whatsapp", DisplayName: "WhatsApp"},
	{Name: "pico", DisplayName: "Pico (Web)"},
	{Name: "maixcam", DisplayName: "MaixCAM"},
}

// channelResponse is the JSON structure returned for each channel in the list.
type channelResponse struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name"`
	Enabled     bool           `json:"enabled"`
	Configured  bool           `json:"configured"`
	Config      map[string]any `json:"config"`
}

// handleListChannels returns all channels with their enabled status and masked secrets.
//
//	GET /api/channels
func (h *Handler) handleListChannels(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.loadFilteredConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	channels := make([]channelResponse, 0, len(channelRegistry))
	for _, meta := range channelRegistry {
		cr := channelResponse{
			Name:        meta.Name,
			DisplayName: meta.DisplayName,
		}
		cr.Enabled, cr.Configured, cr.Config = extractChannelInfo(meta.Name, &cfg.Channels)
		channels = append(channels, cr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"channels": channels,
	})
}

// handleUpdateChannel replaces a channel's configuration.
// Secret fields sent as empty strings are preserved from the existing config.
//
//	PUT /api/channels/{name}
func (h *Handler) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !isValidChannel(name) {
		http.Error(w, fmt.Sprintf("Unknown channel: %s", name), http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var incoming map[string]any
	if err = json.Unmarshal(body, &incoming); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	applyChannelUpdate(name, &cfg.Channels, incoming)
	if errs := validateConfig(cfg); len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleToggleChannel enables or disables a channel.
//
//	PATCH /api/channels/{name}/toggle
func (h *Handler) handleToggleChannel(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !isValidChannel(name) {
		http.Error(w, fmt.Sprintf("Unknown channel: %s", name), http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err = json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	if req.Enabled == nil {
		http.Error(w, "Missing required field: enabled", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	setChannelEnabled(name, &cfg.Channels, *req.Enabled)
	if errs := validateConfig(cfg); len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func isValidChannel(name string) bool {
	for _, m := range channelRegistry {
		if m.Name == name {
			return true
		}
	}
	return false
}

func writeValidationErrors(w http.ResponseWriter, errs []string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]any{
		"status": "validation_error",
		"errors": errs,
	})
}

// extractChannelInfo returns enabled, configured status and masked config for a channel.
func extractChannelInfo(name string, ch *config.ChannelsConfig) (bool, bool, map[string]any) {
	var enabled, configured bool
	cfg := make(map[string]any)
	switch name {
	case "telegram":
		c := ch.Telegram
		enabled = c.Enabled
		configured = c.Token != ""
		cfg["token"] = maskAPIKey(c.Token)
		cfg["base_url"] = c.BaseURL
		cfg["proxy"] = c.Proxy
		cfg["allow_from"] = []string(c.AllowFrom)
		cfg["group_trigger"] = c.GroupTrigger
		cfg["typing"] = c.Typing
		cfg["placeholder"] = c.Placeholder
	case "discord":
		c := ch.Discord
		enabled = c.Enabled
		configured = c.Token != ""
		cfg["token"] = maskAPIKey(c.Token)
		cfg["proxy"] = c.Proxy
		cfg["allow_from"] = []string(c.AllowFrom)
		cfg["group_trigger"] = c.GroupTrigger
		cfg["typing"] = c.Typing
		cfg["placeholder"] = c.Placeholder
	case "slack":
		c := ch.Slack
		enabled = c.Enabled
		configured = c.BotToken != ""
		cfg["bot_token"] = maskAPIKey(c.BotToken)
		cfg["app_token"] = maskAPIKey(c.AppToken)
		cfg["allow_from"] = []string(c.AllowFrom)
		cfg["group_trigger"] = c.GroupTrigger
		cfg["typing"] = c.Typing
		cfg["placeholder"] = c.Placeholder
	case "feishu":
		c := ch.Feishu
		enabled = c.Enabled
		configured = c.AppID != "" && c.AppSecret != ""
		cfg["app_id"] = c.AppID
		cfg["app_secret"] = maskAPIKey(c.AppSecret)
		cfg["encrypt_key"] = maskAPIKey(c.EncryptKey)
		cfg["verification_token"] = maskAPIKey(c.VerificationToken)
		cfg["allow_from"] = []string(c.AllowFrom)
		cfg["group_trigger"] = c.GroupTrigger
		cfg["placeholder"] = c.Placeholder
	case "dingtalk":
		c := ch.DingTalk
		enabled = c.Enabled
		configured = c.ClientID != "" && c.ClientSecret != ""
		cfg["client_id"] = c.ClientID
		cfg["client_secret"] = maskAPIKey(c.ClientSecret)
		cfg["allow_from"] = []string(c.AllowFrom)
		cfg["group_trigger"] = c.GroupTrigger
	case "line":
		c := ch.LINE
		enabled = c.Enabled
		configured = c.ChannelSecret != "" && c.ChannelAccessToken != ""
		cfg["channel_secret"] = maskAPIKey(c.ChannelSecret)
		cfg["channel_access_token"] = maskAPIKey(c.ChannelAccessToken)
		cfg["webhook_host"] = c.WebhookHost
		cfg["webhook_port"] = c.WebhookPort
		cfg["webhook_path"] = c.WebhookPath
		cfg["allow_from"] = []string(c.AllowFrom)
		cfg["group_trigger"] = c.GroupTrigger
		cfg["typing"] = c.Typing
		cfg["placeholder"] = c.Placeholder
	case "qq":
		c := ch.QQ
		enabled = c.Enabled
		configured = c.AppID != "" && c.AppSecret != ""
		cfg["app_id"] = c.AppID
		cfg["app_secret"] = maskAPIKey(c.AppSecret)
		cfg["allow_from"] = []string(c.AllowFrom)
		cfg["group_trigger"] = c.GroupTrigger
	case "onebot":
		c := ch.OneBot
		enabled = c.Enabled
		configured = c.WSUrl != ""
		cfg["ws_url"] = c.WSUrl
		cfg["access_token"] = maskAPIKey(c.AccessToken)
		cfg["reconnect_interval"] = c.ReconnectInterval
		cfg["allow_from"] = []string(c.AllowFrom)
		cfg["group_trigger"] = c.GroupTrigger
		cfg["typing"] = c.Typing
		cfg["placeholder"] = c.Placeholder
	case "wecom":
		c := ch.WeCom
		enabled = c.Enabled
		configured = c.Token != ""
		cfg["token"] = maskAPIKey(c.Token)
		cfg["encoding_aes_key"] = maskAPIKey(c.EncodingAESKey)
		cfg["webhook_url"] = c.WebhookURL
		cfg["webhook_host"] = c.WebhookHost
		cfg["webhook_port"] = c.WebhookPort
		cfg["webhook_path"] = c.WebhookPath
		cfg["allow_from"] = []string(c.AllowFrom)
		cfg["reply_timeout"] = c.ReplyTimeout
		cfg["group_trigger"] = c.GroupTrigger
	case "wecom_app":
		c := ch.WeComApp
		enabled = c.Enabled
		configured = c.CorpID != "" && c.CorpSecret != ""
		cfg["corp_id"] = c.CorpID
		cfg["corp_secret"] = maskAPIKey(c.CorpSecret)
		cfg["agent_id"] = c.AgentID
		cfg["token"] = maskAPIKey(c.Token)
		cfg["encoding_aes_key"] = maskAPIKey(c.EncodingAESKey)
		cfg["webhook_host"] = c.WebhookHost
		cfg["webhook_port"] = c.WebhookPort
		cfg["webhook_path"] = c.WebhookPath
		cfg["allow_from"] = []string(c.AllowFrom)
		cfg["reply_timeout"] = c.ReplyTimeout
		cfg["group_trigger"] = c.GroupTrigger
	case "wecom_aibot":
		c := ch.WeComAIBot
		enabled = c.Enabled
		configured = c.Token != ""
		cfg["token"] = maskAPIKey(c.Token)
		cfg["encoding_aes_key"] = maskAPIKey(c.EncodingAESKey)
		cfg["webhook_path"] = c.WebhookPath
		cfg["allow_from"] = []string(c.AllowFrom)
		cfg["reply_timeout"] = c.ReplyTimeout
		cfg["max_steps"] = c.MaxSteps
		cfg["welcome_message"] = c.WelcomeMessage
	case "whatsapp":
		c := ch.WhatsApp
		enabled = c.Enabled
		configured = c.BridgeURL != "" || c.UseNative
		cfg["bridge_url"] = c.BridgeURL
		cfg["use_native"] = c.UseNative
		cfg["session_store_path"] = c.SessionStorePath
		cfg["allow_from"] = []string(c.AllowFrom)
	case "pico":
		c := ch.Pico
		enabled = c.Enabled
		configured = true // Always considered configured (built-in WebSocket channel)
		cfg["token"] = maskAPIKey(c.Token)
		cfg["allow_token_query"] = c.AllowTokenQuery
		cfg["allow_origins"] = c.AllowOrigins
		cfg["ping_interval"] = c.PingInterval
		cfg["read_timeout"] = c.ReadTimeout
		cfg["write_timeout"] = c.WriteTimeout
		cfg["max_connections"] = c.MaxConnections
		cfg["allow_from"] = []string(c.AllowFrom)
		cfg["placeholder"] = c.Placeholder
	case "maixcam":
		c := ch.MaixCam
		enabled = c.Enabled
		configured = c.Host != ""
		cfg["host"] = c.Host
		cfg["port"] = c.Port
		cfg["allow_from"] = []string(c.AllowFrom)
	}
	return enabled, configured, cfg
}

// setChannelEnabled sets the enabled flag for a channel.
func setChannelEnabled(name string, ch *config.ChannelsConfig, enabled bool) {
	switch name {
	case "telegram":
		ch.Telegram.Enabled = enabled
	case "discord":
		ch.Discord.Enabled = enabled
	case "slack":
		ch.Slack.Enabled = enabled
	case "feishu":
		ch.Feishu.Enabled = enabled
	case "dingtalk":
		ch.DingTalk.Enabled = enabled
	case "line":
		ch.LINE.Enabled = enabled
	case "qq":
		ch.QQ.Enabled = enabled
	case "onebot":
		ch.OneBot.Enabled = enabled
	case "wecom":
		ch.WeCom.Enabled = enabled
	case "wecom_app":
		ch.WeComApp.Enabled = enabled
	case "wecom_aibot":
		ch.WeComAIBot.Enabled = enabled
	case "whatsapp":
		ch.WhatsApp.Enabled = enabled
	case "pico":
		ch.Pico.Enabled = enabled
	case "maixcam":
		ch.MaixCam.Enabled = enabled
	}
}

// applyChannelUpdate applies incoming config fields to the corresponding channel.
// Empty secret fields are preserved from the existing config.
func applyChannelUpdate(name string, ch *config.ChannelsConfig, incoming map[string]any) {
	getString := func(key string) string {
		if v, ok := incoming[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	getBool := func(key string) bool {
		if v, ok := incoming[key]; ok {
			if b, ok := v.(bool); ok {
				return b
			}
		}
		return false
	}
	getInt := func(key string) int {
		if v, ok := incoming[key]; ok {
			if f, ok := v.(float64); ok {
				return int(f)
			}
		}
		return 0
	}
	getInt64 := func(key string) int64 {
		if v, ok := incoming[key]; ok {
			if f, ok := v.(float64); ok {
				return int64(f)
			}
		}
		return 0
	}
	getStringSlice := func(key string) config.FlexibleStringSlice {
		if v, ok := incoming[key]; ok {
			if arr, ok := v.([]any); ok {
				result := make(config.FlexibleStringSlice, 0, len(arr))
				for _, item := range arr {
					if s, ok := item.(string); ok {
						result = append(result, s)
					}
				}
				return result
			}
		}
		return nil
	}
	getStringArray := func(key string) ([]string, bool) {
		v, ok := incoming[key]
		if !ok {
			return nil, false
		}

		switch arr := v.(type) {
		case []any:
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						result = append(result, s)
					}
				}
			}
			return result, true
		case []string:
			result := make([]string, 0, len(arr))
			for _, s := range arr {
				s = strings.TrimSpace(s)
				if s != "" {
					result = append(result, s)
				}
			}
			return result, true
		case string:
			if strings.TrimSpace(arr) == "" {
				return []string{}, true
			}
			parts := strings.Split(arr, ",")
			result := make([]string, 0, len(parts))
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					result = append(result, part)
				}
			}
			return result, true
		default:
			return nil, false
		}
	}
	getGroupTrigger := func() config.GroupTriggerConfig {
		if v, ok := incoming["group_trigger"]; ok {
			if m, ok := v.(map[string]any); ok {
				gt := config.GroupTriggerConfig{}
				if b, ok := m["mention_only"].(bool); ok {
					gt.MentionOnly = b
				}
				if arr, ok := m["prefixes"].([]any); ok {
					for _, item := range arr {
						if s, ok := item.(string); ok {
							gt.Prefixes = append(gt.Prefixes, s)
						}
					}
				}
				return gt
			}
		}
		return config.GroupTriggerConfig{}
	}
	getTyping := func() config.TypingConfig {
		if v, ok := incoming["typing"]; ok {
			if m, ok := v.(map[string]any); ok {
				if b, ok := m["enabled"].(bool); ok {
					return config.TypingConfig{Enabled: b}
				}
			}
		}
		return config.TypingConfig{}
	}
	getPlaceholder := func() config.PlaceholderConfig {
		if v, ok := incoming["placeholder"]; ok {
			if m, ok := v.(map[string]any); ok {
				pc := config.PlaceholderConfig{}
				if b, ok := m["enabled"].(bool); ok {
					pc.Enabled = b
				}
				if s, ok := m["text"].(string); ok {
					pc.Text = s
				}
				return pc
			}
		}
		return config.PlaceholderConfig{}
	}

	// preserveSecret returns the incoming value if non-empty, or keeps existing.
	preserveSecret := func(incoming, existing string) string {
		if incoming == "" {
			return existing
		}
		return incoming
	}

	switch name {
	case "telegram":
		c := &ch.Telegram
		c.Enabled = getBool("enabled")
		c.Token = preserveSecret(getString("token"), c.Token)
		c.BaseURL = getString("base_url")
		c.Proxy = getString("proxy")
		c.AllowFrom = getStringSlice("allow_from")
		c.GroupTrigger = getGroupTrigger()
		c.Typing = getTyping()
		c.Placeholder = getPlaceholder()
	case "discord":
		c := &ch.Discord
		c.Enabled = getBool("enabled")
		c.Token = preserveSecret(getString("token"), c.Token)
		c.Proxy = getString("proxy")
		c.AllowFrom = getStringSlice("allow_from")
		c.GroupTrigger = getGroupTrigger()
		c.Typing = getTyping()
		c.Placeholder = getPlaceholder()
	case "slack":
		c := &ch.Slack
		c.Enabled = getBool("enabled")
		c.BotToken = preserveSecret(getString("bot_token"), c.BotToken)
		c.AppToken = preserveSecret(getString("app_token"), c.AppToken)
		c.AllowFrom = getStringSlice("allow_from")
		c.GroupTrigger = getGroupTrigger()
		c.Typing = getTyping()
		c.Placeholder = getPlaceholder()
	case "feishu":
		c := &ch.Feishu
		c.Enabled = getBool("enabled")
		c.AppID = getString("app_id")
		c.AppSecret = preserveSecret(getString("app_secret"), c.AppSecret)
		c.EncryptKey = preserveSecret(getString("encrypt_key"), c.EncryptKey)
		c.VerificationToken = preserveSecret(getString("verification_token"), c.VerificationToken)
		c.AllowFrom = getStringSlice("allow_from")
		c.GroupTrigger = getGroupTrigger()
		c.Placeholder = getPlaceholder()
	case "dingtalk":
		c := &ch.DingTalk
		c.Enabled = getBool("enabled")
		c.ClientID = getString("client_id")
		c.ClientSecret = preserveSecret(getString("client_secret"), c.ClientSecret)
		c.AllowFrom = getStringSlice("allow_from")
		c.GroupTrigger = getGroupTrigger()
	case "line":
		c := &ch.LINE
		c.Enabled = getBool("enabled")
		c.ChannelSecret = preserveSecret(getString("channel_secret"), c.ChannelSecret)
		c.ChannelAccessToken = preserveSecret(getString("channel_access_token"), c.ChannelAccessToken)
		c.WebhookHost = getString("webhook_host")
		c.WebhookPort = getInt("webhook_port")
		c.WebhookPath = getString("webhook_path")
		c.AllowFrom = getStringSlice("allow_from")
		c.GroupTrigger = getGroupTrigger()
		c.Typing = getTyping()
		c.Placeholder = getPlaceholder()
	case "qq":
		c := &ch.QQ
		c.Enabled = getBool("enabled")
		c.AppID = getString("app_id")
		c.AppSecret = preserveSecret(getString("app_secret"), c.AppSecret)
		c.AllowFrom = getStringSlice("allow_from")
		c.GroupTrigger = getGroupTrigger()
	case "onebot":
		c := &ch.OneBot
		c.Enabled = getBool("enabled")
		c.WSUrl = getString("ws_url")
		c.AccessToken = preserveSecret(getString("access_token"), c.AccessToken)
		c.ReconnectInterval = getInt("reconnect_interval")
		c.AllowFrom = getStringSlice("allow_from")
		c.GroupTrigger = getGroupTrigger()
		c.Typing = getTyping()
		c.Placeholder = getPlaceholder()
	case "wecom":
		c := &ch.WeCom
		c.Enabled = getBool("enabled")
		c.Token = preserveSecret(getString("token"), c.Token)
		c.EncodingAESKey = preserveSecret(getString("encoding_aes_key"), c.EncodingAESKey)
		c.WebhookURL = getString("webhook_url")
		c.WebhookHost = getString("webhook_host")
		c.WebhookPort = getInt("webhook_port")
		c.WebhookPath = getString("webhook_path")
		c.AllowFrom = getStringSlice("allow_from")
		c.ReplyTimeout = getInt("reply_timeout")
		c.GroupTrigger = getGroupTrigger()
	case "wecom_app":
		c := &ch.WeComApp
		c.Enabled = getBool("enabled")
		c.CorpID = getString("corp_id")
		c.CorpSecret = preserveSecret(getString("corp_secret"), c.CorpSecret)
		c.AgentID = getInt64("agent_id")
		c.Token = preserveSecret(getString("token"), c.Token)
		c.EncodingAESKey = preserveSecret(getString("encoding_aes_key"), c.EncodingAESKey)
		c.WebhookHost = getString("webhook_host")
		c.WebhookPort = getInt("webhook_port")
		c.WebhookPath = getString("webhook_path")
		c.AllowFrom = getStringSlice("allow_from")
		c.ReplyTimeout = getInt("reply_timeout")
		c.GroupTrigger = getGroupTrigger()
	case "wecom_aibot":
		c := &ch.WeComAIBot
		c.Enabled = getBool("enabled")
		c.Token = preserveSecret(getString("token"), c.Token)
		c.EncodingAESKey = preserveSecret(getString("encoding_aes_key"), c.EncodingAESKey)
		c.WebhookPath = getString("webhook_path")
		c.AllowFrom = getStringSlice("allow_from")
		c.ReplyTimeout = getInt("reply_timeout")
		c.MaxSteps = getInt("max_steps")
		c.WelcomeMessage = getString("welcome_message")
	case "whatsapp":
		c := &ch.WhatsApp
		c.Enabled = getBool("enabled")
		c.BridgeURL = getString("bridge_url")
		c.UseNative = getBool("use_native")
		c.SessionStorePath = getString("session_store_path")
		c.AllowFrom = getStringSlice("allow_from")
	case "pico":
		c := &ch.Pico
		c.Enabled = getBool("enabled")
		c.Token = preserveSecret(getString("token"), c.Token)
		c.AllowTokenQuery = getBool("allow_token_query")
		if origins, ok := getStringArray("allow_origins"); ok {
			c.AllowOrigins = origins
		}
		c.PingInterval = getInt("ping_interval")
		c.ReadTimeout = getInt("read_timeout")
		c.WriteTimeout = getInt("write_timeout")
		c.MaxConnections = getInt("max_connections")
		c.AllowFrom = getStringSlice("allow_from")
		c.Placeholder = getPlaceholder()
	case "maixcam":
		c := &ch.MaixCam
		c.Enabled = getBool("enabled")
		c.Host = getString("host")
		c.Port = getInt("port")
		c.AllowFrom = getStringSlice("allow_from")
	}
}
