package utils

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// MediaDir is the subdirectory name under os.TempDir() where downloaded media files are stored.
const MediaDir = "picoclaw_media"

// IsAudioFile checks if a file is an audio file based on its filename extension and content type.
func IsAudioFile(filename, contentType string) bool {
	audioExtensions := []string{".mp3", ".wav", ".ogg", ".m4a", ".flac", ".aac", ".wma"}
	audioTypes := []string{"audio/", "application/ogg", "application/x-ogg"}

	for _, ext := range audioExtensions {
		if strings.HasSuffix(strings.ToLower(filename), ext) {
			return true
		}
	}

	for _, audioType := range audioTypes {
		if strings.HasPrefix(strings.ToLower(contentType), audioType) {
			return true
		}
	}

	return false
}

// SanitizeFilename removes potentially dangerous characters from a filename
// and returns a safe version for local filesystem storage.
func SanitizeFilename(filename string) string {
	// Get the base filename without path
	base := filepath.Base(filename)

	// Remove any directory traversal attempts
	base = strings.ReplaceAll(base, "..", "")
	base = strings.ReplaceAll(base, "/", "_")
	base = strings.ReplaceAll(base, "\\", "_")

	return base
}

// DownloadOptions holds optional parameters for downloading files
type DownloadOptions struct {
	Timeout      time.Duration
	ExtraHeaders map[string]string
	LoggerPrefix string
}

// DownloadFile downloads a file from URL to a local temp directory.
// Returns the local file path or empty string on error.
func DownloadFile(url, filename string, opts DownloadOptions) string {
	// Set defaults
	if opts.Timeout == 0 {
		opts.Timeout = 60 * time.Second
	}
	if opts.LoggerPrefix == "" {
		opts.LoggerPrefix = "utils"
	}

	mediaDir := filepath.Join(os.TempDir(), MediaDir)
	if err := os.MkdirAll(mediaDir, 0o700); err != nil {
		logger.ErrorCF(opts.LoggerPrefix, "Failed to create media directory", map[string]any{
			"error": err.Error(),
		})
		return ""
	}

	// Generate unique filename with UUID prefix to prevent conflicts
	safeName := SanitizeFilename(filename)
	localPath := filepath.Join(mediaDir, uuid.New().String()[:8]+"_"+safeName)

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.ErrorCF(opts.LoggerPrefix, "Failed to create download request", map[string]any{
			"error": err.Error(),
		})
		return ""
	}

	// Add extra headers (e.g., Authorization for Slack)
	for key, value := range opts.ExtraHeaders {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: opts.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		logger.ErrorCF(opts.LoggerPrefix, "Failed to download file", map[string]any{
			"error": err.Error(),
			"url":   url,
		})
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.ErrorCF(opts.LoggerPrefix, "File download returned non-200 status", map[string]any{
			"status": resp.StatusCode,
			"url":    url,
		})
		return ""
	}

	out, err := os.Create(localPath)
	if err != nil {
		logger.ErrorCF(opts.LoggerPrefix, "Failed to create local file", map[string]any{
			"error": err.Error(),
		})
		return ""
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		os.Remove(localPath)
		logger.ErrorCF(opts.LoggerPrefix, "Failed to write file", map[string]any{
			"error": err.Error(),
		})
		return ""
	}

	logger.DebugCF(opts.LoggerPrefix, "File downloaded successfully", map[string]any{
		"path": localPath,
	})

	return localPath
}

// DownloadFileSimple is a simplified version of DownloadFile without options
func DownloadFileSimple(url, filename string) string {
	return DownloadFile(url, filename, DownloadOptions{
		LoggerPrefix: "media",
	})
}

// MediaCleaner periodically removes old files from the media temp directory.
type MediaCleaner struct {
	interval time.Duration
	maxAge   time.Duration
	stop     chan struct{}
	once     sync.Once
}

// NewMediaCleaner creates a new MediaCleaner with the given settings.
// If intervalMinutes or maxAgeMinutes are <= 0, defaults are used (5 and 30 respectively).
func NewMediaCleaner(intervalMinutes, maxAgeMinutes int) *MediaCleaner {
	interval := time.Duration(intervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	maxAge := time.Duration(maxAgeMinutes) * time.Minute
	if maxAge <= 0 {
		maxAge = 30 * time.Minute
	}
	return &MediaCleaner{
		interval: interval,
		maxAge:   maxAge,
		stop:     make(chan struct{}),
	}
}

// Start begins the background cleanup goroutine. Safe to call multiple times.
func (mc *MediaCleaner) Start() {
	mc.once.Do(func() {
		go mc.loop()
		logger.InfoCF("media", "Media cleaner started", map[string]any{
			"interval": mc.interval.String(),
			"max_age":  mc.maxAge.String(),
		})
	})
}

// Stop signals the cleanup goroutine to exit. Safe to call multiple times.
func (mc *MediaCleaner) Stop() {
	select {
	case <-mc.stop:
	default:
		close(mc.stop)
		logger.InfoC("media", "Media cleaner stopped")
	}
}

func (mc *MediaCleaner) loop() {
	ticker := time.NewTicker(mc.interval)
	defer ticker.Stop()
	for {
		select {
		case <-mc.stop:
			return
		case <-ticker.C:
			mc.cleanup()
		}
	}
}

func (mc *MediaCleaner) cleanup() {
	mediaDir := filepath.Join(os.TempDir(), MediaDir)
	entries, err := os.ReadDir(mediaDir)
	if err != nil {
		return
	}

	now := time.Now()
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) > mc.maxAge {
			path := filepath.Join(mediaDir, entry.Name())
			if err := os.Remove(path); err == nil {
				removed++
			}
		}
	}
	if removed > 0 {
		logger.DebugCF("media", "Cleaned up old media files", map[string]any{
			"removed": removed,
		})
	}
}
