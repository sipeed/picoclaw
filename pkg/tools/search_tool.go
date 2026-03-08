package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/sipeed/picoclaw/pkg/utils"
)

type RegexSearchTool struct {
	registry         *ToolRegistry
	ttl              int
	maxSearchResults int
}

func NewRegexSearchTool(r *ToolRegistry, ttl int, maxSearchResults int) *RegexSearchTool {
	return &RegexSearchTool{registry: r, ttl: ttl, maxSearchResults: maxSearchResults}
}

func (t *RegexSearchTool) Name() string {
	return "tool_search_tool_regex"
}

func (t *RegexSearchTool) Description() string {
	return "Search available hidden tools on-demand using a regex pattern. Returns JSON schemas of discovered tools."
}

func (t *RegexSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Regex pattern to match tool name or description",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *RegexSearchTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	pattern, ok := args["pattern"].(string)
	if !ok || strings.TrimSpace(pattern) == "" {
		// An empty string regex (?i) will match every hidden tool,
		// dumping massive payloads into the context and burning tokens.
		return ErrorResult("Missing or invalid 'pattern' argument. Must be a non-empty string.")
	}

	res, err := t.registry.SearchRegex(pattern, t.maxSearchResults)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Invalid regex pattern syntax: %v. Please fix your regex and try again.", err))
	}

	return formatDiscoveryResponse(t.registry, res, t.ttl)
}

type BM25SearchTool struct {
	registry         *ToolRegistry
	ttl              int
	maxSearchResults int
}

func NewBM25SearchTool(r *ToolRegistry, ttl int, maxSearchResults int) *BM25SearchTool {
	return &BM25SearchTool{registry: r, ttl: ttl, maxSearchResults: maxSearchResults}
}

func (t *BM25SearchTool) Name() string {
	return "tool_search_tool_bm25"
}

func (t *BM25SearchTool) Description() string {
	return "Search available hidden tools on-demand using natural language query describing the action you need to perform. Returns JSON schemas of discovered tools."
}

func (t *BM25SearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query",
			},
		},
		"required": []string{"query"},
	}
}

func (t *BM25SearchTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	query, ok := args["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		// An empty string query will match every hidden tool,
		// dumping massive payloads into the context and burning tokens.
		return ErrorResult("Missing or invalid 'query' argument. Must be a non-empty string.")
	}

	return formatDiscoveryResponse(t.registry, t.registry.SearchBM25(query, t.maxSearchResults), t.ttl)
}

type CallDiscoveredTool struct {
	registry *ToolRegistry
	ttl      int
}

func NewCallDiscoveredTool(r *ToolRegistry, ttl int) *CallDiscoveredTool {
	return &CallDiscoveredTool{registry: r, ttl: ttl}
}

func (t *CallDiscoveredTool) Name() string {
	return "call_discovered_tool"
}

func (t *CallDiscoveredTool) Description() string {
	return "Fallback tool. Execute a tool found via search by passing its required arguments as a JSON object."
}

func (t *CallDiscoveredTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tool_name": map[string]any{
				"type": "string",
			},
			"arguments": map[string]any{
				"type":        "object",
				"description": "Arguments to pass to the tool",
			},
		},
		"required": []string{"tool_name"},
	}
}

func (t *CallDiscoveredTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	name, ok := args["tool_name"].(string)
	if !ok || name == "" {
		return ErrorResult("Missing or invalid 'tool_name' argument")
	}

	parsedArgs := make(map[string]any)

	// Check whether the key "arguments" exists in the payload
	if argVal, exists := args["arguments"]; exists && argVal != nil {
		// If it exists, we try to map cast it
		var valid bool
		parsedArgs, valid = argVal.(map[string]any)
		if !valid {
			// The LLM has passed something, but it is NOT a JSON object!
			// We have to tell him clearly to get him to correct.
			return ErrorResult(fmt.Sprintf(
				"Invalid 'arguments' format for tool '%s'. Expected a JSON object, but got %T. Please fix and try again.",
				name,
				argVal,
			))
		}
	}

	// Renew the TTL to keep it visible if it is actively used
	t.registry.PromoteTool(name, t.ttl)

	return t.registry.Execute(ctx, name, parsedArgs)
}

// ToolSearchResult represents the result returned to the LLM.
type ToolSearchResult struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

func (r *ToolRegistry) SearchRegex(pattern string, maxSearchResults int) ([]ToolSearchResult, error) {
	regex, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex pattern %q: %w", pattern, err)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []ToolSearchResult

	for name, entry := range r.tools {
		// Search only among the hidden tools (Core tools are already visible)
		if !entry.IsCore {
			// Directly call interface methods! No reflection/unmarshalling needed.
			desc := entry.Tool.Description()

			if regex.MatchString(name) || regex.MatchString(desc) {
				results = append(results, ToolSearchResult{
					Name:        name,
					Description: desc,
					Parameters:  entry.Tool.Parameters(),
				})
				if len(results) >= maxSearchResults {
					break // Stop searching once we hit the max! Saves CPU.
				}
			}
		}
	}

	return results, nil
}

func formatDiscoveryResponse(registry *ToolRegistry, results []ToolSearchResult, ttl int) *ToolResult {
	if len(results) == 0 {
		return SilentResult("No tools found matching the query.")
	}

	for _, r := range results {
		registry.PromoteTool(r.Name, ttl)
	}

	b, err := json.Marshal(results)
	if err != nil {
		return ErrorResult("Failed to format search results: " + err.Error())
	}

	msg := fmt.Sprintf(
		"Found %d tools:\n%s\n\nSUCCESS: These tools have been temporarily UNLOCKED as native tools! In your next response, you can call them directly just like any normal tool, without needing 'call_discovered_tool'.",
		len(results),
		string(b),
	)

	return SilentResult(msg)
}

// Lightweight internal type
type searchDoc struct {
	Name        string
	Description string
	Tool        Tool // Hold the interface reference
}

// SearchBM25 ranks hidden tools against query using BM25 via utils.BM25Engine.
// The corpus snapshot is built under the registry read-lock, then released
// before scoring so the lock is not held during CPU-intensive work.
func (r *ToolRegistry) SearchBM25(query string, maxSearchResults int) []ToolSearchResult {
	// We copy only the lightweight searchDoc values (name, description,
	// Tool interface reference). This keeps the lock window short and avoids
	// holding it during BM25 indexing and scoring.
	r.mu.RLock()
	snapshot := make([]searchDoc, 0, len(r.tools))
	for name, entry := range r.tools {
		if !entry.IsCore {
			snapshot = append(snapshot, searchDoc{
				Name:        name,
				Description: entry.Tool.Description(),
				Tool:        entry.Tool,
			})
		}
	}
	r.mu.RUnlock()

	if len(snapshot) == 0 {
		return nil
	}

	// Delegate scoring to the generic BM25 engine
	engine := utils.NewBM25Engine(
		snapshot,
		func(doc searchDoc) string {
			return doc.Name + " " + doc.Description
		},
	)

	ranked := engine.Search(query, maxSearchResults)
	if len(ranked) == 0 {
		return nil
	}

	out := make([]ToolSearchResult, len(ranked))
	for i, r := range ranked {
		out[i] = ToolSearchResult{
			Name:        r.Document.Name,
			Description: r.Document.Description,
			Parameters:  r.Document.Tool.Parameters(),
		}
	}
	return out
}
