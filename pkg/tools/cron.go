package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/cron"
	taskregistry "github.com/sipeed/picoclaw/pkg/tasks"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// JobExecutor is the interface for executing cron jobs through the agent
type JobExecutor interface {
	ProcessDirectWithChannel(ctx context.Context, content, sessionKey, channel, chatID string) (string, error)
	// PublishResponseIfNeeded sends response to the outbound bus only when the
	// agent did not already deliver content through the message tool in this round.
	PublishResponseIfNeeded(ctx context.Context, channel, chatID, sessionKey, response string)
}

type scheduledJobExecutor interface {
	ProcessScheduledWithChannel(ctx context.Context, content, sessionKey, channel, chatID string) (string, error)
}

// CronTool provides scheduling capabilities for the agent
type CronTool struct {
	cronService  *cron.CronService
	executor     JobExecutor
	msgBus       *bus.MessageBus
	execTool     *ExecTool
	allowCommand bool
	execEnabled  bool
	taskRegistry *taskregistry.Registry
}

// NewCronTool creates a new CronTool
// execTimeout: 0 means no timeout, >0 sets the timeout duration
func NewCronTool(
	cronService *cron.CronService, executor JobExecutor, msgBus *bus.MessageBus, workspace string, restrict bool,
	execTimeout time.Duration, config *config.Config,
) (*CronTool, error) {
	allowCommand := true
	execEnabled := true
	if config != nil {
		allowCommand = config.Tools.Cron.AllowCommand
		execEnabled = config.Tools.Exec.Enabled
	}

	var execTool *ExecTool
	if execEnabled {
		var err error
		execTool, err = NewExecToolWithConfig(workspace, restrict, config)
		if err != nil {
			return nil, fmt.Errorf("unable to configure exec tool: %w", err)
		}
	}

	if execTool != nil {
		execTool.SetTimeout(execTimeout)
	}
	return &CronTool{
		cronService:  cronService,
		executor:     executor,
		msgBus:       msgBus,
		execTool:     execTool,
		allowCommand: allowCommand,
		execEnabled:  execEnabled,
	}, nil
}

func (t *CronTool) SetTaskRegistry(registry *taskregistry.Registry) {
	if t != nil {
		t.taskRegistry = registry
	}
}

// Name returns the tool name
func (t *CronTool) Name() string {
	return "cron"
}

// Description returns the tool description
func (t *CronTool) Description() string {
	return `Schedule, inspect, and update reminders, tasks, or system commands. 
IMPORTANT: When user asks to be reminded or scheduled, you MUST call this tool. 
Use 'at_seconds' for one-time reminders (e.g., 'remind me in 10 minutes' → at_seconds=600). 
Use 'every_seconds' ONLY for recurring tasks (e.g., 'every 2 hours' → every_seconds=7200). 
Use 'cron_expr' for complex recurring schedules. 
Use 'command' to execute shell commands directly.`
}

// Parameters returns the tool parameters schema
//
//nolint:dupl // Tool parameter schemas intentionally use similar JSON-schema map literals.
func (t *CronTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"add", "list", "get", "update", "remove", "enable", "disable"},
				"description": "Action to perform. Use 'get' before editing and 'update' to change existing jobs without losing their payload. Remote channels can only list/get/update jobs for the current channel/chat_id.",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Optional job display name for update or add.",
			},
			"message": map[string]any{
				"type":        "string",
				"description": "The reminder/task message to display when triggered. If 'command' is used, this describes what the command does.",
			},
			"command": map[string]any{
				"type":        "string",
				"description": "Optional: Shell command to execute directly (e.g., 'df -h'). If set, the agent will run this command and report output instead of just showing the message. For update, omit to preserve the command or pass an empty string to clear it.",
			},
			"command_confirm": map[string]any{
				"type":        "boolean",
				"description": "Optional explicit confirmation flag for scheduling a shell command. Command execution must also be enabled via tools.cron.allow_command.",
			},
			"at_seconds": map[string]any{
				"type":        "integer",
				"description": "One-time reminder: seconds from now when to trigger (e.g., 600 for 10 minutes later). Use this for one-time reminders like 'remind me in 10 minutes'.",
			},
			"every_seconds": map[string]any{
				"type":        "integer",
				"description": "Recurring interval in seconds (e.g., 3600 for every hour). Use this ONLY for recurring tasks like 'every 2 hours' or 'daily reminder'.",
			},
			"cron_expr": map[string]any{
				"type":        "string",
				"description": "Cron expression for complex recurring schedules (e.g., '0 9 * * *' for daily at 9am). Use this for complex recurring schedules.",
			},
			"job_id": map[string]any{
				"type":        "string",
				"description": "Job ID (for get/update/remove/enable/disable)",
			},
		},
		"required": []string{"action"},
	}
}

// Execute runs the tool with the given arguments
func (t *CronTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}

	switch action {
	case "add":
		return t.addJob(ctx, args)
	case "list":
		return t.listJobs(ctx)
	case "get":
		return t.getJob(ctx, args)
	case "update":
		return t.updateJob(ctx, args)
	case "remove":
		return t.removeJob(args)
	case "enable":
		return t.enableJob(args, true)
	case "disable":
		return t.enableJob(args, false)
	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

func (t *CronTool) addJob(ctx context.Context, args map[string]any) *ToolResult {
	channel := ToolChannel(ctx)
	chatID := ToolChatID(ctx)

	if channel == "" || chatID == "" {
		return ErrorResult("no session context (channel/chat_id not set). Use this tool in an active conversation.")
	}

	message, ok := args["message"].(string)
	if !ok || message == "" {
		return ErrorResult("message is required for add")
	}

	var schedule cron.CronSchedule

	// Check for at_seconds (one-time), every_seconds (recurring), or cron_expr
	atSeconds, hasAt := args["at_seconds"].(float64)
	everySeconds, hasEvery := args["every_seconds"].(float64)
	cronExpr, hasCron := args["cron_expr"].(string)

	// Fix: type assertions return true for zero values, need additional validity checks
	// This prevents LLMs that fill unused optional parameters with defaults (0) from triggering wrong type
	hasAt = hasAt && atSeconds > 0
	hasEvery = hasEvery && everySeconds > 0
	hasCron = hasCron && cronExpr != ""

	// Priority: at_seconds > every_seconds > cron_expr
	if hasAt {
		atMS := time.Now().UnixMilli() + int64(atSeconds)*1000
		schedule = cron.CronSchedule{
			Kind: "at",
			AtMS: &atMS,
		}
	} else if hasEvery {
		everyMS := int64(everySeconds) * 1000
		schedule = cron.CronSchedule{
			Kind:    "every",
			EveryMS: &everyMS,
		}
	} else if hasCron {
		schedule = cron.CronSchedule{
			Kind: "cron",
			Expr: cronExpr,
		}
	} else {
		return ErrorResult("one of at_seconds, every_seconds, or cron_expr is required")
	}

	// GHSA-pv8c-p6jf-3fpp: command scheduling requires internal channel. When
	// allow_command is disabled, explicit confirmation is required as an override.
	// Non-command reminders remain open to all channels.
	command, _ := args["command"].(string)
	commandConfirm, _ := args["command_confirm"].(bool)
	if command != "" {
		if !t.execEnabled {
			return ErrorResult("command execution is disabled")
		}
		if !constants.IsInternalChannel(channel) {
			return ErrorResult("scheduling command execution is restricted to internal channels")
		}
		if !t.allowCommand && !commandConfirm {
			return ErrorResult("command_confirm=true is required when allow_command is disabled")
		}
	}

	// Truncate message for job name (max 30 chars)
	messagePreview := utils.Truncate(message, 30)

	job, err := t.cronService.AddJob(
		messagePreview,
		schedule,
		message,
		channel,
		chatID,
	)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Error adding job: %v", err))
	}

	// Apply optional payload fields and persist in a single UpdateJob call
	needsUpdate := false
	if command != "" {
		job.Payload.Command = command
		needsUpdate = true
	}
	if needsUpdate {
		t.cronService.UpdateJob(job)
	}

	return SilentResult(fmt.Sprintf("Cron job added: %s (id: %s)", job.Name, job.ID))
}

func (t *CronTool) listJobs(ctx context.Context) *ToolResult {
	jobs := t.cronService.ListJobs(false)

	var accessibleJobs []cron.CronJob
	for _, job := range jobs {
		if t.canAccessJob(ctx, &job) {
			accessibleJobs = append(accessibleJobs, job)
		}
	}
	jobs = accessibleJobs

	if len(jobs) == 0 {
		return SilentResult("No scheduled jobs")
	}

	var result strings.Builder
	result.WriteString("Scheduled jobs:\n")
	for _, j := range jobs {
		var scheduleInfo string
		if j.Schedule.Kind == "every" && j.Schedule.EveryMS != nil {
			scheduleInfo = fmt.Sprintf("every %ds", *j.Schedule.EveryMS/1000)
		} else if j.Schedule.Kind == "cron" {
			scheduleInfo = j.Schedule.Expr
		} else if j.Schedule.Kind == "at" {
			scheduleInfo = "one-time"
		} else {
			scheduleInfo = "unknown"
		}
		result.WriteString(fmt.Sprintf("- %s (id: %s, %s)\n", j.Name, j.ID, scheduleInfo))
	}

	return SilentResult(result.String())
}

func (t *CronTool) getJob(ctx context.Context, args map[string]any) *ToolResult {
	jobID, errResult := requiredCronJobID(args, "get")
	if errResult != nil {
		return errResult
	}

	job, ok := t.cronService.GetJob(jobID)
	if !ok {
		return ErrorResult(fmt.Sprintf("Job %s not found", jobID))
	}
	if !t.canAccessJob(ctx, job) {
		return ErrorResult(fmt.Sprintf("Job %s is not accessible from this channel", jobID))
	}

	return SilentResult(formatCronJobJSON(job))
}

func (t *CronTool) updateJob(ctx context.Context, args map[string]any) *ToolResult {
	jobID, errResult := requiredCronJobID(args, "update")
	if errResult != nil {
		return errResult
	}

	job, ok := t.cronService.GetJob(jobID)
	if !ok {
		return ErrorResult(fmt.Sprintf("Job %s not found", jobID))
	}
	if !t.canAccessJob(ctx, job) {
		return ErrorResult(fmt.Sprintf("Job %s is not accessible from this channel", jobID))
	}

	patches := 0

	name, namePresent, nameErr := optionalNonEmptyString(args, "name")
	if nameErr != nil {
		return nameErr
	}
	if namePresent {
		job.Name = name
		patches++
	}

	message, messagePresent, messageErr := optionalNonEmptyString(args, "message")
	if messageErr != nil {
		return messageErr
	}
	if messagePresent {
		job.Payload.Message = message
		patches++
	}

	schedule, hasSchedule, errResult := schedulePatch(args)
	if errResult != nil {
		return errResult
	}
	if hasSchedule {
		job.Schedule = schedule
		job.DeleteAfterRun = schedule.Kind == "at"
		patches++
	}

	command, commandPresent, errResult := optionalString(args, "command")
	if errResult != nil {
		return errResult
	}
	if commandPresent {
		if errResult := t.validateCommandMutation(ctx, args); errResult != nil {
			return errResult
		}
		job.Payload.Command = command
		patches++
	}

	if patches == 0 {
		return ErrorResult("at least one update field is required")
	}

	if err := t.cronService.UpdateJob(job); err != nil {
		return ErrorResult(fmt.Sprintf("Error updating job: %v", err))
	}

	updated, _ := t.cronService.GetJob(jobID)
	return SilentResult(fmt.Sprintf("Cron job updated:\n%s", formatCronJobJSON(updated)))
}

func (t *CronTool) removeJob(args map[string]any) *ToolResult {
	jobID, ok := args["job_id"].(string)
	if !ok || jobID == "" {
		return ErrorResult("job_id is required for remove")
	}

	if t.cronService.RemoveJob(jobID) {
		return SilentResult(fmt.Sprintf("Cron job removed: %s", jobID))
	}
	return ErrorResult(fmt.Sprintf("Job %s not found", jobID))
}

func requiredCronJobID(args map[string]any, action string) (string, *ToolResult) {
	jobID, ok := args["job_id"].(string)
	if !ok || jobID == "" {
		return "", ErrorResult(fmt.Sprintf("job_id is required for %s", action))
	}
	return jobID, nil
}

func optionalNonEmptyString(args map[string]any, key string) (string, bool, *ToolResult) {
	_, present := args[key]
	if !present {
		return "", false, nil
	}
	text, _, errResult := optionalString(args, key)
	if errResult != nil {
		return "", false, errResult
	}
	if strings.TrimSpace(text) == "" {
		return "", false, ErrorResult(fmt.Sprintf("%s cannot be empty", key))
	}
	return text, true, nil
}

func optionalString(args map[string]any, key string) (string, bool, *ToolResult) {
	value, present := args[key]
	if !present {
		return "", false, nil
	}
	text, ok := value.(string)
	if !ok {
		return "", false, ErrorResult(fmt.Sprintf("%s must be a string", key))
	}
	return text, true, nil
}

func schedulePatch(args map[string]any) (cron.CronSchedule, bool, *ToolResult) {
	var schedule cron.CronSchedule
	patches := 0

	if _, present := args["at_seconds"]; present {
		seconds, errResult := positiveSeconds(args, "at_seconds")
		if errResult != nil {
			return cron.CronSchedule{}, false, errResult
		}
		atMS := time.Now().UnixMilli() + seconds*1000
		schedule = cron.CronSchedule{Kind: "at", AtMS: &atMS}
		patches++
	}

	if _, present := args["every_seconds"]; present {
		seconds, errResult := positiveSeconds(args, "every_seconds")
		if errResult != nil {
			return cron.CronSchedule{}, false, errResult
		}
		everyMS := seconds * 1000
		schedule = cron.CronSchedule{Kind: "every", EveryMS: &everyMS}
		patches++
	}

	if _, present := args["cron_expr"]; present {
		cronExpr, ok := args["cron_expr"].(string)
		if !ok {
			return cron.CronSchedule{}, false, ErrorResult("cron_expr must be a string")
		}
		if strings.TrimSpace(cronExpr) == "" {
			return cron.CronSchedule{}, false, ErrorResult("cron_expr cannot be empty")
		}
		schedule = cron.CronSchedule{Kind: "cron", Expr: cronExpr}
		patches++
	}

	if patches > 1 {
		return cron.CronSchedule{}, false, ErrorResult("only one of at_seconds, every_seconds, or cron_expr can be set")
	}
	return schedule, patches == 1, nil
}

func positiveSeconds(args map[string]any, key string) (int64, *ToolResult) {
	value := args[key]
	var seconds int64
	switch v := value.(type) {
	case float64:
		if v != float64(int64(v)) {
			return 0, ErrorResult(fmt.Sprintf("%s must be a positive integer", key))
		}
		seconds = int64(v)
	case int:
		seconds = int64(v)
	case int64:
		seconds = v
	default:
		return 0, ErrorResult(fmt.Sprintf("%s must be a positive integer", key))
	}
	if seconds <= 0 {
		return 0, ErrorResult(fmt.Sprintf("%s must be a positive integer", key))
	}
	return seconds, nil
}

func (t *CronTool) validateCommandMutation(ctx context.Context, args map[string]any) *ToolResult {
	if !t.execEnabled {
		return ErrorResult("command execution is disabled")
	}
	if !constants.IsInternalChannel(ToolChannel(ctx)) {
		return ErrorResult("updating command execution is restricted to internal channels")
	}
	commandConfirm, _ := args["command_confirm"].(bool)
	if !t.allowCommand && !commandConfirm {
		return ErrorResult("command_confirm=true is required when allow_command is disabled")
	}
	return nil
}

func (t *CronTool) canAccessJob(ctx context.Context, job *cron.CronJob) bool {
	channel := ToolChannel(ctx)
	if constants.IsInternalChannel(channel) {
		return true
	}
	chatID := ToolChatID(ctx)
	if channel == "" || chatID == "" {
		return false
	}
	if job.Payload.Command != "" {
		return false
	}
	return job.Payload.Channel == channel && job.Payload.To == chatID
}

func formatCronJobJSON(job *cron.CronJob) string {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Sprintf("%+v", *job)
	}
	return string(data)
}

func (t *CronTool) enableJob(args map[string]any, enable bool) *ToolResult {
	jobID, ok := args["job_id"].(string)
	if !ok || jobID == "" {
		return ErrorResult("job_id is required for enable/disable")
	}

	job := t.cronService.EnableJob(jobID, enable)
	if job == nil {
		return ErrorResult(fmt.Sprintf("Job %s not found", jobID))
	}

	status := "enabled"
	if !enable {
		status = "disabled"
	}
	return SilentResult(fmt.Sprintf("Cron job '%s' %s", job.Name, status))
}

// ExecuteJob executes a cron job through the agent
func (t *CronTool) ExecuteJob(ctx context.Context, job *cron.CronJob) string {
	// Get channel/chatID from job payload
	channel := job.Payload.Channel
	chatID := job.Payload.To

	// Default values if not set
	if channel == "" {
		channel = "cli"
	}
	if chatID == "" {
		chatID = "direct"
	}
	taskID := t.startCronTaskRecord(job, channel, chatID)

	// Execute command if present
	if job.Payload.Command != "" {
		if !t.execEnabled || t.execTool == nil {
			output := "Error executing scheduled command: command execution is disabled"
			pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer pubCancel()
			err := t.msgBus.PublishOutbound(pubCtx, bus.OutboundMessage{
				Context: bus.NewOutboundContext(channel, chatID, ""),
				Content: output,
			})
			t.finishCronTaskRecord(taskID, taskregistry.StatusFailed, cronDeliveryStatusForPublish(err), output, err)
			return "ok"
		}

		args := map[string]any{
			"action":    "run",
			"command":   job.Payload.Command,
			"__channel": channel,
			"__chat_id": chatID,
		}

		result := t.execTool.Execute(ctx, args)
		var output string
		if result.IsError {
			output = fmt.Sprintf("Error executing scheduled command: %s", result.ForLLM)
		} else {
			output = fmt.Sprintf("Scheduled command '%s' executed:\n%s", job.Payload.Command, result.ForLLM)
		}

		pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer pubCancel()
		err := t.msgBus.PublishOutbound(pubCtx, bus.OutboundMessage{
			Context: bus.NewOutboundContext(channel, chatID, ""),
			Content: output,
		})
		if result.IsError {
			t.finishCronTaskRecord(taskID, taskregistry.StatusFailed, cronDeliveryStatusForPublish(err), output, err)
		} else {
			t.finishCronTaskRecord(taskID, taskregistry.StatusSucceeded, cronDeliveryStatusForPublish(err), output, err)
		}
		return "ok"
	}

	sessionKey := fmt.Sprintf("agent:cron-%s-%s", job.ID, uuid.New().String())

	// Call agent with the job message. Scheduled agent turns should not emit
	// interactive progress/tool-feedback messages; they should only publish a
	// final response when the job has something actionable to say.
	var response string
	var err error
	if scheduledExecutor, ok := t.executor.(scheduledJobExecutor); ok {
		response, err = scheduledExecutor.ProcessScheduledWithChannel(
			ctx,
			job.Payload.Message,
			sessionKey,
			channel,
			chatID,
		)
	} else {
		response, err = t.executor.ProcessDirectWithChannel(
			ctx,
			job.Payload.Message,
			sessionKey,
			channel,
			chatID,
		)
	}
	if err != nil {
		t.finishCronTaskRecord(taskID, taskregistry.StatusFailed, taskregistry.DeliveryFailed, "", err)
		return fmt.Sprintf("Error: %v", err)
	}

	if response != "" {
		trimmed := strings.TrimSpace(response)
		if strings.EqualFold(trimmed, "NO_REPLY") || strings.EqualFold(trimmed, "HEARTBEAT_OK") {
			t.finishCronTaskRecord(taskID, taskregistry.StatusSucceeded, taskregistry.DeliveryNotApplicable, trimmed, nil)
			return "ok"
		}
		t.executor.PublishResponseIfNeeded(ctx, channel, chatID, sessionKey, response)
		t.finishCronTaskRecord(taskID, taskregistry.StatusSucceeded, taskregistry.DeliveryDelivered, response, nil)
		return "ok"
	}
	t.finishCronTaskRecord(taskID, taskregistry.StatusSucceeded, taskregistry.DeliveryNotApplicable, "", nil)
	return "ok"
}

func (t *CronTool) startCronTaskRecord(job *cron.CronJob, channel, chatID string) string {
	if t == nil || t.taskRegistry == nil || job == nil {
		return ""
	}
	runID := uuid.New().String()
	jobID := strings.TrimSpace(job.ID)
	if jobID == "" {
		jobID = "unknown"
	}
	taskID := fmt.Sprintf("cron-%s-%s", jobID, runID)
	taskText := job.Payload.Message
	if strings.TrimSpace(job.Payload.Command) != "" {
		taskText = job.Payload.Command
	}
	_ = t.taskRegistry.Upsert(taskregistry.Record{
		TaskID:         taskID,
		Runtime:        taskregistry.RuntimeCron,
		TaskKind:       "cron",
		Channel:        channel,
		ChatID:         chatID,
		Label:          job.Name,
		Task:           taskText,
		Status:         taskregistry.StatusRunning,
		DeliveryStatus: taskregistry.DeliveryPending,
		NotifyPolicy:   taskregistry.NotifyDoneOnly,
		DeliveryMode:   "deliver",
	})
	return taskID
}

func (t *CronTool) finishCronTaskRecord(
	taskID string,
	status taskregistry.Status,
	delivery taskregistry.DeliveryStatus,
	summary string,
	err error,
) {
	if t == nil || t.taskRegistry == nil || strings.TrimSpace(taskID) == "" {
		return
	}
	now := time.Now().UnixMilli()
	_ = t.taskRegistry.Update(taskID, func(rec *taskregistry.Record) {
		rec.Status = status
		rec.DeliveryStatus = delivery
		rec.EndedAt = now
		rec.LastEventAt = now
		rec.TerminalSummary = strings.TrimSpace(summary)
		if err != nil {
			rec.Error = err.Error()
			rec.DeliveryError = err.Error()
		}
		if delivery == taskregistry.DeliveryDelivered || delivery == taskregistry.DeliveryNotApplicable {
			rec.DeliveredAt = now
			rec.DeliveryError = ""
		}
	})
}

func cronDeliveryStatusForPublish(err error) taskregistry.DeliveryStatus {
	if err != nil {
		return taskregistry.DeliveryFailed
	}
	return taskregistry.DeliveryDelivered
}
