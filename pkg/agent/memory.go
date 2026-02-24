// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MemoryStore manages persistent memory for the agent.
// - Long-term memory: memory/MEMORY.md
// - Daily notes: memory/YYYYMM/YYYYMMDD.md
type MemoryStore struct {
	workspace  string
	memoryDir  string
	memoryFile string
}

// NewMemoryStore creates a new MemoryStore with the given workspace path.
// It ensures the memory directory exists.
func NewMemoryStore(workspace string) *MemoryStore {
	memoryDir := filepath.Join(workspace, "memory")
	memoryFile := filepath.Join(memoryDir, "MEMORY.md")

	// Ensure memory directory exists
	os.MkdirAll(memoryDir, 0o755)

	return &MemoryStore{
		workspace:  workspace,
		memoryDir:  memoryDir,
		memoryFile: memoryFile,
	}
}

// getTodayFile returns the path to today's daily note file (memory/YYYYMM/YYYYMMDD.md).
func (ms *MemoryStore) getTodayFile() string {
	today := time.Now().Format("20060102") // YYYYMMDD
	monthDir := today[:6]                  // YYYYMM
	filePath := filepath.Join(ms.memoryDir, monthDir, today+".md")
	return filePath
}

// ReadLongTerm reads the long-term memory (MEMORY.md).
// Returns empty string if the file doesn't exist.
func (ms *MemoryStore) ReadLongTerm() string {
	if data, err := os.ReadFile(ms.memoryFile); err == nil {
		return string(data)
	}
	return ""
}

// WriteLongTerm writes content to the long-term memory file (MEMORY.md).
func (ms *MemoryStore) WriteLongTerm(content string) error {
	return os.WriteFile(ms.memoryFile, []byte(content), 0o644)
}

// ClearLongTerm removes the long-term memory file.
func (ms *MemoryStore) ClearLongTerm() error {
	if err := os.Remove(ms.memoryFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ReadToday reads today's daily note.
// Returns empty string if the file doesn't exist.
func (ms *MemoryStore) ReadToday() string {
	todayFile := ms.getTodayFile()
	if data, err := os.ReadFile(todayFile); err == nil {
		return string(data)
	}
	return ""
}

// AppendToday appends content to today's daily note.
// If the file doesn't exist, it creates a new file with a date header.
func (ms *MemoryStore) AppendToday(content string) error {
	todayFile := ms.getTodayFile()

	// Ensure month directory exists
	monthDir := filepath.Dir(todayFile)
	os.MkdirAll(monthDir, 0o755)

	var existingContent string
	if data, err := os.ReadFile(todayFile); err == nil {
		existingContent = string(data)
	}

	var newContent string
	if existingContent == "" {
		// Add header for new day
		header := fmt.Sprintf("# %s\n\n", time.Now().Format("2006-01-02"))
		newContent = header + content
	} else {
		// Append to existing content
		newContent = existingContent + "\n" + content
	}

	return os.WriteFile(todayFile, []byte(newContent), 0o644)
}

// GetRecentDailyNotes returns daily notes from the last N days.
// Contents are joined with "---" separator.
func (ms *MemoryStore) GetRecentDailyNotes(days int) string {
	var sb strings.Builder
	first := true

	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("20060102") // YYYYMMDD
		monthDir := dateStr[:6]            // YYYYMM
		filePath := filepath.Join(ms.memoryDir, monthDir, dateStr+".md")

		if data, err := os.ReadFile(filePath); err == nil {
			if !first {
				sb.WriteString("\n\n---\n\n")
			}
			sb.Write(data)
			first = false
		}
	}

	return sb.String()
}

// ---------- Plan state query methods ----------

var (
	reActivePlan  = regexp.MustCompile(`(?m)^# Active Plan`)
	reStatus      = regexp.MustCompile(`(?m)^> Status:\s*(.+)`)
	rePhase       = regexp.MustCompile(`(?m)^> Phase:\s*(\d+)`)
	rePhaseHeader = regexp.MustCompile(`(?m)^## Phase (\d+):\s*(.*)`)
	reStepDone    = regexp.MustCompile(`(?m)^- \[x\] `)
	reStepTodo    = regexp.MustCompile(`(?m)^- \[ \] `)
	reWorkDir     = regexp.MustCompile(`(?m)^> WorkDir:\s*(.+)`)
)

// HasActivePlan returns true if MEMORY.md contains an active plan.
func (ms *MemoryStore) HasActivePlan() bool {
	content := ms.ReadLongTerm()
	return reActivePlan.MatchString(content)
}

// GetPlanStatus returns the plan status: "interviewing", "executing", or "".
func (ms *MemoryStore) GetPlanStatus() string {
	content := ms.ReadLongTerm()
	m := reStatus.FindStringSubmatch(content)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// GetCurrentPhase returns the current phase number from "> Phase: N".
func (ms *MemoryStore) GetCurrentPhase() int {
	content := ms.ReadLongTerm()
	m := rePhase.FindStringSubmatch(content)
	if len(m) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// GetTotalPhases returns the total number of phases (max ## Phase N).
func (ms *MemoryStore) GetTotalPhases() int {
	content := ms.ReadLongTerm()
	matches := rePhaseHeader.FindAllStringSubmatch(content, -1)
	max := 0
	for _, m := range matches {
		if len(m) >= 2 {
			n, _ := strconv.Atoi(m[1])
			if n > max {
				max = n
			}
		}
	}
	return max
}

// IsPlanComplete returns true if all steps in all phases are [x].
func (ms *MemoryStore) IsPlanComplete() bool {
	phases := ms.GetPlanPhases()
	if len(phases) == 0 {
		return false
	}
	hasSteps := false
	for _, p := range phases {
		for _, s := range p.Steps {
			hasSteps = true
			if !s.Done {
				return false
			}
		}
	}
	return hasSteps
}

// IsCurrentPhaseComplete returns true if all steps in the current phase are [x].
func (ms *MemoryStore) IsCurrentPhaseComplete() bool {
	current := ms.GetCurrentPhase()
	if current == 0 {
		return false
	}
	phases := ms.GetPlanPhases()
	for _, p := range phases {
		if p.Number == current {
			if len(p.Steps) == 0 {
				return false
			}
			for _, s := range p.Steps {
				if !s.Done {
					return false
				}
			}
			return true
		}
	}
	return false
}

// extractPhaseContent returns the content of a specific phase section.
func (ms *MemoryStore) extractPhaseContent(content string, phase int) string {
	lines := strings.Split(content, "\n")
	inPhase := false
	var result []string

	phasePrefix := fmt.Sprintf("## Phase %d:", phase)

	for _, line := range lines {
		if strings.HasPrefix(line, phasePrefix) {
			inPhase = true
			continue
		}
		if inPhase {
			// Stop at next phase header or Context section
			if strings.HasPrefix(line, "## Phase ") || strings.HasPrefix(line, "## Context") {
				break
			}
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

// PlanPhase represents a phase with its steps, for structured API output.
type PlanPhase struct {
	Number int        `json:"number"`
	Title  string     `json:"title"`
	Steps  []PlanStep `json:"steps"`
}

// PlanStep represents a single step within a phase.
type PlanStep struct {
	Index       int    `json:"index"` // 1-based within the phase
	Description string `json:"description"`
	Done        bool   `json:"done"`
}

// GetPlanPhases parses MEMORY.md and returns all phases with their steps.
func (ms *MemoryStore) GetPlanPhases() []PlanPhase {
	return ms.getPlanPhasesFrom(ms.ReadLongTerm())
}

func (ms *MemoryStore) getPlanPhasesFrom(content string) []PlanPhase {
	if !reActivePlan.MatchString(content) {
		return nil
	}

	totalPhases := maxPhaseNumber(content)
	phases := make([]PlanPhase, 0, totalPhases)

	for p := 1; p <= totalPhases; p++ {
		title := ms.getPhaseTitle(content, p)
		phaseContent := ms.extractPhaseContent(content, p)

		var steps []PlanStep
		stepIdx := 0
		for _, line := range strings.Split(phaseContent, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "- [x] ") {
				stepIdx++
				steps = append(steps, PlanStep{
					Index:       stepIdx,
					Description: line[6:],
					Done:        true,
				})
			} else if strings.HasPrefix(line, "- [ ] ") {
				stepIdx++
				steps = append(steps, PlanStep{
					Index:       stepIdx,
					Description: line[6:],
					Done:        false,
				})
			}
		}

		phases = append(phases, PlanPhase{
			Number: p,
			Title:  title,
			Steps:  steps,
		})
	}

	return phases
}

// ---------- Plan mutation methods ----------

// SetStatus sets the plan status (interviewing or executing).
func (ms *MemoryStore) SetStatus(status string) error {
	content := ms.ReadLongTerm()
	if m := reStatus.FindString(content); m != "" {
		content = strings.Replace(content, m, "> Status: "+status, 1)
	}
	return ms.WriteLongTerm(content)
}

// AdvancePhase increments the current phase number by 1.
func (ms *MemoryStore) AdvancePhase() error {
	content := ms.ReadLongTerm()
	m := rePhase.FindStringSubmatch(content)
	if len(m) < 2 {
		return fmt.Errorf("no phase marker found")
	}
	current, _ := strconv.Atoi(m[1])
	next := current + 1
	content = strings.Replace(content, m[0], fmt.Sprintf("> Phase: %d", next), 1)
	return ms.WriteLongTerm(content)
}

// SetPhase sets the current phase number to n.
func (ms *MemoryStore) SetPhase(n int) error {
	content := ms.ReadLongTerm()
	m := rePhase.FindString(content)
	if m == "" {
		return fmt.Errorf("no phase marker found")
	}
	content = strings.Replace(content, m, fmt.Sprintf("> Phase: %d", n), 1)
	return ms.WriteLongTerm(content)
}

// MarkStep marks the nth step (1-based) in the given phase as done [x].
func (ms *MemoryStore) MarkStep(phase, step int) error {
	content := ms.ReadLongTerm()
	lines := strings.Split(content, "\n")
	phasePrefix := fmt.Sprintf("## Phase %d:", phase)

	inPhase := false
	stepCount := 0
	for i, line := range lines {
		if strings.HasPrefix(line, phasePrefix) {
			inPhase = true
			continue
		}
		if inPhase {
			if strings.HasPrefix(line, "## Phase ") || strings.HasPrefix(line, "## Context") {
				break
			}
			if strings.HasPrefix(line, "- [ ] ") {
				stepCount++
				if stepCount == step {
					lines[i] = strings.Replace(line, "- [ ] ", "- [x] ", 1)
					return ms.WriteLongTerm(strings.Join(lines, "\n"))
				}
			}
		}
	}
	return fmt.Errorf("step %d not found in phase %d", step, phase)
}

// AddStep appends a new step to the given phase.
func (ms *MemoryStore) AddStep(phase int, desc string) error {
	content := ms.ReadLongTerm()
	lines := strings.Split(content, "\n")
	phasePrefix := fmt.Sprintf("## Phase %d:", phase)

	inPhase := false
	insertIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(line, phasePrefix) {
			inPhase = true
			continue
		}
		if inPhase {
			if strings.HasPrefix(line, "## Phase ") || strings.HasPrefix(line, "## Context") {
				insertIdx = i
				break
			}
			// Track last step line
			if strings.HasPrefix(line, "- [") {
				insertIdx = i + 1
			}
		}
	}

	if insertIdx < 0 {
		// Phase not found or empty; append at end
		if inPhase {
			insertIdx = len(lines)
		} else {
			return fmt.Errorf("phase %d not found", phase)
		}
	}

	newStep := "- [ ] " + desc
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:insertIdx]...)
	newLines = append(newLines, newStep)
	newLines = append(newLines, lines[insertIdx:]...)

	return ms.WriteLongTerm(strings.Join(newLines, "\n"))
}

// ValidatePlanStructure checks that the plan has valid structure for
// transitioning out of the interview phase. Returns nil if valid,
// or an error describing the first problem found.
func (ms *MemoryStore) ValidatePlanStructure() error {
	content := ms.ReadLongTerm()

	// 1. Header: # Active Plan must exist
	if !reActivePlan.MatchString(content) {
		return fmt.Errorf("missing '# Active Plan' header")
	}

	// 2. Required metadata lines
	if !reStatus.MatchString(content) {
		return fmt.Errorf("missing '> Status:' line")
	}
	if !rePhase.MatchString(content) {
		return fmt.Errorf("missing '> Phase:' line")
	}

	// 3. At least one phase header (## Phase N: title)
	phases := ms.GetPlanPhases()
	if len(phases) == 0 {
		return fmt.Errorf("no '## Phase N:' sections found")
	}

	// 4. Every phase must have at least one checkbox step
	for _, p := range phases {
		if len(p.Steps) == 0 {
			return fmt.Errorf("Phase %d has no checkbox steps (use '- [ ] ...')", p.Number)
		}
	}

	return nil
}

// ---------- Selective injection methods ----------

// GetPlanWorkDir returns the WorkDir from the plan metadata, or "".
func (ms *MemoryStore) GetPlanWorkDir() string {
	content := ms.ReadLongTerm()
	m := reWorkDir.FindStringSubmatch(content)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// reTaskLine extracts the task name from "> Task: <description>".
var reTaskLine = regexp.MustCompile(`(?m)^> Task:\s*(.+)`)

// GetPlanTaskName returns the task description from the plan metadata, or "".
func (ms *MemoryStore) GetPlanTaskName() string {
	content := ms.ReadLongTerm()
	m := reTaskLine.FindStringSubmatch(content)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// interviewSeed is the initial content written to MEMORY.md when /plan starts.
const interviewSeedTemplate = `# Active Plan

> Task: %s
> WorkDir: %s
> Status: interviewing
> Phase: 1
`

// BuildInterviewSeed creates the initial plan seed for a given task description.
func BuildInterviewSeed(task, workDir string) string {
	return fmt.Sprintf(interviewSeedTemplate, task, workDir)
}

// GetInterviewContext returns context for injection during the interviewing phase.
// Includes the full seed + interview guide + target format template.
func (ms *MemoryStore) GetInterviewContext() string {
	return ms.getInterviewContextFrom(ms.ReadLongTerm())
}

func (ms *MemoryStore) getInterviewContextFrom(content string) string {
	var sb strings.Builder
	sb.WriteString("## Active Plan (interviewing)\n\n")
	sb.WriteString(content)
	sb.WriteString("\n\n### Interview Guide\n")
	sb.WriteString("Ask about:\n")
	sb.WriteString("- Goals and success criteria\n")
	sb.WriteString("- Constraints (time, budget, platform)\n")
	sb.WriteString("- Environment (OS, language, runtime versions)\n")
	sb.WriteString("- Tooling preferences (test framework, linter, formatter, CI)\n")
	sb.WriteString("- Key commands the user already runs (build, test, deploy)\n")
	sb.WriteString("\n### Rules\n")
	sb.WriteString("- NEVER remove or overwrite the header block (`# Active Plan`, `> Task:`, `> Status:`, `> Phase:` lines). The system parses these to track state.\n")
	sb.WriteString("- After each answer, use edit_file to append findings to the ## Context section of memory/MEMORY.md.\n")
	sb.WriteString("- When you have enough information, use edit_file to add ## Phase, ## Commands, and ## Context sections BELOW the header block.\n")
	sb.WriteString("- Each step MUST use checkbox syntax: `- [ ] description`. The system parses checkboxes to track progress.\n")
	sb.WriteString("- Organize into 2-5 phases with 3-5 steps each.\n")
	sb.WriteString("- After writing Phases, change `> Status: interviewing` to `> Status: review` via edit_file. The user must approve with /plan start before execution begins.\n")
	sb.WriteString("\n### Target Format (MANDATORY — system parses this exact structure)\n")
	sb.WriteString("\n")
	sb.WriteString("# Active Plan\n")
	sb.WriteString("> Task: <description>\n")
	sb.WriteString("> WorkDir: <path>\n")
	sb.WriteString("> Status: interviewing\n")
	sb.WriteString("> Phase: 1\n")
	sb.WriteString("\n")
	sb.WriteString("## Phase 1: <title>\n")
	sb.WriteString("- [ ] Step description\n")
	sb.WriteString("- [ ] Step description\n")
	sb.WriteString("\n")
	sb.WriteString("## Phase 2: <title>\n")
	sb.WriteString("- [ ] Step description\n")
	sb.WriteString("- [ ] Step description\n")
	sb.WriteString("\n")
	sb.WriteString("## Commands\n")
	sb.WriteString("build: <project-specific build command>\n")
	sb.WriteString("test: <project-specific test command>\n")
	sb.WriteString("lint: <project-specific lint command>\n")
	sb.WriteString("\n")
	sb.WriteString("## Context\n")
	sb.WriteString("<collected requirements, decisions, environment>\n")
	return sb.String()
}

// GetReviewContext returns context for injection during the review phase.
// Shows the full plan and instructs the AI to wait for user approval.
func (ms *MemoryStore) GetReviewContext() string {
	return ms.getReviewContextFrom(ms.ReadLongTerm())
}

func (ms *MemoryStore) getReviewContextFrom(content string) string {
	var sb strings.Builder
	sb.WriteString("## Active Plan (awaiting approval)\n\n")
	sb.WriteString(content)
	sb.WriteString("\n\nThe plan is awaiting user approval.\n")
	sb.WriteString("- If the user requests changes, update memory/MEMORY.md via edit_file.\n")
	sb.WriteString("- Do NOT change Status yourself. The user will run /plan start to approve.\n")
	return sb.String()
}

// GetPlanContext returns context for injection during the executing phase.
// Only the current phase is shown in detail; completed phases are compressed
// to one-line summaries; future phases are omitted.
func (ms *MemoryStore) GetPlanContext() string {
	return ms.getPlanContextFrom(ms.ReadLongTerm())
}

func (ms *MemoryStore) getPlanContextFrom(content string) string {
	var currentPhase int
	if m := rePhase.FindStringSubmatch(content); len(m) >= 2 {
		currentPhase, _ = strconv.Atoi(m[1])
	}
	totalPhases := maxPhaseNumber(content)

	taskLine := ""
	if m := reTaskLine.FindStringSubmatch(content); len(m) >= 2 {
		taskLine = strings.TrimSpace(m[1])
	}

	var sb strings.Builder
	sb.WriteString("## Active Plan\n")
	fmt.Fprintf(&sb, "Task: %s | Phase %d/%d\n", taskLine, currentPhase, totalPhases)

	// Completed phases: one-line summaries
	for p := 1; p < currentPhase; p++ {
		title := ms.getPhaseTitle(content, p)
		fmt.Fprintf(&sb, "Done: Phase %d (%s)\n", p, title)
	}

	// Current phase: full detail
	if currentPhase > 0 {
		title := ms.getPhaseTitle(content, currentPhase)
		fmt.Fprintf(&sb, "### Current: Phase %d — %s\n", currentPhase, title)
		phaseContent := ms.extractPhaseContent(content, currentPhase)
		sb.WriteString(strings.TrimSpace(phaseContent))
		sb.WriteString("\n")
	}

	// Commands section: always included if present
	commandsContent := ms.extractCommandsSection(content)
	if commandsContent != "" {
		sb.WriteString("### Commands\n")
		sb.WriteString(commandsContent)
		sb.WriteString("\n")
	}

	// Context section: always included
	contextContent := ms.extractContextSection(content)
	if contextContent != "" {
		sb.WriteString("### Context\n")
		sb.WriteString(contextContent)
		sb.WriteString("\n")
	}

	return sb.String()
}

// maxPhaseNumber returns the highest phase number found in content.
func maxPhaseNumber(content string) int {
	matches := rePhaseHeader.FindAllStringSubmatch(content, -1)
	max := 0
	for _, m := range matches {
		if len(m) >= 2 {
			n, _ := strconv.Atoi(m[1])
			if n > max {
				max = n
			}
		}
	}
	return max
}

// getPhaseTitle extracts the title of a phase from "## Phase N: Title".
func (ms *MemoryStore) getPhaseTitle(content string, phase int) string {
	matches := rePhaseHeader.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		if len(m) >= 3 {
			n, _ := strconv.Atoi(m[1])
			if n == phase {
				return strings.TrimSpace(m[2])
			}
		}
	}
	return ""
}

// extractSection extracts a named ## section from the plan content.
// It returns everything between "## <name>" and the next "## " header.
func (ms *MemoryStore) extractSection(content, name string) string {
	lines := strings.Split(content, "\n")
	prefix := "## " + name
	inSection := false
	var result []string
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			inSection = true
			continue
		}
		if inSection {
			if strings.HasPrefix(line, "## ") {
				break
			}
			result = append(result, line)
		}
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

// extractContextSection extracts the ## Context section from the plan.
func (ms *MemoryStore) extractContextSection(content string) string {
	return ms.extractSection(content, "Context")
}

// extractCommandsSection extracts the ## Commands section from the plan.
func (ms *MemoryStore) extractCommandsSection(content string) string {
	return ms.extractSection(content, "Commands")
}

// FormatPlanDisplay returns a user-facing display of the full plan with emoji indicators.
func (ms *MemoryStore) FormatPlanDisplay() string {
	content := ms.ReadLongTerm()
	if !reActivePlan.MatchString(content) {
		return "No active plan."
	}

	taskLine := ""
	if m := reTaskLine.FindStringSubmatch(content); len(m) >= 2 {
		taskLine = strings.TrimSpace(m[1])
	}
	var status string
	if m := reStatus.FindStringSubmatch(content); len(m) >= 2 {
		status = strings.TrimSpace(m[1])
	}
	var currentPhase int
	if m := rePhase.FindStringSubmatch(content); len(m) >= 2 {
		currentPhase, _ = strconv.Atoi(m[1])
	}
	phases := ms.getPlanPhasesFrom(content)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Plan: %s\n", taskLine))
	sb.WriteString(fmt.Sprintf("Status: %s | Phase %d/%d\n\n", status, currentPhase, len(phases)))

	for _, p := range phases {
		// Determine phase emoji
		var emoji string
		if p.Number < currentPhase {
			emoji = "\u2705" // checkmark
		} else if p.Number == currentPhase {
			emoji = "\u25B6\uFE0F" // play button
		} else {
			emoji = "\u23F3" // hourglass
		}

		sb.WriteString(fmt.Sprintf("%s Phase %d: %s\n", emoji, p.Number, p.Title))

		// Show steps for current and completed phases
		if p.Number <= currentPhase {
			for _, s := range p.Steps {
				if s.Done {
					sb.WriteString("  \u2611 " + s.Description + "\n")
				} else {
					sb.WriteString("  \u2610 " + s.Description + "\n")
				}
			}
		}
	}

	commandsContent := ms.extractCommandsSection(content)
	if commandsContent != "" {
		sb.WriteString("\nCommands:\n")
		for _, line := range strings.Split(commandsContent, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				sb.WriteString("  " + line + "\n")
			}
		}
	}

	contextContent := ms.extractContextSection(content)
	if contextContent != "" {
		sb.WriteString("\nContext: " + contextContent + "\n")
	}

	return sb.String()
}

// ---------- GetMemoryContext (plan-aware) ----------

// GetMemoryContext returns formatted memory context for the agent prompt.
// When an active plan exists, it uses selective injection based on plan status.
// During interviewing: full seed + interview guide.
// During executing: current phase only with compressed completed phases.
// When plan is active, daily notes injection is suppressed to save context.
func (ms *MemoryStore) GetMemoryContext() string {
	var parts []string

	longTerm := ms.ReadLongTerm()
	hasActivePlan := longTerm != "" && reActivePlan.MatchString(longTerm)

	if longTerm != "" {
		if hasActivePlan {
			var status string
			if m := reStatus.FindStringSubmatch(longTerm); len(m) >= 2 {
				status = strings.TrimSpace(m[1])
			}
			switch status {
			case "interviewing":
				parts = append(parts, ms.getInterviewContextFrom(longTerm))
			case "review":
				parts = append(parts, ms.getReviewContextFrom(longTerm))
			default:
				parts = append(parts, ms.getPlanContextFrom(longTerm))
			}
		} else {
			parts = append(parts, "## Long-term Memory\n\n"+longTerm)
		}
	}

	// Suppress daily notes when a plan is active to save context
	if !hasActivePlan {
		recentNotes := ms.GetRecentDailyNotes(3)
		if recentNotes != "" {
			parts = append(parts, "## Recent Daily Notes\n\n"+recentNotes)
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n\n---\n\n")
}
