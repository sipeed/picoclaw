//go:build !cdp

package tools

import (
	"context"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/media"
)

// BrowserTool stub for non-CDP builds.
type BrowserTool struct{}

// NewBrowserTool returns an error when compiled without the cdp build tag.
func NewBrowserTool(_ config.BrowserToolConfig) (*BrowserTool, error) {
	return nil, fmt.Errorf(
		"browser tool not compiled in; rebuild with: go build -tags cdp")
}

// Name implements Tool interface (stub).
func (t *BrowserTool) Name() string { return "browser" }

// Description implements Tool interface (stub).
func (t *BrowserTool) Description() string { return "Browser automation (not compiled)" }

// Parameters implements Tool interface (stub).
func (t *BrowserTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// Execute implements Tool interface (stub).
func (t *BrowserTool) Execute(_ context.Context, _ map[string]any) *ToolResult {
	return ErrorResult("browser tool not compiled in; rebuild with: go build -tags cdp")
}

// SetMediaStore implements mediaStoreAware interface (stub).
func (t *BrowserTool) SetMediaStore(_ media.MediaStore) {}
