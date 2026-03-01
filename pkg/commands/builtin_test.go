package commands

import "testing"

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

func TestBuiltinDefinitions_SessionCommandsHaveHandlers(t *testing.T) {
	defs := BuiltinDefinitions(nil)

	defByName := map[string]Definition{}
	for _, def := range defs {
		defByName[def.Name] = def
	}

	newDef, ok := defByName["new"]
	if !ok {
		t.Fatalf("missing /new definition")
	}
	if newDef.Handler == nil {
		t.Fatalf("/new should provide a runtime-backed handler")
	}
	if !contains(newDef.Aliases, "reset") {
		t.Fatalf("/new aliases=%v, want alias \"reset\"", newDef.Aliases)
	}

	sessionDef, ok := defByName["session"]
	if !ok {
		t.Fatalf("missing /session definition")
	}
	if sessionDef.Handler == nil {
		t.Fatalf("/session should provide a runtime-backed handler")
	}
}
