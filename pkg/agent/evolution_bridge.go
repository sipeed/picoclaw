package agent

import (
	"context"
	"sync"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/evolution"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

const evolutionObserverHookName = "evolution-observer"

type evolutionBridge struct {
	cfg            config.EvolutionConfig
	registry       *AgentRegistry
	runtime        *evolution.Runtime
	coldPathRunner *evolution.ColdPathRunner
	bgCtx          context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

func newEvolutionBridge(registry *AgentRegistry, cfg *config.Config, provider providers.LLMProvider) (*evolutionBridge, error) {
	if cfg == nil {
		return nil, nil
	}

	modelID := ""
	if provider != nil {
		modelID = provider.GetDefaultModel()
	}
	runtime, err := evolution.NewRuntime(evolution.RuntimeOptions{
		Config: cfg.Evolution,
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

	return bridge, nil
}

func (b *evolutionBridge) Close() error {
	if b == nil {
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

func (b *evolutionBridge) handleTurnEndAsync(meta EventMeta, payload TurnEndPayload) {
	if b == nil || b.runtime == nil {
		return
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

	ctx := b.bgCtx
	if ctx == nil {
		ctx = context.Background()
	}

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		if err := b.runtime.FinalizeTurn(ctx, input); err != nil {
			logger.WarnCF("agent", "Evolution finalize turn failed", map[string]any{
				"error":     err.Error(),
				"turn_id":   input.TurnID,
				"workspace": input.Workspace,
			})
			return
		}
		if b.coldPathRunner != nil {
			b.coldPathRunner.Trigger(input.Workspace)
		}
	}()
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
