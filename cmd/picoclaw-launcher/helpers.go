package main

import (
	"strings"
	"sync"
)

// LogBuffer is a thread-safe ring buffer that stores the most recent N log lines.
type LogBuffer struct {
	mu    sync.RWMutex
	lines []string
	cap   int
	total int
	runID int
}

func NewLogBuffer(capacity int) *LogBuffer {
	return &LogBuffer{lines: make([]string, 0, capacity), cap: capacity}
}

func (b *LogBuffer) Append(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.lines) < b.cap {
		b.lines = append(b.lines, line)
	} else {
		b.lines[b.total%b.cap] = line
	}
	b.total++
}

func (b *LogBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = b.lines[:0]
	b.total = 0
	b.runID++
}

func (b *LogBuffer) LinesSince(offset int) (lines []string, total int, runID int) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	total = b.total
	runID = b.runID
	if offset >= b.total {
		return nil, total, runID
	}
	buffered := len(b.lines)
	newCount := b.total - offset
	if newCount > buffered {
		newCount = buffered
	}
	result := make([]string, newCount)
	if b.total <= b.cap {
		copy(result, b.lines[buffered-newCount:])
	} else {
		start := (b.total - newCount) % b.cap
		for i := range newCount {
			result[i] = b.lines[(start+i)%b.cap]
		}
	}
	return result, total, runID
}

// buildModelField constructs the "protocol/model" string for config.
// If the model already starts with the detected protocol prefix, it is returned as-is.
// Otherwise the protocol prefix is prepended.
// E.g. buildModelField("nvidia", "nvidia/minimaxai/minimax-m2.5") → "nvidia/minimaxai/minimax-m2.5"
//
//	buildModelField("openai", "gpt-4o") → "openai/gpt-4o"
func buildModelField(protocol, model string) string {
	if strings.HasPrefix(model, protocol+"/") {
		return model
	}
	return protocol + "/" + model
}

// detectProtocol guesses the provider protocol from the API base URL.
func detectProtocol(baseURL string) string {
	lower := strings.ToLower(baseURL)
	switch {
	case strings.Contains(lower, "anthropic"):
		return "anthropic"
	case strings.Contains(lower, "googleapis") || strings.Contains(lower, "generativelanguage"):
		return "gemini"
	case strings.Contains(lower, "openrouter"):
		return "openrouter"
	case strings.Contains(lower, "nvidia") || strings.Contains(lower, "integrate.api"):
		return "nvidia"
	case strings.Contains(lower, "deepseek"):
		return "deepseek"
	case strings.Contains(lower, "groq"):
		return "groq"
	case strings.Contains(lower, "bigmodel.cn") || strings.Contains(lower, "zhipu"):
		return "zhipu"
	case strings.Contains(lower, "moonshot"):
		return "moonshot"
	case strings.Contains(lower, "dashscope") || strings.Contains(lower, "aliyun"):
		return "qwen"
	case strings.Contains(lower, "cerebras"):
		return "cerebras"
	case strings.Contains(lower, "volces.com") || strings.Contains(lower, "volcengine"):
		return "volcengine"
	case strings.Contains(lower, "shengsuanyun"):
		return "shengsuanyun"
	case strings.Contains(lower, "mistral"):
		return "mistral"
	case strings.Contains(lower, "localhost:11434") || strings.Contains(lower, "ollama"):
		return "ollama"
	case strings.Contains(lower, "localhost:8000"):
		return "vllm"
	default:
		return "openai"
	}
}
