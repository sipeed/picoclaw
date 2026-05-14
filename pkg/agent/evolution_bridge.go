package agent

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/evolution"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type evolutionBridge struct {
	cfg            config.EvolutionConfig
	registry       *AgentRegistry
	runtime        *evolution.Runtime
	coldPathRunner *evolution.ColdPathRunner
	runtimeSub     runtimeevents.Subscription
	bgCtx          context.Context
	cancel         context.CancelFunc
	closeMu        sync.Mutex
	closed         bool
	wg             sync.WaitGroup
	isCurrent      func(*evolutionBridge) bool

	scheduledMu         sync.Mutex
	scheduledWorkspaces map[string]struct{}
}

const evolutionDirectDeliveryAttr = "evolution_direct_delivery"

func newEvolutionBridge(
	registry *AgentRegistry,
	cfg *config.Config,
	provider providers.LLMProvider,
) (*evolutionBridge, error) {
	if cfg == nil {
		return nil, nil
	}

	modelID := resolvedEvolutionModelID(cfg, provider)
	runtime, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: cfg.Evolution,
		PatternClusterer: evolution.NewLLMPatternClusterer(
			provider,
			modelID,
			evolution.NewHeuristicPatternClusterer(cfg.Evolution.EffectiveMinTaskCount(), nil),
			cfg.Evolution.EffectiveMinTaskCount(),
			nil,
		),
		GeneratorFactory: func(workspace string) evolution.DraftGenerator {
			return evolution.NewDraftGeneratorForWorkspace(workspace, provider, modelID)
		},
		SuccessJudgeFactory: func(workspace string) evolution.SuccessJudge {
			return evolution.NewLLMTaskSuccessJudge(provider, modelID, &evolution.HeuristicSuccessJudge{})
		},
		ApplierFactory: func(workspace string) *evolution.Applier {
			return evolution.NewApplier(evolution.NewPaths(workspace, cfg.Evolution.StateDir), nil)
		},
	})
	if err != nil {
		return nil, err
	}
	bgCtx, cancel := context.WithCancel(context.Background())

	bridge := &evolutionBridge{
		cfg:      cfg.Evolution,
		registry: registry,
		runtime:  runtime,
		bgCtx:    bgCtx,
		cancel:   cancel,
	}
	if cfg.Evolution.RunsColdPathAutomatically() {
		bridge.coldPathRunner = evolution.NewColdPathRunnerWithErrorHandler(runtime, func(err error) {
			logger.WarnCF("agent", "Cold path run failed", map[string]any{
				"error": err.Error(),
			})
		})
	}
	if cfg.Evolution.RunsColdPathScheduled() {
		bridge.startScheduledColdPath(cfg.Agents.Defaults.Workspace, cfg.Evolution.EffectiveColdPathTimes())
		bridge.rememberScheduledColdPathWorkspaces(registryWorkspaces(registry))
	}

	return bridge, nil
}

func resolvedEvolutionModelID(cfg *config.Config, provider providers.LLMProvider) string {
	if cfg != nil {
		if modelID := cfg.Agents.Defaults.GetModelName(); modelID != "" {
			return modelID
		}
	}
	if provider != nil {
		return provider.GetDefaultModel()
	}
	return ""
}

func (b *evolutionBridge) Close() error {
	if b == nil {
		return nil
	}

	if b.runtimeSub != nil {
		if err := b.runtimeSub.Close(); err != nil {
			logger.WarnCF("agent", "Failed to close evolution runtime subscription", map[string]any{
				"error": err.Error(),
			})
		}
		<-b.runtimeSub.Done()
	}

	b.closeMu.Lock()
	alreadyClosed := b.closed
	b.closed = true
	b.closeMu.Unlock()
	if alreadyClosed {
		return nil
	}
	if b.cancel != nil {
		b.cancel()
	}
	var closeErr error
	if b.coldPathRunner != nil {
		closeErr = b.coldPathRunner.Close()
	}
	b.wg.Wait()
	return closeErr
}

func (b *evolutionBridge) OnEvent(_ context.Context, evt Event) error {
	if b == nil || !b.cfg.Enabled || b.runtime == nil {
		return nil
	}

	switch evt.Kind {
	case EventKindTurnEnd:
		payload, ok := evt.Payload.(TurnEndPayload)
		if !ok {
			return nil
		}
		b.handleTurnEndAsync(evt.Meta, payload)
		return nil
	}

	return nil
}

func (b *evolutionBridge) OnRuntimeEvent(_ context.Context, evt runtimeevents.Event) error {
	if b == nil || !b.cfg.Enabled || b.runtime == nil || evt.Kind != runtimeevents.KindAgentTurnEnd {
		return nil
	}
	if b.isCurrent != nil && !b.isCurrent(b) {
		return nil
	}
	if deliveredDirectly, _ := evt.Attrs[evolutionDirectDeliveryAttr].(bool); deliveredDirectly {
		return nil
	}
	payload, ok := evt.Payload.(TurnEndPayload)
	if !ok {
		return nil
	}
	b.handleTurnEndAsync(hookMetaFromRuntimeEvent(evt), payload)
	return nil
}

func (b *evolutionBridge) handleRuntimeTurnEnd(evt runtimeevents.Event) bool {
	if b == nil || !b.cfg.Enabled || b.runtime == nil || evt.Kind != runtimeevents.KindAgentTurnEnd {
		return false
	}
	payload, ok := evt.Payload.(TurnEndPayload)
	if !ok {
		return false
	}
	return b.handleTurnEndAsync(hookMetaFromRuntimeEvent(evt), payload)
}

func (b *evolutionBridge) handleTurnEndAsync(meta EventMeta, payload TurnEndPayload) bool {
	if b == nil || b.runtime == nil {
		return false
	}

	input := evolution.TurnCaseInput{
		Workspace:             payload.Workspace,
		WorkspaceID:           payload.Workspace,
		TurnID:                meta.TurnID,
		SessionKey:            meta.SessionKey,
		AgentID:               meta.AgentID,
		Status:                string(payload.Status),
		UserMessage:           payload.UserMessage,
		FinalContent:          payload.FinalContent,
		ToolKinds:             append([]string(nil), payload.ToolKinds...),
		ToolExecutions:        toEvolutionToolExecutions(payload.ToolExecutions),
		ActiveSkillNames:      append([]string(nil), payload.ActiveSkills...),
		AttemptedSkillNames:   append([]string(nil), payload.AttemptedSkills...),
		FinalSuccessfulPath:   append([]string(nil), payload.FinalSuccessfulPath...),
		SkillContextSnapshots: toEvolutionSkillContextSnapshots(payload.SkillContextSnapshots),
	}
	b.rememberScheduledColdPathWorkspace(input.Workspace)

	b.closeMu.Lock()
	if b.closed {
		b.closeMu.Unlock()
		return false
	}
	b.wg.Add(1)
	b.closeMu.Unlock()
	go func() {
		defer b.wg.Done()
		if err := b.runtime.FinalizeTurn(b.bgCtx, input); err != nil {
			logger.WarnCF("agent", "Evolution finalize turn failed", map[string]any{
				"error":     err.Error(),
				"turn_id":   input.TurnID,
				"workspace": input.Workspace,
			})
			return
		}
		if b.coldPathRunner != nil && b.cfg.RunsColdPathAfterTurn() {
			b.coldPathRunner.Trigger(input.Workspace)
		}
	}()
	return true
}

func (b *evolutionBridge) subscribeRuntimeEvents(ch runtimeevents.EventChannel) error {
	if b == nil || ch == nil {
		return nil
	}
	sub, err := ch.Source("agent").OfKind(runtimeevents.KindAgentTurnEnd).Subscribe(
		b.bgCtx,
		runtimeevents.SubscribeOptions{
			Name:         "evolution-bridge",
			Buffer:       hookObserverBufferSize,
			Backpressure: runtimeevents.Block,
			Concurrency:  runtimeevents.Locked,
		},
		func(ctx context.Context, evt runtimeevents.Event) error {
			return b.OnRuntimeEvent(ctx, evt)
		},
	)
	if err != nil {
		return err
	}
	b.runtimeSub = sub
	return nil
}

func (b *evolutionBridge) setCurrentCheck(check func(*evolutionBridge) bool) {
	if b == nil {
		return
	}
	b.closeMu.Lock()
	defer b.closeMu.Unlock()
	b.isCurrent = check
}

func (b *evolutionBridge) startScheduledColdPath(workspace string, times []string) {
	if b == nil || b.coldPathRunner == nil || len(times) == 0 {
		return
	}
	b.rememberScheduledColdPathWorkspace(workspace)
	schedule := parseColdPathSchedule(times)
	if len(schedule) == 0 {
		logger.WarnCF("agent", "No valid evolution cold path schedule times configured", map[string]any{
			"times": times,
		})
		return
	}

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		for {
			now := time.Now()
			next := nextColdPathScheduledTime(now, schedule)
			timer := time.NewTimer(time.Until(next))
			select {
			case <-timer.C:
				for _, workspace := range b.scheduledColdPathWorkspaces() {
					b.coldPathRunner.Trigger(workspace)
				}
			case <-b.bgCtx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return
			}
		}
	}()
}

func (b *evolutionBridge) rememberScheduledColdPathWorkspace(workspace string) {
	if b == nil || !b.cfg.RunsColdPathScheduled() {
		return
	}
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return
	}
	b.scheduledMu.Lock()
	defer b.scheduledMu.Unlock()
	if b.scheduledWorkspaces == nil {
		b.scheduledWorkspaces = make(map[string]struct{})
	}
	b.scheduledWorkspaces[workspace] = struct{}{}
}

func (b *evolutionBridge) rememberScheduledColdPathWorkspaces(workspaces []string) {
	for _, workspace := range workspaces {
		b.rememberScheduledColdPathWorkspace(workspace)
	}
}

func (b *evolutionBridge) scheduledColdPathWorkspaces() []string {
	if b == nil {
		return nil
	}
	b.scheduledMu.Lock()
	defer b.scheduledMu.Unlock()
	out := make([]string, 0, len(b.scheduledWorkspaces))
	for workspace := range b.scheduledWorkspaces {
		out = append(out, workspace)
	}
	sort.Strings(out)
	return out
}

func registryWorkspaces(registry *AgentRegistry) []string {
	if registry == nil {
		return nil
	}
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	out := make([]string, 0, len(registry.agents))
	seen := make(map[string]struct{}, len(registry.agents))
	for _, agent := range registry.agents {
		if agent == nil {
			continue
		}
		workspace := strings.TrimSpace(agent.Workspace)
		if workspace == "" {
			continue
		}
		if _, ok := seen[workspace]; ok {
			continue
		}
		seen[workspace] = struct{}{}
		out = append(out, workspace)
	}
	sort.Strings(out)
	return out
}

type coldPathScheduleTime struct {
	hour   int
	minute int
}

func parseColdPathSchedule(values []string) []coldPathScheduleTime {
	out := make([]coldPathScheduleTime, 0, len(values))
	seen := make(map[coldPathScheduleTime]struct{}, len(values))
	for _, value := range values {
		parts := strings.Split(strings.TrimSpace(value), ":")
		if len(parts) != 2 {
			continue
		}
		hour, err := strconv.Atoi(parts[0])
		if err != nil || hour < 0 || hour > 23 {
			continue
		}
		minute, err := strconv.Atoi(parts[1])
		if err != nil || minute < 0 || minute > 59 {
			continue
		}
		item := coldPathScheduleTime{hour: hour, minute: minute}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].hour != out[j].hour {
			return out[i].hour < out[j].hour
		}
		return out[i].minute < out[j].minute
	})
	return out
}

func nextColdPathScheduledTime(now time.Time, schedule []coldPathScheduleTime) time.Time {
	for _, item := range schedule {
		candidate := time.Date(now.Year(), now.Month(), now.Day(), item.hour, item.minute, 0, 0, now.Location())
		if candidate.After(now) {
			return candidate
		}
	}
	first := schedule[0]
	tomorrow := now.AddDate(0, 0, 1)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), first.hour, first.minute, 0, 0, now.Location())
}

func toEvolutionSkillContextSnapshots(input []SkillContextSnapshot) []evolution.SkillContextSnapshot {
	if len(input) == 0 {
		return nil
	}

	out := make([]evolution.SkillContextSnapshot, 0, len(input))
	for _, snapshot := range input {
		out = append(out, evolution.SkillContextSnapshot{
			Sequence:   snapshot.Sequence,
			Trigger:    snapshot.Trigger,
			SkillNames: append([]string(nil), snapshot.SkillNames...),
		})
	}
	return out
}

func toEvolutionToolExecutions(input []ToolExecutionRecord) []evolution.ToolExecutionRecord {
	if len(input) == 0 {
		return nil
	}

	out := make([]evolution.ToolExecutionRecord, 0, len(input))
	for _, record := range input {
		out = append(out, evolution.ToolExecutionRecord{
			Name:         record.Name,
			Success:      record.Success,
			ErrorSummary: record.ErrorSummary,
			SkillNames:   append([]string(nil), record.SkillNames...),
		})
	}
	return out
}
