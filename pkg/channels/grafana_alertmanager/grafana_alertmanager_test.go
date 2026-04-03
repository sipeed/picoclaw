package grafana_alertmanager

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewGrafanaAlertmanagerChannel_RequiresSecretWhenAllowFromRestrictive(t *testing.T) {
	tests := []struct {
		name      string
		allowFrom []string
		secret    string
		wantErr   bool
	}{
		{
			name:      "open access (empty allow_from) without secret is allowed",
			allowFrom: nil,
			secret:    "",
			wantErr:   false,
		},
		{
			name:      "open access (wildcard) without secret is allowed",
			allowFrom: []string{"*"},
			secret:    "",
			wantErr:   false,
		},
		{
			name:      "restrictive allow_from without secret fails",
			allowFrom: []string{"specific-sender"},
			secret:    "",
			wantErr:   true,
		},
		{
			name:      "restrictive allow_from with secret is allowed",
			allowFrom: []string{"specific-sender"},
			secret:    "mysecret",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.GrafanaAlertmanagerConfig{
				Enabled:   true,
				AllowFrom: tt.allowFrom,
			}
			if tt.secret != "" {
				cfg.Secret = *config.NewSecureString(tt.secret)
			}
			msgBus := bus.NewMessageBus()

			_, err := NewGrafanaAlertmanagerChannel(cfg, msgBus)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewGrafanaAlertmanagerChannel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWebhookRejectsNonPostMethod(t *testing.T) {
	ch := &GrafanaAlertmanagerChannel{}

	req := httptest.NewRequest(http.MethodGet, "/webhook/grafana-alertmanager", nil)
	rec := httptest.NewRecorder()

	ch.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestWebhookRejectsOversizedBody(t *testing.T) {
	ch := &GrafanaAlertmanagerChannel{}

	oversized := bytes.Repeat([]byte("A"), maxWebhookBodySize+1)
	req := httptest.NewRequest(http.MethodPost, "/webhook/grafana-alertmanager", bytes.NewReader(oversized))
	rec := httptest.NewRecorder()

	ch.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
}

func TestWebhookAcceptsMaxBodySize(t *testing.T) {
	ch := &GrafanaAlertmanagerChannel{
		config: config.GrafanaAlertmanagerConfig{
			Secret: *config.NewSecureString("testsecret"),
		},
	}

	// Create a body exactly at the limit
	body := bytes.Repeat([]byte("A"), maxWebhookBodySize)
	req := httptest.NewRequest(http.MethodPost, "/webhook/grafana-alertmanager", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	ch.ServeHTTP(rec, req)

	// Should fail on signature check, not body size
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d (forbidden due to missing signature), got %d", http.StatusForbidden, rec.Code)
	}
}

func TestWebhookRejectsMissingSignatureWhenSecretConfigured(t *testing.T) {
	ch := &GrafanaAlertmanagerChannel{
		config: config.GrafanaAlertmanagerConfig{
			Secret: *config.NewSecureString("testsecret"),
		},
	}

	body := `{"status":"firing","alerts":[]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook/grafana-alertmanager", strings.NewReader(body))
	rec := httptest.NewRecorder()

	ch.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestWebhookRejectsInvalidSignature(t *testing.T) {
	ch := &GrafanaAlertmanagerChannel{
		config: config.GrafanaAlertmanagerConfig{
			Secret: *config.NewSecureString("testsecret"),
		},
	}

	body := `{"status":"firing","alerts":[]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook/grafana-alertmanager", strings.NewReader(body))
	req.Header.Set("X-Grafana-Alerting-Signature", "invalidsignature")
	rec := httptest.NewRecorder()

	ch.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestWebhookRejectsInvalidJSON(t *testing.T) {
	ch := &GrafanaAlertmanagerChannel{
		config: config.GrafanaAlertmanagerConfig{}, // no secret, open access
	}

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/webhook/grafana-alertmanager", strings.NewReader(body))
	rec := httptest.NewRecorder()

	ch.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestWebhookAcceptsValidPayloadWithoutSecret(t *testing.T) {
	msgBus := bus.NewMessageBus()
	cfg := config.GrafanaAlertmanagerConfig{
		Enabled: true,
		ChatID:  "test-chat",
	}

	ch, err := NewGrafanaAlertmanagerChannel(cfg, msgBus)
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	body := `{
		"status": "firing",
		"receiver": "test-receiver",
		"alerts": [
			{
				"status": "firing",
				"labels": {"alertname": "TestAlert", "severity": "critical"},
				"annotations": {"summary": "Test summary", "description": "Test description"}
			}
		],
		"groupKey": "test-group",
		"title": "Test Alert Title"
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook/grafana-alertmanager", strings.NewReader(body))
	rec := httptest.NewRecorder()

	ch.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != `{"status":"ok"}` {
		t.Errorf("expected body %q, got %q", `{"status":"ok"}`, rec.Body.String())
	}
}

func TestWebhookAcceptsValidPayloadWithValidSignature(t *testing.T) {
	secret := "testsecret123"
	msgBus := bus.NewMessageBus()
	cfg := config.GrafanaAlertmanagerConfig{
		Enabled: true,
		Secret:  *config.NewSecureString(secret),
		ChatID:  "test-chat",
	}

	ch, err := NewGrafanaAlertmanagerChannel(cfg, msgBus)
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	body := `{"status":"firing","receiver":"test","alerts":[],"groupKey":"g1","title":"Test"}`

	// Compute valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhook/grafana-alertmanager", strings.NewReader(body))
	req.Header.Set("X-Grafana-Alerting-Signature", signature)
	rec := httptest.NewRecorder()

	ch.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWebhookAcceptsSignatureWithSha256Prefix(t *testing.T) {
	secret := "testsecret123"
	msgBus := bus.NewMessageBus()
	cfg := config.GrafanaAlertmanagerConfig{
		Enabled: true,
		Secret:  *config.NewSecureString(secret),
		ChatID:  "test-chat",
	}

	ch, err := NewGrafanaAlertmanagerChannel(cfg, msgBus)
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	body := `{"status":"firing","receiver":"test","alerts":[],"groupKey":"g1","title":"Test"}`

	// Compute valid signature with sha256= prefix
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhook/grafana-alertmanager", strings.NewReader(body))
	req.Header.Set("X-Grafana-Alerting-Signature", signature)
	rec := httptest.NewRecorder()

	ch.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestVerifySignature(t *testing.T) {
	ch := &GrafanaAlertmanagerChannel{
		config: config.GrafanaAlertmanagerConfig{
			Secret: *config.NewSecureString("mysecret"),
		},
	}

	body := []byte("test body content")
	mac := hmac.New(sha256.New, []byte("mysecret"))
	mac.Write(body)
	validSig := hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		body      []byte
		signature string
		want      bool
	}{
		{
			name:      "valid signature",
			body:      body,
			signature: validSig,
			want:      true,
		},
		{
			name:      "valid signature with sha256 prefix",
			body:      body,
			signature: "sha256=" + validSig,
			want:      true,
		},
		{
			name:      "empty signature",
			body:      body,
			signature: "",
			want:      false,
		},
		{
			name:      "invalid hex",
			body:      body,
			signature: "notvalidhex!!!",
			want:      false,
		},
		{
			name:      "wrong signature",
			body:      body,
			signature: strings.Repeat("ab", 32), // valid hex but wrong value
			want:      false,
		},
		{
			name:      "wrong length",
			body:      body,
			signature: "abcd",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ch.verifySignature(tt.body, tt.signature)
			if got != tt.want {
				t.Errorf("verifySignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatAlertMessage(t *testing.T) {
	ch := &GrafanaAlertmanagerChannel{}

	payload := &WebhookPayload{
		Status:   "firing",
		Receiver: "test-receiver",
		Title:    "High CPU Usage",
		Message:  "Server is experiencing high CPU load",
		Alerts: []Alert{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "HighCPU",
					"severity":  "critical",
					"host":      "server1",
				},
				Annotations: map[string]string{
					"summary":     "CPU usage above 90%",
					"description": "The server CPU has been above 90% for 5 minutes",
				},
				ValueString:  "cpu_usage: 95%",
				DashboardURL: "http://grafana/dashboard",
				PanelURL:     "http://grafana/panel",
				SilenceURL:   "http://grafana/silence",
			},
		},
		ExternalURL: "http://grafana",
	}

	result := ch.formatAlertMessage(payload)

	// Check key elements are present
	checks := []string{
		"**[FIRING] High CPU Usage**",
		"Server is experiencing high CPU load",
		"**HighCPU** (firing)",
		"Summary: CPU usage above 90%",
		"Description: The server CPU has been above 90% for 5 minutes",
		"Value: cpu_usage: 95%",
		"severity=critical",
		"host=server1",
		"Dashboard: http://grafana/dashboard",
		"Panel: http://grafana/panel",
		"Silence: http://grafana/silence",
		"Grafana: http://grafana",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("formatted message missing %q\nGot:\n%s", check, result)
		}
	}
}

func TestFormatAlertMessage_DefaultTitle(t *testing.T) {
	ch := &GrafanaAlertmanagerChannel{}

	payload := &WebhookPayload{
		Status: "resolved",
		Alerts: []Alert{},
	}

	result := ch.formatAlertMessage(payload)

	if !strings.Contains(result, "**[RESOLVED] Grafana Alert**") {
		t.Errorf("expected default title, got: %s", result)
	}
}

func TestIsOpenAccess(t *testing.T) {
	tests := []struct {
		name      string
		allowFrom []string
		want      bool
	}{
		{"nil", nil, true},
		{"empty", []string{}, true},
		{"wildcard only", []string{"*"}, true},
		{"wildcard with others", []string{"foo", "*", "bar"}, true},
		{"specific values", []string{"foo", "bar"}, false},
		{"single specific", []string{"foo"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOpenAccess(tt.allowFrom); got != tt.want {
				t.Errorf("isOpenAccess(%v) = %v, want %v", tt.allowFrom, got, tt.want)
			}
		})
	}
}
