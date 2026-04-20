package providers

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestCooldown_Persistence(t *testing.T) {
	tempDir := t.TempDir()
	persistPath := filepath.Join(tempDir, "cooldowns.json")

	now := time.Now()
	ct, current := newTestTracker(now)
	if err := ct.SetPersistencePath(persistPath); err != nil {
		t.Fatalf("SetPersistencePath failed: %v", err)
	}

	// 1. Mark a failure and verify it saves
	ct.MarkFailure("openai", FailoverRateLimit) // 1 min cooldown
	if ct.IsAvailable("openai") {
		t.Error("openai should be in cooldown")
	}

	if _, err := os.Stat(persistPath); os.IsNotExist(err) {
		t.Fatal("persistence file was not created")
	}

	// 2. Create a NEW tracker and load the file
	ct2, _ := newTestTracker(now)
	if err := ct2.SetPersistencePath(persistPath); err != nil {
		t.Fatalf("SetPersistencePath on second tracker failed: %v", err)
	}

	if ct2.IsAvailable("openai") {
		t.Error("newly loaded tracker should still have openai in cooldown")
	}
	if ct2.ErrorCount("openai") != 1 {
		t.Errorf("error count = %d, want 1", ct2.ErrorCount("openai"))
	}

	// 3. Mark success and verify it clears and persists
	ct2.MarkSuccess("openai")
	if !ct2.IsAvailable("openai") {
		t.Error("openai should be available after success")
	}

	ct3, _ := newTestTracker(now)
	if err := ct3.SetPersistencePath(persistPath); err != nil {
		t.Fatalf("SetPersistencePath on third tracker failed: %v", err)
	}
	if !ct3.IsAvailable("openai") {
		t.Error("fourth tracker should see openai as available after success was persisted")
	}

	// 4. Verify expiration filtering
	ct3.MarkFailure("anthropic", FailoverRateLimit) // 1 min cooldown
	*current = now.Add(2 * time.Minute)             // Advance time past expiration

	ct4, ct4Current := newTestTracker(*current) // ct4 sees the future
	if err := ct4.SetPersistencePath(persistPath); err != nil {
		t.Fatalf("SetPersistencePath on fourth tracker failed: %v", err)
	}

	// Since current time (2 min later) is past the 1 min cooldown, it should be filtered out on load
	if !ct4.IsAvailable("anthropic") {
		t.Error("anthropic should be available (expired cooldown filtered on load)")
	}

	// Verify that MarkFailure on ct4 uses the correct time
	ct4.MarkFailure("groq", FailoverRateLimit)
	if ct4.IsAvailable("groq") {
		t.Error("groq should be in cooldown on ct4")
	}
	_ = ct4Current // keep compiler happy
}

func newTestTracker(now time.Time) (*CooldownTracker, *time.Time) {
	current := now
	ct := NewCooldownTracker()
	ct.nowFunc = func() time.Time { return current }
	return ct, &current
}

func TestCooldown_InitiallyAvailable(t *testing.T) {
	ct := NewCooldownTracker()
	if !ct.IsAvailable("openai") {
		t.Error("new provider should be available")
	}
	if ct.ErrorCount("openai") != 0 {
		t.Error("new provider should have 0 errors")
	}
}

func TestCooldown_StandardEscalation(t *testing.T) {
	now := time.Now()
	ct, current := newTestTracker(now)

	// 1st error → 1 min cooldown
	ct.MarkFailure("openai", FailoverRateLimit)
	if ct.IsAvailable("openai") {
		t.Error("should be in cooldown after 1st error")
	}

	// Advance 61 seconds → available
	*current = now.Add(61 * time.Second)
	if !ct.IsAvailable("openai") {
		t.Error("should be available after 1 min cooldown")
	}

	// 2nd error → 5 min cooldown
	ct.MarkFailure("openai", FailoverRateLimit)
	*current = now.Add(61*time.Second + 4*time.Minute)
	if ct.IsAvailable("openai") {
		t.Error("should be in cooldown (5 min) after 2nd error")
	}
	*current = now.Add(61*time.Second + 6*time.Minute)
	if !ct.IsAvailable("openai") {
		t.Error("should be available after 5 min cooldown")
	}
}

func TestCooldown_StandardCap(t *testing.T) {
	// Verify formula: 1m, 5m, 25m, 1h, 1h, 1h...
	expected := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		25 * time.Minute,
		1 * time.Hour,
		1 * time.Hour,
	}

	for i, want := range expected {
		got := calculateStandardCooldown(i + 1)
		if got != want {
			t.Errorf("calculateStandardCooldown(%d) = %v, want %v", i+1, got, want)
		}
	}
}

func TestCooldown_BillingEscalation(t *testing.T) {
	now := time.Now()
	ct, current := newTestTracker(now)

	// 1st billing error → 5h cooldown
	ct.MarkFailure("openai", FailoverBilling)
	if ct.IsAvailable("openai") {
		t.Error("should be disabled after billing error")
	}

	// Advance 4h → still disabled
	*current = now.Add(4 * time.Hour)
	if ct.IsAvailable("openai") {
		t.Error("should still be disabled (5h cooldown)")
	}

	// Advance 5h + 1s → available
	*current = now.Add(5*time.Hour + 1*time.Second)
	if !ct.IsAvailable("openai") {
		t.Error("should be available after 5h billing cooldown")
	}
}

func TestCooldown_BillingCap(t *testing.T) {
	expected := []time.Duration{
		5 * time.Hour,
		10 * time.Hour,
		20 * time.Hour,
		24 * time.Hour,
		24 * time.Hour,
	}

	for i, want := range expected {
		got := calculateBillingCooldown(i + 1)
		if got != want {
			t.Errorf("calculateBillingCooldown(%d) = %v, want %v", i+1, got, want)
		}
	}
}

func TestCooldown_SuccessReset(t *testing.T) {
	ct := NewCooldownTracker()

	ct.MarkFailure("openai", FailoverRateLimit)
	ct.MarkFailure("openai", FailoverBilling)
	if ct.ErrorCount("openai") != 2 {
		t.Errorf("error count = %d, want 2", ct.ErrorCount("openai"))
	}

	ct.MarkSuccess("openai")
	if ct.ErrorCount("openai") != 0 {
		t.Errorf("error count after success = %d, want 0", ct.ErrorCount("openai"))
	}
	if !ct.IsAvailable("openai") {
		t.Error("should be available after success")
	}
	if ct.FailureCount("openai", FailoverRateLimit) != 0 {
		t.Error("failure counts should be reset after success")
	}
	if ct.FailureCount("openai", FailoverBilling) != 0 {
		t.Error("billing failure count should be reset after success")
	}
}

func TestCooldown_FailureWindowReset(t *testing.T) {
	now := time.Now()
	ct, current := newTestTracker(now)

	// 4 errors → 1h cooldown
	for range 4 {
		ct.MarkFailure("openai", FailoverRateLimit)
		*current = current.Add(2 * time.Second) // small advance between errors
	}
	if ct.ErrorCount("openai") != 4 {
		t.Errorf("error count = %d, want 4", ct.ErrorCount("openai"))
	}

	// Advance 25 hours (past 24h failure window)
	*current = now.Add(25 * time.Hour)

	// Next error should reset counters first, then increment to 1
	ct.MarkFailure("openai", FailoverRateLimit)
	if ct.ErrorCount("openai") != 1 {
		t.Errorf("error count after window reset = %d, want 1 (reset + 1)", ct.ErrorCount("openai"))
	}
}

func TestCooldown_PerReasonTracking(t *testing.T) {
	ct := NewCooldownTracker()

	ct.MarkFailure("openai", FailoverRateLimit)
	ct.MarkFailure("openai", FailoverRateLimit)
	ct.MarkFailure("openai", FailoverBilling)
	ct.MarkFailure("openai", FailoverAuth)

	if ct.FailureCount("openai", FailoverRateLimit) != 2 {
		t.Errorf("rate_limit count = %d, want 2", ct.FailureCount("openai", FailoverRateLimit))
	}
	if ct.FailureCount("openai", FailoverBilling) != 1 {
		t.Errorf("billing count = %d, want 1", ct.FailureCount("openai", FailoverBilling))
	}
	if ct.FailureCount("openai", FailoverAuth) != 1 {
		t.Errorf("auth count = %d, want 1", ct.FailureCount("openai", FailoverAuth))
	}
	if ct.ErrorCount("openai") != 4 {
		t.Errorf("total error count = %d, want 4", ct.ErrorCount("openai"))
	}
}

func TestCooldown_BillingTakesPrecedence(t *testing.T) {
	now := time.Now()
	ct, current := newTestTracker(now)

	// Standard cooldown (1 min) + billing disable (5h)
	ct.MarkFailure("openai", FailoverRateLimit) // 1 min cooldown
	ct.MarkFailure("openai", FailoverBilling)   // 5h disable

	// After 2 min: standard cooldown expired but billing still active
	*current = now.Add(2 * time.Minute)
	if ct.IsAvailable("openai") {
		t.Error("billing disable should take precedence over standard cooldown")
	}

	// After 5h + 1s: both expired
	*current = now.Add(5*time.Hour + 1*time.Second)
	if !ct.IsAvailable("openai") {
		t.Error("should be available after all cooldowns expire")
	}
}

func TestCooldown_CooldownRemaining(t *testing.T) {
	now := time.Now()
	ct, current := newTestTracker(now)

	// No failures → 0 remaining
	if ct.CooldownRemaining("openai") != 0 {
		t.Error("expected 0 remaining for new provider")
	}

	ct.MarkFailure("openai", FailoverRateLimit)

	*current = now.Add(30 * time.Second)
	remaining := ct.CooldownRemaining("openai")
	if remaining <= 0 || remaining > 1*time.Minute {
		t.Errorf("remaining = %v, expected ~30s", remaining)
	}
}

func TestCooldown_SuccessOnUnknownProvider(t *testing.T) {
	ct := NewCooldownTracker()
	// Should not panic
	ct.MarkSuccess("nonexistent")
	if !ct.IsAvailable("nonexistent") {
		t.Error("nonexistent provider should be available")
	}
}

func TestCooldown_ConcurrentAccess(t *testing.T) {
	ct := NewCooldownTracker()
	var wg sync.WaitGroup

	for range 100 {
		wg.Add(3)
		go func() {
			defer wg.Done()
			ct.MarkFailure("openai", FailoverRateLimit)
		}()
		go func() {
			defer wg.Done()
			ct.IsAvailable("openai")
		}()
		go func() {
			defer wg.Done()
			ct.MarkSuccess("openai")
		}()
	}

	wg.Wait()
	// If we got here without panic, concurrent access is safe
}

func TestCooldown_MultipleProviders(t *testing.T) {
	ct := NewCooldownTracker()

	ct.MarkFailure("openai", FailoverRateLimit)
	ct.MarkFailure("anthropic", FailoverBilling)

	if ct.IsAvailable("openai") {
		t.Error("openai should be in cooldown")
	}
	if ct.IsAvailable("anthropic") {
		t.Error("anthropic should be in cooldown")
	}
	// groq was never touched
	if !ct.IsAvailable("groq") {
		t.Error("groq should be available")
	}
}
