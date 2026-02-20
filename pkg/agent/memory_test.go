package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestMemoryStore(t *testing.T) (*MemoryStore, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	ms := NewMemoryStore(tmpDir)
	return ms, func() { os.RemoveAll(tmpDir) }
}

const testPlanInterviewing = `# Active Plan

> Task: Set up server monitoring
> Status: interviewing
> Phase: 1
`

const testPlanExecuting = `# Active Plan

> Task: Set up server monitoring
> Status: executing
> Phase: 2

## Phase 1: Prometheus Install
- [x] Install Prometheus
- [x] Configure node_exporter

## Phase 2: Grafana Setup
- [ ] Install Grafana
- [ ] Create dashboard

## Phase 3: Alert Configuration
- [ ] Set up alert rules
- [ ] Configure Telegram notifications

## Commands
build: go build ./...
test: go test ./pkg/... -count=1
lint: golangci-lint run

## Context
Pi: Debian Bookworm arm64, ports: 3000/9090
`

const testPlanPhase1Complete = `# Active Plan

> Task: Set up server monitoring
> Status: executing
> Phase: 1

## Phase 1: Prometheus Install
- [x] Install Prometheus
- [x] Configure node_exporter

## Phase 2: Grafana Setup
- [ ] Install Grafana
- [ ] Create dashboard

## Context
Pi: Debian Bookworm arm64
`

const testPlanAllComplete = `# Active Plan

> Task: Set up server monitoring
> Status: executing
> Phase: 2

## Phase 1: Prometheus Install
- [x] Install Prometheus
- [x] Configure node_exporter

## Phase 2: Grafana Setup
- [x] Install Grafana
- [x] Create dashboard

## Context
Pi: Debian Bookworm arm64
`

func TestHasActivePlan(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	// No plan
	if ms.HasActivePlan() {
		t.Error("expected no active plan for empty memory")
	}

	// With regular content
	ms.WriteLongTerm("Some random notes")
	if ms.HasActivePlan() {
		t.Error("expected no active plan for regular content")
	}

	// With active plan
	ms.WriteLongTerm(testPlanExecuting)
	if !ms.HasActivePlan() {
		t.Error("expected active plan to be detected")
	}
}

func TestGetPlanStatus(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	// No plan
	if status := ms.GetPlanStatus(); status != "" {
		t.Errorf("expected empty status, got %q", status)
	}

	// Interviewing
	ms.WriteLongTerm(testPlanInterviewing)
	if status := ms.GetPlanStatus(); status != "interviewing" {
		t.Errorf("expected 'interviewing', got %q", status)
	}

	// Executing
	ms.WriteLongTerm(testPlanExecuting)
	if status := ms.GetPlanStatus(); status != "executing" {
		t.Errorf("expected 'executing', got %q", status)
	}
}

func TestGetCurrentPhase(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	// No plan
	if phase := ms.GetCurrentPhase(); phase != 0 {
		t.Errorf("expected phase 0, got %d", phase)
	}

	// Phase 2
	ms.WriteLongTerm(testPlanExecuting)
	if phase := ms.GetCurrentPhase(); phase != 2 {
		t.Errorf("expected phase 2, got %d", phase)
	}
}

func TestGetTotalPhases(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	// No plan
	if total := ms.GetTotalPhases(); total != 0 {
		t.Errorf("expected 0 phases, got %d", total)
	}

	// 3 phases
	ms.WriteLongTerm(testPlanExecuting)
	if total := ms.GetTotalPhases(); total != 3 {
		t.Errorf("expected 3 phases, got %d", total)
	}
}

func TestIsPlanComplete(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	// Not complete
	ms.WriteLongTerm(testPlanExecuting)
	if ms.IsPlanComplete() {
		t.Error("expected plan to be incomplete")
	}

	// All complete
	ms.WriteLongTerm(testPlanAllComplete)
	if !ms.IsPlanComplete() {
		t.Error("expected plan to be complete")
	}

	// No plan
	ms.ClearLongTerm()
	if ms.IsPlanComplete() {
		t.Error("expected false when no plan exists")
	}
}

func TestIsCurrentPhaseComplete(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	// Phase 2 not complete
	ms.WriteLongTerm(testPlanExecuting)
	if ms.IsCurrentPhaseComplete() {
		t.Error("expected current phase to be incomplete")
	}

	// Phase 1 complete (current=1)
	ms.WriteLongTerm(testPlanPhase1Complete)
	if !ms.IsCurrentPhaseComplete() {
		t.Error("expected phase 1 to be complete")
	}
}

func TestSetStatus(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	ms.WriteLongTerm(testPlanInterviewing)
	if err := ms.SetStatus("executing"); err != nil {
		t.Fatalf("SetStatus failed: %v", err)
	}
	if status := ms.GetPlanStatus(); status != "executing" {
		t.Errorf("expected 'executing', got %q", status)
	}
}

func TestAdvancePhase(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	ms.WriteLongTerm(testPlanPhase1Complete)
	if err := ms.AdvancePhase(); err != nil {
		t.Fatalf("AdvancePhase failed: %v", err)
	}
	if phase := ms.GetCurrentPhase(); phase != 2 {
		t.Errorf("expected phase 2 after advance, got %d", phase)
	}
}

func TestMarkStep(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	ms.WriteLongTerm(testPlanExecuting)

	// Mark step 1 in phase 2
	if err := ms.MarkStep(2, 1); err != nil {
		t.Fatalf("MarkStep failed: %v", err)
	}

	content := ms.ReadLongTerm()
	// Phase 2 should have first step checked
	lines := strings.Split(content, "\n")
	foundChecked := false
	inPhase2 := false
	for _, line := range lines {
		if strings.HasPrefix(line, "## Phase 2:") {
			inPhase2 = true
			continue
		}
		if inPhase2 && strings.HasPrefix(line, "## ") {
			break
		}
		if inPhase2 && strings.HasPrefix(line, "- [x] Install Grafana") {
			foundChecked = true
		}
	}
	if !foundChecked {
		t.Error("expected 'Install Grafana' to be marked [x]")
	}

	// Error case: invalid step
	if err := ms.MarkStep(2, 99); err == nil {
		t.Error("expected error for invalid step number")
	}
}

func TestAddStep(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	ms.WriteLongTerm(testPlanExecuting)

	// Add step to phase 2
	if err := ms.AddStep(2, "Test dashboard"); err != nil {
		t.Fatalf("AddStep failed: %v", err)
	}

	content := ms.ReadLongTerm()
	if !strings.Contains(content, "- [ ] Test dashboard") {
		t.Error("expected new step to be added")
	}

	// Verify it's in the right place (before Phase 3)
	idx := strings.Index(content, "- [ ] Test dashboard")
	phase3Idx := strings.Index(content, "## Phase 3:")
	if idx > phase3Idx {
		t.Error("expected new step to be before Phase 3")
	}
}

func TestClearLongTerm(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	ms.WriteLongTerm(testPlanExecuting)
	if err := ms.ClearLongTerm(); err != nil {
		t.Fatalf("ClearLongTerm failed: %v", err)
	}
	if content := ms.ReadLongTerm(); content != "" {
		t.Errorf("expected empty memory after clear, got %q", content)
	}

	// Clearing again should not error
	if err := ms.ClearLongTerm(); err != nil {
		t.Fatalf("ClearLongTerm (idempotent) failed: %v", err)
	}
}

func TestGetInterviewContext(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	ms.WriteLongTerm(testPlanInterviewing)
	ctx := ms.GetInterviewContext()

	if !strings.Contains(ctx, "Active Plan (interviewing)") {
		t.Error("expected 'Active Plan (interviewing)' header")
	}
	if !strings.Contains(ctx, "Interview Guide") {
		t.Error("expected 'Interview Guide' section")
	}
	if !strings.Contains(ctx, "Target Format") {
		t.Error("expected 'Target Format' section")
	}
	if !strings.Contains(ctx, "Set up server monitoring") {
		t.Error("expected task description in context")
	}
	// Should guide AI to ask about tooling
	if !strings.Contains(ctx, "test framework") || !strings.Contains(ctx, "linter") {
		t.Error("expected interview guide to mention test framework and linter")
	}
	// Target format should include Commands section example
	if !strings.Contains(ctx, "## Commands") {
		t.Error("expected target format to include ## Commands section")
	}
	if !strings.Contains(ctx, "go test") {
		t.Error("expected target format Commands to include test command example")
	}
	if !strings.Contains(ctx, "golangci-lint") {
		t.Error("expected target format Commands to include lint command example")
	}
}

func TestGetPlanContext(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	ms.WriteLongTerm(testPlanExecuting)
	ctx := ms.GetPlanContext()

	// Should have task summary
	if !strings.Contains(ctx, "Phase 2/3") {
		t.Error("expected 'Phase 2/3' in plan context")
	}

	// Completed phase should be summarized
	if !strings.Contains(ctx, "Done: Phase 1") {
		t.Error("expected completed phase summary")
	}

	// Current phase should have full detail
	if !strings.Contains(ctx, "Current: Phase 2") {
		t.Error("expected current phase detail")
	}
	if !strings.Contains(ctx, "Install Grafana") {
		t.Error("expected current phase steps")
	}

	// Future phases should NOT appear
	if strings.Contains(ctx, "Phase 3") {
		t.Error("expected future phases to be omitted")
	}

	// Commands should be included
	if !strings.Contains(ctx, "### Commands") {
		t.Error("expected Commands section in plan context")
	}
	if !strings.Contains(ctx, "go test") {
		t.Error("expected test command in Commands section")
	}
	if !strings.Contains(ctx, "golangci-lint") {
		t.Error("expected lint command in Commands section")
	}

	// Context should be included
	if !strings.Contains(ctx, "Debian Bookworm") {
		t.Error("expected Context section")
	}
}

func TestGetMemoryContext_PlanActive_SuppressesDailyNotes(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	// Write a daily note
	ms.AppendToday("Today's note")

	// Without plan, daily notes should appear
	ctx := ms.GetMemoryContext()
	if !strings.Contains(ctx, "Recent Daily Notes") {
		t.Error("expected daily notes when no plan active")
	}

	// With plan, daily notes should be suppressed
	ms.WriteLongTerm(testPlanExecuting)
	ctx = ms.GetMemoryContext()
	if strings.Contains(ctx, "Recent Daily Notes") {
		t.Error("expected daily notes to be suppressed when plan is active")
	}
}

func TestGetMemoryContext_InterviewingMode(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	ms.WriteLongTerm(testPlanInterviewing)
	ctx := ms.GetMemoryContext()

	if !strings.Contains(ctx, "interviewing") {
		t.Error("expected interviewing context")
	}
	if !strings.Contains(ctx, "Interview Guide") {
		t.Error("expected interview guide in context")
	}
}

func TestGetMemoryContext_ExecutingMode(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	ms.WriteLongTerm(testPlanExecuting)
	ctx := ms.GetMemoryContext()

	if !strings.Contains(ctx, "Active Plan") {
		t.Error("expected active plan in context")
	}
	if !strings.Contains(ctx, "Current: Phase 2") {
		t.Error("expected current phase in context")
	}
}

func TestGetMemoryContext_RegularMemory(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	ms.WriteLongTerm("Some notes about projects")
	ctx := ms.GetMemoryContext()

	if !strings.Contains(ctx, "Long-term Memory") {
		t.Error("expected regular long-term memory section")
	}
	if !strings.Contains(ctx, "Some notes about projects") {
		t.Error("expected memory content")
	}
}

func TestBuildInterviewSeed(t *testing.T) {
	seed := BuildInterviewSeed("Deploy monitoring stack")

	if !strings.Contains(seed, "# Active Plan") {
		t.Error("expected '# Active Plan' header")
	}
	if !strings.Contains(seed, "Deploy monitoring stack") {
		t.Error("expected task description")
	}
	if !strings.Contains(seed, "interviewing") {
		t.Error("expected interviewing status")
	}
	if !strings.Contains(seed, "> Phase: 1") {
		t.Error("expected Phase: 1")
	}
}

func TestFormatPlanDisplay(t *testing.T) {
	ms, cleanup := newTestMemoryStore(t)
	defer cleanup()

	// No plan
	display := ms.FormatPlanDisplay()
	if display != "No active plan." {
		t.Errorf("expected 'No active plan.', got %q", display)
	}

	// With plan
	ms.WriteLongTerm(testPlanExecuting)
	display = ms.FormatPlanDisplay()

	if !strings.Contains(display, "Set up server monitoring") {
		t.Error("expected task name in display")
	}
	if !strings.Contains(display, "Phase 2/3") {
		t.Error("expected phase count in display")
	}
	// Commands section should be visible
	if !strings.Contains(display, "Commands:") {
		t.Error("expected Commands section in display")
	}
	if !strings.Contains(display, "go test") {
		t.Error("expected test command in display")
	}
}

func TestMemoryStoreCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ms := NewMemoryStore(tmpDir)

	// Verify memory directory was created
	memoryDir := filepath.Join(tmpDir, "memory")
	if _, err := os.Stat(memoryDir); os.IsNotExist(err) {
		t.Error("expected memory directory to be created")
	}

	// Verify memory file path
	expectedFile := filepath.Join(memoryDir, "MEMORY.md")
	if ms.memoryFile != expectedFile {
		t.Errorf("expected memory file %q, got %q", expectedFile, ms.memoryFile)
	}
}
