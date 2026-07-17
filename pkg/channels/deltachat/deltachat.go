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
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mdp/qrterminal/v3"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/identity"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
)

// chatTypeSingle is Delta Chat's Chattype::Single — a 1:1 direct chat.
// The wire value is a string enum ("Single", "Group", "Mailinglist",
// "OutBroadcast", "InBroadcast"); anything other than Single is a group.
const chatTypeSingle = "Single"

// configureTimeout bounds the (network-bound) account configuration step.
const configureTimeout = 90 * time.Second

type chatmailRelay struct {
	Domain   string
	Location string
}

// Keep this list in sync with Parla's CHATMAIL_RELAYS in ../parla/src/relay_picker.vala.
var defaultChatmailRelays = []chatmailRelay{
	{"nine.testrun.org", "Default"},
	{"mehl.cloud", "German"},
	{"mailchat.pl", "Poland"},
	{"chatmail.woodpeckersnest.space", "Italy"},
	{"chatmail.culturanerd.it", "Italy"},
	{"chat.adminforge.de", "Falkenstein, Germany"},
	{"chika.aangat.lahat.computer", "Santa Clara, USA"},
	{"tarpit.fun", "Nuremberg, Germany"},
	{"d.gaufr.es", "Roubaix, France"},
	{"chtml.ca", "Quebec, Canada"},
	{"chatmail.au", "Melbourne, Australia"},
	{"e2ee.wang", "Johannesburg, South Africa"},
	{"chat.privittytech.com", "Bangalore, India"},
	{"e2ee.im", "Orastie, Romania"},
	{"chatmail.email", "Warsaw, Poland"},
	{"danneskjold.de", "Helsinki, Finland"},
	{"chat.in-the.eu", "Falkenstein, Germany"},
	{"chat.nuvon.app", "Prague, Czechia"},
	{"nibblehole.com", "Zug, Switzerland"},
	{"chat.zashm.org", "Lviv, Ukraine"},
	{"chat.sus.fr", "Iceland/Japan/Kenya/South Africa"},
	{"delta.thelab.uno", "Gravelines, France"},
	{"chat.vim.wtf", "Frankfurt, Germany"},
	{"uninterest.ing", "Elk Grove Village, USA"},
	{"sweetfern.net", "Ashburn, USA"},
	{"delta.disobey.net", "Roon, Netherlands"},
}

var managedAccountConfigKeys = []string{
	"addr",
	"mail_server",
	"mail_port",
	"send_server",
	"send_port",
}

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
	NameAndAddr string `json:"nameAndAddr"`
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
	CanSend      bool   `json:"canSend"`
}

// dcMessageData mirrors the fields of Delta Chat's MessageData (camelCase on the
// wire) that PicoClaw sets when calling send_msg. Viewtype is normally left empty
// so Delta Chat infers it from the file (image/gif/video/file…); it is set only
// for voice replies, which must be Viewtype::Voice to render as a voice bubble
// rather than a generic audio attachment.
type dcMessageData struct {
	Text     string `json:"text,omitempty"`
	File     string `json:"file,omitempty"`
	Filename string `json:"filename,omitempty"`
	Viewtype string `json:"viewtype,omitempty"`
}

// Ensure DeltaChatChannel satisfies the optional capability interfaces so the
// Manager routes media to it and the gateway advertises voice support.
var (
	_ channels.MediaSender             = (*DeltaChatChannel)(nil)
	_ channels.VoiceCapabilityProvider = (*DeltaChatChannel)(nil)
)

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

func parseDeltaChatEmailSetting(value string) (string, bool, error) {
	email := strings.TrimSpace(value)
	if email == "" {
		return "", false, fmt.Errorf(
			"deltachat: email is required.\nNext step: choose one of the chatmail servers below and set channel_list.deltachat.settings.email to %q, or use the same @server form with another chatmail relay. Run `picoclaw g` again; PicoClaw will create the account, print the generated full email address, and stop so you can save that address in the config.\nAvailable chatmail servers:\n%s",
			"@"+defaultChatmailRelays[0].Domain,
			formatChatmailRelayList(),
		)
	}
	if !strings.HasPrefix(email, "@") {
		return email, false, nil
	}
	domain := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(email, "@")))
	if domain == "" {
		return "", false, fmt.Errorf("deltachat: email %q is missing a chatmail server. Use %q or one of:\n%s",
			email, "@"+defaultChatmailRelays[0].Domain, formatChatmailRelayList())
	}
	if strings.Contains(domain, "@") || strings.ContainsAny(domain, "/\\ \t\r\n") {
		return "", false, fmt.Errorf("deltachat: invalid chatmail server marker %q; use settings.email like %q",
			email, "@"+defaultChatmailRelays[0].Domain)
	}
	return domain, true, nil
}

func formatChatmailRelayList() string {
	var b strings.Builder
	for _, relay := range defaultChatmailRelays {
		fmt.Fprintf(&b, "  @%-34s %s\n", relay.Domain, relay.Location)
	}
	return strings.TrimRight(b.String(), "\n")
}

func buildChatmailAccountQR(domain string) string {
	return fmt.Sprintf("DCACCOUNT:https://%s/new", domain)
}

// NewDeltaChatChannel validates config and resolves the RPC server + data dir.
func NewDeltaChatChannel(
	bc *config.Channel,
	cfg *config.DeltaChatSettings,
	messageBus *bus.MessageBus,
) (*DeltaChatChannel, error) {
	if _, _, err := parseDeltaChatEmailSetting(cfg.Email); err != nil {
		return nil, err
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

// Send delivers an outbound message to a Delta Chat chat. ChatID can be the
// numeric Delta Chat chat id, an email address, or a known contact/chat name.
func (c *DeltaChatChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}
	if strings.TrimSpace(msg.Content) == "" {
		return nil, nil
	}

	chatID, err := c.resolveOutboundChatID(ctx, msg.ChatID, msg.Context, msg.Scope)
	if err != nil {
		return nil, err
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

// SendMedia implements channels.MediaSender. Each part is resolved to a local
// file and delivered as its own Delta Chat message, with the part caption as the
// message text. Delta Chat copies the file into its blob store and infers the
// view type from the file itself.
func (c *DeltaChatChannel) SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	chatID, err := c.resolveOutboundChatID(ctx, msg.ChatID, msg.Context, msg.Scope)
	if err != nil {
		return nil, err
	}

	store := c.GetMediaStore()
	if store == nil {
		return nil, fmt.Errorf("deltachat: no media store available: %w", channels.ErrSendFailed)
	}

	var messageIDs []string
	for _, part := range msg.Parts {
		localPath, meta, err := store.ResolveWithMeta(part.Ref)
		if err != nil {
			logger.ErrorCF("deltachat", "Failed to resolve media ref", map[string]any{
				"ref":   part.Ref,
				"error": err.Error(),
			})
			continue
		}
		// Delta Chat needs a path it can open from its own working directory;
		// absolutize defensively in case the store ever yields a relative one.
		if abs, absErr := filepath.Abs(localPath); absErr == nil {
			localPath = abs
		}

		data := dcMessageData{
			Text:     part.Caption,
			File:     localPath,
			Filename: part.Filename,
			Viewtype: deltaChatViewtype(part, meta),
		}
		raw, err := c.rpc.call(ctx, "send_msg", c.accountID, chatID, data)
		if err != nil {
			logger.ErrorCF("deltachat", "Failed to send media", map[string]any{
				"ref":   part.Ref,
				"error": err.Error(),
			})
			return messageIDs, fmt.Errorf("deltachat send media: %w", channels.ErrTemporary)
		}

		// send_msg returns the new message id as a bare integer.
		var messageID int64
		if err := json.Unmarshal(raw, &messageID); err == nil && messageID > 0 {
			messageIDs = append(messageIDs, strconv.FormatInt(messageID, 10))
		}
	}

	return messageIDs, nil
}

func (c *DeltaChatChannel) resolveOutboundChatID(
	ctx context.Context,
	target string,
	outboundCtx bus.InboundContext,
	outboundScope *bus.OutboundScope,
) (int64, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return 0, fmt.Errorf("deltachat: empty chat target: %w", channels.ErrSendFailed)
	}

	if chatID, ok := parsePositiveInt64(target); ok {
		if err := c.requireOutboundNumericChatAllowed(outboundCtx, outboundScope, chatID); err != nil {
			return 0, err
		}
		return chatID, nil
	}
	if chatID, ok := parsePrefixedChatID(target); ok {
		if err := c.requireOutboundNumericChatAllowed(outboundCtx, outboundScope, chatID); err != nil {
			return 0, err
		}
		return chatID, nil
	}

	if err := c.requireOutboundRecipientResolution(outboundCtx, outboundScope); err != nil {
		return 0, err
	}

	if address, ok := emailAddressFromTarget(target); ok {
		chatID, err := c.resolveEmailChatID(ctx, address)
		if err != nil {
			return 0, fmt.Errorf("deltachat: resolve %q: %w", target, err)
		}
		return chatID, nil
	}

	chatID, err := c.resolveAliasChatID(ctx, target)
	if err != nil {
		return 0, fmt.Errorf("deltachat: resolve %q: %w", target, err)
	}
	if chatID <= 0 {
		return 0, fmt.Errorf("deltachat: unknown chat target %q: %w", target, channels.ErrSendFailed)
	}
	return chatID, nil
}

func (c *DeltaChatChannel) requireOutboundNumericChatAllowed(
	outboundCtx bus.InboundContext,
	outboundScope *bus.OutboundScope,
	chatID int64,
) error {
	callerChatID := outboundCallerChatID(outboundCtx, outboundScope)
	if callerChatID == "" || callerChatID == strconv.FormatInt(chatID, 10) {
		return nil
	}
	return c.requireOutboundRecipientResolution(outboundCtx, outboundScope)
}

func outboundCallerChatID(outboundCtx bus.InboundContext, outboundScope *bus.OutboundScope) string {
	if chatID := deltaChatChatIDFromScope(outboundScope); chatID != "" {
		return chatID
	}
	return strings.TrimSpace(outboundCtx.ChatID)
}

func deltaChatChatIDFromScope(scope *bus.OutboundScope) string {
	if scope == nil || (scope.Channel != "" && !strings.EqualFold(scope.Channel, config.ChannelDeltaChat)) {
		return ""
	}
	chat := strings.TrimSpace(scope.Values["chat"])
	if chat == "" {
		return ""
	}
	if _, value, ok := strings.Cut(chat, ":"); ok {
		chat = value
	}
	if value, _, ok := strings.Cut(chat, "/"); ok {
		chat = value
	}
	return strings.TrimSpace(chat)
}

func (c *DeltaChatChannel) requireOutboundRecipientResolution(
	outboundCtx bus.InboundContext,
	outboundScope *bus.OutboundScope,
) error {
	senderID := outboundSenderID(outboundCtx, outboundScope)
	if c.config != nil && c.config.AllowCrosspost && c.canCrosspost(senderID) {
		return nil
	}
	return fmt.Errorf(
		"deltachat: crosspost recipient resolution is disabled or caller %q is not allowed by allow_from; enable settings.allow_crosspost and allow the sender in allow_from: %w",
		senderID,
		channels.ErrSendFailed,
	)
}

func outboundSenderID(outboundCtx bus.InboundContext, outboundScope *bus.OutboundScope) string {
	if senderID := strings.TrimSpace(outboundCtx.SenderID); senderID != "" {
		return senderID
	}
	if outboundScope == nil ||
		(outboundScope.Channel != "" && !strings.EqualFold(outboundScope.Channel, config.ChannelDeltaChat)) {
		return ""
	}
	return strings.TrimSpace(outboundScope.Values["sender"])
}

func (c *DeltaChatChannel) canCrosspost(senderID string) bool {
	if c.bc == nil {
		return false
	}
	senderID = strings.TrimSpace(senderID)
	localPart, _, _ := strings.Cut(senderID, "@")
	sender := bus.SenderInfo{
		Platform:    config.ChannelDeltaChat,
		PlatformID:  senderID,
		CanonicalID: identity.BuildCanonicalID(config.ChannelDeltaChat, senderID),
		Username:    localPart,
	}
	for _, allowed := range c.bc.AllowFrom {
		entry := strings.TrimSpace(allowed)
		if entry == "*" {
			return true
		}
		if entry != "" && senderID != "" &&
			(identity.MatchAllowed(sender, entry) || strings.EqualFold(entry, senderID)) {
			return true
		}
	}
	return false
}

func parsePositiveInt64(value string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func parsePrefixedChatID(value string) (int64, bool) {
	prefix, id, ok := strings.Cut(strings.TrimSpace(value), ":")
	if !ok {
		return 0, false
	}
	switch strings.ToLower(strings.TrimSpace(prefix)) {
	case "chat", "chatid", "chat_id", config.ChannelDeltaChat:
		return parsePositiveInt64(id)
	default:
		return 0, false
	}
}

func emailAddressFromTarget(target string) (string, bool) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", false
	}
	if strings.HasPrefix(strings.ToLower(target), "mailto:") {
		target = strings.TrimSpace(target[len("mailto:"):])
		if before, _, ok := strings.Cut(target, "?"); ok {
			target = before
		}
	}
	if !strings.Contains(target, "@") {
		return "", false
	}
	if parsed, err := mail.ParseAddress(target); err == nil && parsed.Address != "" {
		return strings.ToLower(parsed.Address), true
	}
	if strings.Count(target, "@") == 1 && !strings.ContainsAny(target, " \t\r\n<>") {
		return strings.ToLower(target), true
	}
	return "", false
}

func (c *DeltaChatChannel) resolveEmailChatID(ctx context.Context, address string) (int64, error) {
	contactID, err := c.lookupContactIDByAddress(ctx, address)
	if err != nil {
		return 0, err
	}
	if contactID <= 0 {
		contactID, err = c.createContact(ctx, address, "")
		if err != nil {
			return 0, err
		}
	}
	return c.chatIDForContact(ctx, contactID)
}

func (c *DeltaChatChannel) lookupContactIDByAddress(ctx context.Context, address string) (int64, error) {
	raw, err := c.rpc.call(ctx, "lookup_contact_id_by_addr", c.accountID, address)
	if err != nil {
		return 0, fmt.Errorf("lookup contact by address: %w", err)
	}
	return decodeOptionalInt64(raw, "lookup contact by address")
}

func (c *DeltaChatChannel) createContact(ctx context.Context, address, name string) (int64, error) {
	var displayName any
	if strings.TrimSpace(name) != "" {
		displayName = strings.TrimSpace(name)
	}
	raw, err := c.rpc.call(ctx, "create_contact", c.accountID, address, displayName)
	if err != nil {
		return 0, fmt.Errorf("create contact: %w", err)
	}
	var contactID int64
	if err := json.Unmarshal(raw, &contactID); err != nil {
		return 0, fmt.Errorf("create contact decode: %w", err)
	}
	if contactID <= 0 {
		return 0, fmt.Errorf("create contact returned empty id: %w", channels.ErrSendFailed)
	}
	return contactID, nil
}

func (c *DeltaChatChannel) chatIDForContact(ctx context.Context, contactID int64) (int64, error) {
	raw, err := c.rpc.call(ctx, "get_chat_id_by_contact_id", c.accountID, contactID)
	if err != nil {
		return 0, fmt.Errorf("get chat by contact: %w", err)
	}
	chatID, err := decodeOptionalInt64(raw, "get chat by contact")
	if err != nil {
		return 0, err
	}
	if chatID > 0 {
		return chatID, nil
	}

	raw, err = c.rpc.call(ctx, "create_chat_by_contact_id", c.accountID, contactID)
	if err != nil {
		return 0, fmt.Errorf("create chat by contact: %w", err)
	}
	if err := json.Unmarshal(raw, &chatID); err != nil {
		return 0, fmt.Errorf("create chat by contact decode: %w", err)
	}
	if chatID <= 0 {
		return 0, fmt.Errorf("create chat by contact returned empty id: %w", channels.ErrSendFailed)
	}
	return chatID, nil
}

func decodeOptionalInt64(raw json.RawMessage, label string) (int64, error) {
	var id *int64
	if err := json.Unmarshal(raw, &id); err != nil {
		return 0, fmt.Errorf("%s decode: %w", label, err)
	}
	if id == nil {
		return 0, nil
	}
	return *id, nil
}

func (c *DeltaChatChannel) resolveAliasChatID(ctx context.Context, target string) (int64, error) {
	for _, query := range aliasQueries(target) {
		contacts, err := c.findMatchingContacts(ctx, query)
		if err != nil {
			return 0, err
		}
		if len(contacts) == 1 {
			return c.chatIDForContact(ctx, contacts[0].ID)
		}
		if len(contacts) > 1 {
			return 0, ambiguousRecipientError(target, contactRecipientLabels(contacts))
		}

		chats, err := c.findMatchingChats(ctx, query)
		if err != nil {
			return 0, err
		}
		if len(chats) == 1 {
			return chats[0].ID, nil
		}
		if len(chats) > 1 {
			return 0, ambiguousRecipientError(target, chatRecipientLabels(chats))
		}
	}
	return 0, nil
}

func aliasQueries(target string) []string {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	queries := []string{target}
	if unwrapped := strings.Trim(target, "<>"); unwrapped != "" && unwrapped != target {
		queries = append(queries, unwrapped)
	}
	if unprefixed := strings.TrimPrefix(target, "@"); unprefixed != "" && unprefixed != target {
		queries = append(queries, unprefixed)
	}
	return uniqueStrings(queries)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func (c *DeltaChatChannel) findMatchingContacts(ctx context.Context, query string) ([]dcContact, error) {
	raw, err := c.rpc.call(ctx, "get_contacts", c.accountID, 0, query)
	if err != nil {
		return nil, fmt.Errorf("search contacts: %w", err)
	}
	var contacts []dcContact
	if err := json.Unmarshal(raw, &contacts); err != nil {
		return nil, fmt.Errorf("search contacts decode: %w", err)
	}
	contacts = uniqueContacts(contacts)
	if exact := exactContactMatches(query, contacts); len(exact) > 0 {
		return exact, nil
	}
	if len(contacts) == 1 {
		return contacts, nil
	}
	return nil, nil
}

func uniqueContacts(contacts []dcContact) []dcContact {
	seen := make(map[int64]struct{}, len(contacts))
	out := make([]dcContact, 0, len(contacts))
	for _, contact := range contacts {
		if contact.ID <= 0 {
			continue
		}
		if _, ok := seen[contact.ID]; ok {
			continue
		}
		seen[contact.ID] = struct{}{}
		out = append(out, contact)
	}
	return out
}

func exactContactMatches(query string, contacts []dcContact) []dcContact {
	var matches []dcContact
	for _, contact := range contacts {
		if contactMatchesAlias(contact, query) {
			matches = append(matches, contact)
		}
	}
	return matches
}

func contactMatchesAlias(contact dcContact, query string) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return false
	}
	aliases := []string{
		contact.DisplayName,
		contact.Name,
		contact.Address,
		contact.NameAndAddr,
	}
	if local, _, ok := strings.Cut(contact.Address, "@"); ok {
		aliases = append(aliases, local, "@"+local)
	}
	for _, alias := range aliases {
		if strings.EqualFold(strings.TrimSpace(alias), query) {
			return true
		}
	}
	return false
}

func (c *DeltaChatChannel) findMatchingChats(ctx context.Context, query string) ([]dcChat, error) {
	raw, err := c.rpc.call(ctx, "get_chatlist_entries", c.accountID, 0, query, nil)
	if err != nil {
		return nil, fmt.Errorf("search chats: %w", err)
	}
	var chatIDs []int64
	if err := json.Unmarshal(raw, &chatIDs); err != nil {
		return nil, fmt.Errorf("search chats decode: %w", err)
	}
	var chats []dcChat
	for _, chatID := range uniqueInt64s(chatIDs) {
		if chatID <= 0 {
			continue
		}
		chat, err := c.getFullChatByContext(ctx, chatID)
		if err != nil {
			return nil, err
		}
		if chat.IsDeviceChat {
			continue
		}
		chats = append(chats, *chat)
	}
	if exact := exactChatMatches(query, chats); len(exact) > 0 {
		return exact, nil
	}
	if len(chats) == 1 {
		return chats, nil
	}
	return nil, nil
}

func uniqueInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func exactChatMatches(query string, chats []dcChat) []dcChat {
	var matches []dcChat
	for _, chat := range chats {
		if strings.EqualFold(strings.TrimSpace(chat.Name), strings.TrimSpace(query)) {
			matches = append(matches, chat)
		}
	}
	return matches
}

func (c *DeltaChatChannel) getFullChatByContext(ctx context.Context, chatID int64) (*dcChat, error) {
	raw, err := c.rpc.call(ctx, "get_full_chat_by_id", c.accountID, chatID)
	if err != nil {
		return nil, fmt.Errorf("get chat %d: %w", chatID, err)
	}
	var chat dcChat
	if err := json.Unmarshal(raw, &chat); err != nil {
		return nil, fmt.Errorf("get chat %d decode: %w", chatID, err)
	}
	return &chat, nil
}

func ambiguousRecipientError(target string, labels []string) error {
	return fmt.Errorf(
		"ambiguous recipient %q matches %s: %w",
		target,
		strings.Join(labels, ", "),
		channels.ErrSendFailed,
	)
}

func contactRecipientLabels(contacts []dcContact) []string {
	labels := make([]string, 0, len(contacts))
	for _, contact := range contacts {
		name := strings.TrimSpace(contact.DisplayName)
		if name == "" {
			name = strings.TrimSpace(contact.Name)
		}
		if name != "" && contact.Address != "" {
			labels = append(labels, fmt.Sprintf("%s <%s>", name, contact.Address))
		} else if contact.Address != "" {
			labels = append(labels, contact.Address)
		} else {
			labels = append(labels, strconv.FormatInt(contact.ID, 10))
		}
	}
	return labels
}

func chatRecipientLabels(chats []dcChat) []string {
	labels := make([]string, 0, len(chats))
	for _, chat := range chats {
		name := strings.TrimSpace(chat.Name)
		if name == "" {
			name = strconv.FormatInt(chat.ID, 10)
		}
		labels = append(labels, fmt.Sprintf("%s (chat %d)", name, chat.ID))
	}
	return labels
}

// deltaChatViewtype returns the explicit Delta Chat view type for an outbound
// media part, or "" to let Delta Chat infer it from the file. Only voice replies
// are forced (to Viewtype::Voice) so they render as playable voice bubbles;
// images, GIFs, and video keep Delta Chat's native auto-detection. A part is
// treated as voice when it is audio and either came from the send_tts tool or
// carries a "voice" filename hint (matching the convention other channels use).
func deltaChatViewtype(part bus.MediaPart, meta media.MediaMeta) string {
	isAudio := part.Type == "audio" ||
		strings.HasPrefix(strings.ToLower(part.ContentType), "audio/") ||
		strings.HasPrefix(strings.ToLower(meta.ContentType), "audio/")
	if !isAudio {
		return ""
	}

	name := strings.ToLower(part.Filename)
	if name == "" {
		name = strings.ToLower(meta.Filename)
	}
	if meta.Source == "tool:send_tts" || strings.Contains(name, "voice") {
		return "Voice"
	}
	return ""
}

// VoiceCapabilities implements channels.VoiceCapabilityProvider. Delta Chat can
// receive voice notes (which the agent's ASR transcribes) and deliver
// synthesized speech as voice messages, so it advertises both ASR and TTS. The
// gateway still gates actual availability on configured ASR/TTS providers.
func (c *DeltaChatChannel) VoiceCapabilities() channels.VoiceCapabilities {
	return channels.VoiceCapabilities{ASR: true, TTS: true}
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
	server, bootstrap, err := parseDeltaChatEmailSetting(c.config.Email)
	if err != nil {
		return err
	}
	if bootstrap {
		return c.createChatmailBootstrapAccount(ctx, server)
	}

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
		if c.config.Password.String() == "" {
			return c.passwordRequiredError("account not found")
		}

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
	} else if c.config.Password.String() != "" {
		changed, err := c.accountConfigChanged(ctx, accountID)
		if err != nil {
			logger.WarnCF("deltachat", "Could not read account config; reconfiguring", map[string]any{
				"email": c.config.Email,
				"error": err.Error(),
			})
			changed = true
		}
		if changed {
			if err := c.configureAccount(ctx, accountID); err != nil {
				return err
			}
		}
	}

	if _, err := c.rpc.call(ctx, "select_account", accountID); err != nil {
		return fmt.Errorf("deltachat select_account: %w", err)
	}
	if err := c.applyProfileConfig(ctx, accountID); err != nil {
		return err
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

func (c *DeltaChatChannel) createChatmailBootstrapAccount(ctx context.Context, server string) error {
	raw, err := c.rpc.call(ctx, "add_account")
	if err != nil {
		return fmt.Errorf("deltachat add_account: %w", err)
	}
	var accountID int64
	if decodeErr := json.Unmarshal(raw, &accountID); decodeErr != nil {
		return fmt.Errorf("deltachat add_account decode: %w", decodeErr)
	}

	created := false
	defer func() {
		if !created {
			c.cleanupPendingAccount(context.Background(), accountID)
		}
	}()

	confCtx, cancel := context.WithTimeout(ctx, configureTimeout)
	defer cancel()
	if _, callErr := c.rpc.call(
		confCtx,
		"add_transport_from_qr",
		accountID,
		buildChatmailAccountQR(server),
	); callErr != nil {
		return fmt.Errorf("deltachat create chatmail account on %s: %w", server, callErr)
	}
	created = true

	if profileErr := c.applyProfileConfig(ctx, accountID); profileErr != nil {
		logger.WarnCF(
			"deltachat",
			"Could not apply profile config to new account",
			map[string]any{"error": profileErr.Error()},
		)
	}

	addr, err := c.getAccountConfigString(ctx, accountID, "addr")
	if err != nil {
		return fmt.Errorf("deltachat created account on %s, but could not read generated email: %w", server, err)
	}
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return fmt.Errorf("deltachat created account on %s, but generated email is empty", server)
	}

	return fmt.Errorf(
		"deltachat: created chatmail account %q on %s. Update channel_list.deltachat.settings.email from %q to %q, remove the @server bootstrap marker, then run PicoClaw again to use the account",
		addr,
		server,
		c.config.Email,
		addr,
	)
}

func (c *DeltaChatChannel) cleanupPendingAccount(ctx context.Context, accountID int64) {
	if accountID <= 0 || c.rpc == nil {
		return
	}
	stopCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	_, _ = c.rpc.call(stopCtx, "stop_ongoing_process", accountID)
	cancel()

	removeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	_, _ = c.rpc.call(removeCtx, "remove_account", accountID)
	cancel()
}

func (c *DeltaChatChannel) getAccountConfigString(ctx context.Context, accountID int64, key string) (string, error) {
	raw, err := c.rpc.call(ctx, "get_config", accountID, key)
	if err != nil {
		return "", fmt.Errorf("get config %s: %w", key, err)
	}
	var value *string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("decode config %s: %w", key, err)
	}
	if value == nil {
		return "", nil
	}
	return *value, nil
}

func (c *DeltaChatChannel) passwordRequiredError(reason string) error {
	return fmt.Errorf("deltachat: account %s is not configured in data_dir %s (%s)",
		c.config.Email, c.dataDir, reason)
}

func (c *DeltaChatChannel) applyProfileConfig(ctx context.Context, accountID int64) error {
	cfgMap := map[string]*string{}
	if name := strings.TrimSpace(c.config.DisplayName); name != "" {
		cfgMap["displayname"] = accountConfigString(name)
	}
	if avatar := strings.TrimSpace(c.config.AvatarImage); avatar != "" {
		avatar = expandHome(avatar)
		if !fileExists(avatar) {
			logger.WarnCF("deltachat", "avatar_image not found; leaving current avatar unchanged", map[string]any{
				"avatar_image": avatar,
			})
		} else {
			cfgMap["selfavatar"] = accountConfigString(avatar)
		}
	}
	if len(cfgMap) == 0 {
		return nil
	}
	if _, err := c.rpc.call(ctx, "batch_set_config", accountID, cfgMap); err != nil {
		return fmt.Errorf("deltachat set profile config: %w", err)
	}
	return nil
}

// configureAccount writes the managed account settings and runs the (network-bound)
// provider auto-configuration.
func (c *DeltaChatChannel) configureAccount(ctx context.Context, accountID int64) error {
	if c.config.Password.String() == "" {
		return c.passwordRequiredError("account is not configured")
	}

	cfgMap := accountConfigMap(c.config)
	if _, err := c.rpc.call(ctx, "batch_set_config", accountID, cfgMap); err != nil {
		return fmt.Errorf("deltachat set account config: %w", err)
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

func (c *DeltaChatChannel) accountConfigChanged(ctx context.Context, accountID int64) (bool, error) {
	want := accountConfigMap(c.config)
	for _, key := range managedAccountConfigKeys {
		raw, err := c.rpc.call(ctx, "get_config", accountID, key)
		if err != nil {
			return false, fmt.Errorf("deltachat get config %s: %w", key, err)
		}
		var got *string
		if err := json.Unmarshal(raw, &got); err != nil {
			return false, fmt.Errorf("deltachat get config %s decode: %w", key, err)
		}
		if !accountConfigValueEqual(got, want[key]) {
			logger.InfoCF("deltachat", "Account config changed; reconfiguring", map[string]any{
				"email": c.config.Email,
				"key":   key,
			})
			return true, nil
		}
	}
	return false, nil
}

func accountConfigValueEqual(got, want *string) bool {
	if want == nil {
		return got == nil || *got == ""
	}
	if got == nil {
		return *want == ""
	}
	return *got == *want
}

func accountConfigMap(cfg *config.DeltaChatSettings) map[string]*string {
	cfgMap := map[string]*string{
		"addr":        accountConfigString(cfg.Email),
		"mail_server": accountConfigOptionalString(cfg.IMAPServer),
		"mail_port":   accountConfigOptionalInt(cfg.IMAPPort),
		"send_server": accountConfigOptionalString(cfg.SMTPServer),
		"send_port":   accountConfigOptionalInt(cfg.SMTPPort),
	}
	if password := cfg.Password.String(); password != "" {
		cfgMap["mail_pw"] = accountConfigString(password)
	}
	return cfgMap
}

func accountConfigString(value string) *string {
	return &value
}

func accountConfigOptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return accountConfigString(value)
}

func accountConfigOptionalInt(value int) *string {
	if value <= 0 {
		return nil
	}
	return accountConfigString(strconv.Itoa(value))
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

// resolveServerPath validates the configured deltachat-rpc-server path, or
// falls back to deltachat-rpc-server on PATH.
func resolveServerPath(configured string) (string, error) {
	if configured == "" {
		p, err := exec.LookPath("deltachat-rpc-server")
		if err != nil {
			return "", fmt.Errorf("deltachat: deltachat-rpc-server not found on PATH " +
				"(set rpc_server_path to the binary path if it is installed elsewhere)")
		}
		return p, nil
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
