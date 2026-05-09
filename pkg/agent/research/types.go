package research

import (
	"time"
)

// ReportStatus represents the status of a research report
type ReportStatus string

const (
	ReportStatusInProgress ReportStatus = "in-progress"
	ReportStatusComplete   ReportStatus = "complete"
)

// Agent represents a research agent
type Agent struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Active   bool       `json:"active"`
	Type     string     `json:"type"`
	Progress int        `json:"progress"`
	RAM      string     `json:"ram"`
	Status   AgentState `json:"status,omitempty"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// AgentState represents the runtime state of a research agent
type AgentState string

const (
	AgentStateIdle       AgentState = "idle"
	AgentStateRunning    AgentState = "running"
	AgentStatePaused     AgentState = "paused"
	AgentStateCompleted  AgentState = "completed"
	AgentStateFailed     AgentState = "failed"
)

// Node represents a node in the research knowledge graph
type Node struct {
	Name string  `json:"name"`
	Abbr string  `json:"abbr"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Type string  `json:"type,omitempty"`
}

// Report represents a research report
type Report struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Pages       int         `json:"pages"`
	Words       int         `json:"words"`
	Status      ReportStatus `json:"status"`
	Progress    int         `json:"progress,omitempty"`
	CreatedAt   time.Time   `json:"created_at,omitempty"`
	UpdatedAt   time.Time   `json:"updated_at,omitempty"`
}

// AgentListResponse represents the response for listing research agents
type AgentListResponse struct {
	Agents []Agent `json:"agents"`
}

// NodeListResponse represents the response for listing research graph nodes
type NodeListResponse struct {
	Nodes []Node `json:"nodes"`
}

// ReportListResponse represents the response for listing research reports
type ReportListResponse struct {
	Reports []Report `json:"reports"`
}

// Config represents research configuration settings
type Config struct {
	Type            string `json:"type"`
	Depth           string `json:"depth"`
	RestrictToGraph bool   `json:"restrict_to_graph"`
}

// DefaultConfig returns default research configuration
func DefaultConfig() Config {
	return Config{
		Type:            "comprehensive",
		Depth:           "deep",
		RestrictToGraph: false,
	}
}