//go:build mipsle || netbsd || (freebsd && arm)

package agent

import (
	"encoding/json"
	"fmt"
)

// newSeahorseContextManager is unavailable on platforms where modernc sqlite/libc
// currently has no stable build path for this project.
func newSeahorseContextManager(_ json.RawMessage, _ *AgentLoop) (ContextManager, error) {
	return nil, fmt.Errorf("seahorse context manager is unavailable on this platform")
}

func init() {
	if err := RegisterContextManager("seahorse", newSeahorseContextManager); err != nil {
		panic(fmt.Sprintf("register seahorse context manager: %v", err))
	}
}
