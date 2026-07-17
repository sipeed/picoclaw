package deltachat

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
)

// listen is the inbound message loop. It blocks on wait_next_msgs and feeds
// each new message into the PicoClaw inbound pipeline.
func (c *DeltaChatChannel) listen() {
	logger.InfoCF("deltachat", "Listening for messages", map[string]any{
		"account_id": c.accountID,
		"email":      c.selfAddr,
	})
	for c.IsRunning() && c.ctx.Err() == nil {
		raw, err := c.rpc.call(c.ctx, "wait_next_msgs", c.accountID)
		if err != nil {
			if c.ctx.Err() != nil || !c.IsRunning() {
				return
			}
			logger.ErrorCF("deltachat", "wait_next_msgs failed", map[string]any{"error": err.Error()})
			time.Sleep(time.Second)
			continue
		}

		var messageIDs []int64
		if err := json.Unmarshal(raw, &messageIDs); err != nil {
			continue
		}

		if len(messageIDs) > 0 {
			logger.DebugCF("deltachat", "Received message batch", map[string]any{
				"count": len(messageIDs),
			})
		}
		for _, messageID := range messageIDs {
			c.handleMessage(messageID)
		}
	}
}

// handleMessage fetches one message, applies inbound filtering, and publishes it.
func (c *DeltaChatChannel) handleMessage(messageID int64) {
	msg, err := c.getMessage(messageID)
	if err != nil {
		logger.DebugCF("deltachat", "get_message failed", map[string]any{
			"message_id": messageID,
			"error":      err.Error(),
		})
		return
	}

	if msg.IsInfo || (strings.TrimSpace(msg.Text) == "" && msg.File == "") {
		return
	}

	senderAddr := ""
	if msg.Sender != nil {
		senderAddr = msg.Sender.Address
	}
	if senderAddr != "" && strings.EqualFold(senderAddr, c.selfAddr) {
		logger.DebugCF("deltachat", "Drop: own message", map[string]any{"message_id": messageID})
		return
	}

	chat, err := c.getFullChat(msg.ChatID)
	if err != nil {
		logger.DebugCF("deltachat", "get_full_chat_by_id failed", map[string]any{
			"chat_id": msg.ChatID,
			"error":   err.Error(),
		})
		return
	}
	// Device messages are core-generated notices, not real conversations.
	if chat.IsDeviceChat {
		logger.DebugCF("deltachat", "Drop: device message", map[string]any{"chat_id": msg.ChatID})
		return
	}
	isGroup := chat.ChatType != chatTypeSingle

	logger.DebugCF("deltachat", "Inbound message", map[string]any{
		"message_id": messageID,
		"chat_id":    msg.ChatID,
		"from":       senderAddr,
		"is_group":   isGroup,
		"has_file":   msg.File != "",
		"text_len":   len(strings.TrimSpace(msg.Text)),
	})

	senderName := senderAddr
	if msg.Sender != nil {
		if msg.Sender.DisplayName != "" {
			senderName = msg.Sender.DisplayName
		} else if msg.Sender.Name != "" {
			senderName = msg.Sender.Name
		}
	}
	if senderName == "" {
		senderName = "unknown"
	}

	chatID := strconv.FormatInt(msg.ChatID, 10)
	messageIDStr := strconv.FormatInt(msg.ID, 10)

	content := strings.TrimSpace(msg.Text)

	// Register any attachment with the media store so the agent pipeline can
	// view images and operate on files. The ref is scoped to the same key the
	// BaseChannel derives for this message, so it is released with the turn.
	var mediaRefs []string
	if msg.File != "" {
		scope := channels.BuildMediaScope(c.Name(), chatID, messageIDStr)
		if ref := c.registerInboundFile(scope, msg); ref != "" {
			mediaRefs = append(mediaRefs, ref)
		} else {
			// Fallback when no media store is available: surface the path inline
			// so the attachment is not silently lost.
			annotation := fmt.Sprintf("[attachment: %s]", msg.File)
			if content == "" {
				content = annotation
			} else {
				content = content + "\n" + annotation
			}
		}
	}

	// A file with no caption still warrants a turn; give the agent a minimal
	// placeholder so the message survives the empty-content guard below. Audio
	// gets a "[voice]" tag specifically, so the agent's transcription step can
	// substitute the transcript in place rather than appending it.
	if content == "" && len(mediaRefs) > 0 {
		if utils.IsAudioFile(msg.FileName, msg.FileMime) {
			content = "[voice]"
		} else {
			content = "[media]"
		}
	}

	sender := bus.SenderInfo{
		Platform:    config.ChannelDeltaChat,
		PlatformID:  senderAddr,
		CanonicalID: identity.BuildCanonicalID(config.ChannelDeltaChat, senderAddr),
		Username:    senderAddr,
		DisplayName: senderName,
	}

	if !c.IsAllowedSender(sender) {
		logger.DebugCF("deltachat", "Drop: sender not in allow_from", map[string]any{
			"from": senderAddr,
		})
		return
	}

	isMentioned := false
	if isGroup {
		botName := c.config.DisplayName
		if botName == "" {
			botName = c.selfAddr
		}
		isMentioned = mentionsBot(content, botName, c.selfAddr)
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, content)
		if !respond {
			logger.DebugCF("deltachat", "Drop: group trigger not satisfied", map[string]any{
				"chat_id":   msg.ChatID,
				"mentioned": isMentioned,
			})
			return
		}
		content = cleaned
	}

	if strings.TrimSpace(content) == "" {
		return
	}

	metadata := map[string]string{
		"platform":  config.ChannelDeltaChat,
		"chat_name": chat.Name,
	}
	if msg.File != "" {
		metadata["file"] = msg.File
		metadata["file_name"] = msg.FileName
		metadata["file_mime"] = msg.FileMime
	}

	inboundCtx := bus.InboundContext{
		Channel:   config.ChannelDeltaChat,
		ChatID:    chatID,
		SenderID:  senderAddr,
		MessageID: messageIDStr,
		Mentioned: isMentioned,
		Raw:       metadata,
	}
	if isGroup {
		inboundCtx.ChatType = "group"
	} else {
		inboundCtx.ChatType = "direct"
	}

	logger.DebugCF("deltachat", "Dispatching to agent", map[string]any{
		"chat_id":   chatID,
		"chat_type": inboundCtx.ChatType,
		"from":      senderAddr,
	})
	if err := c.HandleInboundContext(c.ctx, chatID, content, mediaRefs, inboundCtx, sender); err != nil {
		logger.ErrorCF("deltachat", "Dispatch failed; leaving message unseen", map[string]any{
			"message_id": messageID,
			"chat_id":    chatID,
			"error":      err.Error(),
		})
		return
	}
	if _, err := c.rpc.call(c.ctx, "markseen_msgs", c.accountID, []int64{messageID}); err != nil {
		logger.WarnCF("deltachat", "Failed to mark message seen", map[string]any{
			"message_id": messageID,
			"chat_id":    chatID,
			"error":      err.Error(),
		})
	}
}

// registerInboundFile records an inbound attachment with the media store under
// the given scope and returns its media:// ref. Returns "" when there is no
// media store or registration fails, letting the caller fall back to an inline
// path annotation.
//
// Delta Chat stores attachments inside the account directory, next to the
// credential database — a location tools are intentionally NOT allowed to read.
// We therefore copy the single attachment out into the shared media temp dir
// (which read_file/load_image are permitted to access) and register that copy,
// so the agent can actually open the file. The copy is store-managed and deleted
// when the turn's scope is released.
func (c *DeltaChatChannel) registerInboundFile(scope string, msg *dcMessage) string {
	store := c.GetMediaStore()
	if store == nil {
		return ""
	}
	filename := msg.FileName
	if filename == "" {
		filename = filepath.Base(msg.File)
	}

	localPath, err := copyToMediaTemp(msg.File, filename)
	if err != nil {
		logger.WarnCF("deltachat", "Failed to copy attachment into media dir", map[string]any{
			"file":  msg.File,
			"error": err.Error(),
		})
		return ""
	}

	ref, err := store.Store(localPath, media.MediaMeta{
		Filename:      filename,
		ContentType:   msg.FileMime,
		Source:        config.ChannelDeltaChat,
		CleanupPolicy: media.CleanupPolicyDeleteOnCleanup,
	}, scope)
	if err != nil {
		logger.WarnCF("deltachat", "Failed to register attachment with media store", map[string]any{
			"file":  localPath,
			"error": err.Error(),
		})
		_ = os.Remove(localPath)
		return ""
	}
	return ref
}

// copyToMediaTemp copies srcPath into the shared media temp directory under a
// unique name and returns the destination path. The media temp dir is the
// location the read_file/load_image tools are permitted to read, so copying here
// makes the attachment readable without exposing Delta Chat's account directory.
func copyToMediaTemp(srcPath, filename string) (string, error) {
	if err := os.MkdirAll(media.TempDir(), 0o700); err != nil {
		return "", err
	}
	safe := utils.SanitizeFilename(filename)
	if safe == "" {
		safe = filepath.Base(srcPath)
	}
	dstPath := filepath.Join(media.TempDir(), uuid.NewString()[:8]+"_"+safe)

	src, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		_ = os.Remove(dstPath)
		return "", err
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(dstPath)
		return "", err
	}
	return dstPath, nil
}

func (c *DeltaChatChannel) getMessage(messageID int64) (*dcMessage, error) {
	raw, err := c.rpc.call(c.ctx, "get_message", c.accountID, messageID)
	if err != nil {
		return nil, err
	}
	var msg dcMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (c *DeltaChatChannel) getFullChat(chatID int64) (*dcChat, error) {
	raw, err := c.rpc.call(c.ctx, "get_full_chat_by_id", c.accountID, chatID)
	if err != nil {
		return nil, err
	}
	var chat dcChat
	if err := json.Unmarshal(raw, &chat); err != nil {
		return nil, err
	}
	return &chat, nil
}

// mentionsBot reports whether the message references the bot by display name or
// the local-part of its email address (a common addressing convention).
func mentionsBot(content, displayName, email string) bool {
	if containsMentionToken(content, displayName) {
		return true
	}
	if local, _, ok := strings.Cut(email, "@"); ok && local != "" {
		if containsMentionToken(content, "@"+local) {
			return true
		}
	}
	return false
}

func containsMentionToken(content, token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	contentRunes := []rune(strings.ToLower(content))
	tokenRunes := []rune(strings.ToLower(token))
	if len(tokenRunes) == 0 || len(tokenRunes) > len(contentRunes) {
		return false
	}
	for i := 0; i <= len(contentRunes)-len(tokenRunes); i++ {
		if !sameRunes(contentRunes[i:i+len(tokenRunes)], tokenRunes) {
			continue
		}
		before := i == 0 || !isMentionWordRune(contentRunes[i-1])
		afterIdx := i + len(tokenRunes)
		after := afterIdx >= len(contentRunes) || !isMentionWordRune(contentRunes[afterIdx])
		if before && after {
			return true
		}
	}
	return false
}

func sameRunes(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isMentionWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
