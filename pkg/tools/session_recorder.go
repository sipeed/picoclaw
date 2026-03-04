package tools

import "github.com/sipeed/picoclaw/pkg/providers"

// SessionRecorder records session DAG events (fork, turns, completion, report).
// Implemented by pkg/agent to bridge tools → session without circular imports.
type SessionRecorder interface {
	// RecordFork creates a child session forked from the conductor session.
	RecordFork(conductorSessionKey, subagentSessionKey, taskID, label string) error

	// RecordSubagentTurn records the initial + final messages of a subagent run.
	RecordSubagentTurn(subagentSessionKey string, messages []providers.Message) error

	// RecordCompletion marks a subagent session as completed/failed.
	RecordCompletion(subagentSessionKey, status, result string) error

	// RecordReport injects a TurnReport into the conductor session.
	RecordReport(conductorSessionKey, subagentSessionKey, senderID, content string) error

	// RecordQuestion injects a TurnQuestion into the conductor session (subagent escalation).
	RecordQuestion(conductorKey, subagentKey, taskID, question string) error

	// RecordPlanSubmit injects a TurnPlanSubmit into the conductor session (plan review request).
	RecordPlanSubmit(conductorKey, subagentKey, taskID, planText string) error
}
