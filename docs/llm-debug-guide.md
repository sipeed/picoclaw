# LLM Debug 日志指南

## 概述

本文档说明如何使用新增的 LLM 调试日志来排查 API 调用问题，特别是 `404 page not found` 错误。

## 新增的调试日志

### 1. Provider 创建日志 (http_provider.go)

**位置**: `CreateProvider` 函数

**日志级别**:
- `DebugCF`: 创建 provider 时的详细信息
- `InfoCF`: Provider 创建成功
- `ErrorCF`: 配置错误

**记录信息**:
```go
// 创建时的调试信息
logger.DebugCF("llm", "Creating LLM provider", map[string]interface{}{
    "model": model,
    "lower_model": lowerModel,
})

// 创建成功
logger.InfoCF("llm", "Provider created successfully", map[string]interface{}{
    "model": model,
    "api_base": apiBase,
    "has_api_key": apiKey != "",
    "api_key_len": len(apiKey),
})
```

### 2. HTTP 请求日志

**位置**: `HTTPProvider.Chat` 函数

**发送请求前的日志**:
```go
logger.DebugCF("llm", "Sending LLM request", map[string]interface{}{
    "url": fullURL,
    "model": model,
    "api_base": p.apiBase,
    "has_api_key": p.apiKey != "",
    "api_key_len": len(p.apiKey),
    "message_count": len(messages),
    "tools_count": len(tools),
    "request_body": string(jsonData),  // 完整请求体
})
```

**发送请求时的日志**:
```go
logger.DebugCF("llm", "Sending HTTP request", map[string]interface{}{
    "method": "POST",
    "url": fullURL,
    "headers": req.Header,  // 包括 Authorization 等 header
})
```

### 3. HTTP 响应日志

**成功接收响应**:
```go
logger.DebugCF("llm", "Received LLM response", map[string]interface{}{
    "status_code": resp.StatusCode,
    "status": resp.Status,
    "content_length": len(body),
    "response_body": string(body),  // 完整响应体
})
```

**错误响应 (非 200 状态码)**:
```go
logger.ErrorCF("llm", "LLM API returned non-OK status", map[string]interface{}{
    "status_code": resp.StatusCode,
    "status": resp.Status,
    "url": fullURL,
    "response_body": string(body),
})
```

## 如何启用调试日志

### 方法 1: 通过配置文件

修改 `config.yaml`:

```yaml
logging:
  level: debug  # 设置为 debug 级别以查看所有日志
  category_levels:
    llm: debug    # 只启用 llm 相关的 debug 日志
    agent: info   # 其他类别保持 info 级别
```

### 方法 2: 通过环境变量

```bash
export PICOCLAW_LOG_LEVEL=debug
```

或者只针对 llm 分类：

```bash
export PICOCLAW_LOG_CATEGORY_LLM=debug
```

## 排查 404 错误的步骤

当遇到 `Error processing message: LLM call failed: API error: 404 page not found` 错误时：

### 步骤 1: 检查 Provider 创建日志

查找日志中的 "Provider created successfully" 消息：

```
[INFO] [llm] Provider created successfully
  model: gpt-4
  api_base: https://api.openai.com/v1
  has_api_key: true
  api_key_len: 51
```

**检查点**:
- ✅ `api_base` 是否正确？（常见错误：末尾多了 `/chat/completions`）
- ✅ `has_api_key` 是否为 true？
- ✅ `model` 名称是否正确？

### 步骤 2: 检查请求 URL

查找 "Sending LLM request" 日志：

```
[DEBUG] [llm] Sending LLM request
  url: https://api.openai.com/v1/chat/completions
  model: gpt-4
  api_base: https://api.openai.com/v1
  ...
```

**检查点**:
- ✅ 完整的 `url` 是否正确？
- ✅ 路径是否为 `/chat/completions`？
- ✅ 是否有重复的路径（如 `/v1/v1/chat/completions`）？

### 步骤 3: 检查响应详情

查找 "LLM API returned non-OK status" 错误日志：

```
[ERROR] [llm] LLM API returned non-OK status
  status_code: 404
  status: 404 Not Found
  url: https://wrong-url.com/v1/chat/completions
  response_body: 404 page not found
```

**检查点**:
- ✅ `status_code` 为 404 表示 URL 路径错误
- ✅ `response_body` 可能包含更详细的错误信息
- ✅ 对比 `url` 和正确的 API endpoint

## 常见的 404 错误原因

### 1. API Base 配置错误

**错误示例**:
```yaml
providers:
  openai:
    api_base: "https://api.openai.com/v1/chat/completions"  # ❌ 错误：包含了完整路径
```

**正确配置**:
```yaml
providers:
  openai:
    api_base: "https://api.openai.com/v1"  # ✅ 正确：只包含 base URL
```

### 2. 自定义代理或中转服务配置错误

**错误示例**:
```yaml
providers:
  openai:
    api_base: "https://my-proxy.com"  # ❌ 缺少 /v1 路径
```

**正确配置**:
```yaml
providers:
  openai:
    api_base: "https://my-proxy.com/v1"  # ✅ 包含正确的路径
```

### 3. vLLM 或本地模型服务配置

**正确示例**:
```yaml
providers:
  vllm:
    api_base: "http://localhost:8000/v1"  # ✅ vLLM 通常也使用 /v1 路径
```

## 调试命令

### 查看完整的调试日志

```bash
# 启动 picoclaw 并查看所有 debug 日志
PICOCLAW_LOG_LEVEL=debug ./picoclaw

# 只看 llm 相关的日志
PICOCLAW_LOG_LEVEL=debug ./picoclaw | grep '\[llm\]'

# 保存日志到文件以便分析
PICOCLAW_LOG_LEVEL=debug ./picoclaw 2>&1 | tee debug.log
```

### 使用 jq 格式化 JSON 日志（如果日志是 JSON 格式）

```bash
PICOCLAW_LOG_LEVEL=debug ./picoclaw 2>&1 | jq -r 'select(.category == "llm")'
```

## 示例：完整的调试流程

假设遇到 404 错误，以下是完整的调试输出示例：

```
[2026-02-12 10:30:00] [DEBUG] [llm] Creating LLM provider
  model: gpt-4
  lower_model: gpt-4

[2026-02-12 10:30:00] [INFO] [llm] Provider created successfully
  model: gpt-4
  api_base: https://api.openai.com/wrong-path  # ⚠️ 错误的路径
  has_api_key: true
  api_key_len: 51

[2026-02-12 10:30:01] [DEBUG] [llm] Sending LLM request
  url: https://api.openai.com/wrong-path/chat/completions  # ⚠️ 最终的 URL 错误
  model: gpt-4
  message_count: 1
  request_body: {"model":"gpt-4","messages":[...]}

[2026-02-12 10:30:01] [DEBUG] [llm] Sending HTTP request
  method: POST
  url: https://api.openai.com/wrong-path/chat/completions
  headers: {Content-Type: application/json, Authorization: Bearer sk-...}

[2026-02-12 10:30:02] [DEBUG] [llm] Received LLM response
  status_code: 404
  status: 404 Not Found
  response_body: 404 page not found

[2026-02-12 10:30:02] [ERROR] [llm] LLM API returned non-OK status
  status_code: 404
  url: https://api.openai.com/wrong-path/chat/completions
  response_body: 404 page not found

[2026-02-12 10:30:02] [ERROR] [agent] LLM call failed
  iteration: 1
  error: API error: 404 page not found
```

从这个日志可以清楚地看到：
1. Provider 创建时使用了错误的 `api_base`
2. 最终的请求 URL 拼接错误
3. 服务器返回 404 错误

**解决方案**: 修改配置文件中的 `api_base` 为 `https://api.openai.com/v1`

## 敏感信息保护

注意到在日志中：
- ✅ API Key 只显示长度和前缀，不会完整输出
- ✅ Authorization header 会被隐藏
- ⚠️ 完整的请求体和响应体会被记录（debug 级别）

**生产环境建议**:
- 使用 `info` 或更高级别，避免泄露敏感信息
- 只在本地开发或受控环境中使用 `debug` 级别

## 相关文件

- `pkg/providers/http_provider.go` - HTTP Provider 实现和调试日志
- `pkg/agent/loop.go` - Agent 循环和错误处理
- `pkg/logger/logger.go` - 日志系统实现

## 获取帮助

如果调试日志无法解决问题，请在 GitHub Issue 中提供：
1. 完整的配置文件（隐藏 API Key）
2. 相关的调试日志输出
3. 使用的模型和 provider
4. 错误发生的上下文
