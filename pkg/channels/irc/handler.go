package irc

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	multilineBatchType = "draft/multiline"
	multilineConcatTag = "draft/multiline-concat"
)

// onConnect is called after a successful connection (and on reconnect).
func (c *IRCChannel) onConnect(conn *ircevent.Connection) {
	// NickServ auth (only if SASL is not configured)
	if c.config.NickServPassword.String() != "" && c.config.SASLUser == "" {
		conn.Privmsg("NickServ", "IDENTIFY "+c.config.NickServPassword.String())
	}

	// Join configured channels
	for _, ch := range c.config.Channels {
		conn.Join(ch)
		logger.InfoCF("irc", "Joined IRC channel", map[string]any{
			"channel": ch,
		})
	}
}

// onMultilineBatch combines a completed IRCv3 multiline batch and sends it
// through the normal message path exactly once. If an outer batch contains a
// multiline child, it is flattened here so nested multiline batches still pass
// through this callback instead of being naively reduced to individual lines.
func (c *IRCChannel) onMultilineBatch(conn *ircevent.Connection, batch *ircevent.Batch) bool {
	msg, ok := assembleMultilineBatch(batch)
	if ok {
		conn.HandleMessage(msg)
		return true
	}
	if !containsMultilineBatch(batch) {
		return false
	}
	handleBatchItems(conn, batch.Items)
	return true
}

func containsMultilineBatch(batch *ircevent.Batch) bool {
	if batch == nil {
		return false
	}
	stack := append([]*ircevent.Batch(nil), batch.Items...)
	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if item == nil {
			continue
		}
		if item.Command == "BATCH" && len(item.Params) >= 2 && item.Params[1] == multilineBatchType {
			return true
		}
		stack = append(stack, item.Items...)
	}
	return false
}

func handleBatchItems(conn *ircevent.Connection, items []*ircevent.Batch) {
	stack := make([]*ircevent.Batch, 0, len(items))
	for i := len(items) - 1; i >= 0; i-- {
		stack = append(stack, items[i])
	}
	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if item == nil {
			continue
		}
		if item.Command != "BATCH" {
			conn.HandleMessage(item.Message)
			continue
		}
		if msg, ok := assembleMultilineBatch(item); ok {
			conn.HandleMessage(msg)
			continue
		}
		for i := len(item.Items) - 1; i >= 0; i-- {
			stack = append(stack, item.Items[i])
		}
	}
}

func assembleMultilineBatch(batch *ircevent.Batch) (ircmsg.Message, bool) {
	if batch == nil || batch.Command != "BATCH" || len(batch.Params) < 3 {
		return ircmsg.Message{}, false
	}
	if batch.Params[1] != multilineBatchType {
		return ircmsg.Message{}, false
	}

	target := batch.Params[2]
	if target == "" || len(batch.Items) == 0 {
		return ircmsg.Message{}, false
	}

	var combined strings.Builder
	result := batch.Message
	source := batch.Source
	for i, item := range batch.Items {
		if item == nil || item.Command != "PRIVMSG" || len(item.Params) < 2 || item.Params[0] != target {
			return ircmsg.Message{}, false
		}
		if i == 0 {
			if source == "" {
				source = item.Source
			}
			for name, value := range item.AllTags() {
				if name != "batch" && name != multilineConcatTag && !result.HasTag(name) {
					result.SetTag(name, value)
				}
			}
		} else if !item.HasTag(multilineConcatTag) {
			combined.WriteByte('\n')
		}
		if source != "" && item.Source != "" && item.Source != source {
			return ircmsg.Message{}, false
		}
		combined.WriteString(item.Params[1])
	}

	result.Source = source
	result.Command = "PRIVMSG"
	result.Params = []string{target, combined.String()}
	result.DeleteTag("batch")
	result.DeleteTag(multilineConcatTag)
	return result, true
}

// onPrivmsg handles incoming PRIVMSG events.
func (c *IRCChannel) onPrivmsg(conn *ircevent.Connection, e ircmsg.Message) {
	if len(e.Params) < 2 {
		return
	}

	nick := e.Nick()
	currentNick := conn.CurrentNick()

	// Ignore own messages
	if strings.EqualFold(nick, currentNick) {
		return
	}

	target := e.Params[0]  // channel name or bot's nick
	content := e.Params[1] // message text

	// Determine if this is a DM or channel message
	isDM := !strings.HasPrefix(target, "#") && !strings.HasPrefix(target, "&")

	var chatID string

	if isDM {
		chatID = nick
	} else {
		chatID = target
	}

	sender := bus.SenderInfo{
		Platform:    "irc",
		PlatformID:  nick,
		CanonicalID: identity.BuildCanonicalID("irc", nick),
		Username:    nick,
		DisplayName: nick,
	}

	if !c.IsAllowedSender(sender) {
		return
	}

	isMentioned := false

	// For channel messages, check group trigger (mention detection)
	if !isDM {
		isMentioned = isBotMentioned(content, currentNick)
		if isMentioned {
			content = stripBotMention(content, currentNick)
		}
		respond, cleaned := c.ShouldRespondInGroup(isMentioned, content)
		if !respond {
			return
		}
		content = cleaned
	}

	if strings.TrimSpace(content) == "" {
		return
	}

	messageID := fmt.Sprintf("%s-%d", nick, time.Now().UnixNano())

	metadata := map[string]string{
		"platform": "irc",
		"server":   c.config.Server,
	}
	if !isDM {
		metadata["channel"] = target
	}

	inboundCtx := bus.InboundContext{
		Channel:   "irc",
		ChatID:    chatID,
		SenderID:  nick,
		MessageID: messageID,
		Mentioned: isMentioned,
		Raw:       metadata,
	}
	if isDM {
		inboundCtx.ChatType = "direct"
	} else {
		inboundCtx.ChatType = "group"
	}

	c.HandleInboundContext(c.ctx, chatID, content, nil, inboundCtx, sender)
}

// nickMentionedAt returns the byte index where botNick is mentioned in content
// with word-boundary checks, or -1 if not found. Also checks for "nick:" /
// "nick," prefix convention.
func nickMentionedAt(content, botNick string) int {
	lower := strings.ToLower(content)
	lowerNick := strings.ToLower(botNick)

	// "nick:" or "nick," at start (most common IRC convention)
	if strings.HasPrefix(lower, lowerNick+":") || strings.HasPrefix(lower, lowerNick+",") {
		return 0
	}

	// Word-boundary match anywhere in the message
	idx := strings.Index(lower, lowerNick)
	if idx < 0 {
		return -1
	}
	runes := []rune(lower)
	nickRunes := []rune(lowerNick)
	endIdx := idx + len(string(nickRunes))
	before := idx == 0 || !unicode.IsLetter(runes[idx-1]) && !unicode.IsDigit(runes[idx-1])
	after := endIdx >= len(lower) || !unicode.IsLetter(rune(lower[endIdx])) && !unicode.IsDigit(rune(lower[endIdx]))
	if before && after {
		return idx
	}
	return -1
}

// isBotMentioned checks if the bot's nick appears in the message.
func isBotMentioned(content, botNick string) bool {
	return nickMentionedAt(content, botNick) >= 0
}

// stripBotMention removes "nick: " or "nick, " prefix from content.
func stripBotMention(content, botNick string) string {
	idx := nickMentionedAt(content, botNick)
	if idx != 0 {
		return content
	}
	lowerNick := strings.ToLower(botNick)
	lower := strings.ToLower(content)
	for _, sep := range []string{":", ","} {
		prefix := lowerNick + sep
		if strings.HasPrefix(lower, prefix) {
			return strings.TrimSpace(content[len(prefix):])
		}
	}
	return content
}
