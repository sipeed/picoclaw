package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/smtp"
	"strings"
	"time"

	imap "github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	gomail "github.com/emersion/go-message/mail"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// EmailChannel implements the Channel interface using SMTP (outbound) and IMAP polling (inbound).
type EmailChannel struct {
	*channels.BaseChannel
	config config.EmailConfig
	ctx    context.Context
	cancel context.CancelFunc
}

// NewEmailChannel creates a new email channel.
func NewEmailChannel(cfg config.EmailConfig, messageBus *bus.MessageBus) (*EmailChannel, error) {
	if cfg.SMTPHost == "" {
		return nil, fmt.Errorf("email smtp_host is required")
	}
	if cfg.SMTPFrom.String() == "" {
		return nil, fmt.Errorf("email smtp_from is required")
	}
	if cfg.IMAPHost == "" {
		return nil, fmt.Errorf("email imap_host is required")
	}
	if cfg.IMAPUser.String() == "" {
		return nil, fmt.Errorf("email imap_user is required")
	}

	base := channels.NewBaseChannel("email", cfg, messageBus, cfg.AllowFrom,
		channels.WithReasoningChannelID(cfg.ReasoningChannelID),
	)

	return &EmailChannel{
		BaseChannel: base,
		config:      cfg,
	}, nil
}

// Start begins IMAP polling.
func (c *EmailChannel) Start(ctx context.Context) error {
	logger.InfoC("email", "Starting email channel")
	c.ctx, c.cancel = context.WithCancel(ctx)

	interval := c.config.PollIntervalSecs
	if interval <= 0 {
		interval = 30
	}

	go c.pollLoop(time.Duration(interval) * time.Second)

	c.SetRunning(true)
	logger.InfoCF("email", "Email channel started", map[string]any{
		"smtp_host": c.config.SMTPHost,
		"imap_host": c.config.IMAPHost,
		"interval":  interval,
	})
	return nil
}

// Stop cancels the polling loop.
func (c *EmailChannel) Stop(ctx context.Context) error {
	logger.InfoC("email", "Stopping email channel")
	c.SetRunning(false)
	if c.cancel != nil {
		c.cancel()
	}
	logger.InfoC("email", "Email channel stopped")
	return nil
}

// Send delivers an outbound message via SMTP.
// msg.ChatID is the recipient email address.
func (c *EmailChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	to := msg.ChatID
	if to == "" {
		return nil, fmt.Errorf("chat ID (recipient address) is empty: %w", channels.ErrSendFailed)
	}
	if strings.TrimSpace(msg.Content) == "" {
		return nil, nil
	}

	subject := c.config.DefaultSubject
	if subject == "" {
		subject = "Message"
	}

	smtpPort := c.config.SMTPPort
	if smtpPort == 0 {
		smtpPort = 587
	}

	addr := fmt.Sprintf("%s:%d", c.config.SMTPHost, smtpPort)
	smtpUser := c.config.SMTPUser.String()
	if smtpUser == "" {
		smtpUser = c.config.SMTPFrom.String()
	}

	body := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		c.config.SMTPFrom.String(), to, subject, msg.Content)

	var auth smtp.Auth
	if c.config.SMTPPassword.String() != "" {
		auth = smtp.PlainAuth("", smtpUser, c.config.SMTPPassword.String(), c.config.SMTPHost)
	}

	// Port 465 uses implicit TLS; port 587 and others use STARTTLS.
	if smtpPort == 465 {
		tlsCfg := &tls.Config{ServerName: c.config.SMTPHost}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return nil, fmt.Errorf("smtp tls dial: %w", channels.ErrTemporary)
		}
		client, err := smtp.NewClient(conn, c.config.SMTPHost)
		if err != nil {
			return nil, fmt.Errorf("smtp new client: %w", channels.ErrTemporary)
		}
		defer client.Close()
		if err := sendViaSMTPClient(client, auth, c.config.SMTPFrom.String(), to, []byte(body)); err != nil {
			return nil, err
		}
	} else {
		if err := smtp.SendMail(addr, auth, c.config.SMTPFrom.String(), []string{to}, []byte(body)); err != nil {
			return nil, fmt.Errorf("smtp send: %w: %w", err, channels.ErrTemporary)
		}
	}

	logger.DebugCF("email", "Message sent", map[string]any{"to": to})
	return nil, nil
}

func sendViaSMTPClient(client *smtp.Client, auth smtp.Auth, from, to string, body []byte) error {
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w: %w", err, channels.ErrSendFailed)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w: %w", err, channels.ErrSendFailed)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w: %w", err, channels.ErrSendFailed)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w: %w", err, channels.ErrSendFailed)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("smtp write body: %w: %w", err, channels.ErrSendFailed)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w: %w", err, channels.ErrSendFailed)
	}
	return client.Quit()
}

func (c *EmailChannel) pollLoop(interval time.Duration) {
	// Poll once immediately on start, then on ticker.
	c.pollIMAP()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.pollIMAP()
		}
	}
}

func (c *EmailChannel) pollIMAP() {
	imapPort := c.config.IMAPPort
	if imapPort == 0 {
		imapPort = 993
	}
	addr := fmt.Sprintf("%s:%d", c.config.IMAPHost, imapPort)

	var (
		client *imapclient.Client
		err    error
	)

	// Port 993 = implicit TLS, anything else = plain (STARTTLS not yet supported).
	if imapPort == 993 {
		tlsCfg := &tls.Config{ServerName: c.config.IMAPHost}
		client, err = imapclient.DialTLS(addr, &imapclient.Options{TLSConfig: tlsCfg})
	} else {
		client, err = imapclient.DialInsecure(addr, nil)
	}
	if err != nil {
		logger.WarnCF("email", "IMAP dial failed", map[string]any{"err": err})
		return
	}
	defer client.Close()

	if err = client.Login(c.config.IMAPUser.String(), c.config.IMAPPassword.String()).Wait(); err != nil {
		logger.WarnCF("email", "IMAP login failed", map[string]any{"err": err})
		return
	}

	if _, err = client.Select("INBOX", nil).Wait(); err != nil {
		logger.WarnCF("email", "IMAP SELECT INBOX failed", map[string]any{"err": err})
		return
	}

	searchData, err := client.Search(&imap.SearchCriteria{
		NotFlag: []imap.Flag{imap.FlagSeen},
	}, nil).Wait()
	if err != nil {
		logger.WarnCF("email", "IMAP SEARCH failed", map[string]any{"err": err})
		return
	}

	if len(searchData.AllSeqNums()) == 0 {
		return
	}

	seqSet := imap.SeqSetNum(searchData.AllSeqNums()...)
	bodySection := &imap.FetchItemBodySection{}
	fetchOptions := &imap.FetchOptions{
		Envelope:    true,
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}

	fetchCmd := client.Fetch(seqSet, fetchOptions)

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		var (
			envelope        *imap.Envelope
			bodySectionData *imapclient.FetchItemDataBodySection
			seqNum          uint32
		)

		seqNum = msg.SeqNum

		for {
			item := msg.Next()
			if item == nil {
				break
			}
			switch v := item.(type) {
			case imapclient.FetchItemDataEnvelope:
				envelope = v.Envelope
			case imapclient.FetchItemDataBodySection:
				bodySectionData = &v
			}
		}

		if envelope == nil || bodySectionData == nil {
			continue
		}

		fromAddr := extractFrom(envelope)
		if fromAddr == "" {
			continue
		}

		messageID := envelope.MessageID
		plainText := extractPlainText(bodySectionData.Literal)

		sender := bus.SenderInfo{
			Platform:    "email",
			PlatformID:  fromAddr,
			CanonicalID: "email:" + fromAddr,
			DisplayName: displayName(envelope),
		}

		c.HandleMessage(c.ctx,
			bus.Peer{Kind: "direct", ID: fromAddr},
			messageID, fromAddr, fromAddr, plainText,
			nil, nil,
			sender,
		)

		// Mark message as \Seen
		storeSeq := imap.SeqSetNum(seqNum)
		storeFlags := imap.StoreFlags{
			Op:     imap.StoreFlagsAdd,
			Flags:  []imap.Flag{imap.FlagSeen},
			Silent: true,
		}
		if err := client.Store(storeSeq, &storeFlags, nil).Close(); err != nil {
			logger.WarnCF("email", "IMAP STORE \\Seen failed", map[string]any{"err": err, "seq": seqNum})
		}
	}

	if err := fetchCmd.Close(); err != nil {
		logger.WarnCF("email", "IMAP FETCH close error", map[string]any{"err": err})
	}
}

func extractFrom(env *imap.Envelope) string {
	if len(env.From) == 0 {
		return ""
	}
	addr := env.From[0]
	if addr.Host == "" {
		return addr.Mailbox
	}
	return addr.Mailbox + "@" + addr.Host
}

func displayName(env *imap.Envelope) string {
	if len(env.From) == 0 {
		return ""
	}
	if env.From[0].Name != "" {
		return env.From[0].Name
	}
	return extractFrom(env)
}

// extractPlainText reads the message body and returns the first text/plain part.
// Falls back to the raw body if parsing fails.
func extractPlainText(r io.Reader) string {
	mr, err := gomail.CreateReader(r)
	if err != nil {
		// Fallback: read raw bytes
		b, _ := io.ReadAll(r)
		return strings.TrimSpace(string(b))
	}

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		inlineHeader, ok := p.Header.(*gomail.InlineHeader)
		if !ok {
			continue
		}
		ct, _, _ := inlineHeader.ContentType()
		if ct == "text/plain" || ct == "" {
			b, _ := io.ReadAll(p.Body)
			text := strings.TrimSpace(string(b))
			if text != "" {
				return text
			}
		}
	}
	return ""
}
