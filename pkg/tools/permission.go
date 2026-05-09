package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/sipeed/picoclaw/pkg/logger"
)

type PermissionCache struct {
	mu    sync.RWMutex
	perms map[string]string // path → "once"|"session"|"denied"
}

func NewPermissionCache() *PermissionCache {
	return &PermissionCache{
		perms: make(map[string]string),
	}
}

// Check returns "once", "session", "denied", or "" (no permission)
func (pc *PermissionCache) Check(path string) string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	// Check exact match
	if val, ok := pc.perms[path]; ok {
		return val
	}

	// Check if any parent path is granted "session"
	for cachedPath, val := range pc.perms {
		if val == "session" && len(cachedPath) < len(path) && path[:len(cachedPath)] == cachedPath {
			return "session"
		}
	}

	return ""
}

func (pc *PermissionCache) Grant(path, duration string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.perms[path] = duration
}

func (pc *PermissionCache) Revoke(path string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	delete(pc.perms, path)
}

// RequestPermissionTool prompts user for permission when exec tool needs access outside workspace.
type RequestPermissionTool struct {
	cache *PermissionCache
}

func NewRequestPermissionTool(cache *PermissionCache) *RequestPermissionTool {
	return &RequestPermissionTool{cache: cache}
}

func (t *RequestPermissionTool) Name() string {
	return "request_permission"
}

func (t *RequestPermissionTool) Description() string {
	return "Request user permission for accessing paths outside workspace. Returns prompt for user."
}

func (t *RequestPermissionTool) PromptMetadata() PromptMetadata {
	return PromptMetadata{
		Layer:  ToolPromptLayerCapability,
		Slot:   ToolPromptSlotTooling,
		Source: ToolPromptSourceRegistry,
	}
}

func (t *RequestPermissionTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path that needs permission",
			},
			"command": map[string]any{
				"type":        "string",
				"description": "Original command (for context)",
			},
		},
		"required": []string{"path"},
	}
}

func (t *RequestPermissionTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["path"].(string)
	command, _ := args["command"].(string)

	logger.InfoCF("permission", "Permission request", map[string]any{"path": path, "command": command})

	// Return structured prompt that LLM will show to user
	return &ToolResult{
		ForLLM: fmt.Sprintf("User permission needed for %s. Ask user: 'Allow once' or 'Allow for session'?", path),
		ForUser: fmt.Sprintf("⚠️ **Permission Required**\n\nPicoClaw wants to execute: `%s`\nThis accesses `%s` which is outside your workspace.\n\n**How would you like to proceed?**\n- Type `once` for one-time access\n- Type `session` to allow all access to this path for this session\n- Type `no` to deny", command, path),
	}
}
