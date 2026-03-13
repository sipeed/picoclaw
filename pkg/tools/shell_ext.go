package tools

import (
	"context"
	"sync"
)

// execToolExt holds fork-specific fields for ExecTool.
// Embedded in ExecTool so existing field access (t.bgProcesses, etc.) continues to work.
type execToolExt struct {
	allowRules [][]string // pre-split command prefix allowlist

	localNetOnly bool // restrict curl/wget to localhost + RFC 1918

	// Background process management

	bgMu sync.Mutex

	bgProcesses map[string]*bgProcess

	bgNextID int

	bgShutdown context.CancelFunc // cancels all bg monitor goroutines

	bgCtx context.Context
}
