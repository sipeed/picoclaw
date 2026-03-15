package agent

import (
	"context"
	"reflect"
	"testing"

	"github.com/sipeed/picoclaw/pkg/tools"
)

// ====================== Test Helper: Event Collector ======================
type eventCollector struct {
	events []any
}

func (c *eventCollector) collect(e any) {
	c.events = append(c.events, e)
}

func (c *eventCollector) hasEventOfType(typ any) bool {
	targetType := reflect.TypeOf(typ)
	for _, e := range c.events {
		if reflect.TypeOf(e) == targetType {
			return true
		}
	}
	return false
}

func (c *eventCollector) countOfType(typ any) int {
	targetType := reflect.TypeOf(typ)
	count := 0
	for _, e := range c.events {
		if reflect.TypeOf(e) == targetType {
			count++
		}
	}
	return count
}

// ====================== Main Test Function ======================
func TestSpawnSubTurn(t *testing.T) {
	tests := []struct {
		name          string
		parentDepth   int
		config        SubTurnConfig
		wantErr       error
		wantSpawn     bool
		wantEnd       bool
		wantDepthFail bool
	}{
		{
			name:        "Basic success path - Single layer sub-turn",
			parentDepth: 0,
			config: SubTurnConfig{
				Model: "gpt-4o-mini",
				Tools: []tools.Tool{}, // At least one tool
			},
			wantErr:   nil,
			wantSpawn: true,
			wantEnd:   true,
		},
		{
			name:        "Nested 2 layers - Normal",
			parentDepth: 1,
			config: SubTurnConfig{
				Model: "gpt-4o-mini",
				Tools: []tools.Tool{},
			},
			wantErr:   nil,
			wantSpawn: true,
			wantEnd:   true,
		},
		{
			name:        "Depth limit triggered - 4th layer fails",
			parentDepth: 3,
			config: SubTurnConfig{
				Model: "gpt-4o-mini",
				Tools: []tools.Tool{},
			},
			wantErr:       ErrDepthLimitExceeded,
			wantSpawn:     false,
			wantEnd:       false,
			wantDepthFail: true,
		},
		{
			name:        "Invalid config - Empty Model",
			parentDepth: 0,
			config: SubTurnConfig{
				Model: "",
				Tools: []tools.Tool{},
			},
			wantErr:   ErrInvalidSubTurnConfig,
			wantSpawn: false,
			wantEnd:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare parent Turn
			parent := &turnState{
				ctx:            context.Background(),
				turnID:         "parent-1",
				depth:          tt.parentDepth,
				childTurnIDs:   []string{},
				pendingResults: make(chan *ToolResult, 10),
				session:        &Session{History: []Message{}},
			}

			// Replace mock with test collector
			collector := &eventCollector{}
			originalEmit := MockEventBus.Emit
			MockEventBus.Emit = collector.collect
			defer func() { MockEventBus.Emit = originalEmit }()

			// Execute spawnSubTurn
			result, err := spawnSubTurn(context.Background(), parent, tt.config)

			// Assert errors
			if tt.wantErr != nil {
				if err == nil || err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify result
			if result == nil {
				t.Error("expected non-nil result")
			}

			// Verify event emission
			if tt.wantSpawn {
				if !collector.hasEventOfType(SubTurnSpawnEvent{}) {
					t.Error("SubTurnSpawnEvent not emitted")
				}
			}
			if tt.wantEnd {
				if !collector.hasEventOfType(SubTurnEndEvent{}) {
					t.Error("SubTurnEndEvent not emitted")
				}
			}

			// Verify turn tree
			if len(parent.childTurnIDs) == 0 && !tt.wantDepthFail {
				t.Error("child Turn not added to parent.childTurnIDs")
			}

			// Verify result delivery (pendingResults or history)
			if len(parent.pendingResults) > 0 || len(parent.session.History) > 0 {
				// Result delivered via at least one path
			} else {
				t.Error("child result not delivered")
			}
		})
	}
}

// ====================== Extra Independent Test: Ephemeral Session Isolation ======================
func TestSpawnSubTurn_EphemeralSessionIsolation(t *testing.T) {
	parent := &turnState{
		ctx:     context.Background(),
		turnID:  "parent-1",
		depth:   0,
		session: &Session{History: []Message{{Content: "parent msg"}}},
	}

	cfg := SubTurnConfig{Model: "gpt-4o-mini", Tools: []tools.Tool{}}

	// Record main session length before execution
	originalLen := len(parent.session.History)

	_, _ = spawnSubTurn(context.Background(), parent, cfg)

	// After sub-turn ends, main session must remain unchanged
	if len(parent.session.History) != originalLen {
		t.Error("ephemeral session polluted the main session")
	}
}

// ====================== Extra Independent Test: Result Delivery Path ======================
func TestSpawnSubTurn_ResultDelivery(t *testing.T) {
	parent := &turnState{
		ctx:            context.Background(),
		turnID:         "parent-1",
		depth:          0,
		pendingResults: make(chan *ToolResult, 1),
		session:        &Session{},
	}

	cfg := SubTurnConfig{Model: "gpt-4o-mini", Tools: []tools.Tool{}}

	_, _ = spawnSubTurn(context.Background(), parent, cfg)

	// Check if pendingResults received the result
	select {
	case res := <-parent.pendingResults:
		if res == nil {
			t.Error("received nil result in pendingResults")
		}
	default:
		t.Error("result did not enter pendingResults")
	}
}

// ====================== Extra Independent Test: Orphan Result Routing ======================
func TestSpawnSubTurn_OrphanResultRouting(t *testing.T) {
	parentCtx, cancelParent := context.WithCancel(context.Background())
	parent := &turnState{
		ctx:            parentCtx,
		cancelFunc:     cancelParent,
		turnID:         "parent-1",
		depth:          0,
		pendingResults: make(chan *ToolResult, 1),
		session:        &Session{History: []Message{}},
	}

	collector := &eventCollector{}
	originalEmit := MockEventBus.Emit
	MockEventBus.Emit = collector.collect
	defer func() { MockEventBus.Emit = originalEmit }()

	// Simulate parent finishing before child delivers result
	parent.Finish()

	// Call deliverSubTurnResult directly to simulate a delayed child
	deliverSubTurnResult(parent, "delayed-child", &ToolResult{Content: "late result"})

	// Verify Orphan event is emitted
	if !collector.hasEventOfType(SubTurnOrphanResultEvent{}) {
		t.Error("SubTurnOrphanResultEvent not emitted for finished parent")
	}

	// Verify history is NOT polluted
	if len(parent.session.History) != 0 {
		t.Error("Parent history was polluted by orphan result")
	}
}
