package commands

import (
	"context"
	"strings"
)

type Handler func(ctx context.Context, req Request, rt *Runtime) error

type Request struct {
	Channel  string
	ChatID   string
	SenderID string
	Text     string
	Reply    func(text string) error
}

const unavailableMsg = "Command unavailable in current context."

var commandPrefixes = []string{"/", "!"}

// parseCommandName accepts "/name", "!name", and Telegram's "/name@bot", then
// normalizes to lowercase command names.
func parseCommandName(input string) (string, bool) {
	token := nthToken(input, 0)
	if token == "" {
		return "", false
	}

	name, ok := trimCommandPrefix(token)
	if !ok {
		return "", false
	}
	if i := strings.Index(name, "@"); i >= 0 {
		name = name[:i]
	}
	name = normalizeCommandName(name)
	if name == "" {
		return "", false
	}
	return name, true
}

func trimCommandPrefix(token string) (string, bool) {
	for _, prefix := range commandPrefixes {
		if strings.HasPrefix(token, prefix) {
			return strings.TrimPrefix(token, prefix), true
		}
	}
	return "", false
}

// HasCommandPrefix returns true if the input starts with a recognized
// command prefix (e.g. "/" or "!").
func HasCommandPrefix(input string) bool {
	token := nthToken(input, 0)
	if token == "" {
		return false
	}
	_, ok := trimCommandPrefix(token)
	return ok
}

// nthToken returns the 0-indexed token from whitespace-split input.
func nthToken(input string, n int) string {
	parts := strings.Fields(strings.TrimSpace(input))
	if n >= len(parts) {
		return ""
	}
	return parts[n]
}

func normalizeCommandName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
