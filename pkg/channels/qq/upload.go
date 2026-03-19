package qq

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tencent-connect/botgo/constant"
	"github.com/tencent-connect/botgo/dto"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
)

func qqFileType(mediaType string) uint64 {
	switch mediaType {
	case "image":
		return 1
	case "video":
		return 2
	case "audio":
		return 3
	default:
		return 4
	}
}

func (c *QQChannel) uploadLocalMedia(
	ctx context.Context,
	chatKind string,
	chatID string,
	partType string,
	filename string,
	localPath string,
) ([]byte, error) {
	content, err := os.ReadFile(localPath)
	if err != nil {
		return nil, fmt.Errorf("read local media: %w", err)
	}

	filename = strings.TrimSpace(filename)
	if filename == "" {
		filename = filepath.Base(localPath)
	} else {
		filename = filepath.Base(filename)
	}

	payload := map[string]any{
		"file_type":    qqFileType(partType),
		"file_name":    filename,
		"srv_send_msg": false,
		"file_data":    base64.StdEncoding.EncodeToString(content),
	}

	respBody, err := c.api.Transport(ctx, http.MethodPost, c.uploadURL(chatKind, chatID), payload)
	if err != nil {
		return nil, fmt.Errorf("upload local media: %w", err)
	}

	var uploaded dto.Message
	if err := json.Unmarshal(respBody, &uploaded); err != nil {
		return nil, fmt.Errorf("decode upload response: %w", err)
	}
	if len(uploaded.FileInfo) == 0 {
		return nil, fmt.Errorf("upload local media: empty file_info")
	}

	return uploaded.FileInfo, nil
}

func (c *QQChannel) uploadURL(chatKind, chatID string) string {
	if chatKind == "group" {
		return fmt.Sprintf("%s/v2/groups/%s/files", constant.APIDomain, chatID)
	}
	return fmt.Sprintf("%s/v2/users/%s/files", constant.APIDomain, chatID)
}

func (c *QQChannel) sendUploadedMedia(
	ctx context.Context,
	chatKind string,
	chatID string,
	part bus.MediaPart,
	fileInfo []byte,
) error {
	msgToCreate := &dto.MessageToCreate{
		Content: part.Caption,
		MsgType: dto.RichMediaMsg,
		Media: &dto.MediaInfo{
			FileInfo: fileInfo,
		},
	}

	c.applyReplyContext(chatID, msgToCreate)

	var err error
	if chatKind == "group" {
		_, err = c.api.PostGroupMessage(ctx, chatID, msgToCreate)
	} else {
		_, err = c.api.PostC2CMessage(ctx, chatID, msgToCreate)
	}
	if err != nil {
		return fmt.Errorf("qq send uploaded media: %w", channels.ErrTemporary)
	}

	return nil
}
