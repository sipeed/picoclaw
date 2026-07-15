> 回到 [README](../../project/README.zh-tw.md)

# Slack

Slack 是主流的企業即時通訊平台。PicoClaw 使用 Slack 的 Socket Mode 進行即時雙向通訊，不需要設定公開的 Webhook 端點。

## 設定

```json
{
  "channel_list": {
    "slack": {
      "enabled": true,
      "type": "slack",
      "bot_token": "xoxb-...",
      "app_token": "xapp-...",
      "allow_from": []
    }
  }
}
```

| 欄位       | 型別   | 必填 | 說明                                                    |
| ---------- | ------ | ---- | ------------------------------------------------------- |
| enabled    | bool   | 是   | 是否啟用 Slack 頻道                                     |
| bot_token  | string | 是   | Slack Bot 的 Bot User OAuth Token（以 xoxb- 開頭）      |
| app_token  | string | 是   | Slack 應用程式的 Socket Mode App Level Token（以 xapp- 開頭） |
| allow_from | array  | 否   | 使用者 ID 允許清單；留空表示允許所有使用者              |

## 設定步驟

1. 前往 [Slack API](https://api.slack.com/) 並建立新的 Slack 應用程式
2. 啟用 Socket Mode 並取得 App Level Token
3. 加入 Bot Token Scopes（例如 `chat:write`、`im:history` 等）
4. 將應用程式安裝到工作區，並取得 Bot User OAuth Token
5. 在設定檔填入 Bot Token 與 App Token
