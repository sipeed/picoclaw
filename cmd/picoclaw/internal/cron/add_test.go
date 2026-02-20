package cron

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNewAddSubcommand(t *testing.T) {
	fn := func() string { return "" }
	cmd := newAddCommand(fn)

	if cmd == nil {
		t.Fatalf("expected non-nil command")
	}

	if cmd.Use != "add" {
		t.Errorf("expected command name 'add', got %q", cmd.Use)
	}

	if cmd.Short != "Add a new scheduled job" {
		t.Errorf("expected command short description, got %q", cmd.Short)
	}

	if !cmd.HasFlags() {
		t.Error("expected command to have flags")
	}

	if cmd.Flags().Lookup("every") == nil {
		t.Error("expected command to have every flag")
	}

	if cmd.Flags().Lookup("cron") == nil {
		t.Error("expected command to have cron flag")
	}

	if cmd.Flags().Lookup("deliver") == nil {
		t.Error("expected command to have deliver flag")
	}

	if cmd.Flags().Lookup("to") == nil {
		t.Error("expected command to have to flag")
	}

	if cmd.Flags().Lookup("channel") == nil {
		t.Error("expected command to have channel flag")
	}

	hasNameFlag := cmd.Flags().Lookup("name") != nil
	hasMessageFlag := cmd.Flags().Lookup("message") != nil

	if !hasNameFlag {
		t.Error("expected command to have name flag")
	}

	if !hasMessageFlag {
		t.Error("expected command to have message flag")
	}

	if hasNameFlag {
		var val []string
		var found bool
		nameFlag := cmd.Flag("name")

		val, found = nameFlag.Annotations[cobra.BashCompOneRequiredFlag]

		if !found || val[0] != "true" {
			t.Errorf("expected name flag to be required, got %v", val)
		}
	}

	if hasMessageFlag {
		var val []string
		var found bool
		messageFlag := cmd.Flag("message")

		val, found = messageFlag.Annotations[cobra.BashCompOneRequiredFlag]

		if !found || val[0] != "true" {
			t.Errorf("expected message flag to be required, got %v", val)
		}
	}
}
