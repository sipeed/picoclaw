package multiagent

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
)

func TestBlackboard_SetGet(t *testing.T) {
	bb := NewBlackboard()
	bb.Set("goal", "build feature X", "main")

	if got := bb.Get("goal"); got != "build feature X" {
		t.Errorf("Get(goal) = %q, want %q", got, "build feature X")
	}
}

func TestBlackboard_GetMissing(t *testing.T) {
	bb := NewBlackboard()
	if got := bb.Get("missing"); got != "" {
		t.Errorf("Get(missing) = %q, want empty", got)
	}
}

func TestBlackboard_GetEntry(t *testing.T) {
	bb := NewBlackboard()
	bb.Set("status", "in-progress", "coder")

	entry := bb.GetEntry("status")
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Author != "coder" {
		t.Errorf("Author = %q, want %q", entry.Author, "coder")
	}
	if entry.Scope != "shared" {
		t.Errorf("Scope = %q, want %q", entry.Scope, "shared")
	}
}

func TestBlackboard_GetEntryMissing(t *testing.T) {
	bb := NewBlackboard()
	if entry := bb.GetEntry("nope"); entry != nil {
		t.Error("expected nil entry for missing key")
	}
}

func TestBlackboard_Overwrite(t *testing.T) {
	bb := NewBlackboard()
	bb.Set("counter", "1", "a")
	bb.Set("counter", "2", "b")

	entry := bb.GetEntry("counter")
	if entry.Value != "2" {
		t.Errorf("Value = %q after overwrite, want %q", entry.Value, "2")
	}
	if entry.Author != "b" {
		t.Errorf("Author = %q after overwrite, want %q", entry.Author, "b")
	}
}

func TestBlackboard_Delete(t *testing.T) {
	bb := NewBlackboard()
	bb.Set("tmp", "value", "main")

	if !bb.Delete("tmp") {
		t.Error("Delete(tmp) returned false, expected true")
	}
	if bb.Delete("tmp") {
		t.Error("Delete(tmp) second call returned true, expected false")
	}
	if bb.Get("tmp") != "" {
		t.Error("Get(tmp) after delete should return empty")
	}
}

func TestBlackboard_List(t *testing.T) {
	bb := NewBlackboard()
	bb.Set("b", "2", "a")
	bb.Set("a", "1", "a")
	bb.Set("c", "3", "a")

	keys := bb.List()
	if len(keys) != 3 {
		t.Fatalf("List() returned %d keys, want 3", len(keys))
	}
	if keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Errorf("List() = %v, want [a b c]", keys)
	}
}

func TestBlackboard_Snapshot(t *testing.T) {
	bb := NewBlackboard()
	if s := bb.Snapshot(); s != "" {
		t.Errorf("empty blackboard Snapshot() = %q, want empty", s)
	}

	bb.Set("goal", "test", "main")
	s := bb.Snapshot()
	if s == "" {
		t.Error("Snapshot() returned empty for non-empty blackboard")
	}
	if !contains(s, "goal") || !contains(s, "main") || !contains(s, "test") {
		t.Errorf("Snapshot() = %q, expected to contain key/author/value", s)
	}
}

func TestBlackboard_Size(t *testing.T) {
	bb := NewBlackboard()
	if bb.Size() != 0 {
		t.Errorf("Size() = %d, want 0", bb.Size())
	}
	bb.Set("a", "1", "x")
	bb.Set("b", "2", "x")
	if bb.Size() != 2 {
		t.Errorf("Size() = %d, want 2", bb.Size())
	}
}

func TestBlackboard_ConcurrentAccess(t *testing.T) {
	bb := NewBlackboard()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key"
			bb.Set(key, "val", "agent")
			bb.Get(key)
			bb.List()
			bb.Snapshot()
		}(i)
	}
	wg.Wait()
}

func TestBlackboard_JSON(t *testing.T) {
	bb := NewBlackboard()
	bb.Set("x", "1", "a")
	bb.Set("y", "2", "b")

	data, err := json.Marshal(bb)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	bb2 := NewBlackboard()
	if err := json.Unmarshal(data, bb2); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if bb2.Get("x") != "1" || bb2.Get("y") != "2" {
		t.Error("roundtrip lost data")
	}
}

func TestBlackboardTool_Write(t *testing.T) {
	bb := NewBlackboard()
	tool := NewBlackboardTool(bb, "test-agent")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "write",
		"key":    "task",
		"value":  "implement feature",
	})
	if result.IsError {
		t.Fatalf("write failed: %s", result.ForLLM)
	}

	if bb.Get("task") != "implement feature" {
		t.Error("write did not persist")
	}
	entry := bb.GetEntry("task")
	if entry.Author != "test-agent" {
		t.Errorf("Author = %q, want %q", entry.Author, "test-agent")
	}
}

func TestBlackboardTool_Read(t *testing.T) {
	bb := NewBlackboard()
	bb.Set("info", "hello", "other")
	tool := NewBlackboardTool(bb, "reader")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "read",
		"key":    "info",
	})
	if result.IsError {
		t.Fatalf("read failed: %s", result.ForLLM)
	}
	if !contains(result.ForLLM, "hello") {
		t.Errorf("read result = %q, expected to contain 'hello'", result.ForLLM)
	}
}

func TestBlackboardTool_ReadMissing(t *testing.T) {
	bb := NewBlackboard()
	tool := NewBlackboardTool(bb, "reader")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "read",
		"key":    "nope",
	})
	if result.IsError {
		t.Fatalf("read missing should not be error: %s", result.ForLLM)
	}
	if !contains(result.ForLLM, "No entry") {
		t.Errorf("expected 'No entry' message, got %q", result.ForLLM)
	}
}

func TestBlackboardTool_List(t *testing.T) {
	bb := NewBlackboard()
	bb.Set("a", "1", "x")
	bb.Set("b", "2", "y")
	tool := NewBlackboardTool(bb, "lister")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "list",
	})
	if result.IsError {
		t.Fatalf("list failed: %s", result.ForLLM)
	}
	if !contains(result.ForLLM, "a") || !contains(result.ForLLM, "b") {
		t.Errorf("list result = %q, expected keys", result.ForLLM)
	}
}

func TestBlackboardTool_Delete(t *testing.T) {
	bb := NewBlackboard()
	bb.Set("tmp", "val", "x")
	tool := NewBlackboardTool(bb, "deleter")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "delete",
		"key":    "tmp",
	})
	if result.IsError {
		t.Fatalf("delete failed: %s", result.ForLLM)
	}
	if bb.Size() != 0 {
		t.Error("delete did not remove entry")
	}
}

func TestBlackboardTool_InvalidAction(t *testing.T) {
	bb := NewBlackboard()
	tool := NewBlackboardTool(bb, "test")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "invalid",
	})
	if !result.IsError {
		t.Error("expected error for invalid action")
	}
}

func TestBlackboardTool_MissingKey(t *testing.T) {
	bb := NewBlackboard()
	tool := NewBlackboardTool(bb, "test")

	// read without key
	result := tool.Execute(context.Background(), map[string]interface{}{
		"action": "read",
	})
	if !result.IsError {
		t.Error("expected error for read without key")
	}

	// write without key
	result = tool.Execute(context.Background(), map[string]interface{}{
		"action": "write",
		"value":  "test",
	})
	if !result.IsError {
		t.Error("expected error for write without key")
	}

	// write without value
	result = tool.Execute(context.Background(), map[string]interface{}{
		"action": "write",
		"key":    "k",
	})
	if !result.IsError {
		t.Error("expected error for write without value")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
