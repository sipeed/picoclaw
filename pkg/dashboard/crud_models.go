package dashboard

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/health"
)

func registerModelsCRUD(srv *health.Server, cfg *config.Config, configPath string, auth func(http.HandlerFunc) http.HandlerFunc) {
	srv.HandleFunc("/dashboard/fragments/model-edit", auth(fragmentModelEdit(cfg)))
	srv.HandleFunc("/dashboard/fragments/model-add", auth(fragmentModelAdd()))
	srv.HandleFunc("/dashboard/crud/models/create", auth(modelCreateHandler(cfg, configPath)))
	srv.HandleFunc("/dashboard/crud/models/update", auth(modelUpdateHandler(cfg, configPath)))
	srv.HandleFunc("/dashboard/crud/models/delete", auth(modelDeleteHandler(cfg, configPath)))
}

const modelFormCSS = `<style>
.form-group { margin-bottom: 12px; }
.form-group label { display: block; font-size: 12px; color: var(--fg2); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 4px; }
.form-group input, .form-group select { width: 100%; padding: 8px 10px; background: var(--bg); border: 1px solid var(--border); border-radius: 4px; color: var(--fg); font-family: inherit; font-size: 13px; box-sizing: border-box; }
.form-group input:focus { border-color: var(--blue); outline: none; }
.form-actions { display: flex; gap: 8px; margin-top: 16px; }
.btn-primary { padding: 8px 16px; background: var(--blue); color: var(--bg); border: none; border-radius: 4px; cursor: pointer; font-family: inherit; }
.btn-secondary { padding: 8px 16px; background: var(--bg3); color: var(--fg); border: 1px solid var(--border); border-radius: 4px; cursor: pointer; font-family: inherit; }
.btn-danger { padding: 8px 16px; background: var(--red); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-family: inherit; }
.success { color: var(--green, #3fb950); text-align: center; padding: 16px; }
.error { color: var(--red); text-align: center; padding: 16px; }
</style>`

const modelEditTmpl = modelFormCSS + `<div>
  <h3>Edit Model</h3>
  <form hx-post="/dashboard/crud/models/update" hx-target="#modal-content" hx-swap="innerHTML">
    <input type="hidden" name="idx" value="{{.Idx}}">
    <div class="form-group">
      <label>Model Name</label>
      <input type="text" name="model_name" value="{{.ModelName}}" required>
    </div>
    <div class="form-group">
      <label>Model</label>
      <input type="text" name="model" value="{{.Model}}" required>
    </div>
    <div class="form-group">
      <label>API Base</label>
      <input type="text" name="api_base" value="{{.APIBase}}">
    </div>
    <div class="form-group">
      <label>API Key</label>
      <input type="text" name="api_key" value="{{.APIKey}}">
    </div>
    <div class="form-group">
      <label>Proxy</label>
      <input type="text" name="proxy" value="{{.Proxy}}">
    </div>
    <div class="form-actions">
      <button type="submit" class="btn-primary">Save</button>
      <button type="button" class="btn-secondary" onclick="closeModal()">Cancel</button>
    </div>
  </form>
</div>`

const modelAddTmpl = modelFormCSS + `<div>
  <h3>Add Model</h3>
  <form hx-post="/dashboard/crud/models/create" hx-target="#modal-content" hx-swap="innerHTML">
    <div class="form-group">
      <label>Model Name</label>
      <input type="text" name="model_name" value="" required>
    </div>
    <div class="form-group">
      <label>Model</label>
      <input type="text" name="model" value="" required>
    </div>
    <div class="form-group">
      <label>API Base</label>
      <input type="text" name="api_base" value="">
    </div>
    <div class="form-group">
      <label>API Key</label>
      <input type="text" name="api_key" value="">
    </div>
    <div class="form-group">
      <label>Proxy</label>
      <input type="text" name="proxy" value="">
    </div>
    <div class="form-actions">
      <button type="submit" class="btn-primary">Add</button>
      <button type="button" class="btn-secondary" onclick="closeModal()">Cancel</button>
    </div>
  </form>
</div>`

func fragmentModelEdit(cfg *config.Config) http.HandlerFunc {
	t := template.Must(template.New("model-edit").Parse(modelEditTmpl))

	return func(w http.ResponseWriter, r *http.Request) {
		idxStr := r.URL.Query().Get("idx")
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 0 || idx >= len(cfg.ModelList) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<p class="error">Invalid model index</p>`)
			return
		}

		m := cfg.ModelList[idx]
		data := map[string]any{
			"Idx":       idx,
			"ModelName": m.ModelName,
			"Model":     m.Model,
			"APIBase":   m.APIBase,
			"APIKey":    m.APIKey,
			"Proxy":     m.Proxy,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		t.Execute(w, data)
	}
}

func fragmentModelAdd() http.HandlerFunc {
	t := template.Must(template.New("model-add").Parse(modelAddTmpl))

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		t.Execute(w, nil)
	}
}

func modelCreateHandler(cfg *config.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, `<p class="error">Method not allowed</p>`)
			return
		}

		modelName := r.FormValue("model_name")
		model := r.FormValue("model")
		if modelName == "" || model == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `<p class="error">model_name and model are required</p>`)
			return
		}

		configMu.Lock()
		cfg.ModelList = append(cfg.ModelList, config.ModelConfig{
			ModelName: modelName,
			Model:     model,
			APIBase:   r.FormValue("api_base"),
			APIKey:    r.FormValue("api_key"),
			Proxy:     r.FormValue("proxy"),
		})
		configMu.Unlock()

		if configPath != "" {
			if err := saveConfig(configPath, cfg); err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `<p class="error">Failed to save: %s</p>`, template.HTMLEscapeString(err.Error()))
				return
			}
		}

		w.Header().Set("HX-Trigger", "refreshModels, closeModal")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<p class="success">Model added</p>`)
	}
}

func modelUpdateHandler(cfg *config.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, `<p class="error">Method not allowed</p>`)
			return
		}

		idxStr := r.FormValue("idx")
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 0 || idx >= len(cfg.ModelList) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `<p class="error">Invalid model index</p>`)
			return
		}

		configMu.Lock()
		cfg.ModelList[idx].ModelName = r.FormValue("model_name")
		cfg.ModelList[idx].Model = r.FormValue("model")
		cfg.ModelList[idx].APIBase = r.FormValue("api_base")
		cfg.ModelList[idx].APIKey = r.FormValue("api_key")
		cfg.ModelList[idx].Proxy = r.FormValue("proxy")
		configMu.Unlock()

		if configPath != "" {
			if err := saveConfig(configPath, cfg); err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `<p class="error">Failed to save: %s</p>`, template.HTMLEscapeString(err.Error()))
				return
			}
		}

		w.Header().Set("HX-Trigger", "refreshModels, closeModal")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<p class="success">Model updated</p>`)
	}
}

func modelDeleteHandler(cfg *config.Config, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprint(w, `<p class="error">Method not allowed</p>`)
			return
		}

		idxStr := r.FormValue("idx")
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 0 || idx >= len(cfg.ModelList) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `<p class="error">Invalid model index</p>`)
			return
		}

		configMu.Lock()
		cfg.ModelList = append(cfg.ModelList[:idx], cfg.ModelList[idx+1:]...)
		configMu.Unlock()

		if configPath != "" {
			if err := saveConfig(configPath, cfg); err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `<p class="error">Failed to save: %s</p>`, template.HTMLEscapeString(err.Error()))
				return
			}
		}

		w.Header().Set("HX-Trigger", "refreshModels, closeModal")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<p class="success">Model deleted</p>`)
	}
}
