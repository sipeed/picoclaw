package service

import (
	"os"
	"path/filepath"
	"strings"
)

func buildSystemdPath(installerPath, brewPrefix string) string {
	paths := make([]string, 0, 24)

	// Deterministic baseline first.
	for _, p := range []string{
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
		"/usr/local/sbin",
		"/usr/sbin",
		"/sbin",
	} {
		paths = appendUniquePath(paths, p)
	}

	// Known Homebrew/Linuxbrew locations.
	for _, p := range []string{
		"/opt/homebrew/bin",
		"/opt/homebrew/sbin",
		"/home/linuxbrew/.linuxbrew/bin",
		"/home/linuxbrew/.linuxbrew/sbin",
	} {
		paths = appendUniquePath(paths, p)
	}

	// If brew --prefix resolves, include it explicitly.
	if strings.TrimSpace(brewPrefix) != "" {
		paths = appendUniquePath(paths, filepath.Join(brewPrefix, "bin"))
		paths = appendUniquePath(paths, filepath.Join(brewPrefix, "sbin"))
	}

	// Managed workspace virtualenv location.
	for _, p := range managedVenvBinCandidates() {
		paths = appendUniquePath(paths, p)
	}

	// Include PATH from the shell running install for custom bins.
	for _, p := range strings.Split(installerPath, string(os.PathListSeparator)) {
		paths = appendUniquePath(paths, p)
	}

	return strings.Join(paths, string(os.PathListSeparator))
}

func managedVenvBinCandidates() []string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return nil
	}
	return []string{
		filepath.Join(home, ".picoclaw", "workspace", ".venv", "bin"),
	}
}

func appendUniquePath(paths []string, path string) []string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return paths
	}
	for _, existing := range paths {
		if existing == trimmed {
			return paths
		}
	}
	return append(paths, trimmed)
}
