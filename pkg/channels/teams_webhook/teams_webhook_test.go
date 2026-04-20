package teamswebhook

import (
	"context"
	"errors"
	"testing"

	goteamsnotify "github.com/atc0005/go-teams-notify/v2"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

// mockTeamsClient implements teamsMessageSender for testing.
type mockTeamsClient struct {
	sendFunc func(ctx context.Context, webhookURL string, message goteamsnotify.TeamsMessage) error
}

func (m *mockTeamsClient) SendWithContext(
	ctx context.Context,
	webhookURL string,
	message goteamsnotify.TeamsMessage,
) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, webhookURL, message)
	}
	return nil
}

func TestNewTeamsWebhookChannel(t *testing.T) {
	msgBus := bus.NewMessageBus()

	// Test missing webhooks
	bc := &config.Channel{Type: config.ChannelTeamsWebHook, Enabled: true}
	cfg := config.TeamsWebhookSettings{
		Webhooks: nil,
	}
	_, err := NewTeamsWebhookChannel(bc, &cfg, msgBus)
	if err == nil {
		t.Error("expected error for missing webhooks")
	}

	// Test missing "default" webhook
	cfg.Webhooks = map[string]config.TeamsWebhookTarget{
		"alerts": {
			WebhookURL: *config.NewSecureString("https://example.com/webhook"),
			Title:      "Alerts",
		},
	}
	_, err = NewTeamsWebhookChannel(bc, &cfg, msgBus)
	if err == nil {
		t.Error("expected error for missing 'default' webhook")
	}

	// Test empty webhook URL
	cfg.Webhooks = map[string]config.TeamsWebhookTarget{
		"default": {Title: "Default"},
	}
	_, err = NewTeamsWebhookChannel(bc, &cfg, msgBus)
	if err == nil {
		t.Error("expected error for empty webhook_url")
	}

	// Test HTTP URL (should fail, must be HTTPS)
	cfg.Webhooks = map[string]config.TeamsWebhookTarget{
		"default": {
			WebhookURL: *config.NewSecureString("http://example.com/webhook"),
			Title:      "Default",
		},
	}
	_, err = NewTeamsWebhookChannel(bc, &cfg, msgBus)
	if err == nil {
		t.Error("expected error for HTTP webhook URL (must be HTTPS)")
	}

	// Test valid config with HTTPS (must include "default")
	cfg.Webhooks = map[string]config.TeamsWebhookTarget{
		"default": {
			WebhookURL: *config.NewSecureString("https://example.com/webhook-default"),
			Title:      "Default",
		},
		"alerts": {
			WebhookURL: *config.NewSecureString("https://example.com/webhook1"),
			Title:      "Alerts",
		},
	}
	ch, err := NewTeamsWebhookChannel(bc, &cfg, msgBus)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ch.Name() != "teams_webhook" {
		t.Errorf("expected name 'teams_webhook', got %q", ch.Name())
	}
}

func TestTeamsWebhookChannel_StartStop(t *testing.T) {
	msgBus := bus.NewMessageBus()
	bc := &config.Channel{Type: config.ChannelTeamsWebHook, Enabled: true}
	cfg := config.TeamsWebhookSettings{
		Webhooks: map[string]config.TeamsWebhookTarget{
			"default": {
				WebhookURL: *config.NewSecureString("https://example.com/webhook"),
			},
		},
	}
	ch, err := NewTeamsWebhookChannel(bc, &cfg, msgBus)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()

	if ch.IsRunning() {
		t.Error("channel should not be running before Start")
	}

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !ch.IsRunning() {
		t.Error("channel should be running after Start")
	}

	if err := ch.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if ch.IsRunning() {
		t.Error("channel should not be running after Stop")
	}
}

func TestTeamsWebhookChannel_BuildAdaptiveCard(t *testing.T) {
	msgBus := bus.NewMessageBus()
	bc := &config.Channel{Type: config.ChannelTeamsWebHook, Enabled: true}
	cfg := config.TeamsWebhookSettings{
		Webhooks: map[string]config.TeamsWebhookTarget{
			"default": {
				WebhookURL: *config.NewSecureString("https://example.com/webhook-default"),
				Title:      "Default",
			},
			"alerts": {
				WebhookURL: *config.NewSecureString("https://example.com/webhook"),
				Title:      "Custom Title",
			},
		},
	}
	ch, err := NewTeamsWebhookChannel(bc, &cfg, msgBus)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	target := ch.config.Webhooks["alerts"]
	msg := bus.OutboundMessage{
		Content: "Test message content",
		ChatID:  "alerts",
	}

	card, err := ch.buildAdaptiveCard(msg, target)
	if err != nil {
		t.Fatalf("buildAdaptiveCard failed: %v", err)
	}

	if card.Type != "AdaptiveCard" {
		t.Errorf("expected card type 'AdaptiveCard', got %q", card.Type)
	}
}

func TestTeamsWebhookChannel_SendNotRunning(t *testing.T) {
	msgBus := bus.NewMessageBus()
	bc := &config.Channel{Type: config.ChannelTeamsWebHook, Enabled: true}
	cfg := config.TeamsWebhookSettings{
		Webhooks: map[string]config.TeamsWebhookTarget{
			"default": {
				WebhookURL: *config.NewSecureString("https://example.com/webhook"),
			},
		},
	}
	ch, err := NewTeamsWebhookChannel(bc, &cfg, msgBus)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	msg := bus.OutboundMessage{Content: "test", ChatID: "default"}

	_, err = ch.Send(ctx, msg)
	if err == nil {
		t.Error("expected error when sending while not running")
	}
}

func TestTeamsWebhookChannel_SendDefaultTargetFallback(t *testing.T) {
	tests := []struct {
		name   string
		chatID string
	}{
		{"unknown target falls back to default", "unknown"},
		{"empty ChatID uses default", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgBus := bus.NewMessageBus()
			bc := &config.Channel{Type: config.ChannelTeamsWebHook, Enabled: true}
			cfg := config.TeamsWebhookSettings{
				Webhooks: map[string]config.TeamsWebhookTarget{
					"default": {
						WebhookURL: *config.NewSecureString("https://example.com/webhook-default"),
					},
					"alerts": {
						WebhookURL: *config.NewSecureString("https://example.com/webhook-alerts"),
					},
				},
			}
			ch, err := NewTeamsWebhookChannel(bc, &cfg, msgBus)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var sentURL string
			ch.client = &mockTeamsClient{
				sendFunc: func(ctx context.Context, webhookURL string, message goteamsnotify.TeamsMessage) error {
					sentURL = webhookURL
					return nil
				},
			}

			ctx := context.Background()
			_ = ch.Start(ctx)
			defer ch.Stop(ctx)

			msg := bus.OutboundMessage{Content: "test", ChatID: tt.chatID}
			_, err = ch.Send(ctx, msg)
			if err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}

			if sentURL != "https://example.com/webhook-default" {
				t.Errorf("expected default webhook URL, got %q", sentURL)
			}
		})
	}
}

func TestTeamsWebhookChannel_SendSuccess(t *testing.T) {
	msgBus := bus.NewMessageBus()
	bc := &config.Channel{Type: config.ChannelTeamsWebHook, Enabled: true}
	cfg := config.TeamsWebhookSettings{
		Webhooks: map[string]config.TeamsWebhookTarget{
			"default": {
				WebhookURL: *config.NewSecureString("https://example.com/webhook-default"),
				Title:      "Default",
			},
			"alerts": {
				WebhookURL: *config.NewSecureString("https://example.com/webhook-alerts"),
				Title:      "Test Alerts",
			},
		},
	}
	ch, err := NewTeamsWebhookChannel(bc, &cfg, msgBus)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Inject mock client
	var sentURL string
	ch.client = &mockTeamsClient{
		sendFunc: func(ctx context.Context, webhookURL string, message goteamsnotify.TeamsMessage) error {
			sentURL = webhookURL
			return nil
		},
	}

	ctx := context.Background()
	_ = ch.Start(ctx)
	defer ch.Stop(ctx)

	msg := bus.OutboundMessage{Content: "Hello Teams!", ChatID: "alerts"}

	_, err = ch.Send(ctx, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sentURL != "https://example.com/webhook-alerts" {
		t.Errorf("expected webhook URL 'https://example.com/webhook-alerts', got %q", sentURL)
	}
}

func TestTeamsWebhookChannel_SendError(t *testing.T) {
	msgBus := bus.NewMessageBus()
	bc := &config.Channel{Type: config.ChannelTeamsWebHook, Enabled: true}
	cfg := config.TeamsWebhookSettings{
		Webhooks: map[string]config.TeamsWebhookTarget{
			"default": {
				WebhookURL: *config.NewSecureString("https://example.com/webhook-default"),
			},
			"alerts": {
				WebhookURL: *config.NewSecureString("https://example.com/webhook-alerts"),
			},
		},
	}
	ch, err := NewTeamsWebhookChannel(bc, &cfg, msgBus)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Inject mock client that returns an error
	ch.client = &mockTeamsClient{
		sendFunc: func(ctx context.Context, webhookURL string, message goteamsnotify.TeamsMessage) error {
			return errors.New("error on notification: 401 Unauthorized, forbidden")
		},
	}

	ctx := context.Background()
	_ = ch.Start(ctx)
	defer ch.Stop(ctx)

	msg := bus.OutboundMessage{Content: "test", ChatID: "alerts"}

	_, err = ch.Send(ctx, msg)
	if err == nil {
		t.Error("expected error from failed send")
	}
}

func TestSplitContentWithTables(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantSegs int
		wantTbl  int // number of table segments
	}{
		{
			name:     "no tables",
			content:  "Just some text\nwith multiple lines",
			wantSegs: 1,
			wantTbl:  0,
		},
		{
			name: "single table",
			content: `| Col1 | Col2 |
|------|------|
| A    | B    |
| C    | D    |`,
			wantSegs: 1,
			wantTbl:  1,
		},
		{
			name: "text before table",
			content: `Here is some text.

| Col1 | Col2 |
|------|------|
| A    | B    |`,
			wantSegs: 2,
			wantTbl:  1,
		},
		{
			name: "text before and after table",
			content: `Before table.

| Col1 | Col2 |
|------|------|
| A    | B    |

After table.`,
			wantSegs: 3,
			wantTbl:  1,
		},
		{
			name: "multiple tables",
			content: `First table:

| A | B |
|---|---|
| 1 | 2 |

Second table:

| X | Y |
|---|---|
| 3 | 4 |`,
			wantSegs: 4,
			wantTbl:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segs := splitContentWithTables(tt.content)
			if len(segs) != tt.wantSegs {
				t.Errorf("got %d segments, want %d", len(segs), tt.wantSegs)
			}
			tableCount := 0
			for _, s := range segs {
				if s.isTable {
					tableCount++
				}
			}
			if tableCount != tt.wantTbl {
				t.Errorf("got %d tables, want %d", tableCount, tt.wantTbl)
			}
		})
	}
}

func TestParseMarkdownTable(t *testing.T) {
	tableStr := `| Name | Value |
|------|-------|
| foo  | 123   |
| bar  | 456   |`

	elem, err := parseMarkdownTable(tableStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if elem.Type != "Table" {
		t.Errorf("expected type 'Table', got %q", elem.Type)
	}

	// Should have 3 rows (header + 2 data rows)
	if len(elem.Rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(elem.Rows))
	}

	// Should have 2 columns with widths based on content length
	if len(elem.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(elem.Columns))
	}
}

func TestParseMarkdownTableColumnWidths(t *testing.T) {
	// Column widths are based on HEADER row only:
	// Col1: "Description" (11 chars)
	// Col2: "X" (1 char)
	// Col3: "Amount" (6 chars)
	tableStr := `| Description | X | Amount |
|-------------|---|--------|
| Short       | Y | 100    |
| Longer text | Z | 50     |`

	elem, err := parseMarkdownTable(tableStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(elem.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(elem.Columns))
	}

	// Verify column widths are based on header content length
	w1, ok1 := elem.Columns[0].Width.(int)
	w2, ok2 := elem.Columns[1].Width.(int)
	w3, ok3 := elem.Columns[2].Width.(int)

	if !ok1 || !ok2 || !ok3 {
		t.Fatalf("expected int widths, got types: %T, %T, %T",
			elem.Columns[0].Width, elem.Columns[1].Width, elem.Columns[2].Width)
	}

	// Header lengths: "Description" = 11, "X" = 1, "Amount" = 6
	if w1 != 11 {
		t.Errorf("expected col1 width 11 (from 'Description'), got %d", w1)
	}
	if w2 != 1 {
		t.Errorf("expected col2 width 1 (from 'X'), got %d", w2)
	}
	if w3 != 6 {
		t.Errorf("expected col3 width 6 (from 'Amount'), got %d", w3)
	}
}

func TestCalculateColumnWidths(t *testing.T) {
	tests := []struct {
		name       string
		maxLengths []int
		wantWidths []int
	}{
		{
			name:       "equal lengths",
			maxLengths: []int{10, 10, 10},
			wantWidths: []int{10, 10, 10},
		},
		{
			name:       "varying lengths",
			maxLengths: []int{5, 20, 10},
			wantWidths: []int{5, 20, 10},
		},
		{
			name:       "zero length gets minimum of 1",
			maxLengths: []int{0, 5, 0},
			wantWidths: []int{1, 5, 1},
		},
		{
			name:       "empty input",
			maxLengths: []int{},
			wantWidths: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols := calculateColumnWidths(tt.maxLengths)

			if tt.wantWidths == nil {
				if cols != nil {
					t.Errorf("expected nil, got %v", cols)
				}
				return
			}

			if len(cols) != len(tt.wantWidths) {
				t.Fatalf("expected %d columns, got %d", len(tt.wantWidths), len(cols))
			}

			for i, col := range cols {
				width, ok := col.Width.(int)
				if !ok {
					t.Errorf("column %d: expected int width, got %T", i, col.Width)
					continue
				}
				if width != tt.wantWidths[i] {
					t.Errorf("column %d: expected width %d, got %d", i, tt.wantWidths[i], width)
				}
				if col.Type != "TableColumnDefinition" {
					t.Errorf("column %d: expected type 'TableColumnDefinition', got %q", i, col.Type)
				}
			}
		})
	}
}

func TestParseTableRow(t *testing.T) {
	tests := []struct {
		line string
		want []string
	}{
		{"| A | B | C |", []string{"A", "B", "C"}},
		{"|A|B|C|", []string{"A", "B", "C"}},
		{"| foo | bar |", []string{"foo", "bar"}},
		{"", nil},
	}

	for _, tt := range tests {
		got := parseTableRow(tt.line)
		if len(got) != len(tt.want) {
			t.Errorf("parseTableRow(%q): got %v, want %v", tt.line, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseTableRow(%q)[%d]: got %q, want %q", tt.line, i, got[i], tt.want[i])
			}
		}
	}
}

func TestIsSeparatorRow(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"|---|---|", true},
		{"| --- | --- |", true},
		{"|:---|---:|", true},
		{"| :---: | :---: |", true},
		{"| A | B |", false},
		{"| foo | bar |", false},
	}

	for _, tt := range tests {
		got := isSeparatorRow(tt.line)
		if got != tt.want {
			t.Errorf("isSeparatorRow(%q): got %v, want %v", tt.line, got, tt.want)
		}
	}
}
