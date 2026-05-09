package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/cron"
)

// registerCronRoutes registers cron job API routes on the ServeMux.
func (h *Handler) registerCronRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/cron/jobs", h.handleListCronJobs)
	mux.HandleFunc("POST /api/cron/jobs", h.handleAddCronJob)
	mux.HandleFunc("DELETE /api/cron/jobs/{id}", h.handleDeleteCronJob)
	mux.HandleFunc("POST /api/cron/jobs/{id}/enable", h.handleEnableCronJob)
	mux.HandleFunc("POST /api/cron/jobs/{id}/disable", h.handleDisableCronJob)
}

type cronJobResponse struct {
	Jobs []cron.CronJob `json:"jobs"`
}

type cronAddRequest struct {
	Name       string           `json:"name" binding:"required"`
	EveryMS    *int64           `json:"every_ms,omitempty"`
	CronExpr   string           `json:"cron_expr,omitempty"`
	Message    string           `json:"message" binding:"required"`
	Channel    string           `json:"channel,omitempty"`
	To         string           `json:"to,omitempty"`
}

// handleListCronJobs lists all cron jobs from the cron service.
func (h *Handler) handleListCronJobs(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	storePath := cfg.WorkspacePath() + "/cron/jobs.json"
	cs := cron.NewCronService(storePath, nil)
	if err := cs.Load(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to load cron store: %v", err), http.StatusInternalServerError)
		return
	}
	jobs := cs.ListJobs(true)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cronJobResponse{Jobs: jobs})
}

// handleAddCronJob adds a new cron job.
func (h *Handler) handleAddCronJob(w http.ResponseWriter, r *http.Request) {
	var req cronAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Message == "" {
		http.Error(w, "name and message are required", http.StatusBadRequest)
		return
	}
	var schedule cron.CronSchedule
	if req.EveryMS != nil {
		schedule = cron.CronSchedule{Kind: "every", EveryMS: req.EveryMS}
	} else if req.CronExpr != "" {
		schedule = cron.CronSchedule{Kind: "cron", Expr: req.CronExpr}
	} else {
		http.Error(w, "either every_ms or cron_expr must be specified", http.StatusBadRequest)
		return
	}
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	storePath := cfg.WorkspacePath() + "/cron/jobs.json"
	cs := cron.NewCronService(storePath, nil)
	job, err := cs.AddJob(req.Name, schedule, req.Message, req.Channel, req.To)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to add job: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"job": job, "status": "ok"})
}

// handleDeleteCronJob deletes a cron job by ID.
func (h *Handler) handleDeleteCronJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "job id is required", http.StatusBadRequest)
		return
	}
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	storePath := cfg.WorkspacePath() + "/cron/jobs.json"
	cs := cron.NewCronService(storePath, nil)
	if removed := cs.RemoveJob(id); !removed {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleEnableCronJob enables a cron job by ID.
func (h *Handler) handleEnableCronJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "job id is required", http.StatusBadRequest)
		return
	}
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	storePath := cfg.WorkspacePath() + "/cron/jobs.json"
	cs := cron.NewCronService(storePath, nil)
	if job := cs.EnableJob(id, true); job == nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleDisableCronJob disables a cron job by ID.
func (h *Handler) handleDisableCronJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "job id is required", http.StatusBadRequest)
		return
	}
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	storePath := cfg.WorkspacePath() + "/cron/jobs.json"
	cs := cron.NewCronService(storePath, nil)
	if job := cs.EnableJob(id, false); job == nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}