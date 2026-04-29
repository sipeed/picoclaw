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

// resolveMediaRefs resolves media:// refs in messages.
// For user messages: images get path tags only ([image:/path]) so the LLM
// can decide whether to view them via load_image or operate on the file.
// For tool messages: images are base64-encoded and appended as a synthetic
// user message (many APIs don't support image_url in tool messages).
// Non-image files always get path tags regardless of role.
// Returns a new slice; original messages are not mutated.
func resolveMediaRefs(messages []providers.Message, store media.MediaStore, maxSize int) []providers.Message {
	if store == nil {
		return messages
	}

	result := make([]providers.Message, 0, len(messages))

	for _, m := range messages {
		if len(m.Media) == 0 {
			result = append(result, m)
			continue
		}

		msg := m
		resolved := make([]string, 0, len(m.Media))
		var pathTags []string
		var toolImageDataURLs []string

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

			mime := detectMIME(localPath, meta)
			pathTags = append(pathTags, buildPathTag(mime, localPath))

			// For tool results (e.g. load_image), base64-encode images into a
			// separate user message — many LLM APIs don't support image_url in
			// tool messages.
			if m.Role == "tool" && strings.HasPrefix(mime, "image/") {
				dataURL := encodeImageToDataURL(localPath, mime, info, maxSize)
				if dataURL != "" {
					toolImageDataURLs = append(toolImageDataURLs, dataURL)
				}
			}
		}

		msg.Media = resolved
		if len(pathTags) > 0 {
			msg.Content = injectPathTags(msg.Content, pathTags)
		}
		result = append(result, msg)

		// Append a synthetic user message carrying the image data so the LLM
		// can see it (tool messages don't support image_url in most APIs).
		if len(toolImageDataURLs) > 0 {
			result = append(result, providers.Message{
				Role:  "user",
				Content: "[Loaded image from tool result above]",
				Media: toolImageDataURLs,
			})
		}
	}

	return result
}

// encodeImageToDataURL base64-encodes an image file into a data URL.
// Returns empty string if the file exceeds maxSize or encoding fails.
func encodeImageToDataURL(localPath, mime string, info os.FileInfo, maxSize int) string {
	if info.Size() > int64(maxSize) {
		logger.WarnCF("agent", "Media file too large, skipping", map[string]any{
			"path":     localPath,
			"size":     info.Size(),
			"max_size": maxSize,
		})
		return ""
	}

	f, err := os.Open(localPath)
	if err != nil {
		logger.WarnCF("agent", "Failed to open media file", map[string]any{
			"path":  localPath,
			"error": err.Error(),
		})
		return ""
	}
	defer f.Close()

	prefix := "data:" + mime + ";base64,"
	encodedLen := base64.StdEncoding.EncodedLen(int(info.Size()))
	var buf bytes.Buffer
	buf.Grow(len(prefix) + encodedLen)
	buf.WriteString(prefix)

	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	if _, err := io.Copy(encoder, f); err != nil {
		logger.WarnCF("agent", "Failed to encode media file", map[string]any{
			"path":  localPath,
			"error": err.Error(),
		})
		return ""
	}
	encoder.Close()

	return buf.String()
}

func buildArtifactTags(store media.MediaStore, refs []string) []string {
	if store == nil || len(refs) == 0 {
		return nil
	}

	tags := make([]string, 0, len(refs))
	for _, ref := range refs {
		localPath, meta, err := store.ResolveWithMeta(ref)
		if err != nil {
			continue
		}
		mime := detectMIME(localPath, meta)
		tags = append(tags, buildPathTag(mime, localPath))
	}

	return tags
}

func buildProviderAttachments(store media.MediaStore, refs []string) []providers.Attachment {
	if store == nil || len(refs) == 0 {
		return nil
	}

	attachments := make([]providers.Attachment, 0, len(refs))
	for _, ref := range refs {
		attachment := providers.Attachment{Ref: ref}
		if _, meta, err := store.ResolveWithMeta(ref); err == nil {
			attachment.Filename = meta.Filename
			attachment.ContentType = meta.ContentType
			attachment.Type = inferMediaType(meta.Filename, meta.ContentType)
		}
		attachments = append(attachments, attachment)
	}

	return attachments
}

// detectMIME determines the MIME type from metadata or magic-bytes detection.
// Returns empty string if detection fails.
func detectMIME(localPath string, meta media.MediaMeta) string {
	if meta.ContentType != "" {
		return meta.ContentType
	}
	kind, err := filetype.MatchFile(localPath)
	if err != nil || kind == filetype.Unknown {
		return ""
	}
	return kind.MIME.Value
}

// buildPathTag creates a structured tag exposing the local file path.
// Tag type is derived from MIME: [image:/path], [audio:/path], [video:/path], or [file:/path].
func buildPathTag(mime, localPath string) string {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "[image:" + localPath + "]"
	case strings.HasPrefix(mime, "audio/"):
		return "[audio:" + localPath + "]"
	case strings.HasPrefix(mime, "video/"):
		return "[video:" + localPath + "]"
	default:
		return "[file:" + localPath + "]"
	}
}

// injectPathTags replaces generic media tags in content with path-bearing versions,
// or appends if no matching generic tag is found.
func injectPathTags(content string, tags []string) string {
	for _, tag := range tags {
		var generic string
		switch {
		case strings.HasPrefix(tag, "[image:"):
			generic = "[image: photo]"
		case strings.HasPrefix(tag, "[audio:"):
			generic = "[audio]"
		case strings.HasPrefix(tag, "[video:"):
			generic = "[video]"
		case strings.HasPrefix(tag, "[file:"):
			generic = "[file]"
		}

		if generic != "" && strings.Contains(content, generic) {
			content = strings.Replace(content, generic, tag, 1)
		} else if content == "" {
			content = tag
		} else {
			content += " " + tag
		}
	}
	return content
}
