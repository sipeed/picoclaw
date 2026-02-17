package command

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
)

type SessionCommand struct{}

func (c *SessionCommand) Name() string {
	return "/session"
}

func (c *SessionCommand) Description() string {
	return "Manage conversation sessions (new, switch, ls, remove)"
}

func (c *SessionCommand) Execute(ctx context.Context, agent AgentState, args []string, msg bus.InboundMessage) (string, error) {
	if len(args) < 1 {
		return "Usage: /session <new|switch|ls|remove> [name]", nil
	}

	subcmd := args[0]
	// Use ChatID from message for unique session base key channel:chatID
	baseKey := fmt.Sprintf("%s:%s", msg.Channel, msg.ChatID)

	// Define interfaces for accessing SessionManager and StateManager
	type SessionManager interface {
		ListSessions(prefix string) []string
		DeleteSession(key string) error
	}

	type StateManager interface {
		SetUserSession(userID, sessionName string) error
		GetUserSession(userID string) string
	}

	type ManagersProvider interface {
		GetSessionManager() interface{}
		GetStateManager() interface{}
	}

	provider, ok := agent.(ManagersProvider)
	if !ok {
		return "Error: Agent does not support session management", nil
	}

	smRaw := provider.GetSessionManager()
	stRaw := provider.GetStateManager()

	sm, okSm := smRaw.(SessionManager)
	st, okSt := stRaw.(StateManager)

	if !okSm || !okSt {
		return "Error: Failed to access internal managers", nil
	}

	switch subcmd {
	case "ls", "list":
		return c.handleList(sm, st, baseKey)
	case "new", "switch":
		if len(args) < 2 {
			return fmt.Sprintf("Usage: /session %s <name>", subcmd), nil
		}
		name := args[1]
		return c.handleSwitch(st, baseKey, name)
	case "remove", "rm", "delete":
		if len(args) < 2 {
			return "Usage: /session remove <name>", nil
		}
		name := args[1]
		return c.handleRemove(sm, st, baseKey, name)
	default:
		return fmt.Sprintf("Unknown subcommand: %s", subcmd), nil
	}
}

func (c *SessionCommand) handleList(sm interface {
	ListSessions(prefix string) []string
}, st interface {
	GetUserSession(userID string) string
}, baseKey string) (string, error) {
	keys := sm.ListSessions(baseKey)
	currentSessionName := st.GetUserSession(baseKey)
	if currentSessionName == "" {
		currentSessionName = "default"
	}

	sessionMap := make(map[string]bool)
	sessionMap["default"] = true // Default always implicitly exists

	for _, key := range keys {
		if key == baseKey {
			sessionMap["default"] = true
		} else if strings.HasPrefix(key, baseKey+":") {
			name := strings.TrimPrefix(key, baseKey+":")
			sessionMap[name] = true
		}
	}

	var names []string
	for name := range sessionMap {
		names = append(names, name)
	}
	sort.Strings(names)

	var sb strings.Builder
	sb.WriteString("Available sessions:\n")
	for _, name := range names {
		marker := " "
		if name == currentSessionName {
			marker = "*"
		}
		sb.WriteString(fmt.Sprintf("%s %s\n", marker, name))
	}

	return sb.String(), nil
}

func (c *SessionCommand) handleSwitch(st interface {
	SetUserSession(userID, sessionName string) error
}, baseKey, name string) (string, error) {
	target := name
	if name == "default" {
		target = "" // empty string denotes default session in state
	}

	if err := st.SetUserSession(baseKey, target); err != nil {
		return "", fmt.Errorf("failed to switch session: %w", err)
	}

	return fmt.Sprintf("Switched to session: %s", name), nil
}

func (c *SessionCommand) handleRemove(sm interface {
	DeleteSession(key string) error
}, st interface {
	GetUserSession(userID string) string
}, baseKey, name string) (string, error) {
	if name == "default" {
		return "Error: Cannot remove default session", nil
	}

	current := st.GetUserSession(baseKey)
	if current == name {
		return "Error: Cannot remove active session. Switch to another session first.", nil
	}

	// Session key format: baseKey:name
	sessionKey := fmt.Sprintf("%s:%s", baseKey, name)
	if err := sm.DeleteSession(sessionKey); err != nil {
		return "", fmt.Errorf("failed to remove session: %w", err)
	}

	return fmt.Sprintf("Session '%s' removed.", name), nil
}
