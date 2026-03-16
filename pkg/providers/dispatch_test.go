// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers_test

import (
	"context"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// minimalCfg builds a *config.Config with one ModelConfig entry that will
// successfully create an antigravity provider (no API key required).
func minimalCfg(modelKey string) *config.Config {
	return &config.Config{
		ModelList: []config.ModelConfig{
			{
				ModelName: "test-alias",
				Model:     modelKey,
			},
		},
	}
}

// TestProviderDispatcher_Get_CachesInstance verifies that calling Get twice with
// the same protocol+modelID returns the exact same provider instance.
func TestProviderDispatcher_Get_CachesInstance(t *testing.T) {
	cfg := minimalCfg("antigravity/test")
	d := providers.NewProviderDispatcher(cfg)

	p1, err := d.Get("antigravity", "test")
	if err != nil {
		t.Fatalf("first Get: unexpected error: %v", err)
	}
	if p1 == nil {
		t.Fatal("first Get: returned nil provider")
	}

	p2, err := d.Get("antigravity", "test")
	if err != nil {
		t.Fatalf("second Get: unexpected error: %v", err)
	}

	if p1 != p2 {
		t.Errorf("expected cached provider instance, got different pointers: %p vs %p", p1, p2)
	}
}

// TestProviderDispatcher_Get_UnknownProtocol verifies that Get returns an error
// when no ModelConfig entry matches the requested protocol+modelID.
func TestProviderDispatcher_Get_UnknownProtocol(t *testing.T) {
	cfg := minimalCfg("antigravity/test")
	d := providers.NewProviderDispatcher(cfg)

	_, err := d.Get("unknown-protocol", "no-such-model")
	if err == nil {
		t.Fatal("expected error for unknown protocol/model, got nil")
	}
}

// TestProviderDispatcher_Flush verifies that Flush clears the cache so that a
// subsequent Get creates a new provider instance rather than returning the old one.
func TestProviderDispatcher_Flush(t *testing.T) {
	cfg := minimalCfg("antigravity/test")
	d := providers.NewProviderDispatcher(cfg)

	p1, err := d.Get("antigravity", "test")
	if err != nil {
		t.Fatalf("pre-flush Get: %v", err)
	}

	// Flush with the same config (simulating a reload).
	d.Flush(cfg)

	p2, err := d.Get("antigravity", "test")
	if err != nil {
		t.Fatalf("post-flush Get: %v", err)
	}

	if p1 == p2 {
		t.Error("expected new provider instance after Flush, but got the same pointer")
	}
}

// TestProviderDispatcher_Get_ThreadSafe exercises concurrent Gets to verify
// there are no data races. Run with: go test -race ./pkg/providers/...
func TestProviderDispatcher_Get_ThreadSafe(t *testing.T) {
	cfg := minimalCfg("antigravity/concurrent")
	d := providers.NewProviderDispatcher(cfg)

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			p, err := d.Get("antigravity", "concurrent")
			if err != nil {
				t.Errorf("concurrent Get: %v", err)
				return
			}
			// Exercise the provider slightly to ensure no race on the cached value.
			_ = p.GetDefaultModel()
		}()
	}

	wg.Wait()
}

// TestProviderDispatcher_Get_FlushRace exercises concurrent Gets and Flushes
// together to verify the mutex correctly protects both operations.
func TestProviderDispatcher_Get_FlushRace(t *testing.T) {
	cfg := minimalCfg("antigravity/race")
	d := providers.NewProviderDispatcher(cfg)

	var wg sync.WaitGroup

	// Half goroutines call Get, half call Flush.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = d.Get("antigravity", "race")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.Flush(cfg)
		}()
	}

	wg.Wait()
}

// Compile-time check: antigravity provider satisfies LLMProvider.
var _ providers.LLMProvider = (func() providers.LLMProvider {
	p, _, _ := providers.CreateProviderFromConfig(&config.ModelConfig{Model: "antigravity/x"})
	return p
})()

// TestAntigravityProvider_Chat is a lightweight smoke test confirming the
// antigravity provider (used in dispatcher tests) satisfies the interface.
func TestAntigravityProvider_Chat(t *testing.T) {
	cfg := minimalCfg("antigravity/smoke")
	d := providers.NewProviderDispatcher(cfg)

	p, err := d.Get("antigravity", "smoke")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	// The antigravity provider's Chat is a no-op stub; just ensure it doesn't panic.
	resp, err := p.Chat(context.Background(), nil, nil, "smoke", nil)
	// antigravity may return nil response + nil error or an error; either is fine.
	_ = resp
	_ = err
}
