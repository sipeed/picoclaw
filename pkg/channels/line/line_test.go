package line

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestWebhookRejectsOversizedBody(t *testing.T) {
	ch := &LINEChannel{config: &config.LINESettings{}}

	oversized := bytes.Repeat([]byte("A"), maxWebhookBodySize+1)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(oversized))
	rec := httptest.NewRecorder()

	ch.webhookHandler(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
}

func TestWebhookAcceptsMaxBodySize(t *testing.T) {
	ch := &LINEChannel{config: &config.LINESettings{}}

	body := bytes.Repeat([]byte("A"), maxWebhookBodySize)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ch.webhookHandler(rec, req)

	// Missing signature should be rejected, but the body size should not trigger 413.
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestWebhookRejectsOversizedBodyBeforeSignatureCheck(t *testing.T) {
	ch := &LINEChannel{config: &config.LINESettings{}}

	oversized := bytes.Repeat([]byte("A"), maxWebhookBodySize+1)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(oversized))
	req.Header.Set("X-Line-Signature", "invalidsignature")
	rec := httptest.NewRecorder()

	ch.webhookHandler(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
}

func TestWebhookRejectsNonPostMethod(t *testing.T) {
	ch := &LINEChannel{config: &config.LINESettings{}}

	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	rec := httptest.NewRecorder()

	ch.webhookHandler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestWebhookRejectsInvalidSignature(t *testing.T) {
	ch := &LINEChannel{
		config: &config.LINESettings{},
	}

	body := `{"events":[]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", "invalidsignature")
	rec := httptest.NewRecorder()

	ch.webhookHandler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestInertBindKeys(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.LINESettings
		want []string
	}{
		{name: "unset", cfg: config.LINESettings{}},
		{
			name: "host only",
			cfg:  config.LINESettings{WebhookHost: "0.0.0.0"},
			want: []string{"webhook_host"},
		},
		{
			name: "port only",
			cfg:  config.LINESettings{WebhookPort: 18791},
			want: []string{"webhook_port"},
		},
		{
			name: "both",
			cfg:  config.LINESettings{WebhookHost: "0.0.0.0", WebhookPort: 18791},
			want: []string{"webhook_host", "webhook_port"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inertBindKeys(&tt.cfg)
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("inertBindKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWebhookPathIgnoresBindSettings(t *testing.T) {
	ch := &LINEChannel{config: &config.LINESettings{
		WebhookHost: "127.0.0.1",
		WebhookPort: 9999,
	}}

	if got := ch.WebhookPath(); got != "/webhook/line" {
		t.Errorf("WebhookPath() = %q, want %q", got, "/webhook/line")
	}
}
