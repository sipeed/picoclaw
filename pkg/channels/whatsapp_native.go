// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package channels

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mdp/qrterminal/v3"
	_ "modernc.org/sqlite"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

const (
	sqliteDriver   = "sqlite"
	whatsappDBName = "store.db"

	reconnectInitial   = 5 * time.Second
	reconnectMax       = 5 * time.Minute
	reconnectMultiplier = 2.0
)

// WhatsAppNativeChannel implements the WhatsApp channel using whatsmeow (in-process, no external bridge).
type WhatsAppNativeChannel struct {
	*BaseChannel
	config       config.WhatsAppConfig
	storePath    string
	client       *whatsmeow.Client
	container    *sqlstore.Container
	mu           sync.Mutex
	runCtx       context.Context
	runCancel    context.CancelFunc
	reconnectMu  sync.Mutex
	reconnecting bool
}

// NewWhatsAppNativeChannel creates a WhatsApp channel that uses whatsmeow for connection.
// storePath is the directory for the SQLite session store (e.g. workspace/whatsapp).
func NewWhatsAppNativeChannel(cfg config.WhatsAppConfig, bus *bus.MessageBus, storePath string) (*WhatsAppNativeChannel, error) {
	base := NewBaseChannel("whatsapp", cfg, bus, cfg.AllowFrom)
	if storePath == "" {
		storePath = "whatsapp"
	}
	return &WhatsAppNativeChannel{
		BaseChannel: base,
		config:      cfg,
		storePath:   storePath,
	}, nil
}

func (c *WhatsAppNativeChannel) Start(ctx context.Context) error {
	logger.InfoCF("channels", "Starting WhatsApp native channel (whatsmeow)", map[string]any{"store": c.storePath})

	if err := os.MkdirAll(c.storePath, 0700); err != nil {
		return fmt.Errorf("create session store dir: %w", err)
	}

	dbPath := filepath.Join(c.storePath, whatsappDBName)
	connStr := "file:" + dbPath + "?_foreign_keys=on"

	// Open DB and enable foreign keys explicitly (modernc.org/sqlite does not set them from URI).
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
	client.AddEventHandler(c.eventHandler)

	c.mu.Lock()
	c.container = container
	c.client = client
	c.mu.Unlock()

	if client.Store.ID == nil {
		qrChan, err := client.GetQRChannel(ctx)
		if err != nil {
			_ = container.Close()
			return fmt.Errorf("get QR channel: %w", err)
		}
		if err := client.Connect(); err != nil {
			_ = container.Close()
			return fmt.Errorf("connect: %w", err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				logger.InfoCF("channels", "Scan this QR code with WhatsApp (Linked Devices):", nil)
				qrterminal.GenerateWithConfig(evt.Code, qrterminal.Config{
					Level:      qrterminal.L,
					Writer:     os.Stdout,
					HalfBlocks: true,
				})
			} else {
				logger.InfoCF("channels", "WhatsApp login event", map[string]any{"event": evt.Event})
			}
		}
	} else {
		if err := client.Connect(); err != nil {
			_ = container.Close()
			return fmt.Errorf("connect: %w", err)
		}
	}

	c.runCtx, c.runCancel = context.WithCancel(ctx)
	c.setRunning(true)
	logger.InfoCF("channels", "WhatsApp native channel connected", nil)
	return nil
}

func (c *WhatsAppNativeChannel) Stop(ctx context.Context) error {
	logger.InfoCF("channels", "Stopping WhatsApp native channel", nil)
	if c.runCancel != nil {
		c.runCancel()
	}
	c.mu.Lock()
	client := c.client
	container := c.container
	c.client = nil
	c.container = nil
	c.mu.Unlock()

	if client != nil {
		client.Disconnect()
	}
	if container != nil {
		_ = container.Close()
	}
	c.setRunning(false)
	return nil
}

func (c *WhatsAppNativeChannel) eventHandler(evt interface{}) {
	switch evt.(type) {
	case *events.Message:
		c.handleIncoming(evt.(*events.Message))
	case *events.Disconnected:
		logger.InfoCF("channels", "WhatsApp disconnected, will attempt reconnection", nil)
		c.reconnectMu.Lock()
		if c.reconnecting {
			c.reconnectMu.Unlock()
			return
		}
		c.reconnecting = true
		c.reconnectMu.Unlock()
		go c.reconnectWithBackoff()
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

		logger.InfoCF("channels", "WhatsApp reconnecting", map[string]any{"backoff": backoff.String()})
		err := client.Connect()
		if err == nil {
			logger.InfoCF("channels", "WhatsApp reconnected", nil)
			return
		}

		logger.WarnCF("channels", "WhatsApp reconnect failed", map[string]any{"error": err.Error()})

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

	if content == "" { return } // ignore empty messages

	var mediaPaths []string
	// Optional: resolve media to local paths if needed; for now we only forward text to the bus.
	_ = mediaPaths

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

	logger.InfoCF("channels", "WhatsApp message received", map[string]any{"sender_id": senderID, "content_preview": utils.Truncate(content, 50)})
	c.HandleMessage(senderID, chatID, content, mediaPaths, metadata)
}

func (c *WhatsAppNativeChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("whatsapp connection not established")
	}

	to, err := parseJID(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid chat id %q: %w", msg.ChatID, err)
	}

	waMsg := &waE2E.Message{
		Conversation: proto.String(msg.Content),
	}

	_, err = client.SendMessage(ctx, to, waMsg)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
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
	// Assume phone number for user chat.
	return types.NewJID(s, types.DefaultUserServer), nil
}
