package channels

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNormalizeTelegramAPIBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "empty value is allowed",
			input: "",
			want:  "",
		},
		{
			name:  "trim spaces and trailing slash",
			input: "  https://telegram-proxy.example.com/custom/  ",
			want:  "https://telegram-proxy.example.com/custom",
		},
		{
			name:    "missing scheme",
			input:   "telegram-proxy.example.com",
			wantErr: true,
		},
		{
			name:    "unsupported scheme",
			input:   "ftp://telegram-proxy.example.com",
			wantErr: true,
		},
		{
			name:    "query not allowed",
			input:   "https://telegram-proxy.example.com/api?x=1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeTelegramAPIBaseURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeTelegramAPIBaseURL error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeTelegramAPIBaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewTelegramChannel_UsesCustomAPIBaseURL(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Token = "123456:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghi"
	cfg.Channels.Telegram.APIBaseURL = "https://telegram-proxy.example.com/custom/"

	channel, err := NewTelegramChannel(cfg, bus.NewMessageBus())
	if err != nil {
		t.Fatalf("NewTelegramChannel error: %v", err)
	}

	got := channel.bot.FileDownloadURL("photos/abc.jpg")
	wantPrefix := "https://telegram-proxy.example.com/custom/file/bot123456:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghi/"
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("FileDownloadURL prefix = %q, want prefix %q", got, wantPrefix)
	}
}

func TestNewTelegramChannel_InvalidCustomAPIBaseURLErrors(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Channels.Telegram.Token = "123456:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghi"
	cfg.Channels.Telegram.APIBaseURL = "telegram-proxy.example.com"

	_, err := NewTelegramChannel(cfg, bus.NewMessageBus())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid telegram api_base_url") {
		t.Fatalf("error = %q, expected invalid api_base_url message", err.Error())
	}
}
