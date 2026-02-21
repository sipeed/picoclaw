# QQ

PicoClaw 通过 QQ 开放平台的官方机器人 API 提供对 QQ 的支持。

## 配置

```json
{
  "channels": {
    "qq": {
      "enabled": true,
      "app_id": "YOUR_APP_ID",
      "app_secret": "YOUR_APP_SECRET",
      "allow_from": []
    }
  }
}
```

| 字段       | 类型   | 必填 | 描述                             |
| ---------- | ------ | ---- | -------------------------------- |
| enabled    | bool   | 是   | 是否启用 QQ Channel              |
| app_id     | string | 是   | QQ 机器人应用的 App ID           |
| app_secret | string | 是   | QQ 机器人应用的 App Secret       |
| allow_from | array  | 否   | 用户ID白名单，空表示允许所有用户 |

## 设置流程

1. 前往 [QQ 开放平台](https://q.qq.com/) 创建一个机器人
2. 通过仪表盘获取 App ID 和 App Secret
3. 开启机器人沙箱模式, 将用户和群添加到沙箱中
4. 将 App ID 和 App Secret 填入配置文件中
