# 企业微信自建应用 (WeCom App) 配置指南

本文档介绍如何在 PicoClaw 中配置企业微信自建应用 (wecom-app) 通道。

## 功能特性

| 功能 | 支持状态 |
|------|---------|
| 被动接收消息 | ✅ |
| 主动发送消息 | ✅ |
| 私聊 | ✅ |
| 群聊 | ❌ |

## 配置步骤

### 1. 企业微信后台配置

1. 登录 [企业微信管理后台](https://work.weixin.qq.com/wework_admin)
2. 进入"应用管理" → 选择自建应用
3. 记录以下信息：
   - **AgentId**: 应用详情页显示
   - **Secret**: 点击"查看"获取
4. 进入"我的企业"页面，记录 **企业ID** (CorpID)

### 2. 接收消息配置

1. 在应用详情页，点击"接收消息"的"设置API接收"
2. 填写以下信息：
   - **URL**: `http://your-server:18792/webhook/wecom-app`
   - **Token**: 随机生成或自定义（用于签名验证）
   - **EncodingAESKey**: 点击"随机生成"生成43字符的密钥
3. 点击"保存"时，企业微信会发送验证请求

### 3. PicoClaw 配置

在 `config.json` 中添加以下配置：

```json
{
  "channels": {
    "wecom_app": {
      "enabled": true,
      "corp_id": "wwxxxxxxxxxxxxxxxx",           // 企业ID
      "corp_secret": "xxxxxxxxxxxxxxxxxxxxxxxx", // 应用Secret
      "agent_id": 1000002,                        // 应用AgentId
      "token": "your_token",                      // 接收消息配置的Token
      "encoding_aes_key": "your_encoding_aes_key", // 接收消息配置的EncodingAESKey
      "webhook_host": "0.0.0.0",
      "webhook_port": 18792,
      "webhook_path": "/webhook/wecom-app",
      "allow_from": [],
      "reply_timeout": 5
    }
  }
}
```

## 常见问题

### 1. 回调URL验证失败

**症状**: 企业微信保存API接收消息时提示验证失败

**检查项**:
- 确认服务器防火墙已开放 18792 端口
- 确认 `corp_id`、`token`、`encoding_aes_key` 配置正确
- 查看 PicoClaw 日志是否有请求到达

### 2. 中文消息解密失败

**症状**: 发送中文消息时出现 `invalid padding size` 错误

**原因**: 企业微信使用非标准的 PKCS7 填充（32字节块大小）

**解决**: 确保使用最新版本的 PicoClaw，已修复此问题。

### 3. 端口冲突

**症状**: 启动时提示端口已被占用

**解决**: 修改 `webhook_port` 为其他端口，如 18794

## 技术细节

### 加密算法

- **算法**: AES-256-CBC
- **密钥**: EncodingAESKey Base64解码后的32字节
- **IV**: AESKey的前16字节
- **填充**: PKCS7（块大小为32字节，非标准16字节）
- **消息格式**: XML

### 消息结构

解密后的消息格式：
```
random(16B) + msg_len(4B) + msg + receiveid
```

其中 `receiveid` 对于自建应用是 `corp_id`。

## 调试

启用调试模式查看详细日志：

```bash
picoclaw gateway --debug
```

关键日志标识：
- `wecom_app`: WeCom App 通道相关日志
- `wecom_common`: 加密解密相关日志

## 参考文档

- [企业微信官方文档 - 接收消息](https://developer.work.weixin.qq.com/document/path/96211)
- [企业微信官方加解密库](https://github.com/sbzhu/weworkapi_golang)
