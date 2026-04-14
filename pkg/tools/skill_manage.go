package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/skills"
)

// SkillManageTool exposes the SkillManager to the LLM agent so it can
// create, read, update, and delete workspace skills dynamically.
type SkillManageTool struct {
	mgr *skills.SkillManager
}

// NewSkillManageTool creates a new SkillManageTool backed by the given manager.
func NewSkillManageTool(mgr *skills.SkillManager) *SkillManageTool {
	return &SkillManageTool{mgr: mgr}
}

func (t *SkillManageTool) Name() string { return "skill_manage" }

func (t *SkillManageTool) Description() string {
	return "Create, read, update, or delete workspace skills. Use this to persist reusable procedures the agent discovers during conversations."
}

func (t *SkillManageTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"enum":        []string{"create", "read", "update", "delete", "list"},
				"description": "The operation to perform",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Skill name (required for create/read/update/delete)",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "SKILL.md content with YAML frontmatter (required for create/update)",
			},
			"category": map[string]any{
				"type":        "string",
				"description": "Optional subdirectory category for create",
			},
		},
		"required": []string{"operation"},
	}
}

func (t *SkillManageTool) Execute(_ context.Context, args map[string]any) *ToolResult {
	op, _ := args["operation"].(string)
	name, _ := args["name"].(string)
	content, _ := args["content"].(string)
	category, _ := args["category"].(string)

	switch op {
	case "create":
		if name == "" || content == "" {
			return ErrorResult("create requires 'name' and 'content'")
		}
		if err := t.mgr.CreateSkill(name, content, category); err != nil {
			return ErrorResult(fmt.Sprintf("create failed: %v", err))
		}
		return NewToolResult(fmt.Sprintf("Skill %q created successfully", name))

	case "read":
		if name == "" {
			return ErrorResult("read requires 'name'")
		}
		info, ok := t.mgr.FindSkill(name)
		if !ok {
			return ErrorResult(fmt.Sprintf("skill %q not found", name))
		}
		return NewToolResult(fmt.Sprintf("Name: %s\nPath: %s\nDescription: %s", info.Name, info.Path, info.Description))

	case "update":
		if name == "" || content == "" {
			return ErrorResult("update requires 'name' and 'content'")
		}
		if err := t.mgr.EditSkill(name, content); err != nil {
			return ErrorResult(fmt.Sprintf("update failed: %v", err))
		}
		return NewToolResult(fmt.Sprintf("Skill %q updated successfully", name))

	case "delete":
		if name == "" {
			return ErrorResult("delete requires 'name'")
		}
		if err := t.mgr.DeleteSkill(name); err != nil {
			return ErrorResult(fmt.Sprintf("delete failed: %v", err))
		}
		return NewToolResult(fmt.Sprintf("Skill %q deleted", name))

	case "list":
		allSkills := t.mgr.ListSkills()
		if len(allSkills) == 0 {
			return NewToolResult("No skills found in workspace")
		}
		var sb strings.Builder
		for _, s := range allSkills {
			fmt.Fprintf(&sb, "- %s: %s\n", s.Name, s.Description)
		}
		return NewToolResult(sb.String())

	default:
		return ErrorResult(fmt.Sprintf("unknown operation %q — use create/read/update/delete/list", op))
	}
}
