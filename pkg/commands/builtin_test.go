package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/session"
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
	defs := BuiltinDefinitions(nil)
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
	})
	if err != nil {
		t.Fatalf("/help handler error: %v", err)
	}
	if !strings.Contains(reply, "/new - Start a new chat session") {
		t.Fatalf("/help reply missing /new usage, got %q", reply)
	}
	if !strings.Contains(reply, "/session [list|resume <index>] - Manage chat sessions") {
		t.Fatalf("/help reply missing /session usage, got %q", reply)
	}
}

func TestBuiltinDefinitions_SessionCommandsRemainPassthroughWithoutRuntime(t *testing.T) {
	defs := BuiltinDefinitions(nil)

	newDef := findDefinitionByName(t, defs, "new")
	if !contains(newDef.Aliases, "reset") {
		t.Fatalf("/new aliases=%v, want alias reset", newDef.Aliases)
	}
	if newDef.Handler != nil {
		t.Fatalf("/new should remain passthrough without runtime handler")
	}

	sessionDef := findDefinitionByName(t, defs, "session")
	if sessionDef.Handler != nil {
		t.Fatalf("/session should remain passthrough without runtime handler")
	}
}

func TestBuiltinShowChannel_PreservesUserVisibleBehavior(t *testing.T) {
	defs := BuiltinDefinitions(&config.Config{})
	showDef := findDefinitionByName(t, defs, "show")
	if showDef.Handler == nil {
		t.Fatalf("/show handler should not be nil")
	}

	cases := []string{"telegram", "whatsapp"}
	for _, channel := range cases {
		var reply string
		err := showDef.Handler(context.Background(), Request{
			Channel: channel,
			Text:    "/show channel",
			Reply: func(text string) error {
				reply = text
				return nil
			},
		})
		if err != nil {
			t.Fatalf("/show channel handler error on %s: %v", channel, err)
		}
		want := "Current Channel: " + channel
		if reply != want {
			t.Fatalf("/show channel reply=%q, want=%q", reply, want)
		}
	}
}

func TestBuiltinListChannels_UsesConfigEnabledChannels(t *testing.T) {
	cfg := &config.Config{}
	cfg.Channels.Telegram.Enabled = true
	cfg.Channels.Slack.Enabled = true

	defs := BuiltinDefinitions(cfg)
	listDef := findDefinitionByName(t, defs, "list")
	if listDef.Handler == nil {
		t.Fatalf("/list handler should not be nil")
	}

	var reply string
	err := listDef.Handler(context.Background(), Request{
		Text: "/list channels",
		Reply: func(text string) error {
			reply = text
			return nil
		},
	})
	if err != nil {
		t.Fatalf("/list channels handler error: %v", err)
	}
	if !strings.Contains(reply, "telegram") || !strings.Contains(reply, "slack") {
		t.Fatalf("/list channels reply=%q, want telegram and slack", reply)
	}
}

type builtinTestSessionOps struct{}

func (f *builtinTestSessionOps) ResolveActive(scopeKey string) (string, error) { return "", nil }
func (f *builtinTestSessionOps) StartNew(scopeKey string) (string, error)      { return "", nil }
func (f *builtinTestSessionOps) List(scopeKey string) ([]session.SessionMeta, error) {
	return nil, nil
}
func (f *builtinTestSessionOps) Resume(scopeKey string, index int) (string, error) {
	return "", nil
}
func (f *builtinTestSessionOps) Prune(scopeKey string, limit int) ([]string, error) {
	return nil, nil
}

type builtinTestRuntime struct {
	scope string
	ops   SessionOps
}

func (f *builtinTestRuntime) ScopeKey() string       { return f.scope }
func (f *builtinTestRuntime) SessionOps() SessionOps { return f.ops }
func (f *builtinTestRuntime) Config() *config.Config { return nil }

func TestBuiltinDefinitionsWithRuntime_EnablesSessionHandlers(t *testing.T) {
	runtime := &builtinTestRuntime{scope: "scope", ops: &builtinTestSessionOps{}}
	defs := BuiltinDefinitionsWithRuntime(nil, runtime)

	newDef := findDefinitionByName(t, defs, "new")
	sessionDef := findDefinitionByName(t, defs, "session")
	if newDef.Handler == nil {
		t.Fatalf("/new should provide runtime-backed handler when runtime is available")
	}
	if sessionDef.Handler == nil {
		t.Fatalf("/session should provide runtime-backed handler when runtime is available")
	}
}
