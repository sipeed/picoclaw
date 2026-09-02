package repoaudit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/fileutil"
)

const storeDirectory = "repository_reviews"

const (
	maxRepositoryIdentityBytes       = 4096
	maxReviewFiles                   = 100_000
	maxReviewObservations            = 100_000
	maxFindingsPerObservation        = 256
	maxFindingTextBytes              = 64 << 10
	maxMatchHintItems                = 32
	maxMatchHintIdentityBytes        = 4096
	maxFixEffortLOC                  = 1_000_000
	maxIssueDraftBodyBytes           = 60 << 10
	maxReviewFileMetadataBytes       = 16 << 20
	maxStateFileBytes          int64 = 64 << 20
)

var (
	ErrConflict    = errors.New("repository review state changed")
	ErrInvalidPlan = errors.New("invalid repository review plan")
	storeLocks     sync.Map
)

type Store struct {
	root        string
	now         func() time.Time
	loadForTest func(string) (RepositoryState, error)
}

func NewStore(workspace string) Store {
	return Store{root: filepath.Join(workspace, storeDirectory), now: time.Now}
}

func (s Store) Plan(
	ctx context.Context,
	repository, commitSHA, inventoryHash string,
	files []FileRef,
	force bool,
) (Plan, error) {
	return s.PlanWithProfile(ctx, repository, commitSHA, inventoryHash, "repository-bug-finder-v1", files, force)
}

func (s Store) PlanWithProfile(
	ctx context.Context,
	repository, commitSHA, inventoryHash, profileHash string,
	files []FileRef,
	force bool,
) (Plan, error) {
	return s.PlanWithProfileLimit(
		ctx, repository, commitSHA, inventoryHash, profileHash, files, force, maxReviewFiles,
	)
}

func (s Store) PlanWithProfileLimit(
	ctx context.Context,
	repository, commitSHA, inventoryHash, profileHash string,
	files []FileRef,
	force bool,
	maximumPending int,
) (Plan, error) {
	return s.PlanWithProfileLimitAuthoritative(
		ctx, repository, commitSHA, inventoryHash, profileHash,
		files, force, maximumPending, false,
	)
}

func (s Store) PlanWithProfileLimitAuthoritative(
	ctx context.Context,
	repository, commitSHA, inventoryHash, profileHash string,
	files []FileRef,
	force bool,
	maximumPending int,
	authoritative bool,
) (Plan, error) {
	return s.planWithProfileLimitAuthoritative(
		ctx, repository, commitSHA, inventoryHash, profileHash, "", 0, nil,
		files, force, maximumPending, authoritative,
	)
}

// PlanWithProfileLimitAuthoritativeForCampaign plans work only for a campaign
// previously installed through BeginCampaign. It may bind that campaign's
// remaining immutable scope metadata, but it cannot create or replace campaign
// authority.
func (s Store) PlanWithProfileLimitAuthoritativeForCampaign(
	ctx context.Context,
	repository, commitSHA, inventoryHash, profileHash, campaignID string,
	requiredAssignments int,
	files []FileRef,
	force bool,
	maximumPending int,
	authoritative bool,
) (Plan, error) {
	if requiredAssignments < 1 || requiredAssignments > maxRepositoryReviewRequiredAssignments {
		return Plan{}, ErrInvalidPlan
	}
	return s.planWithProfileLimitAuthoritative(
		ctx, repository, commitSHA, inventoryHash, profileHash, campaignID, requiredAssignments, nil,
		files, force, maximumPending, authoritative,
	)
}

// PlanAssignmentsForCampaign selects distinct incomplete files and freezes one
// missing-only scope for every assignment in catalog. The catalog is part of
// campaign identity and cannot drift after its first successful binding.
func (s Store) PlanAssignmentsForCampaign(
	ctx context.Context,
	repository, commitSHA, inventoryHash, profileHash, campaignID string,
	catalog []RepositoryReviewAssignment,
	files []FileRef,
	force bool,
	maximumPending int,
	authoritative bool,
) (Plan, error) {
	normalized, err := NormalizeRepositoryReviewAssignmentCatalog(catalog)
	if err != nil {
		return Plan{}, err
	}
	return s.planWithProfileLimitAuthoritative(
		ctx,
		repository,
		commitSHA,
		inventoryHash,
		profileHash,
		campaignID,
		repositoryReviewRequiredAssignmentCount(normalized),
		normalized,
		files,
		force,
		maximumPending,
		authoritative,
	)
}

func (s Store) planWithProfileLimitAuthoritative(
	ctx context.Context,
	repository, commitSHA, inventoryHash, profileHash, campaignID string,
	requiredAssignments int,
	assignmentCatalog []RepositoryReviewAssignment,
	files []FileRef,
	force bool,
	maximumPending int,
	authoritative bool,
) (Plan, error) {
	if err := ctx.Err(); err != nil {
		return Plan{}, err
	}
	repository = strings.TrimSpace(repository)
	commitSHA = strings.TrimSpace(commitSHA)
	inventoryHash = strings.TrimSpace(inventoryHash)
	profileHash = strings.TrimSpace(profileHash)
	campaignID = strings.TrimSpace(campaignID)
	if campaignID != "" {
		commitSHA = strings.ToLower(commitSHA)
	}
	if !validBoundedText(repository, maxRepositoryIdentityBytes) ||
		!validBoundedText(commitSHA, 256) || !validBoundedText(inventoryHash, 256) ||
		!validBoundedText(profileHash, 256) ||
		(campaignID != "" && (!ValidRepositoryReviewCampaignID(campaignID) || !authoritative)) {
		return Plan{}, fmt.Errorf("%w: repository, commit SHA, and inventory hash are required", ErrInvalidPlan)
	}
	files, err := normalizeFiles(files)
	if err != nil {
		return Plan{}, err
	}
	if len(files) > maxReviewFiles {
		return Plan{}, fmt.Errorf("%w: too many review files", ErrInvalidPlan)
	}
	if maximumPending < 1 || maximumPending > maxReviewFiles {
		return Plan{}, fmt.Errorf("%w: invalid pending-file limit", ErrInvalidPlan)
	}
	unlock, err := s.lock(repository)
	if err != nil {
		return Plan{}, err
	}
	defer unlock()
	state, err := s.load(repository)
	if err != nil {
		return Plan{}, err
	}
	now := s.clock()
	campaignChanged := false
	if campaignID != "" {
		scopeDigest, _ := repositoryReviewCampaignScopeDigestForFiles(files)
		if len(assignmentCatalog) > 0 {
			campaignChanged, err = bindRepositoryReviewCampaignAssignmentCatalog(
				&state, campaignID, commitSHA, inventoryHash, profileHash, scopeDigest,
				assignmentCatalog, len(files),
			)
		} else {
			campaignChanged, err = bindRepositoryReviewCampaignScope(
				&state, campaignID, commitSHA, inventoryHash, profileHash, scopeDigest,
				requiredAssignments, len(files),
			)
		}
		if err != nil {
			return Plan{}, err
		}
	}
	forceCampaignID := ""
	if force {
		if state.ActiveForceCampaignID != "" &&
			state.ActiveForceProfileHash == profileHash &&
			state.ActiveForceCommitSHA == commitSHA {
			forceCampaignID = state.ActiveForceCampaignID
		} else {
			forceCampaignID = stableID(
				"rfc_", repository, commitSHA, profileHash,
				fmt.Sprint(state.ReviewVersion), fmt.Sprint(now.UnixNano()),
			)
		}
	}
	candidates := make([]FileRef, 0, len(files))
	unchanged := make([]FileRef, 0, len(files))
	planUnsupported := make([]UnsupportedFile, 0)
	previouslyReviewed := 0
	for _, file := range files {
		if campaignID != "" && len(assignmentCatalog) > 0 &&
			state.CurrentCampaign.Paths[file.Path].Unsupported {
			unsupported := state.Unsupported[file.Path]
			unsupported.FileRef = file
			unsupported.CommitSHA = commitSHA
			unsupported.ProfileHash = profileHash
			if strings.TrimSpace(unsupported.Reason) == "" {
				unsupported.Reason = "campaign_terminal"
			}
			planUnsupported = append(planUnsupported, unsupported)
			continue
		}
		if unsupported, exists := state.Unsupported[file.Path]; exists &&
			unsupported.BlobSHA == file.BlobSHA && unsupported.SizeBytes == file.SizeBytes &&
			unsupported.Mode == file.Mode && unsupported.ProfileHash == profileHash &&
			(!force || unsupported.ForceCampaignID == forceCampaignID) {
			// Classification metadata is inventory-owned. Preserve the durable
			// terminal reason/provenance while rebinding the exact current FileRef.
			unsupported.FileRef = file
			planUnsupported = append(planUnsupported, unsupported)
			continue
		}
		previous, reviewed := state.Files[file.Path]
		if reviewed {
			previouslyReviewed++
		}
		matchesBase := reviewed && previous.BlobSHA == file.BlobSHA &&
			previous.SizeBytes == file.SizeBytes && previous.Mode == file.Mode &&
			previous.ProfileHash == profileHash
		campaignComplete := false
		if campaignID != "" && len(assignmentCatalog) > 0 {
			if pathCoverage, exists := state.CurrentCampaign.Paths[file.Path]; exists &&
				!pathCoverage.Unsupported {
				projected, projectionErr := projectRepositoryReviewAssignmentCoverage(
					pathCoverage, assignmentCatalog,
				)
				if projectionErr != nil {
					return Plan{}, projectionErr
				}
				campaignComplete = projected.Completed
			}
		}
		if campaignID != "" && len(assignmentCatalog) > 0 && campaignComplete ||
			(campaignID == "" || len(assignmentCatalog) == 0) &&
				matchesBase && (!force || previous.ForceCampaignID == forceCampaignID) {
			unchanged = append(unchanged, file)
			continue
		}
		candidates = append(candidates, file)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left := reviewAttemptsFor(state, candidates[i], profileHash)
		right := reviewAttemptsFor(state, candidates[j], profileHash)
		if left != right {
			return left < right
		}
		return candidates[i].Path < candidates[j].Path
	})
	pendingEnd := min(maximumPending, len(candidates))
	pending := append([]FileRef(nil), candidates[:pendingEnd]...)
	deferred := append([]FileRef(nil), candidates[pendingEnd:]...)
	plan := Plan{
		CampaignID: campaignID,
		Repository: repository, CommitSHA: commitSHA, InventoryHash: inventoryHash,
		ProfileHash: profileHash, RequiredAssignments: requiredAssignments,
		AssignmentCatalog: append([]RepositoryReviewAssignment(nil), assignmentCatalog...),
		ForceCampaignID:   forceCampaignID, Authoritative: authoritative,
		TargetIsDefault: true,
		StateVersion:    state.ReviewVersion, PendingFiles: pending, DeferredFiles: deferred,
		UnchangedFiles:     unchanged,
		UnsupportedFiles:   planUnsupported,
		PreviouslyReviewed: previouslyReviewed, CreatedAt: now,
	}
	if len(assignmentCatalog) > 0 {
		plan.AssignmentPlans = make([]RepositoryReviewAssignmentPlan, 0, len(assignmentCatalog))
		for _, assignment := range assignmentCatalog {
			missing := make([]FileRef, 0, len(pending))
			for _, file := range pending {
				complete, assignmentErr := repositoryReviewAssignmentComplete(
					state.CurrentCampaign.Paths[file.Path], assignmentCatalog, assignment.ID,
				)
				if assignmentErr != nil {
					return Plan{}, assignmentErr
				}
				if !complete {
					missing = append(missing, file)
				}
			}
			if len(missing) == 0 {
				continue
			}
			reviewerModel := assignment.Reviewer
			if reviewerModel == "default" {
				reviewerModel = ""
			}
			plan.AssignmentPlans = append(plan.AssignmentPlans, RepositoryReviewAssignmentPlan{
				AssignmentID: assignment.ID,
				FocusID:      assignment.FocusID,
				Label:        assignment.FocusID,
				Reviewer:     reviewerModel,
				Optional:     !assignment.Required,
				Files:        missing,
			})
		}
	}
	if campaignID != "" {
		for _, file := range unchanged {
			changed, coverageErr := mergeRepositoryReviewCampaignPath(
				state.CurrentCampaign, file.Path,
				RepositoryReviewCampaignPathCoverage{Completed: true},
			)
			if coverageErr != nil {
				return Plan{}, coverageErr
			}
			campaignChanged = campaignChanged || changed
		}
		for _, unsupported := range planUnsupported {
			changed, coverageErr := mergeRepositoryReviewCampaignPath(
				state.CurrentCampaign, unsupported.Path,
				RepositoryReviewCampaignPathCoverage{Unsupported: true},
			)
			if coverageErr != nil {
				return Plan{}, coverageErr
			}
			campaignChanged = campaignChanged || changed
		}
		if campaignChanged {
			state.Version++
			state.ReviewVersion++
			state.UpdatedAt = now
			if err := s.save(&state); err != nil {
				return Plan{}, err
			}
			plan.StateVersion = state.ReviewVersion
		}
	}
	plan.ID = planDigest(plan)
	return plan, nil
}

// BindPlanBranch adds resolved branch provenance before a plan is dispatched.
// The returned plan receives a new digest so Record can continue to verify the
// entire immutable plan envelope.
func BindPlanBranch(
	plan Plan,
	targetBranch string,
	advertisedDefaultBranch string,
	targetIsDefault bool,
) (Plan, error) {
	targetBranch = strings.TrimSpace(targetBranch)
	advertisedDefaultBranch = strings.TrimSpace(advertisedDefaultBranch)
	if (targetBranch != "" && !validBoundedText(targetBranch, maxRepositoryReviewBranchBytes)) ||
		(advertisedDefaultBranch != "" &&
			!validBoundedText(advertisedDefaultBranch, maxRepositoryReviewBranchBytes)) {
		return Plan{}, ErrInvalidPlan
	}
	plan.ID = ""
	plan.TargetBranch = targetBranch
	plan.AdvertisedDefaultBranch = advertisedDefaultBranch
	plan.TargetIsDefault = targetIsDefault
	plan.ID = planDigest(plan)
	return plan, nil
}

func (s Store) Record(ctx context.Context, request RecordRequest) (RecordResult, error) {
	if err := ctx.Err(); err != nil {
		return RecordResult{}, err
	}
	request.RunID = strings.TrimSpace(request.RunID)
	if request.RunID == "" || request.Plan.ID == "" || request.Plan.ID != planDigest(request.Plan) {
		return RecordResult{}, ErrInvalidPlan
	}
	rawCampaignID := request.Plan.CampaignID
	request.Plan.CampaignID = strings.TrimSpace(rawCampaignID)
	if rawCampaignID != request.Plan.CampaignID || request.Plan.CampaignID != "" &&
		!ValidRepositoryReviewCampaignID(request.Plan.CampaignID) {
		return RecordResult{}, ErrInvalidPlan
	}
	if request.Plan.ForceCampaignID != "" &&
		!validBoundedText(request.Plan.ForceCampaignID, 256) {
		return RecordResult{}, ErrInvalidPlan
	}
	if !validBoundedText(request.RunID, 1024) || len(request.Observations) > maxReviewObservations {
		return RecordResult{}, ErrInvalidPlan
	}
	if request.ExcludedFiles < 0 || request.ExcludedFiles > maxReviewFiles {
		return RecordResult{}, ErrInvalidPlan
	}
	if err := normalizeRecordBranchProvenance(&request); err != nil {
		return RecordResult{}, err
	}
	campaignSelectedFiles := 0
	if request.Plan.CampaignID != "" {
		selectedFiles, campaignErr := validateRepositoryReviewCampaignPlan(request.Plan)
		if campaignErr != nil || request.InspectedFiles == nil {
			return RecordResult{}, ErrInvalidPlan
		}
		campaignSelectedFiles = selectedFiles
	}
	files, err := normalizeFiles(request.Plan.PendingFiles)
	if err != nil || len(files) != len(request.Plan.PendingFiles) {
		return RecordResult{}, ErrInvalidPlan
	}
	deferred, err := normalizeFiles(request.Plan.DeferredFiles)
	if err != nil || len(deferred) != len(request.Plan.DeferredFiles) {
		return RecordResult{}, ErrInvalidPlan
	}
	paths := make(map[string]struct{}, len(files)+len(deferred))
	for _, file := range append(append([]FileRef(nil), files...), deferred...) {
		if _, duplicate := paths[file.Path]; duplicate {
			return RecordResult{}, ErrInvalidPlan
		}
		paths[file.Path] = struct{}{}
	}
	request.Plan.PendingFiles = files
	request.Plan.DeferredFiles = deferred
	allowed := make(map[string]FileRef, len(files))
	for _, file := range files {
		allowed[file.Path] = file
	}
	var inspectedFiles, completedFiles []FileRef
	campaignScopeDigest := ""
	inspectedPaths := make(map[string]struct{})
	if request.Plan.CampaignID != "" {
		if request.ReviewEvidence == nil {
			return RecordResult{}, fmt.Errorf("%w: campaign review evidence is required", ErrInvalidPlan)
		}
		campaignScopeDigest, _ = repositoryReviewCampaignScopeDigestForPlan(request.Plan)
		unsupportedEvidencePaths := make(map[string]struct{}, len(request.UnsupportedFiles))
		for _, unsupported := range request.UnsupportedFiles {
			bound, ok := allowed[unsupported.Path]
			if !ok || bound != unsupported.FileRef ||
				!validBoundedText(strings.TrimSpace(unsupported.Reason), 256) ||
				unsupported.Reason != strings.TrimSpace(unsupported.Reason) {
				return RecordResult{}, ErrInvalidPlan
			}
			if _, duplicate := unsupportedEvidencePaths[unsupported.Path]; duplicate {
				return RecordResult{}, ErrInvalidPlan
			}
			unsupportedEvidencePaths[unsupported.Path] = struct{}{}
		}
		derivedObservations, derivedInspected, derivedCompleted, evidenceErr := deriveRepositoryReviewCampaignEvidence(
			request.ReviewEvidence, allowed, request.Plan.RequiredAssignments,
			unsupportedEvidencePaths,
		)
		if evidenceErr != nil {
			return RecordResult{}, evidenceErr
		}
		inspectedFiles, err = bindRepositoryReviewCampaignFiles(request.InspectedFiles, allowed)
		if err != nil {
			return RecordResult{}, fmt.Errorf("inspected review files: %w", err)
		}
		completedFiles, err = bindRepositoryReviewCampaignFiles(request.CompletedFiles, allowed)
		if err != nil {
			return RecordResult{}, fmt.Errorf("completed review files: %w", err)
		}
		if !reflect.DeepEqual(inspectedFiles, derivedInspected) ||
			!reflect.DeepEqual(completedFiles, derivedCompleted) {
			return RecordResult{}, fmt.Errorf(
				"%w: campaign review projections do not match child evidence", ErrInvalidPlan,
			)
		}
		request.Observations = derivedObservations
		for _, file := range inspectedFiles {
			inspectedPaths[file.Path] = struct{}{}
		}
		request.InspectedFiles = inspectedFiles
		request.CompletedFiles = completedFiles
	}
	unlock, err := s.lock(request.Plan.Repository)
	if err != nil {
		return RecordResult{}, err
	}
	defer unlock()
	state, err := s.load(request.Plan.Repository)
	if err != nil {
		return RecordResult{}, err
	}
	if previous, ok := replayedRun(state, request); ok {
		return RecordResult{
			State: state, Run: previous,
			AcceptedFindingIDs: append([]string(nil), previous.FindingIDs...),
		}, nil
	}
	if state.ReviewVersion != request.Plan.StateVersion {
		return RecordResult{}, ErrConflict
	}
	if request.Plan.CampaignID != "" {
		if _, bindErr := bindRepositoryReviewCampaignScope(
			&state,
			request.Plan.CampaignID,
			strings.ToLower(strings.TrimSpace(request.Plan.CommitSHA)),
			strings.TrimSpace(request.Plan.InventoryHash),
			strings.TrimSpace(request.Plan.ProfileHash),
			campaignScopeDigest,
			request.Plan.RequiredAssignments,
			campaignSelectedFiles,
		); bindErr != nil {
			return RecordResult{}, bindErr
		}
	}
	completedAt := request.CompletedAt.UTC()
	if completedAt.IsZero() {
		completedAt = s.clock()
	}
	unsupportedFiles := make(map[string]UnsupportedFile, len(request.UnsupportedFiles))
	for _, unsupported := range request.UnsupportedFiles {
		bound, ok := allowed[unsupported.Path]
		unsupported.Reason = strings.TrimSpace(unsupported.Reason)
		if !ok || bound.BlobSHA != unsupported.BlobSHA || bound.SizeBytes != unsupported.SizeBytes ||
			bound.Mode != unsupported.Mode || !validBoundedText(unsupported.Reason, 256) {
			return RecordResult{}, ErrInvalidPlan
		}
		unsupported.FileRef = bound
		if request.Plan.CampaignID != "" {
			if _, inspected := inspectedPaths[unsupported.Path]; inspected ||
				containsRepositoryReviewFile(completedFiles, unsupported.Path) {
				return RecordResult{}, fmt.Errorf(
					"%w: unsupported file %q overlaps reviewed evidence", ErrInvalidPlan, unsupported.Path,
				)
			}
		}
		unsupported.CommitSHA = request.Plan.CommitSHA
		unsupported.ProfileHash = request.Plan.ProfileHash
		unsupported.ForceCampaignID = request.Plan.ForceCampaignID
		unsupported.UpdatedAt = completedAt
		unsupportedFiles[unsupported.Path] = unsupported
	}
	contexts := make([]FindingContext, 0, len(request.Observations))
	existingContexts := make(map[string]int, len(state.Contexts))
	for index, contextRecord := range state.Contexts {
		existingContexts[contextRecord.ID] = index
	}
	covered := make(map[string]FileRef, len(files))
	var acceptedIDs []string
	rejected := 0
	models := make([]string, 0)
	for observationIndex, observation := range request.Observations {
		observation.Model = strings.TrimSpace(observation.Model)
		observation.ModelAlias = strings.TrimSpace(observation.ModelAlias)
		observation.Account = strings.TrimSpace(observation.Account)
		missingExactProvenance := request.Plan.CampaignID != "" &&
			(observation.ModelAlias == "" || observation.Account == "")
		if !validFindingSourceProvenance(
			observation.Model, observation.ModelAlias, observation.Account,
		) || missingExactProvenance ||
			len(observation.Findings) > maxFindingsPerObservation {
			return RecordResult{}, fmt.Errorf("observation %d has invalid model provenance", observationIndex)
		}
		var scope []FileRef
		var scopeErr error
		if request.Plan.CampaignID != "" {
			scope, scopeErr = bindRepositoryReviewCampaignFiles(observation.ScopeFiles, allowed)
		} else {
			scope, scopeErr = bindScopeFiles(observation.ScopeFiles, allowed)
		}
		if scopeErr != nil {
			return RecordResult{}, fmt.Errorf("observation %d: %w", observationIndex, scopeErr)
		}
		contextRecord := FindingContext{
			CampaignID: request.Plan.CampaignID,
			Repository: request.Plan.Repository, CommitSHA: request.Plan.CommitSHA,
			InventoryHash: request.Plan.InventoryHash, ProfileHash: request.Plan.ProfileHash,
			RunID: request.RunID,
			Model: observation.Model, ModelAlias: observation.ModelAlias, Account: observation.Account,
			Reviewer: strings.TrimSpace(observation.Reviewer),
			Files:    scope, RawDigest: strings.TrimSpace(observation.RawDigest), CreatedAt: completedAt,
		}
		contextRecord.ID = stableID("rctx_", contextBindingDigest(contextRecord))
		contextUsed := false
		if request.Plan.CampaignID == "" && request.CompletedFiles == nil {
			for _, file := range scope {
				covered[file.Path] = file
			}
		}
		contributorModel := observation.Model
		if observation.ModelAlias != "" {
			contributorModel = observation.ModelAlias
		}
		models = appendUnique(models, contributorModel)
		for findingIndex, candidate := range observation.Findings {
			candidate = normalizeCandidate(candidate)
			if candidate.Validation.Status != "confirmed" {
				rejected++
				continue
			}
			primary, ok := fileInScope(candidate.File, scope)
			if !ok {
				return RecordResult{}, fmt.Errorf(
					"observation %d finding %d references a file outside its exact context",
					observationIndex,
					findingIndex,
				)
			}
			if err := validateCandidate(candidate); err != nil {
				rejected++
				continue
			}
			deduplicatedID, persistErr := persistLegacyRecordFinding(
				&state, request.Plan, request.RunID, observationIndex, findingIndex,
				contextRecord, observation, primary, candidate, completedAt,
			)
			if persistErr != nil {
				return RecordResult{}, persistErr
			}
			projectionIndex := findingIndexByID(state.Findings, deduplicatedID)
			if projectionIndex < 0 {
				return RecordResult{}, ErrConflict
			}
			finding := &state.Findings[projectionIndex]
			finding.Models = []string{contributorModel}
			finding.Observations = []FindingObservation{findingObservationFrom(
				candidate, contextRecord.ID, observation.Model, observation.ModelAlias,
				observation.Account, observation.Reviewer,
			)}
			acceptedIDs = appendUnique(acceptedIDs, deduplicatedID)
			contextUsed = true
			continue
		}
		if contextUsed {
			if existingIndex, exists := existingContexts[contextRecord.ID]; exists {
				if existingIndex < len(state.Contexts) {
					state.Contexts[existingIndex] = contextRecord
				} else {
					contexts[existingIndex-len(state.Contexts)] = contextRecord
				}
				continue
			}
			existingContexts[contextRecord.ID] = len(state.Contexts) + len(contexts)
			contexts = append(contexts, contextRecord)
		}
	}
	if request.Plan.CampaignID != "" {
		for _, file := range completedFiles {
			covered[file.Path] = file
		}
	} else if request.CompletedFiles != nil {
		completed, completedErr := bindScopeFiles(request.CompletedFiles, allowed)
		if completedErr != nil && len(request.CompletedFiles) > 0 {
			return RecordResult{}, fmt.Errorf("completed review files: %w", completedErr)
		}
		for _, file := range completed {
			covered[file.Path] = file
		}
	}
	state.Contexts = append(state.Contexts, contexts...)
	pruneUnreferencedFindingContexts(&state)
	reconcileFindingsProcessingCounters(&state)
	if len(state.RawFindings) > 0 {
		state.FindingsProcessing.UpdatedAt = completedAt
	}
	var unreviewedPaths []string
	for _, file := range files {
		if _, complete := covered[file.Path]; complete {
			delete(state.ReviewAttempts, file.Path)
			delete(state.ReviewAttemptIdentities, file.Path)
			delete(state.Unsupported, file.Path)
			continue
		}
		if unsupported, terminal := unsupportedFiles[file.Path]; terminal {
			state.Unsupported[file.Path] = unsupported
			delete(state.ReviewAttempts, file.Path)
			delete(state.ReviewAttemptIdentities, file.Path)
			continue
		}
		identity := reviewAttemptIdentity(file, request.Plan.ProfileHash)
		if state.ReviewAttemptIdentities[file.Path] != identity {
			state.ReviewAttempts[file.Path] = 0
		}
		state.ReviewAttemptIdentities[file.Path] = identity
		state.ReviewAttempts[file.Path]++
		unreviewedPaths = append(unreviewedPaths, file.Path)
	}
	for _, file := range covered {
		state.Files[file.Path] = ReviewedFile{
			FileRef: file, CommitSHA: request.Plan.CommitSHA,
			ProfileHash: request.Plan.ProfileHash, ForceCampaignID: request.Plan.ForceCampaignID,
			RunID: request.RunID, ReviewedAt: completedAt,
		}
	}
	if request.Plan.CampaignID != "" {
		for _, file := range request.Plan.UnchangedFiles {
			if _, mergeErr := mergeRepositoryReviewCampaignPath(
				state.CurrentCampaign, file.Path,
				RepositoryReviewCampaignPathCoverage{Completed: true},
			); mergeErr != nil {
				return RecordResult{}, mergeErr
			}
		}
		for _, unsupported := range request.Plan.UnsupportedFiles {
			if _, mergeErr := mergeRepositoryReviewCampaignPath(
				state.CurrentCampaign, unsupported.Path,
				RepositoryReviewCampaignPathCoverage{Unsupported: true},
			); mergeErr != nil {
				return RecordResult{}, mergeErr
			}
		}
		for _, file := range inspectedFiles {
			if _, mergeErr := mergeRepositoryReviewCampaignPath(
				state.CurrentCampaign, file.Path,
				RepositoryReviewCampaignPathCoverage{Inspected: true},
			); mergeErr != nil {
				return RecordResult{}, mergeErr
			}
		}
		for _, file := range completedFiles {
			// Every completed file was already merged as inspected above, so the
			// monotonic completion promotion cannot reclassify a terminal path.
			_, _ = mergeRepositoryReviewCampaignPath(
				state.CurrentCampaign, file.Path,
				RepositoryReviewCampaignPathCoverage{Inspected: true, Completed: true},
			)
		}
		for _, unsupported := range unsupportedFiles {
			if _, mergeErr := mergeRepositoryReviewCampaignPath(
				state.CurrentCampaign, unsupported.Path,
				RepositoryReviewCampaignPathCoverage{Unsupported: true},
			); mergeErr != nil {
				return RecordResult{}, mergeErr
			}
		}
	}
	var unsupportedPaths []string
	for pathValue := range unsupportedFiles {
		unsupportedPaths = append(unsupportedPaths, pathValue)
	}
	sort.Strings(unsupportedPaths)
	run := ReviewRun{
		ID: request.RunID, CampaignID: request.Plan.CampaignID,
		PlanID: request.Plan.ID, CommitSHA: request.Plan.CommitSHA,
		InventoryHash: request.Plan.InventoryHash, ProfileHash: request.Plan.ProfileHash,
		ScopeDigest: campaignScopeDigest, ReviewedFiles: len(covered),
		InspectedFiles:   len(inspectedFiles),
		UnreviewedFiles:  len(files) - len(covered) - len(unsupportedFiles),
		UnsupportedCount: len(unsupportedFiles),
		RemainingFiles:   len(request.Plan.DeferredFiles) + len(files) - len(covered) - len(unsupportedFiles),
		UnreviewedPaths:  unreviewedPaths,
		UnsupportedPaths: unsupportedPaths,
		SkippedFiles:     len(request.Plan.UnchangedFiles), AcceptedFindings: len(acceptedIDs),
		ExcludedFiles:    request.ExcludedFiles,
		FindingIDs:       append([]string(nil), acceptedIDs...),
		RejectedFindings: rejected, Models: models, CompletedAt: completedAt,
		TargetBranch:            request.TargetBranch,
		AdvertisedDefaultBranch: request.AdvertisedDefaultBranch,
		TargetIsDefault:         request.TargetIsDefault,
	}
	state.Runs = append(state.Runs, run)
	if len(state.Runs) > 1000 {
		state.Runs = append([]ReviewRun(nil), state.Runs[len(state.Runs)-1000:]...)
	}
	pruneCheckpointMetadata(&state, request.Plan, files)
	state.LastCommitSHA = request.Plan.CommitSHA
	state.LastExcludedFiles = request.ExcludedFiles
	ensureMappingJobsForFindings(&state, acceptedIDs, completedAt)
	if request.Plan.ForceCampaignID != "" && run.RemainingFiles > 0 {
		state.ActiveForceCampaignID = request.Plan.ForceCampaignID
		state.ActiveForceProfileHash = request.Plan.ProfileHash
		state.ActiveForceCommitSHA = request.Plan.CommitSHA
	} else {
		state.ActiveForceCampaignID = ""
		state.ActiveForceProfileHash = ""
		state.ActiveForceCommitSHA = ""
	}
	state.Version++
	state.ReviewVersion++
	state.UpdatedAt = completedAt
	if err := s.save(&state); err != nil {
		return RecordResult{}, err
	}
	return RecordResult{State: state, Run: run, AcceptedFindingIDs: acceptedIDs}, nil
}

// SnapshotMappingJobs freezes the assigned reviewer/profile/account into
// newly created pending mapping jobs before dispatch. Existing snapshots are
// immutable, so retries after profile changes continue with original
// provenance.
func (s Store) SnapshotMappingJobs(
	repository string,
	findingIDs []string,
	snapshot RepositoryMappingModelSnapshot,
) (RepositoryState, error) {
	if err := validateMappingModelSnapshot(snapshot); err != nil || mappingModelSnapshotEmpty(snapshot) {
		if err == nil {
			err = errors.New("mapping model snapshot is required")
		}
		return RepositoryState{}, err
	}
	wanted := make(map[string]struct{}, len(findingIDs))
	for _, findingID := range findingIDs {
		if findingID = strings.TrimSpace(findingID); findingID != "" {
			wanted[findingID] = struct{}{}
		}
	}
	if len(wanted) == 0 {
		return RepositoryState{}, errors.New("mapping finding IDs are required")
	}
	unlock, err := s.lock(repository)
	if err != nil {
		return RepositoryState{}, err
	}
	defer unlock()
	state, err := s.load(repository)
	if err != nil {
		return RepositoryState{}, err
	}
	if err := HistoricalDeduplicationMutationAllowed(state); err != nil {
		return RepositoryState{}, err
	}
	now := s.clock()
	changed := false
	for index := range state.MappingJobs {
		job := &state.MappingJobs[index]
		if _, ok := wanted[job.ReviewFindingID]; !ok || job.State == RepositoryMappingCompleted {
			continue
		}
		if !mappingModelSnapshotEmpty(job.ModelSnapshot) {
			if !mappingModelSnapshotsEqual(job.ModelSnapshot, snapshot) {
				return RepositoryState{}, ErrConflict
			}
			continue
		}
		job.ModelSnapshot = snapshot
		job.UpdatedAt = now
		changed = true
	}
	if !changed {
		return state, nil
	}
	state.Version++
	state.UpdatedAt = now
	if err := s.save(&state); err != nil {
		return RepositoryState{}, err
	}
	return state, nil
}

func (s Store) FinalizeNoopPlan(plan Plan, excludedFiles ...int) (RepositoryState, error) {
	if plan.ID == "" || plan.ID != planDigest(plan) || len(plan.PendingFiles) != 0 ||
		len(plan.DeferredFiles) != 0 || !plan.Authoritative {
		return RepositoryState{}, ErrInvalidPlan
	}
	campaignSelectedFiles := 0
	if plan.CampaignID != "" {
		var campaignErr error
		campaignSelectedFiles, campaignErr = validateRepositoryReviewCampaignPlan(plan)
		if campaignErr != nil {
			return RepositoryState{}, campaignErr
		}
	}
	unlock, err := s.lock(plan.Repository)
	if err != nil {
		return RepositoryState{}, err
	}
	defer unlock()
	state, err := s.load(plan.Repository)
	if err != nil {
		return RepositoryState{}, err
	}
	if state.ReviewVersion != plan.StateVersion {
		return RepositoryState{}, ErrConflict
	}
	changed := false
	if plan.CampaignID != "" {
		scopeDigest, _ := repositoryReviewCampaignScopeDigestForPlan(plan)
		bound, bindErr := bindRepositoryReviewCampaignScope(
			&state, plan.CampaignID, plan.CommitSHA, plan.InventoryHash, plan.ProfileHash, scopeDigest,
			plan.RequiredAssignments,
			campaignSelectedFiles,
		)
		if bindErr != nil {
			return RepositoryState{}, bindErr
		}
		changed = bound
		for _, file := range plan.UnchangedFiles {
			merged, mergeErr := mergeRepositoryReviewCampaignPath(
				state.CurrentCampaign, file.Path,
				RepositoryReviewCampaignPathCoverage{Completed: true},
			)
			if mergeErr != nil {
				return RepositoryState{}, mergeErr
			}
			changed = changed || merged
		}
		for _, unsupported := range plan.UnsupportedFiles {
			merged, mergeErr := mergeRepositoryReviewCampaignPath(
				state.CurrentCampaign, unsupported.Path,
				RepositoryReviewCampaignPathCoverage{Unsupported: true},
			)
			if mergeErr != nil {
				return RepositoryState{}, mergeErr
			}
			changed = changed || merged
		}
	}
	changed = pruneCheckpointMetadata(&state, plan, nil) || changed
	excluded := 0
	if len(excludedFiles) > 0 {
		excluded = excludedFiles[0]
	}
	if excluded < 0 || excluded > maxReviewFiles {
		return RepositoryState{}, ErrInvalidPlan
	}
	if state.LastExcludedFiles != excluded {
		state.LastExcludedFiles = excluded
		changed = true
	}
	if state.LastCommitSHA != plan.CommitSHA {
		state.LastCommitSHA = plan.CommitSHA
		changed = true
	}
	if !changed {
		return state, nil
	}
	state.Version++
	state.ReviewVersion++
	state.UpdatedAt = s.clock()
	if err := s.save(&state); err != nil {
		return RepositoryState{}, err
	}
	return state, nil
}

func pruneCheckpointMetadata(state *RepositoryState, plan Plan, pending []FileRef) bool {
	if state == nil || !plan.Authoritative {
		return false
	}
	current := make(
		map[string]struct{},
		len(pending)+len(plan.DeferredFiles)+len(plan.UnchangedFiles)+len(plan.UnsupportedFiles),
	)
	for _, file := range append(append(append([]FileRef(nil), pending...), plan.DeferredFiles...), plan.UnchangedFiles...) {
		current[file.Path] = struct{}{}
	}
	for _, unsupported := range plan.UnsupportedFiles {
		current[unsupported.Path] = struct{}{}
	}
	changed := false
	for pathValue := range state.Files {
		if _, exists := current[pathValue]; !exists {
			delete(state.Files, pathValue)
			changed = true
		}
	}
	for pathValue := range state.Unsupported {
		if _, exists := current[pathValue]; !exists {
			delete(state.Unsupported, pathValue)
			changed = true
		}
	}
	for pathValue := range state.ReviewAttempts {
		if _, exists := current[pathValue]; !exists {
			delete(state.ReviewAttempts, pathValue)
			delete(state.ReviewAttemptIdentities, pathValue)
			changed = true
		}
	}
	return changed
}

func (s Store) Get(repository string) (RepositoryState, bool, error) {
	unlock, err := s.lock(repository)
	if err != nil {
		return RepositoryState{}, false, err
	}
	defer unlock()
	state, err := s.load(repository)
	if err != nil {
		return RepositoryState{}, false, err
	}
	return state, state.Version > 0, nil
}

func (s Store) GetByID(id string) (RepositoryState, bool, error) {
	id = strings.TrimSpace(id)
	suffix, valid := strings.CutPrefix(id, "rrp_")
	if !valid || len(suffix) != 64 || !validHexDigest(suffix) {
		return RepositoryState{}, false, nil
	}
	if err := s.requireSafeRoot(true); err != nil {
		return RepositoryState{}, false, err
	}
	statePath := filepath.Join(s.root, "repo_"+suffix+".json")
	info, err := os.Lstat(statePath)
	if os.IsNotExist(err) {
		return RepositoryState{}, false, nil
	}
	if err != nil {
		return RepositoryState{}, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > maxStateFileBytes {
		return RepositoryState{}, false, errors.New("invalid repository review state")
	}
	file, err := os.Open(statePath)
	if err != nil {
		return RepositoryState{}, false, err
	}
	var summary RepositorySummary
	decodeErr := json.NewDecoder(file).Decode(&summary)
	closeErr := file.Close()
	if decodeErr != nil || closeErr != nil {
		return RepositoryState{}, false, errors.Join(decodeErr, closeErr)
	}
	if summary.ID != id || summary.ID != RepositoryID(summary.Repository) {
		return RepositoryState{}, false, errors.New("repository review state ID mismatch")
	}
	return s.Get(summary.Repository)
}

func (s Store) ListSummaries() ([]RepositorySummary, error) {
	return s.listSummaries(10_000)
}

func (s Store) listSummaries(maximum int) ([]RepositorySummary, error) {
	if err := s.requireSafeRoot(true); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.root)
	if os.IsNotExist(err) {
		return []RepositorySummary{}, nil
	}
	if err != nil {
		return nil, err
	}
	summaries := make([]RepositorySummary, 0, len(entries))
	stateCount := 0
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("repository review state %q must not be a symlink", entry.Name())
		}
		if entry.IsDir() || !entry.Type().IsRegular() || !repositoryReviewStateFilename(entry.Name()) {
			continue
		}
		stateCount++
		if stateCount > maximum {
			return nil, errors.New("repository review catalog exceeds its repository limit")
		}
		summary, summaryErr := repositoryReviewSummaryFromEntry(s.root, entry)
		if summaryErr != nil {
			return nil, summaryErr
		}
		if _, purging, purgeErr := s.loadPurgeFence(summary.Repository); purgeErr != nil {
			return nil, purgeErr
		} else if purging {
			return nil, ErrRepositoryReviewPurgeInProgress
		}
		if summary.SchemaVersion != SchemaVersion {
			state, found, migrationErr := s.Get(summary.Repository)
			if migrationErr != nil {
				return nil, migrationErr
			}
			if !found {
				return nil, errors.New("repository review state disappeared during migration")
			}
			summary = Summarize(state)
		}
		summaries = append(summaries, summary)
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].UpdatedAt.After(summaries[j].UpdatedAt) })
	return summaries, nil
}

func repositoryReviewSummaryFromEntry(root string, entry os.DirEntry) (RepositorySummary, error) {
	statePath := filepath.Join(root, entry.Name())
	stateInfo, infoErr := entry.Info()
	if infoErr != nil || stateInfo.Size() > maxStateFileBytes {
		return RepositorySummary{}, errors.Join(infoErr, errors.New("repository review state exceeds its size limit"))
	}
	summaryPath := strings.TrimSuffix(statePath, ".json") + ".summary.json"
	readPath := statePath
	if summaryInfo, summaryErr := os.Lstat(summaryPath); summaryErr == nil {
		if summaryInfo.Mode()&os.ModeSymlink != 0 || !summaryInfo.Mode().IsRegular() {
			return RepositorySummary{}, errors.New("repository review summary must be a regular file")
		}
		if !summaryInfo.ModTime().Before(stateInfo.ModTime()) {
			readPath = summaryPath
		}
	} else if !os.IsNotExist(summaryErr) {
		return RepositorySummary{}, summaryErr
	}
	file, openErr := os.Open(readPath)
	if openErr != nil {
		return RepositorySummary{}, openErr
	}
	var summary RepositorySummary
	decodeErr := json.NewDecoder(file).Decode(&summary)
	closeErr := file.Close()
	if decodeErr != nil || closeErr != nil {
		return RepositorySummary{}, errors.Join(decodeErr, closeErr)
	}
	if (summary.SchemaVersion < 1 || summary.SchemaVersion > SchemaVersion) ||
		summary.ID != RepositoryID(summary.Repository) {
		return RepositorySummary{}, errors.New("invalid repository review summary")
	}
	return summary, nil
}

func (s Store) SetFindingStatus(
	repository string,
	findingID string,
	status FindingStatus,
	expectedVersion int64,
) (RepositoryState, error) {
	if status != FindingOpen && status != FindingDismissed {
		return RepositoryState{}, errors.New("invalid repository review finding status")
	}
	unlock, err := s.lock(repository)
	if err != nil {
		return RepositoryState{}, err
	}
	defer unlock()
	state, err := s.load(repository)
	if err != nil {
		return RepositoryState{}, err
	}
	if err := HistoricalDeduplicationMutationAllowed(state); err != nil {
		return RepositoryState{}, err
	}
	index := -1
	for candidate := range state.Findings {
		if state.Findings[candidate].ID == strings.TrimSpace(findingID) {
			index = candidate
			break
		}
	}
	if index < 0 {
		return RepositoryState{}, os.ErrNotExist
	}
	if state.Findings[index].DeduplicationPending {
		return RepositoryState{}, ErrConflict
	}
	if state.Findings[index].Status == status {
		return state, nil
	}
	if state.Findings[index].Status == FindingPosted || state.Findings[index].IssueDraftID != "" {
		return RepositoryState{}, ErrConflict
	}
	if expectedVersion < 1 || state.Version != expectedVersion {
		return RepositoryState{}, ErrConflict
	}
	now := s.clock()
	state.Findings[index].Status = status
	state.Findings[index].Version++
	state.Findings[index].UpdatedAt = now
	state.Version++
	state.UpdatedAt = now
	if err := s.save(&state); err != nil {
		return RepositoryState{}, err
	}
	return state, nil
}

func (s Store) PrepareIssue(request IssueDraftRequest) (RepositoryState, IssueDraft, error) {
	request.Repository = strings.TrimSpace(request.Repository)
	if len(request.FindingIDs) != 1 {
		return RepositoryState{}, IssueDraft{}, errors.New(
			"legacy repository review issue drafts require exactly one finding",
		)
	}
	unlock, err := s.lock(request.Repository)
	if err != nil {
		return RepositoryState{}, IssueDraft{}, err
	}
	defer unlock()
	state, err := s.load(request.Repository)
	if err != nil {
		return RepositoryState{}, IssueDraft{}, err
	}
	if historicalErr := HistoricalDeduplicationMutationAllowed(state); historicalErr != nil {
		return RepositoryState{}, IssueDraft{}, historicalErr
	}
	findings, ids, err := selectedFindings(state.Findings, request.FindingIDs)
	if err != nil {
		return RepositoryState{}, IssueDraft{}, err
	}
	for _, finding := range findings {
		if !repositoryFindingAllowsIssueActions(state, finding) {
			return RepositoryState{}, IssueDraft{}, ErrConflict
		}
	}
	title := strings.TrimSpace(request.Title)
	if title == "" {
		title = defaultIssueTitle(findings)
	}
	body := strings.TrimSpace(request.Body)
	if body == "" {
		body = defaultIssueBody(state, findings)
	}
	labels := normalizeLabels(request.Labels)
	if len(labels) == 0 {
		labels = []string{"bug"}
	}
	if !validBoundedText(title, 256) || !validBoundedText(body, maxIssueDraftBodyBytes) {
		return RepositoryState{}, IssueDraft{}, errors.New("invalid repository review issue draft")
	}
	draftID := stableID(
		"rid_", state.Repository, strings.Join(ids, "\x00"), title, body,
		strings.Join(labels, "\x00"),
	)
	for _, existing := range state.IssueDrafts {
		if existing.ID == draftID {
			return state, existing, nil
		}
	}
	if request.ExpectedVersion < 1 || state.Version != request.ExpectedVersion {
		return RepositoryState{}, IssueDraft{}, ErrConflict
	}
	now := s.clock()
	draft := IssueDraft{
		ID:         draftID,
		Repository: state.Repository, FindingIDs: ids, Title: title, Body: body,
		Origin: IssueDraftOriginLegacy,
		Labels: labels, State: IssueDraftEditing, Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	state.IssueDrafts = append(state.IssueDrafts, draft)
	state.Version++
	state.UpdatedAt = now
	if err := s.save(&state); err != nil {
		return RepositoryState{}, IssueDraft{}, err
	}
	if index := issueDraftIndexByID(state.IssueDrafts, draft.ID); index >= 0 {
		draft = state.IssueDrafts[index]
	}
	return state, draft, nil
}

func (s Store) UpdateIssueDraft(
	repository, draftID, title, body string,
	labels []string,
	expectedVersion int64,
) (RepositoryState, IssueDraft, error) {
	unlock, err := s.lock(repository)
	if err != nil {
		return RepositoryState{}, IssueDraft{}, err
	}
	defer unlock()
	state, err := s.load(repository)
	if err != nil {
		return RepositoryState{}, IssueDraft{}, err
	}
	if err := HistoricalDeduplicationMutationAllowed(state); err != nil {
		return RepositoryState{}, IssueDraft{}, err
	}
	index := -1
	for candidate := range state.IssueDrafts {
		if state.IssueDrafts[candidate].ID == strings.TrimSpace(draftID) {
			index = candidate
			break
		}
	}
	if index < 0 {
		return RepositoryState{}, IssueDraft{}, os.ErrNotExist
	}
	draft := &state.IssueDrafts[index]
	if draft.State != IssueDraftEditing || !draft.Canonical {
		return RepositoryState{}, IssueDraft{}, ErrConflict
	}
	title, body = strings.TrimSpace(title), strings.TrimSpace(body)
	labels = normalizeLabels(labels)
	if draft.Title == title && draft.Body == body &&
		strings.Join(draft.Labels, "\x00") == strings.Join(labels, "\x00") {
		return state, *draft, nil
	}
	if expectedVersion < 1 || draft.Version != expectedVersion {
		return RepositoryState{}, IssueDraft{}, ErrConflict
	}
	if !validBoundedText(title, 256) || !validBoundedText(body, maxIssueDraftBodyBytes) {
		return RepositoryState{}, IssueDraft{}, errors.New("invalid repository review issue draft")
	}
	draft.Title, draft.Body, draft.Labels = title, body, labels
	draft.Version++
	draft.UpdatedAt = s.clock()
	state.Version++
	state.UpdatedAt = draft.UpdatedAt
	if err := s.save(&state); err != nil {
		return RepositoryState{}, IssueDraft{}, err
	}
	return state, *draft, nil
}

func (s Store) SetIssueDraftPublication(
	repository, draftID string,
	expectedVersion int64,
	publicationState IssueDraftState,
	externalID, externalURL string,
) (RepositoryState, IssueDraft, error) {
	if publicationState != IssueDraftEditing && publicationState != IssueDraftPosted &&
		publicationState != IssueDraftUnknown {
		return RepositoryState{}, IssueDraft{}, errors.New("invalid issue publication state")
	}
	unlock, err := s.lock(repository)
	if err != nil {
		return RepositoryState{}, IssueDraft{}, err
	}
	defer unlock()
	state, err := s.load(repository)
	if err != nil {
		return RepositoryState{}, IssueDraft{}, err
	}
	if err := HistoricalDeduplicationMutationAllowed(state); err != nil {
		return RepositoryState{}, IssueDraft{}, err
	}
	index := -1
	for candidate := range state.IssueDrafts {
		if state.IssueDrafts[candidate].ID == strings.TrimSpace(draftID) {
			index = candidate
			break
		}
	}
	if index < 0 {
		return RepositoryState{}, IssueDraft{}, os.ErrNotExist
	}
	draft := &state.IssueDrafts[index]
	if !draft.Canonical {
		return RepositoryState{}, IssueDraft{}, ErrConflict
	}
	if draft.State == IssueDraftPosted {
		return state, *draft, nil
	}
	if expectedVersion < 1 || draft.Version != expectedVersion {
		return RepositoryState{}, IssueDraft{}, ErrConflict
	}
	if draft.State != IssueDraftPublishing && draft.State != IssueDraftUnknown {
		return RepositoryState{}, IssueDraft{}, ErrConflict
	}
	if publicationState == IssueDraftEditing && draft.State != IssueDraftPublishing {
		return RepositoryState{}, IssueDraft{}, ErrConflict
	}
	if publicationState == IssueDraftPosted &&
		(!validBoundedText(strings.TrimSpace(externalID), 1024) ||
			!validBoundedText(strings.TrimSpace(externalURL), 4096) ||
			!strings.HasPrefix(strings.TrimSpace(externalURL), "https://")) {
		return RepositoryState{}, IssueDraft{}, errors.New("posted issue identity is required")
	}
	if publicationState == IssueDraftPosted {
		for _, findingID := range draft.FindingIDs {
			findingIndex := findingIndexByID(state.Findings, findingID)
			if findingIndex < 0 ||
				!repositoryFindingAllowsIssueActions(state, state.Findings[findingIndex]) {
				return RepositoryState{}, IssueDraft{}, ErrConflict
			}
		}
	}
	now := s.clock()
	draft.State = publicationState
	draft.ExternalID = strings.TrimSpace(externalID)
	draft.ExternalURL = strings.TrimSpace(externalURL)
	draft.Version++
	draft.UpdatedAt = now
	if publicationState == IssueDraftPosted {
		selected := make(map[string]struct{}, len(draft.FindingIDs))
		for _, id := range draft.FindingIDs {
			selected[id] = struct{}{}
		}
		for findingIndex := range state.Findings {
			if _, ok := selected[state.Findings[findingIndex].ID]; !ok {
				continue
			}
			state.Findings[findingIndex].Status = FindingPosted
			state.Findings[findingIndex].Version++
			state.Findings[findingIndex].UpdatedAt = now
		}
	}
	state.Version++
	state.UpdatedAt = now
	if err := s.save(&state); err != nil {
		return RepositoryState{}, IssueDraft{}, err
	}
	return state, *draft, nil
}

func (s Store) ClaimIssueDraftPublication(
	repository, draftID string,
	expectedVersion int64,
) (RepositoryState, IssueDraft, bool, error) {
	unlock, err := s.lock(repository)
	if err != nil {
		return RepositoryState{}, IssueDraft{}, false, err
	}
	defer unlock()
	state, err := s.load(repository)
	if err != nil {
		return RepositoryState{}, IssueDraft{}, false, err
	}
	historicalMutationErr := HistoricalDeduplicationMutationAllowed(state)
	index := -1
	for candidate := range state.IssueDrafts {
		if state.IssueDrafts[candidate].ID == strings.TrimSpace(draftID) {
			index = candidate
			break
		}
	}
	if index < 0 {
		if historicalMutationErr != nil {
			return RepositoryState{}, IssueDraft{}, false, historicalMutationErr
		}
		return RepositoryState{}, IssueDraft{}, false, os.ErrNotExist
	}
	draft := &state.IssueDrafts[index]
	eligibility := EvaluateIssuePublication(state, *draft)
	if draft.State == IssueDraftPosted {
		if !eligibility.AllowsPostedAcknowledgement() {
			if eligibility.HasBlocker(IssuePublicationHistoricalMergeActive) {
				return RepositoryState{}, IssueDraft{}, false, ErrHistoricalDeduplicationInProgress
			}
			return RepositoryState{}, IssueDraft{}, false, ErrConflict
		}
		return state, *draft, false, nil
	}
	if !eligibility.CanPublish {
		if eligibility.HasBlocker(IssuePublicationHistoricalMergeActive) {
			return RepositoryState{}, IssueDraft{}, false, ErrHistoricalDeduplicationInProgress
		}
		return RepositoryState{}, IssueDraft{}, false, ErrConflict
	}
	if draft.State == IssueDraftPublishing || draft.State == IssueDraftUnknown {
		return state, *draft, false, nil
	}
	if draft.State != IssueDraftEditing || expectedVersion < 1 || draft.Version != expectedVersion {
		return RepositoryState{}, IssueDraft{}, false, ErrConflict
	}
	now := s.clock()
	draft.State = IssueDraftPublishing
	draft.Version++
	draft.UpdatedAt = now
	state.Version++
	state.UpdatedAt = now
	if err := s.save(&state); err != nil {
		return RepositoryState{}, IssueDraft{}, false, err
	}
	return state, *draft, true, nil
}

func (s Store) List() ([]RepositoryState, error) {
	return s.listStates(true)
}

func (s Store) listStates(enforcePurgeFence bool) ([]RepositoryState, error) {
	if err := s.requireSafeRoot(true); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.root)
	if os.IsNotExist(err) {
		return []RepositoryState{}, nil
	}
	if err != nil {
		return nil, err
	}
	states := make([]RepositoryState, 0, len(entries))
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("repository review state %q must not be a symlink", entry.Name())
		}
		if entry.IsDir() || !entry.Type().IsRegular() || !repositoryReviewStateFilename(entry.Name()) {
			continue
		}
		state, stateErr := repositoryReviewStateFromEntry(s.root, entry)
		if stateErr != nil {
			return nil, stateErr
		}
		if enforcePurgeFence {
			if _, purging, purgeErr := s.loadPurgeFence(state.Repository); purgeErr != nil {
				return nil, purgeErr
			} else if purging {
				return nil, ErrRepositoryReviewPurgeInProgress
			}
		}
		states = append(states, state)
	}
	sort.Slice(states, func(i, j int) bool { return states[i].UpdatedAt.After(states[j].UpdatedAt) })
	return states, nil
}

func repositoryReviewStateFromEntry(root string, entry os.DirEntry) (RepositoryState, error) {
	info, infoErr := entry.Info()
	if infoErr != nil {
		return RepositoryState{}, infoErr
	}
	if info.Size() > maxStateFileBytes {
		return RepositoryState{}, errors.New("repository review state exceeds its size limit")
	}
	data, readErr := os.ReadFile(filepath.Join(root, entry.Name()))
	if readErr != nil {
		return RepositoryState{}, readErr
	}
	var state RepositoryState
	if jsonErr := json.Unmarshal(data, &state); jsonErr != nil {
		return RepositoryState{}, jsonErr
	}
	if _, migrationErr := migrateRepositoryState(&state); migrationErr != nil {
		return RepositoryState{}, migrationErr
	}
	backfillCanonicalIssueAssociations(&state)
	if err := validateState(state); err != nil {
		return RepositoryState{}, err
	}
	return state, nil
}

func (s Store) load(repository string) (RepositoryState, error) {
	if _, purging, err := s.loadPurgeFence(repository); err != nil {
		return RepositoryState{}, err
	} else if purging {
		return RepositoryState{}, ErrRepositoryReviewPurgeInProgress
	}
	return s.loadIgnoringPurge(repository)
}

func (s Store) loadIgnoringPurge(repository string) (RepositoryState, error) {
	if s.loadForTest != nil {
		return s.loadForTest(repository)
	}
	state := RepositoryState{
		SchemaVersion:           SchemaVersion,
		ID:                      RepositoryID(repository),
		Repository:              strings.TrimSpace(repository),
		Files:                   make(map[string]ReviewedFile),
		Unsupported:             make(map[string]UnsupportedFile),
		ReviewAttempts:          make(map[string]int),
		ReviewAttemptIdentities: make(map[string]string),
		Findings:                []Finding{},
		RawFindings:             []RawReviewFinding{},
		DeduplicatedFindings:    []DeduplicatedReviewFinding{},
		DeduplicationJobs:       []DeduplicationJob{},
		Contexts:                []FindingContext{},
		Runs:                    []ReviewRun{},
		FileAttributions:        []RepositoryReviewFileAttribution{},
		IssueDrafts:             []IssueDraft{},
		RepositoryFindings:      []RepositoryFinding{},
		MappingJobs:             []RepositoryMappingJob{},
		ValidationJobs:          []RepositoryValidationJob{},
	}
	if err := s.requireSafeRoot(true); err != nil {
		return RepositoryState{}, err
	}
	statePath := s.path(repository)
	info, err := os.Lstat(statePath)
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return RepositoryState{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return RepositoryState{}, errors.New("repository review state must be a regular file")
	}
	if info.Size() > maxStateFileBytes {
		return RepositoryState{}, errors.New("repository review state exceeds its size limit")
	}
	data, err := os.ReadFile(statePath)
	if err != nil {
		return RepositoryState{}, err
	}
	if unmarshalErr := json.Unmarshal(data, &state); unmarshalErr != nil {
		return RepositoryState{}, unmarshalErr
	}
	migrated, err := migrateRepositoryState(&state)
	if err != nil {
		return RepositoryState{}, err
	}
	migrated = backfillCanonicalIssueAssociations(&state) || migrated
	if err := validateState(state); err != nil {
		return RepositoryState{}, err
	}
	if state.Repository != strings.TrimSpace(repository) {
		return RepositoryState{}, errors.New("repository review state identity mismatch")
	}
	if migrated {
		if err := s.save(&state); err != nil {
			return RepositoryState{}, err
		}
	}
	return state, nil
}

func (s Store) save(state *RepositoryState) error {
	if state == nil {
		return errors.New("repository review state is required")
	}
	if state.FileAttributions == nil {
		state.FileAttributions = []RepositoryReviewFileAttribution{}
	}
	backfillCanonicalIssueAssociations(state)
	synchronizeRepositoryFindingIssues(state)
	synchronizeDeduplicatedFindingProjections(state)
	summary := Summarize(*state)
	state.FindingCount = summary.FindingCount
	state.RepositoryFindingCount = summary.RepositoryFindingCount
	state.OpenFindingCount = summary.OpenFindingCount
	state.IssueDraftCount = summary.IssueDraftCount
	state.UnsupportedCount = summary.UnsupportedCount
	state.ReviewedFileCount = summary.ReviewedFileCount
	if err := validateState(*state); err != nil {
		return err
	}
	if err := s.ensureSafeRoot(fileutil.MkdirAllDurable); err != nil {
		return err
	}
	if info, err := os.Lstat(s.path(state.Repository)); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return errors.New("repository review state must be a regular file")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if int64(len(data)) > maxStateFileBytes {
		return errors.New("repository review state exceeds its size limit")
	}
	statePath := s.path(state.Repository)
	summaryPath := strings.TrimSuffix(statePath, ".json") + ".summary.json"
	if removeErr := os.Remove(summaryPath); removeErr != nil && !os.IsNotExist(removeErr) {
		return removeErr
	}
	if writeErr := fileutil.WriteFileAtomic(statePath, data, 0o600); writeErr != nil {
		return writeErr
	}
	// RepositorySummary contains only JSON-native scalar and time fields.
	summaryData, _ := json.Marshal(Summarize(*state))
	// The sidecar is a rebuildable list projection. The authoritative state is
	// already committed, so a projection write failure must not turn a successful
	// versioned mutation into an ambiguous failure.
	_ = fileutil.WriteFileAtomic(summaryPath, summaryData, 0o600)
	return nil
}

func (s Store) requireSafeRoot(allowMissing bool) error {
	info, err := os.Lstat(s.root)
	if os.IsNotExist(err) && allowMissing {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("repository review storage root must be a real directory")
	}
	return nil
}

func (s Store) ensureSafeRoot(mkdir func(string, os.FileMode) error) error {
	if err := s.requireSafeRoot(true); err != nil {
		return err
	}
	if err := mkdir(s.root, 0o700); err != nil {
		return err
	}
	return s.requireSafeRoot(false)
}

func (s Store) path(repository string) string {
	return filepath.Join(s.root, stableID("repo_", strings.TrimSpace(repository))+".json")
}

func repositoryReviewStateFilename(name string) bool {
	return strings.HasPrefix(name, "repo_") && strings.HasSuffix(name, ".json") &&
		!strings.HasSuffix(name, ".summary.json")
}

func validHexDigest(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' && character < 'a' || character > 'f' {
			return false
		}
	}
	return value != ""
}

func (s Store) lock(repository string) (func(), error) {
	key := s.root + "\x00" + strings.TrimSpace(repository)
	value, _ := storeLocks.LoadOrStore(key, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	unlockFile, err := lockRepositoryReviewStore(s.root)
	if err != nil {
		mutex.Unlock()
		return nil, err
	}
	return func() {
		unlockFile()
		mutex.Unlock()
	}, nil
}

func (s Store) clock() time.Time {
	if s.now == nil {
		return time.Now().UTC()
	}
	return s.now().UTC()
}

func normalizeFiles(files []FileRef) ([]FileRef, error) {
	out := make([]FileRef, 0, len(files))
	seen := make(map[string]struct{}, len(files))
	metadataBytes := 0
	for _, file := range files {
		file.Path = strings.TrimSpace(filepath.ToSlash(file.Path))
		file.BlobSHA = strings.ToLower(strings.TrimSpace(file.BlobSHA))
		file.Category = strings.TrimSpace(file.Category)
		file.Mode = strings.TrimSpace(file.Mode)
		if !validRepositoryReviewPath(file.Path) ||
			!validBlobSHA(file.BlobSHA) || file.SizeBytes < 0 {
			return nil, fmt.Errorf("%w: invalid file reference", ErrInvalidPlan)
		}
		if _, duplicate := seen[file.Path]; duplicate {
			return nil, fmt.Errorf("%w: duplicate file path %q", ErrInvalidPlan, file.Path)
		}
		metadataBytes += len(file.Path) + len(file.BlobSHA) + len(file.Category) + len(file.Mode) + 32
		if metadataBytes > maxReviewFileMetadataBytes {
			return nil, fmt.Errorf("%w: file inventory metadata exceeds its size limit", ErrInvalidPlan)
		}
		seen[file.Path] = struct{}{}
		out = append(out, file)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func validRepositoryReviewPath(pathValue string) bool {
	if !validBoundedText(pathValue, 4096) || pathValue == "." ||
		pathValue != strings.TrimSpace(filepath.ToSlash(pathValue)) {
		return false
	}
	cleanPath := path.Clean(pathValue)
	return cleanPath == pathValue && !strings.HasPrefix(cleanPath, "../") &&
		!strings.HasPrefix(cleanPath, "/")
}

func reviewAttemptIdentity(file FileRef, profileHash string) string {
	return stableID(
		"rat_", file.Path, file.BlobSHA, fmt.Sprint(file.SizeBytes), file.Mode,
		strings.TrimSpace(profileHash),
	)
}

func reviewAttemptsFor(state RepositoryState, file FileRef, profileHash string) int {
	if state.ReviewAttemptIdentities[file.Path] != reviewAttemptIdentity(file, profileHash) {
		return 0
	}
	return state.ReviewAttempts[file.Path]
}

func validBlobSHA(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' && character < 'a' || character > 'f' {
			return false
		}
	}
	return true
}

func replayedRun(state RepositoryState, request RecordRequest) (ReviewRun, bool) {
	for _, run := range state.Runs {
		if run.ID != request.RunID {
			continue
		}
		return run, run.PlanID == request.Plan.ID
	}
	return ReviewRun{}, false
}

func deriveRepositoryReviewCampaignEvidence(
	evidence []RepositoryReviewEvidence,
	allowed map[string]FileRef,
	requiredAssignments int,
	unsupportedPaths map[string]struct{},
) ([]Observation, []FileRef, []FileRef, error) {
	if requiredAssignments < 1 || requiredAssignments > maxRepositoryReviewRequiredAssignments ||
		len(evidence) > maxReviewObservations {
		return nil, nil, nil, ErrInvalidPlan
	}
	if len(evidence) == 0 {
		for pathValue := range allowed {
			if _, unsupported := unsupportedPaths[pathValue]; !unsupported {
				return nil, nil, nil, ErrInvalidPlan
			}
		}
	}
	evidenceMetadataBytes := 0
	addEvidenceFiles := func(files []FileRef) error {
		for _, file := range files {
			evidenceMetadataBytes += len(file.Path) + len(file.BlobSHA) +
				len(file.Category) + len(file.Mode) + 32
			if evidenceMetadataBytes > maxReviewFileMetadataBytes {
				return fmt.Errorf("%w: campaign review evidence exceeds its size limit", ErrInvalidPlan)
			}
		}
		return nil
	}
	observations := make([]Observation, 0, len(evidence))
	fileRefs := make(map[string]FileRef)
	requiredCoverage := make(map[string]int)
	successfulRequiredCoverage := make(map[string]int)
	inspectedPaths := make(map[string]struct{})
	assignments := make(map[string]struct{}, len(evidence))
	for index, child := range evidence {
		if err := addEvidenceFiles(child.ScopeFiles); err != nil {
			return nil, nil, nil, err
		}
		if err := addEvidenceFiles(child.AcknowledgedFiles); err != nil {
			return nil, nil, nil, err
		}
		if child.Observation != nil {
			if err := addEvidenceFiles(child.Observation.ScopeFiles); err != nil {
				return nil, nil, nil, err
			}
		}
		if !validBoundedText(child.AssignmentID, 256) ||
			child.AssignmentID != strings.TrimSpace(child.AssignmentID) {
			return nil, nil, nil, fmt.Errorf(
				"%w: campaign review evidence %d has an invalid assignment ID",
				ErrInvalidPlan, index,
			)
		}
		if _, duplicate := assignments[child.AssignmentID]; duplicate {
			return nil, nil, nil, fmt.Errorf(
				"%w: duplicate campaign review assignment %q", ErrInvalidPlan, child.AssignmentID,
			)
		}
		assignments[child.AssignmentID] = struct{}{}
		scope, err := bindRepositoryReviewCampaignFiles(child.ScopeFiles, allowed)
		if err != nil || len(scope) == 0 || !reflect.DeepEqual(scope, child.ScopeFiles) {
			return nil, nil, nil, fmt.Errorf(
				"campaign review evidence %d scope: %w", index, ErrInvalidPlan,
			)
		}
		for _, file := range scope {
			fileRefs[file.Path] = file
			if child.Required {
				requiredCoverage[file.Path]++
			}
		}
		if !child.Successful {
			if child.Observation != nil || len(child.AcknowledgedFiles) != 0 {
				return nil, nil, nil, fmt.Errorf(
					"%w: unsuccessful campaign evidence %d contains successful output",
					ErrInvalidPlan, index,
				)
			}
			continue
		}
		if child.Observation == nil ||
			!validFindingSourceProvenance(
				strings.TrimSpace(child.Observation.Model),
				strings.TrimSpace(child.Observation.ModelAlias),
				strings.TrimSpace(child.Observation.Account),
			) ||
			child.Observation.Model != strings.TrimSpace(child.Observation.Model) ||
			child.Observation.ModelAlias != strings.TrimSpace(child.Observation.ModelAlias) ||
			child.Observation.Account != strings.TrimSpace(child.Observation.Account) {
			return nil, nil, nil, fmt.Errorf(
				"%w: successful campaign evidence %d has no valid observation",
				ErrInvalidPlan, index,
			)
		}
		observationScope, err := bindRepositoryReviewCampaignFiles(
			child.Observation.ScopeFiles, allowed,
		)
		if err != nil || !reflect.DeepEqual(observationScope, scope) {
			return nil, nil, nil, fmt.Errorf(
				"%w: campaign evidence %d observation scope does not match assignment",
				ErrInvalidPlan, index,
			)
		}
		acknowledged, err := bindRepositoryReviewCampaignFiles(
			child.AcknowledgedFiles, allowed,
		)
		if err != nil {
			return nil, nil, nil, fmt.Errorf(
				"campaign evidence %d acknowledgements: %w", index, err,
			)
		}
		scopePaths := make(map[string]struct{}, len(scope))
		for _, file := range scope {
			scopePaths[file.Path] = struct{}{}
		}
		for _, file := range acknowledged {
			if _, assigned := scopePaths[file.Path]; !assigned {
				return nil, nil, nil, fmt.Errorf(
					"%w: campaign evidence %d acknowledged an unassigned file",
					ErrInvalidPlan, index,
				)
			}
			inspectedPaths[file.Path] = struct{}{}
			if child.Required {
				successfulRequiredCoverage[file.Path]++
			}
		}
		acknowledgedPaths := make(map[string]struct{}, len(acknowledged))
		for _, file := range acknowledged {
			acknowledgedPaths[file.Path] = struct{}{}
		}
		for findingIndex, finding := range child.Observation.Findings {
			findingPath := strings.TrimSpace(filepath.ToSlash(finding.File))
			if _, acknowledged := acknowledgedPaths[findingPath]; !acknowledged {
				return nil, nil, nil, fmt.Errorf(
					"%w: campaign evidence %d finding %d has no child acknowledgement",
					ErrInvalidPlan, index, findingIndex,
				)
			}
		}
		observation := *child.Observation
		observation.ScopeFiles = observationScope
		observations = append(observations, observation)
	}
	for pathValue := range allowed {
		if _, unsupported := unsupportedPaths[pathValue]; unsupported {
			continue
		}
		if requiredCoverage[pathValue] != requiredAssignments {
			return nil, nil, nil, fmt.Errorf(
				"%w: file %q has %d required assignments, want %d",
				ErrInvalidPlan, pathValue, requiredCoverage[pathValue], requiredAssignments,
			)
		}
	}
	inspected := make([]FileRef, 0, len(inspectedPaths))
	for pathValue := range inspectedPaths {
		inspected = append(inspected, fileRefs[pathValue])
	}
	completed := make([]FileRef, 0)
	for pathValue, total := range requiredCoverage {
		if total > 0 && successfulRequiredCoverage[pathValue] == total {
			completed = append(completed, fileRefs[pathValue])
		}
	}
	sort.Slice(inspected, func(i, j int) bool { return inspected[i].Path < inspected[j].Path })
	sort.Slice(completed, func(i, j int) bool { return completed[i].Path < completed[j].Path })
	return observations, inspected, completed, nil
}

func validateRepositoryReviewCampaignPlan(plan Plan) (int, error) {
	if !ValidRepositoryReviewCampaignID(plan.CampaignID) ||
		!plan.Authoritative ||
		plan.CampaignID != strings.TrimSpace(plan.CampaignID) ||
		!validBoundedText(plan.Repository, maxRepositoryIdentityBytes) ||
		plan.Repository != strings.TrimSpace(plan.Repository) ||
		!validRepositoryReviewCommitSHA(plan.CommitSHA) ||
		plan.CommitSHA != strings.ToLower(strings.TrimSpace(plan.CommitSHA)) ||
		!validBoundedText(strings.TrimSpace(plan.InventoryHash), 256) ||
		plan.InventoryHash != strings.TrimSpace(plan.InventoryHash) ||
		!validBoundedText(strings.TrimSpace(plan.ProfileHash), 256) ||
		plan.ProfileHash != strings.TrimSpace(plan.ProfileHash) ||
		(plan.ForceCampaignID != "" &&
			(plan.ForceCampaignID != strings.TrimSpace(plan.ForceCampaignID) ||
				!validBoundedText(plan.ForceCampaignID, 256))) ||
		plan.RequiredAssignments < 1 || plan.RequiredAssignments > maxRepositoryReviewRequiredAssignments ||
		plan.StateVersion < 0 || plan.PreviouslyReviewed < 0 || plan.PreviouslyReviewed > maxReviewFiles {
		return 0, ErrInvalidPlan
	}
	if len(plan.AssignmentCatalog) > 0 {
		catalog, err := NormalizeRepositoryReviewAssignmentCatalog(plan.AssignmentCatalog)
		if err != nil || !repositoryReviewAssignmentCatalogEqual(catalog, plan.AssignmentCatalog) ||
			catalog[0].ProfileHash != plan.ProfileHash ||
			repositoryReviewRequiredAssignmentCount(catalog) != plan.RequiredAssignments {
			return 0, ErrInvalidPlan
		}
		allowed := make(map[string]FileRef, len(plan.PendingFiles))
		for _, file := range plan.PendingFiles {
			allowed[file.Path] = file
		}
		plans, planErr := normalizeRepositoryReviewAssignmentPlans(
			plan.AssignmentPlans, catalog, allowed,
		)
		if planErr != nil || len(plans) != len(plan.AssignmentPlans) ||
			len(plans) > 0 && !reflect.DeepEqual(plans, plan.AssignmentPlans) {
			return 0, ErrInvalidPlan
		}
	} else if len(plan.AssignmentPlans) != 0 {
		return 0, ErrInvalidPlan
	}
	selected := make(map[string]struct{})
	for _, group := range [][]FileRef{plan.PendingFiles, plan.DeferredFiles, plan.UnchangedFiles} {
		canonical, err := canonicalRepositoryReviewCampaignFiles(group)
		if err != nil || len(canonical) != len(group) {
			return 0, ErrInvalidPlan
		}
		for _, file := range canonical {
			if _, duplicate := selected[file.Path]; duplicate {
				return 0, ErrInvalidPlan
			}
			selected[file.Path] = struct{}{}
		}
	}
	unsupportedRefs := make([]FileRef, 0, len(plan.UnsupportedFiles))
	for _, unsupported := range plan.UnsupportedFiles {
		if !validBoundedText(strings.TrimSpace(unsupported.Reason), 256) ||
			unsupported.Reason != strings.TrimSpace(unsupported.Reason) {
			return 0, ErrInvalidPlan
		}
		unsupportedRefs = append(unsupportedRefs, unsupported.FileRef)
	}
	canonicalUnsupported, err := canonicalRepositoryReviewCampaignFiles(unsupportedRefs)
	if err != nil || len(canonicalUnsupported) != len(unsupportedRefs) {
		return 0, ErrInvalidPlan
	}
	for _, file := range canonicalUnsupported {
		if _, duplicate := selected[file.Path]; duplicate {
			return 0, ErrInvalidPlan
		}
		selected[file.Path] = struct{}{}
	}
	if len(selected) > maxReviewFiles {
		return 0, ErrInvalidPlan
	}
	return len(selected), nil
}

func repositoryReviewCampaignScopeDigestForPlan(plan Plan) (string, error) {
	files, err := repositoryReviewCampaignFilesForPlan(plan)
	if err != nil {
		return "", err
	}
	return repositoryReviewCampaignScopeDigestForFiles(files)
}

func repositoryReviewCampaignFilesForPlan(plan Plan) ([]FileRef, error) {
	files := make([]FileRef, 0,
		len(plan.PendingFiles)+len(plan.DeferredFiles)+len(plan.UnchangedFiles)+len(plan.UnsupportedFiles),
	)
	files = append(files, plan.PendingFiles...)
	files = append(files, plan.DeferredFiles...)
	files = append(files, plan.UnchangedFiles...)
	for _, unsupported := range plan.UnsupportedFiles {
		files = append(files, unsupported.FileRef)
	}
	return canonicalRepositoryReviewCampaignFiles(files)
}

func repositoryReviewCampaignScopeDigestForFiles(files []FileRef) (string, error) {
	canonical, err := canonicalRepositoryReviewCampaignFiles(files)
	if err != nil {
		return "", err
	}
	data, _ := json.Marshal(canonical)
	return stableID("sha256:", string(data)), nil
}

// CanonicalRepositoryReviewCampaignScope validates and sorts an exact campaign
// manifest. It is exposed for trusted controller recovery code; callers still
// need ReconcileCampaign's CAS boundary before the manifest becomes durable.
func CanonicalRepositoryReviewCampaignScope(files []FileRef) ([]FileRef, error) {
	return canonicalRepositoryReviewCampaignFiles(files)
}

// RepositoryReviewCampaignScopeDigest returns the digest ReconcileCampaign
// binds to a canonical exact-file manifest.
func RepositoryReviewCampaignScopeDigest(files []FileRef) (string, error) {
	return repositoryReviewCampaignScopeDigestForFiles(files)
}

func canonicalRepositoryReviewCampaignFiles(files []FileRef) ([]FileRef, error) {
	canonical, err := normalizeFiles(files)
	if err != nil {
		return nil, err
	}
	byPath := make(map[string]FileRef, len(canonical))
	for _, file := range canonical {
		byPath[file.Path] = file
	}
	for _, file := range files {
		if normalized, exists := byPath[file.Path]; !exists || normalized != file {
			return nil, ErrInvalidPlan
		}
	}
	return canonical, nil
}

func bindRepositoryReviewCampaignFiles(
	files []FileRef,
	allowed map[string]FileRef,
) ([]FileRef, error) {
	canonical, err := canonicalRepositoryReviewCampaignFiles(files)
	if err != nil {
		return nil, err
	}
	for _, file := range canonical {
		trusted, ok := allowed[file.Path]
		if !ok || trusted != file {
			return nil, fmt.Errorf("%w: file %q is outside the exact pending plan", ErrInvalidPlan, file.Path)
		}
	}
	return canonical, nil
}

func containsRepositoryReviewFile(files []FileRef, pathValue string) bool {
	for _, file := range files {
		if file.Path == pathValue {
			return true
		}
	}
	return false
}

func bindScopeFiles(files []FileRef, allowed map[string]FileRef) ([]FileRef, error) {
	normalized, err := normalizeFiles(files)
	if err != nil || len(normalized) == 0 {
		return nil, errors.New("exact finding context is empty or invalid")
	}
	for index, file := range normalized {
		trusted, ok := allowed[file.Path]
		if !ok || trusted.BlobSHA != file.BlobSHA || trusted.SizeBytes != file.SizeBytes {
			return nil, fmt.Errorf("context file %q is outside the immutable review plan", file.Path)
		}
		normalized[index] = trusted
	}
	return normalized, nil
}

func fileInScope(path string, scope []FileRef) (FileRef, bool) {
	path = strings.TrimSpace(filepath.ToSlash(path))
	for _, file := range scope {
		if file.Path == path {
			return file, true
		}
	}
	return FileRef{}, false
}

func normalizeCandidate(candidate FindingCandidate) FindingCandidate {
	candidate.Severity = strings.ToLower(strings.TrimSpace(candidate.Severity))
	candidate.Title = strings.TrimSpace(candidate.Title)
	candidate.Symbol = strings.TrimSpace(candidate.Symbol)
	candidate.File = strings.TrimSpace(filepath.ToSlash(candidate.File))
	candidate.Message = strings.TrimSpace(candidate.Message)
	candidate.Evidence = strings.TrimSpace(candidate.Evidence)
	candidate.Impact = strings.TrimSpace(candidate.Impact)
	candidate.Validation.Status = strings.ToLower(strings.TrimSpace(candidate.Validation.Status))
	candidate.Validation.Summary = strings.TrimSpace(candidate.Validation.Summary)
	candidate.MatchHints.Component = strings.TrimSpace(candidate.MatchHints.Component)
	candidate.MatchHints.Operation = strings.TrimSpace(candidate.MatchHints.Operation)
	candidate.MatchHints.FailureMode = strings.TrimSpace(candidate.MatchHints.FailureMode)
	candidate.MatchHints.Trigger = strings.TrimSpace(candidate.MatchHints.Trigger)
	candidate.MatchHints.ViolatedInvariant = strings.TrimSpace(candidate.MatchHints.ViolatedInvariant)
	candidate.MatchHints.ObservableOutcome = strings.TrimSpace(candidate.MatchHints.ObservableOutcome)
	candidate.MatchHints.RelatedSymbols = normalizeFindingIdentityHints(
		candidate.MatchHints.RelatedSymbols,
	)
	candidate.MatchHints.SourceAnchors = normalizeFindingIdentityHints(
		candidate.MatchHints.SourceAnchors,
	)
	candidate.MatchHints.DistinguishingFacts = normalizeFindingIdentityHints(
		candidate.MatchHints.DistinguishingFacts,
	)
	candidate.FixEffort.Quick.Class = strings.ToLower(strings.TrimSpace(candidate.FixEffort.Quick.Class))
	candidate.FixEffort.Quick.Rationale = strings.TrimSpace(candidate.FixEffort.Quick.Rationale)
	candidate.FixEffort.Quality.Class = strings.ToLower(strings.TrimSpace(candidate.FixEffort.Quality.Class))
	candidate.FixEffort.Quality.Rationale = strings.TrimSpace(candidate.FixEffort.Quality.Rationale)
	return candidate
}

// NormalizeRepositoryReviewFindingCandidate returns the detached canonical
// form persisted by Record. Recovery uses it before exact evidence comparison.
func NormalizeRepositoryReviewFindingCandidate(candidate FindingCandidate) FindingCandidate {
	return normalizeCandidate(candidate)
}

// ValidateRepositoryReviewLegacyContextIdentity verifies the stable identity
// originally assigned to an untagged legacy finding context.
func ValidateRepositoryReviewLegacyContextIdentity(contextRecord FindingContext) bool {
	contextRecord.CampaignID = ""
	return contextRecord.ID != "" && contextRecord.ID == stableID(
		"rctx_", contextBindingDigest(contextRecord),
	)
}

// ValidateRepositoryReviewLegacyFindingIdentity verifies immutable finding
// identity and first-observation fields with the exact Record algorithms.
func ValidateRepositoryReviewLegacyFindingIdentity(
	finding Finding,
	origin FindingContext,
	candidate FindingCandidate,
) bool {
	candidate = normalizeCandidate(candidate)
	primary, inScope := fileInScope(candidate.File, origin.Files)
	if !inScope {
		return false
	}
	fingerprint := findingFingerprint(primary, candidate)
	currentID := stableID(
		"rfn_", finding.Repository, finding.CommitSHA, origin.RunID, fingerprint,
	)
	// Before immutable run occurrences were introduced, repository review
	// findings coalesced across runs and their stable ID omitted commit/run
	// identity. Recovery accepts only that exact predecessor formula.
	legacyID := stableID("rfn_", finding.Repository, fingerprint)
	return finding.Fingerprint == fingerprint &&
		(finding.ID == currentID || finding.ID == legacyID) &&
		finding.File == primary && reflect.DeepEqual(finding.Line, candidate.Line) &&
		finding.Title == candidate.Title && finding.Symbol == candidate.Symbol &&
		finding.Message == candidate.Message && finding.Evidence == candidate.Evidence &&
		finding.Impact == candidate.Impact && reflect.DeepEqual(finding.Validation, candidate.Validation) &&
		reflect.DeepEqual(finding.MatchHints, candidate.MatchHints) &&
		reflect.DeepEqual(finding.FixEffort, candidate.FixEffort)
}

func normalizeFindingIdentityHints(values []string) []string {
	if values == nil {
		return nil
	}
	normalized := make([]string, len(values))
	for index, value := range values {
		normalized[index] = strings.TrimSpace(value)
	}
	return normalized
}

func validateCandidate(candidate FindingCandidate) error {
	switch candidate.Severity {
	case "critical", "high", "medium", "low":
	default:
		return errors.New("invalid severity")
	}
	if candidate.Title == "" || candidate.File == "" || candidate.Evidence == "" || candidate.Impact == "" ||
		candidate.Validation.Summary == "" {
		return errors.New("finding is incomplete")
	}
	for _, value := range []string{
		candidate.Title, candidate.File, candidate.Evidence,
		candidate.Impact, candidate.Validation.Summary,
	} {
		if !validBoundedText(value, maxFindingTextBytes) {
			return errors.New("finding text exceeds its limit or is invalid UTF-8")
		}
	}
	if candidate.Message != "" && !validBoundedText(candidate.Message, maxFindingTextBytes) {
		return errors.New("finding message exceeds its limit or is invalid UTF-8")
	}
	if candidate.Symbol != "" && !validBoundedText(candidate.Symbol, 4096) {
		return errors.New("finding symbol is invalid")
	}
	if len(candidate.Validation.Checks) > 128 {
		return errors.New("finding validation has too many checks")
	}
	for _, check := range candidate.Validation.Checks {
		if !validBoundedText(strings.TrimSpace(check), 4096) {
			return errors.New("finding validation check is invalid")
		}
	}
	if candidate.Line != nil && *candidate.Line < 1 {
		return errors.New("finding line must be positive")
	}
	if findingCandidateHasEnrichment(candidate) {
		if err := validateMatchHints(candidate.MatchHints); err != nil {
			return err
		}
		if err := validateFixEffort(candidate.FixEffort); err != nil {
			return err
		}
	}
	return nil
}

// ValidateGeneratedFindingCandidate applies the complete current finder
// contract. Store.Record deliberately accepts candidates without enrichment so
// legacy direct callers and persisted review occurrences remain readable; the
// workflow's native model-output boundary calls this stricter validator.
func ValidateGeneratedFindingCandidate(candidate FindingCandidate) error {
	candidate = normalizeCandidate(candidate)
	if candidate.Symbol == "" || candidate.Message == "" || candidate.Validation.Status != "confirmed" {
		return errors.New("generated finding is incomplete or unconfirmed")
	}
	if !findingCandidateHasEnrichment(candidate) {
		return errors.New("generated finding is missing match hints and fix effort")
	}
	return validateCandidate(candidate)
}

func findingCandidateHasEnrichment(candidate FindingCandidate) bool {
	hints := candidate.MatchHints
	return hints.Component != "" || hints.Operation != "" || hints.FailureMode != "" ||
		hints.Trigger != "" || hints.ViolatedInvariant != "" || hints.ObservableOutcome != "" ||
		len(hints.RelatedSymbols) > 0 || len(hints.SourceAnchors) > 0 ||
		len(hints.DistinguishingFacts) > 0 || candidate.FixEffort != (FixEffort{})
}

func validateMatchHints(hints MatchHints) error {
	for _, value := range []string{
		hints.Component, hints.Operation, hints.FailureMode, hints.Trigger,
		hints.ViolatedInvariant, hints.ObservableOutcome,
	} {
		if !validBoundedText(value, maxMatchHintIdentityBytes) {
			return errors.New("finding match hints are incomplete or invalid")
		}
	}
	for _, values := range [][]string{
		hints.RelatedSymbols, hints.SourceAnchors, hints.DistinguishingFacts,
	} {
		if len(values) > maxMatchHintItems {
			return errors.New("finding match hints have too many identity values")
		}
		seen := make(map[string]struct{}, len(values))
		for _, value := range values {
			if !validBoundedText(value, maxMatchHintIdentityBytes) ||
				strings.ContainsAny(value, "\r\n") {
				return errors.New("finding match hint identity value is invalid")
			}
			key := normalizedText(value)
			if _, duplicate := seen[key]; duplicate {
				return errors.New("finding match hints contain a duplicate identity value")
			}
			seen[key] = struct{}{}
		}
	}
	return nil
}

func validateFixEffort(effort FixEffort) error {
	if err := validateFixEffortEstimate(effort.Quick); err != nil {
		return err
	}
	if err := validateFixEffortEstimate(effort.Quality); err != nil {
		return err
	}
	if effort.Quick.LOCMin > effort.Quality.LOCMin ||
		effort.Quick.LOCMax > effort.Quality.LOCMax {
		return errors.New("finding quality fix effort must not be smaller than quick containment")
	}
	return nil
}

func validateFixEffortEstimate(estimate FixEffortEstimate) error {
	if estimate.LOCMin < 1 || estimate.LOCMin > estimate.LOCMax ||
		estimate.LOCMax > maxFixEffortLOC ||
		!validBoundedText(estimate.Rationale, maxMatchHintIdentityBytes) {
		return errors.New("finding fix effort range or rationale is invalid")
	}
	wantClass := "refactor"
	switch {
	case estimate.LOCMax <= 10:
		wantClass = "tiny"
	case estimate.LOCMax <= 40:
		wantClass = "small"
	case estimate.LOCMax <= 150:
		wantClass = "medium"
	case estimate.LOCMax <= 500:
		wantClass = "large"
	}
	rationale := strings.ToLower(estimate.Rationale)
	architecturalRefactor := estimate.Class == "refactor" &&
		strings.Contains(rationale, "cross-subsystem") &&
		(strings.Contains(rationale, "architectural") ||
			strings.Contains(rationale, "contract migration"))
	if estimate.Class != wantClass && !architecturalRefactor {
		return errors.New("finding fix effort class is inconsistent with its maximum LOC")
	}
	return nil
}

func findingFingerprint(file FileRef, candidate FindingCandidate) string {
	line := 0
	if candidate.Line != nil {
		line = *candidate.Line
	}
	values := []string{
		file.Path, file.BlobSHA, fmt.Sprint(line),
		normalizedText(candidate.Symbol), normalizedText(candidate.Title),
		normalizedText(candidate.Message), normalizedText(candidate.Evidence),
	}
	if findingCandidateHasEnrichment(candidate) {
		encodedHints, _ := json.Marshal(candidate.MatchHints)
		values = append(values, string(encodedHints))
	}
	return stableID("sha256:", values...)
}

func findingIndexByFingerprint(findings []Finding, fingerprint string) int {
	for index := range findings {
		if findings[index].Fingerprint == fingerprint {
			return index
		}
	}
	return -1
}

func semanticFindingIndex(findings []Finding, file FileRef, candidate FindingCandidate) int {
	candidateTitle := findingTokens(candidate.Title)
	candidateBody := findingTokens(candidate.Title + "\n" + candidate.Message + "\n" + candidate.Evidence)
	for index, finding := range findings {
		if finding.File.Path != file.Path || finding.File.BlobSHA != file.BlobSHA ||
			!nearbyLines(finding.Line, candidate.Line) || candidate.Symbol == "" ||
			normalizedText(finding.Symbol) != normalizedText(candidate.Symbol) {
			continue
		}
		if findingCandidateHasEnrichment(candidate) && !matchHintsEmpty(finding.MatchHints) {
			conflicts := append(
				repositoryHardCausalConflicts(candidate.MatchHints, finding.MatchHints),
				repositoryCausalFieldConflicts(candidate.MatchHints, finding.MatchHints)...,
			)
			if len(conflicts) > 0 {
				continue
			}
			causalSimilarity := tokenDice(
				findingTokens(repositoryCausalText(candidate.MatchHints)),
				findingTokens(repositoryCausalText(finding.MatchHints)),
			)
			causalTitleSimilarity := tokenDice(findingTokens(finding.Title), candidateTitle)
			anchorSimilarity := repositoryAnchorJaccard(
				candidate.MatchHints.SourceAnchors, finding.MatchHints.SourceAnchors,
			)
			if causalSimilarity < 0.72 &&
				!(causalTitleSimilarity >= 0.65 && causalSimilarity >= 0.50 &&
					anchorSimilarity >= 0.50) {
				continue
			}
		}
		titleSimilarity := tokenDice(findingTokens(finding.Title), candidateTitle)
		bodySimilarity := tokenDice(
			findingTokens(finding.Title+"\n"+finding.Message+"\n"+finding.Evidence),
			candidateBody,
		)
		if titleSimilarity >= 0.65 && bodySimilarity >= 0.35 || bodySimilarity >= 0.72 {
			return index
		}
	}
	return -1
}

func moreSevere(left, right string) string {
	rank := map[string]int{"low": 1, "medium": 2, "high": 3, "critical": 4}
	if rank[right] > rank[left] {
		return right
	}
	return left
}

func findingObservationFrom(
	candidate FindingCandidate,
	contextID, model string,
	provenance ...string,
) FindingObservation {
	var modelAlias, account, reviewer string
	if len(provenance) >= 3 {
		modelAlias, account, reviewer = provenance[0], provenance[1], provenance[2]
	} else if len(provenance) > 0 {
		reviewer = provenance[0]
	}
	return FindingObservation{
		ContextID: contextID, Model: strings.TrimSpace(model),
		ModelAlias: strings.TrimSpace(modelAlias), Account: strings.TrimSpace(account),
		Reviewer: strings.TrimSpace(reviewer),
		Severity: candidate.Severity, Title: candidate.Title, Symbol: candidate.Symbol,
		Line: candidate.Line, Message: candidate.Message, Evidence: candidate.Evidence,
		Impact: candidate.Impact, Validation: candidate.Validation,
		MatchHints: candidate.MatchHints, FixEffort: candidate.FixEffort,
	}
}

func upsertFindingObservation(
	observations []FindingObservation,
	candidate FindingObservation,
) ([]FindingObservation, bool) {
	for index := range observations {
		if observations[index].ContextID == candidate.ContextID {
			observations[index] = candidate
			return observations, false
		}
	}
	if len(observations) >= 64 {
		copy(observations, observations[len(observations)-63:])
		observations = observations[:63]
	}
	return append(observations, candidate), true
}

func findingObservationContextIDs(observations []FindingObservation) []string {
	contexts := make([]string, 0, len(observations))
	for _, observation := range observations {
		contexts = appendUnique(contexts, observation.ContextID)
	}
	return contexts
}

func pruneUnreferencedFindingContexts(state *RepositoryState) {
	if state == nil {
		return
	}
	referenced := make(map[string]struct{})
	for _, finding := range state.Findings {
		for _, contextID := range finding.ContextIDs {
			referenced[contextID] = struct{}{}
		}
	}
	for _, raw := range state.RawFindings {
		if raw.ContextID != "" {
			referenced[raw.ContextID] = struct{}{}
		}
	}
	contexts := state.Contexts[:0]
	for _, contextRecord := range state.Contexts {
		if _, keep := referenced[contextRecord.ID]; keep {
			contexts = append(contexts, contextRecord)
		}
	}
	state.Contexts = contexts
}

func nearbyLines(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	delta := *left - *right
	if delta < 0 {
		delta = -delta
	}
	return delta <= 5
}

func findingTokens(value string) map[string]struct{} {
	stop := map[string]struct{}{
		"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {},
		"by": {}, "for": {}, "from": {}, "in": {}, "is": {}, "it": {}, "of": {},
		"on": {}, "or": {}, "that": {}, "the": {}, "this": {}, "to": {}, "with": {},
	}
	tokens := make(map[string]struct{})
	for _, token := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if len(token) > 4 && strings.HasSuffix(token, "ing") {
			token = token[:len(token)-3]
		} else if len(token) > 3 && strings.HasSuffix(token, "s") && !strings.HasSuffix(token, "ss") {
			token = token[:len(token)-1]
		}
		if _, ignored := stop[token]; ignored {
			continue
		}
		tokens[token] = struct{}{}
	}
	return tokens
}

func tokenDice(left, right map[string]struct{}) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	shared := 0
	for token := range left {
		if _, ok := right[token]; ok {
			shared++
		}
	}
	return float64(2*shared) / float64(len(left)+len(right))
}

func normalizedText(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(value)), " ")
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func stableID(prefix string, values ...string) string {
	hash := sha256.New()
	for _, value := range values {
		_, _ = hash.Write([]byte(value))
		_, _ = hash.Write([]byte{0})
	}
	return prefix + hex.EncodeToString(hash.Sum(nil))
}

func planDigest(plan Plan) string {
	digestPlan := plan
	digestPlan.ID = ""
	data, _ := json.Marshal(digestPlan)
	return stableID("rpl_", string(data))
}

func contextBindingDigest(context FindingContext) string {
	context.ID = ""
	context.CreatedAt = time.Time{}
	data, _ := json.Marshal(context)
	return stableID("sha256:", string(data))
}

func validateState(state RepositoryState) error {
	if state.SchemaVersion != SchemaVersion || strings.TrimSpace(state.Repository) == "" ||
		!validBoundedText(state.Repository, maxRepositoryIdentityBytes) ||
		state.ID != RepositoryID(state.Repository) || state.Version < 0 || state.ReviewVersion < 0 ||
		state.LastExcludedFiles < 0 || state.LastExcludedFiles > maxReviewFiles ||
		len(state.Files) > maxReviewFiles || len(state.Unsupported) > maxReviewFiles ||
		len(state.ReviewAttempts) > maxReviewFiles ||
		len(state.ReviewAttemptIdentities) > maxReviewFiles ||
		len(state.Contexts) > 1_000_000 ||
		len(state.Findings) > 100_000 || len(state.RawFindings) > 100_000 ||
		len(state.DeduplicatedFindings) > 100_000 || len(state.DeduplicationJobs) > 100_000 ||
		len(state.Runs) > 100_000 || len(state.IssueDrafts) > 100_000 {
		return errors.New("invalid repository review state")
	}
	if err := validateRepositoryReviewCampaignCoverage(state.CurrentCampaign); err != nil {
		return err
	}
	if err := validateRepositoryReviewCampaignHistory(state.CampaignHistory); err != nil {
		return err
	}
	if err := validateRepositoryReviewActiveRun(state); err != nil {
		return err
	}
	if err := validateRepositoryReviewFileAttributions(state.FileAttributions); err != nil {
		return err
	}
	if err := validateDeduplicationState(state); err != nil {
		return err
	}
	if err := validateHistoricalDeduplicationReplay(state); err != nil {
		return err
	}
	if state.CurrentCampaign != nil &&
		state.CampaignHistory[state.CurrentCampaign.ID] != state.CurrentCampaign.CommitSHA {
		return errors.New("current repository review campaign is absent from history")
	}
	activeForceFields := 0
	for _, value := range []string{
		state.ActiveForceCampaignID, state.ActiveForceProfileHash, state.ActiveForceCommitSHA,
	} {
		if value != "" {
			if !validBoundedText(value, 256) {
				return errors.New("invalid repository review force campaign")
			}
			activeForceFields++
		}
	}
	if activeForceFields != 0 && activeForceFields != 3 {
		return errors.New("invalid repository review force campaign")
	}
	for pathValue, attempts := range state.ReviewAttempts {
		if !validBoundedText(pathValue, 4096) || attempts < 0 {
			return errors.New("invalid repository review attempt state")
		}
	}
	for pathValue, identity := range state.ReviewAttemptIdentities {
		if !validBoundedText(pathValue, 4096) || !validBoundedText(identity, 128) {
			return errors.New("invalid repository review attempt identity")
		}
		if _, exists := state.ReviewAttempts[pathValue]; !exists {
			return errors.New("invalid repository review attempt identity")
		}
	}
	for pathValue, unsupported := range state.Unsupported {
		if pathValue != unsupported.Path || !validBoundedText(unsupported.Reason, 256) ||
			!validBlobSHA(unsupported.BlobSHA) || unsupported.SizeBytes < 0 {
			return errors.New("invalid repository review unsupported file state")
		}
	}
	for _, finding := range state.Findings {
		if len(finding.Observations) > 64 || len(finding.ContextIDs) > 64 ||
			(finding.CampaignID != "" &&
				(!ValidRepositoryReviewCampaignID(finding.CampaignID) ||
					state.CampaignHistory[finding.CampaignID] != finding.CommitSHA)) {
			return errors.New("invalid repository review finding observations")
		}
	}
	for _, contextRecord := range state.Contexts {
		if contextRecord.CampaignID != "" &&
			(!ValidRepositoryReviewCampaignID(contextRecord.CampaignID) ||
				state.CampaignHistory[contextRecord.CampaignID] != contextRecord.CommitSHA ||
				!validBoundedText(contextRecord.ProfileHash, 256)) {
			return errors.New("invalid repository review finding context campaign")
		}
	}
	for _, run := range state.Runs {
		if run.InspectedFiles < 0 || run.InspectedFiles > maxReviewFiles ||
			run.LegacyRecovered && run.CampaignID == "" ||
			(run.CampaignID != "" && (!ValidRepositoryReviewCampaignID(run.CampaignID) ||
				state.CampaignHistory[run.CampaignID] != run.CommitSHA ||
				!validBoundedText(run.ProfileHash, 256) ||
				!validRepositoryReviewCampaignScopeDigest(run.ScopeDigest) ||
				run.InspectedFiles > run.ReviewedFiles+run.UnreviewedFiles+run.UnsupportedCount)) {
			return errors.New("invalid repository review run campaign")
		}
		if len(run.CheckpointDigests) > maxRepositoryReviewRequiredAssignments {
			return errors.New("invalid repository review run checkpoint digests")
		}
		if len(run.CheckpointScopes) != len(run.CheckpointDigests) {
			return errors.New("invalid repository review run checkpoint scopes")
		}
		for assignmentID, files := range run.CheckpointScopes {
			canonical, err := canonicalRepositoryReviewCampaignFiles(files)
			if _, found := run.CheckpointDigests[assignmentID]; !found || err != nil ||
				len(canonical) == 0 || !reflect.DeepEqual(canonical, files) {
				return errors.New("invalid repository review run checkpoint scope")
			}
		}
		for assignmentID, digest := range run.CheckpointDigests {
			if !validBoundedText(assignmentID, 128) ||
				!validRepositoryReviewCheckpointDigest(digest) {
				return errors.New("invalid repository review run checkpoint digest")
			}
		}
	}
	if err := validateRepositoryReviewCampaignRecordBindings(state); err != nil {
		return err
	}
	if err := validateIssueAssociations(state); err != nil {
		return err
	}
	if err := validateRepositoryLifecycleState(state); err != nil {
		return err
	}
	return nil
}

func validBoundedText(value string, maximum int) bool {
	return value != "" && utf8.ValidString(value) && len(value) <= maximum && !strings.ContainsRune(value, 0)
}

func validFindingSourceProvenance(model, modelAlias, account string) bool {
	if !validBoundedText(model, 256) {
		return false
	}
	// Legacy observations predate exact alias/account capture. New provenance is
	// atomic so a partial source identity can never be mistaken for exact data.
	if modelAlias == "" && account == "" {
		return true
	}
	return validBoundedText(modelAlias, 256) && validBoundedText(account, 256)
}

func RepositoryID(repository string) string {
	return stableID("rrp_", strings.TrimSpace(repository))
}

func selectedFindings(all []Finding, requested []string) ([]Finding, []string, error) {
	if len(requested) == 0 || len(requested) > 200 {
		return nil, nil, errors.New("one to 200 finding IDs are required")
	}
	byID := make(map[string]Finding, len(all))
	for _, finding := range all {
		byID[finding.ID] = finding
	}
	selected := make([]Finding, 0, len(requested))
	ids := make([]string, 0, len(requested))
	seen := make(map[string]struct{})
	for _, id := range requested {
		id = strings.TrimSpace(id)
		if _, duplicate := seen[id]; duplicate {
			return nil, nil, errors.New("duplicate finding ID")
		}
		finding, ok := byID[id]
		if !ok || finding.DeduplicationPending {
			return nil, nil, os.ErrNotExist
		}
		seen[id] = struct{}{}
		selected, ids = append(selected, finding), append(ids, id)
	}
	return selected, ids, nil
}

func defaultIssueTitle(findings []Finding) string {
	if len(findings) == 1 {
		return truncateUTF8Bytes(findings[0].Title, 256)
	}
	return fmt.Sprintf("Repository review: %d validated bugs", len(findings))
}

func truncateUTF8Bytes(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if maximum < 1 || len(value) <= maximum {
		return value
	}
	value = value[:maximum]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return strings.TrimSpace(value)
}

func defaultIssueBody(state RepositoryState, findings []Finding) string {
	const maximumBodyBytes = maxIssueDraftBodyBytes
	const issueFieldBytes = 8 << 10
	var builder strings.Builder
	fmt.Fprintf(
		&builder,
		"## Repository review findings\n\nRepository: `%s`\nLatest reviewed commit: `%s`\n\n",
		state.Repository,
		state.LastCommitSHA,
	)
	for _, finding := range findings {
		section := strings.Builder{}
		fmt.Fprintf(
			&section,
			"### [%s] %s\n\n",
			strings.ToUpper(finding.Severity),
			truncateUTF8Bytes(finding.Title, 256),
		)
		fmt.Fprintf(&section, "Finding ID: `%s`\n\nLocation: `%s", finding.ID, finding.File.Path)
		if finding.Line != nil {
			fmt.Fprintf(&section, ":%d", *finding.Line)
		}
		fmt.Fprintf(
			&section,
			"` (commit `%s`, blob `%s`)\n\n%s\n\nImpact: %s\n\nValidation: %s\n\n",
			finding.CommitSHA,
			finding.File.BlobSHA,
			truncateUTF8Bytes(finding.Evidence, issueFieldBytes),
			truncateUTF8Bytes(finding.Impact, issueFieldBytes),
			truncateUTF8Bytes(finding.Validation.Summary, issueFieldBytes),
		)
		if builder.Len()+section.Len()+128 > maximumBodyBytes {
			fmt.Fprintf(
				&builder,
				"\n%d additional selected findings are retained in draft finding_ids but omitted here to keep the issue body bounded.\n",
				len(findings)-strings.Count(builder.String(), "Finding ID: `"),
			)
			break
		}
		builder.WriteString(section.String())
	}
	return strings.TrimSpace(builder.String())
}

func normalizeLabels(labels []string) []string {
	out := make([]string, 0, len(labels))
	seen := make(map[string]struct{})
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if !validBoundedText(label, 50) {
			continue
		}
		if _, duplicate := seen[label]; duplicate {
			continue
		}
		seen[label] = struct{}{}
		out = append(out, label)
		if len(out) == 20 {
			break
		}
	}
	return out
}
