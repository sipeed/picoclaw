package commands

import (
	"context"
	"fmt"
	"strings"
)

func versionCommand() Definition {
	return Definition{
		Name:        "version",
		Description: "Show version info",
		Usage:       "/version",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.GetVersion == nil {
				return req.Reply("Version info unavailable")
			}
			return req.Reply(fmt.Sprintf("PicoClaw version %s", rt.GetVersion()))
		},
	}
}

func pingCommand() Definition {
	return Definition{
		Name:        "ping",
		Description: "Connectivity check",
		Usage:       "/ping",
		Handler: func(_ context.Context, req Request, _ *Runtime) error {
			return req.Reply("pong")
		},
	}
}

func toolsCommand() Definition {
	return Definition{
		Name:        "tools",
		Description: "List available tools",
		Usage:       "/tools",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.ListTools == nil {
				return req.Reply("Tools list unavailable")
			}
			toolsList := rt.ListTools()
			if len(toolsList) == 0 {
				return req.Reply("No tools available.")
			}
			return req.Reply("Available tools:\n" + strings.Join(toolsList, "\n"))
		},
	}
}

func modelCommand() Definition {
	return Definition{
		Name:        "model",
		Description: "Show or switch the active model",
		Usage:       "/model [name]",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.GetModelInfo == nil {
				return req.Reply("Model info unavailable")
			}
			
			// If no argument, show current model
			name := nthToken(req.Text, 1)
			if name == "" {
				m, p := rt.GetModelInfo()
				return req.Reply(fmt.Sprintf("Current model: %s (Provider: %s)", m, p))
			}

			// If argument, try to switch
			if rt.SwitchModel == nil {
				return req.Reply("Model switching unavailable")
			}
			oldModel, err := rt.SwitchModel(name)
			if err != nil {
				return req.Reply(fmt.Sprintf("Failed to switch model: %v", err))
			}
			return req.Reply(fmt.Sprintf("Switched model from %s to %s", oldModel, name))
		},
	}
}

func vpsCommand() Definition {
	return Definition{
		Name:        "vps",
		Description: "Configure VPS credentials",
		Usage:       "/vps login <password>",
		SubCommands: []SubCommand{
			{
				Name:        "login",
				Description: "Set VPS password securely",
				ArgsUsage:   "<password>",
				Handler: func(_ context.Context, req Request, _ *Runtime) error {
					password := nthToken(req.Text, 2)
					if password == "" {
						return req.Reply("Usage: /vps login <password>")
					}
					cred := &auth.AuthCredential{
						AccessToken: password,
						Provider:    "vps",
						AuthMethod:  "password",
					}
					if err := auth.SetCredential("vps", cred); err != nil {
						return req.Reply(fmt.Sprintf("Failed to save VPS credentials: %v", err))
					}
					return req.Reply("VPS credentials saved securely.")
				},
			},
		},
	}
}
