package weixin

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/h2non/filetype"

	"github.com/sipeed/picoclaw/pkg/bus"
	basechannels "github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
)

const (
	weixinMediaMaxBytes         = 100 << 20
	weixinTypingKeepAlive       = 5 * time.Second
	weixinUploadRetryMax        = 3
	weixinDownloadRetryMax      = 2
	weixinDownloadRetryDelay    = 300 * time.Millisecond
	weixinVoiceTranscodeTimeout = 15 * time.Second
)

type uploadedFileInfo struct {
	downloadParam string
	aesKeyHex     string
	fileSize      int64
	cipherSize    int64
	filename      string
}

func pkcs7Pad(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	if padding == 0 {
		padding = blockSize
	}
	out := make([]byte, len(src)+padding)
	copy(out, src)
	for i := len(src); i < len(out); i++ {
		out[i] = byte(padding)
	}
	return out
}

func pkcs7Unpad(src []byte, blockSize int) ([]byte, error) {
	if len(src) == 0 || len(src)%blockSize != 0 {
		return nil, fmt.Errorf("invalid padded data size %d", len(src))
	}
	padding := int(src[len(src)-1])
	if padding <= 0 || padding > blockSize || padding > len(src) {
		return nil, fmt.Errorf("invalid padding size %d", padding)
	}
	for i := len(src) - padding; i < len(src); i++ {
		if src[i] != byte(padding) {
			return nil, fmt.Errorf("invalid padding content")
		}
	}
	return src[:len(src)-padding], nil
}

func encryptAESECB(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	padded := pkcs7Pad(plaintext, block.BlockSize())
	out := make([]byte, len(padded))
	for i := 0; i < len(padded); i += block.BlockSize() {
		block.Encrypt(out[i:i+block.BlockSize()], padded[i:i+block.BlockSize()])
	}
	return out, nil
}

func decryptAESECB(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("invalid ciphertext size %d", len(ciphertext))
	}
	out := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += block.BlockSize() {
		block.Decrypt(out[i:i+block.BlockSize()], ciphertext[i:i+block.BlockSize()])
	}
	return pkcs7Unpad(out, block.BlockSize())
}

func parseWeixinMediaAESKey(aesKeyBase64 string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(aesKeyBase64)
	if err != nil {
		return nil, err
	}
	if len(decoded) == 16 {
		return decoded, nil
	}
	if len(decoded) == 32 {
		if raw, err := hex.DecodeString(string(decoded)); err == nil && len(raw) == 16 {
			return raw, nil
		}
	}
	return nil, fmt.Errorf("unsupported aes_key length %d", len(decoded))
}

func imageAESKey(img *ImageItem) ([]byte, bool, error) {
	if img == nil {
		return nil, false, nil
	}
	if img.Aeskey != "" {
		raw, err := hex.DecodeString(img.Aeskey)
		if err != nil {
			return nil, false, err
		}
		return raw, true, nil
	}
	if img.Media != nil && img.Media.AesKey != "" {
		raw, err := parseWeixinMediaAESKey(img.Media.AesKey)
		if err != nil {
			return nil, false, err
		}
		return raw, true, nil
	}
	return nil, false, nil
}

func genericMediaAESKey(mediaRef *CDNMedia) ([]byte, error) {
	if mediaRef == nil || mediaRef.AesKey == "" {
		return nil, fmt.Errorf("missing aes_key")
	}
	return parseWeixinMediaAESKey(mediaRef.AesKey)
}

func aesEcbPaddedSize(size int64) int64 {
	return (size/16 + 1) * 16
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func buildCDNDownloadURL(base, encryptedQueryParam string) string {
	return strings.TrimRight(base, "/") +
		"/download?encrypted_query_param=" + url.QueryEscape(encryptedQueryParam)
}

func shouldRetryCDNDownload(statusCode int) bool {
	// statusCode=0 represents transport/build errors from the HTTP client.
	return statusCode == 0 || statusCode >= 500 || statusCode == http.StatusTooManyRequests
}

func buildCDNUploadURL(base, uploadParam, filekey string) string {
	return strings.TrimRight(base, "/") +
		"/upload?encrypted_query_param=" + url.QueryEscape(uploadParam) +
		"&filekey=" + url.QueryEscape(filekey)
}

func uniqCDNURLs(urls []string) []string {
	seen := make(map[string]struct{}, len(urls))
	out := make([]string, 0, len(urls))
	for _, raw := range urls {
		u := strings.TrimSpace(raw)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out
}

func (c *WeixinChannel) downloadCDNBufferOnce(ctx context.Context, downloadURL string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := c.api.HttpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, resp.StatusCode, fmt.Errorf("cdn download HTTP %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, weixinMediaMaxBytes+1))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if len(data) > weixinMediaMaxBytes {
		return nil, resp.StatusCode, fmt.Errorf("cdn media too large: %d bytes", len(data))
	}
	return data, resp.StatusCode, nil
}

func (c *WeixinChannel) downloadCDNBuffer(
	ctx context.Context,
	encryptedQueryParam,
	fullURL string,
) ([]byte, error) {
	candidates := uniqCDNURLs([]string{
		strings.TrimSpace(fullURL),
		func() string {
			if strings.TrimSpace(encryptedQueryParam) == "" {
				return ""
			}
			return buildCDNDownloadURL(c.cdnBaseURL(), encryptedQueryParam)
		}(),
	})
	if len(candidates) == 0 {
		return nil, fmt.Errorf("missing CDN download URL")
	}

	var lastErr error
	for _, downloadURL := range candidates {
		for attempt := 1; attempt <= weixinDownloadRetryMax; attempt++ {
			data, statusCode, err := c.downloadCDNBufferOnce(ctx, downloadURL)
			if err == nil {
				return data, nil
			}
			lastErr = fmt.Errorf("%w (attempt=%d url=%s)", err, attempt, downloadURL)
			if !shouldRetryCDNDownload(statusCode) {
				break
			}
			if attempt < weixinDownloadRetryMax {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(weixinDownloadRetryDelay):
				}
			}
		}
	}
	return nil, lastErr
}

func (c *WeixinChannel) downloadAndDecryptCDNBuffer(
	ctx context.Context,
	encryptedQueryParam string,
	fullURL string,
	key []byte,
) ([]byte, error) {
	data, err := c.downloadCDNBuffer(ctx, encryptedQueryParam, fullURL)
	if err != nil {
		return nil, err
	}
	if len(key) == 0 {
		return data, nil
	}
	return decryptAESECB(data, key)
}

func (c *WeixinChannel) downloadImageBuffer(
	ctx context.Context,
	img *ImageItem,
	key []byte,
) ([]byte, error) {
	if img == nil {
		return nil, fmt.Errorf("image item is nil")
	}
	if img.Media != nil {
		data, err := c.downloadAndDecryptCDNBuffer(ctx, img.Media.EncryptQueryParam, img.Media.FullURL, key)
		if err == nil {
			return data, nil
		}
		if img.ThumbMedia == nil {
			return nil, fmt.Errorf("image download failed: %w", err)
		}
	}
	if img.ThumbMedia != nil {
		data, err := c.downloadAndDecryptCDNBuffer(ctx, img.ThumbMedia.EncryptQueryParam, img.ThumbMedia.FullURL, key)
		if err == nil {
			return data, nil
		}
		return nil, fmt.Errorf("image download failed: %w", err)
	}
	return nil, fmt.Errorf("image media is nil")
}

func detectMediaMetadata(data []byte, fallbackName, fallbackContentType string) (string, string) {
	contentType := strings.TrimSpace(fallbackContentType)
	ext := filepath.Ext(fallbackName)
	if kind, err := filetype.Match(data); err == nil && kind != filetype.Unknown {
		contentType = kind.MIME.Value
		if kind.Extension != "" {
			ext = "." + kind.Extension
		}
	}
	if contentType == "" && ext != "" {
		contentType = mime.TypeByExtension(strings.ToLower(ext))
	}
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if ext == "" && contentType != "" {
		if exts, err := mime.ExtensionsByType(contentType); err == nil && len(exts) > 0 {
			ext = exts[0]
		}
	}

	filename := sanitizeFilename(fallbackName)
	if filename == "" {
		filename = "media"
	}
	if filepath.Ext(filename) == "" && ext != "" {
		filename += ext
	}
	return filename, contentType
}

func sanitizeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == "/" || name == "" {
		return ""
	}
	return name
}

func writeManagedTempFile(prefix, filename string, data []byte) (string, error) {
	if err := os.MkdirAll(media.TempDir(), 0o700); err != nil {
		return "", err
	}
	pattern := prefix + "-*"
	if ext := filepath.Ext(filename); ext != "" {
		pattern += ext
	}
	f, err := os.CreateTemp(media.TempDir(), pattern)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func (c *WeixinChannel) storeInboundBytes(
	chatID,
	messageID,
	filename,
	contentType string,
	data []byte,
) (string, error) {
	store := c.GetMediaStore()
	if store == nil {
		return "", fmt.Errorf("no media store available")
	}
	filename, contentType = detectMediaMetadata(data, filename, contentType)
	tmpPath, err := writeManagedTempFile("weixin-inbound", filename, data)
	if err != nil {
		return "", err
	}
	ref, err := store.Store(tmpPath, media.MediaMeta{
		Filename:      filename,
		ContentType:   contentType,
		Source:        c.Name(),
		CleanupPolicy: media.CleanupPolicyDeleteOnCleanup,
	}, basechannels.BuildMediaScope(c.Name(), chatID, messageID))
	if err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	return ref, nil
}

func isDownloadableMediaItem(item *MessageItem) bool {
	if item == nil {
		return false
	}

	switch item.Type {
	case MessageItemTypeImage:
		return item.ImageItem != nil && item.ImageItem.Media != nil &&
			(item.ImageItem.Media.EncryptQueryParam != "" || item.ImageItem.Media.FullURL != "")
	case MessageItemTypeVideo:
		return item.VideoItem != nil && item.VideoItem.Media != nil &&
			(item.VideoItem.Media.EncryptQueryParam != "" || item.VideoItem.Media.FullURL != "")
	case MessageItemTypeFile:
		return item.FileItem != nil && item.FileItem.Media != nil &&
			(item.FileItem.Media.EncryptQueryParam != "" || item.FileItem.Media.FullURL != "")
	case MessageItemTypeVoice:
		return item.VoiceItem != nil &&
			item.VoiceItem.Media != nil &&
			(item.VoiceItem.Media.EncryptQueryParam != "" || item.VoiceItem.Media.FullURL != "") &&
			strings.TrimSpace(item.VoiceItem.Text) == ""
	default:
		return false
	}
}

func selectInboundMediaItem(msg WeixinMessage) *MessageItem {
	priorities := []int{
		MessageItemTypeImage,
		MessageItemTypeVideo,
		MessageItemTypeFile,
		MessageItemTypeVoice,
	}

	for _, want := range priorities {
		for i := range msg.ItemList {
			item := &msg.ItemList[i]
			if item.Type == want && isDownloadableMediaItem(item) {
				return item
			}
		}
	}

	for i := range msg.ItemList {
		item := &msg.ItemList[i]
		if item.Type != MessageItemTypeText || item.RefMsg == nil || item.RefMsg.MessageItem == nil {
			continue
		}
		if isDownloadableMediaItem(item.RefMsg.MessageItem) {
			return item.RefMsg.MessageItem
		}
	}

	return nil
}

func tryTranscodeSilkToWAV(ctx context.Context, silk []byte) ([]byte, error) {
	decoders := []struct {
		name string
		args func(inputPath, outputPath string) []string
	}{
		{
			name: "silk_v3_decoder",
			args: func(inputPath, outputPath string) []string { return []string{inputPath, outputPath, "24000"} },
		},
		{
			name: "silk_decoder",
			args: func(inputPath, outputPath string) []string { return []string{inputPath, outputPath, "24000"} },
		},
		{
			name: "ffmpeg",
			args: func(inputPath, outputPath string) []string {
				return []string{"-y", "-i", inputPath, outputPath}
			},
		},
	}

	for _, decoder := range decoders {
		bin, err := exec.LookPath(decoder.name)
		if err != nil {
			continue
		}

		tmpIn, err := writeManagedTempFile("weixin-voice", "voice.silk", silk)
		if err != nil {
			return nil, err
		}
		tmpOut := filepath.Join(media.TempDir(), "weixin-voice-"+uuid.New().String()+".wav")
		wav, ok := func() ([]byte, bool) {
			defer os.Remove(tmpIn)
			defer os.Remove(tmpOut)

			runCtx, cancel := context.WithTimeout(ctx, weixinVoiceTranscodeTimeout)
			cmd := exec.CommandContext(runCtx, bin, decoder.args(tmpIn, tmpOut)...)
			out, runErr := cmd.CombinedOutput()
			cancel()
			if runErr != nil {
				logger.DebugCF("weixin", "SILK transcode command failed", map[string]any{
					"decoder": decoder.name,
					"error":   runErr.Error(),
					"output":  strings.TrimSpace(string(out)),
				})
				return nil, false
			}

			wav, readErr := os.ReadFile(tmpOut)
			if readErr != nil {
				logger.DebugCF("weixin", "Failed to read transcoded WAV", map[string]any{
					"decoder": decoder.name,
					"error":   readErr.Error(),
				})
				return nil, false
			}
			return wav, len(wav) > 0
		}()
		if ok {
			return wav, nil
		}
	}

	return nil, fmt.Errorf("no SILK decoder available")
}

func (c *WeixinChannel) downloadMediaFromItem(
	ctx context.Context,
	chatID,
	messageID string,
	item *MessageItem,
) (string, error) {
	if item == nil {
		return "", nil
	}

	switch item.Type {
	case MessageItemTypeImage:
		if item.ImageItem == nil {
			return "", fmt.Errorf("image media is nil")
		}
		key, ok, err := imageAESKey(item.ImageItem)
		if err != nil {
			return "", err
		}
		decryptKey := func() []byte {
			if ok {
				return key
			}
			return nil
		}()
		data, err := c.downloadImageBuffer(ctx, item.ImageItem, decryptKey)
		if err != nil {
			return "", err
		}
		return c.storeInboundBytes(chatID, messageID, "image", "", data)

	case MessageItemTypeVoice:
		key, err := genericMediaAESKey(item.VoiceItem.Media)
		if err != nil {
			return "", err
		}
		silk, err := c.downloadAndDecryptCDNBuffer(
			ctx,
			item.VoiceItem.Media.EncryptQueryParam,
			item.VoiceItem.Media.FullURL,
			key,
		)
		if err != nil {
			return "", err
		}
		if wav, err := tryTranscodeSilkToWAV(ctx, silk); err == nil && len(wav) > 0 {
			return c.storeInboundBytes(chatID, messageID, "voice.wav", "audio/wav", wav)
		}
		return c.storeInboundBytes(chatID, messageID, "voice.silk", "audio/silk", silk)

	case MessageItemTypeFile:
		key, err := genericMediaAESKey(item.FileItem.Media)
		if err != nil {
			return "", err
		}
		data, err := c.downloadAndDecryptCDNBuffer(
			ctx,
			item.FileItem.Media.EncryptQueryParam,
			item.FileItem.Media.FullURL,
			key,
		)
		if err != nil {
			return "", err
		}
		filename := item.FileItem.FileName
		if filename == "" {
			filename = "file.bin"
		}
		contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
		return c.storeInboundBytes(chatID, messageID, filename, contentType, data)

	case MessageItemTypeVideo:
		key, err := genericMediaAESKey(item.VideoItem.Media)
		if err != nil {
			return "", err
		}
		data, err := c.downloadAndDecryptCDNBuffer(
			ctx,
			item.VideoItem.Media.EncryptQueryParam,
			item.VideoItem.Media.FullURL,
			key,
		)
		if err != nil {
			return "", err
		}
		return c.storeInboundBytes(chatID, messageID, "video.mp4", "video/mp4", data)
	}

	return "", nil
}

func outboundMediaKind(partType, filename, contentType string) int {
	switch strings.ToLower(strings.TrimSpace(partType)) {
	case "image":
		return UploadMediaTypeImage
	case "video":
		return UploadMediaTypeVideo
	}

	ct := strings.ToLower(contentType)
	switch {
	case strings.HasPrefix(ct, "image/"):
		return UploadMediaTypeImage
	case strings.HasPrefix(ct, "video/"):
		return UploadMediaTypeVideo
	default:
		return UploadMediaTypeFile
	}
}

func detectLocalContentType(localPath, hintContentType string) string {
	if strings.TrimSpace(hintContentType) != "" {
		return hintContentType
	}
	if kind, err := filetype.MatchFile(localPath); err == nil && kind != filetype.Unknown {
		return kind.MIME.Value
	}
	if ext := filepath.Ext(localPath); ext != "" {
		if ct := mime.TypeByExtension(strings.ToLower(ext)); ct != "" {
			return ct
		}
	}
	return "application/octet-stream"
}

func downloadFilenameFromURL(rawURL, fallback string) string {
	if fallback = sanitizeFilename(fallback); fallback != "" {
		return fallback
	}
	parsed, err := url.Parse(rawURL)
	if err == nil {
		if base := sanitizeFilename(path.Base(parsed.Path)); base != "" {
			return base
		}
	}
	return "remote-media"
}

func (c *WeixinChannel) downloadRemoteMediaToTemp(
	ctx context.Context,
	rawURL,
	fallbackName string,
) (string, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", "", "", err
	}
	resp, err := c.api.HttpClient.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", "", "", fmt.Errorf("remote media HTTP %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, weixinMediaMaxBytes+1))
	if err != nil {
		return "", "", "", err
	}
	if len(data) > weixinMediaMaxBytes {
		return "", "", "", fmt.Errorf("remote media too large: %d bytes", len(data))
	}

	filename, contentType := detectMediaMetadata(
		data,
		downloadFilenameFromURL(rawURL, fallbackName),
		resp.Header.Get("Content-Type"),
	)
	tmpPath, err := writeManagedTempFile("weixin-remote", filename, data)
	if err != nil {
		return "", "", "", err
	}
	return tmpPath, filename, contentType, nil
}

func (c *WeixinChannel) resolveOutboundPart(
	ctx context.Context,
	part bus.MediaPart,
) (string, string, string, func(), error) {
	cleanup := func() {}
	filename := sanitizeFilename(part.Filename)
	contentType := strings.TrimSpace(part.ContentType)

	switch {
	case strings.HasPrefix(part.Ref, "http://") || strings.HasPrefix(part.Ref, "https://"):
		localPath, name, ct, err := c.downloadRemoteMediaToTemp(ctx, part.Ref, filename)
		if err != nil {
			return "", "", "", cleanup, err
		}
		return localPath, name, ct, func() { os.Remove(localPath) }, nil

	case strings.HasPrefix(part.Ref, "media://"):
		store := c.GetMediaStore()
		if store == nil {
			return "", "", "", cleanup, fmt.Errorf("no media store available")
		}
		localPath, meta, err := store.ResolveWithMeta(part.Ref)
		if err != nil {
			return "", "", "", cleanup, err
		}
		if filename == "" {
			filename = sanitizeFilename(meta.Filename)
		}
		if contentType == "" {
			contentType = meta.ContentType
		}
		if strings.HasPrefix(localPath, "http://") || strings.HasPrefix(localPath, "https://") {
			tmpPath, name, ct, err := c.downloadRemoteMediaToTemp(ctx, localPath, filename)
			if err != nil {
				return "", "", "", cleanup, err
			}
			return tmpPath, name, ct, func() { os.Remove(tmpPath) }, nil
		}
		if filename == "" {
			filename = sanitizeFilename(filepath.Base(localPath))
		}
		if contentType == "" {
			contentType = detectLocalContentType(localPath, "")
		}
		return localPath, filename, contentType, cleanup, nil

	case strings.HasPrefix(part.Ref, "file://"):
		u, err := url.Parse(part.Ref)
		if err != nil {
			return "", "", "", cleanup, err
		}
		localPath := u.Path
		if filename == "" {
			filename = sanitizeFilename(filepath.Base(localPath))
		}
		if contentType == "" {
			contentType = detectLocalContentType(localPath, "")
		}
		return localPath, filename, contentType, cleanup, nil

	default:
		localPath := part.Ref
		if filename == "" {
			filename = sanitizeFilename(filepath.Base(localPath))
		}
		if contentType == "" {
			contentType = detectLocalContentType(localPath, "")
		}
		return localPath, filename, contentType, cleanup, nil
	}
}

func (c *WeixinChannel) uploadLocalFile(
	ctx context.Context,
	localPath,
	filename,
	toUserID string,
	mediaType int,
) (*uploadedFileInfo, error) {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, err
	}
	if len(data) > weixinMediaMaxBytes {
		return nil, fmt.Errorf("media too large: %d bytes", len(data))
	}

	filekey, err := randomHex(16)
	if err != nil {
		return nil, err
	}
	aesKey := make([]byte, 16)
	if _, readErr := rand.Read(aesKey); readErr != nil {
		return nil, readErr
	}
	aesKeyHex := hex.EncodeToString(aesKey)
	rawMD5 := md5.Sum(data)

	resp, err := c.api.GetUploadUrl(ctx, GetUploadUrlReq{
		Filekey:     filekey,
		MediaType:   mediaType,
		ToUserID:    toUserID,
		Rawsize:     int64(len(data)),
		RawfileMD5:  hex.EncodeToString(rawMD5[:]),
		Filesize:    aesEcbPaddedSize(int64(len(data))),
		NoNeedThumb: true,
		Aeskey:      aesKeyHex,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("getuploadurl returned nil response")
	}
	if resp.Ret != 0 || resp.Errcode != 0 {
		if isSessionExpiredStatus(resp.Ret, resp.Errcode) {
			c.pauseSession("getuploadurl", resp.Ret, resp.Errcode, resp.Errmsg)
		}
		return nil, fmt.Errorf("getuploadurl failed: ret=%d errcode=%d errmsg=%s", resp.Ret, resp.Errcode, resp.Errmsg)
	}
	uploadParam := strings.TrimSpace(resp.UploadParam)
	uploadFullURL := strings.TrimSpace(resp.UploadFullURL)
	if uploadParam == "" && uploadFullURL == "" {
		return nil, fmt.Errorf("getuploadurl returned no upload URL")
	}

	downloadParam, err := c.uploadBufferToCDN(ctx, data, uploadParam, uploadFullURL, filekey, aesKey)
	if err != nil {
		return nil, err
	}

	return &uploadedFileInfo{
		downloadParam: downloadParam,
		aesKeyHex:     aesKeyHex,
		fileSize:      int64(len(data)),
		cipherSize:    aesEcbPaddedSize(int64(len(data))),
		filename:      filename,
	}, nil
}

func (c *WeixinChannel) uploadBufferToCDN(
	ctx context.Context,
	plaintext []byte,
	uploadParam,
	uploadFullURL,
	filekey string,
	aesKey []byte,
) (string, error) {
	ciphertext, err := encryptAESECB(plaintext, aesKey)
	if err != nil {
		return "", err
	}

	uploadURL := strings.TrimSpace(uploadFullURL)
	if uploadURL == "" {
		if strings.TrimSpace(uploadParam) == "" {
			return "", fmt.Errorf("missing CDN upload URL")
		}
		uploadURL = buildCDNUploadURL(c.cdnBaseURL(), uploadParam, filekey)
	}
	var lastErr error

	for attempt := 1; attempt <= weixinUploadRetryMax; attempt++ {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(ciphertext))
		if reqErr != nil {
			return "", reqErr
		}
		req.Header.Set("Content-Type", "application/octet-stream")

		resp, doErr := c.api.HttpClient.Do(req)
		if doErr != nil {
			lastErr = doErr
		} else {
			func() {
				defer resp.Body.Close()
				if resp.StatusCode >= 400 && resp.StatusCode < 500 {
					body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
					lastErr = fmt.Errorf(
						"cdn upload client error %d: %s",
						resp.StatusCode,
						strings.TrimSpace(string(body)),
					)
					return
				}
				if resp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
					lastErr = fmt.Errorf(
						"cdn upload server error %d: %s",
						resp.StatusCode,
						strings.TrimSpace(string(body)),
					)
					return
				}
				if encrypted := strings.TrimSpace(resp.Header.Get("X-Encrypted-Param")); encrypted != "" {
					lastErr = nil
					uploadParam = encrypted
					return
				}
				lastErr = fmt.Errorf("cdn upload missing x-encrypted-param header")
			}()
		}

		if lastErr == nil {
			return uploadParam, nil
		}
		if strings.Contains(lastErr.Error(), "client error") || attempt == weixinUploadRetryMax {
			break
		}
	}

	return "", lastErr
}

func (c *WeixinChannel) sendMessageItem(
	ctx context.Context,
	toUserID,
	contextToken string,
	item MessageItem,
) error {
	resp, err := c.api.SendMessage(ctx, SendMessageReq{
		Msg: WeixinMessage{
			ToUserID:     toUserID,
			ClientID:     "picoclaw-" + uuid.New().String(),
			MessageType:  MessageTypeBot,
			MessageState: MessageStateFinish,
			ItemList:     []MessageItem{item},
			ContextToken: contextToken,
		},
	})
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("sendmessage returned nil response")
	}
	if resp.Ret != 0 || resp.Errcode != 0 {
		if isSessionExpiredStatus(resp.Ret, resp.Errcode) {
			c.pauseSession("sendmessage", resp.Ret, resp.Errcode, resp.Errmsg)
		}
		return fmt.Errorf("sendmessage failed: ret=%d errcode=%d errmsg=%s", resp.Ret, resp.Errcode, resp.Errmsg)
	}
	return nil
}

func (c *WeixinChannel) sendTextMessage(
	ctx context.Context,
	toUserID,
	contextToken,
	text string,
) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	return c.sendMessageItem(ctx, toUserID, contextToken, MessageItem{
		Type: MessageItemTypeText,
		TextItem: &TextItem{
			Text: text,
		},
	})
}

func encodeWeixinOutboundAESKey(aesKeyHex string) string {
	return base64.StdEncoding.EncodeToString([]byte(aesKeyHex))
}

func (c *WeixinChannel) sendUploadedMedia(
	ctx context.Context,
	toUserID,
	contextToken,
	caption string,
	mediaType int,
	uploaded *uploadedFileInfo,
) error {
	if err := c.sendTextMessage(ctx, toUserID, contextToken, caption); err != nil {
		return err
	}

	mediaRef := &CDNMedia{
		EncryptQueryParam: uploaded.downloadParam,
		AesKey:            encodeWeixinOutboundAESKey(uploaded.aesKeyHex),
		EncryptType:       1,
	}

	switch mediaType {
	case UploadMediaTypeImage:
		return c.sendMessageItem(ctx, toUserID, contextToken, MessageItem{
			Type: MessageItemTypeImage,
			ImageItem: &ImageItem{
				Media:   mediaRef,
				MidSize: uploaded.cipherSize,
			},
		})

	case UploadMediaTypeVideo:
		return c.sendMessageItem(ctx, toUserID, contextToken, MessageItem{
			Type: MessageItemTypeVideo,
			VideoItem: &VideoItem{
				Media:     mediaRef,
				VideoSize: uploaded.cipherSize,
			},
		})

	default:
		return c.sendMessageItem(ctx, toUserID, contextToken, MessageItem{
			Type: MessageItemTypeFile,
			FileItem: &FileItem{
				Media:    mediaRef,
				FileName: uploaded.filename,
				Len:      fmt.Sprintf("%d", uploaded.fileSize),
			},
		})
	}
}

func (c *WeixinChannel) sendTypingStatus(
	ctx context.Context,
	chatID,
	typingTicket string,
	status int,
) error {
	resp, err := c.api.SendTyping(ctx, SendTypingReq{
		IlinkUserID:  chatID,
		TypingTicket: typingTicket,
		Status:       status,
	})
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("sendtyping returned nil response")
	}
	if resp.Ret != 0 || resp.Errcode != 0 {
		if isSessionExpiredStatus(resp.Ret, resp.Errcode) {
			c.pauseSession("sendtyping", resp.Ret, resp.Errcode, resp.Errmsg)
		}
		return fmt.Errorf("sendtyping failed: ret=%d errcode=%d errmsg=%s", resp.Ret, resp.Errcode, resp.Errmsg)
	}
	return nil
}

// StartTyping implements channels.TypingCapable.
func (c *WeixinChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	if strings.TrimSpace(chatID) == "" {
		return func() {}, nil
	}
	if c.remainingPause() > 0 {
		return func() {}, nil
	}

	ticket, err := c.getTypingTicket(ctx, chatID)
	if err != nil {
		if ticket == "" {
			return func() {}, err
		}
		logger.DebugCF("weixin", "GetConfig refresh failed; using cached typing ticket", map[string]any{
			"chat_id": chatID,
			"error":   err.Error(),
		})
	}
	if ticket == "" {
		return func() {}, nil
	}

	typingCtx, cancel := context.WithCancel(ctx)
	var once sync.Once
	stop := func() {
		once.Do(func() {
			cancel()
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer stopCancel()
			if err := c.sendTypingStatus(stopCtx, chatID, ticket, TypingStatusCancel); err != nil {
				logger.DebugCF("weixin", "Failed to cancel typing indicator", map[string]any{
					"chat_id": chatID,
					"error":   err.Error(),
				})
			}
		})
	}

	if err := c.sendTypingStatus(typingCtx, chatID, ticket, TypingStatusTyping); err != nil {
		stop()
		return func() {}, err
	}

	ticker := time.NewTicker(weixinTypingKeepAlive)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				if err := c.sendTypingStatus(typingCtx, chatID, ticket, TypingStatusTyping); err != nil {
					logger.DebugCF("weixin", "Failed to refresh typing indicator", map[string]any{
						"chat_id": chatID,
						"error":   err.Error(),
					})
				}
			}
		}
	}()

	return stop, nil
}

// SendMedia implements channels.MediaSender.
func (c *WeixinChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, basechannels.ErrNotRunning
	}
	if err := c.ensureSessionActive(); err != nil {
		return nil, err
	}

	contextToken := ""
	if v, ok := c.contextTokens.Load(msg.ChatID); ok {
		contextToken, _ = v.(string)
	}
	if contextToken == "" {
		return nil, fmt.Errorf(
			"weixin send media: missing context token for chat %s: %w",
			msg.ChatID,
			basechannels.ErrSendFailed,
		)
	}

	for _, part := range msg.Parts {
		localPath, filename, contentType, cleanup, err := c.resolveOutboundPart(ctx, part)
		if err != nil {
			logger.ErrorCF("weixin", "Failed to resolve outbound media", map[string]any{
				"chat_id": msg.ChatID,
				"ref":     part.Ref,
				"error":   err.Error(),
			})
			return nil, fmt.Errorf("weixin send media: %w", basechannels.ErrSendFailed)
		}
		func() {
			if cleanup != nil {
				defer cleanup()
			}

			kind := outboundMediaKind(part.Type, filename, contentType)
			uploaded, uploadErr := c.uploadLocalFile(ctx, localPath, filename, msg.ChatID, kind)
			if uploadErr != nil {
				err = uploadErr
				return
			}
			err = c.sendUploadedMedia(ctx, msg.ChatID, contextToken, part.Caption, kind, uploaded)
		}()
		if err != nil {
			logger.ErrorCF("weixin", "Failed to send outbound media", map[string]any{
				"chat_id": msg.ChatID,
				"ref":     part.Ref,
				"error":   err.Error(),
			})
			if c.remainingPause() > 0 {
				return nil, fmt.Errorf("weixin send media: %w", basechannels.ErrSendFailed)
			}
			return nil, fmt.Errorf("weixin send media: %w", basechannels.ErrTemporary)
		}
	}

	return nil, nil
}
