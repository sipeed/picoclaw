package line

import (
	"context"
	"strings"
	"sync"
	"time"

	"jane/pkg/logger"
)

// StartTyping implements channels.TypingCapable using LINE's loading animation.
//
// NOTE: The LINE loading animation API only works for 1:1 chats.
// Group/room chat IDs (starting with "C" or "R") are detected automatically;
// for these, a no-op stop function is returned without calling the API.
func (c *LINEChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	if chatID == "" {
		return func() {}, nil
	}

	// Group/room chats: LINE loading animation is 1:1 only.
	if strings.HasPrefix(chatID, "C") || strings.HasPrefix(chatID, "R") {
		return func() {}, nil
	}

	typingCtx, cancel := context.WithCancel(ctx)
	var once sync.Once
	stop := func() { once.Do(cancel) }

	// Send immediately, then refresh periodically for long-running tasks.
	if err := c.sendLoading(typingCtx, chatID); err != nil {
		stop()
		return stop, err
	}

	ticker := time.NewTicker(50 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				if err := c.sendLoading(typingCtx, chatID); err != nil {
					logger.DebugCF("line", "Failed to refresh loading indicator", map[string]any{
						"error": err.Error(),
					})
				}
			}
		}
	}()

	return stop, nil
}

// sendLoading sends a loading animation indicator to the chat.
func (c *LINEChannel) sendLoading(ctx context.Context, chatID string) error {
	payload := map[string]any{
		"chatId":         chatID,
		"loadingSeconds": 60,
	}
	return c.callAPI(ctx, lineLoadingEndpoint, payload)
}
