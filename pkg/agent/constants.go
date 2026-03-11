package agent

import "time"

// Agent loop constants — extracted from inline magic numbers for maintainability.
const (
	// DefaultMaxLLMRetries is the maximum number of retries for LLM calls
	// when a context window error is detected.
	DefaultMaxLLMRetries = 2

	// SummarizeMessageThreshold is the minimum number of messages in session
	// history before summarization is triggered.
	SummarizeMessageThreshold = 20

	// ContextWindowUsagePercent is the percentage of the context window that,
	// when exceeded by the token estimate, triggers summarization.
	ContextWindowUsagePercent = 75

	// MessagesKeptAfterSummary is the number of most recent messages kept
	// after summarization to maintain conversational continuity.
	MessagesKeptAfterSummary = 4

	// SummarizationTimeout is the maximum time allowed for the summarization
	// LLM call(s) before they are cancelled.
	SummarizationTimeout = 120 * time.Second

	// SummarizeMaxTokens is the max tokens requested from the LLM for
	// summarization responses.
	SummarizeMaxTokens = 1024

	// SummarizeTemperature is the temperature used for summarization calls
	// (low for deterministic output).
	SummarizeTemperature = 0.3

	// MinHistoryForCompression is the minimum number of messages in history
	// before force compression will attempt to reduce it.
	MinHistoryForCompression = 4

	// MultiPartSummarizationThreshold is the number of messages above which
	// summarization splits messages into two batches for separate summarization
	// before merging.
	MultiPartSummarizationThreshold = 10
)
