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
// It implements StatusProvider to inject a lightweight research catalog
// and focus state into the system prompt.
type ResearchTool struct {
	store     *research.ResearchStore
	workspace string
	focus     *research.FocusTracker
}

// NewResearchTool creates a new ResearchTool.
func NewResearchTool(store *research.ResearchStore, workspace string, focus *research.FocusTracker) *ResearchTool {
	return &ResearchTool{
		store:     store,
		workspace: workspace,
		focus:     focus,
	}
}

func (t *ResearchTool) Name() string { return "research" }

func (t *ResearchTool) Description() string {
	return "Manage research tasks and findings. Actions: list_tasks, get_task, set_status, set_interval, add_finding, recall (load findings into context), forget (release focus when changing topics)."
}

func (t *ResearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": []string{
					"list_tasks",
					"get_task",
					"set_status",
					"set_interval",
					"add_finding",
					"recall",
					"forget",
				},
				"description": "Action to perform.",
			},
			"interval": map[string]any{
				"type":        "string",
				"description": "Research interval (for set_interval). Examples: '6h', '1d', '7d'.",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Search query to find tasks by title/slug (for recall/forget). Alternative to task_id.",
			},
			"task_id": map[string]any{
				"type":        "string",
				"description": "Task ID (required for get_task, set_status, add_finding; optional for recall/forget).",
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
		return t.setStatus(ctx, args)
	case "set_interval":
		return t.setInterval(args)
	case "add_finding":
		return t.addFinding(args)
	case "recall":
		return t.recall(args)
	case "forget":
		return t.forget(args)
	// Keep backward compat for load_context
	case "load_context":
		return t.recall(args)
	default:
		return ErrorResult("unknown action: " + action)
	}
}

// RuntimeStatus implements StatusProvider. It injects a lightweight research
// catalog (titles only) and current focus state into the system prompt.
func (t *ResearchTool) RuntimeStatus() string {
	tasks, err := t.store.ListTasks("")
	if err != nil || len(tasks) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("# Research Knowledge Base\n\n")
	b.WriteString("Available research topics:\n")
	for _, task := range tasks {
		docCount, _ := t.store.DocumentCount(task.ID)
		b.WriteString(fmt.Sprintf("- \"%s\" [%s, %d docs] (id: %s)\n",
			task.Title, task.Status, docCount, task.ID))
	}

	if focusID, focusTitle := t.focus.Current(); focusID != "" {
		b.WriteString(fmt.Sprintf("\nCurrently focused: \"%s\" (id: %s)\n", focusTitle, focusID))
	}

	b.WriteString("\nUse `research recall` to load findings when the user asks about a research topic.\n")
	b.WriteString("Use `research forget` to release focus when changing topics.\n")

	return b.String()
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

func (t *ResearchTool) setStatus(ctx context.Context, args map[string]any) *ToolResult {
	taskID, _ := args["task_id"].(string)
	status, _ := args["status"].(string)
	if taskID == "" || status == "" {
		return ErrorResult("task_id and status are required")
	}

	// Guard: during heartbeat, prevent completing tasks with too few findings
	if IsHeartbeatContext(ctx) && research.TaskStatus(status) == research.StatusCompleted {
		docCount, err := t.store.DocumentCount(taskID)
		if err != nil {
			return ErrorResult(fmt.Sprintf("check document count: %v", err))
		}
		if docCount < research.MinFindingsForCompletion {
			return ErrorResult(fmt.Sprintf(
				"cannot mark as completed during heartbeat: task has %d findings (minimum %d required). Add more findings first.",
				docCount,
				research.MinFindingsForCompletion,
			))
		}
	}

	if err := t.store.SetTaskStatus(taskID, research.TaskStatus(status)); err != nil {
		return ErrorResult(fmt.Sprintf("set status: %v", err))
	}
	return NewToolResult(fmt.Sprintf("Task status updated to %s.", status))
}

func (t *ResearchTool) setInterval(args map[string]any) *ToolResult {
	taskID, _ := args["task_id"].(string)
	interval, _ := args["interval"].(string)
	if taskID == "" || interval == "" {
		return ErrorResult("task_id and interval are required")
	}
	if err := t.store.SetInterval(taskID, interval); err != nil {
		return ErrorResult(fmt.Sprintf("set interval: %v", err))
	}
	return NewToolResult(fmt.Sprintf("Research interval updated to %s.", interval))
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

	// Touch last_researched_at for interval tracking
	_ = t.store.TouchLastResearched(taskID)

	return NewToolResult(fmt.Sprintf(
		"Finding recorded:\n- File: %s\n- Document ID: %s\n- Seq: %d",
		relPath, doc.ID, doc.Seq,
	))
}

// maxContextBytes caps the total size of loaded research context.
const maxContextBytes = 100 * 1024 // 100KB

// recall loads all findings for a task and marks it as focused in the system prompt.
func (t *ResearchTool) recall(args map[string]any) *ToolResult {
	task, err := t.resolveTask(args)
	if err != nil {
		return ErrorResult(err.Error())
	}

	docs, err := t.store.ListDocuments(task.ID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("list documents: %v", err))
	}

	// Mark as focused
	t.focus.Focus(task.ID, task.Title)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Research Context: %s\n\n", task.Title))
	b.WriteString(fmt.Sprintf("**Status**: %s | **Documents**: %d\n", task.Status, len(docs)))
	if task.Description != "" {
		b.WriteString(fmt.Sprintf("**Description**: %s\n", task.Description))
	}
	b.WriteString("\n---\n\n")

	if len(docs) == 0 {
		b.WriteString("No findings recorded yet.\n")
		return NewToolResult(b.String())
	}

	totalBytes := b.Len()
	loaded := 0
	for _, d := range docs {
		absPath := filepath.Join(t.workspace, d.FilePath)
		data, readErr := os.ReadFile(absPath)
		if readErr != nil {
			b.WriteString(fmt.Sprintf("## [%d] %s\n\n(file not found: %s)\n\n", d.Seq, d.Title, d.FilePath))
			loaded++
			continue
		}

		content := string(data)
		entrySize := len(d.Title) + len(content) + 40 // rough overhead for headers
		if totalBytes+entrySize > maxContextBytes {
			remaining := len(docs) - loaded
			b.WriteString(
				fmt.Sprintf(
					"\n---\n*Context limit reached. %d more finding(s) not shown. Use get_task to see the full list.*\n",
					remaining,
				),
			)
			break
		}

		b.WriteString(fmt.Sprintf("## [%d] %s\n\n", d.Seq, d.Title))
		b.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			b.WriteByte('\n')
		}
		b.WriteString("\n---\n\n")
		totalBytes += entrySize
		loaded++
	}

	return NewToolResult(b.String())
}

// forget removes a task from the focus set.
func (t *ResearchTool) forget(args map[string]any) *ToolResult {
	taskID, _ := args["task_id"].(string)
	query, _ := args["query"].(string)

	// If no args, forget all
	if taskID == "" && query == "" {
		count := t.focus.UnfocusAll()
		if count == 0 {
			return NewToolResult("No research topics were focused.")
		}
		return NewToolResult(fmt.Sprintf("Released focus on %d research topic(s).", count))
	}

	task, err := t.resolveTask(args)
	if err != nil {
		return ErrorResult(err.Error())
	}

	if !t.focus.Unfocus(task.ID) {
		return NewToolResult(fmt.Sprintf("Research topic \"%s\" was not focused.", task.Title))
	}
	return NewToolResult(
		fmt.Sprintf("Released focus on \"%s\". Research findings are no longer in active context.", task.Title),
	)
}

// resolveTask finds a task by task_id or query.
func (t *ResearchTool) resolveTask(args map[string]any) (*research.Task, error) {
	taskID, _ := args["task_id"].(string)
	query, _ := args["query"].(string)

	if taskID == "" && query == "" {
		return nil, fmt.Errorf("task_id or query is required")
	}

	if taskID != "" {
		task, err := t.store.GetTask(taskID)
		if err != nil {
			return nil, fmt.Errorf("get task: %w", err)
		}
		return task, nil
	}

	tasks, err := t.store.SearchTasks(query)
	if err != nil {
		return nil, fmt.Errorf("search tasks: %w", err)
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no research task found matching %q", query)
	}
	return tasks[0], nil
}
