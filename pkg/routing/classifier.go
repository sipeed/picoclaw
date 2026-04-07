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
// no API calls, and sub-microsecond latency. The raw sum is capped at 1.0.
//
// Signal weights (designed for a 0.65 threshold):
//
//	token > 500 (≈1500 chars):  0.30  — very long prompts add meaningful weight
//	token 150-500:              0.10  — medium length; not enough alone
//	code block present:         0.40  — coding tasks are the primary escalation signal
//	tool calls > 3 (recent):    0.35  — dense tool chain signals a complex workflow
//	tool calls 1-3 (recent):    0.10  — light tool activity
//	conversation depth > 10:    0.10  — long sessions carry implicit complexity
//	attachments (image/audio):  0.10  — images alone stay on Gemini Flash (vision);
//	                                     only escalate when combined with heavy signals
//
// Practical routing outcomes at threshold 0.65:
//   - Greeting / trivial Q&A:                        0.00 → Gemini  ✓
//   - Medium prose (150–500 tokens):                 0.10 → Gemini  ✓
//   - Long prose (>500 tokens):                      0.30 → Gemini  ✓
//   - Image only:                                    0.10 → Gemini  ✓  (vision)
//   - Image + medium text:                           0.20 → Gemini  ✓
//   - Short message with a code block:               0.40 → Gemini  ✓
//   - Long message with a code block:                0.70 → Claude  ✓
//   - Dense multi-tool chain (>3 calls):             0.35 → Gemini  (acceptable)
//   - Dense tool chain + code block:                 0.75 → Claude  ✓
type RuleClassifier struct{}

// Score computes the complexity score for the given feature set.
// The returned value is in [0, 1].
func (c *RuleClassifier) Score(f Features) float64 {
	var score float64

	// Attachments (image/audio): small weight so images alone stay on the
	// vision-capable light model (Gemini Flash).
	if f.HasAttachments {
		score += 0.10
	}

	// Token estimate — verbosity signal (higher bar than before)
	switch {
	case f.TokenEstimate > 500:
		score += 0.30
	case f.TokenEstimate > 150:
		score += 0.10
	}

	// Fenced code blocks — primary escalation signal
	if f.CodeBlockCount > 0 {
		score += 0.40
	}

	// Recent tool call density — indicates an ongoing agentic workflow
	switch {
	case f.RecentToolCalls > 3:
		score += 0.35
	case f.RecentToolCalls > 0:
		score += 0.10
	}

	// Conversation depth — accumulated context implies compound task
	if f.ConversationDepth > 10 {
		score += 0.10
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}
	return score
}
