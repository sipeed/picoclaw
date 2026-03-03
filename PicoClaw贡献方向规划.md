# PicoClaw 贡献方向规划（3月1日更新）

## 个人情况

- Go 开发者，会 Python，在学 AI Agent
- 已合并 PR：#173（多bug修复）、#186（安全加固）
- 已提交 PR：#732（JSONL session store，等待 review）
- 已关闭 PR：#719（SQLite 方案，被维护者建议改用 JSONL）

---

## 项目当前态势（3月1日）

### 已完成的重构
- Provider 重构：daming #492 — 完成
- Channel 重构 Phase 1：alexhoshina #662 — 完成
- Channel 重构 Phase 2：alexhoshina #877 (10,926行) — 2月27日合并
- Migrate 重构：lxowalle #910 — 2月28日合并

### 正在进行的重构
- **Tools 系统重构**：lxowalle PR #846（50个文件）— OPEN
- **Plugin 系统**：gh-xj PR #936-939（4个PR系列）— OPEN
- **Agent 系统重构**：alexhoshina Issue #772（roadmap）— 只有 issue，还没有 PR

### 我的行动记录
- 2月24日：在 #772 评论，将 PR #732 定位为 Agent 重构的 memory 子任务
- 3月1日：在 #295 评论，提出模型路由设计方案

---

## 战略方向

### 方向 1：智能模型路由（#295）— 主攻 ✅ 代码已完成

**为什么选这个**：
1. Zepan（创始人）亲自创建的 issue，roadmap 标签
2. 有大量社区讨论但零 PR
3. 独立模块 `pkg/routing/`，不碰任何重构区文件
4. 面试价值极高

**已完成（分支 feat/model-routing）**：
- `pkg/routing/features.go` — ExtractFeatures：5维结构评分，纯语言无关
- `pkg/routing/classifier.go` — Classifier 接口 + RuleClassifier（加权求和，上限 1.0）
- `pkg/routing/router.go` — Router：SelectModel，阈值默认 0.35
- `pkg/routing/router_test.go` — 34 个测试，全部通过
- `pkg/config/config.go` — RoutingConfig 添加到 AgentDefaults
- `pkg/agent/instance.go` — 预计算 Router + LightCandidates
- `pkg/agent/loop.go` — selectCandidates helper，turn 级别粘性路由

**3 个 commit，773 行新增，33 行修改，0 个新依赖**

**配置**：
```json
{
  "agents": {
    "defaults": {
      "model": "claude-sonnet-4-6",
      "routing": {
        "enabled": true,
        "light_model": "gemini-flash",
        "threshold": 0.35
      }
    }
  }
}
```

**下一步**：向上游 push 并开 PR，PR body 引用 issue #295

### 方向 2：JSONL Store 集成 — 等待时机

PR #732 已提交。等 Tools 重构 (#846) 合并后再做集成 PR。
已在 #772 评论建立关联。

### 方向 3：sessions CLI 子命令（#575）— 备选快速 PR

如果需要一个快速能合并的 PR 来积累信任：
- `picoclaw sessions list/clear/export`
- 不碰任何重构区文件
- 实用性强

---

## 需要避开的区域

| 区域 | 原因 |
|------|------|
| Tools 系统 | lxowalle PR #846 正在重构 |
| Plugin 系统 | gh-xj PR #936-939 正在做 |
| Channel 任何东西 | alexhoshina 刚完成大重构 |
| Provider 配置 | daming 已定型 |
| MCP | 两个竞争 PR (#282, #376) |
| Hooks 基础 | gh-xj #936 包含 pkg/hooks/ |
| AgentLoop 拆分 | SaiBalusu-usf PR #699 |
| Tool pair 修复 | QuietyAwe PR #871 |

---

## 关键人物（更新）

| 人 | GitHub | 角色 | 最近活动 |
|---|--------|------|---------|
| Zepan | @Zepan | 创始人 | #806 WebUI issue |
| daming | @yinwm | Provider/审核 | 审核 PR #877 |
| alexhoshina | @alexhoshina | Channel+Agent 重构 | #877 合并，#772 发起 |
| lxowalle | @lxowalle | Tools+审核 | #846 Tools重构中 |
| gh-xj | @gh-xj | Plugin 系统 | #936-939 四个 PR |
| nikolasdehor | @nikolasdehor | 社区活跃评论者 | 每个 issue 都有他 |
