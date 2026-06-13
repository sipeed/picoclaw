package tools

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/gif"
	"image/jpeg"
	"image/png"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/sipeed/picoclaw/pkg/media"
)

const (
	largeBase64OmittedMessage = "[Tool returned a large base64-like payload; omitted from model context.]"
	inlineMediaOmittedMessage = "[Tool returned inline media content; omitted from model context.]"
	inlineMediaStoredMessage  = "[Tool returned inline media content (%s); omitted from model context and registered as a media attachment.]"
	inlineDataURLPayloadLimit = 1024
)

var (
	inlineMarkdownDataURLRe = regexp.MustCompile(`!\[[^\]]*\]\((data:[^)]+)\)`)
	inlineRawDataURLRe      = regexp.MustCompile(`data:[^;\s]+;base64,[A-Za-z0-9+/=\r\n]+`)
)

type NormalizeOptions struct {
	AllowInlineMediaExtraction bool
}

func normalizeToolResult(
	result *ToolResult,
	toolName string,
	store media.MediaStore,
	channel string,
	chatID string,
	opts NormalizeOptions,
) *ToolResult {
	if result == nil {
		return nil
	}

	notes := make([]string, 0, 2)
	seen := make(map[string]struct{})

	if opts.AllowInlineMediaExtraction && store != nil && channel != "" && chatID != "" {
		var refs []string
		var extractedNotes []string

		result.ForLLM, refs, extractedNotes = extractInlineMediaRefs(
			result.ForLLM,
			toolName,
			store,
			channel,
			chatID,
			seen,
		)
		result.Media = append(result.Media, refs...)
		notes = append(notes, extractedNotes...)

		result.ForUser, refs, extractedNotes = extractInlineMediaRefs(
			result.ForUser,
			toolName,
			store,
			channel,
			chatID,
			seen,
		)
		result.Media = append(result.Media, refs...)
		notes = append(notes, extractedNotes...)
	}

	if opts.AllowInlineMediaExtraction {
		result.ForLLM = sanitizeInlineDataURLsForLLM(result.ForLLM, true)
	} else {
		result.ForLLM = sanitizeToolLLMContent(result.ForLLM)
	}

	if len(notes) > 0 {
		if strings.TrimSpace(result.ForLLM) == "" {
			result.ForLLM = strings.Join(notes, "\n")
		} else {
			result.ForLLM = strings.TrimSpace(result.ForLLM) + "\n" + strings.Join(notes, "\n")
		}
	}
	if len(result.Media) > 0 && strings.TrimSpace(result.ForLLM) == "" {
		result.ForLLM = "[Tool returned media content; omitted from model context and registered as a media attachment.]"
	}

	return result
}

func sanitizeToolLLMContent(text string) string {
	if cleaned := sanitizeInlineDataURLsForLLM(text, false); cleaned != text {
		return cleaned
	}

	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return text
	}
	if looksLikeLargeBase64Payload(trimmed) {
		return largeBase64OmittedMessage
	}
	return text
}

func sanitizeInlineDataURLsForLLM(text string, all bool) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return text
	}

	changed := false
	cleaned := inlineMarkdownDataURLRe.ReplaceAllStringFunc(trimmed, func(match string) string {
		groups := inlineMarkdownDataURLRe.FindStringSubmatch(match)
		if len(groups) < 2 {
			return match
		}
		if !all && !inlineDataURLShouldBeSanitized(groups[1]) {
			return match
		}
		changed = true
		return ""
	})

	cleaned = inlineRawDataURLRe.ReplaceAllStringFunc(cleaned, func(match string) string {
		if !all && !inlineDataURLShouldBeSanitized(match) {
			return match
		}
		changed = true
		return ""
	})

	if !changed {
		return text
	}

	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return inlineMediaOmittedMessage
	}
	return cleaned + "\n" + inlineMediaOmittedMessage
}

func inlineDataURLShouldBeSanitized(dataURL string) bool {
	comma := strings.IndexByte(dataURL, ',')
	if comma < 0 {
		return false
	}
	payload := strings.NewReplacer("\n", "", "\r", "", "\t", "", " ", "").Replace(dataURL[comma+1:])
	return len(payload) >= inlineDataURLPayloadLimit
}

func looksLikeLargeBase64Payload(text string) bool {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) < 1024 {
		return false
	}

	nonSpace := 0
	base64Like := 0
	spaceCount := 0

	for _, r := range trimmed {
		if unicode.IsSpace(r) {
			spaceCount++
			continue
		}
		nonSpace++
		if (r >= 'A' && r <= 'Z') ||
			(r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			r == '+' || r == '/' || r == '=' {
			base64Like++
		}
	}

	if nonSpace == 0 {
		return false
	}

	ratio := float64(base64Like) / float64(nonSpace)
	return ratio >= 0.97 && spaceCount <= len(trimmed)/128
}

func extractInlineMediaRefs(
	text string,
	toolName string,
	store media.MediaStore,
	channel string,
	chatID string,
	seen map[string]struct{},
) (cleaned string, refs []string, notes []string) {
	cleaned = text

	matches := inlineMarkdownDataURLRe.FindAllStringSubmatch(cleaned, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		dataURL := match[1]
		ref, note := storeInlineDataURL(toolName, store, channel, chatID, dataURL, seen)
		if ref != "" {
			refs = append(refs, ref)
		}
		if note != "" {
			notes = append(notes, note)
		}
		cleaned = strings.ReplaceAll(cleaned, match[0], "")
	}

	rawMatches := inlineRawDataURLRe.FindAllString(cleaned, -1)
	for _, dataURL := range rawMatches {
		ref, note := storeInlineDataURL(toolName, store, channel, chatID, dataURL, seen)
		if ref != "" {
			refs = append(refs, ref)
		}
		if note != "" {
			notes = append(notes, note)
		}
		cleaned = strings.ReplaceAll(cleaned, dataURL, "")
	}

	return strings.TrimSpace(cleaned), refs, notes
}

func storeInlineDataURL(
	toolName string,
	store media.MediaStore,
	channel string,
	chatID string,
	dataURL string,
	seen map[string]struct{},
) (ref string, note string) {
	dataURL = strings.TrimSpace(dataURL)
	if _, ok := seen[dataURL]; ok {
		return "", ""
	}
	seen[dataURL] = struct{}{}

	if !strings.HasPrefix(strings.ToLower(dataURL), "data:") {
		return "", ""
	}

	comma := strings.IndexByte(dataURL, ',')
	if comma <= 5 {
		return "", "[Tool returned inline media content that could not be parsed.]"
	}

	metaPart := dataURL[:comma]
	payload := dataURL[comma+1:]
	if !strings.Contains(strings.ToLower(metaPart), ";base64") {
		return "", "[Tool returned inline media content that was not base64-encoded.]"
	}

	mimeType := strings.TrimSpace(metaPart[len("data:"):])
	if semi := strings.IndexByte(mimeType, ';'); semi >= 0 {
		mimeType = mimeType[:semi]
	}
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	payload = strings.NewReplacer("\n", "", "\r", "", "\t", "", " ", "").Replace(payload)
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Sprintf("[Tool returned inline media content (%s) that could not be decoded.]", mimeType)
	}
	if err = validateInlineImagePayload(mimeType, decoded); err != nil {
		return "", fmt.Sprintf("[Tool returned inline media content (%s) that was rejected: %v.]", mimeType, err)
	}

	dir := media.TempDir()
	if err = os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Sprintf("[Tool returned inline media content (%s) but it could not be stored.]", mimeType)
	}

	ext := extensionForMIMEType(mimeType)
	tmpFile, err := os.CreateTemp(dir, "tool-inline-*"+ext)
	if err != nil {
		return "", fmt.Sprintf("[Tool returned inline media content (%s) but it could not be stored.]", mimeType)
	}
	tmpPath := tmpFile.Name()
	if _, err = tmpFile.Write(decoded); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Sprintf("[Tool returned inline media content (%s) but it could not be stored.]", mimeType)
	}
	if err = tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Sprintf("[Tool returned inline media content (%s) but it could not be stored.]", mimeType)
	}

	filename := sanitizeIdentifierComponent(toolName) + ext
	scope := fmt.Sprintf(
		"tool:inline:%s:%s:%s:%d",
		sanitizeIdentifierComponent(toolName),
		channel,
		chatID,
		time.Now().UnixNano(),
	)

	ref, err = store.Store(tmpPath, media.MediaMeta{
		Filename:    filename,
		ContentType: mimeType,
		Source:      fmt.Sprintf("tool:inline:%s", sanitizeIdentifierComponent(toolName)),
	}, scope)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Sprintf("[Tool returned inline media content (%s) but it could not be registered.]", mimeType)
	}

	return ref, fmt.Sprintf(inlineMediaStoredMessage, mimeType)
}

func validateInlineImagePayload(mimeType string, decoded []byte) error {
	if len(decoded) == 0 {
		return fmt.Errorf("empty payload")
	}

	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/png":
		if !looksLikePNG(decoded) {
			return fmt.Errorf("payload does not match PNG magic bytes")
		}
		if _, err := png.DecodeConfig(bytes.NewReader(decoded)); err != nil {
			return fmt.Errorf("invalid PNG image")
		}
	case "image/jpeg":
		if !looksLikeJPEG(decoded) {
			return fmt.Errorf("payload does not match JPEG magic bytes")
		}
		if _, err := jpeg.DecodeConfig(bytes.NewReader(decoded)); err != nil {
			return fmt.Errorf("invalid JPEG image")
		}
	case "image/gif":
		if !looksLikeGIF(decoded) {
			return fmt.Errorf("payload does not match GIF magic bytes")
		}
		if _, err := gif.DecodeConfig(bytes.NewReader(decoded)); err != nil {
			return fmt.Errorf("invalid GIF image")
		}
	case "image/webp":
		if !looksLikeWebP(decoded) {
			return fmt.Errorf("payload does not match WebP magic bytes")
		}
	default:
		return fmt.Errorf("unsupported MIME type")
	}

	return nil
}

func looksLikePNG(b []byte) bool {
	return len(b) >= 8 &&
		b[0] == 0x89 &&
		b[1] == 0x50 &&
		b[2] == 0x4E &&
		b[3] == 0x47 &&
		b[4] == 0x0D &&
		b[5] == 0x0A &&
		b[6] == 0x1A &&
		b[7] == 0x0A
}

func looksLikeJPEG(b []byte) bool {
	return len(b) >= 3 &&
		b[0] == 0xFF &&
		b[1] == 0xD8 &&
		b[2] == 0xFF
}

func looksLikeGIF(b []byte) bool {
	return len(b) >= 6 &&
		(string(b[:6]) == "GIF87a" || string(b[:6]) == "GIF89a")
}

func looksLikeWebP(b []byte) bool {
	if len(b) < 20 ||
		string(b[0:4]) != "RIFF" ||
		string(b[8:12]) != "WEBP" {
		return false
	}
	declaredSize := int(b[4]) | int(b[5])<<8 | int(b[6])<<16 | int(b[7])<<24
	if declaredSize > len(b)-8 {
		return false
	}
	switch string(b[12:16]) {
	case "VP8 ", "VP8L", "VP8X":
		return true
	default:
		return false
	}
}

func extensionForMIMEType(mimeType string) string {
	if mimeType == "" {
		return ".bin"
	}
	if exts, err := mime.ExtensionsByType(mimeType); err == nil && len(exts) > 0 {
		return exts[0]
	}

	switch strings.ToLower(mimeType) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "audio/wav", "audio/x-wav":
		return ".wav"
	case "audio/mpeg":
		return ".mp3"
	case "audio/ogg":
		return ".ogg"
	case "video/mp4":
		return ".mp4"
	default:
		return filepath.Ext(mimeType)
	}
}
