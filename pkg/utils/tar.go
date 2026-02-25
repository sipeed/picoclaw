package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/logger"
)

const maxTarFileSize = 5 * 1024 * 1024 // 5MB, same as zip

// ExtractTarGzFile extracts a gzip-compressed tar archive from disk to targetDir.
// Security: rejects path traversal (.., absolute paths), symlinks, and limits single-file size.
func ExtractTarGzFile(tarGzPath string, targetDir string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return fmt.Errorf("open tar.gz: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("invalid gzip: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	return extractTarReader(tr, targetDir, tarGzPath)
}

func extractTarReader(tr *tar.Reader, targetDir string, srcLabel string) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("failed to create target dir: %w", err)
	}

	targetDirClean := filepath.Clean(targetDir)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
			_, _ = io.CopyN(io.Discard, tr, hdr.Size)
			return fmt.Errorf("tar entry has unsafe path: %q", hdr.Name)
		}

		destPath := filepath.Join(targetDir, cleanName)
		if !strings.HasPrefix(filepath.Clean(destPath), targetDirClean+string(filepath.Separator)) &&
			filepath.Clean(destPath) != targetDirClean {
			_, _ = io.CopyN(io.Discard, tr, hdr.Size)
			return fmt.Errorf("tar entry escapes target dir: %q", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeSymlink, tar.TypeLink:
			return fmt.Errorf("tar contains link %q; symlinks/hardlinks are not allowed", hdr.Name)
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return err
			}
			continue
		case tar.TypeReg, tar.TypeRegA:
			// regular file
		default:
			// skip other types (char, block, fifo, etc.)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		outFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("create %q: %w", destPath, err)
		}
		written, err := io.CopyN(outFile, tr, maxTarFileSize+1)
		outFile.Close()
		if err != nil && err != io.EOF {
			os.Remove(destPath)
			return fmt.Errorf("extract %q: %w", hdr.Name, err)
		}
		if written > maxTarFileSize {
			os.Remove(destPath)
			return fmt.Errorf("tar entry %q exceeds max size (%d bytes)", hdr.Name, written)
		}
	}

	logger.DebugCF("tar", "Extracted tar.gz", map[string]any{
		"src":        srcLabel,
		"target_dir": targetDir,
	})
	return nil
}
