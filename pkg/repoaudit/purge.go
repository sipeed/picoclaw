package repoaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/fileutil"
)

const repositoryReviewPurgeIntentSchemaVersion = 1

var (
	ErrRepositoryReviewPurgeBlocked    = errors.New("repository review purge is blocked")
	ErrRepositoryReviewHistoryAbsent   = errors.New("repository review history is absent")
	ErrRepositoryReviewPurgeInProgress = errors.New("repository review purge is in progress")
	repositoryReviewPurgeRemoveLedger  = func(
		store Store,
		targets []repositoryReviewPurgeLedgerTarget,
	) error {
		return store.removeRepositoryReviewLedgers(targets)
	}
	repositoryReviewPurgeLstat           = os.Lstat
	repositoryReviewPurgeReadFile        = os.ReadFile
	repositoryReviewPurgeReadDir         = os.ReadDir
	repositoryReviewPurgeWriteFileAtomic = fileutil.WriteFileAtomic
	repositoryReviewPurgeIntentLimit     = maxAutomationCount
)

type RepositoryReviewPurgeBlockerCode string

const (
	RepositoryReviewPurgeBlockerReviewActive                  RepositoryReviewPurgeBlockerCode = "review_active"
	RepositoryReviewPurgeBlockerFindingProcessingActive       RepositoryReviewPurgeBlockerCode = "finding_processing_active"
	RepositoryReviewPurgeBlockerResolutionCheckActive         RepositoryReviewPurgeBlockerCode = "resolution_check_active"
	RepositoryReviewPurgeBlockerIssueGenerationActive         RepositoryReviewPurgeBlockerCode = "issue_generation_active"
	RepositoryReviewPurgeBlockerPublicationActive             RepositoryReviewPurgeBlockerCode = "publication_active"
	RepositoryReviewPurgeBlockerHistoricalConsolidationActive RepositoryReviewPurgeBlockerCode = "historical_consolidation_active"
	RepositoryReviewPurgeBlockerRetentionUnavailable          RepositoryReviewPurgeBlockerCode = "retention_unavailable"
)

// RepositoryReviewPurgeBlocker is deliberately shape-only. It never exposes
// finding, provider, repository-path, prompt, or external-issue content.
type RepositoryReviewPurgeBlocker struct {
	Code    RepositoryReviewPurgeBlockerCode `json:"code"`
	Count   int                              `json:"count"`
	Message string                           `json:"message"`
}

// RepositoryReviewPurgeSummary is the bounded confirmation projection for a
// whole-ledger purge. ExternalIssueAssociations counts local canonical
// associations; purging never calls or mutates the external provider.
type RepositoryReviewPurgeSummary struct {
	RepositoryVersion         int64  `json:"repository_version"`
	LedgerFence               string `json:"ledger_fence"`
	RawFindings               int    `json:"raw_findings"`
	DeduplicatedFindings      int    `json:"deduplicated_findings"`
	RepositoryFindings        int    `json:"repository_findings"`
	IssuePreviews             int    `json:"issue_previews"`
	ExternalIssueAssociations int    `json:"external_issue_associations"`
}

type RepositoryReviewPurgeEligibility struct {
	HistoryFound bool                           `json:"history_found"`
	CanPurge     bool                           `json:"can_purge_history"`
	CanRemove    bool                           `json:"can_remove_repository"`
	Blockers     []RepositoryReviewPurgeBlocker `json:"purge_blockers"`
	Summary      RepositoryReviewPurgeSummary   `json:"purge_summary"`
}

type repositoryReviewPurgeMode string
type repositoryReviewPurgePhase string

const (
	repositoryReviewPurgeReset  repositoryReviewPurgeMode = "reset"
	repositoryReviewPurgeRemove repositoryReviewPurgeMode = "remove"

	repositoryReviewPurgePrepared             repositoryReviewPurgePhase = "prepared"
	repositoryReviewPurgeAutomationCommitting repositoryReviewPurgePhase = "automation_committing"
	repositoryReviewPurgeAutomationApplied    repositoryReviewPurgePhase = "automation_applied"
	repositoryReviewPurgeLedgerCommitting     repositoryReviewPurgePhase = "ledger_committing"
	repositoryReviewPurgeLedgerRemoved        repositoryReviewPurgePhase = "ledger_removed"
)

type repositoryReviewPurgeIntent struct {
	SchemaVersion             int                                 `json:"schema_version"`
	Mode                      repositoryReviewPurgeMode           `json:"mode"`
	Phase                     repositoryReviewPurgePhase          `json:"phase"`
	AutomationID              string                              `json:"automation_id"`
	ConfiguredRepository      string                              `json:"configured_repository"`
	Repository                string                              `json:"repository"`
	LedgerTargets             []repositoryReviewPurgeLedgerTarget `json:"ledger_targets"`
	ExpectedAutomationVersion int64                               `json:"expected_automation_version"`
	ExpectedRepositoryVersion int64                               `json:"expected_repository_version"`
	CreatedAt                 time.Time                           `json:"created_at"`
}

type repositoryReviewPurgeLedgerTarget struct {
	Repository string `json:"repository"`
	Version    int64  `json:"version"`
}

func EvaluateRepositoryReviewPurge(
	automation RepositoryReviewAutomation,
	state RepositoryState,
	historyFound bool,
) RepositoryReviewPurgeEligibility {
	states := []RepositoryState{}
	if historyFound {
		states = append(states, state)
	}
	return evaluateRepositoryReviewPurgeInventory(automation, state.Version, states)
}

func evaluateRepositoryReviewPurgeInventory(
	automation RepositoryReviewAutomation,
	primaryVersion int64,
	states []RepositoryState,
) RepositoryReviewPurgeEligibility {
	eligibility := RepositoryReviewPurgeEligibility{
		HistoryFound: len(states) > 0,
		Blockers:     []RepositoryReviewPurgeBlocker{},
		Summary: RepositoryReviewPurgeSummary{
			RepositoryVersion: primaryVersion,
		},
	}
	targets := make([]repositoryReviewPurgeLedgerTarget, 0, len(states))
	for _, state := range states {
		targets = append(targets, repositoryReviewPurgeLedgerTarget{
			Repository: state.Repository, Version: state.Version,
		})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Repository < targets[j].Repository })
	eligibility.Summary.LedgerFence = repositoryReviewPurgeLedgerFence(targets)
	counts := make(map[RepositoryReviewPurgeBlockerCode]int)
	activeRuns := make(map[string]struct{})
	externalIssues := make(map[string]struct{})
	if automation.Status == RepositoryReviewAutomationRunning ||
		automation.Status == RepositoryReviewAutomationStopping ||
		strings.TrimSpace(automation.ActiveRunID) != "" {
		activeID := strings.TrimSpace(automation.ActiveRunID)
		if activeID == "" {
			activeID = "automation"
		}
		activeRuns[activeID] = struct{}{}
	}
	for _, state := range states {
		eligibility.Summary.RawFindings += len(state.RawFindings)
		eligibility.Summary.DeduplicatedFindings += len(state.DeduplicatedFindings)
		eligibility.Summary.RepositoryFindings += len(state.RepositoryFindings)
		eligibility.Summary.IssuePreviews += len(state.IssueDrafts)
		for _, finding := range state.RepositoryFindings {
			if issueURL := strings.TrimSpace(finding.Issue.URL); issueURL != "" {
				externalIssues[issueURL] = struct{}{}
			}
			for _, conflictURL := range finding.Issue.ConflictURLs {
				if conflictURL = strings.TrimSpace(conflictURL); conflictURL != "" {
					externalIssues[conflictURL] = struct{}{}
				}
			}
		}
		if state.ActiveReviewRun != nil {
			activeID := strings.TrimSpace(state.ActiveReviewRun.ID)
			if activeID == "" {
				activeID = "ledger:" + state.Repository
			}
			activeRuns[activeID] = struct{}{}
		}
		for _, job := range state.DeduplicationJobs {
			if job.State == DeduplicationJobRunning {
				counts[RepositoryReviewPurgeBlockerFindingProcessingActive]++
			}
		}
		for _, job := range state.MappingJobs {
			if job.State == RepositoryMappingRunning {
				counts[RepositoryReviewPurgeBlockerFindingProcessingActive]++
			}
		}
		for _, job := range state.ValidationJobs {
			if job.State == RepositoryValidationRunning {
				counts[RepositoryReviewPurgeBlockerResolutionCheckActive]++
			}
		}
		for _, draft := range state.IssueDrafts {
			switch draft.State {
			case IssueDraftGenerating:
				counts[RepositoryReviewPurgeBlockerIssueGenerationActive]++
			case IssueDraftPublishing:
				counts[RepositoryReviewPurgeBlockerPublicationActive]++
			}
		}
		replay := state.HistoricalDeduplication
		if replay.Required && (replay.Status == HistoricalDeduplicationPending ||
			replay.Status == HistoricalDeduplicationReplaying ||
			replay.Status == HistoricalDeduplicationMerging) || replay.MergeLease.ID != "" {
			counts[RepositoryReviewPurgeBlockerHistoricalConsolidationActive]++
		}
	}
	counts[RepositoryReviewPurgeBlockerReviewActive] = len(activeRuns)
	eligibility.Summary.ExternalIssueAssociations = len(externalIssues)
	ordered := []RepositoryReviewPurgeBlockerCode{
		RepositoryReviewPurgeBlockerReviewActive,
		RepositoryReviewPurgeBlockerFindingProcessingActive,
		RepositoryReviewPurgeBlockerResolutionCheckActive,
		RepositoryReviewPurgeBlockerIssueGenerationActive,
		RepositoryReviewPurgeBlockerPublicationActive,
		RepositoryReviewPurgeBlockerHistoricalConsolidationActive,
	}
	for _, code := range ordered {
		if count := counts[code]; count > 0 {
			eligibility.Blockers = append(eligibility.Blockers, RepositoryReviewPurgeBlocker{
				Code: code, Count: count, Message: repositoryReviewPurgeBlockerMessage(code),
			})
		}
	}
	eligibility.CanRemove = len(eligibility.Blockers) == 0
	eligibility.CanPurge = eligibility.HistoryFound && eligibility.CanRemove
	return eligibility
}

func repositoryReviewPurgeBlockerMessage(code RepositoryReviewPurgeBlockerCode) string {
	switch code {
	case RepositoryReviewPurgeBlockerReviewActive:
		return "Stop the active repository review before deleting its history."
	case RepositoryReviewPurgeBlockerFindingProcessingActive:
		return "Wait for active finding processing to finish before deleting history."
	case RepositoryReviewPurgeBlockerResolutionCheckActive:
		return "Wait for active resolution checks to finish before deleting history."
	case RepositoryReviewPurgeBlockerIssueGenerationActive:
		return "Wait for active issue-preview generation to finish before deleting history."
	case RepositoryReviewPurgeBlockerPublicationActive:
		return "Wait for active GitHub publication to settle before deleting history."
	case RepositoryReviewPurgeBlockerHistoricalConsolidationActive:
		return "Wait for historical finding consolidation to finish before deleting history."
	case RepositoryReviewPurgeBlockerRetentionUnavailable:
		return "History deletion status is unavailable."
	default:
		return "Repository review history cannot be deleted while related work is active."
	}
}

// PurgeAutomationHistory permanently removes one repository ledger while
// retaining its reusable configuration as a fresh idle automation.
func (s Store) PurgeAutomationHistory(
	ctx context.Context,
	id string,
	expectedAutomationVersion int64,
	expectedRepositoryVersion int64,
	expectedLedgerFence string,
	confirmRepository string,
) (RepositoryReviewAutomation, RepositoryReviewPurgeEligibility, error) {
	if strings.TrimSpace(expectedLedgerFence) == "" {
		return RepositoryReviewAutomation{}, RepositoryReviewPurgeEligibility{}, ErrInvalidAutomation
	}
	return s.purgeAutomation(
		ctx, id, expectedAutomationVersion, expectedRepositoryVersion,
		expectedLedgerFence, confirmRepository, repositoryReviewPurgeReset,
	)
}

// DeleteAutomationAndHistory permanently removes both the repository
// assignment and its repository-review ledger. It deliberately leaves generic
// workflow-run storage and every external issue unchanged.
func (s Store) DeleteAutomationAndHistory(
	ctx context.Context,
	id string,
	expectedAutomationVersion int64,
	expectedRepositoryVersion int64,
	expectedLedgerFence string,
	confirmRepository string,
) (RepositoryReviewPurgeEligibility, error) {
	if strings.TrimSpace(expectedLedgerFence) == "" {
		return RepositoryReviewPurgeEligibility{}, ErrInvalidAutomation
	}
	_, eligibility, err := s.purgeAutomation(
		ctx, id, expectedAutomationVersion, expectedRepositoryVersion,
		expectedLedgerFence, confirmRepository, repositoryReviewPurgeRemove,
	)
	return eligibility, err
}

func (s Store) RepositoryReviewPurgeEligibilityForAutomation(
	automation RepositoryReviewAutomation,
) (RepositoryReviewPurgeEligibility, error) {
	unlock, err := s.lock("repository-review-purge-eligibility:" + automation.ID)
	if err != nil {
		return RepositoryReviewPurgeEligibility{}, err
	}
	defer unlock()
	current, found, err := s.loadAutomationIgnoringPurge(automation.ID)
	if err != nil {
		return RepositoryReviewPurgeEligibility{}, err
	}
	if !found || current.Version != automation.Version || current.Repository != automation.Repository {
		return RepositoryReviewPurgeEligibility{}, ErrConflict
	}
	if purging, err := s.repositoryReviewPurgeConfigured(current.Repository); err != nil {
		return RepositoryReviewPurgeEligibility{}, err
	} else if purging {
		return RepositoryReviewPurgeEligibility{}, ErrRepositoryReviewPurgeInProgress
	}
	state, _, _, states, err := s.resolveRepositoryPurgeInventory(current)
	if err != nil {
		return RepositoryReviewPurgeEligibility{}, err
	}
	return evaluateRepositoryReviewPurgeInventory(current, state.Version, states), nil
}

func (s Store) purgeAutomation(
	ctx context.Context,
	id string,
	expectedAutomationVersion int64,
	expectedRepositoryVersion int64,
	expectedLedgerFence string,
	confirmRepository string,
	mode repositoryReviewPurgeMode,
) (RepositoryReviewAutomation, RepositoryReviewPurgeEligibility, error) {
	if err := ctx.Err(); err != nil {
		return RepositoryReviewAutomation{}, RepositoryReviewPurgeEligibility{}, err
	}
	id = strings.TrimSpace(id)
	if !validAutomationID(id) || expectedAutomationVersion < 1 || expectedRepositoryVersion < 0 ||
		(mode != repositoryReviewPurgeReset && mode != repositoryReviewPurgeRemove) {
		return RepositoryReviewAutomation{}, RepositoryReviewPurgeEligibility{}, ErrInvalidAutomation
	}
	unlock, err := s.lock("repository-review-purge:" + id)
	if err != nil {
		return RepositoryReviewAutomation{}, RepositoryReviewPurgeEligibility{}, err
	}
	defer unlock()
	if err := ctx.Err(); err != nil {
		return RepositoryReviewAutomation{}, RepositoryReviewPurgeEligibility{}, err
	}
	if existing, existingFound, loadErr := s.loadPurgeIntentForAutomation(id); loadErr != nil {
		return RepositoryReviewAutomation{}, RepositoryReviewPurgeEligibility{}, loadErr
	} else if existingFound {
		if _, applyErr := s.applyPurgeIntent(existing); applyErr != nil {
			return RepositoryReviewAutomation{}, RepositoryReviewPurgeEligibility{}, applyErr
		}
		return RepositoryReviewAutomation{}, RepositoryReviewPurgeEligibility{}, ErrConflict
	}
	automation, found, err := s.loadAutomationIgnoringPurge(id)
	if err != nil {
		return RepositoryReviewAutomation{}, RepositoryReviewPurgeEligibility{}, err
	}
	if !found {
		return RepositoryReviewAutomation{}, RepositoryReviewPurgeEligibility{}, os.ErrNotExist
	}
	if automation.Version != expectedAutomationVersion ||
		confirmRepository != automation.Repository {
		return RepositoryReviewAutomation{}, RepositoryReviewPurgeEligibility{}, ErrConflict
	}
	state, historyFound, ledgerTargets, ledgerStates, err := s.resolveRepositoryPurgeInventory(automation)
	if err != nil {
		return RepositoryReviewAutomation{}, RepositoryReviewPurgeEligibility{}, err
	}
	ledgerRepository := state.Repository
	if !historyFound {
		ledgerRepository = CanonicalRepositoryIdentity(automation.Repository)
		state = RepositoryState{Repository: ledgerRepository}
	}
	if historyFound && state.Version != expectedRepositoryVersion ||
		!historyFound && expectedRepositoryVersion != 0 {
		return RepositoryReviewAutomation{}, RepositoryReviewPurgeEligibility{}, ErrConflict
	}
	eligibility := evaluateRepositoryReviewPurgeInventory(automation, state.Version, ledgerStates)
	if expectedLedgerFence != "" && expectedLedgerFence != eligibility.Summary.LedgerFence {
		return RepositoryReviewAutomation{}, eligibility, ErrConflict
	}
	if mode == repositoryReviewPurgeReset && !eligibility.CanPurge {
		if !historyFound && eligibility.CanRemove {
			return RepositoryReviewAutomation{}, eligibility, ErrRepositoryReviewHistoryAbsent
		}
		return RepositoryReviewAutomation{}, eligibility, ErrRepositoryReviewPurgeBlocked
	}
	if mode == repositoryReviewPurgeRemove && !eligibility.CanRemove {
		return RepositoryReviewAutomation{}, eligibility, ErrRepositoryReviewPurgeBlocked
	}
	intent := repositoryReviewPurgeIntent{
		SchemaVersion: repositoryReviewPurgeIntentSchemaVersion,
		Mode:          mode, Phase: repositoryReviewPurgePrepared,
		AutomationID: automation.ID, ConfiguredRepository: automation.Repository,
		Repository:                ledgerRepository,
		LedgerTargets:             ledgerTargets,
		ExpectedAutomationVersion: automation.Version,
		ExpectedRepositoryVersion: state.Version,
		CreatedAt:                 s.clock(),
	}
	if err := s.savePurgeIntent(intent); err != nil {
		return RepositoryReviewAutomation{}, eligibility, err
	}
	updated, err := s.applyPurgeIntent(intent)
	return updated, eligibility, err
}

// ReconcilePurgeIntents completes crash-interrupted purge transactions. The
// controller calls it after acquiring its workspace lease and before admitting
// any review or finding worker.
func (s Store) ReconcilePurgeIntents(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	unlock, err := s.lock("repository-review-purge-recovery")
	if err != nil {
		return 0, err
	}
	defer unlock()
	if err := s.requireSafeRoot(true); err != nil {
		return 0, err
	}
	entries, err := repositoryReviewPurgeReadDir(s.root)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	names := make([]string, 0)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "purge_automation_") && strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) > repositoryReviewPurgeIntentLimit {
		return 0, errors.New("repository review purge catalog exceeds its limit")
	}
	completed := 0
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return completed, err
		}
		intent, found, loadErr := s.loadPurgeIntentPath(filepath.Join(s.root, name))
		if loadErr != nil {
			return completed, loadErr
		}
		if !found {
			continue
		}
		if _, applyErr := s.applyPurgeIntent(intent); applyErr != nil {
			return completed, applyErr
		}
		completed++
	}
	return completed, nil
}

func (s Store) applyPurgeIntent(intent repositoryReviewPurgeIntent) (RepositoryReviewAutomation, error) {
	for {
		if err := validateRepositoryReviewPurgeIntent(intent); err != nil {
			return RepositoryReviewAutomation{}, err
		}
		switch intent.Phase {
		case repositoryReviewPurgePrepared:
			automation, state, _, err := s.validatePreparedPurgeIntent(intent)
			if err != nil {
				return RepositoryReviewAutomation{}, err
			}
			_, _, _, ledgerStates, err := s.resolveRepositoryPurgeInventory(automation)
			if err != nil {
				return RepositoryReviewAutomation{}, err
			}
			eligibility := evaluateRepositoryReviewPurgeInventory(automation, state.Version, ledgerStates)
			if intent.Mode == repositoryReviewPurgeReset && !eligibility.CanPurge ||
				intent.Mode == repositoryReviewPurgeRemove && !eligibility.CanRemove {
				return RepositoryReviewAutomation{}, ErrRepositoryReviewPurgeBlocked
			}
			intent.Phase = repositoryReviewPurgeAutomationCommitting
			if err := s.savePurgeIntent(intent); err != nil {
				return RepositoryReviewAutomation{}, err
			}
		case repositoryReviewPurgeAutomationCommitting:
			if err := s.applyPurgeAutomationPhase(intent); err != nil {
				return RepositoryReviewAutomation{}, err
			}
			intent.Phase = repositoryReviewPurgeAutomationApplied
			if err := s.savePurgeIntent(intent); err != nil {
				return RepositoryReviewAutomation{}, err
			}
		case repositoryReviewPurgeAutomationApplied:
			if err := s.verifyPurgeAutomationApplied(intent); err != nil {
				return RepositoryReviewAutomation{}, err
			}
			if err := s.verifyPurgeLedgerTargets(intent.LedgerTargets, true); err != nil {
				return RepositoryReviewAutomation{}, err
			}
			intent.Phase = repositoryReviewPurgeLedgerCommitting
			if err := s.savePurgeIntent(intent); err != nil {
				return RepositoryReviewAutomation{}, err
			}
		case repositoryReviewPurgeLedgerCommitting:
			if err := s.verifyPurgeAutomationApplied(intent); err != nil {
				return RepositoryReviewAutomation{}, err
			}
			if err := s.verifyPurgeLedgerTargets(intent.LedgerTargets, false); err != nil {
				return RepositoryReviewAutomation{}, err
			}
			if err := repositoryReviewPurgeRemoveLedger(s, intent.LedgerTargets); err != nil {
				return RepositoryReviewAutomation{}, err
			}
			intent.Phase = repositoryReviewPurgeLedgerRemoved
			if err := s.savePurgeIntent(intent); err != nil {
				return RepositoryReviewAutomation{}, err
			}
		case repositoryReviewPurgeLedgerRemoved:
			if err := s.verifyPurgeLedgerTargetsRemoved(intent.LedgerTargets); err != nil {
				return RepositoryReviewAutomation{}, err
			}
			if err := s.removePurgeIntent(intent); err != nil {
				return RepositoryReviewAutomation{}, err
			}
			if intent.Mode == repositoryReviewPurgeReset {
				updated, found, err := s.loadAutomationIgnoringPurge(intent.AutomationID)
				if err != nil || !found {
					if err == nil {
						err = errors.New("repository review purge lost its retained configuration")
					}
					return RepositoryReviewAutomation{}, err
				}
				return cloneAutomation(updated), nil
			}
			return RepositoryReviewAutomation{}, nil
		}
	}
}

func (s Store) validatePreparedPurgeIntent(
	intent repositoryReviewPurgeIntent,
) (RepositoryReviewAutomation, RepositoryState, bool, error) {
	automation, found, err := s.loadAutomationIgnoringPurge(intent.AutomationID)
	if err != nil {
		return RepositoryReviewAutomation{}, RepositoryState{}, false, err
	}
	if !found || automation.Version != intent.ExpectedAutomationVersion ||
		automation.Repository != intent.ConfiguredRepository {
		return RepositoryReviewAutomation{}, RepositoryState{}, false, ErrConflict
	}
	state, historyFound, targets, _, err := s.resolveRepositoryPurgeInventory(automation)
	if err != nil {
		return RepositoryReviewAutomation{}, RepositoryState{}, false, err
	}
	if historyFound {
		if state.Repository != intent.Repository || state.Version != intent.ExpectedRepositoryVersion ||
			!repositoryReviewPurgeLedgerTargetsEqual(targets, intent.LedgerTargets) {
			return RepositoryReviewAutomation{}, RepositoryState{}, false, ErrConflict
		}
	} else if intent.ExpectedRepositoryVersion != 0 || len(intent.LedgerTargets) != 0 ||
		intent.Repository != CanonicalRepositoryIdentity(automation.Repository) {
		return RepositoryReviewAutomation{}, RepositoryState{}, false, ErrConflict
	}
	return automation, state, historyFound, nil
}

func (s Store) applyPurgeAutomationPhase(intent repositoryReviewPurgeIntent) error {
	automation, found, err := s.loadAutomationIgnoringPurge(intent.AutomationID)
	if err != nil {
		return err
	}
	if intent.Mode == repositoryReviewPurgeRemove {
		if !found {
			return nil
		}
		if automation.Version != intent.ExpectedAutomationVersion ||
			automation.Repository != intent.ConfiguredRepository {
			return ErrConflict
		}
		return removeRepositoryReviewRegularFile(s.automationPath(intent.AutomationID))
	}
	if !found || automation.Repository != intent.ConfiguredRepository {
		return ErrConflict
	}
	switch automation.Version {
	case intent.ExpectedAutomationVersion:
		updated := cloneAutomation(automation)
		resetRepositoryReviewAutomationHistory(&updated)
		updated.Version++
		updated.UpdatedAt = s.clock()
		return s.saveAutomation(updated)
	case intent.ExpectedAutomationVersion + 1:
		if repositoryReviewAutomationHistoryReset(automation) {
			return nil
		}
	}
	return ErrConflict
}

func (s Store) verifyPurgeAutomationApplied(intent repositoryReviewPurgeIntent) error {
	automation, found, err := s.loadAutomationIgnoringPurge(intent.AutomationID)
	if err != nil {
		return err
	}
	if intent.Mode == repositoryReviewPurgeRemove {
		if found {
			return ErrConflict
		}
		return nil
	}
	if !found || automation.Repository != intent.ConfiguredRepository ||
		automation.Version != intent.ExpectedAutomationVersion+1 ||
		!repositoryReviewAutomationHistoryReset(automation) {
		return ErrConflict
	}
	return nil
}

func (s Store) resolveRepositoryStateIgnoringPurge(
	automation RepositoryReviewAutomation,
) (RepositoryState, bool, error) {
	state, found, _, _, err := s.resolveRepositoryPurgeInventory(automation)
	return state, found, err
}

func (s Store) resolveRepositoryPurgeInventory(
	automation RepositoryReviewAutomation,
) (RepositoryState, bool, []repositoryReviewPurgeLedgerTarget, []RepositoryState, error) {
	statesByRepository := make(map[string]RepositoryState)
	var primary RepositoryState
	primaryFound := false
	for _, identity := range RepositoryLedgerIdentities(automation.Repository) {
		state, err := s.loadIgnoringPurge(identity)
		if err != nil {
			return RepositoryState{}, false, nil, nil, err
		}
		if state.Version > 0 {
			statesByRepository[state.Repository] = state
			if !primaryFound {
				primary, primaryFound = state, true
			}
		}
	}
	wanted := make(map[string]struct{}, len(automation.RunIDs))
	for _, runID := range automation.RunIDs {
		if runID = strings.TrimSpace(runID); runID != "" {
			wanted[runID] = struct{}{}
		}
	}
	if !primaryFound && len(wanted) > 0 {
		states, err := s.listStates(false)
		if err != nil {
			return RepositoryState{}, false, nil, nil, err
		}
		runMatches := make([]RepositoryState, 0)
		for _, state := range states {
			matched := false
			for _, run := range state.Runs {
				if _, ok := wanted[run.ID]; ok {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
			statesByRepository[state.Repository] = state
			runMatches = append(runMatches, state)
		}
		if !primaryFound {
			if len(runMatches) > 1 {
				return RepositoryState{}, false, nil, nil, errors.New("ambiguous repository review ledger")
			}
			if len(runMatches) == 1 {
				primary, primaryFound = runMatches[0], true
			}
		}
	}
	targets := make([]repositoryReviewPurgeLedgerTarget, 0, len(statesByRepository))
	states := make([]RepositoryState, 0, len(statesByRepository))
	for _, state := range statesByRepository {
		targets = append(targets, repositoryReviewPurgeLedgerTarget{
			Repository: state.Repository,
			Version:    state.Version,
		})
		states = append(states, state)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Repository < targets[j].Repository })
	sort.Slice(states, func(i, j int) bool { return states[i].Repository < states[j].Repository })
	return primary, primaryFound, targets, states, nil
}

func (s Store) loadPurgeTargetLedger(repository string) (RepositoryState, bool, error) {
	state, err := s.loadIgnoringPurge(repository)
	return state, state.Version > 0, err
}

func (s Store) verifyPurgeLedgerTargets(
	targets []repositoryReviewPurgeLedgerTarget,
	requireAll bool,
) error {
	for _, target := range targets {
		state, found, err := s.loadPurgeTargetLedger(target.Repository)
		if err != nil {
			return err
		}
		if requireAll && !found || found && state.Version != target.Version {
			return ErrConflict
		}
	}
	return nil
}

func (s Store) verifyPurgeLedgerTargetsRemoved(
	targets []repositoryReviewPurgeLedgerTarget,
) error {
	for _, target := range targets {
		if _, found, err := s.loadPurgeTargetLedger(target.Repository); err != nil {
			return err
		} else if found {
			return ErrConflict
		}
	}
	return nil
}

func repositoryReviewPurgeLedgerTargetsEqual(
	left, right []repositoryReviewPurgeLedgerTarget,
) bool {
	return reflect.DeepEqual(left, right)
}

func repositoryReviewPurgeLedgerFence(
	targets []repositoryReviewPurgeLedgerTarget,
) string {
	parts := make([]string, 0, 1+len(targets)*2)
	parts = append(parts, "repository-review-ledger-fence-v1")
	for _, target := range targets {
		parts = append(parts, target.Repository, fmt.Sprint(target.Version))
	}
	return stableID("rplf_", parts...)
}

func resetRepositoryReviewAutomationHistory(automation *RepositoryReviewAutomation) {
	if automation == nil {
		return
	}
	automation.EffectiveAccountRef = ""
	automation.ResolvedCommitSHA = ""
	automation.ResolvedTargetBranch = ""
	automation.AdvertisedDefaultBranch = ""
	automation.TargetIsDefault = automation.Ref == ""
	automation.ScopePlan = RepositoryReviewScopePlan{}
	automation.ScopeSelection = nil
	automation.AccountModelRevision = ""
	automation.ModelPrices = nil
	automation.Status = RepositoryReviewAutomationIdle
	automation.PauseReason = ""
	automation.PauseDetail = ""
	automation.RequestedPauseReason = ""
	automation.RequestedPauseDetail = ""
	automation.CampaignID = ""
	automation.CampaignRecoveryPending = false
	automation.ActiveRunID = ""
	automation.RunIDs = nil
	automation.Usage = RepositoryReviewTokenUsage{}
	automation.EstimatedCostUSD = 0
	automation.Progress = RepositoryReviewProgress{}
	automation.ModelStats = nil
	automation.ModelCoverageSketches = nil
	automation.AccountLimitSnapshots = nil
	automation.StartedAt = time.Time{}
	automation.CompletedAt = time.Time{}
}

func repositoryReviewAutomationHistoryReset(automation RepositoryReviewAutomation) bool {
	return automation.Status == RepositoryReviewAutomationIdle &&
		automation.EffectiveAccountRef == "" && automation.ResolvedCommitSHA == "" &&
		automation.ResolvedTargetBranch == "" && automation.AdvertisedDefaultBranch == "" &&
		automation.TargetIsDefault == (automation.Ref == "") &&
		automation.ScopePlan.Hash == "" && automation.ScopeSelection == nil && automation.AccountModelRevision == "" &&
		len(automation.ModelPrices) == 0 && automation.PauseReason == "" && automation.PauseDetail == "" &&
		automation.RequestedPauseReason == "" && automation.RequestedPauseDetail == "" &&
		automation.ActiveRunID == "" && automation.CampaignID == "" && len(automation.RunIDs) == 0 &&
		!automation.CampaignRecoveryPending &&
		automation.Usage == (RepositoryReviewTokenUsage{}) && automation.EstimatedCostUSD == 0 &&
		automation.Progress == (RepositoryReviewProgress{}) && len(automation.ModelStats) == 0 &&
		len(automation.ModelCoverageSketches) == 0 && len(automation.AccountLimitSnapshots) == 0 &&
		automation.StartedAt.IsZero() && automation.CompletedAt.IsZero()
}

func (s Store) removeRepositoryReviewLedger(repository string) error {
	statePath := s.path(repository)
	summaryPath := strings.TrimSuffix(statePath, ".json") + ".summary.json"
	if err := removeRepositoryReviewRegularFile(summaryPath); err != nil {
		return err
	}
	return removeRepositoryReviewRegularFile(statePath)
}

func (s Store) removeRepositoryReviewLedgers(
	targets []repositoryReviewPurgeLedgerTarget,
) error {
	for _, target := range targets {
		if err := s.removeRepositoryReviewLedger(target.Repository); err != nil {
			return err
		}
	}
	return nil
}

func removeRepositoryReviewRegularFile(path string) error {
	info, err := repositoryReviewPurgeLstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("repository review purge target must be a regular file")
	}
	return fileutil.RemoveDurable(path)
}

func (s Store) savePurgeIntent(intent repositoryReviewPurgeIntent) error {
	if err := validateRepositoryReviewPurgeIntent(intent); err != nil {
		return err
	}
	if err := s.ensureSafeRoot(fileutil.MkdirAllDurable); err != nil {
		return err
	}
	// The intent contains only JSON-native scalar/time fields.
	data, _ := json.Marshal(intent)
	paths := s.purgeIntentPaths(intent)
	for _, path := range paths {
		if info, statErr := repositoryReviewPurgeLstat(path); statErr == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
				return errors.New("repository review purge intent must be a private regular file")
			}
			current, found, loadErr := s.loadPurgeIntentPath(path)
			if loadErr != nil {
				return loadErr
			}
			if !found || !repositoryReviewPurgeIntentsSameTransaction(current, intent) {
				return ErrRepositoryReviewPurgeInProgress
			}
			if repositoryReviewPurgePhaseOrder(current.Phase) > repositoryReviewPurgePhaseOrder(intent.Phase) {
				return ErrConflict
			}
		} else if !errors.Is(statErr, fs.ErrNotExist) {
			return statErr
		}
	}
	for _, path := range paths {
		if err := repositoryReviewPurgeWriteFileAtomic(path, data, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) loadPurgeIntentForAutomation(id string) (repositoryReviewPurgeIntent, bool, error) {
	if !validAutomationID(id) {
		return repositoryReviewPurgeIntent{}, false, ErrInvalidAutomation
	}
	return s.loadPurgeIntentPath(s.purgeAutomationIntentPath(id))
}

func (s Store) loadPurgeFence(repository string) (repositoryReviewPurgeIntent, bool, error) {
	repository = strings.TrimSpace(repository)
	if !validBoundedText(repository, maxRepositoryIdentityBytes) {
		return repositoryReviewPurgeIntent{}, false, ErrInvalidAutomation
	}
	return s.loadPurgeIntentPath(s.purgeRepositoryFencePath(repository))
}

func (s Store) loadPurgeIntentPath(path string) (repositoryReviewPurgeIntent, bool, error) {
	info, err := repositoryReviewPurgeLstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return repositoryReviewPurgeIntent{}, false, nil
	}
	if err != nil {
		return repositoryReviewPurgeIntent{}, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() ||
		info.Mode().Perm() != 0o600 || info.Size() > maxAutomationFileBytes {
		return repositoryReviewPurgeIntent{}, false, errors.New("invalid repository review purge intent")
	}
	data, err := repositoryReviewPurgeReadFile(path)
	if err != nil {
		return repositoryReviewPurgeIntent{}, false, err
	}
	var intent repositoryReviewPurgeIntent
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&intent); err != nil {
		return repositoryReviewPurgeIntent{}, false, err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return repositoryReviewPurgeIntent{}, false, errors.New("invalid repository review purge intent")
	}
	if err := validateRepositoryReviewPurgeIntent(intent); err != nil {
		return repositoryReviewPurgeIntent{}, false, err
	}
	validPath := path == s.purgeAutomationIntentPath(intent.AutomationID)
	if !validPath {
		for _, repository := range repositoryReviewPurgeFenceRepositories(intent) {
			if path == s.purgeRepositoryFencePath(repository) {
				validPath = true
				break
			}
		}
	}
	if !validPath {
		return repositoryReviewPurgeIntent{}, false, errors.New("repository review purge intent path mismatch")
	}
	return intent, true, nil
}

func validateRepositoryReviewPurgeIntent(intent repositoryReviewPurgeIntent) error {
	if intent.SchemaVersion != repositoryReviewPurgeIntentSchemaVersion ||
		(intent.Mode != repositoryReviewPurgeReset && intent.Mode != repositoryReviewPurgeRemove) ||
		!validRepositoryReviewPurgePhase(intent.Phase) ||
		!validAutomationID(intent.AutomationID) ||
		!validBoundedText(intent.ConfiguredRepository, maxRepositoryIdentityBytes) ||
		!validAutomationRepository(intent.ConfiguredRepository) ||
		!validBoundedText(intent.Repository, maxRepositoryIdentityBytes) ||
		intent.ExpectedAutomationVersion < 1 || intent.ExpectedRepositoryVersion < 0 ||
		intent.CreatedAt.IsZero() {
		return fmt.Errorf("%w: invalid purge intent", ErrInvalidAutomation)
	}
	if len(intent.LedgerTargets) > maxAutomationCount ||
		(intent.ExpectedRepositoryVersion == 0) != (len(intent.LedgerTargets) == 0) {
		return fmt.Errorf("%w: invalid purge ledger targets", ErrInvalidAutomation)
	}
	primaryFound := intent.ExpectedRepositoryVersion == 0
	previous := ""
	for _, target := range intent.LedgerTargets {
		if !validBoundedText(target.Repository, maxRepositoryIdentityBytes) || target.Version < 1 ||
			previous != "" && target.Repository <= previous {
			return fmt.Errorf("%w: invalid purge ledger target", ErrInvalidAutomation)
		}
		if target.Repository == intent.Repository && target.Version == intent.ExpectedRepositoryVersion {
			primaryFound = true
		}
		previous = target.Repository
	}
	if !primaryFound {
		return fmt.Errorf("%w: purge primary ledger target is missing", ErrInvalidAutomation)
	}
	return nil
}

func validRepositoryReviewPurgePhase(phase repositoryReviewPurgePhase) bool {
	return phase == repositoryReviewPurgePrepared ||
		phase == repositoryReviewPurgeAutomationCommitting ||
		phase == repositoryReviewPurgeAutomationApplied ||
		phase == repositoryReviewPurgeLedgerCommitting ||
		phase == repositoryReviewPurgeLedgerRemoved
}

func repositoryReviewPurgePhaseOrder(phase repositoryReviewPurgePhase) int {
	switch phase {
	case repositoryReviewPurgePrepared:
		return 1
	case repositoryReviewPurgeAutomationCommitting:
		return 2
	case repositoryReviewPurgeAutomationApplied:
		return 3
	case repositoryReviewPurgeLedgerCommitting:
		return 4
	case repositoryReviewPurgeLedgerRemoved:
		return 5
	default:
		return 0
	}
}

func repositoryReviewPurgeIntentsSameTransaction(
	left, right repositoryReviewPurgeIntent,
) bool {
	left.Phase = ""
	right.Phase = ""
	return reflect.DeepEqual(left, right)
}

func (s Store) purgeIntentPaths(intent repositoryReviewPurgeIntent) []string {
	paths := []string{s.purgeAutomationIntentPath(intent.AutomationID)}
	for _, repository := range repositoryReviewPurgeFenceRepositories(intent) {
		paths = append(paths, s.purgeRepositoryFencePath(repository))
	}
	return paths
}

func repositoryReviewPurgeFenceRepositories(intent repositoryReviewPurgeIntent) []string {
	repositories := []string{intent.Repository}
	for _, target := range intent.LedgerTargets {
		if !containsExactString(repositories, target.Repository) {
			repositories = append(repositories, target.Repository)
		}
	}
	for _, identity := range RepositoryLedgerIdentities(intent.ConfiguredRepository) {
		if !containsExactString(repositories, identity) {
			repositories = append(repositories, identity)
		}
	}
	return repositories
}

func (s Store) removePurgeIntent(intent repositoryReviewPurgeIntent) error {
	paths := s.purgeIntentPaths(intent)
	for _, path := range paths {
		current, found, err := s.loadPurgeIntentPath(path)
		if err != nil {
			return err
		}
		if found && !repositoryReviewPurgeIntentsSameTransaction(current, intent) {
			return ErrRepositoryReviewPurgeInProgress
		}
	}
	for index := len(paths) - 1; index >= 0; index-- {
		if err := removeRepositoryReviewRegularFile(paths[index]); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) repositoryReviewPurgeConfigured(repository string) (bool, error) {
	for _, identity := range RepositoryLedgerIdentities(repository) {
		if _, found, err := s.loadPurgeFence(identity); err != nil || found {
			return found, err
		}
	}
	return false, nil
}

func (s Store) purgeAutomationIntentPath(id string) string {
	return filepath.Join(s.root, "purge_automation_"+id+".json")
}

func (s Store) purgeRepositoryFencePath(repository string) string {
	return filepath.Join(
		s.root,
		"purge_repository_"+strings.TrimPrefix(RepositoryID(repository), "rrp_")+".json",
	)
}
