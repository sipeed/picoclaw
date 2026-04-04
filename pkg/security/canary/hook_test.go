package canary

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanaryHook_BeforeLLM(t *testing.T) {
	h, err := NewHook()
	require.NoError(t, err)

	ctx := context.Background()
	req := &agent.LLMHookRequest{
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}

	next, decision, err := h.BeforeLLM(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, agent.HookActionContinue, decision.Action)

	// Check that a system message was added
	require.Len(t, next.Messages, 2)
	assert.Equal(t, "system", next.Messages[0].Role)
	assert.Contains(t, next.Messages[0].Content, h.token)
}

func TestCanaryHook_AfterLLM(t *testing.T) {
	h, err := NewHook()
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("SafeResponse", func(t *testing.T) {
		resp := &agent.LLMHookResponse{
			Response: &providers.LLMResponse{
				Content: "Hello World!",
			},
		}
		next, decision, err := h.AfterLLM(ctx, resp)
		require.NoError(t, err)
		assert.Equal(t, agent.HookActionContinue, decision.Action)
		assert.Equal(t, resp, next)
	})

	t.Run("LeakedResponse", func(t *testing.T) {
		resp := &agent.LLMHookResponse{
			Response: &providers.LLMResponse{
				Content: "My secret token is " + h.token,
			},
		}
		next, decision, err := h.AfterLLM(ctx, resp)
		require.NoError(t, err)
		assert.Equal(t, agent.HookActionHardAbort, decision.Action)
		assert.Contains(t, decision.Reason, "System prompt leakage detected")
		assert.Equal(t, resp, next)
	})
}
