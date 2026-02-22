package miniapp

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/stats"
)

//go:embed static/index.html
var staticFS embed.FS

// PlanPhase mirrors agent.PlanPhase for JSON serialization.
type PlanPhase struct {
	Number int        `json:"number"`
	Title  string     `json:"title"`
	Steps  []PlanStep `json:"steps"`
}

// PlanStep mirrors agent.PlanStep for JSON serialization.
type PlanStep struct {
	Index       int    `json:"index"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
}

// PlanInfo represents the plan state exposed via the API.
type PlanInfo struct {
	HasPlan      bool        `json:"has_plan"`
	Status       string      `json:"status"`
	CurrentPhase int         `json:"current_phase"`
	TotalPhases  int         `json:"total_phases"`
	Display      string      `json:"display"`
	Phases       []PlanPhase `json:"phases"`
	Memory       string      `json:"memory"`
}

// SessionInfo represents an active session entry for the API response.
type SessionInfo struct {
	SessionKey string `json:"session_key"`
	Channel    string `json:"channel"`
	ChatID     string `json:"chat_id"`
	TouchDir   string `json:"touch_dir"`
	LastSeenAt string `json:"last_seen_at"`
	AgeSec     int    `json:"age_sec"`
}

// GitRepoSummary represents a lightweight repo entry for the list view.
type GitRepoSummary struct {
	Name   string `json:"name"`
	Branch string `json:"branch"`
}

// GitInfo represents the git repository state exposed via the API.
type GitInfo struct {
	Name     string      `json:"name"`
	Branch   string      `json:"branch"`
	Commits  []GitCommit `json:"commits"`
	Modified []GitChange `json:"modified"`
}

// GitCommit represents a single commit entry.
type GitCommit struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

// GitChange represents a modified/untracked file entry.
type GitChange struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

// DataProvider is the read-only interface to agent state for the Mini App API.
type DataProvider interface {
	ListSkills() []skills.SkillInfo
	GetPlanInfo() PlanInfo
	GetSessionStats() *stats.Stats
	GetActiveSessions() []SessionInfo
	GetGitRepos() []GitRepoSummary
	GetGitRepoDetail(name string) GitInfo
}

// CommandSender injects a command into the message bus on behalf of a user.
type CommandSender interface {
	SendCommand(senderID, chatID, command string)
}

// StateNotifier broadcasts state-change signals to SSE subscribers.
type StateNotifier struct {
	mu   sync.Mutex
	subs map[chan struct{}]struct{}
	done chan struct{}
}

// NewStateNotifier creates a new StateNotifier.
func NewStateNotifier() *StateNotifier {
	return &StateNotifier{
		subs: make(map[chan struct{}]struct{}),
		done: make(chan struct{}),
	}
}

// Subscribe returns a channel that receives a signal on each state change.
func (n *StateNotifier) Subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	n.mu.Lock()
	n.subs[ch] = struct{}{}
	n.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel.
func (n *StateNotifier) Unsubscribe(ch chan struct{}) {
	n.mu.Lock()
	delete(n.subs, ch)
	n.mu.Unlock()
}

// Close signals all SSE handlers to exit.
func (n *StateNotifier) Close() {
	select {
	case <-n.done:
	default:
		close(n.done)
	}
}

// Done returns a channel that is closed when the notifier is shut down.
func (n *StateNotifier) Done() <-chan struct{} {
	return n.done
}

// Notify sends a signal to all subscribers, coalescing rapid notifications.
func (n *StateNotifier) Notify() {
	n.mu.Lock()
	defer n.mu.Unlock()
	for ch := range n.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// DevTargetSetter allows tools to control the dev proxy target.
type DevTargetSetter interface {
	SetDevTarget(target string) error
	GetDevTarget() string
}

// Handler serves the Mini App HTML and API endpoints.
type Handler struct {
	provider DataProvider
	sender   CommandSender
	botToken string
	notifier *StateNotifier

	devMu     sync.RWMutex
	devTarget *url.URL
	devProxy  *httputil.ReverseProxy
}

// NewHandler creates a new Mini App handler.
func NewHandler(provider DataProvider, sender CommandSender, botToken string, notifier *StateNotifier) *Handler {
	return &Handler{
		provider: provider,
		sender:   sender,
		botToken: botToken,
		notifier: notifier,
	}
}

// SetDevTarget sets the reverse proxy target URL. Only localhost targets are allowed.
// Pass an empty string to disable the proxy.
func (h *Handler) SetDevTarget(target string) error {
	h.devMu.Lock()
	defer h.devMu.Unlock()

	if target == "" {
		h.devTarget = nil
		h.devProxy = nil
		if h.notifier != nil {
			h.notifier.Notify()
		}
		return nil
	}

	u, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	host := u.Hostname()
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return fmt.Errorf("only localhost targets are allowed, got %q", host)
	}

	h.devTarget = u
	h.devProxy = httputil.NewSingleHostReverseProxy(u)
	if h.notifier != nil {
		h.notifier.Notify()
	}
	return nil
}

// GetDevTarget returns the current dev proxy target URL, or empty string if disabled.
func (h *Handler) GetDevTarget() string {
	h.devMu.RLock()
	defer h.devMu.RUnlock()
	if h.devTarget == nil {
		return ""
	}
	return h.devTarget.String()
}

// RegisterRoutes registers Mini App routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/miniapp", h.serveIndex)
	mux.HandleFunc("/miniapp/api/skills", h.requireAuth(h.apiSkills))
	mux.HandleFunc("/miniapp/api/plan", h.requireAuth(h.apiPlan))
	mux.HandleFunc("/miniapp/api/session", h.requireAuth(h.apiSession))
	mux.HandleFunc("/miniapp/api/sessions", h.requireAuth(h.apiSessions))
	mux.HandleFunc("/miniapp/api/command", h.requireAuth(h.apiCommand))
	mux.HandleFunc("/miniapp/api/git", h.requireAuth(h.apiGit))
	mux.HandleFunc("/miniapp/api/dev", h.requireAuth(h.apiDev))
	mux.HandleFunc("/miniapp/api/events", h.requireAuth(h.apiEvents))
	mux.HandleFunc("/miniapp/dev/", h.serveDevProxy)
}

func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		initData := r.URL.Query().Get("initData")
		if initData == "" {
			http.Error(w, `{"error":"missing initData"}`, http.StatusUnauthorized)
			return
		}
		if !ValidateInitData(initData, h.botToken) {
			http.Error(w, `{"error":"invalid initData"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

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

func (h *Handler) apiGit(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	if repo == "" {
		writeJSON(w, h.provider.GetGitRepos())
	} else {
		writeJSON(w, h.provider.GetGitRepoDetail(repo))
	}
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

func (h *Handler) apiDev(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		target := h.GetDevTarget()
		writeJSON(w, map[string]any{
			"active": target != "",
			"target": target,
		})
	case http.MethodPost:
		body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
		if err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}
		var req struct {
			Target string `json:"target"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}
		if err := h.SetDevTarget(req.Target); err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
		target := h.GetDevTarget()
		writeJSON(w, map[string]any{
			"active": target != "",
			"target": target,
		})
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (h *Handler) serveDevProxy(w http.ResponseWriter, r *http.Request) {
	h.devMu.RLock()
	proxy := h.devProxy
	h.devMu.RUnlock()

	if proxy == nil {
		http.Error(w, "dev proxy not configured", http.StatusServiceUnavailable)
		return
	}

	// Strip /miniapp/dev prefix so /miniapp/dev/foo → /foo
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/miniapp/dev")
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}
	proxy.ServeHTTP(w, r)
}

// extractUserFromInitData parses user.id from the initData query string.
// initData contains a "user" param with JSON like {"id":123456,...}.
func extractUserFromInitData(initData string) (userID, chatID string) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return "", ""
	}
	userJSON := values.Get("user")
	if userJSON == "" {
		return "", ""
	}
	var user struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(userJSON), &user); err != nil || user.ID == 0 {
		return "", ""
	}
	id := fmt.Sprintf("%d", user.ID)
	// For Mini App commands, chatID = userID (private chat)
	return id, id
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

	var lastPlan, lastSession, lastSkills, lastDev []byte

	// Send initial state immediately
	sendSSEIfChanged(w, flusher, "plan", h.provider.GetPlanInfo(), &lastPlan)
	sendSSEIfChanged(w, flusher, "session",
		map[string]any{"stats": h.provider.GetSessionStats(), "sessions": h.provider.GetActiveSessions()},
		&lastSession)
	sendSSEIfChanged(w, flusher, "skills", h.provider.ListSkills(), &lastSkills)
	sendSSEIfChanged(w, flusher, "dev", h.devStatus(), &lastDev)

	for {
		select {
		case <-r.Context().Done():
			return
		case <-h.notifier.Done():
			return
		case <-ch:
			sendSSEIfChanged(w, flusher, "plan", h.provider.GetPlanInfo(), &lastPlan)
			sendSSEIfChanged(w, flusher, "session",
				map[string]any{"stats": h.provider.GetSessionStats(), "sessions": h.provider.GetActiveSessions()},
				&lastSession)
			sendSSEIfChanged(w, flusher, "skills", h.provider.ListSkills(), &lastSkills)
			sendSSEIfChanged(w, flusher, "dev", h.devStatus(), &lastDev)
		}
	}
}

func (h *Handler) devStatus() map[string]any {
	target := h.GetDevTarget()
	return map[string]any{
		"active": target != "",
		"target": target,
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

// ValidateInitData verifies the Telegram WebApp initData HMAC-SHA256 signature.
// See https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
func ValidateInitData(initData, botToken string) bool {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return false
	}

	receivedHash := values.Get("hash")
	if receivedHash == "" {
		return false
	}

	// Build the data-check-string: sort all key=value pairs except "hash",
	// join with newlines.
	var pairs []string
	for key := range values {
		if key == "hash" {
			continue
		}
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, values.Get(key)))
	}
	sort.Strings(pairs)
	dataCheckString := strings.Join(pairs, "\n")

	// secret_key = HMAC-SHA256("WebAppData", bot_token)
	secretKeyMac := hmac.New(sha256.New, []byte("WebAppData"))
	secretKeyMac.Write([]byte(botToken))
	secretKey := secretKeyMac.Sum(nil)

	// hash = HMAC-SHA256(secret_key, data_check_string)
	hashMac := hmac.New(sha256.New, secretKey)
	hashMac.Write([]byte(dataCheckString))
	computedHash := hex.EncodeToString(hashMac.Sum(nil))

	return hmac.Equal([]byte(computedHash), []byte(receivedHash))
}
