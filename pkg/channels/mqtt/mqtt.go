package mqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
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
	opts.SetMaxReconnectInterval(1 * time.Minute)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetWriteTimeout(10 * time.Second)
	opts.SetConnectTimeout(30 * time.Second)

	// TLS configuration
	if c.config.TLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
		}

		// Load CA certificate if provided
		if c.config.TLSCA != "" {
			caCert, err := os.ReadFile(c.config.TLSCA)
			if err != nil {
				return fmt.Errorf("failed to read CA certificate: %w", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return fmt.Errorf("failed to parse CA certificate")
			}
			tlsConfig.RootCAs = caCertPool
		}

		// Load client certificate and key if provided
		if c.config.TLSCert != "" && c.config.TLSKey != "" {
			cert, err := tls.LoadX509KeyPair(c.config.TLSCert, c.config.TLSKey)
			if err != nil {
				return fmt.Errorf("failed to load client certificate and key: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		opts.SetTLSConfig(tlsConfig)
	}

	// Set message handler
	opts.SetDefaultPublishHandler(c.onMessage)

	// Set connection lost handler
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		logger.ErrorCF("mqtt", "Connection lost", map[string]any{
			"error": err,
		})
		// Connection will be automatically reconnected by the client
	})

	// Set connect handler
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		logger.InfoC("mqtt", "Connected to MQTT broker")

		// Subscribe to topics after successful connection
		var subscriptionErrors []string
		for _, topic := range c.config.SubscribeTopics {
			if token := c.client.Subscribe(topic, byte(c.config.QoS), nil); token.Wait() && token.Error() != nil {
				errMsg := fmt.Sprintf("topic %s: %v", topic, token.Error())
				subscriptionErrors = append(subscriptionErrors, errMsg)
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

		// Log subscription errors summary
		if len(subscriptionErrors) > 0 {
			logger.ErrorCF("mqtt", "Subscription errors occurred", map[string]any{
				"errors": subscriptionErrors,
				"count":  len(subscriptionErrors),
			})
		}
	})

	c.client = mqtt.NewClient(opts)

	// Connect with retry
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		// Clean up client on connection failure
		c.client = nil
		return fmt.Errorf("mqtt connect failed: %w", token.Error())
	}

	logger.InfoCF("mqtt", "Connected to MQTT broker", map[string]any{
		"broker":    c.config.Broker,
		"client_id": c.config.ClientID,
	})

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

	var content string

	// Check if subscribe_json_key is configured
	if c.config.SubscribeJSONKey != nil && *c.config.SubscribeJSONKey != "" {
		// Parse as JSON and extract the specified key
		var jsonMsg map[string]interface{}
		if err := json.Unmarshal(msg.Payload(), &jsonMsg); err == nil {
			// Successfully parsed as JSON
			if value, exists := jsonMsg[*c.config.SubscribeJSONKey]; exists {
				content = fmt.Sprintf("%v", value)
				logger.InfoCF("mqtt", "Extracted JSON value", map[string]any{
					"key":     *c.config.SubscribeJSONKey,
					"content": content,
				})
			} else {
				logger.WarnCF("mqtt", "JSON key not found in message", map[string]any{
					"key": *c.config.SubscribeJSONKey,
				})
				content = string(msg.Payload()) // Fall back to raw payload
			}
		} else {
			// JSON parsing failed, treat as plain text
			logger.InfoCF("mqtt", "JSON parsing failed, treating as plain text", map[string]any{
				"error":   err.Error(),
				"payload": string(msg.Payload()),
			})
			content = string(msg.Payload())
		}
	} else {
		// No JSON key configured, treat as plain text
		content = string(msg.Payload())
		logger.InfoCF("mqtt", "Received plain text message", map[string]any{
			"payload": content,
		})
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
		// Improved parsing for reply-to: extract the last occurrence
		lastIndex := strings.LastIndex(content, "reply-to:")
		if lastIndex != -1 {
			replyTopicPart := content[lastIndex+len("reply-to:"):]
			// Find the end of the reply-to value (newline or end of string)
			if newlineIndex := strings.Index(replyTopicPart, "\n"); newlineIndex != -1 {
				replyTopic = strings.TrimSpace(replyTopicPart[:newlineIndex])
				content = strings.TrimSpace(content[:lastIndex]) + strings.TrimSpace(replyTopicPart[newlineIndex:])
			} else {
				replyTopic = strings.TrimSpace(replyTopicPart)
				content = strings.TrimSpace(content[:lastIndex])
			}
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
		"platform":    "mqtt",
		"topic":       msg.Topic(),
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
	replyTopic = strings.ReplaceAll(replyTopic, "{timestamp}", fmt.Sprintf("%d", time.Now().Unix()))
	replyTopic = strings.ReplaceAll(replyTopic, "{message_id}", msg.ReplyToMessageID)
	// Add more placeholders as needed

	// Validate reply topic
	if replyTopic == "" {
		return fmt.Errorf("reply topic is empty and no default configured")
	}

	var payload []byte
	var err error

	// Check if reply_json_key is configured
	if c.config.ReplyJSONKey != nil && *c.config.ReplyJSONKey != "" {
		// Send as JSON with the specified key
		jsonMsg := map[string]string{
			*c.config.ReplyJSONKey: msg.Content,
		}
		payload, err = json.Marshal(jsonMsg)
		if err != nil {
			return fmt.Errorf("failed to marshal MQTT JSON message: %w", err)
		}
	} else {
		// Send as plain text
		payload = []byte(msg.Content)
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
