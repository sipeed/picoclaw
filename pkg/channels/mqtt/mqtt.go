package mqtt

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// mqttPayload is the JSON payload for both inbound and outbound messages.
type mqttPayload struct {
	Text string `json:"text"`
}

// MQTTChannel implements the Channel interface for MQTT-based communication.
type MQTTChannel struct {
	*channels.BaseChannel
	bc       *config.Channel
	cfg      *config.MQTTSettings
	client   pahomqtt.Client
	qos      byte
	clientID string
}

// NewMQTTChannel creates a new MQTT channel instance.
func NewMQTTChannel(bc *config.Channel, cfg *config.MQTTSettings, b *bus.MessageBus) (*MQTTChannel, error) {
	if cfg.Broker == "" {
		return nil, fmt.Errorf("mqtt broker is required")
	}
	if cfg.AgentID == "" {
		return nil, fmt.Errorf("mqtt agent_id is required")
	}

	base := channels.NewBaseChannel("mqtt", cfg, b, bc.AllowFrom,
		channels.WithGroupTrigger(bc.GroupTrigger),
		channels.WithReasoningChannelID(bc.ReasoningChannelID),
	)

	mqttClientID := cfg.ClientID
	if mqttClientID == "" {
		var suffix [4]byte
		_, _ = rand.Read(suffix[:])
		mqttClientID = fmt.Sprintf("picoclaw-mqtt-%s-%s", cfg.AgentID, hex.EncodeToString(suffix[:]))
	}

	return &MQTTChannel{
		BaseChannel: base,
		bc:          bc,
		cfg:         cfg,
		qos:         byte(cfg.QoS),
		clientID:    mqttClientID,
	}, nil
}

// Start connects to the MQTT broker and begins listening for inbound messages.
func (c *MQTTChannel) Start(ctx context.Context) error {
	logger.InfoC("mqtt", "Starting MQTT channel")

	keepAlive := c.cfg.KeepAlive
	if keepAlive <= 0 {
		keepAlive = 60
	}

	opts := pahomqtt.NewClientOptions()
	opts.AddBroker(c.cfg.Broker)
	opts.SetClientID(c.clientID)
	opts.SetKeepAlive(time.Duration(keepAlive) * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec

	if c.cfg.Username.String() != "" {
		opts.SetUsername(c.cfg.Username.String())
		opts.SetPassword(c.cfg.Password.String())
	}

	firstSubscribe := make(chan error, 1)
	var once sync.Once

	opts.SetOnConnectHandler(func(client pahomqtt.Client) {
		logger.InfoC("mqtt", "MQTT connected, subscribing to inbound topic")
		err := c.subscribe(client)
		once.Do(func() { firstSubscribe <- err })
	})

	opts.SetConnectionLostHandler(func(_ pahomqtt.Client, err error) {
		logger.WarnCF("mqtt", "MQTT connection lost", map[string]any{"error": err.Error()})
	})

	client := pahomqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		client.Disconnect(250)
		return fmt.Errorf("mqtt connect timed out after 10s (broker: %s)", c.cfg.Broker)
	}
	if err := token.Error(); err != nil {
		client.Disconnect(250)
		return fmt.Errorf("mqtt connect failed: %w", err)
	}

	if err := <-firstSubscribe; err != nil {
		client.Disconnect(250)
		return fmt.Errorf("mqtt subscribe failed: %w", err)
	}

	c.client = client
	c.SetRunning(true)

	logger.InfoCF("mqtt", "MQTT channel started", map[string]any{
		"broker":   c.cfg.Broker,
		"agent_id": c.cfg.AgentID,
	})
	return nil
}

// topicPrefix returns the configured topic prefix, normalizing slashes.
// Trailing slashes are stripped; the result may or may not have a leading slash
// depending on what the user configured.
func (c *MQTTChannel) topicPrefix() string {
	p := strings.TrimRight(c.cfg.TopicPrefix, "/")
	if p == "" {
		return "/picoclaw"
	}
	return p
}

// clientIDFromTopic extracts the client_id segment from a received topic.
// Topic structure: {prefix}/{agent_id}/{client_id}/request
func (c *MQTTChannel) clientIDFromTopic(topic string) (string, bool) {
	prefix := c.topicPrefix()
	// Build the expected fixed portion: {prefix}/{agent_id}/
	fixed := prefix + "/" + c.cfg.AgentID + "/"
	after, ok := strings.CutPrefix(topic, fixed)
	if !ok {
		return "", false
	}
	// after = "{client_id}/request"
	slash := strings.IndexByte(after, '/')
	if slash < 0 {
		return "", false
	}
	return after[:slash], true
}

// subscribe subscribes to the inbound topic for this agent.
func (c *MQTTChannel) subscribe(client pahomqtt.Client) error {
	topic := fmt.Sprintf("%s/%s/+/request", c.topicPrefix(), c.cfg.AgentID)
	token := client.Subscribe(topic, c.qos, func(_ pahomqtt.Client, msg pahomqtt.Message) {
		c.handleInbound(msg)
	})
	token.Wait()
	if err := token.Error(); err != nil {
		logger.ErrorCF("mqtt", "Failed to subscribe", map[string]any{
			"topic": topic,
			"error": err.Error(),
		})
		return err
	}
	logger.InfoCF("mqtt", "Subscribed to inbound topic", map[string]any{"topic": topic})
	return nil
}

// handleInbound processes an inbound MQTT message.
func (c *MQTTChannel) handleInbound(msg pahomqtt.Message) {
	topic := msg.Topic()

	clientID, ok := c.clientIDFromTopic(topic)
	if !ok {
		logger.WarnCF("mqtt", "Unexpected topic format", map[string]any{"topic": topic})
		return
	}
	chatID := "mqtt:" + clientID

	var payload mqttPayload
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		logger.WarnCF("mqtt", "Failed to parse inbound payload", map[string]any{
			"topic": topic,
			"error": err.Error(),
		})
		return
	}

	if payload.Text == "" {
		logger.WarnCF("mqtt", "Inbound payload missing text", map[string]any{"topic": topic})
		return
	}

	inboundCtx := bus.InboundContext{
		Channel:  "mqtt",
		ChatID:   chatID,
		ChatType: "direct",
		SenderID: clientID,
	}

	c.HandleInboundContext(context.Background(), chatID, payload.Text, nil, inboundCtx)
}

// Stop disconnects from the MQTT broker.
func (c *MQTTChannel) Stop(_ context.Context) error {
	logger.InfoC("mqtt", "Stopping MQTT channel")
	c.SetRunning(false)

	if c.client != nil {
		c.client.Disconnect(500)
	}

	logger.InfoC("mqtt", "MQTT channel stopped")
	return nil
}

// Send publishes a response to the client via MQTT.
func (c *MQTTChannel) Send(_ context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	if strings.TrimSpace(msg.Content) == "" {
		return nil, nil
	}

	clientID := strings.TrimPrefix(msg.ChatID, "mqtt:")
	if clientID == msg.ChatID {
		logger.WarnCF("mqtt", "Send called with unexpected chatID format", map[string]any{"chat_id": msg.ChatID})
		return nil, nil
	}

	topic := fmt.Sprintf("%s/%s/%s/response", c.topicPrefix(), c.cfg.AgentID, clientID)

	data, err := json.Marshal(mqttPayload{Text: msg.Content})
	if err != nil {
		return nil, fmt.Errorf("mqtt: failed to marshal outbound payload: %w", err)
	}

	token := c.client.Publish(topic, c.qos, false, data)
	token.Wait()
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("mqtt: publish failed: %w", err)
	}

	logger.DebugCF("mqtt", "Published response", map[string]any{"topic": topic})
	return nil, nil
}
