package telegram

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/commands"
)

func TestStartCommandRegistration_DoesNotBlock(t *testing.T) {
	ch := &TelegramChannel{}
	started := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch.registerFunc = func(context.Context, []commands.Definition) error {
		started <- struct{}{}
		return errors.New("temporary failure")
	}

	ch.startCommandRegistration(ctx, []commands.Definition{{Name: "help"}})

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("registration did not start asynchronously")
	}
}

func TestStartCommandRegistration_RetriesUntilSuccessThenStops(t *testing.T) {
	ch := &TelegramChannel{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	origBackoff := commandRegistrationBackoff
	commandRegistrationBackoff = []time.Duration{5 * time.Millisecond}
	defer func() { commandRegistrationBackoff = origBackoff }()

	var attempts atomic.Int32
	ch.registerFunc = func(context.Context, []commands.Definition) error {
		n := attempts.Add(1)
		if n < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}

	ch.startCommandRegistration(ctx, []commands.Definition{{Name: "help", Description: "Help"}})

	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		if attempts.Load() >= 3 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if attempts.Load() < 3 {
		t.Fatalf("expected at least 3 attempts, got %d", attempts.Load())
	}

	stable := attempts.Load()
	time.Sleep(30 * time.Millisecond)
	if attempts.Load() != stable {
		t.Fatalf("expected retries to stop after success, got %d -> %d", stable, attempts.Load())
	}
}

func TestStartCommandRegistration_StopsAfterCancel(t *testing.T) {
	ch := &TelegramChannel{}
	ctx, cancel := context.WithCancel(context.Background())

	origBackoff := commandRegistrationBackoff
	commandRegistrationBackoff = []time.Duration{5 * time.Millisecond}
	defer func() { commandRegistrationBackoff = origBackoff }()
	defer cancel()

	var attempts atomic.Int32
	ch.registerFunc = func(context.Context, []commands.Definition) error {
		attempts.Add(1)
		return errors.New("always fail")
	}

	ch.startCommandRegistration(ctx, []commands.Definition{{Name: "help", Description: "Help"}})

	time.Sleep(20 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond) // allow in-flight attempt to settle
	stable := attempts.Load()
	time.Sleep(30 * time.Millisecond)
	if attempts.Load() != stable {
		t.Fatalf("expected retries to quiesce after cancel, got %d -> %d", stable, attempts.Load())
	}
}
