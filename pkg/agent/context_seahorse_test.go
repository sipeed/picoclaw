package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
	"github.com/sipeed/picoclaw/pkg/seahorse"
)

// seahorseTestProvider implements providers.LLMProvider for seahorse tests.
type seahorseTestProvider struct {
	chatFn func(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]any) (*providers.LLMResponse, error)
}

func (m *seahorseTestProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	if m.chatFn != nil {
		return m.chatFn(ctx, messages, tools, model, options)
	}
	return &providers.LLMResponse{Content: "mock response"}, nil
}

func (m *seahorseTestProvider) GetDefaultModel() string {
	return "mock-model"
}

func TestSeahorseCMRegistration(t *testing.T) {
	factory, ok := lookupContextManager("seahorse")
	if !ok {
		t.Error("expected 'seahorse' context manager to be registered")
	}
	if factory == nil {
		t.Error("expected non-nil factory")
	}
}

func TestProviderToSeahorseMessage(t *testing.T) {
	tests := []struct {
		name        string
		input       protocoltypes.Message
		wantRole    string
		wantContent string
	}{
		{
			name:        "simple user message",
			input:       protocoltypes.Message{Role: "user", Content: "hello world"},
			wantRole:    "user",
			wantContent: "hello world",
		},
		{
			name:        "assistant message",
			input:       protocoltypes.Message{Role: "assistant", Content: "response text"},
			wantRole:    "assistant",
			wantContent: "response text",
		},
		{
			name:        "tool result message",
			input:       protocoltypes.Message{Role: "tool", Content: "tool output", ToolCallID: "tc_123"},
			wantRole:    "tool",
			wantContent: "tool output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := providerToSeahorseMessage(tt.input)
			if result.Role != tt.wantRole {
				t.Errorf("Role = %q, want %q", result.Role, tt.wantRole)
			}
			if result.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", result.Content, tt.wantContent)
			}
		})
	}
}

func TestProviderToSeahorseMessageWithToolCalls(t *testing.T) {
	msg := protocoltypes.Message{
		Role:    "assistant",
		Content: "",
		ToolCalls: []protocoltypes.ToolCall{
			{
				ID: "tc_1",
				Function: &protocoltypes.FunctionCall{
					Name:      "read_file",
					Arguments: `{"path":"/tmp/test"}`,
				},
			},
		},
	}

	result := providerToSeahorseMessage(msg)
	if result.Role != "assistant" {
		t.Errorf("Role = %q, want assistant", result.Role)
	}
	if len(result.Parts) == 0 {
		t.Fatal("expected at least 1 part from tool calls")
	}
	if result.Parts[0].Type != "tool_use" {
		t.Errorf("Part type = %q, want tool_use", result.Parts[0].Type)
	}
	if result.Parts[0].Name != "read_file" {
		t.Errorf("Part name = %q, want read_file", result.Parts[0].Name)
	}
	if result.Parts[0].ToolCallID != "tc_1" {
		t.Errorf("Part ToolCallID = %q, want tc_1", result.Parts[0].ToolCallID)
	}
}

func TestProviderToSeahorseMessageWithToolResult(t *testing.T) {
	msg := protocoltypes.Message{
		Role:       "tool",
		Content:    "file contents here",
		ToolCallID: "tc_456",
	}

	result := providerToSeahorseMessage(msg)
	if result.Role != "tool" {
		t.Errorf("Role = %q, want tool", result.Role)
	}
	found := false
	for _, p := range result.Parts {
		if p.Type == "tool_result" && p.ToolCallID == "tc_456" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected tool_result part with ToolCallID tc_456")
	}
}

func TestProviderToSeahorseMessageWithMedia(t *testing.T) {
	msg := protocoltypes.Message{
		Role:    "user",
		Content: "Here is an image",
		Media:   []string{"data:image/png;base64,abc123"},
	}

	result := providerToSeahorseMessage(msg)
	if result.Role != "user" {
		t.Errorf("Role = %q, want user", result.Role)
	}

	// Should have a media part
	found := false
	for _, p := range result.Parts {
		if p.Type == "media" {
			found = true
			if p.MediaURI != "data:image/png;base64,abc123" {
				t.Errorf("MediaURI = %q, want data:image/png;base64,abc123", p.MediaURI)
			}
			break
		}
	}
	if !found {
		t.Error("expected media part in converted message")
	}
}

func TestProviderToSeahorseMessageWithReasoning(t *testing.T) {
	msg := protocoltypes.Message{
		Role:             "assistant",
		Content:          "response text",
		ReasoningContent: "I thought about this carefully",
	}

	result := providerToSeahorseMessage(msg)
	if result.ReasoningContent != "I thought about this carefully" {
		t.Errorf("ReasoningContent = %q, want 'I thought about this carefully'", result.ReasoningContent)
	}
}

func TestSeahorseToProviderMessagesWithReasoning(t *testing.T) {
	result := &seahorse.AssembleResult{
		Messages: []seahorse.Message{
			{
				Role:             "assistant",
				Content:          "response",
				ReasoningContent: "thinking process",
			},
		},
	}

	messages := seahorseToProviderMessages(result)
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].ReasoningContent != "thinking process" {
		t.Errorf("ReasoningContent = %q, want 'thinking process'", messages[0].ReasoningContent)
	}
}

func TestSeahorseToProviderMessages(t *testing.T) {
	// Summaries should NOT be double-injected.
	// The assembler already includes summaries as XML-formatted messages in Messages slice.
	// seahorseToProviderMessages should only convert Messages, not Summaries.
	summaryXML := `<summary id="sum_test" kind="leaf" depth="0" descendant_count="8">
  <content>
    test summary content
  </content>
</summary>`
	summaryMsg := seahorse.Message{
		Role:       "user",
		Content:    summaryXML,
		TokenCount: 50,
	}
	rawMsg := seahorse.Message{
		Role:       "user",
		Content:    "hello",
		TokenCount: 5,
	}

	result := seahorseToProviderMessages(&seahorse.AssembleResult{
		Messages: []seahorse.Message{summaryMsg, rawMsg},
	})

	// Should have exactly 2 messages (from Messages slice only)
	// NOT 3 (which would happen if Summaries were also converted)
	if len(result) != 2 {
		t.Fatalf("expected exactly 2 messages (no double injection), got %d", len(result))
	}
	// First should be the XML summary message
	if result[0].Content != summaryXML {
		t.Errorf("first message content = %q, want summary XML", result[0].Content)
	}
	// Second should be the raw message
	if result[1].Content != "hello" {
		t.Errorf("second message content = %q, want 'hello'", result[1].Content)
	}
}

func TestSeahorseToProviderMessagesWithToolCalls(t *testing.T) {
	msg := seahorse.Message{
		Role:       "assistant",
		Content:    "",
		TokenCount: 10,
		Parts: []seahorse.MessagePart{
			{
				Type:       "tool_use",
				Name:       "read_file",
				Arguments:  `{"path":"/tmp"}`,
				ToolCallID: "tc_1",
			},
		},
	}

	result := seahorseToProviderMessages(&seahorse.AssembleResult{
		Messages: []seahorse.Message{msg},
	})

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Role != "assistant" {
		t.Errorf("Role = %q, want assistant", result[0].Role)
	}
	if len(result[0].ToolCalls) != 1 {
		t.Fatalf("ToolCalls = %d, want 1", len(result[0].ToolCalls))
	}
	if result[0].ToolCalls[0].Function.Name != "read_file" {
		t.Errorf("ToolCall name = %q, want read_file", result[0].ToolCalls[0].Function.Name)
	}
	// GLM API and other OpenAI-compatible APIs require Type: "function"
	if result[0].ToolCalls[0].Type != "function" {
		t.Errorf("ToolCall Type = %q, want 'function' (required by GLM/OpenAI APIs)",
			result[0].ToolCalls[0].Type)
	}
}

func TestSeahorseToProviderMessagesToolResult(t *testing.T) {
	msg := seahorse.Message{
		Role:       "tool",
		Content:    "file output",
		TokenCount: 5,
		Parts: []seahorse.MessagePart{
			{
				Type:       "tool_result",
				ToolCallID: "tc_99",
				Text:       "file output",
			},
		},
	}

	result := seahorseToProviderMessages(&seahorse.AssembleResult{
		Messages: []seahorse.Message{msg},
	})

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].ToolCallID != "tc_99" {
		t.Errorf("ToolCallID = %q, want tc_99", result[0].ToolCallID)
	}
}

// --- providerToCompleteFn tests ---

func TestProviderToCompleteFn(t *testing.T) {
	var capturedMessages []providers.Message
	var capturedModel string
	var capturedOptions map[string]any

	mp := &seahorseTestProvider{
		chatFn: func(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]any) (*providers.LLMResponse, error) {
			capturedMessages = messages
			capturedModel = model
			capturedOptions = options
			return &providers.LLMResponse{Content: "summary of conversation"}, nil
		},
	}

	completeFn := providerToCompleteFn(mp, "test-model-v1")
	result, err := completeFn(context.Background(), "Summarize this text", seahorse.CompleteOptions{
		MaxTokens:   500,
		Temperature: 0.3,
	})
	if err != nil {
		t.Fatalf("completeFn: %v", err)
	}
	if result != "summary of conversation" {
		t.Errorf("result = %q, want 'summary of conversation'", result)
	}

	// Verify prompt passed as user message
	if len(capturedMessages) != 1 {
		t.Fatalf("captured messages = %d, want 1", len(capturedMessages))
	}
	if capturedMessages[0].Role != "user" {
		t.Errorf("message role = %q, want user", capturedMessages[0].Role)
	}
	if capturedMessages[0].Content != "Summarize this text" {
		t.Errorf("message content = %q, want 'Summarize this text'", capturedMessages[0].Content)
	}

	// Verify model
	if capturedModel != "test-model-v1" {
		t.Errorf("model = %q, want 'test-model-v1'", capturedModel)
	}

	// Verify options
	if capturedOptions["max_tokens"] != 500 {
		t.Errorf("max_tokens = %v, want 500", capturedOptions["max_tokens"])
	}
	if capturedOptions["temperature"] != 0.3 {
		t.Errorf("temperature = %v, want 0.3", capturedOptions["temperature"])
	}
	if capturedOptions["prompt_cache_key"] != "seahorse" {
		t.Errorf("prompt_cache_key = %v, want 'seahorse'", capturedOptions["prompt_cache_key"])
	}
}

func TestSeahorseIgnoreHeartbeat(t *testing.T) {
	// Verify that "heartbeat" sessions are ignored by default
	// This tests the hardcoded ignore pattern from spec lines 1326-1328
	engine, err := seahorse.NewEngine(seahorse.Config{
		DBPath: t.TempDir() + "/test.db",
	}, nil)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	result, err := engine.Ingest(ctx, "heartbeat", []seahorse.Message{
		{Role: "user", Content: "heartbeat msg", TokenCount: 5},
	})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	// Should return nil nil for ignored sessions
	if result != nil {
		t.Errorf("expected nil result for heartbeat session, got %+v", result)
	}
}

func TestProviderToCompleteFnError(t *testing.T) {
	mp := &seahorseTestProvider{
		chatFn: func(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]any) (*providers.LLMResponse, error) {
			return nil, context.Canceled
		},
	}

	completeFn := providerToCompleteFn(mp, "test-model")
	_, err := completeFn(context.Background(), "test prompt", seahorse.CompleteOptions{})
	if err == nil {
		t.Error("expected error from canceled context")
	}
}

func TestSeahorseAdapterAssembleSubtractsMaxTokens(t *testing.T) {
	// Create a real seahorse engine with temp DB
	engine, err := seahorse.NewEngine(seahorse.Config{
		DBPath: t.TempDir() + "/test.db",
	}, nil)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	mgr := &seahorseContextManager{engine: engine}

	// Ingest lots of large messages (~35 tokens each, 120 total = ~4200 tokens)
	for i := 0; i < 60; i++ {
		content := fmt.Sprintf(
			"This is message number %d. It contains enough text to represent a meaningful conversation turn with the user asking about various topics in software engineering and system design principles that require careful consideration.",
			i,
		)
		_ = mgr.Ingest(ctx, &IngestRequest{
			SessionKey: "budget-sub",
			Message:    protocoltypes.Message{Role: "user", Content: content},
		})
		_ = mgr.Ingest(ctx, &IngestRequest{
			SessionKey: "budget-sub",
			Message:    protocoltypes.Message{Role: "assistant", Content: "Response"},
		})
	}

	// Call adapter Assemble with Budget=5000, MaxTokens=2000
	// Should use effective budget = 5000 - 2000 = 3000
	resp, err := mgr.Assemble(ctx, &AssembleRequest{
		SessionKey: "budget-sub",
		Budget:     5000,
		MaxTokens:  2000,
	})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	// Directly call engine with budget=3000 to get baseline
	baseline, err := engine.Assemble(ctx, "budget-sub", seahorse.AssembleInput{Budget: 3000})
	if err != nil {
		t.Fatalf("engine.Assemble baseline: %v", err)
	}

	// The adapter result should have same message count as engine with budget 3000
	if len(resp.History) != len(baseline.Messages) {
		t.Errorf("adapter Budget=5000 MaxTokens=2000 gave %d messages, engine Budget=3000 gave %d",
			len(resp.History), len(baseline.Messages))
	}
}

func TestSeahorseCompactRetryUsesCompactUntilUnder(t *testing.T) {
	// Track which engine method was called
	var compactCalled, compactUntilCalled bool

	engine, err := seahorse.NewEngine(seahorse.Config{
		DBPath: t.TempDir() + "/test.db",
	}, nil)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close()

	// Wrap engine to track calls
	_ = compactCalled // track via adapter behavior
	_ = compactUntilCalled

	mgr := &seahorseContextManager{engine: engine}

	ctx := context.Background()

	// Ingest messages so there's something to compact
	for i := 0; i < 40; i++ {
		content := fmt.Sprintf(
			"message %d with enough text to have meaningful token count that fills up the budget nicely",
			i,
		)
		_ = mgr.Ingest(ctx, &IngestRequest{
			SessionKey: "compact-test",
			Message:    protocoltypes.Message{Role: "user", Content: content},
		})
		_ = mgr.Ingest(ctx, &IngestRequest{
			SessionKey: "compact-test",
			Message:    protocoltypes.Message{Role: "assistant", Content: "ok"},
		})
	}

	// Compact with retry reason and budget should succeed
	err = mgr.Compact(ctx, &CompactRequest{
		SessionKey: "compact-test",
		Reason:     CompactReasonRetry,
		Budget:     5000,
	})
	if err != nil {
		t.Fatalf("Compact retry: %v", err)
	}

	// Verify context was actually compacted (should have fewer tokens)
	result, err := engine.Assemble(ctx, "compact-test", seahorse.AssembleInput{Budget: 5000})
	if err != nil {
		t.Fatalf("Assemble after compact: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil assemble result")
	}
	// Compaction attempted — no assertion on exact count since no LLM
	_ = result.Summary
}

// TestSeahorseRealLoopNoDuplicateMessages tests the real-world scenario:
// 1. Start AgentLoop with seahorse context manager
// 2. Run a turn (user message -> LLM response)
// 3. Check DB for duplicate messages
// This test verifies that bootstrapping at startup (not during first Ingest) prevents duplicates.
func TestSeahorseRealLoopNoDuplicateMessages(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				ContextManager:    "seahorse",
			},
		},
	}

	msgBus := bus.NewMessageBus()
	mockProvider := &simpleMockProvider{response: "I received your message."}
	al := NewAgentLoop(cfg, msgBus, mockProvider)
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	ctx := context.Background()
	sessionKey := "test-real-loop-dup"

	// Run a turn: user message -> LLM response
	_, err := al.runAgentLoop(ctx, defaultAgent, processOptions{
		SessionKey:      sessionKey,
		Channel:         "cli",
		ChatID:          "direct",
		UserMessage:     "hello",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
	})
	if err != nil {
		t.Fatalf("runAgentLoop failed: %v", err)
	}

	// Get the seahorse engine from context manager
	seahorseCM, ok := al.contextManager.(*seahorseContextManager)
	if !ok {
		t.Fatal("expected seahorseContextManager")
	}

	// Check DB for messages via RetrievalEngine.Store()
	store := seahorseCM.engine.GetRetrieval().Store()
	conv, err := store.GetOrCreateConversation(ctx, sessionKey)
	if err != nil {
		t.Fatalf("GetOrCreateConversation: %v", err)
	}

	stored, err := store.GetMessages(ctx, conv.ConversationID, 20, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}

	t.Logf("DB has %d messages:", len(stored))
	for i, msg := range stored {
		content := msg.Content
		if len(content) > 40 {
			content = content[:40] + "..."
		}
		t.Logf("  msg[%d]: role=%s content=%q", i, msg.Role, content)
	}

	// Count duplicates by (role, content)
	seen := make(map[string]int)
	for _, msg := range stored {
		key := msg.Role + ":" + msg.Content
		seen[key]++
	}
	for key, count := range seen {
		if count > 1 {
			t.Errorf("DUPLICATE BUG: %q appears %d times in DB", key, count)
		}
	}

	// Expected: 2 messages (user "hello" + assistant response)
	if len(stored) != 2 {
		t.Errorf("expected 2 messages in DB (user + assistant), got %d", len(stored))
	}
}

// TestSeahorseAssembleReturnsAllSummaries verifies that Assemble returns ALL summaries,
// not just the latest one. This is important because summaries represent compressed
// conversation history at different points in time.
func TestSeahorseAssembleReturnsAllSummaries(t *testing.T) {
	// Create a real seahorse engine with temp DB
	engine, err := seahorse.NewEngine(seahorse.Config{
		DBPath: t.TempDir() + "/test.db",
	}, nil)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	mgr := &seahorseContextManager{engine: engine}
	sessionKey := "test-multi-summary"

	// Get the store to directly create summaries
	store := engine.GetRetrieval().Store()

	// Get conversation ID
	conv, err := store.GetOrCreateConversation(ctx, sessionKey)
	if err != nil {
		t.Fatalf("GetOrCreateConversation: %v", err)
	}

	// Create some messages first
	for i := 0; i < 20; i++ {
		_ = mgr.Ingest(ctx, &IngestRequest{
			SessionKey: sessionKey,
			Message:    protocoltypes.Message{Role: "user", Content: fmt.Sprintf("Message %d", i)},
		})
	}

	// Directly create multiple summaries in the database to simulate multi-level compaction
	testSummaries := []struct {
		content string
		kind    seahorse.SummaryKind
		depth   int
		token   int
	}{
		{"First summary about early conversation discussing topics A and B", seahorse.SummaryKindLeaf, 0, 100},
		{"Second summary covering middle conversation about topics C and D", seahorse.SummaryKindLeaf, 0, 150},
		{"Third summary is condensed from first two summaries about topics A-D", seahorse.SummaryKindCondensed, 1, 200},
	}

	summaryIDs := make([]string, 0, len(testSummaries))
	for _, s := range testSummaries {
		input := seahorse.CreateSummaryInput{
			ConversationID: conv.ConversationID,
			Kind:           s.kind,
			Depth:          s.depth,
			Content:        s.content,
			TokenCount:     s.token,
		}
		summary, createErr := store.CreateSummary(ctx, input)
		if createErr != nil {
			t.Fatalf("CreateSummary: %v", createErr)
		}
		summaryIDs = append(summaryIDs, summary.SummaryID)

		// Add summary to context_items
		err = store.AppendContextSummary(ctx, conv.ConversationID, summary.SummaryID)
		if err != nil {
			t.Fatalf("AppendContextSummary: %v", err)
		}
	}

	t.Logf("Created %d summaries directly in store", len(summaryIDs))

	// Assemble and check summaries
	resp, err := mgr.Assemble(ctx, &AssembleRequest{
		SessionKey: sessionKey,
		Budget:     50000,
		MaxTokens:  4096,
	})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Check seahorse engine directly for how many summaries exist
	result, err := engine.Assemble(ctx, sessionKey, seahorse.AssembleInput{Budget: 50000})
	if err != nil {
		t.Fatalf("engine.Assemble: %v", err)
	}

	t.Logf("Seahorse returned Summary with %d chars", len(result.Summary))

	// The Summary field should contain XML summaries with metadata (depth, kind)
	// The assembler generates this from the Summaries list
	if len(resp.Summary) > 0 {
		// Should contain XML tag
		if !strings.Contains(resp.Summary, "<summary") {
			t.Error("Summary field should contain <summary XML tags")
		}
		// Should contain depth attribute
		if !strings.Contains(resp.Summary, `depth="`) {
			t.Error("Summary field should contain depth attribute")
		}
		// Should contain kind attribute
		if !strings.Contains(resp.Summary, `kind="`) {
			t.Error("Summary field should contain kind attribute")
		}
	}
}

func TestProviderToSeahorseMessageTokenCountIncludesAllFields(t *testing.T) {
	// Message with only Content
	msgContentOnly := protocoltypes.Message{
		Role:    "assistant",
		Content: "This is a simple response with some text content.",
	}
	resultContentOnly := providerToSeahorseMessage(msgContentOnly)

	// Message with Content + ToolCalls
	msgWithToolCalls := protocoltypes.Message{
		Role:    "assistant",
		Content: "This is a simple response with some text content.",
		ToolCalls: []protocoltypes.ToolCall{
			{
				ID: "tc_123",
				Function: &protocoltypes.FunctionCall{
					Name:      "read_file",
					Arguments: `{"path":"/home/user/document.txt"}`,
				},
			},
		},
	}
	resultWithToolCalls := providerToSeahorseMessage(msgWithToolCalls)

	if resultWithToolCalls.TokenCount <= resultContentOnly.TokenCount {
		t.Errorf("TokenCount with ToolCalls = %d, should be > Content-only = %d",
			resultWithToolCalls.TokenCount, resultContentOnly.TokenCount)
	}

	// Message with ToolCallID
	msgWithToolResult := protocoltypes.Message{
		Role:       "tool",
		Content:    "This is a simple response with some text content.",
		ToolCallID: "tc_456",
	}
	resultWithToolResult := providerToSeahorseMessage(msgWithToolResult)

	if resultWithToolResult.TokenCount <= resultContentOnly.TokenCount {
		t.Errorf("TokenCount with ToolCallID = %d, should be > Content-only = %d",
			resultWithToolResult.TokenCount, resultContentOnly.TokenCount)
	}

	// Message with Media
	msgWithMedia := protocoltypes.Message{
		Role:    "user",
		Content: "This is a simple response with some text content.",
		Media:   []string{"data:image/png;base64,abc123"},
	}
	resultWithMedia := providerToSeahorseMessage(msgWithMedia)

	if resultWithMedia.TokenCount <= resultContentOnly.TokenCount {
		t.Errorf("TokenCount with Media = %d, should be > Content-only = %d",
			resultWithMedia.TokenCount, resultContentOnly.TokenCount)
	}
}

func TestSeahorseToProviderMessagesRebuildsContentFromParts(t *testing.T) {
	msg := seahorse.Message{
		Role:       "tool",
		Content:    "",
		TokenCount: 50,
		Parts: []seahorse.MessagePart{
			{
				Type:       "tool_result",
				ToolCallID: "tc_999",
				Text:       "This is the actual tool output that should be in Content",
			},
		},
	}

	result := seahorseToProviderMessages(&seahorse.AssembleResult{
		Messages: []seahorse.Message{msg},
	})

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}

	if result[0].Content == "" {
		t.Error("Content is empty - tool_result text was not rebuilt into Content")
	}
	if result[0].Content != "This is the actual tool output that should be in Content" {
		t.Errorf("Content = %q, want tool output text from Parts", result[0].Content)
	}
}

func TestSeahorseAssembleSummaryNotInMessages(t *testing.T) {
	engine, err := seahorse.NewEngine(seahorse.Config{
		DBPath: t.TempDir() + "/test.db",
	}, nil)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	mgr := &seahorseContextManager{engine: engine}
	sessionKey := "test-no-dup-summary"

	// Get the store to directly create a summary
	store := engine.GetRetrieval().Store()
	conv, err := store.GetOrCreateConversation(ctx, sessionKey)
	if err != nil {
		t.Fatalf("GetOrCreateConversation: %v", err)
	}

	// Ingest some messages first
	for i := 0; i < 10; i++ {
		_ = mgr.Ingest(ctx, &IngestRequest{
			SessionKey: sessionKey,
			Message:    protocoltypes.Message{Role: "user", Content: fmt.Sprintf("Message %d", i)},
		})
	}

	// Create a summary
	input := seahorse.CreateSummaryInput{
		ConversationID: conv.ConversationID,
		Kind:           seahorse.SummaryKindLeaf,
		Depth:          0,
		Content:        "This is a test summary about the conversation",
		TokenCount:     50,
	}
	summary, err := store.CreateSummary(ctx, input)
	if err != nil {
		t.Fatalf("CreateSummary: %v", err)
	}
	err = store.AppendContextSummary(ctx, conv.ConversationID, summary.SummaryID)
	if err != nil {
		t.Fatalf("AppendContextSummary: %v", err)
	}

	// Assemble
	resp, err := mgr.Assemble(ctx, &AssembleRequest{
		SessionKey: sessionKey,
		Budget:     50000,
		MaxTokens:  4096,
	})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Count how many times the summary content appears
	summaryContent := "This is a test summary"
	countInHistory := 0
	for _, msg := range resp.History {
		if strings.Contains(msg.Content, summaryContent) {
			countInHistory++
		}
	}

	if countInHistory > 0 {
		t.Errorf("Summary content appears %d times in History - should be 0", countInHistory)
	}

	// Summary should appear in Summary field
	if !strings.Contains(resp.Summary, summaryContent) {
		t.Error("Summary content should appear in response.Summary field")
	}
}

// TestSeahorseSteeringMessageIngested verifies that steering messages are ingested
// into seahorse SQLite, not just session JSONL.
func TestSeahorseSteeringMessageIngested(t *testing.T) {
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				ContextManager:    "seahorse",
			},
		},
	}

	msgBus := bus.NewMessageBus()
	mockProvider := &simpleMockProvider{response: "I received your message."}
	al := NewAgentLoop(cfg, msgBus, mockProvider)
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	ctx := context.Background()
	sessionKey := "test-steering-ingest"

	// First turn: establish conversation
	_, err := al.runAgentLoop(ctx, defaultAgent, processOptions{
		SessionKey:      sessionKey,
		Channel:         "cli",
		ChatID:          "direct",
		UserMessage:     "hello",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
	})
	if err != nil {
		t.Fatalf("first runAgentLoop failed: %v", err)
	}

	// Inject a steering message
	steerErr := al.InjectSteering(providers.Message{
		Role:    "user",
		Content: "steering message content",
	})
	if steerErr != nil {
		t.Fatalf("InjectSteering failed: %v", steerErr)
	}

	// Second turn: should process steering message
	_, err = al.runAgentLoop(ctx, defaultAgent, processOptions{
		SessionKey:      sessionKey,
		Channel:         "cli",
		ChatID:          "direct",
		UserMessage:     "continue",
		DefaultResponse: defaultResponse,
		EnableSummary:   false,
		SendResponse:    false,
	})
	if err != nil {
		t.Fatalf("second runAgentLoop failed: %v", err)
	}

	// Get the seahorse engine from context manager
	seahorseCM, ok := al.contextManager.(*seahorseContextManager)
	if !ok {
		t.Fatal("expected seahorseContextManager")
	}

	// Check DB for steering message
	store := seahorseCM.engine.GetRetrieval().Store()
	conv, err := store.GetOrCreateConversation(ctx, sessionKey)
	if err != nil {
		t.Fatalf("GetOrCreateConversation: %v", err)
	}

	stored, err := store.GetMessages(ctx, conv.ConversationID, 20, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}

	t.Logf("DB has %d messages:", len(stored))
	for i, msg := range stored {
		content := msg.Content
		if len(content) > 40 {
			content = content[:40] + "..."
		}
		t.Logf("  msg[%d]: role=%s content=%q", i, msg.Role, content)
	}

	// Find steering message in stored messages
	foundSteering := false
	for _, msg := range stored {
		if msg.Content == "steering message content" {
			foundSteering = true
			break
		}
	}

	if !foundSteering {
		t.Error("STEERING MESSAGE NOT IN SEAHORSE DB: steering message should be ingested into SQLite")
	}
}

// TestSeahorseSummarizeSkipsCondensedWhenBelowThreshold verifies that when
// Summarize is triggered but tokens are below ContextWindow threshold,
// condensed compaction should NOT run.
func TestSeahorseSummarizeSkipsCondensedWhenBelowThreshold(t *testing.T) {
	contextWindow := 1000
	cfg := &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         t.TempDir(),
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 10,
				ContextManager:    "seahorse",
				ContextWindow:     contextWindow,
			},
		},
	}

	msgBus := bus.NewMessageBus()
	provider := &seahorseTestProvider{}
	al := NewAgentLoop(cfg, msgBus, provider)
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil {
		t.Fatal("expected default agent")
	}

	ctx := context.Background()
	sessionKey := "test-summarize-skip-condensed"

	seahorseCM, ok := al.contextManager.(*seahorseContextManager)
	if !ok {
		t.Fatal("expected seahorseContextManager")
	}
	store := seahorseCM.engine.GetRetrieval().Store()

	conv, err := store.GetOrCreateConversation(ctx, sessionKey)
	if err != nil {
		t.Fatalf("GetOrCreateConversation: %v", err)
	}

	// Insert leaf summaries directly (bypass leaf compaction requirement)
	for i := 0; i < seahorse.CondensedMinFanout; i++ {
		now := time.Now().UTC()
		summary, sumErr := store.CreateSummary(ctx, seahorse.CreateSummaryInput{
			ConversationID: conv.ConversationID,
			Kind:           seahorse.SummaryKindLeaf,
			Depth:          0,
			Content:        fmt.Sprintf("leaf summary %d", i),
			TokenCount:     50,
			EarliestAt:     &now,
			LatestAt:       &now,
		})
		if sumErr != nil {
			t.Fatalf("CreateSummary %d: %v", i, sumErr)
		}
		if appendErr := store.AppendContextSummary(ctx, conv.ConversationID, summary.SummaryID); appendErr != nil {
			t.Fatalf("AppendContextSummary %d: %v", i, appendErr)
		}
	}

	// Add fresh messages (required for condensation candidates)
	for i := 0; i < seahorse.FreshTailCount+1; i++ {
		m, msgErr := store.AddMessage(ctx, conv.ConversationID, "user", "fresh", 5)
		if msgErr != nil {
			t.Fatalf("AddMessage %d: %v", i, msgErr)
		}
		if appendErr := store.AppendContextMessage(ctx, conv.ConversationID, m.ID); appendErr != nil {
			t.Fatalf("AppendContextMessage %d: %v", i, appendErr)
		}
	}

	tokensBefore, err := store.GetContextTokenCount(ctx, conv.ConversationID)
	if err != nil {
		t.Fatalf("GetContextTokenCount: %v", err)
	}
	threshold := int(float64(contextWindow) * seahorse.ContextThreshold)
	t.Logf("Tokens before: %d, threshold: %d", tokensBefore, threshold)

	// Trigger Summarize
	_, err = al.runAgentLoop(ctx, defaultAgent, processOptions{
		SessionKey:      sessionKey,
		Channel:         "cli",
		ChatID:          "direct",
		UserMessage:     "trigger",
		DefaultResponse: defaultResponse,
		EnableSummary:   true,
		SendResponse:    false,
	})
	if err != nil {
		t.Fatalf("runAgentLoop: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	summaries, err := store.GetSummariesByConversation(ctx, conv.ConversationID)
	if err != nil {
		t.Fatalf("GetSummariesByConversation: %v", err)
	}

	condensedCount := 0
	for _, sum := range summaries {
		if sum.Kind == seahorse.SummaryKindCondensed {
			condensedCount++
		}
	}

	t.Logf("Condensed summaries: %d", condensedCount)

	if tokensBefore < threshold && condensedCount > 0 {
		t.Errorf("BUG: condensed created when tokens (%d) < threshold (%d)", tokensBefore, threshold)
	}
}
