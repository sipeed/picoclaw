package providers

import (
	"context"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

type ToolCall = protocoltypes.ToolCall
type FunctionCall = protocoltypes.FunctionCall
type ExtraContent = protocoltypes.ExtraContent
type GoogleExtra = protocoltypes.GoogleExtra
type LLMResponse = protocoltypes.LLMResponse
type UsageInfo = protocoltypes.UsageInfo
type Message = protocoltypes.Message
type ToolDefinition = protocoltypes.ToolDefinition
type ToolFunctionDefinition = protocoltypes.ToolFunctionDefinition

type LLMProvider interface {
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]interface{}) (*LLMResponse, error)
	GetDefaultModel() string
}

type FailoverReason string

const (
	FailoverTimeout    FailoverReason = "timeout"
	FailoverStatus     FailoverReason = "status"
	FailoverEmpty      FailoverReason = "empty"
	FailoverFormat     FailoverReason = "format"
	FailoverOverloaded FailoverReason = "overloaded"
	FailoverAuth       FailoverReason = "auth"
	FailoverBilling    FailoverReason = "billing"
	FailoverRateLimit  FailoverReason = "rate_limit"
	FailoverContextWindow FailoverReason = "context_window"
	FailoverUnknown    FailoverReason = "unknown"
)

// FailoverError wraps an LLM provider error with classification metadata.
type FailoverError struct {
	Reason   FailoverReason
	Provider string
	Model    string
	Status   int
	Wrapped  error
}

func (e *FailoverError) Error() string {
	return fmt.Sprintf("failover(%s): provider=%s model=%s status=%d: %v",
		e.Reason, e.Provider, e.Model, e.Status, e.Wrapped)
}

func (e *FailoverError) Unwrap() error {
	return e.Wrapped
}

// IsRetriable returns true if this error should trigger fallback to next candidate.
// Non-retriable: Format errors (bad request structure, image dimension/size).
func (e *FailoverError) IsRetriable() bool {
	return e.Reason != FailoverFormat
}

// ModelConfig holds primary model and fallback list.
type ModelConfig struct {
	Primary   string
	Fallbacks []string
}
