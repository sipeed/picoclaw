// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
)

// TestSafeFileManager_Concurrency tests that SafeFileManager safely handles concurrent reads and writes.
func TestSafeFileManager_Concurrency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "picoclaw-review-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fileMgr := NewSafeFileManager()
	filePath := filepath.Join(tmpDir, "test.txt")

	// Concurrent writes
	var wg sync.WaitGroup
	numWorkers := 20
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			content := fmt.Sprintf("content-from-worker-%d", workerID)
			_ = fileMgr.WriteFile(filePath, []byte(content), 0600)
		}(i)
	}
	wg.Wait()

	// Verify we can read the file safely without race
	data, err := fileMgr.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty file content")
	}
}

// TestCleanMarkdownJSON verifies that markdown wrapped JSON contents are stripped correctly.
func TestCleanMarkdownJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "```json\n{\"actions\": []}\n```",
			expected: "{\"actions\": []}",
		},
		{
			input:    "```\n{\"actions\": []}\n```",
			expected: "{\"actions\": []}",
		},
		{
			input:    "   {\"actions\": []}   ",
			expected: "{\"actions\": []}",
		},
	}

	for _, tc := range tests {
		got := cleanMarkdownJSON(tc.input)
		if got != tc.expected {
			t.Errorf("cleanMarkdownJSON(%q) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}

// TestIsSafePath verifies that directory traversal attempts are blocked.
func TestIsSafePath(t *testing.T) {
	workspace := "/var/picoclaw/workspace"
	tests := []struct {
		path     string
		expected bool
	}{
		{path: "USER.md", expected: true},
		{path: "skills/test/SKILL.md", expected: true},
		{path: "../USER.md", expected: false},
		{path: "/etc/passwd", expected: false},
		{path: "skills/../../USER.md", expected: false},
	}

	for _, tc := range tests {
		got := isSafePath(workspace, tc.path)
		if got != tc.expected {
			t.Errorf("isSafePath(%q, %q) = %v; want %v", workspace, tc.path, got, tc.expected)
		}
	}
}

// mockLLMProvider mock LLMProvider interface for unit testing.
type mockLLMProvider struct {
	respContent string
	err         error
}

func (m *mockLLMProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &providers.LLMResponse{
		Content: m.respContent,
	}, nil
}

func (m *mockLLMProvider) GetDefaultModel() string {
	return "mock-model"
}

// TestRunImmediateReview verifies that runImmediateReview correctly parses JSON and writes files.
func TestRunImmediateReview(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "picoclaw-review-run-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up agent instance and session manager
	sessionsDir := filepath.Join(tmpDir, "sessions")
	os.MkdirAll(sessionsDir, 0755)
	sessionStore := session.NewSessionManager(sessionsDir)

	mockResp := `{
		"actions": [
			{
				"file_path": "USER.md",
				"content": "# User Profile\nPrefers short answers.",
				"action": "write"
			},
			{
				"file_path": "skills/golang-helper/SKILL.md",
				"content": "# Golang Helper\nTips for go coding.",
				"action": "write"
			}
		],
		"reason": "Updated user profile and added go skill"
	}`

	mockProv := &mockLLMProvider{respContent: mockResp}
	agent := &AgentInstance{
		Workspace: tmpDir,
		Model:     "mock-model",
		Provider:  mockProv,
		Sessions:  sessionStore,
	}

	sessionKey := "test-session-1"
	agent.Sessions.AddFullMessage(sessionKey, providers.Message{
		Role:    "user",
		Content: "Remember that I prefer short answers.",
	})
	agent.Sessions.AddFullMessage(sessionKey, providers.Message{
		Role:    "assistant",
		Content: "Sure, I will keep my responses short from now on.",
	})

	fileMgr := NewSafeFileManager()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = runImmediateReview(ctx, agent, sessionKey, fileMgr)
	if err != nil {
		t.Fatalf("runImmediateReview failed: %v", err)
	}

	// Verify USER.md contents
	userFile := filepath.Join(tmpDir, "USER.md")
	userData, err := fileMgr.ReadFile(userFile)
	if err != nil {
		t.Fatalf("failed to read USER.md: %v", err)
	}
	expectedUser := "# User Profile\nPrefers short answers."
	if string(userData) != expectedUser {
		t.Errorf("USER.md content = %q; want %q", string(userData), expectedUser)
	}

	// Verify SKILL.md contents
	skillFile := filepath.Join(tmpDir, "skills", "golang-helper", "SKILL.md")
	skillData, err := fileMgr.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("failed to read SKILL.md: %v", err)
	}
	expectedSkill := "# Golang Helper\nTips for go coding."
	if string(skillData) != expectedSkill {
		t.Errorf("SKILL.md content = %q; want %q", string(skillData), expectedSkill)
	}
}
