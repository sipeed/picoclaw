package discord

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
	"github.com/sipeed/picoclaw/pkg/voice"
)

const (
	transcriptionTimeout = 30 * time.Second
	sendTimeout          = 10 * time.Second
)

type DiscordChannel struct {
	*channels.BaseChannel
	session     *discordgo.Session
	config      config.DiscordConfig
	transcriber *voice.GroqTranscriber
	ctx         context.Context
	cancel      context.CancelFunc
	typingMu    sync.Mutex
	typingStop  map[string]chan struct{} // chatID → stop signal
	botUserID   string                   // stored for mention checking
}

func NewDiscordChannel(cfg config.DiscordConfig, bus *bus.MessageBus) (*DiscordChannel, error) {
	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord session: %w", err)
	}

	base := channels.NewBaseChannel("discord", cfg, bus, cfg.AllowFrom, channels.WithMaxMessageLength(2000))

	return &DiscordChannel{
		BaseChannel: base,
		session:     session,
		config:      cfg,
		transcriber: nil,
		ctx:         context.Background(),
		typingStop:  make(map[string]chan struct{}),
	}, nil
}

func (c *DiscordChannel) SetTranscriber(transcriber *voice.GroqTranscriber) {
	c.transcriber = transcriber
}

func (c *DiscordChannel) Start(ctx context.Context) error {
	logger.InfoC("discord", "Starting Discord bot")

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Get bot user ID before opening session to avoid race condition
	botUser, err := c.session.User("@me")
	if err != nil {
		return fmt.Errorf("failed to get bot user: %w", err)
	}
	c.botUserID = botUser.ID

	c.session.AddHandler(c.handleMessage)

	if err := c.session.Open(); err != nil {
		return fmt.Errorf("failed to open discord session: %w", err)
	}

	c.SetRunning(true)

	logger.InfoCF("discord", "Discord bot connected", map[string]any{
		"username": botUser.Username,
		"user_id":  botUser.ID,
	})

	return nil
}

func (c *DiscordChannel) Stop(ctx context.Context) error {
	logger.InfoC("discord", "Stopping Discord bot")
	c.SetRunning(false)

	// Stop all typing goroutines before closing session
	c.typingMu.Lock()
	for chatID, stop := range c.typingStop {
		close(stop)
		delete(c.typingStop, chatID)
	}
	c.typingMu.Unlock()

	// Cancel our context so typing goroutines using c.ctx.Done() exit
	if c.cancel != nil {
		c.cancel()
	}

	if err := c.session.Close(); err != nil {
		return fmt.Errorf("failed to close discord session: %w", err)
	}

	return nil
}

func (c *DiscordChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	c.stopTyping(msg.ChatID)

	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	channelID := msg.ChatID
	if channelID == "" {
		return fmt.Errorf("channel ID is empty")
	}

	if len([]rune(msg.Content)) == 0 {
		return nil
	}

	return c.sendChunk(ctx, channelID, msg.Content)
}

func (c *DiscordChannel) sendChunk(ctx context.Context, channelID, content string) error {
	// Use the passed ctx for timeout control
	sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := c.session.ChannelMessageSend(channelID, content)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("discord send: %w", channels.ErrTemporary)
		}
		return nil
	case <-sendCtx.Done():
		return sendCtx.Err()
	}
}

// appendContent safely appends content to existing text
func appendContent(content, suffix string) string {
	if content == "" {
		return suffix
	}
	return content + "\n" + suffix
}

func (c *DiscordChannel) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m == nil || m.Author == nil {
		return
	}

	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check allowlist first to avoid downloading attachments and transcribing for rejected users
	if !c.IsAllowed(m.Author.ID) {
		logger.DebugCF("discord", "Message rejected by allowlist", map[string]any{
			"user_id": m.Author.ID,
		})
		return
	}

	// If configured to only respond to mentions, check if bot is mentioned
	// Skip this check for DMs (GuildID is empty) - DMs should always be responded to
	if c.config.MentionOnly && m.GuildID != "" {
		isMentioned := false
		for _, mention := range m.Mentions {
			if mention.ID == c.botUserID {
				isMentioned = true
				break
			}
		}
		if !isMentioned {
			logger.DebugCF("discord", "Message ignored - bot not mentioned", map[string]any{
				"user_id": m.Author.ID,
			})
			return
		}
	}

	senderID := m.Author.ID
	senderName := m.Author.Username
	if m.Author.Discriminator != "" && m.Author.Discriminator != "0" {
		senderName += "#" + m.Author.Discriminator
	}

	content := m.Content
	content = c.stripBotMention(content)
	mediaPaths := make([]string, 0, len(m.Attachments))

	scope := channels.BuildMediaScope("discord", m.ChannelID, m.ID)

	// Helper to register a local file with the media store
	storeMedia := func(localPath, filename string) string {
		if store := c.GetMediaStore(); store != nil {
			ref, err := store.Store(localPath, media.MediaMeta{
				Filename: filename,
				Source:   "discord",
			}, scope)
			if err == nil {
				return ref
			}
		}
		return localPath // fallback
	}

	for _, attachment := range m.Attachments {
		isAudio := utils.IsAudioFile(attachment.Filename, attachment.ContentType)

		if isAudio {
			localPath := c.downloadAttachment(attachment.URL, attachment.Filename)
			if localPath != "" {
				transcribedText := ""
				if c.transcriber != nil && c.transcriber.IsAvailable() {
					ctx, cancel := context.WithTimeout(c.ctx, transcriptionTimeout)
					result, err := c.transcriber.Transcribe(ctx, localPath)
					cancel() // Release context resources immediately to avoid leaks in for loop

					if err != nil {
						logger.ErrorCF("discord", "Voice transcription failed", map[string]any{
							"error": err.Error(),
						})
						transcribedText = fmt.Sprintf("[audio: %s (transcription failed)]", attachment.Filename)
					} else {
						transcribedText = fmt.Sprintf("[audio transcription: %s]", result.Text)
						logger.DebugCF("discord", "Audio transcribed successfully", map[string]any{
							"text": result.Text,
						})
					}
				} else {
					transcribedText = fmt.Sprintf("[audio: %s]", attachment.Filename)
				}

				mediaPaths = append(mediaPaths, storeMedia(localPath, attachment.Filename))
				content = appendContent(content, transcribedText)
			} else {
				logger.WarnCF("discord", "Failed to download audio attachment", map[string]any{
					"url":      attachment.URL,
					"filename": attachment.Filename,
				})
				mediaPaths = append(mediaPaths, attachment.URL)
				content = appendContent(content, fmt.Sprintf("[attachment: %s]", attachment.URL))
			}
		} else {
			mediaPaths = append(mediaPaths, attachment.URL)
			content = appendContent(content, fmt.Sprintf("[attachment: %s]", attachment.URL))
		}
	}

	if content == "" && len(mediaPaths) == 0 {
		return
	}

	if content == "" {
		content = "[media only]"
	}

	// Start typing after all early returns — guaranteed to have a matching Send()
	c.startTyping(m.ChannelID)

	logger.DebugCF("discord", "Received message", map[string]any{
		"sender_name": senderName,
		"sender_id":   senderID,
		"preview":     utils.Truncate(content, 50),
	})

	peerKind := "channel"
	peerID := m.ChannelID
	if m.GuildID == "" {
		peerKind = "direct"
		peerID = senderID
	}

	peer := bus.Peer{Kind: peerKind, ID: peerID}

	metadata := map[string]string{
		"user_id":      senderID,
		"username":     m.Author.Username,
		"display_name": senderName,
		"guild_id":     m.GuildID,
		"channel_id":   m.ChannelID,
		"is_dm":        fmt.Sprintf("%t", m.GuildID == ""),
	}

	c.HandleMessage(peer, m.ID, senderID, m.ChannelID, content, mediaPaths, metadata)
}

// startTyping starts a continuous typing indicator loop for the given chatID.
// It stops any existing typing loop for that chatID before starting a new one.
func (c *DiscordChannel) startTyping(chatID string) {
	c.typingMu.Lock()
	// Stop existing loop for this chatID if any
	if stop, ok := c.typingStop[chatID]; ok {
		close(stop)
	}
	stop := make(chan struct{})
	c.typingStop[chatID] = stop
	c.typingMu.Unlock()

	go func() {
		if err := c.session.ChannelTyping(chatID); err != nil {
			logger.DebugCF("discord", "ChannelTyping error", map[string]any{"chatID": chatID, "err": err})
		}
		ticker := time.NewTicker(8 * time.Second)
		defer ticker.Stop()
		timeout := time.After(5 * time.Minute)
		for {
			select {
			case <-stop:
				return
			case <-timeout:
				return
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				if err := c.session.ChannelTyping(chatID); err != nil {
					logger.DebugCF("discord", "ChannelTyping error", map[string]any{"chatID": chatID, "err": err})
				}
			}
		}
	}()
}

// stopTyping stops the typing indicator loop for the given chatID.
func (c *DiscordChannel) stopTyping(chatID string) {
	c.typingMu.Lock()
	defer c.typingMu.Unlock()
	if stop, ok := c.typingStop[chatID]; ok {
		close(stop)
		delete(c.typingStop, chatID)
	}
}

func (c *DiscordChannel) downloadAttachment(url, filename string) string {
	return utils.DownloadFile(url, filename, utils.DownloadOptions{
		LoggerPrefix: "discord",
	})
}

// stripBotMention removes the bot mention from the message content.
// Discord mentions have the format <@USER_ID> or <@!USER_ID> (with nickname).
func (c *DiscordChannel) stripBotMention(text string) string {
	if c.botUserID == "" {
		return text
	}
	// Remove both regular mention <@USER_ID> and nickname mention <@!USER_ID>
	text = strings.ReplaceAll(text, fmt.Sprintf("<@%s>", c.botUserID), "")
	text = strings.ReplaceAll(text, fmt.Sprintf("<@!%s>", c.botUserID), "")
	return strings.TrimSpace(text)
}
