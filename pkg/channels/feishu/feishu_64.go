//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkdispatcher "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type FeishuChannel struct {
	*channels.BaseChannel
	config   config.FeishuConfig
	client   *lark.Client
	wsClient *larkws.Client

	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewFeishuChannel(cfg config.FeishuConfig, bus *bus.MessageBus) (*FeishuChannel, error) {
	base := channels.NewBaseChannel("feishu", cfg, bus, cfg.AllowFrom,
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &FeishuChannel{
		BaseChannel: base,
		config:      cfg,
		client:      lark.NewClient(cfg.AppID, cfg.AppSecret),
	}, nil
}

func (c *FeishuChannel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.AppSecret == "" {
		return fmt.Errorf("feishu app_id or app_secret is empty")
	}

	dispatcher := larkdispatcher.NewEventDispatcher(c.config.VerificationToken, c.config.EncryptKey).
		OnP2MessageReceiveV1(c.handleMessageReceive)

	runCtx, cancel := context.WithCancel(ctx)

	c.mu.Lock()
	c.cancel = cancel
	c.wsClient = larkws.NewClient(
		c.config.AppID,
		c.config.AppSecret,
		larkws.WithEventHandler(dispatcher),
	)
	wsClient := c.wsClient
	c.mu.Unlock()

	c.SetRunning(true)
	logger.InfoC("feishu", "Feishu channel started (websocket mode)")

	go func() {
		if err := wsClient.Start(runCtx); err != nil {
			logger.ErrorCF("feishu", "Feishu websocket stopped with error", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	return nil
}

func (c *FeishuChannel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.wsClient = nil
	c.mu.Unlock()

	c.SetRunning(false)
	logger.InfoC("feishu", "Feishu channel stopped")
	return nil
}

func (c *FeishuChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	if msg.ChatID == "" {
		return fmt.Errorf("chat ID is empty")
	}

	payload, err := json.Marshal(map[string]string{"text": msg.Content})
	if err != nil {
		return fmt.Errorf("failed to marshal feishu content: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(msg.ChatID).
			MsgType(larkim.MsgTypeText).
			Content(string(payload)).
			Uuid(fmt.Sprintf("picoclaw-%d", time.Now().UnixNano())).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu send: %w", channels.ErrTemporary)
	}

	if !resp.Success() {
		return fmt.Errorf("feishu api error (code=%d msg=%s): %w", resp.Code, resp.Msg, channels.ErrTemporary)
	}

	logger.DebugCF("feishu", "Feishu message sent", map[string]any{
		"chat_id": msg.ChatID,
	})

	return nil
}

func (c *FeishuChannel) handleMessageReceive(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil
	}

	message := event.Event.Message
	sender := event.Event.Sender

	chatID := stringValue(message.ChatId)
	if chatID == "" {
		return nil
	}

	senderID := extractFeishuSenderID(sender)
	if senderID == "" {
		senderID = "unknown"
	}

	content := extractFeishuMessageContent(message)

	messageID := stringValue(message.MessageId)
	scope := channels.BuildMediaScope("feishu", chatID, messageID)

	storeMedia := func(localPath, filename string) string {
		if store := c.GetMediaStore(); store != nil {
			ref, err := store.Store(localPath, media.MediaMeta{
				Filename: filename,
				Source:   "feishu",
			}, scope)
			if err == nil {
				return ref
			}
		}
		return localPath
	}

	var mediaPaths []string

	msgType := stringValue(message.MessageType)
	switch msgType {
	case larkim.MsgTypeImage:
		// Pure image message: {"image_key":"..."}
		if imageKey := extractFeishuImageKey(stringValue(message.Content)); imageKey != "" {
			localPath, err := c.downloadImage(ctx, messageID, imageKey)
			if err != nil {
				logger.ErrorCF("feishu", "Failed to download image", map[string]any{
					"image_key": imageKey,
					"error":     err.Error(),
				})
			} else if localPath != "" {
				mediaPaths = append(mediaPaths, storeMedia(localPath, imageKey))
				if content != "" {
					content += "\n"
				}
				content += "[image: photo]"
			}
		}
	case larkim.MsgTypePost:
		// Rich text (post) message: {"title":"...","content":[[{"tag":"img","image_key":"..."},{"tag":"text","text":"..."}]]}
		for _, imageKey := range extractFeishuPostImageKeys(stringValue(message.Content)) {
			localPath, err := c.downloadImage(ctx, messageID, imageKey)
			if err != nil {
				logger.ErrorCF("feishu", "Failed to download post image", map[string]any{
					"image_key": imageKey,
					"error":     err.Error(),
				})
				continue
			}
			if localPath != "" {
				mediaPaths = append(mediaPaths, storeMedia(localPath, imageKey))
			}
		}
	}

	if content == "" {
		content = "[empty message]"
	}

	metadata := map[string]string{}
	if messageType := stringValue(message.MessageType); messageType != "" {
		metadata["message_type"] = messageType
	}
	if chatType := stringValue(message.ChatType); chatType != "" {
		metadata["chat_type"] = chatType
	}
	if sender != nil && sender.TenantKey != nil {
		metadata["tenant_key"] = *sender.TenantKey
	}

	chatType := stringValue(message.ChatType)
	var peer bus.Peer
	if chatType == "p2p" {
		peer = bus.Peer{Kind: "direct", ID: senderID}
	} else {
		peer = bus.Peer{Kind: "group", ID: chatID}
		// In group chats, apply unified group trigger filtering
		respond, cleaned := c.ShouldRespondInGroup(false, content)
		if !respond {
			return nil
		}
		content = cleaned
	}

	logger.InfoCF("feishu", "Feishu message received", map[string]any{
		"sender_id": senderID,
		"chat_id":   chatID,
		"preview":   utils.Truncate(content, 80),
	})

	senderInfo := bus.SenderInfo{
		Platform:    "feishu",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("feishu", senderID),
	}

	if !c.IsAllowedSender(senderInfo) {
		return nil
	}

	c.HandleMessage(ctx, peer, messageID, senderID, chatID, content, mediaPaths, metadata, senderInfo)
	return nil
}

func extractFeishuSenderID(sender *larkim.EventSender) string {
	if sender == nil || sender.SenderId == nil {
		return ""
	}

	if sender.SenderId.UserId != nil && *sender.SenderId.UserId != "" {
		return *sender.SenderId.UserId
	}
	if sender.SenderId.OpenId != nil && *sender.SenderId.OpenId != "" {
		return *sender.SenderId.OpenId
	}
	if sender.SenderId.UnionId != nil && *sender.SenderId.UnionId != "" {
		return *sender.SenderId.UnionId
	}

	return ""
}

func extractFeishuMessageContent(message *larkim.EventMessage) string {
	if message == nil || message.Content == nil || *message.Content == "" {
		return ""
	}

	if message.MessageType != nil && *message.MessageType == larkim.MsgTypeText {
		var textPayload struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(*message.Content), &textPayload); err == nil {
			return textPayload.Text
		}
	}

	return *message.Content
}

func extractFeishuImageKey(content string) string {
	var payload struct {
		ImageKey string `json:"image_key"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return ""
	}
	return payload.ImageKey
}

// extractFeishuPostImageKeys extracts all image_key values from a post (rich text) message.
// Post format: {"title":"...","content":[[{"tag":"img","image_key":"..."},{"tag":"text","text":"..."}]]}
func extractFeishuPostImageKeys(content string) []string {
	var payload struct {
		Content [][]struct {
			Tag      string `json:"tag"`
			ImageKey string `json:"image_key"`
		} `json:"content"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil
	}
	var keys []string
	for _, line := range payload.Content {
		for _, elem := range line {
			if elem.Tag == "img" && elem.ImageKey != "" {
				keys = append(keys, elem.ImageKey)
			}
		}
	}
	return keys
}

func (c *FeishuChannel) downloadImage(ctx context.Context, messageID, imageKey string) (string, error) {
	req := larkim.NewGetMessageResourceReqBuilder().
		MessageId(messageID).
		FileKey(imageKey).
		Type("image").
		Build()
	resp, err := c.client.Im.V1.MessageResource.Get(ctx, req)
	if err != nil {
		return "", fmt.Errorf("feishu message resource get: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("feishu message resource get: code=%d msg=%s", resp.Code, resp.Msg)
	}

	mediaDir := filepath.Join(os.TempDir(), "picoclaw_media")
	if mkErr := os.MkdirAll(mediaDir, 0o700); mkErr != nil {
		return "", fmt.Errorf("create media dir: %w", mkErr)
	}

	localPath := filepath.Join(mediaDir, uuid.New().String()[:8]+"_feishu_image")
	out, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.File); err != nil {
		out.Close()
		os.Remove(localPath)
		return "", fmt.Errorf("write image: %w", err)
	}

	logger.DebugCF("feishu", "Image downloaded", map[string]any{
		"image_key": imageKey,
		"path":      localPath,
	})
	return localPath, nil
}
