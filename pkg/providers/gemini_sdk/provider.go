package gemini_sdk

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"google.golang.org/genai"

	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

type (
	ToolCall               = protocoltypes.ToolCall
	FunctionCall           = protocoltypes.FunctionCall
	LLMResponse            = protocoltypes.LLMResponse
	UsageInfo              = protocoltypes.UsageInfo
	Message                = protocoltypes.Message
	ToolDefinition         = protocoltypes.ToolDefinition
	ToolFunctionDefinition = protocoltypes.ToolFunctionDefinition
	ExtraContent           = protocoltypes.ExtraContent
	GoogleExtra            = protocoltypes.GoogleExtra
)

const (
	defaultModel          = "gemini-2.5-flash"
	defaultRequestTimeout = 120 * time.Second
	defaultGeminiAPIBase  = "https://generativelanguage.googleapis.com"
)

var apiVersionPattern = regexp.MustCompile(`^v[0-9]+(?:(?:alpha|beta)[0-9]*)?$`)

type Provider struct {
	apiBase    string
	apiVersion string
	httpClient *http.Client
	client     *genai.Client
	initErr    error
}

type Option func(*Provider)

func WithRequestTimeout(timeout time.Duration) Option {
	return func(p *Provider) {
		if timeout > 0 {
			p.httpClient.Timeout = timeout
		}
	}
}

func NewProvider(apiKey, apiBase, proxy string, opts ...Option) *Provider {
	httpClient := &http.Client{Timeout: defaultRequestTimeout}
	if proxy != "" {
		parsed, err := url.Parse(proxy)
		if err == nil {
			httpClient.Transport = &http.Transport{Proxy: http.ProxyURL(parsed)}
		} else {
			log.Printf("gemini_sdk: invalid proxy URL %q: %v", proxy, err)
		}
	}

	baseURL, apiVersion := normalizeAPIBase(apiBase)
	p := &Provider{
		apiBase:    baseURL,
		apiVersion: apiVersion,
		httpClient: httpClient,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}

	clientConfig := &genai.ClientConfig{
		APIKey:     apiKey,
		Backend:    genai.BackendGeminiAPI,
		HTTPClient: p.httpClient,
		HTTPOptions: genai.HTTPOptions{
			BaseURL: p.apiBase,
		},
	}
	if p.apiVersion != "" {
		clientConfig.HTTPOptions.APIVersion = p.apiVersion
	}

	client, err := genai.NewClient(context.Background(), clientConfig)
	if err != nil {
		p.initErr = err
		log.Printf("gemini_sdk: failed to initialize genai client: %v", err)
		return p
	}
	p.client = client
	return p
}

func (p *Provider) GetDefaultModel() string {
	return defaultModel
}

func (p *Provider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	if p.initErr != nil {
		return nil, fmt.Errorf("failed to initialize Gemini SDK client: %w", p.initErr)
	}
	if p.client == nil {
		return nil, fmt.Errorf("Gemini SDK client not initialized")
	}

	contents, systemInstruction := buildGeminiContents(messages)
	config := &genai.GenerateContentConfig{}
	hasConfig := false
	if systemInstruction != nil {
		config.SystemInstruction = systemInstruction
		hasConfig = true
	}
	if mappedTools := buildGeminiTools(tools); len(mappedTools) > 0 {
		config.Tools = mappedTools
		hasConfig = true
	}
	if applyOptions(config, options) {
		hasConfig = true
	}
	if !hasConfig {
		config = nil
	}

	resp, err := p.client.Models.GenerateContent(ctx, normalizeModel(model), contents, config)
	if err != nil {
		var apiErr genai.APIError
		if errors.As(err, &apiErr) {
			return nil, fmt.Errorf(
				"Gemini API request failed (status=%d): %s",
				apiErr.Code,
				strings.TrimSpace(apiErr.Message),
			)
		}
		return nil, fmt.Errorf("Gemini API request failed: %w", err)
	}
	if resp == nil || len(resp.Candidates) == 0 {
		return &LLMResponse{
			Content:      "",
			FinishReason: "stop",
			Usage:        mapUsage(resp),
		}, nil
	}

	choice := resp.Candidates[0]
	content, toolCalls := parseCandidate(choice)
	return &LLMResponse{
		Content:      content,
		ToolCalls:    toolCalls,
		FinishReason: mapFinishReason(choice.FinishReason, len(toolCalls) > 0),
		Usage:        mapUsage(resp),
	}, nil
}

func normalizeAPIBase(apiBase string) (string, string) {
	base := strings.TrimRight(strings.TrimSpace(apiBase), "/")
	if base == "" {
		return defaultGeminiAPIBase, ""
	}

	parsed, err := url.Parse(base)
	if err != nil {
		return base, ""
	}

	path := strings.Trim(parsed.Path, "/")
	if path == "" {
		return strings.TrimRight(parsed.String(), "/"), ""
	}

	parts := strings.Split(path, "/")
	version := parts[len(parts)-1]
	if !apiVersionPattern.MatchString(version) {
		return strings.TrimRight(parsed.String(), "/"), ""
	}

	parts = parts[:len(parts)-1]
	if len(parts) == 0 {
		parsed.Path = ""
	} else {
		parsed.Path = "/" + strings.Join(parts, "/")
	}

	return strings.TrimRight(parsed.String(), "/"), version
}

func normalizeModel(model string) string {
	trimmed := strings.TrimSpace(model)
	lower := strings.ToLower(trimmed)
	switch {
	case strings.HasPrefix(lower, "gemini/"):
		return trimmed[len("gemini/"):]
	case strings.HasPrefix(lower, "google/"):
		return trimmed[len("google/"):]
	default:
		return trimmed
	}
}

func buildGeminiContents(messages []Message) ([]*genai.Content, *genai.Content) {
	contents := make([]*genai.Content, 0, len(messages))
	systemTexts := make([]string, 0)
	toolCallNames := make(map[string]string)

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			systemText := extractSystemText(msg)
			if systemText != "" {
				systemTexts = append(systemTexts, systemText)
			}
		case "assistant":
			modelContent := &genai.Content{Role: string(genai.RoleModel)}
			if msg.Content != "" {
				modelContent.Parts = append(modelContent.Parts, genai.NewPartFromText(msg.Content))
			}
			for _, tc := range msg.ToolCalls {
				name, args, thoughtSignature := normalizeStoredToolCall(tc)
				if name == "" {
					continue
				}
				if tc.ID != "" {
					toolCallNames[tc.ID] = name
				}
				part := genai.NewPartFromFunctionCall(name, args)
				if len(thoughtSignature) > 0 {
					part.ThoughtSignature = thoughtSignature
				}
				modelContent.Parts = append(modelContent.Parts, part)
			}
			if len(modelContent.Parts) > 0 {
				contents = append(contents, modelContent)
			}
		case "tool":
			contents = appendToolResponseContent(contents, msg.ToolCallID, msg.Content, toolCallNames)
		case "user":
			if msg.ToolCallID != "" {
				contents = appendToolResponseContent(contents, msg.ToolCallID, msg.Content, toolCallNames)
			} else if msg.Content != "" {
				contents = append(contents, genai.NewContentFromText(msg.Content, genai.RoleUser))
			}
		default:
			if msg.Content != "" {
				contents = append(contents, genai.NewContentFromText(msg.Content, genai.RoleUser))
			}
		}
	}

	var systemInstruction *genai.Content
	if len(systemTexts) > 0 {
		systemInstruction = &genai.Content{
			Parts: []*genai.Part{
				genai.NewPartFromText(strings.Join(systemTexts, "\n\n")),
			},
		}
	}

	return contents, systemInstruction
}

func extractSystemText(msg Message) string {
	if strings.TrimSpace(msg.Content) != "" {
		return msg.Content
	}
	if len(msg.SystemParts) == 0 {
		return ""
	}

	parts := make([]string, 0, len(msg.SystemParts))
	for _, part := range msg.SystemParts {
		if strings.TrimSpace(part.Text) == "" {
			continue
		}
		parts = append(parts, part.Text)
	}
	return strings.Join(parts, "\n\n")
}

func appendToolResponseContent(
	contents []*genai.Content,
	toolCallID string,
	content string,
	toolCallNames map[string]string,
) []*genai.Content {
	toolName := resolveToolResponseName(toolCallID, toolCallNames)
	if toolName == "" {
		return contents
	}
	resp := map[string]any{"result": content}
	contents = append(contents, genai.NewContentFromFunctionResponse(toolName, resp, genai.RoleUser))
	return contents
}

func normalizeStoredToolCall(tc ToolCall) (string, map[string]any, []byte) {
	name := tc.Name
	if name == "" && tc.Function != nil {
		name = tc.Function.Name
	}

	args := tc.Arguments
	if args == nil {
		args = map[string]any{}
	}
	if len(args) == 0 && tc.Function != nil && tc.Function.Arguments != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &parsed); err == nil && parsed != nil {
			args = parsed
		}
	}

	return name, args, decodeThoughtSignature(extractStoredThoughtSignature(tc))
}

func extractStoredThoughtSignature(tc ToolCall) string {
	if tc.ExtraContent != nil && tc.ExtraContent.Google != nil && tc.ExtraContent.Google.ThoughtSignature != "" {
		return tc.ExtraContent.Google.ThoughtSignature
	}
	if tc.Function != nil && tc.Function.ThoughtSignature != "" {
		return tc.Function.ThoughtSignature
	}
	if tc.ThoughtSignature != "" {
		return tc.ThoughtSignature
	}
	return ""
}

func decodeThoughtSignature(s string) []byte {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b
	}
	if b, err := base64.RawStdEncoding.DecodeString(s); err == nil {
		return b
	}
	return []byte(s)
}

func encodeThoughtSignature(sig []byte) string {
	if len(sig) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(sig)
}

func resolveToolResponseName(toolCallID string, toolCallNames map[string]string) string {
	if toolCallID == "" {
		return ""
	}
	if name, ok := toolCallNames[toolCallID]; ok && name != "" {
		return name
	}
	return inferToolNameFromCallID(toolCallID)
}

func inferToolNameFromCallID(toolCallID string) string {
	if !strings.HasPrefix(toolCallID, "call_") {
		return toolCallID
	}

	rest := strings.TrimPrefix(toolCallID, "call_")
	if idx := strings.LastIndex(rest, "_"); idx > 0 {
		candidate := rest[:idx]
		if candidate != "" {
			return candidate
		}
	}
	return toolCallID
}

func buildGeminiTools(tools []ToolDefinition) []*genai.Tool {
	declarations := make([]*genai.FunctionDeclaration, 0, len(tools))
	for _, tool := range tools {
		if tool.Type != "function" || tool.Function.Name == "" {
			continue
		}
		decl := &genai.FunctionDeclaration{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
		}
		if len(tool.Function.Parameters) > 0 {
			decl.ParametersJsonSchema = sanitizeSchemaForGemini(tool.Function.Parameters)
		}
		declarations = append(declarations, decl)
	}
	if len(declarations) == 0 {
		return nil
	}
	return []*genai.Tool{
		{
			FunctionDeclarations: declarations,
		},
	}
}

func parseCandidate(candidate *genai.Candidate) (string, []ToolCall) {
	if candidate == nil || candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", nil
	}

	textParts := make([]string, 0, len(candidate.Content.Parts))
	toolCalls := make([]ToolCall, 0)
	for _, part := range candidate.Content.Parts {
		if part == nil {
			continue
		}
		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}
		if part.FunctionCall == nil {
			continue
		}

		name := part.FunctionCall.Name
		args := part.FunctionCall.Args
		if args == nil {
			args = map[string]any{}
		}
		argsJSON, err := json.Marshal(args)
		if err != nil {
			argsJSON = []byte("{}")
		}

		thoughtSignature := encodeThoughtSignature(part.ThoughtSignature)
		toolCall := ToolCall{
			ID:        part.FunctionCall.ID,
			Type:      "function",
			Name:      name,
			Arguments: args,
			Function: &FunctionCall{
				Name:             name,
				Arguments:        string(argsJSON),
				ThoughtSignature: thoughtSignature,
			},
			ThoughtSignature: thoughtSignature,
		}
		if toolCall.ID == "" {
			toolCall.ID = fmt.Sprintf("call_%s_%d", name, time.Now().UnixNano())
		}
		if thoughtSignature != "" {
			toolCall.ExtraContent = &ExtraContent{
				Google: &GoogleExtra{
					ThoughtSignature: thoughtSignature,
				},
			}
		}
		toolCalls = append(toolCalls, toolCall)
	}

	return strings.Join(textParts, ""), toolCalls
}

func mapFinishReason(reason genai.FinishReason, hasToolCalls bool) string {
	if hasToolCalls {
		return "tool_calls"
	}
	if reason == genai.FinishReasonMaxTokens {
		return "length"
	}
	return "stop"
}

func mapUsage(resp *genai.GenerateContentResponse) *UsageInfo {
	if resp == nil || resp.UsageMetadata == nil {
		return nil
	}
	usage := resp.UsageMetadata
	if usage.PromptTokenCount == 0 && usage.CandidatesTokenCount == 0 && usage.TotalTokenCount == 0 {
		return nil
	}
	return &UsageInfo{
		PromptTokens:     int(usage.PromptTokenCount),
		CompletionTokens: int(usage.CandidatesTokenCount),
		TotalTokens:      int(usage.TotalTokenCount),
	}
}

func applyOptions(config *genai.GenerateContentConfig, options map[string]any) bool {
	if config == nil || options == nil {
		return false
	}

	changed := false
	if maxTokens, ok := asInt(options["max_tokens"]); ok && maxTokens > 0 {
		config.MaxOutputTokens = int32(maxTokens)
		changed = true
	}
	if temperature, ok := asFloat(options["temperature"]); ok {
		temp := float32(temperature)
		config.Temperature = &temp
		changed = true
	}
	return changed
}

func asInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int8:
		return int(x), true
	case int16:
		return int(x), true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case float32:
		return int(x), true
	default:
		return 0, false
	}
}

func asFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	default:
		return 0, false
	}
}

var geminiUnsupportedKeywords = map[string]bool{
	"patternProperties":    true,
	"additionalProperties": true,
	"$schema":              true,
	"$id":                  true,
	"$ref":                 true,
	"$defs":                true,
	"definitions":          true,
	"examples":             true,
	"minLength":            true,
	"maxLength":            true,
	"minimum":              true,
	"maximum":              true,
	"multipleOf":           true,
	"pattern":              true,
	"format":               true,
	"minItems":             true,
	"maxItems":             true,
	"uniqueItems":          true,
	"minProperties":        true,
	"maxProperties":        true,
}

func sanitizeSchemaForGemini(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}

	result := make(map[string]any)
	for k, v := range schema {
		if geminiUnsupportedKeywords[k] {
			continue
		}
		switch val := v.(type) {
		case map[string]any:
			result[k] = sanitizeSchemaForGemini(val)
		case []any:
			sanitized := make([]any, len(val))
			for i, item := range val {
				if m, ok := item.(map[string]any); ok {
					sanitized[i] = sanitizeSchemaForGemini(m)
				} else {
					sanitized[i] = item
				}
			}
			result[k] = sanitized
		default:
			result[k] = v
		}
	}

	if _, hasProps := result["properties"]; hasProps {
		if _, hasType := result["type"]; !hasType {
			result["type"] = "object"
		}
	}

	return result
}
