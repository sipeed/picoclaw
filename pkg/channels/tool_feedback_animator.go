package channels

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultToolFeedbackAnimationInterval = 3 * time.Second

const initialToolFeedbackAnimationFrame = ""

var toolFeedbackAnimationFrames = []string{"..", "."}

var retryAfterPattern = regexp.MustCompile(`retry after:? (\d+)`)

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

// ToolFeedbackAnimatorConfig controls how often editable progress messages are
// updated. Zero values preserve the legacy behavior: animation edits every
// three seconds and no minimum interval between content edits.
type ToolFeedbackAnimatorConfig struct {
	AnimationInterval time.Duration
	MinEditInterval   time.Duration
}

type ToolFeedbackAnimator struct {
	mu                sync.Mutex
	editFn            func(ctx context.Context, chatID, messageID, content string) error
	entries           map[string]*toolFeedbackAnimationState
	animationInterval time.Duration
	minEditInterval   time.Duration
	lastEditAt        map[string]time.Time
	editPausedTil     map[string]time.Time
}

func NewToolFeedbackAnimator(
	editFn func(ctx context.Context, chatID, messageID, content string) error,
) *ToolFeedbackAnimator {
	return &ToolFeedbackAnimator{
		editFn:            editFn,
		entries:           make(map[string]*toolFeedbackAnimationState),
		animationInterval: defaultToolFeedbackAnimationInterval,
		lastEditAt:        make(map[string]time.Time),
		editPausedTil:     make(map[string]time.Time),
	}
}

func (a *ToolFeedbackAnimator) Configure(cfg ToolFeedbackAnimatorConfig) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if cfg.AnimationInterval > 0 {
		a.animationInterval = cfg.AnimationInterval
	} else {
		a.animationInterval = defaultToolFeedbackAnimationInterval
	}
	if cfg.MinEditInterval > 0 {
		a.minEditInterval = cfg.MinEditInterval
	} else {
		a.minEditInterval = 0
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

	if a.shouldSkipEdit(chatID) {
		a.Record(chatID, msgID, content)
		return msgID, true, nil
	}
	animatedContent := InitialAnimatedToolFeedbackContent(content)
	if err := a.editFn(ctx, strings.TrimSpace(chatID), msgID, animatedContent); err != nil {
		if isMessageNotModifiedError(err) {
			a.Record(chatID, msgID, content)
			return msgID, true, nil
		}
		if delay, ok := a.retryAfterDelay(err); ok {
			a.pauseEdits(chatID, delay)
		}
		a.Record(chatID, msgID, baseContent)
		return "", true, err
	}

	a.markEdit(chatID)
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

	ticker := time.NewTicker(a.getAnimationInterval())
	defer ticker.Stop()

	frameIdx := 1

	for {
		select {
		case <-entry.stop:
			return
		case <-ticker.C:
			if a.editFn == nil {
				continue
			}
			if a.shouldSkipEdit(chatID) {
				continue
			}
			frame := toolFeedbackAnimationFrames[frameIdx%len(toolFeedbackAnimationFrames)]
			content := formatAnimatedToolFeedbackContent(entry.baseContent, frame)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := a.editFn(ctx, chatID, entry.messageID, content); err != nil {
				if delay, ok := a.retryAfterDelay(err); ok {
					a.pauseEdits(chatID, delay)
				}
			} else {
				a.markEdit(chatID)
			}
			cancel()
			frameIdx++
		}
	}
}

func (a *ToolFeedbackAnimator) getAnimationInterval() time.Duration {
	if a == nil {
		return defaultToolFeedbackAnimationInterval
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.animationInterval > 0 {
		return a.animationInterval
	}
	return defaultToolFeedbackAnimationInterval
}

func (a *ToolFeedbackAnimator) shouldSkipEdit(chatID string) bool {
	if a == nil {
		return true
	}
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	if until := a.editPausedTil[chatID]; until.After(now) {
		return true
	}
	if a.minEditInterval <= 0 {
		return false
	}
	if last := a.lastEditAt[chatID]; !last.IsZero() && now.Sub(last) < a.minEditInterval {
		return true
	}
	return false
}

func (a *ToolFeedbackAnimator) markEdit(chatID string) {
	if a == nil {
		return
	}
	a.mu.Lock()
	a.lastEditAt[chatID] = time.Now()
	a.mu.Unlock()
}

func (a *ToolFeedbackAnimator) pauseEdits(chatID string, delay time.Duration) {
	if a == nil || delay <= 0 {
		return
	}
	a.mu.Lock()
	a.editPausedTil[chatID] = time.Now().Add(delay)
	a.mu.Unlock()
}

func (a *ToolFeedbackAnimator) retryAfterDelay(err error) (time.Duration, bool) {
	if err == nil || a == nil {
		return 0, false
	}
	a.mu.Lock()
	minInterval := a.minEditInterval
	a.mu.Unlock()
	if minInterval <= 0 {
		return 0, false
	}
	errText := strings.ToLower(err.Error())
	if !errors.Is(err, ErrRateLimit) &&
		!strings.Contains(errText, "too many requests") &&
		!strings.Contains(errText, "429") {
		return 0, false
	}
	match := retryAfterPattern.FindStringSubmatch(errText)
	if len(match) != 2 {
		return minInterval, true
	}
	seconds, parseErr := strconv.Atoi(match[1])
	if parseErr != nil || seconds <= 0 {
		return minInterval, true
	}
	return time.Duration(seconds) * time.Second, true
}

func isMessageNotModifiedError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "message is not modified")
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
