package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	antigravityBaseURL      = "https://cloudcode-pa.googleapis.com"
	antigravityDefaultModel = "gemini-3-flash"
	antigravityUserAgent    = "antigravity"
	antigravityXGoogClient  = "google-cloud-sdk vscode_cloudshelleditor/0.1"
	antigravityVersion      = "1.15.8"
)

// AntigravityProvider implements LLMProvider using Google's Cloud Code Assist (Antigravity) API.
// This provider authenticates via Google OAuth and provides access to models like Claude and Gemini
// through Google's infrastructure.
type AntigravityProvider struct {
	tokenSource func() (string, string, error) // Returns (accessToken, projectID, error)
	httpClient  *http.Client
}

// NewAntigravityProvider creates a new Antigravity provider using stored auth credentials.
func NewAntigravityProvider() *AntigravityProvider {
	return &AntigravityProvider{
		tokenSource: createAntigravityTokenSource(),
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Chat implements LLMProvider.Chat using the Cloud Code Assist v1internal API.
// The v1internal endpoint wraps the standard Gemini request in an envelope with
// project, model, request, requestType, userAgent, and requestId fields.
func (p *AntigravityProvider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	accessToken, projectID, err := p.tokenSource()
	if err != nil {
		return nil, fmt.Errorf("antigravity auth: %w", err)
	}

	if model == "" || model == "antigravity" || model == "google-antigravity" {
		model = antigravityDefaultModel
	}
	// Strip provider prefixes if present
	model = strings.TrimPrefix(model, "google-antigravity/")
	model = strings.TrimPrefix(model, "antigravity/")

	logger.DebugCF("provider.antigravity", "Starting chat", map[string]any{
		"model":     model,
		"project":   projectID,
		"requestId": fmt.Sprintf("agent-%d-%s", time.Now().UnixMilli(), randomString(9)),
	})

	// Build the inner Gemini-format request
	innerRequest := p.buildRequest(messages, tools, model, options)

	// Wrap in v1internal envelope (matches pi-ai SDK format)
	envelope := map[string]any{
		"project":     projectID,
		"model":       model,
		"request":     innerRequest,
		"requestType": "agent",
		"userAgent":   antigravityUserAgent,
		"requestId":   fmt.Sprintf("agent-%d-%s", time.Now().UnixMilli(), randomString(9)),
	}

	bodyBytes, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	// Build API URL — uses Cloud Code Assist v1internal streaming endpoint
	apiURL := fmt.Sprintf("%s/v1internal:streamGenerateContent?alt=sse", antigravityBaseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Headers matching the pi-ai SDK antigravity format
	clientMetadata, _ := json.Marshal(map[string]string{
		"ideType":    "IDE_UNSPECIFIED",
		"platform":   "PLATFORM_UNSPECIFIED",
		"pluginType": "GEMINI",
	})
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", fmt.Sprintf("antigravity/%s linux/amd64", antigravityVersion))
	req.Header.Set("X-Goog-Api-Client", antigravityXGoogClient)
	req.Header.Set("Client-Metadata", string(clientMetadata))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("antigravity API call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.ErrorCF("provider.antigravity", "API call failed", map[string]any{
			"status_code": resp.StatusCode,
			"response":    string(respBody),
			"model":       model,
		})

		return nil, p.parseAntigravityError(resp.StatusCode, respBody)
	}

	// Response is always SSE from streamGenerateContent — each line is "data: {...}"
	// with a "response" wrapper containing the standard Gemini response
	llmResp, err := p.parseSSEResponse(string(respBody))
	if err != nil {
		return nil, err
	}

	// Check for empty response (some models might return valid success but empty text)
	if llmResp.Content == "" && len(llmResp.ToolCalls) == 0 {
		return nil, fmt.Errorf(
			"antigravity: model returned an empty response (this model might be invalid or restricted)",
		)
	}

	return llmResp, nil
}

// GetDefaultModel returns the default model identifier.
func (p *AntigravityProvider) GetDefaultModel() string {
	return antigravityDefaultModel
}

// --- Request building ---

type antigravityRequest struct {
	Contents     []antigravityContent     `json:"contents"`
	Tools        []antigravityTool        `json:"tools,omitempty"`
	SystemPrompt *antigravitySystemPrompt `json:"systemInstruction,omitempty"`
	Config       *antigravityGenConfig    `json:"generationConfig,omitempty"`
}

type antigravityContent struct {
	Role  string            `json:"role"`
	Parts []antigravityPart `json:"parts"`
}

type antigravityPart struct {
	Text                  string                       `json:"text,omitempty"`
	ThoughtSignature      string                       `json:"thoughtSignature,omitempty"`
	ThoughtSignatureSnake string                       `json:"thought_signature,omitempty"`
	FunctionCall          *antigravityFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse      *antigravityFunctionResponse `json:"functionResponse,omitempty"`
}

type antigravityFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type antigravityFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type antigravityTool struct {
	FunctionDeclarations []antigravityFuncDecl `json:"functionDeclarations"`
}

type antigravityFuncDecl struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type antigravitySystemPrompt struct {
	Parts []antigravityPart `json:"parts"`
}

type antigravityGenConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
}

func (p *AntigravityProvider) buildRequest(
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) antigravityRequest {
	req := antigravityRequest{}
	toolCallNames := make(map[string]string)

	// Build contents from messages
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			req.SystemPrompt = &antigravitySystemPrompt{
				Parts: []antigravityPart{{Text: msg.Content}},
			}
		case "user":
			if msg.ToolCallID != "" {
				toolName := resolveToolResponseName(msg.ToolCallID, toolCallNames)
				// Tool result
				req.Contents = append(req.Contents, antigravityContent{
					Role: "user",
					Parts: []antigravityPart{{
						FunctionResponse: &antigravityFunctionResponse{
							Name: toolName,
							Response: map[string]any{
								"result": msg.Content,
							},
						},
					}},
				})
			} else {
				req.Contents = append(req.Contents, antigravityContent{
					Role:  "user",
					Parts: []antigravityPart{{Text: msg.Content}},
				})
			}
		case "assistant":
			content := antigravityContent{
				Role: "model",
			}
			if msg.Content != "" {
				content.Parts = append(content.Parts, antigravityPart{Text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				toolName, toolArgs, thoughtSignature := normalizeStoredToolCall(tc)
				if toolName == "" {
					logger.WarnCF(
						"provider.antigravity",
						"Skipping tool call with empty name in history",
						map[string]any{
							"tool_call_id": tc.ID,
						},
					)
					continue
				}
				if tc.ID != "" {
					toolCallNames[tc.ID] = toolName
				}
				content.Parts = append(content.Parts, antigravityPart{
					ThoughtSignature:      thoughtSignature,
					ThoughtSignatureSnake: thoughtSignature,
					FunctionCall: &antigravityFunctionCall{
						Name: toolName,
						Args: toolArgs,
					},
				})
			}
			if len(content.Parts) > 0 {
				req.Contents = append(req.Contents, content)
			}
		case "tool":
			toolName := resolveToolResponseName(msg.ToolCallID, toolCallNames)
			req.Contents = append(req.Contents, antigravityContent{
				Role: "user",
				Parts: []antigravityPart{{
					FunctionResponse: &antigravityFunctionResponse{
						Name: toolName,
						Response: map[string]any{
							"result": msg.Content,
						},
					},
				}},
			})
		}
	}

	// Build tools (sanitize schemas for Gemini compatibility)
	if len(tools) > 0 {
		var funcDecls []antigravityFuncDecl
		for _, t := range tools {
			if t.Type != "function" {
				continue
			}
			params := sanitizeSchemaForGemini(t.Function.Parameters)
			funcDecls = append(funcDecls, antigravityFuncDecl{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  params,
			})
		}
		if len(funcDecls) > 0 {
			req.Tools = []antigravityTool{{FunctionDeclarations: funcDecls}}
		}
	}

	// Generation config
	config := &antigravityGenConfig{}
	if val, ok := options["max_tokens"]; ok {
		if maxTokens, ok := val.(int); ok && maxTokens > 0 {
			config.MaxOutputTokens = maxTokens
		} else if maxTokens, ok := val.(float64); ok && maxTokens > 0 {
			config.MaxOutputTokens = int(maxTokens)
		}
	}
	if temp, ok := options["temperature"].(float64); ok {
		config.Temperature = temp
	}
	if config.MaxOutputTokens > 0 || config.Temperature > 0 {
		req.Config = config
	}

	return req
}

func normalizeStoredToolCall(tc ToolCall) (string, map[string]any, string) {
	name := tc.Name
	args := tc.Arguments
	thoughtSignature := ""

	if name == "" && tc.Function != nil {
		name = tc.Function.Name
		thoughtSignature = tc.Function.ThoughtSignature
	} else if tc.Function != nil {
		thoughtSignature = tc.Function.ThoughtSignature
	}

	if args == nil {
		args = map[string]any{}
	}

	if len(args) == 0 && tc.Function != nil && tc.Function.Arguments != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &parsed); err == nil && parsed != nil {
			args = parsed
		}
	}

	return name, args, thoughtSignature
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

// --- Response parsing ---

type antigravityJSONResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text                  string                   `json:"text,omitempty"`
				ThoughtSignature      string                   `json:"thoughtSignature,omitempty"`
				ThoughtSignatureSnake string                   `json:"thought_signature,omitempty"`
				FunctionCall          *antigravityFunctionCall `json:"functionCall,omitempty"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

func (p *AntigravityProvider) parseJSONResponse(body []byte) (*LLMResponse, error) {
	var resp antigravityJSONResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing antigravity response: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("antigravity: no candidates in response")
	}

	candidate := resp.Candidates[0]
	var contentParts []string
	var toolCalls []ToolCall

	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			contentParts = append(contentParts, part.Text)
		}
		if part.FunctionCall != nil {
			argumentsJSON, _ := json.Marshal(part.FunctionCall.Args)
			toolCalls = append(toolCalls, ToolCall{
				ID:        fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, time.Now().UnixNano()),
				Name:      part.FunctionCall.Name,
				Arguments: part.FunctionCall.Args,
				Function: &FunctionCall{
					Name:             part.FunctionCall.Name,
					Arguments:        string(argumentsJSON),
					ThoughtSignature: extractPartThoughtSignature(part.ThoughtSignature, part.ThoughtSignatureSnake),
				},
			})
		}
	}

	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}
	if candidate.FinishReason == "MAX_TOKENS" {
		finishReason = "length"
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
		Content:      strings.Join(contentParts, ""),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}

func (p *AntigravityProvider) parseSSEResponse(body string) (*LLMResponse, error) {
	var contentParts []string
	var toolCalls []ToolCall
	var usage *UsageInfo
	var finishReason string

	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		// v1internal SSE wraps the Gemini response in a "response" field
		var sseChunk struct {
			Response antigravityJSONResponse `json:"response"`
		}
		if err := json.Unmarshal([]byte(data), &sseChunk); err != nil {
			continue
		}
		resp := sseChunk.Response

		for _, candidate := range resp.Candidates {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					contentParts = append(contentParts, part.Text)
				}
				if part.FunctionCall != nil {
					argumentsJSON, _ := json.Marshal(part.FunctionCall.Args)
					toolCalls = append(toolCalls, ToolCall{
						ID:        fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, time.Now().UnixNano()),
						Name:      part.FunctionCall.Name,
						Arguments: part.FunctionCall.Args,
						Function: &FunctionCall{
							Name:      part.FunctionCall.Name,
							Arguments: string(argumentsJSON),
							ThoughtSignature: extractPartThoughtSignature(
								part.ThoughtSignature,
								part.ThoughtSignatureSnake,
							),
						},
					})
				}
			}
			if candidate.FinishReason != "" {
				finishReason = candidate.FinishReason
			}
		}

		if resp.UsageMetadata.TotalTokenCount > 0 {
			usage = &UsageInfo{
				PromptTokens:     resp.UsageMetadata.PromptTokenCount,
				CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
				TotalTokens:      resp.UsageMetadata.TotalTokenCount,
			}
		}
	}

	mappedFinish := "stop"
	if len(toolCalls) > 0 {
		mappedFinish = "tool_calls"
	}
	if finishReason == "MAX_TOKENS" {
		mappedFinish = "length"
	}

	return &LLMResponse{
		Content:      strings.Join(contentParts, ""),
		ToolCalls:    toolCalls,
		FinishReason: mappedFinish,
		Usage:        usage,
	}, nil
}

func extractPartThoughtSignature(thoughtSignature string, thoughtSignatureSnake string) string {
	if thoughtSignature != "" {
		return thoughtSignature
	}
	if thoughtSignatureSnake != "" {
		return thoughtSignatureSnake
	}
	return ""
}

// --- Schema sanitization ---

// Google/Gemini doesn't support many JSON Schema keywords that other providers accept.
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
		// Recursively sanitize nested objects
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

	// Ensure top-level has type: "object" if properties are present
	if _, hasProps := result["properties"]; hasProps {
		if _, hasType := result["type"]; !hasType {
			result["type"] = "object"
		}
	}

	return result
}

// --- Token source ---

func createAntigravityTokenSource() func() (string, string, error) {
	return func() (string, string, error) {
		cred, err := auth.GetCredential("google-antigravity")
		if err != nil {
			return "", "", fmt.Errorf("loading auth credentials: %w", err)
		}
		if cred == nil {
			return "", "", fmt.Errorf(
				"no credentials for google-antigravity. Run: picoclaw auth login --provider google-antigravity",
			)
		}

		// Refresh if needed
		if cred.NeedsRefresh() && cred.RefreshToken != "" {
			oauthCfg := auth.GoogleAntigravityOAuthConfig()
			refreshed, err := auth.RefreshAccessToken(cred, oauthCfg)
			if err != nil {
				return "", "", fmt.Errorf("refreshing token: %w", err)
			}
			refreshed.Email = cred.Email
			if refreshed.ProjectID == "" {
				refreshed.ProjectID = cred.ProjectID
			}
			if err := auth.SetCredential("google-antigravity", refreshed); err != nil {
				return "", "", fmt.Errorf("saving refreshed token: %w", err)
			}
			cred = refreshed
		}

		if cred.IsExpired() {
			return "", "", fmt.Errorf(
				"antigravity credentials expired. Run: picoclaw auth login --provider google-antigravity",
			)
		}

		projectID := cred.ProjectID
		if projectID == "" {
			// Try to fetch project ID from API
			fetchedID, err := FetchAntigravityProjectID(cred.AccessToken)
			if err != nil {
				logger.WarnCF("provider.antigravity", "Could not fetch project ID, using fallback", map[string]any{
					"error": err.Error(),
				})
				projectID = "rising-fact-p41fc" // Default fallback (same as OpenCode)
			} else {
				projectID = fetchedID
				cred.ProjectID = projectID
				_ = auth.SetCredential("google-antigravity", cred)
			}
		}

		return cred.AccessToken, projectID, nil
	}
}

// FetchAntigravityProjectID retrieves the Google Cloud project ID from the loadCodeAssist endpoint.
func FetchAntigravityProjectID(accessToken string) (string, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"ideType":    "IDE_UNSPECIFIED",
			"platform":   "PLATFORM_UNSPECIFIED",
			"pluginType": "GEMINI",
		},
	})

	req, err := http.NewRequest("POST", antigravityBaseURL+"/v1internal:loadCodeAssist", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", antigravityUserAgent)
	req.Header.Set("X-Goog-Api-Client", antigravityXGoogClient)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("loadCodeAssist failed: %s", string(body))
	}

	var result struct {
		CloudAICompanionProject string `json:"cloudaicompanionProject"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.CloudAICompanionProject == "" {
		return "", fmt.Errorf("no project ID in loadCodeAssist response")
	}

	return result.CloudAICompanionProject, nil
}

// FetchAntigravityModels fetches available models from the Cloud Code Assist API.
func FetchAntigravityModels(accessToken, projectID string) ([]AntigravityModelInfo, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"project": projectID,
	})

	req, err := http.NewRequest("POST", antigravityBaseURL+"/v1internal:fetchAvailableModels", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", antigravityUserAgent)
	req.Header.Set("X-Goog-Api-Client", antigravityXGoogClient)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"fetchAvailableModels failed (HTTP %d): %s",
			resp.StatusCode,
			truncateString(string(body), 200),
		)
	}

	var result struct {
		Models map[string]struct {
			DisplayName string `json:"displayName"`
			QuotaInfo   struct {
				RemainingFraction any    `json:"remainingFraction"`
				ResetTime         string `json:"resetTime"`
				IsExhausted       bool   `json:"isExhausted"`
			} `json:"quotaInfo"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing models response: %w", err)
	}

	var models []AntigravityModelInfo
	for id, info := range result.Models {
		models = append(models, AntigravityModelInfo{
			ID:          id,
			DisplayName: info.DisplayName,
			IsExhausted: info.QuotaInfo.IsExhausted,
		})
	}

	// Ensure gemini-3-flash-preview and gemini-3-flash are in the list if they aren't already
	hasFlashPreview := false
	hasFlash := false
	for _, m := range models {
		if m.ID == "gemini-3-flash-preview" {
			hasFlashPreview = true
		}
		if m.ID == "gemini-3-flash" {
			hasFlash = true
		}
	}
	if !hasFlashPreview {
		models = append(models, AntigravityModelInfo{
			ID:          "gemini-3-flash-preview",
			DisplayName: "Gemini 3 Flash (Preview)",
		})
	}
	if !hasFlash {
		models = append(models, AntigravityModelInfo{
			ID:          "gemini-3-flash",
			DisplayName: "Gemini 3 Flash",
		})
	}

	return models, nil
}

type AntigravityModelInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	IsExhausted bool   `json:"is_exhausted"`
}

// --- Helpers ---

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (p *AntigravityProvider) parseAntigravityError(statusCode int, body []byte) error {
	var errResp struct {
		Error struct {
			Code    int              `json:"code"`
			Message string           `json:"message"`
			Status  string           `json:"status"`
			Details []map[string]any `json:"details"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("antigravity API error (HTTP %d): %s", statusCode, truncateString(string(body), 500))
	}

	msg := errResp.Error.Message
	if statusCode == 429 {
		// Try to extract quota reset info
		for _, detail := range errResp.Error.Details {
			if typeVal, ok := detail["@type"].(string); ok && strings.HasSuffix(typeVal, "ErrorInfo") {
				if metadata, ok := detail["metadata"].(map[string]any); ok {
					if delay, ok := metadata["quotaResetDelay"].(string); ok {
						return fmt.Errorf("antigravity rate limit exceeded: %s (reset in %s)", msg, delay)
					}
				}
			}
		}
		return fmt.Errorf("antigravity rate limit exceeded: %s", msg)
	}

	return fmt.Errorf("antigravity API error (%s): %s", errResp.Error.Status, msg)
}
