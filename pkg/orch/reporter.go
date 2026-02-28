package orch

// AgentReporter is the interface for reporting agent lifecycle events.
// Both Broadcaster (real events) and noopReporter (disabled) implement this.
type AgentReporter interface {
	ReportSpawn(id, label, task string)
	ReportStateChange(id string, state AgentState, tool string)
	ReportConversation(from, to, text string)
	ReportGC(id, reason string)
}

type noopReporter struct{}

func (n *noopReporter) ReportSpawn(id, label, task string)                         {}
func (n *noopReporter) ReportStateChange(id string, state AgentState, tool string) {}
func (n *noopReporter) ReportConversation(from, to, text string)                   {}
func (n *noopReporter) ReportGC(id, reason string)                                 {}

// Noop is the AgentReporter to use when orchestration is disabled.
// Allows nil-free code in callers.
var Noop AgentReporter = &noopReporter{}
