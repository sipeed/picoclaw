# Tools

Custom tool descriptions and usage guidelines for the AI assistant.

## Examples

- `search_web`: Use for real-time information, news, weather, and current events.
- `read_file`: Read files from the workspace. Prefer this over shell commands for file content.
- `write_file`: Create or overwrite files. Use `edit_file` for partial modifications.
- `shell`: Execute shell commands. Always prefer specific tools over shell when available.

## Guidelines

- Prefer specific tools over generic ones (e.g., use `search_web` instead of `shell` with curl).
- Always confirm before executing destructive operations (delete, overwrite).
- Use the `spawn` tool for long-running or independent subtasks.

---

Add your custom tool descriptions below this line:
