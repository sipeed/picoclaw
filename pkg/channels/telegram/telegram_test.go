package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
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
	body, err := json.Marshal(parameters)
	if err != nil {
		return nil, err
	}
	return &ta.RequestData{
		ContentType: ta.ContentTypeJSON,
		BodyRaw:     body,
	}, nil
}

func (s *stubConstructor) MultipartRequest(
	parameters map[string]string,
	files map[string]ta.NamedReader,
) (*ta.RequestData, error) {
	body, err := json.Marshal(parameters)
	if err != nil {
		return nil, err
	}
	return &ta.RequestData{
		ContentType: ta.ContentTypeJSON,
		BodyRaw:     body,
	}, nil
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

func successBoolResponse(t *testing.T) *ta.Response {
	t.Helper()
	return &ta.Response{Ok: true, Result: []byte("true")}
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

func decodeCallBody(t *testing.T, call stubCall) map[string]any {
	t.Helper()

	var body map[string]any
	require.NoError(t, json.Unmarshal(call.Data.BodyRaw, &body))
	return body
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

	msgID, err := ch.SendMessageWithID(context.Background(), bus.OutboundMessage{
		ChatID:  "12345",
		Content: "Hello, world!",
	})

	assert.NoError(t, err)
	assert.Equal(t, "1", msgID)
	assert.Len(t, caller.calls, 1, "short message should result in exactly one SendMessage call")
}

func TestSendMessageWithID_ForumTopic_UsesThreadID(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	msgID, err := ch.SendMessageWithID(context.Background(), bus.OutboundMessage{
		ChatID:  "-1001234567890:topic:42",
		Content: "Hello, topic!",
	})

	assert.NoError(t, err)
	assert.Equal(t, "1", msgID)
	require.Len(t, caller.calls, 1)
	body := decodeCallBody(t, caller.calls[0])
	assert.Equal(t, float64(42), body["message_thread_id"])
}

func TestSendMessageWithID_ReplyToMessage_UsesReplyParameters(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	msgID, err := ch.SendMessageWithID(context.Background(), bus.OutboundMessage{
		ChatID:           "12345",
		Content:          "Hello, thread!",
		ReplyToMessageID: "99",
	})

	assert.NoError(t, err)
	assert.Equal(t, "1", msgID)
	require.Len(t, caller.calls, 1)
	body := decodeCallBody(t, caller.calls[0])
	replyParams, ok := body["reply_parameters"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(99), replyParams["message_id"])
	assert.Equal(t, true, replyParams["allow_sending_without_reply"])
}

func TestSendMessageWithID_GeneralTopic_OmitsThreadID(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	msgID, err := ch.SendMessageWithID(context.Background(), bus.OutboundMessage{
		ChatID:  "-1001234567890:topic:1",
		Content: "Hello, general!",
	})

	assert.NoError(t, err)
	assert.Equal(t, "1", msgID)
	require.Len(t, caller.calls, 1)
	body := decodeCallBody(t, caller.calls[0])
	_, hasThreadID := body["message_thread_id"]
	assert.False(t, hasThreadID)
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

func TestSetMessageReaction_SendsConfiguredEmoji(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successBoolResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	err := ch.SetMessageReaction(context.Background(), "12345", "99", "❤️")

	require.NoError(t, err)
	require.Len(t, caller.calls, 1)
	body := decodeCallBody(t, caller.calls[0])
	assert.Equal(t, float64(99), body["message_id"])
	reaction, ok := body["reaction"].([]any)
	require.True(t, ok)
	require.Len(t, reaction, 1)
	first, ok := reaction[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "emoji", first["type"])
	assert.Equal(t, "❤️", first["emoji"])
}

func TestStartTyping_ForumTopic_UsesThreadID(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	stop, err := ch.StartTyping(context.Background(), "-1001234567890:topic:42")
	require.NoError(t, err)
	stop()

	require.NotEmpty(t, caller.calls)
	body := decodeCallBody(t, caller.calls[0])
	assert.Equal(t, float64(42), body["message_thread_id"])
}

func TestStartTyping_GeneralTopic_KeepsThreadID(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	stop, err := ch.StartTyping(context.Background(), "-1001234567890:topic:1")
	require.NoError(t, err)
	stop()

	require.NotEmpty(t, caller.calls)
	body := decodeCallBody(t, caller.calls[0])
	assert.Equal(t, float64(1), body["message_thread_id"])
}

func TestSendPlaceholder_GroupSkipsPlaceholder(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)
	ch.config = config.DefaultConfig()
	ch.config.Channels.Telegram.Placeholder.Enabled = true
	ch.config.Channels.Telegram.Placeholder.Text = "Thinking"

	msgID, err := ch.SendPlaceholder(context.Background(), "-1001234567890:topic:42")
	require.NoError(t, err)
	assert.Empty(t, msgID)
	assert.Empty(t, caller.calls)
}

func TestSendPlaceholder_PrivateChatSendsMessage(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)
	ch.config = config.DefaultConfig()
	ch.config.Channels.Telegram.Placeholder.Enabled = true
	ch.config.Channels.Telegram.Placeholder.Text = "Thinking"

	msgID, err := ch.SendPlaceholder(context.Background(), "12345")
	require.NoError(t, err)
	assert.Equal(t, "1", msgID)
	require.Len(t, caller.calls, 1)
	body := decodeCallBody(t, caller.calls[0])
	assert.Equal(t, "Thinking", body["text"])
}

func TestSendMedia_ForumTopic_UsesThreadID(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)

	tmpFile, err := os.CreateTemp(t.TempDir(), "telegram-media-*.jpg")
	require.NoError(t, err)
	_, err = tmpFile.WriteString("hello")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	ref, err := store.Store(tmpFile.Name(), media.MediaMeta{Filename: "photo.jpg"}, "test-scope")
	require.NoError(t, err)

	err = ch.SendMedia(context.Background(), bus.OutboundMediaMessage{
		ChatID: "-1001234567890:topic:42",
		Parts: []bus.MediaPart{{
			Type: "image",
			Ref:  ref,
		}},
	})
	require.NoError(t, err)
	require.Len(t, caller.calls, 1)
	body := decodeCallBody(t, caller.calls[0])
	assert.Equal(t, "42", fmt.Sprint(body["message_thread_id"]))
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

func TestParseTelegramTarget(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantChatID int64
		wantThread int
		wantErr    bool
	}{
		{name: "base chat", input: "12345", wantChatID: 12345},
		{name: "forum topic", input: "-100123:topic:42", wantChatID: -100123, wantThread: 42},
		{name: "invalid topic", input: "-100123:topic:abc", wantErr: true},
		{name: "invalid chat", input: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, err := parseTelegramTarget(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantChatID, target.ChatID)
			assert.Equal(t, tt.wantThread, target.MessageThreadID)
		})
	}
}

func TestResolveTelegramForumThreadID_GeneralTopicDefaultsToOne(t *testing.T) {
	threadID, ok := resolveTelegramForumThreadID(true, 0)
	require.True(t, ok)
	assert.Equal(t, telegramGeneralTopicID, threadID)
}
