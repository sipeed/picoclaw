package mcp

import (
	"encoding/json"
	"strings"
)

type callResponse struct {
	Content           []contentBlock `json:"content"`
	StructuredContent any            `json:"structuredContent,omitempty"`
	IsError           bool           `json:"isError,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func formatCallPayload(raw json.RawMessage, responseLimit int) (CallResult, error) {
	var payload callResponse
	if err := json.Unmarshal(raw, &payload); err != nil {
		// Fallback for servers that return non-standard payloads.
		return CallResult{
			Content: truncateString(strings.TrimSpace(string(raw)), responseLimit),
			IsError: false,
		}, nil
	}

	parts := make([]string, 0, len(payload.Content)+1)
	for _, block := range payload.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			parts = append(parts, block.Text)
		}
	}

	if payload.StructuredContent != nil {
		if encoded, err := json.Marshal(payload.StructuredContent); err == nil {
			parts = append(parts, string(encoded))
		}
	}

	content := strings.TrimSpace(strings.Join(parts, "\n"))
	if content == "" {
		content = "{}"
	}

	return CallResult{
		Content: truncateString(content, responseLimit),
		IsError: payload.IsError,
	}, nil
}

func truncateString(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	if maxBytes <= 12 {
		return value[:maxBytes]
	}
	return value[:maxBytes-12] + "\n...[truncated]"
}
