# Issue #2373: Gateway不支持stop等命令

## Problem
gateway命令不支持stop子命令，用户只能通过killall来停止，但这会导致进程自动重启。

## Analysis
The gateway command implementation may be missing the stop subcommand handler.

## Recommended Fix
1. Add 'stop' subcommand to gateway command in cmd/picoclaw/internal/gateway/command.go
2. Implement graceful shutdown logic
3. Send signal to running gateway process to stop it
4. Handle PID file management if needed
