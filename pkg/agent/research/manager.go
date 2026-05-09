package research

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/memory"
)

// Store defines the interface for research data persistence
type Store interface {
	ListResearchAgents() ([]memory.ResearchAgent, error)
	UpdateResearchAgent(agent memory.ResearchAgent) error
	ListResearchNodes() ([]memory.ResearchNode, error)
	UpdateResearchNode(node memory.ResearchNode) error
	ListResearchReports() ([]memory.ResearchReport, error)
	UpdateResearchReport(report memory.ResearchReport) error
}

// ConfigStore defines the interface for config persistence
type ConfigStore interface {
	GetResearchConfig() (Config, error)
	SaveResearchConfig(config Config) error
}

// FileConfigStore stores config in a JSON file
type FileConfigStore struct {
	configPath string
}

// NewFileConfigStore creates a new file-based config store
func NewFileConfigStore(workspacePath string) *FileConfigStore {
	return &FileConfigStore{
		configPath: filepath.Join(workspacePath, "research", "config.json"),
	}
}

// GetResearchConfig loads research config from file
func (s *FileConfigStore) GetResearchConfig() (Config, error) {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}
	return cfg, nil
}

// SaveResearchConfig saves research config to file
func (s *FileConfigStore) SaveResearchConfig(cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(s.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(s.configPath, data, 0644)
}

// Manager handles research data operations
type Manager struct {
	store Store
}

// NewManager creates a new research manager with the given store
func NewManager(store Store) *Manager {
	return &Manager{store: store}
}

// ListAgents returns all research agents
func (m *Manager) ListAgents() ([]Agent, error) {
	agents, err := m.store.ListResearchAgents()
	if err != nil {
		return nil, err
	}

	result := make([]Agent, len(agents))
	for i, a := range agents {
		result[i] = convertToAgent(a)
	}
	return result, nil
}

// GetAgent returns a single research agent by ID
func (m *Manager) GetAgent(id string) (*Agent, error) {
	agents, err := m.store.ListResearchAgents()
	if err != nil {
		return nil, err
	}

	for _, a := range agents {
		if a.ID == id {
			agent := convertToAgent(a)
			return &agent, nil
		}
	}
	return nil, nil
}

// UpdateAgent updates an existing research agent
func (m *Manager) UpdateAgent(agent Agent) error {
	memoryAgent := convertFromAgent(agent)
	return m.store.UpdateResearchAgent(memoryAgent)
}

// ListNodes returns all research graph nodes
func (m *Manager) ListNodes() ([]Node, error) {
	nodes, err := m.store.ListResearchNodes()
	if err != nil {
		return nil, err
	}

	result := make([]Node, len(nodes))
	for i, n := range nodes {
		result[i] = convertToNode(n)
	}
	return result, nil
}

// UpdateNode updates an existing research node
func (m *Manager) UpdateNode(node Node) error {
	memoryNode := convertFromNode(node)
	return m.store.UpdateResearchNode(memoryNode)
}

// ListReports returns all research reports
func (m *Manager) ListReports() ([]Report, error) {
	reports, err := m.store.ListResearchReports()
	if err != nil {
		return nil, err
	}

	result := make([]Report, len(reports))
	for i, r := range reports {
		result[i] = convertToReport(r)
	}
	return result, nil
}

// GetReport returns a single research report by ID
func (m *Manager) GetReport(id string) (*Report, error) {
	reports, err := m.store.ListResearchReports()
	if err != nil {
		return nil, err
	}

	for _, r := range reports {
		if r.ID == id {
			report := convertToReport(r)
			return &report, nil
		}
	}
	return nil, nil
}

// UpdateReport updates an existing research report
func (m *Manager) UpdateReport(report Report) error {
	memoryReport := convertFromReport(report)
	return m.store.UpdateResearchReport(memoryReport)
}

// Conversion functions between memory types and domain types

func convertToAgent(a memory.ResearchAgent) Agent {
	return Agent{
		ID:       a.ID,
		Name:     a.Name,
		Active:   a.Active,
		Type:     a.Type,
		Progress: a.Progress,
		RAM:      a.RAM,
	}
}

func convertFromAgent(a Agent) memory.ResearchAgent {
	return memory.ResearchAgent{
		ID:       a.ID,
		Name:     a.Name,
		Active:   a.Active,
		Type:     a.Type,
		Progress: a.Progress,
		RAM:      a.RAM,
	}
}

func convertToNode(n memory.ResearchNode) Node {
	return Node{
		Name: n.Name,
		Abbr: n.Abbr,
		X:    n.X,
		Y:    n.Y,
	}
}

func convertFromNode(n Node) memory.ResearchNode {
	return memory.ResearchNode{
		Name: n.Name,
		Abbr: n.Abbr,
		X:    n.X,
		Y:    n.Y,
	}
}

func convertToReport(r memory.ResearchReport) Report {
	return Report{
		ID:       r.ID,
		Title:    r.Title,
		Pages:    r.Pages,
		Words:    r.Words,
		Status:   ReportStatus(r.Status),
		Progress: r.Progress,
	}
}

func convertFromReport(r Report) memory.ResearchReport {
	return memory.ResearchReport{
		ID:       r.ID,
		Title:    r.Title,
		Pages:    r.Pages,
		Words:    r.Words,
		Status:   string(r.Status),
		Progress: r.Progress,
	}
}

// GetConfig returns the current research configuration
func (m *Manager) GetConfig() (Config, error) {
	if configStore, ok := m.store.(ConfigStore); ok {
		return configStore.GetResearchConfig()
	}
	// Return default if store doesn't support config
	return DefaultConfig(), nil
}

// UpdateConfig updates the research configuration
func (m *Manager) UpdateConfig(cfg Config) error {
	if configStore, ok := m.store.(ConfigStore); ok {
		return configStore.SaveResearchConfig(cfg)
	}
	return nil
}