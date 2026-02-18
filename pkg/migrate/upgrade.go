package migrate

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// UpgradeWorkspace applies incremental upgrades to an existing workspace.
// Each upgrade is idempotent — it checks whether it has already been applied
// before making changes.
func UpgradeWorkspace(workspace string) {
	upgradeAgentMediaSection(workspace)
	upgradeAgentGrounding(workspace)
}

// upgradeAgentMediaSection ensures AGENT.md contains the media sending instructions.
// Added in v0.x to teach the LLM it can send files via the message tool.
func upgradeAgentMediaSection(workspace string) {
	agentPath := filepath.Join(workspace, "AGENT.md")

	data, err := os.ReadFile(agentPath)
	if err != nil {
		return // File doesn't exist or unreadable — skip
	}

	content := string(data)

	// Already applied
	if strings.Contains(content, "## Media & File Sending") {
		return
	}

	section := `

## Media & File Sending

You CAN send files directly to users. When you need to share a file (image, document, audio, video), use the ` + "`message`" + ` tool with the ` + "`media`" + ` parameter containing the local file path(s). The file will be delivered natively through the user's channel (Telegram, Discord, Slack, etc.). Do NOT tell users you cannot send files — just send them.`

	content += section

	if err := os.WriteFile(agentPath, []byte(content), 0644); err != nil {
		logger.ErrorCF("migrate", "Failed to upgrade AGENT.md", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	logger.InfoC("migrate", "Upgraded AGENT.md with media sending instructions")
}

// upgradeAgentGrounding ensures AGENT.md tells the LLM it is already connected
// and should not suggest manual workarounds like tokens or curl commands.
func upgradeAgentGrounding(workspace string) {
	agentPath := filepath.Join(workspace, "AGENT.md")

	data, err := os.ReadFile(agentPath)
	if err != nil {
		return
	}

	content := string(data)

	// Already applied
	if strings.Contains(content, "NEVER suggest manual workarounds") {
		return
	}

	// Replace the old generic intro with the grounded version
	oldIntro := "You are a helpful AI assistant. Be concise, accurate, and friendly."
	newIntro := "You are a helpful AI assistant running inside picoclaw. You are ALREADY connected to the user's chat channel (Telegram, Discord, Slack, etc.). When you use the `message` tool, your message is delivered directly to the user — you do NOT need API keys, bot tokens, or any external setup. Everything is already wired up for you. Just use your tools."

	if strings.Contains(content, oldIntro) {
		content = strings.Replace(content, oldIntro, newIntro, 1)
	}

	// Add guardrail lines to guidelines if not present
	if !strings.Contains(content, "NEVER tell users you lack access") {
		oldGuideline := "- Learn from user feedback"
		newGuideline := `- Learn from user feedback
- NEVER tell users you lack access to send messages, files, or perform actions — use your tools instead
- NEVER suggest manual workarounds (curl commands, scripts, tokens) for things your tools already do`
		content = strings.Replace(content, oldGuideline, newGuideline, 1)
	}

	if err := os.WriteFile(agentPath, []byte(content), 0644); err != nil {
		logger.ErrorCF("migrate", "Failed to upgrade AGENT.md grounding", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	logger.InfoC("migrate", "Upgraded AGENT.md with grounding instructions")
}
