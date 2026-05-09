package dashboardauth

import "errors"

// ErrUnsupportedPlatform reports that the SQLite-backed password store is not
// available for the current target platform.
var ErrUnsupportedPlatform = errors.New("dashboard password store is unavailable on this platform")