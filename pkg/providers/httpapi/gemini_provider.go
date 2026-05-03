package httpapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/providers/common"
)

const (
	geminiDefaultAPIBase = "https://generativelanguage.googleapis.com/v1beta"
	geminiDefaultModel   = "gemini-2.0-flash"
)

type GeminiProvider struct {
	apiKey        string
	apiBase       string
	httpClient    *http.Client
	extraBody     map[string]any
	customHeaders map[string]string
	userAgent     string
}

func NewGeminiProvider(
	apiKey string,
	apiBase string,
	proxy string,
	userAgent string,
	requestTimeoutSeconds int,
	extraBody map[string]any,
	customHeaders map[string]string,
) *GeminiProvider {
	if strings.TrimSpace(apiBase) == "" {
		apiBase = geminiDefaultAPIBase
	}
	client := common.NewHTTPClient(proxy)
	if requestTimeoutSeconds > 0 {
		client.Timeout = time.Duration(requestTimeoutSeconds) * time.Second
	}

	return &GeminiProvider{
		apiKey:        strings.TrimSpace(apiKey),
		apiBase:       strings.TrimRight(strings.TrimSpace(apiBase), "/"),
		httpClient:    client,
		extraBody:     cloneAnyMap(extraBody),
		customHeaders: cloneStringMap(customHeaders),
		userAgent:     strings.TrimSpace(userAgent),
	}
}

func (p *GeminiProvider) GetDefaultModel() string {
	return geminiDefaultModel
}

func (p *GeminiProvider) SupportsThinking() bool {
	return true
}

func (p *GeminiProvider) SupportsVideo() bool {
	return true
}

func (p *GeminiProvider) SupportsAudio() bool {
	return true
}

func (p *GeminiProvider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	if p.apiBase == "" {
		return nil, fmt.Errorf("API base not configured")
	}

	model = normalizeGeminiModel(model)
	requestBody := p.buildRequestBody(messages, tools, model, options)
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", p.apiBase, model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.applyHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, common.HandleErrorResponse(resp, p.apiBase)
	}

	var apiResp geminiGenerateContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return parseGeminiResponse(&apiResp), nil
}

func (p *GeminiProvider) ChatStream(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
	onChunk func(accumulated string),
) (*LLMResponse, error) {
	if p.apiBase == "" {
		return nil, fmt.Errorf("API base not configured")
	}

	model = normalizeGeminiModel(model)
	requestBody := p.buildRequestBody(messages, tools, model, options)
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", p.apiBase, model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.applyHeaders(req)
	req.Header.Set("Accept", "text/event-stream")

	// Streaming should not use a whole-request timeout; context cancellation is the guard.
	streamClient := &http.Client{Transport: p.httpClient.Transport}
	resp, err := streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, common.HandleErrorResponse(resp, p.apiBase)
	}

	return parseGeminiStreamResponse(ctx, resp.Body, onChunk)
}

func (p *GeminiProvider) applyHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("X-Goog-Api-Key", p.apiKey)
	}
	if p.userAgent != "" {
		req.Header.Set("User-Agent", p.userAgent)
	}
	for k, v := range p.customHeaders {
		if strings.TrimSpace(k) == "" {
			continue
		}
		req.Header.Set(k, v)
	}
}

func (p *GeminiProvider) buildRequestBody(
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) map[string]any {
	contents := make([]geminiContent, 0, len(messages))
	toolCallNames := make(map[string]string)
	systemPrompts := make([]string, 0, 1)

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			if strings.TrimSpace(msg.Content) != "" {
				systemPrompts = append(systemPrompts, msg.Content)
			}

		case "user":
			if msg.ToolCallID != "" {
				toolName := common.ResolveToolResponseName(msg.ToolCallID, toolCallNames)
				contents = append(contents, geminiContent{
					Role: "user",
					Parts: []geminiPart{{
						FunctionResponse: buildGeminiFunctionResponse(toolName, msg.ToolCallID, msg.Content, msg.Media),
					}},
				})
				continue
			}

			parts := make([]geminiPart, 0, 1+len(msg.Media))
			if strings.TrimSpace(msg.Content) != "" {
				parts = append(parts, geminiPart{Text: msg.Content})
			}
			parts = append(parts, buildInlineMediaParts(msg.Media)...)
			if len(parts) > 0 {
				contents = append(contents, geminiContent{Role: "user", Parts: parts})
			}

		case "assistant":
			content := geminiContent{Role: "model"}
			if strings.TrimSpace(msg.Content) != "" {
				content.Parts = append(content.Parts, geminiPart{Text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				toolName, toolArgs, thoughtSignature := common.NormalizeStoredToolCall(tc)
				if toolName == "" {
					continue
				}
				if tc.ID != "" {
					toolCallNames[tc.ID] = toolName
				}
				part := geminiPart{
					FunctionCall: &geminiFunctionCall{
						Name: toolName,
						Args: toolArgs,
						ID:   tc.ID,
					},
				}
				if thoughtSignature != "" {
					part.ThoughtSignature = thoughtSignature
				}
				content.Parts = append(content.Parts, part)
			}
			if len(content.Parts) > 0 {
				contents = append(contents, content)
			}

		case "tool":
			toolName := common.ResolveToolResponseName(msg.ToolCallID, toolCallNames)
			contents = append(contents, geminiContent{
				Role: "user",
				Parts: []geminiPart{{
					FunctionResponse: buildGeminiFunctionResponse(toolName, msg.ToolCallID, msg.Content, msg.Media),
				}},
			})
		}
	}

	body := map[string]any{
		"contents": contents,
	}
	if len(systemPrompts) > 0 {
		systemParts := make([]geminiPart, 0, len(systemPrompts))
		for _, prompt := range systemPrompts {
			systemParts = append(systemParts, geminiPart{Text: prompt})
		}
		body["systemInstruction"] = &geminiContent{Parts: systemParts}
	}

	if len(tools) > 0 {
		funcDecls := make([]geminiFunctionDeclaration, 0, len(tools))
		for _, t := range tools {
			if t.Type != "function" {
				continue
			}
			funcDecls = append(funcDecls, geminiFunctionDeclaration{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			})
		}
		if len(funcDecls) > 0 {
			body["tools"] = []geminiTool{{FunctionDeclarations: funcDecls}}
		}
	}

	generationConfig := make(map[string]any)
	if val, ok := options["max_tokens"]; ok {
		if maxTokens, ok := val.(int); ok && maxTokens > 0 {
			generationConfig["maxOutputTokens"] = maxTokens
		} else if maxTokens, ok := val.(float64); ok && maxTokens > 0 {
			generationConfig["maxOutputTokens"] = int(maxTokens)
		}
	}
	if temp, ok := options["temperature"].(float64); ok {
		generationConfig["temperature"] = temp
	}

	if thinkingConfig := buildGeminiThinkingConfig(model, options); len(thinkingConfig) > 0 {
		generationConfig["thinkingConfig"] = thinkingConfig
	}

	if len(generationConfig) > 0 {
		body["generationConfig"] = generationConfig
	}

	for k, v := range p.extraBody {
		body[k] = v
	}

	return body
}

func normalizeGeminiModel(model string) string {
	model = strings.TrimSpace(model)
	model = strings.TrimPrefix(model, "models/")
	if strings.Contains(model, "/") {
		_, modelID := extractProtocol(model)
		if modelID != "" {
			return modelID
		}
	}
	if model == "" {
		return geminiDefaultModel
	}
	return model
}

func mapGeminiThinkingLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "minimal", "off":
		return "minimal"
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high", "xhigh", "adaptive":
		return "high"
	default:
		return ""
	}
}

func buildGeminiThinkingConfig(model string, options map[string]any) map[string]any {
	if !geminiModelSupportsThinkingConfig(model) {
		return nil
	}

	config := map[string]any{}
	rawLevel, _ := options["thinking_level"].(string)
	rawLevel = strings.ToLower(strings.TrimSpace(rawLevel))
	if rawLevel == "" {
		// Align with agent-level default: unset means ThinkingOff.
		rawLevel = "off"
	}

	includeThoughts := rawLevel != "off" && rawLevel != "minimal"
	config["includeThoughts"] = includeThoughts

	if isGemini25Model(model) {
		if isGemini25ProModel(model) && (rawLevel == "off" || rawLevel == "minimal") {
			// Gemini 2.5 Pro cannot disable thinking; keep model-default thinking.
			return config
		}
		if budget, ok := mapGeminiThinkingBudget(rawLevel); ok {
			config["thinkingBudget"] = budget
		}
		return config
	}

	if isGemini3ProModel(model) && (rawLevel == "off" || rawLevel == "minimal") {
		// Gemini 3.x Pro does not support minimal thinking level.
		return config
	}

	if thinkingLevel := mapGeminiThinkingLevel(rawLevel); thinkingLevel != "" {
		config["thinkingLevel"] = thinkingLevel
	}
	return config
}

func geminiModelSupportsThinkingConfig(model string) bool {
	lowerModel := strings.ToLower(strings.TrimSpace(model))
	return strings.Contains(lowerModel, "gemini-3") || isGemini25Model(lowerModel)
}

func isGemini25Model(model string) bool {
	lowerModel := strings.ToLower(strings.TrimSpace(model))
	return strings.Contains(lowerModel, "gemini-2.5") || strings.Contains(lowerModel, "gemini-25")
}

func isGemini25ProModel(model string) bool {
	lowerModel := strings.ToLower(strings.TrimSpace(model))
	return isGemini25Model(lowerModel) && strings.Contains(lowerModel, "pro")
}

func isGemini3ProModel(model string) bool {
	lowerModel := strings.ToLower(strings.TrimSpace(model))
	return strings.Contains(lowerModel, "gemini-3") && strings.Contains(lowerModel, "pro")
}

func mapGeminiThinkingBudget(level string) (int, bool) {
	level = strings.ToLower(strings.TrimSpace(level))
	if level == "" {
		return 0, false
	}

	switch level {
	case "adaptive":
		return -1, true
	case "minimal":
		return 0, true
	case "off":
		return 0, true
	case "low":
		return 1024, true
	case "medium":
		return 4096, true
	case "high":
		return 8192, true
	case "xhigh":
		return 16384, true
	default:
		return 0, false
	}
}

func parseGeminiResponse(resp *geminiGenerateContentResponse) *LLMResponse {
	contentParts := make([]string, 0)
	reasoningParts := make([]string, 0)
	toolCalls := make([]ToolCall, 0)
	finishReason := ""

	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				if part.Thought {
					reasoningParts = append(reasoningParts, part.Text)
				} else {
					contentParts = append(contentParts, part.Text)
				}
			}
			if part.FunctionCall != nil {
				toolCalls = append(toolCalls, buildGeminiToolCall(part))
			}
		}
		if candidate.FinishReason != "" {
			finishReason = candidate.FinishReason
		}
	}

	var usage *UsageInfo
	if resp.UsageMetadata.TotalTokenCount > 0 {
		usage = &UsageInfo{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	return &LLMResponse{
		Content:          strings.Join(contentParts, ""),
		ReasoningContent: strings.Join(reasoningParts, ""),
		ToolCalls:        toolCalls,
		FinishReason:     normalizeGeminiFinishReason(finishReason, len(toolCalls)),
		Usage:            usage,
	}
}

func parseGeminiStreamResponse(
	ctx context.Context,
	reader io.Reader,
	onChunk func(accumulated string),
) (*LLMResponse, error) {
	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder
	var finishReason string
	var usage *UsageInfo

	toolCallsByID := make(map[string]ToolCall)
	toolCallOrder := make([]string, 0)
	fallbackIndex := 0

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			break
		}

		var chunk geminiGenerateContentResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("invalid gemini stream chunk: %w", err)
		}

		for _, candidate := range chunk.Candidates {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					if part.Thought {
						reasoningBuilder.WriteString(part.Text)
					} else {
						contentBuilder.WriteString(part.Text)
						if onChunk != nil {
							onChunk(contentBuilder.String())
						}
					}
				}
				if part.FunctionCall != nil {
					tc := buildGeminiToolCall(part)
					if strings.TrimSpace(tc.Name) == "" {
						continue
					}

					key := strings.TrimSpace(part.FunctionCall.ID)
					if key == "" {
						if len(toolCallOrder) > 0 {
							lastKey := toolCallOrder[len(toolCallOrder)-1]
							if lastTC, exists := toolCallsByID[lastKey]; exists && lastTC.Name == tc.Name {
								key = lastKey
							}
						}
						if key == "" {
							fallbackIndex++
							key = fmt.Sprintf("%s#%d", tc.Name, fallbackIndex)
						}
					}

					tc.ID = key
					if _, exists := toolCallsByID[key]; !exists {
						toolCallOrder = append(toolCallOrder, key)
					}
					toolCallsByID[key] = tc
				}
			}
			if candidate.FinishReason != "" {
				finishReason = candidate.FinishReason
			}
		}

		if chunk.UsageMetadata.TotalTokenCount > 0 {
			usage = &UsageInfo{
				PromptTokens:     chunk.UsageMetadata.PromptTokenCount,
				CompletionTokens: chunk.UsageMetadata.CandidatesTokenCount,
				TotalTokens:      chunk.UsageMetadata.TotalTokenCount,
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("streaming read error: %w", err)
	}

	toolCalls := make([]ToolCall, 0, len(toolCallOrder))
	for _, key := range toolCallOrder {
		toolCalls = append(toolCalls, toolCallsByID[key])
	}

	return &LLMResponse{
		Content:          contentBuilder.String(),
		ReasoningContent: reasoningBuilder.String(),
		ToolCalls:        toolCalls,
		FinishReason:     normalizeGeminiFinishReason(finishReason, len(toolCalls)),
		Usage:            usage,
	}, nil
}

func normalizeGeminiFinishReason(reason string, toolCalls int) string {
	if toolCalls > 0 {
		return "tool_calls"
	}

	switch strings.ToUpper(strings.TrimSpace(reason)) {
	case "MAX_TOKENS":
		return "length"
	case "", "STOP":
		return "stop"
	default:
		return strings.ToLower(strings.TrimSpace(reason))
	}
}

func buildGeminiToolCall(part geminiPart) ToolCall {
	if part.FunctionCall == nil {
		return ToolCall{}
	}

	args := part.FunctionCall.Args
	if args == nil {
		args = make(map[string]any)
	}
	argsJSON, _ := json.Marshal(args)
	thoughtSignature := extractPartThoughtSignature(part.ThoughtSignature, part.ThoughtSignatureSnake)

	toolCall := ToolCall{
		ID:               part.FunctionCall.ID,
		Name:             part.FunctionCall.Name,
		Arguments:        args,
		ThoughtSignature: thoughtSignature,
		Function: &FunctionCall{
			Name:             part.FunctionCall.Name,
			Arguments:        string(argsJSON),
			ThoughtSignature: thoughtSignature,
		},
	}

	if thoughtSignature != "" {
		toolCall.ExtraContent = &ExtraContent{
			Google: &GoogleExtra{ThoughtSignature: thoughtSignature},
		}
	}
	if strings.TrimSpace(toolCall.ID) == "" {
		toolCall.ID = fmt.Sprintf("call_%s_%d", toolCall.Name, time.Now().UnixNano())
	}

	return toolCall
}

func buildInlineMediaParts(media []string) []geminiPart {
	parts := make([]geminiPart, 0, len(media))
	for _, mediaURL := range media {
		mimeType, data, ok := parseBase64DataURL(mediaURL)
		if !ok {
			continue
		}
		parts = append(parts, geminiPart{
			InlineData: &geminiInlineData{
				MIMEType: mimeType,
				Data:     data,
			},
		})
	}
	return parts
}

func buildGeminiFunctionResponse(
	toolName string,
	toolCallID string,
	result string,
	media []string,
) *geminiFunctionResponse {
	response := &geminiFunctionResponse{
		ID:   toolCallID,
		Name: toolName,
		Response: map[string]any{
			"result": result,
		},
	}

	if parts := buildFunctionResponseMediaParts(media); len(parts) > 0 {
		response.Parts = parts
	}

	return response
}

func buildFunctionResponseMediaParts(media []string) []geminiFunctionResponsePart {
	parts := make([]geminiFunctionResponsePart, 0, len(media))
	for i, mediaURL := range media {
		mimeType, data, ok := parseBase64DataURL(mediaURL)
		if !ok {
			continue
		}
		parts = append(parts, geminiFunctionResponsePart{
			InlineData: &geminiInlineData{
				MIMEType:    mimeType,
				Data:        data,
				DisplayName: defaultFunctionResponseDisplayName(mimeType, i+1),
			},
		})
	}
	return parts
}

func defaultFunctionResponseDisplayName(mimeType string, index int) string {
	suffix := "bin"
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/png":
		suffix = "png"
	case "image/jpeg":
		suffix = "jpg"
	case "image/webp":
		suffix = "webp"
	case "application/pdf":
		suffix = "pdf"
	case "text/plain":
		suffix = "txt"
	}
	return fmt.Sprintf("attachment-%d.%s", index, suffix)
}

func parseBase64DataURL(mediaURL string) (mimeType string, data string, ok bool) {
	if !strings.HasPrefix(mediaURL, "data:") {
		return "", "", false
	}

	payload := strings.TrimPrefix(mediaURL, "data:")
	header, data, found := strings.Cut(payload, ",")
	if !found {
		return "", "", false
	}
	mimeType, params, _ := strings.Cut(header, ";")
	mimeType = strings.TrimSpace(mimeType)
	data = strings.TrimSpace(data)
	if mimeType == "" || data == "" {
		return "", "", false
	}
	if !strings.Contains(strings.ToLower(params), "base64") {
		return "", "", false
	}
	return mimeType, data, true
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

type geminiGenerateContentResponse struct {
	Candidates []struct {
		Content struct {
			Role  string       `json:"role"`
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text                  string                  `json:"text,omitempty"`
	Thought               bool                    `json:"thought,omitempty"`
	ThoughtSignature      string                  `json:"thoughtSignature,omitempty"`
	ThoughtSignatureSnake string                  `json:"thought_signature,omitempty"`
	InlineData            *geminiInlineData       `json:"inlineData,omitempty"`
	FunctionCall          *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse      *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiInlineData struct {
	MIMEType    string `json:"mimeType"`
	Data        string `json:"data"`
	DisplayName string `json:"displayName,omitempty"`
}

type geminiFunctionCall struct {
	ID   string         `json:"id,omitempty"`
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

type geminiFunctionResponse struct {
	ID       string                       `json:"id,omitempty"`
	Name     string                       `json:"name"`
	Response map[string]any               `json:"response"`
	Parts    []geminiFunctionResponsePart `json:"parts,omitempty"`
}

type geminiFunctionResponsePart struct {
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations"`
}

type geminiFunctionDeclaration struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}
