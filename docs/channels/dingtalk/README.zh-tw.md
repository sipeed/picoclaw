> 回到 [README](../../project/README.zh-tw.md)

# DingTalk

DingTalk 是阿里巴巴的企業通訊平台，廣泛用於中國的工作場所。此平台使用串流 SDK 維持持續連線。

## 設定

```json
{
  "channel_list": {
    "dingtalk": {
      "enabled": true,
      "type": "dingtalk",
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

| 欄位          | 型別   | 必填 | 說明                                       |
| ------------- | ------ | ---- | ------------------------------------------ |
| enabled       | bool   | 是   | 是否啟用 DingTalk 頻道                     |
| client_id     | string | 是   | DingTalk 應用程式的 Client ID              |
| client_secret | string | 是   | DingTalk 應用程式的 Client Secret          |
| allow_from    | array  | 否   | 使用者 ID 允許清單；留空表示允許所有使用者 |

## 設定步驟

1. 前往 [DingTalk Open Platform](https://open.dingtalk.com/)
2. 建立企業內部應用程式
3. 從應用程式設定取得 Client ID 及 Client Secret
4. 視需要設定 OAuth 及事件訂閱
5. 在設定檔填入 Client ID 與 Client Secret
