package picoclaw

import "embed"

// EmbeddedWorkspace bundles the default workspace templates used by `picoclaw onboard`.
//
//go:embed workspace
var EmbeddedWorkspace embed.FS
