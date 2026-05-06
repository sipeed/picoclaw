package memory

import (
	"testing"
	"strings"
)

func TestParseFrontmatter_ValidYAML(t *testing.T) {
	input := `---
session_id: "abc123"
timestamp: "2026-05-06T14:30:00Z"
tags: ["coding", "bug-fix"]
model: "claude-sonnet-4"
---
{"role": "user", "content": "test"}
`
	fm, body, err := ParseFrontmatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm["session_id"] != "abc123" {
		t.Errorf("expected session_id 'abc123', got %v", fm["session_id"])
	}
	if len(fm["tags"].([]interface{})) != 2 {
		t.Errorf("expected 2 tags, got %v", fm["tags"])
	}
	if !strings.Contains(body, `"role": "user"`) {
		t.Errorf("expected body to contain JSON, got %s", body)
	}
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	input := `{"role": "user", "content": "test"}`
	_, body, err := ParseFrontmatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(body, `"role": "user"`) {
		t.Errorf("expected body unchanged, got %s", body)
	}
}

func TestParseFrontmatter_InvalidYAML(t *testing.T) {
	input := `---
invalid: [unclosed
---
{"role": "user"}
`
	_, _, err := ParseFrontmatter(input)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
