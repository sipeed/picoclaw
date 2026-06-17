// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// SafeFileManager manages thread-safe read and write operations on files.
// It uses a map of RW mutexes per file path to ensure fine-grained locking.
type SafeFileManager struct {
	mu      sync.Mutex
	fileMus map[string]*sync.RWMutex
}

// NewSafeFileManager creates a new thread-safe file manager.
func NewSafeFileManager() *SafeFileManager {
	return &SafeFileManager{
		fileMus: make(map[string]*sync.RWMutex),
	}
}

func (sf *SafeFileManager) getMutex(path string) *sync.RWMutex {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	cleanPath := filepath.Clean(path)
	mu, exists := sf.fileMus[cleanPath]
	if !exists {
		mu = &sync.RWMutex{}
		sf.fileMus[cleanPath] = mu
	}
	return mu
}

// ReadFile reads the content of a file in a thread-safe manner.
func (sf *SafeFileManager) ReadFile(path string) ([]byte, error) {
	mu := sf.getMutex(path)
	mu.RLock()
	defer mu.RUnlock()

	return os.ReadFile(path)
}

// WriteFile writes the content to a file in a thread-safe manner using atomic write.
func (sf *SafeFileManager) WriteFile(path string, data []byte, perm fs.FileMode) error {
	mu := sf.getMutex(path)
	mu.Lock()
	defer mu.Unlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return fileutil.WriteFileAtomic(path, data, perm)
}

// ReviewAction represents a file update action returned by the review LLM.
type ReviewAction struct {
	FilePath string `json:"file_path"` // E.g., "USER.md" or "skills/git-helper/SKILL.md"
	Content  string `json:"content"`   // The new/updated content
	Action   string `json:"action"`    // "write"
}

// ReviewResponse represents the strict JSON structure returned by the reviewer.
type ReviewResponse struct {
	Actions []ReviewAction `json:"actions"`
	Reason  string         `json:"reason"`
}

// StartBackgroundReview triggers the background review process asynchronously.
func StartBackgroundReview(agent *AgentInstance, sessionKey string, fileMgr *SafeFileManager, wg *sync.WaitGroup) {
	if wg != nil {
		wg.Add(1)
	}
	go func() {
		if wg != nil {
			defer wg.Done()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		logger.DebugCF("agent", "Background review started", map[string]any{
			"session_key": sessionKey,
		})

		if err := runImmediateReview(ctx, agent, sessionKey, fileMgr); err != nil {
			logger.WarnCF("agent", "Background review failed", map[string]any{
				"session_key": sessionKey,
				"error":       err.Error(),
			})
		} else {
			logger.DebugCF("agent", "Background review finished", map[string]any{
				"session_key": sessionKey,
			})
		}
	}()
}

func runImmediateReview(ctx context.Context, agent *AgentInstance, sessionKey string, fileMgr *SafeFileManager) error {
	// Retrieve conversation history
	history := agent.Sessions.GetHistory(sessionKey)
	if len(history) == 0 {
		return nil
	}

	// Prepare messages for review
	reviewMessages := []providers.Message{
		{
			Role:    "system",
			Content: ReviewSystemPrompt,
		},
	}

	for _, msg := range history {
		// Suppress large multimodal content to keep background review fast and CGO-free / light
		reviewMessages = append(reviewMessages, providers.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	reviewMessages = append(reviewMessages, providers.Message{
		Role:    "user",
		Content: "Review the conversation above and output the JSON response now.",
	})

	// Invoke provider Chat method
	resp, err := agent.Provider.Chat(ctx, reviewMessages, nil, agent.Model, nil)
	if err != nil {
		return fmt.Errorf("llm call failed: %w", err)
	}

	// Clean JSON output (strip markdown tags if any)
	rawJSON := cleanMarkdownJSON(resp.Content)

	var reviewResp ReviewResponse
	if err := json.Unmarshal([]byte(rawJSON), &reviewResp); err != nil {
		return fmt.Errorf("failed to parse review response (raw: %s): %w", rawJSON, err)
	}

	// Apply review actions safely
	for _, action := range reviewResp.Actions {
		if !isSafePath(agent.Workspace, action.FilePath) {
			logger.WarnCF("agent", "Bypassed unsafe path write in review", map[string]any{
				"file_path": action.FilePath,
			})
			continue
		}

		absPath := filepath.Join(agent.Workspace, action.FilePath)
		if err := fileMgr.WriteFile(absPath, []byte(action.Content), 0600); err != nil {
			logger.ErrorCF("agent", "Failed to write file from background review", map[string]any{
				"path":  absPath,
				"error": err.Error(),
			})
		} else {
			logger.InfoCF("agent", "Successfully updated file via background review", map[string]any{
				"path": action.FilePath,
			})
		}
	}

	return nil
}

func cleanMarkdownJSON(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
	}
	return strings.TrimSpace(content)
}

func isSafePath(workspace, path string) bool {
	// If path is absolute, contains volume name (e.g. C:), or starts with slash, reject it
	if filepath.IsAbs(path) || filepath.VolumeName(path) != "" || strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return false
	}

	targetPath := filepath.Clean(filepath.Join(workspace, path))
	rel, err := filepath.Rel(workspace, targetPath)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}

// ReviewSystemPrompt defines the behavior of the background review agent.
const ReviewSystemPrompt = `You are a background curator and self-improvement assistant for the AI agent.
Your sole job is to review the conversation and decide whether to update the user profile (USER.md) or update/create skill library files (under skills/<skill_name>/SKILL.md).

Focus on:
1. Has the user corrected your style, tone, format, or verbosity? (e.g. "stop explaining", "make responses short")
2. Has the user revealed facts about themselves, their preferences, or work style?
3. Did you discover a non-trivial workaround, coding technique, or bugfix during the session?

Based on the signals, you must generate a JSON array of file modifications.
You can ONLY perform file modifications in this workspace. No other actions are allowed.

Your output MUST be a single JSON object matching this schema exactly:
{
  "actions": [
    {
      "file_path": "USER.md",
      "content": "Full contents of USER.md with updated notes about user preferences",
      "action": "write"
    },
    {
      "file_path": "skills/my-helper/SKILL.md",
      "content": "# My Helper\nDescription of the skill, procedures, and lessons learned...",
      "action": "write"
    }
  ],
  "reason": "Brief summary of the updates made"
}

Rules:
- If no changes are needed, return {"actions": [], "reason": "No updates needed"}.
- Return ONLY valid raw JSON. Do not include markdown code block syntax, conversational filler, or explanations outside the JSON structure.
- Always provide the FULL content of the file being written, not partial edits or diffs.`
