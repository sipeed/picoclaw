package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	ppid "github.com/sipeed/picoclaw/pkg/pid"
)

// registerPicoRoutes binds Pico Channel management endpoints to the ServeMux.
func (h *Handler) registerPicoRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/pico/info", h.handleGetPicoInfo)
	mux.HandleFunc("GET /api/pico/memory-graph", h.handleGetPicoMemoryGraph)
	mux.HandleFunc("GET /api/pico/subagents", h.handleGetPicoSubagents)
	mux.HandleFunc("POST /api/pico/token", h.handleRegenPicoToken)
	mux.HandleFunc("POST /api/pico/setup", h.handlePicoSetup)

	// WebSocket proxy: forward /pico/ws to gateway
	// This allows the frontend to connect via the same port as the web UI,
	// avoiding the need to expose extra ports for WebSocket communication.
	mux.HandleFunc("GET /pico/ws", h.handleWebSocketProxy())
	mux.HandleFunc("GET /pico/media/{id}", h.handlePicoMediaProxy())
	mux.HandleFunc("HEAD /pico/media/{id}", h.handlePicoMediaProxy())
}

type picoSubagentStatusItem struct {
	ID      string `json:"id"`
	Label   string `json:"label,omitempty"`
	Status  string `json:"status"`
	Created int64  `json:"created"`
	Result  string `json:"result,omitempty"`
}

type picoSubagentStatusResponse struct {
	SessionID string                   `json:"session_id"`
	Channel   string                   `json:"channel"`
	ChatID    string                   `json:"chat_id"`
	Tasks     []picoSubagentStatusItem `json:"tasks"`
}

type picoMemoryGraphNode struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Kind    string `json:"kind"`
	Group   string `json:"group"`
	Preview string `json:"preview,omitempty"`
	Weight  int    `json:"weight,omitempty"`
}

type picoMemoryGraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

type picoMemoryGraphResponse struct {
	SessionID   string                `json:"session_id"`
	GeneratedAt string                `json:"generated_at"`
	Nodes       []picoMemoryGraphNode `json:"nodes"`
	Edges       []picoMemoryGraphEdge `json:"edges"`
}

// createWsProxy creates a reverse proxy to the current gateway WebSocket endpoint.
// The gateway bind host and port are resolved from the latest configuration.
func (h *Handler) createWsProxy(origProtocol string, upstreamProtocol string) *httputil.ReverseProxy {
	wsProxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			target := h.gatewayProxyURL()
			r.SetURL(target)
			r.Out.Header.Del(protocolKey)
			if upstreamProtocol != "" {
				r.Out.Header.Set(protocolKey, upstreamProtocol)
			}
		},
		ModifyResponse: func(r *http.Response) error {
			if prot := r.Header.Values(protocolKey); len(prot) > 0 {
				r.Header.Del(protocolKey)
				if origProtocol != "" {
					r.Header.Set(protocolKey, origProtocol)
				}
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Errorf("Failed to proxy WebSocket: %v", err)
			http.Error(w, "Gateway unavailable: "+err.Error(), http.StatusBadGateway)
		},
	}
	return wsProxy
}

func (h *Handler) createPicoHTTPProxy(token string) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			target := h.gatewayProxyURL()
			r.SetURL(target)
			r.Out.Header.Set("Authorization", "Bearer "+token)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Errorf("Failed to proxy Pico HTTP request: %v", err)
			http.Error(w, "Gateway unavailable: "+err.Error(), http.StatusBadGateway)
		},
	}
}

func (h *Handler) gatewayAvailableForProxy() bool {
	gateway.mu.Lock()
	ensurePicoTokenCachedLocked(h.configPath)
	cachedPID := gateway.pidData
	trackedCmd := gateway.cmd
	gateway.mu.Unlock()

	if pidData := h.sanitizeGatewayPidData(ppid.ReadPidFileWithCheck(globalConfigDir()), nil); pidData != nil {
		gateway.mu.Lock()
		gateway.pidData = pidData
		setGatewayRuntimeStatusLocked("running")
		gateway.mu.Unlock()
		return true
	}

	if cachedPID == nil {
		return false
	}

	if isCmdProcessAliveLocked(trackedCmd) {
		return true
	}

	gateway.mu.Lock()
	if gateway.cmd == trackedCmd {
		gateway.pidData = nil
		setGatewayRuntimeStatusLocked("stopped")
	}
	available := gateway.pidData != nil
	gateway.mu.Unlock()
	return available
}

func decodePicoSettings(cfg *config.Config) (config.PicoSettings, bool) {
	if cfg == nil {
		return config.PicoSettings{}, false
	}

	bc := cfg.Channels.GetByType(config.ChannelPico)
	if bc == nil {
		return config.PicoSettings{}, false
	}

	var picoCfg config.PicoSettings
	if err := bc.Decode(&picoCfg); err != nil {
		return config.PicoSettings{}, false
	}

	return picoCfg, bc.Enabled
}

func (h *Handler) writePicoInfoResponse(
	w http.ResponseWriter,
	r *http.Request,
	cfg *config.Config,
	changed *bool,
) {
	picoCfg, enabled := decodePicoSettings(cfg)

	resp := map[string]any{
		"ws_url":  h.buildWsURL(r),
		"enabled": enabled,
	}
	if changed != nil {
		resp["changed"] = *changed
	}
	if picoCfg.Token.String() != "" {
		resp["configured"] = true
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleWebSocketProxy wraps a reverse proxy to handle WebSocket connections.
// It relies on launcher dashboard auth, then injects the raw pico token only
// on the upstream gateway request.
func (h *Handler) handleWebSocketProxy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.gatewayAvailableForProxy() {
			logger.Warnf("Gateway not available for WebSocket proxy")
			http.Error(w, "Gateway not available", http.StatusServiceUnavailable)
			return
		}

		upstreamProtocol := picoGatewayProtocol()
		if upstreamProtocol == "" {
			logger.Warn("Pico token unavailable for WebSocket proxy")
			http.Error(w, "Pico channel not configured", http.StatusServiceUnavailable)
			return
		}

		var origProtocol string
		if prot := r.Header.Values(protocolKey); len(prot) > 0 {
			origProtocol = prot[0]
		}

		h.createWsProxy(origProtocol, upstreamProtocol).ServeHTTP(w, r)
	}
}

func (h *Handler) handlePicoMediaProxy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.gatewayAvailableForProxy() {
			logger.Warnf("Gateway not available for Pico media proxy")
			http.Error(w, "Gateway not available", http.StatusServiceUnavailable)
			return
		}

		gateway.mu.Lock()
		picoToken := gateway.picoToken
		gateway.mu.Unlock()

		if picoToken == "" {
			logger.Warnf("Missing Pico token for media proxy")
			http.Error(w, "Invalid Pico token", http.StatusForbidden)
			return
		}

		h.createPicoHTTPProxy(picoToken).ServeHTTP(w, r)
	}
}

// handleGetPicoInfo returns non-secret Pico connection info for the launcher UI.
//
//	GET /api/pico/info
func (h *Handler) handleGetPicoInfo(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	h.writePicoInfoResponse(w, r, cfg, nil)
}

func (h *Handler) handleGetPicoMemoryGraph(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}

	sessionDir := resolveSessionsDir(cfg.Agents.Defaults.Workspace)
	toolFeedbackMaxArgsLength := cfg.Agents.Defaults.GetToolFeedbackMaxArgsLength()
	workspaceDir := resolveWorkspaceDir(cfg.Agents.Defaults.Workspace)

	ref, refErr := h.findPicoJSONLSession(sessionDir, sessionID)
	var sess sessionFile
	err = refErr
	if refErr == nil {
		sess, err = h.readJSONLSession(sessionDir, ref.Key)
	}
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	nodes, edges := buildPicoMemoryGraph(
		sessionID,
		detailSessionMessages(sess.Messages, toolFeedbackMaxArgsLength),
		workspaceDir,
	)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(picoMemoryGraphResponse{
		SessionID:   sessionID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Nodes:       nodes,
		Edges:       edges,
	})
}

func (h *Handler) handleGetPicoSubagents(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	if !h.gatewayAvailableForProxy() {
		logger.Warnf("Gateway not available for Pico subagent status proxy")
		http.Error(w, "Gateway not available", http.StatusServiceUnavailable)
		return
	}

	gateway.mu.Lock()
	pidData := gateway.pidData
	gateway.mu.Unlock()

	if pidData == nil || pidData.Token == "" {
		logger.Warnf("Gateway auth token not available for Pico subagent status proxy")
		http.Error(w, "Gateway auth token not available", http.StatusServiceUnavailable)
		return
	}

	target := h.gatewayProxyURL()
	target.Path = "/internal/subagents/status"
	query := target.Query()
	query.Set("channel", "pico")
	query.Set("chat_id", sessionID)
	target.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target.String(), nil)
	if err != nil {
		http.Error(w, "Failed to create subagent status request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", "Bearer "+pidData.Token)

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		logger.Errorf("Failed to fetch Pico subagent status: %v", err)
		http.Error(w, "Gateway unavailable: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
		return
	}

	var upstream struct {
		Channel string `json:"channel"`
		ChatID  string `json:"chat_id"`
		Tasks   []struct {
			ID      string `json:"id"`
			Label   string `json:"label"`
			Status  string `json:"status"`
			Created int64  `json:"created"`
			Result  string `json:"result"`
		} `json:"tasks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&upstream); err != nil {
		http.Error(w, "Failed to decode subagent status response", http.StatusBadGateway)
		return
	}

	tasks := make([]picoSubagentStatusItem, 0, len(upstream.Tasks))
	for _, task := range upstream.Tasks {
		tasks = append(tasks, picoSubagentStatusItem{
			ID:      task.ID,
			Label:   task.Label,
			Status:  task.Status,
			Created: task.Created,
			Result:  summarizeSubagentResult(task.Result),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(picoSubagentStatusResponse{
		SessionID: sessionID,
		Channel:   upstream.Channel,
		ChatID:    upstream.ChatID,
		Tasks:     tasks,
	})
}

func summarizeSubagentResult(result string) string {
	result = strings.TrimSpace(result)
	if result == "" {
		return ""
	}
	const maxRunes = 180
	runes := []rune(result)
	if len(runes) <= maxRunes {
		return result
	}
	return string(runes[:maxRunes]) + "..."
}

func resolveWorkspaceDir(workspace string) string {
	if workspace == "" {
		home, _ := os.UserHomeDir()
		workspace = filepath.Join(home, ".picoclaw", "workspace")
	}
	if len(workspace) > 0 && workspace[0] == '~' {
		home, _ := os.UserHomeDir()
		if len(workspace) > 1 && workspace[1] == '/' {
			workspace = home + workspace[1:]
		} else {
			workspace = home
		}
	}
	return workspace
}

func buildPicoMemoryGraph(sessionID string, messages []sessionChatMessage, workspaceDir string) ([]picoMemoryGraphNode, []picoMemoryGraphEdge) {
	nodes := make([]picoMemoryGraphNode, 0, 32)
	edges := make([]picoMemoryGraphEdge, 0, 48)
	seenNodes := make(map[string]struct{})
	seenEdges := make(map[string]struct{})

	addNode := func(node picoMemoryGraphNode) {
		if _, exists := seenNodes[node.ID]; exists {
			return
		}
		seenNodes[node.ID] = struct{}{}
		nodes = append(nodes, node)
	}

	addEdge := func(edge picoMemoryGraphEdge) {
		key := edge.Source + "\x00" + edge.Target + "\x00" + edge.Kind
		if _, exists := seenEdges[key]; exists {
			return
		}
		seenEdges[key] = struct{}{}
		edges = append(edges, edge)
	}

	const (
		memoryRootID  = "memory-root"
		sessionRootID = "session-root"
	)

	addNode(picoMemoryGraphNode{
		ID:      memoryRootID,
		Label:   "Workspace Memory",
		Kind:    "root",
		Group:   "memory",
		Preview: "Long-term memory and recent notes",
		Weight:  5,
	})
	addNode(picoMemoryGraphNode{
		ID:      sessionRootID,
		Label:   "Active Session",
		Kind:    "root",
		Group:   "session",
		Preview: sessionID,
		Weight:  5,
	})
	addEdge(picoMemoryGraphEdge{Source: memoryRootID, Target: sessionRootID, Kind: "context"})

	appendMemoryDocumentGraph(memoryRootID, filepath.Join(workspaceDir, "memory", "MEMORY.md"), "memory", "Long-term Memory", 8, addNode, addEdge)
	appendRecentDailyNoteGraph(memoryRootID, filepath.Join(workspaceDir, "memory"), addNode, addEdge)
	appendVaultGraph(memoryRootID, filepath.Join(workspaceDir, "vault"), addNode, addEdge)
	appendSessionGraph(sessionRootID, messages, addNode, addEdge)

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Source == edges[j].Source {
			if edges[i].Target == edges[j].Target {
				return edges[i].Kind < edges[j].Kind
			}
			return edges[i].Target < edges[j].Target
		}
		return edges[i].Source < edges[j].Source
	})

	return nodes, edges
}

func appendMemoryDocumentGraph(
	rootID string,
	path string,
	group string,
	title string,
	maxItems int,
	addNode func(picoMemoryGraphNode),
	addEdge func(picoMemoryGraphEdge),
) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return
	}

	docID := group + ":document"
	addNode(picoMemoryGraphNode{
		ID:      docID,
		Label:   title,
		Kind:    "document",
		Group:   group,
		Preview: filepath.Base(path),
		Weight:  4,
	})
	addEdge(picoMemoryGraphEdge{Source: rootID, Target: docID, Kind: "contains"})

	lines := strings.Split(string(data), "\n")
	currentParent := docID
	items := 0
	headingIndex := 0
	noteIndex := 0

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			headingText := summarizeGraphText(strings.TrimSpace(strings.TrimLeft(line, "#")), 48)
			if headingText == "" {
				continue
			}
			headingIndex++
			headingID := group + ":heading:" + strconv.Itoa(headingIndex)
			addNode(picoMemoryGraphNode{
				ID:      headingID,
				Label:   headingText,
				Kind:    "heading",
				Group:   group,
				Preview: headingText,
				Weight:  3,
			})
			addEdge(picoMemoryGraphEdge{Source: docID, Target: headingID, Kind: "section"})
			currentParent = headingID
			continue
		}

		trimmed := strings.TrimSpace(strings.TrimLeft(line, "-*0123456789. "))
		if trimmed == "" {
			continue
		}

		noteIndex++
		noteID := group + ":note:" + strconv.Itoa(noteIndex)
		addNode(picoMemoryGraphNode{
			ID:      noteID,
			Label:   summarizeGraphText(trimmed, 38),
			Kind:    "note",
			Group:   group,
			Preview: summarizeGraphText(trimmed, 140),
			Weight:  2,
		})
		addEdge(picoMemoryGraphEdge{Source: currentParent, Target: noteID, Kind: "note"})
		items++
		if items >= maxItems {
			break
		}
	}
}

func appendRecentDailyNoteGraph(
	rootID string,
	memoryDir string,
	addNode func(picoMemoryGraphNode),
	addEdge func(picoMemoryGraphEdge),
) {
	matches, err := filepath.Glob(filepath.Join(memoryDir, "*", "*.md"))
	if err != nil || len(matches) == 0 {
		return
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i] > matches[j]
	})

	for idx, match := range matches {
		if idx >= 3 {
			break
		}
		appendMemoryDocumentGraph(
			rootID,
			match,
			"daily-"+strconv.Itoa(idx+1),
			"Daily Note "+filepath.Base(match),
			4,
			addNode,
			addEdge,
		)
	}
}

func appendSessionGraph(
	rootID string,
	messages []sessionChatMessage,
	addNode func(picoMemoryGraphNode),
	addEdge func(picoMemoryGraphEdge),
) {
	if len(messages) == 0 {
		return
	}

	start := 0
	if len(messages) > 8 {
		start = len(messages) - 8
	}
	recent := messages[start:]
	previousID := rootID

	for index, message := range recent {
		messageID := "session:message:" + strconv.Itoa(index)
		label := strings.ToUpper(message.Role)
		if preview := summarizeGraphText(message.Content, 34); preview != "" {
			label += ": " + preview
		}
		if label == strings.ToUpper(message.Role) && len(message.Attachments) > 0 {
			label += ": attachment"
		}

		addNode(picoMemoryGraphNode{
			ID:      messageID,
			Label:   label,
			Kind:    "message",
			Group:   "session",
			Preview: summarizeGraphText(message.Content, 160),
			Weight:  3,
		})
		addEdge(picoMemoryGraphEdge{Source: previousID, Target: messageID, Kind: "flow"})
		previousID = messageID

		for toolIndex, toolCall := range message.ToolCalls {
			if toolCall.Function == nil {
				continue
			}
			name := strings.TrimSpace(toolCall.Function.Name)
			if name == "" {
				continue
			}
			toolID := messageID + ":tool:" + strconv.Itoa(toolIndex)
			addNode(picoMemoryGraphNode{
				ID:      toolID,
				Label:   name,
				Kind:    "tool",
				Group:   "tool",
				Preview: summarizeGraphText(toolCall.Function.Arguments, 120),
				Weight:  2,
			})
			addEdge(picoMemoryGraphEdge{Source: messageID, Target: toolID, Kind: "tool"})
		}

		for attachmentIndex, attachment := range message.Attachments {
			name := strings.TrimSpace(attachment.Filename)
			if name == "" {
				name = attachment.Type
			}
			if name == "" {
				name = "attachment"
			}
			attachmentID := messageID + ":attachment:" + strconv.Itoa(attachmentIndex)
			addNode(picoMemoryGraphNode{
				ID:      attachmentID,
				Label:   summarizeGraphText(name, 34),
				Kind:    "attachment",
				Group:   "media",
				Preview: summarizeGraphText(attachment.URL, 120),
				Weight:  1,
			})
			addEdge(picoMemoryGraphEdge{Source: messageID, Target: attachmentID, Kind: "attachment"})
		}
	}
}

func summarizeGraphText(text string, maxRunes int) string {
	trimmed := strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	return string(runes[:maxRunes-1]) + "…"
}

// handleRegenPicoToken rotates the raw Pico WebSocket token and returns
// non-secret connection info for the launcher UI.
//
//	POST /api/pico/token
func (h *Handler) handleRegenPicoToken(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	token := generateSecureToken()
	if bc := cfg.Channels.GetByType(config.ChannelPico); bc != nil {
		decoded, err := bc.GetDecoded()
		if err == nil && decoded != nil {
			if settings, ok := decoded.(*config.PicoSettings); ok {
				settings.Token = *config.NewSecureString(token)
			}
		}
	}

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	gateway.mu.Lock()
	gateway.picoToken = token
	gateway.mu.Unlock()

	h.writePicoInfoResponse(w, r, cfg, nil)
}

// EnsurePicoChannel enables the Pico channel with sane defaults if it isn't
// already configured. Returns true when the config was modified.
func (h *Handler) EnsurePicoChannel() (bool, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return false, fmt.Errorf("failed to load config: %w", err)
	}

	changed := false

	bc := cfg.Channels.GetByType(config.ChannelPico)
	if bc == nil {
		bc = &config.Channel{Type: config.ChannelPico}
		cfg.Channels["pico"] = bc
	}

	if !bc.Enabled {
		bc.Enabled = true
		changed = true
	}

	if decoded, err := bc.GetDecoded(); err == nil && decoded != nil {
		if picoCfg, ok := decoded.(*config.PicoSettings); ok {
			if picoCfg.Token.String() == "" {
				picoCfg.Token = *config.NewSecureString(generateSecureToken())
				changed = true
			}
		}
	}

	if changed {
		if err := config.SaveConfig(h.configPath, cfg); err != nil {
			return false, fmt.Errorf("failed to save config: %w", err)
		}
	}

	return changed, nil
}

// handlePicoSetup automatically configures everything needed for the Pico Channel to work.
//
//	POST /api/pico/setup
func (h *Handler) handlePicoSetup(w http.ResponseWriter, r *http.Request) {
	changed, err := h.EnsurePicoChannel()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Reload config (EnsurePicoChannel may have modified it).
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	h.writePicoInfoResponse(w, r, cfg, &changed)
}

func appendVaultGraph(
	rootID string,
	vaultDir string,
	addNode func(picoMemoryGraphNode),
	addEdge func(picoMemoryGraphEdge),
) {
	// Read all .md files from vault directory
	entries, err := os.ReadDir(vaultDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		notePath := filepath.Join(vaultDir, entry.Name())
		data, err := os.ReadFile(notePath)
		if err != nil {
			continue
		}

		noteName := strings.TrimSuffix(entry.Name(), ".md")
		noteID := "vault:" + noteName

		// Parse frontmatter and content
		fm, body := memory.ParseFrontmatter(string(data))

		// Create node
		label := noteName
		if title, ok := fm["title"].(string); ok && title != "" {
			label = title
		}
		preview := string(data)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}

		addNode(picoMemoryGraphNode{
			ID:      noteID,
			Label:   label,
			Kind:    "document",
			Group:   "memory",
			Preview: preview,
			Weight:  3,
		})
		addEdge(picoMemoryGraphEdge{Source: rootID, Target: noteID, Kind: "contains"})

		// Extract wiki-links and create edges
		re := regexp.MustCompile(`\[\[([^\]]+)\]`)
		matches := re.FindAllStringSubmatch(body, -1)
		for _, m := range matches {
			targetID := "vault:" + m[1]
			addEdge(picoMemoryGraphEdge{Source: noteID, Target: targetID, Kind: "wikilink"})
		}
	}
}

// generateSecureToken creates a random 32-character hex string.
func generateSecureToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to something pseudo-random if crypto/rand fails
		return fmt.Sprintf("%032x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
