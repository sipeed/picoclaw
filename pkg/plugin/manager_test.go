package plugin

import (
	"context"
	"errors"
	"testing"

	"github.com/sipeed/picoclaw/pkg/hooks"
)

type testPlugin struct {
	name       string
	registerFn func(*hooks.HookRegistry) error
}

func (p testPlugin) Name() string {
	return p.name
}

func (p testPlugin) Register(r *hooks.HookRegistry) error {
	if p.registerFn != nil {
		return p.registerFn(r)
	}
	return nil
}

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("expected manager")
	}
	if m.HookRegistry() == nil {
		t.Fatal("expected non-nil hook registry")
	}
	if len(m.Names()) != 0 {
		t.Fatalf("expected empty names, got %v", m.Names())
	}
}

func TestRegisterPluginAndTriggerHook(t *testing.T) {
	m := NewManager()
	called := false
	p := testPlugin{
		name: "audit",
		registerFn: func(r *hooks.HookRegistry) error {
			r.OnSessionStart("audit-session", 0, func(_ context.Context, _ *hooks.SessionEvent) error {
				called = true
				return nil
			})
			return nil
		},
	}

	if err := m.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if got := m.Names(); len(got) != 1 || got[0] != "audit" {
		t.Fatalf("unexpected names: %v", got)
	}

	m.HookRegistry().TriggerSessionStart(context.Background(), &hooks.SessionEvent{
		AgentID:    "a1",
		SessionKey: "s1",
	})
	if !called {
		t.Fatal("expected plugin hook to be called")
	}
}

func TestRegisterRejectsNilPlugin(t *testing.T) {
	m := NewManager()
	if err := m.Register(nil); err == nil {
		t.Fatal("expected error for nil plugin")
	}
}

func TestRegisterRejectsEmptyName(t *testing.T) {
	m := NewManager()
	if err := m.Register(testPlugin{}); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestRegisterRejectsDuplicateName(t *testing.T) {
	m := NewManager()
	p := testPlugin{name: "dup"}
	if err := m.Register(p); err != nil {
		t.Fatalf("unexpected first register error: %v", err)
	}
	if err := m.Register(p); err == nil {
		t.Fatal("expected duplicate name error")
	}
}

func TestRegisterPropagatesPluginError(t *testing.T) {
	m := NewManager()
	want := errors.New("register failed")
	p := testPlugin{
		name: "bad",
		registerFn: func(_ *hooks.HookRegistry) error {
			return want
		},
	}
	err := m.Register(p)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, want) {
		t.Fatalf("expected wrapped error %v, got %v", want, err)
	}
}

