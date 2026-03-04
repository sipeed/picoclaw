// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
)

func TestInterruptSignal(t *testing.T) {
	signal := InterruptSignal{
		Type:     "user_message",
		Priority: 9,
		Data:     "test message",
		Source:   "telegram:123",
	}

	if signal.Type != "user_message" {
		t.Errorf("Expected type 'user_message', got '%s'", signal.Type)
	}
	if signal.Priority != 9 {
		t.Errorf("Expected priority 9, got %d", signal.Priority)
	}
}

func TestDefaultInterruptionConfig(t *testing.T) {
	config := DefaultInterruptionConfig()

	// Enabled is true by default (changed from false for Phase 1.5)
	if !config.Enabled {
		t.Error("Expected interruption to be enabled by default")
	}
	if config.MinPriority != 8 {
		t.Errorf("Expected min priority 8, got %d", config.MinPriority)
	}
	if config.CheckInterval != 1*time.Second {
		t.Errorf("Expected check interval 1s, got %v", config.CheckInterval)
	}
}

func TestNoOpInterruptHandler(t *testing.T) {
	handler := NewNoOpInterruptHandler()
	ctx := context.Background()

	signal, err := handler.CheckInterruption(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if signal != nil {
		t.Errorf("Expected nil signal, got %+v", signal)
	}
}

func TestBusInterruptHandler_Disabled(t *testing.T) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()
	config.Enabled = false

	handler := NewBusInterruptHandler(msgBus, config)
	handler.SetContext("telegram", "123")

	ctx := context.Background()
	signal, err := handler.CheckInterruption(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if signal != nil {
		t.Errorf("Expected nil signal when disabled, got %+v", signal)
	}
}

func TestBusInterruptHandler_ContextCancellation(t *testing.T) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()
	config.Enabled = true

	handler := NewBusInterruptHandler(msgBus, config)
	handler.SetContext("telegram", "123")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	signal, err := handler.CheckInterruption(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if signal == nil {
		t.Fatal("Expected cancellation signal, got nil")
	}
	if signal.Type != "cancellation" {
		t.Errorf("Expected type 'cancellation', got '%s'", signal.Type)
	}
	if signal.Priority != 10 {
		t.Errorf("Expected priority 10 for cancellation, got %d", signal.Priority)
	}
}

func TestBusInterruptHandler_RateLimiting(t *testing.T) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()
	config.Enabled = true
	config.CheckInterval = 1 * time.Second

	handler := NewBusInterruptHandler(msgBus, config)
	handler.SetContext("telegram", "123")

	ctx := context.Background()

	// First check
	_, err := handler.CheckInterruption(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Immediate second check should be skipped due to rate limiting
	_, err = handler.CheckInterruption(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Wait for check interval to pass
	time.Sleep(1100 * time.Millisecond)

	// Third check should proceed
	_, err = handler.CheckInterruption(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestBusInterruptHandler_SetContext(t *testing.T) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()

	handler := NewBusInterruptHandler(msgBus, config)

	handler.SetContext("telegram", "123")
	if handler.currentChan != "telegram" {
		t.Errorf("Expected channel 'telegram', got '%s'", handler.currentChan)
	}
	if handler.currentChatID != "123" {
		t.Errorf("Expected chat ID '123', got '%s'", handler.currentChatID)
	}

	handler.SetContext("discord", "456")
	if handler.currentChan != "discord" {
		t.Errorf("Expected channel 'discord', got '%s'", handler.currentChan)
	}
	if handler.currentChatID != "456" {
		t.Errorf("Expected chat ID '456', got '%s'", handler.currentChatID)
	}
}

func TestCalculatePriority_DifferentChannel(t *testing.T) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()

	handler := NewBusInterruptHandler(msgBus, config)
	handler.SetContext("telegram", "123")

	// Message from different channel should have higher priority
	msg := bus.InboundMessage{
		Channel: "discord",
		ChatID:  "456",
		Content: "test message",
	}

	priority := handler.calculatePriority(msg)
	if priority < 8 {
		t.Errorf("Expected priority >= 8 for different channel, got %d", priority)
	}
}

func TestCalculatePriority_UrgentKeywords(t *testing.T) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()

	handler := NewBusInterruptHandler(msgBus, config)
	handler.SetContext("telegram", "123")

	testCases := []struct {
		content  string
		keyword  string
		expected int // minimum expected priority
	}{
		{"urgent: need help", "urgent", 7},
		{"emergency situation", "emergency", 7},
		{"please stop", "stop", 7},
		{"cancel the task", "cancel", 7},
		{"help me", "help", 7},
	}

	for _, tc := range testCases {
		msg := bus.InboundMessage{
			Channel: "telegram",
			ChatID:  "123",
			Content: tc.content,
		}

		priority := handler.calculatePriority(msg)
		if priority < tc.expected {
			t.Errorf("Content '%s' with keyword '%s': expected priority >= %d, got %d",
				tc.content, tc.keyword, tc.expected, priority)
		}
	}
}

func TestCalculatePriority_Command(t *testing.T) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()

	handler := NewBusInterruptHandler(msgBus, config)
	handler.SetContext("telegram", "123")

	msg := bus.InboundMessage{
		Channel: "telegram",
		ChatID:  "123",
		Content: "/status",
	}

	priority := handler.calculatePriority(msg)
	if priority < 6 {
		t.Errorf("Expected priority >= 6 for command, got %d", priority)
	}
}

func TestCalculatePriority_Capped(t *testing.T) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()

	handler := NewBusInterruptHandler(msgBus, config)
	handler.SetContext("telegram", "123")

	// Message with multiple priority boosts
	msg := bus.InboundMessage{
		Channel: "discord", // +3
		ChatID:  "456",
		Content: "/urgent emergency stop", // +1 (command) + 2 (urgent keyword)
	}

	priority := handler.calculatePriority(msg)
	if priority > 10 {
		t.Errorf("Expected priority capped at 10, got %d", priority)
	}
}

func TestContains(t *testing.T) {
	testCases := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"urgent message", "urgent", true},
		{"this is urgent", "urgent", true},
		{"URGENT", "URGENT", true},
		{"emergency", "emer", true},
		{"test", "testing", false},
		{"", "test", false},
		{"test", "", true},
	}

	for _, tc := range testCases {
		result := contains(tc.s, tc.substr)
		if result != tc.expected {
			t.Errorf("contains(%q, %q): expected %v, got %v",
				tc.s, tc.substr, tc.expected, result)
		}
	}
}

func TestErrInterrupted(t *testing.T) {
	if ErrInterrupted == nil {
		t.Error("Expected ErrInterrupted to be defined")
	}
	if ErrInterrupted.Error() != "agent execution interrupted" {
		t.Errorf("Expected error message 'agent execution interrupted', got '%s'",
			ErrInterrupted.Error())
	}
}

// Benchmark tests
func BenchmarkCheckInterruption_NoOp(b *testing.B) {
	handler := NewNoOpInterruptHandler()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.CheckInterruption(ctx)
	}
}

func BenchmarkCheckInterruption_Disabled(b *testing.B) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()
	config.Enabled = false

	handler := NewBusInterruptHandler(msgBus, config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.CheckInterruption(ctx)
	}
}

func BenchmarkCheckInterruption_RateLimited(b *testing.B) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()
	config.Enabled = true
	config.CheckInterval = 1 * time.Hour // Ensure rate limiting kicks in

	handler := NewBusInterruptHandler(msgBus, config)
	handler.SetContext("telegram", "123")
	ctx := context.Background()

	// Prime the rate limiter
	handler.CheckInterruption(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.CheckInterruption(ctx)
	}
}

func BenchmarkCalculatePriority(b *testing.B) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()

	handler := NewBusInterruptHandler(msgBus, config)
	handler.SetContext("telegram", "123")

	msg := bus.InboundMessage{
		Channel: "discord",
		ChatID:  "456",
		Content: "urgent: please help with this emergency",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.calculatePriority(msg)
	}
}

// ============================================================================
// Phase 1.5 Tests: Interrupt Command Support
// ============================================================================

func TestPhase15_InterruptCommands(t *testing.T) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()
	config.Enabled = true
	config.AllowSameSession = true
	config.InterruptCommands = []string{InterruptCommandCancel, InterruptCommandStop, InterruptCommandAbort}

	handler := NewBusInterruptHandler(msgBus, config)
	handler.SetContext("telegram", "123")

	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{"Cancel command", "/cancel", 10},
		{"Stop command", "/stop", 10},
		{"Abort command", "/abort", 10},
		{"Cancel with uppercase", "/CANCEL", 10},
		{"Cancel with text after", "/cancel please stop this", 10},
		{"Stop with extra spaces", "  /stop  ", 10},
		{"Regular message", "hello world", 5},
		{"Command-like but not interrupt", "/help", 8}, // 5 + 2 (contains "help" keyword) + 1 (command) = 8
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := bus.InboundMessage{
				Channel: "telegram",
				ChatID:  "123",
				Content: tt.content,
			}
			priority := handler.calculatePriority(msg)
			if priority != tt.expected {
				t.Errorf("Expected priority %d, got %d for content '%s'", tt.expected, priority, tt.content)
			}
		})
	}
}

func TestPhase15_InterruptCommandsAlwaysWork(t *testing.T) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()
	config.Enabled = true
	config.AllowSameSession = false // Disabled for mind-change detection
	config.InterruptCommands = []string{InterruptCommandCancel}

	handler := NewBusInterruptHandler(msgBus, config)
	handler.SetContext("telegram", "123")

	msg := bus.InboundMessage{
		Channel: "telegram",
		ChatID:  "123",
		Content: "/cancel",
	}

	priority := handler.calculatePriority(msg)
	// Bug fix: Interrupt commands should ALWAYS work, regardless of AllowSameSession
	// AllowSameSession only controls mind-change detection, not interrupt commands
	if priority != 10 {
		t.Errorf("Interrupt command should always work (priority 10), got %d. AllowSameSession only controls mind-change detection.", priority)
	}
}

func TestPhase15_MindChangeDetection(t *testing.T) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()
	config.Enabled = true
	config.AllowSameSession = true
	config.MindChangeWindow = 5 * time.Second

	handler := NewBusInterruptHandler(msgBus, config)
	handler.SetContext("telegram", "123")

	// Simulate first message
	handler.lastMessageTime = time.Now()

	// Wait a moment (but within window)
	time.Sleep(100 * time.Millisecond)

	// Second message from same session
	msg := bus.InboundMessage{
		Channel: "telegram",
		ChatID:  "123",
		Content: "actually, do something else",
	}

	priority := handler.calculatePriority(msg)
	// Should get boosted priority (5 + 4 = 9)
	if priority != 9 {
		t.Errorf("Expected priority 9 for mind-change, got %d", priority)
	}
}

func TestPhase15_MindChangeOutsideWindow(t *testing.T) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()
	config.Enabled = true
	config.AllowSameSession = true
	config.MindChangeWindow = 100 * time.Millisecond

	handler := NewBusInterruptHandler(msgBus, config)
	handler.SetContext("telegram", "123")

	// Simulate first message
	handler.lastMessageTime = time.Now()

	// Wait longer than window
	time.Sleep(150 * time.Millisecond)

	// Second message from same session
	msg := bus.InboundMessage{
		Channel: "telegram",
		ChatID:  "123",
		Content: "another request",
	}

	priority := handler.calculatePriority(msg)
	// Should NOT get boosted priority (still 5)
	if priority != 5 {
		t.Errorf("Expected priority 5 (no boost), got %d", priority)
	}
}

func TestPhase15_MindChangeDisabled(t *testing.T) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()
	config.Enabled = true
	config.AllowSameSession = false // Disabled
	config.MindChangeWindow = 5 * time.Second

	handler := NewBusInterruptHandler(msgBus, config)
	handler.SetContext("telegram", "123")

	handler.lastMessageTime = time.Now()
	time.Sleep(100 * time.Millisecond)

	msg := bus.InboundMessage{
		Channel: "telegram",
		ChatID:  "123",
		Content: "new request",
	}

	priority := handler.calculatePriority(msg)
	// Should not get mind-change boost when AllowSameSession is false
	if priority == 9 {
		t.Error("Mind-change detection should not work when AllowSameSession is false")
	}
}

func TestPhase15_DefaultConfig(t *testing.T) {
	config := DefaultInterruptionConfig()

	// AllowSameSession is true by default (enables mind-change detection)
	// Interrupt commands work regardless of this setting (bug fix)
	if !config.AllowSameSession {
		t.Error("Expected AllowSameSession to be true by default")
	}

	if config.MindChangeWindow != 5*time.Second {
		t.Errorf("Expected MindChangeWindow to be 5s, got %v", config.MindChangeWindow)
	}

	expectedCommands := []string{InterruptCommandCancel, InterruptCommandStop, InterruptCommandAbort}
	if len(config.InterruptCommands) != len(expectedCommands) {
		t.Errorf("Expected %d interrupt commands, got %d", len(expectedCommands), len(config.InterruptCommands))
	}
}

func TestPhase15_InterruptCommandConstants(t *testing.T) {
	if InterruptCommandCancel != "/cancel" {
		t.Errorf("Expected '/cancel', got '%s'", InterruptCommandCancel)
	}
	if InterruptCommandStop != "/stop" {
		t.Errorf("Expected '/stop', got '%s'", InterruptCommandStop)
	}
	if InterruptCommandAbort != "/abort" {
		t.Errorf("Expected '/abort', got '%s'", InterruptCommandAbort)
	}
}

func TestPhase15_CombinedScenarios(t *testing.T) {
	msgBus := bus.NewMessageBus()
	config := DefaultInterruptionConfig()
	config.Enabled = true
	config.AllowSameSession = true
	config.MindChangeWindow = 5 * time.Second
	config.InterruptCommands = []string{InterruptCommandCancel, InterruptCommandStop}

	handler := NewBusInterruptHandler(msgBus, config)
	handler.SetContext("telegram", "123")

	tests := []struct {
		name        string
		setupFunc   func()
		msg         bus.InboundMessage
		expectedMin int
		expectedMax int
		description string
	}{
		{
			name:      "Interrupt command same session",
			setupFunc: func() {},
			msg: bus.InboundMessage{
				Channel: "telegram",
				ChatID:  "123",
				Content: "/cancel",
			},
			expectedMin: 10,
			expectedMax: 10,
			description: "Interrupt command should always get priority 10",
		},
		{
			name: "Mind change with urgent keyword",
			setupFunc: func() {
				handler.lastMessageTime = time.Now()
				time.Sleep(100 * time.Millisecond)
			},
			msg: bus.InboundMessage{
				Channel: "telegram",
				ChatID:  "123",
				Content: "urgent: change of plans",
			},
			expectedMin: 10, // 5 + 4 (mind change) + 2 (urgent) = 11, capped at 10
			expectedMax: 10,
			description: "Mind change + urgent should cap at 10",
		},
		{
			name: "Different channel ignores mind change",
			setupFunc: func() {
				handler.lastMessageTime = time.Now()
				time.Sleep(100 * time.Millisecond)
			},
			msg: bus.InboundMessage{
				Channel: "discord",
				ChatID:  "456",
				Content: "hello from different channel",
			},
			expectedMin: 8, // 5 + 3 (different channel)
			expectedMax: 8,
			description: "Different channel should not trigger mind-change detection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			priority := handler.calculatePriority(tt.msg)
			if priority < tt.expectedMin || priority > tt.expectedMax {
				t.Errorf("%s: Expected priority between %d-%d, got %d",
					tt.description, tt.expectedMin, tt.expectedMax, priority)
			}
		})
	}
}

// TestBugFix_InterruptCommandsWorkRegardlessOfAllowSameSession tests the fix for the bug
// where /stop, /cancel, /abort commands didn't trigger CheckInterruption when AllowSameSession was false.
// Bug reported: 2026-03-02
func TestBugFix_InterruptCommandsWorkRegardlessOfAllowSameSession(t *testing.T) {
	msgBus := bus.NewMessageBus()

	tests := []struct {
		name             string
		allowSameSession bool
		command          string
		expectedPriority int
		description      string
	}{
		{
			name:             "Stop command with AllowSameSession=false",
			allowSameSession: false,
			command:          "/stop",
			expectedPriority: 10,
			description:      "Interrupt commands should work even when AllowSameSession is false",
		},
		{
			name:             "Cancel command with AllowSameSession=false",
			allowSameSession: false,
			command:          "/cancel",
			expectedPriority: 10,
			description:      "Interrupt commands should work even when AllowSameSession is false",
		},
		{
			name:             "Abort command with AllowSameSession=false",
			allowSameSession: false,
			command:          "/abort",
			expectedPriority: 10,
			description:      "Interrupt commands should work even when AllowSameSession is false",
		},
		{
			name:             "Stop command with AllowSameSession=true",
			allowSameSession: true,
			command:          "/stop",
			expectedPriority: 10,
			description:      "Interrupt commands should work when AllowSameSession is true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultInterruptionConfig()
			config.Enabled = true
			config.AllowSameSession = tt.allowSameSession
			config.InterruptCommands = []string{InterruptCommandCancel, InterruptCommandStop, InterruptCommandAbort}

			handler := NewBusInterruptHandler(msgBus, config)
			handler.SetContext("telegram", "123")

			msg := bus.InboundMessage{
				Channel: "telegram",
				ChatID:  "123",
				Content: tt.command,
			}

			priority := handler.calculatePriority(msg)
			if priority != tt.expectedPriority {
				t.Errorf("%s: Expected priority %d, got %d (AllowSameSession=%v)",
					tt.description, tt.expectedPriority, priority, tt.allowSameSession)
			}
		})
	}
}
