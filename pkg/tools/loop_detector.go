package tools

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// Session context key for per-session loop detection isolation.

type loopDetectorContextKey struct{}

// WithSessionKey returns a context carrying the given session key for loop detection.
func WithSessionKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, loopDetectorContextKey{}, key)
}

func sessionKeyFromContext(ctx context.Context) string {
	if key, ok := ctx.Value(loopDetectorContextKey{}).(string); ok {
		return key
	}
	return "_default"
}

// Default thresholds matching OpenClaw's production values.
const (
	DefaultHistorySize            = 30
	DefaultWarningThreshold       = 10
	DefaultCriticalThreshold      = 20
	DefaultCircuitBreakerThreshold = 30
)

// LoopDetectorConfig configures loop detection thresholds.
type LoopDetectorConfig struct {
	HistorySize             int  // sliding window size (default 30)
	WarningThreshold        int  // generic repeat warn level (default 10)
	CriticalThreshold       int  // block execution level (default 20)
	CircuitBreakerThreshold int  // global emergency stop (default 30)
	EnableGenericRepeat     bool // detect any repeated tool+args
	EnablePingPong          bool // detect alternating A,B,A,B patterns
}

// DefaultLoopDetectorConfig returns production-ready defaults.
func DefaultLoopDetectorConfig() LoopDetectorConfig {
	return LoopDetectorConfig{
		HistorySize:             DefaultHistorySize,
		WarningThreshold:        DefaultWarningThreshold,
		CriticalThreshold:       DefaultCriticalThreshold,
		CircuitBreakerThreshold: DefaultCircuitBreakerThreshold,
		EnableGenericRepeat:     true,
		EnablePingPong:          true,
	}
}

// LoopVerdict is the result of loop analysis.
type LoopVerdict struct {
	Blocked bool
	Warning bool
	Reason  string
	Count   int
}

// toolCallRecord represents a single tool call in the sliding window.
type toolCallRecord struct {
	ToolName   string
	ArgsHash   string
	ResultHash string // filled after execution via AfterExecute
	Timestamp  time.Time
}

// sessionState holds per-session detection state.
type sessionState struct {
	history []toolCallRecord
	mu      sync.Mutex
}

// LoopDetector implements ToolHook for detecting repetitive tool call patterns.
// It uses per-session state keyed by the session key in context.Context.
type LoopDetector struct {
	config   LoopDetectorConfig
	sessions sync.Map // sessionKey -> *sessionState
}

// NewLoopDetector creates a loop detector with the given configuration.
// Zero or negative thresholds are replaced with production defaults.
func NewLoopDetector(config LoopDetectorConfig) *LoopDetector {
	if config.HistorySize <= 0 {
		config.HistorySize = DefaultHistorySize
	}
	if config.WarningThreshold <= 0 {
		config.WarningThreshold = DefaultWarningThreshold
	}
	if config.CriticalThreshold <= 0 {
		config.CriticalThreshold = DefaultCriticalThreshold
	}
	if config.CircuitBreakerThreshold <= 0 {
		config.CircuitBreakerThreshold = DefaultCircuitBreakerThreshold
	}
	return &LoopDetector{config: config}
}

func (d *LoopDetector) getSession(key string) *sessionState {
	if v, ok := d.sessions.Load(key); ok {
		return v.(*sessionState)
	}
	s := &sessionState{}
	actual, _ := d.sessions.LoadOrStore(key, s)
	return actual.(*sessionState)
}

// BeforeExecute checks for loops before tool execution.
// Returns an error to block execution if a critical loop is detected.
func (d *LoopDetector) BeforeExecute(ctx context.Context, toolName string, args map[string]interface{}) error {
	sessionKey := sessionKeyFromContext(ctx)
	state := d.getSession(sessionKey)
	argsHash := hashArgs(args)

	state.mu.Lock()
	defer state.mu.Unlock()

	// Check for loops before recording this call
	verdict := d.detect(state, toolName, argsHash)

	// Record this call
	state.history = append(state.history, toolCallRecord{
		ToolName:  toolName,
		ArgsHash:  argsHash,
		Timestamp: time.Now(),
	})
	// Trim sliding window
	if len(state.history) > d.config.HistorySize {
		state.history = state.history[len(state.history)-d.config.HistorySize:]
	}

	if verdict.Blocked {
		logger.WarnCF("loop_detector", "Loop blocked",
			map[string]interface{}{
				"tool":    toolName,
				"reason":  verdict.Reason,
				"count":   verdict.Count,
				"session": sessionKey,
			})
		return fmt.Errorf("loop detected: %s (count: %d)", verdict.Reason, verdict.Count)
	}

	if verdict.Warning {
		logger.WarnCF("loop_detector", "Loop warning",
			map[string]interface{}{
				"tool":    toolName,
				"reason":  verdict.Reason,
				"count":   verdict.Count,
				"session": sessionKey,
			})
	}

	return nil
}

// AfterExecute records the tool result hash for no-progress detection.
func (d *LoopDetector) AfterExecute(ctx context.Context, toolName string, args map[string]interface{}, result *ToolResult) {
	sessionKey := sessionKeyFromContext(ctx)
	state := d.getSession(sessionKey)
	argsHash := hashArgs(args)
	resultHash := hashResult(result)

	state.mu.Lock()
	defer state.mu.Unlock()

	// Find the most recent matching record without a result hash
	for i := len(state.history) - 1; i >= 0; i-- {
		rec := &state.history[i]
		if rec.ToolName == toolName && rec.ArgsHash == argsHash && rec.ResultHash == "" {
			rec.ResultHash = resultHash
			break
		}
	}
}

// ResetSession clears the detection state for a session.
func (d *LoopDetector) ResetSession(sessionKey string) {
	d.sessions.Delete(sessionKey)
}

// detect runs all detection engines and returns the highest-severity verdict.
func (d *LoopDetector) detect(state *sessionState, toolName, argsHash string) LoopVerdict {
	// 1. Global circuit breaker: no-progress on same tool+args
	noProgressStreak := d.getNoProgressStreak(state, toolName, argsHash)
	if noProgressStreak >= d.config.CircuitBreakerThreshold {
		return LoopVerdict{
			Blocked: true,
			Reason:  "circuit breaker: repeated calls with identical results",
			Count:   noProgressStreak,
		}
	}

	// 2. Generic repeat detection (any tool+args combination)
	if d.config.EnableGenericRepeat {
		count := d.countRepeats(state, toolName, argsHash)
		if count >= d.config.CriticalThreshold {
			return LoopVerdict{
				Blocked: true,
				Reason:  "tool call repeated too many times",
				Count:   count,
			}
		}
		if count >= d.config.WarningThreshold {
			return LoopVerdict{
				Warning: true,
				Reason:  "possible loop: tool call repeated",
				Count:   count,
			}
		}
	}

	// 3. Ping-pong detection (alternating A,B,A,B pattern)
	if d.config.EnablePingPong {
		streak := d.getPingPongStreak(state, toolName, argsHash)
		if streak >= d.config.CriticalThreshold && d.hasPingPongNoProgress(state, streak) {
			return LoopVerdict{
				Blocked: true,
				Reason:  "ping-pong loop with no progress",
				Count:   streak,
			}
		}
		if streak >= d.config.WarningThreshold {
			return LoopVerdict{
				Warning: true,
				Reason:  "possible ping-pong pattern",
				Count:   streak,
			}
		}
	}

	return LoopVerdict{}
}

// countRepeats counts how many times this tool+args combination appears in history.
func (d *LoopDetector) countRepeats(state *sessionState, toolName, argsHash string) int {
	count := 0
	for _, rec := range state.history {
		if rec.ToolName == toolName && rec.ArgsHash == argsHash {
			count++
		}
	}
	return count
}

// getPingPongStreak detects alternating A,B,A,B patterns.
// Returns the alternation streak length (number of entries in the alternating tail).
func (d *LoopDetector) getPingPongStreak(state *sessionState, toolName, argsHash string) int {
	h := state.history
	if len(h) < 2 {
		return 0
	}

	currentSig := toolName + ":" + argsHash
	lastSig := h[len(h)-1].ToolName + ":" + h[len(h)-1].ArgsHash

	// Same signature as last call — not alternation
	if currentSig == lastSig {
		return 0
	}

	// Count alternating tail backwards from the end.
	// History ends with ...lastSig. Current call would be currentSig.
	// Pattern: ...currentSig, lastSig, currentSig, lastSig
	streak := 1 // count the last entry
	for i := len(h) - 2; i >= 0; i-- {
		sig := h[i].ToolName + ":" + h[i].ArgsHash
		// Positions from end: even indices should match lastSig, odd should match currentSig
		distFromEnd := len(h) - 1 - i
		var expected string
		if distFromEnd%2 == 0 {
			expected = lastSig
		} else {
			expected = currentSig
		}
		if sig != expected {
			break
		}
		streak++
	}

	return streak
}

// getNoProgressStreak counts consecutive tail calls with same tool+args AND same result.
func (d *LoopDetector) getNoProgressStreak(state *sessionState, toolName, argsHash string) int {
	h := state.history
	if len(h) == 0 {
		return 0
	}

	// Find the most recent matching entry with a result hash
	var referenceHash string
	for i := len(h) - 1; i >= 0; i-- {
		if h[i].ToolName == toolName && h[i].ArgsHash == argsHash && h[i].ResultHash != "" {
			referenceHash = h[i].ResultHash
			break
		}
	}
	if referenceHash == "" {
		return 0 // no completed calls to compare
	}

	// Count consecutive matching entries from the tail
	streak := 0
	for i := len(h) - 1; i >= 0; i-- {
		rec := h[i]
		if rec.ToolName != toolName || rec.ArgsHash != argsHash {
			break
		}
		if rec.ResultHash == "" {
			// Call recorded but not yet completed — include in streak (conservative)
			streak++
			continue
		}
		if rec.ResultHash != referenceHash {
			break // result changed = progress
		}
		streak++
	}

	return streak
}

// hasPingPongNoProgress checks that both sides of a ping-pong have stable (unchanged) results.
func (d *LoopDetector) hasPingPongNoProgress(state *sessionState, streak int) bool {
	h := state.history
	if len(h) < 4 || streak < 4 {
		return false
	}

	// Check the last 4 entries: [A, B, A, B]
	// Side A = indices -4, -2; Side B = indices -3, -1
	tail := h[len(h)-4:]
	if tail[0].ResultHash == "" || tail[1].ResultHash == "" ||
		tail[2].ResultHash == "" || tail[3].ResultHash == "" {
		return false // need result hashes on all 4
	}

	sideAStable := tail[0].ResultHash == tail[2].ResultHash
	sideBStable := tail[1].ResultHash == tail[3].ResultHash
	return sideAStable && sideBStable
}

// hashArgs produces a deterministic hash of tool arguments.
// Go's json.Marshal sorts map keys alphabetically, ensuring stability.
func hashArgs(args map[string]interface{}) string {
	if len(args) == 0 {
		return "empty"
	}
	data, err := json.Marshal(args)
	if err != nil {
		return "error"
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8]) // 16 hex chars — sufficient for dedup
}

// hashResult produces a hash of the tool result for no-progress detection.
func hashResult(result *ToolResult) string {
	if result == nil {
		return "nil"
	}
	content := result.ForLLM
	if len(content) > 1024 {
		content = content[:1024] // cap for hashing performance
	}
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h[:8])
}
