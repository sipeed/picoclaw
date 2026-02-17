package tools

import (
	"context"
	"strings"
	"testing"
)

func TestBrowserTool_Name(t *testing.T) {
	tool := NewBrowserTool(BrowserToolOptions{})
	if tool.Name() != "browser" {
		t.Errorf("Expected name 'browser', got %q", tool.Name())
	}
}

func TestBrowserTool_Description(t *testing.T) {
	tool := NewBrowserTool(BrowserToolOptions{})
	desc := tool.Description()
	if !strings.Contains(desc, "agent-browser") {
		t.Error("Description should mention agent-browser")
	}
	if !strings.Contains(desc, "snapshot") {
		t.Error("Description should mention snapshot command")
	}
}

func TestBrowserTool_Parameters(t *testing.T) {
	tool := NewBrowserTool(BrowserToolOptions{})
	params := tool.Parameters()

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties map")
	}

	if _, ok := props["command"]; !ok {
		t.Error("Expected 'command' in properties")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Expected required slice")
	}
	if len(required) != 1 || required[0] != "command" {
		t.Errorf("Expected required=['command'], got %v", required)
	}
}

func TestBrowserTool_MissingCommand(t *testing.T) {
	tool := NewBrowserTool(BrowserToolOptions{})
	ctx := context.Background()

	// Empty args
	result := tool.Execute(ctx, map[string]interface{}{})
	if !result.IsError {
		t.Error("Expected error for missing command")
	}

	// Empty string
	result = tool.Execute(ctx, map[string]interface{}{"command": ""})
	if !result.IsError {
		t.Error("Expected error for empty command")
	}

	// Whitespace only
	result = tool.Execute(ctx, map[string]interface{}{"command": "   "})
	if !result.IsError {
		t.Error("Expected error for whitespace-only command")
	}
}

func TestBrowserTool_BuildArgs(t *testing.T) {
	tests := []struct {
		name     string
		session  string
		command  string
		wantArgs []string
	}{
		{
			name:     "simple command",
			command:  "open https://example.com",
			wantArgs: []string{"--cdp", "9222", "--headed", "--json", "open", "https://example.com"},
		},
		{
			name:     "with session",
			session:  "test-session",
			command:  "snapshot -i",
			wantArgs: []string{"--cdp", "9222", "--session", "test-session", "--headed", "--json", "snapshot", "-i"},
		},
		{
			name:     "quoted arguments",
			command:  `fill @e3 "hello world"`,
			wantArgs: []string{"--cdp", "9222", "--headed", "--json", "fill", "@e3", "hello world"},
		},
		{
			name:     "single quoted",
			command:  `fill @e3 'hello world'`,
			wantArgs: []string{"--cdp", "9222", "--headed", "--json", "fill", "@e3", "hello world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewBrowserTool(BrowserToolOptions{Session: tt.session})
			got := tool.buildArgs(tt.command)

			if len(got) != len(tt.wantArgs) {
				t.Errorf("buildArgs(%q) = %v (len %d), want %v (len %d)",
					tt.command, got, len(got), tt.wantArgs, len(tt.wantArgs))
				return
			}

			for i := range got {
				if got[i] != tt.wantArgs[i] {
					t.Errorf("buildArgs(%q)[%d] = %q, want %q",
						tt.command, i, got[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"open https://example.com", []string{"open", "https://example.com"}},
		{`fill @e3 "test@example.com"`, []string{"fill", "@e3", "test@example.com"}},
		{"snapshot -i -c -d 3", []string{"snapshot", "-i", "-c", "-d", "3"}},
		{`eval "document.title"`, []string{"eval", "document.title"}},
		{"  click   @e2  ", []string{"click", "@e2"}},
		{`get text @e1`, []string{"get", "text", "@e1"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitCommand(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("splitCommand(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitCommand(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}
