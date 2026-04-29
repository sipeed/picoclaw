# slack_webhook Channel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an output-only Slack webhook channel that sends messages via Incoming Webhooks with Block Kit formatting.

**Architecture:** New `pkg/channels/slack_webhook/` package with markdown-to-mrkdwn converter. Config structs in `pkg/config/config.go`. Factory registration via `init.go`. Follows teams_webhook patterns.

**Tech Stack:** Go, Slack Block Kit JSON, HTTP client

---

## File Structure

| File | Responsibility |
|------|----------------|
| `pkg/channels/slack_webhook/convert.go` | Markdown → Slack mrkdwn conversion |
| `pkg/channels/slack_webhook/convert_test.go` | Converter unit tests |
| `pkg/channels/slack_webhook/slack_webhook.go` | Channel implementation |
| `pkg/channels/slack_webhook/slack_webhook_test.go` | Channel unit tests |
| `pkg/channels/slack_webhook/init.go` | Factory registration |
| `pkg/config/config.go` | SlackWebhookSettings struct |
| `pkg/config/config_channel.go` | Channel constant + factory map |

---

## Task 1: Add config structs

**Files:**
- Modify: `pkg/config/config.go` (after TeamsWebhookTarget, ~line 515)

- [ ] **Step 1: Add SlackWebhookSettings and SlackWebhookTarget structs**

Add after `TeamsWebhookTarget` struct (around line 515):

```go
// SlackWebhookSettings configures the output-only Slack webhook channel.
type SlackWebhookSettings struct {
	Webhooks map[string]SlackWebhookTarget `json:"webhooks" yaml:"webhooks,omitempty"`
}

// SlackWebhookTarget represents a single Slack Incoming Webhook destination.
type SlackWebhookTarget struct {
	WebhookURL SecureString `json:"webhook_url,omitzero" yaml:"webhook_url,omitempty"`
	Username   string       `json:"username,omitempty"   yaml:"-"`
	IconEmoji  string       `json:"icon_emoji,omitempty" yaml:"-"`
}
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./pkg/config/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add pkg/config/config.go
git commit -m "feat(config): add SlackWebhookSettings struct"
```

---

## Task 2: Register channel type constant

**Files:**
- Modify: `pkg/config/config_channel.go` (lines 35 and 642)

- [ ] **Step 1: Add channel constant**

Add after `ChannelTeamsWebHook` (line 35):

```go
	ChannelSlackWebhook   = "slack_webhook"
```

- [ ] **Step 2: Add to channelSettingsFactory map**

Add after the `ChannelTeamsWebHook` entry (around line 642):

```go
	ChannelSlackWebhook:   (SlackWebhookSettings{}),
```

- [ ] **Step 3: Verify file compiles**

Run: `go build ./pkg/config/...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add pkg/config/config_channel.go
git commit -m "feat(config): register slack_webhook channel type"
```

---

## Task 3: Add config validation case in manager

**Files:**
- Modify: `pkg/channels/manager.go` (around line 609)

- [ ] **Step 1: Add case for SlackWebhookSettings**

Find the switch statement with `case *config.TeamsWebhookSettings:` (around line 609) and add after it:

```go
	case *config.SlackWebhookSettings:
		return bc, true
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./pkg/channels/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add pkg/channels/manager.go
git commit -m "feat(channels): add slack_webhook config validation case"
```

---

## Task 4: Create markdown converter tests

**Files:**
- Create: `pkg/channels/slack_webhook/convert_test.go`

- [ ] **Step 1: Create slack_webhook directory**

Run: `mkdir -p pkg/channels/slack_webhook`

- [ ] **Step 2: Write converter tests**

Create `pkg/channels/slack_webhook/convert_test.go`:

```go
package slackwebhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertMarkdownToMrkdwn(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bold double asterisk",
			input:    "This is **bold** text",
			expected: "This is *bold* text",
		},
		{
			name:     "italic single asterisk",
			input:    "This is *italic* text",
			expected: "This is _italic_ text",
		},
		{
			name:     "italic underscore",
			input:    "This is _italic_ text",
			expected: "This is _italic_ text",
		},
		{
			name:     "strikethrough",
			input:    "This is ~~struck~~ text",
			expected: "This is ~struck~ text",
		},
		{
			name:     "inline code unchanged",
			input:    "Use `code` here",
			expected: "Use `code` here",
		},
		{
			name:     "link conversion",
			input:    "Click [here](https://example.com) now",
			expected: "Click <https://example.com|here> now",
		},
		{
			name:     "header to bold",
			input:    "# Header One",
			expected: "*Header One*",
		},
		{
			name:     "header level 2",
			input:    "## Header Two",
			expected: "*Header Two*",
		},
		{
			name:     "bullet list",
			input:    "- item one\n- item two",
			expected: "• item one\n• item two",
		},
		{
			name:     "mixed formatting",
			input:    "**bold** and *italic* and [link](http://x.com)",
			expected: "*bold* and _italic_ and <http://x.com|link>",
		},
		{
			name:     "code block unchanged",
			input:    "```\ncode here\n```",
			expected: "```\ncode here\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertMarkdownToMrkdwn(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitContentWithTables(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedCount  int
		expectedTables int
	}{
		{
			name:           "no table",
			input:          "Just some text",
			expectedCount:  1,
			expectedTables: 0,
		},
		{
			name:           "simple table",
			input:          "| A | B |\n|---|---|\n| 1 | 2 |",
			expectedCount:  1,
			expectedTables: 1,
		},
		{
			name:           "text before table",
			input:          "Intro text\n\n| A | B |\n|---|---|\n| 1 | 2 |",
			expectedCount:  2,
			expectedTables: 1,
		},
		{
			name:           "text before and after table",
			input:          "Before\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\nAfter",
			expectedCount:  3,
			expectedTables: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := splitContentWithTables(tt.input)
			assert.Equal(t, tt.expectedCount, len(segments))
			tableCount := 0
			for _, seg := range segments {
				if seg.isTable {
					tableCount++
				}
			}
			assert.Equal(t, tt.expectedTables, tableCount)
		})
	}
}

func TestRenderTable(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectCode  bool
	}{
		{
			name:       "narrow table renders as text",
			input:      "| A | B |\n|---|---|\n| 1 | 2 |",
			expectCode: false,
		},
		{
			name:       "wide table renders as code block",
			input:      "| This is a very long column header | Another extremely long column header here |\n|---|---|\n| Some long value content here | More long value content |",
			expectCode: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderTable(tt.input)
			if tt.expectCode {
				assert.Contains(t, result, "```")
			} else {
				assert.NotContains(t, result, "```")
				assert.Contains(t, result, "*") // Bold headers
			}
		})
	}
}
```

- [ ] **Step 3: Verify tests fail (functions don't exist yet)**

Run: `go test ./pkg/channels/slack_webhook/... -v`
Expected: Compilation error (functions not defined)

- [ ] **Step 4: Commit test file**

```bash
git add pkg/channels/slack_webhook/convert_test.go
git commit -m "test(slack_webhook): add converter tests"
```

---

## Task 5: Implement markdown converter

**Files:**
- Create: `pkg/channels/slack_webhook/convert.go`

- [ ] **Step 1: Create convert.go with converter functions**

Create `pkg/channels/slack_webhook/convert.go`:

```go
package slackwebhook

import (
	"regexp"
	"strings"
)

const maxTableRowWidth = 60

var (
	boldRe          = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	italicAsterisk  = regexp.MustCompile(`(?:^|[^*])\*([^*]+)\*(?:[^*]|$)`)
	strikeRe        = regexp.MustCompile(`~~([^~]+)~~`)
	linkRe          = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	headerRe        = regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)
	bulletRe        = regexp.MustCompile(`(?m)^- (.+)$`)
	markdownTableRe = regexp.MustCompile(`(?m)^(\|[^\n]+\|)\n(\|[-:\|\s]+\|)\n((?:\|[^\n]+\|\n?)+)`)
)

type contentSegment struct {
	content string
	isTable bool
}

func convertMarkdownToMrkdwn(text string) string {
	// Protect code blocks from conversion
	var codeBlocks []string
	codeBlockRe := regexp.MustCompile("(?s)```.*?```")
	text = codeBlockRe.ReplaceAllStringFunc(text, func(match string) string {
		codeBlocks = append(codeBlocks, match)
		return "\x00CODEBLOCK\x00"
	})

	// Protect inline code
	var inlineCode []string
	inlineCodeRe := regexp.MustCompile("`[^`]+`")
	text = inlineCodeRe.ReplaceAllStringFunc(text, func(match string) string {
		inlineCode = append(inlineCode, match)
		return "\x00INLINE\x00"
	})

	// Convert bold **text** → *text*
	text = boldRe.ReplaceAllString(text, "*$1*")

	// Convert italic *text* → _text_ (but not inside bold)
	// This is tricky - we need to handle *text* that isn't bold
	text = convertItalicAsterisk(text)

	// Convert strikethrough ~~text~~ → ~text~
	text = strikeRe.ReplaceAllString(text, "~$1~")

	// Convert links [text](url) → <url|text>
	text = linkRe.ReplaceAllString(text, "<$2|$1>")

	// Convert headers # text → *text*
	text = headerRe.ReplaceAllString(text, "*$1*")

	// Convert bullet lists - item → • item
	text = bulletRe.ReplaceAllString(text, "• $1")

	// Restore inline code
	for _, code := range inlineCode {
		text = strings.Replace(text, "\x00INLINE\x00", code, 1)
	}

	// Restore code blocks
	for _, block := range codeBlocks {
		text = strings.Replace(text, "\x00CODEBLOCK\x00", block, 1)
	}

	return text
}

func convertItalicAsterisk(text string) string {
	// Convert standalone *text* to _text_
	// Must not be part of **bold** (already converted to *bold*)
	result := strings.Builder{}
	i := 0
	for i < len(text) {
		if text[i] == '*' && (i == 0 || text[i-1] != '*') {
			// Look for closing *
			end := strings.Index(text[i+1:], "*")
			if end != -1 && (i+1+end+1 >= len(text) || text[i+1+end+1] != '*') {
				// Found matching *, convert to _
				result.WriteByte('_')
				result.WriteString(text[i+1 : i+1+end])
				result.WriteByte('_')
				i = i + 1 + end + 1
				continue
			}
		}
		result.WriteByte(text[i])
		i++
	}
	return result.String()
}

func splitContentWithTables(content string) []contentSegment {
	var segments []contentSegment

	matches := markdownTableRe.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return []contentSegment{{content: content, isTable: false}}
	}

	lastEnd := 0
	for _, match := range matches {
		if match[0] > lastEnd {
			segments = append(segments, contentSegment{
				content: content[lastEnd:match[0]],
				isTable: false,
			})
		}
		segments = append(segments, contentSegment{
			content: content[match[0]:match[1]],
			isTable: true,
		})
		lastEnd = match[1]
	}

	if lastEnd < len(content) {
		segments = append(segments, contentSegment{
			content: content[lastEnd:],
			isTable: false,
		})
	}

	return segments
}

func renderTable(tableStr string) string {
	lines := strings.Split(strings.TrimSpace(tableStr), "\n")
	if len(lines) < 2 {
		return "```\n" + tableStr + "\n```"
	}

	// Check if any row exceeds max width
	for _, line := range lines {
		if len(line) > maxTableRowWidth {
			return "```\n" + tableStr + "\n```"
		}
	}

	// Render as formatted text with bold headers
	var result strings.Builder
	for i, line := range lines {
		if i == 1 && isSeparatorRow(line) {
			continue
		}
		cells := parseTableRow(line)
		if len(cells) == 0 {
			continue
		}
		if i == 0 {
			// Header row - bold each cell
			var boldCells []string
			for _, cell := range cells {
				boldCells = append(boldCells, "*"+strings.TrimSpace(cell)+"*")
			}
			result.WriteString(strings.Join(boldCells, " | "))
		} else {
			result.WriteString(strings.Join(cells, " | "))
		}
		result.WriteString("\n")
	}
	return strings.TrimSuffix(result.String(), "\n")
}

func isSeparatorRow(line string) bool {
	cleaned := strings.ReplaceAll(line, "|", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, ":", "")
	return cleaned == ""
}

func parseTableRow(line string) []string {
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
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./pkg/channels/slack_webhook/... -v`
Expected: All tests pass

- [ ] **Step 3: Commit**

```bash
git add pkg/channels/slack_webhook/convert.go
git commit -m "feat(slack_webhook): implement markdown to mrkdwn converter"
```

---

## Task 6: Create channel tests

**Files:**
- Create: `pkg/channels/slack_webhook/slack_webhook_test.go`

- [ ] **Step 1: Write channel tests**

Create `pkg/channels/slack_webhook/slack_webhook_test.go`:

```go
package slackwebhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/bus"
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
	var receivedPayload map[string]any
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedPayload)
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
	assert.Equal(t, "TestBot", receivedPayload["username"])
	assert.Equal(t, ":test:", receivedPayload["icon_emoji"])
	blocks, ok := receivedPayload["blocks"].([]any)
	require.True(t, ok)
	require.Len(t, blocks, 1)
}

func TestSlackWebhookChannel_FallbackToDefault(t *testing.T) {
	var requestCount int
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
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
	assert.Equal(t, 1, requestCount)
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
			// Error classification is internal; just verify we got an error
		})
	}
}
```

- [ ] **Step 2: Verify tests fail (channel not implemented yet)**

Run: `go test ./pkg/channels/slack_webhook/... -v`
Expected: Compilation error (NewSlackWebhookChannel not defined)

- [ ] **Step 3: Commit test file**

```bash
git add pkg/channels/slack_webhook/slack_webhook_test.go
git commit -m "test(slack_webhook): add channel tests"
```

---

## Task 7: Implement channel

**Files:**
- Create: `pkg/channels/slack_webhook/slack_webhook.go`

- [ ] **Step 1: Create slack_webhook.go with channel implementation**

Create `pkg/channels/slack_webhook/slack_webhook.go`:

```go
package slackwebhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

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
		client:      &http.Client{},
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
			"target": msg.ChatID,
		})
		return nil, fmt.Errorf("slack_webhook: send failed: %w", channels.ClassifyNetError(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		logger.ErrorCF("slack_webhook", "Slack API error", map[string]any{
			"target": msg.ChatID,
			"status": resp.StatusCode,
		})
		return nil, fmt.Errorf("slack_webhook: send failed: %w", channels.ClassifySendError(resp.StatusCode, fmt.Errorf("status %d", resp.StatusCode)))
	}

	logger.DebugCF("slack_webhook", "Message sent successfully", map[string]any{
		"target": msg.ChatID,
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
			blocks = append(blocks, c.textSection(tableText))
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
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	for len(text) > maxLen {
		splitAt := maxLen
		if idx := strings.LastIndex(text[:maxLen], "\n"); idx > 0 {
			splitAt = idx + 1
		} else if idx := strings.LastIndex(text[:maxLen], " "); idx > 0 {
			splitAt = idx + 1
		}
		chunks = append(chunks, text[:splitAt])
		text = text[splitAt:]
	}
	if len(text) > 0 {
		chunks = append(chunks, text)
	}
	return chunks
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./pkg/channels/slack_webhook/... -v`
Expected: All tests pass

- [ ] **Step 3: Commit**

```bash
git add pkg/channels/slack_webhook/slack_webhook.go
git commit -m "feat(slack_webhook): implement channel"
```

---

## Task 8: Add factory registration

**Files:**
- Create: `pkg/channels/slack_webhook/init.go`

- [ ] **Step 1: Create init.go with factory registration**

Create `pkg/channels/slack_webhook/init.go`:

```go
package slackwebhook

import (
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	channels.RegisterFactory(
		config.ChannelSlackWebhook,
		func(channelName, channelType string, cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
			bc := cfg.Channels[channelName]
			decoded, err := bc.GetDecoded()
			if err != nil {
				return nil, err
			}
			c, ok := decoded.(*config.SlackWebhookSettings)
			if !ok {
				return nil, channels.ErrSendFailed
			}
			ch, err := NewSlackWebhookChannel(bc, c, b)
			if err != nil {
				return nil, err
			}
			if channelName != config.ChannelSlackWebhook {
				ch.SetName(channelName)
			}
			return ch, nil
		},
	)
}
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./pkg/channels/slack_webhook/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add pkg/channels/slack_webhook/init.go
git commit -m "feat(slack_webhook): register channel factory"
```

---

## Task 9: Add blank import to gateway

**Files:**
- Modify: `cmd/picoclaw/main.go` or relevant gateway file

- [ ] **Step 1: Find and add blank import**

First, find where other channel imports are:

Run: `grep -r "slack_webhook\|teams_webhook" cmd/ --include="*.go" | head -5`

Then add the blank import alongside other channel imports:

```go
	_ "github.com/sipeed/picoclaw/pkg/channels/slack_webhook"
```

- [ ] **Step 2: Verify full build passes**

Run: `go build ./...`
Expected: No errors

- [ ] **Step 3: Run all tests**

Run: `go test ./pkg/channels/slack_webhook/... -v`
Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add cmd/
git commit -m "feat(gateway): register slack_webhook channel"
```

---

## Task 10: Final verification

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1`
Expected: All tests pass

- [ ] **Step 2: Verify lint passes**

Run: `golangci-lint run ./pkg/channels/slack_webhook/...` (if available)
Expected: No lint errors

- [ ] **Step 3: Create final commit with all files if any unstaged changes**

```bash
git status
# If any changes, add and commit
```
