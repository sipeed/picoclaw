package commands

import (
	"context"
	"strings"
	"testing"
)

func findDefinitionByName(t *testing.T, defs []Definition, name string) Definition {
	t.Helper()
	for _, def := range defs {
		if def.Name == name {
			return def
		}
	}
	t.Fatalf("missing /%s definition", name)
	return Definition{}
}

func TestBuiltinHelpHandler_ReturnsFormattedMessage(t *testing.T) {
	defs := BuiltinDefinitions()
	helpDef := findDefinitionByName(t, defs, "help")
	if helpDef.Handler == nil {
		t.Fatalf("/help handler should not be nil")
	}

	var reply string
	err := helpDef.Handler(context.Background(), Request{
		Text: "/help",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	}, nil)
	if err != nil {
		t.Fatalf("/help handler error: %v", err)
	}
	// Now uses auto-generated EffectiveUsage which includes agents
	if !strings.Contains(reply, "/show [model|channel|agents|mcp <server>]") {
		t.Fatalf("/help reply missing /show usage, got %q", reply)
	}
	if !strings.Contains(reply, "/list [models|channels|agents|skills|mcp]") {
		t.Fatalf("/help reply missing /list usage, got %q", reply)
	}
	if !strings.Contains(reply, "/use <skill> <message>") {
		if !strings.Contains(reply, "/use <skill> [message]") {
			t.Fatalf("/help reply missing /use usage, got %q", reply)
		}
	}
}

func TestBuiltinShowChannel_PreservesUserVisibleBehavior(t *testing.T) {
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), nil)

	cases := []string{"telegram", "whatsapp"}
	for _, channel := range cases {
		var reply string
		res := ex.Execute(context.Background(), Request{
			Channel: channel,
			Text:    "/show channel",
			Reply: func(text string) error {
				reply = text
				return nil
			},
		})
		if res.Outcome != OutcomeHandled {
			t.Fatalf("/show channel on %s: outcome=%v, want=%v", channel, res.Outcome, OutcomeHandled)
		}
		want := "Current Channel: " + channel
		if reply != want {
			t.Fatalf("/show channel reply=%q, want=%q", reply, want)
		}
	}
}

func TestBuiltinListChannels_UsesGetEnabledChannels(t *testing.T) {
	rt := &Runtime{
		GetEnabledChannels: func() []string {
			return []string{"telegram", "slack"}
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list channels",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list channels: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "telegram") || !strings.Contains(reply, "slack") {
		t.Fatalf("/list channels reply=%q, want telegram and slack", reply)
	}
}

func TestBuiltinShowAgents_RestoresOldBehavior(t *testing.T) {
	rt := &Runtime{
		ListAgentIDs: func() []string {
			return []string{"default", "coder"}
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/show agents",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/show agents: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "default") || !strings.Contains(reply, "coder") {
		t.Fatalf("/show agents reply=%q, want agent IDs", reply)
	}
}

func TestBuiltinListAgents_RestoresOldBehavior(t *testing.T) {
	rt := &Runtime{
		ListAgentIDs: func() []string {
			return []string{"default", "coder"}
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list agents",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list agents: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "default") || !strings.Contains(reply, "coder") {
		t.Fatalf("/list agents reply=%q, want agent IDs", reply)
	}
}

func TestBuiltinListSkills_UsesRuntimeSkillNames(t *testing.T) {
	rt := &Runtime{
		ListSkillNames: func() []string {
			return []string{"shell", "git"}
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list skills",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list skills: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "shell") || !strings.Contains(reply, "git") {
		t.Fatalf("/list skills reply=%q, want installed skill names", reply)
	}
}

func TestBuiltinListMCP_UsesRuntimeServerStatus(t *testing.T) {
	rt := &Runtime{
		ListMCPServers: func(context.Context) []MCPServerInfo {
			return []MCPServerInfo{
				{Name: "filesystem", Enabled: true, Deferred: true, Connected: false},
				{Name: "github", Enabled: true, Deferred: false, Connected: true, ToolCount: 3},
			}
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/list mcp",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/list mcp: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "- `filesystem`\n  Enabled: yes\n  Deferred: yes\n  "+
		"Connected: no\n  Active tools: unavailable") {
		t.Fatalf("/list mcp reply=%q, want formatted filesystem block", reply)
	}
	if !strings.Contains(reply, "- `github`\n  Enabled: yes\n  Deferred: no\n  "+
		"Connected: yes\n  Active tools: 3") {
		t.Fatalf("/list mcp reply=%q, want formatted github block", reply)
	}
}

func TestBuiltinShowMCP_UsesRuntimeToolNames(t *testing.T) {
	rt := &Runtime{
		ListMCPTools: func(_ context.Context, serverName string) ([]MCPToolInfo, error) {
			if serverName != "github" {
				t.Fatalf("serverName=%q, want github", serverName)
			}
			return []MCPToolInfo{
				{
					Name:        "create_issue",
					Description: "Create a GitHub issue",
					Parameters: []MCPToolParameterInfo{
						{Name: "body", Type: "string", Description: "Issue body"},
						{Name: "title", Type: "string", Description: "Issue title", Required: true},
					},
				},
				{
					Name:        "list_prs",
					Description: "List open pull requests",
				},
			}, nil
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/show mcp github",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/show mcp: outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if !strings.Contains(reply, "Active MCP tools for `github`:\n- `create_issue`") {
		t.Fatalf("/show mcp reply=%q, want tool header", reply)
	}
	if !strings.Contains(reply, "Description: Create a GitHub issue") {
		t.Fatalf("/show mcp reply=%q, want description", reply)
	}
	if !strings.Contains(reply, "    - `title` (string, required): Issue title") {
		t.Fatalf("/show mcp reply=%q, want required parameter", reply)
	}
	if !strings.Contains(reply, "    - `body` (string): Issue body") {
		t.Fatalf("/show mcp reply=%q, want optional parameter", reply)
	}
	if !strings.Contains(reply, "- `list_prs`\n  Description: List open pull requests\n  Parameters: none") {
		t.Fatalf("/show mcp reply=%q, want empty parameter block", reply)
	}
}

func TestBuiltinUseCommand_PassthroughsToAgentLogic(t *testing.T) {
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), nil)

	res := ex.Execute(context.Background(), Request{
		Text: "/use shell run ls",
	})
	if res.Outcome != OutcomePassthrough {
		t.Fatalf("/use outcome=%v, want=%v", res.Outcome, OutcomePassthrough)
	}
	if res.Command != "use" {
		t.Fatalf("/use command=%q, want=%q", res.Command, "use")
	}
}

func TestBuiltinBtwCommand_UsesSideQuestionRuntime(t *testing.T) {
	rt := &Runtime{
		AskSideQuestion: func(ctx context.Context, question string) (string, error) {
			if question != "what is 2+2?" {
				t.Fatalf("question=%q, want %q", question, "what is 2+2?")
			}
			return "4", nil
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/btw what is 2+2?",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/btw outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "4" {
		t.Fatalf("/btw reply=%q, want=%q", reply, "4")
	}
}

func TestBuiltinBtwCommand_MissingQuestion(t *testing.T) {
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), &Runtime{
		AskSideQuestion: func(context.Context, string) (string, error) {
			return "", nil
		},
	})

	var reply string
	res := ex.Execute(context.Background(), Request{
		Text: "/btw",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/btw outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
	if reply != "Usage: /btw <question>" {
		t.Fatalf("/btw reply=%q, want usage message", reply)
	}
}

func TestBuiltinBtwCommand_PreservesQuestionWhitespace(t *testing.T) {
	const want = "explain:\n    fmt.Println(\"hi\")"
	rt := &Runtime{
		AskSideQuestion: func(ctx context.Context, question string) (string, error) {
			if question != want {
				t.Fatalf("question=%q, want %q", question, want)
			}
			return "ok", nil
		},
	}
	defs := BuiltinDefinitions()
	ex := NewExecutor(NewRegistry(defs), rt)

	res := ex.Execute(context.Background(), Request{
		Text: "/btw " + want,
		Reply: func(text string) error {
			return nil
		},
	})
	if res.Outcome != OutcomeHandled {
		t.Fatalf("/btw outcome=%v, want=%v", res.Outcome, OutcomeHandled)
	}
}
