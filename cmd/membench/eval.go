package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sipeed/picoclaw/pkg/seahorse"
)

// EvalResult holds per-sample evaluation results for one mode.
type EvalResult struct {
	Mode      string     `json:"mode"`
	SampleID  string     `json:"sampleId"`
	QAResults []QAResult `json:"qaResults"`
	Agg       AggMetrics `json:"aggregated"`
}

// QAResult holds metrics for a single QA pair.
type QAResult struct {
	Question   string  `json:"question"`
	Category   int     `json:"category"`
	GoldAnswer string  `json:"goldAnswer"`
	TokenF1    float64 `json:"tokenF1"`
	HitRate    float64 `json:"hitRate"`
}

// AggMetrics holds aggregated evaluation metrics.
type AggMetrics struct {
	OverallF1      float64             `json:"overallF1"`
	OverallHitRate float64             `json:"overallHitRate"`
	ByCategory     map[int]*CatMetrics `json:"byCategory"`
	TotalQuestions int                 `json:"totalQuestions"`
}

// CatMetrics holds metrics for a single category.
type CatMetrics struct {
	F1            float64 `json:"f1"`
	HitRate       float64 `json:"hitRate"`
	QuestionCount int     `json:"questionCount"`
}

// EvalLegacy evaluates using legacy session store (raw history + budget truncation).
func EvalLegacy(
	ctx context.Context,
	samples []LocomoSample,
	legacy *LegacyStore,
	budgetTokens int,
) []EvalResult {
	results := make([]EvalResult, 0, len(samples))
	for si := range samples {
		sample := &samples[si]
		history := legacy.GetHistory(sample.SampleID)

		// Convert messages to content strings
		allContent := make([]string, 0, len(history))
		for _, msg := range history {
			allContent = append(allContent, msg.Content)
		}

		qaResults := make([]QAResult, 0, len(sample.QA))
		for qi := range sample.QA {
			qa := &sample.QA[qi]
			// Budget truncate the full history
			truncated, _ := BudgetTruncate(allContent, budgetTokens)
			context := StringListToContent(truncated)

			f1 := TokenOverlapF1(context, qa.AnswerString())
			hitRate := RecallHitRate(qa.Evidence, sample, context)

			qaResults = append(qaResults, QAResult{
				Question:   qa.Question,
				Category:   qa.Category,
				GoldAnswer: qa.AnswerString(),
				TokenF1:    f1,
				HitRate:    hitRate,
			})
		}

		results = append(results, EvalResult{
			Mode:      "legacy",
			SampleID:  sample.SampleID,
			QAResults: qaResults,
			Agg:       aggregateMetrics(qaResults),
		})
	}
	return results
}

// EvalSeahorse evaluates using seahorse short memory (per-keyword search + expand).
func EvalSeahorse(
	ctx context.Context,
	samples []LocomoSample,
	ir *SeahorseIngestResult,
	budgetTokens int,
) []EvalResult {
	store := ir.Engine.GetRetrieval().Store()
	retrieval := ir.Engine.GetRetrieval()

	results := make([]EvalResult, 0, len(samples))
	for si := range samples {
		sample := &samples[si]
		convID, ok := ir.ConvMap[sample.SampleID]
		if !ok {
			log.Printf("WARN: no conversation ID for sample %s", sample.SampleID)
			continue
		}

		qaResults := make([]QAResult, 0, len(sample.QA))
		for qi := range sample.QA {
			qa := &sample.QA[qi]
			keywords := ExtractKeywords(qa.Question)

			// Search each keyword individually and union results,
			// tracking best BM25 rank per message for relevance sorting.
			bestRank := map[int64]float64{}
			for _, kw := range keywords {
				searchResults, err := store.SearchMessages(ctx, seahorse.SearchInput{
					Pattern:        kw,
					ConversationID: convID,
					Limit:          20,
				})
				if err != nil {
					log.Printf("WARN: search failed for keyword %q: %v", kw, err)
					continue
				}
				for _, sr := range searchResults {
					if sr.MessageID > 0 {
						if prev, ok := bestRank[sr.MessageID]; !ok || sr.Rank < prev {
							bestRank[sr.MessageID] = sr.Rank
						}
					}
				}
			}
			// Sort messageIDs by rank ascending (best/most-negative first).
			// BudgetTruncate walks from the front, keeping best-ranked messages.
			// Note: SQLite FTS5 bm25() returns negative values where more
			// negative = better match.
			messageIDs := make([]int64, 0, len(bestRank))
			for id := range bestRank {
				messageIDs = append(messageIDs, id)
			}
			sort.Slice(messageIDs, func(i, j int) bool {
				return bestRank[messageIDs[i]] < bestRank[messageIDs[j]]
			})

			// Expand messages to get full content
			var contentParts []string
			if len(messageIDs) > 0 {
				expandResult, err := retrieval.ExpandMessages(ctx, messageIDs)
				if err != nil {
					log.Printf("WARN: expand failed for sample %s: %v", sample.SampleID, err)
				} else {
					for _, msg := range expandResult.Messages {
						contentParts = append(contentParts, msg.Content)
					}
				}
			}

			if len(contentParts) == 0 {
				qaResults = append(qaResults, QAResult{
					Question:   qa.Question,
					Category:   qa.Category,
					GoldAnswer: qa.AnswerString(),
					TokenF1:    0.0,
					HitRate:    0.0,
				})
				continue
			}

			// Budget truncate (drop worst-ranked)
			truncated, _ := BudgetTruncate(contentParts, budgetTokens)
			context := StringListToContent(truncated)

			f1 := TokenOverlapF1(context, qa.AnswerString())
			hitRate := RecallHitRate(qa.Evidence, sample, context)

			qaResults = append(qaResults, QAResult{
				Question:   qa.Question,
				Category:   qa.Category,
				GoldAnswer: qa.AnswerString(),
				TokenF1:    f1,
				HitRate:    hitRate,
			})
		}

		results = append(results, EvalResult{
			Mode:      "seahorse",
			SampleID:  sample.SampleID,
			QAResults: qaResults,
			Agg:       aggregateMetrics(qaResults),
		})
	}
	return results
}

// aggregateMetrics computes overall and per-category metrics.
func aggregateMetrics(qaResults []QAResult) AggMetrics {
	byCat := map[int]*CatMetrics{}
	totalF1 := 0.0
	totalHitRate := 0.0
	for _, qr := range qaResults {
		totalF1 += qr.TokenF1
		totalHitRate += qr.HitRate
		cat, ok := byCat[qr.Category]
		if !ok {
			cat = &CatMetrics{}
			byCat[qr.Category] = cat
		}
		cat.F1 += qr.TokenF1
		cat.HitRate += qr.HitRate
		cat.QuestionCount++
	}
	n := len(qaResults)
	if n == 0 {
		n = 1
	}
	agg := AggMetrics{
		OverallF1:      totalF1 / float64(n),
		OverallHitRate: totalHitRate / float64(n),
		ByCategory:     byCat,
		TotalQuestions: len(qaResults),
	}
	for _, cat := range agg.ByCategory {
		if cat.QuestionCount > 0 {
			cat.F1 /= float64(cat.QuestionCount)
			cat.HitRate /= float64(cat.QuestionCount)
		}
	}
	return agg
}

// SaveResults writes per-sample eval results to JSON files.
func SaveResults(results []EvalResult, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	for _, r := range results {
		path := filepath.Join(outDir, fmt.Sprintf("eval_%s_%s.json", r.Mode, r.SampleID))
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal result: %w", err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("write result: %w", err)
		}
	}
	return nil
}

// SaveAggregated writes a combined results.json with all modes.
func SaveAggregated(results []EvalResult, outDir string) error {
	byMode := map[string][]EvalResult{}
	for _, r := range results {
		byMode[r.Mode] = append(byMode[r.Mode], r)
	}

	aggMap := map[string]AggMetrics{}
	for mode, modeResults := range byMode {
		aggMap[mode] = computeModeAgg(modeResults)
	}

	data, err := json.MarshalIndent(aggMap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "results.json"), data, 0o644)
}

// computeModeAgg aggregates results for a single mode using weighted averaging
// (weighted by question count per sample). All modes must have the same Mode field.
func computeModeAgg(results []EvalResult) AggMetrics {
	agg := AggMetrics{ByCategory: map[int]*CatMetrics{}}
	for _, r := range results {
		agg.OverallF1 += r.Agg.OverallF1 * float64(r.Agg.TotalQuestions)
		agg.OverallHitRate += r.Agg.OverallHitRate * float64(r.Agg.TotalQuestions)
		agg.TotalQuestions += r.Agg.TotalQuestions
		for cat, cm := range r.Agg.ByCategory {
			existing, ok := agg.ByCategory[cat]
			if !ok {
				existing = &CatMetrics{}
				agg.ByCategory[cat] = existing
			}
			existing.F1 += cm.F1 * float64(cm.QuestionCount)
			existing.HitRate += cm.HitRate * float64(cm.QuestionCount)
			existing.QuestionCount += cm.QuestionCount
		}
	}
	if agg.TotalQuestions > 0 {
		agg.OverallF1 /= float64(agg.TotalQuestions)
		agg.OverallHitRate /= float64(agg.TotalQuestions)
	}
	for _, cat := range agg.ByCategory {
		if cat.QuestionCount > 0 {
			cat.F1 /= float64(cat.QuestionCount)
			cat.HitRate /= float64(cat.QuestionCount)
		}
	}
	return agg
}

// printSection prints a single comparison table section.
func printSection(title string, results []EvalResult) {
	fmt.Printf("\n--- %s ---\n", title)
	byMode := map[string][]EvalResult{}
	for _, r := range results {
		byMode[r.Mode] = append(byMode[r.Mode], r)
	}

	modes := map[string]AggMetrics{}
	for mode, modeResults := range byMode {
		modes[mode] = computeModeAgg(modeResults)
	}

	modeKeys := make([]string, 0, len(modes))
	for k := range modes {
		modeKeys = append(modeKeys, k)
	}
	sort.Strings(modeKeys)

	// Collect all category keys across modes
	catSet := map[int]bool{}
	for _, agg := range modes {
		for cat := range agg.ByCategory {
			catSet[cat] = true
		}
	}
	cats := make([]int, 0, len(catSet))
	for cat := range catSet {
		cats = append(cats, cat)
	}
	sort.Ints(cats)

	fmt.Printf("%-10s %-8s %-8s", "Mode", "HitRate", "F1")
	for _, cat := range cats {
		fmt.Printf(" %-7s", fmt.Sprintf("C%d", cat))
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 10+8+8+7*len(cats)+8))

	for _, mode := range modeKeys {
		agg := modes[mode]
		fmt.Printf("%-10s %-8.4f %-8.4f", mode, agg.OverallHitRate, agg.OverallF1)
		for _, cat := range cats {
			if cm, ok := agg.ByCategory[cat]; ok {
				fmt.Printf(" %-7.4f", cm.HitRate)
			} else {
				fmt.Printf(" %-7s", "N/A")
			}
		}
		fmt.Println()
	}
}

// PrintComparison outputs a human-readable comparison table to stdout.
func PrintComparison(results []EvalResult, llmResults []EvalResult) {
	printSection("No LLM generation", results)
	if len(llmResults) > 0 {
		printSection("With LLM", llmResults)
	}
}
