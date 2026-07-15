> 回到 [README](../../project/README.zh-tw.md)

# Telegram

Telegram 頻道會使用 Telegram Bot API 的長輪詢機制進行 Bot 通訊。此頻道支援文字訊息、媒體附件（相片、語音、音訊、文件）、語音轉錄（請參閱 [設定方式](../../guides/providers.zh-tw.md#語音轉錄)），以及內建命令處理功能。

## 設定

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
      "allow_from": ["123456789"],
      "proxy": "",
      "use_markdown_v2": false,
      "media_group_delay_ms": 500
    }
  }
}
```

| 欄位                 | 型別   | 必填 | 說明                                                         |
| -------------------- | ------ | ---- | ------------------------------------------------------------ |
| enabled              | bool   | 是   | 是否啟用 Telegram 頻道                                      |
| token                | string | 是   | Telegram Bot API Token                                       |
| allow_from           | array  | 否   | 使用者 ID 允許清單；留空表示允許所有使用者                   |
| proxy                | string | 否   | 用來連線至 Telegram API 的 Proxy URL（例如 http://127.0.0.1:7890） |
| use_markdown_v2      | bool   | 否   | 啟用 Telegram MarkdownV2 格式                                |
| media_group_delay_ms | int    | 否   | 處理 Telegram 媒體群組／相簿前的閒置等候時間，預設為 500 ms |

## 設定步驟

1. 在 Telegram 搜尋 `@BotFather`
2. 傳送 `/newbot` 命令，並依照提示建立新的 Bot
3. 取得 HTTP API Token
4. 在設定檔填入 Token
5. （選用）設定 `allow_from`，限制可互動的使用者 ID（可利用 `@userinfobot` 取得 ID）

## 內建命令

Telegram 會在 PicoClaw 啟動時自動註冊頂層 Bot 命令，包括 `/start`、`/help`、`/show`、`/list` 及 `/use`。

技能相關命令：

- `/list skills`：列出目前代理人可見的已安裝技能。
- `/list mcp`：列出已設定的 MCP 伺服器及其延後載入／連線狀態。
- `/show mcp <server>`：列出已連線 MCP 伺服器的作用中工具。
- `/use <skill> <message>`：強制單次要求使用指定技能。
- `/use <skill>`：指定同一對話中的下一則訊息使用該技能。
- `/use clear`：清除尚未套用的技能覆寫設定。

範例：

```text
/list skills
/list mcp
/show mcp github
/use git explain how to squash the last 3 commits
/use git
explain how to squash the last 3 commits
```

## 進階格式

將 `use_markdown_v2` 設為 `true`，即可啟用更豐富的格式功能。Bot 因此能完整使用 Telegram MarkdownV2 功能，包括巢狀樣式、隱藏文字及自訂固定寬度區塊。

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"],
      "use_markdown_v2": true
    }
  }
}
```
