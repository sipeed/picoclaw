package grafana_alertmanager

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	// maxWebhookBodySize limits request body to prevent memory exhaustion.
	maxWebhookBodySize = 1 << 20 // 1 MiB
)

// Alert represents a single alert from Grafana Alertmanager.
type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
	SilenceURL   string            `json:"silenceURL,omitempty"`
	DashboardURL string            `json:"dashboardURL,omitempty"`
	PanelURL     string            `json:"panelURL,omitempty"`
	Values       map[string]any    `json:"values,omitempty"`
	ValueString  string            `json:"valueString,omitempty"`
}

// WebhookPayload represents the payload sent by Grafana Alertmanager.
type WebhookPayload struct {
	Receiver          string            `json:"receiver"`
	Status            string            `json:"status"`
	Alerts            []Alert           `json:"alerts"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	OrgID             int               `json:"orgId"`
	Title             string            `json:"title"`
	State             string            `json:"state"`
	Message           string            `json:"message"`
}

// GrafanaAlertmanagerChannel implements a webhook endpoint for Grafana Alertmanager.
type GrafanaAlertmanagerChannel struct {
	*channels.BaseChannel
	config config.GrafanaAlertmanagerConfig
}

// NewGrafanaAlertmanagerChannel creates a new Grafana Alertmanager channel.
// Returns an error if allow_from is restrictive but no secret is configured,
// since allow_from cannot meaningfully restrict webhook callers (the senderID
// is always "grafana-alertmanager"). Use secret for authentication instead.
func NewGrafanaAlertmanagerChannel(
	cfg config.GrafanaAlertmanagerConfig,
	messageBus *bus.MessageBus,
) (*GrafanaAlertmanagerChannel, error) {
	// Validate: if allow_from is restrictive, secret must be configured
	// because allow_from checks senderID which is always "grafana-alertmanager"
	if !isOpenAccess(cfg.AllowFrom) && cfg.Secret.String() == "" {
		return nil, fmt.Errorf(
			"grafana_alertmanager: secret is required when allow_from is restrictive; " +
				"allow_from cannot restrict webhook callers (use secret for authentication)",
		)
	}

	base := channels.NewBaseChannel("grafana_alertmanager", cfg, messageBus, cfg.AllowFrom)

	return &GrafanaAlertmanagerChannel{
		BaseChannel: base,
		config:      cfg,
	}, nil
}

// isOpenAccess returns true if allowFrom permits open access (empty or contains "*").
func isOpenAccess(allowFrom []string) bool {
	if len(allowFrom) == 0 {
		return true
	}
	for _, v := range allowFrom {
		if v == "*" {
			return true
		}
	}
	return false
}

// Start initializes the channel.
func (c *GrafanaAlertmanagerChannel) Start(ctx context.Context) error {
	logger.InfoC("grafana_alertmanager", "Starting Grafana Alertmanager channel (Webhook Mode)")
	c.SetRunning(true)
	return nil
}

// Stop gracefully stops the channel.
func (c *GrafanaAlertmanagerChannel) Stop(ctx context.Context) error {
	logger.InfoC("grafana_alertmanager", "Stopping Grafana Alertmanager channel")
	c.SetRunning(false)
	return nil
}

// Send is a no-op for this input-only channel.
// Returns nil, nil to avoid triggering retry logic in the channel manager.
func (c *GrafanaAlertmanagerChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	// This is an input-only channel, no outbound messages supported.
	// Return nil, nil to signal "nothing to do" without triggering retries.
	return nil, nil
}

// WebhookPath returns the path for registering on the shared HTTP server.
func (c *GrafanaAlertmanagerChannel) WebhookPath() string {
	if c.config.WebhookPath != "" {
		return c.config.WebhookPath
	}
	return "/webhook/grafana-alertmanager"
}

// ServeHTTP implements http.Handler for the shared HTTP server.
func (c *GrafanaAlertmanagerChannel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodySize+1))
	if err != nil {
		logger.ErrorCF("grafana_alertmanager", "Failed to read request body", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if int64(len(body)) > maxWebhookBodySize {
		logger.WarnC("grafana_alertmanager", "Webhook request body too large, rejected")
		http.Error(w, "Request entity too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Verify signature if secret is configured
	if c.config.Secret.String() != "" {
		// Grafana Alerting sends signatures in "X-Grafana-Alerting-Signature" header
		signature := r.Header.Get("X-Grafana-Alerting-Signature")
		if !c.verifySignature(body, signature) {
			logger.WarnC("grafana_alertmanager", "Invalid webhook signature")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		logger.ErrorCF("grafana_alertmanager", "Failed to parse webhook payload", map[string]any{
			"error": err.Error(),
		})
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	logger.InfoCF("grafana_alertmanager", "Received alert webhook", map[string]any{
		"status":      payload.Status,
		"alert_count": len(payload.Alerts),
		"receiver":    payload.Receiver,
		"title":       payload.Title,
	})

	// Format the alert message
	content := c.formatAlertMessage(&payload)

	// Determine chat ID - use configured chat_id or generate from receiver
	chatID := c.config.ChatID
	if chatID == "" {
		chatID = "grafana:" + payload.Receiver
	}

	// Use receiver as sender ID
	senderID := "grafana-alertmanager"

	// Publish the inbound message
	c.HandleMessage(
		r.Context(),
		bus.Peer{Kind: "webhook", ID: payload.Receiver},
		payload.GroupKey, // messageID
		senderID,
		chatID,
		content,
		nil, // no media
		nil, // no metadata
	)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// verifySignature verifies the HMAC-SHA256 signature of the webhook payload.
// The signature is expected to be a hex-encoded string, optionally prefixed with "sha256=".
func (c *GrafanaAlertmanagerChannel) verifySignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	// Remove "sha256=" prefix if present
	signature = strings.TrimPrefix(signature, "sha256=")

	// Decode the provided signature from hex
	providedMAC, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	// Compute expected MAC
	mac := hmac.New(sha256.New, []byte(c.config.Secret.String()))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)

	// Constant-time comparison (only works correctly when lengths match)
	if len(providedMAC) != len(expectedMAC) {
		return false
	}

	return hmac.Equal(providedMAC, expectedMAC)
}

// formatAlertMessage formats the alert payload into a readable message.
func (c *GrafanaAlertmanagerChannel) formatAlertMessage(payload *WebhookPayload) string {
	var sb strings.Builder

	// Title and status
	status := strings.ToUpper(payload.Status)
	if payload.Title != "" {
		sb.WriteString(fmt.Sprintf("**[%s] %s**\n\n", status, payload.Title))
	} else {
		sb.WriteString(fmt.Sprintf("**[%s] Grafana Alert**\n\n", status))
	}

	// Message if present
	if payload.Message != "" {
		sb.WriteString(payload.Message)
		sb.WriteString("\n\n")
	}

	// Alert details
	for i, alert := range payload.Alerts {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}

		// Alert name from labels
		alertName := alert.Labels["alertname"]
		if alertName == "" {
			alertName = fmt.Sprintf("Alert %d", i+1)
		}
		sb.WriteString(fmt.Sprintf("**%s** (%s)\n", alertName, alert.Status))

		// Summary and description from annotations
		if summary := alert.Annotations["summary"]; summary != "" {
			sb.WriteString(fmt.Sprintf("Summary: %s\n", summary))
		}
		if description := alert.Annotations["description"]; description != "" {
			sb.WriteString(fmt.Sprintf("Description: %s\n", description))
		}

		// Value string if present
		if alert.ValueString != "" {
			sb.WriteString(fmt.Sprintf("Value: %s\n", alert.ValueString))
		}

		// Key labels (excluding alertname)
		var labels []string
		for k, v := range alert.Labels {
			if k != "alertname" && k != "__alert_rule_uid__" && k != "__alert_rule_namespace_uid__" {
				labels = append(labels, fmt.Sprintf("%s=%s", k, v))
			}
		}
		if len(labels) > 0 {
			sb.WriteString(fmt.Sprintf("Labels: %s\n", strings.Join(labels, ", ")))
		}

		// Links
		if alert.DashboardURL != "" {
			sb.WriteString(fmt.Sprintf("Dashboard: %s\n", alert.DashboardURL))
		}
		if alert.PanelURL != "" {
			sb.WriteString(fmt.Sprintf("Panel: %s\n", alert.PanelURL))
		}
		if alert.SilenceURL != "" {
			sb.WriteString(fmt.Sprintf("Silence: %s\n", alert.SilenceURL))
		}
	}

	// External URL
	if payload.ExternalURL != "" {
		sb.WriteString(fmt.Sprintf("\nGrafana: %s\n", payload.ExternalURL))
	}

	return sb.String()
}
