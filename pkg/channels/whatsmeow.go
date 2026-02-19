package channels

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type WhatsmeowChannel struct {
	*BaseChannel
	client    *whatsmeow.Client
	config    config.WhatsmeowConfig
	container *sqlstore.Container
	mu        sync.Mutex
	cancel    context.CancelFunc
}

func NewWhatsmeowChannel(cfg config.WhatsmeowConfig, msgBus *bus.MessageBus) (*WhatsmeowChannel, error) {
	base := NewBaseChannel("whatsmeow", cfg, msgBus, cfg.AllowFrom)

	return &WhatsmeowChannel{
		BaseChannel: base,
		config:      cfg,
	}, nil
}

func (c *WhatsmeowChannel) Start(ctx context.Context) error {
	dbPath := expandHomePath(c.config.DBPath)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("failed to create db directory: %w", err)
	}

	dbLog := waLog.Noop
	container, err := sqlstore.New(ctx, "sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", dbPath), dbLog)
	if err != nil {
		return fmt.Errorf("failed to open whatsmeow db: %w", err)
	}
	c.container = container

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	if deviceStore.ID == nil {
		return fmt.Errorf("no linked WhatsApp device found; run 'picoclaw whatsapp link' first")
	}

	clientLog := waLog.Noop
	client := whatsmeow.NewClient(deviceStore, clientLog)
	c.client = client

	client.AddEventHandler(c.eventHandler)

	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect whatsmeow: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.setRunning(true)
	logger.InfoC("whatsmeow", "WhatsApp (whatsmeow) channel connected")

	// Keep context alive for cleanup
	go func() {
		<-ctx.Done()
	}()

	return nil
}

func (c *WhatsmeowChannel) Stop(ctx context.Context) error {
	logger.InfoC("whatsmeow", "Stopping WhatsApp (whatsmeow) channel...")

	if c.cancel != nil {
		c.cancel()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		c.client.Disconnect()
		c.client = nil
	}

	c.setRunning(false)
	return nil
}

func (c *WhatsmeowChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil {
		return fmt.Errorf("whatsmeow client not connected")
	}

	jid, err := types.ParseJID(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid chat ID %q: %w", msg.ChatID, err)
	}

	chunks := utils.SplitMessage(msg.Content, 4096)
	for _, chunk := range chunks {
		_, err := client.SendMessage(ctx, jid, &waE2E.Message{
			Conversation: stringPtr(chunk),
		})
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}

	return nil
}

func (c *WhatsmeowChannel) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		c.handleIncomingMessage(v)
	case *events.Connected:
		logger.InfoC("whatsmeow", "WhatsApp connected")
	case *events.Disconnected:
		logger.WarnC("whatsmeow", "WhatsApp disconnected")
	case *events.LoggedOut:
		logger.ErrorCF("whatsmeow", "WhatsApp logged out", map[string]interface{}{
			"reason": v.Reason,
		})
		c.setRunning(false)
	}
}

func (c *WhatsmeowChannel) handleIncomingMessage(msg *events.Message) {
	// Skip status broadcasts
	if msg.Info.Chat.User == "status" {
		return
	}

	// Skip own messages
	if msg.Info.IsFromMe {
		return
	}

	// Extract sender phone number as senderID
	senderID := msg.Info.Sender.User

	// Full JID as chatID for routing responses
	chatID := msg.Info.Chat.String()

	// Extract text content
	content := ""
	if msg.Message.GetConversation() != "" {
		content = msg.Message.GetConversation()
	} else if msg.Message.GetExtendedTextMessage() != nil {
		content = msg.Message.GetExtendedTextMessage().GetText()
	}

	// Handle media
	var mediaPaths []string

	if img := msg.Message.GetImageMessage(); img != nil {
		if path, err := c.downloadMedia(img, ".jpg"); err == nil {
			mediaPaths = append(mediaPaths, path)
		}
		if caption := img.GetCaption(); caption != "" && content == "" {
			content = caption
		}
	}

	if audio := msg.Message.GetAudioMessage(); audio != nil {
		if path, err := c.downloadMedia(audio, ".ogg"); err == nil {
			mediaPaths = append(mediaPaths, path)
		}
	}

	if doc := msg.Message.GetDocumentMessage(); doc != nil {
		ext := ".bin"
		if fn := doc.GetFileName(); fn != "" {
			ext = filepath.Ext(fn)
		}
		if path, err := c.downloadMedia(doc, ext); err == nil {
			mediaPaths = append(mediaPaths, path)
		}
	}

	if video := msg.Message.GetVideoMessage(); video != nil {
		if path, err := c.downloadMedia(video, ".mp4"); err == nil {
			mediaPaths = append(mediaPaths, path)
		}
	}

	// Skip if no content and no media
	if content == "" && len(mediaPaths) == 0 {
		return
	}

	// Build metadata
	metadata := map[string]string{
		"message_id": msg.Info.ID,
	}
	if msg.Info.PushName != "" {
		metadata["push_name"] = msg.Info.PushName
	}
	if msg.Info.IsGroup {
		metadata["peer_kind"] = "group"
		metadata["peer_id"] = msg.Info.Chat.User
	} else {
		metadata["peer_kind"] = "dm"
		metadata["peer_id"] = msg.Info.Sender.User
	}

	logger.InfoCF("whatsmeow", "Message received", map[string]interface{}{
		"sender": senderID,
		"chat":   chatID,
		"len":    len(content),
	})

	c.HandleMessage(senderID, chatID, content, mediaPaths, metadata)
}

// downloadMedia downloads a WhatsApp media message to a temporary file.
func (c *WhatsmeowChannel) downloadMedia(msg whatsmeow.DownloadableMessage, ext string) (string, error) {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil {
		return "", fmt.Errorf("client not connected")
	}

	data, err := client.Download(context.Background(), msg)
	if err != nil {
		logger.ErrorCF("whatsmeow", "Failed to download media", map[string]interface{}{
			"error": err.Error(),
		})
		return "", err
	}

	dir := filepath.Join(os.TempDir(), "picoclaw_media")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("wa_%d%s", time.Now().UnixNano(), ext)
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}

	return path, nil
}

func stringPtr(s string) *string {
	return &s
}

func expandHomePath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
