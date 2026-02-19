package providers

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestAuthRotator_NextAvailable_RoundRobin(t *testing.T) {
	profiles := []AuthProfile{
		{ID: "p:0", APIKey: "key-0"},
		{ID: "p:1", APIKey: "key-1"},
		{ID: "p:2", APIKey: "key-2"},
	}

	cooldown := NewCooldownTracker()
	rotator := NewAuthRotator(profiles, cooldown)

	// First call should return earliest (all have same lastUsed = zero)
	p1 := rotator.NextAvailable()
	if p1 == nil {
		t.Fatal("expected a profile, got nil")
	}

	// Second call should return a different profile (p1 now has newest lastUsed)
	p2 := rotator.NextAvailable()
	if p2 == nil {
		t.Fatal("expected a profile, got nil")
	}
	if p2.ID == p1.ID {
		t.Errorf("round-robin should select different profile, got same: %s", p2.ID)
	}

	// Third call should return the remaining profile
	p3 := rotator.NextAvailable()
	if p3 == nil {
		t.Fatal("expected a profile, got nil")
	}
	if p3.ID == p1.ID || p3.ID == p2.ID {
		t.Errorf("expected third unique profile, got %s (p1=%s, p2=%s)", p3.ID, p1.ID, p2.ID)
	}
}

func TestAuthRotator_NextAvailable_SkipsCooldown(t *testing.T) {
	profiles := []AuthProfile{
		{ID: "p:0", APIKey: "key-0"},
		{ID: "p:1", APIKey: "key-1"},
	}

	cooldown := NewCooldownTracker()
	rotator := NewAuthRotator(profiles, cooldown)

	// Put p:0 in cooldown
	cooldown.MarkFailure("p:0", FailoverRateLimit)

	// Should skip p:0 and return p:1
	p := rotator.NextAvailable()
	if p == nil {
		t.Fatal("expected a profile, got nil")
	}
	if p.ID != "p:1" {
		t.Errorf("expected p:1 (p:0 in cooldown), got %s", p.ID)
	}
}

func TestAuthRotator_NextAvailable_AllInCooldown(t *testing.T) {
	profiles := []AuthProfile{
		{ID: "p:0", APIKey: "key-0"},
		{ID: "p:1", APIKey: "key-1"},
	}

	cooldown := NewCooldownTracker()
	rotator := NewAuthRotator(profiles, cooldown)

	// Put both in cooldown
	cooldown.MarkFailure("p:0", FailoverRateLimit)
	cooldown.MarkFailure("p:1", FailoverBilling)

	p := rotator.NextAvailable()
	if p != nil {
		t.Errorf("expected nil when all in cooldown, got %s", p.ID)
	}
}

func TestAuthRotator_MarkSuccess_ResetsCooldown(t *testing.T) {
	profiles := []AuthProfile{
		{ID: "p:0", APIKey: "key-0"},
	}

	cooldown := NewCooldownTracker()
	rotator := NewAuthRotator(profiles, cooldown)

	// Put in cooldown
	cooldown.MarkFailure("p:0", FailoverRateLimit)
	if cooldown.IsAvailable("p:0") {
		t.Fatal("should be in cooldown after failure")
	}

	// Mark success resets
	rotator.MarkSuccess("p:0")
	if !cooldown.IsAvailable("p:0") {
		t.Fatal("should be available after success")
	}
}

func TestAuthRotator_AvailableCount(t *testing.T) {
	profiles := []AuthProfile{
		{ID: "p:0", APIKey: "key-0"},
		{ID: "p:1", APIKey: "key-1"},
		{ID: "p:2", APIKey: "key-2"},
	}

	cooldown := NewCooldownTracker()
	rotator := NewAuthRotator(profiles, cooldown)

	if rotator.AvailableCount() != 3 {
		t.Errorf("expected 3 available, got %d", rotator.AvailableCount())
	}

	cooldown.MarkFailure("p:1", FailoverRateLimit)
	if rotator.AvailableCount() != 2 {
		t.Errorf("expected 2 available, got %d", rotator.AvailableCount())
	}
}

func TestBuildAuthProfiles(t *testing.T) {
	keys := []string{"sk-key1", "sk-key2", "sk-key3"}
	profiles := BuildAuthProfiles("openrouter", keys)

	if len(profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(profiles))
	}
	if profiles[0].ID != "openrouter:0" {
		t.Errorf("profiles[0].ID = %q, want %q", profiles[0].ID, "openrouter:0")
	}
	if profiles[2].APIKey != "sk-key3" {
		t.Errorf("profiles[2].APIKey = %q, want %q", profiles[2].APIKey, "sk-key3")
	}
}

// mockRotatingProvider tracks which provider was called.
type mockRotatingProvider struct {
	apiKey    string
	callCount int
	mu        sync.Mutex
	failErr   error
}

func (m *mockRotatingProvider) Chat(_ context.Context, _ []Message, _ []ToolDefinition, _ string, _ map[string]interface{}) (*LLMResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	if m.failErr != nil {
		return nil, m.failErr
	}
	return &LLMResponse{Content: "ok from " + m.apiKey}, nil
}

func (m *mockRotatingProvider) GetDefaultModel() string { return "mock" }

func TestAuthRotatingProvider_RotatesOnSuccess(t *testing.T) {
	profiles := []AuthProfile{
		{ID: "p:0", APIKey: "key-0"},
		{ID: "p:1", APIKey: "key-1"},
	}

	providers := make(map[string]*mockRotatingProvider)
	cooldown := NewCooldownTracker()
	factory := func(apiKey string) LLMProvider {
		p := &mockRotatingProvider{apiKey: apiKey}
		providers[apiKey] = p
		return p
	}

	rp := NewAuthRotatingProvider(profiles, cooldown, factory)

	// First call
	resp, err := rp.Chat(context.Background(), nil, nil, "model", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	// Second call should use different provider (round-robin)
	_, err = rp.Chat(context.Background(), nil, nil, "model", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both providers should have been called once
	p0 := providers["key-0"]
	p1 := providers["key-1"]
	if p0.callCount+p1.callCount != 2 {
		t.Errorf("expected 2 total calls, got %d + %d", p0.callCount, p1.callCount)
	}
	if p0.callCount == 0 || p1.callCount == 0 {
		t.Errorf("expected both providers called, got p0=%d, p1=%d", p0.callCount, p1.callCount)
	}
}

func TestAuthRotatingProvider_MarksFailure(t *testing.T) {
	profiles := []AuthProfile{
		{ID: "p:0", APIKey: "key-0"},
		{ID: "p:1", APIKey: "key-1"},
	}

	cooldown := NewCooldownTracker()
	factory := func(apiKey string) LLMProvider {
		return &mockRotatingProvider{
			apiKey:  apiKey,
			failErr: fmt.Errorf("429 too many requests"),
		}
	}

	rp := NewAuthRotatingProvider(profiles, cooldown, factory)

	// First call fails — should mark p:0 failure
	_, err := rp.Chat(context.Background(), nil, nil, "model", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	// p:0 should now be in cooldown
	if cooldown.IsAvailable("p:0") {
		t.Error("p:0 should be in cooldown after rate limit failure")
	}

	// p:1 should still be available
	if !cooldown.IsAvailable("p:1") {
		t.Error("p:1 should still be available")
	}
}

func TestAuthRotatingProvider_AllInCooldown(t *testing.T) {
	profiles := []AuthProfile{
		{ID: "p:0", APIKey: "key-0"},
	}

	cooldown := NewCooldownTracker()
	cooldown.MarkFailure("p:0", FailoverRateLimit)

	factory := func(apiKey string) LLMProvider {
		return &mockRotatingProvider{apiKey: apiKey}
	}

	rp := NewAuthRotatingProvider(profiles, cooldown, factory)

	_, err := rp.Chat(context.Background(), nil, nil, "model", nil)
	if err == nil {
		t.Fatal("expected error when all profiles in cooldown")
	}
	if err.Error() != "all auth profiles in cooldown (1 total)" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAuthRotatingProvider_SingleKey_NoCooldownRotation(t *testing.T) {
	profiles := []AuthProfile{
		{ID: "p:0", APIKey: "key-only"},
	}

	cooldown := NewCooldownTracker()
	factory := func(apiKey string) LLMProvider {
		return &mockRotatingProvider{apiKey: apiKey}
	}

	rp := NewAuthRotatingProvider(profiles, cooldown, factory)

	// Multiple calls should all succeed using the single key
	for i := 0; i < 5; i++ {
		_, err := rp.Chat(context.Background(), nil, nil, "model", nil)
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}
}

func TestAuthRotator_ConcurrentAccess(t *testing.T) {
	profiles := []AuthProfile{
		{ID: "p:0", APIKey: "key-0"},
		{ID: "p:1", APIKey: "key-1"},
		{ID: "p:2", APIKey: "key-2"},
	}

	cooldown := NewCooldownTracker()
	rotator := NewAuthRotator(profiles, cooldown)

	var wg sync.WaitGroup
	seen := sync.Map{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p := rotator.NextAvailable()
			if p != nil {
				seen.Store(p.ID, true)
			}
		}()
	}

	wg.Wait()

	// All profiles should have been used
	count := 0
	seen.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	if count != 3 {
		t.Errorf("expected all 3 profiles used concurrently, got %d", count)
	}
}

func TestAuthRotator_BillingCooldown_LongerDuration(t *testing.T) {
	profiles := []AuthProfile{
		{ID: "p:0", APIKey: "key-0"},
		{ID: "p:1", APIKey: "key-1"},
	}

	cooldown := NewCooldownTracker()
	rotator := NewAuthRotator(profiles, cooldown)

	// Mark billing failure (should have 5h cooldown)
	rotator.MarkFailure("p:0", FailoverBilling)

	remaining := cooldown.CooldownRemaining("p:0")
	// Billing cooldown should be >= 4.5 hours (5h minus some time elapsed)
	if remaining < 4*time.Hour {
		t.Errorf("billing cooldown should be ~5h, got %v", remaining)
	}

	// Standard failure should have much shorter cooldown
	rotator.MarkFailure("p:1", FailoverRateLimit)
	remaining2 := cooldown.CooldownRemaining("p:1")
	if remaining2 > 2*time.Minute {
		t.Errorf("standard cooldown should be ~1min, got %v", remaining2)
	}
}
