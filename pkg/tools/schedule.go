package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/cron"
)

// ScheduleTool provides a scheduling interface for the AI agent
// to create, list, and cancel scheduled tasks.
type ScheduleTool struct {
	cronService *cron.CronService
	channel     string
	chatID      string
	mu          sync.RWMutex
}

// NewScheduleTool creates a new ScheduleTool
func NewScheduleTool(cronService *cron.CronService) *ScheduleTool {
	return &ScheduleTool{
		cronService: cronService,
	}
}

// Name returns the tool name
func (t *ScheduleTool) Name() string {
	return "schedule"
}

// Description returns the tool description
func (t *ScheduleTool) Description() string {
	return "Create, list, or cancel scheduled tasks. Supports one-time tasks (at), recurring intervals (every), and cron expressions."
}

// Parameters returns the tool parameters schema
func (t *ScheduleTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"create", "list", "cancel"},
				"description": "Action to perform: 'create' a new job, 'list' all jobs, or 'cancel' a job.",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Job name (required for create). A descriptive name for the scheduled task.",
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Message content when job executes (required for create). This is what will be sent when the task triggers.",
			},
			"schedule": map[string]interface{}{
				"type":        "object",
				"description": "Schedule configuration (required for create). Defines when and how often the task should run.",
				"properties": map[string]interface{}{
					"kind": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"at", "every", "cron"},
						"description": "Schedule type: 'at' for one-time at specific time, 'every' for recurring interval, 'cron' for cron expression.",
					},
					"at": map[string]interface{}{
						"type":        "string",
						"description": "ISO datetime for 'at' kind (e.g., '2024-01-01T12:00:00'). Use format: YYYY-MM-DDTHH:MM:SS.",
					},
					"every_seconds": map[string]interface{}{
						"type":        "integer",
						"description": "Interval in seconds for 'every' kind. Example: 3600 for every hour. Must be a positive integer.",
					},
					"expr": map[string]interface{}{
						"type":        "string",
						"description": "Cron expression for 'cron' kind (e.g., '0 9 * * *' for daily at 9am). Format: min hour day month dow.",
					},
					"timezone": map[string]interface{}{
						"type":        "string",
						"description": "Timezone for interpreting 'at' times without explicit offset (e.g., 'Asia/Shanghai', 'UTC'). Only applies to 'at' kind. If not specified, uses system timezone.",
					},
				},
				"required": []string{"kind"},
			},
			"deliver": map[string]interface{}{
				"type":        "boolean",
				"description": "Send result to user when job fires (default: true). If true, the message is delivered directly. If false, it's processed by the agent.",
			},
			"id": map[string]interface{}{
				"type":        "string",
				"description": "Job ID to cancel (required for cancel action). Use the ID returned when creating a job.",
			},
		},
		"required": []string{"action"},
	}
}

// SetContext sets the current session context for job creation
func (t *ScheduleTool) SetContext(channel, chatID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.channel = channel
	t.chatID = chatID
}

// Execute runs the tool with the given arguments
func (t *ScheduleTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required and must be one of: create, list, cancel")
	}

	switch action {
	case "create":
		return t.createJob(ctx, args)
	case "list":
		return t.listJobs()
	case "cancel":
		return t.cancelJob(args)
	default:
		return ErrorResult(fmt.Sprintf("invalid action: %s (must be create, list, or cancel)", action))
	}
}

// createJob creates a new scheduled job
func (t *ScheduleTool) createJob(ctx context.Context, args map[string]interface{}) *ToolResult {
	t.mu.RLock()
	channel := t.channel
	chatID := t.chatID
	t.mu.RUnlock()

	if channel == "" || chatID == "" {
		return ErrorResult("no session context (channel/chat_id not set). Use this tool in an active conversation.")
	}

	// Get required parameters
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return ErrorResult("name is required for create action")
	}

	message, ok := args["message"].(string)
	if !ok || message == "" {
		return ErrorResult("message is required for create action")
	}

	// Get schedule configuration
	scheduleArg, ok := args["schedule"]
	if !ok {
		return ErrorResult("schedule is required for create action")
	}

	scheduleMap, ok := scheduleArg.(map[string]interface{})
	if !ok {
		return ErrorResult("schedule must be an object with kind property")
	}

	schedule, err := t.parseSchedule(scheduleMap)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid schedule: %v", err))
	}

	// Get deliver parameter, default to true
	deliver := true
	if d, ok := args["deliver"].(bool); ok {
		deliver = d
	}

	// Add job via cron service
	job, err := t.cronService.AddJob(name, schedule, message, deliver, channel, chatID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("error creating job: %v", err))
	}

	// Format next run time for response
	var nextRunInfo string
	if job.State.NextRunAtMS != nil {
		nextTime := time.UnixMilli(*job.State.NextRunAtMS)
		nextRunInfo = fmt.Sprintf(", next run: %s", nextTime.Format("2006-01-02 15:04:05"))
	}

	return SilentResult(fmt.Sprintf("Scheduled job '%s' (id: %s%s)", job.Name, job.ID, nextRunInfo))
}

// parseSchedule parses the schedule configuration into a CronSchedule
func (t *ScheduleTool) parseSchedule(scheduleMap map[string]interface{}) (cron.CronSchedule, error) {
	kind, ok := scheduleMap["kind"].(string)
	if !ok {
		return cron.CronSchedule{}, fmt.Errorf("kind is required in schedule")
	}

	var schedule cron.CronSchedule
	schedule.Kind = kind

	// Get timezone if specified
	tz, _ := scheduleMap["timezone"].(string)
	schedule.TZ = tz

	switch kind {
	case "at":
		atStr, ok := scheduleMap["at"].(string)
		if !ok || atStr == "" {
			return cron.CronSchedule{}, fmt.Errorf("at field is required for 'at' kind")
		}

		// Determine location to use for parsing times without explicit offset
		var loc *time.Location
		if tz != "" {
			var err error
			loc, err = time.LoadLocation(tz)
			if err != nil {
				return cron.CronSchedule{}, fmt.Errorf("invalid timezone: %v", err)
			}
		} else {
			loc = time.Local
		}

		// Parse ISO datetime (with explicit offset)
		atTime, err := time.Parse(time.RFC3339, atStr)
		usedNaiveLayout := false
		if err != nil {
			// Try other common formats (without timezone, interpret in loc)
			atTime, err = time.ParseInLocation("2006-01-02T15:04:05", atStr, loc)
			if err != nil {
				return cron.CronSchedule{}, fmt.Errorf("invalid at datetime format: use ISO format like '2024-01-01T12:00:00'")
			}
			usedNaiveLayout = true
		}

		// Apply timezone only for offset-aware inputs (naive inputs already parsed in loc)
		if tz != "" && !usedNaiveLayout {
			atTime = atTime.In(loc)
		}

		atMS := atTime.UnixMilli()
		schedule.AtMS = &atMS

	case "every":
		everySeconds, ok := scheduleMap["every_seconds"].(float64)
		if !ok {
			return cron.CronSchedule{}, fmt.Errorf("every_seconds field is required for 'every' kind")
		}
		if everySeconds <= 0 {
			return cron.CronSchedule{}, fmt.Errorf("every_seconds must be positive")
		}
		// Reject non-integer values to avoid surprising schedules
		if everySeconds != float64(int64(everySeconds)) {
			return cron.CronSchedule{}, fmt.Errorf("every_seconds must be an integer value, got %.2f", everySeconds)
		}
		everyMS := int64(everySeconds) * 1000
		schedule.EveryMS = &everyMS

	case "cron":
		expr, ok := scheduleMap["expr"].(string)
		if !ok || expr == "" {
			return cron.CronSchedule{}, fmt.Errorf("expr field is required for 'cron' kind")
		}
		schedule.Expr = expr

	default:
		return cron.CronSchedule{}, fmt.Errorf("invalid schedule kind: %s (must be at, every, or cron)", kind)
	}

	return schedule, nil
}

// listJobs lists all scheduled jobs
func (t *ScheduleTool) listJobs() *ToolResult {
	jobs := t.cronService.ListJobs(true) // Include disabled jobs too

	if len(jobs) == 0 {
		return SilentResult("No scheduled jobs")
	}

	result := "Scheduled jobs:\n"
	for _, j := range jobs {
		var scheduleInfo string
		switch j.Schedule.Kind {
		case "every":
			if j.Schedule.EveryMS != nil {
				scheduleInfo = fmt.Sprintf("every %ds", *j.Schedule.EveryMS/1000)
			} else {
				scheduleInfo = "every (unknown interval)"
			}
		case "cron":
			scheduleInfo = fmt.Sprintf("cron: %s", j.Schedule.Expr)
		case "at":
			scheduleInfo = "one-time"
		default:
			scheduleInfo = "unknown"
		}

		// Add next run time if available
		var nextRun string
		if j.State.NextRunAtMS != nil {
			nextTime := time.UnixMilli(*j.State.NextRunAtMS)
			nextRun = fmt.Sprintf(", next: %s", nextTime.Format("2006-01-02 15:04:05"))
		}

		// Add status
		status := "enabled"
		if !j.Enabled {
			status = "disabled"
		}

		result += fmt.Sprintf("- %s [%s] (id: %s, %s%s)\n", j.Name, status, j.ID, scheduleInfo, nextRun)
	}

	return SilentResult(result)
}

// cancelJob cancels/removes a scheduled job
func (t *ScheduleTool) cancelJob(args map[string]interface{}) *ToolResult {
	jobID, ok := args["id"].(string)
	if !ok || jobID == "" {
		return ErrorResult("id is required for cancel action")
	}

	if t.cronService.RemoveJob(jobID) {
		return SilentResult(fmt.Sprintf("Cancelled job: %s", jobID))
	}
	return ErrorResult(fmt.Sprintf("job not found: %s", jobID))
}
