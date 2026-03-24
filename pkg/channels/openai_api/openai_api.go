package openai_api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

const (
	maxRequestBodySize  = 4 << 20
	responseIdleWindow  = 250 * time.Millisecond
	responseWaitTimeout = 5 * time.Minute
)

var defaultAllowedOrigins = []string{"localhost"}

type responseTask struct {
	ctx     context.Context
	cancel  context.CancelFunc
	updates chan string
}

type chatCompletionRequest struct {
	Model    string                  `json:"model"`
	Messages []chatCompletionMessage `json:"messages"`
	Stream   bool                    `json:"stream,omitempty"`
	User     string                  `json:"user,omitempty"`
}

type chatCompletionMessage struct {
	Role       string         `json:"role"`
	Content    any            `json:"content"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
}

type chatToolCall struct {
	ID       string            `json:"id,omitempty"`
	Type     string            `json:"type,omitempty"`
	Function *chatToolFunction `json:"function,omitempty"`
}

type chatToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type translatedConversation struct {
	CurrentMessage    string
	InjectedHistory   []providers.Message
	ExtraSystemPrompt string
}

type OpenAIAPIChannel struct {
	*channels.BaseChannel
	config         config.OpenAIAPIConfig
	allowedOrigins []string
	listenHost     string
	models         []config.ModelConfig
	messageBus     *bus.MessageBus
	server         *http.Server
	listener       net.Listener
	ctx            context.Context
	cancel         context.CancelFunc
	taskMu         sync.RWMutex
	tasks          map[string]*responseTask
}

func NewOpenAIAPIChannel(cfg *config.Config, messageBus *bus.MessageBus) (*OpenAIAPIChannel, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if strings.TrimSpace(cfg.Channels.OpenAIAPI.APIKey) == "" {
		return nil, fmt.Errorf("openai_api api_key is required")
	}

	channelConfig := cfg.Channels.OpenAIAPI
	channelConfig.AllowOrigins = normalizeAllowedOrigins(channelConfig.AllowOrigins)

	listenHost := strings.TrimSpace(cfg.Gateway.Host)
	if listenHost == "" {
		listenHost = "127.0.0.1"
	}

	base := channels.NewBaseChannel("openai_api", channelConfig, messageBus, nil)

	return &OpenAIAPIChannel{
		BaseChannel:    base,
		config:         channelConfig,
		allowedOrigins: append([]string(nil), channelConfig.AllowOrigins...),
		listenHost:     listenHost,
		models:         append([]config.ModelConfig(nil), cfg.ModelList...),
		messageBus:     messageBus,
		tasks:          make(map[string]*responseTask),
	}, nil
}

func (c *OpenAIAPIChannel) Start(ctx context.Context) error {
	addr := net.JoinHostPort(c.listenHost, strconv.Itoa(c.config.Port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("openai_api listen %s: %w", addr, err)
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.listener = listener

	mux := http.NewServeMux()
	mux.HandleFunc("OPTIONS /v1/models", c.handleOptions)
	mux.HandleFunc("GET /v1/models", c.handleModels)
	mux.HandleFunc("OPTIONS /v1/chat/completions", c.handleOptions)
	mux.HandleFunc("POST /v1/chat/completions", c.handleChatCompletions)
	mux.HandleFunc("GET /health", c.handleHealth)

	c.server = &http.Server{
		Handler:     mux,
		ReadTimeout: 30 * time.Second,
		// Streaming responses stay open until the request finishes.
		WriteTimeout: 0,
	}

	c.SetRunning(true)
	logger.InfoCF("openai_api", "OpenAI API channel listening", map[string]any{
		"addr": addr,
	})

	go func() {
		if err := c.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.ErrorCF("openai_api", "OpenAI API server stopped unexpectedly", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	return nil
}

func (c *OpenAIAPIChannel) Stop(ctx context.Context) error {
	c.SetRunning(false)

	if c.cancel != nil {
		c.cancel()
	}

	c.taskMu.Lock()
	for chatID, task := range c.tasks {
		if task.cancel != nil {
			task.cancel()
		}
		delete(c.tasks, chatID)
	}
	c.taskMu.Unlock()

	if c.server != nil {
		if err := c.server.Shutdown(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (c *OpenAIAPIChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return nil
	}

	task := c.getTask(msg.ChatID)
	if task == nil {
		logger.DebugCF("openai_api", "Dropping outbound response with no waiting request", map[string]any{
			"chat_id": msg.ChatID,
		})
		return nil
	}

	select {
	case task.updates <- content:
		return nil
	case <-task.ctx.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *OpenAIAPIChannel) handleOptions(w http.ResponseWriter, r *http.Request) {
	if !c.applyCORSHeaders(w, r) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *OpenAIAPIChannel) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !c.applyCORSHeaders(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := writeJSONResponse(w, http.StatusOK, map[string]any{"status": "ok"}); err != nil {
		return
	}
}

func (c *OpenAIAPIChannel) handleModels(w http.ResponseWriter, r *http.Request) {
	if !c.applyCORSHeaders(w, r) {
		return
	}
	if !c.authenticate(r) {
		if err := writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "invalid_api_key"); err != nil {
			return
		}
		return
	}

	type modelObject struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}

	seen := make(map[string]bool)
	items := make([]modelObject, 0, len(c.models))
	for _, model := range c.models {
		id := strings.TrimSpace(model.ModelName)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		items = append(items, modelObject{
			ID:      id,
			Object:  "model",
			Created: 0,
			OwnedBy: "picoclaw",
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	w.Header().Set("Content-Type", "application/json")
	if err := writeJSONResponse(w, http.StatusOK, map[string]any{
		"object": "list",
		"data":   items,
	}); err != nil {
		return
	}
}

func (c *OpenAIAPIChannel) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if !c.applyCORSHeaders(w, r) {
		return
	}
	if !c.authenticate(r) {
		if err := writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "invalid_api_key"); err != nil {
			return
		}
		return
	}

	var req chatCompletionRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodySize))
	if err := decoder.Decode(&req); err != nil {
		logger.WarnCF("openai_api", "Invalid chat completion request body", map[string]any{
			"error": err.Error(),
		})
		if err := writeOpenAIError(w, http.StatusBadRequest, "Invalid JSON request body", "invalid_request_error", "invalid_json"); err != nil {
			return
		}
		return
	}

	if strings.TrimSpace(req.Model) == "" {
		if err := writeOpenAIError(w, http.StatusBadRequest, "model is required", "invalid_request_error", "missing_model"); err != nil {
			return
		}
		return
	}
	if !c.supportsModel(req.Model) {
		if err := writeOpenAIError(w, http.StatusBadRequest, fmt.Sprintf("model %q is not configured", req.Model), "invalid_request_error", "model_not_found"); err != nil {
			return
		}
		return
	}
	if len(req.Messages) == 0 {
		if err := writeOpenAIError(w, http.StatusBadRequest, "messages must not be empty", "invalid_request_error", "missing_messages"); err != nil {
			return
		}
		return
	}

	translated, err := translateConversation(req.Messages)
	if err != nil {
		logger.WarnCF("openai_api", "Invalid chat completion message sequence", map[string]any{
			"error": err.Error(),
		})
		if err := writeOpenAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "invalid_messages"); err != nil {
			return
		}
		return
	}

	reqCtx, cancel := context.WithTimeout(r.Context(), responseWaitTimeout)
	defer cancel()

	chatID := "openai_api:" + uuid.NewString()
	task := &responseTask{
		ctx:     reqCtx,
		cancel:  cancel,
		updates: make(chan string, 32),
	}
	c.setTask(chatID, task)
	defer c.deleteTask(chatID)

	metadata := map[string]string{
		"no_history":      "true",
		"requested_model": req.Model,
	}
	if strings.TrimSpace(translated.ExtraSystemPrompt) != "" {
		metadata["extra_system_prompt"] = translated.ExtraSystemPrompt
	}
	if len(translated.InjectedHistory) > 0 {
		rawHistory, err := json.Marshal(translated.InjectedHistory)
		if err != nil {
			logger.ErrorCF("openai_api", "Failed to encode injected history", map[string]any{
				"error": err.Error(),
			})
			if err := writeOpenAIError(w, http.StatusInternalServerError, "Failed to encode conversation history", "server_error", "history_encode_failed"); err != nil {
				return
			}
			return
		}
		metadata["injected_history"] = string(rawHistory)
	}

	senderID := strings.TrimSpace(req.User)
	if senderID == "" {
		senderID = "openai-client"
	}

	if err := c.messageBus.PublishInbound(reqCtx, bus.InboundMessage{
		Channel:  c.Name(),
		SenderID: senderID,
		Sender: bus.SenderInfo{
			Platform:    c.Name(),
			PlatformID:  senderID,
			CanonicalID: c.Name() + ":" + senderID,
			DisplayName: senderID,
		},
		ChatID:   chatID,
		Content:  translated.CurrentMessage,
		Peer:     bus.Peer{Kind: "direct", ID: senderID},
		Metadata: metadata,
	}); err != nil {
		logger.ErrorCF("openai_api", "Failed to publish inbound OpenAI API request", map[string]any{
			"error": err.Error(),
		})
		if err := writeOpenAIError(w, http.StatusBadGateway, fmt.Sprintf("Failed to submit request: %v", err), "server_error", "publish_failed"); err != nil {
			return
		}
		return
	}

	firstChunk, err := waitForFirstChunk(reqCtx, task)
	if err != nil {
		logger.ErrorCF("openai_api", "Timed out waiting for first assistant chunk", map[string]any{
			"error": err.Error(),
		})
		if err := writeOpenAIError(w, http.StatusGatewayTimeout, "Timed out waiting for assistant response", "server_error", "response_timeout"); err != nil {
			return
		}
		return
	}

	completionID := "chatcmpl_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	createdAt := time.Now().Unix()

	if req.Stream {
		flusher, ok := w.(http.Flusher)
		if !ok {
			if err := writeOpenAIError(w, http.StatusInternalServerError, "Streaming is not supported by this server", "server_error", "stream_not_supported"); err != nil {
				return
			}
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		if err := writeChatCompletionChunk(w, completionID, createdAt, req.Model, firstChunk, true, false); err != nil {
			logger.ErrorCF("openai_api", "Failed to write first streaming chunk", map[string]any{
				"error": err.Error(),
			})
			return
		}
		flusher.Flush()

		if err := streamRemainingChunks(reqCtx, task, w, flusher, completionID, createdAt, req.Model); err != nil {
			logger.ErrorCF("openai_api", "Failed to stream assistant chunks", map[string]any{
				"error": err.Error(),
			})
			return
		}

		if err := writeChatCompletionChunk(w, completionID, createdAt, req.Model, "", false, true); err != nil {
			logger.ErrorCF("openai_api", "Failed to write final streaming chunk", map[string]any{
				"error": err.Error(),
			})
			return
		}
		if _, err := fmt.Fprint(w, "data: [DONE]\n\n"); err != nil {
			logger.ErrorCF("openai_api", "Failed to write stream terminator", map[string]any{
				"error": err.Error(),
			})
			return
		}
		flusher.Flush()
		return
	}

	chunks, err := collectRemainingChunks(reqCtx, task, []string{firstChunk})
	if err != nil {
		logger.ErrorCF("openai_api", "Timed out collecting assistant response chunks", map[string]any{
			"error": err.Error(),
		})
		if err := writeOpenAIError(w, http.StatusGatewayTimeout, "Timed out waiting for assistant response", "server_error", "response_timeout"); err != nil {
			return
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := writeJSONResponse(w, http.StatusOK, map[string]any{
		"id":      completionID,
		"object":  "chat.completion",
		"created": createdAt,
		"model":   req.Model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": strings.Join(chunks, "\n\n"),
				},
				"finish_reason": "stop",
			},
		},
	}); err != nil {
		return
	}
}

func translateConversation(messages []chatCompletionMessage) (translatedConversation, error) {
	var out translatedConversation
	var nonSystem []providers.Message
	var systemPrompts []string

	for _, message := range messages {
		role := strings.ToLower(strings.TrimSpace(message.Role))
		content := strings.TrimSpace(extractMessageContent(message))

		switch role {
		case "system", "developer":
			if content != "" {
				systemPrompts = append(systemPrompts, content)
			}
		case "user", "assistant", "tool":
			pm := providers.Message{
				Role:    role,
				Content: content,
			}
			if role == "tool" {
				pm.ToolCallID = strings.TrimSpace(message.ToolCallID)
			}
			nonSystem = append(nonSystem, pm)
		default:
			if content == "" {
				continue
			}
			nonSystem = append(nonSystem, providers.Message{
				Role:    "user",
				Content: fmt.Sprintf("[%s]\n%s", role, content),
			})
		}
	}

	if len(nonSystem) == 0 {
		return translatedConversation{}, fmt.Errorf("at least one non-system message is required")
	}

	out.ExtraSystemPrompt = strings.Join(systemPrompts, "\n\n")
	last := nonSystem[len(nonSystem)-1]

	switch last.Role {
	case "assistant":
		return translatedConversation{}, fmt.Errorf("last message must be a user message, got assistant")
	case "tool":
		return translatedConversation{}, fmt.Errorf("last message must be a user message, got tool")
	}
	if strings.TrimSpace(last.Content) == "" {
		return translatedConversation{}, fmt.Errorf("last user message must not be empty")
	}
	out.CurrentMessage = last.Content
	out.InjectedHistory = append([]providers.Message(nil), nonSystem[:len(nonSystem)-1]...)
	return out, nil
}

func extractMessageContent(message chatCompletionMessage) string {
	content := strings.TrimSpace(extractContentText(message.Content))
	toolCalls := strings.TrimSpace(renderToolCalls(message.ToolCalls))

	switch {
	case content == "":
		return toolCalls
	case toolCalls == "":
		return content
	default:
		return content + "\n\n" + toolCalls
	}
}

func renderToolCalls(toolCalls []chatToolCall) string {
	if len(toolCalls) == 0 {
		return ""
	}

	lines := make([]string, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		if toolCall.Function == nil {
			continue
		}
		name := strings.TrimSpace(toolCall.Function.Name)
		args := strings.TrimSpace(toolCall.Function.Arguments)
		if name == "" {
			continue
		}
		if args == "" {
			lines = append(lines, fmt.Sprintf("[assistant_tool_call] %s()", name))
			continue
		}
		lines = append(lines, fmt.Sprintf("[assistant_tool_call] %s(%s)", name, args))
	}
	return strings.Join(lines, "\n")
}

func extractContentText(content any) string {
	switch value := content.(type) {
	case nil:
		return ""
	case string:
		return value
	case []any:
		parts := make([]string, 0, len(value))
		for _, rawPart := range value {
			part, ok := rawPart.(map[string]any)
			if !ok {
				continue
			}
			partType, _ := part["type"].(string)
			switch partType {
			case "text", "input_text", "output_text":
				if text, ok := stringValue(part["text"]); ok && text != "" {
					parts = append(parts, text)
					continue
				}
				if text, ok := stringValue(part["input_text"]); ok && text != "" {
					parts = append(parts, text)
					continue
				}
				if text, ok := stringValue(part["output_text"]); ok && text != "" {
					parts = append(parts, text)
				}
			case "image_url", "input_image":
				if url := extractImageURL(part["image_url"]); url != "" {
					parts = append(parts, "[image] "+url)
				}
			default:
				if text, ok := stringValue(part["text"]); ok && text != "" {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func extractImageURL(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		if url, ok := stringValue(v["url"]); ok {
			return url
		}
	}
	return ""
}

func stringValue(value any) (string, bool) {
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	return text, true
}

func waitForFirstChunk(ctx context.Context, task *responseTask) (string, error) {
	for {
		select {
		case chunk := <-task.updates:
			if strings.TrimSpace(chunk) != "" {
				return chunk, nil
			}
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

func collectRemainingChunks(ctx context.Context, task *responseTask, initial []string) ([]string, error) {
	chunks := append([]string(nil), initial...)
	timer := time.NewTimer(responseIdleWindow)
	defer timer.Stop()

	for {
		select {
		case chunk := <-task.updates:
			if strings.TrimSpace(chunk) == "" {
				continue
			}
			chunks = append(chunks, chunk)
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(responseIdleWindow)
		case <-timer.C:
			return chunks, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func streamRemainingChunks(
	ctx context.Context,
	task *responseTask,
	w http.ResponseWriter,
	flusher http.Flusher,
	completionID string,
	createdAt int64,
	model string,
) error {
	timer := time.NewTimer(responseIdleWindow)
	defer timer.Stop()

	for {
		select {
		case chunk := <-task.updates:
			if strings.TrimSpace(chunk) == "" {
				continue
			}
			if err := writeChatCompletionChunk(w, completionID, createdAt, model, chunk, false, false); err != nil {
				return err
			}
			flusher.Flush()
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(responseIdleWindow)
		case <-timer.C:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func writeChatCompletionChunk(
	w http.ResponseWriter,
	completionID string,
	createdAt int64,
	model string,
	content string,
	includeRole bool,
	finished bool,
) error {
	delta := map[string]any{}
	if includeRole {
		delta["role"] = "assistant"
	}
	if content != "" {
		delta["content"] = content
	}

	finishReason := any(nil)
	if finished {
		finishReason = "stop"
	}

	payload, err := json.Marshal(map[string]any{
		"id":      completionID,
		"object":  "chat.completion.chunk",
		"created": createdAt,
		"model":   model,
		"choices": []map[string]any{
			{
				"index":         0,
				"delta":         delta,
				"finish_reason": finishReason,
			},
		},
	})
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "data: %s\n\n", payload)
	return err
}

func writeOpenAIError(w http.ResponseWriter, status int, message, errorType, code string) error {
	w.Header().Set("Content-Type", "application/json")
	return writeJSONResponse(w, status, map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    errorType,
			"code":    code,
		},
	})
}

func writeJSONResponse(w http.ResponseWriter, status int, payload any) error {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logger.ErrorCF("openai_api", "Failed to write JSON response", map[string]any{
			"error": err.Error(),
		})
		return err
	}
	return nil
}

func normalizeAllowedOrigins(origins []string) []string {
	normalized := make([]string, 0, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		normalized = append(normalized, origin)
	}
	if len(normalized) == 0 {
		return append([]string(nil), defaultAllowedOrigins...)
	}
	return normalized
}

func (c *OpenAIAPIChannel) applyCORSHeaders(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	if !originAllowed(c.allowedOrigins, origin) {
		logger.WarnCF("openai_api", "Rejected request from disallowed origin", map[string]any{
			"origin": origin,
		})
		if err := writeOpenAIError(w, http.StatusForbidden, "Origin is not allowed", "invalid_request_error", "origin_not_allowed"); err != nil {
			return false
		}
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	addVaryHeader(w.Header(), "Origin")
	return true
}

func originAllowed(allowedOrigins []string, origin string) bool {
	parsedOrigin, err := url.Parse(origin)
	if err != nil || parsedOrigin.Scheme == "" || parsedOrigin.Host == "" {
		return false
	}

	requestHost := strings.ToLower(parsedOrigin.Hostname())
	for _, allowed := range allowedOrigins {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}

		if strings.Contains(allowed, "://") {
			parsedAllowed, err := url.Parse(allowed)
			if err != nil || parsedAllowed.Scheme == "" || parsedAllowed.Host == "" {
				continue
			}
			if strings.EqualFold(parsedAllowed.Scheme, parsedOrigin.Scheme) && strings.EqualFold(parsedAllowed.Host, parsedOrigin.Host) {
				return true
			}
			continue
		}

		if strings.EqualFold(allowed, requestHost) {
			return true
		}
	}

	return false
}

func addVaryHeader(headers http.Header, value string) {
	for _, existing := range headers.Values("Vary") {
		for _, part := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) {
				return
			}
		}
	}
	headers.Add("Vary", value)
}

func (c *OpenAIAPIChannel) authenticate(r *http.Request) bool {
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if token == "" {
		return false
	}
	expected := strings.TrimSpace(c.config.APIKey)
	if expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

func (c *OpenAIAPIChannel) supportsModel(requested string) bool {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return false
	}

	for _, model := range c.models {
		if strings.TrimSpace(model.ModelName) == requested {
			return true
		}
		if strings.TrimSpace(model.Model) == requested {
			return true
		}
		_, modelID := providers.ExtractProtocol(model.Model)
		if modelID == requested {
			return true
		}
	}

	return false
}

func (c *OpenAIAPIChannel) setTask(chatID string, task *responseTask) {
	c.taskMu.Lock()
	defer c.taskMu.Unlock()
	c.tasks[chatID] = task
}

func (c *OpenAIAPIChannel) getTask(chatID string) *responseTask {
	c.taskMu.RLock()
	defer c.taskMu.RUnlock()
	return c.tasks[chatID]
}

func (c *OpenAIAPIChannel) deleteTask(chatID string) {
	c.taskMu.Lock()
	defer c.taskMu.Unlock()
	if task, ok := c.tasks[chatID]; ok && task.cancel != nil {
		task.cancel()
	}
	delete(c.tasks, chatID)
}
