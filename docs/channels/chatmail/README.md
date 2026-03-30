> Back to [README](../../../README.md)

# Chatmail (Delta Chat)

The Chatmail channel enables PicoClaw to communicate via [Delta Chat](https://delta.chat/), a decentralized messaging platform based on email. It uses the [chatmail/rpc-client-go](https://github.com/chatmail/rpc-client-go) library to interact with Delta Chat through its RPC interface.

## Prerequisites

- **deltachat-rpc-server**: The Delta Chat RPC server binary must be installed and available in your system's PATH.
- **Chatmail account or compatible chatemail account**: Check the chatmail relays [https://chatmail.at/relays](https://chatmail.at/relays) or compatibility list [https://providers.delta.chat/](https://providers.delta.chat/).

### Installing deltachat-rpc-server

#### Cargo

```bash
# Server
cargo install --git https://github.com/chatmail/core/ deltachat-rpc-server
# Repl for configure account
cargo install --git https://github.com/chatmail/core/ deltachat-repl
```

#### From Source

```bash
# Server
git clone https://github.com/chatmail/core.git
cd deltachat-rpc-server
cargo build --release
sudo cp target/release/deltachat-rpc-server /usr/local/bin/
# Repl for configure account
cd ../deltachat-repl
cargo build --release
sudo cp target/release/deltachat-repl /usr/local/bin/
```

## Configuration

Add this to your `config.json`:

```json
{
  "channels": {
    "chatmail": {
      "enabled": true,
      "account_path": "",
      "allow_from": [],
      "group_trigger": {
        "mention_only": false,
        "prefixes": []
      },
      "reasoning_channel_id": ""
    }
  }
}
```

## Field Reference

| Field                | Type     | Required | Description                                                                                          |
|----------------------|----------|----------|------------------------------------------------------------------------------------------------------|
| enabled              | bool     | Yes      | Whether to enable the Chatmail channel                                                               |
| account_path         | string   | No       | Path to store the Delta Chat account database. Default: `~/.accounts/chatmail`                      |
| allow_from           | []string | No       | Allowlist of contact IDs; empty means all contacts are allowed                                       |
| group_trigger        | object   | No       | Group trigger strategy (`mention_only` / `prefixes`)                                                 |
| reasoning_channel_id | string   | No       | Target channel ID for reasoning output                                                               |

### Group Trigger Configuration

| Field        | Type     | Description                                                                                    |
|--------------|----------|------------------------------------------------------------------------------------------------|
| mention_only | bool     | When `true`, the bot only responds when mentioned in group chats                               |
| prefixes     | []string | List of prefixes that trigger bot responses in groups (e.g., `["!", "/"]`)                     |

## Setup

### Step 1: Enable the Channel

Set `enabled: true` in the configuration file.

### Step 2: Start PicoClaw

When you start PicoClaw with the Chatmail channel enabled, you will see an invite link in the console:

```
Chatmail invite link: https://i.delta.chat/#B2AE34...
Scan this QR code with your Delta Chat app to start chatting.
```

Open the url and scan the QR code.

### Step 3: Configure Your Bot Account

1. **First Run**: On the first startup, the channel creates a new Delta Chat account.
2. **Scan the QR Code**: Use your Delta Chat app to scan the invite link QR code or manually enter the invite code.
3. **Start Chatting**: Once connected, you can send messages to the bot from your Delta Chat app.

### Step 4: Send Messages

- **Direct Messages**: Send any message directly to the bot's chat.
- **Group Chats**: Add the bot to a group chat. Configure `group_trigger` if you want the bot to only respond when mentioned.

## Account Storage

The Delta Chat account is stored at the path specified by `account_path` (default: `~/.accounts/chatmail`). This includes:

- Account database
- Encryption keys
- Message cache

**Important**: Keep this directory secure as it contains your private keys.

## Behavior

### Startup Behavior

When the channel starts:

1. Creates or loads the Delta Chat account
2. Generates and displays an invite link for pairing
3. **Ignores all pending messages** - only processes new messages received after startup
4. Starts listening for incoming messages

### Message Handling

- **Direct messages**: All messages are processed
- **Group messages**: Controlled by `group_trigger` configuration
- **Bot replies**: Sent as regular chat messages with full markdown support

## Supported Features

| Feature              | Status | Notes                                                    |
|----------------------|--------|----------------------------------------------------------|
| Text messages        | ✅      | Send and receive                                         |
| Direct messages      | ✅      | Full support                                             |
| Group chats          | ✅      | With trigger configuration                               |
| Markdown rendering  | ✅      | Messages formatted with markdown (only for ArcaneChat client) |
| Reactions            | ✅      | 👀 reaction on incoming messages, removed after response  |
| Media attachments   | ❌      | Not yet integrated                                         |
| Typing indicators   | ❌      | Not yet integrated                                         |
| Message editing      | ❌      | Not yet integrated                                        |

## Security Considerations

1. **End-to-End Encryption**: Delta Chat uses Autocrypt for automatic E2E encryption. Messages are encrypted by default.
2. **Account Keys**: Private keys are stored in `account_path`. Protect this directory.
3. **Allowlist**: Use `allow_from` to restrict which contacts can interact with your bot.

## Troubleshooting

### "deltachat-rpc-server not found"

Ensure `deltachat-rpc-server` is installed and in your PATH:

```bash
which deltachat-rpc-server
```

### Bot not receiving messages

1. Verify the account is properly configured by checking the logs
2. Ensure you've scanned the QR code or entered the invite code
3. Check that the contact sending messages is in the `allow_from` list (if configured)

### Invite link not appearing

If no invite link appears in the logs:

1. Check that the channel is enabled in the configuration
2. Verify write permissions for `account_path`
3. Check for errors in the PicoClaw logs

## Example Configuration with Group Trigger

```json
{
  "channels": {
    "chatmail": {
      "enabled": true,
      "account_path": "/var/lib/picoclaw/chatmail",
      "allow_from": [],
      "group_trigger": {
        "mention_only": true,
        "prefixes": ["!", "/"]
      }
    }
  }
}
```

With this configuration, the bot will only respond in group chats when:
- The bot is directly mentioned (`@botname your message`), OR
- The message starts with one of the configured prefixes (`!command` or `/command`)

## Multiple Accounts

Each PicoClaw instance can have one Chatmail channel configured. To run multiple bot accounts, use separate configuration files with different `account_path` values.
