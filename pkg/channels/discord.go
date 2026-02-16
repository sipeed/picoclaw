package channels

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
	"github.com/sipeed/picoclaw/pkg/voice"
)

const (
	transcriptionTimeout = 30 * time.Second
	sendTimeout          = 10 * time.Second
)

type DiscordChannel struct {
	*BaseChannel
	session     *discordgo.Session
	config      config.DiscordConfig
	transcriber *voice.GroqTranscriber
	ctx         context.Context
}

func NewDiscordChannel(cfg config.DiscordConfig, bus *bus.MessageBus) (*DiscordChannel, error) {
	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord session: %w", err)
	}

	base := NewBaseChannel("discord", cfg, bus, cfg.AllowFrom)

	return &DiscordChannel{
		BaseChannel: base,
		session:     session,
		config:      cfg,
		transcriber: nil,
		ctx:         context.Background(),
	}, nil
}

func (c *DiscordChannel) SetTranscriber(transcriber *voice.GroqTranscriber) {
	c.transcriber = transcriber
}

func (c *DiscordChannel) getContext() context.Context {
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

func (c *DiscordChannel) Start(ctx context.Context) error {
	logger.InfoC("discord", "Starting Discord bot")

	c.ctx = ctx
	c.session.AddHandler(c.handleMessage)

	if err := c.session.Open(); err != nil {
		return fmt.Errorf("failed to open discord session: %w", err)
	}

	c.setRunning(true)

	botUser, err := c.session.User("@me")
	if err != nil {
		return fmt.Errorf("failed to get bot user: %w", err)
	}
	logger.InfoCF("discord", "Discord bot connected", map[string]any{
		"username": botUser.Username,
		"user_id":  botUser.ID,
	})

	return nil
}

func (c *DiscordChannel) Stop(ctx context.Context) error {
	logger.InfoC("discord", "Stopping Discord bot")
	c.setRunning(false)

	if err := c.session.Close(); err != nil {
		return fmt.Errorf("failed to close discord session: %w", err)
	}

	return nil
}

func (c *DiscordChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("discord bot not running")
	}

	channelID := msg.ChatID
	if channelID == "" {
		return fmt.Errorf("channel ID is empty")
	}

	runes := []rune(msg.Content)
	if len(runes) == 0 {
		return nil
	}

	chunks := splitMessage(msg.Content, 1500) // Discord has a limit of 2000 characters per message, leave 500 for natural split e.g. code blocks

	for _, chunk := range chunks {
		if err := c.sendChunk(ctx, channelID, chunk); err != nil {
			return err
		}
	}

	return nil
}

// splitMessage splits long messages into chunks, preserving code block integrity.
// All length calculations use rune count (characters) since Discord's 2000-char
// limit is character-based, not byte-based.
func splitMessage(content string, limit int) []string {
	var messages []string
	runes := []rune(content)

	for len(runes) > 0 {
		if len(runes) <= limit {
			messages = append(messages, string(runes))
			break
		}

		msgEnd := limit

		// Find natural split point within the limit
		msgEnd = findLastRuneNewline(runes[:limit], 200)
		if msgEnd <= 0 {
			msgEnd = findLastRuneSpace(runes[:limit], 100)
		}
		if msgEnd <= 0 {
			msgEnd = limit
		}

		// Check if this would end with an incomplete code block
		unclosedRuneIdx := findLastUnclosedCodeBlockRune(runes[:msgEnd])

		if unclosedRuneIdx >= 0 {
			// Message would end with incomplete code block
			// Try to extend to include the closing ``` (with some buffer)
			extendedLimit := limit + 400
			if len(runes) > extendedLimit {
				closingRuneIdx := findNextClosingCodeBlockRune(runes, msgEnd)
				if closingRuneIdx > 0 && closingRuneIdx <= extendedLimit {
					msgEnd = closingRuneIdx
				} else {
					// Can't find closing, split before the code block
					msgEnd = findLastRuneNewline(runes[:unclosedRuneIdx], 200)
					if msgEnd <= 0 {
						msgEnd = findLastRuneSpace(runes[:unclosedRuneIdx], 100)
					}
					if msgEnd <= 0 {
						msgEnd = unclosedRuneIdx
					}
				}
			} else {
				// Remaining content fits within extended limit
				msgEnd = len(runes)
			}
		}

		if msgEnd <= 0 {
			msgEnd = limit
		}

		messages = append(messages, string(runes[:msgEnd]))
		remaining := strings.TrimSpace(string(runes[msgEnd:]))
		runes = []rune(remaining)
	}

	return messages
}

// findLastUnclosedCodeBlockRune finds the last opening ``` that doesn't have a closing ```
// using rune-based indexing. Returns the rune position or -1 if all code blocks are complete.
func findLastUnclosedCodeBlockRune(runes []rune) int {
	count := 0
	lastOpenIdx := -1

	for i := 0; i < len(runes); i++ {
		if i+2 < len(runes) && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
			if count%2 == 0 {
				lastOpenIdx = i
			}
			count++
			i += 2
		}
	}

	if count%2 == 1 {
		return lastOpenIdx
	}
	return -1
}

// findNextClosingCodeBlockRune finds the next closing ``` starting from a rune position.
// Returns the rune position after the closing ``` or -1 if not found.
func findNextClosingCodeBlockRune(runes []rune, startIdx int) int {
	for i := startIdx; i < len(runes); i++ {
		if i+2 < len(runes) && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
			// Include any trailing newline after the closing ```
			end := i + 3
			if end < len(runes) && runes[end] == '\n' {
				end++
			}
			return end
		}
	}
	return -1
}

// findLastRuneNewline finds the last newline within the last N runes.
func findLastRuneNewline(runes []rune, searchWindow int) int {
	searchStart := len(runes) - searchWindow
	if searchStart < 0 {
		searchStart = 0
	}
	for i := len(runes) - 1; i >= searchStart; i-- {
		if runes[i] == '\n' {
			return i
		}
	}
	return -1
}

// findLastRuneSpace finds the last space within the last N runes.
func findLastRuneSpace(runes []rune, searchWindow int) int {
	searchStart := len(runes) - searchWindow
	if searchStart < 0 {
		searchStart = 0
	}
	for i := len(runes) - 1; i >= searchStart; i-- {
		if runes[i] == ' ' || runes[i] == '\t' {
			return i
		}
	}
	return -1
}

func (c *DiscordChannel) sendChunk(ctx context.Context, channelID, content string) error {
	// 使用传入的 ctx 进行超时控制
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
			return fmt.Errorf("failed to send discord message: %w", err)
		}
		return nil
	case <-sendCtx.Done():
		return fmt.Errorf("send message timeout: %w", sendCtx.Err())
	}
}

// appendContent 安全地追加内容到现有文本
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

	if err := c.session.ChannelTyping(m.ChannelID); err != nil {
		logger.ErrorCF("discord", "Failed to send typing indicator", map[string]any{
			"error": err.Error(),
		})
	}

	// 检查白名单，避免为被拒绝的用户下载附件和转录
	if !c.IsAllowed(m.Author.ID) {
		logger.DebugCF("discord", "Message rejected by allowlist", map[string]any{
			"user_id": m.Author.ID,
		})
		return
	}

	senderID := m.Author.ID
	senderName := m.Author.Username
	if m.Author.Discriminator != "" && m.Author.Discriminator != "0" {
		senderName += "#" + m.Author.Discriminator
	}

	content := m.Content
	mediaPaths := make([]string, 0, len(m.Attachments))
	localFiles := make([]string, 0, len(m.Attachments))

	// 确保临时文件在函数返回时被清理
	defer func() {
		for _, file := range localFiles {
			if err := os.Remove(file); err != nil {
				logger.DebugCF("discord", "Failed to cleanup temp file", map[string]any{
					"file":  file,
					"error": err.Error(),
				})
			}
		}
	}()

	for _, attachment := range m.Attachments {
		isAudio := utils.IsAudioFile(attachment.Filename, attachment.ContentType)

		if isAudio {
			localPath := c.downloadAttachment(attachment.URL, attachment.Filename)
			if localPath != "" {
				localFiles = append(localFiles, localPath)

				transcribedText := ""
				if c.transcriber != nil && c.transcriber.IsAvailable() {
					ctx, cancel := context.WithTimeout(c.getContext(), transcriptionTimeout)
					result, err := c.transcriber.Transcribe(ctx, localPath)
					cancel() // 立即释放context资源，避免在for循环中泄漏

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

	logger.DebugCF("discord", "Received message", map[string]any{
		"sender_name": senderName,
		"sender_id":   senderID,
		"preview":     utils.Truncate(content, 50),
	})

	metadata := map[string]string{
		"message_id":   m.ID,
		"user_id":      senderID,
		"username":     m.Author.Username,
		"display_name": senderName,
		"guild_id":     m.GuildID,
		"channel_id":   m.ChannelID,
		"is_dm":        fmt.Sprintf("%t", m.GuildID == ""),
	}

	c.HandleMessage(senderID, m.ChannelID, content, mediaPaths, metadata)
}

func (c *DiscordChannel) downloadAttachment(url, filename string) string {
	return utils.DownloadFile(url, filename, utils.DownloadOptions{
		LoggerPrefix: "discord",
	})
}
