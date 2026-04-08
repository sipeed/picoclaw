package teamswebhook

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	goteamsnotify "github.com/atc0005/go-teams-notify/v2"
	"github.com/atc0005/go-teams-notify/v2/adaptivecard"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

// statusCodeRe extracts HTTP status codes from error messages like "401 Unauthorized".
var statusCodeRe = regexp.MustCompile(`\b([45]\d{2})\b`)

// markdownTableRe matches a markdown table block (header + separator + rows).
// It captures the entire table including all rows.
var markdownTableRe = regexp.MustCompile(`(?m)^(\|[^\n]+\|)\n(\|[-:\|\s]+\|)\n((?:\|[^\n]+\|\n?)+)`)

// teamsMessageSender abstracts the Teams client for testability.
type teamsMessageSender interface {
	SendWithContext(ctx context.Context, webhookURL string, message goteamsnotify.TeamsMessage) error
}

// classifyTeamsError extracts HTTP status code from error message and classifies it.
// The go-teams-notify library returns errors like "error on notification: 401 Unauthorized, ...".
// This allows proper retry behavior: 4xx errors are permanent, 5xx are temporary.
func classifyTeamsError(err error) error {
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	if matches := statusCodeRe.FindStringSubmatch(errMsg); len(matches) > 1 {
		if statusCode, parseErr := strconv.Atoi(matches[1]); parseErr == nil {
			return channels.ClassifySendError(statusCode, err)
		}
	}
	// Fallback: treat as temporary network error (retryable)
	return channels.ClassifyNetError(err)
}

// TeamsWebhookChannel is an output-only channel that sends messages
// to Microsoft Teams via Power Automate workflow webhooks.
// Multiple webhook targets can be configured and selected via ChatID.
type TeamsWebhookChannel struct {
	*channels.BaseChannel
	config config.TeamsWebhookConfig
	client teamsMessageSender
}

// NewTeamsWebhookChannel creates a new Teams webhook channel.
func NewTeamsWebhookChannel(
	cfg config.TeamsWebhookConfig,
	bus *bus.MessageBus,
) (*TeamsWebhookChannel, error) {
	if len(cfg.Webhooks) == 0 {
		return nil, fmt.Errorf("teams_webhook: at least one webhook target is required")
	}

	// Require "default" webhook target
	if _, hasDefault := cfg.Webhooks["default"]; !hasDefault {
		return nil, fmt.Errorf("teams_webhook: a 'default' webhook target is required")
	}

	// Validate all webhook targets have valid HTTPS URLs
	for name, target := range cfg.Webhooks {
		webhookURL := target.WebhookURL.String()
		if webhookURL == "" {
			return nil, fmt.Errorf("teams_webhook: webhook %q has empty webhook_url", name)
		}
		parsed, err := url.Parse(webhookURL)
		if err != nil {
			return nil, fmt.Errorf("teams_webhook: webhook %q has invalid URL: %w", name, err)
		}
		if !strings.EqualFold(parsed.Scheme, "https") {
			return nil, fmt.Errorf("teams_webhook: webhook %q must use HTTPS (got %q)", name, parsed.Scheme)
		}
	}

	base := channels.NewBaseChannel(
		"teams_webhook",
		cfg,
		bus,
		[]string{
			"*",
		}, // Output-only channel; "*" suppresses misleading "allows EVERYONE" audit warning
		channels.WithMaxMessageLength(24000), // Power Automate webhook payload limit is 28KB
	)

	client := goteamsnotify.NewTeamsClient()

	return &TeamsWebhookChannel{
		BaseChannel: base,
		config:      cfg,
		client:      client,
	}, nil
}

// Start initializes the channel. For output-only channels, this is a no-op.
func (c *TeamsWebhookChannel) Start(ctx context.Context) error {
	targets := make([]string, 0, len(c.config.Webhooks))
	for name := range c.config.Webhooks {
		targets = append(targets, name)
	}
	sort.Strings(targets)
	logger.InfoCF("teams_webhook", "Starting Teams webhook channel (output-only)", map[string]any{
		"targets": targets,
	})
	c.SetRunning(true)
	return nil
}

// Stop shuts down the channel.
func (c *TeamsWebhookChannel) Stop(ctx context.Context) error {
	logger.InfoC("teams_webhook", "Stopping Teams webhook channel")
	c.SetRunning(false)
	return nil
}

// Send delivers a message to the specified Teams webhook target.
// The target is selected by msg.ChatID which must match a key in the webhooks map.
func (c *TeamsWebhookChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error) {
	if !c.IsRunning() {
		return nil, channels.ErrNotRunning
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Look up webhook target by ChatID, fall back to "default" if empty or unknown
	targetName := msg.ChatID
	if targetName == "" {
		targetName = "default"
	}

	target, ok := c.config.Webhooks[targetName]
	if !ok {
		// Log warning and fall back to default target
		logger.WarnCF("teams_webhook", "Unknown target, falling back to default", map[string]any{
			"requested": msg.ChatID,
			"using":     "default",
		})
		target = c.config.Webhooks["default"]
	}

	// Build an Adaptive Card for rich formatting
	card, err := c.buildAdaptiveCard(msg, target)
	if err != nil {
		return nil, fmt.Errorf("teams_webhook: failed to build card: %w", err)
	}

	// Create the message with the card
	teamsMsg, err := adaptivecard.NewMessageFromCard(card)
	if err != nil {
		return nil, fmt.Errorf("teams_webhook: failed to create message: %w", err)
	}

	// Send to Teams
	if err := c.client.SendWithContext(ctx, target.WebhookURL.String(), teamsMsg); err != nil {
		// Log without raw error to avoid leaking webhook URL (embedded in net/http errors)
		logger.ErrorCF("teams_webhook", "Failed to send message to Teams webhook", map[string]any{
			"target": msg.ChatID,
		})
		// Classify error based on status code extracted from error message.
		// The go-teams-notify library includes status in errors like "401 Unauthorized".
		// Use ClassifySendError for proper retry behavior (4xx = permanent, 5xx = temporary).
		classifiedErr := classifyTeamsError(err)
		return nil, fmt.Errorf("teams_webhook: send failed: %w", classifiedErr)
	}

	logger.DebugCF("teams_webhook", "Message sent successfully", map[string]any{
		"target": msg.ChatID,
	})

	return nil, nil
}

// buildAdaptiveCard creates a formatted Adaptive Card from the outbound message.
// It detects markdown tables and converts them to native Adaptive Card Table elements,
// since TextBlocks only support a limited markdown subset (no tables).
func (c *TeamsWebhookChannel) buildAdaptiveCard(
	msg bus.OutboundMessage,
	target config.TeamsWebhookTarget,
) (adaptivecard.Card, error) {
	card := adaptivecard.NewCard()
	card.Type = adaptivecard.TypeAdaptiveCard

	// Set full width for Teams rendering
	card.MSTeams.Width = "Full"

	// Add title if configured on the target
	title := target.Title
	if title == "" {
		title = "PicoClaw Notification"
	}

	titleBlock := adaptivecard.NewTextBlock(title, true)
	titleBlock.Size = adaptivecard.SizeLarge
	titleBlock.Weight = adaptivecard.WeightBolder
	titleBlock.Style = adaptivecard.TextBlockStyleHeading

	if err := card.AddElement(false, titleBlock); err != nil {
		return card, err
	}

	content := msg.Content
	if content == "" {
		content = "(empty message)"
	}

	// Split content into text segments and tables
	// TextBlocks support: bold, italic, bullet/numbered lists, links
	// TextBlocks do NOT support: headers, tables, images
	segments := splitContentWithTables(content)

	for _, seg := range segments {
		if seg.isTable {
			// Convert markdown table to Adaptive Card Table element
			tableElement, err := parseMarkdownTable(seg.content)
			if err != nil {
				// Fallback: render as preformatted text if parsing fails
				logger.WarnCF("teams_webhook", "Failed to parse markdown table, using fallback", map[string]any{
					"error": err.Error(),
				})
				block := adaptivecard.NewTextBlock("```\n"+seg.content+"\n```", true)
				block.Wrap = true
				if err := card.AddElement(false, block); err != nil {
					return card, err
				}
				continue
			}
			if err := card.AddElement(false, tableElement); err != nil {
				return card, err
			}
		} else {
			// Regular text content
			text := strings.TrimSpace(seg.content)
			if text == "" {
				continue
			}
			block := adaptivecard.NewTextBlock(text, true)
			block.Wrap = true
			if err := card.AddElement(false, block); err != nil {
				return card, err
			}
		}
	}

	return card, nil
}

// contentSegment represents either a text block or a table in the message content.
type contentSegment struct {
	content string
	isTable bool
}

// splitContentWithTables splits content into alternating text and table segments.
func splitContentWithTables(content string) []contentSegment {
	var segments []contentSegment

	matches := markdownTableRe.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		// No tables found, return entire content as text
		return []contentSegment{{content: content, isTable: false}}
	}

	lastEnd := 0
	for _, match := range matches {
		// Text before this table
		if match[0] > lastEnd {
			segments = append(segments, contentSegment{
				content: content[lastEnd:match[0]],
				isTable: false,
			})
		}
		// The table itself
		segments = append(segments, contentSegment{
			content: content[match[0]:match[1]],
			isTable: true,
		})
		lastEnd = match[1]
	}

	// Text after the last table
	if lastEnd < len(content) {
		segments = append(segments, contentSegment{
			content: content[lastEnd:],
			isTable: false,
		})
	}

	return segments
}

// parseMarkdownTable converts a markdown table string to an Adaptive Card Table element.
func parseMarkdownTable(tableStr string) (adaptivecard.Element, error) {
	lines := strings.Split(strings.TrimSpace(tableStr), "\n")
	if len(lines) < 2 {
		return adaptivecard.Element{}, fmt.Errorf("table must have at least header and separator rows")
	}

	// Track header content length per column for width calculation
	var headerLengths []int

	// Parse all rows (header + data rows, skip separator)
	var allRows [][]adaptivecard.TableCell
	for i, line := range lines {
		// Skip separator row (contains only |, -, :, and spaces)
		if i == 1 && isSeparatorRow(line) {
			continue
		}

		cells := parseTableRow(line)
		if len(cells) == 0 {
			continue
		}

		var tableCells []adaptivecard.TableCell
		for _, cellText := range cells {
			trimmedText := strings.TrimSpace(cellText)

			// Use header row (first row) to determine column widths
			if i == 0 {
				headerLengths = append(headerLengths, len(trimmedText))
			}

			textBlock := adaptivecard.Element{
				Type: adaptivecard.TypeElementTextBlock,
				Text: trimmedText,
				Wrap: true,
			}
			cell := adaptivecard.TableCell{
				Type:  adaptivecard.TypeTableCell,
				Items: []*adaptivecard.Element{&textBlock},
			}
			tableCells = append(tableCells, cell)
		}
		allRows = append(allRows, tableCells)
	}

	if len(allRows) == 0 {
		return adaptivecard.Element{}, fmt.Errorf("no valid rows found in table")
	}

	// Create table with first row as headers
	firstRowAsHeaders := true
	showGridLines := true

	table, err := adaptivecard.NewTableFromTableCells(allRows, 0, firstRowAsHeaders, showGridLines)
	if err != nil {
		return adaptivecard.Element{}, fmt.Errorf("failed to create table: %w", err)
	}

	// Set column widths based on header content length
	table.Columns = calculateColumnWidths(headerLengths)

	return table, nil
}

// calculateColumnWidths creates TableColumnDefinition entries with widths
// proportional to the max content length of each column.
func calculateColumnWidths(maxLengths []int) []adaptivecard.Column {
	if len(maxLengths) == 0 {
		return nil
	}

	// Use content length as relative weight, with a minimum of 1
	columns := make([]adaptivecard.Column, len(maxLengths))
	for i, length := range maxLengths {
		weight := length
		if weight < 1 {
			weight = 1
		}
		columns[i] = adaptivecard.Column{
			Type:  "TableColumnDefinition",
			Width: weight,
		}
	}

	return columns
}

// isSeparatorRow checks if a line is a markdown table separator (e.g., |---|---|).
func isSeparatorRow(line string) bool {
	// Remove pipes and spaces, check if only dashes and colons remain
	cleaned := strings.ReplaceAll(line, "|", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, ":", "")
	return cleaned == ""
}

// parseTableRow extracts cell values from a markdown table row.
func parseTableRow(line string) []string {
	// Trim leading/trailing pipes and split by |
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")

	if line == "" {
		return nil
	}

	parts := strings.Split(line, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}
