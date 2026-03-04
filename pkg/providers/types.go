package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

type (
	ToolCall               = protocoltypes.ToolCall
	FunctionCall           = protocoltypes.FunctionCall
	LLMResponse            = protocoltypes.LLMResponse
	UsageInfo              = protocoltypes.UsageInfo
	Message                = protocoltypes.Message
	ToolDefinition         = protocoltypes.ToolDefinition
	ToolFunctionDefinition = protocoltypes.ToolFunctionDefinition
	ExtraContent           = protocoltypes.ExtraContent
	GoogleExtra            = protocoltypes.GoogleExtra
	ContentBlock           = protocoltypes.ContentBlock
	CacheControl           = protocoltypes.CacheControl
)

type LLMProvider interface {
	Chat(
		ctx context.Context,
		messages []Message,
		tools []ToolDefinition,
		model string,
		options map[string]any,
	) (*LLMResponse, error)
	GetDefaultModel() string
}

type StatefulProvider interface {
	LLMProvider
	Close()
}

// FailoverReason classifies why an LLM request failed for fallback decisions.
type FailoverReason string

const (
	FailoverAuth       FailoverReason = "auth"
	FailoverRateLimit  FailoverReason = "rate_limit"
	FailoverBilling    FailoverReason = "billing"
	FailoverTimeout    FailoverReason = "timeout"
	FailoverFormat     FailoverReason = "format"
	FailoverOverloaded FailoverReason = "overloaded"
	FailoverUnknown    FailoverReason = "unknown"
)

// FailoverError wraps an LLM provider error with classification metadata.
type FailoverError struct {
	Reason      FailoverReason
	Provider    string
	Model       string
	Status      int
	Wrapped     error
	Timestamp   time.Time // Timestamp of when the error occurred
	RequestID   string    // Request ID for tracking
	Correlation string    // Correlation ID
}

func (e *FailoverError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("failover(%s): provider=%s model=%s status=%d req_id=%s correlation=%s timestamp=%s: %v",
			e.Reason, e.Provider, e.Model, e.Status, e.RequestID, e.Correlation, e.Timestamp.Format(time.RFC3339), e.Wrapped)
	}
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

// SetTimestamp stores when the error occurred
func (e *FailoverError) SetTimestamp(t time.Time) *FailoverError {
	e.Timestamp = t
	return e
}

// WithRequestID sets the request ID for this error
func (e *FailoverError) WithRequestID(id string) *FailoverError {
	e.RequestID = id
	return e
}

// WithCorrelation sets the correlation identifier for this error
func (e *FailoverError) WithCorrelation(correlation string) *FailoverError {
	e.Correlation = correlation
	return e
}

// ModelConfig holds primary model and fallback list.
type ModelConfig struct {
	Primary   string
	Fallbacks []string
}
