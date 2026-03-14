# Configuration Reference

`config/config.example.json` is the copy-paste template. This document is the field guide for the parts that are easy to miss when you only skim `README.md`.

## Top-Level Layout

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `agents` | object | required | Agent defaults plus optional `agents.list` overrides |
| `bindings` | array | `[]` | Route specific channels/peers/accounts to specific agent IDs |
| `session` | object | `{"dm_scope":"per-channel-peer"}` | Controls DM session isolation and cross-platform identity linking |
| `model_list` | array | built-in starter list | Preferred provider/model configuration |
| `channels` | object | channel-specific defaults | Each channel stays disabled until `enabled: true` |
| `tools` | object | see [Tools Configuration](tools_configuration.md) | Global tool availability and limits |
| `heartbeat` | object | `{"enabled":true,"interval":30}` | Periodic task loop |
| `devices` | object | `{"enabled":false,"monitor_usb":true}` | USB/device integration |
| `voice` | object | `{"echo_transcription":false}` | Voice UX tweaks |
| `gateway` | object | `{"host":"127.0.0.1","port":18790}` | Shared HTTP server for webhook channels |

## Agent Defaults

`agents.defaults` sets the baseline for every agent instance. Entries in `agents.list` override these fields per agent.

| Field | Type | Default | Notes |
| --- | --- | --- | --- |
| `workspace` | string | `~/.picoclaw/workspace` | Base workspace for the implicit main agent |
| `restrict_to_workspace` | bool | `true` | Restricts file and exec tools to the workspace boundary |
| `allow_read_outside_workspace` | bool | `false` | Lets read-only tools escape the workspace while writes stay restricted |
| `model_name` | string | empty | Preferred alias from `model_list` |
| `model` | string | empty | Deprecated alias for `model_name`; still accepted |
| `model_fallbacks` | array | `[]` | Fallback aliases from `model_list` |
| `image_model` | string | empty | Separate model alias for image tasks |
| `image_model_fallbacks` | array | `[]` | Image-model fallback aliases |
| `max_tokens` | int | `32768` | Agent response budget |
| `temperature` | number or null | provider default | `null` means do not override provider defaults |
| `max_tool_iterations` | int | `50` | Upper bound on tool loop turns |
| `summarize_message_threshold` | int | `20` | Start compressing history after this many messages |
| `summarize_token_percent` | int | `75` | Compression target relative to current context |
| `max_media_size` | int | `20971520` | Per-attachment cap in bytes (20 MB) |
| `routing` | object | disabled | Lightweight model-routing classifier for cheap/simple turns |

### Routing

When `agents.defaults.routing.enabled` is `true`, PicoClaw scores each incoming message. Messages below `threshold` are sent to `light_model`; everything else stays on the agent's primary model.

| Field | Type | Default | Notes |
| --- | --- | --- | --- |
| `enabled` | bool | `false` | Turns routing on/off |
| `light_model` | string | empty | Alias from `model_list` used for simpler turns |
| `threshold` | number | `0` | Score in `[0,1]`; lower scores are considered simpler |

## Multi-Agent Setup

`agents.list` is an array, not a map. Each entry creates a concrete agent instance.

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `id` | string | yes | Stable agent ID used by bindings and subagent permissions |
| `default` | bool | no | Marks the default agent; otherwise the first list entry wins |
| `name` | string | no | Human-readable label |
| `workspace` | string | no | Per-agent workspace override |
| `model` | string or object | no | Either `"gpt-5.4"` or `{"primary":"gpt-5.4","fallbacks":["claude-sonnet-4.6"]}` |
| `skills` | array | no | Limits which `SKILL.md` entries are exposed to this agent |
| `subagents` | object | no | Controls which agents this agent may spawn |

### Subagent Rules

| Field | Type | Notes |
| --- | --- | --- |
| `allow_agents` | array | Agent IDs this agent may spawn, or `"*"` to allow every configured agent |
| `model` | string or object | Override the model/fallback set used when this agent spawns a subagent |

### Example

```json
{
  "agents": {
    "defaults": {
      "model_name": "gpt-5.4",
      "routing": {
        "enabled": true,
        "light_model": "deepseek",
        "threshold": 0.35
      }
    },
    "list": [
      {
        "id": "main",
        "default": true,
        "model": "gpt-5.4",
        "subagents": {
          "allow_agents": ["coder"]
        }
      },
      {
        "id": "coder",
        "workspace": "~/.picoclaw/workspace/code",
        "model": {
          "primary": "claude-sonnet-4.6",
          "fallbacks": ["gpt-5.4"]
        },
        "skills": ["Code", "git-essentials"]
      }
    ]
  },
  "bindings": [
    {
      "agent_id": "coder",
      "match": {
        "channel": "telegram",
        "account_id": "*",
        "peer": {
          "kind": "direct",
          "id": "123456789"
        }
      }
    }
  ]
}
```

`skills` only filters visible skills. Tool permissions still come from the global `tools.*` config and the workspace sandbox.

## Bindings

Bindings route inbound traffic to an agent before any tool execution starts.

| Field | Type | Notes |
| --- | --- | --- |
| `agent_id` | string | Target agent ID |
| `match.channel` | string | Required channel name such as `telegram`, `discord`, `feishu` |
| `match.account_id` | string | Optional multi-account filter; use `*` for any account |
| `match.peer.kind` | string | Optional peer type: `direct`, `group`, or `channel` |
| `match.peer.id` | string | Optional peer ID |
| `match.guild_id` | string | Optional Discord guild match |
| `match.team_id` | string | Optional Slack team match |

Binding priority is fixed in code:

1. `peer`
2. `parent_peer`
3. `guild_id`
4. `team_id`
5. `account_id`
6. channel-wide wildcard (`account_id: "*"`)
7. default agent

## Session Management

### DM Scope

`session.dm_scope` only affects direct messages. Group and channel peers always keep their own per-peer session keys.

| Value | Behavior |
| --- | --- |
| `main` | Reuse a single shared main session |
| `per-peer` | One DM session per person across all channels |
| `per-channel-peer` | One DM session per channel/person pair |
| `per-account-channel-peer` | One DM session per account + channel + person pair |

### Identity Links

`session.identity_links` collapses multiple platform IDs into one canonical identity when DM scope is not `main`.

```json
{
  "session": {
    "dm_scope": "per-peer",
    "identity_links": {
      "duomi": [
        "telegram:123456789",
        "discord:duomi#1234",
        "feishu:ou_xxx"
      ]
    }
  }
}
```

If a DM arrives from any listed identity, PicoClaw reuses the canonical session key (`duomi` in this example).

## Channels

Channel docs under `docs/channels/` remain the setup guide for tokens and platform-specific steps. The table below focuses on shared runtime fields that were missing from the central docs.

### Shared Channel Fields

| Field | Applies To | Notes |
| --- | --- | --- |
| `allow_from` | most channels | Whitelist of users, groups, or room IDs allowed to talk to the bot |
| `group_trigger.mention_only` | group-capable channels | Reply only when mentioned |
| `group_trigger.prefixes` | prefix-based channels | Reply when a message starts with one of the configured prefixes |
| `typing.enabled` | Telegram, Discord, Slack, LINE, OneBot, IRC | Enables typing indicators where supported |
| `placeholder.enabled` | Telegram, Feishu, Discord, Matrix, LINE, OneBot, Pico, Slack | Send a quick placeholder response before the final answer |
| `placeholder.text` | same as above | Placeholder message text |
| `reasoning_channel_id` | supported channels | Redirect long reasoning traces to a separate channel/room |

### Less-Documented Channel Blocks

| Channel | Field | Default | Notes |
| --- | --- | --- | --- |
| `qq` | `max_message_length` | `2000` | Hard cap before QQ responses are split |
| `qq` | `send_markdown` | `false` | Prefer QQ markdown cards when supported |
| `matrix` | `device_id` | empty | Optional device identifier for login/session reuse |
| `matrix` | `join_on_invite` | `true` | Auto-join rooms when invited |
| `matrix` | `message_format` | empty | Rendering mode override |
| `maixcam` | `host` / `port` | `0.0.0.0` / `18790` | TCP endpoint PicoClaw listens on for MaixCam clients |
| `wecom` / `wecom_app` / `wecom_aibot` | `reply_timeout` | `5` | Timeout in seconds before falling back to async behavior |
| `wecom_aibot` | `max_steps` | `10` | Maximum streaming steps per response |
| `wecom_aibot` | `welcome_message` | built-in greeting | Sent on `enter_chat`; empty disables it |
| `pico` | `allow_token_query` | `false` | Accept auth token in query string instead of headers |
| `pico` | `allow_origins` | `[]` | CORS allow-list for browser clients |
| `pico` | `ping_interval` | `30` | WebSocket ping interval in seconds |
| `pico` | `read_timeout` | `60` | Max read wait in seconds |
| `pico` | `write_timeout` | `10` | Max write wait in seconds |
| `pico` | `max_connections` | `100` | Concurrent connection ceiling |
| `irc` | `request_caps` | `["server-time","message-tags"]` in example | Requested IRC capabilities |
| `irc` | `typing.enabled` | `false` | Whether to emit typing indicators into IRC |

## Tool Limits and Escape Hatches

See [Tools Configuration](tools_configuration.md) for full per-tool examples. These are the fields most likely to affect safety or operational limits:

| Field | Default | Notes |
| --- | --- | --- |
| `tools.allow_read_paths` | `null` | Regex allow-list applied on top of the workspace sandbox for read-only tools |
| `tools.allow_write_paths` | `null` | Regex allow-list for write operations |
| `tools.web.proxy` | empty | Optional proxy for web search/fetch |
| `tools.web.fetch_limit_bytes` | `10485760` | Max bytes fetched by web tools before truncation/refusal |
| `tools.exec.allow_remote` | `true` | Allows exec from remote channels; set `false` to require internal channels only |
| `tools.exec.custom_allow_patterns` | `null` | Regex allow-list evaluated before deny rules |
| `tools.exec.custom_deny_patterns` | `null` | Extra regex deny rules |
| `tools.exec.timeout_seconds` | `60` | Per-command timeout |
| `tools.read_file.max_read_file_size` | `65536` | Max bytes returned by the read-file tool |
| `tools.media_cleanup.max_age_minutes` | `30` | Delete old media files after this many minutes |
| `tools.media_cleanup.interval_minutes` | `5` | Cleanup sweep interval |
| `tools.skills.max_concurrent_searches` | `2` | Limits parallel skill searches/download lookups |
| `tools.skills.search_cache.max_size` | `50` | Skill search cache size |
| `tools.skills.search_cache.ttl_seconds` | `300` | Skill search cache TTL |
| `tools.mcp.discovery.ttl` | `5` | How many turns a discovered MCP tool remains unlocked |

## See Also

- [config/config.example.json](../config/config.example.json)
- [Tools Configuration](tools_configuration.md)
- [Model-list Migration Guide](migration/model-list-migration.md)
