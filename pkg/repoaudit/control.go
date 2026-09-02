package repoaudit

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/fileutil"
)

const (
	RepositoryReviewAutomationSchemaVersion = 2

	DefaultRepositoryReviewAssignmentTimeoutSeconds = 3_600
	MinRepositoryReviewAssignmentTimeoutSeconds     = 60
	MaxRepositoryReviewAssignmentTimeoutSeconds     = 86_400

	defaultAutomationMaxFilesPerRun              = 24
	defaultAutomationMaxContentBytes             = 512 << 10
	defaultAutomationMaxParallelChildren         = 8
	defaultAutomationEstimatedOutputTokens       = 1_800
	maxAutomationFileBytes                 int64 = 4 << 20
	maxAutomationCount                           = 10_000
	maxAutomationRunIDs                          = 1_000
	maxAutomationReviewers                       = 32
	maxAutomationAccountSnapshots                = 256
	automationModelCoverageSketchBytes           = 8 << 10
	maxAutomationModelPrice                      = 1_000_000.0
	maxAutomationEstimatedCost                   = 1_000_000_000.0
	maxAutomationTokens                    int64 = 1_000_000_000_000
)

var (
	ErrInvalidAutomation                  = errors.New("invalid repository review automation")
	ErrAutomationControllerLocked         = errors.New("repository review automation controller is already active")
	ErrAutomationActive                   = errors.New("repository review automation is active")
	ErrRepositoryReviewRepositoryConflict = errors.New("repository already has a review configuration")
)

// RepositoryReviewAutomationStatus describes the durable controller state.
type RepositoryReviewAutomationStatus string

const (
	RepositoryReviewAutomationIdle      RepositoryReviewAutomationStatus = "idle"
	RepositoryReviewAutomationRunning   RepositoryReviewAutomationStatus = "running"
	RepositoryReviewAutomationStopping  RepositoryReviewAutomationStatus = "stopping"
	RepositoryReviewAutomationPaused    RepositoryReviewAutomationStatus = "paused"
	RepositoryReviewAutomationCompleted RepositoryReviewAutomationStatus = "completed"
	RepositoryReviewAutomationFailed    RepositoryReviewAutomationStatus = "failed"
)

// RepositoryReviewPauseReason records why a controller stopped admitting work.
type RepositoryReviewPauseReason string

const (
	RepositoryReviewPauseManual          RepositoryReviewPauseReason = "manual"
	RepositoryReviewPauseTokenBudget     RepositoryReviewPauseReason = "token_budget"
	RepositoryReviewPauseCostBudget      RepositoryReviewPauseReason = "cost_budget"
	RepositoryReviewPauseAccountLimit    RepositoryReviewPauseReason = "account_limit"
	RepositoryReviewPauseGuardExpression RepositoryReviewPauseReason = "guard_expression"
	RepositoryReviewPauseNoProgress      RepositoryReviewPauseReason = "no_progress"
	RepositoryReviewPauseRunFailed       RepositoryReviewPauseReason = "run_failed"
	RepositoryReviewPauseServiceRestart  RepositoryReviewPauseReason = "service_restart"
)

// RepositoryReviewBudgetPolicy controls task admission. GuardExpression is a
// bounded, diagnosis-independent predicate evaluated immediately before a
// managed review worker claims its next task.
type RepositoryReviewBudgetPolicy struct {
	GuardExpression string `json:"guard_expression,omitempty"`
}

// RepositoryReviewModelPrice is a server-resolved accounting snapshot keyed by
// the reviewer alias selected in ReviewerModels. Subscription inheritance is
// resolved from central model configuration before this snapshot is stored.
type RepositoryReviewModelPrice struct {
	InputPricePer1M  float64 `json:"input_price_per_1m"`
	OutputPricePer1M float64 `json:"output_price_per_1m"`
}

// RepositoryReviewTokenUsage is the cumulative token accounting accepted by
// the controller. CachedTokens is included in, rather than added to,
// PromptTokens when providers report it that way.
type RepositoryReviewTokenUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	CachedTokens     int64 `json:"cached_tokens,omitempty"`
	TotalTokens      int64 `json:"total_tokens"`
}

// RepositoryReviewAssignmentFocusProgress is the public coverage projection
// for one stable repository-review focus. Counts are file-assignment pairs,
// not files or provider calls.
type RepositoryReviewAssignmentFocusProgress struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Pending   int `json:"pending"`
	Active    int `json:"active"`
}

// RepositoryReviewAssignmentFocusesProgress keeps the public per-focus shape
// fixed to the four stable focus IDs used by every campaign catalog.
type RepositoryReviewAssignmentFocusesProgress struct {
	CorrectnessState      RepositoryReviewAssignmentFocusProgress `json:"correctness_state"`
	SecurityTrust         RepositoryReviewAssignmentFocusProgress `json:"security_trust"`
	ConcurrencyRecovery   RepositoryReviewAssignmentFocusProgress `json:"concurrency_recovery"`
	IntegrationValidation RepositoryReviewAssignmentFocusProgress `json:"integration_validation"`
}

// RepositoryReviewAssignmentProgress reports durable assignment coverage for
// the current campaign.
type RepositoryReviewAssignmentProgress struct {
	Total     int                                       `json:"total"`
	Completed int                                       `json:"completed"`
	Pending   int                                       `json:"pending"`
	Active    int                                       `json:"active"`
	ByFocus   RepositoryReviewAssignmentFocusesProgress `json:"by_focus"`
}

type RepositoryReviewProgress struct {
	Stage                string `json:"stage,omitempty"`
	CompletedBatches     int    `json:"completed_batches"`
	TotalBatches         int    `json:"total_batches"`
	CoverageAvailable    bool   `json:"coverage_available"`
	CoverageExact        bool   `json:"coverage_exact"`
	SelectedFiles        int    `json:"selected_files"`
	InspectedFiles       int    `json:"inspected_files"`
	ReviewedFiles        int    `json:"reviewed_files"`
	RemainingFiles       int    `json:"remaining_files"`
	UnsupportedFiles     int    `json:"unsupported_files"`
	RawFindings          int    `json:"raw_findings"`
	DeduplicatedFindings int    `json:"deduplicated_findings"`
	// Findings is the deprecated alias for DeduplicatedFindings.
	Findings               int                                `json:"findings"`
	FindingAggregates      int                                `json:"finding_aggregates"`
	PendingFindingMappings int                                `json:"unaggregated_findings"`
	AssignmentProgress     RepositoryReviewAssignmentProgress `json:"assignment_progress"`
	// ScopeFrozen is a public projection marker derived from the internal
	// durable scope selection. Stores never treat this caller-visible value as
	// campaign authority.
	ScopeFrozen bool `json:"scope_frozen,omitempty"`
}

// RepositoryReviewFileProgress is the truthful file-resolution projection for
// one campaign. Batch counts remain operational telemetry and never determine
// this percentage.
type RepositoryReviewFileProgress struct {
	ResolvedFiles int     `json:"resolved_files"`
	TotalFiles    int     `json:"total_files"`
	Percent       float64 `json:"percent"`
}

// RepositoryReviewAutomationFileProgress derives campaign progress from fully
// completed and unsupported files. A frozen scope supplies the authoritative
// total; legacy campaigns fall back to their durable file counters.
func RepositoryReviewAutomationFileProgress(
	automation RepositoryReviewAutomation,
) RepositoryReviewFileProgress {
	progress := automation.Progress
	if progress.CoverageAvailable && progress.CoverageExact {
		selected := max(0, progress.SelectedFiles)
		resolved := min(
			selected,
			max(0, progress.ReviewedFiles)+max(0, progress.UnsupportedFiles),
		)
		if automation.Status == RepositoryReviewAutomationCompleted {
			resolved = selected
		}
		return repositoryReviewFileProgress(resolved, selected, automation.Status)
	}
	selected := max(0, automation.ScopePlan.Counts.SelectedFiles)
	if automation.ScopeSelection != nil && selected > 0 {
		resolved := min(selected, max(0, selected-max(0, progress.RemainingFiles)))
		noFileEvidence := automation.Status != RepositoryReviewAutomationCompleted &&
			progress.ReviewedFiles == 0 && progress.RemainingFiles == 0 &&
			progress.UnsupportedFiles == 0
		if noFileEvidence {
			resolved = 0
		}
		if automation.Status == RepositoryReviewAutomationCompleted {
			resolved = selected
		}
		return repositoryReviewFileProgress(resolved, selected, automation.Status)
	}

	resolved := max(0, progress.ReviewedFiles) + max(0, progress.UnsupportedFiles)
	total := max(
		max(0, progress.ReviewedFiles)+max(0, progress.RemainingFiles)+
			max(0, progress.UnsupportedFiles),
		resolved,
	)
	if automation.Status == RepositoryReviewAutomationCompleted {
		resolved = total
	}
	return repositoryReviewFileProgress(resolved, total, automation.Status)
}

func repositoryReviewFileProgress(
	resolved int,
	total int,
	status RepositoryReviewAutomationStatus,
) RepositoryReviewFileProgress {
	resolved = min(max(0, resolved), max(0, total))
	total = max(0, total)
	percent := 0.0
	if status == RepositoryReviewAutomationCompleted {
		percent = 100
	} else if total > 0 {
		percent = math.Round(min(100, max(0, float64(resolved)/float64(total)*100)))
	}
	return RepositoryReviewFileProgress{
		ResolvedFiles: resolved,
		TotalFiles:    total,
		Percent:       percent,
	}
}

type RepositoryReviewModelStats struct {
	Tokens           RepositoryReviewTokenUsage `json:"tokens"`
	EstimatedCostUSD float64                    `json:"estimated_cost_usd"`
	Requests         int64                      `json:"requests"`
	Failures         int64                      `json:"failures"`
	Findings         int                        `json:"findings"`
	ReviewedFiles    int                        `json:"reviewed_files"`
	LatencyMillis    int64                      `json:"latency_millis"`
}

// RepositoryReviewAccountLimitSnapshot is a flattened account/window reading.
// A nil RemainingPercent means the account limit was unknown at CheckedAt.
type RepositoryReviewAccountLimitSnapshot struct {
	AccountID        string    `json:"account_id"`
	Name             string    `json:"name,omitempty"`
	Window           string    `json:"window"`
	RemainingPercent *float64  `json:"remaining_percent,omitempty"`
	ResetsAt         time.Time `json:"resets_at,omitempty"`
	CheckedAt        time.Time `json:"checked_at"`
	Detail           string    `json:"detail,omitempty"`
}

// RepositoryReviewAutomation is independent of a RepositoryState and may be
// created before the first review plan, run, or finding exists. ScopeSelection
// is internal durable controller state, not user configuration, and must be
// cleared together with ScopePlan.
type RepositoryReviewAutomation struct {
	SchemaVersion                    int                                    `json:"schema_version"`
	ID                               string                                 `json:"id"`
	Version                          int64                                  `json:"version"`
	ProfileID                        string                                 `json:"profile_id,omitempty"`
	ProfileVersion                   int64                                  `json:"profile_version,omitempty"`
	AccountRef                       string                                 `json:"account_ref,omitempty"`
	EffectiveAccountRef              string                                 `json:"effective_account_ref,omitempty"`
	Name                             string                                 `json:"name"`
	Repository                       string                                 `json:"repository"`
	Ref                              string                                 `json:"ref,omitempty"`
	ResolvedCommitSHA                string                                 `json:"resolved_commit_sha,omitempty"`
	ResolvedTargetBranch             string                                 `json:"resolved_target_branch,omitempty"`
	AdvertisedDefaultBranch          string                                 `json:"advertised_default_branch,omitempty"`
	TargetIsDefault                  bool                                   `json:"target_is_default"`
	Target                           string                                 `json:"target"`
	ReviewFocus                      string                                 `json:"review_focus"`
	ScopePolicy                      RepositoryReviewScopePolicy            `json:"scope_policy"`
	ScopePlan                        RepositoryReviewScopePlan              `json:"scope_plan"`
	ScopeSelection                   *RepositoryReviewScopeSelection        `json:"scope_selection,omitempty"`
	ReviewerModels                   []string                               `json:"reviewer_models"`
	DeduplicationModel               string                                 `json:"deduplication_model,omitempty"`
	DeduplicationSimilarityThreshold int                                    `json:"deduplication_similarity_threshold"`
	DeduplicationCandidateLimit      int                                    `json:"deduplication_candidate_limit"`
	DeduplicationSettingsSpecified   bool                                   `json:"-"`
	AccountModelRevision             string                                 `json:"account_model_revision,omitempty"`
	IssueWriterModel                 string                                 `json:"issue_writer_model"`
	CompareModels                    bool                                   `json:"compare_models"`
	ModelPrices                      map[string]RepositoryReviewModelPrice  `json:"model_prices,omitempty"`
	Force                            bool                                   `json:"force"`
	AutoContinue                     bool                                   `json:"auto_continue"`
	MaxFilesPerRun                   int                                    `json:"max_files_per_run"`
	MaxContentBytes                  int64                                  `json:"max_content_bytes"`
	MaxParallelChildren              int                                    `json:"max_parallel_children"`
	AssignmentTimeoutSeconds         int                                    `json:"assignment_timeout_seconds"`
	EstimatedOutputTokens            int                                    `json:"-"`
	BudgetPolicy                     RepositoryReviewBudgetPolicy           `json:"budget"`
	Status                           RepositoryReviewAutomationStatus       `json:"status"`
	PauseReason                      RepositoryReviewPauseReason            `json:"pause_reason,omitempty"`
	PauseDetail                      string                                 `json:"pause_detail,omitempty"`
	RequestedPauseReason             RepositoryReviewPauseReason            `json:"requested_pause_reason,omitempty"`
	RequestedPauseDetail             string                                 `json:"requested_pause_detail,omitempty"`
	CampaignID                       string                                 `json:"campaign_id,omitempty"`
	CampaignRecoveryPending          bool                                   `json:"campaign_recovery_pending,omitempty"`
	ActiveRunID                      string                                 `json:"active_run_id,omitempty"`
	RunIDs                           []string                               `json:"run_ids"`
	Usage                            RepositoryReviewTokenUsage             `json:"usage"`
	EstimatedCostUSD                 float64                                `json:"estimated_cost_usd"`
	Progress                         RepositoryReviewProgress               `json:"progress"`
	ModelStats                       map[string]RepositoryReviewModelStats  `json:"model_stats"`
	ModelCoverageSketches            map[string]string                      `json:"model_coverage_sketches,omitempty"`
	AccountLimitSnapshots            []RepositoryReviewAccountLimitSnapshot `json:"account_limits"`
	StartedAt                        time.Time                              `json:"started_at,omitempty"`
	CompletedAt                      time.Time                              `json:"completed_at,omitempty"`
	CreatedAt                        time.Time                              `json:"created_at"`
	UpdatedAt                        time.Time                              `json:"updated_at"`
}

func (s Store) ListAutomations(ctx context.Context) ([]RepositoryReviewAutomation, error) {
	return s.listAutomations(ctx, maxAutomationCount)
}

func (s Store) listAutomations(ctx context.Context, maximum int) ([]RepositoryReviewAutomation, error) {
	return s.listAutomationsWithLoader(ctx, maximum, s.loadAutomation)
}

func (s Store) listAutomationsWithLoader(
	ctx context.Context,
	maximum int,
	load func(string) (RepositoryReviewAutomation, bool, error),
) ([]RepositoryReviewAutomation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	unlock, lockErr := s.lock("repository-review-automations")
	if lockErr != nil {
		return nil, lockErr
	}
	defer unlock()
	if contextErr := ctx.Err(); contextErr != nil {
		return nil, contextErr
	}
	return s.listAutomationsUnlockedWithLoader(maximum, load)
}

func (s Store) listAutomationsUnlocked(maximum int) ([]RepositoryReviewAutomation, error) {
	return s.listAutomationsUnlockedWithLoader(maximum, s.loadAutomation)
}

func (s Store) listAutomationsUnlockedWithLoader(
	maximum int,
	load func(string) (RepositoryReviewAutomation, bool, error),
) ([]RepositoryReviewAutomation, error) {
	if rootErr := s.requireSafeRoot(true); rootErr != nil {
		return nil, rootErr
	}
	entries, err := os.ReadDir(s.root)
	if os.IsNotExist(err) {
		return []RepositoryReviewAutomation{}, nil
	}
	if err != nil {
		return nil, err
	}
	automations := make([]RepositoryReviewAutomation, 0)
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "automation_") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
			return nil, fmt.Errorf("repository review automation %q must be a regular file", entry.Name())
		}
		id := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "automation_"), ".json")
		if !validAutomationID(id) || entry.Name() != automationFilename(id) {
			return nil, fmt.Errorf("%w: invalid automation filename", ErrInvalidAutomation)
		}
		if len(automations) >= maximum {
			return nil, fmt.Errorf("%w: automation catalog exceeds its limit", ErrInvalidAutomation)
		}
		automation, found, loadErr := load(id)
		if loadErr != nil {
			return nil, loadErr
		}
		if !found {
			return nil, errors.New("repository review automation disappeared while locked")
		}
		automations = append(automations, automation)
	}
	sort.Slice(automations, func(i, j int) bool {
		if automations[i].UpdatedAt.Equal(automations[j].UpdatedAt) {
			return automations[i].ID < automations[j].ID
		}
		return automations[i].UpdatedAt.After(automations[j].UpdatedAt)
	})
	return automations, nil
}

func (s Store) GetAutomation(ctx context.Context, id string) (RepositoryReviewAutomation, bool, error) {
	if err := ctx.Err(); err != nil {
		return RepositoryReviewAutomation{}, false, err
	}
	id = strings.TrimSpace(id)
	if !validAutomationID(id) {
		return RepositoryReviewAutomation{}, false, fmt.Errorf("%w: invalid ID", ErrInvalidAutomation)
	}
	unlock, err := s.lock("automation:" + id)
	if err != nil {
		return RepositoryReviewAutomation{}, false, err
	}
	defer unlock()
	if err := ctx.Err(); err != nil {
		return RepositoryReviewAutomation{}, false, err
	}
	return s.loadAutomation(id)
}

func (s Store) CreateAutomation(
	ctx context.Context,
	automation RepositoryReviewAutomation,
) (RepositoryReviewAutomation, error) {
	if err := ctx.Err(); err != nil {
		return RepositoryReviewAutomation{}, err
	}
	automation = cloneAutomation(automation)
	if strings.TrimSpace(automation.ID) == "" {
		automation.ID = newAutomationID()
	}
	automation.ID = strings.TrimSpace(automation.ID)
	if !validAutomationID(automation.ID) {
		return RepositoryReviewAutomation{}, fmt.Errorf("%w: invalid ID", ErrInvalidAutomation)
	}
	unlock, err := s.lock("automation:" + automation.ID)
	if err != nil {
		return RepositoryReviewAutomation{}, err
	}
	defer unlock()
	if err := ctx.Err(); err != nil {
		return RepositoryReviewAutomation{}, err
	}
	if _, found, err := s.loadAutomation(automation.ID); err != nil {
		return RepositoryReviewAutomation{}, err
	} else if found {
		return RepositoryReviewAutomation{}, ErrConflict
	}
	now := s.clock()
	automation.SchemaVersion = RepositoryReviewAutomationSchemaVersion
	automation.Version = 1
	automation.CreatedAt = now
	automation.UpdatedAt = now
	if automation.Status == "" {
		automation.Status = RepositoryReviewAutomationIdle
	}
	if err := normalizeAutomation(&automation); err != nil {
		return RepositoryReviewAutomation{}, err
	}
	if purging, err := s.repositoryReviewPurgeConfigured(automation.Repository); err != nil {
		return RepositoryReviewAutomation{}, err
	} else if purging {
		return RepositoryReviewAutomation{}, ErrRepositoryReviewPurgeInProgress
	}
	if automation.ProfileID != "" {
		if err := s.validateAutomationProfileSnapshotUnlocked(automation); err != nil {
			return RepositoryReviewAutomation{}, err
		}
		if err := s.ensureRepositoryAutomationUniqueUnlocked(automation.ID, automation.Repository); err != nil {
			return RepositoryReviewAutomation{}, err
		}
	}
	if err := s.saveAutomation(automation); err != nil {
		return RepositoryReviewAutomation{}, err
	}
	return cloneAutomation(automation), nil
}

func (s Store) UpdateAutomation(
	ctx context.Context,
	id string,
	expectedVersion int64,
	mutate func(*RepositoryReviewAutomation) error,
) (RepositoryReviewAutomation, error) {
	if err := ctx.Err(); err != nil {
		return RepositoryReviewAutomation{}, err
	}
	id = strings.TrimSpace(id)
	if !validAutomationID(id) || mutate == nil {
		return RepositoryReviewAutomation{}, fmt.Errorf("%w: invalid update", ErrInvalidAutomation)
	}
	unlock, lockErr := s.lock("automation:" + id)
	if lockErr != nil {
		return RepositoryReviewAutomation{}, lockErr
	}
	defer unlock()
	if contextErr := ctx.Err(); contextErr != nil {
		return RepositoryReviewAutomation{}, contextErr
	}
	current, found, err := s.loadAutomation(id)
	if err != nil {
		return RepositoryReviewAutomation{}, err
	}
	if !found {
		return RepositoryReviewAutomation{}, os.ErrNotExist
	}
	if expectedVersion < 1 || current.Version != expectedVersion {
		return RepositoryReviewAutomation{}, ErrConflict
	}
	candidate := cloneAutomation(current)
	if err := mutate(&candidate); err != nil {
		return RepositoryReviewAutomation{}, err
	}
	if candidate.ID != current.ID || candidate.Version != current.Version ||
		!candidate.CreatedAt.Equal(current.CreatedAt) || candidate.SchemaVersion != current.SchemaVersion {
		return RepositoryReviewAutomation{}, fmt.Errorf("%w: immutable fields changed", ErrInvalidAutomation)
	}
	// A controller callback may assign slices, maps, or snapshot pointers owned
	// by its request. Detach them before normalization and persistence.
	candidate = cloneAutomation(candidate)
	candidate.Version++
	candidate.UpdatedAt = s.clock()
	if err := normalizeAutomation(&candidate); err != nil {
		return RepositoryReviewAutomation{}, err
	}
	if candidate.ProfileID != "" {
		if current.ProfileID != candidate.ProfileID ||
			current.ProfileVersion != candidate.ProfileVersion ||
			!repositoryReviewAutomationProfilePolicyEqual(current, candidate) ||
			repositoryReviewAutomationAdmissionTransition(current, candidate) {
			if err := s.validateAutomationProfileSnapshotUnlocked(candidate); err != nil {
				return RepositoryReviewAutomation{}, err
			}
		}
	}
	if canonicalAutomationRepository(current.Repository) !=
		canonicalAutomationRepository(candidate.Repository) ||
		candidate.ProfileID != "" &&
			repositoryReviewAutomationAdmissionTransition(current, candidate) {
		if err := s.ensureRepositoryAutomationUniqueUnlocked(candidate.ID, candidate.Repository); err != nil {
			return RepositoryReviewAutomation{}, err
		}
	}
	if err := s.saveAutomation(candidate); err != nil {
		return RepositoryReviewAutomation{}, err
	}
	return cloneAutomation(candidate), nil
}

// DeleteAutomation is the legacy configuration-only storage primitive.
// Product/API removal must use DeleteAutomationAndHistory so a repository
// assignment cannot leave an undiscoverable ledger behind.
//
// Deprecated: use DeleteAutomationAndHistory.
func (s Store) DeleteAutomation(ctx context.Context, id string, expectedVersion int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if !validAutomationID(id) {
		return fmt.Errorf("%w: invalid ID", ErrInvalidAutomation)
	}
	unlock, lockErr := s.lock("automation:" + id)
	if lockErr != nil {
		return lockErr
	}
	defer unlock()
	if contextErr := ctx.Err(); contextErr != nil {
		return contextErr
	}
	automation, found, err := s.loadAutomation(id)
	if err != nil {
		return err
	}
	if !found {
		return os.ErrNotExist
	}
	if expectedVersion < 1 || automation.Version != expectedVersion {
		return ErrConflict
	}
	if automation.Status == RepositoryReviewAutomationRunning ||
		automation.Status == RepositoryReviewAutomationStopping {
		return ErrAutomationActive
	}
	return fileutil.RemoveDurable(s.automationPath(id))
}

func (s Store) loadAutomation(id string) (RepositoryReviewAutomation, bool, error) {
	return s.loadAutomationState(id, true)
}

func (s Store) loadAutomationIgnoringPurge(id string) (RepositoryReviewAutomation, bool, error) {
	return s.loadAutomationState(id, false)
}

func (s Store) loadAutomationState(
	id string,
	enforcePurgeFence bool,
) (RepositoryReviewAutomation, bool, error) {
	if !validAutomationID(id) {
		return RepositoryReviewAutomation{}, false, fmt.Errorf("%w: invalid ID", ErrInvalidAutomation)
	}
	statePath := s.automationPath(id)
	data, found, err := s.readStateFile(
		statePath,
		maxAutomationFileBytes,
		"repository review automation",
	)
	if err != nil {
		return RepositoryReviewAutomation{}, false, err
	}
	if !found {
		return RepositoryReviewAutomation{}, false, nil
	}
	var automation RepositoryReviewAutomation
	hadLegacyGuard, err := unmarshalRepositoryReviewGuardState(data, &automation)
	if err != nil {
		return RepositoryReviewAutomation{}, false, err
	}
	var persistedFields map[string]json.RawMessage
	_ = json.Unmarshal(data, &persistedFields)
	_, hadDeduplicationThreshold := persistedFields["deduplication_similarity_threshold"]
	_, hadDeduplicationCandidateLimit := persistedFields["deduplication_candidate_limit"]
	var persistedProgress map[string]json.RawMessage
	_ = json.Unmarshal(persistedFields["progress"], &persistedProgress)
	_, hadRawFindingProgress := persistedProgress["raw_findings"]
	_, hadDeduplicatedFindingProgress := persistedProgress["deduplicated_findings"]
	if !hadDeduplicatedFindingProgress {
		automation.Progress.DeduplicatedFindings = automation.Progress.Findings
	}
	automation.DeduplicationSettingsSpecified = hadDeduplicationThreshold ||
		hadDeduplicationCandidateLimit
	// unmarshalRepositoryReviewGuardState already decoded this exact JSON.
	legacy, _ := decodeLegacyAutomationPriceMetadata(data)
	hasLegacyPriceMetadata := false
	for _, price := range legacy.ModelPrices {
		if _, exists := price["subscription"]; exists {
			hasLegacyPriceMetadata = true
		}
		if _, exists := price["equivalent_model"]; exists {
			hasLegacyPriceMetadata = true
		}
	}
	hadLegacyIssueWriter := strings.TrimSpace(automation.IssueWriterModel) == ""
	hadLegacyAssignmentTimeout := automation.AssignmentTimeoutSeconds == 0
	if automation.ID != id {
		return RepositoryReviewAutomation{}, false, errors.New("repository review automation identity mismatch")
	}
	if enforcePurgeFence {
		if _, purging, purgeErr := s.loadPurgeIntentForAutomation(id); purgeErr != nil {
			return RepositoryReviewAutomation{}, false, purgeErr
		} else if purging {
			return RepositoryReviewAutomation{}, false, ErrRepositoryReviewPurgeInProgress
		}
	}
	hadLegacySchema := false
	if automation.SchemaVersion == 1 {
		automation.SchemaVersion = RepositoryReviewAutomationSchemaVersion
		hadLegacySchema = true
	}
	if err := normalizeAutomation(&automation); err != nil {
		return RepositoryReviewAutomation{}, false, err
	}
	if hasLegacyPriceMetadata || hadLegacyGuard || hadLegacyIssueWriter ||
		hadLegacyAssignmentTimeout || hadLegacySchema ||
		!hadDeduplicationThreshold || !hadDeduplicationCandidateLimit ||
		!hadRawFindingProgress || !hadDeduplicatedFindingProgress {
		if err := s.saveAutomation(automation); err != nil {
			return RepositoryReviewAutomation{}, false, err
		}
	}
	return automation, true, nil
}

func (s Store) readStateFile(
	statePath string,
	maximumBytes int64,
	kind string,
) ([]byte, bool, error) {
	if err := s.requireSafeRoot(true); err != nil {
		return nil, false, err
	}
	info, err := os.Lstat(statePath)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, false, fmt.Errorf("%s must be a regular file", kind)
	}
	if info.Size() > maximumBytes {
		return nil, false, fmt.Errorf("%s exceeds its size limit", kind)
	}
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func (s Store) saveAutomation(automation RepositoryReviewAutomation) error {
	if err := normalizeAutomation(&automation); err != nil {
		return err
	}
	if err := s.ensureSafeRoot(fileutil.MkdirAllDurable); err != nil {
		return err
	}
	statePath := s.automationPath(automation.ID)
	if info, err := os.Lstat(statePath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return errors.New("repository review automation must be a regular file")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	data, err := json.Marshal(automation)
	if err != nil {
		return err
	}
	if err := validateEncodedAutomationSize(data); err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(statePath, data, 0o600)
}

type legacyAutomationPriceMetadata struct {
	ModelPrices map[string]map[string]json.RawMessage `json:"model_prices"`
}

func decodeLegacyAutomationPriceMetadata(data []byte) (legacyAutomationPriceMetadata, error) {
	var legacy legacyAutomationPriceMetadata
	err := json.Unmarshal(data, &legacy)
	return legacy, err
}

func validateEncodedAutomationSize(data []byte) error {
	if int64(len(data)) > maxAutomationFileBytes {
		return errors.New("repository review automation exceeds its size limit")
	}
	return nil
}

func (s Store) automationPath(id string) string {
	return filepath.Join(s.root, automationFilename(id))
}

func automationFilename(id string) string {
	return "automation_" + id + ".json"
}

func newAutomationID() string {
	return "rra_" + strings.ToLower(rand.Text())
}

func validAutomationID(id string) bool {
	if !strings.HasPrefix(id, "rra_") || len(id) < 5 || len(id) > 128 {
		return false
	}
	for index, character := range id[4:] {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' ||
			index > 0 && (character == '_' || character == '-') {
			continue
		}
		return false
	}
	return true
}

// ValidRepositoryReviewAutomationID reports whether id is already in the
// canonical durable automation identity form.
func ValidRepositoryReviewAutomationID(id string) bool {
	return validAutomationID(id)
}

func normalizeAutomation(automation *RepositoryReviewAutomation) error {
	if automation == nil {
		return fmt.Errorf("%w: state is required", ErrInvalidAutomation)
	}
	rawRef := automation.Ref
	automation.ID = strings.TrimSpace(automation.ID)
	automation.ProfileID = strings.TrimSpace(automation.ProfileID)
	automation.AccountRef = strings.TrimSpace(automation.AccountRef)
	automation.EffectiveAccountRef = strings.TrimSpace(automation.EffectiveAccountRef)
	automation.Name = strings.TrimSpace(automation.Name)
	automation.Repository = strings.TrimSpace(automation.Repository)
	automation.Ref = strings.TrimSpace(automation.Ref)
	automation.ResolvedCommitSHA = strings.ToLower(strings.TrimSpace(automation.ResolvedCommitSHA))
	automation.ResolvedTargetBranch = strings.TrimSpace(automation.ResolvedTargetBranch)
	automation.AdvertisedDefaultBranch = strings.TrimSpace(automation.AdvertisedDefaultBranch)
	automation.Target = strings.TrimSpace(automation.Target)
	automation.ReviewFocus = strings.TrimSpace(automation.ReviewFocus)
	automation.DeduplicationModel = strings.TrimSpace(automation.DeduplicationModel)
	automation.AccountModelRevision = strings.TrimSpace(automation.AccountModelRevision)
	automation.IssueWriterModel = strings.TrimSpace(automation.IssueWriterModel)
	automation.BudgetPolicy.GuardExpression = strings.TrimSpace(automation.BudgetPolicy.GuardExpression)
	automation.PauseDetail = strings.TrimSpace(automation.PauseDetail)
	automation.RequestedPauseDetail = strings.TrimSpace(automation.RequestedPauseDetail)
	automation.CampaignID = strings.TrimSpace(automation.CampaignID)
	automation.ActiveRunID = strings.TrimSpace(automation.ActiveRunID)
	automation.Progress.Stage = strings.TrimSpace(automation.Progress.Stage)
	if automation.Progress.DeduplicatedFindings == 0 && automation.Progress.Findings > 0 {
		automation.Progress.DeduplicatedFindings = automation.Progress.Findings
	}
	if automation.Progress.Findings == 0 && automation.Progress.DeduplicatedFindings > 0 {
		automation.Progress.Findings = automation.Progress.DeduplicatedFindings
	}
	automation.Status = RepositoryReviewAutomationStatus(strings.ToLower(strings.TrimSpace(string(automation.Status))))
	automation.PauseReason = RepositoryReviewPauseReason(
		strings.ToLower(strings.TrimSpace(string(automation.PauseReason))),
	)
	if automation.ProfileID != "" {
		branch, err := NormalizeRepositoryReviewBranch(rawRef)
		if err != nil {
			return err
		}
		automation.Ref = branch
		automation.Target = "all"
	}
	if automation.Target == "" {
		automation.Target = "all"
	}
	if automation.Ref == "" {
		automation.TargetIsDefault = true
	}
	if automation.MaxFilesPerRun == 0 {
		automation.MaxFilesPerRun = defaultAutomationMaxFilesPerRun
	}
	if automation.MaxContentBytes == 0 {
		automation.MaxContentBytes = defaultAutomationMaxContentBytes
	}
	if automation.MaxParallelChildren == 0 {
		automation.MaxParallelChildren = defaultAutomationMaxParallelChildren
	}
	if automation.AssignmentTimeoutSeconds == 0 {
		automation.AssignmentTimeoutSeconds = DefaultRepositoryReviewAssignmentTimeoutSeconds
	}
	if !automation.DeduplicationSettingsSpecified &&
		automation.DeduplicationSimilarityThreshold == 0 &&
		automation.DeduplicationCandidateLimit == 0 {
		automation.DeduplicationSimilarityThreshold = DeduplicationDefaultThreshold
		automation.DeduplicationCandidateLimit = DeduplicationDefaultCandidateLimit
	}
	automation.DeduplicationSettingsSpecified = true
	if automation.EstimatedOutputTokens == 0 {
		automation.EstimatedOutputTokens = defaultAutomationEstimatedOutputTokens
	}
	var err error
	automation.ReviewerModels, err = normalizeUniqueAutomationStrings(
		automation.ReviewerModels, maxAutomationReviewers, 256, "reviewer model",
	)
	if err != nil {
		return err
	}
	if len(automation.RunIDs) > maxAutomationRunIDs {
		automation.RunIDs = append([]string(nil), automation.RunIDs[len(automation.RunIDs)-maxAutomationRunIDs:]...)
	}
	if automation.IssueWriterModel == "" && len(automation.ReviewerModels) > 0 {
		automation.IssueWriterModel = automation.ReviewerModels[0]
	}
	automation.RunIDs, err = normalizeUniqueAutomationStrings(
		automation.RunIDs, maxAutomationRunIDs, 1024, "run ID",
	)
	if err != nil {
		return err
	}
	if automation.Usage.TotalTokens == 0 &&
		automation.Usage.PromptTokens >= 0 && automation.Usage.CompletionTokens >= 0 {
		automation.Usage.TotalTokens = automation.Usage.PromptTokens + automation.Usage.CompletionTokens
	}
	if err := normalizeModelPrices(automation); err != nil {
		return err
	}
	if err := normalizeModelStats(automation); err != nil {
		return err
	}
	if err := normalizeModelCoverageSketches(automation); err != nil {
		return err
	}
	if err := normalizeRepositoryReviewScopePolicy(&automation.ScopePolicy); err != nil {
		return err
	}
	if err := normalizeRepositoryReviewScopePlan(&automation.ScopePlan); err != nil {
		return err
	}
	if automation.ScopeSelection != nil {
		selection := *automation.ScopeSelection
		if err := normalizeRepositoryReviewScopeSelection(&selection); err != nil {
			return err
		}
		automation.ScopeSelection = &selection
		if repositoryReviewScopePlanEmpty(automation.ScopePlan) {
			return fmt.Errorf(
				"%w: frozen scope selection requires a commit-bound scope plan",
				ErrInvalidAutomation,
			)
		}
	}
	if err := normalizeAccountSnapshots(automation); err != nil {
		return err
	}
	automation.StartedAt = automation.StartedAt.UTC()
	automation.CompletedAt = automation.CompletedAt.UTC()
	automation.CreatedAt = automation.CreatedAt.UTC()
	automation.UpdatedAt = automation.UpdatedAt.UTC()
	return validateAutomation(*automation)
}

func validateAutomation(automation RepositoryReviewAutomation) error {
	for _, branch := range []string{automation.ResolvedTargetBranch, automation.AdvertisedDefaultBranch} {
		if branch == "" {
			continue
		}
		if normalized, err := NormalizeRepositoryReviewBranch(branch); err != nil || normalized != branch {
			return ErrInvalidAutomation
		}
	}
	if automation.ResolvedTargetBranch != "" && automation.AdvertisedDefaultBranch != "" &&
		automation.TargetIsDefault !=
			(automation.ResolvedTargetBranch == automation.AdvertisedDefaultBranch) {
		return ErrInvalidAutomation
	}
	if automation.SchemaVersion != RepositoryReviewAutomationSchemaVersion ||
		!validAutomationID(automation.ID) || automation.Version < 1 ||
		!validBoundedText(automation.Name, 256) ||
		!validBoundedText(automation.Repository, maxRepositoryIdentityBytes) ||
		!validAutomationRepository(automation.Repository) ||
		!validOptionalAutomationText(automation.Ref, 1024) ||
		(automation.ResolvedCommitSHA != "" &&
			!validRepositoryReviewCommitSHA(automation.ResolvedCommitSHA)) ||
		!validOptionalAutomationText(automation.ResolvedTargetBranch, maxRepositoryReviewBranchBytes) ||
		!validOptionalAutomationText(automation.AdvertisedDefaultBranch, maxRepositoryReviewBranchBytes) ||
		!validOptionalAutomationText(automation.AccountRef, 256) ||
		!validOptionalAutomationText(automation.EffectiveAccountRef, 256) ||
		!validOptionalAutomationText(automation.DeduplicationModel, 256) ||
		!validOptionalAutomationText(automation.AccountModelRevision, 256) ||
		automation.DeduplicationSimilarityThreshold < 0 ||
		automation.DeduplicationSimilarityThreshold > 100 ||
		automation.DeduplicationCandidateLimit < 0 ||
		automation.DeduplicationCandidateLimit > DeduplicationMaximumShortlist ||
		!validBoundedText(automation.Target, 4096) ||
		!validBoundedText(automation.ReviewFocus, maxFindingTextBytes) ||
		!validBoundedText(automation.IssueWriterModel, 256) ||
		len(automation.ReviewerModels) == 0 || len(automation.ReviewerModels) > maxAutomationReviewers ||
		automation.CompareModels && len(automation.ReviewerModels) < 2 ||
		automation.MaxFilesPerRun < 1 || automation.MaxFilesPerRun > maxReviewFiles ||
		automation.MaxContentBytes < 1 || automation.MaxContentBytes > defaultAutomationMaxContentBytes ||
		automation.MaxParallelChildren < 1 || automation.MaxParallelChildren > 64 ||
		automation.AssignmentTimeoutSeconds < MinRepositoryReviewAssignmentTimeoutSeconds ||
		automation.AssignmentTimeoutSeconds > MaxRepositoryReviewAssignmentTimeoutSeconds ||
		automation.AssignmentTimeoutSeconds%60 != 0 ||
		automation.EstimatedOutputTokens < 1 || automation.EstimatedOutputTokens > 65_536 ||
		!validOptionalAutomationText(automation.PauseDetail, 4096) ||
		!validOptionalAutomationText(automation.RequestedPauseDetail, 4096) ||
		(automation.CampaignID != "" && !ValidRepositoryReviewCampaignID(automation.CampaignID)) ||
		(automation.CampaignRecoveryPending && (automation.CampaignID == "" ||
			automation.ScopeSelection == nil || repositoryReviewScopePlanEmpty(automation.ScopePlan) ||
			automation.ResolvedCommitSHA == "" ||
			automation.ScopePlan.CommitSHA != automation.ResolvedCommitSHA ||
			len(automation.RunIDs) == 0 || automation.StartedAt.IsZero() ||
			!automation.CompletedAt.IsZero() || automation.ActiveRunID != "" ||
			(automation.Status != RepositoryReviewAutomationPaused &&
				automation.Status != RepositoryReviewAutomationFailed &&
				(automation.Status != RepositoryReviewAutomationIdle ||
					automation.Progress.Stage != "next batch queued")))) ||
		!validOptionalAutomationText(automation.ActiveRunID, 1024) ||
		len(automation.RunIDs) > maxAutomationRunIDs ||
		!finiteNonnegative(automation.EstimatedCostUSD, maxAutomationEstimatedCost) ||
		automation.CreatedAt.IsZero() || automation.UpdatedAt.IsZero() ||
		automation.UpdatedAt.Before(automation.CreatedAt) {
		return ErrInvalidAutomation
	}
	if automation.ProfileID == "" {
		if automation.ProfileVersion != 0 {
			return fmt.Errorf("%w: profile version requires a profile", ErrInvalidAutomation)
		}
	} else if !validProfileID(automation.ProfileID) || automation.ProfileVersion < 1 ||
		automation.Target != "all" || len(automation.ReviewerModels) != 1 || automation.CompareModels {
		return fmt.Errorf("%w: invalid profile assignment", ErrInvalidAutomation)
	}
	if err := validateBudgetPolicy(automation.BudgetPolicy); err != nil {
		return err
	}
	if err := validateTokenUsage(automation.Usage); err != nil {
		return err
	}
	if err := validateProgress(automation.Progress); err != nil {
		return err
	}
	if !validAutomationStatus(automation.Status) || !validAutomationPauseReason(automation.PauseReason) ||
		!validAutomationPauseReason(automation.RequestedPauseReason) {
		return ErrInvalidAutomation
	}
	switch automation.Status {
	case RepositoryReviewAutomationRunning:
		if automation.ActiveRunID == "" || automation.PauseReason != "" || automation.PauseDetail != "" ||
			automation.RequestedPauseReason != "" || automation.RequestedPauseDetail != "" {
			return fmt.Errorf("%w: running status requires a run and no pause request", ErrInvalidAutomation)
		}
	case RepositoryReviewAutomationStopping:
		if automation.ActiveRunID == "" || automation.PauseReason != "" || automation.PauseDetail != "" ||
			automation.RequestedPauseReason == "" {
			return fmt.Errorf("%w: stopping status requires a run and requested pause reason", ErrInvalidAutomation)
		}
	case RepositoryReviewAutomationPaused:
		if automation.ActiveRunID != "" || automation.PauseReason == "" ||
			automation.RequestedPauseReason != "" || automation.RequestedPauseDetail != "" {
			return fmt.Errorf("%w: paused status requires a reason and no active run", ErrInvalidAutomation)
		}
	case RepositoryReviewAutomationFailed:
		if automation.ActiveRunID != "" || automation.PauseReason != RepositoryReviewPauseRunFailed ||
			automation.RequestedPauseReason != "" || automation.RequestedPauseDetail != "" {
			return fmt.Errorf("%w: failed status requires run_failed", ErrInvalidAutomation)
		}
	case RepositoryReviewAutomationIdle, RepositoryReviewAutomationCompleted:
		if automation.ActiveRunID != "" || automation.PauseReason != "" || automation.PauseDetail != "" ||
			automation.RequestedPauseReason != "" || automation.RequestedPauseDetail != "" {
			return fmt.Errorf("%w: inactive status has active pause or run state", ErrInvalidAutomation)
		}
	}
	if automation.ActiveRunID != "" && !containsAutomationString(automation.RunIDs, automation.ActiveRunID) {
		return fmt.Errorf("%w: active run is missing from run history", ErrInvalidAutomation)
	}
	return nil
}

func validAutomationRepository(repository string) bool {
	repository = strings.TrimSpace(repository)
	if repository == "" || !strings.Contains(repository, "://") {
		return repository != ""
	}
	parsed, err := url.Parse(repository)
	return err == nil && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == ""
}

func (s Store) ensureRepositoryAutomationUniqueUnlocked(id, repository string) error {
	canonical := canonicalAutomationRepository(repository)
	if canonical == "" {
		return ErrInvalidAutomation
	}
	automations, err := s.listAutomationsUnlocked(maxAutomationCount)
	if err != nil {
		return err
	}
	for _, existing := range automations {
		if existing.ID != id && canonicalAutomationRepository(existing.Repository) == canonical {
			return ErrRepositoryReviewRepositoryConflict
		}
	}
	return nil
}

func (s Store) validateAutomationProfileSnapshotUnlocked(
	automation RepositoryReviewAutomation,
) error {
	profile, found, err := s.loadProfile(automation.ProfileID)
	if err != nil {
		return err
	}
	if !found {
		return os.ErrNotExist
	}
	if profile.Version != automation.ProfileVersion {
		return ErrConflict
	}
	materialized, err := MaterializeRepositoryReviewAutomation(profile, automation)
	if err != nil {
		return err
	}
	if automation.ReviewFocus != materialized.ReviewFocus ||
		automation.AccountRef != materialized.AccountRef ||
		!reflect.DeepEqual(automation.ScopePolicy, materialized.ScopePolicy) ||
		!reflect.DeepEqual(automation.ReviewerModels, materialized.ReviewerModels) ||
		automation.DeduplicationModel != materialized.DeduplicationModel ||
		automation.DeduplicationSimilarityThreshold != materialized.DeduplicationSimilarityThreshold ||
		automation.DeduplicationCandidateLimit != materialized.DeduplicationCandidateLimit ||
		automation.IssueWriterModel != materialized.IssueWriterModel ||
		automation.CompareModels != materialized.CompareModels ||
		!reflect.DeepEqual(automation.ModelPrices, materialized.ModelPrices) ||
		automation.Force != materialized.Force ||
		automation.AutoContinue != materialized.AutoContinue ||
		automation.MaxFilesPerRun != materialized.MaxFilesPerRun ||
		automation.MaxContentBytes != materialized.MaxContentBytes ||
		automation.MaxParallelChildren != materialized.MaxParallelChildren ||
		automation.AssignmentTimeoutSeconds != materialized.AssignmentTimeoutSeconds ||
		automation.EstimatedOutputTokens != materialized.EstimatedOutputTokens ||
		!reflect.DeepEqual(automation.BudgetPolicy, materialized.BudgetPolicy) ||
		automation.Target != materialized.Target {
		return fmt.Errorf(
			"%w: repository review profile snapshot does not match its assigned profile",
			ErrInvalidAutomation,
		)
	}
	return nil
}

func repositoryReviewAutomationProfilePolicyEqual(
	left, right RepositoryReviewAutomation,
) bool {
	return left.ReviewFocus == right.ReviewFocus &&
		left.AccountRef == right.AccountRef &&
		reflect.DeepEqual(left.ScopePolicy, right.ScopePolicy) &&
		reflect.DeepEqual(left.ReviewerModels, right.ReviewerModels) &&
		left.DeduplicationModel == right.DeduplicationModel &&
		left.DeduplicationSimilarityThreshold == right.DeduplicationSimilarityThreshold &&
		left.DeduplicationCandidateLimit == right.DeduplicationCandidateLimit &&
		left.IssueWriterModel == right.IssueWriterModel &&
		left.CompareModels == right.CompareModels &&
		reflect.DeepEqual(left.ModelPrices, right.ModelPrices) &&
		left.Force == right.Force && left.AutoContinue == right.AutoContinue &&
		left.MaxFilesPerRun == right.MaxFilesPerRun &&
		left.MaxContentBytes == right.MaxContentBytes &&
		left.MaxParallelChildren == right.MaxParallelChildren &&
		left.AssignmentTimeoutSeconds == right.AssignmentTimeoutSeconds &&
		left.EstimatedOutputTokens == right.EstimatedOutputTokens &&
		reflect.DeepEqual(left.BudgetPolicy, right.BudgetPolicy) &&
		left.Target == right.Target
}

func repositoryReviewAutomationAdmissionTransition(
	current, candidate RepositoryReviewAutomation,
) bool {
	currentActive := current.Status == RepositoryReviewAutomationRunning ||
		current.Status == RepositoryReviewAutomationStopping
	candidateActive := candidate.Status == RepositoryReviewAutomationRunning ||
		candidate.Status == RepositoryReviewAutomationStopping
	return !currentActive && candidateActive
}

func canonicalAutomationRepository(repository string) string {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return ""
	}
	if github := GitHubRepositoryIdentity(repository); github != "" {
		return github
	}
	if filepath.IsAbs(repository) {
		return filepath.Clean(repository)
	}
	if parsed, err := url.Parse(repository); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		pathValue := strings.Trim(strings.TrimSuffix(parsed.Path, "/"), "/")
		pathValue = strings.TrimSuffix(pathValue, ".git")
		return strings.ToLower(parsed.Hostname() + "/" + pathValue)
	}
	if identity, remotePath, ok := strings.Cut(repository, ":"); ok {
		host := identity
		if _, parsedHost, hasUser := strings.Cut(identity, "@"); hasUser {
			host = parsedHost
		}
		if strings.TrimSpace(host) != "" && strings.TrimSpace(remotePath) != "" {
			return strings.ToLower(strings.TrimSpace(host) + "/" +
				strings.TrimSuffix(strings.Trim(remotePath, "/"), ".git"))
		}
	}
	return strings.ToLower(strings.TrimSuffix(strings.Trim(repository, "/"), ".git"))
}

func validateBudgetPolicy(policy RepositoryReviewBudgetPolicy) error {
	if err := ValidateRepositoryReviewGuardExpression(policy.GuardExpression); err != nil {
		return fmt.Errorf("%w: invalid task admission guard: %v", ErrInvalidAutomation, err)
	}
	return nil
}

func validateTokenUsage(usage RepositoryReviewTokenUsage) error {
	if usage.PromptTokens < 0 || usage.CompletionTokens < 0 || usage.CachedTokens < 0 ||
		usage.TotalTokens < usage.PromptTokens+usage.CompletionTokens ||
		usage.PromptTokens > maxAutomationTokens || usage.CompletionTokens > maxAutomationTokens ||
		usage.CachedTokens > usage.PromptTokens || usage.TotalTokens > maxAutomationTokens {
		return fmt.Errorf("%w: invalid token usage", ErrInvalidAutomation)
	}
	return nil
}

func validateProgress(progress RepositoryReviewProgress) error {
	if !validOptionalAutomationText(progress.Stage, 256) || progress.CompletedBatches < 0 ||
		progress.TotalBatches < 0 || progress.CompletedBatches > progress.TotalBatches ||
		progress.CoverageExact && !progress.CoverageAvailable ||
		progress.SelectedFiles < 0 || progress.SelectedFiles > maxReviewFiles ||
		progress.InspectedFiles < 0 || progress.InspectedFiles > maxReviewFiles ||
		progress.ReviewedFiles < 0 || progress.ReviewedFiles > maxReviewFiles ||
		progress.RemainingFiles < 0 || progress.RemainingFiles > maxReviewFiles ||
		progress.UnsupportedFiles < 0 || progress.UnsupportedFiles > maxReviewFiles ||
		progress.RawFindings < 0 || progress.RawFindings > maxReviewObservations ||
		progress.DeduplicatedFindings < 0 || progress.DeduplicatedFindings > maxReviewObservations ||
		progress.Findings < 0 || progress.Findings > maxReviewObservations ||
		progress.DeduplicatedFindings != progress.Findings ||
		progress.FindingAggregates < 0 || progress.FindingAggregates > maxReviewObservations ||
		progress.PendingFindingMappings < 0 || progress.PendingFindingMappings > maxReviewObservations ||
		progress.FindingAggregates+progress.PendingFindingMappings > progress.Findings ||
		(progress.CoverageAvailable && (progress.InspectedFiles > progress.SelectedFiles ||
			progress.ReviewedFiles+progress.UnsupportedFiles > progress.SelectedFiles ||
			progress.CoverageExact && progress.RemainingFiles !=
				progress.SelectedFiles-progress.ReviewedFiles-progress.UnsupportedFiles)) ||
		!validRepositoryReviewAssignmentProgress(progress.AssignmentProgress) {
		return fmt.Errorf("%w: invalid progress", ErrInvalidAutomation)
	}
	return nil
}

func validRepositoryReviewAssignmentProgress(progress RepositoryReviewAssignmentProgress) bool {
	maximum := maxReviewFiles * maxRepositoryReviewRequiredAssignments
	validCounts := func(total, completed, pending, active int) bool {
		return total >= 0 && total <= maximum &&
			completed >= 0 && completed <= total &&
			pending >= 0 && pending <= total &&
			active >= 0 && active <= total
	}
	if !validCounts(progress.Total, progress.Completed, progress.Pending, progress.Active) {
		return false
	}
	for _, counts := range []RepositoryReviewAssignmentFocusProgress{
		progress.ByFocus.CorrectnessState,
		progress.ByFocus.SecurityTrust,
		progress.ByFocus.ConcurrencyRecovery,
		progress.ByFocus.IntegrationValidation,
	} {
		if !validCounts(counts.Total, counts.Completed, counts.Pending, counts.Active) {
			return false
		}
	}
	return true
}

func normalizeModelPrices(automation *RepositoryReviewAutomation) error {
	if len(automation.ModelPrices) > maxAutomationReviewers {
		return fmt.Errorf("%w: too many model prices", ErrInvalidAutomation)
	}
	selected := automationReviewerSet(automation.ReviewerModels)
	normalized := make(map[string]RepositoryReviewModelPrice, len(automation.ModelPrices))
	for rawAlias, price := range automation.ModelPrices {
		alias := strings.TrimSpace(rawAlias)
		if alias == "" || alias != rawAlias && containsAutomationMapKey(normalized, alias) {
			return fmt.Errorf("%w: duplicate model price alias", ErrInvalidAutomation)
		}
		if _, exists := selected[alias]; !exists || !validBoundedText(alias, 256) ||
			!finiteNonnegative(price.InputPricePer1M, maxAutomationModelPrice) ||
			!finiteNonnegative(price.OutputPricePer1M, maxAutomationModelPrice) {
			return fmt.Errorf("%w: invalid model price", ErrInvalidAutomation)
		}
		if _, duplicate := normalized[alias]; duplicate {
			return fmt.Errorf("%w: duplicate model price alias", ErrInvalidAutomation)
		}
		normalized[alias] = price
	}
	automation.ModelPrices = normalized
	return nil
}

func normalizeModelStats(automation *RepositoryReviewAutomation) error {
	if len(automation.ModelStats) > maxAutomationReviewers {
		return fmt.Errorf("%w: too many model statistics", ErrInvalidAutomation)
	}
	selected := automationReviewerSet(automation.ReviewerModels)
	normalized := make(map[string]RepositoryReviewModelStats, len(automation.ModelStats))
	for rawAlias, stats := range automation.ModelStats {
		alias := strings.TrimSpace(rawAlias)
		if _, exists := selected[alias]; !exists || !validBoundedText(alias, 256) {
			return fmt.Errorf("%w: invalid model statistics alias", ErrInvalidAutomation)
		}
		if stats.Tokens.TotalTokens == 0 && stats.Tokens.PromptTokens >= 0 && stats.Tokens.CompletionTokens >= 0 {
			stats.Tokens.TotalTokens = stats.Tokens.PromptTokens + stats.Tokens.CompletionTokens
		}
		if err := validateTokenUsage(stats.Tokens); err != nil {
			return err
		}
		if !finiteNonnegative(stats.EstimatedCostUSD, maxAutomationEstimatedCost) ||
			stats.Requests < 0 || stats.Failures < 0 || stats.Failures > stats.Requests ||
			stats.Findings < 0 || stats.Findings > maxReviewObservations ||
			stats.ReviewedFiles < 0 || stats.ReviewedFiles > maxReviewFiles ||
			stats.LatencyMillis < 0 {
			return fmt.Errorf("%w: invalid model statistics", ErrInvalidAutomation)
		}
		if _, duplicate := normalized[alias]; duplicate {
			return fmt.Errorf("%w: duplicate model statistics alias", ErrInvalidAutomation)
		}
		normalized[alias] = stats
	}
	automation.ModelStats = normalized
	return nil
}

func normalizeModelCoverageSketches(automation *RepositoryReviewAutomation) error {
	selected := automationReviewerSet(automation.ReviewerModels)
	normalized := make(map[string]string, len(automation.ModelCoverageSketches))
	for rawAlias, encoded := range automation.ModelCoverageSketches {
		alias := strings.TrimSpace(rawAlias)
		if _, exists := selected[alias]; !exists || !validBoundedText(alias, 256) {
			return fmt.Errorf("%w: invalid model coverage alias", ErrInvalidAutomation)
		}
		raw, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(encoded))
		if err != nil || len(raw) != automationModelCoverageSketchBytes {
			return fmt.Errorf("%w: invalid model coverage sketch", ErrInvalidAutomation)
		}
		normalized[alias] = base64.RawStdEncoding.EncodeToString(raw)
	}
	automation.ModelCoverageSketches = normalized
	return nil
}

func normalizeAccountSnapshots(automation *RepositoryReviewAutomation) error {
	if len(automation.AccountLimitSnapshots) > maxAutomationAccountSnapshots {
		return fmt.Errorf("%w: too many account-limit snapshots", ErrInvalidAutomation)
	}
	seen := make(map[string]struct{}, len(automation.AccountLimitSnapshots))
	for index := range automation.AccountLimitSnapshots {
		snapshot := &automation.AccountLimitSnapshots[index]
		snapshot.AccountID = strings.TrimSpace(snapshot.AccountID)
		snapshot.Name = strings.TrimSpace(snapshot.Name)
		snapshot.Window = strings.ToLower(strings.TrimSpace(snapshot.Window))
		snapshot.Detail = strings.TrimSpace(snapshot.Detail)
		snapshot.ResetsAt = snapshot.ResetsAt.UTC()
		snapshot.CheckedAt = snapshot.CheckedAt.UTC()
		if !validBoundedText(snapshot.AccountID, 1024) ||
			!validOptionalAutomationText(snapshot.Name, 256) ||
			!validBoundedText(snapshot.Window, 64) ||
			!validOptionalAutomationText(snapshot.Detail, 1024) || snapshot.CheckedAt.IsZero() {
			return fmt.Errorf("%w: invalid account-limit snapshot", ErrInvalidAutomation)
		}
		if snapshot.RemainingPercent != nil && !validPercent(*snapshot.RemainingPercent) {
			return fmt.Errorf("%w: invalid remaining percentage", ErrInvalidAutomation)
		}
		key := snapshot.AccountID + "\x00" + snapshot.Name + "\x00" + snapshot.Window
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("%w: duplicate account-limit snapshot", ErrInvalidAutomation)
		}
		seen[key] = struct{}{}
	}
	sort.Slice(automation.AccountLimitSnapshots, func(i, j int) bool {
		left, right := automation.AccountLimitSnapshots[i], automation.AccountLimitSnapshots[j]
		if left.AccountID == right.AccountID && left.Name == right.Name {
			return left.Window < right.Window
		}
		if left.AccountID == right.AccountID {
			return left.Name < right.Name
		}
		return left.AccountID < right.AccountID
	})
	return nil
}

func normalizeUniqueAutomationStrings(values []string, maximum, maxBytes int, field string) ([]string, error) {
	if len(values) > maximum {
		return nil, fmt.Errorf("%w: too many %ss", ErrInvalidAutomation, field)
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if !validBoundedText(value, maxBytes) {
			return nil, fmt.Errorf("%w: invalid %s", ErrInvalidAutomation, field)
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, fmt.Errorf("%w: duplicate %s", ErrInvalidAutomation, field)
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out, nil
}

func cloneAutomation(automation RepositoryReviewAutomation) RepositoryReviewAutomation {
	automation.ReviewerModels = append([]string{}, automation.ReviewerModels...)
	automation.ScopePolicy.CodeTypes = append(
		[]RepositoryReviewCodeType{}, automation.ScopePolicy.CodeTypes...,
	)
	automation.ScopePolicy.IncludeFolders = append(
		[]string{}, automation.ScopePolicy.IncludeFolders...,
	)
	automation.ScopePolicy.ExcludeFolders = append(
		[]string{}, automation.ScopePolicy.ExcludeFolders...,
	)
	automation.ScopePlan.Warnings = append([]string{}, automation.ScopePlan.Warnings...)
	if automation.ScopeSelection != nil {
		selection := *automation.ScopeSelection
		selection.IncludePrefixes = append([]string{}, selection.IncludePrefixes...)
		selection.ExcludePrefixes = append([]string{}, selection.ExcludePrefixes...)
		selection.CandidateIDs = append([]string{}, selection.CandidateIDs...)
		selection.HotpathCandidateIDs = append([]string{}, selection.HotpathCandidateIDs...)
		automation.ScopeSelection = &selection
	}
	automation.RunIDs = append([]string{}, automation.RunIDs...)
	if automation.ModelPrices != nil {
		prices := make(map[string]RepositoryReviewModelPrice, len(automation.ModelPrices))
		for alias, price := range automation.ModelPrices {
			prices[alias] = price
		}
		automation.ModelPrices = prices
	}
	if automation.ModelStats != nil {
		stats := make(map[string]RepositoryReviewModelStats, len(automation.ModelStats))
		for alias, value := range automation.ModelStats {
			stats[alias] = value
		}
		automation.ModelStats = stats
	}
	if automation.ModelCoverageSketches != nil {
		sketches := make(map[string]string, len(automation.ModelCoverageSketches))
		for alias, value := range automation.ModelCoverageSketches {
			sketches[alias] = value
		}
		automation.ModelCoverageSketches = sketches
	}
	automation.AccountLimitSnapshots = append(
		[]RepositoryReviewAccountLimitSnapshot(nil), automation.AccountLimitSnapshots...,
	)
	for index := range automation.AccountLimitSnapshots {
		if remaining := automation.AccountLimitSnapshots[index].RemainingPercent; remaining != nil {
			remainingPercent := *remaining
			automation.AccountLimitSnapshots[index].RemainingPercent = &remainingPercent
		}
	}
	return automation
}

func automationReviewerSet(reviewers []string) map[string]struct{} {
	set := make(map[string]struct{}, len(reviewers))
	for _, reviewer := range reviewers {
		set[reviewer] = struct{}{}
	}
	return set
}

func containsAutomationString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsAutomationMapKey[V any](values map[string]V, key string) bool {
	_, exists := values[key]
	return exists
}

func validAutomationStatus(status RepositoryReviewAutomationStatus) bool {
	switch status {
	case RepositoryReviewAutomationIdle, RepositoryReviewAutomationRunning,
		RepositoryReviewAutomationStopping, RepositoryReviewAutomationPaused,
		RepositoryReviewAutomationCompleted, RepositoryReviewAutomationFailed:
		return true
	default:
		return false
	}
}

func validAutomationPauseReason(reason RepositoryReviewPauseReason) bool {
	switch reason {
	case "", RepositoryReviewPauseManual, RepositoryReviewPauseTokenBudget,
		RepositoryReviewPauseCostBudget, RepositoryReviewPauseAccountLimit,
		RepositoryReviewPauseGuardExpression, RepositoryReviewPauseNoProgress,
		RepositoryReviewPauseRunFailed,
		RepositoryReviewPauseServiceRestart:
		return true
	default:
		return false
	}
}

func finiteNonnegative(value, maximum float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= maximum
}

func validPercent(value float64) bool {
	return finiteNonnegative(value, 100)
}

func validOptionalAutomationText(value string, maximum int) bool {
	return value == "" || validBoundedText(value, maximum)
}
