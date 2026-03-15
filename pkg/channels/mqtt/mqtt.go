package mqtt

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// MQTTChannel implements the Channel interface for MQTT brokers.
type MQTTChannel struct {
	*channels.BaseChannel
	config config.MQTTConfig
	client mqtt.Client
	ctx    context.Context
	cancel context.CancelFunc
}

// MQTTMessage represents the JSON structure for MQTT messages.
type MQTTMessage struct {
	Status string `json:"status"`
}

// NewMQTTChannel creates a new MQTT channel.
func NewMQTTChannel(cfg config.MQTTConfig, messageBus *bus.MessageBus) (*MQTTChannel, error) {
	if cfg.Broker == "" {
		return nil, fmt.Errorf("mqtt broker is required")
	}
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("mqtt client_id is required")
	}
	if len(cfg.SubscribeTopics) == 0 {
		return nil, fmt.Errorf("mqtt subscribe_topics is required")
	}

	base := channels.NewBaseChannel("mqtt", cfg, messageBus, cfg.AllowFrom,
		channels.WithMaxMessageLength(4000),
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &MQTTChannel{
		BaseChannel: base,
		config:      cfg,
	}, nil
}

// Name returns the channel name.
func (c *MQTTChannel) Name() string {
	return "mqtt"
}

// Start connects to the MQTT broker and begins listening.
func (c *MQTTChannel) Start(ctx context.Context) error {
	logger.InfoC("mqtt", "Starting MQTT channel")

	c.ctx, c.cancel = context.WithCancel(ctx)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(c.config.Broker)
	opts.SetClientID(c.config.ClientID)
	opts.SetUsername(c.config.Username)
	opts.SetPassword(c.config.Password)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetMaxReconnectInterval(5 * time.Minute)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetWriteTimeout(10 * time.Second)
	opts.SetConnectTimeout(30 * time.Second)

	// TLS configuration
	if c.config.TLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
		}
		if c.config.TLSCA != "" {
			// Load CA cert if provided
			// For simplicity, assuming file path
			// In production, load cert properly
		}
		if c.config.TLSCert != "" && c.config.TLSKey != "" {
			// Load client cert
		}
		opts.SetTLSConfig(tlsConfig)
	}

	// Set message handler
	opts.SetDefaultPublishHandler(c.onMessage)

	c.client = mqtt.NewClient(opts)

	// Connect with retry
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("mqtt connect failed: %w", token.Error())
	}

	logger.InfoCF("mqtt", "Connected to MQTT broker", map[string]any{
		"broker":   c.config.Broker,
		"client_id": c.config.ClientID,
	})

	// Subscribe to topics
	for _, topic := range c.config.SubscribeTopics {
		if token := c.client.Subscribe(topic, byte(c.config.QoS), nil); token.Wait() && token.Error() != nil {
		logger.ErrorCF("mqtt", "Failed to subscribe to topic", map[string]any{
			"topic": topic,
			"error": token.Error(),
		})
		} else {
			logger.InfoCF("mqtt", "Subscribed to topic", map[string]any{
				"topic": topic,
			})
		}
	}

	c.SetRunning(true)
	logger.InfoC("mqtt", "MQTT channel started")
	return nil
}

// Stop disconnects from the MQTT broker.
func (c *MQTTChannel) Stop(ctx context.Context) error {
	logger.InfoC("mqtt", "Stopping MQTT channel")
	c.SetRunning(false)

	if c.cancel != nil {
		c.cancel()
	}

	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
	}

	logger.InfoC("mqtt", "MQTT channel stopped")
	return nil
}

// onMessage handles incoming MQTT messages.
func (c *MQTTChannel) onMessage(client mqtt.Client, msg mqtt.Message) {
	logger.DebugCF("mqtt", "Received message", map[string]any{
		"topic":   msg.Topic(),
		"payload": string(msg.Payload()),
	})

	// Try to parse as JSON first
	var mqttMsg MQTTMessage
	var content string
	var err error
	
	// First try to parse as JSON
	if err = json.Unmarshal(msg.Payload(), &mqttMsg); err == nil {
		// Successfully parsed as JSON
		content = mqttMsg.Status
	} else {
		// If JSON parsing fails, try to clean up common malformed JSON issues
		payloadStr := string(msg.Payload())
		
		// Try to extract JSON from malformed strings (e.g., extra quotes or braces)
		// Look for a valid JSON object within the string
		if strings.HasPrefix(payloadStr, "{") && (strings.HasSuffix(payloadStr, "}") || strings.HasSuffix(payloadStr, "}\"")) {
			// Try to parse as-is first
			if err2 := json.Unmarshal(msg.Payload(), &mqttMsg); err2 == nil {
				content = mqttMsg.Status
			} else {
				// Try to clean up common issues
				cleaned := strings.TrimSpace(payloadStr)
				
				// Remove extra quotes and braces from the end
				for strings.HasSuffix(cleaned, "}") && strings.Count(cleaned, "{") < strings.Count(cleaned, "}") {
					cleaned = cleaned[:len(cleaned)-1]
				}
				for strings.HasSuffix(cleaned, "\"}") && strings.Count(cleaned, "{") < strings.Count(cleaned, "}") {
					cleaned = cleaned[:len(cleaned)-1]
				}
				// Remove any trailing quotes (simple check for extra quotes at the end)
				for strings.HasSuffix(cleaned, "\"") && !strings.HasSuffix(cleaned, "\"}") {
					cleaned = cleaned[:len(cleaned)-1]
				}
				
				// Remove extra opening braces
				for strings.HasPrefix(cleaned, "{") && strings.Count(cleaned, "{") > strings.Count(cleaned, "}") {
					cleaned = cleaned[1:]
				}
				
				if cleaned != payloadStr {
					if err3 := json.Unmarshal([]byte(cleaned), &mqttMsg); err3 == nil {
						content = mqttMsg.Status
						logger.InfoCF("mqtt", "Successfully parsed cleaned JSON", map[string]any{
							"original": payloadStr,
							"cleaned":  cleaned,
						})
					} else {
						// Fall back to plain text
						content = payloadStr
						logger.InfoCF("mqtt", "Received plain text message (JSON parsing failed)", map[string]any{
							"error":   err.Error(),
							"payload": content,
						})
					}
				} else {
					// Fall back to plain text
					content = payloadStr
					logger.InfoCF("mqtt", "Received plain text message (JSON parsing failed)", map[string]any{
						"error":   err.Error(),
						"payload": content,
					})
				}
			}
		} else {
			// Not JSON-like, treat as plain text
			content = payloadStr
			logger.InfoCF("mqtt", "Received plain text message (not JSON-like)", map[string]any{
				"payload": content,
			})
		}
	}

	if content == "" {
		logger.WarnC("mqtt", "Empty content in MQTT message")
		return
	}

	// Determine sender ID
	senderID := fmt.Sprintf("mqtt:%s", c.config.ClientID)
	if msg.Topic() != "" {
		senderID = fmt.Sprintf("mqtt:%s", strings.ReplaceAll(msg.Topic(), "/", "_"))
	}

	// Check if message has reply-to header (in payload or topic)
	replyTopic := c.config.ReplyTopic
	if strings.Contains(content, "reply-to:") {
		// Simple parsing for reply-to
		parts := strings.SplitN(content, "reply-to:", 2)
		if len(parts) == 2 {
			replyTopic = strings.TrimSpace(parts[1])
			content = strings.TrimSpace(parts[0])
		}
	}

	// Create peer and sender info
	peer := bus.Peer{Kind: "direct", ID: msg.Topic()} // MQTT topics are like channels
	sender := bus.SenderInfo{
		Platform:    "mqtt",
		PlatformID:  senderID,
		CanonicalID: senderID,
		Username:    senderID,
		DisplayName: senderID,
	}

	messageID := fmt.Sprintf("mqtt-%d", time.Now().UnixNano())

	metadata := map[string]string{
		"platform": "mqtt",
		"topic":    msg.Topic(),
		"reply_topic": replyTopic,
	}

	if c.config.Instruction != "" {
		content = c.config.Instruction + "\n\n" + content
	}

	c.HandleMessage(c.ctx, peer, messageID, senderID, msg.Topic(), content, nil, metadata, sender)
}

// Send sends a message to an MQTT topic.
func (c *MQTTChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("MQTT channel is not running")
	}

	replyTopic := c.config.ReplyTopic

	// If chatID contains reply topic info, use it
	if strings.HasPrefix(msg.ChatID, "reply:") {
		replyTopic = strings.TrimPrefix(msg.ChatID, "reply:")
	}

	// Replace placeholders in reply topic
	replyTopic = strings.ReplaceAll(replyTopic, "{client_id}", c.config.ClientID)
	replyTopic = strings.ReplaceAll(replyTopic, "{topic}", msg.ChatID)
	// Add more placeholders as needed

	mqttMsg := MQTTMessage{
		Status: msg.Content,
	}

	payload, err := json.Marshal(mqttMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal MQTT message: %w", err)
	}

	token := c.client.Publish(replyTopic, byte(c.config.QoS), c.config.Retain, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish MQTT message: %w", token.Error())
	}

	logger.DebugCF("mqtt", "Published message", map[string]any{
		"topic":   replyTopic,
		"payload": string(payload),
	})
	return nil
}