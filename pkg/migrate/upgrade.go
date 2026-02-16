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
