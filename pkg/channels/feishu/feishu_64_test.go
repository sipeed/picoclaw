//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package feishu

import (
	"context"
	"errors"
	"testing"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

type feishuRetryResp struct {
	code int
	msg  string
}

func TestExtractContent(t *testing.T) {
	tests := []struct {
		name        string
		messageType string
		rawContent  string
		want        string
	}{
		{
			name:        "text message",
			messageType: "text",
			rawContent:  `{"text": "hello world"}`,
			want:        "hello world",
		},
		{
			name:        "text message invalid JSON",
			messageType: "text",
			rawContent:  `not json`,
			want:        "not json",
		},
		{
			name:        "post message returns raw JSON",
			messageType: "post",
			rawContent:  `{"title": "test post"}`,
			want:        `{"title": "test post"}`,
		},
		{
			name:        "image message returns empty",
			messageType: "image",
			rawContent:  `{"image_key": "img_xxx"}`,
			want:        "",
		},
		{
			name:        "file message with filename",
			messageType: "file",
			rawContent:  `{"file_key": "file_xxx", "file_name": "report.pdf"}`,
			want:        "report.pdf",
		},
		{
			name:        "file message without filename",
			messageType: "file",
			rawContent:  `{"file_key": "file_xxx"}`,
			want:        "",
		},
		{
			name:        "audio message with filename",
			messageType: "audio",
			rawContent:  `{"file_key": "file_xxx", "file_name": "recording.ogg"}`,
			want:        "recording.ogg",
		},
		{
			name:        "media message with filename",
			messageType: "media",
			rawContent:  `{"file_key": "file_xxx", "file_name": "video.mp4"}`,
			want:        "video.mp4",
		},
		{
			name:        "unknown message type returns raw",
			messageType: "sticker",
			rawContent:  `{"sticker_id": "sticker_xxx"}`,
			want:        `{"sticker_id": "sticker_xxx"}`,
		},
		{
			name:        "empty raw content",
			messageType: "text",
			rawContent:  "",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractContent(tt.messageType, tt.rawContent)
			if got != tt.want {
				t.Errorf("extractContent(%q, %q) = %q, want %q", tt.messageType, tt.rawContent, got, tt.want)
			}
		})
	}
}

func TestAppendMediaTags(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		messageType string
		mediaRefs   []string
		want        string
	}{
		{
			name:        "no refs returns content unchanged",
			content:     "hello",
			messageType: "image",
			mediaRefs:   nil,
			want:        "hello",
		},
		{
			name:        "empty refs returns content unchanged",
			content:     "hello",
			messageType: "image",
			mediaRefs:   []string{},
			want:        "hello",
		},
		{
			name:        "image with content",
			content:     "check this",
			messageType: "image",
			mediaRefs:   []string{"ref1"},
			want:        "check this [image: photo]",
		},
		{
			name:        "image empty content",
			content:     "",
			messageType: "image",
			mediaRefs:   []string{"ref1"},
			want:        "[image: photo]",
		},
		{
			name:        "audio",
			content:     "listen",
			messageType: "audio",
			mediaRefs:   []string{"ref1"},
			want:        "listen [audio]",
		},
		{
			name:        "media/video",
			content:     "watch",
			messageType: "media",
			mediaRefs:   []string{"ref1"},
			want:        "watch [video]",
		},
		{
			name:        "file",
			content:     "report.pdf",
			messageType: "file",
			mediaRefs:   []string{"ref1"},
			want:        "report.pdf [file]",
		},
		{
			name:        "unknown type",
			content:     "something",
			messageType: "sticker",
			mediaRefs:   []string{"ref1"},
			want:        "something [attachment]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendMediaTags(tt.content, tt.messageType, tt.mediaRefs)
			if got != tt.want {
				t.Errorf(
					"appendMediaTags(%q, %q, %v) = %q, want %q",
					tt.content,
					tt.messageType,
					tt.mediaRefs,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestExtractFeishuSenderID(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name   string
		sender *larkim.EventSender
		want   string
	}{
		{
			name:   "nil sender",
			sender: nil,
			want:   "",
		},
		{
			name:   "nil sender ID",
			sender: &larkim.EventSender{SenderId: nil},
			want:   "",
		},
		{
			name: "userId preferred",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  strPtr("u_abc123"),
					OpenId:  strPtr("ou_def456"),
					UnionId: strPtr("on_ghi789"),
				},
			},
			want: "u_abc123",
		},
		{
			name: "openId fallback",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  strPtr(""),
					OpenId:  strPtr("ou_def456"),
					UnionId: strPtr("on_ghi789"),
				},
			},
			want: "ou_def456",
		},
		{
			name: "unionId fallback",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  strPtr(""),
					OpenId:  strPtr(""),
					UnionId: strPtr("on_ghi789"),
				},
			},
			want: "on_ghi789",
		},
		{
			name: "all empty strings",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  strPtr(""),
					OpenId:  strPtr(""),
					UnionId: strPtr(""),
				},
			},
			want: "",
		},
		{
			name: "nil userId pointer falls through",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  nil,
					OpenId:  strPtr("ou_def456"),
					UnionId: nil,
				},
			},
			want: "ou_def456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFeishuSenderID(tt.sender)
			if got != tt.want {
				t.Errorf("extractFeishuSenderID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWithRESTClientAuthRetry_RecreatesClientOnInvalidAuth(t *testing.T) {
	first := &lark.Client{}
	second := &lark.Client{}
	newClientCalls := 0

	ch := &FeishuChannel{
		client: first,
		newClient: func() *lark.Client {
			newClientCalls++
			return second
		},
	}

	attempts := 0
	resp, err := withRESTClientAuthRetry(
		ch,
		func(client *lark.Client) (feishuRetryResp, error) {
			attempts++
			switch attempts {
			case 1:
				if client != first {
					t.Fatalf("first attempt used %p, want %p", client, first)
				}
				return feishuRetryResp{
					code: feishuInvalidAccessTokenCode,
					msg:  "Invalid access token for authorization. Please make a request with token attached.",
				}, nil
			case 2:
				if client != second {
					t.Fatalf("second attempt used %p, want %p", client, second)
				}
				return feishuRetryResp{}, nil
			default:
				t.Fatalf("unexpected attempt %d", attempts)
				return feishuRetryResp{}, nil
			}
		},
		func(resp feishuRetryResp) (int, string, bool) {
			return resp.code, resp.msg, true
		},
	)
	if err != nil {
		t.Fatalf("withRESTClientAuthRetry() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	if newClientCalls != 1 {
		t.Fatalf("newClientCalls = %d, want 1", newClientCalls)
	}
	if ch.currentRESTClient() != second {
		t.Fatal("expected channel client to be replaced after auth retry")
	}
	if resp.code != 0 || resp.msg != "" {
		t.Fatalf("final response = %+v, want success response", resp)
	}
}

func TestWithRESTClientAuthRetry_DoesNotRetryNonAuthError(t *testing.T) {
	first := &lark.Client{}
	newClientCalls := 0
	expectedErr := errors.New("network timeout")

	ch := &FeishuChannel{
		client: first,
		newClient: func() *lark.Client {
			newClientCalls++
			return &lark.Client{}
		},
	}

	attempts := 0
	_, err := withRESTClientAuthRetry(
		ch,
		func(client *lark.Client) (feishuRetryResp, error) {
			attempts++
			if client != first {
				t.Fatalf("attempt used %p, want %p", client, first)
			}
			return feishuRetryResp{}, expectedErr
		},
		func(resp feishuRetryResp) (int, string, bool) {
			return resp.code, resp.msg, true
		},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("withRESTClientAuthRetry() error = %v, want %v", err, expectedErr)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
	if newClientCalls != 0 {
		t.Fatalf("newClientCalls = %d, want 0", newClientCalls)
	}
	if ch.currentRESTClient() != first {
		t.Fatal("expected channel client to remain unchanged for non-auth error")
	}
}

func TestShouldRetryFeishuAuth_ErrorMessageFallback(t *testing.T) {
	retry := shouldRetryFeishuAuth(
		feishuRetryResp{},
		context.DeadlineExceeded,
		func(resp feishuRetryResp) (int, string, bool) {
			return resp.code, resp.msg, false
		},
	)
	if retry {
		t.Fatal("shouldRetryFeishuAuth() retried unexpected non-auth error")
	}

	retry = shouldRetryFeishuAuth(
		feishuRetryResp{},
		errors.New("code=99991663 msg=Invalid access token for authorization"),
		func(resp feishuRetryResp) (int, string, bool) {
			return resp.code, resp.msg, false
		},
	)
	if !retry {
		t.Fatal("shouldRetryFeishuAuth() should retry invalid access token errors")
	}
}
