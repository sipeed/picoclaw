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
	content := ms.ReadLongTerm()
	if !reActivePlan.MatchString(content) {
		return false
	}
	// Must have at least one step
	if !reStepDone.MatchString(content) && !reStepTodo.MatchString(content) {
		return false
	}
	// No unchecked steps
	return !reStepTodo.MatchString(content)
}

// IsCurrentPhaseComplete returns true if all steps in the current phase are [x].
func (ms *MemoryStore) IsCurrentPhaseComplete() bool {
	content := ms.ReadLongTerm()
	phase := ms.GetCurrentPhase()
	if phase == 0 {
		return false
	}
	phaseContent := ms.extractPhaseContent(content, phase)
	if phaseContent == "" {
		return false
	}
	// Must have at least one step
	if !reStepDone.MatchString(phaseContent) && !reStepTodo.MatchString(phaseContent) {
		return false
	}
	return !reStepTodo.MatchString(phaseContent)
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

// ---------- Selective injection methods ----------

// interviewSeed is the initial content written to MEMORY.md when /plan starts.
const interviewSeedTemplate = `# Active Plan

> Task: %s
> Status: interviewing
> Phase: 1
`

// BuildInterviewSeed creates the initial plan seed for a given task description.
func BuildInterviewSeed(task string) string {
	return fmt.Sprintf(interviewSeedTemplate, task)
}

// GetInterviewContext returns context for injection during the interviewing phase.
// Includes the full seed + interview guide + target format template.
func (ms *MemoryStore) GetInterviewContext() string {
	content := ms.ReadLongTerm()

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
	sb.WriteString("- After each answer, use edit_file to append findings to the ## Context section of memory/MEMORY.md.\n")
	sb.WriteString("- When you have enough information, use edit_file to write ## Phase, ## Commands, and ## Context sections into memory/MEMORY.md.\n")
	sb.WriteString("- Organize into 2-5 phases with 3-5 steps each.\n")
	sb.WriteString("- After writing Phases, set Status to executing. The system will handle the rest.\n")
	sb.WriteString("\n### Target Format\n")
	sb.WriteString("```\n")
	sb.WriteString("## Phase 1: <title>\n")
	sb.WriteString("- [ ] Step\n")
	sb.WriteString("## Phase 2: <title>\n")
	sb.WriteString("- [ ] Step\n")
	sb.WriteString("## Commands\n")
	sb.WriteString("build: go build ./...\n")
	sb.WriteString("test: go test ./pkg/... -count=1\n")
	sb.WriteString("lint: golangci-lint run\n")
	sb.WriteString("## Context\n")
	sb.WriteString("<collected requirements, decisions, environment>\n")
	sb.WriteString("```\n")
	return sb.String()
}

// GetReviewContext returns context for injection during the review phase.
// Shows the full plan and instructs the AI to wait for user approval.
func (ms *MemoryStore) GetReviewContext() string {
	content := ms.ReadLongTerm()

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
	content := ms.ReadLongTerm()
	currentPhase := ms.GetCurrentPhase()
	totalPhases := ms.GetTotalPhases()

	// Extract task description
	taskLine := ""
	if m := regexp.MustCompile(`(?m)^> Task:\s*(.+)`).FindStringSubmatch(content); len(m) >= 2 {
		taskLine = strings.TrimSpace(m[1])
	}

	var sb strings.Builder
	sb.WriteString("## Active Plan\n")
	sb.WriteString(fmt.Sprintf("Task: %s | Phase %d/%d\n", taskLine, currentPhase, totalPhases))

	// Completed phases: one-line summaries
	for p := 1; p < currentPhase; p++ {
		title := ms.getPhaseTitle(content, p)
		sb.WriteString(fmt.Sprintf("Done: Phase %d (%s)\n", p, title))
	}

	// Current phase: full detail
	if currentPhase > 0 {
		title := ms.getPhaseTitle(content, currentPhase)
		sb.WriteString(fmt.Sprintf("### Current: Phase %d — %s\n", currentPhase, title))
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
	if !ms.HasActivePlan() {
		return "No active plan."
	}

	taskLine := ""
	if m := regexp.MustCompile(`(?m)^> Task:\s*(.+)`).FindStringSubmatch(content); len(m) >= 2 {
		taskLine = strings.TrimSpace(m[1])
	}
	status := ms.GetPlanStatus()
	currentPhase := ms.GetCurrentPhase()
	totalPhases := ms.GetTotalPhases()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Plan: %s\n", taskLine))
	sb.WriteString(fmt.Sprintf("Status: %s | Phase %d/%d\n\n", status, currentPhase, totalPhases))

	for p := 1; p <= totalPhases; p++ {
		title := ms.getPhaseTitle(content, p)
		phaseContent := ms.extractPhaseContent(content, p)

		// Determine phase emoji
		var emoji string
		if p < currentPhase {
			emoji = "\u2705" // checkmark
		} else if p == currentPhase {
			emoji = "\u25B6\uFE0F" // play button
		} else {
			emoji = "\u23F3" // hourglass
		}

		sb.WriteString(fmt.Sprintf("%s Phase %d: %s\n", emoji, p, title))

		// Show steps for current and completed phases
		if p <= currentPhase {
			lines := strings.Split(phaseContent, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "- [x] ") {
					sb.WriteString("  \u2611 " + line[6:] + "\n")
				} else if strings.HasPrefix(line, "- [ ] ") {
					sb.WriteString("  \u2610 " + line[6:] + "\n")
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
	if longTerm != "" {
		if ms.HasActivePlan() {
			status := ms.GetPlanStatus()
			switch status {
			case "interviewing":
				parts = append(parts, ms.GetInterviewContext())
			case "review":
				parts = append(parts, ms.GetReviewContext())
			default:
				parts = append(parts, ms.GetPlanContext())
			}
		} else {
			parts = append(parts, "## Long-term Memory\n\n"+longTerm)
		}
	}

	// Suppress daily notes when a plan is active to save context
	if !ms.HasActivePlan() {
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
