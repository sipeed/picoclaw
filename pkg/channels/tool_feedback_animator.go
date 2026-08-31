package channels

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

const (
	toolFeedbackAnimationInterval    = 3 * time.Second
	toolFeedbackAnimationMaxDuration = 5 * time.Minute
	toolFeedbackEditTimeout          = 5 * time.Second
)

const initialToolFeedbackAnimationFrame = ""

var toolFeedbackAnimationFrames = []string{"..", "."}

// MaxToolFeedbackAnimationFrameLength returns the largest frame suffix length
// so callers can reserve room before sending messages to length-limited APIs.
func MaxToolFeedbackAnimationFrameLength() int {
	maxLen := len([]rune(initialToolFeedbackAnimationFrame))
	for _, frame := range toolFeedbackAnimationFrames {
		if frameLen := len([]rune(frame)); frameLen > maxLen {
			maxLen = frameLen
		}
	}
	return maxLen
}

type toolFeedbackAnimationState struct {
	messageID   string
	baseContent string
	stop        chan struct{}
	done        chan struct{}
}

type ToolFeedbackAnimator struct {
	mu                sync.Mutex
	editFn            func(ctx context.Context, chatID, messageID, content string) error
	entries           map[string]*toolFeedbackAnimationState
	animationInterval time.Duration
	maxDuration       time.Duration
	editTimeout       time.Duration
}

func NewToolFeedbackAnimator(
	editFn func(ctx context.Context, chatID, messageID, content string) error,
) *ToolFeedbackAnimator {
	return &ToolFeedbackAnimator{
		editFn:            editFn,
		entries:           make(map[string]*toolFeedbackAnimationState),
		animationInterval: toolFeedbackAnimationInterval,
		maxDuration:       toolFeedbackAnimationMaxDuration,
		editTimeout:       toolFeedbackEditTimeout,
	}
}

func (a *ToolFeedbackAnimator) Current(chatID string) (string, bool) {
	if a == nil || strings.TrimSpace(chatID) == "" {
		return "", false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	entry, ok := a.entries[chatID]
	if !ok || strings.TrimSpace(entry.messageID) == "" {
		return "", false
	}
	return entry.messageID, true
}

func (a *ToolFeedbackAnimator) Record(chatID, messageID, content string) {
	if a == nil {
		return
	}
	chatID = strings.TrimSpace(chatID)
	messageID = strings.TrimSpace(messageID)
	content = strings.TrimSpace(content)
	if chatID == "" || messageID == "" || content == "" {
		return
	}

	entry := &toolFeedbackAnimationState{
		messageID:   messageID,
		baseContent: content,
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
	}

	var previous *toolFeedbackAnimationState
	a.mu.Lock()
	if old, ok := a.entries[chatID]; ok {
		previous = old
	}
	a.entries[chatID] = entry
	a.mu.Unlock()

	stopToolFeedbackAnimation(previous)
	go a.run(chatID, entry)
}

func (a *ToolFeedbackAnimator) Clear(chatID string) {
	if a == nil || strings.TrimSpace(chatID) == "" {
		return
	}
	entry := a.detach(chatID)
	stopToolFeedbackAnimation(entry)
}

func (a *ToolFeedbackAnimator) Take(chatID string) (string, string, bool) {
	if a == nil || strings.TrimSpace(chatID) == "" {
		return "", "", false
	}
	entry := a.detach(chatID)
	if entry == nil || strings.TrimSpace(entry.messageID) == "" {
		return "", "", false
	}
	stopToolFeedbackAnimation(entry)
	return entry.messageID, entry.baseContent, true
}

// Update edits an existing tracked feedback message. If the edit fails, the
// previous feedback state is restored so callers can retry without orphaning
// the old progress message.
func (a *ToolFeedbackAnimator) Update(ctx context.Context, chatID, content string) (string, bool, error) {
	if a == nil || a.editFn == nil {
		return "", false, nil
	}
	msgID, baseContent, ok := a.Take(chatID)
	if !ok {
		return "", false, nil
	}

	animatedContent := InitialAnimatedToolFeedbackContent(content)
	if err := a.editFn(ctx, strings.TrimSpace(chatID), msgID, animatedContent); err != nil {
		a.Record(chatID, msgID, baseContent)
		return "", true, err
	}

	a.Record(chatID, msgID, content)
	return msgID, true, nil
}

func (a *ToolFeedbackAnimator) StopAll() {
	if a == nil {
		return
	}
	a.mu.Lock()
	entries := make([]*toolFeedbackAnimationState, 0, len(a.entries))
	for chatID, entry := range a.entries {
		entries = append(entries, entry)
		delete(a.entries, chatID)
	}
	a.mu.Unlock()

	for _, entry := range entries {
		stopToolFeedbackAnimation(entry)
	}
}

func (a *ToolFeedbackAnimator) detach(chatID string) *toolFeedbackAnimationState {
	if a == nil || strings.TrimSpace(chatID) == "" {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	entry := a.entries[chatID]
	delete(a.entries, chatID)
	return entry
}

func (a *ToolFeedbackAnimator) run(chatID string, entry *toolFeedbackAnimationState) {
	defer close(entry.done)

	ticker := time.NewTicker(a.animationInterval)
	defer ticker.Stop()
	lifetime := time.NewTimer(a.maxDuration)
	defer lifetime.Stop()

	frameIdx := 1

	for {
		select {
		case <-entry.stop:
			return
		case <-lifetime.C:
			logger.WarnCF("channels", "Tool feedback animation reached its maximum duration", map[string]any{
				"max_duration": a.maxDuration.String(),
			})
			return
		case <-ticker.C:
			if a.editFn == nil {
				continue
			}
			frame := toolFeedbackAnimationFrames[frameIdx%len(toolFeedbackAnimationFrames)]
			content := formatAnimatedToolFeedbackContent(entry.baseContent, frame)
			ctx, cancel := context.WithTimeout(context.Background(), a.editTimeout)
			err := a.editFn(ctx, chatID, entry.messageID, content)
			cancel()
			if err != nil {
				logger.WarnCF("channels", "Tool feedback animation stopped after edit failure", map[string]any{
					"error": err.Error(),
				})
				return
			}
			frameIdx++
		}
	}
}

func InitialAnimatedToolFeedbackContent(baseContent string) string {
	return formatAnimatedToolFeedbackContent(baseContent, initialToolFeedbackAnimationFrame)
}

func formatAnimatedToolFeedbackContent(baseContent, frame string) string {
	baseContent = strings.TrimSpace(baseContent)
	frame = strings.TrimSpace(frame)
	if baseContent == "" {
		return ""
	}
	if frame == "" {
		return baseContent
	}
	lineBreak := strings.IndexByte(baseContent, '\n')
	if lineBreak < 0 {
		return appendToolFeedbackFrame(baseContent, frame)
	}
	return appendToolFeedbackFrame(baseContent[:lineBreak], frame) + baseContent[lineBreak:]
}

func appendToolFeedbackFrame(firstLine, frame string) string {
	firstLine = strings.TrimSpace(firstLine)
	frame = strings.TrimSpace(frame)
	if firstLine == "" {
		return ""
	}
	if frame == "" {
		return firstLine
	}

	openTick := strings.IndexByte(firstLine, '`')
	if openTick >= 0 {
		if closeOffset := strings.IndexByte(firstLine[openTick+1:], '`'); closeOffset >= 0 {
			closeTick := openTick + 1 + closeOffset
			return firstLine[:closeTick] + frame + firstLine[closeTick:]
		}
	}

	return firstLine + frame
}

func stopToolFeedbackAnimation(entry *toolFeedbackAnimationState) {
	if entry == nil {
		return
	}
	select {
	case <-entry.stop:
	default:
		close(entry.stop)
	}
	<-entry.done
}
