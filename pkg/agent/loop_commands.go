package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/commands"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/sipeed/picoclaw/pkg/stats"
)

// buildCommandsRuntime constructs a commands.Runtime wired to the current
// agent and loop state. This is the upstream pattern for providing runtime
// dependencies to command handlers.
func (al *AgentLoop) buildCommandsRuntime(agent *AgentInstance, sessionKey string) *commands.Runtime {
	return &commands.Runtime{
		Config: al.GetConfig(),
		GetModelInfo: func() (string, string) {
			if agent == nil {
				return "unknown", "unknown"
			}
			prov, _ := providers.ExtractProtocol(agent.Model)
			return agent.Model, prov
		},
		ListAgentIDs: func() []string {
			return al.GetRegistry().ListAgentIDs()
		},
		ListDefinitions: func() []commands.Definition {
			return al.cmdRegistry.Definitions()
		},
		GetEnabledChannels: func() []string {
			if al.channelManager == nil {
				return nil
			}
			return al.channelManager.GetEnabledChannels()
		},
		SwitchModel: func(value string) (string, error) {
			if agent == nil {
				return "", fmt.Errorf("no default agent configured")
			}
			old := agent.Model
			agent.Model = value
			return old, nil
		},
		SwitchChannel: func(value string) error {
			if al.channelManager == nil {
				return fmt.Errorf("channel manager not initialized")
			}
			if _, exists := al.channelManager.GetChannel(value); !exists && value != "cli" {
				return fmt.Errorf("channel '%s' not found or not enabled", value)
			}
			return nil
		},
		ClearHistory: func() error {
			if agent == nil || sessionKey == "" {
				return fmt.Errorf("no active session")
			}
			agent.Sessions.SetHistory(sessionKey, nil)
			agent.Sessions.SetSummary(sessionKey, "")
			return agent.Sessions.Save(sessionKey)
		},
		ReloadConfig: func() error {
			if al.reloadFunc != nil {
				return al.reloadFunc()
			}
			return fmt.Errorf("reload not available")
		},
	}
}

// handleCommand processes slash commands. It first tries the upstream
// commands.Executor (for /show, /list, /switch, /check, /clear, /reload, etc.),
// then falls back to fork-specific commands (/session, /skills, /plan, /heartbeat).
func (al *AgentLoop) handleCommand(
	ctx context.Context,
	msg bus.InboundMessage,
	agent *AgentInstance,
	sessionKey string,
) (string, bool) {
	content := strings.TrimSpace(msg.Content)
	if !commands.HasCommandPrefix(content) {
		return "", false
	}

	// Build a reply collector — the Executor calls req.Reply with the response.
	var response string
	replyFn := func(text string) error {
		response = text
		return nil
	}

	rt := al.buildCommandsRuntime(agent, sessionKey)
	exec := commands.NewExecutor(al.cmdRegistry, rt)
	result := exec.Execute(ctx, commands.Request{
		Channel:  msg.Channel,
		ChatID:   msg.ChatID,
		SenderID: msg.SenderID,
		Text:     content,
		Reply:    replyFn,
	})

	if result.Outcome == commands.OutcomeHandled {
		if result.Err != nil {
			return fmt.Sprintf("Command error: %v", result.Err), true
		}
		return response, true
	}

	// Fallback: fork-specific commands not in the upstream registry
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return "", false
	}
	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/session":
		return al.handleSessionCommand(args, msg.SessionKey), true

	case "/skills":
		return al.handleSkillsCommand(), true

	case "/plan":
		resp, handled := al.handlePlanCommand(args, msg.SessionKey)
		if handled {
			al.notifyStateChange()
		}
		return resp, handled

	case "/heartbeat":
		resp, handled := al.handleHeartbeatCommand(args, msg)
		if handled {
			al.notifyStateChange()
		}
		return resp, handled
	}

	return "", false
}

func (al *AgentLoop) handleHeartbeatCommand(args []string, msg bus.InboundMessage) (string, bool) {
	if len(args) == 0 {
		return "Usage: /heartbeat thread [here|off|<thread_id>]", true
	}

	if args[0] != "thread" {
		return "Usage: /heartbeat thread [here|off|<thread_id>]", true
	}

	if len(args) < 2 {
		return "Usage: /heartbeat thread [here|off|<thread_id>]", true
	}

	if msg.Channel != "telegram" {
		return "/heartbeat thread is only supported from Telegram chats.", true
	}

	baseChatID, currentThreadID := splitChatAndThread(msg.ChatID)

	if baseChatID == "" {
		return "Unable to detect Telegram chat ID for heartbeat routing.", true
	}

	arg := strings.ToLower(strings.TrimSpace(args[1]))

	var threadID int

	var err error

	switch arg {
	case "off", "disable", "clear":

		threadID = 0

	case "here", "this":

		if currentThreadID <= 0 {
			return "Current Telegram message is not in a thread. Usage: /heartbeat thread <thread_id>", true
		}

		threadID = currentThreadID

	default:

		threadID, err = strconv.Atoi(arg)

		if err != nil || threadID < 0 {
			return "Usage: /heartbeat thread [here|off|<thread_id>]", true
		}
	}

	al.cfg.Channels.Telegram.HeartbeatThreadID = threadID

	if al.state != nil {
		_ = al.state.SetHeartbeatTarget(fmt.Sprintf("telegram:%s", baseChatID))
	}

	if al.onHeartbeatThreadUpdate != nil {
		al.onHeartbeatThreadUpdate(threadID)
	}

	if al.saveConfig != nil {
		if err := al.saveConfig(al.cfg); err != nil {
			return fmt.Sprintf("Failed to persist config.json: %v", err), true
		}
	}

	if threadID == 0 {
		return fmt.Sprintf("Heartbeat thread routing disabled for chat %s and saved to config.json.", baseChatID), true
	}

	return fmt.Sprintf("Heartbeat thread set to %d for chat %s and saved to config.json.", threadID, baseChatID), true
}

func splitChatAndThread(chatID string) (baseChatID string, threadID int) {
	baseChatID = strings.TrimSpace(chatID)

	if baseChatID == "" {
		return "", 0
	}

	if slash := strings.Index(baseChatID, "/"); slash >= 0 {
		threadPart := strings.TrimSpace(baseChatID[slash+1:])

		baseChatID = strings.TrimSpace(baseChatID[:slash])

		if tid, err := strconv.Atoi(threadPart); err == nil && tid > 0 {
			threadID = tid
		}
	}

	return baseChatID, threadID
}

// handleSessionCommand dispatches /session subcommands.

func (al *AgentLoop) handleSessionCommand(args []string, sessionKey string) string {
	sub := ""

	if len(args) > 0 {
		sub = strings.ToLower(strings.TrimSpace(args[0]))
	}

	switch sub {
	case "list":

		return al.handleSessionList()

	case "graph":

		return al.handleSessionGraph()

	case "fork":

		return al.handleSessionFork(args[1:], sessionKey)

	case "reset":

		if al.stats == nil {
			return "Stats tracking is disabled."
		}

		al.stats.Reset()

		return "Session statistics have been reset."

	default:

		return al.handleSessionStats()
	}
}

func (al *AgentLoop) handleSessionStats() string {
	agent := al.registry.GetDefaultAgent()

	store := agent.Sessions.Store()

	// Session DAG summary

	sessions, _ := store.List(nil)

	var sb strings.Builder

	fmt.Fprintf(&sb, "Sessions: %d in store\n", len(sessions))

	if len(sessions) > 0 {
		active, completed := 0, 0

		for _, s := range sessions {
			switch s.Status {
			case "active":

				active++

			case "completed":

				completed++
			}
		}

		fmt.Fprintf(&sb, "  active=%d completed=%d\n", active, completed)
	}

	sb.WriteString("\nUse: /session list | graph | fork [label]\n")

	// Token stats if available

	if al.stats != nil {
		s := al.stats.GetStats()

		fmt.Fprintf(&sb,

			"\nToken Stats — Today (%s):\n  Prompts: %d  LLM calls: %d  Tokens: %s (in: %s, out: %s)\n"+

				"All time (since %s):\n  Prompts: %d  LLM calls: %d  Tokens: %s (in: %s, out: %s)",

			s.Today.Date,

			s.Today.Prompts,

			s.Today.Requests,

			stats.FormatTokenCount(s.Today.TotalTokens),

			stats.FormatTokenCount(s.Today.PromptTokens),

			stats.FormatTokenCount(s.Today.CompletionTokens),

			s.Since.Format("2006-01-02"),

			s.TotalPrompts,

			s.TotalRequests,

			stats.FormatTokenCount(s.TotalTokens),

			stats.FormatTokenCount(s.TotalPromptTokens),

			stats.FormatTokenCount(s.TotalCompletionTokens),
		)
	}

	return sb.String()
}

// shortSessionKey truncates long session keys for display.

func shortSessionKey(key string) string {
	parts := strings.Split(key, ":")

	if len(parts) > 2 {
		return strings.Join(parts[2:], ":")
	}

	return key
}

func (al *AgentLoop) handleSessionList() string {
	agent := al.registry.GetDefaultAgent()

	store := agent.Sessions.Store()

	sessions, err := store.List(nil)
	if err != nil {
		return fmt.Sprintf("Error listing sessions: %v", err)
	}

	if len(sessions) == 0 {
		return "No sessions in store."
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "Sessions (%d)\n", len(sessions))

	for _, s := range sessions {
		age := time.Since(s.UpdatedAt).Truncate(time.Second)

		label := s.Label

		if label == "" {
			label = shortSessionKey(s.Key)
		}

		parent := ""

		if s.ParentKey != "" {
			parent = " parent=" + shortSessionKey(s.ParentKey)
		}

		fmt.Fprintf(&sb, "- %s [%s] (%s) turns=%d%s\n",

			label, s.Status, age, s.TurnCount, parent)
	}

	return sb.String()
}

func (al *AgentLoop) handleSessionGraph() string {
	agent := al.registry.GetDefaultAgent()

	store := agent.Sessions.Store()

	sessions, err := store.List(nil)
	if err != nil {
		return fmt.Sprintf("Error listing sessions: %v", err)
	}

	if len(sessions) == 0 {
		return "No sessions in store."
	}

	// Build parent→children map and find roots

	byKey := make(map[string]*session.SessionInfo, len(sessions))

	children := make(map[string][]string)

	var roots []string

	for _, s := range sessions {
		byKey[s.Key] = s

		if s.ParentKey == "" {
			roots = append(roots, s.Key)
		} else {
			children[s.ParentKey] = append(children[s.ParentKey], s.Key)
		}
	}

	var sb strings.Builder

	sb.WriteString("Session Graph\n")

	for i, root := range roots {
		last := i == len(roots)-1

		printSessionTree(&sb, root, byKey, children, "", last)
	}

	return sb.String()
}

func printSessionTree(
	sb *strings.Builder,
	key string,
	byKey map[string]*session.SessionInfo,
	children map[string][]string,
	prefix string,
	last bool,
) {
	s := byKey[key]

	if s == nil {
		return
	}

	connector := "├── "

	if last {
		connector = "└── "
	}

	icon := "●"

	if s.Status == "completed" {
		icon = "✓"
	}

	label := s.Label

	if label == "" {
		label = shortSessionKey(s.Key)
	}

	fmt.Fprintf(sb, "%s%s%s %s (turns=%d)\n", prefix, connector, icon, label, s.TurnCount)

	childPrefix := prefix + "│   "

	if last {
		childPrefix = prefix + "    "
	}

	kids := children[key]

	for i, childKey := range kids {
		printSessionTree(sb, childKey, byKey, children, childPrefix, i == len(kids)-1)
	}
}

func (al *AgentLoop) handleSessionFork(args []string, sessionKey string) string {
	if sessionKey == "" {
		return "Cannot fork: no active session key."
	}

	agent := al.registry.GetDefaultAgent()

	store := agent.Sessions.Store()

	label := "fork"

	if len(args) > 0 {
		label = strings.Join(args, " ")
	}

	childKey := sessionKey + ":fork:" + time.Now().Format("20060102T150405")

	err := store.Fork(sessionKey, childKey, &session.CreateOpts{Label: label})
	if err != nil {
		return fmt.Sprintf("Fork failed: %v", err)
	}

	return fmt.Sprintf(
		"Forked session\n  parent: %s\n  child:  %s",
		shortSessionKey(sessionKey),
		shortSessionKey(childKey),
	)
}

type SessionGraphNode struct {
	Key string `json:"key"`

	Label string `json:"label"`

	Status string `json:"status"`

	Summary string `json:"summary"`

	ParentKey string `json:"parent_key"`

	ForkTurnID string `json:"fork_turn_id"`

	TurnCount int `json:"turn_count"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`
}

// GetSessionGraph returns all sessions as a flat list of graph nodes.

func (al *AgentLoop) GetSessionGraph() []SessionGraphNode {
	agent := al.registry.GetDefaultAgent()

	store := agent.Sessions.Store()

	sessions, err := store.List(nil)
	if err != nil {
		return nil
	}

	nodes := make([]SessionGraphNode, 0, len(sessions))

	for _, s := range sessions {
		nodes = append(nodes, SessionGraphNode{
			Key: s.Key,

			Label: s.Label,

			Status: s.Status,

			Summary: s.Summary,

			ParentKey: s.ParentKey,

			ForkTurnID: s.ForkTurnID,

			TurnCount: s.TurnCount,

			CreatedAt: s.CreatedAt,

			UpdatedAt: s.UpdatedAt,
		})
	}

	return nodes
}

// expandSkillCommand detects "/skill <name> [message]" and returns:

//   - expanded: full content with SKILL.md injected (for LLM)

//   - compact: skill name tag + user message only (for history)

//   - ok: whether expansion happened

func (al *AgentLoop) expandSkillCommand(msg bus.InboundMessage) (expanded string, compact string, ok bool) {
	content := strings.TrimSpace(msg.Content)

	if !strings.HasPrefix(content, "/skill ") {
		return "", "", false
	}

	// Parse: /skill <name> [message]

	rest := strings.TrimSpace(content[7:]) // len("/skill ") == 7

	parts := strings.SplitN(rest, " ", 2)

	if len(parts) == 0 || parts[0] == "" {
		return "", "", false
	}

	skillName := parts[0]

	userMessage := ""

	if len(parts) > 1 {
		userMessage = strings.TrimSpace(parts[1])
	}

	agent := al.registry.GetDefaultAgent()

	if agent == nil {
		return "", "", false
	}

	skillContent, found := agent.ContextBuilder.LoadSkill(skillName)

	if !found {
		return "", "", false
	}

	tag := fmt.Sprintf("[Skill: %s]", skillName)

	// Build expanded message: skill instructions + user message (for LLM)

	var sb strings.Builder

	sb.WriteString(tag)

	sb.WriteString("\n\n")

	sb.WriteString(skillContent)

	if userMessage != "" {
		sb.WriteString("\n\n---\n\n")

		sb.WriteString(userMessage)
	}

	// Build compact form: skill name tag + user message only (for history)

	compactForm := tag

	if userMessage != "" {
		compactForm = tag + "\n" + userMessage
	}

	return sb.String(), compactForm, true
}

// handleSkillsCommand lists all available skills.

func (al *AgentLoop) handleSkillsCommand() string {
	agent := al.registry.GetDefaultAgent()

	if agent == nil {
		return "No agent configured."
	}

	skillsList := agent.ContextBuilder.ListSkills()

	if len(skillsList) == 0 {
		return "No skills available.\nAdd skills to your workspace/skills/ directory."
	}

	var sb strings.Builder

	sb.WriteString("Available Skills\n\n")

	for _, s := range skillsList {
		fmt.Fprintf(&sb, "**%s** (%s)\n", s.Name, s.Source)

		if s.Description != "" {
			fmt.Fprintf(&sb, "```\n%s\n```\n", s.Description)
		}
	}

	sb.WriteString("\nUse: /skill <name> [message]")

	return sb.String()
}
