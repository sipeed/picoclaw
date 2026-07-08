> Back to [README](../../../README.md)

# Delta Chat Channel

PicoClaw can run as a Delta Chat bot by launching a local
`deltachat-rpc-server` process and talking to it over JSON-RPC. The RPC server
handles the email account, IMAP/SMTP connection, message store, and encryption
keys.

## Install

Install the Delta Chat RPC server. If `deltachat-rpc-server` is on `PATH`,
PicoClaw can find it automatically; otherwise set `rpc_server_path` to the
exact binary path.

```bash
pip install deltachat-rpc-server
which deltachat-rpc-server
```

Prebuilt binaries are also available from the
[Delta Chat core releases](https://github.com/deltachat/deltachat-core-rust/releases).

## Configure

The easiest setup is to let PicoClaw create a chatmail account in Delta
Chat's local account store. Put a relay marker in `email` using an empty local
part, for example `@nine.testrun.org`:

```json
{
  "channel_list": {
    "deltachat": {
      "enabled": true,
      "type": "deltachat",
      "allow_from": ["friend@example.org"],
      "group_trigger": {
        "mention_only": true
      },
      "settings": {
        "email": "@nine.testrun.org",
        "display_name": "PicoClaw Bot",
        "avatar_image": "/home/me/bot-avatar.png"
      }
    }
  }
}
```

On startup, PicoClaw creates the account through `deltachat-rpc-server`, then
stops with an error that contains the generated address. Replace the relay
marker with that full email address and run PicoClaw again:

```json
{
  "email": "bot123@nine.testrun.org",
  "display_name": "PicoClaw Bot",
  "avatar_image": "/home/me/bot-avatar.png"
}
```

If `email` is missing, the startup error lists the built-in relay choices copied
from Parla. You can use one of those relay markers, or a custom chatmail relay
with the same `@server.name` form.

`password` is not needed for PicoClaw-created chatmail accounts. Omit it when
`email` points to an already configured account in `data_dir`; the JSON-RPC
server owns the mailbox password. The legacy password-based path remains only
for classic email accounts that PicoClaw must configure itself. In that mode,
`password` is a secure field; on first config load it is moved to
`~/.picoclaw/.security.yml`, and it can also be set with
`PICOCLAW_CHANNELS_DELTACHAT_PASSWORD`.

`display_name` and `avatar_image` are optional profile settings. When present,
PicoClaw applies them on every startup, so changing the avatar path in config is
enough to update the bot profile.

| Field | Required | Description |
|-------|----------|-------------|
| `email` | Yes | Full bot mailbox address, or first-run relay marker such as `@nine.testrun.org` |
| `rpc_server_path` | No | Path to `deltachat-rpc-server`; only needed when it is not on `PATH` |
| `password` | No | Legacy only; required when PicoClaw must configure/reconfigure a classic mailbox itself |
| `display_name` | No | Startup-applied profile name shown to contacts and used for group mention detection |
| `avatar_image` | No | Startup-applied profile avatar image path; `~` is expanded. Missing files are warned and ignored |
| `data_dir` | No | Account database directory. Default: `~/.picoclaw/deltachat/<channel-name>` |
| `invite_link` | No | Delta Chat invite link to join on startup |
| `allow_crosspost` | No | Default `false`. When `true`, senders allowed by `allow_from` may use `message` tool targets outside the current chat, or resolve recipients by email/contact/chat name |
| `imap_server`, `imap_port` | No | Manual IMAP override for password-based configuration |
| `smtp_server`, `smtp_port` | No | Manual SMTP override for password-based configuration |

Standard channel fields such as `allow_from`, `group_trigger`, and
`reasoning_channel_id` also apply.

## First Run

With `email` set to `@server`, PicoClaw creates the chatmail account, prints
the generated full email in the startup error, and exits. Update `email` to that
full address and run PicoClaw again. On later runs, PicoClaw selects the
configured account by `email`, applies optional profile settings, marks it as a
bot, and starts IO.

With a new `data_dir` plus legacy `password`, PicoClaw can still configure a
classic email account and validate the mailbox credentials; after that, the
account is reused from the local data directory.

Delta Chat requires peers to learn the bot's encryption key before messaging
it. On startup PicoClaw prints the bot invite link and QR code. Add the bot from
Delta Chat with that invite, not by typing the bare email address.

## Behavior

- Direct chats always respond after `allow_from` passes.
- Group chats follow `group_trigger`; without one, every group message is
  handled.
- Messages from the bot itself, device chats, and info/system messages are
  ignored.
- Accepted inbound messages are marked seen after the allow-list check.
- Incoming attachments (images, audio, video, documents) are registered with
  the media store and handed to the agent, so it can view images or operate on
  the files directly. If no media store is available, the path is appended
  inline as `[attachment: /path]` instead.
- Outbound attachments are supported: when the agent emits media, each file is
  sent as a Delta Chat message (with the caption as text). Delta Chat infers the
  view type from the file, so images, GIFs, and videos render natively.
- Crosspost recipient lookup is disabled by default. The agent can always reply
  to the current numeric chat ID, but sending to another numeric chat ID or
  resolving an email/contact/chat name requires `allow_crosspost: true`
  and the current sender must pass `allow_from`; `allow_from: ["*"]` allows this
  for any sender.
- Voice is supported in both directions when voice providers are configured:
  incoming voice notes are transcribed by the agent's ASR and the transcript is
  passed to the model; the agent can reply with synthesized speech, which is
  delivered as a native Delta Chat voice message (`send_tts`). This requires an
  ASR and/or TTS provider under `voice` — it is not Delta Chat-specific config.

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `deltachat-rpc-server not found on PATH` or `rpc_server_path ... not found` | Install the RPC server on PATH, or set `rpc_server_path` to an absolute path |
| `email is required` | Choose one listed chatmail server, set `email` to a first-run marker such as `@nine.testrun.org`, run `picoclaw g`, then replace it with the generated full email |
| `created chatmail account ...` | Replace the `@server` marker in `email` with the generated full email and run PicoClaw again |
| `account ... is not configured in data_dir` | Point `data_dir` at the existing JSON-RPC account store, or use `email="@server"` to create one |
| `configure (check email/password/server)` | Check credentials, app password requirements, or IMAP/SMTP overrides |
| Bot does not answer in a group | Check `group_trigger`; mention `display_name` or use a configured prefix |
| Bot ignores a sender | Add the sender email to `allow_from`, or use `["*"]` for open access |
| Sender cannot message the bot | Re-add the bot with the startup QR/invite so Delta Chat can establish encryption |
| Agent cannot send to an email/name/other chat ID | Enable `settings.allow_crosspost` and allow the controlling sender in `allow_from`; this capability is disabled by default for privacy |
