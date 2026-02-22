# 钉钉

钉钉是阿里巴巴的企业通讯平台，在中国职场中广受欢迎。它采用流式 SDK 来维持持久连接。

## 配置

```json
{
  "channels": {
    "dingtalk": {
      "enabled": true,
      "client_id": "YOUR_CLIENT_ID",
      "client_secret": "YOUR_CLIENT_SECRET",
      "allow_from": []
    }
  }
}
```

| 字段          | 类型   | 必填 | 描述                             |
| ------------- | ------ | ---- | -------------------------------- |
| enabled       | bool   | 是   | 是否启用钉钉频道                 |
| client_id     | string | 是   | 钉钉应用的 Client ID             |
| client_secret | string | 是   | 钉钉应用的 Client Secret         |
| allow_from    | array  | 否   | 用户ID白名单，空表示允许所有用户 |

## 设置流程

1. 前往 [钉钉开放平台](https://open.dingtalk.com/)
2. 创建一个企业内部应用
3. 从应用设置中获取 Client ID 和 Client Secret
4. 配置OAuth和事件订阅(如需要)
5. 将 Client ID 和 Client Secret 填入配置文件中
