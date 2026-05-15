package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/cron"
)

type stubJobExecutor struct {
	response        string
	err             error
	alreadySent     bool // simulate message tool having already sent in this round
	lastPrompt      string
	lastKey         string
	lastChan        string
	lastChatID      string
	scheduledUsed   bool
	publishedResp   string
	publishedChan   string
	publishedChatID string
	publishedKey    string
}

func (s *stubJobExecutor) ProcessDirectWithChannel(
	_ context.Context,
	content, sessionKey, channel, chatID string,
) (string, error) {
	s.lastPrompt = content
	s.lastKey = sessionKey
	s.lastChan = channel
	s.lastChatID = chatID
	return s.response, s.err
}

func (s *stubJobExecutor) ProcessScheduledWithChannel(
	_ context.Context,
	content, sessionKey, channel, chatID string,
) (string, error) {
	s.scheduledUsed = true
	s.lastPrompt = content
	s.lastKey = sessionKey
	s.lastChan = channel
	s.lastChatID = chatID
	return s.response, s.err
}

func (s *stubJobExecutor) PublishResponseIfNeeded(
	_ context.Context,
	channel, chatID, sessionKey, response string,
) {
	if s.alreadySent {
		return
	}
	s.publishedResp = response
	s.publishedChan = channel
	s.publishedChatID = chatID
	s.publishedKey = sessionKey
}

func newTestCronToolWithExecutorAndConfig(t *testing.T, executor JobExecutor, cfg *config.Config) *CronTool {
	t.Helper()
	storePath := filepath.Join(t.TempDir(), "cron.json")
	cronService := cron.NewCronService(storePath, nil)
	msgBus := bus.NewMessageBus()
	tool, err := NewCronTool(cronService, executor, msgBus, t.TempDir(), true, 0, cfg)
	if err != nil {
		t.Fatalf("NewCronTool() error: %v", err)
	}
	return tool
}

func newTestCronToolWithConfig(t *testing.T, cfg *config.Config) *CronTool {
	t.Helper()
	return newTestCronToolWithExecutorAndConfig(t, nil, cfg)
}

func newTestCronTool(t *testing.T) *CronTool {
	t.Helper()
	return newTestCronToolWithConfig(t, config.DefaultConfig())
}

// TestCronTool_CommandBlockedFromRemoteChannel verifies command scheduling is restricted to internal channels
func TestCronTool_CommandBlockedFromRemoteChannel(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	result := tool.Execute(ctx, map[string]any{
		"action":          "add",
		"message":         "check disk",
		"command":         "df -h",
		"command_confirm": true,
		"at_seconds":      float64(60),
	})

	if !result.IsError {
		t.Fatal("expected command scheduling to be blocked from remote channel")
	}
	if !strings.Contains(result.ForLLM, "restricted to internal channels") {
		t.Errorf("expected 'restricted to internal channels', got: %s", result.ForLLM)
	}
}

func TestCronTool_CommandDoesNotRequireConfirmByDefault(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	result := tool.Execute(ctx, map[string]any{
		"action":     "add",
		"message":    "check disk",
		"command":    "df -h",
		"at_seconds": float64(60),
	})

	if result.IsError {
		t.Fatalf("expected command scheduling without confirm to succeed by default, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Cron job added") {
		t.Errorf("expected 'Cron job added', got: %s", result.ForLLM)
	}
}

func TestCronTool_CommandRequiresConfirmWhenAllowCommandDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Cron.AllowCommand = false

	tool := newTestCronToolWithConfig(t, cfg)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	result := tool.Execute(ctx, map[string]any{
		"action":     "add",
		"message":    "check disk",
		"command":    "df -h",
		"at_seconds": float64(60),
	})

	if !result.IsError {
		t.Fatal("expected command scheduling to require confirm when allow_command is disabled")
	}
	if !strings.Contains(result.ForLLM, "command_confirm=true") {
		t.Errorf("expected command_confirm requirement message, got: %s", result.ForLLM)
	}
}

func TestCronTool_CommandAllowedWithConfirmWhenAllowCommandDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Cron.AllowCommand = false

	tool := newTestCronToolWithConfig(t, cfg)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	result := tool.Execute(ctx, map[string]any{
		"action":          "add",
		"message":         "check disk",
		"command":         "df -h",
		"command_confirm": true,
		"at_seconds":      float64(60),
	})

	if result.IsError {
		t.Fatalf(
			"expected command scheduling with confirm to succeed when allow_command is disabled, got: %s",
			result.ForLLM,
		)
	}
	if !strings.Contains(result.ForLLM, "Cron job added") {
		t.Errorf("expected 'Cron job added', got: %s", result.ForLLM)
	}
}

func TestCronTool_CommandBlockedWhenExecDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Exec.Enabled = false

	tool := newTestCronToolWithConfig(t, cfg)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	result := tool.Execute(ctx, map[string]any{
		"action":          "add",
		"message":         "check disk",
		"command":         "df -h",
		"command_confirm": true,
		"at_seconds":      float64(60),
	})

	if !result.IsError {
		t.Fatal("expected command scheduling to be blocked when exec is disabled")
	}
	if !strings.Contains(result.ForLLM, "command execution is disabled") {
		t.Errorf("expected exec disabled message, got: %s", result.ForLLM)
	}
}

// TestCronTool_CommandAllowedFromInternalChannel verifies command scheduling works from internal channels
func TestCronTool_CommandAllowedFromInternalChannel(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	result := tool.Execute(ctx, map[string]any{
		"action":          "add",
		"message":         "check disk",
		"command":         "df -h",
		"command_confirm": true,
		"at_seconds":      float64(60),
	})

	if result.IsError {
		t.Fatalf("expected command scheduling to succeed from internal channel, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Cron job added") {
		t.Errorf("expected 'Cron job added', got: %s", result.ForLLM)
	}
}

// TestCronTool_AddJobRequiresSessionContext verifies fail-closed when channel/chatID missing
func TestCronTool_AddJobRequiresSessionContext(t *testing.T) {
	tool := newTestCronTool(t)
	result := tool.Execute(context.Background(), map[string]any{
		"action":     "add",
		"message":    "reminder",
		"at_seconds": float64(60),
	})

	if !result.IsError {
		t.Fatal("expected error when session context is missing")
	}
	if !strings.Contains(result.ForLLM, "no session context") {
		t.Errorf("expected 'no session context' message, got: %s", result.ForLLM)
	}
}

// TestCronTool_NonCommandJobAllowedFromRemoteChannel verifies regular reminders work from any channel
func TestCronTool_NonCommandJobAllowedFromRemoteChannel(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	result := tool.Execute(ctx, map[string]any{
		"action":     "add",
		"message":    "time to stretch",
		"at_seconds": float64(600),
	})

	if result.IsError {
		t.Fatalf("expected non-command reminder to succeed from remote channel, got: %s", result.ForLLM)
	}
}

func TestCronTool_ExecuteJobPublishesErrorWhenExecDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Exec.Enabled = false

	tool := newTestCronToolWithConfig(t, cfg)
	job := &cron.CronJob{}
	job.Payload.Channel = "cli"
	job.Payload.To = "direct"
	job.Payload.Command = "df -h"

	if got := tool.ExecuteJob(context.Background(), job); got != "ok" {
		t.Fatalf("ExecuteJob() = %q, want ok", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var msg bus.OutboundMessage
	select {
	case msg = <-tool.msgBus.OutboundChan():
		// got message
	case <-ctx.Done():
		t.Fatal("timeout waiting for outbound message")
	}
	if !strings.Contains(msg.Content, "command execution is disabled") {
		t.Fatalf("expected exec disabled message, got: %s", msg.Content)
	}
}

func TestCronTool_ExecuteJobPublishesAgentResponse(t *testing.T) {
	executor := &stubJobExecutor{response: "generated reply"}
	tool := newTestCronToolWithExecutorAndConfig(t, executor, config.DefaultConfig())

	job := &cron.CronJob{ID: "job-1"}
	job.Payload.Channel = "telegram"
	job.Payload.To = "chat-1"
	job.Payload.Message = "send me a poem"

	if got := tool.ExecuteJob(context.Background(), job); got != "ok" {
		t.Fatalf("ExecuteJob() = %q, want ok", got)
	}

	if !strings.HasPrefix(executor.lastKey, "agent:cron-job-1-") {
		t.Fatalf("sessionKey = %q, want agent:cron-job-1-{uuid}", executor.lastKey)
	}
	if executor.lastChan != "telegram" || executor.lastChatID != "chat-1" {
		t.Fatalf("executor target = %s/%s, want telegram/chat-1", executor.lastChan, executor.lastChatID)
	}
	if executor.lastPrompt != "send me a poem" {
		t.Fatalf("prompt = %q, want original message", executor.lastPrompt)
	}
	if !executor.scheduledUsed {
		t.Fatal("expected cron agent job to use scheduled executor path")
	}
	if executor.publishedResp != "generated reply" {
		t.Fatalf("published response = %q, want generated reply", executor.publishedResp)
	}
	if executor.publishedKey != executor.lastKey {
		t.Fatalf("published sessionKey = %q, want %q", executor.publishedKey, executor.lastKey)
	}
	if executor.publishedChan != "telegram" || executor.publishedChatID != "chat-1" {
		t.Fatalf("published target = %s/%s, want telegram/chat-1", executor.publishedChan, executor.publishedChatID)
	}
}

func TestCronTool_ExecuteJobSkipsEmptyAgentResponse(t *testing.T) {
	executor := &stubJobExecutor{}
	tool := newTestCronToolWithExecutorAndConfig(t, executor, config.DefaultConfig())

	job := &cron.CronJob{ID: "job-empty"}
	job.Payload.Channel = "telegram"
	job.Payload.To = "chat-1"
	job.Payload.Message = "say nothing"

	if got := tool.ExecuteJob(context.Background(), job); got != "ok" {
		t.Fatalf("ExecuteJob() = %q, want ok", got)
	}

	if executor.publishedResp != "" {
		t.Fatalf("unexpected published response: %q", executor.publishedResp)
	}
}

func TestCronTool_ExecuteJobSkipsWhenMessageToolAlreadySent(t *testing.T) {
	executor := &stubJobExecutor{response: "Sent.", alreadySent: true}
	tool := newTestCronToolWithExecutorAndConfig(t, executor, config.DefaultConfig())

	job := &cron.CronJob{ID: "job-msg-sent"}
	job.Payload.Channel = "telegram"
	job.Payload.To = "chat-1"
	job.Payload.Message = "send weather"

	if got := tool.ExecuteJob(context.Background(), job); got != "ok" {
		t.Fatalf("ExecuteJob() = %q, want ok", got)
	}

	if executor.publishedResp != "" {
		t.Fatalf("expected no published response when message tool already sent, got: %q", executor.publishedResp)
	}
}

func TestCronTool_ExecuteJobReturnsErrorWithoutPublish(t *testing.T) {
	executor := &stubJobExecutor{
		response: "this response must not be published",
		err:      fmt.Errorf("agent failure"),
	}
	tool := newTestCronToolWithExecutorAndConfig(t, executor, config.DefaultConfig())

	job := &cron.CronJob{ID: "job-err"}
	job.Payload.Channel = "telegram"
	job.Payload.To = "chat-1"
	job.Payload.Message = "do something"

	got := tool.ExecuteJob(context.Background(), job)
	if !strings.Contains(got, "agent failure") {
		t.Fatalf("ExecuteJob() = %q, want error message", got)
	}

	if executor.publishedResp != "" {
		t.Fatalf("unexpected publish on error path: %q", executor.publishedResp)
	}
}
