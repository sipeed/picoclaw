> 回到 [README](../../project/README.zh-tw.md)

# Feishu

Feishu（國際版名稱：Lark）是 ByteDance 推出的企業協作平台。此平台使用事件驅動的 WebSocket 連線，服務中國及全球市場。

## 設定

```json
{
  "channel_list": {
    "feishu": {
      "enabled": true,
      "type": "feishu",
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    }
  }
}
```

| 欄位                  | 型別   | 必填 | 說明                                                |
| --------------------- | ------ | ---- | --------------------------------------------------- |
| enabled               | bool   | 是   | 是否啟用 Feishu 頻道                                |
| app_id                | string | 是   | Feishu 應用程式的 App ID（以 `cli_` 開頭）          |
| app_secret            | string | 是   | Feishu 應用程式的 App Secret                        |
| encrypt_key           | string | 否   | 事件回呼的加密金鑰                                  |
| verification_token    | string | 否   | 用來驗證 Webhook 事件的 Token                       |
| allow_from            | array  | 否   | 使用者 ID 允許清單；留空表示允許所有使用者          |
| random_reaction_emoji | array  | 否   | 隨機回應表情符號清單；留空時使用預設的 `Pin`        |

## 設定步驟

1. 前往 [Feishu Open Platform](https://open.feishu.cn/) 並建立應用程式
2. 在應用程式設定中啟用 **[Bot]** 功能
3. 建立版本後發布應用程式（設定只會在發布後生效）
4. 取得 **App ID**（以 `cli_` 開頭）及 **App Secret**
5. 在 PicoClaw 設定檔填入 App ID 與 App Secret
6. 執行 `picoclaw gateway` 啟動服務
7. 在 Feishu 搜尋 Bot 名稱並開始對話

> PicoClaw 使用 WebSocket／SDK 模式連線至 Feishu，不需要公開的回呼位址或 Webhook URL。
>
> `encrypt_key` 及 `verification_token` 為選填；正式環境建議啟用事件加密。
>
> 自訂表情符號的參考資料，請參閱：[Feishu 表情符號清單](https://open.larkoffice.com/document/server-docs/im-v1/message-reaction/emojis-introduce)

## 平台限制

> ⚠️ **Feishu 頻道不支援 32 位元裝置。** Feishu SDK 只提供 64 位元版本。使用 armv6、armv7、mipsle 或其他 32 位元架構的裝置無法使用 Feishu 頻道。如需在 32 位元裝置上收發訊息，請改用 Telegram、Discord 或 OneBot。
