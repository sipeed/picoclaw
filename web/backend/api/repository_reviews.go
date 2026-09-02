package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/repoaudit"
)

// Draft text can legally require six-byte JSON escapes per source byte; leave
// bounded envelope/label headroom so every Store-valid draft remains editable.
const repositoryReviewRequestMaxBytes = 8 << 20

type repositoryReviewStatusRequest struct {
	Status          repoaudit.FindingStatus `json:"status"`
	ExpectedVersion int64                   `json:"expected_version"`
}

type repositoryReviewIssueRequest struct {
	FindingIDs      []string `json:"finding_ids"`
	Title           string   `json:"title,omitempty"`
	Body            string   `json:"body,omitempty"`
	Labels          []string `json:"labels,omitempty"`
	ExpectedVersion int64    `json:"expected_version"`
}

type repositoryReviewIssueUpdateRequest struct {
	Title           string   `json:"title"`
	Body            string   `json:"body"`
	Labels          []string `json:"labels,omitempty"`
	ExpectedVersion int64    `json:"expected_version"`
}

func (h *Handler) registerRepositoryReviewRoutes(mux *http.ServeMux) {
	h.registerRepositoryReviewAutomationRoutes(mux)
	mux.HandleFunc("GET /api/repository-reviews", h.handleListRepositoryReviews)
	mux.HandleFunc("GET /api/repository-reviews/{repository_id}", h.handleGetRepositoryReview)
	mux.HandleFunc(
		"PATCH /api/repository-reviews/{repository_id}/findings/{finding_id}",
		h.handleUpdateRepositoryReviewFinding,
	)
	mux.HandleFunc(
		"POST /api/repository-reviews/{repository_id}/issue-drafts",
		h.handlePrepareRepositoryReviewIssue,
	)
	mux.HandleFunc(
		"PATCH /api/repository-reviews/{repository_id}/issue-drafts/{draft_id}",
		h.handleUpdateRepositoryReviewIssue,
	)
	// The legacy repository-owned publish route is dispatched from a tail
	// wildcard so the standard automation-owned issue routes can remain more
	// specific under Go's ServeMux precedence rules.
	mux.HandleFunc(
		"POST /api/repository-reviews/{repository_id}/{legacy_action...}",
		h.handleLegacyRepositoryReviewAction,
	)
}

func (h *Handler) handleLegacyRepositoryReviewAction(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(strings.Trim(r.PathValue("legacy_action"), "/"), "/")
	if len(segments) != 3 || segments[0] != "issue-drafts" || segments[1] == "" ||
		segments[2] != "publish" {
		http.NotFound(w, r)
		return
	}
	r.SetPathValue("draft_id", segments[1])
	h.handlePublishRepositoryReviewIssue(w, r)
}

func (h *Handler) handlePublishRepositoryReviewIssue(w http.ResponseWriter, r *http.Request) {
	if err := validateRepositoryReviewMutation(r); err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	if r.Body == nil || r.ContentLength > 16<<10 {
		writeRepositoryReviewError(w, errors.New("invalid repository review request"))
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, (16<<10)+1))
	if err != nil || len(body) == 0 || len(body) > 16<<10 {
		writeRepositoryReviewError(w, errors.New("invalid repository review request"))
		return
	}
	upstream := "/runtime/repository-reviews/" + r.PathValue("repository_id") +
		"/issue-drafts/" + r.PathValue("draft_id") + "/publish"
	h.proxyPRWorkspaceGateway(w, r, http.MethodPost, upstream, "", body, time.Minute)
}

func (h *Handler) handleListRepositoryReviews(w http.ResponseWriter, _ *http.Request) {
	store, err := h.repositoryReviewStore()
	if err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	states, err := store.ListSummaries()
	if err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	writeRepositoryReviewJSON(w, http.StatusOK, map[string]any{"repositories": states})
}

func (h *Handler) handleGetRepositoryReview(w http.ResponseWriter, r *http.Request) {
	page, err := repositoryReviewPage(r)
	if err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	store, err := h.repositoryReviewStore()
	if err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	state, found, err := store.GetByID(r.PathValue("repository_id"))
	if err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	if !found {
		writeRepositoryReviewError(w, os.ErrNotExist)
		return
	}
	writeRepositoryReviewJSON(w, http.StatusOK, projectRepositoryReviewDetail(state, page))
}

// repositoryReviewDetailResponse is the bounded legacy repository projection.
// Keep this allowlist explicit: embedding RepositoryState would make every new
// durable ledger or controller-authority field public by default.
type repositoryReviewDetailResponse struct {
	SchemaVersion          int                                  `json:"schema_version"`
	ID                     string                               `json:"id"`
	Repository             string                               `json:"repository"`
	Version                int64                                `json:"version"`
	ReviewVersion          int64                                `json:"review_version"`
	LastCommitSHA          string                               `json:"last_commit_sha,omitempty"`
	FindingCount           int                                  `json:"finding_count"`
	RepositoryFindingCount int                                  `json:"repository_finding_count"`
	OpenFindingCount       int                                  `json:"open_finding_count"`
	IssueDraftCount        int                                  `json:"issue_draft_count"`
	UnsupportedCount       int                                  `json:"unsupported_count"`
	ReviewedFileCount      int                                  `json:"reviewed_file_count"`
	ExcludedFileCount      int                                  `json:"excluded_file_count"`
	UpdatedAt              time.Time                            `json:"updated_at"`
	Unsupported            map[string]repoaudit.UnsupportedFile `json:"unsupported,omitempty"`
	Findings               []repoaudit.Finding                  `json:"findings"`
	Contexts               []repoaudit.FindingContext           `json:"contexts"`
	Runs                   []repoaudit.ReviewRun                `json:"runs"`
	IssueDrafts            []repoaudit.IssueDraft               `json:"issue_drafts"`
	FindingOffset          int                                  `json:"finding_offset"`
	FindingTotal           int                                  `json:"finding_total"`
	NextFindingOffset      *int                                 `json:"next_finding_offset,omitempty"`
	DraftOffset            int                                  `json:"draft_offset"`
	DraftTotal             int                                  `json:"draft_total"`
	NextDraftOffset        *int                                 `json:"next_draft_offset,omitempty"`
}

type repositoryReviewPageRequest struct {
	FindingOffset int
	FindingLimit  int
	DraftOffset   int
	DraftLimit    int
}

func repositoryReviewPage(r *http.Request) (repositoryReviewPageRequest, error) {
	if r == nil || r.URL == nil {
		return repositoryReviewPageRequest{}, errors.New("invalid repository review request")
	}
	query := r.URL.Query()
	if len(query) > 4 {
		return repositoryReviewPageRequest{}, errors.New("invalid repository review request")
	}
	for key := range query {
		if key != "offset" && key != "limit" && key != "draft_offset" && key != "draft_limit" {
			return repositoryReviewPageRequest{}, errors.New("invalid repository review request")
		}
		if len(query[key]) != 1 {
			return repositoryReviewPageRequest{}, errors.New("invalid repository review request")
		}
	}
	page := repositoryReviewPageRequest{FindingLimit: 50, DraftLimit: 10}
	var err error
	page.FindingOffset, err = repositoryReviewPageInteger(query.Get("offset"), page.FindingOffset, 0)
	if err != nil {
		return repositoryReviewPageRequest{}, err
	}
	page.FindingLimit, err = repositoryReviewPageInteger(query.Get("limit"), page.FindingLimit, 200)
	if err != nil {
		return repositoryReviewPageRequest{}, err
	}
	page.DraftOffset, err = repositoryReviewPageInteger(query.Get("draft_offset"), page.DraftOffset, 0)
	if err != nil {
		return repositoryReviewPageRequest{}, err
	}
	page.DraftLimit, err = repositoryReviewPageInteger(query.Get("draft_limit"), page.DraftLimit, 20)
	if err != nil {
		return repositoryReviewPageRequest{}, err
	}
	return page, nil
}

func repositoryReviewPageInteger(raw string, fallback, maximum int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 || maximum > 0 && (value < 1 || value > maximum) {
		return 0, errors.New("invalid repository review request")
	}
	return value, nil
}

func projectRepositoryReviewDetail(
	state repoaudit.RepositoryState,
	page repositoryReviewPageRequest,
) repositoryReviewDetailResponse {
	total := len(state.Findings)
	if page.FindingOffset > total {
		page.FindingOffset = total
	}
	end := min(total, page.FindingOffset+page.FindingLimit)
	findings := append([]repoaudit.Finding(nil), state.Findings[page.FindingOffset:end]...)
	for index := range findings {
		findings[index].CampaignID = ""
	}
	contextIDs := make(map[string]struct{})
	for _, finding := range findings {
		for _, contextID := range finding.ContextIDs {
			contextIDs[contextID] = struct{}{}
		}
	}
	contexts := make([]repoaudit.FindingContext, 0, len(contextIDs))
	for _, contextRecord := range state.Contexts {
		if _, selected := contextIDs[contextRecord.ID]; selected {
			contextRecord.CampaignID = ""
			contexts = append(contexts, contextRecord)
		}
	}
	unsupportedPaths := make([]string, 0, len(state.Unsupported))
	for pathValue := range state.Unsupported {
		unsupportedPaths = append(unsupportedPaths, pathValue)
	}
	slices.Sort(unsupportedPaths)
	projectedUnsupported := make(map[string]repoaudit.UnsupportedFile, min(200, len(unsupportedPaths)))
	for _, pathValue := range unsupportedPaths[:min(200, len(unsupportedPaths))] {
		unsupported := state.Unsupported[pathValue]
		unsupported.ForceCampaignID = ""
		projectedUnsupported[pathValue] = unsupported
	}
	runs := append([]repoaudit.ReviewRun(nil), state.Runs...)
	if len(runs) > 50 {
		runs = runs[len(runs)-50:]
	}
	for index := range runs {
		runs[index].CampaignID = ""
		runs[index].CheckpointDigests = nil
		runs[index].CheckpointScopes = nil
	}
	draftTotal := len(state.IssueDrafts)
	page.DraftOffset = min(page.DraftOffset, draftTotal)
	draftEnd := draftTotal - page.DraftOffset
	draftStart := max(0, draftEnd-page.DraftLimit)
	issueDrafts := append([]repoaudit.IssueDraft(nil), state.IssueDrafts[draftStart:draftEnd]...)
	summary := repoaudit.Summarize(state)
	response := repositoryReviewDetailResponse{
		SchemaVersion: summary.SchemaVersion, ID: summary.ID, Repository: summary.Repository,
		Version: summary.Version, ReviewVersion: summary.ReviewVersion,
		LastCommitSHA: summary.LastCommitSHA, FindingCount: summary.FindingCount,
		RepositoryFindingCount: summary.RepositoryFindingCount,
		OpenFindingCount:       summary.OpenFindingCount, IssueDraftCount: summary.IssueDraftCount,
		UnsupportedCount: summary.UnsupportedCount, ReviewedFileCount: summary.ReviewedFileCount,
		ExcludedFileCount: summary.ExcludedFileCount, UpdatedAt: summary.UpdatedAt,
		Unsupported: projectedUnsupported, Findings: findings, Contexts: contexts,
		Runs: runs, IssueDrafts: issueDrafts,
		FindingOffset: page.FindingOffset, FindingTotal: total,
		DraftOffset: page.DraftOffset, DraftTotal: draftTotal,
	}
	if end < total {
		next := end
		response.NextFindingOffset = &next
	}
	if draftStart > 0 {
		next := page.DraftOffset + len(issueDrafts)
		response.NextDraftOffset = &next
	}
	return response
}

func (h *Handler) handleUpdateRepositoryReviewFinding(w http.ResponseWriter, r *http.Request) {
	if err := validateRepositoryReviewMutation(r); err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	_, state, ok := h.repositoryReviewMutationState(w, r.PathValue("repository_id"))
	if !ok {
		return
	}
	var request repositoryReviewStatusRequest
	if err := decodeRepositoryReviewRequest(r, &request); err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	findingIndex := slices.IndexFunc(state.Findings, func(finding repoaudit.Finding) bool {
		return finding.ID == r.PathValue("finding_id")
	})
	if findingIndex < 0 {
		writeRepositoryReviewError(w, os.ErrNotExist)
		return
	}
	if request.Status != repoaudit.FindingOpen && request.Status != repoaudit.FindingDismissed {
		writeRepositoryReviewError(w, errors.New("invalid immutable review finding status mutation"))
		return
	}
	writeRepositoryReviewError(w, repoaudit.ErrConflict)
}

func (h *Handler) handlePrepareRepositoryReviewIssue(w http.ResponseWriter, r *http.Request) {
	if err := validateRepositoryReviewMutation(r); err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	store, state, ok := h.repositoryReviewMutationState(w, r.PathValue("repository_id"))
	if !ok {
		return
	}
	var request repositoryReviewIssueRequest
	if err := decodeRepositoryReviewRequest(r, &request); err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	updated, draft, err := store.PrepareIssue(repoaudit.IssueDraftRequest{
		Repository: state.Repository, FindingIDs: request.FindingIDs,
		Title: request.Title, Body: request.Body, Labels: request.Labels,
		ExpectedVersion: request.ExpectedVersion,
	})
	if err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	writeRepositoryReviewJSON(w, http.StatusCreated, map[string]any{
		"repository": repoaudit.Summarize(updated),
		"draft":      draft,
	})
}

func (h *Handler) handleUpdateRepositoryReviewIssue(w http.ResponseWriter, r *http.Request) {
	if err := validateRepositoryReviewMutation(r); err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	store, state, ok := h.repositoryReviewMutationState(w, r.PathValue("repository_id"))
	if !ok {
		return
	}
	var request repositoryReviewIssueUpdateRequest
	if err := decodeRepositoryReviewRequest(r, &request); err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	updated, draft, err := store.UpdateIssueDraft(
		state.Repository,
		r.PathValue("draft_id"),
		request.Title,
		request.Body,
		request.Labels,
		request.ExpectedVersion,
	)
	if err != nil {
		writeRepositoryReviewError(w, err)
		return
	}
	writeRepositoryReviewJSON(w, http.StatusOK, map[string]any{
		"repository": repoaudit.Summarize(updated),
		"draft":      draft,
	})
}

func (h *Handler) repositoryReviewMutationState(
	w http.ResponseWriter,
	id string,
) (repoaudit.Store, repoaudit.RepositoryState, bool) {
	store, err := h.repositoryReviewStore()
	if err != nil {
		writeRepositoryReviewError(w, err)
		return repoaudit.Store{}, repoaudit.RepositoryState{}, false
	}
	state, found, err := store.GetByID(id)
	if err != nil {
		writeRepositoryReviewError(w, err)
		return repoaudit.Store{}, repoaudit.RepositoryState{}, false
	}
	if !found {
		writeRepositoryReviewError(w, os.ErrNotExist)
		return repoaudit.Store{}, repoaudit.RepositoryState{}, false
	}
	return store, state, true
}

func (h *Handler) repositoryReviewStore() (repoaudit.Store, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return repoaudit.Store{}, err
	}
	return repoaudit.NewStore(cfg.WorkspacePath()), nil
}

func decodeRepositoryReviewRequest(r *http.Request, target any) error {
	if r == nil || r.Body == nil || r.ContentLength > repositoryReviewRequestMaxBytes {
		return errors.New("invalid repository review request")
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, repositoryReviewRequestMaxBytes+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("invalid repository review request")
	}
	return nil
}

func validateRepositoryReviewMutation(r *http.Request) error {
	if r == nil || r.URL == nil || r.URL.RawQuery != "" || prWorkspaceMutationCrossSite(r) ||
		validateEventReplayHeaders(r.Header) != nil {
		return errors.New("invalid repository review request")
	}
	return nil
}

func writeRepositoryReviewError(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "repository_review_unavailable"
	switch {
	case errors.Is(err, os.ErrNotExist):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, repoaudit.ErrRepositoryReviewPurgeInProgress):
		status, code = http.StatusConflict, "repository_review_purge_in_progress"
	case errors.Is(err, repoaudit.ErrHistoricalDeduplicationInProgress):
		status, code = http.StatusConflict, "historical_deduplication_in_progress"
	case errors.Is(err, repoaudit.ErrConflict):
		status, code = http.StatusConflict, "stale_repository_review"
	case errors.Is(err, repoaudit.ErrInvalidPlan),
		errors.Is(err, io.ErrUnexpectedEOF),
		isRepositoryReviewJSONError(err),
		strings.Contains(strings.ToLower(err.Error()), "invalid"),
		strings.Contains(strings.ToLower(err.Error()), "required"),
		strings.Contains(strings.ToLower(err.Error()), "duplicate"),
		strings.Contains(strings.ToLower(err.Error()), "unknown field"),
		strings.Contains(strings.ToLower(err.Error()), "cannot unmarshal"),
		strings.Contains(strings.ToLower(err.Error()), "unexpected end"),
		errors.Is(err, io.EOF):
		status, code = http.StatusBadRequest, "invalid_request"
	}
	writeRepositoryReviewJSON(w, status, map[string]string{"code": code, "message": strings.ReplaceAll(code, "_", " ")})
}

func writeRepositoryReviewJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
