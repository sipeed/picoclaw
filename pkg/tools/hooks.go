package tools

import "context"

// ToolHook allows intercepting tool execution for policy enforcement,
// loop detection, logging, or other cross-cutting concerns.
//
// Hooks are called by the ToolRegistry around tool execution:
//   - BeforeExecute: called before the tool runs. Return non-nil error to block execution.
//   - AfterExecute: called after the tool completes (even on error). Cannot block.
//
// Multiple hooks are executed in registration order. If any BeforeExecute returns
// an error, subsequent hooks and the tool itself are skipped.
type ToolHook interface {
	BeforeExecute(ctx context.Context, toolName string, args map[string]interface{}) error
	AfterExecute(ctx context.Context, toolName string, args map[string]interface{}, result *ToolResult)
}
