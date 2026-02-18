package tools

import (
	"sort"
	"testing"
)

func TestResolveToolNames_GroupExpansion(t *testing.T) {
	result := ResolveToolNames([]string{"group:fs"})
	expected := []string{"read_file", "write_file", "edit_file", "append_file", "list_dir"}

	if len(result) != len(expected) {
		t.Fatalf("len = %d, want %d: %v", len(result), len(expected), result)
	}
	sort.Strings(result)
	sort.Strings(expected)
	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("result[%d] = %q, want %q", i, result[i], expected[i])
		}
	}
}

func TestResolveToolNames_IndividualTool(t *testing.T) {
	result := ResolveToolNames([]string{"exec"})
	if len(result) != 1 || result[0] != "exec" {
		t.Errorf("result = %v, want [exec]", result)
	}
}

func TestResolveToolNames_Mixed(t *testing.T) {
	result := ResolveToolNames([]string{"group:web", "exec"})
	expected := map[string]bool{"web_search": true, "web_fetch": true, "exec": true}
	if len(result) != len(expected) {
		t.Fatalf("len = %d, want %d: %v", len(result), len(expected), result)
	}
	for _, name := range result {
		if !expected[name] {
			t.Errorf("unexpected tool: %q", name)
		}
	}
}

func TestResolveToolNames_Dedup(t *testing.T) {
	result := ResolveToolNames([]string{"group:exec", "exec"})
	if len(result) != 1 {
		t.Errorf("expected 1 (deduped), got %d: %v", len(result), result)
	}
}

func TestResolveToolNames_UnknownGroup(t *testing.T) {
	result := ResolveToolNames([]string{"group:nonexistent"})
	// Unknown group ref treated as a literal tool name
	if len(result) != 1 || result[0] != "group:nonexistent" {
		t.Errorf("result = %v, want [group:nonexistent]", result)
	}
}

func TestResolveToolNames_Empty(t *testing.T) {
	result := ResolveToolNames(nil)
	if len(result) != 0 {
		t.Errorf("result = %v, want empty", result)
	}
	result = ResolveToolNames([]string{})
	if len(result) != 0 {
		t.Errorf("result = %v, want empty", result)
	}
}
