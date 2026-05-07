package commands

import (
	"context"
	"fmt"
	"strings"
)

func toolFeedbackCommand() Definition {
	return Definition{
		Name:        "toolfeedback",
		Description: "Show or change inline tool feedback for this conversation",
		Usage:       "/toolfeedback [on|off|default]",
		Aliases:     []string{"working"},
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.GetToolFeedback == nil || rt.SetToolFeedback == nil {
				return req.Reply(unavailableMsg)
			}

			arg := strings.ToLower(strings.TrimSpace(nthToken(req.Text, 1)))
			if arg == "" {
				enabled, source := rt.GetToolFeedback()
				return req.Reply(fmt.Sprintf(
					"Tool feedback is currently %s (%s) for this conversation.",
					onOff(enabled),
					source,
				))
			}

			switch arg {
			case "on", "off", "default":
			default:
				return req.Reply("Usage: /toolfeedback [on|off|default]")
			}

			enabled, source, err := rt.SetToolFeedback(arg)
			if err != nil {
				return req.Reply("Failed to update tool feedback: " + err.Error())
			}
			return req.Reply(fmt.Sprintf(
				"Tool feedback is now %s (%s) for this conversation.",
				onOff(enabled),
				source,
			))
		},
	}
}

func onOff(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}
