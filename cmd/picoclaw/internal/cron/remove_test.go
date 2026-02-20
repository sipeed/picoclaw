package cron

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRemoveSubcommand(t *testing.T) {
	fn := func() string { return "" }
	cmd := newRemoveCommand(fn)

	require.NotNil(t, cmd)

	assert.Equal(t, "Remove a job by ID", cmd.Short)

	assert.True(t, cmd.HasExample())
}
