package providers

import (
	"fmt"
	"time"
)

// RequestContext holds contextual information about an API request
type RequestContext struct {
	RequestID   string    // Unique identifier for the request
	Timestamp   time.Time // Time of the request
	Model       string    // Model name used in this call
	Provider    string    // Provider name used in this call
	UserID      string    // Associated user (if available)
	SessionID   string    // Session identifier (if available)
	ToolCallID  string    // Tool call ID if this is part of a tool call
	Endpoint    string    // API endpoint that was called
	Correlation string    // Additional correlation identifier
}

// EnhancedError wraps an error with additional contextual information
type EnhancedError struct {
	WrappedErr error           // Original error
	Context    *RequestContext // Contextual information
	Message    string          // Custom message for the enhanced error
	EventType  string          // Type of event that caused the error (e.g. "api_request", "rate_limit", etc.)
}

func NewEnhancedError(wrappedErr error, context *RequestContext, message string, eventType string) *EnhancedError {
	return &EnhancedError{
		WrappedErr: wrappedErr,
		Context:    context,
		Message:    message,
		EventType:  eventType,
	}
}

func (e *EnhancedError) Error() string {
	if e.Context != nil {
		return fmt.Sprintf("error: %s [event=%s, req_id=%s, provider=%s, model=%s, timestamp=%s]: %v",
			e.Message, e.EventType, e.Context.RequestID, e.Context.Provider, e.Context.Model, e.Context.Timestamp.Format(time.RFC3339), e.WrappedErr)
	}
	return fmt.Sprintf("error: %s [event=%s]: %v", e.Message, e.EventType, e.WrappedErr)
}

func (e *EnhancedError) Unwrap() error {
	return e.WrappedErr
}

// FormatDetailedError formats the error with extensive detail including all context
func (e *EnhancedError) FormatDetailedError() string {
	if e.Context == nil {
		return e.Error()
	}

	return fmt.Sprintf(`Enhanced Error Details:
  Error Message: %s
  Event Type: %s
  Request ID: %s
  Model: %s
  Provider: %s
  Endpoint: %s
  Timestamp: %s
  User ID: %s
  Session ID: %s
  Tool Call ID: %s
  Correlation ID: %s
  Wrapped Error: %v`,
		e.Message,
		e.EventType,
		e.Context.RequestID,
		e.Context.Model,
		e.Context.Provider,
		e.Context.Endpoint,
		e.Context.Timestamp.Format(time.RFC3339),
		e.Context.UserID,
		e.Context.SessionID,
		e.Context.ToolCallID,
		e.Context.Correlation,
		e.WrappedErr,
	)
}

// ExtendedFailoverError enhances the original FailoverError with additional context and metadata
type ExtendedFailoverError struct {
	Reason      FailoverReason
	Provider    string
	Model       string
	Status      int
	Wrapped     error
	RequestID   string
	Timestamp   time.Time
	Correlation string
	EventType   string
	Metadata    map[string]interface{} // Arbitrary additional metadata
	Message     string
}

func NewExtendedFailoverError(reason FailoverReason, provider, model string, status int, wrapped error) *ExtendedFailoverError {
	return &ExtendedFailoverError{
		Reason:      reason,
		Provider:    provider,
		Model:       model,
		Status:      status,
		Wrapped:     wrapped,
		RequestID:   "", // Will be set by caller
		Timestamp:   time.Now(),
		Correlation: "", // Will be set by caller
		EventType:   "provider_api_failure",
		Metadata:    make(map[string]interface{}),
		Message:     "", // Optional custom message
	}
}

func (e *ExtendedFailoverError) Error() string {
	baseMsg := fmt.Sprintf("extended_failover(%s): provider=%s model=%s status=%d", e.Reason, e.Provider, e.Model, e.Status)
	if e.RequestID != "" {
		baseMsg += fmt.Sprintf(" request_id=%s", e.RequestID)
	}
	if e.Correlation != "" {
		baseMsg += fmt.Sprintf(" correlation=%s", e.Correlation)
	}
	if e.Message != "" {
		baseMsg += fmt.Sprintf(" message='%s'", e.Message)
	}
	baseMsg += fmt.Sprintf(" timestamp=%s", e.Timestamp.Format(time.RFC3339))

	// Include error details from wrapped error if present
	if e.Wrapped != nil {
		baseMsg += fmt.Sprintf(": %v", e.Wrapped)
	}

	return baseMsg
}

func (e *ExtendedFailoverError) Unwrap() error {
	return e.Wrapped
}

func (e *ExtendedFailoverError) WithRequestID(id string) *ExtendedFailoverError {
	e.RequestID = id
	return e
}

func (e *ExtendedFailoverError) WithTimestamp(ts time.Time) *ExtendedFailoverError {
	e.Timestamp = ts
	return e
}

func (e *ExtendedFailoverError) WithCorrelationID(id string) *ExtendedFailoverError {
	e.Correlation = id
	return e
}

func (e *ExtendedFailoverError) WithEventType(eventType string) *ExtendedFailoverError {
	e.EventType = eventType
	return e
}

func (e *ExtendedFailoverError) WithMetadata(key string, value interface{}) *ExtendedFailoverError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

func (e *ExtendedFailoverError) WithMessage(message string) *ExtendedFailoverError {
	e.Message = message
	return e
}

// IsRetriable returns true if this error should trigger fallback to next candidate.
// Non-retriable: Format errors (bad request structure, image dimension/size).
func (e *ExtendedFailoverError) IsRetriable() bool {
	return e.Reason != FailoverFormat
}

// Convert a regular FailoverError to an ExtendedFailoverError
func (e *FailoverError) AsExtendedError() *ExtendedFailoverError {
	extended := NewExtendedFailoverError(e.Reason, e.Provider, e.Model, e.Status, e.Wrapped)
	extended.Timestamp = time.Now()
	return extended
}
