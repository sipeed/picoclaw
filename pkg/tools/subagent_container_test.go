package tools

import (
	"testing"
)

func TestIsDeliberatePreset(t *testing.T) {
	tests := []struct {
		preset Preset
		want   bool
	}{
		{PresetScout, false},
		{PresetAnalyst, false},
		{PresetCoder, true},
		{PresetWorker, true},
		{PresetCoordinator, true},
		{"unknown", false},
	}
	for _, tt := range tests {
		got := isDeliberatePreset(tt.preset)
		if got != tt.want {
			t.Errorf("isDeliberatePreset(%q) = %v, want %v", tt.preset, got, tt.want)
		}
	}
}

func TestContainerMessageChannels(t *testing.T) {
	// Simulate channel creation for a deliberate preset task.
	task := &SubagentTask{
		ID:    "subagent-1",
		inCh:  make(chan string, 1),
		outCh: make(chan ContainerMessage, 4),
	}

	// Subagent sends a question.
	task.outCh <- ContainerMessage{
		Type:    "question",
		Content: "Which DB schema?",
		TaskID:  task.ID,
	}

	// Drain pending messages.
	var msgs []ContainerMessage
	for {
		select {
		case msg := <-task.outCh:
			msgs = append(msgs, msg)
		default:
			goto done
		}
	}
done:
	if len(msgs) != 1 {
		t.Fatalf("msgs count = %d, want 1", len(msgs))
	}
	if msgs[0].Type != "question" {
		t.Errorf("Type = %q, want %q", msgs[0].Type, "question")
	}
	if msgs[0].Content != "Which DB schema?" {
		t.Errorf("Content = %q, want %q", msgs[0].Content, "Which DB schema?")
	}

	// Conductor answers.
	task.inCh <- "Use PostgreSQL"
	answer := <-task.inCh
	if answer != "Use PostgreSQL" {
		t.Errorf("answer = %q, want %q", answer, "Use PostgreSQL")
	}
}

func TestPendingQuestionsAndAnswerQuestion(t *testing.T) {
	mgr := &SubagentManager{
		tasks: map[string]*SubagentTask{
			"subagent-1": {
				ID:    "subagent-1",
				outCh: make(chan ContainerMessage, 4),
				inCh:  make(chan string, 1),
			},
			"subagent-2": {
				ID: "subagent-2",
				// No channels — exploratory preset.
			},
		},
	}

	// Send question from subagent-1.
	mgr.tasks["subagent-1"].outCh <- ContainerMessage{
		Type:    "question",
		Content: "What port?",
		TaskID:  "subagent-1",
	}

	msgs := mgr.PendingQuestions()
	if len(msgs) != 1 {
		t.Fatalf("pending count = %d, want 1", len(msgs))
	}
	if msgs[0].TaskID != "subagent-1" {
		t.Errorf("TaskID = %q, want %q", msgs[0].TaskID, "subagent-1")
	}

	// Second call should return empty (already drained).
	msgs2 := mgr.PendingQuestions()
	if len(msgs2) != 0 {
		t.Errorf("second pending count = %d, want 0", len(msgs2))
	}

	// Answer the question.
	if err := mgr.AnswerQuestion("subagent-1", "8080"); err != nil {
		t.Fatalf("AnswerQuestion: %v", err)
	}
	answer := <-mgr.tasks["subagent-1"].inCh
	if answer != "8080" {
		t.Errorf("answer = %q, want %q", answer, "8080")
	}

	// Answer non-existent task.
	if err := mgr.AnswerQuestion("subagent-99", "x"); err == nil {
		t.Error("expected error for non-existent task")
	}

	// Answer task without channels.
	if err := mgr.AnswerQuestion("subagent-2", "x"); err == nil {
		t.Error("expected error for task without escalation channel")
	}
}
