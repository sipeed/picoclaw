package irc

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// IRCChannel implements the Channel interface for IRC servers.
type IRCChannel struct {
	*channels.BaseChannel
	config config.IRCConfig
	conn   *ircevent.Connection
	ctx    context.Context
	cancel context.CancelFunc
}

// NewIRCChannel creates a new IRC channel.
func NewIRCChannel(cfg config.IRCConfig, messageBus *bus.MessageBus) (*IRCChannel, error) {
	if cfg.Server == "" {
		return nil, fmt.Errorf("irc server is required")
	}
	if cfg.Nick == "" {
		return nil, fmt.Errorf("irc nick is required")
	}

	base := channels.NewBaseChannel("irc", cfg, messageBus, cfg.AllowFrom,
		channels.WithMaxMessageLength(400),
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &IRCChannel{
		BaseChannel: base,
		config:      cfg,
	}, nil
}

// Start connects to the IRC server and begins listening.
func (c *IRCChannel) Start(ctx context.Context) error {
	logger.InfoC("irc", "Starting IRC channel")
	c.ctx, c.cancel = context.WithCancel(ctx)

	user := c.config.User
	if user == "" {
		user = c.config.Nick
	}
	realName := c.config.RealName
	if realName == "" {
		realName = c.config.Nick
	}
	caps := []string(c.config.RequestCaps)
	if len(caps) == 0 {
		caps = []string{"server-time", "message-tags"}
	}

	conn := &ircevent.Connection{
		Server:      c.config.Server,
		Nick:        c.config.Nick,
		User:        user,
		RealName:    realName,
		Password:    c.config.Password(),
		UseTLS:      c.config.TLS,
		RequestCaps: caps,
		QuitMessage: "Goodbye",
		Debug:       false,
		Log:         nil,
	}

	if c.config.TLS {
		conn.TLSConfig = &tls.Config{
			ServerName: extractHost(c.config.Server),
		}
	}

	// SASL auth (takes priority over NickServ)
	if c.config.SASLUser != "" && c.config.SASLPassword() != "" {
		conn.SASLLogin = c.config.SASLUser
		conn.SASLPassword = c.config.SASLPassword()
	}

	// Register event handlers
	conn.AddConnectCallback(func(e ircmsg.Message) {
		c.onConnect(conn)
	})
	conn.AddCallback("PRIVMSG", func(e ircmsg.Message) {
		c.onPrivmsg(conn, e)
	})

	if err := conn.Connect(); err != nil {
		return fmt.Errorf("irc connect failed: %w", err)
	}

	c.conn = conn

	// ircevent.Connection.Loop() handles reconnection internally.
	go conn.Loop()

	c.SetRunning(true)
	logger.InfoCF("irc", "IRC channel started", map[string]any{
		"server": c.config.Server,
		"nick":   c.config.Nick,
	})
	return nil
}

// Stop disconnects from the IRC server.
func (c *IRCChannel) Stop(ctx context.Context) error {
	logger.InfoC("irc", "Stopping IRC channel")
	c.SetRunning(false)

	if c.conn != nil {
		c.conn.Quit()
	}
	if c.cancel != nil {
		c.cancel()
	}

	logger.InfoC("irc", "IRC channel stopped")
	return nil
}

// Send sends a message to an IRC channel or user.
func (c *IRCChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	target := msg.ChatID
	if target == "" {
		return fmt.Errorf("chat ID is empty: %w", channels.ErrSendFailed)
	}

	if strings.TrimSpace(msg.Content) == "" {
		return nil
	}

	// Send each line separately (IRC is line-oriented)
	lines := strings.Split(msg.Content, "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		c.conn.Privmsg(target, line)
	}

	logger.DebugCF("irc", "Message sent", map[string]any{
		"target": target,
		"lines":  len(lines),
	})
	return nil
}

// StartTyping implements channels.TypingCapable using IRCv3 +typing client tag.
// Requires typing.enabled in config and server support for message-tags capability.
func (c *IRCChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	noop := func() {}

	if !c.config.Typing.Enabled || !c.IsRunning() || c.conn == nil {
		return noop, nil
	}

	// Check if server supports message-tags (required for TAGMSG)
	if _, ok := c.conn.AcknowledgedCaps()["message-tags"]; !ok {
		return noop, nil
	}

	c.conn.SendWithTags(map[string]string{"+typing": "active"}, "TAGMSG", chatID)

	return func() {
		if c.IsRunning() && c.conn != nil {
			c.conn.SendWithTags(map[string]string{"+typing": "done"}, "TAGMSG", chatID)
		}
	}, nil
}

// extractHost returns the hostname portion of a host:port string.
func extractHost(server string) string {
	host, _, found := strings.Cut(server, ":")
	if found {
		return host
	}
	return server
}
