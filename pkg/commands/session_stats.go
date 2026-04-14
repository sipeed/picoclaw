package commands

type SessionStats struct {
	MessageCount   int
	TokenEstimate  int
	ContextPercent float64
	ContextWindow  int
	SessionKey     string
	SessionUpdated string
	Version        string
	ThinkEnabled   bool
	HasSummary     bool
}
