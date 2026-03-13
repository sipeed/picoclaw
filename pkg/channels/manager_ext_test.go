package channels

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/sipeed/picoclaw/pkg/bus"
)

// mockEditorWithSendID implements MessageEditor and MessageSenderWithID.
type mockEditorWithSendID struct {
	mockChannel
	editFn     func(ctx context.Context, chatID, messageID, content string) error
	sendWithID func(ctx context.Context, chatID, content string) (string, error)
}

func (m *mockEditorWithSendID) EditMessage(
	ctx context.Context, chatID, messageID, content string,
) error {
	return m.editFn(ctx, chatID, messageID, content)
}

func (m *mockEditorWithSendID) SendWithID(ctx context.Context, chatID, content string) (string, error) {
	return m.sendWithID(ctx, chatID, content)
}

func TestHandleStatusSend_EditsPlaceholder(t *testing.T) {
	m := newTestManager()
	var editCalled bool
	var editedContent string

	ch := &mockEditorWithSendID{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		editFn: func(_ context.Context, _, messageID, content string) error {
			editCalled = true
			editedContent = content
			if messageID != "ph-42" {
				t.Fatalf("expected messageID ph-42, got %s", messageID)
			}
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			t.Fatal("SendWithID should not be called when placeholder exists")
			return "", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	m.RecordPlaceholder("test", "123", "ph-42")

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "status update 1", IsStatus: true}
	m.handleStatusSend(context.Background(), "test", w, msg)

	if !editCalled {
		t.Fatal("expected EditMessage to be called on placeholder")
	}
	if editedContent != "status update 1" {
		t.Fatalf("expected content 'status update 1', got %s", editedContent)
	}
}

func TestHandleStatusSend_EditsTrackedStatus(t *testing.T) {
	m := newTestManager()
	var editCalled bool

	ch := &mockEditorWithSendID{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		editFn: func(_ context.Context, _, messageID, _ string) error {
			editCalled = true
			if messageID != "status-99" {
				t.Fatalf("expected messageID status-99, got %s", messageID)
			}
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			t.Fatal("SendWithID should not be called when statusMsgID exists")
			return "", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	m.statusMsgIDs.Store("test:123", statusMsgEntry{messageID: "status-99", createdAt: time.Now()})

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "update 2", IsStatus: true}
	m.handleStatusSend(context.Background(), "test", w, msg)

	if !editCalled {
		t.Fatal("expected EditMessage to be called on tracked status message")
	}
}

func TestHandleStatusSend_SendsNewAndTracks(t *testing.T) {
	m := newTestManager()
	var sendWithIDCalled bool

	ch := &mockEditorWithSendID{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		editFn: func(_ context.Context, _, _, _ string) error {
			return nil
		},
		sendWithID: func(_ context.Context, chatID, content string) (string, error) {
			sendWithIDCalled = true
			if chatID != "123" {
				t.Fatalf("expected chatID 123, got %s", chatID)
			}
			return "new-msg-1", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "first status", IsStatus: true}
	m.handleStatusSend(context.Background(), "test", w, msg)

	if !sendWithIDCalled {
		t.Fatal("expected SendWithID to be called")
	}

	v, ok := m.statusMsgIDs.Load("test:123")
	if !ok {
		t.Fatal("expected statusMsgIDs to contain tracked entry")
	}
	entry := v.(statusMsgEntry)
	if entry.messageID != "new-msg-1" {
		t.Fatalf("expected messageID new-msg-1, got %s", entry.messageID)
	}
}

func TestHandleTaskStatusSend_EditsExisting(t *testing.T) {
	m := newTestManager()
	var editCalled bool

	ch := &mockEditorWithSendID{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		editFn: func(_ context.Context, _, messageID, content string) error {
			editCalled = true
			if messageID != "task-msg-1" {
				t.Fatalf("expected messageID task-msg-1, got %s", messageID)
			}
			if content != "task progress 50%" {
				t.Fatalf("expected content 'task progress 50%%', got %s", content)
			}
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			t.Fatal("SendWithID should not be called when task message exists")
			return "", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	m.taskMsgIDs.Store(
		taskStatusKey("test", "123", "task-abc"),
		statusMsgEntry{messageID: "task-msg-1", createdAt: time.Now()},
	)

	msg := bus.OutboundMessage{
		Channel:      "test",
		ChatID:       "123",
		Content:      "task progress 50%",
		IsTaskStatus: true,
		TaskID:       "task-abc",
	}
	m.handleTaskStatusSend(context.Background(), "test", w, msg)

	if !editCalled {
		t.Fatal("expected EditMessage to be called")
	}
}

func TestHandleTaskStatusSend_SendsNewAndTracks(t *testing.T) {
	m := newTestManager()
	var sendWithIDCalled bool

	ch := &mockEditorWithSendID{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		editFn: func(_ context.Context, _, _, _ string) error { return nil },
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			sendWithIDCalled = true
			return "new-task-msg", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	msg := bus.OutboundMessage{
		Channel:      "test",
		ChatID:       "123",
		Content:      "task started",
		IsTaskStatus: true,
		TaskID:       "task-xyz",
	}
	m.handleTaskStatusSend(context.Background(), "test", w, msg)

	if !sendWithIDCalled {
		t.Fatal("expected SendWithID to be called")
	}

	v, ok := m.taskMsgIDs.Load(taskStatusKey("test", "123", "task-xyz"))
	if !ok {
		t.Fatal("expected taskMsgIDs to contain tracked entry")
	}
	entry := v.(statusMsgEntry)
	if entry.messageID != "new-task-msg" {
		t.Fatalf("expected messageID new-task-msg, got %s", entry.messageID)
	}
}

func TestHandleTaskStatusSend_FallbackToSend(t *testing.T) {
	m := newTestManager()
	var sendCalled bool

	ch := &mockChannel{
		sendFn: func(_ context.Context, msg bus.OutboundMessage) error {
			sendCalled = true
			if msg.Content != "task status" {
				t.Fatalf("expected content 'task status', got %s", msg.Content)
			}
			return nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	msg := bus.OutboundMessage{
		Channel:      "test",
		ChatID:       "123",
		Content:      "task status",
		IsTaskStatus: true,
		TaskID:       "task-fallback",
	}
	m.handleTaskStatusSend(context.Background(), "test", w, msg)

	if !sendCalled {
		t.Fatal("expected fallback Send to be called")
	}
}

func TestPreSend_EditsStatusMessage(t *testing.T) {
	m := newTestManager()
	var editCalled bool

	ch := &mockMessageEditor{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		editFn: func(_ context.Context, _, messageID, _ string) error {
			editCalled = true
			if messageID != "status-msg-77" {
				t.Fatalf("expected messageID status-msg-77, got %s", messageID)
			}
			return nil
		},
	}

	m.statusMsgIDs.Store("test:123", statusMsgEntry{messageID: "status-msg-77", createdAt: time.Now()})

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "final response"}
	edited := m.preSend(context.Background(), "test", msg, ch)

	if !edited {
		t.Fatal("expected preSend to return true (status message edited)")
	}
	if !editCalled {
		t.Fatal("expected EditMessage to be called")
	}

	if _, loaded := m.statusMsgIDs.Load("test:123"); loaded {
		t.Fatal("expected statusMsgIDs entry to be deleted after preSend")
	}
}

func TestRunWorker_RoutesStatusMessages(t *testing.T) {
	m := newTestManager()

	var regularSendCount atomic.Int32
	var sendWithIDCount atomic.Int32

	ch := &mockEditorWithSendID{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
				regularSendCount.Add(1)
				return nil
			},
		},
		editFn: func(_ context.Context, _, _, _ string) error {
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			sendWithIDCount.Add(1)
			return "tracked-1", nil
		},
	}

	w := &channelWorker{
		ch:      ch,
		queue:   make(chan bus.OutboundMessage, 10),
		done:    make(chan struct{}),
		limiter: rate.NewLimiter(rate.Inf, 1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go m.runWorker(ctx, "test", w)

	w.queue <- bus.OutboundMessage{Channel: "test", ChatID: "1", Content: "status", IsStatus: true}

	w.queue <- bus.OutboundMessage{Channel: "test", ChatID: "2", Content: "task", IsTaskStatus: true, TaskID: "t1"}

	w.queue <- bus.OutboundMessage{Channel: "test", ChatID: "3", Content: "hello"}

	time.Sleep(200 * time.Millisecond)

	if regularSendCount.Load() != 1 {
		t.Fatalf("expected 1 regular Send call, got %d", regularSendCount.Load())
	}
	if sendWithIDCount.Load() != 2 {
		t.Fatalf("expected 2 SendWithID calls (status + task), got %d", sendWithIDCount.Load())
	}
}

func TestStatusMsgTTLJanitor(t *testing.T) {
	m := newTestManager()

	m.statusMsgIDs.Store("test:old", statusMsgEntry{
		messageID: "old-status",
		createdAt: time.Now().Add(-10 * time.Minute),
	})
	m.taskMsgIDs.Store("task-old", statusMsgEntry{
		messageID: "old-task",
		createdAt: time.Now().Add(-60 * time.Minute),
	})

	m.statusMsgIDs.Store("test:fresh", statusMsgEntry{
		messageID: "fresh-status",
		createdAt: time.Now(),
	})

	now := time.Now()
	m.statusMsgIDs.Range(func(key, value any) bool {
		if entry, ok := value.(statusMsgEntry); ok {
			if now.Sub(entry.createdAt) > statusMsgTTL {
				m.statusMsgIDs.Delete(key)
			}
		}
		return true
	})
	m.taskMsgIDs.Range(func(key, value any) bool {
		if entry, ok := value.(statusMsgEntry); ok {
			if now.Sub(entry.createdAt) > taskMsgTTL {
				m.taskMsgIDs.Delete(key)
			}
		}
		return true
	})

	if _, loaded := m.statusMsgIDs.Load("test:old"); loaded {
		t.Fatal("expected old status entry to be evicted")
	}
	if _, loaded := m.taskMsgIDs.Load("task-old"); loaded {
		t.Fatal("expected old task entry to be evicted")
	}
	if _, loaded := m.statusMsgIDs.Load("test:fresh"); !loaded {
		t.Fatal("expected fresh status entry to survive")
	}
}

// mockDraftSender implements DraftSender + MessageSenderWithID + MessageEditor.
type mockDraftSender struct {
	mockChannel
	draftFn    func(ctx context.Context, chatID string, draftID int, content string) error
	editFn     func(ctx context.Context, chatID, messageID, content string) error
	sendWithID func(ctx context.Context, chatID, content string) (string, error)
}

func (m *mockDraftSender) EditMessage(
	ctx context.Context, chatID, messageID, content string,
) error {
	return m.editFn(ctx, chatID, messageID, content)
}

func (m *mockDraftSender) SendDraft(ctx context.Context, chatID string, draftID int, content string) error {
	return m.draftFn(ctx, chatID, draftID, content)
}

func (m *mockDraftSender) SendWithID(ctx context.Context, chatID, content string) (string, error) {
	return m.sendWithID(ctx, chatID, content)
}

func TestHandleStatusSend_UsesDraftSender(t *testing.T) {
	m := newTestManager()
	var draftCalled bool
	var draftContent string
	var draftDID int

	ch := &mockDraftSender{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		draftFn: func(_ context.Context, chatID string, draftID int, content string) error {
			draftCalled = true
			draftContent = content
			draftDID = draftID
			return nil
		},
		editFn: func(_ context.Context, _, _, _ string) error {
			t.Fatal("EditMessage should not be called when draft succeeds")
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			t.Fatal("SendWithID should not be called when draft succeeds")
			return "", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "streaming preview", IsStatus: true}
	m.handleStatusSend(context.Background(), "test", w, msg)

	if !draftCalled {
		t.Fatal("expected SendDraft to be called")
	}
	if draftContent != "streaming preview" {
		t.Fatalf("expected draft content 'streaming preview', got %s", draftContent)
	}
	if draftDID == 0 {
		t.Fatal("expected non-zero draftID")
	}

	draftCalled = false
	var secondDID int
	ch.draftFn = func(_ context.Context, _ string, draftID int, _ string) error {
		draftCalled = true
		secondDID = draftID
		return nil
	}
	msg.Content = "streaming preview updated"
	m.handleStatusSend(context.Background(), "test", w, msg)

	if !draftCalled {
		t.Fatal("expected SendDraft to be called again")
	}
	if secondDID != draftDID {
		t.Fatalf("expected same draftID %d, got %d", draftDID, secondDID)
	}
}

func TestHandleStatusSend_DraftFails_FallsToEdit(t *testing.T) {
	m := newTestManager()
	var editCalled bool

	ch := &mockDraftSender{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		draftFn: func(_ context.Context, _ string, _ int, _ string) error {
			return fmt.Errorf("draft not supported in group")
		},
		editFn: func(_ context.Context, _, _, _ string) error {
			editCalled = true
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			return "msg-1", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "preview", IsStatus: true}
	m.handleStatusSend(context.Background(), "test", w, msg)

	if editCalled {
		t.Fatal("expected EditMessage NOT to be called (no placeholder)")
	}
}

func TestHandleStatusSend_DraftFailure_DoesNotClobberTrackedMessageID(t *testing.T) {
	m := newTestManager()
	var sendWithIDCount int
	var editCount int
	var editedMessageID string

	ch := &mockDraftSender{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		draftFn: func(_ context.Context, _ string, _ int, _ string) error {
			return fmt.Errorf("draft unsupported")
		},
		editFn: func(_ context.Context, _, messageID, _ string) error {
			editCount++
			editedMessageID = messageID
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			sendWithIDCount++
			return "msg-1", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	msg := bus.OutboundMessage{Channel: "test", ChatID: "group-main", Content: "preview-1", IsStatus: true}
	m.handleStatusSend(context.Background(), "test", w, msg)

	msg.Content = "preview-2"
	m.handleStatusSend(context.Background(), "test", w, msg)

	if sendWithIDCount != 1 {
		t.Fatalf("expected SendWithID to be called once, got %d", sendWithIDCount)
	}
	if editCount != 1 {
		t.Fatalf("expected EditMessage to be called once, got %d", editCount)
	}
	if editedMessageID != "msg-1" {
		t.Fatalf("expected EditMessage target msg-1, got %s", editedMessageID)
	}
}

func TestHandleTaskStatusSend_UsesDraftSender(t *testing.T) {
	m := newTestManager()
	var draftCalled bool

	ch := &mockDraftSender{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		draftFn: func(_ context.Context, _ string, _ int, _ string) error {
			draftCalled = true
			return nil
		},
		editFn: func(_ context.Context, _, _, _ string) error {
			t.Fatal("EditMessage should not be called when draft succeeds")
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			t.Fatal("SendWithID should not be called when draft succeeds")
			return "", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	msg := bus.OutboundMessage{
		Channel:      "test",
		ChatID:       "123",
		Content:      "task progress 50%",
		IsTaskStatus: true,
		TaskID:       "task-draft",
	}
	m.handleTaskStatusSend(context.Background(), "test", w, msg)

	if !draftCalled {
		t.Fatal("expected SendDraft to be called for task status")
	}
}

func TestHandleTaskStatusSend_Final_UpdatesDraftInPlace(t *testing.T) {
	m := newTestManager()
	var draftUpdateCalled bool
	var draftUpdateDraftID int
	var draftUpdateContent string

	ch := &mockDraftSender{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error {
				t.Fatal("Send should not be called when draft update succeeds")
				return nil
			},
		},
		draftFn: func(_ context.Context, chatID string, draftID int, content string) error {
			draftUpdateCalled = true
			draftUpdateDraftID = draftID
			draftUpdateContent = content
			if chatID != "123" {
				t.Fatalf("expected chatID 123, got %s", chatID)
			}
			return nil
		},
		editFn: func(_ context.Context, _, _, _ string) error { return nil },
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			t.Fatal("SendWithID should not be called when draft update succeeds")
			return "", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	m.taskMsgIDs.Store(taskStatusKey("test", "123", "task-final"), statusMsgEntry{draftID: 42, createdAt: time.Now()})
	m.statusEditTimes.Store(taskStatusKey("test", "123", "task-final"), time.Now())

	msg := bus.OutboundMessage{
		Channel:      "test",
		ChatID:       "123",
		Content:      "task completed",
		IsTaskStatus: true,
		TaskID:       "task-final",
		Final:        true,
	}
	m.handleTaskStatusSend(context.Background(), "test", w, msg)

	if !draftUpdateCalled {
		t.Fatal("expected SendDraft to update draft with final content")
	}
	if draftUpdateDraftID != 42 {
		t.Fatalf("expected draftID 42, got %d", draftUpdateDraftID)
	}
	if draftUpdateContent != "task completed" {
		t.Fatalf("expected draft content 'task completed', got %q", draftUpdateContent)
	}
	if _, loaded := m.taskMsgIDs.Load(taskStatusKey("test", "123", "task-final")); loaded {
		t.Fatal("expected taskMsgIDs entry to be deleted for final task status")
	}
	if _, loaded := m.statusEditTimes.Load(taskStatusKey("test", "123", "task-final")); loaded {
		t.Fatal("expected statusEditTimes entry to be deleted for final task status")
	}
}

func TestHandleTaskStatusSend_DraftStreaming_IsolatedByChatThread(t *testing.T) {
	m := newTestManager()

	type draftCall struct {
		chatID  string
		draftID int
		content string
	}
	calls := make([]draftCall, 0, 2)

	ch := &mockDraftSender{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		draftFn: func(_ context.Context, chatID string, draftID int, content string) error {
			calls = append(calls, draftCall{chatID: chatID, draftID: draftID, content: content})
			return nil
		},
		editFn:     func(_ context.Context, _, _, _ string) error { return nil },
		sendWithID: func(_ context.Context, _, _ string) (string, error) { return "", nil },
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	msgA := bus.OutboundMessage{
		Channel:      "test",
		ChatID:       "-100/10",
		Content:      "A:10%",
		IsTaskStatus: true,
		TaskID:       "shared-task",
	}
	msgB := bus.OutboundMessage{
		Channel:      "test",
		ChatID:       "-100/20",
		Content:      "B:10%",
		IsTaskStatus: true,
		TaskID:       "shared-task",
	}

	m.handleTaskStatusSend(context.Background(), "test", w, msgA)
	m.handleTaskStatusSend(context.Background(), "test", w, msgB)

	if len(calls) != 2 {
		t.Fatalf("expected 2 SendDraft calls, got %d", len(calls))
	}
	if calls[0].chatID == calls[1].chatID {
		t.Fatalf("expected different chat threads, got %q and %q", calls[0].chatID, calls[1].chatID)
	}
	if calls[0].draftID == calls[1].draftID {
		t.Fatalf("expected distinct draft IDs per thread key, both got %d", calls[0].draftID)
	}

	if _, loaded := m.taskMsgIDs.Load(taskStatusKey("test", "-100/10", "shared-task")); !loaded {
		t.Fatal("expected taskMsgIDs entry for thread A")
	}
	if _, loaded := m.taskMsgIDs.Load(taskStatusKey("test", "-100/20", "shared-task")); !loaded {
		t.Fatal("expected taskMsgIDs entry for thread B")
	}
}

func TestHandleTaskStatusSend_DraftFailure_DoesNotClobberTrackedMessageID(t *testing.T) {
	m := newTestManager()
	var sendWithIDCount int
	var editCount int
	var editedMessageID string

	ch := &mockDraftSender{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		draftFn: func(_ context.Context, _ string, _ int, _ string) error {
			return fmt.Errorf("draft unsupported")
		},
		editFn: func(_ context.Context, _, messageID, _ string) error {
			editCount++
			editedMessageID = messageID
			return nil
		},
		sendWithID: func(_ context.Context, _, _ string) (string, error) {
			sendWithIDCount++
			return "task-msg-1", nil
		},
	}

	w := &channelWorker{ch: ch, limiter: rate.NewLimiter(rate.Inf, 1)}

	msg := bus.OutboundMessage{
		Channel:      "test",
		ChatID:       "group-main",
		Content:      "task-10%",
		IsTaskStatus: true,
		TaskID:       "task-1",
	}
	m.handleTaskStatusSend(context.Background(), "test", w, msg)

	msg.Content = "task-20%"
	m.handleTaskStatusSend(context.Background(), "test", w, msg)

	if sendWithIDCount != 1 {
		t.Fatalf("expected SendWithID to be called once, got %d", sendWithIDCount)
	}
	if editCount != 1 {
		t.Fatalf("expected EditMessage to be called once, got %d", editCount)
	}
	if editedMessageID != "task-msg-1" {
		t.Fatalf("expected EditMessage target task-msg-1, got %s", editedMessageID)
	}
}

func TestPreSend_ClearsDraftState(t *testing.T) {
	m := newTestManager()

	ch := &mockChannel{
		sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
	}

	m.statusMsgIDs.Store("test:123", statusMsgEntry{draftID: 42, createdAt: time.Now()})

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "final response"}
	edited := m.preSend(context.Background(), "test", msg, ch)

	if edited {
		t.Fatal("expected preSend to return false for draft-based status (sendMessage replaces draft)")
	}

	if _, loaded := m.statusMsgIDs.Load("test:123"); loaded {
		t.Fatal("expected draft status entry to be deleted after preSend")
	}
}

func TestGenerateDraftID_Stable(t *testing.T) {
	id1 := generateDraftID("telegram:123")
	id2 := generateDraftID("telegram:123")
	if id1 != id2 {
		t.Fatalf("expected stable draft ID, got %d vs %d", id1, id2)
	}
	if id1 == 0 {
		t.Fatal("expected non-zero draft ID")
	}

	id3 := generateDraftID("telegram:456")
	if id1 == id3 {
		t.Fatalf("expected different draft IDs for different keys, both got %d", id1)
	}
}

// TestPreSend_DismissesDraftBeforeSend verifies that preSend explicitly
// dismisses a draft-based status bubble (via SendDraft with empty text)
// before proceeding to send the permanent message.  This prevents ghost
// draft bubbles when a user message arrives between the last draft update
// and the final sendMessage.
// TestPreSend_DismissesDraftBeforeSend verifies that preSend explicitly
// dismisses a draft-based status bubble (via SendDraft with empty text)
// before proceeding to send the permanent message.  This prevents ghost
// draft bubbles when a user message arrives between the last draft update
// and the final sendMessage.
func TestPreSend_DismissesDraftBeforeSend(t *testing.T) {
	m := newTestManager()

	var dismissCalled bool
	var dismissContent string

	ch := &mockDraftSender{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		draftFn: func(_ context.Context, _ string, _ int, content string) error {
			dismissCalled = true
			dismissContent = content
			return nil
		},
		editFn:     func(_ context.Context, _, _, _ string) error { return nil },
		sendWithID: func(_ context.Context, _, _ string) (string, error) { return "", nil },
	}

	m.statusMsgIDs.Store("test:123", statusMsgEntry{draftID: 42, createdAt: time.Now()})

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "final response"}
	edited := m.preSend(context.Background(), "test", msg, ch)

	if edited {
		t.Fatal("expected preSend to return false for draft-based status")
	}
	if !dismissCalled {
		t.Fatal("expected preSend to call SendDraft to dismiss the draft")
	}
	if dismissContent != "" {
		t.Fatalf("expected empty dismiss content, got %q", dismissContent)
	}
}

// TestRecordTypingStop_CleansUpOldEntry verifies that recording a new
// typing stop function calls the previous stop first.
// TestRecordTypingStop_CleansUpOldEntry verifies that recording a new
// typing stop function calls the previous stop first.
func TestRecordTypingStop_CleansUpOldEntry(t *testing.T) {
	m := newTestManager()

	var oldStopped atomic.Bool

	m.RecordTypingStop("tg", "42", func() { oldStopped.Store(true) })

	m.RecordTypingStop("tg", "42", func() {})

	if !oldStopped.Load() {
		t.Fatal("expected old typing stop to be called when new entry is recorded")
	}
}

// TestRecordReactionUndo_CleansUpOldEntry verifies that recording a new
// reaction undo function calls the previous undo first.
// TestRecordReactionUndo_CleansUpOldEntry verifies that recording a new
// reaction undo function calls the previous undo first.
func TestRecordReactionUndo_CleansUpOldEntry(t *testing.T) {
	m := newTestManager()

	var oldUndone atomic.Bool

	m.RecordReactionUndo("tg", "42", func() { oldUndone.Store(true) })

	m.RecordReactionUndo("tg", "42", func() {})

	if !oldUndone.Load() {
		t.Fatal("expected old reaction undo to be called when new entry is recorded")
	}
}

// TestPreSend_DraftDismiss_ClearsEditTimes verifies that dismissing a draft
// in preSend also clears the statusEditTimes entry for that key, preventing
// stale throttle state from affecting the next processing cycle.
// TestPreSend_DraftDismiss_ClearsEditTimes verifies that dismissing a draft
// in preSend also clears the statusEditTimes entry for that key, preventing
// stale throttle state from affecting the next processing cycle.
func TestPreSend_DraftDismiss_ClearsEditTimes(t *testing.T) {
	m := newTestManager()

	ch := &mockDraftSender{
		mockChannel: mockChannel{
			sendFn: func(_ context.Context, _ bus.OutboundMessage) error { return nil },
		},
		draftFn:    func(_ context.Context, _ string, _ int, _ string) error { return nil },
		editFn:     func(_ context.Context, _, _, _ string) error { return nil },
		sendWithID: func(_ context.Context, _, _ string) (string, error) { return "", nil },
	}

	key := "test:123"
	m.statusMsgIDs.Store(key, statusMsgEntry{draftID: 42, createdAt: time.Now()})
	m.statusEditTimes.Store(key, time.Now())

	msg := bus.OutboundMessage{Channel: "test", ChatID: "123", Content: "final"}
	m.preSend(context.Background(), "test", msg, ch)

	if _, loaded := m.statusEditTimes.Load(key); loaded {
		t.Fatal("expected statusEditTimes to be cleared after draft dismiss")
	}
}
