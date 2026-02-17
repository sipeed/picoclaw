package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/skills"
)

func setupSkillFixture(t *testing.T) (string, *skills.SkillsLoader) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "skill-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create a test skill
	skillDir := filepath.Join(tmpDir, "skills", "test-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: test-skill
description: "A test skill for unit testing"
---

# Test Skill

This is the skill content.
`), 0644)

	loader := skills.NewSkillsLoader(tmpDir, "", "")
	return tmpDir, loader
}

func TestSkillTool_SkillList(t *testing.T) {
	tmpDir, loader := setupSkillFixture(t)
	defer os.RemoveAll(tmpDir)

	tool := NewSkillTool(loader)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "skill_list",
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if result.ForLLM == "No skills available" {
		t.Error("expected skills to be listed")
	}
}

func TestSkillTool_SkillList_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "skill-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	loader := skills.NewSkillsLoader(tmpDir, "", "")
	tool := NewSkillTool(loader)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "skill_list",
	})

	if result.IsError {
		t.Error("empty list should not be an error")
	}
	if result.ForLLM != "No skills available" {
		t.Errorf("expected 'No skills available', got '%s'", result.ForLLM)
	}
}

func TestSkillTool_SkillRead(t *testing.T) {
	tmpDir, loader := setupSkillFixture(t)
	defer os.RemoveAll(tmpDir)

	tool := NewSkillTool(loader)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "skill_read",
		"name":   "test-skill",
	})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if result.ForLLM == "" {
		t.Error("expected skill content")
	}
}

func TestSkillTool_SkillRead_NotFound(t *testing.T) {
	tmpDir, loader := setupSkillFixture(t)
	defer os.RemoveAll(tmpDir)

	tool := NewSkillTool(loader)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "skill_read",
		"name":   "nonexistent",
	})

	if !result.IsError {
		t.Error("expected error for nonexistent skill")
	}
}

func TestSkillTool_SkillRead_MissingName(t *testing.T) {
	tmpDir, loader := setupSkillFixture(t)
	defer os.RemoveAll(tmpDir)

	tool := NewSkillTool(loader)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "skill_read",
	})

	if !result.IsError {
		t.Error("expected error for missing name")
	}
}

func TestSkillTool_UnknownAction(t *testing.T) {
	tmpDir, loader := setupSkillFixture(t)
	defer os.RemoveAll(tmpDir)

	tool := NewSkillTool(loader)
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "skill_delete",
	})

	if !result.IsError {
		t.Error("expected error for unknown action")
	}
}

func TestSkillTool_MissingAction(t *testing.T) {
	tmpDir, loader := setupSkillFixture(t)
	defer os.RemoveAll(tmpDir)

	tool := NewSkillTool(loader)
	result := tool.Execute(context.Background(), map[string]interface{}{})

	if !result.IsError {
		t.Error("expected error for missing action")
	}
}

func TestSkillTool_NameAndDescription(t *testing.T) {
	tmpDir, loader := setupSkillFixture(t)
	defer os.RemoveAll(tmpDir)

	tool := NewSkillTool(loader)

	if tool.Name() != "skill" {
		t.Errorf("expected name 'skill', got '%s'", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
	if tool.Parameters() == nil {
		t.Error("parameters should not be nil")
	}
}
