package agent

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/KarakuriAgent/clawdroid/pkg/logger"
	"github.com/KarakuriAgent/clawdroid/pkg/providers"
)

// PersistMedia saves base64 data URL images to the mediaDir as files.
// It returns the list of saved file paths. Items that are not data URLs
// (e.g. already file paths) are skipped.
func PersistMedia(media []string, mediaDir string) []string {
	if len(media) == 0 || mediaDir == "" {
		return nil
	}

	var paths []string
	ts := time.Now().Format("20060102_150405")

	for i, item := range media {
		if !strings.HasPrefix(item, "data:") {
			continue
		}

		// Parse data URL: data:<mime>;base64,<data>
		ext, data, err := parseDataURL(item)
		if err != nil {
			logger.WarnCF("media", "Failed to parse data URL",
				map[string]interface{}{"index": i, "error": err.Error()})
			continue
		}

		filename := fmt.Sprintf("%s_%d%s", ts, i, ext)
		filePath := filepath.Join(mediaDir, filename)

		if err := os.WriteFile(filePath, data, 0644); err != nil {
			logger.WarnCF("media", "Failed to write media file",
				map[string]interface{}{"path": filePath, "error": err.Error()})
			continue
		}

		paths = append(paths, filePath)
	}

	return paths
}

// parseDataURL extracts extension and decoded bytes from a data URL.
func parseDataURL(dataURL string) (ext string, data []byte, err error) {
	// Expected format: data:<mime>;base64,<base64data>
	if !strings.HasPrefix(dataURL, "data:") {
		return "", nil, fmt.Errorf("not a data URL")
	}

	commaIdx := strings.Index(dataURL, ",")
	if commaIdx < 0 {
		return "", nil, fmt.Errorf("invalid data URL: no comma separator")
	}

	header := dataURL[5:commaIdx] // after "data:"
	encoded := dataURL[commaIdx+1:]

	// Extract MIME type (before ;base64)
	mime := header
	if idx := strings.Index(header, ";"); idx >= 0 {
		mime = header[:idx]
	}

	ext = mimeToExt(mime)

	data, err = base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, fmt.Errorf("base64 decode failed: %w", err)
	}

	return ext, data, nil
}

// mimeToExt maps common MIME types to file extensions.
func mimeToExt(mime string) string {
	switch strings.ToLower(mime) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/bmp":
		return ".bmp"
	default:
		return ".bin"
	}
}

// imagePathRe matches [Image: <path>] tags embedded in message content.
var imagePathRe = regexp.MustCompile(`\[Image: ([^\]]+)\]`)

// CleanupMediaFiles extracts [Image: <path>] references from messages
// and deletes the corresponding files.
func CleanupMediaFiles(messages []providers.Message) {
	for _, msg := range messages {
		matches := imagePathRe.FindAllStringSubmatch(msg.Content, -1)
		for _, m := range matches {
			path := m[1]
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				logger.WarnCF("media", "Failed to remove media file",
					map[string]interface{}{"path": path, "error": err.Error()})
			}
		}
	}
}
