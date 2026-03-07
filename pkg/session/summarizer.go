package session

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// Summarizer is the LLM-calling strategy injected by the agent layer.
// The session package is agnostic to how the LLM is called — it only
// invokes this interface to produce textual summaries.
type Summarizer interface {
	Summarize(ctx context.Context, messages []providers.Message, existingSummary string) (string, error)
}

// Compile-time interface satisfaction checks.
var (
	_ Summarizer = (*LLMSummarizer)(nil)
	_ Summarizer = SummarizeFunc(nil)
)

// SummarizeFunc is a function adapter for Summarizer.
// It allows passing a plain function where a Summarizer is expected.
type SummarizeFunc func(ctx context.Context, messages []providers.Message, existingSummary string) (string, error)

// Summarize implements the Summarizer interface.
func (f SummarizeFunc) Summarize(
	ctx context.Context,
	messages []providers.Message,
	existingSummary string,
) (string, error) {
	return f(ctx, messages, existingSummary)
}

// BuildSummarizationPrompt formats messages into a summarization prompt string.
// This is intended to be used inside the SummarizeFunc implementation provided
// by the agent layer.
func BuildSummarizationPrompt(batch []providers.Message, existingSummary string) string {
	var sb strings.Builder
	sb.WriteString(
		"Provide a concise summary of this conversation segment, preserving core context and key points.\n",
	)
	if existingSummary != "" {
		sb.WriteString("Existing context: ")
		sb.WriteString(existingSummary)
		sb.WriteString("\n")
	}
	sb.WriteString("\nCONVERSATION:\n")
	for _, m := range batch {
		fmt.Fprintf(&sb, "%s: %s\n", m.Role, m.Content)
	}
	return sb.String()
}

// LLMSummarizer implements Summarizer by calling an LLM provider.
// This is the default implementation used in production.
type LLMSummarizer struct {
	provider providers.LLMProvider
	model    string
	agentID  string
	cfg      config.SummarizationConfig
}

// Summarize formats the messages into a summarization prompt and calls the
// LLM to produce a concise summary.
func (s *LLMSummarizer) Summarize(
	ctx context.Context,
	msgs []providers.Message,
	existingSummary string,
) (string, error) {
	prompt := BuildSummarizationPrompt(msgs, existingSummary)
	resp, err := s.provider.Chat(
		ctx,
		[]providers.Message{{Role: "user", Content: prompt}},
		nil,
		s.model,
		map[string]any{
			"max_tokens":       s.cfg.SummaryMaxTokens,
			"temperature":      s.cfg.SummaryTemperature,
			"prompt_cache_key": s.agentID,
		},
	)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// WithLLMSummarizer is a convenience option that creates an LLMSummarizer
// and configures the SessionManager for summarization in one call.
func WithLLMSummarizer(provider providers.LLMProvider, model, agentID string, cfg config.SummarizationConfig) Option {
	cfg = cfg.WithDefaults()
	return WithSummarizer(&LLMSummarizer{
		provider: provider,
		model:    model,
		agentID:  agentID,
		cfg:      cfg,
	}, cfg)
}
