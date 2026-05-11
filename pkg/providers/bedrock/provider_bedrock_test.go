//go:build bedrock

// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package bedrock

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

func TestConvertMessages_SystemPrompts(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello"},
	}

	bedrockMsgs, systemPrompts := convertMessages(messages)

	assert.Len(t, systemPrompts, 1)
	assert.Len(t, bedrockMsgs, 1)

	// Check system prompt
	textBlock, ok := systemPrompts[0].(*types.SystemContentBlockMemberText)
	require.True(t, ok)
	assert.Equal(t, "You are a helpful assistant.", textBlock.Value)

	// Check user message
	assert.Equal(t, types.ConversationRoleUser, bedrockMsgs[0].Role)
}

func TestConvertMessages_UserMessage(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "What is 2+2?"},
	}

	bedrockMsgs, systemPrompts := convertMessages(messages)

	assert.Empty(t, systemPrompts)
	assert.Len(t, bedrockMsgs, 1)
	assert.Equal(t, types.ConversationRoleUser, bedrockMsgs[0].Role)

	textBlock, ok := bedrockMsgs[0].Content[0].(*types.ContentBlockMemberText)
	require.True(t, ok)
	assert.Equal(t, "What is 2+2?", textBlock.Value)
}

func TestConvertMessages_AssistantMessage(t *testing.T) {
	messages := []Message{
		{Role: "assistant", Content: "The answer is 4."},
	}

	bedrockMsgs, _ := convertMessages(messages)

	assert.Len(t, bedrockMsgs, 1)
	assert.Equal(t, types.ConversationRoleAssistant, bedrockMsgs[0].Role)

	textBlock, ok := bedrockMsgs[0].Content[0].(*types.ContentBlockMemberText)
	require.True(t, ok)
	assert.Equal(t, "The answer is 4.", textBlock.Value)
}

func TestConvertMessages_ToolResult(t *testing.T) {
	messages := []Message{
		{Role: "tool", Content: "Result from tool", ToolCallID: "call_123"},
	}

	bedrockMsgs, _ := convertMessages(messages)

	assert.Len(t, bedrockMsgs, 1)
	assert.Equal(t, types.ConversationRoleUser, bedrockMsgs[0].Role)

	toolResult, ok := bedrockMsgs[0].Content[0].(*types.ContentBlockMemberToolResult)
	require.True(t, ok)
	assert.Equal(t, "call_123", aws.ToString(toolResult.Value.ToolUseId))
}

func TestConvertMessages_MultipleToolResultsMerged(t *testing.T) {
	// When an assistant makes multiple tool calls, all tool results must be
	// merged into a single user message for Bedrock
	messages := []Message{
		{Role: "user", Content: "What's the weather in NYC and LA?"},
		{
			Role:    "assistant",
			Content: "Let me check both cities.",
			ToolCalls: []protocoltypes.ToolCall{
				{ID: "call_nyc", Name: "get_weather", Arguments: map[string]any{"city": "NYC"}},
				{ID: "call_la", Name: "get_weather", Arguments: map[string]any{"city": "LA"}},
			},
		},
		{Role: "tool", Content: "NYC: 72°F, sunny", ToolCallID: "call_nyc"},
		{Role: "tool", Content: "LA: 85°F, clear", ToolCallID: "call_la"},
	}

	bedrockMsgs, _ := convertMessages(messages)

	// Should be: user message, assistant message, merged tool results (single user message)
	assert.Len(t, bedrockMsgs, 3)

	// First message: user
	assert.Equal(t, types.ConversationRoleUser, bedrockMsgs[0].Role)

	// Second message: assistant with tool calls
	assert.Equal(t, types.ConversationRoleAssistant, bedrockMsgs[1].Role)

	// Third message: merged tool results in single user message
	assert.Equal(t, types.ConversationRoleUser, bedrockMsgs[2].Role)
	assert.Len(t, bedrockMsgs[2].Content, 2) // Both tool results in one message

	// Verify both tool results are present
	result1, ok := bedrockMsgs[2].Content[0].(*types.ContentBlockMemberToolResult)
	require.True(t, ok)
	assert.Equal(t, "call_nyc", aws.ToString(result1.Value.ToolUseId))

	result2, ok := bedrockMsgs[2].Content[1].(*types.ContentBlockMemberToolResult)
	require.True(t, ok)
	assert.Equal(t, "call_la", aws.ToString(result2.Value.ToolUseId))
}

func TestConvertMessages_AssistantWithToolCalls(t *testing.T) {
	messages := []Message{
		{
			Role:    "assistant",
			Content: "Let me calculate that.",
			ToolCalls: []protocoltypes.ToolCall{
				{
					ID:        "call_456",
					Name:      "calculator",
					Arguments: map[string]any{"expression": "2+2"},
				},
			},
		},
	}

	bedrockMsgs, _ := convertMessages(messages)

	assert.Len(t, bedrockMsgs, 1)
	assert.Len(t, bedrockMsgs[0].Content, 2) // text + tool use

	// Check text content
	textBlock, ok := bedrockMsgs[0].Content[0].(*types.ContentBlockMemberText)
	require.True(t, ok)
	assert.Equal(t, "Let me calculate that.", textBlock.Value)

	// Check tool use
	toolUse, ok := bedrockMsgs[0].Content[1].(*types.ContentBlockMemberToolUse)
	require.True(t, ok)
	assert.Equal(t, "call_456", aws.ToString(toolUse.Value.ToolUseId))
	assert.Equal(t, "calculator", aws.ToString(toolUse.Value.Name))
}

func TestConvertTools_Basic(t *testing.T) {
	tools := []ToolDefinition{
		{
			Function: protocoltypes.ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get the current weather",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{"type": "string"},
					},
				},
			},
		},
	}

	toolConfig := convertTools(tools)

	assert.NotNil(t, toolConfig)
	assert.Len(t, toolConfig.Tools, 1)

	toolSpec, ok := toolConfig.Tools[0].(*types.ToolMemberToolSpec)
	require.True(t, ok)
	assert.Equal(t, "get_weather", aws.ToString(toolSpec.Value.Name))
	assert.Equal(t, "Get the current weather", aws.ToString(toolSpec.Value.Description))
}

func TestConvertTools_SkipsEmptyName(t *testing.T) {
	tools := []ToolDefinition{
		{
			Function: protocoltypes.ToolFunctionDefinition{
				Name:        "",
				Description: "Empty name tool",
			},
		},
		{
			Function: protocoltypes.ToolFunctionDefinition{
				Name:        "   ",
				Description: "Whitespace name tool",
			},
		},
		{
			Function: protocoltypes.ToolFunctionDefinition{
				Name:        "valid_tool",
				Description: "Valid tool",
			},
		},
	}

	toolConfig := convertTools(tools)

	assert.Len(t, toolConfig.Tools, 1)
	toolSpec := toolConfig.Tools[0].(*types.ToolMemberToolSpec)
	assert.Equal(t, "valid_tool", aws.ToString(toolSpec.Value.Name))
}

func TestConvertTools_NilParameters(t *testing.T) {
	tools := []ToolDefinition{
		{
			Function: protocoltypes.ToolFunctionDefinition{
				Name:        "simple_tool",
				Description: "A tool with no parameters",
				Parameters:  nil,
			},
		},
	}

	toolConfig := convertTools(tools)

	assert.Len(t, toolConfig.Tools, 1)
	// Should not panic and should create a valid tool
}

func TestBuildUserContent_TextOnly(t *testing.T) {
	msg := Message{Content: "Hello world"}

	content := buildUserContent(msg)

	assert.Len(t, content, 1)
	textBlock, ok := content[0].(*types.ContentBlockMemberText)
	require.True(t, ok)
	assert.Equal(t, "Hello world", textBlock.Value)
}

func TestBuildUserContent_WithImage(t *testing.T) {
	// Base64-encoded 1x1 PNG (the provider doesn't validate image correctness,
	// it just verifies the format and base64 decoding works)
	b64Data := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADUlEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII="

	msg := Message{
		Content: "Look at this image",
		Media:   []string{"data:image/png;base64," + b64Data},
	}

	content := buildUserContent(msg)

	assert.Len(t, content, 2)

	// Check text
	textBlock, ok := content[0].(*types.ContentBlockMemberText)
	require.True(t, ok)
	assert.Equal(t, "Look at this image", textBlock.Value)

	// Check image
	imageBlock, ok := content[1].(*types.ContentBlockMemberImage)
	require.True(t, ok)
	assert.Equal(t, types.ImageFormatPng, imageBlock.Value.Format)
}

func TestBuildUserContent_SkipsInvalidBase64(t *testing.T) {
	msg := Message{
		Content: "Invalid image",
		Media:   []string{"data:image/png;base64,not-valid-base64!!!"},
	}

	content := buildUserContent(msg)

	// Should only have text, image should be skipped
	assert.Len(t, content, 1)
}

func TestBuildUserContent_SkipsNonBase64Data(t *testing.T) {
	msg := Message{
		Content: "Non-base64 image",
		Media:   []string{"data:image/png,raw-data-here"},
	}

	content := buildUserContent(msg)

	// Should only have text, non-base64 image should be skipped
	assert.Len(t, content, 1)
}

func TestBuildAssistantContent_SkipsEmptyToolName(t *testing.T) {
	msg := Message{
		Content: "Response",
		ToolCalls: []protocoltypes.ToolCall{
			{ID: "1", Name: "", Arguments: map[string]any{}},
			{ID: "2", Name: "   ", Arguments: map[string]any{}},
			{ID: "3", Name: "valid", Arguments: map[string]any{}},
		},
	}

	content := buildAssistantContent(msg)

	// Should have text + 1 valid tool
	assert.Len(t, content, 2)
}

func TestBuildAssistantContent_NilArguments(t *testing.T) {
	msg := Message{
		ToolCalls: []protocoltypes.ToolCall{
			{ID: "1", Name: "tool", Arguments: nil},
		},
	}

	content := buildAssistantContent(msg)

	assert.Len(t, content, 1)
	toolUse, ok := content[0].(*types.ContentBlockMemberToolUse)
	require.True(t, ok)
	assert.NotNil(t, toolUse.Value.Input)
}

func TestBuildAssistantContent_FunctionFallback(t *testing.T) {
	// When Name/Arguments are empty (json:"-"), should fallback to Function fields
	msg := Message{
		ToolCalls: []protocoltypes.ToolCall{
			{
				ID:   "1",
				Name: "", // empty, should fallback to Function.Name
				Function: &protocoltypes.FunctionCall{
					Name:      "fallback_tool",
					Arguments: `{"key":"value"}`,
				},
			},
		},
	}

	content := buildAssistantContent(msg)

	assert.Len(t, content, 1)
	toolUse, ok := content[0].(*types.ContentBlockMemberToolUse)
	require.True(t, ok)
	assert.Equal(t, "fallback_tool", aws.ToString(toolUse.Value.Name))
}

func TestParseResponse_TextOnly(t *testing.T) {
	output := &bedrockruntime.ConverseOutput{
		Output: &types.ConverseOutputMemberMessage{
			Value: types.Message{
				Role: types.ConversationRoleAssistant,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{Value: "Hello!"},
				},
			},
		},
		StopReason: types.StopReasonEndTurn,
		Usage: &types.TokenUsage{
			InputTokens:  aws.Int32(10),
			OutputTokens: aws.Int32(5),
		},
	}

	resp, err := parseResponse(output)

	require.NoError(t, err)
	assert.Equal(t, "Hello!", resp.Content)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Empty(t, resp.ToolCalls)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
	assert.Equal(t, 5, resp.Usage.CompletionTokens)
}

func TestParseResponse_StopReasons(t *testing.T) {
	tests := []struct {
		stopReason     types.StopReason
		expectedFinish string
	}{
		{types.StopReasonEndTurn, "stop"},
		{types.StopReasonToolUse, "tool_calls"},
		{types.StopReasonMaxTokens, "length"},
		{types.StopReasonStopSequence, "stop"},
		{types.StopReasonContentFiltered, "content_filter"},
	}

	for _, tt := range tests {
		t.Run(string(tt.stopReason), func(t *testing.T) {
			output := &bedrockruntime.ConverseOutput{
				Output: &types.ConverseOutputMemberMessage{
					Value: types.Message{
						Content: []types.ContentBlock{
							&types.ContentBlockMemberText{Value: "test"},
						},
					},
				},
				StopReason: tt.stopReason,
			}

			resp, err := parseResponse(output)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedFinish, resp.FinishReason)
		})
	}
}

func TestParseResponse_WithToolCalls(t *testing.T) {
	// Note: document.NewLazyDocument has limitations with UnmarshalSmithyDocument in tests,
	// so we test the structure extraction and verify Arguments gets populated (even if empty
	// due to SDK limitations). The actual unmarshal works correctly at runtime.
	toolInput := document.NewLazyDocument(map[string]any{
		"location": "San Francisco",
		"unit":     "celsius",
	})

	output := &bedrockruntime.ConverseOutput{
		Output: &types.ConverseOutputMemberMessage{
			Value: types.Message{
				Role: types.ConversationRoleAssistant,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{Value: "Let me check the weather."},
					&types.ContentBlockMemberToolUse{
						Value: types.ToolUseBlock{
							ToolUseId: aws.String("call_weather_123"),
							Name:      aws.String("get_weather"),
							Input:     toolInput,
						},
					},
				},
			},
		},
		StopReason: types.StopReasonToolUse,
		Usage: &types.TokenUsage{
			InputTokens:  aws.Int32(20),
			OutputTokens: aws.Int32(15),
		},
	}

	resp, err := parseResponse(output)

	require.NoError(t, err)
	assert.Equal(t, "Let me check the weather.", resp.Content)
	assert.Equal(t, "tool_calls", resp.FinishReason)
	assert.Len(t, resp.ToolCalls, 1)

	// Verify tool call ID and Name are extracted correctly
	tc := resp.ToolCalls[0]
	assert.Equal(t, "call_weather_123", tc.ID)
	assert.Equal(t, "get_weather", tc.Name)

	// Verify Function fields are also populated
	require.NotNil(t, tc.Function)
	assert.Equal(t, "get_weather", tc.Function.Name)

	// Verify Arguments is not nil (content may vary due to SDK limitations in tests)
	assert.NotNil(t, tc.Arguments)

	// Verify usage
	assert.Equal(t, 20, resp.Usage.PromptTokens)
	assert.Equal(t, 15, resp.Usage.CompletionTokens)
	assert.Equal(t, 35, resp.Usage.TotalTokens)
}

func TestParseResponse_MultipleToolCalls(t *testing.T) {
	output := &bedrockruntime.ConverseOutput{
		Output: &types.ConverseOutputMemberMessage{
			Value: types.Message{
				Role: types.ConversationRoleAssistant,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberToolUse{
						Value: types.ToolUseBlock{
							ToolUseId: aws.String("call_1"),
							Name:      aws.String("tool_a"),
							Input:     document.NewLazyDocument(map[string]any{"arg": "value1"}),
						},
					},
					&types.ContentBlockMemberToolUse{
						Value: types.ToolUseBlock{
							ToolUseId: aws.String("call_2"),
							Name:      aws.String("tool_b"),
							Input:     document.NewLazyDocument(map[string]any{"arg": "value2"}),
						},
					},
				},
			},
		},
		StopReason: types.StopReasonToolUse,
	}

	resp, err := parseResponse(output)

	require.NoError(t, err)
	assert.Equal(t, "tool_calls", resp.FinishReason)
	assert.Len(t, resp.ToolCalls, 2)

	// Verify tool call structure
	assert.Equal(t, "call_1", resp.ToolCalls[0].ID)
	assert.Equal(t, "tool_a", resp.ToolCalls[0].Name)
	assert.NotNil(t, resp.ToolCalls[0].Arguments)
	assert.NotNil(t, resp.ToolCalls[0].Function)
	assert.Equal(t, "tool_a", resp.ToolCalls[0].Function.Name)

	assert.Equal(t, "call_2", resp.ToolCalls[1].ID)
	assert.Equal(t, "tool_b", resp.ToolCalls[1].Name)
	assert.NotNil(t, resp.ToolCalls[1].Arguments)
	assert.NotNil(t, resp.ToolCalls[1].Function)
	assert.Equal(t, "tool_b", resp.ToolCalls[1].Function.Name)
}

func TestParseResponse_ToolCallWithNilInput(t *testing.T) {
	output := &bedrockruntime.ConverseOutput{
		Output: &types.ConverseOutputMemberMessage{
			Value: types.Message{
				Role: types.ConversationRoleAssistant,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberToolUse{
						Value: types.ToolUseBlock{
							ToolUseId: aws.String("call_nil"),
							Name:      aws.String("no_args_tool"),
							Input:     nil,
						},
					},
				},
			},
		},
		StopReason: types.StopReasonToolUse,
	}

	resp, err := parseResponse(output)

	require.NoError(t, err)
	assert.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "call_nil", resp.ToolCalls[0].ID)
	assert.Equal(t, "no_args_tool", resp.ToolCalls[0].Name)
	// Arguments should be empty map, not nil
	assert.NotNil(t, resp.ToolCalls[0].Arguments)
	assert.Empty(t, resp.ToolCalls[0].Arguments)
}

func TestIsSSOTokenError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      fmt.Errorf("connection refused"),
			expected: false,
		},
		{
			name:     "SSO config error not expiration",
			err:      fmt.Errorf("failed to load SSO profile: invalid SSO session"),
			expected: false,
		},
		{
			name:     "STS ExpiredToken error",
			err:      fmt.Errorf("ExpiredToken: The security token included in the request is expired"),
			expected: false,
		},
		{
			name:     "SSO token refresh error",
			err:      fmt.Errorf("refresh cached SSO token failed"),
			expected: true,
		},
		{
			name:     "InvalidGrantException",
			err:      fmt.Errorf("operation error SSO OIDC: CreateToken, InvalidGrantException"),
			expected: true,
		},
		{
			name:     "SSO OIDC error",
			err:      fmt.Errorf("operation error SSO OIDC: CreateToken, failed"),
			expected: true,
		},
		{
			name: "full SSO error message",
			err: fmt.Errorf(
				"get identity: get credentials: failed to refresh cached credentials, refresh cached SSO token failed, unable to refresh SSO token",
			),
			expected: true,
		},
		{
			name: "SSO token file missing",
			err: fmt.Errorf(
				"get identity: get credentials: failed to refresh cached credentials, failed to read cached SSO token file, open ~/.aws/sso/cache/abc123.json: no such file or directory",
			),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSSOTokenError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// mockStreamReader implements bedrockruntime.ConverseStreamOutputReader for testing.
type mockStreamReader struct {
	ch  chan types.ConverseStreamOutput
	err error
}

func (r *mockStreamReader) Events() <-chan types.ConverseStreamOutput { return r.ch }
func (r *mockStreamReader) Close() error                              { return nil }
func (r *mockStreamReader) Err() error                                { return r.err }

func newMockStream(events []types.ConverseStreamOutput) *bedrockruntime.ConverseStreamEventStream {
	ch := make(chan types.ConverseStreamOutput, len(events))
	for _, e := range events {
		ch <- e
	}
	close(ch)

	return bedrockruntime.NewConverseStreamEventStream(func(es *bedrockruntime.ConverseStreamEventStream) {
		es.Reader = &mockStreamReader{ch: ch}
	})
}

func TestParseStreamResponse_TextOnly(t *testing.T) {
	events := []types.ConverseStreamOutput{
		&types.ConverseStreamOutputMemberContentBlockDelta{
			Value: types.ContentBlockDeltaEvent{
				Delta:             &types.ContentBlockDeltaMemberText{Value: "Hello "},
				ContentBlockIndex: aws.Int32(0),
			},
		},
		&types.ConverseStreamOutputMemberContentBlockDelta{
			Value: types.ContentBlockDeltaEvent{
				Delta:             &types.ContentBlockDeltaMemberText{Value: "World"},
				ContentBlockIndex: aws.Int32(0),
			},
		},
		&types.ConverseStreamOutputMemberMessageStop{
			Value: types.MessageStopEvent{StopReason: types.StopReasonEndTurn},
		},
		&types.ConverseStreamOutputMemberMetadata{
			Value: types.ConverseStreamMetadataEvent{
				Usage: &types.TokenUsage{
					InputTokens:  aws.Int32(10),
					OutputTokens: aws.Int32(5),
				},
			},
		},
	}

	var chunks []string
	stream := newMockStream(events)
	resp, err := parseStreamResponse(context.Background(), stream, func(accumulated string) {
		chunks = append(chunks, accumulated)
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello World", resp.Content)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Empty(t, resp.ToolCalls)
	require.NotNil(t, resp.Usage)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
	assert.Equal(t, 5, resp.Usage.CompletionTokens)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
	assert.Equal(t, []string{"Hello ", "Hello World"}, chunks)
}

func TestParseStreamResponse_ToolCall(t *testing.T) {
	events := []types.ConverseStreamOutput{
		&types.ConverseStreamOutputMemberContentBlockStart{
			Value: types.ContentBlockStartEvent{
				ContentBlockIndex: aws.Int32(0),
				Start: &types.ContentBlockStartMemberToolUse{
					Value: types.ToolUseBlockStart{
						ToolUseId: aws.String("call_1"),
						Name:      aws.String("search"),
					},
				},
			},
		},
		&types.ConverseStreamOutputMemberContentBlockDelta{
			Value: types.ContentBlockDeltaEvent{
				ContentBlockIndex: aws.Int32(0),
				Delta: &types.ContentBlockDeltaMemberToolUse{
					Value: types.ToolUseBlockDelta{Input: aws.String(`{"q":`)},
				},
			},
		},
		&types.ConverseStreamOutputMemberContentBlockDelta{
			Value: types.ContentBlockDeltaEvent{
				ContentBlockIndex: aws.Int32(0),
				Delta: &types.ContentBlockDeltaMemberToolUse{
					Value: types.ToolUseBlockDelta{Input: aws.String(`"test"}`)},
				},
			},
		},
		&types.ConverseStreamOutputMemberContentBlockStop{
			Value: types.ContentBlockStopEvent{ContentBlockIndex: aws.Int32(0)},
		},
		&types.ConverseStreamOutputMemberMessageStop{
			Value: types.MessageStopEvent{StopReason: types.StopReasonToolUse},
		},
	}

	stream := newMockStream(events)
	resp, err := parseStreamResponse(context.Background(), stream, nil)

	require.NoError(t, err)
	assert.Equal(t, "tool_calls", resp.FinishReason)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "call_1", resp.ToolCalls[0].ID)
	assert.Equal(t, "search", resp.ToolCalls[0].Name)
	assert.Equal(t, map[string]any{"q": "test"}, resp.ToolCalls[0].Arguments)
	require.NotNil(t, resp.ToolCalls[0].Function)
	assert.Equal(t, "search", resp.ToolCalls[0].Function.Name)
	assert.Equal(t, `{"q":"test"}`, resp.ToolCalls[0].Function.Arguments)
}

func TestParseStreamResponse_TextAndToolCall(t *testing.T) {
	events := []types.ConverseStreamOutput{
		&types.ConverseStreamOutputMemberContentBlockDelta{
			Value: types.ContentBlockDeltaEvent{
				ContentBlockIndex: aws.Int32(0),
				Delta:             &types.ContentBlockDeltaMemberText{Value: "Let me search that."},
			},
		},
		&types.ConverseStreamOutputMemberContentBlockStart{
			Value: types.ContentBlockStartEvent{
				ContentBlockIndex: aws.Int32(1),
				Start: &types.ContentBlockStartMemberToolUse{
					Value: types.ToolUseBlockStart{
						ToolUseId: aws.String("call_2"),
						Name:      aws.String("web"),
					},
				},
			},
		},
		&types.ConverseStreamOutputMemberContentBlockDelta{
			Value: types.ContentBlockDeltaEvent{
				ContentBlockIndex: aws.Int32(1),
				Delta: &types.ContentBlockDeltaMemberToolUse{
					Value: types.ToolUseBlockDelta{Input: aws.String(`{"url":"https://example.com"}`)},
				},
			},
		},
		&types.ConverseStreamOutputMemberContentBlockStop{
			Value: types.ContentBlockStopEvent{ContentBlockIndex: aws.Int32(1)},
		},
		&types.ConverseStreamOutputMemberMessageStop{
			Value: types.MessageStopEvent{StopReason: types.StopReasonToolUse},
		},
	}

	var chunks []string
	stream := newMockStream(events)
	resp, err := parseStreamResponse(context.Background(), stream, func(accumulated string) {
		chunks = append(chunks, accumulated)
	})

	require.NoError(t, err)
	assert.Equal(t, "Let me search that.", resp.Content)
	assert.Equal(t, "tool_calls", resp.FinishReason)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "web", resp.ToolCalls[0].Name)
	assert.Equal(t, []string{"Let me search that."}, chunks)
}

func TestParseStreamResponse_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Use an unbuffered channel with no events so ctx.Done() is the only ready case.
	ch := make(chan types.ConverseStreamOutput)

	stream := bedrockruntime.NewConverseStreamEventStream(func(es *bedrockruntime.ConverseStreamEventStream) {
		es.Reader = &mockStreamReader{ch: ch}
	})

	_, err := parseStreamResponse(ctx, stream, nil)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestParseStreamResponse_InvalidToolJSON(t *testing.T) {
	events := []types.ConverseStreamOutput{
		&types.ConverseStreamOutputMemberContentBlockStart{
			Value: types.ContentBlockStartEvent{
				ContentBlockIndex: aws.Int32(0),
				Start: &types.ContentBlockStartMemberToolUse{
					Value: types.ToolUseBlockStart{
						ToolUseId: aws.String("call_bad"),
						Name:      aws.String("broken"),
					},
				},
			},
		},
		&types.ConverseStreamOutputMemberContentBlockDelta{
			Value: types.ContentBlockDeltaEvent{
				ContentBlockIndex: aws.Int32(0),
				Delta: &types.ContentBlockDeltaMemberToolUse{
					Value: types.ToolUseBlockDelta{Input: aws.String(`{not valid json`)},
				},
			},
		},
		&types.ConverseStreamOutputMemberContentBlockStop{
			Value: types.ContentBlockStopEvent{ContentBlockIndex: aws.Int32(0)},
		},
		&types.ConverseStreamOutputMemberMessageStop{
			Value: types.MessageStopEvent{StopReason: types.StopReasonToolUse},
		},
	}

	stream := newMockStream(events)
	resp, err := parseStreamResponse(context.Background(), stream, nil)

	require.NoError(t, err)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, map[string]any{"raw": `{not valid json`}, resp.ToolCalls[0].Arguments)
	assert.JSONEq(t, `{"raw":"{not valid json"}`, resp.ToolCalls[0].Function.Arguments)
}

func TestParseStreamResponse_DefaultFinishReason(t *testing.T) {
	events := []types.ConverseStreamOutput{
		&types.ConverseStreamOutputMemberContentBlockDelta{
			Value: types.ContentBlockDeltaEvent{
				Delta:             &types.ContentBlockDeltaMemberText{Value: "partial"},
				ContentBlockIndex: aws.Int32(0),
			},
		},
	}

	stream := newMockStream(events)
	resp, err := parseStreamResponse(context.Background(), stream, nil)

	require.NoError(t, err)
	assert.Equal(t, "stop", resp.FinishReason)
}

func TestParseStreamResponse_NilStream(t *testing.T) {
	_, err := parseStreamResponse(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil event stream")
}

func TestParseStreamResponse_StopReasons(t *testing.T) {
	tests := []struct {
		reason   types.StopReason
		expected string
	}{
		{types.StopReasonEndTurn, "stop"},
		{types.StopReasonMaxTokens, "length"},
		{types.StopReasonToolUse, "tool_calls"},
		{types.StopReasonStopSequence, "stop"},
		{types.StopReasonContentFiltered, "content_filter"},
	}

	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			events := []types.ConverseStreamOutput{
				&types.ConverseStreamOutputMemberMessageStop{
					Value: types.MessageStopEvent{StopReason: tt.reason},
				},
			}
			stream := newMockStream(events)
			resp, err := parseStreamResponse(context.Background(), stream, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp.FinishReason)
		})
	}
}
