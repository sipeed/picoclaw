package sandbox

import (
	"context"
	"errors"
	"fmt"
)

type unavailableSandbox struct {
	err error
	fs  FsBridge
}

func NewUnavailableSandbox(err error) Sandbox {
	if err == nil {
		err = errors.New("sandbox unavailable")
	}
	return &unavailableSandbox{
		err: err,
		fs:  &errorFS{err: err},
	}
}

func (u *unavailableSandbox) Start(ctx context.Context) error { return u.err }
func (u *unavailableSandbox) Prune(ctx context.Context) error { return nil }
func (u *unavailableSandbox) Fs() FsBridge                    { return u.fs }
func (u *unavailableSandbox) Exec(ctx context.Context, req ExecRequest) (*ExecResult, error) {
	return aggregateExecStream(func(onEvent func(ExecEvent) error) (*ExecResult, error) {
		return u.ExecStream(ctx, req, onEvent)
	})
}

func (u *unavailableSandbox) ExecStream(ctx context.Context, req ExecRequest, onEvent func(ExecEvent) error) (*ExecResult, error) {
	return nil, u.err
}

type errorFS struct {
	err error
}

func (e *errorFS) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return nil, fmt.Errorf("sandbox unavailable: %w", e.err)
}

func (e *errorFS) WriteFile(ctx context.Context, path string, data []byte, mkdir bool) error {
	return fmt.Errorf("sandbox unavailable: %w", e.err)
}
