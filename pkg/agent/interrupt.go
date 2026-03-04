// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// DEPRECATED: This file contains the legacy interrupt handler for Phase 1.5 command-based interruption.
// The new steering architecture (nanobot-inspired) uses message injection with LLM-driven decisions.
// This code is kept for backward compatibility but will be removed in a future version.
// See: pkg/agent/interruption_checker.go for the new implementation.

package agent

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// Interrupt command constants
const (
	InterruptCommandCancel = "/cancel"
	InterruptCommandStop   = "/stop"
	InterruptCommandAbort  = "/abort"
)

// ErrInterrupted is returned when agent execution is interrupted by a high-priority signal
var ErrInterrupted = errors.New("agent execution interrupted")

// InterruptSignal represents an interruption event
type InterruptSignal struct {
	Type     string // "user_message", "system_alert", "timeout", "cancellation"
	Priority int    // Priority level 1-10, higher means more urgent
	Data     any    // Signal-specific data
	Source   string // Source identifier (e.g., channel:chatID)
}

// InterruptHandler checks for interruption signals during agent execution
type InterruptHandler interface {
	// CheckInterruption checks if there is a pending interruption signal
	// Returns nil if no interruption, or an InterruptSignal if one exists
	CheckInterruption(ctx context.Context) (*InterruptSignal, error)
}

// InterruptionConfig configures interruption behavior
type InterruptionConfig struct {
	Enabled           bool          // Whether interruption checking is enabled
	CheckInterval     time.Duration // Minimum interval between checks
	MinPriority       int           // Minimum priority to trigger interruption (1-10)
	MaxQueueSize      int           // Maximum pending messages to check
	SkipInternalChan  bool          // Skip checking for internal channels
	AllowSameSession  bool          // Allow interruption within the same session
	MindChangeWindow  time.Duration // Time window to detect "mind change" (e.g., 5 seconds)
	InterruptCommands []string      // Commands that trigger immediate interruption
}

// DefaultInterruptionConfig returns sensible defaults
func DefaultInterruptionConfig() InterruptionConfig {
	return InterruptionConfig{
		Enabled:           true, // Disabled by default for backward compatibility
		CheckInterval:     1 * time.Second,
		MinPriority:       8, // Only interrupt for high-priority messages
		MaxQueueSize:      10,
		SkipInternalChan:  true,
		AllowSameSession:  true, // Disabled by default
		MindChangeWindow:  5 * time.Second,
		InterruptCommands: []string{InterruptCommandCancel, InterruptCommandStop, InterruptCommandAbort},
	}
}

// BusInterruptHandler checks for new inbound messages on the message bus
type BusInterruptHandler struct {
	bus             *bus.MessageBus
	config          InterruptionConfig
	currentChan     string
	currentChatID   string
	lastCheck       time.Time
	lastMessageTime time.Time // Track last message time for mind-change detection
}

// NewBusInterruptHandler creates a new interrupt handler that monitors the message bus
func NewBusInterruptHandler(msgBus *bus.MessageBus, config InterruptionConfig) *BusInterruptHandler {
	return &BusInterruptHandler{
		bus:    msgBus,
		config: config,
	}
}

// SetContext updates the current execution context (channel and chat ID)
func (h *BusInterruptHandler) SetContext(channel, chatID string) {
	h.currentChan = channel
	h.currentChatID = chatID
}

// CheckInterruption implements InterruptHandler interface
func (h *BusInterruptHandler) CheckInterruption(ctx context.Context) (*InterruptSignal, error) {
	// Check if interruption is enabled
	if !h.config.Enabled {
		return nil, nil
	}

	// Rate limiting: avoid checking too frequently
	if time.Since(h.lastCheck) < h.config.CheckInterval {
		return nil, nil
	}
	h.lastCheck = time.Now()

	// Check context cancellation first
	select {
	case <-ctx.Done():
		return &InterruptSignal{
			Type:     "cancellation",
			Priority: 10, // Highest priority
			Data:     ctx.Err(),
			Source:   "context",
		}, nil
	default:
		// Continue checking
	}

	// Non-blocking check for new inbound messages
	// This is a simplified implementation - in production, you might want to
	// peek at the bus queue without consuming messages
	select {
	case msg := <-h.peekInbound():
		// Calculate priority based on message characteristics
		priority := h.calculatePriority(msg)

		// PHASE 1.5: Update lastMessageTime for mind-change detection
		if msg.Channel == h.currentChan && msg.ChatID == h.currentChatID {
			h.lastMessageTime = time.Now()
		}

		if priority >= h.config.MinPriority {
			logger.InfoCF("agent", "High-priority interruption detected",
				map[string]any{
					"type":     "user_message",
					"priority": priority,
					"channel":  msg.Channel,
					"chat_id":  msg.ChatID,
				})

			return &InterruptSignal{
				Type:     "user_message",
				Priority: priority,
				Data:     msg,
				Source:   msg.Channel + ":" + msg.ChatID,
			}, nil
		}

		// Put the message back if priority is not high enough
		// (In real implementation, we wouldn't consume it in the first place)
		h.bus.PublishInbound(ctx, msg)

	default:
		// No new messages
	}

	return nil, nil
}

// peekInbound attempts to peek at the inbound bus without blocking
// This is a simplified implementation for demonstration
func (h *BusInterruptHandler) peekInbound() <-chan bus.InboundMessage {
	// In a real implementation, you might want to add a Peek method to the MessageBus
	// For now, we'll use a goroutine with timeout
	ch := make(chan bus.InboundMessage, 1)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		msg, ok := h.bus.ConsumeInbound(ctx)
		if ok {
			ch <- msg
		}
		close(ch)
	}()

	return ch
}

// calculatePriority determines the priority of a message
func (h *BusInterruptHandler) calculatePriority(msg bus.InboundMessage) int {
	content := strings.ToLower(strings.TrimSpace(msg.Content))
	isSameSession := (msg.Channel == h.currentChan && msg.ChatID == h.currentChatID)

	// PHASE 1.5: Check for interrupt commands first (highest priority)
	// Interrupt commands should ALWAYS have max priority, regardless of session
	for _, cmd := range h.config.InterruptCommands {
		if strings.HasPrefix(content, strings.ToLower(cmd)) {
			logger.DebugCF("agent", "Interrupt command detected in priority calculation",
				map[string]any{
					"command":        cmd,
					"content":        msg.Content,
					"same_session":   isSameSession,
					"current_chan":   h.currentChan,
					"current_chatid": h.currentChatID,
					"msg_chan":       msg.Channel,
					"msg_chatid":     msg.ChatID,
				})
			return 10 // Maximum priority for explicit interrupt commands
		}
	}

	// Default priority
	priority := 5

	// PHASE 1.5: Mind-change detection (boost same-session priority)
	if h.config.AllowSameSession && isSameSession {
		if !h.lastMessageTime.IsZero() && time.Since(h.lastMessageTime) < h.config.MindChangeWindow {
			// User sent new message quickly after previous one - likely changing their mind
			priority += 4 // Boost to 9, just above threshold of 8
		}
	}

	// Higher priority for different channels (same channel/chat = lower priority)
	if !isSameSession {
		priority += 3
	}

	// Keywords that indicate urgency
	urgentKeywords := []string{"urgent", "emergency", "stop", "cancel", "help"}
	for _, keyword := range urgentKeywords {
		if contains(msg.Content, keyword) {
			priority += 2
			break
		}
	}

	// Commands typically have higher priority
	if len(msg.Content) > 0 && msg.Content[0] == '/' {
		priority += 1
	}

	// Cap at 10
	if priority > 10 {
		priority = 10
	}

	return priority
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr))
}

// NoOpInterruptHandler is a no-op implementation that never interrupts
type NoOpInterruptHandler struct{}

// CheckInterruption always returns nil (no interruption)
func (h *NoOpInterruptHandler) CheckInterruption(ctx context.Context) (*InterruptSignal, error) {
	return nil, nil
}

// NewNoOpInterruptHandler creates a new no-op interrupt handler
func NewNoOpInterruptHandler() *NoOpInterruptHandler {
	return &NoOpInterruptHandler{}
}
