//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package feishu

import (
	"context"
	"fmt"
	"strings"
	"time"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const messageCacheTTL = 30 * time.Second

const (
	maxReplyContextLen = 600
)

func (c *FeishuChannel) prependReplyContext(
	ctx context.Context,
	message *larkim.EventMessage,
	chatID string,
	content string,
	mediaRefs []string,
) (string, []string) {
	if message == nil {
		return content, mediaRefs
	}

	lookupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	targetMessageID := c.resolveReplyTargetMessageID(lookupCtx, message)
	if targetMessageID == "" {
		logger.DebugCF("feishu", "No reply target resolved; skip reply context", map[string]any{
			"message_id": stringValue(message.MessageId),
			"parent_id":  stringValue(message.ParentId),
			"root_id":    stringValue(message.RootId),
			"thread_id":  stringValue(message.ThreadId),
		})
		return content, mediaRefs
	}

	repliedMessage, err := c.fetchMessageByID(lookupCtx, targetMessageID)
	if err != nil {
		logger.DebugCF("feishu", "Failed to fetch replied message context", map[string]any{
			"target_message_id": targetMessageID,
			"error":             err.Error(),
		})
		return content, mediaRefs
	}

	messageType := stringValue(repliedMessage.MsgType)
	rawContent := ""
	if repliedMessage.Body != nil {
		rawContent = stringValue(repliedMessage.Body.Content)
	}

	var repliedMediaRefs []string
	if store := c.GetMediaStore(); store != nil {
		repliedMediaRefs = c.downloadInboundMedia(lookupCtx, chatID, targetMessageID, messageType, rawContent, store)
		if messageType == larkim.MsgTypeInteractive {
			_, externalURLs := extractCardImageKeys(rawContent)
			if len(externalURLs) > 0 {
				repliedMediaRefs = append(repliedMediaRefs, externalURLs...)
			}
		}
	}

	repliedContent := normalizeRepliedContent(messageType, rawContent, repliedMediaRefs)
	if len(repliedMediaRefs) > 0 {
		mediaRefs = append(repliedMediaRefs, mediaRefs...)
	}

	return formatReplyContext(targetMessageID, repliedContent, content), mediaRefs
}

func (c *FeishuChannel) resolveReplyTargetMessageID(ctx context.Context, message *larkim.EventMessage) string {
	if targetID := replyTargetID(message); targetID != "" {
		logger.DebugCF("feishu", "Resolved reply target from event payload", map[string]any{
			"message_id": stringValue(message.MessageId),
			"parent_id":  stringValue(message.ParentId),
			"root_id":    stringValue(message.RootId),
			"target_id":  targetID,
		})
		return targetID
	}

	currentMessageID := stringValue(message.MessageId)
	if currentMessageID == "" {
		return ""
	}

	if stringValue(message.ThreadId) == "" {
		logger.DebugCF("feishu", "No reply target found; message is not in a thread", map[string]any{
			"message_id": stringValue(message.MessageId),
		})
		return ""
	}

	msg, err := c.fetchMessageByID(ctx, currentMessageID)
	if err != nil {
		logger.DebugCF("feishu", "Failed to query current message detail for reply info", map[string]any{
			"message_id": currentMessageID,
			"error":      err.Error(),
		})
		return ""
	}

	targetID := replyTargetIDFromMessage(msg)
	if targetID != "" {
		logger.DebugCF("feishu", "Resolved reply target from message detail", map[string]any{
			"message_id": currentMessageID,
			"parent_id":  stringValue(msg.ParentId),
			"root_id":    stringValue(msg.RootId),
			"target_id":  targetID,
		})
	}
	return targetID
}

func (c *FeishuChannel) fetchMessageByID(ctx context.Context, messageID string) (*larkim.Message, error) {
	if cached, ok := c.messageCache.Load(messageID); ok {
		cm := cached.(*cachedMessage)
		if time.Now().Before(cm.expiry) {
			return cm.msg, nil
		}
		c.messageCache.Delete(messageID)
	}

	req := larkim.NewGetMessageReqBuilder().
		MessageId(messageID).
		Build()

	resp, err := c.client.Im.V1.Message.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("feishu get message: %w", err)
	}
	if !resp.Success() {
		c.invalidateTokenOnAuthError(resp.Code)
		return nil, fmt.Errorf("feishu get message api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}
	if resp.Data == nil || len(resp.Data.Items) == 0 || resp.Data.Items[0] == nil {
		return nil, fmt.Errorf("feishu get message: empty response")
	}
	// Items[0] contains the target message - the Feishu API returns a list
	// but we request a single message by ID, so the list always has at most one item.
	msg := resp.Data.Items[0]
	c.messageCache.Store(messageID, &cachedMessage{msg: msg, expiry: time.Now().Add(messageCacheTTL)})
	return msg, nil
}

func replyTargetID(message *larkim.EventMessage) string {
	if message == nil {
		return ""
	}
	if parentID := stringValue(message.ParentId); parentID != "" {
		return parentID
	}
	return stringValue(message.RootId)
}

func replyTargetIDFromMessage(message *larkim.Message) string {
	if message == nil {
		return ""
	}
	if parentID := stringValue(message.ParentId); parentID != "" {
		return parentID
	}
	return stringValue(message.RootId)
}

func buildInboundMetadata(message *larkim.EventMessage, sender *larkim.EventSender) map[string]string {
	metadata := map[string]string{}
	if message == nil {
		return metadata
	}

	messageID := stringValue(message.MessageId)
	if messageID != "" {
		metadata["message_id"] = messageID
	}

	messageType := stringValue(message.MessageType)
	if messageType != "" {
		metadata["message_type"] = messageType
	}

	chatType := stringValue(message.ChatType)
	if chatType != "" {
		metadata["chat_type"] = chatType
	}

	parentID := stringValue(message.ParentId)
	if parentID != "" {
		metadata["parent_id"] = parentID
	}

	rootID := stringValue(message.RootId)
	if rootID != "" {
		metadata["root_id"] = rootID
	}

	if replyTo := replyTargetID(message); replyTo != "" {
		metadata["reply_to_message_id"] = replyTo
	}

	threadID := stringValue(message.ThreadId)
	if threadID != "" {
		metadata["thread_id"] = threadID
	}

	if sender != nil && sender.TenantKey != nil && *sender.TenantKey != "" {
		metadata["tenant_key"] = *sender.TenantKey
	}

	return metadata
}

func normalizeRepliedContent(messageType, rawContent string, mediaRefs []string) string {
	content := extractContent(messageType, rawContent)

	if containsFeishuUpgradePlaceholder(rawContent) || containsFeishuUpgradePlaceholder(content) {
		content = ""
	}

	content = appendMediaTags(content, messageType, mediaRefs)
	if strings.TrimSpace(content) != "" {
		return content
	}

	switch messageType {
	case larkim.MsgTypeImage:
		return "[replied image]"
	case larkim.MsgTypeFile:
		return "[replied file]"
	case larkim.MsgTypeAudio:
		return "[replied audio]"
	case larkim.MsgTypeMedia:
		return "[replied video]"
	case larkim.MsgTypeInteractive:
		return "[replied interactive card]"
	default:
		return "[replied message content unavailable]"
	}
}

func containsFeishuUpgradePlaceholder(s string) bool {
	upgradePrompt := "\u8bf7\u5347\u7ea7\u81f3\u6700\u65b0\u7248\u672c\u5ba2\u6237\u7aef"
	upgradePromptEscaped := "\\u8bf7\\u5347\\u7ea7\\u81f3\\u6700\\u65b0\\u7248\\u672c\\u5ba2\\u6237\\u7aef"
	return strings.Contains(s, upgradePrompt) || strings.Contains(s, upgradePromptEscaped)
}

func formatReplyContext(parentID, repliedContent, content string) string {
	parentID = strings.TrimSpace(parentID)
	repliedContent = strings.TrimSpace(repliedContent)
	content = strings.TrimSpace(content)

	if parentID == "" || repliedContent == "" {
		return content
	}

	repliedContent = utils.Truncate(repliedContent, maxReplyContextLen)
	repliedContent = sanitizeReplyContextContent(repliedContent)
	content = sanitizeReplyContextContent(content)
	header := fmt.Sprintf("[replied_message id=%q]", parentID)
	footer := "[/replied_message]"
	if content == "" {
		return header + "\n" + repliedContent + "\n" + footer
	}
	if hasLeadingCommandPrefix(content) {
		return content + "\n\n" + header + "\n" + repliedContent + "\n" + footer
	}
	return header + "\n" + repliedContent + "\n" + footer + "\n\n[current_message]\n" + content + "\n[/current_message]"
}

func hasLeadingCommandPrefix(s string) bool {
	tokens := strings.Fields(strings.TrimSpace(s))
	if len(tokens) == 0 {
		return false
	}
	first := tokens[0]
	return strings.HasPrefix(first, "/") || strings.HasPrefix(first, "!")
}

func sanitizeReplyContextContent(s string) string {
	tagEscaper := strings.NewReplacer(
		"[replied_message", `\[replied_message`,
		"[/replied_message]", `\[/replied_message]`,
		"[current_message]", `\[current_message]`,
		"[/current_message]", `\[/current_message]`,
	)
	return tagEscaper.Replace(s)
}
