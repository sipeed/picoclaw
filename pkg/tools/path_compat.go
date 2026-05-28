package tools

import (
	"regexp"

	fstools "github.com/sipeed/picoclaw/pkg/tools/fs"
)

func validatePathWithAllowPaths(
	path, workspace string,
	restrict bool,
	patterns []*regexp.Regexp,
) (string, error) {
	return fstools.ValidatePathWithAllowPaths(path, workspace, restrict, patterns)
}

func isAllowedPath(path string, patterns []*regexp.Regexp) bool {
	return fstools.IsAllowedPath(path, patterns)
}
