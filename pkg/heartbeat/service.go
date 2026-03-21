// Piconomous - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 Piconomous contributors

package heartbeat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/piconomous/pkg/bus"
	"github.com/sipeed/piconomous/pkg/constants"
	"github.com/sipeed/piconomous/pkg/fileutil"
	"github.com/sipeed/piconomous/pkg/logger"
	"github.com/sipeed/piconomous/pkg/state"
	"github.com/sipeed/piconomous/pkg/tools"
)

const (
	minIntervalMinutes     = 5
	defaultIntervalMinutes = 30
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
	autonomousEnabled bool // when true, heartbeat drives goal-pursuit loop
	mu                sync.RWMutex
	stopChan          chan struct{}
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

// SetAutonomous activates the fully autonomous goal-driven operating mode.
// When enabled, each heartbeat cycle reads GOALS.md and builds a richer prompt
// that instructs the agent to make concrete progress toward its active goals,
// create follow-up cron tasks for itself, and write new goals as it discovers work.
func (hs *HeartbeatService) SetAutonomous(enabled bool) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.autonomousEnabled = enabled
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

	// Get last channel info for context
	lastChannel := hs.state.GetLastChannel()
	channel, chatID := hs.parseLastChannel(lastChannel)

	// Debug log for channel resolution
	hs.logInfof("Resolved channel: %s, chatID: %s (from lastChannel: %s)", channel, chatID, lastChannel)

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

	// Send result to user
	if result.ForUser != "" {
		hs.sendResponse(result.ForUser)
	} else if result.ForLLM != "" {
		hs.sendResponse(result.ForLLM)
	}

	hs.logInfof("Heartbeat completed: %s", result.ForLLM)
}

// buildPrompt builds the heartbeat prompt from HEARTBEAT.md and optionally GOALS.md.
// When autonomous mode is enabled, a richer goal-pursuit prompt is included.
func (hs *HeartbeatService) buildPrompt() string {
	hs.mu.RLock()
	autonomous := hs.autonomousEnabled
	hs.mu.RUnlock()

	heartbeatPath := filepath.Join(hs.workspace, "HEARTBEAT.md")
	data, err := os.ReadFile(heartbeatPath)
	if err != nil {
		if os.IsNotExist(err) {
			hs.createDefaultHeartbeatTemplate()
		} else {
			hs.logErrorf("Error reading HEARTBEAT.md: %v", err)
		}
		// In autonomous mode, still produce a prompt even without HEARTBEAT.md
		if !autonomous {
			return ""
		}
		data = nil
	}

	now := time.Now().Format("2006-01-02 15:04:05")

	if autonomous {
		return hs.buildAutonomousPrompt(now, string(data))
	}

	content := string(data)
	if len(content) == 0 {
		return ""
	}
	return fmt.Sprintf(`# Heartbeat Check

Current time: %s

You are a proactive AI assistant. This is a scheduled heartbeat check.
Review the following tasks and execute any necessary actions using available skills.
If there is nothing that requires attention, respond ONLY with: HEARTBEAT_OK

%s
`, now, content)
}

// buildAutonomousPrompt constructs a goal-driven autonomous operation prompt.
// It includes active goals from GOALS.md so the agent can make measurable progress
// toward them, self-schedule next steps, and update goal status.
func (hs *HeartbeatService) buildAutonomousPrompt(now, heartbeatContent string) string {
	goalsPath := filepath.Join(hs.workspace, "GOALS.md")
	goalsData, err := os.ReadFile(goalsPath)
	if err != nil {
		if os.IsNotExist(err) {
			hs.createDefaultGoalsTemplate()
		} else {
			hs.logErrorf("Error reading GOALS.md: %v", err)
		}
	}
	goalsContent := strings.TrimSpace(string(goalsData))
	heartbeatSection := strings.TrimSpace(heartbeatContent)

	var extras string
	if heartbeatSection != "" {
		extras = fmt.Sprintf("\n## Periodic Tasks (HEARTBEAT.md)\n\n%s\n", heartbeatSection)
	}
	if goalsContent == "" {
		goalsContent = "*No active goals. Use the 'goal' tool with action='set' to define goals.*"
	}

	instructions := "## Autonomous Operating Instructions\n\n" +
		"You are a fully autonomous software agent. Follow the complete development lifecycle without waiting for human input:\n\n" +
		"### Every Cycle — Do ALL of the following:\n\n" +
		"1. **Review active goals**: Pick the highest-priority active goal. Determine the single most impactful action you can take toward it RIGHT NOW.\n\n" +
		"2. **Execute immediately**: Use available tools to make concrete, measurable progress:\n" +
		"   - Write/edit code files (write_file, edit_file, append_file)\n" +
		"   - Run tests and checks (exec)\n" +
		"   - Search documentation or web (web_fetch, web_search)\n" +
		"   - Manage project structure (list_dir, read_file)\n\n" +
		"3. **Project development lifecycle** — when working on software projects:\n" +
		"   - **Feature development**: Write clean, working code. Add new features incrementally.\n" +
		"   - **Testing**: After writing code, always run tests (exec: go test, pytest, npm test, etc.). Fix any failures before proceeding.\n" +
		"   - **Code quality**: After tests pass, check for obvious inefficiencies. Refactor only when it improves correctness or maintainability.\n" +
		"   - **Do not break working code**: Read existing code before changing it. Make changes incrementally and verify each step with tests.\n" +
		"   - **Commit progress**: Use exec to git commit when a meaningful increment is complete.\n\n" +
		"4. **Self-schedule next steps**: Use the 'cron' tool to schedule follow-up actions:\n" +
		"   - After writing code -> schedule a test run\n" +
		"   - After tests pass -> schedule code review/optimization\n" +
		"   - For recurring checks -> schedule at an appropriate interval\n\n" +
		"5. **Update goals**:\n" +
		"   - Use 'goal' with action='complete' when a goal is fully done (tested and working).\n" +
		"   - Use 'goal' with action='update' to add progress notes to a goal.\n" +
		"   - Use 'goal' with action='set' to capture newly discovered work.\n" +
		"   - Use 'goal' with action='drop' only if a goal is truly no longer relevant.\n\n" +
		"6. **Report only when needed**: Send a message to the user ONLY if:\n" +
		"   - A major milestone was completed (feature shipped, tests all passing)\n" +
		"   - You are blocked and need human input\n" +
		"   - Something went wrong that requires attention\n" +
		"   - Otherwise respond ONLY with: HEARTBEAT_OK\n\n" +
		"### Key Principles:\n" +
		"- **Never idle**: There is always something useful to do. If current goals are done, create new improvement goals.\n" +
		"- **Test before claiming done**: A goal is complete only when its implementation is tested and verified.\n" +
		"- **Incremental progress**: Small, safe changes with verification at each step.\n" +
		"- **Continuity**: Each cycle builds on the last. Read your own previous work before adding to it.\n\n" +
		"Think step-by-step. Be decisive. Make progress every cycle.\n"

	return fmt.Sprintf("# Autonomous Operation Cycle\n\nCurrent time: %s\n\n"+
		"You are operating in **fully autonomous mode**. No human is present — act independently and continuously.\n\n"+
		"## Your Active Goals (GOALS.md)\n\n%s\n%s%s",
		now, goalsContent, extras, instructions)
}

// createDefaultGoalsTemplate writes an initial GOALS.md so the agent has a starting point.
func (hs *HeartbeatService) createDefaultGoalsTemplate() {
	goalsPath := filepath.Join(hs.workspace, "GOALS.md")
	defaultContent := "# Goals\n\n" +
		"*Managed automatically by Piconomous in autonomous mode.*\n" +
		"*Use the `goal` tool (action: set/list/complete/update/drop) to manage goals.*\n\n" +
		"## How Autonomous Mode Works\n\n" +
		"The agent checks this file every heartbeat cycle and:\n" +
		"1. Picks the highest-priority active goal\n" +
		"2. Takes immediate action toward it (writes code, runs tests, commits progress)\n" +
		"3. Schedules follow-up steps with the cron tool\n" +
		"4. Marks goals complete only when tested and working\n" +
		"5. Creates new improvement goals when current ones are done\n\n" +
		"## Active Goals\n\n" +
		"*No active goals yet. Add goals with: goal(action='set', title='...', description='...', priority=1-5)*\n\n" +
		"## Completed Goals\n\n" +
		"## Dropped Goals\n"
	if err := fileutil.WriteFileAtomic(goalsPath, []byte(defaultContent), 0o644); err != nil {
		hs.logErrorf("Failed to create default GOALS.md: %v", err)
	} else {
		hs.logInfof("Created default GOALS.md template")
	}
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

// sendResponse sends the heartbeat response to the last channel
func (hs *HeartbeatService) sendResponse(response string) {
	hs.mu.RLock()
	msgBus := hs.bus
	hs.mu.RUnlock()

	if msgBus == nil {
		hs.logInfof("No message bus configured, heartbeat result not sent")
		return
	}

	// Get last channel from state
	lastChannel := hs.state.GetLastChannel()
	if lastChannel == "" {
		hs.logInfof("No last channel recorded, heartbeat result not sent")
		return
	}

	platform, userID := hs.parseLastChannel(lastChannel)

	// Skip internal channels that can't receive messages
	if platform == "" || userID == "" {
		return
	}

	pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pubCancel()
	msgBus.PublishOutbound(pubCtx, bus.OutboundMessage{
		Channel: platform,
		ChatID:  userID,
		Content: response,
	})

	hs.logInfof("Heartbeat result sent to %s", platform)
}

// parseLastChannel parses the last channel string into platform and userID.
// Returns empty strings for invalid or internal channels.
func (hs *HeartbeatService) parseLastChannel(lastChannel string) (platform, userID string) {
	if lastChannel == "" {
		return "", ""
	}

	// Parse channel format: "platform:user_id" (e.g., "telegram:123456")
	parts := strings.SplitN(lastChannel, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		hs.logErrorf("Invalid last channel format: %s", lastChannel)
		return "", ""
	}

	platform, userID = parts[0], parts[1]

	// Skip internal channels
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
