// Package deltachat implements a PicoClaw channel for Delta Chat, an
// email-based, end-to-end encrypted messenger.
//
// PicoClaw does not link the Delta Chat core directly. Instead it drives a
// local `deltachat-rpc-server` process (shipped with the `deltachat-rpc-server`
// pip package or the precompiled release binary) over newline-delimited
// JSON-RPC 2.0 on stdio. This keeps the Go binary free of CGO/native deps.
package deltachat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mdp/qrterminal/v3"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// chatTypeSingle is Delta Chat's Chattype::Single — a 1:1 direct chat.
// The wire value is a string enum ("Single", "Group", "Mailinglist",
// "OutBroadcast", "InBroadcast"); anything other than Single is a group.
const chatTypeSingle = "Single"

// configureTimeout bounds the (network-bound) account configuration step.
const configureTimeout = 90 * time.Second

// dcAccount is one entry from get_all_accounts.
type dcAccount struct {
	ID   int64  `json:"id"`
	Kind string `json:"kind"`
	Addr string `json:"addr"`
}

// dcContact is the subset of Delta Chat's ContactObject we consume.
// NOTE: Delta Chat serializes object fields in camelCase on the wire (method
// names and config keys are snake_case, but struct fields are camelCase).
type dcContact struct {
	ID          int64  `json:"id"`
	Address     string `json:"address"`
	DisplayName string `json:"displayName"`
	Name        string `json:"name"`
}

// dcMessage is the subset of Delta Chat's MessageObject we consume.
type dcMessage struct {
	ID        int64      `json:"id"`
	ChatID    int64      `json:"chatId"`
	FromID    int64      `json:"fromId"`
	Text      string     `json:"text"`
	File      string     `json:"file"`
	FileName  string     `json:"fileName"`
	FileMime  string     `json:"fileMime"`
	Timestamp int64      `json:"timestamp"`
	IsInfo    bool       `json:"isInfo"`
	Sender    *dcContact `json:"sender"`
}

// dcChat is the subset of Delta Chat's FullChat we consume.
type dcChat struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	ChatType     string `json:"chatType"`
	IsDeviceChat bool   `json:"isDeviceChat"`
}

// DeltaChatChannel implements channels.Channel on top of deltachat-rpc-server.
type DeltaChatChannel struct {
	*channels.BaseChannel
	bc     *config.Channel
	config *config.DeltaChatSettings

	serverPath string
	dataDir    string

	rpc       *rpcClient
	accountID int64
	selfAddr  string

	ctx    context.Context
	cancel context.CancelFunc
}

// NewDeltaChatChannel validates config and resolves the RPC server + data dir.
func NewDeltaChatChannel(
	bc *config.Channel,
	cfg *config.DeltaChatSettings,
	messageBus *bus.MessageBus,
) (*DeltaChatChannel, error) {
	if cfg.Email == "" {
		return nil, fmt.Errorf("deltachat: email is required")
	}
	if cfg.Password.String() == "" {
		return nil, fmt.Errorf("deltachat: password is required")
	}

	serverPath, err := resolveServerPath(cfg.RPCServerPath)
	if err != nil {
		return nil, err
	}
	dataDir := resolveDataDir(cfg.DataDir, bc.Name())

	base := channels.NewBaseChannel(config.ChannelDeltaChat, cfg, messageBus, bc.AllowFrom,
		channels.WithMaxMessageLength(0), // email has no practical length limit
		channels.WithGroupTrigger(bc.GroupTrigger),
		channels.WithReasoningChannelID(bc.ReasoningChannelID),
	)

	ch := &DeltaChatChannel{
		BaseChannel: base,
		bc:          bc,
		config:      cfg,
		serverPath:  serverPath,
		dataDir:     dataDir,
	}
	base.SetOwner(ch)
	return ch, nil
}

// Start spawns the RPC server, ensures the account is configured, and begins
// listening for messages.
func (c *DeltaChatChannel) Start(ctx context.Context) error {
	logger.InfoC("deltachat", "Starting Delta Chat channel")
	c.ctx, c.cancel = context.WithCancel(ctx)

	if err := os.MkdirAll(c.dataDir, 0o700); err != nil {
		return fmt.Errorf("deltachat: create data dir %s: %w", c.dataDir, err)
	}

	rpc, err := startRPC(c.serverPath, c.dataDir)
	if err != nil {
		return err
	}
	c.rpc = rpc

	if err := c.waitReady(c.ctx); err != nil {
		c.rpc.close()
		return err
	}

	if err := c.ensureAccount(c.ctx); err != nil {
		c.rpc.close()
		return err
	}

	if err := c.joinInviteLink(c.ctx); err != nil {
		logger.WarnCF("deltachat", "Failed to join invite link", map[string]any{"error": err.Error()})
	}

	c.SetRunning(true)
	go c.listen()

	logger.InfoCF("deltachat", "Delta Chat channel started", map[string]any{
		"email":      c.selfAddr,
		"account_id": c.accountID,
	})

	// Print the bot's invite link + QR so users can add it. Delta Chat / chatmail
	// require end-to-end encryption, so peers must obtain the bot's key via this
	// invite (adding the bare email address will not work).
	c.printInviteLink(c.ctx)

	return nil
}

// printInviteLink fetches the account-level secure-join invite link and prints
// it (with a scannable QR) to the terminal and log.
func (c *DeltaChatChannel) printInviteLink(ctx context.Context) {
	raw, err := c.rpc.call(ctx, "get_chat_securejoin_qr_code", c.accountID, nil)
	if err != nil {
		logger.WarnCF("deltachat", "Could not generate invite link", map[string]any{"error": err.Error()})
		return
	}
	var link string
	if err := json.Unmarshal(raw, &link); err != nil || link == "" {
		return
	}

	logger.InfoCF("deltachat", "Invite link", map[string]any{"link": link})
	fmt.Printf("\n📨 Delta Chat invite for %s — scan with Delta Chat (➕ → Scan/Paste QR) to message the bot:\n   %s\n\n",
		c.config.Email, link)
	qrterminal.GenerateWithConfig(link, qrterminal.Config{
		Level:      qrterminal.L,
		Writer:     os.Stdout,
		HalfBlocks: true,
	})
	fmt.Println()
}

// Stop stops IO and terminates the RPC server.
func (c *DeltaChatChannel) Stop(ctx context.Context) error {
	logger.InfoC("deltachat", "Stopping Delta Chat channel")
	c.SetRunning(false)
	if c.cancel != nil {
		c.cancel()
	}
	if c.rpc != nil && c.accountID > 0 {
		stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, _ = c.rpc.call(stopCtx, "stop_io", c.accountID)
		cancel()
	}
	if c.rpc != nil {
		c.rpc.close()
	}
	logger.InfoC("deltachat", "Delta Chat channel stopped")
	return nil
}

// Send delivers an outbound message to a Delta Chat chat. ChatID is the numeric
// Delta Chat chat id (as a string).
func (c *DeltaChatChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}
	if strings.TrimSpace(msg.Content) == "" {
		return nil, nil
	}

	chatID, err := strconv.ParseInt(strings.TrimSpace(msg.ChatID), 10, 64)
	if err != nil || chatID <= 0 {
		return nil, fmt.Errorf("deltachat: invalid chat id %q: %w", msg.ChatID, channels.ErrSendFailed)
	}

	// misc_send_msg(account_id, chat_id, text, file, name, location, quoted_message_id)
	raw, err := c.rpc.call(ctx, "misc_send_msg", c.accountID, chatID, msg.Content, nil, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("deltachat send: %w", err)
	}

	// Result is [message_id, message_object]; we only need the id.
	var result []json.RawMessage
	if err := json.Unmarshal(raw, &result); err == nil && len(result) > 0 {
		var messageID int64
		if err := json.Unmarshal(result[0], &messageID); err == nil {
			return []string{strconv.FormatInt(messageID, 10)}, nil
		}
	}
	return nil, nil
}

// StartTyping implements channels.TypingCapable. Delta Chat has no typing
// indicator over email, so this is a no-op that satisfies the interface and
// lets the Manager skip the placeholder dance gracefully.
func (c *DeltaChatChannel) StartTyping(ctx context.Context, chatID string) (func(), error) {
	return func() {}, nil
}

// waitReady polls get_system_info until the RPC server responds.
func (c *DeltaChatChannel) waitReady(ctx context.Context) error {
	for attempt := 0; attempt < 40; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		_, err := c.rpc.call(callCtx, "get_system_info")
		cancel()
		if err == nil {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("deltachat: rpc server did not become ready")
}

// ensureAccount finds or creates the configured account and starts its IO.
func (c *DeltaChatChannel) ensureAccount(ctx context.Context) error {
	c.selfAddr = strings.ToLower(c.config.Email)

	accounts, err := c.listAccounts(ctx)
	if err != nil {
		return err
	}

	var accountID int64
	for _, acc := range accounts {
		if acc.Kind == "Configured" && strings.EqualFold(acc.Addr, c.config.Email) {
			accountID = acc.ID
			break
		}
	}

	if accountID == 0 {
		raw, callErr := c.rpc.call(ctx, "add_account")
		if callErr != nil {
			return fmt.Errorf("deltachat add_account: %w", callErr)
		}
		if decErr := json.Unmarshal(raw, &accountID); decErr != nil {
			return fmt.Errorf("deltachat add_account decode: %w", decErr)
		}
	}

	configured, err := c.isConfigured(ctx, accountID)
	if err != nil {
		return err
	}
	if !configured {
		if err := c.configureAccount(ctx, accountID); err != nil {
			return err
		}
	}

	if _, err := c.rpc.call(ctx, "select_account", accountID); err != nil {
		return fmt.Errorf("deltachat select_account: %w", err)
	}
	// Mark this account as a bot so the core delivers all messages to us.
	if _, err := c.rpc.call(ctx, "batch_set_config", accountID, map[string]string{"bot": "1"}); err != nil {
		return fmt.Errorf("deltachat set bot config: %w", err)
	}
	if _, err := c.rpc.call(ctx, "start_io", accountID); err != nil {
		return fmt.Errorf("deltachat start_io: %w", err)
	}

	c.accountID = accountID
	return nil
}

// configureAccount writes the credentials and runs the (network-bound)
// provider auto-configuration.
func (c *DeltaChatChannel) configureAccount(ctx context.Context, accountID int64) error {
	cfgMap := accountConfigMap(c.config)
	if _, err := c.rpc.call(ctx, "batch_set_config", accountID, cfgMap); err != nil {
		return fmt.Errorf("deltachat set credentials: %w", err)
	}

	logger.InfoCF("deltachat", "Configuring account (validating credentials)", map[string]any{
		"email": c.config.Email,
	})
	confCtx, cancel := context.WithTimeout(ctx, configureTimeout)
	defer cancel()
	if _, err := c.rpc.call(confCtx, "configure", accountID); err != nil {
		return fmt.Errorf("deltachat configure (check email/password/server): %w", err)
	}
	return nil
}

func accountConfigMap(cfg *config.DeltaChatSettings) map[string]string {
	cfgMap := map[string]string{
		"addr":    cfg.Email,
		"mail_pw": cfg.Password.String(),
	}
	if cfg.DisplayName != "" {
		cfgMap["displayname"] = cfg.DisplayName
	}
	if cfg.IMAPServer != "" {
		cfgMap["mail_server"] = cfg.IMAPServer
	}
	if cfg.IMAPPort > 0 {
		cfgMap["mail_port"] = strconv.Itoa(cfg.IMAPPort)
	}
	if cfg.SMTPServer != "" {
		cfgMap["send_server"] = cfg.SMTPServer
	}
	if cfg.SMTPPort > 0 {
		cfgMap["send_port"] = strconv.Itoa(cfg.SMTPPort)
	}
	return cfgMap
}

func (c *DeltaChatChannel) listAccounts(ctx context.Context) ([]dcAccount, error) {
	raw, err := c.rpc.call(ctx, "get_all_accounts")
	if err != nil {
		return nil, fmt.Errorf("deltachat get_all_accounts: %w", err)
	}
	var accounts []dcAccount
	if err := json.Unmarshal(raw, &accounts); err != nil {
		return nil, fmt.Errorf("deltachat get_all_accounts decode: %w", err)
	}
	return accounts, nil
}

func (c *DeltaChatChannel) isConfigured(ctx context.Context, accountID int64) (bool, error) {
	raw, err := c.rpc.call(ctx, "is_configured", accountID)
	if err != nil {
		return false, fmt.Errorf("deltachat is_configured: %w", err)
	}
	var ok bool
	if err := json.Unmarshal(raw, &ok); err != nil {
		return false, fmt.Errorf("deltachat is_configured decode: %w", err)
	}
	return ok, nil
}

// joinInviteLink optionally joins a chat via a configured invite/QR link.
func (c *DeltaChatChannel) joinInviteLink(ctx context.Context) error {
	link := strings.TrimSpace(c.config.InviteLink)
	if link == "" {
		return nil
	}
	chatRaw, err := c.rpc.call(ctx, "secure_join", c.accountID, link)
	if err != nil {
		return err
	}
	var chatID int64
	if err := json.Unmarshal(chatRaw, &chatID); err == nil && chatID > 0 {
		_, _ = c.rpc.call(ctx, "accept_chat", c.accountID, chatID)
		logger.InfoCF("deltachat", "Joined invite chat", map[string]any{"chat_id": chatID})
	}
	return nil
}

// resolveServerPath validates the user-configured deltachat-rpc-server path.
func resolveServerPath(configured string) (string, error) {
	if configured == "" {
		return "", fmt.Errorf("deltachat: rpc_server_path is required " +
			"(set it to the path of your deltachat-rpc-server binary)")
	}
	p := expandHome(configured)
	if !fileExists(p) {
		return "", fmt.Errorf("deltachat: rpc_server_path %q not found", p)
	}
	return p, nil
}

// resolveDataDir picks where the account database lives.
func resolveDataDir(configured, channelName string) string {
	if configured != "" {
		return expandHome(configured)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	name := channelName
	if name == "" {
		name = config.ChannelDeltaChat
	}
	return filepath.Join(home, ".picoclaw", "deltachat", name)
}

func expandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if len(path) == 1 {
		return home
	}
	if path[1] == '/' {
		return filepath.Join(home, path[2:])
	}
	return path
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
