package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/repoaudit"
	"github.com/sipeed/picoclaw/pkg/reviews"
	"github.com/sipeed/picoclaw/pkg/workflows"
)

const repositoryReviewPublicationRoute = "/runtime/repository-reviews/"

var repositoryReviewHTTPStatusPattern = regexp.MustCompile(`(?i)(?:status|http(?:/\d(?:\.\d)?)?)[:\s]+(\d{3})`)

type repositoryReviewPublicationHandler struct {
	loop              atomic.Pointer[agent.AgentLoop]
	publishMu         sync.Mutex
	newToolRunner     func(*agent.AgentLoop, string) (workflows.ToolRunner, error)
	newGitHubProvider func(workflows.ToolRunner, string) (*reviews.GitHubProvider, error)
}

type repositoryReviewPublishRequest struct {
	ExpectedVersion int64 `json:"expected_version"`
}

func newRepositoryReviewPublicationHandler(loop *agent.AgentLoop) *repositoryReviewPublicationHandler {
	handler := &repositoryReviewPublicationHandler{
		newToolRunner: agent.NewWorkflowToolRunner, newGitHubProvider: reviews.NewGitHubProvider,
	}
	handler.loop.Store(loop)
	return handler
}

func prepareRepositoryReviewPublicationRoute(
	runningServices *services,
	loop *agent.AgentLoop,
) error {
	if runningServices == nil || runningServices.ChannelManager == nil || loop == nil {
		return errors.New("repository review publication requires the live gateway runtime")
	}
	if runningServices.HealthServer == nil || strings.TrimSpace(runningServices.authToken) == "" ||
		!runningServices.HealthServer.UsesBearerToken(runningServices.authToken) {
		return errors.New("repository review publication requires the protected gateway runtime")
	}
	if runningServices.repositoryReviewPublicationRelease != nil {
		if runningServices.repositoryReviewPublicationHandler == nil {
			return errors.New("repository review publication route state is unavailable")
		}
		runningServices.repositoryReviewPublicationHandler.loop.Store(loop)
		return nil
	}
	handler := newRepositoryReviewPublicationHandler(loop)
	release, err := runningServices.ChannelManager.RegisterHTTPRoute(
		repositoryReviewPublicationRoute,
		runningServices.HealthServer.Protect(handler),
	)
	if err != nil {
		return fmt.Errorf("register repository review publication API: %w", err)
	}
	runningServices.repositoryReviewPublicationHandler = handler
	runningServices.repositoryReviewPublicationRelease = release
	return nil
}

func releaseRepositoryReviewPublicationRoute(runningServices *services) {
	if runningServices == nil || runningServices.repositoryReviewPublicationRelease == nil {
		return
	}
	runningServices.repositoryReviewPublicationRelease()
	runningServices.repositoryReviewPublicationRelease = nil
	runningServices.repositoryReviewPublicationHandler = nil
}

func (handler *repositoryReviewPublicationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setRepositoryReviewPublicationHeaders(w)
	if operation, ok := repositoryReviewAutomationOperationFromRequest(r); ok {
		handler.serveRepositoryReviewAutomationOperation(w, r, operation)
		return
	}
	repositoryID, draftID, ok := repositoryReviewPublicationRouteIDs(r)
	if !ok {
		writeRepositoryReviewPublicationError(w, http.StatusNotFound, "not_found")
		return
	}
	var request repositoryReviewPublishRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || request.ExpectedVersion < 1 {
		writeRepositoryReviewPublicationError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		writeRepositoryReviewPublicationError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	loop := handler.loop.Load()
	if loop == nil || loop.GetConfig() == nil {
		writeRepositoryReviewPublicationError(w, http.StatusServiceUnavailable, "publication_unavailable")
		return
	}
	handler.publishMu.Lock()
	defer handler.publishMu.Unlock()
	store := repoaudit.NewStore(loop.GetConfig().WorkspacePath())
	state, found, err := store.GetByID(repositoryID)
	if err != nil || !found {
		writeRepositoryReviewPublicationStoreError(w, err, found)
		return
	}
	draft, found := repositoryReviewDraft(state, draftID)
	if !found {
		writeRepositoryReviewPublicationError(w, http.StatusNotFound, "not_found")
		return
	}
	eligibility := repoaudit.EvaluateIssuePublication(state, draft)
	if draft.State == repoaudit.IssueDraftPosted {
		if !eligibility.AllowsPostedAcknowledgement() {
			writeRepositoryReviewPublicationEligibilityError(w, eligibility)
			return
		}
		writeRepositoryReviewPublicationJSON(w, http.StatusOK, map[string]any{
			"repository": repoaudit.Summarize(state), "draft": draft,
		})
		return
	}
	if draft.Version != request.ExpectedVersion {
		writeRepositoryReviewPublicationError(w, http.StatusConflict, "stale_repository_review")
		return
	}
	if !eligibility.CanPublish {
		writeRepositoryReviewPublicationEligibilityError(w, eligibility)
		return
	}
	runner, err := handler.newToolRunner(loop, "")
	if err != nil {
		writeRepositoryReviewPublicationError(w, http.StatusServiceUnavailable, "publication_unavailable")
		return
	}
	provider, err := handler.newGitHubProvider(runner, githubMCPArtifactRoot(loop.GetConfig(), loop))
	if err != nil {
		writeRepositoryReviewPublicationError(w, http.StatusServiceUnavailable, "publication_unavailable")
		return
	}
	state, draft, claimed, err := store.ClaimIssueDraftPublication(
		state.Repository,
		draft.ID,
		request.ExpectedVersion,
	)
	if err != nil {
		writeRepositoryReviewPublicationStoreError(w, err, true)
		return
	}
	marker := repositoryReviewIssueMarker(draft.ID)
	if recovered, exists, searchErr := findRepositoryReviewIssue(
		r.Context(),
		provider,
		state.Repository,
		marker,
	); searchErr != nil {
		if claimed {
			_, _, _ = store.SetIssueDraftPublication(
				state.Repository, draft.ID, draft.Version, repoaudit.IssueDraftEditing, "", "",
			)
		}
		writeRepositoryReviewPublicationError(w, http.StatusServiceUnavailable, "publication_unavailable")
		return
	} else if exists {
		updated, posted, updateErr := store.SetIssueDraftPublication(
			state.Repository, draft.ID, draft.Version, repoaudit.IssueDraftPosted,
			recovered.ExternalID, recovered.ExternalURL,
		)
		if updateErr != nil {
			writeRepositoryReviewPublicationStoreError(w, updateErr, true)
			return
		}
		writeRepositoryReviewPublicationJSON(w, http.StatusOK, map[string]any{
			"repository": repoaudit.Summarize(updated), "draft": posted,
		})
		return
	}
	if !claimed {
		if draft.State == repoaudit.IssueDraftPublishing {
			var updateErr error
			state, draft, updateErr = store.SetIssueDraftPublication(
				state.Repository, draft.ID, draft.Version, repoaudit.IssueDraftUnknown, "", "",
			)
			if updateErr != nil {
				writeRepositoryReviewPublicationStoreError(w, updateErr, true)
				return
			}
		}
		writeRepositoryReviewPublicationJSON(w, http.StatusAccepted, map[string]any{
			"repository": repoaudit.Summarize(state), "draft": draft, "outcome": "unknown",
		})
		return
	}
	result, createErr := createRepositoryReviewIssue(r.Context(), provider, state.Repository, draft, marker)
	if createErr != nil {
		if repositoryReviewIssueCreateAmbiguous(createErr) {
			unknownState, unknownDraft, updateErr := store.SetIssueDraftPublication(
				state.Repository, draft.ID, draft.Version, repoaudit.IssueDraftUnknown, "", "",
			)
			if updateErr != nil {
				writeRepositoryReviewPublicationStoreError(w, updateErr, true)
				return
			}
			writeRepositoryReviewPublicationJSON(w, http.StatusAccepted, map[string]any{
				"repository": repoaudit.Summarize(unknownState), "draft": unknownDraft, "outcome": "unknown",
			})
			return
		}
		_, _, updateErr := store.SetIssueDraftPublication(
			state.Repository, draft.ID, draft.Version, repoaudit.IssueDraftEditing, "", "",
		)
		if updateErr != nil {
			writeRepositoryReviewPublicationStoreError(w, updateErr, true)
			return
		}
		writeRepositoryReviewPublicationError(w, http.StatusServiceUnavailable, "publication_failed")
		return
	}
	updated, posted, err := store.SetIssueDraftPublication(
		state.Repository, draft.ID, draft.Version, repoaudit.IssueDraftPosted,
		result.ExternalID, result.ExternalURL,
	)
	if err != nil {
		writeRepositoryReviewPublicationStoreError(w, err, true)
		return
	}
	writeRepositoryReviewPublicationJSON(w, http.StatusOK, map[string]any{
		"repository": repoaudit.Summarize(updated), "draft": posted,
	})
}

func validRepositoryReviewGitHubIdentity(repository string) bool {
	return repoaudit.IsCanonicalGitHubRepository(repository)
}

func repositoryReviewIssueCreateAmbiguous(err error) bool {
	if errors.Is(err, workflows.ErrToolCallNotDispatched) {
		return false
	}
	if !reviews.WorkspaceProviderCallMayHaveChangedExternalState(err) {
		return false
	}
	classified := providers.ClassifyError(err, "github", "issue-create")
	if classified == nil {
		if matched := repositoryReviewHTTPStatusPattern.FindStringSubmatch(err.Error()); len(matched) == 2 {
			status, _ := strconv.Atoi(matched[1])
			if status >= 400 && status < 500 && status != http.StatusRequestTimeout &&
				status != http.StatusTooManyRequests {
				return false
			}
		}
	}
	return classified == nil || classified.Reason == providers.FailoverNetwork ||
		classified.Reason == providers.FailoverTimeout ||
		classified.Reason == providers.FailoverRateLimit ||
		classified.Reason == providers.FailoverOverloaded
}

func repositoryReviewPublicationRouteIDs(r *http.Request) (string, string, bool) {
	if r == nil || r.URL == nil || r.Method != http.MethodPost || r.URL.RawQuery != "" ||
		r.URL.EscapedPath() != r.URL.Path || !strings.HasPrefix(r.URL.Path, repositoryReviewPublicationRoute) {
		return "", "", false
	}
	segments := strings.Split(strings.TrimPrefix(r.URL.Path, repositoryReviewPublicationRoute), "/")
	if len(segments) != 4 || segments[0] == "" || segments[1] != "issue-drafts" ||
		segments[2] == "" || segments[3] != "publish" {
		return "", "", false
	}
	return segments[0], segments[2], true
}

func repositoryReviewDraft(state repoaudit.RepositoryState, id string) (repoaudit.IssueDraft, bool) {
	for _, draft := range state.IssueDrafts {
		if draft.ID == id {
			return draft, true
		}
	}
	return repoaudit.IssueDraft{}, false
}

type repositoryReviewIssueIdentity struct {
	ExternalID  string
	ExternalURL string
}

func createRepositoryReviewIssue(
	ctx context.Context,
	provider *reviews.GitHubProvider,
	repository string,
	draft repoaudit.IssueDraft,
	marker string,
) (repositoryReviewIssueIdentity, error) {
	owner, repo, ok := strings.Cut(repository, "/")
	if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return repositoryReviewIssueIdentity{}, errors.New("repository is not a GitHub owner/repository identity")
	}
	raw, err := provider.CreateWorkspaceIssueJSON(ctx, map[string]any{
		"owner": owner, "repo": repo, "title": draft.Title,
		"body":   strings.TrimSpace(draft.Body) + "\n\n<!-- " + marker + " -->",
		"labels": draft.Labels,
	})
	if err != nil {
		return repositoryReviewIssueIdentity{}, err
	}
	issue, err := decodePRWorkspaceIssue(raw)
	if err != nil || !issueURLBelongsToRepository(issueURL(issue), "https://github.com", repository) {
		return repositoryReviewIssueIdentity{}, errors.New("GitHub issue response is invalid")
	}
	return repositoryReviewIssueIdentity{ExternalID: issueID(issue), ExternalURL: issueURL(issue)}, nil
}

func findRepositoryReviewIssue(
	ctx context.Context,
	provider *reviews.GitHubProvider,
	repository, marker string,
) (repositoryReviewIssueIdentity, bool, error) {
	raw, err := provider.SearchWorkspaceIssuesJSON(ctx, map[string]any{
		"query": `"` + marker + `" in:body is:issue repo:` + repository,
	})
	if err != nil {
		return repositoryReviewIssueIdentity{}, false, err
	}
	var envelope struct {
		Items []prWorkspaceGitHubIssueWire `json:"items"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		var direct []prWorkspaceGitHubIssueWire
		if directErr := json.Unmarshal(raw, &direct); directErr != nil {
			return repositoryReviewIssueIdentity{}, false, errors.New("GitHub issue search response is invalid")
		}
		envelope.Items = direct
	}
	matched := make([]prWorkspaceGitHubIssueWire, 0, 1)
	for _, issue := range envelope.Items {
		if strings.Contains(issue.Body, marker) &&
			issueURLBelongsToRepository(issueURL(issue), "https://github.com", repository) {
			matched = append(matched, issue)
		}
	}
	if len(matched) == 0 {
		return repositoryReviewIssueIdentity{}, false, nil
	}
	if len(matched) != 1 {
		return repositoryReviewIssueIdentity{}, false, errors.New(
			"multiple GitHub issues contain the publication marker",
		)
	}
	return repositoryReviewIssueIdentity{ExternalID: issueID(matched[0]), ExternalURL: issueURL(matched[0])}, true, nil
}

func repositoryReviewIssueMarker(draftID string) string {
	return "picoclaw-repository-review:" + draftID
}

func writeRepositoryReviewPublicationStoreError(w http.ResponseWriter, err error, found bool) {
	switch {
	case !found || errors.Is(err, os.ErrNotExist):
		writeRepositoryReviewPublicationError(w, http.StatusNotFound, "not_found")
	case errors.Is(err, repoaudit.ErrConflict):
		writeRepositoryReviewPublicationError(w, http.StatusConflict, "stale_repository_review")
	case errors.Is(err, repoaudit.ErrRepositoryReviewPurgeInProgress):
		writeRepositoryReviewPublicationError(w, http.StatusConflict, "repository_review_purge_in_progress")
	case errors.Is(err, repoaudit.ErrHistoricalDeduplicationInProgress):
		writeRepositoryReviewPublicationError(
			w,
			http.StatusConflict,
			string(repoaudit.IssuePublicationHistoricalMergeActive),
		)
	default:
		writeRepositoryReviewPublicationError(w, http.StatusServiceUnavailable, "publication_unavailable")
	}
}

func setRepositoryReviewPublicationHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func writeRepositoryReviewPublicationError(w http.ResponseWriter, status int, code string) {
	writeRepositoryReviewPublicationJSON(w, status, map[string]string{
		"code": code, "message": strings.ReplaceAll(code, "_", " "),
	})
}

func writeRepositoryReviewPublicationEligibilityError(
	w http.ResponseWriter,
	eligibility repoaudit.IssuePublicationEligibility,
) {
	if eligibility.CanPublish || len(eligibility.PublishBlockers) == 0 {
		writeRepositoryReviewPublicationError(w, http.StatusConflict, "publication_not_allowed")
		return
	}
	first := eligibility.PublishBlockers[0]
	status := http.StatusConflict
	if first.Code == repoaudit.IssuePublicationRepositoryNotGitHub {
		status = http.StatusBadRequest
	}
	writeRepositoryReviewPublicationJSON(w, status, map[string]any{
		"code": first.Code, "message": first.Message,
		"publish_blockers": eligibility.PublishBlockers,
	})
}

func writeRepositoryReviewPublicationJSON(w http.ResponseWriter, status int, value any) {
	setRepositoryReviewPublicationHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
