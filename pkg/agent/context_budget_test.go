package agent

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// msgUser creates a user message.
func msgUser(content string) providers.Message {
	return providers.Message{Role: "user", Content: content}
}

// msgAssistant creates a plain assistant message (no tool calls).
func msgAssistant(content string) providers.Message {
	return providers.Message{Role: "assistant", Content: content}
}

// msgAssistantTC creates an assistant message with tool calls.
func msgAssistantTC(toolIDs ...string) providers.Message {
	tcs := make([]providers.ToolCall, len(toolIDs))
	for i, id := range toolIDs {
		tcs[i] = providers.ToolCall{
			ID:   id,
			Type: "function",
			Name: "tool_" + id,
			Function: &providers.FunctionCall{
				Name:      "tool_" + id,
				Arguments: `{"key":"value"}`,
			},
		}
	}
	return providers.Message{Role: "assistant", ToolCalls: tcs}
}

// msgTool creates a tool result message.
func msgTool(callID, content string) providers.Message {
	return providers.Message{Role: "tool", ToolCallID: callID, Content: content}
}

func TestParseTurnBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		history []providers.Message
		want    []int
	}{
		{
			name:    "empty history",
			history: nil,
			want:    nil,
		},
		{
			name: "simple exchange",
			history: []providers.Message{
				msgUser("q1"),
				msgAssistant("a1"),
				msgUser("q2"),
				msgAssistant("a2"),
			},
			want: []int{0, 2},
		},
		{
			name: "tool-call Turn",
			history: []providers.Message{
				msgUser("search"),
				msgAssistantTC("tc1"),
				msgTool("tc1", "result"),
				msgAssistant("found it"),
				msgUser("thanks"),
				msgAssistant("welcome"),
			},
			want: []int{0, 4},
		},
		{
			name: "chained tool calls in single Turn",
			history: []providers.Message{
				msgUser("save and notify"),
				msgAssistantTC("tc_save"),
				msgTool("tc_save", "saved"),
				msgAssistantTC("tc_notify"),
				msgTool("tc_notify", "notified"),
				msgAssistant("done"),
			},
			want: []int{0},
		},
		{
			name: "no user messages",
			history: []providers.Message{
				msgAssistant("a1"),
				msgAssistant("a2"),
			},
			want: nil,
		},
		{
			name: "leading non-user messages",
			history: []providers.Message{
				msgAssistantTC("tc1"),
				msgTool("tc1", "r1"),
				msgAssistant("greeting"),
				msgUser("hello"),
				msgAssistant("hi"),
			},
			want: []int{3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTurnBoundaries(tt.history)
			if len(got) != len(tt.want) {
				t.Errorf("parseTurnBoundaries() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseTurnBoundaries()[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsSafeBoundary(t *testing.T) {
	tests := []struct {
		name    string
		history []providers.Message
		index   int
		want    bool
	}{
		{
			name:    "empty history, index 0",
			history: nil,
			index:   0,
			want:    true,
		},
		{
			name:    "single user message, index 0",
			history: []providers.Message{msgUser("hi")},
			index:   0,
			want:    true,
		},
		{
			name:    "single user message, index 1 (end)",
			history: []providers.Message{msgUser("hi")},
			index:   1,
			want:    true,
		},
		{
			name: "at user message",
			history: []providers.Message{
				msgAssistant("hello"),
				msgUser("how are you"),
				msgAssistant("fine"),
			},
			index: 1,
			want:  true,
		},
		{
			name: "at assistant without tool calls",
			history: []providers.Message{
				msgUser("hello"),
				msgAssistant("response"),
				msgUser("follow up"),
			},
			index: 1,
			want:  false,
		},
		{
			name: "at assistant with tool calls",
			history: []providers.Message{
				msgUser("search something"),
				msgAssistantTC("tc1"),
				msgTool("tc1", "result"),
				msgAssistant("here is what I found"),
			},
			index: 1,
			want:  false,
		},
		{
			name: "at tool result",
			history: []providers.Message{
				msgUser("do something"),
				msgAssistantTC("tc1"),
				msgTool("tc1", "done"),
				msgAssistant("completed"),
			},
			index: 2,
			want:  false,
		},
		{
			name: "negative index",
			history: []providers.Message{
				msgUser("hello"),
			},
			index: -1,
			want:  true,
		},
		{
			name: "index beyond length",
			history: []providers.Message{
				msgUser("hello"),
			},
			index: 5,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSafeBoundary(tt.history, tt.index)
			if got != tt.want {
				t.Errorf("isSafeBoundary(history, %d) = %v, want %v", tt.index, got, tt.want)
			}
		})
	}
}

func TestFindSafeBoundary(t *testing.T) {
	tests := []struct {
		name        string
		history     []providers.Message
		targetIndex int
		want        int
	}{
		{
			name:        "empty history",
			history:     nil,
			targetIndex: 0,
			want:        0,
		},
		{
			name:        "target at 0",
			history:     []providers.Message{msgUser("hi")},
			targetIndex: 0,
			want:        0,
		},
		{
			name:        "target beyond length",
			history:     []providers.Message{msgUser("hi")},
			targetIndex: 5,
			want:        1,
		},
		{
			name: "target already at user message",
			history: []providers.Message{
				msgUser("q1"),
				msgAssistant("a1"),
				msgUser("q2"),
				msgAssistant("a2"),
			},
			targetIndex: 2,
			want:        2,
		},
		{
			name: "target at assistant, scan backward finds user",
			history: []providers.Message{
				msgUser("q1"),
				msgAssistant("a1"),
				msgUser("q2"),
				msgAssistant("a2"),
				msgUser("q3"),
			},
			targetIndex: 3, // assistant "a2"
			want:        2, // backward to user "q2"
		},
		{
			name: "target inside tool sequence, scan backward finds user",
			history: []providers.Message{
				msgUser("q1"),
				msgAssistant("a1"),
				msgUser("q2"),
				msgAssistantTC("tc1", "tc2"),
				msgTool("tc1", "r1"),
				msgTool("tc2", "r2"),
				msgAssistant("summary"),
				msgUser("q3"),
			},
			targetIndex: 4, // tool result "r1"
			want:        2, // backward: 3=assistant+TC (not safe), 2=user → safe
		},
		{
			name: "target inside tool sequence, backward finds user before chain",
			history: []providers.Message{
				msgUser("q1"),
				msgAssistant("a1"),
				msgUser("q2"),
				msgAssistantTC("tc1", "tc2"),
				msgTool("tc1", "r1"),
				msgTool("tc2", "r2"),
				msgAssistant("summary"),
				msgUser("q3"),
			},
			targetIndex: 5, // tool result "r2"
			want:        2, // backward: 4=tool, 3=assistant+TC, 2=user → safe
		},
		{
			name: "no backward user, scan forward finds one",
			history: []providers.Message{
				msgAssistantTC("tc1"),
				msgTool("tc1", "r1"),
				msgAssistant("a1"),
				msgUser("q1"),
			},
			targetIndex: 1, // tool result
			want:        3, // forward to user "q1"
		},
		{
			name: "multi-step tool chain preserves atomicity",
			history: []providers.Message{
				msgUser("q1"),
				msgAssistant("a1"),
				msgUser("q2"),
				msgAssistantTC("tc1"),
				msgTool("tc1", "r1"),
				msgAssistantTC("tc2"),
				msgTool("tc2", "r2"),
				msgAssistant("final"),
				msgUser("q3"),
				msgAssistant("a3"),
			},
			targetIndex: 5, // second assistant+TC
			want:        2, // backward: 4=tool, 3=assistant+TC, 2=user → safe
		},
		{
			name: "all non-user messages returns target unchanged",
			history: []providers.Message{
				msgAssistant("a1"),
				msgAssistant("a2"),
				msgAssistant("a3"),
			},
			targetIndex: 1,
			want:        1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findSafeBoundary(tt.history, tt.targetIndex)
			if got != tt.want {
				t.Errorf("findSafeBoundary(history, %d) = %d, want %d",
					tt.targetIndex, got, tt.want)
			}
		})
	}
}

func TestFindSafeBoundary_BackwardScanSkipsToolSequence(t *testing.T) {
	// A long tool-call chain: user → assistant+TC → tool → tool → ... → assistant → user
	// Target is inside the chain; boundary should skip the entire chain backward.
	history := []providers.Message{
		msgUser("start"),                 // 0
		msgAssistant("before chain"),     // 1
		msgUser("trigger"),               // 2 ← expected safe boundary
		msgAssistantTC("t1", "t2", "t3"), // 3
		msgTool("t1", "r1"),              // 4
		msgTool("t2", "r2"),              // 5
		msgTool("t3", "r3"),              // 6
		msgAssistantTC("t4"),             // 7
		msgTool("t4", "r4"),              // 8
		msgAssistant("chain done"),       // 9
		msgUser("next"),                  // 10
	}

	// Target at index 6 (middle of tool results)
	got := findSafeBoundary(history, 6)
	if got != 2 {
		t.Errorf("findSafeBoundary(history, 6) = %d, want 2 (user before chain)", got)
	}
}

func TestEstimateMessageTokens(t *testing.T) {
	tests := []struct {
		name string
		msg  providers.Message
		want int // minimum expected tokens (exact value depends on overhead)
	}{
		{
			name: "plain user message",
			msg:  msgUser("Hello, world!"),
			want: 1, // at least some tokens
		},
		{
			name: "empty message still has overhead",
			msg:  providers.Message{Role: "user"},
			want: 1, // message overhead alone
		},
		{
			name: "assistant with tool calls",
			msg:  msgAssistantTC("tc_123"),
			want: 1,
		},
		{
			name: "tool result with ID",
			msg:  msgTool("call_abc", "Here is the search result with lots of content"),
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateMessageTokens(tt.msg)
			if got < tt.want {
				t.Errorf("estimateMessageTokens() = %d, want >= %d", got, tt.want)
			}
		})
	}
}

func TestEstimateMessageTokens_ToolCallsContribute(t *testing.T) {
	plain := msgAssistant("thinking")
	withTC := providers.Message{
		Role:    "assistant",
		Content: "thinking",
		ToolCalls: []providers.ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Name: "web_search",
				Function: &providers.FunctionCall{
					Name:      "web_search",
					Arguments: `{"query":"picoclaw agent framework","max_results":5}`,
				},
			},
		},
	}

	plainTokens := estimateMessageTokens(plain)
	withTCTokens := estimateMessageTokens(withTC)

	if withTCTokens <= plainTokens {
		t.Errorf("message with ToolCalls (%d tokens) should exceed plain message (%d tokens)",
			withTCTokens, plainTokens)
	}
}

func TestEstimateMessageTokens_MultibyteContent(t *testing.T) {
	// Multi-byte characters (e.g. emoji, accented letters) are single runes
	// but may map to different token counts. The heuristic should still produce
	// reasonable estimates via RuneCountInString.
	msg := msgUser("caf\u00e9 na\u00efve r\u00e9sum\u00e9 \u00fcber stra\u00dfe")
	tokens := estimateMessageTokens(msg)
	if tokens <= 0 {
		t.Errorf("multibyte message should produce positive token count, got %d", tokens)
	}
}

func TestEstimateMessageTokens_LargeArguments(t *testing.T) {
	// Simulate a tool call with large JSON arguments.
	largeArgs := fmt.Sprintf(`{"content":"%s"}`, strings.Repeat("x", 5000))
	msg := providers.Message{
		Role: "assistant",
		ToolCalls: []providers.ToolCall{
			{
				ID:   "call_large",
				Type: "function",
				Name: "write_file",
				Function: &providers.FunctionCall{
					Name:      "write_file",
					Arguments: largeArgs,
				},
			},
		},
	}

	tokens := estimateMessageTokens(msg)
	// 5000+ chars → at least 2000 tokens with the 2.5 char/token heuristic
	if tokens < 2000 {
		t.Errorf("large tool call arguments should produce significant token count, got %d", tokens)
	}
}

func TestEstimateMessageTokens_ReasoningContent(t *testing.T) {
	plain := msgAssistant("result")
	withReasoning := providers.Message{
		Role:             "assistant",
		Content:          "result",
		ReasoningContent: strings.Repeat("thinking step ", 200),
	}

	plainTokens := estimateMessageTokens(plain)
	reasoningTokens := estimateMessageTokens(withReasoning)

	if reasoningTokens <= plainTokens {
		t.Errorf("message with ReasoningContent (%d tokens) should exceed plain message (%d tokens)",
			reasoningTokens, plainTokens)
	}
}

func TestEstimateMessageTokens_MediaItems(t *testing.T) {
	plain := msgUser("describe this")
	withMedia := providers.Message{
		Role:    "user",
		Content: "describe this",
		Media:   []string{"media://img1.png", "media://img2.png"},
	}

	plainTokens := estimateMessageTokens(plain)
	mediaTokens := estimateMessageTokens(withMedia)

	if mediaTokens <= plainTokens {
		t.Errorf("message with Media (%d tokens) should exceed plain message (%d tokens)",
			mediaTokens, plainTokens)
	}
}

// --- estimateToolDefsTokens tests ---

func TestEstimateToolDefsTokens(t *testing.T) {
	tests := []struct {
		name string
		defs []providers.ToolDefinition
		want int // minimum expected tokens
	}{
		{
			name: "empty tool list",
			defs: nil,
			want: 0,
		},
		{
			name: "single tool with params",
			defs: []providers.ToolDefinition{
				{
					Type: "function",
					Function: providers.ToolFunctionDefinition{
						Name:        "web_search",
						Description: "Search the web for information",
						Parameters: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"query": map[string]any{"type": "string"},
							},
							"required": []any{"query"},
						},
					},
				},
			},
			want: 1,
		},
		{
			name: "tool without params",
			defs: []providers.ToolDefinition{
				{
					Type: "function",
					Function: providers.ToolFunctionDefinition{
						Name:        "list_dir",
						Description: "List directory contents",
					},
				},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateToolDefsTokens(tt.defs)
			if got < tt.want {
				t.Errorf("estimateToolDefsTokens() = %d, want >= %d", got, tt.want)
			}
		})
	}
}

func TestEstimateToolDefsTokens_ScalesWithCount(t *testing.T) {
	makeTool := func(name string) providers.ToolDefinition {
		return providers.ToolDefinition{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        name,
				Description: "A test tool that does something useful",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"input": map[string]any{"type": "string", "description": "Input value"},
					},
				},
			},
		}
	}

	one := estimateToolDefsTokens([]providers.ToolDefinition{makeTool("tool_a")})
	three := estimateToolDefsTokens([]providers.ToolDefinition{
		makeTool("tool_a"), makeTool("tool_b"), makeTool("tool_c"),
	})

	if three <= one {
		t.Errorf("3 tools (%d tokens) should exceed 1 tool (%d tokens)", three, one)
	}
}

// --- isOverContextBudget tests ---

func TestIsOverContextBudget(t *testing.T) {
	systemMsg := providers.Message{Role: "system", Content: strings.Repeat("x", 1000)}
	userMsg := msgUser("hello")
	smallHistory := []providers.Message{systemMsg, msgUser("q1"), msgAssistant("a1"), userMsg}

	tools := []providers.ToolDefinition{
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters:  map[string]any{"type": "object"},
			},
		},
	}

	tests := []struct {
		name          string
		contextWindow int
		messages      []providers.Message
		toolDefs      []providers.ToolDefinition
		maxTokens     int
		want          bool
	}{
		{
			name:          "within budget",
			contextWindow: 100000,
			messages:      smallHistory,
			toolDefs:      tools,
			maxTokens:     4096,
			want:          false,
		},
		{
			name:          "over budget with small window",
			contextWindow: 100, // very small window
			messages:      smallHistory,
			toolDefs:      tools,
			maxTokens:     4096,
			want:          true,
		},
		{
			name:          "large max_tokens eats budget",
			contextWindow: 2000,
			messages:      smallHistory,
			toolDefs:      tools,
			maxTokens:     1800, // leaves almost no room
			want:          true,
		},
		{
			name:          "empty messages within budget",
			contextWindow: 10000,
			messages:      nil,
			toolDefs:      nil,
			maxTokens:     4096,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOverContextBudget(tt.contextWindow, tt.messages, tt.toolDefs, tt.maxTokens)
			if got != tt.want {
				t.Errorf("isOverContextBudget() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Tests reflecting actual session data shape ---
// Session history never contains system messages. The system prompt is
// built dynamically by BuildMessages. These tests use realistic history
// shapes: user/assistant/tool only, with tool chains and reasoning content.

func TestFindSafeBoundary_SessionHistoryNoSystem(t *testing.T) {
	// Real session history starts with a user message, not a system message.
	history := []providers.Message{
		msgUser("hello"),               // 0
		msgAssistant("hi there"),       // 1
		msgUser("search for X"),        // 2
		msgAssistantTC("tc1"),          // 3
		msgTool("tc1", "found X"),      // 4
		msgAssistant("here is X"),      // 5
		msgUser("thanks"),              // 6
		msgAssistant("you're welcome"), // 7
	}

	// Mid-point is 4 (tool result). Should snap backward to 2 (user).
	got := findSafeBoundary(history, 4)
	if got != 2 {
		t.Errorf("findSafeBoundary(session_history, 4) = %d, want 2", got)
	}
}

func TestFindSafeBoundary_SessionWithChainedTools(t *testing.T) {
	// Session with chained tool calls (save then notify).
	history := []providers.Message{
		msgUser("save and notify"),       // 0
		msgAssistantTC("tc_save"),        // 1
		msgTool("tc_save", "saved"),      // 2
		msgAssistantTC("tc_notify"),      // 3
		msgTool("tc_notify", "notified"), // 4
		msgAssistant("done"),             // 5
		msgUser("check status"),          // 6
		msgAssistant("all good"),         // 7
	}

	// Target at 3 (inside chain). Should find user at 0, but backward
	// scan stops at i>0, so forward scan finds user at 6.
	// Actually: backward from 3: 2=tool (no), 1=assistantTC (no). Forward: 4=tool, 5=asst, 6=user ✓
	got := findSafeBoundary(history, 3)
	if got != 6 {
		t.Errorf("findSafeBoundary(chained_tools, 3) = %d, want 6", got)
	}
}

func TestEstimateMessageTokens_WithReasoningAndMedia(t *testing.T) {
	// Message with all fields populated — mirrors what AddFullMessage stores.
	msg := providers.Message{
		Role:             "assistant",
		Content:          "Here is the analysis.",
		ReasoningContent: strings.Repeat("Let me think about this carefully. ", 50),
		ToolCalls: []providers.ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Name: "analyze",
				Function: &providers.FunctionCall{
					Name:      "analyze",
					Arguments: `{"data":"sample","depth":3}`,
				},
			},
		},
	}

	tokens := estimateMessageTokens(msg)

	// ReasoningContent alone is ~1700 chars → ~680 tokens.
	// Content + TC + overhead adds more. Should be well above 500.
	if tokens < 500 {
		t.Errorf("message with reasoning+toolcalls should have significant tokens, got %d", tokens)
	}

	// Compare without reasoning to ensure it's counted.
	msgNoReasoning := msg
	msgNoReasoning.ReasoningContent = ""
	tokensNoReasoning := estimateMessageTokens(msgNoReasoning)

	if tokens <= tokensNoReasoning {
		t.Errorf("reasoning content should add tokens: with=%d, without=%d", tokens, tokensNoReasoning)
	}
}

func TestIsOverContextBudget_RealisticSession(t *testing.T) {
	// Simulate what BuildMessages produces: system + session history + current user.
	// System message is built by BuildMessages, not stored in session.
	systemMsg := providers.Message{
		Role:    "system",
		Content: strings.Repeat("system prompt content ", 100),
	}
	sessionHistory := []providers.Message{
		msgUser("first question"),
		msgAssistant("first answer"),
		msgUser("use tool X"),
		{
			Role:    "assistant",
			Content: "I'll use tool X",
			ToolCalls: []providers.ToolCall{
				{
					ID: "tc1", Type: "function", Name: "tool_x",
					Function: &providers.FunctionCall{
						Name:      "tool_x",
						Arguments: `{"query":"test","verbose":true}`,
					},
				},
			},
		},
		{Role: "tool", Content: strings.Repeat("result data ", 200), ToolCallID: "tc1"},
		msgAssistant("Here are the results from tool X."),
	}
	currentUser := msgUser("follow up question")

	// Assemble as BuildMessages would.
	messages := make([]providers.Message, 0, 1+len(sessionHistory)+1)
	messages = append(messages, systemMsg)
	messages = append(messages, sessionHistory...)
	messages = append(messages, currentUser)

	tools := []providers.ToolDefinition{
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "tool_x",
				Description: "A useful tool",
				Parameters:  map[string]any{"type": "object"},
			},
		},
	}

	// With a large context window, should be within budget.
	if isOverContextBudget(131072, messages, tools, 32768) {
		t.Error("realistic session should be within 131072 context window")
	}

	// With a tiny context window, should exceed budget.
	if !isOverContextBudget(500, messages, tools, 32768) {
		t.Error("realistic session should exceed 500 context window")
	}
}
