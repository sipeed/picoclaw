package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validSkillContent = `---
name: test-skill
description: A test skill for unit testing
version: 1.0.0
---

# Test Skill

## When to Use
When testing the skill manager.

## Procedure
1. Step one
2. Step two
`

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "my-skill", false},
		{"valid multi-hyphen", "go-code-improve", false},
		{"valid single word", "skill123", false},
		{"valid uppercase", "MySkill", false},
		{"empty", "", true},
		{"spaces", "my skill", true},
		{"starts with dash", "-skill", true},
		{"dots", "go.code", true},
		{"underscore", "my_skill", true},
		{"too long", strings.Repeat("a", MaxNameLength+1), true},
		{"max length", strings.Repeat("a", MaxNameLength), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"valid", validSkillContent, false},
		{"no frontmatter", "# Just markdown", true},
		{"no closing", "---\nname: x\n# Body", true},
		{"missing name", "---\ndescription: x\n---\n", true},
		{"missing desc", "---\nname: x\n---\n", true},
		{"empty name", "---\nname: \"\"\ndescription: x\n---\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFrontmatter(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSize(t *testing.T) {
	if err := ValidateSize("small"); err != nil {
		t.Errorf("small content should pass: %v", err)
	}
	big := strings.Repeat("x", MaxContentSize+1)
	if err := ValidateSize(big); err == nil {
		t.Error("oversized content should fail")
	}
}

func TestCreateSkill(t *testing.T) {
	dir := t.TempDir()
	sm := NewSkillManager(dir)

	err := sm.CreateSkill("test-skill", validSkillContent, "")
	if err != nil {
		t.Fatalf("CreateSkill failed: %v", err)
	}

	// Verify file exists.
	path := filepath.Join(dir, "test-skill", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("SKILL.md not found: %v", err)
	}
	if string(data) != validSkillContent {
		t.Error("content mismatch")
	}
}

func TestCreateSkillWithCategory(t *testing.T) {
	dir := t.TempDir()
	sm := NewSkillManager(dir)

	err := sm.CreateSkill("my-tool", validSkillContent, "devops")
	if err != nil {
		t.Fatalf("CreateSkill with category failed: %v", err)
	}

	path := filepath.Join(dir, "devops", "my-tool", "SKILL.md")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("SKILL.md not created in category directory")
	}
}

func TestCreateSkillDuplicate(t *testing.T) {
	dir := t.TempDir()
	sm := NewSkillManager(dir)

	sm.CreateSkill("test-skill", validSkillContent, "")
	err := sm.CreateSkill("test-skill", validSkillContent, "")
	if err == nil {
		t.Error("expected error for duplicate skill")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestCreateSkillInvalidName(t *testing.T) {
	dir := t.TempDir()
	sm := NewSkillManager(dir)

	err := sm.CreateSkill("invalid.name", validSkillContent, "")
	if err == nil {
		t.Error("expected error for invalid name (dots not allowed)")
	}
}

func TestCreateSkillInvalidFrontmatter(t *testing.T) {
	dir := t.TempDir()
	sm := NewSkillManager(dir)

	err := sm.CreateSkill("test-skill", "# No frontmatter", "")
	if err == nil {
		t.Error("expected error for missing frontmatter")
	}
}

func TestPatchSkill(t *testing.T) {
	dir := t.TempDir()
	sm := NewSkillManager(dir)
	sm.CreateSkill("test-skill", validSkillContent, "")

	err := sm.PatchSkill("test-skill", "Step one", "Step one (updated)")
	if err != nil {
		t.Fatalf("PatchSkill failed: %v", err)
	}

	// Verify patch applied.
	path := filepath.Join(dir, "test-skill", "SKILL.md")
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "Step one (updated)") {
		t.Error("patch not applied")
	}
}

func TestPatchSkillNotFound(t *testing.T) {
	dir := t.TempDir()
	sm := NewSkillManager(dir)

	err := sm.PatchSkill("nonexistent", "old", "new")
	if err == nil {
		t.Error("expected error for nonexistent skill")
	}
}

func TestPatchSkillStringNotFound(t *testing.T) {
	dir := t.TempDir()
	sm := NewSkillManager(dir)
	sm.CreateSkill("test-skill", validSkillContent, "")

	err := sm.PatchSkill("test-skill", "NONEXISTENT STRING", "new")
	if err == nil {
		t.Error("expected error for string not found")
	}
}

func TestEditSkill(t *testing.T) {
	dir := t.TempDir()
	sm := NewSkillManager(dir)
	sm.CreateSkill("test-skill", validSkillContent, "")

	newContent := strings.Replace(validSkillContent, "A test skill", "Updated skill", 1)
	err := sm.EditSkill("test-skill", newContent)
	if err != nil {
		t.Fatalf("EditSkill failed: %v", err)
	}

	path := filepath.Join(dir, "test-skill", "SKILL.md")
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "Updated skill") {
		t.Error("edit not applied")
	}
}

func TestDeleteSkill(t *testing.T) {
	dir := t.TempDir()
	sm := NewSkillManager(dir)
	sm.CreateSkill("test-skill", validSkillContent, "")

	err := sm.DeleteSkill("test-skill")
	if err != nil {
		t.Fatalf("DeleteSkill failed: %v", err)
	}

	// Verify directory removed.
	skillDir := filepath.Join(dir, "test-skill")
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Error("skill directory should be removed")
	}
}

func TestFindSkill(t *testing.T) {
	dir := t.TempDir()
	sm := NewSkillManager(dir)
	sm.CreateSkill("test-skill", validSkillContent, "")

	info, found := sm.FindSkill("test-skill")
	if !found {
		t.Fatal("skill not found")
	}
	if info.Name != "test-skill" {
		t.Errorf("name = %q, want %q", info.Name, "test-skill")
	}
}

func TestFindSkillNotFound(t *testing.T) {
	dir := t.TempDir()
	sm := NewSkillManager(dir)

	_, found := sm.FindSkill("nonexistent")
	if found {
		t.Error("should not find nonexistent skill")
	}
}

func TestListSkills(t *testing.T) {
	dir := t.TempDir()
	sm := NewSkillManager(dir)
	sm.CreateSkill("skill-a", validSkillContent, "")

	contentB := strings.Replace(validSkillContent, "test-skill", "skill-b", 1)
	sm.CreateSkill("skill-b", contentB, "tools")

	skills := sm.ListSkills()
	if len(skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(skills))
	}
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	err := atomicWrite(path, "hello world")
	if err != nil {
		t.Fatalf("atomicWrite failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "hello world" {
		t.Errorf("content = %q, want %q", string(data), "hello world")
	}

	// Verify no temp file left.
	matches, _ := filepath.Glob(filepath.Join(dir, "*.tmp.*"))
	if len(matches) > 0 {
		t.Errorf("temp files left behind: %v", matches)
	}
}

func TestValidateCategory(t *testing.T) {
	if err := validateCategory("devops"); err != nil {
		t.Errorf("valid category failed: %v", err)
	}
	if err := validateCategory(".."); err == nil {
		t.Error("path traversal should fail")
	}
	if err := validateCategory("a/b"); err == nil {
		t.Error("nested category should fail")
	}
}
