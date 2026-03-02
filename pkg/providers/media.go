package providers

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

// maxImageFileSize is the maximum raw file size for inline base64 images (5 MB).
const maxImageFileSize = 5 * 1024 * 1024

// supportedImageExts maps lowercase file extensions to MIME types.
var supportedImageExts = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
}

// LoadMediaAsContentParts converts a list of local file paths (or URLs)
// into ContentBlock slices suitable for multimodal LLM requests.
//
// - Local files are read, base64-encoded, and wrapped in a data URL.
// - http(s) URLs are passed through as image_url blocks directly.
// - Non-image files and files exceeding maxImageFileSize are skipped.
func LoadMediaAsContentParts(paths []string) []protocoltypes.ContentBlock {
	var parts []protocoltypes.ContentBlock

	for _, p := range paths {
		// HTTP(S) URL — pass through
		if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
			parts = append(parts, protocoltypes.ContentBlock{
				Type:     "image_url",
				ImageURL: &protocoltypes.ImageURL{URL: p},
			})
			continue
		}

		// Local file
		ext := strings.ToLower(filepath.Ext(p))
		mimeType, ok := supportedImageExts[ext]
		if !ok {
			logger.DebugCF("media", "Skipping non-image file", map[string]any{"path": p, "ext": ext})
			continue
		}

		info, err := os.Stat(p)
		if err != nil {
			logger.WarnCF("media", "Cannot stat media file", map[string]any{"path": p, "error": err.Error()})
			continue
		}
		if info.Size() > maxImageFileSize {
			logger.WarnCF("media", "Skipping oversized image", map[string]any{
				"path":      p,
				"size":      info.Size(),
				"max_bytes": maxImageFileSize,
			})
			continue
		}

		data, err := os.ReadFile(p)
		if err != nil {
			logger.WarnCF("media", "Cannot read media file", map[string]any{"path": p, "error": err.Error()})
			continue
		}

		dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data))
		parts = append(parts, protocoltypes.ContentBlock{
			Type:     "image_url",
			ImageURL: &protocoltypes.ImageURL{URL: dataURL},
		})

		logger.DebugCF("media", "Loaded image", map[string]any{
			"path":      p,
			"mime_type": mimeType,
			"size":      info.Size(),
		})
	}

	return parts
}
