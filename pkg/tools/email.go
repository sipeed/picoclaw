package tools

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/smtp"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// ReadEmailTool fetches recent emails and returns them as text.
type ReadEmailTool struct {
	config config.EmailConfig
}

func NewReadEmailTool(cfg config.EmailConfig) *ReadEmailTool {
	return &ReadEmailTool{config: cfg}
}

func (t *ReadEmailTool) Name() string {
	return "read_email"
}

func (t *ReadEmailTool) Description() string {
	return "Fetch and read recent emails from the inbox. Use this when the user asks to 'check mail', 'read email', or sees a notification. Returns the actual email content."
}

func (t *ReadEmailTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"count": map[string]interface{}{
				"type":        "integer",
				"description": "Number of recent emails to fetch (default: 5, max: 10)",
			},
			"unread_only": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, only fetch unread emails (default: false)",
			},
			"account": map[string]interface{}{
				"type":        "string",
				"description": "Optional: Specific email account to check (e.g. 'user@gmail.com'). If omitted, checks all.",
			},
		},
	}
}

func (t *ReadEmailTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if !t.config.Enabled {
		return ErrorResult("Email channel is not enabled in configuration.")
	}

	count := 5
	if c, ok := args["count"].(float64); ok {
		count = int(c)
	}
	if count > 10 {
		count = 10
	}

	unreadOnly := false
	if u, ok := args["unread_only"].(bool); ok {
		unreadOnly = u
	}

	targetAccount, _ := args["account"].(string)

	var sb strings.Builder
	totalFound := 0

	// Helper function to check one account
	checkAccount := func(acc config.EmailAccountConfig) error {
		addr := fmt.Sprintf("%s:%d", acc.IMAPServer, acc.IMAPPort)
		var c *client.Client
		var err error

		if acc.IMAPPort == 993 {
			c, err = client.DialTLS(addr, nil)
		} else {
			c, err = client.Dial(addr)
		}
		if err != nil {
			return fmt.Errorf("failed to connect to %s: %v", acc.Email, err)
		}
		defer c.Logout()

		if err := c.Login(acc.IMAPUser, acc.IMAPPassword); err != nil {
			return fmt.Errorf("failed to login to %s: %v", acc.Email, err)
		}

		mbox, err := c.Select("INBOX", false)
		if err != nil {
			return fmt.Errorf("failed to select INBOX for %s: %v", acc.Email, err)
		}

		if mbox.Messages == 0 {
			return nil
		}

		// Search
		criteria := imap.NewSearchCriteria()
		if unreadOnly {
			criteria.WithoutFlags = []string{imap.SeenFlag}
		}

		var uids []uint32
		if unreadOnly {
			uids, err = c.Search(criteria)
			if err != nil {
				return fmt.Errorf("failed to search %s: %v", acc.Email, err)
			}
		}

		seqset := new(imap.SeqSet)
		if unreadOnly {
			if len(uids) == 0 {
				return nil
			}
			if len(uids) > count {
				uids = uids[len(uids)-count:]
			}
			seqset.AddNum(uids...)
		} else {
			from := uint32(1)
			if mbox.Messages > uint32(count) {
				from = mbox.Messages - uint32(count) + 1
			}
			to := mbox.Messages
			seqset.AddRange(from, to)
		}

		section := &imap.BodySectionName{}
		items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope}

		messages := make(chan *imap.Message)
		done := make(chan error, 1)

		go func() {
			done <- c.Fetch(seqset, items, messages)
		}()

		accountData := fmt.Sprintf("\n📧 Account: %s\n", acc.Email)
		hasEmails := false

		for msg := range messages {
			hasEmails = true
			accountData += "---\n"
			sender := "unknown"
			if len(msg.Envelope.From) > 0 {
				sender = fmt.Sprintf("%s@%s", msg.Envelope.From[0].MailboxName, msg.Envelope.From[0].HostName)
			}
			accountData += fmt.Sprintf("From: %s\n", sender)
			accountData += fmt.Sprintf("Subject: %s\n", msg.Envelope.Subject)
			accountData += fmt.Sprintf("Date: %s\n", msg.Envelope.Date)

			r := msg.GetBody(section)
			if r != nil {
				mr, err := mail.CreateReader(r)
				if err == nil {
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
								accountData += fmt.Sprintf("\n%s\n", string(b))
							}
						}
					}
				}
			}
		}

		if err := <-done; err != nil {
			return fmt.Errorf("fetch failed for %s: %v", acc.Email, err)
		}

		if hasEmails {
			sb.WriteString(accountData)
			totalFound++
		}
		return nil
	}

	// Iterate over accounts
	accountsToCheck := t.config.Accounts
	// Fallback to legacy single account if Accounts is empty
	if len(accountsToCheck) == 0 && t.config.IMAPServer != "" {
		accountsToCheck = []config.EmailAccountConfig{{
			Email:        t.config.IMAPUser,
			IMAPServer:   t.config.IMAPServer,
			IMAPPort:     t.config.IMAPPort,
			IMAPUser:     t.config.IMAPUser,
			IMAPPassword: t.config.IMAPPassword,
		}}
	}

	for _, acc := range accountsToCheck {
		if targetAccount != "" && !strings.EqualFold(acc.Email, targetAccount) {
			continue
		}
		if err := checkAccount(acc); err != nil {
			sb.WriteString(fmt.Sprintf("\n❌ Error fetching from %s: %v\n", acc.Email, err))
		}
	}

	if sb.Len() == 0 {
		return SilentResult("No recent emails found.")
	}

	return &ToolResult{
		ForLLM:  sb.String(),
		ForUser: fmt.Sprintf("Checked emails for %d accounts.", len(accountsToCheck)),
	}
}

// SendEmailTool sends an email.
type SendEmailTool struct {
	config config.EmailConfig
}

func NewSendEmailTool(cfg config.EmailConfig) *SendEmailTool {
	return &SendEmailTool{config: cfg}
}

func (t *SendEmailTool) Name() string {
	return "send_email"
}

func (t *SendEmailTool) Description() string {
	return "Send an email to a specific address."
}

func (t *SendEmailTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"to": map[string]interface{}{
				"type":        "string",
				"description": "Recipient email address (e.g. user@example.com)",
			},
			"subject": map[string]interface{}{
				"type":        "string",
				"description": "Email subject",
			},
			"body": map[string]interface{}{
				"type":        "string",
				"description": "Email body content",
			},
			"from_account": map[string]interface{}{
				"type":        "string",
				"description": "Optional: Email address to send FROM. Must match a configured account.",
			},
		},
		"required": []string{"to", "subject", "body"},
	}
}

func (t *SendEmailTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if !t.config.Enabled {
		return ErrorResult("Email channel is not enabled in configuration.")
	}

	to, _ := args["to"].(string)
	subject, _ := args["subject"].(string)
	bodyContent, _ := args["body"].(string)
	fromAccount, _ := args["from_account"].(string)

	if to == "" {
		return ErrorResult("Recipient (to) is required.")
	}

	// Select account
	var account config.EmailAccountConfig
	found := false

	// Fallback legacy
	if len(t.config.Accounts) == 0 && t.config.SMTPServer != "" {
		account = config.EmailAccountConfig{
			Email:        t.config.IMAPUser, // best guess
			SMTPServer:   t.config.SMTPServer,
			SMTPPort:     t.config.SMTPPort,
			SMTPUser:     t.config.SMTPUser,
			SMTPPassword: t.config.SMTPPassword,
		}
		found = true
	} else {
		// Try to match requested account
		if fromAccount != "" {
			for _, acc := range t.config.Accounts {
				if strings.EqualFold(acc.Email, fromAccount) {
					account = acc
					found = true
					break
				}
			}
			if !found {
				return ErrorResult(fmt.Sprintf("Configured account not found for email: %s", fromAccount))
			}
		} else {
			// Default to first account
			if len(t.config.Accounts) > 0 {
				account = t.config.Accounts[0]
				found = true
			}
		}
	}

	if !found {
		return ErrorResult("No valid email account configuration found.")
	}

	addr := fmt.Sprintf("%s:%d", account.SMTPServer, account.SMTPPort)
	auth := smtp.PlainAuth("", account.SMTPUser, account.SMTPPassword, account.SMTPServer)

	// RFC 822 format
	msg := fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", to, subject, bodyContent)

	logger.InfoCF("email", "Sending email via tool", map[string]interface{}{
		"to":   to,
		"from": account.Email,
	})

	err := t.sendMail(addr, auth, account.SMTPUser, []string{to}, []byte(msg), account)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to send email: %v", err))
	}

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Email sent successfully to %s using account %s", to, account.Email),
		ForUser: fmt.Sprintf("Sent email to %s (via %s)", to, account.Email),
	}
}

func (t *SendEmailTool) sendMail(addr string, a smtp.Auth, from string, to []string, msg []byte, acc config.EmailAccountConfig) error {
	// Handle TLS logic
	if acc.SMTPPort == 465 {
		// Direct TLS
		tlsConfig := &tls.Config{
			ServerName: acc.SMTPServer,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return err
		}
		defer conn.Close()

		c, err := smtp.NewClient(conn, acc.SMTPServer)
		if err != nil {
			return err
		}
		defer c.Quit()

		if err = c.Auth(a); err != nil {
			return err
		}
		if err = c.Mail(from); err != nil {
			return err
		}
		for _, addr := range to {
			if err = c.Rcpt(addr); err != nil {
				return err
			}
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(msg)
		if err != nil {
			return err
		}
		return w.Close()
	} else {
		// STARTTLS
		return smtp.SendMail(addr, a, from, to, msg)
	}
}
