package tools

import (
	"sync"
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
