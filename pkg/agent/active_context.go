// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// ActiveContext holds the structured per-channel context that Phase 1 uses
// to understand short/ambiguous user messages.
//
// Design choices:
//   - CurrentFiles: last 5 file paths touched by tool calls (read/write/edit/append/list_dir).
//   - RecentErrors: last 3 tool failure messages.
//   - CurrentTask / RecentSummaries are intentionally omitted — they overlap with
//     the recent-M turns in instant memory and would be redundant.
type ActiveContext struct {
	CurrentFiles []string `json:"current_files"` // newest first, max 5
	RecentErrors []string `json:"recent_errors"` // newest first, max 3
}

// ActiveContextStore is a thread-safe in-memory map of channel:chatID → ActiveContext.
// On startup it is loaded from disk; on stop it is flushed back.
type ActiveContextStore struct {
	mu   sync.RWMutex
	data map[string]*ActiveContext // key = "channel:chatID"
}

// NewActiveContextStore creates an empty store.
func NewActiveContextStore() *ActiveContextStore {
	return &ActiveContextStore{
		data: make(map[string]*ActiveContext),
	}
}

// Get returns a copy of the ActiveContext for the given key (never nil).
func (s *ActiveContextStore) Get(key string) *ActiveContext {
	s.mu.RLock()
	ac, ok := s.data[key]
	s.mu.RUnlock()

	if !ok || ac == nil {
		return &ActiveContext{}
	}
	// Return a shallow copy to avoid callers mutating the store.
	cp := *ac
	cp.CurrentFiles = append([]string(nil), ac.CurrentFiles...)
	cp.RecentErrors = append([]string(nil), ac.RecentErrors...)
	return &cp
}

// fileExtractingTools is the set of tool names whose arguments may carry file paths.
// Keys are lowercase tool names; values indicate the argument name(s) to inspect.
var fileExtractingTools = map[string][]string{
	"read_file":   {"path", "file_path", "filename"},
	"write_file":  {"path", "file_path", "filename"},
	"edit_file":   {"path", "file_path", "filename"},
	"append_file": {"path", "file_path", "filename"},
	"list_dir":    {"path", "dir_path", "directory"},
}

// Update applies the outcomes of a completed turn to the ActiveContext for key.
// It extracts file paths from tool call arguments and captures error messages.
func (s *ActiveContextStore) Update(key string, input RuntimeInput) {
	if key == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ac, ok := s.data[key]
	if !ok || ac == nil {
		ac = &ActiveContext{}
		s.data[key] = ac
	}

	// Extract file paths from tool calls.
	for _, tc := range input.ToolCalls {
		name := strings.ToLower(tc.Name)
		argFields, relevant := fileExtractingTools[name]
		if !relevant {
			continue
		}
		// tc.Args is stored as JSON string or we can check tc.ArgsRaw if available.
		// Since ToolCallRecord only has Name/Error/Duration, we skip argument extraction
		// here and rely on callers passing a richer input in the future (M5).
		// For now we still handle errors.
		_ = argFields
	}

	// Capture tool errors.
	for _, tc := range input.ToolCalls {
		if tc.Error == "" {
			continue
		}
		msg := fmt.Sprintf("[%s] %s", tc.Name, tc.Error)
		// Prepend (newest first) and cap at 3.
		ac.RecentErrors = prependCapped(ac.RecentErrors, msg, 3)
	}
}

// UpdateWithFiles is an extended update that also receives file paths extracted
// by the loop (call this when tool argument parsing is available).
func (s *ActiveContextStore) UpdateWithFiles(key string, input RuntimeInput, filePaths []string) {
	s.Update(key, input)

	if len(filePaths) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ac, ok := s.data[key]
	if !ok || ac == nil {
		ac = &ActiveContext{}
		s.data[key] = ac
	}

	for _, p := range filePaths {
		if p != "" {
			ac.CurrentFiles = prependCapped(ac.CurrentFiles, p, 5)
		}
	}
}

// prependCapped prepends item to slice and caps the result at max length.
// Deduplicates: if item already exists it is moved to the front.
func prependCapped(slice []string, item string, max int) []string {
	// Remove duplicate.
	filtered := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			filtered = append(filtered, s)
		}
	}
	result := append([]string{item}, filtered...)
	if len(result) > max {
		result = result[:max]
	}
	return result
}

// Format renders the context as a markdown block for injection into a user message.
// Returns empty string when there is nothing to show.
func (ac *ActiveContext) Format() string {
	if len(ac.CurrentFiles) == 0 && len(ac.RecentErrors) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## Current Context\n")
	if len(ac.CurrentFiles) > 0 {
		sb.WriteString("Files in use: ")
		sb.WriteString(strings.Join(ac.CurrentFiles, ", "))
		sb.WriteString("\n")
	}
	if len(ac.RecentErrors) > 0 {
		sb.WriteString("Recent errors:\n")
		for _, e := range ac.RecentErrors {
			sb.WriteString("  - ")
			sb.WriteString(e)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// Persistence
// ---------------------------------------------------------------------------

// persistedStore is the on-disk JSON format for ActiveContextStore.
type persistedStore struct {
	Contexts map[string]*ActiveContext `json:"contexts"`
}

// Flush serialises the store to a JSON file at the given path.
func (s *ActiveContextStore) Flush(path string) error {
	s.mu.RLock()
	out := persistedStore{Contexts: make(map[string]*ActiveContext, len(s.data))}
	for k, v := range s.data {
		cp := *v
		cp.CurrentFiles = append([]string(nil), v.CurrentFiles...)
		cp.RecentErrors = append([]string(nil), v.RecentErrors...)
		out.Contexts[k] = &cp
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("active_context: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("active_context: write %s: %w", path, err)
	}
	logger.DebugCF("active_context", "Flushed to disk", map[string]any{"path": path, "keys": len(out.Contexts)})
	return nil
}

// Load deserialises the store from a JSON file at the given path.
// Missing or unreadable files are silently ignored (returns nil).
func (s *ActiveContextStore) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("active_context: read %s: %w", path, err)
	}
	var out persistedStore
	if err := json.Unmarshal(data, &out); err != nil {
		return fmt.Errorf("active_context: unmarshal: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range out.Contexts {
		if v != nil {
			s.data[k] = v
		}
	}
	logger.DebugCF("active_context", "Loaded from disk", map[string]any{"path": path, "keys": len(out.Contexts)})
	return nil
}
