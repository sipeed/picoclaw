# Workspace Temp Directory

Picoclaw workspaces have a standard scratch directory:

```text
<workspace>/tmp
```

Agents should put temporary scripts, one-off drafts, intermediate JSON payloads,
generated helper files, and other disposable artifacts under this directory
instead of creating `tmp_*` files in the workspace root.

## Why

The workspace root contains durable agent state and operator-facing files such as
`AGENT.md`, `USER.md`, `SOUL.md`, `TODO.md`, `memory/`, `skills/`, and
`state/`. Root-level scratch files make the workspace hard to inspect and can be
mistaken for persistent state.

Using a single temp directory keeps transient files isolated while still making
them available to tools that are restricted to the workspace.

## Runtime Contract

The standard temp path is implemented by `pkg/workspace.TempDir`.

When the `exec` tool is configured with a workspace, Picoclaw creates the temp
directory and exposes it to subprocesses as:

```sh
PICOCLAW_WORKSPACE_TMP=<workspace>/tmp
```

Shell commands should use that variable for scratch work:

```sh
cat > "$PICOCLAW_WORKSPACE_TMP/check_payload.py" <<'PY'
print("temporary helper")
PY
python3 "$PICOCLAW_WORKSPACE_TMP/check_payload.py"
```

File tools should use paths under `tmp/` for temporary files:

```text
tmp/intermediate-result.json
tmp/generated-report.md
```

## Persistent Outputs

Do not use `tmp/` for files that are part of the user's durable workspace state.
Persistent outputs should go to their normal domain-specific locations, for
example:

- `memory/` for memory and daily notes.
- `skills/<skill-name>/` for reusable skill instructions.
- `state/` for durable runtime state.
- A user-requested path when the user explicitly asks for a specific file.

## Cleanup

`tmp/` is a scratch area. Operators may delete old files from it when they are no
longer needed. Code that creates large temporary artifacts should prefer
task-specific names or subdirectories so cleanup is straightforward.
