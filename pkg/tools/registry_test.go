package tools

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/stretchr/testify/assert"
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

type mockCtxTool struct {
	mockRegistryTool
	channel string
	chatID  string
}

func (m *mockCtxTool) SetContext(channel, chatID string) {
	m.channel = channel
	m.chatID = chatID
}

type mockAsyncRegistryTool struct {
	mockRegistryTool
	cb AsyncCallback
}

func (m *mockAsyncRegistryTool) SetCallback(cb AsyncCallback) {
	m.cb = cb
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

func TestToolRegistry_ExecuteWithContext_ContextualTool(t *testing.T) {
	r := NewToolRegistry()
	ct := &mockCtxTool{
		mockRegistryTool: *newMockTool("ctx_tool", "needs context"),
	}
	r.Register(ct)

	r.ExecuteWithContext(context.Background(), "ctx_tool", nil, "telegram", "chat-42", nil)

	if ct.channel != "telegram" {
		t.Errorf("expected channel 'telegram', got %q", ct.channel)
	}
	if ct.chatID != "chat-42" {
		t.Errorf("expected chatID 'chat-42', got %q", ct.chatID)
	}
}

func TestToolRegistry_ExecuteWithContext_SkipsEmptyContext(t *testing.T) {
	r := NewToolRegistry()
	ct := &mockCtxTool{
		mockRegistryTool: *newMockTool("ctx_tool", "needs context"),
	}
	r.Register(ct)

	r.ExecuteWithContext(context.Background(), "ctx_tool", nil, "", "", nil)

	if ct.channel != "" || ct.chatID != "" {
		t.Error("SetContext should not be called with empty channel/chatID")
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
	if at.cb == nil {
		t.Error("expected SetCallback to have been called")
	}
	if !result.Async {
		t.Error("expected async result")
	}

	at.cb(context.Background(), SilentResult("done"))
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

func TestToolRegistry_ConcurrentAccess(t *testing.T) {
	r := NewToolRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
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

func TestToolRegistry_RegisterWithFilter_NoFilter(t *testing.T) {
	r := NewToolRegistry()
	tool := newMockTool("public_tool", "always visible")
	r.RegisterWithFilter(tool, nil)

	// Tool should be visible in all contexts
	ctx := ToolVisibilityContext{
		Channel:   "telegram",
		ChatID:    "chat-123",
		UserID:    "user-456",
		UserRoles: []string{"user"},
	}

	defs := r.GetDefinitionsForContext(ctx)
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	if defs[0]["function"].(map[string]any)["name"] != "public_tool" {
		t.Error("expected public_tool to be visible")
	}
}

func TestToolRegistry_RegisterWithFilter_AdminOnly(t *testing.T) {
	r := NewToolRegistry()

	// Register admin-only tool
	adminTool := newMockTool("admin_tool", "admin only")
	r.RegisterWithFilter(adminTool, func(ctx ToolVisibilityContext) bool {
		for _, role := range ctx.UserRoles {
			if role == "admin" {
				return true
			}
		}
		return false
	})

	// Regular user should not see it
	userCtx := ToolVisibilityContext{
		Channel:   "telegram",
		ChatID:    "chat-123",
		UserID:    "user-456",
		UserRoles: []string{"user"},
	}

	defs := r.GetDefinitionsForContext(userCtx)
	if len(defs) != 0 {
		t.Errorf("expected 0 definitions for regular user, got %d", len(defs))
	}

	// Admin should see it
	adminCtx := ToolVisibilityContext{
		Channel:   "telegram",
		ChatID:    "chat-123",
		UserID:    "admin-789",
		UserRoles: []string{"admin", "user"},
	}

	defs = r.GetDefinitionsForContext(adminCtx)
	if len(defs) != 1 {
		t.Errorf("expected 1 definition for admin, got %d", len(defs))
	}
}

func TestToolRegistry_RegisterWithFilter_ChannelSpecific(t *testing.T) {
	r := NewToolRegistry()

	// Register Telegram-only tool
	telegramTool := newMockTool("telegram_tool", "telegram only")
	r.RegisterWithFilter(telegramTool, func(ctx ToolVisibilityContext) bool {
		return ctx.Channel == "telegram"
	})

	// Should be visible in Telegram
	telegramCtx := ToolVisibilityContext{
		Channel:   "telegram",
		ChatID:    "chat-123",
		UserID:    "user-456",
		UserRoles: []string{"user"},
	}

	defs := r.GetDefinitionsForContext(telegramCtx)
	if len(defs) != 1 {
		t.Errorf("expected 1 definition in Telegram, got %d", len(defs))
	}

	// Should not be visible in other channels
	slackCtx := ToolVisibilityContext{
		Channel:   "slack",
		ChatID:    "chat-123",
		UserID:    "user-456",
		UserRoles: []string{"user"},
	}

	defs = r.GetDefinitionsForContext(slackCtx)
	if len(defs) != 0 {
		t.Errorf("expected 0 definitions in Slack, got %d", len(defs))
	}
}

func TestToolRegistry_RegisterWithFilter_MixedTools(t *testing.T) {
	r := NewToolRegistry()

	// Register mix of filtered and unfiltered tools
	r.RegisterWithFilter(newMockTool("public1", "always visible"), nil)
	r.RegisterWithFilter(newMockTool("public2", "always visible"), nil)

	r.RegisterWithFilter(newMockTool("admin_only", "admin only"), func(ctx ToolVisibilityContext) bool {
		for _, role := range ctx.UserRoles {
			if role == "admin" {
				return true
			}
		}
		return false
	})

	r.RegisterWithFilter(newMockTool("telegram_only", "telegram only"), func(ctx ToolVisibilityContext) bool {
		return ctx.Channel == "telegram"
	})

	// Regular user in Telegram should see: public1, public2, telegram_only (3 tools)
	userCtx := ToolVisibilityContext{
		Channel:   "telegram",
		ChatID:    "chat-123",
		UserID:    "user-456",
		UserRoles: []string{"user"},
	}

	defs := r.GetDefinitionsForContext(userCtx)
	if len(defs) != 3 {
		t.Errorf("expected 3 definitions for regular user in Telegram, got %d", len(defs))
	}

	// Admin in Slack should see: public1, public2, admin_only (3 tools)
	adminCtx := ToolVisibilityContext{
		Channel:   "slack",
		ChatID:    "chat-123",
		UserID:    "admin-789",
		UserRoles: []string{"admin"},
	}

	defs = r.GetDefinitionsForContext(adminCtx)
	if len(defs) != 3 {
		t.Errorf("expected 3 definitions for admin in Slack, got %d", len(defs))
	}

	// Admin in Telegram should see all 4 tools
	adminTelegramCtx := ToolVisibilityContext{
		Channel:   "telegram",
		ChatID:    "chat-123",
		UserID:    "admin-789",
		UserRoles: []string{"admin"},
	}

	defs = r.GetDefinitionsForContext(adminTelegramCtx)
	if len(defs) != 4 {
		t.Errorf("expected 4 definitions for admin in Telegram, got %d", len(defs))
	}
}

func TestToolRegistry_GetDefinitionsForContext_EmptyRegistry(t *testing.T) {
	r := NewToolRegistry()

	ctx := ToolVisibilityContext{
		Channel:   "telegram",
		ChatID:    "chat-123",
		UserID:    "user-456",
		UserRoles: []string{"user"},
	}

	defs := r.GetDefinitionsForContext(ctx)
	if len(defs) != 0 {
		t.Errorf("expected 0 definitions in empty registry, got %d", len(defs))
	}
}

func TestToolRegistry_RegisterWithFilter_UpdateFilter(t *testing.T) {
	r := NewToolRegistry()

	// Register with filter
	r.RegisterWithFilter(newMockTool("changing_tool", "changes filter"), func(ctx ToolVisibilityContext) bool {
		return ctx.Channel == "telegram"
	})

	// Should only be visible in Telegram
	telegramCtx := ToolVisibilityContext{Channel: "telegram"}
	slackCtx := ToolVisibilityContext{Channel: "slack"}

	if len(r.GetDefinitionsForContext(telegramCtx)) != 1 {
		t.Error("expected tool visible in Telegram")
	}
	if len(r.GetDefinitionsForContext(slackCtx)) != 0 {
		t.Error("expected tool hidden in Slack")
	}

	// Re-register without filter (should make it public)
	r.RegisterWithFilter(newMockTool("changing_tool", "now public"), nil)

	if len(r.GetDefinitionsForContext(telegramCtx)) != 1 {
		t.Error("expected tool still visible in Telegram")
	}
	if len(r.GetDefinitionsForContext(slackCtx)) != 1 {
		t.Error("expected tool now visible in Slack")
	}
}

func TestToolRegistry_GetDefinitionsForContext_ConcurrentAccess(t *testing.T) {
	r := NewToolRegistry()

	// Register some tools with different filters
	r.RegisterWithFilter(newMockTool("public", "always visible"), nil)
	r.RegisterWithFilter(newMockTool("admin_only", "admin only"), func(ctx ToolVisibilityContext) bool {
		for _, role := range ctx.UserRoles {
			return role == "admin"
		}
		return false
	})

	var wg sync.WaitGroup
	errorChan := make(chan string, 100)

	// Multiple goroutines querying different contexts
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(iteration int) {
			defer wg.Done()
			ctx := ToolVisibilityContext{
				Channel:   "telegram",
				ChatID:    "chat-" + string(rune(iteration)),
				UserID:    "user-" + string(rune(iteration)),
				UserRoles: []string{"user", "admin"},
			}

			defs := r.GetDefinitionsForContext(ctx)
			// Should always get consistent results
			if len(defs) < 1 || len(defs) > 2 {
				errorChan <- "unexpected definition count: " + string(rune(len(defs)))
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	for errMsg := range errorChan {
		t.Error(errMsg)
	}
}

func TestToolRegistry_MultiTenant_AdminVsRegularUser(t *testing.T) {
	registry := NewToolRegistry()

	// Register a general tool visible to everyone
	generalTool := newMockTool("read_file", "Read files")
	registry.Register(generalTool)

	// Register an admin-only tool with filter
	adminTool := newMockTool("admin_deploy", "Deploy application")
	registry.RegisterWithFilter(adminTool, func(ctx ToolVisibilityContext) bool {
		for _, role := range ctx.UserRoles {
			if role == "admin" {
				return true
			}
		}
		return false
	})

	// Register a user-only tool
	userTool := newMockTool("user_feedback", "Submit feedback")
	registry.RegisterWithFilter(userTool, func(ctx ToolVisibilityContext) bool {
		for _, role := range ctx.UserRoles {
			if role == "user" {
				return true
			}
		}
		return false
	})

	t.Run("admin user sees all tools", func(t *testing.T) {
		ctx := ToolVisibilityContext{
			Channel:   "telegram",
			ChatID:    "chat123",
			UserID:    "admin_user",
			UserRoles: []string{"admin"},
			Timestamp: time.Now(),
		}

		definitions := registry.GetDefinitionsForContext(ctx)

		// Admin should see both general and admin tools
		assert.Len(t, definitions, 2)

		// Check that admin tool is included
		foundAdmin := false
		foundGeneral := false
		for _, def := range definitions {
			// ToolToSchema returns: {"type": "function", "function": {"name": "...", ...}}
			if function, ok := def["function"].(map[string]any); ok {
				if name, ok := function["name"].(string); ok {
					if name == "admin_deploy" {
						foundAdmin = true
					}
					if name == "read_file" {
						foundGeneral = true
					}
				}
			}
		}

		assert.True(t, foundAdmin, "admin should see admin_deploy tool")
		assert.True(t, foundGeneral, "admin should see read_file tool")
	})

	t.Run("regular user sees limited tools", func(t *testing.T) {
		ctx := ToolVisibilityContext{
			Channel:   "telegram",
			ChatID:    "chat456",
			UserID:    "regular_user",
			UserRoles: []string{"user"},
			Timestamp: time.Now(),
		}

		definitions := registry.GetDefinitionsForContext(ctx)

		// Regular user should see general and user tools, but not admin tools
		assert.Len(t, definitions, 2)

		foundUser := false
		foundGeneral := false
		foundAdmin := false

		for _, def := range definitions {
			if function, ok := def["function"].(map[string]any); ok {
				if name, ok := function["name"].(string); ok {
					if name == "user_feedback" {
						foundUser = true
					}
					if name == "read_file" {
						foundGeneral = true
					}
					if name == "admin_deploy" {
						foundAdmin = true
					}
				}
			}
		}

		assert.True(t, foundUser, "user should see user_feedback tool")
		assert.True(t, foundGeneral, "user should see read_file tool")
		assert.False(t, foundAdmin, "user should NOT see admin_deploy tool")
	})

	t.Run("user with multiple roles", func(t *testing.T) {
		ctx := ToolVisibilityContext{
			Channel:   "slack",
			ChatID:    "chat789",
			UserID:    "power_user",
			UserRoles: []string{"user", "admin"},
			Timestamp: time.Now(),
		}

		definitions := registry.GetDefinitionsForContext(ctx)

		// User with both roles should see all tools
		assert.Len(t, definitions, 3)
	})

	t.Run("user with no roles", func(t *testing.T) {
		ctx := ToolVisibilityContext{
			Channel:   "discord",
			ChatID:    "chat999",
			UserID:    "guest",
			UserRoles: []string{},
			Timestamp: time.Now(),
		}

		definitions := registry.GetDefinitionsForContext(ctx)

		// Guest should only see general tools (no role-based restrictions)
		assert.Len(t, definitions, 1)

		// Check the tool name
		if len(definitions) > 0 {
			if function, ok := definitions[0]["function"].(map[string]any); ok {
				if name, ok := function["name"].(string); ok {
					assert.Contains(t, name, "read_file")
				}
			}
		}
	})
}

func TestToolRegistry_ChannelSpecificToolVisibility(t *testing.T) {
	registry := NewToolRegistry()

	// Register a Telegram-specific tool (e.g., sticker creation)
	telegramTool := newMockTool("create_sticker", "Create sticker")
	registry.RegisterWithFilter(telegramTool, func(ctx ToolVisibilityContext) bool {
		return ctx.Channel == "telegram"
	})

	// Register a Slack-specific tool (e.g., channel management)
	slackTool := newMockTool("manage_channel", "Manage Slack channel")
	registry.RegisterWithFilter(slackTool, func(ctx ToolVisibilityContext) bool {
		return ctx.Channel == "slack"
	})

	// Register a general tool available on all channels
	generalTool := newMockTool("search_web", "Search web")
	registry.Register(generalTool)

	t.Run("Telegram channel", func(t *testing.T) {
		ctx := ToolVisibilityContext{
			Channel:   "telegram",
			ChatID:    "tg_chat",
			UserID:    "user1",
			UserRoles: []string{"user"},
			Timestamp: time.Now(),
		}

		definitions := registry.GetDefinitionsForContext(ctx)

		// Should see general + Telegram-specific tools
		assert.Len(t, definitions, 2)

		foundTelegram := false
		foundGeneral := false
		for _, def := range definitions {
			if function, ok := def["function"].(map[string]any); ok {
				if name, ok := function["name"].(string); ok {
					if name == "create_sticker" {
						foundTelegram = true
					}
					if name == "search_web" {
						foundGeneral = true
					}
				}
			}
		}

		assert.True(t, foundTelegram, "should see Telegram-specific tool")
		assert.True(t, foundGeneral, "should see general tool")
	})

	t.Run("Slack channel", func(t *testing.T) {
		ctx := ToolVisibilityContext{
			Channel:   "slack",
			ChatID:    "slack_channel",
			UserID:    "user2",
			UserRoles: []string{"user"},
			Timestamp: time.Now(),
		}

		definitions := registry.GetDefinitionsForContext(ctx)

		// Should see general + Slack-specific tools
		assert.Len(t, definitions, 2)

		foundSlack := false
		foundGeneral := false
		for _, def := range definitions {
			if function, ok := def["function"].(map[string]any); ok {
				if name, ok := function["name"].(string); ok {
					if name == "manage_channel" {
						foundSlack = true
					}
					if name == "search_web" {
						foundGeneral = true
					}
				}
			}
		}

		assert.True(t, foundSlack, "should see Slack-specific tool")
		assert.True(t, foundGeneral, "should see general tool")
	})

	t.Run("Other channel (no specific tools)", func(t *testing.T) {
		ctx := ToolVisibilityContext{
			Channel:   "discord",
			ChatID:    "discord_server",
			UserID:    "user3",
			UserRoles: []string{"user"},
			Timestamp: time.Now(),
		}

		definitions := registry.GetDefinitionsForContext(ctx)

		// Should only see general tools
		assert.Len(t, definitions, 1)

		if len(definitions) > 0 {
			if function, ok := definitions[0]["function"].(map[string]any); ok {
				if name, ok := function["name"].(string); ok {
					assert.Equal(t, "search_web", name)
				}
			}
		}
	})
}

func TestToolRegistry_ChatSpecificToolVisibility(t *testing.T) {
	registry := NewToolRegistry()

	// Register a tool only visible in specific chat (e.g., VIP support chat)
	vipTool := newMockTool("vip_support", "VIP support")
	registry.RegisterWithFilter(vipTool, func(ctx ToolVisibilityContext) bool {
		return ctx.ChatID == "vip_chat_001"
	})

	// Register a general tool
	generalTool := newMockTool("help", "Get help")
	registry.Register(generalTool)

	t.Run("VIP chat", func(t *testing.T) {
		ctx := ToolVisibilityContext{
			Channel:   "wecom",
			ChatID:    "vip_chat_001",
			UserID:    "vip_user",
			UserRoles: []string{"vip"},
			Timestamp: time.Now(),
		}

		definitions := registry.GetDefinitionsForContext(ctx)

		// VIP should see both general and VIP tools
		assert.Len(t, definitions, 2)
	})

	t.Run("Regular chat", func(t *testing.T) {
		ctx := ToolVisibilityContext{
			Channel:   "wecom",
			ChatID:    "regular_chat",
			UserID:    "regular_user",
			UserRoles: []string{"user"},
			Timestamp: time.Now(),
		}

		definitions := registry.GetDefinitionsForContext(ctx)

		// Regular users should only see general tool
		assert.Len(t, definitions, 1)

		if len(definitions) > 0 {
			if function, ok := definitions[0]["function"].(map[string]any); ok {
				if name, ok := function["name"].(string); ok {
					assert.Equal(t, "help", name)
				}
			}
		}
	})
}
