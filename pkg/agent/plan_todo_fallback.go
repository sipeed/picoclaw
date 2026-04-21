package agent

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels/pico"
)

var (
	planHeadingRe  = regexp.MustCompile(`^\s*(#{1,6})\s+(.+?)\s*$`)
	planCheckboxRe = regexp.MustCompile(`^\s*[-*+]\s*\[( |x|X)\]\s+(.+?)\s*$`)
	planBulletRe   = regexp.MustCompile(`^\s*[-*+]\s+(.+?)\s*$`)
	planNumberRe   = regexp.MustCompile(`^\s*\d+[\.)]\s+(.+?)\s*$`)
)

func attachPlanTodoFallback(outboundCtx *bus.InboundContext, response string) {
	if outboundCtx == nil {
		return
	}
	structured := parsePlanTodoFallback(outboundCtx, response)
	if structured == nil {
		return
	}
	rawStructured, err := json.Marshal(structured)
	if err != nil {
		return
	}
	if outboundCtx.Raw == nil {
		outboundCtx.Raw = make(map[string]string, 1)
	}
	outboundCtx.Raw[metadataKeyStructuredData] = string(rawStructured)
}

func parsePlanTodoFallback(inboundCtx *bus.InboundContext, response string) map[string]any {
	if inboundCtx == nil || strings.TrimSpace(response) == "" {
		return nil
	}
	if inboundCtx.Channel != "pico" {
		return nil
	}
	if strings.ToLower(strings.TrimSpace(inboundCtx.Raw[pico.PayloadKeyMode])) != pico.ChatModePlan {
		return nil
	}

	title, content, items := extractPlanTodoItems(response)
	if len(items) == 0 {
		return nil
	}

	payload := map[string]any{
		"type":  "todo",
		"title": title,
		"items": items,
	}
	if content != "" {
		payload["content"] = content
	}
	return payload
}

func extractPlanTodoItems(response string) (string, string, []map[string]any) {
	lines := strings.Split(strings.ReplaceAll(response, "\r\n", "\n"), "\n")
	title := "Plan"
	content := ""
	headingTasks := make([]map[string]any, 0, 8)
	listTasks := make([]map[string]any, 0, 12)
	firstHeadingSeen := false
	firstParagraphSeen := false

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || line == "---" {
			continue
		}

		if matches := planHeadingRe.FindStringSubmatch(line); len(matches) == 3 {
			headingText := cleanPlanLine(matches[2])
			if headingText == "" {
				continue
			}
			if !firstHeadingSeen {
				title = headingText
				firstHeadingSeen = true
				continue
			}
			if isTaskHeading(headingText) {
				headingTasks = append(headingTasks, map[string]any{
					"title":  headingText,
					"status": "not-started",
				})
			}
			continue
		}

		if !firstParagraphSeen && !looksLikeListLine(line) {
			content = cleanPlanLine(line)
			firstParagraphSeen = content != ""
		}

		if matches := planCheckboxRe.FindStringSubmatch(line); len(matches) == 3 {
			status := "not-started"
			if strings.EqualFold(matches[1], "x") {
				status = "completed"
			}
			if item := cleanPlanLine(matches[2]); item != "" {
				listTasks = append(listTasks, map[string]any{
					"title":  item,
					"status": status,
				})
			}
			continue
		}
		if matches := planBulletRe.FindStringSubmatch(line); len(matches) == 2 {
			if item := cleanPlanLine(matches[1]); item != "" {
				listTasks = append(listTasks, map[string]any{
					"title":  item,
					"status": inferPlanStatus(item),
				})
			}
			continue
		}
		if matches := planNumberRe.FindStringSubmatch(line); len(matches) == 2 {
			if item := cleanPlanLine(matches[1]); item != "" {
				listTasks = append(listTasks, map[string]any{
					"title":  item,
					"status": inferPlanStatus(item),
				})
			}
		}
	}

	items := headingTasks
	if len(items) == 0 {
		items = listTasks
	}
	if len(items) == 0 {
		return title, content, nil
	}
	if len(items) > 8 {
		items = items[:8]
	}

	hasExplicitProgress := false
	for _, item := range items {
		status, _ := item["status"].(string)
		if status == "in-progress" || status == "completed" {
			hasExplicitProgress = true
			break
		}
	}
	if !hasExplicitProgress && len(items) > 0 {
		items[0]["status"] = "in-progress"
	}

	return title, content, items
}

func isTaskHeading(text string) bool {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "项目目标") || strings.Contains(lower, "任务拆解") || strings.Contains(lower, "core goal") {
		return false
	}
	return strings.Contains(lower, "阶段") ||
		strings.Contains(lower, "phase") ||
		strings.Contains(lower, "step") ||
		strings.Contains(lower, "milestone") ||
		strings.Contains(lower, "实现") ||
		strings.Contains(lower, "测试") ||
		strings.Contains(lower, "验证") ||
		strings.Contains(lower, "优化")
}

func looksLikeListLine(line string) bool {
	return planCheckboxRe.MatchString(line) || planBulletRe.MatchString(line) || planNumberRe.MatchString(line)
}

func inferPlanStatus(text string) string {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "completed") || strings.Contains(lower, "done") || strings.Contains(lower, "已完成") {
		return "completed"
	}
	if strings.Contains(lower, "in-progress") || strings.Contains(lower, "running") || strings.Contains(lower, "进行中") {
		return "in-progress"
	}
	return "not-started"
}

func cleanPlanLine(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Trim(text, "*")
	text = strings.TrimSpace(text)
	text = strings.Trim(text, "`")
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "__", "")
	return strings.TrimSpace(text)
}
