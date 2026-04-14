package policy

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChecker_ApproveTool(t *testing.T) {
	cfg := Config{
		DisallowedTools:  map[string]bool{"exec": true},
		RequiresApproval: map[string]bool{"write_file": true},
		AllowedTools:     map[string]bool{"read_file": true, "write_file": true, "ls": true},
	}
	c := NewChecker(cfg)
	ctx := context.Background()

	t.Run("Disallowed", func(t *testing.T) {
		req := &agent.ToolApprovalRequest{Tool: "exec"}
		decision, err := c.ApproveTool(ctx, req)
		require.NoError(t, err)
		assert.False(t, decision.Approved)
		assert.Contains(t, decision.Reason, "globally disallowed")
	})

	t.Run("NotWhitelisted", func(t *testing.T) {
		req := &agent.ToolApprovalRequest{Tool: "send_file"}
		decision, err := c.ApproveTool(ctx, req)
		require.NoError(t, err)
		assert.False(t, decision.Approved)
		assert.Contains(t, decision.Reason, "not in the allowed tools whitelist")
	})

	t.Run("RequiresApproval", func(t *testing.T) {
		req := &agent.ToolApprovalRequest{Tool: "write_file"}
		decision, err := c.ApproveTool(ctx, req)
		require.NoError(t, err)
		assert.False(t, decision.Approved)
		assert.Contains(t, decision.Reason, "requires explicit human approval")
	})

	t.Run("Allowed", func(t *testing.T) {
		req := &agent.ToolApprovalRequest{Tool: "read_file"}
		decision, err := c.ApproveTool(ctx, req)
		require.NoError(t, err)
		assert.True(t, decision.Approved)
	})
}
