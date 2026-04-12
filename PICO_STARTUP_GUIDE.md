# PicoClaw 启动方式指南

## 🎯 推荐方式：Web UI Launcher（桌面用户）

这是官方推荐的桌面用户使用方式：

### 方式一：双击启动（推荐）

1. 下载 PicoClaw
   - 访问 https://picoclaw.io
   - 下载对应平台的版本

2. 双击 `picoclaw-launcher.exe`（Windows）或 `picoclaw-launcher`（macOS/Linux）
   - 浏览器会自动打开 http://localhost:18800

3. 在 Web UI 中配置：
   - **Provider** 页面：配置 MiniMax API Key
   - **Channel** 页面：配置 Telegram Bot Token
   - **Gateway** 页面：启动服务

### 方式二：命令行启动 Web UI

```bash
cd 下载目录
picoclaw-launcher
# 在浏览器打开 http://localhost:18800
```

### 远程访问（Docker/虚拟机）

```bash
picoclaw-launcher -public
```

---

## 💻 命令行模式（服务器/无头环境）

### 交互式对话

```bash
picoclaw agent
# 输入消息与 AI 对话
```

### 单次消息模式

```bash
picoclaw agent -m "你好"
```

### 启动 Gateway（连接聊天平台）

```bash
picoclaw gateway
```

---

## ⚙️ 配置文件位置

### 默认配置目录

- **Windows**: `C:\Users\你的用户名\.picoclaw\`
- **macOS/Linux**: `~/.picoclaw/`

### 配置文件

- `config.json` - 主配置文件
- `workspace/` - 工作区目录

### 初始化配置

如果需要重新初始化配置：

```bash
picoclaw onboard
```

---

## 🔧 问题排查

### Web UI 中模型为空

如果 Web UI Launcher 中看不到配置的模型：

1. 检查配置文件：`C:\Users\用户名\.picoclaw\config.json`
2. 确保 `model_list` 中包含模型配置
3. 确保 `agents.defaults.model_name` 设置了默认模型

### MiniMax Token Plan 配置示例

```json
{
  "model_list": [
    {
      "model_name": "MiniMax-M2.7",
      "model": "minimax/MiniMax-M2.7",
      "api_base": "https://api.minimaxi.com/v1",
      "api_keys": ["你的Token Plan Key"]
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "MiniMax-M2.7"
    }
  }
}
```

### Telegram 连接配置

```json
{
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "你的Bot Token",
      "allow_from": ["你的User ID"],
      "proxy": "http://127.0.0.1:代理端口"
    }
  }
}
```

---

## 🚀 官方资源

- **官网**: https://picoclaw.io
- **文档**: https://docs.picoclaw.io
- **GitHub**: https://github.com/sipeed/picoclaw

---

## 💡 最佳实践

1. **桌面用户**: 使用 Web UI Launcher
2. **服务器/无头环境**: 使用命令行模式
3. **开发测试**: 使用 `picoclaw agent -m` 单次测试
4. **生产环境**: 使用 `picoclaw gateway` 后台运行

---

## 🔒 安全提示

- 禁用 `deny_patterns` 后，PicoClaw 可以执行任何命令
- 确保 Telegram Bot 的 `allow_from` 设置了你的 User ID
- 妥善保管 API Key，不要泄露
