package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpgradeRegistryForConcurrency(t *testing.T) {
	// Create a standard tool registry
	original := NewToolRegistry()

	// Register a mix of tools: some upgradeable, some not
	readTool := NewReadFileTool("", false)
	listTool := NewListDirTool("", false) // Not upgradeable
	writeTool := NewWriteFileTool("", false)

	original.Register(readTool)
	original.Register(listTool)
	original.Register(writeTool)

	// Perform the upgrade
	upgraded := upgradeRegistryForConcurrency(original)

	// Verify count matches
	assert.Equal(t, len(original.ListTools()), len(upgraded.ListTools()), "Upgraded registry should have same number of tools")

	// Verify ReadFileTool got upgraded
	actualReadTool, ok := upgraded.Get("read_file")
	assert.True(t, ok)
	if upgradedRead, isUpgraded := actualReadTool.(*ReadFileTool); isUpgraded {
		_, isConcurrent := upgradedRead.fs.(*ConcurrentFS)
		assert.True(t, isConcurrent, "read_file should have been upgraded to ConcurrentFS")
	}

	// Verify WriteFileTool got upgraded
	actualWriteTool, ok := upgraded.Get("write_file")
	assert.True(t, ok)
	if upgradedWrite, isUpgraded := actualWriteTool.(*WriteFileTool); isUpgraded {
		_, isConcurrent := upgradedWrite.fs.(*ConcurrentFS)
		assert.True(t, isConcurrent, "write_file should have been upgraded to ConcurrentFS")
	}

	// Verify ListDirTool remained the same
	actualListTool, ok := upgraded.Get("list_dir")
	assert.True(t, ok)
	_, isListDir := actualListTool.(*ListDirTool)
	assert.True(t, isListDir, "list_dir should still be ListDirTool")

	// Double check list_dir doesn't randomly have ConcurrentFS injected
	if listImpl, ok := actualListTool.(*ListDirTool); ok {
		_, isConcurrent := listImpl.fs.(*ConcurrentFS)
		assert.False(t, isConcurrent, "list_dir should NOT have ConcurrentFS because it's not upgradeable")
	}

	// Double check original registry was entirely unmodified
	origReadTool, _ := original.Get("read_file")
	if origRead, _ := origReadTool.(*ReadFileTool); origRead != nil {
		_, isConcurrent := origRead.fs.(*ConcurrentFS)
		assert.False(t, isConcurrent, "Original registry components MUST REMAIN completely lock-free")
	}
}
