package miniapp

import (
	"sync"

	"github.com/sipeed/picoclaw/pkg/skills"
	"github.com/sipeed/picoclaw/pkg/stats"
)

// PlanPhase mirrors agent.PlanPhase for JSON serialization.
type PlanPhase struct {
	Number int        `json:"number"`
	Title  string     `json:"title"`
	Steps  []PlanStep `json:"steps"`
}

// PlanStep mirrors agent.PlanStep for JSON serialization.
type PlanStep struct {
	Index       int    `json:"index"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
}

// PlanInfo represents the plan state exposed via the API.
type PlanInfo struct {
	HasPlan      bool        `json:"has_plan"`
	Status       string      `json:"status"`
	CurrentPhase int         `json:"current_phase"`
	TotalPhases  int         `json:"total_phases"`
	Display      string      `json:"display"`
	Phases       []PlanPhase `json:"phases"`
	Memory       string      `json:"memory"`
}

// SessionInfo represents an active session entry for the API response.
type SessionInfo struct {
	SessionKey  string `json:"session_key"`
	Channel     string `json:"channel"`
	ChatID      string `json:"chat_id"`
	TouchDir    string `json:"touch_dir"`
	ProjectPath string `json:"project_path,omitempty"`
	Purpose     string `json:"purpose,omitempty"`
	Branch      string `json:"branch,omitempty"`
	LastSeenAt  string `json:"last_seen_at"`
	AgeSec      int    `json:"age_sec"`
}

// GitRepoSummary represents a lightweight repo entry for the list view.
type GitRepoSummary struct {
	Name   string `json:"name"`
	Branch string `json:"branch"`
}

// GitInfo represents the git repository state exposed via the API.
type GitInfo struct {
	Name     string      `json:"name"`
	Branch   string      `json:"branch"`
	Commits  []GitCommit `json:"commits"`
	Modified []GitChange `json:"modified"`
}

// GitCommit represents a single commit entry.
type GitCommit struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

// GitChange represents a modified/untracked file entry.
type GitChange struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

// BootstrapFileInfo describes a resolved bootstrap file for the context API.
type BootstrapFileInfo struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Scope string `json:"scope"`
}

// ContextInfo describes the agent's directory context and bootstrap file resolution.
type ContextInfo struct {
	WorkDir     string              `json:"work_dir"`
	PlanWorkDir string              `json:"plan_work_dir"`
	Workspace   string              `json:"workspace"`
	Bootstrap   []BootstrapFileInfo `json:"bootstrap"`
}

// SessionGraphData holds the full session DAG for the Mini App.
type SessionGraphData struct {
	Nodes []SessionGraphNode `json:"nodes"`
	Edges []SessionGraphEdge `json:"edges"`
}

// SessionGraphNode represents a single session in the graph.
type SessionGraphNode struct {
	Key        string `json:"key"`
	ShortKey   string `json:"short_key"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	TurnCount  int    `json:"turn_count"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Summary    string `json:"summary,omitempty"`
	ForkTurnID string `json:"fork_turn_id,omitempty"`
}

// SessionGraphEdge represents a parent→child fork relationship.
type SessionGraphEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	ForkTurnID string `json:"fork_turn_id,omitempty"`
}

// DataProvider is the read-only interface to agent state for the Mini App API.
type DataProvider interface {
	ListSkills() []skills.SkillInfo
	GetPlanInfo() PlanInfo
	GetSessionStats() *stats.Stats
	GetActiveSessions() []SessionInfo
	GetSessionGraph() *SessionGraphData
	GetGitRepos() []GitRepoSummary
	GetGitRepoDetail(name string) GitInfo
	GetContextInfo() ContextInfo
	GetSystemPrompt() string
}

// CommandSender injects a command into the message bus on behalf of a user.
type CommandSender interface {
	SendCommand(senderID, chatID, command string)
}

// DevTarget represents a registered dev server target.
type DevTarget struct {
	ID     string `json:"id"`
	Name   string `json:"name"`   // display name (e.g. "frontend")
	Target string `json:"target"` // URL (e.g. "http://localhost:3000")
}

// DevTargetManager allows tools to register, activate, and deactivate dev proxy targets.
type DevTargetManager interface {
	RegisterDevTarget(name, target string) (id string, err error)
	UnregisterDevTarget(id string) error
	ActivateDevTarget(id string) error
	DeactivateDevTarget() error
	GetDevTarget() string
	ListDevTargets() []DevTarget
}

// StateNotifier broadcasts state-change signals to SSE subscribers.
type StateNotifier struct {
	mu   sync.Mutex
	subs map[chan struct{}]struct{}
	done chan struct{}
}

// NewStateNotifier creates a new StateNotifier.
func NewStateNotifier() *StateNotifier {
	return &StateNotifier{
		subs: make(map[chan struct{}]struct{}),
		done: make(chan struct{}),
	}
}

// Subscribe returns a channel that receives a signal on each state change.
func (n *StateNotifier) Subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	n.mu.Lock()
	n.subs[ch] = struct{}{}
	n.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel.
func (n *StateNotifier) Unsubscribe(ch chan struct{}) {
	n.mu.Lock()
	delete(n.subs, ch)
	n.mu.Unlock()
}

// Close signals all SSE handlers to exit.
func (n *StateNotifier) Close() {
	select {
	case <-n.done:
	default:
		close(n.done)
	}
}

// Done returns a channel that is closed when the notifier is shut down.
func (n *StateNotifier) Done() <-chan struct{} {
	return n.done
}

// Notify sends a signal to all subscribers, coalescing rapid notifications.
func (n *StateNotifier) Notify() {
	n.mu.Lock()
	defer n.mu.Unlock()
	for ch := range n.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
