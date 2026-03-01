package sandbox

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/logger"
)

var defaultSeedFiles = []string{
	"AGENTS.md",
	"MEMORY.md",
	"IDENTITY.md",
	"TOOLS.md",
	"SOUL.md",
	"BOOTSTRAP.md",
	"USER.md",
}

// syncAgentWorkspace copies base agent files and the skills directory
// from the agentWorkspace to the isolated container workspace.
func syncAgentWorkspace(agentWorkspace, containerWorkspace string) error {
	if agentWorkspace == "" || containerWorkspace == "" {
		return nil
	}

	// 1. Seed base agent files
	for _, file := range defaultSeedFiles {
		src := filepath.Join(agentWorkspace, file)
		dst := filepath.Join(containerWorkspace, file)

		// Check if source exists
		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			logger.WarnCF("sandbox", "failed to stat seed source file", map[string]any{"file": src, "error": err})
			continue
		}

		// Check if destination already exists. If yes, preserve it.
		if _, err := os.Stat(dst); err == nil {
			continue // preserved
		} else if !os.IsNotExist(err) {
			logger.WarnCF("sandbox", "failed to stat seed destination file", map[string]any{"file": dst, "error": err})
			continue
		}

		if err := copyFile(src, dst); err != nil {
			logger.WarnCF("sandbox", "failed to seed file", map[string]any{"file": file, "error": err})
		}
	}

	// 2. Sync skills directory (complete overwrite)
	skillsSrc := filepath.Join(agentWorkspace, "skills")
	skillsDst := filepath.Join(containerWorkspace, "skills")

	if _, err := os.Stat(skillsSrc); err == nil {
		// Remove existing destination to ensure clean sync
		_ = os.RemoveAll(skillsDst)
		if errCopy := copyDir(skillsSrc, skillsDst); errCopy != nil {
			return fmt.Errorf("failed to sync skills directory: %w", errCopy)
		}
	} else if !os.IsNotExist(err) {
		logger.WarnCF(
			"sandbox",
			"failed to stat skills source directory",
			map[string]any{"dir": skillsSrc, "error": err},
		)
	}

	return nil
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// copyDir recursively copies a directory tree, creating directories and copying files.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(targetPath, info.Mode())
		}

		return copyFile(path, targetPath)
	})
}
