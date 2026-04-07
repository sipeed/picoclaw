package pico

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func newTestPicoChannel(t *testing.T) *PicoChannel {
	t.Helper()

	cfg := config.PicoConfig{}
	cfg.SetToken("test-token")
	ch, err := NewPicoChannel(cfg, bus.NewMessageBus())
	if err != nil {
		t.Fatalf("NewPicoChannel: %v", err)
	}

	ch.ctx = context.Background()
	ch.SetRunning(true)
	return ch
}

func TestCreateAndAddConnection_RespectsMaxConnectionsConcurrently(t *testing.T) {
	ch := newTestPicoChannel(t)

	const (
		maxConns   = 5
		goroutines = 64
		sessionID  = "session-a"
	)

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	errCount := 0

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			pc, err := ch.createAndAddConnection(nil, sessionID, maxConns)
			mu.Lock()
			defer mu.Unlock()

			if err == nil {
				successCount++
				if pc == nil {
					t.Errorf("pc is nil on success")
				}
				return
			}
			if !errors.Is(err, channels.ErrTemporary) {
				t.Errorf("unexpected error: %v", err)
				return
			}
			errCount++
		}()
	}
	wg.Wait()

	if successCount > maxConns {
		t.Fatalf("successCount=%d > maxConns=%d", successCount, maxConns)
	}
	if successCount+errCount != goroutines {
		t.Fatalf("success=%d err=%d total=%d want=%d", successCount, errCount, successCount+errCount, goroutines)
	}
	if got := ch.currentConnCount(); got != maxConns {
		t.Fatalf("currentConnCount=%d want=%d", got, maxConns)
	}
}

func TestRemoveConnection_CleansBothIndexes(t *testing.T) {
	ch := newTestPicoChannel(t)

	pc, err := ch.createAndAddConnection(nil, "session-cleanup", 10)
	if err != nil {
		t.Fatalf("createAndAddConnection: %v", err)
	}

	removed := ch.removeConnection(pc.id)
	if removed == nil {
		t.Fatal("removeConnection returned nil")
	}

	ch.connsMu.RLock()
	defer ch.connsMu.RUnlock()

	if _, ok := ch.connections[pc.id]; ok {
		t.Fatalf("connID %s still exists in connections", pc.id)
	}
	if _, ok := ch.sessionConnections[pc.sessionID]; ok {
		t.Fatalf("session %s still exists in sessionConnections", pc.sessionID)
	}
	if got := len(ch.connections); got != 0 {
		t.Fatalf("len(connections)=%d want=0", got)
	}
}

func TestBroadcastToSession_TargetsOnlyRequestedSession(t *testing.T) {
	ch := newTestPicoChannel(t)

	target := &picoConn{id: "target", sessionID: "s-target"}
	target.closed.Store(true)
	ch.addConnForTest(target)

	other := &picoConn{id: "other", sessionID: "s-other"}
	ch.addConnForTest(other)

	err := ch.broadcastToSession("pico:s-target", newMessage(TypeMessageCreate, map[string]any{"content": "hello"}))
	if err == nil {
		t.Fatal("expected send failure due to closed target connection")
	}
	if !errors.Is(err, channels.ErrSendFailed) {
		t.Fatalf("expected ErrSendFailed, got %v", err)
	}
}

func TestDeleteMessage_SendsDeleteEvent(t *testing.T) {
	ch := newTestPicoChannel(t)

	messages := make([]PicoMessage, 0, 1)
	conn := &picoConn{
		id:        "delete",
		sessionID: "sess-del",
		writeJSONFunc: func(v any) error {
			msg, ok := v.(PicoMessage)
			if ok {
				messages = append(messages, msg)
			}
			return nil
		},
	}
	ch.addConnForTest(conn)

	if err := ch.DeleteMessage(context.Background(), "pico:sess-del", "msg-1"); err != nil {
		t.Fatalf("DeleteMessage() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 delete message, got %d", len(messages))
	}
	if messages[0].Type != TypeMessageDelete {
		t.Fatalf("message type = %q, want %q", messages[0].Type, TypeMessageDelete)
	}
	if messages[0].Payload["message_id"] != "msg-1" {
		t.Fatalf("message_id = %v, want msg-1", messages[0].Payload["message_id"])
	}
}

func TestBeginStream_SendsCreateAndUpdatesMessage(t *testing.T) {
	ch := newTestPicoChannel(t)

	messages := make([]PicoMessage, 0, 3)
	conn := &picoConn{
		id:        "stream",
		sessionID: "sess-1",
		writeJSONFunc: func(v any) error {
			msg, ok := v.(PicoMessage)
			if ok {
				messages = append(messages, msg)
			}
			return nil
		},
	}
	ch.addConnForTest(conn)

	streamer, err := ch.BeginStream(context.Background(), "pico:sess-1")
	if err != nil {
		t.Fatalf("BeginStream() error = %v", err)
	}
	if len(messages) == 0 {
		t.Fatal("expected initial message.create for stream placeholder")
	}
	createPayload := messages[0].Payload
	messageID, _ := createPayload["message_id"].(string)
	if messageID == "" {
		t.Fatal("expected message_id in initial stream create payload")
	}
	if err := streamer.Update(context.Background(), "hello"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := streamer.Finalize(context.Background(), "hello world"); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
	if len(messages) < 3 {
		t.Fatalf("expected at least 3 messages, got %d", len(messages))
	}
	updatePayload := messages[1].Payload
	if updatePayload["message_id"] != messageID {
		t.Fatalf("update message_id = %v, want %s", updatePayload["message_id"], messageID)
	}
	if updatePayload["content"] != "hello" {
		t.Fatalf("update content = %v, want hello", updatePayload["content"])
	}
	finalPayload := messages[2].Payload
	if finalPayload["content"] != "hello world" {
		t.Fatalf("final content = %v, want hello world", finalPayload["content"])
	}
}

func (c *PicoChannel) addConnForTest(pc *picoConn) {
	c.connsMu.Lock()
	defer c.connsMu.Unlock()
	if c.connections == nil {
		c.connections = make(map[string]*picoConn)
	}
	if c.sessionConnections == nil {
		c.sessionConnections = make(map[string]map[string]*picoConn)
	}
	if _, exists := c.connections[pc.id]; exists {
		panic(fmt.Sprintf("duplicate conn id in test: %s", pc.id))
	}
	c.connections[pc.id] = pc
	bySession, ok := c.sessionConnections[pc.sessionID]
	if !ok {
		bySession = make(map[string]*picoConn)
		c.sessionConnections[pc.sessionID] = bySession
	}
	bySession[pc.id] = pc
}
