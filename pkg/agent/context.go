package agent

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type ContextRuntimeSettings struct {
	ContextWindowTokens      int
	PruningMode              string
	IncludeOldChitChat       bool
	SoftToolResultChars      int
	HardToolResultChars      int
	TriggerRatio             float64
	BootstrapSnapshotEnabled bool
	MemoryVectorEnabled      bool
	MemoryVectorDimensions   int
	MemoryVectorTopK         int
	MemoryVectorMinScore     float64
	MemoryVectorMaxChars     int
	MemoryVectorRecentDays   int
}

type ContextBuilder struct {
	workspace      string
	skillsLoader   *skills.SkillsLoader
	memory         *MemoryStore
	tools          *tools.ToolRegistry // Direct reference to tool registry
	settings       ContextRuntimeSettings
	bootstrapMu    sync.RWMutex
	bootstrapCache map[string]string

	// Cache for static system prompt to avoid rebuilding on every call.
	// Dynamic per-request data (time/session/summary) is appended in BuildMessages.
	systemPromptMutex  sync.RWMutex
	cachedSystemPrompt string
	cachedAt           time.Time
	existedAtCache     map[string]bool
}

func getGlobalConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".picoclaw")
}

func NewContextBuilder(workspace string) *ContextBuilder {
	// builtin skills: skills directory in current project
	// Use the skills/ directory under the current working directory
	wd, _ := os.Getwd()
	builtinSkillsDir := filepath.Join(wd, "skills")
	globalSkillsDir := filepath.Join(getGlobalConfigDir(), "skills")

	defaultSettings := ContextRuntimeSettings{
		PruningMode:              "tools_only",
		IncludeOldChitChat:       true,
		SoftToolResultChars:      2000,
		HardToolResultChars:      350,
		TriggerRatio:             0.8,
		BootstrapSnapshotEnabled: true,
		MemoryVectorEnabled:      true,
		MemoryVectorDimensions:   defaultMemoryVectorDimensions,
		MemoryVectorTopK:         defaultMemoryVectorTopK,
		MemoryVectorMinScore:     defaultMemoryVectorMinScore,
		MemoryVectorMaxChars:     defaultMemoryVectorMaxContextChars,
		MemoryVectorRecentDays:   defaultMemoryVectorRecentDailyDays,
	}

	memoryStore := NewMemoryStore(workspace)
	memoryStore.SetVectorSettings(MemoryVectorSettings{
		Enabled:         defaultSettings.MemoryVectorEnabled,
		Dimensions:      defaultSettings.MemoryVectorDimensions,
		TopK:            defaultSettings.MemoryVectorTopK,
		MinScore:        defaultSettings.MemoryVectorMinScore,
		MaxContextChars: defaultSettings.MemoryVectorMaxChars,
		RecentDailyDays: defaultSettings.MemoryVectorRecentDays,
	})

	return &ContextBuilder{
		workspace:      workspace,
		skillsLoader:   skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir),
		memory:         memoryStore,
		bootstrapCache: map[string]string{},
		settings:       defaultSettings,
	}
}

// SetToolsRegistry sets the tools registry for dynamic tool summary generation.
func (cb *ContextBuilder) SetToolsRegistry(registry *tools.ToolRegistry) {
	cb.tools = registry
	cb.InvalidateCache()
}

func (cb *ContextBuilder) SetRuntimeSettings(settings ContextRuntimeSettings) {
	if settings.PruningMode == "" {
		settings.PruningMode = "tools_only"
	}
	if settings.SoftToolResultChars <= 0 {
		settings.SoftToolResultChars = 2000
	}
	if settings.HardToolResultChars <= 0 {
		settings.HardToolResultChars = 350
	}
	if settings.TriggerRatio <= 0 || settings.TriggerRatio >= 1 {
		settings.TriggerRatio = 0.8
	}
	if settings.MemoryVectorDimensions <= 0 {
		settings.MemoryVectorDimensions = defaultMemoryVectorDimensions
	}
	if settings.MemoryVectorTopK <= 0 {
		settings.MemoryVectorTopK = defaultMemoryVectorTopK
	}
	if settings.MemoryVectorMinScore < 0 || settings.MemoryVectorMinScore >= 1 {
		settings.MemoryVectorMinScore = defaultMemoryVectorMinScore
	}
	if settings.MemoryVectorMaxChars <= 0 {
		settings.MemoryVectorMaxChars = defaultMemoryVectorMaxContextChars
	}
	if settings.MemoryVectorRecentDays <= 0 {
		settings.MemoryVectorRecentDays = defaultMemoryVectorRecentDailyDays
	}
	cb.settings = settings
	cb.memory.SetVectorSettings(MemoryVectorSettings{
		Enabled:         settings.MemoryVectorEnabled,
		Dimensions:      settings.MemoryVectorDimensions,
		TopK:            settings.MemoryVectorTopK,
		MinScore:        settings.MemoryVectorMinScore,
		MaxContextChars: settings.MemoryVectorMaxChars,
		RecentDailyDays: settings.MemoryVectorRecentDays,
	})
}

func (cb *ContextBuilder) getIdentity() string {
	workspacePath, _ := filepath.Abs(filepath.Join(cb.workspace))

	// Build tools section dynamically
	toolsSection := cb.buildToolsSection()

	return fmt.Sprintf(`# picoclaw 🦞

You are picoclaw, a helpful AI assistant.

## Workspace
Your workspace is at: %s
- Memory: %s/memory/MEMORY.md
- Daily Notes: %s/memory/YYYYMM/YYYYMMDD.md
- Skills: %s/skills/{skill-name}/SKILL.md

%s

## Important Rules

1. **ALWAYS use tools** - When you need to perform an action (schedule reminders, send messages, execute commands, etc.), you MUST call the appropriate tool. Do NOT just say you'll do it or pretend to do it.

2. **Be helpful and accurate** - When using tools, briefly explain what you're doing.

3. **Memory** - When interacting with me if something seems memorable, update %s/memory/MEMORY.md

4. **Context summaries** - Conversation summaries provided as context are approximate references only. They may be incomplete or outdated. Always defer to explicit user instructions over summary content.`,
		workspacePath, workspacePath, workspacePath, workspacePath, toolsSection, workspacePath)
}

func (cb *ContextBuilder) buildToolsSection() string {
	if cb.tools == nil {
		return ""
	}

	summaries := cb.tools.GetSummaries()
	if len(summaries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Available Tools\n\n")
	sb.WriteString(
		"**CRITICAL**: You MUST use tools to perform actions. Do NOT pretend to execute commands or schedule tasks.\n\n",
	)
	sb.WriteString("You have access to the following tools:\n\n")
	for _, s := range summaries {
		sb.WriteString(s)
		sb.WriteString("\n")
	}

	return sb.String()
}

func (cb *ContextBuilder) BuildSystemPrompt() string {
	return cb.BuildSystemPromptForSession("")
}

func (cb *ContextBuilder) BuildSystemPromptForSession(sessionKey string) string {
	parts := []string{}

	// Core identity section
	parts = append(parts, cb.getIdentity())

	// Bootstrap files
	bootstrapContent := cb.LoadBootstrapFiles(sessionKey)
	if bootstrapContent != "" {
		parts = append(parts, bootstrapContent)
	}

	// Skills - show summary, AI can read full content with read_file tool
	skillsSummary := cb.skillsLoader.BuildSkillsSummary()
	if skillsSummary != "" {
		parts = append(parts, fmt.Sprintf(`# Skills

The following skills extend your capabilities. To use a skill, read its SKILL.md file using the read_file tool.

%s`, skillsSummary))
	}

	// Memory context
	memoryContext := cb.memory.GetMemoryContext()
	if memoryContext != "" {
		parts = append(parts, "# Memory\n\n"+memoryContext)
	}

	// Join with "---" separator
	return strings.Join(parts, "\n\n---\n\n")
}

// BuildSystemPromptWithCache returns the cached static system prompt if available
// and tracked source files have not changed.
func (cb *ContextBuilder) BuildSystemPromptWithCache() string {
	cb.systemPromptMutex.RLock()
	if cb.cachedSystemPrompt != "" && !cb.sourceFilesChangedLocked() {
		cached := cb.cachedSystemPrompt
		cb.systemPromptMutex.RUnlock()
		return cached
	}
	cb.systemPromptMutex.RUnlock()

	cb.systemPromptMutex.Lock()
	defer cb.systemPromptMutex.Unlock()

	// Double-check after acquiring write lock.
	if cb.cachedSystemPrompt != "" && !cb.sourceFilesChangedLocked() {
		return cb.cachedSystemPrompt
	}

	baseline := cb.buildCacheBaseline()
	prompt := cb.BuildSystemPrompt()
	cb.cachedSystemPrompt = prompt
	cb.cachedAt = baseline.maxMtime
	cb.existedAtCache = baseline.existed

	return prompt
}

// InvalidateCache clears cached static prompt state.
func (cb *ContextBuilder) InvalidateCache() {
	cb.systemPromptMutex.Lock()
	defer cb.systemPromptMutex.Unlock()

	cb.cachedSystemPrompt = ""
	cb.cachedAt = time.Time{}
	cb.existedAtCache = nil
}

// sourcePaths returns tracked file paths for static prompt invalidation.
func (cb *ContextBuilder) sourcePaths() []string {
	return []string{
		filepath.Join(cb.workspace, "AGENTS.md"),
		filepath.Join(cb.workspace, "SOUL.md"),
		filepath.Join(cb.workspace, "USER.md"),
		filepath.Join(cb.workspace, "IDENTITY.md"),
		filepath.Join(cb.workspace, "memory", "MEMORY.md"),
	}
}

type cacheBaseline struct {
	existed  map[string]bool
	maxMtime time.Time
}

func (cb *ContextBuilder) buildCacheBaseline() cacheBaseline {
	skillsDir := filepath.Join(cb.workspace, "skills")
	allPaths := append(cb.sourcePaths(), skillsDir)

	existed := make(map[string]bool, len(allPaths))
	var maxMtime time.Time
	for _, p := range allPaths {
		info, err := os.Stat(p)
		existed[p] = err == nil
		if err == nil && info.ModTime().After(maxMtime) {
			maxMtime = info.ModTime()
		}
	}

	_ = filepath.WalkDir(skillsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr == nil && !d.IsDir() {
			if info, err := os.Stat(path); err == nil && info.ModTime().After(maxMtime) {
				maxMtime = info.ModTime()
			}
		}
		return nil
	})

	if maxMtime.IsZero() {
		// Keep non-zero baseline for empty workspace so future file creation
		// always has a later mtime.
		maxMtime = time.Unix(1, 0)
	}

	return cacheBaseline{existed: existed, maxMtime: maxMtime}
}

// sourceFilesChangedLocked checks whether tracked files changed since cache.
// Caller must hold at least a read lock on systemPromptMutex.
func (cb *ContextBuilder) sourceFilesChangedLocked() bool {
	if cb.cachedAt.IsZero() {
		return true
	}

	for _, p := range cb.sourcePaths() {
		if cb.fileChangedSince(p) {
			return true
		}
	}

	skillsDir := filepath.Join(cb.workspace, "skills")
	if cb.fileChangedSince(skillsDir) {
		return true
	}
	return skillFilesModifiedSince(skillsDir, cb.cachedAt)
}

func (cb *ContextBuilder) fileChangedSince(path string) bool {
	if cb.existedAtCache == nil {
		return true
	}

	existedBefore := cb.existedAtCache[path]
	info, err := os.Stat(path)
	existsNow := err == nil
	if existedBefore != existsNow {
		return true
	}
	if !existsNow {
		return false
	}
	return info.ModTime().After(cb.cachedAt)
}

var errWalkStop = errors.New("walk stop")

func skillFilesModifiedSince(skillsDir string, t time.Time) bool {
	changed := false
	err := filepath.WalkDir(skillsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr == nil && !d.IsDir() {
			if info, statErr := os.Stat(path); statErr == nil && info.ModTime().After(t) {
				changed = true
				return errWalkStop
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, errWalkStop) && !os.IsNotExist(err) {
		logger.DebugCF("agent", "skills walk error", map[string]any{"error": err.Error()})
	}
	return changed
}

func (cb *ContextBuilder) LoadBootstrapFiles(sessionKey string) string {
	if cb.settings.BootstrapSnapshotEnabled && strings.TrimSpace(sessionKey) != "" {
		cb.bootstrapMu.RLock()
		cached, ok := cb.bootstrapCache[sessionKey]
		cb.bootstrapMu.RUnlock()
		if ok {
			return cached
		}
	}

	bootstrapFiles := []string{
		"AGENTS.md",
		"SOUL.md",
		"USER.md",
		"IDENTITY.md",
	}

	var sb strings.Builder
	for _, filename := range bootstrapFiles {
		filePath := filepath.Join(cb.workspace, filename)
		if data, err := os.ReadFile(filePath); err == nil {
			fmt.Fprintf(&sb, "## %s\n\n%s\n\n", filename, data)
		}
	}

	content := sb.String()
	if cb.settings.BootstrapSnapshotEnabled && strings.TrimSpace(sessionKey) != "" {
		cb.bootstrapMu.Lock()
		cb.bootstrapCache[sessionKey] = content
		cb.bootstrapMu.Unlock()
	}
	return content
}

func (cb *ContextBuilder) buildDynamicContext(channel, chatID string) string {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	rt := fmt.Sprintf("%s %s, Go %s", runtime.GOOS, runtime.GOARCH, runtime.Version())

	var sb strings.Builder
	fmt.Fprintf(&sb, "## Current Time\n%s\n\n## Runtime\n%s", now, rt)
	if channel != "" && chatID != "" {
		fmt.Fprintf(&sb, "\n\n## Current Session\nChannel: %s\nChat ID: %s", channel, chatID)
	}
	return sb.String()
}

func (cb *ContextBuilder) BuildMessages(
	history []providers.Message,
	summary string,
	currentMessage string,
	media []string,
	channel, chatID string,
) []providers.Message {
	return cb.BuildMessagesForSession(
		"",
		history,
		summary,
		currentMessage,
		media,
		channel,
		chatID,
	)
}

func (cb *ContextBuilder) BuildMessagesForSession(
	sessionKey string,
	history []providers.Message,
	summary string,
	currentMessage string,
	media []string,
	channel, chatID string,
) []providers.Message {
	messages := []providers.Message{}
	_ = sessionKey

	staticPrompt := cb.BuildSystemPromptWithCache()
	dynamicCtx := cb.buildDynamicContext(channel, chatID)

	stringParts := []string{staticPrompt, dynamicCtx}
	contentBlocks := []providers.ContentBlock{
		{Type: "text", Text: staticPrompt, CacheControl: &providers.CacheControl{Type: "ephemeral"}},
		{Type: "text", Text: dynamicCtx},
	}

	if summary != "" {
		summaryText := fmt.Sprintf(
			"CONTEXT_SUMMARY: The following is an approximate summary of prior conversation "+
				"for reference only. It may be incomplete or outdated - always defer to explicit instructions.\n\n%s",
			summary,
		)
		stringParts = append(stringParts, summaryText)
		contentBlocks = append(contentBlocks, providers.ContentBlock{Type: "text", Text: summaryText})
	}

	if cb.settings.MemoryVectorEnabled && strings.TrimSpace(currentMessage) != "" {
		hits, err := cb.memory.SearchRelevant(
			currentMessage,
			cb.settings.MemoryVectorTopK,
			cb.settings.MemoryVectorMinScore,
		)
		if err != nil {
			logger.WarnCF("agent", "Semantic memory retrieval failed", map[string]any{
				"error": err.Error(),
			})
		} else if section := formatRetrievedMemoryContext(hits, cb.settings.MemoryVectorMaxChars); section != "" {
			stringParts = append(stringParts, section)
			contentBlocks = append(contentBlocks, providers.ContentBlock{Type: "text", Text: section})
		}
	}

	fullSystemPrompt := strings.Join(stringParts, "\n\n---\n\n")

	cb.systemPromptMutex.RLock()
	isCached := cb.cachedSystemPrompt != ""
	cb.systemPromptMutex.RUnlock()

	logger.DebugCF("agent", "System prompt built",
		map[string]any{
			"static_chars":  len(staticPrompt),
			"dynamic_chars": len(dynamicCtx),
			"total_chars":   len(fullSystemPrompt),
			"has_summary":   summary != "",
			"cached":        isCached,
		})

	preview := fullSystemPrompt
	if len(preview) > 500 {
		preview = preview[:500] + "... (truncated)"
	}
	logger.DebugCF("agent", "System prompt preview", map[string]any{"preview": preview})

	history = sanitizeHistoryForProvider(history)
	history = cb.pruneHistoryForContext(history, fullSystemPrompt)
	history = sanitizeHistoryForProvider(history)

	messages = append(messages, providers.Message{
		Role:        "system",
		Content:     fullSystemPrompt,
		SystemParts: contentBlocks,
	})

	messages = append(messages, history...)

	if strings.TrimSpace(currentMessage) != "" {
		messages = append(messages, providers.Message{
			Role:    "user",
			Content: currentMessage,
		})
	}

	return messages
}

func formatRetrievedMemoryContext(hits []MemoryVectorHit, maxChars int) string {
	if len(hits) == 0 {
		return ""
	}
	if maxChars <= 0 {
		maxChars = defaultMemoryVectorMaxContextChars
	}

	var sb strings.Builder
	sb.WriteString("# Retrieved Memory\n\n")
	sb.WriteString("Use these semantic hits as hints; prefer current workspace files when they conflict.\n\n")

	remaining := maxChars
	for _, hit := range hits {
		if remaining <= 0 {
			break
		}
		text := compactWhitespace(hit.Text)
		if text == "" {
			continue
		}
		line := fmt.Sprintf("- (score=%.2f, source=%s) %s\n", hit.Score, hit.Source, text)
		if len(line) > remaining {
			if remaining > 6 {
				line = line[:remaining-4] + "...\n"
			} else {
				break
			}
		}
		sb.WriteString(line)
		remaining -= len(line)
	}

	if remaining == maxChars {
		return ""
	}
	return strings.TrimSpace(sb.String())
}

func (cb *ContextBuilder) pruneHistoryForContext(
	history []providers.Message,
	systemPrompt string,
) []providers.Message {
	if len(history) == 0 || cb.settings.PruningMode == "off" || cb.settings.ContextWindowTokens <= 0 {
		return history
	}

	estimateTokens := func(msg providers.Message) int {
		chars := utf8.RuneCountInString(msg.Content)
		for _, tc := range msg.ToolCalls {
			chars += utf8.RuneCountInString(tc.Name)
			if tc.Function != nil {
				chars += utf8.RuneCountInString(tc.Function.Name)
				chars += utf8.RuneCountInString(tc.Function.Arguments)
			}
		}
		if chars == 0 {
			return 0
		}
		return chars * 2 / 5
	}

	totalTokens := utf8.RuneCountInString(systemPrompt) * 2 / 5
	for _, msg := range history {
		totalTokens += estimateTokens(msg)
	}

	ratio := float64(totalTokens) / float64(cb.settings.ContextWindowTokens)
	if ratio < cb.settings.TriggerRatio {
		return history
	}

	cutoff := len(history) - 8
	if cutoff <= 0 {
		return history
	}

	pruned := make([]providers.Message, 0, len(history))
	for i := 0; i < cutoff; i++ {
		msg := history[i]

		if cb.settings.PruningMode == "tools_only" && msg.Role == "tool" && cb.settings.SoftToolResultChars > 0 {
			raw := msg.Content
			if len(raw) > cb.settings.SoftToolResultChars {
				head := cb.settings.SoftToolResultChars * 7 / 10
				tail := cb.settings.SoftToolResultChars * 2 / 10
				if head+tail > len(raw) {
					head = len(raw)
					tail = 0
				}
				msg.Content = raw[:head] +
					"\n...\n[tool result condensed for context stability]\n...\n" +
					raw[len(raw)-tail:]
			}
		}

		pruned = append(pruned, msg)
	}
	pruned = append(pruned, history[cutoff:]...)

	if cb.settings.IncludeOldChitChat {
		pruned = compactOldChitChat(pruned, cutoff)
	}

	totalTokens = utf8.RuneCountInString(systemPrompt) * 2 / 5
	for _, msg := range pruned {
		totalTokens += estimateTokens(msg)
	}
	ratio = float64(totalTokens) / float64(cb.settings.ContextWindowTokens)
	if ratio < cb.settings.TriggerRatio || cb.settings.HardToolResultChars <= 0 {
		return pruned
	}

	scanLimit := minInt(cutoff, len(pruned))
	for i := 0; i < scanLimit; i++ {
		if ratio < cb.settings.TriggerRatio {
			break
		}
		msg := pruned[i]
		if msg.Role != "tool" || len(msg.Content) <= cb.settings.HardToolResultChars {
			continue
		}
		pruned[i].Content = "[tool result omitted for context stability; details preserved in session history]"
		totalTokens = utf8.RuneCountInString(systemPrompt) * 2 / 5
		for _, m := range pruned {
			totalTokens += estimateTokens(m)
		}
		ratio = float64(totalTokens) / float64(cb.settings.ContextWindowTokens)
	}

	return pruned
}

func compactOldChitChat(history []providers.Message, cutoff int) []providers.Message {
	if len(history) == 0 || cutoff <= 0 {
		return history
	}

	isLowSignal := func(msg providers.Message) bool {
		if msg.Role != "user" && msg.Role != "assistant" {
			return false
		}
		if len(msg.ToolCalls) > 0 || msg.ToolCallID != "" {
			return false
		}
		text := strings.ToLower(strings.TrimSpace(msg.Content))
		if text == "" || len(text) > 40 {
			return false
		}
		switch text {
		case "ok", "okay", "thanks", "thank you", "got it", "roger", "understood", "好的", "收到", "谢谢":
			return true
		}
		return false
	}

	result := make([]providers.Message, 0, len(history))
	i := 0
	for i < len(history) {
		if i >= cutoff || !isLowSignal(history[i]) {
			result = append(result, history[i])
			i++
			continue
		}

		j := i
		for j < cutoff && isLowSignal(history[j]) {
			j++
		}
		runLen := j - i
		if runLen >= 2 {
			result = append(result, providers.Message{
				Role:    "assistant",
				Content: fmt.Sprintf("[History note: %d brief acknowledgements condensed]", runLen),
			})
		} else {
			result = append(result, history[i])
		}
		i = j
	}

	return result
}

func sanitizeHistoryForProvider(history []providers.Message) []providers.Message {
	if len(history) == 0 {
		return history
	}

	sanitized := make([]providers.Message, 0, len(history))
	var pendingToolCalls map[string]struct{}
	var pendingToolCallOrder []string
	flushPendingToolCalls := func() {
		if len(pendingToolCalls) == 0 {
			pendingToolCalls = nil
			pendingToolCallOrder = nil
			return
		}
		for _, id := range pendingToolCallOrder {
			if _, ok := pendingToolCalls[id]; !ok {
				continue
			}
			sanitized = append(sanitized, providers.Message{
				Role:       "tool",
				ToolCallID: id,
				Content:    "[tool result missing in transcript; synthesized placeholder for provider compatibility]",
			})
		}
		pendingToolCalls = nil
		pendingToolCallOrder = nil
	}

	for _, msg := range history {
		switch msg.Role {
		case "system":
			// BuildMessages constructs a single authoritative system message.
			// Keep provider input compatible by dropping system messages from history.
			logger.DebugCF("agent", "Dropping system message from history", map[string]any{})
			continue

		case "tool":
			if pendingToolCalls == nil {
				logger.DebugCF("agent", "Dropping orphaned tool message", map[string]any{})
				continue
			}

			// When tool call IDs are available, require exact call_id matches.
			if len(pendingToolCalls) > 0 {
				if msg.ToolCallID == "" {
					logger.DebugCF("agent", "Dropping orphaned tool message with empty call id", map[string]any{})
					continue
				}
				if _, ok := pendingToolCalls[msg.ToolCallID]; !ok {
					logger.DebugCF(
						"agent",
						"Dropping duplicate/orphaned tool message with unknown call id",
						map[string]any{"tool_call_id": msg.ToolCallID},
					)
					continue
				}
				delete(pendingToolCalls, msg.ToolCallID)
			}
			sanitized = append(sanitized, msg)

		case "assistant":
			flushPendingToolCalls()

			if len(msg.ToolCalls) > 0 {
				if len(sanitized) == 0 {
					logger.DebugCF("agent", "Dropping assistant tool-call turn at history start", map[string]any{})
					continue
				}
				prev := sanitized[len(sanitized)-1]
				if prev.Role != "user" && prev.Role != "tool" {
					logger.DebugCF(
						"agent",
						"Dropping assistant tool-call turn with invalid predecessor",
						map[string]any{"prev_role": prev.Role},
					)
					continue
				}

				pendingToolCalls = make(map[string]struct{}, len(msg.ToolCalls))
				pendingToolCallOrder = make([]string, 0, len(msg.ToolCalls))
				for _, tc := range msg.ToolCalls {
					if tc.ID != "" {
						if _, exists := pendingToolCalls[tc.ID]; exists {
							continue
						}
						pendingToolCalls[tc.ID] = struct{}{}
						pendingToolCallOrder = append(pendingToolCallOrder, tc.ID)
					}
				}
			}
			sanitized = append(sanitized, msg)

		default:
			flushPendingToolCalls()
			sanitized = append(sanitized, msg)
		}
	}
	flushPendingToolCalls()

	return sanitized
}

func (cb *ContextBuilder) AddToolResult(
	messages []providers.Message,
	toolCallID, toolName, result string,
) []providers.Message {
	messages = append(messages, providers.Message{
		Role:       "tool",
		Content:    result,
		ToolCallID: toolCallID,
	})
	return messages
}

func (cb *ContextBuilder) AddAssistantMessage(
	messages []providers.Message,
	content string,
	toolCalls []map[string]any,
) []providers.Message {
	msg := providers.Message{
		Role:    "assistant",
		Content: content,
	}
	// Always add assistant message, whether or not it has tool calls
	messages = append(messages, msg)
	return messages
}

// GetSkillsInfo returns information about loaded skills.
func (cb *ContextBuilder) GetSkillsInfo() map[string]any {
	allSkills := cb.skillsLoader.ListSkills()
	skillNames := make([]string, 0, len(allSkills))
	for _, s := range allSkills {
		skillNames = append(skillNames, s.Name)
	}
	return map[string]any{
		"total":     len(allSkills),
		"available": len(allSkills),
		"names":     skillNames,
	}
}
