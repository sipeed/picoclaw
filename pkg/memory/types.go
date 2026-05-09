package memory

// ResearchAgent represents a research agent
type ResearchAgent struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
	Type   string `json:"type"`
	Progress int   `json:"progress"`
	RAM    string `json:"ram"`
}

// ResearchNode represents a node in the research knowledge graph
type ResearchNode struct {
	Name string  `json:"name"`
	Abbr string  `json:"abbr"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

// ResearchReport represents a research report
type ResearchReport struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Pages    int    `json:"pages"`
	Words    int    `json:"words"`
	Status   string `json:"status"` // "in-progress" or "complete"
	Progress  int    `json:"progress,omitempty"`
}

// ResearchReportStore manages research reports
type ResearchReportStore interface {
	ListReports() ([]ResearchReport, error)
	UpdateReport(report ResearchReport) error
}

// ResearchAgentStore manages research agents
type ResearchAgentStore interface {
	ListResearchAgents() ([]ResearchAgent, error)
	UpdateResearchAgent(agent ResearchAgent) error
}

// ResearchGraphStore manages research graph nodes
type ResearchGraphStore interface {
	ListResearchNodes() ([]ResearchNode, error)
	UpdateResearchNode(node ResearchNode) error
}
