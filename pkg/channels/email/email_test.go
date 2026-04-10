package email

import (
	"strings"
	"testing"

	imap "github.com/emersion/go-imap/v2"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewEmailChannel(t *testing.T) {
	msgBus := bus.NewMessageBus()

	validCfg := config.EmailConfig{
		SMTPHost: "smtp.example.com",
		SMTPFrom: *config.NewSecureString("bot@example.com"),
		IMAPHost: "imap.example.com",
		IMAPUser: *config.NewSecureString("bot@example.com"),
	}

	t.Run("missing smtp_host", func(t *testing.T) {
		cfg := validCfg
		cfg.SMTPHost = ""
		_, err := NewEmailChannel(cfg, msgBus)
		if err == nil {
			t.Error("expected error for missing smtp_host, got nil")
		}
	})

	t.Run("missing smtp_from", func(t *testing.T) {
		cfg := validCfg
		cfg.SMTPFrom = *config.NewSecureString("")
		_, err := NewEmailChannel(cfg, msgBus)
		if err == nil {
			t.Error("expected error for missing smtp_from, got nil")
		}
	})

	t.Run("missing imap_host", func(t *testing.T) {
		cfg := validCfg
		cfg.IMAPHost = ""
		_, err := NewEmailChannel(cfg, msgBus)
		if err == nil {
			t.Error("expected error for missing imap_host, got nil")
		}
	})

	t.Run("missing imap_user", func(t *testing.T) {
		cfg := validCfg
		cfg.IMAPUser = *config.NewSecureString("")
		_, err := NewEmailChannel(cfg, msgBus)
		if err == nil {
			t.Error("expected error for missing imap_user, got nil")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		ch, err := NewEmailChannel(validCfg, msgBus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name() != "email" {
			t.Errorf("Name() = %q, want %q", ch.Name(), "email")
		}
		if ch.IsRunning() {
			t.Error("new channel should not be running")
		}
	})
}

func TestExtractFrom(t *testing.T) {
	tests := []struct {
		name string
		from []imap.Address
		want string
	}{
		{
			name: "full address",
			from: []imap.Address{{Mailbox: "user", Host: "example.com"}},
			want: "user@example.com",
		},
		{
			name: "no host",
			from: []imap.Address{{Mailbox: "local"}},
			want: "local",
		},
		{
			name: "empty from list",
			from: []imap.Address{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &imap.Envelope{From: tt.from}
			got := extractFrom(env)
			if got != tt.want {
				t.Errorf("extractFrom() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDisplayName(t *testing.T) {
	tests := []struct {
		name string
		from []imap.Address
		want string
	}{
		{
			name: "address with display name",
			from: []imap.Address{{Name: "Alice", Mailbox: "alice", Host: "example.com"}},
			want: "Alice",
		},
		{
			name: "address without display name",
			from: []imap.Address{{Mailbox: "alice", Host: "example.com"}},
			want: "alice@example.com",
		},
		{
			name: "empty from list",
			from: []imap.Address{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &imap.Envelope{From: tt.from}
			got := displayName(env)
			if got != tt.want {
				t.Errorf("displayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractPlainText(t *testing.T) {
	plainMIME := "MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nHello, world!"
	htmlMIME := "MIME-Version: 1.0\r\nContent-Type: text/html; charset=utf-8\r\n\r\n<html><body>Hi</body></html>"
	multipartMIME := strings.Join([]string{
		"MIME-Version: 1.0",
		`Content-Type: multipart/alternative; boundary="boundary"`,
		"",
		"--boundary",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"Plain text part",
		"--boundary",
		"Content-Type: text/html; charset=utf-8",
		"",
		"<html><body>HTML part</body></html>",
		"--boundary--",
	}, "\r\n")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "text/plain MIME",
			input: plainMIME,
			want:  "Hello, world!",
		},
		{
			name:  "html-only MIME",
			input: htmlMIME,
			want:  "",
		},
		{
			name:  "multipart with text/plain",
			input: multipartMIME,
			want:  "Plain text part",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPlainText(strings.NewReader(tt.input))
			if got != tt.want {
				t.Errorf("extractPlainText() = %q, want %q", got, tt.want)
			}
		})
	}
}
