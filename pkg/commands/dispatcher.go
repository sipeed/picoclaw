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

func firstToken(input string) string {
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func parseCommandName(input string) (string, bool) {
	token := firstToken(input)
	if token == "" || !strings.HasPrefix(token, "/") {
		return "", false
	}

	name := strings.TrimPrefix(token, "/")
	if i := strings.Index(name, "@"); i >= 0 {
		name = name[:i]
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	return name, true
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
