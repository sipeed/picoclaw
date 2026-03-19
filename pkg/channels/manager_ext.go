package channels

import (
	"context"
	"hash/fnv"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
)

const (
	statusMsgTTL = 5 * time.Minute
	taskMsgTTL   = 30 * time.Minute

	// statusEditInterval is the minimum interval between EditMessage calls
	// for the same status/task bubble. EditMessage APIs are more rate-sensitive
	// than SendMessageDraft, so we throttle edits to avoid "(edited)" flicker
	// and API rate limit errors. Draft-based channels bypass this throttle.
	// Telegram enforces ~20 messages/min per group chat, so 3s is safe.
	statusEditInterval = 3 * time.Second
)

// statusMsgEntry tracks a status or task message ID for later editing.
type statusMsgEntry struct {
	messageID string
	draftID   int // non-zero when using draft-based streaming
	createdAt time.Time
}

// managerExt holds fork-specific fields for Manager.
// Embedded in Manager so existing field access continues to work.
type managerExt struct {
	statusMsgIDs    sync.Map // "channel:chatID" → statusMsgEntry (streaming preview)
	taskMsgIDs      sync.Map // "channel:chatID:taskID" → statusMsgEntry (background task status)
	statusEditTimes sync.Map // key → time.Time — last EditMessage time for throttling
}

// handleStatusSend processes IsStatus messages (streaming previews).
// It reuses an existing placeholder or tracked status message, or sends a new
// one via SendWithID so subsequent status updates edit the same bubble.
// For channels implementing DraftSender (e.g. Telegram private chats),
// sendMessageDraft is preferred as it avoids the "(edited)" indicator.
// If the channel doesn't support editing, the message is silently dropped.
func (m *Manager) handleStatusSend(ctx context.Context, name string, w *channelWorker, msg bus.OutboundMessage) {
	if err := w.limiter.Wait(ctx); err != nil {
		return
	}

	key := name + ":" + msg.ChatID

	// 0. Draft-based streaming (preferred for supported channels)
	if drafter, ok := w.ch.(DraftSender); ok {
		var did int
		if v, loaded := m.statusMsgIDs.Load(key); loaded {
			if entry, ok := v.(statusMsgEntry); ok && entry.draftID != 0 {
				did = entry.draftID
			}
		}
		if did == 0 {
			did = generateDraftID(key)
		}
		if err := drafter.SendDraft(ctx, msg.ChatID, did, msg.Content); err == nil {
			// Track draft only after successful send. If draft fails (e.g. group
			// main thread), keep existing messageID entry so fallback edits can
			// reuse the same status bubble instead of creating duplicates.
			m.statusMsgIDs.Store(key, statusMsgEntry{
				draftID:   did,
				createdAt: time.Now(),
			})
			return
		}
		// Draft failed — fall through to edit-based approach
	}

	// Edit-based path: throttle to statusEditInterval per key to avoid
	// API rate limit errors and "(edited)" flicker.
	if v, loaded := m.statusEditTimes.Load(key); loaded {
		if t, ok := v.(time.Time); ok && time.Since(t) < statusEditInterval {
			return // too recent, skip this update
		}
	}

	// 1. Try editing an existing placeholder
	if v, loaded := m.placeholders.Load(key); loaded {
		if entry, ok := v.(placeholderEntry); ok && entry.id != "" {
			if editor, ok := w.ch.(MessageEditor); ok {
				// Always record edit time to prevent retry storms on 429 errors.
				m.statusEditTimes.Store(key, time.Now())
				if err := editor.EditMessage(ctx, msg.ChatID, entry.id, msg.Content); err == nil {
					return
				}
			}
		}
	}

	// 2. Try editing a previously tracked status message
	if v, loaded := m.statusMsgIDs.Load(key); loaded {
		if entry, ok := v.(statusMsgEntry); ok && entry.messageID != "" {
			if editor, ok := w.ch.(MessageEditor); ok {
				m.statusEditTimes.Store(key, time.Now())
				if err := editor.EditMessage(ctx, msg.ChatID, entry.messageID, msg.Content); err == nil {
					return
				}
			}
		}
	}

	// 3. Send new message via SendWithID and track it
	if sender, ok := w.ch.(MessageSenderWithID); ok {
		if msgID, err := sender.SendWithID(ctx, msg.ChatID, msg.Content); err == nil && msgID != "" {
			m.statusMsgIDs.Store(key, statusMsgEntry{
				messageID: msgID,
				createdAt: time.Now(),
			})
			return
		}
	}

	// 4. Channel doesn't support SendWithID or editing — drop silently
}

func taskStatusKey(channel, chatID, taskID string) string {
	if taskID == "" {
		return ""
	}
	if channel == "" || chatID == "" {
		return taskID
	}
	return channel + ":" + chatID + ":" + taskID
}

// handleTaskStatusSend processes IsTaskStatus messages (background task status).
// It reuses a previously tracked task message, or sends a new one via SendWithID.
// For channels implementing DraftSender, sendMessageDraft is used to avoid "(edited)".
// If the channel doesn't support editing, falls back to regular Send.
func (m *Manager) handleTaskStatusSend(ctx context.Context, name string, w *channelWorker, msg bus.OutboundMessage) {
	if err := w.limiter.Wait(ctx); err != nil {
		return
	}

	taskKey := taskStatusKey(name, msg.ChatID, msg.TaskID)

	// Final message: reuse the existing bubble when possible to avoid
	// duplicate messages. If a permanent message (messageID) is tracked,
	// edit it in-place. If a draft (draftID) is tracked, update it with
	// the completion content (the draft persists in Telegram and serves
	// as the visible message; sending a separate permanent message would
	// create a duplicate).
	if msg.Final {
		v, loaded := m.taskMsgIDs.LoadAndDelete(taskKey)
		m.statusEditTimes.Delete(taskKey)

		if loaded {
			if entry, ok := v.(statusMsgEntry); ok {
				// Path A: a permanent message exists — edit it in-place.
				if entry.messageID != "" {
					if editor, ok := w.ch.(MessageEditor); ok {
						if err := editor.EditMessage(ctx, msg.ChatID, entry.messageID, msg.Content); err == nil {
							return
						}
					}
					// Edit failed — fall through to send a new message.
				}

				// Path B: a draft exists — update it with the final
				// content. Drafts persist visibly in Telegram, so do NOT
				// send a separate permanent message (that causes duplicates).
				if entry.draftID != 0 {
					if drafter, ok := w.ch.(DraftSender); ok {
						if err := drafter.SendDraft(ctx, msg.ChatID, entry.draftID, msg.Content); err == nil {
							return
						}
					}
					// Draft update failed — fall through to send permanent.
				}
			}
		}

		// No existing bubble to reuse — send a new permanent message.
		if sender, ok := w.ch.(MessageSenderWithID); ok {
			if msgID, err := sender.SendWithID(ctx, msg.ChatID, msg.Content); err == nil && msgID != "" {
				return
			}
		}
		_ = w.ch.Send(ctx, msg)
		return
	}

	// 0. Draft-based streaming (preferred for supported channels)
	if drafter, ok := w.ch.(DraftSender); ok && taskKey != "" {
		var did int
		if v, loaded := m.taskMsgIDs.Load(taskKey); loaded {
			if entry, ok := v.(statusMsgEntry); ok && entry.draftID != 0 {
				did = entry.draftID
			}
		}
		if did == 0 {
			did = generateDraftID(taskKey)
		}
		if err := drafter.SendDraft(ctx, msg.ChatID, did, msg.Content); err == nil {
			// Track draft only after successful send to avoid clobbering an
			// existing messageID entry when drafts are unsupported.
			m.taskMsgIDs.Store(taskKey, statusMsgEntry{
				draftID:   did,
				createdAt: time.Now(),
			})
			return
		}
		// Draft failed — fall through to edit-based approach
	}

	// Edit-based path: throttle to statusEditInterval per task key.
	if taskKey != "" {
		if v, loaded := m.statusEditTimes.Load(taskKey); loaded {
			if t, ok := v.(time.Time); ok && time.Since(t) < statusEditInterval {
				return
			}
		}
	}

	// 1. Try editing an existing task message
	if taskKey != "" {
		if v, loaded := m.taskMsgIDs.Load(taskKey); loaded {
			if entry, ok := v.(statusMsgEntry); ok && entry.messageID != "" {
				if editor, ok := w.ch.(MessageEditor); ok {
					m.statusEditTimes.Store(taskKey, time.Now())
					if err := editor.EditMessage(ctx, msg.ChatID, entry.messageID, msg.Content); err == nil {
						return
					}
				}
			}
		}
	}

	// 2. Send new message via SendWithID and track it
	if sender, ok := w.ch.(MessageSenderWithID); ok {
		if msgID, err := sender.SendWithID(ctx, msg.ChatID, msg.Content); err == nil && msgID != "" {
			if taskKey != "" {
				m.taskMsgIDs.Store(taskKey, statusMsgEntry{
					messageID: msgID,
					createdAt: time.Now(),
				})
			}
			return
		}
	}

	// 3. Fallback: regular Send (for channels without SendWithID)
	_ = w.ch.Send(ctx, msg)
}

// generateDraftID produces a stable non-zero int from a key string.
// The same key always maps to the same draft ID so successive calls
// animate the same Telegram draft bubble.
func generateDraftID(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	v := int(h.Sum32())
	if v == 0 {
		v = 1 // draftID must be non-zero
	}
	if v < 0 {
		v = -v
	}
	return v
}

// PromoteStatusToTask moves the tracked streaming status message for the given
// channel:chatID key into the task message map under channel:chatID:taskID. This allows the
// next IsTaskStatus publish to edit the streaming bubble instead of creating a
// new message. Returns true if a status message was found and promoted.
func (m *Manager) PromoteStatusToTask(statusKey, taskID string) bool {
	v, loaded := m.statusMsgIDs.LoadAndDelete(statusKey)
	if !loaded {
		return false
	}

	parts := strings.SplitN(statusKey, ":", 2)
	if len(parts) == 2 {
		m.taskMsgIDs.Store(taskStatusKey(parts[0], parts[1], taskID), v)
		return true
	}

	m.taskMsgIDs.Store(taskID, v)
	return true
}
