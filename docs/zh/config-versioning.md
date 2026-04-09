# 配置文件版本控制指南

## 概述

PicoClaw 使用模式版本控制系统来管理 `config.json`，确保在配置格式演进时能够顺利升级。

## 版本历史

### 版本 1
- **引入时间**：初始版本，支持 version 字段
- **变更**：向 Config 结构体添加了 `version` 字段
- **迁移**：现有配置无需结构性变更

### 版本 2
- **引入时间**：模型启用/禁用支持和渠道配置统一
- **变更**：
  - 向 `ModelConfig` 添加了 `enabled` 字段 — 允许禁用单个模型条目而无需删除
  - 在 V1→V2 迁移期间，`enabled` 会自动推断：带有 API 密钥或保留名称 `local-model` 的模型会被启用；其他模型默认禁用
  - 迁移了旧版渠道字段：Discord `mention_only` → `group_trigger.mention_only`，OneBot `group_trigger_prefix` → `group_trigger.prefixes`
  - V0 配置现在直接迁移到 CurrentVersion（V2），而不是经过 V1
  - `makeBackup()` 现在使用仅日期后缀（如 `config.json.20260330.bak`），同时也会备份 `.security.yml`

## 工作原理

### 自动迁移
加载配置文件时：
1. 系统首先从 JSON 中读取 `version` 字段
2. 根据检测到的版本，加载相应的配置结构体（`configV0`、`configV1` 等）
3. 如果加载的版本低于最新版本，则增量应用迁移
4. 保存前，系统会自动创建 `config.json` 和 `.security.yml` 的带日期戳备份
5. 版本号会自动更新
6. 迁移后的配置会自动保存到磁盘

### 版本字段
`config.json` 中的 `version` 字段表示模式版本：
- `0` 或缺失：旧版配置（无 version 字段）
- `1`：上一版本（加载时会自动迁移到 V2）
- `2`：当前版本

```json
{
  "version": 2,
  "agents": {...},
  ...
}
```

## 添加新的迁移

对配置模式进行破坏性变更时：

### 步骤 1：定义新版本结构体

如果结构发生重大变更，创建新版本的结构体：

```go
// ConfigV2 代表版本 2 的配置结构
type ConfigV2 struct {
    Version   int             `json:"version"`
    Agents    AgentsConfig    `json:"agents"`
    // ... 其他字段，新结构
}
```

### 步骤 2：更新当前配置版本

```go
const CurrentVersion = 2  // 递增此值
```

### 步骤 3：添加加载器函数

```go
// loadConfigV3 加载版本 3 的配置
func loadConfigV3(data []byte) (*Config, error) {
    cfg := DefaultConfig()

    // 解析为 ConfigV3 结构体
    var v3 ConfigV3
    if err := json.Unmarshal(data, &v3); err != nil {
        return nil, err
    }

    // 转换为当前 Config
    cfg.Version = v3.Version
    cfg.Agents = v3.Agents
    // ... 映射其他字段

    return cfg, nil
}
```

### 步骤 4：添加迁移逻辑

```go
func (c *configV2) Migrate() (*Config, error) {
    // 在此处应用 V2→V3 的结构性变更
    migrated := &c.Config
    migrated.Version = 3
    // 应用结构性变更
    return migrated, nil
}
```

### 步骤 5：更新 LoadConfig 开关

```go
func LoadConfig(path string) (*Config, error) {
    // ... 读取文件 ...

    switch versionInfo.Version {
    case 0:
        cfg, err = loadConfigV0(data)
    case 1:
        cfg, err = loadConfigV1(data)
    case 2:
        cfg, err = loadConfig(data)
    case 3:
        cfg, err = loadConfigV3(data)
    default:
        return nil, fmt.Errorf("unsupported config version: %d", versionInfo.Version)
    }

    // ... 迁移和验证 ...
}
```

### 步骤 6：测试迁移

在 `config_migration_test.go` 中创建测试：

```go
func TestMigrateV2ToV3(t *testing.T) {
    // 创建版本 2 的配置
    v2Config := Config{
        Version: 2,
        // ... 设置测试数据
    }

    // 应用迁移
    migrated, err := v2Config.Migrate()
    if err != nil {
        t.Fatalf("Migration failed: %v", err)
    }

    // 验证版本已更新
    if migrated.Version != 3 {
        t.Errorf("Expected version 3, got %d", migrated.Version)
    }

    // 验证数据正确保留/转换
    // ...
}
```

## 迁移最佳实践

1. **版本特定结构体**：为每个有结构性变更的版本定义单独的结构体
2. **向后兼容**：确保旧配置仍能用其特定结构体加载
3. **无数据丢失**：迁移应保留所有用户设置
4. **幂等性**：多次运行相同迁移应该是安全的
5. **自动保存**：迁移后的配置会自动保存以更新用户的文件
6. **自动备份**：保存前，系统会创建 `config.json` 和 `.security.yml` 的带日期戳备份
7. **全面测试**：使用真实用户配置文件进行测试
8. **更新默认值**：使 `defaults.go` 与最新模式保持同步

## 迁移示例

### 场景：添加带默认值的新字段

旧配置（版本 2）：
```json
{
  "version": 2,
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4"
    }
  ]
}
```

迁移到版本 3：
```go
func (c *configV2) Migrate() (*Config, error) {
    migrated := &c.Config
    migrated.Version = 3

    // 如果未设置，添加带默认值的新字段
    // ...

    return migrated, nil
}
```

新配置（版本 3）：
```json
{
  "version": 3,
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "new_option": true
    }
  ]
}
```

## 故障排除

### 配置未升级
- 检查 `CurrentVersion` 是否已递增
- 验证迁移逻辑是否处理目标版本
- 确保 `Migrate()` 在 `LoadConfig()` 中被调用

### 迁移错误
- 检查错误消息以获取特定迁移失败信息
- 审查迁移逻辑的边缘情况
- 确保所有必需字段正确初始化
- 验证源版本的加载器函数

### 迁移后数据丢失
- 确保迁移期间所有字段都被复制
- 检查迁移不会不必要地用默认值覆盖值
- 审查加载器函数中的转换逻辑
- 检查自动备份文件（如 `config.json.20260330.bak`）以恢复原始数据
