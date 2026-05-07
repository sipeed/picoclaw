package seahorse

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tokenizer"
)

// CompactInput controls compaction behavior.
type CompactInput struct {
	Budget *int // Token budget override
	Force  bool // Force compaction even if below threshold
}

// CompactResult describes what was compacted.
type CompactResult struct {
	SummariesCreated   []string `json:"summariesCreated"`
	TokensSaved        int      `json:"tokensSaved"`
	LeafSummaries      int      `json:"leafSummaries"`
	CondensedSummaries int      `json:"condensedSummaries"`
}

// NeedsCompaction returns true if context tokens >= ContextThreshold × contextWindow.
func (e *CompactionEngine) NeedsCompaction(ctx context.Context, convID int64, contextWindow int) (bool, error) {
	tokens, err := e.store.GetContextTokenCount(ctx, convID)
	if err != nil {
		return false, fmt.Errorf("get token count: %w", err)
	}
	threshold := int(float64(contextWindow) * ContextThreshold)
	return tokens >= threshold, nil
}

// Close cancels the shutdown context, stopping async goroutines.
func (e *CompactionEngine) Close() {
	if e.shutdownCancel != nil {
		e.shutdownCancel()
	}
}

// Compact runs leaf compaction (sync) and optionally condensed compaction.
func (e *CompactionEngine) Compact(ctx context.Context, convID int64, input CompactInput) (*CompactResult, error) {
	result := &CompactResult{}

	// Phase 1: leaf compaction (synchronous, every turn)
	summaryID, err := e.compactLeaf(ctx, convID)
	if err != nil {
		return nil, fmt.Errorf("compact leaf: %w", err)
	}
	if summaryID != nil {
		result.SummariesCreated = append(result.SummariesCreated, *summaryID)
		result.LeafSummaries++
		logger.InfoCF("seahorse", "compact: leaf", map[string]any{
			"conv_id":    convID,
			"summary_id": *summaryID,
		})
	}

	// Phase 2: condensed compaction if over threshold
	tokensBefore, _ := e.store.GetContextTokenCount(ctx, convID)
	var budget int
	if input.Budget != nil {
		budget = *input.Budget
		if budget == 0 {
			logger.ErrorCF("seahorse", "Compact: budget is 0, this should not happen", map[string]any{
				"conv_id": convID,
			})
		}
	} else {
		budget = int(float64(tokensBefore) * ContextThreshold)
	}

	if input.Force || (tokensBefore > budget && budget > 0) {
		// Launch async condensed compaction with dedup
		if _, loaded := e.condensing.LoadOrStore(convID, struct{}{}); !loaded {
			go func() {
				defer e.condensing.Delete(convID)
				e.runCondensedLoop(e.shutdownCtx, convID)
			}()
		}
	}

	tokensAfter, _ := e.store.GetContextTokenCount(ctx, convID)
	if tokensAfter < tokensBefore {
		result.TokensSaved = tokensBefore - tokensAfter
	}

	return result, nil
}

// CompactUntilUnder aggressively compacts until context is under budget.
func (e *CompactionEngine) CompactUntilUnder(ctx context.Context, convID int64, budget int) (*CompactResult, error) {
	result := &CompactResult{}
	prevTokens := 0
	logger.InfoCF("seahorse", "compact_until_under: start", map[string]any{"conv_id": convID, "budget": budget})

	for iter := 0; iter < MaxCompactIterations; iter++ {
		tokens, err := e.store.GetContextTokenCount(ctx, convID)
		if err != nil {
			return result, fmt.Errorf("get tokens: %w", err)
		}
		if tokens <= budget {
			logger.InfoCF("seahorse", "compact_until_under: done", map[string]any{
				"conv_id":   convID,
				"budget":    budget,
				"tokens":    tokens,
				"leaf":      result.LeafSummaries,
				"condensed": result.CondensedSummaries,
			})
			return result, nil
		}

		// Try leaf first
		summaryID, err := e.compactLeaf(ctx, convID, true)
		if err != nil {
			return result, err
		}
		if summaryID != nil {
			result.SummariesCreated = append(result.SummariesCreated, *summaryID)
			result.LeafSummaries++
			logger.InfoCF("seahorse", "compact_until_under: leaf", map[string]any{
				"conv_id":    convID,
				"summary_id": *summaryID,
			})
			continue
		}

		// Try condensed with forced fanout
		condensedID, err := e.compactCondensed(ctx, convID)
		if err != nil {
			return result, err
		}
		if condensedID != nil {
			result.SummariesCreated = append(result.SummariesCreated, *condensedID)
			result.CondensedSummaries++
			logger.InfoCF("seahorse", "compact_until_under: condensed", map[string]any{
				"conv_id":    convID,
				"summary_id": *condensedID,
			})
			continue
		}

		// No progress
		newTokens, _ := e.store.GetContextTokenCount(ctx, convID)
		if newTokens >= prevTokens {
			logger.WarnCF("seahorse", "compact_until_under: no progress", map[string]any{
				"conv_id": convID,
				"tokens":  newTokens,
			})
			return result, nil
		}
		prevTokens = newTokens
	}

	// Safety cap exceeded — see MaxCompactIterations doc for rationale.
	logger.WarnCF("seahorse", "compact_until_under: exceeded max iterations", map[string]any{
		"conv_id":    convID,
		"budget":     budget,
		"iterations": MaxCompactIterations,
		"tokens":     prevTokens,
	})
	return result, nil
}

// compactLeaf compresses the oldest contiguous message chunk into a leaf summary.
// When force is true, FreshTailCount protection is bypassed (used by CompactUntilUnder).
func (e *CompactionEngine) compactLeaf(ctx context.Context, convID int64, force ...bool) (*string, error) {
	items, err := e.store.GetContextItems(ctx, convID)
	if err != nil {
		return nil, err
	}

	// Find oldest contiguous message chunk outside fresh tail
	msgCount := 0
	msgTokens := 0
	for _, item := range items {
		if item.ItemType == "message" {
			msgCount++
			msgTokens += item.TokenCount
		}
	}

	// Trigger if either message count or token threshold is met
	if msgCount < LeafMinFanout && msgTokens < LeafChunkTokens {
		return nil, nil
	}

	// Calculate fresh tail boundary (bypass when forced)
	useForce := len(force) > 0 && force[0]
	tailStartIdx := len(items) - FreshTailCount
	if useForce {
		tailStartIdx = len(items) // allow compacting everything
	}
	if tailStartIdx < 0 {
		tailStartIdx = 0
	}

	// Find oldest contiguous message chunk, accumulating up to LeafChunkTokens
	var chunk []ContextItem
	chunkStart := -1
	chunkEnd := -1
	accumTokens := 0
	for i := 0; i < tailStartIdx; i++ {
		if items[i].ItemType == "message" {
			if chunkStart == -1 {
				chunkStart = i
			}
			chunkEnd = i
			accumTokens += items[i].TokenCount
			// Stop accumulating once we reach the token budget
			if accumTokens >= LeafChunkTokens {
				break
			}
		} else {
			// Non-message breaks the chunk
			if chunkStart != -1 && (chunkEnd-chunkStart+1) >= LeafMinFanout {
				break
			}
			chunkStart = -1
			chunkEnd = -1
			accumTokens = 0
		}
	}

	if chunkStart == -1 || (chunkEnd-chunkStart+1) < LeafMinFanout {
		return nil, nil
	}

	chunk = items[chunkStart : chunkEnd+1]

	// Collect messages for the chunk
	var messages []Message
	for _, item := range chunk {
		msg, innerErr := e.store.GetMessageByID(ctx, item.MessageID)
		if innerErr != nil {
			return nil, innerErr
		}
		messages = append(messages, *msg)
	}

	// Get prior summaries for context
	priorSummary := ""
	priorCount := 0
	for i := chunkStart - 1; i >= 0 && priorCount < 2; i-- {
		if items[i].ItemType == "summary" {
			sum, innerErr2 := e.store.GetSummary(ctx, items[i].SummaryID)
			if innerErr2 == nil {
				priorSummary = sum.Content + "\n" + priorSummary
				priorCount++
			}
		}
	}

	// Generate summary
	content, err := e.generateLeafSummary(ctx, messages, priorSummary)
	if err != nil {
		return nil, err
	}

	// Create summary in store
	tokenCount := tokenizer.EstimateMessageTokens(providers.Message{Content: content})

	var earliestAt, latestAt *time.Time
	if len(messages) > 0 {
		earliestAt = &messages[0].CreatedAt
		latestAt = &messages[len(messages)-1].CreatedAt
	}

	summary, err := e.store.CreateSummary(ctx, CreateSummaryInput{
		ConversationID:      convID,
		Kind:                SummaryKindLeaf,
		Depth:               0,
		Content:             content,
		TokenCount:          tokenCount,
		EarliestAt:          earliestAt,
		LatestAt:            latestAt,
		SourceMessageTokens: sumMessageTokens(messages),
	})
	if err != nil {
		return nil, err
	}

	// Link to source messages
	msgIDs := make([]int64, len(messages))
	for i, m := range messages {
		msgIDs[i] = m.ID
	}
	if err := e.store.LinkSummaryToMessages(ctx, summary.SummaryID, msgIDs); err != nil {
		return nil, err
	}

	// Replace context range with summary
	if err := e.store.ReplaceContextRangeWithSummary(
		ctx, convID, chunk[0].Ordinal, chunk[len(chunk)-1].Ordinal, summary.SummaryID,
	); err != nil {
		return nil, err
	}

	return &summary.SummaryID, nil
}

// compactCondensed compresses multiple summaries into one higher-level summary.
func (e *CompactionEngine) compactCondensed(ctx context.Context, convID int64) (*string, error) {
	// Try ordinal-aware selection first (respects consecutive ordering)
	var candidates []Summary

	depths, err := e.store.GetDistinctDepthsInContext(ctx, convID, 0)
	if err != nil {
		return nil, err
	}
	for _, depth := range depths {
		var chunkAtDepth []Summary
		var err2 error
		chunkAtDepth, err2 = e.selectOldestChunkAtDepth(ctx, convID, depth)
		if err2 != nil {
			continue
		}
		if len(chunkAtDepth) > 0 {
			candidates = chunkAtDepth
			break
		}
	}

	// Fallback to depth-grouping selection
	if len(candidates) == 0 {
		candidates, err = e.selectShallowestCondensationCandidate(ctx, convID, false)
		if err != nil {
			return nil, err
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	// Generate condensed summary
	content, err := e.generateCondensedSummary(ctx, candidates)
	if err != nil {
		return nil, err
	}

	// Merge metadata
	maxDepth := 0
	descendantCount := 0
	descendantTokenCount := 0
	sourceMessageTokens := 0
	var earliestAt, latestAt *time.Time

	parentIDs := make([]string, len(candidates))
	for i, c := range candidates {
		parentIDs[i] = c.SummaryID
		if c.Depth > maxDepth {
			maxDepth = c.Depth
		}
		descendantCount += c.DescendantCount + 1
		descendantTokenCount += c.TokenCount + c.DescendantTokenCount
		sourceMessageTokens += c.SourceMessageTokenCount
		if c.EarliestAt != nil {
			if earliestAt == nil || c.EarliestAt.Before(*earliestAt) {
				earliestAt = c.EarliestAt
			}
		}
		if c.LatestAt != nil {
			if latestAt == nil || c.LatestAt.After(*latestAt) {
				latestAt = c.LatestAt
			}
		}
	}

	tokenCount := tokenizer.EstimateMessageTokens(providers.Message{Content: content})

	summary, err := e.store.CreateSummary(ctx, CreateSummaryInput{
		ConversationID:       convID,
		Kind:                 SummaryKindCondensed,
		Depth:                maxDepth + 1,
		Content:              content,
		TokenCount:           tokenCount,
		EarliestAt:           earliestAt,
		LatestAt:             latestAt,
		DescendantCount:      descendantCount,
		DescendantTokenCount: descendantTokenCount,
		SourceMessageTokens:  sourceMessageTokens,
		ParentIDs:            parentIDs,
	})
	if err != nil {
		return nil, err
	}

	// Find the ordinal range for the candidate summaries in context
	items, err := e.store.GetContextItems(ctx, convID)
	if err != nil {
		return nil, err
	}

	candidateSet := make(map[string]bool)
	for _, c := range candidates {
		candidateSet[c.SummaryID] = true
	}

	startOrd := -1
	endOrd := -1
	hasNonCandidate := false
	for _, item := range items {
		if item.ItemType == "summary" && candidateSet[item.SummaryID] {
			if startOrd == -1 {
				startOrd, endOrd = item.Ordinal, item.Ordinal
			} else {
				// Check for non-candidate items between endOrd and current ordinal
				for _, it := range items {
					if it.Ordinal > endOrd && it.Ordinal <= item.Ordinal {
						if it.ItemType != "summary" || !candidateSet[it.SummaryID] {
							hasNonCandidate = true
							break
						}
					}
				}
				if hasNonCandidate {
					break
				}
				if item.Ordinal < startOrd {
					startOrd = item.Ordinal
				}
				if item.Ordinal > endOrd {
					endOrd = item.Ordinal
				}
			}
		}
	}

	if startOrd == -1 || endOrd == -1 {
		return nil, nil
	}

	// Collect candidate summary IDs
	candidateIDs := make([]string, 0, len(candidates))
	for _, c := range candidates {
		candidateIDs = append(candidateIDs, c.SummaryID)
	}

	if hasNonCandidate {
		// Use safe per-item deletion to avoid deleting non-candidate items
		if err := e.store.ReplaceContextItemsWithSummary(ctx, convID, candidateIDs, summary.SummaryID); err != nil {
			return nil, err
		}
	} else {
		// Candidates are consecutive, use efficient range deletion
		if err := e.store.ReplaceContextRangeWithSummary(ctx, convID, startOrd, endOrd, summary.SummaryID); err != nil {
			return nil, err
		}
	}

	return &summary.SummaryID, nil
}

// selectShallowestCondensationCandidate finds the shallowest consecutive summary group.
func (e *CompactionEngine) selectShallowestCondensationCandidate(
	ctx context.Context, convID int64, forced bool,
) ([]Summary, error) {
	items, err := e.store.GetContextItems(ctx, convID)
	if err != nil {
		return nil, err
	}

	// Group by depth, find consecutive runs
	tailStartIdx := len(items) - FreshTailCount
	if tailStartIdx < 0 {
		tailStartIdx = 0
	}

	minFanout := CondensedMinFanout
	if forced {
		minFanout = CondensedMinFanoutHard
	}

	// Track depth groups
	depthGroups := make(map[int][]ContextItem)
	for i := 0; i < tailStartIdx; i++ {
		item := items[i]
		if item.ItemType != "summary" {
			continue
		}
		sum, err := e.store.GetSummary(ctx, item.SummaryID)
		if err != nil {
			continue
		}
		depthGroups[sum.Depth] = append(depthGroups[sum.Depth], item)
	}

	// Find shallowest depth with enough candidates
	// Collect all depths and sort to handle non-consecutive depths
	var depths []int
	for depth := range depthGroups {
		depths = append(depths, depth)
	}
	sort.Ints(depths)

	for _, depth := range depths {
		group := depthGroups[depth]
		if len(group) >= minFanout {
			// Load summaries
			var result []Summary
			for _, item := range group[:minFanout] {
				sum, err := e.store.GetSummary(ctx, item.SummaryID)
				if err != nil {
					continue
				}
				result = append(result, *sum)
			}
			return result, nil
		}
	}

	return nil, nil
}

// selectOldestChunkAtDepth scans context_items from oldest ordinal, collecting consecutive
// summaries at the given depth. Stops at non-summary items, different depth, fresh tail, or
// token overflow. Returns contiguous chunk of summaries.
func (e *CompactionEngine) selectOldestChunkAtDepth(
	ctx context.Context, convID int64, targetDepth int,
) ([]Summary, error) {
	items, err := e.store.GetContextItems(ctx, convID)
	if err != nil {
		return nil, err
	}

	tailStartIdx := len(items) - FreshTailCount
	if tailStartIdx < 0 {
		tailStartIdx = 0
	}

	var chunk []Summary
	accumTokens := 0

	for i := 0; i < tailStartIdx; i++ {
		item := items[i]
		if item.ItemType != "summary" {
			// Non-summary breaks the chunk
			break
		}
		sum, err := e.store.GetSummary(ctx, item.SummaryID)
		if err != nil {
			break
		}
		if sum.Depth != targetDepth {
			// Different depth breaks the chunk
			break
		}
		if accumTokens+sum.TokenCount > LeafChunkTokens {
			// Token overflow stops collection
			break
		}
		chunk = append(chunk, *sum)
		accumTokens += sum.TokenCount
	}

	// Min tokens check: spec line 808
	// chunk tokens must be >= max(CondensedTargetTokens, LeafChunkTokens × 0.1) = 2000
	minTokens := CondensedTargetTokens // 2000
	if accumTokens < minTokens {
		return nil, nil
	}

	return chunk, nil
}

// generateLeafSummary calls the LLM to generate a leaf summary with 3-level escalation.
// Level 1: normal LLM prompt. Level 2: aggressive prompt. Level 3: deterministic truncation.
func (e *CompactionEngine) generateLeafSummary(
	ctx context.Context,
	messages []Message,
	previousSummary string,
) (string, error) {
	if e.complete == nil {
		return truncateSummary(messages), nil
	}

	sourceText := formatMessagesForSummary(messages)
	inputTokens := sumMessageTokens(messages)
	targetTokens := minInt(LeafTargetTokens, int(float64(inputTokens)*0.35))

	// Level 1: normal prompt
	prompt := buildLeafSummaryPrompt(sourceText, previousSummary, targetTokens)
	content, err := e.complete(ctx, prompt, CompleteOptions{
		MaxTokens:   LeafTargetTokens * 2,
		Temperature: 0.3,
	})
	if err != nil {
		return "", err
	}
	if content == "" {
		// Retry with temperature=0
		content, err = e.complete(ctx, prompt, CompleteOptions{
			MaxTokens:   LeafTargetTokens * 2,
			Temperature: 0,
		})
		if err != nil {
			return "", err
		}
	}

	// Level 1 only succeeds if it actually reaches the requested target size.
	if content != "" && tokenizer.EstimateMessageTokens(providers.Message{Content: content}) <= targetTokens {
		return content, nil
	}

	// Level 2: aggressive prompt
	aggressiveTarget := minInt(640, int(float64(inputTokens)*0.20))
	aggressivePrompt := buildAggressiveLeafSummaryPrompt(sourceText, previousSummary, aggressiveTarget)
	content, err = e.complete(ctx, aggressivePrompt, CompleteOptions{
		MaxTokens:   aggressiveTarget * 2,
		Temperature: 0.3,
	})
	if err != nil {
		return "", err
	}
	if content == "" {
		// Retry with temperature=0
		content, err = e.complete(ctx, aggressivePrompt, CompleteOptions{
			MaxTokens:   aggressiveTarget * 2,
			Temperature: 0,
		})
		if err != nil {
			return "", err
		}
	}
	if content != "" && tokenizer.EstimateMessageTokens(providers.Message{Content: content}) <= aggressiveTarget {
		return content, nil
	}

	// Level 3: deterministic truncation
	return truncateSummary(messages), nil
}

// generateCondensedSummary calls the LLM to generate a condensed summary with 3-level escalation.
func (e *CompactionEngine) generateCondensedSummary(ctx context.Context, summaries []Summary) (string, error) {
	if e.complete == nil {
		return truncateCondensedSummaries(summaries), nil
	}

	sourceText := formatSummariesForCondensation(summaries)
	inputTokens := sumSummaryTokens(summaries)
	targetTokens := minInt(CondensedTargetTokens, int(float64(inputTokens)*0.35))

	// Level 1: normal prompt
	prompt := buildCondensedSummaryPrompt(sourceText, targetTokens)
	content, err := e.complete(ctx, prompt, CompleteOptions{
		MaxTokens:   CondensedTargetTokens * 2,
		Temperature: 0.3,
	})
	if err != nil {
		return "", err
	}
	if content == "" {
		content, err = e.complete(ctx, prompt, CompleteOptions{
			MaxTokens:   CondensedTargetTokens * 2,
			Temperature: 0,
		})
		if err != nil {
			return "", err
		}
	}
	if content != "" {
		return content, nil
	}

	// Level 2: aggressive prompt
	aggressiveTarget := minInt(640, int(float64(inputTokens)*0.20))
	aggressivePrompt := buildCondensedSummaryPrompt(sourceText, aggressiveTarget)
	content, err = e.complete(ctx, aggressivePrompt, CompleteOptions{
		MaxTokens:   aggressiveTarget * 2,
		Temperature: 0.3,
	})
	if err != nil {
		return "", err
	}
	if content != "" {
		return content, nil
	}

	// Level 3: deterministic fallback
	return truncateCondensedSummaries(summaries), nil
}

// runCondensedLoop runs condensed compaction in a loop until:
// a) context tokens <= threshold (success), OR
// b) No candidate found (nothing to condense), OR
// c) tokensAfter >= tokensBefore (no progress this iteration), OR
// d) tokensAfter >= previousTokens (no improvement over last iteration)
func (e *CompactionEngine) runCondensedLoop(ctx context.Context, convID int64) {
	var prevTokens int
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		tokensBefore, err := e.store.GetContextTokenCount(ctx, convID)
		if err != nil {
			logger.ErrorCF("seahorse", "condensed: get tokens", map[string]any{"error": err.Error()})
			return
		}

		condensedID, err := e.compactCondensed(ctx, convID)
		if err != nil {
			logger.ErrorCF("seahorse", "condensed: compact", map[string]any{"error": err.Error()})
			return
		}
		if condensedID == nil {
			// No candidate found
			logger.DebugCF("seahorse", "condensed: no candidate", map[string]any{"conv_id": convID})
			return
		}

		tokensAfter, _ := e.store.GetContextTokenCount(ctx, convID)

		if tokensAfter >= tokensBefore {
			// No progress this iteration
			logger.DebugCF(
				"seahorse",
				"condensed: no progress",
				map[string]any{"conv_id": convID, "tokens_before": tokensBefore, "tokens_after": tokensAfter},
			)
			return
		}
		if tokensAfter >= prevTokens && prevTokens > 0 {
			// No improvement over last iteration
			logger.DebugCF(
				"seahorse",
				"condensed: no improvement",
				map[string]any{"conv_id": convID, "tokens": tokensAfter},
			)
			return
		}

		prevTokens = tokensAfter
	}
}

// --- Helper functions ---

func formatMessagesForSummary(messages []Message) string {
	var result string
	for _, m := range messages {
		ts := m.CreatedAt.Format("2006-01-02 15:04 MST")
		content := m.Content
		if content == "" && len(m.Parts) > 0 {
			content = partsToReadableContent(m.Parts)
		}
		result += fmt.Sprintf("[%s]\n%s\n\n", ts, content)
	}
	return result
}

func formatSummariesForCondensation(summaries []Summary) string {
	var result string
	for _, s := range summaries {
		earliest := ""
		if s.EarliestAt != nil {
			earliest = s.EarliestAt.Format("2006-01-02")
		}
		latest := ""
		if s.LatestAt != nil {
			latest = s.LatestAt.Format("2006-01-02")
		}
		result += fmt.Sprintf("[%s - %s]\n%s\n\n", earliest, latest, s.Content)
	}
	return result
}

func buildLeafSummaryPrompt(sourceText, previousSummary string, targetTokens int) string {
	prev := "(none)"
	if previousSummary != "" {
		prev = previousSummary
	}
	return fmt.Sprintf(`You summarize a SEGMENT of a conversation for future model turns.
Treat this as incremental memory compaction input, not a full-conversation summary.

Normal summary policy:
- Preserve key decisions, rationale, constraints, and active tasks.
- Keep essential technical details needed to continue work safely.
- Remove obvious repetition and conversational filler.

Output requirements:
- Plain text only.
- No preamble, headings, or markdown formatting.
- Track file operations (created, modified, deleted, renamed) with file paths and current status.
- If no file operations appear, include exactly: "Files: none".
- End with exactly: "Expand for details about: <comma-separated list of what was dropped or compressed>".
- Target length: about %d tokens or less.

<previous_context>
%s
</previous_context>

<conversation_segment>
%s
</conversation_segment>`, targetTokens, prev, sourceText)
}

func buildCondensedSummaryPrompt(sourceText string, targetTokens int) string {
	return fmt.Sprintf(`You condense multiple summaries into a single higher-level summary.
Preserve all important decisions, constraints, and outcomes.
Merge overlapping topics. Keep technical details intact.

Output requirements:
- Plain text only.
- No preamble, headings, or markdown formatting.
- End with exactly: "Expand for details about: <comma-separated list>".
- Target length: about %d tokens or less.

<summaries>
%s
</summaries>`, targetTokens, sourceText)
}

func buildAggressiveLeafSummaryPrompt(sourceText, previousSummary string, targetTokens int) string {
	prev := "(none)"
	if previousSummary != "" {
		prev = previousSummary
	}
	return fmt.Sprintf(`You summarize a SEGMENT of a conversation for future model turns.
Aggressive summary policy:
- Keep only durable facts and current task state.
- Remove examples, repetition, and low-value narrative details.
- Preserve explicit TODOs, blockers, decisions, and constraints.

Output requirements:
- Plain text only.
- No preamble, headings, or markdown formatting.
- Track file operations (created, modified, deleted, renamed) with file paths and current status.
- If no file operations appear, include exactly: "Files: none".
- End with exactly: "Expand for details about: <comma-separated list of what was dropped or compressed>".
- Target length: about %d tokens or less.

<previous_context>
%s
</previous_context>

<conversation_segment>
%s
</conversation_segment>`, targetTokens, prev, sourceText)
}

func truncateSummary(messages []Message) string {
	content := ""
	for _, m := range messages {
		c := m.Content
		if c == "" && len(m.Parts) > 0 {
			c = partsToReadableContent(m.Parts)
		}
		content += c + "\n"
	}
	if len(content) > 2048 {
		content = content[:2048]
	}
	content += fmt.Sprintf("\n[Truncated from %d messages]", len(messages))
	return content
}

func truncateCondensedSummaries(summaries []Summary) string {
	content := ""
	for _, s := range summaries {
		content += s.Content + "\n"
	}
	if len(content) > 2048 {
		content = content[:2048]
	}
	content += fmt.Sprintf("\n[Condensed from %d summaries]", len(summaries))
	return content
}

func sumMessageTokens(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += m.TokenCount
	}
	return total
}

func sumSummaryTokens(summaries []Summary) int {
	total := 0
	for _, s := range summaries {
		total += s.TokenCount
	}
	return total
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
