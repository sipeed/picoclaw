//go:build !amd64 && !arm64 && !riscv64 && !mips64 && !ppc64

package channels

import (
	"context"
	"errors"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// FeishuChannel is a stub implementation for 32-bit architectures
type FeishuChannel struct {
	*BaseChannel
}

// NewFeishuChannel returns an error on 32-bit architectures where the Feishu SDK is not supported
func NewFeishuChannel(cfg config.FeishuConfig, bus *bus.MessageBus) (*FeishuChannel, error) {
	return nil, errors.New("feishu channel is not supported on 32-bit architectures (armv7l, 386, etc.). Please use a 64-bit system or disable feishu in your config")
}

// Start is a stub method to satisfy the Channel interface
func (c *FeishuChannel) Start(ctx context.Context) error {
	return nil
}

// Stop is a stub method to satisfy the Channel interface
func (c *FeishuChannel) Stop(ctx context.Context) error {
	return nil
}

// Send is a stub method to satisfy the Channel interface
func (c *FeishuChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	return errors.New("feishu channel is not supported on 32-bit architectures")
}
