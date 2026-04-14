package commands

import (
	"context"
	"strings"
)

func btwCommand() Definition {
	return Definition{
		Name:        "btw",
		Description: "Ask a side question without changing session history",
		Usage:       "/btw <question>",
		Handler: func(ctx context.Context, req Request, rt *Runtime) error {
			const emptyAnswerMsg = "The model returned an empty response. This may indicate a provider error or token limit."

			if rt == nil || rt.AskSideQuestion == nil {
				return req.Reply(unavailableMsg)
			}

			parts := strings.Fields(strings.TrimSpace(req.Text))
			if len(parts) < 2 {
				return req.Reply("Usage: /btw <question>")
			}
			question := strings.Join(parts[1:], " ")

			answer, err := rt.AskSideQuestion(ctx, question)
			if err != nil {
				return req.Reply(err.Error())
			}
			if strings.TrimSpace(answer) == "" {
				return req.Reply(emptyAnswerMsg)
			}

			return req.Reply(answer)
		},
	}
}
