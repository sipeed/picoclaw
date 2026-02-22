// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package jetstream

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// Consumer manages a JetStream consumer for memory operations
type Consumer struct {
	js         nats.JetStreamContext
	streamName string
	name       string
	sub        *nats.Subscription
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewConsumer creates a new consumer
func NewConsumer(js nats.JetStreamContext, streamName, consumerName string) *Consumer {
	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		js:         js,
		streamName: streamName,
		name:       consumerName,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Subscribe creates a subscription for this consumer
func (c *Consumer) Subscribe(subject string, handler func(*nats.Msg), opts ...nats.SubOpt) error {
	// Create consumer config if it doesn't exist
	consumerCfg := &nats.ConsumerConfig{
		Durable:       c.name,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    3,
		FilterSubject: subject,
		DeliverPolicy: nats.DeliverAllPolicy,
	}

	// Try to get existing consumer info
	_, err := c.js.ConsumerInfo(c.streamName, c.name)
	if err != nil {
		// Consumer doesn't exist, create it
		_, err = c.js.AddConsumer(c.streamName, consumerCfg)
		if err != nil && err != nats.ErrConsumerNameAlreadyInUse {
			return fmt.Errorf("failed to create consumer: %w", err)
		}
	}

	// Create pull subscription
	sub, err := c.js.PullSubscribe(subject, c.name, opts...)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	c.sub = sub

	// Start message processing loop
	go c.processMessages(handler)

	logger.InfoCF("memory", "Consumer subscribed", map[string]interface{}{
		"stream":   c.streamName,
		"consumer": c.name,
		"subject":  subject,
	})

	return nil
}

// processMessages processes messages from the subscription
func (c *Consumer) processMessages(handler func(*nats.Msg)) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if c.sub == nil {
				continue
			}

			msgs, err := c.sub.Fetch(10, nats.MaxWait(500*time.Millisecond))
			if err == nats.ErrTimeout {
				continue
			}
			if err != nil {
				logger.ErrorCF("memory", "Consumer fetch error", map[string]interface{}{
					"error":    err.Error(),
					"consumer": c.name,
				})
				continue
			}

			for _, msg := range msgs {
				handler(msg)
			}
		}
	}
}

// Ack acknowledges a message
func (c *Consumer) Ack(msg *nats.Msg) error {
	return msg.Ack()
}

// Nak negatively acknowledges a message
func (c *Consumer) Nak(msg *nats.Msg) error {
	return msg.Nak()
}

// AckSync acknowledges a message synchronously
func (c *Consumer) AckSync(msg *nats.Msg) error {
	return msg.AckSync()
}

// Term terminates a message with no retry
func (c *Consumer) Term(msg *nats.Msg) error {
	return msg.Term()
}

// InProgress tells the server that work is ongoing
func (c *Consumer) InProgress(msg *nats.Msg) error {
	return msg.InProgress()
}

// Close closes the consumer
func (c *Consumer) Close() error {
	c.cancel()

	if c.sub != nil {
		if err := c.sub.Unsubscribe(); err != nil {
			return fmt.Errorf("failed to unsubscribe: %w", err)
		}
	}

	logger.InfoCF("memory", "Consumer closed", map[string]interface{}{
		"consumer": c.name,
	})

	return nil
}

// GetInfo returns consumer information
func (c *Consumer) GetInfo() (*nats.ConsumerInfo, error) {
	return c.js.ConsumerInfo(c.streamName, c.name)
}

// Pause pauses message delivery
func (c *Consumer) Pause() error {
	if c.sub == nil {
		return fmt.Errorf("no active subscription")
	}

	return c.sub.SetPendingLimits(-1, -1)
}

// Resume resumes message delivery
func (c *Consumer) Resume() error {
	if c.sub == nil {
		return fmt.Errorf("no active subscription")
	}

	return c.sub.SetPendingLimits(1000, 100*1024*1024)
}

// ConsumerSet manages multiple consumers
type ConsumerSet struct {
	js         nats.JetStreamContext
	streamName string
	consumers  map[string]*Consumer
}

// NewConsumerSet creates a new consumer set
func NewConsumerSet(js nats.JetStreamContext, streamName string) *ConsumerSet {
	return &ConsumerSet{
		js:         js,
		streamName: streamName,
		consumers:  make(map[string]*Consumer),
	}
}

// Add adds a consumer to the set
func (cs *ConsumerSet) Add(name string, subject string, handler func(*nats.Msg)) error {
	consumer := NewConsumer(cs.js, cs.streamName, name)

	if err := consumer.Subscribe(subject, handler); err != nil {
		return err
	}

	cs.consumers[name] = consumer
	return nil
}

// Get gets a consumer by name
func (cs *ConsumerSet) Get(name string) *Consumer {
	return cs.consumers[name]
}

// Remove removes a consumer from the set
func (cs *ConsumerSet) Remove(name string) error {
	consumer, ok := cs.consumers[name]
	if !ok {
		return fmt.Errorf("consumer not found: %s", name)
	}

	if err := consumer.Close(); err != nil {
		return err
	}

	delete(cs.consumers, name)
	return nil
}

// Close closes all consumers
func (cs *ConsumerSet) Close() error {
	for name, consumer := range cs.consumers {
		if err := consumer.Close(); err != nil {
			logger.ErrorCF("memory", "Error closing consumer", map[string]interface{}{
				"consumer": name,
				"error":    err.Error(),
			})
		}
	}

	cs.consumers = make(map[string]*Consumer)
	return nil
}

// List returns all consumer names
func (cs *ConsumerSet) List() []string {
	names := make([]string, 0, len(cs.consumers))
	for name := range cs.consumers {
		names = append(names, name)
	}
	return names
}

// MessageHandler is a function that handles memory messages
type MessageHandler func(msg *MemoryMessage) error

// MemoryMessage wraps a NATS message with memory metadata
type MemoryMessage struct {
	*nats.Msg
	ID        string
	OwnerHID  string
	OwnerSID  string
	Type      string
	Timestamp int64
}

// MemoryConsumer is a specialized consumer for memory operations
type MemoryConsumer struct {
	*Consumer
	handler MessageHandler
}

// NewMemoryConsumer creates a new memory consumer
func NewMemoryConsumer(js nats.JetStreamContext, streamName, consumerName string, handler MessageHandler) *MemoryConsumer {
	base := NewConsumer(js, streamName, consumerName)

	return &MemoryConsumer{
		Consumer: base,
		handler:  handler,
	}
}

// Subscribe subscribes to memory messages
func (mc *MemoryConsumer) Subscribe(filterSubject string) error {
	return mc.Consumer.Subscribe(filterSubject, func(msg *nats.Msg) {
		// Parse memory metadata
		memMsg := &MemoryMessage{
			Msg: msg,
		}

		// Extract metadata from headers
		if msg.Header != nil {
			memMsg.ID = msg.Header.Get("X-Memory-ID")
			memMsg.OwnerHID = msg.Header.Get("X-Owner-HID")
			memMsg.OwnerSID = msg.Header.Get("X-Owner-SID")
			memMsg.Type = msg.Header.Get("X-Memory-Type")

			// Parse timestamp
			if ts := msg.Header.Get("X-Timestamp"); ts != "" {
				fmt.Sscanf(ts, "%d", &memMsg.Timestamp)
			}
		}

		// Call handler
		if err := mc.handler(memMsg); err != nil {
			logger.ErrorCF("memory", "Handler error", map[string]interface{}{
				"error":    err.Error(),
				"msg_id":   memMsg.ID,
				"consumer": mc.name,
			})
			msg.Nak()
		} else {
			msg.Ack()
		}
	})
}

// BatchConsumer handles messages in batches
type BatchConsumer struct {
	*Consumer
	handler     func([]*MemoryMessage) error
	batchSize   int
	batchTimeout time.Duration
}

// NewBatchConsumer creates a new batch consumer
func NewBatchConsumer(js nats.JetStreamContext, streamName, consumerName string, handler func([]*MemoryMessage) error, batchSize int, batchTimeout time.Duration) *BatchConsumer {
	base := NewConsumer(js, streamName, consumerName)

	return &BatchConsumer{
		Consumer:     base,
		handler:      handler,
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
	}
}

// Subscribe subscribes to messages in batch mode
func (bc *BatchConsumer) Subscribe(filterSubject string) error {
	batch := make([]*MemoryMessage, 0, bc.batchSize)
	ticker := time.NewTicker(bc.batchTimeout)
	defer ticker.Stop()

	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}

		messages := make([]*MemoryMessage, len(batch))
		copy(messages, batch)

		err := bc.handler(messages)

		// Ack all messages
		for _, msg := range batch {
			if err != nil {
				msg.Msg.Nak()
			} else {
				msg.Msg.Ack()
			}
		}

		batch = batch[:0]
		return err
	}

	return bc.Consumer.Subscribe(filterSubject, func(msg *nats.Msg) {
		memMsg := &MemoryMessage{
			Msg: msg,
			ID:  msg.Header.Get("X-Memory-ID"),
		}

		batch = append(batch, memMsg)

		if len(batch) >= bc.batchSize {
			if err := flushBatch(); err != nil {
				logger.ErrorCF("memory", "Batch handler error", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	})
}

// GetConsumerInfo retrieves detailed information about a consumer
func GetConsumerInfo(js nats.JetStreamContext, streamName, consumerName string) (*nats.ConsumerInfo, error) {
	return js.ConsumerInfo(streamName, consumerName)
}

// ResetConsumer resets a consumer to start from the beginning
func ResetConsumer(js nats.JetStreamContext, streamName, consumerName string) error {
	info, err := js.ConsumerInfo(streamName, consumerName)
	if err != nil {
		return err
	}

	cfg := info.Config
	cfg.DeliverPolicy = nats.DeliverAllPolicy

	_, err = js.UpdateConsumer(streamName, &cfg)
	return err
}

// SetAckWait sets the acknowledgment timeout for a consumer
func SetAckWait(js nats.JetStreamContext, streamName, consumerName string, ackWait time.Duration) error {
	info, err := js.ConsumerInfo(streamName, consumerName)
	if err != nil {
		return err
	}

	cfg := info.Config
	cfg.AckWait = ackWait

	_, err = js.UpdateConsumer(streamName, &cfg)
	return err
}

// SetMaxDeliveries sets the maximum delivery count for a consumer
func SetMaxDeliveries(js nats.JetStreamContext, streamName, consumerName string, maxDeliveries int) error {
	info, err := js.ConsumerInfo(streamName, consumerName)
	if err != nil {
		return err
	}

	cfg := info.Config
	cfg.MaxDeliver = maxDeliveries

	_, err = js.UpdateConsumer(streamName, &cfg)
	return err
}
