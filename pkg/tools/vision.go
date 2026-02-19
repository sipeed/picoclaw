package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

const (
	visionMaxFileSize   = 10 * 1024 * 1024
	visionMaxTokens     = 1000
	visionTimeout       = 60 * time.Second
	visionDefaultPrompt = "Describe exactly what you see in this image in detail. If it's a screenshot, read the texts and describe the UI elements."
)

type AnalyzeImageTool struct {
	apiKey    string
	apiURL    string
	model     string
	workspace string
	restrict  bool
}

func NewAnalyzeImageTool(opts config.VisionToolConfig) *AnalyzeImageTool {
	return &AnalyzeImageTool{
		workspace: opts.Workspace,
		restrict:  opts.Restrict,
		apiKey:    opts.ApiKey,
		apiURL:    opts.ApiURL,
		model:     opts.Model,
	}
}

func (t *AnalyzeImageTool) Name() string {
	return "analyze_image"
}

func (t *AnalyzeImageTool) Description() string {
	return "Analyze an image file (e.g., png, jpeg) and return a detailed textual description of its contents. Use this to understand screenshots or photos."
}

func (t *AnalyzeImageTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the image file to analyze",
			},
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "Optional specific question about the image (e.g. 'Read the error message', 'Where is the login button?'). Defaults to a general description.",
			},
		},
		"required": []string{"path"},
	}
}

func getMimeType(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png", nil
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".webp":
		return "image/webp", nil
	case ".gif":
		return "image/gif", nil
	default:
		return "", fmt.Errorf("unsupported image extension: %s", ext)
	}
}

func (t *AnalyzeImageTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if t.apiKey == "" {
		return ErrorResult("analyze_image tool is not configured properly: missing Vision API key")
	}

	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	prompt := visionDefaultPrompt
	if customPrompt, ok := args["prompt"].(string); ok && customPrompt != "" {
		prompt = customPrompt
	}

	resolvedPath, err := validatePath(path, t.workspace, t.restrict)
	if err != nil {
		return ErrorResult(err.Error())
	}

	// Secure image file reading, limited to 10MB to avoid huge payloads
	fileInfo, err := os.Stat(resolvedPath)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to stat image: %v", err))
	}
	if fileInfo.Size() > visionMaxFileSize {
		return ErrorResult("image is too large (max 10MB allowed for analysis)")
	}

	imgData, err := os.ReadFile(resolvedPath)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read image: %v", err))
	}

	contentType := http.DetectContentType(imgData)
	if !strings.HasPrefix(contentType, "image/") {
		return ErrorResult(fmt.Sprintf("file is not a valid image, detected type: %s", contentType))
	}

	base64Image := base64.StdEncoding.EncodeToString(imgData)
	mimeType, err := getMimeType(resolvedPath)
	if err != nil {
		return ErrorResult(err.Error())
	}

	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image)

	payloadBytes, err := json.Marshal(map[string]interface{}{
		"model": t.model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{"type": "text", "text": prompt},
					{"type": "image_url", "image_url": map[string]string{"url": dataURI}},
				},
			},
		},
		"max_tokens": visionMaxTokens,
	})
	if err != nil {
		return ErrorResult("failed to prepare vision API payload")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.apiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return ErrorResult("failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	client := &http.Client{Timeout: visionTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("vision API request failed: %v", err))
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read vision API response body: %v", err))
	}

	if resp.StatusCode != http.StatusOK {
		return ErrorResult(fmt.Sprintf("vision API returned error (%d): %s", resp.StatusCode, string(bodyBytes)))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return ErrorResult("failed to parse vision API response")
	}

	if len(apiResp.Choices) == 0 {
		return ErrorResult("vision API returned no content")
	}

	analysisResult := apiResp.Choices[0].Message.Content
	finalOutput := fmt.Sprintf("[Image Analysis Result for %s]\n%s", filepath.Base(resolvedPath), analysisResult)

	return NewToolResult(finalOutput)
}
