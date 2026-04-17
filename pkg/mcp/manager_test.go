package mcp

import (
	"context"
	"testing"
	"time"
)

func TestCallToolWithNilArguments(t *testing.T) {
	m := NewManager()
	m.wg.Add(1)

	// Simula chiamata a tool senza argomenti (arguments=nil)
	err := m.CallTool(context.Background(), "test_server", "test_tool", nil)

	if err != nil {
		t.Fatalf("CallTool con arguments=nil restituisce errore: %v", err)
	}

	// Verifica che il manager sia ancora funzionante
	m.wg.Wait()
}

func TestCallToolWithEmptyMap(t *testing.T) {
	m := NewManager()
	m.wg.Add(1)

	// Chiamata con empty map (comportamento corretto)
	err := m.CallTool(context.Background(), "test_server", "test_tool", map[string]any{})

	if err != nil {
		t.Fatalf("CallTool con empty map restituisce errore: %v", err)
	}

	m.wg.Wait()
}

func TestCallToolWithValidArguments(t *testing.T) {
	m := NewManager()
	m.wg.Add(1)

	// Chiamata con argomenti validi
	arguments := map[string]any{"key": "value"}
	err := m.CallTool(context.Background(), "test_server", "test_tool", arguments)

	if err != nil {
		t.Fatalf("CallTool con arguments validi restituisce errore: %v", err)
	}

	m.wg.Wait()
}
