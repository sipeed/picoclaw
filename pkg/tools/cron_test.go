package tools

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/cron"
)

// MockJobExecutor implements JobExecutor for testing
type MockJobExecutor struct {
	processDirectFunc func(ctx context.Context, content, sessionKey, channel, chatID string) (string, error)
}

func (m *MockJobExecutor) ProcessDirectWithChannel(ctx context.Context, content, sessionKey, channel, chatID string) (string, error) {
	if m.processDirectFunc != nil {
		return m.processDirectFunc(ctx, content, sessionKey, channel, chatID)
	}
	return "mock response", nil
}

func TestCronToolName(t *testing.T) {
	cronService := cron.NewCronService("", nil)
	tool := NewCronTool(cronService, nil, nil, "", true, 0, &config.Config{})
	assert.Equal(t, "cron", tool.Name())
}

func TestCronToolDescription(t *testing.T) {
	cronService := cron.NewCronService("", nil)
	tool := NewCronTool(cronService, nil, nil, "", true, 0, &config.Config{})
	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "Schedule")
	assert.Contains(t, desc, "reminder")
	assert.Contains(t, desc, "at_seconds")
}

func TestCronToolParameters(t *testing.T) {
	cronService := cron.NewCronService("", nil)
	tool := NewCronTool(cronService, nil, nil, "", true, 0, &config.Config{})
	params := tool.Parameters()

	props, ok := params["properties"].(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, props, "action")
	assert.Contains(t, props, "message")
	assert.Contains(t, props, "command")
	assert.Contains(t, props, "at_seconds")
	assert.Contains(t, props, "every_seconds")
	assert.Contains(t, props, "cron_expr")
	assert.Contains(t, props, "job_id")
	assert.Contains(t, props, "deliver")

	required, ok := params["required"].([]string)
	assert.True(t, ok)
	assert.Contains(t, required, "action")
}

func TestCronToolMissingAction(t *testing.T) {
	cronService := cron.NewCronService("", nil)
	tool := NewCronTool(cronService, nil, nil, "", true, 0, &config.Config{})
	result := tool.Execute(context.Background(), map[string]any{})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "action is required")
}

func TestCronToolUnknownAction(t *testing.T) {
	cronService := cron.NewCronService("", nil)
	tool := NewCronTool(cronService, nil, nil, "", true, 0, &config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"action": "invalid",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "unknown action")
}

func TestCronToolAddJobMissingMessage(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron.json")
	cronService := cron.NewCronService(storePath, nil)
	executor := &MockJobExecutor{}
	msgBus := bus.NewMessageBus()
	tool := NewCronTool(cronService, executor, msgBus, tmpDir, true, 0, &config.Config{})

	tool.SetContext("telegram", "123")

	result := tool.Execute(context.Background(), map[string]any{
		"action":     "add",
		"at_seconds": 60.0,
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "message is required")
}

func TestCronToolAddJobNoSchedule(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron.json")
	cronService := cron.NewCronService(storePath, nil)
	executor := &MockJobExecutor{}
	msgBus := bus.NewMessageBus()
	tool := NewCronTool(cronService, executor, msgBus, tmpDir, true, 0, &config.Config{})

	tool.SetContext("telegram", "123")

	result := tool.Execute(context.Background(), map[string]any{
		"action":  "add",
		"message": "test reminder",
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "one of at_seconds, every_seconds, or cron_expr is required")
}

func TestCronToolAddJobAtSeconds(t *testing.T) {
	cronService, tmpDir := newCronServiceForTest(t)
	executor := &MockJobExecutor{}
	msgBus := bus.NewMessageBus()
	tool := NewCronTool(cronService, executor, msgBus, tmpDir, true, 0, &config.Config{})

	// Set context first
	tool.SetContext("telegram", "123")

	result := tool.Execute(context.Background(), map[string]any{
		"action":     "add",
		"message":    "test reminder",
		"at_seconds": 600.0, // 10 minutes
	})

	assert.False(t, result.IsError)
	assert.Contains(t, result.ForLLM, "Cron job added")

	// Verify job was created
	jobs := cronService.ListJobs(true)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "test reminder", jobs[0].Name)
	assert.Equal(t, "at", jobs[0].Schedule.Kind)
}

func TestCronToolAddJobEverySeconds(t *testing.T) {
	cronService, tmpDir := newCronServiceForTest(t)
	executor := &MockJobExecutor{}
	msgBus := bus.NewMessageBus()
	tool := NewCronTool(cronService, executor, msgBus, tmpDir, true, 0, &config.Config{})

	tool.SetContext("telegram", "123")

	result := tool.Execute(context.Background(), map[string]any{
		"action":        "add",
		"message":       "recurring task",
		"every_seconds": 3600.0, // every hour
	})

	assert.False(t, result.IsError)
	assert.Contains(t, result.ForLLM, "Cron job added")

	jobs := cronService.ListJobs(true)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "recurring task", jobs[0].Name)
	assert.Equal(t, "every", jobs[0].Schedule.Kind)
}

func TestCronToolAddJobCronExpr(t *testing.T) {
	cronService, tmpDir := newCronServiceForTest(t)
	executor := &MockJobExecutor{}
	msgBus := bus.NewMessageBus()
	tool := NewCronTool(cronService, executor, msgBus, tmpDir, true, 0, &config.Config{})

	tool.SetContext("telegram", "123")

	result := tool.Execute(context.Background(), map[string]any{
		"action":    "add",
		"message":   "daily task",
		"cron_expr": "0 9 * * *", // daily at 9am
	})

	assert.False(t, result.IsError)
	assert.Contains(t, result.ForLLM, "Cron job added")

	jobs := cronService.ListJobs(true)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "daily task", jobs[0].Name)
	assert.Equal(t, "cron", jobs[0].Schedule.Kind)
	assert.Equal(t, "0 9 * * *", jobs[0].Schedule.Expr)
}

func TestCronToolAddJobWithContext(t *testing.T) {
	cronService := cron.NewCronService("", nil)
	executor := &MockJobExecutor{}
	msgBus := bus.NewMessageBus()
	workspace := t.TempDir()
	tool := NewCronTool(cronService, executor, msgBus, workspace, true, 0, &config.Config{})

	// Without context should fail
	result := tool.Execute(context.Background(), map[string]any{
		"action":     "add",
		"message":    "test",
		"at_seconds": 60.0,
	})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "no session context")
}

func TestCronToolAddJobWithCommand(t *testing.T) {
	cronService, tmpDir := newCronServiceForTest(t)
	executor := &MockJobExecutor{}
	msgBus := bus.NewMessageBus()
	tool := NewCronTool(cronService, executor, msgBus, tmpDir, true, 0, &config.Config{})

	tool.SetContext("telegram", "123")

	result := tool.Execute(context.Background(), map[string]any{
		"action":     "add",
		"message":    "check disk space",
		"command":    "df -h",
		"at_seconds": 60.0,
	})

	assert.False(t, result.IsError)
	assert.Contains(t, result.ForLLM, "Cron job added")

	jobs := cronService.ListJobs(true)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "check disk space", jobs[0].Name)
	assert.Equal(t, "df -h", jobs[0].Payload.Command)
	assert.False(t, jobs[0].Payload.Deliver) // command forces deliver=false
}

func TestCronToolListJobsEmpty(t *testing.T) {
	cronService, tmpDir := newCronServiceForTest(t)
	tool := NewCronTool(cronService, nil, nil, tmpDir, true, 0, &config.Config{})

	result := tool.Execute(context.Background(), map[string]any{
		"action": "list",
	})

	assert.False(t, result.IsError)
	assert.Contains(t, result.ForLLM, "No scheduled jobs")
}

func TestCronToolListJobs(t *testing.T) {
	cronService, tmpDir := newCronServiceForTest(t)
	executor := &MockJobExecutor{}
	msgBus := bus.NewMessageBus()
	tool := NewCronTool(cronService, executor, msgBus, tmpDir, true, 0, &config.Config{})

	tool.SetContext("telegram", "123")

	// Add multiple jobs
	tool.Execute(context.Background(), map[string]any{
		"action":     "add",
		"message":    "reminder 1",
		"at_seconds": 60.0,
	})
	tool.Execute(context.Background(), map[string]any{
		"action":        "add",
		"message":       "reminder 2",
		"every_seconds": 3600.0,
	})

	result := tool.Execute(context.Background(), map[string]any{
		"action": "list",
	})

	assert.False(t, result.IsError)
	assert.Contains(t, result.ForLLM, "Scheduled jobs:")
	assert.Contains(t, result.ForLLM, "reminder 1")
	assert.Contains(t, result.ForLLM, "reminder 2")
}

func TestCronToolRemoveJob(t *testing.T) {
	cronService, tmpDir := newCronServiceForTest(t)
	executor := &MockJobExecutor{}
	msgBus := bus.NewMessageBus()
	tool := NewCronTool(cronService, executor, msgBus, tmpDir, true, 0, &config.Config{})

	tool.SetContext("telegram", "123")

	// Add a job first
	addResult := tool.Execute(context.Background(), map[string]any{
		"action":     "add",
		"message":    "to be removed",
		"at_seconds": 60.0,
	})
	if !addResult.IsError {
		// Extract job ID from result
		var jobID string
		for _, line := range splitLines(addResult.ForLLM) {
			if contains(line, "id:") {
				jobID = extractJobID(line)
				break
			}
		}

		if jobID != "" {
			// Remove the job
			result := tool.Execute(context.Background(), map[string]any{
				"action": "remove",
				"job_id": jobID,
			})

			assert.False(t, result.IsError)
			assert.Contains(t, result.ForLLM, "Cron job removed")

			// Verify job is removed
			jobs := cronService.ListJobs(true)
			assert.Empty(t, jobs)
		}
	}
}

func TestCronToolRemoveJobNotFound(t *testing.T) {
	cronService, tmpDir := newCronServiceForTest(t)
	tool := NewCronTool(cronService, nil, nil, tmpDir, true, 0, &config.Config{})

	result := tool.Execute(context.Background(), map[string]any{
		"action": "remove",
		"job_id": "nonexistent",
	})

	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "not found")
}

func TestCronToolEnableDisableJob(t *testing.T) {
	cronService, tmpDir := newCronServiceForTest(t)
	executor := &MockJobExecutor{}
	msgBus := bus.NewMessageBus()
	tool := NewCronTool(cronService, executor, msgBus, tmpDir, true, 0, &config.Config{})

	tool.SetContext("telegram", "123")

	// Add a job first
	addResult := tool.Execute(context.Background(), map[string]any{
		"action":        "add",
		"message":       "toggle job",
		"every_seconds": 3600.0,
	})
	if !addResult.IsError {
		jobID := extractJobID(addResult.ForLLM)

		if jobID != "" {
			// Disable the job
			result := tool.Execute(context.Background(), map[string]any{
				"action": "disable",
				"job_id": jobID,
			})

			assert.False(t, result.IsError)
			assert.Contains(t, result.ForLLM, "disabled")

			// Enable the job
			result = tool.Execute(context.Background(), map[string]any{
				"action": "enable",
				"job_id": jobID,
			})

			assert.False(t, result.IsError)
			assert.Contains(t, result.ForLLM, "enabled")
		}
	}
}

func TestCronToolEnableDisableJobNotFound(t *testing.T) {
	cronService, tmpDir := newCronServiceForTest(t)
	tool := NewCronTool(cronService, nil, nil, tmpDir, true, 0, &config.Config{})

	result := tool.Execute(context.Background(), map[string]any{
		"action": "enable",
		"job_id": "nonexistent",
	})

	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "not found")
}

func TestCronToolSetContext(t *testing.T) {
	cronService, tmpDir := newCronServiceForTest(t)
	tool := NewCronTool(cronService, nil, nil, tmpDir, true, 0, &config.Config{})

	tool.SetContext("discord", "456")

	// Context is set internally, we verify by checking if add_job works
	// This is tested in TestCronToolAddJobWithContext
}

// Helper functions
func splitLines(s string) []string {
	var lines []string
	current := ""
	for _, c := range s {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func extractJobID(s string) string {
	// Simple extraction: look for "id: xxx" pattern
	start := findSubstringIndex(s, "id: ")
	if start == -1 {
		return ""
	}
	start += 4 // skip "id: "
	end := start
	for end < len(s) && s[end] != ')' && s[end] != ',' && s[end] != ' ' {
		end++
	}
	return s[start:end]
}

func findSubstringIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Helper function to create cron service for tests
func newCronServiceForTest(t *testing.T) (*cron.CronService, string) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "cron.json")
	cs := cron.NewCronService(storePath, nil)
	return cs, tmpDir
}
