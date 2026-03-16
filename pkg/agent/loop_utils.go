// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"jane/pkg/bus"
	"jane/pkg/providers"
	"jane/pkg/routing"
	"jane/pkg/utils"
)

// formatMessagesForLog formats messages for logging
func formatMessagesForLog(messages []providers.Message) string {
	if len(messages) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, msg := range messages {
		sb.WriteString("  [")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("] Role: ")
		sb.WriteString(msg.Role)
		sb.WriteString("\n")

		if len(msg.ToolCalls) > 0 {
			sb.WriteString("  ToolCalls:\n")
			for _, tc := range msg.ToolCalls {
				sb.WriteString("    - ID: ")
				sb.WriteString(tc.ID)
				sb.WriteString(", Type: ")
				sb.WriteString(tc.Type)
				sb.WriteString(", Name: ")
				sb.WriteString(tc.Name)
				sb.WriteString("\n")

				if tc.Function != nil {
					sb.WriteString("      Arguments: ")
					sb.WriteString(utils.Truncate(tc.Function.Arguments, 200))
					sb.WriteString("\n")
				}
			}
		}
		if msg.Content != "" {
			content := utils.Truncate(msg.Content, 200)
			sb.WriteString("  Content: ")
			sb.WriteString(content)
			sb.WriteString("\n")
		}
		if msg.ToolCallID != "" {
			sb.WriteString("  ToolCallID: ")
			sb.WriteString(msg.ToolCallID)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("]")
	return sb.String()
}

// formatToolsForLog formats tool definitions for logging
func formatToolsForLog(toolDefs []providers.ToolDefinition) string {
	if len(toolDefs) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, tool := range toolDefs {
		sb.WriteString("  [")
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("] Type: ")
		sb.WriteString(tool.Type)
		sb.WriteString(", Name: ")
		sb.WriteString(tool.Function.Name)
		sb.WriteString("\n")

		sb.WriteString("      Description: ")
		sb.WriteString(tool.Function.Description)
		sb.WriteString("\n")

		if len(tool.Function.Parameters) > 0 {
			sb.WriteString("      Parameters: ")
			sb.WriteString(utils.Truncate(fmt.Sprintf("%v", tool.Function.Parameters), 200))
			sb.WriteString("\n")
		}
	}
	sb.WriteString("]")
	return sb.String()
}

// inferMediaType determines the media type ("image", "audio", "video", "file")
// from a filename and MIME content type.
func inferMediaType(filename, contentType string) string {
	ct := strings.ToLower(contentType)
	fn := strings.ToLower(filename)

	if strings.HasPrefix(ct, "image/") {
		return "image"
	}
	if strings.HasPrefix(ct, "audio/") || ct == "application/ogg" {
		return "audio"
	}
	if strings.HasPrefix(ct, "video/") {
		return "video"
	}

	// Fallback: infer from extension
	ext := filepath.Ext(fn)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return "image"
	case ".mp3", ".wav", ".ogg", ".m4a", ".flac", ".aac", ".wma", ".opus":
		return "audio"
	case ".mp4", ".avi", ".mov", ".webm", ".mkv":
		return "video"
	}

	return "file"
}

// extractPeer extracts the routing peer from the inbound message's structured Peer field.
func extractPeer(msg bus.InboundMessage) *routing.RoutePeer {
	if msg.Peer.Kind == "" {
		return nil
	}
	peerID := msg.Peer.ID
	if peerID == "" {
		if msg.Peer.Kind == "direct" {
			peerID = msg.SenderID
		} else {
			peerID = msg.ChatID
		}
	}
	return &routing.RoutePeer{Kind: msg.Peer.Kind, ID: peerID}
}

func inboundMetadata(msg bus.InboundMessage, key string) string {
	if msg.Metadata == nil {
		return ""
	}
	return msg.Metadata[key]
}

// extractParentPeer extracts the parent peer (reply-to) from inbound message metadata.
func extractParentPeer(msg bus.InboundMessage) *routing.RoutePeer {
	parentKind := inboundMetadata(msg, metadataKeyParentPeerKind)
	parentID := inboundMetadata(msg, metadataKeyParentPeerID)
	if parentKind == "" || parentID == "" {
		return nil
	}
	return &routing.RoutePeer{Kind: parentKind, ID: parentID}
}
