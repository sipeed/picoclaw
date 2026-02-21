package dashboard

import (
	"crypto/hmac"
	"fmt"
	"net/http"
	"strconv"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/health"
)

func registerSettingsCRUD(
	srv *health.Server,
	cfg *config.Config,
	configPath string,
	currentPassword string,
	auth func(http.HandlerFunc) http.HandlerFunc,
) {
	srv.HandleFunc("/dashboard/fragments/settings", auth(fragmentSettings(cfg, currentPassword)))
	srv.HandleFunc(
		"/dashboard/crud/settings/password",
		auth(passwordChangeHandler(cfg, configPath)),
	)
	srv.HandleFunc("/dashboard/crud/settings/gateway", auth(gatewayUpdateHandler(cfg, configPath)))
	srv.HandleFunc(
		"/dashboard/crud/settings/heartbeat",
		auth(heartbeatUpdateHandler(cfg, configPath)),
	)
	srv.HandleFunc("/dashboard/crud/settings/devices", auth(devicesUpdateHandler(cfg, configPath)))
}

const settingsCSS = `<style>
.settings-section { margin-bottom: 24px; padding-bottom: 20px; border-bottom: 1px solid var(--border); }
.settings-section h4 { font-size: 13px; color: var(--fg2); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 12px; }
.form-group { margin-bottom: 12px; }
.form-group label { display: block; font-size: 12px; color: var(--fg2); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 4px; }
.form-group input { width: 100%; padding: 8px 10px; background: var(--bg); border: 1px solid var(--border); border-radius: 4px; color: var(--fg); font-family: inherit; font-size: 13px; }
.form-group input:focus { border-color: var(--blue); outline: none; }
.form-group input[type="checkbox"], .form-group input[type="number"] { width: auto; }
.form-actions { display: flex; gap: 8px; margin-top: 12px; }
.btn-primary { padding: 8px 16px; background: var(--blue); color: var(--bg); border: none; border-radius: 4px; cursor: pointer; font-family: inherit; }
.success { color: #3fb950; padding: 8px; font-size: 13px; }
.error { color: #f85149; padding: 8px; font-size: 13px; }
.note { color: var(--fg2); font-size: 11px; margin-top: 4px; }
</style>`

func fragmentSettings(cfg *config.Config, currentPassword string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		checkedAttr := func(b bool) string {
			if b {
				return " checked"
			}
			return ""
		}

		html := settingsCSS + `<div id="settings-content">
<div id="settings-result"></div>

<div class="settings-section">
  <h4>Dashboard Password</h4>
  <form hx-post="/dashboard/crud/settings/password" hx-target="#settings-result" hx-swap="innerHTML">
    <div class="form-group">
      <label>Current Password</label>
      <input type="password" name="current_password" required>
    </div>
    <div class="form-group">
      <label>New Password</label>
      <input type="password" name="new_password" required minlength="8">
    </div>
    <div class="form-actions">
      <button type="submit" class="btn-primary">Change Password</button>
    </div>
  </form>
</div>

<div class="settings-section">
  <h4>Gateway</h4>
  <form hx-post="/dashboard/crud/settings/gateway" hx-target="#settings-result" hx-swap="innerHTML">
    <div class="form-group">
      <label>Host</label>
      <input type="text" name="host" value="` + cfg.Gateway.Host + `">
    </div>
    <div class="form-group">
      <label>Port</label>
      <input type="number" name="port" value="` + strconv.Itoa(cfg.Gateway.Port) + `">
    </div>
    <div class="form-actions">
      <button type="submit" class="btn-primary">Save Gateway</button>
    </div>
    <p class="note">Changes take effect on restart</p>
  </form>
</div>

<div class="settings-section">
  <h4>Heartbeat</h4>
  <form hx-post="/dashboard/crud/settings/heartbeat" hx-target="#settings-result" hx-swap="innerHTML">
    <div class="form-group">
      <label><input type="checkbox" name="enabled"` + checkedAttr(cfg.Heartbeat.Enabled) + `> Enabled</label>
    </div>
    <div class="form-group">
      <label>Interval (minutes)</label>
      <input type="number" name="interval" value="` + strconv.Itoa(cfg.Heartbeat.Interval) + `" min="1">
    </div>
    <div class="form-actions">
      <button type="submit" class="btn-primary">Save Heartbeat</button>
    </div>
  </form>
</div>

<div class="settings-section">
  <h4>Devices</h4>
  <form hx-post="/dashboard/crud/settings/devices" hx-target="#settings-result" hx-swap="innerHTML">
    <div class="form-group">
      <label><input type="checkbox" name="enabled"` + checkedAttr(cfg.Devices.Enabled) + `> Enabled</label>
    </div>
    <div class="form-group">
      <label><input type="checkbox" name="monitor_usb"` + checkedAttr(cfg.Devices.MonitorUSB) + `> Monitor USB</label>
    </div>
    <div class="form-actions">
      <button type="submit" class="btn-primary">Save Devices</button>
    </div>
  </form>
</div>

</div>`
		fmt.Fprint(w, html)
	}
}

func passwordChangeHandler(cfg *config.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, `<p class="error">Method not allowed</p>`)
			return
		}

		currentPassword := r.FormValue("current_password")
		newPassword := r.FormValue("new_password")

		if !hmac.Equal([]byte(currentPassword), []byte(cfg.Dashboard.Password)) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `<p class="error">Current password is incorrect</p>`)
			return
		}

		if len(newPassword) < 8 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `<p class="error">New password must be at least 8 characters</p>`)
			return
		}

		cfg.Dashboard.Password = newPassword

		if configPath != "" {
			if err := saveConfig(configPath, cfg); err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `<p class="error">Failed to save config</p>`)
				return
			}
		}

		value, expiry := signSession(newPassword)
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    value,
			Path:     "/dashboard",
			Expires:  expiry,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<p class="success">Password changed successfully</p>`)
	}
}

func gatewayUpdateHandler(cfg *config.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, `<p class="error">Method not allowed</p>`)
			return
		}

		host := r.FormValue("host")
		portStr := r.FormValue("port")
		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 || port >= 65536 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `<p class="error">Port must be between 1 and 65535</p>`)
			return
		}

		cfg.Gateway.Host = host
		cfg.Gateway.Port = port

		if configPath != "" {
			if err := saveConfig(configPath, cfg); err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `<p class="error">Failed to save config</p>`)
				return
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<p class="success">Gateway settings saved (restart required)</p>`)
	}
}

func heartbeatUpdateHandler(cfg *config.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, `<p class="error">Method not allowed</p>`)
			return
		}

		enabled := r.FormValue("enabled") == "on"
		intervalStr := r.FormValue("interval")
		interval, err := strconv.Atoi(intervalStr)
		if err != nil || interval < 1 {
			interval = cfg.Heartbeat.Interval
		}

		cfg.Heartbeat.Enabled = enabled
		cfg.Heartbeat.Interval = interval

		if configPath != "" {
			if err := saveConfig(configPath, cfg); err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `<p class="error">Failed to save config</p>`)
				return
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<p class="success">Heartbeat settings saved</p>`)
	}
}

func devicesUpdateHandler(cfg *config.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, `<p class="error">Method not allowed</p>`)
			return
		}

		cfg.Devices.Enabled = r.FormValue("enabled") == "on"
		cfg.Devices.MonitorUSB = r.FormValue("monitor_usb") == "on"

		if configPath != "" {
			if err := saveConfig(configPath, cfg); err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `<p class="error">Failed to save config</p>`)
				return
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<p class="success">Devices settings saved</p>`)
	}
}
