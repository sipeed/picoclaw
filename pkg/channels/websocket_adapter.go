// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package channels

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels/websocket"
	"github.com/sipeed/picoclaw/pkg/config"
)

// NewWebSocketChannel creates a new WebSocket channel instance.
// This is an adapter to maintain naming consistency with other channel constructors.
func NewWebSocketChannel(cfg config.WebSocketConfig, messageBus *bus.MessageBus) (Channel, error) {
	return websocket.NewChannel(cfg, messageBus)
}
