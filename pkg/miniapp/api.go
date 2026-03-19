package miniapp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/git"
)

func (h *Handler) apiSkills(w http.ResponseWriter, r *http.Request) {
	skillsList := h.provider.ListSkills()
	writeJSON(w, skillsList)
}

func (h *Handler) apiPlan(w http.ResponseWriter, r *http.Request) {
	info := h.provider.GetPlanInfo()
	writeJSON(w, info)
}

func (h *Handler) apiSessions(w http.ResponseWriter, r *http.Request) {
	sessions := h.provider.GetActiveSessions()
	if sessions == nil {
		sessions = []SessionInfo{}
	}
	writeJSON(w, sessions)
}

func (h *Handler) apiSession(w http.ResponseWriter, r *http.Request) {
	s := h.provider.GetSessionStats()
	if s == nil {
		writeJSON(w, map[string]string{"status": "stats not enabled"})
		return
	}
	writeJSON(w, s)
}

func (h *Handler) apiContext(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.provider.GetContextInfo())
}

func (h *Handler) apiPrompt(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"prompt": h.provider.GetSystemPrompt()})
}

func (h *Handler) apiGit(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	if repo == "" {
		writeJSON(w, h.provider.GetGitRepos())
	} else {
		writeJSON(w, h.provider.GetGitRepoDetail(repo))
	}
}

func (h *Handler) apiWorktrees(w http.ResponseWriter, r *http.Request) {
	repoRoot := git.FindRepoRoot(h.workspace)
	if repoRoot == "" {
		http.Error(w, `{"error":"workspace is not a git repository"}`, http.StatusBadRequest)
		return
	}
	worktreesDir := filepath.Join(h.workspace, ".worktrees")

	switch r.Method {
	case http.MethodGet:
		items, err := git.ListManagedWorktrees(repoRoot, worktreesDir)
		if err != nil {
			http.Error(w, `{"error":"failed to list worktrees"}`, http.StatusInternalServerError)
			return
		}
		writeJSON(w, items)

	case http.MethodPost:
		body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
		if err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}

		var req struct {
			Action     string `json:"action"`
			Name       string `json:"name"`
			Force      bool   `json:"force"`
			BaseBranch string `json:"base_branch"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}

		req.Action = strings.ToLower(strings.TrimSpace(req.Action))
		req.Name = strings.TrimSpace(req.Name)
		req.BaseBranch = strings.TrimSpace(req.BaseBranch)
		if req.Action == "" || req.Name == "" {
			http.Error(w, `{"error":"action and name are required"}`, http.StatusBadRequest)
			return
		}

		switch req.Action {
		case "merge":
			res, baseBranch, err := git.MergeManagedWorktree(repoRoot, worktreesDir, req.Name, req.BaseBranch)
			if err != nil {
				if writeWorktreeAPIError(w, err) {
					return
				}
				http.Error(w, `{"error":"merge failed"}`, http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]any{
				"status":      "ok",
				"action":      "merge",
				"name":        req.Name,
				"base_branch": baseBranch,
				"result":      res,
			})

		case "dispose":
			wt, err := git.GetManagedWorktree(repoRoot, worktreesDir, req.Name)
			if err != nil {
				if writeWorktreeAPIError(w, err) {
					return
				}
				http.Error(w, `{"error":"failed to inspect worktree"}`, http.StatusInternalServerError)
				return
			}
			if wt.HasUncommitted && !req.Force {
				http.Error(
					w,
					`{"error":"worktree has uncommitted changes; retry with force=true"}`,
					http.StatusConflict,
				)
				return
			}

			res, err := git.DisposeManagedWorktree(repoRoot, worktreesDir, req.Name, req.BaseBranch)
			if err != nil {
				if writeWorktreeAPIError(w, err) {
					return
				}
				http.Error(w, `{"error":"dispose failed"}`, http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]any{
				"status": "ok",
				"action": "dispose",
				"name":   req.Name,
				"result": res,
			})

		default:
			http.Error(w, `{"error":"unknown action"}`, http.StatusBadRequest)
		}

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (h *Handler) apiSessionGraph(w http.ResponseWriter, r *http.Request) {
	graph := h.provider.GetSessionGraph()
	if graph == nil {
		graph = &SessionGraphData{
			Nodes: []SessionGraphNode{},
			Edges: []SessionGraphEdge{},
		}
	}
	writeJSON(w, graph)
}

func (h *Handler) apiCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
	if err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(body, &req); err != nil || req.Command == "" {
		http.Error(w, `{"error":"missing command"}`, http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(req.Command, "/") {
		http.Error(w, `{"error":"command must start with /"}`, http.StatusBadRequest)
		return
	}

	// Extract user ID from initData to identify the sender
	initData := r.URL.Query().Get("initData")
	userID, chatID := extractUserFromInitData(initData)
	if userID == "" {
		http.Error(w, `{"error":"cannot identify user"}`, http.StatusBadRequest)
		return
	}

	h.sender.SendCommand(userID, chatID, req.Command)
	writeJSON(w, map[string]string{"status": "ok"})
}

func (h *Handler) apiEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := h.notifier.Subscribe()
	defer h.notifier.Unsubscribe(ch)

	var lastPlan, lastSession, lastSkills, lastDev, lastContext, lastPrompt []byte

	// Send initial state immediately
	sendSSEIfChanged(w, flusher, "plan", h.provider.GetPlanInfo(), &lastPlan)
	sendSSEIfChanged(
		w,
		flusher,
		"session",
		map[string]any{
			"stats":    h.provider.GetSessionStats(),
			"sessions": h.provider.GetActiveSessions(),
			"graph":    h.provider.GetSessionGraph(),
		},
		&lastSession,
	)
	sendSSEIfChanged(w, flusher, "skills", h.provider.ListSkills(), &lastSkills)
	sendSSEIfChanged(w, flusher, "dev", h.devStatus(), &lastDev)
	sendSSEIfChanged(w, flusher, "context", h.provider.GetContextInfo(), &lastContext)
	sendSSEIfChanged(w, flusher, "prompt", map[string]string{"prompt": h.provider.GetSystemPrompt()}, &lastPrompt)

	for {
		select {
		case <-r.Context().Done():
			return
		case <-h.notifier.Done():
			return
		case <-ch:
			sendSSEIfChanged(w, flusher, "plan", h.provider.GetPlanInfo(), &lastPlan)
			sendSSEIfChanged(
				w,
				flusher,
				"session",
				map[string]any{
					"stats":    h.provider.GetSessionStats(),
					"sessions": h.provider.GetActiveSessions(),
					"graph":    h.provider.GetSessionGraph(),
				},
				&lastSession,
			)
			sendSSEIfChanged(w, flusher, "skills", h.provider.ListSkills(), &lastSkills)
			sendSSEIfChanged(w, flusher, "dev", h.devStatus(), &lastDev)
			sendSSEIfChanged(w, flusher, "context", h.provider.GetContextInfo(), &lastContext)
			sendSSEIfChanged(
				w,
				flusher,
				"prompt",
				map[string]string{"prompt": h.provider.GetSystemPrompt()},
				&lastPrompt,
			)
		}
	}
}

func sendSSEIfChanged(w http.ResponseWriter, f http.Flusher, event string, v any, last *[]byte) {
	data, _ := json.Marshal(v)
	if !bytes.Equal(data, *last) {
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
		f.Flush()
		*last = data
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeWorktreeAPIError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, git.ErrInvalidWorktreeName):
		http.Error(w, `{"error":"invalid worktree name"}`, http.StatusBadRequest)
		return true
	case errors.Is(err, git.ErrWorktreeNotFound):
		http.Error(w, `{"error":"worktree not found"}`, http.StatusNotFound)
		return true
	default:
		return false
	}
}

// apiDevConsole receives console output from dev preview iframes.

// apiCache returns a list of media cache entries.
func (h *Handler) apiCache(w http.ResponseWriter, r *http.Request) {
	entryType := r.URL.Query().Get("type")
	entries := h.provider.ListMediaCache(entryType)
	if entries == nil {
		entries = []MediaCacheEntry{}
	}
	writeJSON(w, entries)
}
