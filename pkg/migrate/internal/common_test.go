package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandHome(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		result := ExpandHome(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestExpandHomeWithTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result := ExpandHome("~/path")
	assert.Equal(t, home+"/path", result)

	result = ExpandHome("~")
	assert.Equal(t, home, result)
}

func TestResolveWorkspace(t *testing.T) {
	result := ResolveWorkspace("/home/user/.picoclaw")
	assert.Equal(t, "/home/user/.picoclaw/workspace", result)
}

func TestRelPath(t *testing.T) {
	result := RelPath("/home/user/.picoclaw/workspace/file.txt", "/home/user/.picoclaw")
	assert.Equal(t, "workspace/file.txt", result)
}

func TestRelPathError(t *testing.T) {
	result := RelPath("relative/path", "/different/base")
	assert.Equal(t, "path", result)
}

func TestResolveTargetHome(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result, err := ResolveTargetHome("")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".picoclaw"), result)
}

func TestResolveTargetHomeWithOverride(t *testing.T) {
	result, err := ResolveTargetHome("/custom/path")
	require.NoError(t, err)
	assert.Equal(t, "/custom/path", result)
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	sourceFile := filepath.Join(tmpDir, "source.txt")
	err := os.WriteFile(sourceFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	dstFile := filepath.Join(tmpDir, "dest.txt")
	err = CopyFile(sourceFile, dstFile)
	require.NoError(t, err)

	content, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}

func TestCopyFileSourceNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	err := CopyFile(filepath.Join(tmpDir, "nonexistent.txt"), filepath.Join(tmpDir, "dest.txt"))
	require.Error(t, err)
}

func TestPlanWorkspaceMigration(t *testing.T) {
	tmpDir := t.TempDir()
	srcWorkspace := filepath.Join(tmpDir, "src", "workspace")
	dstWorkspace := filepath.Join(tmpDir, "dst", "workspace")

	err := os.MkdirAll(srcWorkspace, 0o755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(srcWorkspace, "file1.txt"), []byte("content"), 0o644)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(srcWorkspace, "subdir"), 0o755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(srcWorkspace, "subdir", "file2.txt"), []byte("content"), 0o644)
	require.NoError(t, err)

	actions, err := PlanWorkspaceMigration(
		srcWorkspace,
		dstWorkspace,
		[]string{"file1.txt"},
		[]string{"subdir"},
		false,
	)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(actions), 1)
}

func TestPlanWorkspaceMigrationWithExistingDestination(t *testing.T) {
	tmpDir := t.TempDir()
	srcWorkspace := filepath.Join(tmpDir, "src", "workspace")
	dstWorkspace := filepath.Join(tmpDir, "dst", "workspace")

	err := os.MkdirAll(srcWorkspace, 0o755)
	require.NoError(t, err)

	err = os.MkdirAll(dstWorkspace, 0o755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(srcWorkspace, "file1.txt"), []byte("source"), 0o644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dstWorkspace, "file1.txt"), []byte("existing"), 0o644)
	require.NoError(t, err)

	actions, err := PlanWorkspaceMigration(
		srcWorkspace,
		dstWorkspace,
		[]string{"file1.txt"},
		[]string{},
		false,
	)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(actions), 1)
	assert.Equal(t, ActionBackup, actions[0].Type)
}

func TestPlanWorkspaceMigrationForce(t *testing.T) {
	tmpDir := t.TempDir()
	srcWorkspace := filepath.Join(tmpDir, "src", "workspace")
	dstWorkspace := filepath.Join(tmpDir, "dst", "workspace")

	err := os.MkdirAll(srcWorkspace, 0o755)
	require.NoError(t, err)

	err = os.MkdirAll(dstWorkspace, 0o755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(srcWorkspace, "file1.txt"), []byte("source"), 0o644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dstWorkspace, "file1.txt"), []byte("existing"), 0o644)
	require.NoError(t, err)

	actions, err := PlanWorkspaceMigration(
		srcWorkspace,
		dstWorkspace,
		[]string{"file1.txt"},
		[]string{},
		true,
	)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(actions), 1)
	assert.Equal(t, ActionCopy, actions[0].Type)
}

func TestPlanWorkspaceMigrationNonExistentSource(t *testing.T) {
	tmpDir := t.TempDir()

	actions, err := PlanWorkspaceMigration(
		filepath.Join(tmpDir, "nonexistent"),
		filepath.Join(tmpDir, "dst", "workspace"),
		[]string{"file1.txt"},
		[]string{},
		false,
	)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, ActionSkip, actions[0].Type)
	assert.Contains(t, actions[0].Description, "source file not found")
}
