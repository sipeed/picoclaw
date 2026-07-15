> 回到 [README](../../project/README.zh-tw.md)

# Delta Chat 頻道

PicoClaw 可以啟動本機 `deltachat-rpc-server` 處理程序，並使用 JSON-RPC 與其通訊，以 Delta Chat Bot 身分執行。RPC 伺服器會處理電子郵件帳戶、IMAP／SMTP 連線、訊息儲存區及加密金鑰。

## 安裝

請安裝 Delta Chat RPC 伺服器。如果 `deltachat-rpc-server` 位於 `PATH`，PicoClaw 可以自動找到；否則，請將 `rpc_server_path` 設為確切的二進位檔路徑。

```bash
pip install deltachat-rpc-server
which deltachat-rpc-server
```

也可以從 [Delta Chat Core 發行版本](https://github.com/deltachat/deltachat-core-rust/releases) 取得預先建置的二進位檔。

## 設定

最簡單的設定方式，是讓 PicoClaw 在 Delta Chat 的本機帳戶儲存區中建立 chatmail 帳戶。請在 `email` 中填入本機部分留空的 Relay 標記，例如 `@nine.testrun.org`：

```json
{
  "channel_list": {
    "deltachat": {
      "enabled": true,
      "type": "deltachat",
      "allow_from": ["friend@example.org"],
      "group_trigger": {
        "mention_only": true
      },
      "settings": {
        "email": "@nine.testrun.org",
        "display_name": "PicoClaw Bot",
        "avatar_image": "/home/me/bot-avatar.png"
      }
    }
  }
}
```

PicoClaw 啟動時會使用 `deltachat-rpc-server` 建立帳戶，接著停止執行，並在錯誤訊息中顯示剛建立的完整位址。請將 Relay 標記換成該完整電子郵件地址，再次執行 PicoClaw：

```json
{
  "email": "bot123@nine.testrun.org",
  "display_name": "PicoClaw Bot",
  "avatar_image": "/home/me/bot-avatar.png"
}
```

如果缺少 `email`，啟動錯誤會列出從 Parla 複製而來的內建 Relay 選項。可以使用其中一個 Relay 標記，也可以使用相同 `@server.name` 格式的自訂 chatmail Relay。

PicoClaw 建立的 chatmail 帳戶不需要 `password`。當 `email` 指向 `data_dir` 中已設定的帳戶時，請省略此欄位；信箱密碼由 JSON-RPC 伺服器管理。舊版密碼設定方式只保留給必須由 PicoClaw 自行設定的傳統電子郵件帳戶。在這種模式下，`password` 是安全欄位；系統第一次載入設定時，會將其移至 `~/.picoclaw/.security.yml`，也可以使用 `PICOCLAW_CHANNELS_DELTACHAT_PASSWORD` 設定。

`display_name` 及 `avatar_image` 是選用的個人資料設定。如有設定，PicoClaw 每次啟動時都會套用，因此只要變更設定中的大頭貼路徑，即可更新 Bot 個人資料。

| 欄位 | 必填 | 說明 |
|------|------|------|
| `email` | 是 | Bot 信箱的完整地址，或首次執行使用的 Relay 標記，例如 `@nine.testrun.org` |
| `rpc_server_path` | 否 | `deltachat-rpc-server` 的路徑；只有未將其放入 `PATH` 時才需要設定 |
| `password` | 否 | 僅供舊版使用；PicoClaw 必須自行設定或重新設定傳統信箱時才需要 |
| `display_name` | 否 | 啟動時套用的個人資料名稱；聯絡人會看到此名稱，群組提及偵測也會使用 |
| `avatar_image` | 否 | 啟動時套用的大頭貼圖片路徑；系統會展開 `~`。如果檔案不存在，系統會顯示警告並略過 |
| `data_dir` | 否 | 帳戶資料庫目錄。預設值：`~/.picoclaw/deltachat/<channel-name>` |
| `invite_link` | 否 | 啟動時要加入的 Delta Chat 邀請連結 |
| `allow_crosspost` | 否 | 預設為 `false`。設為 `true` 後，`allow_from` 允許的傳送者可以使用 `message` 工具，將訊息傳送至目前對話以外的目標，或依電子郵件／聯絡人／聊天名稱解析收件者 |
| `imap_server`、`imap_port` | 否 | 覆寫密碼式設定使用的 IMAP 設定 |
| `smtp_server`、`smtp_port` | 否 | 覆寫密碼式設定使用的 SMTP 設定 |

`allow_from`、`group_trigger` 及 `reasoning_channel_id` 等標準頻道欄位也適用。

## 首次執行

將 `email` 設為 `@server` 後，PicoClaw 會建立 chatmail 帳戶，在啟動錯誤中顯示建立的完整電子郵件地址，然後結束執行。請將 `email` 更新為該完整地址，再次執行 PicoClaw。後續執行時，PicoClaw 會依 `email` 選取已設定的帳戶、套用選用的個人資料設定、將帳戶標示為 Bot，然後開始 I/O。

使用新的 `data_dir` 及舊版 `password` 時，PicoClaw 仍可設定傳統電子郵件帳戶並驗證信箱認證資訊；完成後會重複使用本機資料目錄中的帳戶。

Delta Chat 要求對等端先取得 Bot 的加密金鑰，才能傳送訊息。PicoClaw 啟動時會顯示 Bot 邀請連結及 QR Code。請在 Delta Chat 中使用該邀請加入 Bot，不要直接輸入電子郵件地址。

## 行為

- 私人聊天經 `allow_from` 檢查後一律回覆。
- 群組聊天依循 `group_trigger`；未設定時會處理所有群組訊息。
- 系統會略過 Bot 本身傳送的訊息、裝置聊天，以及資訊／系統訊息。
- 收到的訊息經允許清單檢查後，系統會標示為已讀。
- 系統會將收到的附件（圖片、音訊、影片、文件）登記至媒體儲存區，再交給代理人，讓代理人可以直接瀏覽圖片或操作檔案。如果沒有可用的媒體儲存區，系統會改為在訊息內附加 `[attachment: /path]` 路徑。
- 支援傳送附件：代理人輸出媒體時，系統會將每個檔案各自以 Delta Chat 訊息傳送（說明文字會放在訊息本文）。Delta Chat 會依檔案推斷顯示類型，因此圖片、GIF 及影片都能以原生方式呈現。
- 跨對話收件者查詢預設為停用。代理人一律可以回覆目前的數字聊天 ID；如需傳送至其他數字聊天 ID，或依電子郵件／聯絡人／聊天名稱解析目標，必須設定 `allow_crosspost: true`，而且目前的傳送者必須符合 `allow_from` 規則；`allow_from: ["*"]` 會允許所有傳送者使用此功能。
- 設定語音服務供應商後，即可雙向使用語音功能：代理人會使用 ASR 轉錄收到的語音訊息，並將逐字稿交給模型；代理人也能使用合成語音回覆，系統會以原生 Delta Chat 語音訊息傳送（`send_tts`）。這需要在 `voice` 下設定 ASR、TTS 或兩者，並非 Delta Chat 專用設定。

## 疑難排解

| 現象 | 修正方式 |
|------|----------|
| `deltachat-rpc-server not found on PATH` 或 `rpc_server_path ... not found` | 將 RPC 伺服器安裝至 PATH，或將 `rpc_server_path` 設為絕對路徑 |
| `email is required` | 選擇列出的其中一個 chatmail 伺服器，將 `email` 設為首次執行標記（例如 `@nine.testrun.org`），執行 `picoclaw g`，再換成建立的完整電子郵件地址 |
| `created chatmail account ...` | 將 `email` 中的 `@server` 標記換成建立的完整電子郵件地址，再次執行 PicoClaw |
| `account ... is not configured in data_dir` | 將 `data_dir` 指向現有的 JSON-RPC 帳戶儲存區，或使用 `email="@server"` 建立帳戶 |
| `configure (check email/password/server)` | 檢查認證資訊、應用程式密碼要求或 IMAP／SMTP 覆寫設定 |
| Bot 未在群組中回覆 | 檢查 `group_trigger`；提及 `display_name`，或使用已設定的前置字串 |
| Bot 略過某位傳送者 | 將傳送者的電子郵件地址加入 `allow_from`，或使用 `["*"]` 開放存取 |
| 傳送者無法傳訊息給 Bot | 使用啟動時顯示的 QR Code／邀請重新加入 Bot，讓 Delta Chat 建立加密連線 |
| 代理人無法傳送至電子郵件／名稱／其他聊天 ID | 啟用 `settings.allow_crosspost`，並在 `allow_from` 中允許控制端傳送者；基於隱私考量，此功能預設為停用 |
