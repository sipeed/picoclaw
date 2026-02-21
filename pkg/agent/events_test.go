package agent

import (
	"fmt"
	"sync"
	"testing"
)

// mockEventListener records events for test assertions
type mockEventListener struct {
	mu     sync.Mutex
	events []AgentEvent
}

func (m *mockEventListener) OnEvent(event AgentEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockEventListener) getEvents() []AgentEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]AgentEvent, len(m.events))
	copy(copied, m.events)
	return copied
}

func TestAgentEventTypes_AreDistinct(t *testing.T) {
	types := []AgentEventType{
		EventThinkingStarted,
		EventToolCallStarted,
		EventToolCallCompleted,
		EventResponseComplete,
		EventError,
	}

	seen := make(map[AgentEventType]bool)
	for _, et := range types {
		if seen[et] {
			t.Errorf("duplicate event type value: %d", et)
		}
		seen[et] = true
	}

	if len(seen) != 5 {
		t.Errorf("expected 5 distinct event types, got %d", len(seen))
	}
}

func TestMockEventListener_ReceivesEvents(t *testing.T) {
	listener := &mockEventListener{}

	tests := []struct {
		name      string
		event     AgentEvent
		checkData func(t *testing.T, data any)
	}{
		{
			name: "thinking started",
			event: AgentEvent{
				Type: EventThinkingStarted,
				Data: nil,
			},
			checkData: func(t *testing.T, data any) {
				if data != nil {
					t.Errorf("expected nil data for ThinkingStarted, got %v", data)
				}
			},
		},
		{
			name: "tool call started",
			event: AgentEvent{
				Type: EventToolCallStarted,
				Data: ToolCallStartedData{
					ID:   "call_123",
					Name: "exec",
					Args: `{"command":"ls"}`,
				},
			},
			checkData: func(t *testing.T, data any) {
				d, ok := data.(ToolCallStartedData)
				if !ok {
					t.Fatalf("expected ToolCallStartedData, got %T", data)
				}
				if d.ID != "call_123" {
					t.Errorf("expected ID 'call_123', got %q", d.ID)
				}
				if d.Name != "exec" {
					t.Errorf("expected Name 'exec', got %q", d.Name)
				}
				if d.Args != `{"command":"ls"}` {
					t.Errorf("expected Args `{\"command\":\"ls\"}`, got %q", d.Args)
				}
			},
		},
		{
			name: "tool call completed",
			event: AgentEvent{
				Type: EventToolCallCompleted,
				Data: ToolCallCompletedData{
					ID:      "call_123",
					Name:    "exec",
					Result:  "file1.txt\nfile2.txt",
					IsError: false,
				},
			},
			checkData: func(t *testing.T, data any) {
				d, ok := data.(ToolCallCompletedData)
				if !ok {
					t.Fatalf("expected ToolCallCompletedData, got %T", data)
				}
				if d.ID != "call_123" {
					t.Errorf("expected ID 'call_123', got %q", d.ID)
				}
				if d.IsError {
					t.Error("expected IsError=false")
				}
				if d.Result != "file1.txt\nfile2.txt" {
					t.Errorf("unexpected Result: %q", d.Result)
				}
			},
		},
		{
			name: "tool call completed with error",
			event: AgentEvent{
				Type: EventToolCallCompleted,
				Data: ToolCallCompletedData{
					ID:      "call_456",
					Name:    "read_file",
					Result:  "file not found",
					IsError: true,
				},
			},
			checkData: func(t *testing.T, data any) {
				d, ok := data.(ToolCallCompletedData)
				if !ok {
					t.Fatalf("expected ToolCallCompletedData, got %T", data)
				}
				if !d.IsError {
					t.Error("expected IsError=true")
				}
			},
		},
		{
			name: "response complete",
			event: AgentEvent{
				Type: EventResponseComplete,
				Data: ResponseCompleteData{
					Content: "Here is the answer.",
				},
			},
			checkData: func(t *testing.T, data any) {
				d, ok := data.(ResponseCompleteData)
				if !ok {
					t.Fatalf("expected ResponseCompleteData, got %T", data)
				}
				if d.Content != "Here is the answer." {
					t.Errorf("expected content 'Here is the answer.', got %q", d.Content)
				}
			},
		},
		{
			name: "error",
			event: AgentEvent{
				Type: EventError,
				Data: ErrorData{
					Err: fmt.Errorf("LLM call failed"),
				},
			},
			checkData: func(t *testing.T, data any) {
				d, ok := data.(ErrorData)
				if !ok {
					t.Fatalf("expected ErrorData, got %T", data)
				}
				if d.Err == nil {
					t.Fatal("expected non-nil error")
				}
				if d.Err.Error() != "LLM call failed" {
					t.Errorf("expected error 'LLM call failed', got %q", d.Err.Error())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener.OnEvent(tt.event)
		})
	}

	// Verify all events were received in order
	events := listener.getEvents()
	if len(events) != len(tests) {
		t.Fatalf("expected %d events, got %d", len(tests), len(events))
	}

	for i, tt := range tests {
		t.Run(tt.name+"/verify", func(t *testing.T) {
			if events[i].Type != tt.event.Type {
				t.Errorf("event %d: expected type %d, got %d", i, tt.event.Type, events[i].Type)
			}
			tt.checkData(t, events[i].Data)
		})
	}
}

func TestFireEvent_WithListener(t *testing.T) {
	// Create a minimal AgentLoop with just the eventListener field set
	listener := &mockEventListener{}
	al := &AgentLoop{eventListener: listener}

	al.fireEvent(AgentEvent{Type: EventThinkingStarted})
	al.fireEvent(AgentEvent{
		Type: EventToolCallStarted,
		Data: ToolCallStartedData{ID: "1", Name: "exec", Args: `{"command":"ls"}`},
	})

	events := listener.getEvents()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != EventThinkingStarted {
		t.Errorf("event 0: expected EventThinkingStarted, got %d", events[0].Type)
	}
	if events[1].Type != EventToolCallStarted {
		t.Errorf("event 1: expected EventToolCallStarted, got %d", events[1].Type)
	}
	d, ok := events[1].Data.(ToolCallStartedData)
	if !ok {
		t.Fatalf("event 1: expected ToolCallStartedData, got %T", events[1].Data)
	}
	if d.ID != "1" || d.Name != "exec" || d.Args != `{"command":"ls"}` {
		t.Errorf("event 1: unexpected data: %+v", d)
	}
}

func TestFireEvent_WithoutListener(t *testing.T) {
	al := &AgentLoop{} // No listener set
	// Should not panic
	al.fireEvent(AgentEvent{Type: EventThinkingStarted})
}
