package tools

import (
	"context"
	"fmt"
	"sync"
)

type ToolRegistry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

func (tr *ToolRegistry) Register(tool Tool) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.tools[tool.Name()] = tool
}

func (tr *ToolRegistry) Get(name string) (Tool, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	tool, ok := tr.tools[name]
	return tool, ok
}

func (tr *ToolRegistry) Execute(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	tool, ok := tr.Get(name)
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}

	return tool.Execute(ctx, args)
}

func (tr *ToolRegistry) GetDefinitions() []map[string]interface{} {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	var definitions []map[string]interface{}
	for _, tool := range tr.tools {
		definitions = append(definitions, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name(),
				"description": tool.Description(),
				"parameters":  tool.Parameters(),
			},
		})
	}
	return definitions
}

// Clone creates a shallow copy of the registry
func (tr *ToolRegistry) Clone() *ToolRegistry {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	newReg := NewToolRegistry()
	for name, tool := range tr.tools {
		newReg.tools[name] = tool
	}
	return newReg
}