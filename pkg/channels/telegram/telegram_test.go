package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
)

const testToken = "1234567890:aaaabbbbaaaabbbbaaaabbbbaaaabbbbccc"

// stubCaller implements ta.Caller for testing.
type stubCaller struct {
	calls  []stubCall
	callFn func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error)
}

type stubCall struct {
	URL  string
	Data *ta.RequestData
}

func (s *stubCaller) Call(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
	s.calls = append(s.calls, stubCall{URL: url, Data: data})
	return s.callFn(ctx, url, data)
}

// stubConstructor implements ta.RequestConstructor for testing.
type stubConstructor struct{}

func (s *stubConstructor) JSONRequest(parameters any) (*ta.RequestData, error) {
	return &ta.RequestData{}, nil
}

func (s *stubConstructor) MultipartRequest(
	parameters map[string]string,
	files map[string]ta.NamedReader,
) (*ta.RequestData, error) {
	return &ta.RequestData{}, nil
}

// successResponse returns a ta.Response that telego will treat as a successful SendMessage.
func successResponse(t *testing.T) *ta.Response {
	t.Helper()
	msg := &telego.Message{MessageID: 1}
	b, err := json.Marshal(msg)
	require.NoError(t, err)
	return &ta.Response{Ok: true, Result: b}
}

// newTestChannel creates a TelegramChannel with a mocked bot for unit testing.
func newTestChannel(t *testing.T, caller *stubCaller) *TelegramChannel {
	t.Helper()

	bot, err := telego.NewBot(testToken,
		telego.WithAPICaller(caller),
		telego.WithRequestConstructor(&stubConstructor{}),
		telego.WithDiscardLogger(),
	)
	require.NoError(t, err)

	base := channels.NewBaseChannel("telegram", nil, nil, nil,
		channels.WithMaxMessageLength(4000),
	)
	base.SetRunning(true)

	return &TelegramChannel{
		BaseChannel: base,
		bot:         bot,
		chatIDs:     make(map[string]int64),
	}
}

func TestSend_EmptyContent(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			t.Fatal("SendMessage should not be called for empty content")
			return nil, nil
		},
	}
	ch := newTestChannel(t, caller)

	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "12345",
		Content: "",
	})

	assert.NoError(t, err)
	assert.Empty(t, caller.calls, "no API calls should be made for empty content")
}

func TestSend_ShortMessage_SingleCall(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "12345",
		Content: "Hello, world!",
	})

	assert.NoError(t, err)
	assert.Len(t, caller.calls, 1, "short message should result in exactly one SendMessage call")
}

func TestSend_LongMessage_SingleCall(t *testing.T) {
	// With WithMaxMessageLength(4000), the Manager pre-splits messages before
	// they reach Send(). A message at exactly 4000 chars should go through
	// as a single SendMessage call (no re-split needed since HTML expansion
	// won't exceed 4096 for plain text).
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	longContent := strings.Repeat("a", 4000)

	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "12345",
		Content: longContent,
	})

	assert.NoError(t, err)
	assert.Len(t, caller.calls, 1, "pre-split message within limit should result in one SendMessage call")
}

func TestSend_HTMLFallback_PerChunk(t *testing.T) {
	callCount := 0
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			callCount++
			// Fail on odd calls (HTML attempt), succeed on even calls (plain text fallback)
			if callCount%2 == 1 {
				return nil, errors.New("Bad Request: can't parse entities")
			}
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "12345",
		Content: "Hello **world**",
	})

	assert.NoError(t, err)
	// One short message → 1 HTML attempt (fail) + 1 plain text fallback (success) = 2 calls
	assert.Equal(t, 2, len(caller.calls), "should have HTML attempt + plain text fallback")
}

func TestSend_HTMLFallback_BothFail(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return nil, errors.New("send failed")
		},
	}
	ch := newTestChannel(t, caller)

	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "12345",
		Content: "Hello",
	})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, channels.ErrTemporary), "error should wrap ErrTemporary")
	assert.Equal(t, 2, len(caller.calls), "should have HTML attempt + plain text attempt")
}

func TestSend_LongMessage_HTMLFallback_StopsOnError(t *testing.T) {
	// With a long message that gets split into 2 chunks, if both HTML and
	// plain text fail on the first chunk, Send should return early.
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return nil, errors.New("send failed")
		},
	}
	ch := newTestChannel(t, caller)

	longContent := strings.Repeat("x", 4001)

	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "12345",
		Content: longContent,
	})

	assert.Error(t, err)
	// Should fail on the first chunk (2 calls: HTML + fallback), never reaching the second chunk.
	assert.Equal(t, 2, len(caller.calls), "should stop after first chunk fails both HTML and plain text")
}

func TestSend_MarkdownShortButHTMLLong_MultipleCalls(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	// Create markdown whose length is <= 4000 but whose HTML expansion is much longer.
	// "**a** " (6 chars) becomes "<b>a</b> " (9 chars) in HTML, so repeating it many times
	// yields HTML that exceeds Telegram's limit while markdown stays within it.
	markdownContent := strings.Repeat("**a** ", 600) // 3600 chars markdown, HTML ~5400+ chars
	assert.LessOrEqual(t, len([]rune(markdownContent)), 4000, "markdown content must not exceed chunk size")

	htmlExpanded := markdownToTelegramHTML(markdownContent)
	assert.Greater(
		t, len([]rune(htmlExpanded)), 4096,
		"HTML expansion must exceed Telegram limit for this test to be meaningful",
	)

	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "12345",
		Content: markdownContent,
	})

	assert.NoError(t, err)
	assert.Greater(
		t, len(caller.calls), 1,
		"markdown-short but HTML-long message should be split into multiple SendMessage calls",
	)
}

func TestSend_NotRunning(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
	}
	ch := newTestChannel(t, caller)
	ch.SetRunning(false)

	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "12345",
		Content: "Hello",
	})

	assert.ErrorIs(t, err, channels.ErrNotRunning)
	assert.Empty(t, caller.calls)
}

func TestSend_InvalidChatID(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
	}
	ch := newTestChannel(t, caller)

	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "not-a-number",
		Content: "Hello",
	})

	assert.Error(t, err)
	assert.True(t, errors.Is(err, channels.ErrSendFailed), "error should wrap ErrSendFailed")
	assert.Empty(t, caller.calls)
}

func TestParseTelegramChatID_Plain(t *testing.T) {
	cid, tid, err := parseTelegramChatID("12345")
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), cid)
	assert.Equal(t, 0, tid)
}

func TestParseTelegramChatID_NegativeGroup(t *testing.T) {
	cid, tid, err := parseTelegramChatID("-1001234567890")
	assert.NoError(t, err)
	assert.Equal(t, int64(-1001234567890), cid)
	assert.Equal(t, 0, tid)
}

func TestParseTelegramChatID_WithThreadID(t *testing.T) {
	cid, tid, err := parseTelegramChatID("-1001234567890/42")
	assert.NoError(t, err)
	assert.Equal(t, int64(-1001234567890), cid)
	assert.Equal(t, 42, tid)
}

func TestParseTelegramChatID_GeneralTopic(t *testing.T) {
	cid, tid, err := parseTelegramChatID("-100123/1")
	assert.NoError(t, err)
	assert.Equal(t, int64(-100123), cid)
	assert.Equal(t, 1, tid)
}

func TestParseTelegramChatID_Invalid(t *testing.T) {
	_, _, err := parseTelegramChatID("not-a-number")
	assert.Error(t, err)
}

func TestParseTelegramChatID_InvalidThreadID(t *testing.T) {
	_, _, err := parseTelegramChatID("-100123/not-a-thread")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid thread ID")
}

func TestSend_WithForumThreadID(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "-1001234567890/42",
		Content: "Hello from topic",
	})

	assert.NoError(t, err)
	assert.Len(t, caller.calls, 1)
}

func TestHandleMessage_ForumTopic_SetsMetadata(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
	}

	msg := &telego.Message{
		Text:            "hello from topic",
		MessageID:       10,
		MessageThreadID: 42,
		Chat: telego.Chat{
			ID:      -1001234567890,
			Type:    "supergroup",
			IsForum: true,
		},
		From: &telego.User{
			ID:        7,
			FirstName: "Alice",
		},
	}

	err := ch.handleMessage(context.Background(), msg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	inbound, ok := messageBus.ConsumeInbound(ctx)
	require.True(t, ok, "expected inbound message")

	// Composite chatID should include thread ID
	assert.Equal(t, "-1001234567890/42", inbound.ChatID)

	// Peer ID should include thread ID for session key isolation
	assert.Equal(t, "group", inbound.Peer.Kind)
	assert.Equal(t, "-1001234567890/42", inbound.Peer.ID)

	// Parent peer metadata should be set for agent binding
	assert.Equal(t, "topic", inbound.Metadata["parent_peer_kind"])
	assert.Equal(t, "42", inbound.Metadata["parent_peer_id"])
}

func TestHandleMessage_NoForum_NoThreadMetadata(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
	}

	msg := &telego.Message{
		Text:      "regular group message",
		MessageID: 11,
		Chat: telego.Chat{
			ID:   -100999,
			Type: "group",
		},
		From: &telego.User{
			ID:        8,
			FirstName: "Bob",
		},
	}

	err := ch.handleMessage(context.Background(), msg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	inbound, ok := messageBus.ConsumeInbound(ctx)
	require.True(t, ok)

	// Plain chatID without thread suffix
	assert.Equal(t, "-100999", inbound.ChatID)

	// Peer ID should be raw chat ID (no thread suffix)
	assert.Equal(t, "group", inbound.Peer.Kind)
	assert.Equal(t, "-100999", inbound.Peer.ID)

	// No parent peer metadata
	assert.Empty(t, inbound.Metadata["parent_peer_kind"])
	assert.Empty(t, inbound.Metadata["parent_peer_id"])
}
