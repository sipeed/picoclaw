package commands

import (
	"context"
	"fmt"
)

type Outcome int

const (
	OutcomePassthrough Outcome = iota
	OutcomeHandled
)

type ExecuteResult struct {
	Outcome Outcome
	Command string
	Reply   string
	Err     error
}

type Executor struct {
	reg *Registry
}

func NewExecutor(reg *Registry) *Executor {
	return &Executor{reg: reg}
}

// Execute implements a two-state command decision:
// 1) handled: execute command immediately;
// 2) passthrough: not a command or intentionally deferred to agent logic.
func (e *Executor) Execute(ctx context.Context, req Request) ExecuteResult {
	cmdName, ok := parseCommandName(req.Text)
	if !ok {
		return ExecuteResult{Outcome: OutcomePassthrough}
	}

	if e == nil || e.reg == nil {
		return ExecuteResult{Outcome: OutcomePassthrough, Command: cmdName}
	}

	def, found := e.reg.Lookup(cmdName)
	if !found {
		return ExecuteResult{Outcome: OutcomePassthrough, Command: cmdName}
	}

	return e.executeDefinition(ctx, req, def)
}

func (e *Executor) executeDefinition(ctx context.Context, req Request, def Definition) ExecuteResult {
	// Simple command — no sub-commands
	if len(def.SubCommands) == 0 {
		if def.Handler == nil {
			return ExecuteResult{Outcome: OutcomePassthrough, Command: def.Name}
		}
		err := def.Handler(ctx, req)
		return ExecuteResult{Outcome: OutcomeHandled, Command: def.Name, Err: err}
	}

	// Sub-command routing
	subName := secondToken(req.Text)
	if subName == "" {
		if req.Reply != nil {
			_ = req.Reply("Usage: " + def.EffectiveUsage())
		}
		return ExecuteResult{Outcome: OutcomeHandled, Command: def.Name}
	}

	normalized := normalizeCommandName(subName)
	for _, sc := range def.SubCommands {
		if normalizeCommandName(sc.Name) == normalized {
			if sc.Handler == nil {
				return ExecuteResult{Outcome: OutcomePassthrough, Command: def.Name}
			}
			err := sc.Handler(ctx, req)
			return ExecuteResult{Outcome: OutcomeHandled, Command: def.Name, Err: err}
		}
	}

	// Unknown sub-command
	if req.Reply != nil {
		_ = req.Reply(fmt.Sprintf("Unknown parameter: %s. Usage: %s", subName, def.EffectiveUsage()))
	}
	return ExecuteResult{Outcome: OutcomeHandled, Command: def.Name}
}
