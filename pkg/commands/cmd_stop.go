package commands

import "context"

func stopCommand() Definition {
	return Definition{
		Name:        "stop",
		Description: "Stop the current running task",
		Usage:       "/stop",
		Strict:      true,
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt.CancelCurrentTask == nil {
				return req.Reply("Stop command is not available in this context.")
			}
			if rt.CancelCurrentTask() {
				// Don't send response here - processMessageAsync will send "Task stopped"
				// when it detects the context cancellation.
				return nil
			}
			return req.Reply("No task is currently running.")
		},
	}
}