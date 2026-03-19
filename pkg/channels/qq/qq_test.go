package qq

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/openapi/options"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/media"
)

func TestHandleC2CMessage_IncludesAccountIDMetadata(t *testing.T) {
	messageBus := bus.NewMessageBus()
	ch := &QQChannel{
		BaseChannel: channels.NewBaseChannel("qq", nil, messageBus, nil),
		dedup:       make(map[string]time.Time),
		done:        make(chan struct{}),
		ctx:         context.Background(),
	}

	err := ch.handleC2CMessage()(nil, &dto.WSC2CMessageData{
		ID:      "msg-1",
		Content: "hello",
		Author: &dto.User{
			ID: "7750283E123456",
		},
	})
	if err != nil {
		t.Fatalf("handleC2CMessage() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("timeout waiting for inbound message")
			return
		case inbound, ok := <-messageBus.InboundChan():
			if !ok {
				t.Fatal("expected inbound message")
			}
			if inbound.Metadata["account_id"] != "7750283E123456" {
				t.Fatalf("account_id metadata = %q, want %q", inbound.Metadata["account_id"], "7750283E123456")
			}
			return
		}
	}
}

type fakeQQAPI struct {
	openapi.OpenAPI

	transportCalls []transportCall
	transportResp  []byte
	transportErr   error

	groupMessages []dto.APIMessage
	groupErr      error
}

type transportCall struct {
	method string
	url    string
	body   any
}

func (f *fakeQQAPI) Transport(ctx context.Context, method, url string, body any) ([]byte, error) {
	f.transportCalls = append(f.transportCalls, transportCall{
		method: method,
		url:    url,
		body:   body,
	})
	return f.transportResp, f.transportErr
}

func (f *fakeQQAPI) PostGroupMessage(
	ctx context.Context,
	groupID string,
	msg dto.APIMessage,
	opt ...options.Option,
) (*dto.Message, error) {
	f.groupMessages = append(f.groupMessages, msg)
	if f.groupErr != nil {
		return nil, f.groupErr
	}
	return &dto.Message{}, nil
}

func TestSendMedia_LocalFileUploadsThenSendsRichMediaMessage(t *testing.T) {
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "report.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4 test"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	store := media.NewFileMediaStore()
	ref, err := store.Store(pdfPath, media.MediaMeta{
		Filename:    "report.pdf",
		ContentType: "application/pdf",
		Source:      "test",
	}, "scope")
	if err != nil {
		t.Fatalf("store media: %v", err)
	}

	uploadedFileInfo := []byte("uploaded-file-info")
	respBody, err := json.Marshal(struct {
		FileInfo []byte `json:"file_info"`
	}{
		FileInfo: uploadedFileInfo,
	})
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	api := &fakeQQAPI{transportResp: respBody}
	ch := &QQChannel{
		BaseChannel: channels.NewBaseChannel("qq", nil, bus.NewMessageBus(), nil),
		api:         api,
	}
	ch.SetRunning(true)
	ch.SetMediaStore(store)
	ch.chatType.Store("group-1", "group")

	err = ch.SendMedia(context.Background(), bus.OutboundMediaMessage{
		ChatID: "group-1",
		Parts: []bus.MediaPart{
			{Ref: ref, Type: "file"},
		},
	})
	if err != nil {
		t.Fatalf("SendMedia() error = %v", err)
	}

	if len(api.transportCalls) != 1 {
		t.Fatalf("transport call count = %d, want 1", len(api.transportCalls))
	}

	payload, ok := api.transportCalls[0].body.(map[string]any)
	if !ok {
		t.Fatalf("transport body type = %T, want map[string]any", api.transportCalls[0].body)
	}
	switch v := payload["file_type"].(type) {
	case int:
		if v != 4 {
			t.Fatalf("file_type = %v, want 4", payload["file_type"])
		}
	case float64:
		if v != 4 {
			t.Fatalf("file_type = %v, want 4", payload["file_type"])
		}
	case uint64:
		if v != 4 {
			t.Fatalf("file_type = %v, want 4", payload["file_type"])
		}
	default:
		t.Fatalf("file_type type = %T, want numeric 4", payload["file_type"])
	}
	if payload["srv_send_msg"] != false {
		t.Fatalf("srv_send_msg = %v, want false", payload["srv_send_msg"])
	}
	fileData, _ := payload["file_data"].(string)
	if fileData == "" {
		t.Fatal("file_data is empty, want base64-encoded local file contents")
	}
	if payload["file_name"] != "report.pdf" {
		t.Fatalf("file_name = %v, want %q", payload["file_name"], "report.pdf")
	}

	if len(api.groupMessages) != 1 {
		t.Fatalf("group message count = %d, want 1", len(api.groupMessages))
	}

	msg, ok := api.groupMessages[0].(*dto.MessageToCreate)
	if !ok {
		t.Fatalf("group message type = %T, want *dto.MessageToCreate", api.groupMessages[0])
	}
	if msg.MsgType != dto.RichMediaMsg {
		t.Fatalf("msg_type = %v, want %v", msg.MsgType, dto.RichMediaMsg)
	}
	if msg.Media == nil {
		t.Fatal("msg.Media is nil, want uploaded file_info")
	}
	if string(msg.Media.FileInfo) != string(uploadedFileInfo) {
		t.Fatalf("file_info = %q, want %q", string(msg.Media.FileInfo), string(uploadedFileInfo))
	}
}

func TestSendMedia_LocalFileUploadFailureReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "notes.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	store := media.NewFileMediaStore()
	ref, err := store.Store(filePath, media.MediaMeta{
		Filename:    "notes.txt",
		ContentType: "text/plain",
		Source:      "test",
	}, "scope")
	if err != nil {
		t.Fatalf("store media: %v", err)
	}

	api := &fakeQQAPI{transportErr: errors.New("upload failed")}
	ch := &QQChannel{
		BaseChannel: channels.NewBaseChannel("qq", nil, bus.NewMessageBus(), nil),
		api:         api,
	}
	ch.SetRunning(true)
	ch.SetMediaStore(store)
	ch.chatType.Store("group-1", "group")

	err = ch.SendMedia(context.Background(), bus.OutboundMediaMessage{
		ChatID: "group-1",
		Parts: []bus.MediaPart{
			{Ref: ref, Type: "file"},
		},
	})
	if err == nil {
		t.Fatal("SendMedia() error = nil, want upload failure")
	}
	if len(api.groupMessages) != 0 {
		t.Fatalf("group message count = %d, want 0 after upload failure", len(api.groupMessages))
	}
}

func TestSendMedia_RemoteURLStillUsesRichMediaDirectSend(t *testing.T) {
	api := &fakeQQAPI{}
	ch := &QQChannel{
		BaseChannel: channels.NewBaseChannel("qq", nil, bus.NewMessageBus(), nil),
		api:         api,
	}
	ch.SetRunning(true)
	ch.chatType.Store("group-1", "group")

	err := ch.SendMedia(context.Background(), bus.OutboundMediaMessage{
		ChatID: "group-1",
		Parts: []bus.MediaPart{
			{
				Ref:  "https://example.com/report.pdf",
				Type: "file",
			},
		},
	})
	if err != nil {
		t.Fatalf("SendMedia() error = %v", err)
	}
	if len(api.transportCalls) != 0 {
		t.Fatalf("transport call count = %d, want 0 for remote URL", len(api.transportCalls))
	}
	if len(api.groupMessages) != 1 {
		t.Fatalf("group message count = %d, want 1", len(api.groupMessages))
	}

	msg, ok := api.groupMessages[0].(*dto.RichMediaMessage)
	if !ok {
		t.Fatalf("group message type = %T, want *dto.RichMediaMessage", api.groupMessages[0])
	}
	if msg.URL != "https://example.com/report.pdf" {
		t.Fatalf("URL = %q, want %q", msg.URL, "https://example.com/report.pdf")
	}
	if !msg.SrvSendMsg {
		t.Fatal("SrvSendMsg = false, want true")
	}
}
