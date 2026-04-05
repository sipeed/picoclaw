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
	return "Manage workspace skills (create, read, update, patch, delete, list). " +
		"Skills are your procedural memory -- reusable approaches for recurring task types.\n\n" +
		"Create when: complex task succeeded (5+ tool calls), errors overcome, " +
		"user-corrected approach worked, non-trivial workflow discovered.\n" +
		"Update when: instructions stale/wrong, missing steps or pitfalls found during use. " +
		"If you used a skill and hit issues not covered by it, patch it immediately.\n\n" +
		"After difficult/iterative tasks, offer to save as a skill. " +
		"Skip for simple one-offs. Confirm with user before creating/deleting.\n\n" +
		"Good skills: trigger conditions, numbered steps with exact commands, " +
		"pitfalls section, verification steps."
}

func (t *SkillManageTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"enum":        []string{"create", "read", "update", "patch", "delete", "list"},
				"description": "The operation to perform. Use 'patch' for targeted fixes (preferred over 'update' for small changes).",
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
			"old_string": map[string]any{
				"type":        "string",
				"description": "Text to find in SKILL.md (required for patch). Must appear exactly once.",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "Replacement text (required for patch). Use empty string to delete matched text.",
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
	oldStr, _ := args["old_string"].(string)
	newStr, _ := args["new_string"].(string)

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

	case "patch":
		if name == "" || oldStr == "" {
			return ErrorResult("patch requires 'name' and 'old_string'")
		}
		if err := t.mgr.PatchSkill(name, oldStr, newStr); err != nil {
			return ErrorResult(fmt.Sprintf("patch failed: %v", err))
		}
		return NewToolResult(fmt.Sprintf("Skill %q patched successfully", name))

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
		return ErrorResult(fmt.Sprintf("unknown operation %q -- use create/read/update/patch/delete/list", op))
	}
}
