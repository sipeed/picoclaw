---
name: resend-calendar
description: "Send calendar invitations (.ics) via email. Trigger when a user wants to schedule a meeting, book a time slot, or send a calendar event invite to someone."
metadata: {"nanobot":{"emoji":"📅"}}
---

# Send Calendar Invites

Use the `send_calendar_invite` tool to email an iCalendar (.ics) invitation. The invite appears as a native calendar event in Gmail, Outlook, and Apple Calendar.

## When to use

- "Schedule a meeting with alice@example.com"
- "Send a calendar invite for our 3pm call"
- "Book a 1-hour slot with bob@company.com tomorrow at 10am"

## Parameters

| Parameter     | Required | Description                                          |
|---------------|----------|------------------------------------------------------|
| `to`          | yes      | Recipient email address                              |
| `summary`     | yes      | Event title shown in the calendar                    |
| `start`       | yes      | Start time in ISO 8601 (e.g. `2026-03-10T14:00:00`) |
| `end`         | yes      | End time in ISO 8601 (e.g. `2026-03-10T15:00:00`)   |
| `description` | no       | Meeting notes or agenda                              |

## Example

User: "Invite charlie@example.com to a sync tomorrow at 2pm for 30 minutes"

Call `send_calendar_invite` with:
- `to`: `charlie@example.com`
- `summary`: `Sync`
- `start`: `2026-03-02T14:00:00`
- `end`: `2026-03-02T14:30:00`
- `description`: `Quick sync call`
