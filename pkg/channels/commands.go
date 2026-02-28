package channels

import (
	"context"
	"sort"
	"sync"

	"github.com/sipeed/picoclaw/pkg/bus"
)

// CommandHandler processes a slash command and returns a text response.
// args contains everything after the command name (e.g. for "/nodes verbose", args = "verbose").
// msg provides the full inbound message context (channel, sender, chat, etc.).
type CommandHandler func(ctx context.Context, args string, msg bus.InboundMessage) (string, error)

// CommandEntry holds a registered command.
type CommandEntry struct {
	Name        string         // command name without leading slash, e.g. "nodes"
	Description string         // human-readable description, e.g. "List swarm cluster nodes"
	Handler     CommandHandler // function to execute
}

// CommandRegistry is a thread-safe registry of slash commands.
// External modules register commands here; the agent loop checks it when
// processing inbound messages that start with "/".
type CommandRegistry struct {
	mu       sync.RWMutex
	commands map[string]*CommandEntry
}

// NewCommandRegistry creates an empty command registry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*CommandEntry),
	}
}

// Register adds or replaces a command in the registry.
// name should NOT include the leading slash.
func (r *CommandRegistry) Register(name, description string, handler CommandHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[name] = &CommandEntry{
		Name:        name,
		Description: description,
		Handler:     handler,
	}
}

// Get looks up a command by name. Returns nil, false if not found.
func (r *CommandRegistry) Get(name string) (*CommandEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.commands[name]
	return entry, ok
}

// List returns all registered commands sorted by name.
func (r *CommandRegistry) List() []*CommandEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entries := make([]*CommandEntry, 0, len(r.commands))
	for _, e := range r.commands {
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return entries
}

// Remove unregisters a command by name. No-op if not found.
func (r *CommandRegistry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.commands, name)
}
