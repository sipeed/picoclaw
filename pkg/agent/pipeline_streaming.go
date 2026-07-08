package agent

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func (p *Pipeline) tryConfiguredStreamingLLM(
	ctx context.Context,
	ts *turnState,
	exec *turnExecution,
	messagesForCall []providers.Message,
	toolDefsForCall []providers.ToolDefinition,
) (*providers.LLMResponse, bool, error) {
	exec.streamingPublisher = nil
	exec.streamingFallback = false
	if !p.configuredStreamingEligible(ts, exec) {
		return nil, false, nil
	}
	streamProvider, ok := exec.activeProvider.(providers.StreamingProvider)
	if !ok {
		logger.DebugCF("agent", "configured streaming not used", map[string]any{
			"agent_id": ts.agent.ID,
			"channel":  ts.channel,
			"model":    exec.activeModel,
			"reason":   "provider_not_streaming",
		})
		return nil, false, nil
	}

	streamer, ok := p.Bus.GetStreamer(ctx, ts.channel, ts.chatID, ts.sessionKey)
	if !ok || streamer == nil {
		logger.DebugCF("agent", "configured streaming not used", map[string]any{
			"agent_id": ts.agent.ID,
			"channel":  ts.channel,
			"chat_id":  ts.chatID,
			"model":    exec.activeModel,
			"reason":   "streamer_unavailable",
		})
		return nil, false, nil
	}

	publisher := &streamingChunkPublisher{
		streamer:  streamer,
		channel:   ts.channel,
		chatID:    ts.chatID,
		modelName: exec.llmModelName,
		ts:        ts,
	}

	logger.DebugCF("agent", "configured streaming enabled", map[string]any{
		"agent_id": ts.agent.ID,
		"channel":  ts.channel,
		"chat_id":  ts.chatID,
		"model":    exec.llmModel,
	})

	chunkCount := 0
	firstChunkAt := time.Time{}
	lastChunkAt := time.Time{}
	recordChunk := func() {
		now := time.Now()
		chunkCount++
		if firstChunkAt.IsZero() {
			firstChunkAt = now
		}
		lastChunkAt = now
	}
	var response *providers.LLMResponse
	var streamErr error
	if eventProvider, ok := exec.activeProvider.(providers.StreamingEventProvider); ok {
		response, streamErr = eventProvider.ChatStreamEvents(
			ctx,
			messagesForCall,
			toolDefsForCall,
			exec.llmModel,
			exec.llmOpts,
			func(chunk providers.StreamChunk) {
				recordChunk()
				if !exec.suppressReasoning && strings.TrimSpace(chunk.ReasoningContent) != "" {
					publisher.UpdateReasoning(ctx, chunk.ReasoningContent)
				}
				if strings.TrimSpace(chunk.Content) != "" {
					publisher.Update(ctx, chunk.Content)
				}
			},
		)
	} else {
		response, streamErr = streamProvider.ChatStream(
			ctx,
			messagesForCall,
			toolDefsForCall,
			exec.llmModel,
			exec.llmOpts,
			func(accumulated string) {
				recordChunk()
				publisher.Update(ctx, accumulated)
			},
		)
	}
	logConfiguredStreamingSummary(ts, exec, chunkCount, firstChunkAt, lastChunkAt, streamErr)
	if streamErr == nil {
		if updateErr := publisher.Err(); updateErr != nil {
			logFields := map[string]any{
				"agent_id": ts.agent.ID,
				"channel":  ts.channel,
				"model":    exec.llmModel,
				"error":    updateErr.Error(),
			}
			if publisher.Published() {
				logger.WarnCF("agent", "ChatStream update failed after visible output", logFields)
				return nil, true, configuredStreamingVisibleError{err: updateErr}
			}
			logger.WarnCF("agent", "ChatStream update failed before visible output; retrying with Chat", logFields)
			publisher.Cancel(ctx)
			fallbackResponse, err := exec.activeProvider.Chat(
				ctx,
				messagesForCall,
				toolDefsForCall,
				exec.llmModel,
				exec.llmOpts,
			)
			if err == nil && fallbackResponse != nil {
				exec.streamingFallback = true
			}
			return fallbackResponse, true, err
		}
	}
	if streamErr != nil {
		if !publisher.Published() {
			logger.WarnCF("agent", "ChatStream failed before visible output; retrying with Chat", map[string]any{
				"agent_id": ts.agent.ID,
				"channel":  ts.channel,
				"model":    exec.llmModel,
				"error":    streamErr.Error(),
			})
			publisher.Cancel(ctx)
			fallbackResponse, err := exec.activeProvider.Chat(
				ctx,
				messagesForCall,
				toolDefsForCall,
				exec.llmModel,
				exec.llmOpts,
			)
			if err == nil && fallbackResponse != nil {
				exec.streamingFallback = true
			}
			return fallbackResponse, true, err
		}
		return nil, true, configuredStreamingVisibleError{err: streamErr}
	}

	if response != nil {
		exec.streamingPublisher = publisher
	}

	return response, true, nil
}

func logConfiguredStreamingSummary(
	ts *turnState,
	exec *turnExecution,
	chunkCount int,
	firstChunkAt time.Time,
	lastChunkAt time.Time,
	streamErr error,
) {
	fields := map[string]any{
		"chunks": chunkCount,
	}
	if ts != nil {
		fields["agent_id"] = ts.agent.ID
		fields["channel"] = ts.channel
	}
	if exec != nil {
		fields["model"] = exec.llmModel
	}
	if !firstChunkAt.IsZero() && !lastChunkAt.IsZero() {
		fields["chunk_span_ms"] = lastChunkAt.Sub(firstChunkAt).Milliseconds()
	}
	if streamErr != nil {
		fields["error"] = streamErr.Error()
	}
	logger.DebugCF("agent", "configured streaming completed", fields)
}

type configuredStreamingVisibleError struct {
	err error
}

func (e configuredStreamingVisibleError) Error() string {
	if e.err == nil {
		return "configured streaming failed after visible output"
	}
	return e.err.Error()
}

func (e configuredStreamingVisibleError) Unwrap() error {
	return e.err
}

func isConfiguredStreamingVisibleError(err error) bool {
	var visibleErr configuredStreamingVisibleError
	return errors.As(err, &visibleErr)
}

func finalizeConfiguredStreamingLLM(
	ctx context.Context,
	ts *turnState,
	exec *turnExecution,
	content string,
	contextUsage *bus.ContextUsage,
) error {
	if exec == nil || exec.streamingPublisher == nil {
		return nil
	}
	publisher := exec.streamingPublisher
	exec.streamingPublisher = nil
	visibleBeforeFinalize := publisher.Published()
	if err := publisher.Finalize(ctx, content, contextUsage); err != nil {
		if visibleBeforeFinalize {
			logger.WarnCF("agent", "stream final flush failed after visible output", map[string]any{
				"agent_id": ts.agent.ID,
				"channel":  ts.channel,
				"model":    exec.llmModel,
				"error":    err.Error(),
			})
			return configuredStreamingVisibleError{err: err}
		}
		publisher.Cancel(ctx)
		logger.WarnCF("agent", "stream final flush failed", map[string]any{
			"agent_id": ts.agent.ID,
			"channel":  ts.channel,
			"model":    exec.llmModel,
			"error":    err.Error(),
		})
		return err
	}
	return nil
}

func cancelConfiguredStreamingLLM(ctx context.Context, exec *turnExecution) {
	if exec == nil || exec.streamingPublisher == nil {
		return
	}
	publisher := exec.streamingPublisher
	exec.streamingPublisher = nil
	publisher.Cancel(ctx)
}

func (p *Pipeline) configuredStreamingEligible(ts *turnState, exec *turnExecution) bool {
	if p == nil || ts == nil || exec == nil || p.Bus == nil {
		logger.DebugCF("agent", "configured streaming not used", map[string]any{
			"reason": "missing_pipeline_state",
		})
		return false
	}
	if strings.TrimSpace(ts.channel) == "" || strings.TrimSpace(ts.chatID) == "" {
		logger.DebugCF("agent", "configured streaming not used", map[string]any{
			"agent_id": ts.agent.ID,
			"channel":  ts.channel,
			"chat_id":  ts.chatID,
			"model":    exec.activeModel,
			"reason":   "missing_channel_context",
		})
		return false
	}
	if !ts.opts.SendResponse && !ts.opts.AllowInterimPicoPublish {
		logger.DebugCF("agent", "configured streaming not used", map[string]any{
			"agent_id": ts.agent.ID,
			"channel":  ts.channel,
			"chat_id":  ts.chatID,
			"model":    exec.activeModel,
			"reason":   "turn_output_disabled",
		})
		return false
	}
	if len(exec.activeCandidates) != 1 {
		logger.DebugCF("agent", "configured streaming not used", map[string]any{
			"agent_id":   ts.agent.ID,
			"channel":    ts.channel,
			"model":      exec.activeModel,
			"candidates": len(exec.activeCandidates),
			"reason":     "fallback_candidates_enabled",
		})
		return false
	}
	if exec.activeModelConfig == nil || !exec.activeModelConfig.Streaming.Enabled {
		modelName := ""
		modelStreaming := false
		if exec.activeModelConfig != nil {
			modelName = exec.activeModelConfig.ModelName
			modelStreaming = exec.activeModelConfig.Streaming.Enabled
		}
		logger.DebugCF("agent", "configured streaming not used", map[string]any{
			"agent_id":         ts.agent.ID,
			"channel":          ts.channel,
			"model":            exec.activeModel,
			"model_name":       modelName,
			"model_streaming":  modelStreaming,
			"has_model_config": exec.activeModelConfig != nil,
			"reason":           "model_streaming_disabled",
		})
		return false
	}
	channelStreaming, ok := p.channelStreamingConfig(ts.channel)
	if !ok || !channelStreaming.Enabled {
		logger.DebugCF("agent", "configured streaming not used", map[string]any{
			"agent_id":           ts.agent.ID,
			"channel":            ts.channel,
			"model":              exec.activeModel,
			"channel_streaming":  channelStreaming.Enabled,
			"has_channel_config": ok,
			"reason":             "channel_streaming_disabled",
		})
		return false
	}
	return true
}

func (p *Pipeline) channelStreamingConfig(channelName string) (config.StreamingConfig, bool) {
	if p == nil || p.Cfg == nil || p.Cfg.Channels == nil {
		return config.StreamingConfig{}, false
	}
	ch := p.Cfg.Channels[channelName]
	if ch == nil {
		return config.StreamingConfig{}, false
	}
	decoded, err := ch.GetDecoded()
	if err != nil {
		logger.WarnCF("agent", "channel streaming config decode failed", map[string]any{
			"channel": channelName,
			"error":   err.Error(),
		})
		return config.StreamingConfig{}, false
	}
	return streamingConfigFromDecodedSettings(decoded)
}

func streamingConfigFromDecodedSettings(decoded any) (config.StreamingConfig, bool) {
	value := reflect.ValueOf(decoded)
	if !value.IsValid() {
		return config.StreamingConfig{}, false
	}
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return config.StreamingConfig{}, false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return config.StreamingConfig{}, false
	}

	field := value.FieldByName("Streaming")
	if !field.IsValid() || !field.CanInterface() {
		return config.StreamingConfig{}, false
	}
	streaming, ok := field.Interface().(config.StreamingConfig)
	return streaming, ok
}

type streamingChunkPublisher struct {
	streamer           bus.Streamer
	channel            string
	chatID             string
	modelName          string
	published          bool
	reasoningPublished bool
	err                error
	ts                 *turnState
}

func (p *streamingChunkPublisher) Update(ctx context.Context, accumulated string) {
	if p == nil || p.streamer == nil || strings.TrimSpace(accumulated) == "" {
		return
	}
	if setter, ok := p.streamer.(interface{ SetModelName(modelName string) }); ok {
		setter.SetModelName(p.modelName)
	}
	if err := p.streamer.Update(ctx, accumulated); err != nil {
		p.err = err
		logger.WarnCF("agent", "stream update failed", map[string]any{
			"channel": p.channel,
			"chat_id": p.chatID,
			"error":   err.Error(),
		})
		return
	}
	p.published = true
}

func (p *streamingChunkPublisher) UpdateReasoning(ctx context.Context, accumulated string) {
	if p == nil || p.streamer == nil || strings.TrimSpace(accumulated) == "" {
		return
	}
	if setter, ok := p.streamer.(interface{ SetModelName(modelName string) }); ok {
		setter.SetModelName(p.modelName)
	}
	reasoningStreamer, ok := p.streamer.(bus.ReasoningStreamer)
	if !ok {
		return
	}
	if err := reasoningStreamer.UpdateReasoning(ctx, accumulated); err != nil {
		p.err = err
		logger.WarnCF("agent", "stream reasoning update failed", map[string]any{
			"channel": p.channel,
			"chat_id": p.chatID,
			"error":   err.Error(),
		})
		return
	}
	p.reasoningPublished = true
}

func (p *streamingChunkPublisher) Published() bool {
	return p != nil && p.published
}

func (p *streamingChunkPublisher) ReasoningPublished() bool {
	return p != nil && p.reasoningPublished
}

func (p *streamingChunkPublisher) Err() error {
	if p == nil {
		return nil
	}
	return p.err
}

func (p *streamingChunkPublisher) Finalize(ctx context.Context, content string, contextUsage *bus.ContextUsage) error {
	if p == nil || p.streamer == nil {
		return nil
	}
	if strings.TrimSpace(content) == "" && !p.published {
		return nil
	}
	if setter, ok := p.streamer.(interface{ SetModelName(modelName string) }); ok {
		setter.SetModelName(p.modelName)
	}
	if usage := p.ts.GetLastUsage(); usage != nil {
		if setter, ok := p.streamer.(interface{ SetTurnUsage(in, out int) }); ok {
			setter.SetTurnUsage(usage.PromptTokens, usage.CompletionTokens)
		}
	}
	var err error
	if streamer, ok := p.streamer.(bus.ContextUsageStreamer); ok {
		err = streamer.FinalizeWithContext(ctx, content, contextUsage)
	} else {
		err = p.streamer.Finalize(ctx, content)
	}
	if err != nil {
		return fmt.Errorf("stream finalize: %w", err)
	}
	p.published = true
	return nil
}

func (p *streamingChunkPublisher) FinalizeReasoning(ctx context.Context, content string) error {
	if p == nil || p.streamer == nil || !p.reasoningPublished || strings.TrimSpace(content) == "" {
		return nil
	}
	reasoningStreamer, ok := p.streamer.(bus.ReasoningStreamer)
	if !ok {
		return nil
	}
	if err := reasoningStreamer.FinalizeReasoning(ctx, content); err != nil {
		return fmt.Errorf("stream reasoning finalize: %w", err)
	}
	return nil
}

func (p *streamingChunkPublisher) ClearFinalizedStreamMarker() {
	if p == nil || p.streamer == nil {
		return
	}
	if cleaner, ok := p.streamer.(interface{ ClearFinalizedStreamMarker() }); ok {
		cleaner.ClearFinalizedStreamMarker()
	}
}

func (p *streamingChunkPublisher) Cancel(ctx context.Context) {
	if p == nil || p.streamer == nil {
		return
	}
	p.streamer.Cancel(ctx)
}
