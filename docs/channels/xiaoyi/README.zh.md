# 小艺 (XiaoYi)

小艺 Channel 通过华为小艺 A2A (Agent-to-Agent) 协议实现智能体通信，支持文本消息、文件附件、流式响应和状态更新。

## 配置

```json
{
  "channels": {
    "xiaoyi": {
      "enabled": true,
      "ak": "your-access-key",
      "sk": "your-secret-key",
      "agent_id": "your-agent-id",
      "ws_url1": "",
      "ws_url2": "",
      "allow_from": []
    }
  }
}
```

| 字段       | 类型   | 必填 | 描述                                         |
| ---------- | ------ | ---- | -------------------------------------------- |
| enabled    | bool   | 是   | 是否启用小艺频道                             |
| ak         | string | 是   | Access Key                                   |
| sk         | string | 是   | Secret Key                                   |
| agent_id   | string | 是   | Agent 标识                                   |
| ws_url1    | string | 否   | 服务器1 URL，默认为小艺官方服务器            |
| ws_url2    | string | 否   | 服务器2 URL，默认为小艺备用服务器            |
| allow_from | array  | 否   | 用户ID白名单，空表示允许所有用户             |

## 设置流程

详细教程请参考华为官方文档：[OpenClaw 接入指南](https://developer.huawei.com/consumer/cn/doc/service/openclaw-0000002518410344)

1. 在华为开发者平台注册并创建 OpenClaw 类型的智能体
2. 获取 AK (Access Key) 和 SK (Secret Key)
3. 获取 Agent ID
4. 将配置填入配置文件中
5. 启动 PicoClaw，小艺 Channel 将自动连接到小艺服务器

## 特性

- **WebSocket 长连接**：支持双服务器热备
- **自动重连**：指数退避策略，最大重试 50 次
- **心跳机制**：协议层 + 应用层双重心跳
- **流式响应**：支持逐步返回结果
- **状态更新**：收到消息后立即发送"处理中"状态
