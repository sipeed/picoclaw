// Package mattermost implements a Mattermost channel for picoclaw.
//
// Uses WebSocket API v4 for receiving events and REST API v4 for sending
// messages and uploading files. Supports threading, typing indicators,
// message editing, and placeholder messages.
//
// No external Mattermost SDK — only gorilla/websocket (already a project
// dependency) and net/http.
package mattermost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	maxMessageLen = 4000             // Mattermost display limit
	wsReadTimeout = 90 * time.Second // read deadline for WebSocket
)

// Package-level compiled regexes to avoid recompilation per message.
var multiSpaceRe = regexp.MustCompile(`\s{2,}`)

// MattermostChannel connects to a Mattermost server via WebSocket + REST API v4.
// It supports threaded replies, typing indicators, message editing, placeholder
// messages, and file uploads.
type MattermostChannel struct {
	*channels.BaseChannel
	config      config.MattermostConfig
	baseURL     string // normalized base URL (no trailing slash)
	httpClient  *http.Client
	ws          *websocket.Conn
	wsMu        sync.Mutex
	botUserID   string
	botUsername string
	mentionRe   *regexp.Regexp // compiled at start, nil if no username
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewMattermostChannel creates a new Mattermost channel. Both URL and Token
// must be provided in the config.
func NewMattermostChannel(cfg config.MattermostConfig, messageBus *bus.MessageBus) (*MattermostChannel, error) {
	if cfg.URL == "" || cfg.Token == "" {
		return nil, fmt.Errorf("mattermost url and token are required")
	}

	parsed, err := url.Parse(cfg.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("mattermost url must be a valid http:// or https:// URL")
	}
	normalizedURL := parsed.Scheme + "://" + parsed.Host + strings.TrimRight(parsed.Path, "/")

	base := channels.NewBaseChannel("mattermost", cfg, messageBus, cfg.AllowFrom,
		channels.WithMaxMessageLength(maxMessageLen),
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &MattermostChannel{
		BaseChannel: base,
		config:      cfg,
		baseURL:     normalizedURL,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Start connects to Mattermost and begins listening for events.
func (c *MattermostChannel) Start(ctx context.Context) error {
	logger.InfoC("mattermost", "Starting Mattermost channel")

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Verify credentials and get bot info.
	me, err := c.apiGet(c.ctx, "/api/v4/users/me")
	if err != nil {
		c.cancel()
		return fmt.Errorf("mattermost auth failed: %w", err)
	}

	c.botUserID, _ = me["id"].(string)
	c.botUsername, _ = me["username"].(string)

	logger.InfoCF("mattermost", "Bot authenticated", map[string]any{
		"username": c.botUsername,
		"user_id":  c.botUserID,
	})

	// Compile mention regex now that we know the bot username.
	c.compileMentionPattern()

	// Connect WebSocket.
	if err := c.connectWS(); err != nil {
		c.cancel()
		return fmt.Errorf("mattermost websocket connect: %w", err)
	}

	// Spawn listener + reconnect loop.
	go c.listenLoop()

	c.SetRunning(true)
	logger.InfoC("mattermost", "Mattermost channel started")
	return nil
}

// Stop disconnects from Mattermost.
func (c *MattermostChannel) Stop(ctx context.Context) error {
	logger.InfoC("mattermost", "Stopping Mattermost channel")

	if c.cancel != nil {
		c.cancel()
	}

	c.wsMu.Lock()
	if c.ws != nil {
		c.ws.Close()
		c.ws = nil
	}
	c.wsMu.Unlock()

	c.SetRunning(false)
	logger.InfoC("mattermost", "Mattermost channel stopped")
	return nil
}

// Send sends a text message to Mattermost. The Manager handles message
// splitting via MaxMessageLength, so this sends a single chunk.
func (c *MattermostChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	channelID, rootID := parseChatID(msg.ChatID)
	if channelID == "" {
		return fmt.Errorf("%w: invalid mattermost chat ID: %s", channels.ErrSendFailed, msg.ChatID)
	}

	body := map[string]string{
		"channel_id": channelID,
		"message":    msg.Content,
	}
	if rootID != "" {
		body["root_id"] = rootID
	}

	if err := c.apiPost(ctx, "/api/v4/posts", body); err != nil {
		return err
	}

	logger.DebugCF("mattermost", "Message sent", map[string]any{
		"channel_id": channelID,
		"root_id":    rootID,
	})

	return nil
}

// SendMedia implements channels.MediaSender. It uploads files to Mattermost
// and attaches them to a post.
func (c *MattermostChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	channelID, rootID := parseChatID(msg.ChatID)
	if channelID == "" {
		return fmt.Errorf("%w: invalid mattermost chat ID: %s", channels.ErrSendFailed, msg.ChatID)
	}

	store := c.GetMediaStore()
	if store == nil {
		return fmt.Errorf("%w: no media store available", channels.ErrSendFailed)
	}

	var fileIDs []string
	var captions []string

	for _, part := range msg.Parts {
		localPath, err := store.Resolve(part.Ref)
		if err != nil {
			logger.ErrorCF("mattermost", "Failed to resolve media ref", map[string]any{
				"ref":   part.Ref,
				"error": err.Error(),
			})
			continue
		}

		filename := part.Filename
		if filename == "" {
			filename = "file"
		}

		fileID, err := c.uploadFile(ctx, channelID, localPath, filename)
		if err != nil {
			logger.ErrorCF("mattermost", "Failed to upload file", map[string]any{
				"filename": filename,
				"error":    err.Error(),
			})
			return fmt.Errorf("upload failed: %w", err)
		}

		fileIDs = append(fileIDs, fileID)
		if part.Caption != "" {
			captions = append(captions, part.Caption)
		}
	}

	if len(fileIDs) == 0 {
		return nil
	}

	// Create a post with the uploaded file IDs.
	postBody := map[string]any{
		"channel_id": channelID,
		"message":    strings.Join(captions, "\n"),
		"file_ids":   fileIDs,
	}
	if rootID != "" {
		postBody["root_id"] = rootID
	}

	return c.apiPost(ctx, "/api/v4/posts", postBody)
}

// StartTyping implements channels.TypingCapable. It sends a typing indicator
// and returns a stop function (no-op since Mattermost typing expires automatically).
func (c *MattermostChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	if !c.IsRunning() || !c.config.Typing.Enabled {
		return func() {}, nil
	}

	channelID, _ := parseChatID(chatID)
	if channelID == "" {
		return func() {}, nil
	}

	if err := c.apiPost(ctx, "/api/v4/users/me/typing", map[string]string{
		"channel_id": channelID,
	}); err != nil {
		logger.WarnCF("mattermost", "Failed to send typing indicator", map[string]any{
			"channel_id": channelID,
			"error":      err.Error(),
		})
	}

	return func() {}, nil
}

// EditMessage implements channels.MessageEditor. It updates the content of
// an existing Mattermost post.
func (c *MattermostChannel) EditMessage(ctx context.Context, chatID string, messageID string, content string) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	if messageID == "" {
		return fmt.Errorf("%w: empty message ID", channels.ErrSendFailed)
	}

	body := map[string]string{
		"id":      messageID,
		"message": content,
	}

	return c.apiPut(ctx, "/api/v4/posts/"+messageID+"/patch", body)
}

// SendPlaceholder implements channels.PlaceholderCapable. It sends a
// placeholder message that will later be edited with the actual response.
// The placeholder text is configurable via channels.mattermost.placeholder.text.
func (c *MattermostChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	if !c.IsRunning() {
		return "", channels.ErrNotRunning
	}

	if !c.config.Placeholder.Enabled {
		return "", nil
	}

	channelID, rootID := parseChatID(chatID)
	if channelID == "" {
		return "", fmt.Errorf("%w: invalid chat ID", channels.ErrSendFailed)
	}

	text := c.config.Placeholder.Text
	if text == "" {
		text = "Thinking... 💭"
	}

	body := map[string]string{
		"channel_id": channelID,
		"message":    text,
	}
	if rootID != "" {
		body["root_id"] = rootID
	}

	resp, err := c.apiPostJSON(ctx, "/api/v4/posts", body)
	if err != nil {
		return "", err
	}

	postID, _ := resp["id"].(string)
	return postID, nil
}

// -- WebSocket connection ---------------------------------------------------

func (c *MattermostChannel) connectWS() error {
	wsURL := c.buildWSURL()
	logger.InfoCF("mattermost", "Connecting WebSocket", map[string]any{"url": wsURL})

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	conn, resp, err := dialer.DialContext(c.ctx, wsURL, nil)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return err
	}

	// Set read deadline so ReadMessage doesn't block forever on half-open connections.
	// The server sends pings; gorilla auto-replies with pongs. We reset the read
	// deadline on each ping to keep the connection alive.
	conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	conn.SetPingHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
		// Write pong back (gorilla default behavior, but we need to do it
		// explicitly when overriding the ping handler).
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(5*time.Second))
	})

	// Authenticate via the WebSocket authentication challenge.
	authMsg := map[string]any{
		"seq":    1,
		"action": "authentication_challenge",
		"data":   map[string]string{"token": c.config.Token},
	}
	if err := conn.WriteJSON(authMsg); err != nil {
		conn.Close()
		return fmt.Errorf("ws auth: %w", err)
	}

	c.wsMu.Lock()
	c.ws = conn
	c.wsMu.Unlock()

	logger.InfoC("mattermost", "WebSocket connected")
	return nil
}

func (c *MattermostChannel) listenLoop() {
	backoff := 5 * time.Second
	maxBackoff := 60 * time.Second

	for {
		if c.ctx.Err() != nil {
			return
		}

		c.wsMu.Lock()
		ws := c.ws
		c.wsMu.Unlock()

		if ws == nil {
			logger.InfoC("mattermost", "Attempting WebSocket reconnect...")
			if err := c.connectWS(); err != nil {
				logger.ErrorCF("mattermost", "WebSocket reconnect failed", map[string]any{
					"error":   err.Error(),
					"backoff": backoff.String(),
				})
				select {
				case <-time.After(backoff):
				case <-c.ctx.Done():
					return
				}
				backoff = min(backoff*2, maxBackoff)
				continue
			}

			c.wsMu.Lock()
			ws = c.ws
			c.wsMu.Unlock()
		}

		// Reset read deadline on each successful read to keep the
		// connection alive as long as the server is sending events.
		ws.SetReadDeadline(time.Now().Add(wsReadTimeout))
		_, raw, err := ws.ReadMessage()
		if err != nil {
			if c.ctx.Err() != nil {
				return
			}
			logger.WarnCF("mattermost", "WebSocket read error", map[string]any{
				"error":   err.Error(),
				"backoff": backoff.String(),
			})
			c.wsMu.Lock()
			if c.ws != nil {
				c.ws.Close()
				c.ws = nil
			}
			c.wsMu.Unlock()
			// Apply backoff before reconnecting to avoid tight loops when
			// connections succeed but immediately drop (server restart, etc.).
			select {
			case <-time.After(backoff):
			case <-c.ctx.Done():
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// Successful read — reset backoff.
		backoff = 5 * time.Second

		c.handleWSMessage(raw)
	}
}

func (c *MattermostChannel) handleWSMessage(raw []byte) {
	var evt struct {
		Event string          `json:"event"`
		Data  json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &evt); err != nil {
		return
	}

	switch evt.Event {
	case "posted":
		c.handlePosted(evt.Data)
	case "hello":
		logger.InfoC("mattermost", "WebSocket hello (server ready)")
	case "":
		// Acknowledgement frame, ignore.
	default:
		logger.DebugCF("mattermost", "Unhandled WS event", map[string]any{
			"event": evt.Event,
		})
	}
}

func (c *MattermostChannel) handlePosted(data json.RawMessage) {
	var d struct {
		Post        string `json:"post"`
		ChannelType string `json:"channel_type"`
		SenderName  string `json:"sender_name"`
		TeamID      string `json:"team_id"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		logger.WarnC("mattermost", "Failed to parse posted event data")
		return
	}

	// The post field is double-encoded JSON.
	var post struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		UserID    string `json:"user_id"`
		ChannelID string `json:"channel_id"`
		RootID    string `json:"root_id"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal([]byte(d.Post), &post); err != nil {
		logger.WarnC("mattermost", "Failed to parse post JSON")
		return
	}

	// Ignore system-generated messages (joins, leaves, header changes, etc.).
	// Regular user messages have an empty type field.
	if post.Type != "" {
		return
	}

	// Ignore own messages.
	if post.UserID == c.botUserID {
		return
	}

	content := strings.TrimSpace(post.Message)

	// Detect whether the bot was @mentioned before stripping.
	isMentioned := c.hasBotMention(content)
	content = c.stripBotMention(content)

	if content == "" {
		return
	}

	sender := bus.SenderInfo{
		Platform:    "mattermost",
		PlatformID:  post.UserID,
		CanonicalID: identity.BuildCanonicalID("mattermost", post.UserID),
		Username:    d.SenderName,
	}

	if !c.IsAllowedSender(sender) {
		logger.DebugCF("mattermost", "Message rejected by allowlist", map[string]any{
			"user_id": post.UserID,
		})
		return
	}

	// "D" = direct message, "G" = group direct message; both bypass group-trigger
	// filtering and threading, matching the intended "DMs stay flat" behavior.
	isDM := d.ChannelType == "D" || d.ChannelType == "G"

	// In non-DM channels, apply group trigger filtering.
	if !isDM {
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, content)
		if !respond {
			return
		}
		content = cleaned
	}

	// Build chatID with threading context:
	//   DMs: just channelID (no threading)
	//   Existing thread: channelID/rootID (continue thread)
	//   Channel + reply_in_thread: channelID/postID (start new thread)
	//   Channel + no threading: just channelID (flat reply)
	chatID := post.ChannelID
	if post.RootID != "" {
		chatID = post.ChannelID + "/" + post.RootID
	} else if !isDM && c.config.ReplyInThread {
		chatID = post.ChannelID + "/" + post.ID
	}

	peerKind := "channel"
	peerID := post.ChannelID
	if isDM {
		peerKind = "direct"
		peerID = post.UserID
	}

	peer := bus.Peer{Kind: peerKind, ID: peerID}

	metadata := map[string]string{
		"post_id":      post.ID,
		"channel_id":   post.ChannelID,
		"root_id":      post.RootID,
		"channel_type": d.ChannelType,
		"sender_name":  d.SenderName,
		"team_id":      d.TeamID,
		"platform":     "mattermost",
	}

	logger.DebugCF("mattermost", "Received message", map[string]any{
		"sender":  d.SenderName,
		"chat_id": chatID,
		"preview": utils.Truncate(content, 50),
		"is_dm":   isDM,
	})

	c.HandleMessage(c.ctx, peer, post.ID, post.UserID, chatID, content, nil, metadata, sender)
}

// -- HTTP helpers -----------------------------------------------------------

// apiGet performs a GET request against the Mattermost REST API v4 and
// returns the parsed JSON response.
func (c *MattermostChannel) apiGet(ctx context.Context, path string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.Token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, channels.ClassifyNetError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, channels.ClassifySendError(resp.StatusCode, fmt.Errorf("reading error response: %w", readErr))
		}
		return nil, channels.ClassifySendError(resp.StatusCode, fmt.Errorf("%s", string(body)))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// apiDo performs an HTTP request with a JSON body and discards the response body.
// Errors are classified as ErrRateLimit, ErrTemporary, or ErrSendFailed
// based on the HTTP status code.
func (c *MattermostChannel) apiDo(ctx context.Context, method, path string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return channels.ClassifyNetError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return channels.ClassifySendError(resp.StatusCode, fmt.Errorf("reading error response: %w", readErr))
		}
		return channels.ClassifySendError(resp.StatusCode, fmt.Errorf("%s", string(respBody)))
	}
	return nil
}

// apiPost performs a POST request against the Mattermost REST API v4.
func (c *MattermostChannel) apiPost(ctx context.Context, path string, body any) error {
	return c.apiDo(ctx, "POST", path, body)
}

// apiPostJSON performs a POST and returns the parsed JSON response body.
func (c *MattermostChannel) apiPostJSON(ctx context.Context, path string, body any) (map[string]any, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, channels.ClassifyNetError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, channels.ClassifySendError(resp.StatusCode, fmt.Errorf("reading error response: %w", readErr))
		}
		return nil, channels.ClassifySendError(resp.StatusCode, fmt.Errorf("%s", string(respBody)))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// apiPut performs a PUT request against the Mattermost REST API v4.
func (c *MattermostChannel) apiPut(ctx context.Context, path string, body any) error {
	return c.apiDo(ctx, "PUT", path, body)
}

// uploadFile uploads a local file to Mattermost using streaming multipart
// upload and returns the file ID.
func (c *MattermostChannel) uploadFile(ctx context.Context, channelID, localPath, filename string) (string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("%w: %w", channels.ErrSendFailed, err)
	}
	defer file.Close()

	// Use io.Pipe for streaming multipart upload to avoid buffering large
	// files entirely in memory. We create the writer (for content type) and
	// request before starting the goroutine to avoid leaking it on early errors.
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	contentType := writer.FormDataContentType()

	uploadURL := c.baseURL + "/api/v4/files"
	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, pr)
	if err != nil {
		pw.Close()
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.Token)
	req.Header.Set("Content-Type", contentType)

	// Start the pipe writer goroutine only after the request is created.
	go func() {
		var werr error
		defer pw.Close()
		if werr = writer.WriteField("channel_id", channelID); werr != nil {
			pw.CloseWithError(werr)
			return
		}
		var part io.Writer
		if part, werr = writer.CreateFormFile("files", filename); werr != nil {
			pw.CloseWithError(werr)
			return
		}
		if _, werr = io.Copy(part, file); werr != nil {
			pw.CloseWithError(werr)
			return
		}
		if werr = writer.Close(); werr != nil {
			pw.CloseWithError(werr)
		}
	}()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Close the pipe reader so the writer goroutine unblocks and exits.
		pr.Close()
		return "", channels.ClassifyNetError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", channels.ClassifySendError(resp.StatusCode, fmt.Errorf("reading error response: %w", readErr))
		}
		return "", channels.ClassifySendError(resp.StatusCode, fmt.Errorf("%s", string(respBody)))
	}

	var result struct {
		FileInfos []struct {
			ID string `json:"id"`
		} `json:"file_infos"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.FileInfos) == 0 {
		return "", fmt.Errorf("%w: no file ID returned from upload", channels.ErrSendFailed)
	}
	return result.FileInfos[0].ID, nil
}

// -- Utility helpers --------------------------------------------------------

// buildWSURL converts the Mattermost server URL to a WebSocket URL
// (https → wss, http → ws) and appends the WebSocket API path.
// Preserves any base path prefix from the original URL.
func (c *MattermostChannel) buildWSURL() string {
	parsed, err := url.Parse(c.baseURL)
	if err != nil {
		return strings.Replace(c.baseURL, "https://", "wss://", 1) + "/api/v4/websocket"
	}
	scheme := "wss"
	if parsed.Scheme == "http" {
		scheme = "ws"
	}
	parsed.Scheme = scheme
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/api/v4/websocket"
	return parsed.String()
}

// compileMentionPattern builds and caches the bot mention regex. Called once
// after authentication when the bot username is known.
func (c *MattermostChannel) compileMentionPattern() {
	username := c.config.Username
	if username == "" {
		username = c.botUsername
	}
	if username == "" {
		return
	}
	// Match @username followed by a word boundary (not as a substring of
	// another word like @mybotany). Case-insensitive.
	c.mentionRe = regexp.MustCompile(`(?i)@` + regexp.QuoteMeta(username) + `\b`)
}

// hasBotMention checks whether the message text contains a whole-word @mention
// of the bot, before stripping. Used to pass isMentioned to ShouldRespondInGroup.
func (c *MattermostChannel) hasBotMention(text string) bool {
	if c.mentionRe == nil {
		return false
	}
	return c.mentionRe.MatchString(text)
}

// stripBotMention removes whole-word @botusername mentions from message text
// so the agent receives clean content without its own mention.
func (c *MattermostChannel) stripBotMention(text string) string {
	if c.mentionRe == nil {
		return text
	}
	text = c.mentionRe.ReplaceAllString(text, " ")
	text = multiSpaceRe.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// parseChatID splits a chat ID into channelID and optional rootID.
// Format: "channelID" or "channelID/rootID" for threaded conversations.
// SplitN(..., 2) is intentional: post IDs contain no slashes, so any extra
// slashes would be part of a malformed input and are folded into rootID.
func parseChatID(chatID string) (channelID, rootID string) {
	parts := strings.SplitN(chatID, "/", 2)
	channelID = parts[0]
	if len(parts) > 1 {
		rootID = parts[1]
	}
	return channelID, rootID
}
