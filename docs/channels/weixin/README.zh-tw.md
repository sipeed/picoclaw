# 💬 微信（WeChat 個人帳戶）頻道

PicoClaw 支援使用騰訊官方 iLink API 連線至微信個人帳戶。

## 🚀 快速綁定

設定微信頻道最簡單的方式，是使用互動式綁定命令：

```bash
picoclaw auth weixin
```

此命令會：
1. 向 iLink API 要求 QR Code，並顯示在終端機中。
2. 等候使用微信行動應用程式掃描 QR Code。
3. 核准後，自動將取得的 Access Token 儲存至 `~/.picoclaw/config.json`。

完成綁定後，即可啟動閘道服務：

```bash
picoclaw gateway
```

---

## ⚙️ 設定

也可以在 `config.json` 的 `channels.weixin` 區段手動設定篩選規則。

```json
{
  "channel_list": {
    "weixin": {
      "enabled": true,
      "type": "weixin",
      "token": "YOUR_WEIXIN_TOKEN",
      "allow_from": [
        "user_id_1",
        "user_id_2"
      ],
      "proxy": ""
    }
  }
}
```

### 設定欄位

| 欄位 | 說明 |
|---|---|
| `enabled` | 設為 `true`，即可在啟動時啟用此頻道。 |
| `token` | 掃描 QR Code 登入後取得的驗證 Token。 |
| `allow_from` | （選用）允許與 Bot 互動的微信使用者 ID 清單。留空時，任何能傳送訊息給已連線帳戶的人都可觸發 Bot。 |
| `proxy` | （選用）HTTP Proxy 位址（例如 `http://localhost:7890`），適用於無法直接連線至 `ilinkai.weixin.qq.com` 的環境。 |

## ⚠️ 重要注意事項

- **僅限一個帳戶**：iLink Token 會綁定至單一工作階段。如果另一部裝置完成授權，之後開始新的互動通常會使舊 Token 失效。
- **訊息速率限制**：為避免微信的垃圾訊息防護機制限制帳戶，請避免循環觸發或高頻率廣播。
