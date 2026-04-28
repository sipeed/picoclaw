package toolshared

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/session"
)

// Tool is the interface that all tools must implement.
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, args map[string]any) *ToolResult
}

const (
	ToolPromptLayerCapability = "capability"
	ToolPromptSlotTooling     = "tooling"
	ToolPromptSlotMCP         = "mcp"
	ToolPromptSourceRegistry  = "tool_registry:native"
	ToolPromptSourceDiscovery = "tool_registry:discovery"
)

type PromptMetadata struct {
	Layer  string
	Slot   string
	Source string
}

type PromptMetadataProvider interface {
	PromptMetadata() PromptMetadata
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
	ctxKeyMessageID        = &toolCtxKey{"messageID"}
	ctxKeyReplyToMessageID = &toolCtxKey{"replyToMessageID"}
	ctxKeyAgentID          = &toolCtxKey{"agentID"}
	ctxKeySessionKey       = &toolCtxKey{"sessionKey"}
	ctxKeySessionScope     = &toolCtxKey{"sessionScope"}
	ctxKeyMCPHeaders       = &toolCtxKey{"mcpHeaders"}
)

// WithToolContext returns a child context carrying channel and chatID.
func WithToolContext(ctx context.Context, channel, chatID string) context.Context {
	ctx = context.WithValue(ctx, ctxKeyChannel, channel)
	ctx = context.WithValue(ctx, ctxKeyChatID, chatID)
	return ctx
}

// WithToolMessageContext returns a child context carrying inbound message IDs.
func WithToolMessageContext(ctx context.Context, messageID, replyToMessageID string) context.Context {
	ctx = context.WithValue(ctx, ctxKeyMessageID, messageID)
	ctx = context.WithValue(ctx, ctxKeyReplyToMessageID, replyToMessageID)
	return ctx
}

// WithToolInboundContext returns a child context carrying channel/chat and inbound IDs.
func WithToolInboundContext(
	ctx context.Context,
	channel, chatID, messageID, replyToMessageID string,
) context.Context {
	ctx = WithToolContext(ctx, channel, chatID)
	ctx = WithToolMessageContext(ctx, messageID, replyToMessageID)
	return ctx
}

// WithToolSessionContext returns a child context carrying turn-scoped session metadata.
func WithToolSessionContext(
	ctx context.Context,
	agentID, sessionKey string,
	scope *session.SessionScope,
) context.Context {
	ctx = context.WithValue(ctx, ctxKeyAgentID, agentID)
	ctx = context.WithValue(ctx, ctxKeySessionKey, sessionKey)
	ctx = context.WithValue(ctx, ctxKeySessionScope, session.CloneScope(scope))
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

// ToolMessageID extracts the current inbound message ID from ctx, or "" if unset.
func ToolMessageID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyMessageID).(string)
	return v
}

// ToolReplyToMessageID extracts the current inbound reply target from ctx, or "" if unset.
func ToolReplyToMessageID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyReplyToMessageID).(string)
	return v
}

// ToolAgentID extracts the active turn's agent ID from ctx, or "" if unset.
func ToolAgentID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyAgentID).(string)
	return v
}

// ToolSessionKey extracts the active turn's session key from ctx, or "" if unset.
func ToolSessionKey(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeySessionKey).(string)
	return v
}

// ToolSessionScope extracts the active turn's structured session scope from ctx.
func ToolSessionScope(ctx context.Context) *session.SessionScope {
	scope, _ := ctx.Value(ctxKeySessionScope).(*session.SessionScope)
	return session.CloneScope(scope)
}

// WithMCPHeaders returns a child context carrying per-request headers for MCP HTTP transports.
func WithMCPHeaders(ctx context.Context, headers map[string]string) context.Context {
	return context.WithValue(ctx, ctxKeyMCPHeaders, headers)
}

// MCPHeaders extracts per-request MCP headers from ctx, or nil if unset.
func MCPHeaders(ctx context.Context) map[string]string {
	v, _ := ctx.Value(ctxKeyMCPHeaders).(map[string]string)
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
