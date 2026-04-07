package agent

import (
	"fmt"
	"time"
)

type Metrics struct {
	Version    string
	Agent      string
	Route      string
	Model      string
	Complexity int
	Tokens     int
	Processing time.Duration
}

func WrapResponse(rawOutput string, m Metrics) string {
	header := fmt.Sprintf(
		"System: picoclaw-%s\nAgent: %s\nRoute: %s\nModel: %s\nComplexity: %d\nTokens: %d\nProcessing: %.1fs\n\n",
		m.Version, m.Agent, m.Route, m.Model, m.Complexity, m.Tokens, m.Processing.Seconds(),
	)
	return header + rawOutput
}

func calculateComplexity(promptTokens, completionTokens int) int {
	return int(float64(promptTokens)*0.1 + float64(completionTokens)*0.5)
}
