package agent

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/providers"
)

func messagesContainMedia(messages []providers.Message) bool {
	for _, msg := range messages {
		for _, ref := range msg.Media {
			if strings.TrimSpace(ref) != "" {
				return true
			}
		}
	}
	return false
}

func stripMessageMedia(messages []providers.Message) []providers.Message {
	if !messagesContainMedia(messages) {
		return messages
	}
	stripped := make([]providers.Message, len(messages))
	for i, msg := range messages {
		stripped[i] = msg
		stripped[i].Media = nil
	}
	return stripped
}

func callLLMWithVisionUnsupportedRetry(
	messages []providers.Message,
	call func([]providers.Message) (*providers.LLMResponse, error),
	beforeRetry func(error),
) (*providers.LLMResponse, []providers.Message, bool, error) {
	response, err := call(messages)
	if err == nil {
		return response, messages, false, nil
	}
	if !messagesContainMedia(messages) || !isVisionUnsupportedError(err) {
		return response, messages, false, err
	}

	if beforeRetry != nil {
		beforeRetry(err)
	}
	stripped := stripMessageMedia(messages)
	response, err = call(stripped)
	return response, stripped, true, err
}

func isVisionUnsupportedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())

	// OpenRouter (and OpenAI-compatible) style.
	if strings.Contains(msg, "no endpoints found that support image input") {
		return true
	}

	// Common provider variants.
	if strings.Contains(msg, "does not support image input") ||
		strings.Contains(msg, "does not support image inputs") ||
		strings.Contains(msg, "does not support images") ||
		strings.Contains(msg, "image input is not supported") ||
		strings.Contains(msg, "images are not supported") ||
		strings.Contains(msg, "does not support vision") ||
		strings.Contains(msg, "unsupported content type: image_url") {
		return true
	}

	// Some providers return a generic "invalid" message that still mentions image_url.
	if strings.Contains(msg, "image_url") && strings.Contains(msg, "invalid") {
		return true
	}

	return false
}
