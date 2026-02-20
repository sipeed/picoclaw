package tools

import (
	"context"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/cron"
)

func TestScheduleTool_Name(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)

	if st.Name() != "schedule" {
		t.Errorf("expected name 'schedule', got '%s'", st.Name())
	}
}

func TestScheduleTool_Description(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)

	desc := st.Description()
	if desc == "" {
		t.Error("description should not be empty")
	}
}

func TestScheduleTool_Parameters(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)

	params := st.Parameters()
	if params["type"] != "object" {
		t.Error("parameters should be type object")
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties should be a map")
	}

	// Check required fields
	required, ok := params["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "action" {
		t.Error("action should be required")
	}

	// Check schedule properties
	schedule, ok := props["schedule"].(map[string]interface{})
	if !ok {
		t.Fatal("schedule should be a map")
	}

	scheduleProps, ok := schedule["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("schedule properties should be a map")
	}

	// Verify kind, at, every_seconds, expr, timezone exist
	for _, key := range []string{"kind", "at", "every_seconds", "expr", "timezone"} {
		if _, ok := scheduleProps[key]; !ok {
			t.Errorf("schedule.%s should exist", key)
		}
	}
}

func TestScheduleTool_SetContext(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)

	st.SetContext("test-channel", "test-chat-id")

	// Can't directly access private fields, but we can verify it doesn't panic
	st.SetContext("another-channel", "another-chat-id")
}

func TestScheduleTool_Execute_InvalidAction(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{"action": "invalid"})
	if !result.IsError {
		t.Error("invalid action should return error")
	}

	result = st.Execute(ctx, map[string]interface{}{})
	if !result.IsError {
		t.Error("missing action should return error")
	}
}

func TestScheduleTool_Execute_ListEmpty(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{"action": "list"})
	if result.IsError {
		t.Errorf("list should not error: %s", result.ForLLM)
	}
	if result.ForLLM != "No scheduled jobs" {
		t.Errorf("expected 'No scheduled jobs', got '%s'", result.ForLLM)
	}
}

func TestScheduleTool_Execute_Create_NoContext(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"name":    "Test Job",
		"message": "Test message",
		"schedule": map[string]interface{}{
			"kind": "every",
			"every_seconds": float64(60),
		},
	})

	if !result.IsError {
		t.Error("create without context should return error")
	}
}

func TestScheduleTool_Execute_Create_MissingName(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action": "create",
		"schedule": map[string]interface{}{
			"kind": "every",
			"every_seconds": float64(60),
		},
	})

	if !result.IsError {
		t.Error("create without name should return error")
	}
}

func TestScheduleTool_Execute_Create_MissingMessage(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action": "create",
		"name": "Test Job",
		"schedule": map[string]interface{}{
			"kind": "every",
			"every_seconds": float64(60),
		},
	})

	if !result.IsError {
		t.Error("create without message should return error")
	}
}

func TestScheduleTool_Execute_Create_MissingSchedule(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"name":    "Test Job",
		"message": "Test message",
	})

	if !result.IsError {
		t.Error("create without schedule should return error")
	}
}

func TestScheduleTool_Execute_Create_Every(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"name":    "Test Every Job",
		"message": "Test message",
		"schedule": map[string]interface{}{
			"kind":          "every",
			"every_seconds": float64(60),
		},
	})

	if result.IsError {
		t.Errorf("create every job should succeed: %s", result.ForLLM)
	}

	// Verify job was created
	jobs := cs.ListJobs(true)
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Name != "Test Every Job" {
		t.Errorf("expected job name 'Test Every Job', got '%s'", jobs[0].Name)
	}
}

func TestScheduleTool_Execute_Create_Cron(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"name":    "Daily Job",
		"message": "Daily message",
		"schedule": map[string]interface{}{
			"kind": "cron",
			"expr": "0 9 * * *",
		},
	})

	if result.IsError {
		t.Errorf("create cron job should succeed: %s", result.ForLLM)
	}

	jobs := cs.ListJobs(true)
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
}

func TestScheduleTool_Execute_Create_At(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	// Use a future time
	futureTime := time.Now().Add(1 * time.Hour).Format("2006-01-02T15:04:05")

	result := st.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"name":    "One-time Job",
		"message": "One-time message",
		"schedule": map[string]interface{}{
			"kind": "at",
			"at":   futureTime,
		},
	})

	if result.IsError {
		t.Errorf("create at job should succeed: %s", result.ForLLM)
	}

	jobs := cs.ListJobs(true)
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Schedule.Kind != "at" {
		t.Errorf("expected schedule kind 'at', got '%s'", jobs[0].Schedule.Kind)
	}
}

func TestScheduleTool_Execute_Create_At_WithTimezone(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	// Use a naive datetime with timezone
	futureTime := time.Now().Add(1 * time.Hour).Format("2006-01-02T15:04:05")

	result := st.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"name":    "TZ Job",
		"message": "TZ message",
		"schedule": map[string]interface{}{
			"kind":     "at",
			"at":       futureTime,
			"timezone": "UTC",
		},
	})

	if result.IsError {
		t.Errorf("create at job with timezone should succeed: %s", result.ForLLM)
	}

	jobs := cs.ListJobs(true)
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Schedule.TZ != "UTC" {
		t.Errorf("expected TZ 'UTC', got '%s'", jobs[0].Schedule.TZ)
	}
}

func TestScheduleTool_Execute_Create_Every_NonInteger(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"name":    "Test Job",
		"message": "Test message",
		"schedule": map[string]interface{}{
			"kind":          "every",
			"every_seconds": float64(60.5),
		},
	})

	if !result.IsError {
		t.Error("create with non-integer every_seconds should return error")
	}
}

func TestScheduleTool_Execute_Create_Every_Negative(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"name":    "Test Job",
		"message": "Test message",
		"schedule": map[string]interface{}{
			"kind":          "every",
			"every_seconds": float64(-60),
		},
	})

	if !result.IsError {
		t.Error("create with negative every_seconds should return error")
	}
}

func TestScheduleTool_Execute_Create_InvalidKind(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"name":    "Test Job",
		"message": "Test message",
		"schedule": map[string]interface{}{
			"kind": "invalid",
		},
	})

	if !result.IsError {
		t.Error("create with invalid kind should return error")
	}
}

func TestScheduleTool_Execute_Create_At_InvalidFormat(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"name":    "Test Job",
		"message": "Test message",
		"schedule": map[string]interface{}{
			"kind": "at",
			"at":   "invalid-date",
		},
	})

	if !result.IsError {
		t.Error("create with invalid date format should return error")
	}
}

func TestScheduleTool_Execute_Create_At_MissingAt(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"name":    "Test Job",
		"message": "Test message",
		"schedule": map[string]interface{}{
			"kind": "at",
		},
	})

	if !result.IsError {
		t.Error("create at job without at field should return error")
	}
}

func TestScheduleTool_Execute_Create_Every_MissingSeconds(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"name":    "Test Job",
		"message": "Test message",
		"schedule": map[string]interface{}{
			"kind": "every",
		},
	})

	if !result.IsError {
		t.Error("create every job without every_seconds should return error")
	}
}

func TestScheduleTool_Execute_Create_Cron_MissingExpr(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"name":    "Test Job",
		"message": "Test message",
		"schedule": map[string]interface{}{
			"kind": "cron",
		},
	})

	if !result.IsError {
		t.Error("create cron job without expr should return error")
	}
}

func TestScheduleTool_Execute_Create_InvalidTimezone(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	futureTime := time.Now().Add(1 * time.Hour).Format("2006-01-02T15:04:05")

	result := st.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"name":    "Test Job",
		"message": "Test message",
		"schedule": map[string]interface{}{
			"kind":     "at",
			"at":       futureTime,
			"timezone": "Invalid/Timezone",
		},
	})

	if !result.IsError {
		t.Error("create with invalid timezone should return error")
	}
}

func TestScheduleTool_Execute_ListWithJobs(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	// Create a job first
	_, err := cs.AddJob("Test Job", cron.CronSchedule{
		Kind:     "every",
		EveryMS:  func() *int64 { v := int64(60000); return &v }(),
	}, "Test message", true, "test-channel", "test-chat-id")
	if err != nil {
		t.Fatalf("failed to create test job: %v", err)
	}

	result := st.Execute(ctx, map[string]interface{}{"action": "list"})
	if result.IsError {
		t.Errorf("list should not error: %s", result.ForLLM)
	}
	if result.ForLLM == "No scheduled jobs" {
		t.Error("should show jobs, not 'No scheduled jobs'")
	}
}

func TestScheduleTool_Execute_Cancel_MissingID(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action": "cancel",
	})

	if !result.IsError {
		t.Error("cancel without id should return error")
	}
}

func TestScheduleTool_Execute_Cancel_NotFound(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action": "cancel",
		"id":     "nonexistent-id",
	})

	if !result.IsError {
		t.Error("cancel with non-existent id should return error")
	}
}

func TestScheduleTool_Execute_Cancel_Success(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	// Create a job first
	job, err := cs.AddJob("Test Job", cron.CronSchedule{
		Kind:    "every",
		EveryMS: func() *int64 { v := int64(60000); return &v }(),
	}, "Test message", true, "test-channel", "test-chat-id")
	if err != nil {
		t.Fatalf("failed to create test job: %v", err)
	}

	result := st.Execute(ctx, map[string]interface{}{
		"action": "cancel",
		"id":     job.ID,
	})

	if result.IsError {
		t.Errorf("cancel should succeed: %s", result.ForLLM)
	}

	// Verify job was removed
	jobs := cs.ListJobs(true)
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs after cancel, got %d", len(jobs))
	}
}

func TestScheduleTool_Execute_DeliverFalse(t *testing.T) {
	cs := cron.NewCronService(t.TempDir()+"/test.json", nil)
	st := NewScheduleTool(cs)
	st.SetContext("test-channel", "test-chat-id")
	ctx := context.Background()

	result := st.Execute(ctx, map[string]interface{}{
		"action":  "create",
		"name":    "Test Job",
		"message": "Test message",
		"schedule": map[string]interface{}{
			"kind":          "every",
			"every_seconds": float64(60),
		},
		"deliver": false,
	})

	if result.IsError {
		t.Errorf("create with deliver=false should succeed: %s", result.ForLLM)
	}

	jobs := cs.ListJobs(true)
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Payload.Deliver {
		t.Error("expected Deliver to be false")
	}
}
