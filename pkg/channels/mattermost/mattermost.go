package mattermost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	sendTimeout     = 10 * time.Second
	maxPostChars    = 4000
	typingIntervalS = 4 * time.Second
	typingTimeoutS  = 3 * time.Minute
	wsReconnectMax  = 60 * time.Second
	wsPingInterval  = 20 * time.Second
	wsPingTimeout   = 10 * time.Second
)

// MattermostChannel implements the PicoClaw Channel interface for Mattermost.
type MattermostChannel struct {
	*channels.BaseChannel
	config      config.MattermostConfig
	httpClient  *http.Client
	botUserID   string
	botUsername string
	ctx         context.Context
	cancel      context.CancelFunc
	typingMu    sync.Mutex
	typingStop  map[string]chan struct{}
}

func NewMattermostChannel(cfg config.MattermostConfig, messageBus *bus.MessageBus) (*MattermostChannel, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("mattermost url is required")
	}
	if cfg.BotToken.String() == "" {
		return nil, fmt.Errorf("mattermost bot_token is required")
	}

	base := channels.NewBaseChannel("mattermost", cfg, messageBus, cfg.AllowFrom,
		channels.WithMaxMessageLength(maxPostChars),
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &MattermostChannel{
		BaseChannel: base,
		config:      cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		typingStop: make(map[string]chan struct{}),
	}, nil
}

// --- Lifecycle ---

func (c *MattermostChannel) Start(ctx context.Context) error {
	logger.InfoC("mattermost", "Starting Mattermost channel")

	c.ctx, c.cancel = context.WithCancel(ctx)

	if err := c.fetchBotInfo(); err != nil {
		return fmt.Errorf("mattermost: failed to fetch bot info: %w", err)
	}

	go c.websocketLoop()

	c.SetRunning(true)
	logger.InfoCF("mattermost", "Mattermost bot connected", map[string]any{
		"username": c.botUsername,
		"user_id":  c.botUserID,
	})
	return nil
}

func (c *MattermostChannel) Stop(ctx context.Context) error {
	logger.InfoC("mattermost", "Stopping Mattermost channel")
	c.SetRunning(false)

	c.typingMu.Lock()
	for chatID, stop := range c.typingStop {
		close(stop)
		delete(c.typingStop, chatID)
	}
	c.typingMu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

// --- REST API helpers ---

func (c *MattermostChannel) apiURL(path string) string {
	return strings.TrimRight(c.config.URL, "/") + path
}

func (c *MattermostChannel) doJSON(method, path string, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(c.ctx, method, c.apiURL(path), reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.BotToken.String())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.httpClient.Do(req)
}

func (c *MattermostChannel) fetchBotInfo() error {
	resp, err := c.doJSON(http.MethodGet, "/api/v4/users/me", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET /users/me returned %d", resp.StatusCode)
	}
	var user struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return err
	}
	c.botUserID = user.ID
	c.botUsername = user.Username
	return nil
}

// --- WebSocket loop ---

func (c *MattermostChannel) websocketLoop() {
	wsURL := strings.Replace(c.config.URL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	wsURL = strings.TrimRight(wsURL, "/") + "/api/v4/websocket"

	reconnectDelay := time.Second

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		connected, err := c.runWebSocket(wsURL)
		if err != nil {
			if c.ctx.Err() != nil {
				return
			}
			logger.WarnCF("mattermost", "WebSocket error, reconnecting", map[string]any{
				"error": err.Error(),
				"delay": reconnectDelay.String(),
			})
		}
		if connected {
			// Reset backoff after at least one successful connection so future
			// disconnects recover quickly instead of inheriting stale high delays.
			reconnectDelay = time.Second
		}

		select {
		case <-c.ctx.Done():
			return
		case <-time.After(reconnectDelay):
		}
		reconnectDelay = min(reconnectDelay*2, wsReconnectMax)
	}
}

func (c *MattermostChannel) runWebSocket(wsURL string) (bool, error) {
	dialer := websocket.DefaultDialer
	conn, resp, err := dialer.DialContext(c.ctx, wsURL, nil)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return false, fmt.Errorf("dial: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	// Authenticate
	authMsg := map[string]any{
		"seq":    1,
		"action": "authentication_challenge",
		"data":   map[string]any{"token": c.config.BotToken.String()},
	}
	authData, _ := json.Marshal(authMsg)
	_ = conn.SetWriteDeadline(time.Now().Add(wsPingTimeout))
	if err := conn.WriteMessage(websocket.TextMessage, authData); err != nil {
		return false, fmt.Errorf("auth: %w", err)
	}

	logger.InfoC("mattermost", "WebSocket connected and authenticated")

	wsCtx, wsCancel := context.WithCancel(c.ctx)
	defer wsCancel()

	connCloseDone := make(chan struct{})
	defer close(connCloseDone)
	go func() {
		select {
		case <-wsCtx.Done():
			_ = conn.Close()
		case <-connCloseDone:
		}
	}()

	// Keepalive: detect half-open connections and force reconnect.
	pingErrCh := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(wsPingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-wsCtx.Done():
				return
			case <-ticker.C:
				err := conn.WriteControl(
					websocket.PingMessage,
					nil,
					time.Now().Add(wsPingTimeout),
				)
				if err != nil {
					select {
					case pingErrCh <- fmt.Errorf("ping: %w", err):
					default:
					}
					_ = conn.Close()
					return
				}
			}
		}
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if c.ctx.Err() != nil {
				return true, nil
			}
			select {
			case pingErr := <-pingErrCh:
				return true, pingErr
			default:
			}
			return true, fmt.Errorf("read: %w", err)
		}

		var event wsEvent
		if err := json.Unmarshal(data, &event); err != nil {
			continue
		}
		if event.Event == "posted" {
			go c.handlePostedEvent(event)
		}
	}
}

type wsEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type wsPostedData struct {
	Post        string `json:"post"`
	ChannelType string `json:"channel_type"`
}

type mmPost struct {
	ID        string   `json:"id"`
	ChannelID string   `json:"channel_id"`
	UserID    string   `json:"user_id"`
	RootID    string   `json:"root_id"`
	Message   string   `json:"message"`
	FileIDs   []string `json:"file_ids"`
	Metadata  struct {
		Files []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"files"`
	} `json:"metadata"`
}

// --- Message handler ---

func (c *MattermostChannel) handlePostedEvent(event wsEvent) {
	var posted wsPostedData
	if err := json.Unmarshal(event.Data, &posted); err != nil {
		return
	}

	var post mmPost
	if err := json.Unmarshal([]byte(posted.Post), &post); err != nil {
		return
	}

	// Ignore own messages
	if post.UserID == c.botUserID {
		return
	}

	isDM := posted.ChannelType == "D"

	// Build sender info for allowlist check
	sender := bus.SenderInfo{
		Platform:    "mattermost",
		PlatformID:  post.UserID,
		CanonicalID: identity.BuildCanonicalID("mattermost", post.UserID),
	}

	if !c.IsAllowedSender(sender) {
		logger.DebugCF("mattermost", "Message rejected by allowlist", map[string]any{
			"user_id": post.UserID,
		})
		return
	}

	content := post.Message

	// Group trigger filtering
	if !isDM {
		isMentioned := strings.Contains(
			strings.ToLower(content),
			"@"+strings.ToLower(c.botUsername),
		)
		content = c.stripBotMention(content)
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, content)
		if !respond {
			return
		}
		content = cleaned
	} else {
		content = c.stripBotMention(content)
	}

	// Determine chatID and peer
	peerKind := "channel"
	chatID := post.ChannelID
	if isDM {
		peerKind = "direct"
	}
	peer := bus.Peer{Kind: peerKind, ID: chatID}

	// Thread root_id: for replies, use existing root_id; for new channel messages, use post.ID
	rootID := post.RootID
	if !isDM && rootID == "" {
		rootID = post.ID
	}

	// Download attachments
	mediaPaths := make([]string, 0, len(post.FileIDs))
	scope := channels.BuildMediaScope("mattermost", chatID, post.ID)

	hintMap := make(map[string]string)
	for _, fi := range post.Metadata.Files {
		if fi.ID != "" {
			hintMap[fi.ID] = fi.Name
		}
	}

	for _, fid := range post.FileIDs {
		filename := hintMap[fid]
		localPath := c.downloadFile(fid, filename)
		if localPath == "" {
			continue
		}
		if store := c.GetMediaStore(); store != nil {
			ref, err := store.Store(localPath, media.MediaMeta{
				Filename: filename,
				Source:   "mattermost",
			}, scope)
			if err == nil {
				mediaPaths = append(mediaPaths, ref)
				continue
			}
		}
		mediaPaths = append(mediaPaths, localPath)
	}

	if content == "" && len(mediaPaths) == 0 {
		return
	}
	if content == "" {
		content = "[media only]"
	}

	metadata := map[string]string{
		"user_id":      post.UserID,
		"channel_id":   post.ChannelID,
		"channel_type": posted.ChannelType,
		"post_id":      post.ID,
		"root_id":      rootID,
		"is_dm":        fmt.Sprintf("%t", isDM),
	}

	logger.DebugCF("mattermost", "Received message", map[string]any{
		"sender_id": post.UserID,
		"preview":   utils.Truncate(content, 50),
		"is_dm":     isDM,
	})

	c.HandleMessage(c.ctx, peer, post.ID, post.UserID, chatID, content, mediaPaths, metadata, sender)
}

func (c *MattermostChannel) stripBotMention(text string) string {
	if c.botUsername == "" {
		return text
	}
	// Case-insensitive replace of @botname
	lower := strings.ToLower(text)
	mention := "@" + strings.ToLower(c.botUsername)
	for {
		idx := strings.Index(lower, mention)
		if idx < 0 {
			break
		}
		text = text[:idx] + text[idx+len(mention):]
		lower = strings.ToLower(text)
	}
	return strings.TrimSpace(text)
}

// --- Send ---

func (c *MattermostChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	if msg.ChatID == "" {
		return fmt.Errorf("mattermost: chatID is empty")
	}
	if len([]rune(msg.Content)) == 0 {
		return nil
	}

	rootID := msg.ReplyToMessageID
	return c.postMessage(msg.ChatID, msg.Content, rootID)
}

func (c *MattermostChannel) postMessage(channelID, text, rootID string) error {
	payload := map[string]any{
		"channel_id": channelID,
		"message":    text,
	}
	if rootID != "" {
		payload["root_id"] = rootID
	}
	resp, err := c.doJSON(http.MethodPost, "/api/v4/posts", payload)
	if err != nil {
		return fmt.Errorf("mattermost post: %w", channels.ErrTemporary)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		logger.ErrorCF("mattermost", "Post failed", map[string]any{
			"status": resp.StatusCode,
			"body":   string(body),
		})
		return fmt.Errorf("mattermost post %d: %w", resp.StatusCode, channels.ErrTemporary)
	}
	return nil
}

// --- EditMessage (MessageEditor) ---

func (c *MattermostChannel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	payload := map[string]any{
		"id":      messageID,
		"message": content,
	}
	resp, err := c.doJSON(http.MethodPut, "/api/v4/posts/"+messageID, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mattermost edit %d", resp.StatusCode)
	}
	return nil
}

// --- SendPlaceholder (PlaceholderCapable) ---

func (c *MattermostChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	if !c.config.Placeholder.Enabled {
		return "", nil
	}
	payload := map[string]any{
		"channel_id": chatID,
		"message":    c.config.Placeholder.GetRandomText(),
	}
	resp, err := c.doJSON(http.MethodPost, "/api/v4/posts", payload)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("placeholder post %d", resp.StatusCode)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.ID, nil
}

// --- Typing (TypingCapable) ---

func (c *MattermostChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	if !c.config.Typing.Enabled {
		return func() {}, nil
	}
	c.startTypingLoop(chatID)
	return func() { c.stopTypingLoop(chatID) }, nil
}

func (c *MattermostChannel) startTypingLoop(chatID string) {
	c.typingMu.Lock()
	if stop, ok := c.typingStop[chatID]; ok {
		close(stop)
	}
	stop := make(chan struct{})
	c.typingStop[chatID] = stop
	c.typingMu.Unlock()

	go func() {
		c.sendTyping(chatID)
		ticker := time.NewTicker(typingIntervalS)
		defer ticker.Stop()
		timeout := time.After(typingTimeoutS)
		for {
			select {
			case <-stop:
				return
			case <-timeout:
				return
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				c.sendTyping(chatID)
			}
		}
	}()
}

func (c *MattermostChannel) stopTypingLoop(chatID string) {
	c.typingMu.Lock()
	defer c.typingMu.Unlock()
	if stop, ok := c.typingStop[chatID]; ok {
		close(stop)
		delete(c.typingStop, chatID)
	}
}

func (c *MattermostChannel) sendTyping(chatID string) {
	payload := map[string]any{"channel_id": chatID}
	resp, err := c.doJSON(http.MethodPost, "/api/v4/users/"+c.botUserID+"/typing", payload)
	if err != nil {
		logger.DebugCF("mattermost", "Typing indicator failed", map[string]any{"error": err.Error()})
		return
	}
	if resp == nil || resp.Body == nil {
		return
	}
	defer resp.Body.Close()
}

// --- SendMedia (MediaSender) ---

func (c *MattermostChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	if msg.ChatID == "" {
		return fmt.Errorf("mattermost: chatID is empty")
	}

	store := c.GetMediaStore()
	if store == nil {
		return fmt.Errorf("no media store: %w", channels.ErrSendFailed)
	}

	for _, part := range msg.Parts {
		localPath, err := store.Resolve(part.Ref)
		if err != nil {
			logger.ErrorCF("mattermost", "Failed to resolve media ref", map[string]any{
				"ref":   part.Ref,
				"error": err.Error(),
			})
			continue
		}

		fileID, err := c.uploadFile(msg.ChatID, localPath, part.Filename)
		if err != nil {
			logger.ErrorCF("mattermost", "Upload failed", map[string]any{
				"path":  localPath,
				"error": err.Error(),
			})
			// Fallback: send path as text
			_ = c.postMessage(msg.ChatID, fmt.Sprintf("[Attachment: %s]", part.Filename), "")
			continue
		}

		payload := map[string]any{
			"channel_id": msg.ChatID,
			"message":    part.Caption,
			"file_ids":   []string{fileID},
		}
		resp, err := c.doJSON(http.MethodPost, "/api/v4/posts", payload)
		if err != nil {
			return fmt.Errorf("mattermost media post: %w", channels.ErrTemporary)
		}
		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
			resp.Body.Close()
			logger.ErrorCF("mattermost", "Media post failed", map[string]any{
				"status": resp.StatusCode,
				"body":   string(body),
			})
			return fmt.Errorf("mattermost media post %d: %w", resp.StatusCode, channels.ErrTemporary)
		}
		resp.Body.Close()
	}
	return nil
}

// --- File helpers ---

func (c *MattermostChannel) downloadFile(fileID, filename string) string {
	url := c.apiURL("/api/v4/files/" + fileID)
	return utils.DownloadFile(url, filename, utils.DownloadOptions{
		LoggerPrefix: "mattermost",
		ExtraHeaders: map[string]string{"Authorization": "Bearer " + c.config.BotToken.String()},
	})
}

func (c *MattermostChannel) uploadFile(channelID, localPath, filename string) (string, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if filename == "" {
		filename = filepath.Base(localPath)
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("channel_id", channelID)
	part, err := w.CreateFormFile("files", filename)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, f)
	if err != nil {
		return "", err
	}
	w.Close()

	req, err := http.NewRequestWithContext(c.ctx, http.MethodPost, c.apiURL("/api/v4/files"), &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.BotToken.String())
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("upload returned %d", resp.StatusCode)
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
		return "", fmt.Errorf("upload returned no file_infos")
	}
	return result.FileInfos[0].ID, nil
}
