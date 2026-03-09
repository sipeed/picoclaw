package commands

import (
	"context"
	"strings"
)

func cmdModeCommand() Definition {
	return Definition{
		Name:        "cmd",
		Description: "Switch to command mode (execute shell commands)",
		Usage:       "/cmd",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.SetModeCmd == nil {
				return req.Reply(unavailableMsg)
			}
			return req.Reply(rt.SetModeCmd())
		},
	}
}

func picoModeCommand() Definition {
	return Definition{
		Name:        "pico",
		Description: "Switch to chat mode (AI conversation)",
		Usage:       "/pico",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.SetModePico == nil {
				return req.Reply(unavailableMsg)
			}
			return req.Reply(rt.SetModePico())
		},
	}
}

func hipicoCmnd() Definition {
	return Definition{
		Name:        "hipico",
		Description: "Ask AI for one-shot help (works from command mode)",
		Usage:       "/hipico <message>",
		Handler: func(ctx context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.RunOneShot == nil {
				return req.Reply(unavailableMsg)
			}
			// Strip the command prefix to get the message body
			msg := strings.TrimSpace(req.Text)
			for _, prefix := range []string{"/hipico", "!hipico"} {
				if strings.HasPrefix(msg, prefix) {
					msg = strings.TrimSpace(msg[len(prefix):])
					break
				}
			}
			if msg == "" {
				return req.Reply("👋 Hi~ What can I help you with?\n\n💡 Examples:\n  /hipico what files are in the working directory?\n  /hipico what time is it now?\n  /hipico what's the progress on the previous task?")
			}
			reply, err := rt.RunOneShot(ctx, msg)
			if err != nil {
				return req.Reply("Error: " + err.Error())
			}
			return req.Reply(reply)
		},
	}
}
