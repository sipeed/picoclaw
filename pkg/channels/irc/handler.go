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

// onConnect is called after a successful connection (and on reconnect).
func (c *IRCChannel) onConnect(conn *ircevent.Connection) {
	// NickServ auth (only if SASL is not configured)
	if c.config.NickServPassword() != "" && c.config.SASLUser == "" {
		conn.Privmsg("NickServ", "IDENTIFY "+c.config.NickServPassword())
	}

	// Join configured channels
	for _, ch := range c.config.Channels {
		conn.Join(ch)
		logger.InfoCF("irc", "Joined IRC channel", map[string]any{
			"channel": ch,
		})
	}
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
	var peer bus.Peer

	if isDM {
		chatID = nick
		peer = bus.Peer{Kind: "direct", ID: nick}
	} else {
		chatID = target
		peer = bus.Peer{Kind: "group", ID: target}
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

	// For channel messages, check group trigger (mention detection)
	if !isDM {
		isMentioned := isBotMentioned(content, currentNick)
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

	c.HandleMessage(c.ctx, peer, messageID, nick, chatID, content, nil, metadata, sender)
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
