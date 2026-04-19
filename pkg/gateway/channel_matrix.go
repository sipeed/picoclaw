//go:build !mipsle && !netbsd && !(freebsd && arm) && !android

package gateway

import (
	// Matrix currently pulls in mautrix crypto and modernc sqlite transitively.
	//
	// We exclude it on:
	// - linux/mipsle: mautrix crypto falls back to libolm when the `goolm` build
	//   tag is unavailable, and modernc.org/sqlite/modernc.org/libc also lacks a
	//   working build path for our mipsle + softfloat target.
	// - netbsd/*: modernc.org/sqlite v1.46.1 fails to compile due to broken
	//   generated mutex code on NetBSD (for example sqlite_netbsd_amd64.go calls
	//   mu.enter/mu.leave, but the generated mutex type does not define them).
	// - freebsd/arm: modernc.org/libc v1.67.6 fails to compile due to broken
	//   generated 32-bit FreeBSD code (size_t/uint64 and int32/int64 mismatches
	//   in libc_freebsd.go).
	//
	// This means Matrix is currently unavailable on those targets. The proper
	// long-term fix is to split Matrix basic support from its E2EE/sqlite-backed
	// crypto path, or to upgrade/replace the upstream sqlite dependency once the
	// affected targets are supported.
	_ "github.com/sipeed/picoclaw/pkg/channels/matrix"
)
