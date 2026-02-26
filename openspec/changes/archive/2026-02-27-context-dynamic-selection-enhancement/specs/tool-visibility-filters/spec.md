## ADDED Requirements

### Requirement: Tool Visibility Filter Registration
系统必须支持在注册工具时设置可见性规则，基于多种条件动态控制工具的可见性。

#### Scenario: Register tool with admin-only filter
- **WHEN** developer calls `registry.RegisterWithFilter(adminTool, func(ctx) bool { return contains(ctx.UserRoles, "admin") })`
- **THEN** the adminTool is only visible to users with "admin" role

#### Scenario: Register tool with channel-specific filter
- **WHEN** developer registers a message tool with filter checking `ctx.Channel == "telegram"`
- **THEN** the message tool is only visible in Telegram channel, not in CLI or other channels

#### Scenario: Register tool without filter (always visible)
- **WHEN** developer calls `registry.Register(readFileTool)` without filter
- **THEN** readFileTool is visible to all channels and users
