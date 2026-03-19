package tools

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// clarifyingSystemPrompt returns the system prompt for the clarifying phase.

func clarifyingSystemPrompt() string {
	return `You are a deliberate subagent in the CLARIFYING phase.

Your job is to understand the task fully before acting. You MUST:

1. Read relevant files and gather context using your tools.

2. If anything is unclear, use ask_conductor to ask the conductor.

3. When you have a clear plan, use submit_plan with a goal and steps.



Do NOT execute any changes yet. Only investigate and plan.

Available escalation tools: ask_conductor, submit_plan.`
}

// executingSystemPrompt returns the system prompt for the executing phase.

func executingSystemPrompt() string {
	return `You are a deliberate subagent in the EXECUTING phase. Your plan was approved.

Execute the plan steps methodically. Use all available tools to complete the work.

After completing, provide a clear summary of what was done and how it was verified.



If you encounter a blocker, use ask_conductor to escalate.`
}

// exploratorySystemPrompt returns the system prompt for exploratory presets.

func exploratorySystemPrompt(p Preset) string {
	switch p {
	case PresetScout, PresetAnalyst:

		return `You are an exploratory subagent. Investigate the task and report your findings.

Use your best judgment when encountering ambiguity. Use tools as needed.

Return clear findings and observations.`

	default:

		return `You are a subagent. Complete the given task independently and report the result.

You have access to tools - use them as needed to complete your task.

After completing the task, provide a clear summary of what was done.`
	}
}

// runExploratoryTask runs a single-phase tool loop for exploratory presets.

func (sm *SubagentManager) runExploratoryTask(
	ctx context.Context,
	task *SubagentTask,
	preset Preset,
	callback AsyncCallback,
) {
	systemPrompt := buildSubagentSystemPrompt(exploratorySystemPrompt(preset), sm.workspace)

	messages := []providers.Message{
		{Role: "system", Content: systemPrompt},

		{Role: "user", Content: task.Task},
	}

	select {
	case <-ctx.Done():

		sm.mu.Lock()

		task.Status = "canceled"

		task.Result = "Task canceled before execution"

		sm.mu.Unlock()

		return

	default:
	}

	sm.mu.RLock()

	reg := sm.tools

	if IsValidPreset(preset) {
		reg = sm.buildPresetRegistry(preset, sm.workspace, task)
	}

	maxIter := sm.maxIterations

	sm.mu.RUnlock()

	sm.reporter.ReportConversation("conductor", task.ID, task.Task)

	loopResult, err := RunToolLoop(ctx, ToolLoopConfig{
		Provider: sm.provider,

		Model: sm.defaultModel,

		Tools: reg,

		MaxIterations: maxIter,

		LLMOptions: sm.getLLMOptions(),

		Reporter: sm.reporter,

		AgentID: task.ID,
	}, messages, task.OriginChannel, task.OriginChatID)

	sm.finishTask(ctx, task, messages, loopResult, err, callback)
}

// buildPresetRegistry constructs a ToolRegistry for the given preset with appropriate restrictions.

// If task is non-nil and has escalation channels, ask_conductor and submit_plan are registered.

func (sm *SubagentManager) buildPresetRegistry(preset Preset, writeRoot string, task ...*SubagentTask) *ToolRegistry {
	registry := NewToolRegistry()

	config := SandboxConfigForPreset(preset, writeRoot)

	readRoot := writeRoot

	if readRoot == "" {
		readRoot = sm.workspace
	}

	// Register read_file and list_dir with restrict=true

	if config.AllowedTools["read_file"] {
		registry.Register(NewReadFileTool(readRoot, true, 0))
	}

	if config.AllowedTools["list_dir"] {
		registry.Register(NewListDirTool(readRoot, true))
	}

	// Register write tools only if allowed and writeRoot is set

	if config.AllowedTools["write_file"] && writeRoot != "" {
		registry.Register(NewWriteFileTool(writeRoot, true))

		registry.Register(NewEditFileTool(writeRoot, true))

		registry.Register(NewAppendFileTool(writeRoot, true))
	}

	// Register exec and bg_monitor if allowed.

	// Each subagent gets its own ExecTool to avoid mutating the shared instance's

	// allowRules (which would leak sandbox restrictions to the conductor).

	if config.AllowedTools["exec"] {
		execWorkDir := writeRoot

		if execWorkDir == "" {
			execWorkDir = sm.workspace
		}

		execTool, err := NewExecTool(execWorkDir, true)
		if err != nil {
			// exec disabled for this subagent; skip registration

			return registry
		}

		if config.ExecPolicy != nil {
			execTool.SetAllowRules(config.ExecPolicy.AllowRules)

			execTool.SetLocalNetOnly(config.ExecPolicy.LocalNetOnly)
		}

		registry.Register(execTool)

		if config.AllowedTools["bg_monitor"] {
			registry.Register(NewBgMonitorTool(execTool))
		}
	}

	// Register git tools (worktree-safe push and PR creation)

	if config.AllowedTools["git_push"] {
		registry.Register(NewGitPushTool())
	}

	if config.AllowedTools["create_pr"] {
		registry.Register(NewCreatePRTool())
	}

	// Register web tools

	if config.AllowedTools["web_search"] {
		webSearchTool, _ := NewWebSearchTool(sm.webSearchOpts)

		if webSearchTool != nil {
			registry.Register(webSearchTool)
		}
	}

	if config.AllowedTools["web_fetch"] {
		if fetchTool, err := NewWebFetchTool(50000, "", 0); err == nil {
			registry.Register(fetchTool)
		}
	}

	// Register message tool (always available)

	registry.Register(NewMessageTool())

	// Register spawn tool only for coordinator preset

	if config.AllowedTools["spawn"] && preset == PresetCoordinator {
		spawnTool := NewSpawnTool(sm)

		registry.Register(spawnTool)
	}

	// Register escalation tools for deliberate presets with channels.

	if len(task) > 0 && task[0] != nil && task[0].outCh != nil {
		t := task[0]

		subKey := "subagent:" + t.ID

		registry.Register(NewAskConductorTool(

			t.ID, sm.conductorSessionKey, subKey,

			t.outCh, t.inCh, sm.recorder,
		))

		submitPlan := NewSubmitPlanTool(

			t.ID, sm.conductorSessionKey, subKey,

			t.outCh, t.inCh, sm.recorder,
		)

		submitPlan.SetPlanCallback(func(goal string, steps []string) {
			t.PlanGoal = goal

			t.PlanSteps = steps
		})

		registry.Register(submitPlan)
	}

	return registry
}

// extractPlanContext reads MEMORY.md from the workspace and extracts relevant

// sections (Task, Context, Commands) to provide as subagent environment.

func extractPlanContext(workspace string) string {
	memPath := filepath.Join(workspace, "memory", "MEMORY.md")

	data, err := os.ReadFile(memPath)
	if err != nil {
		return ""
	}

	content := string(data)

	var sections []string

	// Extract key sections by header.

	for _, header := range []string{"## Context", "## Commands", "## Orchestration"} {
		if section := extractSection(content, header); section != "" {
			sections = append(sections, section)
		}
	}

	// Also extract the task line from the header block.

	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "> Task:") {
			sections = append([]string{strings.TrimSpace(line)}, sections...)

			break
		}
	}

	if len(sections) == 0 {
		return ""
	}

	return strings.Join(sections, "\n\n")
}

// extractSection extracts a markdown section by header (including its content

// until the next section of the same or higher level).

func extractSection(content, header string) string {
	idx := strings.Index(content, header)

	if idx < 0 {
		return ""
	}

	// Determine header level.

	level := 0

	for _, c := range header {
		if c == '#' {
			level++
		} else {
			break
		}
	}

	start := idx

	rest := content[idx+len(header):]

	// Find next section at same or higher level.

	nextHeader := "\n" + strings.Repeat("#", level) + " "

	end := strings.Index(rest, nextHeader)

	if end < 0 {
		return strings.TrimSpace(content[start:])
	}

	return strings.TrimSpace(content[start : start+len(header)+end])
}

// buildSubagentSystemPrompt builds an enriched system prompt for a subagent

// by combining the base prompt with environment context from MEMORY.md.

func buildSubagentSystemPrompt(basePrompt, workspace string) string {
	envContext := extractPlanContext(workspace)

	if envContext == "" {
		return basePrompt
	}

	return basePrompt + "\n\n## Environment Context\n\n" + envContext
}

// formatToolStats formats a tool stats map as a compact string: "exec:3,read_file:5".

// Keys are sorted alphabetically for deterministic output.

func formatToolStats(stats map[string]int) string {
	keys := make([]string, 0, len(stats))

	for k := range stats {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	parts := make([]string, 0, len(keys))

	for _, k := range keys {
		parts = append(parts, k+":"+strconv.Itoa(stats[k]))
	}

	return strings.Join(parts, ",")
}
