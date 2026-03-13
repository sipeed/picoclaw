package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/skills"
)

// FindSkillsTool allows the LLM agent to search for installable skills from registries.
type FindSkillsTool struct {
	registryMgr *skills.RegistryManager
	cache       *skills.SearchCache
}

// NewFindSkillsTool creates a new FindSkillsTool.
// registryMgr is the shared registry manager (built from config in createToolRegistry).
// cache is the search cache for deduplicating similar queries.
func NewFindSkillsTool(registryMgr *skills.RegistryManager, cache *skills.SearchCache) *FindSkillsTool {
	return &FindSkillsTool{
		registryMgr: registryMgr,
		cache:       cache,
	}
}

func (t *FindSkillsTool) Name() string {
	return "find_skills"
}

func (t *FindSkillsTool) Description() string {
	return "Search for installable skills from skill registries. Returns skill slugs, descriptions, versions, and relevance scores. Use this to discover skills before installing them with install_skill."
}

func (t *FindSkillsTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query describing the desired skill capability (e.g., 'github integration', 'database management')",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return (1-20, default 5)",
				"minimum":     1.0,
				"maximum":     20.0,
			},
		},
		"required": []string{"query"},
	}
}

func (t *FindSkillsTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	query, ok := args["query"].(string)
	query = strings.ToLower(strings.TrimSpace(query))
	if !ok || query == "" {
		return ErrorResult("query is required and must be a non-empty string")
	}

	limit := 5
	if l, ok := args["limit"].(float64); ok {
		li := int(l)
		if li >= 1 && li <= 20 {
			limit = li
		}
	}

	// Check cache first.
	if t.cache != nil {
		if cached, hit := t.cache.Get(query); hit {
			return SilentResult(formatSearchResults(query, cached, true))
		}
	}

	// Search all registries.
	results, err := t.registryMgr.SearchAll(ctx, query, limit)
	if err != nil {
		return ErrorResult(fmt.Sprintf("skill search failed: %v", err))
	}

	// Cache the results.
	if t.cache != nil && len(results) > 0 {
		t.cache.Put(query, results)
	}

	return SilentResult(formatSearchResults(query, results, false))
}

func formatSearchResults(query string, results []skills.SearchResult, cached bool) string {
	if len(results) == 0 {
		return fmt.Sprintf("No skills found for query: %q", query)
	}

	var sb strings.Builder
	source := ""
	if cached {
		source = " (cached)"
	}
	sb.WriteString(fmt.Sprintf("Found %d skills for %q%s:\n\n", len(results), query, source))

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. **%s**", i+1, r.Slug))
		if r.Version != "" {
			sb.WriteString(fmt.Sprintf(" v%s", r.Version))
		}
		sb.WriteString(fmt.Sprintf("  (score: %.3f, registry: %s)\n", r.Score, r.RegistryName))
		if r.DisplayName != "" && r.DisplayName != r.Slug {
			sb.WriteString(fmt.Sprintf("   Name: %s\n", r.DisplayName))
		}
		if r.Summary != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", r.Summary))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Use install_skill with the slug to install a skill.")
	return sb.String()
}
