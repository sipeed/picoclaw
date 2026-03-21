> 返回 [README](../../README.zh.md)

# 在 PicoClaw 中使用 Antigravity 提供商

本指南介绍如何在 PicoClaw 中设置和使用 **Antigravity**（Google Cloud Code Assist）提供商。

## 前提条件

1.  一个 Google 账户。
2.  已启用 Google Cloud Code Assist（通常通过"Gemini for Google Cloud"引导流程获取）。

## 1. 身份验证

要使用 Antigravity 进行身份验证，请运行以下命令：

```bash
picoclaw auth login --provider antigravity
```

### 手动验证（无界面/VPS 环境）
如果你在服务器（Coolify/Docker）上运行且无法访问 `localhost`，请按照以下步骤操作：
1.  运行上述命令。
2.  复制提供的 URL 并在本地浏览器中打开。
3.  完成登录。
4.  浏览器将重定向到 `localhost:51121` URL（页面将无法加载）。
5.  **从浏览器地址栏复制该最终 URL**。
6.  **将其粘贴回 PicoClaw 正在等待的终端中**。

PicoClaw 将自动提取授权码并完成流程。

## 2. 管理模型

### 列出可用模型
查看你的项目可以访问哪些模型并检查其配额：

```bash
picoclaw auth models
```

### 切换模型
你可以在 `~/.picoclaw/config.json` 中更改默认模型，或通过 CLI 覆盖：

```bash
# 为单个命令覆盖
picoclaw agent -m "Hello" --model claude-opus-4-6-thinking
```

## 3. 实际使用（Coolify/Docker）

如果你通过 Coolify 或 Docker 部署，请按照以下步骤进行测试：

1.  **环境变量**：
    *   `PICOCLAW_AGENTS_DEFAULTS_MODEL=gemini-flash`
2.  **身份验证持久化**：
    如果你已在本地登录，可以将凭据复制到服务器：
    ```bash
    scp ~/.picoclaw/auth.json user@your-server:~/.picoclaw/
    ```
    *或者*，如果你有终端访问权限，可以在服务器上运行一次 `auth login` 命令。

## 4. 故障排除

*   **空响应**：如果模型返回空回复，可能是该模型在你的项目中受到限制。请尝试 `gemini-3-flash` 或 `claude-opus-4-6-thinking`。
*   **429 速率限制**：Antigravity 有严格的配额限制。如果触发限制，PicoClaw 将在错误消息中显示"重置时间"。
*   **404 未找到**：确保你使用的是 `picoclaw auth models` 列表中的模型 ID。请使用短 ID（例如 `gemini-3-flash`），而非完整路径。

## 5. 可用模型总结

根据测试，以下模型最为可靠：
*   `gemini-3-flash`（快速，高可用性）
*   `gemini-2.5-flash-lite`（轻量级）
*   `claude-opus-4-6-thinking`（强大，包含推理能力）
