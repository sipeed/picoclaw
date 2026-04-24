# PicoClaw 自进化技能系统设计

## 1. 设计定义

本设计描述 PicoClaw 自进化技能系统与现有 PicoClaw 运行时、memory、skills 体系之间的关系，以及自进化系统内部各部分之间的逻辑关系。

本设计采用精简概念模型：将原来的 `Learning Note` 和 `Learning Topic` 合并到同一个概念家族中，对外统一使用 `Learning Record`。

## 2. 一句话定义

PicoClaw 自进化技能系统的作用是：

“在不拖慢用户当前任务的前提下，把多次任务中反复出现、确实有价值的做法，沉淀成可审核、可回滚、可淘汰的 workspace 技能。”

## 3. 先看现有 PicoClaw，再看自进化系统

### 3.1 PicoClaw 现在已有的核心概念

当前 PicoClaw 可以简单理解成 4 层：

1. `Session / Turn`
   - 用户发来一条消息，agent 跑一轮推理、工具调用和回复
2. `Memory`
   - 保存事实、偏好、每日笔记，不等于技能
3. `Skill`
   - `workspace/skills/<name>/SKILL.md`
   - 是正式的流程型能力
4. `Skills Loader`
   - 从 workspace / global / builtin 读取正式技能并注入运行时

### 3.2 现在缺的是什么

PicoClaw 现在缺的是中间层：

- 任务结束后，哪些经验值得留下？
- 多次类似任务后，什么时候该变成技能？
- 变成技能时，是新增、补充、替换还是合并？
- 新技能如果有问题，怎么回滚？
- 长期不用的技能，怎么降温和退出？

这就是自进化系统要补上的部分。

### 3.3 自进化系统在整个 PicoClaw 里的位置

逻辑关系如下：

```text
用户任务
  -> PicoClaw 正常执行
  -> 任务结束后写入学习记录
  -> 后台维护流程分析学习记录
  -> 产出技能草案
  -> 草案通过审核/验证后更新正式技能
  -> 正式技能继续由现有 Skills Loader 加载
```

换句话说：

- 自进化系统不替代 `memory`
- 自进化系统不替代 `skills loader`
- 自进化系统只负责“经验如何进入正式技能体系”

## 4. 概念简化后的最终模型

为了简化理解，本设计只保留 3 个新的核心数据概念。

### 4.1 `Learning Record`

中文建议叫：`学习记录`

这是原来 `Learning Note` 和 `Learning Topic` 合并后的概念家族。

它有两种固定类型：

1. `task`
   - 表示“单次任务结束后留下的一条原始学习记录”
2. `pattern`
   - 表示“后台把多条相似 task 记录聚合后的模式记录”

所以：

- 原 `Learning Note` 不再单独作为公开概念存在
- 原 `Learning Topic` 不再单独作为公开概念存在
- 对外统一叫 `Learning Record`
- 仅在实现层区分 `kind=task` 和 `kind=pattern`

这样做的好处：

- 对第一次读文档的人更直观
- 少一个术语层级
- 仍然保留实现上“单次记录”和“聚合记录”的必要区别

### 4.2 `Skill Draft`

中文建议叫：`技能草案`

它表示一份尚未正式生效的技能变更提案。

可能是：

- 新建一个技能
- 给已有技能补一段
- 替换已有技能的一段
- 合并多个技能

### 4.3 `Skill Profile`

中文建议叫：`技能档案`

它不是技能正文，而是正式技能的侧车元数据。

它负责记录：

- 当前版本
- 使用次数
- 最近使用时间
- 风险等级
- 为什么创建或修改
- 有没有回滚过
- 当前处于 active / cold / archived / deleted 哪个状态

### 4.4 三个概念之间的关系

```text
Learning Record(task)
  -> 聚合
Learning Record(pattern)
  -> 生成
Skill Draft
  -> 接受并应用
正式 SKILL.md + Skill Profile
```

## 5. 与当前 PicoClaw 概念的完整关系

这部分必须明确，否则很容易误把系统看成“又做了一套 skill 系统”。

### 5.1 `Memory` 与自进化系统

- `Memory` 存事实、偏好、每日信息
- 自进化系统存流程学习证据与技能候选

它们的区别是：

- `memory` 回答“记住了什么”
- 自进化系统回答“以后应该怎么做”

### 5.2 `Skill` 与自进化系统

- `Skill` 是正式资产，表现形式是 `SKILL.md`
- 自进化系统不是 skill，本身不会被 skills loader 当作 skill 加载

它们的关系是：

- 自进化系统负责产出或更新正式 skill
- 正式 skill 仍由现有 `pkg/skills` 体系加载

### 5.3 `Tool` 与自进化系统

v1 中，自进化系统不是普通用户工具，也不是普通 workspace 技能。

它的定位是：

- 一段集成在 runtime 中的程序逻辑
- 通过任务完成钩子和后台维护流程运行

未来可以增加 operator tool 或 UI，但这不影响核心设计。

### 5.4 `Session / Turn` 与自进化系统

- 正常用户回合只负责写入原始学习记录
- 不负责生成技能草案
- 不负责审核草案
- 不负责应用草案

所以自进化系统对用户当前任务的影响应当非常小。

## 6. 运行路径总览

系统有且只有两条路径：

### 6.1 热路径：任务结束后立即发生

热路径目标：

- 不依赖 LLM
- 不明显增加当前任务延迟
- 只记录最小必要证据

热路径步骤：

1. 用户任务结束
2. agent loop 产出任务结果
3. 自进化系统写一条 `Learning Record(kind=task)`

热路径禁止做的事：

- 调用 LLM 生成技能草案
- 读取大量已有技能
- 对全量技能做相似性比较
- 直接修改 `SKILL.md`
- 清理 lifecycle

### 6.2 冷路径：后台维护流程

冷路径目标：

- 处理重逻辑
- 允许依赖 LLM，但不能挡住用户当前任务

冷路径步骤：

1. 读取若干 `Learning Record(kind=task)`
2. 聚合成 `Learning Record(kind=pattern)`
3. 计算该模式是否成熟
4. 召回相似正式技能
5. 如果值得升级，则让 LLM 生成 `Skill Draft`
6. 对草案做结构校验与安全扫描
7. 将草案放入 candidate / quarantined / accepted
8. 若草案被接受，则更新正式 `SKILL.md`
9. 更新对应 `Skill Profile`
10. 对长期不用技能执行 cold / archived / deleted 迁移

## 7. LLM 依赖清单

### 7.1 不依赖 LLM 的部分

- 写 `Learning Record(kind=task)`
- 聚合 `Learning Record(kind=pattern)`
- 计算模式成熟度
- 召回相似技能的第一阶段过滤
- 结构校验
- 安全扫描
- 版本备份
- 回滚
- 生命周期状态更新

### 7.2 依赖 LLM 的部分

- 从 `Learning Record(kind=pattern)` 生成 `Skill Draft`

只有这一处是必须依赖 LLM 的核心点。

### 7.3 明确的性能风险

最容易拖慢用户体验的地方只有两个：

1. 把草案生成放到同步任务结束后
2. 每次任务都去全量扫描已有技能

因此 v1 的硬约束是：

- 草案生成必须走冷路径
- 相似技能召回必须先走轻量规则或索引

## 8. 三个核心数据对象的精确定义

这部分不再用“建议字段”这种模糊说法，而是直接说明每个对象：

- 以什么形式存在
- 存在哪
- 谁写
- 谁读
- 在什么阶段使用

### 8.1 `Learning Record`

#### 它是什么

`Learning Record` 是自进化系统的输入数据。

它有两种固定类型：

- `kind=task`
- `kind=pattern`

#### 它以什么形式存在

推荐持久化形式：

```text
<stateDir>/evolution/<workspace-hash>/learning-records.jsonl
```

每条记录一行 JSON，带 `kind` 字段。

这样做的原因：

- 追加写成本低
- 易于审计
- 易于后续做聚合和回放

#### 谁写它

- 热路径写 `kind=task`
- 冷路径维护流程写 `kind=pattern`

#### 谁读它

- 冷路径聚合器读取 `kind=task`
- 草案生成器读取 `kind=pattern`
- 审计工具或未来 UI 可读取两者

#### 它在哪些阶段使用

- `kind=task`：作为单次任务学习证据
- `kind=pattern`：作为生成技能草案的直接输入

#### `kind=task` 必含字段

- `id`
- `kind`
- `workspace_id`
- `session_key`
- `created_at`
- `task_hash`
- `task_summary`
- `success`
- `tool_calls_count`
- `tool_kinds`
- `active_skill_names`
- `had_user_correction`
- `attempt_trail`
- `signals`

#### `kind=pattern` 必含字段

- `id`
- `kind`
- `workspace_id`
- `created_at`
- `updated_at`
- `source_record_ids`
- `pattern_key`
- `summary`
- `event_count`
- `success_rate`
- `repeat_score`
- `maturity_score`
- `winning_path`
- `matched_skill_names`
- `status`

#### 为什么必须存在

它的作用是把“单次任务经验”和“重复模式经验”从正式 skill 里隔离出去。
如果没有它，系统就只能在任务结束后直接写 skill，这会导致：

- 噪声大
- 风险高
- 很难解释为什么学出了这个 skill

### 8.2 `Skill Draft`

#### 它是什么

`Skill Draft` 是候选技能变更提案。

#### 它以什么形式存在

推荐持久化形式：

```text
<stateDir>/evolution/<workspace-hash>/skill-drafts.json
```

按数组或 map 存储，每个 draft 带唯一 id 和状态。

#### 谁写它

- 仅冷路径中的 draft generator 写入

#### 谁读它

- 验证器
- 安全扫描器
- candidate 决策器
- 草案应用器
- 未来的 operator UI / tool

#### 它在哪些阶段使用

- 模式成熟后生成
- 通过验证后进入 candidate
- 被接受后转化成正式 skill 更新

#### 必含字段

- `id`
- `workspace_id`
- `created_at`
- `updated_at`
- `source_pattern_id`
- `target_skill_name`
- `draft_type`
  - `workflow`
  - `shortcut`
- `change_kind`
  - `create`
  - `append`
  - `replace`
  - `merge`
- `human_summary`
- `usage_scope`
- `preferred_entry_path`
- `avoid_patterns`
- `review_notes`
- `risk_level`
- `body_or_patch`
- `similar_skill_refs`
- `scan_findings`
- `status`

#### 为什么必须存在

它的作用是防止系统直接把学习结果写进正式技能。

如果没有 `Skill Draft` 这一层，就无法优雅支持：

- 候选态
- 安全扫描
- 人工审核
- 回滚前的变更解释

### 8.3 `Skill Profile`

#### 它是什么

`Skill Profile` 是正式技能的生命周期档案。

#### 它以什么形式存在

推荐持久化形式：

```text
<stateDir>/evolution/<workspace-hash>/profiles/<skill-name>.json
```

它不是 `SKILL.md` 正文，而是旁边的审计与生命周期卡。

#### 谁写它

- 草案应用器在技能生效时写入或更新
- 生命周期管理器在冷却、归档、删除时更新
- 运行时命中统计器在技能使用后更新

#### 谁读它

- 生命周期管理器
- 命中排序逻辑
- 回滚逻辑
- 未来的 operator surface

#### 它在哪些阶段使用

- 技能应用时
- 技能被命中时
- 技能进入 cold / archived / deleted 时
- 技能发生回滚时

#### 必含字段

- `skill_name`
- `workspace_id`
- `current_version`
- `status`
  - `active`
  - `cold`
  - `archived`
  - `deleted`
- `origin`
  - `manual`
  - `imported`
  - `evolved`
- `human_summary`
- `intended_use_cases`
- `non_goals`
- `preferred_entry_path`
- `avoid_patterns`
- `review_checklist`
- `risk_level`
- `change_reason`
- `last_used_at`
- `use_count`
- `success_count`
- `shortcut_win_count`
- `superseded_count`
- `retention_score`
- `version_history`

#### 为什么必须存在

如果只有 `SKILL.md` 而没有 `Skill Profile`，审核者很难回答：

- 这个技能最近还在用吗？
- 这个技能是系统学出来的还是人工写的？
- 这个技能为什么被改过？
- 上一个稳定版本在哪里？

## 9. 技能变更的完整逻辑

### 9.1 草案类型

`Skill Draft` 有两类：

1. `workflow`
   - 学到的是完整流程
2. `shortcut`
   - 学到的是“这类任务起手应该先走哪条路径”

### 9.2 变更类型

草案支持 4 种变更：

1. `create`
   - 没有合适技能时，新建
2. `append`
   - 现有技能是对的，只需要补充
3. `replace`
   - 现有技能某一段已经过时或错误，需要替换
4. `merge`
   - 多个相近技能合并成一个更清晰的技能

### 9.3 默认优先级

为降低风险，默认决策优先级是：

```text
append > create > replace > merge
```

### 9.4 为什么要支持 shortcut

一个典型场景是：

- agent 尝试了多个 skill
- 前几个 skill 失败或绕远
- 最后一个路径稳定成功

这时系统学到的不是“最后一个 skill 很好”，而是：

- 以后遇到类似任务，应当优先从最后那条成功路径开始

这类学习结果应写成：

- `## Start Here`
- `## Preferred Path`
- `## Avoid Common Dead Ends`

而不是简单复制一次完整流程。

## 10. 回滚与安全逻辑

### 10.1 安全扫描

每一份 `Skill Draft` 在进入正式技能前，都必须经过：

- 结构校验
- 安全扫描

失败结果：

- 不进入正式技能
- 状态改为 `quarantined`

### 10.2 备份

以下操作必须先备份旧技能：

- `append`
- `replace`
- `merge`

推荐备份路径：

```text
<stateDir>/evolution/<workspace-hash>/backups/<skill-name>/<timestamp>/
```

### 10.3 回滚触发条件

v1 采用结构驱动回滚，不做行为回滚。

触发条件包括：

- 应用后 `SKILL.md` 结构不合法
- frontmatter 非法
- patch 应用失败
- 文件缺失
- skills loader 无法再识别该技能

### 10.4 回滚记录

所有回滚都必须写入 `Skill Profile.version_history`，记录：

- 触发回滚的草案 id
- 回滚原因
- 恢复到的版本
- 对人工审核有帮助的变更摘要

## 11. 生命周期逻辑

正式技能状态固定为：

- `active`
- `cold`
- `archived`
- `deleted`

### 11.1 各状态含义

- `active`
  - 正常参与运行时
- `cold`
  - 保留但降权
- `archived`
  - 保留但默认不再参与正常推荐
- `deleted`
  - 已删除，仅保留极简墓碑

### 11.2 状态迁移

- `active -> cold`
  - 最近不常用，且经常被别的技能或路径替代
- `cold -> active`
  - 再次命中并证明有效
- `cold -> archived`
  - 长期不用，且保留价值有限
- `archived -> active`
  - 后续再次命中并成功
- `archived -> deleted`
  - 长期无关、保留价值低、且不触发保护规则

### 11.3 误删保护

以下技能不应轻易自动删除：

- `origin=manual`
- 低频但高价值
- 最近刚被替换、仍承担回滚支点作用
- 新近转正、还在观察窗口内

## 12. 与 OpenClaw 自进化能力的区别

这部分单独列出，避免混淆。

### 12.1 OpenClaw 已有、PicoClaw 也应借鉴的能力

| 能力 | OpenClaw | 本设计 |
|---|---|---|
| 短期证据再晋升 | 有，体现在 memory dreaming | 有，体现在 `Learning Record(kind=task -> pattern)` |
| proposal/candidate 思路 | 有，体现在 `skill-workshop` | 有，体现在 `Skill Draft` |
| `create/append/replace` | 有 | 有 |
| 安全扫描与隔离 | 有 | 有 |

### 12.2 OpenClaw 有，但本设计明确不照搬的点

| 能力 | OpenClaw | 本设计的选择 |
|---|---|---|
| 同步 `agent_end` 上的 reviewer 风格学习 | 有相关设计 | v1 不做，避免拖慢用户当前任务 |
| 技能学习与运行时更紧耦合 | 相对更重 | PicoClaw v1 更强调冷路径异步化 |

### 12.3 本设计有、OpenClaw 当前不完整或没有明确提供的点

| 能力 | OpenClaw | 本设计 |
|---|---|---|
| 从“多次试错后最后成功路径”学习 shortcut | 没有明确独立建模 | 有，作为 `shortcut` 草案 |
| 正式技能的 `active/cold/archived/deleted` 生命周期 | 没有完整公开模型 | 有 |
| 明确的版本备份与结构驱动回滚 | 相对较弱或不完整 | 有，且是硬要求 |
| 面向人工审核的 `Skill Profile` | 没有完整强调 | 有，作为一等设计目标 |
| 将 `Learning Note` / `Topic` 统一成更易理解的公开概念 | 没有 | 有，统一成 `Learning Record` |

## 13. 三个典型运行例子

### 例子 1：天气技能学出 shortcut

场景：

- 多次中文城市天气查询
- agent 常常先试通用 geocode
- 最后 native-name query 才稳定成功

运行路径：

1. 每次任务结束写一个 `Learning Record(kind=task)`
2. 冷路径聚合出一个 `Learning Record(kind=pattern)`，识别到稳定 winning path
3. LLM 生成一个 `Skill Draft(draft_type=shortcut, change_kind=append)`
4. 草案建议给 `weather` skill 增加 `Start Here`
5. 通过验证后进入 candidate
6. 再次命中成功后转正
7. `weather/SKILL.md` 被补充，`Skill Profile` 更新版本

### 例子 2：修复一个已有错误技能

场景：

- 某个已有技能里包含过时流程
- 后续多次任务都证明这段流程会误导 agent

运行路径：

1. 多次任务写入 `Learning Record(kind=task)`
2. 聚合成 `kind=pattern`
3. 冷路径发现应使用 `replace`
4. 生成 `Skill Draft(change_kind=replace)`
5. 应用前先备份旧技能
6. 如果新版本结构异常，则立即回滚
7. `Skill Profile.version_history` 记录这次替换和回滚

### 例子 3：一个阶段性技能被清退

场景：

- 某个 release 流程技能只在一次项目迁移期间有用
- 后续几个月都不再命中

运行路径：

1. 技能长时间未使用
2. `Skill Profile.retention_score` 持续下降
3. 状态从 `active -> cold`
4. 继续长时间无命中，则 `cold -> archived`
5. 若之后依然长期无关，且不属于 manual / high-value 类型，则可 `archived -> deleted`

## 14. 推荐的 v1 实施范围

v1 应包含：

- `Learning Record`
- `Skill Draft`
- `Skill Profile`
- 热路径只写 `kind=task`
- 冷路径聚合出 `kind=pattern`
- 冷路径 LLM 生成草案
- `create` / `append` / `replace`
- shortcut 学习
- 结构验证
- 安全扫描
- 备份
- 回滚
- lifecycle 四状态

v1 暂缓：

- 跨 workspace 自进化
- 行为级回滚
- 全自动 `merge`
- 完整 UI
- 热路径 LLM rerank

## 15. 结论

这套设计的核心逻辑关系可以简化成一句话：

```text
PicoClaw 正常执行任务
  -> 自进化系统记录学习证据
  -> 自进化系统在后台把重复经验提炼成技能草案
  -> 草案经过验证和治理后更新正式技能
  -> 正式技能继续回到 PicoClaw 现有技能体系中运行
```

因此：

- PicoClaw 是主系统
- 自进化系统是主系统旁边的一条“技能学习与治理管道”
- 正式技能仍然属于 PicoClaw 现有 skill 概念
- 自进化系统不是独立 agent，不是普通 tool，也不是普通 skill

这就是 PicoClaw 与自进化系统的完整关系；
而 `Learning Record -> Skill Draft -> Skill Profile -> 正式 Skill`，就是自进化系统内部的完整逻辑关系。
