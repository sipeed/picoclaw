package dashboard

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/logger"
)

func Mount(
	srv *health.Server,
	cfg *config.Config,
	al *agent.AgentLoop,
	cm *channels.Manager,
	configPath ...string,
) {
	startTime := time.Now()
	broker := NewBroker()
	password := cfg.Dashboard.Password

	cfgPath := ""
	if len(configPath) > 0 {
		cfgPath = configPath[0]
	}

	// Start background polling for SSE events
	go pollStatus(broker, cfg, al, cm)

	// Static files (public — needed for login page)
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		logger.ErrorCF("dashboard", "Failed to create sub FS", map[string]any{"error": err.Error()})
		return
	}
	srv.Handle(
		"/dashboard/static/",
		http.StripPrefix("/dashboard/static/", http.FileServer(http.FS(staticFS))),
	)

	// Auth routes (public)
	srv.HandleFunc("/dashboard/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			loginHandler(password)(w, r)
			return
		}
		loginPage(password)(w, r)
	})
	srv.HandleFunc("/dashboard/logout", logoutHandler())

	// Protected: wrap with authMiddleware
	auth := func(h http.HandlerFunc) http.HandlerFunc {
		return authMiddleware(password, h)
	}

	// Main page
	srv.HandleFunc("/dashboard", auth(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		indexHTML, err := staticFiles.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		w.Write(indexHTML)
	}))

	// SSE endpoint
	srv.HandleFunc("/dashboard/events", auth(broker.Subscribe))

	// JSON API
	srv.HandleFunc("/dashboard/api/status", auth(statusHandler(cfg, al, cm, startTime)))
	srv.HandleFunc("/dashboard/api/config", auth(configGetHandler(cfg)))
	srv.HandleFunc("/dashboard/api/agents", auth(agentsHandler(cfg)))
	srv.HandleFunc("/dashboard/api/models", auth(modelsHandler(cfg)))

	// HTMX fragments
	srv.HandleFunc("/dashboard/fragments/status", auth(fragmentStatus(cfg, al, cm, startTime)))
	srv.HandleFunc("/dashboard/fragments/agents", auth(fragmentAgents(cfg, al)))
	srv.HandleFunc("/dashboard/fragments/agent-detail", auth(fragmentAgentDetail(cfg)))
	srv.HandleFunc("/dashboard/fragments/tools", auth(fragmentTools(al)))
	srv.HandleFunc("/dashboard/fragments/channels", auth(fragmentChannels(cm)))
	srv.HandleFunc("/dashboard/fragments/models", auth(fragmentModels(cfg)))

	// CRUD routes (require configPath for saving)
	registerModelsCRUD(srv, cfg, cfgPath, auth)
	registerAgentsCRUD(srv, cfg, cfgPath, auth)
	registerChannelsCRUD(srv, cfg, cfgPath, auth)
	registerSettingsCRUD(srv, cfg, cfgPath, password, auth)

	logger.InfoC("dashboard", "Dashboard mounted at /dashboard")
}

func pollStatus(broker *SSEBroker, cfg *config.Config, al *agent.AgentLoop, cm *channels.Manager) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if broker.ClientCount() == 0 {
			continue
		}
		info := al.GetStartupInfo()
		channelStatus := cm.GetStatus()
		data := map[string]any{
			"tools":    info["tools"],
			"agents":   info["agents"],
			"channels": channelStatus,
			"model":    cfg.Agents.Defaults.Model,
		}
		jsonData, err := json.Marshal(data)
		if err == nil {
			broker.Publish("status", string(jsonData))
		}
	}
}

var funcMap = template.FuncMap{
	"maskKey":         maskKey,
	"extractProvider": extractProvider,
}

func fragmentStatus(
	cfg *config.Config,
	al *agent.AgentLoop,
	cm *channels.Manager,
	startTime time.Time,
) http.HandlerFunc {
	const tmpl = `<div id="status-bar" class="status-bar">
  <span class="indicator {{if .Running}}running{{else}}stopped{{end}}"></span>
  <span>{{.Model}}</span>
  <span class="sep">|</span>
  <span>Uptime: {{.Uptime}}</span>
  <span class="sep">|</span>
  <span>Tools: {{.ToolCount}}</span>
  <span class="sep">|</span>
  <span>Agents: {{.AgentCount}}</span>
  <span class="sep">|</span>
  <span>Channels: {{.ChannelCount}}</span>
</div>`
	t := template.Must(template.New("status").Parse(tmpl))

	return func(w http.ResponseWriter, r *http.Request) {
		info := al.GetStartupInfo()
		toolsInfo, _ := info["tools"].(map[string]any)
		agentsInfo, _ := info["agents"].(map[string]any)
		channelStatus := cm.GetStatus()

		toolCount := 0
		if tc, ok := toolsInfo["count"]; ok {
			toolCount, _ = tc.(int)
		}
		agentCount := 0
		if ac, ok := agentsInfo["count"]; ok {
			agentCount, _ = ac.(int)
		}

		data := map[string]any{
			"Running":      true,
			"Model":        cfg.Agents.Defaults.Model,
			"Uptime":       formatUptime(time.Since(startTime)),
			"ToolCount":    toolCount,
			"AgentCount":   agentCount,
			"ChannelCount": len(channelStatus),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		t.Execute(w, data)
	}
}

func fragmentAgents(cfg *config.Config, al *agent.AgentLoop) http.HandlerFunc {
	const tmpl = `<div id="agents-table" hx-trigger="refreshAgents from:body" hx-get="/dashboard/fragments/agents" hx-swap="outerHTML">
  <div style="display:flex;justify-content:flex-end;gap:8px;margin-bottom:8px">
    <button class="action-btn edit" hx-get="/dashboard/fragments/defaults-edit" hx-target="#modal-content" hx-swap="innerHTML">Defaults</button>
    <button class="action-btn add" hx-get="/dashboard/fragments/agent-add" hx-target="#modal-content" hx-swap="innerHTML">+ Add Agent</button>
  </div>
  <div class="table-wrap">
  <table>
    <thead>
      <tr><th>ID</th><th>Name</th><th>Model</th><th>Default</th><th>Skills</th><th>Actions</th></tr>
    </thead>
    <tbody>
      {{range .Agents}}
      <tr>
        <td>{{.ID}}</td>
        <td>{{.Name}}</td>
        <td>{{if .Model}}{{.Model.Primary}}{{else}}<em>default</em>{{end}}</td>
        <td>{{if .Default}}&#10003;{{end}}</td>
        <td>{{range .Skills}}<span class="tag">{{.}}</span> {{end}}</td>
        <td class="actions">
          <button class="action-btn edit" hx-get="/dashboard/fragments/agent-edit?id={{.ID}}" hx-target="#modal-content" hx-swap="innerHTML">Edit</button>
          <button class="action-btn delete" hx-post="/dashboard/crud/agents/delete" hx-vals='{"id":"{{.ID}}"}' hx-target="#modal-content" hx-swap="innerHTML" hx-confirm="Delete agent {{.ID}}?">Del</button>
        </td>
      </tr>
      {{end}}
      {{if not .Agents}}
      <tr><td colspan="6" class="empty">No agents configured (using defaults)</td></tr>
      {{end}}
    </tbody>
  </table>
  </div>
  <p class="note">Default model: {{.DefaultModel}} | Max tokens: {{.MaxTokens}} | Click a row for details</p>
</div>`
	t := template.Must(template.New("agents").Parse(tmpl))

	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Agents":       cfg.Agents.List,
			"DefaultModel": cfg.Agents.Defaults.Model,
			"MaxTokens":    cfg.Agents.Defaults.MaxTokens,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		t.Execute(w, data)
	}
}

func fragmentAgentDetail(cfg *config.Config) http.HandlerFunc {
	const tmpl = `<div>
  <h3>Agent: {{.Name}}</h3>
  <div class="detail-grid">
    <span class="detail-label">ID</span>
    <span class="detail-value">{{.ID}}</span>
    <span class="detail-label">Name</span>
    <span class="detail-value">{{if .Name}}{{.Name}}{{else}}<em>unnamed</em>{{end}}</span>
    <span class="detail-label">Default</span>
    <span class="detail-value">{{if .Default}}Yes{{else}}No{{end}}</span>
    <span class="detail-label">Model</span>
    <span class="detail-value">{{if .Model}}{{.Model.Primary}}{{if .Model.Fallbacks}} (fallbacks: {{range $i, $f := .Model.Fallbacks}}{{if $i}}, {{end}}{{$f}}{{end}}){{end}}{{else}}<em>inherited</em>{{end}}</span>
    <span class="detail-label">Workspace</span>
    <span class="detail-value">{{if .Workspace}}{{.Workspace}}{{else}}<em>default</em>{{end}}</span>
  </div>
  {{if .Skills}}
  <div class="detail-section">
    <h4>Skills</h4>
    <div class="tool-list">
      {{range .Skills}}<span class="tool-tag">{{.}}</span>{{end}}
    </div>
  </div>
  {{end}}
  {{if .Subagents}}
  <div class="detail-section">
    <h4>Allowed Subagents</h4>
    <div class="tool-list">
      {{range .Subagents.AllowAgents}}<span class="tool-tag">{{.}}</span>{{end}}
    </div>
  </div>
  {{end}}
</div>`
	t := template.Must(template.New("agent-detail").Parse(tmpl))

	return func(w http.ResponseWriter, r *http.Request) {
		agentID := r.URL.Query().Get("id")
		if agentID == "" {
			http.Error(w, "missing id parameter", http.StatusBadRequest)
			return
		}
		for _, a := range cfg.Agents.List {
			if a.ID == agentID {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				t.Execute(w, a)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(
			w,
			`<div><h3>Agent not found</h3><p>No agent with ID "%s"</p></div>`,
			template.HTMLEscapeString(agentID),
		)
	}
}

func fragmentTools(al *agent.AgentLoop) http.HandlerFunc {
	const tmpl = `<div id="tools-section">
  <div class="detail-section" style="margin-top:0;padding-top:0;border-top:none">
    <h4>Tools ({{.ToolCount}})</h4>
    <div class="tool-list">
      {{range .Tools}}<span class="tool-tag">{{.}}</span>{{end}}
      {{if not .Tools}}<span class="empty">No tools loaded</span>{{end}}
    </div>
  </div>
  <div class="detail-section">
    <h4>Skills ({{.SkillAvailable}}/{{.SkillTotal}})</h4>
    <div class="tool-list">
      {{range .Skills}}<span class="tool-tag">{{.}}</span>{{end}}
      {{if not .Skills}}<span class="empty">No skills available</span>{{end}}
    </div>
  </div>
</div>`
	t := template.Must(template.New("tools").Parse(tmpl))

	return func(w http.ResponseWriter, r *http.Request) {
		info := al.GetStartupInfo()

		toolsInfo, _ := info["tools"].(map[string]any)
		skillsInfo, _ := info["skills"].(map[string]any)

		toolNames, _ := toolsInfo["names"].([]string)
		toolCount := 0
		if tc, ok := toolsInfo["count"]; ok {
			toolCount, _ = tc.(int)
		}

		skillNames, _ := skillsInfo["names"].([]string)
		skillAvailable := 0
		if sa, ok := skillsInfo["available"]; ok {
			skillAvailable, _ = sa.(int)
		}
		skillTotal := 0
		if st, ok := skillsInfo["total"]; ok {
			skillTotal, _ = st.(int)
		}

		data := map[string]any{
			"Tools":          toolNames,
			"ToolCount":      toolCount,
			"Skills":         skillNames,
			"SkillAvailable": skillAvailable,
			"SkillTotal":     skillTotal,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		t.Execute(w, data)
	}
}

func fragmentChannels(cm *channels.Manager) http.HandlerFunc {
	const tmpl = `<div id="channels-table" hx-trigger="refreshChannels from:body" hx-get="/dashboard/fragments/channels" hx-swap="outerHTML">
  <div class="table-wrap">
  <table>
    <thead>
      <tr><th>Channel</th><th>Enabled</th><th>Running</th><th>Actions</th></tr>
    </thead>
    <tbody>
      {{range $name, $status := .}}
      <tr>
        <td>{{$name}}</td>
        <td>{{if index $status "enabled"}}&#10003;{{else}}&#10007;{{end}}</td>
        <td>
          {{if index $status "running"}}
          <span class="indicator running"></span> Running
          {{else}}
          <span class="indicator stopped"></span> Stopped
          {{end}}
        </td>
        <td class="actions">
          <button class="action-btn edit" hx-get="/dashboard/fragments/channel-edit?name={{$name}}" hx-target="#modal-content" hx-swap="innerHTML">Edit</button>
        </td>
      </tr>
      {{end}}
      {{if not .}}
      <tr><td colspan="4" class="empty">No channels configured</td></tr>
      {{end}}
    </tbody>
  </table>
  </div>
</div>`
	t := template.Must(template.New("channels").Parse(tmpl))

	return func(w http.ResponseWriter, r *http.Request) {
		status := cm.GetStatus()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		t.Execute(w, status)
	}
}

func fragmentModels(cfg *config.Config) http.HandlerFunc {
	const tmpl = `<div id="models-table" hx-trigger="refreshModels from:body" hx-get="/dashboard/fragments/models" hx-swap="outerHTML">
  <div style="display:flex;justify-content:flex-end;margin-bottom:8px">
    <button class="action-btn add" hx-get="/dashboard/fragments/model-add" hx-target="#modal-content" hx-swap="innerHTML">+ Add Model</button>
  </div>
  <div class="table-wrap">
  <table>
    <thead>
      <tr><th>Name</th><th>Provider</th><th>Model</th><th>API Base</th><th>API Key</th><th>Actions</th></tr>
    </thead>
    <tbody>
      {{range $i, $m := .}}
      <tr>
        <td>{{$m.ModelName}}</td>
        <td>{{extractProvider $m.Model}}</td>
        <td>{{$m.Model}}</td>
        <td>{{if $m.APIBase}}{{$m.APIBase}}{{else}}<em>default</em>{{end}}</td>
        <td>{{maskKey $m.APIKey}}</td>
        <td class="actions">
          <button class="action-btn edit" hx-get="/dashboard/fragments/model-edit?idx={{$i}}" hx-target="#modal-content" hx-swap="innerHTML">Edit</button>
          <button class="action-btn delete" hx-post="/dashboard/crud/models/delete" hx-vals='{"idx":"{{$i}}"}' hx-target="#modal-content" hx-swap="innerHTML" hx-confirm="Delete model {{$m.ModelName}}?">Del</button>
        </td>
      </tr>
      {{end}}
      {{if not .}}
      <tr><td colspan="6" class="empty">No models configured</td></tr>
      {{end}}
    </tbody>
  </table>
  </div>
</div>`
	t := template.Must(template.New("models").Funcs(funcMap).Parse(tmpl))

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		t.Execute(w, cfg.ModelList)
	}
}

func formatUptime(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	if hours > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm%02ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
