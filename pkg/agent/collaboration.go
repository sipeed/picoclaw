package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/collab"
	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/tools"
)

const (
	collaborationChannel        = "inter-agent"
	collaborationAccount        = "collaboration"
	defaultCollaborationTimeout = 5 * time.Minute
	collaborationContentPreview = 240
)

type AgentCollaborationBus struct {
	al    *AgentLoop
	store *collab.Store

	waitersMu sync.Mutex
	waiters   map[string][]chan collab.Envelope

	queueMu sync.Mutex
	queues  map[string]*collaborationSessionQueue
}

type collaborationSessionQueue struct {
	processing    bool
	targetAgentID string
	threadID      string
	messageIDs    []string
}

func NewAgentCollaborationBus(al *AgentLoop, dir string) (*AgentCollaborationBus, error) {
	store, err := collab.NewStore(dir)
	if err != nil {
		return nil, err
	}
	return &AgentCollaborationBus{
		al:      al,
		store:   store,
		waiters: make(map[string][]chan collab.Envelope),
		queues:  make(map[string]*collaborationSessionQueue),
	}, nil
}

func (b *AgentCollaborationBus) Request(
	ctx context.Context,
	params tools.AgentRequestParams,
) (*tools.AgentRequestResponse, error) {
	if b == nil || b.store == nil || b.al == nil {
		return nil, fmt.Errorf("collaboration bus is not available")
	}

	fromAgentID := currentToolAgentID(ctx)
	if fromAgentID == "" {
		return nil, fmt.Errorf("agent_request requires an active agent turn")
	}
	toAgentID := routing.NormalizeAgentID(params.ToAgentID)
	if toAgentID == "" {
		return nil, fmt.Errorf("target agent is required")
	}
	if fromAgentID == toAgentID {
		return nil, fmt.Errorf("cannot send collaboration request to self")
	}

	threadID := strings.TrimSpace(params.ThreadID)
	var thread collab.Thread
	var threadExists bool
	if threadID == "" {
		threadID = "thread_" + uuid.New().String()
	} else {
		var ok bool
		thread, ok = b.store.GetThread(threadID)
		if !ok {
			return nil, fmt.Errorf("collaboration thread %q was not found", threadID)
		}
		threadExists = true
	}

	messageID := "msg_" + uuid.New().String()
	traceID := "trace_" + uuid.New().String()
	parentTurnID := currentTurnID(ctx)
	policy := collab.NormalizeContextPolicy(params.ContextPolicy)
	sharedContext, err := b.buildSharedContext(ctx, fromAgentID, policy, params.SelectedContext)
	if err != nil {
		return nil, err
	}

	if !b.al.registry.CanCollaborate(fromAgentID, toAgentID) {
		env := collab.Envelope{
			ID:            messageID,
			ThreadID:      threadID,
			FromAgentID:   fromAgentID,
			ToAgentIDs:    []string{toAgentID},
			Kind:          collab.KindRequest,
			Content:       strings.TrimSpace(params.Content),
			ExpectReply:   true,
			Priority:      "normal",
			Status:        collab.StatusBlocked,
			TraceID:       traceID,
			ParentTurnID:  parentTurnID,
			ContextPolicy: policy,
			ContextShared: sharedContext,
			ErrorSummary:  "communication policy blocked this request",
			CreatedAt:     time.Now().UTC(),
		}
		if _, ensureErr := b.store.EnsureThread(collab.Thread{
			ID:                  threadID,
			ParticipantAgentIDs: []string{fromAgentID, toAgentID},
			Status:              collab.ThreadStatusOpen,
		}); ensureErr == nil {
			_, _ = b.store.AddMessage(env)
		}
		b.emitCollaborationEvent(runtimeevents.KindAgentMessageBlocked, env, toAgentID)
		return nil, fmt.Errorf("agent %q is not allowed to message agent %q", fromAgentID, toAgentID)
	}
	if threadExists {
		if thread.Status == collab.ThreadStatusClosed {
			return nil, fmt.Errorf("collaboration thread %q is closed", threadID)
		}
		if !threadIncludesParticipant(thread, fromAgentID) || !threadIncludesParticipant(thread, toAgentID) {
			return nil, fmt.Errorf(
				"collaboration thread %q does not include both agent %q and agent %q",
				threadID,
				fromAgentID,
				toAgentID,
			)
		}
	}

	if policyErr := b.enforceThreadPolicy(
		fromAgentID,
		[]string{toAgentID},
		threadID,
		collaborationPayloadCharCount(strings.TrimSpace(params.Content), sharedContext),
		0,
	); policyErr != nil {
		return nil, policyErr
	}

	now := time.Now().UTC()
	parentID := ""
	if threadExists {
		parentID = strings.TrimSpace(thread.LastMessageID)
	}
	env := collab.Envelope{
		ID:            messageID,
		ThreadID:      threadID,
		ParentID:      parentID,
		FromAgentID:   fromAgentID,
		ToAgentIDs:    []string{toAgentID},
		Kind:          collab.KindRequest,
		Content:       strings.TrimSpace(params.Content),
		ExpectReply:   true,
		Priority:      "normal",
		Status:        collab.StatusQueued,
		TraceID:       traceID,
		ParentTurnID:  parentTurnID,
		ContextPolicy: policy,
		ContextShared: sharedContext,
		CreatedAt:     now,
	}
	if params.DeadlineSeconds > 0 {
		deadline := now.Add(time.Duration(params.DeadlineSeconds) * time.Second)
		env.Deadline = &deadline
	}

	if _, ensureErr := b.store.EnsureThread(collab.Thread{
		ID:                  threadID,
		ParticipantAgentIDs: []string{fromAgentID, toAgentID},
		Status:              collab.ThreadStatusOpen,
	}); ensureErr != nil {
		return nil, ensureErr
	}
	if _, addErr := b.store.AddMessage(env); addErr != nil {
		return nil, addErr
	}
	b.emitCollaborationEvent(runtimeevents.KindAgentMessageSent, env, toAgentID)

	var waiter chan collab.Envelope
	if params.Wait {
		waiter = make(chan collab.Envelope, 1)
		b.addWaiter(messageID, waiter)
		defer b.removeWaiter(messageID, waiter)
	}

	sessionKey := collaborationSessionKey(toAgentID, threadID)
	b.enqueueRequest(sessionKey, toAgentID, threadID, messageID)
	if _, statusErr := b.store.UpdateMessageStatus(
		messageID,
		collab.StatusDelivered,
		"",
	); statusErr == nil {
		if latest, ok := b.store.GetMessage(messageID); ok {
			b.emitCollaborationEvent(runtimeevents.KindAgentMessageDelivered, latest, toAgentID)
		}
	}

	if !params.Wait {
		return &tools.AgentRequestResponse{
			ThreadID:  threadID,
			MessageID: messageID,
			Status:    collab.StatusDelivered,
		}, nil
	}

	response, err := b.waitForResponse(ctx, threadID, messageID, waiter, env.Deadline)
	if err != nil {
		return nil, err
	}
	return &tools.AgentRequestResponse{
		ThreadID:       response.ThreadID,
		MessageID:      messageID,
		ReplyMessageID: response.ID,
		FromAgentID:    response.FromAgentID,
		Content:        response.Content,
		Status:         response.Status,
	}, nil
}

func (b *AgentCollaborationBus) Reply(
	ctx context.Context,
	params tools.AgentReplyParams,
) (*tools.AgentReplyResponse, error) {
	if b == nil || b.store == nil || b.al == nil {
		return nil, fmt.Errorf("collaboration bus is not available")
	}

	fromAgentID := currentToolAgentID(ctx)
	if fromAgentID == "" {
		return nil, fmt.Errorf("agent_reply requires an active agent turn")
	}
	threadID := strings.TrimSpace(params.ThreadID)
	if threadID == "" {
		return nil, fmt.Errorf("thread_id is required")
	}
	parentID := strings.TrimSpace(params.InReplyTo)
	if parentID == "" {
		return nil, fmt.Errorf("in_reply_to is required")
	}
	if strings.TrimSpace(params.Content) == "" {
		return nil, fmt.Errorf("content is required")
	}

	thread, ok := b.store.GetThread(threadID)
	if !ok {
		return nil, fmt.Errorf("collaboration thread %q was not found", threadID)
	}
	if thread.Status == collab.ThreadStatusClosed {
		return nil, fmt.Errorf("collaboration thread %q is closed", threadID)
	}
	if !threadIncludesParticipant(thread, fromAgentID) {
		return nil, fmt.Errorf("agent %q is not a participant in thread %q", fromAgentID, threadID)
	}
	parentMsg, ok := b.store.GetMessage(parentID)
	if !ok || parentMsg.ThreadID != threadID {
		return nil, fmt.Errorf("message %q was not found in thread %q", parentID, threadID)
	}

	recipients := replyRecipients(parentMsg, fromAgentID, thread.ParticipantAgentIDs)
	if len(recipients) == 0 {
		return nil, fmt.Errorf("no collaboration recipient could be resolved for thread %q", threadID)
	}
	for _, toAgentID := range recipients {
		if !b.al.registry.CanCollaborate(fromAgentID, toAgentID) {
			return nil, fmt.Errorf("agent %q is not allowed to reply to agent %q", fromAgentID, toAgentID)
		}
	}

	if policyErr := b.enforceThreadPolicy(
		fromAgentID,
		recipients,
		threadID,
		collaborationPayloadCharCount(strings.TrimSpace(params.Content), ""),
		len(params.Artifacts),
	); policyErr != nil {
		return nil, policyErr
	}

	replyEnv := collab.Envelope{
		ID:           "msg_" + uuid.New().String(),
		ThreadID:     threadID,
		ParentID:     parentID,
		FromAgentID:  fromAgentID,
		ToAgentIDs:   recipients,
		Kind:         collab.KindReply,
		Content:      strings.TrimSpace(params.Content),
		Artifacts:    append([]providers.Attachment(nil), params.Artifacts...),
		Priority:     "normal",
		Status:       collab.StatusDelivered,
		TraceID:      parentMsg.TraceID,
		ParentTurnID: currentTurnID(ctx),
		CreatedAt:    time.Now().UTC(),
	}
	if _, err := b.store.AddMessage(replyEnv); err != nil {
		return nil, err
	}
	if _, err := b.store.UpdateMessageStatus(parentID, collab.StatusReplied, ""); err != nil {
		logger.WarnCF("agent", "Failed to mark collaboration request as replied", map[string]any{
			"thread_id":  threadID,
			"message_id": parentID,
			"error":      err.Error(),
		})
	}
	b.appendReplyHistory(ctx, replyEnv)
	b.emitCollaborationEvent(runtimeevents.KindAgentMessageReply, replyEnv, strings.Join(recipients, ","))
	b.notifyWaiters(parentID, replyEnv)

	return &tools.AgentReplyResponse{
		ThreadID:  threadID,
		MessageID: replyEnv.ID,
		Status:    replyEnv.Status,
	}, nil
}

func (b *AgentCollaborationBus) Inbox(
	ctx context.Context,
	params tools.AgentInboxParams,
) (*tools.AgentInboxResponse, error) {
	if b == nil || b.store == nil || b.al == nil {
		return nil, fmt.Errorf("collaboration bus is not available")
	}

	agentID := currentToolAgentID(ctx)
	if agentID == "" {
		return nil, fmt.Errorf("agent_inbox requires an active agent turn")
	}

	inbox := &tools.AgentInboxResponse{
		AgentID:  agentID,
		ThreadID: strings.TrimSpace(params.ThreadID),
	}

	if inbox.ThreadID != "" {
		thread, ok := b.store.GetThread(inbox.ThreadID)
		if !ok {
			return nil, fmt.Errorf("collaboration thread %q was not found", inbox.ThreadID)
		}
		if !containsString(thread.ParticipantAgentIDs, agentID) {
			return nil, fmt.Errorf("agent %q is not a participant in thread %q", agentID, inbox.ThreadID)
		}
		inbox.ThreadStatus = string(thread.Status)
		inbox.Participants = append([]string(nil), thread.ParticipantAgentIDs...)
	}

	messages := b.store.ListMailbox(agentID, inbox.ThreadID, params.Status)
	inbox.Messages = make([]tools.AgentInboxMessage, 0, len(messages))
	for _, message := range messages {
		inbox.Messages = append(inbox.Messages, tools.AgentInboxMessage{
			ID:             message.ID,
			ThreadID:       message.ThreadID,
			ParentID:       message.ParentID,
			FromAgentID:    message.FromAgentID,
			Kind:           message.Kind,
			Status:         message.Status,
			ExpectReply:    message.ExpectReply,
			ContentPreview: truncateForCollaboration(message.Content, collaborationContentPreview),
			ArtifactCount:  len(message.Artifacts),
			CreatedAt:      message.CreatedAt,
		})
	}

	return inbox, nil
}

func (b *AgentCollaborationBus) enqueueRequest(sessionKey, targetAgentID, threadID, messageID string) {
	b.queueMu.Lock()
	queue := b.queues[sessionKey]
	if queue == nil {
		queue = &collaborationSessionQueue{
			targetAgentID: targetAgentID,
			threadID:      threadID,
		}
		b.queues[sessionKey] = queue
	}
	queue.targetAgentID = targetAgentID
	queue.threadID = threadID
	queue.messageIDs = append(queue.messageIDs, messageID)
	if queue.processing {
		b.queueMu.Unlock()
		return
	}
	queue.processing = true
	b.queueMu.Unlock()

	go b.processSessionQueue(sessionKey)
}

func (b *AgentCollaborationBus) processSessionQueue(sessionKey string) {
	for {
		b.queueMu.Lock()
		queue := b.queues[sessionKey]
		if queue == nil || len(queue.messageIDs) == 0 {
			if queue != nil {
				queue.processing = false
				delete(b.queues, sessionKey)
			}
			b.queueMu.Unlock()
			return
		}
		targetAgentID := queue.targetAgentID
		threadID := queue.threadID
		messageID := queue.messageIDs[0]
		queue.messageIDs = queue.messageIDs[1:]
		b.queueMu.Unlock()

		if err := b.processQueuedRequest(sessionKey, targetAgentID, threadID, messageID); err != nil {
			logger.WarnCF("agent", "Failed to process queued collaboration request", map[string]any{
				"session_key": sessionKey,
				"thread_id":   threadID,
				"message_id":  messageID,
				"error":       err.Error(),
			})
		}
	}
}

func (b *AgentCollaborationBus) processQueuedRequest(
	sessionKey, targetAgentID, threadID, messageID string,
) error {
	requestEnv, ok := b.store.GetMessage(messageID)
	if !ok {
		return fmt.Errorf("collaboration message %q not found", messageID)
	}
	if requestEnv.Status == collab.StatusBlocked || requestEnv.Status == collab.StatusClosed {
		return nil
	}

	targetAgent, ok := b.al.registry.GetAgent(targetAgentID)
	if !ok || targetAgent == nil {
		return b.recordRequestFailure(requestEnv, fmt.Errorf("target agent %q not found", targetAgentID))
	}

	if _, err := b.store.UpdateMessageStatus(messageID, collab.StatusReceived, ""); err == nil {
		if latest, ok := b.store.GetMessage(messageID); ok {
			b.emitCollaborationEvent(runtimeevents.KindAgentMessageReceived, latest, targetAgentID)
		}
	}

	inbound := &bus.InboundContext{
		Channel:   collaborationChannel,
		Account:   collaborationAccount,
		ChatID:    threadID,
		ChatType:  "direct",
		SenderID:  requestEnv.FromAgentID,
		MessageID: requestEnv.ID,
	}
	scope := collaborationSessionScope(targetAgentID, threadID)
	requestMsg := interAgentPromptMessage(b.renderRequestMessage(requestEnv), requestEnv.Artifacts)

	runCtx, cancel := b.collaborationRunContext(requestEnv.Deadline)
	defer cancel()

	if err := b.al.ensureHooksInitialized(runCtx); err != nil {
		return b.recordRequestFailure(requestEnv, err)
	}
	if err := b.al.ensureMCPInitialized(runCtx); err != nil {
		return b.recordRequestFailure(requestEnv, err)
	}

	finalContent, err := b.al.runAgentLoop(runCtx, targetAgent, processOptions{
		Dispatch: DispatchRequest{
			SessionKey:     sessionKey,
			SessionAliases: collaborationSessionAliases(targetAgentID, threadID),
			InboundContext: inbound,
			SessionScope:   scope,
		},
		DefaultResponse:      "",
		EnableSummary:        true,
		SendResponse:         false,
		SuppressToolFeedback: true,
		RootPromptMessage:    &requestMsg,
	})
	if err != nil {
		return b.recordRequestFailure(requestEnv, err)
	}

	if _, exists := b.store.FindResponseTo(threadID, requestEnv.ID); exists {
		_, _ = b.store.UpdateMessageStatus(requestEnv.ID, collab.StatusReplied, "")
		return nil
	}

	finalContent = strings.TrimSpace(finalContent)
	if finalContent == "" {
		finalContent = "No reply was produced for this collaboration request."
	}
	return b.synthesizeReply(requestEnv, targetAgentID, finalContent, collab.KindReply, true)
}

func (b *AgentCollaborationBus) synthesizeReply(
	requestEnv collab.Envelope,
	fromAgentID, content string,
	kind collab.MessageKind,
	markRequestReplied bool,
) error {
	replyEnv := collab.Envelope{
		ID:           "msg_" + uuid.New().String(),
		ThreadID:     requestEnv.ThreadID,
		ParentID:     requestEnv.ID,
		FromAgentID:  fromAgentID,
		ToAgentIDs:   []string{requestEnv.FromAgentID},
		Kind:         kind,
		Content:      strings.TrimSpace(content),
		Priority:     "normal",
		Status:       collab.StatusDelivered,
		TraceID:      requestEnv.TraceID,
		CreatedAt:    time.Now().UTC(),
		ParentTurnID: "",
	}
	if _, err := b.store.AddMessage(replyEnv); err != nil {
		return err
	}
	if markRequestReplied {
		if _, err := b.store.UpdateMessageStatus(requestEnv.ID, collab.StatusReplied, ""); err != nil {
			logger.WarnCF("agent", "Failed to mark synthesized collaboration reply", map[string]any{
				"thread_id":  requestEnv.ThreadID,
				"message_id": requestEnv.ID,
				"error":      err.Error(),
			})
		}
	}
	b.emitCollaborationEvent(runtimeevents.KindAgentMessageReply, replyEnv, requestEnv.FromAgentID)
	b.notifyWaiters(requestEnv.ID, replyEnv)
	return nil
}

func (b *AgentCollaborationBus) recordRequestFailure(requestEnv collab.Envelope, err error) error {
	if _, statusErr := b.store.UpdateMessageStatus(requestEnv.ID, collab.StatusError, err.Error()); statusErr != nil {
		logger.WarnCF("agent", "Failed to update collaboration error status", map[string]any{
			"thread_id":  requestEnv.ThreadID,
			"message_id": requestEnv.ID,
			"error":      statusErr.Error(),
		})
	}
	if synthErr := b.synthesizeReply(
		requestEnv,
		firstRecipient(requestEnv.ToAgentIDs),
		fmt.Sprintf("Collaboration request failed: %v", err),
		collab.KindStatus,
		false,
	); synthErr != nil {
		return synthErr
	}
	return err
}

func (b *AgentCollaborationBus) buildSharedContext(
	ctx context.Context,
	fromAgentID string,
	policy collab.ContextPolicy,
	selectedContext string,
) (string, error) {
	switch policy {
	case collab.ContextPolicySummary:
		sessionKey := tools.ToolSessionKey(ctx)
		if sessionKey == "" {
			return "", nil
		}
		senderAgent, ok := b.al.registry.GetAgent(fromAgentID)
		if !ok || senderAgent == nil || senderAgent.Sessions == nil {
			return "", nil
		}
		return strings.TrimSpace(senderAgent.Sessions.GetSummary(sessionKey)), nil
	case collab.ContextPolicySelectedContext:
		selectedContext = strings.TrimSpace(selectedContext)
		if selectedContext == "" {
			return "", fmt.Errorf("selected_context is required when context_policy is selected_context")
		}
		return selectedContext, nil
	default:
		return "", nil
	}
}

func (b *AgentCollaborationBus) renderRequestMessage(env collab.Envelope) string {
	var builder strings.Builder
	builder.WriteString("[Internal collaboration message]\n")
	fmt.Fprintf(&builder, "From agent: %s\n", env.FromAgentID)
	fmt.Fprintf(&builder, "Thread ID: %s\n", env.ThreadID)
	fmt.Fprintf(&builder, "Message ID: %s\n", env.ID)
	fmt.Fprintf(&builder, "Kind: %s\n", env.Kind)
	if env.ExpectReply {
		builder.WriteString("Reply expected: yes\n")
	}
	if env.Deadline != nil {
		fmt.Fprintf(&builder, "Deadline: %s\n", env.Deadline.UTC().Format(time.RFC3339))
	}
	if strings.TrimSpace(env.ContextShared) != "" {
		fmt.Fprintf(&builder, "Shared context (%s):\n%s\n", env.ContextPolicy, env.ContextShared)
	}
	if len(env.Artifacts) > 0 {
		builder.WriteString("Artifacts:\n")
		for _, artifact := range env.Artifacts {
			fmt.Fprintf(
				&builder,
				"- type=%s ref=%s url=%s filename=%s content_type=%s\n",
				strings.TrimSpace(artifact.Type),
				strings.TrimSpace(artifact.Ref),
				strings.TrimSpace(artifact.URL),
				strings.TrimSpace(artifact.Filename),
				strings.TrimSpace(artifact.ContentType),
			)
		}
	}
	builder.WriteString("\nTask:\n")
	builder.WriteString(env.Content)
	builder.WriteString("\n\nUse the collaboration tools to reply inside this thread when appropriate.")
	return builder.String()
}

func (b *AgentCollaborationBus) waitForResponse(
	ctx context.Context,
	threadID, messageID string,
	waiter chan collab.Envelope,
	deadline *time.Time,
) (collab.Envelope, error) {
	if response, ok := b.store.FindResponseTo(threadID, messageID); ok {
		return response, nil
	}

	timeout, err := b.collaborationWaitTimeout(deadline)
	if err != nil {
		return collab.Envelope{}, err
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case response := <-waiter:
		return response, nil
	case <-ctx.Done():
		return collab.Envelope{}, ctx.Err()
	case <-timer.C:
		return collab.Envelope{}, fmt.Errorf(
			"timed out waiting for a collaboration reply in thread %s for message %s",
			threadID,
			messageID,
		)
	}
}

func (b *AgentCollaborationBus) collaborationRunContext(deadline *time.Time) (context.Context, context.CancelFunc) {
	if deadline != nil && !deadline.IsZero() {
		return context.WithDeadline(context.Background(), *deadline)
	}
	return context.WithTimeout(context.Background(), b.defaultCollaborationTimeout())
}

func (b *AgentCollaborationBus) collaborationWaitTimeout(deadline *time.Time) (time.Duration, error) {
	if deadline != nil {
		timeout := time.Until(*deadline)
		if timeout <= 0 {
			return 0, fmt.Errorf("collaboration reply deadline expired before a response was received")
		}
		return timeout, nil
	}
	return b.defaultCollaborationTimeout(), nil
}

func (b *AgentCollaborationBus) defaultCollaborationTimeout() time.Duration {
	if b != nil && b.al != nil {
		timeout := b.al.getSubTurnConfig().defaultTimeout
		if timeout > 0 {
			return timeout
		}
	}
	return defaultCollaborationTimeout
}

func (b *AgentCollaborationBus) addWaiter(messageID string, waiter chan collab.Envelope) {
	b.waitersMu.Lock()
	defer b.waitersMu.Unlock()
	b.waiters[messageID] = append(b.waiters[messageID], waiter)
}

func (b *AgentCollaborationBus) removeWaiter(messageID string, waiter chan collab.Envelope) {
	b.waitersMu.Lock()
	defer b.waitersMu.Unlock()

	waiters := b.waiters[messageID]
	if len(waiters) == 0 {
		return
	}
	filtered := waiters[:0]
	for _, current := range waiters {
		if current != waiter {
			filtered = append(filtered, current)
		}
	}
	if len(filtered) == 0 {
		delete(b.waiters, messageID)
		return
	}
	b.waiters[messageID] = filtered
}

func (b *AgentCollaborationBus) notifyWaiters(messageID string, env collab.Envelope) {
	b.waitersMu.Lock()
	waiters := append([]chan collab.Envelope(nil), b.waiters[messageID]...)
	delete(b.waiters, messageID)
	b.waitersMu.Unlock()

	for _, waiter := range waiters {
		select {
		case waiter <- env:
		default:
		}
	}
}

func (b *AgentCollaborationBus) appendReplyHistory(ctx context.Context, replyEnv collab.Envelope) {
	ts := turnStateFromContext(ctx)
	if ts == nil || ts.agent == nil || ts.agent.Sessions == nil {
		return
	}
	scope := collaborationSessionScope(replyEnv.FromAgentID, replyEnv.ThreadID)
	expectedSessionKey := collaborationSessionKey(replyEnv.FromAgentID, replyEnv.ThreadID)
	if ts.sessionKey != expectedSessionKey {
		return
	}
	ensureSessionMetadata(
		ts.agent.Sessions,
		expectedSessionKey,
		scope,
		collaborationSessionAliases(replyEnv.FromAgentID, replyEnv.ThreadID),
	)
	msg := providers.Message{
		Role: "assistant",
		Content: fmt.Sprintf(
			"[Collaboration reply to %s]\n%s",
			strings.Join(replyEnv.ToAgentIDs, ", "),
			replyEnv.Content,
		),
		Attachments: append([]providers.Attachment(nil), replyEnv.Artifacts...),
	}
	ts.agent.Sessions.AddFullMessage(expectedSessionKey, msg)
}

func (b *AgentCollaborationBus) enforceThreadPolicy(
	fromAgentID string,
	toAgentIDs []string,
	threadID string,
	contentLen, artifactCount int,
) error {
	limit := b.al.registry.CommunicationMaxThreadTurns(fromAgentID)
	charLimit := b.al.registry.CommunicationMaxMessageChars(fromAgentID)
	artifactsAllowed := b.al.registry.CommunicationAllowsArtifacts(fromAgentID)
	for _, toAgentID := range toAgentIDs {
		if turns := b.al.registry.CommunicationMaxThreadTurns(toAgentID); turns > 0 && turns < limit {
			limit = turns
		}
		if chars := b.al.registry.CommunicationMaxMessageChars(toAgentID); chars > 0 && chars < charLimit {
			charLimit = chars
		}
		artifactsAllowed = artifactsAllowed && b.al.registry.CommunicationAllowsArtifacts(toAgentID)
	}
	if contentLen > charLimit {
		return fmt.Errorf("collaboration message exceeds max_message_chars=%d", charLimit)
	}
	if artifactCount > 0 && !artifactsAllowed {
		return fmt.Errorf("artifact transfer is not allowed by collaboration policy")
	}
	if limit > 0 && b.store.ThreadMessageCount(threadID) >= limit {
		if _, err := b.store.CloseThread(threadID, "max_thread_turns exceeded"); err == nil {
			if closed, ok := b.store.GetThread(threadID); ok {
				b.emitThreadClosedEvent(closed)
			}
		}
		return fmt.Errorf("collaboration thread %s reached max_thread_turns=%d", threadID, limit)
	}
	return nil
}

func (b *AgentCollaborationBus) emitCollaborationEvent(
	kind runtimeevents.Kind,
	env collab.Envelope,
	toAgentID string,
) {
	if b == nil || b.al == nil {
		return
	}
	meta := HookMeta{
		AgentID:      env.FromAgentID,
		SessionKey:   collaborationSessionKey(firstNonEmpty(toAgentID, firstRecipient(env.ToAgentIDs)), env.ThreadID),
		ParentTurnID: env.ParentTurnID,
		TracePath:    env.TraceID,
		Source:       "collaboration",
		turnContext: &TurnContext{
			Inbound: &bus.InboundContext{
				Channel:   collaborationChannel,
				Account:   collaborationAccount,
				ChatID:    env.ThreadID,
				ChatType:  "direct",
				SenderID:  env.FromAgentID,
				MessageID: env.ID,
			},
		},
	}
	b.al.emitEvent(kind, meta, AgentMessageEventPayload{
		ThreadID:        env.ThreadID,
		MessageID:       env.ID,
		ParentMessageID: env.ParentID,
		FromAgentID:     env.FromAgentID,
		ToAgentID:       strings.TrimSpace(toAgentID),
		Status:          string(env.Status),
		TraceID:         env.TraceID,
		ParentTurnID:    env.ParentTurnID,
		Priority:        env.Priority,
		Deadline:        env.Deadline,
		ArtifactCount:   len(env.Artifacts),
	})
}

func (b *AgentCollaborationBus) emitThreadClosedEvent(thread collab.Thread) {
	if b == nil || b.al == nil {
		return
	}
	b.al.emitEvent(runtimeevents.KindAgentThreadClosed, HookMeta{
		AgentID:    firstRecipient(thread.ParticipantAgentIDs),
		SessionKey: "",
		Source:     "collaboration",
		TracePath:  "thread_closed:" + thread.ID,
	}, AgentThreadClosedPayload{
		ThreadID:     thread.ID,
		Status:       string(thread.Status),
		CloseReason:  thread.CloseReason,
		Participants: append([]string(nil), thread.ParticipantAgentIDs...),
	})
}

func collaborationSessionScope(targetAgentID, threadID string) *session.SessionScope {
	return &session.SessionScope{
		Version:    session.ScopeVersionV1,
		AgentID:    routing.NormalizeAgentID(targetAgentID),
		Channel:    collaborationChannel,
		Account:    collaborationAccount,
		Dimensions: []string{"thread"},
		Values: map[string]string{
			"thread": strings.TrimSpace(threadID),
		},
	}
}

func collaborationSessionKey(targetAgentID, threadID string) string {
	return session.BuildSessionKey(*collaborationSessionScope(targetAgentID, threadID))
}

func collaborationSessionAliases(targetAgentID, threadID string) []string {
	return []string{
		strings.ToLower(fmt.Sprintf(
			"agent:%s:%s:thread:%s",
			routing.NormalizeAgentID(targetAgentID),
			collaborationChannel,
			strings.TrimSpace(threadID),
		)),
	}
}

func currentToolAgentID(ctx context.Context) string {
	if agentID := routing.NormalizeAgentID(tools.ToolAgentID(ctx)); agentID != "" {
		return agentID
	}
	if ts := turnStateFromContext(ctx); ts != nil {
		return routing.NormalizeAgentID(ts.agentID)
	}
	return ""
}

func currentTurnID(ctx context.Context) string {
	if ts := turnStateFromContext(ctx); ts != nil {
		return strings.TrimSpace(ts.turnID)
	}
	return ""
}

func replyRecipients(
	parentMsg collab.Envelope,
	fromAgentID string,
	participants []string,
) []string {
	recipients := make([]string, 0, 1)
	if parentMsg.FromAgentID != "" && parentMsg.FromAgentID != fromAgentID {
		recipients = append(recipients, parentMsg.FromAgentID)
	}
	for _, participant := range participants {
		if participant == "" || participant == fromAgentID || containsString(recipients, participant) {
			continue
		}
		recipients = append(recipients, participant)
	}
	return recipients
}

func firstRecipient(values []string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func threadIncludesParticipant(thread collab.Thread, agentID string) bool {
	agentID = routing.NormalizeAgentID(agentID)
	for _, participant := range thread.ParticipantAgentIDs {
		if routing.NormalizeAgentID(participant) == agentID {
			return true
		}
	}
	return false
}

func truncateForCollaboration(content string, limit int) string {
	content = strings.TrimSpace(content)
	if limit <= 0 || len(content) <= limit {
		return content
	}
	if limit <= 3 {
		return content[:limit]
	}
	return content[:limit-3] + "..."
}

func collaborationPayloadCharCount(content, sharedContext string) int {
	return utf8.RuneCountInString(strings.TrimSpace(content)) +
		utf8.RuneCountInString(strings.TrimSpace(sharedContext))
}

func (b *AgentCollaborationBus) isQueueActive(sessionKey string) bool {
	if b == nil {
		return false
	}
	b.queueMu.Lock()
	defer b.queueMu.Unlock()
	queue := b.queues[sessionKey]
	return queue != nil && (queue.processing || len(queue.messageIDs) > 0)
}
