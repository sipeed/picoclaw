package routing

import (
	"github.com/sipeed/picoclaw/pkg/providers"
)

// defaultThreshold is used when the config threshold is zero or negative.
// At 0.35 a message needs at least one strong signal (code block, long text,
// or an attachment) before the heavy model is chosen.
const defaultThreshold = 0.35

// RoutingTier defines a single tier for model routing.
type RoutingTier struct {
	Model     string
	Threshold float64
}

// RouterConfig holds the validated model routing settings.
// It mirrors config.RoutingConfig but lives in pkg/routing to keep the
// dependency graph simple: pkg/agent resolves config → routing, not the reverse.
type RouterConfig struct {
	Tiers []RoutingTier
}

// Router selects the appropriate model tier for each incoming message.
// It is safe for concurrent use from multiple goroutines.
type Router struct {
	cfg        RouterConfig
	classifier Classifier
}

// New creates a Router with the given config and the default RuleClassifier.
func New(cfg RouterConfig) *Router {
	return &Router{
		cfg:        cfg,
		classifier: &RuleClassifier{},
	}
}

// newWithClassifier creates a Router with a custom Classifier.
// Intended for unit tests that need to inject a deterministic scorer.
func newWithClassifier(cfg RouterConfig, c Classifier) *Router {
	return &Router{cfg: cfg, classifier: c}
}

// SelectModel returns the model to use for this conversation turn along with
// the computed complexity score (for logging and debugging).
//
// The router selects the tier whose threshold is <= score.
// If multiple tiers match, it prefers the one with the highest threshold.
// If no tier matches, it returns the primary model.
//
// The caller is responsible for resolving the returned model name into
// provider candidates.
func (r *Router) SelectModel(
	msg string,
	history []providers.Message,
	primaryModel string,
) (model string, usedLight bool, score float64) {
	features := ExtractFeatures(msg, history)
	score = r.classifier.Score(features)

	selectedModel := primaryModel
	maxMatchedThreshold := -1.0
	matched := false

	// Find the highest threshold that is <= score
	for _, tier := range r.cfg.Tiers {
		if score >= tier.Threshold && tier.Threshold > maxMatchedThreshold {
			selectedModel = tier.Model
			maxMatchedThreshold = tier.Threshold
			matched = true
		}
	}

	// usedLight is a bit of a legacy concept now, but we can set it to true if we didn't use primaryModel
	usedLight = matched && selectedModel != primaryModel

	return selectedModel, usedLight, score
}

// Tiers returns the configured routing tiers.
func (r *Router) Tiers() []RoutingTier {
	return r.cfg.Tiers
}
