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
	all := extractAllJSONStringFields(content, field)
	if len(all) == 0 {
		return ""
	}
	return all[0]
}

func extractAllJSONStringFields(content, field string) []string {
	var decoded any
	if err := json.Unmarshal([]byte(content), &decoded); err != nil {
		return nil
	}
	return collectJSONStringFields(decoded, field)
}

func collectJSONStringFields(v any, field string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0)
	var walk func(any)
	walk = func(node any) {
		switch vv := node.(type) {
		case map[string]any:
			if value, ok := vv[field].(string); ok {
				value = strings.TrimSpace(value)
				if value != "" {
					if _, exists := seen[value]; !exists {
						seen[value] = struct{}{}
						result = append(result, value)
					}
				}
			}
			for _, nested := range vv {
				walk(nested)
			}
		case []any:
			for _, nested := range vv {
				walk(nested)
			}
		}
	}
	walk(v)
	return result
}

// extractImageKey extracts the first image_key from a Feishu image message content JSON.
func extractImageKey(content string) string { return extractJSONStringField(content, "image_key") }

func extractImageKeys(content string) []string { return extractAllJSONStringFields(content, "image_key") }

// extractFileKey extracts the first file_key from a Feishu file/audio message content JSON.
func extractFileKey(content string) string { return extractJSONStringField(content, "file_key") }

func extractFileKeys(content string) []string { return extractAllJSONStringFields(content, "file_key") }

// extractFileName extracts the file_name from a Feishu file message content JSON.
func extractFileName(content string) string { return extractJSONStringField(content, "file_name") }

func extractFileNames(content string) []string { return extractAllJSONStringFields(content, "file_name") }

// stripMentionPlaceholders removes @_user_N placeholders from the text content.
// These are inserted by Feishu when users @mention someone in a message.
func stripMentionPlaceholders(content string, mentions []*larkim.MentionEvent) string {
	if len(mentions) == 0 {
		return content
	}
	for _, m := range mentions {
		if m.Key != nil && *m.Key != "" {
			content = strings.ReplaceAll(content, *m.Key, "")
		}
	}
	// Also clean up any remaining @_user_N patterns
	content = mentionPlaceholderRegex.ReplaceAllString(content, "")
	return strings.TrimSpace(content)
}
