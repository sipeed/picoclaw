package tools

import (
	"sort"
	"testing"
)

func setupTestRegistry(names ...string) *ToolRegistry {
	reg := NewToolRegistry()
	for _, name := range names {
		reg.Register(&dummyTool{name: name})
	}
	return reg
}

func registryNames(reg *ToolRegistry) []string {
	names := reg.List()
	sort.Strings(names)
	return names
}

func TestApplyPolicy_AllowOnly(t *testing.T) {
	reg := setupTestRegistry("read_file", "write_file", "exec", "web_search")
	ApplyPolicy(reg, ToolPolicy{Allow: []string{"read_file", "exec"}})

	names := registryNames(reg)
	if len(names) != 2 {
		t.Fatalf("count = %d, want 2: %v", len(names), names)
	}
	if names[0] != "exec" || names[1] != "read_file" {
		t.Errorf("names = %v, want [exec, read_file]", names)
	}
}

func TestApplyPolicy_DenyOnly(t *testing.T) {
	reg := setupTestRegistry("read_file", "write_file", "exec", "web_search")
	ApplyPolicy(reg, ToolPolicy{Deny: []string{"exec", "web_search"}})

	names := registryNames(reg)
	if len(names) != 2 {
		t.Fatalf("count = %d, want 2: %v", len(names), names)
	}
	if names[0] != "read_file" || names[1] != "write_file" {
		t.Errorf("names = %v, want [read_file, write_file]", names)
	}
}

func TestApplyPolicy_AllowAndDeny(t *testing.T) {
	reg := setupTestRegistry("read_file", "write_file", "exec", "web_search")
	ApplyPolicy(reg, ToolPolicy{
		Allow: []string{"read_file", "write_file", "exec"},
		Deny:  []string{"exec"},
	})

	names := registryNames(reg)
	if len(names) != 2 {
		t.Fatalf("count = %d, want 2: %v", len(names), names)
	}
	if names[0] != "read_file" || names[1] != "write_file" {
		t.Errorf("names = %v, want [read_file, write_file]", names)
	}
}

func TestApplyPolicy_EmptyPolicy(t *testing.T) {
	reg := setupTestRegistry("read_file", "write_file", "exec")
	ApplyPolicy(reg, ToolPolicy{})

	if reg.Count() != 3 {
		t.Errorf("count = %d, want 3 (no-op)", reg.Count())
	}
}

func TestApplyPolicy_GroupRefs(t *testing.T) {
	reg := setupTestRegistry("read_file", "write_file", "edit_file", "append_file", "list_dir", "web_search", "web_fetch", "exec")
	ApplyPolicy(reg, ToolPolicy{Deny: []string{"group:web"}})

	names := registryNames(reg)
	for _, name := range names {
		if name == "web_search" || name == "web_fetch" {
			t.Errorf("web tool %q should have been denied", name)
		}
	}
	if reg.Count() != 6 {
		t.Errorf("count = %d, want 6", reg.Count())
	}
}

func TestDepthDenyList_Zero(t *testing.T) {
	result := DepthDenyList(0, 3)
	if result != nil {
		t.Errorf("depth 0 should return nil, got %v", result)
	}
}

func TestDepthDenyList_AtMax(t *testing.T) {
	result := DepthDenyList(3, 3)
	expected := []string{"spawn", "handoff", "list_agents"}
	if len(result) != len(expected) {
		t.Fatalf("len = %d, want %d", len(result), len(expected))
	}
	for i, name := range expected {
		if result[i] != name {
			t.Errorf("result[%d] = %q, want %q", i, result[i], name)
		}
	}
}

func TestDepthDenyList_BelowMax(t *testing.T) {
	result := DepthDenyList(1, 3)
	if result != nil {
		t.Errorf("mid-chain should return nil, got %v", result)
	}
}

func TestRegistryClone(t *testing.T) {
	reg := setupTestRegistry("tool_a", "tool_b", "tool_c")
	cloned := reg.Clone()

	// Same tools
	if cloned.Count() != 3 {
		t.Fatalf("cloned count = %d, want 3", cloned.Count())
	}

	// Independent: removing from clone doesn't affect original
	cloned.Remove("tool_b")
	if cloned.Count() != 2 {
		t.Errorf("cloned count after remove = %d, want 2", cloned.Count())
	}
	if reg.Count() != 3 {
		t.Errorf("original count after clone remove = %d, want 3", reg.Count())
	}
}

func TestRegistryRemove(t *testing.T) {
	reg := setupTestRegistry("tool_a", "tool_b")
	reg.Remove("tool_a")

	if reg.Count() != 1 {
		t.Fatalf("count = %d, want 1", reg.Count())
	}
	if _, ok := reg.Get("tool_a"); ok {
		t.Error("tool_a should have been removed")
	}
	if _, ok := reg.Get("tool_b"); !ok {
		t.Error("tool_b should still exist")
	}

	// Remove nonexistent tool — no panic
	reg.Remove("nonexistent")
	if reg.Count() != 1 {
		t.Errorf("count = %d, want 1 after removing nonexistent", reg.Count())
	}
}

func TestPolicyPipeline_Compose(t *testing.T) {
	// Simulate: global allow → per-agent deny → depth deny
	reg := setupTestRegistry(
		"read_file", "write_file", "exec",
		"web_search", "spawn", "handoff", "list_agents",
	)

	// Layer 1: per-agent policy (deny web)
	ApplyPolicy(reg, ToolPolicy{Deny: []string{"web_search"}})

	// Layer 2: depth policy (leaf: deny spawn/handoff/list_agents)
	denyList := DepthDenyList(3, 3) // at max depth
	ApplyPolicy(reg, ToolPolicy{Deny: denyList})

	names := registryNames(reg)
	expected := map[string]bool{"read_file": true, "write_file": true, "exec": true}
	if len(names) != len(expected) {
		t.Fatalf("count = %d, want %d: %v", len(names), len(expected), names)
	}
	for _, name := range names {
		if !expected[name] {
			t.Errorf("unexpected tool: %q", name)
		}
	}
}
