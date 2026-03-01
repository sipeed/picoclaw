package commands

import (
	"context"
	"strings"
)

type Handler func(ctx context.Context, req Request) error

type Request struct {
	Channel   string
	ChatID    string
	SenderID  string
	Text      string
	MessageID string
	Reply     func(text string) error
}

type Result struct {
	Matched bool
	Handled bool
	Command string
	Err     error
}

type Dispatcher struct {
	reg *Registry
}

type Dispatching interface {
	Dispatch(ctx context.Context, req Request) Result
}

type DispatchFunc func(ctx context.Context, req Request) Result

func (f DispatchFunc) Dispatch(ctx context.Context, req Request) Result {
	return f(ctx, req)
}

func NewDispatcher(reg *Registry) *Dispatcher {
	return &Dispatcher{reg: reg}
}

func (d *Dispatcher) Dispatch(ctx context.Context, req Request) Result {
	cmdName, ok := parseCommandName(req.Text)
	if !ok {
		return Result{Matched: false}
	}

	for _, def := range d.reg.ForChannel(req.Channel) {
		if def.Name != cmdName && !contains(def.Aliases, cmdName) {
			continue
		}
		if def.Handler == nil {
			// Definition-only command (for menu registration / discovery).
			// Let the inbound message continue to the agent loop.
			return Result{Matched: false, Handled: false, Command: def.Name}
		}
		err := def.Handler(ctx, req)
		return Result{Matched: true, Handled: true, Command: def.Name, Err: err}
	}

	return Result{Matched: false}
}

func firstToken(input string) string {
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func parseCommandName(input string) (string, bool) {
	token := firstToken(input)
	if token == "" || !strings.HasPrefix(token, "/") {
		return "", false
	}

	name := strings.TrimPrefix(token, "/")
	if i := strings.Index(name, "@"); i >= 0 {
		name = name[:i]
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	return name, true
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
