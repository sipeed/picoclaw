package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// TestAgentMessage tests basic AgentMessage functionality
func TestAgentMessage(t *testing.T) {
	t.Run("NewUserMessage", func(t *testing.T) {
		msg := NewUserMessage("Hello, assistant!")
		if msg.Role != "user" {
			t.Errorf("Expected role 'user', got '%s'", msg.Role)
		}
		if msg.Type != MessageTypeUser {
			t.Errorf("Expected type 'user', got '%s'", msg.Type)
		}
		if msg.Content != "Hello, assistant!" {
			t.Errorf("Expected content 'Hello, assistant!', got '%s'", msg.Content)
		}
	})

	t.Run("NewArtifactMessage", func(t *testing.T) {
		content := "package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}"
		msg := NewArtifactMessage("artifact_123", ArtifactTypeCode, content)

		if msg.Type != MessageTypeArtifact {
			t.Errorf("Expected type 'artifact', got '%s'", msg.Type)
		}
		if msg.ArtifactID != "artifact_123" {
			t.Errorf("Expected artifact ID 'artifact_123', got '%s'", msg.ArtifactID)
		}
		if msg.ArtifactType != ArtifactTypeCode {
			t.Errorf("Expected artifact type 'code', got '%s'", msg.ArtifactType)
		}
		if msg.ArtifactSize != int64(len(content)) {
			t.Errorf("Expected artifact size %d, got %d", len(content), msg.ArtifactSize)
		}
	})

	t.Run("ToLLMMessage", func(t *testing.T) {
		agentMsg := NewUserMessage("Test message")
		agentMsg.ToolCalls = []providers.ToolCall{
			{ID: "call_1", Name: "test_tool", Arguments: map[string]any{}},
		}

		llmMsg := agentMsg.ToLLMMessage()
		if llmMsg.Role != "user" {
			t.Errorf("Expected role 'user', got '%s'", llmMsg.Role)
		}
		if llmMsg.Content != "Test message" {
			t.Errorf("Expected content 'Test message', got '%s'", llmMsg.Content)
		}
		if len(llmMsg.ToolCalls) != 1 {
			t.Errorf("Expected 1 tool call, got %d", len(llmMsg.ToolCalls))
		}
	})

	t.Run("ToLLMMessageWithContext_Artifact", func(t *testing.T) {
		msg := NewArtifactMessage("artifact_123", ArtifactTypeCode, "code content")
		llmMsg := msg.ToLLMMessageWithContext()

		// Should include artifact reference in content
		if llmMsg.Content == "" {
			t.Error("Expected non-empty content with artifact reference")
		}
		if llmMsg.Content[:10] != "[Artifact " {
			t.Errorf("Expected content to start with '[Artifact', got '%s'", llmMsg.Content[:20])
		}
	})

	t.Run("FromProviderMessage", func(t *testing.T) {
		providerMsg := providers.Message{
			Role:    "assistant",
			Content: "Hello, user!",
		}

		agentMsg := FromProviderMessage(providerMsg)
		if agentMsg.Role != "assistant" {
			t.Errorf("Expected role 'assistant', got '%s'", agentMsg.Role)
		}
		if agentMsg.Type != MessageTypeAssistant {
			t.Errorf("Expected type 'assistant', got '%s'", agentMsg.Type)
		}
		if agentMsg.Content != "Hello, user!" {
			t.Errorf("Expected content 'Hello, user!', got '%s'", agentMsg.Content)
		}
	})

	t.Run("WithMetadata", func(t *testing.T) {
		msg := NewUserMessage("Test").
			WithMetadata("key1", "value1").
			WithMetadata("key2", 42)

		if msg.Metadata["key1"] != "value1" {
			t.Errorf("Expected metadata key1='value1', got '%v'", msg.Metadata["key1"])
		}
		if msg.Metadata["key2"] != 42 {
			t.Errorf("Expected metadata key2=42, got '%v'", msg.Metadata["key2"])
		}
	})

	t.Run("Clone", func(t *testing.T) {
		original := NewUserMessage("Test").
			WithMetadata("key", "value").
			WithSessionID("session_1")

		cloned := original.Clone()

		// Modify clone
		cloned.Content = "Modified"
		cloned.Metadata["key"] = "modified"

		// Original should be unchanged
		if original.Content != "Test" {
			t.Error("Original content was modified")
		}
		if original.Metadata["key"] != "value" {
			t.Error("Original metadata was modified")
		}
	})

	t.Run("IsStandardLLMType", func(t *testing.T) {
		testCases := []struct {
			msgType  AgentMessageType
			expected bool
		}{
			{MessageTypeUser, true},
			{MessageTypeAssistant, true},
			{MessageTypeTool, true},
			{MessageTypeSystem, true},
			{MessageTypeArtifact, false},
			{MessageTypeAttachment, false},
			{MessageTypeEvent, false},
		}

		for _, tc := range testCases {
			msg := &AgentMessage{Type: tc.msgType}
			if msg.IsStandardLLMType() != tc.expected {
				t.Errorf("Type %s: expected IsStandardLLMType=%v, got %v",
					tc.msgType, tc.expected, msg.IsStandardLLMType())
			}
		}
	})
}

// TestMessageConverter tests the message converter functionality
func TestMessageConverter(t *testing.T) {
	t.Run("TransformContext_NoLimit", func(t *testing.T) {
		messages := []*AgentMessage{
			NewUserMessage("Message 1"),
			NewAssistantMessage("Response 1"),
			NewUserMessage("Message 2"),
		}

		converter := NewMessageConverter(DefaultTransformOptions())
		opts := TransformOptions{MaxMessages: 0} // No limit
		result := converter.TransformContext(messages, opts)

		if len(result) != len(messages) {
			t.Errorf("Expected %d messages, got %d", len(messages), len(result))
		}
	})

	t.Run("TransformContext_WithLimit", func(t *testing.T) {
		messages := []*AgentMessage{
			NewUserMessage("Message 1"),
			NewAssistantMessage("Response 1"),
			NewUserMessage("Message 2"),
			NewAssistantMessage("Response 2"),
			NewUserMessage("Message 3"),
		}

		converter := NewMessageConverter(DefaultTransformOptions())
		opts := TransformOptions{MaxMessages: 3}
		result := converter.TransformContext(messages, opts)

		if len(result) != 3 {
			t.Errorf("Expected 3 messages, got %d", len(result))
		}

		// Should keep the most recent messages
		if result[len(result)-1].Content != "Message 3" {
			t.Error("Expected most recent message to be preserved")
		}
	})

	t.Run("TransformContext_PreserveSystem", func(t *testing.T) {
		messages := []*AgentMessage{
			{Role: "system", Type: MessageTypeSystem, Content: "System prompt"},
			NewUserMessage("Message 1"),
			NewAssistantMessage("Response 1"),
			NewUserMessage("Message 2"),
			NewAssistantMessage("Response 2"),
		}

		converter := NewMessageConverter(DefaultTransformOptions())
		opts := TransformOptions{
			MaxMessages:    2,
			PreserveSystem: true,
		}
		result := converter.TransformContext(messages, opts)

		// Should preserve system message plus 2 recent messages
		if len(result) < 3 {
			t.Errorf("Expected at least 3 messages (1 system + 2 recent), got %d", len(result))
		}

		// First message should be system
		if result[0].Type != MessageTypeSystem {
			t.Error("Expected first message to be system message")
		}
	})

	t.Run("TransformContext_PreserveArtifacts", func(t *testing.T) {
		messages := []*AgentMessage{
			NewUserMessage("Message 1"),
			NewArtifactMessage("art_1", ArtifactTypeCode, "code 1"),
			NewUserMessage("Message 2"),
			NewArtifactMessage("art_2", ArtifactTypeCode, "code 2"),
			NewUserMessage("Message 3"),
		}

		converter := NewMessageConverter(DefaultTransformOptions())
		opts := TransformOptions{
			MaxMessages:       2,
			PreserveArtifacts: 2,
		}
		result := converter.TransformContext(messages, opts)

		artifactCount := 0
		for _, msg := range result {
			if msg.Type == MessageTypeArtifact {
				artifactCount++
			}
		}

		if artifactCount != 2 {
			t.Errorf("Expected 2 artifacts preserved, got %d", artifactCount)
		}
	})

	t.Run("ConvertToLLM", func(t *testing.T) {
		messages := []*AgentMessage{
			NewUserMessage("User message"),
			NewAssistantMessage("Assistant response"),
			NewArtifactMessage("art_1", ArtifactTypeCode, "code"),
			NewEventMessage("task_started", map[string]any{"task_id": "123"}),
		}

		converter := NewMessageConverter(TransformOptions{IncludeContext: false})
		result := converter.ConvertToLLM(messages)

		// Should have 2 messages (user + assistant), artifact and event dropped
		if len(result) != 2 {
			t.Errorf("Expected 2 LLM messages, got %d", len(result))
		}
	})

	t.Run("ConvertToLLM_WithContext", func(t *testing.T) {
		messages := []*AgentMessage{
			NewUserMessage("User message"),
			NewArtifactMessage("art_1", ArtifactTypeCode, "code"),
		}

		converter := NewMessageConverter(TransformOptions{IncludeContext: true})
		result := converter.ConvertToLLM(messages)

		// Should have 2 messages (user + artifact with context)
		if len(result) != 2 {
			t.Errorf("Expected 2 LLM messages, got %d", len(result))
		}

		// Artifact should include reference
		if result[1].Content[:10] != "[Artifact " {
			t.Error("Expected artifact reference in content")
		}
	})
}

// TestMessageFilters tests message filtering functionality
func TestMessageFilters(t *testing.T) {
	messages := []*AgentMessage{
		NewUserMessage("User 1"),
		NewAssistantMessage("Assistant 1"),
		NewArtifactMessage("art_1", ArtifactTypeCode, "code"),
		{Role: "system", Type: MessageTypeSystem, Content: "System"},
		NewEventMessage("event_1", nil),
	}

	t.Run("FilterByType", func(t *testing.T) {
		result := FilterMessages(messages, FilterByType(MessageTypeArtifact))
		if len(result) != 1 {
			t.Errorf("Expected 1 artifact message, got %d", len(result))
		}
		if result[0].Type != MessageTypeArtifact {
			t.Error("Expected artifact message")
		}
	})

	t.Run("FilterByRole", func(t *testing.T) {
		result := FilterMessages(messages, FilterByRole("user"))
		if len(result) != 1 {
			t.Errorf("Expected 1 user message, got %d", len(result))
		}
	})

	t.Run("FilterStandardTypes", func(t *testing.T) {
		result := FilterMessages(messages, FilterStandardTypes())
		if len(result) != 3 {
			t.Errorf("Expected 3 standard messages, got %d", len(result))
		}
	})

	t.Run("FilterExtendedTypes", func(t *testing.T) {
		result := FilterMessages(messages, FilterExtendedTypes())
		if len(result) != 2 {
			t.Errorf("Expected 2 extended messages, got %d", len(result))
		}
	})
}

// TestComputeStats tests message statistics
func TestComputeStats(t *testing.T) {
	messages := []*AgentMessage{
		NewUserMessage("User message"),
		NewAssistantMessage("Assistant response"),
		NewArtifactMessage("art_1", ArtifactTypeCode, "code content"),
		NewAttachmentMessage("/path/file.txt", "file.txt", 1024),
	}

	stats := ComputeStats(messages)

	if stats.Total != 4 {
		t.Errorf("Expected 4 total messages, got %d", stats.Total)
	}

	if stats.ArtifactCount != 1 {
		t.Errorf("Expected 1 artifact, got %d", stats.ArtifactCount)
	}

	if stats.AttachmentCount != 1 {
		t.Errorf("Expected 1 attachment, got %d", stats.AttachmentCount)
	}

	if stats.ByType[MessageTypeUser] != 1 {
		t.Errorf("Expected 1 user message, got %d", stats.ByType[MessageTypeUser])
	}

	if stats.TotalSize == 0 {
		t.Error("Expected non-zero total size")
	}
}

// TestBatchConversion tests batch message conversion
func TestBatchConversion(t *testing.T) {
	t.Run("BatchConvertFromProvider", func(t *testing.T) {
		providerMsgs := []providers.Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there"},
		}

		agentMsgs := BatchConvertFromProvider(providerMsgs)

		if len(agentMsgs) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(agentMsgs))
		}

		if agentMsgs[0].Type != MessageTypeUser {
			t.Error("Expected first message to be user type")
		}
	})

	t.Run("BatchConvertToProvider", func(t *testing.T) {
		agentMsgs := []*AgentMessage{
			NewUserMessage("Hello"),
			NewAssistantMessage("Hi there"),
		}

		providerMsgs := BatchConvertToProvider(agentMsgs)

		if len(providerMsgs) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(providerMsgs))
		}

		if providerMsgs[0].Role != "user" {
			t.Error("Expected first message to have user role")
		}
	})
}

// BenchmarkMessageConversion benchmarks message conversion
func BenchmarkMessageConversion(b *testing.B) {
	msg := NewUserMessage("Test message")

	b.Run("ToLLMMessage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = msg.ToLLMMessage()
		}
	})

	b.Run("ToLLMMessageWithContext", func(b *testing.B) {
		artifactMsg := NewArtifactMessage("art_1", ArtifactTypeCode, "code")
		for i := 0; i < b.N; i++ {
			_ = artifactMsg.ToLLMMessageWithContext()
		}
	})

	b.Run("Clone", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = msg.Clone()
		}
	})
}

// BenchmarkMessageFilter benchmarks message filtering
func BenchmarkMessageFilter(b *testing.B) {
	messages := make([]*AgentMessage, 100)
	for i := 0; i < 100; i++ {
		if i%3 == 0 {
			messages[i] = NewArtifactMessage("art", ArtifactTypeCode, "code")
		} else {
			messages[i] = NewUserMessage("message")
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FilterMessages(messages, FilterByType(MessageTypeArtifact))
	}
}
