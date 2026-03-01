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
