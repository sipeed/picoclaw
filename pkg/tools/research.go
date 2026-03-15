package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sipeed/picoclaw/pkg/research"
)

// ResearchTool provides research task management for the agent.
type ResearchTool struct {
	store     *research.ResearchStore
	workspace string
}

// NewResearchTool creates a new ResearchTool.
func NewResearchTool(store *research.ResearchStore, workspace string) *ResearchTool {
	return &ResearchTool{store: store, workspace: workspace}
}

func (t *ResearchTool) Name() string { return "research" }

func (t *ResearchTool) Description() string {
	return "Manage research tasks and findings. Use list_tasks to discover pending research, set_status to update task state, add_finding to record research results as markdown documents, and get_task to view task details."
}

func (t *ResearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"list_tasks", "get_task", "set_status", "add_finding"},
				"description": "Action to perform.",
			},
			"task_id": map[string]any{
				"type":        "string",
				"description": "Task ID (required for get_task, set_status, add_finding).",
			},
			"status_filter": map[string]any{
				"type":        "string",
				"enum":        []string{"pending", "active", "completed", "failed", "canceled"},
				"description": "Filter tasks by status (for list_tasks). Omit to list all.",
			},
			"status": map[string]any{
				"type":        "string",
				"enum":        []string{"pending", "active", "completed", "failed", "canceled"},
				"description": "New status (for set_status).",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Finding title (for add_finding).",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Markdown content to write (for add_finding).",
			},
			"summary": map[string]any{
				"type":        "string",
				"description": "Brief summary of the finding (for add_finding).",
			},
		},
		"required": []string{"action"},
	}
}

func (t *ResearchTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, _ := args["action"].(string)
	switch action {
	case "list_tasks":
		return t.listTasks(args)
	case "get_task":
		return t.getTask(args)
	case "set_status":
		return t.setStatus(args)
	case "add_finding":
		return t.addFinding(args)
	default:
		return ErrorResult("unknown action: " + action)
	}
}

func (t *ResearchTool) listTasks(args map[string]any) *ToolResult {
	filter, _ := args["status_filter"].(string)
	tasks, err := t.store.ListTasks(research.TaskStatus(filter))
	if err != nil {
		return ErrorResult(fmt.Sprintf("list tasks: %v", err))
	}
	if len(tasks) == 0 {
		return NewToolResult("No research tasks found.")
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d research task(s):\n\n", len(tasks)))
	for _, task := range tasks {
		docCount, _ := t.store.DocumentCount(task.ID)
		b.WriteString(fmt.Sprintf("- **%s** [%s] (id: %s, docs: %d)\n  %s\n",
			task.Title, task.Status, task.ID, docCount, task.Description))
	}
	return NewToolResult(b.String())
}

func (t *ResearchTool) getTask(args map[string]any) *ToolResult {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return ErrorResult("task_id is required")
	}

	task, err := t.store.GetTask(taskID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("get task: %v", err))
	}

	docs, _ := t.store.ListDocuments(taskID)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("## %s\n", task.Title))
	b.WriteString(fmt.Sprintf("- **Status**: %s\n", task.Status))
	b.WriteString(fmt.Sprintf("- **ID**: %s\n", task.ID))
	b.WriteString(fmt.Sprintf("- **Output dir**: %s\n", task.OutputDir))
	if task.Description != "" {
		b.WriteString(fmt.Sprintf("- **Description**: %s\n", task.Description))
	}
	b.WriteString(fmt.Sprintf("- **Created**: %s\n", task.CreatedAt.Format("2006-01-02 15:04")))

	if len(docs) > 0 {
		b.WriteString(fmt.Sprintf("\n### Documents (%d)\n", len(docs)))
		for _, d := range docs {
			b.WriteString(fmt.Sprintf("- [%d] %s (%s) — %s\n  path: %s\n",
				d.Seq, d.Title, d.DocType, d.Summary, d.FilePath))
		}
	} else {
		b.WriteString("\nNo documents yet.\n")
	}

	return NewToolResult(b.String())
}

func (t *ResearchTool) setStatus(args map[string]any) *ToolResult {
	taskID, _ := args["task_id"].(string)
	status, _ := args["status"].(string)
	if taskID == "" || status == "" {
		return ErrorResult("task_id and status are required")
	}

	if err := t.store.SetTaskStatus(taskID, research.TaskStatus(status)); err != nil {
		return ErrorResult(fmt.Sprintf("set status: %v", err))
	}
	return NewToolResult(fmt.Sprintf("Task status updated to %s.", status))
}

var sanitizeRe = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func (t *ResearchTool) addFinding(args map[string]any) *ToolResult {
	taskID, _ := args["task_id"].(string)
	title, _ := args["title"].(string)
	content, _ := args["content"].(string)
	summary, _ := args["summary"].(string)
	if taskID == "" || title == "" || content == "" {
		return ErrorResult("task_id, title, and content are required")
	}

	task, err := t.store.GetTask(taskID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("get task: %v", err))
	}

	// Determine next seq
	docs, _ := t.store.ListDocuments(taskID)
	nextSeq := len(docs) + 1

	// Build filename
	sanitized := sanitizeRe.ReplaceAllString(strings.ToLower(title), "-")
	sanitized = strings.Trim(sanitized, "-")
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}
	filename := fmt.Sprintf("%03d-%s.md", nextSeq, sanitized)
	relPath := filepath.Join(task.OutputDir, filename)
	absPath := filepath.Join(t.workspace, relPath)

	// Ensure parent dir exists
	if mkErr := os.MkdirAll(filepath.Dir(absPath), 0o755); mkErr != nil {
		return ErrorResult(fmt.Sprintf("create dir: %v", mkErr))
	}

	// Write markdown
	if wErr := os.WriteFile(absPath, []byte(content), 0o644); wErr != nil {
		return ErrorResult(fmt.Sprintf("write file: %v", wErr))
	}

	// Record in DB
	doc, err := t.store.AddDocument(taskID, title, relPath, "finding", summary)
	if err != nil {
		return ErrorResult(fmt.Sprintf("add document: %v", err))
	}

	return NewToolResult(fmt.Sprintf(
		"Finding recorded:\n- File: %s\n- Document ID: %s\n- Seq: %d",
		relPath, doc.ID, doc.Seq,
	))
}
