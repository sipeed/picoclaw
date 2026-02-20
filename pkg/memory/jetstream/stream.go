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

// StreamManager manages JetStream streams for memory storage
type StreamManager struct {
	store *Store
}

// NewStreamManager creates a new stream manager
func NewStreamManager(store *Store) *StreamManager {
	return &StreamManager{store: store}
}

// CreateMemoryStream creates a stream for a specific H-id
func (sm *StreamManager) CreateMemoryStream(ctx context.Context, hid string, cfg *StreamConfig) error {
	streamName := fmt.Sprintf("PICOCLAW_MEMORY_%s", hid)

	if cfg == nil {
		cfg = DefaultStreamConfig()
	}
	cfg.Name = streamName

	streamCfg := &nats.StreamConfig{
		Name:     cfg.Name,
		Subjects: cfg.Subjects,
		MaxAge:   cfg.MaxAge,
		MaxBytes: cfg.MaxBytes,
		Replicas: cfg.Replicas,
		Retention: nats.LimitsPolicy,
		Discard:   nats.DiscardOld,
		Storage:   nats.FileStorage,
	}

	if cfg.Compression {
		streamCfg.Compression = nats.S2Compression
	}

	js := sm.store.js
	_, err := js.AddStream(streamCfg)
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		return fmt.Errorf("failed to create memory stream: %w", err)
	}

	logger.InfoCF("memory", "Created memory stream", map[string]interface{}{
		"name":     streamName,
		"subjects": fmt.Sprintf("%v", cfg.Subjects),
		"max_age":  cfg.MaxAge.String(),
	})

	return nil
}

// DeleteMemoryStream deletes a stream for an H-id
func (sm *StreamManager) DeleteMemoryStream(ctx context.Context, hid string) error {
	streamName := fmt.Sprintf("PICOCLAW_MEMORY_%s", hid)

	js := sm.store.js
	err := js.DeleteStream(streamName)
	if err != nil {
		return fmt.Errorf("failed to delete stream: %w", err)
	}

	logger.InfoCF("memory", "Deleted memory stream", map[string]interface{}{
		"name": streamName,
	})

	return nil
}

// GetStreamInfo returns information about a memory stream
func (sm *StreamManager) GetStreamInfo(hid string) (*nats.StreamInfo, error) {
	streamName := fmt.Sprintf("PICOCLAW_MEMORY_%s", hid)

	js := sm.store.js
	return js.StreamInfo(streamName)
}

// ListStreams lists all memory streams
func (sm *StreamManager) ListStreams() ([]*nats.StreamInfo, error) {
	js := sm.store.js

	streams := make([]*nats.StreamInfo, 0)
	streamChan := js.Streams()

	for info := range streamChan {
		// Filter for memory streams
		if len(info.Config.Name) > 15 && info.Config.Name[:15] == "PICOCLAW_MEMORY_" {
			streams = append(streams, info)
		}
	}

	return streams, nil
}

// PurgeStream purges all messages from a stream
func (sm *StreamManager) PurgeStream(hid string) error {
	streamName := fmt.Sprintf("PICOCLAW_MEMORY_%s", hid)

	js := sm.store.js
	stream, err := js.StreamInfo(streamName)
	if err != nil {
		return fmt.Errorf("failed to get stream info: %w", err)
	}

	if err := js.PurgeStream(streamName); err != nil {
		return fmt.Errorf("failed to purge stream: %w", err)
	}

	logger.InfoCF("memory", "Purged memory stream", map[string]interface{}{
		"name":           streamName,
		"messages_before": stream.State.Msgs,
	})

	return nil
}

// StreamConfig holds configuration for a memory stream
type StreamConfig struct {
	Name        string
	Subjects    []string
	MaxAge      time.Duration
	MaxBytes    int64
	Replicas    int
	Compression bool
}

// DefaultStreamConfig returns default stream configuration
func DefaultStreamConfig() *StreamConfig {
	return &StreamConfig{
		Subjects:    []string{"picocraw.memory.>"},
		MaxAge:      30 * 24 * time.Hour, // 30 days
		MaxBytes:    256 * 1024 * 1024,  // 256MB
		Replicas:    1,
		Compression: false,
	}
}

// StreamStats holds statistics about a stream
type StreamStats struct {
	Name          string
	Messages      uint64
	Bytes         uint64
	FirstSequence uint64
	LastSequence  uint64
	CreateTime    time.Time
}

// GetStreamStats returns statistics for a stream
func (sm *StreamManager) GetStreamStats(hid string) (*StreamStats, error) {
	info, err := sm.GetStreamInfo(hid)
	if err != nil {
		return nil, err
	}

	return &StreamStats{
		Name:          info.Config.Name,
		Messages:      info.State.Msgs,
		Bytes:         info.State.Bytes,
		FirstSequence: info.State.FirstSeq,
		LastSequence:  info.State.LastSeq,
		CreateTime:    info.Created,
	}, nil
}

// GetAllStats returns statistics for all memory streams
func (sm *StreamManager) GetAllStats() (map[string]*StreamStats, error) {
	streams, err := sm.ListStreams()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]*StreamStats)

	for _, stream := range streams {
		// Extract H-id from stream name
		if len(stream.Config.Name) > 15 {
			hid := stream.Config.Name[15:]
			stats[hid] = &StreamStats{
				Name:          stream.Config.Name,
				Messages:      stream.State.Msgs,
				Bytes:         stream.State.Bytes,
				FirstSequence: stream.State.FirstSeq,
				LastSequence:  stream.State.LastSeq,
				CreateTime:    stream.Created,
			}
		}
	}

	return stats, nil
}

// CompactStream compacts a stream by removing deleted messages
func (sm *StreamManager) CompactStream(hid string) error {
	streamName := fmt.Sprintf("PICOCLAW_MEMORY_%s", hid)

	js := sm.store.js

	// Get current state
	info, err := js.StreamInfo(streamName)
	if err != nil {
		return fmt.Errorf("failed to get stream info: %w", err)
	}

	messagesBefore := info.State.Msgs

	// Compact the stream
	if err := js.DeleteStream(streamName); err != nil {
		return fmt.Errorf("failed to delete stream for compaction: %w", err)
	}

	// Recreate with same config
	streamCfg := info.Config
	_, err = js.AddStream(&streamCfg)
	if err != nil {
		return fmt.Errorf("failed to recreate stream: %w", err)
	}

	logger.InfoCF("memory", "Compacted memory stream", map[string]interface{}{
		"name":            streamName,
		"messages_before": messagesBefore,
	})

	return nil
}

// ScaleStream adjusts the replication factor of a stream
func (sm *StreamManager) ScaleStream(hid string, replicas int) error {
	streamName := fmt.Sprintf("PICOCLAW_MEMORY_%s", hid)

	js := sm.store.js

	info, err := js.StreamInfo(streamName)
	if err != nil {
		return fmt.Errorf("failed to get stream info: %w", err)
	}

	// Update config
	cfg := info.Config
	cfg.Replicas = replicas

	_, err = js.UpdateStream(&cfg)
	if err != nil {
		return fmt.Errorf("failed to update stream: %w", err)
	}

	logger.InfoCF("memory", "Scaled memory stream", map[string]interface{}{
		"name":     streamName,
		"replicas": replicas,
	})

	return nil
}

// CreateConsumer creates a consumer for a stream
func (sm *StreamManager) CreateConsumer(hid string, consumerName string, cfg *ConsumerConfig) error {
	streamName := fmt.Sprintf("PICOCLAW_MEMORY_%s", hid)

	if cfg == nil {
		cfg = DefaultConsumerConfig()
	}

	js := sm.store.js

	consumerCfg := &nats.ConsumerConfig{
		Durable:       consumerName,
		Description:   consumerName,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       cfg.AckWait,
		MaxDeliver:    cfg.MaxDeliveries,
		FilterSubject: cfg.FilterSubject,
		ReplayPolicy:  nats.ReplayInstantPolicy,
	}

	_, err := js.AddConsumer(streamName, consumerCfg)
	if err != nil && err != nats.ErrConsumerNameAlreadyInUse {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	logger.InfoCF("memory", "Created consumer", map[string]interface{}{
		"stream":   streamName,
		"consumer": consumerName,
		"filter":   cfg.FilterSubject,
	})

	return nil
}

// DeleteConsumer deletes a consumer
func (sm *StreamManager) DeleteConsumer(hid, consumerName string) error {
	streamName := fmt.Sprintf("PICOCLAW_MEMORY_%s", hid)

	js := sm.store.js
	err := js.DeleteConsumer(streamName, consumerName)
	if err != nil {
		return fmt.Errorf("failed to delete consumer: %w", err)
	}

	logger.InfoCF("memory", "Deleted consumer", map[string]interface{}{
		"stream":   streamName,
		"consumer": consumerName,
	})

	return nil
}

// ListConsumers lists all consumers for a stream
func (sm *StreamManager) ListConsumers(hid string) ([]*nats.ConsumerInfo, error) {
	streamName := fmt.Sprintf("PICOCLAW_MEMORY_%s", hid)

	js := sm.store.js

	// Get consumer info channel from the stream
	consumerChan := js.Consumers(streamName)

	consumers := make([]*nats.ConsumerInfo, 0)
	for cInfo := range consumerChan {
		consumers = append(consumers, cInfo)
	}

	return consumers, nil
}

// ConsumerConfig holds configuration for a consumer
type ConsumerConfig struct {
	FilterSubject string
	AckWait       time.Duration
	MaxDeliveries int
}

// DefaultConsumerConfig returns default consumer configuration
func DefaultConsumerConfig() *ConsumerConfig {
	return &ConsumerConfig{
		FilterSubject: ">",
		AckWait:       30 * time.Second,
		MaxDeliveries: 3,
	}
}
