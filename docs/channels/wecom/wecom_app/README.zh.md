# 企业微信自建应用

企业微信自建应用是指企业在企业微信中创建的应用，主要用于企业内部使用。通过企业微信自建应用，企业可以实现与员工的高效沟通和协作，提高工作效率。

## 配置

```json
{
  "channels": {
    "wecom_app": {
      "enabled": true,
      "corp_id": "wwxxxxxxxxxxxxxxxx",
      "corp_secret": "YOUR_CORP_SECRET",
      "agent_id": 1000002,
      "token": "YOUR_TOKEN",
      "encoding_aes_key": "YOUR_ENCODING_AES_KEY",
      "webhook_path": "/webhook/wecom-app",
      "allow_from": [],
      "reply_timeout": 5
    }
  }
}
```

| 字段             | 类型   | 必填 | 描述                                     |
| ---------------- | ------ | ---- | ---------------------------------------- |
| corp_id          | string | 是   | 企业 ID                                  |
| corp_secret      | string | 是   | 应用程序密钥                             |
| agent_id         | int    | 是   | 应用程序代理 ID                          |
| token            | string | 是   | 回调验证令牌                             |
| encoding_aes_key | string | 是   | 43 字符 AES 密钥                         |
| webhook_path     | string | 否   | Webhook 路径（默认：/webhook/wecom-app） |
| allow_from       | array  | 否   | 用户 ID 白名单                           |
| reply_timeout    | int    | 否   | 回复超时时间（秒）                       |

## 设置流程

1. 登录 [企业微信管理后台](https://work.weixin.qq.com/)
2. 进入“应用管理” -> “创建应用”
3. 获取企业 ID (CorpID) 和应用 Secret
4. 在应用设置中配置“接收消息”，获取 Token 和 EncodingAESKey
5. 设置回调 URL 为 `http://<your-server-ip>:<port>/webhook/wecom-app`
6. 将 CorpID, Secret, AgentID 等信息填入配置文件

   注意: PicoClaw 现在使用共享的 Gateway HTTP 服务器来接收所有渠道的 webhook 回调，默认监听地址为 127.0.0.1:18790。如需从公网接收回调，请把外部域名反向代理到 Gateway（默认端口 18790）。
