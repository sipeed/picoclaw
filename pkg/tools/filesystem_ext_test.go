package tools

import (
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestHostFs_Read_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test: running as root")
	}

	tmpDir := t.TempDir()

	protected := filepath.Join(tmpDir, "protected.txt")

	err := os.WriteFile(protected, []byte("secret"), 0o000)

	assert.NoError(t, err)

	defer os.Chmod(protected, 0o644)

	_, err = (&hostFs{}).ReadFile(protected)

	assert.Error(t, err)

	assert.Contains(t, err.Error(), "access denied")
}

func TestHostFs_Read_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := (&hostFs{}).ReadFile(tmpDir)

	assert.Error(t, err, "expected error when reading a directory as a file")
}

func TestSandboxFs_Read_Directory(t *testing.T) {
	workspace := t.TempDir()

	root, err := os.OpenRoot(workspace)

	assert.NoError(t, err)

	defer root.Close()

	err = root.Mkdir("subdir", 0o755)

	assert.NoError(t, err)

	_, err = (&sandboxFs{workspace: workspace}).ReadFile("subdir")

	assert.Error(t, err, "expected error when reading a directory as a file")
}

func TestHostFs_Write_ParentDirMissing(t *testing.T) {
	tmpDir := t.TempDir()

	target := filepath.Join(tmpDir, "a", "b", "c", "file.txt")

	err := (&hostFs{}).WriteFile(target, []byte("hello"))

	assert.NoError(t, err)

	data, err := os.ReadFile(target)

	assert.NoError(t, err)

	assert.Equal(t, "hello", string(data))
}

func TestSandboxFs_Write_ParentDirMissing(t *testing.T) {
	workspace := t.TempDir()

	relPath := "x/y/z/file.txt"

	err := (&sandboxFs{workspace: workspace}).WriteFile(relPath, []byte("nested"))

	assert.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(workspace, relPath))

	assert.NoError(t, err)

	assert.Equal(t, "nested", string(data))
}

func TestHostFs_Write(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "atomic_test.txt")

	testData := []byte("atomic test content")

	err := (&hostFs{}).WriteFile(testFile, testData)

	assert.NoError(t, err)

	content, err := os.ReadFile(testFile)

	assert.NoError(t, err)

	assert.Equal(t, testData, content)

	newData := []byte("new atomic content")

	err = (&hostFs{}).WriteFile(testFile, newData)

	assert.NoError(t, err)

	content, err = os.ReadFile(testFile)

	assert.NoError(t, err)

	assert.Equal(t, newData, content)
}

func TestSandboxFs_Write(t *testing.T) {
	tmpDir := t.TempDir()

	relPath := "atomic_root_test.txt"

	testData := []byte("atomic root test content")

	erw := &sandboxFs{workspace: tmpDir}

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

func TestValidatePath_OutsideWorkspace_IncludesPath(t *testing.T) {
	workspace := t.TempDir()

	outsidePath := filepath.Join(t.TempDir(), "secret.txt")

	_, err := validatePath(outsidePath, workspace, true)

	assert.Error(t, err)

	assert.Contains(t, err.Error(), "access denied")

	assert.Contains(t, err.Error(), workspace)
}
