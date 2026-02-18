package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpgradeAgentMediaSection(t *testing.T) {
	t.Run("appends media section to existing AGENT.md", func(t *testing.T) {
		workspace := t.TempDir()
		agentPath := filepath.Join(workspace, "AGENT.md")
		os.WriteFile(agentPath, []byte("# Agent Instructions\n\nBe helpful."), 0644)

		UpgradeWorkspace(workspace)

		data, err := os.ReadFile(agentPath)
		if err != nil {
			t.Fatalf("reading AGENT.md: %v", err)
		}
		content := string(data)

		if !strings.Contains(content, "## Media & File Sending") {
			t.Error("expected media section to be appended")
		}
		if !strings.Contains(content, "You CAN send files directly") {
			t.Error("expected media instructions in content")
		}
		// Original content preserved
		if !strings.Contains(content, "# Agent Instructions") {
			t.Error("original content should be preserved")
		}
	})

	t.Run("idempotent — does not duplicate section", func(t *testing.T) {
		workspace := t.TempDir()
		agentPath := filepath.Join(workspace, "AGENT.md")
		os.WriteFile(agentPath, []byte("# Agent Instructions\n\nBe helpful."), 0644)

		UpgradeWorkspace(workspace)
		UpgradeWorkspace(workspace)

		data, _ := os.ReadFile(agentPath)
		count := strings.Count(string(data), "## Media & File Sending")
		if count != 1 {
			t.Errorf("expected exactly 1 media section, got %d", count)
		}
	})

	t.Run("skips when AGENT.md does not exist", func(t *testing.T) {
		workspace := t.TempDir()
		// No AGENT.md created — should not panic or error
		UpgradeWorkspace(workspace)
	})

	t.Run("skips when section already present", func(t *testing.T) {
		workspace := t.TempDir()
		agentPath := filepath.Join(workspace, "AGENT.md")
		original := "# Agent\n\n## Media & File Sending\n\nAlready here."
		os.WriteFile(agentPath, []byte(original), 0644)

		UpgradeWorkspace(workspace)

		data, _ := os.ReadFile(agentPath)
		if string(data) != original {
			t.Error("file should not be modified when section already exists")
		}
	})
}
