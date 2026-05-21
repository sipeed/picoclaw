package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
)

const asyncCompletionSynthesisTimeout = 120 * time.Second

// AsyncCompletionInput is the typed internal form of an async tool completion
// that needs parent-agent synthesis. Legacy system inbound messages are still
// adapted into this shape for compatibility, but new runtime code should call
// processAsyncCompletion directly instead of publishing a synthetic chat
// message.
type AsyncCompletionInput struct {
	SourceTool   string
	CompletionID string
	Content      string
	Origin       bus.InboundContext
	SenderID     string
}

func asyncCompletionID(turnID, toolCallID, toolName string) string {
	parts := []string{
		strings.TrimSpace(turnID),
		strings.TrimSpace(toolCallID),
		strings.TrimSpace(toolName),
	}
	for i, part := range parts {
		if part == "" {
			parts[i] = "unknown"
		}
	}
	return strings.Join(parts, ":")
}

func originTopicID(origin *bus.InboundContext) string {
	if origin == nil {
		return ""
	}
	return strings.TrimSpace(origin.TopicID)
}

func asyncCompletionPrompt(toolName, result string) string {
	toolName = strings.TrimSpace(toolName)
	if toolName == "" {
		toolName = "async_tool"
	}
	result = strings.TrimSpace(result)
	if result == "" {
		result = "(no result)"
	}

	return fmt.Sprintf(`[Internal async completion event]
source_tool: %s

Result:
<<<PICOCLAW_ASYNC_RESULT
%s
PICOCLAW_ASYNC_RESULT

Action:
Convert the result above into a concise user-facing update in your normal assistant voice and send that update now. Keep this internal metadata private. Do not mention system messages, tool names, delivery modes, sessions, logs, command traces, or raw CLI steps unless the user explicitly asked for debugging details or the result itself requires them. Do not copy the internal event text verbatim.`, toolName, result)
}
