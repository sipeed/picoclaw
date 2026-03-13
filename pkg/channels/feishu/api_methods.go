//go:build amd64 || arm64 || riscv64 || mips64 || ppc64

package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

func (c *FeishuChannel) GetMessage(ctx context.Context, messageID string) (*MessageDetail, error) {
	if strings.TrimSpace(messageID) == "" {
		return nil, fmt.Errorf("message ID is empty")
	}
	var payload struct {
		Code int            `json:"code"`
		Msg  string         `json:"msg"`
		Data map[string]any `json:"data"`
	}
	if err := c.getJSON(ctx, "/open-apis/im/v1/messages/"+url.PathEscape(messageID), nil, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu get message api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	item := payload.Data
	if message, ok := payload.Data["message"].(map[string]any); ok {
		item = message
	}
	detail := normalizeMessageDetail(item)
	return &detail, nil
}

func (c *FeishuChannel) GetMessageByID(ctx context.Context, messageID string) (*MessageDetail, error) {
	return c.GetMessage(ctx, messageID)
}

func (c *FeishuChannel) ListMessages(ctx context.Context, containerID, containerType string, pageSize int, pageToken string) (*MessageList, error) {
	if strings.TrimSpace(containerID) == "" {
		return nil, fmt.Errorf("container ID is empty")
	}
	if containerType == "" {
		containerType = "chat"
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	query := map[string]string{
		"container_id":      containerID,
		"container_id_type": containerType,
		"page_size":         fmt.Sprintf("%d", pageSize),
	}
	if pageToken != "" {
		query["page_token"] = pageToken
	}
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Items     []map[string]any `json:"items"`
			HasMore   bool             `json:"has_more"`
			PageToken string           `json:"page_token"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, "/open-apis/im/v1/messages", query, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu list messages api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	result := &MessageList{HasMore: payload.Data.HasMore, PageToken: payload.Data.PageToken}
	for _, item := range payload.Data.Items {
		result.Items = append(result.Items, normalizeMessageDetail(item))
	}
	return result, nil
}

func (c *FeishuChannel) ReplyMessage(ctx context.Context, messageID, text string) error {
	if strings.TrimSpace(messageID) == "" {
		return fmt.Errorf("message ID is empty")
	}
	req := larkim.NewReplyMessageReqBuilder().
		MessageId(messageID).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			Content(string(mustJSONMarshal(map[string]string{"text": text}))).
			MsgType(larkim.MsgTypeText).
			Build()).
		Build()
	resp, err := c.client.Im.V1.Message.Reply(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu reply message: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("feishu reply message api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}
	return nil
}

func (c *FeishuChannel) ParseMessageContent(raw map[string]any) *ParsedMessage {
	return parseMessageContentPayload(raw)
}

func (c *FeishuChannel) GetMessageFromShareLink(ctx context.Context, shareLink string) (*ShareLinkLookupResult, error) {
	token := extractShareLinkToken(shareLink)
	if token == "" {
		return nil, fmt.Errorf("invalid share link format")
	}
	decoded, _ := url.QueryUnescape(token)
	result := &ShareLinkLookupResult{Token: token, DecodedMessageID: decoded}
	message, err := c.GetMessage(ctx, decoded)
	if err == nil {
		result.Message = message
		return result, nil
	}
	result.FallbackError = err.Error()
	message, err = c.GetMessage(ctx, token)
	if err == nil {
		result.Message = message
		return result, nil
	}
	if err != nil {
		result.FallbackError = strings.Trim(strings.Join([]string{result.FallbackError, err.Error()}, "; "), "; ")
	}
	return result, fmt.Errorf("unable to resolve message from share link")
}

func (c *FeishuChannel) GetUserInfo(ctx context.Context, userID string) (*UserSummary, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is empty")
	}
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			User map[string]any `json:"user"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, "/open-apis/contact/v3/users/"+url.PathEscape(userID), nil, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu get user api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	summary := normalizeUserSummary(payload.Data.User)
	return &summary, nil
}

func (c *FeishuChannel) ListUsers(ctx context.Context, pageSize int, userIDType, pageToken string) (*UserList, error) {
	if pageSize <= 0 {
		pageSize = 50
	}
	if userIDType == "" {
		userIDType = "open_id"
	}
	query := map[string]string{"page_size": fmt.Sprintf("%d", pageSize), "user_id_type": userIDType}
	if pageToken != "" {
		query["page_token"] = pageToken
	}
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			HasMore   bool             `json:"has_more"`
			PageToken string           `json:"page_token"`
			Items     []map[string]any `json:"items"`
			Data      struct {
				HasMore   bool             `json:"has_more"`
				PageToken string           `json:"page_token"`
				Items     []map[string]any `json:"items"`
			} `json:"data"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, "/open-apis/contact/v3/users", query, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu list users api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	data := payload.Data
	items := data.Items
	hasMore := data.HasMore
	nextToken := data.PageToken
	if len(items) == 0 && len(data.Data.Items) > 0 {
		items = data.Data.Items
		hasMore = data.Data.HasMore
		nextToken = data.Data.PageToken
	}
	result := &UserList{HasMore: hasMore, PageToken: nextToken}
	for _, item := range items {
		result.Items = append(result.Items, normalizeUserSummary(item))
	}
	return result, nil
}

func (c *FeishuChannel) GetUserIDByEmail(ctx context.Context, email string) (string, error) {
	users, err := c.ListUsers(ctx, 100, "open_id", "")
	if err != nil {
		return "", err
	}
	for _, user := range users.Items {
		if strings.EqualFold(user.Email, email) {
			return firstNonEmpty(user.OpenID, user.ID, user.UserID), nil
		}
	}
	return "", nil
}

func (c *FeishuChannel) GetUserIDByMobile(ctx context.Context, mobile string) (string, error) {
	query := map[string]string{"mobile_phone": mobile, "user_id_type": "open_id", "page_size": "1"}
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, "/open-apis/contact/v3/users", query, &payload); err != nil {
		return "", err
	}
	if payload.Code != 0 {
		return "", fmt.Errorf("feishu get user by mobile api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	if len(payload.Data.Items) == 0 {
		return "", nil
	}
	user := normalizeUserSummary(payload.Data.Items[0])
	return firstNonEmpty(user.OpenID, user.ID, user.UserID), nil
}

func (c *FeishuChannel) CreateGroup(ctx context.Context, name string) (*ChatSummary, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("group name is empty")
	}
	body := map[string]any{"name": name, "member_type": "ADMIN"}
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Chat map[string]any `json:"chat"`
		} `json:"data"`
	}
	if err := c.postJSON(ctx, "/open-apis/im/v1/chats", body, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu create group api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	chat := normalizeChatSummary(payload.Data.Chat)
	return &chat, nil
}

func (c *FeishuChannel) GetGroupInfo(ctx context.Context, chatID string) (*ChatSummary, error) {
	if strings.TrimSpace(chatID) == "" {
		return nil, fmt.Errorf("chat ID is empty")
	}
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Chat map[string]any `json:"chat"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, "/open-apis/im/v1/chats/"+url.PathEscape(chatID), nil, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu get chat api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	chat := normalizeChatSummary(payload.Data.Chat)
	return &chat, nil
}

func (c *FeishuChannel) ListGroupMembers(ctx context.Context, chatID string, pageSize int, pageToken string) (*UserList, error) {
	if strings.TrimSpace(chatID) == "" {
		return nil, fmt.Errorf("chat ID is empty")
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	query := map[string]string{"page_size": fmt.Sprintf("%d", pageSize)}
	if pageToken != "" {
		query["page_token"] = pageToken
	}
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			HasMore   bool             `json:"has_more"`
			PageToken string           `json:"page_token"`
			Items     []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, "/open-apis/im/v1/chats/"+url.PathEscape(chatID)+"/members", query, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu list chat members api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	result := &UserList{HasMore: payload.Data.HasMore, PageToken: payload.Data.PageToken}
	for _, item := range payload.Data.Items {
		result.Items = append(result.Items, normalizeUserSummary(item))
	}
	return result, nil
}

func (c *FeishuChannel) ListGroups(ctx context.Context, pageSize int, pageToken string) (*ChatList, error) {
	if pageSize <= 0 {
		pageSize = 50
	}
	query := map[string]string{"page_size": fmt.Sprintf("%d", pageSize)}
	if pageToken != "" {
		query["page_token"] = pageToken
	}
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			HasMore   bool             `json:"has_more"`
			PageToken string           `json:"page_token"`
			Items     []map[string]any `json:"items"`
			Data      struct {
				HasMore   bool             `json:"has_more"`
				PageToken string           `json:"page_token"`
				Items     []map[string]any `json:"items"`
			} `json:"data"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, "/open-apis/im/v1/chats", query, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu list chats api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	items := payload.Data.Items
	hasMore := payload.Data.HasMore
	nextToken := payload.Data.PageToken
	if len(items) == 0 && len(payload.Data.Data.Items) > 0 {
		items = payload.Data.Data.Items
		hasMore = payload.Data.Data.HasMore
		nextToken = payload.Data.Data.PageToken
	}
	result := &ChatList{HasMore: hasMore, PageToken: nextToken}
	for _, item := range items {
		result.Items = append(result.Items, normalizeChatSummary(item))
	}
	return result, nil
}

func (c *FeishuChannel) SendGroupMessage(ctx context.Context, chatID, text string) error {
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeText).
			Content(string(mustJSONMarshal(map[string]string{"text": text}))).
			Build()).
		Build()
	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu send group message: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("feishu send group message api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}
	return nil
}

func (c *FeishuChannel) GetDriveRootFolder(ctx context.Context) (*DriveFolderSummary, error) {
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data map[string]any `json:"data"`
	}
	if err := c.getJSON(ctx, "/open-apis/drive/explorer/v2/root_folder/meta", nil, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu drive root folder api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	folder := normalizeDriveFolder(payload.Data)
	return &folder, nil
}

func (c *FeishuChannel) GetDriveFolder(ctx context.Context, folderToken string) (*DriveFolderSummary, error) {
	if strings.TrimSpace(folderToken) == "" {
		return nil, fmt.Errorf("folder token is empty")
	}
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data map[string]any `json:"data"`
	}
	path := "/open-apis/drive/explorer/v2/folder/" + url.PathEscape(folderToken) + "/meta"
	if err := c.getJSON(ctx, path, nil, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu get drive folder api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	folder := normalizeDriveFolder(payload.Data)
	return &folder, nil
}

func (c *FeishuChannel) GetDriveFile(ctx context.Context, fileToken string) (*DriveFileSummary, error) {
	if strings.TrimSpace(fileToken) == "" {
		return nil, fmt.Errorf("file token is empty")
	}
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			File map[string]any `json:"file"`
		} `json:"data"`
	}
	path := "/open-apis/drive/v1/files/" + url.PathEscape(fileToken)
	if err := c.getJSON(ctx, path, nil, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu get drive file api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	file := normalizeDriveFile(payload.Data.File)
	return &file, nil
}

func (c *FeishuChannel) ListDriveFiles(ctx context.Context, folderToken, pageToken string, pageSize int) (*DriveFileList, error) {
	query := map[string]string{}
	if folderToken != "" {
		query["folder_token"] = folderToken
	}
	if pageToken != "" {
		query["page_token"] = pageToken
	}
	if pageSize > 0 {
		query["page_size"] = fmt.Sprintf("%d", pageSize)
	}
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			HasMore   bool             `json:"has_more"`
			PageToken string           `json:"page_token"`
			Files     []map[string]any `json:"files"`
			Items     []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, "/open-apis/drive/v1/files", query, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu list drive files api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	items := payload.Data.Files
	if len(items) == 0 {
		items = payload.Data.Items
	}
	result := &DriveFileList{HasMore: payload.Data.HasMore, PageToken: payload.Data.PageToken}
	for _, item := range items {
		result.Items = append(result.Items, normalizeDriveFile(item))
	}
	return result, nil
}

func (c *FeishuChannel) DownloadDriveFile(ctx context.Context, fileToken string) (*DownloadedFile, error) {
	if strings.TrimSpace(fileToken) == "" {
		return nil, fmt.Errorf("file token is empty")
	}
	resp, err := c.client.Do(ctx, &larkcore.ApiReq{HttpMethod: http.MethodGet, ApiPath: "/open-apis/drive/v1/files/" + url.PathEscape(fileToken) + "/download", SupportedAccessTokenTypes: []larkcore.AccessTokenType{larkcore.AccessTokenTypeTenant}})
	if err != nil {
		return nil, err
	}
	contentType := ""
	if resp.Header != nil {
		contentType = resp.Header.Get("Content-Type")
	}
	name := fileToken
	if resp.Header != nil {
		if headerName := filenameFromHeader(resp.Header); headerName != "" {
			name = headerName
		}
	}
	return &DownloadedFile{Name: name, ContentType: contentType, Data: resp.RawBody}, nil
}

func (c *FeishuChannel) DeleteDriveFile(ctx context.Context, fileToken string) error {
	if strings.TrimSpace(fileToken) == "" {
		return fmt.Errorf("file token is empty")
	}
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := c.doJSON(ctx, http.MethodDelete, "/open-apis/drive/v1/files/"+url.PathEscape(fileToken), nil, nil, &payload); err != nil {
		return err
	}
	if payload.Code != 0 {
		return fmt.Errorf("feishu delete drive file api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	return nil
}

func (c *FeishuChannel) UploadDriveFile(ctx context.Context, parentToken, name string, r io.Reader) (*DriveFileSummary, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("file name is empty")
	}
	body, contentType, err := buildDriveUploadBody(parentToken, name, r)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			File map[string]any `json:"file"`
		} `json:"data"`
	}
	if err := c.doMultipart(ctx, http.MethodPost, "/open-apis/drive/v1/files/upload_all", body.Bytes(), contentType, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu upload drive file api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	file := normalizeDriveFile(payload.Data.File)
	return &file, nil
}

func (c *FeishuChannel) InitiateMultipartUpload(ctx context.Context, parentToken, name string, size int64) (*MultipartUploadSession, error) {
	body := map[string]any{"file_name": name, "size": size}
	if parentToken != "" {
		body["parent_type"] = "explorer"
		body["parent_node"] = parentToken
	}
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			FileToken string `json:"file_token"`
			UploadID  string `json:"upload_id"`
			BlockSize int    `json:"block_size"`
		} `json:"data"`
	}
	if err := c.postJSON(ctx, "/open-apis/drive/v1/files/upload_prepare", body, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu initiate multipart upload api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	return &MultipartUploadSession{FileToken: payload.Data.FileToken, UploadID: payload.Data.UploadID, BlockSize: payload.Data.BlockSize}, nil
}

func (c *FeishuChannel) UploadMultipartChunk(ctx context.Context, uploadID string, seq int, data []byte) error {
	if strings.TrimSpace(uploadID) == "" {
		return fmt.Errorf("upload ID is empty")
	}
	body, contentType, err := buildMultipartChunkBody(seq, data)
	if err != nil {
		return err
	}
	path := "/open-apis/drive/v1/files/upload_part?upload_id=" + url.QueryEscape(uploadID)
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := c.doMultipart(ctx, http.MethodPost, path, body.Bytes(), contentType, &payload); err != nil {
		return err
	}
	if payload.Code != 0 {
		return fmt.Errorf("feishu upload multipart chunk api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	return nil
}

func (c *FeishuChannel) CompleteMultipartUpload(ctx context.Context, uploadID string, blockNum int) (*DriveFileSummary, error) {
	body := map[string]any{"upload_id": uploadID, "block_num": blockNum}
	var payload struct {
		Code int `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			File map[string]any `json:"file"`
		} `json:"data"`
	}
	if err := c.postJSON(ctx, "/open-apis/drive/v1/files/upload_finish", body, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("feishu complete multipart upload api error (code=%d msg=%s)", payload.Code, payload.Msg)
	}
	file := normalizeDriveFile(payload.Data.File)
	return &file, nil
}

func (c *FeishuChannel) getJSON(ctx context.Context, apiPath string, query map[string]string, out any) error {
	return c.doJSON(ctx, http.MethodGet, apiPath, query, nil, out)
}

func (c *FeishuChannel) postJSON(ctx context.Context, apiPath string, body any, out any) error {
	return c.doJSON(ctx, http.MethodPost, apiPath, nil, body, out)
}

func (c *FeishuChannel) doJSON(ctx context.Context, method, apiPath string, query map[string]string, body any, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(query) > 0 {
		values := url.Values{}
		for k, v := range query {
			if v != "" {
				values.Set(k, v)
			}
		}
		if encoded := values.Encode(); encoded != "" {
			if strings.Contains(apiPath, "?") {
				apiPath += "&" + encoded
			} else {
				apiPath += "?" + encoded
			}
		}
	}
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	resp, err := c.client.Do(ctx, &larkcore.ApiReq{
		HttpMethod:                method,
		ApiPath:                   apiPath,
		Body:                      payload,
		SupportedAccessTokenTypes: []larkcore.AccessTokenType{larkcore.AccessTokenTypeTenant},
	})
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(resp.RawBody, out)
}

func (c *FeishuChannel) doMultipart(ctx context.Context, method, apiPath string, body []byte, contentType string, out any) error {
	_ = contentType
	resp, err := c.client.Do(ctx, &larkcore.ApiReq{
		HttpMethod:                method,
		ApiPath:                   apiPath,
		Body:                      body,
		SupportedAccessTokenTypes: []larkcore.AccessTokenType{larkcore.AccessTokenTypeTenant},
	})
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(resp.RawBody, out)
}

func mustJSONMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func (c *FeishuChannel) downloadToPath(ctx context.Context, apiPath string) (*DownloadedFile, error) {
	resp, err := c.client.Do(ctx, &larkcore.ApiReq{HttpMethod: http.MethodGet, ApiPath: apiPath, SupportedAccessTokenTypes: []larkcore.AccessTokenType{larkcore.AccessTokenTypeTenant}})
	if err != nil {
		return nil, err
	}
	name := "download.bin"
	if resp.Header != nil {
		if fileName := filenameFromHeader(resp.Header); fileName != "" {
			name = fileName
		}
	}
	contentType := ""
	if resp.Header != nil {
		contentType = resp.Header.Get("Content-Type")
	}
	return &DownloadedFile{Name: name, ContentType: contentType, Data: append([]byte(nil), resp.RawBody...)}, nil
}

func bytesReader(data []byte) io.Reader { return bytes.NewReader(data) }

