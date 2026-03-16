package matrix

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"maunium.net/go/mautrix/id"

	"jane/pkg/logger"
)

type typingSession struct {
	stopCh chan struct{}
	once   sync.Once
}

func newTypingSession() *typingSession {
	return &typingSession{
		stopCh: make(chan struct{}),
	}
}

func (s *typingSession) stop() {
	s.once.Do(func() {
		close(s.stopCh)
	})
}

// StartTyping implements channels.TypingCapable.
func (c *MatrixChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	if !c.IsRunning() {
		return func() {}, nil
	}

	roomID := id.RoomID(strings.TrimSpace(chatID))
	if roomID == "" {
		return func() {}, fmt.Errorf("matrix room ID is empty")
	}

	session := newTypingSession()

	c.typingMu.Lock()
	if prev := c.typingSessions[chatID]; prev != nil {
		prev.stop()
	}
	c.typingSessions[chatID] = session
	c.typingMu.Unlock()

	parent := c.baseContext()
	go c.typingLoop(parent, roomID, session)

	var once sync.Once
	stop := func() {
		once.Do(func() {
			session.stop()
			c.typingMu.Lock()
			if current := c.typingSessions[chatID]; current == session {
				delete(c.typingSessions, chatID)
			}
			c.typingMu.Unlock()
			_, _ = c.client.UserTyping(context.Background(), roomID, false, 0)
		})
	}

	return stop, nil
}

func (c *MatrixChannel) typingLoop(ctx context.Context, roomID id.RoomID, session *typingSession) {
	sendTyping := func() {
		_, err := c.client.UserTyping(ctx, roomID, true, typingServerTTL)
		if err != nil {
			logger.DebugCF("matrix", "Failed to send typing status", map[string]any{
				"room_id": roomID.String(),
				"error":   err.Error(),
			})
		}
	}

	sendTyping()
	ticker := time.NewTicker(typingRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-session.stopCh:
			return
		case <-ticker.C:
			sendTyping()
		}
	}
}

func (c *MatrixChannel) stopTypingSessions(ctx context.Context) {
	c.typingMu.Lock()
	sessions := c.typingSessions
	c.typingSessions = make(map[string]*typingSession)
	c.typingMu.Unlock()

	stopCtx := ctx
	if stopCtx == nil {
		stopCtx = context.Background()
	}
	for roomID, session := range sessions {
		session.stop()
		_, _ = c.client.UserTyping(stopCtx, id.RoomID(roomID), false, 0)
	}
}
