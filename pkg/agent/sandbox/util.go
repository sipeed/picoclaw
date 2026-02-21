package sandbox

import (
	"os/exec"
	"time"
)

func durationMs(ms int64) time.Duration {
	return time.Duration(ms) * time.Millisecond
}

func asExitError(err error, target **exec.ExitError) bool {
	ee, ok := err.(*exec.ExitError)
	if ok {
		*target = ee
	}
	return ok
}
