package channels

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type EmailChannel struct {
	*BaseChannel
	config      config.EmailConfig
	bus         *bus.MessageBus
	imapClients map[string]*client.Client // Map email address -> client
	stopChan    chan struct{}
	manualCheck chan bool
	mu          sync.Mutex
}

func NewEmailChannel(cfg config.EmailConfig, bus *bus.MessageBus) *EmailChannel {
	// If Accounts is empty but single fields are set, populate Accounts with one entry
	if len(cfg.Accounts) == 0 && cfg.IMAPServer != "" {
		cfg.Accounts = []config.EmailAccountConfig{{
			Email:        cfg.IMAPUser, // Default to IMAP user as email
			IMAPServer:   cfg.IMAPServer,
			IMAPPort:     cfg.IMAPPort,
			IMAPUser:     cfg.IMAPUser,
			IMAPPassword: cfg.IMAPPassword,
			SMTPServer:   cfg.SMTPServer,
			SMTPPort:     cfg.SMTPPort,
			SMTPUser:     cfg.SMTPUser,
			SMTPPassword: cfg.SMTPPassword,
		}}
	}

	base := NewBaseChannel("email", cfg, bus, cfg.AllowFrom)

	return &EmailChannel{
		BaseChannel: base,
		config:      cfg,
		bus:         bus,
		imapClients: make(map[string]*client.Client),
		stopChan:    make(chan struct{}),
		manualCheck: make(chan bool, 1),
	}
}

func (c *EmailChannel) Start(ctx context.Context) error {
	logger.InfoC("email", "Starting Email channel polling...")

	// Initial connection
	c.connectAllIMAP()

	c.setRunning(true)
	go c.pollLoop(ctx)
	return nil
}

func (c *EmailChannel) connectAllIMAP() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, acc := range c.config.Accounts {
		if acc.Email == "" {
			continue // Skip invalid config
		}

		// Skip if already connected
		if client, ok := c.imapClients[acc.Email]; ok && client.State() == imap.AuthenticatedState {
			continue
		}

		logger.DebugCF("email", "Connecting to IMAP", map[string]interface{}{"email": acc.Email, "server": acc.IMAPServer})
		client, err := c.connectIMAPAccount(acc)
		if err != nil {
			logger.ErrorCF("email", "Failed to connect to IMAP account", map[string]interface{}{
				"email": acc.Email,
				"error": err.Error(),
			})
			continue
		}
		c.imapClients[acc.Email] = client
		logger.InfoCF("email", "Connected to IMAP account", map[string]interface{}{"email": acc.Email})
	}
}

func (c *EmailChannel) connectIMAPAccount(acc config.EmailAccountConfig) (*client.Client, error) {
	addr := fmt.Sprintf("%s:%d", acc.IMAPServer, acc.IMAPPort)
	var cClient *client.Client
	var err error

	if acc.IMAPPort == 993 {
		cClient, err = client.DialTLS(addr, nil)
	} else {
		cClient, err = client.Dial(addr)
	}
	if err != nil {
		return nil, err
	}

	if err := cClient.Login(acc.IMAPUser, acc.IMAPPassword); err != nil {
		cClient.Logout()
		return nil, err
	}

	return cClient, nil
}

func (c *EmailChannel) Stop(ctx context.Context) error {
	logger.InfoC("email", "Stopping Email channel...")
	c.setRunning(false)
	close(c.stopChan)

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, client := range c.imapClients {
		client.Logout()
	}
	return nil
}

func (c *EmailChannel) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(c.config.PollInterval) * time.Second)
	defer ticker.Stop()

	// Initial check
	c.checkAllMail()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.checkAllMail()
		case <-c.manualCheck:
			c.checkAllMail()
		}
	}
}

func (c *EmailChannel) CheckNow() {
	select {
	case c.manualCheck <- true:
		logger.InfoC("email", "Manual check triggered")
	default:
		logger.DebugC("email", "Manual check already queued, skipping")
	}
}

func (c *EmailChannel) checkAllMail() {
	fmt.Println("📧 Checking for new emails on all accounts...")

	// Ensure connections
	c.connectAllIMAP()

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, acc := range c.config.Accounts {
		client, ok := c.imapClients[acc.Email]
		if !ok || client.State() == imap.LogoutState {
			continue
		}
		c.checkAccountMail(client, acc)
	}
}

func (c *EmailChannel) checkAccountMail(imapClient *client.Client, acc config.EmailAccountConfig) {
	// Select INBOX
	mbox, err := imapClient.Select("INBOX", false)
	if err != nil {
		logger.ErrorCF("email", "Failed to select INBOX", map[string]interface{}{"email": acc.Email, "error": err.Error()})
		// Invalidate connection
		imapClient.Logout()
		return
	}

	if mbox.Messages == 0 {
		return
	}

	// Search for unread messages
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	uids, err := imapClient.Search(criteria)
	if err != nil {
		logger.ErrorCF("email", "Failed to search emails", map[string]interface{}{"email": acc.Email, "error": err.Error()})
		return
	}

	if len(uids) == 0 {
		return
	}

	// Limit to last 10 emails
	const maxEmails = 10
	if len(uids) > maxEmails {
		// Take the newest ones (highest UIDs)
		uids = uids[len(uids)-maxEmails:]
	}

	fmt.Printf("📧 [%s] Found %d unread emails, fetching details...\n", acc.Email, len(uids))

	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	// Fetch envelope and body structure to check dates first?
	// Actually just fetch everything for the small batch.
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid}

	messages := make(chan *imap.Message)
	done := make(chan error, 1)

	go func() {
		done <- imapClient.Fetch(seqset, items, messages)
	}()

	for msg := range messages {
		// Time filter: Ignore emails older than 1 hour to prevent backlog flood on restart
		if msg.Envelope != nil && time.Since(msg.Envelope.Date) > 1*time.Hour {
			logger.DebugCF("email", "Skipping old unread email", map[string]interface{}{
				"subject": msg.Envelope.Subject,
				"date":    msg.Envelope.Date,
			})
			continue
		}

		c.processMessage(msg, section, acc.Email) // Pass account email to know recipient context

		// Mark as seen
		item := imap.FormatFlagsOp(imap.AddFlags, true)
		flags := []interface{}{imap.SeenFlag}
		seq := new(imap.SeqSet)
		seq.AddNum(msg.Uid)
		if err := imapClient.Store(seq, item, flags, nil); err != nil {
			logger.ErrorCF("email", "Failed to mark email as seen", map[string]interface{}{"uid": msg.Uid})
		}
	}

	if err := <-done; err != nil {
		logger.ErrorCF("email", "Fetch failed", map[string]interface{}{"error": err.Error()})
	}
}

func (c *EmailChannel) processMessage(msg *imap.Message, section *imap.BodySectionName, accountEmail string) {
	if msg == nil || msg.Envelope == nil {
		return
	}

	subject := msg.Envelope.Subject
	from := msg.Envelope.From

	if len(from) == 0 {
		return
	}

	sender := fmt.Sprintf("%s@%s", from[0].MailboxName, from[0].HostName)

	// Check allowlist
	if !c.IsAllowed(sender) {
		logger.DebugCF("email", "Ignoring email from non-allowed sender", map[string]interface{}{"sender": sender})
		return
	}

	// Get body
	r := msg.GetBody(section)
	if r == nil {
		return
	}

	mr, err := mail.CreateReader(r)
	if err != nil {
		logger.ErrorCF("email", "Failed to create mail reader", map[string]interface{}{"error": err.Error()})
		return
	}

	var body string
	// Simple body extraction
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			break
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			contentType, _, _ := h.ContentType()
			if contentType == "text/plain" {
				b, _ := io.ReadAll(p.Body)
				body = string(b)
			} else if contentType == "text/html" && body == "" {
				// Fallback to HTML if no text/plain yet, ideally strip tags
				b, _ := io.ReadAll(p.Body)
				body = string(b) // TODO: Strip HTML
			}
		}
	}

	// ChatID logic:
	// For now, we treat the sender as the ChatID.
	// To distinguish which account received it, we could prepend/append, but let's keep it simple.
	// The User ID is the sender.
	chatID := sender

	// Forwarding logic
	if c.config.ForwardTo != "" {
		parts := strings.SplitN(c.config.ForwardTo, ":", 2)
		if len(parts) == 2 {
			channel, targetChatID := parts[0], parts[1]
			forwardContent := fmt.Sprintf("📧 **New Email [%s]**\n**From:** %s\n**Subject:** %s\n\n%s", accountEmail, sender, subject, body)

			// Truncate body
			if len(body) > 500 {
				forwardContent = fmt.Sprintf("📧 **New Email [%s]**\n**From:** %s\n**Subject:** %s\n\n%s...", accountEmail, sender, subject, body[:500])
			}

			c.bus.PublishOutbound(bus.OutboundMessage{
				Channel: channel,
				ChatID:  targetChatID,
				Content: forwardContent,
			})
			fmt.Printf("➡️ Forwarded email to %s:%s\n", channel, targetChatID)
		}
	}

	c.HandleMessage(sender, chatID, contentWithContext(accountEmail, subject, body), nil, map[string]string{
		"subject": subject,
		"email":   sender,
		"to":      accountEmail,
	})
	fmt.Printf("✅ Processed email from %s to %s: %s\n", sender, accountEmail, subject)
}

func contentWithContext(account, subject, body string) string {
	return fmt.Sprintf("[Received at %s]\nSubject: %s\n\n%s", account, subject, body)
}

func (c *EmailChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	logger.DebugCF("email", "Send received", map[string]interface{}{"content": msg.Content, "chat_id": msg.ChatID})

	// Check for command
	if strings.TrimSpace(msg.Content) == "CMD:CHECK" {
		c.CheckNow()
		return nil
	}

	// For simple replies, we don't know which account to send FROM unless we track state or infer.
	// We'll Default to the first account, or try to find one.
	// Ideally, the tool `send_email` should be used which calls this.
	// If this is a direct reply from Agent, it might lack context.
	// However, we can use the first account as default.

	account := c.config.Accounts[0]

	// If msg.Metadata has "from", use it
	// But OutboundMessage doesn't have arbitrary metadata map on struct usually?
	// It does NOT.
	// So we use default account for general replies.

	addr := fmt.Sprintf("%s:%d", account.SMTPServer, account.SMTPPort)
	auth := smtp.PlainAuth("", account.SMTPUser, account.SMTPPassword, account.SMTPServer)

	to := []string{msg.ChatID}
	subject := "Re: PicoClaw Response"
	body := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", msg.ChatID, subject, msg.Content)

	if account.SMTPPort == 465 {
		tlsConfig := &tls.Config{ServerName: account.SMTPServer}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return err
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, account.SMTPServer)
		if err != nil {
			return err
		}
		defer client.Quit()

		if err = client.Auth(auth); err != nil {
			return err
		}
		if err = client.Mail(account.SMTPUser); err != nil {
			return err
		}
		if err = client.Rcpt(to[0]); err != nil {
			return err
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(body))
		if err != nil {
			return err
		}
		return w.Close()
	} else {
		return smtp.SendMail(addr, auth, account.SMTPUser, to, []byte(body))
	}
}
