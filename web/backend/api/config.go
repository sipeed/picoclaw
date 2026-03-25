package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// registerConfigRoutes binds configuration management endpoints to the ServeMux.
func (h *Handler) registerConfigRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/config", h.handleGetConfig)
	mux.HandleFunc("PUT /api/config", h.handleUpdateConfig)
	mux.HandleFunc("PATCH /api/config", h.handlePatchConfig)
	mux.HandleFunc("POST /api/config/test-command-patterns", h.handleTestCommandPatterns)
}

// handleGetConfig returns the complete system configuration.
//
//	GET /api/config
func (h *Handler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cfg); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleUpdateConfig updates the complete system configuration.
//
//	PUT /api/config
func (h *Handler) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var cfg config.Config
	if err = json.Unmarshal(body, &cfg); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	var raw map[string]any
	if err = json.Unmarshal(body, &raw); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	if execAllowRemoteOmitted(body) {
		cfg.Tools.Exec.AllowRemote = config.DefaultConfig().Tools.Exec.AllowRemote
	}

	// Load existing config and copy security credentials before validation,
	// so that security-managed fields (e.g. pico token) are available.
	err = cfg.SecurityCopyFrom(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to apply security config: %v", err), http.StatusInternalServerError)
		return
	}
	applyConfigSecretsFromMap(&cfg, raw)

	if errs := validateConfig(&cfg); len(errs) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "validation_error",
			"errors": errs,
		})
		return
	}

	logger.Infof("configuration updated successfully")

	if err := config.SaveConfig(h.configPath, &cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func execAllowRemoteOmitted(body []byte) bool {
	var raw struct {
		Tools *struct {
			Exec *struct {
				AllowRemote *bool `json:"allow_remote"`
			} `json:"exec"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return false
	}
	return raw.Tools == nil || raw.Tools.Exec == nil || raw.Tools.Exec.AllowRemote == nil
}

// handlePatchConfig partially updates the system configuration using JSON Merge Patch (RFC 7396).
// Only the fields present in the request body will be updated; all other fields remain unchanged.
//
//	PATCH /api/config
func (h *Handler) handlePatchConfig(w http.ResponseWriter, r *http.Request) {
	patchBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate the patch is valid JSON
	var patch map[string]any
	if err = json.Unmarshal(patchBody, &patch); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Load existing config and marshal to a map for merging
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	existing, err := json.Marshal(cfg)
	if err != nil {
		http.Error(w, "Failed to serialize current config", http.StatusInternalServerError)
		return
	}

	var base map[string]any
	if err = json.Unmarshal(existing, &base); err != nil {
		http.Error(w, "Failed to parse current config", http.StatusInternalServerError)
		return
	}

	// Recursively merge patch into base
	mergeMap(base, patch)

	// Convert merged map back to Config struct
	merged, err := json.Marshal(base)
	if err != nil {
		http.Error(w, "Failed to serialize merged config", http.StatusInternalServerError)
		return
	}

	var newCfg config.Config
	if err = json.Unmarshal(merged, &newCfg); err != nil {
		http.Error(w, fmt.Sprintf("Merged config is invalid: %v", err), http.StatusBadRequest)
		return
	}

	// Restore security fields (tokens/keys) from the loaded config before validation,
	// because private fields are lost during JSON round-trip.
	if err = newCfg.SecurityCopyFrom(h.configPath); err != nil {
		http.Error(w, fmt.Sprintf("Failed to apply security config: %v", err), http.StatusInternalServerError)
		return
	}
	applyConfigSecretsFromMap(&newCfg, base)

	if errs := validateConfig(&newCfg); len(errs) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "validation_error",
			"errors": errs,
		})
		return
	}

	if err := config.SaveConfig(h.configPath, &newCfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleTestCommandPatterns tests a command against whitelist and blacklist patterns.
//
//	POST /api/config/test-command-patterns
func (h *Handler) handleTestCommandPatterns(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		AllowPatterns []string `json:"allow_patterns"`
		DenyPatterns  []string `json:"deny_patterns"`
		Command       string   `json:"command"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	lower := strings.ToLower(strings.TrimSpace(req.Command))

	type result struct {
		Allowed          bool    `json:"allowed"`
		Blocked          bool    `json:"blocked"`
		MatchedWhitelist *string `json:"matched_whitelist,omitempty"`
		MatchedBlacklist *string `json:"matched_blacklist,omitempty"`
	}

	resp := result{Allowed: false, Blocked: false}

	// Check whitelist first
	for _, pattern := range req.AllowPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue // skip invalid patterns
		}
		if re.MatchString(lower) {
			resp.Allowed = true
			resp.MatchedWhitelist = &pattern
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
	}

	// Check blacklist
	for _, pattern := range req.DenyPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(lower) {
			resp.Blocked = true
			resp.MatchedBlacklist = &pattern
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// validateConfig checks the config for common errors before saving.
// Returns a list of human-readable error strings; empty means valid.
func validateConfig(cfg *config.Config) []string {
	var errs []string

	// Validate model_list entries
	if err := cfg.ValidateModelList(); err != nil {
		errs = append(errs, err.Error())
	}

	// Gateway port range
	if cfg.Gateway.Port != 0 && (cfg.Gateway.Port < 1 || cfg.Gateway.Port > 65535) {
		errs = append(errs, fmt.Sprintf("gateway.port %d is out of valid range (1-65535)", cfg.Gateway.Port))
	}

	// Pico channel: token required when enabled
	if cfg.Channels.Pico.Enabled && cfg.Channels.Pico.Token.String() == "" {
		errs = append(errs, "channels.pico.token is required when pico channel is enabled")
	}

	// Telegram: token required when enabled
	if cfg.Channels.Telegram.Enabled && cfg.Channels.Telegram.Token.String() == "" {
		errs = append(errs, "channels.telegram.token is required when telegram channel is enabled")
	}

	// Discord: token required when enabled
	if cfg.Channels.Discord.Enabled && cfg.Channels.Discord.Token.String() == "" {
		errs = append(errs, "channels.discord.token is required when discord channel is enabled")
	}

	if cfg.Channels.WeCom.Enabled {
		if cfg.Channels.WeCom.BotID == "" {
			errs = append(errs, "channels.wecom.bot_id is required when wecom channel is enabled")
		}
		if cfg.Channels.WeCom.Secret.String() == "" {
			errs = append(errs, "channels.wecom.secret is required when wecom channel is enabled")
		}
	}

	if cfg.Tools.Exec.Enabled {
		if cfg.Tools.Exec.EnableDenyPatterns {
			errs = append(
				errs,
				validateRegexPatterns("tools.exec.custom_deny_patterns", cfg.Tools.Exec.CustomDenyPatterns)...)
		}
		errs = append(
			errs,
			validateRegexPatterns("tools.exec.custom_allow_patterns", cfg.Tools.Exec.CustomAllowPatterns)...)
	}

	return errs
}

func validateRegexPatterns(field string, patterns []string) []string {
	var errs []string
	for index, pattern := range patterns {
		if _, err := regexp.Compile(pattern); err != nil {
			errs = append(errs, fmt.Sprintf("%s[%d] is not a valid regular expression: %v", field, index, err))
		}
	}
	return errs
}

// mergeMap recursively merges src into dst (JSON Merge Patch semantics).
// - If a key in src has a null value, it is deleted from dst.
// - If both dst and src have a nested object for the same key, merge recursively.
// - Otherwise the value from src overwrites dst.
func mergeMap(dst, src map[string]any) {
	for key, srcVal := range src {
		if srcVal == nil {
			delete(dst, key)
			continue
		}
		srcMap, srcIsMap := srcVal.(map[string]any)
		dstMap, dstIsMap := dst[key].(map[string]any)
		if srcIsMap && dstIsMap {
			mergeMap(dstMap, srcMap)
		} else {
			dst[key] = srcVal
		}
	}
}

func asMapField(value map[string]any, key string) (map[string]any, bool) {
	raw, ok := value[key]
	if !ok {
		return nil, false
	}
	m, ok := raw.(map[string]any)
	return m, ok
}

func getSecretString(m map[string]any, key string) (string, bool) {
	if raw, ok := m[key]; ok {
		s, ok := raw.(string)
		if ok {
			return s, true
		}
	}
	if raw, ok := m["_"+key]; ok {
		s, ok := raw.(string)
		if ok {
			return s, true
		}
	}
	return "", false
}

func applyConfigSecretsFromMap(cfg *config.Config, raw map[string]any) {
	channels, ok := asMapField(raw, "channels")
	if ok {
		if telegram, ok := asMapField(channels, "telegram"); ok {
			if token, ok := getSecretString(telegram, "token"); ok {
				cfg.Channels.Telegram.SetToken(token)
			}
		}
		if feishu, ok := asMapField(channels, "feishu"); ok {
			if appSecret, ok := getSecretString(feishu, "app_secret"); ok {
				cfg.Channels.Feishu.SetAppSecret(appSecret)
			}
			if encryptKey, ok := getSecretString(feishu, "encrypt_key"); ok {
				cfg.Channels.Feishu.SetEncryptKey(encryptKey)
			}
			if verificationToken, ok := getSecretString(feishu, "verification_token"); ok {
				cfg.Channels.Feishu.SetVerificationToken(verificationToken)
			}
		}
		if discord, ok := asMapField(channels, "discord"); ok {
			if token, ok := getSecretString(discord, "token"); ok {
				cfg.Channels.Discord.SetToken(token)
			}
		}
		if weixin, ok := asMapField(channels, "weixin"); ok {
			if token, ok := getSecretString(weixin, "token"); ok {
				cfg.Channels.Weixin.SetToken(token)
			}
		}
		if qq, ok := asMapField(channels, "qq"); ok {
			if appSecret, ok := getSecretString(qq, "app_secret"); ok {
				cfg.Channels.QQ.SetAppSecret(appSecret)
			}
		}
		if dingtalk, ok := asMapField(channels, "dingtalk"); ok {
			if clientSecret, ok := getSecretString(dingtalk, "client_secret"); ok {
				cfg.Channels.DingTalk.SetClientSecret(clientSecret)
			}
		}
		if slack, ok := asMapField(channels, "slack"); ok {
			if botToken, ok := getSecretString(slack, "bot_token"); ok {
				cfg.Channels.Slack.SetBotToken(botToken)
			}
			if appToken, ok := getSecretString(slack, "app_token"); ok {
				cfg.Channels.Slack.SetAppToken(appToken)
			}
		}
		if matrix, ok := asMapField(channels, "matrix"); ok {
			if accessToken, ok := getSecretString(matrix, "access_token"); ok {
				cfg.Channels.Matrix.SetAccessToken(accessToken)
			}
		}
		if line, ok := asMapField(channels, "line"); ok {
			if channelSecret, ok := getSecretString(line, "channel_secret"); ok {
				cfg.Channels.LINE.SetChannelSecret(channelSecret)
			}
			if channelAccessToken, ok := getSecretString(line, "channel_access_token"); ok {
				cfg.Channels.LINE.SetChannelAccessToken(channelAccessToken)
			}
		}
		if onebot, ok := asMapField(channels, "onebot"); ok {
			if accessToken, ok := getSecretString(onebot, "access_token"); ok {
				cfg.Channels.OneBot.SetAccessToken(accessToken)
			}
		}
		if wecom, ok := asMapField(channels, "wecom"); ok {
			if secret, ok := getSecretString(wecom, "secret"); ok {
				cfg.Channels.WeCom.SetSecret(secret)
			}
		}
		if pico, ok := asMapField(channels, "pico"); ok {
			if token, ok := getSecretString(pico, "token"); ok {
				cfg.Channels.Pico.SetToken(token)
			}
		}
		if irc, ok := asMapField(channels, "irc"); ok {
			if password, ok := getSecretString(irc, "password"); ok {
				cfg.Channels.IRC.SetPassword(password)
			}
			if nickservPassword, ok := getSecretString(irc, "nickserv_password"); ok {
				cfg.Channels.IRC.SetNickServPassword(nickservPassword)
			}
			if saslPassword, ok := getSecretString(irc, "sasl_password"); ok {
				cfg.Channels.IRC.SetSASLPassword(saslPassword)
			}
		}
	}

	tools, ok := asMapField(raw, "tools")
	if !ok {
		return
	}
	skills, ok := asMapField(tools, "skills")
	if !ok {
		return
	}
	if github, ok := asMapField(skills, "github"); ok {
		if token, ok := getSecretString(github, "token"); ok {
			cfg.Tools.Skills.Github.SetToken(token)
		}
	}
	registries, ok := asMapField(skills, "registries")
	if !ok {
		return
	}
	if clawHub, ok := asMapField(registries, "clawhub"); ok {
		if authToken, ok := getSecretString(clawHub, "auth_token"); ok {
			cfg.Tools.Skills.Registries.ClawHub.SetAuthToken(authToken)
		}
	}
}
