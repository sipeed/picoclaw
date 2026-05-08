package agent

import "fmt"

// ContextOverflowError is a structured error for context window exceeded.
type ContextOverflowError struct {
	Model           string
	ContextWindow   int
	RequestedTokens int
	Reason          string
}

func (e *ContextOverflowError) Error() string {
	return fmt.Sprintf("context window exceeded: model=%s window=%d requested=%d reason=%s",
		e.Model, e.ContextWindow, e.RequestedTokens, e.Reason)
}
