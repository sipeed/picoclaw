package gateway

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFileFingerprintDetectsSameSizeSameTimestampChanges(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	initialContent := []byte(`{"model":"aaaa"}`)
	require.NoError(t, os.WriteFile(configPath, initialContent, 0o600))

	original := getFileFingerprint(configPath)
	require.NotZero(t, original.ModTime)
	require.NotZero(t, original.Size)

	updatedContent := []byte(`{"model":"bbbb"}`)
	require.Len(t, updatedContent, len(initialContent))
	require.NoError(t, os.WriteFile(configPath, updatedContent, 0o600))

	// Simulate a coarse-grained filesystem timestamp where two writes inside the
	// same second end up with the same reported modification time.
	require.NoError(t, os.Chtimes(configPath, original.ModTime, original.ModTime))

	updated := getFileFingerprint(configPath)

	assert.Equal(t, original.ModTime, updated.ModTime)
	assert.Equal(t, original.Size, updated.Size)
	assert.NotEqual(t, original.Hash, updated.Hash)
	assert.NotEqual(t, original, updated)
}

func TestGetFileFingerprintMissingFile(t *testing.T) {
	fp := getFileFingerprint(filepath.Join(t.TempDir(), "missing.json"))

	assert.Equal(t, fileFingerprint{}, fp)
	assert.True(t, fp.ModTime.IsZero())
	assert.Equal(t, int64(0), fp.Size)
	assert.Equal(t, [32]byte{}, fp.Hash)
}
