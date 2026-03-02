//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

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

	metadata := map[string]string{}
	messageID := ""
	if mid := stringValue(message.MessageId); mid != "" {
		messageID = mid
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

	// 处理媒体消息
	scope := messageID

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

	content, mediaPaths, err := extractFeishuMessageContent(ctx, c, message, scope, storeMedia)
	if err != nil {
		logger.ErrorCF("feishu", "Failed to extract message content", map[string]any{
			"sender_id": senderID,
			"chat_id":   chatID,
			"error":     err.Error(),
		})
		c.Send(ctx, bus.OutboundMessage{
			ChatID:  chatID,
			Content: fmt.Sprintf("消息处理失败: %s", err.Error()),
		})
		return nil
	}

	if content == "" {
		content = "[empty message]"
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

func extractFeishuMessageContent(ctx context.Context, c *FeishuChannel, message *larkim.EventMessage, scope string, storeMedia func(string, string) string) (string, []string, error) {
	if message == nil || message.Content == nil || *message.Content == "" {
		return "", nil, fmt.Errorf("empty message")
	}

	var content string
	var mediaPaths []string

	if message.MessageType == nil {
		return *message.Content, nil, nil
	}

	msgType := *message.MessageType

	switch msgType {
	case larkim.MsgTypeText:
		var textPayload struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(*message.Content), &textPayload); err == nil {
			content = textPayload.Text
		} else {
			content = *message.Content
		}

	case larkim.MsgTypePost:
		if message.MessageId == nil {
			return "", nil, fmt.Errorf("消息ID为空")
		}
		var err error
		content, err = extractPostContent(ctx, c, *message.MessageId, *message.Content, &mediaPaths, storeMedia)
		if err != nil {
			return "", nil, fmt.Errorf("解析富文本失败: %w", err)
		}
		if content == "" {
			content = "[post]"
		}

	case larkim.MsgTypeImage:
		var imagePayload struct {
			ImageKey string `json:"image_key"`
		}
		if err := json.Unmarshal([]byte(*message.Content), &imagePayload); err != nil {
			return "", nil, fmt.Errorf("解析图片消息失败: %w", err)
		}
		if imagePayload.ImageKey == "" {
			return "", nil, fmt.Errorf("图片key为空")
		}
		if message.MessageId == nil {
			return "", nil, fmt.Errorf("消息ID为空")
		}
		imagePath, err := c.downloadImage(ctx, *message.MessageId, imagePayload.ImageKey)
		if err != nil {
			return "", nil, fmt.Errorf("下载图片失败: %w", err)
		}
		if imagePath == "" {
			return "", nil, fmt.Errorf("图片下载失败: 下载结果为空")
		}
		mediaPaths = append(mediaPaths, storeMedia(imagePath, "image.jpg"))
		content = "[image]"

	case larkim.MsgTypeAudio:
		var audioPayload struct {
			FileKey string `json:"file_key"`
		}
		if err := json.Unmarshal([]byte(*message.Content), &audioPayload); err != nil {
			return "", nil, fmt.Errorf("解析语音消息失败: %w", err)
		}
		if audioPayload.FileKey == "" {
			return "", nil, fmt.Errorf("语音文件key为空")
		}
		if message.MessageId == nil {
			return "", nil, fmt.Errorf("消息ID为空")
		}
		audioPath, err := c.downloadFile(ctx, *message.MessageId, audioPayload.FileKey)
		if err != nil {
			return "", nil, fmt.Errorf("下载语音失败: %w", err)
		}
		if audioPath == "" {
			return "", nil, fmt.Errorf("语音下载失败: 下载结果为空")
		}
		mediaPaths = append(mediaPaths, storeMedia(audioPath, "audio.amr"))
		content = "[audio]"

	case larkim.MsgTypeFile:
		var filePayload struct {
			FileKey string `json:"file_key"`
		}
		if err := json.Unmarshal([]byte(*message.Content), &filePayload); err != nil {
			return "", nil, fmt.Errorf("解析文件消息失败: %w", err)
		}
		if filePayload.FileKey == "" {
			return "", nil, fmt.Errorf("文件key为空")
		}
		if message.MessageId == nil {
			return "", nil, fmt.Errorf("消息ID为空")
		}
		filePath, err := c.downloadFile(ctx, *message.MessageId, filePayload.FileKey)
		if err != nil {
			return "", nil, fmt.Errorf("下载文件失败: %w", err)
		}
		if filePath == "" {
			return "", nil, fmt.Errorf("文件下载失败: 下载结果为空")
		}
		mediaPaths = append(mediaPaths, storeMedia(filePath, "file"))
		content = "[file]"

	case larkim.MsgTypeMedia:
		var mediaPayload struct {
			FileKey  string `json:"file_key"`
			ImageKey string `json:"image_key"`
		}
		if err := json.Unmarshal([]byte(*message.Content), &mediaPayload); err != nil {
			return "", nil, fmt.Errorf("解析视频消息失败: %w", err)
		}
		if mediaPayload.FileKey == "" && mediaPayload.ImageKey == "" {
			return "", nil, fmt.Errorf("视频文件key和图片key都为空")
		}
		if message.MessageId == nil {
			return "", nil, fmt.Errorf("消息ID为空")
		}
		messageId := *message.MessageId
		if mediaPayload.FileKey != "" {
			videoPath, err := c.downloadFile(ctx, messageId, mediaPayload.FileKey)
			if err != nil {
				return "", nil, fmt.Errorf("下载视频失败: %w", err)
			}
			if videoPath == "" {
				return "", nil, fmt.Errorf("视频下载失败: 下载结果为空")
			}
			mediaPaths = append(mediaPaths, storeMedia(videoPath, "video.mp4"))
		}
		if mediaPayload.ImageKey != "" {
			imagePath, err := c.downloadImage(ctx, messageId, mediaPayload.ImageKey)
			if err != nil {
				return "", nil, fmt.Errorf("下载视频封面失败: %w", err)
			}
			if imagePath == "" {
				return "", nil, fmt.Errorf("视频封面下载失败: 下载结果为空")
			}
			mediaPaths = append(mediaPaths, storeMedia(imagePath, "video_cover.jpg"))
		}
		content = "[video]"

	default:
		content = *message.Content
	}

	return content, mediaPaths, nil
}

func extractPostContent(ctx context.Context, c *FeishuChannel, messageId string, contentStr string, mediaPaths *[]string, storeMedia func(string, string) string) (string, error) {
	var postPayload struct {
		Title   string             `json:"title"`
		Content [][]map[string]any `json:"content"`
	}

	if err := json.Unmarshal([]byte(contentStr), &postPayload); err != nil {
		return "", fmt.Errorf("解析富文本内容失败: %w", err)
	}

	var textContent string

	if postPayload.Title != "" {
		textContent += postPayload.Title + "\n"
	}

	for _, paragraph := range postPayload.Content {
		paragraphText := ""
		for _, element := range paragraph {
			tag, ok := element["tag"].(string)
			if !ok {
				continue
			}
			switch tag {
			case "text":
				if text, ok := element["text"].(string); ok {
					paragraphText += text
				}
			case "a":
				if text, ok := element["text"].(string); ok {
					if href, ok := element["href"].(string); ok {
						paragraphText += "[" + text + "](" + href + ")"
					}
				}
			case "at":
				if name, ok := element["name"].(string); ok {
					paragraphText += "@" + name
				}
			case "img":
				if imageKey, ok := element["image_key"].(string); ok {
					imagePath, err := c.downloadImage(ctx, messageId, imageKey)
					if err != nil {
						return "", fmt.Errorf("下载富文本图片失败: %w", err)
					}
					if imagePath != "" {
						*mediaPaths = append(*mediaPaths, storeMedia(imagePath, "post_image.jpg"))
					}
					paragraphText += "[image]"
				}
			case "media":
				if fileKey, ok := element["file_key"].(string); ok {
					if fileKey != "" {
						videoPath, err := c.downloadFile(ctx, messageId, fileKey)
						if err != nil {
							return "", fmt.Errorf("下载富文本视频失败: %w", err)
						}
						if videoPath != "" {
							*mediaPaths = append(*mediaPaths, storeMedia(videoPath, "post_video.mp4"))
						}
					}
				}
				if imageKey, ok := element["image_key"].(string); ok {
					if imageKey != "" {
						imagePath, err := c.downloadImage(ctx, messageId, imageKey)
						if err != nil {
							return "", fmt.Errorf("下载富文本视频封面失败: %w", err)
						}
						if imagePath != "" {
							*mediaPaths = append(*mediaPaths, storeMedia(imagePath, "post_video_cover.jpg"))
						}
					}
				}
				paragraphText += "[video]"
			case "emotion":
				if emoji, ok := element["emoji"].(string); ok {
					paragraphText += emoji
				}
			case "code_block":
				if lang, ok := element["lang"].(string); ok {
					if code, ok := element["code"].(string); ok {
						paragraphText += "```" + lang + "\n" + code + "\n```"
					}
				}
			case "hr":
				paragraphText += "---"
			}
		}
		if paragraphText != "" {
			textContent += paragraphText + "\n"
		}
	}

	return textContent, nil
}

func (c *FeishuChannel) downloadImage(ctx context.Context, messageId, imageKey string) (string, error) {
	if imageKey == "" {
		return "", fmt.Errorf("图片key为空")
	}
	if messageId == "" {
		return "", fmt.Errorf("消息ID为空")
	}

	logger.InfoCF("feishu", "Starting to download image", map[string]any{
		"message_id": messageId,
		"image_key":  imageKey,
	})

	ext := ".jpg"
	filename := filepath.Join("/tmp/picoclaw_media", "feishu_image_"+imageKey+ext)

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return "", fmt.Errorf("创建媒体目录失败: %w", err)
	}

	req := larkim.NewGetMessageResourceReqBuilder().
		MessageId(messageId).
		FileKey(imageKey).
		Type("image").
		Build()

	resp, err := c.client.Im.MessageResource.Get(ctx, req)
	if err != nil {
		return "", fmt.Errorf("下载图片请求失败: %w", err)
	}

	if !resp.Success() {
		errorMsg := fmt.Sprintf("下载图片失败 (code=%d msg=%s)", resp.Code, resp.Msg)

		switch resp.Code {
		case 234001:
			errorMsg = fmt.Sprintf("下载图片失败: 请求参数无效，请检查message_id和image_key是否匹配 (code=%d msg=%s)", resp.Code, resp.Msg)
		case 234003:
			errorMsg = fmt.Sprintf("下载图片失败: 该资源不属于当前消息 (code=%d msg=%s)", resp.Code, resp.Msg)
		case 234004:
			errorMsg = fmt.Sprintf("下载图片失败: 应用不在消息所在的群组中 (code=%d msg=%s)", resp.Code, resp.Msg)
		case 234005:
			errorMsg = fmt.Sprintf("下载图片失败: 图片已被删除 (code=%d msg=%s)", resp.Code, resp.Msg)
		}

		return "", fmt.Errorf("%s", errorMsg)
	}

	if err := resp.WriteFile(filename); err != nil {
		return "", fmt.Errorf("写入图片文件失败: %w", err)
	}

	logger.InfoCF("feishu", "Image downloaded successfully", map[string]any{
		"message_id": messageId,
		"image_key":  imageKey,
		"path":       filename,
	})

	return filename, nil
}

func (c *FeishuChannel) downloadFile(ctx context.Context, messageId, fileKey string) (string, error) {
	if fileKey == "" {
		return "", fmt.Errorf("文件key为空")
	}
	if messageId == "" {
		return "", fmt.Errorf("消息ID为空")
	}

	logger.InfoCF("feishu", "Starting to download file", map[string]any{
		"message_id": messageId,
		"file_key":   fileKey,
	})

	ext := ".file"
	filename := filepath.Join("/tmp/picoclaw_media", "feishu_file_"+fileKey+ext)

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return "", fmt.Errorf("创建媒体目录失败: %w", err)
	}

	req := larkim.NewGetMessageResourceReqBuilder().
		MessageId(messageId).
		FileKey(fileKey).
		Type("file").
		Build()

	resp, err := c.client.Im.MessageResource.Get(ctx, req)
	if err != nil {
		return "", fmt.Errorf("下载文件请求失败: %w", err)
	}

	if !resp.Success() {
		errorMsg := fmt.Sprintf("下载文件失败 (code=%d msg=%s)", resp.Code, resp.Msg)

		switch resp.Code {
		case 234001:
			errorMsg = fmt.Sprintf("下载文件失败: 请求参数无效，请检查message_id和file_key是否匹配 (code=%d msg=%s)", resp.Code, resp.Msg)
		case 234003:
			errorMsg = fmt.Sprintf("下载文件失败: 该资源不属于当前消息 (code=%d msg=%s)", resp.Code, resp.Msg)
		case 234004:
			errorMsg = fmt.Sprintf("下载文件失败: 应用不在消息所在的群组中 (code=%d msg=%s)", resp.Code, resp.Msg)
		case 234005:
			errorMsg = fmt.Sprintf("下载文件失败: 文件已被删除 (code=%d msg=%s)", resp.Code, resp.Msg)
		}

		return "", fmt.Errorf("%s", errorMsg)
	}

	if err := resp.WriteFile(filename); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	logger.InfoCF("feishu", "File downloaded successfully", map[string]any{
		"message_id": messageId,
		"file_key":   fileKey,
		"path":       filename,
	})

	return filename, nil
}
