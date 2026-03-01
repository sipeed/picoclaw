package channels

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/commands"
)

type mockRegistrar struct{}

func (mockRegistrar) RegisterCommands(context.Context, []commands.Definition) error { return nil }

type mockParser struct{}

func (mockParser) DispatchCommand(context.Context, commands.Request) commands.Result {
	return commands.Result{Matched: false}
}

func TestCommandRegistrarCapable_Compiles(t *testing.T) {
	var _ CommandRegistrarCapable = mockRegistrar{}
}

func TestCommandParserCapable_Compiles(t *testing.T) {
	var _ CommandParserCapable = mockParser{}
}
