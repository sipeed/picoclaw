package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

// magic bytes for a valid PNG image to fool http.DetectContentType
var validPNGBody = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xf8\x00\x00\x00\x00IEND\xaeB`\x82")

func TestGetMimeType(t *testing.T) {
	tests := []struct {
		path        string
		expected    string
		expectError bool
	}{
		{"test.png", "image/png", false},
		{"TEST.JPG", "image/jpeg", false},
		{"image.jpeg", "image/jpeg", false},
		{"anim.gif", "image/gif", false},
		{"pic.webp", "image/webp", false},
		{"document.pdf", "", true},
		{"no_extension", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			mime, err := getMimeType(tt.path)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for path %s, but got none", tt.path)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for path %s: %v", tt.path, err)
			}
			if mime != tt.expected {
				t.Errorf("Expected mime %s, got %s", tt.expected, mime)
			}
		})
	}
}

func TestAnalyzeImageTool_Basic(t *testing.T) {
	tool := NewAnalyzeImageTool(config.VisionToolConfig{})

	if tool.Name() != "analyze_image" {
		t.Errorf("Expected name 'analyze_image', got %s", tool.Name())
	}

	if !strings.Contains(tool.Description(), "Analyze an image") {
		t.Errorf("Expected description to contain 'Analyze an image', got %s", tool.Description())
	}

	params := tool.Parameters()
	if params["type"] != "object" {
		t.Errorf("Expected parameters to be object type")
	}
}

func TestAnalyzeImageTool_Execute_Validations(t *testing.T) {
	ctx := context.Background()

	t.Run("Missing API Key", func(t *testing.T) {
		tool := NewAnalyzeImageTool(config.VisionToolConfig{ApiKey: ""})
		res := tool.Execute(ctx, map[string]interface{}{"path": "test.png"})
		if !res.IsError || !strings.Contains(res.ForLLM, "missing Vision API key") {
			t.Errorf("Expected missing API key error, got: %v", res.ForLLM)
		}
	})

	t.Run("Missing Path", func(t *testing.T) {
		tool := NewAnalyzeImageTool(config.VisionToolConfig{ApiKey: "test-key"})
		res := tool.Execute(ctx, map[string]interface{}{})
		if !res.IsError || !strings.Contains(res.ForLLM, "path is required") {
			t.Errorf("Expected missing path error, got: %v", res.ForLLM)
		}
	})
}

func TestAnalyzeImageTool_Execute_FileChecks(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	tool := NewAnalyzeImageTool(config.VisionToolConfig{
		ApiKey:    "test-key",
		Workspace: tmpDir,
		Restrict:  true,
	})

	t.Run("Not An Image", func(t *testing.T) {
		fakeImg := filepath.Join(tmpDir, "fake.png")
		os.WriteFile(fakeImg, []byte("this is plain text, not an image"), 0644)

		res := tool.Execute(ctx, map[string]interface{}{"path": fakeImg})
		if !res.IsError || !strings.Contains(res.ForLLM, "not a valid image") {
			t.Errorf("Expected invalid image error, got: %v", res.ForLLM)
		}
	})

	t.Run("Unsupported Extension", func(t *testing.T) {
		unsupportedFile := filepath.Join(tmpDir, "real_image.txt")
		// valid image, but wrong extension
		os.WriteFile(unsupportedFile, validPNGBody, 0644)

		res := tool.Execute(ctx, map[string]interface{}{"path": unsupportedFile})
		if !res.IsError || !strings.Contains(res.ForLLM, "unsupported image extension") {
			t.Errorf("Expected unsupported extension error, got: %v", res.ForLLM)
		}
	})

	t.Run("File Too Large", func(t *testing.T) {
		largeImg := filepath.Join(tmpDir, "large.png")
		// file slightly larger than 10MB
		f, _ := os.Create(largeImg)
		f.Truncate(10*1024*1024 + 100)
		f.Close()

		res := tool.Execute(ctx, map[string]interface{}{"path": largeImg})
		if !res.IsError || !strings.Contains(res.ForLLM, "image is too large") {
			t.Errorf("Expected too large error, got: %v", res.ForLLM)
		}
	})
}

func TestAnalyzeImageTool_Execute_Success(t *testing.T) {
	expectedAnalysis := "I see a test image with a button."
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Header.Get("Authorization") != "Bearer mock-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {"content": "` + expectedAnalysis + `"}
			}]
		}`))
	}))
	defer ts.Close()

	ctx := context.Background()
	tmpDir := t.TempDir()

	tool := NewAnalyzeImageTool(config.VisionToolConfig{
		ApiKey:    "mock-api-key",
		ApiURL:    ts.URL,
		Model:     "gpt-4o-mini",
		Workspace: tmpDir,
		Restrict:  true,
	})

	testImgPath := filepath.Join(tmpDir, "success.png")
	os.WriteFile(testImgPath, validPNGBody, 0644)

	res := tool.Execute(ctx, map[string]interface{}{
		"path":   testImgPath,
		"prompt": "What is this?",
	})

	if res.IsError {
		t.Fatalf("Expected success, got error: %s", res.ForLLM)
	}

	if !strings.Contains(res.ForLLM, expectedAnalysis) {
		t.Errorf("Expected output to contain %q, got: %s", expectedAnalysis, res.ForLLM)
	}
}

func TestAnalyzeImageTool_Execute_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal Server Error"}`))
	}))
	defer ts.Close()

	ctx := context.Background()
	tmpDir := t.TempDir()

	tool := NewAnalyzeImageTool(config.VisionToolConfig{
		ApiKey:    "mock-api-key",
		ApiURL:    ts.URL,
		Workspace: tmpDir,
	})

	testImgPath := filepath.Join(tmpDir, "error.png")
	os.WriteFile(testImgPath, validPNGBody, 0644)

	res := tool.Execute(ctx, map[string]interface{}{"path": testImgPath})

	if !res.IsError {
		t.Fatal("Expected API error, got success")
	}

	if !strings.Contains(res.ForLLM, "vision API returned error (500)") {
		t.Errorf("Expected 500 error message, got: %s", res.ForLLM)
	}
}
