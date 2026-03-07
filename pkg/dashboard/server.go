// Package dashboard provides a web-based management UI for PicoClaw.
package dashboard

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/cron"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type Server struct {
	cfg         *config.Config
	cronService *cron.CronService
	startTime   time.Time
	workDir     string
	configPath  string
}

func NewServer(cfg *config.Config, cronService *cron.CronService, configPath string) *Server {
	return &Server{
		cfg:         cfg,
		cronService: cronService,
		startTime:   time.Now(),
		workDir:     cfg.WorkspacePath(),
		configPath:  configPath,
	}
}

func (s *Server) RegisterOnMux(mux *http.ServeMux) {
	mux.HandleFunc("/dashboard", s.handleUI)
	mux.HandleFunc("/dashboard/", s.handleUI)
	mux.HandleFunc("/api/status", s.withCORS(s.handleStatus))
	mux.HandleFunc("/api/config", s.withCORS(s.handleConfig))
	mux.HandleFunc("/api/agents", s.withCORS(s.handleAgents))
	mux.HandleFunc("/api/skills", s.withCORS(s.handleSkills))
	mux.HandleFunc("/api/cron", s.withCORS(s.handleCron))
	mux.HandleFunc("/api/sessions", s.withCORS(s.handleSessions))
	mux.HandleFunc("/api/logs", s.withCORS(s.handleLogs))
	mux.HandleFunc("/api/soul", s.withCORS(s.handleSoul))
	mux.HandleFunc("/api/skill", s.withCORS(s.handleSkillFile))
	logger.InfoCF("dashboard", "Dashboard registered", map[string]any{"path": "/dashboard"})
}

func (s *Server) withCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(dashboardHTML))
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"uptime":    time.Since(s.startTime).String(),
		"workspace": s.workDir,
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data, err := os.ReadFile(s.configPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "cannot read config: "+err.Error())
			return
		}
		var raw any
		if err := json.Unmarshal(data, &raw); err != nil {
			writeError(w, http.StatusInternalServerError, "invalid config JSON: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, raw)
	case http.MethodPut:
		var raw any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		data, err := json.MarshalIndent(raw, "", "  ")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "marshal error: "+err.Error())
			return
		}
		if err := os.WriteFile(s.configPath, data, 0o600); err != nil {
			writeError(w, http.StatusInternalServerError, "cannot write config: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

type agentInfo struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Default   bool     `json:"default"`
	Skills    []string `json:"skills"`
	HasSoul   bool     `json:"has_soul"`
	ModelName string   `json:"model_name"`
	Workspace string   `json:"workspace"`
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	result := make([]agentInfo, 0)
	soulPath := filepath.Join(s.workDir, "SOUL.md")
	_, hasSoul := os.Stat(soulPath)

	// PicoClaw uses a single default agent — expose it as the main agent
	defaults := s.cfg.Agents.Defaults
	skillsDir := filepath.Join(s.workDir, "skills")
	skills := []string{}
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				skills = append(skills, e.Name())
			}
		}
	}
	result = append(result, agentInfo{
		ID:        "main",
		Name:      "Default Agent",
		Default:   true,
		Skills:    skills,
		HasSoul:   hasSoul == nil,
		ModelName: defaults.ModelName,
		Workspace: defaults.Workspace,
	})

	// Also add any named agents from the list if present
	for _, a := range s.cfg.Agents.List {
		result = append(result, agentInfo{
			ID:      a.ID,
			Name:    a.Name,
			Default: a.Default,
			Skills:  a.Skills,
			HasSoul: hasSoul == nil,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

type skillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	HasMeta     bool   `json:"has_meta"`
}

func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	skillsDir := filepath.Join(s.workDir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		writeJSON(w, http.StatusOK, []skillInfo{})
		return
	}
	result := make([]skillInfo, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillPath := filepath.Join(skillsDir, e.Name())
		desc := ""
		if data, err := os.ReadFile(filepath.Join(skillPath, "SKILL.md")); err == nil {
			for _, l := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(l, "description:") {
					desc = strings.Trim(strings.TrimSpace(strings.TrimPrefix(l, "description:")), `"`)
					break
				}
			}
		}
		_, hasMeta := os.Stat(filepath.Join(skillPath, "_meta.json"))
		result = append(result, skillInfo{
			Name: e.Name(), Description: desc, Path: skillPath, HasMeta: hasMeta == nil,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleCron(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.cronService.ListJobs(true))
	case http.MethodPost:
		var p struct {
			Name     string            `json:"name"`
			Schedule cron.CronSchedule `json:"schedule"`
			Message  string            `json:"message"`
			Command  string            `json:"command"`
			Deliver  bool              `json:"deliver"`
			Channel  string            `json:"channel"`
			To       string            `json:"to"`
		}
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		job, err := s.cronService.AddJob(p.Name, p.Schedule, p.Message, p.Deliver, p.Channel, p.To)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, job)
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "missing id")
			return
		}
		if !s.cronService.RemoveJob(id) {
			writeError(w, http.StatusNotFound, "job not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	case http.MethodPut:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "missing id")
			return
		}
		job := s.cronService.EnableJob(id, r.URL.Query().Get("enabled") == "true")
		if job == nil {
			writeError(w, http.StatusNotFound, "job not found")
			return
		}
		writeJSON(w, http.StatusOK, job)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	type sessionMeta struct {
		Key      string    `json:"key"`
		File     string    `json:"file"`
		Modified time.Time `json:"modified"`
		Size     int64     `json:"size_bytes"`
	}
	entries, err := os.ReadDir(filepath.Join(s.workDir, "sessions"))
	if err != nil {
		writeJSON(w, http.StatusOK, []sessionMeta{})
		return
	}
	result := make([]sessionMeta, 0)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, _ := e.Info()
		key := strings.ReplaceAll(strings.TrimSuffix(e.Name(), ".json"), "_", ":")
		result = append(result, sessionMeta{Key: key, File: e.Name(), Modified: info.ModTime(), Size: info.Size()})
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(filepath.Join(s.workDir, "logs"))
	if err != nil {
		writeJSON(w, http.StatusOK, []string{})
		return
	}
	var latest string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".log") {
			latest = filepath.Join(s.workDir, "logs", e.Name())
		}
	}
	if latest == "" {
		writeJSON(w, http.StatusOK, []string{})
		return
	}
	data, err := os.ReadFile(latest)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > 200 {
		lines = lines[len(lines)-200:]
	}
	writeJSON(w, http.StatusOK, lines)
}

// ── /api/soul ─────────────────────────────────────────────────────────────────
// GET  /api/soul         → returns SOUL.md content
// PUT  /api/soul         → writes SOUL.md content
func (s *Server) handleSoul(w http.ResponseWriter, r *http.Request) {
	soulPath := filepath.Join(s.workDir, "SOUL.md")
	switch r.Method {
	case http.MethodGet:
		data, err := os.ReadFile(soulPath)
		if err != nil {
			// Return empty string if not found yet
			writeJSON(w, http.StatusOK, map[string]string{"content": ""})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"content": string(data)})
	case http.MethodPut:
		var body struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if err := os.WriteFile(soulPath, []byte(body.Content), 0o644); err != nil {
			writeError(w, http.StatusInternalServerError, "cannot write SOUL.md: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ── /api/skill ────────────────────────────────────────────────────────────────
// GET  /api/skill?name=X         → returns SKILL.md content for skill X
// PUT  /api/skill?name=X         → writes SKILL.md for skill X
// POST /api/skill?name=X         → creates new skill folder + SKILL.md
// DELETE /api/skill?name=X       → deletes skill folder
func (s *Server) handleSkillFile(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "missing name parameter")
		return
	}
	// Sanitize name — no path traversal
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "")
	skillDir := filepath.Join(s.workDir, "skills", name)
	skillMd := filepath.Join(skillDir, "SKILL.md")

	switch r.Method {
	case http.MethodGet:
		data, err := os.ReadFile(skillMd)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]string{"content": ""})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"content": string(data)})

	case http.MethodPut:
		var body struct{ Content string `json:"content"` }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := os.WriteFile(skillMd, []byte(body.Content), 0o644); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})

	case http.MethodPost:
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		var body struct{ Content string `json:"content"` }
		_ = json.NewDecoder(r.Body).Decode(&body)
		content := body.Content
		if content == "" {
			content = "# " + name + "\n\ndescription: \"Describe what this skill does\"\n\n## Instructions\n\nWrite your skill instructions here.\n"
		}
		if err := os.WriteFile(skillMd, []byte(content), 0o644); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})

	case http.MethodDelete:
		if err := os.RemoveAll(skillDir); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
