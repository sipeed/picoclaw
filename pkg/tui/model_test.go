// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTruncateSessionKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "short key unchanged",
			input: "abc",
			want:  "abc",
		},
		{
			name:  "exact 15 chars unchanged",
			input: "123456789012345",
			want:  "123456789012345",
		},
		{
			name:  "long key truncated with ellipsis",
			input: "1234567890123456",
			want:  "123456789012345…",
		},
		{
			name:  "empty key",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateSessionKey(tt.input)
			if got != tt.want {
				t.Errorf("truncateSessionKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// newTestModel creates a Model for testing without requiring a real AgentLoop.
// The model is constructed directly, bypassing NewModel. We must use
// textarea.New() because the bubbles textarea panics on SetWidth if its
// internal state is nil.
func newTestModel() Model {
	ta := textarea.New()
	ta.SetHeight(textareaHeight)
	return Model{
		textarea:    ta,
		sessionKey:  "test-session",
		modelName:   "test-model",
		messages:    []chatMessage{},
		eventBridge: newEventBridge(),
	}
}

// initTestModel creates a test model and sends a WindowSizeMsg to make it ready.
func initTestModel(t *testing.T) Model {
	t.Helper()
	m := newTestModel()
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model, ok := result.(Model)
	if !ok {
		t.Fatal("Update did not return a Model")
	}
	return model
}

func TestModel_View_NotReady(t *testing.T) {
	m := newTestModel()
	view := m.View()
	if !strings.Contains(view, "Initializing...") {
		t.Errorf("expected 'Initializing...' in view, got %q", view)
	}
}

func TestModel_View_Quitting(t *testing.T) {
	m := newTestModel()
	m.quitting = true
	view := m.View()
	if !strings.Contains(view, "Goodbye!") {
		t.Errorf("expected 'Goodbye!' in view, got %q", view)
	}
}

func TestModel_Update_CtrlC_SetsQuitting(t *testing.T) {
	m := initTestModel(t)

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model, ok := result.(Model)
	if !ok {
		t.Fatal("Update did not return a Model")
	}

	if !model.quitting {
		t.Error("expected quitting to be true after Ctrl+C")
	}
	if cmd == nil {
		t.Error("expected quit command, got nil")
	}
}

func TestModel_Update_ResponseMsg_AddsAssistantMessage(t *testing.T) {
	m := initTestModel(t)
	m.thinking = true

	result, _ := m.Update(ResponseMsg{Content: "Hello, world!"})
	model, ok := result.(Model)
	if !ok {
		t.Fatal("Update did not return a Model")
	}

	if model.thinking {
		t.Error("expected thinking to be false after ResponseMsg")
	}

	found := false
	for _, msg := range model.messages {
		if msg.role == roleAssistant && msg.content == "Hello, world!" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected assistant message 'Hello, world!' in messages")
	}
}

func TestModel_Update_ToolCallStarted_AddsToolMessage(t *testing.T) {
	m := initTestModel(t)

	result, _ := m.Update(ToolCallStartedMsg{
		ID:   "tool-1",
		Name: "web_search",
		Args: `{"query":"test"}`,
	})
	model, ok := result.(Model)
	if !ok {
		t.Fatal("Update did not return a Model")
	}

	if len(model.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(model.messages))
	}

	msg := model.messages[0]
	if msg.role != roleTool {
		t.Errorf("expected role 'tool', got %q", msg.role)
	}
	if msg.toolName != "web_search" {
		t.Errorf("expected toolName 'web_search', got %q", msg.toolName)
	}
	if msg.toolID != "tool-1" {
		t.Errorf("expected toolID 'tool-1', got %q", msg.toolID)
	}
	if msg.toolDone {
		t.Error("expected toolDone to be false")
	}
}

func TestModel_Update_ToolCallCompleted_UpdatesToolMessage(t *testing.T) {
	m := initTestModel(t)

	// First start a tool
	result, _ := m.Update(ToolCallStartedMsg{
		ID:   "tool-1",
		Name: "web_search",
	})
	m = result.(Model)

	// Then complete it
	result, _ = m.Update(ToolCallCompletedMsg{
		ID:   "tool-1",
		Name: "web_search",
	})
	model := result.(Model)

	if len(model.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(model.messages))
	}

	msg := model.messages[0]
	if !msg.toolDone {
		t.Error("expected toolDone to be true after completion")
	}
	if msg.toolErr {
		t.Error("expected toolErr to be false for successful completion")
	}
	if !strings.Contains(msg.content, "done") {
		t.Errorf("expected content to contain 'done', got %q", msg.content)
	}
}

func TestModel_Update_ToolCallCompleted_Error(t *testing.T) {
	m := initTestModel(t)

	result, _ := m.Update(ToolCallStartedMsg{
		ID:   "tool-2",
		Name: "exec",
	})
	m = result.(Model)

	result, _ = m.Update(ToolCallCompletedMsg{
		ID:      "tool-2",
		Name:    "exec",
		IsError: true,
	})
	model := result.(Model)

	msg := model.messages[0]
	if !msg.toolErr {
		t.Error("expected toolErr to be true for error completion")
	}
	if !strings.Contains(msg.content, "failed") {
		t.Errorf("expected content to contain 'failed', got %q", msg.content)
	}
}

func TestModel_Update_ErrorMsg_AddsSystemMessage(t *testing.T) {
	m := initTestModel(t)
	m.thinking = true

	result, _ := m.Update(ErrorMsg{Err: fmt.Errorf("connection refused")})
	model, ok := result.(Model)
	if !ok {
		t.Fatal("Update did not return a Model")
	}

	if model.thinking {
		t.Error("expected thinking to be false after ErrorMsg")
	}

	found := false
	for _, msg := range model.messages {
		if msg.role == roleSystem && strings.Contains(msg.content, "connection refused") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected system message containing 'connection refused'")
	}
}

func TestModel_Update_SlashCommandResult(t *testing.T) {
	m := initTestModel(t)

	result, _ := m.Update(SlashCommandResultMsg{Result: "Session: default"})
	model := result.(Model)

	found := false
	for _, msg := range model.messages {
		if msg.role == roleSystem && msg.content == "Session: default" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected system message with slash command result")
	}
}

func TestModel_Update_WindowSizeMsg_SetsReady(t *testing.T) {
	m := newTestModel()

	if m.ready {
		t.Fatal("expected model to not be ready initially")
	}

	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := result.(Model)

	if !model.ready {
		t.Error("expected model to be ready after WindowSizeMsg")
	}
	if model.width != 120 {
		t.Errorf("expected width 120, got %d", model.width)
	}
	if model.height != 40 {
		t.Errorf("expected height 40, got %d", model.height)
	}
}
