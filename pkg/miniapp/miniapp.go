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
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strconv"
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

// DevTarget represents a registered dev server target.
type DevTarget struct {
	ID     string `json:"id"`
	Name   string `json:"name"`   // display name (e.g. "frontend")
	Target string `json:"target"` // URL (e.g. "http://localhost:3000")
}

// DevTargetManager allows tools to register, activate, and deactivate dev proxy targets.
type DevTargetManager interface {
	RegisterDevTarget(name, target string) (id string, err error)
	UnregisterDevTarget(id string) error
	ActivateDevTarget(id string) error
	DeactivateDevTarget() error
	GetDevTarget() string
	ListDevTargets() []DevTarget
}

// Handler serves the Mini App HTML and API endpoints.
type Handler struct {
	provider DataProvider
	sender   CommandSender
	botToken string
	notifier *StateNotifier

	devMu       sync.RWMutex
	devTarget   *url.URL
	devProxy    *httputil.ReverseProxy
	devTargets  map[string]*DevTarget // registered targets (ID→DevTarget)
	devNextID   int
	devActiveID string
}

// NewHandler creates a new Mini App handler.
func NewHandler(provider DataProvider, sender CommandSender, botToken string, notifier *StateNotifier) *Handler {
	return &Handler{
		provider:   provider,
		sender:     sender,
		botToken:   botToken,
		notifier:   notifier,
		devTargets: make(map[string]*DevTarget),
	}
}

// validateLocalhostURL parses and validates that a URL targets localhost.
func validateLocalhostURL(target string) (*url.URL, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	host := u.Hostname()
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return nil, fmt.Errorf("only localhost targets are allowed, got %q", host)
	}
	return u, nil
}

// RegisterDevTarget registers a new dev server target. Only localhost targets are allowed.
func (h *Handler) RegisterDevTarget(name, target string) (string, error) {
	if _, err := validateLocalhostURL(target); err != nil {
		return "", err
	}

	h.devMu.Lock()
	defer h.devMu.Unlock()

	h.devNextID++
	id := strconv.Itoa(h.devNextID)

	h.devTargets[id] = &DevTarget{ID: id, Name: name, Target: target}
	if h.notifier != nil {
		h.notifier.Notify()
	}
	return id, nil
}

// UnregisterDevTarget removes a registered target. If it was active, the proxy is disabled.
func (h *Handler) UnregisterDevTarget(id string) error {
	h.devMu.Lock()
	defer h.devMu.Unlock()

	if _, ok := h.devTargets[id]; !ok {
		return fmt.Errorf("target %q not found", id)
	}
	delete(h.devTargets, id)

	if h.devActiveID == id {
		h.devActiveID = ""
		h.devTarget = nil
		h.devProxy = nil
	}
	if h.notifier != nil {
		h.notifier.Notify()
	}
	return nil
}

// ActivateDevTarget sets the reverse proxy to the registered target with the given ID.
func (h *Handler) ActivateDevTarget(id string) error {
	h.devMu.Lock()
	defer h.devMu.Unlock()

	dt, ok := h.devTargets[id]
	if !ok {
		return fmt.Errorf("target %q not found", id)
	}

	u, err := url.Parse(dt.Target)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Fix IPv6: resolve "localhost" to 127.0.0.1 to avoid connection refused on systems
	// where localhost resolves to [::1] but the dev server only listens on IPv4.
	if u.Hostname() == "localhost" {
		u.Host = net.JoinHostPort("127.0.0.1", u.Port())
	}

	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Prevent browser/WebView from caching dev proxy responses (CSS, JS, etc.)
		resp.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		resp.Header.Del("ETag")
		resp.Header.Del("Last-Modified")

		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "text/html") {
			return nil
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		resp.Body.Close()
		modified := injectDevProxyScript(body)
		resp.Body = io.NopCloser(bytes.NewReader(modified))
		resp.ContentLength = int64(len(modified))
		resp.Header.Set("Content-Length", strconv.Itoa(len(modified)))
		resp.Header.Del("Content-Encoding")
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><style>
body{background:#1c1c1e;color:#fff;font-family:-apple-system,sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;margin:0}
.box{text-align:center;padding:32px}
h2{margin:0 0 12px;font-size:20px;font-weight:600}
p{color:#8e8e93;font-size:14px;margin:0}
</style></head><body><div class="box"><h2>Cannot connect</h2><p>%s</p><p style="margin-top:8px;font-size:12px">Target: %s</p></div></body></html>`,
			escapeHTMLString(err.Error()), escapeHTMLString(dt.Target))
	}

	h.devTarget = u
	h.devProxy = proxy
	h.devActiveID = id
	if h.notifier != nil {
		h.notifier.Notify()
	}
	return nil
}

// DeactivateDevTarget disables the reverse proxy without removing registrations.
func (h *Handler) DeactivateDevTarget() error {
	h.devMu.Lock()
	defer h.devMu.Unlock()

	h.devActiveID = ""
	h.devTarget = nil
	h.devProxy = nil
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

// ListDevTargets returns all registered dev targets.
func (h *Handler) ListDevTargets() []DevTarget {
	h.devMu.RLock()
	defer h.devMu.RUnlock()

	targets := make([]DevTarget, 0, len(h.devTargets))
	for _, dt := range h.devTargets {
		targets = append(targets, *dt)
	}
	// Sort by ID for stable order
	sort.Slice(targets, func(i, j int) bool { return targets[i].ID < targets[j].ID })
	return targets
}

// devProxyScript is the JavaScript injected into HTML responses from the dev proxy.
// It rewrites fetch() and XMLHttpRequest.open() so that absolute paths like
// "/api/items" are prefixed with "/miniapp/dev", matching the reverse proxy mount.
const devProxyScript = `<script data-dev-proxy>
(function(){
  var B='/miniapp/dev';
  function rw(u){
    if(typeof u==='string'&&u.startsWith('/')&&!u.startsWith('//')&&!u.startsWith(B))return B+u;
    return u;
  }
  var _f=window.fetch;
  window.fetch=function(r,i){
    if(typeof r==='string')r=rw(r);
    else if(r instanceof Request)r=new Request(rw(r.url),r);
    return _f.call(this,r,i);
  };
  var _o=XMLHttpRequest.prototype.open;
  XMLHttpRequest.prototype.open=function(m,u){
    arguments[1]=rw(u);
    return _o.apply(this,arguments);
  };
})();
</script>`

// injectDevProxyScript inserts the dev proxy rewrite script into an HTML document.
// Insertion priority: before </head>, after <body...>, or prepend to document.
func injectDevProxyScript(html []byte) []byte {
	script := []byte(devProxyScript)

	// Priority 1: before </head>
	if idx := bytes.Index(bytes.ToLower(html), []byte("</head>")); idx >= 0 {
		out := make([]byte, 0, len(html)+len(script))
		out = append(out, html[:idx]...)
		out = append(out, script...)
		out = append(out, html[idx:]...)
		return out
	}

	// Priority 2: after <body ...>
	lower := bytes.ToLower(html)
	if idx := bytes.Index(lower, []byte("<body")); idx >= 0 {
		// Find the closing '>' of the <body> tag
		closeIdx := bytes.IndexByte(lower[idx:], '>')
		if closeIdx >= 0 {
			insertAt := idx + closeIdx + 1
			out := make([]byte, 0, len(html)+len(script))
			out = append(out, html[:insertAt]...)
			out = append(out, script...)
			out = append(out, html[insertAt:]...)
			return out
		}
	}

	// Priority 3: prepend
	out := make([]byte, 0, len(html)+len(script))
	out = append(out, script...)
	out = append(out, html...)
	return out
}

// escapeHTMLString escapes HTML special characters in a string.
func escapeHTMLString(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
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
		writeJSON(w, h.devStatus())
	case http.MethodPost:
		body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
		if err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}
		var req struct {
			Action string `json:"action"`
			ID     string `json:"id"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}
		switch req.Action {
		case "activate":
			if req.ID == "" {
				writeJSON(w, map[string]any{"error": "id is required"})
				return
			}
			if err := h.ActivateDevTarget(req.ID); err != nil {
				writeJSON(w, map[string]any{"error": err.Error()})
				return
			}
		case "deactivate":
			if err := h.DeactivateDevTarget(); err != nil {
				writeJSON(w, map[string]any{"error": err.Error()})
				return
			}
		case "unregister":
			if req.ID == "" {
				writeJSON(w, map[string]any{"error": "id is required"})
				return
			}
			if err := h.UnregisterDevTarget(req.ID); err != nil {
				writeJSON(w, map[string]any{"error": err.Error()})
				return
			}
		default:
			writeJSON(w, map[string]any{"error": "unknown action"})
			return
		}
		writeJSON(w, h.devStatus())
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
	h.devMu.RLock()
	defer h.devMu.RUnlock()

	active := h.devTarget != nil
	target := ""
	if h.devTarget != nil {
		target = h.devTargets[h.devActiveID].Target // original URL before IPv6 rewrite
	}

	targets := make([]DevTarget, 0, len(h.devTargets))
	for _, dt := range h.devTargets {
		targets = append(targets, *dt)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].ID < targets[j].ID })

	return map[string]any{
		"active":    active,
		"active_id": h.devActiveID,
		"target":    target,
		"targets":   targets,
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
