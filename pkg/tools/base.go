package tools

import "context"

// Tool is the interface that all tools must implement.
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, args map[string]any) *ToolResult
}

// --- Request-scoped tool context (channel / chatID) ---
//
// Carried via context.Value so that concurrent tool calls each receive
// their own immutable copy — no mutable state on singleton tool instances.
//
// Keys are unexported pointer-typed vars — guaranteed collision-free,
// and only accessible through the helper functions below.

type toolCtxKey struct{ name string }

var (
	ctxKeyChannel          = &toolCtxKey{"channel"}
	ctxKeyChatID           = &toolCtxKey{"chatID"}
	ctxKeySessionKey       = &toolCtxKey{"sessionKey"}
	ctxKeyCurrentMessageID = &toolCtxKey{"currentMessageID"}
	ctxKeyParentMessageID  = &toolCtxKey{"parentMessageID"}
)

// WithToolContext returns a child context carrying channel and chatID.
func WithToolContext(ctx context.Context, channel, chatID string) context.Context {
	ctx = context.WithValue(ctx, ctxKeyChannel, channel)
	ctx = context.WithValue(ctx, ctxKeyChatID, chatID)
	return ctx
}

// WithToolSessionKey returns a child context carrying the session key.
func WithToolSessionKey(ctx context.Context, sessionKey string) context.Context {
	return context.WithValue(ctx, ctxKeySessionKey, sessionKey)
}

// WithToolReplyContext returns a child context carrying the current and parent
// inbound platform message IDs for reply routing decisions.
func WithToolReplyContext(
	ctx context.Context,
	currentMessageID, parentMessageID string,
) context.Context {
	ctx = context.WithValue(ctx, ctxKeyCurrentMessageID, currentMessageID)
	ctx = context.WithValue(ctx, ctxKeyParentMessageID, parentMessageID)
	return ctx
}

// ToolChannel extracts the channel from ctx, or "" if unset.
func ToolChannel(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyChannel).(string)
	return v
}

// ToolChatID extracts the chatID from ctx, or "" if unset.
func ToolChatID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyChatID).(string)
	return v
}

// ToolSessionKey extracts the session key from ctx, or "" if unset.
func ToolSessionKey(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeySessionKey).(string)
	return v
}

// ToolCurrentMessageID extracts the current inbound platform message ID.
func ToolCurrentMessageID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyCurrentMessageID).(string)
	return v
}

// ToolParentMessageID extracts the parent/replied-to inbound platform message ID.
func ToolParentMessageID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyParentMessageID).(string)
	return v
}

// AsyncCallback is a function type that async tools use to notify completion.
// When an async tool finishes its work, it calls this callback with the result.
//
// The ctx parameter allows the callback to be canceled if the agent is shutting down.
// The result parameter contains the tool's execution result.
type AsyncCallback func(ctx context.Context, result *ToolResult)

// AsyncExecutor is an optional interface that tools can implement to support
// asynchronous execution with completion callbacks.
//
// Unlike the old AsyncTool pattern (SetCallback + Execute), AsyncExecutor
// receives the callback as a parameter of ExecuteAsync. This eliminates the
// data race where concurrent calls could overwrite each other's callbacks
// on a shared tool instance.
//
// This is useful for:
//   - Long-running operations that shouldn't block the agent loop
//   - Subagent spawns that complete independently
//   - Background tasks that need to report results later
//
// Example:
//
//	func (t *SpawnTool) ExecuteAsync(ctx context.Context, args map[string]any, cb AsyncCallback) *ToolResult {
//	    go func() {
//	        result := t.runSubagent(ctx, args)
//	        if cb != nil { cb(ctx, result) }
//	    }()
//	    return AsyncResult("Subagent spawned, will report back")
//	}
type AsyncExecutor interface {
	Tool
	// ExecuteAsync runs the tool asynchronously. The callback cb will be
	// invoked (possibly from another goroutine) when the async operation
	// completes. cb is guaranteed to be non-nil by the caller (registry).
	ExecuteAsync(ctx context.Context, args map[string]any, cb AsyncCallback) *ToolResult
}

// SequentialTool marks tools that must execute in model order within a single
// LLM turn, instead of being fanned out in parallel with sibling tool calls.
// This is intended for tools whose calls mutate shared state and can depend on
// earlier calls from the same assistant message.
type SequentialTool interface {
	Tool
	ExecuteSequentially() bool
}

// AvailabilityAwareTool marks tools whose visibility depends on the current
// request context, such as channel-specific tools.
type AvailabilityAwareTool interface {
	Tool
	Available(ctx context.Context) bool
}

func ToolToSchema(tool Tool) map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        tool.Name(),
			"description": tool.Description(),
			"parameters":  tool.Parameters(),
		},
	}
}

// AdvancedMessageManager represents tools that require direct, synchronous
// interaction with messaging channels (e.g., sending placeholders and editing messages).
type AdvancedMessageManager interface {
	Tool
	SetCallbacks(
		sendPlaceholder func(ctx context.Context, channel, chatID, content string) (string, error),
		editMessage func(ctx context.Context, channel, chatID, messageID, content string) error,
	)
}
