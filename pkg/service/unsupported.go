package service

import (
	"context"
	"fmt"
	"io"
)

type unsupportedManager struct {
	detail string
}

func newUnsupportedManager(detail string) Manager {
	return &unsupportedManager{detail: detail}
}

func (m *unsupportedManager) Backend() string { return BackendUnsupported }

func (m *unsupportedManager) Install() error {
	return fmt.Errorf("service management unavailable: %s", m.detail)
}

func (m *unsupportedManager) Uninstall() error {
	return fmt.Errorf("service management unavailable: %s", m.detail)
}

func (m *unsupportedManager) Start() error {
	return fmt.Errorf("service management unavailable: %s", m.detail)
}

func (m *unsupportedManager) Stop() error {
	return fmt.Errorf("service management unavailable: %s", m.detail)
}

func (m *unsupportedManager) Restart() error {
	return fmt.Errorf("service management unavailable: %s", m.detail)
}

func (m *unsupportedManager) Status() (Status, error) {
	return Status{
		Backend: BackendUnsupported,
		Detail:  m.detail,
	}, nil
}

func (m *unsupportedManager) Logs(lines int) (string, error) {
	return "", fmt.Errorf("service logs unavailable: %s", m.detail)
}

func (m *unsupportedManager) LogsFollow(ctx context.Context, lines int, w io.Writer) error {
	return fmt.Errorf("service logs unavailable: %s", m.detail)
}
