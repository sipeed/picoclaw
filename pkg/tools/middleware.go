package tools

import (
	"fmt"
	"sync"
	"time"
)

// ToolPolicy defines per-tool security constraints.
type ToolPolicy struct {
	MaxArgSize     int  // Max total size of all args in bytes (0 = unlimited)
	MaxCallsPerMin int  // Rate limit: calls per minute (0 = unlimited)
	Enabled        bool // Whether the tool is allowed to execute
}

// ToolMiddleware provides pre-execution security checks for tool calls.
type ToolMiddleware struct {
	policies map[string]ToolPolicy
	limiters map[string]*rateBucket
	mu       sync.RWMutex
	nowFunc  func() time.Time
}

// NewToolMiddleware creates a new middleware with default settings.
func NewToolMiddleware() *ToolMiddleware {
	return &ToolMiddleware{
		policies: make(map[string]ToolPolicy),
		limiters: make(map[string]*rateBucket),
		nowFunc:  time.Now,
	}
}

// SetPolicy configures the security policy for a specific tool.
func (m *ToolMiddleware) SetPolicy(toolName string, policy ToolPolicy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies[toolName] = policy
	if policy.MaxCallsPerMin > 0 {
		m.limiters[toolName] = newRateBucket(policy.MaxCallsPerMin, m.nowFunc)
	} else {
		delete(m.limiters, toolName)
	}
}

// Check validates a tool call against its policy.
// Returns nil if allowed, or an error describing why it was blocked.
func (m *ToolMiddleware) Check(toolName string, args map[string]any) error {
	m.mu.RLock()
	policy, hasPolicy := m.policies[toolName]
	limiter := m.limiters[toolName]
	m.mu.RUnlock()

	if !hasPolicy {
		return nil // no policy = allow
	}

	if !policy.Enabled {
		return fmt.Errorf("tool %q is disabled by policy", toolName)
	}

	// Input size validation
	if policy.MaxArgSize > 0 {
		totalSize := estimateArgSize(args)
		if totalSize > policy.MaxArgSize {
			return fmt.Errorf("tool %q input too large (%d bytes, max %d)", toolName, totalSize, policy.MaxArgSize)
		}
	}

	// Rate limiting
	if limiter != nil && !limiter.Allow() {
		return fmt.Errorf("tool %q rate limited (max %d/min)", toolName, policy.MaxCallsPerMin)
	}

	return nil
}

// estimateArgSize calculates the approximate size of tool arguments in bytes.
func estimateArgSize(args map[string]any) int {
	total := 0
	for _, v := range args {
		switch val := v.(type) {
		case string:
			total += len(val)
		case float64:
			total += 8
		case bool:
			total += 1
		default:
			total += 64 // estimate for complex types
		}
	}
	return total
}
