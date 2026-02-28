package orch

// AgentState is a typed string representing the lifecycle state of an agent session.
// Values are sent as-is over the WebSocket event stream to the Mini App canvas.
type AgentState string

const (
	// AgentStateIdle is the resting state, set automatically by Broadcaster on spawn.
	AgentStateIdle AgentState = "idle"

	// AgentStateWaiting means the agent is waiting for an LLM response.
	AgentStateWaiting AgentState = "waiting"

	// AgentStateToolCall means the agent is executing a tool.
	// The tool name is carried in the Tool field of the event.
	AgentStateToolCall AgentState = "toolcall"

	// AgentStatePlanInterviewing means the conductor is in plan-mode interview phase,
	// clarifying goals and constraints with the user.
	AgentStatePlanInterviewing AgentState = "plan_interviewing"

	// AgentStatePlanReview means the conductor has submitted a plan and is waiting
	// for user approval before executing.
	AgentStatePlanReview AgentState = "plan_review"

	// AgentStatePlanExecuting means the conductor is executing an approved plan.
	AgentStatePlanExecuting AgentState = "plan_executing"

	// AgentStatePlanCompleted means all plan steps have been completed.
	AgentStatePlanCompleted AgentState = "plan_completed"
)
