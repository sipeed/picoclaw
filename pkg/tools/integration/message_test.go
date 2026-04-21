package integrationtools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/session"
)

func TestMessageTool_Execute_Success(t *testing.T) {
	tool := NewMessageTool()

	var sentChannel, sentChatID, sentContent string
	tool.SetSendCallback(func(ctx context.Context, channel, chatID, content, replyToMessageID string) error {
		sentChannel = channel
		sentChatID = chatID
		sentContent = content
		if ToolAgentID(ctx) != "" || ToolSessionKey(ctx) != "" || ToolSessionScope(ctx) != nil {
			t.Fatalf("expected empty turn metadata in basic context, got agent=%q session=%q scope=%+v",
				ToolAgentID(ctx), ToolSessionKey(ctx), ToolSessionScope(ctx))
		}
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{
		"content": "Hello, world!",
	}

	result := tool.Execute(ctx, args)

	// Verify message was sent with correct parameters
	if sentChannel != "test-channel" {
		t.Errorf("Expected channel 'test-channel', got '%s'", sentChannel)
	}
	if sentChatID != "test-chat-id" {
		t.Errorf("Expected chatID 'test-chat-id', got '%s'", sentChatID)
	}
	if sentContent != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", sentContent)
	}

	// Verify ToolResult meets US-011 criteria:
	// - Send success returns SilentResult (Silent=true)
	if !result.Silent {
		t.Error("Expected Silent=true for successful send")
	}

	// - ForLLM contains send status description
	if result.ForLLM != "Message sent to test-channel:test-chat-id" {
		t.Errorf("Expected ForLLM 'Message sent to test-channel:test-chat-id', got '%s'", result.ForLLM)
	}

	// - ForUser is empty (user already received message directly)
	if result.ForUser != "" {
		t.Errorf("Expected ForUser to be empty, got '%s'", result.ForUser)
	}

	// - IsError should be false
	if result.IsError {
		t.Error("Expected IsError=false for successful send")
	}
}

func TestMessageTool_Execute_WithCustomChannel(t *testing.T) {
	tool := NewMessageTool()

	var sentChannel, sentChatID string
	tool.SetSendCallback(func(ctx context.Context, channel, chatID, content, replyToMessageID string) error {
		sentChannel = channel
		sentChatID = chatID
		return nil
	})

	ctx := WithToolContext(context.Background(), "default-channel", "default-chat-id")
	args := map[string]any{
		"content": "Test message",
		"channel": "custom-channel",
		"chat_id": "custom-chat-id",
	}

	result := tool.Execute(ctx, args)

	// Verify custom channel/chatID were used instead of defaults
	if sentChannel != "custom-channel" {
		t.Errorf("Expected channel 'custom-channel', got '%s'", sentChannel)
	}
	if sentChatID != "custom-chat-id" {
		t.Errorf("Expected chatID 'custom-chat-id', got '%s'", sentChatID)
	}

	if !result.Silent {
		t.Error("Expected Silent=true")
	}
	if result.ForLLM != "Message sent to custom-channel:custom-chat-id" {
		t.Errorf("Expected ForLLM 'Message sent to custom-channel:custom-chat-id', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Execute_SendFailure(t *testing.T) {
	tool := NewMessageTool()

	sendErr := errors.New("network error")
	tool.SetSendCallback(func(ctx context.Context, channel, chatID, content, replyToMessageID string) error {
		return sendErr
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{
		"content": "Test message",
	}

	result := tool.Execute(ctx, args)

	// Verify ToolResult for send failure:
	// - Send failure returns ErrorResult (IsError=true)
	if !result.IsError {
		t.Error("Expected IsError=true for failed send")
	}

	// - ForLLM contains error description
	expectedErrMsg := "sending message: network error"
	if result.ForLLM != expectedErrMsg {
		t.Errorf("Expected ForLLM '%s', got '%s'", expectedErrMsg, result.ForLLM)
	}

	// - Err field should contain original error
	if result.Err == nil {
		t.Error("Expected Err to be set")
	}
	if result.Err != sendErr {
		t.Errorf("Expected Err to be sendErr, got %v", result.Err)
	}
}

func TestMessageTool_Execute_MissingContent(t *testing.T) {
	tool := NewMessageTool()

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{} // content missing

	result := tool.Execute(ctx, args)

	// Verify error result for missing content
	if !result.IsError {
		t.Error("Expected IsError=true for missing content")
	}
	if result.ForLLM != "content is required" {
		t.Errorf("Expected ForLLM 'content is required', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Execute_NoTargetChannel(t *testing.T) {
	tool := NewMessageTool()
	// No WithToolContext — channel/chatID are empty

	tool.SetSendCallback(func(ctx context.Context, channel, chatID, content, replyToMessageID string) error {
		return nil
	})

	ctx := context.Background()
	args := map[string]any{
		"content": "Test message",
	}

	result := tool.Execute(ctx, args)

	// Verify error when no target channel specified
	if !result.IsError {
		t.Error("Expected IsError=true when no target channel")
	}
	if result.ForLLM != "No target channel/chat specified" {
		t.Errorf("Expected ForLLM 'No target channel/chat specified', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Execute_NotConfigured(t *testing.T) {
	tool := NewMessageTool()
	// No SetSendCallback called

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{
		"content": "Test message",
	}

	result := tool.Execute(ctx, args)

	// Verify error when send callback not configured
	if !result.IsError {
		t.Error("Expected IsError=true when send callback not configured")
	}
	if result.ForLLM != "Message sending not configured" {
		t.Errorf("Expected ForLLM 'Message sending not configured', got '%s'", result.ForLLM)
	}
}

func TestMessageTool_Name(t *testing.T) {
	tool := NewMessageTool()
	if tool.Name() != "message" {
		t.Errorf("Expected name 'message', got '%s'", tool.Name())
	}
}

func TestMessageTool_Description(t *testing.T) {
	tool := NewMessageTool()
	desc := tool.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}

	for _, snippet := range []string{
		"card",
		"options",
	} {
		if !strings.Contains(desc, snippet) {
			t.Fatalf("Description() missing snippet %q", snippet)
		}
	}
}

func TestMessageTool_Parameters(t *testing.T) {
	tool := NewMessageTool()
	params := tool.Parameters()

	// Verify parameters structure
	typ, ok := params["type"].(string)
	if !ok || typ != "object" {
		t.Error("Expected type 'object'")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	// Check required properties
	required, ok := params["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "content" {
		t.Error("Expected 'content' to be required")
	}

	// Check content property
	contentProp, ok := props["content"].(map[string]any)
	if !ok {
		t.Error("Expected 'content' property")
	}
	if contentProp["type"] != "string" {
		t.Error("Expected content type to be 'string'")
	}

	// Check channel property (optional)
	channelProp, ok := props["channel"].(map[string]any)
	if !ok {
		t.Error("Expected 'channel' property")
	}
	if channelProp["type"] != "string" {
		t.Error("Expected channel type to be 'string'")
	}

	// Check chat_id property (optional)
	chatIDProp, ok := props["chat_id"].(map[string]any)
	if !ok {
		t.Error("Expected 'chat_id' property")
	}
	if chatIDProp["type"] != "string" {
		t.Error("Expected chat_id type to be 'string'")
	}

	// Check reply_to_message_id property (optional)
	replyToProp, ok := props["reply_to_message_id"].(map[string]any)
	if !ok {
		t.Error("Expected 'reply_to_message_id' property")
	}
	if replyToProp["type"] != "string" {
		t.Error("Expected reply_to_message_id type to be 'string'")
	}

	structuredProp, ok := props["structured"].(map[string]any)
	if !ok {
		t.Fatal("Expected 'structured' property")
	}
	if _, ok := structuredProp["description"].(string); !ok {
		t.Fatal("Expected structured description")
	}
	// structured must use oneOf to enumerate part types as machine-readable JSON Schema
	oneOf, ok := structuredProp["oneOf"].([]any)
	if !ok || len(oneOf) == 0 {
		t.Fatal("Expected structured.oneOf")
	}
	// find the single-part schema (object with oneOf) and verify canonical types are present
	foundTypes := map[string]bool{}
	var checkSchema func(v any)
	checkSchema = func(v any) {
		switch vt := v.(type) {
		case map[string]any:
			if c, ok := vt["const"].(string); ok {
				foundTypes[c] = true
			}
			if e, ok := vt["enum"].([]string); ok {
				for _, s := range e {
					foundTypes[s] = true
				}
			}
			// also scan description strings for kind hints (e.g. "'form'|'progress'|...")
			if d, ok := vt["description"].(string); ok {
				for _, kw := range []string{"form", "progress", "todo", "alert", "options", "card"} {
					if strings.Contains(d, kw) {
						foundTypes[kw] = true
					}
				}
			}
			for _, val := range vt {
				checkSchema(val)
			}
		case []any:
			for _, item := range vt {
				checkSchema(item)
			}
		}
	}
	for _, item := range oneOf {
		checkSchema(item)
	}
	for _, kind := range []string{"options", "card", "form", "progress", "todo", "alert"} {
		if !foundTypes[kind] {
			t.Errorf("structured oneOf schema missing type/kind %q", kind)
		}
	}
}

func TestMessageTool_Execute_WithReplyToMessageID(t *testing.T) {
	tool := NewMessageTool()

	var sentReplyTo string
	tool.SetSendCallback(func(ctx context.Context, channel, chatID, content, replyToMessageID string) error {
		sentReplyTo = replyToMessageID
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	args := map[string]any{
		"content":             "Reply test",
		"reply_to_message_id": "msg-123",
	}

	result := tool.Execute(ctx, args)
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if sentReplyTo != "msg-123" {
		t.Fatalf("expected reply_to_message_id msg-123, got %q", sentReplyTo)
	}
}

func TestMessageTool_Execute_PropagatesTurnSessionMetadata(t *testing.T) {
	tool := NewMessageTool()

	var gotAgentID, gotSessionKey string
	var gotScope *session.SessionScope
	tool.SetSendCallback(func(ctx context.Context, channel, chatID, content, replyToMessageID string) error {
		gotAgentID = ToolAgentID(ctx)
		gotSessionKey = ToolSessionKey(ctx)
		gotScope = ToolSessionScope(ctx)
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	ctx = WithToolSessionContext(ctx, "main", "sk_v1_tool", &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "main",
		Channel:    "telegram",
		Dimensions: []string{"chat"},
		Values: map[string]string{
			"chat": "direct:test-chat-id",
		},
	})

	result := tool.Execute(ctx, map[string]any{"content": "Hello, world!"})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	if gotAgentID != "main" {
		t.Fatalf("ToolAgentID() = %q, want main", gotAgentID)
	}
	if gotSessionKey != "sk_v1_tool" {
		t.Fatalf("ToolSessionKey() = %q, want sk_v1_tool", gotSessionKey)
	}
	if gotScope == nil || gotScope.Values["chat"] != "direct:test-chat-id" {
		t.Fatalf("ToolSessionScope() = %+v, want chat scope", gotScope)
	}
}

func TestMessageTool_Execute_RejectsLegacyOptionsArgs(t *testing.T) {
	tool := NewMessageTool()
	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	result := tool.Execute(ctx, map[string]any{
		"content": "Choose one:",
		"options": []any{
			map[string]any{"label": "A", "value": "alpha"},
		},
	})
	if !result.IsError {
		t.Fatal("expected options on message tool to fail")
	}
	if result.ForLLM != "message does not accept top-level options; use structured.type='options'" {
		t.Fatalf("ForLLM = %q, want message does not accept top-level options; use structured.type='options'", result.ForLLM)
	}
}

func TestMessageTool_Execute_WithStructuredOptionsPayload(t *testing.T) {
	tool := NewMessageTool()

	var gotStructured any
	tool.SetStructuredSendCallback(func(ctx context.Context, channel, chatID, content, replyToMessageID string, structured any) error {
		gotStructured = structured
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	result := tool.Execute(ctx, map[string]any{
		"content": "Choose one:",
		"structured": map[string]any{
			"type":        "options",
			"mode":        "multiple",
			"allowCustom": true,
			"submitLabel": "Confirm",
			"options": []any{
				map[string]any{"label": "A", "value": "alpha"},
			},
		},
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	structuredMap, ok := gotStructured.(map[string]any)
	if !ok {
		t.Fatal("expected structured payload")
	}
	if structuredMap["type"] != "options" {
		t.Fatalf("structured type = %v, want options", structuredMap["type"])
	}
	items, ok := structuredMap["options"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("structured options = %+v, want 1 option", structuredMap["options"])
	}
	first, ok := items[0].(map[string]any)
	if !ok || first["label"] != "A" || first["value"] != "alpha" {
		t.Fatalf("first option = %+v, want normalized option", items[0])
	}
	if structuredMap["mode"] != "multiple" {
		t.Fatalf("structured mode = %v, want multiple", structuredMap["mode"])
	}
	if structuredMap["allowCustom"] != true {
		t.Fatalf("structured allowCustom = %v, want true", structuredMap["allowCustom"])
	}
	if structuredMap["submitLabel"] != "Confirm" {
		t.Fatalf("structured submitLabel = %v, want Confirm", structuredMap["submitLabel"])
	}
}

func TestMessageTool_Execute_InvalidStructuredOptionsPayload(t *testing.T) {
	tool := NewMessageTool()
	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	result := tool.Execute(ctx, map[string]any{
		"content": "Choose one:",
		"structured": map[string]any{
			"type": "options",
		},
	})
	if !result.IsError {
		t.Fatal("expected invalid structured options payload to fail")
	}
	if result.ForLLM != "structured.options must be an array when structured.type='options'" {
		t.Fatalf("ForLLM = %q, want structured.options must be an array when structured.type='options'", result.ForLLM)
	}
}

func TestMessageTool_Execute_WithStructuredCardPayload(t *testing.T) {
	tool := NewMessageTool()

	var gotStructured any
	tool.SetStructuredSendCallback(func(ctx context.Context, channel, chatID, content, replyToMessageID string, structured any) error {
		gotStructured = structured
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	result := tool.Execute(ctx, map[string]any{
		"content": "审批卡片摘要",
		"structured": map[string]any{
			"type":  "card",
			"kind":  "custom/approval-card",
			"title": "待审批",
			"blocks": []any{
				map[string]any{"type": "text", "text": "张三提交了请假申请"},
			},
		},
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	structuredMap, ok := gotStructured.(map[string]any)
	if !ok {
		t.Fatal("expected structured payload")
	}
	if structuredMap["type"] != "card" {
		t.Fatalf("structured type = %v, want card", structuredMap["type"])
	}
	if structuredMap["kind"] != "custom/approval-card" {
		t.Fatalf("structured kind = %v, want custom/approval-card", structuredMap["kind"])
	}
}

func TestMessageTool_Execute_WithStructuredListPayload(t *testing.T) {
	tool := NewMessageTool()

	var gotStructured any
	tool.SetStructuredSendCallback(func(ctx context.Context, channel, chatID, content, replyToMessageID string, structured any) error {
		gotStructured = structured
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	result := tool.Execute(ctx, map[string]any{
		"content": "Mixed structured payload",
		"structured": []any{
			map[string]any{"type": "progress", "title": "Syncing", "status": "running"},
			map[string]any{"type": "alert", "level": "info", "content": "Waiting"},
		},
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	items, ok := gotStructured.([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("structured payload = %#v, want 2-item list", gotStructured)
	}
	progress, ok := items[0].(map[string]any)
	if !ok || progress["type"] != "card" || progress["kind"] != "progress" {
		t.Fatalf("first structured item = %#v, want card kind progress", items[0])
	}
	alert, ok := items[1].(map[string]any)
	if !ok || alert["type"] != "card" || alert["kind"] != "alert" {
		t.Fatalf("second structured item = %#v, want card kind alert", items[1])
	}
}

func TestMessageTool_Execute_NormalizesStructuredAliases(t *testing.T) {
	tool := NewMessageTool()

	var gotStructured any
	tool.SetStructuredSendCallback(func(ctx context.Context, channel, chatID, content, replyToMessageID string, structured any) error {
		gotStructured = structured
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	result := tool.Execute(ctx, map[string]any{
		"content": "Mixed structured payload",
		"structured": []any{
			map[string]any{"type": "progress", "message": "Working", "status": "running"},
			map[string]any{"type": "todo", "items": []any{map[string]any{"label": "Review", "done": true}}},
			map[string]any{"type": "alert", "severity": "info", "message": "Heads up"},
		},
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	items, ok := gotStructured.([]any)
	if !ok || len(items) != 3 {
		t.Fatalf("structured payload = %#v, want 3-item list", gotStructured)
	}

	progress, ok := items[0].(map[string]any)
	if !ok || progress["type"] != "card" || progress["kind"] != "progress" || progress["content"] != "Working" {
		t.Fatalf("progress = %#v, want canonical card progress payload", items[0])
	}
	todo, ok := items[1].(map[string]any)
	if !ok || todo["type"] != "card" || todo["kind"] != "todo" {
		t.Fatalf("todo = %#v, want canonical card todo payload", items[1])
	}
	todoItems, ok := todo["items"].([]any)
	if !ok || len(todoItems) != 1 {
		t.Fatalf("todo items = %#v, want 1 item", todo["items"])
	}
	firstTodo, ok := todoItems[0].(map[string]any)
	if !ok || firstTodo["title"] != "Review" || firstTodo["status"] != "completed" {
		t.Fatalf("first todo = %#v, want normalized title/status", todoItems[0])
	}
	alert, ok := items[2].(map[string]any)
	if !ok || alert["type"] != "card" || alert["kind"] != "alert" || alert["level"] != "info" || alert["content"] != "Heads up" {
		t.Fatalf("alert = %#v, want canonical card alert payload", items[2])
	}
}

func TestMessageTool_Execute_CanonicalizesStructuredFormAlias(t *testing.T) {
	tool := NewMessageTool()

	var gotStructured any
	tool.SetStructuredSendCallback(func(ctx context.Context, channel, chatID, content, replyToMessageID string, structured any) error {
		gotStructured = structured
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	result := tool.Execute(ctx, map[string]any{
		"content": "填写反馈",
		"structured": map[string]any{
			"type":    "form",
			"title":   "反馈表",
			"content": "请补充你的意见",
			"fields": []any{
				map[string]any{"name": "feedback", "label": "你的反馈"},
			},
		},
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.ForLLM)
	}
	structuredMap, ok := gotStructured.(map[string]any)
	if !ok {
		t.Fatal("expected structured payload")
	}
	if structuredMap["type"] != "card" || structuredMap["kind"] != "form" {
		t.Fatalf("structured payload = %#v, want card kind form", structuredMap)
	}
	fields, ok := structuredMap["fields"].([]any)
	if !ok || len(fields) != 1 {
		t.Fatalf("structured fields = %#v, want 1 field", structuredMap["fields"])
	}
}

func TestMessageTool_Execute_InvalidStructuredCardPayload(t *testing.T) {
	tool := NewMessageTool()
	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	result := tool.Execute(ctx, map[string]any{
		"content": "bad card",
		"structured": map[string]any{
			"type":  "card",
			"title": "Card without body",
		},
	})
	if !result.IsError {
		t.Fatal("expected invalid structured card payload to fail")
	}
	if result.ForLLM != "structured.blocks or structured.actions must be a non-empty array when structured.type='card'" {
		t.Fatalf("ForLLM = %q, want repairable card error", result.ForLLM)
	}
}

func TestMessageTool_Execute_InvalidStructuredFormPayload(t *testing.T) {
	tool := NewMessageTool()
	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	result := tool.Execute(ctx, map[string]any{
		"content": "bad form",
		"structured": map[string]any{
			"type":  "form",
			"title": "Missing fields",
		},
	})
	if !result.IsError {
		t.Fatal("expected invalid structured form payload to fail")
	}
	if result.ForLLM != "structured.fields must be an array when structured.type='form'" {
		t.Fatalf("ForLLM = %q, want repairable form error", result.ForLLM)
	}
}

func TestMessageTool_Execute_InvalidStructuredProgressPayload(t *testing.T) {
	tool := NewMessageTool()
	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	result := tool.Execute(ctx, map[string]any{
		"content": "bad progress",
		"structured": map[string]any{
			"type":  "progress",
			"title": "Syncing",
		},
	})
	if !result.IsError {
		t.Fatal("expected invalid structured progress payload to fail")
	}
	if result.ForLLM != "structured.status is required when structured.type='progress'" {
		t.Fatalf("ForLLM = %q, want repairable progress error", result.ForLLM)
	}
}

func TestMessageTool_Execute_InvalidStructuredTodoPayload(t *testing.T) {
	tool := NewMessageTool()
	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	result := tool.Execute(ctx, map[string]any{
		"content": "bad todo",
		"structured": map[string]any{
			"type":  "todo",
			"items": []any{map[string]any{"status": "waiting"}},
		},
	})
	if !result.IsError {
		t.Fatal("expected invalid structured todo payload to fail")
	}
	if result.ForLLM != "structured.items[0].title is required when structured.type='todo'" {
		t.Fatalf("ForLLM = %q, want repairable todo error", result.ForLLM)
	}
}

func TestMessageTool_Execute_InvalidStructuredAlertPayload(t *testing.T) {
	tool := NewMessageTool()
	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	result := tool.Execute(ctx, map[string]any{
		"content": "bad alert",
		"structured": map[string]any{
			"type":     "alert",
			"severity": "info",
		},
	})
	if !result.IsError {
		t.Fatal("expected invalid structured alert payload to fail")
	}
	if result.ForLLM != "structured.content is required when structured.type='alert'" {
		t.Fatalf("ForLLM = %q, want repairable alert error", result.ForLLM)
	}
}

func TestMessageTool_Execute_InvalidStructuredListPayload(t *testing.T) {
	tool := NewMessageTool()
	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	result := tool.Execute(ctx, map[string]any{
		"content":    "bad",
		"structured": []any{map[string]any{"title": "missing type"}},
	})
	if !result.IsError {
		t.Fatal("expected invalid structured list payload to fail")
	}
	if result.ForLLM != "structured[0].type is required" {
		t.Fatalf("ForLLM = %q, want structured[0].type is required", result.ForLLM)
	}
}

func TestMessageTool_Execute_InvalidStructuredPayload(t *testing.T) {
	tool := NewMessageTool()
	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	result := tool.Execute(ctx, map[string]any{
		"content":    "bad",
		"structured": map[string]any{},
	})
	if !result.IsError {
		t.Fatal("expected invalid structured payload to fail")
	}
	if result.ForLLM != "structured.type is required" {
		t.Fatalf("ForLLM = %q, want structured.type is required", result.ForLLM)
	}
}

func TestMessageTool_Execute_UnknownTypePassesThrough(t *testing.T) {
	tool := NewMessageTool()

	var gotStructured any
	tool.SetStructuredSendCallback(func(ctx context.Context, channel, chatID, content, replyToMessageID string, structured any) error {
		gotStructured = structured
		return nil
	})

	ctx := WithToolContext(context.Background(), "test-channel", "test-chat-id")
	result := tool.Execute(ctx, map[string]any{
		"content": "Custom component",
		"structured": map[string]any{
			"type":     "approval",
			"title":    "Deploy to prod",
			"approver": "alice",
		},
	})
	if result.IsError {
		t.Fatalf("expected unknown type to pass through, got error: %s", result.ForLLM)
	}
	structuredMap, ok := gotStructured.(map[string]any)
	if !ok {
		t.Fatal("expected structured payload")
	}
	if structuredMap["type"] != "approval" {
		t.Fatalf("structured type = %v, want approval", structuredMap["type"])
	}
	if structuredMap["approver"] != "alice" {
		t.Fatalf("structured approver = %v, want alice", structuredMap["approver"])
	}
}
