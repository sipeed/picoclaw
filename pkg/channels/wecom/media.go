package wecom

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/h2non/filetype"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/media"
)

const (
	wecomOutboundMediaMaxBytes = 20 << 20
	wecomOutboundImageMaxBytes = 2 << 20
	wecomOutboundVoiceMaxBytes = 2 << 20
	wecomOutboundVideoMaxBytes = 10 << 20
	wecomUploadChunkMaxBytes   = 512 << 10
	wecomUploadMaxChunks       = 100
	wecomUploadMinBytes        = 5
)

type wecomOutboundMedia struct {
	MsgType     string
	MediaID     string
	Title       string
	Description string
}

func (m *wecomOutboundMedia) respondBody() wecomRespondMsgBody {
	body := wecomRespondMsgBody{MsgType: m.MsgType}
	switch m.MsgType {
	case "file":
		body.File = &wecomMediaRefContent{MediaID: m.MediaID}
	case "image":
		body.Image = &wecomMediaRefContent{MediaID: m.MediaID}
	case "voice":
		body.Voice = &wecomMediaRefContent{MediaID: m.MediaID}
	case "video":
		body.Video = &wecomVideoContent{
			MediaID:     m.MediaID,
			Title:       m.Title,
			Description: m.Description,
		}
	}
	return body
}

func (m *wecomOutboundMedia) sendBody(chatID string, chatType uint32) wecomSendMsgBody {
	body := wecomSendMsgBody{
		ChatID:   chatID,
		ChatType: chatType,
		MsgType:  m.MsgType,
	}
	switch m.MsgType {
	case "file":
		body.File = &wecomMediaRefContent{MediaID: m.MediaID}
	case "image":
		body.Image = &wecomMediaRefContent{MediaID: m.MediaID}
	case "voice":
		body.Voice = &wecomMediaRefContent{MediaID: m.MediaID}
	case "video":
		body.Video = &wecomVideoContent{
			MediaID:     m.MediaID,
			Title:       m.Title,
			Description: m.Description,
		}
	}
	return body
}

func decodeMediaAESKey(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	key, err := base64.StdEncoding.DecodeString(value)
	if err == nil && len(key) == 32 {
		return key, nil
	}
	key, err = base64.StdEncoding.DecodeString(value + "=")
	if err != nil {
		return nil, fmt.Errorf("decode AES key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid AES key length %d", len(key))
	}
	return key, nil
}

func decryptAESCBC(key, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("ciphertext is empty")
	}
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext length %d is not a multiple of block size", len(ciphertext))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	plaintext := make([]byte, len(ciphertext))
	iv := key[:aes.BlockSize]
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, ciphertext)
	return pkcs7Unpad(plaintext)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty plaintext")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > 32 || padding > len(data) {
		return nil, fmt.Errorf("invalid padding size %d", padding)
	}
	for i := 0; i < padding; i++ {
		if data[len(data)-1-i] != byte(padding) {
			return nil, fmt.Errorf("invalid padding byte")
		}
	}
	return data[:len(data)-padding], nil
}

func inferMediaExt(contentType, fallback string) string {
	contentType = normalizeWeComContentType(contentType)
	switch contentType {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	case "video/mp4":
		return ".mp4"
	default:
		return fallback
	}
}

func normalizeWeComContentType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if idx := strings.Index(value, ";"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	return value
}

func isGenericWeComContentType(value string) bool {
	switch normalizeWeComContentType(value) {
	case "", "application/octet-stream", "binary/octet-stream", "application/unknown", "application/binary":
		return true
	default:
		return false
	}
}

func sanitizeWeComFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == "/" || name == "" {
		return ""
	}
	return name
}

func candidateWeComFilename(resourceURL, contentDisposition, fallbackName string) string {
	if _, params, err := mime.ParseMediaType(contentDisposition); err == nil {
		if name := sanitizeWeComFilename(params["filename"]); name != "" {
			return name
		}
		if name := sanitizeWeComFilename(params["filename*"]); name != "" {
			return name
		}
	}

	if parsed, err := url.Parse(resourceURL); err == nil {
		query := parsed.Query()
		for _, key := range []string{"filename", "file_name", "name"} {
			if name := sanitizeWeComFilename(query.Get(key)); name != "" {
				return name
			}
		}
		if name := sanitizeWeComFilename(parsed.Path); name != "" {
			return name
		}
	}

	return sanitizeWeComFilename(fallbackName)
}

func detectWeComFiletype(data []byte) (string, string) {
	kind, err := filetype.Match(data)
	if err != nil || kind == filetype.Unknown {
		return "", ""
	}
	ext := ""
	if kind.Extension != "" {
		ext = "." + strings.ToLower(kind.Extension)
	}
	return normalizeWeComContentType(kind.MIME.Value), ext
}

func detectWeComMediaMetadata(
	data []byte,
	fallbackName, fallbackContentType, resourceURL, contentDisposition string,
) (string, string) {
	filename := candidateWeComFilename(resourceURL, contentDisposition, fallbackName)
	if filename == "" {
		filename = "media"
	}

	ext := strings.ToLower(filepath.Ext(filename))
	contentType := normalizeWeComContentType(fallbackContentType)
	detectedType, detectedExt := detectWeComFiletype(data)

	if ext != "" && isGenericWeComContentType(contentType) {
		if byExt := normalizeWeComContentType(mime.TypeByExtension(ext)); byExt != "" {
			contentType = byExt
		}
	}

	if detectedType != "" {
		switch {
		case contentType == "":
			contentType = detectedType
		case isGenericWeComContentType(contentType):
			contentType = detectedType
		case strings.HasPrefix(detectedType, "image/") && !strings.HasPrefix(contentType, "image/"):
			contentType = detectedType
		case strings.HasPrefix(detectedType, "audio/") && !strings.HasPrefix(contentType, "audio/"):
			contentType = detectedType
		case strings.HasPrefix(detectedType, "video/") && !strings.HasPrefix(contentType, "video/"):
			contentType = detectedType
		}
	}

	if contentType == "" && ext != "" {
		contentType = normalizeWeComContentType(mime.TypeByExtension(ext))
	}
	if contentType == "" {
		contentType = normalizeWeComContentType(http.DetectContentType(data))
	}

	if ext == "" {
		ext = detectedExt
	}
	if ext == "" && contentType != "" {
		if exts, err := mime.ExtensionsByType(contentType); err == nil && len(exts) > 0 {
			ext = strings.ToLower(exts[0])
		}
	}

	if filepath.Ext(filename) == "" && ext != "" {
		filename += ext
	}
	return filename, contentType
}

func (c *WeComChannel) storeRemoteMedia(
	ctx context.Context,
	scope, msgID, resourceURL, aesKey, fallbackExt string,
) (string, error) {
	store := c.GetMediaStore()
	if store == nil {
		return "", fmt.Errorf("no media store available")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resourceURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := c.mediaClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download media: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download media returned HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, wecomOutboundMediaMaxBytes+1))
	if err != nil {
		return "", fmt.Errorf("read media: %w", err)
	}
	if len(data) > wecomOutboundMediaMaxBytes {
		return "", fmt.Errorf("media too large")
	}

	if aesKey != "" {
		key, keyErr := decodeMediaAESKey(aesKey)
		if keyErr != nil {
			return "", keyErr
		}
		data, err = decryptAESCBC(key, data)
		if err != nil {
			return "", fmt.Errorf("decrypt media: %w", err)
		}
	}

	filename, contentType := detectWeComMediaMetadata(
		data,
		msgID+fallbackExt,
		resp.Header.Get("Content-Type"),
		resourceURL,
		resp.Header.Get("Content-Disposition"),
	)
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = inferMediaExt(contentType, fallbackExt)
	}
	mediaDir := filepath.Join(os.TempDir(), "picoclaw_media")
	if mkdirErr := os.MkdirAll(mediaDir, 0o700); mkdirErr != nil {
		return "", fmt.Errorf("mkdir media dir: %w", mkdirErr)
	}
	tmpFile, err := os.CreateTemp(mediaDir, msgID+"-*"+ext)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	if _, writeErr := tmpFile.Write(data); writeErr != nil {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("write temp file: %w", writeErr)
	}
	if closeErr := tmpFile.Close(); closeErr != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("close temp file: %w", closeErr)
	}

	ref, err := store.Store(tmpPath, media.MediaMeta{
		Filename:      filename,
		ContentType:   contentType,
		Source:        "wecom",
		CleanupPolicy: media.CleanupPolicyDeleteOnCleanup,
	}, scope)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return ref, nil
}

func detectLocalWeComContentType(localPath, hint string) string {
	contentType := normalizeWeComContentType(hint)
	if !isGenericWeComContentType(contentType) {
		return contentType
	}

	if kind, err := filetype.MatchFile(localPath); err == nil && kind != filetype.Unknown {
		return normalizeWeComContentType(kind.MIME.Value)
	}

	if ext := strings.ToLower(filepath.Ext(localPath)); ext != "" {
		if byExt := normalizeWeComContentType(mime.TypeByExtension(ext)); byExt != "" {
			return byExt
		}
	}

	file, err := os.Open(localPath)
	if err != nil {
		return contentType
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return contentType
	}
	if n == 0 {
		return contentType
	}
	return normalizeWeComContentType(http.DetectContentType(buf[:n]))
}

func writeWeComTempFile(prefix, filename string, data []byte) (string, error) {
	mediaDir := filepath.Join(os.TempDir(), "picoclaw_media")
	if err := os.MkdirAll(mediaDir, 0o700); err != nil {
		return "", fmt.Errorf("mkdir media dir: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filename))
	tmpFile, err := os.CreateTemp(mediaDir, prefix+"-*"+ext)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("close temp file: %w", err)
	}
	return tmpPath, nil
}

func (c *WeComChannel) downloadRemoteMediaToTemp(
	ctx context.Context,
	resourceURL, fallbackName string,
) (string, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resourceURL, nil)
	if err != nil {
		return "", "", "", fmt.Errorf("create request: %w", err)
	}

	resp, err := c.mediaClient.Do(req)
	if err != nil {
		return "", "", "", fmt.Errorf("download media: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", "", "", fmt.Errorf("download media returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, wecomOutboundMediaMaxBytes+1))
	if err != nil {
		return "", "", "", fmt.Errorf("read media: %w", err)
	}
	if len(data) > wecomOutboundMediaMaxBytes {
		return "", "", "", fmt.Errorf("media too large")
	}

	filename, contentType := detectWeComMediaMetadata(
		data,
		fallbackName,
		resp.Header.Get("Content-Type"),
		resourceURL,
		resp.Header.Get("Content-Disposition"),
	)
	tmpPath, err := writeWeComTempFile("wecom-outbound", filename, data)
	if err != nil {
		return "", "", "", err
	}
	return tmpPath, filename, contentType, nil
}

func (c *WeComChannel) resolveOutboundPart(
	ctx context.Context,
	part bus.MediaPart,
) (string, string, string, func(), error) {
	cleanup := func() {}
	filename := sanitizeWeComFilename(part.Filename)
	contentType := normalizeWeComContentType(part.ContentType)
	ref := strings.TrimSpace(part.Ref)

	switch {
	case ref == "":
		return "", filename, contentType, cleanup, nil

	case strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://"):
		localPath, name, ct, err := c.downloadRemoteMediaToTemp(ctx, ref, filename)
		if err != nil {
			return "", "", "", cleanup, err
		}
		return localPath, name, ct, func() { _ = os.Remove(localPath) }, nil

	case strings.HasPrefix(ref, "media://"):
		store := c.GetMediaStore()
		if store == nil {
			return "", "", "", cleanup, fmt.Errorf("no media store available")
		}

		localPath, meta, err := store.ResolveWithMeta(ref)
		if err != nil {
			return "", "", "", cleanup, err
		}
		if filename == "" {
			filename = sanitizeWeComFilename(meta.Filename)
		}
		if contentType == "" {
			contentType = normalizeWeComContentType(meta.ContentType)
		}
		if strings.HasPrefix(localPath, "http://") || strings.HasPrefix(localPath, "https://") {
			tmpPath, name, ct, err := c.downloadRemoteMediaToTemp(ctx, localPath, filename)
			if err != nil {
				return "", "", "", cleanup, err
			}
			return tmpPath, name, ct, func() { _ = os.Remove(tmpPath) }, nil
		}
		if _, err := os.Stat(localPath); err != nil {
			return "", "", "", cleanup, err
		}
		if filename == "" {
			filename = sanitizeWeComFilename(filepath.Base(localPath))
		}
		if contentType == "" {
			contentType = detectLocalWeComContentType(localPath, "")
		}
		return localPath, filename, contentType, cleanup, nil

	case strings.HasPrefix(ref, "file://"):
		u, err := url.Parse(ref)
		if err != nil {
			return "", "", "", cleanup, err
		}
		localPath := u.Path
		if _, err := os.Stat(localPath); err != nil {
			return "", "", "", cleanup, err
		}
		if filename == "" {
			filename = sanitizeWeComFilename(filepath.Base(localPath))
		}
		if contentType == "" {
			contentType = detectLocalWeComContentType(localPath, "")
		}
		return localPath, filename, contentType, cleanup, nil

	default:
		if _, err := os.Stat(ref); err != nil {
			return "", "", "", cleanup, err
		}
		if filename == "" {
			filename = sanitizeWeComFilename(filepath.Base(ref))
		}
		if contentType == "" {
			contentType = detectLocalWeComContentType(ref, "")
		}
		return ref, filename, contentType, cleanup, nil
	}
}

func canWeComSendImage(contentType, ext string, size int64) bool {
	if size > wecomOutboundImageMaxBytes {
		return false
	}
	switch normalizeWeComContentType(contentType) {
	case "image/jpeg", "image/jpg", "image/png", "image/gif":
		return true
	}
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png", ".gif":
		return true
	default:
		return false
	}
}

func canWeComSendVoice(contentType, ext string, size int64) bool {
	if size > wecomOutboundVoiceMaxBytes {
		return false
	}
	contentType = normalizeWeComContentType(contentType)
	return strings.Contains(contentType, "amr") || strings.EqualFold(ext, ".amr")
}

func canWeComSendVideo(contentType, ext string, size int64) bool {
	if size > wecomOutboundVideoMaxBytes {
		return false
	}
	return normalizeWeComContentType(contentType) == "video/mp4" || strings.EqualFold(ext, ".mp4")
}

func outboundWeComMediaKind(partType, filename, contentType string, size int64) string {
	if size < wecomUploadMinBytes {
		return ""
	}

	partType = strings.ToLower(strings.TrimSpace(partType))
	contentType = normalizeWeComContentType(contentType)
	ext := strings.ToLower(filepath.Ext(filename))

	if partType == "file" {
		if size <= wecomOutboundMediaMaxBytes {
			return "file"
		}
		return ""
	}

	if (partType == "image" || partType == "") && canWeComSendImage(contentType, ext, size) {
		return "image"
	}
	if (partType == "audio" || partType == "voice" || partType == "") && canWeComSendVoice(contentType, ext, size) {
		return "voice"
	}
	if (partType == "video" || partType == "") && canWeComSendVideo(contentType, ext, size) {
		return "video"
	}
	if size <= wecomOutboundMediaMaxBytes {
		return "file"
	}
	return ""
}

func trimWeComBytes(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	size := 0
	var out strings.Builder
	for _, r := range value {
		width := len(string(r))
		if size+width > limit {
			break
		}
		size += width
		out.WriteRune(r)
	}
	return out.String()
}

func ensureWeComOutboundFilename(filename, localPath, contentType string) string {
	filename = sanitizeWeComFilename(filename)
	if filename == "" {
		filename = sanitizeWeComFilename(filepath.Base(localPath))
	}
	if filename == "" {
		filename = "media"
	}
	if filepath.Ext(filename) == "" {
		fallbackExt := inferMediaExt(contentType, strings.ToLower(filepath.Ext(localPath)))
		if fallbackExt != "" {
			filename += fallbackExt
		}
	}
	filename = trimWeComBytes(filename, 256)
	if filename == "" {
		return "media"
	}
	return filename
}

func buildWeComVideoContent(mediaID, filename, description string) *wecomVideoContent {
	title := strings.TrimSuffix(filename, filepath.Ext(filename))
	title = trimWeComBytes(title, 64)
	if title == "" {
		title = "video"
	}
	description = trimWeComBytes(description, 512)
	return &wecomVideoContent{
		MediaID:     mediaID,
		Title:       title,
		Description: description,
	}
}

func decodeWeComEnvelopeBody[T any](env wecomEnvelope) (T, error) {
	var out T
	if len(env.Body) == 0 {
		return out, fmt.Errorf("wecom response body is empty")
	}
	if err := json.Unmarshal(env.Body, &out); err != nil {
		return out, fmt.Errorf("decode wecom response body: %w", err)
	}
	return out, nil
}

func (c *WeComChannel) uploadOutboundMedia(
	ctx context.Context,
	localPath, filename, contentType string,
	part bus.MediaPart,
) (*wecomOutboundMedia, error) {
	_ = ctx

	contentType = detectLocalWeComContentType(localPath, contentType)
	filename = ensureWeComOutboundFilename(filename, localPath, contentType)

	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, fmt.Errorf("read media file: %w", err)
	}
	size := int64(len(data))
	kind := outboundWeComMediaKind(part.Type, filename, contentType, size)
	if kind == "" {
		return nil, fmt.Errorf("unsupported wecom media type or size for %q", filename)
	}

	totalChunks := (len(data) + wecomUploadChunkMaxBytes - 1) / wecomUploadChunkMaxBytes
	if totalChunks <= 0 || totalChunks > wecomUploadMaxChunks {
		return nil, fmt.Errorf("wecom upload requires 1-%d chunks, got %d", wecomUploadMaxChunks, totalChunks)
	}

	sum := md5.Sum(data)
	initEnv, err := c.sendCommandAck(wecomCommand{
		Cmd:     wecomCmdUploadMediaInit,
		Headers: wecomHeaders{ReqID: randomID(10)},
		Body: wecomUploadMediaInitBody{
			Type:        kind,
			Filename:    filename,
			TotalSize:   size,
			TotalChunks: totalChunks,
			MD5:         hex.EncodeToString(sum[:]),
		},
	}, wecomUploadTimeout)
	if err != nil {
		return nil, err
	}
	initResp, err := decodeWeComEnvelopeBody[wecomUploadMediaInitResponse](initEnv)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(initResp.UploadID) == "" {
		return nil, fmt.Errorf("wecom upload init returned empty upload_id")
	}

	for idx, offset := 0, 0; offset < len(data); idx, offset = idx+1, offset+wecomUploadChunkMaxBytes {
		end := offset + wecomUploadChunkMaxBytes
		if end > len(data) {
			end = len(data)
		}
		sendErr := c.sendCommand(wecomCommand{
			Cmd:     wecomCmdUploadMediaChunk,
			Headers: wecomHeaders{ReqID: randomID(10)},
			Body: wecomUploadMediaChunkBody{
				UploadID:   initResp.UploadID,
				ChunkIndex: idx,
				Base64Data: base64.StdEncoding.EncodeToString(data[offset:end]),
			},
		}, wecomUploadTimeout)
		if sendErr != nil {
			return nil, sendErr
		}
	}

	finishEnv, err := c.sendCommandAck(wecomCommand{
		Cmd:     wecomCmdUploadMediaEnd,
		Headers: wecomHeaders{ReqID: randomID(10)},
		Body: wecomUploadMediaFinishBody{
			UploadID: initResp.UploadID,
		},
	}, wecomUploadTimeout)
	if err != nil {
		return nil, err
	}
	finishResp, err := decodeWeComEnvelopeBody[wecomUploadMediaFinishResponse](finishEnv)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(finishResp.MediaID) == "" {
		return nil, fmt.Errorf("wecom upload finish returned empty media_id")
	}

	uploaded := &wecomOutboundMedia{
		MsgType: kind,
		MediaID: finishResp.MediaID,
	}
	if kind == "video" {
		video := buildWeComVideoContent(finishResp.MediaID, filename, part.Caption)
		uploaded.Title = video.Title
		uploaded.Description = video.Description
	}
	return uploaded, nil
}

func fallbackWeComMediaText(part bus.MediaPart, kind, filename string) string {
	var lines []string
	if caption := strings.TrimSpace(part.Caption); caption != "" {
		lines = append(lines, caption)
	}

	label := kind
	if label == "" {
		label = "media"
	}
	if filename != "" {
		lines = append(lines, fmt.Sprintf("[%s: %s]", label, filename))
	} else {
		lines = append(lines, fmt.Sprintf("[%s attachment]", label))
	}

	ref := strings.TrimSpace(part.Ref)
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		lines = append(lines, ref)
	}

	return strings.Join(lines, "\n")
}

func (c *WeComChannel) resolveMediaRoute(chatID string) (wecomTurn, uint32, bool) {
	if turn, ok := c.getTurn(chatID); ok {
		if time.Since(turn.CreatedAt) <= wecomStreamMaxDuration {
			return turn, turn.ChatType, true
		}
		c.deleteTurn(chatID)
	}
	if route, ok := c.routes.Get(chatID); ok {
		return wecomTurn{ChatID: route.ChatID, ChatType: route.ChatType}, route.ChatType, false
	}
	return wecomTurn{ChatID: chatID}, 0, false
}
