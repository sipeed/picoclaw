package agent

import (
	"context"

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
		ApplierFactory: func(workspace string) *evolution.Applier {
			return evolution.NewApplier(evolution.NewPaths(workspace, cfg.Evolution.StateDir), nil)
		},
	})
	if err != nil {
		return nil, err
	}

	bridge := &evolutionBridge{
		cfg:      cfg.Evolution,
		registry: registry,
		runtime:  runtime,
	}
	if cfg.Evolution.AutoRunColdPath {
		bridge.coldPathRunner = evolution.NewColdPathRunnerWithErrorHandler(runtime, func(err error) {
			logger.WarnCF("agent", "Cold path run failed", map[string]any{
				"error": err.Error(),
			})
		})
	}

	return bridge, nil
}

func (b *evolutionBridge) Close() error {
	if b == nil || b.coldPathRunner == nil {
		return nil
	}
	return b.coldPathRunner.Close()
}

func (b *evolutionBridge) OnEvent(ctx context.Context, evt Event) error {
	if b == nil || !b.cfg.Enabled || b.runtime == nil {
		return nil
	}

	switch evt.Kind {
	case EventKindTurnEnd:
		payload, ok := evt.Payload.(TurnEndPayload)
		if !ok {
			return nil
		}
		if err := b.runtime.FinalizeTurn(ctx, evolution.TurnCaseInput{
			Workspace:             payload.Workspace,
			WorkspaceID:           payload.Workspace,
			TurnID:                evt.Meta.TurnID,
			SessionKey:            evt.Meta.SessionKey,
			AgentID:               evt.Meta.AgentID,
			Status:                string(payload.Status),
			ToolKinds:             append([]string(nil), payload.ToolKinds...),
			ActiveSkillNames:      append([]string(nil), payload.ActiveSkills...),
			AttemptedSkillNames:   append([]string(nil), payload.AttemptedSkills...),
			FinalSuccessfulPath:   append([]string(nil), payload.FinalSuccessfulPath...),
			SkillContextSnapshots: toEvolutionSkillContextSnapshots(payload.SkillContextSnapshots),
		}); err != nil {
			return err
		}
		if b.coldPathRunner != nil {
			b.coldPathRunner.Trigger(payload.Workspace)
		}
		return nil
	}

	return nil
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
