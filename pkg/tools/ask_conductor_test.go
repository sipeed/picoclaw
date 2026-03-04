package tools

import (
	"context"
	"testing"
	"time"
)

func TestAskConductorTool_Execute(t *testing.T) {
	outCh := make(chan ContainerMessage, 4)

	inCh := make(chan string, 1)

	tool := NewAskConductorTool("subagent-1", "conductor:main", "subagent:subagent-1", outCh, inCh, nil)

	if tool.Name() != "ask_conductor" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "ask_conductor")
	}

	// Simulate conductor answering in background.

	go func() {
		msg := <-outCh

		if msg.Type != "question" {
			t.Errorf("msg.Type = %q, want %q", msg.Type, "question")
		}

		if msg.Content != "What port?" {
			t.Errorf("msg.Content = %q, want %q", msg.Content, "What port?")
		}

		inCh <- "Use port 8080"
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	result := tool.Execute(ctx, map[string]any{"question": "What port?"})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	if result.ForUser != "Use port 8080" {
		t.Errorf("ForUser = %q, want %q", result.ForUser, "Use port 8080")
	}
}

func TestAskConductorTool_MissingQuestion(t *testing.T) {
	tool := NewAskConductorTool("subagent-1", "conductor:main", "subagent:subagent-1", nil, nil, nil)

	result := tool.Execute(context.Background(), map[string]any{})

	if !result.IsError {
		t.Error("expected error for missing question")
	}
}

func TestAskConductorTool_ContextCanceled(t *testing.T) {
	outCh := make(chan ContainerMessage) // unbuffered, will block

	inCh := make(chan string)

	tool := NewAskConductorTool("subagent-1", "conductor:main", "subagent:subagent-1", outCh, inCh, nil)

	ctx, cancel := context.WithCancel(context.Background())

	cancel() // cancel immediately

	result := tool.Execute(ctx, map[string]any{"question": "test?"})

	if !result.IsError {
		t.Error("expected error on canceled context")
	}
}
