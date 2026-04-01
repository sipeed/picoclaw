package matrix

import (
	"context"
	"database/sql"
	"fmt"
	"html"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gomarkdown/markdown"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"go.mau.fi/util/dbutil"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	_ "modernc.org/sqlite"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
)

const (
	sqliteDriver = "sqlite"
	dbName       = "store.db"

	typingRefreshInterval      = 20 * time.Second
	typingServerTTL            = 30 * time.Second
	roomKindCacheTTL           = 5 * time.Minute
	roomKindCacheCleanupPeriod = 1 * time.Minute
	roomKindCacheMaxEntries    = 2048
)

var matrixMentionHrefRegexp = regexp.MustCompile(`(?i)<a[^>]+href=["']([^"']+)["']`)

type roomKindCacheEntry struct {
	isGroup   bool
	expiresAt time.Time
	touchedAt time.Time
}

type roomKindCache struct {
	mu         sync.Mutex
	entries    map[string]roomKindCacheEntry
	maxEntries int
	ttl        time.Duration
}

func newRoomKindCache(maxEntries int, ttl time.Duration) *roomKindCache {
	if maxEntries <= 0 {
		maxEntries = roomKindCacheMaxEntries
	}
	if ttl <= 0 {
		ttl = roomKindCacheTTL
	}

	return &roomKindCache{
		entries:    make(map[string]roomKindCacheEntry),
		maxEntries: maxEntries,
		ttl:        ttl,
	}
}

func (c *roomKindCache) get(roomID string, now time.Time) (bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[roomID]
	if !ok {
		return false, false
	}
	if !entry.expiresAt.After(now) {
		delete(c.entries, roomID)
		return false, false
	}

	return entry.isGroup, true
}

func (c *roomKindCache) set(roomID string, isGroup bool, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.entries[roomID]; ok {
		entry.isGroup = isGroup
		entry.expiresAt = now.Add(c.ttl)
		entry.touchedAt = now
		c.entries[roomID] = entry
		return
	}

	c.cleanupExpiredLocked(now)
	for len(c.entries) >= c.maxEntries {
		if !c.evictOldestLocked() {
			break
		}
	}

	c.entries[roomID] = roomKindCacheEntry{
		isGroup:   isGroup,
		expiresAt: now.Add(c.ttl),
		touchedAt: now,
	}
}

func (c *roomKindCache) cleanupExpired(now time.Time) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cleanupExpiredLocked(now)
}

func (c *roomKindCache) cleanupExpiredLocked(now time.Time) int {
	removed := 0
	for roomID, entry := range c.entries {
		if !entry.expiresAt.After(now) {
			delete(c.entries, roomID)
			removed++
		}
	}
	return removed
}

func (c *roomKindCache) evictOldestLocked() bool {
	if len(c.entries) == 0 {
		return false
	}

	var (
		oldestRoomID string
		oldestAt     time.Time
	)

	for roomID, entry := range c.entries {
		if oldestRoomID == "" || entry.touchedAt.Before(oldestAt) {
			oldestRoomID = roomID
			oldestAt = entry.touchedAt
		}
	}

	delete(c.entries, oldestRoomID)
	return true
}

type typingSession struct {
	stopCh chan struct{}
	once   sync.Once
}

func newTypingSession() *typingSession {
	return &typingSession{
		stopCh: make(chan struct{}),
	}
}

func (s *typingSession) stop() {
	s.once.Do(func() {
		close(s.stopCh)
	})
}

// MatrixChannel implements the Channel interface for Matrix.
type MatrixChannel struct {
	*channels.BaseChannel

	client *mautrix.Client
	config config.MatrixConfig
	syncer *mautrix.DefaultSyncer

	ctx       context.Context
	cancel    context.CancelFunc
	startTime time.Time

	typingMu       sync.Mutex
	typingSessions map[string]*typingSession // roomID -> session

	roomKindCache     *roomKindCache
	localpartMentionR *regexp.Regexp

	cryptoHelper *cryptohelper.CryptoHelper
	cryptoDbPath string
}

func NewMatrixChannel(
	cfg config.MatrixConfig,
	messageBus *bus.MessageBus,
	cryptoDatabasePath string,
) (*MatrixChannel, error) {
	homeserver := strings.TrimSpace(cfg.Homeserver)
	userID := strings.TrimSpace(cfg.UserID)
	accessToken := strings.TrimSpace(cfg.AccessToken.String())
	if homeserver == "" {
		return nil, fmt.Errorf("matrix homeserver is required")
	}
	if userID == "" {
		return nil, fmt.Errorf("matrix user_id is required")
	}
	if accessToken == "" {
		return nil, fmt.Errorf("matrix access_token is required")
	}

	client, err := mautrix.NewClient(homeserver, id.UserID(userID), accessToken)
	if err != nil {
		return nil, fmt.Errorf("create matrix client: %w", err)
	}
	if cfg.DeviceID != "" {
		client.DeviceID = id.DeviceID(cfg.DeviceID)
	}

	syncer, ok := client.Syncer.(*mautrix.DefaultSyncer)
	if !ok {
		return nil, fmt.Errorf("matrix syncer is not *mautrix.DefaultSyncer")
	}

	base := channels.NewBaseChannel(
		"matrix",
		cfg,
		messageBus,
		cfg.AllowFrom,
		channels.WithMaxMessageLength(65536),
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &MatrixChannel{
		BaseChannel:       base,
		client:            client,
		config:            cfg,
		syncer:            syncer,
		typingSessions:    make(map[string]*typingSession),
		startTime:         time.Now(),
		roomKindCache:     newRoomKindCache(roomKindCacheMaxEntries, roomKindCacheTTL),
		localpartMentionR: localpartMentionRegexp(matrixLocalpart(client.UserID)),
		typingMu:          sync.Mutex{},
		cryptoDbPath:      cryptoDatabasePath,
	}, nil
}

func (c *MatrixChannel) Start(ctx context.Context) error {
	logger.InfoC("matrix", "Starting Matrix channel")

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.startTime = time.Now()

	// Initialize crypto helper if database and passphrase are configured
	if c.cryptoDbPath != "" && c.config.CryptoPassphrase != "" {
		if err := c.initCrypto(ctx); err != nil {
			logger.WarnCF(
				"matrix",
				"Failed to initialize crypto, continuing without encryption support",
				map[string]any{
					"error": err.Error(),
				},
			)
		}
	}

	c.syncer.OnEventType(event.EventMessage, c.handleMessageEvent)
	c.syncer.OnEventType(event.EventEncrypted, c.handleMessageEvent)
	c.syncer.OnEventType(event.StateMember, c.handleMemberEvent)

	c.SetRunning(true)
	go c.runRoomKindCacheJanitor(c.ctx)

	go func() {
		if err := c.client.SyncWithContext(c.ctx); err != nil && c.ctx.Err() == nil {
			logger.ErrorCF("matrix", "Matrix sync stopped unexpectedly", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	logger.InfoC("matrix", "Matrix channel started")
	return nil
}

func (c *MatrixChannel) Stop(ctx context.Context) error {
	logger.InfoC("matrix", "Stopping Matrix channel")
	c.SetRunning(false)

	if c.cancel != nil {
		c.cancel()
	}
	c.stopTypingSessions(ctx)

	// Close crypto helper if initialized
	if c.cryptoHelper != nil {
		c.cryptoHelper.Close()
		c.cryptoHelper = nil
		c.client.Crypto = nil
	}

	logger.InfoC("matrix", "Matrix channel stopped")
	return nil
}

func (c *MatrixChannel) initCrypto(ctx context.Context) error {
	logger.InfoC("matrix", "Initializing crypto helper")

	// Ensure the crypto database directory exists
	if err := os.MkdirAll(c.cryptoDbPath, 0o700); err != nil {
		return fmt.Errorf("create crypto database directory: %w", err)
	}

	// Create database with sqlite driver (modernc.org/sqlite)
	dbPath := filepath.Join(c.cryptoDbPath, dbName)
	connStr := "file:" + dbPath + "?_foreign_keys=on"

	db, err := sql.Open(sqliteDriver, connStr)
	if err != nil {
		return fmt.Errorf("open crypto database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Execute PRAGMA statements
	// This is equivalent to the "sqlite3-fk-wal" dialect used by cryptohelper
	pragmaStmts := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 5000",
	}
	for _, pragma := range pragmaStmts {
		if _, err = db.ExecContext(ctx, pragma); err != nil {
			_ = db.Close()
			return fmt.Errorf("execute %s: %w", pragma, err)
		}
	}

	// Wrap with dbutil for dialect support
	wrappedDB, err := dbutil.NewWithDB(db, sqliteDriver)
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("wrap database: %w", err)
	}

	cryptoHelper, err := cryptohelper.NewCryptoHelper(c.client, []byte(c.config.CryptoPassphrase), wrappedDB)
	if err != nil {
		return fmt.Errorf("create crypto helper: %w", err)
	}

	if c.client.DeviceID == "" {
		resp, whoamiErr := c.client.Whoami(ctx)
		if whoamiErr != nil {
			_ = db.Close()
			return fmt.Errorf("get device ID via whoami: %w", whoamiErr)
		}
		c.client.DeviceID = resp.DeviceID
	}

	if err = cryptoHelper.Init(ctx); err != nil {
		cryptoHelper.Close()
		return fmt.Errorf("init crypto helper: %w", err)
	}

	c.client.Crypto = cryptoHelper
	c.cryptoHelper = cryptoHelper

	logger.InfoC("matrix", "Crypto helper initialized successfully")
	return nil
}

func markdownToHTML(md string) string {
	extensions := (parser.CommonExtensions | parser.NoEmptyLineBeforeBlock) &^ parser.DefinitionLists
	p := parser.NewWithExtensions(extensions)
	renderer := mdhtml.NewRenderer(mdhtml.RendererOptions{Flags: mdhtml.UseXHTML})
	return strings.TrimSpace(string(markdown.ToHTML([]byte(md), p, renderer)))
}

func (c *MatrixChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	roomID := id.RoomID(strings.TrimSpace(msg.ChatID))
	if roomID == "" {
		return nil, fmt.Errorf("matrix room ID is empty: %w", channels.ErrSendFailed)
	}

	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return nil, nil
	}

	resp, err := c.client.SendMessageEvent(ctx, roomID, event.EventMessage, c.messageContent(content))
	if err != nil {
		return nil, fmt.Errorf("matrix send: %w", channels.ErrTemporary)
	}
	return []string{resp.EventID.String()}, nil
}

func (c *MatrixChannel) messageContent(text string) *event.MessageEventContent {
	mc := &event.MessageEventContent{MsgType: event.MsgText, Body: text}
	if c.config.MessageFormat != "plain" {
		mc.Format = event.FormatHTML
		mc.FormattedBody = markdownToHTML(text)
	}
	return mc
}

// SendMedia implements channels.MediaSender.
func (c *MatrixChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}
	sendCtx := ctx
	if sendCtx == nil {
		sendCtx = context.Background()
	}

	roomID := id.RoomID(strings.TrimSpace(msg.ChatID))
	if roomID == "" {
		return nil, fmt.Errorf("matrix room ID is empty: %w", channels.ErrSendFailed)
	}

	store := c.GetMediaStore()
	if store == nil {
		return nil, fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	var eventIDs []string
	for _, part := range msg.Parts {
		if err := sendCtx.Err(); err != nil {
			return nil, err
		}

		localPath, meta, err := store.ResolveWithMeta(part.Ref)
		if err != nil {
			logger.ErrorCF("matrix", "Failed to resolve media ref", map[string]any{
				"ref":   part.Ref,
				"error": err.Error(),
			})
			continue
		}

		fileInfo, err := os.Stat(localPath)
		if err != nil {
			logger.ErrorCF("matrix", "Failed to stat media file", map[string]any{
				"path":  localPath,
				"error": err.Error(),
			})
			continue
		}

		file, err := os.Open(localPath)
		if err != nil {
			logger.ErrorCF("matrix", "Failed to open media file", map[string]any{
				"path":  localPath,
				"error": err.Error(),
			})
			continue
		}

		filename := strings.TrimSpace(part.Filename)
		if filename == "" {
			filename = strings.TrimSpace(meta.Filename)
		}
		if filename == "" {
			filename = filepath.Base(localPath)
		}
		if filename == "" {
			filename = "file"
		}

		contentType := strings.TrimSpace(part.ContentType)
		if contentType == "" {
			contentType = strings.TrimSpace(meta.ContentType)
		}
		if contentType == "" {
			contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
		}
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		uploadResp, err := c.client.UploadMedia(sendCtx, mautrix.ReqUploadMedia{
			Content:       file,
			ContentLength: fileInfo.Size(),
			ContentType:   contentType,
			FileName:      filename,
		})
		file.Close()
		if err != nil {
			logger.ErrorCF("matrix", "Failed to upload media", map[string]any{
				"path":  localPath,
				"type":  part.Type,
				"error": err.Error(),
			})
			return nil, fmt.Errorf("matrix upload media: %w", channels.ErrTemporary)
		}

		msgType := matrixOutboundMsgType(part.Type, filename, contentType)
		content := matrixOutboundContent(
			part.Caption,
			filename,
			msgType,
			contentType,
			fileInfo.Size(),
			uploadResp.ContentURI.CUString(),
		)

		sendResp, err := c.client.SendMessageEvent(sendCtx, roomID, event.EventMessage, content)
		if err != nil {
			logger.ErrorCF("matrix", "Failed to send media message", map[string]any{
				"room_id": roomID.String(),
				"type":    msgType,
				"error":   err.Error(),
			})
			return nil, fmt.Errorf("matrix send media: %w", channels.ErrTemporary)
		}
		if sendResp != nil {
			eventIDs = append(eventIDs, sendResp.EventID.String())
		}
	}

	return eventIDs, nil
}

// StartTyping implements channels.TypingCapable.
func (c *MatrixChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	if !c.IsRunning() {
		return func() {}, nil
	}

	roomID := id.RoomID(strings.TrimSpace(chatID))
	if roomID == "" {
		return func() {}, fmt.Errorf("matrix room ID is empty")
	}

	session := newTypingSession()

	c.typingMu.Lock()
	if prev := c.typingSessions[chatID]; prev != nil {
		prev.stop()
	}
	c.typingSessions[chatID] = session
	c.typingMu.Unlock()

	parent := c.baseContext()
	go c.typingLoop(parent, roomID, session)

	var once sync.Once
	stop := func() {
		once.Do(func() {
			session.stop()
			c.typingMu.Lock()
			if current := c.typingSessions[chatID]; current == session {
				delete(c.typingSessions, chatID)
			}
			c.typingMu.Unlock()
			_, _ = c.client.UserTyping(context.Background(), roomID, false, 0)
		})
	}

	return stop, nil
}

// SendPlaceholder implements channels.PlaceholderCapable.
func (c *MatrixChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	if !c.config.Placeholder.Enabled {
		return "", nil
	}

	roomID := id.RoomID(strings.TrimSpace(chatID))
	if roomID == "" {
		return "", fmt.Errorf("matrix room ID is empty")
	}

	text := c.config.Placeholder.GetRandomText()

	resp, err := c.client.SendMessageEvent(ctx, roomID, event.EventMessage, &event.MessageEventContent{
		MsgType: event.MsgNotice,
		Body:    text,
	})
	if err != nil {
		return "", err
	}

	return resp.EventID.String(), nil
}

// EditMessage implements channels.MessageEditor.
func (c *MatrixChannel) EditMessage(ctx context.Context, chatID string, messageID string, content string) error {
	roomID := id.RoomID(strings.TrimSpace(chatID))
	if roomID == "" {
		return fmt.Errorf("matrix room ID is empty")
	}
	if strings.TrimSpace(messageID) == "" {
		return fmt.Errorf("matrix message ID is empty")
	}

	editContent := c.messageContent(content)
	editContent.SetEdit(id.EventID(messageID))

	_, err := c.client.SendMessageEvent(ctx, roomID, event.EventMessage, editContent)
	return err
}

func (c *MatrixChannel) handleMemberEvent(ctx context.Context, evt *event.Event) {
	if !c.config.JoinOnInvite {
		return
	}
	if evt == nil {
		return
	}

	member := evt.Content.AsMember()
	if member.Membership != event.MembershipInvite {
		return
	}
	if evt.GetStateKey() != c.client.UserID.String() {
		return
	}

	_, err := c.client.JoinRoomByID(c.baseContext(), evt.RoomID)
	if err != nil {
		logger.WarnCF("matrix", "Failed to auto-join invited room", map[string]any{
			"room_id": evt.RoomID.String(),
			"error":   err.Error(),
		})
		return
	}

	logger.InfoCF("matrix", "Joined room after invite", map[string]any{
		"room_id": evt.RoomID.String(),
	})
}

func (c *MatrixChannel) handleMessageEvent(ctx context.Context, evt *event.Event) {
	if evt == nil {
		return
	}

	// Ignore our own messages.
	if evt.Sender == c.client.UserID {
		return
	}

	// Ignore historical events on first sync.
	if time.UnixMilli(evt.Timestamp).Before(c.startTime) {
		return
	}

	var msgEvt *event.MessageEventContent
	switch evt.Type {
	case event.EventMessage:
		// When crypto is enabled, events marked WasEncrypted=true are
		// re-dispatched by c.cryptoHelper after decryption and will be
		// processed again in the EventEncrypted branch. Skip to avoid duplication.
		if c.client.Crypto != nil && evt.Mautrix.WasEncrypted {
			return
		}

		msgEvt = evt.Content.AsMessage()
		if msgEvt == nil || msgEvt.MsgType == "" {
			return
		}
	case event.EventEncrypted:
		var ok bool
		msgEvt, ok = c.decryptEvent(ctx, evt)
		if !ok {
			return
		}
	}

	// Ignore edits.
	if msgEvt.RelatesTo != nil && msgEvt.RelatesTo.GetReplaceID() != "" {
		return
	}

	roomID := evt.RoomID.String()
	scope := channels.BuildMediaScope("matrix", roomID, evt.ID.String())

	content, mediaPaths, ok := c.extractInboundContent(ctx, msgEvt, scope)
	if !ok {
		return
	}
	content = strings.TrimSpace(content)
	if content == "" && len(mediaPaths) == 0 {
		return
	}

	senderID := evt.Sender.String()
	sender := bus.SenderInfo{
		Platform:    "matrix",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("matrix", senderID),
		Username:    senderID,
		DisplayName: senderID,
	}

	if !c.IsAllowedSender(sender) {
		logger.DebugCF("matrix", "Message rejected by allowlist", map[string]any{
			"sender_id": senderID,
		})
		return
	}

	isGroup := c.isGroupRoom(ctx, evt.RoomID)
	if isGroup {
		isMentioned := c.isBotMentioned(msgEvt)
		if isMentioned {
			content = c.stripSelfMention(content)
		}
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, content)
		if !respond {
			logger.DebugCF("matrix", "Ignoring group message by trigger rules", map[string]any{
				"room_id":      roomID,
				"is_mentioned": isMentioned,
				"mention_only": c.config.GroupTrigger.MentionOnly,
				"prefixes":     c.config.GroupTrigger.Prefixes,
			})
			return
		}
		content = cleaned
	} else {
		content = c.stripSelfMention(content)
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return
	}

	peerKind := "direct"
	peerID := senderID
	if isGroup {
		peerKind = "group"
		peerID = roomID
	}

	metadata := map[string]string{
		"room_id":    roomID,
		"timestamp":  fmt.Sprintf("%d", evt.Timestamp),
		"is_group":   fmt.Sprintf("%t", isGroup),
		"sender_raw": senderID,
	}
	if replyTo := msgEvt.GetRelatesTo().GetReplyTo(); replyTo != "" {
		metadata["reply_to_msg_id"] = replyTo.String()
	}

	c.HandleMessage(
		c.baseContext(),
		bus.Peer{Kind: peerKind, ID: peerID},
		evt.ID.String(),
		senderID,
		roomID,
		content,
		mediaPaths,
		metadata,
		sender,
	)
}

// decryptEvent decrypts an encrypted event and returns the decrypted message event content.
// It returns the decrypted content and a boolean indicating whether decryption was successful.
func (c *MatrixChannel) decryptEvent(ctx context.Context, evt *event.Event) (*event.MessageEventContent, bool) {
	if c.client.Crypto == nil {
		logger.DebugCF("matrix", "Received encrypted message but crypto is not enabled", map[string]any{
			"room_id": evt.RoomID.String(),
		})
		return nil, false
	}

	decrypted, err := c.client.Crypto.Decrypt(ctx, evt)
	if err != nil {
		logger.WarnCF("matrix", "Failed to decrypt message", map[string]any{
			"room_id": evt.RoomID.String(),
			"error":   err.Error(),
		})
		return nil, false
	}

	if decrypted.Type != event.EventMessage {
		logger.DebugCF("matrix", "Decrypted event is not a message event", map[string]any{
			"room_id": evt.RoomID.String(),
			"type":    decrypted.Type.String(),
		})
		return nil, false
	}

	return decrypted.Content.AsMessage(), true
}

func (c *MatrixChannel) extractInboundContent(
	ctx context.Context,
	msgEvt *event.MessageEventContent,
	scope string,
) (string, []string, bool) {
	switch msgEvt.MsgType {
	case event.MsgText, event.MsgNotice:
		return msgEvt.Body, nil, true
	case event.MsgImage, event.MsgAudio, event.MsgVideo, event.MsgFile:
		return c.extractInboundMedia(ctx, msgEvt, scope)
	default:
		logger.DebugCF("matrix", "Ignoring unsupported matrix msgtype", map[string]any{
			"msgtype": msgEvt.MsgType,
		})
		return "", nil, false
	}
}

func (c *MatrixChannel) extractInboundMedia(
	ctx context.Context,
	msgEvt *event.MessageEventContent,
	scope string,
) (string, []string, bool) {
	mediaKind := matrixMediaKind(msgEvt.MsgType)
	label := matrixMediaLabel(msgEvt, mediaKind)
	content := fmt.Sprintf("[%s: %s]", mediaKind, label)
	if caption := strings.TrimSpace(msgEvt.GetCaption()); caption != "" {
		content = caption + "\n" + content
	}

	localPath, err := c.downloadMedia(ctx, msgEvt, mediaKind)
	if err != nil {
		logger.WarnCF("matrix", "Failed to download media; forwarding as text-only marker", map[string]any{
			"msgtype": msgEvt.MsgType,
			"error":   err.Error(),
		})
		return content, nil, true
	}

	filename := matrixMediaFilename(label, mediaKind, matrixContentType(msgEvt))
	ref := c.storeMedia(localPath, media.MediaMeta{
		Filename:    filename,
		ContentType: matrixContentType(msgEvt),
		Source:      "matrix",
	}, scope)
	return content, []string{ref}, true
}

func (c *MatrixChannel) storeMedia(localPath string, meta media.MediaMeta, scope string) string {
	if store := c.GetMediaStore(); store != nil {
		if meta.CleanupPolicy == "" {
			meta.CleanupPolicy = media.CleanupPolicyDeleteOnCleanup
		}
		ref, err := store.Store(localPath, meta, scope)
		if err == nil {
			return ref
		}
		logger.WarnCF("matrix", "Failed to store media in MediaStore, falling back to local path", map[string]any{
			"path":  localPath,
			"error": err.Error(),
		})
	}
	return localPath
}

func (c *MatrixChannel) downloadMedia(
	ctx context.Context,
	msgEvt *event.MessageEventContent,
	mediaKind string,
) (string, error) {
	uri := matrixMediaURI(msgEvt)
	if uri == "" {
		return "", fmt.Errorf("empty matrix media URL")
	}
	parsed := uri.ParseOrIgnore()
	if parsed.IsEmpty() {
		return "", fmt.Errorf("invalid matrix media URL: %s", uri)
	}

	dlCtx := c.baseContext()
	if ctx != nil {
		dlCtx = ctx
	}
	reqCtx, cancel := context.WithTimeout(dlCtx, 20*time.Second)
	defer cancel()

	resp, err := c.client.Download(reqCtx, parsed)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	reader := resp.Body
	readerClose := func() error { return nil }

	// Encrypted attachments put URL in msgEvt.File and require client-side decryption.
	if msgEvt != nil && msgEvt.File != nil && msgEvt.URL == "" {
		if err = msgEvt.File.PrepareForDecryption(); err != nil {
			return "", fmt.Errorf("decrypt matrix media: %w", err)
		}
		decryptReader := msgEvt.File.DecryptStream(resp.Body)
		reader = decryptReader
		readerClose = decryptReader.Close
	}

	label := matrixMediaLabel(msgEvt, mediaKind)
	ext := matrixMediaExt(label, matrixContentType(msgEvt), mediaKind)
	mediaDir, err := matrixMediaTempDir()
	if err != nil {
		return "", fmt.Errorf("create matrix media directory: %w", err)
	}
	tmp, err := os.CreateTemp(mediaDir, "matrix-media-*"+ext)
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		_ = tmp.Close()
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	_, err = io.Copy(tmp, reader)
	if err != nil {
		return "", err
	}
	if err = readerClose(); err != nil {
		return "", fmt.Errorf("decrypt matrix media: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return "", err
	}

	cleanup = false
	return tmpPath, nil
}

func matrixContentType(msgEvt *event.MessageEventContent) string {
	if msgEvt != nil && msgEvt.Info != nil {
		return strings.TrimSpace(msgEvt.Info.MimeType)
	}
	return ""
}

func matrixMediaURI(msgEvt *event.MessageEventContent) id.ContentURIString {
	if msgEvt == nil {
		return ""
	}
	if msgEvt.URL != "" {
		return msgEvt.URL
	}
	if msgEvt.File != nil {
		return msgEvt.File.URL
	}
	return ""
}

func matrixMediaKind(msgType event.MessageType) string {
	switch msgType {
	case event.MsgAudio:
		return "audio"
	case event.MsgVideo:
		return "video"
	case event.MsgFile:
		return "file"
	default:
		return "image"
	}
}

func matrixOutboundMsgType(partType, filename, contentType string) event.MessageType {
	switch strings.ToLower(strings.TrimSpace(partType)) {
	case "image":
		return event.MsgImage
	case "audio", "voice":
		return event.MsgAudio
	case "video":
		return event.MsgVideo
	case "file", "document":
		return event.MsgFile
	}

	ct := strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.HasPrefix(ct, "image/"):
		return event.MsgImage
	case strings.HasPrefix(ct, "audio/"), ct == "application/ogg", ct == "application/x-ogg":
		return event.MsgAudio
	case strings.HasPrefix(ct, "video/"):
		return event.MsgVideo
	}

	switch strings.ToLower(strings.TrimSpace(filepath.Ext(filename))) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return event.MsgImage
	case ".mp3", ".wav", ".ogg", ".m4a", ".flac", ".aac", ".wma", ".opus":
		return event.MsgAudio
	case ".mp4", ".avi", ".mov", ".webm", ".mkv":
		return event.MsgVideo
	default:
		return event.MsgFile
	}
}

func matrixOutboundContent(
	caption, filename string,
	msgType event.MessageType,
	contentType string,
	size int64,
	uri id.ContentURIString,
) *event.MessageEventContent {
	body := strings.TrimSpace(caption)
	if body == "" {
		body = filename
	}
	if body == "" {
		body = matrixMediaKind(msgType)
	}

	info := &event.FileInfo{MimeType: strings.TrimSpace(contentType)}
	if size > 0 && size <= int64(int(^uint(0)>>1)) {
		info.Size = int(size)
	}

	content := &event.MessageEventContent{
		MsgType:  msgType,
		Body:     body,
		URL:      uri,
		FileName: filename,
		Info:     info,
	}
	return content
}

func matrixMediaLabel(msgEvt *event.MessageEventContent, fallback string) string {
	if msgEvt == nil {
		return fallback
	}
	if v := strings.TrimSpace(msgEvt.FileName); v != "" {
		return v
	}
	if v := strings.TrimSpace(msgEvt.Body); v != "" {
		return v
	}
	return fallback
}

func matrixMediaFilename(label, mediaKind, contentType string) string {
	filename := strings.TrimSpace(label)
	if filename == "" {
		filename = mediaKind
	}
	if filepath.Ext(filename) == "" {
		filename += matrixMediaExt("", contentType, mediaKind)
	}
	return filename
}

func matrixMediaExt(filename, contentType, mediaKind string) string {
	if ext := strings.TrimSpace(filepath.Ext(filename)); ext != "" {
		return ext
	}
	if contentType != "" {
		if exts, err := mime.ExtensionsByType(contentType); err == nil && len(exts) > 0 {
			return exts[0]
		}
	}
	switch mediaKind {
	case "audio":
		return ".ogg"
	case "video":
		return ".mp4"
	case "file":
		return ".bin"
	default:
		return ".jpg"
	}
}

func (c *MatrixChannel) isGroupRoom(ctx context.Context, roomID id.RoomID) bool {
	now := time.Now()
	if isGroup, ok := c.roomKindCache.get(roomID.String(), now); ok {
		return isGroup
	}

	qctx := c.baseContext()
	if ctx != nil {
		qctx = ctx
	}
	reqCtx, cancel := context.WithTimeout(qctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.JoinedMembers(reqCtx, roomID)
	if err != nil {
		logger.DebugCF("matrix", "Failed to query room members; assume direct", map[string]any{
			"room_id": roomID.String(),
			"error":   err.Error(),
		})
		return false
	}

	isGroup := len(resp.Joined) > 2
	c.roomKindCache.set(roomID.String(), isGroup, now)
	return isGroup
}

func (c *MatrixChannel) isBotMentioned(msgEvt *event.MessageEventContent) bool {
	if msgEvt == nil {
		return false
	}

	if msgEvt.Mentions != nil && msgEvt.Mentions.Has(c.client.UserID) {
		return true
	}

	userID := c.client.UserID.String()
	if userID != "" && strings.Contains(msgEvt.Body, userID) {
		return true
	}
	if mentionsUserInFormattedBody(msgEvt.FormattedBody, c.client.UserID) {
		return true
	}

	mentionR := c.localpartMentionR
	if mentionR == nil {
		mentionR = localpartMentionRegexp(matrixLocalpart(c.client.UserID))
	}
	if mentionR == nil {
		return false
	}

	// Matrix users are addressed as MXID "@localpart:server", but many clients
	// emit plain-text mentions as "@localpart". Both forms are handled here.
	return mentionR.MatchString(msgEvt.Body) || mentionR.MatchString(msgEvt.FormattedBody)
}

func mentionsUserInFormattedBody(formattedBody string, userID id.UserID) bool {
	target := strings.ToLower(strings.TrimSpace(userID.String()))
	if target == "" {
		return false
	}

	formattedBody = strings.TrimSpace(formattedBody)
	if formattedBody == "" {
		return false
	}

	if strings.Contains(strings.ToLower(formattedBody), target) {
		return true
	}

	matches := matrixMentionHrefRegexp.FindAllStringSubmatch(formattedBody, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		decoded := decodeMatrixMentionHref(match[1])
		if strings.Contains(strings.ToLower(decoded), target) {
			return true
		}

		u, err := url.Parse(decoded)
		if err != nil {
			continue
		}

		if strings.Contains(strings.ToLower(u.Path), target) || strings.Contains(strings.ToLower(u.Fragment), target) {
			return true
		}
		if strings.Contains(strings.ToLower(decodeMatrixMentionHref(u.Fragment)), target) {
			return true
		}
	}

	return false
}

func decodeMatrixMentionHref(v string) string {
	decoded := html.UnescapeString(strings.TrimSpace(v))
	if decoded == "" {
		return ""
	}

	for i := 0; i < 2; i++ {
		next, err := url.QueryUnescape(decoded)
		if err != nil || next == decoded {
			break
		}
		decoded = next
	}
	return decoded
}

func (c *MatrixChannel) typingLoop(ctx context.Context, roomID id.RoomID, session *typingSession) {
	sendTyping := func() {
		_, err := c.client.UserTyping(ctx, roomID, true, typingServerTTL)
		if err != nil {
			logger.DebugCF("matrix", "Failed to send typing status", map[string]any{
				"room_id": roomID.String(),
				"error":   err.Error(),
			})
		}
	}

	sendTyping()
	ticker := time.NewTicker(typingRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-session.stopCh:
			return
		case <-ticker.C:
			sendTyping()
		}
	}
}

func (c *MatrixChannel) stopTypingSessions(ctx context.Context) {
	c.typingMu.Lock()
	sessions := c.typingSessions
	c.typingSessions = make(map[string]*typingSession)
	c.typingMu.Unlock()

	stopCtx := ctx
	if stopCtx == nil {
		stopCtx = context.Background()
	}
	for roomID, session := range sessions {
		session.stop()
		_, _ = c.client.UserTyping(stopCtx, id.RoomID(roomID), false, 0)
	}
}

func (c *MatrixChannel) baseContext() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}

func (c *MatrixChannel) runRoomKindCacheJanitor(ctx context.Context) {
	ticker := time.NewTicker(roomKindCacheCleanupPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			c.roomKindCache.cleanupExpired(now)
		}
	}
}

func (c *MatrixChannel) stripSelfMention(text string) string {
	return stripUserMentionWithRegexp(text, c.client.UserID, c.localpartMentionR)
}

func matrixMediaTempDir() (string, error) {
	mediaDir := media.TempDir()
	if err := os.MkdirAll(mediaDir, 0o700); err != nil {
		return "", err
	}
	return mediaDir, nil
}

func matrixLocalpart(userID id.UserID) string {
	s := strings.TrimPrefix(userID.String(), "@")
	localpart, _, _ := strings.Cut(s, ":")
	return strings.TrimSpace(localpart)
}

func localpartMentionRegexp(localpart string) *regexp.Regexp {
	localpart = strings.TrimSpace(localpart)
	if localpart == "" {
		return nil
	}

	// Match Matrix mentions in plain text while avoiding false positives:
	//   "@picoclaw" and "@picoclaw:matrix.org" should match,
	//   "test@example.com" and "hellopicoclawworld" should not.
	pattern := `(?i)(^|[^[:alnum:]_])@` + regexp.QuoteMeta(localpart) + `(?::[A-Za-z0-9._:-]+)?([^[:alnum:]_]|$)`
	return regexp.MustCompile(pattern)
}

func stripUserMention(text string, userID id.UserID) string {
	return stripUserMentionWithRegexp(text, userID, localpartMentionRegexp(matrixLocalpart(userID)))
}

func stripUserMentionWithRegexp(text string, userID id.UserID, mentionR *regexp.Regexp) string {
	cleaned := strings.ReplaceAll(text, userID.String(), "")

	if mentionR != nil {
		cleaned = mentionR.ReplaceAllString(cleaned, "$1$2")
	}

	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.TrimLeft(cleaned, ",:; ")
	return strings.TrimSpace(cleaned)
}

// VoiceCapabilities returns the voice capabilities of the channel.
func (c *MatrixChannel) VoiceCapabilities() channels.VoiceCapabilities {
	return channels.VoiceCapabilities{ASR: true, TTS: true}
}
