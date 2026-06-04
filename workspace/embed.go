package workspace

import "embed"

// FS embeds the default workspace templates used by `picoclaw onboard`.
//
// Keep this package next to the template files so the repository has a single
// source of truth. The onboard command imports this FS directly instead of
// relying on generated copies under cmd/picoclaw/internal/onboard.
//
//go:embed AGENT.md SOUL.md USER.md memory skills
var FS embed.FS
