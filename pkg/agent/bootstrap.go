package agent

import (
	"os"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// bootstrapFiles are the recognized bootstrap file names copied from a config
// directory into the agent workspace.
var bootstrapFiles = []string{"AGENTS.md", "IDENTITY.md", "SOUL.md", "USER.md"}

// CopyBootstrapFiles copies recognized bootstrap files from srcDir into dstDir.
// Missing files in srcDir are silently skipped.
func CopyBootstrapFiles(srcDir, dstDir string) {
	for _, filename := range bootstrapFiles {
		srcPath := filepath.Join(srcDir, filename)
		data, err := os.ReadFile(srcPath)
		if err != nil {
			continue // file not present, skip
		}
		dstPath := filepath.Join(dstDir, filename)
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			logger.WarnCF("agent", "Failed to write bootstrap file",
				map[string]any{"path": dstPath, "error": err.Error()})
		}
	}
}
