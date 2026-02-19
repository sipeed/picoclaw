package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// AuthProfile represents a single API key with rotation metadata.
type AuthProfile struct {
	ID     string // unique identifier (e.g. "openrouter:0")
	APIKey string
}

// AuthRotator manages round-robin selection across multiple API keys,
// with per-key cooldown tracking via CooldownTracker.
type AuthRotator struct {
	profiles []AuthProfile
	cooldown *CooldownTracker
	mu       sync.Mutex
	lastUsed map[string]time.Time
}

// NewAuthRotator creates a rotator for the given profiles.
// Uses the provided CooldownTracker for per-key cooldown state.
func NewAuthRotator(profiles []AuthProfile, cooldown *CooldownTracker) *AuthRotator {
	lastUsed := make(map[string]time.Time, len(profiles))
	for _, p := range profiles {
		lastUsed[p.ID] = time.Time{} // never used
	}
	return &AuthRotator{
		profiles: profiles,
		cooldown: cooldown,
		lastUsed: lastUsed,
	}
}

// NextAvailable returns the best available profile using round-robin
// (oldest lastUsed first), skipping profiles in cooldown.
// Returns nil if all profiles are in cooldown.
func (r *AuthRotator) NextAvailable() *AuthProfile {
	r.mu.Lock()
	defer r.mu.Unlock()

	var best *AuthProfile
	var bestTime time.Time
	first := true

	for i := range r.profiles {
		p := &r.profiles[i]
		if !r.cooldown.IsAvailable(p.ID) {
			continue
		}
		lu := r.lastUsed[p.ID]
		if first || lu.Before(bestTime) {
			best = p
			bestTime = lu
			first = false
		}
	}

	if best != nil {
		r.lastUsed[best.ID] = time.Now()
	}
	return best
}

// MarkFailure records a failure for a specific profile.
func (r *AuthRotator) MarkFailure(profileID string, reason FailoverReason) {
	r.cooldown.MarkFailure(profileID, reason)
	logger.WarnCF("auth_rotation", "Profile marked as failed", map[string]interface{}{
		"profile_id": profileID,
		"reason":     string(reason),
		"remaining":  r.cooldown.CooldownRemaining(profileID).Round(time.Second).String(),
	})
}

// MarkSuccess resets counters for a specific profile.
func (r *AuthRotator) MarkSuccess(profileID string) {
	r.cooldown.MarkSuccess(profileID)
}

// AvailableCount returns the number of profiles not in cooldown.
func (r *AuthRotator) AvailableCount() int {
	count := 0
	for _, p := range r.profiles {
		if r.cooldown.IsAvailable(p.ID) {
			count++
		}
	}
	return count
}

// ProfileCount returns the total number of profiles.
func (r *AuthRotator) ProfileCount() int {
	return len(r.profiles)
}

// AuthRotatingProvider wraps multiple LLM providers (one per API key)
// and rotates between them using AuthRotator.
type AuthRotatingProvider struct {
	providers map[string]LLMProvider // profileID -> provider
	rotator   *AuthRotator
	model     string // default model from first provider
}

// NewAuthRotatingProvider creates a rotating provider.
// factory is called once per profile to create the underlying provider.
func NewAuthRotatingProvider(
	profiles []AuthProfile,
	cooldown *CooldownTracker,
	factory func(apiKey string) LLMProvider,
) *AuthRotatingProvider {
	providerMap := make(map[string]LLMProvider, len(profiles))
	var defaultModel string
	for _, p := range profiles {
		prov := factory(p.APIKey)
		providerMap[p.ID] = prov
		if defaultModel == "" {
			defaultModel = prov.GetDefaultModel()
		}
	}

	rotator := NewAuthRotator(profiles, cooldown)

	logger.InfoCF("auth_rotation", "Auth rotation initialized", map[string]interface{}{
		"profiles": len(profiles),
	})

	return &AuthRotatingProvider{
		providers: providerMap,
		rotator:   rotator,
		model:     defaultModel,
	}
}

// Chat selects the best available profile and delegates to its provider.
// On failure, marks the profile and returns the error (FallbackChain handles retry).
func (p *AuthRotatingProvider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	opts map[string]interface{},
) (*LLMResponse, error) {
	profile := p.rotator.NextAvailable()
	if profile == nil {
		return nil, fmt.Errorf("all auth profiles in cooldown (%d total)", p.rotator.ProfileCount())
	}

	provider := p.providers[profile.ID]
	resp, err := provider.Chat(ctx, messages, tools, model, opts)

	if err != nil {
		// Classify and record failure against this specific profile.
		if failErr := ClassifyError(err, profile.ID, model); failErr != nil && failErr.IsRetriable() {
			p.rotator.MarkFailure(profile.ID, failErr.Reason)
		}
		return nil, err
	}

	p.rotator.MarkSuccess(profile.ID)
	return resp, nil
}

// GetDefaultModel returns the default model from the underlying providers.
func (p *AuthRotatingProvider) GetDefaultModel() string {
	return p.model
}

// BuildAuthProfiles creates AuthProfile entries from a list of API keys.
// Profile IDs follow the pattern "provider:N" (e.g. "openrouter:0").
func BuildAuthProfiles(providerName string, apiKeys []string) []AuthProfile {
	profiles := make([]AuthProfile, len(apiKeys))
	for i, key := range apiKeys {
		profiles[i] = AuthProfile{
			ID:     fmt.Sprintf("%s:%d", providerName, i),
			APIKey: key,
		}
	}
	return profiles
}
