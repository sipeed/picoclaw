package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/sipeed/picoclaw/pkg/collectionquery"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
	"github.com/sipeed/picoclaw/pkg/repoaudit"
)

type repositoryReviewAutomationConfigRequest struct {
	Name                     string                                 `json:"name"`
	Repository               string                                 `json:"repository"`
	ProfileID                string                                 `json:"profile_id,omitempty"`
	Branch                   string                                 `json:"branch,omitempty"`
	Ref                      string                                 `json:"ref,omitempty"`
	Target                   string                                 `json:"target"`
	ReviewFocus              string                                 `json:"review_focus"`
	AccountRef               string                                 `json:"account_ref,omitempty"`
	ScopePolicy              repoaudit.RepositoryReviewScopePolicy  `json:"scope_policy"`
	ReviewerModels           []string                               `json:"reviewer_models"`
	CompareModels            bool                                   `json:"compare_models"`
	Force                    bool                                   `json:"force"`
	AutoContinue             *bool                                  `json:"auto_continue,omitempty"`
	MaxFilesPerRun           int                                    `json:"max_files_per_run"`
	MaxContentBytes          int64                                  `json:"max_content_bytes"`
	MaxParallelChildren      int                                    `json:"max_parallel_children"`
	AssignmentTimeoutSeconds repositoryReviewOptionalInt            `json:"assignment_timeout_seconds"`
	Budget                   repoaudit.RepositoryReviewBudgetPolicy `json:"budget"`
	ExpectedVersion          int64                                  `json:"expected_version,omitempty"`
}

type repositoryReviewAutomationActionRequest struct {
	ExpectedVersion int64  `json:"expected_version"`
	CommitSHA       string `json:"commit_sha,omitempty"`
	RunID           string `json:"run_id,omitempty"`
}

type repositoryReviewPurgeRequest struct {
	ExpectedVersion           int64  `json:"expected_version"`
	ExpectedRepositoryVersion int64  `json:"expected_repository_version"`
	ExpectedLedgerFence       string `json:"expected_ledger_fence"`
	ConfirmRepository         string `json:"confirm_repository"`
}

type repositoryReviewCommitReference struct {
	SHA      string `json:"sha"`
	ShortSHA string `json:"short_sha"`
	URL      string `json:"url,omitempty"`
}

type repositoryReviewCommitOptionsResponse struct {
	ExpectedVersion      int64                           `json:"expected_version"`
	Remembered           repositoryReviewCommitReference `json:"remembered"`
	Latest               repositoryReviewCommitReference `json:"latest"`
	NewerCommitAvailable bool                            `json:"newer_commit_available"`
}

var repositoryReviewAutomationCollectionSchema = mustCollectionQuerySchema(
	[]collectionquery.FieldSchema{
		{Name: "id", Type: collectionquery.TypeString, Sortable: true},
		{Name: "name", Type: collectionquery.TypeString, Sortable: true},
		{Name: "repository", Type: collectionquery.TypeString, Sortable: true},
		{Name: "branch", Type: collectionquery.TypeString, Sortable: true},
		{
			Name: "status", Type: collectionquery.TypeEnum, Sortable: true,
			SuggestedValues: []string{"idle", "running", "stopping", "paused", "completed", "failed"},
		},
		{Name: "progress", Type: collectionquery.TypeNumber, Sortable: true},
		{Name: "reviewed", Type: collectionquery.TypeNumber, Sortable: true},
		{Name: "raw_findings", Type: collectionquery.TypeNumber, Sortable: true},
		{Name: "findings", Type: collectionquery.TypeNumber, Sortable: true},
		{Name: "updated", Type: collectionquery.TypeTimestamp, Sortable: true},
	},
	[]collectionquery.SortField{{Field: "updated", Direction: collectionquery.Descending}},
)

type repositoryReviewModelOption struct {
	Alias            string  `json:"alias"`
	ResolvedModel    string  `json:"resolved_model"`
	Provider         string  `json:"provider,omitempty"`
	Available        bool    `json:"available"`
	BlockedReason    string  `json:"blocked_reason,omitempty"`
	PriceKnown       bool    `json:"price_known"`
	InputPricePer1M  float64 `json:"input_price_per_1m,omitempty"`
	OutputPricePer1M float64 `json:"output_price_per_1m,omitempty"`
	Subscription     bool    `json:"subscription,omitempty"`
	EquivalentModel  string  `json:"equivalent_model,omitempty"`
	Default          bool    `json:"default,omitempty"`
}

type repositoryReviewAccountOption struct {
	ID           string                               `json:"id"`
	Provider     string                               `json:"provider,omitempty"`
	Label        string                               `json:"label"`
	Status       string                               `json:"status"`
	Available    bool                                 `json:"available"`
	Default      bool                                 `json:"default,omitempty"`
	Models       []string                             `json:"models"`
	WriterModels []string                             `json:"writer_models"`
	Entries      []repositoryReviewAccountLimitOption `json:"entries"`
}

type repositoryReviewAccountLimitOption struct {
	Name             string   `json:"name"`
	Status           string   `json:"status"`
	Window           string   `json:"window,omitempty"`
	UsedPercent      *int     `json:"used_percent,omitempty"`
	RemainingPercent *float64 `json:"remaining_percent,omitempty"`
	RefreshesAt      string   `json:"refreshes_at,omitempty"`
}

func (h *Handler) registerRepositoryReviewAutomationRoutes(mux *http.ServeMux) {
	h.registerRepositoryReviewProfileRoutes(mux)
	mux.HandleFunc("GET /api/repository-reviews/automations", h.handleListRepositoryReviewAutomations)
	mux.HandleFunc("POST /api/repository-reviews/automations", h.handleCreateRepositoryReviewAutomation)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}",
		h.handleGetRepositoryReviewAutomation,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/file-attributions",
		h.handleListRepositoryReviewFileAttributionsCollection,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/finding-health",
		h.handleGetRepositoryReviewFindingHealth,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/findings",
		h.handleGetRepositoryReviewAutomationReport,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/run-findings",
		h.handleListRepositoryReviewRunFindingsCollection,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/run-findings/{finding_id}",
		h.handleGetRepositoryReviewRunFinding,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/findings/status",
		h.handleRetryRepositoryReviewRunFindingStatus,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/report",
		h.handleGetRepositoryReviewAutomationReport,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/findings/{finding_id}",
		h.handleGetRepositoryReviewDeduplicatedFinding,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/findings/{finding_id}/sources",
		h.handleListRepositoryReviewRawSources,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/findings/{finding_id}/sources/{source_id}",
		h.handleGetRepositoryReviewRawSource,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/raw-findings",
		h.handleListRepositoryReviewRawFindingsCollection,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/raw-findings/{source_id}",
		h.handleGetRepositoryReviewRawSource,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/raw-findings/{source_id}/retry",
		h.handleRetryRepositoryReviewRawSource,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/findings-processing",
		h.handleListRepositoryReviewFindingsProcessing,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/findings-processing/retry",
		h.handleRetryRepositoryReviewProcessingSources,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/findings-processing/sources/{source_id}",
		h.handleGetRepositoryReviewProcessingSource,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/findings-processing/sources/{source_id}/retry",
		h.handleRetryRepositoryReviewProcessingSource,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/campaigns/{campaign_id}/findings-processing",
		h.handleGetRepositoryReviewFindingsProcessing,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/campaigns/{campaign_id}/findings-processing/sources/{source_id}",
		h.handleGetRepositoryReviewRawSource,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/campaigns/{campaign_id}/findings-processing/sources/{source_id}/retry",
		h.handleRetryRepositoryReviewRawSource,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/historical-deduplication",
		h.handleGetRepositoryReviewHistoricalDeduplication,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/historical-deduplication/retry",
		h.handleRetryRepositoryReviewHistoricalDeduplication,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/historical-deduplication/restart",
		h.handleRestartRepositoryReviewHistoricalDeduplication,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/repository-findings",
		h.handleListRepositoryReviewRepositoryFindingsCollection,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/repository-findings/{finding_id}",
		h.handleGetRepositoryReviewAutomationRepositoryFinding,
	)
	mux.HandleFunc(
		"PATCH /api/repository-reviews/automations/{automation_id}/findings/{finding_id}",
		h.handleUpdateRepositoryReviewAutomationFinding,
	)
	mux.HandleFunc(
		"PATCH /api/repository-reviews/automations/{automation_id}/repository-findings/{repository_finding_id}",
		h.handleUpdateRepositoryReviewFindingLifecycle,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/repository-findings/{repository_finding_id}/duplicates",
		h.handleResolveRepositoryReviewPossibleDuplicate,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/repository-findings/validations",
		h.handleReserveRepositoryReviewValidations,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/repository-findings/{repository_finding_id}/sync",
		h.handleSyncRepositoryReviewFinding,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/issues",
		h.handleListRepositoryReviewAutomationIssues,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/issues/generations",
		h.handleGenerateRepositoryReviewAutomationIssues,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/issues/{draft_id}",
		h.handleGetRepositoryReviewAutomationIssue,
	)
	mux.HandleFunc(
		"PATCH /api/repository-reviews/automations/{automation_id}/issues/{draft_id}",
		h.handleUpdateRepositoryReviewAutomationIssue,
	)
	mux.HandleFunc(
		"DELETE /api/repository-reviews/automations/{automation_id}/issues/{draft_id}",
		h.handleDeleteRepositoryReviewAutomationIssue,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/issues/{draft_id}/regenerate",
		h.handleRegenerateRepositoryReviewAutomationIssue,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/issues/{draft_id}/publish",
		h.handlePublishRepositoryReviewAutomationIssue,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/issues/publish",
		h.handlePublishRepositoryReviewAutomationIssues,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/findings/{finding_id}/issue-link/candidates",
		h.handleRepositoryReviewIssueLinkCandidates,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/findings/{finding_id}/issue-link",
		h.handleRepositoryReviewIssueLink,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/findings/{finding_id}/post",
		h.handlePostRepositoryReviewAutomationFinding,
	)
	mux.HandleFunc(
		"DELETE /api/repository-reviews/automations/{automation_id}/findings/{finding_id}/issue-link",
		h.handleRepositoryReviewIssueLink,
	)
	mux.HandleFunc(
		"PATCH /api/repository-reviews/automations/{automation_id}",
		h.handleUpdateRepositoryReviewAutomation,
	)
	mux.HandleFunc(
		"DELETE /api/repository-reviews/automations/{automation_id}",
		h.handleDeleteRepositoryReviewAutomation,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/purge-history",
		h.handlePurgeRepositoryReviewAutomationHistory,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/start",
		h.handleStartRepositoryReviewAutomation,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/pause",
		h.handlePauseRepositoryReviewAutomation,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/resume",
		h.handleResumeRepositoryReviewAutomation,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/automations/{automation_id}/restart",
		h.handleRestartRepositoryReviewAutomation,
	)
	mux.HandleFunc(
		"GET /api/repository-reviews/automations/{automation_id}/commit-options",
		h.handleRepositoryReviewAutomationCommitOptions,
	)
	mux.HandleFunc("GET /api/repository-reviews/automation-options", h.handleRepositoryReviewAutomationOptions)
}

func (h *Handler) handleListRepositoryReviewAutomations(w http.ResponseWriter, r *http.Request) {
	store, err := h.repositoryReviewStore()
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	automations, err := store.ListAutomations(r.Context())
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	for index := range automations {
		state, found, resolveErr := store.ResolveRepositoryState(
			automations[index].Repository,
			automations[index].RunIDs,
		)
		if resolveErr != nil {
			writeRepositoryReviewAutomationError(w, resolveErr)
			return
		}
		if found {
			applyRepositoryReviewLiveMetrics(&automations[index], state)
			automations[index].Progress.AssignmentProgress = repoaudit.CurrentCampaignAssignmentProgress(
				state,
				automations[index].CampaignID,
			)
		}
	}
	query, _ := collectionquery.Parse("", repositoryReviewAutomationCollectionSchema)
	var projected []repoaudit.RepositoryReviewAutomation
	total := len(automations)
	nextCursor := ""
	if r.URL != nil && r.URL.RawQuery != "" {
		listRequest, ok := parseCollectionListRequest(w, r, repositoryReviewAutomationCollectionSchema)
		if !ok {
			return
		}
		query = listRequest.Query
		page, pageErr := collectionquery.Paginate(
			automations,
			query,
			listRequest.Cursor,
			listRequest.Limit,
			listRequest.Now,
			collectionquery.PageOptions[repoaudit.RepositoryReviewAutomation]{
				ID: func(automation repoaudit.RepositoryReviewAutomation) (string, error) {
					return automation.ID, nil
				},
				Clone: projectRepositoryReviewAutomation,
				Resolve: func(
					automation repoaudit.RepositoryReviewAutomation,
					field collectionquery.Field,
					_ time.Time,
				) (collectionquery.FieldValue, bool) {
					return repositoryReviewAutomationCollectionField(automation, field)
				},
			},
		)
		if pageErr != nil {
			writeCollectionPageError(w, pageErr)
			return
		}
		projected = page.Items
		total = page.Total
		nextCursor = page.NextCursor
	} else {
		projected = make([]repoaudit.RepositoryReviewAutomation, len(automations))
		for index := range automations {
			projected[index] = projectRepositoryReviewAutomation(automations[index])
		}
	}
	repositories := make([]string, 0, len(automations))
	branches := make([]string, 0, len(automations))
	names := make([]string, 0, len(automations))
	for _, automation := range automations {
		repositories = append(repositories, automation.Repository)
		branches = append(branches, automation.Ref)
		names = append(names, automation.Name)
	}
	writeRepositoryReviewJSON(w, http.StatusOK, map[string]any{
		"automations":     projected,
		"total":           total,
		"next_cursor":     nextCursor,
		"canonical_query": query.Canonical(),
		"query_schema": collectionSchemaWithSuggestions(
			repositoryReviewAutomationCollectionSchema,
			map[collectionquery.Field][]string{
				"name": names, "repository": repositories, "branch": branches,
			},
		),
	})
}

func repositoryReviewAutomationCollectionField(
	automation repoaudit.RepositoryReviewAutomation,
	field collectionquery.Field,
) (collectionquery.FieldValue, bool) {
	switch field {
	case "id":
		return collectionquery.StringValue(automation.ID), true
	case "name":
		return collectionquery.StringValue(automation.Name), true
	case "repository":
		return collectionquery.StringValue(automation.Repository), true
	case "branch":
		return collectionquery.StringValue(automation.Ref), true
	case "status":
		return collectionquery.EnumValue(string(automation.Status)), true
	case "progress":
		progress := repoaudit.RepositoryReviewAutomationFileProgress(automation)
		return collectionquery.NumberValue(progress.Percent), true
	case "reviewed":
		return collectionquery.NumberValue(float64(automation.Progress.ReviewedFiles)), true
	case "raw_findings":
		return collectionquery.NumberValue(float64(automation.Progress.RawFindings)), true
	case "findings":
		return collectionquery.NumberValue(float64(automation.Progress.Findings)), true
	case "updated":
		return collectionquery.TimestampValue(automation.UpdatedAt), true
	default:
		return collectionquery.FieldValue{}, false
	}
}

func (h *Handler) handleCreateRepositoryReviewAutomation(w http.ResponseWriter, r *http.Request) {
	if err := validateRepositoryReviewMutation(r); err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	var request repositoryReviewAutomationConfigRequest
	if err := decodeRepositoryReviewRequest(r, &request); err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	if !validRepositoryReviewAssignmentTimeoutRequest(request.AssignmentTimeoutSeconds) {
		writeRepositoryReviewAutomationError(w, repoaudit.ErrInvalidAutomation)
		return
	}
	if strings.TrimSpace(request.ProfileID) == "" {
		writeRepositoryReviewAutomationError(
			w,
			fmt.Errorf("%w: profile_id is required", repoaudit.ErrInvalidAutomation),
		)
		return
	}
	store, err := h.repositoryReviewStore()
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	automation, err := materializeRepositoryReviewAutomationRequest(
		r.Context(), store, repositoryReviewAutomationFromRequest(request), request,
	)
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	if pricingErr := h.refreshRepositoryReviewAccountingSnapshot(&automation); pricingErr != nil {
		writeRepositoryReviewAutomationError(w, pricingErr)
		return
	}
	created, err := store.CreateAutomation(r.Context(), automation)
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	writeRepositoryReviewJSON(
		w,
		http.StatusCreated,
		map[string]any{"automation": projectRepositoryReviewAutomationWithStore(store, created)},
	)
}

func (h *Handler) handleUpdateRepositoryReviewAutomation(w http.ResponseWriter, r *http.Request) {
	if err := validateRepositoryReviewMutation(r); err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	var request repositoryReviewAutomationConfigRequest
	if err := decodeRepositoryReviewRequest(r, &request); err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	if !validRepositoryReviewAssignmentTimeoutRequest(request.AssignmentTimeoutSeconds) {
		writeRepositoryReviewAutomationError(w, repoaudit.ErrInvalidAutomation)
		return
	}
	store, err := h.repositoryReviewStore()
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	prepared := repositoryReviewAutomationFromRequest(request)
	prepared, err = materializeRepositoryReviewAutomationRequest(r.Context(), store, prepared, request)
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	if pricingErr := h.refreshRepositoryReviewAccountingSnapshot(&prepared); pricingErr != nil {
		writeRepositoryReviewAutomationError(w, pricingErr)
		return
	}
	updated, err := store.UpdateAutomation(
		r.Context(), r.PathValue("automation_id"), request.ExpectedVersion,
		func(candidate *repoaudit.RepositoryReviewAutomation) error {
			if candidate.Status == repoaudit.RepositoryReviewAutomationRunning ||
				candidate.Status == repoaudit.RepositoryReviewAutomationStopping {
				return errRepositoryReviewAutomationBusy
			}
			previous := *candidate
			if prepared.ProfileID != "" {
				applyRepositoryReviewMaterializedPolicy(candidate, prepared)
			} else {
				if candidate.ProfileID != "" {
					return fmt.Errorf(
						"%w: profile_id is required when updating an assigned repository",
						repoaudit.ErrInvalidAutomation,
					)
				}
				applyRepositoryReviewAutomationRequest(candidate, request)
				candidate.Ref = prepared.Ref
			}
			executionChanged := repositoryReviewExecutionConfigurationChanged(previous, *candidate)
			if executionChanged {
				candidate.Status = repoaudit.RepositoryReviewAutomationIdle
				candidate.CampaignID = ""
				candidate.CampaignRecoveryPending = false
				candidate.ScopePlan = repoaudit.RepositoryReviewScopePlan{}
				candidate.ScopeSelection = nil
				candidate.ResolvedCommitSHA = ""
				candidate.ResolvedTargetBranch = ""
				candidate.AdvertisedDefaultBranch = ""
				candidate.TargetIsDefault = candidate.Ref == ""
				candidate.PauseReason = ""
				candidate.PauseDetail = ""
				candidate.RequestedPauseReason = ""
				candidate.RequestedPauseDetail = ""
				candidate.Progress = repoaudit.RepositoryReviewProgress{}
				candidate.Usage = repoaudit.RepositoryReviewTokenUsage{}
				candidate.EstimatedCostUSD = 0
				candidate.ModelStats = make(map[string]repoaudit.RepositoryReviewModelStats)
				candidate.ModelCoverageSketches = make(map[string]string)
				candidate.EffectiveAccountRef = ""
				candidate.StartedAt = time.Time{}
				candidate.CompletedAt = time.Time{}
				candidate.AccountLimitSnapshots = nil
			} else {
				if !previous.StartedAt.IsZero() {
					candidate.ModelPrices = maps.Clone(previous.ModelPrices)
				}
				repositoryReviewClearStaleAccountLimits(candidate, previous)
			}
			return nil
		},
	)
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	writeRepositoryReviewJSON(
		w,
		http.StatusOK,
		map[string]any{"automation": projectRepositoryReviewAutomationWithStore(store, updated)},
	)
}

func (h *Handler) handleDeleteRepositoryReviewAutomation(w http.ResponseWriter, r *http.Request) {
	if err := validateRepositoryReviewMutation(r); err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	var request repositoryReviewPurgeRequest
	if err := decodeRepositoryReviewRequest(r, &request); err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	controller := h.repositoryReviewControllerInstance()
	if controller == nil {
		writeRepositoryReviewAutomationError(w, errors.New("repository review controller unavailable"))
		return
	}
	if err := controller.Start(); err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	eligibility, err := controller.leasedStore.DeleteAutomationAndHistory(
		r.Context(), r.PathValue("automation_id"), request.ExpectedVersion,
		request.ExpectedRepositoryVersion, request.ExpectedLedgerFence,
		request.ConfirmRepository,
	)
	if errors.Is(err, repoaudit.ErrRepositoryReviewPurgeBlocked) {
		writeRepositoryReviewPurgeBlocked(w, eligibility)
		return
	}
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handlePurgeRepositoryReviewAutomationHistory(w http.ResponseWriter, r *http.Request) {
	if err := validateRepositoryReviewMutation(r); err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	var request repositoryReviewPurgeRequest
	if err := decodeRepositoryReviewRequest(r, &request); err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	controller := h.repositoryReviewControllerInstance()
	if controller == nil {
		writeRepositoryReviewAutomationError(w, errors.New("repository review controller unavailable"))
		return
	}
	if err := controller.Start(); err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	updated, eligibility, err := controller.leasedStore.PurgeAutomationHistory(
		r.Context(), r.PathValue("automation_id"), request.ExpectedVersion,
		request.ExpectedRepositoryVersion, request.ExpectedLedgerFence,
		request.ConfirmRepository,
	)
	if errors.Is(err, repoaudit.ErrRepositoryReviewPurgeBlocked) {
		writeRepositoryReviewPurgeBlocked(w, eligibility)
		return
	}
	if errors.Is(err, repoaudit.ErrRepositoryReviewHistoryAbsent) {
		writeRepositoryReviewJSON(w, http.StatusNotFound, map[string]any{
			"code":    "repository_review_history_not_found",
			"message": "repository review history not found",
		})
		return
	}
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	writeRepositoryReviewJSON(w, http.StatusOK, map[string]any{
		"automation": projectRepositoryReviewAutomation(updated),
		"outcome":    "history_purged",
	})
}

func writeRepositoryReviewPurgeBlocked(
	w http.ResponseWriter,
	eligibility repoaudit.RepositoryReviewPurgeEligibility,
) {
	writeRepositoryReviewJSON(w, http.StatusConflict, map[string]any{
		"code":           "repository_review_purge_blocked",
		"message":        "repository review history cannot be deleted while related work is active",
		"purge_blockers": eligibility.Blockers,
	})
}

func (h *Handler) handleStartRepositoryReviewAutomation(w http.ResponseWriter, r *http.Request) {
	h.handleRepositoryReviewAutomationStartAction(w, r, false, false)
}

func (h *Handler) handleResumeRepositoryReviewAutomation(w http.ResponseWriter, r *http.Request) {
	h.handleRepositoryReviewAutomationStartAction(w, r, true, false)
}

func (h *Handler) handleRestartRepositoryReviewAutomation(w http.ResponseWriter, r *http.Request) {
	h.handleRepositoryReviewAutomationStartAction(w, r, true, true)
}

func (h *Handler) handleRepositoryReviewAutomationStartAction(
	w http.ResponseWriter,
	r *http.Request,
	resume bool,
	restart bool,
) {
	if err := validateRepositoryReviewMutation(r); err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	var request repositoryReviewAutomationActionRequest
	if err := decodeRepositoryReviewRequest(r, &request); err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	reset := false
	action := "start"
	if restart {
		action = "restart"
	} else if resume {
		action = "resume"
	}
	controller := h.repositoryReviewControllerInstance()
	if controller == nil {
		writeRepositoryReviewAutomationError(w, errors.New("repository review controller unavailable"))
		return
	}
	automation, err := controller.startAutomationAtCommit(
		r.Context(), r.PathValue("automation_id"), request.ExpectedVersion, reset, action,
		request.CommitSHA,
	)
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	projected := projectRepositoryReviewAutomation(automation)
	if store, storeErr := h.repositoryReviewStore(); storeErr == nil {
		projected = projectRepositoryReviewAutomationWithStore(store, automation)
	}
	writeRepositoryReviewJSON(w, http.StatusAccepted, map[string]any{
		"automation": projected,
		"outcome":    "started",
	})
}

func (h *Handler) handleRepositoryReviewAutomationCommitOptions(
	w http.ResponseWriter,
	r *http.Request,
) {
	controller := h.repositoryReviewControllerInstance()
	if controller == nil {
		writeRepositoryReviewAutomationError(w, errors.New("repository review controller unavailable"))
		return
	}
	automation, remembered, latest, err := controller.repositoryReviewCommitOptions(
		r.Context(),
		r.PathValue("automation_id"),
	)
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	writeRepositoryReviewJSON(w, http.StatusOK, repositoryReviewCommitOptionsResponse{
		ExpectedVersion:      automation.Version,
		Remembered:           repositoryReviewCommitReferenceForAutomation(automation, remembered),
		Latest:               repositoryReviewCommitReferenceForAutomation(automation, latest),
		NewerCommitAvailable: remembered != latest,
	})
}

func (h *Handler) handlePauseRepositoryReviewAutomation(w http.ResponseWriter, r *http.Request) {
	if err := validateRepositoryReviewMutation(r); err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	var request repositoryReviewAutomationActionRequest
	if err := decodeRepositoryReviewRequest(r, &request); err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	controller := h.repositoryReviewControllerInstance()
	if controller == nil {
		writeRepositoryReviewAutomationError(w, errors.New("repository review controller unavailable"))
		return
	}
	automation, err := controller.pauseAutomationForRun(
		r.Context(), r.PathValue("automation_id"), request.ExpectedVersion, request.RunID,
	)
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	projected := projectRepositoryReviewAutomation(automation)
	if store, storeErr := h.repositoryReviewStore(); storeErr == nil {
		projected = projectRepositoryReviewAutomationWithStore(store, automation)
	}
	writeRepositoryReviewJSON(w, http.StatusAccepted, map[string]any{
		"automation": projected,
	})
}

func projectRepositoryReviewAutomation(
	automation repoaudit.RepositoryReviewAutomation,
) repoaudit.RepositoryReviewAutomation {
	automation.CampaignID = ""
	automation.CampaignRecoveryPending = false
	automation.ModelCoverageSketches = nil
	automation.Progress.ScopeFrozen = automation.ScopeSelection != nil
	automation.ScopeSelection = nil
	return automation
}

func projectRepositoryReviewAutomationWithStore(
	store repoaudit.Store,
	automation repoaudit.RepositoryReviewAutomation,
) repoaudit.RepositoryReviewAutomation {
	if state, found, err := store.ResolveRepositoryState(
		automation.Repository, automation.RunIDs,
	); err == nil && found {
		applyRepositoryReviewLiveMetrics(&automation, state)
		automation.Progress.AssignmentProgress = repoaudit.CurrentCampaignAssignmentProgress(
			state,
			automation.CampaignID,
		)
	}
	return projectRepositoryReviewAutomation(automation)
}

func repositoryReviewCommitReferenceForAutomation(
	automation repoaudit.RepositoryReviewAutomation,
	commit string,
) repositoryReviewCommitReference {
	commit = strings.ToLower(strings.TrimSpace(commit))
	short := commit
	if len(short) > 8 {
		short = short[:8]
	}
	return repositoryReviewCommitReference{
		SHA:      commit,
		ShortSHA: short,
		URL:      repositoryReviewGitHubCommitURL(automation.Repository, commit),
	}
}

func repositoryReviewGitHubCommitURL(repository string, commit string) string {
	commit = strings.ToLower(strings.TrimSpace(commit))
	if !repositoryReviewValidCommitSHA(commit) {
		return ""
	}
	repository = strings.TrimSpace(repository)
	host := ""
	repositoryPath := ""
	if owner, name, found := strings.Cut(repository, "/"); found &&
		!strings.Contains(name, "/") && validRepositoryReviewGitHubSegment(owner) &&
		validRepositoryReviewGitHubSegment(strings.TrimSuffix(name, ".git")) {
		host = "github.com"
		repositoryPath = owner + "/" + name
	} else if parsed, err := url.Parse(repository); err == nil && parsed.Scheme != "" {
		host = parsed.Hostname()
		repositoryPath = parsed.Path
	} else if identity, remotePath, found := strings.Cut(repository, ":"); found {
		_, host, _ = strings.Cut(identity, "@")
		repositoryPath = remotePath
	}
	if !strings.EqualFold(strings.TrimSpace(host), "github.com") {
		return ""
	}
	components := strings.Split(strings.Trim(strings.TrimSpace(repositoryPath), "/"), "/")
	if len(components) != 2 || components[0] == "" || components[1] == "" {
		return ""
	}
	name := strings.TrimSuffix(components[1], ".git")
	if name == "" {
		return ""
	}
	return (&url.URL{
		Scheme: "https",
		Host:   "github.com",
		Path:   "/" + components[0] + "/" + name + "/commit/" + commit,
	}).String()
}

func repositoryReviewAutomationFromRequest(
	request repositoryReviewAutomationConfigRequest,
) repoaudit.RepositoryReviewAutomation {
	automation := repoaudit.RepositoryReviewAutomation{
		Name: request.Name, Repository: request.Repository, Ref: request.Ref,
		Target: request.Target, ReviewFocus: request.ReviewFocus, AccountRef: request.AccountRef,
		ScopePolicy:    request.ScopePolicy,
		ReviewerModels: request.ReviewerModels, CompareModels: request.CompareModels,
		Force:          request.Force,
		MaxFilesPerRun: request.MaxFilesPerRun, MaxContentBytes: request.MaxContentBytes,
		MaxParallelChildren:   request.MaxParallelChildren,
		EstimatedOutputTokens: 1_800,
		BudgetPolicy:          request.Budget,
		Status:                repoaudit.RepositoryReviewAutomationIdle,
	}
	if request.AssignmentTimeoutSeconds.Present {
		automation.AssignmentTimeoutSeconds = request.AssignmentTimeoutSeconds.Value
	}
	if request.AutoContinue == nil {
		automation.AutoContinue = true
	} else {
		automation.AutoContinue = *request.AutoContinue
	}
	return automation
}

func materializeRepositoryReviewAutomationRequest(
	ctx context.Context,
	store repoaudit.Store,
	automation repoaudit.RepositoryReviewAutomation,
	request repositoryReviewAutomationConfigRequest,
) (repoaudit.RepositoryReviewAutomation, error) {
	repository, err := normalizeRepositoryReviewAutomationRepository(automation.Repository)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	automation.Repository = repository
	branch, err := repositoryReviewBranchFromRequest(request)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	automation.Ref = branch
	profileID := strings.TrimSpace(request.ProfileID)
	if profileID == "" {
		return automation, nil
	}
	profile, found, err := store.GetProfile(ctx, profileID)
	if err != nil {
		return repoaudit.RepositoryReviewAutomation{}, err
	}
	if !found {
		return repoaudit.RepositoryReviewAutomation{}, fmt.Errorf("repository review profile %q not found", profileID)
	}
	automation.ProfileID = profile.ID
	automation.ProfileVersion = profile.Version
	automation.Name = repositoryReviewAssignedAutomationName(automation.Repository, profile.Name)
	return repoaudit.MaterializeRepositoryReviewAutomation(profile, automation)
}

func normalizeRepositoryReviewAutomationRepository(repository string) (string, error) {
	if repository == "" || repository != strings.TrimSpace(repository) {
		return "", fmt.Errorf("%w: invalid repository", repoaudit.ErrInvalidAutomation)
	}
	if filepath.IsAbs(repository) {
		return filepath.Clean(repository), nil
	}
	if !strings.Contains(repository, "://") {
		if strings.Contains(repository, "@") && strings.Contains(repository, ":") {
			return normalizeRepositoryReviewSCPRepository(repository)
		}
		owner, name, ok := strings.Cut(repository, "/")
		if !ok || strings.Contains(name, "/") ||
			!validRepositoryReviewGitHubSegment(owner) || !validRepositoryReviewGitHubSegment(name) {
			return "", fmt.Errorf(
				"%w: repository must be an owner/repository shorthand, safe URL, or absolute local path",
				repoaudit.ErrInvalidAutomation,
			)
		}
		if strings.HasSuffix(strings.ToLower(name), ".git") {
			name = name[:len(name)-len(".git")]
		}
		if !validRepositoryReviewGitHubSegment(name) {
			return "", fmt.Errorf("%w: invalid GitHub repository shorthand", repoaudit.ErrInvalidAutomation)
		}
		return "https://github.com/" + owner + "/" + name + ".git", nil
	}
	parsed, err := url.Parse(repository)
	if err != nil {
		return "", fmt.Errorf("%w: invalid repository URL", repoaudit.ErrInvalidAutomation)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if parsed.Host == "" ||
		(scheme != "https" && scheme != "http" && scheme != "ssh" && scheme != "git") ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("%w: invalid repository URL", repoaudit.ErrInvalidAutomation)
	}
	if host := parsed.Hostname(); net.ParseIP(host) == nil && !validRepositoryReviewRemoteHost(host) {
		return "", fmt.Errorf("%w: invalid repository URL host", repoaudit.ErrInvalidAutomation)
	}
	port := parsed.Port()
	switch scheme {
	case "http":
		if port != "" && port != "80" {
			return "", fmt.Errorf("%w: invalid repository URL port", repoaudit.ErrInvalidAutomation)
		}
	case "https":
		if port != "" && port != "443" {
			return "", fmt.Errorf("%w: invalid repository URL port", repoaudit.ErrInvalidAutomation)
		}
	case "ssh":
		if port != "" && port != "22" {
			return "", fmt.Errorf("%w: invalid repository URL port", repoaudit.ErrInvalidAutomation)
		}
	case "git":
		if port != "" {
			return "", fmt.Errorf("%w: invalid repository URL port", repoaudit.ErrInvalidAutomation)
		}
	}
	if parsed.User != nil {
		_, hasPassword := parsed.User.Password()
		if scheme != "ssh" || parsed.User.Username() == "" || hasPassword {
			return "", fmt.Errorf("%w: repository URL credentials are not allowed", repoaudit.ErrInvalidAutomation)
		}
	}
	parsed.Scheme = scheme
	normalizedHost := strings.ToLower(parsed.Hostname())
	if strings.Contains(normalizedHost, ":") {
		parsed.Host = "[" + normalizedHost + "]"
	} else {
		parsed.Host = normalizedHost
	}
	normalizedPath, err := normalizeRepositoryReviewRemotePath(parsed.Hostname(), parsed.Path)
	if err != nil {
		return "", err
	}
	parsed.Path = "/" + normalizedPath
	parsed.RawPath = ""
	return parsed.String(), nil
}

func normalizeRepositoryReviewSCPRepository(repository string) (string, error) {
	identity, repositoryPath, hasPath := strings.Cut(repository, ":")
	user, host, hasHost := strings.Cut(identity, "@")
	if !hasPath || !hasHost || user != "git" || !validRepositoryReviewRemoteHost(host) {
		return "", fmt.Errorf("%w: invalid SCP repository", repoaudit.ErrInvalidAutomation)
	}
	normalizedPath, err := normalizeRepositoryReviewRemotePath(host, repositoryPath)
	if err != nil {
		return "", err
	}
	return "git@" + strings.ToLower(host) + ":" + normalizedPath, nil
}

func normalizeRepositoryReviewRemotePath(host, repositoryPath string) (string, error) {
	if repositoryPath == "" || strings.HasPrefix(repositoryPath, "~") ||
		strings.ContainsAny(repositoryPath, "\\?#") {
		return "", fmt.Errorf("%w: invalid repository remote path", repoaudit.ErrInvalidAutomation)
	}
	components := strings.Split(strings.Trim(repositoryPath, "/"), "/")
	if len(components) < 2 {
		return "", fmt.Errorf(
			"%w: repository remote path requires an owner and repository",
			repoaudit.ErrInvalidAutomation,
		)
	}
	for _, component := range components {
		if component == "" || component == "." || component == ".." ||
			strings.IndexFunc(component, func(character rune) bool {
				return unicode.IsSpace(character) || unicode.IsControl(character)
			}) >= 0 {
			return "", fmt.Errorf("%w: invalid repository remote path", repoaudit.ErrInvalidAutomation)
		}
	}
	if strings.EqualFold(strings.TrimSpace(host), "github.com") && len(components) != 2 {
		return "", fmt.Errorf("%w: GitHub repositories require owner/repository", repoaudit.ErrInvalidAutomation)
	}
	normalized := pathpkg.Join(components...)
	if strings.HasSuffix(strings.ToLower(normalized), ".git") {
		return normalized[:len(normalized)-len(".git")] + ".git", nil
	}
	return normalized + ".git", nil
}

func validRepositoryReviewRemoteHost(host string) bool {
	if host == "" || host != strings.TrimSpace(host) || strings.HasPrefix(host, ".") ||
		strings.HasSuffix(host, ".") {
		return false
	}
	for _, character := range host {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' || character == '-' || character == '.' {
			continue
		}
		return false
	}
	return true
}

func validRepositoryReviewGitHubSegment(value string) bool {
	if value == "" || value == "." || value == ".." {
		return false
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' || character == '-' || character == '_' || character == '.' {
			continue
		}
		return false
	}
	return true
}

func repositoryReviewBranchFromRequest(
	request repositoryReviewAutomationConfigRequest,
) (string, error) {
	branch := request.Branch
	legacy := request.Ref
	if branch != "" && legacy != "" {
		normalizedBranch, err := repoaudit.NormalizeRepositoryReviewBranch(branch)
		if err != nil {
			return "", err
		}
		normalizedLegacy, err := repoaudit.NormalizeRepositoryReviewBranch(legacy)
		if err != nil {
			return "", err
		}
		if normalizedBranch != normalizedLegacy {
			return "", fmt.Errorf("invalid repository review branch: branch and legacy ref disagree")
		}
		return normalizedBranch, nil
	}
	if branch == "" {
		branch = legacy
	}
	return repoaudit.NormalizeRepositoryReviewBranch(branch)
}

func repositoryReviewAssignedAutomationName(repository, profileName string) string {
	repository = strings.TrimSpace(repository)
	profileName = strings.TrimSpace(profileName)
	if repository == "" {
		return profileName
	}
	if profileName == "" {
		return repository
	}
	return repository + " · " + profileName
}

func applyRepositoryReviewMaterializedPolicy(
	candidate *repoaudit.RepositoryReviewAutomation,
	materialized repoaudit.RepositoryReviewAutomation,
) {
	if candidate == nil {
		return
	}
	candidate.ProfileID = materialized.ProfileID
	candidate.ProfileVersion = materialized.ProfileVersion
	candidate.AccountRef = materialized.AccountRef
	candidate.Name = materialized.Name
	candidate.Repository = materialized.Repository
	candidate.Ref = materialized.Ref
	candidate.Target = "all"
	candidate.ReviewFocus = materialized.ReviewFocus
	candidate.ScopePolicy = materialized.ScopePolicy
	candidate.ReviewerModels = append([]string(nil), materialized.ReviewerModels...)
	candidate.IssueWriterModel = materialized.IssueWriterModel
	candidate.CompareModels = false
	candidate.ModelPrices = maps.Clone(materialized.ModelPrices)
	candidate.Force = materialized.Force
	candidate.AutoContinue = materialized.AutoContinue
	candidate.MaxFilesPerRun = materialized.MaxFilesPerRun
	candidate.MaxContentBytes = materialized.MaxContentBytes
	candidate.MaxParallelChildren = materialized.MaxParallelChildren
	candidate.AssignmentTimeoutSeconds = materialized.AssignmentTimeoutSeconds
	candidate.EstimatedOutputTokens = materialized.EstimatedOutputTokens
	candidate.BudgetPolicy = materialized.BudgetPolicy
}

func applyRepositoryReviewAutomationRequest(
	automation *repoaudit.RepositoryReviewAutomation,
	request repositoryReviewAutomationConfigRequest,
) {
	if automation == nil {
		return
	}
	automation.Name = request.Name
	automation.Repository = request.Repository
	automation.Ref = request.Ref
	automation.Target = request.Target
	automation.ReviewFocus = request.ReviewFocus
	automation.AccountRef = request.AccountRef
	automation.ScopePolicy = request.ScopePolicy
	automation.ReviewerModels = append([]string(nil), request.ReviewerModels...)
	automation.CompareModels = request.CompareModels
	automation.Force = request.Force
	if request.AutoContinue != nil {
		automation.AutoContinue = *request.AutoContinue
	}
	automation.MaxFilesPerRun = request.MaxFilesPerRun
	automation.MaxContentBytes = request.MaxContentBytes
	automation.MaxParallelChildren = request.MaxParallelChildren
	if request.AssignmentTimeoutSeconds.Present {
		automation.AssignmentTimeoutSeconds = request.AssignmentTimeoutSeconds.Value
	}
	automation.EstimatedOutputTokens = 1_800
	automation.BudgetPolicy = request.Budget
}

func repositoryReviewExecutionConfigurationChanged(
	previous, next repoaudit.RepositoryReviewAutomation,
) bool {
	return previous.ProfileID != next.ProfileID || previous.ProfileVersion != next.ProfileVersion ||
		previous.AccountRef != next.AccountRef ||
		previous.Repository != next.Repository || previous.Ref != next.Ref ||
		previous.Target != next.Target || previous.ReviewFocus != next.ReviewFocus ||
		!repositoryReviewScopePoliciesEqual(previous.ScopePolicy, next.ScopePolicy) ||
		previous.IssueWriterModel != next.IssueWriterModel ||
		previous.CompareModels != next.CompareModels || previous.Force != next.Force ||
		previous.MaxContentBytes != next.MaxContentBytes || previous.MaxParallelChildren != next.MaxParallelChildren ||
		previous.AssignmentTimeoutSeconds != next.AssignmentTimeoutSeconds ||
		!reflect.DeepEqual(previous.BudgetPolicy, next.BudgetPolicy) ||
		!slicesEqual(previous.ReviewerModels, next.ReviewerModels)
}

func repositoryReviewClearStaleAccountLimits(
	next *repoaudit.RepositoryReviewAutomation,
	previous repoaudit.RepositoryReviewAutomation,
) {
	if !reflect.DeepEqual(previous.BudgetPolicy, next.BudgetPolicy) ||
		previous.AccountRef != next.AccountRef {
		next.AccountLimitSnapshots = nil
	}
}

func repositoryReviewScopePoliciesEqual(
	left, right repoaudit.RepositoryReviewScopePolicy,
) bool {
	left, leftErr := repoaudit.NormalizeRepositoryReviewScopePolicy(left)
	right, rightErr := repoaudit.NormalizeRepositoryReviewScopePolicy(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return slicesEqualRepositoryReviewCodeTypes(left.CodeTypes, right.CodeTypes) &&
		slicesEqual(left.IncludeFolders, right.IncludeFolders) &&
		slicesEqual(left.ExcludeFolders, right.ExcludeFolders) &&
		left.FreeText == right.FreeText
}

func slicesEqualRepositoryReviewCodeTypes(
	left, right []repoaudit.RepositoryReviewCodeType,
) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func slicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (h *Handler) handleRepositoryReviewAutomationOptions(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		writeRepositoryReviewAutomationError(w, err)
		return
	}
	limitsCtx, cancelLimits := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancelLimits()
	limits, limitsErr := loadCodexAccountLimits(limitsCtx)
	accounts := repositoryReviewAccountOptions(cfg, limits)
	selectableAccountRefs := make([]string, 0, len(accounts))
	for _, account := range accounts {
		if account.Available {
			selectableAccountRefs = append(selectableAccountRefs, account.ID)
		}
	}
	models := repositoryReviewModelOptions(cfg, selectableAccountRefs...)
	response := map[string]any{"models": models, "accounts": accounts}
	if limitsError := repositoryReviewLimitsError(limits, limitsErr); limitsError != "" {
		response["limits_error"] = limitsError
	}
	writeRepositoryReviewJSON(w, http.StatusOK, response)
}

func repositoryReviewLimitsError(limits codexAccountLimitsResponse, limitsErr error) string {
	if limitsErr != nil {
		return limitsErr.Error()
	}
	return strings.TrimSpace(limits.Error)
}

func repositoryReviewModelOptions(
	cfg *config.Config,
	additionalAccountRefs ...string,
) []repositoryReviewModelOption {
	if cfg == nil {
		return []repositoryReviewModelOption{}
	}
	defaultModel := strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
	options := make([]repositoryReviewModelOption, 0, len(cfg.ModelAliases))
	for _, alias := range cfg.ModelAliases {
		resolved := strings.TrimSpace(alias.Model)
		provider, concrete := protocoltypes.SplitKnownProviderModel(resolved)
		if provider != "" {
			resolved = concrete
		}
		option := repositoryReviewModelOption{
			Alias: alias.Name, ResolvedModel: resolved, Provider: provider,
			Available: repositoryReviewAliasAvailableForRuntime(cfg, alias, additionalAccountRefs...),
			Default:   alias.Name == defaultModel,
		}
		if repositoryReviewAliasUsesAgenticCLI(cfg, alias) {
			option.Available = false
			option.BlockedReason = "Agentic CLI models are not allowed for immutable repository review."
		} else if !option.Available {
			option.BlockedReason = "This alias is unavailable for every active review account."
		}
		if account, ok := repositoryReviewConservativePricedAccount(cfg, alias); ok {
			option.PriceKnown = account.InputPricePerMTok > 0 || account.OutputPricePerMTok > 0
			option.InputPricePer1M = account.InputPricePerMTok
			option.OutputPricePer1M = account.OutputPricePerMTok
			option.Subscription = account.Subscription
			option.EquivalentModel = account.SubscriptionEquivalentModel
			if option.Provider == "" {
				option.Provider = protocoltypes.NormalizeProvider(account.Provider)
			}
		}
		options = append(options, option)
	}
	sort.Slice(options, func(i, j int) bool {
		if options[i].Default != options[j].Default {
			return options[i].Default
		}
		return options[i].Alias < options[j].Alias
	})
	return options
}

func (h *Handler) refreshRepositoryReviewAccountingSnapshot(
	automation *repoaudit.RepositoryReviewAutomation,
) error {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return err
	}
	return repositoryReviewRefreshAccountingSnapshot(cfg, automation)
}

func repositoryReviewRefreshAccountingSnapshot(
	cfg *config.Config,
	automation *repoaudit.RepositoryReviewAutomation,
) error {
	if automation == nil {
		return fmt.Errorf("%w: repository review automation is required", repoaudit.ErrInvalidAutomation)
	}
	aliases := make(map[string]config.ModelAliasConfig)
	if cfg != nil {
		aliases = make(map[string]config.ModelAliasConfig, len(cfg.ModelAliases))
		for _, alias := range cfg.ModelAliases {
			aliases[strings.TrimSpace(alias.Name)] = alias
		}
	}
	snapshot := make(map[string]repoaudit.RepositoryReviewModelPrice)
	accountRef := repositoryReviewEffectiveAccountRef(cfg, automation.AccountRef)
	for _, aliasName := range repositoryReviewExecutionModels(*automation) {
		_, found := aliases[strings.TrimSpace(aliasName)]
		if !found {
			continue
		}
		resolved, known := repositoryReviewAliasPriceForAccount(
			cfg, aliasName, accountRef, make(map[string]bool),
		)
		if !known || resolved.InputPricePerMTok <= 0 && resolved.OutputPricePerMTok <= 0 {
			continue
		}
		snapshot[aliasName] = repoaudit.RepositoryReviewModelPrice{
			InputPricePer1M:  resolved.InputPricePerMTok,
			OutputPricePer1M: resolved.OutputPricePerMTok,
		}
	}
	automation.ModelPrices = snapshot
	if !repoaudit.RepositoryReviewGuardUsesSpend(automation.BudgetPolicy.GuardExpression) {
		return nil
	}
	for _, aliasName := range repositoryReviewExecutionModels(*automation) {
		price, known := snapshot[aliasName]
		if !known || price.InputPricePer1M <= 0 && price.OutputPricePer1M <= 0 {
			return fmt.Errorf(
				"%w: spend.total.* requires centrally configured pricing for reviewer %q on account %q",
				repoaudit.ErrInvalidAutomation,
				aliasName, accountRef,
			)
		}
	}
	return nil
}

func repositoryReviewAliasAvailableForRuntime(
	cfg *config.Config,
	alias config.ModelAliasConfig,
	additionalAccountRefs ...string,
) bool {
	if cfg == nil || strings.TrimSpace(alias.Name) == "" {
		return false
	}
	accountRefs := append(repositoryReviewSelectableAccountRefs(cfg), additionalAccountRefs...)
	seen := make(map[string]struct{}, len(accountRefs))
	for _, accountRef := range accountRefs {
		accountRef = strings.TrimSpace(accountRef)
		if accountRef == "" {
			continue
		}
		if _, duplicate := seen[accountRef]; duplicate {
			continue
		}
		seen[accountRef] = struct{}{}
		if repositoryReviewAliasAvailableForAccount(cfg, alias.Name, accountRef) {
			return true
		}
	}
	return false
}

func repositoryReviewEffectiveAccountRef(cfg *config.Config, accountRef string) string {
	accountRef = strings.TrimSpace(accountRef)
	if accountRef == "" && cfg != nil {
		accountRef = strings.TrimSpace(cfg.Agents.Defaults.AccountRef)
	}
	return accountRef
}

func repositoryReviewAccountRefsForSelection(cfg *config.Config, accountRef string) []string {
	accountRef = repositoryReviewEffectiveAccountRef(cfg, accountRef)
	if accountRef == "" || cfg == nil {
		return nil
	}
	for index := range cfg.AccountRouters {
		router := &cfg.AccountRouters[index]
		if router.Enabled && strings.TrimSpace(router.Name) == accountRef {
			return repositoryReviewReachableAccountRouterRefs(router)
		}
	}
	if account, err := cfg.GetEnabledModelConfig(
		accountRef,
	); err == nil && account != nil &&
		account.IsAccountRouter() {
		return repositoryReviewReachableAccountRouterRefs(account.Router)
	}
	return []string{accountRef}
}

func repositoryReviewAliasAvailableForAccount(
	cfg *config.Config,
	aliasName string,
	accountRef string,
) bool {
	if cfg == nil || strings.TrimSpace(aliasName) == "" {
		return false
	}
	refs := repositoryReviewAccountRefsForSelection(cfg, accountRef)
	if len(refs) == 0 {
		return false
	}
	for _, concrete := range refs {
		resolved, err := cfg.ResolveModelAlias(aliasName, concrete)
		if err != nil || strings.TrimSpace(resolved) == "" {
			return false
		}
	}
	return true
}

func repositoryReviewAliasUsesAgenticCLIOnAccount(
	cfg *config.Config,
	aliasName string,
	accountRef string,
) bool {
	if cfg == nil {
		return false
	}
	for _, concrete := range repositoryReviewAccountRefsForSelection(cfg, accountRef) {
		model, err := cfg.ResolveModelAlias(aliasName, concrete)
		if err != nil {
			continue
		}
		provider, _ := protocoltypes.SplitKnownProviderModel(strings.TrimSpace(model))
		if provider == "codex-cli" || provider == "claude-cli" {
			return true
		}
		if account, accountErr := cfg.GetEnabledModelConfig(concrete); accountErr == nil && account != nil {
			provider = protocoltypes.NormalizeProvider(account.Provider)
			if provider == "codex-cli" || provider == "claude-cli" {
				return true
			}
		}
	}
	return false
}

func repositoryReviewAliasPriceForAccount(
	cfg *config.Config,
	aliasName string,
	accountRef string,
	visiting map[string]bool,
) (*config.ModelConfig, bool) {
	if cfg == nil || strings.TrimSpace(aliasName) == "" || visiting[aliasName] {
		return nil, false
	}
	visiting[aliasName] = true
	defer delete(visiting, aliasName)
	refs := repositoryReviewAccountRefsForSelection(cfg, accountRef)
	if len(refs) == 0 {
		return nil, false
	}
	aggregate := &config.ModelConfig{}
	for _, concrete := range refs {
		resolved, err := cfg.ResolveModelAliasConfig(aliasName, concrete)
		if err != nil {
			// Credential-backed virtual accounts have no ModelConfig of their
			// own. Central equivalent-model metadata remains the price authority.
			if _, credential := config.AccountRouterCredentialAccountID(concrete); credential {
				fallback, ok := repositoryReviewAliasPrice(cfg, aliasName, make(map[string]bool))
				if !ok {
					return nil, false
				}
				resolved = fallback
			} else {
				return nil, false
			}
		}
		equivalent := strings.TrimSpace(resolved.SubscriptionEquivalentModel)
		inputPrice, outputPrice, inherited := repositoryReviewResolvedAliasPrices(
			resolved,
			func() (*config.ModelConfig, bool) {
				return repositoryReviewAliasPriceForAccount(cfg, equivalent, accountRef, visiting)
			},
		)
		if !inherited {
			return nil, false
		}
		if inputPrice <= 0 && outputPrice <= 0 {
			return nil, false
		}
		aggregate.InputPricePerMTok = max(aggregate.InputPricePerMTok, inputPrice)
		aggregate.OutputPricePerMTok = max(aggregate.OutputPricePerMTok, outputPrice)
		aggregate.Subscription = aggregate.Subscription || resolved.Subscription
		if aggregate.SubscriptionEquivalentModel == "" {
			aggregate.SubscriptionEquivalentModel = equivalent
		}
		if aggregate.Provider == "" {
			aggregate.Provider = resolved.Provider
		}
	}
	return aggregate, true
}

func repositoryReviewResolvedAliasPrices(
	resolved *config.ModelConfig,
	inherit func() (*config.ModelConfig, bool),
) (float64, float64, bool) {
	inputPrice, outputPrice := resolved.InputPricePerMTok, resolved.OutputPricePerMTok
	equivalent := strings.TrimSpace(resolved.SubscriptionEquivalentModel)
	if inputPrice <= 0 && outputPrice <= 0 && resolved.Subscription && equivalent != "" {
		inherited, ok := inherit()
		if !ok {
			return 0, 0, false
		}
		inputPrice, outputPrice = inherited.InputPricePerMTok, inherited.OutputPricePerMTok
	}
	return inputPrice, outputPrice, true
}

func repositoryReviewAliasUsesAgenticCLI(
	cfg *config.Config,
	alias config.ModelAliasConfig,
) bool {
	models := make([]string, 0, len(alias.AccountOverrides)+1)
	models = append(models, alias.Model)
	relevantAccounts := make(map[string]struct{})
	for _, accountRef := range repositoryReviewRuntimeAccountRefs(cfg) {
		relevantAccounts[accountRef] = struct{}{}
		if override, exists := alias.AccountOverrides[accountRef]; exists {
			models = append(models, override)
		}
	}
	for _, model := range models {
		provider, _ := protocoltypes.SplitKnownProviderModel(strings.TrimSpace(model))
		if provider == "codex-cli" || provider == "claude-cli" {
			return true
		}
	}
	if cfg != nil {
		for accountRef := range relevantAccounts {
			account, err := cfg.GetEnabledModelConfig(accountRef)
			if err != nil || account == nil {
				continue
			}
			provider := protocoltypes.NormalizeProvider(account.Provider)
			if (provider == "codex-cli" || provider == "claude-cli") &&
				func() bool { _, resolveErr := cfg.ResolveModelAlias(alias.Name, accountRef); return resolveErr == nil }() {
				return true
			}
		}
	}
	return false
}

func repositoryReviewRuntimeAccountRefs(cfg *config.Config) []string {
	if cfg == nil {
		return nil
	}
	defaultRef := strings.TrimSpace(cfg.Agents.Defaults.AccountRef)
	refs := []string{defaultRef}
	var router *config.AccountRouterConfig
	for index := range cfg.AccountRouters {
		if cfg.AccountRouters[index].Enabled &&
			strings.TrimSpace(cfg.AccountRouters[index].Name) == defaultRef {
			router = &cfg.AccountRouters[index]
			break
		}
	}
	if router == nil {
		if account, err := cfg.GetEnabledModelConfig(defaultRef); err == nil &&
			account != nil && account.IsAccountRouter() {
			router = account.Router
		}
	}
	if router != nil {
		refs = repositoryReviewReachableAccountRouterRefs(router)
	}
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref == "" {
			continue
		}
		out = append(out, ref)
	}
	return out
}

func repositoryReviewSelectableAccountRefs(cfg *config.Config) []string {
	if cfg == nil {
		return nil
	}
	seen := make(map[string]struct{})
	refs := make([]string, 0, len(cfg.ModelList)+len(cfg.AccountRouters)+1)
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, duplicate := seen[value]; duplicate {
			return
		}
		seen[value] = struct{}{}
		refs = append(refs, value)
	}
	add(cfg.Agents.Defaults.AccountRef)
	for _, account := range cfg.ModelList {
		if account != nil && account.Enabled && !account.IsModelRouter() {
			add(account.ModelName)
		}
	}
	for index := range cfg.AccountRouters {
		if cfg.AccountRouters[index].Enabled {
			add(cfg.AccountRouters[index].Name)
		}
	}
	sort.Strings(refs)
	return refs
}

func repositoryReviewReachableAccountRouterRefs(
	router *config.AccountRouterConfig,
) []string {
	if router == nil {
		return nil
	}
	blocks := make(map[string]config.AccountRouterBlock, len(router.Blocks))
	for _, block := range router.Blocks {
		if id := strings.TrimSpace(block.ID); id != "" {
			blocks[id] = block
		}
	}
	seenBlocks := make(map[string]struct{}, len(blocks))
	seenAccounts := make(map[string]struct{})
	refs := make([]string, 0)
	add := func(ref string) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return
		}
		if _, exists := seenAccounts[ref]; exists {
			return
		}
		seenAccounts[ref] = struct{}{}
		refs = append(refs, ref)
	}
	var walk func(string)
	walk = func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, visited := seenBlocks[id]; visited {
			return
		}
		block, exists := blocks[id]
		if !exists {
			return
		}
		seenBlocks[id] = struct{}{}
		switch strings.TrimSpace(block.Type) {
		case config.AccountRouterBlockTypeAccount:
			add(block.Account)
		case config.AccountRouterBlockTypeLoadBalance:
			for _, account := range block.Accounts {
				add(account)
			}
		case config.AccountRouterBlockTypeBranch:
			walk(block.Then)
			walk(block.Else)
		}
		walk(block.Fallback)
	}
	walk(router.Entry)
	return refs
}

func repositoryReviewConservativePricedAccount(
	cfg *config.Config,
	alias config.ModelAliasConfig,
) (*config.ModelConfig, bool) {
	return repositoryReviewAliasPrice(cfg, alias.Name, make(map[string]bool))
}

func repositoryReviewAliasPrice(
	cfg *config.Config,
	aliasName string,
	visiting map[string]bool,
) (*config.ModelConfig, bool) {
	if cfg == nil {
		return nil, false
	}
	aliasName = strings.TrimSpace(aliasName)
	if aliasName == "" || visiting[aliasName] {
		return nil, false
	}
	visiting[aliasName] = true
	defer delete(visiting, aliasName)
	aggregate := &config.ModelConfig{}
	reachable := false
	for _, accountRef := range repositoryReviewRuntimeAccountRefs(cfg) {
		account, err := cfg.GetEnabledModelConfig(accountRef)
		if err != nil || account == nil || account.IsAccountRouter() || account.IsModelRouter() {
			continue
		}
		resolved, err := cfg.ResolveModelAliasConfig(aliasName, accountRef)
		if err != nil {
			continue
		}
		reachable = true
		inputPrice := resolved.InputPricePerMTok
		outputPrice := resolved.OutputPricePerMTok
		equivalent := strings.TrimSpace(resolved.SubscriptionEquivalentModel)
		if inputPrice <= 0 && outputPrice <= 0 && resolved.Subscription && equivalent != "" {
			if inherited, ok := repositoryReviewEquivalentAliasPrice(cfg, equivalent, visiting); ok {
				inputPrice = inherited.InputPricePerMTok
				outputPrice = inherited.OutputPricePerMTok
			}
		}
		if inputPrice <= 0 && outputPrice <= 0 {
			return nil, false
		}
		aggregate.InputPricePerMTok = max(aggregate.InputPricePerMTok, inputPrice)
		aggregate.OutputPricePerMTok = max(aggregate.OutputPricePerMTok, outputPrice)
		aggregate.Subscription = aggregate.Subscription || resolved.Subscription
		if aggregate.SubscriptionEquivalentModel == "" && equivalent != "" {
			aggregate.SubscriptionEquivalentModel = equivalent
		}
		if aggregate.Provider == "" {
			aggregate.Provider = resolved.Provider
		}
	}
	return aggregate, reachable
}

// repositoryReviewEquivalentAliasPrice resolves comparison-only rates from any
// centrally configured direct account. These accounts are pricing authorities,
// not execution routes for the original subscription-backed alias.
func repositoryReviewEquivalentAliasPrice(
	cfg *config.Config,
	aliasName string,
	visiting map[string]bool,
) (*config.ModelConfig, bool) {
	if cfg == nil {
		return nil, false
	}
	aliasName = strings.TrimSpace(aliasName)
	if aliasName == "" || visiting[aliasName] {
		return nil, false
	}
	visiting[aliasName] = true
	defer delete(visiting, aliasName)
	aggregate := &config.ModelConfig{}
	found := false
	for _, account := range cfg.ModelList {
		if account == nil || !account.Enabled || account.IsAccountRouter() || account.IsModelRouter() {
			continue
		}
		resolved, err := cfg.ResolveModelAliasConfig(aliasName, account.ModelName)
		if err != nil {
			continue
		}
		inputPrice := resolved.InputPricePerMTok
		outputPrice := resolved.OutputPricePerMTok
		equivalent := strings.TrimSpace(resolved.SubscriptionEquivalentModel)
		if inputPrice <= 0 && outputPrice <= 0 && resolved.Subscription && equivalent != "" {
			if inherited, ok := repositoryReviewEquivalentAliasPrice(cfg, equivalent, visiting); ok {
				inputPrice = inherited.InputPricePerMTok
				outputPrice = inherited.OutputPricePerMTok
			}
		}
		if inputPrice <= 0 && outputPrice <= 0 {
			continue
		}
		found = true
		aggregate.InputPricePerMTok = max(aggregate.InputPricePerMTok, inputPrice)
		aggregate.OutputPricePerMTok = max(aggregate.OutputPricePerMTok, outputPrice)
	}
	return aggregate, found
}

func repositoryReviewAccountAvailable(
	cfg *config.Config,
	accountRef string,
	telemetry codexAccountLimitAccount,
	hasTelemetry bool,
) bool {
	accountRef = strings.TrimSpace(accountRef)
	if accountRef == "" {
		return false
	}
	if _, credential := config.AccountRouterCredentialAccountID(accountRef); credential {
		if !credentialAccountAvailable(accountRef) {
			return false
		}
		return !hasTelemetry ||
			strings.EqualFold(strings.TrimSpace(telemetry.CredentialStatus), "available")
	}
	if cfg == nil {
		return false
	}
	for index := range cfg.AccountRouters {
		if cfg.AccountRouters[index].Enabled &&
			strings.TrimSpace(cfg.AccountRouters[index].Name) == accountRef {
			return true
		}
	}
	account, err := cfg.GetEnabledModelConfig(accountRef)
	return err == nil && account != nil && !account.IsModelRouter()
}

func repositoryReviewAccountOptions(
	cfg *config.Config,
	limits codexAccountLimitsResponse,
) []repositoryReviewAccountOption {
	byTelemetryID := make(map[string]codexAccountLimitAccount, len(limits.Accounts))
	for _, account := range limits.Accounts {
		byTelemetryID[strings.ToLower(strings.TrimSpace(account.ID))] = account
	}
	refs := repositoryReviewSelectableAccountRefs(cfg)
	for _, account := range limits.Accounts {
		refs = append(refs, config.AccountRouterCredentialAccountPrefix+strings.ToLower(strings.TrimSpace(account.ID)))
	}
	seen := make(map[string]struct{}, len(refs))
	accounts := make([]repositoryReviewAccountOption, 0, len(refs))
	defaultRef := repositoryReviewEffectiveAccountRef(cfg, "")
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if _, duplicate := seen[ref]; duplicate {
			continue
		}
		seen[ref] = struct{}{}
		provider, telemetryID, label := "", "", ref
		if credentialID, ok := config.AccountRouterCredentialAccountID(ref); ok {
			telemetryID = credentialID
			provider, _ = config.AccountRouterCredentialAccountProvider(ref)
		} else if cfg != nil {
			if configured, err := cfg.GetEnabledModelConfig(ref); err == nil && configured != nil {
				provider = protocoltypes.NormalizeProvider(configured.Provider)
				telemetryID = strings.ToLower(strings.TrimSpace(configured.CredentialID))
			}
		}
		telemetry, hasTelemetry := byTelemetryID[telemetryID]
		if hasTelemetry {
			if email := strings.TrimSpace(telemetry.Email); email != "" {
				label = ref + " · " + email
			} else if accountID := strings.TrimSpace(telemetry.AccountID); accountID != "" {
				label = ref + " · " + accountID
			}
			if provider == "" {
				provider = telemetry.Provider
			}
		}
		status := firstRepositoryReviewLimitDetail(
			telemetry.LimitsStatus, telemetry.CredentialStatus, telemetry.LimitsError,
		)
		if !hasTelemetry {
			status = "available"
		}
		option := repositoryReviewAccountOption{
			ID: ref, Provider: provider, Label: label, Status: status, Default: ref == defaultRef,
			Available:    repositoryReviewAccountAvailable(cfg, ref, telemetry, hasTelemetry),
			Entries:      make([]repositoryReviewAccountLimitOption, 0, len(telemetry.Entries)),
			Models:       []string{},
			WriterModels: []string{},
		}
		if cfg != nil {
			for _, alias := range cfg.ModelAliases {
				if repositoryReviewAliasAvailableForAccount(cfg, alias.Name, ref) &&
					!repositoryReviewAliasUsesAgenticCLI(cfg, alias) &&
					!repositoryReviewAliasUsesAgenticCLIOnAccount(cfg, alias.Name, ref) {
					option.Models = append(option.Models, alias.Name)
				}
				if repositoryReviewAliasAvailableForAccount(cfg, alias.Name, ref) &&
					!repositoryReviewAliasUsesAgenticCLIOnAccount(cfg, alias.Name, ref) {
					option.WriterModels = append(option.WriterModels, alias.Name)
				}
			}
			sort.Strings(option.Models)
			sort.Strings(option.WriterModels)
		}
		for _, entry := range telemetry.Entries {
			limit := repositoryReviewAccountLimitOption{
				Name: entry.Name, Status: entry.Status, Window: entry.Window,
				UsedPercent: entry.UsedPercent, RefreshesAt: entry.RefreshesAt,
			}
			if entry.UsedPercent != nil {
				remaining := math.Max(0, math.Min(100, 100-float64(*entry.UsedPercent)))
				limit.RemainingPercent = &remaining
			}
			option.Entries = append(option.Entries, limit)
		}
		accounts = append(accounts, option)
	}
	sort.SliceStable(accounts, func(i, j int) bool {
		if accounts[i].Default != accounts[j].Default {
			return accounts[i].Default
		}
		return accounts[i].Label < accounts[j].Label
	})
	return accounts
}

func writeRepositoryReviewAutomationError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "repository_review_automation_unavailable"
	switch {
	case errors.Is(err, os.ErrNotExist) || strings.Contains(strings.ToLower(err.Error()), "not found"):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, repoaudit.ErrRepositoryReviewPurgeInProgress):
		status, code = http.StatusConflict, "repository_review_purge_in_progress"
	case errors.Is(err, repoaudit.ErrHistoricalDeduplicationRestartRequired):
		status, code = http.StatusConflict, "historical_consolidation_restart_required"
	case errors.Is(err, repoaudit.ErrHistoricalDeduplicationInProgress),
		errors.Is(err, repoaudit.ErrHistoricalDeduplicationNotQuiescent):
		status, code = http.StatusConflict, "historical_deduplication_in_progress"
	case errors.Is(err, errRepositoryReviewCommitSelection):
		status, code = http.StatusConflict, "repository_review_commit_selection_required"
	case errors.Is(err, repoaudit.ErrConflict), errors.Is(err, repoaudit.ErrAutomationActive),
		errors.Is(err, errRepositoryReviewAutomationBusy),
		errors.Is(err, errRepositoryReviewInvalidTransition),
		errors.Is(err, repoaudit.ErrAutomationControllerLocked):
		status, code = http.StatusConflict, "stale_repository_review_automation"
	case errors.Is(err, repoaudit.ErrRepositoryReviewRepositoryConflict):
		status, code = http.StatusConflict, "repository_review_repository_assigned"
	case errors.Is(err, repoaudit.ErrInvalidAutomation),
		errors.Is(err, io.EOF),
		errors.Is(err, io.ErrUnexpectedEOF),
		func() bool { var target *json.SyntaxError; return errors.As(err, &target) }(),
		func() bool { var target *json.UnmarshalTypeError; return errors.As(err, &target) }(),
		strings.Contains(strings.ToLower(err.Error()), "invalid"),
		strings.Contains(strings.ToLower(err.Error()), "required"),
		strings.Contains(strings.ToLower(err.Error()), "unknown field"),
		strings.Contains(strings.ToLower(err.Error()), "cannot unmarshal"),
		strings.Contains(strings.ToLower(err.Error()), "unexpected end"):
		status, code = http.StatusBadRequest, "invalid_repository_review_automation"
	}
	message := strings.TrimSpace(err.Error())
	if status >= 500 {
		message = strings.ReplaceAll(code, "_", " ")
	}
	writeRepositoryReviewJSON(w, status, map[string]string{"code": code, "message": message})
}
