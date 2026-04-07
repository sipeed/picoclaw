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

type channelSecretPresence struct {
	key        string
	configured bool
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

func buildChannelConfigResponse(cfg *config.Config, item channelCatalogItem) channelConfigResponse {
	resp := channelConfigResponse{
		ConfiguredSecrets: []string{},
		ConfigKey:         item.ConfigKey,
		Variant:           item.Variant,
	}

	switch item.Name {
	case "weixin":
		channelCfg := cfg.Channels.Weixin
		resp.ConfiguredSecrets = collectConfiguredSecrets(
			channelSecretPresence{key: "token", configured: channelCfg.Token.String() != ""},
		)
		channelCfg.Token = config.SecureString{}
		resp.Config = channelCfg
	case "telegram":
		channelCfg := cfg.Channels.Telegram
		resp.ConfiguredSecrets = collectConfiguredSecrets(
			channelSecretPresence{key: "token", configured: channelCfg.Token.String() != ""},
		)
		channelCfg.Token = config.SecureString{}
		resp.Config = channelCfg
	case "discord":
		channelCfg := cfg.Channels.Discord
		resp.ConfiguredSecrets = collectConfiguredSecrets(
			channelSecretPresence{key: "token", configured: channelCfg.Token.String() != ""},
		)
		channelCfg.Token = config.SecureString{}
		resp.Config = channelCfg
	case "slack":
		channelCfg := cfg.Channels.Slack
		resp.ConfiguredSecrets = collectConfiguredSecrets(
			channelSecretPresence{key: "bot_token", configured: channelCfg.BotToken.String() != ""},
			channelSecretPresence{key: "app_token", configured: channelCfg.AppToken.String() != ""},
		)
		channelCfg.BotToken = config.SecureString{}
		channelCfg.AppToken = config.SecureString{}
		resp.Config = channelCfg
	case "feishu":
		channelCfg := cfg.Channels.Feishu
		resp.ConfiguredSecrets = collectConfiguredSecrets(
			channelSecretPresence{key: "app_secret", configured: channelCfg.AppSecret.String() != ""},
			channelSecretPresence{key: "encrypt_key", configured: channelCfg.EncryptKey.String() != ""},
			channelSecretPresence{key: "verification_token", configured: channelCfg.VerificationToken.String() != ""},
		)
		channelCfg.AppSecret = config.SecureString{}
		channelCfg.EncryptKey = config.SecureString{}
		channelCfg.VerificationToken = config.SecureString{}
		resp.Config = channelCfg
	case "dingtalk":
		channelCfg := cfg.Channels.DingTalk
		resp.ConfiguredSecrets = collectConfiguredSecrets(
			channelSecretPresence{key: "client_secret", configured: channelCfg.ClientSecret.String() != ""},
		)
		channelCfg.ClientSecret = config.SecureString{}
		resp.Config = channelCfg
	case "line":
		channelCfg := cfg.Channels.LINE
		resp.ConfiguredSecrets = collectConfiguredSecrets(
			channelSecretPresence{key: "channel_secret", configured: channelCfg.ChannelSecret.String() != ""},
			channelSecretPresence{
				key:        "channel_access_token",
				configured: channelCfg.ChannelAccessToken.String() != "",
			},
		)
		channelCfg.ChannelSecret = config.SecureString{}
		channelCfg.ChannelAccessToken = config.SecureString{}
		resp.Config = channelCfg
	case "qq":
		channelCfg := cfg.Channels.QQ
		resp.ConfiguredSecrets = collectConfiguredSecrets(
			channelSecretPresence{key: "app_secret", configured: channelCfg.AppSecret.String() != ""},
		)
		channelCfg.AppSecret = config.SecureString{}
		resp.Config = channelCfg
	case "onebot":
		channelCfg := cfg.Channels.OneBot
		resp.ConfiguredSecrets = collectConfiguredSecrets(
			channelSecretPresence{key: "access_token", configured: channelCfg.AccessToken.String() != ""},
		)
		channelCfg.AccessToken = config.SecureString{}
		resp.Config = channelCfg
	case "wecom":
		channelCfg := cfg.Channels.WeCom
		resp.ConfiguredSecrets = collectConfiguredSecrets(
			channelSecretPresence{key: "secret", configured: channelCfg.Secret.String() != ""},
		)
		channelCfg.Secret = config.SecureString{}
		resp.Config = channelCfg
	case "whatsapp", "whatsapp_native":
		resp.Config = cfg.Channels.WhatsApp
	case "pico":
		channelCfg := cfg.Channels.Pico
		resp.ConfiguredSecrets = collectConfiguredSecrets(
			channelSecretPresence{key: "token", configured: channelCfg.Token.String() != ""},
		)
		channelCfg.Token = config.SecureString{}
		resp.Config = channelCfg
	case "maixcam":
		resp.Config = cfg.Channels.MaixCam
	case "matrix":
		channelCfg := cfg.Channels.Matrix
		resp.ConfiguredSecrets = collectConfiguredSecrets(
			channelSecretPresence{key: "access_token", configured: channelCfg.AccessToken.String() != ""},
		)
		channelCfg.AccessToken = config.SecureString{}
		resp.Config = channelCfg
	case "irc":
		channelCfg := cfg.Channels.IRC
		resp.ConfiguredSecrets = collectConfiguredSecrets(
			channelSecretPresence{key: "password", configured: channelCfg.Password.String() != ""},
			channelSecretPresence{key: "nickserv_password", configured: channelCfg.NickServPassword.String() != ""},
			channelSecretPresence{key: "sasl_password", configured: channelCfg.SASLPassword.String() != ""},
		)
		channelCfg.Password = config.SecureString{}
		channelCfg.NickServPassword = config.SecureString{}
		channelCfg.SASLPassword = config.SecureString{}
		resp.Config = channelCfg
	default:
		resp.Config = map[string]any{}
	}

	return resp
}

func collectConfiguredSecrets(secrets ...channelSecretPresence) []string {
	configured := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		if secret.configured {
			configured = append(configured, secret.key)
		}
	}
	return configured
}
