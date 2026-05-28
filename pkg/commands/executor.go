package commands

import (
	"context"
	"fmt"
)

type Outcome int

const (
	// OutcomePassthrough means this input should continue through normal agent flow.
	OutcomePassthrough Outcome = iota
	// OutcomeHandled means a command handler executed (with or without handler error).
	OutcomeHandled
)

type ExecuteResult struct {
	Outcome Outcome
	Command string
	Err     error
}

type Executor struct {
	reg *Registry
	rt  *Runtime
}

func NewExecutor(reg *Registry, rt *Runtime) *Executor {
	return &Executor{reg: reg, rt: rt}
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
	// Ensure Reply is always non-nil so handlers don't need to check.
	if req.Reply == nil {
		req.Reply = func(string) error { return nil }
	}

	// Simple command — no sub-commands
	if len(def.SubCommands) == 0 {
		if def.Handler == nil {
			return ExecuteResult{Outcome: OutcomePassthrough, Command: def.Name}
		}
		err := def.Handler(ctx, req, e.rt)
		return ExecuteResult{Outcome: OutcomeHandled, Command: def.Name, Err: err}
	}

	// Sub-command routing
	subName := nthToken(req.Text, 1)
	if subName == "" {
		err := req.Reply("Usage: " + def.EffectiveUsage())
		return ExecuteResult{Outcome: OutcomeHandled, Command: def.Name, Err: err}
	}

	normalized := normalizeCommandName(subName)
	for _, sc := range def.SubCommands {
		if normalizeCommandName(sc.Name) == normalized {
			if sc.Handler == nil {
				return ExecuteResult{Outcome: OutcomePassthrough, Command: def.Name}
			}
			err := sc.Handler(ctx, req, e.rt)
			return ExecuteResult{Outcome: OutcomeHandled, Command: def.Name, Err: err}
		}
	}

	// Unknown sub-command
	err := req.Reply(fmt.Sprintf("Unknown option: %s. Usage: %s", subName, def.EffectiveUsage()))
	return ExecuteResult{Outcome: OutcomeHandled, Command: def.Name, Err: err}
}
