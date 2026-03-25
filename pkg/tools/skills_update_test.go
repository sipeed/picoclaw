package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateSkillTool_Name(t *testing.T) {
	tool := NewUpdateSkillTool(t.TempDir())
	assert.Equal(t, "update_skill", tool.Name())
	assert.NotEmpty(t, tool.Description())
	assert.NotNil(t, tool.Parameters())
}

func TestUpdateSkillTool_Execute(t *testing.T) {
	workspace := t.TempDir()
	tool := NewUpdateSkillTool(workspace)

	ctx := context.Background()

	// Missing args
	res := tool.Execute(ctx, map[string]any{})
	assert.True(t, res.IsError)
	assert.Contains(t, res.ForLLM, "required")

	// Success case
	args := map[string]any{
		"analysis":          "I was slow to find the file.",
		"skills_to_improve": "Use grep more effectively.",
		"markdown_content":  "Always use `grep -rn` when searching for strings.",
	}

	res = tool.Execute(ctx, args)
	assert.False(t, res.IsError)
	assert.Contains(t, res.ForLLM, "Successfully learned and updated SKILL.md")
	assert.Contains(t, res.ForUser, "I was slow to find the file.")

	// Check file contents
	skillFilePath := filepath.Join(workspace, "SKILL.md")
	content, err := os.ReadFile(skillFilePath)
	assert.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "Learned Skill")
	assert.Contains(t, contentStr, "I was slow to find the file.")
	assert.Contains(t, contentStr, "Use grep more effectively.")
	assert.Contains(t, contentStr, "Always use `grep -rn` when searching for strings.")

	// Test append
	args2 := map[string]any{
		"analysis":          "Another analysis.",
		"skills_to_improve": "Another skill.",
		"markdown_content":  "Another markdown.",
	}

	res2 := tool.Execute(ctx, args2)
	assert.False(t, res2.IsError)

	content2, err2 := os.ReadFile(skillFilePath)
	assert.NoError(t, err2)

	contentStr2 := string(content2)
	assert.Contains(t, contentStr2, "I was slow to find the file.") // Old content still there
	assert.Contains(t, contentStr2, "Another analysis.")            // New content added
}
