//go:build !windows

package main

import _ "embed"

//go:embed icon.png
var iconData []byte
