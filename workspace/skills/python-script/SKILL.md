---
name: python-script
description: Run an existing local Python script through the exec tool and return the result clearly.
metadata: {"nanobot":{"emoji":"🐍","requires":{"bins":["python3"]}}}
---

# Python Script

Use this skill when the user wants PicoClaw to run an existing Python script in the workspace or another explicitly allowed path.

## Rules

- Prefer `python3`, not `python`.
- Treat the script path and arguments as required inputs. If they are missing, ask for them.
- If the script is unfamiliar, inspect it with `read_file` before executing it.
- Prefer machine-readable output. If the script supports `--json`, use it.
- Do not invent results. Report stdout, stderr, exit code, and any missing dependency errors exactly.
- If the script writes files or updates state, tell the user what changed and where.

## Command Patterns

Run a script:
```bash
python3 path/to/script.py
```

Run with arguments:
```bash
python3 path/to/script.py --input data.json --limit 10
```

Request JSON output when supported:
```bash
python3 path/to/script.py --json
```

Pass temporary environment variables inline:
```bash
FOO=bar python3 path/to/script.py --json
```

## Response Pattern

1. Restate the script path and arguments you will run.
2. Use the `exec` tool to execute the command.
3. Summarize stdout briefly.
4. If stderr is non-empty or the exit code is non-zero, surface that clearly.
5. If stdout is JSON, extract the important fields instead of dumping raw output unless the user asks for the full payload.
