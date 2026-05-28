package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/seahorse"
)

const answerSystemPrompt = `You are a helpful assistant. Given conversation context, answer the question concisely and accurately. If the answer is not in the context, say "I don't know". Answer in 1-3 sentences maximum.`

const judgeSystemPrompt = `You are an impartial judge evaluating answer quality.
Compare the candidate answer against the reference answer.
Consider semantic equivalence — different wording expressing the same meaning should score high.

Output ONLY a single integer score from 1 to 5:
1 = completely wrong or irrelevant
2 = partially related but mostly incorrect
3 = partially correct, missing key details
4 = mostly correct with minor omissions
5 = fully correct, semantically equivalent

Output ONLY the number, nothing else.`

// generateAnswer asks the LLM to answer a question given retrieved context.
func generateAnswer(ctx context.Context, client *LLMClient, contextText, question string) (string, error) {
	// Truncate context to avoid exceeding model limits while preserving valid UTF-8.
	contextRunes := []rune(contextText)
	if len(contextRunes) > 6000 {
		contextText = string(contextRunes[:6000]) + "\n... [truncated]"
	}

	userPrompt := fmt.Sprintf("## Conversation Context\n\n%s\n\n## Question\n\n%s", contextText, question)
	return client.Complete(ctx, answerSystemPrompt, userPrompt)
}

// scoreRe matches the first standalone integer 1-5 in the judge response.
var scoreRe = regexp.MustCompile(`\b([1-5])\b`)

// judgeAnswer asks the LLM to score the candidate answer vs the gold answer.
// Returns a score from 0.0 to 1.0, or -1.0 on parse failure.
func judgeAnswer(
	ctx context.Context,
	judgeClient *LLMClient,
	question, goldAnswer, candidateAnswer string,
) (float64, error) {
	userPrompt := fmt.Sprintf(
		"Question: %s\n\nReference Answer: %s\n\nCandidate Answer: %s\n\nScore:",
		question, goldAnswer, candidateAnswer,
	)

	response, err := judgeClient.Complete(ctx, judgeSystemPrompt, userPrompt)
	if err != nil {
		return -1.0, err
	}

	response = strings.TrimSpace(response)
	if m := scoreRe.FindStringSubmatch(response); len(m) == 2 {
		score, _ := strconv.Atoi(m[1])
		return float64(score-1) / 4.0, nil // Normalize 1-5 to 0.0-1.0
	}
	log.Printf("WARNING: could not parse judge score from: %q, returning -1", response)
	return -1.0, nil
}

// qaWork describes one QA evaluation unit.
type qaWork struct {
	sampleID    string
	qaIndex     int
	globalIndex int
	totalQA     int
	qa          *LocomoQA
	contextText string
	sample      *LocomoSample
}

// qaResult collects one QA evaluation output.
type qaResultOut struct {
	index  int // position in the flat QA list for ordering
	result QAResult
	answer string
	score  float64
}

// evalQAWorker processes a single QA item: generate answer + judge score.
func evalQAWorker(
	ctx context.Context,
	w qaWork,
	answerClient, judgeClient *LLMClient,
	logPrefix string,
) qaResultOut {
	llmAnswer, err := generateAnswer(ctx, answerClient, w.contextText, w.qa.Question)
	if err != nil {
		log.Printf("WARN: LLM generation failed for sample %s Q%d: %v", w.sampleID, w.qaIndex, err)
		llmAnswer = ""
	}

	score := -1.0
	if llmAnswer != "" {
		score, err = judgeAnswer(ctx, judgeClient, w.qa.Question, w.qa.AnswerString(), llmAnswer)
		if err != nil {
			log.Printf("WARN: LLM judge failed for sample %s Q%d: %v", w.sampleID, w.qaIndex, err)
		}
	}

	hitRate := RecallHitRate(w.qa.Evidence, w.sample, w.contextText)

	log.Printf("[%s] sample=%s q=%d/%d score=%.2f answer=%q",
		logPrefix, w.sampleID, w.globalIndex, w.totalQA, score, truncateStr(llmAnswer, 80))

	return qaResultOut{
		index: w.globalIndex,
		result: QAResult{
			Question:   w.qa.Question,
			Category:   w.qa.Category,
			GoldAnswer: w.qa.AnswerString(),
			TokenF1:    score,
			HitRate:    hitRate,
		},
		answer: llmAnswer,
		score:  score,
	}
}

// EvalLegacyLLM evaluates legacy store using LLM generation + LLM-as-Judge.
func EvalLegacyLLM(
	ctx context.Context,
	samples []LocomoSample,
	legacy *LegacyStore,
	budgetTokens int,
	answerClient, judgeClient *LLMClient,
	concurrency int,
) []EvalResult {
	if concurrency < 1 {
		concurrency = 1
	}
	totalQA := countTotalQA(samples)
	results := make([]EvalResult, 0, len(samples))

	for si := range samples {
		sample := &samples[si]
		history := legacy.GetHistory(sample.SampleID)

		allContent := make([]string, 0, len(history))
		for _, msg := range history {
			allContent = append(allContent, msg.Content)
		}

		truncated, _ := BudgetTruncate(allContent, budgetTokens)
		contextText := StringListToContent(truncated)

		qaResults := make([]QAResult, len(sample.QA))

		if concurrency <= 1 {
			for qi := range sample.QA {
				out := evalQAWorker(ctx, qaWork{
					sampleID: sample.SampleID, qaIndex: qi,
					globalIndex: si*len(sample.QA) + qi + 1, totalQA: totalQA,
					qa: &sample.QA[qi], contextText: contextText, sample: sample,
				}, answerClient, judgeClient, "legacy-llm")
				qaResults[qi] = out.result
			}
		} else {
			sem := make(chan struct{}, concurrency)
			var wg sync.WaitGroup
			for qi := range sample.QA {
				wg.Add(1)
				go func() {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					out := evalQAWorker(ctx, qaWork{
						sampleID: sample.SampleID, qaIndex: qi,
						globalIndex: si*len(sample.QA) + qi + 1, totalQA: totalQA,
						qa: &sample.QA[qi], contextText: contextText, sample: sample,
					}, answerClient, judgeClient, "legacy-llm")
					qaResults[qi] = out.result // safe: each goroutine writes distinct index
				}()
			}
			wg.Wait()
		}

		results = append(results, EvalResult{
			Mode:      "legacy-llm",
			SampleID:  sample.SampleID,
			QAResults: qaResults,
			Agg:       aggregateMetrics(qaResults),
		})
	}
	return results
}

// buildSeahorseContext retrieves context for a seahorse QA item.
func buildSeahorseContext(
	ctx context.Context,
	ir *SeahorseIngestResult,
	sample *LocomoSample,
	qa *LocomoQA,
	budgetTokens int,
) string {
	store := ir.Engine.GetRetrieval().Store()
	retrieval := ir.Engine.GetRetrieval()
	convID := ir.ConvMap[sample.SampleID]

	keywords := ExtractKeywords(qa.Question)
	bestRank := map[int64]float64{}
	for _, kw := range keywords {
		searchResults, err := store.SearchMessages(ctx, seahorse.SearchInput{
			Pattern:        kw,
			ConversationID: convID,
			Limit:          20,
		})
		if err != nil {
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

	messageIDs := make([]int64, 0, len(bestRank))
	for id := range bestRank {
		messageIDs = append(messageIDs, id)
	}
	sort.Slice(messageIDs, func(i, j int) bool {
		return bestRank[messageIDs[i]] < bestRank[messageIDs[j]]
	})

	var contentParts []string
	if len(messageIDs) > 0 {
		expandResult, err := retrieval.ExpandMessages(ctx, messageIDs)
		if err == nil {
			for _, msg := range expandResult.Messages {
				contentParts = append(contentParts, msg.Content)
			}
		}
	}
	if len(contentParts) == 0 {
		return ""
	}
	truncated, _ := BudgetTruncate(contentParts, budgetTokens)
	return StringListToContent(truncated)
}

// EvalSeahorseLLM evaluates seahorse retrieval using LLM generation + LLM-as-Judge.
func EvalSeahorseLLM(
	ctx context.Context,
	samples []LocomoSample,
	ir *SeahorseIngestResult,
	budgetTokens int,
	answerClient, judgeClient *LLMClient,
	concurrency int,
) []EvalResult {
	if concurrency < 1 {
		concurrency = 1
	}
	totalQA := countTotalQA(samples)
	results := make([]EvalResult, 0, len(samples))

	for si := range samples {
		sample := &samples[si]
		if _, ok := ir.ConvMap[sample.SampleID]; !ok {
			log.Printf("WARN: no conversation ID for sample %s", sample.SampleID)
			continue
		}

		qaResults := make([]QAResult, len(sample.QA))

		evalOne := func(qi int) {
			qa := &sample.QA[qi]
			contextText := buildSeahorseContext(ctx, ir, sample, qa, budgetTokens)
			if contextText == "" {
				qaResults[qi] = QAResult{
					Question:   qa.Question,
					Category:   qa.Category,
					GoldAnswer: qa.AnswerString(),
					TokenF1:    0.0,
					HitRate:    0.0,
				}
				log.Printf("[seahorse-llm] sample=%s q=%d/%d score=0.00 answer=(no context)",
					sample.SampleID, si*len(sample.QA)+qi+1, totalQA)
				return
			}
			out := evalQAWorker(ctx, qaWork{
				sampleID: sample.SampleID, qaIndex: qi,
				globalIndex: si*len(sample.QA) + qi + 1, totalQA: totalQA,
				qa: qa, contextText: contextText, sample: sample,
			}, answerClient, judgeClient, "seahorse-llm")
			qaResults[qi] = out.result
		}

		if concurrency <= 1 {
			for qi := range sample.QA {
				evalOne(qi)
			}
		} else {
			sem := make(chan struct{}, concurrency)
			var wg sync.WaitGroup
			for qi := range sample.QA {
				wg.Add(1)
				go func() {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					evalOne(qi)
				}()
			}
			wg.Wait()
		}

		results = append(results, EvalResult{
			Mode:      "seahorse-llm",
			SampleID:  sample.SampleID,
			QAResults: qaResults,
			Agg:       aggregateMetrics(qaResults),
		})
	}
	return results
}

func countTotalQA(samples []LocomoSample) int {
	n := 0
	for i := range samples {
		n += len(samples[i].QA)
	}
	return n
}

func truncateStr(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return s
}
