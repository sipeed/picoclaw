//go:build !bedrock

// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package bedrock

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider_ReturnsStubError(t *testing.T) {
	provider, err := NewProvider(context.Background())

	assert.Nil(t, provider)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "build with -tags bedrock"),
		"error should mention build tag requirement, got: %s", err.Error())
}

func TestNewProvider_WithOptions_ReturnsStubError(t *testing.T) {
	provider, err := NewProvider(context.Background(), WithRegion("us-west-2"), WithProfile("test"))

	assert.Nil(t, provider)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "build with -tags bedrock"),
		"error should mention build tag requirement, got: %s", err.Error())
}
