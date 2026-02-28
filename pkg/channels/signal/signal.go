package signal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	signalMaxMessageLength  = 6000
	signalSSEReconnectDelay = 5 * time.Second
	signalRPCTimeout        = 30 * time.Second
	signalTypingInterval    = 8 * time.Second
	signalTypingTimeout     = 5 * time.Minute
	signalDefaultCLIURL     = "http://localhost:8080"
)

// SignalChannel implements the Channel interface for Signal via signal-cli daemon.
// It connects to signal-cli's HTTP API: SSE for receiving events, JSON-RPC for sending.
//
// Implements: channels.Channel, channels.TypingCapable, channels.ReactionCapable
type SignalChannel struct {
	*channels.BaseChannel
	config     config.SignalConfig
	httpClient *http.Client
	ctx        context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// Signal SSE event types

type signalEvent struct {
	Envelope signalEnvelope `json:"envelope"`
	Account  string         `json:"account"`
}

type signalEnvelope struct {
	Source       string             `json:"source"`
	SourceNumber string             `json:"sourceNumber"`
	SourceUUID   string             `json:"sourceUuid"`
	SourceName   string             `json:"sourceName"`
	SourceDevice int                `json:"sourceDevice"`
	Timestamp    int64              `json:"timestamp"`
	DataMessage  *signalDataMessage `json:"dataMessage"`
}

type signalDataMessage struct {
	Timestamp        int64              `json:"timestamp"`
	Message          string             `json:"message"`
	ExpiresInSeconds int                `json:"expiresInSeconds"`
	ViewOnce         bool               `json:"viewOnce"`
	GroupInfo        *signalGroupInfo   `json:"groupInfo"`
	Attachments      []signalAttachment `json:"attachments"`
	Mentions         []signalMention    `json:"mentions"`
}

type signalMention struct {
	Start  int    `json:"start"`
	Length int    `json:"length"`
	UUID   string `json:"uuid"`
	Number string `json:"number"`
}

type signalGroupInfo struct {
	GroupID string `json:"groupId"`
	Type    string `json:"type"`
}

type signalAttachment struct {
	ContentType string `json:"contentType"`
	Filename    string `json:"filename"`
	ID          string `json:"id"`
	Size        int64  `json:"size"`
}

// JSON-RPC types

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	ID      int    `json:"id"`
	Params  any    `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonRPCError   `json:"error"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func NewSignalChannel(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
	signalCfg := cfg.Channels.Signal

	if signalCfg.SignalCLIURL == "" {
		signalCfg.SignalCLIURL = signalDefaultCLIURL
	}

	opts := []channels.BaseChannelOption{
		channels.WithMaxMessageLength(signalMaxMessageLength),
		channels.WithGroupTrigger(signalCfg.GroupTrigger),
	}
	if signalCfg.ReasoningChannelID != "" {
		opts = append(opts, channels.WithReasoningChannelID(signalCfg.ReasoningChannelID))
	}

	base := channels.NewBaseChannel("signal", signalCfg, b, signalCfg.AllowFrom, opts...)

	return &SignalChannel{
		BaseChannel: base,
		config:      signalCfg,
		httpClient:  &http.Client{Timeout: signalRPCTimeout},
	}, nil
}

func (c *SignalChannel) Start(ctx context.Context) error {
	logger.InfoCF("signal", "Starting Signal channel", map[string]any{
		"signal_cli_url": c.config.SignalCLIURL,
		"account":        c.config.Account,
	})

	c.ctx, c.cancel = context.WithCancel(ctx)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.sseLoop()
	}()

	c.SetRunning(true)
	logger.InfoC("signal", "Signal channel started")
	return nil
}

func (c *SignalChannel) Stop(ctx context.Context) error {
	logger.InfoC("signal", "Stopping Signal channel")

	if c.cancel != nil {
		c.cancel()
	}

	// Wait for goroutines with context deadline
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		logger.WarnC("signal", fmt.Sprintf("Stop context canceled before goroutines finished: %v", ctx.Err()))
	}

	c.SetRunning(false)
	return nil
}

func (c *SignalChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := c.sendMessage(ctx, msg.ChatID, msg.Content); err != nil {
		return fmt.Errorf("signal send: %w: %v", channels.ErrTemporary, err)
	}

	return nil
}

// StartTyping implements channels.TypingCapable.
// It sends a typing indicator immediately and then repeats every 8 seconds
// (signal-cli's typing indicator expires after ~10s) in a background goroutine.
// The returned stop function is idempotent and cancels the goroutine.
func (c *SignalChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	c.sendTyping(chatID)

	typingCtx, cancel := context.WithCancel(ctx)
	var once sync.Once
	stop := func() {
		once.Do(func() {
			cancel()
		})
	}

	go func() {
		ticker := time.NewTicker(signalTypingInterval)
		defer ticker.Stop()
		timeout := time.After(signalTypingTimeout)
		for {
			select {
			case <-typingCtx.Done():
				return
			case <-timeout:
				return
			case <-ticker.C:
				c.sendTyping(chatID)
			}
		}
	}()

	return stop, nil
}

// ReactToMessage implements channels.ReactionCapable.
// It sends a 👀 emoji reaction to the inbound message and returns an undo
// function that removes the reaction. The Manager auto-calls this on inbound
// and undoes it before sending the bot's response.
func (c *SignalChannel) ReactToMessage(ctx context.Context, chatID, messageID string) (func(), error) {
	// messageID is encoded as "timestamp:senderPhone" by handleEvent
	ts, senderPhone, ok := parseMessageID(messageID)
	if !ok {
		return func() {}, nil // non-critical, skip silently
	}

	c.sendReaction(ctx, chatID, senderPhone, ts, "👀", false)

	var once sync.Once
	undo := func() {
		once.Do(func() {
			undoCtx, undoCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer undoCancel()
			c.sendReaction(undoCtx, chatID, senderPhone, ts, "👀", true)
		})
	}

	return undo, nil
}

// SSE event loop with automatic reconnection

func (c *SignalChannel) sseLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if err := c.connectSSE(); err != nil {
				logger.ErrorCF("signal", "SSE connection error", map[string]any{
					"error": err.Error(),
				})
			}

			select {
			case <-c.ctx.Done():
				return
			case <-time.After(signalSSEReconnectDelay):
				logger.InfoC("signal", "Reconnecting SSE...")
			}
		}
	}
}

func (c *SignalChannel) connectSSE() error {
	url := fmt.Sprintf("%s/api/v1/events", c.config.SignalCLIURL)

	req, err := http.NewRequestWithContext(c.ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	sseClient := &http.Client{Timeout: 0}
	resp, err := sseClient.Do(req)
	if err != nil {
		return fmt.Errorf("SSE connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("SSE returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.InfoC("signal", "SSE connected successfully")

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-c.ctx.Done():
			return nil
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimSpace(data)
		if data == "" {
			continue
		}

		var event signalEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			logger.DebugCF("signal", "Failed to parse SSE event", map[string]any{
				"error": err.Error(),
				"data":  utils.Truncate(data, 100),
			})
			continue
		}

		c.handleEvent(event)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("SSE stream error: %w", err)
	}

	return fmt.Errorf("SSE stream ended")
}

// Event handling

func (c *SignalChannel) handleEvent(event signalEvent) {
	envelope := event.Envelope

	if envelope.DataMessage == nil {
		return
	}

	dm := envelope.DataMessage

	senderPhone := envelope.SourceNumber
	if senderPhone == "" {
		senderPhone = envelope.Source
	}
	if senderPhone == "" {
		return
	}

	// Build structured sender info for the new identity system
	sender := bus.SenderInfo{
		Platform:    "signal",
		PlatformID:  senderPhone,
		CanonicalID: identity.BuildCanonicalID("signal", senderPhone),
		DisplayName: envelope.SourceName,
	}

	if !c.IsAllowedSender(sender) {
		logger.DebugCF("signal", "Message rejected by allowlist", map[string]any{
			"sender": senderPhone,
		})
		return
	}

	isGroup := dm.GroupInfo != nil
	chatID := senderPhone
	peerKind := "direct"
	peerID := senderPhone

	if isGroup {
		chatID = dm.GroupInfo.GroupID
		peerKind = "group"
		peerID = dm.GroupInfo.GroupID
	}

	content := dm.Message

	// In group chats, apply unified group trigger filtering
	if isGroup {
		isMentioned := c.isBotMentioned(dm.Mentions)
		if isMentioned {
			content = c.stripMention(content, dm.Mentions)
		}
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, content)
		if !respond {
			return
		}
		content = cleaned
	}
	mediaPaths := []string{}
	localFiles := []string{}

	defer func() {
		for _, file := range localFiles {
			if err := os.Remove(file); err != nil {
				logger.DebugCF("signal", "Failed to cleanup temp file", map[string]any{
					"file":  file,
					"error": err.Error(),
				})
			}
		}
	}()

	for _, att := range dm.Attachments {
		localPath := c.downloadAttachment(att)
		if localPath == "" {
			continue
		}
		localFiles = append(localFiles, localPath)
		mediaPaths = append(mediaPaths, localPath)

		if strings.HasPrefix(att.ContentType, "image/") {
			content = appendContent(content, "[image: photo]")
		} else if utils.IsAudioFile(att.Filename, att.ContentType) {
			content = appendContent(content, "[voice message]")
		} else {
			name := att.Filename
			if name == "" {
				name = att.ContentType
			}
			content = appendContent(content, fmt.Sprintf("[file: %s]", name))
		}
	}

	if content == "" && len(mediaPaths) == 0 {
		return
	}
	if content == "" {
		content = "[media only]"
	}

	peer := bus.Peer{Kind: peerKind, ID: peerID}

	// Encode messageID as "timestamp:senderPhone" so ReactToMessage can extract both
	messageID := fmt.Sprintf("%d:%s", dm.Timestamp, senderPhone)

	metadata := map[string]string{
		"timestamp":   fmt.Sprintf("%d", dm.Timestamp),
		"source_uuid": envelope.SourceUUID,
		"source_name": envelope.SourceName,
		"phone":       senderPhone,
		"is_group":    fmt.Sprintf("%t", isGroup),
		"peer_kind":   peerKind,
		"peer_id":     peerID,
		"message_id":  messageID,
	}
	if isGroup {
		metadata["group_id"] = dm.GroupInfo.GroupID
	}

	logger.DebugCF("signal", "Received message", map[string]any{
		"sender":   senderPhone,
		"chat_id":  chatID,
		"is_group": isGroup,
		"preview":  utils.Truncate(content, 50),
	})

	c.HandleMessage(c.ctx, peer, messageID, senderPhone, chatID, content, mediaPaths, metadata, sender)
}

// isBotMentioned checks whether the bot was @mentioned in a group message
// by looking for its account number or UUID in the structured mentions array.
//
// Note: signal-cli v0.13.24 has a bug (https://github.com/AsamK/signal-cli/issues/1940)
// where the mentions array is empty due to binary ACI parsing issues. This will
// work correctly once the fix (PR #1944) is released.
func (c *SignalChannel) isBotMentioned(mentions []signalMention) bool {
	for _, m := range mentions {
		if m.Number == c.config.Account || m.UUID == c.config.Account {
			return true
		}
	}
	return false
}

// stripMention removes the bot's @mention from the message content using
// the precise UTF-16 offsets from the structured mention data.
// Signal represents mentions as U+FFFC (object replacement character) in the text.
func (c *SignalChannel) stripMention(content string, mentions []signalMention) string {
	for _, m := range mentions {
		if m.Number != c.config.Account && m.UUID != c.config.Account {
			continue
		}
		runes := []rune(content)
		runeStart, runeLen := utf16PosToRunePos(runes, m.Start, m.Length)
		if runeStart >= 0 && runeStart+runeLen <= len(runes) {
			before := strings.TrimRight(string(runes[:runeStart]), " ")
			after := strings.TrimLeft(string(runes[runeStart+runeLen:]), " ")
			if before == "" {
				return after
			}
			if after == "" {
				return before
			}
			return before + " " + after
		}
	}
	return content
}

// utf16PosToRunePos converts a UTF-16 code unit position and length to rune position and length.
func utf16PosToRunePos(runes []rune, utf16Start, utf16Len int) (int, int) {
	pos := 0
	runeStart := -1
	runeLen := 0
	for i, r := range runes {
		if pos == utf16Start {
			runeStart = i
		}
		units := 1
		if r >= 0x10000 {
			units = 2 // surrogate pair
		}
		if runeStart >= 0 {
			runeLen++
			if pos+units >= utf16Start+utf16Len {
				break
			}
		}
		pos += units
	}
	return runeStart, runeLen
}

// Media handling

func (c *SignalChannel) downloadAttachment(att signalAttachment) string {
	if att.ID == "" {
		return ""
	}

	url := fmt.Sprintf("%s/api/v1/attachments/%s", c.config.SignalCLIURL, att.ID)
	filename := att.Filename
	if filename == "" {
		filename = "attachment" + extensionFromMIME(att.ContentType)
	}

	return utils.DownloadFile(url, filename, utils.DownloadOptions{
		LoggerPrefix: "signal",
	})
}

func extensionFromMIME(mime string) string {
	switch {
	case strings.HasPrefix(mime, "image/jpeg"):
		return ".jpg"
	case strings.HasPrefix(mime, "image/png"):
		return ".png"
	case strings.HasPrefix(mime, "image/gif"):
		return ".gif"
	case strings.HasPrefix(mime, "image/webp"):
		return ".webp"
	case strings.HasPrefix(mime, "audio/mpeg"), strings.HasPrefix(mime, "audio/mp3"):
		return ".mp3"
	case strings.HasPrefix(mime, "audio/ogg"):
		return ".ogg"
	case strings.HasPrefix(mime, "audio/mp4"), strings.HasPrefix(mime, "audio/aac"):
		return ".m4a"
	case strings.HasPrefix(mime, "video/mp4"):
		return ".mp4"
	default:
		return ""
	}
}

func appendContent(content, suffix string) string {
	if content == "" {
		return suffix
	}
	return content + "\n" + suffix
}

// Sending messages via JSON-RPC

func (c *SignalChannel) sendMessage(ctx context.Context, chatID, content string) error {
	plainText, textStyles := markdownToSignal(content)

	params := map[string]any{
		"account": c.config.Account,
		"message": plainText,
	}

	if len(textStyles) > 0 {
		params["textStyle"] = textStyles
	}

	if isGroupChat(chatID) {
		params["groupId"] = chatID
	} else {
		params["recipient"] = []string{chatID}
	}

	_, err := c.rpcCall(ctx, "send", params)
	return err
}

func (c *SignalChannel) sendReaction(ctx context.Context, chatID, targetAuthor string, targetTimestamp int64, emoji string, remove bool) {
	params := map[string]any{
		"account":         c.config.Account,
		"emoji":           emoji,
		"targetAuthor":    targetAuthor,
		"targetTimestamp": targetTimestamp,
		"remove":          remove,
	}
	if isGroupChat(chatID) {
		params["groupId"] = chatID
	} else {
		params["recipient"] = []string{chatID}
	}

	if _, err := c.rpcCall(ctx, "sendReaction", params); err != nil {
		logger.DebugCF("signal", "Failed to send reaction", map[string]any{
			"error":  err.Error(),
			"remove": remove,
		})
	}
}

func (c *SignalChannel) rpcCall(ctx context.Context, method string, params any) (*jsonRPCResponse, error) {
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		ID:      1,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RPC request: %w", err)
	}

	rpcURL := c.config.SignalCLIURL + "/api/v1/rpc"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", rpcURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("RPC request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read RPC response: %w", err)
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return &rpcResp, nil
}

// Typing indicator

func (c *SignalChannel) sendTyping(chatID string) {
	params := map[string]any{
		"account": c.config.Account,
	}

	if isGroupChat(chatID) {
		params["groupId"] = chatID
	} else {
		params["recipient"] = []string{chatID}
	}

	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	if _, err := c.rpcCall(ctx, "sendTyping", params); err != nil {
		logger.DebugCF("signal", "Failed to send typing indicator", map[string]any{
			"error":   err.Error(),
			"chat_id": chatID,
		})
	}
}

// isGroupChat determines if a chatID is a Signal group (base64-encoded) or a phone number.
// This is safe because chatID is always set by handleEvent: either the sender's E.164 phone
// number (starts with "+") for DMs, or the base64-encoded GroupInfo.GroupID for groups.
func isGroupChat(chatID string) bool {
	return chatID != "" && !strings.HasPrefix(chatID, "+")
}

// parseMessageID extracts timestamp and sender phone from the encoded messageID
// format "timestamp:senderPhone".
func parseMessageID(messageID string) (timestamp int64, senderPhone string, ok bool) {
	idx := strings.Index(messageID, ":")
	if idx <= 0 || idx == len(messageID)-1 {
		return 0, "", false
	}
	ts, err := strconv.ParseInt(messageID[:idx], 10, 64)
	if err != nil {
		return 0, "", false
	}
	return ts, messageID[idx+1:], true
}

// markdownToSignal converts markdown-formatted text to plain text with signal-cli
// textStyle ranges. Returns the converted text and a slice of style strings in
// "START:LENGTH:STYLE" format for signal-cli's textStyle parameter.
// Handles: **bold**, *italic*, ~~strikethrough~~, `code`, ```code blocks```,
// [links](url), heading stripping, list markers, blockquotes.
func markdownToSignal(text string) (string, []string) {
	if text == "" {
		return text, nil
	}

	// Step 0: extract code blocks and inline code into placeholders.
	// This prevents code content (e.g. *ptr inside code) from being
	// processed as markdown in later steps.
	var codeBlocks []string
	var inlineCodes []string

	reCodeBlock := regexp.MustCompile("(?s)```[\\w]*\\n?(.*?)```")
	text = reCodeBlock.ReplaceAllStringFunc(text, func(m string) string {
		inner := reCodeBlock.FindStringSubmatch(m)[1]
		idx := len(codeBlocks)
		codeBlocks = append(codeBlocks, inner)
		return fmt.Sprintf("\x00CB%d\x00", idx)
	})

	reInlineCode := regexp.MustCompile("`([^`]+)`")
	text = reInlineCode.ReplaceAllStringFunc(text, func(m string) string {
		inner := reInlineCode.FindStringSubmatch(m)[1]
		idx := len(inlineCodes)
		inlineCodes = append(inlineCodes, inner)
		return fmt.Sprintf("\x00IC%d\x00", idx)
	})

	// Step 1: line-level markdown (headings, lists, blockquotes)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			trimmed := strings.TrimLeft(line, "#")
			lines[i] = strings.TrimLeft(trimmed, " ")
		} else if strings.HasPrefix(line, "- ") {
			lines[i] = "• " + line[2:]
		} else if strings.HasPrefix(line, "* ") {
			lines[i] = "• " + line[2:]
		} else if strings.HasPrefix(line, "> ") {
			lines[i] = line[2:]
		}
	}
	text = strings.Join(lines, "\n")

	// Step 1b: convert markdown links [text](url) → text (url)
	reLink := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	text = reLink.ReplaceAllString(text, "$1 ($2)")

	// Step 2: inline styles → textStyle position ranges
	type styleEntry struct {
		start  int
		length int
		style  string
	}

	var styles []styleEntry
	var result []rune
	runes := []rune(text)
	i := 0

	utf16Pos := func() int {
		return len(utf16.Encode(result))
	}

	// Check if current position is the start of a placeholder (\x00CB0\x00 or \x00IC0\x00)
	matchPlaceholder := func(pos int) (kind string, idx int, end int, ok bool) {
		if pos+4 >= len(runes) || runes[pos] != 0 {
			return "", 0, 0, false
		}
		// Find the closing \x00
		j := pos + 1
		for j < len(runes) && runes[j] != 0 {
			j++
		}
		if j >= len(runes) {
			return "", 0, 0, false
		}
		tag := string(runes[pos+1 : j])
		if strings.HasPrefix(tag, "CB") {
			n := 0
			if _, err := fmt.Sscanf(tag, "CB%d", &n); err == nil {
				return "CB", n, j + 1, true
			}
		} else if strings.HasPrefix(tag, "IC") {
			n := 0
			if _, err := fmt.Sscanf(tag, "IC%d", &n); err == nil {
				return "IC", n, j + 1, true
			}
		}
		return "", 0, 0, false
	}

	for i < len(runes) {
		// Code placeholders → restore content with MONOSPACE style
		if kind, idx, end, ok := matchPlaceholder(i); ok {
			var code string
			if kind == "CB" && idx < len(codeBlocks) {
				code = strings.TrimRight(codeBlocks[idx], "\n")
			} else if kind == "IC" && idx < len(inlineCodes) {
				code = inlineCodes[idx]
			}
			if code != "" {
				codeRunes := []rune(code)
				start := utf16Pos()
				styles = append(styles, styleEntry{start, len(utf16.Encode(codeRunes)), "MONOSPACE"})
				result = append(result, codeRunes...)
			}
			i = end
			continue
		}

		// Strikethrough: ~~text~~
		if i+1 < len(runes) && runes[i] == '~' && runes[i+1] == '~' {
			if end := signalFindDouble(runes, i+2, '~'); end > i+2 {
				inner := runes[i+2 : end]
				start := utf16Pos()
				styles = append(styles, styleEntry{start, len(utf16.Encode(inner)), "STRIKETHROUGH"})
				result = append(result, inner...)
				i = end + 2
				continue
			}
		}

		// Bold: **text**
		if i+1 < len(runes) && runes[i] == '*' && runes[i+1] == '*' {
			if end := signalFindDouble(runes, i+2, '*'); end > i+2 {
				inner := runes[i+2 : end]
				start := utf16Pos()
				styles = append(styles, styleEntry{start, len(utf16.Encode(inner)), "BOLD"})
				result = append(result, inner...)
				i = end + 2
				continue
			}
		}

		// Italic: *text* (single *, not followed by another *)
		if runes[i] == '*' && (i+1 < len(runes) && runes[i+1] != '*') {
			if end := signalFindSingle(runes, i+1, '*'); end > i+1 {
				inner := runes[i+1 : end]
				start := utf16Pos()
				styles = append(styles, styleEntry{start, len(utf16.Encode(inner)), "ITALIC"})
				result = append(result, inner...)
				i = end + 1
				continue
			}
		}

		result = append(result, runes[i])
		i++
	}

	if len(styles) == 0 {
		return string(result), nil
	}

	strs := make([]string, len(styles))
	for idx, s := range styles {
		strs[idx] = fmt.Sprintf("%d:%d:%s", s.start, s.length, s.style)
	}
	return string(result), strs
}

// signalFindDouble finds the next occurrence of two consecutive ch runes starting from pos.
func signalFindDouble(runes []rune, start int, ch rune) int {
	for i := start; i+1 < len(runes); i++ {
		if runes[i] == ch && runes[i+1] == ch {
			return i
		}
	}
	return -1
}

// signalFindSingle finds the next occurrence of ch that is NOT followed by another ch.
func signalFindSingle(runes []rune, start int, ch rune) int {
	for i := start; i < len(runes); i++ {
		if runes[i] == ch && (i+1 >= len(runes) || runes[i+1] != ch) {
			return i
		}
	}
	return -1
}
