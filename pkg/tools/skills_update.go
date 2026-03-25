package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// UpdateSkillTool allows the LLM agent to analyze its own performance and
// write new skills or improvements into SKILL.md.
type UpdateSkillTool struct {
	workspace string
	mu        sync.Mutex
}

// NewUpdateSkillTool creates a new UpdateSkillTool.
func NewUpdateSkillTool(workspace string) *UpdateSkillTool {
	return &UpdateSkillTool{
		workspace: workspace,
		mu:        sync.Mutex{},
	}
}

func (t *UpdateSkillTool) Name() string {
	return "update_skill"
}

func (t *UpdateSkillTool) Description() string {
	return "Analyze a completed conversation to extract strengths, weaknesses, and new knowledge, then update the local SKILL.md file to improve future performance. Call this tool after solving a complex problem to learn from it."
}

func (t *UpdateSkillTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"analysis": map[string]any{
				"type":        "string",
				"description": "Analysis of the conversation, noting strengths and weaknesses.",
			},
			"skills_to_improve": map[string]any{
				"type":        "string",
				"description": "Specific skills, commands, or knowledge identified that need improvement to solve similar problems faster next time.",
			},
			"markdown_content": map[string]any{
				"type":        "string",
				"description": "The exact markdown content to append to SKILL.md containing the new learned skill or instruction.",
			},
		},
		"required": []string{"analysis", "skills_to_improve", "markdown_content"},
	}
}

func (t *UpdateSkillTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	t.mu.Lock()
	defer t.mu.Unlock()

	analysis, _ := args["analysis"].(string)
	skillsToImprove, _ := args["skills_to_improve"].(string)
	markdownContent, _ := args["markdown_content"].(string)

	if analysis == "" || skillsToImprove == "" || markdownContent == "" {
		return ErrorResult("analysis, skills_to_improve, and markdown_content are all required")
	}

	skillFilePath := filepath.Join(t.workspace, "SKILL.md")

	// Prepare the content to append
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	contentToAppend := fmt.Sprintf("\n\n## Learned Skill: %s\n\n**Analysis**: %s\n\n**Skills Improved**: %s\n\n%s\n",
		timestamp,
		analysis,
		skillsToImprove,
		markdownContent,
	)

	// Create directory if it doesn't exist (though workspace should exist)
	if err := os.MkdirAll(t.workspace, 0o755); err != nil {
		return ErrorResult(fmt.Sprintf("failed to create workspace directory: %v", err))
	}

	// Append to file
	f, err := os.OpenFile(skillFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to open SKILL.md for writing: %v", err))
	}
	defer f.Close()

	if _, err := f.WriteString(contentToAppend); err != nil {
		return ErrorResult(fmt.Sprintf("failed to append to SKILL.md: %v", err))
	}

	output := fmt.Sprintf("Successfully learned and updated SKILL.md.\n\nAnalysis: %s\nSkills Improved: %s\n", analysis, skillsToImprove)

	// The response is passed back to the LLM.
	// We also populate the ForUser field to notify the user.
	res := SilentResult(output)
	res.ForUser = fmt.Sprintf("I have analyzed our conversation and improved my skills.\n\n**My Analysis**:\n%s\n\n**Skills I've Improved/Added**:\n%s\n\nI have saved these learnings to `SKILL.md`.", analysis, skillsToImprove)
	return res
}
