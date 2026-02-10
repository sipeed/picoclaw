package channels

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/voice"
)

type TelegramChannel struct {
	*BaseChannel
	bot          *tgbotapi.BotAPI
	config       config.TelegramConfig
	chatIDs      map[string]int64
	updates      tgbotapi.UpdatesChannel
	transcriber  *voice.GroqTranscriber
	placeholders sync.Map // chatID -> messageID
	stopThinking sync.Map // chatID -> chan struct{}
}

func NewTelegramChannel(cfg config.TelegramConfig, bus *bus.MessageBus) (*TelegramChannel, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	base := NewBaseChannel("telegram", cfg, bus, cfg.AllowFrom)

	return &TelegramChannel{
		BaseChannel:  base,
		bot:          bot,
		config:       cfg,
		chatIDs:      make(map[string]int64),
		transcriber:  nil,
		placeholders: sync.Map{},
		stopThinking: sync.Map{},
	}, nil
}

func (c *TelegramChannel) SetTranscriber(transcriber *voice.GroqTranscriber) {
	c.transcriber = transcriber
}

func (c *TelegramChannel) Start(ctx context.Context) error {
	log.Printf("Starting Telegram bot (polling mode)...")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := c.bot.GetUpdatesChan(u)
	c.updates = updates

	c.setRunning(true)

	botInfo, err := c.bot.GetMe()
	if err != nil {
		return fmt.Errorf("failed to get bot info: %w", err)
	}
	log.Printf("Telegram bot @%s connected", botInfo.UserName)

	// Start stream event handler
	go c.handleStreamEvents(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case update, ok := <-updates:
				if !ok {
					log.Printf("Updates channel closed, reconnecting...")
					return
				}
				if update.Message != nil {
					c.handleMessage(update)
				}
			}
		}
	}()

	return nil
}

func (c *TelegramChannel) Stop(ctx context.Context) error {
	log.Println("Stopping Telegram bot...")
	c.setRunning(false)

	if c.updates != nil {
		c.bot.StopReceivingUpdates()
		c.updates = nil
	}

	return nil
}

func (c *TelegramChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("telegram bot not running")
	}

	chatID, err := parseChatID(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	// Stop thinking animation
	if stop, ok := c.stopThinking.Load(msg.ChatID); ok {
		close(stop.(chan struct{}))
		c.stopThinking.Delete(msg.ChatID)
	}

	htmlContent := markdownToTelegramHTML(msg.Content)

	// Try to edit placeholder
	if pID, ok := c.placeholders.Load(msg.ChatID); ok {
		c.placeholders.Delete(msg.ChatID)
		editMsg := tgbotapi.NewEditMessageText(chatID, pID.(int), htmlContent)
		editMsg.ParseMode = tgbotapi.ModeHTML

		if _, err := c.bot.Send(editMsg); err == nil {
			return nil
		}
		// Fallback to new message if edit fails
	}

	tgMsg := tgbotapi.NewMessage(chatID, htmlContent)
	tgMsg.ParseMode = tgbotapi.ModeHTML

	if _, err := c.bot.Send(tgMsg); err != nil {
		log.Printf("HTML parse failed, falling back to plain text: %v", err)
		tgMsg = tgbotapi.NewMessage(chatID, msg.Content)
		tgMsg.ParseMode = ""
		_, err = c.bot.Send(tgMsg)
		return err
	}

	return nil
}

func (c *TelegramChannel) handleMessage(update tgbotapi.Update) {
	message := update.Message
	if message == nil {
		return
	}

	user := message.From
	if user == nil {
		return
	}

	senderID := fmt.Sprintf("%d", user.ID)
	if user.UserName != "" {
		senderID = fmt.Sprintf("%d|%s", user.ID, user.UserName)
	}

	chatID := message.Chat.ID
	c.chatIDs[senderID] = chatID

	content := ""
	mediaPaths := []string{}

	if message.Text != "" {
		content += message.Text
	}

	if message.Caption != "" {
		if content != "" {
			content += "\n"
		}
		content += message.Caption
	}

	if message.Photo != nil && len(message.Photo) > 0 {
		photo := message.Photo[len(message.Photo)-1]
		photoPath := c.downloadPhoto(photo.FileID)
		if photoPath != "" {
			mediaPaths = append(mediaPaths, photoPath)
			if content != "" {
				content += "\n"
			}
			content += fmt.Sprintf("[image: %s]", photoPath)
		}
	}

	if message.Voice != nil {
		voicePath := c.downloadFile(message.Voice.FileID, ".ogg")
		if voicePath != "" {
			mediaPaths = append(mediaPaths, voicePath)

			transcribedText := ""
			if c.transcriber != nil && c.transcriber.IsAvailable() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				result, err := c.transcriber.Transcribe(ctx, voicePath)
				if err != nil {
					log.Printf("Voice transcription failed: %v", err)
					transcribedText = fmt.Sprintf("[voice: %s (transcription failed)]", voicePath)
				} else {
					transcribedText = fmt.Sprintf("[voice transcription: %s]", result.Text)
					log.Printf("Voice transcribed successfully: %s", result.Text)
				}
			} else {
				transcribedText = fmt.Sprintf("[voice: %s]", voicePath)
			}

			if content != "" {
				content += "\n"
			}
			content += transcribedText
		}
	}

	if message.Audio != nil {
		audioPath := c.downloadFile(message.Audio.FileID, ".mp3")
		if audioPath != "" {
			mediaPaths = append(mediaPaths, audioPath)
			if content != "" {
				content += "\n"
			}
			content += fmt.Sprintf("[audio: %s]", audioPath)
		}
	}

	if message.Document != nil {
		docPath := c.downloadFile(message.Document.FileID, "")
		if docPath != "" {
			mediaPaths = append(mediaPaths, docPath)
			if content != "" {
				content += "\n"
			}
			content += fmt.Sprintf("[file: %s]", docPath)
		}
	}

	if content == "" {
		content = "[empty message]"
	}

	log.Printf("Telegram message from %s: %s...", senderID, truncateString(content, 50))

	metadata := map[string]string{
		"message_id": fmt.Sprintf("%d", message.MessageID),
		"user_id":    fmt.Sprintf("%d", user.ID),
		"username":   user.UserName,
		"first_name": user.FirstName,
		"is_group":   fmt.Sprintf("%t", message.Chat.Type != "private"),
	}

	// Create session key
	sessionKey := fmt.Sprintf("telegram:%d", chatID)

	// Send inbound message with session key
	msg := bus.InboundMessage{
		Channel:    c.Name(),
		SenderID:   senderID,
		ChatID:     fmt.Sprintf("%d", chatID),
		Content:    content,
		Media:      mediaPaths,
		Metadata:   metadata,
		SessionKey: sessionKey,
	}

	c.BaseChannel.bus.PublishInbound(msg)
}

func (c *TelegramChannel) downloadPhoto(fileID string) string {
	file, err := c.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		log.Printf("Failed to get photo file: %v", err)
		return ""
	}

	return c.downloadFileWithInfo(&file, ".jpg")
}

func (c *TelegramChannel) downloadFileWithInfo(file *tgbotapi.File, ext string) string {
	if file.FilePath == "" {
		return ""
	}

	url := file.Link(c.bot.Token)
	log.Printf("File URL: %s", url)

	mediaDir := "/tmp/picoclaw_media"

	return fmt.Sprintf("%s/%s%s", mediaDir, file.FilePath[:min(16, len(file.FilePath))], ext)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *TelegramChannel) downloadFile(fileID, ext string) string {
	file, err := c.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		log.Printf("Failed to get file: %v", err)
		return ""
	}

	if file.FilePath == "" {
		return ""
	}

	url := file.Link(c.bot.Token)
	log.Printf("File URL: %s", url)

	mediaDir := "/tmp/picoclaw_media"

	return fmt.Sprintf("%s/%s%s", mediaDir, fileID[:16], ext)
}

func parseChatID(chatIDStr string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(chatIDStr, "%d", &id)
	return id, err
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func markdownToTelegramHTML(text string) string {
	if text == "" {
		return ""
	}

	codeBlocks := extractCodeBlocks(text)
	text = codeBlocks.text

	inlineCodes := extractInlineCodes(text)
	text = inlineCodes.text

	text = regexp.MustCompile(`^#{1,6}\s+(.+)$`).ReplaceAllString(text, "$1")

	text = regexp.MustCompile(`^>\s*(.*)$`).ReplaceAllString(text, "$1")

	text = escapeHTML(text)

	text = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`).ReplaceAllString(text, `<a href="$2">$1</a>`)

	text = regexp.MustCompile(`\*\*(.+?)\*\*`).ReplaceAllString(text, "<b>$1</b>")

	text = regexp.MustCompile(`__(.+?)__`).ReplaceAllString(text, "<b>$1</b>")

	reItalic := regexp.MustCompile(`_([^_]+)_`)
	text = reItalic.ReplaceAllStringFunc(text, func(s string) string {
		match := reItalic.FindStringSubmatch(s)
		if len(match) < 2 {
			return s
		}
		return "<i>" + match[1] + "</i>"
	})

	text = regexp.MustCompile(`~~(.+?)~~`).ReplaceAllString(text, "<s>$1</s>")

	text = regexp.MustCompile(`^[-*]\s+`).ReplaceAllString(text, "• ")

	for i, code := range inlineCodes.codes {
		escaped := escapeHTML(code)
		text = strings.ReplaceAll(text, fmt.Sprintf("\x00IC%d\x00", i), fmt.Sprintf("<code>%s</code>", escaped))
	}

	for i, code := range codeBlocks.codes {
		escaped := escapeHTML(code)
		text = strings.ReplaceAll(text, fmt.Sprintf("\x00CB%d\x00", i), fmt.Sprintf("<pre><code>%s</code></pre>", escaped))
	}

	return text
}

type codeBlockMatch struct {
	text  string
	codes []string
}

func extractCodeBlocks(text string) codeBlockMatch {
	re := regexp.MustCompile("```[\\w]*\\n?([\\s\\S]*?)```")
	matches := re.FindAllStringSubmatch(text, -1)

	codes := make([]string, 0, len(matches))
	for _, match := range matches {
		codes = append(codes, match[1])
	}

	text = re.ReplaceAllStringFunc(text, func(m string) string {
		return fmt.Sprintf("\x00CB%d\x00", len(codes)-1)
	})

	return codeBlockMatch{text: text, codes: codes}
}

type inlineCodeMatch struct {
	text  string
	codes []string
}

func extractInlineCodes(text string) inlineCodeMatch {
	re := regexp.MustCompile("`([^`]+)`")
	matches := re.FindAllStringSubmatch(text, -1)

	codes := make([]string, 0, len(matches))
	for _, match := range matches {
		codes = append(codes, match[1])
	}

	text = re.ReplaceAllStringFunc(text, func(m string) string {
		return fmt.Sprintf("\x00IC%d\x00", len(codes)-1)
	})

	return inlineCodeMatch{text: text, codes: codes}
}

func escapeHTML(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}

// handleStreamEvents listens for streaming events and updates messages in real-time
func (c *TelegramChannel) handleStreamEvents(ctx context.Context) {
	// Track current status message for each chat
	statusMessages := sync.Map{} // chatID -> messageID
	lastUpdate := sync.Map{}     // chatID -> time.Time

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		event, ok := c.BaseChannel.bus.ConsumeStreamEvent(ctx)
		if !ok {
			continue
		}

		// Only handle events for telegram channel
		if event.Channel != "telegram" {
			continue
		}

		chatID, err := parseChatID(event.ChatID)
		if err != nil {
			continue
		}

		// Rate limiting: update at most once per second
		if lastTime, ok := lastUpdate.Load(event.ChatID); ok {
			if time.Since(lastTime.(time.Time)) < time.Second {
				continue
			}
		}
		lastUpdate.Store(event.ChatID, time.Now())

		var statusText string
		var shouldDelete bool

		switch event.Type {
		case bus.StreamEventThinking:
			statusText = "💭 " + event.Content
		case bus.StreamEventProgress:
			statusText = "⏳ " + event.Content
		case bus.StreamEventToolCall:
			if toolName, ok := event.Metadata["tool_name"].(string); ok {
				statusText = fmt.Sprintf("🔧 Executing tool: %s", toolName)
			} else {
				statusText = "🔧 " + event.Content
			}
		case bus.StreamEventToolResult:
			if toolName, ok := event.Metadata["tool_name"].(string); ok {
				statusText = fmt.Sprintf("✓ Tool %s completed", toolName)
			} else {
				statusText = "✓ " + event.Content
			}
		case bus.StreamEventComplete, bus.StreamEventContent:
			// Delete status message on completion
			shouldDelete = true
		case bus.StreamEventError:
			statusText = "❌ " + event.Content
		}

		if shouldDelete {
			// Delete the status message
			if msgID, ok := statusMessages.LoadAndDelete(event.ChatID); ok {
				deleteMsg := tgbotapi.NewDeleteMessage(chatID, msgID.(int))
				c.bot.Send(deleteMsg)
			}
			continue
		}

		if statusText == "" {
			continue
		}

		// Update or create status message
		if msgID, ok := statusMessages.Load(event.ChatID); ok {
			// Update existing message
			editMsg := tgbotapi.NewEditMessageText(chatID, msgID.(int), statusText)
			if _, err := c.bot.Send(editMsg); err != nil {
				// If edit fails, delete and create new
				deleteMsg := tgbotapi.NewDeleteMessage(chatID, msgID.(int))
				c.bot.Send(deleteMsg)
				statusMessages.Delete(event.ChatID)
			}
		}

		// Create new status message if none exists
		if _, ok := statusMessages.Load(event.ChatID); !ok {
			newMsg, err := c.bot.Send(tgbotapi.NewMessage(chatID, statusText))
			if err == nil {
				statusMessages.Store(event.ChatID, newMsg.MessageID)
			}
		}
	}
}
