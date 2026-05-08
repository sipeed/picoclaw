package commands

import (
	"context"
	"fmt"
	"strings"
)

func stopCommand() Definition {
	return Definition{
		Name:        "stop",
		Description: "Stop the current task",
		Usage:       "/stop",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.StopActiveTurn == nil {
				return req.Reply(unavailableMsg)
			}

			result, err := rt.StopActiveTurn()
			if err != nil {
				return req.Reply("Failed to stop task: " + err.Error())
			}

			return req.Reply(FormatStopReply(result))
		},
	}
}

// FormatStopReply renders a user-facing reply for a stop request.
func FormatStopReply(result StopResult) string {
	if !result.Stopped {
		return "No active task to stop."
	}

	taskName := compactStopTaskName(result.TaskName)
	if taskName == "" {
		return "Task stopped. Current task was canceled."
	}

	return fmt.Sprintf("Task stopped. %q was canceled.", taskName)
}

func compactStopTaskName(taskName string) string {
	taskName = strings.Join(strings.Fields(strings.TrimSpace(taskName)), " ")
	if taskName == "" {
		return ""
	}
	if len(taskName) > 80 {
		return taskName[:77] + "..."
	}
	return taskName
}
