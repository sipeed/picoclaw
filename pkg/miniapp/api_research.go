package miniapp

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/research"
)

// SetResearchStore injects the research store into the handler.
func (h *Handler) SetResearchStore(rs *research.ResearchStore) {
	h.researchStore = rs
}

// SetResearchFocus injects the shared focus tracker for recall/forget control.
func (h *Handler) SetResearchFocus(ft *research.FocusTracker) {
	h.researchFocus = ft
}

// apiResearch handles GET /miniapp/api/research (list) and POST (create).
func (h *Handler) apiResearch(w http.ResponseWriter, r *http.Request) {
	if h.researchStore == nil {
		http.Error(w, `{"error":"research store not available"}`, http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.apiResearchList(w, r)
	case http.MethodPost:
		h.apiResearchCreate(w, r)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

type researchTaskResponse struct {
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
	Focused          bool   `json:"focused"`
}

func taskToResponse(t *research.Task, docCount int) researchTaskResponse {
	resp := researchTaskResponse{
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
		resp.LastResearchedAt = t.LastResearchedAt.Format("2006-01-02T15:04:05Z")
	}
	if !t.CompletedAt.IsZero() {
		resp.CompletedAt = t.CompletedAt.Format("2006-01-02T15:04:05Z")
	}
	return resp
}

func (h *Handler) apiResearchList(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")
	tasks, err := h.researchStore.ListTasks(research.TaskStatus(statusFilter))
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	result := make([]researchTaskResponse, 0, len(tasks))
	for _, t := range tasks {
		docCount, _ := h.researchStore.DocumentCount(t.ID)
		resp := taskToResponse(t, docCount)
		if h.researchFocus != nil {
			resp.Focused = h.researchFocus.IsFocused(t.ID)
		}
		result = append(result, resp)
	}
	writeJSON(w, result)
}

func (h *Handler) apiResearchCreate(w http.ResponseWriter, r *http.Request) {
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

	task, err := h.researchStore.CreateTask(
		strings.TrimSpace(req.Title),
		strings.TrimSpace(req.Description),
		strings.TrimSpace(req.Interval),
	)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, taskToResponse(task, 0))
}

// apiResearchDetail handles /miniapp/api/research/{id} and /miniapp/api/research/{id}/doc/{docId}.
func (h *Handler) apiResearchDetail(w http.ResponseWriter, r *http.Request) {
	if h.researchStore == nil {
		http.Error(w, `{"error":"research store not available"}`, http.StatusServiceUnavailable)
		return
	}

	// Parse path: /miniapp/api/research/{id} or /miniapp/api/research/{id}/doc/{docId}
	path := strings.TrimPrefix(r.URL.Path, "/miniapp/api/research/")
	parts := strings.Split(path, "/")

	if len(parts) == 1 && parts[0] != "" {
		// /miniapp/api/research/{id}
		taskID := parts[0]
		switch r.Method {
		case http.MethodGet:
			h.apiResearchGetTask(w, taskID)
		case http.MethodPost:
			h.apiResearchTaskAction(w, r, taskID)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) == 3 && parts[1] == "doc" && parts[2] != "" {
		// /miniapp/api/research/{id}/doc/{docId}
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		h.apiResearchGetDoc(w, parts[0], parts[2])
		return
	}

	http.NotFound(w, r)
}

type researchDocResponse struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Title     string `json:"title"`
	FilePath  string `json:"file_path"`
	DocType   string `json:"doc_type"`
	Seq       int    `json:"seq"`
	Summary   string `json:"summary"`
	CreatedAt string `json:"created_at"`
}

type researchTaskDetailResponse struct {
	researchTaskResponse
	Documents []researchDocResponse `json:"documents"`
}

func (h *Handler) apiResearchGetTask(w http.ResponseWriter, taskID string) {
	task, err := h.researchStore.GetTask(taskID)
	if err != nil {
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}

	docs, _ := h.researchStore.ListDocuments(taskID)
	docResponses := make([]researchDocResponse, 0, len(docs))
	for _, d := range docs {
		docResponses = append(docResponses, researchDocResponse{
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

	resp := taskToResponse(task, len(docs))
	if h.researchFocus != nil {
		resp.Focused = h.researchFocus.IsFocused(taskID)
	}

	writeJSON(w, researchTaskDetailResponse{
		researchTaskResponse: resp,
		Documents:            docResponses,
	})
}

func (h *Handler) apiResearchTaskAction(w http.ResponseWriter, r *http.Request, taskID string) {
	var req struct {
		Action      string `json:"action"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Interval    string `json:"interval"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	switch req.Action {
	case "cancel":
		if err := h.researchStore.SetTaskStatus(taskID, research.StatusCanceled); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
	case "reopen":
		if err := h.researchStore.SetTaskStatus(taskID, research.StatusPending); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
	case "activate":
		if err := h.researchStore.SetTaskStatus(taskID, research.StatusActive); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
	case "complete":
		if err := h.researchStore.SetTaskStatus(taskID, research.StatusCompleted); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
	case "update":
		task, err := h.researchStore.GetTask(taskID)
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
		if err := h.researchStore.UpdateTask(taskID, title, desc); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
	case "set_interval":
		if req.Interval == "" {
			http.Error(w, `{"error":"interval is required"}`, http.StatusBadRequest)
			return
		}
		if err := h.researchStore.SetInterval(taskID, req.Interval); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, `{"error":"unknown action"}`, http.StatusBadRequest)
		return
	}

	// Return updated task
	h.apiResearchGetTask(w, taskID)
}

func (h *Handler) apiResearchGetDoc(w http.ResponseWriter, taskID, docID string) {
	docs, err := h.researchStore.ListDocuments(taskID)
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

	// Read file content - use workspace-relative path
	absPath := found.FilePath
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(h.workspace, absPath)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		http.Error(w, `{"error":"failed to read document"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"id":       found.ID,
		"title":    found.Title,
		"doc_type": found.DocType,
		"content":  string(content),
	})
}

// apiResearchFocus handles GET/POST /miniapp/api/research/focus.
// GET returns the current focus state; POST sets focus/unfocus for a task.
func (h *Handler) apiResearchFocus(w http.ResponseWriter, r *http.Request) {
	if h.researchFocus == nil {
		http.Error(w, `{"error":"research focus not available"}`, http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.apiResearchFocusGet(w)
	case http.MethodPost:
		h.apiResearchFocusSet(w, r)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (h *Handler) apiResearchFocusGet(w http.ResponseWriter) {
	taskID, title := h.researchFocus.Current()
	writeJSON(w, map[string]any{
		"focused_id":    taskID,
		"focused_title": title,
	})
}

func (h *Handler) apiResearchFocusSet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"` // "recall" or "forget"
		TaskID string `json:"task_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<14)).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	switch req.Action {
	case "recall":
		if req.TaskID == "" {
			http.Error(w, `{"error":"task_id is required"}`, http.StatusBadRequest)
			return
		}
		task, err := h.researchStore.GetTask(req.TaskID)
		if err != nil {
			http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
			return
		}
		h.researchFocus.Focus(task.ID, task.Title)
	case "forget":
		if req.TaskID == "" {
			h.researchFocus.UnfocusAll()
		} else {
			h.researchFocus.Unfocus(req.TaskID)
		}
	default:
		http.Error(w, `{"error":"action must be recall or forget"}`, http.StatusBadRequest)
		return
	}

	h.apiResearchFocusGet(w)
}
