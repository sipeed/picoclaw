# Shellguard

`pkg/tools/shellguard` is the reusable validation layer for shell commands
before the `exec` tool launches a process. The `exec` tool still owns process
execution, sessions, PTY handling, timeouts, and channel-level remote access
policy. Shellguard owns command safety decisions.

## Responsibilities

Shellguard validates a command using a structured `Decision`:

- `Allowed`: whether execution may continue.
- `Reason`: human-readable explanation for blocked commands.
- `Category`: stable machine-readable reason, such as `dangerous_pattern`,
  `not_allowlisted`, `path_outside_working_dir`, or `permission_mode`.
- `CommandClass`: conservative semantic class: `read_only`, `write`,
  `destructive`, or `unknown`.

Current checks run in this order:

1. Strip quoted heredoc bodies before deny-pattern matching, so PR comments or
   patches containing blocked-looking text do not trip the guard accidentally.
2. Apply deny patterns unless the command matches a custom allow pattern.
3. Apply an optional allowlist.
4. Apply workspace path restrictions when the caller enables them.
5. Apply mode-aware permission policy, such as `read_only`.

The order is intentional: known dangerous patterns and path escapes stay
blocked even when higher-level modes are permissive.

## Permission Modes

`tools.exec.permission_mode` configures mode-aware enforcement.

- Empty string: default behavior. Deny patterns, allowlists, remote-channel
  policy, and workspace path checks still apply, but command class alone does
  not block execution.
- `read_only`: only commands classified as `read_only` are allowed. `write`,
  `destructive`, and `unknown` commands are blocked.

Invalid modes are rejected during config load. Values are normalized for
whitespace and case, so `READ_ONLY` becomes `read_only`.

## Classification Model

Shellguard uses a conservative heuristic classifier, not a full shell parser.
It handles common command separators outside quotes (`;`, `|`, `&&`, `||`, and
single `&`) and combines command classes across a compound command:

1. `destructive` wins over all other classes.
2. `write` wins over `unknown` and `read_only`.
3. `unknown` wins over `read_only`.
4. all-read-only commands remain `read_only`.

This prevents a read-only prefix from hiding a later mutation, for example:

```bash
git status && touch file.txt
git status & touch file.txt
```

Both classify as `write`.

Git and GitHub CLI commands have subcommand-specific handling because their
top-level binaries are not inherently read-only or write-only. For example,
`git fetch`, `git status`, and query-style `git branch` commands are read-only,
while `git config user.name Anton`, `git branch feature`, and `gh pr comment`
are writes.

## Path Scope

When workspace restriction is enabled, shellguard checks absolute path
candidates, resolves symlinks, blocks workspace escapes, and allows configured
external path patterns. It also avoids treating URL path components as local
filesystem paths.

This path logic currently lives inside shellguard because only shell execution
needs it directly. If other tools need the same path-scope behavior, extract a
separate package instead of copying the logic. A future package could expose a
smaller API such as:

```go
type Scope struct {
    Root string
    AllowedPathPatterns []*regexp.Regexp
}

func (s Scope) ValidatePathCandidate(path string) Decision
func (s Scope) ValidateCommandPaths(command string) Decision
```

That should happen only when a second caller needs it.

## Limitations

Shellguard is a safety guard, not a sandbox.

- It does not execute in a separate OS sandbox.
- It does not parse every shell grammar feature.
- It cannot prove arbitrary custom tools are read-only.
- It cannot inspect child processes launched by build tools, scripts, or
  interpreters.
- Unknown commands are blocked only in restrictive modes such as `read_only`.

For stronger isolation, use OS-level sandboxing or run commands in a constrained
container. Shellguard should remain a fast preflight validator that catches
common dangerous or policy-disallowed command shapes before process launch.

## Extension Rules

When changing shellguard:

- Prefer conservative classification. If the validator cannot prove something
  is read-only, classify it as `unknown` or `write`.
- Add regression tests for both `ClassifyCommand` and the relevant validator
  policy path.
- Keep process-execution behavior in `pkg/tools/shell.go`; keep command-safety
  decisions in `pkg/tools/shellguard`.
- Do not add workspace-specific prompt policy here. Shellguard should enforce
  explicit config and tool policy, not agent behavior preferences.
