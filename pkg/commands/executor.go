package commands

import (
	"context"
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

	passthroughCommand := ""

	for _, def := range e.reg.Definitions() {
		if !matchesCommand(def, cmdName) {
			continue
		}
		if passthroughCommand == "" {
			passthroughCommand = def.Name
		}
		if def.Handler == nil {
			continue
		}
		err := def.Handler(ctx, req)
		return ExecuteResult{Outcome: OutcomeHandled, Command: def.Name, Err: err}
	}
	if passthroughCommand != "" {
		return ExecuteResult{Outcome: OutcomePassthrough, Command: passthroughCommand}
	}

	return ExecuteResult{Outcome: OutcomePassthrough, Command: cmdName}
}

func matchesCommand(def Definition, cmdName string) bool {
	if normalizeCommandName(def.Name) == cmdName {
		return true
	}
	for _, alias := range def.Aliases {
		if normalizeCommandName(alias) == cmdName {
			return true
		}
	}
	return false
}
