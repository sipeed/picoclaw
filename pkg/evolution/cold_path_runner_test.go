package evolution

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type blockingColdPathRuntime struct {
	runCount    atomic.Int32
	cancelCount atomic.Int32
	started     chan string
	release     chan struct{}
}

func (r *blockingColdPathRuntime) RunColdPathOnce(ctx context.Context, workspace string) error {
	r.runCount.Add(1)
	r.started <- workspace
	select {
	case <-r.release:
		return nil
	case <-ctx.Done():
		r.cancelCount.Add(1)
		return ctx.Err()
	}
}

func TestColdPathRunner_QueuesPendingRunForWorkspace(t *testing.T) {
	runtime := &blockingColdPathRuntime{
		started: make(chan string, 4),
		release: make(chan struct{}, 4),
	}
	runner := NewColdPathRunner(runtime)
	defer runner.Close()

	if scheduled := runner.Trigger("workspace-a"); !scheduled {
		t.Fatal("expected first trigger to be scheduled")
	}

	select {
	case workspace := <-runtime.started:
		if workspace != "workspace-a" {
			t.Fatalf("workspace = %q, want workspace-a", workspace)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first cold path run")
	}

	if scheduled := runner.Trigger("workspace-a"); !scheduled {
		t.Fatal("expected second trigger to queue a pending run")
	}

	select {
	case workspace := <-runtime.started:
		t.Fatalf("unexpected early pending cold path run for %q", workspace)
	case <-time.After(150 * time.Millisecond):
	}

	runtime.release <- struct{}{}

	select {
	case workspace := <-runtime.started:
		if workspace != "workspace-a" {
			t.Fatalf("workspace = %q, want workspace-a", workspace)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for pending cold path run")
	}

	runtime.release <- struct{}{}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.runCount.Load() == 2 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("runCount = %d, want 2", runtime.runCount.Load())
}

func TestColdPathRunner_CloseCancelsActiveRunAndDropsPendingWork(t *testing.T) {
	runtime := &blockingColdPathRuntime{
		started: make(chan string, 4),
		release: make(chan struct{}, 4),
	}
	runner := NewColdPathRunner(runtime)

	if scheduled := runner.Trigger("workspace-a"); !scheduled {
		t.Fatal("expected first trigger to be scheduled")
	}

	select {
	case <-runtime.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first cold path run")
	}

	if scheduled := runner.Trigger("workspace-a"); !scheduled {
		t.Fatal("expected second trigger to mark pending work")
	}

	closeDone := make(chan struct{})
	go func() {
		defer close(closeDone)
		if err := runner.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !runner.Trigger("workspace-a") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if runner.Trigger("workspace-a") {
		t.Fatal("expected Trigger to reject new work after Close")
	}

	select {
	case <-closeDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Close to finish")
	}

	select {
	case workspace := <-runtime.started:
		t.Fatalf("unexpected pending cold path run after Close for %q", workspace)
	case <-time.After(150 * time.Millisecond):
	}

	if got := runtime.runCount.Load(); got != 1 {
		t.Fatalf("runCount = %d, want 1", got)
	}
	if got := runtime.cancelCount.Load(); got != 1 {
		t.Fatalf("cancelCount = %d, want 1", got)
	}
}
