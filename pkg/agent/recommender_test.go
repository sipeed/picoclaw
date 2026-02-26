package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillRecommender_ChannelSpecificSkillLoading(t *testing.T) {
	workspace := t.TempDir()
	globalSkills := t.TempDir()
	builtinSkills := t.TempDir()

	// Create channel-specific skills
	createSkill := func(basePath, name, desc string) {
		skillDir := filepath.Join(basePath, "skills", name)
		require.NoError(t, os.MkdirAll(skillDir, 0o755))

		skillFile := filepath.Join(skillDir, "SKILL.md")
		content := `---
name: ` + name + `
description: ` + desc + `
---

# ` + name
		require.NoError(t, os.WriteFile(skillFile, []byte(content), 0o644))
	}

	// Telegram-specific skills
	createSkill(workspace, "telegram-sticker", "Create and send stickers on Telegram")
	createSkill(workspace, "telegram-poll", "Create polls in Telegram groups")

	// Slack-specific skills
	createSkill(workspace, "slack-huddle", "Start Slack huddle calls")
	createSkill(workspace, "slack-workflow", "Create Slack workflows")

	// WeCom-specific skills
	createSkill(workspace, "wecom-approval", "Handle WeCom approval requests")
	createSkill(workspace, "wecom-meeting", "Schedule WeCom meetings")

	// General skills (available on all channels)
	createSkill(workspace, "web-search", "Search the web for information")
	createSkill(workspace, "file-manager", "Manage files")

	skillsLoader := skills.NewSkillsLoader(workspace, globalSkills, builtinSkills)

	// Verify skills are loaded
	allSkills := skillsLoader.ListSkills()
	assert.Len(t, allSkills, 8)

	t.Run("Telegram channel loads Telegram skills", func(t *testing.T) {
		filtered := skillsLoader.BuildSkillsSummaryFiltered([]string{
			"telegram-sticker",
			"telegram-poll",
			"web-search",
		})

		assert.Contains(t, filtered, "telegram-sticker")
		assert.Contains(t, filtered, "telegram-poll")
		assert.Contains(t, filtered, "web-search")
		assert.NotContains(t, filtered, "slack-huddle")
		assert.NotContains(t, filtered, "wecom-approval")
	})

	t.Run("Slack channel loads Slack skills", func(t *testing.T) {
		filtered := skillsLoader.BuildSkillsSummaryFiltered([]string{
			"slack-huddle",
			"slack-workflow",
			"web-search",
		})

		assert.Contains(t, filtered, "slack-huddle")
		assert.Contains(t, filtered, "slack-workflow")
		assert.Contains(t, filtered, "web-search")
		assert.NotContains(t, filtered, "telegram-sticker")
		assert.NotContains(t, filtered, "wecom-meeting")
	})

	t.Run("WeCom channel loads WeCom skills", func(t *testing.T) {
		filtered := skillsLoader.BuildSkillsSummaryFiltered([]string{
			"wecom-approval",
			"wecom-meeting",
			"file-manager",
		})

		assert.Contains(t, filtered, "wecom-approval")
		assert.Contains(t, filtered, "wecom-meeting")
		assert.Contains(t, filtered, "file-manager")
		assert.NotContains(t, filtered, "telegram-poll")
		assert.NotContains(t, filtered, "slack-huddle")
	})
}

func TestSkillRecommender_Integration_ChannelBasedRecommendation(t *testing.T) {
	workspace := t.TempDir()
	globalSkills := t.TempDir()
	builtinSkills := t.TempDir()

	// Create channel-specific skills with descriptive names
	createSkill := func(basePath, name, desc string) {
		skillDir := filepath.Join(basePath, "skills", name)
		require.NoError(t, os.MkdirAll(skillDir, 0o755))

		skillFile := filepath.Join(skillDir, "SKILL.md")
		content := `---
name: ` + name + `
description: ` + desc + `
---

# ` + name
		require.NoError(t, os.WriteFile(skillFile, []byte(content), 0o644))
	}

	// Telegram skills
	createSkill(workspace, "send-sticker", "Send sticker messages on Telegram")
	createSkill(workspace, "create-poll", "Create polls in Telegram groups")

	// Slack skills
	createSkill(workspace, "start-huddle", "Start a huddle call in Slack")
	createSkill(workspace, "manage-emoji", "Manage custom emojis in Slack")

	// WeCom skills
	createSkill(workspace, "submit-approval", "Submit approval request in WeCom")
	createSkill(workspace, "book-meeting-room", "Book meeting room in WeCom")

	// General skills
	createSkill(workspace, "search-web", "Search web for information")

	skillsLoader := skills.NewSkillsLoader(workspace, globalSkills, builtinSkills)

	// Test that each channel has distinct skill sets
	t.Run("Telegram specific skills", func(t *testing.T) {
		summary := skillsLoader.BuildSkillsSummary()
		assert.Contains(t, summary, "send-sticker")
		assert.Contains(t, summary, "create-poll")
	})

	t.Run("Slack specific skills", func(t *testing.T) {
		summary := skillsLoader.BuildSkillsSummary()
		assert.Contains(t, summary, "start-huddle")
		assert.Contains(t, summary, "manage-emoji")
	})

	t.Run("WeCom specific skills", func(t *testing.T) {
		summary := skillsLoader.BuildSkillsSummary()
		assert.Contains(t, summary, "submit-approval")
		assert.Contains(t, summary, "book-meeting-room")
	})

	t.Run("Filter by channel type", func(t *testing.T) {
		// Simulate channel-based filtering
		telegramSkills := []string{"send-sticker", "create-poll", "search-web"}
		slackSkills := []string{"start-huddle", "manage-emoji", "search-web"}
		wecomSkills := []string{"submit-approval", "book-meeting-room", "search-web"}

		telegramSummary := skillsLoader.BuildSkillsSummaryFiltered(telegramSkills)
		slackSummary := skillsLoader.BuildSkillsSummaryFiltered(slackSkills)
		wecomSummary := skillsLoader.BuildSkillsSummaryFiltered(wecomSkills)

		// Verify each channel gets its specific skills
		assert.Contains(t, telegramSummary, "send-sticker")
		assert.Contains(t, telegramSummary, "create-poll")
		assert.NotContains(t, telegramSummary, "start-huddle")

		assert.Contains(t, slackSummary, "start-huddle")
		assert.Contains(t, slackSummary, "manage-emoji")
		assert.NotContains(t, slackSummary, "send-sticker")

		assert.Contains(t, wecomSummary, "submit-approval")
		assert.Contains(t, wecomSummary, "book-meeting-room")
		assert.NotContains(t, wecomSummary, "create-poll")
	})
}
