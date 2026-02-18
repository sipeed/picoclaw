package security

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestPolicyMode_IsOff(t *testing.T) {
	tests := []struct {
		mode PolicyMode
		want bool
	}{
		{ModeOff, true},
		{"off", true},
		{"", true},
		{ModeBlock, false},
		{ModeApprove, false},
		{"unknown", false},
	}
	for _, tt := range tests {
		if got := tt.mode.IsOff(); got != tt.want {
			t.Errorf("PolicyMode(%q).IsOff() = %v, want %v", tt.mode, got, tt.want)
		}
	}
}

func TestPolicyEngine_GetMode(t *testing.T) {
	cfg := &config.SecurityConfig{
		ExecGuard:       "block",
		SSRFProtection:  "approve",
		PathValidation:  "off",
		SkillValidation: "",
	}
	pe := NewPolicyEngine(cfg, nil)

	tests := []struct {
		category string
		want     PolicyMode
	}{
		{"exec_guard", ModeBlock},
		{"ssrf", ModeApprove},
		{"path_validation", ModeOff},
		{"skill_validation", ModeOff},
		{"unknown_category", ModeOff},
	}
	for _, tt := range tests {
		if got := pe.GetMode(tt.category); got != tt.want {
			t.Errorf("GetMode(%q) = %v, want %v", tt.category, got, tt.want)
		}
	}
}

func TestPolicyEngine_Evaluate_Off(t *testing.T) {
	pe := NewPolicyEngine(&config.SecurityConfig{}, nil)
	err := pe.Evaluate(context.Background(), ModeOff, Violation{Reason: "test"}, "telegram", "chat1")
	if err != nil {
		t.Errorf("ModeOff should allow, got: %v", err)
	}
}

func TestPolicyEngine_Evaluate_Block(t *testing.T) {
	pe := NewPolicyEngine(&config.SecurityConfig{}, nil)
	err := pe.Evaluate(context.Background(), ModeBlock, Violation{
		Category: "exec_guard",
		Reason:   "dangerous pattern",
	}, "telegram", "chat1")
	if err == nil {
		t.Fatal("ModeBlock should reject")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("error should contain 'blocked', got: %v", err)
	}
}

func TestPolicyEngine_Evaluate_Approve_CLIFallback(t *testing.T) {
	pe := NewPolicyEngine(&config.SecurityConfig{ApprovalTimeout: 5}, bus.NewMessageBus())
	err := pe.Evaluate(context.Background(), ModeApprove, Violation{
		Category: "exec_guard",
		Reason:   "test",
	}, "cli", "direct")
	if err == nil {
		t.Fatal("CLI should fall back to block")
	}
	if !strings.Contains(err.Error(), "unavailable in CLI") {
		t.Errorf("error should mention CLI, got: %v", err)
	}
}

func TestPolicyEngine_Evaluate_Approve_Approved(t *testing.T) {
	msgBus := bus.NewMessageBus()
	pe := NewPolicyEngine(&config.SecurityConfig{ApprovalTimeout: 5}, msgBus)

	// Start approval in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- pe.Evaluate(context.Background(), ModeApprove, Violation{
			Category: "exec_guard",
			Tool:     "exec",
			Action:   "rm -rf /tmp/test",
			Reason:   "dangerous pattern",
		}, "telegram", "chat123")
	}()

	// Consume the outbound approval notification
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	outMsg, ok := msgBus.SubscribeOutbound(ctx)
	if !ok {
		t.Fatal("expected outbound approval message")
	}
	if !strings.Contains(outMsg.Content, "Approval Required") {
		t.Errorf("approval message should contain 'Approval Required', got: %s", outMsg.Content)
	}

	// Send approval reply
	time.Sleep(50 * time.Millisecond) // small delay to ensure interceptor is registered
	msgBus.PublishInbound(bus.InboundMessage{
		Channel: "telegram",
		ChatID:  "chat123",
		Content: "approve",
	})

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("expected approval to succeed, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("approval timed out")
	}
}

func TestPolicyEngine_Evaluate_Approve_Denied(t *testing.T) {
	msgBus := bus.NewMessageBus()
	pe := NewPolicyEngine(&config.SecurityConfig{ApprovalTimeout: 5}, msgBus)

	errCh := make(chan error, 1)
	go func() {
		errCh <- pe.Evaluate(context.Background(), ModeApprove, Violation{
			Category: "ssrf",
			Reason:   "private IP",
		}, "feishu", "chat456")
	}()

	// Drain outbound message
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	msgBus.SubscribeOutbound(ctx)

	time.Sleep(50 * time.Millisecond)
	msgBus.PublishInbound(bus.InboundMessage{
		Channel: "feishu",
		ChatID:  "chat456",
		Content: "deny",
	})

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected denial error")
		}
		if !strings.Contains(err.Error(), "denied") {
			t.Errorf("error should contain 'denied', got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for denial")
	}
}

func TestPolicyEngine_Evaluate_Approve_Timeout(t *testing.T) {
	msgBus := bus.NewMessageBus()
	pe := NewPolicyEngine(&config.SecurityConfig{ApprovalTimeout: 1}, msgBus)

	errCh := make(chan error, 1)
	go func() {
		errCh <- pe.Evaluate(context.Background(), ModeApprove, Violation{
			Category: "exec_guard",
			Reason:   "test",
		}, "telegram", "chat789")
	}()

	// Drain outbound
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	msgBus.SubscribeOutbound(ctx)

	// Don't reply - let it timeout
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected timeout error")
		}
		if !strings.Contains(err.Error(), "timed out") {
			t.Errorf("error should contain 'timed out', got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("test itself timed out")
	}
}

func TestPolicyEngine_Evaluate_Approve_NonMatchingMessages(t *testing.T) {
	msgBus := bus.NewMessageBus()
	pe := NewPolicyEngine(&config.SecurityConfig{ApprovalTimeout: 2}, msgBus)

	errCh := make(chan error, 1)
	go func() {
		errCh <- pe.Evaluate(context.Background(), ModeApprove, Violation{
			Category: "exec_guard",
			Reason:   "test",
		}, "telegram", "chat100")
	}()

	// Drain outbound
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	msgBus.SubscribeOutbound(ctx)

	time.Sleep(50 * time.Millisecond)

	// Send non-matching messages (different chat, random text)
	msgBus.PublishInbound(bus.InboundMessage{Channel: "telegram", ChatID: "other-chat", Content: "approve"})
	msgBus.PublishInbound(bus.InboundMessage{Channel: "telegram", ChatID: "chat100", Content: "hello"})

	// Now send actual approval
	time.Sleep(50 * time.Millisecond)
	msgBus.PublishInbound(bus.InboundMessage{Channel: "telegram", ChatID: "chat100", Content: "approve"})

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("expected approval after non-matching messages, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out")
	}
}
