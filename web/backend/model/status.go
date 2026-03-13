package model

// StatusResponse represents the response payload for the GET /api/status endpoint.
type StatusResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}
