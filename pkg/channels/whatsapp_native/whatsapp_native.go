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
	"os"
	"path/filepath"
	"regexp"
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
	switch v := evt.(type) {
	case *events.Message:
		c.handleIncoming(v)
	case *events.CallOffer:
		c.mu.Lock()
		client := c.client
		c.mu.Unlock()
		if client != nil {
			_ = client.RejectCall(context.Background(), v.BasicCallMeta.From, v.BasicCallMeta.CallID)
			logger.InfoCF("whatsapp", "Call rejected automatically", map[string]any{"from": v.BasicCallMeta.From.String()})
		}
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
	content := evt.Message.GetConversation()
	if content == "" && evt.Message.ExtendedTextMessage != nil {
		content = evt.Message.ExtendedTextMessage.GetText()
	}
	content = utils.SanitizeMessageContent(content)

	if content == "" {
		return
	}

	var mediaPaths []string

	metadata := make(map[string]string)
	metadata["message_id"] = evt.Info.ID
	if evt.Info.PushName != "" {
		metadata["user_name"] = evt.Info.PushName
	}

	// Resolve LID to phone number if available
	if evt.Info.Sender.Server == types.HiddenUserServer {
		metadata["lid"] = evt.Info.Sender.User
		c.mu.Lock()
		client := c.client
		c.mu.Unlock()
		if client != nil && client.Store.LIDs != nil {
			if pnJID, err := client.Store.LIDs.GetPNForLID(c.runCtx, evt.Info.Sender); err == nil && !pnJID.IsEmpty() {
				metadata["phone_number"] = "+" + pnJID.User
				senderID = pnJID.String()
				logger.DebugCF("whatsapp", "LID resolved", map[string]any{"lid": evt.Info.Sender.User, "phone": "+" + pnJID.User})
			} else {
				logger.WarnCF("whatsapp", "LID not found in DB — unknown sender", map[string]any{"lid": evt.Info.Sender.User})
			}
		}
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

	logger.DebugCF(
		"whatsapp",
		"WhatsApp message received",
		map[string]any{"sender_id": senderID, "content_preview": utils.Truncate(content, 50)},
	)

	// Send read receipt (blue ticks)
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client != nil {
		senderJID := evt.Info.Sender
		if evt.Info.Chat.Server != types.GroupServer {
			senderJID = types.EmptyJID
		}
		_ = client.MarkRead(c.runCtx, []types.MessageID{evt.Info.ID}, time.Now(), evt.Info.Chat, senderJID)
	}

	c.HandleMessage(c.runCtx, peer, messageID, senderID, chatID, content, mediaPaths, metadata, sender)
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

	var waMsg *waE2E.Message
	if url := extractURL(msg.Content); url != "" {
		waMsg = &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text:        proto.String(msg.Content),
				MatchedText: proto.String(url),
			},
		}
	} else {
		waMsg = &waE2E.Message{Conversation: proto.String(msg.Content)}
	}

	if _, err = client.SendMessage(ctx, to, waMsg); err != nil {
		return fmt.Errorf("whatsapp send: %w", channels.ErrTemporary)
	}
	return nil
}

// SendMedia implements the channels.MediaSender interface.
func (c *WhatsAppNativeChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
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
		if err := c.sendMediaPart(ctx, client, to, store, part); err != nil {
			return err
		}
	}
	return nil
}

// sendMediaPart uploads and sends a single media part via whatsmeow.
func (c *WhatsAppNativeChannel) sendMediaPart(
	ctx context.Context,
	client *whatsmeow.Client,
	to types.JID,
	store media.MediaStore,
	part bus.MediaPart,
) error {
	localPath, err := store.Resolve(part.Ref)
	if err != nil {
		logger.ErrorCF("whatsapp", "Failed to resolve media ref", map[string]any{
			"ref":   part.Ref,
			"error": err.Error(),
		})
		return fmt.Errorf("whatsapp resolve media ref %q: %w", part.Ref, channels.ErrSendFailed)
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		logger.ErrorCF("whatsapp", "Failed to read media file", map[string]any{
			"path":  localPath,
			"error": err.Error(),
		})
		return fmt.Errorf("whatsapp read media: %w", channels.ErrSendFailed)
	}

	mimeType := part.ContentType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	var waMsg *waE2E.Message

	switch part.Type {
	case "image":
		resp, err := client.Upload(ctx, data, whatsmeow.MediaImage)
		if err != nil {
			return fmt.Errorf("whatsapp upload image: %w", channels.ErrTemporary)
		}
		waMsg = &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				Caption:       proto.String(part.Caption),
				Mimetype:      proto.String(mimeType),
				URL:           proto.String(resp.URL),
				DirectPath:    proto.String(resp.DirectPath),
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    proto.Uint64(resp.FileLength),
			},
		}

	case "video":
		resp, err := client.Upload(ctx, data, whatsmeow.MediaVideo)
		if err != nil {
			return fmt.Errorf("whatsapp upload video: %w", channels.ErrTemporary)
		}
		waMsg = &waE2E.Message{
			VideoMessage: &waE2E.VideoMessage{
				Caption:       proto.String(part.Caption),
				Mimetype:      proto.String(mimeType),
				URL:           proto.String(resp.URL),
				DirectPath:    proto.String(resp.DirectPath),
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    proto.Uint64(resp.FileLength),
			},
		}

	case "audio":
		resp, err := client.Upload(ctx, data, whatsmeow.MediaAudio)
		if err != nil {
			return fmt.Errorf("whatsapp upload audio: %w", channels.ErrTemporary)
		}
		// Detect voice messages (PTT) by filename convention
		fn := strings.ToLower(part.Filename)
		isPTT := strings.Contains(fn, "voice") || strings.Contains(fn, "ptt")
		waMsg = &waE2E.Message{
			AudioMessage: &waE2E.AudioMessage{
				Mimetype:      proto.String(mimeType),
				PTT:           proto.Bool(isPTT),
				URL:           proto.String(resp.URL),
				DirectPath:    proto.String(resp.DirectPath),
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    proto.Uint64(resp.FileLength),
			},
		}

	default: // "file" or unknown → send as document
		resp, err := client.Upload(ctx, data, whatsmeow.MediaDocument)
		if err != nil {
			return fmt.Errorf("whatsapp upload document: %w", channels.ErrTemporary)
		}
		filename := part.Filename
		if filename == "" {
			filename = filepath.Base(localPath)
		}
		waMsg = &waE2E.Message{
			DocumentMessage: &waE2E.DocumentMessage{
				Caption:       proto.String(part.Caption),
				Title:         proto.String(filename),
				FileName:      proto.String(filename),
				Mimetype:      proto.String(mimeType),
				URL:           proto.String(resp.URL),
				DirectPath:    proto.String(resp.DirectPath),
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    proto.Uint64(resp.FileLength),
			},
		}
	}

	if _, err := client.SendMessage(ctx, to, waMsg); err != nil {
		logger.ErrorCF("whatsapp", "Failed to send media", map[string]any{
			"type":  part.Type,
			"error": err.Error(),
		})
		return fmt.Errorf("whatsapp send media: %w", channels.ErrTemporary)
	}

	logger.DebugCF("whatsapp", "Media sent", map[string]any{
		"type": part.Type,
		"to":   to.String(),
	})
	return nil
}

// extractURL returns the first URL found in text, or empty string.
var urlPattern = regexp.MustCompile(`https?://\S+`)

func extractURL(text string) string {
	return urlPattern.FindString(text)
}

// StartTyping sends a "composing" chat presence to the given chat.
// It repeats every 4 seconds (WhatsApp indicator expires after ~5s).
// The returned stop function sends a "paused" presence and is idempotent.
func (c *WhatsAppNativeChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	jid, err := parseJID(chatID)
	if err != nil {
		return func() {}, fmt.Errorf("start typing: %w", err)
	}

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client == nil {
		return func() {}, nil
	}

	_ = client.SendChatPresence(ctx, jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)

	typingCtx, typingCancel := context.WithCancel(ctx)
	var once sync.Once

	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				c.mu.Lock()
				cl := c.client
				c.mu.Unlock()
				if cl != nil {
					_ = cl.SendChatPresence(typingCtx, jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)
				}
			}
		}
	}()

	stop := func() {
		once.Do(func() {
			typingCancel()
			c.mu.Lock()
			cl := c.client
			c.mu.Unlock()
			if cl != nil {
				_ = cl.SendChatPresence(context.Background(), jid, types.ChatPresencePaused, "")
			}
		})
	}
	return stop, nil
}

// EditMessage edits a previously sent message.
func (c *WhatsAppNativeChannel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	jid, err := parseJID(chatID)
	if err != nil {
		return fmt.Errorf("edit message: %w", err)
	}

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client == nil {
		return fmt.Errorf("whatsapp not connected")
	}

	edited := client.BuildEdit(jid, types.MessageID(messageID), &waE2E.Message{
		Conversation: proto.String(content),
	})
	_, err = client.SendMessage(ctx, jid, edited)
	return err
}

// DeleteMessage revokes (deletes for everyone) a previously sent message.
func (c *WhatsAppNativeChannel) DeleteMessage(ctx context.Context, chatID, messageID string) error {
	jid, err := parseJID(chatID)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client == nil {
		return fmt.Errorf("whatsapp not connected")
	}

	revoke := client.BuildRevoke(jid, types.EmptyJID, types.MessageID(messageID))
	_, err = client.SendMessage(ctx, jid, revoke)
	return err
}

// ReactToMessage adds a 👀 reaction to an inbound message.
// The returned undo function removes the reaction (idempotent).
func (c *WhatsAppNativeChannel) ReactToMessage(ctx context.Context, chatID, messageID string) (func(), error) {
	jid, err := parseJID(chatID)
	if err != nil {
		return func() {}, fmt.Errorf("react: %w", err)
	}

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client == nil {
		return func() {}, nil
	}

	reaction := client.BuildReaction(jid, types.EmptyJID, types.MessageID(messageID), "👀")
	_, _ = client.SendMessage(ctx, jid, reaction)

	var once sync.Once
	undo := func() {
		once.Do(func() {
			c.mu.Lock()
			cl := c.client
			c.mu.Unlock()
			if cl != nil {
				unreact := cl.BuildReaction(jid, types.EmptyJID, types.MessageID(messageID), "")
				_, _ = cl.SendMessage(context.Background(), jid, unreact)
			}
		})
	}
	return undo, nil
}

// SendPlaceholder sends a temporary "thinking" message and returns its ID
// so it can be edited later with the real response.
func (c *WhatsAppNativeChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	jid, err := parseJID(chatID)
	if err != nil {
		return "", fmt.Errorf("placeholder: %w", err)
	}

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client == nil {
		return "", fmt.Errorf("whatsapp not connected")
	}

	msg := &waE2E.Message{Conversation: proto.String("Buscando... ⏳")}
	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// IsOnWhatsApp checks if a phone number is registered on WhatsApp.
func (c *WhatsAppNativeChannel) IsOnWhatsApp(ctx context.Context, phone string) (bool, error) {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client == nil {
		return false, fmt.Errorf("whatsapp not connected")
	}

	cleaned := "+" + strings.TrimLeft(phone, "+")
	resp, err := client.IsOnWhatsApp(ctx, []string{cleaned})
	if err != nil {
		return false, err
	}
	if len(resp) == 0 {
		return false, nil
	}
	return resp[0].IsIn, nil
}

// CreateNewsletter creates a WhatsApp Channel (newsletter) and returns its JID.
func (c *WhatsAppNativeChannel) CreateNewsletter(ctx context.Context, name, description string) (string, error) {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client == nil {
		return "", fmt.Errorf("whatsapp not connected")
	}

	meta, err := client.CreateNewsletter(ctx, whatsmeow.CreateNewsletterParams{
		Name:        name,
		Description: description,
	})
	if err != nil {
		return "", err
	}
	return meta.ID.String(), nil
}

// SendToNewsletter sends a message to a WhatsApp Channel (newsletter).
func (c *WhatsAppNativeChannel) SendToNewsletter(ctx context.Context, newsletterJID, content string) error {
	jid, err := types.ParseJID(newsletterJID)
	if err != nil {
		return fmt.Errorf("invalid newsletter JID: %w", err)
	}

	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client == nil {
		return fmt.Errorf("whatsapp not connected")
	}

	msg := &waE2E.Message{Conversation: proto.String(content)}
	_, err = client.SendMessage(ctx, jid, msg)
	return err
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
