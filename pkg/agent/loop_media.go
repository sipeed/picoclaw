// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"bytes"
	"encoding/base64"
	"io"
	"os"
	"strings"

	"github.com/h2non/filetype"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// resolveMediaRefs replaces media:// refs in message Media fields with base64 data URLs.
// Uses streaming base64 encoding (file handle → encoder → buffer) to avoid holding
// both raw bytes and encoded string in memory simultaneously.
// Returns a new slice; original messages are not mutated.
func resolveMediaRefs(messages []providers.Message, store media.MediaStore, maxSize int) []providers.Message {
	if store == nil {
		return messages
	}

	result := make([]providers.Message, len(messages))
	copy(result, messages)

	for i, m := range result {
		if len(m.Media) == 0 {
			continue
		}

		resolved := make([]string, 0, len(m.Media))
		for _, ref := range m.Media {
			if !strings.HasPrefix(ref, "media://") {
				resolved = append(resolved, ref)
				continue
			}

			localPath, meta, err := store.ResolveWithMeta(ref)
			if err != nil {
				logger.WarnCF("agent", "Failed to resolve media ref", map[string]any{
					"ref":   ref,
					"error": err.Error(),
				})
				continue
			}

			info, err := os.Stat(localPath)
			if err != nil {
				logger.WarnCF("agent", "Failed to stat media file", map[string]any{
					"path":  localPath,
					"error": err.Error(),
				})
				continue
			}
			if info.Size() > int64(maxSize) {
				logger.WarnCF("agent", "Media file too large, skipping", map[string]any{
					"path":     localPath,
					"size":     info.Size(),
					"max_size": maxSize,
				})
				continue
			}

			// Determine MIME type: prefer metadata, fallback to magic-bytes detection
			mime := meta.ContentType
			if mime == "" {
				kind, ftErr := filetype.MatchFile(localPath)
				if ftErr != nil || kind == filetype.Unknown {
					logger.WarnCF("agent", "Unknown media type, skipping", map[string]any{
						"path": localPath,
					})
					continue
				}
				mime = kind.MIME.Value
			}

			// Streaming base64: open file → base64 encoder → buffer
			// Peak memory: ~1.33x file size (buffer only, no raw bytes copy)
			f, err := os.Open(localPath)
			if err != nil {
				logger.WarnCF("agent", "Failed to open media file", map[string]any{
					"path":  localPath,
					"error": err.Error(),
				})
				continue
			}

			prefix := "data:" + mime + ";base64,"
			encodedLen := base64.StdEncoding.EncodedLen(int(info.Size()))
			var buf bytes.Buffer
			buf.Grow(len(prefix) + encodedLen)
			buf.WriteString(prefix)

			encoder := base64.NewEncoder(base64.StdEncoding, &buf)
			if _, err := io.Copy(encoder, f); err != nil {
				f.Close()
				logger.WarnCF("agent", "Failed to encode media file", map[string]any{
					"path":  localPath,
					"error": err.Error(),
				})
				continue
			}
			encoder.Close()
			f.Close()

			resolved = append(resolved, buf.String())
		}

		result[i].Media = resolved
	}

	return result
}
