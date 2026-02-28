//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	larkevent "github.com/larksuite/oapi-sdk-go/v3/event"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkdispatcher "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

var feishuMarkdownImageRe = regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`)

type FeishuChannel struct {
	*BaseChannel
	config   config.FeishuConfig
	client   *lark.Client
	wsClient *larkws.Client

	mu     sync.Mutex
	cancel context.CancelFunc
}

type feishuEventEnvelope struct {
	Header *struct {
		EventType string `json:"event_type"`
	} `json:"header"`
	Event *struct {
		Sender  *larkim.EventSender  `json:"sender"`
		Message *larkim.EventMessage `json:"message"`
	} `json:"event"`
}

func NewFeishuChannel(cfg config.FeishuConfig, bus *bus.MessageBus) (*FeishuChannel, error) {
	base := NewBaseChannel("feishu", cfg, bus, cfg.AllowFrom)

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
		OnCustomizedEvent("im.message.receive_v1", c.handleMessageReceiveRaw).
		OnCustomizedEvent("im.message.receive_v2", c.handleMessageReceiveRaw)

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

	c.setRunning(true)
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

	c.setRunning(false)
	logger.InfoC("feishu", "Feishu channel stopped")
	return nil
}

func (c *FeishuChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("feishu channel not running")
	}

	if msg.ChatID == "" {
		return fmt.Errorf("chat ID is empty")
	}

	textContent, imagePaths := splitFeishuOutboundContent(msg.Content)

	if textContent == "" && len(imagePaths) == 0 {
		return nil
	}

	if textContent != "" {
		if err := c.sendFeishuTextMessage(ctx, msg.ChatID, textContent); err != nil {
			return err
		}
	}

	for _, imagePath := range imagePaths {
		imageKey, err := c.uploadFeishuImage(ctx, imagePath)
		if err != nil {
			return fmt.Errorf("failed to upload feishu image %q: %w", imagePath, err)
		}

		if err := c.sendFeishuImageMessage(ctx, msg.ChatID, imageKey); err != nil {
			return fmt.Errorf("failed to send feishu image %q: %w", imagePath, err)
		}

		logger.DebugCF("feishu", "Feishu image sent", map[string]any{
			"chat_id":   msg.ChatID,
			"image_path": imagePath,
		})
	}

	return nil
}

func (c *FeishuChannel) sendFeishuTextMessage(ctx context.Context, chatID, content string) error {
	payload, err := json.Marshal(map[string]string{"text": content})
	if err != nil {
		return fmt.Errorf("failed to marshal feishu text content: %w", err)
	}

	if err := c.sendFeishuMessage(ctx, chatID, larkim.MsgTypeText, string(payload)); err != nil {
		return err
	}

	logger.DebugCF("feishu", "Feishu text message sent", map[string]any{
		"chat_id": chatID,
	})

	return nil
}

func (c *FeishuChannel) sendFeishuImageMessage(ctx context.Context, chatID, imageKey string) error {
	payload, err := json.Marshal(map[string]string{"image_key": imageKey})
	if err != nil {
		return fmt.Errorf("failed to marshal feishu image content: %w", err)
	}

	return c.sendFeishuMessage(ctx, chatID, larkim.MsgTypeImage, string(payload))
}

func (c *FeishuChannel) sendFeishuMessage(ctx context.Context, chatID, msgType, content string) error {

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(msgType).
			Content(content).
			Uuid(fmt.Sprintf("picoclaw-%d", time.Now().UnixNano())).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send feishu message: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("feishu api error: code=%d msg=%s", resp.Code, resp.Msg)
	}

	return nil
}

func (c *FeishuChannel) uploadFeishuImage(ctx context.Context, imagePath string) (string, error) {
	body, err := larkim.NewCreateImagePathReqBodyBuilder().
		ImageType("message").
		ImagePath(imagePath).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}

	req := larkim.NewCreateImageReqBuilder().
		Body(body).
		Build()

	resp, err := c.client.Im.V1.Image.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	if !resp.Success() {
		return "", fmt.Errorf("feishu image api error: code=%d msg=%s", resp.Code, resp.Msg)
	}

	if resp.Data == nil || resp.Data.ImageKey == nil || *resp.Data.ImageKey == "" {
		return "", fmt.Errorf("feishu image upload succeeded but image_key is empty")
	}

	return *resp.Data.ImageKey, nil
}

func (c *FeishuChannel) handleMessageReceiveRaw(_ context.Context, event *larkevent.EventReq) error {
	if event == nil || len(event.Body) == 0 {
		logger.WarnC("feishu", "Received empty custom Feishu event payload")
		return nil
	}

	var envelope feishuEventEnvelope
	if err := json.Unmarshal(event.Body, &envelope); err != nil {
		logger.ErrorCF("feishu", "Failed to parse Feishu message event payload", map[string]any{
			"error":   err.Error(),
			"payload": utils.Truncate(string(event.Body), 300),
		})
		return err
	}

	if envelope.Event == nil || envelope.Event.Message == nil {
		eventType := ""
		if envelope.Header != nil {
			eventType = envelope.Header.EventType
		}
		logger.DebugCF("feishu", "Ignored Feishu event without message body", map[string]any{
			"event_type": eventType,
		})
		return nil
	}

	message := envelope.Event.Message
	sender := envelope.Event.Sender

	chatID := stringValue(message.ChatId)
	if chatID == "" {
		return nil
	}

	senderID := extractFeishuSenderID(sender)
	if senderID == "" {
		senderID = "unknown"
	}

	content := extractFeishuMessageContent(message)
	if content == "" {
		content = "[empty message]"
	}

	metadata := map[string]string{}
	if messageID := stringValue(message.MessageId); messageID != "" {
		metadata["message_id"] = messageID
	}
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
	if chatType == "p2p" {
		metadata["peer_kind"] = "direct"
		metadata["peer_id"] = senderID
	} else {
		metadata["peer_kind"] = "group"
		metadata["peer_id"] = chatID
	}

	logger.InfoCF("feishu", "Feishu message received", map[string]any{
		"sender_id": senderID,
		"chat_id":   chatID,
		"preview":   utils.Truncate(content, 80),
	})

	c.HandleMessage(senderID, chatID, content, nil, metadata)
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

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func splitFeishuOutboundContent(content string) (string, []string) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", nil
	}

	imagePaths := make([]string, 0, 2)

	cleaned := feishuMarkdownImageRe.ReplaceAllStringFunc(trimmed, func(segment string) string {
		matches := feishuMarkdownImageRe.FindStringSubmatch(segment)
		if len(matches) < 2 {
			return segment
		}

		if imagePath, ok := normalizeFeishuLocalImagePath(matches[1]); ok {
			imagePaths = append(imagePaths, imagePath)
			return ""
		}

		return segment
	})

	cleaned = strings.TrimSpace(cleaned)

	if len(imagePaths) == 0 {
		if imagePath, ok := normalizeFeishuLocalImagePath(cleaned); ok {
			return "", []string{imagePath}
		}
	}

	if len(imagePaths) == 0 {
		return cleaned, nil
	}

	return cleaned, dedupeStrings(imagePaths)
}

func normalizeFeishuLocalImagePath(candidate string) (string, bool) {
	candidate = strings.TrimSpace(candidate)
	candidate = strings.Trim(candidate, "\"'")
	candidate = strings.Trim(candidate, "<>")
	if candidate == "" {
		return "", false
	}

	if strings.HasPrefix(strings.ToLower(candidate), "file://") {
		parsed, err := url.Parse(candidate)
		if err != nil || parsed.Scheme != "file" {
			return "", false
		}
		if parsed.Host != "" && parsed.Host != "localhost" {
			return "", false
		}

		path, err := url.PathUnescape(parsed.Path)
		if err != nil {
			return "", false
		}
		candidate = path
	}

	if candidate == "" {
		return "", false
	}

	absPath, err := filepath.Abs(candidate)
	if err != nil {
		return "", false
	}

	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		return "", false
	}

	if !isFeishuSupportedImageFile(absPath) {
		return "", false
	}

	return absPath, true
}

func isFeishuSupportedImageFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".tiff", ".tif", ".bmp", ".ico":
		return true
	default:
		return false
	}
}

func dedupeStrings(values []string) []string {
	if len(values) <= 1 {
		return values
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
