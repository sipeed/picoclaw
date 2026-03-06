package routing

// Classifier evaluates a feature set and returns a complexity score in [0, 1].
// A higher score indicates a more complex task that benefits from a heavy model.
// The score is compared against the configured threshold: score >= threshold selects
// the primary (heavy) model; score < threshold selects the light model.
//
// Classifier is an interface so that future implementations (ML-based, embedding-based,
// or any other approach) can be swapped in without changing routing infrastructure.
type Classifier interface {
	Score(f Features) float64
}

// RuleClassifier is the v1 implementation.
// It uses a weighted sum of structural signals with no external dependencies,
// no API calls, and sub-microsecond latency. The raw sum is capped at 1.0 so
// that the returned score always falls within the [0, 1] contract.
//
// Individual weights (multiple signals can fire simultaneously):
//
//	token > 200 (≈600 chars): 0.35  — very long prompts are almost always complex
//	token 50-200:             0.15  — medium length; may or may not be complex
//	code block present:       0.40  — coding tasks need the heavy model
//	tool calls > 3 (recent):  0.25  — dense tool usage signals an agentic workflow
//	tool calls 1-3 (recent):  0.10  — some tool activity
//	conversation depth > 10:  0.10  — long sessions carry implicit complexity
//	attachments present:      1.00  — hard gate; multi-modal always needs heavy model
//
// Default threshold is 0.35, so:
//   - Pure greetings / trivial Q&A:                 0.00 → light  ✓
//   - Medium prose message (50–200 tokens):          0.15 → light  ✓
//   - Message with code block:                       0.40 → heavy  ✓
//   - Long message (>200 tokens):                    0.35 → heavy  ✓
//   - Active tool session + medium message:          0.25 → light  (acceptable)
//   - Any message with an image/audio attachment:    1.00 → heavy  ✓
type RuleClassifier struct{}

// Score computes the complexity score for the given feature set.
// The returned value is in [0, 1]. Attachments short-circuit to 1.0.
func (c *RuleClassifier) Score(f Features) float64 {
	// Hard gate: multi-modal inputs always require the heavy model.
	if f.HasAttachments {
		return 1.0
	}

	var score float64

	// Token estimate — primary verbosity signal
	switch {
	case f.TokenEstimate > 200:
		score += 0.35
	case f.TokenEstimate > 50:
		score += 0.15
	}

	// Fenced code blocks — strongest indicator of a coding/technical task
	if f.CodeBlockCount > 0 {
		score += 0.40
	}

	// Recent tool call density — indicates an ongoing agentic workflow
	switch {
	case f.RecentToolCalls > 3:
		score += 0.25
	case f.RecentToolCalls > 0:
		score += 0.10
	}

	// Conversation depth — accumulated context implies compound task
	if f.ConversationDepth > 10 {
		score += 0.10
	}

	// Cap at 1.0 to honor the [0, 1] contract even when multiple signals fire
	// simultaneously (e.g., long message + code block + tool chain = 1.10 raw).
	if score > 1.0 {
		score = 1.0
	}
	return score
}
