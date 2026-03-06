package magicform

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// WebhookPayload is the inbound payload from MagicForm.
type WebhookPayload struct {
	StackID        string `json:"stackId"`
	ConversationID string `json:"conversationId"`
	UserID         string `json:"userId"`
	Message        string `json:"message"`
	Workspace      string `json:"workspace"`            // e.g. "s1/c1" — agent working directory (relative to workspace_root)
	ConfigDir      string `json:"configDir,omitempty"`   // e.g. "s1/config" — pre-provisioned config directory (relative to workspace_root)
	CallbackURL    string `json:"callbackUrl"`

	// Tool/skill filtering
	AllowedTools  []string `json:"allowedTools,omitempty"`  // Tool allowlist (empty = all)
	AllowedSkills []string `json:"allowedSkills,omitempty"` // Skill filter (empty = all)
}

// CallbackPayload is the outbound payload sent back to MagicForm.
type CallbackPayload struct {
	StackID        string `json:"stackId"`
	ConversationID string `json:"conversationId"`
	Response       string `json:"response"`
	Type           string `json:"type"` // "final"
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
	config     config.MagicFormConfig
	httpClient *http.Client
	requests   sync.Map // chatID → *requestContext
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewMagicFormChannel creates a new MagicForm channel.
func NewMagicFormChannel(cfg config.MagicFormConfig, msgBus *bus.MessageBus) (*MagicFormChannel, error) {
	base := channels.NewBaseChannel(
		"magicform",
		cfg,
		msgBus,
		cfg.AllowFrom,
	)

	ctx, cancel := context.WithCancel(context.Background())

	ch := &MagicFormChannel{
		BaseChannel: base,
		config:      cfg,
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
	if c.config.WebhookPath != "" {
		return c.config.WebhookPath
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

// resolveWorkspace validates and resolves the workspace path.
// If workspace_root is configured, the workspace must be a relative path that
// resolves under the root. If workspace_root is not configured, workspace is
// rejected (no arbitrary path writes allowed).
func (c *MagicFormChannel) resolveWorkspace(workspace string) (string, error) {
	root := c.config.WorkspaceRoot
	if root == "" {
		return "", fmt.Errorf("workspace_root not configured; workspace overrides are not allowed")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("invalid workspace_root: %w", err)
	}

	// Join the root with the provided workspace (which may be relative)
	resolved := filepath.Join(absRoot, workspace)
	resolved = filepath.Clean(resolved)

	// Ensure the resolved path is under the root (prevents ../../../etc traversal)
	if !strings.HasPrefix(resolved, absRoot+string(filepath.Separator)) && resolved != absRoot {
		return "", fmt.Errorf("workspace path escapes workspace_root")
	}

	return resolved, nil
}

// verifyToken checks the Authorization Bearer token using constant-time comparison.
func (c *MagicFormChannel) verifyToken(r *http.Request) bool {
	if c.config.Token == "" {
		return true // No token configured = allow all (dev mode)
	}

	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	return subtle.ConstantTimeCompare([]byte(token), []byte(c.config.Token)) == 1
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

	peer := bus.Peer{Kind: "direct", ID: p.ConversationID}
	sender := bus.SenderInfo{
		Platform:    "magicform",
		PlatformID:  senderID,
		CanonicalID: "magicform:" + senderID,
	}

	// Session key: per-stack per-conversation isolation
	sessionKey := fmt.Sprintf("agent:main:magicform:%s:%s", p.StackID, p.ConversationID)

	metadata := map[string]string{
		"platform":        "magicform",
		"stack_id":        p.StackID,
		"conversation_id": p.ConversationID,
	}

	if p.CallbackURL != "" {
		metadata["callback_url"] = p.CallbackURL
	}

	// Workspace override — agent loop will pick this up
	if p.Workspace != "" {
		metadata["workspace_override"] = p.Workspace
	}

	// Config directory — agent loop reads config.json and copies bootstrap files
	if p.ConfigDir != "" {
		metadata["config_dir"] = p.ConfigDir
	}

	// Tool/skill filtering — passed via metadata, picked up by agent loop
	if len(p.AllowedTools) > 0 {
		metadata["allowed_tools"] = strings.Join(trimSlice(p.AllowedTools), ",")
	}
	if len(p.AllowedSkills) > 0 {
		metadata["allowed_skills"] = strings.Join(trimSlice(p.AllowedSkills), ",")
	}

	messageID := fmt.Sprintf("mf-%s-%d", p.ConversationID, time.Now().UnixMilli())

	// Build InboundMessage directly (not via HandleMessage) to set SessionKey.
	// MagicForm is API-to-API, so typing/reaction/placeholder don't apply.
	msg := bus.InboundMessage{
		Channel:    "magicform",
		SenderID:   sender.CanonicalID,
		Sender:     sender,
		ChatID:     chatID,
		Content:    p.Message,
		Peer:       peer,
		MessageID:  messageID,
		SessionKey: sessionKey,
		Metadata:   metadata,
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
func (c *MagicFormChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	// Look up request context
	val, ok := c.requests.LoadAndDelete(msg.ChatID)
	if !ok {
		return fmt.Errorf("%w: no request context for chatID %s", channels.ErrSendFailed, msg.ChatID)
	}
	reqCtx := val.(*requestContext)

	// Resolve callback URL
	callbackURL := reqCtx.callbackURL
	if callbackURL == "" {
		callbackURL = c.config.BackendURL + "/claw-agent/callback"
	}

	if callbackURL == "" {
		return fmt.Errorf("%w: no callback URL available", channels.ErrSendFailed)
	}

	// Build callback payload
	payload := CallbackPayload{
		StackID:        reqCtx.stackID,
		ConversationID: reqCtx.conversationID,
		Response:       msg.Content,
		Type:           "final",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%w: marshal callback payload: %v", channels.ErrSendFailed, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%w: create callback request: %v", channels.ErrSendFailed, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return channels.ClassifyNetError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return channels.ClassifySendError(resp.StatusCode, fmt.Errorf("callback error: %s", respBody))
	}

	logger.InfoCF("magicform", "Callback sent",
		map[string]any{
			"conversation_id": reqCtx.conversationID,
			"status":          resp.StatusCode,
		})

	return nil
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

