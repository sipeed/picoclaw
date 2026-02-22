package maixcam

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type MaixCamChannel struct {
	*channels.BaseChannel
	config     config.MaixCamConfig
	listener   net.Listener
	ctx        context.Context
	cancel     context.CancelFunc
	clients    map[net.Conn]bool
	clientsMux sync.RWMutex
}

type MaixCamMessage struct {
	Type      string         `json:"type"`
	Tips      string         `json:"tips"`
	Timestamp float64        `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

func NewMaixCamChannel(cfg config.MaixCamConfig, bus *bus.MessageBus) (*MaixCamChannel, error) {
	base := channels.NewBaseChannel("maixcam", cfg, bus, cfg.AllowFrom)

	return &MaixCamChannel{
		BaseChannel: base,
		config:      cfg,
		clients:     make(map[net.Conn]bool),
	}, nil
}

func (c *MaixCamChannel) Start(ctx context.Context) error {
	logger.InfoC("maixcam", "Starting MaixCam channel server")

	c.ctx, c.cancel = context.WithCancel(ctx)

	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		c.cancel()
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	c.listener = listener
	c.SetRunning(true)

	logger.InfoCF("maixcam", "MaixCam server listening", map[string]any{
		"host": c.config.Host,
		"port": c.config.Port,
	})

	go c.acceptConnections()

	return nil
}

func (c *MaixCamChannel) acceptConnections() {
	logger.DebugC("maixcam", "Starting connection acceptor")

	for {
		select {
		case <-c.ctx.Done():
			logger.InfoC("maixcam", "Stopping connection acceptor")
			return
		default:
			conn, err := c.listener.Accept()
			if err != nil {
				if c.IsRunning() {
					logger.ErrorCF("maixcam", "Failed to accept connection", map[string]any{
						"error": err.Error(),
					})
				}
				return
			}

			logger.InfoCF("maixcam", "New connection from MaixCam device", map[string]any{
				"remote_addr": conn.RemoteAddr().String(),
			})

			c.clientsMux.Lock()
			c.clients[conn] = true
			c.clientsMux.Unlock()

			go c.handleConnection(conn)
		}
	}
}

func (c *MaixCamChannel) handleConnection(conn net.Conn) {
	logger.DebugC("maixcam", "Handling MaixCam connection")

	defer func() {
		conn.Close()
		c.clientsMux.Lock()
		delete(c.clients, conn)
		c.clientsMux.Unlock()
		logger.DebugC("maixcam", "Connection closed")
	}()

	decoder := json.NewDecoder(conn)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			var msg MaixCamMessage
			if err := decoder.Decode(&msg); err != nil {
				if err.Error() != "EOF" {
					logger.ErrorCF("maixcam", "Failed to decode message", map[string]any{
						"error": err.Error(),
					})
				}
				return
			}

			c.processMessage(msg, conn)
		}
	}
}

func (c *MaixCamChannel) processMessage(msg MaixCamMessage, conn net.Conn) {
	switch msg.Type {
	case "person_detected":
		c.handlePersonDetection(msg)
	case "heartbeat":
		logger.DebugC("maixcam", "Received heartbeat")
	case "status":
		c.handleStatusUpdate(msg)
	default:
		logger.WarnCF("maixcam", "Unknown message type", map[string]any{
			"type": msg.Type,
		})
	}
}

func (c *MaixCamChannel) handlePersonDetection(msg MaixCamMessage) {
	logger.InfoCF("maixcam", "", map[string]any{
		"timestamp": msg.Timestamp,
		"data":      msg.Data,
	})

	senderID := "maixcam"
	chatID := "default"

	classInfo, ok := msg.Data["class_name"].(string)
	if !ok {
		classInfo = "person"
	}

	score, _ := msg.Data["score"].(float64)
	x, _ := msg.Data["x"].(float64)
	y, _ := msg.Data["y"].(float64)
	w, _ := msg.Data["w"].(float64)
	h, _ := msg.Data["h"].(float64)

	content := fmt.Sprintf("📷 Person detected!\nClass: %s\nConfidence: %.2f%%\nPosition: (%.0f, %.0f)\nSize: %.0fx%.0f",
		classInfo, score*100, x, y, w, h)

	metadata := map[string]string{
		"timestamp": fmt.Sprintf("%.0f", msg.Timestamp),
		"class_id":  fmt.Sprintf("%.0f", msg.Data["class_id"]),
		"score":     fmt.Sprintf("%.2f", score),
		"x":         fmt.Sprintf("%.0f", x),
		"y":         fmt.Sprintf("%.0f", y),
		"w":         fmt.Sprintf("%.0f", w),
		"h":         fmt.Sprintf("%.0f", h),
	}

	c.HandleMessage(bus.Peer{Kind: "channel", ID: "default"}, "", senderID, chatID, content, []string{}, metadata)
}

func (c *MaixCamChannel) handleStatusUpdate(msg MaixCamMessage) {
	logger.InfoCF("maixcam", "Status update from MaixCam", map[string]any{
		"status": msg.Data,
	})
}

func (c *MaixCamChannel) Stop(ctx context.Context) error {
	logger.InfoC("maixcam", "Stopping MaixCam channel")
	c.SetRunning(false)

	// Cancel context first to signal goroutines to exit
	if c.cancel != nil {
		c.cancel()
	}

	if c.listener != nil {
		c.listener.Close()
	}

	c.clientsMux.Lock()
	defer c.clientsMux.Unlock()

	for conn := range c.clients {
		conn.Close()
	}
	c.clients = make(map[net.Conn]bool)

	logger.InfoC("maixcam", "MaixCam channel stopped")
	return nil
}

func (c *MaixCamChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	// Check ctx before entering write path
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.clientsMux.RLock()
	defer c.clientsMux.RUnlock()

	if len(c.clients) == 0 {
		logger.WarnC("maixcam", "No MaixCam devices connected")
		return fmt.Errorf("no connected MaixCam devices")
	}

	response := map[string]any{
		"type":      "command",
		"timestamp": float64(0),
		"message":   msg.Content,
		"chat_id":   msg.ChatID,
	}

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	var sendErr error
	for conn := range c.clients {
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if _, err := conn.Write(data); err != nil {
			logger.ErrorCF("maixcam", "Failed to send to client", map[string]any{
				"client": conn.RemoteAddr().String(),
				"error":  err.Error(),
			})
			sendErr = fmt.Errorf("maixcam send: %w", channels.ErrTemporary)
		}
		_ = conn.SetWriteDeadline(time.Time{})
	}

	return sendErr
}
