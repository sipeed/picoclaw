package agent

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	runtimeEventLoggerBuffer       = 256
	runtimeEventLoggerDrainTimeout = 2 * time.Second
)

type runtimeEventLogger struct {
	mu  sync.RWMutex
	cfg config.EventLoggingConfig
}

func (al *AgentLoop) refreshRuntimeEventLogger(cfg *config.Config) {
	if al == nil {
		return
	}
	logCfg := config.EffectiveEventLoggingConfig(cfg)

	al.runtimeEventLogMu.Lock()
	if !logCfg.Enabled {
		oldSub := al.runtimeEventLogSub
		al.runtimeEventLogger = nil
		al.runtimeEventLogSub = nil
		al.runtimeEventLogMu.Unlock()
		closeRuntimeEventLoggerSubscription(oldSub)
		return
	}

	if al.runtimeEventLogger != nil && al.runtimeEventLogSub != nil {
		al.runtimeEventLogger.updateConfig(logCfg)
		al.runtimeEventLogMu.Unlock()
		return
	}
	al.runtimeEventLogMu.Unlock()

	eventLogger := newRuntimeEventLoggerFromConfig(logCfg)
	sub, err := eventLogger.subscribe(context.Background(), al.runtimeEvents)
	if err != nil {
		logger.WarnCF("events", "Failed to subscribe runtime event logger", map[string]any{"error": err.Error()})
		return
	}

	al.runtimeEventLogMu.Lock()
	oldSub := al.runtimeEventLogSub
	al.runtimeEventLogger = eventLogger
	al.runtimeEventLogSub = sub
	al.runtimeEventLogMu.Unlock()
	closeRuntimeEventLoggerSubscription(oldSub)
}

func (al *AgentLoop) closeRuntimeEventLogger() {
	if al == nil {
		return
	}
	al.runtimeEventLogMu.Lock()
	oldSub := al.runtimeEventLogSub
	al.runtimeEventLogger = nil
	al.runtimeEventLogSub = nil
	al.runtimeEventLogMu.Unlock()
	closeRuntimeEventLoggerSubscription(oldSub)
}

func closeRuntimeEventLoggerSubscription(sub runtimeevents.Subscription) {
	if sub == nil {
		return
	}
	if err := sub.Close(); err != nil {
		logger.WarnCF("events", "Failed to close runtime event logger subscription", map[string]any{
			"error": err.Error(),
		})
	}

	timer := time.NewTimer(runtimeEventLoggerDrainTimeout)
	defer timer.Stop()
	select {
	case <-sub.Done():
	case <-timer.C:
		logger.WarnCF("events", "Timed out waiting for runtime event logger to drain", map[string]any{
			"timeout": runtimeEventLoggerDrainTimeout.String(),
		})
	}
}

func newRuntimeEventLogger(cfg *config.Config) *runtimeEventLogger {
	logCfg := config.EffectiveEventLoggingConfig(cfg)
	if !logCfg.Enabled {
		return nil
	}
	return newRuntimeEventLoggerFromConfig(logCfg)
}

func newRuntimeEventLoggerFromConfig(logCfg config.EventLoggingConfig) *runtimeEventLogger {
	return &runtimeEventLogger{cfg: logCfg}
}

func (l *runtimeEventLogger) updateConfig(cfg config.EventLoggingConfig) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.cfg = cfg
	l.mu.Unlock()
}

func (l *runtimeEventLogger) configSnapshot() config.EventLoggingConfig {
	if l == nil {
		return config.EventLoggingConfig{}
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.cfg
}

func (l *runtimeEventLogger) subscribe(
	ctx context.Context,
	eventBus runtimeevents.Bus,
) (runtimeevents.Subscription, error) {
	if l == nil || eventBus == nil {
		return nil, nil
	}
	return eventBus.Channel().Subscribe(ctx, runtimeevents.SubscribeOptions{
		Name:         "runtime-event-logger",
		Buffer:       runtimeEventLoggerBuffer,
		Concurrency:  runtimeevents.Locked,
		Backpressure: runtimeevents.DropNewest,
		PanicPolicy:  runtimeevents.RecoverAndLog,
	}, l.handle)
}

func (l *runtimeEventLogger) handle(_ context.Context, evt runtimeevents.Event) error {
	if l == nil || !l.shouldLog(evt) {
		return nil
	}

	fields := runtimeEventLogFields(evt)
	if l.configSnapshot().IncludePayload && evt.Payload != nil {
		fields["payload"] = evt.Payload
	}

	logRuntimeEvent(evt, fields)
	return nil
}

func (l *runtimeEventLogger) shouldLog(evt runtimeevents.Event) bool {
	if l == nil {
		return false
	}
	cfg := l.configSnapshot()
	if !cfg.Enabled {
		return false
	}
	if runtimeEventSeverityRank(evt.Severity) < runtimeEventSeverityRank(parseRuntimeEventSeverity(cfg.MinSeverity)) {
		return false
	}

	kind := evt.Kind.String()
	if !matchAnyRuntimeEventPattern(cfg.Include, kind, true) {
		return false
	}
	return !matchAnyRuntimeEventPattern(cfg.Exclude, kind, false)
}

func logRuntimeEvent(evt runtimeevents.Event, fields map[string]any) {
	message := fmt.Sprintf("Runtime event: %s", evt.Kind.String())
	switch normalizeRuntimeEventSeverity(evt.Severity) {
	case runtimeevents.SeverityDebug:
		logger.DebugCF("events", message, fields)
	case runtimeevents.SeverityWarn:
		logger.WarnCF("events", message, fields)
	case runtimeevents.SeverityError:
		logger.ErrorCF("events", message, fields)
	default:
		logger.InfoCF("events", message, fields)
	}
}

func runtimeEventLogFields(evt runtimeevents.Event) map[string]any {
	fields := map[string]any{
		"event_id":   evt.ID,
		"event_kind": evt.Kind.String(),
		"severity":   string(normalizeRuntimeEventSeverity(evt.Severity)),
	}
	if !evt.Time.IsZero() {
		fields["event_time"] = evt.Time.Format(time.RFC3339Nano)
	}
	appendRuntimeEventSourceFields(fields, evt.Source)
	appendRuntimeEventScopeFields(fields, evt.Scope)
	appendRuntimeEventCorrelationFields(fields, evt.Correlation)
	appendRuntimeEventAttrs(fields, evt.Attrs)
	appendRuntimeEventPayloadSummary(fields, evt.Payload)
	return fields
}

func appendRuntimeEventSourceFields(fields map[string]any, source runtimeevents.Source) {
	if source.Component != "" {
		fields["source_component"] = source.Component
	}
	if source.Name != "" {
		fields["source_name"] = source.Name
	}
}

func appendRuntimeEventScopeFields(fields map[string]any, scope runtimeevents.Scope) {
	setStringField(fields, "runtime_id", scope.RuntimeID)
	setStringField(fields, "agent_id", scope.AgentID)
	setStringField(fields, "session_key", scope.SessionKey)
	setStringField(fields, "turn_id", scope.TurnID)
	setStringField(fields, "channel", scope.Channel)
	setStringField(fields, "account", scope.Account)
	setStringField(fields, "chat_id", scope.ChatID)
	setStringField(fields, "topic_id", scope.TopicID)
	setStringField(fields, "space_id", scope.SpaceID)
	setStringField(fields, "space_type", scope.SpaceType)
	setStringField(fields, "chat_type", scope.ChatType)
	setStringField(fields, "sender_id", scope.SenderID)
	setStringField(fields, "message_id", scope.MessageID)
}

func appendRuntimeEventCorrelationFields(fields map[string]any, correlation runtimeevents.Correlation) {
	setStringField(fields, "trace_id", correlation.TraceID)
	setStringField(fields, "parent_turn_id", correlation.ParentTurnID)
	setStringField(fields, "request_id", correlation.RequestID)
	setStringField(fields, "reply_to_id", correlation.ReplyToID)
}

func appendRuntimeEventAttrs(fields map[string]any, attrs map[string]any) {
	for key, value := range attrs {
		if key == "" || value == nil {
			continue
		}
		if _, exists := fields[key]; exists {
			fields["attr_"+key] = value
			continue
		}
		fields[key] = value
	}
}

func appendRuntimeEventPayloadSummary(fields map[string]any, payload any) {
	switch payload := payload.(type) {
	case TurnStartPayload:
		fields["user_len"] = len(payload.UserMessage)
		fields["media_count"] = payload.MediaCount
	case TurnEndPayload:
		fields["status"] = payload.Status
		fields["iterations_total"] = payload.Iterations
		fields["duration_ms"] = payload.Duration.Milliseconds()
		fields["final_len"] = payload.FinalContentLen
	case LLMRequestPayload:
		fields["model"] = payload.Model
		fields["messages"] = payload.MessagesCount
		fields["tools"] = payload.ToolsCount
		fields["max_tokens"] = payload.MaxTokens
	case LLMDeltaPayload:
		fields["content_delta_len"] = payload.ContentDeltaLen
		fields["reasoning_delta_len"] = payload.ReasoningDeltaLen
	case LLMResponsePayload:
		fields["content_len"] = payload.ContentLen
		fields["tool_calls"] = payload.ToolCalls
		fields["has_reasoning"] = payload.HasReasoning
	case LLMRetryPayload:
		fields["attempt"] = payload.Attempt
		fields["max_retries"] = payload.MaxRetries
		fields["reason"] = payload.Reason
		fields["error"] = payload.Error
		fields["backoff_ms"] = payload.Backoff.Milliseconds()
	case ContextCompressPayload:
		fields["reason"] = payload.Reason
		fields["dropped_messages"] = payload.DroppedMessages
		fields["remaining_messages"] = payload.RemainingMessages
	case SessionSummarizePayload:
		fields["summarized_messages"] = payload.SummarizedMessages
		fields["kept_messages"] = payload.KeptMessages
		fields["summary_len"] = payload.SummaryLen
		fields["omitted_oversized"] = payload.OmittedOversized
	case ToolExecStartPayload:
		fields["tool"] = payload.Tool
		fields["args_count"] = len(payload.Arguments)
	case ToolExecEndPayload:
		fields["tool"] = payload.Tool
		fields["duration_ms"] = payload.Duration.Milliseconds()
		fields["for_llm_len"] = payload.ForLLMLen
		fields["for_user_len"] = payload.ForUserLen
		fields["is_error"] = payload.IsError
		fields["async"] = payload.Async
	case ToolExecSkippedPayload:
		fields["tool"] = payload.Tool
		fields["reason"] = payload.Reason
	case SteeringInjectedPayload:
		fields["count"] = payload.Count
		fields["total_content_len"] = payload.TotalContentLen
	case FollowUpQueuedPayload:
		fields["source_tool"] = payload.SourceTool
		fields["content_len"] = payload.ContentLen
	case InterruptReceivedPayload:
		fields["interrupt_kind"] = payload.Kind
		fields["role"] = payload.Role
		fields["content_len"] = payload.ContentLen
		fields["queue_depth"] = payload.QueueDepth
		fields["hint_len"] = payload.HintLen
	case SubTurnSpawnPayload:
		fields["child_agent_id"] = payload.AgentID
		fields["label"] = payload.Label
	case SubTurnEndPayload:
		fields["child_agent_id"] = payload.AgentID
		fields["status"] = payload.Status
	case SubTurnResultDeliveredPayload:
		fields["target_channel"] = payload.TargetChannel
		fields["target_chat_id"] = payload.TargetChatID
		fields["content_len"] = payload.ContentLen
	case SubTurnOrphanPayload:
		fields["parent_turn_id"] = payload.ParentTurnID
		fields["child_turn_id"] = payload.ChildTurnID
		fields["reason"] = payload.Reason
	case ErrorPayload:
		fields["stage"] = payload.Stage
		fields["error"] = payload.Message
	}
}

func setStringField(fields map[string]any, key, value string) {
	if value != "" {
		fields[key] = value
	}
}

func matchAnyRuntimeEventPattern(patterns []string, kind string, emptyMatches bool) bool {
	if len(patterns) == 0 {
		return emptyMatches
	}
	for _, pattern := range patterns {
		if matchRuntimeEventPattern(pattern, kind) {
			return true
		}
	}
	return false
}

func matchRuntimeEventPattern(pattern, kind string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		return strings.HasPrefix(kind, strings.TrimSuffix(pattern, "*"))
	}
	matched, err := path.Match(pattern, kind)
	if err == nil {
		return matched
	}
	return pattern == kind
}

func parseRuntimeEventSeverity(severity string) runtimeevents.Severity {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "debug":
		return runtimeevents.SeverityDebug
	case "warn", "warning":
		return runtimeevents.SeverityWarn
	case "error":
		return runtimeevents.SeverityError
	default:
		return runtimeevents.SeverityInfo
	}
}

func normalizeRuntimeEventSeverity(severity runtimeevents.Severity) runtimeevents.Severity {
	switch severity {
	case runtimeevents.SeverityDebug,
		runtimeevents.SeverityInfo,
		runtimeevents.SeverityWarn,
		runtimeevents.SeverityError:
		return severity
	default:
		return runtimeevents.SeverityInfo
	}
}

func runtimeEventSeverityRank(severity runtimeevents.Severity) int {
	switch normalizeRuntimeEventSeverity(severity) {
	case runtimeevents.SeverityDebug:
		return 0
	case runtimeevents.SeverityInfo:
		return 1
	case runtimeevents.SeverityWarn:
		return 2
	case runtimeevents.SeverityError:
		return 3
	default:
		return 1
	}
}
