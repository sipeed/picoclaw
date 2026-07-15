> 回到 [README](../../project/README.zh-tw.md)

# QQ

PicoClaw 使用 QQ 開放平台的官方 Bot API 提供 QQ 頻道功能。

## 設定

```json
{
  "channel_list": {
    "qq": {
      "enabled": true,
      "type": "qq",
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

| 欄位       | 型別   | 必填 | 說明                                       |
| ---------- | ------ | ---- | ------------------------------------------ |
| enabled    | bool   | 是   | 是否啟用 QQ 頻道                           |
| app_id     | string | 是   | QQ Bot 應用程式的 App ID                   |
| app_secret | string | 是   | QQ Bot 應用程式的 App Secret               |
| allow_from | array  | 否   | 使用者 ID 允許清單；留空表示允許所有使用者 |

## 設定步驟

### 快速設定（建議）

QQ 開放平台提供按一下即可建立 Bot 的入口：

1. 開啟 [QQ Bot 快速建立頁面](https://q.qq.com/qqbot/openclaw/index.html)，並掃描 QR Code 登入
2. 系統會自動建立 Bot，請複製 **App ID** 及 **App Secret**
3. 在 PicoClaw 設定檔填入認證資訊
4. 執行 `picoclaw gateway` 啟動服務
5. 開啟 QQ 並開始與 Bot 對話

> App Secret 只會顯示一次，請立即妥善儲存。再次檢視會強制重設 App Secret。
>
> 從快速入口建立的 Bot 僅供建立者本人使用，不支援群組聊天。如需使用群組聊天，請在 [QQ 開放平台](https://q.qq.com/) 設定沙盒模式。

### 手動設定

1. 使用 QQ 帳戶登入 [QQ 開放平台](https://q.qq.com/)，並註冊為開發人員
2. 建立 QQ Bot 並自訂大頭貼與名稱
3. 從 Bot 設定取得 **App ID** 及 **App Secret**
4. 在 PicoClaw 設定檔填入認證資訊
5. 執行 `picoclaw gateway` 啟動服務
6. 在 QQ 搜尋 Bot 並開始對話

> 開發期間，建議啟用沙盒模式，並將測試使用者與群組加入沙盒以便除錯。
