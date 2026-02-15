package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SerpAPITool struct {
	apiKey     string
	maxResults int
}

func NewSerpAPITool(apiKey string, maxResults int) *SerpAPITool {
	if maxResults <= 0 {
		maxResults = 10
	}
	return &SerpAPITool{
		apiKey:     apiKey,
		maxResults: maxResults,
	}
}

func (t *SerpAPITool) Name() string {
	return "serp_api"
}

func (t *SerpAPITool) Description() string {
	return "Search Google and other search engines using SerpAPI. Returns organic results, knowledge graph, related questions, and more."
}

func (t *SerpAPITool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query",
			},
			"engine": map[string]interface{}{
				"type":        "string",
				"description": "Search engine (google, bing, yahoo, duckduckgo, yandex)",
				"enum":        []string{"google", "bing", "yahoo", "duckduckgo", "yandex"},
			},
			"location": map[string]interface{}{
				"type":        "string",
				"description": "Location for localized results (e.g., 'Austin, Texas, United States')",
			},
			"hl": map[string]interface{}{
				"type":        "string",
				"description": "Language code (e.g., 'en', 'es', 'fr')",
			},
			"gl": map[string]interface{}{
				"type":        "string",
				"description": "Country code (e.g., 'us', 'uk', 'ca')",
			},
			"num": map[string]interface{}{
				"type":        "integer",
				"description": "Number of results (1-100)",
				"minimum":     1,
				"maximum":     100,
			},
		},
		"required": []string{"query"},
	}
}

func (t *SerpAPITool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if t.apiKey == "" {
		return ErrorResult("SerpAPI key not configured. Get one at https://serpapi.com/")
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return ErrorResult("query is required")
	}

	// Build query parameters
	params := url.Values{}
	params.Set("q", query)
	params.Set("api_key", t.apiKey)
	params.Set("output", "json")

	// Set engine (default to google)
	engine := "google"
	if e, ok := args["engine"].(string); ok && e != "" {
		engine = e
	}
	params.Set("engine", engine)

	// Set number of results
	num := t.maxResults
	if n, ok := args["num"].(float64); ok && n > 0 {
		num = int(n)
		if num > 100 {
			num = 100
		}
	}
	params.Set("num", fmt.Sprintf("%d", num))

	// Optional parameters
	if location, ok := args["location"].(string); ok && location != "" {
		params.Set("location", location)
	}
	if hl, ok := args["hl"].(string); ok && hl != "" {
		params.Set("hl", hl)
	}
	if gl, ok := args["gl"].(string); ok && gl != "" {
		params.Set("gl", gl)
	}

	// Make request
	apiURL := fmt.Sprintf("https://serpapi.com/search?%s", params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to create request: %v", err))
	}

	client := &http.Client{Timeout: 30 * time.Second}
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

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse response: %v", err))
	}

	// Check for SerpAPI error
	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		return ErrorResult(fmt.Sprintf("SerpAPI error: %s", errMsg))
	}

	// Build output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Search Results for: %s\n\n", query))

	// Search metadata
	if searchMetadata, ok := result["search_metadata"].(map[string]interface{}); ok {
		if status, ok := searchMetadata["status"].(string); ok {
			output.WriteString(fmt.Sprintf("**Status:** %s\n", status))
		}
		if totalTime, ok := searchMetadata["total_time_taken"].(float64); ok {
			output.WriteString(fmt.Sprintf("**Time:** %.2fs\n", totalTime))
		}
	}

	// Search parameters info
	if searchParams, ok := result["search_parameters"].(map[string]interface{}); ok {
		if engine, ok := searchParams["engine"].(string); ok {
			output.WriteString(fmt.Sprintf("**Engine:** %s\n", engine))
		}
		if location, ok := searchParams["location"].(string); ok && location != "" {
			output.WriteString(fmt.Sprintf("**Location:** %s\n", location))
		}
	}
	output.WriteString("\n")

	// Organic results
	if organicResults, ok := result["organic_results"].([]interface{}); ok && len(organicResults) > 0 {
		output.WriteString(fmt.Sprintf("## Organic Results (%d)\n\n", len(organicResults)))

		for i, r := range organicResults {
			if i >= t.maxResults {
				break
			}

			result, ok := r.(map[string]interface{})
			if !ok {
				continue
			}

			position := i + 1
			if pos, ok := result["position"].(float64); ok {
				position = int(pos)
			}

			title := ""
			if t, ok := result["title"].(string); ok {
				title = t
			}

			link := ""
			if l, ok := result["link"].(string); ok {
				link = l
			}

			snippet := ""
			if s, ok := result["snippet"].(string); ok {
				snippet = s
			}

			output.WriteString(fmt.Sprintf("### %d. %s\n", position, title))
			output.WriteString(fmt.Sprintf("**URL:** %s\n", link))
			if snippet != "" {
				output.WriteString(fmt.Sprintf("%s\n", snippet))
			}
			output.WriteString("\n")
		}
	}

	// Knowledge Graph
	if kg, ok := result["knowledge_graph"].(map[string]interface{}); ok && len(kg) > 0 {
		output.WriteString("## Knowledge Graph\n\n")

		if title, ok := kg["title"].(string); ok {
			output.WriteString(fmt.Sprintf("**Title:** %s\n", title))
		}
		if description, ok := kg["description"].(string); ok {
			output.WriteString(fmt.Sprintf("**Description:** %s\n", description))
		}
		if link, ok := kg["knowledge_graph_search_link"].(string); ok {
			output.WriteString(fmt.Sprintf("**More Info:** %s\n", link))
		}

		// Additional attributes
		for key, value := range kg {
			if key == "title" || key == "description" || key == "knowledge_graph_search_link" {
				continue
			}
			if strVal, ok := value.(string); ok {
				output.WriteString(fmt.Sprintf("- **%s:** %s\n", strings.Title(key), strVal))
			}
		}
		output.WriteString("\n")
	}

	// Related Questions (People Also Ask)
	if relatedQuestions, ok := result["related_questions"].([]interface{}); ok && len(relatedQuestions) > 0 {
		output.WriteString(fmt.Sprintf("## People Also Ask (%d)\n\n", len(relatedQuestions)))

		for i, q := range relatedQuestions {
			if i >= 5 { // Limit to 5 questions
				break
			}

			question, ok := q.(map[string]interface{})
			if !ok {
				continue
			}

			if questionText, ok := question["question"].(string); ok {
				output.WriteString(fmt.Sprintf("**Q:** %s\n", questionText))
			}
			if snippet, ok := question["snippet"].(string); ok {
				output.WriteString(fmt.Sprintf("**A:** %s\n", snippet))
			}
			if link, ok := question["link"].(string); ok {
				output.WriteString(fmt.Sprintf("*Source: %s*\n", link))
			}
			output.WriteString("\n")
		}
	}

	// Related Searches
	if relatedSearches, ok := result["related_searches"].([]interface{}); ok && len(relatedSearches) > 0 {
		output.WriteString("## Related Searches\n\n")

		for i, s := range relatedSearches {
			if i >= 10 { // Limit to 10
				break
			}

			search, ok := s.(map[string]interface{})
			if !ok {
				continue
			}

			if query, ok := search["query"].(string); ok {
				output.WriteString(fmt.Sprintf("- %s\n", query))
			}
		}
		output.WriteString("\n")
	}

	return &ToolResult{
		ForLLM:  output.String(),
		ForUser: output.String(),
	}
}
