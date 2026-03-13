package routing

import "fmt"

// BuildSubagentSessionKey returns "subagent:<taskID>" for subagent sessions.
func BuildSubagentSessionKey(taskID string) string {
	return fmt.Sprintf("subagent:%s", taskID)
}
