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

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/skills"
)

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ContextStrategy represents the strategy for building context messages.
type ContextStrategy int

const (
	// ContextStrategyFull builds complete context including identity, bootstrap, skills, tools, memory, and runtime info.
	ContextStrategyFull ContextStrategy = iota
	// ContextStrategyLite builds minimal context with only core identity, current time, and session info.
	ContextStrategyLite
	// ContextStrategyCustom builds custom context with explicitly specified includes.
	ContextStrategyCustom
)

// ContextBuildOptions holds options for custom context building.
type ContextBuildOptions struct {
	Strategy       ContextStrategy // Context building strategy
	IncludeTools   []string        // Tool names to include (for Custom strategy)
	ExcludeTools   []string        // Tool names to exclude (for Custom strategy)
	IncludeSkills  []string        // Skill names to include (for Custom strategy)
	ExcludeSkills  []string        // Skill names to exclude (for Custom strategy)
	IncludeMemory  bool            // Whether to include memory context (default: true)
	IncludeRuntime bool            // Whether to include runtime info (default: true)
}

type ContextBuilder struct {
	workspace    string
	skillsLoader *skills.SkillsLoader
	memory       *MemoryStore

	// SkillsFilter holds the list of skill names to include in system prompt.
	// When set, only these skills will be shown to the LLM.
	// This is used for dynamic skill filtering based on context.
	skillsFilterMutex sync.RWMutex
	skillsFilter      []string

	// SkillRecommender provides intelligent skill recommendations based on context.
	// Optional component for dynamic skill selection.
	skillRecommender *SkillRecommender

	// Cache for system prompt to avoid rebuilding on every call.
	// This fixes issue #607: repeated reprocessing of the entire context.
	// The cache auto-invalidates when workspace source files change (mtime check).
	systemPromptMutex  sync.RWMutex
	cachedSystemPrompt string
	cachedAt           time.Time // max observed mtime across tracked paths at cache build time

	// existedAtCache tracks which source file paths existed the last time the
	// cache was built. This lets sourceFilesChanged detect files that are newly
	// created (didn't exist at cache time, now exist) or deleted (existed at
	// cache time, now gone) — both of which should trigger a cache rebuild.
	existedAtCache map[string]bool
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

	return &ContextBuilder{
		workspace:    workspace,
		skillsLoader: skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir),
		memory:       NewMemoryStore(workspace),
	}
}

func (cb *ContextBuilder) getIdentity() string {
	workspacePath, _ := filepath.Abs(filepath.Join(cb.workspace))

	return fmt.Sprintf(`# picoclaw 🦞

You are picoclaw, a helpful AI assistant.

## Workspace
Your workspace is at: %s
- Memory: %s/memory/MEMORY.md
- Daily Notes: %s/memory/YYYYMM/YYYYMMDD.md
- Skills: %s/skills/{skill-name}/SKILL.md

## Important Rules

1. **ALWAYS use tools** - When you need to perform an action (schedule reminders, send messages, execute commands, etc.), you MUST call the appropriate tool. Do NOT just say you'll do it or pretend to do it.

2. **Be helpful and accurate** - When using tools, briefly explain what you're doing.

3. **Memory** - When interacting with me if something seems memorable, update %s/memory/MEMORY.md

4. **Context summaries** - Conversation summaries provided as context are approximate references only. They may be incomplete or outdated. Always defer to explicit user instructions over summary content.`,
		workspacePath, workspacePath, workspacePath, workspacePath, workspacePath)
}

func (cb *ContextBuilder) BuildSystemPrompt() string {
	parts := []string{}

	// Core identity section
	parts = append(parts, cb.getIdentity())

	// Bootstrap files
	bootstrapContent := cb.LoadBootstrapFiles()
	if bootstrapContent != "" {
		parts = append(parts, bootstrapContent)
	}

	// Skills - show summary filtered by skillsFilter if set
	cb.skillsFilterMutex.RLock()
	filter := cb.skillsFilter
	cb.skillsFilterMutex.RUnlock()

	skillsSummary := cb.skillsLoader.BuildSkillsSummaryFiltered(filter)
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

// BuildSystemPromptWithCache returns the cached system prompt if available
// and source files haven't changed, otherwise builds and caches it.
// Source file changes are detected via mtime checks (cheap stat calls).
func (cb *ContextBuilder) BuildSystemPromptWithCache() string {
	// Try read lock first — fast path when cache is valid
	cb.systemPromptMutex.RLock()
	if cb.cachedSystemPrompt != "" && !cb.sourceFilesChangedLocked() {
		result := cb.cachedSystemPrompt
		cb.systemPromptMutex.RUnlock()
		return result
	}
	cb.systemPromptMutex.RUnlock()

	// Acquire write lock for building
	cb.systemPromptMutex.Lock()
	defer cb.systemPromptMutex.Unlock()

	// Double-check: another goroutine may have rebuilt while we waited
	if cb.cachedSystemPrompt != "" && !cb.sourceFilesChangedLocked() {
		return cb.cachedSystemPrompt
	}

	// Snapshot the baseline (existence + max mtime) BEFORE building the prompt.
	// This way cachedAt reflects the pre-build state: if a file is modified
	// during BuildSystemPrompt, its new mtime will be > baseline.maxMtime,
	// so the next sourceFilesChangedLocked check will correctly trigger a
	// rebuild. The alternative (baseline after build) risks caching stale
	// content with a too-new baseline, making the staleness invisible.
	baseline := cb.buildCacheBaseline()
	prompt := cb.BuildSystemPrompt()
	cb.cachedSystemPrompt = prompt
	cb.cachedAt = baseline.maxMtime
	cb.existedAtCache = baseline.existed

	logger.DebugCF("agent", "System prompt cached",
		map[string]any{
			"length": len(prompt),
		})

	return prompt
}

// InvalidateCache clears the cached system prompt.
// Normally not needed because the cache auto-invalidates via mtime checks,
// but this is useful for tests or explicit reload commands.
func (cb *ContextBuilder) InvalidateCache() {
	cb.systemPromptMutex.Lock()
	defer cb.systemPromptMutex.Unlock()

	cb.cachedSystemPrompt = ""
	cb.cachedAt = time.Time{}
	cb.existedAtCache = nil

	logger.DebugCF("agent", "System prompt cache invalidated", nil)
}

// SetSkillsFilter sets the skills filter for this context builder.
// When set, only the specified skills will be included in the system prompt.
// This triggers cache invalidation to ensure the next request uses the updated skill set.
//
// Parameters:
//   - filters: List of skill names to include. Empty or nil clears the filter.
func (cb *ContextBuilder) SetSkillsFilter(filters []string) {
	cb.skillsFilterMutex.Lock()
	defer cb.skillsFilterMutex.Unlock()

	// Create a copy to prevent external modification
	if filters == nil {
		cb.skillsFilter = nil
	} else {
		cb.skillsFilter = make([]string, len(filters))
		copy(cb.skillsFilter, filters)
	}

	// Trigger cache invalidation
	cb.InvalidateCache()
}

// GetSkillsFilter returns the current skills filter.
// Returns nil if no filter is set (all skills available).
//
// The returned slice is a copy to prevent concurrent modification.
func (cb *ContextBuilder) GetSkillsFilter() []string {
	cb.skillsFilterMutex.RLock()
	defer cb.skillsFilterMutex.RUnlock()

	if cb.skillsFilter == nil {
		return nil
	}

	// Return a copy
	result := make([]string, len(cb.skillsFilter))
	copy(result, cb.skillsFilter)
	return result
}

// sourcePaths returns the workspace source file paths tracked for cache
// invalidation (bootstrap files + memory). The skills directory is handled
// separately in sourceFilesChangedLocked because it requires both directory-
// level and recursive file-level mtime checks.
func (cb *ContextBuilder) sourcePaths() []string {
	return []string{
		filepath.Join(cb.workspace, "AGENTS.md"),
		filepath.Join(cb.workspace, "SOUL.md"),
		filepath.Join(cb.workspace, "USER.md"),
		filepath.Join(cb.workspace, "IDENTITY.md"),
		filepath.Join(cb.workspace, "memory", "MEMORY.md"),
	}
}

// cacheBaseline holds the file existence snapshot and the latest observed
// mtime across all tracked paths. Used as the cache reference point.
type cacheBaseline struct {
	existed  map[string]bool
	maxMtime time.Time
}

// buildCacheBaseline records which tracked paths currently exist and computes
// the latest mtime across all tracked files + skills directory contents.
// Called under write lock when the cache is built.
func (cb *ContextBuilder) buildCacheBaseline() cacheBaseline {
	skillsDir := filepath.Join(cb.workspace, "skills")

	// All paths whose existence we track: source files + skills dir.
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

	// Walk skills files to capture their mtimes too.
	// Use os.Stat (not d.Info) to match the stat method used in
	// fileChangedSince / skillFilesModifiedSince for consistency.
	_ = filepath.WalkDir(skillsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr == nil && !d.IsDir() {
			if info, err := os.Stat(path); err == nil && info.ModTime().After(maxMtime) {
				maxMtime = info.ModTime()
			}
		}
		return nil
	})

	// If no tracked files exist yet (empty workspace), maxMtime is zero.
	// Use a very old non-zero time so that:
	// 1. cachedAt.IsZero() won't trigger perpetual rebuilds.
	// 2. Any real file created afterwards has mtime > cachedAt, so it
	//    will be detected by fileChangedSince (unlike time.Now() which
	//    could race with a file whose mtime <= Now).
	if maxMtime.IsZero() {
		maxMtime = time.Unix(1, 0)
	}

	return cacheBaseline{existed: existed, maxMtime: maxMtime}
}

// sourceFilesChangedLocked checks whether any workspace source file has been
// modified, created, or deleted since the cache was last built.
//
// IMPORTANT: The caller MUST hold at least a read lock on systemPromptMutex.
// Go's sync.RWMutex is not reentrant, so this function must NOT acquire the
// lock itself (it would deadlock when called from BuildSystemPromptWithCache
// which already holds RLock or Lock).
func (cb *ContextBuilder) sourceFilesChangedLocked() bool {
	if cb.cachedAt.IsZero() {
		return true
	}

	// Check tracked source files (bootstrap + memory).
	for _, p := range cb.sourcePaths() {
		if cb.fileChangedSince(p) {
			return true
		}
	}

	// --- Skills directory (handled separately from sourcePaths) ---
	//
	// 1. Creation/deletion: tracked via existedAtCache, same as bootstrap files.
	skillsDir := filepath.Join(cb.workspace, "skills")
	if cb.fileChangedSince(skillsDir) {
		return true
	}

	// 2. Structural changes (add/remove entries inside the dir) are reflected
	//    in the directory's own mtime, which fileChangedSince already checks.
	//
	// 3. Content-only edits to files inside skills/ do NOT update the parent
	//    directory mtime on most filesystems, so we recursively walk to check
	//    individual file mtimes at any nesting depth.
	if skillFilesModifiedSince(skillsDir, cb.cachedAt) {
		return true
	}

	return false
}

// fileChangedSince returns true if a tracked source file has been modified,
// newly created, or deleted since the cache was built.
//
// Four cases:
//   - existed at cache time, exists now -> check mtime
//   - existed at cache time, gone now   -> changed (deleted)
//   - absent at cache time,  exists now -> changed (created)
//   - absent at cache time,  gone now   -> no change
func (cb *ContextBuilder) fileChangedSince(path string) bool {
	// Defensive: if existedAtCache was never initialized, treat as changed
	// so the cache rebuilds rather than silently serving stale data.
	if cb.existedAtCache == nil {
		return true
	}

	existedBefore := cb.existedAtCache[path]
	info, err := os.Stat(path)
	existsNow := err == nil

	if existedBefore != existsNow {
		return true // file was created or deleted
	}
	if !existsNow {
		return false // didn't exist before, doesn't exist now
	}
	return info.ModTime().After(cb.cachedAt)
}

// errWalkStop is a sentinel error used to stop filepath.WalkDir early.
// Using a dedicated error (instead of fs.SkipAll) makes the early-exit
// intent explicit and avoids the nilerr linter warning that would fire
// if the callback returned nil when its err parameter is non-nil.
var errWalkStop = errors.New("walk stop")

// skillFilesModifiedSince recursively walks the skills directory and checks
// whether any file was modified after t. This catches content-only edits at
// any nesting depth (e.g. skills/name/docs/extra.md) that don't update
// parent directory mtimes.
func skillFilesModifiedSince(skillsDir string, t time.Time) bool {
	changed := false
	err := filepath.WalkDir(skillsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr == nil && !d.IsDir() {
			if info, statErr := os.Stat(path); statErr == nil && info.ModTime().After(t) {
				changed = true
				return errWalkStop // stop walking
			}
		}
		return nil
	})
	// errWalkStop is expected (early exit on first changed file).
	// os.IsNotExist means the skills dir doesn't exist yet — not an error.
	// Any other error is unexpected and worth logging.
	if err != nil && !errors.Is(err, errWalkStop) && !os.IsNotExist(err) {
		logger.DebugCF("agent", "skills walk error", map[string]any{"error": err.Error()})
	}
	return changed
}

func (cb *ContextBuilder) LoadBootstrapFiles() string {
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

	return sb.String()
}

// buildDynamicContext returns a short dynamic context string with per-request info.
// This changes every request (time, session) so it is NOT part of the cached prompt.
// LLM-side KV cache reuse is achieved by each provider adapter's native mechanism:
//   - Anthropic: per-block cache_control (ephemeral) on the static SystemParts block
//   - OpenAI / Codex: prompt_cache_key for prefix-based caching
//
// See: https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching
// See: https://platform.openai.com/docs/guides/prompt-caching
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

// buildLiteContext builds minimal context with only core identity, current time, and session info.
// This is useful for simple queries that don't require tools or skills.
func (cb *ContextBuilder) buildLiteContext(channel, chatID string) string {
	var sb strings.Builder

	// Core identity only
	sb.WriteString(cb.getIdentity())

	// Add dynamic context (time, runtime, session)
	dynamicCtx := cb.buildDynamicContext(channel, chatID)
	sb.WriteString("\n\n---\n\n")
	sb.WriteString(dynamicCtx)

	return sb.String()
}

// buildCustomContext builds custom context with explicitly specified includes/excludes.
// This provides fine-grained control over what context is included.
func (cb *ContextBuilder) buildCustomContext(opts ContextBuildOptions, channel, chatID string) string {
	parts := []string{}

	// Core identity (always included)
	parts = append(parts, cb.getIdentity())

	// Bootstrap files (optional, default: include)
	bootstrapContent := cb.LoadBootstrapFiles()
	if bootstrapContent != "" {
		parts = append(parts, bootstrapContent)
	}

	// Skills - filtered based on options
	if len(opts.IncludeSkills) > 0 || len(opts.ExcludeSkills) > 0 {
		skillsSummary := cb.skillsLoader.BuildSkillsSummaryFiltered(opts.IncludeSkills)
		if skillsSummary != "" {
			parts = append(parts, fmt.Sprintf(`# Skills

The following skills extend your capabilities. To use a skill, read its SKILL.md file using the read_file tool.

%s`, skillsSummary))
		}
	}

	// Memory context (optional, default: true)
	if opts.IncludeMemory {
		memoryContext := cb.memory.GetMemoryContext()
		if memoryContext != "" {
			parts = append(parts, "# Memory\n\n"+memoryContext)
		}
	}

	// Runtime info (optional, default: true)
	if opts.IncludeRuntime {
		dynamicCtx := cb.buildDynamicContext(channel, chatID)
		parts = append(parts, dynamicCtx)
	}

	// Join with "---" separator
	return strings.Join(parts, "\n\n---\n\n")
}

// BuildMessagesWithOptions builds context messages with specified strategy and options.
// This is the main entry point for building context, supporting multiple strategies:
//   - Full: Complete context (default, backward compatible)
//   - Lite: Minimal context for simple queries
//   - Custom: Explicitly specified includes/excludes
//
// Parameters:
//   - history: Conversation history
//   - summary: Optional conversation summary
//   - currentMessage: Current user message
//   - media: Optional media attachments
//   - channel: Channel type (e.g., "telegram", "wecom")
//   - chatID: Chat identifier
//   - opts: Build options (strategy, includes, excludes)
//
// Returns:
//   - Array of provider-compatible messages
func (cb *ContextBuilder) BuildMessagesWithOptions(
	history []providers.Message,
	summary string,
	currentMessage string,
	media []string,
	channel, chatID string,
	opts ContextBuildOptions,
) []providers.Message {
	messages := []providers.Message{}

	// Set default values for optional fields
	if !opts.IncludeMemory {
		opts.IncludeMemory = true
	}
	if !opts.IncludeRuntime {
		opts.IncludeRuntime = true
	}

	// Auto-detect recommended skills if recommender is enabled and no explicit filter
	if cb.skillRecommender != nil && len(opts.IncludeSkills) == 0 && len(cb.skillsFilter) == 0 {
		recommendations, err := cb.skillRecommender.RecommendSkillsForContext(channel, chatID, currentMessage, history)
		if err == nil && len(recommendations) > 0 {
			// Extract top N recommended skills (with score > threshold)
			recommendedSkillNames := make([]string, 0)
			for _, rec := range recommendations {
				if rec.Score >= 30.0 { // Only include skills with score >= 30%
					recommendedSkillNames = append(recommendedSkillNames, rec.Name)
				}
			}

			if len(recommendedSkillNames) > 0 {
				opts.IncludeSkills = recommendedSkillNames
				logger.DebugCF("agent", "Auto-recommended skills for context",
					map[string]any{
						"channel": channel,
						"skills":  recommendedSkillNames,
						"count":   len(recommendedSkillNames),
						"message": currentMessage[:min(50, len(currentMessage))] + "...",
					})
			}
		}
	}

	// Build static prompt based on strategy
	var staticPrompt string
	switch opts.Strategy {
	case ContextStrategyLite:
		staticPrompt = cb.buildLiteContext(channel, chatID)
	case ContextStrategyCustom:
		staticPrompt = cb.buildCustomContext(opts, channel, chatID)
	case ContextStrategyFull:
		fallthrough
	default:
		// Use cached system prompt for full strategy
		staticPrompt = cb.BuildSystemPromptWithCache()
	}

	// Build dynamic context (always included for non-lite strategies)
	var dynamicCtx string
	if opts.Strategy != ContextStrategyLite && opts.IncludeRuntime {
		dynamicCtx = cb.buildDynamicContext(channel, chatID)
	}

	// Compose system message parts
	stringParts := []string{staticPrompt}
	contentBlocks := []providers.ContentBlock{
		{Type: "text", Text: staticPrompt},
	}

	if dynamicCtx != "" {
		stringParts = append(stringParts, dynamicCtx)
		contentBlocks = append(contentBlocks, providers.ContentBlock{
			Type:         "text",
			Text:         dynamicCtx,
			CacheControl: &providers.CacheControl{Type: "ephemeral"},
		})
	}

	if summary != "" {
		summaryText := fmt.Sprintf(
			"CONTEXT_SUMMARY: The following is an approximate summary of prior conversation "+
				"for reference only. It may be incomplete or outdated — always defer to explicit instructions.\n\n%s",
			summary)
		stringParts = append(stringParts, summaryText)
		contentBlocks = append(contentBlocks, providers.ContentBlock{Type: "text", Text: summaryText})
	}

	fullSystemPrompt := strings.Join(stringParts, "\n\n---\n\n")

	// Log system prompt summary for debugging (debug mode only).
	cb.systemPromptMutex.RLock()
	isCached := cb.cachedSystemPrompt != ""
	cb.systemPromptMutex.RUnlock()

	logger.DebugCF("agent", "System prompt built with options",
		map[string]any{
			"strategy":      opts.Strategy,
			"static_chars":  len(staticPrompt),
			"dynamic_chars": len(dynamicCtx),
			"total_chars":   len(fullSystemPrompt),
			"has_summary":   summary != "",
			"cached":        isCached,
		})

	// Log preview of system prompt (avoid logging huge content)
	preview := fullSystemPrompt
	if len(preview) > 500 {
		preview = preview[:500] + "... (truncated)"
	}
	logger.DebugCF("agent", "System prompt preview",
		map[string]any{
			"preview": preview,
		})

	history = sanitizeHistoryForProvider(history)

	// Single system message containing all context — compatible with all providers.
	messages = append(messages, providers.Message{
		Role:        "system",
		Content:     fullSystemPrompt,
		SystemParts: contentBlocks,
	})

	// Add conversation history
	messages = append(messages, history...)

	// Add current user message
	if strings.TrimSpace(currentMessage) != "" {
		messages = append(messages, providers.Message{
			Role:    "user",
			Content: currentMessage,
		})
	}

	return messages
}

func (cb *ContextBuilder) BuildMessages(
	history []providers.Message,
	summary string,
	currentMessage string,
	media []string,
	channel, chatID string,
) []providers.Message {
	// Delegate to BuildMessagesWithOptions with default Full strategy
	opts := ContextBuildOptions{
		Strategy:       ContextStrategyFull,
		IncludeMemory:  true,
		IncludeRuntime: true,
	}
	return cb.BuildMessagesWithOptions(history, summary, currentMessage, media, channel, chatID, opts)
}

func sanitizeHistoryForProvider(history []providers.Message) []providers.Message {
	if len(history) == 0 {
		return history
	}

	sanitized := make([]providers.Message, 0, len(history))
	for _, msg := range history {
		switch msg.Role {
		case "system":
			// Drop system messages from history. BuildMessages always
			// constructs its own single system message (static + dynamic +
			// summary); extra system messages would break providers that
			// only accept one (Anthropic, Codex).
			logger.DebugCF("agent", "Dropping system message from history", map[string]any{})
			continue

		case "tool":
			if len(sanitized) == 0 {
				logger.DebugCF("agent", "Dropping orphaned leading tool message", map[string]any{})
				continue
			}
			// Walk backwards to find the nearest assistant message,
			// skipping over any preceding tool messages (multi-tool-call case).
			foundAssistant := false
			for i := len(sanitized) - 1; i >= 0; i-- {
				if sanitized[i].Role == "tool" {
					continue
				}
				if sanitized[i].Role == "assistant" && len(sanitized[i].ToolCalls) > 0 {
					foundAssistant = true
				}
				break
			}
			if !foundAssistant {
				logger.DebugCF("agent", "Dropping orphaned tool message", map[string]any{})
				continue
			}
			sanitized = append(sanitized, msg)

		case "assistant":
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
			}
			sanitized = append(sanitized, msg)

		default:
			sanitized = append(sanitized, msg)
		}
	}

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

// SetSkillRecommender sets the skill recommender for this context builder.
// When set, the recommender will be used to intelligently select skills based on context.
// This is an optional enhancement that can improve LLM performance by reducing context size.
func (cb *ContextBuilder) SetSkillRecommender(recommender *SkillRecommender) {
	cb.skillRecommender = recommender
}

// GetSkillsInfo returns information about loaded skills.
func (cb *ContextBuilder) GetSkillsInfo() map[string]any {
	allSkills := cb.skillsLoader.ListSkills()
	skillNames := make([]string, 0, len(allSkills))
	for _, s := range allSkills {
		skillNames = append(skillNames, s.Name)
	}

	result := map[string]any{
		"total":     len(allSkills),
		"available": len(allSkills),
		"names":     skillNames,
	}

	// Include recommender info if available
	if cb.skillRecommender != nil {
		result["recommender"] = "enabled"
	} else {
		result["recommender"] = "disabled"
	}

	return result
}
