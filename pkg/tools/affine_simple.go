package tools

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
)

// AffineSimpleTool provides access to Affine workspace via HTTP MCP endpoint
type AffineSimpleTool struct {
	mcpEndpoint string
	apiKey      string
	workspaceID string
	httpClient  *http.Client
}

// AffineSimpleToolOptions configures the Affine simple tool
type AffineSimpleToolOptions struct {
	MCPEndpoint    string
	APIKey         string
	WorkspaceID    string
	TimeoutSeconds int
}

// NewAffineSimpleTool creates a new Affine simple tool instance
func NewAffineSimpleTool(opts AffineSimpleToolOptions) *AffineSimpleTool {
	timeout := time.Duration(opts.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &AffineSimpleTool{
		mcpEndpoint: opts.MCPEndpoint,
		apiKey:      opts.APIKey,
		workspaceID: opts.WorkspaceID,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (t *AffineSimpleTool) Name() string {
	return "affine"
}

func (t *AffineSimpleTool) Description() string {
	return "Manage and search documents in your Affine workspace. List, search, read documents and manage tags."
}

func (t *AffineSimpleTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": []string{
					"list_docs",
					"search",
					"semantic_search",
					"read",
					"get_doc",
					"export_markdown",
				},
				"description": "Action: 'list_docs' to list all documents, 'search' for keyword search, 'semantic_search' for meaning-based search, 'read' to get document content, 'get_doc' for metadata, 'export_markdown' to export as markdown",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Search query (for search actions) or document ID (for read/get_doc/export_markdown actions). Optional for list_docs.",
			},
			"limit": map[string]any{
				"type":        "number",
				"description": "Maximum number of documents to return (for list_docs). Default: 20",
			},
			"skip": map[string]any{
				"type":        "number",
				"description": "Number of documents to skip (for pagination in list_docs). Default: 0",
			},
		},
		"required": []string{"action"},
	}
}

func (t *AffineSimpleTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}

	switch action {
	case "list_docs":
		limit := 20
		skip := 0
		if l, ok := args["limit"].(float64); ok {
			limit = int(l)
		}
		if s, ok := args["skip"].(float64); ok {
			skip = int(s)
		}
		return t.listDocs(ctx, limit, skip)
	case "search":
		query, ok := args["query"].(string)
		if !ok {
			return ErrorResult("query is required for search")
		}
		return t.search(ctx, query)
	case "semantic_search":
		query, ok := args["query"].(string)
		if !ok {
			return ErrorResult("query is required for semantic_search")
		}
		return t.semanticSearch(ctx, query)
	case "read":
		query, ok := args["query"].(string)
		if !ok {
			return ErrorResult("query (document ID) is required for read")
		}
		return t.read(ctx, query)
	case "get_doc":
		query, ok := args["query"].(string)
		if !ok {
			return ErrorResult("query (document ID) is required for get_doc")
		}
		return t.getDoc(ctx, query)
	case "export_markdown":
		query, ok := args["query"].(string)
		if !ok {
			return ErrorResult("query (document ID) is required for export_markdown")
		}
		return t.exportMarkdown(ctx, query)
	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

func (t *AffineSimpleTool) search(ctx context.Context, query string) *ToolResult {
	// Call MCP endpoint with search request
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "keyword_search",
			"arguments": map[string]interface{}{
				"query": query,
			},
		},
	}

	result, err := t.callMCP(ctx, reqBody)
	if err != nil {
		return ErrorResult(fmt.Sprintf("search failed: %v", err))
	}

	// Parse search results
	type SearchDoc struct {
		DocID     string `json:"docId"`
		Title     string `json:"title"`
		Snippet   string `json:"snippet,omitempty"`
		CreatedAt string `json:"createdAt,omitempty"`
	}
	var searchResults []SearchDoc

	// Try to extract from result
	if resultMap, ok := result.(map[string]interface{}); ok {
		if content, ok := resultMap["content"].([]interface{}); ok {
			for _, item := range content {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if text, ok := itemMap["text"].(string); ok {
						// Try parsing as single object first
						var doc SearchDoc
						if err := json.Unmarshal([]byte(text), &doc); err == nil {
							searchResults = append(searchResults, doc)
						} else {
							// Try parsing as array
							var docs []SearchDoc
							if err := json.Unmarshal([]byte(text), &docs); err == nil {
								searchResults = append(searchResults, docs...)
							}
						}
					}
				}
			}
		}
	}

	if len(searchResults) == 0 {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("No results found for: %s", query),
			ForUser: fmt.Sprintf("No results found for: %s", query),
		}
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Found %d results for '%s':", len(searchResults), query))
	for i, doc := range searchResults {
		lines = append(lines, fmt.Sprintf("%d. %s (ID: %s)", i+1, doc.Title, doc.DocID))
		if doc.Snippet != "" {
			lines = append(lines, fmt.Sprintf("   %s", doc.Snippet))
		}
	}

	output := strings.Join(lines, "\n")
	return &ToolResult{
		ForLLM:  output,
		ForUser: output,
	}
}

func (t *AffineSimpleTool) semanticSearch(ctx context.Context, query string) *ToolResult {
	// Call MCP endpoint with semantic search request
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "semantic_search",
			"arguments": map[string]interface{}{
				"query": query,
			},
		},
	}

	result, err := t.callMCP(ctx, reqBody)
	if err != nil {
		return ErrorResult(fmt.Sprintf("semantic search failed: %v", err))
	}

	// Parse search results (same format as keyword search)
	type SearchDoc struct {
		DocID     string `json:"docId"`
		Title     string `json:"title"`
		Snippet   string `json:"snippet,omitempty"`
		CreatedAt string `json:"createdAt,omitempty"`
	}
	var searchResults []SearchDoc

	// Try to extract from result
	if resultMap, ok := result.(map[string]interface{}); ok {
		if content, ok := resultMap["content"].([]interface{}); ok {
			for _, item := range content {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if text, ok := itemMap["text"].(string); ok {
						// Try parsing as single object first
						var doc SearchDoc
						if err := json.Unmarshal([]byte(text), &doc); err == nil {
							searchResults = append(searchResults, doc)
						} else {
							// Try parsing as array
							var docs []SearchDoc
							if err := json.Unmarshal([]byte(text), &docs); err == nil {
								searchResults = append(searchResults, docs...)
							}
						}
					}
				}
			}
		}
	}

	if len(searchResults) == 0 {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("No semantic matches found for: %s", query),
			ForUser: fmt.Sprintf("No semantic matches found for: %s", query),
		}
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Found %d semantic matches for '%s':", len(searchResults), query))
	for i, doc := range searchResults {
		lines = append(lines, fmt.Sprintf("%d. %s (ID: %s)", i+1, doc.Title, doc.DocID))
		if doc.Snippet != "" {
			lines = append(lines, fmt.Sprintf("   %s", doc.Snippet))
		}
	}

	output := strings.Join(lines, "\n")
	return &ToolResult{
		ForLLM:  output,
		ForUser: output,
	}
}

func (t *AffineSimpleTool) read(ctx context.Context, docID string) *ToolResult {
	// Call MCP endpoint with read request
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "read_document",
			"arguments": map[string]interface{}{
				"docId": docID,
			},
		},
	}

	result, err := t.callMCP(ctx, reqBody)
	if err != nil {
		return ErrorResult(fmt.Sprintf("read failed: %v", err))
	}

	// Extract content from result
	var content string
	var title string

	if resultMap, ok := result.(map[string]interface{}); ok {
		if contentArray, ok := resultMap["content"].([]interface{}); ok {
			for _, item := range contentArray {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if text, ok := itemMap["text"].(string); ok {
						content += text + "\n"
					}
				}
			}
		}
	}

	if content == "" {
		content = fmt.Sprintf("Document ID: %s\n(Content could not be extracted)", docID)
	}

	output := fmt.Sprintf("Document: %s\n\n%s", title, content)
	return &ToolResult{
		ForLLM:  output,
		ForUser: output,
	}
}

func (t *AffineSimpleTool) callMCP(ctx context.Context, reqBody map[string]interface{}) (interface{}, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.mcpEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Check if response is SSE (text/event-stream)
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return t.parseSSEResponse(resp.Body)
	}

	// Otherwise parse as JSON
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var mcpResp struct {
		Result interface{} `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &mcpResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if mcpResp.Error != nil {
		return nil, fmt.Errorf("MCP error %d: %s", mcpResp.Error.Code, mcpResp.Error.Message)
	}

	return mcpResp.Result, nil
}

func (t *AffineSimpleTool) parseSSEResponse(body io.Reader) (interface{}, error) {
	// Read SSE stream and extract the final JSON-RPC response
	scanner := bufio.NewScanner(body)
	var lastEvent string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			lastEvent = strings.TrimPrefix(line, "data: ")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read SSE stream: %w", err)
	}

	if lastEvent == "" {
		return nil, fmt.Errorf("no data in SSE stream")
	}

	var mcpResp struct {
		Result interface{} `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal([]byte(lastEvent), &mcpResp); err != nil {
		return nil, fmt.Errorf("decode SSE data: %w", err)
	}

	if mcpResp.Error != nil {
		return nil, fmt.Errorf("MCP error %d: %s", mcpResp.Error.Code, mcpResp.Error.Message)
	}

	return mcpResp.Result, nil
}


func (t *AffineSimpleTool) listDocs(ctx context.Context, limit, skip int) *ToolResult {
	// Call MCP endpoint to list documents
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "list_docs",
			"arguments": map[string]interface{}{
				"limit": limit,
				"skip":  skip,
			},
		},
	}

	result, err := t.callMCP(ctx, reqBody)
	if err != nil {
		return ErrorResult(fmt.Sprintf("list_docs failed: %v", err))
	}

	// Parse document list
	type DocInfo struct {
		DocID     string   `json:"docId"`
		Title     string   `json:"title"`
		CreatedAt string   `json:"createdAt,omitempty"`
		UpdatedAt string   `json:"updatedAt,omitempty"`
		Tags      []string `json:"tags,omitempty"`
	}
	var docs []DocInfo

	// Extract from result
	if resultMap, ok := result.(map[string]interface{}); ok {
		if content, ok := resultMap["content"].([]interface{}); ok {
			for _, item := range content {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if text, ok := itemMap["text"].(string); ok {
						// Try parsing as array
						var docList []DocInfo
						if err := json.Unmarshal([]byte(text), &docList); err == nil {
							docs = append(docs, docList...)
						} else {
							// Try parsing as single object
							var doc DocInfo
							if err := json.Unmarshal([]byte(text), &doc); err == nil {
								docs = append(docs, doc)
							}
						}
					}
				}
			}
		}
	}

	if len(docs) == 0 {
		return &ToolResult{
			ForLLM:  "No documents found in workspace",
			ForUser: "No documents found in workspace",
		}
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Found %d documents (limit: %d, skip: %d):", len(docs), limit, skip))
	for i, doc := range docs {
		line := fmt.Sprintf("%d. %s (ID: %s)", i+1, doc.Title, doc.DocID)
		if len(doc.Tags) > 0 {
			line += fmt.Sprintf(" [Tags: %s]", strings.Join(doc.Tags, ", "))
		}
		lines = append(lines, line)
		if doc.CreatedAt != "" {
			lines = append(lines, fmt.Sprintf("   Created: %s", doc.CreatedAt))
		}
	}

	output := strings.Join(lines, "\n")
	return &ToolResult{
		ForLLM:  output,
		ForUser: output,
	}
}

func (t *AffineSimpleTool) getDoc(ctx context.Context, docID string) *ToolResult {
	// Call MCP endpoint to get document metadata
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "get_doc",
			"arguments": map[string]interface{}{
				"docId": docID,
			},
		},
	}

	result, err := t.callMCP(ctx, reqBody)
	if err != nil {
		return ErrorResult(fmt.Sprintf("get_doc failed: %v", err))
	}

	// Parse document metadata
	type DocMetadata struct {
		DocID     string   `json:"docId"`
		Title     string   `json:"title"`
		CreatedAt string   `json:"createdAt,omitempty"`
		UpdatedAt string   `json:"updatedAt,omitempty"`
		Tags      []string `json:"tags,omitempty"`
		IsPublic  bool     `json:"isPublic,omitempty"`
	}
	var metadata DocMetadata

	// Extract from result
	if resultMap, ok := result.(map[string]interface{}); ok {
		if content, ok := resultMap["content"].([]interface{}); ok {
			for _, item := range content {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if text, ok := itemMap["text"].(string); ok {
						json.Unmarshal([]byte(text), &metadata)
						break
					}
				}
			}
		}
	}

	if metadata.DocID == "" {
		return ErrorResult(fmt.Sprintf("Could not retrieve metadata for document: %s", docID))
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Document: %s", metadata.Title))
	lines = append(lines, fmt.Sprintf("ID: %s", metadata.DocID))
	if metadata.CreatedAt != "" {
		lines = append(lines, fmt.Sprintf("Created: %s", metadata.CreatedAt))
	}
	if metadata.UpdatedAt != "" {
		lines = append(lines, fmt.Sprintf("Updated: %s", metadata.UpdatedAt))
	}
	if len(metadata.Tags) > 0 {
		lines = append(lines, fmt.Sprintf("Tags: %s", strings.Join(metadata.Tags, ", ")))
	}
	if metadata.IsPublic {
		lines = append(lines, "Status: Public")
	} else {
		lines = append(lines, "Status: Private")
	}

	output := strings.Join(lines, "\n")
	return &ToolResult{
		ForLLM:  output,
		ForUser: output,
	}
}

func (t *AffineSimpleTool) exportMarkdown(ctx context.Context, docID string) *ToolResult {
	// Call MCP endpoint to export document as markdown
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "export_doc_markdown",
			"arguments": map[string]interface{}{
				"docId": docID,
			},
		},
	}

	result, err := t.callMCP(ctx, reqBody)
	if err != nil {
		return ErrorResult(fmt.Sprintf("export_markdown failed: %v", err))
	}

	// Extract markdown content
	var markdown string

	if resultMap, ok := result.(map[string]interface{}); ok {
		if content, ok := resultMap["content"].([]interface{}); ok {
			for _, item := range content {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if text, ok := itemMap["text"].(string); ok {
						markdown += text
					}
				}
			}
		}
	}

	if markdown == "" {
		return ErrorResult(fmt.Sprintf("Could not export document %s as markdown", docID))
	}

	output := fmt.Sprintf("Markdown export of document %s:\n\n%s", docID, markdown)
	return &ToolResult{
		ForLLM:  output,
		ForUser: output,
	}
}
