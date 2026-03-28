package pii

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactor_Redact(t *testing.T) {
	r := NewRedactor(true)

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello, contact me at steve@example.com", "Hello, contact me at [EMAIL]"},
		{"My IP is 192.168.1.1", "My IP is [IP]"},
		{"Call me at +1 555-123-4567", "Call me at [PHONE]"},
		{"Nothing sensitive here", "Nothing sensitive here"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, r.redact(tt.input))
	}
}

func TestRedactor_BeforeLLM(t *testing.T) {
	r := NewRedactor(true)
	ctx := context.Background()

	req := &agent.LLMHookRequest{
		Messages: []providers.Message{
			{Role: "user", Content: "My email is user@foo.com"},
			{Role: "system", Content: "Keep 127.0.0.1"}, // system message should not be redacted
		},
	}

	next, decision, err := r.BeforeLLM(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, agent.HookActionContinue, decision.Action)

	assert.Equal(t, "My email is [EMAIL]", next.Messages[0].Content)
	assert.Equal(t, "Keep 127.0.0.1", next.Messages[1].Content)
}

func TestRedactor_AfterLLM(t *testing.T) {
	r := NewRedactor(true)
	ctx := context.Background()

	resp := &agent.LLMHookResponse{
		Response: &providers.LLMResponse{
			Content: "The user's email was user@foo.com",
		},
	}

	next, decision, err := r.AfterLLM(ctx, resp)
	require.NoError(t, err)
	assert.Equal(t, agent.HookActionContinue, decision.Action)

	assert.Equal(t, "The user's email was [EMAIL]", next.Response.Content)
}
