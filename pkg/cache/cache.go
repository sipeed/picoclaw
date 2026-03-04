// Package cache implements response caching for PicoClaw
package cache

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// Cache interface defines the common methods for caching
type Cache interface {
	Get(ctx context.Context, key string) (*LLMResponse, error)
	Set(ctx context.Context, key string, response *LLMResponse, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Close() error
}

// LLMResponse mirrors the response structure from providers package
type LLMResponse struct {
	Content          string            `json:"content"`
	ReasoningContent string            `json:"reasoning_content"`
	Reasoning        string            `json:"reasoning"`
	ReasoningDetails []ReasoningDetail `json:"reasoning_details"`
	ToolCalls        []ToolCall        `json:"tool_calls"`
	FinishReason     string            `json:"finish_reason"`
	Usage            *UsageInfo        `json:"usage,omitempty"`
	Intent           string            `json:"intent,omitempty"`
	ExtraContent     *ExtraContent     `json:"extra_content,omitempty"`
}

type ToolCall struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Arguments        map[string]interface{} `json:"arguments"`
	ThoughtSignature string                 `json:"thought_signature,omitempty"`
	ExtraContent     *ExtraContent          `json:"extra_content,omitempty"`
}

type ReasoningDetail struct {
	Text string `json:"text"`
}

type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ExtraContent struct {
	Google *GoogleExtra `json:"google,omitempty"`
}

type GoogleExtra struct {
	ThoughtSignature string `json:"thought_signature"`
}

// InMemoryCache implements cache interface using a thread-safe map
type InMemoryCache struct {
	data sync.Map
	ttls sync.Map
	mu   sync.RWMutex
}

// NewInMemoryCache creates a new in-memory cache instance
func NewInMemoryCache() *InMemoryCache {
	cache := &InMemoryCache{}
	// Start cleanup routine to remove expired entries
	go cache.cleanupExpired()
	return cache
}

// cleanupExpired periodically removes expired entries
func (c *InMemoryCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		c.ttls.Range(func(key, value interface{}) bool {
			exp, ok := value.(time.Time)
			if !ok || time.Now().After(exp) {
				c.data.Delete(key)
				c.ttls.Delete(key)
			}
			return true
		})
		c.mu.Unlock()
	}
}

// Get retrieves a cached LLM response for the given key
func (c *InMemoryCache) Get(ctx context.Context, key string) (*LLMResponse, error) {
	if value, ok := c.data.Load(key); ok {
		// Check if the entry has expired
		if expInterface, expOk := c.ttls.Load(key); expOk {
			if exp, ok := expInterface.(time.Time); ok && time.Now().After(exp) {
				c.data.Delete(key)
				c.ttls.Delete(key)
				return nil, fmt.Errorf("cache key %s expired", key)
			}
		}

		if response, ok := value.(*LLMResponse); ok {
			return response, nil
		}
		return nil, fmt.Errorf("cached value is not a valid LLMResponse")
	}
	return nil, fmt.Errorf("key %s not found in cache", key)
}

// Set caches an LLM response with the given key and TTL
func (c *InMemoryCache) Set(ctx context.Context, key string, response *LLMResponse, ttl time.Duration) error {
	expirationTime := time.Now().Add(ttl)
	c.data.Store(key, response)
	c.ttls.Store(key, expirationTime)
	return nil
}

// Delete removes a key from the cache
func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	c.data.Delete(key)
	c.ttls.Delete(key)
	return nil
}

// Exists checks if a key exists in the cache and hasn't expired
func (c *InMemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	if _, ok := c.data.Load(key); ok {
		// Check if the entry has expired
		if expInterface, expOk := c.ttls.Load(key); expOk {
			if exp, ok := expInterface.(time.Time); ok && time.Now().After(exp) {
				c.data.Delete(key)
				c.ttls.Delete(key)
				return false, nil
			}
		}
		return true, nil
	}
	return false, nil
}

// Close closes the cache (noop for in-memory cache)
func (c *InMemoryCache) Close() error {
	return nil
}

// RedisCache implements cache interface using Redis
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(address, password string, db int) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       db,
	})

	return &RedisCache{
		client: client,
	}
}

// Get retrieves a cached LLM response for the given key from Redis
func (rc *RedisCache) Get(ctx context.Context, key string) (*LLMResponse, error) {
	jsonData, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("key %s not found in cache", key)
		}
		return nil, fmt.Errorf("error getting from Redis: %w", err)
	}

	var response LLMResponse
	if err := json.Unmarshal([]byte(jsonData), &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling cached response: %w", err)
	}

	return &response, nil
}

// Set caches an LLM response with the given key and TTL in Redis
func (rc *RedisCache) Set(ctx context.Context, key string, response *LLMResponse, ttl time.Duration) error {
	jsonData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("error marshaling response: %w", err)
	}

	if err := rc.client.SetEX(ctx, key, jsonData, ttl).Err(); err != nil {
		return fmt.Errorf("error setting to Redis: %w", err)
	}

	return nil
}

// Delete removes a key from the Redis cache
func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	if err := rc.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("error deleting from Redis: %w", err)
	}
	return nil
}

// Exists checks if a key exists in the Redis cache
func (rc *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := rc.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("error checking if key exists in Redis: %w", err)
	}

	return exists > 0, nil
}

// Close closes the Redis client connection
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

// CacheProvider provides configurable caching implementation
type CacheProvider struct {
	cache   Cache
	enabled bool
	ttl     time.Duration
}

// NewCacheProvider creates a new cache provider with the specified implementation
func NewCacheProvider(cacheType string, config map[string]string, enabled bool, ttl time.Duration) (*CacheProvider, error) {
	var cache Cache

	switch strings.ToLower(cacheType) {
	case "memory":
		cache = NewInMemoryCache()
	case "redis":
		address := config["address"]
		password := config["password"]
		dbStr := config["database"]

		if address == "" {
			address = "localhost:6379"
		}

		db := 0
		if dbStr != "" {
			parsedDB, err := strconv.Atoi(dbStr)
			if err == nil {
				db = parsedDB
			}
		}

		cache = NewRedisCache(address, password, db)
	default:
		return nil, fmt.Errorf("unsupported cache type: %s", cacheType)
	}

	return &CacheProvider{
		cache:   cache,
		enabled: enabled,
		ttl:     ttl,
	}, nil
}

// Get retrieves an LLM response from the cache
func (cp *CacheProvider) Get(ctx context.Context, key string) (*LLMResponse, error) {
	if !cp.enabled {
		return nil, fmt.Errorf("cache disabled")
	}
	return cp.cache.Get(ctx, key)
}

// Set stores an LLM response in the cache
func (cp *CacheProvider) Set(ctx context.Context, key string, response *LLMResponse, customTTL *time.Duration) error {
	if !cp.enabled {
		return nil
	}

	ttl := cp.ttl
	if customTTL != nil {
		ttl = *customTTL
	}

	return cp.cache.Set(ctx, key, response, ttl)
}

// Delete removes a key from the cache
func (cp *CacheProvider) Delete(ctx context.Context, key string) error {
	if !cp.enabled {
		return nil
	}
	return cp.cache.Delete(ctx, key)
}

// Exists checks if a key exists in the cache
func (cp *CacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	if !cp.enabled {
		return false, nil
	}
	return cp.cache.Exists(ctx, key)
}

// GenerateKey generates a cache key for the given LLM request parameters
func (cp *CacheProvider) GenerateKey(messages []interface{}, model string, tools []interface{}, options map[string]interface{}) string {
	// Create a content hash of messages, model, tools and options
	content := fmt.Sprintf("%v:%s:%v:%v", messages, model, tools, options)
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("llm_response_%s", hex.EncodeToString(hash[:]))
}

// Close terminates the cache provider
func (cp *CacheProvider) Close() error {
	if cp.cache == nil {
		return nil
	}
	return cp.cache.Close()
}
