package seahorse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// escapeXML escapes special characters for safe inclusion in XML content.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// resolvedItem is a context item resolved to its full content with token count.
type resolvedItem struct {
	ordinal    int
	itemType   string // "message" or "summary"
	message    *Message
	summary    *Summary
	tokenCount int
}

// Assemble builds budget-constrained context from summaries + messages.
//
// Algorithm:
//  1. Fetch context_items, resolve to full content
//  2. Split into evictable prefix + protected fresh tail
//  3. If evictable fits in remaining budget → include all
//  4. Else walk evictable from newest to oldest, keep while fits
func (a *Assembler) Assemble(ctx context.Context, convID int64, input AssembleInput) (*AssembleResult, error) {
	items, err := a.store.GetContextItems(ctx, convID)
	if err != nil {
		return nil, fmt.Errorf("get context items: %w", err)
	}
	if len(items) == 0 {
		return &AssembleResult{}, nil
	}

	// Resolve all items
	resolved := make([]resolvedItem, len(items))
	for i, item := range items {
		r, err := a.resolveItem(ctx, item)
		if err != nil {
			return nil, err
		}
		resolved[i] = r
	}

	// Split into evictable prefix and protected fresh tail
	tailStart := len(resolved) - FreshTailCount
	if tailStart < 0 {
		tailStart = 0
	}
	evictable := resolved[:tailStart]
	freshTail := resolved[tailStart:]

	// Calculate fresh tail tokens
	freshTailTokens := 0
	for _, r := range freshTail {
		freshTailTokens += r.tokenCount
	}

	// If the protected tail alone exceeds budget, trim from the oldest end at
	// provider-safe boundaries. The rebuild path later sanitizes leading
	// assistant(tool_calls)/tool messages, so splitting the active turn here can
	// silently discard the very context we are trying to protect.
	if freshTailTokens > input.Budget {
		originalTailCount := len(freshTail)
		originalFreshTailTokens := freshTailTokens
		var preservedActiveTurn bool
		freshTail, freshTailTokens, preservedActiveTurn = trimFreshTailToSafeBudget(freshTail, input.Budget)
		logFields := map[string]any{
			"budget":                input.Budget,
			"fresh_tail_tokens":     freshTailTokens,
			"fresh_tail_count":      len(freshTail),
			"trimmed_fresh_items":   originalTailCount - len(freshTail),
			"original_fresh_tokens": originalFreshTailTokens,
			"preserved_active_turn": preservedActiveTurn,
		}
		if preservedActiveTurn {
			logger.WarnCF("seahorse", "assemble: preserving active turn over budget", logFields)
		} else {
			logger.InfoCF("seahorse", "assemble: trimmed fresh tail to safe boundary", logFields)
		}
	}
	remainingBudget := input.Budget - freshTailTokens
	if remainingBudget < 0 {
		remainingBudget = 0
	}

	var selected []resolvedItem
	evictableTokens := 0
	for _, r := range evictable {
		evictableTokens += r.tokenCount
	}

	if evictableTokens <= remainingBudget {
		// All evictable fit
		selected = append(selected, evictable...)
	} else {
		// Walk from newest to oldest, keep while fits
		var kept []resolvedItem
		accum := 0
		for i := len(evictable) - 1; i >= 0; i-- {
			if accum+evictable[i].tokenCount <= remainingBudget {
				kept = append(kept, evictable[i])
				accum += evictable[i].tokenCount
			} else {
				break
			}
		}
		// Reverse to restore chronological order
		for i, j := 0, len(kept)-1; i < j; i, j = i+1, j-1 {
			kept[i], kept[j] = kept[j], kept[i]
		}
		selected = append(selected, kept...)
	}

	// Combine: selected evictable + fresh tail
	final := append(selected, freshTail...)

	// Build result
	var messages []Message
	var summaries []Summary
	var sourceIDs []string
	totalTokens := 0
	maxDepth := 0
	condensedCount := 0

	for _, r := range final {
		totalTokens += r.tokenCount
		if r.itemType == "message" && r.message != nil {
			messages = append(messages, *r.message)
			sourceIDs = append(sourceIDs, fmt.Sprintf("msg:%d", r.message.ID))
		} else if r.itemType == "summary" && r.summary != nil {
			summaries = append(summaries, *r.summary)
			if r.summary.Depth > maxDepth {
				maxDepth = r.summary.Depth
			}
			if r.summary.Kind == SummaryKindCondensed {
				condensedCount++
			}
		}
	}

	// Build depth-aware system prompt addition
	systemPromptAddition := ""
	if len(summaries) > 0 {
		if maxDepth >= 2 || condensedCount >= 2 {
			systemPromptAddition = "Your context has been heavily compressed through multi-level summarization.\n" +
				"- Do NOT assert specific facts (commands, SHAs, paths, timestamps) from summaries without expanding.\n" +
				"- When uncertain, use expand to recover original detail before making claims.\n" +
				"- Tool escalation: grep \xe2\x86\x92 describe \xe2\x86\x92 expand"
		} else {
			systemPromptAddition = "Some earlier messages have been summarized. Use expand tools to recover details if needed."
		}
	}

	// Build Summary field: all XML summaries + system prompt addition
	var summaryParts []string
	for _, sum := range summaries {
		if sum.Content == "" {
			continue
		}
		// Load parent IDs for XML formatting
		parentSummaries, err := a.store.GetSummaryParents(ctx, sum.SummaryID)
		if err != nil {
			logger.WarnCF("seahorse", "assemble: get summary parents", map[string]any{
				"summary_id": sum.SummaryID,
				"error":      err.Error(),
			})
		}
		var parentIDs []string
		for _, ps := range parentSummaries {
			parentIDs = append(parentIDs, ps.SummaryID)
		}
		summaryParts = append(summaryParts, FormatSummaryXML(&sum, parentIDs))
	}
	summary := strings.Join(summaryParts, "\n\n")
	if systemPromptAddition != "" {
		if summary != "" {
			summary += "\n\n"
		}
		summary += systemPromptAddition
	}

	return &AssembleResult{
		Messages: messages,
		Summary:  summary,
	}, nil
}

func trimFreshTailToSafeBudget(tail []resolvedItem, budget int) ([]resolvedItem, int, bool) {
	tailTokens := resolvedItemsTokenCount(tail)
	if tailTokens <= budget {
		return tail, tailTokens, false
	}

	latestTurnStart := lastUserMessageIndex(tail)
	if latestTurnStart >= 0 {
		latestTurnTokens := resolvedItemsTokenCount(tail[latestTurnStart:])
		if latestTurnTokens > budget {
			return tail[latestTurnStart:], latestTurnTokens, true
		}
	}

	start := 0
	for tailTokens > budget && start < len(tail) {
		tailTokens -= tail[start].tokenCount
		start++
	}
	for start < len(tail) && !isProviderSafeHistoryStart(tail[start:]) {
		tailTokens -= tail[start].tokenCount
		start++
	}

	return tail[start:], tailTokens, false
}

func resolvedItemsTokenCount(items []resolvedItem) int {
	total := 0
	for _, item := range items {
		total += item.tokenCount
	}
	return total
}

func lastUserMessageIndex(items []resolvedItem) int {
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].itemType != "message" || items[i].message == nil {
			continue
		}
		if items[i].message.Role == "user" {
			return i
		}
	}
	return -1
}

func isProviderSafeHistoryStart(items []resolvedItem) bool {
	for _, item := range items {
		if item.itemType != "message" || item.message == nil {
			continue
		}
		if item.message.Role == "tool" {
			return false
		}
		if item.message.Role == "assistant" && messageHasToolUse(item.message) {
			return false
		}
		return true
	}
	return true
}

func messageHasToolUse(msg *Message) bool {
	if msg == nil {
		return false
	}
	for _, part := range msg.Parts {
		if part.Type == "tool_use" {
			return true
		}
	}
	return false
}

// resolveItem loads the full message or summary for a context item.
func (a *Assembler) resolveItem(ctx context.Context, item ContextItem) (resolvedItem, error) {
	if item.ItemType == "message" {
		msg, err := a.store.GetMessageByID(ctx, item.MessageID)
		if err != nil {
			return resolvedItem{}, err
		}
		tokens := item.TokenCount
		if tokens == 0 {
			tokens = msg.TokenCount
		}
		return resolvedItem{
			ordinal:    item.Ordinal,
			itemType:   "message",
			message:    msg,
			tokenCount: tokens,
		}, nil
	}

	if item.ItemType == "summary" {
		sum, err := a.store.GetSummary(ctx, item.SummaryID)
		if err != nil {
			return resolvedItem{}, err
		}
		tokens := item.TokenCount
		if tokens == 0 {
			tokens = sum.TokenCount
		}
		return resolvedItem{
			ordinal:    item.Ordinal,
			itemType:   "summary",
			summary:    sum,
			tokenCount: tokens,
		}, nil
	}

	return resolvedItem{
		ordinal:    item.Ordinal,
		itemType:   item.ItemType,
		tokenCount: item.TokenCount,
	}, nil
}

// FormatSummaryXML formats a summary as XML for LLM context.
// This is exported so context managers can format summaries consistently.
func FormatSummaryXML(s *Summary, parentIDs []string) string {
	// Build time attributes if available
	var attrs string
	if s.EarliestAt != nil {
		attrs += fmt.Sprintf(` earliest_at="%s"`, s.EarliestAt.Format(time.RFC3339))
	}
	if s.LatestAt != nil {
		attrs += fmt.Sprintf(` latest_at="%s"`, s.LatestAt.Format(time.RFC3339))
	}

	var parentsSection string
	if s.Kind == SummaryKindCondensed && len(parentIDs) > 0 {
		parents := "<parents>\n"
		for _, pid := range parentIDs {
			parents += fmt.Sprintf("    <summary_ref id=\"%s\" />\n", pid)
		}
		parents += "  </parents>\n"
		parentsSection = parents
	}
	return fmt.Sprintf(
		"<summary id=\"%s\" kind=\"%s\" depth=\"%d\" descendant_count=\"%d\"%s>\n  <content>\n    %s\n  </content>\n%s</summary>",
		s.SummaryID,
		string(s.Kind),
		s.Depth,
		s.DescendantCount,
		attrs,
		escapeXML(s.Content),
		parentsSection,
	)
}
