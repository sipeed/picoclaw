package seahorse

// Short-term memory configuration constants — all are experience-based defaults.

const (
	// OrdinalStep is the gap between ordinals in context_items.
	// Insert at midpoint; resequence only when precision exhausted.
	OrdinalStep = 100

	// ContextThreshold is the compaction trigger for the context window.
	ContextThreshold      float64 = 0.75 // Compact at 75% of context window
	FreshTailCount        int     = 32   // Recent messages protected from compaction (deprecated: use token-based tail)

	// Tail protection constants (from OpenCode's proven approach)
	MinPreserveRecentTokens = 2000 // Never preserve less than this many tokens
	MaxPreserveRecentTokens = 8000 // Never preserve more than this many tokens
	PreserveRecentRatio     = 0.25 // 25% of usable context for fresh tail
	DefaultTailTurns        = 2    // Minimum number of turns to preserve

	// LeafMinFanout is the fanout parameter.
	LeafMinFanout          int = 8 // Min messages per leaf summary
	CondensedMinFanout     int = 4 // Min summaries per condensed
	CondensedMinFanoutHard int = 2 // Min for forced compaction

	// LeafChunkTokens is the token target.
	LeafChunkTokens       int = 20000 // Max tokens per leaf chunk
	LeafTargetTokens      int = 1200  // Target tokens for leaf summaries
	CondensedTargetTokens int = 2000  // Target tokens for condensed summaries
	MaxExpandTokens       int = 4000  // Token cap for expansion queries

	// MaxCompactIterations caps CompactUntilUnder to prevent infinite loops.
	// Each iteration reduces ~4x tokens via leaf (8:1) or condensed (4:1) compaction.
	// With a 200k token context window and 75% threshold, ~20 iterations is enough
	// for any realistic scenario. If exceeded, the issue is logged as a warning.
	MaxCompactIterations int = 20
)
