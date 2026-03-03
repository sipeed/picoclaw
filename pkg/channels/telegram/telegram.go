package telegram

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
)

var reHeading = regexp.MustCompile(`(?m)^#{1,6}\s+([^\n]+)`)

type TelegramChannel struct {
	*channels.BaseChannel
	bot      *telego.Bot
	bh       *telegohandler.BotHandler
	commands TelegramCommander
	config   *config.Config
	chatIDs  map[string]int64
	ctx      context.Context
	cancel   context.CancelFunc
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
	} else if os.Getenv("HTTP_PROXY") != "" || os.Getenv("HTTPS_PROXY") != "" {
		// Use environment proxy if configured
		opts = append(opts, telego.WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		}))
	}

	bot, err := telego.NewBot(telegramCfg.Token, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	base := channels.NewBaseChannel(
		"telegram",
		telegramCfg,
		bus,
		telegramCfg.AllowFrom,
		channels.WithMaxMessageLength(4096),
		channels.WithGroupTrigger(telegramCfg.GroupTrigger),
		channels.WithReasoningChannelID(telegramCfg.ReasoningChannelID),
	)

	return &TelegramChannel{
		BaseChannel: base,
		commands:    NewTelegramCommands(bot, cfg),
		bot:         bot,
		config:      cfg,
		chatIDs:     make(map[string]int64),
	}, nil
}

func (c *TelegramChannel) Start(ctx context.Context) error {
	logger.InfoC("telegram", "Starting Telegram bot (polling mode)...")

	c.ctx, c.cancel = context.WithCancel(ctx)

	updates, err := c.bot.UpdatesViaLongPolling(c.ctx, &telego.GetUpdatesParams{
		Timeout: 30,
	})
	if err != nil {
		c.cancel()
		return fmt.Errorf("failed to start long polling: %w", err)
	}

	bh, err := telegohandler.NewBotHandler(c.bot, updates)
	if err != nil {
		c.cancel()
		return fmt.Errorf("failed to create bot handler: %w", err)
	}
	c.bh = bh

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

	c.SetRunning(true)
	logger.InfoCF("telegram", "Telegram bot connected", map[string]any{
		"username": c.bot.Username(),
	})

	go bh.Start()

	return nil
}

func (c *TelegramChannel) Stop(ctx context.Context) error {
	logger.InfoC("telegram", "Stopping Telegram bot...")
	c.SetRunning(false)

	// Stop the bot handler
	if c.bh != nil {
		c.bh.Stop()
	}

	// Cancel our context (stops long polling)
	if c.cancel != nil {
		c.cancel()
	}

	return nil
}

func (c *TelegramChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	chatID, err := parseChatID(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid chat ID %s: %w", msg.ChatID, channels.ErrSendFailed)
	}

	markdownV2Content := markdownToTelegramMarkdownV2(msg.Content)

	tgMsg := tu.Message(tu.ID(chatID), markdownV2Content).
		WithParseMode(telego.ModeMarkdownV2)

	if _, err = c.bot.SendMessage(ctx, tgMsg); err != nil {
		logger.ErrorCF("telegram", "MarkdownV2 parse failed, falling back to plain text", map[string]any{
			"error": err.Error(),
		})
		if _, err = c.bot.SendMessage(ctx, tu.Message(tu.ID(chatID), msg.Content)); err != nil {
			return fmt.Errorf("telegram send: %w", channels.ErrTemporary)
		}
	}

	return nil
}

// StartTyping implements channels.TypingCapable.
// It sends ChatAction(typing) immediately and then repeats every 4 seconds
// (Telegram's typing indicator expires after ~5s) in a background goroutine.
// The returned stop function is idempotent and cancels the goroutine.
func (c *TelegramChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	cid, err := parseChatID(chatID)
	if err != nil {
		return func() {}, err
	}

	// Send the first typing action immediately
	_ = c.bot.SendChatAction(ctx, tu.ChatAction(tu.ID(cid), telego.ChatActionTyping))

	typingCtx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				_ = c.bot.SendChatAction(typingCtx, tu.ChatAction(tu.ID(cid), telego.ChatActionTyping))
			}
		}
	}()

	return cancel, nil
}

// EditMessage implements channels.MessageEditor.
func (c *TelegramChannel) EditMessage(ctx context.Context, chatID string, messageID string, content string) error {
	cid, err := parseChatID(chatID)
	if err != nil {
		return err
	}
	mid, err := strconv.Atoi(messageID)
	if err != nil {
		return err
	}
	md2Content := markdownToTelegramMarkdownV2(content)
	editMsg := tu.EditMessageText(tu.ID(cid), mid, md2Content).
		WithParseMode(telego.ModeMarkdownV2)
	_, err = c.bot.EditMessageText(ctx, editMsg)
	return err
}

// SendPlaceholder implements channels.PlaceholderCapable.
// It sends a placeholder message (e.g. "Thinking... 💭") that will later be
// edited to the actual response via EditMessage (channels.MessageEditor).
func (c *TelegramChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	phCfg := c.config.Channels.Telegram.Placeholder
	if !phCfg.Enabled {
		return "", nil
	}

	text := phCfg.Text
	if text == "" {
		text = "Thinking... 💭"
	}

	cid, err := parseChatID(chatID)
	if err != nil {
		return "", err
	}

	pMsg, err := c.bot.SendMessage(ctx, tu.Message(tu.ID(cid), text))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d", pMsg.MessageID), nil
}

// SendMedia implements the channels.MediaSender interface.
func (c *TelegramChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	chatID, err := parseChatID(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid chat ID %s: %w", msg.ChatID, channels.ErrSendFailed)
	}

	store := c.GetMediaStore()
	if store == nil {
		return fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	for _, part := range msg.Parts {
		localPath, err := store.Resolve(part.Ref)
		if err != nil {
			logger.ErrorCF("telegram", "Failed to resolve media ref", map[string]any{
				"ref":   part.Ref,
				"error": err.Error(),
			})
			continue
		}

		file, err := os.Open(localPath)
		if err != nil {
			logger.ErrorCF("telegram", "Failed to open media file", map[string]any{
				"path":  localPath,
				"error": err.Error(),
			})
			continue
		}

		switch part.Type {
		case "image":
			params := &telego.SendPhotoParams{
				ChatID:  tu.ID(chatID),
				Photo:   telego.InputFile{File: file},
				Caption: part.Caption,
			}
			_, err = c.bot.SendPhoto(ctx, params)
		case "audio":
			params := &telego.SendAudioParams{
				ChatID:  tu.ID(chatID),
				Audio:   telego.InputFile{File: file},
				Caption: part.Caption,
			}
			_, err = c.bot.SendAudio(ctx, params)
		case "video":
			params := &telego.SendVideoParams{
				ChatID:  tu.ID(chatID),
				Video:   telego.InputFile{File: file},
				Caption: part.Caption,
			}
			_, err = c.bot.SendVideo(ctx, params)
		default: // "file" or unknown types
			params := &telego.SendDocumentParams{
				ChatID:   tu.ID(chatID),
				Document: telego.InputFile{File: file},
				Caption:  part.Caption,
			}
			_, err = c.bot.SendDocument(ctx, params)
		}

		file.Close()

		if err != nil {
			logger.ErrorCF("telegram", "Failed to send media", map[string]any{
				"type":  part.Type,
				"error": err.Error(),
			})
			return fmt.Errorf("telegram send media: %w", channels.ErrTemporary)
		}
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

	platformID := fmt.Sprintf("%d", user.ID)
	sender := bus.SenderInfo{
		Platform:    "telegram",
		PlatformID:  platformID,
		CanonicalID: identity.BuildCanonicalID("telegram", platformID),
		Username:    user.Username,
		DisplayName: user.FirstName,
	}

	// check allowlist to avoid downloading attachments for rejected users
	if !c.IsAllowedSender(sender) {
		logger.DebugCF("telegram", "Message rejected by allowlist", map[string]any{
			"user_id": platformID,
		})
		return nil
	}

	chatID := message.Chat.ID
	c.chatIDs[platformID] = chatID

	content := ""
	mediaPaths := []string{}

	chatIDStr := fmt.Sprintf("%d", chatID)
	messageIDStr := fmt.Sprintf("%d", message.MessageID)
	scope := channels.BuildMediaScope("telegram", chatIDStr, messageIDStr)

	// Helper to register a local file with the media store
	storeMedia := func(localPath, filename string) string {
		if store := c.GetMediaStore(); store != nil {
			ref, err := store.Store(localPath, media.MediaMeta{
				Filename: filename,
				Source:   "telegram",
			}, scope)
			if err == nil {
				return ref
			}
		}
		return localPath // fallback: use raw path
	}

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
			mediaPaths = append(mediaPaths, storeMedia(photoPath, "photo.jpg"))
			if content != "" {
				content += "\n"
			}
			content += "[image: photo]"
		}
	}

	if message.Voice != nil {
		voicePath := c.downloadFile(ctx, message.Voice.FileID, ".ogg")
		if voicePath != "" {
			mediaPaths = append(mediaPaths, storeMedia(voicePath, "voice.ogg"))

			if content != "" {
				content += "\n"
			}
			content += "[voice]"
		}
	}

	if message.Audio != nil {
		audioPath := c.downloadFile(ctx, message.Audio.FileID, ".mp3")
		if audioPath != "" {
			mediaPaths = append(mediaPaths, storeMedia(audioPath, "audio.mp3"))
			if content != "" {
				content += "\n"
			}
			content += "[audio]"
		}
	}

	if message.Document != nil {
		docPath := c.downloadFile(ctx, message.Document.FileID, "")
		if docPath != "" {
			mediaPaths = append(mediaPaths, storeMedia(docPath, "document"))
			if content != "" {
				content += "\n"
			}
			content += "[file]"
		}
	}

	if content == "" {
		content = "[empty message]"
	}

	// In group chats, apply unified group trigger filtering
	if message.Chat.Type != "private" {
		isMentioned := c.isBotMentioned(message)
		if isMentioned {
			content = c.stripBotMention(content)
		}
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, content)
		if !respond {
			return nil
		}
		content = cleaned
	}

	logger.DebugCF("telegram", "Received message", map[string]any{
		"sender_id": sender.CanonicalID,
		"chat_id":   fmt.Sprintf("%d", chatID),
		"preview":   utils.Truncate(content, 50),
	})

	// Placeholder is now auto-triggered by BaseChannel.HandleMessage via PlaceholderCapable

	peerKind := "direct"
	peerID := fmt.Sprintf("%d", user.ID)
	if message.Chat.Type != "private" {
		peerKind = "group"
		peerID = fmt.Sprintf("%d", chatID)
	}

	peer := bus.Peer{Kind: peerKind, ID: peerID}
	messageID := fmt.Sprintf("%d", message.MessageID)

	metadata := map[string]string{
		"user_id":    fmt.Sprintf("%d", user.ID),
		"username":   user.Username,
		"first_name": user.FirstName,
		"is_group":   fmt.Sprintf("%t", message.Chat.Type != "private"),
	}

	c.HandleMessage(c.ctx,
		peer,
		messageID,
		platformID,
		fmt.Sprintf("%d", chatID),
		content,
		mediaPaths,
		metadata,
		sender,
	)
	return nil
}

func (c *TelegramChannel) downloadPhoto(ctx context.Context, fileID string) string {
	file, err := c.bot.GetFile(ctx, &telego.GetFileParams{FileID: fileID})
	if err != nil {
		logger.ErrorCF("telegram", "Failed to get photo file", map[string]any{
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
	logger.DebugCF("telegram", "File URL", map[string]any{"url": url})

	// Use FilePath as filename for better identification
	filename := file.FilePath + ext
	return utils.DownloadFile(url, filename, utils.DownloadOptions{
		LoggerPrefix: "telegram",
	})
}

func (c *TelegramChannel) downloadFile(ctx context.Context, fileID, ext string) string {
	file, err := c.bot.GetFile(ctx, &telego.GetFileParams{FileID: fileID})
	if err != nil {
		logger.ErrorCF("telegram", "Failed to get file", map[string]any{
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

// markdownToTelegramMarkdownV2 takes a standardized markdown string and
// strictly escapes or transforms it to fit Telegram's MarkdownV2 requirements.
// https://core.telegram.org/bots/api#formatting-options
func markdownToTelegramMarkdownV2(text string) string {
	// replace Heading to bolding
	text = reHeading.ReplaceAllString(text, "*$1*")

	var result strings.Builder
	runes := []rune(text)
	length := len(runes)

	// List of characters that must be escaped in standard text contexts
	needsNormalEscape := func(r rune) bool {
		switch r {
		case '_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!':
			return true
		}
		return false
	}

	i := 0
	for i < length {
		// 1. Check for Pre-formatted Code Block (```...```)
		if i+2 < length && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
			result.WriteString("```")
			i += 3
			// Find closing ```
			for i < length {
				if i+2 < length && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
					result.WriteString("```")
					i += 3
					break
				}
				// Inside code blocks, escape `\` and `\`
				if runes[i] == '\\' || runes[i] == '`' {
					result.WriteRune('\\')
				}
				result.WriteRune(runes[i])
				i++
			}
			continue
		}

		// 2. Check for Inline Code (`...`)
		if runes[i] == '`' {
			result.WriteRune('`')
			i++
			for i < length {
				if runes[i] == '`' {
					result.WriteRune('`')
					i++
					break
				}
				if runes[i] == '\\' || runes[i] == '`' {
					result.WriteRune('\\')
				}
				result.WriteRune(runes[i])
				i++
			}
			continue
		}

		// 3. Link or Custom Emoji definition: URL part (...)
		// We detect this by checking if the previous non-space character closed a bracket ']',
		// and we are currently on '('. To keep logic linear, we handle it as we traverse.
		// NOTE: A true deep-parser would link `[` to `](...)`. For safety, whenever we see `(`,
		// if it looks like a URL part, we escape it via URL rules. Let's do a basic lookbehind.
		if runes[i] == '(' && i > 0 && runes[i-1] == ']' {
			result.WriteRune('(')
			i++
			for i < length {
				if runes[i] == ')' {
					// Unescaped closing bracket ends the URL
					result.WriteRune(')')
					i++
					break
				}
				// In URL part, escape `\` and `)`
				if runes[i] == '\\' || runes[i] == ')' {
					result.WriteRune('\\')
				}
				result.WriteRune(runes[i])
				i++
			}
			continue
		}

		// 4. Handle blockquotes starts
		if runes[i] == '>' && (i == 0 || runes[i-1] == '\n') {
			result.WriteRune('>')
			i++
			continue
		}

		// 5. Handle standard Markdown Entities Boundaries
		// If they are part of valid markdown boundaries, we write them as-is.
		// We trust the syntax rules: * _ ~ || [ ]
		// (Assuming the text is a valid markdown, we don't escape these if formatting is intended)

		// Note on Ambiguity (__ vs _):
		// Telegram parses `__` from left to right greedily.
		if i+1 < length && runes[i] == '_' && runes[i+1] == '_' {
			result.WriteString("__")
			i += 2
			continue
		}

		if i+1 < length && runes[i] == '|' && runes[i+1] == '|' {
			result.WriteString("||")
			i += 2
			continue
		}

		// Standard single-char boundaries
		if runes[i] == '*' || runes[i] == '_' || runes[i] == '~' || runes[i] == '[' || runes[i] == ']' {
			result.WriteRune(runes[i])
			i++
			continue
		}

		// Custom emoji boundary check `![`
		if i+1 < length && runes[i] == '!' && runes[i+1] == '[' {
			result.WriteString("![")
			i += 2
			continue
		}

		// 6. Handle plain text characters
		// Escape remaining special characters if they aren't forming intended valid markup
		if needsNormalEscape(runes[i]) {
			// Check if it's already escaped; if an escape character exists, consume it legitimately
			if runes[i] == '\\' && i+1 < length && needsNormalEscape(runes[i+1]) {
				// Keep the backslash and the escaped char as is, avoiding double escaping
				result.WriteRune('\\')
				result.WriteRune(runes[i+1])
				i += 2
				continue
			}

			// Auto-escape the character
			result.WriteRune('\\')
		}

		result.WriteRune(runes[i])
		i++
	}

	return result.String()
}

// isBotMentioned checks if the bot is mentioned in the message via entities.
func (c *TelegramChannel) isBotMentioned(message *telego.Message) bool {
	botUsername := c.bot.Username()
	if botUsername == "" {
		return false
	}

	entities := message.Entities
	if entities == nil {
		entities = message.CaptionEntities
	}

	for _, entity := range entities {
		if entity.Type == "mention" {
			// Extract the mention text from the message
			text := message.Text
			if text == "" {
				text = message.Caption
			}
			runes := []rune(text)
			end := entity.Offset + entity.Length
			if end <= len(runes) {
				mention := string(runes[entity.Offset:end])
				if strings.EqualFold(mention, "@"+botUsername) {
					return true
				}
			}
		}
		if entity.Type == "text_mention" && entity.User != nil {
			if entity.User.Username == botUsername {
				return true
			}
		}
	}
	return false
}

// stripBotMention removes the @bot mention from the content.
func (c *TelegramChannel) stripBotMention(content string) string {
	botUsername := c.bot.Username()
	if botUsername == "" {
		return content
	}
	// Case-insensitive replacement
	re := regexp.MustCompile(`(?i)@` + regexp.QuoteMeta(botUsername))
	content = re.ReplaceAllString(content, "")
	return strings.TrimSpace(content)
}
