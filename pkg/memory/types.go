package memory

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
