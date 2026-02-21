package dashboard

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/health"
)

func registerAgentsCRUD(
	srv *health.Server,
	cfg *config.Config,
	configPath string,
	auth func(http.HandlerFunc) http.HandlerFunc,
) {
	srv.HandleFunc("/dashboard/fragments/agent-edit", auth(fragmentAgentEdit(cfg)))
	srv.HandleFunc("/dashboard/fragments/agent-add", auth(fragmentAgentAdd()))
	srv.HandleFunc("/dashboard/crud/agents/create", auth(agentCreateHandler(cfg, configPath)))
	srv.HandleFunc("/dashboard/crud/agents/update", auth(agentUpdateHandler(cfg, configPath)))
	srv.HandleFunc("/dashboard/crud/agents/delete", auth(agentDeleteHandler(cfg, configPath)))
	srv.HandleFunc("/dashboard/fragments/defaults-edit", auth(fragmentDefaultsEdit(cfg)))
	srv.HandleFunc("/dashboard/crud/agents/defaults", auth(defaultsUpdateHandler(cfg, configPath)))
}

const agentFormCSS = `<style>
.form-group { margin-bottom: 12px; }
.form-group label { display: block; font-size: 12px; color: var(--fg2); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 4px; }
.form-group input, .form-group select, .form-group textarea { width: 100%; padding: 8px 10px; background: var(--bg); border: 1px solid var(--border); border-radius: 4px; color: var(--fg); font-family: inherit; font-size: 13px; }
.form-group textarea { min-height: 120px; resize: vertical; line-height: 1.5; }
.form-group input:focus, .form-group textarea:focus { border-color: var(--blue); outline: none; }
.form-group input[type="checkbox"] { width: auto; }
.form-actions { display: flex; gap: 8px; margin-top: 16px; }
.btn-primary { padding: 8px 16px; background: var(--blue); color: var(--bg); border: none; border-radius: 4px; cursor: pointer; font-family: inherit; }
.btn-secondary { padding: 8px 16px; background: var(--bg3); color: var(--fg); border: 1px solid var(--border); border-radius: 4px; cursor: pointer; font-family: inherit; }
.success { color: #3fb950; text-align: center; padding: 16px; }
</style>`

func fragmentAgentEdit(cfg *config.Config) http.HandlerFunc {
	const tmpl = `{{.CSS}}
<h3>Edit Agent</h3>
<form hx-post="/dashboard/crud/agents/update" hx-target="#modal-content" hx-swap="innerHTML">
  <div class="form-group">
    <label>ID</label>
    <input type="text" name="id" value="{{.ID}}" readonly>
  </div>
  <div class="form-group">
    <label>Name</label>
    <input type="text" name="name" value="{{.Name}}">
  </div>
  <div class="form-group">
    <label>Model (primary)</label>
    <input type="text" name="model" value="{{.Model}}" placeholder="inherited from defaults">
  </div>
  <div class="form-group">
    <label>Skills (comma-separated)</label>
    <input type="text" name="skills" value="{{.Skills}}">
  </div>
  <div class="form-group">
    <label>Workspace</label>
    <input type="text" name="workspace" value="{{.Workspace}}" placeholder="inherited from defaults">
  </div>
  <div class="form-group">
    <label>Instructions (AGENT.md)</label>
    <textarea name="instructions" placeholder="You are an expert in...">{{.Instructions}}</textarea>
  </div>
  <div class="form-group">
    <label><input type="checkbox" name="default" value="true" {{if .Default}}checked{{end}}> Default agent</label>
  </div>
  <div class="form-actions">
    <button type="submit" class="btn-primary">Save</button>
    <button type="button" class="btn-secondary" onclick="closeModal()">Cancel</button>
  </div>
</form>`
	t := template.Must(template.New("agent-edit").Parse(tmpl))

	return func(w http.ResponseWriter, r *http.Request) {
		agentID := r.URL.Query().Get("id")
		if agentID == "" {
			http.Error(w, "missing id parameter", http.StatusBadRequest)
			return
		}

		configMu.Lock()
		var found *config.AgentConfig
		for i := range cfg.Agents.List {
			if cfg.Agents.List[i].ID == agentID {
				found = &cfg.Agents.List[i]
				break
			}
		}
		configMu.Unlock()

		if found == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<p>Agent %q not found</p>`, template.HTMLEscapeString(agentID))
			return
		}

		model := ""
		if found.Model != nil {
			model = found.Model.Primary
		}

		ws := resolveAgentWorkspace(cfg, found)
		instructions := readAgentInstructions(ws)

		data := map[string]any{
			"CSS":          template.HTML(agentFormCSS),
			"ID":           found.ID,
			"Name":         found.Name,
			"Model":        model,
			"Skills":       strings.Join(found.Skills, ", "),
			"Workspace":    found.Workspace,
			"Default":      found.Default,
			"Instructions": instructions,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		t.Execute(w, data)
	}
}

func fragmentAgentAdd() http.HandlerFunc {
	const tmpl = `{{.CSS}}
<h3>Add Agent</h3>
<form hx-post="/dashboard/crud/agents/create" hx-target="#modal-content" hx-swap="innerHTML">
  <div class="form-group">
    <label>ID (required)</label>
    <input type="text" name="id" required placeholder="my-agent">
  </div>
  <div class="form-group">
    <label>Name</label>
    <input type="text" name="name" placeholder="My Agent">
  </div>
  <div class="form-group">
    <label>Model (primary)</label>
    <input type="text" name="model" placeholder="inherited from defaults">
  </div>
  <div class="form-group">
    <label>Skills (comma-separated)</label>
    <input type="text" name="skills" placeholder="search, code">
  </div>
  <div class="form-group">
    <label>Instructions (AGENT.md)</label>
    <textarea name="instructions" placeholder="You are an expert in..."></textarea>
  </div>
  <div class="form-group">
    <label><input type="checkbox" name="default" value="true"> Default agent</label>
  </div>
  <div class="form-actions">
    <button type="submit" class="btn-primary">Create</button>
    <button type="button" class="btn-secondary" onclick="closeModal()">Cancel</button>
  </div>
</form>`
	t := template.Must(template.New("agent-add").Parse(tmpl))

	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"CSS": template.HTML(agentFormCSS),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		t.Execute(w, data)
	}
}

func fragmentDefaultsEdit(cfg *config.Config) http.HandlerFunc {
	const tmpl = `{{.CSS}}
<h3>Agent Defaults</h3>
<form hx-post="/dashboard/crud/agents/defaults" hx-target="#modal-content" hx-swap="innerHTML">
  <div class="form-group">
    <label>Model</label>
    <input type="text" name="model" value="{{.Model}}">
  </div>
  <div class="form-group">
    <label>Max Tokens</label>
    <input type="number" name="max_tokens" value="{{.MaxTokens}}">
  </div>
  <div class="form-group">
    <label>Max Tool Iterations</label>
    <input type="number" name="max_tool_iterations" value="{{.MaxToolIterations}}">
  </div>
  <div class="form-group">
    <label>Workspace</label>
    <input type="text" name="workspace" value="{{.Workspace}}">
  </div>
  <div class="form-group">
    <label><input type="checkbox" name="restrict_to_workspace" value="true" {{if .RestrictToWorkspace}}checked{{end}}> Restrict to workspace</label>
  </div>
  <div class="form-actions">
    <button type="submit" class="btn-primary">Save</button>
    <button type="button" class="btn-secondary" onclick="closeModal()">Cancel</button>
  </div>
</form>`
	t := template.Must(template.New("defaults-edit").Parse(tmpl))

	return func(w http.ResponseWriter, r *http.Request) {
		configMu.Lock()
		data := map[string]any{
			"CSS":                 template.HTML(agentFormCSS),
			"Model":               cfg.Agents.Defaults.Model,
			"MaxTokens":           cfg.Agents.Defaults.MaxTokens,
			"MaxToolIterations":   cfg.Agents.Defaults.MaxToolIterations,
			"Workspace":           cfg.Agents.Defaults.Workspace,
			"RestrictToWorkspace": cfg.Agents.Defaults.RestrictToWorkspace,
		}
		configMu.Unlock()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		t.Execute(w, data)
	}
}

func agentCreateHandler(cfg *config.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := strings.TrimSpace(r.FormValue("id"))
		if id == "" {
			jsonError(w, "id is required", http.StatusBadRequest)
			return
		}

		configMu.Lock()
		defer configMu.Unlock()

		for _, a := range cfg.Agents.List {
			if a.ID == id {
				jsonError(w, "agent with this ID already exists", http.StatusConflict)
				return
			}
		}

		agent := config.AgentConfig{
			ID:      id,
			Name:    strings.TrimSpace(r.FormValue("name")),
			Default: r.FormValue("default") == "true",
		}

		model := strings.TrimSpace(r.FormValue("model"))
		if model != "" {
			agent.Model = &config.AgentModelConfig{Primary: model}
		}

		agent.Skills = parseSkills(r.FormValue("skills"))

		cfg.Agents.List = append(cfg.Agents.List, agent)

		if err := config.SaveConfig(configPath, cfg); err != nil {
			jsonError(w, "failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if instructions := r.FormValue("instructions"); strings.TrimSpace(instructions) != "" {
			ws := resolveAgentWorkspace(cfg, &agent)
			if err := writeAgentInstructions(ws, instructions); err != nil {
				jsonError(
					w,
					"agent created but failed to write AGENT.md: "+err.Error(),
					http.StatusInternalServerError,
				)
				return
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Trigger", "refreshAgents, closeModal")
		fmt.Fprint(w, `<div class="success">Agent created successfully</div>`)
	}
}

func agentUpdateHandler(cfg *config.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := strings.TrimSpace(r.FormValue("id"))
		if id == "" {
			jsonError(w, "id is required", http.StatusBadRequest)
			return
		}

		configMu.Lock()
		defer configMu.Unlock()

		var found *config.AgentConfig
		for i := range cfg.Agents.List {
			if cfg.Agents.List[i].ID == id {
				found = &cfg.Agents.List[i]
				break
			}
		}
		if found == nil {
			jsonError(w, "agent not found", http.StatusNotFound)
			return
		}

		found.Name = strings.TrimSpace(r.FormValue("name"))
		found.Default = r.FormValue("default") == "true"
		found.Workspace = strings.TrimSpace(r.FormValue("workspace"))
		found.Skills = parseSkills(r.FormValue("skills"))

		model := strings.TrimSpace(r.FormValue("model"))
		if model != "" {
			if found.Model == nil {
				found.Model = &config.AgentModelConfig{}
			}
			found.Model.Primary = model
		} else {
			found.Model = nil
		}

		if err := config.SaveConfig(configPath, cfg); err != nil {
			jsonError(w, "failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		instructions := r.FormValue("instructions")
		ws := resolveAgentWorkspace(cfg, found)
		if err := writeAgentInstructions(ws, instructions); err != nil {
			jsonError(
				w,
				"agent updated but failed to write AGENT.md: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Trigger", "refreshAgents, closeModal")
		fmt.Fprint(w, `<div class="success">Agent updated successfully</div>`)
	}
}

func agentDeleteHandler(cfg *config.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := strings.TrimSpace(r.FormValue("id"))
		if id == "" {
			jsonError(w, "id is required", http.StatusBadRequest)
			return
		}

		configMu.Lock()
		defer configMu.Unlock()

		idx := -1
		for i := range cfg.Agents.List {
			if cfg.Agents.List[i].ID == id {
				idx = i
				break
			}
		}
		if idx == -1 {
			jsonError(w, "agent not found", http.StatusNotFound)
			return
		}

		cfg.Agents.List = append(cfg.Agents.List[:idx], cfg.Agents.List[idx+1:]...)

		if err := config.SaveConfig(configPath, cfg); err != nil {
			jsonError(w, "failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Trigger", "refreshAgents, closeModal")
		fmt.Fprint(w, `<div class="success">Agent deleted successfully</div>`)
	}
}

func defaultsUpdateHandler(cfg *config.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		configMu.Lock()
		defer configMu.Unlock()

		cfg.Agents.Defaults.Model = strings.TrimSpace(r.FormValue("model"))
		cfg.Agents.Defaults.Workspace = strings.TrimSpace(r.FormValue("workspace"))
		cfg.Agents.Defaults.RestrictToWorkspace = r.FormValue("restrict_to_workspace") == "true"

		if v := strings.TrimSpace(r.FormValue("max_tokens")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				cfg.Agents.Defaults.MaxTokens = n
			}
		}
		if v := strings.TrimSpace(r.FormValue("max_tool_iterations")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				cfg.Agents.Defaults.MaxToolIterations = n
			}
		}

		if err := config.SaveConfig(configPath, cfg); err != nil {
			jsonError(w, "failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Trigger", "refreshAgents, closeModal")
		fmt.Fprint(w, `<div class="success">Defaults updated successfully</div>`)
	}
}

func resolveAgentWorkspace(cfg *config.Config, agent *config.AgentConfig) string {
	if strings.TrimSpace(agent.Workspace) != "" {
		ws := strings.TrimSpace(agent.Workspace)
		if strings.HasPrefix(ws, "~/") {
			home, _ := os.UserHomeDir()
			ws = filepath.Join(home, ws[2:])
		}
		return ws
	}
	if agent.Default || agent.ID == "" || agent.ID == "main" {
		ws := cfg.Agents.Defaults.Workspace
		if strings.HasPrefix(ws, "~/") {
			home, _ := os.UserHomeDir()
			ws = filepath.Join(home, ws[2:])
		}
		return ws
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", "workspace-"+agent.ID)
}

func readAgentInstructions(workspace string) string {
	data, err := os.ReadFile(filepath.Join(workspace, "AGENT.md"))
	if err != nil {
		return ""
	}
	return string(data)
}

func writeAgentInstructions(workspace, content string) error {
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(workspace, "AGENT.md"), []byte(content), 0o644)
}

func parseSkills(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var skills []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			skills = append(skills, p)
		}
	}
	return skills
}
