package command

import (
	"context"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
)

// Registry manages the set of available commands.
type Registry struct {
	commands map[string]Command
}

// NewRegistry creates a new command registry.
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]Command),
	}
}

// Register adds a command to the registry.
func (r *Registry) Register(cmd Command) {
	r.commands[cmd.Name()] = cmd
}

// Execute attempts to execute a command by name.
// Returns response string, handled boolean, and error.
func (r *Registry) Execute(ctx context.Context, agent AgentState, name string, args []string, msg bus.InboundMessage) (string, bool, error) {
	cmd, ok := r.commands[name]
	if !ok {
		return "", false, nil
	}
	resp, err := cmd.Execute(ctx, agent, args, msg)
	return resp, true, err
}

// ListCommands returns a map of registered commands.
func (r *Registry) ListCommands() map[string]Command {
	return r.commands
}

// Parse extracts command and arguments from a message string.
// Returns command name, arguments, and true if it looks like a command.
func (r *Registry) Parse(content string) (string, []string, bool) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "/") {
		return "", nil, false
	}

	parts := strings.Fields(content)
	if len(parts) == 0 {
		return "", nil, false
	}

	// Command name includes the slash, e.g., "/show"
	cmdName := parts[0]
	args := parts[1:]

	return cmdName, args, true
}
