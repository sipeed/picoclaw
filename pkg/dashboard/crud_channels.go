package dashboard

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/health"
)

func registerChannelsCRUD(srv *health.Server, cfg *config.Config, configPath string, auth func(http.HandlerFunc) http.HandlerFunc) {
	srv.HandleFunc("/dashboard/fragments/channel-edit", auth(fragmentChannelEdit(cfg)))
	srv.HandleFunc("/dashboard/crud/channels/update", auth(channelUpdateHandler(cfg, configPath)))
	srv.HandleFunc("/dashboard/crud/channels/toggle", auth(channelToggleHandler(cfg, configPath)))
}

const channelFormCSS = `<style>
.form-group { margin-bottom: 12px; }
.form-group label { display: block; font-size: 12px; color: var(--fg2); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 4px; }
.form-group input, .form-group select { width: 100%; padding: 8px 10px; background: var(--bg); border: 1px solid var(--border); border-radius: 4px; color: var(--fg); font-family: inherit; font-size: 13px; box-sizing: border-box; }
.form-group input:focus { border-color: var(--blue); outline: none; }
.form-group input[type="checkbox"] { width: auto; }
.form-actions { display: flex; gap: 8px; margin-top: 16px; }
.btn-primary { padding: 8px 16px; background: var(--blue); color: var(--bg); border: none; border-radius: 4px; cursor: pointer; font-family: inherit; }
.btn-secondary { padding: 8px 16px; background: var(--bg3); color: var(--fg); border: 1px solid var(--border); border-radius: 4px; cursor: pointer; font-family: inherit; }
.success { color: var(--green, #3fb950); text-align: center; padding: 16px; }
.error { color: var(--red); text-align: center; padding: 16px; }
</style>`

func channelEditFormHTML(name string, cfg *config.Config) string {
	var enabled bool
	var allowFrom []string
	var fields string

	switch name {
	case "telegram":
		ch := cfg.Channels.Telegram
		enabled = ch.Enabled
		allowFrom = ch.AllowFrom
		fields = textField("token", "Token", ch.Token) +
			textField("proxy", "Proxy", ch.Proxy)
	case "discord":
		ch := cfg.Channels.Discord
		enabled = ch.Enabled
		allowFrom = ch.AllowFrom
		fields = textField("token", "Token", ch.Token) +
			checkboxField("mention_only", "Mention Only", ch.MentionOnly)
	case "slack":
		ch := cfg.Channels.Slack
		enabled = ch.Enabled
		allowFrom = ch.AllowFrom
		fields = textField("bot_token", "Bot Token", ch.BotToken) +
			textField("app_token", "App Token", ch.AppToken)
	case "whatsapp":
		ch := cfg.Channels.WhatsApp
		enabled = ch.Enabled
		allowFrom = ch.AllowFrom
		fields = textField("bridge_url", "Bridge URL", ch.BridgeURL)
	case "feishu":
		ch := cfg.Channels.Feishu
		enabled = ch.Enabled
		allowFrom = ch.AllowFrom
		fields = textField("app_id", "App ID", ch.AppID) +
			textField("app_secret", "App Secret", ch.AppSecret) +
			textField("encrypt_key", "Encrypt Key", ch.EncryptKey) +
			textField("verification_token", "Verification Token", ch.VerificationToken)
	case "dingtalk":
		ch := cfg.Channels.DingTalk
		enabled = ch.Enabled
		allowFrom = ch.AllowFrom
		fields = textField("client_id", "Client ID", ch.ClientID) +
			textField("client_secret", "Client Secret", ch.ClientSecret)
	case "qq":
		ch := cfg.Channels.QQ
		enabled = ch.Enabled
		allowFrom = ch.AllowFrom
		fields = textField("app_id", "App ID", ch.AppID) +
			textField("app_secret", "App Secret", ch.AppSecret)
	case "line":
		ch := cfg.Channels.LINE
		enabled = ch.Enabled
		allowFrom = ch.AllowFrom
		fields = textField("channel_secret", "Channel Secret", ch.ChannelSecret) +
			textField("channel_access_token", "Channel Access Token", ch.ChannelAccessToken) +
			textField("webhook_host", "Webhook Host", ch.WebhookHost) +
			textField("webhook_port", "Webhook Port", strconv.Itoa(ch.WebhookPort)) +
			textField("webhook_path", "Webhook Path", ch.WebhookPath)
	case "maixcam":
		ch := cfg.Channels.MaixCam
		enabled = ch.Enabled
		allowFrom = ch.AllowFrom
		fields = textField("host", "Host", ch.Host) +
			textField("port", "Port", strconv.Itoa(ch.Port))
	case "onebot":
		ch := cfg.Channels.OneBot
		enabled = ch.Enabled
		allowFrom = ch.AllowFrom
		fields = textField("ws_url", "WebSocket URL", ch.WSUrl) +
			textField("access_token", "Access Token", ch.AccessToken) +
			textField("reconnect_interval", "Reconnect Interval", strconv.Itoa(ch.ReconnectInterval))
	default:
		return ""
	}

	checkedAttr := ""
	if enabled {
		checkedAttr = " checked"
	}

	allowFromStr := strings.Join(allowFrom, ", ")

	return channelFormCSS + `<div>
  <h3>Edit Channel: ` + template.HTMLEscapeString(name) + `</h3>
  <form hx-post="/dashboard/crud/channels/update" hx-target="#modal-content" hx-swap="innerHTML">
    <input type="hidden" name="name" value="` + template.HTMLEscapeString(name) + `">
    <div class="form-group">
      <label>Enabled</label>
      <input type="checkbox" name="enabled" value="true"` + checkedAttr + `>
    </div>` +
		fields + `
    <div class="form-group">
      <label>Allow From (comma-separated)</label>
      <input type="text" name="allow_from" value="` + template.HTMLEscapeString(allowFromStr) + `">
    </div>
    <div class="form-actions">
      <button type="submit" class="btn-primary">Save</button>
      <button type="button" class="btn-secondary" onclick="closeModal()">Cancel</button>
    </div>
  </form>
</div>`
}

func textField(name, label, value string) string {
	return `
    <div class="form-group">
      <label>` + template.HTMLEscapeString(label) + `</label>
      <input type="text" name="` + template.HTMLEscapeString(name) + `" value="` + template.HTMLEscapeString(value) + `">
    </div>`
}

func checkboxField(name, label string, checked bool) string {
	checkedAttr := ""
	if checked {
		checkedAttr = " checked"
	}
	return `
    <div class="form-group">
      <label>` + template.HTMLEscapeString(label) + `</label>
      <input type="checkbox" name="` + template.HTMLEscapeString(name) + `" value="true"` + checkedAttr + `>
    </div>`
}

func fragmentChannelEdit(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		html := channelEditFormHTML(name, cfg)
		if html == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<p class="error">Unknown channel: %s</p>`, template.HTMLEscapeString(name))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	}
}

func parseAllowFrom(s string) config.FlexibleStringSlice {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make(config.FlexibleStringSlice, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func channelUpdateHandler(cfg *config.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, `<p class="error">Method not allowed</p>`)
			return
		}

		name := r.FormValue("name")
		enabled := r.FormValue("enabled") == "true"
		allowFrom := parseAllowFrom(r.FormValue("allow_from"))

		configMu.Lock()
		switch name {
		case "telegram":
			cfg.Channels.Telegram.Enabled = enabled
			cfg.Channels.Telegram.Token = r.FormValue("token")
			cfg.Channels.Telegram.Proxy = r.FormValue("proxy")
			cfg.Channels.Telegram.AllowFrom = allowFrom
		case "discord":
			cfg.Channels.Discord.Enabled = enabled
			cfg.Channels.Discord.Token = r.FormValue("token")
			cfg.Channels.Discord.MentionOnly = r.FormValue("mention_only") == "true"
			cfg.Channels.Discord.AllowFrom = allowFrom
		case "slack":
			cfg.Channels.Slack.Enabled = enabled
			cfg.Channels.Slack.BotToken = r.FormValue("bot_token")
			cfg.Channels.Slack.AppToken = r.FormValue("app_token")
			cfg.Channels.Slack.AllowFrom = allowFrom
		case "whatsapp":
			cfg.Channels.WhatsApp.Enabled = enabled
			cfg.Channels.WhatsApp.BridgeURL = r.FormValue("bridge_url")
			cfg.Channels.WhatsApp.AllowFrom = allowFrom
		case "feishu":
			cfg.Channels.Feishu.Enabled = enabled
			cfg.Channels.Feishu.AppID = r.FormValue("app_id")
			cfg.Channels.Feishu.AppSecret = r.FormValue("app_secret")
			cfg.Channels.Feishu.EncryptKey = r.FormValue("encrypt_key")
			cfg.Channels.Feishu.VerificationToken = r.FormValue("verification_token")
			cfg.Channels.Feishu.AllowFrom = allowFrom
		case "dingtalk":
			cfg.Channels.DingTalk.Enabled = enabled
			cfg.Channels.DingTalk.ClientID = r.FormValue("client_id")
			cfg.Channels.DingTalk.ClientSecret = r.FormValue("client_secret")
			cfg.Channels.DingTalk.AllowFrom = allowFrom
		case "qq":
			cfg.Channels.QQ.Enabled = enabled
			cfg.Channels.QQ.AppID = r.FormValue("app_id")
			cfg.Channels.QQ.AppSecret = r.FormValue("app_secret")
			cfg.Channels.QQ.AllowFrom = allowFrom
		case "line":
			cfg.Channels.LINE.Enabled = enabled
			cfg.Channels.LINE.ChannelSecret = r.FormValue("channel_secret")
			cfg.Channels.LINE.ChannelAccessToken = r.FormValue("channel_access_token")
			cfg.Channels.LINE.WebhookHost = r.FormValue("webhook_host")
			if p, err := strconv.Atoi(r.FormValue("webhook_port")); err == nil {
				cfg.Channels.LINE.WebhookPort = p
			}
			cfg.Channels.LINE.WebhookPath = r.FormValue("webhook_path")
			cfg.Channels.LINE.AllowFrom = allowFrom
		case "maixcam":
			cfg.Channels.MaixCam.Enabled = enabled
			cfg.Channels.MaixCam.Host = r.FormValue("host")
			if p, err := strconv.Atoi(r.FormValue("port")); err == nil {
				cfg.Channels.MaixCam.Port = p
			}
			cfg.Channels.MaixCam.AllowFrom = allowFrom
		case "onebot":
			cfg.Channels.OneBot.Enabled = enabled
			cfg.Channels.OneBot.WSUrl = r.FormValue("ws_url")
			cfg.Channels.OneBot.AccessToken = r.FormValue("access_token")
			if ri, err := strconv.Atoi(r.FormValue("reconnect_interval")); err == nil {
				cfg.Channels.OneBot.ReconnectInterval = ri
			}
			cfg.Channels.OneBot.AllowFrom = allowFrom
		default:
			configMu.Unlock()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<p class="error">Unknown channel: %s</p>`, template.HTMLEscapeString(name))
			return
		}
		configMu.Unlock()

		if configPath != "" {
			if err := saveConfig(configPath, cfg); err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `<p class="error">Failed to save: %s</p>`, template.HTMLEscapeString(err.Error()))
				return
			}
		}

		w.Header().Set("HX-Trigger", "refreshChannels, closeModal")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<p class="success">Channel updated</p>`)
	}
}

func channelToggleHandler(cfg *config.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, `<p class="error">Method not allowed</p>`)
			return
		}

		name := r.FormValue("name")
		enabled := r.FormValue("enabled") == "true"

		configMu.Lock()
		switch name {
		case "telegram":
			cfg.Channels.Telegram.Enabled = enabled
		case "discord":
			cfg.Channels.Discord.Enabled = enabled
		case "slack":
			cfg.Channels.Slack.Enabled = enabled
		case "whatsapp":
			cfg.Channels.WhatsApp.Enabled = enabled
		case "feishu":
			cfg.Channels.Feishu.Enabled = enabled
		case "dingtalk":
			cfg.Channels.DingTalk.Enabled = enabled
		case "qq":
			cfg.Channels.QQ.Enabled = enabled
		case "line":
			cfg.Channels.LINE.Enabled = enabled
		case "maixcam":
			cfg.Channels.MaixCam.Enabled = enabled
		case "onebot":
			cfg.Channels.OneBot.Enabled = enabled
		default:
			configMu.Unlock()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<p class="error">Unknown channel: %s</p>`, template.HTMLEscapeString(name))
			return
		}
		configMu.Unlock()

		if configPath != "" {
			if err := saveConfig(configPath, cfg); err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `<p class="error">Failed to save: %s</p>`, template.HTMLEscapeString(err.Error()))
				return
			}
		}

		w.Header().Set("HX-Trigger", "refreshChannels")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<p class="success">Channel toggled</p>`)
	}
}
