//go:build !amd64 && !arm64 && !riscv64 && !mips64 && !ppc64

package feishu

import (
	"context"
	"errors"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

// FeishuChannel is a stub implementation for 32-bit architectures
type FeishuChannel struct {
	*channels.BaseChannel
}

var errUnsupported = errors.New("feishu channel is not supported on 32-bit architectures")

// NewFeishuChannel returns an error on 32-bit architectures where the Feishu SDK is not supported
func NewFeishuChannel(bc *config.Channel, cfg *config.FeishuSettings, bus *bus.MessageBus) (*FeishuChannel, error) {
	return nil, errors.New(
		"feishu channel is not supported on 32-bit architectures (armv7l, 386, etc.). Please use a 64-bit system or disable feishu in your config",
	)
}

// Start is a stub method to satisfy the Channel interface
func (c *FeishuChannel) Start(ctx context.Context) error {
	return errUnsupported
}

// Stop is a stub method to satisfy the Channel interface
func (c *FeishuChannel) Stop(ctx context.Context) error {
	return errUnsupported
}

// Send is a stub method to satisfy the Channel interface
func (c *FeishuChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	return nil, errUnsupported
}

// EditMessage is a stub method to satisfy MessageEditor
func (c *FeishuChannel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	return errUnsupported
}

// SendPlaceholder is a stub method to satisfy PlaceholderCapable
func (c *FeishuChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	return "", errUnsupported
}

// ReactToMessage is a stub method to satisfy ReactionCapable
func (c *FeishuChannel) ReactToMessage(ctx context.Context, chatID, messageID string) (func(), error) {
	return func() {}, errUnsupported
}

// SendMedia is a stub method to satisfy MediaSender
func (c *FeishuChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) ([]string, error) {
	return nil, errUnsupported
}
