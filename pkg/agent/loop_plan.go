package agent

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/git"
	"github.com/sipeed/picoclaw/pkg/orch"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// interviewRejectMessage is the fixed rejection text injected when tool calls

// are blocked during the interview phase.  It is deliberately short to avoid

// wasting tokens, and ends with a purpose reminder to steer the LLM back.

const interviewRejectMessage = "[System] Tool call rejected. " +

	"You are in interview mode — ask the user questions and update MEMORY.md. " +

	"Do not execute, edit, or write project files."

// buildPlanReminder returns a reminder message for plan pre-execution states

// (interviewing / review) to keep the AI focused on the interview workflow

// during tool-call iterations.

func buildPlanReminder(planStatus string) (providers.Message, bool) {
	var content string

	switch planStatus {
	case "interviewing":

		content = "[System] You are interviewing the user to build a plan. " +

			"Ask clarifying questions and save findings to ## Context in memory/MEMORY.md using edit_file. " +

			"When you have enough information, write ## Phase sections with `- [ ]` checkbox steps, and ## Commands section. " +

			"Then change > Status: to review. Do NOT set it to executing."

	case "review":

		content = "[System] The plan is under review. " +

			"Wait for the user to approve or request changes. Do not proceed with execution."

	default:

		return providers.Message{}, false
	}

	return providers.Message{Role: "user", Content: content}, true
}

func (al *AgentLoop) handlePlanCommand(args []string, sessionKey string) (string, bool) {
	agent := al.registry.GetDefaultAgent()

	if agent == nil {
		return "No agent configured.", true
	}

	if len(args) == 0 {
		// /plan — show current plan

		return agent.ContextBuilder.FormatPlanDisplay(), true
	}

	sub := args[0]

	switch sub {
	case "clear":

		if agent.ContextBuilder.ReadMemory() == "" {
			return "No active plan to clear.", true
		}

		// Deactivate worktree on plan clear

		if sessionKey != "" {
			agent.DeactivateWorktree(sessionKey, "", true)
		}

		if err := agent.ContextBuilder.ClearMemory(); err != nil {
			return fmt.Sprintf("Error clearing plan: %v", err), true
		}

		return "Plan cleared.", true

	case "done":

		if !agent.ContextBuilder.HasActivePlan() {
			return "No active plan.", true
		}

		if len(args) < 2 {
			return "Usage: /plan done <step number>", true
		}

		stepNum, err := strconv.Atoi(args[1])

		if err != nil || stepNum < 1 {
			return "Step number must be a positive integer.", true
		}

		phase := agent.ContextBuilder.GetCurrentPhase()

		if err := agent.ContextBuilder.MarkStep(phase, stepNum); err != nil {
			return fmt.Sprintf("Error: %v", err), true
		}

		return fmt.Sprintf("Marked step %d in phase %d as done.", stepNum, phase), true

	case "add":

		if !agent.ContextBuilder.HasActivePlan() {
			return "No active plan.", true
		}

		if len(args) < 2 {
			return "Usage: /plan add <step description>", true
		}

		desc := strings.Join(args[1:], " ")

		phase := agent.ContextBuilder.GetCurrentPhase()

		if err := agent.ContextBuilder.AddStep(phase, desc); err != nil {
			return fmt.Sprintf("Error: %v", err), true
		}

		return fmt.Sprintf("Added step to phase %d: %s", phase, desc), true

	case "start":

		if !agent.ContextBuilder.HasActivePlan() {
			return "No active plan.", true
		}

		status := agent.ContextBuilder.GetPlanStatus()

		if status == "executing" {
			return "Plan is already executing.", true
		}

		if status != "interviewing" && status != "review" {
			return fmt.Sprintf("Cannot start from status %q.", status), true
		}

		if agent.ContextBuilder.GetTotalPhases() == 0 {
			return "Cannot start: no phases defined yet. Complete the interview first.", true
		}

		if err := agent.ContextBuilder.SetPlanStatus("executing"); err != nil {
			return fmt.Sprintf("Error: %v", err), true
		}

		al.reporter().ReportStateChange(sessionKey, orch.AgentStatePlanExecuting, "")

		al.planStartPending = true

		clearHistory := len(args) > 1 && args[1] == "clear"

		al.planClearHistory = clearHistory

		if clearHistory {
			return "Plan approved. Executing with clean history.", true
		}

		return "Plan approved. Executing.", true

	case "next":

		if !agent.ContextBuilder.HasActivePlan() {
			return "No active plan.", true
		}

		if err := agent.ContextBuilder.AdvancePhase(); err != nil {
			return fmt.Sprintf("Error: %v", err), true
		}

		phase := agent.ContextBuilder.GetCurrentPhase()

		return fmt.Sprintf("Advanced to phase %d.", phase), true

	case "worktrees":

		return al.handlePlanWorktreesCommand(agent, args[1:]), true

	default:

		// /plan <task description> — start new plan

		// Block if a plan is already active (fast-path error).

		if agent.ContextBuilder.HasActivePlan() {
			return "A plan is already active. Use /plan clear first.", true
		}

		// Not handled here — let the message flow to the LLM queue.

		// expandPlanCommand will write the seed and rewrite the content.

		return "", false
	}
}

func (al *AgentLoop) handlePlanWorktreesCommand(agent *AgentInstance, args []string) string {
	repoRoot := git.FindRepoRoot(agent.Workspace)

	if repoRoot == "" {
		return "Workspace is not a git repository."
	}

	worktreesDir := filepath.Join(agent.Workspace, ".worktrees")

	sub := "list"

	if len(args) > 0 {
		sub = strings.ToLower(strings.TrimSpace(args[0]))
	}

	switch sub {
	case "", "list":

		items, err := git.ListManagedWorktrees(repoRoot, worktreesDir)
		if err != nil {
			return fmt.Sprintf("Error listing worktrees: %v", err)
		}

		if len(items) == 0 {
			return "No active worktrees in workspace/.worktrees."
		}

		var sb strings.Builder

		sb.WriteString("Active worktrees\n\n")

		for _, wt := range items {
			status := "clean"

			if wt.HasUncommitted {
				status = "dirty"
			}

			last := "(no commits)"

			if wt.LastCommitHash != "" {
				if wt.LastCommitAge != "" {
					last = fmt.Sprintf("%s %s (%s)", wt.LastCommitHash, wt.LastCommitSubject, wt.LastCommitAge)
				} else {
					last = fmt.Sprintf("%s %s", wt.LastCommitHash, wt.LastCommitSubject)
				}
			}

			fmt.Fprintf(&sb, "- %s\n  branch: %s\n  status: %s\n  last: %s\n", wt.Name, wt.Branch, status, last)
		}

		sb.WriteString("\nCommands:\n")

		sb.WriteString("/plan worktrees inspect <name>\n")

		sb.WriteString("/plan worktrees merge <name>\n")

		sb.WriteString("/plan worktrees dispose <name> [force]")

		return sb.String()

	case "inspect":

		if len(args) < 2 {
			return "Usage: /plan worktrees inspect <name>"
		}

		name := args[1]

		wt, err := git.GetManagedWorktree(repoRoot, worktreesDir, name)
		if err != nil {
			if errors.Is(err, git.ErrInvalidWorktreeName) {
				return "Invalid worktree name."
			}

			if errors.Is(err, git.ErrWorktreeNotFound) {
				return fmt.Sprintf("Worktree %q not found.", name)
			}

			return fmt.Sprintf("Error inspecting worktree %q: %v", name, err)
		}

		statusOut, _ := git.WorktreeStatusShort(wt.Path)

		diffOut, _ := git.WorktreeDiffStat(wt.Path)

		logOut, _ := git.WorktreeRecentLog(wt.Path, 10)

		if statusOut == "" {
			statusOut = "(clean)"
		}

		var sb strings.Builder

		fmt.Fprintf(&sb, "Worktree: %s\nBranch: %s\nDirty: %t\n", wt.Name, wt.Branch, wt.HasUncommitted)

		if wt.LastCommitHash != "" {
			fmt.Fprintf(&sb, "Last commit: %s %s", wt.LastCommitHash, wt.LastCommitSubject)

			if wt.LastCommitAge != "" {
				fmt.Fprintf(&sb, " (%s)", wt.LastCommitAge)
			}

			sb.WriteString("\n")
		}

		sb.WriteString("\nStatus:\n```\n")

		sb.WriteString(statusOut)

		sb.WriteString("\n```\n")

		if diffOut != "" {
			sb.WriteString("\nDiff (stat):\n```\n")

			sb.WriteString(diffOut)

			sb.WriteString("\n```\n")
		}

		if logOut != "" {
			sb.WriteString("\nRecent commits:\n```\n")

			sb.WriteString(logOut)

			sb.WriteString("\n```")
		}

		return sb.String()

	case "merge":

		if len(args) < 2 {
			return "Usage: /plan worktrees merge <name>"
		}

		name := args[1]

		res, base, err := git.MergeManagedWorktree(repoRoot, worktreesDir, name, "")
		if err != nil {
			if errors.Is(err, git.ErrInvalidWorktreeName) {
				return "Invalid worktree name."
			}

			if errors.Is(err, git.ErrWorktreeNotFound) {
				return fmt.Sprintf("Worktree %q not found.", name)
			}

			return fmt.Sprintf("Error merging worktree %q: %v", name, err)
		}

		if res.Conflict {
			return fmt.Sprintf("Merge conflict while merging `%s` into `%s`. Merge was aborted.", res.Branch, base)
		}

		if res.Merged {
			return fmt.Sprintf("Merged `%s` into `%s`.", res.Branch, base)
		}

		return fmt.Sprintf("No merge was performed for `%s`.", name)

	case "dispose":

		if len(args) < 2 {
			return "Usage: /plan worktrees dispose <name> [force]"
		}

		name := args[1]

		force := len(args) > 2 && strings.EqualFold(args[2], "force")

		wt, err := git.GetManagedWorktree(repoRoot, worktreesDir, name)
		if err != nil {
			if errors.Is(err, git.ErrInvalidWorktreeName) {
				return "Invalid worktree name."
			}

			if errors.Is(err, git.ErrWorktreeNotFound) {
				return fmt.Sprintf("Worktree %q not found.", name)
			}

			return fmt.Sprintf("Error disposing worktree %q: %v", name, err)
		}

		if wt.HasUncommitted && !force {
			return fmt.Sprintf(

				"Worktree `%s` has uncommitted changes. Re-run with `/plan worktrees dispose %s force` to confirm.",

				name,

				name,
			)
		}

		res, err := git.DisposeManagedWorktree(repoRoot, worktreesDir, name, "")
		if err != nil {
			return fmt.Sprintf("Error disposing worktree %q: %v", name, err)
		}

		parts := []string{fmt.Sprintf("Disposed worktree `%s` (branch `%s`).", name, res.Branch)}

		if res.AutoCommitted {
			parts = append(parts, "Uncommitted changes were auto-committed.")
		}

		if res.CommitsAhead > 0 {
			parts = append(parts, fmt.Sprintf("Branch has %d unique commit(s); branch was kept.", res.CommitsAhead))
		}

		if res.BranchDeleted {
			parts = append(parts, "Branch was deleted (no unique commits).")
		}

		return strings.Join(parts, " ")
	}

	return "Usage: /plan worktrees [list|inspect <name>|merge <name>|dispose <name> [force]]"
}

// isPlanPreExecution returns true if the plan is in a pre-execution state

// (interviewing or review) where tool restrictions and iteration caps apply.

func isPlanPreExecution(status string) bool {
	return status == "interviewing" || status == "review"
}

// interviewAllowedTools is the single source of truth for tool names that may

// be sent to the LLM (and subsequently invoked) during the interview phase.

// filterInterviewTools uses this to strip tool *definitions* before the LLM call,

// while isToolAllowedDuringInterview adds argument-level checks as a second gate.

var interviewAllowedTools = map[string]bool{
	"readfile": true,

	"listdir": true,

	"websearch": true,

	"webfetch": true,

	"message": true,

	"editfile": true,

	"appendfile": true,

	"writefile": true,

	"exec": true,

	"logs": true,
}

// filterInterviewTools removes tool definitions that are not in the

// interviewAllowedTools whitelist, reducing token usage and preventing the

// LLM from attempting disallowed tool calls during the interview phase.

func filterInterviewTools(defs []providers.ToolDefinition) []providers.ToolDefinition {
	filtered := make([]providers.ToolDefinition, 0, len(defs))

	for _, d := range defs {
		if interviewAllowedTools[tools.NormalizeToolName(d.Function.Name)] {
			filtered = append(filtered, d)
		}
	}

	return filtered
}

// isToolAllowedDuringInterview checks whether a tool call is permitted while the

// plan is in a pre-execution state. Uses the shared interviewAllowedTools map for

// name-level gating, then applies argument-level constraints for write-type tools

// (MEMORY.md only) and exec (read-only commands only).

func isToolAllowedDuringInterview(toolName string, args map[string]any) bool {
	norm := tools.NormalizeToolName(toolName)

	if !interviewAllowedTools[norm] {
		return false
	}

	// Argument-level constraints

	switch norm {
	case "editfile", "appendfile", "writefile":

		path, _ := args["path"].(string)

		return strings.HasSuffix(path, "MEMORY.md")

	case "exec":

		cmd, _ := args["command"].(string)

		return isReadOnlyCommand(cmd)
	}

	return true
}

// isReadOnlyCommand returns true when cmd is a safe, read-only shell command

// that an LLM may run during the interview phase.

func isReadOnlyCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)

	if cmd == "" {
		return false
	}

	// Reject write operators anywhere in the command

	for _, op := range []string{">", ">>", "| tee "} {
		if strings.Contains(cmd, op) {
			return false
		}
	}

	// Reject path traversal (defense in depth; ExecTool.guardCommand also enforces workspace restriction)

	if strings.Contains(cmd, "..") {
		return false
	}

	// Block absolute paths in arguments (allow "cd /path && cmd" which is stripped later)

	for _, field := range strings.Fields(cmd) {
		if strings.HasPrefix(field, "/") && !strings.HasPrefix(cmd, "cd ") {
			return false
		}
	}

	// Strip "cd /path &&" prefix (LLM habit)

	if strings.HasPrefix(cmd, "cd ") {
		if idx := strings.Index(cmd, "&&"); idx >= 0 {
			cmd = strings.TrimSpace(cmd[idx+2:])
		}
	}

	fields := strings.Fields(cmd)

	if len(fields) == 0 {
		return false
	}

	first := filepath.Base(fields[0])

	switch first {
	case "find", "ls", "cat", "head", "tail", "grep", "rg",

		"tree", "wc", "file", "which", "pwd",

		"uname", "df", "du", "stat", "realpath", "dirname",

		"basename", "date":

		return true
	}

	return false
}

// isWriteTool returns true if the tool can modify files.

func isWriteTool(name string) bool {
	switch tools.NormalizeToolName(name) {
	case "writefile", "editfile", "appendfile", "exec":

		return true
	}

	return false
}

// expandPlanCommand detects "/plan <task>" (new plan start) and:

//   - writes the interview seed to MEMORY.md

//   - rewrites the message content for the LLM

//   - returns a compact form for session history

//

// This follows the same pattern as expandSkillCommand: the message is

// rewritten before reaching the LLM, so the AI sees the task description

// while the system prompt contains the interview guide.

func (al *AgentLoop) expandPlanCommand(msg bus.InboundMessage) (expanded string, compact string, ok bool) {
	content := strings.TrimSpace(msg.Content)

	if !strings.HasPrefix(content, "/plan ") {
		return "", "", false
	}

	task := strings.TrimSpace(content[6:]) // len("/plan ") == 6

	if task == "" {
		return "", "", false
	}

	// Known subcommands are handled by handlePlanCommand (fast path).

	firstWord := strings.Fields(task)[0]

	switch firstWord {
	case "clear", "done", "add", "start", "next", "worktrees":

		return "", "", false
	}

	agent := al.registry.GetDefaultAgent()

	if agent == nil {
		return "", "", false
	}

	// If a plan is already active, don't expand — handleCommand will

	// catch it and return the error on the fast path.

	if agent.ContextBuilder.HasActivePlan() {
		return "", "", false
	}

	// Write the interview seed

	seed := BuildInterviewSeed(task, agent.Workspace)

	if err := agent.ContextBuilder.WriteMemory(seed); err != nil {
		return "", "", false
	}

	al.notifyStateChange()

	// Expanded: the task description goes to LLM.

	// The system prompt already contains the interview guide.

	expanded = task

	compact = fmt.Sprintf("[Plan: %s]", utils.Truncate(task, 80))

	return expanded, compact, true
}
