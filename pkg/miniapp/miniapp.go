package miniapp

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/research"
)

//go:generate bun install --cwd frontend
//go:generate bun run --cwd frontend build
//go:embed static
var staticFS embed.FS

var (
	miniappStaticFS = mustMiniappStaticFS()
	miniappTemplate = template.Must(template.ParseFS(staticFS, "static/index.html"))
)

func mustMiniappStaticFS() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("miniapp: failed to create static sub filesystem: " + err.Error())
	}
	return sub
}

// Handler serves the Mini App HTML and API endpoints.
type Handler struct {
	provider        DataProvider
	sender          CommandSender
	botToken        string
	notifier        *StateNotifier
	allowList       []string
	workspace       string
	orchBroadcaster *orch.Broadcaster
	researchStore   *research.ResearchStore
	researchFocus   *research.FocusTracker

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
func NewHandler(
	provider DataProvider,
	sender CommandSender,
	botToken string,
	notifier *StateNotifier,
	allowList []string,
	workspace string,
) *Handler {
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

func (h *Handler) handleProtectedFunc(mux *http.ServeMux, pattern string, handler http.HandlerFunc) {
	mux.HandleFunc(pattern, h.requireAuth(handler))
}

// RegisterRoutes registers Mini App routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/miniapp", h.serveIndex)
	mux.HandleFunc("/miniapp/index.html", h.serveIndex)
	mux.HandleFunc("/miniapp/", h.serveStatic)
	h.handleProtectedFunc(mux, "/miniapp/api/skills", h.apiSkills)
	h.handleProtectedFunc(mux, "/miniapp/api/plan", h.apiPlan)
	h.handleProtectedFunc(mux, "/miniapp/api/session", h.apiSession)
	h.handleProtectedFunc(mux, "/miniapp/api/sessions", h.apiSessions)
	h.handleProtectedFunc(mux, "/miniapp/api/sessions/graph", h.apiSessionGraph)
	h.handleProtectedFunc(mux, "/miniapp/api/command", h.apiCommand)
	h.handleProtectedFunc(mux, "/miniapp/api/context", h.apiContext)
	h.handleProtectedFunc(mux, "/miniapp/api/prompt", h.apiPrompt)
	h.handleProtectedFunc(mux, "/miniapp/api/git", h.apiGit)
	h.handleProtectedFunc(mux, "/miniapp/api/worktrees", h.apiWorktrees)
	h.handleProtectedFunc(mux, "/miniapp/api/dev", h.apiDev)
	h.handleProtectedFunc(mux, "/miniapp/api/events", h.apiEvents)
	h.handleProtectedFunc(mux, "/miniapp/api/logs/ws", h.wsLogs)
	h.handleProtectedFunc(mux, "/miniapp/api/logs/snapshot", h.apiLogsSnapshot)
	h.handleProtectedFunc(mux, "/miniapp/api/logs/snapshot/", h.apiLogsSnapshotDownload)
	h.handleProtectedFunc(mux, "/miniapp/api/orchestration/ws", h.wsOrchestration)
	mux.HandleFunc("/miniapp/dev/console", h.apiDevConsole)
	mux.HandleFunc("/miniapp/dev/", h.serveDevProxy)
	h.handleProtectedFunc(mux, "/miniapp/api/cache", h.apiCache)
	h.handleProtectedFunc(mux, "/miniapp/api/cache/", h.apiCacheEntry)
	h.handleProtectedFunc(mux, "/miniapp/api/research", h.apiResearch)
	h.handleProtectedFunc(mux, "/miniapp/api/research/focus", h.apiResearchFocus)
	h.handleProtectedFunc(mux, "/miniapp/api/research/", h.apiResearchDetail)
}

func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct {
		OrchEnabled bool
	}{
		OrchEnabled: h.orchBroadcaster != nil,
	}
	if err := miniappTemplate.Execute(w, data); err != nil {
		http.Error(w, "failed to render template", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) serveStatic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/miniapp/" {
		h.serveIndex(w, r)
		return
	}
	http.StripPrefix("/miniapp/", http.FileServer(http.FS(miniappStaticFS))).ServeHTTP(w, r)
}
