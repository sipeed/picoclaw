package evolution

import (
	"context"
	"time"
)

const (
	llmTaskSuccessJudgeTimeout = 15 * time.Second
	llmPatternClusterTimeout   = 45 * time.Second
	llmDraftGenerationTimeout  = 60 * time.Second
)

func withLLMCallTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if timeout <= 0 {
		return context.WithCancel(parent)
	}
	if deadline, ok := parent.Deadline(); ok && time.Until(deadline) <= timeout {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}
