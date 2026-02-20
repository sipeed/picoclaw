// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package jetstream

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// Store is a NATS JetStream-backed memory store
type Store struct {
	conn   *nats.Conn
	js     nats.JetStreamContext
	stream string
	bucket string // For KV store

	mu sync.RWMutex
}

// Config holds configuration for the JetStream store
type Config struct {
	// StreamName is the name of the JetStream stream
	StreamName string

	// BucketName is the name of the KV bucket (optional, for CRUD operations)
	BucketName string

	// Subjects is the list of subjects to include in the stream
	Subjects []string

	// MaxAge is the maximum age of messages in the stream
	MaxAge time.Duration

	// MaxBytes is the maximum size of the stream
	MaxBytes int64

	// Replicas is the number of stream replicas
	Replicas int
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		StreamName: "PICOCLAW_MEMORY",
		BucketName: "PICOCLAW_KV",
		Subjects:   []string{"picoclaw.memory.>"},
		MaxAge:     24 * time.Hour * 30, // 30 days
		MaxBytes:   1024 * 1024 * 1024, // 1GB
		Replicas:   1,
	}
}

// NewStore creates a new JetStream memory store
func NewStore(conn *nats.Conn) (*Store, error) {
	js, err := conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	return &Store{
		conn: conn,
		js:   js,
	}, nil
}

// Initialize sets up the JetStream stream and KV bucket
func (s *Store) Initialize(ctx context.Context, cfg *Config) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	s.mu.Lock()
	s.stream = cfg.StreamName
	s.bucket = cfg.BucketName
	s.mu.Unlock()

	// Create stream
	if err := s.createStream(ctx, cfg); err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	// Create KV bucket
	if cfg.BucketName != "" {
		if err := s.createBucket(ctx, cfg); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	logger.InfoCF("memory", "JetStream memory store initialized", map[string]interface{}{
		"stream": cfg.StreamName,
		"bucket": cfg.BucketName,
	})

	return nil
}

// createStream creates the JetStream stream
func (s *Store) createStream(ctx context.Context, cfg *Config) error {
	stream, err := s.js.StreamInfo(cfg.StreamName)
	if err != nil {
		// Stream doesn't exist, create it
		streamCfg := &nats.StreamConfig{
			Name:       cfg.StreamName,
			Subjects:   cfg.Subjects,
			MaxAge:     cfg.MaxAge,
			MaxBytes:   cfg.MaxBytes,
			Replicas:   cfg.Replicas,
			Retention:  nats.LimitsPolicy,
			Discard:    nats.DiscardOld,
			Storage:    nats.FileStorage,
		}

		_, err = s.js.AddStream(streamCfg)
		if err != nil {
			return fmt.Errorf("failed to add stream: %w", err)
		}

		logger.InfoCF("memory", "Created JetStream stream", map[string]interface{}{
			"name":     cfg.StreamName,
			"subjects": fmt.Sprintf("%v", cfg.Subjects),
		})
	} else {
		logger.DebugCF("memory", "JetStream stream already exists", map[string]interface{}{
			"name": stream.Config.Name,
		})
	}

	return nil
}

// createBucket creates the KV bucket
func (s *Store) createBucket(ctx context.Context, cfg *Config) error {
	_, err := s.js.KeyValue(cfg.BucketName)
	if err != nil {
		// Bucket doesn't exist, create it
		_, err = s.js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket:       cfg.BucketName,
			Description:  "PicoClaw Memory KV Store",
			MaxBytes:     cfg.MaxBytes / 10, // 10% of stream size
			TTL:          cfg.MaxAge,
			Storage:      nats.FileStorage,
			Replicas:     cfg.Replicas,
		})
		if err != nil {
			return fmt.Errorf("failed to create KV bucket: %w", err)
		}

		logger.InfoCF("memory", "Created KV bucket", map[string]interface{}{
			"name": cfg.BucketName,
		})
	} else {
		logger.DebugCF("memory", "KV bucket already exists", map[string]interface{}{
			"name": cfg.BucketName,
		})
	}

	return nil
}

// Publish publishes a memory item to the stream
func (s *Store) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := s.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}

	return nil
}

// Subscribe subscribes to memory updates
func (s *Store) Subscribe(ctx context.Context, subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
	// Create consumer subscription
	sub, err := s.js.PullSubscribe(subject, "", nats.AckExplicit(), nats.DeliverAll())
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	// Start consumer loop
	go func() {
		for {
			select {
			case <-ctx.Done():
				sub.Unsubscribe()
				return
			default:
				msgs, err := sub.Fetch(10, nats.MaxWait(2*time.Second))
				if err == nats.ErrTimeout {
					continue
				}
				if err != nil {
					logger.ErrorCF("memory", "Fetch error", map[string]interface{}{
						"error": err.Error(),
					})
					continue
				}

				for _, msg := range msgs {
					handler(msg)
					msg.Ack()
				}
			}
		}
	}()

	return sub, nil
}

// Get retrieves a value from KV store
func (s *Store) Get(key string) ([]byte, error) {
	s.mu.RLock()
	bucket := s.bucket
	s.mu.RUnlock()

	if bucket == "" {
		return nil, fmt.Errorf("no KV bucket configured")
	}

	kv, err := s.js.KeyValue(bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get KV bucket: %w", err)
	}

	entry, err := kv.Get(key)
	if err != nil {
		if err == nats.ErrKeyNotFound {
			return nil, ErrKeyNotFound{key}
		}
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	return entry.Value(), nil
}

// Put stores a value in KV store
func (s *Store) Put(key string, value []byte) error {
	s.mu.RLock()
	bucket := s.bucket
	s.mu.RUnlock()

	if bucket == "" {
		return fmt.Errorf("no KV bucket configured")
	}

	kv, err := s.js.KeyValue(bucket)
	if err != nil {
		return fmt.Errorf("failed to get KV bucket: %w", err)
	}

	_, err = kv.Put(key, value)
	if err != nil {
		return fmt.Errorf("failed to put key: %w", err)
	}

	return nil
}

// Delete removes a key from KV store
func (s *Store) Delete(key string) error {
	s.mu.RLock()
	bucket := s.bucket
	s.mu.RUnlock()

	if bucket == "" {
		return fmt.Errorf("no KV bucket configured")
	}

	kv, err := s.js.KeyValue(bucket)
	if err != nil {
		return fmt.Errorf("failed to get KV bucket: %w", err)
	}

	err = kv.Delete(key)
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	return nil
}

// Query performs a wildcard query on the KV store
func (s *Store) Query(prefix string) (map[string][]byte, error) {
	s.mu.RLock()
	bucket := s.bucket
	s.mu.RUnlock()

	if bucket == "" {
		return nil, fmt.Errorf("no KV bucket configured")
	}

	kv, err := s.js.KeyValue(bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get KV bucket: %w", err)
	}

	// Get all keys and values
	keys, err := kv.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	results := make(map[string][]byte)

	for _, key := range keys {
		if matchesPrefix(key, prefix) {
			entry, err := kv.Get(key)
			if err != nil {
				if err == nats.ErrKeyNotFound {
					continue
				}
				return nil, fmt.Errorf("failed to get key %s: %w", key, err)
			}
			results[key] = entry.Value()
		}
	}

	return results, nil
}

// Watch creates a watcher for key changes
func (s *Store) Watch(prefix string, callback func(key string, value []byte)) error {
	s.mu.RLock()
	bucket := s.bucket
	s.mu.RUnlock()

	if bucket == "" {
		return fmt.Errorf("no KV bucket configured")
	}

	kv, err := s.js.KeyValue(bucket)
	if err != nil {
		return fmt.Errorf("failed to get KV bucket: %w", err)
	}

	watcher, err := kv.WatchAll()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	go func() {
		for entry := range watcher.Updates() {
			if entry != nil {
				callback(entry.Key(), entry.Value())
			}
		}
	}()

	return nil
}

// GetStreamInfo returns information about the stream
func (s *Store) GetStreamInfo() (*nats.StreamInfo, error) {
	s.mu.RLock()
	stream := s.stream
	s.mu.RUnlock()

	return s.js.StreamInfo(stream)
}

// GetBucketStatus returns status of the KV bucket
func (s *Store) GetBucketStatus() (*nats.KeyValueStatus, error) {
	s.mu.RLock()
	bucket := s.bucket
	s.mu.RUnlock()

	if bucket == "" {
		return nil, fmt.Errorf("no KV bucket configured")
	}

	kv, err := s.js.KeyValue(bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get KV bucket: %w", err)
	}

	status, err := kv.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	return &status, nil
}

// Close closes the store connection
func (s *Store) Close() error {
	// NATS connection is managed externally, just clear references
	s.mu.Lock()
	s.stream = ""
	s.bucket = ""
	s.mu.Unlock()
	return nil
}

// ErrKeyNotFound is returned when a key is not found
type ErrKeyNotFound struct {
	Key string
}

func (e ErrKeyNotFound) Error() string {
	return fmt.Sprintf("key not found: %s", e.Key)
}

// matchesPrefix checks if a key matches a prefix
func matchesPrefix(key, prefix string) bool {
	if prefix == "" || prefix == "*" {
		return true
	}
	if prefix[len(prefix)-1] == '>' || prefix[len(prefix)-1] == '*' {
		// Wildcard prefix
		return len(key) >= len(prefix)-1 && key[:len(prefix)-1] == prefix[:len(prefix)-1]
	}
	return key == prefix
}

// JSONPut is a convenience method to store JSON values
func (s *Store) JSONPut(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return s.Put(key, data)
}

// JSONGet is a convenience method to retrieve JSON values
func (s *Store) JSONGet(key string, dest interface{}) error {
	data, err := s.Get(key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// PublishJSON publishes a JSON message to the stream
func (s *Store) PublishJSON(ctx context.Context, subject string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return s.Publish(ctx, subject, data)
}
