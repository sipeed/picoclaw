//go:build integration

package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sipeed/picoclaw/pkg/config"
)

// Run with: go test -tags=integration ./pkg/mcp
func TestIntegration_StreamableHTTPCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name                    string
		transportType           string
		jsonResponse            bool
		rejectStandaloneGET     bool
		wantResponseContentType string
	}{
		{
			name:                    "http/json-only-without-get-listener",
			transportType:           "http",
			jsonResponse:            true,
			rejectStandaloneGET:     true,
			wantResponseContentType: "application/json",
		},
		{
			name:                    "http/streaming-post-responses",
			transportType:           "http",
			jsonResponse:            false,
			rejectStandaloneGET:     false,
			wantResponseContentType: "text/event-stream",
		},
		{
			name:                    "streamable-http-alias/json-only-without-get-listener",
			transportType:           "streamable-http",
			jsonResponse:            true,
			rejectStandaloneGET:     true,
			wantResponseContentType: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server, recorder := newRecordedGoSDKStreamableServer(t, tt.jsonResponse, tt.rejectStandaloneGET)
			defer server.Close()

			mgr := NewManager()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := mgr.ConnectServer(ctx, "compat", config.MCPServerConfig{
				Enabled: true,
				Type:    tt.transportType,
				URL:     server.URL,
				Headers: map[string]string{
					"Authorization": "Bearer integration-token",
				},
			})
			if err != nil {
				t.Fatalf("ConnectServer() error = %v", err)
			}

			tools := mgr.GetAllTools()
			if got := len(tools["compat"]); got != 1 {
				t.Fatalf("len(GetAllTools()[\"compat\"]) = %d, want 1", got)
			}

			result, err := mgr.CallTool(ctx, "compat", "echo", map[string]any{
				"message": "hello from integration",
			})
			if err != nil {
				t.Fatalf("CallTool() error = %v", err)
			}
			if got, want := extractTextResult(t, result), "hello from integration"; got != want {
				t.Fatalf("CallTool() text = %q, want %q", got, want)
			}

			if err := mgr.Close(); err != nil {
				t.Fatalf("Manager.Close() error = %v", err)
			}

			assertRecordedCompatibility(t, recorder.snapshot(), tt.wantResponseContentType)
		})
	}
}

type recordedRequest struct {
	Method              string
	Path                string
	JSONRPCMethod       string
	RequestSessionID    string
	Authorization       string
	ResponseStatusCode  int
	ResponseContentType string
}

type requestRecorder struct {
	mu       sync.Mutex
	requests []recordedRequest
}

func (r *requestRecorder) add(req recordedRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, req)
}

func (r *requestRecorder) snapshot() []recordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]recordedRequest, len(r.requests))
	copy(out, r.requests)
	return out
}

func newRecordedGoSDKStreamableServer(
	t *testing.T,
	jsonResponse bool,
	rejectStandaloneGET bool,
) (*httptest.Server, *requestRecorder) {
	t.Helper()

	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "streamable-integration-server",
		Version: "1.0.0",
	}, nil)
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "echo",
		Description: "Echo a message",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, args map[string]any) (*sdkmcp.CallToolResult, any, error) {
		message, _ := args["message"].(string)
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{
				&sdkmcp.TextContent{Text: message},
			},
		}, nil, nil
	})

	recorder := &requestRecorder{}
	handler := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server {
		return server
	}, &sdkmcp.StreamableHTTPOptions{
		JSONResponse: jsonResponse,
	})

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rejectStandaloneGET && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			recorder.add(recordedRequest{
				Method:              r.Method,
				Path:                r.URL.Path,
				RequestSessionID:    r.Header.Get("Mcp-Session-Id"),
				Authorization:       r.Header.Get("Authorization"),
				ResponseStatusCode:  http.StatusMethodNotAllowed,
				ResponseContentType: normalizeContentType(w.Header().Get("Content-Type")),
			})
			return
		}

		recorded := recordedRequest{
			Method:           r.Method,
			Path:             r.URL.Path,
			RequestSessionID: r.Header.Get("Mcp-Session-Id"),
			Authorization:    r.Header.Get("Authorization"),
		}
		if r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("reading request body: %v", err)
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			var envelope struct {
				Method string `json:"method"`
			}
			if err := json.Unmarshal(body, &envelope); err == nil {
				recorded.JSONRPCMethod = envelope.Method
			}
		}

		rw := &recordingResponseWriter{ResponseWriter: w}
		handler.ServeHTTP(rw, r)

		recorded.ResponseStatusCode = rw.statusCode()
		recorded.ResponseContentType = normalizeContentType(rw.Header().Get("Content-Type"))
		recorder.add(recorded)
	}))

	return httpServer, recorder
}

type recordingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *recordingResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *recordingResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(p)
}

func (w *recordingResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *recordingResponseWriter) statusCode() int {
	if w.status != 0 {
		return w.status
	}
	return http.StatusOK
}

func extractTextResult(t *testing.T, result *sdkmcp.CallToolResult) string {
	t.Helper()
	if result == nil || len(result.Content) != 1 {
		t.Fatalf("unexpected CallToolResult: %#v", result)
	}
	text, ok := result.Content[0].(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("CallToolResult content type = %T, want *sdkmcp.TextContent", result.Content[0])
	}
	return text.Text
}

func assertRecordedCompatibility(
	t *testing.T,
	requests []recordedRequest,
	wantResponseContentType string,
) {
	t.Helper()

	var (
		getCount                     int
		deleteCount                  int
		postWithSession              int
		deleteWithSession            int
		requestsMissingAuth          []string
		observedContentTypesByMethod = map[string]string{}
	)

	for _, req := range requests {
		switch req.Method {
		case http.MethodGet:
			getCount++
		case http.MethodPost:
			if req.RequestSessionID != "" {
				postWithSession++
			}
			if req.JSONRPCMethod != "" && observedContentTypesByMethod[req.JSONRPCMethod] == "" {
				observedContentTypesByMethod[req.JSONRPCMethod] = req.ResponseContentType
			}
		case http.MethodDelete:
			deleteCount++
			if req.RequestSessionID != "" {
				deleteWithSession++
			}
		}

		if req.Authorization != "Bearer integration-token" {
			requestsMissingAuth = append(requestsMissingAuth, req.Method+" "+req.Path)
		}
	}

	if getCount != 0 {
		t.Fatalf("expected no standalone GET requests for streamable HTTP mode, saw %d", getCount)
	}
	if deleteCount != 1 {
		t.Fatalf("DELETE count = %d, want 1", deleteCount)
	}
	if postWithSession == 0 {
		t.Fatal("expected at least one POST request with Mcp-Session-Id")
	}
	if deleteWithSession != 1 {
		t.Fatalf("expected exactly one DELETE with Mcp-Session-Id, got %d", deleteWithSession)
	}
	if len(requestsMissingAuth) > 0 {
		t.Fatalf("Authorization header missing on requests: %v", requestsMissingAuth)
	}

	for _, method := range []string{"initialize", "tools/list", "tools/call"} {
		if observedContentTypesByMethod[method] == "" {
			t.Fatalf("did not observe POST response for JSON-RPC method %q", method)
		}
		if observedContentTypesByMethod[method] != wantResponseContentType {
			t.Fatalf(
				"response content-type for %s = %q, want %q",
				method,
				observedContentTypesByMethod[method],
				wantResponseContentType,
			)
		}
	}
}

func normalizeContentType(value string) string {
	return strings.TrimSpace(strings.SplitN(value, ";", 2)[0])
}
