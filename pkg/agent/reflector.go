// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"os"

	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/shell"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// ---------------------------------------------------------------------------
// Runtime — unified execution engine
//
// The Runtime serves two purposes:
//
//  1. Post-LLM processing: runs async processors after the main LLM responds
//     (memory extraction, CoT feedback, error tracking).
//
//  2. Slash commands: handles /{cmd} {args} from users, executed synchronously.
//
// Both share the same MemoryStore and lightweight LLM provider.
// ---------------------------------------------------------------------------

// --- Post-LLM Processing ---------------------------------------------------

// RuntimeInput captures everything that happened during a single agent turn.
type RuntimeInput struct {
	UserMessage    string   // Original user message
	AssistantReply string   // Main LLM's final response
	Intent         string   // Pre-LLM detected intent
	Tags           []string // Pre-LLM extracted tags
	CotPrompt      string   // Generated thinking strategy
	ToolCalls      []ToolCallRecord
	Iterations     int // Number of LLM iterations used
	Score          int // Phase 3 CalcTurnScore result (set by SyncPhase3)
	ChannelKey     string // "channel:chatID" (set by runAgentLoop)
}

// ToolCallRecord captures one tool invocation and its outcome.
type ToolCallRecord struct {
	Name     string
	Error    string        // Empty if success
	Duration time.Duration // How long the tool took
}

// RuntimeProcessor is a single post-LLM processing step.
type RuntimeProcessor interface {
	Name() string
	Process(ctx context.Context, input RuntimeInput, memory *MemoryStore) error
}

// --- Slash Commands ---------------------------------------------------------

// CommandHandler handles a single /{cmd} invocation.
type CommandHandler func(args []string, memory *MemoryStore) string

// CommandDef defines a registered slash command.
type CommandDef struct {
	Name        string // e.g. "memory"
	Usage       string // e.g. "/memory [list|add|search] ..."
	Description string
	Handler     CommandHandler
}

// --- Reflector (Phase 3) ----------------------------------------------------

// Reflector manages post-LLM processors and slash commands.
// This is Phase 3 (Reflect) of the Runtime Loop.
type Reflector struct {
	provider       providers.LLMProvider
	model          string
	processors     []RuntimeProcessor
	commands       map[string]CommandDef
	mu             sync.RWMutex
	timeout        time.Duration
	toolRegistry   *tools.ToolRegistry    // For /shell command
	agentRegistry  *AgentRegistry         // For /show, /list, /switch
	channelManager *channels.Manager      // For /list channels, /switch channel
}


// NewReflector creates a new Reflector (Phase 3) with built-in processors and commands.
func NewReflector(provider providers.LLMProvider, model string) *Reflector {
	r := &Reflector{
		provider: provider,
		model:    model,
		timeout:  30 * time.Second,
		commands: make(map[string]CommandDef),
	}

	// Built-in processors (post-LLM, async).
	// Note: CotEvaluator and MemoryExtractor are intentionally removed from the
	// default pipeline — memory extraction is now handled by MemoryDigestWorker
	// (batch, background) rather than per-turn inline LLM calls.
	r.RegisterProcessor(&ErrorTracker{})

	// Built-in slash commands.
	r.RegisterCommand(CommandDef{
		Name:        "help",
		Usage:       "/help",
		Description: "Show all available commands",
		Handler:     r.cmdHelp,
	})
	r.RegisterCommand(CommandDef{
		Name:        "memory",
		Usage:       "/memory [list|add|delete|edit|search|stats] ...",
		Description: "Manage long-term memory",
		Handler:     cmdMemory,
	})
	r.RegisterCommand(CommandDef{
		Name:        "cot",
		Usage:       "/cot [feedback|stats|history] ...",
		Description: "Manage CoT learning",
		Handler:     cmdCot,
	})
	r.RegisterCommand(CommandDef{
		Name:        "runtime",
		Usage:       "/runtime [status|processors]",
		Description: "Runtime status and diagnostics",
		Handler:     r.cmdRuntimeStatus,
	})
	r.RegisterCommand(CommandDef{
		Name:        "shell",
		Usage:       "/shell <cmd> [args...]",
		Description: "Execute shell command in workspace",
		Handler:     r.cmdShell,
	})

	// System commands (migrated from handleCommand).
	r.RegisterCommand(CommandDef{
		Name:        "show",
		Usage:       "/show [model|channel|agents]",
		Description: "Show current settings",
		Handler:     r.cmdShow,
	})
	r.RegisterCommand(CommandDef{
		Name:        "list",
		Usage:       "/list [models|channels|agents]",
		Description: "List available resources",
		Handler:     r.cmdList,
	})
	r.RegisterCommand(CommandDef{
		Name:        "switch",
		Usage:       "/switch [model|channel] to <name>",
		Description: "Switch model or channel",
		Handler:     r.cmdSwitch,
	})

	return r
}


// RegisterProcessor adds a post-LLM processor.
func (r *Reflector) RegisterProcessor(p RuntimeProcessor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.processors = append(r.processors, p)
}

// RegisterCommand adds a slash command.
func (r *Reflector) RegisterCommand(cmd CommandDef) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[cmd.Name] = cmd
}

// SetTools sets the tool registry for /shell command support.
func (r *Reflector) SetTools(registry *tools.ToolRegistry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.toolRegistry = registry
}

// SetAgentInfo provides the Runtime with agent and channel references
// needed by system commands (/show, /list, /switch).
func (r *Reflector) SetAgentInfo(reg *AgentRegistry, cm *channels.Manager) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agentRegistry = reg
	r.channelManager = cm
}

// ---------------------------------------------------------------------------
// Post-LLM: async execution
// ---------------------------------------------------------------------------

// SyncPhase3 runs the synchronous, low-latency part of Phase 3:
// it calculates the Turn score and returns it. The caller must invoke this
// BEFORE PublishOutbound so that Active Context is ready for the next turn.
// Execution target: < 2ms (pure CPU, no I/O).
func (r *Reflector) SyncPhase3(input RuntimeInput) int {
	score := CalcTurnScore(input)
	logger.DebugCF("reflector", "SyncPhase3 score",
		map[string]any{"score": score, "intent": input.Intent, "tools": len(input.ToolCalls)})
	return score
}

// AsyncPhase3 runs the asynchronous post-turn work: persisting TurnRecord,
// running legacy processors, etc. Call this AFTER PublishOutbound.
func (r *Reflector) AsyncPhase3(input RuntimeInput, memory *MemoryStore, turnStore *TurnStore, activeCtx *ActiveContextStore) {
	if r == nil {
		return
	}

	r.mu.RLock()
	processors := make([]RuntimeProcessor, len(r.processors))
	copy(processors, r.processors)
	r.mu.RUnlock()

	go func() {
		tctx, cancel := context.WithTimeout(context.Background(), r.timeout)
		defer cancel()

		// Run registered processors (currently: ErrorTracker).
		if memory != nil {
			for _, p := range processors {
				select {
				case <-tctx.Done():
					return
				default:
				}
				start := time.Now()
				if err := p.Process(tctx, input, memory); err != nil {
					logger.WarnCF("reflector", "Processor failed",
						map[string]any{"processor": p.Name(), "error": err.Error(),
							"ms": time.Since(start).Milliseconds()})
				}
			}
		}

		// Persist TurnRecord to turns.db.
		if turnStore != nil && input.UserMessage != "" {
			record := TurnRecord{
				Ts:         time.Now().Unix(),
				ChannelKey: input.ChannelKey,
				Score:      input.Score,
				Intent:     input.Intent,
				Tags:       input.Tags,
				Status:     "pending",
				UserMsg:    input.UserMessage,
				Reply:      input.AssistantReply,
				ToolCalls:  input.ToolCalls,
			}
			if err := turnStore.Insert(record); err != nil {
				logger.WarnCF("reflector", "TurnRecord insert failed",
					map[string]any{"error": err.Error()})
			}
		}
	}()
}

// RunPostLLM is kept for backward compatibility. New code should use
// SyncPhase3 + AsyncPhase3 instead.
func (r *Reflector) RunPostLLM(input RuntimeInput, memory *MemoryStore) {
	r.AsyncPhase3(input, memory, nil, nil)
}

// ---------------------------------------------------------------------------
// Slash commands: synchronous execution
// ---------------------------------------------------------------------------

// HandleCommand tries to handle a /{cmd} message.
// Returns (response, true) if handled, ("", false) if not a known command.
func (r *Reflector) HandleCommand(content string, memory *MemoryStore) (string, bool) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "/") {
		return "", false
	}

	parts := strings.Fields(content)
	if len(parts) == 0 {
		return "", false
	}

	cmdName := strings.TrimPrefix(parts[0], "/")
	args := parts[1:]

	r.mu.RLock()
	cmd, ok := r.commands[cmdName]
	r.mu.RUnlock()

	if !ok {
		return "", false // Not our command — let AgentLoop's handleCommand try.
	}

	if memory == nil {
		return "⚠️ Memory store not available", true
	}

	return cmd.Handler(args, memory), true
}

// ListCommands returns a formatted help text for all registered commands.
func (r *Reflector) ListCommands() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("**Runtime Commands**\n\n")
	for _, cmd := range r.commands {
		fmt.Fprintf(&sb, "• `%s` — %s\n", cmd.Usage, cmd.Description)
	}
	return sb.String()
}

// ===========================================================================
// Built-in slash commands
// ===========================================================================

// --- /help ------------------------------------------------------------------

func (r *Reflector) cmdHelp(_ []string, _ *MemoryStore) string {
	var sb strings.Builder
	sb.WriteString("📖 **Available Commands**\n\n")

	r.mu.RLock()
	for _, cmd := range r.commands {
		fmt.Fprintf(&sb, "• `%s` — %s\n", cmd.Usage, cmd.Description)
	}
	r.mu.RUnlock()

	return sb.String()
}

// --- /memory ----------------------------------------------------------------

func cmdMemory(args []string, memory *MemoryStore) string {
	if len(args) == 0 {
		return "Usage: /memory [list|add|delete|edit|search|stats]\n" +
			"  /memory list           — show recent memories\n" +
			"  /memory add <text> #tags — add a memory\n" +
			"  /memory delete <id>    — delete a memory\n" +
			"  /memory edit <id> <text> — edit a memory\n" +
			"  /memory search <query> — search by tags\n" +
			"  /memory stats          — memory statistics"
	}

	switch args[0] {
	case "list":
		limit := 10
		entries, err := memory.ListEntries(limit)
		if err != nil {
			return fmt.Sprintf("❌ Error: %v", err)
		}
		if len(entries) == 0 {
			return "📭 No memories stored yet."
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "📝 **Recent Memories** (%d)\n\n", len(entries))
		for _, e := range entries {
			tags := ""
			if len(e.Tags) > 0 {
				tags = " [" + strings.Join(e.Tags, ", ") + "]"
			}
			preview := e.Content
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			fmt.Fprintf(&sb, "• #%d%s: %s\n", e.ID, tags, preview)
		}
		return sb.String()

	case "add":
		if len(args) < 2 {
			return "Usage: /memory add <text> #tag1 #tag2"
		}
		// Separate content from #tags.
		var content []string
		var tags []string
		for _, a := range args[1:] {
			if strings.HasPrefix(a, "#") {
				tags = append(tags, strings.TrimPrefix(a, "#"))
			} else {
				content = append(content, a)
			}
		}
		text := strings.Join(content, " ")
		if text == "" {
			return "❌ Memory content cannot be empty"
		}
		id, err := memory.AddEntry(text, tags)
		if err != nil {
			return fmt.Sprintf("❌ Failed to add: %v", err)
		}
		return fmt.Sprintf("✅ Memory #%d saved (tags: %v)", id, tags)

	case "search":
		if len(args) < 2 {
			return "Usage: /memory search <tag1> [tag2] ..."
		}
		entries, err := memory.SearchByAnyTag(args[1:])
		if err != nil {
			return fmt.Sprintf("❌ Error: %v", err)
		}
		if len(entries) == 0 {
			return fmt.Sprintf("🔍 No memories found for tags: %v", args[1:])
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "🔍 **Found %d memories**\n\n", len(entries))
		for _, e := range entries {
			tags := ""
			if len(e.Tags) > 0 {
				tags = " [" + strings.Join(e.Tags, ", ") + "]"
			}
			preview := e.Content
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			fmt.Fprintf(&sb, "• #%d%s: %s\n", e.ID, tags, preview)
		}
		return sb.String()

	case "stats":
		tags, _ := memory.ListAllTags()
		entries, _ := memory.ListEntries(9999)
		var sb strings.Builder
		sb.WriteString("📊 **Memory Stats**\n")
		fmt.Fprintf(&sb, "• Total entries: %d\n", len(entries))
		fmt.Fprintf(&sb, "• Total tags: %d\n", len(tags))
		if len(tags) > 0 {
			preview := tags
			if len(preview) > 20 {
				preview = preview[:20]
			}
			fmt.Fprintf(&sb, "• Tags: %s", strings.Join(preview, ", "))
			if len(tags) > 20 {
				fmt.Fprintf(&sb, " ... (+%d more)", len(tags)-20)
			}
			sb.WriteString("\n")
		}
		return sb.String()

	case "delete":
		if len(args) < 2 {
			return "Usage: /memory delete <id>"
		}
		var id int64
		if _, err := fmt.Sscanf(args[1], "%d", &id); err != nil {
			return "❌ Invalid ID. Usage: /memory delete <id>"
		}
		if err := memory.DeleteEntry(id); err != nil {
			return fmt.Sprintf("❌ Failed: %v", err)
		}
		return fmt.Sprintf("✅ Memory #%d deleted", id)

	case "edit":
		if len(args) < 3 {
			return "Usage: /memory edit <id> <new content> #tags"
		}
		var id int64
		if _, err := fmt.Sscanf(args[1], "%d", &id); err != nil {
			return "❌ Invalid ID. Usage: /memory edit <id> <text>"
		}
		var content []string
		var tags []string
		for _, a := range args[2:] {
			if strings.HasPrefix(a, "#") {
				tags = append(tags, strings.TrimPrefix(a, "#"))
			} else {
				content = append(content, a)
			}
		}
		text := strings.Join(content, " ")
		if text == "" {
			return "❌ Content cannot be empty"
		}
		if err := memory.UpdateEntry(id, text, tags); err != nil {
			return fmt.Sprintf("❌ Failed: %v", err)
		}
		return fmt.Sprintf("✅ Memory #%d updated", id)

	default:
		return fmt.Sprintf("Unknown subcommand: %s. Use /memory for help.", args[0])
	}
}

// --- /cot -------------------------------------------------------------------

func cmdCot(args []string, memory *MemoryStore) string {
	if len(args) == 0 {
		return "Usage: /cot [feedback|stats|history]\n" +
			"  /cot feedback <1|0|-1> — rate last CoT strategy\n" +
			"  /cot stats             — show CoT performance\n" +
			"  /cot history [N]       — show recent CoT usage"
	}

	switch args[0] {
	case "feedback":
		if len(args) < 2 {
			return "Usage: /cot feedback <1|0|-1>"
		}
		var score int
		switch args[1] {
		case "1", "+1", "good":
			score = 1
		case "-1", "bad":
			score = -1
		case "0", "neutral":
			score = 0
		default:
			return "❌ Score must be 1 (good), 0 (neutral), or -1 (bad)"
		}
		if err := memory.UpdateLatestCotFeedback(score); err != nil {
			return fmt.Sprintf("❌ Failed: %v", err)
		}
		labels := map[int]string{1: "👍 good", 0: "😐 neutral", -1: "👎 bad"}
		return fmt.Sprintf("✅ CoT feedback recorded: %s", labels[score])

	case "stats":
		stats, err := memory.GetCotStats(30)
		if err != nil || len(stats) == 0 {
			return "📊 No CoT usage data yet."
		}
		var sb strings.Builder
		sb.WriteString("📊 **CoT Stats (last 30 days)**\n\n")
		for _, s := range stats {
			scoreLabel := "neutral"
			if s.AvgScore > 0.3 {
				scoreLabel = "good"
			} else if s.AvgScore < -0.3 {
				scoreLabel = "poor"
			}
			fmt.Fprintf(&sb, "• Intent '%s': %d uses, avg=%s (%.1f)\n",
				s.Intent, s.TotalUses, scoreLabel, s.AvgScore)
		}
		return sb.String()

	case "history":
		limit := 5
		if len(args) > 1 {
			fmt.Sscanf(args[1], "%d", &limit)
		}
		records, err := memory.GetRecentCotUsage(limit)
		if err != nil || len(records) == 0 {
			return "📜 No CoT history yet."
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "📜 **Recent CoT Usage** (%d)\n\n", len(records))
		for _, r := range records {
			fb := "😐"
			if r.Feedback > 0 {
				fb = "👍"
			} else if r.Feedback < 0 {
				fb = "👎"
			}
			tags := ""
			if len(r.Tags) > 0 {
				tags = " [" + strings.Join(r.Tags, ", ") + "]"
			}
			prompt := r.CotPrompt
			if len(prompt) > 80 {
				prompt = prompt[:80] + "..."
			}
			fmt.Fprintf(&sb, "• #%d %s %s%s: %s\n", r.ID, fb, r.Intent, tags, prompt)
		}
		return sb.String()

	default:
		return fmt.Sprintf("Unknown subcommand: %s. Use /cot for help.", args[0])
	}
}

// --- /runtime ---------------------------------------------------------------

func (r *Reflector) cmdRuntimeStatus(args []string, memory *MemoryStore) string {
	if len(args) == 0 {
		return "Usage: /runtime [status|processors|commands]"
	}

	switch args[0] {
	case "status":
		r.mu.RLock()
		nProc := len(r.processors)
		nCmd := len(r.commands)
		r.mu.RUnlock()

		var sb strings.Builder
		sb.WriteString("⚙️ **Runtime Status**\n")
		fmt.Fprintf(&sb, "• Processors: %d\n", nProc)
		fmt.Fprintf(&sb, "• Commands: %d\n", nCmd)
		fmt.Fprintf(&sb, "• Timeout: %s\n", r.timeout)
		if r.model != "" {
			fmt.Fprintf(&sb, "• Model: %s\n", r.model)
		}
		return sb.String()

	case "processors":
		r.mu.RLock()
		defer r.mu.RUnlock()
		var sb strings.Builder
		sb.WriteString("⚙️ **Processors**\n")
		for i, p := range r.processors {
			fmt.Fprintf(&sb, "• %d. %s\n", i+1, p.Name())
		}
		return sb.String()

	case "commands":
		return r.ListCommands()

	default:
		return fmt.Sprintf("Unknown: %s. Use /runtime for help.", args[0])
	}
}

// --- /shell -----------------------------------------------------------------

const shellMaxOutput = 4000

// shellDenySubstrings blocks injection attempts for dev tool passthrough.
var shellDenySubstrings = []string{
	"| sh", "| bash", "| powershell", "| cmd",
	"; rm ", "; del ", "&& rm ", "&& del ",
	"$(", "${", "`",
	"> /dev/", ">> /dev/",
}

func (r *Reflector) cmdShell(args []string, _ *MemoryStore) string {
	if len(args) == 0 {
		return "Usage: /shell <command> [args...]\n" +
			"  Built-in: ls, cat, head, tail, grep, wc, find, diff, tree, stat, pwd, echo\n" +
			"  Dev tools (passthrough): go, git, node, python, npm, cargo, make\n" +
			"  File ops: touch, mkdir, cp, mv"
	}

	baseCmd := strings.ToLower(args[0])
	cmdArgs := args[1:]

	// 1. Try built-in Go implementation (cross-platform).
	if handler, ok := shell.BuiltinCmds[baseCmd]; ok {
		cwd, _ := os.Getwd()
		output := handler(cmdArgs, cwd)
		return shellFormatOutput(output)
	}

	// 2. Try dev tool passthrough via ExecTool.
	if shell.DevToolPassthrough[baseCmd] {
		// Injection check.
		command := strings.Join(args, " ")
		cmdLower := strings.ToLower(command)
		for _, deny := range shellDenySubstrings {
			if strings.Contains(cmdLower, deny) {
				return fmt.Sprintf("❌ Command blocked: restricted pattern '%s'", deny)
			}
		}

		r.mu.RLock()
		registry := r.toolRegistry
		r.mu.RUnlock()

		if registry == nil {
			return "⚠️ Dev tool passthrough not available (no tool registry)"
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result := registry.Execute(ctx, "exec", map[string]any{
			"command": command,
		})

		if result.IsError || result.Err != nil {
			errMsg := result.ForLLM
			if errMsg == "" && result.Err != nil {
				errMsg = result.Err.Error()
			}
			return fmt.Sprintf("❌ %s", errMsg)
		}
		return shellFormatOutput(result.ForLLM)
	}

	return fmt.Sprintf("❌ Unknown command '%s'. Use /shell for available commands.", baseCmd)
}

func shellFormatOutput(output string) string {
	if output == "" {
		return "✅ (no output)"
	}
	if len(output) > shellMaxOutput {
		output = output[:shellMaxOutput] + fmt.Sprintf("\n... (truncated, %d chars total)", len(output))
	}
	return "```\n" + output + "\n```"
}
// --- /show ------------------------------------------------------------------

func (r *Reflector) cmdShow(args []string, _ *MemoryStore) string {
	if len(args) < 1 {
		return "Usage: /show [model|channel|agents]"
	}

	r.mu.RLock()
	reg := r.agentRegistry
	r.mu.RUnlock()

	switch args[0] {
	case "model":
		if reg == nil {
			return "⚠️ Agent registry not available"
		}
		agent := reg.GetDefaultAgent()
		if agent == nil {
			return "No default agent configured"
		}
		return fmt.Sprintf("Current model: %s", agent.Model)
	case "channel":
		return "Use /list channels to see enabled channels"
	case "agents":
		if reg == nil {
			return "⚠️ Agent registry not available"
		}
		ids := reg.ListAgentIDs()
		return fmt.Sprintf("Registered agents: %s", strings.Join(ids, ", "))
	default:
		return fmt.Sprintf("Unknown show target: %s", args[0])
	}
}

// --- /list ------------------------------------------------------------------

func (r *Reflector) cmdList(args []string, _ *MemoryStore) string {
	if len(args) < 1 {
		return "Usage: /list [models|channels|agents]"
	}

	r.mu.RLock()
	reg := r.agentRegistry
	cm := r.channelManager
	r.mu.RUnlock()

	switch args[0] {
	case "models":
		return "Available models: configured in config.json per agent"
	case "channels":
		if cm == nil {
			return "Channel manager not initialized"
		}
		chs := cm.GetEnabledChannels()
		if len(chs) == 0 {
			return "No channels enabled"
		}
		return fmt.Sprintf("Enabled channels: %s", strings.Join(chs, ", "))
	case "agents":
		if reg == nil {
			return "⚠️ Agent registry not available"
		}
		ids := reg.ListAgentIDs()
		return fmt.Sprintf("Registered agents: %s", strings.Join(ids, ", "))
	default:
		return fmt.Sprintf("Unknown list target: %s", args[0])
	}
}

// --- /switch ----------------------------------------------------------------

func (r *Reflector) cmdSwitch(args []string, _ *MemoryStore) string {
	if len(args) < 3 || args[1] != "to" {
		return "Usage: /switch [model|channel] to <name>"
	}

	target := args[0]
	value := args[2]

	r.mu.RLock()
	reg := r.agentRegistry
	cm := r.channelManager
	r.mu.RUnlock()

	switch target {
	case "model":
		if reg == nil {
			return "⚠️ Agent registry not available"
		}
		agent := reg.GetDefaultAgent()
		if agent == nil {
			return "No default agent configured"
		}
		oldModel := agent.Model
		agent.Model = value
		return fmt.Sprintf("Switched model from %s to %s", oldModel, value)
	case "channel":
		if cm == nil {
			return "Channel manager not initialized"
		}
		if _, exists := cm.GetChannel(value); !exists && value != "cli" {
			return fmt.Sprintf("Channel '%s' not found or not enabled", value)
		}
		return fmt.Sprintf("Switched target channel to %s", value)
	default:
		return fmt.Sprintf("Unknown switch target: %s", target)
	}
}

// ===========================================================================
// Built-in processors (post-LLM, async)
// ===========================================================================

// --- ErrorTracker (no LLM) --------------------------------------------------

type ErrorTracker struct{}

func (e *ErrorTracker) Name() string { return "error_tracker" }

func (e *ErrorTracker) Process(_ context.Context, input RuntimeInput, _ *MemoryStore) error {
	for _, tc := range input.ToolCalls {
		if tc.Error == "" {
			continue
		}
		logger.InfoCF("reflector", "Tool error recorded",
			map[string]any{"tool": tc.Name, "error": tc.Error})
	}
	return nil
}

// --- CotEvaluator (LLM) ----------------------------------------------------

type CotEvaluator struct {
	provider providers.LLMProvider
	model    string
}

func (c *CotEvaluator) Name() string { return "cot_evaluator" }

const cotEvalPrompt = `Rate how well the thinking strategy helped answer the user's question.

Question: %s
Strategy: %s
Response (first 500 chars): %s

Respond with ONLY one JSON: {"score": <-1|0|1>}
1 = good, 0 = neutral, -1 = poor`

func (c *CotEvaluator) Process(ctx context.Context, input RuntimeInput, memory *MemoryStore) error {
	if input.CotPrompt == "" {
		return nil
	}

	reply := input.AssistantReply
	if len(reply) > 500 {
		reply = reply[:500]
	}

	resp, err := c.provider.Chat(ctx, []providers.Message{
		{Role: "user", Content: fmt.Sprintf(cotEvalPrompt, input.UserMessage, input.CotPrompt, reply)},
	}, nil, c.model, map[string]any{"max_tokens": 32, "temperature": 0.1})
	if err != nil {
		return fmt.Errorf("eval LLM failed: %w", err)
	}

	// Parse JSON (strip markdown fences if present).
	raw := strings.TrimSpace(resp.Content)
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) > 2 {
			raw = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	var evalResult struct {
		Score int `json:"score"`
	}
	if err := json.Unmarshal([]byte(raw), &evalResult); err != nil {
		// Fallback: string matching.
		if strings.Contains(raw, `"score": 1`) || strings.Contains(raw, `"score":1`) {
			evalResult.Score = 1
		} else if strings.Contains(raw, `"score": -1`) || strings.Contains(raw, `"score":-1`) {
			evalResult.Score = -1
		}
	}

	if evalResult.Score != 0 {
		if err := memory.UpdateLatestCotFeedback(evalResult.Score); err != nil {
			return err
		}
		logger.InfoCF("reflector", "CoT feedback auto-recorded",
			map[string]any{"score": evalResult.Score, "intent": input.Intent})
	}
	return nil
}

// --- MemoryExtractor (LLM) --------------------------------------------------

type MemoryExtractor struct {
	provider providers.LLMProvider
	model    string
}

func (m *MemoryExtractor) Name() string { return "memory_extractor" }

const memoryExtractPrompt = `Extract important facts worth remembering from this conversation.

User: %s
Assistant (first 800 chars): %s

Respond with ONLY JSON: {"memories": [{"content": "<fact>", "tags": ["tag1"]}]}
Rules: max 3 memories, max 3 tags each, lowercase tags, skip trivial chat.
If nothing worth remembering: {"memories": []}`

type memExtractResult struct {
	Memories []struct {
		Content string   `json:"content"`
		Tags    []string `json:"tags"`
	} `json:"memories"`
}

func (m *MemoryExtractor) Process(ctx context.Context, input RuntimeInput, memory *MemoryStore) error {
	if len(input.UserMessage) < 20 || input.Intent == "chat" {
		return nil
	}

	reply := input.AssistantReply
	if len(reply) > 800 {
		reply = reply[:800]
	}

	resp, err := m.provider.Chat(ctx, []providers.Message{
		{Role: "user", Content: fmt.Sprintf(memoryExtractPrompt, input.UserMessage, reply)},
	}, nil, m.model, map[string]any{"max_tokens": 256, "temperature": 0.1})
	if err != nil {
		return fmt.Errorf("memory extract LLM failed: %w", err)
	}

	// Parse JSON (strip markdown fences if present).
	raw := strings.TrimSpace(resp.Content)
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) > 2 {
			raw = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var result memExtractResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil // Parsing failed — skip silently.
	}

	for _, mem := range result.Memories {
		content := strings.TrimSpace(mem.Content)
		if content == "" {
			continue
		}
		tags := make([]string, 0, len(mem.Tags))
		for _, t := range mem.Tags {
			t = strings.ToLower(strings.TrimSpace(t))
			if t != "" {
				tags = append(tags, t)
			}
		}
		if id, err := memory.AddEntry(content, tags); err != nil {
			logger.WarnCF("reflector", "Failed to save memory",
				map[string]any{"error": err.Error()})
		} else {
			logger.InfoCF("reflector", "Memory extracted",
				map[string]any{"id": id, "tags": tags, "content": content})
		}
	}
	return nil
}
