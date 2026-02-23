package tools

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware_NoPolicy_Allows(t *testing.T) {
	mw := NewToolMiddleware()
	err := mw.Check("any_tool", map[string]any{"key": "value"})
	assert.NoError(t, err)
}

func TestMiddleware_Disabled_Blocks(t *testing.T) {
	mw := NewToolMiddleware()
	mw.SetPolicy("dangerous_tool", ToolPolicy{Enabled: false})

	err := mw.Check("dangerous_tool", map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled by policy")
}

func TestMiddleware_Enabled_Allows(t *testing.T) {
	mw := NewToolMiddleware()
	mw.SetPolicy("safe_tool", ToolPolicy{Enabled: true})

	err := mw.Check("safe_tool", map[string]any{"key": "value"})
	assert.NoError(t, err)
}

func TestMiddleware_ArgSizeLimit_Blocks(t *testing.T) {
	mw := NewToolMiddleware()
	mw.SetPolicy("exec", ToolPolicy{Enabled: true, MaxArgSize: 100})

	// Small args — should pass
	err := mw.Check("exec", map[string]any{"cmd": "ls"})
	assert.NoError(t, err)

	// Large args — should block
	largeCmd := strings.Repeat("a", 200)
	err = mw.Check("exec", map[string]any{"cmd": largeCmd})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "input too large")
}

func TestMiddleware_RateLimit_AllowsThenBlocks(t *testing.T) {
	now := time.Now()
	mw := NewToolMiddleware()
	mw.nowFunc = func() time.Time { return now }
	mw.SetPolicy("web_fetch", ToolPolicy{Enabled: true, MaxCallsPerMin: 3})

	args := map[string]any{"url": "https://example.com"}

	// First 3 calls should pass
	for i := 0; i < 3; i++ {
		err := mw.Check("web_fetch", args)
		assert.NoError(t, err, "call %d should be allowed", i+1)
	}

	// 4th call should be blocked
	err := mw.Check("web_fetch", args)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limited")
}

func TestMiddleware_RateLimit_ResetsAfterWindow(t *testing.T) {
	now := time.Now()
	currentTime := now
	mw := NewToolMiddleware()
	mw.nowFunc = func() time.Time { return currentTime }
	mw.SetPolicy("exec", ToolPolicy{Enabled: true, MaxCallsPerMin: 2})

	args := map[string]any{"cmd": "ls"}

	assert.NoError(t, mw.Check("exec", args))
	assert.NoError(t, mw.Check("exec", args))
	assert.Error(t, mw.Check("exec", args), "should be blocked at limit")

	// Advance past window
	currentTime = now.Add(61 * time.Second)
	assert.NoError(t, mw.Check("exec", args), "should be allowed after window reset")
}

func TestMiddleware_DifferentTools_IndependentPolicies(t *testing.T) {
	mw := NewToolMiddleware()
	mw.SetPolicy("exec", ToolPolicy{Enabled: true, MaxCallsPerMin: 1})
	mw.SetPolicy("web_fetch", ToolPolicy{Enabled: true, MaxCallsPerMin: 1})

	// Use up exec limit
	assert.NoError(t, mw.Check("exec", map[string]any{"cmd": "ls"}))
	assert.Error(t, mw.Check("exec", map[string]any{"cmd": "pwd"}))

	// web_fetch should still have its own limit
	assert.NoError(t, mw.Check("web_fetch", map[string]any{"url": "https://example.com"}))
	assert.Error(t, mw.Check("web_fetch", map[string]any{"url": "https://other.com"}))
}

func TestMiddleware_UnknownTool_Allowed(t *testing.T) {
	mw := NewToolMiddleware()
	mw.SetPolicy("exec", ToolPolicy{Enabled: false})

	// Tool without a policy should be allowed
	err := mw.Check("read_file", map[string]any{"path": "/tmp/file"})
	assert.NoError(t, err)
}

func TestMiddleware_ConcurrentAccess(t *testing.T) {
	now := time.Now()
	mw := NewToolMiddleware()
	mw.nowFunc = func() time.Time { return now }
	mw.SetPolicy("exec", ToolPolicy{Enabled: true, MaxCallsPerMin: 50})

	var wg sync.WaitGroup
	allowed := 0
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := mw.Check("exec", map[string]any{"cmd": "ls"})
			if err == nil {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, 50, allowed, "exactly 50 calls should be allowed")
}

func TestEstimateArgSize(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		expected int
	}{
		{
			name:     "empty args",
			args:     map[string]any{},
			expected: 0,
		},
		{
			name:     "string arg",
			args:     map[string]any{"cmd": "hello"},
			expected: 5,
		},
		{
			name:     "float64 arg",
			args:     map[string]any{"count": float64(42)},
			expected: 8,
		},
		{
			name:     "bool arg",
			args:     map[string]any{"verbose": true},
			expected: 1,
		},
		{
			name:     "mixed args",
			args:     map[string]any{"cmd": "ls -la", "count": float64(5), "verbose": true},
			expected: 6 + 8 + 1, // "ls -la" + float64 + bool
		},
		{
			name:     "unknown type",
			args:     map[string]any{"data": []int{1, 2, 3}},
			expected: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateArgSize(tt.args)
			assert.Equal(t, tt.expected, got)
		})
	}
}
