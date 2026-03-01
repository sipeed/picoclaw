package plugin

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/hooks"
)

type testPlugin struct {
	name       string
	apiVersion string
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

func (p testPlugin) APIVersion() string {
	if p.apiVersion == "" {
		return APIVersion
	}
	return p.apiVersion
}

type descriptorTestPlugin struct {
	testPlugin
	info PluginInfo
}

func (p descriptorTestPlugin) Info() PluginInfo {
	return p.info
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

func TestRegisterRejectsPluginVersionMismatch(t *testing.T) {
	m := NewManager()
	p := testPlugin{
		name:       "old-plugin",
		apiVersion: "v0",
	}
	err := m.Register(p)
	if err == nil {
		t.Fatal("expected version mismatch error")
	}
}

func TestDescribeAll_UsesDescriptorWhenImplemented(t *testing.T) {
	m := NewManager()
	p := descriptorTestPlugin{
		testPlugin: testPlugin{name: "descriptor"},
		info: PluginInfo{
			Name:       " descriptor-visible ",
			APIVersion: " custom-v1 ",
			Status:     " active ",
		},
	}

	if err := m.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	got := m.DescribeAll()
	want := []PluginInfo{
		{
			Name:       "descriptor-visible",
			APIVersion: "custom-v1",
			Status:     "active",
		},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("DescribeAll() mismatch: got %v, want %v", got, want)
	}
}

func TestDescribeAll_FallsBackForPlainPlugin(t *testing.T) {
	m := NewManager()
	p := testPlugin{name: "plain"}

	if err := m.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	got := m.DescribeAll()
	want := []PluginInfo{
		{
			Name:       "plain",
			APIVersion: APIVersion,
			Status:     "enabled",
		},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("DescribeAll() mismatch: got %v, want %v", got, want)
	}
}

func TestDescribeEnabled_MatchesDescribeAllForNow(t *testing.T) {
	m := NewManager()
	plain := testPlugin{name: "plain"}
	described := descriptorTestPlugin{
		testPlugin: testPlugin{name: "described"},
		info: PluginInfo{
			Name: " described-visible ",
		},
	}

	if err := m.RegisterAll(plain, described); err != nil {
		t.Fatalf("RegisterAll() error = %v", err)
	}

	all := m.DescribeAll()
	enabled := m.DescribeEnabled()
	if !slices.Equal(enabled, all) {
		t.Fatalf("DescribeEnabled() mismatch: got %v, want %v", enabled, all)
	}

	wantAll := []PluginInfo{
		{
			Name:       "plain",
			APIVersion: APIVersion,
			Status:     "enabled",
		},
		{
			Name:       "described-visible",
			APIVersion: APIVersion,
			Status:     "enabled",
		},
	}
	if !slices.Equal(all, wantAll) {
		t.Fatalf("DescribeAll() order/content mismatch: got %v, want %v", all, wantAll)
	}
}

func TestResolveSelection_DefaultEnabled(t *testing.T) {
	result, err := ResolveSelection(
		[]string{"beta", "alpha", "gamma"},
		SelectionInput{
			DefaultEnabled: true,
			Disabled:       []string{"beta"},
		},
	)
	if err != nil {
		t.Fatalf("ResolveSelection() error = %v", err)
	}
	if !slices.Equal(result.EnabledNames, []string{"alpha", "gamma"}) {
		t.Fatalf("EnabledNames mismatch: got %v", result.EnabledNames)
	}
	if !slices.Equal(result.DisabledNames, []string{"beta"}) {
		t.Fatalf("DisabledNames mismatch: got %v", result.DisabledNames)
	}
}

func TestResolveSelection_EnabledListOnly(t *testing.T) {
	result, err := ResolveSelection(
		[]string{"a", "b", "c"},
		SelectionInput{
			DefaultEnabled: true,
			Enabled:        []string{"c", "a"},
		},
	)
	if err != nil {
		t.Fatalf("ResolveSelection() error = %v", err)
	}
	if !slices.Equal(result.EnabledNames, []string{"a", "c"}) {
		t.Fatalf("EnabledNames mismatch: got %v", result.EnabledNames)
	}
	if !slices.Equal(result.DisabledNames, []string{"b"}) {
		t.Fatalf("DisabledNames mismatch: got %v", result.DisabledNames)
	}
}

func TestResolveSelection_DisabledWinsOverlap(t *testing.T) {
	result, err := ResolveSelection(
		[]string{"a", "b", "c"},
		SelectionInput{
			Enabled:  []string{"a", "b"},
			Disabled: []string{"b"},
		},
	)
	if err != nil {
		t.Fatalf("ResolveSelection() error = %v", err)
	}
	if !slices.Equal(result.EnabledNames, []string{"a"}) {
		t.Fatalf("EnabledNames mismatch: got %v", result.EnabledNames)
	}
	if !slices.Equal(result.DisabledNames, []string{"b", "c"}) {
		t.Fatalf("DisabledNames mismatch: got %v", result.DisabledNames)
	}
}

func TestResolveSelection_UnknownEnabledFails(t *testing.T) {
	result, err := ResolveSelection(
		[]string{"a"},
		SelectionInput{
			Enabled: []string{"missing"},
		},
	)
	if err == nil {
		t.Fatal("expected error for unknown enabled plugin")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected error to mention unknown plugin, got %v", err)
	}
	if !slices.Equal(result.UnknownEnabled, []string{"missing"}) {
		t.Fatalf("UnknownEnabled mismatch: got %v", result.UnknownEnabled)
	}
}

func TestResolveSelection_UnknownDisabledWarns(t *testing.T) {
	result, err := ResolveSelection(
		[]string{"a"},
		SelectionInput{
			DefaultEnabled: true,
			Disabled:       []string{"missing"},
		},
	)
	if err != nil {
		t.Fatalf("ResolveSelection() error = %v", err)
	}
	if !slices.Equal(result.EnabledNames, []string{"a"}) {
		t.Fatalf("EnabledNames mismatch: got %v", result.EnabledNames)
	}
	if len(result.DisabledNames) != 0 {
		t.Fatalf("DisabledNames mismatch: got %v", result.DisabledNames)
	}
	if !slices.Equal(result.UnknownDisabled, []string{"missing"}) {
		t.Fatalf("UnknownDisabled mismatch: got %v", result.UnknownDisabled)
	}
	if !hasWarningSubstring(result.Warnings, `unknown disabled plugin "missing" ignored`) {
		t.Fatalf("expected unknown disabled warning, got %v", result.Warnings)
	}
}

func TestResolveSelection_NormalizationAndDedupe(t *testing.T) {
	result, err := ResolveSelection(
		[]string{" Alpha ", "beta", "gamma"},
		SelectionInput{
			Enabled:  []string{"ALPHA", " alpha ", "BETA", "beta"},
			Disabled: []string{" beta", "BETA", "missing", " MISSING "},
		},
	)
	if err != nil {
		t.Fatalf("ResolveSelection() error = %v", err)
	}
	if !slices.Equal(result.EnabledNames, []string{"alpha"}) {
		t.Fatalf("EnabledNames mismatch: got %v", result.EnabledNames)
	}
	if !slices.Equal(result.DisabledNames, []string{"beta", "gamma"}) {
		t.Fatalf("DisabledNames mismatch: got %v", result.DisabledNames)
	}
	if !slices.Equal(result.UnknownDisabled, []string{"missing"}) {
		t.Fatalf("UnknownDisabled mismatch: got %v", result.UnknownDisabled)
	}
	if !hasWarningSubstring(result.Warnings, `duplicate enabled plugin "alpha" ignored`) {
		t.Fatalf("expected duplicate enabled warning for alpha, got %v", result.Warnings)
	}
	if !hasWarningSubstring(result.Warnings, `duplicate enabled plugin "beta" ignored`) {
		t.Fatalf("expected duplicate enabled warning for beta, got %v", result.Warnings)
	}
	if !hasWarningSubstring(result.Warnings, `duplicate disabled plugin "beta" ignored`) {
		t.Fatalf("expected duplicate disabled warning for beta, got %v", result.Warnings)
	}
	if !hasWarningSubstring(result.Warnings, `duplicate disabled plugin "missing" ignored`) {
		t.Fatalf("expected duplicate disabled warning for missing, got %v", result.Warnings)
	}
	if !hasWarningSubstring(result.Warnings, `unknown disabled plugin "missing" ignored`) {
		t.Fatalf("expected unknown disabled warning, got %v", result.Warnings)
	}
}

func hasWarningSubstring(warnings []string, sub string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, sub) {
			return true
		}
	}
	return false
}
