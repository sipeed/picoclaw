package behavior

import (
	"context"
	"fmt"
	"sync"

	"github.com/sipeed/picoclaw/pkg/agent"
)

type turnStats struct {
	toolCalls  int
	totalBytes int64
}

// Monitor implements agent.ToolInterceptor and agent.EventObserver to detect behavioral anomalies.
type Monitor struct {
	MaxToolCalls  int
	MaxTotalBytes int64

	mu    sync.Mutex
	turns map[string]*turnStats
}

// Ensure Monitor implements necessary interfaces.
var _ agent.ToolInterceptor = (*Monitor)(nil)
var _ agent.EventObserver = (*Monitor)(nil)

// NewMonitor creates a new behavioral monitor.
func NewMonitor(maxCalls int, maxBytes int64) *Monitor {
	return &Monitor{
		MaxToolCalls:  maxCalls,
		MaxTotalBytes: maxBytes,
		turns:         make(map[string]*turnStats),
	}
}

func (m *Monitor) OnEvent(ctx context.Context, evt agent.Event) error {
	if evt.Kind == agent.EventKindTurnEnd {
		m.mu.Lock()
		delete(m.turns, evt.Meta.TurnID)
		m.mu.Unlock()
	}
	return nil
}

func (m *Monitor) BeforeTool(ctx context.Context, call *agent.ToolCallHookRequest) (*agent.ToolCallHookRequest, agent.HookDecision, error) {
	if call == nil {
		return nil, agent.HookDecision{}, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	stats, ok := m.turns[call.Meta.TurnID]
	if !ok {
		stats = &turnStats{}
		m.turns[call.Meta.TurnID] = stats
	}

	stats.toolCalls++

	if m.MaxToolCalls > 0 && stats.toolCalls > m.MaxToolCalls {
		return call, agent.HookDecision{
			Action: agent.HookActionAbortTurn,
			Reason: fmt.Sprintf("Behavioral defense: Tool call limit (%d) exceeded in a single turn", m.MaxToolCalls),
		}, nil
	}

	return call, agent.HookDecision{Action: agent.HookActionContinue}, nil
}

func (m *Monitor) AfterTool(ctx context.Context, resp *agent.ToolResultHookResponse) (*agent.ToolResultHookResponse, agent.HookDecision, error) {
	if resp == nil || resp.Result == nil {
		return resp, agent.HookDecision{Action: agent.HookActionContinue}, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	stats, ok := m.turns[resp.Meta.TurnID]
	if !ok {
		// Should have been created in BeforeTool, but handle just in case.
		return resp, agent.HookDecision{Action: agent.HookActionContinue}, nil
	}

	stats.totalBytes += int64(len(resp.Result.ForLLM))

	if m.MaxTotalBytes > 0 && stats.totalBytes > m.MaxTotalBytes {
		return resp, agent.HookDecision{
			Action: agent.HookActionAbortTurn,
			Reason: fmt.Sprintf("Behavioral defense: Cumulative tool output size limit (%d bytes) exceeded in a single turn", m.MaxTotalBytes),
		}, nil
	}

	return resp, agent.HookDecision{Action: agent.HookActionContinue}, nil
}
