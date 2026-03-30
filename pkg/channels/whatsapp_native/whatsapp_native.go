//go:build whatsapp_native

// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	sqliteDriver   = "sqlite"
	whatsappDBName = "store.db"

	reconnectInitial    = 5 * time.Second
	reconnectMax        = 5 * time.Minute
	reconnectMultiplier = 2.0

	maxTypingDuration      = 5 * time.Minute
	defaultPlaceholderText = "Thinking... 💭"
)

// WhatsAppNativeChannel implements the WhatsApp channel using whatsmeow (in-process, no external bridge).
type WhatsAppNativeChannel struct {
	*channels.BaseChannel
	config       config.WhatsAppConfig
	storePath    string
	client       *whatsmeow.Client
	container    *sqlstore.Container
	mu           sync.Mutex
	runCtx       context.Context
	runCancel    context.CancelFunc
	reconnectMu  sync.Mutex
	reconnecting bool
	stopping     atomic.Bool    // set once Stop begins; prevents new wg.Add calls
	wg           sync.WaitGroup // tracks background goroutines (QR handler, reconnect)
}

// NewWhatsAppNativeChannel creates a WhatsApp channel that uses whatsmeow for connection.
// storePath is the directory for the SQLite session store (e.g. workspace/whatsapp).
func NewWhatsAppNativeChannel(
	cfg config.WhatsAppConfig,
	bus *bus.MessageBus,
	storePath string,
) (channels.Channel, error) {
	base := channels.NewBaseChannel("whatsapp_native", cfg, bus, cfg.AllowFrom, channels.WithMaxMessageLength(65536))
	if storePath == "" {
		storePath = "whatsapp"
	}
	c := &WhatsAppNativeChannel{
		BaseChannel: base,
		config:      cfg,
		storePath:   storePath,
	}
	return c, nil
}

func (c *WhatsAppNativeChannel) Start(ctx context.Context) error {
	logger.InfoCF("whatsapp", "Starting WhatsApp native channel (whatsmeow)", map[string]any{"store": c.storePath})

	// Reset lifecycle state from any previous Stop() so a restarted channel
	// behaves correctly.  Use reconnectMu to be consistent with eventHandler
	// and Stop() which coordinate under the same lock.
	c.reconnectMu.Lock()
	c.stopping.Store(false)
	c.reconnecting = false
	c.reconnectMu.Unlock()

	if err := os.MkdirAll(c.storePath, 0o700); err != nil {
		return fmt.Errorf("create session store dir: %w", err)
	}

	dbPath := filepath.Join(c.storePath, whatsappDBName)
	connStr := "file:" + dbPath + "?_foreign_keys=on"

	db, err := sql.Open(sqliteDriver, connStr)
	if err != nil {
		return fmt.Errorf("open whatsapp store: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err = db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return fmt.Errorf("enable foreign keys: %w", err)
	}

	waLogger := waLog.Stdout("WhatsApp", "WARN", true)
	container := sqlstore.NewWithDB(db, sqliteDriver, waLogger)
	if err = container.Upgrade(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("open whatsapp store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		_ = container.Close()
		return fmt.Errorf("get device store: %w", err)
	}

	client := whatsmeow.NewClient(deviceStore, waLogger)

	// Create runCtx/runCancel BEFORE registering event handler and starting
	// goroutines so that Stop() can cancel them at any time, including during
	// the QR-login flow.
	c.runCtx, c.runCancel = context.WithCancel(ctx)

	client.AddEventHandler(c.eventHandler)

	c.mu.Lock()
	c.container = container
	c.client = client
	c.mu.Unlock()

	// cleanupOnError clears struct references and releases resources when
	// Start() fails after fields are already assigned.  This prevents
	// Stop() from operating on stale references (double-close, disconnect
	// of a partially-initialized client, or stray event handler callbacks).
	startOK := false
	defer func() {
		if startOK {
			return
		}
		c.runCancel()
		client.Disconnect()
		c.mu.Lock()
		c.client = nil
		c.container = nil
		c.mu.Unlock()
		_ = container.Close()
	}()

	if client.Store.ID == nil {
		qrChan, err := client.GetQRChannel(c.runCtx)
		if err != nil {
			return fmt.Errorf("get QR channel: %w", err)
		}
		if err := client.Connect(); err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		// Handle QR events in a background goroutine so Start() returns
		// promptly.  The goroutine is tracked via c.wg and respects
		// c.runCtx for cancellation.
		// Guard wg.Add with reconnectMu + stopping check (same protocol
		// as eventHandler) so a concurrent Stop() cannot enter wg.Wait()
		// while we call wg.Add(1).
		c.reconnectMu.Lock()
		if c.stopping.Load() {
			c.reconnectMu.Unlock()
			return fmt.Errorf("channel stopped during QR setup")
		}
		c.wg.Add(1)
		c.reconnectMu.Unlock()
		go func() {
			defer c.wg.Done()
			for {
				select {
				case <-c.runCtx.Done():
					return
				case evt, ok := <-qrChan:
					if !ok {
						return
					}
					if evt.Event == "code" {
						logger.InfoCF("whatsapp", "Scan this QR code with WhatsApp (Linked Devices):", nil)
						qrterminal.GenerateWithConfig(evt.Code, qrterminal.Config{
							Level:      qrterminal.L,
							Writer:     os.Stdout,
							HalfBlocks: true,
						})
					} else {
						logger.InfoCF("whatsapp", "WhatsApp login event", map[string]any{"event": evt.Event})
					}
				}
			}
		}()
	} else {
		if err := client.Connect(); err != nil {
			return fmt.Errorf("connect: %w", err)
		}
	}

	startOK = true
	c.SetRunning(true)
	logger.InfoC("whatsapp", "WhatsApp native channel connected")
	return nil
}

func (c *WhatsAppNativeChannel) Stop(ctx context.Context) error {
	logger.InfoC("whatsapp", "Stopping WhatsApp native channel")

	// Mark as stopping under reconnectMu so the flag is visible to
	// eventHandler atomically with respect to its wg.Add(1) call.
	// This closes the TOCTOU window where eventHandler could check
	// stopping (false), then Stop sets it true + enters wg.Wait,
	// then eventHandler calls wg.Add(1) — causing a panic.
	c.reconnectMu.Lock()
	c.stopping.Store(true)
	c.reconnectMu.Unlock()

	if c.runCancel != nil {
		c.runCancel()
	}

	// Disconnect the client first so any blocking Connect()/reconnect loops
	// can be interrupted before we wait on the goroutines.
	c.mu.Lock()
	client := c.client
	container := c.container
	c.mu.Unlock()

	if client != nil {
		client.Disconnect()
	}

	// Wait for background goroutines (QR handler, reconnect) to finish in a
	// context-aware way so Stop can be bounded by ctx.
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines have finished.
	case <-ctx.Done():
		// Context canceled or timed out; log and proceed with best-effort cleanup.
		logger.WarnC("whatsapp", fmt.Sprintf("Stop context canceled before all goroutines finished: %v", ctx.Err()))
	}

	// Now it is safe to clear and close resources.
	c.mu.Lock()
	c.client = nil
	c.container = nil
	c.mu.Unlock()

	if container != nil {
		_ = container.Close()
	}
	c.SetRunning(false)
	return nil
}

func (c *WhatsAppNativeChannel) eventHandler(evt any) {
	switch evt.(type) {
	case *events.Message:
		c.handleIncoming(evt.(*events.Message))
	case *events.Disconnected:
		logger.InfoCF("whatsapp", "WhatsApp disconnected, will attempt reconnection", nil)
		c.reconnectMu.Lock()
		if c.reconnecting {
			c.reconnectMu.Unlock()
			return
		}
		// Check stopping while holding the lock so the check and wg.Add
		// are atomic with respect to Stop() setting the flag + calling
		// wg.Wait(). This prevents the TOCTOU race.
		if c.stopping.Load() {
			c.reconnectMu.Unlock()
			return
		}
		c.reconnecting = true
		c.wg.Add(1)
		c.reconnectMu.Unlock()
		go func() {
			defer c.wg.Done()
			c.reconnectWithBackoff()
		}()
	}
}

func (c *WhatsAppNativeChannel) reconnectWithBackoff() {
	defer func() {
		c.reconnectMu.Lock()
		c.reconnecting = false
		c.reconnectMu.Unlock()
	}()

	backoff := reconnectInitial
	for {
		select {
		case <-c.runCtx.Done():
			return
		default:
		}

		c.mu.Lock()
		client := c.client
		c.mu.Unlock()
		if client == nil {
			return
		}

		logger.InfoCF("whatsapp", "WhatsApp reconnecting", map[string]any{"backoff": backoff.String()})
		err := client.Connect()
		if err == nil {
			logger.InfoC("whatsapp", "WhatsApp reconnected")
			return
		}

		logger.WarnCF("whatsapp", "WhatsApp reconnect failed", map[string]any{"error": err.Error()})

		select {
		case <-c.runCtx.Done():
			return
		case <-time.After(backoff):
			if backoff < reconnectMax {
				next := time.Duration(float64(backoff) * reconnectMultiplier)
				if next > reconnectMax {
					next = reconnectMax
				}
				backoff = next
			}
		}
	}
}

func (c *WhatsAppNativeChannel) handleIncoming(evt *events.Message) {
	if evt.Message == nil {
		return
	}
	senderID := evt.Info.Sender.String()
	chatID := evt.Info.Chat.String()
	messageID := evt.Info.ID

	sender := bus.SenderInfo{
		Platform:    "whatsapp",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("whatsapp", senderID),
		DisplayName: evt.Info.PushName,
	}
	if !c.IsAllowedSender(sender) {
		return
	}

	m := evt.Message
	needsClient := m.GetImageMessage() != nil || m.GetVideoMessage() != nil ||
		m.GetAudioMessage() != nil || m.GetDocumentMessage() != nil || m.GetStickerMessage() != nil

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if needsClient && client == nil {
		return
	}

	content := m.GetConversation()
	if content == "" && m.ExtendedTextMessage != nil {
		content = m.ExtendedTextMessage.GetText()
	}

	mediaPaths := []string{}
	scope := channels.BuildMediaScope("whatsapp_native", chatID, messageID)

	storeMedia := func(localPath, filename string) string {
		if store := c.GetMediaStore(); store != nil {
			ref, err := store.Store(localPath, media.MediaMeta{
				Filename: filename,
				Source:   "whatsapp_native",
			}, scope)
			if err == nil {
				return ref
			}
			logger.WarnCF("whatsapp", "Failed to store WhatsApp media in MediaStore", map[string]any{
				"path": localPath, "error": err.Error(),
			})
		}
		return localPath
	}

	appendCaption := func(cap string) {
		cap = strings.TrimSpace(cap)
		if cap == "" {
			return
		}
		if content != "" {
			content += "\n"
		}
		content += cap
	}

	appendAnnotation := func(line string) {
		if content != "" {
			content += "\n"
		}
		content += line
	}

	downloadAndStore := func(dm whatsmeow.DownloadableMessage, storeName, annotation string) {
		localPath, err := c.downloadWhatsAppMediaToTemp(c.runCtx, client, dm)
		if err != nil {
			logger.ErrorCF("whatsapp", "Failed to download WhatsApp media", map[string]any{
				"error": err.Error(),
			})
			return
		}
		mediaPaths = append(mediaPaths, storeMedia(localPath, storeName))
		appendAnnotation(annotation)
	}

	if img := m.GetImageMessage(); img != nil {
		appendCaption(img.GetCaption())
		fname := imageFilenameFromMime(img.GetMimetype())
		downloadAndStore(img, fname, "[image: photo]")
	}
	if vid := m.GetVideoMessage(); vid != nil {
		appendCaption(vid.GetCaption())
		downloadAndStore(vid, "video.mp4", "[video]")
	}
	if aud := m.GetAudioMessage(); aud != nil {
		ann := "[audio]"
		if aud.GetPTT() {
			ann = "[voice]"
		}
		fname := "audio.ogg"
		if mt := aud.GetMimetype(); strings.Contains(mt, "mpeg") || strings.Contains(mt, "mp3") {
			fname = "audio.mp3"
		}
		downloadAndStore(aud, fname, ann)
	}
	if doc := m.GetDocumentMessage(); doc != nil {
		appendCaption(doc.GetCaption())
		fname := doc.GetFileName()
		if strings.TrimSpace(fname) == "" {
			fname = "document"
		}
		downloadAndStore(doc, fname, "[file: "+fname+"]")
	}
	if st := m.GetStickerMessage(); st != nil {
		downloadAndStore(st, "sticker.webp", "[sticker]")
	}

	content = utils.SanitizeMessageContent(content)
	if content == "" && len(mediaPaths) == 0 {
		return
	}

	metadata := make(map[string]string)
	metadata["message_id"] = evt.Info.ID
	if evt.Info.PushName != "" {
		metadata["user_name"] = evt.Info.PushName
	}
	if evt.Info.Chat.Server == types.GroupServer {
		metadata["peer_kind"] = "group"
		metadata["peer_id"] = chatID
	} else {
		metadata["peer_kind"] = "direct"
		metadata["peer_id"] = senderID
	}

	peerKind := "direct"
	if evt.Info.Chat.Server == types.GroupServer {
		peerKind = "group"
	}
	peer := bus.Peer{Kind: peerKind, ID: chatID}

	logger.DebugCF(
		"whatsapp",
		"WhatsApp message received",
		map[string]any{"sender_id": senderID, "content_preview": utils.Truncate(content, 50), "media_count": len(mediaPaths)},
	)
	c.HandleMessage(c.runCtx, peer, messageID, senderID, chatID, content, mediaPaths, metadata, sender)
}

// downloadWhatsAppMediaToTemp decrypts and writes WhatsApp attachment bytes to a temp file.
func (c *WhatsAppNativeChannel) downloadWhatsAppMediaToTemp(ctx context.Context, cli *whatsmeow.Client, msg whatsmeow.DownloadableMessage) (path string, err error) {
	data, err := cli.Download(ctx, msg)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "whatsapp-native-media-*")
	if err != nil {
		return "", err
	}
	path = f.Name()
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", err
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func imageFilenameFromMime(mimetype string) string {
	switch {
	case strings.Contains(mimetype, "png"):
		return "photo.png"
	case strings.Contains(mimetype, "gif"):
		return "photo.gif"
	case strings.Contains(mimetype, "webp"):
		return "photo.webp"
	default:
		return "photo.jpg"
	}
}

// defaultPlaceholderMessageText returns the placeholder body (config or Telegram-style default).
func defaultPlaceholderMessageText(cfg config.PlaceholderConfig) string {
	if cfg.Text != "" {
		return cfg.Text
	}
	return defaultPlaceholderText
}

// StartTyping implements channels.TypingCapable.
// It sends composing presence immediately and refreshes every ~4s until stop() or maxTypingDuration.
// stop is idempotent and sends ChatPresencePaused once.
func (c *WhatsAppNativeChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	if !c.config.Typing.Enabled {
		return func() {}, nil
	}
	to, err := parseJID(chatID)
	if err != nil {
		return func() {}, err
	}

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client == nil || !c.IsRunning() || !client.IsConnected() || client.Store.ID == nil {
		return func() {}, nil
	}

	_ = client.SendChatPresence(ctx, to, types.ChatPresenceComposing, types.ChatPresenceMediaText)

	typingCtx, cancel := context.WithCancel(ctx)
	maxCtx, maxCancel := context.WithTimeout(typingCtx, maxTypingDuration)

	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			cancel()
			maxCancel()
			c.mu.Lock()
			cl := c.client
			c.mu.Unlock()
			if cl != nil && cl.IsConnected() && cl.Store.ID != nil {
				_ = cl.SendChatPresence(context.Background(), to, types.ChatPresencePaused, types.ChatPresenceMediaText)
			}
		})
	}

	go func() {
		defer maxCancel()
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-maxCtx.Done():
				return
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				c.mu.Lock()
				cl := c.client
				c.mu.Unlock()
				if cl == nil || !cl.IsConnected() || cl.Store.ID == nil {
					return
				}
				_ = cl.SendChatPresence(typingCtx, to, types.ChatPresenceComposing, types.ChatPresenceMediaText)
			}
		}
	}()

	return stop, nil
}

// SendPlaceholder implements channels.PlaceholderCapable.
func (c *WhatsAppNativeChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	if !c.config.Placeholder.Enabled {
		return "", nil
	}
	text := defaultPlaceholderMessageText(c.config.Placeholder)

	if !c.IsRunning() {
		return "", channels.ErrNotRunning
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil || !client.IsConnected() {
		return "", fmt.Errorf("whatsapp connection not established: %w", channels.ErrTemporary)
	}
	if client.Store.ID == nil {
		return "", fmt.Errorf("whatsapp not yet paired (QR login pending): %w", channels.ErrTemporary)
	}

	to, err := parseJID(chatID)
	if err != nil {
		return "", fmt.Errorf("invalid chat id %q: %w", chatID, err)
	}

	waMsg := &waE2E.Message{
		Conversation: proto.String(text),
	}
	resp, err := client.SendMessage(ctx, to, waMsg)
	if err != nil {
		return "", fmt.Errorf("whatsapp send placeholder: %w", channels.ErrTemporary)
	}
	return string(resp.ID), nil
}

// EditMessage implements channels.MessageEditor.
func (c *WhatsAppNativeChannel) EditMessage(ctx context.Context, chatID string, messageID string, content string) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("whatsapp connection not established: %w", channels.ErrTemporary)
	}
	if client.Store.ID == nil {
		return fmt.Errorf("whatsapp not yet paired (QR login pending): %w", channels.ErrTemporary)
	}

	to, err := parseJID(chatID)
	if err != nil {
		return fmt.Errorf("invalid chat id %q: %w", chatID, err)
	}

	id := types.MessageID(messageID)
	edit := client.BuildEdit(to, id, &waE2E.Message{Conversation: proto.String(content)})
	if _, err = client.SendMessage(ctx, to, edit); err != nil {
		return fmt.Errorf("whatsapp edit message: %w", channels.ErrTemporary)
	}
	return nil
}

func (c *WhatsAppNativeChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("whatsapp connection not established: %w", channels.ErrTemporary)
	}

	// Detect unpaired state: the client is connected (to WhatsApp servers)
	// but has not completed QR-login yet, so sending would fail.
	if client.Store.ID == nil {
		return fmt.Errorf("whatsapp not yet paired (QR login pending): %w", channels.ErrTemporary)
	}

	to, err := parseJID(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid chat id %q: %w", msg.ChatID, err)
	}

	waMsg := &waE2E.Message{
		Conversation: proto.String(msg.Content),
	}

	if _, err = client.SendMessage(ctx, to, waMsg); err != nil {
		return fmt.Errorf("whatsapp send: %w", channels.ErrTemporary)
	}
	return nil
}

// SendMedia implements channels.MediaSender for outbound images, audio, video, and files.
func (c *WhatsAppNativeChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("whatsapp connection not established: %w", channels.ErrTemporary)
	}
	if client.Store.ID == nil {
		return fmt.Errorf("whatsapp not yet paired (QR login pending): %w", channels.ErrTemporary)
	}

	to, err := parseJID(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid chat id %q: %w", msg.ChatID, channels.ErrSendFailed)
	}

	store := c.GetMediaStore()
	if store == nil {
		return fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	for _, part := range msg.Parts {
		localPath, err := store.Resolve(part.Ref)
		if err != nil {
			logger.ErrorCF("whatsapp", "Failed to resolve media ref for send", map[string]any{
				"ref": part.Ref, "error": err.Error(),
			})
			return fmt.Errorf("resolve media ref: %w", channels.ErrSendFailed)
		}

		data, err := os.ReadFile(localPath)
		if err != nil {
			logger.ErrorCF("whatsapp", "Failed to read media file for send", map[string]any{
				"path": localPath, "error": err.Error(),
			})
			return fmt.Errorf("read media: %w", channels.ErrSendFailed)
		}

		mt := whatsappOutboundMimetype(part, localPath)
		waMsg, err := buildWhatsAppMediaMessage(ctx, client, part.Type, data, mt, part.Caption, part.Filename)
		if err != nil {
			logger.ErrorCF("whatsapp", "Failed to build/upload WhatsApp media message", map[string]any{
				"type": part.Type, "error": err.Error(),
			})
			return fmt.Errorf("whatsapp media upload: %w", channels.ErrTemporary)
		}

		if _, err = client.SendMessage(ctx, to, waMsg); err != nil {
			logger.ErrorCF("whatsapp", "Failed to send WhatsApp media", map[string]any{
				"type": part.Type, "error": err.Error(),
			})
			return fmt.Errorf("whatsapp send media: %w", channels.ErrTemporary)
		}
	}

	return nil
}

func whatsappOutboundMimetype(part bus.MediaPart, localPath string) string {
	if part.ContentType != "" {
		return part.ContentType
	}
	ext := filepath.Ext(localPath)
	if ext != "" {
		if m := mime.TypeByExtension(ext); m != "" {
			return m
		}
	}
	switch part.Type {
	case "image":
		return "image/jpeg"
	case "audio":
		return "audio/mpeg"
	case "video":
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}

func buildWhatsAppMediaMessage(
	ctx context.Context,
	client *whatsmeow.Client,
	partType string,
	data []byte,
	mimetype, caption, filenameHint string,
) (*waE2E.Message, error) {
	switch partType {
	case "image":
		resp, err := client.Upload(ctx, data, whatsmeow.MediaImage)
		if err != nil {
			return nil, err
		}
		img := &waE2E.ImageMessage{
			Mimetype:      proto.String(mimetype),
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(resp.FileLength),
		}
		if caption != "" {
			img.Caption = proto.String(caption)
		}
		return &waE2E.Message{ImageMessage: img}, nil

	case "video":
		resp, err := client.Upload(ctx, data, whatsmeow.MediaVideo)
		if err != nil {
			return nil, err
		}
		vid := &waE2E.VideoMessage{
			Mimetype:      proto.String(mimetype),
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(resp.FileLength),
		}
		if caption != "" {
			vid.Caption = proto.String(caption)
		}
		return &waE2E.Message{VideoMessage: vid}, nil

	case "audio":
		resp, err := client.Upload(ctx, data, whatsmeow.MediaAudio)
		if err != nil {
			return nil, err
		}
		ptt := strings.Contains(mimetype, "ogg") || strings.Contains(mimetype, "opus")
		aud := &waE2E.AudioMessage{
			Mimetype:      proto.String(mimetype),
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(resp.FileLength),
			PTT:           proto.Bool(ptt),
		}
		return &waE2E.Message{AudioMessage: aud}, nil

	default: // "file" and unknown
		resp, err := client.Upload(ctx, data, whatsmeow.MediaDocument)
		if err != nil {
			return nil, err
		}
		fname := strings.TrimSpace(filenameHint)
		if fname == "" {
			fname = "file"
		}
		doc := &waE2E.DocumentMessage{
			Mimetype:      proto.String(mimetype),
			FileName:      proto.String(fname),
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(resp.FileLength),
		}
		if caption != "" {
			doc.Caption = proto.String(caption)
		}
		return &waE2E.Message{DocumentMessage: doc}, nil
	}
}

// parseJID converts a chat ID (phone number or JID string) to types.JID.
func parseJID(s string) (types.JID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return types.JID{}, fmt.Errorf("empty chat id")
	}
	if strings.Contains(s, "@") {
		return types.ParseJID(s)
	}
	return types.NewJID(s, types.DefaultUserServer), nil
}
