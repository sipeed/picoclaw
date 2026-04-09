# 安全配置

## 概述

PicoClaw 支持通过将敏感数据（API 密钥、令牌、密钥、密码）与主配置分离存储在 `.security.yml` 文件中来提高安全性：

1. **关注点分离**：配置设置和密钥在不同的文件中
2. **更易于共享**：主配置可以共享而不会暴露敏感数据
3. **更好的版本控制**：`.security.yml` 应添加到 `.gitignore`
4. **灵活的部署**：不同环境可以使用不同的安全文件

## 文件结构

```
~/.picoclaw/
├── config.json          # 主配置（可安全共享）
└── .security.yml        # 安全数据（永不共享）
```

## 工作原理

安全配置通过**直接字段映射**工作，而不是通过 `ref:` 字符串引用。系统自动从 `.security.yml` 加载值并将其应用到 `config.json` 中的相应字段。

### 关键点：

- `.security.yml` 中的值会自动映射到配置中的相应字段
- 映射基于字段名称和结构，而不是引用字符串
- 如果 `.security.yml` 中存在某个值，它**会覆盖** `config.json` 中的值
- 您可以完全从 `config.json` 中省略敏感字段（推荐）

## 安全配置结构

### 完整示例：.security.yml

```yaml
# 模型 API 密钥
# 所有模型必须使用 `api_keys`（复数）数组格式
# 即使是单个密钥也必须作为包含一个元素的数组提供
model_list:
  gpt-5.4:
    api_keys:
      - "sk-proj-your-actual-openai-key-1"
      - "sk-proj-your-actual-openai-key-2"  # 可选：用于故障转移的多个密钥
  claude-sonnet-4.6:
    api_keys:
      - "sk-ant-your-actual-anthropic-key"  # 数组格式中的单个密钥

# 渠道令牌
channels:
  telegram:
    token: "your-telegram-bot-token"
  feishu:
    app_secret: "your-feishu-app-secret"
    encrypt_key: "your-feishu-encrypt-key"
    verification_token: "your-feishu-verification-token"
  discord:
    token: "your-discord-bot-token"
  weixin:
    token: "your-weixin-token"
  qq:
    app_secret: "your-qq-app-secret"
  dingtalk:
    client_secret: "your-dingtalk-client-secret"
  slack:
    bot_token: "your-slack-bot-token"
    app_token: "your-slack-app-token"
  matrix:
    access_token: "your-matrix-access-token"
  line:
    channel_secret: "your-line-channel-secret"
    channel_access_token: "your-line-channel-access-token"
  onebot:
    access_token: "your-onebot-access-token"
  wecom:
    token: "your-wecom-token"
    encoding_aes_key: "your-wecom-encoding-aes-key"
  wecom_app:
    corp_secret: "your-wecom-app-corp-secret"
    token: "your-wecom-app-token"
    encoding_aes_key: "your-wecom-app-encoding-aes-key"
  wecom_aibot:
    secret: "your-wecom-aibot-secret"
    token: "your-wecom-aibot-token"
    encoding_aes_key: "your-wecom-aibot-encoding-aes-key"
  pico:
    token: "your-pico-token"
  irc:
    password: "your-irc-password"
    nickserv_password: "your-irc-nickserv-password"
    sasl_password: "your-irc-sasl-password"

# Web 工具 API 密钥
web:
  brave:
    api_keys:
      - "BSAyour-brave-api-key-1"
      - "BSAyour-brave-api-key-2"  # 可选：用于故障转移的多个密钥
  tavily:
    api_keys:
      - "tvly-your-tavily-api-key"  # 数组格式中的单个密钥
  perplexity:
    api_keys:
      - "pplx-your-perplexity-api-key"  # 数组格式中的单个密钥
  glm_search:
    api_key: "your-glm-search-api-key"  # GLMSearch 使用单个密钥格式（不是数组）
  baidu_search:
    api_key: "your-baidu-search-api-key"

# 技能注册表令牌
skills:
  github:
    token: "your-github-token"
  clawhub:
    auth_token: "your-clawhub-auth-token"
```

## 使用方法

### 步骤 1：创建 .security.yml

创建或复制安全文件：
```bash
cp security.example.yml ~/.picoclaw/.security.yml
```

### 步骤 2：填写实际值

编辑 `~/.picoclaw/.security.yml`，用您实际的 API 密钥和令牌替换占位符值。

### 步骤 3：设置正确的权限

```bash
chmod 600 ~/.picoclaw/.security.yml
```

### 步骤 4：简化 config.json（推荐）

现在您可以从 `config.json` 中删除敏感字段，因为它们是从 `.security.yml` 加载的：

**之前：**
```json
{
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api.openai.com/v1",
      "api_key": "sk-your-actual-api-key-here"
    }
  ],
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"
    }
  }
}
```

**之后：**
```json
{
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_base": "https://api.openai.com/v1"
      // api_key 现在从 .security.yml 加载
    }
  ],
  "channels": {
    "telegram": {
      "enabled": true
      // token 现在从 .security.yml 加载
    }
  }
}
```

### 步骤 5：验证

重启 PicoClaw 并验证它正确加载：
```bash
picoclaw --version
```

## 字段映射规则

### 模型

**在 .security.yml 中：**
```yaml
model_list:
  <model_name>:
    api_keys:
      - "key-1"
      - "key-2"
```

**映射：**
- 字段 `api_keys`（数组）映射到模型的 API 密钥
- `<model_name>` 必须与 `config.json` 中的 `model_name` 字段匹配
- 支持索引名称（如 "gpt-5.4:0"）— 系统也会尝试基础名称（"gpt-5.4"）

### 渠道

每个渠道直接映射其字段：

**在 .security.yml 中：**
```yaml
channels:
  telegram:
    token: "value"
  feishu:
    app_secret: "value"
    encrypt_key: "value"
    verification_token: "value"
  discord:
    token: "value"
```

**映射：**
- `channels.telegram.token` → `config.channels.telegram.token`
- `channels.feishu.app_secret` → `config.channels.feishu.app_secret`
- 等等。

### Web 工具

**Brave、Tavily、Perplexity：**
```yaml
web:
  brave:
    api_keys:
      - "key-1"
      - "key-2"
```
- 使用 `api_keys`（复数）数组格式

**GLMSearch：**
```yaml
web:
  glm_search:
    api_key: "single-key-here"
```
- 使用 `api_key`（单数）单个字符串格式

**BaiduSearch：**
```yaml
web:
  baidu_search:
    api_key: "your-key"
```
- 使用 `api_key`（单数）单个字符串格式

### 技能

**在 .security.yml 中：**
```yaml
skills:
  github:
    token: "value"
  clawhub:
    auth_token: "value"
```

## API 密钥格式

### 模型 - 单个密钥

使用包含一个元素的数组格式：
```yaml
model_list:
  gpt-5.4:
    api_keys:
      - "sk-your-key"
```

### 模型 - 多个密钥（负载均衡与故障转移）

使用包含多个元素的数组格式：
```yaml
model_list:
  gpt-5.4:
    api_keys:
      - "sk-your-key-1"
      - "sk-your-key-2"
      - "sk-your-key-3"
```

**好处：**
- **负载均衡**：请求分布在多个密钥之间
- **故障转移**：如果一个密钥失败，自动切换到另一个密钥
- **速率限制管理**：在多个密钥之间分配使用
- **高可用性**：减少 API 提供商问题期间的停机时间

### Web 工具（Brave/Tavily/Perplexity）- 单个密钥

```yaml
web:
  brave:
    api_keys:
      - "BSA-your-key"
```

### Web 工具（Brave/Tavily/Perplexity）- 多个密钥

```yaml
web:
  brave:
    api_keys:
      - "BSA-key-1"
      - "BSA-key-2"
```

### Web 工具（GLMSearch/BaiduSearch）- 仅限单个密钥

```yaml
web:
  glm_search:
    api_key: "your-glm-key"  # 单个字符串（不是数组）
  baidu_search:
    api_key: "your-baidu-key"  # 单个字符串（不是数组）
```

## 模型名称匹配

系统支持 `.security.yml` 中智能模型名称匹配：

### 示例 1：精确匹配

**config.json：**
```json
{
  "model_name": "gpt-5.4:0"
}
```

**.security.yml（带索引的精确匹配）：**
```yaml
model_list:
  gpt-5.4:0:
    api_keys: ["key-1"]
```

### 示例 2：基础名称匹配

**config.json：**
```json
{
  "model_name": "gpt-5.4:0"
}
```

**.security.yml（不带索引的基础名称）：**
```yaml
model_list:
  gpt-5.4:
    api_keys: ["key-1", "key-2"]
```

两种方法都有效。基础名称匹配允许您在配置使用索引模型名称进行负载均衡时在 `.security.yml` 中使用更简单的密钥。

## 向后兼容

系统保持完全向后兼容：

1. **直接值**：您仍可以在 `config.json` 中使用直接值（不推荐用于生产环境）
2. **混合使用**：您可以在 `.security.yml` 和 `config.json` 中都有某些字段
3. **可选安全文件**：如果 `.security.yml` 不存在，系统将仅使用 `config.json` 中的值
4. **覆盖行为**：如果两个文件都存在某字段，`.security.yml` 的值优先

## 环境变量

您可以使用环境变量覆盖任何安全值：

**对于模型：**
```bash
export PICOCLAW_CHANNELS_TELEGRAM_TOKEN="token-from-env"
```

**对于渠道：**
```bash
export PICOCLAW_CHANNELS_TELEGRAM_TOKEN="token-from-env"
export PICOCLAW_CHANNELS_FEISHU_APP_SECRET="secret-from-env"
```

**对于 Web 工具：**
```bash
export PICOCLAW_TOOLS_WEB_BRAVE_API_KEY="key-from-env"
export PICOCLAW_TOOLS_WEB_BAIDU_API_KEY="baidu-key-from-env"
```

环境变量具有最高优先级，会覆盖 `config.json` 和 `.security.yml` 的值。

格式为：`PICOCLAW_<SECTION>_<KEY>_<FIELD>`，用下划线分隔路径段并转换为大写。

## 安全最佳实践

1. **永不提交 `.security.yml`** 到版本控制
2. **添加到 .gitignore**：确保 `.security.yml` 在您的 `.gitignore` 文件中
3. **设置文件权限**：`chmod 600 ~/.picoclaw/.security.yml`
4. **不同环境使用不同密钥**（开发、暂存、生产）
5. **定期轮换密钥**并更新 `.security.yml`
6. **安全备份**：加密包含 `.security.yml` 的备份。请注意，配置迁移会自动创建带日期戳的备份（如 `config.json.20260330.bak` 和 `.security.yml.20260330.bak`）
7. **审查访问权限**：确保只有授权用户能读取该文件

## API

### loadSecurityConfig

```go
func loadSecurityConfig(securityPath string) (*SecurityConfig, error)
```

从 `.security.yml` 加载安全配置。如果文件不存在，返回空的 `SecurityConfig`。

### saveSecurityConfig

```go
func saveSecurityConfig(securityPath string, sec *SecurityConfig) error
```

以 `0o600` 权限将安全配置保存到 `.security.yml`。

### applySecurityConfig

```go
func applySecurityConfig(cfg *Config, sec *SecurityConfig) error
```

通过将值从 `.security.yml` 复制到配置的相应字段来应用安全配置。

### securityPath

```go
func securityPath(configPath string) string
```

返回配置文件中 `.security.yml` 的相对路径。

## 测试

运行安全配置测试：

```bash
go test ./pkg/config -run TestSecurityConfig
```

## 故障排除

### 错误："failed to load security config"

- 验证 `.security.yml` 存在于 `config.json` 的同一目录中
- 检查 YAML 语法是否有效（使用 YAML 验证器）
- 确保文件权限允许读取

### 错误："model security entry not found"

- 确保 `config.json` 中的模型名称与 `.security.yml` 中的完全匹配
- 检查 `.security.yml` 中存在 `model_list` 部分
- 对于带索引名称的模型（如 "gpt-5.4:0"），确保使用确切的名称或不带索引的基础名称
- 验证 YAML 结构正确（正确的缩进）

### 多个 API 密钥不工作

- 确保对模型和 Web 工具使用 `api_keys`（复数）（GLMSearch/BaiduSearch 除外）
- 检查数组格式在 YAML 中正确（破折号的正确缩进）
- 记住：模型、Brave、Tavily、Perplexity 必须使用 `api_keys`（数组格式）
- GLMSearch 和 BaiduSearch 必须使用 `api_key`（单个字符串格式）

### 负载均衡/故障转移问题

- 验证 `api_keys` 数组中的所有 API 密钥都有效
- 检查所有密钥具有相同的速率限制和权限
- 监控日志以查看正在使用哪些密钥以及哪些失败
- 确保 `api_keys` 数组在 YAML 中格式正确

### 密钥未被应用

- 检查 `.security.yml` 与 `config.json` 在同一目录中
- 验证文件权限允许读取（`chmod 600 ~/.picoclaw/.security.yml`）
- 确保 YAML 结构与预期格式匹配
- 检查字段名称中的拼写错误（区分大小写）
- 验证模型/渠道名称完全匹配（区分大小写）

## 迁移指南

### 步骤 1：备份您的配置

系统在保存迁移后的配置前会自动创建带日期戳的备份（如 `config.json.20260330.bak` 和 `.security.yml.20260330.bak`）。如果您更喜欢手动备份：

```bash
cp ~/.picoclaw/config.json ~/.picoclaw/config.json.backup
```

### 步骤 2：创建 .security.yml

```bash
cp security.example.yml ~/.picoclaw/.security.yml
```

### 步骤 3：填写您的 API 密钥

编辑 `~/.picoclaw/.security.yml`，用您实际的密钥替换占位符值。

### 步骤 4：从 config.json 中删除敏感字段

从 `config.json` 中删除或注释敏感字段：
- `model_list` 条目中的 `api_key` 字段
- `channels` 中的 `token` 字段
- `tools.web` 中的 `api_key` 字段
- `tools.skills` 中的 `token`/`auth_token` 字段

### 步骤 5：设置正确的权限

```bash
chmod 600 ~/.picoclaw/.security.yml
```

### 步骤 6：测试

```bash
picoclaw --version
```

### 步骤 7：验证功能

测试您的模型和渠道以确保一切正常工作。

### 步骤 8：清理（可选）

如果一切正常，您可以删除备份：
```bash
rm ~/.picoclaw/config.json.backup
# 也可以删除自动生成的带日期戳的备份：
rm ~/.picoclaw/config.json.20*.bak ~/.picoclaw/.security.yml.20*.bak
```

## 高级：加密 API 密钥

PicoClaw 支持加密安全文件中的 API 密钥以提供额外保护。

### 设置

1. 通过环境变量设置密码短语：
```bash
export PICOCLAW_CREDENTIAL_PASSPHRASE="your-secure-passphrase"
```

2. 保存配置时，API 密钥将自动加密：
```go
SaveConfig(path, config)
```

### 加密格式

加密密钥存储为：
```yaml
model_list:
  gpt-5.4:
    api_keys:
      - "enc://encrypted-base64-string"
```

系统在加载配置时会在运行时自动解密密钥。

### 好处

- 额外的安全层
- 密钥静态加密
- 密码短语可以与配置文件分开管理

### 重要说明

- 始终安全地备份您的密码短语
- 如果丢失密码短语，您将失去对加密密钥的访问权限
- 使用强且唯一的密码短语
- 永不将密码短语提交到版本控制
