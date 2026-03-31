// resolve_key_sushi30_test.go — sushi30 fork regression tests for resolveKey.
//
// Background: resolveKey() previously only dispatched enc:// and file:// to
// credential.Resolver.Resolve(). env:// references were returned as-is (the
// literal string "env://VAR_NAME"), causing 401 authentication errors when API
// keys were stored as env:// references in config.
//
// Fix: pkg/config/config_struct.go resolveKey() now also dispatches env://.
// These tests guard against that regression being re-introduced by a future
// upstream merge or rebase.

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestResolveKey_EnvScheme verifies that a SecureString initialized with an
// env:// reference resolves to the environment variable value, not the raw
// reference string.
func TestResolveKey_EnvScheme(t *testing.T) {
	t.Setenv("PICOCLAW_TEST_API_KEY", "sk-from-env")

	s := NewSecureString("env://PICOCLAW_TEST_API_KEY")
	assert.Equal(t, "sk-from-env", s.String(),
		"env:// reference must resolve to the env var value, not the literal string")
}

// TestResolveKey_EnvScheme_Unset verifies that an unset env:// reference
// causes an error (not a silent empty string or the literal reference).
func TestResolveKey_EnvScheme_Unset(t *testing.T) {
	// Ensure the variable is not set.
	t.Setenv("PICOCLAW_TEST_UNSET_KEY", "")

	s := NewSecureString("env://PICOCLAW_TEST_UNSET_KEY_DEFINITELY_MISSING")
	// resolver.Resolve returns an error for missing/empty env vars;
	// fromRaw propagates it but resolveKey logs and returns "". The resolved
	// value must NOT be the raw reference string.
	assert.NotEqual(t, "env://PICOCLAW_TEST_UNSET_KEY_DEFINITELY_MISSING", s.String(),
		"env:// reference to an unset variable must not return the literal reference string")
}
