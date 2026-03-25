package tools

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/h2non/filetype"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
)

const (
	sendFileTempMaxAge    = 1 * time.Hour
	sendFileDownloadLimit = 50 * 1024 * 1024 // 50 MB download limit
)

// sendFileTempDir returns the directory for URL-downloaded temp files.
// Uses the shared media temp directory so that MediaStore can resolve paths
// after Execute returns (MediaStore only stores a path reference, not a copy).
func sendFileTempDir() string {
	return filepath.Join(media.TempDir(), "sendfile")
}

// SendFileTool allows the LLM to send a local file (image, document, etc.)
// to the user on the current chat channel via the MediaStore pipeline.
type SendFileTool struct {
	workspace   string
	restrict    bool
	maxFileSize int
	mediaStore  media.MediaStore
	allowPaths  []*regexp.Regexp

	defaultChannel string
	defaultChatID  string
}

func NewSendFileTool(
	workspace string,
	restrict bool,
	maxFileSize int,
	store media.MediaStore,
	allowPaths ...[]*regexp.Regexp,
) *SendFileTool {
	if maxFileSize <= 0 {
		maxFileSize = config.DefaultMaxMediaSize
	}
	var patterns []*regexp.Regexp
	if len(allowPaths) > 0 {
		patterns = allowPaths[0]
	}
	return &SendFileTool{
		workspace:   workspace,
		restrict:    restrict,
		maxFileSize: maxFileSize,
		mediaStore:  store,
		allowPaths:  patterns,
	}
}

func (t *SendFileTool) Name() string { return "send_file" }
func (t *SendFileTool) Description() string {
	return "Send a file to the user on the current chat channel. Accepts a local file path or an HTTP(S) URL (the URL will be downloaded automatically)."
}

func (t *SendFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to a local file or an HTTP(S) URL. Relative paths are resolved from workspace. URLs are downloaded automatically.",
			},
			"filename": map[string]any{
				"type":        "string",
				"description": "Optional display filename. Defaults to the basename of path.",
			},
		},
		"required": []string{"path"},
	}
}

func (t *SendFileTool) SetContext(channel, chatID string) {
	t.defaultChannel = channel
	t.defaultChatID = chatID
}

func (t *SendFileTool) SetMediaStore(store media.MediaStore) {
	t.mediaStore = store
}

func (t *SendFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["path"].(string)
	if strings.TrimSpace(path) == "" {
		return ErrorResult("path is required")
	}

	// Prefer context-injected channel/chatID (set by ExecuteWithContext), fall back to SetContext values.
	channel := ToolChannel(ctx)
	if channel == "" {
		channel = t.defaultChannel
	}
	chatID := ToolChatID(ctx)
	if chatID == "" {
		chatID = t.defaultChatID
	}
	if channel == "" || chatID == "" {
		return ErrorResult("no target channel/chat available")
	}

	if t.mediaStore == nil {
		return ErrorResult("media store not configured")
	}

	// Handle URL downloads
	var resolved string
	var dlResult downloadResult
	var isURL bool
	if isHTTPURL(path) {
		var err error
		dlResult, err = downloadToTemp(ctx, path)
		if err != nil {
			return ErrorResult(fmt.Sprintf("download failed: %v", err))
		}
		resolved = dlResult.Path
		isURL = true
		// Do NOT delete the temp file here — MediaStore only stores a path
		// reference, so the file must survive until the channel sends it.
		// CleanupTempFiles() handles stale files periodically.
	} else {
		var err error
		resolved, err = validatePathWithAllowPaths(path, t.workspace, t.restrict, t.allowPaths)
		if err != nil {
			return ErrorResult(fmt.Sprintf("invalid path: %v", err))
		}
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return ErrorResult(fmt.Sprintf("file not found: %v", err))
	}
	if info.IsDir() {
		return ErrorResult("path is a directory, expected a file")
	}
	if info.Size() > int64(t.maxFileSize) {
		return ErrorResult(fmt.Sprintf(
			"file too large: %d bytes (max %d bytes)",
			info.Size(), t.maxFileSize,
		))
	}

	filename, _ := args["filename"].(string)
	if filename == "" {
		if isURL {
			filename = filenameForDownload(path, dlResult.ContentDisposition, dlResult.ContentType)
		} else {
			filename = filepath.Base(resolved)
		}
	}

	mediaType := detectMediaType(resolved)
	// If magic bytes and extension both failed, use Content-Type from HTTP response
	if isURL && mediaType == "application/octet-stream" && dlResult.ContentType != "" {
		mediaType = dlResult.ContentType
	}
	scope := fmt.Sprintf("tool:send_file:%s:%s", channel, chatID)

	ref, err := t.mediaStore.Store(resolved, media.MediaMeta{
		Filename:      filename,
		ContentType:   mediaType,
		Source:        "tool:send_file",
		CleanupPolicy: media.CleanupPolicyForgetOnly,
	}, scope)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to register media: %v", err))
	}

	return MediaResult(fmt.Sprintf("File %q sent to user", filename), []string{ref})
}

// isHTTPURL returns true if the path looks like an HTTP(S) URL.
func isHTTPURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// downloadResult holds the downloaded temp file path and HTTP response metadata.
type downloadResult struct {
	Path               string
	ContentType        string // from Content-Type header (media type only, no params)
	ContentDisposition string // raw Content-Disposition header
}

// downloadToTemp downloads a URL to a temporary file under sendFileTempDir().
// The file must persist until the channel has sent it; cleanup is handled by
// CleanupTempFiles (periodic) rather than the caller.
func downloadToTemp(ctx context.Context, rawURL string) (downloadResult, error) {
	dir := sendFileTempDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return downloadResult{}, fmt.Errorf("create temp dir: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return downloadResult{}, fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return downloadResult{}, fmt.Errorf("HTTP GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return downloadResult{}, fmt.Errorf("HTTP %d %s", resp.StatusCode, resp.Status)
	}

	// Extract Content-Type (media type only)
	respCT := resp.Header.Get("Content-Type")
	mediaType, _, _ := mime.ParseMediaType(respCT)

	// Determine file extension: prefer URL path, then Content-Type header
	ext := extFromURL(rawURL)
	if ext == "" && mediaType != "" {
		ext = preferredExtension(mediaType)
	}

	// Generate a random temp filename to avoid collisions
	var randBytes [8]byte
	if _, err := rand.Read(randBytes[:]); err != nil {
		return downloadResult{}, err
	}
	tmpPath := filepath.Join(dir, fmt.Sprintf("dl_%x%s", randBytes, ext))

	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return downloadResult{}, fmt.Errorf("create temp file: %w", err)
	}

	n, copyErr := io.Copy(f, io.LimitReader(resp.Body, sendFileDownloadLimit+1))
	if closeErr := f.Close(); closeErr != nil && copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		os.Remove(tmpPath)
		return downloadResult{}, fmt.Errorf("write temp file: %w", copyErr)
	}
	if n > sendFileDownloadLimit {
		os.Remove(tmpPath)
		return downloadResult{}, fmt.Errorf("download exceeds %d byte limit", sendFileDownloadLimit)
	}

	return downloadResult{
		Path:               tmpPath,
		ContentType:        mediaType,
		ContentDisposition: resp.Header.Get("Content-Disposition"),
	}, nil
}

// extFromURL extracts a file extension from a URL path, e.g. ".jpg".
func extFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return filepath.Ext(u.Path)
}

// filenameForDownload derives a display filename using Content-Disposition,
// the URL path, and Content-Type as progressive fallbacks.
func filenameForDownload(rawURL, contentDisposition, contentType string) string {
	// 1. Try Content-Disposition header
	if contentDisposition != "" {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err == nil {
			if fn := params["filename"]; fn != "" {
				return fn
			}
		}
	}

	// 2. Try URL path basename
	u, err := url.Parse(rawURL)
	if err == nil {
		base := filepath.Base(u.Path)
		if base != "" && base != "." && base != "/" {
			// If basename has no extension, try to add one from Content-Type
			if filepath.Ext(base) == "" && contentType != "" {
				if ext := preferredExtension(contentType); ext != "" {
					return base + ext
				}
			}
			return base
		}
	}

	// 3. Fallback
	if contentType != "" {
		if ext := preferredExtension(contentType); ext != "" {
			return "download" + ext
		}
	}
	return "download"
}

// preferredExtension returns a file extension (with dot) for a MIME type.
// Uses a short map of common types for deterministic results, then falls back
// to mime.ExtensionsByType.
func preferredExtension(mediaType string) string {
	// mime.ExtensionsByType returns multiple options in undefined order;
	// hardcode the most common ones for determinism.
	preferred := map[string]string{
		"image/png":       ".png",
		"image/jpeg":      ".jpg",
		"image/gif":       ".gif",
		"image/webp":      ".webp",
		"image/svg+xml":   ".svg",
		"image/bmp":       ".bmp",
		"image/tiff":      ".tiff",
		"video/mp4":       ".mp4",
		"video/webm":      ".webm",
		"audio/mpeg":      ".mp3",
		"audio/ogg":       ".ogg",
		"application/pdf": ".pdf",
		"application/zip": ".zip",
		"text/plain":      ".txt",
		"text/html":       ".html",
		"application/json": ".json",
	}
	if ext, ok := preferred[mediaType]; ok {
		return ext
	}
	exts, err := mime.ExtensionsByType(mediaType)
	if err == nil && len(exts) > 0 {
		return exts[0]
	}
	return ""
}

// CleanupTempFiles removes old temp files from the sendfile temp directory.
// Files older than sendFileTempMaxAge are deleted.
func CleanupTempFiles() {
	dir := sendFileTempDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-sendFileTempMaxAge)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// detectMediaType determines the MIME type of a file.
// Uses magic-bytes detection (h2non/filetype) first, then falls back to
// extension-based lookup via mime.TypeByExtension.
func detectMediaType(path string) string {
	kind, err := filetype.MatchFile(path)
	if err == nil && kind != filetype.Unknown {
		return kind.MIME.Value
	}

	if ext := filepath.Ext(path); ext != "" {
		if t := mime.TypeByExtension(ext); t != "" {
			return t
		}
	}

	return "application/octet-stream"
}
