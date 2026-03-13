package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

type mockFeishuRemoteClient struct{}

func (m *mockFeishuRemoteClient) GetMessage(ctx context.Context, messageID string) (any, error) {
	switch messageID {
	case "om_with_image":
		return map[string]any{
			"message_id": messageID,
			"body": map[string]any{"content": `{"image_key":"img_from_message"}`},
		}, nil
	case "om_without_image":
		return map[string]any{
			"message_id": messageID,
			"body": map[string]any{"content": `{"text":"hello"}`},
		}, nil
	default:
		return map[string]any{"message_id": messageID}, nil
	}
}
func (m *mockFeishuRemoteClient) ListMessages(ctx context.Context, containerID, containerType string, pageSize int, pageToken string) (any, error) {
	return map[string]any{"container_id": containerID, "container_type": containerType, "page_size": pageSize, "page_token": pageToken}, nil
}
func (m *mockFeishuRemoteClient) ReplyMessage(ctx context.Context, messageID, text string) error { return nil }
func (m *mockFeishuRemoteClient) GetMessageFromShareLink(ctx context.Context, shareLink string) (any, error) {
	return map[string]any{"share_link": shareLink}, nil
}
func (m *mockFeishuRemoteClient) GetUserInfo(ctx context.Context, userID string) (any, error) {
	return map[string]any{"user_id": userID}, nil
}
func (m *mockFeishuRemoteClient) ListUsers(ctx context.Context, pageSize int, userIDType, pageToken string) (any, error) {
	return map[string]any{"page_size": pageSize, "user_id_type": userIDType, "page_token": pageToken}, nil
}
func (m *mockFeishuRemoteClient) GetUserIDByEmail(ctx context.Context, email string) (string, error) {
	return "ou_email", nil
}
func (m *mockFeishuRemoteClient) GetUserIDByMobile(ctx context.Context, mobile string) (string, error) {
	return "ou_mobile", nil
}
func (m *mockFeishuRemoteClient) CreateGroup(ctx context.Context, name string) (any, error) {
	return map[string]any{"name": name}, nil
}
func (m *mockFeishuRemoteClient) GetGroupInfo(ctx context.Context, chatID string) (any, error) {
	return map[string]any{"chat_id": chatID}, nil
}
func (m *mockFeishuRemoteClient) ListGroupMembers(ctx context.Context, chatID string, pageSize int, pageToken string) (any, error) {
	return map[string]any{"chat_id": chatID, "page_size": pageSize, "page_token": pageToken}, nil
}
func (m *mockFeishuRemoteClient) ListGroups(ctx context.Context, pageSize int, pageToken string) (any, error) {
	return map[string]any{"page_size": pageSize, "page_token": pageToken}, nil
}
func (m *mockFeishuRemoteClient) SendGroupMessage(ctx context.Context, chatID, text string) error { return nil }
func (m *mockFeishuRemoteClient) GetDriveRootFolder(ctx context.Context) (any, error) {
	return map[string]any{"folder_token": "root"}, nil
}
func (m *mockFeishuRemoteClient) GetDriveFolder(ctx context.Context, folderToken string) (any, error) {
	return map[string]any{"folder_token": folderToken}, nil
}
func (m *mockFeishuRemoteClient) GetDriveFile(ctx context.Context, fileToken string) (any, error) {
	return map[string]any{"file_token": fileToken}, nil
}
func (m *mockFeishuRemoteClient) ListDriveFiles(ctx context.Context, folderToken, pageToken string, pageSize int) (any, error) {
	return map[string]any{"folder_token": folderToken, "page_token": pageToken, "page_size": pageSize}, nil
}
func (m *mockFeishuRemoteClient) DownloadDriveFile(ctx context.Context, fileToken string) (any, error) {
	return map[string]any{"name": "demo.txt", "content_type": "text/plain", "data_base64": base64.StdEncoding.EncodeToString([]byte("demo"))}, nil
}
func (m *mockFeishuRemoteClient) DeleteDriveFile(ctx context.Context, fileToken string) error { return nil }
func (m *mockFeishuRemoteClient) UploadDriveFile(ctx context.Context, parentToken, name string, data []byte) (any, error) {
	return map[string]any{"parent_token": parentToken, "name": name, "size": len(data)}, nil
}
func (m *mockFeishuRemoteClient) InitiateMultipartUpload(ctx context.Context, parentToken, name string, size int64) (any, error) {
	return map[string]any{"upload_id": "upload_1", "parent_token": parentToken, "name": name, "size": size}, nil
}
func (m *mockFeishuRemoteClient) UploadMultipartChunk(ctx context.Context, uploadID string, seq int, data []byte) error { return nil }
func (m *mockFeishuRemoteClient) CompleteMultipartUpload(ctx context.Context, uploadID string, blockNum int) (any, error) {
	return map[string]any{"upload_id": uploadID, "block_num": blockNum}, nil
}
func (m *mockFeishuRemoteClient) SendImageMessage(ctx context.Context, chatID string, data []byte, fileName string) error {
	return nil
}
func (m *mockFeishuRemoteClient) SendFileMessage(ctx context.Context, chatID string, data []byte, fileName, fileType string) error {
	return nil
}

type errFeishuRemoteClient struct{}

func (e *errFeishuRemoteClient) GetMessage(ctx context.Context, messageID string) (any, error) { return nil, fmt.Errorf("boom") }
func (e *errFeishuRemoteClient) ListMessages(ctx context.Context, containerID, containerType string, pageSize int, pageToken string) (any, error) {
	return nil, fmt.Errorf("boom")
}
func (e *errFeishuRemoteClient) ReplyMessage(ctx context.Context, messageID, text string) error { return fmt.Errorf("boom") }
func (e *errFeishuRemoteClient) GetMessageFromShareLink(ctx context.Context, shareLink string) (any, error) { return nil, fmt.Errorf("boom") }
func (e *errFeishuRemoteClient) GetUserInfo(ctx context.Context, userID string) (any, error) { return nil, fmt.Errorf("boom") }
func (e *errFeishuRemoteClient) ListUsers(ctx context.Context, pageSize int, userIDType, pageToken string) (any, error) {
	return nil, fmt.Errorf("boom")
}
func (e *errFeishuRemoteClient) GetUserIDByEmail(ctx context.Context, email string) (string, error) { return "", fmt.Errorf("boom") }
func (e *errFeishuRemoteClient) GetUserIDByMobile(ctx context.Context, mobile string) (string, error) { return "", fmt.Errorf("boom") }
func (e *errFeishuRemoteClient) CreateGroup(ctx context.Context, name string) (any, error) { return nil, fmt.Errorf("boom") }
func (e *errFeishuRemoteClient) GetGroupInfo(ctx context.Context, chatID string) (any, error) { return nil, fmt.Errorf("boom") }
func (e *errFeishuRemoteClient) ListGroupMembers(ctx context.Context, chatID string, pageSize int, pageToken string) (any, error) {
	return nil, fmt.Errorf("boom")
}
func (e *errFeishuRemoteClient) ListGroups(ctx context.Context, pageSize int, pageToken string) (any, error) { return nil, fmt.Errorf("boom") }
func (e *errFeishuRemoteClient) SendGroupMessage(ctx context.Context, chatID, text string) error { return fmt.Errorf("boom") }
func (e *errFeishuRemoteClient) GetDriveRootFolder(ctx context.Context) (any, error) { return nil, fmt.Errorf("boom") }
func (e *errFeishuRemoteClient) GetDriveFolder(ctx context.Context, folderToken string) (any, error) { return nil, fmt.Errorf("boom") }
func (e *errFeishuRemoteClient) GetDriveFile(ctx context.Context, fileToken string) (any, error) { return nil, fmt.Errorf("boom") }
func (e *errFeishuRemoteClient) ListDriveFiles(ctx context.Context, folderToken, pageToken string, pageSize int) (any, error) {
	return nil, fmt.Errorf("boom")
}
func (e *errFeishuRemoteClient) DownloadDriveFile(ctx context.Context, fileToken string) (any, error) { return nil, fmt.Errorf("boom") }
func (e *errFeishuRemoteClient) DeleteDriveFile(ctx context.Context, fileToken string) error { return fmt.Errorf("boom") }
func (e *errFeishuRemoteClient) UploadDriveFile(ctx context.Context, parentToken, name string, data []byte) (any, error) {
	return nil, fmt.Errorf("boom")
}
func (e *errFeishuRemoteClient) InitiateMultipartUpload(ctx context.Context, parentToken, name string, size int64) (any, error) {
	return nil, fmt.Errorf("boom")
}
func (e *errFeishuRemoteClient) UploadMultipartChunk(ctx context.Context, uploadID string, seq int, data []byte) error {
	return fmt.Errorf("boom")
}
func (e *errFeishuRemoteClient) CompleteMultipartUpload(ctx context.Context, uploadID string, blockNum int) (any, error) {
	return nil, fmt.Errorf("boom")
}
func (e *errFeishuRemoteClient) SendImageMessage(ctx context.Context, chatID string, data []byte, fileName string) error {
	return fmt.Errorf("boom")
}
func (e *errFeishuRemoteClient) SendFileMessage(ctx context.Context, chatID string, data []byte, fileName, fileType string) error {
	return fmt.Errorf("boom")
}

func TestFeishuParseToolShareLink(t *testing.T) {
	tool := NewFeishuParseTool()
	result := tool.Execute(context.Background(), map[string]any{"mode": "share_link", "content": "https://applink.feishu.cn/client/message/link/open?token=om_abc%3D%3D&foo=bar"})
	if result.IsError || !strings.Contains(result.ForLLM, "om_abc%3D%3D") {
		t.Fatalf("unexpected result: %s", result.ForLLM)
	}
}

func TestFeishuParseToolMessageContent(t *testing.T) {
	tool := NewFeishuParseTool()
	result := tool.Execute(context.Background(), map[string]any{"mode": "message_content", "content": `{"text":"hello"}`})
	if result.IsError || !strings.Contains(result.ForLLM, "hello") {
		t.Fatalf("unexpected result: %s", result.ForLLM)
	}
}

func TestFeishuParseToolCard(t *testing.T) {
	tool := NewFeishuParseTool()
	result := tool.Execute(context.Background(), map[string]any{"mode": "card", "content": `{"header":{"title":{"content":"demo"}},"elements":[{"tag":"div","text":{"content":"body"}}]}`})
	if result.IsError || !strings.Contains(result.ForLLM, "demo") || !strings.Contains(result.ForLLM, "body") {
		t.Fatalf("unexpected result: %s", result.ForLLM)
	}
}

func TestFeishuParseToolCardWithPythonStyleContent(t *testing.T) {
	tool := NewFeishuParseTool()
	result := tool.Execute(context.Background(), map[string]any{"mode": "card", "content": `{"title":"demo title","content":[{"tag":"text","text":"plain body"},{"tag":"img","image_key":"img_v2_key"},{"tag":"action","actions":[{"text":{"content":"Click"},"type":"primary"}]}]}`})
	if result.IsError || !strings.Contains(result.ForLLM, "demo title") || !strings.Contains(result.ForLLM, "plain body") || !strings.Contains(result.ForLLM, "img_v2_key") || !strings.Contains(result.ForLLM, "Click") {
		t.Fatalf("unexpected result: %s", result.ForLLM)
	}
}

func TestRegisterFeishuToolsEnabled(t *testing.T) {
	registry := NewToolRegistry()
	cfg := &config.Config{}
	RegisterFeishuTools(registry, cfg)
	if _, ok := registry.Get("feishu_parse"); !ok {
		t.Fatal("expected feishu_parse to be registered by default")
	}
}

func TestRegisterFeishuToolsDisabled(t *testing.T) {
	registry := NewToolRegistry()
	cfg := &config.Config{}
	RegisterFeishuTools(registry, cfg)
	if _, ok := registry.Get("feishu_parse"); !ok {
		t.Fatal("expected feishu_parse to be registered")
	}
}

func TestFeishuRemoteToolGetMessage(t *testing.T) {
	tool := NewFeishuRemoteTool(&mockFeishuRemoteClient{})
	result := tool.Execute(context.Background(), map[string]any{"action": "get_message", "id": "om_123"})
	if result.IsError || !strings.Contains(result.ForLLM, "om_123") {
		t.Fatalf("unexpected result: %s", result.ForLLM)
	}
}

func TestFeishuRemoteToolListMessages(t *testing.T) {
	tool := NewFeishuRemoteTool(&mockFeishuRemoteClient{})
	result := tool.Execute(context.Background(), map[string]any{"action": "list_messages", "id": "oc_123", "container_type": "chat", "page_size": 5, "page_token": "next"})
	if result.IsError || !strings.Contains(result.ForLLM, "oc_123") || !strings.Contains(result.ForLLM, "next") {
		t.Fatalf("unexpected result: %s", result.ForLLM)
	}
}

func TestFeishuRemoteToolReplyMessage(t *testing.T) {
	tool := NewFeishuRemoteTool(&mockFeishuRemoteClient{})
	result := tool.Execute(context.Background(), map[string]any{"action": "reply_message", "id": "om_123", "text": "hello"})
	if result.IsError || !strings.Contains(result.ForLLM, "om_123") {
		t.Fatalf("unexpected result: %s", result.ForLLM)
	}
}

func TestFeishuRemoteToolUserLookupActions(t *testing.T) {
	tool := NewFeishuRemoteTool(&mockFeishuRemoteClient{})
	byEmail := tool.Execute(context.Background(), map[string]any{"action": "get_user_id_by_email", "id": "demo@example.com"})
	if byEmail.IsError || !strings.Contains(byEmail.ForLLM, "ou_email") {
		t.Fatalf("unexpected email lookup result: %s", byEmail.ForLLM)
	}
	byMobile := tool.Execute(context.Background(), map[string]any{"action": "get_user_id_by_mobile", "id": "13800000000"})
	if byMobile.IsError || !strings.Contains(byMobile.ForLLM, "ou_mobile") {
		t.Fatalf("unexpected mobile lookup result: %s", byMobile.ForLLM)
	}
	byPhone := tool.Execute(context.Background(), map[string]any{"action": "get_user_id_by_phone", "id": "13800000000"})
	if byPhone.IsError || !strings.Contains(byPhone.ForLLM, "ou_mobile") || !strings.Contains(byPhone.ForLLM, "phone") {
		t.Fatalf("unexpected phone lookup result: %s", byPhone.ForLLM)
	}
}

func TestFeishuRemoteToolGroupAndDriveActions(t *testing.T) {
	tool := NewFeishuRemoteTool(&mockFeishuRemoteClient{})
	cases := []struct {
		name string
		args map[string]any
		want string
	}{
		{name: "create group", args: map[string]any{"action": "create_group", "name": "demo-group"}, want: "demo-group"},
		{name: "list group members", args: map[string]any{"action": "list_group_members", "id": "oc_1", "page_size": 2}, want: "oc_1"},
		{name: "list groups", args: map[string]any{"action": "list_groups", "page_size": 2}, want: "page_size"},
		{name: "send group message", args: map[string]any{"action": "send_group_message", "id": "oc_1", "text": "hello"}, want: "hello"},
		{name: "get drive root", args: map[string]any{"action": "get_drive_root_folder"}, want: "root"},
		{name: "get drive folder", args: map[string]any{"action": "get_drive_folder", "id": "fld_1"}, want: "fld_1"},
		{name: "list drive files", args: map[string]any{"action": "list_drive_files", "id": "fld_1", "page_size": 2}, want: "fld_1"},
		{name: "download drive file", args: map[string]any{"action": "download_drive_file", "id": "file_1"}, want: "demo.txt"},
		{name: "delete drive file", args: map[string]any{"action": "delete_drive_file", "id": "file_1"}, want: "deleted"},
		{name: "initiate multipart", args: map[string]any{"action": "initiate_multipart_upload", "name": "big.bin", "parent_token": "fld_1", "size": 128}, want: "upload_1"},
		{name: "upload multipart chunk", args: map[string]any{"action": "upload_multipart_chunk", "id": "upload_1", "seq": 0, "data_base64": base64.StdEncoding.EncodeToString([]byte("chunk"))}, want: "upload_1"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.Execute(context.Background(), tt.args)
			if result.IsError || !strings.Contains(result.ForLLM, tt.want) {
				t.Fatalf("unexpected result: %s", result.ForLLM)
			}
		})
	}
}

func TestFeishuRemoteToolSendMediaActions(t *testing.T) {
	tool := NewFeishuRemoteTool(&mockFeishuRemoteClient{})
	img := tool.Execute(context.Background(), map[string]any{
		"action":      "send_image",
		"id":          "oc_1",
		"name":        "demo.png",
		"data_base64": base64.StdEncoding.EncodeToString([]byte("imgdata")),
	})
	if img.IsError || !strings.Contains(img.ForLLM, "demo.png") || !strings.Contains(img.ForLLM, "oc_1") {
		t.Fatalf("unexpected send_image result: %s", img.ForLLM)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("pngbytes"))
	}))
	defer ts.Close()

	imgFromURL := tool.Execute(context.Background(), map[string]any{
		"action": "send_image_from_url",
		"id":     "oc_1",
		"url":    ts.URL + "/demo.png",
	})
	if imgFromURL.IsError || !strings.Contains(imgFromURL.ForLLM, "demo.png") || !strings.Contains(imgFromURL.ForLLM, "source_url") {
		t.Fatalf("unexpected send_image_from_url result: %s", imgFromURL.ForLLM)
	}

	file := tool.Execute(context.Background(), map[string]any{
		"action":      "send_file",
		"id":          "oc_1",
		"name":        "demo.txt",
		"file_type":   "stream",
		"data_base64": base64.StdEncoding.EncodeToString([]byte("filedata")),
	})
	if file.IsError || !strings.Contains(file.ForLLM, "demo.txt") || !strings.Contains(file.ForLLM, "stream") {
		t.Fatalf("unexpected send_file result: %s", file.ForLLM)
	}
}

func TestFeishuRemoteToolDownloadImageAlias(t *testing.T) {
	tool := NewFeishuRemoteTool(&mockFeishuRemoteClient{})
	result := tool.Execute(context.Background(), map[string]any{"action": "download_image_to_bytes", "id": "img_1"})
	if result.IsError || !strings.Contains(result.ForLLM, "data_base64") || !strings.Contains(result.ForLLM, "demo.txt") {
		t.Fatalf("unexpected download_image_to_bytes result: %s", result.ForLLM)
	}
}

func TestFeishuRemoteToolDownloadImageFromMessage(t *testing.T) {
	tool := NewFeishuRemoteTool(&mockFeishuRemoteClient{})
	explicit := tool.Execute(context.Background(), map[string]any{
		"action":      "download_image_from_message",
		"id":          "om_1",
		"resource_id": "img_explicit",
	})
	if explicit.IsError || !strings.Contains(explicit.ForLLM, "img_explicit") || !strings.Contains(explicit.ForLLM, "message_id") {
		t.Fatalf("unexpected explicit download_image_from_message result: %s", explicit.ForLLM)
	}

	auto := tool.Execute(context.Background(), map[string]any{
		"action": "download_image_from_message",
		"id":     "om_with_image",
	})
	if auto.IsError || !strings.Contains(auto.ForLLM, "img_from_message") || !strings.Contains(auto.ForLLM, "download") {
		t.Fatalf("unexpected auto download_image_from_message result: %s", auto.ForLLM)
	}
}

func TestFeishuRemoteToolValidationErrors(t *testing.T) {
	tool := NewFeishuRemoteTool(&mockFeishuRemoteClient{})
	cases := []struct {
		name string
		args map[string]any
		want string
	}{
		{name: "missing reply text", args: map[string]any{"action": "reply_message", "id": "om_1"}, want: "id and text are required"},
		{name: "missing create group name", args: map[string]any{"action": "create_group"}, want: "name is required"},
		{name: "missing upload payload", args: map[string]any{"action": "upload_drive_file", "name": "demo.txt"}, want: "name and data_base64 are required"},
		{name: "missing chunk fields", args: map[string]any{"action": "upload_multipart_chunk", "id": "upload_1"}, want: "id, seq and data_base64 are required"},
		{name: "missing send image payload", args: map[string]any{"action": "send_image", "id": "oc_1"}, want: "id and data_base64 are required"},
		{name: "missing send file name", args: map[string]any{"action": "send_file", "id": "oc_1", "data_base64": base64.StdEncoding.EncodeToString([]byte("x"))}, want: "name is required"},
		{name: "missing send image url", args: map[string]any{"action": "send_image_from_url", "id": "oc_1"}, want: "id and url are required"},
		{name: "missing message image token", args: map[string]any{"action": "download_image_from_message", "id": "om_without_image"}, want: "resource_id is required when message content does not contain an image token"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.Execute(context.Background(), tt.args)
			if !result.IsError || !strings.Contains(result.ForLLM, tt.want) {
				t.Fatalf("expected error containing %q, got: %s", tt.want, result.ForLLM)
			}
		})
	}
}

func TestRegisterFeishuToolsWithClient(t *testing.T) {
	registry := NewToolRegistry()
	cfg := &config.Config{}
	RegisterFeishuToolsWithClient(registry, cfg, &mockFeishuRemoteClient{})
	if _, ok := registry.Get("feishu_parse"); !ok {
		t.Fatal("expected feishu_parse to be registered")
	}
	if _, ok := registry.Get("feishu_remote"); !ok {
		t.Fatal("expected feishu_remote to be registered")
	}
}

func TestFeishuChannelAdapterReady(t *testing.T) {
	adapter := NewFeishuChannelAdapter(&mockFeishuRemoteClient{})
	if !adapter.Ready() {
		t.Fatal("expected adapter to be ready")
	}
}

func TestFeishuChannelAdapterNotReady(t *testing.T) {
	adapter := NewFeishuChannelAdapter(struct{}{})
	if adapter.Ready() {
		t.Fatal("did not expect adapter to be ready")
	}
	if _, err := adapter.GetMessage(context.Background(), "om_123"); err == nil {
		t.Fatal("expected error for missing method implementation")
	}
}
