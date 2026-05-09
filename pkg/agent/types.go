package agent

// Research agent type constants
const (
	ResearchAgentLiterature = "literature-analyzer"
	ResearchAgentExtractor = "data-extractor"
	ResearchAgentValidator  = "fact-validator"
	ResearchAgentSynthesizer = "synthesizer"
)

// ResearchAgentConfig holds configuration for research agents
type ResearchAgentConfig struct {
	Type     string `json:"type"`
	Progress int    `json:"progress"`
	RAM      string `json:"ram"`
	Active   bool   `json:"active"`
}
