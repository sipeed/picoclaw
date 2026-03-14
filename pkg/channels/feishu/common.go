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

// extractCardText recursively extracts all text content from a Feishu interactive card.
// It handles both JSON 1.0 (legacy) and JSON 2.0 schema formats.
func extractCardText(rawContent string) string {
	if rawContent == "" {
		return ""
	}

	var card map[string]any
	if err := json.Unmarshal([]byte(rawContent), &card); err != nil {
		return ""
	}

	var texts []string

	// Extract header title
	if header, ok := card["header"].(map[string]any); ok {
		if title := extractTextFromElement(header["title"]); title != "" {
			texts = append(texts, title)
		}
	}

	// JSON 2.0 schema: body.elements
	if body, ok := card["body"].(map[string]any); ok {
		if elements, ok := body["elements"].([]any); ok {
			for _, elem := range elements {
				extractTextFromElementsRecursive(elem, &texts)
			}
		}
	}

	// JSON 1.0 schema: elements (legacy format)
	if elements, ok := card["elements"].([]any); ok {
		for _, elem := range elements {
			extractTextFromElementsRecursive(elem, &texts)
		}
	}

	if len(texts) == 0 {
		return ""
	}
	return strings.Join(texts, "\n")
}

// extractTextFromElementsRecursive recursively traverses card elements to extract text.
func extractTextFromElementsRecursive(v any, texts *[]string) {
	switch val := v.(type) {
	case map[string]any:
		// Check for text content in common fields
		if text := extractTextFromElement(val); text != "" {
			*texts = append(*texts, text)
		}
		// Recurse into nested structures
		for key, child := range val {
			switch key {
			case "elements", "actions", "columns", "extra":
				extractTextFromElementsRecursive(child, texts)
			}
		}
	case []any:
		for _, item := range val {
			extractTextFromElementsRecursive(item, texts)
		}
	}
}

// extractTextFromElement extracts text from a single card element.
func extractTextFromElement(elem any) string {
	m, ok := elem.(map[string]any)
	if !ok {
		return ""
	}

	// Direct content field (markdown element in JSON 2.0)
	if content, ok := m["content"].(string); ok && content != "" {
		return content
	}

	// Text object with tag and content (lark_md, plain_text)
	if text, ok := m["text"].(map[string]any); ok {
		if content, ok := text["content"].(string); ok && content != "" {
			return content
		}
	}

	// Title object with tag and content
	if title, ok := m["title"].(map[string]any); ok {
		if content, ok := title["content"].(string); ok && content != "" {
			return content
		}
	}

	return ""
}

// extractCardImageKeys recursively extracts all image keys from a Feishu interactive card.
// Image keys are used to download images from Feishu API.
func extractCardImageKeys(rawContent string) []string {
	if rawContent == "" {
		return nil
	}

	var card map[string]any
	if err := json.Unmarshal([]byte(rawContent), &card); err != nil {
		return nil
	}

	var keys []string
	extractImageKeysRecursive(card, &keys)
	return keys
}

// extractImageKeysRecursive traverses card structure to find all image keys.
func extractImageKeysRecursive(v any, keys *[]string) {
	switch val := v.(type) {
	case map[string]any:
		// Check if this is an img element
		if tag, ok := val["tag"].(string); ok {
			switch tag {
			case "img":
				// Try img_key first (most common)
				if imgKey, ok := val["img_key"].(string); ok && imgKey != "" {
					*keys = append(*keys, imgKey)
				}
				// Also try src (alternative format)
				if src, ok := val["src"].(string); ok && src != "" {
					*keys = append(*keys, src)
				}
			case "icon":
				// Icon elements use icon_key
				if iconKey, ok := val["icon_key"].(string); ok && iconKey != "" {
					*keys = append(*keys, iconKey)
				}
			}
		}
		// Recurse into all nested structures
		for _, child := range val {
			extractImageKeysRecursive(child, keys)
		}
	case []any:
		for _, item := range val {
			extractImageKeysRecursive(item, keys)
		}
	}
}
