# DDR: Shell Tool Hardening — AST-Based Guard + Interpreter Execution

| Field           | Value                 |
| --------------- | --------------------- |
| **Status**      | IMPLEMENTED           |
| **Date**        | 2026-03-04            |
| **Author**      | —                     |
| **Review date** | 2026-06-04 (3 months) |

---

## 1. Problem Frame

- **System**: PicoClaw `ExecTool` (`pkg/tools/shell.go`) — the shell command execution tool exposed to the LLM agent.
- **Audience**: Maintainers, security reviewers, contributors.
- **Baseline rule**: Commands are guarded by ~40 regex patterns matched against the raw command string (`guardCommand()`). Execution is via `sh -c <command>` (Unix) or `powershell` (Windows). Environment is fully inherited (`cmd.Env` is never set). Directory restriction uses regex-based absolute-path detection.
- **Baseline config** (`ExecConfig`):
  ```go
  EnableDenyPatterns  bool     // default: true
  CustomDenyPatterns  []string // default: []
  CustomAllowPatterns []string // default: []
  ```
- **The problem**:
  1. **Regex guards are trivially bypassable.** Variable indirection (`x=rm; $x -rf /`), command substitution (`` `echo rm` -rf / ``), quoting tricks, IFS manipulation, hex/octal encoding, and Unicode escapes all evade string-level pattern matching. The guard operates on surface syntax, not semantics.
  2. **Full environment inheritance.** Child processes receive every env var from the parent — `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, provider tokens, database credentials. An LLM-generated `curl` command can exfiltrate these.
  3. **No structured risk classification.** The system is binary (blocked / allowed) with no granularity, no feedback to the LLM about _why_ a command was blocked, and no user-configurable risk threshold.
  4. **Path validation is regex-based.** Absolute path detection via regex is incomplete and bypassable via symlinks, relative path tricks, and shell expansion.

- **Proposed change**: Replace the regex guard and `sh -c` execution with (a) a proper shell parser (`mvdan.cc/sh/v3/syntax`), (b) an in-process shell interpreter (`mvdan.cc/sh/v3/interp`) with `ExecHandlers` middleware for runtime command interception, (c) a 4-level risk classifier, (d) env-var allowlisting, and (e) `OpenHandler`-based file-access sandboxing. Hard switch — no fallback to the old system.

- **Constraints**:
  - MUST NOT introduce a new external runtime dependency (no Docker, chroot, or OS-level sandboxing required).
  - MUST cover every command currently blocked by regex deny patterns.
  - MUST preserve the `Tool` interface contract (`Name`, `Description`, `Parameters`, `Execute`).
  - MUST work cross-platform (interpreter replaces both `sh -c` and `powershell` paths).
  - Config changes MUST be additive where possible; removed fields MUST produce a logged migration warning.

- **Non-goals**:
  - OS-level sandboxing (namespaces, seccomp, chroot).
  - Full bash compatibility (job control, complete `shopt`/`set` support).
  - Web-UI or approval workflows for blocked commands.

- **⚠ Hard limitation — external program file access**:

  > **Executed programs can read and write any file the OS process has access to.** The `OpenHandler` only intercepts file opens from _shell redirections_ (`>`, `<`, `>>`). When the interpreter execs an external binary (e.g., `cat /etc/shadow`, `python script.py`, `tar czf /tmp/x.tar /`), that binary runs as a normal OS process with the full filesystem permissions of the picoclaw user. **This design does NOT sandbox external program I/O.** Mitigating this requires OS-level isolation (namespaces, seccomp, chroot) which is explicitly out of scope. The risk classifier and env sanitization reduce the _likelihood_ of damage but do not eliminate filesystem access.

- **Stakeholders**: Maintainers (implementation), self-hosters (config migration), LLM agent loop (structured error contract), channel integrations (no change).

- **Evidence**:
  - Current regex bypass is demonstrable: `x=rm; $x -rf /` passes all 40 deny patterns.
  - `cmd.Env` is never set — confirmed by code audit of `shell.go` L198–L212.
  - `mvdan.cc/sh/v3`: 244 importers, BSD-3-Clause, v3.12.0, actively maintained (powers `shfmt`).
  - `go-safe-cmd-runner` risk model reviewed: 4-level classification (low/medium/high/critical), env allowlisting, path validation. Concept adopted; no code dependency (0-star repo, 3/4 contributors are bots).

- **Acceptance criteria**:
  - [x] AC-1: All 40+ existing regex deny-pattern test cases pass under the new system (no security regression).
  - [x] AC-2: Variable indirection (`x=rm; $x -rf /`), command substitution, and quoting tricks are blocked at runtime via `ExecHandlers`.
  - [x] AC-3: Child processes receive only allowlisted env vars; `*_KEY`, `*_TOKEN`, `*_SECRET` vars are absent.
  - [x] AC-4: Commands classified above the configured `risk_threshold` return a structured `ToolResult` containing risk level, command name, and reason.
  - [x] AC-5: Shell redirections (`> /etc/passwd`) to paths outside the workspace are blocked by `OpenHandler`.
  - [x] AC-6: Common LLM-generated commands (`grep`, `find`, `cat`, `python -c`, `wc`, `jq`) execute successfully.
  - [x] AC-7: Deprecated config fields (`enable_deny_patterns`, `custom_deny_patterns`, `custom_allow_patterns`) produce a logged warning with migration guidance.
  - [x] AC-8: Cron tool uses the same guard system.

---

## 2. Decision

### 2.1 Definitions

- **Risk level**: One of `low`, `medium`, `high`, `critical` — a classification of the potential damage a resolved shell command can cause.
- **Risk threshold**: The maximum risk level the system will allow to execute. Commands above this level are blocked.
- **Env allowlist**: The set of environment variable names permitted to propagate to child processes. Everything else is stripped.
- **Resolved command**: The actual binary path and arguments after all shell expansion (variables, globs, substitution) has been performed by the interpreter.

### 2.2 Rule Changes

1. The `ExecTool` MUST parse all command strings using `mvdan.cc/sh/v3/syntax` with `Variant(syntax.LangBash)`. Unparseable input MUST be rejected with a structured error.

2. The `ExecTool` MUST execute commands using `mvdan.cc/sh/v3/interp.Runner` instead of `os/exec` with `sh -c` or `powershell`. The Windows-specific PowerShell code path MUST be removed.

3. The `interp.Runner` MUST be configured with an `ExecHandlers` middleware that intercepts every resolved external command and classifies it against a risk table before execution.

4. The risk classifier MUST implement four levels:

   | Level      | Default action | Characterization                                        |
   | ---------- | -------------- | ------------------------------------------------------- |
   | `low`      | Allow          | Read-only, informational (ls, cat, grep, find, wc)      |
   | `medium`   | Allow (logged) | File modification, network read (cp, mv, curl GET)      |
   | `high`     | Block          | Destructive, system-modifying (rm, chmod, git push)     |
   | `critical` | Block          | Privilege escalation, always dangerous (sudo, dd, eval) |

   Shell interpreters (`sh`, `bash`, `zsh`, `dash`, `fish`, `ksh`, `csh`, `tcsh`, `powershell`, `pwsh`, `cmd`) MUST be classified as `critical` because they can execute arbitrary nested commands that bypass the risk classifier entirely (e.g., `sh -c 'rm -rf /'`).

5. The risk classifier MUST apply argument-aware modifiers (e.g., `curl` is `medium`, but `curl -X POST` is `high`; `git` is `medium`, but `git push` is `high`). All matching modifiers MUST be scanned and the highest level that exceeds the base level is applied (highest-match-wins, not first-match-wins).

5a. `risk_overrides` sets the **base level** for a command (replacing the built-in table entry). Argument modifiers MUST still be applied on top of the overridden base level and can elevate it further. This means `risk_overrides: {"rm": "medium"}` allows plain `rm` but `rm -rf` is still elevated to `critical` by the built-in modifier.

6. When a command is blocked, the `ToolResult` MUST include:
   - Risk level of the command.
   - The blocked command name and arguments.
   - The configured threshold.
   - A human-readable reason string.

7. The `ExecTool` MUST construct a sanitized environment for the `interp.Runner` using a strict allowlist. The default allowlist MUST be:

   ```
   PATH, HOME, USER, LANG, SHELL, TERM, PWD, OLDPWD,
   HOSTNAME, LOGNAME, TZ, DISPLAY, TMPDIR, EDITOR, PAGER
   ```

   Plus all variables matching the `LC_*` prefix.

8. The `interp.Runner` MUST be configured with an `OpenHandler` that validates all file-open paths (from shell redirections) resolve within the configured workspace directory. Paths to safe pseudo-devices (`/dev/null`, `/dev/zero`, `/dev/urandom`, `/dev/stdin`, `/dev/stdout`, `/dev/stderr`) MUST be exempted.

9. The existing regex-based guard (`defaultDenyPatterns`, `guardCommand()`) MUST be removed entirely. The `ExecConfig` fields `EnableDenyPatterns`, `CustomDenyPatterns`, and `CustomAllowPatterns` MUST be removed from the struct.

10. The `ExecConfig` struct MUST be extended with:

    ```go
    RiskThreshold  string                          `json:"risk_threshold"`   // "low"|"medium"|"high"|"critical"; default "medium"
    RiskOverrides  map[string]string               `json:"risk_overrides"`   // command → level override
    EnvAllowlist   []string                        `json:"env_allowlist"`    // extra vars to pass (extends defaults)
    EnvSet         map[string]string               `json:"env_set"`          // explicit var=value pairs
    ArgModifiers   map[string][]ArgModifierConfig   `json:"arg_modifiers"`    // command → argument-aware risk adjustments (extends built-ins)
    ```

    `ArgModifierConfig` is `struct { Args []string; Level string }`. User-defined modifiers are checked alongside built-ins using highest-match-wins semantics (the maximum level across all matching modifiers is applied).

11. If deprecated config fields (`enable_deny_patterns`, `custom_deny_patterns`, `custom_allow_patterns`) are present in user config, the system SHOULD log a warning with migration instructions. The system MUST NOT fail to start.

12. The cron tool (`pkg/tools/cron.go`) MUST use the same `ExecTool` with the same guard system.

13. The `ExecTool` MUST implement the `AsyncTool` interface (`SetCallback(AsyncCallback)`). When the LLM passes `background=true` and a callback is registered, the command MUST be launched in a goroutine; the result is delivered via the callback. When `background=true` but no callback is registered, execution falls through to synchronous mode. Compile-time interface check: `var _ AsyncTool = (*ExecTool)(nil)`.

14. The implementation MUST use a subpackage structure: `pkg/tools/shell/` contains the core logic (risk classifier, env sanitizer, sandbox, runner) and `pkg/tools/shell_tool.go` is the thin adapter implementing the `Tool` + `AsyncTool` interfaces. Tests live alongside their source in both locations.

### 2.3 Migration

- **Config**: Old fields are silently ignored with a logged warning. No config version bump required. Add a migration note to `docs/tools_configuration.md`.
- **Behavioral**: Commands that previously passed regex checks but are genuinely dangerous (variable indirection bypasses) will now be blocked. This is intentional and constitutes the security fix.
- **Backward compatibility**: The `Tool` interface (`Name`, `Description`, `Parameters`, `Execute`) is unchanged. Callers (`ToolRegistry`, `RunToolLoop`, agent instance) require no changes.

---

## 3. Rationale

### Core reasons

- Regex pattern matching against raw shell strings is a fundamentally wrong abstraction level. Shell is a programming language; security analysis requires parsing it as one.
- `mvdan.cc/sh/v3` provides a battle-tested parser (powers `shfmt`, 244+ importers) and an interpreter with middleware hooks (`ExecHandlers`, `OpenHandler`, `Env`) designed exactly for sandboxed execution.
- Runtime interception via `ExecHandlers` catches dynamic command construction (variable expansion, command substitution, arithmetic evaluation) that static analysis cannot.
- Env inheritance is a silent data-exfiltration vector. The LLM generates arbitrary commands; any `curl`/`wget`/`nc` call can leak every secret in the parent process environment.
- A 4-level risk classifier with structured LLM feedback is strictly better than binary block/allow — it lets the agent retry with safer alternatives.

### Alternatives considered

| #   | Alternative                                                                                | Disposition                   | Why                                                                                                                                                                                           |
| --- | ------------------------------------------------------------------------------------------ | ----------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Do nothing**                                                                             | Rejected                      | Regex guards are demonstrably bypassable. Env leak is an active vulnerability.                                                                                                                |
| 2   | **Improve regexes** (add more patterns, normalize input before matching)                   | Rejected                      | Arms race against shell syntax. Every new evasion requires a new regex. Fundamentally wrong abstraction. Input normalization is itself a parsing problem.                                     |
| 3   | **Static AST analysis only** (parse with `mvdan.cc/sh`, walk tree, keep `sh -c` execution) | Rejected                      | Cannot resolve dynamic constructs (`$x`, `$(...)`) at parse time. Catches structural patterns but misses the primary bypass vector (variable indirection).                                    |
| 4   | **Hybrid: static AST + `sh -c` with improved guards**                                      | Rejected                      | Two systems to maintain, still bypassable at exec time. Adds complexity without solving the core problem.                                                                                     |
| 5   | **OS-level sandboxing** (namespaces, seccomp-bpf, chroot)                                  | Rejected as primary mechanism | Requires elevated privileges, platform-specific, heavy. Out of scope. Could be layered on top later.                                                                                          |
| 6   | **Import `go-safe-cmd-runner` as dependency**                                              | Rejected                      | Overkill framework (TOML config, SUID privilege management, ULID audit trails). 0 stars, 3/4 contributors are bots. The useful concept (risk levels + env allowlist) is simple enough to own. |

### Trade-offs

| Dimension         | Cost                                                                                                                                                                                                                                                 | Benefit                                                                                                                                                                                                                                            |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Complexity**    | New dependency (`mvdan.cc/sh/v3`), new risk table to maintain                                                                                                                                                                                        | Eliminates regex arms race, single enforcement point                                                                                                                                                                                               |
| **Compatibility** | Interpreter is not 100% bash (no job control, limited shopt)                                                                                                                                                                                         | Covers >95% of LLM-generated commands; bash-isms parse and execute                                                                                                                                                                                 |
| **Performance**   | Parse step adds <1ms per command; interpreter overhead comparable to `sh -c`                                                                                                                                                                         | No meaningful regression for typical agent commands                                                                                                                                                                                                |
| **Binary size**   | `mvdan.cc/sh/v3` adds ~450 KB to the compiled binary (parser + interpreter + syntax tables)                                                                                                                                                          | Acceptable for a security-critical component; no runtime memory cost beyond parse/exec                                                                                                                                                             |
| **Safety**        | **`OpenHandler` only catches shell redirections.** External programs (`cat`, `python`, `tar`, etc.) can still read/write any file the process user has access to — this is an OS-level limitation, not solvable without namespace/seccomp isolation. | Significant improvement over regex path detection for shell-level I/O. Env sanitization prevents secret exfiltration via env. Risk classifier blocks known-dangerous commands. But **filesystem access by executed binaries remains unsandboxed.** |
| **Usability**     | Some previously-allowed commands may be blocked by risk classifier                                                                                                                                                                                   | Configurable threshold + per-command overrides; structured feedback enables LLM retry                                                                                                                                                              |
| **Migration**     | Config fields removed; users with custom patterns lose them                                                                                                                                                                                          | Clear migration path; new system is strictly more expressive                                                                                                                                                                                       |

### Risks and mitigations

| Risk                                                | Likelihood | Impact                                  | Mitigation                                                                                                                                                                                                                                                                                                                                                                         |
| --------------------------------------------------- | ---------- | --------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Interpreter fails on valid bash edge case           | Medium     | Medium (command fails)                  | Hard switch makes failures visible immediately. LangBash covers most LLM output. Users can file bugs against `mvdan/sh`.                                                                                                                                                                                                                                                           |
| Risk table misclassifies a command (false positive) | Medium     | Low (command blocked, LLM retries)      | Per-command `risk_overrides` config. Structured error tells LLM what happened.                                                                                                                                                                                                                                                                                                     |
| Risk table misclassifies a command (false negative) | Low        | High (dangerous command runs)           | Defense in depth: env sanitization limits damage. OpenHandler blocks shell-level file access. Risk table starts conservative.                                                                                                                                                                                                                                                      |
| **External programs read/write arbitrary files**    | **High**   | **High** (data exfil, data destruction) | **Not mitigated by this design.** `cat`, `python`, `curl --upload-file`, `tar` etc. run as normal OS processes with full FS access. Risk classifier blocks _known_ dangerous commands but cannot prevent a novel `python -c "open('/etc/passwd').read()"`. OS-level sandboxing (future work) is the only real fix. Env sanitization reduces secret exposure but not file exposure. |
| `mvdan.cc/sh/v3` dependency becomes unmaintained    | Low        | Medium                                  | Widely used (shfmt), BSD-licensed, vendorable. Could fork if needed.                                                                                                                                                                                                                                                                                                               |
| Binary size increase (~450 KB)                      | Certain    | Low                                     | One-time cost. Acceptable for security infrastructure. No runtime memory overhead beyond parse/exec.                                                                                                                                                                                                                                                                               |
| Performance regression on command-heavy agent loops | Low        | Low                                     | Parser is <1ms. Benchmark before/after in CI.                                                                                                                                                                                                                                                                                                                                      |

---

## 4. Consequences

### Immediate impacts

| Who / what                                                    | Impact                                                                                                         |
| ------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| `pkg/tools/shell_tool.go`                                     | Thin adapter: `ExecTool` struct implementing `Tool` + `AsyncTool`. Delegates to `pkg/tools/shell/` subpackage. |
| `pkg/tools/shell_tool_test.go`                                | Tests for `ExecTool` sync/async behavior and interface compliance.                                             |
| `pkg/tools/shell_process_unix.go`, `shell_process_windows.go` | Removed. Interpreter manages process lifecycle.                                                                |
| `pkg/config/config.go` (`ExecConfig`)                         | Three fields removed, four fields added.                                                                       |
| `pkg/config/defaults.go`                                      | Default `RiskThreshold` set to `"medium"`.                                                                     |
| `pkg/tools/cron.go`                                           | Updated to use new `ExecTool` constructor.                                                                     |
| `docs/tools_configuration.md`                                 | Rewritten for new config fields and risk model.                                                                |
| `config/config.example.json`                                  | Updated.                                                                                                       |
| `go.mod`                                                      | `mvdan.cc/sh/v3` added.                                                                                        |
| Self-hosters with `custom_deny_patterns`                      | Logged warning on startup. Patterns no longer functional. Must migrate to `risk_overrides`.                    |
| LLM agent loop                                                | No code change. Receives richer error messages on blocked commands.                                            |

### New files

| File                              | Purpose                                                                                                                                    |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `pkg/tools/shell/risk.go`         | Risk classifier: command→level mapping, argument modifiers (built-in + configurable), `ClassifyCommand` with highest-match-wins semantics. |
| `pkg/tools/shell/risk_test.go`    | Table-driven tests for all 4 levels, argument modifiers, overrides, extra modifiers.                                                       |
| `pkg/tools/shell/env.go`          | Env sanitization: allowlist builder, default list, `LC_*` prefix matching, `env_set` application.                                          |
| `pkg/tools/shell/env_test.go`     | Verify secrets stripped, allowlist passed, `env_set` applied.                                                                              |
| `pkg/tools/shell/sandbox.go`      | `OpenHandler` implementation, path-within-workspace validation, pseudo-device exemptions.                                                  |
| `pkg/tools/shell/sandbox_test.go` | Redirect inside/outside workspace, symlink escape, safe-path exemption.                                                                    |
| `pkg/tools/shell/runner.go`       | `Run` function: parser + interpreter + `ExecHandlers` middleware integration.                                                              |
| `pkg/tools/shell/runner_test.go`  | End-to-end runner tests: timeout, working dir, env sanitization, pipelines.                                                                |
| `pkg/tools/shell_tool.go`         | Adapter: `ExecTool` struct, `NewExecToolWithConfig`, `AsyncTool` impl, arg modifier wiring.                                                |
| `pkg/tools/shell_tool_test.go`    | `ExecTool` sync/async tests, interface compliance checks.                                                                                  |
| `pkg/tools/cron_exec_test.go`     | AC-8: cron-originated `ExecTool` blocks dangerous commands identically to agent-created one; safe commands pass.                            |

### Follow-up tasks

| #   | Task                                                                                        | Owner (role)    | Blocks | Status   |
| --- | ------------------------------------------------------------------------------------------- | --------------- | ------ | -------- |
| 1   | Implement risk classifier (`risk.go`) with command table                                    | Maintainer      | 3, 4   | **Done** |
| 2   | Implement env sanitization (`env.go`)                                                       | Maintainer      | 4      | **Done** |
| 3   | Implement sandbox OpenHandler (`sandbox.go`)                                                | Maintainer      | 4      | **Done** |
| 4   | Rewrite shell tool: parser + interpreter + middleware (`shell/runner.go` + `shell_tool.go`) | Maintainer      | 5, 6   | **Done** |
| 5   | Port all existing test cases + add bypass tests                                             | Maintainer      | 7      | **Done** |
| 6   | Update `ExecConfig`, defaults, migration warning                                            | Maintainer      | 7      | **Done** |
| 7   | Implement `AsyncTool` on `ExecTool`                                                         | Maintainer      | —      | **Done** |
| 8   | Implement configurable `ArgModifiers` (user-defined, highest-match-wins)                    | Maintainer      | —      | **Done** |
| 9   | Fix runner_test PATH resolution for external binaries in sandboxed interpreter              | Maintainer      | —      | **Done** |
| 10  | Update cron tool, docs, config example                                                      | Maintainer      | —      | **Done** |
| 11  | CI: add benchmark comparison (old vs new exec latency)                                      | Maintainer      | —      | Open     |
| 12  | Run against corpus of real LLM-generated commands (compatibility)                           | QA / Maintainer | —      | Open     |

### Test / verification plan

| AC   | Test                                                   | Method                                                                                                                                                    |
| ---- | ------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| AC-1 | Port all 40+ regex deny-pattern test cases             | Unit tests in `shell_test.go` — same inputs, same block expectations                                                                                      |
| AC-2 | Variable indirection, cmd substitution, quoting bypass | New unit tests: `x=rm; $x -rf /`, `` `echo rm` -rf / ``, `'r'"m"' -rf /`                                                                                  |
| AC-3 | Env sanitization                                       | Unit test: set `OPENAI_API_KEY=secret` in parent, verify absent in child env. Verify `PATH`, `HOME` present.                                              |
| AC-4 | Structured error on blocked command                    | Unit test: execute `sudo ls`, verify `ToolResult.ForLLM` contains risk level, command, threshold, reason. Verify `IsError == true`.                       |
| AC-5 | OpenHandler path validation                            | Unit test: `echo test > /etc/shadow` — blocked. `echo test > ./output.txt` — allowed. Symlink to outside — blocked. `/dev/null` — allowed.                |
| AC-6 | Compatibility with common commands                     | Integration test: `grep -r pattern .`, `find . -name '*.go'`, `cat file.txt`, `python3 -c "print(1)"`, `wc -l file`, `jq .field file.json` — all succeed. |
| AC-7 | Config migration warning                               | Unit test: load config with `enable_deny_patterns: false` — verify logged warning, no startup failure.                                                    |
| AC-8 | Cron uses same system                                  | Unit test: cron-created `ExecTool` blocks `sudo ls` identically to agent-created one.                                                                     |

### Rollback plan

- The change is contained within `pkg/tools/` and `pkg/config/`. Git revert of the implementation commits restores the regex-based system.
- Assumption: no other packages depend on `ExecTool` internals (only the `Tool` interface is public contract). Verified: only `pkg/agent/instance.go` and `pkg/tools/cron.go` call `NewExecToolWithConfig`.
- If the interpreter causes widespread command failures post-deploy, revert and reconsider the hybrid approach (Alternative #4).

### Decision review date

**2026-06-04** — Review:

- False-positive rate (commands incorrectly blocked).
- False-negative rate (dangerous commands that slipped through).
- Interpreter compatibility issues reported.
- Whether `OpenHandler` limitations warrant OS-level sandboxing investment.

---

## Assumptions

- **Assumption**: >95% of LLM-generated shell commands are compatible with `mvdan.cc/sh/v3/interp` in LangBash mode. Based on: LLMs generate simple pipelines, not job-control or advanced bash internals.
- **Assumption**: The `ExecHandlers` middleware sees the fully-resolved command (after variable expansion, globbing, etc.) before execution. Based on: `mvdan.cc/sh/v3/interp` documentation for `ExecHandlerFunc`.
- **Assumption**: No downstream consumers depend on the `ExecConfig` fields being removed. Based on: config is JSON-deserialized; unknown fields are ignored by default in Go's `encoding/json`.
- **Assumption**: `mvdan.cc/sh/v3` will remain maintained for the foreseeable future. Based on: BSD-3-Clause, powers `shfmt`, 244+ importers, latest release v3.12.0 (Jul 2025).
- **Assumption**: `mvdan.cc/sh/v3` adds ~450 KB to compiled binary size. Based on: typical Go module size for parser+interpreter; to be verified with `go build` size comparison before/after.
- **Assumption**: Users accept that executed external programs have full filesystem access. This is inherent to exec-based execution and is not a regression — the current `sh -c` system has the same property. The new system makes this limitation _explicit_ rather than hiding it behind regex theater.
