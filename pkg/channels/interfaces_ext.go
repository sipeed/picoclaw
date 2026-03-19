package channels

import "context"

// MessageSenderWithID — channels that can send a message and return its platform-specific ID.
// Used by Manager to track status/task messages for later editing.
type MessageSenderWithID interface {
	SendWithID(ctx context.Context, chatID string, content string) (messageID string, err error)
}

// DraftSender — channels that can send progressive draft messages.
// Used for streaming LLM output without the "edited" indicator.
// draftID must be non-zero and consistent across updates for the same draft.
type DraftSender interface {
	SendDraft(ctx context.Context, chatID string, draftID int, content string) error
}

// MessageDeleter — channels that can delete a previously sent message.
type MessageDeleter interface {
	DeleteMessage(ctx context.Context, chatID string, messageID string) error
}
