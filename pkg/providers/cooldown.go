package providers

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	defaultFailureWindow = 24 * time.Hour
)

// CooldownTracker manages per-provider cooldown state for the fallback chain.
// Thread-safe via sync.RWMutex. In-memory only (resets on restart).
type CooldownTracker struct {
	mu            sync.RWMutex
	entries       map[string]*cooldownEntry
	failureWindow time.Duration
	storagePath   string
	nowFunc       func() time.Time // for testing
}

type cooldownEntry struct {
	ErrorCount     int
	FailureCounts  map[FailoverReason]int
	CooldownEnd    time.Time      // standard cooldown expiry
	DisabledUntil  time.Time      // billing-specific disable expiry
	DisabledReason FailoverReason // reason for disable (billing)
	LastFailure    time.Time
}

// NewCooldownTracker creates a tracker with default 24h failure window.
func NewCooldownTracker(storagePath string) *CooldownTracker {
	ct := &CooldownTracker{
		entries:       make(map[string]*cooldownEntry),
		failureWindow: defaultFailureWindow,
		storagePath:   storagePath,
		nowFunc:       time.Now,
	}
	if storagePath != "" {
		ct.Load()
	}
	return ct
}

// MarkFailure records a failure for a provider and sets appropriate cooldown.
// Resets error counts if last failure was more than failureWindow ago.
func (ct *CooldownTracker) MarkFailure(provider string, reason FailoverReason) {
	ct.mu.Lock()
	now := ct.nowFunc()
	entry := ct.getOrCreate(provider)

	// 24h failure window reset: if no failure in failureWindow, reset counters.
	if !entry.LastFailure.IsZero() && now.Sub(entry.LastFailure) > ct.failureWindow {
		entry.ErrorCount = 0
		entry.FailureCounts = make(map[FailoverReason]int)
	}

	entry.ErrorCount++
	if entry.FailureCounts == nil {
		entry.FailureCounts = make(map[FailoverReason]int)
	}
	entry.FailureCounts[reason]++
	entry.LastFailure = now

	if reason == FailoverBilling {
		billingCount := entry.FailureCounts[FailoverBilling]
		entry.DisabledUntil = now.Add(calculateBillingCooldown(billingCount))
		entry.DisabledReason = FailoverBilling
	} else {
		entry.CooldownEnd = now.Add(calculateStandardCooldown(entry.ErrorCount))
	}

	// Capture state for saving outside the lock
	toSave := ct.copyEntriesLocked()
	ct.mu.Unlock()

	ct.persist(toSave)
}

// MarkSuccess resets all counters and cooldowns for a provider.
func (ct *CooldownTracker) MarkSuccess(provider string) {
	ct.mu.Lock()
	entry := ct.entries[provider]
	if entry == nil {
		ct.mu.Unlock()
		return
	}

	entry.ErrorCount = 0
	entry.FailureCounts = make(map[FailoverReason]int)
	entry.CooldownEnd = time.Time{}
	entry.DisabledUntil = time.Time{}
	entry.DisabledReason = ""

	// Capture state for saving outside the lock
	toSave := ct.copyEntriesLocked()
	ct.mu.Unlock()

	ct.persist(toSave)
}

// IsAvailable returns true if the provider is not in cooldown or disabled.
func (ct *CooldownTracker) IsAvailable(provider string) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	entry := ct.entries[provider]
	if entry == nil {
		return true
	}

	now := ct.nowFunc()

	// Billing disable takes precedence (longer cooldown).
	if !entry.DisabledUntil.IsZero() && now.Before(entry.DisabledUntil) {
		return false
	}

	// Standard cooldown.
	if !entry.CooldownEnd.IsZero() && now.Before(entry.CooldownEnd) {
		return false
	}

	return true
}

// CooldownRemaining returns how long until the provider becomes available.
// Returns 0 if already available.
func (ct *CooldownTracker) CooldownRemaining(provider string) time.Duration {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	entry := ct.entries[provider]
	if entry == nil {
		return 0
	}

	now := ct.nowFunc()
	var remaining time.Duration

	if !entry.DisabledUntil.IsZero() && now.Before(entry.DisabledUntil) {
		d := entry.DisabledUntil.Sub(now)
		if d > remaining {
			remaining = d
		}
	}

	if !entry.CooldownEnd.IsZero() && now.Before(entry.CooldownEnd) {
		d := entry.CooldownEnd.Sub(now)
		if d > remaining {
			remaining = d
		}
	}

	return remaining
}

// ErrorCount returns the current error count for a provider.
func (ct *CooldownTracker) ErrorCount(provider string) int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	entry := ct.entries[provider]
	if entry == nil {
		return 0
	}
	return entry.ErrorCount
}

// FailureCount returns the failure count for a specific reason.
func (ct *CooldownTracker) FailureCount(provider string, reason FailoverReason) int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	entry := ct.entries[provider]
	if entry == nil {
		return 0
	}
	return entry.FailureCounts[reason]
}

// Load reads cooldown state from disk.
func (ct *CooldownTracker) Load() {
	if ct.storagePath == "" {
		return
	}

	data, err := os.ReadFile(ct.storagePath)
	if err != nil {
		return // ignore missing file
	}

	var loaded map[string]*cooldownEntry
	if err := json.Unmarshal(data, &loaded); err != nil {
		return
	}

	ct.mu.Lock()
	defer ct.mu.Unlock()

	now := ct.nowFunc()
	for k, v := range loaded {
		// Only load entries that aren't fully expired yet
		if (!v.CooldownEnd.IsZero() && now.Before(v.CooldownEnd)) ||
			(!v.DisabledUntil.IsZero() && now.Before(v.DisabledUntil)) ||
			(!v.LastFailure.IsZero() && now.Sub(v.LastFailure) < ct.failureWindow) {
			if v.FailureCounts == nil {
				v.FailureCounts = make(map[FailoverReason]int)
			}
			ct.entries[k] = v
		}
	}
}

// Save writes cooldown state to disk.
func (ct *CooldownTracker) Save() {
	ct.mu.RLock()
	toSave := ct.copyEntriesLocked()
	ct.mu.RUnlock()

	ct.persist(toSave)
}

func (ct *CooldownTracker) copyEntriesLocked() map[string]*cooldownEntry {
	toSave := make(map[string]*cooldownEntry)
	now := ct.nowFunc()
	for k, v := range ct.entries {
		if v == nil {
			continue
		}
		// Only save entries that are still relevant
		if (!v.CooldownEnd.IsZero() && now.Before(v.CooldownEnd)) ||
			(!v.DisabledUntil.IsZero() && now.Before(v.DisabledUntil)) ||
			(!v.LastFailure.IsZero() && now.Sub(v.LastFailure) < ct.failureWindow) {

			// Deep copy the entry to avoid data races when serializing outside the lock
			copy := *v
			if v.FailureCounts != nil {
				copy.FailureCounts = make(map[FailoverReason]int)
				for r, c := range v.FailureCounts {
					copy.FailureCounts[r] = c
				}
			}
			toSave[k] = &copy
		}
	}
	return toSave
}

func (ct *CooldownTracker) persist(toSave map[string]*cooldownEntry) {
	if ct.storagePath == "" {
		return
	}

	if len(toSave) == 0 {
		os.Remove(ct.storagePath) // cleanup if empty
		return
	}

	data, err := json.MarshalIndent(toSave, "", "  ")
	if err != nil {
		return
	}

	os.MkdirAll(filepath.Dir(ct.storagePath), 0755)
	os.WriteFile(ct.storagePath, data, 0644)
}

func (ct *CooldownTracker) getOrCreate(provider string) *cooldownEntry {
	entry := ct.entries[provider]
	if entry == nil {
		entry = &cooldownEntry{
			FailureCounts: make(map[FailoverReason]int),
		}
		ct.entries[provider] = entry
	}
	return entry
}

// calculateStandardCooldown computes standard exponential backoff.
// Formula from OpenClaw: min(1h, 1min * 5^min(n-1, 3))
//
//	1 error  → 1 min
//	2 errors → 5 min
//	3 errors → 25 min
//	4+ errors → 1 hour (cap)
func calculateStandardCooldown(errorCount int) time.Duration {
	n := max(1, errorCount)
	exp := min(n-1, 3)
	ms := 60_000 * int(math.Pow(5, float64(exp)))
	ms = min(3_600_000, ms) // cap at 1 hour
	return time.Duration(ms) * time.Millisecond
}

// calculateBillingCooldown computes billing-specific exponential backoff.
// Formula from OpenClaw: min(24h, 5h * 2^min(n-1, 10))
//
//	1 error  → 5 hours
//	2 errors → 10 hours
//	3 errors → 20 hours
//	4+ errors → 24 hours (cap)
func calculateBillingCooldown(billingErrorCount int) time.Duration {
	const baseMs = 5 * 60 * 60 * 1000 // 5 hours
	const maxMs = 24 * 60 * 60 * 1000 // 24 hours

	n := max(1, billingErrorCount)
	exp := min(n-1, 10)
	raw := float64(baseMs) * math.Pow(2, float64(exp))
	ms := int(math.Min(float64(maxMs), raw))
	return time.Duration(ms) * time.Millisecond
}
