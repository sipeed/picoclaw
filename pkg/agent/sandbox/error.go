package sandbox

import "errors"

var ErrOutsideWorkspace = errors.New("access denied: symlink resolves outside workspace")
