package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sipeed/picoclaw/pkg/skills"
)

func TestFindSkillsToolName(t *testing.T) {
	tool := NewFindSkillsTool(skills.NewRegistryManager(), nil)
	assert.Equal(t, "find_skills", tool.Name())
}

func TestFindSkillsToolMissingQuery(t *testing.T) {
	tool := NewFindSkillsTool(skills.NewRegistryManager(), nil)
	result := tool.Execute(context.Background(), map[string]any{})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "query is required")
}

func TestFindSkillsToolEmptyQuery(t *testing.T) {
	tool := NewFindSkillsTool(skills.NewRegistryManager(), nil)
	result := tool.Execute(context.Background(), map[string]any{
		"query": "   ",
	})
	assert.True(t, result.IsError)
}

func TestFindSkillsToolCacheHit(t *testing.T) {
	cache := skills.NewSearchCache(10, 5*60*1000*1000*1000) // 5 min
	cache.Put("github", []skills.SearchResult{
		{Slug: "github", Score: 0.9, RegistryName: "clawhub"},
	})

	tool := NewFindSkillsTool(skills.NewRegistryManager(), cache)
	result := tool.Execute(context.Background(), map[string]any{
		"query": "github",
	})

	assert.False(t, result.IsError)
	assert.Contains(t, result.ForLLM, "github")
	assert.Contains(t, result.ForLLM, "cached")
}

func TestFindSkillsToolParameters(t *testing.T) {
	tool := NewFindSkillsTool(skills.NewRegistryManager(), nil)
	params := tool.Parameters()

	props, ok := params["properties"].(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, props, "query")
	assert.Contains(t, props, "limit")

	required, ok := params["required"].([]string)
	assert.True(t, ok)
	assert.Contains(t, required, "query")
}

func TestFindSkillsToolDescription(t *testing.T) {
	tool := NewFindSkillsTool(skills.NewRegistryManager(), nil)
	assert.NotEmpty(t, tool.Description())
	assert.Contains(t, tool.Description(), "skill")
}

func TestFormatSearchResultsEmpty(t *testing.T) {
	result := formatSearchResults("test query", nil, false)
	assert.Contains(t, result, "No skills found")
}

func TestFormatSearchResultsWithData(t *testing.T) {
	results := []skills.SearchResult{
		{
			Slug:         "github",
			Score:        0.95,
			DisplayName:  "GitHub",
			Summary:      "GitHub API integration",
			Version:      "1.0.0",
			RegistryName: "clawhub",
		},
	}
	output := formatSearchResults("github", results, false)
	assert.Contains(t, output, "github")
	assert.Contains(t, output, "v1.0.0")
	assert.Contains(t, output, "0.950")
	assert.Contains(t, output, "clawhub")
	assert.Contains(t, output, "install_skill")
}
