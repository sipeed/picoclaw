package discord

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
)

type DiscordCommander interface {
	Help(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error
	Show(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error
	List(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error
	Switch(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error
}

type cmd struct {
	session *discordgo.Session
	config  *config.Config
	bus     *bus.MessageBus
}

func NewDiscordCommands(session *discordgo.Session, cfg *config.Config, bus *bus.MessageBus) DiscordCommander {
	return &cmd{
		session: session,
		config:  cfg,
		bus:     bus,
	}
}

// parseCommand extracts the command and arguments from message content.
func parseCommand(content string) (string, string) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "/") {
		return "", ""
	}

	parts := strings.SplitN(content, " ", 2)
	cmd := strings.TrimPrefix(parts[0], "/")

	if len(parts) < 2 {
		return cmd, ""
	}

	return cmd, strings.TrimSpace(parts[1])
}

func (c *cmd) Help(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error {
	msg := `**PicoClaw Commands**

/start - Start the bot
/help - Show this help message
/show [model|channel] - Show current configuration
/list [models|channels] - List available options
/switch model <name> - Switch to a different model

**Examples:**
/switch model gpt-4
/switch model claude-sonnet-4.6

Use /list models to see all available models.
`

	_, err := s.ChannelMessageSend(m.ChannelID, msg)
	return err
}

func (c *cmd) Show(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error {
	cmd, args := parseCommand(m.Content)
	if cmd != "show" {
		return fmt.Errorf("invalid command format")
	}

	if args == "" {
		_, err := s.ChannelMessageSend(m.ChannelID, "Usage: /show [model|channel]")
		return err
	}

	var response string
	switch args {
	case "model":
		currentModel := c.config.Agents.Defaults.GetModelName()
		provider := c.config.Agents.Defaults.Provider
		response = fmt.Sprintf("**Current Model:** %s\n**Provider:** %s", currentModel, provider)
	case "channel":
		response = "**Current Channel:** discord"
	default:
		response = fmt.Sprintf("Unknown parameter: %s. Try 'model' or 'channel'.", args)
	}

	_, err := s.ChannelMessageSend(m.ChannelID, response)
	return err
}

func (c *cmd) List(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error {
	cmd, args := parseCommand(m.Content)
	if cmd != "list" {
		return fmt.Errorf("invalid command format")
	}

	if args == "" {
		_, err := s.ChannelMessageSend(m.ChannelID, "Usage: /list [models|channels]")
		return err
	}

	var response string
	switch args {
	case "models":
		response = c.formatModelsList()
	case "channels":
		response = c.formatChannelsList()
	default:
		response = fmt.Sprintf("Unknown parameter: %s. Try 'models' or 'channels'.", args)
	}

	_, err := s.ChannelMessageSend(m.ChannelID, response)
	return err
}

func (c *cmd) Switch(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error {
	cmd, args := parseCommand(m.Content)
	if cmd != "switch" {
		return fmt.Errorf("invalid command format")
	}

	if args == "" {
		_, err := s.ChannelMessageSend(m.ChannelID, "Usage: /switch model <name>\nUse /list models to see available models.")
		return err
	}

	// Parse "model <name>" format
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 || parts[0] != "model" {
		_, err := s.ChannelMessageSend(m.ChannelID, "Usage: /switch model <name>\nUse /list models to see available models.")
		return err
	}

	modelName := strings.TrimSpace(parts[1])

	// Optional: validate model exists to provide immediate feedback
	if _, err := c.config.GetModelConfig(modelName); err != nil {
		available := c.formatModelsList()
		_, sendErr := c.session.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf("❌ Model not found: %s\n\n**Available models:**\n%s", modelName, available),
		)
		return sendErr
	}

	if c.bus == nil {
		_, err := c.session.ChannelMessageSend(m.ChannelID, "❌ Internal error: message bus not initialized")
		return err
	}

	// Forward a normalized switch command to the agent loop via the message bus
	// so that the agent can apply the change and persist any in-memory state.
	inbound := bus.InboundMessage{
		Channel:  "discord",
		SenderID: m.Author.ID,
		Sender: bus.SenderInfo{
			Platform:    "discord",
			PlatformID:  m.Author.ID,
			CanonicalID: identity.BuildCanonicalID("discord", m.Author.ID),
			Username:    m.Author.Username,
			DisplayName: m.Author.Username,
		},
		ChatID:    m.ChannelID,
		Content:   fmt.Sprintf("/switch model to %s", modelName),
		MessageID: m.ID,
		Metadata: map[string]string{
			"guild_id":   m.GuildID,
			"channel_id": m.ChannelID,
		},
	}

	if err := c.bus.PublishInbound(ctx, inbound); err != nil {
		_, sendErr := c.session.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Failed to switch model: %v", err))
		return sendErr
	}

	// The agent will respond via the normal outbound flow; no immediate reply here.
	return nil
}

func (c *cmd) formatModelsList() string {
	if len(c.config.ModelList) == 0 {
		return "No models configured. Please check your configuration."
	}

	currentModel := c.config.Agents.Defaults.GetModelName()

	var sb strings.Builder
	sb.WriteString("**Available Models:**\n\n")

	for _, mc := range c.config.ModelList {
		if mc.ModelName == "" {
			continue
		}

		prefix := "  "
		if mc.ModelName == currentModel {
			prefix = "✓ "
		}

		provider := "openai"
		if strings.Contains(mc.Model, "/") {
			protocolParts := strings.SplitN(mc.Model, "/", 2)
			if len(protocolParts) > 0 {
				provider = protocolParts[0]
			}
		}

		sb.WriteString(fmt.Sprintf("%s**%s** - %s (%s)\n", prefix, mc.ModelName, mc.Model, provider))
	}

	return sb.String()
}

func (c *cmd) formatChannelsList() string {
	var enabled []string
	if c.config.Channels.Telegram.Enabled {
		enabled = append(enabled, "telegram")
	}
	if c.config.Channels.WhatsApp.Enabled {
		enabled = append(enabled, "whatsapp")
	}
	if c.config.Channels.Feishu.Enabled {
		enabled = append(enabled, "feishu")
	}
	if c.config.Channels.Discord.Enabled {
		enabled = append(enabled, "discord")
	}
	if c.config.Channels.Slack.Enabled {
		enabled = append(enabled, "slack")
	}
	if c.config.Channels.LINE.Enabled {
		enabled = append(enabled, "line")
	}
	if c.config.Channels.QQ.Enabled {
		enabled = append(enabled, "qq")
	}
	if c.config.Channels.OneBot.Enabled {
		enabled = append(enabled, "onebot")
	}
	if c.config.Channels.WeCom.Enabled {
		enabled = append(enabled, "wecom")
	}
	if c.config.Channels.WeComApp.Enabled {
		enabled = append(enabled, "wecom_app")
	}
	if c.config.Channels.WeComAIBot.Enabled {
		enabled = append(enabled, "wecom_aibot")
	}
	if c.config.Channels.Pico.Enabled {
		enabled = append(enabled, "pico")
	}

	if len(enabled) == 0 {
		return "No channels enabled."
	}

	return fmt.Sprintf("Enabled channels:\n- %s", strings.Join(enabled, "\n- "))
}
