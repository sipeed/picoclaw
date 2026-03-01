package commands

import (
	"context"
	"fmt"
)

type Outcome int

const (
	OutcomePassthrough Outcome = iota
	OutcomeHandled
	OutcomeRejected
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

func (e *Executor) Execute(ctx context.Context, req Request, _ any) ExecuteResult {
	cmdName, ok := parseCommandName(req.Text)
	if !ok {
		return ExecuteResult{Outcome: OutcomePassthrough}
	}

	if e == nil || e.reg == nil {
		return ExecuteResult{Outcome: OutcomePassthrough, Command: cmdName}
	}

	for _, def := range e.reg.ForChannel(req.Channel) {
		if !matchesCommand(def, cmdName) {
			continue
		}
		if def.Handler == nil {
			return ExecuteResult{Outcome: OutcomePassthrough, Command: def.Name}
		}
		err := def.Handler(ctx, req)
		return ExecuteResult{Outcome: OutcomeHandled, Command: def.Name, Err: err}
	}

	for _, def := range e.reg.defs {
		if !matchesCommand(def, cmdName) {
			continue
		}
		return ExecuteResult{
			Outcome: OutcomeRejected,
			Command: def.Name,
			Reply:   fmt.Sprintf("Command /%s is not supported on %s.", def.Name, req.Channel),
		}
	}

	return ExecuteResult{Outcome: OutcomePassthrough, Command: cmdName}
}

func matchesCommand(def Definition, cmdName string) bool {
	if def.Name == cmdName {
		return true
	}
	return contains(def.Aliases, cmdName)
}
