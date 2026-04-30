package slackwebhook

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestNewSlackWebhookChannel_Validation(t *testing.T) {
	tests := []struct {
		name      string
		webhooks  map[string]config.SlackWebhookTarget
		expectErr string
	}{
		{
			name:      "empty webhooks",
			webhooks:  map[string]config.SlackWebhookTarget{},
			expectErr: "at least one webhook target is required",
		},
		{
			name: "missing default",
			webhooks: map[string]config.SlackWebhookTarget{
				"alerts": {WebhookURL: *config.NewSecureString("https://hooks.slack.com/services/T/B/x")},
			},
			expectErr: "a 'default' webhook target is required",
		},
		{
			name: "empty webhook URL",
			webhooks: map[string]config.SlackWebhookTarget{
				"default": {WebhookURL: *config.NewSecureString("")},
			},
			expectErr: "has empty webhook_url",
		},
		{
			name: "non-HTTPS URL",
			webhooks: map[string]config.SlackWebhookTarget{
				"default": {WebhookURL: *config.NewSecureString("http://hooks.slack.com/services/T/B/x")},
			},
			expectErr: "must use HTTPS",
		},
		{
			name: "valid config",
			webhooks: map[string]config.SlackWebhookTarget{
				"default": {
					WebhookURL: *config.NewSecureString("https://hooks.slack.com/services/T/B/x"),
					Username:   "TestBot",
					IconEmoji:  ":robot_face:",
				},
			},
			expectErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.SlackWebhookSettings{Webhooks: tt.webhooks}
			bc := &config.Channel{Enabled: true}
			mb := bus.NewMessageBus()

			ch, err := NewSlackWebhookChannel(bc, cfg, mb)
			if tt.expectErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, ch)
			}
		})
	}
}

func TestSlackWebhookChannel_Send(t *testing.T) {
	payloadCh := make(chan map[string]any, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		json.Unmarshal(body, &payload)
		payloadCh <- payload
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.SlackWebhookSettings{
		Webhooks: map[string]config.SlackWebhookTarget{
			"default": {
				WebhookURL: *config.NewSecureString(server.URL),
				Username:   "TestBot",
				IconEmoji:  ":test:",
			},
		},
	}
	bc := &config.Channel{Enabled: true}
	mb := bus.NewMessageBus()

	ch, err := NewSlackWebhookChannel(bc, cfg, mb)
	require.NoError(t, err)

	// Use the test server's client to skip TLS verification
	ch.client = server.Client()

	err = ch.Start(context.Background())
	require.NoError(t, err)

	_, err = ch.Send(context.Background(), bus.OutboundMessage{
		Content: "Hello **world**",
		ChatID:  "default",
	})
	require.NoError(t, err)

	// Verify payload structure
	receivedPayload := <-payloadCh
	assert.Equal(t, "TestBot", receivedPayload["username"])
	assert.Equal(t, ":test:", receivedPayload["icon_emoji"])
	blocks, ok := receivedPayload["blocks"].([]any)
	require.True(t, ok)
	require.Len(t, blocks, 1)
}

func TestSlackWebhookChannel_FallbackToDefault(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.SlackWebhookSettings{
		Webhooks: map[string]config.SlackWebhookTarget{
			"default": {WebhookURL: *config.NewSecureString(server.URL)},
		},
	}
	bc := &config.Channel{Enabled: true}
	mb := bus.NewMessageBus()

	ch, err := NewSlackWebhookChannel(bc, cfg, mb)
	require.NoError(t, err)
	ch.client = server.Client()
	ch.Start(context.Background())

	// Send to unknown target - should fall back to default
	_, err = ch.Send(context.Background(), bus.OutboundMessage{
		Content: "Test",
		ChatID:  "unknown_target",
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), requestCount.Load())
}

func TestSlackWebhookChannel_ErrorClassification(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expectTemp bool
	}{
		{"400 Bad Request", 400, false},
		{"401 Unauthorized", 401, false},
		{"403 Forbidden", 403, false},
		{"404 Not Found", 404, false},
		{"500 Internal Error", 500, true},
		{"502 Bad Gateway", 502, true},
		{"503 Service Unavailable", 503, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			cfg := &config.SlackWebhookSettings{
				Webhooks: map[string]config.SlackWebhookTarget{
					"default": {WebhookURL: *config.NewSecureString(server.URL)},
				},
			}
			bc := &config.Channel{Enabled: true}
			mb := bus.NewMessageBus()

			ch, _ := NewSlackWebhookChannel(bc, cfg, mb)
			ch.client = server.Client()
			ch.Start(context.Background())

			_, err := ch.Send(context.Background(), bus.OutboundMessage{Content: "Test"})
			require.Error(t, err)

			if tt.expectTemp {
				assert.True(t, errors.Is(err, channels.ErrTemporary), "expected temporary error for %d", tt.statusCode)
			} else {
				assert.True(t, errors.Is(err, channels.ErrSendFailed), "expected permanent error for %d", tt.statusCode)
			}
		})
	}
}
