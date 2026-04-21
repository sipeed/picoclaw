package agent

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels/pico"
)

func TestParsePlanTodoFallback_UsesPhaseHeadings(t *testing.T) {
	payload := parsePlanTodoFallback(&bus.InboundContext{
		Channel: "pico",
		Raw: map[string]string{
			pico.PayloadKeyMode: pico.ChatModePlan,
		},
	}, `# 项目规划

先给出实施方案。

## 任务拆解

### 阶段 1: 调研与设计
- 分析现有实现
- 明确目标体验

### 阶段 2: 后端实现
- 增加 structured todo

### 阶段 3: 测试与验证
- 补测试
- 手工验证
`)

	if payload == nil {
		t.Fatal("expected structured payload")
	}
	if payload["type"] != "todo" {
		t.Fatalf("type = %v, want todo", payload["type"])
	}
	items, ok := payload["items"].([]map[string]any)
	if !ok {
		t.Fatalf("items = %#v, want []map[string]any", payload["items"])
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	if items[0]["title"] != "阶段 1: 调研与设计" {
		t.Fatalf("first title = %v, want 阶段 1: 调研与设计", items[0]["title"])
	}
	if items[0]["status"] != "in-progress" {
		t.Fatalf("first status = %v, want in-progress", items[0]["status"])
	}
	if items[1]["status"] != "not-started" {
		t.Fatalf("second status = %v, want not-started", items[1]["status"])
	}
}

func TestParsePlanTodoFallback_UsesBulletFallback(t *testing.T) {
	payload := parsePlanTodoFallback(&bus.InboundContext{
		Channel: "pico",
		Raw: map[string]string{
			pico.PayloadKeyMode: pico.ChatModePlan,
		},
	}, `Plan:
- [x] Review current plan mode
- Implement structured todo renderer
- Validate in browser
`)

	if payload == nil {
		t.Fatal("expected structured payload")
	}
	items, ok := payload["items"].([]map[string]any)
	if !ok || len(items) != 3 {
		t.Fatalf("items = %#v, want 3 entries", payload["items"])
	}
	if items[0]["status"] != "completed" {
		t.Fatalf("first status = %v, want completed", items[0]["status"])
	}
	if items[1]["status"] != "not-started" {
		t.Fatalf("second status = %v, want not-started", items[1]["status"])
	}
}

func TestAttachPlanTodoFallback_SkipsNonPlanMessages(t *testing.T) {
	ctx := bus.InboundContext{Channel: "pico", Raw: map[string]string{}}
	attachPlanTodoFallback(&ctx, "- one\n- two")
	if len(ctx.Raw) != 0 {
		t.Fatalf("raw = %#v, want unchanged", ctx.Raw)
	}
}
