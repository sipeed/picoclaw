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
	"github.com/sipeed/picoclaw/pkg/tools"
)

const orchestrationGuidance = `## Orchestration



You are the conductor, not the performer. **Your primary job is to delegate, not to implement.**



### spawn (non-blocking) — DEFAULT choice

Returns immediately. Use for any task that can run independently.

Call the spawn tool with JSON arguments like this:



Tool: spawn

Arguments: {"task": "Examine pkg/auth/ and report middleware pattern", "preset": "scout", "label": "auth-scout"}



Tool: spawn

Arguments: {"task": "Implement rate limiter in pkg/ratelimit/ with tests", "preset": "coder", "label": "rate-limiter"}



### subagent (blocking) — only when you need the answer NOW

Blocks until the subagent finishes. Use only when you cannot proceed without the result.

Does not take a preset — it runs with default tools.



Tool: subagent

Arguments: {"task": "Read pkg/config/config.go and list all SubagentsConfig fields", "label": "config-check"}



### When to use which

- spawn: parallel tasks, independent work, implementation, long analysis, >2 tool calls

- subagent: you need the result before your next decision

- inline: single quick tool call where delegation overhead is wasteful



### Presets (for spawn only)

| preset | role | can write | can exec |

|--------|------|-----------|----------|

| scout | explore, investigate | no | no |

| analyst | analyze, run tests | no | go test/vet, git |

| coder | implement + verify | yes (sandbox) | test/lint/fmt |

| worker | build + install | yes (sandbox) | build/package mgr |

| coordinator | orchestrate others | yes (sandbox) | general + spawn |



### Parallel spawning

Spawn multiple independent tasks at once — do NOT wait between them:



Tool: spawn

Arguments: {"task": "Analyze error handling patterns in pkg/providers/", "preset": "analyst", "label": "error-patterns"}



Tool: spawn

Arguments: {"task": "List all HTTP endpoints in pkg/miniapp/", "preset": "scout", "label": "endpoints"}



After spawning, record the assignment in ## Orchestration > Delegated in MEMORY.md.

When results come back, synthesize findings and decide the next fork.



### Subagent escalation

Deliberate subagents (coder/worker/coordinator) may ask you questions or submit plans for review.

When a subagent question appears, respond with the appropriate tool:

- answer_subagent: Answer a subagent's clarifying question

- review_subagent_plan: Approve or reject a subagent's execution plan (decision: "approved" or rejection feedback)



### Orchestration Memory

Maintain these sections in MEMORY.md under ## Orchestration:

- **Delegated**: Active subagent assignments (task ID, preset, description)

- **Findings**: Synthesized results from completed subagents

- **Decisions**: Key architectural/implementation decisions made during orchestration`

type ContextBuilder struct {
	workspace string

	workDir string // session-specific working directory (worktree or project subdir)

	skillsLoader *skills.SkillsLoader

	memory *MemoryStore

	tools *tools.ToolRegistry // Direct reference to tool registry

	peerNote string // set per-call from loop.go for peer session awareness

	orchestrationEnabled bool // set from AgentLoop when --orchestration flag is used

	// Cache for system prompt to avoid rebuilding on every call.

	// This fixes issue #607: repeated reprocessing of the entire context.

	// The cache auto-invalidates when workspace source files change (mtime check).

	systemPromptMutex sync.RWMutex

	cachedSystemPrompt string

	cachedAt time.Time // max observed mtime across tracked paths at cache build time

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
		workspace: workspace,

		skillsLoader: skills.NewSkillsLoader(workspace, globalSkillsDir, builtinSkillsDir),

		memory: NewMemoryStore(workspace),
	}
}

// SetToolsRegistry sets the tools registry for dynamic tool summary generation.

func (cb *ContextBuilder) SetToolsRegistry(registry *tools.ToolRegistry) {
	cb.tools = registry
}

// SetWorkDir sets the session-specific working directory (e.g., worktree path

// or project subdirectory). Bootstrap files found here take priority over workspace.

func (cb *ContextBuilder) SetWorkDir(dir string) {
	cb.workDir = dir
}

// SetPeerNote sets the peer session awareness note for the current call.

func (cb *ContextBuilder) SetPeerNote(note string) {
	cb.peerNote = note
}

// SetOrchestrationEnabled sets whether orchestration is enabled.

func (cb *ContextBuilder) SetOrchestrationEnabled(enabled bool) {
	cb.orchestrationEnabled = enabled
}

func (cb *ContextBuilder) getIdentity() string {
	workspacePath, _ := filepath.Abs(filepath.Join(cb.workspace))

	// Build tools section dynamically

	toolsSection := cb.buildToolsSection()

	// Build prompt with optional orchestration banner

	var prompt string

	if cb.orchestrationEnabled {
		prompt = ` /_/_/_/_/_/_/_/_/_/_/_/_/_/_/

 O R C H E S T R A  M O D E

/_/_/_/_/_/_/_/_/_/_/_/_/_/_/



`
	}

	// Conditional identity and plan executing rule for orchestration mode

	identity := "a helpful AI assistant"

	executingRule := `Work through the current Phase's steps.

     Mark each "- [x]" via edit_file. The system will auto-advance phases.`

	if cb.orchestrationEnabled {
		identity = "a conductor AI agent that orchestrates subagents"

		executingRule = `Delegate the current Phase's steps to subagents using spawn.

     For each step: spawn a subagent with the appropriate preset (scout for investigation,

     coder for implementation, analyst for review). Spawn multiple independent steps in parallel.

     When a subagent completes, mark "- [x]" via edit_file and record findings in

     ## Orchestration > Findings in MEMORY.md.

     Only do a step inline if it's a single quick tool call (e.g., reading one file).`
	}

	return fmt.Sprintf(prompt+`# picoclaw 🦞



You are picoclaw, %s.



## Workspace

Your workspace is at: %s

- Memory: %s/memory/MEMORY.md

- Daily Notes: %s/memory/YYYYMM/YYYYMMDD.md

- Skills: %s/skills/{skill-name}/SKILL.md



%s



## Important Rules



1. **ALWAYS use tools** - When you need to perform an action (schedule reminders, send messages, execute commands, etc.), you MUST call the appropriate tool. Do NOT just say you'll do it or pretend to do it.



2. **Be helpful and accurate** - When using tools, briefly explain what you're doing.



3. **Memory & Plans**

   - Use memory/MEMORY.md for structured plans.

   - NEVER remove or overwrite the header block (# Active Plan, > Task:, > Status:, > Phase:). The system parses these lines to track plan state.

   - If Status is "interviewing": Ask clarifying questions.

     After each answer, use edit_file to save findings to ## Context in memory/MEMORY.md.

     When you have enough information, add ## Phase sections with "- [ ]" checkbox steps, and ## Commands section below the header. Then change > Status: to "review".

   - If Status is "review": The plan is awaiting user approval. Do NOT change Status yourself.

   - If Status is "executing": %s

   - Plan format (header is written by the system — do NOT delete it):

     # Active Plan

     > Task: <description>

     > Status: interviewing | review | executing

     > Phase: <current phase number>

     ## Phase 1: <title>

     - [ ] Step 1

     - [ ] Step 2

     ## Phase 2: <title>

     - [ ] Step 1

     ## Commands

     build: <build command>

     test: <test command>

     lint: <lint command>

     ## Context

     <requirements, decisions, environment>

   - Keep each phase to 3-5 steps. Do NOT create plans without /plan.

   - Always ask about build/test/lint commands during interview.



4. **Response Formatting**

   - NEVER use ASCII box-drawing characters (┌─┐│└─┘╔═╗║╚═╝ etc.) or ASCII art diagrams.

   - Use markdown headings, bold, lists, and indentation for structure.

   - Keep lines short — most users read on mobile.

   - For architecture/flow, use arrow text: CLI → Pipeline → Adapters



5. **Context summaries** - Conversation summaries provided as context are approximate references only. They may be incomplete or outdated. Always defer to explicit user instructions over summary content.`,

		identity, workspacePath, workspacePath, workspacePath, workspacePath, toolsSection, executingRule)
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
	parts := []string{}

	// Core identity section

	parts = append(parts, cb.getIdentity())

	// Orchestration guidance — injected only when spawn tool is registered

	if cb.tools != nil {
		if _, hasSpawn := cb.tools.Get("spawn"); hasSpawn {
			parts = append(parts, orchestrationGuidance)
		}
	}

	// Bootstrap files

	bootstrapContent := cb.LoadBootstrapFiles()

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

	// Runtime status from tools (e.g., background processes)

	if cb.tools != nil {
		if status := cb.tools.GetRuntimeStatus(); status != "" {
			parts = append(parts, status)
		}
	}

	// Peer session coordination

	if cb.peerNote != "" {
		parts = append(parts, "## Active Sessions\n\n"+cb.peerNote)
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

// sourcePaths returns the workspace source file paths tracked for cache

// invalidation (bootstrap files + memory). The skills directory is handled

// separately in sourceFilesChangedLocked because it requires both directory-

// level and recursive file-level mtime checks.

func (cb *ContextBuilder) sourcePaths() []string {
	// Include bootstrap files from all search directories (workDir, planWorkDir, workspace).

	seen := map[string]bool{}

	var paths []string

	for _, spec := range bootstrapSpecs {
		var dirs []string

		if spec.Scope == "global" {
			dirs = []string{cb.workspace}
		} else {
			dirs = cb.bootstrapProjectDirs()
		}

		for _, dir := range dirs {
			p := filepath.Join(dir, spec.Name)

			if !seen[p] {
				seen[p] = true

				paths = append(paths, p)
			}
		}
	}

	// Always track memory file.

	memPath := filepath.Join(cb.workspace, "memory", "MEMORY.md")

	if !seen[memPath] {
		paths = append(paths, memPath)
	}

	return paths
}

// cacheBaseline holds the file existence snapshot and the latest observed

// mtime across all tracked paths. Used as the cache reference point.

type cacheBaseline struct {
	existed map[string]bool

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

// BootstrapFileInfo describes a resolved bootstrap file.

type BootstrapFileInfo struct {
	Name string `json:"name"`

	Path string `json:"path"` // empty = not found

	Scope string `json:"scope"` // "project" or "global"
}

// bootstrapFileSpec defines the search scope for each bootstrap file.

type bootstrapFileSpec struct {
	Name string

	Scope string // "project" = workDir→planWorkDir→workspace, "global" = workspace only
}

var bootstrapSpecs = []bootstrapFileSpec{
	{Name: "AGENTS.md", Scope: "project"},

	{Name: "IDENTITY.md", Scope: "project"},

	{Name: "SOUL.md", Scope: "global"},

	{Name: "USER.md", Scope: "global"},
}

// bootstrapProjectDirs returns de-duplicated search directories for project-scoped files.

func (cb *ContextBuilder) bootstrapProjectDirs() []string {
	seen := map[string]bool{}

	var dirs []string

	for _, d := range []string{cb.workDir, cb.memory.GetPlanWorkDir(), cb.workspace} {
		if d != "" && !seen[d] {
			seen[d] = true

			dirs = append(dirs, d)
		}
	}

	return dirs
}

func (cb *ContextBuilder) LoadBootstrapFiles() string {
	projectDirs := cb.bootstrapProjectDirs()

	var sb strings.Builder

	for _, spec := range bootstrapSpecs {
		var dirs []string

		if spec.Scope == "global" {
			dirs = []string{cb.workspace}
		} else {
			dirs = projectDirs
		}

		for _, dir := range dirs {
			filePath := filepath.Join(dir, spec.Name)

			if data, err := os.ReadFile(filePath); err == nil {
				fmt.Fprintf(&sb, "## %s\n\n%s\n\n", spec.Name, data)

				break
			}
		}
	}

	return sb.String()
}

// ResolveBootstrapPaths returns path resolution info for each bootstrap file

// using the same search logic as LoadBootstrapFiles.

func (cb *ContextBuilder) ResolveBootstrapPaths() []BootstrapFileInfo {
	projectDirs := cb.bootstrapProjectDirs()

	result := make([]BootstrapFileInfo, 0, len(bootstrapSpecs))

	for _, spec := range bootstrapSpecs {
		info := BootstrapFileInfo{Name: spec.Name, Scope: spec.Scope}

		var dirs []string

		if spec.Scope == "global" {
			dirs = []string{cb.workspace}
		} else {
			dirs = projectDirs
		}

		for _, dir := range dirs {
			filePath := filepath.Join(dir, spec.Name)

			if _, err := os.Stat(filePath); err == nil {
				info.Path = filePath

				break
			}
		}

		result = append(result, info)
	}

	return result
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

func (cb *ContextBuilder) BuildMessages(
	history []providers.Message,

	summary string,

	currentMessage string,

	media []string,

	channel, chatID string,
) []providers.Message {
	messages := []providers.Message{}

	// The static part (identity, bootstrap, skills, memory) is cached locally to

	// avoid repeated file I/O and string building on every call (fixes issue #607).

	// Dynamic parts (time, session, summary) are appended per request.

	// Everything is sent as a single system message for provider compatibility:

	// - Anthropic adapter extracts messages[0] (Role=="system") and maps its content

	//   to the top-level "system" parameter in the Messages API request. A single

	//   contiguous system block makes this extraction straightforward.

	// - Codex maps only the first system message to its instructions field.

	// - OpenAI-compat passes messages through as-is.

	staticPrompt := cb.BuildSystemPromptWithCache()

	// Build short dynamic context (time, runtime, session) — changes per request

	dynamicCtx := cb.buildDynamicContext(channel, chatID)

	// Compose a single system message: static (cached) + dynamic + optional summary.

	// Keeping all system content in one message ensures every provider adapter can

	// extract it correctly (Anthropic adapter -> top-level system param,

	// Codex -> instructions field).

	//

	// SystemParts carries the same content as structured blocks so that

	// cache-aware adapters (Anthropic) can set per-block cache_control.

	// The static block is marked "ephemeral" — its prefix hash is stable

	// across requests, enabling LLM-side KV cache reuse.

	stringParts := []string{staticPrompt, dynamicCtx}

	contentBlocks := []providers.ContentBlock{
		{Type: "text", Text: staticPrompt, CacheControl: &providers.CacheControl{Type: "ephemeral"}},

		{Type: "text", Text: dynamicCtx},
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

	// Read cachedSystemPrompt under lock to avoid a data race with

	// concurrent InvalidateCache / BuildSystemPromptWithCache writes.

	cb.systemPromptMutex.RLock()

	isCached := cb.cachedSystemPrompt != ""

	cb.systemPromptMutex.RUnlock()

	logger.DebugCF("agent", "System prompt built",

		map[string]any{
			"static_chars": len(staticPrompt),

			"dynamic_chars": len(dynamicCtx),

			"total_chars": len(fullSystemPrompt),

			"has_summary": summary != "",

			"cached": isCached,
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

	// SystemParts enables cache-aware adapters to set per-block cache_control;

	// Content is the concatenated fallback for adapters that don't read SystemParts.

	messages = append(messages, providers.Message{
		Role: "system",

		Content: fullSystemPrompt,

		SystemParts: contentBlocks,
	})

	// Add conversation history

	messages = append(messages, history...)

	// Add current user message

	if strings.TrimSpace(currentMessage) != "" {
		messages = append(messages, providers.Message{
			Role: "user",

			Content: currentMessage,
		})
	}

	return messages
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
		Role: "tool",

		Content: result,

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
		Role: "assistant",

		Content: content,
	}

	// Always add assistant message, whether or not it has tool calls

	messages = append(messages, msg)

	return messages
}

// LoadSkill loads a skill by name, returning its content (with frontmatter stripped) and whether it was found.

func (cb *ContextBuilder) LoadSkill(name string) (string, bool) {
	return cb.skillsLoader.LoadSkill(name)
}

// ListSkills returns all available skills from all tiers.

func (cb *ContextBuilder) ListSkills() []skills.SkillInfo {
	return cb.skillsLoader.ListSkills()
}

// Memory returns the underlying MemoryStore for direct plan queries.

func (cb *ContextBuilder) Memory() *MemoryStore {
	return cb.memory
}

// ---------- Plan passthrough methods ----------

// ReadMemory reads the long-term memory (MEMORY.md).

func (cb *ContextBuilder) ReadMemory() string {
	return cb.memory.ReadLongTerm()
}

// WriteMemory writes content to the long-term memory file.

func (cb *ContextBuilder) WriteMemory(content string) error {
	return cb.memory.WriteLongTerm(content)
}

// ClearMemory removes the long-term memory file.

func (cb *ContextBuilder) ClearMemory() error {
	return cb.memory.ClearLongTerm()
}

// HasActivePlan returns true if MEMORY.md contains an active plan.

func (cb *ContextBuilder) HasActivePlan() bool {
	return cb.memory.HasActivePlan()
}

// GetPlanStatus returns the plan status: "interviewing", "executing", or "".

func (cb *ContextBuilder) GetPlanStatus() string {
	return cb.memory.GetPlanStatus()
}

// IsPlanComplete returns true if all steps in all phases are [x].

func (cb *ContextBuilder) IsPlanComplete() bool {
	return cb.memory.IsPlanComplete()
}

// IsCurrentPhaseComplete returns true if all steps in the current phase are [x].

func (cb *ContextBuilder) IsCurrentPhaseComplete() bool {
	return cb.memory.IsCurrentPhaseComplete()
}

// AdvancePhase increments the current phase number by 1.

func (cb *ContextBuilder) AdvancePhase() error {
	return cb.memory.AdvancePhase()
}

// SetCurrentPhase sets the current phase number to n.

func (cb *ContextBuilder) SetCurrentPhase(n int) error {
	return cb.memory.SetPhase(n)
}

// GetCurrentPhase returns the current phase number.

func (cb *ContextBuilder) GetCurrentPhase() int {
	return cb.memory.GetCurrentPhase()
}

// GetTotalPhases returns the total number of phases in the plan.

func (cb *ContextBuilder) GetTotalPhases() int {
	return cb.memory.GetTotalPhases()
}

// FormatPlanDisplay returns a user-facing display of the full plan.

func (cb *ContextBuilder) FormatPlanDisplay() string {
	return cb.memory.FormatPlanDisplay()
}

// MarkStep marks a step as done in the specified phase.

func (cb *ContextBuilder) MarkStep(phase, step int) error {
	return cb.memory.MarkStep(phase, step)
}

// AddStep appends a new step to the given phase.

func (cb *ContextBuilder) AddStep(phase int, desc string) error {
	return cb.memory.AddStep(phase, desc)
}

// ValidatePlanStructure validates plan structure for interview->review transition.

func (cb *ContextBuilder) ValidatePlanStructure() error {
	return cb.memory.ValidatePlanStructure()
}

// SetPlanStatus sets the plan status.

func (cb *ContextBuilder) SetPlanStatus(status string) error {
	return cb.memory.SetStatus(status)
}

// GetPlanWorkDir returns the WorkDir from the plan metadata, or "".

func (cb *ContextBuilder) GetPlanWorkDir() string {
	return cb.memory.GetPlanWorkDir()
}

// GetPlanTaskName returns the task description from the plan metadata, or "".

func (cb *ContextBuilder) GetPlanTaskName() string {
	return cb.memory.GetPlanTaskName()
}

// GetSkillsInfo returns information about loaded skills.

func (cb *ContextBuilder) GetSkillsInfo() map[string]any {
	allSkills := cb.skillsLoader.ListSkills()

	skillNames := make([]string, 0, len(allSkills))

	for _, s := range allSkills {
		skillNames = append(skillNames, s.Name)
	}

	return map[string]any{
		"total": len(allSkills),

		"available": len(allSkills),

		"names": skillNames,
	}
}
