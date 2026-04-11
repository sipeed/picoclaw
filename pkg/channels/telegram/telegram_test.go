package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
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

type multipartCall struct {
	Parameters map[string]string
	FileSizes  map[string]int
}

func (s *stubConstructor) JSONRequest(parameters any) (*ta.RequestData, error) {
	b, err := json.Marshal(parameters)
	if err != nil {
		return nil, err
	}
	return &ta.RequestData{
		ContentType: "application/json",
		BodyRaw:     b,
	}, nil
}

func (s *stubConstructor) MultipartRequest(
	parameters map[string]string,
	files map[string]ta.NamedReader,
) (*ta.RequestData, error) {
	return &ta.RequestData{}, nil
}

type multipartRecordingConstructor struct {
	stubConstructor
	calls []multipartCall
}

func (s *multipartRecordingConstructor) MultipartRequest(
	parameters map[string]string,
	files map[string]ta.NamedReader,
) (*ta.RequestData, error) {
	call := multipartCall{
		Parameters: make(map[string]string, len(parameters)),
		FileSizes:  make(map[string]int, len(files)),
	}
	for k, v := range parameters {
		call.Parameters[k] = v
	}
	for field, file := range files {
		if file == nil {
			continue
		}
		data, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}
		call.FileSizes[field] = len(data)
	}
	s.calls = append(s.calls, call)
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

func successUserResponse(t *testing.T, user *telego.User) *ta.Response {
	t.Helper()
	b, err := json.Marshal(user)
	require.NoError(t, err)
	return &ta.Response{Ok: true, Result: b}
}

// newTestChannel creates a TelegramChannel with a mocked bot for unit testing.
func newTestChannel(t *testing.T, caller *stubCaller) *TelegramChannel {
	return newTestChannelWithConstructor(t, caller, &stubConstructor{})
}

func newTestChannelWithConstructor(
	t *testing.T,
	caller *stubCaller,
	constructor ta.RequestConstructor,
) *TelegramChannel {
	t.Helper()

	bot, err := telego.NewBot(testToken,
		telego.WithAPICaller(caller),
		telego.WithRequestConstructor(constructor),
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
		bc:          &config.Channel{Type: config.ChannelTelegram, Enabled: true},
		tgCfg:       &config.TelegramSettings{},
	}
}

func TestSendMedia_ImageFallbacksToDocumentOnInvalidDimensions(t *testing.T) {
	constructor := &multipartRecordingConstructor{}
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			switch {
			case strings.Contains(url, "sendPhoto"):
				return nil, errors.New(`api: 400 "Bad Request: PHOTO_INVALID_DIMENSIONS"`)
			case strings.Contains(url, "sendDocument"):
				return successResponse(t), nil
			default:
				t.Fatalf("unexpected API call: %s", url)
				return nil, nil
			}
		},
	}
	ch := newTestChannelWithConstructor(t, caller, constructor)

	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "woodstock-en-10s.png")
	content := []byte("fake-png-content")
	require.NoError(t, os.WriteFile(localPath, content, 0o644))

	ref, err := store.Store(
		localPath,
		media.MediaMeta{Filename: "woodstock-en-10s.png", ContentType: "image/png"},
		"scope-1",
	)
	require.NoError(t, err)

	_, err = ch.SendMedia(context.Background(), bus.OutboundMediaMessage{
		ChatID: "12345",
		Parts: []bus.MediaPart{{
			Type:    "image",
			Ref:     ref,
			Caption: "caption",
		}},
	})

	require.NoError(t, err)
	require.Len(t, caller.calls, 2)
	assert.Contains(t, caller.calls[0].URL, "sendPhoto")
	assert.Contains(t, caller.calls[1].URL, "sendDocument")
	require.Len(t, constructor.calls, 2)
	assert.Equal(t, len(content), constructor.calls[0].FileSizes["photo"])
	assert.Equal(t, len(content), constructor.calls[1].FileSizes["document"])
	assert.Equal(t, "caption", constructor.calls[1].Parameters["caption"])
}

func TestSendMedia_ImageNonDimensionErrorDoesNotFallback(t *testing.T) {
	constructor := &multipartRecordingConstructor{}
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return nil, errors.New("api: 500 \"server exploded\"")
		},
	}
	ch := newTestChannelWithConstructor(t, caller, constructor)

	store := media.NewFileMediaStore()
	ch.SetMediaStore(store)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "image.png")
	require.NoError(t, os.WriteFile(localPath, []byte("fake-png-content"), 0o644))

	ref, err := store.Store(localPath, media.MediaMeta{Filename: "image.png", ContentType: "image/png"}, "scope-1")
	require.NoError(t, err)

	_, err = ch.SendMedia(context.Background(), bus.OutboundMediaMessage{
		ChatID: "12345",
		Parts: []bus.MediaPart{{
			Type: "image",
			Ref:  ref,
		}},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, channels.ErrTemporary)
	require.Len(t, caller.calls, 1)
	assert.Contains(t, caller.calls[0].URL, "sendPhoto")
	require.Len(t, constructor.calls, 1)
	assert.NotContains(t, caller.calls[0].URL, "sendDocument")
}

func TestSend_EmptyContent(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			t.Fatal("SendMessage should not be called for empty content")
			return nil, nil
		},
	}
	ch := newTestChannel(t, caller)

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
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

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
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

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
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

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
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

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
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

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
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

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "12345",
		Content: markdownContent,
	})

	assert.NoError(t, err)
	assert.Greater(
		t, len(caller.calls), 1,
		"markdown-short but HTML-long message should be split into multiple SendMessage calls",
	)
}

func TestSend_HTMLOverflow_WordBoundary(t *testing.T) {
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			return successResponse(t), nil
		},
	}
	ch := newTestChannel(t, caller)

	// We want to force a split near index ~2600 while keeping markdown length <= 4000.
	// Prefix of 430 bold units (6 chars each) = 2580 chars.
	// Expansion per unit is +3 chars when converted to HTML, so 2580 + 430*3 = 3870.
	prefix := strings.Repeat("**a** ", 430)
	targetWord := "TARGETWORDTHATSTAYSTOGETHER"
	// Suffix of 230 bold units (6 chars each) = 1380 chars.
	// Total markdown length: 2580 (prefix) + 27 (target word) + 1380 (suffix) = 3987 <= 4000.
	// HTML expansion adds ~3 chars per bold unit: (430 + 230)*3 = 1980 extra chars,
	// so total HTML length comfortably exceeds 4096.
	suffix := strings.Repeat(" **b**", 230)
	content := prefix + targetWord + suffix

	// Ensure the test content matches the intended boundary conditions.
	assert.LessOrEqual(t, len([]rune(content)), 4000, "markdown content must not exceed chunk size for this test")

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
		ChatID:  "123456",
		Content: content,
	})

	assert.NoError(t, err)

	foundFullWord := false
	for i, call := range caller.calls {
		var params map[string]any
		err := json.Unmarshal(call.Data.BodyRaw, &params)
		require.NoError(t, err)
		text, _ := params["text"].(string)

		hasWord := strings.Contains(text, targetWord)
		t.Logf("Chunk %d length: %d, contains target word: %v", i, len(text), hasWord)

		if hasWord {
			foundFullWord = true
			break
		}
	}

	assert.True(t, foundFullWord, "The target word should not be split between chunks")
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

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
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

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
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

	_, err := ch.Send(context.Background(), bus.OutboundMessage{
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

	inbound, ok := <-messageBus.InboundChan()
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

	inbound, ok := <-messageBus.InboundChan()
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

func TestHandleMessage_ReplyThread_NonForum_NoIsolation(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
	}

	// In regular groups, reply threads set MessageThreadID to the original
	// message ID. This should NOT trigger per-thread session isolation.
	msg := &telego.Message{
		Text:            "reply in thread",
		MessageID:       20,
		MessageThreadID: 15,
		Chat: telego.Chat{
			ID:      -100999,
			Type:    "supergroup",
			IsForum: false,
		},
		From: &telego.User{
			ID:        9,
			FirstName: "Carol",
		},
	}

	err := ch.handleMessage(context.Background(), msg)
	require.NoError(t, err)

	inbound, ok := <-messageBus.InboundChan()
	require.True(t, ok)

	// chatID should NOT include thread suffix for non-forum groups
	assert.Equal(t, "-100999", inbound.ChatID)

	// Peer ID should be raw chat ID (shared session for whole group)
	assert.Equal(t, "group", inbound.Peer.Kind)
	assert.Equal(t, "-100999", inbound.Peer.ID)

	// No parent peer metadata
	assert.Empty(t, inbound.Metadata["parent_peer_kind"])
	assert.Empty(t, inbound.Metadata["parent_peer_id"])
}

func assertHandleMessageQuotedUserReply(
	t *testing.T,
	chatID int64,
	messageID int,
	userID int64,
	userName string,
	userText string,
	replyMessageID int,
	replyText string,
	replyCaption string,
	replyAuthorID int64,
	replyAuthorName string,
	expectedContent string,
) {
	t.Helper()

	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
	}

	msg := &telego.Message{
		Text:      userText,
		MessageID: messageID,
		Chat: telego.Chat{
			ID:   chatID,
			Type: "private",
		},
		From: &telego.User{
			ID:        userID,
			FirstName: userName,
		},
		ReplyToMessage: &telego.Message{
			MessageID: replyMessageID,
			Text:      replyText,
			Caption:   replyCaption,
			From: &telego.User{
				ID:        replyAuthorID,
				FirstName: replyAuthorName,
			},
		},
	}

	err := ch.handleMessage(context.Background(), msg)
	require.NoError(t, err)

	inbound, ok := <-messageBus.InboundChan()
	require.True(t, ok)
	assert.Equal(t, strconv.Itoa(replyMessageID), inbound.Metadata["reply_to_message_id"])
	assert.Equal(t, expectedContent, inbound.Content)
}

func TestHandleMessage_ReplyToMessage_PrependsQuotedTextAndMetadata(t *testing.T) {
	assertHandleMessageQuotedUserReply(
		t,
		456,
		21,
		11,
		"Alice",
		"follow up",
		99,
		"old context",
		"",
		12,
		"Bob",
		"[quoted user message from Bob]: old context\n\nfollow up",
	)
}

func TestHandleMessage_ReplyToMessage_UsesCaptionWhenQuotedTextMissing(t *testing.T) {
	assertHandleMessageQuotedUserReply(
		t,
		789,
		22,
		13,
		"Carol",
		"answer this",
		100,
		"",
		"caption context",
		14,
		"Dave",
		"[quoted user message from Dave]: caption context\n\nanswer this",
	)
}

func TestHandleMessage_ReplyToOwnBotMessage_UsesAssistantRole(t *testing.T) {
	messageBus := bus.NewMessageBus()
	caller := &stubCaller{
		callFn: func(ctx context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
			if strings.Contains(url, "getMe") {
				return successUserResponse(t, &telego.User{
					ID:        42,
					IsBot:     true,
					FirstName: "Pico",
					Username:  "afjcjsbx_picoclaw_bot",
				}), nil
			}
			t.Fatalf("unexpected API call: %s", url)
			return nil, nil
		},
	}
	ch := newTestChannel(t, caller)
	ch.BaseChannel = channels.NewBaseChannel("telegram", nil, messageBus, nil)
	ch.ctx = context.Background()

	msg := &telego.Message{
		Text:      "ti ricordi questo file?",
		MessageID: 23,
		Chat: telego.Chat{
			ID:   999,
			Type: "private",
		},
		From: &telego.User{
			ID:        15,
			FirstName: "Eve",
		},
		ReplyToMessage: &telego.Message{
			MessageID: 101,
			Text:      "Fatto! Ho creato il file notizie_2026_03_28.md",
			From: &telego.User{
				ID:        42,
				IsBot:     true,
				FirstName: "Pico",
				Username:  "afjcjsbx_picoclaw_bot",
			},
		},
	}

	err := ch.handleMessage(context.Background(), msg)
	require.NoError(t, err)

	inbound, ok := <-messageBus.InboundChan()
	require.True(t, ok)
	assert.Equal(t, "101", inbound.Metadata["reply_to_message_id"])
	assert.Equal(
		t,
		"[quoted assistant message from afjcjsbx_picoclaw_bot]: Fatto! Ho creato il file notizie_2026_03_28.md\n\nti ricordi questo file?",
		inbound.Content,
	)
}

func TestTelegramQuotedContent_IncludesVoiceMarkerAlongsideCaption(t *testing.T) {
	msg := &telego.Message{
		Caption: "listen to this",
		Voice: &telego.Voice{
			FileID: "voice-file",
		},
	}

	assert.Equal(t, "listen to this\n[voice]", telegramQuotedContent(msg))
}

func TestQuotedTelegramMediaRefs_ResolvesQuotedAudioInOrder(t *testing.T) {
	msg := &telego.Message{
		Voice: &telego.Voice{FileID: "voice-file"},
		Audio: &telego.Audio{FileID: "audio-file"},
	}

	var calls []string
	refs := quotedTelegramMediaRefs(msg, func(fileID, ext, filename string) string {
		calls = append(calls, fileID+"|"+ext+"|"+filename)
		return "ref://" + filename
	})

	assert.Equal(
		t,
		[]string{"voice-file|.ogg|voice.ogg", "audio-file|.mp3|audio.mp3"},
		calls,
	)
	assert.Equal(t, []string{"ref://voice.ogg", "ref://audio.mp3"}, refs)
}

func TestHandleMessage_EmptyContent_Ignored(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &TelegramChannel{
		BaseChannel: channels.NewBaseChannel("telegram", nil, messageBus, nil),
		chatIDs:     make(map[string]int64),
		ctx:         context.Background(),
	}

	// Service message with no text/caption/media (like ForumTopicCreated)
	msg := &telego.Message{
		MessageID: 123,
		Chat: telego.Chat{
			ID:   456,
			Type: "group",
		},
		From: &telego.User{
			ID:        789,
			FirstName: "User",
		},
	}

	err := ch.handleMessage(context.Background(), msg)
	require.NoError(t, err)

	// Should NOT publish to message bus
	select {
	case <-messageBus.InboundChan():
		t.Fatal("Empty message should not be published to message bus")
	default:
	}
}
