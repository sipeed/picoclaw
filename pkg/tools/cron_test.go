package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/cron"
	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
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

func parseCronJobResult(t *testing.T, result *ToolResult) cron.CronJob {
	t.Helper()
	text := result.ForLLM
	if idx := strings.Index(text, "{"); idx >= 0 {
		text = text[idx:]
	}
	var job cron.CronJob
	if err := json.Unmarshal([]byte(text), &job); err != nil {
		t.Fatalf("failed to parse cron job JSON %q: %v", result.ForLLM, err)
	}
	return job
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

func TestCronTool_GetReturnsFullJobPayload(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	everyMS := int64(60_000)
	message := strings.Repeat("daily briefing details ", 8)
	job, err := tool.cronService.AddJob(
		"daily",
		cron.CronSchedule{Kind: "every", EveryMS: &everyMS},
		message,
		"telegram",
		"chat-1",
	)
	if err != nil {
		t.Fatalf("AddJob() error: %v", err)
	}

	result := tool.Execute(ctx, map[string]any{
		"action": "get",
		"job_id": job.ID,
	})

	if result.IsError {
		t.Fatalf("get failed: %s", result.ForLLM)
	}
	got := parseCronJobResult(t, result)
	if got.ID != job.ID || got.Payload.Message != message || got.Payload.Channel != "telegram" ||
		got.Payload.To != "chat-1" {
		t.Fatalf("get returned wrong payload: %+v", got)
	}
	if got.Schedule.Kind != "every" || got.Schedule.EveryMS == nil || *got.Schedule.EveryMS != everyMS {
		t.Fatalf("get returned wrong schedule: %+v", got.Schedule)
	}
	if got.State.NextRunAtMS == nil {
		t.Fatal("get should include next run state")
	}
}

func TestCronTool_UpdateSchedulePreservesPayload(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	original, err := tool.cronService.AddJob(
		"AI daily",
		cron.CronSchedule{Kind: "cron", Expr: "0 8 * * *"},
		"fetch RSS, include source links",
		"weixin",
		"chat-1",
	)
	if err != nil {
		t.Fatalf("AddJob() error: %v", err)
	}

	result := tool.Execute(ctx, map[string]any{
		"action":    "update",
		"job_id":    original.ID,
		"cron_expr": "30 10 * * *",
	})

	if result.IsError {
		t.Fatalf("update failed: %s", result.ForLLM)
	}
	updated, ok := tool.cronService.GetJob(original.ID)
	if !ok {
		t.Fatal("updated job not found")
	}
	if updated.ID != original.ID || updated.CreatedAtMS != original.CreatedAtMS {
		t.Fatalf("identity changed after update: before=%+v after=%+v", original, updated)
	}
	if updated.Payload.Message != original.Payload.Message || updated.Payload.Channel != original.Payload.Channel ||
		updated.Payload.To != original.Payload.To {
		t.Fatalf("payload was not preserved: %+v", updated.Payload)
	}
	if updated.Schedule.Kind != "cron" || updated.Schedule.Expr != "30 10 * * *" {
		t.Fatalf("schedule not updated: %+v", updated.Schedule)
	}
	if updated.DeleteAfterRun {
		t.Fatal("cron schedule should not delete after run")
	}
}

func TestCronTool_UpdateMessagePreservesScheduleAndNextRun(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	everyMS := int64(120_000)
	original, err := tool.cronService.AddJob(
		"reminder",
		cron.CronSchedule{Kind: "every", EveryMS: &everyMS},
		"old message",
		"telegram",
		"chat-1",
	)
	if err != nil {
		t.Fatalf("AddJob() error: %v", err)
	}
	if original.State.NextRunAtMS == nil {
		t.Fatal("expected original next run")
	}
	nextRunBefore := *original.State.NextRunAtMS

	result := tool.Execute(ctx, map[string]any{
		"action":  "update",
		"job_id":  original.ID,
		"message": "new message",
	})

	if result.IsError {
		t.Fatalf("update failed: %s", result.ForLLM)
	}
	updated, _ := tool.cronService.GetJob(original.ID)
	if updated.Payload.Message != "new message" {
		t.Fatalf("message not updated: %+v", updated.Payload)
	}
	if updated.Name != "reminder" {
		t.Fatalf("name should be preserved, got %q", updated.Name)
	}
	if updated.Schedule.Kind != "every" || updated.Schedule.EveryMS == nil || *updated.Schedule.EveryMS != everyMS {
		t.Fatalf("schedule should be preserved: %+v", updated.Schedule)
	}
	if updated.State.NextRunAtMS == nil || *updated.State.NextRunAtMS != nextRunBefore {
		t.Fatalf("next run should be preserved: before=%d after=%v", nextRunBefore, updated.State.NextRunAtMS)
	}
}

func TestCronTool_UpdateValidationErrors(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	job, err := tool.cronService.AddJob(
		"job",
		cron.CronSchedule{Kind: "cron", Expr: "0 8 * * *"},
		"message",
		"cli",
		"direct",
	)
	if err != nil {
		t.Fatalf("AddJob() error: %v", err)
	}

	tests := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "invalid job id",
			args: map[string]any{"action": "update", "job_id": "missing", "message": "new"},
			want: "not found",
		},
		{
			name: "missing patch",
			args: map[string]any{"action": "update", "job_id": job.ID},
			want: "at least one update field",
		},
		{
			name: "multiple schedule fields",
			args: map[string]any{
				"action":        "update",
				"job_id":        job.ID,
				"every_seconds": float64(60),
				"cron_expr":     "0 9 * * *",
			},
			want: "only one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.Execute(ctx, tt.args)
			if !result.IsError {
				t.Fatalf("expected error, got: %s", result.ForLLM)
			}
			if !strings.Contains(result.ForLLM, tt.want) {
				t.Fatalf("error = %q, want substring %q", result.ForLLM, tt.want)
			}
		})
	}
}

func TestCronTool_ListFiltersJobsForRemoteChannel(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "telegram", "chat-1")
	everyMS := int64(60_000)

	ownJob, err := tool.cronService.AddJob(
		"own",
		cron.CronSchedule{Kind: "every", EveryMS: &everyMS},
		"visible",
		"telegram",
		"chat-1",
	)
	if err != nil {
		t.Fatalf("AddJob() error: %v", err)
	}
	otherChatJob, err := tool.cronService.AddJob(
		"other-chat",
		cron.CronSchedule{Kind: "every", EveryMS: &everyMS},
		"hidden",
		"telegram",
		"chat-2",
	)
	if err != nil {
		t.Fatalf("AddJob() error: %v", err)
	}
	otherChannelJob, err := tool.cronService.AddJob(
		"other-channel",
		cron.CronSchedule{Kind: "every", EveryMS: &everyMS},
		"hidden",
		"feishu",
		"chat-1",
	)
	if err != nil {
		t.Fatalf("AddJob() error: %v", err)
	}
	commandJob, err := tool.cronService.AddJob(
		"command",
		cron.CronSchedule{Kind: "every", EveryMS: &everyMS},
		"hidden command",
		"telegram",
		"chat-1",
	)
	if err != nil {
		t.Fatalf("AddJob() error: %v", err)
	}
	commandJob.Payload.Command = "df -h"
	if err := tool.cronService.UpdateJob(commandJob); err != nil {
		t.Fatalf("UpdateJob() error: %v", err)
	}

	result := tool.Execute(ctx, map[string]any{"action": "list"})

	if result.IsError {
		t.Fatalf("list failed: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, ownJob.ID) {
		t.Fatalf("list should include own job %s, got: %s", ownJob.ID, result.ForLLM)
	}
	for _, hiddenID := range []string{otherChatJob.ID, otherChannelJob.ID, commandJob.ID} {
		if strings.Contains(result.ForLLM, hiddenID) {
			t.Fatalf("list should not include hidden job %s, got: %s", hiddenID, result.ForLLM)
		}
	}
}

func TestCronTool_RemoteCannotAccessOtherChatJob(t *testing.T) {
	tool := newTestCronTool(t)
	job, err := tool.cronService.AddJob(
		"private",
		cron.CronSchedule{Kind: "cron", Expr: "0 8 * * *"},
		"secret",
		"telegram",
		"chat-1",
	)
	if err != nil {
		t.Fatalf("AddJob() error: %v", err)
	}
	ctx := WithToolContext(context.Background(), "telegram", "chat-2")

	getResult := tool.Execute(ctx, map[string]any{"action": "get", "job_id": job.ID})
	if !getResult.IsError || !strings.Contains(getResult.ForLLM, "not accessible") {
		t.Fatalf("expected inaccessible get, got: %+v", getResult)
	}

	updateResult := tool.Execute(ctx, map[string]any{"action": "update", "job_id": job.ID, "message": "changed"})
	if !updateResult.IsError || !strings.Contains(updateResult.ForLLM, "not accessible") {
		t.Fatalf("expected inaccessible update, got: %+v", updateResult)
	}
	unchanged, ok := tool.cronService.GetJob(job.ID)
	if !ok {
		t.Fatal("job should still exist")
	}
	if unchanged.Payload.Message != "secret" {
		t.Fatalf("unauthorized update mutated job: %+v", unchanged.Payload)
	}
}

func TestCronTool_RemoteCannotAccessCommandJob(t *testing.T) {
	tool := newTestCronTool(t)
	job, err := tool.cronService.AddJob(
		"command",
		cron.CronSchedule{Kind: "cron", Expr: "0 8 * * *"},
		"run command",
		"telegram",
		"chat-1",
	)
	if err != nil {
		t.Fatalf("AddJob() error: %v", err)
	}
	job.Payload.Command = "df -h"
	if err := tool.cronService.UpdateJob(job); err != nil {
		t.Fatalf("UpdateJob() error: %v", err)
	}
	ctx := WithToolContext(context.Background(), "telegram", "chat-1")

	getResult := tool.Execute(ctx, map[string]any{"action": "get", "job_id": job.ID})
	if !getResult.IsError || !strings.Contains(getResult.ForLLM, "not accessible") {
		t.Fatalf("expected inaccessible get, got: %+v", getResult)
	}

	updateResult := tool.Execute(ctx, map[string]any{"action": "update", "job_id": job.ID, "message": "changed"})
	if !updateResult.IsError || !strings.Contains(updateResult.ForLLM, "not accessible") {
		t.Fatalf("expected inaccessible update, got: %+v", updateResult)
	}
	unchanged, ok := tool.cronService.GetJob(job.ID)
	if !ok {
		t.Fatal("job should still exist")
	}
	if unchanged.Payload.Message != "run command" || unchanged.Payload.Command != "df -h" {
		t.Fatalf("unauthorized update mutated command job: %+v", unchanged.Payload)
	}
}

func TestCronTool_CommandUpdateSafetyGates(t *testing.T) {
	t.Run("exec disabled", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Tools.Exec.Enabled = false
		tool := newTestCronToolWithConfig(t, cfg)
		ctx := WithToolContext(context.Background(), "cli", "direct")
		job, err := tool.cronService.AddJob(
			"job",
			cron.CronSchedule{Kind: "cron", Expr: "0 8 * * *"},
			"message",
			"cli",
			"direct",
		)
		if err != nil {
			t.Fatalf("AddJob() error: %v", err)
		}

		result := tool.Execute(ctx, map[string]any{
			"action":          "update",
			"job_id":          job.ID,
			"command":         "df -h",
			"command_confirm": true,
		})

		if !result.IsError || !strings.Contains(result.ForLLM, "command execution is disabled") {
			t.Fatalf("expected exec disabled error, got: %+v", result)
		}
	})

	t.Run("confirm required", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Tools.Cron.AllowCommand = false
		tool := newTestCronToolWithConfig(t, cfg)
		ctx := WithToolContext(context.Background(), "cli", "direct")
		job, err := tool.cronService.AddJob(
			"job",
			cron.CronSchedule{Kind: "cron", Expr: "0 8 * * *"},
			"message",
			"cli",
			"direct",
		)
		if err != nil {
			t.Fatalf("AddJob() error: %v", err)
		}

		result := tool.Execute(ctx, map[string]any{
			"action":  "update",
			"job_id":  job.ID,
			"command": "df -h",
		})

		if !result.IsError || !strings.Contains(result.ForLLM, "command_confirm=true") {
			t.Fatalf("expected confirm error, got: %+v", result)
		}

		result = tool.Execute(ctx, map[string]any{
			"action":          "update",
			"job_id":          job.ID,
			"command":         "df -h",
			"command_confirm": true,
		})

		if result.IsError {
			t.Fatalf("expected confirmed command update to succeed, got: %s", result.ForLLM)
		}
		updated, _ := tool.cronService.GetJob(job.ID)
		if updated.Payload.Command != "df -h" {
			t.Fatalf("command not updated: %+v", updated.Payload)
		}

		result = tool.Execute(ctx, map[string]any{
			"action":          "update",
			"job_id":          job.ID,
			"command":         "",
			"command_confirm": true,
		})

		if result.IsError {
			t.Fatalf("expected empty command update to clear command, got: %s", result.ForLLM)
		}
		updated, _ = tool.cronService.GetJob(job.ID)
		if updated.Payload.Command != "" {
			t.Fatalf("command not cleared: %+v", updated.Payload)
		}
	})
}

func TestCronTool_InternalCanAccessCommandJobFromAnyChannel(t *testing.T) {
	tool := newTestCronTool(t)
	ctx := WithToolContext(context.Background(), "cli", "direct")
	job, err := tool.cronService.AddJob(
		"command",
		cron.CronSchedule{Kind: "cron", Expr: "0 8 * * *"},
		"run command",
		"telegram",
		"chat-1",
	)
	if err != nil {
		t.Fatalf("AddJob() error: %v", err)
	}
	job.Payload.Command = "df -h"
	if err := tool.cronService.UpdateJob(job); err != nil {
		t.Fatalf("UpdateJob() error: %v", err)
	}

	getResult := tool.Execute(ctx, map[string]any{"action": "get", "job_id": job.ID})
	if getResult.IsError {
		t.Fatalf("get failed: %s", getResult.ForLLM)
	}
	got := parseCronJobResult(t, getResult)
	if got.Payload.Command != "df -h" || got.Payload.Channel != "telegram" || got.Payload.To != "chat-1" {
		t.Fatalf("get returned wrong command job: %+v", got.Payload)
	}

	updateResult := tool.Execute(ctx, map[string]any{
		"action":    "update",
		"job_id":    job.ID,
		"cron_expr": "30 10 * * *",
	})
	if updateResult.IsError {
		t.Fatalf("update failed: %s", updateResult.ForLLM)
	}
	updated, _ := tool.cronService.GetJob(job.ID)
	if updated.Payload.Command != "df -h" {
		t.Fatalf("command should be preserved: %+v", updated.Payload)
	}
	if updated.Schedule.Kind != "cron" || updated.Schedule.Expr != "30 10 * * *" {
		t.Fatalf("schedule not updated: %+v", updated.Schedule)
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
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	tool.SetTaskRegistry(registry)

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

	rec := singleCronTaskRecord(t, registry)
	if rec.Runtime != taskregistry.RuntimeCron || rec.TaskKind != "cron" {
		t.Fatalf("runtime/task_kind = %s/%s, want cron/cron", rec.Runtime, rec.TaskKind)
	}
	if rec.Status != taskregistry.StatusSucceeded {
		t.Fatalf("Status = %q, want succeeded", rec.Status)
	}
	if rec.DeliveryStatus != taskregistry.DeliveryDelivered {
		t.Fatalf("DeliveryStatus = %q, want delivered", rec.DeliveryStatus)
	}
	if rec.Channel != "telegram" || rec.ChatID != "chat-1" {
		t.Fatalf("scope = %s/%s, want telegram/chat-1", rec.Channel, rec.ChatID)
	}
	if rec.Task != "send me a poem" {
		t.Fatalf("Task = %q, want prompt", rec.Task)
	}
	if rec.TerminalSummary != "generated reply" {
		t.Fatalf("TerminalSummary = %q, want generated reply", rec.TerminalSummary)
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

func TestCronTool_ExecuteJobSkipsNoReplySentinel(t *testing.T) {
	executor := &stubJobExecutor{response: " NO_REPLY \n"}
	tool := newTestCronToolWithExecutorAndConfig(t, executor, config.DefaultConfig())

	job := &cron.CronJob{ID: "job-no-reply"}
	job.Payload.Channel = "telegram"
	job.Payload.To = "chat-1"
	job.Payload.Message = "only reply if actionable"

	if got := tool.ExecuteJob(context.Background(), job); got != "ok" {
		t.Fatalf("ExecuteJob() = %q, want ok", got)
	}

	if executor.publishedResp != "" {
		t.Fatalf("unexpected published response: %q", executor.publishedResp)
	}
}

func TestCronTool_ExecuteJobSkipsHeartbeatOKSentinel(t *testing.T) {
	executor := &stubJobExecutor{response: " HEARTBEAT_OK \n"}
	tool := newTestCronToolWithExecutorAndConfig(t, executor, config.DefaultConfig())

	job := &cron.CronJob{ID: "job-heartbeat-ok"}
	job.Payload.Channel = "telegram"
	job.Payload.To = "chat-1"
	job.Payload.Message = "heartbeat"

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

func TestCronTool_ExecuteJobRunsCommand(t *testing.T) {
	tool := newTestCronToolWithConfig(t, config.DefaultConfig())
	job := &cron.CronJob{}
	job.Payload.Channel = "cli"
	job.Payload.To = "direct"
	job.Payload.Command = "echo cron-test-ok"

	if got := tool.ExecuteJob(context.Background(), job); got != "ok" {
		t.Fatalf("ExecuteJob() = %q, want ok", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var msg bus.OutboundMessage
	select {
	case msg = <-tool.msgBus.OutboundChan():
	case <-ctx.Done():
		t.Fatal("timeout waiting for outbound message")
	}
	if !strings.Contains(msg.Content, "cron-test-ok") {
		t.Fatalf("expected command output containing 'cron-test-ok', got: %s", msg.Content)
	}
}

func TestCronTool_ExecuteJobReturnsErrorWithoutPublish(t *testing.T) {
	executor := &stubJobExecutor{
		response: "this response must not be published",
		err:      fmt.Errorf("agent failure"),
	}
	tool := newTestCronToolWithExecutorAndConfig(t, executor, config.DefaultConfig())
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	tool.SetTaskRegistry(registry)

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

	rec := singleCronTaskRecord(t, registry)
	if rec.Status != taskregistry.StatusFailed {
		t.Fatalf("Status = %q, want failed", rec.Status)
	}
	if rec.DeliveryStatus != taskregistry.DeliveryFailed {
		t.Fatalf("DeliveryStatus = %q, want failed", rec.DeliveryStatus)
	}
	if !strings.Contains(rec.Error, "agent failure") {
		t.Fatalf("Error = %q, want agent failure", rec.Error)
	}
}

func TestCronTool_ExecuteJobCommandRecordsTaskRegistry(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Exec.Enabled = false

	tool := newTestCronToolWithConfig(t, cfg)
	registry := taskregistry.NewRegistry(taskregistry.WorkspaceStorePath(t.TempDir()))
	tool.SetTaskRegistry(registry)
	job := &cron.CronJob{ID: "job-command"}
	job.Payload.Channel = "cli"
	job.Payload.To = "direct"
	job.Payload.Command = "df -h"

	if got := tool.ExecuteJob(context.Background(), job); got != "ok" {
		t.Fatalf("ExecuteJob() = %q, want ok", got)
	}

	rec := singleCronTaskRecord(t, registry)
	if rec.Status != taskregistry.StatusFailed {
		t.Fatalf("Status = %q, want failed", rec.Status)
	}
	if rec.DeliveryStatus != taskregistry.DeliveryDelivered {
		t.Fatalf("DeliveryStatus = %q, want delivered", rec.DeliveryStatus)
	}
	if rec.Task != "df -h" {
		t.Fatalf("Task = %q, want command", rec.Task)
	}
	if !strings.Contains(rec.TerminalSummary, "command execution is disabled") {
		t.Fatalf("TerminalSummary = %q, want command disabled", rec.TerminalSummary)
	}
}

func singleCronTaskRecord(t *testing.T, registry *taskregistry.Registry) taskregistry.Record {
	t.Helper()
	records := registry.List()
	if len(records) != 1 {
		t.Fatalf("task record count = %d, want 1: %+v", len(records), records)
	}
	if !strings.HasPrefix(records[0].TaskID, "cron-") {
		t.Fatalf("TaskID = %q, want cron-*", records[0].TaskID)
	}
	return records[0]
}
