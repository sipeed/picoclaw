package qqc2c

import (
	"context"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// streamSession implements bus.Streamer on top of
// POST /v2/users/{openid}/stream_messages.
//
// Platform rules honoured here:
//   - the first chunk must not carry stream_msg_id; the server returns it
//   - subsequent chunks carry stream_msg_id and an incrementing index
//   - input_state=1 while generating, 10 on the final chunk
//   - 50 QPS limit, so increments are merged (default every 300ms)
type streamSession struct {
	api    *apiClient
	openID string
	msgID  string
	msgSeq int

	interval time.Duration

	mu          sync.Mutex
	streamMsgID string
	index       int
	lastSent    time.Time
	last        string
	closed      bool
}

func (s *streamSession) send(ctx context.Context, content string, state int) error {
	req := &StreamRequest{
		InputMode:   "replace",
		InputState:  state,
		Index:       s.index,
		ContentType: "markdown",
		ContentRaw:  content,
		MsgID:       s.msgID,
		MsgSeq:      s.msgSeq,
		StreamMsgID: s.streamMsgID,
	}
	sent, err := s.api.StreamC2C(ctx, s.openID, req)
	if err != nil {
		return err
	}
	if sent != nil && sent.ID != "" {
		s.streamMsgID = sent.ID
	}
	s.index++
	s.lastSent = time.Now()
	s.last = content
	return nil
}

// Update sends a throttled incremental chunk.
func (s *streamSession) Update(ctx context.Context, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || content == "" || content == s.last {
		return nil
	}
	if time.Since(s.lastSent) < s.interval {
		return nil
	}
	if err := s.send(ctx, content, 1); err != nil {
		logger.DebugCF("qq_c2c", "stream update failed", map[string]any{"error": err.Error()})
		return err
	}
	return nil
}

// Finalize always pushes the complete text with input_state=10.
func (s *streamSession) Finalize(ctx context.Context, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if content == "" {
		content = s.last
	}
	return s.send(ctx, content, 10)
}

// Cancel closes the stream without further output.
func (s *streamSession) Cancel(_ context.Context) {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
}
