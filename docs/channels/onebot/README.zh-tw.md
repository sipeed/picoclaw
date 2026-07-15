> 回到 [README](../../project/README.zh-tw.md)

# OneBot

OneBot 是 QQ Bot 的開放通訊協定標準，為各種 QQ Bot 實作（例如 go-cqhttp、Mirai）提供統一介面。此通訊協定使用 WebSocket 進行通訊。

## 設定

```json
{
  "channel_list": {
    "onebot": {
      "enabled": true,
      "type": "onebot",
      "ws_url": "ws://localhost:8080",
      "access_token": "",
      "allow_from": []
    }
  }
}
```

| 欄位         | 型別   | 必填 | 說明                                       |
| ------------ | ------ | ---- | ------------------------------------------ |
| enabled      | bool   | 是   | 是否啟用 OneBot 頻道                       |
| ws_url       | string | 是   | OneBot 伺服器的 WebSocket URL              |
| access_token | string | 否   | 用來連線至 OneBot 伺服器的 Access Token    |
| allow_from   | array  | 否   | 使用者 ID 允許清單；留空表示允許所有使用者 |

## 設定步驟

1. 部署與 OneBot 相容的實作（例如 napcat）
2. 設定 OneBot 實作以啟用 WebSocket 服務，並視需要設定 Access Token
3. 在設定檔填入 WebSocket URL 與 Access Token
