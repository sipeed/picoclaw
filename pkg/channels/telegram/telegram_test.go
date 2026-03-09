package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

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
	return successResponseWithID(t, 1)
}

func successResponseWithID(t *testing.T, id int) *ta.Response {
	t.Helper()
	msg := &telego.Message{MessageID: id}
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

func TestSend_Wrapper(t *testing.T) {
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
	assert.Len(t, caller.calls, 1, "wrapper should call inner function")
}

func TestSendMessageWithID_EmptyContent(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			t.Fatal("SendMessage should not be called for empty content")
			return nil, nil
		},
	}
	ch := newTestChannel(t, caller)

	msgID, err := ch.SendMessageWithID(context.Background(), bus.OutboundMessage{ChatID: "12345", Content: ""})

	assert.NoError(t, err)
	assert.Empty(t, msgID)
	assert.Empty(t, caller.calls, "no API calls should be made for empty content")
}

func TestSendMessageWithID_ShortMessage_SingleCall(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	msgID, err := ch.SendMessageWithID(
		context.Background(),
		bus.OutboundMessage{ChatID: "12345", Content: "Hello, world!"},
	)

	assert.NoError(t, err)
	assert.Equal(t, "1", msgID)
	assert.Len(t, caller.calls, 1, "short message should result in exactly one SendMessage call")
}

func TestSendMessageWithID_LongMessage_SingleCall(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	longContent := strings.Repeat("a", 4000)

	msgID, err := ch.SendMessageWithID(context.Background(), bus.OutboundMessage{ChatID: "12345", Content: longContent})

	assert.NoError(t, err)
	assert.Equal(t, "1", msgID)
	assert.Len(t, caller.calls, 1, "pre-split message within limit should result in one SendMessage call")
}

func TestSendMessageWithID_HTMLFallback_PerChunk(t *testing.T) {
	callCount := 0
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			callCount++
			if callCount%2 == 1 {
				return nil, errors.New("Bad Request: can't parse entities")
			}
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	msgID, err := ch.SendMessageWithID(
		context.Background(),
		bus.OutboundMessage{ChatID: "12345", Content: "Hello **world**"},
	)

	assert.NoError(t, err)
	assert.Equal(t, "1", msgID)
	assert.Equal(t, 2, len(caller.calls), "should have HTML attempt + plain text fallback")
}

func TestSendMessageWithID_HTMLFallback_BothFail(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return nil, errors.New("send failed")
		},
	}
	ch := newTestChannel(t, caller)

	msgID, err := ch.SendMessageWithID(context.Background(), bus.OutboundMessage{ChatID: "12345", Content: "Hello"})

	assert.Error(t, err)
	assert.Empty(t, msgID)
	assert.True(t, errors.Is(err, channels.ErrTemporary), "error should wrap ErrTemporary")
	assert.Equal(t, 2, len(caller.calls), "should have HTML attempt + plain text attempt")
}

func TestSendMessageWithID_LongMessage_HTMLFallback_StopsOnError(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return nil, errors.New("send failed")
		},
	}
	ch := newTestChannel(t, caller)

	longContent := strings.Repeat("x", 4001)

	msgID, err := ch.SendMessageWithID(context.Background(), bus.OutboundMessage{ChatID: "12345", Content: longContent})

	assert.Error(t, err)
	assert.Empty(t, msgID)
	assert.Equal(t, 2, len(caller.calls), "should stop after first chunk fails both HTML and plain text")
}

func TestSendMessageWithID_MarkdownShortButHTMLLong_MultipleCalls(t *testing.T) {
	callCount := 0
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			callCount++
			return successResponseWithID(t, callCount), nil
		},
	}
	ch := newTestChannel(t, caller)

	markdownContent := strings.Repeat("**a** ", 600)
	assert.LessOrEqual(t, len([]rune(markdownContent)), 4000)

	msgID, err := ch.SendMessageWithID(
		context.Background(),
		bus.OutboundMessage{ChatID: "12345", Content: markdownContent},
	)

	assert.NoError(t, err)
	assert.Greater(
		t,
		len(caller.calls),
		1,
		"markdown-short but HTML-long message should be split into multiple SendMessage calls",
	)
	assert.Equal(t, "1,2", msgID)
}

func TestEditMessage_MultipleChunkIDs(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	content := strings.Repeat("**a** ", 600)

	err := ch.EditMessage(context.Background(), "12345", "1,2", content)

	assert.NoError(t, err)
	assert.Len(t, caller.calls, 2, "multi-part edit should update every tracked message")
}

func TestSendMessageWithID_NotRunning(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
	}
	ch := newTestChannel(t, caller)
	ch.SetRunning(false)

	msgID, err := ch.SendMessageWithID(context.Background(), bus.OutboundMessage{ChatID: "12345", Content: "Hello"})

	assert.ErrorIs(t, err, channels.ErrNotRunning)
	assert.Empty(t, msgID)
	assert.Empty(t, caller.calls)
}

func TestSendMessageWithID_InvalidChatID(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
	}
	ch := newTestChannel(t, caller)

	msgID, err := ch.SendMessageWithID(
		context.Background(),
		bus.OutboundMessage{ChatID: "not-a-number", Content: "Hello"},
	)

	assert.Error(t, err)
	assert.Empty(t, msgID)
	assert.True(t, errors.Is(err, channels.ErrSendFailed), "error should wrap ErrSendFailed")
	assert.Empty(t, caller.calls)
}
