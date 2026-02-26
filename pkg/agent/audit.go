package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type AuditFinding struct {
	TaskID         string `json:"task_id"`
	Category       string `json:"category"`
	Severity       string `json:"severity"`
	Message        string `json:"message"`
	Recommendation string `json:"recommendation,omitempty"`
}

type AuditReport struct {
	GeneratedAt time.Time      `json:"generated_at"`
	Lookback    time.Duration  `json:"lookback"`
	TotalTasks  int            `json:"total_tasks"`
	Findings    []AuditFinding `json:"findings"`
}

func (r *AuditReport) FormatMessage() string {
	if r == nil {
		return "Task audit report unavailable."
	}

	var b strings.Builder
	b.WriteString("Task Audit Report\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n", r.GeneratedAt.Format("2006-01-02 15:04:05")))
	b.WriteString(fmt.Sprintf("Lookback: %dm\n", int(r.Lookback.Minutes())))
	b.WriteString(fmt.Sprintf("Tasks scanned: %d\n", r.TotalTasks))
	b.WriteString(fmt.Sprintf("Findings: %d\n", len(r.Findings)))

	if len(r.Findings) == 0 {
		b.WriteString("\nNo issues detected.")
		return b.String()
	}

	limit := len(r.Findings)
	if limit > 10 {
		limit = 10
	}
	b.WriteString("\nTop findings:\n")
	for i := 0; i < limit; i++ {
		f := r.Findings[i]
		b.WriteString(fmt.Sprintf(
			"%d. [%s/%s] task=%s - %s\n",
			i+1,
			strings.ToUpper(f.Severity),
			f.Category,
			f.TaskID,
			f.Message,
		))
		if f.Recommendation != "" {
			b.WriteString(fmt.Sprintf("   Action: %s\n", f.Recommendation))
		}
	}
	if len(r.Findings) > limit {
		b.WriteString(fmt.Sprintf("... and %d more findings.", len(r.Findings)-limit))
	}
	return b.String()
}

type supervisorReview struct {
	Score  float64 `json:"score"`
	Issues []struct {
		Category string `json:"category"`
		Severity string `json:"severity"`
		Message  string `json:"message"`
	} `json:"issues"`
}

func (al *AgentLoop) runAuditLoop(ctx context.Context) {
	if al.cfg == nil || !al.cfg.Audit.Enabled {
		return
	}

	intervalMinutes := al.cfg.Audit.IntervalMinutes
	if intervalMinutes <= 0 {
		intervalMinutes = 30
	}
	interval := time.Duration(intervalMinutes) * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run once shortly after startup.
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			al.executeAuditCycle(ctx)
		case <-ticker.C:
			al.executeAuditCycle(ctx)
		}
	}
}

func (al *AgentLoop) executeAuditCycle(ctx context.Context) {
	report, err := al.RunTaskAudit(ctx)
	if err != nil {
		logger.WarnCF("audit", "Task audit failed", map[string]any{"error": err.Error()})
		return
	}
	if report == nil {
		return
	}

	logger.InfoCF("audit", "Task audit completed", map[string]any{
		"tasks_scanned": report.TotalTasks,
		"findings":      len(report.Findings),
	})

	if len(report.Findings) == 0 {
		return
	}

	al.applyAutoRemediation(ctx, report)
	al.publishAuditReport(report)
}

func (al *AgentLoop) RunTaskAudit(ctx context.Context) (*AuditReport, error) {
	if al.taskLedger == nil || al.cfg == nil {
		return nil, nil
	}

	lookback := time.Duration(al.cfg.Audit.LookbackMinutes) * time.Minute
	if lookback <= 0 {
		lookback = 3 * time.Hour
	}

	records := al.taskLedger.ListSince(time.Now().Add(-lookback))
	report := &AuditReport{
		GeneratedAt: time.Now(),
		Lookback:    lookback,
		TotalTasks:  len(records),
		Findings:    make([]AuditFinding, 0),
	}
	nowMS := time.Now().UnixMilli()

	timeoutSeconds := al.cfg.Orchestration.DefaultTaskTimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 180
	}
	timeoutMS := int64(timeoutSeconds) * 1000
	retryLimit := al.cfg.Orchestration.RetryLimitPerTask
	if retryLimit < 0 {
		retryLimit = 0
	}
	inconsistencyPolicy := strings.ToLower(strings.TrimSpace(al.cfg.Audit.InconsistencyPolicy))
	if inconsistencyPolicy == "" {
		inconsistencyPolicy = "strict"
	}

	for _, record := range records {
		switch record.Status {
		case tools.TaskStatusPlanned:
			overdue := false
			if record.DeadlineAtMS != nil && nowMS > *record.DeadlineAtMS {
				overdue = true
			}
			if !overdue && nowMS-record.CreatedAtMS > timeoutMS {
				overdue = true
			}
			if overdue {
				report.Findings = append(report.Findings, AuditFinding{
					TaskID:         record.ID,
					Category:       "missed",
					Severity:       "high",
					Message:        "Task is still planned but appears overdue.",
					Recommendation: "Rerun or escalate this task.",
				})
			}
		case tools.TaskStatusRunning:
			if nowMS-record.UpdatedAtMS > timeoutMS {
				report.Findings = append(report.Findings, AuditFinding{
					TaskID:         record.ID,
					Category:       "missed",
					Severity:       "high",
					Message:        "Task is running past expected timeout.",
					Recommendation: "Cancel and retry with a narrower scope.",
				})
			}
		case tools.TaskStatusCompleted:
			if strings.TrimSpace(record.Result) == "" {
				report.Findings = append(report.Findings, AuditFinding{
					TaskID:         record.ID,
					Category:       "quality",
					Severity:       "medium",
					Message:        "Task completed but produced an empty result.",
					Recommendation: "Re-run task and require explicit output fields.",
				})
			}

			if len(record.Evidence) == 0 && inconsistencyPolicy == "strict" {
				report.Findings = append(report.Findings, AuditFinding{
					TaskID:         record.ID,
					Category:       "inconsistency",
					Severity:       "medium",
					Message:        "No execution evidence was captured for a completed task.",
					Recommendation: "Re-run with trace capture enabled.",
				})
			}
		case tools.TaskStatusFailed:
			if record.RetryCount < retryLimit {
				report.Findings = append(report.Findings, AuditFinding{
					TaskID:         record.ID,
					Category:       "missed",
					Severity:       "medium",
					Message:        "Task failed and still has retry budget.",
					Recommendation: "Retry this task automatically or manually.",
				})
			}
		}
	}

	modelFindings, err := al.supervisorModelAudit(ctx, records)
	if err != nil {
		logger.WarnCF("audit", "Supervisor model audit skipped", map[string]any{"error": err.Error()})
	} else {
		report.Findings = append(report.Findings, modelFindings...)
	}

	return report, nil
}

func (al *AgentLoop) supervisorModelAudit(
	ctx context.Context,
	records []tools.TaskLedgerEntry,
) ([]AuditFinding, error) {
	if al.cfg == nil || !al.cfg.Audit.Supervisor.Enabled {
		return nil, nil
	}
	modelCfg := al.cfg.Audit.Supervisor.Model
	if modelCfg == nil || strings.TrimSpace(modelCfg.Primary) == "" {
		return nil, fmt.Errorf("audit supervisor model is not configured")
	}

	provider, modelID, err := al.createProviderForModelAlias(modelCfg.Primary)
	if err != nil {
		return nil, err
	}
	if provider == nil || strings.TrimSpace(modelID) == "" {
		return nil, fmt.Errorf("unable to initialize supervisor provider")
	}
	if closable, ok := provider.(providers.StatefulProvider); ok {
		defer closable.Close()
	}

	minConfidence := al.cfg.Audit.MinConfidence
	if minConfidence <= 0 {
		minConfidence = 0.75
	}

	options := map[string]any{}
	if al.cfg.Audit.Supervisor.Temperature != nil {
		options["temperature"] = *al.cfg.Audit.Supervisor.Temperature
	}
	if al.cfg.Audit.Supervisor.MaxTokens > 0 {
		options["max_tokens"] = al.cfg.Audit.Supervisor.MaxTokens
	}

	findings := make([]AuditFinding, 0)
	for _, record := range records {
		if record.Status != tools.TaskStatusCompleted {
			continue
		}
		review, err := al.reviewTaskWithSupervisor(ctx, provider, modelID, options, record)
		if err != nil {
			continue
		}

		if review.Score < minConfidence {
			findings = append(findings, AuditFinding{
				TaskID:         record.ID,
				Category:       "quality",
				Severity:       "medium",
				Message:        fmt.Sprintf("Supervisor confidence %.2f is below threshold %.2f.", review.Score, minConfidence),
				Recommendation: "Rerun task with stricter acceptance criteria.",
			})
		}
		for _, issue := range review.Issues {
			category := strings.TrimSpace(strings.ToLower(issue.Category))
			if category == "" {
				category = "quality"
			}
			severity := strings.TrimSpace(strings.ToLower(issue.Severity))
			if severity == "" {
				severity = "medium"
			}
			findings = append(findings, AuditFinding{
				TaskID:         record.ID,
				Category:       category,
				Severity:       severity,
				Message:        issue.Message,
				Recommendation: "Investigate and re-run affected parts of the task.",
			})
		}
	}
	return findings, nil
}

func (al *AgentLoop) reviewTaskWithSupervisor(
	ctx context.Context,
	provider providers.LLMProvider,
	modelID string,
	options map[string]any,
	record tools.TaskLedgerEntry,
) (*supervisorReview, error) {
	taskJSON, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, err
	}

	systemPrompt := `You are a strict operations auditor.
Review the task execution data and output ONLY JSON:
{"score":0.0,"issues":[{"category":"quality|inconsistency|missed","severity":"low|medium|high","message":"..."}]}`
	userPrompt := fmt.Sprintf("Task data:\n%s", string(taskJSON))

	resp, err := provider.Chat(ctx, []providers.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}, nil, modelID, options)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(resp.Content) == "" {
		return nil, fmt.Errorf("empty supervisor response")
	}

	parsed, err := parseSupervisorReview(resp.Content)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func parseSupervisorReview(raw string) (*supervisorReview, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty review content")
	}
	var review supervisorReview
	if err := json.Unmarshal([]byte(raw), &review); err == nil {
		return &review, nil
	}

	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no json object in review response")
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &review); err != nil {
		return nil, err
	}
	return &review, nil
}

func (al *AgentLoop) applyAutoRemediation(ctx context.Context, report *AuditReport) {
	if report == nil || len(report.Findings) == 0 || al.taskLedger == nil || al.cfg == nil {
		return
	}

	mode := strings.ToLower(strings.TrimSpace(al.cfg.Audit.AutoRemediation))
	if mode == "" || mode == "disabled" || mode == "off" || mode == "none" {
		return
	}

	if mode == "safe_only" {
		for _, finding := range report.Findings {
			if finding.Category != "missed" {
				continue
			}
			_ = al.taskLedger.AddRemediation(finding.TaskID, tools.TaskRemediation{
				Action: "notify",
				Status: "queued",
				Note:   finding.Message,
			})
		}
		return
	}

	// retry/auto-fix modes
	// - retry_missed: only retry tasks in "missed" category
	// - retry_all: retry missed + rerun quality/inconsistency findings
	// - retry: alias for retry_missed
	switch mode {
	case "retry":
		mode = "retry_missed"
	case "retry_missed", "retry_all":
	default:
		// Unknown mode: fail closed.
		return
	}

	maxPerCycle := al.cfg.Audit.MaxAutoRemediationsPerCycle
	if maxPerCycle <= 0 {
		maxPerCycle = 3
	}
	cooldownMinutes := al.cfg.Audit.RemediationCooldownMinutes
	if cooldownMinutes <= 0 {
		cooldownMinutes = 10
	}
	cooldownMS := int64(cooldownMinutes) * 60 * 1000
	retryLimit := al.cfg.Orchestration.RetryLimitPerTask
	if retryLimit < 0 {
		retryLimit = 0
	}

	targetAgentID := strings.TrimSpace(al.cfg.Audit.RemediationAgentID)
	defaultAgent := al.registry.GetDefaultAgent()
	if defaultAgent == nil || defaultAgent.SubagentManager == nil {
		return
	}
	if targetAgentID != "" {
		normalized, ok := al.registry.GetAgent(targetAgentID)
		if !ok || normalized == nil {
			logger.WarnCF("audit", "Remediation agent not found; falling back to default agent", map[string]any{
				"remediation_agent_id": targetAgentID,
			})
			targetAgentID = ""
		} else {
			targetAgentID = normalized.ID
		}
		if targetAgentID != "" && targetAgentID != defaultAgent.ID && !al.registry.CanSpawnSubagent(defaultAgent.ID, targetAgentID) {
			logger.WarnCF("audit", "Remediation agent not allowed by subagent allowlist; falling back to default agent", map[string]any{
				"parent_agent_id":      defaultAgent.ID,
				"remediation_agent_id": targetAgentID,
			})
			targetAgentID = ""
		}
	}

	nowMS := time.Now().UnixMilli()
	thresholdMS := nowMS - cooldownMS
	spawned := 0
	for _, finding := range report.Findings {
		if spawned >= maxPerCycle {
			return
		}

		if strings.TrimSpace(finding.TaskID) == "" {
			continue
		}
		// Decide whether this finding should trigger an automatic rerun.
		switch finding.Category {
		case "missed":
			// always eligible in retry modes
		case "quality", "inconsistency":
			if mode != "retry_all" {
				continue
			}
		default:
			continue
		}

		entry, ok := al.taskLedger.Get(finding.TaskID)
		if !ok {
			continue
		}
		if retryLimit > 0 && entry.RetryCount >= retryLimit {
			continue
		}

		intent := strings.TrimSpace(entry.Intent)
		if intent == "" {
			_ = al.taskLedger.AddRemediation(entry.ID, tools.TaskRemediation{
				Action: "retry",
				Status: "skipped",
				Note:   "missing task intent; cannot auto-retry",
			})
			continue
		}

		if hasRecentRetryRemediation(entry.Remediations, thresholdMS) {
			continue
		}

		originChannel, originChatID := al.resolveRemediationDestination(entry)
		if originChannel == "" || originChatID == "" {
			continue
		}

		retryTask := buildRetryTask(finding, entry)
		label := fmt.Sprintf("audit-%s:%s", finding.Category, entry.ID)

		taskInfo, err := defaultAgent.SubagentManager.SpawnTask(
			ctx,
			retryTask,
			label,
			targetAgentID,
			originChannel,
			originChatID,
			nil,
		)
		if err != nil {
			_ = al.taskLedger.AddRemediation(entry.ID, tools.TaskRemediation{
				Action: "retry",
				Status: "error",
				Note:   fmt.Sprintf("failed to spawn retry task: %v", err),
			})
			continue
		}

		_ = al.taskLedger.IncrementRetry(entry.ID)
		_ = al.taskLedger.AddRemediation(entry.ID, tools.TaskRemediation{
			Action: "retry",
			Status: "spawned",
			Note:   fmt.Sprintf("spawned %s (agent_id=%s)", taskInfo.ID, strings.TrimSpace(targetAgentID)),
		})
		spawned++
	}
}

func hasRecentRetryRemediation(remediations []tools.TaskRemediation, thresholdMS int64) bool {
	for _, r := range remediations {
		if strings.ToLower(strings.TrimSpace(r.Action)) != "retry" {
			continue
		}
		if r.CreatedAtMS < thresholdMS {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(r.Status)) {
		case "queued", "spawned", "running", "skipped":
			return true
		}
	}
	return false
}

func (al *AgentLoop) resolveRemediationDestination(entry tools.TaskLedgerEntry) (string, string) {
	channel := strings.TrimSpace(entry.OriginChannel)
	chatID := strings.TrimSpace(entry.OriginChatID)
	if channel != "" && chatID != "" && !constants.IsInternalChannel(channel) {
		return channel, chatID
	}

	channel, chatID = al.resolveAuditDestination()
	if channel == "" || chatID == "" || constants.IsInternalChannel(channel) {
		return "", ""
	}
	return channel, chatID
}

func buildRetryTask(finding AuditFinding, entry tools.TaskLedgerEntry) string {
	intent := strings.TrimSpace(entry.Intent)
	if intent == "" {
		return ""
	}

	reason := strings.TrimSpace(finding.Message)
	if reason == "" {
		reason = "Task requires follow-up."
	}

	var b strings.Builder
	b.WriteString("You are running an automatic remediation retry for a previously problematic task.\n")
	b.WriteString("Be concise and deliver a complete result.\n\n")
	b.WriteString(fmt.Sprintf("Original task id: %s\n", entry.ID))
	b.WriteString(fmt.Sprintf("Original status: %s\n", entry.Status))
	b.WriteString(fmt.Sprintf("Finding category: %s\n", finding.Category))
	b.WriteString(fmt.Sprintf("Reason: %s\n\n", reason))
	b.WriteString("Task:\n")
	b.WriteString(intent)
	b.WriteString("\n\nAcceptance criteria:\n")
	switch finding.Category {
	case "quality":
		b.WriteString("- Produce a non-empty result.\n- Include concrete deliverables.\n")
	case "inconsistency":
		b.WriteString("- If tools are required, use them and complete the task end-to-end.\n")
	default:
		b.WriteString("- Complete the task end-to-end.\n")
	}
	return b.String()
}

func (al *AgentLoop) publishAuditReport(report *AuditReport) {
	if report == nil || len(report.Findings) == 0 || al.bus == nil {
		return
	}
	channel, chatID := al.resolveAuditDestination()
	if channel == "" || chatID == "" || constants.IsInternalChannel(channel) {
		return
	}

	al.bus.PublishOutbound(bus.OutboundMessage{
		Channel: channel,
		ChatID:  chatID,
		Content: report.FormatMessage(),
	})
}

func (al *AgentLoop) resolveAuditDestination() (string, string) {
	if al.cfg == nil {
		return "", ""
	}

	notify := strings.TrimSpace(al.cfg.Audit.NotifyChannel)
	last := ""
	if al.state != nil {
		last = al.state.GetLastChannel()
	}
	lastChannel, lastChatID := splitChannelChat(last)

	if notify == "" || notify == "last_active" {
		return lastChannel, lastChatID
	}
	if strings.Contains(notify, ":") {
		return splitChannelChat(notify)
	}
	if lastChatID != "" {
		return notify, lastChatID
	}
	return "", ""
}

func splitChannelChat(value string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	channel := strings.TrimSpace(parts[0])
	chatID := strings.TrimSpace(parts[1])
	if channel == "" || chatID == "" {
		return "", ""
	}
	return channel, chatID
}

func (al *AgentLoop) createProviderForModelAlias(modelAlias string) (providers.LLMProvider, string, error) {
	if al.cfg == nil {
		return nil, "", fmt.Errorf("config is nil")
	}
	modelCfg, err := al.cfg.GetModelConfig(modelAlias)
	if err != nil {
		return nil, "", err
	}
	cfgCopy := *modelCfg
	if cfgCopy.Workspace == "" {
		cfgCopy.Workspace = al.cfg.WorkspacePath()
	}
	return providers.CreateProviderFromConfig(&cfgCopy)
}
