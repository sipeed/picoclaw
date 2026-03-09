package feishu

import (
	"encoding/json"
	"regexp"
	"strings"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// mentionPlaceholderRegex matches @_user_N placeholders inserted by Feishu for mentions.
var mentionPlaceholderRegex = regexp.MustCompile(`@_user_\d+`)

// stringValue safely dereferences a *string pointer.
func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// buildMarkdownCard builds a Feishu Interactive Card JSON 2.0 string with markdown content.
// JSON 2.0 cards support full CommonMark standard markdown syntax.
func buildMarkdownCard(content string) (string, error) {
	card := map[string]any{
		"schema": "2.0",
		"body": map[string]any{
			"elements": []map[string]any{
				{
					"tag":     "markdown",
					"content": content,
				},
			},
		},
	}
	data, err := json.Marshal(card)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// extractJSONStringField unmarshals content as JSON and returns the value of the given string field.
// Returns "" if the content is invalid JSON or the field is missing/empty.
func extractJSONStringField(content, field string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &m); err != nil {
		return ""
	}
	raw, ok := m[field]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

// extractImageKey extracts the image_key from a Feishu image message content JSON.
// Format: {"image_key": "img_xxx"}
func extractImageKey(content string) string { return extractJSONStringField(content, "image_key") }

// extractFileKey extracts the file_key from a Feishu file/audio message content JSON.
// Format: {"file_key": "file_xxx", "file_name": "...", ...}
func extractFileKey(content string) string { return extractJSONStringField(content, "file_key") }

// extractFileName extracts the file_name from a Feishu file message content JSON.
func extractFileName(content string) string { return extractJSONStringField(content, "file_name") }

// stripMentionPlaceholders replaces @_user_N placeholders in the text content.
// Bot mentions are removed; other user mentions are replaced with @Name(open_id:xxx)
// so downstream tools can still extract the user ID.
func stripMentionPlaceholders(content string, mentions []*larkim.MentionEvent, botOpenID string) string {
	if len(mentions) == 0 {
		return content
	}
	for _, m := range mentions {
		if m.Key == nil || *m.Key == "" {
			continue
		}
		// If this mention is the bot itself, strip it.
		if botOpenID != "" && m.Id != nil && m.Id.OpenId != nil && *m.Id.OpenId == botOpenID {
			content = strings.ReplaceAll(content, *m.Key, "")
			continue
		}
		// Replace with @Name(open_id:xxx) to preserve identity for downstream use.
		replacement := ""
		name := ""
		if m.Name != nil {
			name = *m.Name
		}
		openID := ""
		if m.Id != nil && m.Id.OpenId != nil {
			openID = *m.Id.OpenId
		}
		if name != "" && openID != "" {
			replacement = "@" + name + "(open_id:" + openID + ")"
		} else if name != "" {
			replacement = "@" + name
		} else if openID != "" {
			replacement = "@user(open_id:" + openID + ")"
		}
		content = strings.ReplaceAll(content, *m.Key, replacement)
	}
	// Clean up any remaining @_user_N patterns not covered by mentions list
	content = mentionPlaceholderRegex.ReplaceAllString(content, "")
	return strings.TrimSpace(content)
}
