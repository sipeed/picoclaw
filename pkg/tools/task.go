// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package tools

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// TaskTool provides task/schedule management capabilities
type TaskTool struct {
	taskStore *TaskStore
}

// NewTaskTool creates a new TaskTool
func NewTaskTool(taskStore *TaskStore) *TaskTool {
	return &TaskTool{
		taskStore: taskStore,
	}
}

// Name returns the tool name
func (t *TaskTool) Name() string {
	return "manage_tasks"
}

// Description returns the tool description
func (t *TaskTool) Description() string {
	return `Manage tasks and schedules. Use this when:
- User wants to add a task or schedule
- User asks about today's or upcoming tasks
- User wants to mark a task as complete
- User wants to delete a task

Actions:
- add: Add a new task with optional due date, time, and repeat pattern
- list: List tasks (all, today, upcoming, completed)
- complete: Mark a task as completed
- delete: Delete a task
- get_today: Get today's tasks (shorthand)

Date formats: YYYY-MM-DD (e.g., 2026-02-03)
Weekdays: 1=Monday, 2=Tuesday, ..., 5=Friday
Repeat: none, daily, weekly, monthly, weekdays (Mon-Fri)`
}

// Parameters returns the tool parameters schema
func (t *TaskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"add", "list", "complete", "delete", "get_today"},
				"description": "Action to perform",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Task title (required for add)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Task description/details (optional for add)",
			},
			"due_date": map[string]any{
				"type":        "string",
				"description": "Due date in YYYY-MM-DD format (optional for add)",
			},
			"due_time": map[string]any{
				"type":        "string",
				"description": "Due time in HH:MM format (optional for add)",
			},
			"due_weekday": map[string]any{
				"type":        "number",
				"description": "Day of week: 1=Monday, 2=Tuesday, ..., 7=Sunday (optional for add)",
			},
			"repeat": map[string]any{
				"type":        "string",
				"enum":        []string{"none", "daily", "weekly", "monthly", "weekdays"},
				"description": "Repeat pattern: none, daily, weekly, monthly, weekdays (Mon-Fri)",
			},
			"remind_before": map[string]any{
				"type":        "number",
				"description": "Remind N minutes before (optional for add)",
			},
			"task_id": map[string]any{
				"type":        "string",
				"description": "Task ID (required for complete/delete)",
			},
			"scope": map[string]any{
				"type":        "string",
				"enum":        []string{"all", "today", "upcoming", "completed"},
				"description": "Scope for list action: all, today, upcoming, completed",
			},
		},
		"required": []string{"action"},
	}
}

// Execute runs the tool with the given arguments
func (t *TaskTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}

	switch action {
	case "add":
		return t.addTask(args)
	case "list":
		return t.listTasks(args)
	case "complete":
		return t.completeTask(args)
	case "delete":
		return t.deleteTask(args)
	case "get_today":
		return t.getTodayTasks(args)
	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

// addTask adds a new task
func (t *TaskTool) addTask(args map[string]any) *ToolResult {
	title, ok := args["title"].(string)
	if !ok || strings.TrimSpace(title) == "" {
		return ErrorResult("title is required for add action")
	}

	description, _ := args["description"].(string)
	dueDate, _ := args["due_date"].(string)
	dueTime, _ := args["due_time"].(string)
	dueWeekday, _ := args["due_weekday"].(float64)
	repeat, _ := args["repeat"].(string)
	remindBefore, _ := args["remind_before"].(float64)

	// Validate repeat
	if repeat == "" {
		repeat = "none"
	}
	validRepeats := map[string]bool{"none": true, "daily": true, "weekly": true, "monthly": true, "weekdays": true}
	if !validRepeats[repeat] {
		return ErrorResult(fmt.Sprintf("invalid repeat value: %s (use: none, daily, weekly, monthly, weekdays)", repeat))
	}

	// Validate due_weekday if provided
	var weekdayInt int
	if dueWeekday > 0 {
		weekdayInt = int(dueWeekday)
		if weekdayInt < 1 || weekdayInt > 7 {
			return ErrorResult("due_weekday must be between 1 (Monday) and 7 (Sunday)")
		}
	}

	task := Task{
		Title:       title,
		Description: description,
		DueDate:     dueDate,
		DueTime:     dueTime,
		DueWeekday:  weekdayInt,
		Repeat:      repeat,
		RemindBefore: int(remindBefore),
		Completed:   false,
	}

	tasks, err := t.taskStore.AddTask(task)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to add task: %v", err))
	}

	// Build success message
	var scheduleInfo []string
	if dueDate != "" {
		scheduleInfo = append(scheduleInfo, fmt.Sprintf("日期: %s", dueDate))
	}
	if dueTime != "" {
		scheduleInfo = append(scheduleInfo, fmt.Sprintf("时间: %s", dueTime))
	}
	if dueWeekday > 0 {
		scheduleInfo = append(scheduleInfo, fmt.Sprintf("星期: %s", weekdayName(int(dueWeekday))))
	}
	if repeat != "none" {
		scheduleInfo = append(scheduleInfo, fmt.Sprintf("重复: %s", repeat))
	}

	scheduleStr := ""
	if len(scheduleInfo) > 0 {
		scheduleStr = " (" + strings.Join(scheduleInfo, ", ") + ")"
	}

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Task added successfully: %s (ID: %s)%s", title, tasks[len(tasks)-1].ID, scheduleStr),
		ForUser: fmt.Sprintf("✅ 已添加任务: %s%s", title, scheduleStr),
		Silent:  false,
		IsError: false,
		Async:   false,
	}
}

// listTasks lists tasks based on scope
func (t *TaskTool) listTasks(args map[string]any) *ToolResult {
	scope, _ := args["scope"].(string)
	if scope == "" {
		scope = "all"
	}

	var tasks []Task
	var err error

	switch scope {
	case "all":
		tasks, err = t.taskStore.GetAllTasks()
	case "today":
		tasks, err = t.taskStore.GetTasksForDate(time.Now())
	case "upcoming":
		tasks, err = t.getUpcomingTasks(7)
	case "completed":
		tasks, err = t.taskStore.GetAllTasks()
	default:
		return ErrorResult(fmt.Sprintf("invalid scope: %s (use: all, today, upcoming, completed)", scope))
	}

	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to list tasks: %v", err))
	}

	// Filter for completed if needed
	if scope == "completed" {
		var completed []Task
		for _, task := range tasks {
			if task.Completed {
				completed = append(completed, task)
			}
		}
		tasks = completed
	}

	if len(tasks) == 0 {
		var msg string
		switch scope {
		case "all":
			msg = "暂无任务"
		case "today":
			msg = "今天没有任务"
		case "upcoming":
			msg = "未来 7 天没有任务"
		case "completed":
			msg = "没有已完成的任务"
		}
		return &ToolResult{
			ForLLM:  msg,
			ForUser: msg,
			Silent:  false,
			IsError: false,
			Async:   false,
		}
	}

	// Format output
	var sb strings.Builder

	switch scope {
	case "all":
		sb.WriteString("📋 所有任务:\n\n")
	case "today":
		sb.WriteString("📅 今日任务:\n\n")
	case "upcoming":
		sb.WriteString("📆 近期任务 (未来 7 天):\n\n")
	case "completed":
		sb.WriteString("✅ 已完成任务:\n\n")
	}

	for _, task := range tasks {
		sb.WriteString(t.formatTask(&task))
		sb.WriteString("\n")
	}

	return &ToolResult{
		ForLLM:  sb.String(),
		ForUser: sb.String(),
		Silent:  false,
		IsError: false,
		Async:   false,
	}
}

// getTodayTasks gets today's tasks (shorthand)
func (t *TaskTool) getTodayTasks(args map[string]any) *ToolResult {
	return t.listTasks(map[string]any{"scope": "today"})
}

// completeTask marks a task as completed
func (t *TaskTool) completeTask(args map[string]any) *ToolResult {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return ErrorResult("task_id is required for complete action")
	}

	tasks, err := t.taskStore.CompleteTask(taskID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to complete task: %v", err))
	}

	// Find the task to show which one was completed
	for _, task := range tasks {
		if task.ID == taskID {
			return &ToolResult{
				ForLLM:  fmt.Sprintf("Task completed: %s", task.Title),
				ForUser: fmt.Sprintf("✅ 已完成任务: %s", task.Title),
				Silent:  false,
				IsError: false,
				Async:   false,
			}
		}
	}

	return ErrorResult("Task not found after completion")
}

// deleteTask deletes a task
func (t *TaskTool) deleteTask(args map[string]any) *ToolResult {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return ErrorResult("task_id is required for delete action")
	}

	// Get task before deleting for message
	task, err := t.taskStore.GetTaskByID(taskID)
	if err != nil || task == nil {
		return ErrorResult(fmt.Sprintf("Task not found: %s", taskID))
	}

	taskTitle := task.Title

	_, err = t.taskStore.DeleteTask(taskID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to delete task: %v", err))
	}

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Task deleted: %s", taskTitle),
		ForUser: fmt.Sprintf("🗑️ 已删除任务: %s", taskTitle),
		Silent:  false,
		IsError: false,
		Async:   false,
	}
}

// getUpcomingTasks gets tasks for the next N days
func (t *TaskTool) getUpcomingTasks(days int) ([]Task, error) {
	var allTasks []Task

	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, i)
		tasks, err := t.taskStore.GetTasksForDate(date)
		if err != nil {
			return nil, err
		}
		allTasks = append(allTasks, tasks...)
	}

	return allTasks, nil
}

// formatTask formats a task for display
func (t *TaskTool) formatTask(task *Task) string {
	var sb strings.Builder

	// Checkbox
	if task.Completed {
		sb.WriteString("☑️ ")
	} else {
		sb.WriteString("⬜ ")
	}

	// Title
	sb.WriteString(task.Title)

	// Show ID for reference
	sb.WriteString(fmt.Sprintf(" [ID:%s]", task.ID))

	// Description
	if task.Description != "" {
		sb.WriteString(fmt.Sprintf("\n   📝 %s", task.Description))
	}

	// Schedule info
	var schedule []string
	if task.DueDate != "" {
		schedule = append(schedule, task.DueDate)
	}
	if task.DueTime != "" {
		schedule = append(schedule, task.DueTime)
	}
	if task.DueWeekday > 0 {
		schedule = append(schedule, weekdayName(task.DueWeekday))
	}
	if task.Repeat != "none" {
		repeatNames := map[string]string{
			"daily":    "每天",
			"weekly":   "每周",
			"monthly":  "每月",
			"weekdays": "工作日",
		}
		schedule = append(schedule, repeatNames[task.Repeat])
	}

	if len(schedule) > 0 {
		sb.WriteString(fmt.Sprintf("\n   📅 %s", strings.Join(schedule, " ")))
	}

	// Completed time
	if task.Completed && task.CompletedAt != "" {
		sb.WriteString(fmt.Sprintf("\n   ✅ 已完成于 %s", task.CompletedAt))
	}

	return sb.String()
}
