// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package heartbeat

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/tools"
)

const (
	minIntervalMinutes     = 5
	defaultIntervalMinutes = 30
	suppressionTTL         = 24 * time.Hour
)

// HeartbeatHandler is the function type for handling heartbeat.
// It returns a ToolResult that can indicate async operations.
// channel and chatID are derived from the last active user channel.
type HeartbeatHandler func(prompt, channel, chatID string) *tools.ToolResult

// HeartbeatService manages periodic heartbeat checks
type HeartbeatService struct {
	workspace         string
	bus               *bus.MessageBus
	state             *state.Manager
	handler           HeartbeatHandler
	interval          time.Duration
	enabled           bool
	mu                sync.RWMutex
	stopChan          chan struct{}
	lastNotifiedAt    time.Time // when a non-silent result was last sent to user
	heartbeatThreadID int
}

// NewHeartbeatService creates a new heartbeat service
func NewHeartbeatService(workspace string, intervalMinutes int, enabled bool) *HeartbeatService {
	// Apply minimum interval
	if intervalMinutes < minIntervalMinutes && intervalMinutes != 0 {
		intervalMinutes = minIntervalMinutes
	}

	if intervalMinutes == 0 {
		intervalMinutes = defaultIntervalMinutes
	}

	return &HeartbeatService{
		workspace: workspace,
		interval:  time.Duration(intervalMinutes) * time.Minute,
		enabled:   enabled,
		state:     state.NewManager(workspace),
	}
}

// SetBus sets the message bus for delivering heartbeat results.
func (hs *HeartbeatService) SetBus(msgBus *bus.MessageBus) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.bus = msgBus
}

// SetHandler sets the heartbeat handler.
func (hs *HeartbeatService) SetHandler(handler HeartbeatHandler) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.handler = handler
}

// SetHeartbeatThreadID configures Telegram thread routing for heartbeat messages.
func (hs *HeartbeatService) SetHeartbeatThreadID(threadID int) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.heartbeatThreadID = threadID
}

// ResetSuppression clears the notification suppression so the next
// non-silent heartbeat result will be delivered to the user again.
// Typically called when a user message arrives.
func (hs *HeartbeatService) ResetSuppression() {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.lastNotifiedAt = time.Time{}
}

// Start begins the heartbeat service
func (hs *HeartbeatService) Start() error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if hs.stopChan != nil {
		logger.InfoC("heartbeat", "Heartbeat service already running")
		return nil
	}

	if !hs.enabled {
		logger.InfoC("heartbeat", "Heartbeat service disabled")
		return nil
	}

	hs.stopChan = make(chan struct{})
	go hs.runLoop(hs.stopChan)

	logger.InfoCF("heartbeat", "Heartbeat service started", map[string]any{
		"interval_minutes": hs.interval.Minutes(),
	})

	return nil
}

// Stop gracefully stops the heartbeat service
func (hs *HeartbeatService) Stop() {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if hs.stopChan == nil {
		return
	}

	logger.InfoC("heartbeat", "Stopping heartbeat service")
	close(hs.stopChan)
	hs.stopChan = nil
}

// IsRunning returns whether the service is running
func (hs *HeartbeatService) IsRunning() bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.stopChan != nil
}

// runLoop runs the heartbeat ticker
func (hs *HeartbeatService) runLoop(stopChan chan struct{}) {
	ticker := time.NewTicker(hs.interval)
	defer ticker.Stop()

	// Run first heartbeat after initial delay
	time.AfterFunc(time.Second, func() {
		hs.executeHeartbeat()
	})

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			hs.executeHeartbeat()
		}
	}
}

// executeHeartbeat performs a single heartbeat check
func (hs *HeartbeatService) executeHeartbeat() {
	hs.mu.RLock()
	enabled := hs.enabled
	handler := hs.handler
	if !hs.enabled || hs.stopChan == nil {
		hs.mu.RUnlock()
		return
	}
	hs.mu.RUnlock()

	if !enabled {
		return
	}

	logger.DebugC("heartbeat", "Executing heartbeat")

	prompt := hs.buildPrompt()
	if prompt == "" {
		logger.InfoC("heartbeat", "No heartbeat prompt (HEARTBEAT.md empty or missing)")
		return
	}

	if handler == nil {
		hs.logErrorf("Heartbeat handler not configured")
		return
	}

	channel, chatID, reason := hs.resolveHeartbeatTarget()
	hs.logInfof("Resolved channel: %s, chatID: %s (%s)", channel, chatID, reason)

	result := handler(prompt, channel, chatID)

	if result == nil {
		hs.logInfof("Heartbeat handler returned nil result")
		return
	}

	// Handle different result types
	if result.IsError {
		hs.logErrorf("Heartbeat error: %s", result.ForLLM)
		return
	}

	if result.Async {
		hs.logInfof("Async task started: %s", result.ForLLM)
		logger.InfoCF("heartbeat", "Async heartbeat task started",
			map[string]any{
				"message": result.ForLLM,
			})
		return
	}

	// Check if silent
	if result.Silent {
		hs.logInfof("Heartbeat OK - silent")
		return
	}

	// Suppress duplicate notifications within the TTL window
	hs.mu.RLock()
	suppressed := !hs.lastNotifiedAt.IsZero() && time.Since(hs.lastNotifiedAt) < suppressionTTL
	hs.mu.RUnlock()
	if suppressed {
		hs.logInfof("Heartbeat suppressed (already notified user recently)")
		return
	}

	// Skip sendResponse — the task completion message in runAgentLoop already
	// includes the LLM response, so sending here would create a duplicate bubble.

	hs.mu.Lock()
	hs.lastNotifiedAt = time.Now()
	hs.mu.Unlock()

	hs.logInfof("Heartbeat completed: %s", result.ForLLM)
}

func (hs *HeartbeatService) resolveHeartbeatTarget() (channel, chatID, reason string) {
	if explicit := hs.state.GetHeartbeatTarget(); explicit != "" {
		if ch, cid := hs.parseTarget(explicit); ch != "" && cid != "" {
			return ch, cid, fmt.Sprintf("explicit heartbeat target: %s", explicit)
		}
		hs.logErrorf("Invalid explicit heartbeat target: %s", explicit)
	}

	if threadID := hs.telegramHeartbeatThreadID(); threadID > 0 {
		if ch, cid, src := hs.resolveTelegramThreadTarget(threadID); ch != "" && cid != "" {
			return ch, cid, src
		}
	}

	lastChannel := hs.state.GetLastChannel()
	channel, chatID = hs.parseLastChannel(lastChannel)
	return channel, chatID, fmt.Sprintf("fallback last channel: %s", lastChannel)
}

func (hs *HeartbeatService) resolveTelegramThreadTarget(threadID int) (channel, chatID, reason string) {
	candidates := []struct {
		value  string
		reason string
	}{
		{value: hs.state.GetLastHeartbeatTarget(), reason: "last heartbeat target"},
		{value: hs.state.GetLastChannel(), reason: "last channel"},
	}

	for _, candidate := range candidates {
		ch, cid := hs.parseTarget(candidate.value)
		if ch != "telegram" || cid == "" {
			continue
		}
		return ch, withTelegramThread(cid, threadID), fmt.Sprintf("telegram heartbeat_thread_id from %s", candidate.reason)
	}

	return "", "", ""
}

func (hs *HeartbeatService) telegramHeartbeatThreadID() int {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.heartbeatThreadID
}

func withTelegramThread(chatID string, threadID int) string {
	if threadID <= 0 || chatID == "" {
		return chatID
	}
	baseChatID := chatID
	if slash := strings.Index(baseChatID, "/"); slash >= 0 {
		baseChatID = baseChatID[:slash]
	}
	if baseChatID == "" {
		return chatID
	}
	return fmt.Sprintf("%s/%d", baseChatID, threadID)
}

// buildPrompt builds the heartbeat prompt from HEARTBEAT.md
func (hs *HeartbeatService) buildPrompt() string {
	heartbeatPath := filepath.Join(hs.workspace, "HEARTBEAT.md")

	data, err := os.ReadFile(heartbeatPath)
	if err != nil {
		if os.IsNotExist(err) {
			hs.createDefaultHeartbeatTemplate()
			return ""
		}
		hs.logErrorf("Error reading HEARTBEAT.md: %v", err)
		return ""
	}

	content := string(data)
	if len(content) == 0 {
		return ""
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	return fmt.Sprintf(`# Heartbeat Check

Current time: %s

You are a proactive AI assistant. This is a scheduled heartbeat check.
Review the following tasks and execute any necessary actions using available skills.
If there is nothing that requires attention, respond ONLY with: HEARTBEAT_OK

%s
`, now, content)
}

// createDefaultHeartbeatTemplate creates the default HEARTBEAT.md file
func (hs *HeartbeatService) createDefaultHeartbeatTemplate() {
	heartbeatPath := filepath.Join(hs.workspace, "HEARTBEAT.md")

	defaultContent := `# Heartbeat Check List

This file contains tasks for the heartbeat service to check periodically.

## Examples

- Check for unread messages
- Review upcoming calendar events
- Check device status (e.g., MaixCam)

## Instructions

- Execute ALL tasks listed below. Do NOT skip any task.
- For simple tasks (e.g., report current time), respond directly.
- For complex tasks that may take time, use the spawn tool to create a subagent.
- The spawn tool is async - subagent results will be sent to the user automatically.
- After spawning a subagent, CONTINUE to process remaining tasks.
- Only respond with HEARTBEAT_OK when ALL tasks are done AND nothing needs attention.

---

Add your heartbeat tasks below this line:
`

	if err := fileutil.WriteFileAtomic(heartbeatPath, []byte(defaultContent), 0o644); err != nil {
		hs.logErrorf("Failed to create default HEARTBEAT.md: %v", err)
	} else {
		hs.logInfof("Created default HEARTBEAT.md template")
	}
}

// parseLastChannel parses the last channel string into platform and userID.
// Returns empty strings for invalid or internal channels.
func (hs *HeartbeatService) parseLastChannel(lastChannel string) (platform, userID string) {
	return hs.parseTarget(lastChannel)
}

func (hs *HeartbeatService) parseTarget(target string) (platform, userID string) {
	if target == "" {
		return "", ""
	}

	parts := strings.SplitN(target, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		hs.logErrorf("Invalid heartbeat target format: %s", target)
		return "", ""
	}

	platform, userID = parts[0], parts[1]
	if constants.IsInternalChannel(platform) {
		hs.logInfof("Skipping internal channel: %s", platform)
		return "", ""
	}

	return platform, userID
}

// logInfof logs an informational message to the heartbeat log
func (hs *HeartbeatService) logInfof(format string, args ...any) {
	hs.logf("INFO", format, args...)
}

// logErrorf logs an error message to the heartbeat log
func (hs *HeartbeatService) logErrorf(format string, args ...any) {
	hs.logf("ERROR", format, args...)
}

// logf writes a message to the heartbeat log file
func (hs *HeartbeatService) logf(level, format string, args ...any) {
	logFile := filepath.Join(hs.workspace, "heartbeat.log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "[%s] [%s] %s\n", timestamp, level, fmt.Sprintf(format, args...))
}
