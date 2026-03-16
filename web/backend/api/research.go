package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/research"
)

// registerResearchRoutes binds research task endpoints.
// These access research.db directly (same SQLite file as the Gateway).
func (h *Handler) registerResearchRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/research", h.handleResearch)
	mux.HandleFunc("/api/research/", h.handleResearchDetail)
}

func (h *Handler) openResearchStore() (*research.ResearchStore, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return nil, err
	}
	ws := cfg.WorkspacePath()
	return research.OpenResearchStore(filepath.Join(ws, "research.db"), ws)
}

type researchTaskJSON struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Slug             string `json:"slug"`
	Description      string `json:"description"`
	Status           string `json:"status"`
	OutputDir        string `json:"output_dir"`
	Interval         string `json:"interval"`
	LastResearchedAt string `json:"last_researched_at,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	CompletedAt      string `json:"completed_at,omitempty"`
	DocumentCount    int    `json:"document_count"`
}

type researchDocJSON struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Title     string `json:"title"`
	FilePath  string `json:"file_path"`
	DocType   string `json:"doc_type"`
	Seq       int    `json:"seq"`
	Summary   string `json:"summary"`
	CreatedAt string `json:"created_at"`
}

func taskToJSON(t *research.Task, docCount int) researchTaskJSON {
	r := researchTaskJSON{
		ID:            t.ID,
		Title:         t.Title,
		Slug:          t.Slug,
		Description:   t.Description,
		Status:        string(t.Status),
		OutputDir:     t.OutputDir,
		Interval:      t.Interval,
		CreatedAt:     t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     t.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		DocumentCount: docCount,
	}
	if !t.LastResearchedAt.IsZero() {
		r.LastResearchedAt = t.LastResearchedAt.Format("2006-01-02T15:04:05Z")
	}
	if !t.CompletedAt.IsZero() {
		r.CompletedAt = t.CompletedAt.Format("2006-01-02T15:04:05Z")
	}
	return r
}

func writeJSONRes(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// handleResearch handles GET (list) and POST (create) on /api/research.
func (h *Handler) handleResearch(w http.ResponseWriter, r *http.Request) {
	store, err := h.openResearchStore()
	if err != nil {
		http.Error(w, `{"error":"failed to open research store"}`, http.StatusInternalServerError)
		return
	}
	defer store.Close()

	switch r.Method {
	case http.MethodGet:
		statusFilter := r.URL.Query().Get("status")
		tasks, err := store.ListTasks(research.TaskStatus(statusFilter))
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		result := make([]researchTaskJSON, 0, len(tasks))
		for _, t := range tasks {
			dc, _ := store.DocumentCount(t.ID)
			result = append(result, taskToJSON(t, dc))
		}
		writeJSONRes(w, result)

	case http.MethodPost:
		var req struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Interval    string `json:"interval"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Title) == "" {
			http.Error(w, `{"error":"title is required"}`, http.StatusBadRequest)
			return
		}
		task, err := store.CreateTask(
			strings.TrimSpace(req.Title),
			strings.TrimSpace(req.Description),
			strings.TrimSpace(req.Interval),
		)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSONRes(w, taskToJSON(task, 0))

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// handleResearchDetail handles /api/research/{id} and /api/research/{id}/doc/{docId}.
func (h *Handler) handleResearchDetail(w http.ResponseWriter, r *http.Request) {
	store, err := h.openResearchStore()
	if err != nil {
		http.Error(w, `{"error":"failed to open research store"}`, http.StatusInternalServerError)
		return
	}
	defer store.Close()

	path := strings.TrimPrefix(r.URL.Path, "/api/research/")
	parts := strings.Split(path, "/")

	if len(parts) == 1 && parts[0] != "" {
		taskID := parts[0]
		switch r.Method {
		case http.MethodGet:
			h.researchGetTask(w, store, taskID)
		case http.MethodPost:
			h.researchTaskAction(w, r, store, taskID)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) == 3 && parts[1] == "doc" && parts[2] != "" {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		h.researchGetDoc(w, store, parts[0], parts[2])
		return
	}

	http.NotFound(w, r)
}

func (h *Handler) researchGetTask(w http.ResponseWriter, store *research.ResearchStore, taskID string) {
	task, err := store.GetTask(taskID)
	if err != nil {
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}
	docs, _ := store.ListDocuments(taskID)
	docList := make([]researchDocJSON, 0, len(docs))
	for _, d := range docs {
		docList = append(docList, researchDocJSON{
			ID:        d.ID,
			TaskID:    d.TaskID,
			Title:     d.Title,
			FilePath:  d.FilePath,
			DocType:   d.DocType,
			Seq:       d.Seq,
			Summary:   d.Summary,
			CreatedAt: d.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	writeJSONRes(w, struct {
		researchTaskJSON
		Documents []researchDocJSON `json:"documents"`
	}{
		researchTaskJSON: taskToJSON(task, len(docs)),
		Documents:        docList,
	})
}

func (h *Handler) researchTaskAction(
	w http.ResponseWriter, r *http.Request,
	store *research.ResearchStore, taskID string,
) {
	var req struct {
		Action      string `json:"action"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	switch req.Action {
	case "cancel":
		if err := store.SetTaskStatus(taskID, research.StatusCanceled); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
	case "reopen":
		if err := store.SetTaskStatus(taskID, research.StatusPending); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
	case "update":
		task, err := store.GetTask(taskID)
		if err != nil {
			http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
			return
		}
		title := req.Title
		if title == "" {
			title = task.Title
		}
		desc := req.Description
		if desc == "" {
			desc = task.Description
		}
		if err := store.UpdateTask(taskID, title, desc); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, `{"error":"unknown action"}`, http.StatusBadRequest)
		return
	}
	h.researchGetTask(w, store, taskID)
}

func (h *Handler) researchGetDoc(w http.ResponseWriter, store *research.ResearchStore, taskID, docID string) {
	docs, err := store.ListDocuments(taskID)
	if err != nil {
		http.Error(w, `{"error":"failed to list documents"}`, http.StatusInternalServerError)
		return
	}
	var found *research.Document
	for _, d := range docs {
		if d.ID == docID {
			found = d
			break
		}
	}
	if found == nil {
		http.Error(w, `{"error":"document not found"}`, http.StatusNotFound)
		return
	}

	absPath := found.FilePath
	if !filepath.IsAbs(absPath) {
		cfg, cfgErr := config.LoadConfig(h.configPath)
		if cfgErr == nil {
			absPath = filepath.Join(cfg.WorkspacePath(), absPath)
		}
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		http.Error(w, `{"error":"failed to read document"}`, http.StatusInternalServerError)
		return
	}
	writeJSONRes(w, map[string]any{
		"id":       found.ID,
		"title":    found.Title,
		"doc_type": found.DocType,
		"content":  string(content),
	})
}
