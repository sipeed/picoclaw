package tools

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// --- mock types ---

type mockRegistryTool struct {
	name   string
	desc   string
	params map[string]any
	result *ToolResult
}

func (m *mockRegistryTool) Name() string               { return m.name }
func (m *mockRegistryTool) Description() string        { return m.desc }
func (m *mockRegistryTool) Parameters() map[string]any { return m.params }
func (m *mockRegistryTool) Execute(_ context.Context, _ map[string]any) *ToolResult {
	return m.result
}

type mockContextAwareTool struct {
	mockRegistryTool
	lastCtx context.Context
}

func (m *mockContextAwareTool) Execute(ctx context.Context, _ map[string]any) *ToolResult {
	m.lastCtx = ctx
	return m.result
}

type mockAsyncRegistryTool struct {
	mockRegistryTool
	lastCB AsyncCallback
}

func (m *mockAsyncRegistryTool) ExecuteAsync(_ context.Context, args map[string]any, cb AsyncCallback) *ToolResult {
	m.lastCB = cb
	return m.result
}

// --- helpers ---

func newMockTool(name, desc string) *mockRegistryTool {
	return &mockRegistryTool{
		name:   name,
		desc:   desc,
		params: map[string]any{"type": "object"},
		result: SilentResult("ok"),
	}
}

// --- tests ---

func TestNewToolRegistry(t *testing.T) {
	r := NewToolRegistry()
	if r.Count() != 0 {
		t.Errorf("expected empty registry, got count %d", r.Count())
	}
	if len(r.List()) != 0 {
		t.Errorf("expected empty list, got %v", r.List())
	}
}

func TestToolRegistry_RegisterAndGet(t *testing.T) {
	r := NewToolRegistry()
	tool := newMockTool("echo", "echoes input")
	r.Register(tool)

	got, ok := r.Get("echo")
	if !ok {
		t.Fatal("expected to find registered tool")
	}
	if got.Name() != "echo" {
		t.Errorf("expected name 'echo', got %q", got.Name())
	}
}

func TestToolRegistry_Get_NotFound(t *testing.T) {
	r := NewToolRegistry()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected ok=false for unregistered tool")
	}
}

func TestToolRegistry_RegisterOverwrite(t *testing.T) {
	r := NewToolRegistry()
	r.Register(newMockTool("dup", "first"))
	r.Register(newMockTool("dup", "second"))

	if r.Count() != 1 {
		t.Errorf("expected count 1 after overwrite, got %d", r.Count())
	}
	tool, _ := r.Get("dup")
	if tool.Description() != "second" {
		t.Errorf("expected overwritten description 'second', got %q", tool.Description())
	}
}

func TestToolRegistry_Execute_Success(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&mockRegistryTool{
		name:   "greet",
		desc:   "says hello",
		params: map[string]any{},
		result: SilentResult("hello"),
	})

	result := r.Execute(context.Background(), "greet", nil)
	if result.IsError {
		t.Errorf("expected success, got error: %s", result.ForLLM)
	}
	if result.ForLLM != "hello" {
		t.Errorf("expected ForLLM 'hello', got %q", result.ForLLM)
	}
}

func TestToolRegistry_Execute_NotFound(t *testing.T) {
	r := NewToolRegistry()
	result := r.Execute(context.Background(), "missing", nil)
	if !result.IsError {
		t.Error("expected error for missing tool")
	}
	if !strings.Contains(result.ForLLM, "not found") {
		t.Errorf("expected 'not found' in error, got %q", result.ForLLM)
	}
	if result.Err == nil {
		t.Error("expected Err to be set via WithError")
	}
}

func TestToolRegistry_ExecuteWithContext_InjectsToolContext(t *testing.T) {
	r := NewToolRegistry()
	ct := &mockContextAwareTool{
		mockRegistryTool: *newMockTool("ctx_tool", "needs context"),
	}
	r.Register(ct)

	r.ExecuteWithContext(context.Background(), "ctx_tool", nil, "telegram", "chat-42", nil)

	if ct.lastCtx == nil {
		t.Fatal("expected Execute to be called")
	}
	if got := ToolChannel(ct.lastCtx); got != "telegram" {
		t.Errorf("expected channel 'telegram', got %q", got)
	}
	if got := ToolChatID(ct.lastCtx); got != "chat-42" {
		t.Errorf("expected chatID 'chat-42', got %q", got)
	}
}

func TestToolRegistry_ExecuteWithContext_EmptyContext(t *testing.T) {
	r := NewToolRegistry()
	ct := &mockContextAwareTool{
		mockRegistryTool: *newMockTool("ctx_tool", "needs context"),
	}
	r.Register(ct)

	r.ExecuteWithContext(context.Background(), "ctx_tool", nil, "", "", nil)

	if ct.lastCtx == nil {
		t.Fatal("expected Execute to be called")
	}
	// Empty values are still injected; tools decide what to do with them.
	if got := ToolChannel(ct.lastCtx); got != "" {
		t.Errorf("expected empty channel, got %q", got)
	}
	if got := ToolChatID(ct.lastCtx); got != "" {
		t.Errorf("expected empty chatID, got %q", got)
	}
}

func TestToolRegistry_ExecuteWithContext_AsyncCallback(t *testing.T) {
	r := NewToolRegistry()
	at := &mockAsyncRegistryTool{
		mockRegistryTool: *newMockTool("async_tool", "async work"),
	}
	at.result = AsyncResult("started")
	r.Register(at)

	called := false
	cb := func(_ context.Context, _ *ToolResult) { called = true }

	result := r.ExecuteWithContext(context.Background(), "async_tool", nil, "", "", cb)
	if at.lastCB == nil {
		t.Error("expected ExecuteAsync to have received a callback")
	}
	if !result.Async {
		t.Error("expected async result")
	}

	at.lastCB(context.Background(), SilentResult("done"))
	if !called {
		t.Error("expected callback to be invoked")
	}
}

func TestToolRegistry_GetDefinitions(t *testing.T) {
	r := NewToolRegistry()
	r.Register(newMockTool("alpha", "tool A"))

	defs := r.GetDefinitions()
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	if defs[0]["type"] != "function" {
		t.Errorf("expected type 'function', got %v", defs[0]["type"])
	}
	fn, ok := defs[0]["function"].(map[string]any)
	if !ok {
		t.Fatal("expected 'function' key to be a map")
	}
	if fn["name"] != "alpha" {
		t.Errorf("expected name 'alpha', got %v", fn["name"])
	}
	if fn["description"] != "tool A" {
		t.Errorf("expected description 'tool A', got %v", fn["description"])
	}
}

func TestToolRegistry_ToProviderDefs(t *testing.T) {
	r := NewToolRegistry()
	params := map[string]any{"type": "object", "properties": map[string]any{}}
	r.Register(&mockRegistryTool{
		name:   "beta",
		desc:   "tool B",
		params: params,
		result: SilentResult("ok"),
	})

	defs := r.ToProviderDefs()
	if len(defs) != 1 {
		t.Fatalf("expected 1 provider def, got %d", len(defs))
	}

	want := providers.ToolDefinition{
		Type: "function",
		Function: providers.ToolFunctionDefinition{
			Name:        "beta",
			Description: "tool B",
			Parameters:  params,
		},
	}
	got := defs[0]
	if got.Type != want.Type {
		t.Errorf("Type: want %q, got %q", want.Type, got.Type)
	}
	if got.Function.Name != want.Function.Name {
		t.Errorf("Name: want %q, got %q", want.Function.Name, got.Function.Name)
	}
	if got.Function.Description != want.Function.Description {
		t.Errorf("Description: want %q, got %q", want.Function.Description, got.Function.Description)
	}
}

func TestToolRegistry_List(t *testing.T) {
	r := NewToolRegistry()
	r.Register(newMockTool("x", ""))
	r.Register(newMockTool("y", ""))

	names := r.List()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}

	nameSet := map[string]bool{}
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["x"] || !nameSet["y"] {
		t.Errorf("expected names {x, y}, got %v", names)
	}
}

func TestToolRegistry_Count(t *testing.T) {
	r := NewToolRegistry()
	if r.Count() != 0 {
		t.Errorf("expected 0, got %d", r.Count())
	}

	r.Register(newMockTool("a", ""))
	r.Register(newMockTool("b", ""))
	if r.Count() != 2 {
		t.Errorf("expected 2, got %d", r.Count())
	}

	r.Register(newMockTool("a", "replaced"))
	if r.Count() != 2 {
		t.Errorf("expected 2 after overwrite, got %d", r.Count())
	}
}

func TestToolRegistry_GetSummaries(t *testing.T) {
	r := NewToolRegistry()
	r.Register(newMockTool("read_file", "Reads a file"))

	summaries := r.GetSummaries()
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if !strings.Contains(summaries[0], "`read_file`") {
		t.Errorf("expected backtick-quoted name in summary, got %q", summaries[0])
	}
	if !strings.Contains(summaries[0], "Reads a file") {
		t.Errorf("expected description in summary, got %q", summaries[0])
	}
}

func TestToolToSchema(t *testing.T) {
	tool := newMockTool("demo", "demo tool")
	schema := ToolToSchema(tool)

	if schema["type"] != "function" {
		t.Errorf("expected type 'function', got %v", schema["type"])
	}
	fn, ok := schema["function"].(map[string]any)
	if !ok {
		t.Fatal("expected 'function' to be a map")
	}
	if fn["name"] != "demo" {
		t.Errorf("expected name 'demo', got %v", fn["name"])
	}
	if fn["description"] != "demo tool" {
		t.Errorf("expected description 'demo tool', got %v", fn["description"])
	}
	if fn["parameters"] == nil {
		t.Error("expected parameters to be set")
	}
}

func TestToolRegistry_Clone(t *testing.T) {
	r := NewToolRegistry()
	r.Register(newMockTool("read_file", "reads files"))
	r.Register(newMockTool("exec", "runs commands"))
	r.Register(newMockTool("web_search", "searches the web"))

	clone := r.Clone()

	// Clone should have the same tools
	if clone.Count() != 3 {
		t.Errorf("expected clone to have 3 tools, got %d", clone.Count())
	}
	for _, name := range []string{"read_file", "exec", "web_search"} {
		if _, ok := clone.Get(name); !ok {
			t.Errorf("expected clone to have tool %q", name)
		}
	}

	// Registering on parent should NOT affect clone
	r.Register(newMockTool("spawn", "spawns subagent"))
	if r.Count() != 4 {
		t.Errorf("expected parent to have 4 tools, got %d", r.Count())
	}
	if clone.Count() != 3 {
		t.Errorf("expected clone to still have 3 tools after parent mutation, got %d", clone.Count())
	}
	if _, ok := clone.Get("spawn"); ok {
		t.Error("expected clone NOT to have 'spawn' tool registered on parent after cloning")
	}

	// Registering on clone should NOT affect parent
	clone.Register(newMockTool("custom", "custom tool"))
	if clone.Count() != 4 {
		t.Errorf("expected clone to have 4 tools, got %d", clone.Count())
	}
	if _, ok := r.Get("custom"); ok {
		t.Error("expected parent NOT to have 'custom' tool registered on clone")
	}
}

func TestToolRegistry_Clone_Empty(t *testing.T) {
	r := NewToolRegistry()
	clone := r.Clone()
	if clone.Count() != 0 {
		t.Errorf("expected empty clone, got count %d", clone.Count())
	}
}

func TestToolRegistry_Clone_PreservesHiddenToolState(t *testing.T) {
	r := NewToolRegistry()
	r.RegisterHidden(newMockTool("mcp_tool", "dynamic MCP tool"))

	clone := r.Clone()

	// Hidden tools with TTL=0 should not be gettable (same behavior as parent)
	if _, ok := clone.Get("mcp_tool"); ok {
		t.Error("expected hidden tool with TTL=0 to be invisible in clone")
	}

	// But the entry should exist (count includes hidden tools)
	if clone.Count() != 1 {
		t.Errorf("expected clone count 1 (hidden entry exists), got %d", clone.Count())
	}
}

func TestToolRegistry_Clone_PreservesTTLValue(t *testing.T) {
	r := NewToolRegistry()
	r.RegisterHidden(newMockTool("ttl_tool", "tool with TTL"))

	// Manually set a non-zero TTL on the entry
	r.mu.RLock()
	if entry, ok := r.tools["ttl_tool"]; ok {
		entry.TTL = 5
	}
	r.mu.RUnlock()

	clone := r.Clone()

	// Verify TTL value is preserved in the clone
	clone.mu.RLock()
	defer clone.mu.RUnlock()
	entry, ok := clone.tools["ttl_tool"]
	if !ok {
		t.Fatal("expected ttl_tool to exist in clone")
	}
	if entry.TTL != 5 {
		t.Errorf("expected TTL=5 in clone, got %d", entry.TTL)
	}
}

func TestToolRegistry_ConcurrentAccess(t *testing.T) {
	r := NewToolRegistry()
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			name := string(rune('A' + n%26))
			r.Register(newMockTool(name, "concurrent"))
			r.Get(name)
			r.Count()
			r.List()
			r.GetDefinitions()
		}(i)
	}

	wg.Wait()

	if r.Count() == 0 {
		t.Error("expected tools to be registered after concurrent access")
	}
}

// --- Panic and abnormal exit tests ---

// mockPanicTool is a tool that panics during execution
type mockPanicTool struct {
	name       string
	panicValue any
}

func (m *mockPanicTool) Name() string               { return m.name }
func (m *mockPanicTool) Description() string        { return "a tool that panics" }
func (m *mockPanicTool) Parameters() map[string]any { return map[string]any{"type": "object"} }
func (m *mockPanicTool) Execute(_ context.Context, _ map[string]any) *ToolResult {
	panic(m.panicValue)
}

// mockNilResultTool is a tool that returns nil
type mockNilResultTool struct {
	name string
}

func (m *mockNilResultTool) Name() string               { return m.name }
func (m *mockNilResultTool) Description() string        { return "a tool that returns nil" }
func (m *mockNilResultTool) Parameters() map[string]any { return map[string]any{"type": "object"} }
func (m *mockNilResultTool) Execute(_ context.Context, _ map[string]any) *ToolResult {
	return nil
}

func TestToolRegistry_Execute_PanicRecovery(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&mockPanicTool{
		name:       "panic_tool",
		panicValue: "something went terribly wrong",
	})

	// Should not panic, should return error result
	result := r.Execute(context.Background(), "panic_tool", nil)

	if result == nil {
		t.Fatal("expected non-nil result after panic recovery")
	}
	if !result.IsError {
		t.Error("expected IsError=true after panic")
	}
	if !strings.Contains(result.ForLLM, "panic") {
		t.Errorf("expected 'panic' in error message, got %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "panic_tool") {
		t.Errorf("expected tool name in error message, got %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "something went terribly wrong") {
		t.Errorf("expected panic value in error message, got %q", result.ForLLM)
	}
	if result.Err == nil {
		t.Error("expected Err to be set")
	}
}

func TestToolRegistry_Execute_PanicRecovery_ErrorType(t *testing.T) {
	r := NewToolRegistry()

	// Test with error type panic
	r.Register(&mockPanicTool{
		name:       "error_panic_tool",
		panicValue: errors.New("custom error panic"),
	})

	result := r.Execute(context.Background(), "error_panic_tool", nil)

	if !result.IsError {
		t.Error("expected IsError=true")
	}
	if !strings.Contains(result.ForLLM, "custom error panic") {
		t.Errorf("expected error message in ForLLM, got %q", result.ForLLM)
	}
}

func TestToolRegistry_Execute_PanicRecovery_IntType(t *testing.T) {
	r := NewToolRegistry()

	// Test with int type panic
	r.Register(&mockPanicTool{
		name:       "int_panic_tool",
		panicValue: 42,
	})

	result := r.Execute(context.Background(), "int_panic_tool", nil)

	if !result.IsError {
		t.Error("expected IsError=true")
	}
	if !strings.Contains(result.ForLLM, "42") {
		t.Errorf("expected panic value '42' in ForLLM, got %q", result.ForLLM)
	}
}

func TestToolRegistry_Execute_NilResultHandling(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&mockNilResultTool{name: "nil_tool"})

	result := r.Execute(context.Background(), "nil_tool", nil)

	if result == nil {
		t.Fatal("expected non-nil result when tool returns nil")
	}
	if !result.IsError {
		t.Error("expected IsError=true for nil result")
	}
	if !strings.Contains(result.ForLLM, "nil_tool") {
		t.Errorf("expected tool name in error message, got %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "nil result") {
		t.Errorf("expected 'nil result' in error message, got %q", result.ForLLM)
	}
	if result.Err == nil {
		t.Error("expected Err to be set")
	}
}

func TestToolRegistry_ExecuteWithContext_PanicRecovery(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&mockPanicTool{
		name:       "ctx_panic_tool",
		panicValue: "context panic test",
	})

	// Should not panic even with context
	result := r.ExecuteWithContext(
		context.Background(),
		"ctx_panic_tool",
		map[string]any{"key": "value"},
		"telegram",
		"chat-123",
		nil,
	)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.IsError {
		t.Error("expected IsError=true")
	}
	if !strings.Contains(result.ForLLM, "context panic test") {
		t.Errorf("expected panic message, got %q", result.ForLLM)
	}
}

func TestToolRegistry_Execute_PanicDoesNotAffectOtherTools(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&mockPanicTool{name: "bad_tool", panicValue: "boom"})
	r.Register(&mockRegistryTool{
		name:   "good_tool",
		desc:   "works fine",
		params: map[string]any{},
		result: SilentResult("success"),
	})

	// First, trigger the panic
	result1 := r.Execute(context.Background(), "bad_tool", nil)
	if !result1.IsError {
		t.Error("expected error from panic tool")
	}

	// Then, verify the good tool still works
	result2 := r.Execute(context.Background(), "good_tool", nil)
	if result2.IsError {
		t.Errorf("expected success from good tool, got error: %s", result2.ForLLM)
	}
	if result2.ForLLM != "success" {
		t.Errorf("expected 'success', got %q", result2.ForLLM)
	}
}
