package line

import (
	"context"
	"fmt"

	"jane/pkg/bus"
	"jane/pkg/channels"
	"jane/pkg/utils"
)

// SendMedia implements the channels.MediaSender interface.
// LINE requires media to be accessible via public URL; since we only have local files,
// we fall back to sending a text message with the filename/caption.
// For full support, an external file hosting service would be needed.
func (c *LINEChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	store := c.GetMediaStore()
	if store == nil {
		return fmt.Errorf("no media store available: %w", channels.ErrSendFailed)
	}

	// LINE Messaging API requires publicly accessible URLs for media messages.
	// Since we only have local file paths, send caption text as fallback.
	for _, part := range msg.Parts {
		caption := part.Caption
		if caption == "" {
			caption = fmt.Sprintf("[%s: %s]", part.Type, part.Filename)
		}

		if err := c.sendPush(ctx, msg.ChatID, caption, ""); err != nil {
			return err
		}
	}

	return nil
}

// downloadContent downloads media content from the LINE API.
func (c *LINEChannel) downloadContent(messageID, filename string) string {
	url := fmt.Sprintf(lineContentEndpoint, messageID)
	return utils.DownloadFile(url, filename, utils.DownloadOptions{
		LoggerPrefix: "line",
		ExtraHeaders: map[string]string{
			"Authorization": "Bearer " + c.config.ChannelAccessToken,
		},
	})
}
