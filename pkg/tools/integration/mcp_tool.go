package integrationtools

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	runtimeevents "github.com/sipeed/picoclaw/pkg/events"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	toolshared "github.com/sipeed/picoclaw/pkg/tools/shared"
)

// MCPManager defines the interface for MCP manager operations
// This allows for easier testing with mock implementations
type MCPManager interface {
	CallTool(
		ctx context.Context,
		serverName, toolName string,
		arguments map[string]any,
	) (*mcp.CallToolResult, error)
}

// MCPTool wraps an MCP tool to implement the Tool interface
type MCPTool struct {
	manager            MCPManager
	serverName         string
	tool               *mcp.Tool
	mediaStore         media.MediaStore
	workspace          string
	maxInlineTextRunes int
	runtimeEvents      runtimeevents.Bus
}

// MCPToolCallPayload describes MCP tool execution runtime events.
type MCPToolCallPayload struct {
	Server     string `json:"server"`
	Tool       string `json:"tool"`
	DurationMS int64  `json:"duration_ms,omitempty"`
	IsError    bool   `json:"is_error,omitempty"`
	Error      string `json:"error,omitempty"`
}

// NewMCPTool creates a new MCP tool wrapper
func NewMCPTool(manager MCPManager, serverName string, tool *mcp.Tool) *MCPTool {
	return &MCPTool{
		manager:            manager,
		serverName:         serverName,
		tool:               tool,
		maxInlineTextRunes: maxMCPInlineTextRunes,
	}
}

func (t *MCPTool) SetMediaStore(store media.MediaStore) {
	t.mediaStore = store
}

func (t *MCPTool) SetWorkspace(workspace string) {
	t.workspace = strings.TrimSpace(workspace)
}

func (t *MCPTool) SetMaxInlineTextRunes(limit int) {
	if limit > 0 {
		t.maxInlineTextRunes = limit
	}
}

// SetEventPublisher injects the runtime event bus used for MCP tool observations.
func (t *MCPTool) SetEventPublisher(eventBus runtimeevents.Bus) {
	t.runtimeEvents = eventBus
}

const maxMCPInlineTextRunes = 16 * 1024

// sanitizeIdentifierComponent normalizes a string so it can be safely used
// as part of a tool/function identifier for downstream providers.
// It:
//   - lowercases the string
//   - replaces any character not in [a-z0-9_-] with '_'
//   - collapses multiple consecutive '_' into a single '_'
//   - trims leading/trailing '_'
//   - falls back to "unnamed" if the result is empty
//   - truncates overly long components to a reasonable length
func sanitizeIdentifierComponent(s string) string {
	const maxLen = 64

	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))

	prevUnderscore := false
	for _, r := range s {
		isAllowed := (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-'

		if !isAllowed {
			// Normalize any disallowed character to '_'
			if !prevUnderscore {
				b.WriteRune('_')
				prevUnderscore = true
			}
			continue
		}

		if r == '_' {
			if prevUnderscore {
				continue
			}
			prevUnderscore = true
		} else {
			prevUnderscore = false
		}

		b.WriteRune(r)
	}

	result := strings.Trim(b.String(), "_")
	if result == "" {
		result = "unnamed"
	}

	if len(result) > maxLen {
		result = result[:maxLen]
	}

	return result
}

// Name returns the tool name, prefixed with the server name.
// The total length is capped at 64 characters (OpenAI-compatible API limit).
// A short hash of the original (unsanitized) server and tool names is appended
// whenever sanitization is lossy or the name is truncated, ensuring that two
// names which differ only in disallowed characters remain distinct after sanitization.
func (t *MCPTool) Name() string {
	// Prefix with server name to avoid conflicts, and sanitize components
	sanitizedServer := sanitizeIdentifierComponent(t.serverName)
	sanitizedTool := sanitizeIdentifierComponent(t.tool.Name)
	full := fmt.Sprintf("mcp_%s_%s", sanitizedServer, sanitizedTool)

	// Check if sanitization was lossless (only lowercasing, no char replacement/truncation)
	lossless := strings.ToLower(t.serverName) == sanitizedServer &&
		strings.ToLower(t.tool.Name) == sanitizedTool

	const maxTotal = 64
	if lossless && len(full) <= maxTotal {
		return full
	}

	// Sanitization was lossy or name too long: append hash of the ORIGINAL names
	// (not the sanitized names) so different originals always yield different hashes.
	h := fnv.New32a()
	_, _ = h.Write([]byte(t.serverName + "\x00" + t.tool.Name))
	suffix := fmt.Sprintf("%08x", h.Sum32()) // 8 chars

	base := full
	if len(base) > maxTotal-9 {
		base = strings.TrimRight(full[:maxTotal-9], "_")
	}
	return base + "_" + suffix
}

// Description returns the tool description
func (t *MCPTool) Description() string {
	desc := t.tool.Description
	if desc == "" {
		desc = fmt.Sprintf("MCP tool from %s server", t.serverName)
	}
	// Add server info to description
	return fmt.Sprintf("[MCP:%s] %s", t.serverName, desc)
}

func (t *MCPTool) PromptMetadata() toolshared.PromptMetadata {
	return toolshared.PromptMetadata{
		Layer:  toolshared.ToolPromptLayerCapability,
		Slot:   toolshared.ToolPromptSlotMCP,
		Source: "mcp:" + sanitizeIdentifierComponent(t.serverName),
	}
}

// Parameters returns the tool parameters schema
func (t *MCPTool) Parameters() map[string]any {
	// The InputSchema is already a JSON Schema object
	schema := t.tool.InputSchema

	// Handle nil schema
	if schema == nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		}
	}

	// Try direct conversion first (fast path)
	if schemaMap, ok := schema.(map[string]any); ok {
		return schemaMap
	}

	// Handle json.RawMessage and []byte - unmarshal directly
	var jsonData []byte
	if rawMsg, ok := schema.(json.RawMessage); ok {
		jsonData = rawMsg
	} else if bytes, ok := schema.([]byte); ok {
		jsonData = bytes
	}

	if jsonData != nil {
		var result map[string]any
		if err := json.Unmarshal(jsonData, &result); err == nil {
			return result
		}
		// Fallback on error
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		}
	}

	// For other types (structs, etc.), convert via JSON marshal/unmarshal
	var err error
	jsonData, err = json.Marshal(schema)
	if err != nil {
		// Fallback to empty schema if marshaling fails
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		}
	}

	var result map[string]any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		// Fallback to empty schema if unmarshaling fails
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		}
	}

	return result
}

// Execute executes the MCP tool
func (t *MCPTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	startedAt := time.Now()
	t.publishRuntimeEvent(ctx, runtimeevents.KindMCPToolCallStart, startedAt, false, "")

	result, err := t.manager.CallTool(ctx, t.serverName, t.tool.Name, args)
	if err != nil {
		t.publishRuntimeEvent(ctx, runtimeevents.KindMCPToolCallEnd, startedAt, true, err.Error())
		return ErrorResult(fmt.Sprintf("MCP tool execution failed: %v", err)).WithError(err)
	}

	if result == nil {
		nilErr := fmt.Errorf("MCP tool returned nil result without error")
		t.publishRuntimeEvent(ctx, runtimeevents.KindMCPToolCallEnd, startedAt, true, nilErr.Error())
		return ErrorResult("MCP tool execution failed: nil result").WithError(nilErr)
	}

	// Handle error result from server
	if result.IsError {
		errMsg := extractContentText(result.Content)
		t.publishRuntimeEvent(ctx, runtimeevents.KindMCPToolCallEnd, startedAt, true, errMsg)
		return ErrorResult(fmt.Sprintf("MCP tool returned error: %s", errMsg)).
			WithError(fmt.Errorf("MCP tool error: %s", errMsg))
	}

	t.publishRuntimeEvent(ctx, runtimeevents.KindMCPToolCallEnd, startedAt, false, "")
	return t.normalizeResultContent(ctx, result.Content)
}

func (t *MCPTool) publishRuntimeEvent(
	ctx context.Context,
	kind runtimeevents.Kind,
	startedAt time.Time,
	isError bool,
	errMsg string,
) {
	if t == nil || t.runtimeEvents == nil {
		return
	}

	scope := runtimeevents.Scope{
		AgentID:    toolshared.ToolAgentID(ctx),
		SessionKey: toolshared.ToolSessionKey(ctx),
		Channel:    toolshared.ToolChannel(ctx),
		ChatID:     toolshared.ToolChatID(ctx),
		MessageID:  toolshared.ToolMessageID(ctx),
	}
	payload := MCPToolCallPayload{
		Server:     t.serverName,
		Tool:       t.tool.Name,
		DurationMS: time.Since(startedAt).Milliseconds(),
		IsError:    isError,
		Error:      errMsg,
	}
	severity := runtimeevents.SeverityInfo
	if isError {
		severity = runtimeevents.SeverityError
	}

	t.runtimeEvents.PublishNonBlocking(runtimeevents.Event{
		Kind:     kind,
		Source:   runtimeevents.Source{Component: "mcp", Name: t.serverName},
		Scope:    scope,
		Severity: severity,
		Payload:  payload,
		Attrs:    mcpToolCallEventAttrs(payload),
	})
}

func mcpToolCallEventAttrs(payload MCPToolCallPayload) map[string]any {
	attrs := map[string]any{
		"server":      payload.Server,
		"tool":        payload.Tool,
		"duration_ms": payload.DurationMS,
	}
	if payload.IsError {
		attrs["is_error"] = payload.IsError
	}
	if payload.Error != "" {
		attrs["error"] = payload.Error
	}
	return attrs
}

// extractContentText extracts text from MCP content array
func extractContentText(content []mcp.Content) string {
	var parts []string
	for _, c := range content {
		switch v := c.(type) {
		case *mcp.TextContent:
			parts = append(parts, sanitizeToolLLMContent(v.Text))
		case *mcp.ImageContent:
			parts = append(parts, fmt.Sprintf("[Image: %s]", normalizedMIMEType(v.MIMEType)))
		case *mcp.AudioContent:
			parts = append(parts, fmt.Sprintf("[Audio: %s]", normalizedMIMEType(v.MIMEType)))
		case *mcp.ResourceLink:
			parts = append(parts, summarizeResourceLink(v))
		case *mcp.EmbeddedResource:
			parts = append(parts, summarizeEmbeddedResource(v))
		default:
			// For other content types, use string representation
			parts = append(parts, fmt.Sprintf("[Content: %T]", v))
		}
	}
	return sanitizeToolLLMContent(strings.Join(parts, "\n"))
}

func (t *MCPTool) normalizeResultContent(ctx context.Context, content []mcp.Content) *ToolResult {
	llmParts := make([]string, 0, len(content))
	rawTextParts := make([]string, 0, len(content))
	mediaRefs := make([]string, 0, len(content))

	for _, c := range content {
		switch v := c.(type) {
		case *mcp.TextContent:
			rawText := strings.TrimSpace(v.Text)
			if rawText != "" {
				rawTextParts = append(rawTextParts, rawText)
			}
			safeText := strings.TrimSpace(sanitizeToolLLMContent(v.Text))
			if safeText != "" {
				llmParts = append(llmParts, safeText)
			}
		case *mcp.ImageContent:
			ref, note := t.storeBinaryContent(
				ctx,
				"image",
				normalizedMIMEType(v.MIMEType),
				v.Data,
				v.Annotations,
			)
			if ref != "" {
				mediaRefs = append(mediaRefs, ref)
			}
			if note != "" {
				llmParts = append(llmParts, note)
			}
		case *mcp.AudioContent:
			ref, note := t.storeBinaryContent(
				ctx,
				"audio",
				normalizedMIMEType(v.MIMEType),
				v.Data,
				v.Annotations,
			)
			if ref != "" {
				mediaRefs = append(mediaRefs, ref)
			}
			if note != "" {
				llmParts = append(llmParts, note)
			}
		case *mcp.ResourceLink:
			llmParts = append(llmParts, summarizeResourceLink(v))
		case *mcp.EmbeddedResource:
			ref, note, rawText := t.storeEmbeddedResource(ctx, v)
			if ref != "" {
				mediaRefs = append(mediaRefs, ref)
			}
			if rawText != "" {
				rawTextParts = append(rawTextParts, rawText)
			}
			if note != "" {
				llmParts = append(llmParts, note)
			}
		default:
			llmParts = append(llmParts, fmt.Sprintf("[MCP returned unsupported content type %T]", v))
		}
	}

	forLLM := strings.Join(compactStrings(llmParts), "\n")
	rawText := strings.Join(compactStrings(rawTextParts), "\n")
	if artifactResult := t.persistLargeTextArtifact(rawText); artifactResult != nil {
		artifactResult.Media = mediaRefs
		return artifactResult
	}

	result := &ToolResult{
		ForLLM: forLLM,
		Media:  mediaRefs,
	}
	return result
}

func (t *MCPTool) persistLargeTextArtifact(text string) *ToolResult {
	text = strings.TrimSpace(text)
	limit := t.maxInlineTextRunes
	if limit <= 0 {
		limit = maxMCPInlineTextRunes
	}
	size := utf8.RuneCountInString(text)
	if text == "" || size <= limit || t.workspace == "" {
		return nil
	}

	dir := filepath.Join(t.workspace, ".artifacts", "mcp")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return t.largeTextArtifactFallback(text, err)
	}
	// TODO: Add lifecycle cleanup/retention for MCP artifact files.

	pattern := fmt.Sprintf(
		"%s_%s_*.txt",
		sanitizeIdentifierComponent(t.serverName),
		sanitizeIdentifierComponent(t.tool.Name),
	)
	tmpFile, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return t.largeTextArtifactFallback(text, err)
	}
	path := tmpFile.Name()
	if _, err = tmpFile.WriteString(text); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(path)
		return t.largeTextArtifactFallback(text, err)
	}
	if err = tmpFile.Close(); err != nil {
		_ = os.Remove(path)
		return t.largeTextArtifactFallback(text, err)
	}

	return &ToolResult{
		ForLLM: fmt.Sprintf(
			"[MCP returned a large text result (%d chars); omitted from model context and saved as a local artifact.]",
			size,
		),
		ArtifactTags: []string{"[file:" + path + "]"},
	}
}

func (t *MCPTool) largeTextArtifactFallback(text string, err error) *ToolResult {
	size := utf8.RuneCountInString(text)
	logger.WarnCF("tool", "Failed to persist large MCP text artifact", map[string]any{
		"server": t.serverName,
		"tool":   t.tool.Name,
		"chars":  size,
		"error":  err.Error(),
	})
	return &ToolResult{
		ForLLM: fmt.Sprintf(
			"[MCP returned a large text result (%d chars); omitted from model context because artifact persistence failed.]",
			size,
		),
	}
}

func (t *MCPTool) storeEmbeddedResource(ctx context.Context, content *mcp.EmbeddedResource) (string, string, string) {
	if content == nil || content.Resource == nil {
		return "", "[MCP returned an embedded resource without data.]", ""
	}

	resource := content.Resource
	if len(resource.Blob) > 0 {
		ref, note := t.storeBinaryContent(
			ctx,
			"resource",
			normalizedMIMEType(resource.MIMEType),
			resource.Blob,
			content.Annotations,
		)
		return ref, note, ""
	}

	rawText := strings.TrimSpace(resource.Text)
	if rawText != "" {
		return "", sanitizeToolLLMContent(resource.Text), rawText
	}

	return "", summarizeEmbeddedResource(content), ""
}

func (t *MCPTool) storeBinaryContent(
	ctx context.Context,
	kind string,
	mimeType string,
	data []byte,
	annotations *mcp.Annotations,
) (string, string) {
	if len(data) == 0 {
		return "", fmt.Sprintf("[MCP returned %s content (%s) but it was empty.]", kind, mimeType)
	}
	if !annotationsAllowUser(annotations) {
		return "", fmt.Sprintf(
			"[MCP returned %s content (%s) for non-user audience; omitted from model context.]",
			kind,
			mimeType,
		)
	}
	if t.mediaStore == nil {
		return "", fmt.Sprintf(
			"[MCP returned %s content (%s); omitted from model context because media delivery is unavailable.]",
			kind,
			mimeType,
		)
	}

	channel := ToolChannel(ctx)
	chatID := ToolChatID(ctx)
	if channel == "" || chatID == "" {
		return "", fmt.Sprintf(
			"[MCP returned %s content (%s); omitted from model context because no target chat was available.]",
			kind,
			mimeType,
		)
	}

	dir := media.TempDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Sprintf("[MCP returned %s content (%s) but it could not be stored.]", kind, mimeType)
	}

	ext := extensionForMIMEType(mimeType)
	tmpFile, err := os.CreateTemp(dir, "mcp-*"+ext)
	if err != nil {
		return "", fmt.Sprintf("[MCP returned %s content (%s) but it could not be stored.]", kind, mimeType)
	}
	tmpPath := tmpFile.Name()
	if _, err = tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Sprintf("[MCP returned %s content (%s) but it could not be stored.]", kind, mimeType)
	}
	if err = tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Sprintf("[MCP returned %s content (%s) but it could not be stored.]", kind, mimeType)
	}

	scope := fmt.Sprintf(
		"tool:mcp:%s:%s:%s:%d",
		sanitizeIdentifierComponent(t.serverName),
		channel,
		chatID,
		time.Now().UnixNano(),
	)
	filename := fmt.Sprintf(
		"%s_%s%s",
		sanitizeIdentifierComponent(t.serverName),
		sanitizeIdentifierComponent(t.tool.Name),
		ext,
	)

	ref, err := t.mediaStore.Store(tmpPath, media.MediaMeta{
		Filename:    filename,
		ContentType: mimeType,
		Source: fmt.Sprintf(
			"tool:mcp:%s:%s",
			sanitizeIdentifierComponent(t.serverName),
			sanitizeIdentifierComponent(t.tool.Name),
		),
	}, scope)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Sprintf(
			"[MCP returned %s content (%s) but it could not be registered as media.]",
			kind,
			mimeType,
		)
	}

	return ref, fmt.Sprintf(
		"[MCP returned %s content (%s); omitted from model context and stored as a local media artifact.]",
		kind,
		mimeType,
	)
}

func summarizeResourceLink(content *mcp.ResourceLink) string {
	if content == nil {
		return "[MCP returned an empty resource link.]"
	}

	parts := []string{"[MCP returned resource link"}
	if content.Name != "" {
		parts = append(parts, fmt.Sprintf("name=%q", content.Name))
	}
	if content.URI != "" {
		parts = append(parts, fmt.Sprintf("uri=%q", content.URI))
	}
	if content.MIMEType != "" {
		parts = append(parts, fmt.Sprintf("mime=%q", content.MIMEType))
	}
	if content.Description != "" {
		desc := strings.TrimSpace(content.Description)
		if len(desc) > 200 {
			desc = desc[:200] + "..."
		}
		parts = append(parts, fmt.Sprintf("description=%q", desc))
	}
	return strings.Join(parts, ", ") + "]"
}

func summarizeEmbeddedResource(content *mcp.EmbeddedResource) string {
	if content == nil || content.Resource == nil {
		return "[MCP returned an embedded resource.]"
	}

	resource := content.Resource
	if resource.URI != "" {
		return fmt.Sprintf(
			"[MCP returned embedded resource %q (%s).]",
			resource.URI,
			normalizedMIMEType(resource.MIMEType),
		)
	}
	return fmt.Sprintf("[MCP returned embedded resource (%s).]", normalizedMIMEType(resource.MIMEType))
}

func annotationsAllowUser(annotations *mcp.Annotations) bool {
	if annotations == nil || len(annotations.Audience) == 0 {
		return true
	}
	for _, audience := range annotations.Audience {
		if strings.EqualFold(string(audience), "user") {
			return true
		}
	}
	return false
}

func normalizedMIMEType(mimeType string) string {
	if strings.TrimSpace(mimeType) == "" {
		return "application/octet-stream"
	}
	return mimeType
}

func compactStrings(parts []string) []string {
	compact := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		compact = append(compact, part)
	}
	return compact
}
