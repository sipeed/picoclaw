package providers

import (
	"strings"
	"testing"
)

func TestExtractXMLToolCalls_Single(t *testing.T) {
	text := `<vendor:toolcall>
<invoke name="exec">
<parameter name="command">echo hello</parameter>
</invoke>
</vendor:toolcall>`

	calls := extractXMLToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].Name != "exec" {
		t.Errorf("Name = %q, want %q", calls[0].Name, "exec")
	}
	if calls[0].Arguments["command"] != "echo hello" {
		t.Errorf("Arguments[command] = %v, want %q", calls[0].Arguments["command"], "echo hello")
	}
	if calls[0].Function == nil || calls[0].Function.Name != "exec" {
		t.Errorf("Function.Name should be exec")
	}
}

func TestExtractXMLToolCalls_Multiple(t *testing.T) {
	text := `<vendor:toolcall>
<invoke name="web_search">
<parameter name="query">golang testing</parameter>
</invoke>
<invoke name="exec">
<parameter name="command">go test ./...</parameter>
<parameter name="timeout">30</parameter>
</invoke>
</vendor:toolcall>`

	calls := extractXMLToolCalls(text)
	if len(calls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(calls))
	}
	if calls[0].Name != "web_search" {
		t.Errorf("[0].Name = %q, want %q", calls[0].Name, "web_search")
	}
	if calls[1].Name != "exec" {
		t.Errorf("[1].Name = %q, want %q", calls[1].Name, "exec")
	}
	if calls[1].Arguments["timeout"] != "30" {
		t.Errorf("[1].Arguments[timeout] = %v, want %q", calls[1].Arguments["timeout"], "30")
	}
}

func TestExtractXMLToolCalls_NoXML(t *testing.T) {
	calls := extractXMLToolCalls("just regular text")
	if len(calls) != 0 {
		t.Errorf("expected 0 tool calls, got %d", len(calls))
	}
}

func TestStripXMLToolCalls(t *testing.T) {
	text := `Let me run that.
<vendor:toolcall>
<invoke name="exec">
<parameter name="command">echo hello</parameter>
</invoke>
</vendor:toolcall>
Done.`

	got := stripXMLToolCalls(text)
	if strings.Contains(got, "toolcall") {
		t.Errorf("should remove XML block, got %q", got)
	}
	if !strings.Contains(got, "Let me run that.") {
		t.Errorf("should keep text before, got %q", got)
	}
	if !strings.Contains(got, "Done.") {
		t.Errorf("should keep text after, got %q", got)
	}
}

func TestExtractXMLToolCalls_MismatchedCloseTag(t *testing.T) {

	text := `<minimax:toolcall>
<invoke name="readfile">
<parameter name="path">/home/user/project/pyproject.toml</parameter>
</invoke>
</minimax:tool_call>`

	calls := extractXMLToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].Name != "readfile" {
		t.Errorf("Name = %q, want %q", calls[0].Name, "readfile")
	}
	if calls[0].Arguments["path"] != "/home/user/project/pyproject.toml" {
		t.Errorf("Arguments[path] = %v, want pyproject.toml path", calls[0].Arguments["path"])
	}
}

func TestStripXMLToolCalls_MismatchedCloseTag(t *testing.T) {
	text := `今テスト走らせるね。` +
		`
<minimax:toolcall>
<invoke name="exec">
<parameter name="command">cd /home/user && pytest</parameter>
</invoke>
</minimax:tool_call>`

	got := stripXMLToolCalls(text)
	if strings.Contains(got, "toolcall") || strings.Contains(got, "tool_call") {
		t.Errorf("should remove XML block, got %q", got)
	}
	if !strings.Contains(got, "今テスト走らせるね。") {
		t.Errorf("should keep text before, got %q", got)
	}
}

func TestExtractXMLToolCalls_UnderscoreOpenTag(t *testing.T) {

	text := `<minimax:tool_call>
<invoke name="exec">
<parameter name="command">ls -la</parameter>
</invoke>
</minimax:tool_call>`

	calls := extractXMLToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].Name != "exec" {
		t.Errorf("Name = %q, want %q", calls[0].Name, "exec")
	}
	if calls[0].Arguments["command"] != "ls -la" {
		t.Errorf("Arguments[command] = %v, want %q", calls[0].Arguments["command"], "ls -la")
	}
}

func TestExtractXMLToolCalls_HyphenTag(t *testing.T) {

	text := `<vendor:Tool-Call>
<invoke name="read_file">
<parameter name="path">/etc/hosts</parameter>
</invoke>
</vendor:tool-call>`

	calls := extractXMLToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].Name != "read_file" {
		t.Errorf("Name = %q, want %q", calls[0].Name, "read_file")
	}
}

func TestStripXMLToolCalls_UnderscoreOpenTag(t *testing.T) {
	text := `Here is the result.
<minimax:tool_call>
<invoke name="exec">
<parameter name="command">ls</parameter>
</invoke>
</minimax:toolcall>
Finished.`

	got := stripXMLToolCalls(text)
	if strings.Contains(got, "tool_call") || strings.Contains(got, "toolcall") {
		t.Errorf("should remove XML block, got %q", got)
	}
	if !strings.Contains(got, "Here is the result.") {
		t.Errorf("should keep text before, got %q", got)
	}
	if !strings.Contains(got, "Finished.") {
		t.Errorf("should keep text after, got %q", got)
	}
}

func TestExtractXMLToolCalls_OrphanedClosingTag(t *testing.T) {

	text := "了解！確認するね。\n[TOOLCALL]\n<invoke name=\"listdir\">\n<parameter name=\"path\">/home/user/workspace</parameter>\n</invoke>\n</minimax:tool_call>"

	calls := extractXMLToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].Name != "listdir" {
		t.Errorf("Name = %q, want %q", calls[0].Name, "listdir")
	}
	if calls[0].Arguments["path"] != "/home/user/workspace" {
		t.Errorf("Arguments[path] = %v, want /home/user/workspace", calls[0].Arguments["path"])
	}
}

func TestStripXMLToolCalls_OrphanedClosingTag(t *testing.T) {
	text := "了解！確認するね。\n[TOOLCALL]\n<invoke name=\"listdir\">\n<parameter name=\"path\">/home/user</parameter>\n</invoke>\n</minimax:tool_call>"
	got := stripXMLToolCalls(text)
	if strings.Contains(got, "invoke") || strings.Contains(got, "TOOLCALL") || strings.Contains(got, "minimax") {
		t.Errorf("should remove orphaned closing tag block, got %q", got)
	}
	if !strings.Contains(got, "了解") {
		t.Errorf("should keep user-facing text, got %q", got)
	}
}

func TestStripXMLToolCalls_NoXML(t *testing.T) {
	text := "Just regular text."
	got := stripXMLToolCalls(text)
	if got != text {
		t.Errorf("stripXMLToolCalls() = %q, want %q", got, text)
	}
}

func TestNormalizeAlpha(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"toolcall", "toolcall"},
		{"tool_call", "toolcall"},
		{"Tool-Call", "toolcall"},
		{"ReadFile", "readfile"},
		{"read_file", "readfile"},
		{"EXEC", "exec"},
		{"web123search", "websearch"},
		{"", ""},
	}
	for _, tt := range tests {
		got := normalizeAlpha(tt.input)
		if got != tt.want {
			t.Errorf("normalizeAlpha(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"toolcall", "toolcall", 0},
		{"toolcall", "tool_call", 1},
		{"toolcall", "tool-call", 1},
		{"toolcall", "ToolCall", 2},
		{"kitten", "sitting", 3},
	}
	for _, tt := range tests {
		got := levenshtein(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestIsToolCallTag(t *testing.T) {

	for _, name := range []string{"toolcall", "tool_call", "tool-call", "ToolCall", "Toolcall", "toolCall", "TOOLCALL"} {
		if !isToolCallTag(name) {
			t.Errorf("isToolCallTag(%q) = false, want true", name)
		}
	}

	for _, name := range []string{"function_call", "FunctionCall", "functioncall", "FUNCTION_CALL"} {
		if !isToolCallTag(name) {
			t.Errorf("isToolCallTag(%q) = false, want true", name)
		}
	}

	for _, name := range []string{"tool_use", "ToolUse", "tooluse", "TOOL_USE"} {
		if !isToolCallTag(name) {
			t.Errorf("isToolCallTag(%q) = false, want true", name)
		}
	}

	for _, name := range []string{"invoke", "parameter", "function", "result", "hello", "content"} {
		if isToolCallTag(name) {
			t.Errorf("isToolCallTag(%q) = true, want false", name)
		}
	}
}
