package chatmail

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/chatmail/rpc-client-go/v2/deltachat"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type ChatmailChannel struct {
	*channels.BaseChannel
	config    config.ChatmailConfig
	rpc       *deltachat.Rpc
	bot       *deltachat.Bot
	transport *deltachat.IOTransport
	ctx       context.Context
	cancel    context.CancelFunc
	accId     uint32
	mu        sync.RWMutex
}

func NewChatmailChannel(cfg config.ChatmailConfig, messageBus *bus.MessageBus) (*ChatmailChannel, error) {
	base := channels.NewBaseChannel("chatmail", cfg, messageBus, cfg.AllowFrom,
		channels.WithGroupTrigger(cfg.GroupTrigger),
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &ChatmailChannel{
		BaseChannel: base,
		config:      cfg,
	}, nil
}

func (c *ChatmailChannel) Start(ctx context.Context) error {
	logger.InfoC("chatmail", "Starting Chatmail channel")

	c.ctx, c.cancel = context.WithCancel(ctx)

	accountPath := c.config.AccountPath
	if accountPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		accountPath = filepath.Join(homeDir, ".accounts", "chatmail")
	}

	if err := os.MkdirAll(accountPath, 0700); err != nil {
		return fmt.Errorf("failed to create account directory: %w", err)
	}

	transport := deltachat.NewIOTransport()
	transport.AccountsDir = accountPath

	if err := transport.Open(); err != nil {
		return fmt.Errorf("failed to open transport: %w", err)
	}
	c.transport = transport

	c.rpc = &deltachat.Rpc{Context: c.ctx, Transport: transport}

	accounts, err := c.rpc.GetAllAccountIds()
	if err != nil {
		transport.Close()
		return fmt.Errorf("failed to get accounts: %w", err)
	}

	var accId uint32
	if len(accounts) == 0 {
		accId, err = c.rpc.AddAccount()
		if err != nil {
			transport.Close()
			return fmt.Errorf("failed to add account: %w", err)
		}
		logger.InfoCF("chatmail", "Created new account", map[string]any{"account_id": accId})
	} else {
		accId = accounts[0]
		logger.InfoCF("chatmail", "Using existing account", map[string]any{"account_id": accId})
	}
	c.accId = accId

	isConfigured, err := c.rpc.IsConfigured(accId)
	if err != nil {
		transport.Close()
		return fmt.Errorf("failed to check account configuration: %w", err)
	}

	if !isConfigured {
		botFlag := "1"
		if err := c.rpc.SetConfig(accId, "bot", &botFlag); err != nil {
			transport.Close()
			return fmt.Errorf("failed to set bot flag: %w", err)
		}
		logger.InfoC("chatmail", "Account configured as bot")

		inviteQR := c.config.InviteQR
		if inviteQR == "" {
			inviteQR = "dcaccount:https://nine.testrun.org/new"
		}

		if err := c.rpc.AddTransportFromQr(accId, inviteQR); err != nil {
			transport.Close()
			return fmt.Errorf("failed to add transport from QR: %w", err)
		}
		logger.InfoCF("chatmail", "Account configured from invite", map[string]any{"qr": inviteQR})
	}

	inviteLink, err := c.rpc.GetChatSecurejoinQrCode(accId, nil)
	if err != nil {
		logger.WarnCF("chatmail", "Failed to get invite link", map[string]any{"error": err.Error()})
	} else {
		logger.InfoCF("chatmail", "Invite link", map[string]any{"link": inviteLink})
		fmt.Printf("\nChatmail invite link: %s\n", inviteLink)
		fmt.Println("Scan this QR code with your Delta Chat app to start chatting.")
	}

	msgIds, err := c.rpc.GetNextMsgs(accId)
	if err != nil {
		logger.WarnCF("chatmail", "Failed to fetch pending messages", map[string]any{"error": err.Error()})
	} else if len(msgIds) > 0 {
		lastMsgId := fmt.Sprintf("%v", msgIds[len(msgIds)-1])
		if err := c.rpc.SetConfig(accId, "last_msg_id", &lastMsgId); err != nil {
			logger.WarnCF("chatmail", "Failed to set last_msg_id", map[string]any{"error": err.Error()})
		} else {
			// Mark all pending messages as seen
			if err := c.rpc.MarkseenMsgs(accId, msgIds); err != nil {
				logger.DebugCF("chatmail", "Failed to mark pending messages as seen", map[string]any{
					"count": len(msgIds),
					"error": err.Error(),
				})
			}
			logger.InfoCF("chatmail", "Ignored pending messages on startup", map[string]any{"count": len(msgIds)})
		}
	}

	c.bot = deltachat.NewBot(c.rpc)
	c.bot.OnNewMsg(c.onNewMessage)

	go func() {
		if err := c.bot.Run(); err != nil {
			logger.ErrorCF("chatmail", "Bot run error", map[string]any{"error": err.Error()})
		}
	}()

	c.SetRunning(true)
	logger.InfoC("chatmail", "Chatmail channel started")

	return nil
}

func (c *ChatmailChannel) Stop(ctx context.Context) error {
	logger.InfoC("chatmail", "Stopping Chatmail channel")
	c.SetRunning(false)

	if c.bot != nil {
		c.bot.Stop()
	}

	if c.cancel != nil {
		c.cancel()
	}

	if c.transport != nil {
		c.transport.Close()
	}

	logger.InfoC("chatmail", "Chatmail channel stopped")
	return nil
}

func (c *ChatmailChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	chatId, err := c.parseChatID(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", channels.ErrSendFailed)
	}

	if msg.Content == "" {
		return nil
	}

	text := msg.Content
	msgData := deltachat.MessageData{Text: &text}

	_, err = c.rpc.SendMsg(c.accId, chatId, msgData)
	if err != nil {
		logger.ErrorCF("chatmail", "Failed to send message", map[string]any{
			"chat_id": chatId,
			"error":   err.Error(),
		})
		return fmt.Errorf("send failed: %w", channels.ErrSendFailed)
	}

	logger.DebugCF("chatmail", "Message sent", map[string]any{"chat_id": chatId})
	return nil
}

func (c *ChatmailChannel) parseChatID(chatID string) (uint32, error) {
	var chatId uint32
	_, err := fmt.Sscanf(chatID, "%d", &chatId)
	if err != nil {
		return 0, err
	}
	return chatId, nil
}

func (c *ChatmailChannel) parseMsgID(messageID string) (uint32, error) {
	var msgId uint32
	_, err := fmt.Sscanf(messageID, "%d", &msgId)
	if err != nil {
		return 0, err
	}
	return msgId, nil
}

func (c *ChatmailChannel) ReactToMessage(ctx context.Context, chatID, messageID string) (func(), error) {
	if !c.IsRunning() {
		return func() {}, nil
	}

	msgId, err := c.parseMsgID(messageID)
	if err != nil {
		return func() {}, nil
	}

	reactions := []string{"\U0001F440"}
	if _, err := c.rpc.SendReaction(c.accId, msgId, reactions); err != nil {
		logger.DebugCF("chatmail", "Failed to add reaction", map[string]any{"error": err.Error()})
		return func() {}, nil
	}

	// Keep the reaction permanently - return no-op undo function
	return func() {}, nil
}

func (c *ChatmailChannel) onNewMessage(bot *deltachat.Bot, accId uint32, msgId uint32) {
	msg, err := c.rpc.GetMessage(accId, msgId)
	if err != nil {
		logger.ErrorCF("chatmail", "Failed to get message", map[string]any{
			"msg_id": msgId,
			"error":  err.Error(),
		})
		return
	}

	if msg.FromId <= deltachat.ContactLastSpecial {
		return
	}

	// Build sender info early for permission check
	senderID := fmt.Sprintf("%d", msg.FromId)
	sender := bus.SenderInfo{
		Platform:    "chatmail",
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID("chatmail", senderID),
	}

	contact, _ := c.rpc.GetContact(accId, msg.FromId)
	if contact.DisplayName != "" {
		sender.DisplayName = contact.DisplayName
	}

	// Check authorization BEFORE processing
	if !c.IsAllowedSender(sender) {
		accountInfo := senderID

		rejectionText := "⛔ Access denied. You are not authorized to use this bot.\n\n" +
			"To get access, ask the administrator to add this account to the configuration:\n\n" +
			"Account ID: " + accountInfo

		text := rejectionText
		msgData := deltachat.MessageData{Text: &text}
		c.rpc.SendMsg(c.accId, msg.ChatId, msgData)
		c.rpc.MarkseenMsgs(c.accId, []uint32{msgId})

		logger.InfoCF("chatmail", "Unauthorized user rejected", map[string]any{
			"sender_id":    senderID,
			"display_name": sender.DisplayName,
		})
		return
	}

	// Authorized - continue with normal flow
	chatIdStr := fmt.Sprintf("%d", msg.ChatId)
	chat, err := c.rpc.GetBasicChatInfo(accId, msg.ChatId)
	if err != nil {
		logger.ErrorCF("chatmail", "Failed to get chat info", map[string]any{
			"chat_id": msg.ChatId,
			"error":   err.Error(),
		})
		return
	}

	var peer bus.Peer
	isGroup := chat.ChatType == deltachat.ChatTypeGroup
	if isGroup {
		peer = bus.Peer{Kind: "group", ID: chatIdStr}
	} else {
		peer = bus.Peer{Kind: "direct", ID: chatIdStr}
	}

	content := msg.Text
	metadata := map[string]string{
		"platform":  "chatmail",
		"chat_type": string(chat.ChatType),
	}

	c.mu.RLock()
	ctx := c.ctx
	c.mu.RUnlock()

	c.HandleMessage(ctx, peer, fmt.Sprintf("%d", msgId), senderID, chatIdStr, content, nil, metadata, sender)

	// Mark the message as seen after processing
	if err := c.rpc.MarkseenMsgs(accId, []uint32{msgId}); err != nil {
		logger.DebugCF("chatmail", "Failed to mark message as seen", map[string]any{
			"msg_id": msgId,
			"error":  err.Error(),
		})
	}
}
