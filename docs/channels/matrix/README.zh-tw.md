> 回到 [README](../../project/README.zh-tw.md)

# Matrix 頻道設定指南

## 1. 設定範例

將下列內容加入 `config.json`：

```json
{
  "channel_list": {
    "matrix": {
      "enabled": true,
      "type": "matrix",
      "homeserver": "https://matrix.org",
      "user_id": "@your-bot:matrix.org",
      "access_token": "YOUR_MATRIX_ACCESS_TOKEN",
      "device_id": "",
      "join_on_invite": true,
      "allow_from": [],
      "group_trigger": {
        "mention_only": true
      },
      "placeholder": {
        "enabled": true,
        "text": ["Thinking...", "Processing...", "Typing..."]
      },
      "reasoning_channel_id": "",
      "message_format": "richtext",
      "crypto_database_path": "",
      "crypto_passphrase": "YOUR_MATRIX_CRYPTO_PICKLE_KEY"
    }
  }
}
```

## 2. 欄位參考

| 欄位                 | 型別     | 必填 | 說明 |
|----------------------|----------|------|------|
| enabled              | bool     | 是   | 啟用或停用 Matrix 頻道 |
| homeserver           | string   | 是   | Matrix homeserver URL（例如 `https://matrix.org`） |
| user_id              | string   | 是   | Bot 的 Matrix 使用者 ID（例如 `@bot:matrix.org`） |
| access_token         | string   | 是   | Bot Access Token |
| device_id            | string   | 否   | 選用的 Matrix 裝置 ID |
| join_on_invite       | bool     | 否   | 受邀時自動加入聊天室 |
| allow_from           | []string | 否   | 使用者允許清單（Matrix 使用者 ID） |
| group_trigger        | object   | 否   | 群組觸發策略（`mention_only`／`prefixes`） |
| placeholder          | object   | 否   | 暫代訊息設定（請參閱下方說明） |
| reasoning_channel_id | string   | 否   | 用來輸出推理內容的目標頻道 |
| message_format       | string   | 否   | 輸出格式：`"richtext"`（預設）會將 Markdown 轉譯為 HTML；`"plain"` 僅傳送純文字 |
| crypto_database_path | string   | 否   | 儲存加密資料庫的路徑（留空時使用工作區路徑 `~/.picoclaw/workspace`） |
| crypto_passphrase    | string   | 否   | 用來加密資料庫工作階段金鑰的序列化金鑰；設定後不得變更 |

### 暫代訊息設定

| 欄位    | 型別            | 必填 | 說明 |
|---------|-----------------|------|------|
| enabled | bool            | 否   | 啟用暫代訊息（預設：false） |
| text    | string/[]string | 否   | 暫代文字，可使用單一字串或字串陣列。提供多段文字時，執行階段會隨機選取其中一段。預設值為 `Thinking...` |

## 3. 目前支援的功能

- 傳送及接收文字訊息，並轉譯 Markdown 格式（粗體、斜體、標題、程式碼區塊等）
- 可設定訊息格式（`richtext`／`plain`）
- 下載收到的圖片、音訊、影片及檔案（優先使用 MediaStore，失敗時改用本機路徑）
- 將收到的音訊標準化後交由既有轉錄流程處理（`[audio: ...]`）
- 上傳及傳送圖片、音訊、影片與檔案
- 群組觸發規則（包括僅在提及時觸發）
- 輸入狀態（`m.typing`）
- 以最終回覆取代暫代訊息
- 受邀時自動加入聊天室（可停用）
- 加密訊息的端對端加密（E2EE）功能

## 4. 待辦事項

- 改善多媒體中繼資料（例如圖片／影片尺寸與縮圖）
