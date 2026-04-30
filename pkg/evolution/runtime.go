package evolution

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/skills"
)

var ErrApplyDraftFailed = errors.New("apply draft failed")

type RuntimeOptions struct {
	Config              config.EvolutionConfig
	Now                 func() time.Time
	Store               *Store
	Organizer           *Organizer
	SuccessJudge        SuccessJudge
	SkillsRecaller      *SkillsRecaller
	DraftGenerator      DraftGenerator
	GeneratorFactory    func(workspace string) DraftGenerator
	SuccessJudgeFactory func(workspace string) SuccessJudge
	Applier             *Applier
	ApplierFactory      func(workspace string) *Applier
}

type Runtime struct {
	cfg                 config.EvolutionConfig
	mu                  sync.Mutex
	now                 func() time.Time
	writer              *CaseWriter
	store               *Store
	organizer           *Organizer
	successJudge        SuccessJudge
	skillsRecaller      *SkillsRecaller
	draftGenerator      DraftGenerator
	generatorFactory    func(workspace string) DraftGenerator
	successJudgeFactory func(workspace string) SuccessJudge
	applier             *Applier
	applierFactory      func(workspace string) *Applier
}

type TurnCaseInput struct {
	Workspace             string
	WorkspaceID           string
	TurnID                string
	SessionKey            string
	AgentID               string
	Status                string
	UserMessage           string
	FinalContent          string
	ToolKinds             []string
	ToolExecutions        []ToolExecutionRecord
	ActiveSkillNames      []string
	AttemptedSkillNames   []string
	FinalSuccessfulPath   []string
	SkillContextSnapshots []SkillContextSnapshot
}

func NewRuntime(opts RuntimeOptions) (*Runtime, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}

	organizer := opts.Organizer
	if organizer == nil {
		organizer = NewOrganizer(OrganizerOptions{
			MinCaseCount:   opts.Config.MinCaseCount,
			MinSuccessRate: opts.Config.MinSuccessRate,
			Now:            now,
		})
	}

	return &Runtime{
		cfg:                 opts.Config,
		now:                 now,
		store:               opts.Store,
		organizer:           organizer,
		successJudge:        opts.SuccessJudge,
		skillsRecaller:      opts.SkillsRecaller,
		draftGenerator:      opts.DraftGenerator,
		generatorFactory:    opts.GeneratorFactory,
		successJudgeFactory: opts.SuccessJudgeFactory,
		applier:             opts.Applier,
		applierFactory:      opts.ApplierFactory,
	}, nil
}

func (rt *Runtime) FinalizeTurn(ctx context.Context, input TurnCaseInput) error {
	if rt == nil || !rt.cfg.Enabled || input.Workspace == "" || shouldSkipLearningRecord(input) {
		return nil
	}

	success := input.Status == "completed"
	skillUsage := buildSkillUsage(input)
	workspaceID := input.WorkspaceID
	if workspaceID == "" {
		workspaceID = input.Workspace
	}

	toolKinds := cloneToolKinds(input)
	record := LearningRecord{
		ID:                  input.TurnID,
		Kind:                RecordKindTask,
		WorkspaceID:         workspaceID,
		CreatedAt:           rt.now(),
		SessionKey:          input.SessionKey,
		TaskHash:            buildTaskHash(input, skillUsage, toolKinds),
		Summary:             buildRecordSummary(input),
		UserGoal:            summarizeText(input.UserMessage, 240),
		FinalOutput:         summarizeText(input.FinalContent, 240),
		Source:              map[string]any{"turn_id": input.TurnID, "session_key": input.SessionKey, "agent_id": input.AgentID},
		Status:              RecordStatus("new"),
		Success:             &success,
		ToolKinds:           toolKinds,
		ToolExecutions:      cloneToolExecutions(input.ToolExecutions),
		InitialSkillNames:   append([]string(nil), skillUsage.Initial...),
		AddedSkillNames:     append([]string(nil), skillUsage.Added...),
		UsedSkillNames:      append([]string(nil), skillUsage.Used...),
		AllLoadedSkillNames: append([]string(nil), skillUsage.All...),
		ActiveSkillNames:    append([]string(nil), skillUsage.Initial...),
		AttemptTrail:        buildAttemptTrail(skillUsage, success),
		Signals:             buildLearningSignals(skillUsage, success),
	}

	paths := NewPaths(input.Workspace, rt.cfg.StateDir)

	rt.mu.Lock()
	if rt.writer == nil || rt.writer.paths.RootDir != paths.RootDir {
		rt.writer = NewCaseWriter(paths)
	}
	writer := rt.writer
	rt.mu.Unlock()

	if err := writer.AppendCase(ctx, record); err != nil {
		return err
	}

	if err := rt.recordSkillUsage(input, success); err != nil {
		return err
	}

	logger.InfoCF("evolution", "Recorded hot path learning record", map[string]any{
		"workspace":   input.Workspace,
		"turn_id":     input.TurnID,
		"success":     success,
		"tool_count":  len(record.ToolExecutions),
		"used_skills": len(record.UsedSkillNames),
		"all_skills":  len(record.AllLoadedSkillNames),
	})
	return nil
}

type skillUsageRecord struct {
	Initial   []string
	Added     []string
	Used      []string
	All       []string
	Attempted []string
	Final     []string
}

func buildAttemptTrail(usage skillUsageRecord, success bool) *AttemptTrail {
	if len(usage.All) == 0 {
		return nil
	}

	trail := &AttemptTrail{
		AttemptedSkills: append([]string(nil), usage.Attempted...),
	}
	if success {
		finalPath := usage.Final
		if len(finalPath) == 0 {
			finalPath = usage.All
		}
		trail.FinalSuccessfulPath = append([]string(nil), finalPath...)
	}
	return trail
}

func buildTaskHash(input TurnCaseInput, usage skillUsageRecord, toolKinds []string) string {
	parts := make([]string, 0, len(usage.All)+len(toolKinds)+3)
	parts = append(parts, normalizedValues(usage.All)...)
	parts = append(parts, normalizedValues(toolKinds)...)
	if goal := summarizeText(input.UserMessage, 120); goal != "" {
		parts = append(parts, strings.ToLower(goal))
	}
	if strings.TrimSpace(input.Status) != "" {
		parts = append(parts, strings.ToLower(strings.TrimSpace(input.Status)))
	}
	if len(parts) == 0 {
		parts = append(parts, "empty")
	}

	sum := sha1.Sum([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:8])
}

func buildLearningSignals(usage skillUsageRecord, success bool) []string {
	if !success || len(usage.Used) == 0 {
		return nil
	}
	return []string{"potentially_learnable"}
}

func normalizedValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func buildRecordSummary(input TurnCaseInput) string {
	if goal := summarizeText(input.UserMessage, 160); goal != "" {
		return goal
	}
	return fmt.Sprintf("turn %s finished with status=%s", input.TurnID, input.Status)
}

func summarizeText(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if text == "" || maxLen <= 0 {
		return text
	}
	if utf8.RuneCountInString(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		runes := []rune(text)
		return string(runes[:maxLen])
	}
	runes := []rune(text)
	return string(runes[:maxLen-3]) + "..."
}

func cloneToolKinds(input TurnCaseInput) []string {
	if len(input.ToolKinds) > 0 {
		return append([]string(nil), input.ToolKinds...)
	}
	if len(input.ToolExecutions) == 0 {
		return nil
	}
	out := make([]string, 0, len(input.ToolExecutions))
	seen := make(map[string]struct{}, len(input.ToolExecutions))
	for _, exec := range input.ToolExecutions {
		name := strings.TrimSpace(exec.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func cloneToolExecutions(input []ToolExecutionRecord) []ToolExecutionRecord {
	if len(input) == 0 {
		return nil
	}
	out := make([]ToolExecutionRecord, 0, len(input))
	for _, exec := range input {
		name := strings.TrimSpace(exec.Name)
		if name == "" {
			continue
		}
		out = append(out, ToolExecutionRecord{
			Name:         name,
			Success:      exec.Success,
			ErrorSummary: strings.TrimSpace(exec.ErrorSummary),
			SkillNames:   uniqueTrimmedNames(exec.SkillNames),
		})
	}
	return out
}

func buildSkillUsage(input TurnCaseInput) skillUsageRecord {
	initial := uniqueTrimmedNames(input.ActiveSkillNames)
	if len(input.SkillContextSnapshots) > 0 {
		initial = uniqueTrimmedNames(input.SkillContextSnapshots[0].SkillNames)
	}
	all := append([]string(nil), initial...)
	added := make([]string, 0)
	seen := make(map[string]struct{}, len(all))
	for _, skillName := range all {
		seen[strings.ToLower(skillName)] = struct{}{}
	}

	appendNew := func(skillNames []string) {
		for _, skillName := range uniqueTrimmedNames(skillNames) {
			key := strings.ToLower(skillName)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			added = append(added, skillName)
			all = append(all, skillName)
		}
	}

	for _, snapshot := range input.SkillContextSnapshots {
		appendNew(snapshot.SkillNames)
	}
	for _, exec := range input.ToolExecutions {
		appendNew(exec.SkillNames)
	}
	attempted := uniqueTrimmedNames(input.AttemptedSkillNames)
	if len(attempted) == 0 {
		attempted = append([]string(nil), all...)
	} else {
		appendNew(attempted)
	}
	final := uniqueTrimmedNames(input.FinalSuccessfulPath)
	if len(final) > 0 {
		appendNew(final)
	}
	if len(attempted) == 0 {
		attempted = append([]string(nil), all...)
	}
	if len(final) == 0 {
		final = append([]string(nil), added...)
	}

	return skillUsageRecord{
		Initial:   initial,
		Added:     append([]string(nil), added...),
		Used:      append([]string(nil), added...),
		All:       all,
		Attempted: append([]string(nil), attempted...),
		Final:     append([]string(nil), final...),
	}
}

func shouldSkipLearningRecord(input TurnCaseInput) bool {
	if strings.EqualFold(strings.TrimSpace(input.SessionKey), "heartbeat") {
		return true
	}
	return false
}

func uniqueTrimmedNames(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func (rt *Runtime) RunColdPathOnce(ctx context.Context, workspace string) error {
	if rt == nil || !rt.cfg.Enabled || workspace == "" {
		return nil
	}

	mode := rt.cfg.EffectiveMode()
	runID := fmt.Sprintf("%d", rt.now().UnixNano())
	if mode == "" || mode == "observe" {
		logger.InfoCF("evolution", "Skipped cold path run", map[string]any{
			"workspace": workspace,
			"mode":      mode,
			"run_id":    runID,
		})
		return nil
	}

	logger.InfoCF("evolution", "Started cold path run", map[string]any{
		"workspace": workspace,
		"mode":      mode,
		"run_id":    runID,
	})

	store := rt.storeForWorkspace(workspace)
	records, err := store.LoadLearningRecords()
	if err != nil {
		return err
	}
	logger.InfoCF("evolution", "Loaded evolution records", map[string]any{
		"workspace":    workspace,
		"record_count": len(records),
		"run_id":       runID,
	})

	admittedCount := 0
	newRuleCount := 0
	if rt.organizer != nil {
		recordsForOrganizer, err := rt.recordsForColdPath(ctx, workspace, records)
		if err != nil {
			return err
		}
		admittedCount = countTaskLearningRecords(recordsForOrganizer)
		logger.InfoCF("evolution", "Admitted task records for cold path", map[string]any{
			"workspace":       workspace,
			"admitted_tasks":  admittedCount,
			"organizer_input": len(recordsForOrganizer),
			"task_ids":        joinRecordIDs(recordsForOrganizer),
			"run_id":          runID,
		})
		rules, err := rt.organizer.BuildRules(recordsForOrganizer)
		if err != nil {
			return err
		}
		newRules := filterNewRules(records, rules, workspace)
		newRuleCount = len(newRules)
		logger.InfoCF("evolution", "Built learning patterns", map[string]any{
			"workspace":      workspace,
			"pattern_count":  len(rules),
			"new_patterns":   len(newRules),
			"admitted_tasks": admittedCount,
			"patterns":       summarizePatternRecords(rules),
			"run_id":         runID,
		})
		if len(newRules) > 0 {
			if err := store.AppendLearningRecords(newRules); err != nil {
				return err
			}
			records = append(records, newRules...)
		}
	}

	generator := rt.draftGeneratorForWorkspace(workspace)
	if generator == nil {
		logger.InfoCF("evolution", "Skipped drafting because no draft generator is available", map[string]any{
			"workspace": workspace,
			"run_id":    runID,
		})
		return rt.runLifecycleMaintenance(workspace, store, runID)
	}

	recaller := rt.skillsRecallerForWorkspace(workspace)
	applier := rt.applierForWorkspace(workspace)
	readyRules := filterReadyRules(records, workspace)
	if len(readyRules) == 0 {
		logger.InfoCF("evolution", "Finished cold path run without ready patterns", map[string]any{
			"workspace":      workspace,
			"record_count":   len(records),
			"new_patterns":   newRuleCount,
			"admitted_tasks": admittedCount,
			"run_id":         runID,
		})
		return rt.runLifecycleMaintenance(workspace, store, runID)
	}

	existingDrafts, err := store.LoadDrafts()
	if err != nil {
		return err
	}
	existingBySource := existingDraftSourceSet(existingDrafts, workspace)
	logger.InfoCF("evolution", "Selected ready patterns for drafting", map[string]any{
		"workspace":            workspace,
		"ready_patterns":       len(readyRules),
		"existing_draft_count": len(existingBySource),
		"ready_pattern_ids":    joinRecordIDs(readyRules),
		"ready_patterns_info":  summarizePatternRecords(readyRules),
		"run_id":               runID,
	})

	processedRules := 0
	for _, rule := range readyRules {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if _, exists := existingBySource[rule.ID]; exists {
			logger.InfoCF("evolution", "Skipped pattern because a non-quarantined draft already exists", map[string]any{
				"workspace":    workspace,
				"pattern_id":   rule.ID,
				"pattern_info": summarizePatternRecord(rule),
				"run_id":       runID,
			})
			continue
		}

		matches, err := recaller.RecallSimilarSkills(rule)
		if err != nil {
			return err
		}
		logger.InfoCF("evolution", "Generating skill draft", map[string]any{
			"workspace":           workspace,
			"pattern_id":          rule.ID,
			"matched_skill_count": len(matches),
			"pattern_info":        summarizePatternRecord(rule),
			"run_id":              runID,
		})

		draft, err := generator.GenerateDraft(ctx, rule, matches)
		if err != nil {
			return err
		}

		draft = rt.finalizeDraft(workspace, rule, matches, draft)
		logger.InfoCF("evolution", "Finalized skill draft", map[string]any{
			"workspace":    workspace,
			"pattern_id":   rule.ID,
			"draft_id":     draft.ID,
			"target_skill": draft.TargetSkillName,
			"change_kind":  string(draft.ChangeKind),
			"status":       string(draft.Status),
			"run_id":       runID,
		})
		if mode == "apply" && applier != nil && draft.Status == DraftStatusCandidate {
			logger.InfoCF("evolution", "Applying skill draft", map[string]any{
				"workspace":    workspace,
				"draft_id":     draft.ID,
				"target_skill": draft.TargetSkillName,
				"change_kind":  string(draft.ChangeKind),
				"run_id":       runID,
			})
			rollbackApply, err := applier.applyDraftWithRollback(ctx, workspace, draft)
			if err != nil {
				logger.WarnCF("evolution", "Skill draft apply failed", map[string]any{
					"workspace":    workspace,
					"draft_id":     draft.ID,
					"target_skill": draft.TargetSkillName,
					"error":        err.Error(),
					"run_id":       runID,
				})
				draft.Status = DraftStatusQuarantined
				draft.ScanFindings = appendUniqueStrings(draft.ScanFindings, fmt.Sprintf("apply failed: %v", err))
				if auditErr := rt.recordRollbackAudit(store, draft, err); auditErr != nil {
					draft.ScanFindings = appendUniqueStrings(draft.ScanFindings, fmt.Sprintf("rollback audit failed: %v", auditErr))
					if saveErr := store.SaveDrafts([]SkillDraft{draft}); saveErr != nil {
						return errorsJoin(fmt.Errorf("%w: %v", ErrApplyDraftFailed, err), auditErr, saveErr)
					}
					return errorsJoin(fmt.Errorf("%w: %v", ErrApplyDraftFailed, err), auditErr)
				}
				if saveErr := store.SaveDrafts([]SkillDraft{draft}); saveErr != nil {
					return errorsJoin(fmt.Errorf("%w: %v", ErrApplyDraftFailed, err), saveErr)
				}
				return fmt.Errorf("%w: %v", ErrApplyDraftFailed, err)
			}
			draft.Status = DraftStatusAccepted
			if err := rt.saveAppliedProfile(store, workspace, draft); err != nil {
				logger.WarnCF("evolution", "Skill profile save failed after apply", map[string]any{
					"workspace":    workspace,
					"draft_id":     draft.ID,
					"target_skill": draft.TargetSkillName,
					"error":        err.Error(),
					"run_id":       runID,
				})
				draft.Status = DraftStatusQuarantined
				draft.ScanFindings = appendUniqueStrings(draft.ScanFindings, fmt.Sprintf("profile save failed: %v", err))
				if rollbackErr := rollbackApply(); rollbackErr != nil {
					draft.ScanFindings = appendUniqueStrings(draft.ScanFindings, fmt.Sprintf("apply rollback failed: %v", rollbackErr))
					if saveErr := store.SaveDrafts([]SkillDraft{draft}); saveErr != nil {
						return errorsJoin(fmt.Errorf("%w: %v", ErrApplyDraftFailed, err), rollbackErr, saveErr)
					}
					return errorsJoin(fmt.Errorf("%w: %v", ErrApplyDraftFailed, err), rollbackErr)
				}
				if saveErr := store.SaveDrafts([]SkillDraft{draft}); saveErr != nil {
					return errorsJoin(fmt.Errorf("%w: %v", ErrApplyDraftFailed, err), saveErr)
				}
				return fmt.Errorf("%w: %v", ErrApplyDraftFailed, err)
			}
			logger.InfoCF("evolution", "Applied skill draft successfully", map[string]any{
				"workspace":    workspace,
				"draft_id":     draft.ID,
				"target_skill": draft.TargetSkillName,
				"run_id":       runID,
			})
		}

		if err := store.SaveDrafts([]SkillDraft{draft}); err != nil {
			return err
		}
		logger.InfoCF("evolution", "Saved skill draft", map[string]any{
			"workspace":    workspace,
			"draft_id":     draft.ID,
			"target_skill": draft.TargetSkillName,
			"status":       string(draft.Status),
			"run_id":       runID,
		})
		existingBySource[rule.ID] = struct{}{}
		processedRules++
	}

	logger.InfoCF("evolution", "Finished cold path run", map[string]any{
		"workspace":          workspace,
		"ready_patterns":     len(readyRules),
		"processed_patterns": processedRules,
		"new_patterns":       newRuleCount,
		"run_id":             runID,
	})
	return rt.runLifecycleMaintenance(workspace, store, runID)
}

func (rt *Runtime) recordsForColdPath(
	ctx context.Context,
	workspace string,
	records []LearningRecord,
) ([]LearningRecord, error) {
	out := make([]LearningRecord, 0, len(records))
	judge := rt.successJudgeForWorkspace(workspace)

	for _, record := range records {
		if !isTaskRecordKind(record.Kind) {
			out = append(out, record)
			continue
		}
		if record.WorkspaceID != workspace {
			continue
		}
		if !passesColdPathRuleFilter(record) {
			continue
		}
		if judge != nil {
			decision, err := judge.JudgeTaskRecord(ctx, record)
			if err != nil {
				return nil, err
			}
			if !decision.Success {
				continue
			}
		}
		out = append(out, record)
	}
	return out, nil
}

func passesColdPathRuleFilter(record LearningRecord) bool {
	if !isTaskRecordKind(record.Kind) {
		return false
	}
	if record.Success == nil || !*record.Success {
		return false
	}
	if strings.TrimSpace(record.UserGoal) == "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(record.SessionKey), "heartbeat") {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(record.FinalOutput), "HEARTBEAT_OK") {
		return false
	}
	if len(record.ToolExecutions) > 0 {
		allFailed := true
		for _, exec := range record.ToolExecutions {
			if exec.Success {
				allFailed = false
				break
			}
		}
		if allFailed {
			return false
		}
	}
	if len(record.ToolKinds) == 0 && len(record.UsedSkillNames) == 0 && strings.TrimSpace(record.FinalOutput) == "" {
		return false
	}
	return true
}

func (rt *Runtime) storeForWorkspace(workspace string) *Store {
	paths := NewPaths(workspace, rt.cfg.StateDir)

	rt.mu.Lock()
	defer rt.mu.Unlock()

	if rt.store == nil || rt.store.paths.RootDir != paths.RootDir {
		rt.store = NewStore(paths)
	}
	return rt.store
}

func (rt *Runtime) skillsRecallerForWorkspace(workspace string) *SkillsRecaller {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if rt.skillsRecaller == nil || rt.skillsRecaller.workspace != workspace {
		rt.skillsRecaller = NewSkillsRecaller(workspace)
	}
	return rt.skillsRecaller
}

func (rt *Runtime) draftGeneratorForWorkspace(workspace string) DraftGenerator {
	if rt.generatorFactory != nil {
		if generator := rt.generatorFactory(workspace); generator != nil {
			return generator
		}
	}
	if rt.draftGenerator != nil {
		return rt.draftGenerator
	}
	return NewDefaultDraftGenerator(workspace)
}

func (rt *Runtime) successJudgeForWorkspace(workspace string) SuccessJudge {
	if rt.successJudgeFactory != nil {
		if judge := rt.successJudgeFactory(workspace); judge != nil {
			return judge
		}
	}
	if rt.successJudge != nil {
		return rt.successJudge
	}
	return &HeuristicSuccessJudge{}
}

func (rt *Runtime) applierForWorkspace(workspace string) *Applier {
	if rt.applierFactory != nil {
		if applier := rt.applierFactory(workspace); applier != nil {
			return applier
		}
	}
	return rt.applier
}

func (rt *Runtime) finalizeDraft(workspace string, rule LearningRecord, matches []skills.SkillInfo, draft SkillDraft) SkillDraft {
	if draft.ID == "" {
		draft.ID = "draft-" + rule.ID
	}
	if draft.CreatedAt.IsZero() {
		draft.CreatedAt = rt.now()
	}
	draft.WorkspaceID = workspace
	draft.SourceRecordID = rule.ID
	if len(draft.MatchedSkillRefs) == 0 {
		draft.MatchedSkillRefs = collectSkillRefs(matches)
	}

	draft, normalizationNotes := rt.normalizeDraftForWorkspace(workspace, rule, draft)
	review := ReviewDraft(draft)
	draft.Status = review.Status
	draft.ReviewNotes = append([]string(nil), review.ReviewNotes...)
	draft.ReviewNotes = append(draft.ReviewNotes, normalizationNotes...)
	if len(review.Findings) == 0 {
		draft.ScanFindings = nil
		return draft
	}
	draft.ScanFindings = append([]string(nil), review.Findings...)
	return draft
}

func (rt *Runtime) normalizeDraftForWorkspace(
	workspace string,
	rule LearningRecord,
	draft SkillDraft,
) (SkillDraft, []string) {
	target := strings.TrimSpace(draft.TargetSkillName)
	if workspace == "" || target == "" {
		return draft, nil
	}

	notes := make([]string, 0, 4)
	if combinedTarget := inferCombinedSkillName(rule); combinedTarget != "" && combinedTarget != target {
		originalTarget := target
		draft.TargetSkillName = combinedTarget
		target = combinedTarget
		notes = append(notes, fmt.Sprintf(
			"retargeted draft from %q to combined shortcut skill %q because the winning path was a stable multi-skill chain",
			originalTarget,
			combinedTarget,
		))
	}

	skillPath := filepath.Join(workspace, "skills", target, "SKILL.md")
	_, err := os.Stat(skillPath)
	hasExisting := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return draft, notes
	}

	if combinedTarget := inferCombinedSkillName(rule); combinedTarget != "" && combinedTarget == target {
		draft.HumanSummary = buildCombinedSkillHumanSummary(target, rule, hasExisting)
		draft.PreferredEntryPath = []string{target}
		draft.AvoidPatterns = appendUniqueStrings(
			draft.AvoidPatterns,
			buildCombinedSkillAvoidPattern(target, rule),
		)
		if hasExisting {
			draft.ChangeKind = ChangeKindAppend
			draft.BodyOrPatch = synthesizeCombinedSkillAppendBody(target, draft, rule)
			notes = append(notes, "normalized combined shortcut draft to append onto the existing combined skill")
		} else {
			draft.ChangeKind = ChangeKindCreate
			draft.BodyOrPatch = synthesizeCombinedSkillDocument(target, draft, rule)
			notes = append(notes, "normalized combined shortcut draft to create a new standalone shortcut skill")
		}
		return draft, notes
	}

	if !hasExisting {
		switch draft.ChangeKind {
		case ChangeKindAppend, ChangeKindMerge, ChangeKindReplace:
			draft.ChangeKind = ChangeKindCreate
			notes = append(notes, "normalized change_kind to create because target skill did not exist")
			if !looksLikeSkillDocument(draft.BodyOrPatch) {
				draft.BodyOrPatch = synthesizeSkillDocumentFromPartialDraft(target, draft, rule)
				notes = append(notes, "synthesized full skill document because draft body was partial")
			}
		}
		return draft, notes
	}

	if draft.ChangeKind == ChangeKindCreate && !looksLikeSkillDocument(draft.BodyOrPatch) {
		draft.ChangeKind = ChangeKindAppend
		notes = append(notes, "normalized change_kind to append because target skill already existed")
	}
	return draft, notes
}

func looksLikeSkillDocument(body string) bool {
	body = strings.TrimSpace(body)
	return strings.HasPrefix(body, "---\n") && strings.Contains(body, "\n# ")
}

func synthesizeSkillDocumentFromPartialDraft(target string, draft SkillDraft, rule LearningRecord) string {
	description := strings.TrimSpace(draft.HumanSummary)
	if description == "" {
		description = fmt.Sprintf("Learned workflow for %s.", target)
	}

	bodyContent := strings.TrimSpace(draft.BodyOrPatch)
	if bodyContent == "" {
		bodyContent = "No learned content was generated."
	}
	if strings.HasPrefix(bodyContent, "# ") {
		return buildSkillDocument(target, description, bodyContent)
	}

	body := strings.Join([]string{
		"# " + titleCaseSkillName(target),
		"",
		"## Start Here",
		synthesizedStartHereLine(rule, target),
		"",
		"## Learned Evolution",
		bodyContent,
		"",
	}, "\n")
	return buildSkillDocument(target, description, body)
}

func synthesizeCombinedSkillDocument(target string, draft SkillDraft, rule LearningRecord) string {
	description := strings.TrimSpace(draft.HumanSummary)
	if description == "" {
		description = buildCombinedSkillHumanSummary(target, rule, false)
	}

	body := strings.Join([]string{
		"# " + titleCaseSkillName(target),
		"",
		"## Start Here",
		synthesizedCombinedStartHereLine(rule, target),
		"",
		"## When To Use",
		synthesizedCombinedWhenToUseLine(rule, target),
		"",
		"## Wrapped Path",
		synthesizedWrappedPathLine(rule),
		"",
		"## Learned Shortcut",
		synthesizedCombinedLearnedContent(draft.BodyOrPatch),
		"",
	}, "\n")
	return buildSkillDocument(target, description, body)
}

func synthesizeCombinedSkillAppendBody(target string, draft SkillDraft, rule LearningRecord) string {
	lines := []string{
		"## Learned Shortcut Update",
		fmt.Sprintf("- Shortcut skill: `%s`", target),
		fmt.Sprintf("- Task summary: %s", fallbackEvolutionSummary(rule)),
		fmt.Sprintf("- Wrapped path: %s", synthesizedWrappedPathLine(rule)),
		"- Guidance: prefer this shortcut directly instead of replaying the whole path when the task matches.",
		"",
		synthesizedCombinedLearnedContent(draft.BodyOrPatch),
		"",
	}
	return strings.Join(lines, "\n")
}

func synthesizedStartHereLine(rule LearningRecord, target string) string {
	if len(rule.WinningPath) > 0 {
		return fmt.Sprintf("Start with `%s` for tasks like `%s`.", strings.Join(rule.WinningPath, " -> "), strings.TrimSpace(rule.Summary))
	}
	if summary := strings.TrimSpace(rule.Summary); summary != "" {
		return fmt.Sprintf("Use `%s` when the task matches `%s`.", target, summary)
	}
	return fmt.Sprintf("Use `%s` for the learned task pattern.", target)
}

func synthesizedCombinedStartHereLine(rule LearningRecord, target string) string {
	return fmt.Sprintf("Use `%s` directly when the task matches `%s`.", target, fallbackEvolutionSummary(rule))
}

func synthesizedCombinedWhenToUseLine(rule LearningRecord, target string) string {
	if len(rule.WinningPath) == 0 {
		return fmt.Sprintf("Use `%s` when the learned task pattern appears again.", target)
	}
	return fmt.Sprintf("Use `%s` as a direct shortcut instead of replaying `%s` step by step.", target, strings.Join(rule.WinningPath, " -> "))
}

func synthesizedWrappedPathLine(rule LearningRecord) string {
	if len(rule.WinningPath) == 0 {
		return "No explicit wrapped path was recorded."
	}
	return strings.Join(rule.WinningPath, " -> ")
}

func synthesizedCombinedLearnedContent(body string) string {
	content := strings.TrimSpace(stripSkillFrontmatter(body))
	if content == "" {
		return "Use this shortcut directly when the same task pattern appears again."
	}
	return demoteMarkdownHeadings(content)
}

func stripSkillFrontmatter(body string) string {
	trimmed := strings.TrimSpace(body)
	if !strings.HasPrefix(trimmed, "---\n") {
		return trimmed
	}
	rest := strings.TrimPrefix(trimmed, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return trimmed
	}
	return strings.TrimSpace(rest[end+5:])
}

func demoteMarkdownHeadings(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		prefixLen := len(line) - len(trimmed)
		lines[i] = line[:prefixLen] + "##" + trimmed
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func fallbackEvolutionSummary(rule LearningRecord) string {
	if summary := strings.TrimSpace(rule.Summary); summary != "" {
		return summary
	}
	if len(rule.WinningPath) > 0 {
		return strings.Join(rule.WinningPath, " -> ")
	}
	return "the learned task pattern"
}

func buildCombinedSkillHumanSummary(target string, rule LearningRecord, hasExisting bool) string {
	action := "Create"
	if hasExisting {
		action = "Refresh"
	}
	return fmt.Sprintf("%s combined shortcut %s from learned pattern: %s", action, target, fallbackEvolutionSummary(rule))
}

func buildCombinedSkillAvoidPattern(target string, rule LearningRecord) string {
	if len(rule.WinningPath) == 0 {
		return fmt.Sprintf("avoid bypassing `%s` when the same learned task pattern appears again", target)
	}
	return fmt.Sprintf("avoid replaying %s before trying `%s` directly", strings.Join(rule.WinningPath, " -> "), target)
}

func collectSkillRefs(matches []skills.SkillInfo) []string {
	if len(matches) == 0 {
		return nil
	}

	refs := make([]string, 0, len(matches))
	for _, match := range matches {
		if strings := match.Path; strings != "" {
			refs = append(refs, strings)
			continue
		}
		refs = append(refs, match.Source+":"+match.Name)
	}
	return refs
}

func countTaskLearningRecords(records []LearningRecord) int {
	count := 0
	for _, record := range records {
		if isTaskRecordKind(record.Kind) {
			count++
		}
	}
	return count
}

func (rt *Runtime) runLifecycleMaintenance(workspace string, store *Store, runID string) error {
	if rt == nil || store == nil || workspace == "" {
		return nil
	}

	paths := NewPaths(workspace, rt.cfg.StateDir)
	logger.InfoCF("evolution", "Started lifecycle maintenance", map[string]any{
		"workspace": workspace,
		"run_id":    runID,
	})

	summary, err := RunLifecycleOnce(store, paths, workspace, rt.now())
	if err != nil {
		logger.WarnCF("evolution", "Lifecycle maintenance failed", map[string]any{
			"workspace": workspace,
			"run_id":    runID,
			"error":     err.Error(),
		})
		return err
	}

	logger.InfoCF("evolution", "Finished lifecycle maintenance", map[string]any{
		"workspace":             workspace,
		"run_id":                runID,
		"evaluated_profiles":    summary.EvaluatedProfiles,
		"transitioned_profiles": summary.TransitionedProfiles,
		"deleted_skills":        summary.DeletedSkills,
	})
	return nil
}

func joinRecordIDs(records []LearningRecord) string {
	if len(records) == 0 {
		return ""
	}
	ids := make([]string, 0, len(records))
	for _, record := range records {
		if strings.TrimSpace(record.ID) == "" {
			continue
		}
		ids = append(ids, record.ID)
	}
	return strings.Join(ids, ",")
}

func summarizePatternRecords(records []LearningRecord) string {
	if len(records) == 0 {
		return ""
	}
	parts := make([]string, 0, len(records))
	for _, record := range records {
		parts = append(parts, summarizePatternRecord(record))
	}
	return strings.Join(parts, " | ")
}

func summarizePatternRecord(record LearningRecord) string {
	label := strings.TrimSpace(record.ID)
	if label == "" {
		label = "unknown-pattern"
	}

	path := strings.Join(record.WinningPath, " -> ")
	if path == "" {
		path = strings.TrimSpace(record.Summary)
	}
	if path == "" {
		path = "no-summary"
	}

	return fmt.Sprintf("%s[%s]", label, path)
}

func filterNewRules(records []LearningRecord, rules []LearningRecord, workspace string) []LearningRecord {
	existing := make(map[string]struct{}, len(records))
	for _, record := range records {
		if !isPatternRecordKind(record.Kind) || record.WorkspaceID != workspace {
			continue
		}
		existing[record.ID] = struct{}{}
	}

	out := make([]LearningRecord, 0, len(rules))
	for _, rule := range rules {
		if rule.WorkspaceID != workspace {
			continue
		}
		if _, ok := existing[rule.ID]; ok {
			continue
		}
		existing[rule.ID] = struct{}{}
		out = append(out, rule)
	}
	return out
}

func filterReadyRules(records []LearningRecord, workspace string) []LearningRecord {
	seen := make(map[string]LearningRecord)
	for _, record := range records {
		if !isPatternRecordKind(record.Kind) || record.WorkspaceID != workspace || record.Status != RecordStatus("ready") {
			continue
		}
		seen[record.ID] = record
	}

	out := make([]LearningRecord, 0, len(seen))
	for _, record := range seen {
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func existingDraftSourceSet(drafts []SkillDraft, workspace string) map[string]struct{} {
	out := make(map[string]struct{}, len(drafts))
	for _, draft := range drafts {
		if draft.WorkspaceID != workspace || draft.SourceRecordID == "" {
			continue
		}
		if draft.Status == DraftStatusQuarantined {
			continue
		}
		out[draft.SourceRecordID] = struct{}{}
	}
	return out
}

func (rt *Runtime) saveAppliedProfile(store *Store, workspace string, draft SkillDraft) error {
	now := rt.now()

	return SaveAppliedProfile(store, workspace, draft, now)
}

func (rt *Runtime) recordRollbackAudit(store *Store, draft SkillDraft, applyErr error) error {
	profile, err := store.LoadProfile(draft.TargetSkillName)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	now := rt.now()
	profile.VersionHistory = append(profile.VersionHistory, SkillVersionEntry{
		Version:        profile.CurrentVersion,
		Action:         "rollback",
		Timestamp:      now,
		DraftID:        draft.ID,
		Summary:        fmt.Sprintf("Rolled back failed draft apply: %s", draft.HumanSummary),
		Rollback:       true,
		RollbackReason: applyErr.Error(),
	})
	return store.SaveProfile(profile)
}

func profileOrigin(origin string) string {
	if origin == "manual" {
		return origin
	}
	return "evolved"
}

func appendUniqueStrings(existing []string, values ...string) []string {
	seen := make(map[string]struct{}, len(existing))
	for _, value := range existing {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		existing = append(existing, value)
		seen[value] = struct{}{}
	}
	return existing
}

func (rt *Runtime) recordSkillUsage(input TurnCaseInput, success bool) error {
	usage := buildSkillUsage(input)
	if len(usage.All) == 0 {
		return nil
	}

	store := rt.storeForWorkspace(input.Workspace)
	seen := make(map[string]struct{}, len(usage.All))
	for _, skillName := range usage.All {
		skillName = strings.TrimSpace(skillName)
		if skillName == "" {
			continue
		}
		if _, ok := seen[skillName]; ok {
			continue
		}
		seen[skillName] = struct{}{}

		if err := rt.touchSkillProfile(store, input, skillName, success); err != nil {
			return err
		}
	}
	return nil
}

func (rt *Runtime) touchSkillProfile(store *Store, input TurnCaseInput, skillName string, success bool) error {
	now := rt.now()

	profile, err := store.LoadProfile(skillName)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if errors.Is(err, os.ErrNotExist) {
		profile = SkillProfile{
			SkillName:      skillName,
			WorkspaceID:    input.Workspace,
			Status:         SkillStatusActive,
			Origin:         "manual",
			HumanSummary:   skillName,
			RetentionScore: 0.2,
		}
	}

	profile.SkillName = skillName
	profile.WorkspaceID = input.Workspace
	if profile.Status == SkillStatusCold || profile.Status == SkillStatusArchived || profile.Status == "" {
		profile.Status = SkillStatusActive
	}
	if profile.Origin == "" {
		profile.Origin = "manual"
	}
	if strings.TrimSpace(profile.HumanSummary) == "" {
		profile.HumanSummary = skillName
	}
	profile.LastUsedAt = now
	profile.UseCount++
	profile.RetentionScore = nextRetentionScore(profile.RetentionScore, success)
	return store.SaveProfile(profile)
}

func nextRetentionScore(current float64, success bool) float64 {
	increment := 0.05
	if success {
		increment = 0.1
	}
	current += increment
	if current > 1 {
		return 1
	}
	return current
}
