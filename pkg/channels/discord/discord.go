package discord

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

	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/websocket"

	"jane/pkg/bus"
	"jane/pkg/channels"
	"jane/pkg/config"
	"jane/pkg/identity"
	"jane/pkg/logger"
	"jane/pkg/media"
	"jane/pkg/utils"
)

const (
	sendTimeout = 10 * time.Second
)

var (
	// Pre-compiled regexes for resolveDiscordRefs (avoid re-compiling per call)
	channelRefRe = regexp.MustCompile(`<#(\d+)>`)
	msgLinkRe    = regexp.MustCompile(`https://(?:discord\.com|discordapp\.com)/channels/(\d+)/(\d+)/(\d+)`)
)

type DiscordChannel struct {
	*channels.BaseChannel
	session    *discordgo.Session
	config     config.DiscordConfig
	ctx        context.Context
	cancel     context.CancelFunc
	typingMu   sync.Mutex
	typingStop map[string]chan struct{} // chatID → stop signal
	botUserID  string                   // stored for mention checking
}

func NewDiscordChannel(cfg config.DiscordConfig, bus *bus.MessageBus) (*DiscordChannel, error) {
	discordgo.Logger = logger.NewLogger("discord").
		WithLevels(map[int]logger.LogLevel{
			discordgo.LogError:         logger.ERROR,
			discordgo.LogWarning:       logger.WARN,
			discordgo.LogInformational: logger.INFO,
			discordgo.LogDebug:         logger.DEBUG,
		}).Log

	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord session: %w", err)
	}

	if err := applyDiscordProxy(session, cfg.Proxy); err != nil {
		return nil, err
	}
	base := channels.NewBaseChannel("discord", cfg, bus, cfg.AllowFrom,
		channels.WithMaxMessageLength(2000),
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &DiscordChannel{
		BaseChannel: base,
		session:     session,
		config:      cfg,
		ctx:         context.Background(),
		typingStop:  make(map[string]chan struct{}),
	}, nil
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

	return c.sendChunk(ctx, channelID, msg.Content, msg.ReplyToMessageID)
}

// SendMedia implements the channels.MediaSender interface.
func (c *DiscordChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	channelID := msg.ChatID
	if channelID == "" {
		return fmt.Errorf("channel ID is empty")
	}

	store := c.GetMediaStore()
	if store == nil {
		return fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	// Collect all files into a single ChannelMessageSendComplex call
	files := make([]*discordgo.File, 0, len(msg.Parts))
	var caption string

	for _, part := range msg.Parts {
		localPath, err := store.Resolve(part.Ref)
		if err != nil {
			logger.ErrorCF("discord", "Failed to resolve media ref", map[string]any{
				"ref":   part.Ref,
				"error": err.Error(),
			})
			continue
		}

		file, err := os.Open(localPath)
		if err != nil {
			logger.ErrorCF("discord", "Failed to open media file", map[string]any{
				"path":  localPath,
				"error": err.Error(),
			})
			continue
		}
		// Note: discordgo reads from the Reader and we can't close it before send

		filename := part.Filename
		if filename == "" {
			filename = "file"
		}

		files = append(files, &discordgo.File{
			Name:        filename,
			ContentType: part.ContentType,
			Reader:      file,
		})

		if part.Caption != "" && caption == "" {
			caption = part.Caption
		}
	}

	if len(files) == 0 {
		return nil
	}

	sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := c.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content: caption,
			Files:   files,
		})
		done <- err
	}()

	select {
	case err := <-done:
		// Close all file readers
		for _, f := range files {
			if closer, ok := f.Reader.(*os.File); ok {
				closer.Close()
			}
		}
		if err != nil {
			return fmt.Errorf("discord send media: %w", channels.ErrTemporary)
		}
		return nil
	case <-sendCtx.Done():
		// Close all file readers
		for _, f := range files {
			if closer, ok := f.Reader.(*os.File); ok {
				closer.Close()
			}
		}
		return sendCtx.Err()
	}
}

// EditMessage implements channels.MessageEditor.
func (c *DiscordChannel) EditMessage(ctx context.Context, chatID string, messageID string, content string) error {
	_, err := c.session.ChannelMessageEdit(chatID, messageID, content)
	return err
}

// SendPlaceholder implements channels.PlaceholderCapable.
// It sends a placeholder message that will later be edited to the actual
// response via EditMessage (channels.MessageEditor).
func (c *DiscordChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	if !c.config.Placeholder.Enabled {
		return "", nil
	}

	text := c.config.Placeholder.Text
	if text == "" {
		text = "Thinking... 💭"
	}

	msg, err := c.session.ChannelMessageSend(chatID, text)
	if err != nil {
		return "", err
	}

	return msg.ID, nil
}

func (c *DiscordChannel) sendChunk(ctx context.Context, channelID, content, replyToID string) error {
	// Use the passed ctx for timeout control
	sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		var err error

		// If we have an ID, we send the message as "Reply"
		if replyToID != "" {
			_, err = c.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
				Content: content,
				Reference: &discordgo.MessageReference{
					MessageID: replyToID,
					ChannelID: channelID,
				},
			})
		} else {
			// Otherwise, we send a normal message
			_, err = c.session.ChannelMessageSend(channelID, content)
		}

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

	// Check allowlist first to avoid downloading attachments for rejected users
	sender := bus.SenderInfo{
		Platform:    "discord",
		PlatformID:  m.Author.ID,
		CanonicalID: identity.BuildCanonicalID("discord", m.Author.ID),
		Username:    m.Author.Username,
	}
	// Build display name
	displayName := m.Author.Username
	if m.Author.Discriminator != "" && m.Author.Discriminator != "0" {
		displayName += "#" + m.Author.Discriminator
	}
	sender.DisplayName = displayName

	if !c.IsAllowedSender(sender) {
		logger.DebugCF("discord", "Message rejected by allowlist", map[string]any{
			"user_id": m.Author.ID,
		})
		return
	}

	content := m.Content

	// In guild (group) channels, apply unified group trigger filtering
	// DMs (GuildID is empty) always get a response
	if m.GuildID != "" {
		isMentioned := false
		for _, mention := range m.Mentions {
			if mention.ID == c.botUserID {
				isMentioned = true
				break
			}
		}
		content = c.stripBotMention(content)
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, content)
		if !respond {
			logger.DebugCF("discord", "Group message ignored by group trigger", map[string]any{
				"user_id": m.Author.ID,
			})
			return
		}
		content = cleaned
	} else {
		// DMs: just strip bot mention without filtering
		content = c.stripBotMention(content)
	}

	// Resolve Discord refs in main content before concatenation to avoid
	// double-expanding links that appear in the referenced message.
	content = c.resolveDiscordRefs(s, content, m.GuildID)

	// Prepend referenced (quoted) message content if this is a reply
	if m.MessageReference != nil && m.ReferencedMessage != nil {
		refContent := m.ReferencedMessage.Content
		if refContent != "" {
			refAuthor := "unknown"
			if m.ReferencedMessage.Author != nil {
				refAuthor = m.ReferencedMessage.Author.Username
			}
			refContent = c.resolveDiscordRefs(s, refContent, m.GuildID)
			content = fmt.Sprintf("[quoted message from %s]: %s\n\n%s",
				refAuthor, refContent, content)
		}
	}

	senderID := m.Author.ID

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
				mediaPaths = append(mediaPaths, storeMedia(localPath, attachment.Filename))
				content = appendContent(content, fmt.Sprintf("[audio: %s]", attachment.Filename))
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
		"sender_name": sender.DisplayName,
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
		"display_name": sender.DisplayName,
		"guild_id":     m.GuildID,
		"channel_id":   m.ChannelID,
		"is_dm":        fmt.Sprintf("%t", m.GuildID == ""),
	}

	c.HandleMessage(c.ctx, peer, m.ID, senderID, m.ChannelID, content, mediaPaths, metadata, sender)
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

// StartTyping implements channels.TypingCapable.
// It starts a continuous typing indicator and returns an idempotent stop function.
func (c *DiscordChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	c.startTyping(chatID)
	return func() { c.stopTyping(chatID) }, nil
}

func (c *DiscordChannel) downloadAttachment(url, filename string) string {
	return utils.DownloadFile(url, filename, utils.DownloadOptions{
		LoggerPrefix: "discord",
		ProxyURL:     c.config.Proxy,
	})
}

func applyDiscordProxy(session *discordgo.Session, proxyAddr string) error {
	var proxyFunc func(*http.Request) (*url.URL, error)
	if proxyAddr != "" {
		proxyURL, err := url.Parse(proxyAddr)
		if err != nil {
			return fmt.Errorf("invalid discord proxy URL %q: %w", proxyAddr, err)
		}
		proxyFunc = http.ProxyURL(proxyURL)
	} else if os.Getenv("HTTP_PROXY") != "" || os.Getenv("HTTPS_PROXY") != "" {
		proxyFunc = http.ProxyFromEnvironment
	}

	if proxyFunc == nil {
		return nil
	}

	transport := &http.Transport{Proxy: proxyFunc}
	session.Client = &http.Client{
		Timeout:   sendTimeout,
		Transport: transport,
	}

	if session.Dialer != nil {
		dialerCopy := *session.Dialer
		dialerCopy.Proxy = proxyFunc
		session.Dialer = &dialerCopy
	} else {
		session.Dialer = &websocket.Dialer{Proxy: proxyFunc}
	}

	return nil
}

// resolveDiscordRefs resolves channel references (<#id> → #channel-name) and
// expands Discord message links to show the linked message content.
// Only links pointing to the same guild are expanded to prevent cross-guild leakage.
func (c *DiscordChannel) resolveDiscordRefs(s *discordgo.Session, text string, guildID string) string {
	// 1. Resolve channel references: <#id> → #channel-name
	text = channelRefRe.ReplaceAllStringFunc(text, func(match string) string {
		parts := channelRefRe.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		// Prefer session state cache to avoid API calls
		if ch, err := s.State.Channel(parts[1]); err == nil {
			return "#" + ch.Name
		}
		if ch, err := s.Channel(parts[1]); err == nil {
			return "#" + ch.Name
		}
		return match
	})

	// 2. Expand Discord message links (max 3, same guild only)
	matches := msgLinkRe.FindAllStringSubmatch(text, 3)
	for _, m := range matches {
		if len(m) < 4 {
			continue
		}
		linkGuildID, channelID, messageID := m[1], m[2], m[3]
		// Security: only expand links from the same guild
		if linkGuildID != guildID {
			continue
		}
		msg, err := s.ChannelMessage(channelID, messageID)
		if err != nil || msg == nil || msg.Content == "" {
			continue
		}
		author := "unknown"
		if msg.Author != nil {
			author = msg.Author.Username
		}
		text += fmt.Sprintf("\n[linked message from %s]: %s", author, msg.Content)
	}

	return text
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
