package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type FirecrawlTool struct {
	apiKey  string
	apiBase string
}

func NewFirecrawlTool(apiKey, apiBase string) *FirecrawlTool {
	if apiBase == "" {
		apiBase = "https://api.firecrawl.dev/v1"
	}
	return &FirecrawlTool{
		apiKey:  apiKey,
		apiBase: apiBase,
	}
}

func (t *FirecrawlTool) Name() string {
	return "firecrawl"
}

func (t *FirecrawlTool) Description() string {
	return "Scrape web pages and extract structured data using Firecrawl. Supports markdown extraction, screenshot capture, and structured data extraction with LLM."
}

func (t *FirecrawlTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to scrape",
			},
			"formats": map[string]interface{}{
				"type":        "array",
				"description": "Output formats (markdown, html, screenshot, links, extract)",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"only_main_content": map[string]interface{}{
				"type":        "boolean",
				"description": "Extract only main content, excluding navigation, headers, footers",
			},
			"extract": map[string]interface{}{
				"type":        "object",
				"description": "Schema for structured data extraction using LLM",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "Prompt describing what to extract",
					},
					"schema": map[string]interface{}{
						"type":        "object",
						"description": "JSON schema for structured output",
					},
				},
			},
		},
		"required": []string{"url"},
	}
}

func (t *FirecrawlTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if t.apiKey == "" {
		return ErrorResult("Firecrawl API key not configured")
	}

	url, ok := args["url"].(string)
	if !ok || url == "" {
		return ErrorResult("url is required")
	}

	// Build request body
	requestBody := map[string]interface{}{
		"url": url,
	}

	if formats, ok := args["formats"].([]interface{}); ok && len(formats) > 0 {
		requestBody["formats"] = formats
	} else {
		// Default to markdown
		requestBody["formats"] = []string{"markdown"}
	}

	if onlyMain, ok := args["only_main_content"].(bool); ok {
		requestBody["onlyMainContent"] = onlyMain
	}

	if extract, ok := args["extract"].(map[string]interface{}); ok {
		requestBody["extract"] = extract
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to marshal request: %v", err))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.apiBase+"/scrape", bytes.NewReader(jsonData))
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read response: %v", err))
	}

	if resp.StatusCode != http.StatusOK {
		return ErrorResult(fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(body)))
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Markdown   string                 `json:"markdown"`
			HTML       string                 `json:"html"`
			Metadata   map[string]interface{} `json:"metadata"`
			Extract    map[string]interface{} `json:"extract"`
			Screenshot string                 `json:"screenshot"`
			Links      []string               `json:"links"`
		} `json:"data"`
		Error string `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse response: %v", err))
	}

	if !result.Success {
		return ErrorResult(fmt.Sprintf("Firecrawl error: %s", result.Error))
	}

	// Build response
	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Scraped: %s\n\n", url))

	if result.Data.Metadata != nil {
		if title, ok := result.Data.Metadata["title"].(string); ok && title != "" {
			output.WriteString(fmt.Sprintf("**Title:** %s\n\n", title))
		}
		if description, ok := result.Data.Metadata["description"].(string); ok && description != "" {
			output.WriteString(fmt.Sprintf("**Description:** %s\n\n", description))
		}
	}

	if result.Data.Markdown != "" {
		output.WriteString("## Content (Markdown)\n\n")
		output.WriteString(result.Data.Markdown)
		output.WriteString("\n\n")
	}

	if result.Data.Extract != nil && len(result.Data.Extract) > 0 {
		output.WriteString("## Extracted Data\n\n")
		extractJSON, _ := json.MarshalIndent(result.Data.Extract, "", "  ")
		output.WriteString("```json\n")
		output.WriteString(string(extractJSON))
		output.WriteString("\n```\n\n")
	}

	if len(result.Data.Links) > 0 {
		output.WriteString(fmt.Sprintf("## Links Found: %d\n\n", len(result.Data.Links)))
		for i, link := range result.Data.Links {
			if i >= 20 { // Limit to 20 links
				output.WriteString(fmt.Sprintf("\n... and %d more links\n", len(result.Data.Links)-20))
				break
			}
			output.WriteString(fmt.Sprintf("- %s\n", link))
		}
	}

	if result.Data.Screenshot != "" {
		output.WriteString("\n**Screenshot:** [Base64 encoded image available]\n")
	}

	return &ToolResult{
		ForLLM:  output.String(),
		ForUser: output.String(),
	}
}
