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
}

type Result struct {
	Matched bool
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
	token := firstToken(req.Text)
	if token == "" {
		return Result{Matched: false}
	}

	cmdName := strings.TrimPrefix(token, "/")
	for _, def := range d.reg.ForChannel(req.Channel) {
		if def.Name != cmdName {
			continue
		}
		if def.Handler == nil {
			return Result{Matched: true, Command: def.Name}
		}
		err := def.Handler(ctx, req)
		return Result{Matched: true, Command: def.Name, Err: err}
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
