package channels

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/voice"
)

type MatrixChannel struct {
	*BaseChannel
	client       *mautrix.Client
	matrixConfig config.MatrixConfig
	syncer       *mautrix.DefaultSyncer
	stopSyncer   context.CancelFunc
	startTime    time.Time // events before this timestamp are ignored (initial sync flood guard)
	roomNames    sync.Map  // roomID -> room name
	typing       sync.Map  // roomID -> bool (active typing indicator)
	transcriber  voice.Transcriber
}

func NewMatrixChannel(matrixCfg config.MatrixConfig, bus *bus.MessageBus) (*MatrixChannel, error) {
	// Create Matrix client
	client, err := mautrix.NewClient(matrixCfg.Homeserver, id.UserID(matrixCfg.UserID), matrixCfg.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create matrix client: %w", err)
	}

	// Set device ID if provided
	if matrixCfg.DeviceID != "" {
		client.DeviceID = id.DeviceID(matrixCfg.DeviceID)
	}

	base := NewBaseChannel("matrix", matrixCfg, bus, matrixCfg.AllowFrom)

	syncer := client.Syncer.(*mautrix.DefaultSyncer)

	return &MatrixChannel{
		BaseChannel:  base,
		client:       client,
		matrixConfig: matrixCfg,
		syncer:       syncer,
		startTime:    time.Now(),
		roomNames:    sync.Map{},
		typing:       sync.Map{},
		transcriber:  nil,
	}, nil
}

func (c *MatrixChannel) SetTranscriber(transcriber voice.Transcriber) {
	c.transcriber = transcriber
}

func (c *MatrixChannel) Start(ctx context.Context) error {
	logger.InfoC("matrix", "Starting Matrix client...")

	// Set up event handlers
	c.syncer.OnEventType(event.EventMessage, c.handleMessage)
	c.syncer.OnEventType(event.StateMember, c.handleMemberEvent)

	// Create a cancellable context for the syncer
	syncCtx, cancel := context.WithCancel(ctx)
	c.stopSyncer = cancel

	// Start syncing in background
	go func() {
		err := c.client.SyncWithContext(syncCtx)
		if err != nil && syncCtx.Err() == nil {
			logger.ErrorCF("matrix", "Sync error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	c.setRunning(true)
	logger.InfoC("matrix", "Matrix client started successfully")
	return nil
}

func (c *MatrixChannel) Stop(ctx context.Context) error {
	logger.InfoC("matrix", "Stopping Matrix client...")

	if c.stopSyncer != nil {
		c.stopSyncer()
	}

	c.setRunning(false)
	logger.InfoC("matrix", "Matrix client stopped")
	return nil
}

func (c *MatrixChannel) handleMemberEvent(ctx context.Context, evt *event.Event) {
	memberEvt := evt.Content.AsMember()

	// Auto-join rooms if invited and JoinOnInvite is enabled
	if memberEvt.Membership == event.MembershipInvite &&
		evt.GetStateKey() == string(c.client.UserID) &&
		c.matrixConfig.JoinOnInvite {

		roomID := evt.RoomID
		logger.InfoCF("matrix", "Auto-joining room after invite", map[string]interface{}{
			"room_id": roomID.String(),
		})

		_, err := c.client.JoinRoomByID(ctx, roomID)
		if err != nil {
			logger.ErrorCF("matrix", "Failed to join room", map[string]interface{}{
				"room_id": roomID.String(),
				"error":   err.Error(),
			})
		} else {
			logger.InfoCF("matrix", "Successfully joined room", map[string]interface{}{
				"room_id": roomID.String(),
			})
		}
	}
}

func (c *MatrixChannel) handleMessage(ctx context.Context, evt *event.Event) {
	// Ignore our own messages
	if evt.Sender == c.client.UserID {
		return
	}

	// Ignore historical events delivered on initial sync (flood guard).
	// Matrix timestamps are in milliseconds.
	if time.UnixMilli(evt.Timestamp).Before(c.startTime) {
		logger.DebugCF("matrix", "Ignoring historical event", map[string]interface{}{
			"event_id":  evt.ID.String(),
			"event_ts":  evt.Timestamp,
			"start_ts":  c.startTime.UnixMilli(),
		})
		return
	}

	msgEvt := evt.Content.AsMessage()
	roomID := evt.RoomID.String()
	senderID := evt.Sender.String()

	// Ignore edit events (m.replace relations)
	if msgEvt.RelatesTo != nil && msgEvt.RelatesTo.Type == event.RelReplace {
		return
	}

	// Check if sender is allowed
	if !c.IsAllowed(senderID) {
		logger.WarnCF("matrix", "Ignoring message from unauthorized user", map[string]interface{}{
			"sender_id": senderID,
		})
		return
	}

	// Get or cache room name
	roomName := c.getRoomName(ctx, evt.RoomID)

	// Get sender display name
	senderName := c.getUserDisplayName(ctx, evt.RoomID, evt.Sender)

	messageText := msgEvt.Body
	mediaPaths := []string{}
	localFiles := []string{}

	// Clean up temp files when done
	defer func() {
		for _, file := range localFiles {
			if err := os.Remove(file); err != nil {
				logger.DebugCF("matrix", "Failed to cleanup temp file", map[string]interface{}{
					"file":  file,
					"error": err.Error(),
				})
			}
		}
	}()

	// Handle different message types
	switch msgEvt.MsgType {
	case event.MsgText:
		// Text already in messageText

	case event.MsgImage:
		// Download and process image
		if msgEvt.URL != "" {
			imagePath := c.downloadMedia(ctx, msgEvt.URL, msgEvt.Body, ".jpg")
			if imagePath != "" {
				localFiles = append(localFiles, imagePath)
				mediaPaths = append(mediaPaths, imagePath)
				if messageText != "" {
					messageText += "\n"
				}
				messageText += fmt.Sprintf("[image: %s]", msgEvt.Body)
			}
		}

	case event.MsgAudio, event.MsgVideo:
		// Download and transcribe audio/video
		if msgEvt.URL != "" {
			ext := ".ogg"
			if msgEvt.MsgType == event.MsgVideo {
				ext = ".mp4"
			}

			mediaPath := c.downloadMedia(ctx, msgEvt.URL, msgEvt.Body, ext)
			if mediaPath != "" {
				localFiles = append(localFiles, mediaPath)
				mediaPaths = append(mediaPaths, mediaPath)

				// Try transcription for audio/video
				transcribedText := ""
				if c.transcriber != nil && c.transcriber.IsAvailable() {
					tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					defer cancel()

					result, err := c.transcriber.Transcribe(tCtx, mediaPath)
					if err != nil {
						logger.ErrorCF("matrix", "Transcription failed", map[string]interface{}{
							"error": err.Error(),
							"path":  mediaPath,
						})
						transcribedText = fmt.Sprintf("[%s (transcription failed)]", msgEvt.MsgType)
					} else {
						transcribedText = fmt.Sprintf("[%s transcription: %s]", msgEvt.MsgType, result.Text)
						logger.InfoCF("matrix", "Media transcribed successfully", map[string]interface{}{
							"type": msgEvt.MsgType,
							"text": result.Text,
						})
					}
				} else {
					transcribedText = fmt.Sprintf("[%s: %s]", msgEvt.MsgType, msgEvt.Body)
				}

				if messageText != "" {
					messageText += "\n"
				}
				messageText += transcribedText
			}
		}

	case event.MsgFile:
		// Download generic file
		if msgEvt.URL != "" {
			filePath := c.downloadMedia(ctx, msgEvt.URL, msgEvt.Body, "")
			if filePath != "" {
				localFiles = append(localFiles, filePath)
				mediaPaths = append(mediaPaths, filePath)
				if messageText != "" {
					messageText += "\n"
				}
				messageText += fmt.Sprintf("[file: %s]", msgEvt.Body)
			}
		}

	default:
		// Unsupported message type
		logger.DebugCF("matrix", "Ignoring unsupported message type", map[string]interface{}{
			"type": msgEvt.MsgType,
		})
		return
	}

	logger.InfoCF("matrix", "Received message", map[string]interface{}{
		"sender":  senderName,
		"room":    roomName,
		"content": messageText,
		"type":    msgEvt.MsgType,
	})

	// Check if it's a group chat
	memberCount := c.getRoomMemberCount(ctx, evt.RoomID)
	isGroup := memberCount > 2

	// In group chats, check mention requirement
	if isGroup && c.matrixConfig.RequireMentionInGroup {
		mentioned := c.isBotMentioned(msgEvt, c.client.UserID)
		if !mentioned {
			logger.InfoCF("matrix", "Ignoring group message (not mentioned)", map[string]interface{}{
				"room":   roomName,
				"sender": senderName,
			})
			return
		}
		logger.InfoCF("matrix", "Bot mentioned in group chat", map[string]interface{}{
			"room":   roomName,
			"sender": senderName,
		})
		// Remove the mention from the message text
		messageText = c.removeMention(messageText, c.client.UserID)
	}

	// Show typing indicator (native Matrix — no message sent)
	if _, err := c.client.UserTyping(ctx, evt.RoomID, true, 60*time.Second); err != nil {
		logger.WarnCF("matrix", "Failed to send typing indicator", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		c.typing.Store(roomID, true)
	}

	// Prepare metadata
	metadata := map[string]string{
		"sender_name": senderName,
		"room_name":   roomName,
		"timestamp":   fmt.Sprintf("%d", evt.Timestamp),
	}

	if isGroup {
		metadata["is_group_chat"] = "true"
	}

	// Check for reply-to
	replyToID := c.getReplyToID(msgEvt)
	if replyToID != "" {
		metadata["reply_to_msg_id"] = replyToID
	}

	// Handle the message through base channel
	c.HandleMessage(senderID, roomID, messageText, mediaPaths, metadata)
}

// ─── Send (outbound) ──────────────────────────────────────────────────────────

func (c *MatrixChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	roomID := id.RoomID(msg.ChatID)

	// Always clear the typing indicator first
	if _, active := c.typing.LoadAndDelete(msg.ChatID); active {
		if _, err := c.client.UserTyping(ctx, roomID, false, 0); err != nil {
			logger.WarnCF("matrix", "Failed to clear typing indicator", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// 1. Send any media files (each as its own Matrix event)
	for _, mediaPath := range msg.Media {
		if err := c.sendMediaFile(ctx, roomID, mediaPath); err != nil {
			logger.ErrorCF("matrix", "Failed to send media file", map[string]interface{}{
				"error": err.Error(),
				"path":  mediaPath,
			})
		}
	}

	// 2. Send text content
	if msg.Content != "" {
		content := &event.MessageEventContent{
			MsgType: event.MsgText,
			Body:    msg.Content,
		}

		if hasMarkdown(msg.Content) {
			content.Format = event.FormatHTML
			content.FormattedBody = markdownToMatrixHTML(msg.Content)
		}

		_, err := c.client.SendMessageEvent(ctx, roomID, event.EventMessage, content)
		if err != nil {
			return fmt.Errorf("failed to send matrix message: %w", err)
		}
		logger.InfoCF("matrix", "Sent message to room", map[string]interface{}{
			"chat_id": msg.ChatID,
		})
	}

	return nil
}

// ─── Media upload helpers ─────────────────────────────────────────────────────

// sendMediaFile uploads a local file to the Matrix content repository and sends
// it as an appropriate Matrix event (m.image, m.audio, m.video, or m.file).
func (c *MatrixChannel) sendMediaFile(ctx context.Context, roomID id.RoomID, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read media file %q: %w", filePath, err)
	}

	mimeType := detectMIMEType(filePath, data)
	fileName := filepath.Base(filePath)

	logger.InfoCF("matrix", "Uploading media to content repo", map[string]interface{}{
		"path":      filePath,
		"mime_type": mimeType,
		"size":      len(data),
	})

	resp, err := c.client.UploadMedia(ctx, mautrix.ReqUploadMedia{
		ContentBytes: data,
		ContentType:  mimeType,
		FileName:     fileName,
	})
	if err != nil {
		return fmt.Errorf("failed to upload media to Matrix: %w", err)
	}

	mxcURI := resp.ContentURI.CUString()

	// Determine event type based on MIME category
	msgType := mimeToMsgType(mimeType)

	content := &event.MessageEventContent{
		MsgType: msgType,
		Body:    fileName,
		URL:     mxcURI,
		Info: &event.FileInfo{
			MimeType: mimeType,
			Size:     len(data),
		},
	}

	_, err = c.client.SendMessageEvent(ctx, roomID, event.EventMessage, content)
	if err != nil {
		return fmt.Errorf("failed to send media event: %w", err)
	}

	logger.InfoCF("matrix", "Media sent successfully", map[string]interface{}{
		"room_id":   roomID.String(),
		"msg_type":  msgType,
		"mime_type": mimeType,
		"mxc_uri":   string(mxcURI),
	})

	return nil
}

// detectMIMEType guesses the MIME type using the file extension first,
// then falls back to sniffing the first 512 bytes.
func detectMIMEType(filePath string, data []byte) string {
	// Try extension first (most reliable for known formats)
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != "" {
		if mimeType := mime.TypeByExtension(ext); mimeType != "" {
			// Strip parameters (e.g. "text/plain; charset=utf-8" → "text/plain")
			if idx := strings.Index(mimeType, ";"); idx > 0 {
				mimeType = strings.TrimSpace(mimeType[:idx])
			}
			return mimeType
		}
	}

	// Fallback: sniff content
	if len(data) > 0 {
		sniff := data
		if len(sniff) > 512 {
			sniff = sniff[:512]
		}
		return http.DetectContentType(sniff)
	}

	return "application/octet-stream"
}

// mimeToMsgType maps a MIME type to the appropriate Matrix message type.
func mimeToMsgType(mimeType string) event.MessageType {
	base := mimeType
	if idx := strings.Index(mimeType, "/"); idx > 0 {
		base = mimeType[:idx]
	}
	switch base {
	case "image":
		return event.MsgImage
	case "audio":
		return event.MsgAudio
	case "video":
		return event.MsgVideo
	default:
		return event.MsgFile
	}
}

// ─── Room/user helpers ────────────────────────────────────────────────────────

func (c *MatrixChannel) getRoomName(ctx context.Context, roomID id.RoomID) string {
	// Check cache first
	if cached, ok := c.roomNames.Load(roomID.String()); ok {
		return cached.(string)
	}

	// Fetch room name from state event
	var nameEvt event.RoomNameEventContent
	err := c.client.StateEvent(ctx, roomID, event.StateRoomName, "", &nameEvt)
	if err == nil && nameEvt.Name != "" {
		c.roomNames.Store(roomID.String(), nameEvt.Name)
		return nameEvt.Name
	}

	// Fallback to room ID
	roomName := roomID.String()
	c.roomNames.Store(roomID.String(), roomName)
	return roomName
}

func (c *MatrixChannel) getUserDisplayName(ctx context.Context, roomID id.RoomID, userID id.UserID) string {
	resp, err := c.client.GetDisplayName(ctx, userID)
	if err == nil && resp.DisplayName != "" {
		return resp.DisplayName
	}
	return userID.String()
}

func (c *MatrixChannel) getRoomMemberCount(ctx context.Context, roomID id.RoomID) int {
	resp, err := c.client.JoinedMembers(ctx, roomID)
	if err != nil {
		return 0
	}
	return len(resp.Joined)
}

func (c *MatrixChannel) getReplyToID(msgEvt *event.MessageEventContent) string {
	if msgEvt.RelatesTo != nil && msgEvt.RelatesTo.InReplyTo != nil {
		return msgEvt.RelatesTo.InReplyTo.EventID.String()
	}
	return ""
}

func (c *MatrixChannel) isBotMentioned(msgEvt *event.MessageEventContent, botUserID id.UserID) bool {
	// Full Matrix ID mention (e.g. @bot:homeserver)
	if strings.Contains(msgEvt.Body, botUserID.String()) {
		return true
	}

	// Formatted (HTML) body mention
	if msgEvt.Format == event.FormatHTML && strings.Contains(msgEvt.FormattedBody, botUserID.String()) {
		return true
	}

	// Localpart mention (e.g. "wanda")
	localpart := strings.TrimPrefix(botUserID.String(), "@")
	localpart = strings.Split(localpart, ":")[0]
	if strings.Contains(strings.ToLower(msgEvt.Body), strings.ToLower(localpart)) {
		return true
	}

	return false
}

func (c *MatrixChannel) removeMention(text string, botUserID id.UserID) string {
	// Remove full ID (@user:homeserver)
	text = strings.ReplaceAll(text, botUserID.String(), "")

	// Remove localpart with @ prefix
	localpart := strings.TrimPrefix(botUserID.String(), "@")
	localpart = strings.Split(localpart, ":")[0]
	text = strings.ReplaceAll(text, "@"+localpart, "")

	// Remove bare localpart at start/end of message
	text = strings.TrimPrefix(text, localpart)
	text = strings.TrimSuffix(text, localpart)

	return strings.TrimSpace(text)
}

// ─── Inbound media download ───────────────────────────────────────────────────

func (c *MatrixChannel) downloadMedia(ctx context.Context, mxcURL id.ContentURIString, filename, ext string) string {
	if mxcURL == "" {
		return ""
	}

	contentURI := mxcURL.ParseOrIgnore()
	if contentURI.IsEmpty() {
		logger.ErrorCF("matrix", "Invalid media URL", map[string]interface{}{
			"mxc_url": string(mxcURL),
		})
		return ""
	}

	logger.DebugCF("matrix", "Downloading media", map[string]interface{}{
		"mxc_url":  string(mxcURL),
		"filename": filename,
	})

	data, err := c.client.DownloadBytes(ctx, contentURI)
	if err != nil {
		logger.ErrorCF("matrix", "Failed to download media", map[string]interface{}{
			"error":   err.Error(),
			"mxc_url": string(mxcURL),
		})
		return ""
	}

	// Determine file extension
	if ext == "" {
		if strings.Contains(filename, ".") {
			parts := strings.Split(filename, ".")
			ext = "." + parts[len(parts)-1]
		} else {
			ext = ".bin"
		}
	}

	// Write to temp file
	tempFile, err := os.CreateTemp("", "matrix-media-*"+ext)
	if err != nil {
		logger.ErrorCF("matrix", "Failed to create temp file", map[string]interface{}{
			"error": err.Error(),
		})
		return ""
	}
	defer tempFile.Close()

	if _, err := tempFile.Write(data); err != nil {
		logger.ErrorCF("matrix", "Failed to write media file", map[string]interface{}{
			"error": err.Error(),
		})
		os.Remove(tempFile.Name())
		return ""
	}

	logger.InfoCF("matrix", "Media downloaded successfully", map[string]interface{}{
		"path": tempFile.Name(),
		"size": len(data),
	})

	return tempFile.Name()
}

// ─── Markdown → Matrix HTML ───────────────────────────────────────────────────

// hasMarkdown returns true if the text contains common Markdown syntax.
func hasMarkdown(text string) bool {
	return strings.ContainsAny(text, "*_`#[~")
}

// markdownToMatrixHTML converts a subset of Markdown to Matrix-compatible HTML.
// Matrix supports: <strong>, <em>, <code>, <pre>, <del>, <h1>-<h6>, <a>, <ul>, <li>, <blockquote>
func markdownToMatrixHTML(text string) string {
	if text == "" {
		return ""
	}

	// 1. Extract and protect code blocks (```...```) before other processing
	type codeBlock struct{ lang, code string }
	var codeBlocks []codeBlock
	reCodeBlock := regexp.MustCompile("(?s)```([a-zA-Z0-9]*)\n?(.*?)```")
	text = reCodeBlock.ReplaceAllStringFunc(text, func(m string) string {
		match := reCodeBlock.FindStringSubmatch(m)
		lang, code := "", ""
		if len(match) >= 3 {
			lang = match[1]
			code = match[2]
		}
		placeholder := fmt.Sprintf("\x00CB%d\x00", len(codeBlocks))
		codeBlocks = append(codeBlocks, codeBlock{lang, code})
		return placeholder
	})

	// 2. Extract and protect inline code (`...`)
	var inlineCodes []string
	reInlineCode := regexp.MustCompile("`([^`]+)`")
	text = reInlineCode.ReplaceAllStringFunc(text, func(m string) string {
		match := reInlineCode.FindStringSubmatch(m)
		code := ""
		if len(match) >= 2 {
			code = match[1]
		}
		placeholder := fmt.Sprintf("\x00IC%d\x00", len(inlineCodes))
		inlineCodes = append(inlineCodes, code)
		return placeholder
	})

	// 3. Escape HTML special characters in non-code content
	text = matrixEscapeHTML(text)

	// 4. Bold: **text** or __text__
	reBold := regexp.MustCompile(`\*\*(.+?)\*\*`)
	text = reBold.ReplaceAllString(text, "<strong>$1</strong>")
	reBold2 := regexp.MustCompile(`__(.+?)__`)
	text = reBold2.ReplaceAllString(text, "<strong>$1</strong>")

	// 5. Italic: *text* or _text_ (single, not double)
	reItalic := regexp.MustCompile(`\*([^*\n]+)\*`)
	text = reItalic.ReplaceAllString(text, "<em>$1</em>")
	reItalic2 := regexp.MustCompile(`_([^_\n]+)_`)
	text = reItalic2.ReplaceAllString(text, "<em>$1</em>")

	// 6. Strikethrough: ~~text~~
	reStrike := regexp.MustCompile(`~~(.+?)~~`)
	text = reStrike.ReplaceAllString(text, "<del>$1</del>")

	// 7. Links: [label](url)
	reLink := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	text = reLink.ReplaceAllString(text, `<a href="$2">$1</a>`)

	// 8. Headings: # H1, ## H2, etc. (line by line)
	reHeading := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	text = reHeading.ReplaceAllStringFunc(text, func(m string) string {
		match := reHeading.FindStringSubmatch(m)
		if len(match) < 3 {
			return m
		}
		level := len(match[1])
		return fmt.Sprintf("<h%d>%s</h%d>", level, match[2], level)
	})

	// 9. Blockquotes: > text
	reQuote := regexp.MustCompile(`(?m)^>\s?(.*)$`)
	text = reQuote.ReplaceAllString(text, "<blockquote>$1</blockquote>")

	// 10. Unordered list items: - item or * item
	reList := regexp.MustCompile(`(?m)^[-*]\s+(.+)$`)
	text = reList.ReplaceAllString(text, "<li>$1</li>")

	// 11. Newlines → <br> (preserve formatting)
	text = strings.ReplaceAll(text, "\n", "<br>\n")

	// 12. Restore inline code
	for i, code := range inlineCodes {
		escaped := matrixEscapeHTML(code)
		text = strings.ReplaceAll(text, fmt.Sprintf("\x00IC%d\x00", i), fmt.Sprintf("<code>%s</code>", escaped))
	}

	// 13. Restore code blocks
	for i, cb := range codeBlocks {
		escaped := matrixEscapeHTML(cb.code)
		if cb.lang != "" {
			text = strings.ReplaceAll(text, fmt.Sprintf("\x00CB%d\x00", i),
				fmt.Sprintf("<pre><code class=\"language-%s\">%s</code></pre>", cb.lang, escaped))
		} else {
			text = strings.ReplaceAll(text, fmt.Sprintf("\x00CB%d\x00", i),
				fmt.Sprintf("<pre><code>%s</code></pre>", escaped))
		}
	}

	return text
}

// matrixEscapeHTML escapes the three HTML special characters.
func matrixEscapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
