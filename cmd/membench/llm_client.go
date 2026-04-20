package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// LLMClient wraps an OpenAI-compatible chat completion endpoint.
type LLMClient struct {
	BaseURL    string
	Model      string
	APIKey     string
	NoThinking bool // send chat_template_kwargs to disable thinking (llama.cpp specific)
	MaxRetries int  // max retry attempts for transient errors (0 = no retry)
	Client     *http.Client
}

// LLMClientOptions configures the LLM client.
type LLMClientOptions struct {
	BaseURL    string
	Model      string
	APIKey     string
	Timeout    time.Duration
	NoThinking bool
	MaxRetries int // max retry attempts (default 3)
}

// NewLLMClient creates a client for an OpenAI-compatible chat completion API.
func NewLLMClient(opts LLMClientOptions) *LLMClient {
	if opts.Timeout == 0 {
		opts.Timeout = 120 * time.Second
	}
	maxRetries := opts.MaxRetries
	if maxRetries < 0 {
		maxRetries = 3
	}
	return &LLMClient{
		BaseURL:    strings.TrimRight(opts.BaseURL, "/"),
		Model:      opts.Model,
		APIKey:     opts.APIKey,
		NoThinking: opts.NoThinking,
		MaxRetries: maxRetries,
		Client: &http.Client{
			Timeout: opts.Timeout,
		},
	}
}

type chatRequest struct {
	Model              string         `json:"model"`
	Messages           []chatMessage  `json:"messages"`
	Temperature        float64        `json:"temperature"`
	MaxTokens          int            `json:"max_tokens"`
	ChatTemplateKwargs map[string]any `json:"chat_template_kwargs,omitempty"` // llama.cpp
	Think              *bool          `json:"think,omitempty"`                // Ollama
	Thinking           map[string]any `json:"thinking,omitempty"`             // GLM (智谱)
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content,omitempty"`
		} `json:"message"`
	} `json:"choices"`
}

// Complete sends a chat completion request and returns the assistant's reply.
func (c *LLMClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	sysContent := systemPrompt
	if c.NoThinking && sysContent != "" {
		// Prepend /no_think tag — works with Ollama /v1 endpoint and
		// Qwen chat templates where the JSON think field is ignored.
		sysContent = "/no_think\n" + sysContent
	}
	messages := []chatMessage{}
	if sysContent != "" {
		messages = append(messages, chatMessage{Role: "system", Content: sysContent})
	}
	messages = append(messages, chatMessage{Role: "user", Content: userPrompt})

	body := chatRequest{
		Model:       c.Model,
		Messages:    messages,
		Temperature: 0.1,
		MaxTokens:   512,
	}
	if c.NoThinking {
		// llama.cpp: chat_template_kwargs
		body.ChatTemplateKwargs = map[string]any{
			"enable_thinking": false,
		}
		// Ollama (0.9+): think field
		thinkFalse := false
		body.Think = &thinkFalse
		// GLM (智谱): thinking field
		body.Thinking = map[string]any{
			"type": "disabled",
		}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	endpoint := strings.TrimRight(c.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	var respBody []byte
	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<(attempt-1)) * time.Second // 1s, 2s, 4s, ...
			log.Printf("LLM retry %d/%d after %v: %v", attempt, c.MaxRetries, backoff, lastErr)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
			// Rebuild request (body reader is consumed)
			req, err = http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
			if err != nil {
				return "", fmt.Errorf("create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")
			if c.APIKey != "" {
				req.Header.Set("Authorization", "Bearer "+c.APIKey)
			}
		}

		var resp *http.Response
		resp, lastErr = c.Client.Do(req)
		if lastErr != nil {
			continue // network/timeout error → retry
		}

		respBody, lastErr = io.ReadAll(resp.Body)
		resp.Body.Close()
		if lastErr != nil {
			continue
		}

		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
			continue // rate limit or server error → retry
		}
		if resp.StatusCode != 200 {
			return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
		}

		lastErr = nil
		break
	}
	if lastErr != nil {
		return "", fmt.Errorf("after %d retries: %w", c.MaxRetries, lastErr)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	// Strip any residual <think>...</think> blocks
	if idx := strings.Index(content, "</think>"); idx >= 0 {
		content = strings.TrimSpace(content[idx+len("</think>"):])
	}
	// Fallback: GLM/DeepSeek put thinking output in reasoning_content when thinking is enabled
	if content == "" && chatResp.Choices[0].Message.ReasoningContent != "" {
		content = strings.TrimSpace(chatResp.Choices[0].Message.ReasoningContent)
	}
	if content == "" {
		return "", fmt.Errorf("empty LLM response")
	}
	return content, nil
}
