package commands

import (
	"context"
	"fmt"
	"strings"
)

func boostCommand() Definition {
	return Definition{
		Name:        "boost",
		Description: "Use the paid model for your next message",
		Usage:       "/boost",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			paidModel, _ := sessionModelNames(rt)
			if paidModel == "" {
				return req.Reply("Boost unavailable: paid model is not configured.")
			}
			if rt == nil || rt.ArmNextModelMode == nil {
				return req.Reply(unavailableMsg)
			}
			if err := rt.ArmNextModelMode(paidModel); err != nil {
				return req.Reply(err.Error())
			}
			return req.Reply(fmt.Sprintf("Boost armed. Next message will use %s.", paidModel))
		},
	}
}

func paidCommand() Definition {
	return Definition{
		Name:        "paid",
		Description: "Use the paid model for this session",
		Usage:       "/paid",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			paidModel, _ := sessionModelNames(rt)
			if paidModel == "" {
				return req.Reply("Paid mode unavailable: primary model is not configured.")
			}
			if rt == nil || rt.SetSessionModelMode == nil {
				return req.Reply(unavailableMsg)
			}
			if err := rt.SetSessionModelMode(paidModel); err != nil {
				return req.Reply(err.Error())
			}
			return req.Reply(fmt.Sprintf("Session mode set to paid (%s).", paidModel))
		},
	}
}

func freeCommand() Definition {
	return Definition{
		Name:        "free",
		Description: "Use the free model for this session",
		Usage:       "/free",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			_, freeModel := sessionModelNames(rt)
			if freeModel == "" {
				return req.Reply("Free mode unavailable: light model is not configured.")
			}
			if rt == nil || rt.SetSessionModelMode == nil {
				return req.Reply(unavailableMsg)
			}
			if err := rt.SetSessionModelMode(freeModel); err != nil {
				return req.Reply(err.Error())
			}
			return req.Reply(fmt.Sprintf("Session mode set to free (%s).", freeModel))
		},
	}
}

func statusCommand() Definition {
	return Definition{
		Name:        "status",
		Description: "Show the current session model mode",
		Usage:       "/status",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil {
				return req.Reply(unavailableMsg)
			}

			currentModel, provider := "", ""
			if rt.GetModelInfo != nil {
				currentModel, provider = rt.GetModelInfo()
			}

			paidModel, freeModel := sessionModelNames(rt)
			persistent, pending := "", ""
			if rt.GetSessionModelMode != nil {
				persistent, pending = rt.GetSessionModelMode()
			}

			lines := make([]string, 0, 5)
			if currentModel != "" {
				if provider != "" {
					lines = append(lines, fmt.Sprintf("Current Model: %s (Provider: %s)", currentModel, provider))
				} else {
					lines = append(lines, fmt.Sprintf("Current Model: %s", currentModel))
				}
			}
			lines = append(lines, fmt.Sprintf("Session Mode: %s", sessionModeDescription(persistent, pending, paidModel, freeModel)))
			if pending != "" {
				lines = append(lines, fmt.Sprintf("Pending Boost: %s", pending))
			} else {
				lines = append(lines, "Pending Boost: none")
			}
			if paidModel != "" {
				lines = append(lines, fmt.Sprintf("Paid Model: %s", paidModel))
			}
			if freeModel != "" {
				lines = append(lines, fmt.Sprintf("Free Model: %s", freeModel))
			}
			return req.Reply(strings.Join(lines, "\n"))
		},
	}
}

func sessionModelNames(rt *Runtime) (paidModel, freeModel string) {
	if rt == nil || rt.Config == nil {
		return "", ""
	}

	paidModel = strings.TrimSpace(rt.Config.Agents.Defaults.ModelName)
	if rt.Config.Agents.Defaults.Routing != nil {
		freeModel = strings.TrimSpace(rt.Config.Agents.Defaults.Routing.LightModel)
	}
	return paidModel, freeModel
}

func sessionModeDescription(persistent, pending, paidModel, freeModel string) string {
	if pending != "" {
		return fmt.Sprintf("boost armed for next message (%s)", pending)
	}
	if persistent == "" {
		return "route (default)"
	}
	if paidModel != "" && strings.EqualFold(persistent, paidModel) {
		return fmt.Sprintf("paid (%s)", persistent)
	}
	if freeModel != "" && strings.EqualFold(persistent, freeModel) {
		return fmt.Sprintf("free (%s)", persistent)
	}
	return fmt.Sprintf("custom (%s)", persistent)
}
