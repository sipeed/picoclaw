package behavior

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonitor_ToolCallLimit(t *testing.T) {
	m := NewMonitor(2, 0)
	ctx := context.Background()
	turnID := "test-turn-1"

	// Call 1: OK
	req1 := &agent.ToolCallHookRequest{Meta: agent.EventMeta{TurnID: turnID}}
	_, dec1, err := m.BeforeTool(ctx, req1)
	require.NoError(t, err)
	assert.Equal(t, agent.HookActionContinue, dec1.Action)

	// Call 2: OK
	req2 := &agent.ToolCallHookRequest{Meta: agent.EventMeta{TurnID: turnID}}
	_, dec2, err := m.BeforeTool(ctx, req2)
	require.NoError(t, err)
	assert.Equal(t, agent.HookActionContinue, dec2.Action)

	// Call 3: Blocked
	req3 := &agent.ToolCallHookRequest{Meta: agent.EventMeta{TurnID: turnID}}
	_, dec3, err := m.BeforeTool(ctx, req3)
	require.NoError(t, err)
	assert.Equal(t, agent.HookActionAbortTurn, dec3.Action)
	assert.Contains(t, dec3.Reason, "Tool call limit")
}

func TestMonitor_DataLimit(t *testing.T) {
	m := NewMonitor(0, 10)
	ctx := context.Background()
	turnID := "test-turn-2"

	// BeforeTool needed to init stats
	m.BeforeTool(ctx, &agent.ToolCallHookRequest{Meta: agent.EventMeta{TurnID: turnID}})

	// AfterTool 1: OK (5 bytes)
	resp1 := &agent.ToolResultHookResponse{
		Meta:   agent.EventMeta{TurnID: turnID},
		Result: &tools.ToolResult{ForLLM: "12345"},
	}
	_, dec1, err := m.AfterTool(ctx, resp1)
	require.NoError(t, err)
	assert.Equal(t, agent.HookActionContinue, dec1.Action)

	// AfterTool 2: Blocked (accumulated 11 bytes)
	resp2 := &agent.ToolResultHookResponse{
		Meta:   agent.EventMeta{TurnID: turnID},
		Result: &tools.ToolResult{ForLLM: "678901"},
	}
	_, dec2, err := m.AfterTool(ctx, resp2)
	require.NoError(t, err)
	assert.Equal(t, agent.HookActionAbortTurn, dec2.Action)
	assert.Contains(t, dec2.Reason, "Cumulative tool output size limit")
}

func TestMonitor_Cleanup(t *testing.T) {
	m := NewMonitor(1, 0)
	ctx := context.Background()
	turnID := "test-turn-3"

	// Call 1: OK
	m.BeforeTool(ctx, &agent.ToolCallHookRequest{Meta: agent.EventMeta{TurnID: turnID}})

	// End turn
	m.OnEvent(ctx, agent.Event{Kind: agent.EventKindTurnEnd, Meta: agent.EventMeta{TurnID: turnID}})

	// Call 1 again (new turn or same ID after cleanup): should be OK again
	_, dec, err := m.BeforeTool(ctx, &agent.ToolCallHookRequest{Meta: agent.EventMeta{TurnID: turnID}})
	require.NoError(t, err)
	assert.Equal(t, agent.HookActionContinue, dec.Action)
}
