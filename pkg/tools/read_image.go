package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/h2non/filetype"
)

// ReadImageTool reads local image files and converts them to base64-encoded data URLs.
// This enables local images to be recognized by LLMs for image analysis tasks.
type ReadImageTool struct {
	maxSize int64
}

// NewReadImageTool creates a new ReadImageTool instance with the specified max file size.
// If maxSize is 0 or negative, defaults to 10MB.
func NewReadImageTool(maxSize int64) *ReadImageTool {
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024 // Default 10MB
	}
	return &ReadImageTool{
		maxSize: maxSize,
	}
}

// Name returns the tool name.
func (t *ReadImageTool) Name() string {
	return "read_image"
}

// Description returns the tool description for LLM function calling.
func (t *ReadImageTool) Description() string {
	return "Read a local image file and convert it to base64-encoded format for LLM image recognition and analysis. " +
		"Supports common formats: jpg, jpeg, png, gif, webp, bmp. " +
		"The image will be sent to the LLM for content analysis."
}

// Parameters returns the JSON Schema parameter definition for the tool.
func (t *ReadImageTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Full path to the local image file. Supports jpg, jpeg, png, gif, webp, bmp formats.",
			},
		},
		"required": []string{"path"},
	}
}

// Execute reads the image file, validates it, and converts to base64 data URL.
// Returns a ToolResult with Media containing the base64 data and MediaDispatch set to MediaDispatchToLLM.
func (t *ReadImageTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	// Parse path parameter
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return ErrorResult("path parameter is required and must be a string")
	}

	// Check file existence
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrorResult(fmt.Sprintf("file not found: %s", path))
		}
		return ErrorResult(fmt.Sprintf("failed to access file: %v", err))
	}

	// Check file size
	if info.Size() > t.maxSize {
		return ErrorResult(fmt.Sprintf(
			"file too large: %d bytes (max: %d bytes, ~%d MB)",
			info.Size(), t.maxSize, t.maxSize/(1024*1024),
		))
	}

	// Detect MIME type
	mime, err := detectImageMIME(path)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to detect file type: %v", err))
	}

	// Validate it's an image
	if !strings.HasPrefix(mime, "image/") {
		return ErrorResult(fmt.Sprintf("not an image file: %s (detected: %s)", path, mime))
	}

	// Encode to base64 data URL
	dataURL, err := encodeImageToDataURL(path, mime, info, int(t.maxSize))
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to encode image: %v", err))
	}

	if dataURL == "" {
		return ErrorResult("failed to encode image: empty result")
	}

	// Build result with MediaDispatch set to send to LLM
	return &ToolResult{
		ForLLM:        fmt.Sprintf("Image loaded successfully: %s (%s, %d bytes)", path, mime, info.Size()),
		ForUser:       fmt.Sprintf("Image loaded: %s", path),
		Media:         []string{dataURL},
		MediaDispatch: MediaDispatchToLLM,
		Silent:        false,
		IsError:       false,
	}
}

// detectImageMIME detects the MIME type of an image file using magic bytes.
func detectImageMIME(path string) (string, error) {
	kind, err := filetype.MatchFile(path)
	if err != nil {
		return "", err
	}
	if kind == filetype.Unknown {
		return "", fmt.Errorf("unknown file type")
	}
	return kind.MIME.Value, nil
}

// encodeImageToDataURL encodes an image file to a base64 data URL.
// Uses streaming encoding for memory efficiency with large files.
func encodeImageToDataURL(localPath, mime string, info os.FileInfo, maxSize int) (string, error) {
	if info.Size() > int64(maxSize) {
		return "", fmt.Errorf("file too large: %d bytes", info.Size())
	}

	f, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	prefix := "data:" + mime + ";base64,"
	encodedLen := base64.StdEncoding.EncodedLen(int(info.Size()))
	var buf bytes.Buffer
	buf.Grow(len(prefix) + encodedLen)
	buf.WriteString(prefix)

	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	if _, err := io.Copy(encoder, f); err != nil {
		return "", err
	}
	encoder.Close()

	return buf.String(), nil
}
