package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGroupTriggerConfig_MentionOnly_False_Roundtrips documents the regression
// where an explicit `mention_only: false` is silently lost on a JSON roundtrip
// because the struct tag `json:"mention_only,omitempty"` causes Go to omit the
// field whenever its value equals the bool zero value (false).
//
// Chain of damage in production:
//   1. User edits config.json and sets "mention_only": false.
//   2. Gateway loads config → LoadConfig triggers defer SaveConfig on any
//      migration path (config.go:1042/1083/1122).
//   3. SaveConfig re-marshals the struct. `,omitempty` drops the false field.
//   4. The next load merges with defaultChannels() (defaults.go:483-484),
//      which hardcodes `{"mention_only": true}` for the "line" channel.
//   5. User's explicit false silently becomes true.
//
// The fix is to remove `,omitempty` from the tag so false always serializes,
// matching the pattern already used by DiscordConfig.MentionOnly (no omitempty).
func TestGroupTriggerConfig_MentionOnly_False_Roundtrips(t *testing.T) {
	original := GroupTriggerConfig{MentionOnly: false}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	// The bug: with `,omitempty`, the false field is omitted entirely, so the
	// JSON becomes "{}" and the user's explicit choice is indistinguishable
	// from "never set". A caller that fills in defaults for missing fields
	// will then replace false with true.
	assert.Contains(t, string(data), `"mention_only":false`,
		"explicit false must serialize — otherwise defaults logic overwrites it; see defaults.go:483-484")

	var decoded GroupTriggerConfig
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.False(t, decoded.MentionOnly, "mention_only should survive roundtrip as false")
}
