package telegram

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	telegramTopicSeparator = ":topic:"
	telegramGeneralTopicID = 1
)

type telegramTarget struct {
	ChatID          int64
	MessageThreadID int
}

func parseTelegramTarget(raw string) (telegramTarget, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return telegramTarget{}, fmt.Errorf("empty telegram chat ID")
	}

	base, topic, hasTopic := strings.Cut(raw, telegramTopicSeparator)
	chatID, err := parseChatID(base)
	if err != nil {
		return telegramTarget{}, err
	}

	target := telegramTarget{ChatID: chatID}
	if !hasTopic {
		return target, nil
	}

	threadID, err := strconv.Atoi(strings.TrimSpace(topic))
	if err != nil {
		return telegramTarget{}, fmt.Errorf("invalid telegram topic ID %q: %w", topic, err)
	}
	if threadID <= 0 {
		return telegramTarget{}, fmt.Errorf("invalid telegram topic ID %d", threadID)
	}
	target.MessageThreadID = threadID
	return target, nil
}

func buildTelegramTopicChatID(chatID int64, threadID int) string {
	if threadID <= 0 {
		return strconv.FormatInt(chatID, 10)
	}
	return fmt.Sprintf("%d%s%d", chatID, telegramTopicSeparator, threadID)
}

func resolveTelegramForumThreadID(isForum bool, messageThreadID int) (int, bool) {
	if !isForum {
		return 0, false
	}
	if messageThreadID <= 0 {
		return telegramGeneralTopicID, true
	}
	return messageThreadID, true
}

func (t telegramTarget) messageThreadIDForSend() (int, bool) {
	if t.MessageThreadID <= 0 || t.MessageThreadID == telegramGeneralTopicID {
		return 0, false
	}
	return t.MessageThreadID, true
}

func (t telegramTarget) messageThreadIDForTyping() (int, bool) {
	if t.MessageThreadID <= 0 {
		return 0, false
	}
	return t.MessageThreadID, true
}
