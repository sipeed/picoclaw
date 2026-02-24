package miniapp

import (
	"embed"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/sipeed/picoclaw/pkg/orch"
)

//go:embed static/index.html
var staticFS embed.FS

// Handler serves the Mini App HTML and API endpoints.
type Handler struct {
	provider        DataProvider
	sender          CommandSender
	botToken        string
	notifier        *StateNotifier
	allowList       []string
	workspace       string
	orchBroadcaster *orch.Broadcaster

	devMu       sync.RWMutex
	devTarget   *url.URL
	devProxy    *httputil.ReverseProxy
	devTargets  map[string]*DevTarget // registered targets (ID→DevTarget)
	devNextID   int
	devActiveID string

	wsClients   []*wsClient
	wsClientsMu sync.Mutex

	consoleMu       sync.Mutex
	consoleReqCount int
	consoleReqSec   int64
}

// NewHandler creates a new Mini App handler.
func NewHandler(provider DataProvider, sender CommandSender, botToken string, notifier *StateNotifier, allowList []string, workspace string) *Handler {
	return &Handler{
		provider:   provider,
		sender:     sender,
		botToken:   botToken,
		notifier:   notifier,
		allowList:  allowList,
		workspace:  workspace,
		devTargets: make(map[string]*DevTarget),
	}
}

// SetOrchBroadcaster wires the orchestration broadcaster so the Mini App can
// push live agent state to the canvas UI via WebSocket.
func (h *Handler) SetOrchBroadcaster(b *orch.Broadcaster) {
	h.orchBroadcaster = b
}

// RegisterRoutes registers Mini App routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/miniapp", h.serveIndex)
	mux.HandleFunc("/miniapp/api/skills", h.requireAuth(h.apiSkills))
	mux.HandleFunc("/miniapp/api/plan", h.requireAuth(h.apiPlan))
	mux.HandleFunc("/miniapp/api/session", h.requireAuth(h.apiSession))
	mux.HandleFunc("/miniapp/api/sessions", h.requireAuth(h.apiSessions))
	mux.HandleFunc("/miniapp/api/command", h.requireAuth(h.apiCommand))
	mux.HandleFunc("/miniapp/api/context", h.requireAuth(h.apiContext))
	mux.HandleFunc("/miniapp/api/prompt", h.requireAuth(h.apiPrompt))
	mux.HandleFunc("/miniapp/api/git", h.requireAuth(h.apiGit))
	mux.HandleFunc("/miniapp/api/dev", h.requireAuth(h.apiDev))
	mux.HandleFunc("/miniapp/api/events", h.requireAuth(h.apiEvents))
	mux.HandleFunc("/miniapp/api/logs/ws", h.requireAuth(h.wsLogs))
	mux.HandleFunc("/miniapp/api/logs/snapshot", h.requireAuth(h.apiLogsSnapshot))
	mux.HandleFunc("/miniapp/api/logs/snapshot/", h.requireAuth(h.apiLogsSnapshotDownload))
	mux.HandleFunc("/miniapp/api/orchestration/ws", h.requireAuth(h.wsOrchestration))
	mux.HandleFunc("/miniapp/dev/console", h.apiDevConsole)
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
