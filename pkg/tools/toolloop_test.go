package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type flakyToolLoopProvider struct {
	errors []error
	calls  int
}

func (p *flakyToolLoopProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	p.calls++
	if p.calls <= len(p.errors) {
		return nil, p.errors[p.calls-1]
	}
	return &providers.LLMResponse{
		Content:   "ok",
		ToolCalls: []providers.ToolCall{},
	}, nil
}

func (p *flakyToolLoopProvider) GetDefaultModel() string {
	return "mock-toolloop-model"
}

func TestRunToolLoop_TransientRetry(t *testing.T) {
	provider := &flakyToolLoopProvider{
		errors: []error{
			fmt.Errorf("API request failed: status: 502 body: bad gateway"),
		},
	}

	notices := make([]string, 0, 1)
	cfg := ToolLoopConfig{
		Provider:      provider,
		Model:         "test-model",
		MaxIterations: 1,
		RetryPolicy: &utils.RetryPolicy{
			AttemptTimeouts: []time.Duration{time.Second, time.Second},
		},
		RetryNotice: func(content string) {
			notices = append(notices, content)
		},
	}

	result, err := RunToolLoop(
		context.Background(),
		cfg,
		[]providers.Message{{Role: "user", Content: "hello"}},
		"test",
		"chat-1",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.Content != "ok" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if provider.calls != 2 {
		t.Fatalf("provider.calls = %d, want 2", provider.calls)
	}
	if len(notices) != 1 {
		t.Fatalf("notices = %d, want 1", len(notices))
	}
	if !strings.Contains(strings.ToLower(notices[0]), "retry") {
		t.Fatalf("notice = %q, want retry hint", notices[0])
	}
}

func TestRunToolLoop_NonRetryableError_NoRetry(t *testing.T) {
	provider := &flakyToolLoopProvider{
		errors: []error{
			fmt.Errorf("API request failed: status: 400 body: bad request"),
		},
	}

	cfg := ToolLoopConfig{
		Provider:      provider,
		Model:         "test-model",
		MaxIterations: 1,
		RetryPolicy: &utils.RetryPolicy{
			AttemptTimeouts: []time.Duration{time.Second, time.Second},
		},
	}

	_, err := RunToolLoop(
		context.Background(),
		cfg,
		[]providers.Message{{Role: "user", Content: "hello"}},
		"test",
		"chat-1",
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if provider.calls != 1 {
		t.Fatalf("provider.calls = %d, want 1", provider.calls)
	}
}
