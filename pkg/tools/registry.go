package tools

import (
	"context"
	"sort"
	"strings"
	"sync"
)

// NormalizeToolName keeps only lowercase ASCII letters.

// "read_file" → "readfile", "ReadFile" → "readfile", "read-file" → "readfile".

func NormalizeToolName(s string) string {
	var b strings.Builder

	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + 32)
		} else if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
		}
	}

	return b.String()
}

type ToolRegistry struct {
	tools map[string]Tool

	mu sync.RWMutex
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

func (r *ToolRegistry) Register(tool Tool) {
	r.mu.Lock()

	defer r.mu.Unlock()

	r.tools[tool.Name()] = tool
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()

	defer r.mu.RUnlock()

	// Exact match first

	if tool, ok := r.tools[name]; ok {
		return tool, true
	}

	// Fuzzy fallback: normalize and compare (handles "readfile" → "read_file" etc.)

	norm := NormalizeToolName(name)

	for _, tool := range r.tools {
		if NormalizeToolName(tool.Name()) == norm {
			return tool, true
		}
	}

	return nil, false
}

func (r *ToolRegistry) Execute(ctx context.Context, name string, args map[string]any) *ToolResult {
	return r.ExecuteWithContext(ctx, name, args, "", "", nil)
}

// sortedToolNames returns tool names in sorted order for deterministic iteration.

// This is critical for KV cache stability: non-deterministic map iteration would

// produce different system prompts and tool definitions on each call, invalidating

// the LLM's prefix cache even when no tools have changed.

func (r *ToolRegistry) sortedToolNames() []string {
	names := make([]string, 0, len(r.tools))

	for name := range r.tools {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

func (r *ToolRegistry) GetDefinitions() []map[string]any {
	r.mu.RLock()

	defer r.mu.RUnlock()

	sorted := r.sortedToolNames()

	definitions := make([]map[string]any, 0, len(sorted))

	for _, name := range sorted {
		definitions = append(definitions, ToolToSchema(r.tools[name]))
	}

	return definitions
}

// List returns a list of all registered tool names.

func (r *ToolRegistry) List() []string {
	r.mu.RLock()

	defer r.mu.RUnlock()

	return r.sortedToolNames()
}

// Count returns the number of registered tools.

func (r *ToolRegistry) Count() int {
	r.mu.RLock()

	defer r.mu.RUnlock()

	return len(r.tools)
}
