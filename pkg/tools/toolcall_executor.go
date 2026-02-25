package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// ToolCallParallelConfig configures in-batch parallel execution for tool calls.
type ToolCallParallelConfig struct {
	Enabled        bool
	MaxConcurrency int
	Mode           string
	// ToolPolicyOverrides allows per-tool policy overrides.
	// Values: "serial_only" or "parallel_read_only".
	ToolPolicyOverrides map[string]string
}

// ToolCallExecutionOptions controls how tool calls are executed.
type ToolCallExecutionOptions struct {
	Channel  string
	ChatID   string
	SenderID string

	Iteration int
	LogScope  string

	Parallel ToolCallParallelConfig

	// AsyncCallbackForCall creates a callback for async-capable tools.
	// It may be nil when async callbacks are not needed.
	AsyncCallbackForCall func(call providers.ToolCall) AsyncCallback
}

// ToolCallExecution captures one tool call execution result.
type ToolCallExecution struct {
	ToolCall   providers.ToolCall
	Result     *ToolResult
	DurationMS int64
}

// ExecuteToolCalls executes tool calls with optional bounded parallelism while
// preserving output order exactly as provided in the input slice.
func ExecuteToolCalls(
	ctx context.Context,
	registry *ToolRegistry,
	toolCalls []providers.ToolCall,
	opts ToolCallExecutionOptions,
) []ToolCallExecution {
	if len(toolCalls) == 0 {
		return nil
	}
	batchStart := time.Now()

	scope := opts.LogScope
	if scope == "" {
		scope = "tool"
	}

	results := make([]ToolCallExecution, len(toolCalls))
	parallelCount := 0
	serialCount := 0
	mode := normalizeParallelMode(opts.Parallel.Mode)

	shouldParallelize := func(tc providers.ToolCall) bool {
		if registry == nil {
			return false
		}
		if !opts.Parallel.Enabled {
			return false
		}
		if opts.Parallel.MaxConcurrency == 1 {
			return false
		}
		if !registry.IsParallelInstanceSafe(tc.Name) {
			return false
		}
		if override, ok := getOverridePolicy(tc.Name, opts.Parallel.ToolPolicyOverrides); ok {
			return override == ToolParallelReadOnly
		}
		switch mode {
		case ParallelToolsModeAll:
			return true
		case ParallelToolsModeReadOnlyOnly:
			return registry.CanRunToolCallInParallel(tc.Name, ParallelToolsModeReadOnlyOnly)
		default:
			return false
		}
	}

	runOne := func(idx int) {
		tc := toolCalls[idx]
		argsJSON, _ := json.Marshal(tc.Arguments)
		argsPreview := utils.Truncate(string(argsJSON), 200)
		logger.InfoCF(scope, fmt.Sprintf("Tool call: %s(%s)", tc.Name, argsPreview),
			map[string]any{
				"tool":      tc.Name,
				"iteration": opts.Iteration,
			})

		var asyncCallback AsyncCallback
		if opts.AsyncCallbackForCall != nil {
			asyncCallback = opts.AsyncCallbackForCall(tc)
		}

		start := time.Now()
		var toolResult *ToolResult
		if registry != nil {
			toolResult = registry.ExecuteWithContext(
				ctx,
				tc.Name,
				tc.Arguments,
				opts.Channel,
				opts.ChatID,
				opts.SenderID,
				asyncCallback,
			)
		} else {
			toolResult = ErrorResult("No tools available")
		}

		if toolResult == nil {
			toolResult = ErrorResult(fmt.Sprintf("tool %q returned nil result", tc.Name)).
				WithError(fmt.Errorf("tool %q returned nil result", tc.Name))
		}

		results[idx] = ToolCallExecution{
			ToolCall:   tc,
			Result:     toolResult,
			DurationMS: time.Since(start).Milliseconds(),
		}
	}

	runParallelBatch := func(batch []int) {
		if len(batch) == 0 {
			return
		}

		maxConc := opts.Parallel.MaxConcurrency
		if maxConc <= 0 || maxConc > len(batch) {
			maxConc = len(batch)
		}
		if maxConc <= 1 {
			for _, idx := range batch {
				runOne(idx)
			}
			return
		}

		logger.DebugCF(scope, "Executing parallel tool batch", map[string]any{
			"iteration":     opts.Iteration,
			"batch_size":    len(batch),
			"max_parallel":  maxConc,
			"parallel_mode": mode,
		})

		jobs := make(chan int)
		var wg sync.WaitGroup
		for i := 0; i < maxConc; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for idx := range jobs {
					runOne(idx)
				}
			}()
		}

		for _, idx := range batch {
			jobs <- idx
		}
		close(jobs)
		wg.Wait()
	}

	parallelBatch := make([]int, 0, len(toolCalls))
	flushParallelBatch := func() {
		if len(parallelBatch) == 0 {
			return
		}
		runParallelBatch(parallelBatch)
		parallelBatch = parallelBatch[:0]
	}

	for i, tc := range toolCalls {
		if shouldParallelize(tc) {
			parallelCount++
			parallelBatch = append(parallelBatch, i)
			continue
		}
		serialCount++
		flushParallelBatch()
		runOne(i)
	}
	flushParallelBatch()

	errorCount := 0
	durations := make([]int64, 0, len(results))
	for _, executed := range results {
		if executed.Result != nil && executed.Result.IsError {
			errorCount++
		}
		durations = append(durations, executed.DurationMS)
	}
	p50, p95, avg, max := summarizeDurations(durations)

	logger.InfoCF(scope, "Tool call batch summary", map[string]any{
		"iteration":                 opts.Iteration,
		"tool_parallel_enabled":     opts.Parallel.Enabled,
		"max_tool_concurrency":      opts.Parallel.MaxConcurrency,
		"parallel_tools_mode":       mode,
		"parallel_candidate_count":  parallelCount,
		"serial_count":              serialCount,
		"total":                     len(toolCalls),
		"error_count":               errorCount,
		"batch_duration_ms":         time.Since(batchStart).Milliseconds(),
		"tool_call_duration_p50_ms": p50,
		"tool_call_duration_p95_ms": p95,
		"tool_call_duration_avg_ms": avg,
		"tool_call_duration_max_ms": max,
	})

	return results
}

func normalizeParallelMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", ParallelToolsModeReadOnlyOnly:
		return ParallelToolsModeReadOnlyOnly
	case ParallelToolsModeAll:
		return ParallelToolsModeAll
	default:
		return ""
	}
}

func getOverridePolicy(toolName string, overrides map[string]string) (ToolParallelPolicy, bool) {
	if len(overrides) == 0 {
		return "", false
	}
	raw, ok := overrides[toolName]
	if !ok {
		raw, ok = overrides[strings.ToLower(strings.TrimSpace(toolName))]
	}
	if !ok {
		return "", false
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(ToolParallelSerialOnly):
		return ToolParallelSerialOnly, true
	case string(ToolParallelReadOnly):
		return ToolParallelReadOnly, true
	default:
		return "", false
	}
}

func summarizeDurations(durations []int64) (p50, p95, avg, max int64) {
	if len(durations) == 0 {
		return 0, 0, 0, 0
	}

	sorted := append([]int64(nil), durations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	total := int64(0)
	for _, d := range sorted {
		total += d
	}
	avg = total / int64(len(sorted))
	max = sorted[len(sorted)-1]
	p50 = percentileInt64(sorted, 0.50)
	p95 = percentileInt64(sorted, 0.95)
	return p50, p95, avg, max
}

func percentileInt64(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	// Nearest-rank percentile: rank = ceil(p*n), index = rank-1.
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
