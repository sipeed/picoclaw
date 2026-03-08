package llmlog

import (
	"time"
)

// Message 表示简化的消息结构，用于日志记录
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CallRecord 表示一次 LLM 调用记录
type CallRecord struct {
	Timestamp   time.Time     `json:"timestamp"`
	Model       string        `json:"model"`
	Provider    string        `json:"provider"`
	Messages    []Message     `json:"messages"`
	Response    string        `json:"response"`
	Duration    time.Duration `json:"duration"`
	TokensUsed  int           `json:"tokens_used,omitempty"`
	Error       string        `json:"error,omitempty"`
	IsStreaming bool          `json:"is_streaming"`
}

// Logger 定义 LLM 调用日志记录器接口
type Logger interface {
	IsEnabled() bool
	Log(record *CallRecord) error
}