// Package cache provides caching for LLM responses
package cache

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryCache(t *testing.T) {
	cache := NewInMemoryCache()

	ctx := context.Background()
	key := "test-key"
	expectedValue := &LLMResponse{
		Content:      "test response",
		FinishReason: "stop",
	}

	// Test Set
	err := cache.Set(ctx, key, expectedValue, 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Test Get
	value, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	if value.Content != expectedValue.Content {
		t.Errorf("Expected content %s, got %s", expectedValue.Content, value.Content)
	}

	// Test Exists
	exists, err := cache.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}

	if !exists {
		t.Error("Expected key to exist in cache")
	}

	// Test Delete
	err = cache.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to delete cache: %v", err)
	}

	// Verify deletion
	_, err = cache.Get(ctx, key)
	if err == nil {
		t.Error("Expected error when getting deleted key")
	}

	// Verify non-existence after deletion
	exists, err = cache.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Failed to check existence after deletion: %v", err)
	}

	if exists {
		t.Error("Expected key to not exist after deletion")
	}
}

func TestCacheProviderCreation(t *testing.T) {
	// Test in-memory cache
	config := make(map[string]string)
	provider, err := NewCacheProvider("memory", config, true, 10*time.Minute)
	if err != nil {
		t.Fatalf("Failed to create in-memory cache provider: %v", err)
	}

	if provider == nil {
		t.Fatal("Expected non-nil in-memory cache provider")
	}

	defer provider.Close()

	// Test Redis cache
	// Using localhost and default settings for testing purposes
	config = map[string]string{
		"address":  "localhost:6379",
		"database": "0",
	}
	provider, err = NewCacheProvider("redis", config, true, 10*time.Minute)
	// Don't fail if Redis isn't available, just note it
	if err != nil && err.Error() != "dial tcp [::1]:6379: connect: connection refused" &&
		err.Error() != "dial tcp 127.0.0.1:6379: connect: connection refused" {
		t.Logf("Redis cache provider creation failed as expected (Redis maybe not running): %v", err)
	}

	// Test unsupported cache type
	provider, err = NewCacheProvider("unsupported", config, true, 10*time.Minute)
	if err == nil {
		t.Fatal("Expected error for unsupported cache type")
	}
}
