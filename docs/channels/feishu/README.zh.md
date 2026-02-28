# 飞书

飞书（国际版名称：Lark）是字节跳动旗下的企业协作平台。它通过事件驱动的 Webhook 同时支持中国和全球市场。

## 配置

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "app_id": "cli_xxx",
      "app_secret": "xxx",
      "encrypt_key": "",
      "verification_token": "",
      "allow_from": []
    }
  }
}
```

| 字段               | 类型   | 必填 | 描述                             |
| ------------------ | ------ | ---- | -------------------------------- |
| enabled            | bool   | 是   | 是否启用飞书频道                 |
| app_id             | string | 是   | 飞书应用的 App ID(以cli\_开头)   |
| app_secret         | string | 是   | 飞书应用的 App Secret            |
| encrypt_key        | string | 否   | 事件回调加密密钥                 |
| verification_token | string | 否   | 用于Webhook事件验证的Token       |
| allow_from         | array  | 否   | 用户ID白名单，空表示允许所有用户 |

## 设置流程

1. 前往 [飞书开放平台](https://open.feishu.cn/)创建应用程序
2. 获取 App ID 和 App Secret
3. 配置事件订阅和Webhook URL
4. 设置加密(可选,生产环境建议启用)
5. 将 App ID、App Secret、Encrypt Key 和 Verification Token(如果启用加密) 填入配置文件中

## 发送本地图片

飞书频道现已支持通过 `im/v1/image/create` 上传本地图片并发送图片消息。

- 如果发送内容是本地图片路径（如 `/tmp/a.png` 或 `file:///tmp/a.png`），会自动上传并发送图片。
- 如果发送内容包含 Markdown 图片语法（如 `![img](/tmp/a.png)`），会提取并发送图片。
- 同一条消息里如果同时包含文本和 Markdown 图片，文本会先发送，再发送图片。

支持格式：`jpg/jpeg/png/webp/gif/tiff/bmp/ico`。
