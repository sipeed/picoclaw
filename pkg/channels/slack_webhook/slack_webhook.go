package slackwebhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const maxTextBlockLength = 3000

// SlackWebhookChannel is an output-only channel that sends messages
// to Slack via Incoming Webhooks using Block Kit formatting.
type SlackWebhookChannel struct {
	*channels.BaseChannel
	bc     *config.Channel
	config *config.SlackWebhookSettings
	client *http.Client
}

// NewSlackWebhookChannel creates a new Slack webhook channel.
func NewSlackWebhookChannel(
	bc *config.Channel,
	cfg *config.SlackWebhookSettings,
	bus *bus.MessageBus,
) (*SlackWebhookChannel, error) {
	if len(cfg.Webhooks) == 0 {
		return nil, fmt.Errorf("slack_webhook: at least one webhook target is required")
	}

	if _, hasDefault := cfg.Webhooks["default"]; !hasDefault {
		return nil, fmt.Errorf("slack_webhook: a 'default' webhook target is required")
	}

	for name, target := range cfg.Webhooks {
		webhookURL := target.WebhookURL.String()
		if webhookURL == "" {
			return nil, fmt.Errorf("slack_webhook: webhook %q has empty webhook_url", name)
		}
		parsed, err := url.Parse(webhookURL)
		if err != nil {
			return nil, fmt.Errorf("slack_webhook: webhook %q has invalid URL: %w", name, err)
		}
		if !strings.EqualFold(parsed.Scheme, "https") {
			return nil, fmt.Errorf("slack_webhook: webhook %q must use HTTPS (got %q)", name, parsed.Scheme)
		}
	}

	base := channels.NewBaseChannel(
		"slack_webhook",
		cfg,
		bus,
		[]string{"*"},
		channels.WithMaxMessageLength(40000),
	)

	return &SlackWebhookChannel{
		BaseChannel: base,
		bc:          bc,
		config:      cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Start initializes the channel. For output-only channels, this is a no-op.
func (c *SlackWebhookChannel) Start(ctx context.Context) error {
	targets := make([]string, 0, len(c.config.Webhooks))
	for name := range c.config.Webhooks {
		targets = append(targets, name)
	}
	sort.Strings(targets)
	logger.InfoCF("slack_webhook", "Starting Slack webhook channel (output-only)", map[string]any{
		"targets": targets,
	})
	c.SetRunning(true)
	return nil
}

// Stop shuts down the channel.
func (c *SlackWebhookChannel) Stop(ctx context.Context) error {
	logger.InfoC("slack_webhook", "Stopping Slack webhook channel")
	c.SetRunning(false)
	return nil
}

// Send delivers a message to the specified Slack webhook target.
func (c *SlackWebhookChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	targetName := msg.ChatID
	if targetName == "" {
		targetName = "default"
	}

	target, ok := c.config.Webhooks[targetName]
	if !ok {
		logger.WarnCF("slack_webhook", "Unknown target, falling back to default", map[string]any{
			"requested": msg.ChatID,
			"using":     "default",
		})
		target = c.config.Webhooks["default"]
	}

	payload := c.buildPayload(msg, target)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("slack_webhook: failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", target.WebhookURL.String(), bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("slack_webhook: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		logger.ErrorCF("slack_webhook", "Failed to send message", map[string]any{
			"target": targetName,
		})
		return nil, fmt.Errorf("slack_webhook: send failed: %w", channels.ClassifyNetError(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		logger.ErrorCF("slack_webhook", "Slack API error", map[string]any{
			"target":   targetName,
			"status":   resp.StatusCode,
			"response": string(respBody),
		})
		return nil, fmt.Errorf("slack_webhook: send failed (status %d: %s): %w",
			resp.StatusCode, string(respBody), channels.ClassifySendError(resp.StatusCode, nil))
	}

	logger.DebugCF("slack_webhook", "Message sent successfully", map[string]any{
		"target": targetName,
	})

	return nil, nil
}

func (c *SlackWebhookChannel) buildPayload(msg bus.OutboundMessage, target config.SlackWebhookTarget) map[string]any {
	payload := make(map[string]any)

	if target.Username != "" {
		payload["username"] = target.Username
	}
	if target.IconEmoji != "" {
		payload["icon_emoji"] = target.IconEmoji
	}

	content := msg.Content
	if content == "" {
		content = "(empty message)"
	}

	blocks := c.buildBlocks(content)
	payload["blocks"] = blocks

	return payload
}

func (c *SlackWebhookChannel) buildBlocks(content string) []map[string]any {
	var blocks []map[string]any

	segments := splitContentWithTables(content)

	for _, seg := range segments {
		if seg.isTable {
			tableText := renderTable(seg.content)
			for _, chunk := range splitText(tableText, maxTextBlockLength) {
				blocks = append(blocks, c.textSection(chunk))
			}
		} else {
			text := strings.TrimSpace(seg.content)
			if text == "" {
				continue
			}
			converted := convertMarkdownToMrkdwn(text)
			for _, chunk := range splitText(converted, maxTextBlockLength) {
				blocks = append(blocks, c.textSection(chunk))
			}
		}
	}

	if len(blocks) == 0 {
		blocks = append(blocks, c.textSection("(empty message)"))
	}

	return blocks
}

func (c *SlackWebhookChannel) textSection(text string) map[string]any {
	return map[string]any{
		"type": "section",
		"text": map[string]any{
			"type": "mrkdwn",
			"text": text,
		},
	}
}

func splitText(text string, maxLen int) []string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return []string{text}
	}

	var chunks []string
	inFence := false

	for len(runes) > maxLen {
		splitAt := findSplitPoint(runes, maxLen, inFence)
		chunk := string(runes[:splitAt])
		chunks = append(chunks, chunk)
		inFence = endsInsideFence(chunk, inFence)
		runes = runes[splitAt:]
	}
	if len(runes) > 0 {
		chunks = append(chunks, string(runes))
	}
	return chunks
}

func findSplitPoint(runes []rune, maxLen int, inFence bool) int {
	window := string(runes[:maxLen])

	if idx := strings.LastIndex(window, "\n"); idx > 0 {
		splitAt := len([]rune(window[:idx])) + 1
		if !inFence || !endsInsideFence(string(runes[:splitAt]), inFence) {
			return splitAt
		}
	}

	if idx := strings.LastIndex(window, " "); idx > 0 {
		splitAt := len([]rune(window[:idx])) + 1
		if !inFence {
			return splitAt
		}
	}

	if inFence {
		if idx := strings.LastIndex(window, "```"); idx > 0 {
			return len([]rune(window[:idx]))
		}
	}

	return maxLen
}

func endsInsideFence(text string, wasInFence bool) bool {
	return wasInFence != (strings.Count(text, "```")%2 == 1)
}
