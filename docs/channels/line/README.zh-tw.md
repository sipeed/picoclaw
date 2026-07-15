> 回到 [README](../../project/README.zh-tw.md)

# LINE

PicoClaw 使用 LINE Messaging API 及 Webhook 回呼來提供 LINE 頻道功能。

## 設定

```json
{
  "channel_list": {
    "line": {
      "enabled": true,
      "type": "line",
      "channel_secret": "YOUR_CHANNEL_SECRET",
      "channel_access_token": "YOUR_CHANNEL_ACCESS_TOKEN",
      "webhook_path": "/webhook/line",
      "allow_from": []
    }
  }
}
```

| 欄位                 | 型別   | 必填 | 說明                                           |
| -------------------- | ------ | ---- | ---------------------------------------------- |
| enabled              | bool   | 是   | 是否啟用 LINE 頻道                             |
| channel_secret       | string | 是   | LINE Messaging API 的 Channel Secret           |
| channel_access_token | string | 是   | LINE Messaging API 的 Channel Access Token     |
| webhook_path         | string | 否   | Webhook 路徑（預設：/webhook/line）            |
| allow_from           | array  | 否   | 使用者 ID 允許清單；留空表示允許所有使用者     |

## 設定步驟

1. 前往 [LINE Developers Console](https://developers.line.biz/console/)，建立 Provider 及 Messaging API 頻道
2. 取得 Channel Secret 與 Channel Access Token
3. 設定 Webhook：
   - LINE 要求 Webhook 使用 HTTPS，因此必須部署支援 HTTPS 的伺服器，或使用 ngrok 等反向 Proxy 工具，讓外部網路能連線至本機伺服器
   - PicoClaw 使用共用的閘道 HTTP 伺服器接收所有頻道的 Webhook 回呼，預設監聽 `127.0.0.1:18790`
   - 將 Webhook URL 設為 `https://your-domain.com/webhook/line`，再以反向 Proxy 將外部網域轉送至本機閘道（預設連接埠為 18790）
   - 啟用 Webhook 並驗證 URL
4. 在設定檔填入 Channel Secret 與 Channel Access Token
