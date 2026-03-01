package commands

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/session"
)

type fakeSessionOps struct{}

func (f *fakeSessionOps) ResolveActive(scopeKey string) (string, error) {
	return "", nil
}

func (f *fakeSessionOps) StartNew(scopeKey string) (string, error) {
	return "", nil
}

func (f *fakeSessionOps) List(scopeKey string) ([]session.SessionMeta, error) {
	return nil, nil
}

func (f *fakeSessionOps) Resume(scopeKey string, index int) (string, error) {
	return "", nil
}

func (f *fakeSessionOps) Prune(scopeKey string, limit int) ([]string, error) {
	return nil, nil
}

type fakeRuntime struct{}

func (f *fakeRuntime) Channel() string {
	return ""
}

func (f *fakeRuntime) ScopeKey() string {
	return ""
}

func (f *fakeRuntime) SessionOps() SessionOps {
	return &fakeSessionOps{}
}

func (f *fakeRuntime) Config() *config.Config {
	return &config.Config{}
}

func TestRuntimeContracts_MinimalSessionOps(t *testing.T) {
	var _ SessionOps = (*fakeSessionOps)(nil)
	var _ Runtime = (*fakeRuntime)(nil)
}
