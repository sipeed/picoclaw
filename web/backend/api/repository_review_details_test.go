package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/collectionquery"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/repoaudit"
	"github.com/sipeed/picoclaw/pkg/workflows"
)

func TestRepositoryReviewAutomationRoutesRegisterRunFindingStatus(t *testing.T) {
	handler := NewHandler(filepath.Join(t.TempDir(), "config.json"))
	mux := http.NewServeMux()
	handler.registerRepositoryReviewAutomationRoutes(mux)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/repository-reviews/automations/rra_test/findings/status",
		nil,
	)
	_, pattern := mux.Handler(request)
	if pattern != "POST /api/repository-reviews/automations/{automation_id}/findings/status" {
		t.Fatalf("run finding status route pattern=%q", pattern)
	}
}

func TestRepositoryReviewAutomationDetailReportAndFindingRoutes(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	state := seedRepositoryReviewAPIState(t, workspace)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)

	detail := httptest.NewRecorder()
	mux.ServeHTTP(detail, httptest.NewRequest(
		http.MethodGet, "/api/repository-reviews/automations/"+automation.ID, nil,
	))
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), state.ID) ||
		strings.Contains(detail.Body.String(), state.Findings[0].Evidence) {
		t.Fatalf("detail status=%d body=%s", detail.Code, detail.Body.String())
	}

	report := httptest.NewRecorder()
	mux.ServeHTTP(report, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations/"+automation.ID+"/report?scope=current&offset=0&limit=50",
		nil,
	))
	if report.Code != http.StatusOK || !strings.Contains(report.Body.String(), state.Findings[0].Evidence) ||
		!strings.Contains(report.Body.String(), `"scope":"current"`) {
		t.Fatalf("report status=%d body=%s", report.Code, report.Body.String())
	}
	findingsPage := httptest.NewRecorder()
	mux.ServeHTTP(findingsPage, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations/"+automation.ID+"/findings?scope=current&offset=0&limit=50",
		nil,
	))
	if findingsPage.Code != http.StatusOK ||
		!strings.Contains(findingsPage.Body.String(), state.Findings[0].Evidence) ||
		!strings.Contains(findingsPage.Body.String(), `"repository_findings"`) {
		t.Fatalf("findings status=%d body=%s", findingsPage.Code, findingsPage.Body.String())
	}

	findingPath := "/api/repository-reviews/automations/" + automation.ID +
		"/findings/" + state.Findings[0].ID
	finding := httptest.NewRecorder()
	mux.ServeHTTP(finding, httptest.NewRequest(http.MethodGet, findingPath, nil))
	if finding.Code != http.StatusOK || !strings.Contains(finding.Body.String(), state.Contexts[0].ID) ||
		!strings.Contains(finding.Body.String(), `"model":"provider/review-model"`) ||
		!strings.Contains(finding.Body.String(), `"model_alias":"review-model"`) ||
		!strings.Contains(finding.Body.String(), `"account":"api"`) {
		t.Fatalf("finding status=%d body=%s", finding.Code, finding.Body.String())
	}
	updated := repositoryReviewAutomationMutation(t, mux, http.MethodPatch, findingPath, map[string]any{
		"status": "dismissed", "expected_version": state.Findings[0].Version,
	})
	if updated.Code != http.StatusConflict {
		t.Fatalf("immutable finding update status=%d body=%s", updated.Code, updated.Body.String())
	}
}

func TestRepositoryReviewRunFindingStatusProjection(t *testing.T) {
	for _, test := range []struct {
		name    string
		state   repoaudit.RepositoryState
		finding repoaudit.Finding
		want    repositoryReviewRunFindingStatus
	}{
		{
			name: "pending",
			state: repoaudit.RepositoryState{MappingJobs: []repoaudit.RepositoryMappingJob{{
				ReviewFindingID: "rfn_pending", State: repoaudit.RepositoryMappingPending,
				Attempts: 2, Error: "private retry detail",
			}}},
			finding: repoaudit.Finding{ID: "rfn_pending"}, want: repositoryReviewRunFindingPending,
		},
		{
			name: "processing",
			state: repoaudit.RepositoryState{MappingJobs: []repoaudit.RepositoryMappingJob{{
				ReviewFindingID: "rfn_processing", State: repoaudit.RepositoryMappingRunning,
			}}},
			finding: repoaudit.Finding{ID: "rfn_processing"}, want: repositoryReviewRunFindingProcessing,
		},
		{
			name: "failed",
			state: repoaudit.RepositoryState{MappingJobs: []repoaudit.RepositoryMappingJob{{
				ReviewFindingID: "rfn_failed", State: repoaudit.RepositoryMappingPending,
				Attempts: repoaudit.RepositoryRunFindingStatusAttemptLimit,
				Error:    "private failure detail",
			}}},
			finding: repoaudit.Finding{ID: "rfn_failed"}, want: repositoryReviewRunFindingFailed,
		},
		{
			name: "associated new",
			state: repoaudit.RepositoryState{RepositoryFindings: []repoaudit.RepositoryFinding{{
				ID: "rrf_new", MatchState: repoaudit.RepositoryMatchKnown,
				ReviewFindingIDs: []string{"rfn_new", "rfn_later"},
			}}},
			finding: repoaudit.Finding{ID: "rfn_new", RepositoryFindingID: "rrf_new"},
			want:    repositoryReviewRunFindingAssociatedNew,
		},
		{
			name: "associated existing",
			state: repoaudit.RepositoryState{RepositoryFindings: []repoaudit.RepositoryFinding{{
				ID: "rrf_known", MatchState: repoaudit.RepositoryMatchKnown,
				ReviewFindingIDs: []string{"rfn_original", "rfn_known"},
			}}},
			finding: repoaudit.Finding{ID: "rfn_known", RepositoryFindingID: "rrf_known"},
			want:    repositoryReviewRunFindingAssociatedExisting,
		},
		{
			name: "needs review",
			state: repoaudit.RepositoryState{RepositoryFindings: []repoaudit.RepositoryFinding{{
				ID: "rrf_provisional", MatchState: repoaudit.RepositoryMatchProvisional,
			}}},
			finding: repoaudit.Finding{ID: "rfn_provisional", RepositoryFindingID: "rrf_provisional"},
			want:    repositoryReviewRunFindingNeedsReview,
		},
		{
			name:    "known association without aggregate projection",
			finding: repoaudit.Finding{ID: "rfn_known", RepositoryFindingID: "rrf_missing", RepositoryMatchState: repoaudit.RepositoryMatchKnown},
			want:    repositoryReviewRunFindingAssociatedExisting,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := repositoryReviewRunFindingStatusFor(test.state, test.finding); got != test.want {
				t.Fatalf("status=%q want=%q", got, test.want)
			}
		})
	}
}

func TestRepositoryReviewRunFindingStatusRetryRoute(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	state := seedRepositoryReviewAPIState(t, workspace)
	store := repoaudit.NewStore(workspace)
	for attempt := 0; attempt < repoaudit.RepositoryRunFindingStatusAttemptLimit; attempt++ {
		if _, err := store.ProcessPendingMappingJobs(
			t.Context(),
			state.Repository,
			repoaudit.RepositoryMappingProcessOptions{
				DefaultBranchVerified: func(context.Context, repoaudit.Finding) (bool, error) {
					return false, errors.New("private verifier failure")
				},
			},
		); err == nil {
			t.Fatalf("attempt %d succeeded", attempt+1)
		}
	}
	state, found, err := store.Get(state.Repository)
	if err != nil || !found {
		t.Fatalf("failed state found=%v err=%v", found, err)
	}
	findingID := state.Findings[0].ID
	automation := seedRepositoryReviewDetailAutomation(
		t, handler, state.Repository, state.Runs[0].ID,
	)
	base := "/api/repository-reviews/automations/" + automation.ID
	page := httptest.NewRecorder()
	mux.ServeHTTP(page, httptest.NewRequest(
		http.MethodGet, base+"/findings?scope=current", nil,
	))
	if page.Code != http.StatusOK ||
		!strings.Contains(page.Body.String(), `"run_finding_status":"failed"`) ||
		strings.Contains(page.Body.String(), "private verifier failure") ||
		strings.Contains(strings.ToLower(page.Body.String()), "mapping") {
		t.Fatalf("failed projection=%d %s", page.Code, page.Body.String())
	}

	retry := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		base+"/findings/status",
		map[string]any{"finding_ids": []string{findingID}},
	)
	if retry.Code != http.StatusAccepted ||
		!strings.Contains(retry.Body.String(), `"run_finding_status":"pending"`) ||
		!strings.Contains(retry.Body.String(), `"id":"`+findingID+`"`) {
		t.Fatalf("retry=%d %s", retry.Code, retry.Body.String())
	}
	reset, _, resetErr := store.Get(state.Repository)
	if resetErr != nil {
		t.Fatal(resetErr)
	}
	job := reset.MappingJobs[0]
	if job.Attempts != 0 || job.Error != "" || job.State != repoaudit.RepositoryMappingPending {
		t.Fatalf("reset job=%#v", job)
	}
	detail := httptest.NewRecorder()
	mux.ServeHTTP(detail, httptest.NewRequest(
		http.MethodGet, base+"/findings/"+findingID, nil,
	))
	if detail.Code != http.StatusOK ||
		!strings.Contains(detail.Body.String(), `"run_finding_status":"pending"`) {
		t.Fatalf("pending detail=%d %s", detail.Code, detail.Body.String())
	}

	if _, err := store.ProcessPendingMappingJobs(
		t.Context(),
		state.Repository,
		repoaudit.RepositoryMappingProcessOptions{
			DefaultBranchVerified: func(context.Context, repoaudit.Finding) (bool, error) {
				return true, nil
			},
		},
	); err != nil {
		t.Fatal(err)
	}
	associated, _, associatedErr := store.Get(state.Repository)
	if associatedErr != nil || len(associated.RepositoryFindings) != 1 {
		t.Fatalf("associated=%#v err=%v", associated.RepositoryFindings, associatedErr)
	}
	aggregateDetail := httptest.NewRecorder()
	mux.ServeHTTP(aggregateDetail, httptest.NewRequest(
		http.MethodGet,
		base+"/repository-findings/"+associated.RepositoryFindings[0].ID,
		nil,
	))
	if aggregateDetail.Code != http.StatusOK ||
		!strings.Contains(aggregateDetail.Body.String(), `"run_finding_status":"associated_new"`) {
		t.Fatalf("aggregate detail=%d %s", aggregateDetail.Code, aggregateDetail.Body.String())
	}
	conflict := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		base+"/findings/status",
		map[string]any{"finding_ids": []string{findingID}},
	)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("associated retry=%d %s", conflict.Code, conflict.Body.String())
	}
}

func TestRepositoryReviewRunFindingStatusRetryRouteErrors(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	state := seedRepositoryReviewAPIState(t, workspace)
	automation := seedRepositoryReviewDetailAutomation(
		t,
		handler,
		state.Repository,
		state.Runs[0].ID,
	)
	base := "/api/repository-reviews/automations/" + automation.ID + "/findings/status"

	missingHeaders := httptest.NewRecorder()
	mux.ServeHTTP(missingHeaders, httptest.NewRequest(
		http.MethodPost,
		base,
		strings.NewReader(`{"finding_ids":[]}`),
	))
	if missingHeaders.Code < http.StatusBadRequest {
		t.Fatalf("missing-header retry status=%d", missingHeaders.Code)
	}

	queryResponse := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		base+"?unexpected=true",
		map[string]any{"finding_ids": []string{state.Findings[0].ID}},
	)
	if queryResponse.Code < http.StatusBadRequest {
		t.Fatalf("query retry status=%d", queryResponse.Code)
	}

	malformedRequest := httptest.NewRequest(
		http.MethodPost,
		base,
		strings.NewReader(`{`),
	)
	setRepositoryReviewMutationHeaders(malformedRequest)
	malformedResponse := httptest.NewRecorder()
	mux.ServeHTTP(malformedResponse, malformedRequest)
	if malformedResponse.Code < http.StatusBadRequest {
		t.Fatalf("malformed retry status=%d", malformedResponse.Code)
	}

	missingAutomation := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/automations/rra_missing/findings/status",
		map[string]any{"finding_ids": []string{state.Findings[0].ID}},
	)
	if missingAutomation.Code != http.StatusNotFound {
		t.Fatalf(
			"missing-automation retry status=%d body=%s",
			missingAutomation.Code,
			missingAutomation.Body.String(),
		)
	}

	automationStore, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	emptyAutomation := testRepositoryReviewAutomation()
	emptyAutomation.ID = automation.ID + "_empty"
	emptyAutomation.Repository = "owner/repository-without-ledger"
	emptyAutomation.RunIDs = []string{"run_without_ledger"}
	emptyAutomation, err = automationStore.CreateAutomation(t.Context(), emptyAutomation)
	if err != nil {
		t.Fatal(err)
	}
	missingLedger := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/automations/"+emptyAutomation.ID+"/findings/status",
		map[string]any{"finding_ids": []string{state.Findings[0].ID}},
	)
	if missingLedger.Code != http.StatusNotFound {
		t.Fatalf(
			"missing-ledger retry status=%d body=%s",
			missingLedger.Code,
			missingLedger.Body.String(),
		)
	}
}

func TestRepositoryReviewAggregateDetailSkipsUnassociatedOccurrenceBeforeLinkedIssue(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	state := seedRepositoryReviewGenerationFindings(t, workspace, 2)
	store := repoaudit.NewStore(workspace)
	occurrenceIDs := append([]string(nil), state.Runs[0].FindingIDs...)
	slices.Sort(occurrenceIDs)

	jobFor := func(findingID string) repoaudit.RepositoryMappingJob {
		t.Helper()
		for _, job := range state.MappingJobs {
			if job.ReviewFindingID == findingID {
				return job
			}
		}
		t.Fatalf("mapping job for %s not found", findingID)
		return repoaudit.RepositoryMappingJob{}
	}
	claim := func(findingID string) repoaudit.RepositoryMappingJob {
		t.Helper()
		claimedState, job, _, claimed, err := store.ClaimMappingJob(
			state.Repository,
			jobFor(findingID).ID,
			repoaudit.RepositoryMappingModelSnapshot{},
		)
		if err != nil || !claimed {
			t.Fatalf("claim %s: claimed=%v err=%v", findingID, claimed, err)
		}
		state = claimedState
		return job
	}

	firstJob := claim(occurrenceIDs[0])
	var aggregate repoaudit.RepositoryFinding
	var err error
	state, aggregate, err = store.CompleteMappingJob(
		state.Repository,
		repoaudit.RepositoryMappingCompletion{
			JobID: firstJob.ID, CreateMatchState: repoaudit.RepositoryMatchNew,
			DefaultBranchVerified: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	secondJob := claim(occurrenceIDs[1])
	state, aggregate, err = store.CompleteMappingJob(
		state.Repository,
		repoaudit.RepositoryMappingCompletion{
			JobID: secondJob.ID, RepositoryFindingID: aggregate.ID,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	var linkedOccurrence repoaudit.Finding
	for _, finding := range state.Findings {
		if finding.ID == occurrenceIDs[1] {
			linkedOccurrence = finding
			break
		}
	}
	linkedState, linkedIssue, err := store.LinkExistingIssue(repoaudit.ExistingIssueLink{
		Repository: state.Repository, FindingID: linkedOccurrence.ID,
		ExpectedFindingVersion: linkedOccurrence.Version,
		ExternalID:             "91",
		ExternalURL:            "https://github.com/owner/repo-batch/issues/91",
		Title:                  "Existing aggregate issue", Confirmed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	automation := seedRepositoryReviewDetailAutomation(
		t,
		handler,
		linkedState.Repository,
		linkedState.Runs[0].ID,
	)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations/"+automation.ID+"/repository-findings/"+aggregate.ID,
		nil,
	))
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"id":"`+linkedIssue.ID+`"`) ||
		!strings.Contains(response.Body.String(), `"can_unlink_issue":true`) {
		t.Fatalf("linked aggregate status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewAutomationCapabilitiesAreExplicitForUnavailableActions(t *testing.T) {
	t.Run("linked replacement", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		state := seedRepositoryReviewAPIState(t, workspace)
		state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
		store := repoaudit.NewStore(workspace)
		linkedState, _, err := store.LinkExistingIssue(repoaudit.ExistingIssueLink{
			Repository: state.Repository, FindingID: state.Findings[0].ID,
			ExpectedFindingVersion: state.Findings[0].Version,
			ExternalID:             "42",
			ExternalURL:            "https://github.com/owner/repo/issues/42",
			Title:                  "Existing issue", Confirmed: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		automation := seedRepositoryReviewDetailAutomation(
			t, handler, state.Repository, state.Runs[0].ID,
		)
		legacyResponse := httptest.NewRecorder()
		mux.ServeHTTP(legacyResponse, httptest.NewRequest(
			http.MethodGet,
			"/api/repository-reviews/automations/"+automation.ID+"/run-findings/"+
				linkedState.Findings[0].ID,
			nil,
		))
		if legacyResponse.Code != http.StatusNotFound {
			t.Fatalf(
				"modern finding exposed as legacy occurrence status=%d body=%s",
				legacyResponse.Code,
				legacyResponse.Body.String(),
			)
		}
		aggregateResponse := httptest.NewRecorder()
		mux.ServeHTTP(aggregateResponse, httptest.NewRequest(
			http.MethodGet,
			"/api/repository-reviews/automations/"+automation.ID+"/repository-findings/"+
				linkedState.Findings[0].RepositoryFindingID,
			nil,
		))
		aggregateBody := aggregateResponse.Body.String()
		for _, projection := range []string{
			`"id":"` + linkedState.Findings[0].IssueDraftID + `"`,
			`"can_unlink_issue":true`, `"can_replace_issue":true`,
		} {
			if aggregateResponse.Code != http.StatusOK || !strings.Contains(aggregateBody, projection) {
				t.Fatalf(
					"linked aggregate projection %s status=%d body=%s",
					projection,
					aggregateResponse.Code,
					aggregateBody,
				)
			}
		}
	})

	t.Run("legacy preview", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		state := seedRepositoryReviewAPIState(t, workspace)
		state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
		store := repoaudit.NewStore(workspace)
		_, draft, err := store.PrepareIssue(repoaudit.IssueDraftRequest{
			Repository: state.Repository, FindingIDs: []string{state.Findings[0].ID},
			ExpectedVersion: state.Version,
		})
		if err != nil {
			t.Fatal(err)
		}
		automation := seedRepositoryReviewDetailAutomation(
			t, handler, state.Repository, state.Runs[0].ID,
		)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(
			http.MethodGet,
			"/api/repository-reviews/automations/"+automation.ID+"/issues/"+draft.ID,
			nil,
		))
		body := response.Body.String()
		for _, capability := range []string{
			`"can_edit":true`, `"can_regenerate":false`,
		} {
			if response.Code != http.StatusOK || !strings.Contains(body, capability) {
				t.Fatalf("legacy preview capability %s status=%d body=%s", capability, response.Code, body)
			}
		}
	})
}

func TestRepositoryReviewRepositoryFindingLifecycleAndValidationRoutes(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	state := seedRepositoryReviewAPIState(t, workspace)
	store := repoaudit.NewStore(workspace)
	if _, err := store.ProcessPendingMappingJobs(
		t.Context(), state.Repository, repoaudit.RepositoryMappingProcessOptions{
			DefaultBranchVerified: func(context.Context, repoaudit.Finding) (bool, error) {
				return true, nil
			},
		},
	); err != nil {
		t.Fatal(err)
	}
	state, found, err := store.Get(state.Repository)
	if err != nil || !found || len(state.RepositoryFindings) != 1 {
		t.Fatalf("repository findings=%#v found=%v err=%v", state.RepositoryFindings, found, err)
	}
	aggregate := state.RepositoryFindings[0]
	automation := seedRepositoryReviewDetailAutomation(
		t, handler, state.Repository, state.Runs[0].ID,
	)
	page := httptest.NewRecorder()
	mux.ServeHTTP(page, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations/"+automation.ID+"/findings?scope=all",
		nil,
	))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), aggregate.ID) {
		t.Fatalf("repository findings page=%d %s", page.Code, page.Body.String())
	}
	path := "/api/repository-reviews/automations/" + automation.ID +
		"/repository-findings/" + aggregate.ID
	dismissed := repositoryReviewAutomationMutation(t, mux, http.MethodPatch, path, map[string]any{
		"lifecycle": "dismissed", "expected_version": aggregate.Version,
	})
	if dismissed.Code != http.StatusOK || !strings.Contains(dismissed.Body.String(), `"lifecycle":"dismissed"`) {
		t.Fatalf("dismiss=%d %s", dismissed.Code, dismissed.Body.String())
	}
	var dismissedPayload struct {
		Finding repoaudit.RepositoryFinding `json:"repository_finding"`
	}
	if err := json.Unmarshal(dismissed.Body.Bytes(), &dismissedPayload); err != nil {
		t.Fatal(err)
	}
	reopened := repositoryReviewAutomationMutation(t, mux, http.MethodPatch, path, map[string]any{
		"lifecycle": "open", "expected_version": dismissedPayload.Finding.Version,
	})
	if reopened.Code != http.StatusOK {
		t.Fatalf("reopen=%d %s", reopened.Code, reopened.Body.String())
	}
	validation := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/repository-findings/validations",
		map[string]any{"repository_finding_ids": []string{aggregate.ID}},
	)
	if validation.Code != http.StatusAccepted ||
		!strings.Contains(validation.Body.String(), `"state":"pending"`) {
		t.Fatalf("validation=%d %s", validation.Code, validation.Body.String())
	}
}

func TestRepositoryReviewRepositoryFindingDetailProjectsSafeFixCheckFailure(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	store := repoaudit.NewStore(workspace)
	if _, err := store.ProcessPendingMappingJobs(
		t.Context(), state.Repository, repoaudit.RepositoryMappingProcessOptions{
			DefaultBranchVerified: func(context.Context, repoaudit.Finding) (bool, error) {
				return true, nil
			},
		},
	); err != nil {
		t.Fatal(err)
	}
	state, found, err := store.Get(state.Repository)
	if err != nil || !found || len(state.RepositoryFindings) != 1 {
		t.Fatalf("repository findings=%#v found=%v err=%v", state.RepositoryFindings, found, err)
	}
	aggregate := state.RepositoryFindings[0]
	if _, _, reserveErr := store.ReserveValidationJobs(
		state.Repository,
		[]string{aggregate.ID},
		repoaudit.RepositoryMappingModelSnapshot{},
	); reserveErr != nil {
		t.Fatal(reserveErr)
	}
	secret := "provider token=secret private/repository/path.go"
	result, err := store.ProcessPendingValidationJobs(
		t.Context(),
		state.Repository,
		repoaudit.RepositoryValidationProcessOptions{
			Evidence: func(
				context.Context,
				repoaudit.RepositoryFinding,
				[]string,
			) ([]repoaudit.RepositoryValidationEvidence, error) {
				return []repoaudit.RepositoryValidationEvidence{{CurrentSource: "bounded source"}}, nil
			},
			Adjudicate: func(
				context.Context,
				repoaudit.RepositoryMappingModelSnapshot,
				repoaudit.RepositoryFinding,
				[]repoaudit.RepositoryValidationEvidence,
			) (repoaudit.RepositoryValidationDecision, error) {
				return repoaudit.RepositoryValidationDecision{}, repoaudit.WrapRepositoryValidationFailure(
					repoaudit.RepositoryValidationFailureCodeModelOutputInvalid,
					errors.New(secret),
				)
			},
			VerifyAncestry: func(context.Context, string) (bool, error) { return true, nil },
		},
	)
	if err != nil || result.Failed != 1 {
		t.Fatalf("validation result=%#v err=%v", result, err)
	}
	automation := seedRepositoryReviewDetailAutomation(
		t, handler, state.Repository, state.Runs[0].ID,
	)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations/"+automation.ID+
			"/repository-findings/"+aggregate.ID,
		nil,
	))
	var payload struct {
		Finding repoaudit.RepositoryFinding `json:"repository_finding"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	history := payload.Finding.ResolutionHistory
	if response.Code != http.StatusOK || payload.Finding.ValidationState != repoaudit.RepositoryValidationFailed ||
		len(history) != 1 || history[0].Failure == nil ||
		history[0].Failure.Code != repoaudit.RepositoryValidationFailureCodeModelOutputInvalid ||
		strings.Contains(response.Body.String(), secret) ||
		strings.Contains(response.Body.String(), "private/repository/path.go") {
		t.Fatalf("detail status=%d payload=%#v body=%s", response.Code, payload, response.Body.String())
	}
}

func TestRepositoryReviewReportCurrentMembershipAndEmptyActiveLedger(t *testing.T) {
	now := time.Now().UTC()
	state := repoaudit.RepositoryState{
		Findings: []repoaudit.Finding{
			{ID: "historical"},
			{ID: "current"},
			{ID: "context-current", ContextIDs: []string{"context-new"}},
			{ID: "context-old", ContextIDs: []string{"context-old"}},
		},
		Runs: []repoaudit.ReviewRun{
			{ID: "run-old", FindingIDs: []string{"historical"}, CompletedAt: now.Add(-time.Hour)},
			{ID: "run-new", FindingIDs: []string{"current"}, CompletedAt: now.Add(time.Minute)},
		},
		Contexts: []repoaudit.FindingContext{
			{ID: "context-old", RunID: "run-old", CreatedAt: now.Add(-time.Hour)},
			{ID: "context-new", RunID: "run-new", CreatedAt: now.Add(time.Minute)},
		},
	}
	automation := repoaudit.RepositoryReviewAutomation{
		RunIDs: []string{"run-old", "run-new"}, StartedAt: now,
	}
	current := repositoryReviewReportFindings(automation, state, "current")
	all := repositoryReviewReportFindings(automation, state, "all")
	if len(current) != 2 || current[0].ID != "current" || current[1].ID != "context-current" || len(all) != 4 {
		t.Fatalf("current=%#v all=%#v", current, all)
	}

	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	empty := testRepositoryReviewAutomation()
	empty.ID = "rra_empty_active"
	empty.Repository = "owner/empty"
	empty.Status = repoaudit.RepositoryReviewAutomationRunning
	empty.ActiveRunID = "run-empty"
	empty.RunIDs = []string{empty.ActiveRunID}
	empty.StartedAt = now
	empty, err = store.CreateAutomation(t.Context(), empty)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations/"+empty.ID+"/report?scope=current",
		nil,
	))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"findings":[]`) ||
		!strings.Contains(response.Body.String(), `"status":"running"`) {
		t.Fatalf("empty active report status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewAutomationIssueGenerationUsesSnapshottedWriter(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	state := seedRepositoryReviewAPIState(t, workspace)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)

	previous := runRepositoryReviewIssueWriter
	t.Cleanup(func() { runRepositoryReviewIssueWriter = previous })
	type capturedCall struct {
		model        string
		account      string
		instructions string
	}
	captured := make(chan capturedCall, 1)
	runRepositoryReviewIssueWriter = func(
		_ context.Context,
		_ *Handler,
		automation repoaudit.RepositoryReviewAutomation,
		_ repoaudit.Finding,
		_ []repoaudit.FindingContext,
		instructions string,
		account string,
	) (repositoryReviewIssueWriterResult, error) {
		captured <- capturedCall{
			model: automation.IssueWriterModel, account: account, instructions: instructions,
		}
		return repositoryReviewIssueWriterResult{
			Title: "AI issue title", Body: "Grounded evidence and provenance.", Labels: []string{"bug"},
		}, nil
	}

	response := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/issues/generations",
		map[string]any{
			"generation_id": "rrig_test", "finding_ids": []string{state.Findings[0].ID},
			"instructions_mode": "default",
		},
	)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "AI issue title") ||
		!strings.Contains(response.Body.String(), `"state":"editing"`) {
		t.Fatalf("generation status=%d body=%s", response.Code, response.Body.String())
	}
	call := <-captured
	if call.model != "cheap" || call.account != "api" ||
		!strings.Contains(call.instructions, "commit/blob provenance") {
		t.Fatalf("writer call=%#v", call)
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	updated, found, err := store.Get(state.Repository)
	if err != nil || !found || len(updated.IssueDrafts) != 1 ||
		updated.IssueDrafts[0].GeneratorModel != "cheap" ||
		updated.IssueDrafts[0].GeneratorAccount != "api" ||
		updated.Findings[0].IssueDraftID != updated.IssueDrafts[0].ID {
		t.Fatalf("durable generation=%#v found=%v err=%v", updated, found, err)
	}
	regenerated := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/issues/"+
			updated.IssueDrafts[0].ID+"/regenerate",
		map[string]any{"expected_version": updated.IssueDrafts[0].Version},
	)
	if regenerated.Code != http.StatusOK ||
		!strings.Contains(regenerated.Body.String(), `"state":"editing"`) {
		t.Fatalf("regeneration status=%d body=%s", regenerated.Code, regenerated.Body.String())
	}
	regenerationCall := <-captured
	if regenerationCall.model != "cheap" || regenerationCall.account != "api" {
		t.Fatalf("regeneration writer call=%#v", regenerationCall)
	}
}

func TestRepositoryReviewIssueDraftUsesCurrentAssignedProfileAndFreezesProvenance(t *testing.T) {
	_, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	state := seedRepositoryReviewAPIState(t, workspace)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	store := repoaudit.NewStore(workspace)
	profile, err := store.CreateProfile(t.Context(), repoaudit.RepositoryReviewProfile{
		Name: "Current issue policy", ReviewFocus: "Find bugs.", ReviewerModel: "cheap",
		IssueWriterModel: "cheap", IssuePrompt: "Initial issue presentation.", AccountRef: "api",
		AutoContinue: true, MaxFilesPerRun: 4, MaxContentBytes: 65536,
		MaxParallelChildren: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.ID = "rra_current_profile"
	automation.Repository = state.Repository
	automation.RunIDs = []string{state.Runs[0].ID}
	automation.StartedAt = time.Now().UTC().Add(-time.Hour)
	automation, err = repoaudit.MaterializeRepositoryReviewAutomation(profile, automation)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.CreateAutomation(t.Context(), automation); err != nil {
		t.Fatal(err)
	}
	profile, err = store.UpdateProfile(
		t.Context(),
		profile.ID,
		profile.Version,
		func(candidate *repoaudit.RepositoryReviewProfile) error {
			candidate.IssuePrompt = "Current assigned issue presentation."
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	previous := runRepositoryReviewIssueWriter
	t.Cleanup(func() { runRepositoryReviewIssueWriter = previous })
	captured := make(chan string, 1)
	runRepositoryReviewIssueWriter = func(
		_ context.Context,
		_ *Handler,
		_ repoaudit.RepositoryReviewAutomation,
		_ repoaudit.Finding,
		_ []repoaudit.FindingContext,
		instructions string,
		_ string,
	) (repositoryReviewIssueWriterResult, error) {
		captured <- instructions
		return repositoryReviewIssueWriterResult{
			Title: "Current-profile issue", Body: "Grounded diagnosis.", Labels: []string{"bug"},
		}, nil
	}
	response := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/issues/generations",
		map[string]any{
			"generation_id":     "rrig_current_profile",
			"finding_ids":       []string{state.Findings[0].ID},
			"instructions_mode": "default",
		},
	)
	if response.Code != http.StatusOK || <-captured != profile.IssuePrompt {
		t.Fatalf("generation status=%d body=%s", response.Code, response.Body.String())
	}
	updated, found, err := store.Get(state.Repository)
	if err != nil || !found || len(updated.IssueDrafts) != 1 {
		t.Fatalf("updated=%#v found=%v err=%v", updated, found, err)
	}
	draft := updated.IssueDrafts[0]
	if draft.GeneratorProfileID != profile.ID || draft.GeneratorProfileVersion != profile.Version ||
		draft.ResolvedInstructions != profile.IssuePrompt || draft.GeneratorModel != "cheap" ||
		draft.GeneratorAccount != "api" {
		t.Fatalf("draft provenance=%#v", draft)
	}
}

func TestRepositoryReviewDirectPostGeneratesThenPublishesWithoutConfirmationPayload(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	state := seedRepositoryReviewAPIState(t, workspace)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	automation := seedRepositoryReviewDetailAutomation(
		t, handler, state.Repository, state.Runs[0].ID,
	)
	previous := runRepositoryReviewIssueWriter
	t.Cleanup(func() { runRepositoryReviewIssueWriter = previous })
	runRepositoryReviewIssueWriter = func(
		context.Context,
		*Handler,
		repoaudit.RepositoryReviewAutomation,
		repoaudit.Finding,
		[]repoaudit.FindingContext,
		string,
		string,
	) (repositoryReviewIssueWriterResult, error) {
		return repositoryReviewIssueWriterResult{
			Title: "Direct issue", Body: "Grounded diagnosis and provenance.", Labels: []string{"bug"},
		}, nil
	}
	installEventProxyStubs(t, func(request *http.Request, _ time.Duration) (*http.Response, error) {
		if !strings.Contains(request.URL.Path, "/issue-drafts/") ||
			!strings.HasSuffix(request.URL.Path, "/publish") {
			t.Fatalf("unexpected direct-post upstream path %q", request.URL.Path)
		}
		return eventUpstreamResponse(http.StatusOK, `{"outcome":"posted"}`), nil
	})
	response := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/findings/"+
			state.Findings[0].ID+"/post",
		map[string]any{"expected_version": state.Findings[0].Version},
	)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"outcome":"posted"`) ||
		!strings.Contains(response.Body.String(), "Direct issue") {
		t.Fatalf("direct post status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewIssueGenerationBoundsConcurrencyAndRetriesOnlyFailure(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	state := seedRepositoryReviewGenerationFindings(t, workspace, 5)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)

	previous := runRepositoryReviewIssueWriter
	t.Cleanup(func() { runRepositoryReviewIssueWriter = previous })
	var active, maximum, calls atomic.Int64
	started := make(chan struct{}, 10)
	release := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	runRepositoryReviewIssueWriter = func(
		_ context.Context,
		_ *Handler,
		_ repoaudit.RepositoryReviewAutomation,
		finding repoaudit.Finding,
		_ []repoaudit.FindingContext,
		_ string,
		_ string,
	) (repositoryReviewIssueWriterResult, error) {
		calls.Add(1)
		inFlight := active.Add(1)
		defer active.Add(-1)
		for {
			observed := maximum.Load()
			if inFlight <= observed || maximum.CompareAndSwap(observed, inFlight) {
				break
			}
		}
		started <- struct{}{}
		<-release
		if finding.Title == "Finding 5" {
			return repositoryReviewIssueWriterResult{}, errors.New("provider detail must stay private")
		}
		return repositoryReviewIssueWriterResult{
			Title: finding.Title, Body: "Grounded evidence.", Labels: []string{"bug"},
		}, nil
	}

	body, err := json.Marshal(map[string]any{
		"generation_id": "rrig_batch", "finding_ids": state.Runs[0].FindingIDs,
		"instructions_mode": "default",
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/issues/generations",
		strings.NewReader(string(body)),
	)
	setRepositoryReviewMutationHeaders(request)
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		mux.ServeHTTP(response, request)
		close(done)
	}()
	for range 4 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("four issue-writer calls did not start")
		}
	}
	select {
	case <-started:
		t.Fatal("a fifth issue-writer call exceeded the concurrency limit")
	case <-time.After(25 * time.Millisecond):
	}
	releaseOnce.Do(func() { close(release) })
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("issue generation did not complete")
	}
	if response.Code != http.StatusOK || maximum.Load() != 4 || calls.Load() != 5 ||
		!strings.Contains(response.Body.String(), `"state":"failed"`) ||
		strings.Contains(response.Body.String(), "provider detail must stay private") {
		t.Fatalf(
			"generation status=%d max=%d calls=%d body=%s",
			response.Code, maximum.Load(), calls.Load(), response.Body.String(),
		)
	}

	beforeReplay := calls.Load()
	replay := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/issues/generations",
		map[string]any{
			"generation_id": "rrig_batch", "finding_ids": state.Runs[0].FindingIDs,
			"instructions_mode": "default",
		},
	)
	if replay.Code != http.StatusOK || calls.Load()-beforeReplay != 1 {
		t.Fatalf(
			"idempotent replay status=%d new_calls=%d body=%s",
			replay.Code, calls.Load()-beforeReplay, replay.Body.String(),
		)
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	updated, found, err := store.Get(state.Repository)
	if err != nil || !found || len(updated.IssueDrafts) != 5 {
		t.Fatalf("replayed drafts=%d found=%v err=%v", len(updated.IssueDrafts), found, err)
	}
}

func TestRepositoryReviewInterruptedGenerationResumesWithSameGenerationID(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ModelList = append(cfg.ModelList, &config.ModelConfig{
		ModelName: "old-account", Provider: "openai", Model: "openai/old", Enabled: true,
	})
	if saveErr := config.SaveConfig(handler.configPath, cfg); saveErr != nil {
		t.Fatal(saveErr)
	}
	state := seedRepositoryReviewAPIState(t, workspace)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	generationRequest := repoaudit.IssueGenerationRequest{
		Repository: state.Repository, FindingID: state.Findings[0].ID,
		GenerationID:         "rrig_interrupted",
		ResolvedInstructions: "Persisted presentation instructions.",
		InstructionsMode:     repoaudit.IssueDraftInstructionsDefault,
		GeneratorModel:       "quality", GeneratorAccount: "old-account",
	}
	_, draft, reserved, err := store.ReserveIssueGeneration(generationRequest)
	if err != nil || !reserved || draft.State != repoaudit.IssueDraftGenerating {
		t.Fatalf("interrupted reservation=%#v reserved=%v err=%v", draft, reserved, err)
	}
	_, activeDraft, claimed, err := claimRepositoryReviewIssueGeneration(store, generationRequest)
	if err != nil || !claimed {
		t.Fatalf("active generation claim=%#v claimed=%v err=%v", activeDraft, claimed, err)
	}
	defer releaseRepositoryReviewIssueGeneration(activeDraft)
	previous := runRepositoryReviewIssueWriter
	t.Cleanup(func() { runRepositoryReviewIssueWriter = previous })
	var calls atomic.Int64
	var capturedModel, capturedAccount, capturedInstructions string
	runRepositoryReviewIssueWriter = func(
		_ context.Context,
		_ *Handler,
		writerAutomation repoaudit.RepositoryReviewAutomation,
		_ repoaudit.Finding,
		_ []repoaudit.FindingContext,
		instructions string,
		account string,
	) (repositoryReviewIssueWriterResult, error) {
		calls.Add(1)
		capturedModel = writerAutomation.IssueWriterModel
		capturedAccount = account
		capturedInstructions = instructions
		return repositoryReviewIssueWriterResult{
			Title: "Recovered preview", Body: "Grounded evidence.", Labels: []string{"bug"},
		}, nil
	}
	inProgress := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/issues/"+draft.ID+"/regenerate",
		map[string]any{"expected_version": draft.Version},
	)
	if inProgress.Code != http.StatusOK || calls.Load() != 0 ||
		!strings.Contains(inProgress.Body.String(), `"state":"generating"`) {
		t.Fatalf(
			"coalesced retry status=%d calls=%d body=%s",
			inProgress.Code, calls.Load(), inProgress.Body.String(),
		)
	}
	// Releasing the in-memory owner simulates process loss: the durable
	// reservation remains generating, and the same generation ID can be claimed
	// by the next process/request without creating a second draft.
	releaseRepositoryReviewIssueGeneration(activeDraft)
	response := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/issues/"+draft.ID+"/regenerate",
		map[string]any{"expected_version": draft.Version},
	)
	if response.Code != http.StatusOK || calls.Load() != 1 ||
		!strings.Contains(response.Body.String(), `"generation_id":"rrig_interrupted"`) ||
		!strings.Contains(response.Body.String(), `"state":"editing"`) ||
		capturedModel != "quality" || capturedAccount != "old-account" ||
		capturedInstructions != "Persisted presentation instructions." {
		t.Fatalf(
			"interrupted retry status=%d calls=%d model=%q account=%q instructions=%q body=%s",
			response.Code, calls.Load(), capturedModel, capturedAccount,
			capturedInstructions, response.Body.String(),
		)
	}
}

func TestRepositoryReviewIssueWriterRequestIsPrivateEphemeralAndStructured(t *testing.T) {
	canary := repoaudit.NewRepositoryReviewCampaignID()
	request := repositoryReviewIssueWriterAgentRequest(
		repoaudit.RepositoryReviewAutomation{IssueWriterModel: "writer"},
		repoaudit.Finding{ID: "finding", CampaignID: canary},
		[]repoaudit.FindingContext{{ID: "context", CampaignID: canary}},
		"instructions", "account",
	)
	if request.Model != "writer" || request.AccountRef != "account" ||
		!request.EphemeralSession || request.History != "none" || request.Cache != "none" ||
		request.Tools != workflows.AgentToolsNone || !request.PrivateContext ||
		request.IsolatedSystemPrompt != repositoryReviewIssueWriterSystemPrompt ||
		request.Output == nil || request.Output.Format != "json" ||
		request.Output.Schema["additionalProperties"] != false {
		t.Fatalf("issue writer request=%#v", request)
	}
	if strings.Contains(request.Prompt, canary) || strings.Contains(request.Prompt, "campaign_id") {
		t.Fatalf("issue writer prompt exposed campaign authority: %s", request.Prompt)
	}
	invalid := workflows.ValidateAgentStructuredOutput(
		`{"title":"Bug","body":"Evidence","labels":["bug"],"fix":"change the code"}`,
		request.Output,
	)
	if invalid.Valid || !strings.Contains(invalid.Error, "fix") {
		t.Fatalf("extra issue-writer field was accepted: %#v", invalid)
	}
}

func TestRepositoryReviewIssueWriterFailsClosedAfterAliasBecomesAgentic(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ModelList[0].Provider = "codex-cli"
	cfg.ModelList[0].Model = "codex-cli/gpt-5"
	cfg.ModelAliases[0].Model = "codex-cli/gpt-5"
	cfg.ModelAliases[0].AccountOverrides = nil
	if saveErr := config.SaveConfig(handler.configPath, cfg); saveErr != nil {
		t.Fatal(saveErr)
	}
	_, err = defaultRunRepositoryReviewIssueWriter(
		t.Context(), handler,
		repoaudit.RepositoryReviewAutomation{
			IssueWriterModel: "cheap", EffectiveAccountRef: "api",
		},
		repoaudit.Finding{}, nil, repositoryReviewDefaultIssueInstructions, "api",
	)
	if err == nil || !strings.Contains(err.Error(), "agentic CLI") {
		t.Fatalf("agentic issue writer alias was not rejected: %v", err)
	}
}

func TestRepositoryReviewAutomationPublishUsesProtectedLedgerRoute(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	state := seedRepositoryReviewAPIState(t, workspace)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	withDraft, draft, reserved, err := store.ReserveIssueGeneration(repoaudit.IssueGenerationRequest{
		Repository: state.Repository, FindingID: state.Findings[0].ID, GenerationID: "rrig_publish",
		ResolvedInstructions: repositoryReviewDefaultIssueInstructions,
		InstructionsMode:     repoaudit.IssueDraftInstructionsDefault,
		GeneratorModel:       "cheap", GeneratorAccount: "api",
	})
	if err != nil || !reserved {
		t.Fatal(err)
	}
	_, draft, err = store.CompleteIssueGeneration(
		state.Repository, draft.ID, draft.GenerationID, "Issue", "Evidence", []string{"bug"}, "",
	)
	if err != nil {
		t.Fatal(err)
	}
	var capturedPath, capturedBody string
	installEventProxyStubs(t, func(request *http.Request, _ time.Duration) (*http.Response, error) {
		capturedPath = request.URL.Path
		var body map[string]any
		_ = json.NewDecoder(request.Body).Decode(&body)
		encoded, _ := json.Marshal(body)
		capturedBody = string(encoded)
		return eventUpstreamResponse(http.StatusOK, `{"draft":{"id":"`+draft.ID+`","state":"posted"}}`), nil
	})
	response := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/issues/"+draft.ID+"/publish",
		map[string]any{"expected_version": draft.Version, "confirmed": true},
	)
	if response.Code != http.StatusOK ||
		capturedPath != "/runtime/repository-reviews/"+withDraft.ID+"/issue-drafts/"+draft.ID+"/publish" ||
		capturedBody != `{"expected_version":2}` {
		t.Fatalf(
			"publish status=%d path=%q request=%s response=%s",
			response.Code, capturedPath, capturedBody, response.Body.String(),
		)
	}
}

func TestRepositoryReviewAutomationBatchPublishReportsPartialSelectionFailures(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	state := seedRepositoryReviewGenerationFindings(t, workspace, 2)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	drafts := make([]repoaudit.IssueDraft, 0, 2)
	for index, findingID := range state.Runs[0].FindingIDs {
		_, draft, _, reserveErr := store.ReserveIssueGeneration(repoaudit.IssueGenerationRequest{
			Repository: state.Repository, FindingID: findingID,
			GenerationID:         fmt.Sprintf("rrig_publish_%d", index),
			ResolvedInstructions: repositoryReviewDefaultIssueInstructions,
			InstructionsMode:     repoaudit.IssueDraftInstructionsDefault,
			GeneratorModel:       "cheap", GeneratorAccount: "api",
		})
		if reserveErr != nil {
			t.Fatal(reserveErr)
		}
		_, draft, completeErr := store.CompleteIssueGeneration(
			state.Repository, draft.ID, draft.GenerationID,
			fmt.Sprintf("Issue %d", index+1), "Evidence", []string{"bug"}, "",
		)
		if completeErr != nil {
			t.Fatal(completeErr)
		}
		drafts = append(drafts, draft)
	}
	var calls atomic.Int64
	installEventProxyStubs(t, func(_ *http.Request, _ time.Duration) (*http.Response, error) {
		calls.Add(1)
		return eventUpstreamResponse(
			http.StatusOK,
			`{"draft":{"id":"`+drafts[0].ID+`","state":"posted"}}`,
		), nil
	})
	response := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/issues/publish",
		map[string]any{
			"confirmed": true,
			"issues": []map[string]any{
				{"id": drafts[0].ID, "expected_version": drafts[0].Version},
				{"id": drafts[1].ID, "expected_version": drafts[1].Version + 1},
			},
		},
	)
	if response.Code != http.StatusOK || calls.Load() != 1 ||
		!strings.Contains(response.Body.String(), `"outcome":"posted"`) ||
		!strings.Contains(response.Body.String(), `"code":"stale_repository_review"`) {
		t.Fatalf(
			"partial publish status=%d calls=%d body=%s",
			response.Code, calls.Load(), response.Body.String(),
		)
	}
}

func TestRepositoryReviewIssueLinkActionsUseProtectedAutomationRoutes(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	state := seedRepositoryReviewAPIState(t, workspace)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
	var captured []string
	installEventProxyStubs(t, func(request *http.Request, _ time.Duration) (*http.Response, error) {
		captured = append(captured, request.Method+" "+request.URL.Path)
		return eventUpstreamResponse(http.StatusOK, `{"candidates":[]}`), nil
	})
	base := "/api/repository-reviews/automations/" + automation.ID +
		"/findings/" + state.Findings[0].ID + "/issue-link"
	candidates := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost, base+"/candidates",
		map[string]any{"expected_version": state.Findings[0].Version},
	)
	link := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost, base,
		map[string]any{
			"issue_url":        "https://github.com/owner/repo/issues/12",
			"expected_version": state.Findings[0].Version, "confirmed": true,
		},
	)
	if candidates.Code != http.StatusOK || link.Code != http.StatusOK || len(captured) != 2 ||
		captured[0] != "POST /runtime/repository-reviews/automations/"+automation.ID+
			"/findings/"+state.Findings[0].ID+"/issue-link/candidates" ||
		captured[1] != "POST /runtime/repository-reviews/automations/"+automation.ID+
			"/findings/"+state.Findings[0].ID+"/issue-link" {
		t.Fatalf(
			"candidate=%d link=%d captured=%#v",
			candidates.Code, link.Code, captured,
		)
	}
}

func TestRepositoryReviewAutomationIssueRoutesLifecycleAndPaging(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	state := seedRepositoryReviewGenerationFindings(t, workspace, 2)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
	store := repoaudit.NewStore(workspace)
	drafts := make([]repoaudit.IssueDraft, 0, 2)
	for index, findingID := range state.Runs[0].FindingIDs {
		_, draft, reserved, err := store.ReserveIssueGeneration(repoaudit.IssueGenerationRequest{
			Repository: state.Repository, FindingID: findingID,
			GenerationID:         fmt.Sprintf("rrig_lifecycle_%d", index),
			ResolvedInstructions: repositoryReviewDefaultIssueInstructions,
			InstructionsMode:     repoaudit.IssueDraftInstructionsDefault,
			GeneratorModel:       "cheap", GeneratorAccount: "api",
		})
		if err != nil || !reserved {
			t.Fatalf("reserve %d: reserved=%v err=%v", index, reserved, err)
		}
		_, draft, err = store.CompleteIssueGeneration(
			state.Repository, draft.ID, draft.GenerationID,
			fmt.Sprintf("Issue %d", index+1), "Evidence", []string{"bug"}, "",
		)
		if err != nil {
			t.Fatal(err)
		}
		drafts = append(drafts, draft)
	}
	base := "/api/repository-reviews/automations/" + automation.ID + "/issues"

	page := httptest.NewRecorder()
	mux.ServeHTTP(page, httptest.NewRequest(http.MethodGet, base+"?offset=0&limit=1", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), `"total":2`) ||
		!strings.Contains(page.Body.String(), `"next_offset":1`) {
		t.Fatalf("issue page=%d %s", page.Code, page.Body.String())
	}
	filtered := httptest.NewRecorder()
	mux.ServeHTTP(filtered, httptest.NewRequest(
		http.MethodGet, base+"?generation_id="+drafts[1].GenerationID, nil,
	))
	if filtered.Code != http.StatusOK || !strings.Contains(filtered.Body.String(), drafts[1].ID) ||
		strings.Contains(filtered.Body.String(), drafts[0].ID) {
		t.Fatalf("filtered issues=%d %s", filtered.Code, filtered.Body.String())
	}
	for _, target := range []string{
		base + "?unknown=1", base + "?offset=nope", base + "?limit=201",
		base + "?generation_id=" + strings.Repeat("x", 257),
	} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid issue page %q=%d %s", target, response.Code, response.Body.String())
		}
	}

	detailPath := base + "/" + drafts[0].ID
	detail := httptest.NewRecorder()
	mux.ServeHTTP(detail, httptest.NewRequest(http.MethodGet, detailPath, nil))
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), `"can_edit":true`) {
		t.Fatalf("issue detail=%d %s", detail.Code, detail.Body.String())
	}
	stale := repositoryReviewAutomationMutation(t, mux, http.MethodPatch, detailPath, map[string]any{
		"title": "Edited", "body": "Edited body", "labels": []string{"bug"},
		"expected_version": drafts[0].Version + 1,
	})
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale edit=%d %s", stale.Code, stale.Body.String())
	}
	malformed := repositoryReviewAutomationMutation(t, mux, http.MethodPatch, detailPath, map[string]any{
		"title": "Edited",
	})
	if malformed.Code != http.StatusConflict && malformed.Code != http.StatusBadRequest {
		t.Fatalf("malformed edit=%d %s", malformed.Code, malformed.Body.String())
	}
	edited := repositoryReviewAutomationMutation(t, mux, http.MethodPatch, detailPath, map[string]any{
		"title": "Edited", "body": "Edited body", "labels": []string{"bug", "triage"},
		"expected_version": drafts[0].Version,
	})
	if edited.Code != http.StatusOK || !strings.Contains(edited.Body.String(), `"title":"Edited"`) {
		t.Fatalf("edit=%d %s", edited.Code, edited.Body.String())
	}
	var editedPayload struct {
		Issue repoaudit.IssueDraft `json:"issue"`
	}
	if err := json.Unmarshal(edited.Body.Bytes(), &editedPayload); err != nil {
		t.Fatal(err)
	}
	unconfirmed := repositoryReviewAutomationMutation(t, mux, http.MethodDelete, detailPath, map[string]any{
		"expected_version": editedPayload.Issue.Version, "confirmed": false,
	})
	if unconfirmed.Code != http.StatusBadRequest {
		t.Fatalf("unconfirmed delete=%d %s", unconfirmed.Code, unconfirmed.Body.String())
	}
	staleDelete := repositoryReviewAutomationMutation(t, mux, http.MethodDelete, detailPath, map[string]any{
		"expected_version": editedPayload.Issue.Version + 1, "confirmed": true,
	})
	if staleDelete.Code != http.StatusConflict {
		t.Fatalf("stale delete=%d %s", staleDelete.Code, staleDelete.Body.String())
	}
	deleted := repositoryReviewAutomationMutation(t, mux, http.MethodDelete, detailPath, map[string]any{
		"expected_version": editedPayload.Issue.Version, "confirmed": true,
	})
	if deleted.Code != http.StatusOK || !strings.Contains(deleted.Body.String(), `"outcome":"deleted"`) {
		t.Fatalf("delete=%d %s", deleted.Code, deleted.Body.String())
	}
	missing := httptest.NewRecorder()
	mux.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, detailPath, nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("deleted detail=%d %s", missing.Code, missing.Body.String())
	}
}

func TestRepositoryReviewAutomationDetailRequestFailureBoundaries(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	state := seedRepositoryReviewAPIState(t, workspace)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
	base := "/api/repository-reviews/automations/" + automation.ID

	for _, target := range []string{
		"/api/repository-reviews/automations/rra_missing",
		base + "/report?scope=invalid", base + "/report?unexpected=1",
		base + "/report?offset=nope", base + "/report?limit=201",
		base + "/findings/missing", base + "/issues/missing",
	} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusBadRequest && response.Code != http.StatusNotFound {
			t.Fatalf("failure boundary %q=%d %s", target, response.Code, response.Body.String())
		}
	}
	findingPath := base + "/findings/" + state.Findings[0].ID
	badBody := repositoryReviewAutomationMutation(t, mux, http.MethodPatch, findingPath, map[string]any{
		"status": "invalid", "expected_version": state.Findings[0].Version,
	})
	if badBody.Code != http.StatusBadRequest {
		t.Fatalf("invalid status=%d %s", badBody.Code, badBody.Body.String())
	}
	stale := repositoryReviewAutomationMutation(t, mux, http.MethodPatch, findingPath, map[string]any{
		"status": "dismissed", "expected_version": state.Findings[0].Version + 1,
	})
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale status=%d %s", stale.Code, stale.Body.String())
	}
	crossSite := httptest.NewRequest(
		http.MethodPatch, "http://launcher.local"+findingPath,
		strings.NewReader(`{"status":"dismissed","expected_version":1}`),
	)
	crossSite.Header.Set("Content-Type", "application/json")
	crossSite.Header.Set("Sec-Fetch-Site", "cross-site")
	crossResponse := httptest.NewRecorder()
	mux.ServeHTTP(crossResponse, crossSite)
	if crossResponse.Code != http.StatusBadRequest {
		t.Fatalf("cross-site status=%d %s", crossResponse.Code, crossResponse.Body.String())
	}
}

func TestRepositoryReviewIssueGenerationRequestAndRegenerationBoundaries(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	state := seedRepositoryReviewAPIState(t, workspace)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
	generationPath := "/api/repository-reviews/automations/" + automation.ID + "/issues/generations"
	validFinding := state.Findings[0].ID
	cases := []map[string]any{
		{},
		{"generation_id": "", "finding_ids": []string{validFinding}},
		{"generation_id": strings.Repeat("x", 257), "finding_ids": []string{validFinding}},
		{"generation_id": "rrig", "finding_ids": []string{}},
		{"generation_id": "rrig", "finding_ids": []string{""}},
		{"generation_id": "rrig", "finding_ids": []string{validFinding, validFinding}},
		{"generation_id": "rrig", "finding_ids": []string{validFinding}, "instructions_mode": "bad"},
		{"generation_id": "rrig", "finding_ids": []string{validFinding}, "instructions_mode": "custom"},
		{
			"generation_id": "rrig", "finding_ids": []string{validFinding},
			"instructions_mode": "custom", "instructions": strings.Repeat("x", 16<<10),
		},
	}
	for index, body := range cases {
		response := repositoryReviewAutomationMutation(t, mux, http.MethodPost, generationPath, body)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid generation %d=%d %s", index, response.Code, response.Body.String())
		}
	}
	missingAutomation := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/rra_missing/issues/generations",
		map[string]any{"generation_id": "rrig", "finding_ids": []string{validFinding}},
	)
	if missingAutomation.Code != http.StatusNotFound {
		t.Fatalf("missing automation generation=%d %s", missingAutomation.Code, missingAutomation.Body.String())
	}
	store := repoaudit.NewStore(workspace)
	empty := testRepositoryReviewAutomation()
	empty.ID = "rra_generation_empty"
	empty.Repository = "owner/no-ledger"
	empty, err := store.CreateAutomation(t.Context(), empty)
	if err != nil {
		t.Fatal(err)
	}
	missingLedger := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+empty.ID+"/issues/generations",
		map[string]any{"generation_id": "rrig", "finding_ids": []string{validFinding}},
	)
	if missingLedger.Code != http.StatusNotFound {
		t.Fatalf("missing ledger generation=%d %s", missingLedger.Code, missingLedger.Body.String())
	}

	_, draft, reserved, err := store.ReserveIssueGeneration(repoaudit.IssueGenerationRequest{
		Repository: state.Repository, FindingID: validFinding, GenerationID: "rrig_regen_bound",
		ResolvedInstructions: repositoryReviewDefaultIssueInstructions,
		InstructionsMode:     repoaudit.IssueDraftInstructionsDefault,
		GeneratorModel:       "cheap", GeneratorAccount: "api",
	})
	if err != nil || !reserved {
		t.Fatalf("reserve: %v/%v", reserved, err)
	}
	_, draft, err = store.CompleteIssueGeneration(
		state.Repository, draft.ID, draft.GenerationID, "Preview", "Evidence", []string{"bug"}, "",
	)
	if err != nil {
		t.Fatal(err)
	}
	regenPath := "/api/repository-reviews/automations/" + automation.ID + "/issues/" + draft.ID + "/regenerate"
	badRegen := repositoryReviewAutomationMutation(t, mux, http.MethodPost, regenPath, map[string]any{})
	if badRegen.Code != http.StatusConflict {
		t.Fatalf("missing regen version=%d %s", badRegen.Code, badRegen.Body.String())
	}
	staleRegen := repositoryReviewAutomationMutation(t, mux, http.MethodPost, regenPath, map[string]any{
		"expected_version": draft.Version + 1,
	})
	if staleRegen.Code != http.StatusConflict {
		t.Fatalf("stale regen=%d %s", staleRegen.Code, staleRegen.Body.String())
	}
	previousRandom := readRepositoryReviewIssueGenerationRandom
	readRepositoryReviewIssueGenerationRandom = func([]byte) (int, error) {
		return 0, errors.New("entropy unavailable")
	}
	randomFailure := repositoryReviewAutomationMutation(t, mux, http.MethodPost, regenPath, map[string]any{
		"expected_version": draft.Version,
	})
	readRepositoryReviewIssueGenerationRandom = previousRandom
	if randomFailure.Code != http.StatusInternalServerError {
		t.Fatalf("regen entropy failure=%d %s", randomFailure.Code, randomFailure.Body.String())
	}
	missingRegen := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost, baseAutomationIssuePath(automation.ID, "missing")+"/regenerate",
		map[string]any{"expected_version": 1},
	)
	if missingRegen.Code != http.StatusNotFound {
		t.Fatalf("missing regen=%d %s", missingRegen.Code, missingRegen.Body.String())
	}
}

func TestRepositoryReviewDetailHelperBoundaryCoverage(t *testing.T) {
	if _, _, _, err := repositoryReviewReportPage(nil); err == nil {
		t.Fatal("nil report request succeeded")
	}
	if _, _, _, err := repositoryReviewIssuePage(nil); err == nil {
		t.Fatal("nil issue request succeeded")
	}
	for _, target := range []string{
		"/?generation_id=x&generation_id=y", "/?offset=-1", "/?limit=0", "/?limit=201",
	} {
		if _, _, _, err := repositoryReviewIssuePage(httptest.NewRequest(http.MethodGet, target, nil)); err == nil {
			t.Fatalf("issue page accepted %q", target)
		}
	}
	for _, target := range []string{
		"/?scope=current&scope=all", "/?scope=unknown", "/?offset=-1", "/?limit=0",
	} {
		if _, _, _, err := repositoryReviewReportPage(httptest.NewRequest(http.MethodGet, target, nil)); err == nil {
			t.Fatalf("report page accepted %q", target)
		}
	}
	identities := map[string][]string{
		"":                               nil,
		"/tmp/repo":                      {"/tmp/repo"},
		"git@github.com:Owner/Repo.git":  {"owner/repo", "git@github.com:Owner/Repo.git"},
		"git@example.com:Owner/Repo.git": {"git@example.com:Owner/Repo.git"},
		"https://User:secret@github.com/Owner/Repo.git?q=1#fragment": {
			"owner/repo", "https://github.com/Owner/Repo.git",
		},
		"Owner/Repo.git": {"owner/repo", "Owner/Repo.git"},
		"relative":       {"relative"},
	}
	for input, expected := range identities {
		if got := repositoryReviewAutomationLedgerIdentities(input); !slices.Equal(got, expected) {
			t.Fatalf("identities(%q)=%#v want %#v", input, got, expected)
		}
	}
	for _, value := range []string{
		"owner/repo", "owner/repo.name", "owner/repo_name",
	} {
		if !validRepositoryReviewGitHubIdentityAPI(value) {
			t.Fatalf("valid GitHub identity rejected: %q", value)
		}
	}
	for _, value := range []string{
		"", "Owner/repo", "owner", "owner/repo/extra", "/repo", "owner/.",
		strings.Repeat("x", 101) + "/repo", "owner/repo!",
	} {
		if validRepositoryReviewGitHubIdentityAPI(value) {
			t.Fatalf("invalid GitHub identity accepted: %q", value)
		}
	}
	if got := repositoryReviewGenerationFailure("finding", "generation_failed"); got["message"] != "generation failed" {
		t.Fatalf("generation failure=%#v", got)
	}
	if got := repositoryReviewResolvedIssueInstructions(
		repositoryReviewGenerationRequest{},
	); got != repositoryReviewDefaultIssueInstructions {
		t.Fatalf("default instructions=%q", got)
	}
	custom := repositoryReviewResolvedIssueInstructions(repositoryReviewGenerationRequest{
		InstructionsMode: repoaudit.IssueDraftInstructionsCustom, Instructions: "Use terse headings.",
	})
	if !strings.Contains(custom, "Use terse headings.") {
		t.Fatalf("custom instructions=%q", custom)
	}
	if id, err := newRepositoryReviewIssueGenerationID(); err != nil || !strings.HasPrefix(id, "rrig_") {
		t.Fatalf("new generation ID=%q err=%v", id, err)
	}
	previousRandom := readRepositoryReviewIssueGenerationRandom
	t.Cleanup(func() { readRepositoryReviewIssueGenerationRandom = previousRandom })
	readRepositoryReviewIssueGenerationRandom = func([]byte) (int, error) {
		return 0, errors.New("entropy unavailable")
	}
	if _, err := newRepositoryReviewIssueGenerationID(); err == nil {
		t.Fatal("generation ID entropy failure was ignored")
	}
	readRepositoryReviewIssueGenerationRandom = previousRandom
	if _, found := repositoryReviewFindingByID(repoaudit.RepositoryState{}, "missing"); found {
		t.Fatal("missing finding found")
	}
	if repositoryReviewGenerationCanUseOnlyExistingReservations(
		repoaudit.RepositoryState{}, []string{"missing"}, "rrig",
	) {
		t.Fatal("missing finding was treated as an existing reservation")
	}
	state := repoaudit.RepositoryState{
		Findings: []repoaudit.Finding{{ID: "finding", IssueDraftID: "missing"}},
	}
	if repositoryReviewGenerationCanUseOnlyExistingReservations(state, []string{"finding"}, "rrig") {
		t.Fatal("missing draft was treated as an existing reservation")
	}
	state.IssueDrafts = []repoaudit.IssueDraft{{
		ID: "missing", Origin: repoaudit.IssueDraftOriginLinked, GenerationID: "rrig",
	}}
	if repositoryReviewGenerationCanUseOnlyExistingReservations(state, []string{"finding"}, "rrig") {
		t.Fatal("linked draft was treated as an AI reservation")
	}
	state.IssueDrafts[0].Origin = repoaudit.IssueDraftOriginAIGenerated
	state.IssueDrafts[0].GenerationID = "other"
	if repositoryReviewGenerationCanUseOnlyExistingReservations(state, []string{"finding"}, "rrig") {
		t.Fatal("different generation was treated as the same reservation")
	}
	state.IssueDrafts[0].GenerationID = "rrig"
	if !repositoryReviewGenerationCanUseOnlyExistingReservations(state, []string{"finding"}, "rrig") {
		t.Fatal("matching AI reservation was not recognized")
	}
	state.IssueDrafts[0].AttemptGenerationID = "retry"
	if repositoryReviewGenerationCanUseOnlyExistingReservations(state, []string{"finding"}, "rrig") ||
		!repositoryReviewGenerationCanUseOnlyExistingReservations(state, []string{"finding"}, "retry") {
		t.Fatal("active regeneration attempt was not used as the reservation identity")
	}
	for name, test := range map[string]struct {
		outputs map[string]any
		valid   bool
	}{
		"invalid flag":  {outputs: map[string]any{}},
		"unmarshalable": {outputs: map[string]any{"structured_valid": true, "structured": make(chan int)}},
		"unknown field": {outputs: map[string]any{
			"structured_valid": true,
			"structured":       map[string]any{"title": "Title", "body": "Body", "labels": []string{}, "extra": true},
		}},
		"blank": {outputs: map[string]any{
			"structured_valid": true,
			"structured":       map[string]any{"title": "", "body": "Body", "labels": []string{}},
		}},
		"valid": {outputs: map[string]any{
			"structured_valid": true,
			"structured":       map[string]any{"title": " Title ", "body": " Body ", "labels": []string{"bug"}},
		}, valid: true},
	} {
		result, err := repositoryReviewIssueWriterResultFromOutputs(test.outputs)
		if test.valid {
			if err != nil || result.Title != "Title" || result.Body != "Body" {
				t.Fatalf("%s result=%#v err=%v", name, result, err)
			}
		} else if err == nil {
			t.Fatalf("%s invalid output succeeded: %#v", name, result)
		}
	}
}

func TestRepositoryReviewIssueGenerationClaimBoundaryCoverage(t *testing.T) {
	workspace := t.TempDir()
	state := seedRepositoryReviewGenerationFindings(t, workspace, 3)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	store := repoaudit.NewStore(workspace)
	requestFor := func(index int, generationID string) repoaudit.IssueGenerationRequest {
		return repoaudit.IssueGenerationRequest{
			Repository: state.Repository, FindingID: state.Runs[0].FindingIDs[index],
			GenerationID: generationID, ResolvedInstructions: repositoryReviewDefaultIssueInstructions,
			InstructionsMode: repoaudit.IssueDraftInstructionsDefault,
			GeneratorModel:   "cheap", GeneratorAccount: "api",
		}
	}

	firstRequest := requestFor(0, "rrig_claim_active")
	_, firstDraft, claimed, err := claimRepositoryReviewIssueGeneration(store, firstRequest)
	if err != nil || !claimed {
		t.Fatalf("first claim=%v draft=%#v err=%v", claimed, firstDraft, err)
	}
	_, replayDraft, replayClaimed, err := claimRepositoryReviewIssueGeneration(store, firstRequest)
	if err != nil || replayClaimed || replayDraft.ID != firstDraft.ID {
		t.Fatalf("active replay=%v draft=%#v err=%v", replayClaimed, replayDraft, err)
	}
	releaseRepositoryReviewIssueGeneration(firstDraft)
	releaseRepositoryReviewIssueGeneration(firstDraft) // idempotent cleanup covers the nil-release path.

	secondRequest := requestFor(1, "rrig_claim_locked")
	_, secondDraft, reserved, err := store.ReserveIssueGeneration(secondRequest)
	if err != nil || !reserved {
		t.Fatalf("second reserve=%v draft=%#v err=%v", reserved, secondDraft, err)
	}
	externalRelease, acquired, err := store.TryLockIssueGenerationAttempt(
		state.Repository, secondDraft.ID, secondDraft.AttemptGenerationID,
	)
	if err != nil || !acquired {
		t.Fatalf("external claim=%v err=%v", acquired, err)
	}
	_, lockedDraft, lockedClaimed, err := claimRepositoryReviewIssueGeneration(store, secondRequest)
	if err != nil || lockedClaimed || lockedDraft.ID != secondDraft.ID {
		t.Fatalf("locked replay=%v draft=%#v err=%v", lockedClaimed, lockedDraft, err)
	}
	externalRelease()

	thirdRequest := requestFor(2, "rrig_claim_failed")
	_, failedDraft, reserved, err := store.ReserveIssueGeneration(thirdRequest)
	if err != nil || !reserved {
		t.Fatalf("failed reserve=%v err=%v", reserved, err)
	}
	_, failedDraft, err = store.CompleteIssueGeneration(
		state.Repository, failedDraft.ID, failedDraft.AttemptGenerationID,
		"", "", nil, "safe failure",
	)
	if err != nil || failedDraft.State != repoaudit.IssueDraftFailed {
		t.Fatalf("failed completion=%#v err=%v", failedDraft, err)
	}
	_, retriedDraft, retried, err := claimRepositoryReviewIssueGeneration(store, thirdRequest)
	if err != nil || !retried || retriedDraft.State != repoaudit.IssueDraftGenerating {
		t.Fatalf("failed retry=%v draft=%#v err=%v", retried, retriedDraft, err)
	}
	releaseRepositoryReviewIssueGeneration(retriedDraft)

	// Settle the first generation, then exercise the dedicated regeneration
	// claim, active coalescing, and stale version error.
	_, firstDraft, reacquired, err := claimRepositoryReviewIssueGeneration(store, firstRequest)
	if err != nil || !reacquired {
		t.Fatalf("reacquire=%v err=%v", reacquired, err)
	}
	_, firstDraft, err = store.CompleteIssueGeneration(
		state.Repository, firstDraft.ID, firstDraft.AttemptGenerationID,
		"Preview", "Evidence", []string{"bug"}, "",
	)
	releaseRepositoryReviewIssueGeneration(firstDraft)
	if err != nil {
		t.Fatal(err)
	}
	regenRequest := firstRequest
	regenRequest.GenerationID = "rrig_regen_claim"
	regenRequest.ExpectedDraftVersion = firstDraft.Version
	_, regenerating, regenClaimed, err := claimRepositoryReviewIssueRegeneration(
		store, state.Repository, firstDraft.ID, regenRequest,
	)
	if err != nil || !regenClaimed {
		t.Fatalf("regen claim=%v draft=%#v err=%v", regenClaimed, regenerating, err)
	}
	_, coalesced, coalescedClaim, err := claimRepositoryReviewIssueRegeneration(
		store, state.Repository, firstDraft.ID, regenRequest,
	)
	if err != nil || coalescedClaim || coalesced.ID != firstDraft.ID {
		t.Fatalf("regen coalesced=%v draft=%#v err=%v", coalescedClaim, coalesced, err)
	}
	releaseRepositoryReviewIssueGeneration(regenerating)
	staleRequest := regenRequest
	staleRequest.GenerationID = "rrig_regen_stale"
	staleRequest.ExpectedDraftVersion = 1
	if _, _, _, claimErr := claimRepositoryReviewIssueRegeneration(
		store, state.Repository, firstDraft.ID, staleRequest,
	); !errors.Is(claimErr, repoaudit.ErrConflict) {
		t.Fatalf("stale regeneration error=%v", claimErr)
	}

	previousBegin := beginRepositoryReviewIssueRegeneration
	previousTryLock := tryLockRepositoryReviewIssueGenerationAttempt
	t.Cleanup(func() {
		beginRepositoryReviewIssueRegeneration = previousBegin
		tryLockRepositoryReviewIssueGenerationAttempt = previousTryLock
	})
	tryLockRepositoryReviewIssueGenerationAttempt = func(
		repoaudit.Store, string, string, string,
	) (func(), bool, error) {
		return nil, false, errors.New("injected attempt-lock failure")
	}
	if _, _, _, claimErr := claimRepositoryReviewIssueGeneration(store, secondRequest); claimErr == nil {
		t.Fatal("generation attempt-lock failure was ignored")
	}
	if _, _, _, claimErr := claimRepositoryReviewIssueRegeneration(
		store, state.Repository, firstDraft.ID, regenRequest,
	); claimErr == nil {
		t.Fatal("regeneration attempt-lock failure was ignored")
	}
	tryLockRepositoryReviewIssueGenerationAttempt = previousTryLock
	current, found, err := store.Get(state.Repository)
	if err != nil || !found {
		t.Fatalf("load failed retry state found=%v err=%v", found, err)
	}
	currentThird, _ := repositoryReviewIssueByID(current, retriedDraft.ID)
	_, _, err = store.CompleteIssueGeneration(
		state.Repository, currentThird.ID, currentThird.AttemptGenerationID,
		"", "", nil, "failed again",
	)
	if err != nil {
		t.Fatal(err)
	}
	beginRepositoryReviewIssueRegeneration = func(
		repoaudit.Store, string, string, repoaudit.IssueGenerationRequest,
	) (repoaudit.RepositoryState, repoaudit.IssueDraft, bool, error) {
		return repoaudit.RepositoryState{}, repoaudit.IssueDraft{}, false,
			errors.New("injected regeneration failure")
	}
	if _, _, _, err := claimRepositoryReviewIssueGeneration(store, thirdRequest); err == nil {
		t.Fatal("failed-draft regeneration failure was ignored")
	}
	beginRepositoryReviewIssueRegeneration = previousBegin

	unsafeStore := repoaudit.NewStore(filepath.Join(t.TempDir(), "missing", "workspace"))
	if _, _, _, err := claimRepositoryReviewIssueGeneration(unsafeStore, firstRequest); err == nil {
		t.Fatal("unsafe generation claim succeeded")
	}
}

func TestRepositoryReviewIssueWriterAndCapabilityHelperCoverage(t *testing.T) {
	plain := repoaudit.IssueDraft{
		GenerationID: "generation", ResolvedInstructions: "instructions",
		InstructionsMode: repoaudit.IssueDraftInstructionsDefault,
		GeneratorModel:   "model", GeneratorAccount: "account",
	}
	if got := repositoryReviewIssueAttemptGenerationID(plain); got != "generation" {
		t.Fatalf("plain attempt ID=%q", got)
	}
	generationID, instructions, mode, model, account := repositoryReviewIssueAttemptProvenance(plain)
	if generationID != "generation" || instructions != "instructions" ||
		mode != repoaudit.IssueDraftInstructionsDefault || model != "model" || account != "account" {
		t.Fatalf("plain provenance=%q/%q/%q/%q/%q", generationID, instructions, mode, model, account)
	}
	attempt := plain
	attempt.AttemptGenerationID = "attempt"
	attempt.AttemptResolvedInstructions = "attempt instructions"
	attempt.AttemptInstructionsMode = repoaudit.IssueDraftInstructionsCustom
	attempt.AttemptGeneratorModel = "attempt model"
	attempt.AttemptGeneratorAccount = "attempt account"
	if got := repositoryReviewIssueAttemptGenerationID(attempt); got != "attempt" {
		t.Fatalf("attempt ID=%q", got)
	}
	generationID, instructions, mode, model, account = repositoryReviewIssueAttemptProvenance(attempt)
	if generationID != "attempt" || instructions != "attempt instructions" ||
		mode != repoaudit.IssueDraftInstructionsCustom || model != "attempt model" ||
		account != "attempt account" {
		t.Fatalf("attempt provenance=%q/%q/%q/%q/%q", generationID, instructions, mode, model, account)
	}

	state := repoaudit.RepositoryState{Repository: "owner/repo"}
	noncanonical := repositoryReviewIssueCapabilities(state, repoaudit.IssueDraft{})
	if noncanonical.ReadOnlyReason == "" || noncanonical.CanEdit {
		t.Fatalf("noncanonical capabilities=%#v", noncanonical)
	}
	for _, test := range []struct {
		draft repoaudit.IssueDraft
		check func(repositoryReviewCapabilities) bool
	}{
		{
			draft: repoaudit.IssueDraft{Canonical: true, Origin: repoaudit.IssueDraftOriginAIGenerated, State: repoaudit.IssueDraftGenerating},
			check: func(c repositoryReviewCapabilities) bool { return c.CanRegenerate && !c.CanPublish },
		},
		{
			draft: repoaudit.IssueDraft{Canonical: true, Origin: repoaudit.IssueDraftOriginAIGenerated, State: repoaudit.IssueDraftFailed},
			check: func(c repositoryReviewCapabilities) bool { return c.CanDelete && c.CanRegenerate },
		},
		{
			draft: repoaudit.IssueDraft{Canonical: true, Origin: repoaudit.IssueDraftOriginLegacy, State: repoaudit.IssueDraftUnknown},
			check: func(c repositoryReviewCapabilities) bool { return c.CanPublish && !c.CanRegenerate },
		},
		{
			draft: repoaudit.IssueDraft{Canonical: true, Origin: repoaudit.IssueDraftOriginLinked, State: repoaudit.IssueDraftPosted},
			check: func(c repositoryReviewCapabilities) bool { return c.CanUnlinkIssue && !c.CanPublish },
		},
	} {
		capabilities := repositoryReviewIssueCapabilities(state, test.draft)
		if !test.check(capabilities) {
			t.Fatalf("capabilities for %#v = %#v", test.draft, capabilities)
		}
	}
	localCapabilities := repositoryReviewIssueCapabilities(
		repoaudit.RepositoryState{Repository: "/tmp/repo"},
		repoaudit.IssueDraft{
			Canonical: true,
			Origin:    repoaudit.IssueDraftOriginLegacy,
			State:     repoaudit.IssueDraftEditing,
		},
	)
	if localCapabilities.GitHub || localCapabilities.CanPublish || !localCapabilities.CanEdit {
		t.Fatalf("local capabilities=%#v", localCapabilities)
	}

	request := repositoryReviewIssueWriterAgentRequest(
		repoaudit.RepositoryReviewAutomation{IssueWriterModel: "writer"},
		repoaudit.Finding{Observations: []repoaudit.FindingObservation{{Title: "private"}}},
		[]repoaudit.FindingContext{{
			ID: "context", RawDigest: "private-digest", Model: "private-concrete-model",
			ModelAlias: "private-model-alias", Account: "private-account-ref",
		}},
		"instructions", "account",
	)
	if strings.Contains(request.Prompt, "private-digest") ||
		strings.Contains(request.Prompt, "private-concrete-model") ||
		strings.Contains(request.Prompt, "private-model-alias") ||
		strings.Contains(request.Prompt, "private-account-ref") ||
		strings.Contains(request.Prompt, `"observations"`) {
		t.Fatalf("writer prompt retained private projection: %s", request.Prompt)
	}

	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	if account, err := handler.repositoryReviewIssueWriterAccount(repoaudit.RepositoryReviewAutomation{
		EffectiveAccountRef: "explicit",
	}); err != nil || account != "explicit" {
		t.Fatalf("effective account=%q err=%v", account, err)
	}
	if account, err := handler.repositoryReviewIssueWriterAccount(repoaudit.RepositoryReviewAutomation{}); err != nil ||
		account != "api" {
		t.Fatalf("default account=%q err=%v", account, err)
	}
	malformedConfigPath := filepath.Join(t.TempDir(), "malformed.json")
	if err := os.WriteFile(malformedConfigPath, []byte(`{`), 0o600); err != nil {
		t.Fatal(err)
	}
	missingConfig := NewHandler(malformedConfigPath)
	if _, err := missingConfig.repositoryReviewIssueWriterAccount(repoaudit.RepositoryReviewAutomation{}); err == nil {
		t.Fatal("missing config issue writer account succeeded")
	}
	if _, err := defaultRunRepositoryReviewIssueWriter(
		t.Context(), nil, repoaudit.RepositoryReviewAutomation{}, repoaudit.Finding{}, nil, "", "",
	); err == nil {
		t.Fatal("nil handler issue writer succeeded")
	}
	if _, err := defaultRunRepositoryReviewIssueWriter(
		t.Context(), missingConfig, repoaudit.RepositoryReviewAutomation{}, repoaudit.Finding{}, nil, "", "",
	); err == nil {
		t.Fatal("missing config default writer succeeded")
	}
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ModelList[0].APIBase = "http://127.0.0.1:1/v1"
	cfg.ModelList[0].APIKeys = config.SimpleSecureStrings("test-key")
	if saveErr := config.SaveConfig(handler.configPath, cfg); saveErr != nil {
		t.Fatal(saveErr)
	}
	_, err = defaultRunRepositoryReviewIssueWriter(
		t.Context(), handler,
		repoaudit.RepositoryReviewAutomation{IssueWriterModel: "cheap"},
		repoaudit.Finding{Evidence: strings.Repeat("x", repositoryReviewIssuePromptBytes+1)},
		nil, repositoryReviewDefaultIssueInstructions, "api",
	)
	if err == nil || !strings.Contains(err.Error(), "safe bound") {
		t.Fatalf("oversized issue writer error=%v", err)
	}

	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := runRepositoryReviewIssueWriterWithSlot(
		canceled, repoaudit.NewStore(t.TempDir()), handler,
		repoaudit.RepositoryReviewAutomation{}, repoaudit.Finding{}, nil, "", "",
	); err == nil {
		t.Fatal("canceled issue writer slot acquisition succeeded")
	}
}

func TestRepositoryReviewGatewayProxyAndPublicationBoundaryCoverage(t *testing.T) {
	t.Run("link proxy validation", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		state := seedRepositoryReviewAPIState(t, workspace)
		automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
		base := "/api/repository-reviews/automations/" + automation.ID +
			"/findings/" + state.Findings[0].ID + "/issue-link/candidates"
		for name, request := range map[string]*http.Request{
			"cross-site": func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "http://launcher.local"+base, strings.NewReader(`{}`))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Sec-Fetch-Site", "cross-site")
				return r
			}(),
			"empty": httptest.NewRequest(http.MethodPost, base, nil),
			"invalid-json": func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, base, strings.NewReader(`{`))
				setRepositoryReviewMutationHeaders(r)
				return r
			}(),
			"query": func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, base+"?x=1", strings.NewReader(`{}`))
				setRepositoryReviewMutationHeaders(r)
				return r
			}(),
		} {
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("%s proxy=%d %s", name, response.Code, response.Body.String())
			}
		}
	})

	t.Run("single publication boundaries", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		state := seedRepositoryReviewAPIState(t, workspace)
		state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
		automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
		store := repoaudit.NewStore(workspace)
		_, draft, _, err := store.ReserveIssueGeneration(repoaudit.IssueGenerationRequest{
			Repository: state.Repository, FindingID: state.Findings[0].ID,
			GenerationID: "rrig_gateway_bounds", ResolvedInstructions: repositoryReviewDefaultIssueInstructions,
			InstructionsMode: repoaudit.IssueDraftInstructionsDefault,
			GeneratorModel:   "cheap", GeneratorAccount: "api",
		})
		if err != nil {
			t.Fatal(err)
		}
		_, draft, err = store.CompleteIssueGeneration(
			state.Repository, draft.ID, draft.AttemptGenerationID,
			"Preview", "Evidence", []string{"bug"}, "",
		)
		if err != nil {
			t.Fatal(err)
		}
		path := baseAutomationIssuePath(automation.ID, draft.ID) + "/publish"
		for name, body := range map[string]map[string]any{
			"unconfirmed": {"expected_version": draft.Version, "confirmed": false},
			"stale":       {"expected_version": draft.Version + 1, "confirmed": true},
		} {
			response := repositoryReviewAutomationMutation(t, mux, http.MethodPost, path, body)
			want := http.StatusBadRequest
			if name == "stale" {
				want = http.StatusConflict
			}
			if response.Code != want {
				t.Fatalf("%s publication=%d %s", name, response.Code, response.Body.String())
			}
		}
		missing := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost, baseAutomationIssuePath(automation.ID, "missing")+"/publish",
			map[string]any{"expected_version": 1, "confirmed": true},
		)
		if missing.Code != http.StatusNotFound {
			t.Fatalf("missing publication=%d %s", missing.Code, missing.Body.String())
		}
		installEventProxyStubs(t, func(_ *http.Request, _ time.Duration) (*http.Response, error) {
			response := eventUpstreamResponse(
				http.StatusServiceUnavailable,
				`{"code":"publication_failed","message":"safe"}`,
			)
			response.Header.Set("Retry-After", "1")
			return response, nil
		})
		failure := repositoryReviewAutomationMutation(t, mux, http.MethodPost, path, map[string]any{
			"expected_version": draft.Version, "confirmed": true,
		})
		if failure.Code != http.StatusServiceUnavailable || failure.Header().Get("Retry-After") != "1" {
			t.Fatalf(
				"gateway publication failure=%d headers=%v body=%s",
				failure.Code,
				failure.Header(),
				failure.Body.String(),
			)
		}
	})

	t.Run("batch selection and outcomes", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		state := seedRepositoryReviewAPIState(t, workspace)
		automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
		path := "/api/repository-reviews/automations/" + automation.ID + "/issues/publish"
		for name, body := range map[string]map[string]any{
			"unconfirmed":  {"confirmed": false, "issues": []map[string]any{{"id": "draft", "expected_version": 1}}},
			"empty":        {"confirmed": true, "issues": []map[string]any{}},
			"invalid item": {"confirmed": true, "issues": []map[string]any{{"id": "", "expected_version": 0}}},
			"duplicate": {"confirmed": true, "issues": []map[string]any{
				{"id": "draft", "expected_version": 1}, {"id": "draft", "expected_version": 1},
			}},
		} {
			response := repositoryReviewAutomationMutation(t, mux, http.MethodPost, path, body)
			if response.Code != http.StatusBadRequest && response.Code != http.StatusNotFound {
				t.Fatalf("%s batch=%d %s", name, response.Code, response.Body.String())
			}
		}
		missingAutomation := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost,
			"/api/repository-reviews/automations/rra_missing/issues/publish",
			map[string]any{"confirmed": true, "issues": []map[string]any{{"id": "draft", "expected_version": 1}}},
		)
		if missingAutomation.Code != http.StatusNotFound {
			t.Fatalf("missing automation batch=%d %s", missingAutomation.Code, missingAutomation.Body.String())
		}
		emptyAutomation := testRepositoryReviewAutomation()
		emptyAutomation.ID = "rra_publish_empty"
		emptyAutomation.Repository = "owner/no-ledger"
		store := repoaudit.NewStore(workspace)
		if _, err := store.CreateAutomation(t.Context(), emptyAutomation); err != nil {
			t.Fatal(err)
		}
		missingLedger := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost,
			"/api/repository-reviews/automations/"+emptyAutomation.ID+"/issues/publish",
			map[string]any{"confirmed": true, "issues": []map[string]any{{"id": "draft", "expected_version": 1}}},
		)
		if missingLedger.Code != http.StatusNotFound {
			t.Fatalf("missing ledger batch=%d %s", missingLedger.Code, missingLedger.Body.String())
		}

		// A valid ledger with an unknown draft exercises the per-selection
		// not-found result without aborting the whole confirmed batch.
		unknown := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost, path,
			map[string]any{"confirmed": true, "issues": []map[string]any{{"id": "missing", "expected_version": 1}}},
		)
		if unknown.Code != http.StatusOK || !strings.Contains(unknown.Body.String(), `"code":"not_found"`) {
			t.Fatalf("unknown draft batch=%d %s", unknown.Code, unknown.Body.String())
		}
	})

	// Direct publication result classification covers successful reconciliation,
	// ordinary success, and a safe gateway failure without depending on store state.
	handler := NewHandler("")
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	ledger := repositoryReviewAutomationLedger{State: repoaudit.RepositoryState{ID: "rrp_test"}}
	var status int
	var responseBody string
	installEventProxyStubs(t, func(_ *http.Request, _ time.Duration) (*http.Response, error) {
		return eventUpstreamResponse(status, responseBody), nil
	})
	status, responseBody = http.StatusAccepted, `{"outcome":"unknown","draft":{"id":"draft","state":"unknown"}}`
	unknown := handler.publishRepositoryReviewAutomationDraft(request, ledger, "draft", 1)
	status, responseBody = http.StatusOK, `{"draft":{"id":"draft","state":"posted","external_url":"https://github.com/o/r/issues/1"}}`
	posted := handler.publishRepositoryReviewAutomationDraft(request, ledger, "draft", 1)
	status, responseBody = http.StatusServiceUnavailable, `{"code":"finding_status_unresolved","message":"safe","publish_blockers":[{"code":"finding_status_unresolved","count":2,"message":"safe"}]}`
	failed := handler.publishRepositoryReviewAutomationDraft(request, ledger, "draft", 1)
	failedBlockers, blockersOK := failed["publish_blockers"].([]repoaudit.IssuePublicationBlocker)
	if unknown["outcome"] != "unknown" || posted["outcome"] != "posted" ||
		posted["success"] != true || failed["outcome"] != "failed" ||
		failed["code"] != "finding_status_unresolved" || !blockersOK ||
		len(failedBlockers) != 1 || failedBlockers[0].Count != 2 {
		t.Fatalf("publication outcomes unknown=%#v posted=%#v failed=%#v", unknown, posted, failed)
	}
}

func TestRepositoryReviewProfileCollectionFieldCoverage(t *testing.T) {
	profile := repoaudit.RepositoryReviewProfile{
		ID: "rrpf_fields", Name: "Fields", AccountRef: "account",
		ReviewerModel: "reviewer", IssueWriterModel: "writer", Force: true,
		AutoContinue: true, MaxFilesPerRun: 12, MaxParallelChildren: 4,
		Version: 3, UpdatedAt: time.Now().UTC(),
	}
	for _, field := range []string{
		"id", "name", "account", "reviewer", "issue_writer", "force",
		"auto_continue", "files", "parallel", "version", "updated",
	} {
		if _, ok := repositoryReviewProfileCollectionField(profile, collectionquery.Field(field)); !ok {
			t.Fatalf("profile collection field %q not resolved", field)
		}
	}
	profile.IssueWriterModel = ""
	if _, ok := repositoryReviewProfileCollectionField(profile, "issue_writer"); !ok {
		t.Fatal("inherited issue writer not resolved")
	}
	if _, ok := repositoryReviewProfileCollectionField(profile, "unknown"); ok {
		t.Fatal("unknown profile collection field resolved")
	}

	_, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	invalid := httptest.NewRecorder()
	mux.ServeHTTP(invalid, httptest.NewRequest(
		http.MethodGet, "/api/repository-reviews/profiles?unknown=1", nil,
	))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid profile collection request=%d %s", invalid.Code, invalid.Body.String())
	}
	badCursor := httptest.NewRecorder()
	mux.ServeHTTP(badCursor, httptest.NewRequest(
		http.MethodGet, "/api/repository-reviews/profiles?query=ALL&cursor=invalid&limit=1", nil,
	))
	if badCursor.Code != http.StatusBadRequest {
		t.Fatalf("bad profile cursor=%d %s", badCursor.Code, badCursor.Body.String())
	}
}

func TestRepositoryReviewDefaultIssueWriterProviderBoundaries(t *testing.T) {
	responseStatus := http.StatusOK
	responseContent := `{"title":"Generated issue","body":"Grounded evidence.","labels":["bug"]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(responseStatus)
		if responseStatus != http.StatusOK {
			_, _ = w.Write([]byte(`{"error":{"message":"provider unavailable"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-review", "object": "chat.completion",
			"choices": []map[string]any{{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": responseContent},
				"finish_reason": "stop",
			}},
			"usage": map[string]any{
				"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15,
			},
		})
	}))
	t.Cleanup(server.Close)
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ModelList[0].APIBase = server.URL + "/v1"
	cfg.ModelList[0].APIKeys = config.SimpleSecureStrings("test-key")
	if saveErr := config.SaveConfig(handler.configPath, cfg); saveErr != nil {
		t.Fatal(saveErr)
	}
	automation := repoaudit.RepositoryReviewAutomation{
		IssueWriterModel: "cheap", EffectiveAccountRef: "api",
	}
	finding := repoaudit.Finding{
		ID: "finding", Title: "Lost update", Evidence: "A stale write overwrites data.",
		Impact: "Data is lost.", File: repoaudit.FileRef{Path: "service.go", BlobSHA: strings.Repeat("a", 40)},
		CommitSHA: strings.Repeat("b", 40), Validation: repoaudit.Validation{Summary: "Confirmed."},
	}
	generated, err := defaultRunRepositoryReviewIssueWriter(
		t.Context(), handler, automation, finding, nil,
		repositoryReviewDefaultIssueInstructions, "api",
	)
	if err != nil || generated.Title != "Generated issue" || generated.Body != "Grounded evidence." {
		t.Fatalf("generated=%#v err=%v", generated, err)
	}
	responseContent = `not structured JSON`
	if _, writerErr := defaultRunRepositoryReviewIssueWriter(
		t.Context(), handler, automation, finding, nil,
		repositoryReviewDefaultIssueInstructions, "api",
	); writerErr == nil || !strings.Contains(writerErr.Error(), "structured output") {
		t.Fatalf("invalid structured response error=%v", writerErr)
	}
	responseStatus = http.StatusServiceUnavailable
	if _, writerErr := defaultRunRepositoryReviewIssueWriter(
		t.Context(), handler, automation, finding, nil,
		repositoryReviewDefaultIssueInstructions, "api",
	); writerErr == nil {
		t.Fatal("provider failure was accepted")
	}
	responseStatus = http.StatusOK
	cfg, err = config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Agents.List = []config.AgentConfig{{ID: "other", Default: true}}
	if err := config.SaveConfig(handler.configPath, cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := defaultRunRepositoryReviewIssueWriter(
		t.Context(), handler, automation, finding, nil,
		repositoryReviewDefaultIssueInstructions, "api",
	); err == nil || !strings.Contains(err.Error(), "main") {
		t.Fatalf("missing main agent resolution error=%v", err)
	}
}

func TestRepositoryReviewAdditionalHandlerBranchCoverage(t *testing.T) {
	t.Run("report and mutation errors", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		state := seedRepositoryReviewGenerationFindings(t, workspace, 3)
		automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
		base := "/api/repository-reviews/automations/" + automation.ID
		missingReport := httptest.NewRecorder()
		mux.ServeHTTP(missingReport, httptest.NewRequest(
			http.MethodGet, "/api/repository-reviews/automations/rra_missing/report", nil,
		))
		if missingReport.Code != http.StatusNotFound {
			t.Fatalf("missing report=%d %s", missingReport.Code, missingReport.Body.String())
		}
		paged := httptest.NewRecorder()
		mux.ServeHTTP(paged, httptest.NewRequest(
			http.MethodGet, base+"/report?scope=all&limit=1", nil,
		))
		if paged.Code != http.StatusOK || !strings.Contains(paged.Body.String(), `"next_offset":1`) {
			t.Fatalf("paged report=%d %s", paged.Code, paged.Body.String())
		}
		findingPath := base + "/findings/" + state.Findings[0].ID
		malformed := httptest.NewRequest(http.MethodPatch, findingPath, strings.NewReader(`{`))
		setRepositoryReviewMutationHeaders(malformed)
		malformedResponse := httptest.NewRecorder()
		mux.ServeHTTP(malformedResponse, malformed)
		if malformedResponse.Code != http.StatusBadRequest {
			t.Fatalf("malformed finding=%d %s", malformedResponse.Code, malformedResponse.Body.String())
		}
		missingFinding := repositoryReviewAutomationMutation(
			t, mux, http.MethodPatch, base+"/findings/missing",
			map[string]any{"status": "dismissed", "expected_version": 1},
		)
		if missingFinding.Code != http.StatusNotFound {
			t.Fatalf("missing finding mutation=%d %s", missingFinding.Code, missingFinding.Body.String())
		}
		missingIssues := httptest.NewRecorder()
		mux.ServeHTTP(missingIssues, httptest.NewRequest(
			http.MethodGet, "/api/repository-reviews/automations/rra_missing/issues", nil,
		))
		if missingIssues.Code != http.StatusNotFound {
			t.Fatalf("missing issue list=%d %s", missingIssues.Code, missingIssues.Body.String())
		}
	})

	t.Run("issue update and delete errors", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		state := seedRepositoryReviewAPIState(t, workspace)
		state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
		automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
		store := repoaudit.NewStore(workspace)
		_, draft, _, err := store.ReserveIssueGeneration(repoaudit.IssueGenerationRequest{
			Repository: state.Repository, FindingID: state.Findings[0].ID,
			GenerationID: "rrig_issue_errors", ResolvedInstructions: repositoryReviewDefaultIssueInstructions,
			InstructionsMode: repoaudit.IssueDraftInstructionsDefault,
			GeneratorModel:   "cheap", GeneratorAccount: "api",
		})
		if err != nil {
			t.Fatal(err)
		}
		_, draft, err = store.CompleteIssueGeneration(
			state.Repository, draft.ID, draft.AttemptGenerationID,
			"Preview", "Evidence", []string{"bug"}, "",
		)
		if err != nil {
			t.Fatal(err)
		}
		path := baseAutomationIssuePath(automation.ID, draft.ID)
		cross := httptest.NewRequest(http.MethodPatch, "http://launcher.local"+path, strings.NewReader(`{}`))
		cross.Header.Set("Content-Type", "application/json")
		cross.Header.Set("Sec-Fetch-Site", "cross-site")
		crossResponse := httptest.NewRecorder()
		mux.ServeHTTP(crossResponse, cross)
		if crossResponse.Code != http.StatusBadRequest {
			t.Fatalf("cross issue update=%d %s", crossResponse.Code, crossResponse.Body.String())
		}
		badJSON := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(`{`))
		setRepositoryReviewMutationHeaders(badJSON)
		badJSONResponse := httptest.NewRecorder()
		mux.ServeHTTP(badJSONResponse, badJSON)
		if badJSONResponse.Code != http.StatusBadRequest {
			t.Fatalf("bad issue update=%d %s", badJSONResponse.Code, badJSONResponse.Body.String())
		}
		missing := repositoryReviewAutomationMutation(
			t, mux, http.MethodPatch, baseAutomationIssuePath(automation.ID, "missing"),
			map[string]any{"title": "x", "body": "y", "labels": []string{}, "expected_version": 1},
		)
		if missing.Code != http.StatusNotFound {
			t.Fatalf("missing issue update=%d %s", missing.Code, missing.Body.String())
		}
		invalid := repositoryReviewAutomationMutation(t, mux, http.MethodPatch, path, map[string]any{
			"title": "", "body": "", "labels": []string{}, "expected_version": draft.Version,
		})
		if invalid.Code != http.StatusBadRequest {
			t.Fatalf("invalid issue update=%d %s", invalid.Code, invalid.Body.String())
		}
		deleteCross := httptest.NewRequest(http.MethodDelete, "http://launcher.local"+path, strings.NewReader(`{}`))
		deleteCross.Header.Set("Content-Type", "application/json")
		deleteCross.Header.Set("Sec-Fetch-Site", "cross-site")
		deleteCrossResponse := httptest.NewRecorder()
		mux.ServeHTTP(deleteCrossResponse, deleteCross)
		if deleteCrossResponse.Code != http.StatusBadRequest {
			t.Fatalf("cross issue delete=%d %s", deleteCrossResponse.Code, deleteCrossResponse.Body.String())
		}
		missingDelete := repositoryReviewAutomationMutation(
			t, mux, http.MethodDelete, baseAutomationIssuePath(automation.ID, "missing"),
			map[string]any{"expected_version": 1, "confirmed": true},
		)
		if missingDelete.Code != http.StatusNotFound {
			t.Fatalf("missing issue delete=%d %s", missingDelete.Code, missingDelete.Body.String())
		}
		// Publishing makes the draft undeletable while retaining its exact version.
		_, publishing, _, err := store.ClaimIssueDraftPublication(state.Repository, draft.ID, draft.Version)
		if err != nil {
			t.Fatal(err)
		}
		undeletable := repositoryReviewAutomationMutation(t, mux, http.MethodDelete, path, map[string]any{
			"expected_version": publishing.Version, "confirmed": true,
		})
		if undeletable.Code != http.StatusConflict {
			t.Fatalf("undeletable issue=%d %s", undeletable.Code, undeletable.Body.String())
		}
	})

	t.Run("generation and regeneration errors", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		state := seedRepositoryReviewGenerationFindings(t, workspace, 3)
		state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
		automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
		generationPath := "/api/repository-reviews/automations/" + automation.ID + "/issues/generations"
		cross := httptest.NewRequest(http.MethodPost, "http://launcher.local"+generationPath, strings.NewReader(`{}`))
		cross.Header.Set("Content-Type", "application/json")
		cross.Header.Set("Sec-Fetch-Site", "cross-site")
		crossResponse := httptest.NewRecorder()
		mux.ServeHTTP(crossResponse, cross)
		if crossResponse.Code != http.StatusBadRequest {
			t.Fatalf("cross generation=%d %s", crossResponse.Code, crossResponse.Body.String())
		}
		badJSON := httptest.NewRequest(http.MethodPost, generationPath, strings.NewReader(`{`))
		setRepositoryReviewMutationHeaders(badJSON)
		badJSONResponse := httptest.NewRecorder()
		mux.ServeHTTP(badJSONResponse, badJSON)
		if badJSONResponse.Code != http.StatusBadRequest {
			t.Fatalf("bad generation=%d %s", badJSONResponse.Code, badJSONResponse.Body.String())
		}
		store := repoaudit.NewStore(workspace)
		_, legacy, err := store.PrepareIssue(repoaudit.IssueDraftRequest{
			Repository: state.Repository, FindingIDs: []string{state.Findings[0].ID},
			ExpectedVersion: state.Version,
		})
		if err != nil {
			t.Fatal(err)
		}
		legacyRegen := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost, baseAutomationIssuePath(automation.ID, legacy.ID)+"/regenerate",
			map[string]any{"expected_version": legacy.Version},
		)
		if legacyRegen.Code != http.StatusConflict {
			t.Fatalf("legacy regen=%d %s", legacyRegen.Code, legacyRegen.Body.String())
		}
		regenPath := baseAutomationIssuePath(automation.ID, legacy.ID) + "/regenerate"
		crossRegen := httptest.NewRequest(
			http.MethodPost, "http://launcher.local"+regenPath, strings.NewReader(`{}`),
		)
		crossRegen.Header.Set("Content-Type", "application/json")
		crossRegen.Header.Set("Sec-Fetch-Site", "cross-site")
		crossRegenResponse := httptest.NewRecorder()
		mux.ServeHTTP(crossRegenResponse, crossRegen)
		if crossRegenResponse.Code != http.StatusBadRequest {
			t.Fatalf("cross regen=%d %s", crossRegenResponse.Code, crossRegenResponse.Body.String())
		}
		badRegenJSON := httptest.NewRequest(http.MethodPost, regenPath, strings.NewReader(`{`))
		setRepositoryReviewMutationHeaders(badRegenJSON)
		badRegenJSONResponse := httptest.NewRecorder()
		mux.ServeHTTP(badRegenJSONResponse, badRegenJSON)
		if badRegenJSONResponse.Code != http.StatusBadRequest {
			t.Fatalf("bad regen JSON=%d %s", badRegenJSONResponse.Code, badRegenJSONResponse.Body.String())
		}
		previous := runRepositoryReviewIssueWriter
		t.Cleanup(func() { runRepositoryReviewIssueWriter = previous })
		runRepositoryReviewIssueWriter = func(
			_ context.Context, _ *Handler, _ repoaudit.RepositoryReviewAutomation,
			_ repoaudit.Finding, _ []repoaudit.FindingContext, _, _ string,
		) (repositoryReviewIssueWriterResult, error) {
			return repositoryReviewIssueWriterResult{Title: "", Body: ""}, nil
		}
		invalidOutput := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost, generationPath,
			map[string]any{
				"generation_id": "rrig_invalid_output",
				"finding_ids":   []string{state.Findings[1].ID},
			},
		)
		if invalidOutput.Code != http.StatusOK ||
			!strings.Contains(invalidOutput.Body.String(), `"code":"generation_failed"`) {
			t.Fatalf("invalid writer output=%d %s", invalidOutput.Code, invalidOutput.Body.String())
		}
		persisted, found, err := store.Get(state.Repository)
		if err != nil || !found {
			t.Fatalf("load invalid output state found=%v err=%v", found, err)
		}
		invalidDraft, found := repositoryReviewIssueByID(
			persisted, persisted.Findings[1].IssueDraftID,
		)
		if !found {
			t.Fatal("invalid output draft missing")
		}
		invalidRegen := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost,
			baseAutomationIssuePath(automation.ID, invalidDraft.ID)+"/regenerate",
			map[string]any{"expected_version": invalidDraft.Version},
		)
		if invalidRegen.Code != http.StatusBadRequest {
			t.Fatalf("invalid regeneration completion=%d %s", invalidRegen.Code, invalidRegen.Body.String())
		}

		runRepositoryReviewIssueWriter = func(
			_ context.Context, _ *Handler, _ repoaudit.RepositoryReviewAutomation,
			finding repoaudit.Finding, _ []repoaudit.FindingContext, _, _ string,
		) (repositoryReviewIssueWriterResult, error) {
			return repositoryReviewIssueWriterResult{Title: finding.Title, Body: "Evidence"}, nil
		}
		success := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost, generationPath,
			map[string]any{
				"generation_id": "rrig_regen_error",
				"finding_ids":   []string{state.Findings[2].ID},
			},
		)
		if success.Code != http.StatusOK {
			t.Fatalf("regen seed=%d %s", success.Code, success.Body.String())
		}
		persisted, _, _ = store.Get(state.Repository)
		seedDraft, _ := repositoryReviewIssueByID(persisted, persisted.Findings[2].IssueDraftID)
		runRepositoryReviewIssueWriter = func(
			_ context.Context, _ *Handler, _ repoaudit.RepositoryReviewAutomation,
			_ repoaudit.Finding, _ []repoaudit.FindingContext, _, _ string,
		) (repositoryReviewIssueWriterResult, error) {
			return repositoryReviewIssueWriterResult{}, errors.New("private provider error")
		}
		failedRegen := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost,
			baseAutomationIssuePath(automation.ID, seedDraft.ID)+"/regenerate",
			map[string]any{"expected_version": seedDraft.Version},
		)
		if failedRegen.Code != http.StatusOK ||
			!strings.Contains(failedRegen.Body.String(), "Issue preview generation failed") ||
			strings.Contains(failedRegen.Body.String(), "private provider error") {
			t.Fatalf("failed regen=%d %s", failedRegen.Code, failedRegen.Body.String())
		}
	})
}

func TestRepositoryReviewAutomationLedgerResolutionBoundaries(t *testing.T) {
	t.Run("configuration and automation errors", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(configPath, []byte(`{`), 0o600); err != nil {
			t.Fatal(err)
		}
		handler := NewHandler(configPath)
		if _, err := handler.repositoryReviewAutomationLedger(t.Context(), "rra_missing"); err == nil {
			t.Fatal("malformed configuration ledger lookup succeeded")
		}

		handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
		root := filepath.Join(workspace, "repository_reviews")
		if err := os.MkdirAll(root, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			filepath.Join(root, "automation_rra_corrupt.json"), []byte(`{`), 0o600,
		); err != nil {
			t.Fatal(err)
		}
		if _, err := handler.repositoryReviewAutomationLedger(t.Context(), "rra_corrupt"); err == nil {
			t.Fatal("corrupt automation ledger lookup succeeded")
		}
	})

	t.Run("direct state corruption", func(t *testing.T) {
		handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
		state := seedRepositoryReviewAPIState(t, workspace)
		automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
		statePath := filepath.Join(
			workspace, "repository_reviews", "repo_"+strings.TrimPrefix(state.ID, "rrp_")+".json",
		)
		if err := os.WriteFile(statePath, []byte(`{`), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := handler.repositoryReviewAutomationLedger(t.Context(), automation.ID); err == nil {
			t.Fatal("corrupt direct ledger lookup succeeded")
		}
	})

	t.Run("run history scan", func(t *testing.T) {
		handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
		state := seedRepositoryReviewAPIState(t, workspace)
		store := repoaudit.NewStore(workspace)
		unmatchedFile := repoaudit.FileRef{
			Path: "unmatched.go", BlobSHA: strings.Repeat("e", 40), SizeBytes: 10,
			Category: "code", Mode: "100644",
		}
		unmatchedPlan, err := store.Plan(
			t.Context(), "owner/unmatched", strings.Repeat("f", 40), "inventory-unmatched",
			[]repoaudit.FileRef{unmatchedFile}, false,
		)
		if err != nil {
			t.Fatal(err)
		}
		if _, recordErr := store.Record(t.Context(), repoaudit.RecordRequest{
			Plan: unmatchedPlan, RunID: "unmatched-run",
			Observations: []repoaudit.Observation{{Model: "review", ScopeFiles: []repoaudit.FileRef{unmatchedFile}}},
		}); recordErr != nil {
			t.Fatal(recordErr)
		}
		automation := testRepositoryReviewAutomation()
		automation.ID = "rra_scanned_ledger"
		automation.Repository = filepath.Join(t.TempDir(), "detached-clone")
		automation.RunIDs = []string{state.Runs[0].ID}
		automation, err = store.CreateAutomation(t.Context(), automation)
		if err != nil {
			t.Fatal(err)
		}
		ledger, err := handler.repositoryReviewAutomationLedger(t.Context(), automation.ID)
		if err != nil || !ledger.Found || ledger.State.Repository != state.Repository {
			t.Fatalf("scanned ledger=%#v err=%v", ledger, err)
		}
		if _, _, err := handler.repositoryReviewAutomationFinding(
			t.Context(), automation.ID, "missing",
		); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("missing scanned finding error=%v", err)
		}
		if _, _, err := handler.repositoryReviewAutomationIssue(
			t.Context(), automation.ID, "missing",
		); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("missing scanned issue error=%v", err)
		}
	})

	t.Run("ambiguous run history", func(t *testing.T) {
		handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
		first := seedRepositoryReviewAPIState(t, workspace)
		store := repoaudit.NewStore(workspace)
		file := repoaudit.FileRef{
			Path: "other.go", BlobSHA: strings.Repeat("c", 40), SizeBytes: 10,
			Category: "code", Mode: "100644",
		}
		plan, err := store.Plan(
			t.Context(), "owner/other", strings.Repeat("d", 40), "inventory-other",
			[]repoaudit.FileRef{file}, false,
		)
		if err != nil {
			t.Fatal(err)
		}
		if _, recordErr := store.Record(t.Context(), repoaudit.RecordRequest{
			Plan: plan, RunID: first.Runs[0].ID,
			Observations: []repoaudit.Observation{{Model: "review", ScopeFiles: []repoaudit.FileRef{file}}},
		}); recordErr != nil {
			t.Fatal(recordErr)
		}
		automation := testRepositoryReviewAutomation()
		automation.ID = "rra_ambiguous_ledger"
		automation.Repository = filepath.Join(t.TempDir(), "detached-clone")
		automation.RunIDs = []string{first.Runs[0].ID}
		automation, err = store.CreateAutomation(t.Context(), automation)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := handler.repositoryReviewAutomationLedger(t.Context(), automation.ID); err == nil ||
			!strings.Contains(err.Error(), "ambiguous") {
			t.Fatalf("ambiguous ledger error=%v", err)
		}
	})
}

func TestRepositoryReviewGenerationPersistenceAndAccountFailureBoundaries(t *testing.T) {
	t.Run("unavailable effective account", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		cfg, err := config.LoadConfig(handler.configPath)
		if err != nil {
			t.Fatal(err)
		}
		cfg.Agents.Defaults.AccountRef = ""
		if saveErr := config.SaveConfig(handler.configPath, cfg); saveErr != nil {
			t.Fatal(saveErr)
		}
		state := seedRepositoryReviewAPIState(t, workspace)
		state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
		automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
		store := repoaudit.NewStore(workspace)
		_, draft, _, err := store.ReserveIssueGeneration(repoaudit.IssueGenerationRequest{
			Repository: state.Repository, FindingID: state.Findings[0].ID,
			GenerationID: "rrig_no_account_regen", ResolvedInstructions: repositoryReviewDefaultIssueInstructions,
			InstructionsMode: repoaudit.IssueDraftInstructionsDefault,
			GeneratorModel:   "cheap", GeneratorAccount: "api",
		})
		if err != nil {
			t.Fatal(err)
		}
		_, draft, err = store.CompleteIssueGeneration(
			state.Repository, draft.ID, draft.AttemptGenerationID,
			"Preview", "Evidence", []string{"bug"}, "",
		)
		if err != nil {
			t.Fatal(err)
		}
		regen := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost,
			baseAutomationIssuePath(automation.ID, draft.ID)+"/regenerate",
			map[string]any{"expected_version": draft.Version},
		)
		if regen.Code != http.StatusInternalServerError {
			t.Fatalf("missing account regeneration=%d %s", regen.Code, regen.Body.String())
		}
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost,
			"/api/repository-reviews/automations/"+automation.ID+"/issues/generations",
			map[string]any{
				"generation_id": "rrig_no_account", "finding_ids": []string{"missing"},
			},
		)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("missing account generation=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("late persistence failure is safe", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		state := seedRepositoryReviewAPIState(t, workspace)
		state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
		automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
		previous := runRepositoryReviewIssueWriter
		t.Cleanup(func() { runRepositoryReviewIssueWriter = previous })
		runRepositoryReviewIssueWriter = func(
			_ context.Context, _ *Handler, _ repoaudit.RepositoryReviewAutomation,
			_ repoaudit.Finding, _ []repoaudit.FindingContext, _, _ string,
		) (repositoryReviewIssueWriterResult, error) {
			root := filepath.Join(workspace, "repository_reviews")
			if err := os.RemoveAll(root); err != nil {
				return repositoryReviewIssueWriterResult{}, err
			}
			if err := os.WriteFile(root, []byte("not a directory"), 0o600); err != nil {
				return repositoryReviewIssueWriterResult{}, err
			}
			return repositoryReviewIssueWriterResult{}, errors.New("private provider failure")
		}
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost,
			"/api/repository-reviews/automations/"+automation.ID+"/issues/generations",
			map[string]any{
				"generation_id": "rrig_late_failure", "finding_ids": []string{state.Findings[0].ID},
			},
		)
		if response.Code != http.StatusInternalServerError ||
			strings.Contains(response.Body.String(), "not a directory") {
			t.Fatalf("late persistence generation=%d %s", response.Code, response.Body.String())
		}
	})

	state := repoaudit.RepositoryState{
		Repository: "owner/repo",
		Findings: []repoaudit.Finding{{
			ID: "dismissed", Status: repoaudit.FindingDismissed, Version: 1,
		}},
	}
	ledger := repositoryReviewAutomationLedger{
		Store: repoaudit.NewStore(t.TempDir()), State: state, Found: true,
		Automation: repoaudit.RepositoryReviewAutomation{IssueWriterModel: "cheap"},
	}
	_, result := (&Handler{}).generateRepositoryReviewIssue(
		t.Context(), ledger, "dismissed", "rrig", repoaudit.IssueDraftInstructionsDefault,
		repositoryReviewDefaultIssueInstructions, "api",
	)
	if result["code"] != "generation_conflict" {
		t.Fatalf("generation conflict result=%#v", result)
	}
}

func TestRepositoryReviewRemainingGenerationAndLedgerBranches(t *testing.T) {
	t.Run("deterministic canceled slot", func(t *testing.T) {
		workspace := t.TempDir()
		store := repoaudit.NewStore(workspace)
		releases := make([]func(), 0, repositoryReviewIssueWriterConcurrency)
		for range repositoryReviewIssueWriterConcurrency {
			release, err := store.AcquireIssueGenerationSlot(t.Context(), repositoryReviewIssueWriterConcurrency)
			if err != nil {
				t.Fatal(err)
			}
			releases = append(releases, release)
		}
		defer func() {
			for _, release := range releases {
				release()
			}
		}()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if _, err := runRepositoryReviewIssueWriterWithSlot(
			ctx, store, &Handler{}, repoaudit.RepositoryReviewAutomation{},
			repoaudit.Finding{}, nil, "", "",
		); !errors.Is(err, context.Canceled) {
			t.Fatalf("full canceled slot error=%v", err)
		}
	})

	t.Run("external regeneration lock", func(t *testing.T) {
		workspace := t.TempDir()
		state := seedRepositoryReviewAPIState(t, workspace)
		state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
		store := repoaudit.NewStore(workspace)
		_, draft, _, err := store.ReserveIssueGeneration(repoaudit.IssueGenerationRequest{
			Repository: state.Repository, FindingID: state.Findings[0].ID,
			GenerationID: "rrig_external_regen", ResolvedInstructions: repositoryReviewDefaultIssueInstructions,
			InstructionsMode: repoaudit.IssueDraftInstructionsDefault,
			GeneratorModel:   "cheap", GeneratorAccount: "api",
		})
		if err != nil {
			t.Fatal(err)
		}
		_, draft, err = store.CompleteIssueGeneration(
			state.Repository, draft.ID, draft.AttemptGenerationID,
			"Preview", "Evidence", []string{"bug"}, "",
		)
		if err != nil {
			t.Fatal(err)
		}
		request := repoaudit.IssueGenerationRequest{
			Repository: state.Repository, FindingID: state.Findings[0].ID,
			GenerationID: "rrig_external_regen_2", ResolvedInstructions: repositoryReviewDefaultIssueInstructions,
			InstructionsMode: repoaudit.IssueDraftInstructionsDefault,
			GeneratorModel:   "cheap", GeneratorAccount: "api", ExpectedDraftVersion: draft.Version,
		}
		_, generating, reserved, err := store.BeginIssueRegeneration(state.Repository, draft.ID, request)
		if err != nil || !reserved {
			t.Fatalf("begin external regen=%v err=%v", reserved, err)
		}
		release, acquired, err := store.TryLockIssueGenerationAttempt(
			state.Repository, draft.ID, generating.AttemptGenerationID,
		)
		if err != nil || !acquired {
			t.Fatalf("external regen lock=%v err=%v", acquired, err)
		}
		defer release()
		_, replay, claimed, err := claimRepositoryReviewIssueRegeneration(
			store, state.Repository, draft.ID, request,
		)
		if err != nil || claimed || replay.ID != draft.ID {
			t.Fatalf("external regen replay=%v draft=%#v err=%v", claimed, replay, err)
		}
	})

	t.Run("missing final ledger", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		state := seedRepositoryReviewAPIState(t, workspace)
		state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
		automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
		previous := runRepositoryReviewIssueWriter
		t.Cleanup(func() { runRepositoryReviewIssueWriter = previous })
		runRepositoryReviewIssueWriter = func(
			_ context.Context, _ *Handler, _ repoaudit.RepositoryReviewAutomation,
			_ repoaudit.Finding, _ []repoaudit.FindingContext, _, _ string,
		) (repositoryReviewIssueWriterResult, error) {
			statePath := filepath.Join(
				workspace, "repository_reviews", "repo_"+strings.TrimPrefix(state.ID, "rrp_")+".json",
			)
			_ = os.Remove(statePath)
			_ = os.Remove(strings.TrimSuffix(statePath, ".json") + ".summary.json")
			return repositoryReviewIssueWriterResult{Title: "Preview", Body: "Evidence"}, nil
		}
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost,
			"/api/repository-reviews/automations/"+automation.ID+"/issues/generations",
			map[string]any{
				"generation_id": "rrig_missing_final", "finding_ids": []string{state.Findings[0].ID},
			},
		)
		if response.Code != http.StatusNotFound {
			t.Fatalf("missing final ledger=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("ledger list and empty owned lookups", func(t *testing.T) {
		handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
		store := repoaudit.NewStore(workspace)
		empty := testRepositoryReviewAutomation()
		empty.ID = "rra_empty_owned_lookup"
		empty.Repository = filepath.Join(t.TempDir(), "missing")
		empty, err := store.CreateAutomation(t.Context(), empty)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, lookupErr := handler.repositoryReviewAutomationFinding(t.Context(), empty.ID, "finding"); !errors.Is(
			lookupErr,
			os.ErrNotExist,
		) {
			t.Fatalf("empty owned finding error=%v", lookupErr)
		}
		if _, _, lookupErr := handler.repositoryReviewAutomationIssue(t.Context(), empty.ID, "draft"); !errors.Is(
			lookupErr,
			os.ErrNotExist,
		) {
			t.Fatalf("empty owned issue error=%v", lookupErr)
		}
		if _, _, lookupErr := handler.repositoryReviewAutomationFinding(
			t.Context(), "rra_missing", "finding",
		); !errors.Is(
			lookupErr,
			os.ErrNotExist,
		) {
			t.Fatalf("missing automation finding error=%v", lookupErr)
		}
		if _, _, lookupErr := handler.repositoryReviewAutomationIssue(t.Context(), "rra_missing", "draft"); !errors.Is(
			lookupErr,
			os.ErrNotExist,
		) {
			t.Fatalf("missing automation issue error=%v", lookupErr)
		}

		withRuns := testRepositoryReviewAutomation()
		withRuns.ID = "rra_bad_list"
		withRuns.Repository = filepath.Join(t.TempDir(), "detached")
		withRuns.RunIDs = []string{"run"}
		withRuns, err = store.CreateAutomation(t.Context(), withRuns)
		if err != nil {
			t.Fatal(err)
		}
		badState := filepath.Join(
			workspace, "repository_reviews", "repo_"+strings.Repeat("0", 64)+".json",
		)
		if err := os.WriteFile(badState, []byte(`{`), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := handler.repositoryReviewAutomationLedger(t.Context(), withRuns.ID); err == nil {
			t.Fatal("corrupt list ledger lookup succeeded")
		}
	})
}

func TestRepositoryReviewRemainingGatewayPublicationBranches(t *testing.T) {
	seedPublishable := func(t *testing.T) (*Handler, *http.ServeMux, string, repoaudit.RepositoryState, repoaudit.RepositoryReviewAutomation, repoaudit.IssueDraft) {
		t.Helper()
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		state := seedRepositoryReviewAPIState(t, workspace)
		state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
		automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
		store := repoaudit.NewStore(workspace)
		_, draft, _, err := store.ReserveIssueGeneration(repoaudit.IssueGenerationRequest{
			Repository: state.Repository, FindingID: state.Findings[0].ID,
			GenerationID: "rrig_gateway_remaining", ResolvedInstructions: repositoryReviewDefaultIssueInstructions,
			InstructionsMode: repoaudit.IssueDraftInstructionsDefault,
			GeneratorModel:   "cheap", GeneratorAccount: "api",
		})
		if err != nil {
			t.Fatal(err)
		}
		state, draft, err = store.CompleteIssueGeneration(
			state.Repository, draft.ID, draft.AttemptGenerationID,
			"Preview", "Evidence", []string{"bug"}, "",
		)
		if err != nil {
			t.Fatal(err)
		}
		return handler, mux, workspace, state, automation, draft
	}

	t.Run("cross-site single and batch", func(t *testing.T) {
		_, mux, _, state, automation, draft := seedPublishable(t)
		for _, target := range []string{
			baseAutomationIssuePath(automation.ID, draft.ID) + "/publish",
			"/api/repository-reviews/automations/" + automation.ID + "/issues/publish",
		} {
			request := httptest.NewRequest(
				http.MethodPost, "http://launcher.local"+target,
				strings.NewReader(`{"expected_version":1,"confirmed":true}`),
			)
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Sec-Fetch-Site", "cross-site")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("cross publish %q=%d %s state=%s", target, response.Code, response.Body.String(), state.ID)
			}
		}
	})

	t.Run("unknown outcome", func(t *testing.T) {
		_, mux, _, _, automation, draft := seedPublishable(t)
		installEventProxyStubs(t, func(_ *http.Request, _ time.Duration) (*http.Response, error) {
			return eventUpstreamResponse(
				http.StatusAccepted,
				`{"outcome":"unknown","draft":{"id":"`+draft.ID+`","state":"unknown"}}`,
			), nil
		})
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost,
			baseAutomationIssuePath(automation.ID, draft.ID)+"/publish",
			map[string]any{"expected_version": draft.Version, "confirmed": true},
		)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"outcome":"unknown"`) {
			t.Fatalf("unknown single publish=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("draft disappears after gateway", func(t *testing.T) {
		_, mux, workspace, state, automation, draft := seedPublishable(t)
		store := repoaudit.NewStore(workspace)
		installEventProxyStubs(t, func(_ *http.Request, _ time.Duration) (*http.Response, error) {
			_, _ = store.DeleteIssueDraft(state.Repository, draft.ID, draft.Version)
			return eventUpstreamResponse(http.StatusOK, `{"draft":{"id":"`+draft.ID+`","state":"posted"}}`), nil
		})
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost,
			baseAutomationIssuePath(automation.ID, draft.ID)+"/publish",
			map[string]any{"expected_version": draft.Version, "confirmed": true},
		)
		if response.Code != http.StatusNotFound {
			t.Fatalf("disappeared single publish=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("ledger disappears after gateway", func(t *testing.T) {
		_, mux, workspace, state, automation, draft := seedPublishable(t)
		installEventProxyStubs(t, func(_ *http.Request, _ time.Duration) (*http.Response, error) {
			statePath := filepath.Join(
				workspace, "repository_reviews", "repo_"+strings.TrimPrefix(state.ID, "rrp_")+".json",
			)
			_ = os.Remove(statePath)
			_ = os.Remove(strings.TrimSuffix(statePath, ".json") + ".summary.json")
			return eventUpstreamResponse(http.StatusOK, `{"draft":{"id":"`+draft.ID+`","state":"posted"}}`), nil
		})
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost,
			baseAutomationIssuePath(automation.ID, draft.ID)+"/publish",
			map[string]any{"expected_version": draft.Version, "confirmed": true},
		)
		if response.Code != http.StatusNotFound {
			t.Fatalf("missing ledger single publish=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("ledger corrupts after gateway", func(t *testing.T) {
		_, mux, workspace, _, automation, draft := seedPublishable(t)
		installEventProxyStubs(t, func(_ *http.Request, _ time.Duration) (*http.Response, error) {
			root := filepath.Join(workspace, "repository_reviews")
			_ = os.RemoveAll(root)
			_ = os.WriteFile(root, []byte("not a directory"), 0o600)
			return eventUpstreamResponse(http.StatusOK, `{"draft":{"id":"`+draft.ID+`","state":"posted"}}`), nil
		})
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost,
			baseAutomationIssuePath(automation.ID, draft.ID)+"/publish",
			map[string]any{"expected_version": draft.Version, "confirmed": true},
		)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("corrupt ledger single publish=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("noncanonical batch selection", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		state := seedRepositoryReviewAPIState(t, workspace)
		state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
		automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
		store := repoaudit.NewStore(workspace)
		state, older, err := store.PrepareIssue(repoaudit.IssueDraftRequest{
			Repository: state.Repository, FindingIDs: []string{state.Findings[0].ID},
			Title: "Older", Body: "Evidence", ExpectedVersion: state.Version,
		})
		if err != nil {
			t.Fatal(err)
		}
		state, _, err = store.PrepareIssue(repoaudit.IssueDraftRequest{
			Repository: state.Repository, FindingIDs: []string{state.Findings[0].ID},
			Title: "Newer", Body: "Evidence", ExpectedVersion: state.Version,
		})
		if err != nil {
			t.Fatal(err)
		}
		older, _ = repositoryReviewIssueByID(state, older.ID)
		if older.Canonical {
			t.Fatalf("older legacy draft stayed canonical: %#v", older)
		}
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost,
			"/api/repository-reviews/automations/"+automation.ID+"/issues/publish",
			map[string]any{
				"confirmed": true,
				"issues":    []map[string]any{{"id": older.ID, "expected_version": older.Version}},
			},
		)
		if response.Code != http.StatusOK ||
			!strings.Contains(response.Body.String(), `"code":"preview_not_canonical"`) {
			t.Fatalf("noncanonical batch=%d %s", response.Code, response.Body.String())
		}
	})
}

func baseAutomationIssuePath(automationID, draftID string) string {
	return "/api/repository-reviews/automations/" + automationID + "/issues/" + draftID
}

func seedRepositoryReviewDetailAutomation(
	t *testing.T,
	handler *Handler,
	repository string,
	runID string,
) repoaudit.RepositoryReviewAutomation {
	t.Helper()
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.ID = "rra_detail_" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_"))
	automation.Repository = repository
	automation.RunIDs = []string{runID}
	automation.StartedAt = time.Now().UTC().Add(-time.Hour)
	automation.IssueWriterModel = "cheap"
	automation, err = store.CreateAutomation(t.Context(), automation)
	if err != nil {
		t.Fatal(err)
	}
	return automation
}

func seedRepositoryReviewGenerationFindings(
	t *testing.T,
	workspace string,
	count int,
) repoaudit.RepositoryState {
	t.Helper()
	store := repoaudit.NewStore(workspace)
	files := make([]repoaudit.FileRef, count)
	observations := make([]repoaudit.Observation, count)
	for index := range count {
		files[index] = repoaudit.FileRef{
			Path:      fmt.Sprintf("pkg/file_%d.go", index+1),
			BlobSHA:   strings.Repeat(fmt.Sprintf("%x", index+1), 40),
			SizeBytes: 100, Category: "code", Mode: "100644",
		}
		observations[index] = repoaudit.Observation{
			Model: "review-model", ScopeFiles: []repoaudit.FileRef{files[index]},
			Findings: []repoaudit.FindingCandidate{{
				Severity: "high", Title: fmt.Sprintf("Finding %d", index+1), File: files[index].Path,
				Evidence: "The immutable source shows the failure.", Impact: "Data is lost.",
				Validation: repoaudit.Validation{Status: "confirmed", Summary: "Confirmed from source."},
			}},
		}
	}
	plan, err := store.Plan(
		context.Background(), "owner/repo-batch", strings.Repeat("a", 40),
		"inventory-batch", files, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.Record(context.Background(), repoaudit.RecordRequest{
		Plan: plan, RunID: "batch-generation-run", Observations: observations,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.State
}
