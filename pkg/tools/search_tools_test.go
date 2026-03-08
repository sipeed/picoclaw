package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// Dummy tool to fill the registry in our tests.
type mockSearchableTool struct {
	name string
	desc string
}

func (m *mockSearchableTool) Name() string        { return m.name }
func (m *mockSearchableTool) Description() string { return m.desc }
func (m *mockSearchableTool) Parameters() map[string]any {
	return map[string]any{"type": "object"}
}

func (m *mockSearchableTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	return SilentResult("mock executed: " + m.name)
}

// Helper to initialize a populated ToolRegistry
func setupPopulatedRegistry() *ToolRegistry {
	reg := NewToolRegistry()

	// A core tool (NOT to be found by searches)
	reg.Register(&mockSearchableTool{
		name: "core_search",
		desc: "I am a visible core tool for searching files",
	})

	// Hidden tools (must be found by searches)
	reg.RegisterHidden(&mockSearchableTool{
		name: "mcp_read_file",
		desc: "Read the contents of a system file",
	})
	reg.RegisterHidden(&mockSearchableTool{
		name: "mcp_list_dir",
		desc: "List directories and files in the system",
	})
	reg.RegisterHidden(&mockSearchableTool{
		name: "mcp_fetch_net",
		desc: "Fetch data from a network database",
	})

	return reg
}

func TestRegexSearchTool_Execute(t *testing.T) {
	reg := setupPopulatedRegistry()
	tool := NewRegexSearchTool(reg, 5, 10)
	ctx := context.Background()

	t.Run("Empty Pattern Error", func(t *testing.T) {
		res := tool.Execute(ctx, map[string]any{})
		if !res.IsError || !strings.Contains(res.ForLLM, "Missing or invalid 'pattern'") {
			t.Errorf("Expected missing pattern error, got: %v", res.ForLLM)
		}
	})

	t.Run("Invalid Regex Syntax", func(t *testing.T) {
		res := tool.Execute(ctx, map[string]any{"pattern": "[unclosed"})
		if !res.IsError || !strings.Contains(res.ForLLM, "Invalid regex pattern syntax") {
			t.Errorf("Expected regex syntax error, got: %v", res.ForLLM)
		}
	})

	t.Run("No Match Found", func(t *testing.T) {
		res := tool.Execute(ctx, map[string]any{"pattern": "alien"})
		if res.IsError || !strings.Contains(res.ForLLM, "No tools found matching") {
			t.Errorf("Expected 'no tools found' message, got: %v", res.ForLLM)
		}
	})

	t.Run("Successful Match & Promotion", func(t *testing.T) {
		res := tool.Execute(ctx, map[string]any{"pattern": "system"})

		if res.IsError {
			t.Fatalf("Unexpected error: %v", res.ForLLM)
		}
		if !strings.Contains(res.ForLLM, "SUCCESS: These tools have been temporarily UNLOCKED") {
			t.Errorf("Expected success string, got: %v", res.ForLLM)
		}
		if !strings.Contains(res.ForLLM, "mcp_read_file") {
			t.Errorf("Expected 'mcp_read_file' in results")
		}

		// Verify that the TTL has been updated for the tools found
		reg.mu.RLock()
		defer reg.mu.RUnlock()
		if reg.tools["mcp_read_file"].TTL != 5 {
			t.Errorf("Expected TTL of 'mcp_read_file' to be promoted to 5, got %d", reg.tools["mcp_read_file"].TTL)
		}
		if reg.tools["mcp_fetch_net"].TTL != 0 {
			t.Errorf("Expected 'mcp_fetch_net' to NOT be promoted (TTL=0)")
		}
	})
}

func TestBM25SearchTool_Execute(t *testing.T) {
	reg := setupPopulatedRegistry()
	tool := NewBM25SearchTool(reg, 3, 10)
	ctx := context.Background()

	t.Run("Empty Query Error", func(t *testing.T) {
		res := tool.Execute(ctx, map[string]any{"query": "   "})
		if !res.IsError || !strings.Contains(res.ForLLM, "Missing or invalid 'query'") {
			t.Errorf("Expected missing query error, got: %v", res.ForLLM)
		}
	})

	t.Run("No Match Found", func(t *testing.T) {
		res := tool.Execute(ctx, map[string]any{"query": "aliens spaceships"})
		if res.IsError || !strings.Contains(res.ForLLM, "No tools found matching") {
			t.Errorf("Expected 'no tools found', got: %v", res.ForLLM)
		}
	})

	t.Run("Successful Match & Promotion", func(t *testing.T) {
		res := tool.Execute(ctx, map[string]any{"query": "read files"})

		if res.IsError {
			t.Fatalf("Unexpected error: %v", res.ForLLM)
		}
		if !strings.Contains(res.ForLLM, "mcp_read_file") {
			t.Errorf("Expected 'mcp_read_file' in BM25 results")
		}

		reg.mu.RLock()
		defer reg.mu.RUnlock()
		if reg.tools["mcp_read_file"].TTL != 3 {
			t.Errorf("Expected TTL of 'mcp_read_file' to be promoted to 3")
		}
	})
}

func TestCallDiscoveredTool_Execute(t *testing.T) {
	reg := setupPopulatedRegistry()
	tool := NewCallDiscoveredTool(reg, 8)
	ctx := context.Background()

	t.Run("Missing Name", func(t *testing.T) {
		res := tool.Execute(ctx, map[string]any{"arguments": map[string]any{}})
		if !res.IsError {
			t.Error("Expected error for missing tool_name")
		}
	})

	t.Run("Invalid Arguments Type Fallback", func(t *testing.T) {
		// If the LLM hallucinates and passes a string instead of an object/map,
		// does a graceful fallback to empty map.
		res := tool.Execute(ctx, map[string]any{
			"tool_name": "mcp_read_file",
			"arguments": "invalid-string-instead-of-object",
		})
		// It must be an error
		if !res.IsError {
			t.Fatalf("Expected an error for invalid argument type, but got success: %v", res.ForLLM)
		}
		// The error message should contain the explanation that we have added
		if !strings.Contains(res.ForLLM, "Invalid 'arguments' format") {
			t.Errorf("Expected instructional error message, got: %v", res.ForLLM)
		}
	})

	t.Run("Successful Passthrough", func(t *testing.T) {
		res := tool.Execute(ctx, map[string]any{
			"tool_name": "mcp_read_file",
			"arguments": map[string]any{"path": "/tmp/test.txt"},
		})

		if res.IsError {
			t.Fatalf("Unexpected error: %v", res.ForLLM)
		}
		if !strings.Contains(res.ForLLM, "mock executed: mcp_read_file") {
			t.Errorf("Expected underlying tool to be executed, got: %v", res.ForLLM)
		}

		// The tool should renew the TTL of the tool called
		reg.mu.RLock()
		defer reg.mu.RUnlock()
		if reg.tools["mcp_read_file"].TTL != 8 {
			t.Errorf("Expected TTL to be renewed to 8")
		}
	})
}

func TestToolRegistry_SearchLimitsAndCoreFiltering(t *testing.T) {
	reg := NewToolRegistry()

	// Add 1 Core and 10 Hidden, all containing the word "match"
	reg.Register(&mockSearchableTool{"core_match", "I am core with match"})
	for i := 0; i < 10; i++ {
		reg.RegisterHidden(&mockSearchableTool{
			name: fmt.Sprintf("hidden_match_%d", i),
			desc: "this has a match",
		})
	}

	t.Run("Regex limits and core filtering", func(t *testing.T) {
		// Search with Regex and a limit of maxSearchResults = 4
		res, err := reg.SearchRegex("match", 4)
		if err != nil {
			t.Fatalf("SearchRegex failed: %v", err)
		}

		if len(res) != 4 {
			t.Errorf("Expected exactly 4 results due to limit, got %d", len(res))
		}

		for _, r := range res {
			if r.Name == "core_match" {
				t.Errorf("SearchRegex returned a Core tool, which should be excluded")
			}
		}
	})

	t.Run("BM25 limits and core filtering", func(t *testing.T) {
		// Search with BM25 and a limit of maxSearchResults = 3
		res := reg.SearchBM25("match", 3)

		if len(res) != 3 {
			t.Errorf("Expected exactly 3 results due to limit, got %d", len(res))
		}

		for _, r := range res {
			if r.Name == "core_match" {
				t.Errorf("SearchBM25 returned a Core tool, which should be excluded")
			}
		}
	})
}
