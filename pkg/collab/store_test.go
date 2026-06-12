package collab

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStore_PersistsThreadsMessagesAndMailboxes(t *testing.T) {
	dir := t.TempDir()

	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	deadline := time.Now().UTC().Add(2 * time.Minute).Truncate(time.Second)
	if _, ensureErr := store.EnsureThread(Thread{
		ID:                  "thread_persist",
		ParticipantAgentIDs: []string{"planner", "research"},
		Status:              ThreadStatusOpen,
	}); ensureErr != nil {
		t.Fatalf("EnsureThread() error = %v", ensureErr)
	}
	if _, addRequestErr := store.AddMessage(Envelope{
		ID:           "msg_request",
		ThreadID:     "thread_persist",
		FromAgentID:  "planner",
		ToAgentIDs:   []string{"research"},
		Kind:         KindRequest,
		Content:      "Investigate mailbox durability.",
		ExpectReply:  true,
		Status:       StatusDelivered,
		TraceID:      "trace_persist",
		ParentTurnID: "turn_planner",
		Deadline:     &deadline,
	}); addRequestErr != nil {
		t.Fatalf("AddMessage(request) error = %v", addRequestErr)
	}
	if _, addReplyErr := store.AddMessage(Envelope{
		ID:           "msg_reply",
		ThreadID:     "thread_persist",
		ParentID:     "msg_request",
		FromAgentID:  "research",
		ToAgentIDs:   []string{"planner"},
		Kind:         KindReply,
		Content:      "Mailbox durability confirmed.",
		Status:       StatusDelivered,
		TraceID:      "trace_persist",
		ParentTurnID: "turn_research",
	}); addReplyErr != nil {
		t.Fatalf("AddMessage(reply) error = %v", addReplyErr)
	}
	if _, updateErr := store.UpdateMessageStatus("msg_request", StatusReplied, ""); updateErr != nil {
		t.Fatalf("UpdateMessageStatus() error = %v", updateErr)
	}

	reloaded, err := NewStore(filepath.Clean(dir))
	if err != nil {
		t.Fatalf("NewStore(reload) error = %v", err)
	}

	thread, ok := reloaded.GetThread("thread_persist")
	if !ok {
		t.Fatal("expected persisted thread")
	}
	if thread.Status != ThreadStatusOpen {
		t.Fatalf("thread.Status = %q, want open", thread.Status)
	}
	if len(thread.MessageIDs) != 2 {
		t.Fatalf("thread.MessageIDs = %v, want 2 messages", thread.MessageIDs)
	}

	request, ok := reloaded.GetMessage("msg_request")
	if !ok {
		t.Fatal("expected persisted request message")
	}
	if request.Status != StatusReplied {
		t.Fatalf("request.Status = %q, want replied", request.Status)
	}
	if request.Deadline == nil || !request.Deadline.Equal(deadline) {
		t.Fatalf("request.Deadline = %v, want %v", request.Deadline, deadline)
	}

	reply, ok := reloaded.FindResponseTo("thread_persist", "msg_request")
	if !ok {
		t.Fatal("expected persisted reply lookup")
	}
	if reply.Content != "Mailbox durability confirmed." {
		t.Fatalf("reply.Content = %q", reply.Content)
	}

	plannerMailbox := reloaded.ListMailbox("planner", "thread_persist", "")
	if len(plannerMailbox) != 1 || plannerMailbox[0].ID != "msg_reply" {
		t.Fatalf("planner mailbox = %+v", plannerMailbox)
	}

	researchMailbox := reloaded.ListMailbox("research", "thread_persist", "")
	if len(researchMailbox) != 1 || researchMailbox[0].ID != "msg_request" {
		t.Fatalf("research mailbox = %+v", researchMailbox)
	}
}
