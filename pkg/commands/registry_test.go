package commands

import "testing"

func TestRegistry_FilterByChannel(t *testing.T) {
	defs := []Definition{
		{Name: "help", Description: "Show help"},
		{Name: "admin", Description: "Admin only", Channels: []string{"telegram"}},
	}
	r := NewRegistry(defs)

	gotTG := r.ForChannel("telegram")
	if len(gotTG) != 2 {
		t.Fatalf("telegram defs = %d, want 2", len(gotTG))
	}

	gotWA := r.ForChannel("whatsapp")
	if len(gotWA) != 1 || gotWA[0].Name != "help" {
		t.Fatalf("whatsapp defs = %+v, want only help", gotWA)
	}
}
