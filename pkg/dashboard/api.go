package dashboard

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func statusHandler(cfg *config.Config, al *agent.AgentLoop, cm *channels.Manager, startTime time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		info := al.GetStartupInfo()
		channelStatus := cm.GetStatus()

		resp := map[string]any{
			"uptime":   time.Since(startTime).String(),
			"running":  true,
			"tools":    info["tools"],
			"skills":   info["skills"],
			"agents":   info["agents"],
			"channels": channelStatus,
			"model":    cfg.Agents.Defaults.Model,
		}
		writeJSON(w, resp)
	}
}

func configGetHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		masked := maskConfig(cfg)
		writeJSON(w, masked)
	}
}

func agentsHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"defaults": cfg.Agents.Defaults,
			"list":     cfg.Agents.List,
		}
		writeJSON(w, resp)
	}
}

func modelsHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		masked := make([]map[string]any, 0, len(cfg.ModelList))
		for _, m := range cfg.ModelList {
			masked = append(masked, map[string]any{
				"model_name": m.ModelName,
				"model":      m.Model,
				"api_base":   m.APIBase,
				"api_key":    maskKey(m.APIKey),
			})
		}
		writeJSON(w, masked)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func maskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:3] + "..." + key[len(key)-4:]
}

func maskConfig(cfg *config.Config) map[string]any {
	models := make([]map[string]any, 0, len(cfg.ModelList))
	for _, m := range cfg.ModelList {
		models = append(models, map[string]any{
			"model_name": m.ModelName,
			"model":      m.Model,
			"api_base":   m.APIBase,
			"api_key":    maskKey(m.APIKey),
		})
	}

	channelMap := map[string]bool{
		"whatsapp":  cfg.Channels.WhatsApp.Enabled,
		"telegram":  cfg.Channels.Telegram.Enabled,
		"discord":   cfg.Channels.Discord.Enabled,
		"feishu":    cfg.Channels.Feishu.Enabled,
		"maixcam":   cfg.Channels.MaixCam.Enabled,
		"qq":        cfg.Channels.QQ.Enabled,
		"dingtalk":  cfg.Channels.DingTalk.Enabled,
		"slack":     cfg.Channels.Slack.Enabled,
		"line":      cfg.Channels.LINE.Enabled,
		"onebot":    cfg.Channels.OneBot.Enabled,
		"wecom":     cfg.Channels.WeCom.Enabled,
		"wecom_app": cfg.Channels.WeComApp.Enabled,
	}

	return map[string]any{
		"agents": map[string]any{
			"defaults": map[string]any{
				"model":      cfg.Agents.Defaults.Model,
				"provider":   cfg.Agents.Defaults.Provider,
				"workspace":  cfg.Agents.Defaults.Workspace,
				"max_tokens": cfg.Agents.Defaults.MaxTokens,
			},
			"list": cfg.Agents.List,
		},
		"model_list": models,
		"channels":   channelMap,
		"gateway": map[string]any{
			"host": cfg.Gateway.Host,
			"port": cfg.Gateway.Port,
		},
	}
}

func extractProvider(model string) string {
	if idx := strings.Index(model, "/"); idx >= 0 {
		return model[:idx]
	}
	return model
}
