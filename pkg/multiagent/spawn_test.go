package multiagent

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// mockAgentResolver provides test agents.
type mockAgentResolver struct {
	agents map[string]*AgentInfo
}

func (r *mockAgentResolver) GetAgentInfo(id string) *AgentInfo {
	return r.agents[id]
}

func (r *mockAgentResolver) ListAgents() []AgentInfo {
	var list []AgentInfo
	for _, a := range r.agents {
		list = append(list, *a)
	}
	return list
}

// mockLLMProvider returns a fixed response after a configurable delay.
type mockLLMProvider struct {
	response string
	delay    time.Duration
}

func (m *mockLLMProvider) Chat(ctx context.Context, messages []providers.Message, t []providers.ToolDefinition, model string, opts map[string]interface{}) (*providers.LLMResponse, error) {
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.delay):
		}
	}
	return &providers.LLMResponse{Content: m.response}, nil
}

func (m *mockLLMProvider) GetDefaultModel() string { return "mock" }

func newTestResolver() *mockAgentResolver {
	toolReg := tools.NewToolRegistry()
	return &mockAgentResolver{
		agents: map[string]*AgentInfo{
			"worker-a": {
				ID:       "worker-a",
				Name:     "Worker A",
				Role:     "test worker",
				Provider: &mockLLMProvider{response: "result from A", delay: 50 * time.Millisecond},
				Tools:    toolReg,
				MaxIter:  3,
			},
			"worker-b": {
				ID:       "worker-b",
				Name:     "Worker B",
				Role:     "test worker",
				Provider: &mockLLMProvider{response: "result from B", delay: 50 * time.Millisecond},
				Tools:    toolReg,
				MaxIter:  3,
			},
		},
	}
}

func TestAsyncSpawn_Accepted(t *testing.T) {
	registry := NewRunRegistry()
	announcer := NewAnnouncer(10)
	sm := NewSpawnManager(registry, announcer, 5, 10*time.Second)
	resolver := newTestResolver()
	board := NewBlackboard()

	result := sm.AsyncSpawn(context.Background(), resolver, board, SpawnRequest{
		FromAgentID:  "main",
		ToAgentID:    "worker-a",
		Task:         "do something",
		ParentRunKey: "parent-session",
	}, "test", "chat1")

	if result.Status != "accepted" {
		t.Fatalf("expected accepted, got %s: %s", result.Status, result.Error)
	}
	if result.RunID == "" {
		t.Error("expected non-empty RunID")
	}

	// Wait for completion
	time.Sleep(200 * time.Millisecond)

	// Check announcement was delivered
	anns := announcer.Drain("parent-session")
	if len(anns) == 0 {
		t.Fatal("expected at least 1 announcement")
	}
	if anns[0].AgentID != "worker-a" {
		t.Errorf("expected agent worker-a, got %s", anns[0].AgentID)
	}
	if anns[0].Outcome == nil || !anns[0].Outcome.Success {
		t.Error("expected successful outcome")
	}
}

func TestAsyncSpawn_ConcurrencyLimit(t *testing.T) {
	registry := NewRunRegistry()
	announcer := NewAnnouncer(10)
	sm := NewSpawnManager(registry, announcer, 2, 10*time.Second) // max 2 concurrent

	slowProvider := &mockLLMProvider{response: "slow result", delay: 500 * time.Millisecond}
	toolReg := tools.NewToolRegistry()
	resolver := &mockAgentResolver{
		agents: map[string]*AgentInfo{
			"worker": {
				ID: "worker", Name: "Worker", Provider: slowProvider,
				Tools: toolReg, MaxIter: 3,
			},
		},
	}
	board := NewBlackboard()

	// Spawn 2 (should succeed — at limit)
	r1 := sm.AsyncSpawn(context.Background(), resolver, board, SpawnRequest{
		FromAgentID: "main", ToAgentID: "worker", Task: "task 1", ParentRunKey: "parent",
	}, "test", "chat1")
	r2 := sm.AsyncSpawn(context.Background(), resolver, board, SpawnRequest{
		FromAgentID: "main", ToAgentID: "worker", Task: "task 2", ParentRunKey: "parent",
	}, "test", "chat1")

	if r1.Status != "accepted" || r2.Status != "accepted" {
		t.Fatalf("first 2 should be accepted, got r1=%s r2=%s", r1.Status, r2.Status)
	}

	// Spawn 3rd (should be rejected — over limit)
	r3 := sm.AsyncSpawn(context.Background(), resolver, board, SpawnRequest{
		FromAgentID: "main", ToAgentID: "worker", Task: "task 3", ParentRunKey: "parent",
	}, "test", "chat1")

	if r3.Status != "rejected" {
		t.Errorf("3rd spawn should be rejected, got %s", r3.Status)
	}

	// Wait for first 2 to complete
	time.Sleep(700 * time.Millisecond)

	// Now should be able to spawn again
	r4 := sm.AsyncSpawn(context.Background(), resolver, board, SpawnRequest{
		FromAgentID: "main", ToAgentID: "worker", Task: "task 4", ParentRunKey: "parent",
	}, "test", "chat1")

	if r4.Status != "accepted" {
		t.Errorf("4th spawn should be accepted after slots freed, got %s: %s", r4.Status, r4.Error)
	}
}

func TestAsyncSpawn_CascadeStop(t *testing.T) {
	registry := NewRunRegistry()
	announcer := NewAnnouncer(10)
	sm := NewSpawnManager(registry, announcer, 5, 10*time.Second)

	slowProvider := &mockLLMProvider{response: "should not complete", delay: 2 * time.Second}
	toolReg := tools.NewToolRegistry()
	resolver := &mockAgentResolver{
		agents: map[string]*AgentInfo{
			"worker": {
				ID: "worker", Name: "Worker", Provider: slowProvider,
				Tools: toolReg, MaxIter: 3,
			},
		},
	}
	board := NewBlackboard()

	r := sm.AsyncSpawn(context.Background(), resolver, board, SpawnRequest{
		FromAgentID: "main", ToAgentID: "worker", Task: "long task", ParentRunKey: "parent",
	}, "test", "chat1")

	if r.Status != "accepted" {
		t.Fatalf("expected accepted, got %s", r.Status)
	}

	// Give goroutine time to start
	time.Sleep(50 * time.Millisecond)

	// Verify it's registered
	if registry.ActiveCount() == 0 {
		t.Fatal("expected at least 1 active run")
	}

	// Cascade stop should cancel it
	killed := registry.CascadeStop(r.SessionKey)
	if killed == 0 {
		t.Error("expected cascade stop to kill at least 1 run")
	}

	// Wait for goroutine to clean up
	time.Sleep(200 * time.Millisecond)
}

func TestAsyncSpawn_ContextTimeout(t *testing.T) {
	registry := NewRunRegistry()
	announcer := NewAnnouncer(10)
	sm := NewSpawnManager(registry, announcer, 5, 200*time.Millisecond) // very short timeout

	slowProvider := &mockLLMProvider{response: "too slow", delay: 5 * time.Second}
	toolReg := tools.NewToolRegistry()
	resolver := &mockAgentResolver{
		agents: map[string]*AgentInfo{
			"worker": {
				ID: "worker", Name: "Worker", Provider: slowProvider,
				Tools: toolReg, MaxIter: 3,
			},
		},
	}
	board := NewBlackboard()

	sm.AsyncSpawn(context.Background(), resolver, board, SpawnRequest{
		FromAgentID: "main", ToAgentID: "worker", Task: "slow task", ParentRunKey: "parent",
	}, "test", "chat1")

	// Wait for timeout + cleanup
	time.Sleep(500 * time.Millisecond)

	// Should have an announcement with failure
	anns := announcer.Drain("parent")
	if len(anns) == 0 {
		t.Fatal("expected announcement after timeout")
	}
	if anns[0].Outcome.Success {
		t.Error("expected failure after timeout")
	}
}

func TestAsyncSpawn_ParallelFanOut(t *testing.T) {
	registry := NewRunRegistry()
	announcer := NewAnnouncer(20)
	sm := NewSpawnManager(registry, announcer, 10, 10*time.Second)
	resolver := newTestResolver()
	board := NewBlackboard()

	// Fan-out: spawn multiple agents in parallel (Google MapReduce pattern)
	var results []*SpawnResult
	for i := 0; i < 5; i++ {
		target := "worker-a"
		if i%2 == 1 {
			target = "worker-b"
		}
		r := sm.AsyncSpawn(context.Background(), resolver, board, SpawnRequest{
			FromAgentID:  "main",
			ToAgentID:    target,
			Task:         fmt.Sprintf("parallel task %d", i),
			ParentRunKey: "parent",
		}, "test", "chat1")
		results = append(results, r)
	}

	// All should be accepted
	for i, r := range results {
		if r.Status != "accepted" {
			t.Errorf("spawn %d should be accepted, got %s", i, r.Status)
		}
	}

	// Fan-in: wait for all to complete and collect results
	time.Sleep(500 * time.Millisecond)

	anns := announcer.Drain("parent")
	if len(anns) != 5 {
		t.Errorf("expected 5 announcements (fan-in), got %d", len(anns))
	}
}

// TestAnnouncer tests

func TestAnnouncer_DeliverAndDrain(t *testing.T) {
	a := NewAnnouncer(10)

	a.Deliver("session-1", &Announcement{
		RunID:   "run-1",
		AgentID: "worker-a",
		Content: "result 1",
	})
	a.Deliver("session-1", &Announcement{
		RunID:   "run-2",
		AgentID: "worker-b",
		Content: "result 2",
	})

	results := a.Drain("session-1")
	if len(results) != 2 {
		t.Fatalf("expected 2 announcements, got %d", len(results))
	}

	// Drain again should return empty
	results2 := a.Drain("session-1")
	if len(results2) != 0 {
		t.Errorf("expected 0 after drain, got %d", len(results2))
	}
}

func TestAnnouncer_Pending(t *testing.T) {
	a := NewAnnouncer(10)

	if a.Pending("session-1") != 0 {
		t.Error("expected 0 pending for new session")
	}

	a.Deliver("session-1", &Announcement{RunID: "r1"})
	a.Deliver("session-1", &Announcement{RunID: "r2"})

	if a.Pending("session-1") != 2 {
		t.Errorf("expected 2 pending, got %d", a.Pending("session-1"))
	}
}

func TestAnnouncer_BackPressure(t *testing.T) {
	a := NewAnnouncer(2) // tiny buffer

	// Fill buffer
	a.Deliver("session-1", &Announcement{RunID: "r1", Content: "first"})
	a.Deliver("session-1", &Announcement{RunID: "r2", Content: "second"})

	// Overflow — should drop oldest
	a.Deliver("session-1", &Announcement{RunID: "r3", Content: "third"})

	results := a.Drain("session-1")
	if len(results) != 2 {
		t.Fatalf("expected 2 after back-pressure, got %d", len(results))
	}
	// Most recent should be present
	hasThird := false
	for _, r := range results {
		if r.RunID == "r3" {
			hasThird = true
		}
	}
	if !hasThird {
		t.Error("expected the newest announcement (r3) to be present after back-pressure")
	}
}

func TestAnnouncer_ConcurrentDelivery(t *testing.T) {
	a := NewAnnouncer(100)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			a.Deliver("session-1", &Announcement{
				RunID:   fmt.Sprintf("r-%d", n),
				Content: fmt.Sprintf("result %d", n),
			})
		}(i)
	}
	wg.Wait()

	results := a.Drain("session-1")
	if len(results) != 50 {
		t.Errorf("expected 50 concurrent deliveries, got %d", len(results))
	}
}

func TestAnnouncer_Cleanup(t *testing.T) {
	a := NewAnnouncer(10)
	a.Deliver("session-1", &Announcement{RunID: "r1"})
	a.Cleanup("session-1")

	// After cleanup, pending should be 0 (new channel)
	if a.Pending("session-1") != 0 {
		t.Error("expected 0 pending after cleanup")
	}
}
