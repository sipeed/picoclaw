package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/collab"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type collaborationReplyProvider struct{}

type collaborationDeadlineProvider struct {
	mu        sync.Mutex
	sawCtxTTL bool
	remaining time.Duration
}

var (
	threadIDPattern  = regexp.MustCompile(`Thread ID:\s*([^\n]+)`)
	messageIDPattern = regexp.MustCompile(`Message ID:\s*([^\n]+)`)
)

func (p *collaborationReplyProvider) Chat(
	_ context.Context,
	messages []providers.Message,
	toolDefs []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	lastUser := ""
	hasToolReply := false
	for _, msg := range messages {
		if msg.Role == "user" {
			lastUser = msg.Content
		}
		if msg.Role == "tool" && strings.Contains(msg.Content, "Reply recorded in collaboration thread") {
			hasToolReply = true
		}
	}

	if hasToolReply {
		return &providers.LLMResponse{Content: "Reply sent."}, nil
	}

	if strings.Contains(lastUser, "[Internal collaboration message]") && hasTool(toolDefs, "agent_reply") {
		threadID := matchFirst(threadIDPattern, lastUser)
		messageID := matchFirst(messageIDPattern, lastUser)
		return &providers.LLMResponse{
			ToolCalls: []providers.ToolCall{
				{
					ID:   "call-agent-reply",
					Name: "agent_reply",
					Arguments: map[string]any{
						"thread_id":   threadID,
						"in_reply_to": messageID,
						"content":     "Tradeoffs: faster indexing, clearer ownership, and durable follow-up.",
					},
				},
			},
		}, nil
	}

	return &providers.LLMResponse{Content: "generic"}, nil
}

func (p *collaborationReplyProvider) GetDefaultModel() string {
	return "collaboration-reply-model"
}

func (p *collaborationDeadlineProvider) Chat(
	ctx context.Context,
	_ []providers.Message,
	_ []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if deadline, ok := ctx.Deadline(); ok {
		p.sawCtxTTL = true
		p.remaining = time.Until(deadline)
	}
	return &providers.LLMResponse{Content: "Timeout observed."}, nil
}

func (p *collaborationDeadlineProvider) GetDefaultModel() string {
	return "collaboration-deadline-model"
}

func (p *collaborationDeadlineProvider) snapshot() (bool, time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sawCtxTTL, p.remaining
}

func TestCollaborationBus_RequestWaitTrue_ExplicitReplyAndHistory(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := collaborationTestConfig(tmpDir)
	al := NewAgentLoop(cfg, nil, &collaborationReplyProvider{})
	defer al.Close()

	planner, ok := al.GetRegistry().GetAgent("planner")
	if !ok || planner == nil {
		t.Fatal("expected planner agent")
	}
	research, ok := al.GetRegistry().GetAgent("research")
	if !ok || research == nil {
		t.Fatal("expected research agent")
	}
	if al.collaboration == nil {
		t.Fatal("expected collaboration bus to be initialized")
	}

	plannerScope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "planner",
		Channel:    "cli",
		Account:    "default",
		Dimensions: []string{"chat"},
		Values:     map[string]string{"chat": "direct:tester"},
	}
	plannerSessionKey := session.BuildSessionKey(*plannerScope)
	ctx := tools.WithToolSessionContext(context.Background(), planner.ID, plannerSessionKey, plannerScope)

	reply, err := al.collaboration.Request(ctx, tools.AgentRequestParams{
		ToAgentID:     "research",
		Content:       "Please investigate the collaboration bus tradeoffs.",
		ContextPolicy: collab.ContextPolicyTaskOnly,
		Wait:          true,
	})
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}
	if reply.Content != "Tradeoffs: faster indexing, clearer ownership, and durable follow-up." {
		t.Fatalf("reply.Content = %q", reply.Content)
	}
	if reply.FromAgentID != "research" {
		t.Fatalf("reply.FromAgentID = %q, want research", reply.FromAgentID)
	}

	inbox, err := al.collaboration.Inbox(ctx, tools.AgentInboxParams{ThreadID: reply.ThreadID})
	if err != nil {
		t.Fatalf("Inbox() error = %v", err)
	}
	if len(inbox.Messages) != 1 {
		t.Fatalf("Inbox messages = %d, want 1", len(inbox.Messages))
	}
	if inbox.Messages[0].FromAgentID != "research" || inbox.Messages[0].Kind != collab.KindReply {
		t.Fatalf("Inbox message = %+v", inbox.Messages[0])
	}

	researchSessionKey := collaborationSessionKey("research", reply.ThreadID)
	history := research.Sessions.GetHistory(researchSessionKey)
	if len(history) < 2 {
		t.Fatalf("research history len = %d, want at least 2", len(history))
	}
	if !strings.Contains(history[0].Content, "Please investigate the collaboration bus tradeoffs.") {
		t.Fatalf("first history message = %q", history[0].Content)
	}
	foundReplyHistory := false
	for _, msg := range history {
		if strings.Contains(msg.Content, "Tradeoffs: faster indexing, clearer ownership, and durable follow-up.") {
			foundReplyHistory = true
			break
		}
	}
	if !foundReplyHistory {
		t.Fatalf("expected reply content to be present in research session history: %#v", history)
	}
	waitForCollaborationIdle(t, al, reply.ThreadID, "research")
}

func TestCollaborationBus_RequestWaitFalse_ReplyDurableInInbox(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := collaborationTestConfig(tmpDir)
	al := NewAgentLoop(cfg, nil, &collaborationReplyProvider{})
	defer al.Close()

	plannerScope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "planner",
		Channel:    "cli",
		Account:    "default",
		Dimensions: []string{"chat"},
		Values:     map[string]string{"chat": "direct:tester"},
	}
	plannerSessionKey := session.BuildSessionKey(*plannerScope)
	ctx := tools.WithToolSessionContext(context.Background(), "planner", plannerSessionKey, plannerScope)

	reply, err := al.collaboration.Request(ctx, tools.AgentRequestParams{
		ToAgentID:     "research",
		Content:       "Need a durable async reply.",
		ContextPolicy: collab.ContextPolicyTaskOnly,
		Wait:          false,
	})
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}
	if reply.ThreadID == "" || reply.MessageID == "" {
		t.Fatalf("async response = %+v", reply)
	}

	var inbox *tools.AgentInboxResponse
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		inbox, err = al.collaboration.Inbox(ctx, tools.AgentInboxParams{ThreadID: reply.ThreadID})
		if err != nil {
			t.Fatalf("Inbox() error = %v", err)
		}
		if len(inbox.Messages) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if inbox == nil || len(inbox.Messages) == 0 {
		t.Fatal("expected durable reply to appear in sender inbox")
	}
	if inbox.Messages[0].Status != collab.StatusDelivered {
		t.Fatalf("inbox.Messages[0].Status = %q, want delivered", inbox.Messages[0].Status)
	}
	if !strings.Contains(inbox.Messages[0].ContentPreview, "Tradeoffs: faster indexing") {
		t.Fatalf("inbox.Messages[0].ContentPreview = %q", inbox.Messages[0].ContentPreview)
	}
	waitForCollaborationIdle(t, al, reply.ThreadID, "research")
}

func TestCollaborationBus_RequestWithoutDeadline_UsesSubTurnDefaultTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := collaborationTestConfig(tmpDir)
	cfg.Agents.Defaults.SubTurn.DefaultTimeoutMinutes = 1
	provider := &collaborationDeadlineProvider{}
	al := NewAgentLoop(cfg, nil, provider)
	defer al.Close()

	plannerScope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "planner",
		Channel:    "cli",
		Account:    "default",
		Dimensions: []string{"chat"},
		Values:     map[string]string{"chat": "direct:tester"},
	}
	plannerSessionKey := session.BuildSessionKey(*plannerScope)
	ctx := tools.WithToolSessionContext(context.Background(), "planner", plannerSessionKey, plannerScope)

	reply, err := al.collaboration.Request(ctx, tools.AgentRequestParams{
		ToAgentID:     "research",
		Content:       "Verify default timeout handling.",
		ContextPolicy: collab.ContextPolicyTaskOnly,
		Wait:          true,
	})
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}
	if reply.Content != "Timeout observed." {
		t.Fatalf("reply.Content = %q", reply.Content)
	}

	sawDeadline, remaining := provider.snapshot()
	if !sawDeadline {
		t.Fatal("expected collaboration worker context to have a deadline")
	}
	if remaining <= 30*time.Second || remaining > time.Minute {
		t.Fatalf("remaining timeout = %v, want between 30s and 1m", remaining)
	}
	waitForCollaborationIdle(t, al, reply.ThreadID, "research")
}

func TestCollaborationBus_RequestRejectsOversizedSelectedContext(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := collaborationTestConfig(tmpDir)
	cfg.Agents.List[0].Communication.MaxMessageChars = 20
	cfg.Agents.List[1].Communication.MaxMessageChars = 20
	al := NewAgentLoop(cfg, nil, &collaborationReplyProvider{})
	defer al.Close()

	plannerScope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "planner",
		Channel:    "cli",
		Account:    "default",
		Dimensions: []string{"chat"},
		Values:     map[string]string{"chat": "direct:tester"},
	}
	plannerSessionKey := session.BuildSessionKey(*plannerScope)
	ctx := tools.WithToolSessionContext(context.Background(), "planner", plannerSessionKey, plannerScope)

	_, err := al.collaboration.Request(ctx, tools.AgentRequestParams{
		ToAgentID:       "research",
		Content:         "ok",
		ContextPolicy:   collab.ContextPolicySelectedContext,
		SelectedContext: "this shared context is way too large",
		Wait:            false,
	})
	if err == nil {
		t.Fatal("expected oversized selected_context to be rejected")
	}
	if !strings.Contains(err.Error(), "max_message_chars=20") {
		t.Fatalf("err = %v", err)
	}
}

func TestCollaborationBus_RequestRejectsOversizedSummaryContext(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := collaborationTestConfig(tmpDir)
	cfg.Agents.List[0].Communication.MaxMessageChars = 20
	cfg.Agents.List[1].Communication.MaxMessageChars = 20
	al := NewAgentLoop(cfg, nil, &collaborationReplyProvider{})
	defer al.Close()

	planner, ok := al.GetRegistry().GetAgent("planner")
	if !ok || planner == nil {
		t.Fatal("expected planner agent")
	}

	plannerScope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "planner",
		Channel:    "cli",
		Account:    "default",
		Dimensions: []string{"chat"},
		Values:     map[string]string{"chat": "direct:tester"},
	}
	plannerSessionKey := session.BuildSessionKey(*plannerScope)
	planner.Sessions.SetSummary(plannerSessionKey, "this summary is definitely too long")

	ctx := tools.WithToolSessionContext(context.Background(), "planner", plannerSessionKey, plannerScope)
	_, err := al.collaboration.Request(ctx, tools.AgentRequestParams{
		ToAgentID:     "research",
		Content:       "ok",
		ContextPolicy: collab.ContextPolicySummary,
		Wait:          false,
	})
	if err == nil {
		t.Fatal("expected oversized summary context to be rejected")
	}
	if !strings.Contains(err.Error(), "max_message_chars=20") {
		t.Fatalf("err = %v", err)
	}
}

func TestCollaborationBus_RequestBlockedByCommunicationPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := collaborationTestConfig(tmpDir)
	cfg.Agents.List[1].Communication.AllowReceiveFrom = nil
	al := NewAgentLoop(cfg, nil, &collaborationReplyProvider{})
	defer al.Close()

	plannerScope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "planner",
		Channel:    "cli",
		Account:    "default",
		Dimensions: []string{"chat"},
		Values:     map[string]string{"chat": "direct:tester"},
	}
	plannerSessionKey := session.BuildSessionKey(*plannerScope)
	ctx := tools.WithToolSessionContext(context.Background(), "planner", plannerSessionKey, plannerScope)

	_, err := al.collaboration.Request(ctx, tools.AgentRequestParams{
		ToAgentID:     "research",
		Content:       "Should be blocked.",
		ContextPolicy: collab.ContextPolicyTaskOnly,
		Wait:          true,
	})
	if err == nil {
		t.Fatal("expected collaboration request to be blocked")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("err = %v", err)
	}
}

func TestCollaborationBus_RequestRejectsThreadParticipantHijack(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := collaborationTestConfig(tmpDir)
	cfg.Agents.List = append(cfg.Agents.List, config.AgentConfig{
		ID: "reviewer",
		Communication: &config.AgentCommunicationConfig{
			AllowSendTo:      []string{"research"},
			AllowReceiveFrom: []string{"research"},
			MaxThreadTurns:   8,
			MaxMessageChars:  12000,
		},
	})
	cfg.Agents.List[1].Communication.AllowSendTo = []string{"planner", "reviewer"}
	cfg.Agents.List[1].Communication.AllowReceiveFrom = []string{"planner", "reviewer"}

	al := NewAgentLoop(cfg, nil, &collaborationReplyProvider{})
	defer al.Close()

	plannerScope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "planner",
		Channel:    "cli",
		Account:    "default",
		Dimensions: []string{"chat"},
		Values:     map[string]string{"chat": "direct:tester"},
	}
	plannerSessionKey := session.BuildSessionKey(*plannerScope)
	plannerCtx := tools.WithToolSessionContext(context.Background(), "planner", plannerSessionKey, plannerScope)

	reply, err := al.collaboration.Request(plannerCtx, tools.AgentRequestParams{
		ToAgentID:     "research",
		Content:       "Open the original collaboration thread.",
		ContextPolicy: collab.ContextPolicyTaskOnly,
		Wait:          true,
	})
	if err != nil {
		t.Fatalf("planner Request() error = %v", err)
	}
	waitForCollaborationIdle(t, al, reply.ThreadID, "research")

	reviewerScope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "reviewer",
		Channel:    "cli",
		Account:    "default",
		Dimensions: []string{"chat"},
		Values:     map[string]string{"chat": "direct:tester"},
	}
	reviewerSessionKey := session.BuildSessionKey(*reviewerScope)
	reviewerCtx := tools.WithToolSessionContext(context.Background(), "reviewer", reviewerSessionKey, reviewerScope)

	_, err = al.collaboration.Request(reviewerCtx, tools.AgentRequestParams{
		ToAgentID:     "research",
		ThreadID:      reply.ThreadID,
		Content:       "Attempt to hijack the existing thread.",
		ContextPolicy: collab.ContextPolicyTaskOnly,
		Wait:          false,
	})
	if err == nil {
		t.Fatal("expected thread participant hijack to be rejected")
	}
	if !strings.Contains(err.Error(), "does not include both agent") {
		t.Fatalf("err = %v", err)
	}
}

func TestCollaborationBus_FollowUpRequestLinksParentMessage(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := collaborationTestConfig(tmpDir)
	al := NewAgentLoop(cfg, nil, &collaborationReplyProvider{})
	defer al.Close()

	plannerScope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "planner",
		Channel:    "cli",
		Account:    "default",
		Dimensions: []string{"chat"},
		Values:     map[string]string{"chat": "direct:tester"},
	}
	plannerSessionKey := session.BuildSessionKey(*plannerScope)
	plannerCtx := tools.WithToolSessionContext(context.Background(), "planner", plannerSessionKey, plannerScope)

	firstReply, err := al.collaboration.Request(plannerCtx, tools.AgentRequestParams{
		ToAgentID:     "research",
		Content:       "Open a thread for a follow-up request.",
		ContextPolicy: collab.ContextPolicyTaskOnly,
		Wait:          true,
	})
	if err != nil {
		t.Fatalf("first Request() error = %v", err)
	}
	waitForCollaborationIdle(t, al, firstReply.ThreadID, "research")

	followUp, err := al.collaboration.Request(plannerCtx, tools.AgentRequestParams{
		ToAgentID:     "research",
		ThreadID:      firstReply.ThreadID,
		Content:       "Follow up with another question.",
		ContextPolicy: collab.ContextPolicyTaskOnly,
		Wait:          false,
	})
	if err != nil {
		t.Fatalf("follow-up Request() error = %v", err)
	}

	message, ok := al.collaboration.store.GetMessage(followUp.MessageID)
	if !ok {
		t.Fatalf("expected stored follow-up message %s", followUp.MessageID)
	}
	if message.ParentID != firstReply.ReplyMessageID {
		t.Fatalf("message.ParentID = %q, want %q", message.ParentID, firstReply.ReplyMessageID)
	}
	waitForCollaborationIdle(t, al, firstReply.ThreadID, "research")
}

func TestCollaborationBus_ClosedThreadRejectsNewMessages(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := collaborationTestConfig(tmpDir)
	al := NewAgentLoop(cfg, nil, &collaborationReplyProvider{})
	defer al.Close()

	plannerScope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "planner",
		Channel:    "cli",
		Account:    "default",
		Dimensions: []string{"chat"},
		Values:     map[string]string{"chat": "direct:tester"},
	}
	plannerSessionKey := session.BuildSessionKey(*plannerScope)
	plannerCtx := tools.WithToolSessionContext(context.Background(), "planner", plannerSessionKey, plannerScope)

	reply, err := al.collaboration.Request(plannerCtx, tools.AgentRequestParams{
		ToAgentID:     "research",
		Content:       "Create a thread that will be closed.",
		ContextPolicy: collab.ContextPolicyTaskOnly,
		Wait:          true,
	})
	if err != nil {
		t.Fatalf("planner Request() error = %v", err)
	}
	waitForCollaborationIdle(t, al, reply.ThreadID, "research")

	if _, closeErr := al.collaboration.store.CloseThread(reply.ThreadID, "test close"); closeErr != nil {
		t.Fatalf("CloseThread() error = %v", closeErr)
	}

	_, err = al.collaboration.Request(plannerCtx, tools.AgentRequestParams{
		ToAgentID:     "research",
		ThreadID:      reply.ThreadID,
		Content:       "This follow-up should be rejected.",
		ContextPolicy: collab.ContextPolicyTaskOnly,
		Wait:          false,
	})
	if err == nil {
		t.Fatal("expected request on closed thread to be rejected")
	}
	if !strings.Contains(err.Error(), "is closed") {
		t.Fatalf("request err = %v", err)
	}

	researchScope := &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    "research",
		Channel:    "cli",
		Account:    "default",
		Dimensions: []string{"chat"},
		Values:     map[string]string{"chat": "direct:tester"},
	}
	researchSessionKey := session.BuildSessionKey(*researchScope)
	researchCtx := tools.WithToolSessionContext(context.Background(), "research", researchSessionKey, researchScope)

	_, err = al.collaboration.Reply(researchCtx, tools.AgentReplyParams{
		ThreadID:  reply.ThreadID,
		InReplyTo: reply.MessageID,
		Content:   "This reply should also be rejected.",
	})
	if err == nil {
		t.Fatal("expected reply on closed thread to be rejected")
	}
	if !strings.Contains(err.Error(), "is closed") {
		t.Fatalf("reply err = %v", err)
	}
}

func collaborationTestConfig(workspace string) *config.Config {
	return &config.Config{
		Agents: config.AgentsConfig{
			Defaults: config.AgentDefaults{
				Workspace:         workspace,
				ModelName:         "test-model",
				MaxTokens:         4096,
				MaxToolIterations: 4,
			},
			List: []config.AgentConfig{
				{
					ID: "planner",
					Communication: &config.AgentCommunicationConfig{
						AllowSendTo:      []string{"research"},
						AllowReceiveFrom: []string{"research"},
						MaxThreadTurns:   8,
						MaxMessageChars:  12000,
					},
				},
				{
					ID: "research",
					Communication: &config.AgentCommunicationConfig{
						AllowSendTo:      []string{"planner"},
						AllowReceiveFrom: []string{"planner"},
						MaxThreadTurns:   8,
						MaxMessageChars:  12000,
					},
				},
			},
		},
	}
}

func hasTool(toolDefs []providers.ToolDefinition, name string) bool {
	for _, toolDef := range toolDefs {
		if toolDef.Function.Name == name {
			return true
		}
	}
	return false
}

func matchFirst(pattern *regexp.Regexp, value string) string {
	match := pattern.FindStringSubmatch(value)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func TestCollaborationSessionAliases_AreStable(t *testing.T) {
	aliases := collaborationSessionAliases("Research", "thread_123")
	if len(aliases) != 1 {
		t.Fatalf("aliases = %v", aliases)
	}
	want := "agent:research:inter-agent:thread:thread_123"
	if aliases[0] != want {
		t.Fatalf("aliases[0] = %q, want %q", aliases[0], want)
	}
	if got := collaborationSessionKey("research", "thread_123"); got == "" {
		t.Fatal("collaborationSessionKey should not be empty")
	}
	if got := fmt.Sprintf(
		"%v",
		collaborationSessionScope("research", "thread_123").Values["thread"],
	); got != "thread_123" {
		t.Fatalf("thread scope value = %q", got)
	}
}

func waitForCollaborationIdle(t *testing.T, al *AgentLoop, threadID, targetAgentID string) {
	t.Helper()

	sessionKey := collaborationSessionKey(targetAgentID, threadID)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if al.GetActiveTurnBySession(sessionKey) == nil && !al.collaboration.isQueueActive(sessionKey) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("collaboration session %s did not become idle", sessionKey)
}
