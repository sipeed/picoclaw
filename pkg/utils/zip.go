package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// ExtractZipFile extracts a ZIP archive from disk to targetDir.
// It reads entries one at a time from disk, keeping memory usage minimal.
//
// Security: rejects path traversal attempts and symlinks.
func ExtractZipFile(zipPath string, targetDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("invalid ZIP: %w", err)
	}
	defer reader.Close()

	logger.DebugCF("zip", "Extracting ZIP", map[string]interface{}{
		"zip_path":   zipPath,
		"target_dir": targetDir,
		"entries":    len(reader.File),
	})

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target dir: %w", err)
	}

	for _, f := range reader.File {
		// Path traversal protection.
		cleanName := filepath.Clean(f.Name)
		if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
			return fmt.Errorf("zip entry has unsafe path: %q", f.Name)
		}

		destPath := filepath.Join(targetDir, cleanName)

		// Double-check the resolved path is within target.
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(targetDir)) {
			return fmt.Errorf("zip entry escapes target dir: %q", f.Name)
		}

		mode := f.FileInfo().Mode()

		// Reject any symlink.
		if mode&os.ModeSymlink != 0 {
			return fmt.Errorf("zip contains symlink %q; symlinks are not allowed", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return err
			}
			continue
		}

		// Ensure parent directory exists.
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		if err := extractSingleFile(f, destPath); err != nil {
			return err
		}
	}

	return nil
}

// extractSingleFile extracts one zip.File entry to destPath.
func extractSingleFile(f *zip.File, destPath string) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("failed to open zip entry %q: %w", f.Name, err)
	}
	defer rc.Close()

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", destPath, err)
	}

	if _, err := io.Copy(outFile, rc); err != nil {
		_ = outFile.Close()
		_ = os.Remove(destPath)
		return fmt.Errorf("failed to extract %q: %w", f.Name, err)
	}

	if err := outFile.Close(); err != nil {
		_ = os.Remove(destPath)
		return fmt.Errorf("failed to close file %q: %w", destPath, err)
	}

	return nil
}
