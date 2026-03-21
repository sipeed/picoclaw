// Piconomous - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 Piconomous contributors

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sipeed/piconomous/pkg/fileutil"
)

// GoalStatus represents the lifecycle state of a goal.
type GoalStatus string

const (
	GoalStatusActive    GoalStatus = "active"
	GoalStatusCompleted GoalStatus = "completed"
	GoalStatusDropped   GoalStatus = "dropped"
)

// Goal represents a single autonomous goal the agent is pursuing.
type Goal struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Priority    int        `json:"priority"` // 1=highest, 5=lowest
	Status      GoalStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// GoalStore persists goals as a JSON file alongside a human-readable GOALS.md.
type GoalStore struct {
	workspace string
	maxGoals  int
}

func newGoalStore(workspace string, maxGoals int) *GoalStore {
	return &GoalStore{workspace: workspace, maxGoals: maxGoals}
}

func (gs *GoalStore) goalsJSONPath() string {
	return filepath.Join(gs.workspace, "goals.json")
}

func (gs *GoalStore) goalsMDPath() string {
	return filepath.Join(gs.workspace, "GOALS.md")
}

func (gs *GoalStore) load() ([]Goal, error) {
	data, err := os.ReadFile(gs.goalsJSONPath())
	if os.IsNotExist(err) {
		return []Goal{}, nil
	}
	if err != nil {
		return nil, err
	}
	var goals []Goal
	if err := json.Unmarshal(data, &goals); err != nil {
		return nil, err
	}
	return goals, nil
}

func (gs *GoalStore) save(goals []Goal) error {
	data, err := json.MarshalIndent(goals, "", "  ")
	if err != nil {
		return err
	}
	if err := fileutil.WriteFileAtomic(gs.goalsJSONPath(), data, 0o644); err != nil {
		return err
	}
	return gs.writeMD(goals)
}

func (gs *GoalStore) writeMD(goals []Goal) error {
	var sb strings.Builder
	sb.WriteString("# Goals\n\n")
	sb.WriteString("*Managed by the autonomous goal system. Edit goals.json directly or use the `goal` tool.*\n\n")

	active := filterGoals(goals, GoalStatusActive)
	completed := filterGoals(goals, GoalStatusCompleted)
	dropped := filterGoals(goals, GoalStatusDropped)

	if len(active) == 0 {
		sb.WriteString("## Active Goals\n\n*No active goals.*\n\n")
	} else {
		sb.WriteString("## Active Goals\n\n")
		for _, g := range active {
			sb.WriteString(fmt.Sprintf("### [P%d] %s (id: %s)\n", g.Priority, g.Title, g.ID))
			if g.Description != "" {
				sb.WriteString(g.Description + "\n")
			}
			sb.WriteString(fmt.Sprintf("*Created: %s | Updated: %s*\n\n",
				g.CreatedAt.Format("2006-01-02 15:04"),
				g.UpdatedAt.Format("2006-01-02 15:04"),
			))
		}
	}

	if len(completed) > 0 {
		sb.WriteString("## Completed Goals\n\n")
		for _, g := range completed {
			sb.WriteString(fmt.Sprintf("- ~~%s~~ (id: %s, completed: %s)\n",
				g.Title, g.ID, g.UpdatedAt.Format("2006-01-02 15:04")))
		}
		sb.WriteString("\n")
	}

	if len(dropped) > 0 {
		sb.WriteString("## Dropped Goals\n\n")
		for _, g := range dropped {
			sb.WriteString(fmt.Sprintf("- ~~%s~~ (id: %s)\n", g.Title, g.ID))
		}
		sb.WriteString("\n")
	}

	return fileutil.WriteFileAtomic(gs.goalsMDPath(), []byte(sb.String()), 0o644)
}

func filterGoals(goals []Goal, status GoalStatus) []Goal {
	var out []Goal
	for _, g := range goals {
		if g.Status == status {
			out = append(out, g)
		}
	}
	return out
}

// GoalTool provides goal management for fully autonomous operation.
// The agent uses this tool to set, track, and complete long-running goals
// that persist across heartbeat cycles.
type GoalTool struct {
	store *GoalStore
}

// NewGoalTool creates a GoalTool backed by the given workspace directory.
// maxGoals=0 means unlimited.
func NewGoalTool(workspace string, maxGoals int) *GoalTool {
	return &GoalTool{store: newGoalStore(workspace, maxGoals)}
}

func (t *GoalTool) Name() string { return "goal" }

func (t *GoalTool) Description() string {
	return "Manage autonomous goals. Use 'set' to create a new goal, 'list' to see all goals, " +
		"'complete' to mark a goal done, 'update' to change a goal's description or priority, " +
		"and 'drop' to abandon a goal. Goals persist across heartbeat cycles so you can work " +
		"toward them over time without human prompting."
}

func (t *GoalTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"set", "list", "complete", "update", "drop"},
				"description": "Action: 'set' adds a goal, 'list' shows all goals, 'complete'/'drop' closes a goal by id, 'update' modifies a goal.",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Short title of the goal (required for 'set').",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Detailed description of the goal and how to achieve it (used with 'set' and 'update').",
			},
			"priority": map[string]any{
				"type":        "integer",
				"description": "Priority 1-5 where 1 is highest urgency (default: 3). Used with 'set' and 'update'.",
			},
			"id": map[string]any{
				"type":        "string",
				"description": "Goal ID (required for 'complete', 'drop', and 'update').",
			},
		},
		"required": []string{"action"},
	}
}

func (t *GoalTool) Execute(_ context.Context, args map[string]any) *ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}
	switch action {
	case "set":
		return t.setGoal(args)
	case "list":
		return t.listGoals()
	case "complete":
		return t.closeGoal(args, GoalStatusCompleted)
	case "drop":
		return t.closeGoal(args, GoalStatusDropped)
	case "update":
		return t.updateGoal(args)
	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

func (t *GoalTool) setGoal(args map[string]any) *ToolResult {
	title, _ := args["title"].(string)
	if title == "" {
		return ErrorResult("title is required for action 'set'")
	}

	goals, err := t.store.load()
	if err != nil {
		return ErrorResult(fmt.Sprintf("loading goals: %v", err))
	}

	maxGoals := t.store.maxGoals
	if maxGoals > 0 {
		active := filterGoals(goals, GoalStatusActive)
		if len(active) >= maxGoals {
			return ErrorResult(fmt.Sprintf(
				"goal limit reached (%d active goals). Complete or drop a goal before adding a new one.", maxGoals))
		}
	}

	priority := 3
	if p, ok := args["priority"].(float64); ok && p >= 1 && p <= 5 {
		priority = int(p)
	}
	description, _ := args["description"].(string)

	now := time.Now()
	goal := Goal{
		ID:          uuid.New().String()[:8],
		Title:       title,
		Description: description,
		Priority:    priority,
		Status:      GoalStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	goals = append(goals, goal)
	if err := t.store.save(goals); err != nil {
		return ErrorResult(fmt.Sprintf("saving goal: %v", err))
	}

	return SilentResult(fmt.Sprintf("Goal set: [%s] %s (id: %s)", priorityLabel(priority), title, goal.ID))
}

func (t *GoalTool) listGoals() *ToolResult {
	goals, err := t.store.load()
	if err != nil {
		return ErrorResult(fmt.Sprintf("loading goals: %v", err))
	}

	active := filterGoals(goals, GoalStatusActive)
	if len(active) == 0 {
		return SilentResult("No active goals.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Active goals (%d):\n", len(active)))
	for _, g := range active {
		sb.WriteString(fmt.Sprintf("  [%s] %s (id: %s)\n", priorityLabel(g.Priority), g.Title, g.ID))
		if g.Description != "" {
			lines := strings.Split(g.Description, "\n")
			for _, line := range lines {
				sb.WriteString("    " + line + "\n")
			}
		}
	}
	return SilentResult(sb.String())
}

func (t *GoalTool) closeGoal(args map[string]any, status GoalStatus) *ToolResult {
	id, _ := args["id"].(string)
	if id == "" {
		return ErrorResult("id is required")
	}

	goals, err := t.store.load()
	if err != nil {
		return ErrorResult(fmt.Sprintf("loading goals: %v", err))
	}

	found := false
	for i, g := range goals {
		if g.ID == id {
			goals[i].Status = status
			goals[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}
	if !found {
		return ErrorResult(fmt.Sprintf("goal not found: %s", id))
	}

	if err := t.store.save(goals); err != nil {
		return ErrorResult(fmt.Sprintf("saving goals: %v", err))
	}

	verb := "completed"
	if status == GoalStatusDropped {
		verb = "dropped"
	}
	return SilentResult(fmt.Sprintf("Goal %s: %s", verb, id))
}

func (t *GoalTool) updateGoal(args map[string]any) *ToolResult {
	id, _ := args["id"].(string)
	if id == "" {
		return ErrorResult("id is required for action 'update'")
	}

	goals, err := t.store.load()
	if err != nil {
		return ErrorResult(fmt.Sprintf("loading goals: %v", err))
	}

	found := false
	for i, g := range goals {
		if g.ID == id {
			if desc, ok := args["description"].(string); ok && desc != "" {
				goals[i].Description = desc
			}
			if p, ok := args["priority"].(float64); ok && p >= 1 && p <= 5 {
				goals[i].Priority = int(p)
			}
			if title, ok := args["title"].(string); ok && title != "" {
				goals[i].Title = title
			}
			goals[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}
	if !found {
		return ErrorResult(fmt.Sprintf("goal not found: %s", id))
	}

	if err := t.store.save(goals); err != nil {
		return ErrorResult(fmt.Sprintf("saving goals: %v", err))
	}

	return SilentResult(fmt.Sprintf("Goal updated: %s", id))
}

func priorityLabel(p int) string {
	switch p {
	case 1:
		return "P1-critical"
	case 2:
		return "P2-high"
	case 3:
		return "P3-normal"
	case 4:
		return "P4-low"
	case 5:
		return "P5-someday"
	default:
		return fmt.Sprintf("P%d", p)
	}
}
