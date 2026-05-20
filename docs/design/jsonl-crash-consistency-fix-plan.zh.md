# JSONL Store 崩溃一致性修复方案

## 1. 问题定义

`pkg/memory/jsonl.go` 的 `addMsg` 先追加 `.jsonl` 数据行，再更新 `.meta.json`。

当进程在两步之间崩溃时，会出现：

- `.jsonl` 已包含新消息
- `meta.Count` 仍是旧值
- `GetHistory` 能读到新消息，但不会修复 `meta.Count`
- `readMessages` 只按 `meta.Skip` 读取，不校验元数据是否与真实文件一致

这会导致 metadata 长期漂移，并在后续截断、压缩、重启恢复中放大为静默丢消息风险。

## 2. 本次目标

- 在不引入 WAL 的前提下，消除 `Count/Skip` 长期漂移
- 把“崩溃后永久不一致”收敛为“首次访问自动自愈”
- 保持现有 JSONL 存储格式兼容，不做破坏性迁移
- 用单元测试覆盖启动恢复、读取恢复、越界 `Skip` 恢复

## 3. 已实施改动

- `NewJSONLStore` 增加启动期 metadata repair：
  - 扫描现有 `.meta.json`
  - 对比真实 JSONL 行数与 `meta.Count`
  - 当 `Skip > Count` 或 `Count` 漂移时自动回写修复
- 增加统一修复入口 `repairSessionMetaLocked`
  - 使用真实 JSONL 原始非空行数作为 `Count`
  - 将越界 `Skip` 收敛到 `rawCount`
  - 修复缺失的 `Key`
  - 对缺失时间戳的非空会话补齐 `CreatedAt/UpdatedAt`
- 在以下访问路径中接入修复：
  - `GetSessionMeta`
  - `GetHistory`
  - `TruncateHistory`
  - `Compact`

## 4. 设计取舍

- 选择“按需修复 + 启动预修复”，而不是本次直接上 WAL。
  - 优点：兼容现有数据格式，改动面小，风险可控
  - 缺点：追加写和 meta 更新仍不是严格原子
- `Count` 继续定义为 JSONL 原始非空行数，而不是过滤后的 retained message 数。
  - 这是为了与现有 `Skip` 语义、`TruncateHistory` 的 raw-line 计算保持一致
- 对损坏尾行保持容忍：
  - `readMessages` 继续跳过坏行
  - repair 仅修复 metadata，不自动重写 JSONL 文件

## 5. 验证策略

新增测试覆盖：

- `TestGetHistory_RepairsStaleMetaCount`
  - 验证读取路径能修复“JSONL 比 meta 多一行”的崩溃后状态
- `TestNewJSONLStore_RepairsExistingMetaOnStartup`
  - 验证重启后 store 初始化阶段会修复旧 metadata
- `TestGetHistory_RepairsSkipPastEOF`
  - 验证 `Skip` 越界时不会长期维持错误状态

已有测试继续覆盖：

- `TestTruncateHistory_StaleMetaCount`
- `TestCrashRecovery_PartialLine`

## 6. 残余风险

- 当前方案只能修复元数据漂移，不能提供严格的跨文件原子提交
- 如果 `.meta.json` 写成功而 `.jsonl` 后续重写失败，仍会出现“metadata 比数据更前”的窗口
- 启动期 repair 依赖扫描 JSONL，超大历史会话会带来额外启动成本

## 7. 下一阶段建议

按优先级建议后续继续推进：

1. 引入 WAL 或单文件事务日志，保证 append 与 meta update 可恢复地提交
2. 为 repair 增加可观测性指标和日志计数，便于发现线上频繁自愈
3. 评估在 `Compact`/`SetHistory` 路径中加入更强的崩溃恢复标记
4. 若会话规模继续增长，考虑把 `Count/Skip` 与 active count 分离建模
