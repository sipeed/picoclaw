# File System Tools

File system tools allow the agent to read, write, and manage files.

## Tools

### read_file

Read the contents of a file.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | string | Yes | File path (relative to workspace) |

**Example:**

```json
{
  "path": "notes.txt"
}
```

### write_file

Create or overwrite a file with content.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | string | Yes | File path (relative to workspace) |
| `content` | string | Yes | File content |

**Example:**

```json
{
  "path": "output.txt",
  "content": "Hello, World!"
}
```

### list_dir

List contents of a directory.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | string | Yes | Directory path (relative to workspace) |

**Example:**

```json
{
  "path": "."
}
```

### edit_file

Replace text in a file.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | string | Yes | File path (relative to workspace) |
| `old_text` | string | Yes | Text to replace |
| `new_text` | string | Yes | Replacement text |

**Example:**

```json
{
  "path": "config.json",
  "old_text": "\"enabled\": false",
  "new_text": "\"enabled\": true"
}
```

### append_file

Append content to a file.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `path` | string | Yes | File path (relative to workspace) |
| `content` | string | Yes | Content to append |

**Example:**

```json
{
  "path": "log.txt",
  "content": "\nNew log entry"
}
```

## Security

### Workspace Restriction

When `restrict_to_workspace: true`:

- All paths are relative to workspace
- Absolute paths outside workspace are blocked
- Symlink escapes are prevented

### Path Resolution

Paths are resolved relative to the workspace:

| Input | Resolved To |
|-------|-------------|
| `file.txt` | `~/workspace/file.txt` |
| `subdir/file.txt` | `~/workspace/subdir/file.txt` |
| `/etc/passwd` | **BLOCKED** (outside workspace) |

## Usage Examples

```
User: "Read the README.md file"

Agent uses read_file:
{
  "path": "README.md"
}

Agent: "The README.md file contains..."
```

```
User: "Create a todo list file"

Agent uses write_file:
{
  "path": "todo.md",
  "content": "# Todo List\n\n- [ ] Task 1\n- [ ] Task 2"
}

Agent: "Created todo.md with your todo list."
```

## Disabling Restriction

For trusted environments:

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```

**Warning**: This allows access to any file the user can access.

## See Also

- [Tools Overview](README.md)
- [Security Sandbox](../advanced/security-sandbox.md)
