# Agent 自进化技能生命周期设计

## 概要

本设计为 PicoClaw 增加一套轻量级的自进化闭环，让 agent 能从已完成任务中学习可复用流程，同时不拖慢用户主交互路径。

系统将短期学习证据与正式技能分开管理：

- `Learning Note`：任务结束后写下的一条极小结构化学习笔记
- `Learning Topic`：由多条相关学习笔记聚合而成的学习主题
- `Skill Draft`：从成熟主题中提炼出的候选技能草案
- `Skill Profile`：正式技能的生命周期与版本档案

v1 版本采用 workspace 级作用域、分级治理策略，并明确要求：

- 重学习逻辑不进入热路径
- 新技能先进入候选态
- 支持合并、替换、回滚
- 支持长期不用技能的冷却、归档、删除

## 目标

- 让 agent 能从真实任务中学习可复用流程
- 让系统能从反复试错中学出“捷径型”技能
- 不显著增加日常任务的延迟和提示词 token 成本
- 通过候选态、验证、回滚和生命周期治理，避免技能集被污染
- 尽量优先演化已有技能，而不是不断新增新技能

## 非目标

- v1 不做跨 workspace 的全局自进化
- v1 不允许对高影响技能进行完全无治理的自主破坏式修改
- v1 不允许在用户同步回复路径中运行依赖 LLM 的学习逻辑
- 不替代现有 `memory` 或 `skills` 子系统

## 与 PicoClaw 现有结构的关系

PicoClaw 当前已经具备：

- `pkg/skills`：workspace / global / builtin 技能加载能力
- `pkg/agent/context.go`：系统提示词中的技能摘要注入
- `pkg/agent/memory.go`：基于 markdown 的轻量记忆能力

当前缺少的是位于“有价值经验”与“正式技能”之间的生命周期层。本设计补上这层能力，但不改变现有子系统的角色：

- `memory` 继续负责 durable facts 和 notes
- `skills` 继续负责正式的 `SKILL.md` 技能资产
- 新的 evolution 子系统负责判断什么时候经验应当沉淀为技能，以及这些技能之后如何演化和退出

## 设计原则

- 热路径必须足够小：正常回合只允许写少量学习证据
- 重工作必须走冷路径：聚类、提案生成、相似性比对、清理都异步进行
- 记忆与技能必须分离：流程学习不能混进通用 memory
- 候选态优先：新学到的流程先进入候选池，而不是直接变成正式技能
- 优先保留已有价值内容：`append` 比 `replace` 更稳，`replace` 比 `merge` 更稳
- 回滚必须便宜且确定：一旦结构异常，应当能稳定恢复旧版本
- 人类可审阅性是一级约束：生成物、侧车元数据和最终技能结构都必须方便人工阅读、审核和理解

## 人类可审阅性要求

系统不仅要优化 agent 的执行效果，也必须优化人工审核体验。

这一要求适用于三个层面：

1. 运行时元数据
   - 侧车元数据要能解释一个技能是做什么的、为什么存在、改了什么、风险在哪
2. 正式技能内容
   - 生成或更新后的 `SKILL.md` 应当像一份紧凑的操作文档，而不是难以审阅的机器碎片
3. 实现结构
   - evolution 子系统应将学习证据、候选逻辑、生命周期逻辑拆成边界清晰的模块，降低人工读代码的心智负担

理想情况下，人工审核者应当能快速回答：

- 这个技能解决什么问题？
- 它应该在什么场景下使用？
- 它的首选起手路径是什么？
- 它明确建议避免哪些错误路径？
- 它为什么被创建或修改？
- 这次变更是低风险还是高影响？

## 与 OpenClaw 的对照

本设计借鉴 OpenClaw 中两套已被验证的思路，同时针对 PicoClaw 的轻量目标做约束性调整。

### 借鉴点

- 来自 `memory-core` dreaming：
  - 先收集短期证据，再做 durable promotion
  - 通过阈值而不是一次命中就晋升
  - 用后台维护而不是不断往主 prompt 里塞上下文
- 来自 `skill-workshop`：
  - 用 proposal / candidate 思维做技能演化
  - 支持 `create` / `append` / `replace`
  - 在应用前做扫描与隔离

### 需要调整的点

- PicoClaw v1 不应在同步 `agent_end` 路径上运行 LLM reviewer
- PicoClaw 需要比 OpenClaw 更强的版本备份与回滚机制
- PicoClaw 需要正式技能生命周期：`active -> cold -> archived -> deleted`

## 面向人的术语体系

为了方便文档、日志、未来 UI 和代码审核，设计中统一采用以下更直观的命名：

- `Learning Note`：一条学习笔记
- `Learning Topic`：一个学习主题
- `Skill Draft`：一份技能草案
- `Skill Profile`：一个正式技能的档案卡

只有 `skills/<name>/SKILL.md` 才是真正的正式技能。其余对象都是运行时的内部管理结构。

## 总体架构

evolution 子系统可以拆成四层：

1. 证据层
   - 保存 `Learning Note`
   - 低成本、追加式、热路径安全
2. 模式层
   - 把若干学习笔记聚成 `Learning Topic`
   - 负责判断成熟度和是否值得晋升
3. 候选层
   - 生成并管理 `Skill Draft`
   - 负责匹配、验证、隔离、候选入池
4. 生命周期层
   - 管理正式技能的 `Skill Profile`
   - 负责激活、冷却、归档、删除和回滚记录

## 热路径与冷路径

### 热路径

允许出现在正常用户回合中的动作：

- 写一条 `Learning Note`
- 附加轻量规则信号，例如：
  - 疑似重复模式
  - 观察到用户纠正
  - 观察到最终成功路径
  - 观察到技能缺口

不允许出现在正常用户回合中的动作：

- LLM 生成技能草案
- 对全量技能做深度相似性比较
- 草案转正式技能
- 生命周期清理
- 对大量技能做 reviewer 扫描

### 冷路径

由 heartbeat、cron、maintenance run 或显式管理动作触发：

- 聚合学习笔记形成主题
- 计算主题成熟度
- 检索相似的已有技能
- 生成 `Skill Draft`
- 运行结构校验和安全扫描
- 将草案转入 candidate / quarantined / accepted
- 更新 `Skill Profile` 的使用状态与生命周期
- 执行 cold / archived / deleted 清理

## LLM 依赖边界

### 不依赖 LLM 的步骤

- 写入 `Learning Note`
- 打规则信号
- 聚合 `Learning Topic`
- 主题成熟度评分
- 非 LLM 的相似技能召回
- 结构校验
- 安全扫描
- 版本备份与回滚
- 使用计数与生命周期迁移

### 依赖 LLM 的步骤

- 生成 `Skill Draft` 的正文或 patch
- 把一个成熟 `Learning Topic` 重写成：
  - 新 workflow skill
  - 一个追加段落
  - 一个替换 patch
  - 一个以“起手路径”为核心的 shortcut 段落

### 明确的性能风险点

最大的体验风险，是把依赖 LLM 的 review 或 draft 生成放在任务同步结束时做。  
因此 v1 必须保证：所有依赖 LLM 的学习动作都不进入用户回复热路径。

## 核心对象

## `Learning Note`

用途：

- 记录一次已完成任务里可能值得学习的结构化证据

建议字段：

- `id`
- `created_at`
- `workspace_id`
- `session_key`
- `task_hash`
- `task_summary`
- `success`
- `tool_calls_count`
- `tool_kinds`
- `had_user_correction`
- `active_skill_names`
- `signals`
- `artifact_refs`
- `attempt_trail`

实现形态：

- 集成在 runtime 中的内部程序数据
- 存在 state 中，而不是 `skills/`
- 不注入日常 prompt

是否依赖 LLM：

- 否

## `Learning Topic`

用途：

- 表示若干 `Learning Note` 聚合出的可复用流程主题

建议字段：

- `id`
- `created_at`
- `updated_at`
- `workspace_id`
- `fingerprint`
- `title_hint`
- `tool_signature`
- `event_ids`
- `event_count`
- `success_rate`
- `correction_rate`
- `diversity_score`
- `recency_score`
- `promotion_score`
- `matched_skill_candidates`
- `winning_path`
- `status`

实现形态：

- 集成在 runtime 中的内部程序数据
- 由后台维护逻辑构建

是否依赖 LLM：

- v1 不依赖

## `Skill Draft`

用途：

- 表示一份尚未正式生效的技能变更草案

建议字段：

- `id`
- `created_at`
- `updated_at`
- `workspace_id`
- `source_topic_id`
- `source_note_ids`
- `target_skill_name`
- `draft_type`
- `change_kind`
- `reason`
- `description`
- `body_or_patch`
- `similar_skill_refs`
- `human_summary`
- `usage_scope`
- `preferred_entry_path`
- `avoid_patterns`
- `review_notes`
- `risk_level`
- `llm_generation_meta`
- `scan_findings`
- `status`

其中：

- `draft_type`
  - `workflow`
  - `shortcut`
- `change_kind`
  - `create`
  - `append`
  - `replace`
  - `merge`

实现形态：

- 存在 state 中的候选资产
- 在被接受并应用前，不属于正式技能集

是否依赖 LLM：

- 生成草案正文时依赖

面向人工审核的要求：

- 每份草案都应当能在不回看完整 transcript 的前提下被理解
- `human_summary` 要用 1-3 句话说明核心用途
- `usage_scope` 要明确适用范围
- `preferred_entry_path` 要在适用时说明首选起手路径
- `avoid_patterns` 要列出常见死路或反模式
- `review_notes` 要解释为什么本次判定是 `create` / `append` / `replace` / `merge`

## `Skill Profile`

用途：

- 管理一个正式技能的生命周期与版本元数据

建议字段：

- `skill_name`
- `workspace_id`
- `current_version`
- `status`
- `origin`
- `last_used_at`
- `use_count`
- `success_count`
- `shortcut_win_count`
- `superseded_count`
- `specificity_score`
- `retention_score`
- `cooldown_reason`
- `archive_reason`
- `last_matched_topic_id`
- `human_summary`
- `review_tags`
- `owner_scope`
- `intended_use_cases`
- `non_goals`
- `risk_level`
- `change_reason`
- `preferred_entry_path`
- `avoid_patterns`
- `review_checklist`
- `version_history`

实现形态：

- 正式 `SKILL.md` 周围的侧车元数据
- 是内部程序数据，不是 skill 正文本身

是否依赖 LLM：

- 生命周期管理本身不依赖

面向人工审核的要求：

- `Skill Profile` 应当像一张审计卡，而不只是流水账
- 审核者应当能快速看出：
  - 技能要做什么
  - 技能不应该做什么
  - 它为什么被引入或修改
  - 它是 shortcut 型还是完整 workflow 型
  - 高风险修改需要额外验证哪些点

## 正式技能的可读性约定

系统在落地或更新正式 `SKILL.md` 时，应优先采用方便人工扫描的结构。

推荐章节顺序：

1. 简短用途摘要
2. 何时使用
3. `Start Here`
4. `Workflow`
5. `Avoid Common Dead Ends`
6. 验证或审核提示
7. 必要时的参考资料

生成内容应当倾向于：

- 简短的祈使句 bullet
- 明确的适用边界
- 如果学到了 shortcut，就把首选起手路径写清楚
- 如果观察到了重复死路，就明确写出反模式

生成内容应当避免：

- 回放完整 transcript
- 含糊的自传式解释
- 用长篇叙述替代简洁检查清单
- 把关键起手路径埋在文件深处

## 从试错链中学习“捷径技能”

系统必须支持这样一种学习：一次任务里 agent 尝试了多个 skill 或方法，前面多次失败或绕远，最后一个路径成功。

这学到的不是“最后一个 skill 很好”，而是一个路由教训：

- 前面那些路径在这类任务里通常浪费时间或不稳定
- 最后成功的路径更适合作为类似任务的默认起手路径

### 必须记录的证据

`Learning Note.attempt_trail` 需要记录有顺序的尝试链，例如：

- 试了什么 skill / 方法
- 结果类型：`failed`、`partial`、`superseded`、`success`
- 如果可以结构化识别，记录失败原因

### 必须支持的晋升结果

如果重复出现相同的 winning path 证据，系统可以生成 `shortcut` 类型的 `Skill Draft`。

这类草案通常会变成：

- `## Start Here`
- `## Preferred Path`
- `## Avoid Common Dead Ends`

这样下次类似任务就能从正确路径开始，而不是再走一遍旧的试错链。

## 草案生成与匹配流程

当一个 `Learning Topic` 达到成熟阈值时，按以下流程处理：

1. 召回最相近的正式技能与候选草案
2. 判断这次变更更适合：
   - `create`
   - `append`
   - `replace`
   - `merge`
3. 调用 LLM 生成草案正文或 patch
4. 运行结构校验
5. 运行安全扫描
6. 根据结果进入：
   - `candidate`
   - `quarantined`
   - 只有满足进一步晋升条件时才进入 `accepted`

## 变更类型决策规则

## `create`

适用条件：

- 没有足够相近的正式技能
- 学到的是当前 workspace 中真正新的流程

## `append`

适用条件：

- 目标技能整体是对的
- 新学习只是在补充一个段落、例外、检查点或 shortcut

默认情况下，能安全 `append` 时优先 `append`。

## `replace`

适用条件：

- 现有技能中的某一段明显过时、错误或具有误导性
- 新内容要覆盖旧路径，而不是仅做补充

这种情况必须先做版本备份。

## `merge`

适用条件：

- 两个重叠技能应整合成一个更清晰的技能
- 或一个新草案已经实质上覆盖了多个窄技能

这是风险最高的操作，v1 默认只进入候选态，不应静默自动生效。

## `Skill Draft` 状态机

建议状态：

- `draft`
- `candidate`
- `quarantined`
- `accepted`
- `rejected`
- `superseded`
- `rolled_back`

迁移主链路：

1. 主题达到成熟阈值
2. LLM 生成 `draft`
3. 经过验证 / 扫描：
   - 干净 -> `candidate`
   - 被阻断 -> `quarantined`
4. 后续晋升：
   - 低风险场景可自动接受
   - 高影响场景需人工确认
5. 被应用后生成或更新正式技能与 `Skill Profile`
6. 如果应用后技能结构异常 -> `rolled_back`

## 候选草案转正规则

## 低风险自动转正

仅适用于低影响变更，例如：

- 很窄的 `append`
- 不会覆盖核心流程的 shortcut guidance
- 给已有 workspace skill 增补少量说明

需要同时满足：

- 结构校验通过
- 安全扫描通过
- 后续相似任务再次验证该草案有效
- 不涉及高风险的 `replace` 或 `merge`

## 人工确认转正

以下场景应默认需要人工确认：

- `replace`
- `merge`
- 较大的 `create`
- 会显著改变默认行为的草案

这与前述分级治理一致：

- 低风险可自动演化
- 高影响必须确认

## 验证、安全与回滚

## 结构验证与安全扫描

每份草案都必须经过：

- resulting skill 结构验证
- 针对 prompt injection、权限绕过、危险 shell 模式的安全扫描

如果失败，草案直接进入 `quarantined`，不触碰正式技能。

## 备份规则

凡是要修改已有技能的操作，都必须先做备份，包括：

- `append`
- `replace`
- `merge`

备份内容至少包括：

- 原始 `SKILL.md`
- 若有改动，相关支持文件
- proposal / draft id
- 时间戳

## 回滚触发条件

v1 中回滚是确定性的、结构驱动的，不做行为回滚。

触发回滚的典型条件：

- 新技能在落地后结构校验失败
- 关键文件缺失
- frontmatter 不合法
- patch 没正确应用
- 结果技能为空或超限
- loader 无法再识别该技能

## 回滚记录

`Skill Profile.version_history` 应至少记录：

- 哪份草案被应用
- 使用了哪个备份恢复
- 为什么回滚
- 回滚后当前版本是什么

同时，还应保留面向人工的审阅轨迹：

- 简短变更摘要
- 这是 `workflow` 还是 `shortcut`
- 这是 `create` / `append` / `replace` / `merge`
- 为什么旧路径被认为不足

## 正式技能生命周期

正式技能状态统一为：

- `active`
- `cold`
- `archived`
- `deleted`

## `active`

- 参与正常技能使用与推荐

## `cold`

- 仍被保留，但默认降权
- 适合那些“以前有用，但最近明显降频”的技能

## `archived`

- 保留文件与档案，默认退出主舞台
- 不参与常规推荐，但可以在以后重新恢复

## `deleted`

- 在长时间无关且保留价值低时正式删除
- 仍保留极简 tombstone，避免系统马上又学出一个已知低价值技能

## 保留分与误删保护

技能删除不能只看时间，必须综合 `retention_score`：

- 最近使用时间
- 历史使用频率
- 成功可靠性
- 是否经常成为最终成功路径
- 与当前 workspace 的特异性

## 必须遵守的保护规则

- `manual` 来源的技能在 v1 不自动删除
- 低频但高价值技能不自动删除
- 最近被 `replace` 的旧技能版本必须保留一个回滚保护窗口
- 新近转正的技能不应立刻进入冷却

## 生命周期迁移规则

- `active -> cold`
  - 使用下降，且越来越经常被其他路径取代
- `cold -> active`
  - 再次命中并证明有效时恢复
- `cold -> archived`
  - 冷却后继续长期不用，且保留价值有限
- `archived -> active`
  - 再次命中并成功时解档恢复
- `archived -> deleted`
  - 长期无关、保留价值低且不触发保护规则时删除

## 一个完整例子

例子：多次中文城市天气任务中，最终总是 native-name lookup 成功，而前面的通用 geocode 路径浪费时间。

1. 每次完成任务后写一条 `Learning Note`
2. 多条笔记聚成 `Learning Topic`，例如 `weather-cn-routing`
3. 主题中反复出现相同 winning path：
   - 通用 geocode 容易绕远或出错
   - native-name resolution 更稳定
4. 冷路径 maintenance run 调用 LLM 生成一个 `shortcut` 类型的 `Skill Draft`
5. 匹配逻辑判断：最合适的是对现有 `weather` skill 做 `append`
6. 草案内容变成：

```md
## Start Here

For Chinese-city weather requests, try native-name resolution first.
Do not start with generic geocoding unless the native query is ambiguous.
```

7. 草案通过结构校验和安全扫描，进入 `candidate`
8. 后续相似任务再次验证有效后，草案被接受并应用
9. `weather` 的 `Skill Profile` 记下一条新版本记录
10. 如果应用后技能结构异常，系统立即回滚到旧版本

## 在 PicoClaw 中的实现形态

这套能力在 v1 中应实现为集成式 runtime 子系统，而不是普通 workspace skill，也不是面向用户的常规 tool。

建议落点：

- 任务完成后的证据写入逻辑靠近 `pkg/agent`
- evolution state 存在 workspace 或 state-dir 所属的数据区域
- 正式技能仍然保存在 `workspace/skills`
- 生命周期元数据与备份存在 state 或受控侧车文件中

未来可以补 operator surface，但核心机制本身不是“一个技能”。它是帮助系统决定“什么时候该有技能、技能应该如何变化”的一段集成程序。

实现可读性也应当是设计约束：

- 将证据采集、主题聚合、草案生成、生命周期迁移拆成清晰模块
- 尽量避免笼统的 manager 大类命名
- 将面向人工的元数据存为稳定 schema，而不是埋在瞬时日志里
- 让状态迁移既能从代码读清楚，也能从持久化元数据中审计出来，而不必回放原始 transcript

## 成功标准

如果满足以下条件，则说明设计成功：

- 常见用户回合依然快，没有新引入 LLM 学习延迟
- 新学到的流程先以候选态出现，而不是静默改写正式技能
- 反复试错能沉淀成可复用的 shortcut guidance
- 高影响技能修改可以确定性回滚
- 长期不用技能可以退出 active 集合，同时不误删少见但有价值的能力

## 风险与权衡

- 候选治理会增加实现复杂度，但能显著降低 prompt 和技能集污染
- 非 LLM 的相似技能匹配不一定完美，但 v1 为了热路径成本，这个取舍是合理的
- `merge` 很强大，但也是最高风险能力，前期必须保守
- 生命周期清理如果评分太粗糙，可能会过度修剪，这也是为什么 manual / rare / rollback-relevant 技能要得到更强保护

## 推荐的 v1 实施范围

v1 建议包含：

- `Learning Note`
- `Learning Topic`
- `Skill Draft`
- `Skill Profile`
- 冷路径 LLM 草案生成
- `create` / `append` / `replace`
- `shortcut` 类型草案
- 结构验证、扫描、候选态、备份、回滚
- `active` / `cold` / `archived` / `deleted`

v1 建议暂缓：

- 跨 workspace 演化
- 行为回滚
- 全自动 `merge`
- 全生命周期对象的完整 UI
- 热路径上的 LLM reranking
