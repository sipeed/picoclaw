package telegram

import "strings"

// rewriteModelShortcut normalizes Telegram-specific /models shortcuts into
// existing cross-channel commands handled by the command runtime.
func rewriteModelShortcut(input, botUsername string) string {
	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) == 0 {
		return input
	}

	token := parts[0]
	if !strings.HasPrefix(token, "/") {
		return input
	}

	name, target, hasTarget := strings.Cut(strings.TrimPrefix(token, "/"), "@")
	if !strings.EqualFold(name, "models") {
		return input
	}
	if hasTarget && (botUsername == "" || !strings.EqualFold(target, botUsername)) {
		return input
	}

	if len(parts) == 1 {
		return "/list models"
	}

	return "/switch model to " + strings.Join(parts[1:], " ")
}
