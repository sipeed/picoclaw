package commands

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/session"
)

func TestBuiltinDefinitions_ContainsTelegramDefaults(t *testing.T) {
	defs := BuiltinDefinitions(nil)
	names := map[string]bool{}
	for _, d := range defs {
		names[d.Name] = true
	}
	for _, want := range []string{"help", "start", "new", "session", "show", "list"} {
		if !names[want] {
			t.Fatalf("missing command %q", want)
		}
	}
}

func TestBuiltinDefinitions_WhatsAppOnlyHasBasicCommands(t *testing.T) {
	defs := NewRegistry(BuiltinDefinitions(nil)).ForChannel("whatsapp")
	names := map[string]bool{}
	for _, d := range defs {
		names[d.Name] = true
	}
	if !names["start"] || !names["help"] {
		t.Fatalf("whatsapp should include start/help, got %+v", names)
	}
	if !names["new"] || !names["session"] {
		t.Fatalf("whatsapp should include new/session, got %+v", names)
	}
	if names["show"] || names["list"] {
		t.Fatalf("whatsapp should not include show/list, got %+v", names)
	}
}

func TestBuiltinDefinitions_CLIHasSessionCommands(t *testing.T) {
	defs := NewRegistry(BuiltinDefinitions(nil)).ForChannel("cli")
	names := map[string]bool{}
	for _, d := range defs {
		names[d.Name] = true
	}
	if !names["new"] || !names["session"] {
		t.Fatalf("cli should include new/session, got %+v", names)
	}
	if names["show"] || names["list"] {
		t.Fatalf("cli should not include show/list, got %+v", names)
	}
}

func TestBuiltinDefinitions_DefaultSessionCommandsArePassthrough(t *testing.T) {
	defs := BuiltinDefinitions(nil)

	defByName := map[string]Definition{}
	for _, def := range defs {
		defByName[def.Name] = def
	}

	newDef, ok := defByName["new"]
	if !ok {
		t.Fatalf("missing /new definition")
	}
	if newDef.Handler != nil {
		t.Fatalf("/new should be passthrough without runtime wiring")
	}
	if !contains(newDef.Aliases, "reset") {
		t.Fatalf("/new aliases=%v, want alias \"reset\"", newDef.Aliases)
	}

	sessionDef, ok := defByName["session"]
	if !ok {
		t.Fatalf("missing /session definition")
	}
	if sessionDef.Handler != nil {
		t.Fatalf("/session should be passthrough without runtime wiring")
	}
}

type builtinTestSessionOps struct{}

func (f *builtinTestSessionOps) ResolveActive(scopeKey string) (string, error) { return "", nil }
func (f *builtinTestSessionOps) StartNew(scopeKey string) (string, error)      { return "", nil }
func (f *builtinTestSessionOps) List(scopeKey string) ([]session.SessionMeta, error) {
	return nil, nil
}
func (f *builtinTestSessionOps) Resume(scopeKey string, index int) (string, error)  { return "", nil }
func (f *builtinTestSessionOps) Prune(scopeKey string, limit int) ([]string, error) { return nil, nil }

type builtinTestRuntime struct {
	scope string
	ops   SessionOps
}

func (f *builtinTestRuntime) Channel() string        { return "whatsapp" }
func (f *builtinTestRuntime) ScopeKey() string       { return f.scope }
func (f *builtinTestRuntime) SessionOps() SessionOps { return f.ops }
func (f *builtinTestRuntime) Config() *config.Config { return nil }

func TestBuiltinDefinitionsWithRuntime_EnablesSessionHandlers(t *testing.T) {
	runtime := &builtinTestRuntime{scope: "scope", ops: &builtinTestSessionOps{}}
	defs := BuiltinDefinitionsWithRuntime(nil, runtime)

	defByName := map[string]Definition{}
	for _, def := range defs {
		defByName[def.Name] = def
	}

	if defByName["new"].Handler == nil {
		t.Fatalf("/new should provide runtime-backed handler when runtime is available")
	}
	if defByName["session"].Handler == nil {
		t.Fatalf("/session should provide runtime-backed handler when runtime is available")
	}
}

func TestBuiltinDefinitions_ShowAndListAreTelegramOnlyHandlers(t *testing.T) {
	defs := BuiltinDefinitions(&config.Config{})

	defByName := map[string]Definition{}
	for _, def := range defs {
		defByName[def.Name] = def
	}

	for _, name := range []string{"show", "list"} {
		def, ok := defByName[name]
		if !ok {
			t.Fatalf("missing /%s definition", name)
		}
		if def.Handler == nil {
			t.Fatalf("/%s should provide a builtin handler", name)
		}
		if len(def.Channels) != 1 || def.Channels[0] != "telegram" {
			t.Fatalf("/%s channels=%v, want [telegram]", name, def.Channels)
		}
	}
}
