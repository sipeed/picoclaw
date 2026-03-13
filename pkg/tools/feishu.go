package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type FeishuParseTool struct{}

func NewFeishuParseTool() *FeishuParseTool {
	return &FeishuParseTool{}
}

func (t *FeishuParseTool) Name() string {
	return "feishu_parse"
}

func (t *FeishuParseTool) Description() string {
	return "Parse Feishu message content, card JSON, or share links into a structured summary without calling remote APIs. Useful for reasoning about Feishu payloads locally."
}

func (t *FeishuParseTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"mode": map[string]any{
				"type":        "string",
				"description": "One of: message_content, card, share_link",
				"enum":        []string{"message_content", "card", "share_link"},
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Raw Feishu JSON string or share link to parse",
			},
		},
		"required": []string{"mode", "content"},
	}
}

func (t *FeishuParseTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	_ = ctx
	mode, _ := args["mode"].(string)
	content, _ := args["content"].(string)
	if strings.TrimSpace(mode) == "" || strings.TrimSpace(content) == "" {
		return ErrorResult("mode and content are required")
	}

	switch mode {
	case "share_link":
		result := map[string]any{"token": extractFeishuShareToken(content)}
		if decoded := decodeFeishuShareToken(result["token"].(string)); decoded != "" {
			result["decoded_message_id"] = decoded
		}
		return structuredToolResult("feishu share link parsed", result)
	case "message_content":
		var body map[string]any
		if err := json.Unmarshal([]byte(content), &body); err != nil {
			return ErrorResult(fmt.Sprintf("invalid message content JSON: %v", err)).WithError(err)
		}
		result := map[string]any{
			"text":      firstJSONStringField(body, "text"),
			"image_key": firstJSONStringFieldDeep(body, "image_key"),
			"file_key":  firstJSONStringFieldDeep(body, "file_key"),
			"file_name": firstJSONStringFieldDeep(body, "file_name"),
			"raw":       body,
		}
		return structuredToolResult("feishu message content parsed", result)
	case "card":
		var card map[string]any
		if err := json.Unmarshal([]byte(content), &card); err != nil {
			return ErrorResult(fmt.Sprintf("invalid card JSON: %v", err)).WithError(err)
		}
		summary := summarizeFeishuCard(card)
		return structuredToolResult("feishu card parsed", summary)
	default:
		return ErrorResult("unsupported mode")
	}
}

func structuredToolResult(prefix string, value any) *ToolResult {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ErrorResult(fmt.Sprintf("marshal result: %v", err)).WithError(err)
	}
	return UserResult(prefix + "\n" + string(data))
}

func extractFeishuShareToken(link string) string {
	const marker = "token="
	idx := strings.Index(link, marker)
	if idx < 0 {
		return ""
	}
	token := link[idx+len(marker):]
	if cut := strings.Index(token, "&"); cut >= 0 {
		token = token[:cut]
	}
	return token
}

func decodeFeishuShareToken(token string) string {
	replacer := strings.NewReplacer("%3D", "=", "%3d", "=")
	return replacer.Replace(token)
}

func firstJSONStringField(body map[string]any, field string) string {
	if body == nil {
		return ""
	}
	if v, ok := body[field].(string); ok {
		return v
	}
	return ""
}

func firstJSONStringFieldDeep(v any, field string) string {
	switch vv := v.(type) {
	case map[string]any:
		if value, ok := vv[field].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
		for _, nested := range vv {
			if found := firstJSONStringFieldDeep(nested, field); found != "" {
				return found
			}
		}
	case []any:
		for _, nested := range vv {
			if found := firstJSONStringFieldDeep(nested, field); found != "" {
				return found
			}
		}
	}
	return ""
}

func summarizeFeishuCard(card map[string]any) map[string]any {
	summary := map[string]any{
		"title":          "",
		"components":     []map[string]any{},
		"text_contents":  []string{},
		"image_keys":     []string{},
		"action_buttons": []string{},
	}
	if header, ok := card["header"].(map[string]any); ok {
		if title, ok := header["title"].(map[string]any); ok {
			if content, ok := title["content"].(string); ok {
				summary["title"] = content
			}
		}
	}
	if title, ok := card["title"].(string); ok && title != "" {
		summary["title"] = title
	}
	var elements []any
	if raw, ok := card["elements"].([]any); ok {
		elements = raw
	} else if body, ok := card["body"].(map[string]any); ok {
		if raw, ok := body["elements"].([]any); ok {
			elements = raw
		}
	} else if raw, ok := card["content"].([]any); ok {
		elements = raw
	}
	texts := make([]string, 0)
	images := make([]string, 0)
	buttons := make([]string, 0)
	components := make([]map[string]any, 0, len(elements))
	for i, raw := range elements {
		element, _ := raw.(map[string]any)
		tag, _ := element["tag"].(string)
		component := map[string]any{"index": i + 1, "tag": tag, "details": map[string]any{}}
		details := component["details"].(map[string]any)
		switch tag {
		case "div", "note", "header":
			if text, ok := element["text"].(map[string]any); ok {
				if content, ok := text["content"].(string); ok && content != "" {
					texts = append(texts, content)
					details["text"] = content
				}
			}
			if fields, ok := element["fields"].([]any); ok && len(fields) > 0 {
				details["fields"] = fields
				details["field_count"] = len(fields)
			}
		case "text":
			if content, ok := element["text"].(string); ok && content != "" {
				texts = append(texts, content)
				details["text"] = content
			}
			if mode, ok := element["mode"].(string); ok && mode != "" {
				details["mode"] = mode
			}
		case "img":
			if key, ok := element["image_key"].(string); ok && key != "" {
				images = append(images, key)
				details["image_key"] = key
			}
			if alt, ok := element["alt"]; ok {
				details["alt"] = alt
			}
		case "action":
			if actions, ok := element["actions"].([]any); ok {
				parsedActions := make([]map[string]any, 0, len(actions))
				for _, rawAction := range actions {
					action, _ := rawAction.(map[string]any)
					entry := map[string]any{"type": action["type"], "style": action["style"]}
					if text, ok := action["text"].(map[string]any); ok {
						if content, ok := text["content"].(string); ok && content != "" {
							buttons = append(buttons, content)
							entry["text"] = content
						}
					}
					parsedActions = append(parsedActions, entry)
				}
				details["action_count"] = len(actions)
				details["actions"] = parsedActions
			}
		case "at":
			details["user_id"] = element["user_id"]
			details["user_name"] = element["user_name"]
			details["user_avatar"] = element["user_avatar"]
		default:
			for k, v := range element {
				details[k] = v
			}
		}
		components = append(components, component)
	}
	summary["components"] = components
	summary["text_contents"] = texts
	summary["image_keys"] = images
	summary["action_buttons"] = buttons
	return summary
}

type FeishuRemoteClient interface {
	GetMessage(ctx context.Context, messageID string) (any, error)
	ListMessages(ctx context.Context, containerID, containerType string, pageSize int, pageToken string) (any, error)
	ReplyMessage(ctx context.Context, messageID, text string) error
	GetMessageFromShareLink(ctx context.Context, shareLink string) (any, error)
	GetUserInfo(ctx context.Context, userID string) (any, error)
	ListUsers(ctx context.Context, pageSize int, userIDType, pageToken string) (any, error)
	GetUserIDByEmail(ctx context.Context, email string) (string, error)
	GetUserIDByMobile(ctx context.Context, mobile string) (string, error)
	CreateGroup(ctx context.Context, name string) (any, error)
	GetGroupInfo(ctx context.Context, chatID string) (any, error)
	ListGroupMembers(ctx context.Context, chatID string, pageSize int, pageToken string) (any, error)
	ListGroups(ctx context.Context, pageSize int, pageToken string) (any, error)
	SendGroupMessage(ctx context.Context, chatID, text string) error
	GetDriveRootFolder(ctx context.Context) (any, error)
	GetDriveFolder(ctx context.Context, folderToken string) (any, error)
	GetDriveFile(ctx context.Context, fileToken string) (any, error)
	ListDriveFiles(ctx context.Context, folderToken, pageToken string, pageSize int) (any, error)
	DownloadDriveFile(ctx context.Context, fileToken string) (any, error)
	DeleteDriveFile(ctx context.Context, fileToken string) error
	UploadDriveFile(ctx context.Context, parentToken, name string, data []byte) (any, error)
	InitiateMultipartUpload(ctx context.Context, parentToken, name string, size int64) (any, error)
	UploadMultipartChunk(ctx context.Context, uploadID string, seq int, data []byte) error
	CompleteMultipartUpload(ctx context.Context, uploadID string, blockNum int) (any, error)
	SendImageMessage(ctx context.Context, chatID string, data []byte, fileName string) error
	SendFileMessage(ctx context.Context, chatID string, data []byte, fileName, fileType string) error
}

type FeishuRemoteTool struct {
	client FeishuRemoteClient
}

func NewFeishuRemoteTool(client FeishuRemoteClient) *FeishuRemoteTool {
	return &FeishuRemoteTool{client: client}
}

func (t *FeishuRemoteTool) Name() string {
	return "feishu_remote"
}

func (t *FeishuRemoteTool) Description() string {
	return "Query and operate on Feishu messages, users, groups, Drive files, and multipart uploads through an injected Feishu client."
}

func (t *FeishuRemoteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"description": "Feishu remote action to run",
				"enum": []string{
					"get_message", "list_messages", "reply_message", "get_message_from_share_link",
					"get_user", "list_users", "get_user_id_by_email", "get_user_id_by_mobile", "get_user_id_by_phone",
					"create_group", "get_group", "list_group_members", "list_groups", "send_group_message",
					"get_drive_root_folder", "get_drive_folder", "get_drive_file", "list_drive_files",
					"download_drive_file", "download_image_to_bytes", "download_image_from_message", "delete_drive_file", "upload_drive_file",
					"initiate_multipart_upload", "upload_multipart_chunk", "complete_multipart_upload",
					"send_image", "send_image_from_url", "send_file",
				},
			},
			"id": map[string]any{
				"type": "string",
				"description": "Primary identifier such as message ID, chat ID, user ID, share link, file token, or upload ID depending on action",
			},
			"text": map[string]any{
				"type": "string",
				"description": "Text payload used by reply_message and send_group_message",
			},
			"name": map[string]any{
				"type": "string",
				"description": "Group name or upload file name depending on action",
			},
			"container_type": map[string]any{
				"type": "string",
				"description": "For list_messages only; defaults to chat",
			},
			"user_id_type": map[string]any{
				"type": "string",
				"description": "For list_users only; defaults to open_id",
			},
			"page_size": map[string]any{
				"type": "integer",
				"description": "Page size for list operations",
			},
			"page_token": map[string]any{
				"type": "string",
				"description": "Page token for list operations",
			},
			"parent_token": map[string]any{
				"type": "string",
				"description": "Parent folder token for drive upload or multipart prepare",
			},
			"data_base64": map[string]any{
				"type": "string",
				"description": "Base64-encoded bytes for upload_drive_file or upload_multipart_chunk",
			},
			"seq": map[string]any{
				"type": "integer",
				"description": "Chunk sequence number for upload_multipart_chunk",
			},
			"size": map[string]any{
				"type": "integer",
				"description": "File size for initiate_multipart_upload",
			},
			"block_num": map[string]any{
				"type": "integer",
				"description": "Uploaded chunk count for complete_multipart_upload",
			},
			"file_type": map[string]any{
				"type": "string",
				"description": "Optional file type for send_file, such as stream, audio, or video",
			},
			"url": map[string]any{
				"type": "string",
				"description": "Image URL for send_image_from_url",
			},
			"resource_id": map[string]any{
				"type": "string",
				"description": "Optional image token or file token used by download_image_from_message",
			},
		},
		"required": []string{"action"},
	}
}

func (t *FeishuRemoteTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if t.client == nil {
		return ErrorResult("feishu remote client is not configured")
	}
	action, _ := args["action"].(string)
	if strings.TrimSpace(action) == "" {
		return ErrorResult("action is required")
	}
	id, _ := args["id"].(string)
	text, _ := args["text"].(string)
	name, _ := args["name"].(string)
	parentToken, _ := args["parent_token"].(string)
	pageToken, _ := args["page_token"].(string)
	containerType, _ := args["container_type"].(string)
	userIDType, _ := args["user_id_type"].(string)
	dataBase64, _ := args["data_base64"].(string)
	fileType, _ := args["file_type"].(string)
	imageURL, _ := args["url"].(string)
	resourceID, _ := args["resource_id"].(string)

	pageSize := getIntArg(args, "page_size", 20)
	seq := getIntArg(args, "seq", -1)
	blockNum := getIntArg(args, "block_num", -1)
	size := int64(getIntArg(args, "size", -1))

	var (
		result any
		err    error
	)
	switch action {
	case "get_message":
		if strings.TrimSpace(id) == "" { return ErrorResult("id is required") }
		result, err = t.client.GetMessage(ctx, id)
	case "list_messages":
		if strings.TrimSpace(id) == "" { return ErrorResult("id is required") }
		result, err = t.client.ListMessages(ctx, id, containerType, pageSize, pageToken)
	case "reply_message":
		if strings.TrimSpace(id) == "" || strings.TrimSpace(text) == "" { return ErrorResult("id and text are required") }
		err = t.client.ReplyMessage(ctx, id, text)
		result = map[string]any{"message_id": id, "text": text, "status": "ok"}
	case "get_message_from_share_link":
		if strings.TrimSpace(id) == "" { return ErrorResult("id is required") }
		result, err = t.client.GetMessageFromShareLink(ctx, id)
	case "get_user":
		if strings.TrimSpace(id) == "" { return ErrorResult("id is required") }
		result, err = t.client.GetUserInfo(ctx, id)
	case "list_users":
		result, err = t.client.ListUsers(ctx, pageSize, userIDType, pageToken)
	case "get_user_id_by_email":
		if strings.TrimSpace(id) == "" { return ErrorResult("id is required") }
		var userID string
		userID, err = t.client.GetUserIDByEmail(ctx, id)
		result = map[string]any{"email": id, "user_id": userID}
	case "get_user_id_by_mobile":
		if strings.TrimSpace(id) == "" { return ErrorResult("id is required") }
		var userID string
		userID, err = t.client.GetUserIDByMobile(ctx, id)
		result = map[string]any{"mobile": id, "user_id": userID}
	case "get_user_id_by_phone":
		if strings.TrimSpace(id) == "" { return ErrorResult("id is required") }
		var userID string
		userID, err = t.client.GetUserIDByMobile(ctx, id)
		result = map[string]any{"phone": id, "user_id": userID}
	case "create_group":
		if strings.TrimSpace(name) == "" { return ErrorResult("name is required") }
		result, err = t.client.CreateGroup(ctx, name)
	case "get_group":
		if strings.TrimSpace(id) == "" { return ErrorResult("id is required") }
		result, err = t.client.GetGroupInfo(ctx, id)
	case "list_group_members":
		if strings.TrimSpace(id) == "" { return ErrorResult("id is required") }
		result, err = t.client.ListGroupMembers(ctx, id, pageSize, pageToken)
	case "list_groups":
		result, err = t.client.ListGroups(ctx, pageSize, pageToken)
	case "send_group_message":
		if strings.TrimSpace(id) == "" || strings.TrimSpace(text) == "" { return ErrorResult("id and text are required") }
		err = t.client.SendGroupMessage(ctx, id, text)
		result = map[string]any{"chat_id": id, "text": text, "status": "ok"}
	case "get_drive_root_folder":
		result, err = t.client.GetDriveRootFolder(ctx)
	case "get_drive_folder":
		if strings.TrimSpace(id) == "" { return ErrorResult("id is required") }
		result, err = t.client.GetDriveFolder(ctx, id)
	case "get_drive_file":
		if strings.TrimSpace(id) == "" { return ErrorResult("id is required") }
		result, err = t.client.GetDriveFile(ctx, id)
	case "list_drive_files":
		result, err = t.client.ListDriveFiles(ctx, id, pageToken, pageSize)
	case "download_drive_file", "download_image_to_bytes":
		if strings.TrimSpace(id) == "" { return ErrorResult("id is required") }
		result, err = t.client.DownloadDriveFile(ctx, id)
	case "download_image_from_message":
		if strings.TrimSpace(id) == "" { return ErrorResult("id is required") }
		resolvedResourceID := strings.TrimSpace(resourceID)
		message, getErr := t.client.GetMessage(ctx, id)
		if getErr != nil {
			err = getErr
			break
		}
		messageMap := normalizeFeishuMessageToolValue(message)
		if resolvedResourceID == "" {
			resolvedResourceID = extractImageTokenFromMessageValue(messageMap)
			if resolvedResourceID == "" {
				return ErrorResult("resource_id is required when message content does not contain an image token")
			}
		}
		result, err = t.client.DownloadDriveFile(ctx, resolvedResourceID)
		if err == nil {
			result = map[string]any{"message_id": id, "resource_id": resolvedResourceID, "download": result, "message": messageMap}
		}
	case "delete_drive_file":
		if strings.TrimSpace(id) == "" { return ErrorResult("id is required") }
		err = t.client.DeleteDriveFile(ctx, id)
		result = map[string]any{"file_token": id, "status": "deleted"}
	case "upload_drive_file":
		if strings.TrimSpace(name) == "" || strings.TrimSpace(dataBase64) == "" { return ErrorResult("name and data_base64 are required") }
		var data []byte
		data, err = base64.StdEncoding.DecodeString(dataBase64)
		if err == nil {
			result, err = t.client.UploadDriveFile(ctx, parentToken, name, data)
		}
	case "initiate_multipart_upload":
		if strings.TrimSpace(name) == "" || size < 0 { return ErrorResult("name and size are required") }
		result, err = t.client.InitiateMultipartUpload(ctx, parentToken, name, size)
	case "upload_multipart_chunk":
		if strings.TrimSpace(id) == "" || seq < 0 || strings.TrimSpace(dataBase64) == "" { return ErrorResult("id, seq and data_base64 are required") }
		var data []byte
		data, err = base64.StdEncoding.DecodeString(dataBase64)
		if err == nil {
			err = t.client.UploadMultipartChunk(ctx, id, seq, data)
			result = map[string]any{"upload_id": id, "seq": seq, "size": len(data), "status": "ok"}
		}
	case "complete_multipart_upload":
		if strings.TrimSpace(id) == "" { return ErrorResult("id is required") }
		if blockNum < 0 { return ErrorResult("block_num is required") }
		result, err = t.client.CompleteMultipartUpload(ctx, id, blockNum)
	case "send_image":
		if strings.TrimSpace(id) == "" || strings.TrimSpace(dataBase64) == "" { return ErrorResult("id and data_base64 are required") }
		if strings.TrimSpace(name) == "" { name = "image.bin" }
		var data []byte
		data, err = base64.StdEncoding.DecodeString(dataBase64)
		if err == nil {
			err = t.client.SendImageMessage(ctx, id, data, name)
			result = map[string]any{"chat_id": id, "file_name": name, "size": len(data), "status": "ok"}
		}
	case "send_image_from_url":
		if strings.TrimSpace(id) == "" || strings.TrimSpace(imageURL) == "" { return ErrorResult("id and url are required") }
		if strings.TrimSpace(name) == "" { name = imageNameFromURL(imageURL) }
		var data []byte
		data, err = fetchURLBytes(ctx, imageURL)
		if err == nil {
			err = t.client.SendImageMessage(ctx, id, data, name)
			result = map[string]any{"chat_id": id, "file_name": name, "source_url": imageURL, "size": len(data), "status": "ok"}
		}
	case "send_file":
		if strings.TrimSpace(id) == "" || strings.TrimSpace(dataBase64) == "" { return ErrorResult("id and data_base64 are required") }
		if strings.TrimSpace(name) == "" { return ErrorResult("name is required") }
		if strings.TrimSpace(fileType) == "" { fileType = "stream" }
		var data []byte
		data, err = base64.StdEncoding.DecodeString(dataBase64)
		if err == nil {
			err = t.client.SendFileMessage(ctx, id, data, name, fileType)
			result = map[string]any{"chat_id": id, "file_name": name, "file_type": fileType, "size": len(data), "status": "ok"}
		}
	default:
		return ErrorResult("unsupported action")
	}
	if err != nil {
		return ErrorResult(fmt.Sprintf("feishu remote query failed: %v", err)).WithError(err)
	}
	return structuredToolResult("feishu remote query result", result)
}

func getIntArg(args map[string]any, key string, fallback int) int {
	switch v := args[key].(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return fallback
	}
}

func imageNameFromURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "image.bin"
	}
	parts := strings.Split(trimmed, "/")
	last := parts[len(parts)-1]
	if last == "" || !strings.Contains(last, ".") {
		return "image.bin"
	}
	if cut := strings.Index(last, "?"); cut >= 0 {
		last = last[:cut]
	}
	if last == "" {
		return "image.bin"
	}
	return last
}

func fetchURLBytes(ctx context.Context, raw string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func normalizeFeishuMessageToolValue(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		if nested, ok := m["message"].(map[string]any); ok {
			return nested
		}
		return m
	}
	return map[string]any{}
}

func extractImageTokenFromMessageValue(message map[string]any) string {
	if len(message) == 0 {
		return ""
	}
	if parsed, ok := message["parsed"].(map[string]any); ok {
		if token := extractImageTokenFromContentValue(parsed["content"]); token != "" {
			return token
		}
	}
	if body, ok := message["body"].(map[string]any); ok {
		if raw, ok := body["content"].(string); ok && strings.TrimSpace(raw) != "" {
			var decoded any
			if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
				if token := extractImageTokenFromContentValue(decoded); token != "" {
					return token
				}
			}
		}
	}
	return extractImageTokenFromContentValue(message["content"])
}

func extractImageTokenFromContentValue(v any) string {
	switch vv := v.(type) {
	case map[string]any:
		if token, ok := vv["image_token"].(string); ok && strings.TrimSpace(token) != "" {
			return token
		}
		if token, ok := vv["image_key"].(string); ok && strings.TrimSpace(token) != "" {
			return token
		}
	case []any:
		for _, item := range vv {
			if token := extractImageTokenFromContentValue(item); token != "" {
				return token
			}
		}
	}
	return ""
}
