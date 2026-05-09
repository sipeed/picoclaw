# Tool Permission Fix Plan

## Goal
Fix PicoClaw's tool permission system so that when restricted tools are called, instead of denying access, it prompts the user for permission (like OpenClaw does).

## Tasks

### Wave 1: Core Permission Infrastructure (Independent)

- [x] 1. Fix permissionCache initialization in instance.go
  - File: `pkg/agent/instance.go`
  - Initialize `permissionCache` when `cfg.Tools.Exec.AskPermission` is true
  - Register RequestPermissionTool unconditionally when cache is initialized

- [x] 2. Create permission_prompt.go service
  - File: `pkg/tools/permission_prompt.go` (new)
  - Lightweight permission prompt supporting both terminal and chat channels
  - Use bufio.NewReader for terminal (no extra deps)
  - Detect channel capability for chat prompts

- [x] 3. Extend permission.go with "once" and "always" options
  - File: `pkg/tools/permission.go`
  - Add GrantFromUser(path, duration) method
  - Duration options: "once" (one-time), "always" (session-long)

### Wave 2: Extend Permission to Editing Tools (Dependent on Wave 1)

- [ ] 4. Add permission check to list_dir.go
  - File: `pkg/tools/list_dir.go`
  - Add PermissionCache field
  - Check if path outside workspace, request permission if needed

- [ ] 5. Add permission check to write_file.go
  - File: `pkg/tools/write_file.go`
  - Add PermissionCache field
  - Check if path outside workspace, request permission if needed

- [ ] 6. Add permission check to edit_file.go
  - File: `pkg/tools/edit_file.go`
  - Add PermissionCache field
  - Check if path outside workspace, request permission if needed

- [ ] 7. Add permission check to append_file.go
  - File: `pkg/tools/append_file.go`
  - Add PermissionCache field
  - Check if path outside workspace, request permission if needed

### Wave 3: Hook Fix

- [ ] 8. Fix exec_approval hook timeout
  - Investigate and disable/remove the broken process hook
  - Or implement proper approval response

## Implementation Notes

- Follow OpenClaw's approach but adapt to PicoClaw's existing code style
- Keep dependencies minimal (no new deps)
- Focus on resource efficiency (small code, low memory)
- Use existing channel interface for chat prompts
- Default timeout: 60 seconds
- Permission options: "once", "always", "deny"