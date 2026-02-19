package channels

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	th "github.com/mymmrac/telego/telegohandler"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/utils"
	"github.com/sipeed/picoclaw/pkg/voice"
)

type TelegramChannel struct {
	*BaseChannel
	bot          *telego.Bot
	commands     TelegramCommander
	config       *config.Config
	chatIDs      map[string]int64
	transcriber  *voice.GroqTranscriber
	placeholders sync.Map // chatID -> messageID
	stopThinking sync.Map // chatID -> thinkingCancel
}

type thinkingCancel struct {
	fn context.CancelFunc
}

func (c *thinkingCancel) Cancel() {
	if c != nil && c.fn != nil {
		c.fn()
	}
}

const telegramMaxMessageChars = 3900
const markdownTableMaxWidth = 90
const markdownTableMinColWidth = 6

var thinkBlockPattern = regexp.MustCompile(`(?is)<think>.*?</think>`)

func NewTelegramChannel(cfg *config.Config, bus *bus.MessageBus) (*TelegramChannel, error) {
	var opts []telego.BotOption
	telegramCfg := cfg.Channels.Telegram

	if telegramCfg.Proxy != "" {
		proxyURL, parseErr := url.Parse(telegramCfg.Proxy)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", telegramCfg.Proxy, parseErr)
		}
		opts = append(opts, telego.WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}))
	}

	bot, err := telego.NewBot(telegramCfg.Token, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	base := NewBaseChannel("telegram", telegramCfg, bus, telegramCfg.AllowFrom)

	return &TelegramChannel{
		BaseChannel:  base,
		commands:     NewTelegramCommands(bot, cfg),
		bot:          bot,
		config:       cfg,
		chatIDs:      make(map[string]int64),
		transcriber:  nil,
		placeholders: sync.Map{},
		stopThinking: sync.Map{},
	}, nil
}

func (c *TelegramChannel) SetTranscriber(transcriber *voice.GroqTranscriber) {
	c.transcriber = transcriber
}

func (c *TelegramChannel) Start(ctx context.Context) error {
	logger.InfoC("telegram", "Starting Telegram bot (polling mode)...")

	updates, err := c.bot.UpdatesViaLongPolling(ctx, &telego.GetUpdatesParams{
		Timeout: 30,
	})
	if err != nil {
		return fmt.Errorf("failed to start long polling: %w", err)
	}

	bh, err := telegohandler.NewBotHandler(c.bot, updates)
	if err != nil {
		return fmt.Errorf("failed to create bot handler: %w", err)
	}

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		c.commands.Help(ctx, message)
		return nil
	}, th.CommandEqual("help"))
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		return c.commands.Start(ctx, message)
	}, th.CommandEqual("start"))

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		return c.commands.Show(ctx, message)
	}, th.CommandEqual("show"))

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		return c.commands.List(ctx, message)
	}, th.CommandEqual("list"))

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		return c.handleMessage(ctx, &message)
	}, th.AnyMessage())

	c.setRunning(true)
	logger.InfoCF("telegram", "Telegram bot connected", map[string]interface{}{
		"username": c.bot.Username(),
	})

	go bh.Start()

	go func() {
		<-ctx.Done()
		bh.Stop()
	}()

	return nil
}
func (c *TelegramChannel) Stop(ctx context.Context) error {
	logger.InfoC("telegram", "Stopping Telegram bot...")
	c.setRunning(false)
	return nil
}

func (c *TelegramChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("telegram bot not running")
	}

	chatID, err := parseChatID(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	// Stop thinking animation
	if stop, ok := c.stopThinking.Load(msg.ChatID); ok {
		if cf, ok := stop.(*thinkingCancel); ok && cf != nil {
			cf.Cancel()
		}
		c.stopThinking.Delete(msg.ChatID)
	}

	cleanContent := sanitizeTelegramOutgoingContent(msg.Content)
	chunks := utils.SplitMessage(cleanContent, telegramMaxMessageChars)
	if len(chunks) == 0 {
		chunks = []string{cleanContent}
	}

	// Try to edit placeholder
	firstChunkSent := false
	if pID, ok := c.placeholders.Load(msg.ChatID); ok {
		c.placeholders.Delete(msg.ChatID)
		editMsg := tu.EditMessageText(tu.ID(chatID), pID.(int), markdownToTelegramHTML(chunks[0]))
		editMsg.ParseMode = telego.ModeHTML

		if _, err = c.bot.EditMessageText(ctx, editMsg); err == nil {
			firstChunkSent = true
		}
		// Fallback to new message if edit fails
	}

	sendChunk := func(text string) error {
		tgMsg := tu.Message(tu.ID(chatID), markdownToTelegramHTML(text))
		tgMsg.ParseMode = telego.ModeHTML
		if _, sendErr := c.bot.SendMessage(ctx, tgMsg); sendErr != nil {
			logger.ErrorCF("telegram", "HTML parse failed, falling back to plain text", map[string]interface{}{
				"error": sendErr.Error(),
			})
			fallbackMsg := tu.Message(tu.ID(chatID), text)
			fallbackMsg.ParseMode = ""
			_, sendErr = c.bot.SendMessage(ctx, fallbackMsg)
			return sendErr
		}
		return nil
	}

	startIdx := 0
	if firstChunkSent {
		startIdx = 1
	}
	for i := startIdx; i < len(chunks); i++ {
		if err = sendChunk(chunks[i]); err != nil {
			return err
		}
	}

	return nil
}

func sanitizeTelegramOutgoingContent(content string) string {
	cleaned := thinkBlockPattern.ReplaceAllString(content, "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "(empty response)"
	}
	return cleaned
}

func (c *TelegramChannel) handleMessage(ctx context.Context, message *telego.Message) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}

	user := message.From
	if user == nil {
		return fmt.Errorf("message sender (user) is nil")
	}

	senderID := fmt.Sprintf("%d", user.ID)
	if user.Username != "" {
		senderID = fmt.Sprintf("%d|%s", user.ID, user.Username)
	}

	// 检查白名单，避免为被拒绝的用户下载附件
	if !c.IsAllowed(senderID) {
		logger.DebugCF("telegram", "Message rejected by allowlist", map[string]interface{}{
			"user_id": senderID,
		})
		return nil
	}

	chatID := message.Chat.ID
	c.chatIDs[senderID] = chatID

	content := ""
	mediaPaths := []string{}
	localFiles := []string{} // 跟踪需要清理的本地文件

	// 确保临时文件在函数返回时被清理
	defer func() {
		for _, file := range localFiles {
			if err := os.Remove(file); err != nil {
				logger.DebugCF("telegram", "Failed to cleanup temp file", map[string]interface{}{
					"file":  file,
					"error": err.Error(),
				})
			}
		}
	}()

	if message.Text != "" {
		content += message.Text
	}

	if message.Caption != "" {
		if content != "" {
			content += "\n"
		}
		content += message.Caption
	}

	if len(message.Photo) > 0 {
		photo := message.Photo[len(message.Photo)-1]
		photoPath := c.downloadPhoto(ctx, photo.FileID)
		if photoPath != "" {
			localFiles = append(localFiles, photoPath)
			mediaPaths = append(mediaPaths, photoPath)
			if content != "" {
				content += "\n"
			}
			content += "[image: photo]"
		}
	}

	if message.Voice != nil {
		voicePath := c.downloadFile(ctx, message.Voice.FileID, ".ogg")
		if voicePath != "" {
			localFiles = append(localFiles, voicePath)
			mediaPaths = append(mediaPaths, voicePath)

			transcribedText := ""
			if c.transcriber != nil && c.transcriber.IsAvailable() {
				ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()

				result, err := c.transcriber.Transcribe(ctx, voicePath)
				if err != nil {
					logger.ErrorCF("telegram", "Voice transcription failed", map[string]interface{}{
						"error": err.Error(),
						"path":  voicePath,
					})
					transcribedText = "[voice (transcription failed)]"
				} else {
					transcribedText = fmt.Sprintf("[voice transcription: %s]", result.Text)
					logger.InfoCF("telegram", "Voice transcribed successfully", map[string]interface{}{
						"text": result.Text,
					})
				}
			} else {
				transcribedText = "[voice]"
			}

			if content != "" {
				content += "\n"
			}
			content += transcribedText
		}
	}

	if message.Audio != nil {
		audioPath := c.downloadFile(ctx, message.Audio.FileID, ".mp3")
		if audioPath != "" {
			localFiles = append(localFiles, audioPath)
			mediaPaths = append(mediaPaths, audioPath)
			if content != "" {
				content += "\n"
			}
			content += "[audio]"
		}
	}

	if message.Document != nil {
		docPath := c.downloadFile(ctx, message.Document.FileID, "")
		if docPath != "" {
			localFiles = append(localFiles, docPath)
			mediaPaths = append(mediaPaths, docPath)
			if content != "" {
				content += "\n"
			}
			content += "[file]"
		}
	}

	if content == "" {
		content = "[empty message]"
	}

	logger.DebugCF("telegram", "Received message", map[string]interface{}{
		"sender_id": senderID,
		"chat_id":   fmt.Sprintf("%d", chatID),
		"preview":   utils.Truncate(content, 50),
	})

	// Thinking indicator
	err := c.bot.SendChatAction(ctx, tu.ChatAction(tu.ID(chatID), telego.ChatActionTyping))
	if err != nil {
		logger.ErrorCF("telegram", "Failed to send chat action", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Stop any previous thinking animation
	chatIDStr := fmt.Sprintf("%d", chatID)
	if prevStop, ok := c.stopThinking.Load(chatIDStr); ok {
		if cf, ok := prevStop.(*thinkingCancel); ok && cf != nil {
			cf.Cancel()
		}
	}

	// Create cancel function for thinking state
	_, thinkCancel := context.WithTimeout(ctx, 5*time.Minute)
	c.stopThinking.Store(chatIDStr, &thinkingCancel{fn: thinkCancel})

	pMsg, err := c.bot.SendMessage(ctx, tu.Message(tu.ID(chatID), "Thinking... 💭"))
	if err == nil {
		pID := pMsg.MessageID
		c.placeholders.Store(chatIDStr, pID)
	}

	peerKind := "direct"
	peerID := fmt.Sprintf("%d", user.ID)
	if message.Chat.Type != "private" {
		peerKind = "group"
		peerID = fmt.Sprintf("%d", chatID)
	}

	metadata := map[string]string{
		"message_id": fmt.Sprintf("%d", message.MessageID),
		"user_id":    fmt.Sprintf("%d", user.ID),
		"username":   user.Username,
		"first_name": user.FirstName,
		"is_group":   fmt.Sprintf("%t", message.Chat.Type != "private"),
		"peer_kind":  peerKind,
		"peer_id":    peerID,
	}

	c.HandleMessage(fmt.Sprintf("%d", user.ID), fmt.Sprintf("%d", chatID), content, mediaPaths, metadata)
	return nil
}

func (c *TelegramChannel) downloadPhoto(ctx context.Context, fileID string) string {
	file, err := c.bot.GetFile(ctx, &telego.GetFileParams{FileID: fileID})
	if err != nil {
		logger.ErrorCF("telegram", "Failed to get photo file", map[string]interface{}{
			"error": err.Error(),
		})
		return ""
	}

	return c.downloadFileWithInfo(file, ".jpg")
}

func (c *TelegramChannel) downloadFileWithInfo(file *telego.File, ext string) string {
	if file.FilePath == "" {
		return ""
	}

	url := c.bot.FileDownloadURL(file.FilePath)
	logger.DebugCF("telegram", "File URL", map[string]interface{}{"url": url})

	// Use FilePath as filename for better identification
	filename := file.FilePath + ext
	return utils.DownloadFile(url, filename, utils.DownloadOptions{
		LoggerPrefix: "telegram",
	})
}

func (c *TelegramChannel) downloadFile(ctx context.Context, fileID, ext string) string {
	file, err := c.bot.GetFile(ctx, &telego.GetFileParams{FileID: fileID})
	if err != nil {
		logger.ErrorCF("telegram", "Failed to get file", map[string]interface{}{
			"error": err.Error(),
		})
		return ""
	}

	return c.downloadFileWithInfo(file, ext)
}

func parseChatID(chatIDStr string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(chatIDStr, "%d", &id)
	return id, err
}

func markdownToTelegramHTML(text string) string {
	if text == "" {
		return ""
	}

	codeBlocks := extractCodeBlocks(text)
	text = codeBlocks.text

	tables := extractMarkdownTables(text)
	text = tables.text

	inlineCodes := extractInlineCodes(text)
	text = inlineCodes.text

	// Convert headings to bold markdown first, then transform to HTML later.
	text = regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`).ReplaceAllString(text, "**$1**")

	// Strip quote markers per line.
	text = regexp.MustCompile(`(?m)^>\s*(.*)$`).ReplaceAllString(text, "$1")

	text = escapeHTML(text)

	text = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`).ReplaceAllString(text, `<a href="$2">$1</a>`)

	text = regexp.MustCompile(`\*\*(.+?)\*\*`).ReplaceAllString(text, "<b>$1</b>")

	text = regexp.MustCompile(`__(.+?)__`).ReplaceAllString(text, "<b>$1</b>")

	reItalic := regexp.MustCompile(`_([^_]+)_`)
	text = reItalic.ReplaceAllStringFunc(text, func(s string) string {
		match := reItalic.FindStringSubmatch(s)
		if len(match) < 2 {
			return s
		}
		return "<i>" + match[1] + "</i>"
	})

	text = regexp.MustCompile(`~~(.+?)~~`).ReplaceAllString(text, "<s>$1</s>")

	for i, code := range inlineCodes.codes {
		escaped := escapeHTML(code)
		text = strings.ReplaceAll(text, fmt.Sprintf("\x00IC%d\x00", i), fmt.Sprintf("<code>%s</code>", escaped))
	}

	for i, table := range tables.tables {
		escaped := escapeHTML(table)
		text = strings.ReplaceAll(text, fmt.Sprintf("\x00TB%d\x00", i), fmt.Sprintf("<pre><code>%s</code></pre>", escaped))
	}

	for i, code := range codeBlocks.codes {
		escaped := escapeHTML(code)
		text = strings.ReplaceAll(text, fmt.Sprintf("\x00CB%d\x00", i), fmt.Sprintf("<pre><code>%s</code></pre>", escaped))
	}

	return text
}

type codeBlockMatch struct {
	text  string
	codes []string
}

type tableBlockMatch struct {
	text   string
	tables []string
}

func extractCodeBlocks(text string) codeBlockMatch {
	re := regexp.MustCompile("```[\\w]*\\n?([\\s\\S]*?)```")
	matches := re.FindAllStringSubmatch(text, -1)

	codes := make([]string, 0, len(matches))
	for _, match := range matches {
		codes = append(codes, match[1])
	}

	i := 0
	text = re.ReplaceAllStringFunc(text, func(m string) string {
		placeholder := fmt.Sprintf("\x00CB%d\x00", i)
		i++
		return placeholder
	})

	return codeBlockMatch{text: text, codes: codes}
}

type inlineCodeMatch struct {
	text  string
	codes []string
}

func extractInlineCodes(text string) inlineCodeMatch {
	re := regexp.MustCompile("`([^`]+)`")
	matches := re.FindAllStringSubmatch(text, -1)

	codes := make([]string, 0, len(matches))
	for _, match := range matches {
		codes = append(codes, match[1])
	}

	i := 0
	text = re.ReplaceAllStringFunc(text, func(m string) string {
		placeholder := fmt.Sprintf("\x00IC%d\x00", i)
		i++
		return placeholder
	})

	return inlineCodeMatch{text: text, codes: codes}
}

func extractMarkdownTables(text string) tableBlockMatch {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	tables := make([]string, 0)
	placeholderIdx := 0

	for i := 0; i < len(lines); {
		if i+1 < len(lines) && isTableRowLine(lines[i]) && isTableSeparatorLine(lines[i+1]) {
			start := i
			i += 2
			for i < len(lines) && isTableRowLine(lines[i]) {
				i++
			}
			block := lines[start:i]
			formatted := formatMarkdownTable(block)
			placeholder := fmt.Sprintf("\x00TB%d\x00", placeholderIdx)
			placeholderIdx++
			tables = append(tables, formatted)
			out = append(out, placeholder)
			continue
		}
		out = append(out, lines[i])
		i++
	}

	return tableBlockMatch{
		text:   strings.Join(out, "\n"),
		tables: tables,
	}
}

func isTableRowLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.Count(trimmed, "|") >= 2
}

func isTableSeparatorLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	trimmed = strings.Trim(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	if len(parts) == 0 {
		return false
	}
	for _, p := range parts {
		c := strings.TrimSpace(p)
		if c == "" {
			return false
		}
		for _, r := range c {
			if r != '-' && r != ':' && r != ' ' {
				return false
			}
		}
		if !strings.Contains(c, "-") {
			return false
		}
	}
	return true
}

func parseTableCells(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.Trim(trimmed, "|")
	raw := strings.Split(trimmed, "|")
	cells := make([]string, 0, len(raw))
	for _, c := range raw {
		cells = append(cells, strings.TrimSpace(c))
	}
	return cells
}

func formatMarkdownTable(lines []string) string {
	if len(lines) < 2 {
		return strings.Join(lines, "\n")
	}

	rows := make([][]string, 0, len(lines)-1)
	rows = append(rows, parseTableCells(lines[0])) // header
	for _, l := range lines[2:] {                  // skip separator
		rows = append(rows, parseTableCells(l))
	}

	cols := 0
	for _, r := range rows {
		if len(r) > cols {
			cols = len(r)
		}
	}
	if cols == 0 {
		return strings.Join(lines, "\n")
	}

	widths := make([]int, cols)
	for _, r := range rows {
		for i := 0; i < cols; i++ {
			cell := ""
			if i < len(r) {
				cell = r[i]
			}
			w := displayWidth(cell)
			if w > widths[i] {
				widths[i] = w
			}
		}
	}

	// Enforce horizontal width limit by shrinking widest columns first.
	totalWidth := func(ws []int) int {
		sum := 0
		for _, w := range ws {
			sum += w
		}
		if len(ws) == 0 {
			return 0
		}
		// "| " + " | ".join(cols) + " |"
		return sum + 4 + (len(ws)-1)*3
	}

	for totalWidth(widths) > markdownTableMaxWidth {
		maxIdx := -1
		maxW := 0
		for i, w := range widths {
			if w > maxW && w > markdownTableMinColWidth {
				maxW = w
				maxIdx = i
			}
		}
		if maxIdx == -1 {
			break
		}
		widths[maxIdx]--
	}

	padToWidth := func(s string, w int) string {
		dw := displayWidth(s)
		if dw >= w {
			return s
		}
		return s + strings.Repeat(" ", w-dw)
	}

	renderLogicalRow := func(r []string) []string {
		wrapped := make([][]string, cols)
		maxLines := 1
		for c := 0; c < cols; c++ {
			cell := ""
			if c < len(r) {
				cell = r[c]
			}
			lines := wrapByDisplayWidth(cell, widths[c])
			if len(lines) == 0 {
				lines = []string{""}
			}
			wrapped[c] = lines
			if len(lines) > maxLines {
				maxLines = len(lines)
			}
		}

		out := make([]string, 0, maxLines)
		for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
			var rowBuilder strings.Builder
			rowBuilder.WriteString("| ")
			for c := 0; c < cols; c++ {
				cellLine := ""
				if lineIdx < len(wrapped[c]) {
					cellLine = wrapped[c][lineIdx]
				}
				rowBuilder.WriteString(padToWidth(cellLine, widths[c]))
				if c == cols-1 {
					rowBuilder.WriteString(" |")
				} else {
					rowBuilder.WriteString(" | ")
				}
			}
			out = append(out, rowBuilder.String())
		}
		return out
	}

	var b strings.Builder
	for rIdx, r := range rows {
		rowLines := renderLogicalRow(r)
		for _, line := range rowLines {
			b.WriteString(line)
			b.WriteString("\n")
		}
		if rIdx == 0 {
			b.WriteString("| ")
			for c := 0; c < cols; c++ {
				b.WriteString(strings.Repeat("-", widths[c]))
				if c == cols-1 {
					b.WriteString(" |\n")
				} else {
					b.WriteString(" | ")
				}
			}
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		switch {
		case r == '\u200d' || r == '\u200c' || r == '\ufe0f':
			continue
		case unicode.Is(unicode.Mn, r):
			continue
		case isEmojiRune(r):
			w += 3
		case unicode.In(r,
			unicode.Han,
			unicode.Hiragana,
			unicode.Katakana,
			unicode.Hangul):
			w += 2
		case (r >= 0x3000 && r <= 0x303F) || (r >= 0xFF00 && r <= 0xFFEF):
			// CJK symbols/punctuation and half/fullwidth forms.
			w += 2
		default:
			w++
		}
	}
	return w
}

func isEmojiRune(r rune) bool {
	// Common emoji blocks + Dingbats/Stars used in rating tables.
	return (r >= 0x1F300 && r <= 0x1FAFF) || // Misc emoji/pictographs/symbols
		(r >= 0x2600 && r <= 0x27BF) || // Misc symbols + dingbats
		r == 0x2B50 // WHITE MEDIUM STAR (⭐)
}

func wrapByDisplayWidth(s string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{s}
	}
	if s == "" {
		return []string{""}
	}

	lines := make([]string, 0, 1)
	var cur strings.Builder
	curWidth := 0

	flush := func() {
		lines = append(lines, strings.TrimRight(cur.String(), " "))
		cur.Reset()
		curWidth = 0
	}

	for _, r := range s {
		if r == '\n' {
			flush()
			continue
		}
		rw := displayWidth(string(r))
		if rw == 0 {
			cur.WriteRune(r)
			continue
		}
		if curWidth+rw > maxWidth && cur.Len() > 0 {
			flush()
		}
		cur.WriteRune(r)
		curWidth += rw
	}

	if cur.Len() > 0 || len(lines) == 0 {
		flush()
	}

	return lines
}

func escapeHTML(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}
