package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/sipeed/picoclaw/pkg/config"
)

type CalendarTool struct {
	config  config.CalendarConfig
	service *calendar.Service
}

func NewCalendarTool(cfg config.CalendarConfig) *CalendarTool {
	return &CalendarTool{
		config: cfg,
	}
}

func (t *CalendarTool) Name() string {
	return "calendar"
}

func (t *CalendarTool) Description() string {
	return "Manage Google Calendar events. Can list upcoming events and add new events."
}

func (t *CalendarTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "Action to perform: list_events, add_event",
				"enum":        []string{"list_events", "add_event"},
			},
			"count": map[string]interface{}{
				"type":        "integer",
				"description": "Number of events to list (default: 10). For list_events.",
			},
			"summary": map[string]interface{}{
				"type":        "string",
				"description": "Event title/summary. For add_event.",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Event description. For add_event.",
			},
			"location": map[string]interface{}{
				"type":        "string",
				"description": "Event location. For add_event.",
			},
			"start_time": map[string]interface{}{
				"type":        "string",
				"description": "Start time in RFC3339 format (e.g. 2023-10-27T10:00:00Z). For add_event.",
			},
			"end_time": map[string]interface{}{
				"type":        "string",
				"description": "End time in RFC3339 format. For add_event. If omitted, defaults to 1 hour after start.",
			},
		},
		"required": []string{"action"},
	}
}

func (t *CalendarTool) getService(ctx context.Context) (*calendar.Service, error) {
	if t.service != nil {
		return t.service, nil
	}

	if t.config.CredentialsJSON == "" {
		return nil, fmt.Errorf("calendar credentials_json not configured")
	}

	// Expand home directory if needed
	credPath := t.config.CredentialsJSON
	if strings.HasPrefix(credPath, "~/") {
		home, _ := os.UserHomeDir()
		credPath = filepathJoin(home, credPath[2:])
	}

	b, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	// If using Service Account
	// conf, err := google.JWTConfigFromJSON(b, calendar.CalendarScope)

	// If using OAuth2 Client ID (more common for personal calendars)
	// We need a token. For a CLI tool, we might need a stored token.
	// Implementing robust OAuth flow in a tool is hard.
	// Let's assume Service Account for now as it's easier for server-side,
	// BUT Service Accounts can't access personal Gmail calendars without Domain-Wide Delegation (Workspace only).
	// For personal Gmail, we need OAuth2 User Credentials.

	// Strategy: Use "Application Default Credentials" or specific OAuth token if provided.
	// Simplest for personal: User provides `token.json` generated elsewhere, or we use a Service Account shared with the personal email?
	// Sharing personal calendar with Service Account email is the easiest way!
	// 1. User creates Service Account.
	// 2. User shares their calendar with Service Account email.
	// 3. Tool uses Service Account credentials.

	config, err := google.JWTConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		// Try standard credentials (could be OAuth client secret)
		// But for now let's stick to Service Account as primary recommendation for headless agents.
		return nil, fmt.Errorf("unable to parse service account key file: %v. Please ensure you are using a Service Account key.", err)
	}

	client := config.Client(ctx)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Calendar client: %v", err)
	}

	t.service = srv
	return srv, nil
}

func (t *CalendarTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if !t.config.Enabled {
		return &ToolResult{Err: fmt.Errorf("calendar tool is disabled in config")}
	}

	action, _ := args["action"].(string)
	srv, err := t.getService(ctx)
	if err != nil {
		return &ToolResult{Err: err}
	}

	switch action {
	case "list_events":
		return t.listEvents(srv, args)
	case "add_event":
		return t.addEvent(srv, args)
	default:
		return &ToolResult{Err: fmt.Errorf("unknown action: %s", action)}
	}
}

func (t *CalendarTool) listEvents(srv *calendar.Service, args map[string]interface{}) *ToolResult {
	count := 10
	if c, ok := args["count"].(float64); ok {
		count = int(c)
	}

	tMin := time.Now().Format(time.RFC3339)

	calendarId := "primary"
	if t.config.CalendarID != "" {
		calendarId = t.config.CalendarID
	}

	events, err := srv.Events.List(calendarId).ShowDeleted(false).
		SingleEvents(true).TimeMin(tMin).MaxResults(int64(count)).OrderBy("startTime").Do()
	if err != nil {
		return &ToolResult{Err: fmt.Errorf("unable to retrieve next ten of the user's upcoming events: %v", err)}
	}

	if len(events.Items) == 0 {
		return &ToolResult{ForLLM: "No upcoming events found."}
	}

	var sb strings.Builder
	sb.WriteString("Upcoming events:\n")
	for _, item := range events.Items {
		date := item.Start.DateTime
		if date == "" {
			date = item.Start.Date
		}
		sb.WriteString(fmt.Sprintf("- %s (%s)\n", item.Summary, date))
	}

	return &ToolResult{ForLLM: sb.String(), ForUser: sb.String()}
}

func (t *CalendarTool) addEvent(srv *calendar.Service, args map[string]interface{}) *ToolResult {
	summary, _ := args["summary"].(string)
	description, _ := args["description"].(string)
	location, _ := args["location"].(string)
	startTimeStr, _ := args["start_time"].(string)
	endTimeStr, _ := args["end_time"].(string)

	if summary == "" || startTimeStr == "" {
		return &ToolResult{Err: fmt.Errorf("summary and start_time are required")}
	}

	// Parse start time
	// Try RFC3339 first
	// If failed, maybe try other formats? LLM usually gives ISO/RFC.

	event := &calendar.Event{
		Summary:     summary,
		Location:    location,
		Description: description,
		Start: &calendar.EventDateTime{
			DateTime: startTimeStr,
			TimeZone: "UTC", // Should ideally be user's timezone or inferred
		},
		End: &calendar.EventDateTime{
			DateTime: endTimeStr,
			TimeZone: "UTC",
		},
	}

	if endTimeStr == "" {
		// Default to 1 hour later
		t, err := time.Parse(time.RFC3339, startTimeStr)
		if err == nil {
			event.End.DateTime = t.Add(1 * time.Hour).Format(time.RFC3339)
		} else {
			return &ToolResult{Err: fmt.Errorf("invalid start_time format, expected RFC3339: %v", err)}
		}
	}

	calendarId := "primary"
	if t.config.CalendarID != "" {
		calendarId = t.config.CalendarID
	}

	event, err := srv.Events.Insert(calendarId, event).Do()
	if err != nil {
		return &ToolResult{Err: fmt.Errorf("unable to create event: %v", err)}
	}

	msg := fmt.Sprintf("Event created: %s (%s)", event.HtmlLink, event.Id)
	return &ToolResult{ForLLM: msg, ForUser: fmt.Sprintf("✅ Created event '%s' at %s", summary, startTimeStr)}
}

// Helper to avoid circular dependency or just duplicating logic
func filepathJoin(elem ...string) string {
	for i, e := range elem {
		if e != "" {
			return strings.Join(elem[i:], string(os.PathSeparator))
		}
	}
	return ""
}
