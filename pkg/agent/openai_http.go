// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// openAIChatMessage is a single message in an OpenAI chat completion request.
type openAIChatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []contentPart
	Name    string `json:"name,omitempty"`
}

// openAIChatRequest is the JSON body for POST /v1/chat/completions.
type openAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
	User     string              `json:"user,omitempty"`
}

// extractOpenAITextContent extracts plain text from an OpenAI message content
// field, which may be a plain string or an array of typed content parts.
func extractOpenAITextContent(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var parts []string
		for _, part := range v {
			m, ok := part.(map[string]any)
			if !ok {
				continue
			}
			t, _ := m["type"].(string)
			switch t {
			case "text", "input_text":
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

// buildPromptFromOpenAIMessages extracts the latest user/tool message and an
// optional extra system prompt from an OpenAI-format messages array.
//
// The approach mirrors openclaw's buildAgentPrompt:
//   - Collect all system/developer messages into a combined extra system prompt.
//   - Find the last user or tool message to use as the actual query.
func buildPromptFromOpenAIMessages(messages []openAIChatMessage) (userMessage, extraSystemPrompt string) {
	var systemParts []string

	// Last user/tool message index (search backwards).
	lastUserIdx := -1
	for i := len(messages) - 1; i >= 0; i-- {
		role := strings.TrimSpace(messages[i].Role)
		if role == "user" || role == "tool" || role == "function" {
			content := strings.TrimSpace(extractOpenAITextContent(messages[i].Content))
			if content != "" {
				lastUserIdx = i
				break
			}
		}
	}

	if lastUserIdx >= 0 {
		userMessage = strings.TrimSpace(extractOpenAITextContent(messages[lastUserIdx].Content))
	}

	for _, msg := range messages {
		role := strings.TrimSpace(msg.Role)
		if role == "system" || role == "developer" {
			content := strings.TrimSpace(extractOpenAITextContent(msg.Content))
			if content != "" {
				systemParts = append(systemParts, content)
			}
		}
	}
	extraSystemPrompt = strings.Join(systemParts, "\n\n")
	return userMessage, extraSystemPrompt
}

// writeOpenAISSE writes one SSE event and flushes.
func writeOpenAISSE(w http.ResponseWriter, data any) {
	b, _ := json.Marshal(data)
	fmt.Fprintf(w, "data: %s\n\n", b)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// writeOpenAIDone writes the SSE [DONE] terminator.
func writeOpenAIDone(w http.ResponseWriter) {
	fmt.Fprint(w, "data: [DONE]\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// newOpenAIRunID generates a unique run/completion ID.
func newOpenAIRunID() string {
	//nolint:gosec // non-crypto random is fine for a run ID
	return fmt.Sprintf("chatcmpl-%d%d", time.Now().UnixNano(), rand.Int63n(100000))
}

// OpenAIChatHandlerConfig configures the /v1/chat/completions handler.
type OpenAIChatHandlerConfig struct {
	// Token is an optional shared secret. When non-empty every request must
	// carry "Authorization: Bearer <token>". Leave empty to allow all.
	Token string
}

// NewOpenAIChatHandler returns an http.HandlerFunc that exposes an
// OpenAI-compatible POST /v1/chat/completions endpoint.
//
// Features:
//   - Non-streaming: standard JSON response (chat.completion object).
//   - Streaming: SSE response (stream: true) — runs the agent synchronously
//     then writes the full reply as consecutive SSE chunks.
//   - Session routing: honour X-Picoclaw-Session-Key header; fall back to a
//     key derived from the OpenAI "user" field; otherwise a per-request key.
//   - Agent selection via the "model" field: "picoclaw:<agentId>" or
//     "agent:<agentId>" selects a named agent; any other value uses default.
func (al *AgentLoop) NewOpenAIChatHandler(cfg OpenAIChatHandlerConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST on the exact path.
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			sendOpenAIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
			return
		}

		// ── Authentication ─────────────────────────────────────────────────
		if cfg.Token != "" {
			auth := strings.TrimSpace(r.Header.Get("Authorization"))
			bearer := strings.TrimPrefix(auth, "Bearer ")
			if bearer != cfg.Token {
				sendOpenAIError(w, http.StatusUnauthorized, "authentication_error", "Unauthorized")
				return
			}
		}

		// ── Parse body ──────────────────────────────────────────────────────
		var req openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendOpenAIError(w, http.StatusBadRequest, "invalid_request_error",
				"invalid request body: "+err.Error())
			return
		}

		userMessage, _ := buildPromptFromOpenAIMessages(req.Messages)
		if userMessage == "" {
			sendOpenAIError(w, http.StatusBadRequest, "invalid_request_error",
				"Missing user message in `messages`.")
			return
		}

		// ── Session key ─────────────────────────────────────────────────────
		sessionKey := strings.TrimSpace(r.Header.Get("X-Picoclaw-Session-Key"))
		if sessionKey == "" && req.User != "" {
			// Stable key per OpenAI "user" value.
			sessionKey = "openai:user:" + req.User
		}
		if sessionKey == "" {
			// Stateless: unique key per request (no history sharing).
			sessionKey = fmt.Sprintf("openai:req:%d", time.Now().UnixNano())
		}

		// ── Model / run metadata ────────────────────────────────────────────
		model := req.Model
		if model == "" {
			model = "picoclaw"
		}
		runID := newOpenAIRunID()
		created := int64(time.Now().Unix())

		// ── Run the agent (synchronous) ────────────────────────────────────
		// Use "openai" as the channel so routing/history work naturally.
		response, err := al.ProcessDirectWithChannel(r.Context(), userMessage, sessionKey, "openai", "api")

		// ── Streaming response (SSE) ────────────────────────────────────────
		if req.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("X-Accel-Buffering", "no")
			w.WriteHeader(http.StatusOK)

			if err != nil {
				// Emit the error as content and finish cleanly.
				writeOpenAISSE(w, buildChunk(runID, model, created,
					map[string]any{"role": "assistant", "content": fmt.Sprintf("Error: %v", err)},
					ptrStr("stop")))
				writeOpenAIDone(w)
				return
			}

			// 1) role chunk
			writeOpenAISSE(w, buildChunk(runID, model, created,
				map[string]any{"role": "assistant"}, nil))

			// 2) content chunk
			writeOpenAISSE(w, buildChunk(runID, model, created,
				map[string]any{"content": response}, nil))

			// 3) finish chunk
			writeOpenAISSE(w, buildChunk(runID, model, created,
				map[string]any{}, ptrStr("stop")))

			writeOpenAIDone(w)
			return
		}

		// ── Non-streaming JSON response ─────────────────────────────────────
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"message": err.Error(),
					"type":    "api_error",
				},
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":      runID,
			"object":  "chat.completion",
			"created": created,
			"model":   model,
			"choices": []any{
				map[string]any{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": response,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     0,
				"completion_tokens": 0,
				"total_tokens":      0,
			},
		})
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func sendOpenAIError(w http.ResponseWriter, status int, errType, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"message": msg,
			"type":    errType,
		},
	})
}

// buildChunk constructs an SSE chat.completion.chunk payload.
func buildChunk(id, model string, created int64, delta map[string]any, finishReason *string) map[string]any {
	choice := map[string]any{
		"index": 0,
		"delta": delta,
	}
	if finishReason != nil {
		choice["finish_reason"] = *finishReason
	} else {
		choice["finish_reason"] = nil
	}
	return map[string]any{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []any{choice},
	}
}

func ptrStr(s string) *string { return &s }


