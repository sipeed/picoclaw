# Known Issues (Resolved)

## 2026-05-08: Invalid Unicode Escapes in Go (Critical Compile Error)

- **File**: `web/backend/api/config.go` lines 399-401
- **Issue**: Invalid `\uXXXX` escape sequences in Go string literals (Go only supports `\UXXXXXXXX` for Unicode code points)
- **Impact**: Prevented backend from compiling
- **Fix**: Replaced `\uXXXX` with `\UXXXXXXXX` format (e.g., `\u200B` → `\U0000200B`)
- **Status**: Fixed, verified via `go build ./web/backend/...`
