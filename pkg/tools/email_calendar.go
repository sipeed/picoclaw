package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// SendCalendarInviteTool sends an iCalendar (.ics) meeting invite via Resend.
type SendCalendarInviteTool struct {
	cfg *config.ResendConfig
}

func NewSendCalendarInviteTool(cfg *config.ResendConfig) *SendCalendarInviteTool {
	return &SendCalendarInviteTool{cfg: cfg}
}

func (t *SendCalendarInviteTool) Name() string {
	return "send_calendar_invite"
}

func (t *SendCalendarInviteTool) Description() string {
	return "Send a calendar invitation (.ics file) via email to a recipient. Use this when the user wants to schedule a meeting, set up an appointment, or send a calendar event invite."
}

func (t *SendCalendarInviteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"to": map[string]any{
				"type":        "string",
				"description": "Recipient email address",
			},
			"summary": map[string]any{
				"type":        "string",
				"description": "Event title or subject",
			},
			"start": map[string]any{
				"type":        "string",
				"description": "Event start time in ISO 8601 format (e.g. 2026-03-10T14:00:00 or 2026-03-10T14:00:00Z)",
			},
			"end": map[string]any{
				"type":        "string",
				"description": "Event end time in ISO 8601 format (e.g. 2026-03-10T15:00:00 or 2026-03-10T15:00:00Z)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Optional meeting notes or agenda",
			},
		},
		"required": []string{"to", "summary", "start", "end"},
	}
}

func (t *SendCalendarInviteTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	to, _ := args["to"].(string)
	summary, _ := args["summary"].(string)
	start, _ := args["start"].(string)
	end, _ := args["end"].(string)
	description, _ := args["description"].(string)

	if to == "" || summary == "" || start == "" || end == "" {
		return ErrorResult("to, summary, start, and end are required")
	}

	ics, err := generateICS(summary, start, end, description)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to generate calendar invite: %v", err))
	}

	if err := sendViaResend(ctx, t.cfg, to, summary, ics); err != nil {
		return ErrorResult(fmt.Sprintf("failed to send calendar invite: %v", err))
	}

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Calendar invite for '%s' sent successfully to %s", summary, to),
		ForUser: fmt.Sprintf("Calendar invite sent to %s", to),
	}
}

// generateICS creates an RFC 5545-compliant iCalendar payload.
func generateICS(summary, startStr, endStr, description string) ([]byte, error) {
	startTime, err := parseISOTime(startStr)
	if err != nil {
		return nil, fmt.Errorf("invalid start time %q: %w", startStr, err)
	}
	endTime, err := parseISOTime(endStr)
	if err != nil {
		return nil, fmt.Errorf("invalid end time %q: %w", endStr, err)
	}

	uid := fmt.Sprintf("%d@picoclaw.ai", time.Now().UnixNano())
	now := time.Now().UTC().Format("20060102T150405Z")
	dtstart := startTime.UTC().Format("20060102T150405Z")
	dtend := endTime.UTC().Format("20060102T150405Z")

	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("PRODID:-//PicoClaw AI//picoclaw.ai//\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("METHOD:REQUEST\r\n")
	b.WriteString("BEGIN:VEVENT\r\n")
	fmt.Fprintf(&b, "UID:%s\r\n", uid)
	fmt.Fprintf(&b, "DTSTAMP:%s\r\n", now)
	fmt.Fprintf(&b, "DTSTART:%s\r\n", dtstart)
	fmt.Fprintf(&b, "DTEND:%s\r\n", dtend)
	fmt.Fprintf(&b, "SUMMARY:%s\r\n", escapeICSText(summary))
	if description != "" {
		fmt.Fprintf(&b, "DESCRIPTION:%s\r\n", escapeICSText(description))
	}
	b.WriteString("END:VEVENT\r\n")
	b.WriteString("END:VCALENDAR\r\n")
	return []byte(b.String()), nil
}

func parseISOTime(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised time format; use ISO 8601 (e.g. 2026-03-10T14:00:00)")
}

func escapeICSText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

type resendEmailRequest struct {
	From        string             `json:"from"`
	To          []string           `json:"to"`
	Subject     string             `json:"subject"`
	HTML        string             `json:"html"`
	Attachments []resendAttachment `json:"attachments"`
}

type resendAttachment struct {
	Filename string `json:"filename"`
	Content  []byte `json:"content"` // marshalled as base64 by encoding/json
}

func sendViaResend(ctx context.Context, cfg *config.ResendConfig, to, subject string, ics []byte) error {
	payload := resendEmailRequest{
		From:    cfg.FromAddress,
		To:      []string{to},
		Subject: subject,
		HTML:    fmt.Sprintf("<p>You have been invited to: <strong>%s</strong></p>", subject),
		Attachments: []resendAttachment{
			{Filename: "invite.ics", Content: ics},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API error %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}
	return nil
}
