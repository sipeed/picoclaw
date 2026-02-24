package agent

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// SessionEntry represents an active or recently-active session.
type SessionEntry struct {
	SessionKey  string    `json:"session_key"`
	Channel     string    `json:"channel"`
	ChatID      string    `json:"chat_id"`
	TouchDir    string    `json:"touch_dir"`
	ProjectPath string    `json:"project_path,omitempty"` // canonical project path
	Purpose     string    `json:"purpose,omitempty"`      // 1-line task description
	Branch      string    `json:"branch,omitempty"`       // git branch name
	LastSeenAt  time.Time `json:"last_seen_at"`
}

// TouchMeta carries optional metadata for Touch calls.
type TouchMeta struct {
	ProjectPath string // canonical project path (always original workspace-relative)
	Purpose     string // 1-line task description
	Branch      string // git branch name
}

// PeerInfo is the minimal info shared between sessions on the same project.
type PeerInfo struct {
	SessionKey string
	Purpose    string
	Branch     string
}

// SessionTracker tracks per-session tool-call activity.
// Thread-safe; used by AgentLoop for plan coordination and by the mini app API for observability.
type SessionTracker struct {
	entries sync.Map // sessionKey → *SessionEntry
}

// NewSessionTracker creates a new tracker.
func NewSessionTracker() *SessionTracker {
	return &SessionTracker{}
}

const sessionActivityTimeout = 15 * time.Minute

// Touch records a tool-call activity for a session.
// dir is the workspace-relative directory the tool call targeted.
// If dir is empty, only LastSeenAt is updated.
// meta is optional and carries project coordination metadata.
func (st *SessionTracker) Touch(sessionKey, channel, chatID, dir string, meta *TouchMeta) {
	now := time.Now()
	val, loaded := st.entries.Load(sessionKey)
	if loaded {
		entry := val.(*SessionEntry)
		entry.LastSeenAt = now
		if dir != "" {
			entry.TouchDir = dir
		}
		if channel != "" {
			entry.Channel = channel
		}
		if chatID != "" {
			entry.ChatID = chatID
		}
		if meta != nil {
			if meta.ProjectPath != "" {
				entry.ProjectPath = meta.ProjectPath
			}
			if meta.Purpose != "" {
				entry.Purpose = meta.Purpose
			}
			if meta.Branch != "" {
				entry.Branch = meta.Branch
			}
		}
		return
	}
	entry := &SessionEntry{
		SessionKey: sessionKey,
		Channel:    channel,
		ChatID:     chatID,
		TouchDir:   dir,
		LastSeenAt: now,
	}
	if meta != nil {
		entry.ProjectPath = meta.ProjectPath
		entry.Purpose = meta.Purpose
		entry.Branch = meta.Branch
	}
	st.entries.Store(sessionKey, entry)
}

// IsActiveInDir returns true if any session (excluding those matching excludeKey)
// has touched a directory overlapping with dir within sessionActivityTimeout.
// Overlap = either is a prefix of the other (parent/child relationship).
func (st *SessionTracker) IsActiveInDir(dir, excludeKey string) bool {
	cutoff := time.Now().Add(-sessionActivityTimeout)
	active := false
	st.entries.Range(func(key, val any) bool {
		if key.(string) == excludeKey {
			return true
		}
		entry := val.(*SessionEntry)
		if entry.LastSeenAt.After(cutoff) && entry.TouchDir != "" &&
			(strings.HasPrefix(entry.TouchDir, dir) || strings.HasPrefix(dir, entry.TouchDir)) {
			active = true
			return false
		}
		return true
	})
	return active
}

// ListActive returns all sessions seen within sessionActivityTimeout,
// sorted by LastSeenAt descending (most recent first).
func (st *SessionTracker) ListActive() []SessionEntry {
	cutoff := time.Now().Add(-sessionActivityTimeout)
	var result []SessionEntry
	st.entries.Range(func(key, val any) bool {
		entry := val.(*SessionEntry)
		if entry.LastSeenAt.After(cutoff) {
			result = append(result, *entry) // copy
		}
		return true
	})
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastSeenAt.After(result[j].LastSeenAt)
	})
	return result
}

// GetPeerPurposes returns purposes of other active sessions targeting the same project.
// Used for lightweight coordination without context pollution.
func (st *SessionTracker) GetPeerPurposes(sessionKey, projectPath string) []PeerInfo {
	if projectPath == "" {
		return nil
	}
	cutoff := time.Now().Add(-sessionActivityTimeout)
	var result []PeerInfo
	st.entries.Range(func(key, val any) bool {
		if key.(string) == sessionKey {
			return true
		}
		entry := val.(*SessionEntry)
		if entry.LastSeenAt.After(cutoff) && entry.ProjectPath == projectPath {
			result = append(result, PeerInfo{
				SessionKey: entry.SessionKey,
				Purpose:    entry.Purpose,
				Branch:     entry.Branch,
			})
		}
		return true
	})
	return result
}
