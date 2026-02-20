package auth

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNewLoginSubCommand(t *testing.T) {
	cmd := newLoginCommand()

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Short != "Login via OAuth or paste token" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if !cmd.HasFlags() {
		t.Error("expected command to have flags")
	}

	if cmd.Flags().Lookup("device-code") == nil {
		t.Error("expected command to have device-code flag")
	}

	hasProviderFlag := cmd.Flags().Lookup("provider") != nil
	if !hasProviderFlag {
		t.Error("expected command to have provider flag")
	} else {
		var val []string
		var found bool
		providerFlag := cmd.Flag("provider")

		val, found = providerFlag.Annotations[cobra.BashCompOneRequiredFlag]

		if !found || val[0] != "true" {
			t.Errorf("expected provider flag to be required, got %v", val)
		}
	}
}
