> 返回 [README](../../../README.zh.md)

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
      "allow_from": [],
      "is_lark": false
    }
  }
}
```

| 字段                  | 类型   | 必填 | 描述                                                                                             |
| --------------------- | ------ | ---- | ------------------------------------------------------------------------------------------------ |
| enabled               | bool   | 是   | 是否启用飞书频道                                                                                 |
| app_id                | string | 是   | 飞书应用的 App ID(以cli\_开头)                                                                   |
| app_secret            | string | 是   | 飞书应用的 App Secret                                                                            |
| encrypt_key           | string | 否   | 事件回调加密密钥                                                                                 |
| verification_token    | string | 否   | 用于Webhook事件验证的Token                                                                       |
| allow_from            | array  | 否   | 用户ID白名单，空表示所有用户                                                                     |
| random_reaction_emoji | array  | 否   | 随机添加的表情列表，空则使用默认 "Pin"                                                           |
| is_lark               | bool   | 否   | 是否使用 Lark 国际版域名（`open.larksuite.com`），默认为 `false`（使用飞书域名 `open.feishu.cn`） |

## 设置流程

1. 前往 [飞书开放平台](https://open.feishu.cn/)（国际版用户请前往 [Lark 开放平台](https://open.larksuite.com/)）创建应用
2. 在应用设置中启用**机器人**能力
3. 创建版本并发布应用（应用发布后配置才会生效）
4. 获取 **App ID**（以 `cli_` 开头）和 **App Secret**
5. 将 App ID 和 App Secret 填入 PicoClaw 配置文件
6. 运行 `picoclaw gateway` 启动服务
7. 在飞书中搜索机器人名称，开始对话

> PicoClaw 使用 WebSocket/SDK 模式连接飞书，无需配置公网回调地址或 Webhook URL。
>
> `encrypt_key` 和 `verification_token` 为可选项，生产环境建议启用事件加密。
>
> 自定义表情参考：[飞书表情列表](https://open.larkoffice.com/document/server-docs/im-v1/message-reaction/emojis-introduce)
