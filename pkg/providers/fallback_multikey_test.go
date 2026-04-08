package providers

import (
	"context"
	"errors"
	"testing"
)

// TestMultiKeyFailover tests the complete failover flow with multiple API keys.
// This simulates the config expansion scenario where api_keys: ["key1", "key2", "key3"]
// is expanded into primary + fallbacks.
func TestMultiKeyFailover(t *testing.T) {
	// Simulate expanded config: primary with 2 fallbacks
	// This is what ExpandMultiKeyModels would produce for api_keys: ["key1", "key2", "key3"]
	cfg := ModelConfig{
		Primary:   "glm-4.7",
		Fallbacks: []string{"glm-4.7__key_1", "glm-4.7__key_2"},
	}

	candidates := ResolveCandidates(cfg, "zhipu")

	if len(candidates) != 3 {
		t.Fatalf("expected 3 candidates, got %d: %v", len(candidates), candidates)
	}

	// Create fallback chain
	cooldown := NewCooldownTracker()
	chain := NewFallbackChain(cooldown, nil)

	// Mock run function: first call fails with 429, second succeeds
	callCount := 0
	mockRun := func(ctx context.Context, provider, model string) (*LLMResponse, error) {
		callCount++
		if callCount == 1 {
			// First call: simulate rate limit
			return nil, errors.New("http error: status 429 - rate limit exceeded")
		}
		// Second call: success
		return &LLMResponse{
			Content: "Hello from key2!",
		}, nil
	}

	// Execute fallback chain
	result, err := chain.Execute(context.Background(), candidates, mockRun)
	if err != nil {
		t.Fatalf("expected success after failover, got error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if result.Response.Content != "Hello from key2!" {
		t.Errorf("expected response from key2, got: %s", result.Response.Content)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls (1 fail + 1 success), got %d", callCount)
	}

	// Verify first attempt was recorded
	if len(result.Attempts) != 1 {
		t.Errorf("expected 1 failed attempt recorded, got %d", len(result.Attempts))
	}

	if result.Attempts[0].Reason != FailoverRateLimit {
		t.Errorf(
			"expected first attempt reason to be rate_limit, got: %s",
			result.Attempts[0].Reason,
		)
	}
}

// TestMultiKeyFailoverAllFail tests when all keys hit rate limit
func TestMultiKeyFailoverAllFail(t *testing.T) {
	cfg := ModelConfig{
		Primary:   "glm-4.7",
		Fallbacks: []string{"glm-4.7__key_1", "glm-4.7__key_2"},
	}

	candidates := ResolveCandidates(cfg, "zhipu")

	cooldown := NewCooldownTracker()
	chain := NewFallbackChain(cooldown, nil)

	// Mock run function: all calls fail with rate limit
	callCount := 0
	mockRun := func(ctx context.Context, provider, model string) (*LLMResponse, error) {
		callCount++
		return nil, errors.New("status: 429 - too many requests")
	}

	// Execute fallback chain
	result, err := chain.Execute(context.Background(), candidates, mockRun)

	if err == nil {
		t.Fatal("expected error when all keys fail, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result on failure, got: %v", result)
	}

	if callCount != 3 {
		t.Errorf("expected 3 calls (all fail), got %d", callCount)
	}

	// Verify error type
	var exhausted *FallbackExhaustedError
	if !errors.As(err, &exhausted) {
		t.Errorf("expected FallbackExhaustedError, got: %T - %v", err, err)
	}

	if len(exhausted.Attempts) != 3 {
		t.Errorf("expected 3 attempts in exhausted error, got %d", len(exhausted.Attempts))
	}
}

// TestMultiKeyFailoverCooldown tests that a key in cooldown is skipped
func TestMultiKeyFailoverCooldown(t *testing.T) {
	cfg := ModelConfig{
		Primary:   "glm-4.7",
		Fallbacks: []string{"glm-4.7__key_1"},
	}

	candidates := ResolveCandidates(cfg, "zhipu")

	cooldown := NewCooldownTracker()
	chain := NewFallbackChain(cooldown, nil)

	// Put the first model in cooldown (using ModelKey now, not just provider)
	cooldownKey := ModelKey(candidates[0].Provider, candidates[0].Model)
	cooldown.MarkFailure(cooldownKey, FailoverRateLimit)

	// Verify it's not available
	if cooldown.IsAvailable(cooldownKey) {
		t.Fatal("expected first model to be in cooldown")
	}

	// Mock run function: only second should be called
	callCount := 0
	calledProviders := []string{}
	mockRun := func(ctx context.Context, provider, model string) (*LLMResponse, error) {
		callCount++
		calledProviders = append(calledProviders, provider+"/"+model)
		return &LLMResponse{Content: "success"}, nil
	}

	result, err := chain.Execute(context.Background(), candidates, mockRun)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// First provider should have been skipped
	if callCount != 1 {
		t.Errorf("expected 1 call (first skipped due to cooldown), got %d", callCount)
	}

	// Should have called the second provider/model
	if len(calledProviders) != 1 ||
		calledProviders[0] != candidates[1].Provider+"/"+candidates[1].Model {
		t.Errorf("expected second model to be called, got: %v", calledProviders)
	}

	// Verify first attempt was recorded as skipped
	if len(result.Attempts) != 1 {
		t.Fatalf("expected 1 attempt (skipped), got %d", len(result.Attempts))
	}

	if !result.Attempts[0].Skipped {
		t.Error("expected first attempt to be marked as skipped")
	}
}

// TestMultiKeyFailoverWithFormatError tests that format errors are non-retriable
func TestMultiKeyFailoverWithFormatError(t *testing.T) {
	cfg := ModelConfig{
		Primary:   "glm-4.7",
		Fallbacks: []string{"glm-4.7__key_1"},
	}

	candidates := ResolveCandidates(cfg, "zhipu")

	cooldown := NewCooldownTracker()
	chain := NewFallbackChain(cooldown, nil)

	// Mock run function: first call fails with format error (bad request)
	callCount := 0
	mockRun := func(ctx context.Context, provider, model string) (*LLMResponse, error) {
		callCount++
		return nil, errors.New("invalid request format: tool_use.id missing")
	}

	// Execute fallback chain
	result, err := chain.Execute(context.Background(), candidates, mockRun)

	if err == nil {
		t.Fatal("expected error for format failure, got nil")
	}

	// Format errors should NOT trigger failover (non-retriable)
	// So we should only have 1 call
	if callCount != 1 {
		t.Errorf("expected 1 call (format error is non-retriable), got %d", callCount)
	}

	// Verify the error is a FailoverError with format reason
	var failoverErr *FailoverError
	if !errors.As(err, &failoverErr) {
		t.Errorf("expected FailoverError, got: %T - %v", err, err)
	}

	if failoverErr.Reason != FailoverFormat {
		t.Errorf("expected FailoverFormat reason, got: %s", failoverErr.Reason)
	}

	_ = result // result should be nil
}

// TestMultiKeyWithModelFallback tests multi-key failover combined with model fallback.
// This simulates the scenario: api_keys: ["k1", "k2"] + fallbacks: ["minimax"]
// Expected failover order: glm-4.7 (k1) → glm-4.7__key_1 (k2) → minimax
func TestMultiKeyWithModelFallback(t *testing.T) {
	// Simulate expanded config from:
	// { "model_name": "glm-4.7", "api_keys": ["k1", "k2"], "fallbacks": ["minimax"] }
	// After ExpandMultiKeyModels, primaryEntry.Fallbacks = ["glm-4.7__key_1", "minimax"]
	// Note: In production, "minimax" would be resolved via model lookup to "minimax/minimax"
	// In this test, we use the full format to avoid needing a lookup function.
	cfg := ModelConfig{
		Primary:   "glm-4.7",
		Fallbacks: []string{"glm-4.7__key_1", "minimax/minimax"},
	}

	candidates := ResolveCandidates(cfg, "zhipu")

	// Should have 3 candidates: glm-4.7 (zhipu), glm-4.7__key_1 (zhipu), minimax (minimax)
	if len(candidates) != 3 {
		t.Fatalf("expected 3 candidates, got %d: %v", len(candidates), candidates)
	}

	// Verify candidate order
	if candidates[0].Model != "glm-4.7" || candidates[0].Provider != "zhipu" {
		t.Errorf(
			"expected first candidate to be zhipu/glm-4.7, got: %s/%s",
			candidates[0].Provider,
			candidates[0].Model,
		)
	}
	if candidates[1].Model != "glm-4.7__key_1" || candidates[1].Provider != "zhipu" {
		t.Errorf(
			"expected second candidate to be zhipu/glm-4.7__key_1, got: %s/%s",
			candidates[1].Provider,
			candidates[1].Model,
		)
	}
	if candidates[2].Model != "minimax" || candidates[2].Provider != "minimax" {
		t.Errorf(
			"expected third candidate to be minimax/minimax, got: %s/%s",
			candidates[2].Provider,
			candidates[2].Model,
		)
	}

	cooldown := NewCooldownTracker()
	chain := NewFallbackChain(cooldown, nil)

	// Mock run function: first two fail, third succeeds (model fallback)
	callCount := 0
	calledModels := []string{}
	mockRun := func(ctx context.Context, provider, model string) (*LLMResponse, error) {
		callCount++
		calledModels = append(calledModels, provider+"/"+model)

		switch callCount {
		case 1:
			// k1: rate limit
			return nil, errors.New("status: 429 - rate limit")
		case 2:
			// k2: also rate limit (all zhipu keys exhausted)
			return nil, errors.New("status: 429 - rate limit")
		case 3:
			// minimax: success
			return &LLMResponse{Content: "success from minimax"}, nil
		default:
			return nil, errors.New("unexpected call")
		}
	}

	result, err := chain.Execute(context.Background(), candidates, mockRun)
	if err != nil {
		t.Fatalf("expected success after failover to model fallback, got error: %v", err)
	}

	if callCount != 3 {
		t.Errorf("expected 3 calls (k1 fail + k2 fail + minimax success), got %d", callCount)
	}

	if result.Response.Content != "success from minimax" {
		t.Errorf("expected response from minimax, got: %s", result.Response.Content)
	}

	// Verify call order
	if len(calledModels) != 3 {
		t.Fatalf("expected 3 called models, got %d", len(calledModels))
	}
	if calledModels[0] != "zhipu/glm-4.7" {
		t.Errorf("expected first call to zhipu/glm-4.7, got: %s", calledModels[0])
	}
	if calledModels[1] != "zhipu/glm-4.7__key_1" {
		t.Errorf("expected second call to zhipu/glm-4.7__key_1, got: %s", calledModels[1])
	}
	if calledModels[2] != "minimax/minimax" {
		t.Errorf("expected third call to minimax/minimax, got: %s", calledModels[2])
	}

	// Verify 2 failed attempts recorded
	if len(result.Attempts) != 2 {
		t.Errorf("expected 2 failed attempts, got %d", len(result.Attempts))
	}

	// Both should be rate limit
	for i, attempt := range result.Attempts {
		if attempt.Reason != FailoverRateLimit {
			t.Errorf("expected attempt %d to be rate_limit, got: %s", i, attempt.Reason)
		}
	}
}

// TestMultiKeyFailoverMixedErrors tests failover with different error types
func TestMultiKeyFailoverMixedErrors(t *testing.T) {
	cfg := ModelConfig{
		Primary:   "glm-4.7",
		Fallbacks: []string{"glm-4.7__key_1", "glm-4.7__key_2"},
	}

	candidates := ResolveCandidates(cfg, "zhipu")

	cooldown := NewCooldownTracker()
	chain := NewFallbackChain(cooldown, nil)

	// Mock run function: different errors for each key
	callCount := 0
	mockRun := func(ctx context.Context, provider, model string) (*LLMResponse, error) {
		callCount++
		switch callCount {
		case 1:
			// First: rate limit (retriable)
			return nil, errors.New("status: 429 - rate limit")
		case 2:
			// Second: timeout (retriable)
			return nil, errors.New("context deadline exceeded")
		case 3:
			// Third: success
			return &LLMResponse{Content: "success from key3"}, nil
		default:
			return nil, errors.New("unexpected call")
		}
	}

	result, err := chain.Execute(context.Background(), candidates, mockRun)
	if err != nil {
		t.Fatalf("expected success after 2 failovers, got error: %v", err)
	}

	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}

	// Verify both failed attempts were recorded
	if len(result.Attempts) != 2 {
		t.Errorf("expected 2 failed attempts, got %d", len(result.Attempts))
	}

	// First should be rate limit
	if result.Attempts[0].Reason != FailoverRateLimit {
		t.Errorf("expected first attempt to be rate_limit, got: %s", result.Attempts[0].Reason)
	}

	// Second should be timeout
	if result.Attempts[1].Reason != FailoverTimeout {
		t.Errorf("expected second attempt to be timeout, got: %s", result.Attempts[1].Reason)
	}
}
