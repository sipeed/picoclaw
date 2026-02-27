# Issue #783 调研与修复执行文档

## 1. 问题澄清（已确认）

- 现象：当 `agents.*.model.primary/fallbacks` 使用 `model_name` 别名（如 `step-3.5-flash`）时，fallback 链路将别名当作真实 `provider/model` 解析，导致 `provider` 可能为空、`model` 可能错误。
- 根因：`ResolveCandidates` 仅对字符串做 `ParseModelRef`，未先通过 `model_list` 将别名映射到真实 `model` 字段。
- 影响：
  - fallback 执行可能把别名直接发给 OpenAI-compatible provider，触发 `Unknown Model`。
  - `defaults.provider` 为空时，日志出现 `provider=` 空值。

## 2. 本次目标

- 修复 fallback 候选解析：优先通过 `model_list` 解析别名。
- 兼容旧行为：若未命中 `model_list`，继续走原有 `ParseModelRef` 兜底。
- 补充测试：覆盖别名、嵌套路径模型（如 `openrouter/stepfun/...`）、空默认 provider。
- 验证代码风格：与当前仓库风格保持一致（命名、错误处理、测试结构）。

## 3. 联网最佳实践调研结论（已完成）

- [x] 查阅 OpenAI-compatible 网关（如 OpenRouter）对 `model` 字段的推荐处理。
- [x] 查阅多 provider/fallback 设计最佳实践（候选解析、日志可观测性）。
- [x] 将外部建议映射为本仓库可执行约束。

外部参考要点（来自 OpenRouter/LiteLLM/Cloudflare AI Gateway 等官方文档）：

- 优先显式配置，不依赖字符串切分推断 provider。
- 对网关模型标识应保留完整路径语义，避免截断导致 Unknown Model。
- fallback 与 primary 应复用同一解析策略，避免“主路径正确、降级路径错误”。

参考链接：

- OpenRouter Provider Routing: https://openrouter.ai/docs/guides/routing/provider-selection
- OpenRouter Model Fallbacks: https://openrouter.ai/docs/guides/routing/model-fallbacks
- OpenRouter Chat Completion API: https://openrouter.ai/docs/api-reference/chat-completion
- LiteLLM Router Architecture: https://docs.litellm.ai/docs/router_architecture
- Cloudflare AI Gateway Chat Completion: https://developers.cloudflare.com/ai-gateway/usage/chat-completion/

与本仓库对应的可执行约束：

- 在 fallback candidate 构建阶段先做 `model_name -> model_list.model` 映射。
- 未命中映射时保留旧解析行为，保证兼容性。
- 用新增测试锁定“别名 + 嵌套模型路径 + 空默认 provider”场景。

## 4. 实施步骤（顺序执行）

- [x] Step 1: 对齐现有代码模式，定位最小改动点（`pkg/agent` + `pkg/providers`）。
- [x] Step 2: 实现“基于 model_list 的 fallback 候选解析”。
- [x] Step 3: 增加/更新单元测试，覆盖 issue 场景。
- [x] Step 4: 代码风格一致性复核（与现有文件风格对照）。
- [x] Step 5: 运行质量门禁（LSP + `make check`）。

## 5. 执行记录

- 状态：已完成
- 已完成改动：
  - `pkg/providers/fallback.go`：新增 `ResolveCandidatesWithLookup`，并保持 `ResolveCandidates` 向后兼容。
  - `pkg/agent/instance.go`：在构建 fallback candidates 前，优先通过 `model_list` 解析别名，并对无协议模型补齐默认 `openai/` 前缀后再解析。
  - `pkg/providers/fallback_test.go`：新增别名解析与去重测试。
  - `pkg/agent/instance_test.go`：新增 agent 侧别名解析到嵌套模型路径、无协议模型解析测试。
- 风格对齐检查（完成）：与 `pkg/providers/fallback_test.go`、`pkg/providers/model_ref_test.go` 现有模式一致。
- 质量验证（完成）：先 `make generate`，后 `make check` 全量通过。
