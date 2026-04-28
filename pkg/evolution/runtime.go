package evolution

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/skills"
)

var ErrApplyDraftFailed = errors.New("apply draft failed")

type RuntimeOptions struct {
	Config           config.EvolutionConfig
	Now              func() time.Time
	Store            *Store
	Organizer        *Organizer
	SkillsRecaller   *SkillsRecaller
	DraftGenerator   DraftGenerator
	GeneratorFactory func(workspace string) DraftGenerator
	Applier          *Applier
	ApplierFactory   func(workspace string) *Applier
}

type Runtime struct {
	cfg              config.EvolutionConfig
	mu               sync.Mutex
	now              func() time.Time
	writer           *CaseWriter
	store            *Store
	organizer        *Organizer
	skillsRecaller   *SkillsRecaller
	draftGenerator   DraftGenerator
	generatorFactory func(workspace string) DraftGenerator
	applier          *Applier
	applierFactory   func(workspace string) *Applier
}

type TurnCaseInput struct {
	Workspace             string
	WorkspaceID           string
	TurnID                string
	SessionKey            string
	AgentID               string
	Status                string
	ToolKinds             []string
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
		cfg:              opts.Config,
		now:              now,
		store:            opts.Store,
		organizer:        organizer,
		skillsRecaller:   opts.SkillsRecaller,
		draftGenerator:   opts.DraftGenerator,
		generatorFactory: opts.GeneratorFactory,
		applier:          opts.Applier,
		applierFactory:   opts.ApplierFactory,
	}, nil
}

func (rt *Runtime) FinalizeTurn(ctx context.Context, input TurnCaseInput) error {
	if rt == nil || !rt.cfg.Enabled || input.Workspace == "" {
		return nil
	}

	success := input.Status == "completed"
	workspaceID := input.WorkspaceID
	if workspaceID == "" {
		workspaceID = input.Workspace
	}

	record := LearningRecord{
		ID:               input.TurnID,
		Kind:             RecordKindTask,
		WorkspaceID:      workspaceID,
		CreatedAt:        rt.now(),
		SessionKey:       input.SessionKey,
		TaskHash:         buildTaskHash(input),
		Summary:          fmt.Sprintf("turn %s finished with status=%s", input.TurnID, input.Status),
		Source:           map[string]any{"turn_id": input.TurnID, "session_key": input.SessionKey, "agent_id": input.AgentID},
		Status:           RecordStatus("new"),
		Success:          &success,
		ToolKinds:        append([]string(nil), input.ToolKinds...),
		ActiveSkillNames: append([]string(nil), input.ActiveSkillNames...),
		AttemptTrail:     buildAttemptTrail(input, success),
		Signals:          buildLearningSignals(input, success),
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

	return rt.recordSkillUsage(input, success)
}

func buildAttemptTrail(input TurnCaseInput, success bool) *AttemptTrail {
	attemptedInput := input.AttemptedSkillNames
	if len(attemptedInput) == 0 {
		attemptedInput = input.ActiveSkillNames
	}

	attempted := make([]string, 0, len(attemptedInput))
	for _, skillName := range attemptedInput {
		skillName = strings.TrimSpace(skillName)
		if skillName == "" {
			continue
		}
		attempted = append(attempted, skillName)
	}
	if len(attempted) == 0 {
		return nil
	}

	trail := &AttemptTrail{
		AttemptedSkills: attempted,
	}
	if len(input.SkillContextSnapshots) > 0 {
		trail.SkillContextSnapshots = cloneSkillContextSnapshots(input.SkillContextSnapshots)
	}
	if success {
		finalPathInput := input.FinalSuccessfulPath
		if len(finalPathInput) == 0 {
			finalPathInput = attempted
		}
		finalPath := make([]string, 0, len(finalPathInput))
		for _, skillName := range finalPathInput {
			skillName = strings.TrimSpace(skillName)
			if skillName == "" {
				continue
			}
			finalPath = append(finalPath, skillName)
		}
		trail.FinalSuccessfulPath = finalPath
	}
	return trail
}

func cloneSkillContextSnapshots(input []SkillContextSnapshot) []SkillContextSnapshot {
	if len(input) == 0 {
		return nil
	}

	out := make([]SkillContextSnapshot, 0, len(input))
	for _, snapshot := range input {
		out = append(out, SkillContextSnapshot{
			Sequence:   snapshot.Sequence,
			Trigger:    snapshot.Trigger,
			SkillNames: append([]string(nil), snapshot.SkillNames...),
		})
	}
	return out
}

func buildTaskHash(input TurnCaseInput) string {
	skillValues := input.AttemptedSkillNames
	if len(skillValues) == 0 {
		skillValues = input.ActiveSkillNames
	}

	parts := make([]string, 0, len(skillValues)+len(input.ToolKinds)+2)
	parts = append(parts, normalizedValues(skillValues)...)
	parts = append(parts, normalizedValues(input.ToolKinds)...)
	if strings.TrimSpace(input.Status) != "" {
		parts = append(parts, strings.ToLower(strings.TrimSpace(input.Status)))
	}
	if len(parts) == 0 {
		parts = append(parts, "empty")
	}

	sum := sha1.Sum([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:8])
}

func buildLearningSignals(input TurnCaseInput, success bool) []string {
	if !success {
		return nil
	}
	skillValues := input.AttemptedSkillNames
	if len(skillValues) == 0 {
		skillValues = input.ActiveSkillNames
	}
	if len(normalizedValues(skillValues)) > 1 {
		return []string{"potentially_learnable"}
	}
	return nil
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

func (rt *Runtime) RunColdPathOnce(ctx context.Context, workspace string) error {
	if rt == nil || !rt.cfg.Enabled || workspace == "" {
		return nil
	}

	mode := rt.cfg.EffectiveMode()
	if mode == "" || mode == "observe" {
		return nil
	}

	store := rt.storeForWorkspace(workspace)
	records, err := store.LoadLearningRecords()
	if err != nil {
		return err
	}

	if rt.organizer != nil {
		rules, err := rt.organizer.BuildRules(records)
		if err != nil {
			return err
		}
		newRules := filterNewRules(records, rules, workspace)
		if len(newRules) > 0 {
			if err := store.AppendLearningRecords(newRules); err != nil {
				return err
			}
			records = append(records, newRules...)
		}
	}

	generator := rt.draftGeneratorForWorkspace(workspace)
	if generator == nil {
		return nil
	}

	recaller := rt.skillsRecallerForWorkspace(workspace)
	applier := rt.applierForWorkspace(workspace)
	readyRules := filterReadyRules(records, workspace)
	if len(readyRules) == 0 {
		return nil
	}

	existingDrafts, err := store.LoadDrafts()
	if err != nil {
		return err
	}
	existingBySource := existingDraftSourceSet(existingDrafts, workspace)

	for _, rule := range readyRules {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if _, exists := existingBySource[rule.ID]; exists {
			continue
		}

		matches, err := recaller.RecallSimilarSkills(rule)
		if err != nil {
			return err
		}

		draft, err := generator.GenerateDraft(ctx, rule, matches)
		if err != nil {
			return err
		}

		draft = rt.finalizeDraft(workspace, rule, matches, draft)
		if mode == "apply" && rt.cfg.AutoApply && applier != nil && draft.Status == DraftStatusCandidate {
			rollbackApply, err := applier.applyDraftWithRollback(ctx, workspace, draft)
			if err != nil {
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
		}

		if err := store.SaveDrafts([]SkillDraft{draft}); err != nil {
			return err
		}
		existingBySource[rule.ID] = struct{}{}
	}

	return nil
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

	review := ReviewDraft(draft)
	draft.Status = review.Status
	draft.ReviewNotes = append([]string(nil), review.ReviewNotes...)
	if len(review.Findings) == 0 {
		draft.ScanFindings = nil
		return draft
	}
	draft.ScanFindings = append([]string(nil), review.Findings...)
	return draft
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
	if len(input.ActiveSkillNames) == 0 {
		return nil
	}

	store := rt.storeForWorkspace(input.Workspace)
	seen := make(map[string]struct{}, len(input.ActiveSkillNames))
	for _, skillName := range input.ActiveSkillNames {
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
