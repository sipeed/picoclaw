package feishu

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

func parseMessageContentPayload(messageData map[string]any) *ParsedMessage {
	if len(messageData) == 0 {
		return nil
	}

	items, _ := messageData["items"].([]any)
	if len(items) == 0 {
		if msgMap, ok := messageData["message"].(map[string]any); ok {
			return parseMessageItem(msgMap)
		}
		return parseMessageItem(messageData)
	}
	first, _ := items[0].(map[string]any)
	return parseMessageItem(first)
}

func parseMessageItem(item map[string]any) *ParsedMessage {
	if len(item) == 0 {
		return nil
	}
	body, _ := item["body"].(map[string]any)
	contentStr, _ := body["content"].(string)
	var content any
	if contentStr != "" {
		if err := json.Unmarshal([]byte(contentStr), &content); err != nil {
			content = contentStr
		}
	}
	return &ParsedMessage{
		MessageID: stringFromAny(item["message_id"]),
		ChatID:    stringFromAny(item["chat_id"]),
		MsgType:   stringFromAny(item["msg_type"]),
		CreateTime: stringFromAny(item["create_time"]),
		Sender:    mapFromAny(item["sender"]),
		Content:   content,
	}
}

func parseCardSummary(card any) *CardSummary {
	cardMap, ok := card.(map[string]any)
	if !ok || len(cardMap) == 0 {
		return nil
	}

	summary := &CardSummary{}
	if title := extractCardTitle(cardMap); title != "" {
		summary.Title = title
	}

	components := extractCardComponents(cardMap)
	for i, component := range components {
		compMap, _ := component.(map[string]any)
		if len(compMap) == 0 {
			continue
		}
		parsed := CardComponent{Index: i + 1, Tag: stringFromAny(compMap["tag"]), Details: map[string]any{}}
		tag := parsed.Tag
		switch tag {
		case "img":
			parsed.Details = map[string]any{
				"image_key": stringFromAny(compMap["image_key"]),
				"width":     compMap["width"],
				"height":    compMap["height"],
				"alt":       compMap["alt"],
			}
			if key := stringFromAny(compMap["image_key"]); key != "" {
				summary.ImageKeys = append(summary.ImageKeys, key)
			}
		case "text", "note", "header":
			text := extractTextContent(compMap["text"])
			parsed.Details = map[string]any{"text": text, "style": compMap["style"], "mode": compMap["mode"]}
			if text != "" {
				summary.TextContents = append(summary.TextContents, text)
			}
		case "div":
			text := extractTextContent(compMap["text"])
			parsed.Details = map[string]any{
				"text":   text,
				"style":  compMap["style"],
				"mode":   compMap["mode"],
				"fields": compMap["fields"],
			}
			if text != "" {
				summary.TextContents = append(summary.TextContents, text)
			}
		case "action":
			actions, _ := compMap["actions"].([]any)
			parsedActions := make([]map[string]any, 0, len(actions))
			for _, rawAction := range actions {
				action, _ := rawAction.(map[string]any)
				btnText := extractTextContent(action["text"])
				parsedActions = append(parsedActions, map[string]any{
					"text":  btnText,
					"type":  stringFromAny(action["type"]),
					"style": mapFromAny(action["style"]),
				})
				if btnText != "" {
					summary.ActionButtons = append(summary.ActionButtons, btnText)
				}
			}
			parsed.Details = map[string]any{"action_count": len(actions), "actions": parsedActions}
		case "at":
			parsed.Details = map[string]any{
				"user_id":     stringFromAny(compMap["user_id"]),
				"user_name":   stringFromAny(compMap["user_name"]),
				"user_avatar": stringFromAny(compMap["user_avatar"]),
				"style":       compMap["style"],
			}
		default:
			parsed.Details = compMap
		}
		summary.Components = append(summary.Components, parsed)
	}

	return summary
}

func extractCardTitle(card map[string]any) string {
	if title := stringFromAny(card["title"]); title != "" {
		return title
	}
	if header, ok := card["header"].(map[string]any); ok {
		if titleMap, ok := header["title"].(map[string]any); ok {
			if text := stringFromAny(titleMap["content"]); text != "" {
				return text
			}
		}
	}
	return ""
}

func extractCardComponents(card map[string]any) []any {
	if body, ok := card["body"].(map[string]any); ok {
		if elements, ok := body["elements"].([]any); ok {
			return elements
		}
	}
	if elements, ok := card["elements"].([]any); ok {
		return elements
	}
	if content, ok := card["content"].([]any); ok {
		return content
	}
	return nil
}

func extractTextContent(v any) string {
	switch vv := v.(type) {
	case string:
		return vv
	case map[string]any:
		if s := stringFromAny(vv["content"]); s != "" {
			return s
		}
		if s := stringFromAny(vv["text"]); s != "" {
			return s
		}
	}
	return ""
}

func extractShareLinkToken(shareLink string) string {
	u, err := url.Parse(shareLink)
	if err != nil {
		return ""
	}
	return u.Query().Get("token")
}

func mapFromAny(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func stringFromAny(v any) string {
	switch vv := v.(type) {
	case string:
		return vv
	case *string:
		if vv != nil {
			return *vv
		}
	case json.Number:
		return vv.String()
	case float64:
		return strconv.FormatFloat(vv, 'f', -1, 64)
	case int:
		return strconv.Itoa(vv)
	case int64:
		return strconv.FormatInt(vv, 10)
	case bool:
		return strconv.FormatBool(vv)
	}
	return ""
}

func boolFromAny(v any) bool {
	switch vv := v.(type) {
	case bool:
		return vv
	case string:
		return strings.EqualFold(vv, "true")
	}
	return false
}

func int64FromAny(v any) int64 {
	switch vv := v.(type) {
	case int:
		return int64(vv)
	case int64:
		return vv
	case float64:
		return int64(vv)
	case json.Number:
		n, _ := vv.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(vv), 10, 64)
		return n
	}
	return 0
}

func normalizeMessageDetail(item map[string]any) MessageDetail {
	detail := MessageDetail{
		MessageID:  stringFromAny(item["message_id"]),
		ChatID:     stringFromAny(item["chat_id"]),
		RootID:     stringFromAny(item["root_id"]),
		ParentID:   stringFromAny(item["parent_id"]),
		MsgType:    stringFromAny(item["msg_type"]),
		Deleted:    boolFromAny(item["deleted"]),
		Updated:    boolFromAny(item["updated"]),
		CreateTime: stringFromAny(item["create_time"]),
		UpdateTime: stringFromAny(item["update_time"]),
		Sender:     mapFromAny(item["sender"]),
		Body:       mapFromAny(item["body"]),
		Raw:        item,
	}
	if mentions, ok := item["mentions"].([]any); ok {
		for _, mention := range mentions {
			if m, ok := mention.(map[string]any); ok {
				detail.Mentions = append(detail.Mentions, m)
			}
		}
	}
	if parsed := parseMessageItem(item); parsed != nil {
		detail.Parsed = parsed
		if parsed.MsgType == larkInteractiveMsgType() {
			detail.CardParsed = parseCardSummary(parsed.Content)
		}
	}
	return detail
}

func larkInteractiveMsgType() string { return "interactive" }

func normalizeDriveFile(item map[string]any) DriveFileSummary {
	return DriveFileSummary{
		FileToken:    firstNonEmpty(stringFromAny(item["file_token"]), stringFromAny(item["token"]), stringFromAny(item["file_key"])),
		Name:         firstNonEmpty(stringFromAny(item["name"]), stringFromAny(item["file_name"])),
		Type:         stringFromAny(item["type"]),
		ParentToken:  firstNonEmpty(stringFromAny(item["parent_token"]), stringFromAny(item["parent_node"])),
		Size:         int64FromAny(item["size"]),
		Extension:    firstNonEmpty(stringFromAny(item["extension"]), stringFromAny(item["file_extension"])),
		MimeType:     firstNonEmpty(stringFromAny(item["mime_type"]), stringFromAny(item["content_type"])),
		URL:          firstNonEmpty(stringFromAny(item["url"]), stringFromAny(item["download_url"])),
		CreatedTime:  stringFromAny(item["created_time"]),
		ModifiedTime: stringFromAny(item["modified_time"]),
		Raw:          item,
	}
}

func normalizeDriveFolder(item map[string]any) DriveFolderSummary {
	return DriveFolderSummary{
		FolderToken:  firstNonEmpty(stringFromAny(item["folder_token"]), stringFromAny(item["token"])),
		Name:         firstNonEmpty(stringFromAny(item["name"]), stringFromAny(item["folder_name"])),
		ParentToken:  firstNonEmpty(stringFromAny(item["parent_token"]), stringFromAny(item["parent_node"])),
		URL:          firstNonEmpty(stringFromAny(item["url"]), stringFromAny(item["folder_url"])),
		CreatedTime:  stringFromAny(item["created_time"]),
		ModifiedTime: stringFromAny(item["modified_time"]),
		Raw:          item,
	}
}

func filenameFromHeader(h http.Header) string {
	cd := h.Get("Content-Disposition")
	if cd == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(cd)
	if err != nil {
		return ""
	}
	if v := params["filename*"]; v != "" {
		if i := strings.Index(v, "''"); i >= 0 && i+2 < len(v) {
			if decoded, err := url.QueryUnescape(v[i+2:]); err == nil {
				return decoded
			}
		}
		return v
	}
	return params["filename"]
}

func buildDriveUploadBody(parentToken, name string, r io.Reader) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("file_name", name); err != nil {
		return nil, "", err
	}
	if parentToken != "" {
		if err := w.WriteField("parent_type", "explorer"); err != nil {
			return nil, "", err
		}
		if err := w.WriteField("parent_node", parentToken); err != nil {
			return nil, "", err
		}
	}
	part, err := w.CreateFormFile("file", filepath.Base(name))
	if err != nil {
		return nil, "", err
	}
	if _, err := io.Copy(part, r); err != nil {
		return nil, "", err
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}

func buildMultipartChunkBody(seq int, data []byte) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("seq", strconv.Itoa(seq)); err != nil {
		return nil, "", err
	}
	part, err := w.CreateFormFile("file", "chunk")
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write(data); err != nil {
		return nil, "", err
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}

func normalizeUserSummary(item map[string]any) UserSummary {
	return UserSummary{
		ID:        firstNonEmpty(stringFromAny(item["id"]), stringFromAny(item["open_id"]), stringFromAny(item["user_id"])),
		OpenID:    stringFromAny(item["open_id"]),
		UserID:    stringFromAny(item["user_id"]),
		UnionID:   stringFromAny(item["union_id"]),
		Name:      stringFromAny(item["name"]),
		EnName:    stringFromAny(item["en_name"]),
		Email:     stringFromAny(item["email"]),
		Mobile:    stringFromAny(item["mobile"]),
		AvatarURL: stringFromAny(item["avatar_url"]),
		Status:    mapFromAny(item["status"]),
		Raw:       item,
	}
}

func normalizeChatSummary(item map[string]any) ChatSummary {
	ownerID := stringFromAny(item["owner_id"])
	if ownerID == "" {
		if ownerMap, ok := item["owner_id_type"].(map[string]any); ok {
			ownerID = stringFromAny(ownerMap["user_id"])
		}
	}
	return ChatSummary{
		ChatID:      firstNonEmpty(stringFromAny(item["chat_id"]), stringFromAny(item["open_chat_id"])),
		Name:        stringFromAny(item["name"]),
		Description: stringFromAny(item["description"]),
		ChatMode:    stringFromAny(item["chat_mode"]),
		ChatType:    stringFromAny(item["chat_type"]),
		OwnerID:     ownerID,
		OwnerOpenID: stringFromAny(item["owner_open_id"]),
		External:    boolFromAny(item["external"]),
		TenantKey:   stringFromAny(item["tenant_key"]),
		Avatar:      firstNonEmpty(stringFromAny(item["avatar"]), stringFromAny(item["avatar_path"])),
		Raw:         item,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
