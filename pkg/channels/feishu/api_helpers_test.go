package feishu

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestParseMessageContentPayload(t *testing.T) {
	payload := map[string]any{
		"items": []any{
			map[string]any{
				"message_id":  "om_123",
				"chat_id":     "oc_456",
				"msg_type":    "text",
				"create_time": "1710000000",
				"sender":      map[string]any{"id": "ou_1"},
				"body":        map[string]any{"content": `{"text":"hello"}`},
			},
		},
	}
	parsed := parseMessageContentPayload(payload)
	if parsed == nil {
		t.Fatal("expected parsed payload")
	}
	if parsed.MessageID != "om_123" || parsed.ChatID != "oc_456" || parsed.MsgType != "text" {
		t.Fatalf("unexpected parsed message: %+v", parsed)
	}
	content, ok := parsed.Content.(map[string]any)
	if !ok || content["text"] != "hello" {
		t.Fatalf("unexpected parsed content: %#v", parsed.Content)
	}
}

func TestParseCardSummary(t *testing.T) {
	card := map[string]any{
		"header": map[string]any{"title": map[string]any{"content": "test title"}},
		"elements": []any{
			map[string]any{"tag": "div", "text": map[string]any{"content": "body text"}},
			map[string]any{"tag": "img", "image_key": "img_xxx"},
			map[string]any{"tag": "action", "actions": []any{
				map[string]any{"text": map[string]any{"content": "Confirm"}, "type": "primary"},
			}},
		},
	}
	summary := parseCardSummary(card)
	if summary == nil {
		t.Fatal("expected card summary")
	}
	if summary.Title != "test title" {
		t.Fatalf("unexpected title: %q", summary.Title)
	}
	if len(summary.ImageKeys) != 1 || summary.ImageKeys[0] != "img_xxx" {
		t.Fatalf("unexpected image keys: %#v", summary.ImageKeys)
	}
	if len(summary.TextContents) == 0 || summary.TextContents[0] != "body text" {
		t.Fatalf("unexpected text contents: %#v", summary.TextContents)
	}
	if len(summary.ActionButtons) != 1 || summary.ActionButtons[0] != "Confirm" {
		t.Fatalf("unexpected action buttons: %#v", summary.ActionButtons)
	}
}

func TestExtractShareLinkToken(t *testing.T) {
	link := "https://applink.feishu.cn/client/message/link/open?token=om_abc%3D%3D&foo=bar"
	token := extractShareLinkToken(link)
	if token != "om_abc==" {
		t.Fatalf("unexpected token: %q", token)
	}
}

func TestNormalizeMessageDetailInteractive(t *testing.T) {
	item := map[string]any{
		"message_id": "om_123",
		"chat_id":    "oc_123",
		"msg_type":   "interactive",
		"body": map[string]any{
			"content": `{"header":{"title":{"content":"hello"}},"elements":[{"tag":"div","text":{"content":"world"}}]}`,
		},
	}
	detail := normalizeMessageDetail(item)
	if detail.Parsed == nil {
		t.Fatal("expected parsed detail")
	}
	if detail.CardParsed == nil || detail.CardParsed.Title != "hello" {
		t.Fatalf("unexpected card parse: %#v", detail.CardParsed)
	}
}

func TestNormalizeUserSummary(t *testing.T) {
	user := normalizeUserSummary(map[string]any{"open_id": "ou_1", "name": "Alice", "email": "a@example.com"})
	if user.ID != "ou_1" || user.Name != "Alice" || user.Email != "a@example.com" {
		t.Fatalf("unexpected user summary: %+v", user)
	}
}

func TestNormalizeChatSummary(t *testing.T) {
	chat := normalizeChatSummary(map[string]any{"chat_id": "oc_1", "name": "Group", "description": "desc"})
	if chat.ChatID != "oc_1" || chat.Name != "Group" || chat.Description != "desc" {
		t.Fatalf("unexpected chat summary: %+v", chat)
	}
}

func TestFilenameFromHeader(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Disposition", `attachment; filename="demo.txt"`)
	if got := filenameFromHeader(h); got != "demo.txt" {
		t.Fatalf("unexpected filename: %s", got)
	}
}

func TestFilenameFromHeaderEncoded(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Disposition", "attachment; filename*=UTF-8''hello%20world.txt")
	if got := filenameFromHeader(h); got != "hello world.txt" {
		t.Fatalf("unexpected encoded filename: %s", got)
	}
}

func TestNormalizeDriveFile(t *testing.T) {
	got := normalizeDriveFile(map[string]any{
		"file_token":   "file_token",
		"name":         "demo.txt",
		"type":         "file",
		"parent_token": "folder_token",
		"size":         float64(12),
	})
	if got.FileToken != "file_token" || got.Name != "demo.txt" || got.Size != 12 {
		t.Fatalf("unexpected normalized file: %+v", got)
	}
}

func TestBuildMultipartChunkBody(t *testing.T) {
	body, contentType, err := buildMultipartChunkBody(2, []byte("hello"))
	if err != nil {
		t.Fatalf("build body failed: %v", err)
	}
	if !strings.Contains(contentType, "multipart/form-data") {
		t.Fatalf("unexpected content type: %s", contentType)
	}
	b, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `name="seq"`) || !strings.Contains(s, "2") {
		t.Fatalf("unexpected multipart body: %s", s)
	}
}

func TestBuildDriveUploadBody(t *testing.T) {
	body, contentType, err := buildDriveUploadBody("fld_123", "report.txt", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("build upload body failed: %v", err)
	}
	if !strings.Contains(contentType, "multipart/form-data") {
		t.Fatalf("unexpected content type: %s", contentType)
	}
	b, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	s := string(b)
	for _, expected := range []string{"fld_123", "report.txt", "hello", `name="file"`} {
		if !strings.Contains(s, expected) {
			t.Fatalf("multipart body missing %q: %s", expected, s)
		}
	}
}
