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

func TestRegexSearchTool_PatternTooLong(t *testing.T) {
	reg := setupPopulatedRegistry()
	tool := NewRegexSearchTool(reg, 5, 10)
	ctx := context.Background()

	longPattern := strings.Repeat("a", MaxRegexPatternLength+1)
	res := tool.Execute(ctx, map[string]any{"pattern": longPattern})
	if !res.IsError || !strings.Contains(res.ForLLM, "Pattern too long") {
		t.Errorf("Expected pattern too long error, got: %v", res.ForLLM)
	}
}

func TestSearchRegex_ZeroMaxResults(t *testing.T) {
	reg := setupPopulatedRegistry()

	res, err := reg.SearchRegex("mcp", 0)
	if err != nil {
		t.Fatalf("SearchRegex failed: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Expected 0 results with maxSearchResults=0, got %d", len(res))
	}
}

func TestSearchBM25_ZeroMaxResults(t *testing.T) {
	reg := setupPopulatedRegistry()

	res := reg.SearchBM25("read file", 0)
	if len(res) != 0 {
		t.Errorf("Expected 0 results with maxSearchResults=0, got %d", len(res))
	}
}

func TestSearchRegex_DeterministicOrder(t *testing.T) {
	reg := NewToolRegistry()
	for i := 0; i < 20; i++ {
		reg.RegisterHidden(&mockSearchableTool{
			name: fmt.Sprintf("tool_%02d", i),
			desc: "searchable tool",
		})
	}

	// Run the same search multiple times and verify order is stable
	var firstRun []string
	for attempt := 0; attempt < 10; attempt++ {
		res, err := reg.SearchRegex("searchable", 20)
		if err != nil {
			t.Fatalf("SearchRegex failed: %v", err)
		}

		names := make([]string, len(res))
		for i, r := range res {
			names[i] = r.Name
		}

		if attempt == 0 {
			firstRun = names
		} else {
			for i, name := range names {
				if name != firstRun[i] {
					t.Fatalf("Non-deterministic order at attempt %d, index %d: got %q, want %q",
						attempt, i, name, firstRun[i])
				}
			}
		}
	}
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

func TestGet_HiddenToolTTLLifecycle(t *testing.T) {
	reg := NewToolRegistry()
	reg.RegisterHidden(&mockSearchableTool{name: "hidden_tool", desc: "test"})

	// TTL=0 at registration → not gettable
	_, ok := reg.Get("hidden_tool")
	if ok {
		t.Error("Expected hidden tool with TTL=0 to NOT be gettable")
	}

	// Promote → gettable
	reg.PromoteTools([]string{"hidden_tool"}, 3)
	_, ok = reg.Get("hidden_tool")
	if !ok {
		t.Error("Expected promoted hidden tool to be gettable")
	}

	// Tick down to 0 → not gettable again
	reg.TickTTL() // 3→2
	reg.TickTTL() // 2→1
	reg.TickTTL() // 1→0
	_, ok = reg.Get("hidden_tool")
	if ok {
		t.Error("Expected hidden tool with TTL ticked to 0 to NOT be gettable")
	}

	// Core tools remain always gettable
	reg.Register(&mockSearchableTool{name: "core_tool", desc: "core"})
	_, ok = reg.Get("core_tool")
	if !ok {
		t.Error("Expected core tool to always be gettable")
	}
}

func TestBM25CacheInvalidation(t *testing.T) {
	reg := NewToolRegistry()
	reg.RegisterHidden(&mockSearchableTool{name: "tool_alpha", desc: "alpha functionality"})

	tool := NewBM25SearchTool(reg, 5, 10)
	ctx := context.Background()

	// First search should find tool_alpha
	res := tool.Execute(ctx, map[string]any{"query": "alpha"})
	if !strings.Contains(res.ForLLM, "tool_alpha") {
		t.Fatalf("Expected 'tool_alpha' in first search, got: %v", res.ForLLM)
	}

	// Register a new hidden tool
	reg.RegisterHidden(&mockSearchableTool{name: "tool_beta", desc: "beta functionality"})

	// Cache should be invalidated; new tool should be findable
	res = tool.Execute(ctx, map[string]any{"query": "beta"})
	if !strings.Contains(res.ForLLM, "tool_beta") {
		t.Errorf("Expected 'tool_beta' after cache invalidation, got: %v", res.ForLLM)
	}
}

func TestPromoteTools_ConcurrentWithTickTTL(t *testing.T) {
	reg := NewToolRegistry()
	for i := 0; i < 20; i++ {
		reg.RegisterHidden(&mockSearchableTool{
			name: fmt.Sprintf("concurrent_tool_%d", i),
			desc: "concurrent test tool",
		})
	}

	names := make([]string, 20)
	for i := 0; i < 20; i++ {
		names[i] = fmt.Sprintf("concurrent_tool_%d", i)
	}

	// Hammer PromoteTools and TickTTL concurrently to detect races
	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			reg.PromoteTools(names, 5)
		}
		close(done)
	}()

	for i := 0; i < 1000; i++ {
		reg.TickTTL()
	}
	<-done
}
