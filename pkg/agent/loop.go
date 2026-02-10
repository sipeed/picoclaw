// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type AgentLoop struct {
	bus            *bus.MessageBus
	provider       providers.LLMProvider
	workspace      string
	model          string
	contextWindow  int
	maxIterations  int
	streaming      bool
	sessions       *session.SessionManager
	contextBuilder *ContextBuilder
	tools          *tools.ToolRegistry
	running        bool
	summarizing    sync.Map
}

func NewAgentLoop(cfg *config.Config, bus *bus.MessageBus, provider providers.LLMProvider) *AgentLoop {
	workspace := cfg.WorkspacePath()
	os.MkdirAll(workspace, 0755)

	toolsRegistry := tools.NewToolRegistry()
	toolsRegistry.Register(&tools.ReadFileTool{})
	toolsRegistry.Register(&tools.WriteFileTool{})
	toolsRegistry.Register(&tools.ListDirTool{})
	toolsRegistry.Register(tools.NewExecTool(workspace))

	braveAPIKey := cfg.Tools.Web.Search.APIKey
	toolsRegistry.Register(tools.NewWebSearchTool(braveAPIKey, cfg.Tools.Web.Search.MaxResults))
	toolsRegistry.Register(tools.NewWebFetchTool(50000))

	sessionsManager := session.NewSessionManager(filepath.Join(filepath.Dir(cfg.WorkspacePath()), "sessions"))

	return &AgentLoop{
		bus:            bus,
		provider:       provider,
		workspace:      workspace,
		model:          cfg.Agents.Defaults.Model,
		contextWindow:  cfg.Agents.Defaults.MaxTokens,
		maxIterations:  cfg.Agents.Defaults.MaxToolIterations,
		streaming:      cfg.Agents.Defaults.Streaming,
		sessions:       sessionsManager,
		contextBuilder: NewContextBuilder(workspace),
		tools:          toolsRegistry,
		running:        false,
		summarizing:    sync.Map{},
	}
}

func (al *AgentLoop) Run(ctx context.Context) error {
	al.running = true

	// Start stream event forwarder only if streaming is enabled
	if al.streaming {
		go al.forwardStreamEvents(ctx)
	}

	for al.running {
		select {
		case <-ctx.Done():
			return nil
		default:
			msg, ok := al.bus.ConsumeInbound(ctx)
			if !ok {
				continue
			}

			var response string
			var err error

			// Choose streaming or non-streaming based on config
			if al.streaming {
				response, err = al.processMessageStreaming(ctx, msg, func(event bus.StreamEvent) {
					al.bus.PublishStreamEvent(event)
				})
			} else {
				response, err = al.processMessage(ctx, msg)
			}

			if err != nil {
				response = fmt.Sprintf("Error processing message: %v", err)
			}

			if response != "" {
				al.bus.PublishOutbound(bus.OutboundMessage{
					Channel: msg.Channel,
					ChatID:  msg.ChatID,
					Content: response,
				})
			}
		}
	}

	return nil
}

// forwardStreamEvents handles stream events and can be extended for logging
func (al *AgentLoop) forwardStreamEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Events are consumed by channels, this is just for future extensions
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (al *AgentLoop) Stop() {
	al.running = false
}

func (al *AgentLoop) ProcessDirect(ctx context.Context, content, sessionKey string) (string, error) {
	msg := bus.InboundMessage{
		Channel:    "cli",
		SenderID:   "user",
		ChatID:     "direct",
		Content:    content,
		SessionKey: sessionKey,
	}

	return al.processMessage(ctx, msg)
}

// ProcessDirectStreaming processes a message with streaming updates
func (al *AgentLoop) ProcessDirectStreaming(ctx context.Context, content, sessionKey string, streamCallback func(bus.StreamEvent)) (string, error) {
	msg := bus.InboundMessage{
		Channel:    "cli",
		SenderID:   "user",
		ChatID:     "direct",
		Content:    content,
		SessionKey: sessionKey,
	}

	return al.processMessageStreaming(ctx, msg, streamCallback)
}

// IsStreamingEnabled returns whether streaming is enabled
func (al *AgentLoop) IsStreamingEnabled() bool {
	return al.streaming
}

func (al *AgentLoop) processMessage(ctx context.Context, msg bus.InboundMessage) (string, error) {
	history := al.sessions.GetHistory(msg.SessionKey)
	summary := al.sessions.GetSummary(msg.SessionKey)

	messages := al.contextBuilder.BuildMessages(
		history,
		summary,
		msg.Content,
		nil,
	)

	iteration := 0
	var finalContent string

	for iteration < al.maxIterations {
		iteration++

		toolDefs := al.tools.GetDefinitions()
		providerToolDefs := make([]providers.ToolDefinition, 0, len(toolDefs))
		for _, td := range toolDefs {
			providerToolDefs = append(providerToolDefs, providers.ToolDefinition{
				Type: td["type"].(string),
				Function: providers.ToolFunctionDefinition{
					Name:        td["function"].(map[string]interface{})["name"].(string),
					Description: td["function"].(map[string]interface{})["description"].(string),
					Parameters:  td["function"].(map[string]interface{})["parameters"].(map[string]interface{}),
				},
			})
		}

		response, err := al.provider.Chat(ctx, messages, providerToolDefs, al.model, map[string]interface{}{
			"max_tokens":  8192,
			"temperature": 0.7,
		})

		if err != nil {
			return "", fmt.Errorf("LLM call failed: %w", err)
		}

		if len(response.ToolCalls) == 0 {
			finalContent = response.Content
			break
		}

		assistantMsg := providers.Message{
			Role:    "assistant",
			Content: response.Content,
		}

		for _, tc := range response.ToolCalls {
			argumentsJSON, _ := json.Marshal(tc.Arguments)
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, providers.ToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      tc.Name,
					Arguments: string(argumentsJSON),
				},
			})
		}
		messages = append(messages, assistantMsg)

		for _, tc := range response.ToolCalls {
			result, err := al.tools.Execute(ctx, tc.Name, tc.Arguments)
			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
			}

			toolResultMsg := providers.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolResultMsg)
		}
	}

	if finalContent == "" {
		finalContent = "I've completed processing but have no response to give."
	}

	al.sessions.AddMessage(msg.SessionKey, "user", msg.Content)
	al.sessions.AddMessage(msg.SessionKey, "assistant", finalContent)

	// Context compression logic
	newHistory := al.sessions.GetHistory(msg.SessionKey)

	// Token Awareness (Dynamic)
	// Trigger if history > 20 messages OR estimated tokens > 75% of context window
	tokenEstimate := al.estimateTokens(newHistory)
	threshold := al.contextWindow * 75 / 100

	if len(newHistory) > 20 || tokenEstimate > threshold {
		if _, loading := al.summarizing.LoadOrStore(msg.SessionKey, true); !loading {
			go func() {
				defer al.summarizing.Delete(msg.SessionKey)
				al.summarizeSession(msg.SessionKey)
			}()
		}
	}

	al.sessions.Save(al.sessions.GetOrCreate(msg.SessionKey))

	return finalContent, nil
}

// processMessageStreaming processes a message with streaming updates
func (al *AgentLoop) processMessageStreaming(ctx context.Context, msg bus.InboundMessage, streamCallback func(bus.StreamEvent)) (string, error) {
	// Check if callback is provided, if not fall back to non-streaming
	if streamCallback == nil {
		return al.processMessage(ctx, msg)
	}

	// Helper function to emit events
	emitEvent := func(eventType bus.StreamEventType, content string, metadata map[string]interface{}) {
		if streamCallback != nil {
			streamCallback(bus.StreamEvent{
				Type:       eventType,
				Channel:    msg.Channel,
				ChatID:     msg.ChatID,
				SessionKey: msg.SessionKey,
				Content:    content,
				Metadata:   metadata,
				Timestamp:  time.Now().Unix(),
			})
		}
	}

	history := al.sessions.GetHistory(msg.SessionKey)
	summary := al.sessions.GetSummary(msg.SessionKey)

	messages := al.contextBuilder.BuildMessages(
		history,
		summary,
		msg.Content,
		nil,
	)

	iteration := 0
	var finalContent string

	emitEvent(bus.StreamEventThinking, "Starting to process request...", nil)

	for iteration < al.maxIterations {
		// Check for interrupt
		select {
		case <-ctx.Done():
			emitEvent(bus.StreamEventError, "Processing interrupted", nil)
			return "", fmt.Errorf("processing interrupted")
		default:
		}

		// Check bus for interrupt signal
		if al.bus.CheckInterrupt(msg.SessionKey) {
			emitEvent(bus.StreamEventError, "Interrupted by user", nil)
			return "", fmt.Errorf("interrupted by user")
		}

		iteration++
		emitEvent(bus.StreamEventProgress, fmt.Sprintf("Thinking... (iteration %d/%d)", iteration, al.maxIterations), map[string]interface{}{
			"iteration":     iteration,
			"max_iteration": al.maxIterations,
		})

		toolDefs := al.tools.GetDefinitions()
		providerToolDefs := make([]providers.ToolDefinition, 0, len(toolDefs))
		for _, td := range toolDefs {
			providerToolDefs = append(providerToolDefs, providers.ToolDefinition{
				Type: td["type"].(string),
				Function: providers.ToolFunctionDefinition{
					Name:        td["function"].(map[string]interface{})["name"].(string),
					Description: td["function"].(map[string]interface{})["description"].(string),
					Parameters:  td["function"].(map[string]interface{})["parameters"].(map[string]interface{}),
				},
			})
		}

		// Use streaming API for real-time output
		response, err := al.provider.ChatStream(ctx, messages, providerToolDefs, al.model, map[string]interface{}{
			"max_tokens":  8192,
			"temperature": 0.7,
		}, func(chunk string) {
			// Stream each token as it arrives
			if streamCallback != nil {
				emitEvent(bus.StreamEventContent, chunk, map[string]interface{}{
					"partial": true,
				})
			}
		})

		if err != nil {
			emitEvent(bus.StreamEventError, fmt.Sprintf("LLM call failed: %v", err), nil)
			return "", fmt.Errorf("LLM call failed: %w", err)
		}

		if len(response.ToolCalls) == 0 {
			finalContent = response.Content
			emitEvent(bus.StreamEventComplete, "Content generation complete", nil)
			break
		}

		// Emit tool calls
		for _, tc := range response.ToolCalls {
			emitEvent(bus.StreamEventToolCall, fmt.Sprintf("Executing tool: %s", tc.Name), map[string]interface{}{
				"tool_name": tc.Name,
				"tool_id":   tc.ID,
			})
		}

		assistantMsg := providers.Message{
			Role:    "assistant",
			Content: response.Content,
		}

		for _, tc := range response.ToolCalls {
			argumentsJSON, _ := json.Marshal(tc.Arguments)
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, providers.ToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      tc.Name,
					Arguments: string(argumentsJSON),
				},
			})
		}
		messages = append(messages, assistantMsg)

		for _, tc := range response.ToolCalls {
			// Check for interrupt before tool execution
			select {
			case <-ctx.Done():
				emitEvent(bus.StreamEventError, "Processing interrupted", nil)
				return "", fmt.Errorf("processing interrupted")
			default:
			}

			if al.bus.CheckInterrupt(msg.SessionKey) {
				emitEvent(bus.StreamEventError, "Interrupted by user", nil)
				return "", fmt.Errorf("interrupted by user")
			}

			result, err := al.tools.Execute(ctx, tc.Name, tc.Arguments)
			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
			}

			// Truncate long results for display
			displayResult := result
			if len(displayResult) > 200 {
				displayResult = displayResult[:200] + "..."
			}

			emitEvent(bus.StreamEventToolResult, fmt.Sprintf("Tool %s completed", tc.Name), map[string]interface{}{
				"tool_name": tc.Name,
				"tool_id":   tc.ID,
				"result":    displayResult,
			})

			toolResultMsg := providers.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolResultMsg)
		}
	}

	if finalContent == "" {
		finalContent = "I've completed processing but have no response to give."
		emitEvent(bus.StreamEventContent, finalContent, nil)
	}

	emitEvent(bus.StreamEventComplete, "Processing complete", nil)

	al.sessions.AddMessage(msg.SessionKey, "user", msg.Content)
	al.sessions.AddMessage(msg.SessionKey, "assistant", finalContent)

	// Context compression logic
	newHistory := al.sessions.GetHistory(msg.SessionKey)

	tokenEstimate := al.estimateTokens(newHistory)
	threshold := al.contextWindow * 75 / 100

	if len(newHistory) > 20 || tokenEstimate > threshold {
		if _, loading := al.summarizing.LoadOrStore(msg.SessionKey, true); !loading {
			go func() {
				defer al.summarizing.Delete(msg.SessionKey)
				al.summarizeSession(msg.SessionKey)
			}()
		}
	}

	al.sessions.Save(al.sessions.GetOrCreate(msg.SessionKey))

	return finalContent, nil
}

func (al *AgentLoop) summarizeSession(sessionKey string) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	history := al.sessions.GetHistory(sessionKey)
	summary := al.sessions.GetSummary(sessionKey)

	// Keep last 4 messages for continuity
	if len(history) <= 4 {
		return
	}

	toSummarize := history[:len(history)-4]

	// Oversized Message Guard (Dynamic)
	// Skip messages larger than 50% of context window to prevent summarizer overflow.
	maxMessageTokens := al.contextWindow / 2
	validMessages := make([]providers.Message, 0)
	omitted := false

	for _, m := range toSummarize {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		// Estimate tokens for this message
		msgTokens := len(m.Content) / 4
		if msgTokens > maxMessageTokens {
			omitted = true
			continue
		}
		validMessages = append(validMessages, m)
	}

	if len(validMessages) == 0 {
		return
	}

	// Multi-Part Summarization
	// Split into two parts if history is significant
	var finalSummary string
	if len(validMessages) > 10 {
		mid := len(validMessages) / 2
		part1 := validMessages[:mid]
		part2 := validMessages[mid:]

		s1, _ := al.summarizeBatch(ctx, part1, "")
		s2, _ := al.summarizeBatch(ctx, part2, "")

		// Merge them
		mergePrompt := fmt.Sprintf("Merge these two conversation summaries into one cohesive summary:\n\n1: %s\n\n2: %s", s1, s2)
		resp, err := al.provider.Chat(ctx, []providers.Message{{Role: "user", Content: mergePrompt}}, nil, al.model, map[string]interface{}{
			"max_tokens":  1024,
			"temperature": 0.3,
		})
		if err == nil {
			finalSummary = resp.Content
		} else {
			finalSummary = s1 + " " + s2
		}
	} else {
		finalSummary, _ = al.summarizeBatch(ctx, validMessages, summary)
	}

	if omitted && finalSummary != "" {
		finalSummary += "\n[Note: Some oversized messages were omitted from this summary for efficiency.]"
	}

	if finalSummary != "" {
		al.sessions.SetSummary(sessionKey, finalSummary)
		al.sessions.TruncateHistory(sessionKey, 4)
		al.sessions.Save(al.sessions.GetOrCreate(sessionKey))
	}
}

func (al *AgentLoop) summarizeBatch(ctx context.Context, batch []providers.Message, existingSummary string) (string, error) {
	prompt := "Provide a concise summary of this conversation segment, preserving core context and key points.\n"
	if existingSummary != "" {
		prompt += "Existing context: " + existingSummary + "\n"
	}
	prompt += "\nCONVERSATION:\n"
	for _, m := range batch {
		prompt += fmt.Sprintf("%s: %s\n", m.Role, m.Content)
	}

	response, err := al.provider.Chat(ctx, []providers.Message{{Role: "user", Content: prompt}}, nil, al.model, map[string]interface{}{
		"max_tokens":  1024,
		"temperature": 0.3,
	})
	if err != nil {
		return "", err
	}
	return response.Content, nil
}

func (al *AgentLoop) estimateTokens(messages []providers.Message) int {
	total := 0
	for _, m := range messages {
		total += len(m.Content) / 4 // Simple heuristic: 4 chars per token
	}
	return total
}
