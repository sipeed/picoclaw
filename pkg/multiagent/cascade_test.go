package multiagent

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunRegistry_RegisterAndDeregister(t *testing.T) {
	reg := NewRunRegistry()
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	reg.Register(&ActiveRun{
		SessionKey: "run-1",
		AgentID:    "coder",
		Cancel:     cancel,
		StartedAt:  time.Now(),
	})

	if reg.ActiveCount() != 1 {
		t.Fatalf("ActiveCount = %d, want 1", reg.ActiveCount())
	}

	reg.Deregister("run-1")

	if reg.ActiveCount() != 0 {
		t.Fatalf("ActiveCount = %d after deregister, want 0", reg.ActiveCount())
	}
}

func TestRunRegistry_CascadeStop_SingleRun(t *testing.T) {
	reg := NewRunRegistry()
	ctx, cancel := context.WithCancel(context.Background())

	reg.Register(&ActiveRun{
		SessionKey: "run-1",
		AgentID:    "coder",
		Cancel:     cancel,
		StartedAt:  time.Now(),
	})

	killed := reg.CascadeStop("run-1")
	if killed != 1 {
		t.Fatalf("killed = %d, want 1", killed)
	}

	// Context should be cancelled
	select {
	case <-ctx.Done():
		// expected
	default:
		t.Fatal("context not cancelled after CascadeStop")
	}

	if reg.ActiveCount() != 0 {
		t.Fatalf("ActiveCount = %d after CascadeStop, want 0", reg.ActiveCount())
	}
}

func TestRunRegistry_CascadeStop_ParentChildChain(t *testing.T) {
	reg := NewRunRegistry()

	// Build: parent → child → grandchild
	_, cancelParent := context.WithCancel(context.Background())
	ctxChild, cancelChild := context.WithCancel(context.Background())
	ctxGrandchild, cancelGrandchild := context.WithCancel(context.Background())

	reg.Register(&ActiveRun{
		SessionKey: "parent",
		AgentID:    "main",
		ParentKey:  "",
		Cancel:     cancelParent,
		StartedAt:  time.Now(),
	})
	reg.Register(&ActiveRun{
		SessionKey: "child",
		AgentID:    "coder",
		ParentKey:  "parent",
		Cancel:     cancelChild,
		StartedAt:  time.Now(),
	})
	reg.Register(&ActiveRun{
		SessionKey: "grandchild",
		AgentID:    "researcher",
		ParentKey:  "child",
		Cancel:     cancelGrandchild,
		StartedAt:  time.Now(),
	})

	if reg.ActiveCount() != 3 {
		t.Fatalf("ActiveCount = %d, want 3", reg.ActiveCount())
	}

	// Cascade stop from parent should kill all 3
	killed := reg.CascadeStop("parent")
	if killed != 3 {
		t.Fatalf("killed = %d, want 3", killed)
	}

	// All contexts should be cancelled
	select {
	case <-ctxChild.Done():
	default:
		t.Fatal("child context not cancelled")
	}
	select {
	case <-ctxGrandchild.Done():
	default:
		t.Fatal("grandchild context not cancelled")
	}

	if reg.ActiveCount() != 0 {
		t.Fatalf("ActiveCount = %d after cascade, want 0", reg.ActiveCount())
	}
}

func TestRunRegistry_CascadeStop_MidChain(t *testing.T) {
	reg := NewRunRegistry()

	// Build: parent → child → grandchild
	_, cancelParent := context.WithCancel(context.Background())
	_, cancelChild := context.WithCancel(context.Background())
	ctxGrandchild, cancelGrandchild := context.WithCancel(context.Background())

	reg.Register(&ActiveRun{
		SessionKey: "parent",
		AgentID:    "main",
		ParentKey:  "",
		Cancel:     cancelParent,
		StartedAt:  time.Now(),
	})
	reg.Register(&ActiveRun{
		SessionKey: "child",
		AgentID:    "coder",
		ParentKey:  "parent",
		Cancel:     cancelChild,
		StartedAt:  time.Now(),
	})
	reg.Register(&ActiveRun{
		SessionKey: "grandchild",
		AgentID:    "researcher",
		ParentKey:  "child",
		Cancel:     cancelGrandchild,
		StartedAt:  time.Now(),
	})

	// Cascade stop from child should kill child + grandchild, but NOT parent
	killed := reg.CascadeStop("child")
	if killed != 2 {
		t.Fatalf("killed = %d, want 2", killed)
	}

	// Grandchild context cancelled
	select {
	case <-ctxGrandchild.Done():
	default:
		t.Fatal("grandchild context not cancelled")
	}

	// Parent still active
	if reg.ActiveCount() != 1 {
		t.Fatalf("ActiveCount = %d, want 1 (parent survives)", reg.ActiveCount())
	}
}

func TestRunRegistry_CascadeStop_MultipleSiblings(t *testing.T) {
	reg := NewRunRegistry()

	_, cancelParent := context.WithCancel(context.Background())
	_, cancelSibling1 := context.WithCancel(context.Background())
	_, cancelSibling2 := context.WithCancel(context.Background())

	reg.Register(&ActiveRun{
		SessionKey: "parent",
		AgentID:    "main",
		Cancel:     cancelParent,
		StartedAt:  time.Now(),
	})
	reg.Register(&ActiveRun{
		SessionKey: "sibling-1",
		AgentID:    "coder",
		ParentKey:  "parent",
		Cancel:     cancelSibling1,
		StartedAt:  time.Now(),
	})
	reg.Register(&ActiveRun{
		SessionKey: "sibling-2",
		AgentID:    "researcher",
		ParentKey:  "parent",
		Cancel:     cancelSibling2,
		StartedAt:  time.Now(),
	})

	killed := reg.CascadeStop("parent")
	if killed != 3 {
		t.Fatalf("killed = %d, want 3 (parent + 2 siblings)", killed)
	}
	if reg.ActiveCount() != 0 {
		t.Fatalf("ActiveCount = %d, want 0", reg.ActiveCount())
	}
}

func TestRunRegistry_CascadeStop_CycleProtection(t *testing.T) {
	reg := NewRunRegistry()

	// Manually create a cycle: A→B→A (shouldn't happen in practice, but guard against it)
	_, cancelA := context.WithCancel(context.Background())
	_, cancelB := context.WithCancel(context.Background())

	reg.Register(&ActiveRun{
		SessionKey: "A",
		AgentID:    "agent-a",
		ParentKey:  "B",
		Cancel:     cancelA,
		StartedAt:  time.Now(),
	})
	reg.Register(&ActiveRun{
		SessionKey: "B",
		AgentID:    "agent-b",
		ParentKey:  "A",
		Cancel:     cancelB,
		StartedAt:  time.Now(),
	})

	// Should not infinite-loop, should kill both
	killed := reg.CascadeStop("A")
	if killed != 2 {
		t.Fatalf("killed = %d, want 2", killed)
	}
}

func TestRunRegistry_CascadeStop_NonExistent(t *testing.T) {
	reg := NewRunRegistry()

	killed := reg.CascadeStop("ghost")
	if killed != 0 {
		t.Fatalf("killed = %d, want 0 for non-existent key", killed)
	}
}

func TestRunRegistry_StopAll(t *testing.T) {
	reg := NewRunRegistry()

	var cancelled atomic.Int32
	for i := 0; i < 5; i++ {
		_, cancel := context.WithCancel(context.Background())
		idx := i
		reg.Register(&ActiveRun{
			SessionKey: fmt.Sprintf("run-%d", idx),
			AgentID:    "agent",
			Cancel: func() {
				cancelled.Add(1)
				cancel()
			},
			StartedAt: time.Now(),
		})
	}

	killed := reg.StopAll()
	if killed != 5 {
		t.Fatalf("StopAll killed = %d, want 5", killed)
	}
	if cancelled.Load() != 5 {
		t.Fatalf("cancel called %d times, want 5", cancelled.Load())
	}
	if reg.ActiveCount() != 0 {
		t.Fatalf("ActiveCount = %d after StopAll, want 0", reg.ActiveCount())
	}
}

func TestRunRegistry_GetChildren(t *testing.T) {
	reg := NewRunRegistry()
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	reg.Register(&ActiveRun{SessionKey: "parent", Cancel: cancel, StartedAt: time.Now()})
	reg.Register(&ActiveRun{SessionKey: "child-1", ParentKey: "parent", Cancel: cancel, StartedAt: time.Now()})
	reg.Register(&ActiveRun{SessionKey: "child-2", ParentKey: "parent", Cancel: cancel, StartedAt: time.Now()})
	reg.Register(&ActiveRun{SessionKey: "other", ParentKey: "other-parent", Cancel: cancel, StartedAt: time.Now()})

	children := reg.GetChildren("parent")
	if len(children) != 2 {
		t.Fatalf("GetChildren = %d, want 2", len(children))
	}
}

func TestRunRegistry_ContextCancellationPropagates(t *testing.T) {
	reg := NewRunRegistry()

	// Simulate Go context tree: parent ctx → child ctx
	parentCtx, parentCancel := context.WithCancel(context.Background())
	childCtx, childCancel := context.WithCancel(parentCtx)

	reg.Register(&ActiveRun{
		SessionKey: "parent",
		AgentID:    "main",
		Cancel:     parentCancel,
		StartedAt:  time.Now(),
	})
	reg.Register(&ActiveRun{
		SessionKey: "child",
		AgentID:    "coder",
		ParentKey:  "parent",
		Cancel:     childCancel,
		StartedAt:  time.Now(),
	})

	// Cascade stop on parent should cancel parent context
	reg.CascadeStop("parent")

	// Parent context cancelled
	select {
	case <-parentCtx.Done():
	default:
		t.Fatal("parent context not cancelled")
	}

	// Child context should ALSO be cancelled (Go context tree propagation)
	select {
	case <-childCtx.Done():
	default:
		t.Fatal("child context not cancelled via Go context tree")
	}
}
