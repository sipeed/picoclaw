package magicform

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/pathutil"
)

// WebhookPayload is the inbound payload from MagicForm.
type WebhookPayload struct {
	StackID        string `json:"stackId"`
	ConversationID string `json:"conversationId"`
	UserID         string `json:"userId"`
	Message        string `json:"message"`
	Workspace      string `json:"workspace"`           // e.g. "s1/c1" — agent working directory (relative to workspace_root)
	ConfigDir      string `json:"configDir,omitempty"` // e.g. "s1/config" — pre-provisioned config directory (relative to workspace_root)
	CallbackURL    string `json:"callbackUrl"`

	// Tool/skill filtering
	AllowedTools  []string `json:"allowedTools,omitempty"`  // Tool allowlist (empty = all)
	AllowedSkills []string `json:"allowedSkills,omitempty"` // Skill filter (empty = all)
}

// CallbackPayload is the outbound payload sent back to MagicForm.
type CallbackPayload struct {
	StackID        string `json:"stackId"`
	ConversationID string `json:"conversationId"`
	TaskID         string `json:"taskId,omitempty"`
	Type           string `json:"type"`             // "final", "progress", "escalation"
	Status         string `json:"status"`            // "success", "error"
	Response       string `json:"response"`
	Error          string `json:"error,omitempty"`
	Runtime        string `json:"runtime"`
	DurationMs     int64  `json:"durationMs,omitempty"`
	TokenUsage     *bus.TokenUsage       `json:"tokenUsage,omitempty"`
	ToolCalls      int                   `json:"toolCalls,omitempty"`
	Progress       *ProgressPayload      `json:"progress"`       // null unless type=progress
	Escalation     *EscalationPayload    `json:"escalation"`     // null unless type=escalation
}

// ProgressPayload is populated for type="progress" callbacks.
type ProgressPayload struct {
	Status     string `json:"status"`               // e.g. "thinking"
	ToolName   string `json:"toolName,omitempty"`
	StepNumber int    `json:"stepNumber,omitempty"`
	Message    string `json:"message,omitempty"`
}

// EscalationPayload is populated for type="escalation" callbacks.
type EscalationPayload struct {
	Reason string `json:"reason"`
	Notes  string `json:"notes,omitempty"`
}

// requestContext stores per-request state so Send() can resolve callback info.
type requestContext struct {
	stackID        string
	conversationID string
	userID         string
	callbackURL    string
	createdAt      time.Time
}

// MagicFormChannel implements the MagicForm channel plugin.
type MagicFormChannel struct {
	*channels.BaseChannel
	settings      *config.MagicFormSettings
	workspaceRoot string // effective root: channel-level fallback to global
	httpClient    *http.Client
	requests      sync.Map // chatID → *requestContext
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewMagicFormChannel creates a new MagicForm channel.
// globalWorkspaceRoot is the agents.defaults.workspace_root from the base config.
// The channel uses its own settings.WorkspaceRoot if set, otherwise falls back to
// globalWorkspaceRoot. If neither is configured, the constructor returns an error
// because workspace overrides cannot be validated without a root boundary.
func NewMagicFormChannel(
	bc *config.Channel,
	settings *config.MagicFormSettings,
	globalWorkspaceRoot string,
	msgBus *bus.MessageBus,
) (*MagicFormChannel, error) {
	base := channels.NewBaseChannel(
		"magicform",
		settings,
		msgBus,
		bc.AllowFrom,
	)

	effectiveRoot := settings.WorkspaceRoot
	if effectiveRoot == "" {
		effectiveRoot = globalWorkspaceRoot
	}
	if effectiveRoot == "" {
		return nil, fmt.Errorf("magicform channel requires workspace_root to be configured " +
			"(set channels.magicform.workspace_root or agents.defaults.workspace_root)")
	}

	ctx, cancel := context.WithCancel(context.Background())

	ch := &MagicFormChannel{
		BaseChannel:   base,
		settings:      settings,
		workspaceRoot: effectiveRoot,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		ctx:    ctx,
		cancel: cancel,
	}

	base.SetOwner(ch)
	return ch, nil
}

// Start begins the channel and starts the TTL cleanup goroutine.
func (c *MagicFormChannel) Start(_ context.Context) error {
	c.SetRunning(true)

	// Background goroutine to clean up stale request contexts
	go c.cleanupLoop()

	logger.InfoCF("magicform", "MagicForm channel started", nil)
	return nil
}

// Stop shuts down the channel.
func (c *MagicFormChannel) Stop(_ context.Context) error {
	c.cancel()
	c.SetRunning(false)
	logger.InfoCF("magicform", "MagicForm channel stopped", nil)
	return nil
}

// WebhookPath returns the HTTP path for the inbound webhook.
func (c *MagicFormChannel) WebhookPath() string {
	if c.settings.WebhookPath != "" {
		return c.settings.WebhookPath
	}
	return "/hooks/magicform"
}

// HealthPath returns the HTTP path for the health check endpoint.
func (c *MagicFormChannel) HealthPath() string {
	return "/health/magicform"
}

// HealthHandler handles the health check HTTP request.
func (c *MagicFormChannel) HealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"channel": "magicform",
	})
}

// maxWebhookBodySize is the maximum allowed size for inbound webhook payloads (1 MB).
const maxWebhookBodySize = 1 << 20

// ServeHTTP handles inbound webhook requests from MagicForm.
func (c *MagicFormChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size to prevent memory exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodySize)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	defer r.Body.Close()

	// Verify Bearer token
	if !c.verifyToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if payload.StackID == "" || payload.ConversationID == "" || payload.Message == "" {
		http.Error(w, "Missing required fields: stackId, conversationId, message", http.StatusBadRequest)
		return
	}

	// Validate workspace path: must be relative and resolve under workspace_root
	if payload.Workspace != "" {
		resolved, err := c.resolveWorkspace(payload.Workspace)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid workspace: %v", err), http.StatusBadRequest)
			return
		}
		payload.Workspace = resolved
	}

	// Validate configDir path against workspace_root
	if payload.ConfigDir != "" {
		resolved, err := c.resolveWorkspace(payload.ConfigDir)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid configDir: %v", err), http.StatusBadRequest)
			return
		}
		payload.ConfigDir = resolved
	}

	// Return 200 immediately, process asynchronously
	w.WriteHeader(http.StatusOK)

	go c.processWebhook(c.ctx, payload)
}

// resolveWorkspace validates and resolves the workspace path using the shared
// pathutil.ResolveWorkspacePath boundary check. The effective workspace root is
// determined at construction time (channel-level config with global fallback).
func (c *MagicFormChannel) resolveWorkspace(workspace string) (string, error) {
	return pathutil.ResolveWorkspacePath(c.workspaceRoot, workspace)
}

// verifyToken checks the Authorization Bearer token using constant-time comparison.
func (c *MagicFormChannel) verifyToken(r *http.Request) bool {
	configured := c.settings.Token.String()
	if configured == "" {
		return true // No token configured = allow all (dev mode)
	}

	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	return subtle.ConstantTimeCompare([]byte(token), []byte(configured)) == 1
}

// processWebhook handles an inbound webhook payload asynchronously.
func (c *MagicFormChannel) processWebhook(ctx context.Context, p WebhookPayload) {
	// Check for shutdown before doing any work
	if ctx.Err() != nil {
		logger.WarnCF("magicform", "Skipping webhook processing: channel shutting down",
			map[string]any{"stack_id": p.StackID, "conversation_id": p.ConversationID})
		return
	}

	chatID := "magicform:" + p.ConversationID
	senderID := p.UserID
	if senderID == "" {
		senderID = "anonymous"
	}

	// Store request context for Send() to look up later
	c.requests.Store(chatID, &requestContext{
		stackID:        p.StackID,
		conversationID: p.ConversationID,
		userID:         p.UserID,
		callbackURL:    p.CallbackURL,
		createdAt:      time.Now(),
	})

	sender := bus.SenderInfo{
		Platform:    "magicform",
		PlatformID:  senderID,
		CanonicalID: "magicform:" + senderID,
	}

	// Session key: per-stack per-conversation isolation
	sessionKey := fmt.Sprintf("agent:main:magicform:%s:%s", p.StackID, p.ConversationID)

	// Stash tenant routing + multi-tenancy hints in Context.Raw — the agent loop
	// reads workspace_override / config_dir / allowed_tools / allowed_skills from here.
	raw := map[string]string{
		"platform":        "magicform",
		"stack_id":        p.StackID,
		"conversation_id": p.ConversationID,
	}
	if p.CallbackURL != "" {
		raw["callback_url"] = p.CallbackURL
	}
	if p.Workspace != "" {
		raw["workspace_override"] = p.Workspace
	}
	if p.ConfigDir != "" {
		raw["config_dir"] = p.ConfigDir
	}
	if len(p.AllowedTools) > 0 {
		raw["allowed_tools"] = strings.Join(trimSlice(p.AllowedTools), ",")
	}
	if len(p.AllowedSkills) > 0 {
		raw["allowed_skills"] = strings.Join(trimSlice(p.AllowedSkills), ",")
	}

	messageID := fmt.Sprintf("mf-%s-%d", p.ConversationID, time.Now().UnixMilli())

	inboundCtx := bus.InboundContext{
		Channel:   "magicform",
		ChatID:    chatID,
		ChatType:  "direct",
		SpaceID:   p.StackID,
		SpaceType: "tenant",
		SenderID:  sender.CanonicalID,
		MessageID: messageID,
		Raw:       raw,
	}

	// Build InboundMessage directly (not via HandleMessage) to set SessionKey.
	// MagicForm is API-to-API, so typing/reaction/placeholder don't apply.
	msg := bus.InboundMessage{
		Context:    inboundCtx,
		Sender:     sender,
		Content:    p.Message,
		SessionKey: sessionKey,
		Channel:    "magicform",
		SenderID:   sender.CanonicalID,
		ChatID:     chatID,
		MessageID:  messageID,
	}

	if err := c.Bus().PublishInbound(ctx, msg); err != nil {
		logger.ErrorCF("magicform", "Failed to publish inbound message",
			map[string]any{
				"chat_id":         chatID,
				"stack_id":        p.StackID,
				"conversation_id": p.ConversationID,
				"error":           err.Error(),
			})
	}
}

// trimSlice trims whitespace from each element in the slice.
func trimSlice(s []string) []string {
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = strings.TrimSpace(v)
	}
	return out
}

// Send delivers the agent response back to MagicForm via HTTP callback.
func (c *MagicFormChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	// For progress/escalation messages, Load (keep) the context since the final
	// message is still coming. For final messages, LoadAndDelete to clean up.
	isFinal := msg.Type == bus.MessageTypeFinal
	var reqCtx *requestContext
	if isFinal {
		val, ok := c.requests.LoadAndDelete(msg.ChatID)
		if !ok {
			return nil, fmt.Errorf("%w: no request context for chatID %s", channels.ErrSendFailed, msg.ChatID)
		}
		reqCtx = val.(*requestContext)
	} else {
		val, ok := c.requests.Load(msg.ChatID)
		if !ok {
			return nil, fmt.Errorf("%w: no request context for chatID %s", channels.ErrSendFailed, msg.ChatID)
		}
		reqCtx = val.(*requestContext)
	}

	// Resolve callback URL
	callbackURL := reqCtx.callbackURL
	if callbackURL == "" {
		callbackURL = c.settings.BackendURL + "/claw-agent/callback"
	}

	if callbackURL == "" {
		return nil, fmt.Errorf("%w: no callback URL available", channels.ErrSendFailed)
	}

	// Build callback payload
	taskID := fmt.Sprintf("claw_task_%s_%d", reqCtx.conversationID, reqCtx.createdAt.UnixMilli())
	payload := CallbackPayload{
		StackID:        reqCtx.stackID,
		ConversationID: reqCtx.conversationID,
		TaskID:         taskID,
		Runtime:        "picoclaw",
		Response:       msg.Content,
	}

	switch msg.Type {
	case bus.MessageTypeProgress:
		payload.Type = "progress"
		payload.Status = "success"
		if msg.Progress != nil {
			payload.Progress = &ProgressPayload{
				Status:     msg.Progress.Status,
				ToolName:   msg.Progress.ToolName,
				StepNumber: msg.Progress.StepNumber,
				Message:    msg.Progress.Message,
			}
		}
	case bus.MessageTypeEscalation:
		payload.Type = "escalation"
		payload.Status = "success"
		if msg.Escalation != nil {
			payload.Escalation = &EscalationPayload{
				Reason: msg.Escalation.Reason,
				Notes:  msg.Escalation.Notes,
			}
		}
	default: // final
		payload.Type = "final"
		payload.Status = "success"
		if m := msg.Metrics; m != nil {
			payload.DurationMs = m.DurationMs
			payload.ToolCalls = m.ToolCalls
			payload.TokenUsage = m.TokenUsage
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal callback payload: %v", channels.ErrSendFailed, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: create callback request: %v", channels.ErrSendFailed, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if tok := c.settings.Token.String(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, channels.ClassifyNetError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, channels.ClassifySendError(resp.StatusCode, fmt.Errorf("callback error: %s", respBody))
	}

	logger.InfoCF("magicform", "Callback sent",
		map[string]any{
			"conversation_id": reqCtx.conversationID,
			"type":            payload.Type,
			"status":          resp.StatusCode,
		})

	return nil, nil
}

// cleanupLoop periodically removes stale request contexts.
func (c *MagicFormChannel) cleanupLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.requests.Range(func(key, value any) bool {
				rc := value.(*requestContext)
				if time.Since(rc.createdAt) > 10*time.Minute {
					c.requests.Delete(key)
					logger.DebugCF("magicform", "Cleaned up stale request context",
						map[string]any{"chat_id": key})
				}
				return true
			})
		}
	}
}
