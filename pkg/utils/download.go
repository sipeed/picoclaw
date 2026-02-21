package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// DownloadToFile streams an HTTP response body to a temporary file in small
// chunks (~32KB), keeping peak memory usage constant regardless of file size.
//
// Parameters:
//   - ctx:      context for cancellation/timeout
//   - client:   HTTP client to use (caller controls timeouts, transport, etc.)
//   - req:      fully prepared *http.Request (method, URL, headers, etc.)
//   - maxBytes: maximum bytes to download; 0 means no limit
//
// Returns the path to the temporary file. The caller is responsible for
// removing it when done (defer os.Remove(path)).
//
// On any error the temp file is cleaned up automatically.
func DownloadToFile(ctx context.Context, client *http.Client, req *http.Request, maxBytes int64) (string, error) {
	// Attach context.
	req = req.WithContext(ctx)

	logger.DebugCF("download", "Starting download", map[string]any{
		"url":       req.URL.String(),
		"max_bytes": maxBytes,
	})

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read a small amount for the error message.
		errBody := make([]byte, 512)
		n, _ := io.ReadFull(resp.Body, errBody)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(errBody[:n]))
	}

	// Create temp file.
	tmpFile, err := os.CreateTemp("", "picoclaw-dl-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	logger.DebugCF("download", "Streaming to temp file", map[string]any{
		"path": tmpPath,
	})

	// Cleanup helper — removes the temp file on any error.
	cleanup := func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}

	// Optionally limit the download size.
	var src io.Reader = resp.Body
	if maxBytes > 0 {
		src = io.LimitReader(resp.Body, maxBytes+1) // +1 to detect overflow
	}

	written, err := io.Copy(tmpFile, src)
	if err != nil {
		cleanup()
		return "", fmt.Errorf("download write failed: %w", err)
	}

	if maxBytes > 0 && written > maxBytes {
		cleanup()
		return "", fmt.Errorf("download too large: %d bytes (max %d)", written, maxBytes)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	logger.DebugCF("download", "Download complete", map[string]any{
		"path":          tmpPath,
		"bytes_written": written,
	})

	return tmpPath, nil
}
