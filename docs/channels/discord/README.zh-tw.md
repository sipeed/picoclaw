> 回到 [README](../../project/README.zh-tw.md)

# Discord

Discord 是供社群使用的免費語音、視訊與文字聊天應用程式。PicoClaw 使用 Discord Bot API 連線至 Discord 伺服器，可接收及傳送訊息。

## 設定

```json
{
  "agents": {
    "defaults": {
      "tool_feedback": {
        "enabled": true,
        "max_args_length": 300
      }
    }
  },
  "channel_list": {
    "discord": {
      "enabled": true,
      "type": "discord",
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"],
      "placeholder": {
        "enabled": true,
        "text": ["Thinking... 💭"]
      },
      "group_trigger": {
        "mention_only": false
      },
      "reasoning_channel_id": ""
    }
  }
}
```

| 欄位                 | 型別   | 必填 | 說明                                                   |
| -------------------- | ------ | ---- | ------------------------------------------------------ |
| enabled              | bool   | 是   | 是否啟用 Discord 頻道                                  |
| token                | string | 是   | Discord Bot Token                                      |
| allow_from           | array  | 否   | 使用者 ID 允許清單；留空表示允許所有使用者             |
| placeholder          | object | 否   | 代理人處理工作時顯示的暫代訊息設定                     |
| group_trigger        | object | 否   | 群組觸發設定（例如：{ "mention_only": false }）       |
| reasoning_channel_id | string | 否   | 用來輸出推理／思考內容的選用目標頻道 ID                |

## 可見的執行回饋

Discord 可顯示三種不同的「處理中」回饋：

1. 輸入狀態指示：自動顯示，不需要其他設定。
2. 暫代訊息：啟用 `channel_list.discord.placeholder.enabled` 後，系統會傳送使用者可見的 `Thinking...` 訊息，並在完成時將其編輯為最終回覆。
3. 工具執行回饋：啟用 `agents.defaults.tool_feedback.enabled` 後，系統會在每次呼叫工具前傳送簡短訊息，例如：

```text
🔧 `web_search`
Checking the latest PicoClaw release notes before I answer.
```

如果只看到 `Bot is typing`，請確認執行階段設定確實包含 `placeholder.enabled` 或 `tool_feedback.enabled`。

## 設定步驟

1. 前往 [Discord Developer Portal](https://discord.com/developers/applications) 並建立新的應用程式
2. 啟用 Intents：
   - [訊息內容意圖]
   - [伺服器成員意圖]
3. 取得 Bot Token
4. 在設定檔填入 Bot Token
5. 邀請 Bot 加入伺服器，並授予必要權限（例如 [傳送訊息]、[讀取訊息記錄]）
