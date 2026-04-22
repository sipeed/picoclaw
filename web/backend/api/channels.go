package api

import (
	"encoding/json"
	"net/http"

	"github.com/sipeed/picoclaw/pkg/config"
)

type channelCatalogItem struct {
	Name      string `json:"name"`
	ConfigKey string `json:"config_key"`
	Variant   string `json:"variant,omitempty"`
}

var channelCatalog = []channelCatalogItem{
	{Name: "weixin", ConfigKey: "weixin"},
	{Name: "telegram", ConfigKey: "telegram"},
	{Name: "discord", ConfigKey: "discord"},
	{Name: "slack", ConfigKey: "slack"},
	{Name: "feishu", ConfigKey: "feishu"},
	{Name: "dingtalk", ConfigKey: "dingtalk"},
	{Name: "line", ConfigKey: "line"},
	{Name: "qq", ConfigKey: "qq"},
	{Name: "onebot", ConfigKey: "onebot"},
	{Name: "wecom", ConfigKey: "wecom"},
	{Name: "whatsapp", ConfigKey: "whatsapp", Variant: "bridge"},
	{Name: "whatsapp_native", ConfigKey: "whatsapp", Variant: "native"},
	{Name: "pico", ConfigKey: "pico"},
	{Name: "maixcam", ConfigKey: "maixcam"},
	{Name: "matrix", ConfigKey: "matrix"},
	{Name: "irc", ConfigKey: "irc"},
}

type channelConfigResponse struct {
	Config            any      `json:"config"`
	ConfiguredSecrets []string `json:"configured_secrets"`
	ConfigKey         string   `json:"config_key"`
	Variant           string   `json:"variant,omitempty"`
}

// registerChannelRoutes binds read-only channel catalog endpoints to the ServeMux.
func (h *Handler) registerChannelRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/channels/catalog", h.handleListChannelCatalog)
	mux.HandleFunc("GET /api/channels/{name}/config", h.handleGetChannelConfig)
}

// handleListChannelCatalog returns the channels supported by backend.
//
//	GET /api/channels/catalog
func (h *Handler) handleListChannelCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"channels": channelCatalog,
	})
}

// handleGetChannelConfig returns safe channel config plus secret presence metadata.
//
//	GET /api/channels/{name}/config
func (h *Handler) handleGetChannelConfig(w http.ResponseWriter, r *http.Request) {
	channelName := r.PathValue("name")
	item, ok := findChannelCatalogItem(channelName)
	if !ok {
		http.Error(w, "Channel not found", http.StatusNotFound)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}

	resp := buildChannelConfigResponse(cfg, item)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func findChannelCatalogItem(name string) (channelCatalogItem, bool) {
	for _, item := range channelCatalog {
		if item.Name == name {
			return item, true
		}
	}
	return channelCatalogItem{}, false
}

var channelSecretFieldMap = map[string][]string{
	"weixin":          {"token"},
	"telegram":        {"token"},
	"discord":         {"token"},
	"slack":           {"bot_token", "app_token"},
	"feishu":          {"app_secret", "encrypt_key", "verification_token"},
	"dingtalk":        {"client_secret"},
	"line":            {"channel_secret", "channel_access_token"},
	"qq":              {"app_secret"},
	"onebot":          {"access_token"},
	"wecom":           {"secret"},
	"pico":            {"token"},
	"matrix":          {"access_token"},
	"irc":             {"password", "nickserv_password", "sasl_password"},
	"whatsapp":        {},
	"whatsapp_native": {},
	"maixcam":         {},
}

func buildChannelConfigResponse(cfg *config.Config, item channelCatalogItem) channelConfigResponse {
	resp := channelConfigResponse{
		ConfiguredSecrets: []string{},
		ConfigKey:         item.ConfigKey,
		Variant:           item.Variant,
	}

	bc := cfg.Channels.Get(item.ConfigKey)
	if bc == nil {
		bc = defaultChannelConfig(item.ConfigKey)
		if bc == nil {
			resp.Config = map[string]any{}
			return resp
		}
	}

	// Detect configured secrets by checking the raw Settings JSON
	secrets := detectConfiguredSecrets(bc.Settings, item.Name)
	resp.ConfiguredSecrets = secrets

	// Parse settings into a generic map for JSON response
	settings := map[string]any{}
	if len(bc.Settings) > 0 {
		if err := json.Unmarshal(bc.Settings, &settings); err != nil {
			resp.Config = map[string]any{}
			return resp
		}
	}

	// Remove secure fields from response
	for _, key := range secrets {
		delete(settings, key)
	}
	addChannelCommonConfig(settings, bc)
	resp.Config = settings

	return resp
}

func defaultChannelConfig(configKey string) *config.Channel {
	return config.DefaultConfig().Channels.Get(configKey)
}

func addChannelCommonConfig(settings map[string]any, bc *config.Channel) {
	settings["enabled"] = bc.Enabled
	if len(bc.AllowFrom) > 0 {
		settings["allow_from"] = []string(bc.AllowFrom)
	}
	if bc.ReasoningChannelID != "" {
		settings["reasoning_channel_id"] = bc.ReasoningChannelID
	}
	if bc.GroupTrigger.MentionOnly || len(bc.GroupTrigger.Prefixes) > 0 {
		settings["group_trigger"] = bc.GroupTrigger
	}
	if bc.Typing.Enabled {
		settings["typing"] = bc.Typing
	}
	if bc.Placeholder.Enabled || len(bc.Placeholder.Text) > 0 {
		settings["placeholder"] = bc.Placeholder
	}
}

func detectConfiguredSecrets(settings config.RawNode, channelName string) []string {
	var m map[string]any
	if err := json.Unmarshal(settings, &m); err != nil {
		return nil
	}

	fields, ok := channelSecretFieldMap[channelName]
	if !ok {
		return nil
	}

	var found []string
	for _, key := range fields {
		if val, exists := m[key]; exists {
			switch v := val.(type) {
			case string:
				if v != "" {
					found = append(found, key)
				}
			case map[string]any:
				if s, ok := v["s"].(string); ok && s != "" {
					found = append(found, key)
				}
			}
		}
	}
	if found == nil {
		return []string{}
	}
	return found
}
