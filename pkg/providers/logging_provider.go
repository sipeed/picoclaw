package providers

import (
	"context"
	"time"

	"github.com/sipeed/picoclaw/pkg/llmlog"
)

// LoggingProvider 包装一个 LLMProvider，记录所有调用日志
type LoggingProvider struct {
	inner    LLMProvider
	logger   llmlog.Logger
	provider string // provider 名称，用于日志
}

// NewLoggingProvider 创建一个带日志功能的 Provider 包装器
func NewLoggingProvider(inner LLMProvider, logger llmlog.Logger, providerName string) *LoggingProvider {
	return &LoggingProvider{
		inner:    inner,
		logger:   logger,
		provider: providerName,
	}
}

// Chat 实现 LLMProvider 接口，记录调用日志
func (p *LoggingProvider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	// 如果日志未启用，直接调用内部 provider
	if p.logger == nil || !p.logger.IsEnabled() {
		return p.inner.Chat(ctx, messages, tools, model, options)
	}

	// 记录开始时间
	startTime := time.Now()

	// 调用内部 provider
	resp, err := p.inner.Chat(ctx, messages, tools, model, options)

	// 转换消息格式用于日志
	logMessages := make([]llmlog.Message, len(messages))
	for i, m := range messages {
		logMessages[i] = llmlog.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	// 记录调用
	record := &llmlog.CallRecord{
		Timestamp:   startTime,
		Model:       model,
		Provider:    p.provider,
		Messages:    logMessages,
		Duration:    time.Since(startTime),
		IsStreaming: false,
	}

	if err != nil {
		record.Error = err.Error()
	} else {
		record.Response = resp.Content
		if resp.Usage != nil {
			record.TokensUsed = resp.Usage.TotalTokens
		}
	}

	// 异步写入日志，避免阻塞主流程
	go p.logger.Log(record)

	return resp, err
}

// GetDefaultModel 返回内部 provider 的默认模型
func (p *LoggingProvider) GetDefaultModel() string {
	return p.inner.GetDefaultModel()
}