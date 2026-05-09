package evolution

import (
	"context"
	"errors"
	"sync"
)

type coldPathRuntime interface {
	RunColdPathOnce(ctx context.Context, workspace string) error
}

type ColdPathRunner struct {
	runtime coldPathRuntime
	async   func(func())
	onError func(error)
	ctx     context.Context
	cancel  context.CancelFunc

	mu        sync.Mutex
	wg        sync.WaitGroup
	closeOnce sync.Once
	closed    bool
	running   map[string]workspaceRunState
}

func NewColdPathRunner(runtime coldPathRuntime) *ColdPathRunner {
	return NewColdPathRunnerWithErrorHandler(runtime, nil)
}

func NewColdPathRunnerWithErrorHandler(runtime coldPathRuntime, onError func(error)) *ColdPathRunner {
	if onError == nil {
		onError = func(error) {}
	}
	ctx, cancel := context.WithCancel(context.Background())

	return &ColdPathRunner{
		runtime: runtime,
		async: func(run func()) {
			go run()
		},
		onError: onError,
		ctx:     ctx,
		cancel:  cancel,
		running: make(map[string]workspaceRunState),
	}
}

type workspaceRunState struct {
	running bool
	pending bool
}

func (r *ColdPathRunner) Trigger(workspace string) bool {
	if r == nil || r.runtime == nil || workspace == "" {
		return false
	}

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return false
	}
	state, exists := r.running[workspace]
	if exists && state.running {
		state.pending = true
		r.running[workspace] = state
		r.mu.Unlock()
		return true
	}
	r.running[workspace] = workspaceRunState{running: true}
	r.wg.Add(1)
	r.mu.Unlock()

	r.async(func() {
		defer r.wg.Done()
		r.runWorkspace(workspace)
	})

	return true
}

func (r *ColdPathRunner) runWorkspace(workspace string) {
	for {
		if err := r.runtime.RunColdPathOnce(r.ctx, workspace); err != nil && !errors.Is(err, context.Canceled) {
			r.onError(err)
		}

		r.mu.Lock()
		state, exists := r.running[workspace]
		if !exists || r.closed {
			delete(r.running, workspace)
			r.mu.Unlock()
			return
		}
		if state.pending {
			state.pending = false
			r.running[workspace] = state
			r.mu.Unlock()
			continue
		}
		delete(r.running, workspace)
		r.mu.Unlock()
		return
	}
}

func (r *ColdPathRunner) Close() error {
	if r == nil {
		return nil
	}

	r.closeOnce.Do(func() {
		r.mu.Lock()
		r.closed = true
		r.mu.Unlock()
		r.cancel()
	})
	r.wg.Wait()
	return nil
}
