package qq

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/openapi"
	"golang.org/x/oauth2"
)

// https://bot.q.qq.com/wiki/develop/api-v2/server-inter/message/send-receive/rich-media.html
const (
	apiBase = "https://api.sgroup.qq.com"

	defaultAPITimeout = 30 * time.Second
	fileUploadTimeout = 120 * time.Second

	uploadMaxRetries  = 2
	uploadBaseDelayMs = 1000 // 1s
)

// MediaFileType represents the type of media file
type MediaFileType int

const (
	MediaFileTypeImage MediaFileType = 1
	MediaFileTypeVideo MediaFileType = 2
	MediaFileTypeVoice MediaFileType = 3
	MediaFileTypeFile  MediaFileType = 4
)

// UploadMediaResponse represents the response from uploading media
type UploadMediaResponse struct {
	FileUUID string `json:"file_uuid"`
	FileInfo string `json:"file_info"`
	TTL      int    `json:"ttl"`
	ID       string `json:"id,omitempty"`
}

// MessageResponse represents the response from sending a message
type MessageResponse struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
}

// Client wraps access to the QQ Bot API
type Client struct {
	httpClient  *http.Client
	api         openapi.OpenAPI
	tokenSource oauth2.TokenSource

	// file_info cache (content hash -> file_info)
	fileCache sync.Map // key: string, value: *fileCacheEntry
}

type fileCacheEntry struct {
	fileInfo  string
	expiresAt time.Time
}

// NewClient creates a new QQ Bot API client
func NewClient(api openapi.OpenAPI, tokenSource oauth2.TokenSource) *Client {
	return &Client{
		httpClient:  &http.Client{},
		api:         api,
		tokenSource: tokenSource,
	}
}

func (c *Client) doFetchToken() (string, error) {
	token, err := c.tokenSource.Token()
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

// apiRequest performs an authenticated API request
func (c *Client) apiRequest(ctx context.Context, accessToken, method, path string, body any) ([]byte, error) {
	url := apiBase + path

	// Choose timeout based on whether this is a file upload
	timeout := defaultAPITimeout
	if strings.Contains(path, "/files") {
		timeout = fileUploadTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request [%s]: %w", path, err)
	}
	req.Header.Set("Authorization", "QQBot "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("request timeout [%s]: exceeded %v", path, timeout)
		}
		return nil, fmt.Errorf("network error [%s]: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response [%s]: %w", path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		_ = json.Unmarshal(respBody, &apiErr)
		if apiErr.Message != "" {
			return nil, fmt.Errorf("API error [%s]: %s", path, apiErr.Message)
		}
		return nil, fmt.Errorf("API error [%s]: %s", path, string(respBody))
	}

	return respBody, nil
}

// apiRequestWithRetry wraps apiRequest with exponential backoff retry for upload
func (c *Client) apiRequestWithRetry(ctx context.Context, accessToken, method, path string, body any) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= uploadMaxRetries; attempt++ {
		respBody, err := c.apiRequest(ctx, accessToken, method, path, body)
		if err == nil {
			return respBody, nil
		}

		lastErr = err
		errMsg := err.Error()

		// Fast-fail on non-retriable errors
		if strings.Contains(errMsg, "400") || strings.Contains(errMsg, "401") ||
			strings.Contains(errMsg, "Invalid") || strings.Contains(errMsg, "upload timeout") ||
			strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "Timeout") {
			return nil, lastErr
		}

		if attempt < uploadMaxRetries {
			delay := time.Duration(uploadBaseDelayMs*pow(2, attempt)) * time.Millisecond
			logger.WarnCF("qq", "Upload attempt failed, retrying", map[string]any{
				"attempt": attempt + 1,
				"delay":   delay.String(),
				"error":   truncate(errMsg, 100),
			})

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, lastErr
}

// GetNextMsgSeq generates a unique message sequence number (0~65535)
func GetNextMsgSeq() int {
	timePart := time.Now().UnixMilli() % 100000000
	random := rand.Intn(65536)
	return int((timePart ^ int64(random)) % 65536)
}

// computeFileHash computes SHA-256 hash of the given data
func computeFileHash(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// getCachedFileInfo looks up cached file_info
func (c *Client) getCachedFileInfo(contentHash, scope, targetID string, fileType MediaFileType) (string, bool) {
	key := fmt.Sprintf("%s:%s:%s:%d", contentHash, scope, targetID, fileType)
	val, ok := c.fileCache.Load(key)
	if !ok {
		return "", false
	}
	entry := val.(*fileCacheEntry)
	if time.Now().After(entry.expiresAt) {
		c.fileCache.Delete(key)
		return "", false
	}
	return entry.fileInfo, true
}

// setCachedFileInfo stores file_info in cache
func (c *Client) setCachedFileInfo(contentHash, scope, targetID string, fileType MediaFileType, fileInfo string, ttl int) {
	key := fmt.Sprintf("%s:%s:%s:%d", contentHash, scope, targetID, fileType)
	c.fileCache.Store(key, &fileCacheEntry{
		fileInfo:  fileInfo,
		expiresAt: time.Now().Add(time.Duration(ttl) * time.Second),
	})
}

// uploadC2CMedia uploads a media file for C2C chat
func (c *Client) uploadC2CMedia(
	ctx context.Context,
	accessToken string,
	openid string,
	fileType MediaFileType,
	url string, // public URL (mutually exclusive with fileData)
	fileData string, // base64 encoded data (mutually exclusive with url)
	srvSendMsg bool,
) (*UploadMediaResponse, error) {
	if url == "" && fileData == "" {
		return nil, fmt.Errorf("uploadC2CMedia: url or fileData is required")
	}

	// Check cache if fileData is provided
	if fileData != "" {
		contentHash := computeFileHash(fileData)
		if cached, ok := c.getCachedFileInfo(contentHash, "c2c", openid, fileType); ok {
			logger.InfoC("qq", "uploadC2CMedia: using cached file_info (skip upload)")
			return &UploadMediaResponse{FileInfo: cached}, nil
		}
	}

	body := map[string]any{
		"file_type":    int(fileType),
		"srv_send_msg": srvSendMsg,
	}
	if url != "" {
		body["url"] = url
	} else {
		body["file_data"] = fileData
	}

	path := fmt.Sprintf("/v2/users/%s/files", openid)
	respBody, err := c.apiRequestWithRetry(ctx, accessToken, http.MethodPost, path, body)
	if err != nil {
		return nil, fmt.Errorf("uploadC2CMedia: %w", err)
	}

	var result UploadMediaResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("uploadC2CMedia parse response: %w", err)
	}

	// Store in cache
	if fileData != "" && result.FileInfo != "" && result.TTL > 0 {
		contentHash := computeFileHash(fileData)
		c.setCachedFileInfo(contentHash, "c2c", openid, fileType, result.FileInfo, result.TTL)
	}

	return &result, nil
}

// sendC2CMediaMessage sends a rich media message to a C2C chat
func (c *Client) sendC2CMediaMessage(
	ctx context.Context,
	accessToken string,
	openid string,
	fileInfo string,
	content string,
) (*MessageResponse, error) {

	body := map[string]any{
		"msg_type": dto.RichMediaMsg,
		"media":    map[string]string{"file_info": fileInfo},
	}
	if content != "" {
		body["content"] = content
	}

	path := fmt.Sprintf("/v2/users/%s/messages", openid)
	respBody, err := c.apiRequest(ctx, accessToken, http.MethodPost, path, body)
	if err != nil {
		return nil, fmt.Errorf("sendC2CMediaMessage: %w", err)
	}

	var result MessageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("sendC2CMediaMessage parse response: %w", err)
	}

	return &result, nil
}

// SendC2CMediaMessage uploads a media file and sends it as a C2C message.
func (c *Client) SendC2CMediaMessage(
	ctx context.Context,
	openid string,
	content string,
	fileType MediaFileType,
	fileData []byte,
) (*MessageResponse, error) {
	var uploadResult *UploadMediaResponse
	var err error

	accessToken, err := c.doFetchToken()
	if err != nil {
		return nil, fmt.Errorf("sendC2CMediaMessage get access token: %w", err)
	}
	base64Data := base64.StdEncoding.EncodeToString(fileData)
	uploadResult, err = c.uploadC2CMedia(ctx, accessToken, openid, fileType, "", base64Data, false)
	if err != nil {
		return nil, fmt.Errorf("sendC2CMediaMessage upload: %w", err)
	}

	// Send rich media message
	return c.sendC2CMediaMessage(ctx, accessToken, openid, uploadResult.FileInfo, content)
}

// --- Helper functions ---

func pow(base, exp int) int {
	result := 1
	for i := 0; i < exp; i++ {
		result *= base
	}
	return result
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
