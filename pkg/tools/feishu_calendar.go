package tools

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcalendar "github.com/larksuite/oapi-sdk-go/v3/service/calendar/v4"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const defaultFeishuCalendarTimezone = "Asia/Shanghai"

// FeishuCalendarTool creates calendar events in Feishu/Lark.
type FeishuCalendarTool struct {
	cfg    config.FeishuConfig
	client *lark.Client
}

func NewFeishuCalendarTool(cfg config.FeishuConfig) *FeishuCalendarTool {
	return &FeishuCalendarTool{
		cfg:    cfg,
		client: lark.NewClient(cfg.AppID, cfg.AppSecret),
	}
}

func (t *FeishuCalendarTool) Name() string {
	return "feishu_calendar"
}

func (t *FeishuCalendarTool) Description() string {
	return "Create events in Feishu/Lark calendar. Use this when user asks to add a calendar item, schedule, or agenda in Feishu."
}

func (t *FeishuCalendarTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"summary": map[string]any{
				"type":        "string",
				"description": "Event title/summary.",
			},
			"start_time": map[string]any{
				"type":        "string",
				"description": "Start time. Prefer RFC3339 (e.g. 2026-02-25T14:00:00+08:00). Also supports 'YYYY-MM-DD HH:MM'.",
			},
			"end_time": map[string]any{
				"type":        "string",
				"description": "Optional end time. If omitted, duration_minutes is used.",
			},
			"duration_minutes": map[string]any{
				"type":        "integer",
				"description": "Duration in minutes when end_time is omitted. Default 30.",
			},
			"timezone": map[string]any{
				"type":        "string",
				"description": "Optional IANA timezone (e.g. Asia/Shanghai, UTC). Used for local datetime inputs.",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Optional event description.",
			},
			"calendar_id": map[string]any{
				"type":        "string",
				"description": "Optional calendar ID. If omitted, tool uses primary calendar.",
			},
			"location_name": map[string]any{
				"type":        "string",
				"description": "Optional event location name.",
			},
			"location_address": map[string]any{
				"type":        "string",
				"description": "Optional event location address.",
			},
			"attendee_user_ids": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional attendee user_id list.",
			},
			"reminder_minutes": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "integer"},
				"description": "Optional reminder minutes before start (e.g. [10, 30]).",
			},
			"recurrence": map[string]any{
				"type":        "string",
				"description": "Optional RFC5545 recurrence rule, e.g. FREQ=DAILY;INTERVAL=1.",
			},
			"need_notification": map[string]any{
				"type":        "boolean",
				"description": "Whether to notify attendees by bot notification. Default true.",
			},
		},
		"required": []string{"summary", "start_time"},
	}
}

func (t *FeishuCalendarTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if strings.TrimSpace(t.cfg.AppID) == "" || strings.TrimSpace(t.cfg.AppSecret) == "" {
		return ErrorResult("feishu app_id/app_secret is missing in config.channels.feishu")
	}

	summary, ok := getStringArg(args, "summary")
	if !ok || strings.TrimSpace(summary) == "" {
		return ErrorResult("summary is required")
	}
	summary = strings.TrimSpace(summary)

	startRaw, ok := getStringArg(args, "start_time")
	if !ok || strings.TrimSpace(startRaw) == "" {
		return ErrorResult("start_time is required")
	}

	tzInput, _ := getStringArg(args, "timezone")
	tzInput = strings.TrimSpace(tzInput)
	loc := defaultFeishuLocation()
	if tzInput != "" {
		var err error
		loc, err = time.LoadLocation(tzInput)
		if err != nil {
			return ErrorResult(fmt.Sprintf("invalid timezone %q, use IANA timezone like Asia/Shanghai", tzInput))
		}
	}

	start, startHasExplicitTZ, err := parseFeishuDateTime(strings.TrimSpace(startRaw), loc)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid start_time: %v", err))
	}

	var end time.Time
	if endRaw, ok := getStringArg(args, "end_time"); ok && strings.TrimSpace(endRaw) != "" {
		end, _, err = parseFeishuDateTime(strings.TrimSpace(endRaw), loc)
		if err != nil {
			return ErrorResult(fmt.Sprintf("invalid end_time: %v", err))
		}
	} else {
		duration, err := parseOptionalIntArg(args, "duration_minutes", 30, 1, 24*60*30)
		if err != nil {
			return ErrorResult(err.Error())
		}
		end = start.Add(time.Duration(duration) * time.Minute)
	}

	if !end.After(start) {
		return ErrorResult("end time must be after start time")
	}

	calendarID, _ := getStringArg(args, "calendar_id")
	calendarID = strings.TrimSpace(calendarID)
	if shouldUseFeishuPrimaryCalendar(calendarID) {
		resolvedID, err := t.resolvePrimaryCalendarID(ctx)
		if err != nil {
			return ErrorResult(fmt.Sprintf("resolve primary calendar failed: %v", err))
		}
		calendarID = resolvedID
	}

	eventTZ, err := resolveFeishuEventTimezone(tzInput, start, startHasExplicitTZ)
	if err != nil {
		return ErrorResult(err.Error())
	}

	eventBuilder := larkcalendar.NewCalendarEventBuilder().
		Summary(summary).
		StartTime(buildFeishuTimeInfo(start, eventTZ)).
		EndTime(buildFeishuTimeInfo(end, eventTZ))

	if description, ok := getStringArg(args, "description"); ok && strings.TrimSpace(description) != "" {
		eventBuilder.Description(strings.TrimSpace(description))
	}

	locationName, _ := getStringArg(args, "location_name")
	locationAddress, _ := getStringArg(args, "location_address")
	locationName = strings.TrimSpace(locationName)
	locationAddress = strings.TrimSpace(locationAddress)
	if locationName != "" || locationAddress != "" {
		locationBuilder := larkcalendar.NewEventLocationBuilder()
		if locationName != "" {
			locationBuilder.Name(locationName)
		}
		if locationAddress != "" {
			locationBuilder.Address(locationAddress)
		}
		eventBuilder.Location(locationBuilder.Build())
	}

	if recurrence, ok := getStringArg(args, "recurrence"); ok && strings.TrimSpace(recurrence) != "" {
		eventBuilder.Recurrence(strings.TrimSpace(recurrence))
	}

	attendeeIDs, err := parseStringSliceArg(args, "attendee_user_ids")
	if err != nil {
		return ErrorResult(err.Error())
	}
	inviteeUserIDs := buildFeishuInviteeUserIDs(attendeeIDs, toolExecutionChannel(ctx), toolExecutionSenderID(ctx))

	reminderMinutes, err := parseReminderMinutesArg(args, "reminder_minutes")
	if err != nil {
		return ErrorResult(err.Error())
	}
	if len(reminderMinutes) > 0 {
		reminders := make([]*larkcalendar.Reminder, 0, len(reminderMinutes))
		for _, minutes := range reminderMinutes {
			reminders = append(reminders, larkcalendar.NewReminderBuilder().Minutes(minutes).Build())
		}
		eventBuilder.Reminders(reminders)
	}

	needNotification, err := parseBoolArg(args, "need_notification", true)
	if err != nil {
		return ErrorResult(err.Error())
	}
	eventBuilder.NeedNotification(needNotification)

	req := larkcalendar.NewCreateCalendarEventReqBuilder().
		CalendarId(calendarID).
		UserIdType(larkcalendar.UserIdTypeCreateCalendarEventUserId).
		IdempotencyKey(generateFeishuIdempotencyKey()).
		CalendarEvent(eventBuilder.Build()).
		Build()

	resp, err := t.client.Calendar.V4.CalendarEvent.Create(ctx, req)
	if err != nil {
		logger.ErrorCF("tools.feishu_calendar", "Create Feishu calendar event request failed", map[string]any{
			"calendar_id": calendarID,
			"error":       err.Error(),
		})
		return ErrorResult(fmt.Sprintf("create feishu calendar event failed: %v", err))
	}
	if !resp.Success() {
		fields := map[string]any{
			"calendar_id": calendarID,
			"code":        resp.Code,
			"msg":         resp.Msg,
		}
		if len(resp.RawBody) > 0 {
			fields["raw_body"] = string(resp.RawBody)
		}
		if resp.Code == 99991672 {
			fields["hint"] = "missing app scopes: enable calendar and calendar event write scopes, then publish app version"
		}
		logger.ErrorCF("tools.feishu_calendar", "Feishu calendar event rejected", fields)
		return ErrorResult(fmt.Sprintf("feishu calendar api error: code=%d msg=%s", resp.Code, resp.Msg))
	}

	eventID := ""
	eventLink := ""
	if resp.Data != nil && resp.Data.Event != nil {
		if resp.Data.Event.EventId != nil {
			eventID = *resp.Data.Event.EventId
		}
		if resp.Data.Event.AppLink != nil {
			eventLink = *resp.Data.Event.AppLink
		}
	}

	attendeesAdded := len(inviteeUserIDs) == 0
	attendeeWarning := ""
	if len(inviteeUserIDs) > 0 {
		if strings.TrimSpace(eventID) == "" {
			attendeeWarning = "event_id missing, unable to add attendees"
		} else if err := t.addFeishuEventAttendees(ctx, calendarID, eventID, inviteeUserIDs, needNotification); err != nil {
			attendeeWarning = err.Error()
		} else {
			attendeesAdded = true
		}
	}
	if attendeeWarning != "" {
		eventLink = ""
		logger.WarnCF("tools.feishu_calendar", "Feishu attendee auto-add failed", map[string]any{
			"calendar_id": calendarID,
			"event_id":    eventID,
			"sender_id":   toolExecutionSenderID(ctx),
			"warning":     attendeeWarning,
		})
	}

	logger.InfoCF("tools.feishu_calendar", "Feishu calendar event created", map[string]any{
		"calendar_id": calendarID,
		"event_id":    eventID,
		"start_unix":  start.Unix(),
		"end_unix":    end.Unix(),
		"attendees":   len(inviteeUserIDs),
		"added":       attendeesAdded,
	})

	result := fmt.Sprintf(
		"Feishu calendar event created: %s (%s to %s, calendar_id=%s)",
		summary,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
		calendarID,
	)
	if eventID != "" {
		result += fmt.Sprintf(", event_id=%s", eventID)
	}
	if eventLink != "" {
		result += fmt.Sprintf(", link=%s", eventLink)
	}
	if attendeeWarning != "" {
		result += fmt.Sprintf(", warning=%s", attendeeWarning)
	}

	return SilentResult(result)
}

func (t *FeishuCalendarTool) resolvePrimaryCalendarID(ctx context.Context) (string, error) {
	resp, err := t.client.Calendar.V4.Calendar.Primary(
		ctx,
		larkcalendar.NewPrimaryCalendarReqBuilder().Build(),
	)
	if err != nil {
		return "", fmt.Errorf("call primary calendar api: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("feishu calendar primary api error: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Data == nil || len(resp.Data.Calendars) == 0 {
		return "", fmt.Errorf("no primary calendar returned")
	}

	for _, userCalendar := range resp.Data.Calendars {
		if userCalendar == nil || userCalendar.Calendar == nil || userCalendar.Calendar.CalendarId == nil {
			continue
		}
		calendarID := strings.TrimSpace(*userCalendar.Calendar.CalendarId)
		if calendarID != "" {
			return calendarID, nil
		}
	}

	return "", fmt.Errorf("primary calendar id is empty")
}

func parseFeishuDateTime(raw string, loc *time.Location) (time.Time, bool, error) {
	if raw == "" {
		return time.Time{}, false, fmt.Errorf("empty datetime")
	}

	if unix, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if loc == nil {
			return time.Unix(unix, 0), false, nil
		}
		return time.Unix(unix, 0).In(loc), false, nil
	}

	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, true, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, true, nil
	}

	if loc == nil {
		loc = defaultFeishuLocation()
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, loc); err == nil {
			return t, false, nil
		}
	}

	return time.Time{}, false, fmt.Errorf("unsupported datetime format: %q", raw)
}

func resolveFeishuEventTimezone(requested string, parsed time.Time, hasExplicitTZ bool) (string, error) {
	if requested != "" {
		if _, err := time.LoadLocation(requested); err != nil {
			return "", fmt.Errorf("invalid timezone %q, use IANA timezone like Asia/Shanghai", requested)
		}
		return requested, nil
	}

	if hasExplicitTZ {
		_, offset := parsed.Zone()
		if offset == 0 {
			return "UTC", nil
		}
		if offset%3600 == 0 && offset >= -14*3600 && offset <= 14*3600 {
			hours := offset / 3600
			if hours > 0 {
				return fmt.Sprintf("Etc/GMT-%d", hours), nil
			}
			return fmt.Sprintf("Etc/GMT+%d", -hours), nil
		}
		return "UTC", nil
	}

	return defaultFeishuCalendarTimezone, nil
}

func buildFeishuTimeInfo(ts time.Time, timezone string) *larkcalendar.TimeInfo {
	builder := larkcalendar.NewTimeInfoBuilder().
		Timestamp(strconv.FormatInt(ts.Unix(), 10))
	if strings.TrimSpace(timezone) != "" {
		builder.Timezone(strings.TrimSpace(timezone))
	}
	return builder.Build()
}

func parseReminderMinutesArg(args map[string]any, key string) ([]int, error) {
	raw, exists := args[key]
	if !exists {
		return nil, nil
	}

	normalize := func(v any) (int, error) {
		n, err := toInt(v)
		if err != nil {
			return 0, fmt.Errorf("%s must be integers", key)
		}
		if n < 0 || n > 525600 {
			return 0, fmt.Errorf("%s value %d is out of range [0, 525600]", key, n)
		}
		return n, nil
	}

	values := make([]int, 0)
	switch typed := raw.(type) {
	case []any:
		for _, item := range typed {
			n, err := normalize(item)
			if err != nil {
				return nil, err
			}
			values = append(values, n)
		}
	case []int:
		for _, item := range typed {
			n, err := normalize(item)
			if err != nil {
				return nil, err
			}
			values = append(values, n)
		}
	default:
		n, err := normalize(typed)
		if err != nil {
			return nil, fmt.Errorf("%s must be an integer or array of integers", key)
		}
		values = append(values, n)
	}

	slices.Sort(values)
	return slices.Compact(values), nil
}

func defaultFeishuLocation() *time.Location {
	loc, err := time.LoadLocation(defaultFeishuCalendarTimezone)
	if err != nil {
		return time.Local
	}
	return loc
}

func shouldUseFeishuPrimaryCalendar(calendarID string) bool {
	calendarID = strings.TrimSpace(calendarID)
	return calendarID == "" || strings.EqualFold(calendarID, "primary")
}

func generateFeishuIdempotencyKey() string {
	return uuid.NewString()
}

func buildFeishuInviteeUserIDs(
	attendeeIDs []string,
	channel string,
	senderID string,
) []string {
	ids := make([]string, 0, len(attendeeIDs)+1)
	seen := make(map[string]struct{}, len(attendeeIDs)+1)
	addID := func(raw string) {
		id := strings.TrimSpace(raw)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, attendeeID := range attendeeIDs {
		addID(attendeeID)
	}

	if len(ids) == 0 &&
		strings.EqualFold(strings.TrimSpace(channel), "feishu") &&
		strings.TrimSpace(senderID) != "" &&
		!strings.EqualFold(strings.TrimSpace(senderID), "unknown") {
		addID(senderID)
	}

	return ids
}

func (t *FeishuCalendarTool) addFeishuEventAttendees(
	ctx context.Context,
	calendarID string,
	eventID string,
	inviteeUserIDs []string,
	needNotification bool,
) error {
	if len(inviteeUserIDs) == 0 {
		return nil
	}

	attendees := make([]*larkcalendar.CalendarEventAttendee, 0, len(inviteeUserIDs))
	for _, invitee := range inviteeUserIDs {
		attendees = append(attendees, larkcalendar.NewCalendarEventAttendeeBuilder().
			Type("user").
			UserId(invitee).
			Build())
	}

	req := larkcalendar.NewCreateCalendarEventAttendeeReqBuilder().
		CalendarId(calendarID).
		EventId(eventID).
		UserIdType(larkcalendar.UserIdTypeCreateCalendarEventAttendeeUserId).
		Body(larkcalendar.NewCreateCalendarEventAttendeeReqBodyBuilder().
			Attendees(attendees).
			NeedNotification(needNotification).
			Build()).
		Build()

	resp, err := t.client.Calendar.V4.CalendarEventAttendee.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("add attendees request failed: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("add attendees api error: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return nil
}
