package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/collab"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type AgentRequestParams struct {
	ToAgentID       string
	ThreadID        string
	Content         string
	DeadlineSeconds int
	ContextPolicy   collab.ContextPolicy
	SelectedContext string
	Wait            bool
}

type AgentRequestResponse struct {
	ThreadID       string
	MessageID      string
	ReplyMessageID string
	FromAgentID    string
	Content        string
	Status         collab.MessageStatus
}

type AgentReplyParams struct {
	ThreadID  string
	InReplyTo string
	Content   string
	Artifacts []providers.Attachment
}

type AgentReplyResponse struct {
	ThreadID  string
	MessageID string
	Status    collab.MessageStatus
}

type AgentInboxParams struct {
	ThreadID string
	Status   string
}

type AgentInboxMessage struct {
	ID             string
	ThreadID       string
	ParentID       string
	FromAgentID    string
	Kind           collab.MessageKind
	Status         collab.MessageStatus
	ExpectReply    bool
	ContentPreview string
	ArtifactCount  int
	CreatedAt      time.Time
}

type AgentInboxResponse struct {
	AgentID      string
	ThreadID     string
	ThreadStatus string
	Participants []string
	Messages     []AgentInboxMessage
}

type AgentCollaborationRuntime interface {
	Request(ctx context.Context, params AgentRequestParams) (*AgentRequestResponse, error)
	Reply(ctx context.Context, params AgentReplyParams) (*AgentReplyResponse, error)
	Inbox(ctx context.Context, params AgentInboxParams) (*AgentInboxResponse, error)
}

type AgentRequestTool struct {
	runtime AgentCollaborationRuntime
}

func NewAgentRequestTool() *AgentRequestTool {
	return &AgentRequestTool{}
}

func (t *AgentRequestTool) SetRuntime(runtime AgentCollaborationRuntime) {
	t.runtime = runtime
}

func (t *AgentRequestTool) Name() string {
	return "agent_request"
}

func (t *AgentRequestTool) Description() string {
	return "Send a structured internal collaboration request to another permitted agent. Use this for peer-to-peer coordination that needs a durable thread and mailbox."
}

func (t *AgentRequestTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"to": map[string]any{
				"type":        "string",
				"description": "Target agent ID",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The collaboration request to send",
			},
			"thread_id": map[string]any{
				"type":        "string",
				"description": "Optional existing collaboration thread ID",
			},
			"deadline_seconds": map[string]any{
				"type":        "integer",
				"description": "Optional timeout for waiting and target processing",
			},
			"context_policy": map[string]any{
				"type":        "string",
				"description": "Context sharing policy: task_only, summary, or selected_context",
				"enum":        []string{"task_only", "summary", "selected_context"},
			},
			"selected_context": map[string]any{
				"type":        "string",
				"description": "Explicit context to share when context_policy is selected_context",
			},
			"wait": map[string]any{
				"type":        "boolean",
				"description": "Wait for a reply before returning. Defaults to true.",
			},
		},
		"required": []string{"to", "content"},
	}
}

func (t *AgentRequestTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t.runtime == nil {
		return ErrorResult("agent_request is not configured")
	}

	toAgentID, _ := args["to"].(string)
	if strings.TrimSpace(toAgentID) == "" {
		return ErrorResult("to is required and must be a non-empty string")
	}

	content, _ := args["content"].(string)
	if strings.TrimSpace(content) == "" {
		return ErrorResult("content is required and must be a non-empty string")
	}

	threadID, _ := args["thread_id"].(string)
	deadlineSeconds := intFromAny(args["deadline_seconds"])
	selectedContext, _ := args["selected_context"].(string)
	wait, waitSet := args["wait"].(bool)
	if !waitSet {
		wait = true
	}

	response, err := t.runtime.Request(ctx, AgentRequestParams{
		ToAgentID:       toAgentID,
		ThreadID:        threadID,
		Content:         content,
		DeadlineSeconds: deadlineSeconds,
		ContextPolicy:   collab.NormalizeContextPolicy(collab.ContextPolicy(stringValue(args["context_policy"]))),
		SelectedContext: selectedContext,
		Wait:            wait,
	})
	if err != nil {
		return ErrorResult(err.Error()).WithError(err)
	}

	if wait {
		return &ToolResult{
			ForLLM: fmt.Sprintf(
				"[Reply from agent %q in thread %s]\n%s",
				response.FromAgentID,
				response.ThreadID,
				response.Content,
			),
			ForUser: response.Content,
		}
	}

	return NewToolResult(fmt.Sprintf(
		"Internal request queued for agent %q in thread %s (message_id=%s).",
		toAgentID,
		response.ThreadID,
		response.MessageID,
	))
}

type AgentReplyTool struct {
	runtime AgentCollaborationRuntime
}

func NewAgentReplyTool() *AgentReplyTool {
	return &AgentReplyTool{}
}

func (t *AgentReplyTool) SetRuntime(runtime AgentCollaborationRuntime) {
	t.runtime = runtime
}

func (t *AgentReplyTool) Name() string {
	return "agent_reply"
}

func (t *AgentReplyTool) Description() string {
	return "Reply inside an existing internal collaboration thread. Use this to send a structured response back to another agent."
}

func (t *AgentReplyTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"thread_id": map[string]any{
				"type":        "string",
				"description": "The collaboration thread ID",
			},
			"in_reply_to": map[string]any{
				"type":        "string",
				"description": "The message ID being answered",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The reply content",
			},
			"artifacts": map[string]any{
				"type":        "array",
				"description": "Optional artifact references to include with the reply",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"type":         map[string]any{"type": "string"},
						"ref":          map[string]any{"type": "string"},
						"url":          map[string]any{"type": "string"},
						"filename":     map[string]any{"type": "string"},
						"content_type": map[string]any{"type": "string"},
					},
				},
			},
		},
		"required": []string{"thread_id", "in_reply_to", "content"},
	}
}

func (t *AgentReplyTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t.runtime == nil {
		return ErrorResult("agent_reply is not configured")
	}

	threadID, _ := args["thread_id"].(string)
	if strings.TrimSpace(threadID) == "" {
		return ErrorResult("thread_id is required and must be a non-empty string")
	}

	inReplyTo, _ := args["in_reply_to"].(string)
	if strings.TrimSpace(inReplyTo) == "" {
		return ErrorResult("in_reply_to is required and must be a non-empty string")
	}

	content, _ := args["content"].(string)
	if strings.TrimSpace(content) == "" {
		return ErrorResult("content is required and must be a non-empty string")
	}

	reply, err := t.runtime.Reply(ctx, AgentReplyParams{
		ThreadID:  threadID,
		InReplyTo: inReplyTo,
		Content:   content,
		Artifacts: attachmentsFromAny(args["artifacts"]),
	})
	if err != nil {
		return ErrorResult(err.Error()).WithError(err)
	}

	return NewToolResult(fmt.Sprintf(
		"Reply recorded in collaboration thread %s (message_id=%s, status=%s).",
		reply.ThreadID,
		reply.MessageID,
		reply.Status,
	))
}

type AgentInboxTool struct {
	runtime AgentCollaborationRuntime
}

func NewAgentInboxTool() *AgentInboxTool {
	return &AgentInboxTool{}
}

func (t *AgentInboxTool) SetRuntime(runtime AgentCollaborationRuntime) {
	t.runtime = runtime
}

func (t *AgentInboxTool) Name() string {
	return "agent_inbox"
}

func (t *AgentInboxTool) Description() string {
	return "Inspect the current agent mailbox and collaboration-thread delivery state."
}

func (t *AgentInboxTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"thread_id": map[string]any{
				"type":        "string",
				"description": "Optional thread ID filter",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "Optional message status filter",
			},
		},
	}
}

func (t *AgentInboxTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t.runtime == nil {
		return ErrorResult("agent_inbox is not configured")
	}

	threadID, _ := args["thread_id"].(string)
	status, _ := args["status"].(string)
	inbox, err := t.runtime.Inbox(ctx, AgentInboxParams{
		ThreadID: threadID,
		Status:   status,
	})
	if err != nil {
		return ErrorResult(err.Error()).WithError(err)
	}

	return NewToolResult(formatAgentInbox(inbox))
}

func formatAgentInbox(inbox *AgentInboxResponse) string {
	if inbox == nil {
		return "No mailbox data available."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Mailbox for agent %q", inbox.AgentID)
	if strings.TrimSpace(inbox.ThreadID) != "" {
		fmt.Fprintf(&b, " in thread %s", inbox.ThreadID)
	}
	if strings.TrimSpace(inbox.ThreadStatus) != "" {
		fmt.Fprintf(&b, " (thread_status=%s)", inbox.ThreadStatus)
	}
	if len(inbox.Participants) > 0 {
		fmt.Fprintf(&b, "\nParticipants: %s", strings.Join(inbox.Participants, ", "))
	}
	if len(inbox.Messages) == 0 {
		b.WriteString("\nNo matching messages.")
		return b.String()
	}
	for _, msg := range inbox.Messages {
		fmt.Fprintf(&b,
			"\n- %s | from=%s | kind=%s | status=%s | expect_reply=%t | artifacts=%d | at=%s\n  %s",
			msg.ID,
			msg.FromAgentID,
			msg.Kind,
			msg.Status,
			msg.ExpectReply,
			msg.ArtifactCount,
			msg.CreatedAt.UTC().Format(time.RFC3339),
			msg.ContentPreview,
		)
	}
	return b.String()
}

func attachmentsFromAny(raw any) []providers.Attachment {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return nil
	}

	attachments := make([]providers.Attachment, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		attachments = append(attachments, providers.Attachment{
			Type:        stringValue(m["type"]),
			Ref:         stringValue(m["ref"]),
			URL:         stringValue(m["url"]),
			Filename:    stringValue(m["filename"]),
			ContentType: stringValue(m["content_type"]),
		})
	}
	return attachments
}

func intFromAny(raw any) int {
	switch value := raw.(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func stringValue(raw any) string {
	value, _ := raw.(string)
	return strings.TrimSpace(value)
}
