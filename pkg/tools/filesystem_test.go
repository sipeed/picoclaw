package tools

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRootMkdirAll verifies that root.MkdirAll (used by atomicWriteFileInRoot) handles all cases:
// single dir, deeply nested dirs, already-existing dirs, and a file blocking a directory path.
func TestRootMkdirAll(t *testing.T) {
	workspace := t.TempDir()
	root, err := os.OpenRoot(workspace)
	if err != nil {
		t.Fatalf("failed to open root: %v", err)
	}
	defer root.Close()

	// Case 1: Single directory
	err = root.MkdirAll("dir1", 0o755)
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(workspace, "dir1"))
	assert.NoError(t, err)

	// Case 2: Deeply nested directory
	err = root.MkdirAll("a/b/c/d", 0o755)
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(workspace, "a/b/c/d"))
	assert.NoError(t, err)

	// Case 3: Already exists — must be idempotent
	err = root.MkdirAll("a/b/c/d", 0o755)
	assert.NoError(t, err)

	// Case 4: A regular file blocks directory creation — must error
	err = os.WriteFile(filepath.Join(workspace, "file_exists"), []byte("data"), 0o644)
	assert.NoError(t, err)
	err = root.MkdirAll("file_exists", 0o755)
	assert.Error(t, err, "expected error when a file exists at the directory path")
}

// TestHostRW_Read_PermissionDenied verifies that hostRW.Read surfaces access denied errors.
func TestHostRW_Read_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test: running as root")
	}
	tmpDir := t.TempDir()
	protected := filepath.Join(tmpDir, "protected.txt")
	err := os.WriteFile(protected, []byte("secret"), 0o000)
	assert.NoError(t, err)
	defer os.Chmod(protected, 0o644) // ensure cleanup

	_, err = (&HostFs{}).ReadFile(protected)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

// TestHostRW_Read_Directory verifies that hostRW.Read returns an error when given a directory path.
func TestHostRW_Read_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := (&HostFs{}).ReadFile(tmpDir)
	assert.Error(t, err, "expected error when reading a directory as a file")
}

// TestRootRW_Read_Directory verifies that rootRW.Read returns an error when given a directory.
func TestRootRW_Read_Directory(t *testing.T) {
	workspace := t.TempDir()
	root, err := os.OpenRoot(workspace)
	assert.NoError(t, err)
	defer root.Close()

	// Create a subdirectory
	err = root.Mkdir("subdir", 0o755)
	assert.NoError(t, err)

	_, err = (&SandboxFs{Workspace: workspace}).ReadFile("subdir")
	assert.Error(t, err, "expected error when reading a directory as a file")
}

// TestHostRW_Write_ParentDirMissing verifies that hostRW.Write creates parent dirs automatically.
func TestHostRW_Write_ParentDirMissing(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "a", "b", "c", "file.txt")

	err := (&HostFs{}).WriteFile(target, []byte("hello"))
	assert.NoError(t, err)

	data, err := os.ReadFile(target)
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

// TestRootRW_Write_ParentDirMissing verifies that rootRW.Write creates
// nested parent directories automatically within the sandbox.
func TestRootRW_Write_ParentDirMissing(t *testing.T) {
	workspace := t.TempDir()

	relPath := "x/y/z/file.txt"
	err := (&SandboxFs{Workspace: workspace}).WriteFile(relPath, []byte("nested"))
	assert.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(workspace, relPath))
	assert.NoError(t, err)
	assert.Equal(t, "nested", string(data))
}

// TestHostRW_Write verifies the hostRW.Write helper function
func TestHostRW_Write(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "atomic_test.txt")
	testData := []byte("atomic test content")

	err := (&HostFs{}).WriteFile(testFile, testData)
	assert.NoError(t, err)

	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, testData, content)

	// Verify it overwrites correctly
	newData := []byte("new atomic content")
	err = (&HostFs{}).WriteFile(testFile, newData)
	assert.NoError(t, err)

	content, err = os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, newData, content)
}

// TestRootRW_Write verifies the rootRW.Write helper function
func TestRootRW_Write(t *testing.T) {
	tmpDir := t.TempDir()

	relPath := "atomic_root_test.txt"
	testData := []byte("atomic root test content")

	erw := &SandboxFs{Workspace: tmpDir}
	err := erw.WriteFile(relPath, testData)
	assert.NoError(t, err)

	root, err := os.OpenRoot(tmpDir)
	assert.NoError(t, err)
	defer root.Close()

	f, err := root.Open(relPath)
	assert.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	assert.NoError(t, err)
	assert.Equal(t, testData, content)

	// Verify it overwrites correctly
	newData := []byte("new root atomic content")
	err = erw.WriteFile(relPath, newData)
	assert.NoError(t, err)

	f2, err := root.Open(relPath)
	assert.NoError(t, err)
	defer f2.Close()

	content, err = io.ReadAll(f2)
	assert.NoError(t, err)
	assert.Equal(t, newData, content)
}
