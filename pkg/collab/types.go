package collab

import (
	"slices"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers"
)

type MessageKind string

const (
	KindRequest MessageKind = "request"
	KindReply   MessageKind = "reply"
	KindNotice  MessageKind = "notice"
	KindStatus  MessageKind = "status"
	KindCancel  MessageKind = "cancel"
)

type MessageStatus string

const (
	StatusQueued    MessageStatus = "queued"
	StatusDelivered MessageStatus = "delivered"
	StatusReceived  MessageStatus = "received"
	StatusReplied   MessageStatus = "replied"
	StatusBlocked   MessageStatus = "blocked"
	StatusClosed    MessageStatus = "closed"
	StatusError     MessageStatus = "error"
)

type ThreadStatus string

const (
	ThreadStatusOpen   ThreadStatus = "open"
	ThreadStatusClosed ThreadStatus = "closed"
)

type ContextPolicy string

const (
	ContextPolicyTaskOnly        ContextPolicy = "task_only"
	ContextPolicySummary         ContextPolicy = "summary"
	ContextPolicySelectedContext ContextPolicy = "selected_context"
)

type Envelope struct {
	ID            string                 `json:"id"`
	ThreadID      string                 `json:"thread_id"`
	ParentID      string                 `json:"parent_id,omitempty"`
	FromAgentID   string                 `json:"from_agent_id"`
	ToAgentIDs    []string               `json:"to_agent_ids,omitempty"`
	Kind          MessageKind            `json:"kind"`
	Content       string                 `json:"content"`
	Artifacts     []providers.Attachment `json:"artifacts,omitempty"`
	ExpectReply   bool                   `json:"expect_reply,omitempty"`
	Deadline      *time.Time             `json:"deadline,omitempty"`
	Priority      string                 `json:"priority,omitempty"`
	Status        MessageStatus          `json:"status"`
	TraceID       string                 `json:"trace_id,omitempty"`
	ParentTurnID  string                 `json:"parent_turn_id,omitempty"`
	ContextPolicy ContextPolicy          `json:"context_policy,omitempty"`
	ContextShared string                 `json:"context_shared,omitempty"`
	ErrorSummary  string                 `json:"error_summary,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	DeliveredAt   *time.Time             `json:"delivered_at,omitempty"`
	ReceivedAt    *time.Time             `json:"received_at,omitempty"`
	RepliedAt     *time.Time             `json:"replied_at,omitempty"`
}

type Thread struct {
	ID                  string       `json:"id"`
	ParticipantAgentIDs []string     `json:"participant_agent_ids,omitempty"`
	MessageIDs          []string     `json:"message_ids,omitempty"`
	Status              ThreadStatus `json:"status"`
	CloseReason         string       `json:"close_reason,omitempty"`
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
	LastMessageID       string       `json:"last_message_id,omitempty"`
}

type Mailbox struct {
	AgentID    string    `json:"agent_id"`
	MessageIDs []string  `json:"message_ids,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func NormalizeContextPolicy(policy ContextPolicy) ContextPolicy {
	switch strings.ToLower(strings.TrimSpace(string(policy))) {
	case string(ContextPolicySummary):
		return ContextPolicySummary
	case string(ContextPolicySelectedContext):
		return ContextPolicySelectedContext
	default:
		return ContextPolicyTaskOnly
	}
}

func CloneEnvelope(env Envelope) Envelope {
	cloned := env
	if len(env.ToAgentIDs) > 0 {
		cloned.ToAgentIDs = append([]string(nil), env.ToAgentIDs...)
	}
	if len(env.Artifacts) > 0 {
		cloned.Artifacts = append([]providers.Attachment(nil), env.Artifacts...)
	}
	if env.Deadline != nil {
		deadline := *env.Deadline
		cloned.Deadline = &deadline
	}
	if env.DeliveredAt != nil {
		delivered := *env.DeliveredAt
		cloned.DeliveredAt = &delivered
	}
	if env.ReceivedAt != nil {
		received := *env.ReceivedAt
		cloned.ReceivedAt = &received
	}
	if env.RepliedAt != nil {
		replied := *env.RepliedAt
		cloned.RepliedAt = &replied
	}
	return cloned
}

func CloneThread(thread Thread) Thread {
	cloned := thread
	if len(thread.ParticipantAgentIDs) > 0 {
		cloned.ParticipantAgentIDs = append([]string(nil), thread.ParticipantAgentIDs...)
	}
	if len(thread.MessageIDs) > 0 {
		cloned.MessageIDs = append([]string(nil), thread.MessageIDs...)
	}
	return cloned
}

func MergeParticipants(existing []string, participants ...string) []string {
	if len(participants) == 0 {
		return append([]string(nil), existing...)
	}
	merged := append([]string(nil), existing...)
	for _, participant := range participants {
		participant = strings.TrimSpace(participant)
		if participant == "" || slices.Contains(merged, participant) {
			continue
		}
		merged = append(merged, participant)
	}
	return merged
}
