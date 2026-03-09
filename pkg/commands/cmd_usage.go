package commands

import (
	"context"
	"fmt"
)

func usageCommand() Definition {
	return Definition{
		Name:        "usage",
		Description: "Show model info and token usage for this session",
		Usage:       "/usage",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.GetModelInfo == nil {
				return req.Reply(unavailableMsg)
			}

			name, _ := rt.GetModelInfo()

			var maxTokens int
			var temperature float64
			if rt.Config != nil {
				maxTokens = rt.Config.Agents.Defaults.MaxTokens
				if rt.Config.Agents.Defaults.Temperature != nil {
					temperature = *rt.Config.Agents.Defaults.Temperature
				}
			}

			msg := fmt.Sprintf("Model: %s\nMax tokens: %d\nTemperature: %.1f",
				name, maxTokens, temperature)

			if rt.GetTokenUsage != nil {
				prompt, completion, requests := rt.GetTokenUsage()
				msg += fmt.Sprintf(`

Token usage (this session):
  Prompt tokens:     %d
  Completion tokens: %d
  Total tokens:      %d
  Requests:          %d`,
					prompt, completion, prompt+completion, requests)
			}

			return req.Reply(msg)
		},
	}
}
