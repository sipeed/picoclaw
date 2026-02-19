package mcp

// ServerSummary is a lightweight view of a server for listing.
// This is a Manager-specific type (not part of the MCP SDK).
type ServerSummary struct {
	Name        string
	Description string
	Status      string
}
