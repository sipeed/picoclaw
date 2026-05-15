package telegram

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
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

const (
	defaultMediaGroupDelay = 500 * time.Millisecond
	telegramCaptionLimit   = 1024
)

type TelegramChannel struct {
	*channels.BaseChannel
	bot       *telego.Bot
	bh        *th.BotHandler
	bc        *config.Channel
	chatIDsMu sync.Mutex
	chatIDs   map[string]int64
	ctx       context.Context
	cancel    context.CancelFunc
	tgCfg     *config.TelegramSettings
	progress  *channels.ToolFeedbackAnimator

	registerFunc      func(context.Context, []commands.Definition) error
	commandRegDelayFn func(int) time.Duration
	commandRegCancel  context.CancelFunc

	mediaGroupMu    sync.Mutex
	mediaGroups     map[string]*telegramMediaGroup
	mediaGroupDelay time.Duration
}

type telegramMediaGroup struct {
	messages   []*telego.Message
	timer      *time.Timer
	generation uint64
}

type telegramMessageParts struct {
	content    []string
	mediaPaths []string
}

func NewTelegramChannel(
	bc *config.Channel,
	telegramCfg *config.TelegramSettings,
	bus *bus.MessageBus,
) (*TelegramChannel, error) {
	channelName := bc.Name()
	var opts []telego.BotOption

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

	bot, err := telego.NewBot(telegramCfg.Token.String(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	base := channels.NewBaseChannel(
		channelName,
		telegramCfg,
		bus,
		bc.AllowFrom,
		channels.WithMaxMessageLength(4000),
		channels.WithGroupTrigger(bc.GroupTrigger),
		channels.WithReasoningChannelID(bc.ReasoningChannelID),
	)

	ch := &TelegramChannel{
		BaseChannel: base,
		bot:         bot,
		bc:          bc,
		chatIDs:     make(map[string]int64),
		tgCfg:       telegramCfg,

		mediaGroups:     make(map[string]*telegramMediaGroup),
		mediaGroupDelay: telegramMediaGroupDelay(telegramCfg),
	}
	ch.progress = channels.NewToolFeedbackAnimator(ch.EditMessage)
	return ch, nil
}

func telegramMediaGroupDelay(telegramCfg *config.TelegramSettings) time.Duration {
	if telegramCfg != nil && telegramCfg.MediaGroupDelayMS > 0 {
		return time.Duration(telegramCfg.MediaGroupDelayMS) * time.Millisecond
	}
	return defaultMediaGroupDelay
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
	c.flushPendingMediaGroups(ctx)

	// Cancel our context (stops long polling)
	if c.cancel != nil {
		c.cancel()
	}
	if c.progress != nil {
		c.progress.StopAll()
	}
	if c.commandRegCancel != nil {
		c.commandRegCancel()
	}

	return nil
}

func (c *TelegramChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	useMarkdownV2 := c.tgCfg.UseMarkdownV2

	chatID, threadID, err := resolveTelegramOutboundTarget(msg.ChatID, &msg.Context)
	if err != nil {
		return nil, fmt.Errorf("invalid chat ID %s: %w", msg.ChatID, channels.ErrSendFailed)
	}

	if msg.Content == "" {
		return nil, nil
	}

	isToolFeedback := outboundMessageIsToolFeedback(msg)
	toolFeedbackContent := msg.Content
	if isToolFeedback {
		toolFeedbackContent = fitToolFeedbackForTelegram(msg.Content, useMarkdownV2, 4096)
	}
	trackedChatID := telegramToolFeedbackChatKey(msg.ChatID, &msg.Context)
	if isToolFeedback {
		if msgID, handled, err := c.progress.Update(ctx, trackedChatID, toolFeedbackContent); handled {
			if err != nil {
				return nil, err
			}
			return []string{msgID}, nil
		}
	}
	trackedMsgID, hasTrackedMsg := c.currentToolFeedbackMessage(trackedChatID)
	if !isToolFeedback {
		if msgIDs, handled := c.finalizeToolFeedbackMessageForChat(ctx, trackedChatID, msg); handled {
			return msgIDs, nil
		}
	}

	// The Manager already splits messages to ≤4000 chars (WithMaxMessageLength),
	// so msg.Content is guaranteed to be within that limit. We still need to
	// check if HTML expansion pushes it beyond Telegram's 4096-char API limit.
	replyToID := msg.ReplyToMessageID
	var messageIDs []string
	queue := []string{msg.Content}
	if isToolFeedback {
		queue = []string{channels.InitialAnimatedToolFeedbackContent(toolFeedbackContent)}
	}
	for len(queue) > 0 {
		chunk := queue[0]
		queue = queue[1:]

		content := parseContent(chunk, useMarkdownV2)

		if len([]rune(content)) > 4096 {
			if isToolFeedback {
				fittedChunk := fitToolFeedbackForTelegram(chunk, useMarkdownV2, 4096)
				if fittedChunk != "" && fittedChunk != chunk {
					queue = append([]string{fittedChunk}, queue...)
					continue
				}
			}
			runeChunk := []rune(chunk)
			ratio := float64(len(runeChunk)) / float64(len([]rune(content)))
			smallerLen := int(float64(4096) * ratio * 0.95) // 5% safety margin

			// Guarantee progress: if estimated length is >= chunk length, force it smaller
			if smallerLen >= len(runeChunk) {
				smallerLen = len(runeChunk) - 1
			}

			if smallerLen <= 0 {
				msgID, err := c.sendChunk(ctx, sendChunkParams{
					chatID:        chatID,
					threadID:      threadID,
					content:       content,
					replyToID:     replyToID,
					mdFallback:    chunk,
					useMarkdownV2: useMarkdownV2,
				})
				if err != nil {
					return nil, err
				}
				messageIDs = append(messageIDs, msgID)
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

		msgID, err := c.sendChunk(ctx, sendChunkParams{
			chatID:        chatID,
			threadID:      threadID,
			content:       content,
			replyToID:     replyToID,
			mdFallback:    chunk,
			useMarkdownV2: useMarkdownV2,
		})
		if err != nil {
			return nil, err
		}
		messageIDs = append(messageIDs, msgID)
		// Only the first chunk should be a reply; subsequent chunks are normal messages.
		replyToID = ""
	}

	if isToolFeedback && len(messageIDs) > 0 {
		c.RecordToolFeedbackMessage(trackedChatID, messageIDs[0], toolFeedbackContent)
	} else if !isToolFeedback && hasTrackedMsg {
		c.dismissTrackedToolFeedbackMessage(ctx, trackedChatID, trackedMsgID)
	}

	return messageIDs, nil
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
) (string, error) {
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

	pMsg, err := c.bot.SendMessage(ctx, tgMsg)
	if err != nil {
		logParseFailed(err, params.useMarkdownV2)

		tgMsg.Text = params.mdFallback
		tgMsg.ParseMode = ""
		pMsg, err = c.bot.SendMessage(ctx, tgMsg)
		if err != nil {
			return "", fmt.Errorf("telegram send: %w", channels.ErrTemporary)
		}
	}

	return strconv.Itoa(pMsg.MessageID), nil
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
	useMarkdownV2 := c.tgCfg.UseMarkdownV2
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
		// If it failed because it was already modified (likely from a previous
		// attempt that timed out on our end but landed on Telegram), we treat
		// it as success to prevent the Manager from sending a duplicate message.
		if strings.Contains(err.Error(), "message is not modified") {
			return nil
		}

		// Only fallback to plain text if the error looks like a parsing failure (Bad Request).
		// Network errors or timeouts should NOT trigger a retry with different content.
		if strings.Contains(err.Error(), "Bad Request") {
			logParseFailed(err, useMarkdownV2)
			_, err = c.bot.EditMessageText(ctx, tu.EditMessageText(tu.ID(cid), mid, content))
		}
	}

	if err != nil {
		if strings.Contains(err.Error(), "message is not modified") {
			return nil
		}

		if isPostConnectError(err) {
			logger.WarnCF(
				"telegram",
				"EditMessage likely landed but result is unknown; swallowing error to prevent duplicate",
				map[string]any{
					"chat_id": chatID,
					"mid":     mid,
					"error":   err.Error(),
				},
			)
			return nil // Swallow to prevent Manager fallback to a new SendMessage
		}
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

func outboundMessageIsToolFeedback(msg bus.OutboundMessage) bool {
	if len(msg.Context.Raw) == 0 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(msg.Context.Raw["message_kind"]), "tool_feedback")
}

func (c *TelegramChannel) currentToolFeedbackMessage(chatID string) (string, bool) {
	if c.progress == nil {
		return "", false
	}
	return c.progress.Current(chatID)
}

func (c *TelegramChannel) takeToolFeedbackMessage(chatID string) (string, string, bool) {
	if c.progress == nil {
		return "", "", false
	}
	return c.progress.Take(chatID)
}

func (c *TelegramChannel) RecordToolFeedbackMessage(chatID, messageID, content string) {
	if c.progress == nil {
		return
	}
	c.progress.Record(chatID, messageID, content)
}

func (c *TelegramChannel) ClearToolFeedbackMessage(chatID string) {
	if c.progress == nil {
		return
	}
	c.progress.Clear(chatID)
}

func (c *TelegramChannel) DismissToolFeedbackMessage(ctx context.Context, chatID string) {
	msgID, ok := c.currentToolFeedbackMessage(chatID)
	if !ok {
		return
	}
	c.dismissTrackedToolFeedbackMessage(ctx, chatID, msgID)
}

func (c *TelegramChannel) dismissTrackedToolFeedbackMessage(ctx context.Context, chatID, messageID string) {
	if strings.TrimSpace(chatID) == "" || strings.TrimSpace(messageID) == "" {
		return
	}
	c.ClearToolFeedbackMessage(chatID)
	_ = c.DeleteMessage(ctx, chatID, messageID)
}

func (c *TelegramChannel) finalizeTrackedToolFeedbackMessage(
	ctx context.Context,
	chatID string,
	content string,
	editFn func(context.Context, string, string, string) error,
) ([]string, bool) {
	msgID, baseContent, ok := c.takeToolFeedbackMessage(chatID)
	if !ok || editFn == nil {
		return nil, false
	}
	if err := editFn(ctx, chatID, msgID, content); err != nil {
		c.RecordToolFeedbackMessage(chatID, msgID, baseContent)
		return nil, false
	}
	return []string{msgID}, true
}

func (c *TelegramChannel) FinalizeToolFeedbackMessage(ctx context.Context, msg bus.OutboundMessage) ([]string, bool) {
	if outboundMessageIsToolFeedback(msg) {
		return nil, false
	}
	return c.finalizeToolFeedbackMessageForChat(ctx, telegramToolFeedbackChatKey(msg.ChatID, &msg.Context), msg)
}

func (c *TelegramChannel) finalizeToolFeedbackMessageForChat(
	ctx context.Context,
	chatID string,
	msg bus.OutboundMessage,
) ([]string, bool) {
	return c.finalizeTrackedToolFeedbackMessage(ctx, chatID, msg.Content, c.EditMessage)
}

// SendPlaceholder implements channels.PlaceholderCapable.
// It sends a placeholder message (e.g. "Thinking... 💭") that will later be
// edited to the actual response via EditMessage (channels.MessageEditor).
func (c *TelegramChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	phCfg := c.bc.Placeholder
	if !phCfg.Enabled {
		return "", nil
	}

	text := phCfg.GetRandomText()

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
func (c *TelegramChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}
	trackedChatID := telegramToolFeedbackChatKey(msg.ChatID, &msg.Context)
	trackedMsgID, hasTrackedMsg := c.currentToolFeedbackMessage(trackedChatID)

	chatID, threadID, err := resolveTelegramOutboundTarget(msg.ChatID, &msg.Context)
	if err != nil {
		return nil, fmt.Errorf("invalid chat ID %s: %w", msg.ChatID, channels.ErrSendFailed)
	}

	store := c.GetMediaStore()
	if store == nil {
		return nil, fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	var messageIDs []string
	leadingCaption := telegramLeadingCaption(msg.Parts)
	if len([]rune(leadingCaption)) > telegramCaptionLimit {
		leadingIDs, leadingErr := c.sendCaptionText(ctx, chatID, threadID, leadingCaption)
		if leadingErr != nil {
			return nil, leadingErr
		}
		messageIDs = append(messageIDs, leadingIDs...)
		msg = telegramClearMediaCaptions(msg)
	}

	if len(msg.Parts) > 1 && telegramCanSendMediaGroup(msg.Parts) {
		groupIDs, err := c.sendImageMediaGroups(ctx, chatID, threadID, store, msg.Parts)
		if err != nil {
			logger.ErrorCF("telegram", "Failed to send media group", map[string]any{
				"count": len(msg.Parts),
				"error": err.Error(),
			})
			return nil, fmt.Errorf("telegram send media group: %w", channels.ErrTemporary)
		}
		if len(groupIDs) > 0 {
			messageIDs = append(messageIDs, groupIDs...)
			if hasTrackedMsg {
				c.dismissTrackedToolFeedbackMessage(ctx, trackedChatID, trackedMsgID)
			}
			return messageIDs, nil
		}
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

		var tgResult *telego.Message
		switch part.Type {
		case "image":
			params := &telego.SendPhotoParams{
				ChatID:          tu.ID(chatID),
				MessageThreadID: threadID,
				Photo:           telego.InputFile{File: file},
				Caption:         part.Caption,
			}
			tgResult, err = c.bot.SendPhoto(ctx, params)
			if err != nil && strings.Contains(err.Error(), "PHOTO_INVALID_DIMENSIONS") {
				if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
					file.Close()
					return nil, fmt.Errorf("telegram rewind media after photo failure: %w", channels.ErrTemporary)
				}

				docParams := &telego.SendDocumentParams{
					ChatID:          tu.ID(chatID),
					MessageThreadID: threadID,
					Document:        telego.InputFile{File: file},
					Caption:         part.Caption,
				}
				tgResult, err = c.bot.SendDocument(ctx, docParams)
			}
		case "audio":
			// Send OGG files with "voice" in the filename as Telegram voice
			// bubbles (SendVoice) instead of audio attachments (SendAudio).
			fn := strings.ToLower(part.Filename)
			if strings.Contains(fn, "voice") && (strings.HasSuffix(fn, ".ogg") || strings.HasSuffix(fn, ".oga")) {
				vparams := &telego.SendVoiceParams{
					ChatID:          tu.ID(chatID),
					MessageThreadID: threadID,
					Voice:           telego.InputFile{File: file},
					Caption:         part.Caption,
				}
				tgResult, err = c.bot.SendVoice(ctx, vparams)
			} else {
				params := &telego.SendAudioParams{
					ChatID:          tu.ID(chatID),
					MessageThreadID: threadID,
					Audio:           telego.InputFile{File: file},
					Caption:         part.Caption,
				}
				tgResult, err = c.bot.SendAudio(ctx, params)
			}
		case "video":
			params := &telego.SendVideoParams{
				ChatID:          tu.ID(chatID),
				MessageThreadID: threadID,
				Video:           telego.InputFile{File: file},
				Caption:         part.Caption,
			}
			tgResult, err = c.bot.SendVideo(ctx, params)
		default: // "file" or unknown types
			params := &telego.SendDocumentParams{
				ChatID:          tu.ID(chatID),
				MessageThreadID: threadID,
				Document:        telego.InputFile{File: file},
				Caption:         part.Caption,
			}
			tgResult, err = c.bot.SendDocument(ctx, params)
		}

		if tgResult != nil {
			messageIDs = append(messageIDs, strconv.Itoa(tgResult.MessageID))
		}
		file.Close()

		if err != nil {
			logger.ErrorCF("telegram", "Failed to send media", map[string]any{
				"type":  part.Type,
				"error": err.Error(),
			})
			return nil, fmt.Errorf("telegram send media: %w", channels.ErrTemporary)
		}
	}

	if hasTrackedMsg {
		c.dismissTrackedToolFeedbackMessage(ctx, trackedChatID, trackedMsgID)
	}

	return messageIDs, nil
}

func telegramCanSendMediaGroup(parts []bus.MediaPart) bool {
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part.Type != "image" {
			return false
		}
	}
	return true
}

func (c *TelegramChannel) sendImageMediaGroups(
	ctx context.Context,
	chatID int64,
	threadID int,
	store media.MediaStore,
	parts []bus.MediaPart,
) ([]string, error) {
	const maxGroupSize = 10

	messageIDs := make([]string, 0, len(parts))
	for start := 0; start < len(parts); start += maxGroupSize {
		end := start + maxGroupSize
		if end > len(parts) {
			end = len(parts)
		}
		groupIDs, err := c.sendSingleImageMediaGroup(ctx, chatID, threadID, store, parts[start:end])
		if err != nil {
			return nil, err
		}
		messageIDs = append(messageIDs, groupIDs...)
	}
	return messageIDs, nil
}

func (c *TelegramChannel) sendSingleImageMediaGroup(
	ctx context.Context,
	chatID int64,
	threadID int,
	store media.MediaStore,
	parts []bus.MediaPart,
) ([]string, error) {
	opened := make([]*os.File, 0, len(parts))
	defer func() {
		for _, file := range opened {
			file.Close()
		}
	}()

	inputMedia := make([]telego.InputMedia, 0, len(parts))
	for i, part := range parts {
		localPath, err := store.Resolve(part.Ref)
		if err != nil {
			logger.ErrorCF("telegram", "Failed to resolve media ref for media group", map[string]any{
				"ref":   part.Ref,
				"error": err.Error(),
			})
			return nil, err
		}

		file, err := os.Open(localPath)
		if err != nil {
			logger.ErrorCF("telegram", "Failed to open media file for media group", map[string]any{
				"path":  localPath,
				"error": err.Error(),
			})
			return nil, err
		}
		opened = append(opened, file)

		mediaItem := &telego.InputMediaPhoto{
			Type:  telego.MediaTypePhoto,
			Media: telego.InputFile{File: file},
		}
		if i == 0 {
			mediaItem.Caption = part.Caption
		}
		inputMedia = append(inputMedia, mediaItem)
	}

	results, err := c.bot.SendMediaGroup(ctx, &telego.SendMediaGroupParams{
		ChatID:          tu.ID(chatID),
		MessageThreadID: threadID,
		Media:           inputMedia,
	})
	if err != nil {
		return nil, err
	}

	messageIDs := make([]string, 0, len(results))
	for _, result := range results {
		messageIDs = append(messageIDs, strconv.Itoa(result.MessageID))
	}
	return messageIDs, nil
}

func (c *TelegramChannel) sendCaptionText(
	ctx context.Context,
	chatID int64,
	threadID int,
	text string,
) ([]string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	chunks := channels.SplitMessage(text, c.MaxMessageLength())
	messageIDs := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		msgID, err := c.sendChunk(ctx, sendChunkParams{
			chatID:        chatID,
			threadID:      threadID,
			content:       chunk,
			mdFallback:    chunk,
			useMarkdownV2: false,
		})
		if err != nil {
			return nil, err
		}
		messageIDs = append(messageIDs, msgID)
	}
	return messageIDs, nil
}

func telegramLeadingCaption(parts []bus.MediaPart) string {
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0].Caption)
}

func telegramClearMediaCaptions(msg bus.OutboundMediaMessage) bus.OutboundMediaMessage {
	if len(msg.Parts) == 0 {
		return msg
	}
	cloned := msg
	cloned.Parts = append([]bus.MediaPart(nil), msg.Parts...)
	for i := range cloned.Parts {
		cloned.Parts[i].Caption = ""
	}
	return cloned
}

func (c *TelegramChannel) handleMessage(ctx context.Context, message *telego.Message) error {
	if message != nil && strings.TrimSpace(message.MediaGroupID) != "" {
		return c.bufferMediaGroupMessage(ctx, message)
	}
	return c.handleMessages(ctx, []*telego.Message{message})
}

func (c *TelegramChannel) bufferMediaGroupMessage(ctx context.Context, message *telego.Message) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}
	groupID := strings.TrimSpace(message.MediaGroupID)
	if groupID == "" {
		return c.handleMessages(ctx, []*telego.Message{message})
	}

	msgCopy := *message
	msgCopy.Photo = append([]telego.PhotoSize(nil), message.Photo...)
	key := fmt.Sprintf("%d:%s", message.Chat.ID, groupID)

	c.mediaGroupMu.Lock()
	if c.mediaGroups == nil {
		c.mediaGroups = make(map[string]*telegramMediaGroup)
	}
	group := c.mediaGroups[key]
	if group == nil {
		group = &telegramMediaGroup{}
		c.mediaGroups[key] = group
	}
	group.messages = append(group.messages, &msgCopy)
	group.generation++
	generation := group.generation
	if group.timer != nil {
		group.timer.Stop()
	}
	delay := c.mediaGroupDelay
	if delay <= 0 {
		delay = defaultMediaGroupDelay
	}
	group.timer = time.AfterFunc(delay, func() {
		c.flushMediaGroup(c.ctx, key, generation)
	})
	c.mediaGroupMu.Unlock()

	logger.DebugCF("telegram", "Buffered media group message", map[string]any{
		"chat_id":        message.Chat.ID,
		"media_group_id": groupID,
		"message_id":     message.MessageID,
	})
	return nil
}

func (c *TelegramChannel) flushPendingMediaGroups(ctx context.Context) {
	c.mediaGroupMu.Lock()
	keys := make([]string, 0, len(c.mediaGroups))
	for key, group := range c.mediaGroups {
		if group.timer != nil {
			group.timer.Stop()
		}
		keys = append(keys, key)
	}
	c.mediaGroupMu.Unlock()

	for _, key := range keys {
		c.flushMediaGroup(ctx, key, 0)
	}
}

func (c *TelegramChannel) flushMediaGroup(ctx context.Context, key string, generation uint64) {
	c.mediaGroupMu.Lock()
	group := c.mediaGroups[key]
	if group == nil {
		c.mediaGroupMu.Unlock()
		return
	}
	if generation != 0 && group.generation != generation {
		c.mediaGroupMu.Unlock()
		return
	}
	delete(c.mediaGroups, key)
	if group.timer != nil {
		group.timer.Stop()
	}
	messages := append([]*telego.Message(nil), group.messages...)
	c.mediaGroupMu.Unlock()

	if len(messages) == 0 {
		return
	}
	slices.SortFunc(messages, func(a, b *telego.Message) int {
		switch {
		case a == nil && b == nil:
			return 0
		case a == nil:
			return -1
		case b == nil:
			return 1
		default:
			return a.MessageID - b.MessageID
		}
	})
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.handleMessages(ctx, messages); err != nil {
		logger.ErrorCF("telegram", "Failed to handle media group", map[string]any{
			"key":   key,
			"error": err.Error(),
		})
	}
}

func (c *TelegramChannel) handleMessages(ctx context.Context, messages []*telego.Message) error {
	if len(messages) == 0 {
		return nil
	}
	message := messages[0]
	for _, candidate := range messages {
		if candidate == nil {
			continue
		}
		if strings.TrimSpace(candidate.Text) != "" || strings.TrimSpace(candidate.Caption) != "" {
			message = candidate
			break
		}
	}
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
	c.chatIDsMu.Lock()
	c.chatIDs[platformID] = chatID
	c.chatIDsMu.Unlock()

	content := ""
	mediaPaths := []string{}

	chatIDStr := fmt.Sprintf("%d", chatID)
	messageIDStr := fmt.Sprintf("%d", message.MessageID)
	scope := channels.BuildMediaScope("telegram", chatIDStr, messageIDStr)

	// Helper to register a local file with the media store
	storeMedia := func(localPath, filename string) string {
		if store := c.GetMediaStore(); store != nil {
			ref, err := store.Store(localPath, media.MediaMeta{
				Filename:      filename,
				Source:        "telegram",
				CleanupPolicy: media.CleanupPolicyDeleteOnCleanup,
			}, scope)
			if err == nil {
				return ref
			}
		}
		return localPath // fallback: use raw path
	}

	for i, msg := range messages {
		if msg == nil {
			continue
		}
		parts := c.collectTelegramMessageParts(ctx, msg, i, len(messages), storeMedia)
		for _, part := range parts.content {
			if content != "" {
				content += "\n"
			}
			content += part
		}
		mediaPaths = append(mediaPaths, parts.mediaPaths...)
	}

	if content == "" && len(mediaPaths) == 0 {
		return nil
	}

	if content == "" {
		content = "[media only]"
	}

	// In group chats, apply unified group trigger filtering
	isMentioned := false
	if message.Chat.Type != "private" {
		isMentioned = c.isBotMentioned(message)
		if isMentioned {
			content = c.stripBotMention(content)
		}
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, content)
		if !respond {
			return nil
		}
		content = cleaned
	}

	if message.ReplyToMessage != nil {
		quotedMedia := quotedTelegramMediaRefs(
			message.ReplyToMessage,
			func(fileID, ext, filename string) string {
				localPath := c.downloadFile(ctx, fileID, ext)
				if localPath == "" {
					return ""
				}
				return storeMedia(localPath, filename)
			},
		)
		if len(quotedMedia) > 0 {
			mediaPaths = append(quotedMedia, mediaPaths...)
		}
		content = c.prependTelegramQuotedReply(content, message.ReplyToMessage)
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
	if message.Chat.Type != "private" {
		peerKind = "group"
	}
	messageID := fmt.Sprintf("%d", message.MessageID)

	metadata := map[string]string{
		"user_id":    fmt.Sprintf("%d", user.ID),
		"username":   user.Username,
		"first_name": user.FirstName,
		"is_group":   fmt.Sprintf("%t", message.Chat.Type != "private"),
	}

	inboundCtx := bus.InboundContext{
		Channel:   c.Name(),
		ChatID:    fmt.Sprintf("%d", chatID),
		ChatType:  peerKind,
		SenderID:  platformID,
		MessageID: messageID,
		Mentioned: isMentioned,
		Raw:       metadata,
	}
	if message.Chat.IsForum && threadID != 0 {
		inboundCtx.TopicID = fmt.Sprintf("%d", threadID)
	}
	if message.ReplyToMessage != nil {
		inboundCtx.ReplyToMessageID = fmt.Sprintf("%d", message.ReplyToMessage.MessageID)
	}

	c.HandleMessageWithContext(
		c.ctx,
		compositeChatID,
		content,
		mediaPaths,
		inboundCtx,
		sender,
	)
	return nil
}

func (c *TelegramChannel) collectTelegramMessageParts(
	ctx context.Context,
	msg *telego.Message,
	index int,
	total int,
	storeMedia func(localPath, filename string) string,
) telegramMessageParts {
	parts := telegramMessageParts{}
	if msg == nil {
		return parts
	}
	if text := strings.TrimSpace(msg.Text); text != "" {
		parts.content = append(parts.content, text)
	}
	if caption := strings.TrimSpace(msg.Caption); caption != "" {
		parts.content = append(parts.content, caption)
	}
	if len(msg.Photo) > 0 {
		photo := msg.Photo[len(msg.Photo)-1]
		photoPath := c.downloadPhoto(ctx, photo.FileID)
		if photoPath != "" {
			photoNumber := index + 1
			parts.mediaPaths = append(parts.mediaPaths, storeMedia(photoPath, fmt.Sprintf("photo-%d.jpg", photoNumber)))
			parts.content = append(parts.content, fmt.Sprintf("[image: photo %d]", photoNumber))
		}
	}
	if msg.Voice != nil {
		voicePath := c.downloadFile(ctx, msg.Voice.FileID, ".ogg")
		if voicePath != "" {
			parts.mediaPaths = append(
				parts.mediaPaths,
				storeMedia(voicePath, indexedMediaFilename("voice", ".ogg", index, total)),
			)
			parts.content = append(parts.content, "[voice]")
		}
	}
	if msg.Audio != nil {
		audioPath := c.downloadFile(ctx, msg.Audio.FileID, ".mp3")
		if audioPath != "" {
			filename := msg.Audio.FileName
			if strings.TrimSpace(filename) == "" {
				filename = indexedMediaFilename("audio", ".mp3", index, total)
			}
			parts.mediaPaths = append(parts.mediaPaths, storeMedia(audioPath, filename))
			parts.content = append(parts.content, "[audio]")
		}
	}
	if msg.Document != nil {
		docPath := c.downloadFile(ctx, msg.Document.FileID, "")
		if docPath != "" {
			filename := msg.Document.FileName
			if strings.TrimSpace(filename) == "" {
				filename = indexedMediaFilename("document", "", index, total)
			}
			parts.mediaPaths = append(parts.mediaPaths, storeMedia(docPath, filename))
			parts.content = append(parts.content, "[file]")
		}
	}
	return parts
}

func indexedMediaFilename(prefix, ext string, index int, total int) string {
	if total <= 1 {
		return prefix + ext
	}
	return fmt.Sprintf("%s-%d%s", prefix, index+1, ext)
}

func (c *TelegramChannel) prependTelegramQuotedReply(content string, reply *telego.Message) string {
	quoted := strings.TrimSpace(telegramQuotedContent(reply))
	if quoted == "" {
		return content
	}

	author := telegramQuotedAuthor(reply)
	role := c.telegramQuotedRole(reply)
	if strings.TrimSpace(content) == "" {
		return fmt.Sprintf("[quoted %s message from %s]: %s", role, author, quoted)
	}
	return fmt.Sprintf("[quoted %s message from %s]: %s\n\n%s", role, author, quoted, content)
}

func (c *TelegramChannel) telegramQuotedRole(message *telego.Message) string {
	if message == nil {
		return "unknown"
	}

	if message.From != nil {
		if !message.From.IsBot {
			return "user"
		}
		if c.isOwnBotUser(message.From) {
			return "assistant"
		}
		return "bot"
	}

	if message.SenderChat != nil {
		return "chat"
	}

	return "unknown"
}

func (c *TelegramChannel) isOwnBotUser(user *telego.User) bool {
	if c == nil || c.bot == nil || user == nil || !user.IsBot {
		return false
	}

	if botID := c.bot.ID(); botID != 0 && user.ID == botID {
		return true
	}

	botUsername := strings.TrimPrefix(strings.TrimSpace(c.bot.Username()), "@")
	if botUsername == "" {
		return false
	}
	return strings.EqualFold(strings.TrimPrefix(strings.TrimSpace(user.Username), "@"), botUsername)
}

func telegramQuotedAuthor(message *telego.Message) string {
	if message == nil || message.From == nil {
		return "unknown"
	}
	if username := strings.TrimSpace(message.From.Username); username != "" {
		return username
	}
	if firstName := strings.TrimSpace(message.From.FirstName); firstName != "" {
		return firstName
	}
	return "unknown"
}

func telegramQuotedContent(message *telego.Message) string {
	if message == nil {
		return ""
	}

	var parts []string
	if text := strings.TrimSpace(message.Text); text != "" {
		parts = append(parts, text)
	}
	if caption := strings.TrimSpace(message.Caption); caption != "" {
		parts = append(parts, caption)
	}
	switch {
	case len(message.Photo) > 0:
		parts = append(parts, "[image: photo]")
	}
	switch {
	case message.Voice != nil:
		parts = append(parts, "[voice]")
	case message.Audio != nil:
		parts = append(parts, "[audio]")
	}
	if message.Document != nil {
		parts = append(parts, "[file]")
	}

	return strings.Join(parts, "\n")
}

func quotedTelegramMediaRefs(
	message *telego.Message,
	resolve func(fileID, ext, filename string) string,
) []string {
	if message == nil || resolve == nil {
		return nil
	}

	var refs []string
	if message.Voice != nil {
		if ref := resolve(message.Voice.FileID, ".ogg", "voice.ogg"); ref != "" {
			refs = append(refs, ref)
		}
	}
	if message.Audio != nil {
		if ref := resolve(message.Audio.FileID, ".mp3", "audio.mp3"); ref != "" {
			refs = append(refs, ref)
		}
	}
	return refs
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

func fitToolFeedbackForTelegram(content string, useMarkdownV2 bool, maxParsedLen int) string {
	content = strings.TrimSpace(content)
	if content == "" || maxParsedLen <= 0 {
		return ""
	}
	animationSafeLen := maxParsedLen - channels.MaxToolFeedbackAnimationFrameLength()
	if animationSafeLen <= 0 {
		animationSafeLen = maxParsedLen
	}
	if len([]rune(parseContent(content, useMarkdownV2))) <= animationSafeLen {
		return content
	}

	low := 1
	high := len([]rune(content))
	best := utils.Truncate(content, 1)

	for low <= high {
		mid := (low + high) / 2
		candidate := utils.FitToolFeedbackMessage(content, mid)
		if candidate == "" {
			high = mid - 1
			continue
		}
		if len([]rune(parseContent(candidate, useMarkdownV2))) <= animationSafeLen {
			best = candidate
			low = mid + 1
			continue
		}
		high = mid - 1
	}

	return best
}

func (c *TelegramChannel) PrepareToolFeedbackMessageContent(content string) string {
	if c == nil || c.tgCfg == nil {
		return strings.TrimSpace(content)
	}
	return fitToolFeedbackForTelegram(content, c.tgCfg.UseMarkdownV2, 4096)
}

func telegramToolFeedbackChatKey(chatID string, outboundCtx *bus.InboundContext) string {
	resolvedChatID, threadID, err := resolveTelegramOutboundTarget(chatID, outboundCtx)
	if err != nil || threadID == 0 {
		return strings.TrimSpace(chatID)
	}
	return fmt.Sprintf("%d/%d", resolvedChatID, threadID)
}

func (c *TelegramChannel) ToolFeedbackMessageChatID(chatID string, outboundCtx *bus.InboundContext) string {
	return telegramToolFeedbackChatKey(chatID, outboundCtx)
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

func resolveTelegramOutboundTarget(chatID string, outboundCtx *bus.InboundContext) (int64, int, error) {
	targetChatID := strings.TrimSpace(chatID)
	if targetChatID == "" && outboundCtx != nil {
		targetChatID = strings.TrimSpace(outboundCtx.ChatID)
	}
	resolvedChatID, resolvedThreadID, err := parseTelegramChatID(targetChatID)
	if err != nil {
		return 0, 0, err
	}
	if resolvedThreadID != 0 || outboundCtx == nil {
		return resolvedChatID, resolvedThreadID, nil
	}
	topicID := strings.TrimSpace(outboundCtx.TopicID)
	if topicID == "" {
		return resolvedChatID, resolvedThreadID, nil
	}
	if threadID, convErr := strconv.Atoi(topicID); convErr == nil {
		return resolvedChatID, threadID, nil
	}
	return resolvedChatID, resolvedThreadID, nil
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
	if !c.tgCfg.Streaming.Enabled {
		return nil, fmt.Errorf("streaming disabled in config")
	}

	cid, threadID, err := parseTelegramChatID(chatID)
	if err != nil {
		return nil, err
	}

	streamCfg := c.tgCfg.Streaming
	return &telegramStreamer{
		bot:              c.bot,
		chatID:           cid,
		threadID:         threadID,
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
	threadID         int
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
		ChatID:          s.chatID,
		MessageThreadID: s.threadID,
		DraftID:         s.draftID,
		Text:            htmlContent,
		ParseMode:       telego.ModeHTML,
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
	tgMsg.MessageThreadID = s.threadID
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

// isPostConnectError identifies network errors that likely occurred after
// the request was transmitted to Telegram (e.g. dropped connection while
// waiting for response). Swallowing these for edits prevents duplicate
// fallbacks, at the small risk of leaving a stale placeholder if the
// edit never actually reached the server.
func isPostConnectError(err error) bool {
	if err == nil {
		return false
	}

	// Context errors (timeout/canceled) are too broad; they can be triggered
	// locally before any data is sent. Never swallow them.
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}

	msg := strings.ToLower(err.Error())
	// Narrowly target connection dropouts where the request likely landed.
	return strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "unexpected eof") ||
		strings.Contains(msg, "connection closed by foreign host") ||
		strings.Contains(msg, "broken pipe")
}

// VoiceCapabilities returns the voice capabilities of the channel.
func (c *TelegramChannel) VoiceCapabilities() channels.VoiceCapabilities {
	return channels.VoiceCapabilities{ASR: true, TTS: true}
}
