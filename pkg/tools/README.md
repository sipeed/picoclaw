# Tool Security Checklist

When adding a new built-in tool, include these minimum safety checks:

1. Path boundary: if the tool reads/writes files or executes commands with paths, enforce canonical workspace membership when `restrict_to_workspace=true`.
2. Network boundary: if the tool performs outbound network calls, reject loopback/private/link-local/multicast/unspecified/internal targets and validate redirect hops.
3. Timeout behavior: long-running operations must use deterministic timeout/cancel handling and terminate child processes where process trees are possible.
4. Regression tests: add explicit tests for blocked behavior (not just happy-path errors), including redirect/path traversal/process-leak scenarios where relevant.
5. Error clarity: return explicit denial reasons (`blocked destination`, `outside workspace`, `timed out`) so behavior is auditable in logs.
