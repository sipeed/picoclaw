package wecom

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// Upload temporary media constants.
// Ref: https://developer.work.weixin.qq.com/document/path/101463#上传临时素材
const (
	// wsUploadChunkSize is the maximum raw (pre-base64) bytes per chunk.
	// The API limit is 512 KB before base64 encoding.
	wsUploadChunkSize = 512 << 10 // 512 KB

	// wsUploadMaxChunks is the maximum number of chunks per upload session.
	wsUploadMaxChunks = 100

	wsUploadInitTimeout   = 30 * time.Second
	wsUploadChunkTimeout  = 60 * time.Second // larger: base64 payload may be big
	wsUploadFinishTimeout = 30 * time.Second
)

// ---- Request / response body types ----

// wsUploadInitBody is the body for aibot_upload_media_init.
type wsUploadInitBody struct {
	Type        string `json:"type"`
	Filename    string `json:"filename"`
	TotalSize   int64  `json:"total_size"`
	TotalChunks int    `json:"total_chunks"`
	MD5         string `json:"md5,omitempty"`
}

// wsUploadInitResponse is the body received in the aibot_upload_media_init response.
type wsUploadInitResponse struct {
	UploadID string `json:"upload_id"`
}

// wsUploadChunkBody is the body for aibot_upload_media_chunk.
type wsUploadChunkBody struct {
	UploadID   string `json:"upload_id"`
	ChunkIndex int    `json:"chunk_index"`
	Base64Data string `json:"base64_data"`
}

// wsUploadFinishBody is the body for aibot_upload_media_finish.
type wsUploadFinishBody struct {
	UploadID string `json:"upload_id"`
}

// wsUploadFinishResponse is the body received in the aibot_upload_media_finish response.
type wsUploadFinishResponse struct {
	Type      string `json:"type"`
	MediaID   string `json:"media_id"`
	CreatedAt int64  `json:"created_at"` // Unix timestamp (seconds)
}

// ---- Public API ----

// UploadWSMedia uploads a local file as a temporary WeCom media asset via the
// WebSocket long-connection API and returns the resulting media_id (valid 3 days).
//
// mediaType must be one of:
//   - "image"  — PNG / JPG(JPEG) / GIF, ≤ 2 MB
//   - "voice"  — AMR, ≤ 2 MB
//   - "video"  — MP4, ≤ 10 MB
//   - "file"   — any format, ≤ 20 MB
//
// filename is the logical file name sent to WeCom. When empty it is derived
// from the base name of filePath.
//
// The upload is split into at most wsUploadMaxChunks chunks of wsUploadChunkSize
// bytes each. The full operation must complete within wsUploadSessionTTL (30 min).
// Each individual command call is also independently time-bounded.
func (c *WeComAIBotWSChannel) UploadWSMedia(
	ctx context.Context,
	filePath, filename, mediaType string,
) (string, error) {
	// ---- Validate media type ----
	switch mediaType {
	case "image", "voice", "video", "file":
	default:
		return "", fmt.Errorf("unsupported media type %q: must be image, voice, video, or file", mediaType)
	}

	// ---- Read file ----
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	totalSize := int64(len(data))
	if totalSize == 0 {
		return "", fmt.Errorf("file is empty: %s", filePath)
	}

	if filename == "" {
		filename = filepath.Base(filePath)
	}

	// ---- Compute chunk count ----
	totalChunks := int((totalSize + wsUploadChunkSize - 1) / wsUploadChunkSize)
	if totalChunks > wsUploadMaxChunks {
		return "", fmt.Errorf(
			"file too large: requires %d chunks but limit is %d (max ~%d MB)",
			totalChunks, wsUploadMaxChunks,
			int64(wsUploadMaxChunks)*wsUploadChunkSize>>20,
		)
	}

	// ---- Compute MD5 ----
	rawMD5 := md5.Sum(data) //nolint:gosec // MD5 is required by the WeCom API spec
	fileMD5 := fmt.Sprintf("%x", rawMD5)

	// ---- Step 1: Init ----
	initEnv, err := c.callWSCommand(wsCommand{
		Cmd:     "aibot_upload_media_init",
		Headers: wsHeaders{ReqID: wsGenerateID()},
		Body: wsUploadInitBody{
			Type:        mediaType,
			Filename:    filename,
			TotalSize:   totalSize,
			TotalChunks: totalChunks,
			MD5:         fileMD5,
		},
	}, wsUploadInitTimeout)
	if err != nil {
		return "", fmt.Errorf("upload init: %w", err)
	}
	var initResp wsUploadInitResponse
	if err = json.Unmarshal(initEnv.Body, &initResp); err != nil {
		return "", fmt.Errorf("parse upload init response: %w", err)
	}
	if initResp.UploadID == "" {
		return "", fmt.Errorf("upload init returned empty upload_id")
	}
	uploadID := initResp.UploadID

	logger.InfoCF("wecom_aibot", "Media upload initialized", map[string]any{
		"upload_id":    uploadID,
		"type":         mediaType,
		"filename":     filename,
		"total_size":   totalSize,
		"total_chunks": totalChunks,
	})

	// ---- Step 2: Upload chunks ----
	for i := 0; i < totalChunks; i++ {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", fmt.Errorf("upload aborted before chunk %d: %w", i, ctxErr)
		}

		start := int64(i) * wsUploadChunkSize
		end := start + wsUploadChunkSize
		if end > totalSize {
			end = totalSize
		}

		_, err = c.callWSCommand(wsCommand{
			Cmd:     "aibot_upload_media_chunk",
			Headers: wsHeaders{ReqID: wsGenerateID()},
			Body: wsUploadChunkBody{
				UploadID:   uploadID,
				ChunkIndex: i,
				Base64Data: base64.StdEncoding.EncodeToString(data[start:end]),
			},
		}, wsUploadChunkTimeout)
		if err != nil {
			return "", fmt.Errorf("upload chunk %d/%d: %w", i, totalChunks-1, err)
		}

		logger.DebugCF("wecom_aibot", "Media chunk uploaded", map[string]any{
			"upload_id": uploadID,
			"chunk":     i,
			"total":     totalChunks,
		})
	}

	// ---- Step 3: Finish ----
	finishEnv, err := c.callWSCommand(wsCommand{
		Cmd:     "aibot_upload_media_finish",
		Headers: wsHeaders{ReqID: wsGenerateID()},
		Body:    wsUploadFinishBody{UploadID: uploadID},
	}, wsUploadFinishTimeout)
	if err != nil {
		return "", fmt.Errorf("upload finish: %w", err)
	}
	var finishResp wsUploadFinishResponse
	if err := json.Unmarshal(finishEnv.Body, &finishResp); err != nil {
		return "", fmt.Errorf("parse upload finish response: %w", err)
	}
	if finishResp.MediaID == "" {
		return "", fmt.Errorf("upload finish returned empty media_id")
	}

	logger.InfoCF("wecom_aibot", "Media upload complete", map[string]any{
		"upload_id": uploadID,
		"media_id":  finishResp.MediaID,
		"type":      finishResp.Type,
	})
	return finishResp.MediaID, nil
}

// ---- Internal helper ----

// callWSCommand sends a WebSocket command and returns the raw server envelope.
// It validates the errcode field and returns an error when non-zero.
// Use callWSCommand (over writeWSAndWait) when the response body must be inspected.
func (c *WeComAIBotWSChannel) callWSCommand(cmd wsCommand, timeout time.Duration) (wsEnvelope, error) {
	if cmd.Headers.ReqID == "" {
		return wsEnvelope{}, fmt.Errorf("req_id is empty")
	}
	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()
	if conn == nil {
		return wsEnvelope{}, fmt.Errorf("websocket not connected")
	}
	env, err := c.sendAndWait(conn, cmd.Headers.ReqID, cmd, timeout)
	if err != nil {
		return wsEnvelope{}, err
	}
	if env.ErrCode != 0 {
		return wsEnvelope{}, fmt.Errorf("%s rejected (errcode=%d): %s", cmd.Cmd, env.ErrCode, env.ErrMsg)
	}
	return env, nil
}
