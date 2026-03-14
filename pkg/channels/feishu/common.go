package feishu

import (
	"encoding/json"
	"regexp"
	"strings"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// maxTablesPerMessage is the maximum number of tables allowed per message
const maxTablesPerMessage = 5

// mentionPlaceholderRegex matches @_user_N placeholders inserted by Feishu for mentions.
var mentionPlaceholderRegex = regexp.MustCompile(`@_user_\d+`)

// isTableSeparator checks if a line is a markdown table separator row (|---|---| etc.)
func isTableSeparator(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || !strings.HasPrefix(trimmed, "|") || !strings.HasSuffix(trimmed, "|") {
		return false
	}
	// Remove leading and trailing |
	trimmed = strings.Trim(trimmed, "|")

	// Check if all remaining characters are -, |, : or space
	hasDash := false
	for _, c := range trimmed {
		if c != '-' && c != '|' && c != ' ' && c != ':' {
			return false
		}
		if c == '-' {
			hasDash = true
		}
	}

	// A separator must have at least one dash character
	return hasDash
}

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

// splitContentByTableCount splits the content into multiple parts if it contains too many tables.
// Each part will have at most maxTablesPerMessage tables.
func splitContentByTableCount(content string) []string {
	var parts []string
	lines := strings.Split(content, "\n")

	var currentPart strings.Builder
	var currentTableCount int
	var inTable bool

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this line starts with | and is not a separator
		if strings.HasPrefix(trimmed, "|") && !isTableSeparator(trimmed) {
			// Check if next line is a separator to confirm it's a table header
			isTableHeader := i+1 < len(lines) && isTableSeparator(strings.TrimSpace(lines[i+1]))

			if isTableHeader {
				// Starting a new table (inTable may already be true for consecutive tables)
				inTable = true
				// Check if we need to start a new part
				if currentTableCount >= maxTablesPerMessage && currentPart.Len() > 0 {
					parts = append(parts, strings.TrimSpace(currentPart.String()))
					currentPart.Reset()
					currentTableCount = 0
				}
				currentTableCount++
			}
		} else if inTable && !strings.HasPrefix(trimmed, "|") {
			// Non-table line ends a table
			inTable = false
		}

		currentPart.WriteString(line)
		currentPart.WriteString("\n")
	}

	// Add the last part
	if currentPart.Len() > 0 {
		parts = append(parts, strings.TrimSpace(currentPart.String()))
	}

	return parts
}

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
