package tools

import (
	"testing"
)

func TestSubagentPlanStateString(t *testing.T) {
	tests := []struct {
		state SubagentPlanState
		want  string
	}{
		{PlanNone, "none"},
		{PlanClarifying, "clarifying"},
		{PlanReview, "review"},
		{PlanExecuting, "executing"},
		{PlanCompleted, "completed"},
	}
	for _, tt := range tests {
		got := tt.state.String()
		if got != tt.want {
			t.Errorf("SubagentPlanState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestFormatPlanSteps(t *testing.T) {
	steps := []string{"Read config", "Add middleware", "Write tests"}
	got := formatPlanSteps(steps)
	want := "1. Read config\n2. Add middleware\n3. Write tests\n"
	if got != want {
		t.Errorf("formatPlanSteps = %q, want %q", got, want)
	}
}

func TestClarifyingSystemPrompt(t *testing.T) {
	prompt := clarifyingSystemPrompt()
	if prompt == "" {
		t.Error("clarifyingSystemPrompt returned empty string")
	}
	// Should mention ask_conductor and submit_plan.
	for _, keyword := range []string{"ask_conductor", "submit_plan", "CLARIFYING"} {
		if !containsString(prompt, keyword) {
			t.Errorf("clarifyingSystemPrompt missing keyword %q", keyword)
		}
	}
}

func TestExecutingSystemPrompt(t *testing.T) {
	prompt := executingSystemPrompt()
	if prompt == "" {
		t.Error("executingSystemPrompt returned empty string")
	}
	if !containsString(prompt, "EXECUTING") {
		t.Error("executingSystemPrompt missing keyword EXECUTING")
	}
}

func TestExploratorySystemPrompt(t *testing.T) {
	scoutPrompt := exploratorySystemPrompt(PresetScout)
	if scoutPrompt == "" {
		t.Error("exploratorySystemPrompt(scout) returned empty")
	}
	defaultPrompt := exploratorySystemPrompt("unknown")
	if defaultPrompt == "" {
		t.Error("exploratorySystemPrompt(unknown) returned empty")
	}
	if scoutPrompt == defaultPrompt {
		t.Error("scout and unknown prompts should differ")
	}
}

func TestDeliberateTaskChannelsCreated(t *testing.T) {
	// Verify that channels and initial state are correct for deliberate presets.
	task := &SubagentTask{
		inCh:  make(chan string, 1),
		outCh: make(chan ContainerMessage, 4),
	}
	if task.inCh == nil || task.outCh == nil {
		t.Fatal("expected channels to be non-nil for deliberate task")
	}
	if task.PlanState != PlanNone {
		t.Errorf("initial PlanState = %v, want PlanNone", task.PlanState)
	}
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
