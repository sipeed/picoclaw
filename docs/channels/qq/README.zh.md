> 返回 [README](../../../README.zh.md)

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
      "allow_from": [],
      "max_base64_file_size_mib": 0
    }
  }
}
```

| 字段                 | 类型   | 必填 | 描述                                                         |
| -------------------- | ------ | ---- | ------------------------------------------------------------ |
| enabled              | bool   | 是   | 是否启用 QQ Channel                                          |
| app_id               | string | 是   | QQ 机器人应用的 App ID                                       |
| app_secret           | string | 是   | QQ 机器人应用的 App Secret                                   |
| allow_from           | array  | 否   | 用户ID白名单，空表示允许所有用户                             |
| max_base64_file_size_mib | int | 否   | 本地文件转 base64 上传的最大体积，单位 MiB；`0` 表示不限制。仅影响本地文件，不影响 URL 直传 |

## 设置流程

### 快捷方式（推荐）

QQ 开放平台提供了一键创建入口：

1. 打开 [QQ 机器人快速创建](https://q.qq.com/qqbot/openclaw/index.html)，扫码登录
2. 系统自动创建机器人，复制 **App ID** 和 **App Secret**
3. 将凭证填入 PicoClaw 配置文件
4. 运行 `picoclaw gateway` 启动服务
5. 打开 QQ，与机器人开始对话

> App Secret 仅显示一次，请立即保存。再次查看将强制重置。
>
> 通过快捷入口创建的机器人仅供创建人使用，暂不支持群聊。如需群聊功能，请在 [QQ 开放平台](https://q.qq.com/) 配置沙箱模式。

### 手动创建

1. 使用 QQ 账号登录 [QQ 开放平台](https://q.qq.com/)，注册开发者账号
2. 创建 QQ 机器人，自定义头像和名称
3. 在机器人设置中获取 **App ID** 和 **App Secret**
4. 将凭证填入 PicoClaw 配置文件
5. 运行 `picoclaw gateway` 启动服务
6. 在 QQ 中搜索你的机器人，开始对话

> 开发阶段建议开启沙箱模式，将测试用户和群添加到沙箱中进行调试。
