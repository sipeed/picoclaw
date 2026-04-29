# slack_webhook Channel Design

Output-only channel for pushing messages to Slack via Incoming Webhooks with Block Kit formatting.

## Overview

A new channel type `slack_webhook` that sends messages to Slack channels using Incoming Webhook URLs. Unlike the full `slack` channel (bidirectional bot via Socket Mode), this is output-only and requires no OAuth app setup beyond creating webhooks.

## Package Structure

```
pkg/channels/slack_webhook/
├── slack_webhook.go      # Channel implementation
├── slack_webhook_test.go # Unit tests
├── convert.go            # Markdown → Slack mrkdwn converter
└── convert_test.go       # Converter tests
```

## Configuration

### Config Struct

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

### Example Configuration

```json
{
  "type": "slack_webhook",
  "webhooks": {
    "default": {
      "webhook_url": "https://hooks.slack.com/services/T.../B.../xxx",
      "username": "PicoClaw",
      "icon_emoji": ":robot_face:"
    },
    "alerts": {
      "webhook_url": "https://hooks.slack.com/services/T.../B.../yyy",
      "icon_emoji": ":warning:"
    }
  }
}
```

### Validation Rules

- At least one webhook target required
- "default" webhook target required
- All webhook URLs must be valid HTTPS URLs

## Message Format

Messages use Slack Block Kit for rich formatting:

```json
{
  "username": "PicoClaw",
  "icon_emoji": ":robot_face:",
  "blocks": [
    {
      "type": "section",
      "text": { "type": "mrkdwn", "text": "Message content here..." }
    }
  ]
}
```

### Content Handling

- Content split into segments (text vs table) like teams_webhook
- Text segments → section blocks with mrkdwn
- Tables → formatted text if narrow, code block if wide (>60 chars per row)
- Long content split across multiple sections (3000 char limit per text block)

## Markdown Conversion

| Markdown | Slack mrkdwn |
|----------|--------------|
| `**bold**` | `*bold*` |
| `*italic*` or `_italic_` | `_italic_` |
| `~~strike~~` | `~strike~` |
| `` `code` `` | `` `code` `` |
| ```` ```codeblock``` ```` | ```` ```codeblock``` ```` |
| `[text](url)` | `<url\|text>` |
| `# Header` | `*Header*` (bold) |
| `- item` | `• item` |

### Table Rendering

Tables with all rows ≤60 chars wide:
```
*Col1* | *Col2*
val1   | val2
```

Tables with any row >60 chars → code block:
```
| Col1 | Col2 |
|------|------|
| val1 | val2 |
```

## Error Handling

- Extract HTTP status codes from Slack API errors
- 4xx errors → permanent (channels.ErrPermanent)
- 5xx errors → temporary/retryable (channels.ErrTemporary)
- Unknown ChatID → fall back to "default" target with warning log
- Context cancellation checked before send

## Channel Interface

```go
type SlackWebhookChannel struct {
    *channels.BaseChannel
    config *config.SlackWebhookSettings
    client *http.Client
}

func (c *SlackWebhookChannel) Start(ctx context.Context) error  // No-op for output-only
func (c *SlackWebhookChannel) Stop(ctx context.Context) error   // Set not running
func (c *SlackWebhookChannel) Send(ctx context.Context, msg bus.OutboundMessage) ([]string, error)
```

## Integration Points

### Config Registration

Add to `pkg/config/config_channel.go`:
- `ChannelSlackWebhook` constant
- `SlackWebhookSettings` in channel settings map

### Channel Manager

Add to `pkg/channels/manager.go`:
- Import slack_webhook package
- Create channel in switch statement

## Testing Strategy

- Unit tests for markdown converter (convert_test.go)
- Unit tests for table detection and rendering
- Unit tests for channel validation (webhook URL validation)
- Mock HTTP client for send tests
