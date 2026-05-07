package agent

import (
	"reflect"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newTestLoopWithRoot(t *testing.T, root string) *AgentLoop {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.WorkspaceRoot = root
	return &AgentLoop{cfg: cfg}
}

func TestExtractTenantOverrides_NoRaw(t *testing.T) {
	al := newTestLoopWithRoot(t, t.TempDir())
	got, err := al.extractTenantOverrides(bus.InboundMessage{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.workspace != "" || got.configDir != "" || len(got.allowedTools) > 0 || len(got.allowedSkills) > 0 {
		t.Fatalf("expected zero overrides, got %+v", got)
	}
}

func TestExtractTenantOverrides_AllFields(t *testing.T) {
	root := t.TempDir()
	al := newTestLoopWithRoot(t, root)

	msg := bus.InboundMessage{
		Context: bus.InboundContext{
			Raw: map[string]string{
				"workspace_override": "tenant-a/workspace",
				"config_dir":         "tenant-a/config",
				"allowed_tools":      "read_file, write_file ,exec",
				"allowed_skills":     "search,plan",
			},
		},
	}
	got, err := al.extractTenantOverrides(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.workspace == "" {
		t.Fatalf("expected workspace to be resolved, got empty")
	}
	if got.configDir == "" {
		t.Fatalf("expected configDir to be resolved, got empty")
	}
	wantTools := []string{"read_file", "write_file", "exec"}
	if !reflect.DeepEqual(got.allowedTools, wantTools) {
		t.Fatalf("allowedTools = %v, want %v", got.allowedTools, wantTools)
	}
	wantSkills := []string{"search", "plan"}
	if !reflect.DeepEqual(got.allowedSkills, wantSkills) {
		t.Fatalf("allowedSkills = %v, want %v", got.allowedSkills, wantSkills)
	}
}

func TestExtractTenantOverrides_EscapeRejected(t *testing.T) {
	al := newTestLoopWithRoot(t, t.TempDir())
	msg := bus.InboundMessage{
		Context: bus.InboundContext{
			Raw: map[string]string{
				"workspace_override": "../../../etc",
			},
		},
	}
	if _, err := al.extractTenantOverrides(msg); err == nil {
		t.Fatalf("expected error for path escape, got nil")
	}
}

func TestExtractTenantOverrides_NoRootBoundary(t *testing.T) {
	// workspace_root unset → any override must be rejected.
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.WorkspaceRoot = ""
	al := &AgentLoop{cfg: cfg}

	msg := bus.InboundMessage{
		Context: bus.InboundContext{
			Raw: map[string]string{"workspace_override": "anything"},
		},
	}
	if _, err := al.extractTenantOverrides(msg); err == nil {
		t.Fatalf("expected error when workspace_root is unset, got nil")
	}
}

func TestSplitCSV(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{",,", nil},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{"a,,b", []string{"a", "b"}},
	}
	for _, c := range cases {
		got := splitCSV(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("splitCSV(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestApplyTo(t *testing.T) {
	o := tenantOverrides{
		workspace:     "/ws",
		configDir:     "/cfg",
		allowedTools:  []string{"x"},
		allowedSkills: []string{"y"},
	}
	var opts processOptions
	o.applyTo(&opts)
	if opts.WorkspaceOverride != "/ws" {
		t.Errorf("WorkspaceOverride = %q", opts.WorkspaceOverride)
	}
	if opts.ConfigDir != "/cfg" {
		t.Errorf("ConfigDir = %q", opts.ConfigDir)
	}
	if !reflect.DeepEqual(opts.AllowedTools, []string{"x"}) {
		t.Errorf("AllowedTools = %v", opts.AllowedTools)
	}
	if !reflect.DeepEqual(opts.AllowedSkills, []string{"y"}) {
		t.Errorf("AllowedSkills = %v", opts.AllowedSkills)
	}
}
