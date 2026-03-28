package runtimepaths

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func BinaryPath() string {
	if path := firstEnv("JANE_AI_BINARY", "PICOCLAW_BINARY"); path != "" {
		return path
	}
	names := []string{"jane-ai", "picoclaw"}
	if runtime.GOOS == "windows" {
		names = []string{"jane-ai.exe", "picoclaw.exe"}
	}
	if exe, err := os.Executable(); err == nil {
		for _, name := range names {
			path := filepath.Join(filepath.Dir(exe), name)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return path
			}
		}
	}
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return names[len(names)-1]
}
