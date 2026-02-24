package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func createTestTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for name, content := range files {
		err := tw.WriteHeader(&tar.Header{Name: name, Size: int64(len(content)), Mode: 0o644})
		require.NoError(t, err)
		_, err = tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return buf.Bytes()
}

func TestExtractTarGzFile_valid(t *testing.T) {
	data := createTestTarGz(t, map[string]string{
		"SKILL.md": "# Test Skill",
		"extra.txt": "hello",
	})
	tmpZip := filepath.Join(t.TempDir(), "skill.tar.gz")
	require.NoError(t, os.WriteFile(tmpZip, data, 0o644))

	targetDir := t.TempDir()
	err := ExtractTarGzFile(tmpZip, targetDir)
	require.NoError(t, err)

	skillMD := filepath.Join(targetDir, "SKILL.md")
	content, err := os.ReadFile(skillMD)
	require.NoError(t, err)
	require.Equal(t, "# Test Skill", string(content))
	extra, _ := os.ReadFile(filepath.Join(targetDir, "extra.txt"))
	require.Equal(t, "hello", string(extra))
}

func TestExtractTarGzFile_pathTraversal_rejected(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	// Use a directory entry so we don't need to consume a body before returning the error.
	err := tw.WriteHeader(&tar.Header{Name: "../../evil", Typeflag: tar.TypeDir, Mode: 0o755})
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	tmpFile := filepath.Join(t.TempDir(), "bad.tar.gz")
	require.NoError(t, os.WriteFile(tmpFile, buf.Bytes(), 0o644))
	targetDir := t.TempDir()

	err = ExtractTarGzFile(tmpFile, targetDir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsafe path")
}

func TestExtractTarGzFile_symlink_rejected(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	err := tw.WriteHeader(&tar.Header{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"})
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	tmpFile := filepath.Join(t.TempDir(), "symlink.tar.gz")
	require.NoError(t, os.WriteFile(tmpFile, buf.Bytes(), 0o644))
	targetDir := t.TempDir()

	err = ExtractTarGzFile(tmpFile, targetDir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "link")
}
