package channels

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	th "github.com/mymmrac/telego/telegohandler"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
	"github.com/sipeed/picoclaw/pkg/voice"
)

type TelegramChannel struct {
	*BaseChannel
	bot          *telego.Bot
	commands     TelegramCommander
	config       *config.Config
	chatIDs      map[string]int64
	transcriber  *voice.GroqTranscriber
	typingTasks  sync.Map // chatID -> *typingTask
}

type typingTask struct {
	fn context.CancelFunc
}

func (c *typingTask) Cancel() {
	if c != nil && c.fn != nil {
		c.fn()
	}
}

func NewTelegramChannel(cfg *config.Config, bus *bus.MessageBus) (*TelegramChannel, error) {
	var opts []telego.BotOption
	telegramCfg := cfg.Channels.Telegram

	if telegramCfg.Proxy != "" {
		proxyURL, parseErr := url.Parse(telegramCfg.Proxy)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", telegramCfg.Proxy, parseErr)
		}
		opts = append(opts, telego.WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}))
	}

	bot, err := telego.NewBot(telegramCfg.Token, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	base := NewBaseChannel("telegram", telegramCfg, bus, telegramCfg.AllowFrom)

	return &TelegramChannel{
		BaseChannel:  base,
		commands:     NewTelegramCommands(bot, cfg),
		bot:          bot,
		config:       cfg,
		chatIDs:      make(map[string]int64),
		transcriber:  nil,
		typingTasks:  sync.Map{},
	}, nil
}

func (c *TelegramChannel) SetTranscriber(transcriber *voice.GroqTranscriber) {
	c.transcriber = transcriber
}

func (c *TelegramChannel) Start(ctx context.Context) error {
	logger.InfoC("telegram", "Starting Telegram bot (polling mode)...")

	updates, err := c.bot.UpdatesViaLongPolling(ctx, &telego.GetUpdatesParams{
		Timeout: 30,
	})
	if err != nil {
		return fmt.Errorf("failed to start long polling: %w", err)
	}

	bh, err := telegohandler.NewBotHandler(c.bot, updates)
	if err != nil {
		return fmt.Errorf("failed to create bot handler: %w", err)
	}

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		c.commands.Help(ctx, message)
		return nil
	}, th.CommandEqual("help"))
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		return c.commands.Start(ctx, message)
	}, th.CommandEqual("start"))

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		return c.commands.Show(ctx, message)
	}, th.CommandEqual("show"))

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		return c.commands.List(ctx, message)
	}, th.CommandEqual("list"))

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		return c.handleMessage(ctx, &message)
	}, th.AnyMessage())

	c.setRunning(true)
	logger.InfoCF("telegram", "Telegram bot connected", map[string]interface{}{
		"username": c.bot.Username(),
	})

	go bh.Start()

	go func() {
		<-ctx.Done()
		bh.Stop()
	}()

	return nil
}

func (c *TelegramChannel) Stop(ctx context.Context) error {
	logger.InfoC("telegram", "Stopping Telegram bot...")
	c.setRunning(false)
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

	// Stop typing indicator goroutine
	if stop, ok := c.typingTasks.Load(msg.ChatID); ok {
		if cf, ok := stop.(*typingTask); ok && cf != nil {
			cf.Cancel()
		}
		c.typingTasks.Delete(msg.ChatID)
	}

	htmlContent := markdownToTelegramHTML(msg.Content)

	tgMsg := tu.Message(tu.ID(chatID), htmlContent)
	tgMsg.ParseMode = telego.ModeHTML

	if _, err = c.bot.SendMessage(ctx, tgMsg); err != nil {
		logger.ErrorCF("telegram", "HTML parse failed, falling back to plain text", map[string]interface{}{
			"error": err.Error(),
		})
		tgMsg.ParseMode = ""
		_, err = c.bot.SendMessage(ctx, tgMsg)
		return err
	}

	return nil
}

func (c *TelegramChannel) handleMessage(ctx context.Context, message *telego.Message) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}

	user := message.From
	if user == nil {
		return fmt.Errorf("message sender (user) is nil")
	}

	senderID := fmt.Sprintf("%d", user.ID)
	if user.Username != "" {
		senderID = fmt.Sprintf("%d|%s", user.ID, user.Username)
	}

	// 检查白名单，避免为被拒绝的用户下载附件
	if !c.IsAllowed(senderID) {
		logger.DebugCF("telegram", "Message rejected by allowlist", map[string]interface{}{
			"user_id": senderID,
		})
		return nil
	}

	chatID := message.Chat.ID
	c.chatIDs[senderID] = chatID

	content := ""
	mediaPaths := []string{}
	localFiles := []string{} // 跟踪需要清理的本地文件

	// 确保临时文件在函数返回时被清理
	defer func() {
		for _, file := range localFiles {
			if err := os.Remove(file); err != nil {
				logger.DebugCF("telegram", "Failed to cleanup temp file", map[string]interface{}{
					"file":  file,
					"error": err.Error(),
				})
			}
		}
	}()

	if message.Text != "" {
		content += message.Text
	}

	if message.Caption != "" {
		if content != "" {
			content += "\n"
		}
		content += message.Caption
	}

	if len(message.Photo) > 0 {
		photo := message.Photo[len(message.Photo)-1]
		photoPath := c.downloadPhoto(ctx, photo.FileID)
		if photoPath != "" {
			localFiles = append(localFiles, photoPath)
			mediaPaths = append(mediaPaths, photoPath)
			if content != "" {
				content += "\n"
			}
			content += "[image: photo]"
		}
	}

	if message.Voice != nil {
		voicePath := c.downloadFile(ctx, message.Voice.FileID, ".ogg")
		if voicePath != "" {
			localFiles = append(localFiles, voicePath)
			mediaPaths = append(mediaPaths, voicePath)

			transcribedText := ""
			if c.transcriber != nil && c.transcriber.IsAvailable() {
				ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()

				result, err := c.transcriber.Transcribe(ctx, voicePath)
				if err != nil {
					logger.ErrorCF("telegram", "Voice transcription failed", map[string]interface{}{
						"error": err.Error(),
						"path":  voicePath,
					})
					transcribedText = "[voice (transcription failed)]"
				} else {
					transcribedText = fmt.Sprintf("[voice transcription: %s]", result.Text)
					logger.InfoCF("telegram", "Voice transcribed successfully", map[string]interface{}{
						"text": result.Text,
					})
				}
			} else {
				transcribedText = "[voice]"
			}

			if content != "" {
				content += "\n"
			}
			content += transcribedText
		}
	}

	if message.Audio != nil {
		audioPath := c.downloadFile(ctx, message.Audio.FileID, ".mp3")
		if audioPath != "" {
			localFiles = append(localFiles, audioPath)
			mediaPaths = append(mediaPaths, audioPath)
			if content != "" {
				content += "\n"
			}
			content += "[audio]"
		}
	}

	if message.Document != nil {
		docPath := c.downloadFile(ctx, message.Document.FileID, "")
		if docPath != "" {
			localFiles = append(localFiles, docPath)
			mediaPaths = append(mediaPaths, docPath)
			if content != "" {
				content += "\n"
			}
			content += "[file]"
		}
	}

	if content == "" {
		content = "[empty message]"
	}

	logger.DebugCF("telegram", "Received message", map[string]interface{}{
		"sender_id": senderID,
		"chat_id":   fmt.Sprintf("%d", chatID),
		"preview":   utils.Truncate(content, 50),
	})

	// Start repeating typing indicator (expires after 5s per Telegram API,
	// so we re-send every 4s until the response arrives)
	chatIDStr := fmt.Sprintf("%d", chatID)
	// Cancel any previous typing goroutine for this chat
	if prevStop, ok := c.typingTasks.Load(chatIDStr); ok {
		if cf, ok := prevStop.(*typingTask); ok && cf != nil {
			cf.Cancel()
		}
	}
	typingCtx, typingCancel := context.WithCancel(ctx)
	c.typingTasks.Store(chatIDStr, &typingTask{fn: typingCancel})
	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		// Send immediately first
		_ = c.bot.SendChatAction(typingCtx, tu.ChatAction(tu.ID(chatID), telego.ChatActionTyping))
		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				_ = c.bot.SendChatAction(typingCtx, tu.ChatAction(tu.ID(chatID), telego.ChatActionTyping))
			}
		}
	}()

	metadata := map[string]string{
		"message_id": fmt.Sprintf("%d", message.MessageID),
		"user_id":    fmt.Sprintf("%d", user.ID),
		"username":   user.Username,
		"first_name": user.FirstName,
		"is_group":   fmt.Sprintf("%t", message.Chat.Type != "private"),
	}

	c.HandleMessage(fmt.Sprintf("%d", user.ID), fmt.Sprintf("%d", chatID), content, mediaPaths, metadata)
	return nil
}

func (c *TelegramChannel) downloadPhoto(ctx context.Context, fileID string) string {
	file, err := c.bot.GetFile(ctx, &telego.GetFileParams{FileID: fileID})
	if err != nil {
		logger.ErrorCF("telegram", "Failed to get photo file", map[string]interface{}{
			"error": err.Error(),
		})
		return ""
	}

	return c.downloadFileWithInfo(file, ".jpg")
}

func (c *TelegramChannel) downloadFileWithInfo(file *telego.File, ext string) string {
	if file.FilePath == "" {
		return ""
	}

	url := c.bot.FileDownloadURL(file.FilePath)
	logger.DebugCF("telegram", "File URL", map[string]interface{}{"url": url})

	// Use FilePath as filename for better identification
	filename := file.FilePath + ext
	return utils.DownloadFile(url, filename, utils.DownloadOptions{
		LoggerPrefix: "telegram",
	})
}

func (c *TelegramChannel) downloadFile(ctx context.Context, fileID, ext string) string {
	file, err := c.bot.GetFile(ctx, &telego.GetFileParams{FileID: fileID})
	if err != nil {
		logger.ErrorCF("telegram", "Failed to get file", map[string]interface{}{
			"error": err.Error(),
		})
		return ""
	}

	return c.downloadFileWithInfo(file, ext)
}

func parseChatID(chatIDStr string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(chatIDStr, "%d", &id)
	return id, err
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

	i := 0
	text = re.ReplaceAllStringFunc(text, func(m string) string {
		placeholder := fmt.Sprintf("\x00CB%d\x00", i)
		i++
		return placeholder
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

	i := 0
	text = re.ReplaceAllStringFunc(text, func(m string) string {
		placeholder := fmt.Sprintf("\x00IC%d\x00", i)
		i++
		return placeholder
	})

	return inlineCodeMatch{text: text, codes: codes}
}

func escapeHTML(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}
