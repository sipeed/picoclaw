package onebot

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"jane/pkg/media"
	"jane/pkg/utils"
)

func isAPIResponse(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s == "ok" || s == "failed"
	}
	var bs BotStatus
	if json.Unmarshal(raw, &bs) == nil {
		return bs.Online || bs.Good
	}
	return false
}

func parseJSONInt64(raw json.RawMessage) (int64, error) {
	if len(raw) == 0 {
		return 0, nil
	}

	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strconv.ParseInt(s, 10, 64)
	}
	return 0, fmt.Errorf("cannot parse as int64: %s", string(raw))
}

func parseJSONString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	return string(raw)
}

func (c *OneBotChannel) parseMessageSegments(
	raw json.RawMessage,
	selfID int64,
	store media.MediaStore,
	scope string,
) parseMessageResult {
	if len(raw) == 0 {
		return parseMessageResult{}
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		mentioned := false
		if selfID > 0 {
			cqAt := fmt.Sprintf("[CQ:at,qq=%d]", selfID)
			if strings.Contains(s, cqAt) {
				mentioned = true
				s = strings.ReplaceAll(s, cqAt, "")
				s = strings.TrimSpace(s)
			}
		}
		return parseMessageResult{Text: s, IsBotMentioned: mentioned}
	}

	var segments []map[string]any
	if err := json.Unmarshal(raw, &segments); err != nil {
		return parseMessageResult{}
	}

	var textParts []string
	mentioned := false
	selfIDStr := strconv.FormatInt(selfID, 10)
	var mediaRefs []string
	var replyTo string

	// Helper to register a local file with the media store
	storeFile := func(localPath, filename string) string {
		if store != nil {
			ref, err := store.Store(localPath, media.MediaMeta{
				Filename: filename,
				Source:   "onebot",
			}, scope)
			if err == nil {
				return ref
			}
		}
		return localPath // fallback
	}

	for _, seg := range segments {
		segType, _ := seg["type"].(string)
		data, _ := seg["data"].(map[string]any)

		switch segType {
		case "text":
			if data != nil {
				if t, ok := data["text"].(string); ok {
					textParts = append(textParts, t)
				}
			}

		case "at":
			if data != nil && selfID > 0 {
				qqVal := fmt.Sprintf("%v", data["qq"])
				if qqVal == selfIDStr || qqVal == "all" {
					mentioned = true
				}
			}

		case "image", "video", "file":
			if data != nil {
				url, _ := data["url"].(string)
				if url != "" {
					defaults := map[string]string{"image": "image.jpg", "video": "video.mp4", "file": "file"}
					filename := defaults[segType]
					if f, ok := data["file"].(string); ok && f != "" {
						filename = f
					} else if n, ok := data["name"].(string); ok && n != "" {
						filename = n
					}
					localPath := utils.DownloadFile(url, filename, utils.DownloadOptions{
						LoggerPrefix: "onebot",
					})
					if localPath != "" {
						mediaRefs = append(mediaRefs, storeFile(localPath, filename))
						textParts = append(textParts, fmt.Sprintf("[%s]", segType))
					}
				}
			}

		case "record":
			if data != nil {
				url, _ := data["url"].(string)
				if url != "" {
					localPath := utils.DownloadFile(url, "voice.amr", utils.DownloadOptions{
						LoggerPrefix: "onebot",
					})
					if localPath != "" {
						textParts = append(textParts, "[voice]")
						mediaRefs = append(mediaRefs, storeFile(localPath, "voice.amr"))
					}
				}
			}

		case "reply":
			if data != nil {
				if id, ok := data["id"]; ok {
					replyTo = fmt.Sprintf("%v", id)
				}
			}

		case "face":
			if data != nil {
				faceID, _ := data["id"]
				textParts = append(textParts, fmt.Sprintf("[face:%v]", faceID))
			}

		case "forward":
			textParts = append(textParts, "[forward message]")

		default:
		}
	}

	return parseMessageResult{
		Text:           strings.TrimSpace(strings.Join(textParts, "")),
		IsBotMentioned: mentioned,
		Media:          mediaRefs,
		ReplyTo:        replyTo,
	}
}
