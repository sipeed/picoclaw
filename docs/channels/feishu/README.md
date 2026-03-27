> Back to [README](../../../README.md)

# Feishu

Feishu (international name: Lark) is an enterprise collaboration platform by ByteDance. It supports both Chinese and global markets through event-driven WebSocket connections.

## Configuration

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    }
  }
}
```

| Field                 | Type   | Required | Description                                                        |
| --------------------- | ------ | -------- | ------------------------------------------------------------------ |
| enabled               | bool   | Yes      | Whether to enable the Feishu channel                               |
| app_id                | string | Yes      | App ID of the Feishu application (starts with `cli_`)              |
| app_secret            | string | Yes      | App Secret of the Feishu application                               |
| encrypt_key           | string | No       | Encryption key for event callbacks                                 |
| verification_token    | string | No       | Token used for Webhook event verification                          |
| allow_from            | array  | No       | Allowlist of user IDs; empty means all users are allowed           |
| random_reaction_emoji | array  | No       | List of random reaction emojis; empty uses the default "Pin"       |
| download_dir          | string | No       | Custom directory for downloaded files (relative/absolute paths)    |

## Setup

1. Go to the [Feishu Open Platform](https://open.feishu.cn/) and create an application
2. Enable the **Bot** capability in the application settings
3. Create a version and publish the application (configuration takes effect only after publishing)
4. Obtain the **App ID** (starts with `cli_`) and **App Secret**
5. Fill in the App ID and App Secret in the PicoClaw configuration file
6. Run `picoclaw gateway` to start the service
7. Search for the bot name in Feishu and start a conversation

> PicoClaw connects to Feishu using WebSocket/SDK mode — no public callback address or Webhook URL is required.
>
> `encrypt_key` and `verification_token` are optional; enabling event encryption is recommended for production environments.
>
> For custom emoji references, see: [Feishu Emoji List](https://open.larkoffice.com/document/server-docs/im-v1/message-reaction/emojis-introduce)

## Platform Limitations

> ⚠️ **Feishu channel does not support 32-bit devices.** The Feishu SDK only provides 64-bit builds. Devices running armv6, armv7, mipsle, or other 32-bit architectures cannot use the Feishu channel. For messaging on 32-bit devices, use Telegram, Discord, or OneBot instead.

## Download Directory Configuration

The `download_dir` field specifies where files received from Feishu (images, documents, etc.) are stored.

### Path Resolution

1. **Relative path** (recommended): Resolved relative to the PicoClaw workspace directory
   - Default workspace: `~/.picoclaw/workspace` (or `$PICOCLAW_HOME/workspace`)
   - Example: If workspace is `~/.picoclaw/workspace`, `download_dir: "downloads"` resolves to `~/.picoclaw/workspace/downloads`
   ```json
   {
     "channels": {
       "feishu": {
         "download_dir": "downloads"
       }
     }
   }
   ```

2. **Absolute path**: Use an absolute path directly
   ```json
   {
     "channels": {
       "feishu": {
         "download_dir": "/Users/username/Downloads/feishu"
       }
     }
   }
   ```

3. **Home directory shorthand**: Use `~` for your home directory
   ```json
   {
     "channels": {
       "feishu": {
         "download_dir": "~/Downloads/feishu"
       }
     }
   }
   ```

4. **Default (not configured)**: Files are downloaded to the system temp directory
   - macOS: `/var/folders/.../picoclaw_media`
   - Linux: `/tmp/picoclaw_media`
   - Windows: `C:\Users\xxx\AppData\Local\Temp\picoclaw_media`

### Safety & Fallback

- The directory will be created automatically if it doesn't exist
- Write permissions are verified before use
- If the configured directory cannot be accessed (no permission, path conflicts, etc.), the system automatically falls back to the system temp directory to ensure files are received successfully

### Environment Variable

You can also configure via environment variable:

```bash
export PICOCLAW_CHANNELS_FEISHU_DOWNLOAD_DIR="downloads"
```
