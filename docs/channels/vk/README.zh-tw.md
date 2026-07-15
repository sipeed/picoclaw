# VK（VKontakte）

VK 頻道使用 Bots Long Poll API 與 VK 社群網路進行 Bot 通訊。此頻道支援文字訊息、媒體附件（相片、影片、音訊、文件、貼圖）及群組聊天互動。

## 設定

```json
{
  "channel_list": {
    "vk": {
      "enabled": true,
      "type": "vk",
      "token": "NOT_HERE",
      "group_id": 123456789,
      "allow_from": ["123456789"],
      "group_trigger": {
        "mention_only": false,
        "prefixes": ["/bot", "!bot"]
      }
    }
  }
}
```

| 欄位          | 型別   | 必填 | 說明                                                |
| ------------- | ------ | ---- | --------------------------------------------------- |
| enabled       | bool   | 是   | 是否啟用 VK 頻道                                    |
| token         | string | 是   | 設為 `NOT_HERE`；Token 會安全地另外儲存（請參閱 Token 儲存方式） |
| group_id      | int    | 是   | VK 社群 ID（Group ID）                               |
| allow_from    | array  | 否   | 使用者 ID 允許清單；留空表示允許所有使用者          |
| group_trigger | object | 否   | 群組聊天觸發設定                                    |

### Token 儲存方式

基於安全考量，不應將 VK Access Token 直接儲存在設定檔中。請改用下列方式：

1. 在設定中將 `token` 設為 `"NOT_HERE"`
2. 使用下列任一方式儲存實際 Token：
   - **環境變數**：設定 `PICOCLAW_CHANNELS_VK_TOKEN` 環境變數
   - **安全儲存區**：使用 PicoClaw 的安全 Token 儲存機制

使用環境變數的範例：
```bash
export PICOCLAW_CHANNELS_VK_TOKEN="vk1.a.abc123..."
```

### 群組觸發設定

| 欄位         | 型別     | 說明                                   |
| ------------ | -------- | -------------------------------------- |
| mention_only | bool     | 只有在群組聊天中提及 Bot 時才回應      |
| prefixes     | []string | 觸發 Bot 回應的群組聊天前置字串清單    |

## 設定步驟

### 1. 建立 VK 社群

1. 前往 [VK](https://vk.com) 並登入
2. 建立新社群或使用現有社群
3. 記下社群 ID（可在社群 URL 中找到，例如 `public123456789`）

### 2. 啟用訊息功能

1. 前往社群頁面
2. 按一下 [管理] → [訊息] → [社群訊息]
3. 啟用社群訊息

### 3. 建立 Access Token

1. 前往 [管理] → [API 使用方式] → [Access Token]
2. 按一下 [建立 Token]
3. 選取下列權限：
   - `messages` - 存取訊息
   - `photos` - 存取相片（選用）
   - `docs` - 存取文件（選用）
4. 複製剛取得的 Access Token
5. 安全地儲存 Token（請參閱下方的 Token 儲存方式）

### 4. 設定 PicoClaw

1. 將 Token 加入 PicoClaw 設定
2. 將 `group_id` 設為社群 ID（數值）
3. （選用）設定 `allow_from`，限制可互動的使用者 ID

## 功能

### 支援的訊息類型

- **文字訊息**：完整支援文字訊息
- **相片**：相片會顯示為 `[photo]` 暫代文字
- **影片**：影片會顯示為 `[video]` 暫代文字
- **音訊**：音訊檔案會顯示為 `[audio]` 暫代文字
- **語音訊息**：語音訊息會顯示為 `[voice]` 暫代文字，並支援轉錄
- **文件**：文件會顯示為 `[document: filename]`
- **貼圖**：貼圖會顯示為 `[sticker]` 暫代文字

### 語音功能

VK 頻道支援接收語音訊息及文字轉語音功能：

- **ASR（自動語音辨識）**：可使用已設定的語音模型，將語音訊息轉錄為文字
- **TTS（文字轉語音）**：可將文字回覆轉換為語音訊息

若要啟用語音轉錄，請在服務供應商設定中設定語音模型。詳細資訊請參閱 [語音轉錄](../../guides/providers.zh-tw.md#語音轉錄)。

### 群組聊天功能

VK 頻道支援使用可設定的觸發方式處理群組聊天：

- **僅提及模式**：只有提及 Bot 時才回應
- **前置字串模式**：訊息以指定前置字串開頭時，Bot 才會回應
- **寬鬆模式**：Bot 會回應所有訊息（預設）

### 訊息長度

VK 的訊息長度上限為 4000 個字元。PicoClaw 會自動將較長的訊息分成多個部分。

## 設定範例

### 基本設定

```json
{
  "channel_list": {
    "vk": {
      "enabled": true,
      "type": "vk",
      "token": "NOT_HERE",
      "group_id": 123456789
    }
  }
}
```

### 設有使用者允許清單

```json
{
  "channel_list": {
    "vk": {
      "enabled": true,
      "type": "vk",
      "token": "NOT_HERE",
      "group_id": 123456789,
      "allow_from": ["123456789", "987654321"]
    }
  }
}
```

### 設有群組聊天觸發方式

```json
{
  "channel_list": {
    "vk": {
      "enabled": true,
      "type": "vk",
      "token": "NOT_HERE",
      "group_id": 123456789,
      "group_trigger": {
        "prefixes": ["/bot", "!bot"]
      }
    }
  }
}
```

## 疑難排解

### Bot 沒有回應

1. 檢查 Access Token 是否有效
2. 確認 `group_id` 是否正確
3. 如果已設定 `allow_from`，請確認允許清單中包含該使用者 ID
4. 檢查 PicoClaw 記錄中的錯誤訊息

### 權限錯誤

請確認 Access Token 具有必要權限：
- `messages`：傳送及接收訊息所需的必要權限
- `photos`：處理相片附件的選用權限
- `docs`：處理文件附件的選用權限

### 群組聊天問題

如果 Bot 未在群組聊天中回應：
1. 檢查 `group_trigger` 設定
2. 嘗試使用前置字串觸發 Bot
3. 檢查 Bot 是否具有讀取群組訊息的權限

## API 參考資料

VK 頻道使用 [VK SDK for Go](https://github.com/SevereCloud/vksdk) 程式庫，此程式庫支援 VK API 5.199 版。

如需 VK API 的詳細資訊，請參閱：
- [VK API 文件](https://dev.vk.com/en)
- [VK Bots Long Poll API](https://dev.vk.com/en/api/bots-long-poll/getting-started)
