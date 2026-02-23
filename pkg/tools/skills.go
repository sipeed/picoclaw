package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/KarakuriAgent/clawdroid/pkg/skills"
)

type SkillTool struct {
	loader *skills.SkillsLoader
}

func NewSkillTool(loader *skills.SkillsLoader) *SkillTool {
	return &SkillTool{loader: loader}
}

func (t *SkillTool) Name() string {
	return "skill"
}

func (t *SkillTool) Description() string {
	return "Read skill instructions. Actions: skill_list (list available skills), skill_read (read a skill's content)"
}

func (t *SkillTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "The skill action to perform",
				"enum":        []string{"skill_list", "skill_read"},
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Skill name (required for skill_read)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *SkillTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}

	switch action {
	case "skill_list":
		allSkills := t.loader.ListSkills()
		if len(allSkills) == 0 {
			return SilentResult("No skills available")
		}
		var sb strings.Builder
		for _, s := range allSkills {
			sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", s.Name, s.Source, s.Description))
		}
		return SilentResult(sb.String())

	case "skill_read":
		name, ok := args["name"].(string)
		if !ok || name == "" {
			return ErrorResult("name is required for skill_read")
		}
		content, found := t.loader.LoadSkill(name)
		if !found {
			return ErrorResult(fmt.Sprintf("skill %q not found", name))
		}
		return SilentResult(content)

	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}
