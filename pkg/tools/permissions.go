package tools

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
)

// PermissionFunc asks the user for permission to access a directory outside the workspace.
// Returns true if approved, false if denied. Implementations should block until the user responds.
type PermissionFunc func(ctx context.Context, path string) (bool, error)

// PermissionFuncFactory creates a PermissionFunc for a given channel and chatID.
// This allows channel-specific permission implementations (CLI stdin, Telegram buttons, etc.)
type PermissionFuncFactory func(channel, chatID string) PermissionFunc

// PermissibleTool is an optional interface that tools can implement
// to support permission-based access to paths outside the workspace.
type PermissibleTool interface {
	Tool
	SetPermission(store *PermissionStore, fn PermissionFunc)
}

// PermissionStore tracks approved directories for a session.
type PermissionStore struct {
	mu       sync.RWMutex
	approved map[string]struct{}
}

func NewPermissionStore() *PermissionStore {
	return &PermissionStore{
		approved: make(map[string]struct{}),
	}
}

func (ps *PermissionStore) Approve(dir string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.approved[filepath.Clean(dir)] = struct{}{}
}

func (ps *PermissionStore) IsApproved(path string) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	cleanPath := filepath.Clean(path)
	for dir := range ps.approved {
		if cleanPath == dir || strings.HasPrefix(cleanPath, dir+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
