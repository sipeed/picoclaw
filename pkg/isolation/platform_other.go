//go:build !linux && !windows

package isolation

import (
	"os/exec"

	"github.com/sipeed/picoclaw/pkg/config"
)

func applyPlatformIsolation(cmd *exec.Cmd, isolation config.IsolationConfig, root string) error {
	// Unsupported platforms currently keep the command unchanged. Callers rely on
	// Preflight and higher-level checks to surface unsupported isolation modes.
	return nil
}

func postStartPlatformIsolation(cmd *exec.Cmd, isolation config.IsolationConfig, root string) error {
	return nil
}

func cleanupPendingPlatformResources(cmd *exec.Cmd) {
}
