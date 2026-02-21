package miniapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

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

// DataProvider is the read-only interface to agent state for the Mini App API.
type DataProvider interface {
	ListSkills() []skills.SkillInfo
	GetPlanInfo() PlanInfo
	GetSessionStats() *stats.Stats
	GetActiveSessions() []SessionInfo
}

// CommandSender injects a command into the message bus on behalf of a user.
type CommandSender interface {
	SendCommand(senderID, chatID, command string)
}

// Handler serves the Mini App HTML and API endpoints.
type Handler struct {
	provider DataProvider
	sender   CommandSender
	botToken string
}

// NewHandler creates a new Mini App handler.
func NewHandler(provider DataProvider, sender CommandSender, botToken string) *Handler {
	return &Handler{
		provider: provider,
		sender:   sender,
		botToken: botToken,
	}
}

// RegisterRoutes registers Mini App routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/miniapp", h.serveIndex)
	mux.HandleFunc("/miniapp/api/skills", h.requireAuth(h.apiSkills))
	mux.HandleFunc("/miniapp/api/plan", h.requireAuth(h.apiPlan))
	mux.HandleFunc("/miniapp/api/session", h.requireAuth(h.apiSession))
	mux.HandleFunc("/miniapp/api/sessions", h.requireAuth(h.apiSessions))
	mux.HandleFunc("/miniapp/api/command", h.requireAuth(h.apiCommand))
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
