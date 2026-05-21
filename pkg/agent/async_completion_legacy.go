package agent

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
)

// Legacy synthetic system-message metadata. New async completion producers
// should call processAsyncCompletion with AsyncCompletionInput directly; these
// helpers only keep older queued/stored system messages readable.
const (
	systemFollowUpOriginChannelKey          = "origin_channel"
	systemFollowUpOriginChatIDKey           = "origin_chat_id"
	systemFollowUpOriginChatTypeKey         = "origin_chat_type"
	systemFollowUpOriginTopicIDKey          = "origin_topic_id"
	systemFollowUpOriginMessageIDKey        = "origin_message_id"
	systemFollowUpOriginReplyToMessageIDKey = "origin_reply_to_message_id"
	systemFollowUpKindKey                   = "kind"
	systemFollowUpKindAsyncCompletion       = "async_completion"
	systemFollowUpIDKey                     = "completion_id"
)

func systemFollowUpOriginRaw(origin *bus.InboundContext, channel, chatID string) map[string]string {
	raw := map[string]string{
		systemFollowUpOriginChannelKey: strings.TrimSpace(channel),
		systemFollowUpOriginChatIDKey:  strings.TrimSpace(chatID),
	}
	if origin == nil {
		return raw
	}
	if origin.Channel != "" {
		raw[systemFollowUpOriginChannelKey] = strings.TrimSpace(origin.Channel)
	}
	if origin.ChatID != "" {
		raw[systemFollowUpOriginChatIDKey] = strings.TrimSpace(origin.ChatID)
	}
	if origin.ChatType != "" {
		raw[systemFollowUpOriginChatTypeKey] = strings.TrimSpace(origin.ChatType)
	}
	if origin.TopicID != "" {
		raw[systemFollowUpOriginTopicIDKey] = strings.TrimSpace(origin.TopicID)
	}
	if origin.MessageID != "" {
		raw[systemFollowUpOriginMessageIDKey] = strings.TrimSpace(origin.MessageID)
	}
	if origin.ReplyToMessageID != "" {
		raw[systemFollowUpOriginReplyToMessageIDKey] = strings.TrimSpace(origin.ReplyToMessageID)
	}
	return raw
}

func systemFollowUpAsyncCompletionRaw(origin *bus.InboundContext, channel, chatID, completionID string) map[string]string {
	raw := systemFollowUpOriginRaw(origin, channel, chatID)
	raw[systemFollowUpKindKey] = systemFollowUpKindAsyncCompletion
	if strings.TrimSpace(completionID) != "" {
		raw[systemFollowUpIDKey] = strings.TrimSpace(completionID)
	}
	return raw
}

func isAsyncCompletionSystemMessage(msg bus.InboundMessage) bool {
	if msg.Channel != "system" {
		return false
	}
	if msg.Context.Raw == nil {
		return false
	}
	return strings.TrimSpace(msg.Context.Raw[systemFollowUpKindKey]) == systemFollowUpKindAsyncCompletion
}
