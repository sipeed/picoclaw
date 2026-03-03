package commands

import (
	"context"
	"strings"
)

type Handler func(ctx context.Context, req Request) error

type Request struct {
	Channel   string
	ChatID    string
	SenderID  string
	Text      string
	MessageID string
	Reply     func(text string) error
}

var commandPrefixes = []string{"/", "!"}

func firstToken(input string) string {
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// parseCommandName accepts "/name", "!name", and Telegram's "/name@bot", then
// normalizes to lowercase command names.
func parseCommandName(input string) (string, bool) {
	token := firstToken(input)
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

func normalizeCommandName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
