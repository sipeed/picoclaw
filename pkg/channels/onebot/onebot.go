package onebot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"

	"jane/pkg/bus"
	"jane/pkg/channels"
	"jane/pkg/config"
	"jane/pkg/logger"
)

func NewOneBotChannel(cfg config.OneBotConfig, messageBus *bus.MessageBus) (*OneBotChannel, error) {
	base := channels.NewBaseChannel("onebot", cfg, messageBus, cfg.AllowFrom,
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	const dedupSize = 1024
	return &OneBotChannel{
		BaseChannel: base,
		config:      cfg,
		dedup:       make(map[string]struct{}, dedupSize),
		dedupRing:   make([]string, dedupSize),
		dedupIdx:    0,
		pending:     make(map[string]chan json.RawMessage),
	}, nil
}

func (c *OneBotChannel) Start(ctx context.Context) error {
	if c.config.WSUrl == "" {
		return fmt.Errorf("OneBot ws_url not configured")
	}

	logger.InfoCF("onebot", "Starting OneBot channel", map[string]any{
		"ws_url": c.config.WSUrl,
	})

	c.ctx, c.cancel = context.WithCancel(ctx)

	if err := c.connect(); err != nil {
		logger.WarnCF("onebot", "Initial connection failed, will retry in background", map[string]any{
			"error": err.Error(),
		})
	} else {
		go c.listen()
		c.fetchSelfID()
	}

	if c.config.ReconnectInterval > 0 {
		go c.reconnectLoop()
	} else {
		if c.conn == nil {
			return fmt.Errorf("failed to connect to OneBot and reconnect is disabled")
		}
	}

	c.SetRunning(true)
	logger.InfoC("onebot", "OneBot channel started successfully")

	return nil
}

func (c *OneBotChannel) connect() error {
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	header := make(map[string][]string)
	if c.config.AccessToken != "" {
		header["Authorization"] = []string{"Bearer " + c.config.AccessToken}
	}

	conn, resp, err := dialer.Dial(c.config.WSUrl, header)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return err
	}

	conn.SetPongHandler(func(appData string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	go c.pinger(conn)

	logger.InfoC("onebot", "WebSocket connected")
	return nil
}

func (c *OneBotChannel) reconnectLoop() {
	interval := max(time.Duration(c.config.ReconnectInterval)*time.Second, 5*time.Second)

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(interval):
			c.mu.Lock()
			conn := c.conn
			c.mu.Unlock()

			if conn == nil {
				logger.InfoC("onebot", "Attempting to reconnect...")
				if err := c.connect(); err != nil {
					logger.ErrorCF("onebot", "Reconnect failed", map[string]any{
						"error": err.Error(),
					})
				} else {
					go c.listen()
					c.fetchSelfID()
				}
			}
		}
	}
}

func (c *OneBotChannel) Stop(ctx context.Context) error {
	logger.InfoC("onebot", "Stopping OneBot channel")
	c.SetRunning(false)

	if c.cancel != nil {
		c.cancel()
	}

	c.pendingMu.Lock()
	for echo, ch := range c.pending {
		select {
		case ch <- nil: // non-blocking wake for blocked sendAPIRequest goroutines
		default:
		}
		delete(c.pending, echo)
	}
	c.pendingMu.Unlock()

	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()

	return nil
}
