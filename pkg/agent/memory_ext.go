package agent

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Cache types for MemoryStore.

type longTermFileCache struct {
	loaded  bool
	exists  bool
	modTime time.Time
	size    int64
	content string
}

type parsedPlanStateCache struct {
	loaded        bool
	sourceContent string
	state         parsedPlanState
}

type parsedPlanState struct {
	content       string
	hasActivePlan bool
	status        string
	currentPhase  int
	totalPhases   int
	workDir       string
	taskName      string
	phases        []PlanPhase
}

// InvalidateCache clears all in-memory caches for MEMORY.md content and parsed plan state.
func (ms *MemoryStore) InvalidateCache() {
	ms.cacheMu.Lock()
	defer ms.cacheMu.Unlock()
	ms.longTermCache = longTermFileCache{}
	ms.parsedPlanCache = parsedPlanStateCache{}
}

func (ms *MemoryStore) readLongTermCached() string {
	info, err := os.Stat(ms.memoryFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return ""
		}
		ms.cacheMu.RLock()
		cachedMissing := ms.longTermCache.loaded && !ms.longTermCache.exists
		ms.cacheMu.RUnlock()
		if cachedMissing {
			return ""
		}
		ms.cacheMu.Lock()
		ms.longTermCache = longTermFileCache{loaded: true, exists: false}
		ms.parsedPlanCache = parsedPlanStateCache{}
		ms.cacheMu.Unlock()
		return ""
	}

	modTime := info.ModTime()
	size := info.Size()

	ms.cacheMu.RLock()
	if ms.longTermCache.loaded &&
		ms.longTermCache.exists &&
		ms.longTermCache.modTime.Equal(modTime) &&
		ms.longTermCache.size == size {
		content := ms.longTermCache.content
		ms.cacheMu.RUnlock()
		return content
	}
	ms.cacheMu.RUnlock()

	data, err := os.ReadFile(ms.memoryFile)
	if err != nil {
		if os.IsNotExist(err) {
			ms.cacheMu.Lock()
			ms.longTermCache = longTermFileCache{loaded: true, exists: false}
			ms.parsedPlanCache = parsedPlanStateCache{}
			ms.cacheMu.Unlock()
		}
		return ""
	}

	content := string(data)
	ms.cacheMu.Lock()
	ms.longTermCache = longTermFileCache{
		loaded:  true,
		exists:  true,
		modTime: modTime,
		size:    size,
		content: content,
	}
	if ms.parsedPlanCache.loaded && ms.parsedPlanCache.sourceContent != content {
		ms.parsedPlanCache = parsedPlanStateCache{}
	}
	ms.cacheMu.Unlock()
	return content
}

func (ms *MemoryStore) getParsedPlanState() parsedPlanState {
	content := ms.ReadLongTerm()

	ms.cacheMu.RLock()
	if ms.parsedPlanCache.loaded && ms.parsedPlanCache.sourceContent == content {
		state := ms.parsedPlanCache.state
		ms.cacheMu.RUnlock()
		return state
	}
	ms.cacheMu.RUnlock()

	state := ms.parsePlanState(content)

	ms.cacheMu.Lock()
	if !ms.parsedPlanCache.loaded || ms.parsedPlanCache.sourceContent != content {
		ms.parsedPlanCache = parsedPlanStateCache{
			loaded:        true,
			sourceContent: content,
			state:         state,
		}
	} else {
		state = ms.parsedPlanCache.state
	}
	ms.cacheMu.Unlock()
	return state
}

func (ms *MemoryStore) parsePlanState(content string) parsedPlanState {
	state := parsedPlanState{content: content}
	if content == "" || !reActivePlan.MatchString(content) {
		return state
	}
	state.hasActivePlan = true
	if m := reStatus.FindStringSubmatch(content); len(m) >= 2 {
		state.status = strings.TrimSpace(m[1])
	}
	if m := rePhase.FindStringSubmatch(content); len(m) >= 2 {
		state.currentPhase, _ = strconv.Atoi(m[1])
	}
	state.totalPhases = maxPhaseNumber(content)
	if m := reWorkDir.FindStringSubmatch(content); len(m) >= 2 {
		state.workDir = strings.TrimSpace(m[1])
	}
	if m := reTaskLine.FindStringSubmatch(content); len(m) >= 2 {
		state.taskName = strings.TrimSpace(m[1])
	}
	state.phases = ms.getPlanPhasesFrom(content)
	return state
}

func clonePlanPhases(phases []PlanPhase) []PlanPhase {
	if len(phases) == 0 {
		return nil
	}
	result := make([]PlanPhase, 0, len(phases))
	for _, p := range phases {
		phase := PlanPhase{Number: p.Number, Title: p.Title}
		if len(p.Steps) > 0 {
			phase.Steps = append([]PlanStep(nil), p.Steps...)
		}
		result = append(result, phase)
	}
	return result
}

// ClearLongTerm removes the long-term memory file.
func (ms *MemoryStore) ClearLongTerm() error {
	if err := os.Remove(ms.memoryFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	ms.InvalidateCache()
	return nil
}

// ---------- Plan state query methods ----------

var (
	reActivePlan = regexp.MustCompile(`(?m)^# Active Plan`)
	reStatus     = regexp.MustCompile(`(?m)^> Status:\s*(.+)`)
	rePhase      = regexp.MustCompile(`(?m)^> Phase:\s*(\d+)`)
	rePhaseHeader = regexp.MustCompile(`(?m)^## Phase (\d+):\s*(.*)`)
	reWorkDir    = regexp.MustCompile(`(?m)^> WorkDir:\s*(.+)`)
	reTaskLine   = regexp.MustCompile(`(?m)^> Task:\s*(.+)`)
)

// HasActivePlan returns true if MEMORY.md contains an active plan.
func (ms *MemoryStore) HasActivePlan() bool {
	return ms.getParsedPlanState().hasActivePlan
}

// GetPlanStatus returns the plan status: "interviewing", "executing", or "".
func (ms *MemoryStore) GetPlanStatus() string {
	return ms.getParsedPlanState().status
}

// GetCurrentPhase returns the current phase number from "> Phase: N".
func (ms *MemoryStore) GetCurrentPhase() int {
	return ms.getParsedPlanState().currentPhase
}

// GetTotalPhases returns the total number of phases (max ## Phase N).
func (ms *MemoryStore) GetTotalPhases() int {
	return ms.getParsedPlanState().totalPhases
}

// IsPlanComplete returns true if all steps in all phases are [x].
func (ms *MemoryStore) IsPlanComplete() bool {
	phases := ms.getParsedPlanState().phases
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
	state := ms.getParsedPlanState()
	if state.currentPhase == 0 {
		return false
	}
	for _, p := range state.phases {
		if p.Number == state.currentPhase {
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
	return clonePlanPhases(ms.getParsedPlanState().phases)
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
				steps = append(steps, PlanStep{Index: stepIdx, Description: line[6:], Done: true})
			} else if strings.HasPrefix(line, "- [ ] ") {
				stepIdx++
				steps = append(steps, PlanStep{Index: stepIdx, Description: line[6:], Done: false})
			}
		}
		phases = append(phases, PlanPhase{Number: p, Title: title, Steps: steps})
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
			if strings.HasPrefix(line, "- [") {
				insertIdx = i + 1
			}
		}
	}
	if insertIdx < 0 {
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
// transitioning out of the interview phase.
func (ms *MemoryStore) ValidatePlanStructure() error {
	content := ms.ReadLongTerm()
	if !reActivePlan.MatchString(content) {
		return fmt.Errorf("missing '# Active Plan' header")
	}
	if !reStatus.MatchString(content) {
		return fmt.Errorf("missing '> Status:' line")
	}
	if !rePhase.MatchString(content) {
		return fmt.Errorf("missing '> Phase:' line")
	}
	phases := ms.getPlanPhasesFrom(content)
	if len(phases) == 0 {
		return fmt.Errorf("no '## Phase N:' sections found")
	}
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
	return ms.getParsedPlanState().workDir
}

// GetPlanTaskName returns the task description from the plan metadata, or "".
func (ms *MemoryStore) GetPlanTaskName() string {
	return ms.getParsedPlanState().taskName
}

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
	sb.WriteString("\n### Target Format (MANDATORY — system parses this exact structure)\n\n")
	sb.WriteString("# Active Plan\n")
	sb.WriteString("> Task: <description>\n")
	sb.WriteString("> WorkDir: <path>\n")
	sb.WriteString("> Status: interviewing\n")
	sb.WriteString("> Phase: 1\n\n")
	sb.WriteString("## Phase 1: <title>\n")
	sb.WriteString("- [ ] Step description\n")
	sb.WriteString("- [ ] Step description\n\n")
	sb.WriteString("## Phase 2: <title>\n")
	sb.WriteString("- [ ] Step description\n")
	sb.WriteString("- [ ] Step description\n\n")
	sb.WriteString("## Commands\n")
	sb.WriteString("build: <project-specific build command>\n")
	sb.WriteString("test: <project-specific test command>\n")
	sb.WriteString("lint: <project-specific lint command>\n\n")
	sb.WriteString("## Context\n")
	sb.WriteString("<collected requirements, decisions, environment>\n")
	return sb.String()
}

// GetReviewContext returns context for injection during the review phase.
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

	for p := 1; p < currentPhase; p++ {
		title := ms.getPhaseTitle(content, p)
		fmt.Fprintf(&sb, "Done: Phase %d (%s)\n", p, title)
	}

	if currentPhase > 0 {
		title := ms.getPhaseTitle(content, currentPhase)
		fmt.Fprintf(&sb, "### Current: Phase %d — %s\n", currentPhase, title)
		phaseContent := ms.extractPhaseContent(content, currentPhase)
		sb.WriteString(strings.TrimSpace(phaseContent))
		sb.WriteString("\n")
	}

	if commandsContent := ms.extractCommandsSection(content); commandsContent != "" {
		sb.WriteString("### Commands\n")
		sb.WriteString(commandsContent)
		sb.WriteString("\n")
	}

	if contextContent := ms.extractContextSection(content); contextContent != "" {
		sb.WriteString("### Context\n")
		sb.WriteString(contextContent)
		sb.WriteString("\n")
	}

	if orchContent := ms.extractSection(content, "Orchestration"); orchContent != "" {
		sb.WriteString("### Orchestration\n")
		sb.WriteString(orchContent)
		sb.WriteString("\n")
	}

	return sb.String()
}

func maxPhaseNumber(content string) int {
	matches := rePhaseHeader.FindAllStringSubmatch(content, -1)
	maxN := 0
	for _, m := range matches {
		if len(m) >= 2 {
			n, _ := strconv.Atoi(m[1])
			if n > maxN {
				maxN = n
			}
		}
	}
	return maxN
}

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

func (ms *MemoryStore) extractContextSection(content string) string {
	return ms.extractSection(content, "Context")
}

func (ms *MemoryStore) extractCommandsSection(content string) string {
	return ms.extractSection(content, "Commands")
}

// FormatPlanDisplay returns a user-facing display of the full plan with emoji indicators.
func (ms *MemoryStore) FormatPlanDisplay() string {
	state := ms.getParsedPlanState()
	if !state.hasActivePlan {
		return "No active plan."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Plan: %s\n", state.taskName))
	sb.WriteString(fmt.Sprintf("Status: %s | Phase %d/%d\n\n", state.status, state.currentPhase, len(state.phases)))

	for _, p := range state.phases {
		var emoji string
		if p.Number < state.currentPhase {
			emoji = "\u2705"
		} else if p.Number == state.currentPhase {
			emoji = "\u25B6\uFE0F"
		} else {
			emoji = "\u23F3"
		}
		sb.WriteString(fmt.Sprintf("%s Phase %d: %s\n", emoji, p.Number, p.Title))
		if p.Number <= state.currentPhase {
			for _, s := range p.Steps {
				if s.Done {
					sb.WriteString("  \u2611 " + s.Description + "\n")
				} else {
					sb.WriteString("  \u2610 " + s.Description + "\n")
				}
			}
		}
	}

	if commandsContent := ms.extractCommandsSection(state.content); commandsContent != "" {
		sb.WriteString("\nCommands:\n")
		for _, line := range strings.Split(commandsContent, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				sb.WriteString("  " + line + "\n")
			}
		}
	}

	if contextContent := ms.extractContextSection(state.content); contextContent != "" {
		sb.WriteString("\nContext: " + contextContent + "\n")
	}

	return sb.String()
}

// ---------- GetMemoryContext (plan-aware) ----------

func (ms *MemoryStore) getMemoryContextPlanAware() string {
	var parts []string
	state := ms.getParsedPlanState()
	longTerm := state.content

	if longTerm != "" {
		if state.hasActivePlan {
			switch state.status {
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
	if !state.hasActivePlan {
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
