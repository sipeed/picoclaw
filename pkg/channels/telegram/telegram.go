package telegram

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
)

var (
	reHeading    = regexp.MustCompile(`(?m)^#{1,6}\s+([^\n]+)`)
	reBlockquote = regexp.MustCompile(`^>\s*(.*)$`)
	reLink       = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reBoldStar   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reBoldUnder  = regexp.MustCompile(`__(.+?)__`)
	reItalic     = regexp.MustCompile(`_([^_]+)_`)
	reStrike     = regexp.MustCompile(`~~(.+?)~~`)
	reListItem   = regexp.MustCompile(`^[-*]\s+`)
	reCodeBlock  = regexp.MustCompile("```[\\w]*\\n?([\\s\\S]*?)```")
	reInlineCode = regexp.MustCompile("`([^`]+)`")
)

type TelegramChannel struct {
	*channels.BaseChannel
	bot     *telego.Bot
	bh      *th.BotHandler
	config  *config.Config
	chatIDs map[string]int64
	ctx     context.Context
	cancel  context.CancelFunc

	registerFunc     func(context.Context, []commands.Definition) error
	commandRegCancel context.CancelFunc
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

	if baseURL := strings.TrimRight(strings.TrimSpace(telegramCfg.BaseURL), "/"); baseURL != "" {
		opts = append(opts, telego.WithAPIServer(baseURL))
	}
	opts = append(opts, telego.WithLogger(logger.NewLogger("telego")))

	bot, err := telego.NewBot(telegramCfg.Token, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	base := channels.NewBaseChannel(
		"telegram",
		telegramCfg,
		bus,
		telegramCfg.AllowFrom,
		channels.WithMaxMessageLength(4000),
		channels.WithGroupTrigger(telegramCfg.GroupTrigger),
		channels.WithReasoningChannelID(telegramCfg.ReasoningChannelID),
	)

	return &TelegramChannel{
		BaseChannel: base,
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

	bh, err := th.NewBotHandler(c.bot, updates)
	if err != nil {
		c.cancel()
		return fmt.Errorf("failed to create bot handler: %w", err)
	}
	c.bh = bh

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		return c.handleMessage(ctx, &message)
	}, th.AnyMessage())

	c.SetRunning(true)
	logger.InfoCF("telegram", "Telegram bot connected", map[string]any{
		"username": c.bot.Username(),
	})

	c.startCommandRegistration(c.ctx, commands.BuiltinDefinitions())

	go func() {
		if err = bh.Start(); err != nil {
			logger.ErrorCF("telegram", "Bot handler failed", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	return nil
}

func (c *TelegramChannel) Stop(ctx context.Context) error {
	logger.InfoC("telegram", "Stopping Telegram bot...")
	c.SetRunning(false)

	// Stop the bot handler
	if c.bh != nil {
		_ = c.bh.StopWithContext(ctx)
	}

	// Cancel our context (stops long polling)
	if c.cancel != nil {
		c.cancel()
	}
	if c.commandRegCancel != nil {
		c.commandRegCancel()
	}

	return nil
}

func (c *TelegramChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	useMarkdownV2 := c.config.Channels.Telegram.UseMarkdownV2

	chatID, threadID, err := parseTelegramChatID(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid chat ID %s: %w", msg.ChatID, channels.ErrSendFailed)
	}

	if msg.Content == "" {
		return nil
	}

	// The Manager already splits messages to ≤4000 chars (WithMaxMessageLength),
	// so msg.Content is guaranteed to be within that limit. We still need to
	// check if HTML expansion pushes it beyond Telegram's 4096-char API limit.
	replyToID := msg.ReplyToMessageID
	queue := []string{msg.Content}
	for len(queue) > 0 {
		chunk := queue[0]
		queue = queue[1:]

		content := parseContent(chunk, useMarkdownV2)

		if len([]rune(content)) > 4096 {
			runeChunk := []rune(chunk)
			ratio := float64(len(runeChunk)) / float64(len([]rune(content)))
			smallerLen := int(float64(4096) * ratio * 0.95) // 5% safety margin

			// Guarantee progress: if estimated length is >= chunk length, force it smaller
			if smallerLen >= len(runeChunk) {
				smallerLen = len(runeChunk) - 1
			}

			if smallerLen <= 0 {
				if err := c.sendChunk(ctx, sendChunkParams{
					chatID:        chatID,
					threadID:      threadID,
					content:       content,
					replyToID:     replyToID,
					mdFallback:    chunk,
					useMarkdownV2: useMarkdownV2,
				}); err != nil {
					return err
				}
				replyToID = ""
				continue
			}

			// Use the estimated smaller length as a guide for SplitMessage.
			// SplitMessage will find natural break points (newlines/spaces) and respect code blocks.
			subChunks := channels.SplitMessage(chunk, smallerLen)

			// Safety fallback: If SplitMessage failed to shorten the chunk, force a manual hard split.
			if len(subChunks) == 1 && subChunks[0] == chunk {
				part1 := string(runeChunk[:smallerLen])
				part2 := string(runeChunk[smallerLen:])
				subChunks = []string{part1, part2}
			}

			// Filter out empty chunks to avoid sending empty messages to Telegram.
			nonEmpty := make([]string, 0, len(subChunks))
			for _, s := range subChunks {
				if s != "" {
					nonEmpty = append(nonEmpty, s)
				}
			}

			// Push sub-chunks back to the front of the queue
			queue = append(nonEmpty, queue...)
			continue
		}

		if err := c.sendChunk(ctx, sendChunkParams{
			chatID:        chatID,
			threadID:      threadID,
			content:       content,
			replyToID:     replyToID,
			mdFallback:    chunk,
			useMarkdownV2: useMarkdownV2,
		}); err != nil {
			return err
		}
		// Only the first chunk should be a reply; subsequent chunks are normal messages.
		replyToID = ""
	}

	return nil
}

type sendChunkParams struct {
	chatID        int64
	threadID      int
	content       string
	replyToID     string
	mdFallback    string
	useMarkdownV2 bool
}

// sendChunk sends a single HTML/MarkdownV2 message, falling back to the original
// markdown as plain text on parse failure so users never see raw HTML/MarkdownV2 tags.
func (c *TelegramChannel) sendChunk(
	ctx context.Context,
	params sendChunkParams,
) error {
	tgMsg := tu.Message(tu.ID(params.chatID), params.content)
	tgMsg.MessageThreadID = params.threadID
	if params.useMarkdownV2 {
		tgMsg.WithParseMode(telego.ModeMarkdownV2)
	} else {
		tgMsg.WithParseMode(telego.ModeHTML)
	}

	if params.replyToID != "" {
		if mid, parseErr := strconv.Atoi(params.replyToID); parseErr == nil {
			tgMsg.ReplyParameters = &telego.ReplyParameters{
				MessageID: mid,
			}
		}
	}

	if _, err := c.bot.SendMessage(ctx, tgMsg); err != nil {
		logParseFailed(err, params.useMarkdownV2)

		tgMsg.Text = params.mdFallback
		tgMsg.ParseMode = ""
		if _, err = c.bot.SendMessage(ctx, tgMsg); err != nil {
			return fmt.Errorf("telegram send: %w", channels.ErrTemporary)
		}
	}

	return nil
}

// maxTypingDuration limits how long the typing indicator can run.
// Prevents endless typing when the LLM fails/hangs and preSend never invokes cancel.
// Matches channels.Manager's typingStopTTL (5 min) so behavior is consistent.
const maxTypingDuration = 5 * time.Minute

// StartTyping implements channels.TypingCapable.
// It sends ChatAction(typing) immediately and then repeats every 4 seconds
// (Telegram's typing indicator expires after ~5s) in a background goroutine.
// The returned stop function is idempotent and cancels the goroutine.
// The goroutine also exits automatically after maxTypingDuration if cancel is
// never called (e.g. when the LLM fails or times out without publishing).
func (c *TelegramChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	cid, threadID, err := parseTelegramChatID(chatID)
	if err != nil {
		return func() {}, err
	}

	action := tu.ChatAction(tu.ID(cid), telego.ChatActionTyping)
	action.MessageThreadID = threadID

	// Send the first typing action immediately
	_ = c.bot.SendChatAction(ctx, action)

	typingCtx, cancel := context.WithCancel(ctx)
	// Cap lifetime so the goroutine cannot run indefinitely if cancel is never called
	maxCtx, maxCancel := context.WithTimeout(typingCtx, maxTypingDuration)
	go func() {
		defer maxCancel()
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-maxCtx.Done():
				return
			case <-ticker.C:
				a := tu.ChatAction(tu.ID(cid), telego.ChatActionTyping)
				a.MessageThreadID = threadID
				_ = c.bot.SendChatAction(typingCtx, a)
			}
		}
	}()

	return cancel, nil
}

// EditMessage implements channels.MessageEditor.
func (c *TelegramChannel) EditMessage(ctx context.Context, chatID string, messageID string, content string) error {
	useMarkdownV2 := c.config.Channels.Telegram.UseMarkdownV2
	cid, _, err := parseTelegramChatID(chatID)
	if err != nil {
		return err
	}
	mid, err := strconv.Atoi(messageID)
	if err != nil {
		return err
	}
	parsedContent := parseContent(content, useMarkdownV2)
	editMsg := tu.EditMessageText(tu.ID(cid), mid, parsedContent)
	if useMarkdownV2 {
		editMsg.WithParseMode(telego.ModeMarkdownV2)
	} else {
		editMsg.WithParseMode(telego.ModeHTML)
	}
	_, err = c.bot.EditMessageText(ctx, editMsg)
	if err != nil {
		logParseFailed(err, useMarkdownV2)
		_, err = c.bot.EditMessageText(ctx, tu.EditMessageText(tu.ID(cid), mid, content))
	}

	return err
}

// DeleteMessage implements channels.MessageDeleter.
func (c *TelegramChannel) DeleteMessage(ctx context.Context, chatID string, messageID string) error {
	cid, _, err := parseTelegramChatID(chatID)
	if err != nil {
		return err
	}
	mid, err := strconv.Atoi(messageID)
	if err != nil {
		return err
	}
	return c.bot.DeleteMessage(ctx, &telego.DeleteMessageParams{
		ChatID:    tu.ID(cid),
		MessageID: mid,
	})
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

	cid, threadID, err := parseTelegramChatID(chatID)
	if err != nil {
		return "", err
	}

	phMsg := tu.Message(tu.ID(cid), text)
	phMsg.MessageThreadID = threadID
	pMsg, err := c.bot.SendMessage(ctx, phMsg)
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

	chatID, threadID, err := parseTelegramChatID(msg.ChatID)
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
				ChatID:          tu.ID(chatID),
				MessageThreadID: threadID,
				Photo:           telego.InputFile{File: file},
				Caption:         part.Caption,
			}
			_, err = c.bot.SendPhoto(ctx, params)
			if err != nil && strings.Contains(err.Error(), "PHOTO_INVALID_DIMENSIONS") {
				if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
					file.Close()
					return fmt.Errorf("telegram rewind media after photo failure: %w", channels.ErrTemporary)
				}

				docParams := &telego.SendDocumentParams{
					ChatID:          tu.ID(chatID),
					MessageThreadID: threadID,
					Document:        telego.InputFile{File: file},
					Caption:         part.Caption,
				}
				_, err = c.bot.SendDocument(ctx, docParams)
			}
		case "audio":
			params := &telego.SendAudioParams{
				ChatID:          tu.ID(chatID),
				MessageThreadID: threadID,
				Audio:           telego.InputFile{File: file},
				Caption:         part.Caption,
			}
			_, err = c.bot.SendAudio(ctx, params)
		case "video":
			params := &telego.SendVideoParams{
				ChatID:          tu.ID(chatID),
				MessageThreadID: threadID,
				Video:           telego.InputFile{File: file},
				Caption:         part.Caption,
			}
			_, err = c.bot.SendVideo(ctx, params)
		default: // "file" or unknown types
			params := &telego.SendDocumentParams{
				ChatID:          tu.ID(chatID),
				MessageThreadID: threadID,
				Document:        telego.InputFile{File: file},
				Caption:         part.Caption,
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

	// For forum topics, embed the thread ID as "chatID/threadID" so replies
	// route to the correct topic and each topic gets its own session.
	// Only forum groups (IsForum) are handled; regular group reply threads
	// must share one session per group.
	compositeChatID := fmt.Sprintf("%d", chatID)
	threadID := message.MessageThreadID
	if message.Chat.IsForum && threadID != 0 {
		compositeChatID = fmt.Sprintf("%d/%d", chatID, threadID)
	}

	logger.DebugCF("telegram", "Received message", map[string]any{
		"sender_id": sender.CanonicalID,
		"chat_id":   compositeChatID,
		"thread_id": threadID,
		"preview":   utils.Truncate(content, 50),
	})

	peerKind := "direct"
	peerID := fmt.Sprintf("%d", user.ID)
	if message.Chat.Type != "private" {
		peerKind = "group"
		peerID = compositeChatID
	}

	peer := bus.Peer{Kind: peerKind, ID: peerID}
	messageID := fmt.Sprintf("%d", message.MessageID)

	metadata := map[string]string{
		"user_id":    fmt.Sprintf("%d", user.ID),
		"username":   user.Username,
		"first_name": user.FirstName,
		"is_group":   fmt.Sprintf("%t", message.Chat.Type != "private"),
	}

	// Set parent_peer metadata for per-topic agent binding.
	if message.Chat.IsForum && threadID != 0 {
		metadata["parent_peer_kind"] = "topic"
		metadata["parent_peer_id"] = fmt.Sprintf("%d", threadID)
	}

	c.HandleMessage(c.ctx,
		peer,
		messageID,
		platformID,
		compositeChatID,
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

func parseContent(text string, useMarkdownV2 bool) string {
	if useMarkdownV2 {
		return markdownToTelegramMarkdownV2(text)
	}

	return markdownToTelegramHTML(text)
}

// parseTelegramChatID splits "chatID/threadID" into its components.
// Returns threadID=0 when no "/" is present (non-forum messages).
func parseTelegramChatID(chatID string) (int64, int, error) {
	idx := strings.Index(chatID, "/")
	if idx == -1 {
		cid, err := strconv.ParseInt(chatID, 10, 64)
		return cid, 0, err
	}
	cid, err := strconv.ParseInt(chatID[:idx], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	tid, err := strconv.Atoi(chatID[idx+1:])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid thread ID in chat ID %q: %w", chatID, err)
	}
	return cid, tid, nil
}

func logParseFailed(err error, useMarkdownV2 bool) {
	parsingName := "HTML"
	if useMarkdownV2 {
		parsingName = "MarkdownV2"
	}

	logger.ErrorCF("telegram",
		fmt.Sprintf("%s parse failed, falling back to plain text", parsingName),
		map[string]any{
			"error": err.Error(),
		},
	)
}

// isBotMentioned checks if the bot is mentioned in the message via entities.
func (c *TelegramChannel) isBotMentioned(message *telego.Message) bool {
	text, entities := telegramEntityTextAndList(message)
	if text == "" || len(entities) == 0 {
		return false
	}

	botUsername := ""
	if c.bot != nil {
		botUsername = c.bot.Username()
	}
	runes := []rune(text)

	for _, entity := range entities {
		entityText, ok := telegramEntityText(runes, entity)
		if !ok {
			continue
		}

		switch entity.Type {
		case telego.EntityTypeMention:
			if botUsername != "" && strings.EqualFold(entityText, "@"+botUsername) {
				return true
			}
		case telego.EntityTypeTextMention:
			if botUsername != "" && entity.User != nil && strings.EqualFold(entity.User.Username, botUsername) {
				return true
			}
		case telego.EntityTypeBotCommand:
			if isBotCommandEntityForThisBot(entityText, botUsername) {
				return true
			}
		}
	}
	return false
}

func telegramEntityTextAndList(message *telego.Message) (string, []telego.MessageEntity) {
	if message.Text != "" {
		return message.Text, message.Entities
	}
	return message.Caption, message.CaptionEntities
}

func telegramEntityText(runes []rune, entity telego.MessageEntity) (string, bool) {
	if entity.Offset < 0 || entity.Length <= 0 {
		return "", false
	}
	end := entity.Offset + entity.Length
	if entity.Offset >= len(runes) || end > len(runes) {
		return "", false
	}
	return string(runes[entity.Offset:end]), true
}

func isBotCommandEntityForThisBot(entityText, botUsername string) bool {
	if !strings.HasPrefix(entityText, "/") {
		return false
	}
	command := strings.TrimPrefix(entityText, "/")
	if command == "" {
		return false
	}

	at := strings.IndexRune(command, '@')
	if at == -1 {
		// A bare /command delivered to this bot is intended for this bot.
		return true
	}

	mentionUsername := command[at+1:]
	if mentionUsername == "" || botUsername == "" {
		return false
	}
	return strings.EqualFold(mentionUsername, botUsername)
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

// BeginStream implements channels.StreamingCapable.
func (c *TelegramChannel) BeginStream(ctx context.Context, chatID string) (channels.Streamer, error) {
	if !c.config.Channels.Telegram.Streaming.Enabled {
		return nil, fmt.Errorf("streaming disabled in config")
	}

	cid, _, err := parseTelegramChatID(chatID)
	if err != nil {
		return nil, err
	}

	streamCfg := c.config.Channels.Telegram.Streaming
	return &telegramStreamer{
		bot:              c.bot,
		chatID:           cid,
		draftID:          cryptoRandInt(),
		throttleInterval: time.Duration(streamCfg.ThrottleSeconds) * time.Second,
		minGrowth:        streamCfg.MinGrowthChars,
	}, nil
}

// telegramStreamer streams partial LLM output via Telegram's sendMessageDraft API.
// On first API error (e.g. bot lacks forum mode), it silently degrades: Update
// becomes a no-op, while Finalize still delivers the final message.
type telegramStreamer struct {
	bot              *telego.Bot
	chatID           int64
	draftID          int
	throttleInterval time.Duration
	minGrowth        int
	lastLen          int
	lastAt           time.Time
	failed           bool
	mu               sync.Mutex
}

func (s *telegramStreamer) Update(ctx context.Context, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failed {
		return nil
	}

	// Throttle: skip if not enough time or content has passed
	now := time.Now()
	growth := len(content) - s.lastLen
	if s.lastLen > 0 && now.Sub(s.lastAt) < s.throttleInterval && growth < s.minGrowth {
		return nil
	}

	htmlContent := markdownToTelegramHTML(content)

	err := s.bot.SendMessageDraft(ctx, &telego.SendMessageDraftParams{
		ChatID:    s.chatID,
		DraftID:   s.draftID,
		Text:      htmlContent,
		ParseMode: telego.ModeHTML,
	})
	if err != nil {
		// First error → degrade silently (e.g. no forum mode)
		logger.WarnCF("telegram", "sendMessageDraft failed, disabling streaming", map[string]any{
			"error": err.Error(),
		})
		s.failed = true
		return nil // don't propagate — Finalize will still deliver
	}

	s.lastLen = len(content)
	s.lastAt = now
	return nil
}

func (s *telegramStreamer) Finalize(ctx context.Context, content string) error {
	htmlContent := markdownToTelegramHTML(content)
	tgMsg := tu.Message(tu.ID(s.chatID), htmlContent)
	tgMsg.ParseMode = telego.ModeHTML

	if _, err := s.bot.SendMessage(ctx, tgMsg); err != nil {
		// Fallback to plain text
		tgMsg.ParseMode = ""
		if _, err = s.bot.SendMessage(ctx, tgMsg); err != nil {
			logger.ErrorCF("telegram", "Finalize failed after HTML and plain-text attempts", map[string]any{
				"chat_id": s.chatID,
				"error":   err.Error(),
				"len":     len(content),
			})
			return fmt.Errorf("telegram finalize: %w", err)
		}
	}
	return nil
}

func (s *telegramStreamer) Cancel(ctx context.Context) {
	// Draft auto-expires on Telegram's side; nothing to clean up.
}

// cryptoRandInt returns a non-zero random int using crypto/rand.
func cryptoRandInt() int {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return int(binary.BigEndian.Uint32(b[:])) | 1 // ensure non-zero
}
