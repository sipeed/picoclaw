# MCP Server CLI

> Back to [README](../README.md)

PicoClaw includes an `mcp` CLI command group for managing MCP server entries in `config.json`.

This CLI acts as a **configuration manager**:

- it adds, updates, removes, and validates entries under `tools.mcp.servers`
- it does **not** keep MCP servers running itself
- the gateway / host still starts the configured servers when MCP is enabled

## Where It Writes

The CLI updates the same config file used by the rest of PicoClaw:

- `PICOCLAW_CONFIG` if set
- otherwise `~/.picoclaw/config.json`

When the CLI writes the file, it:

- saves atomically
- preserves the standard 2-space JSON formatting used by PicoClaw
- validates the generated JSON before writing

Behavior notes:

- `picoclaw mcp add ...` enables `tools.mcp.enabled`
- removing the last server with `picoclaw mcp remove ...` disables `tools.mcp.enabled`

## Quick Start

Add a stdio server via `npx`:

```bash
picoclaw mcp add filesystem -- npx -y @modelcontextprotocol/server-filesystem /tmp
```

Add a stdio server with environment variables saved in config:

```bash
picoclaw mcp add github --env GITHUB_PERSONAL_ACCESS_TOKEN=ghp_xxx -- npx -y @modelcontextprotocol/server-github
```

Add a stdio server using an env file for secrets:

```bash
picoclaw mcp add github --env-file .env.github -- npx -y @modelcontextprotocol/server-github
```

Add a remote HTTP server:

```bash
picoclaw mcp add context7 --transport http https://mcp.context7.com/mcp
```

Add a remote HTTP server with auth header, even with flags after the URL:

```bash
picoclaw mcp add apify "https://mcp.apify.com/" -t http --header "Authorization: Bearer OMITTED"
```

Add a stdio server using an explicit command separator:

```bash
picoclaw mcp add --transport stdio --env AIRTABLE_API_KEY=YOUR_KEY airtable -- npx -y airtable-mcp-server
```

Inspect the configured entries:

```bash
picoclaw mcp list
picoclaw mcp list --status
```

Inspect one server's full details and its exposed tools:

```bash
picoclaw mcp show filesystem
```

Probe a single server entry:

```bash
picoclaw mcp test filesystem
```

Open the raw config for advanced editing:

```bash
picoclaw mcp edit
```

## Command Summary

| Command | Purpose |
|---------|---------|
| `picoclaw mcp add <name> [flags] <command-or-url> [args...]` | Add or update an MCP server entry |
| `picoclaw mcp remove <name>` | Remove a server entry from config |
| `picoclaw mcp list` | List configured MCP servers |
| `picoclaw mcp show <name>` | Show full details and tools for one server |
| `picoclaw mcp test <name>` | Try connecting to one configured server |
| `picoclaw mcp edit` | Open `config.json` in `$EDITOR` |

## `picoclaw mcp add`

Syntax:

```bash
picoclaw mcp add <name> [flags] <command-or-url> [args...]
```

Supported flags:

| Flag | Meaning |
|------|---------|
| `--env`, `-e` | Add a stdio environment variable in `KEY=value` format. Repeatable. Values are saved to config. |
| `--env-file` | Attach an env file path to a stdio server. Recommended for secrets you do not want stored inline in `config.json`. |
| `--header`, `-H` | Add an HTTP header in `Name: Value` or `Name=Value` format. Repeatable. |
| `--transport`, `-t` | Transport type: `stdio` (default), `http`, or `sse`. |
| `--force`, `-f` | Overwrite an existing server entry without confirmation. |
| `--deferred` | Mark the server as deferred: tools are hidden and discoverable on demand. |
| `--no-deferred` | Mark the server as non-deferred: tools are always loaded into context. |

When neither `--deferred` nor `--no-deferred` is passed, the `deferred` field is omitted from the stored config and the global `discovery.enabled` value applies at runtime.

Supported forms:

```bash
picoclaw mcp add [flags] <name> <command-or-url> [args...]
picoclaw mcp add [flags] <name> -- <command> [args...]
```

Parsing behavior:

- CLI flags can appear before the name, between the name and target, or after the URL for remote transports
- for `stdio`, the most robust form is `-- <command> [args...]`
- use the `--` separator when the stdio command itself has arguments that may look like PicoClaw CLI flags
- without `--`, PicoClaw treats the first two non-flag tokens as `<name>` and `<command-or-url>`

Secret handling:

- `--env KEY=value` stores the resolved value directly in `config.json`
- use `--env-file` instead when the value is sensitive and should stay outside the main config file

Example:

```bash
picoclaw mcp add sqlite npx -y @modelcontextprotocol/server-sqlite --db ./mydb.db
```

This stores:

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "sqlite": {
          "enabled": true,
          "type": "stdio",
          "command": "npx",
          "args": ["-y", "@modelcontextprotocol/server-sqlite", "--db", "./mydb.db"]
        }
      }
    }
  }
}
```

Adding the same server with `--deferred` stores the extra field:

```bash
picoclaw mcp add --deferred sqlite npx -y @modelcontextprotocol/server-sqlite --db ./mydb.db
```

```json
{
  "sqlite": {
    "enabled": true,
    "type": "stdio",
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-sqlite", "--db", "./mydb.db"],
    "deferred": true
  }
}
```

### Add Command Rules

For `stdio`:

- `<command-or-url>` is treated as the command
- `[args...]` are stored in `args`
- `--env` is supported
- `--env-file` is supported and stored in `env_file`
- `--header` is rejected
- `-- <command> [args...]` is supported and recommended for unambiguous parsing

For `http` / `sse`:

- `<command-or-url>` must be a valid URL
- extra command args are rejected
- `--env` is rejected
- `--env-file` is rejected
- `--header` is supported and stored in `headers`

Overwrite behavior:

- if `<name>` already exists, PicoClaw asks for confirmation
- use `--force` to skip the prompt

Local path validation:

- if the command looks like a local path such as `./server.py` or `/opt/mcp/server`
- PicoClaw checks that the file exists
- on non-Windows platforms, it also checks that the file is executable

Clear URL/transport error:

- if the target looks like `https://...` but transport is still `stdio`, PicoClaw returns an explicit error telling you to use `--transport http` or `--transport sse`

## `picoclaw mcp remove`

Syntax:

```bash
picoclaw mcp remove <name>
```

This removes the named entry from `tools.mcp.servers`.

If the removed server was the last configured MCP server, PicoClaw also disables `tools.mcp.enabled`.

## `picoclaw mcp list`

Syntax:

```bash
picoclaw mcp list
picoclaw mcp list --status
```

On wide terminals the output is a styled box (same look as `mcp show`). On narrow terminals or when stdout is not a TTY, a plain ASCII table is printed instead.

Output fields:

| Field | Meaning |
|-------|---------|
| `Name` | Server key inside `tools.mcp.servers` |
| `Type` | Effective transport: `stdio`, `http`, or `sse` |
| `Command` / `Target` | Stored command line for stdio servers, or URL for remote servers |
| `Status` | `enabled` / `disabled` by default; with `--status`: `ok (N tools)` or `error` |
| `Deferred` | `deferred` if the per-server override is `true`; `eager` if `false`; omitted if not set |

Notes:

- without `--status`, PicoClaw prints configuration state only
- with `--status`, PicoClaw tries to connect to each enabled server and reports `ok (N tools)` or `error`
- to see the full list of tools a server exposes, use `picoclaw mcp show <name>`

## `picoclaw mcp show`

Syntax:

```bash
picoclaw mcp show <name>
picoclaw mcp show <name> --timeout 15s
```

This connects to the named server and prints:

- server metadata: name, transport type, target, enabled state, deferred override, env var names, env file, header names
- every tool the server exposes, with its name, description, and parameters (name, type, required/optional, description)

On wide terminals the output is a styled box matching the `mcp list` look. On narrow terminals or non-TTY stdout, plain text is printed instead.

Example output (wide terminal):

```
╭──────────────────────────────────────────────────────────╮
│ ⬡  filesystem                                            │
│                                                          │
│ Type        stdio                                        │
│ Target      npx -y @modelcontextprotocol/server-fs /tmp  │
│ Enabled     yes                                          │
│ Deferred    no                                           │
│                                                          │
│ Tools (3)                                                │
│                                                          │
│   read_file  [1/3]                                       │
│   Read the complete contents of a file from the disk     │
│                                                          │
│     path  <string>  required                             │
│       Path to the file to read                           │
│ ──────────────────────────────────────────────────────── │
│   ...                                                    │
╰──────────────────────────────────────────────────────────╯
```

Flags:

| Flag | Default | Meaning |
|------|---------|---------|
| `--timeout` | `10s` | Connection timeout |

Notes:

- if the server is disabled in config, `mcp show` prints the metadata only and skips tool discovery
- `mcp show` always connects live to fetch the tool list; use `mcp test` if you only need a reachability check

## `picoclaw mcp test`

Syntax:

```bash
picoclaw mcp test <name>
```

This performs a direct connection test for one configured entry and prints the number of discovered tools when successful.

It is useful when:

- you want to verify a newly added server before starting the gateway
- you want to debug one server without probing the whole list
- the entry is currently disabled in config but you still want to validate its definition

## `picoclaw mcp edit`

Syntax:

```bash
picoclaw mcp edit
```

This opens the config file in the editor pointed to by `$EDITOR`.

Use it when you need to configure MCP fields that are not exposed directly by `picoclaw mcp add`.

If `$EDITOR` is not set, the command fails with an explicit error.

## Recommended Workflow

For common cases:

1. Add the server with `picoclaw mcp add` (include `--deferred` if you want tools hidden by default).
2. Verify connectivity and inspect the exposed tools with `picoclaw mcp show <name>`.
3. Check all servers at a glance with `picoclaw mcp list --status`.
4. Start PicoClaw normally so the configured MCP server is loaded by the host.

For advanced cases:

1. Add the base entry with `picoclaw mcp add`.
2. Run `picoclaw mcp edit` to fill in fields that are not exposed as CLI flags.
3. Run `picoclaw mcp show <name>` to confirm the final configuration and tool list.

## Related Docs

- [Tools Configuration](tools_configuration.md#mcp-tool): MCP config structure, transports, discovery, and examples
- [README](../README.md): high-level overview
