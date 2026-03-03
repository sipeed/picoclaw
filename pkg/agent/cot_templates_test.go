package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCotRegistry_BuiltinTemplates(t *testing.T) {
	dir := t.TempDir()
	r := NewCotRegistry(dir)

	// Should have all built-in templates.
	for _, bt := range builtinCotTemplates {
		tmpl := r.Get(bt.ID)
		if tmpl.ID != bt.ID {
			t.Errorf("expected template %q, got %q", bt.ID, tmpl.ID)
		}
	}

	// "direct" should have empty prompt.
	direct := r.Get("direct")
	if direct.Prompt != "" {
		t.Errorf("direct template should have empty prompt, got %q", direct.Prompt)
	}

	// "code" should have non-empty prompt.
	code := r.Get("code")
	if code.Prompt == "" {
		t.Error("code template should have non-empty prompt")
	}
	if !strings.Contains(code.Prompt, "Code Analysis") {
		t.Error("code template should mention 'Code Analysis'")
	}
}

func TestCotRegistry_UnknownFallsToDefault(t *testing.T) {
	dir := t.TempDir()
	r := NewCotRegistry(dir)

	tmpl := r.Get("nonexistent_template")
	if tmpl.ID != "direct" {
		t.Errorf("expected fallback to 'direct', got %q", tmpl.ID)
	}
}

func TestCotRegistry_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	r := NewCotRegistry(dir)

	tmpl := r.Get("  Code  ")
	if tmpl.ID != "code" {
		t.Errorf("expected 'code', got %q", tmpl.ID)
	}
}

func TestCotRegistry_UserTemplates(t *testing.T) {
	dir := t.TempDir()

	// Create user template.
	cotDir := filepath.Join(dir, "cot_templates")
	os.MkdirAll(cotDir, 0o755)

	content := `Custom strategy for data analysis
---
## Thinking Strategy: Data Analysis

1. Examine the data structure.
2. Identify patterns.
3. Draw conclusions.`

	os.WriteFile(filepath.Join(cotDir, "data_analysis.md"), []byte(content), 0o644)

	r := NewCotRegistry(dir)

	// Should be able to get the user template.
	tmpl := r.Get("data_analysis")
	if tmpl.ID != "data_analysis" {
		t.Errorf("expected 'data_analysis', got %q", tmpl.ID)
	}
	if tmpl.Description != "Custom strategy for data analysis" {
		t.Errorf("description = %q, want 'Custom strategy for data analysis'", tmpl.Description)
	}
	if !strings.Contains(tmpl.Prompt, "Examine the data structure") {
		t.Error("prompt should contain user-defined content")
	}
}

func TestCotRegistry_ListForPrompt(t *testing.T) {
	dir := t.TempDir()
	r := NewCotRegistry(dir)

	list := r.ListForPrompt()

	// Should contain all built-in template IDs.
	for _, bt := range builtinCotTemplates {
		if !strings.Contains(list, bt.ID) {
			t.Errorf("ListForPrompt missing template %q", bt.ID)
		}
	}
}

func TestCotRegistry_ListExamplesForPrompt(t *testing.T) {
	dir := t.TempDir()
	r := NewCotRegistry(dir)

	examples := r.ListExamplesForPrompt()

	// Should contain full example content for key templates.
	if !strings.Contains(examples, "Code Analysis") {
		t.Error("ListExamplesForPrompt missing 'Code Analysis' example")
	}
	if !strings.Contains(examples, "Analytical Reasoning") {
		t.Error("ListExamplesForPrompt missing 'Analytical Reasoning' example")
	}
	if !strings.Contains(examples, "Debugging") {
		t.Error("ListExamplesForPrompt missing 'Debugging' example")
	}
	// Should contain actual steps, not just names.
	if !strings.Contains(examples, "Requirements") {
		t.Error("ListExamplesForPrompt should include actual step content")
	}
}

func TestCotRegistry_UserOverridesBuiltin(t *testing.T) {
	dir := t.TempDir()
	// Create a user template that overrides "code".
	cotDir := filepath.Join(dir, "cot_templates")
	os.MkdirAll(cotDir, 0o755)

	content := `My custom code template
---
## Custom Code Strategy

Think differently about code.`

	os.WriteFile(filepath.Join(cotDir, "code.md"), []byte(content), 0o644)

	r := NewCotRegistry(dir)

	tmpl := r.Get("code")
	if !strings.Contains(tmpl.Prompt, "Think differently about code") {
		t.Error("user template should override built-in 'code' template")
	}
}
