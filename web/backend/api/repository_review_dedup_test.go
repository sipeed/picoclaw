package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/repoaudit"
	"github.com/sipeed/picoclaw/pkg/workflows"
)

func TestRepositoryReviewProfileDeduplicationRequestPreservesExplicitZero(t *testing.T) {
	request := repositoryReviewProfileConfigRequest{
		DeduplicationThreshold:  repositoryReviewOptionalInt{Present: true, Value: 0},
		DeduplicationCandidates: repositoryReviewOptionalInt{Present: true, Value: 0},
	}
	profile := repositoryReviewProfileFromRequest(request)
	if profile.DeduplicationSimilarityThreshold != 0 ||
		profile.DeduplicationCandidateLimit != 0 ||
		!profile.DeduplicationSettingsSpecified ||
		!validRepositoryReviewDeduplicationRequest(request) {
		t.Fatalf("explicit zero deduplication settings = %#v", profile)
	}
	defaults := repositoryReviewProfileFromRequest(repositoryReviewProfileConfigRequest{})
	if defaults.DeduplicationSimilarityThreshold != repoaudit.DeduplicationDefaultThreshold ||
		defaults.DeduplicationCandidateLimit != repoaudit.DeduplicationDefaultCandidateLimit {
		t.Fatalf("default deduplication settings = %#v", defaults)
	}
}

func TestRepositoryReviewDeduplicationModelCallIsIsolatedAndStrict(t *testing.T) {
	handler := newRepositoryReviewAIAdjudicationHandler(t, http.StatusOK, `{}`)
	snapshot := repoaudit.RepositoryReviewDeduplicationSnapshot{
		ReviewerModel: "cheap", DeduplicationModel: "cheap", AccountRef: "api",
		SimilarityThreshold: 90, CandidateLimit: 4,
	}
	request := repoaudit.DeduplicationScoringRequest{
		Finding: repoaudit.DeduplicationDiagnosis{Title: "raw"},
		Candidates: []repoaudit.DeduplicationScoringCandidate{{
			ID: "candidate-000001", Diagnosis: repoaudit.DeduplicationDiagnosis{Title: "candidate"},
		}},
	}
	original := runRepositoryDeduplicationAgent
	t.Cleanup(func() { runRepositoryDeduplicationAgent = original })
	var captured workflows.AgentRequest
	runRepositoryDeduplicationAgent = func(
		_ context.Context,
		_ *webWorkflowRuntimeRunner,
		agentRequest workflows.AgentRequest,
	) (map[string]any, error) {
		captured = agentRequest
		return map[string]any{
			"structured_valid": true,
			"structured": map[string]any{"scores": []any{map[string]any{
				"candidate_id": "candidate-000001", "score": 90,
				"explanation": "Same mechanism.",
			}}},
		}, nil
	}
	response, err := runRepositoryReviewDeduplicationScoring(
		t.Context(), handler, snapshot, repoaudit.DeduplicationScoringInstructions, request,
	)
	if err != nil || repoaudit.ValidateDeduplicationScoringResponse(response, request) != nil {
		t.Fatalf("scoring response=%#v err=%v", response, err)
	}
	if captured.Tools != workflows.AgentToolsNone || captured.History != "none" ||
		captured.Cache != "none" || !captured.EphemeralSession || !captured.PrivateContext ||
		captured.AccountRef != snapshot.AccountRef || captured.Model != snapshot.DeduplicationModel {
		t.Fatalf("deduplication request was not isolated: %#v", captured)
	}
	drifted := snapshot
	drifted.AccountModelRevision = "stale-revision"
	if _, err := runRepositoryReviewDeduplicationScoring(
		t.Context(), handler, drifted, repoaudit.DeduplicationScoringInstructions, request,
	); err == nil || !strings.Contains(err.Error(), "revision changed") {
		t.Fatalf("stale account/model revision error = %v", err)
	}
	runRepositoryDeduplicationAgent = func(
		context.Context, *webWorkflowRuntimeRunner, workflows.AgentRequest,
	) (map[string]any, error) {
		return map[string]any{
			"structured_valid": true,
			"structured":       map[string]any{"decision": "new", "unexpected": true},
		}, nil
	}
	if _, err := runRepositoryReviewDeduplicationJudgment(
		t.Context(), handler, snapshot, repoaudit.DeduplicationJudgeInstructions,
		repoaudit.DeduplicationJudgeRequest{},
	); err == nil {
		t.Fatal("deduplication judgment accepted an unknown field")
	}
}

func TestRepositoryReviewDeduplicatedFindingAndRawProcessingRoutes(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	campaignID := "rrc_api_dedup"
	state.HistoricalDeduplication = repoaudit.HistoricalDeduplicationReplay{
		Required: true, Status: repoaudit.HistoricalDeduplicationFailed,
		Attempts: 2, Error: "historical replay failed", UpdatedAt: time.Now().UTC(),
	}
	state = seedRepositoryReviewDeduplicationAPIState(t, workspace, state, campaignID)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
	base := "/api/repository-reviews/automations/" + automation.ID

	findings := httptest.NewRecorder()
	mux.ServeHTTP(findings, httptest.NewRequest(
		http.MethodGet, base+"/findings?query=ALL", nil,
	))
	if findings.Code != http.StatusOK ||
		!strings.Contains(findings.Body.String(), `"raw_source_count":1`) ||
		!strings.Contains(findings.Body.String(), `"contributors":["review-model"]`) ||
		!strings.Contains(findings.Body.String(), state.DeduplicatedFindings[0].ID) ||
		!strings.Contains(findings.Body.String(), `"findings_processing"`) ||
		!strings.Contains(findings.Body.String(), `"historical_deduplication"`) ||
		strings.Contains(findings.Body.String(), state.RawFindings[1].ID) {
		t.Fatalf("deduplicated findings status=%d body=%s", findings.Code, findings.Body.String())
	}

	rawCollection := httptest.NewRecorder()
	mux.ServeHTTP(rawCollection, httptest.NewRequest(
		http.MethodGet, base+"/raw-findings?limit=2", nil,
	))
	var rawPage struct {
		RawFindings    []repositoryReviewRawFindingSummary `json:"raw_findings"`
		Total          int                                 `json:"total"`
		NextCursor     string                              `json:"next_cursor"`
		CanonicalQuery string                              `json:"canonical_query"`
	}
	if err := json.Unmarshal(rawCollection.Body.Bytes(), &rawPage); err != nil {
		t.Fatal(err)
	}
	if rawCollection.Code != http.StatusOK || rawPage.Total != 3 || len(rawPage.RawFindings) != 2 ||
		rawPage.NextCursor == "" || rawPage.CanonicalQuery != "ALL ORDER BY created DESC" ||
		rawPage.RawFindings[0].ID != state.RawFindings[2].ID ||
		!strings.Contains(rawCollection.Body.String(), `"findings_processing"`) ||
		!strings.Contains(rawCollection.Body.String(), `"historical_deduplication"`) {
		t.Fatalf("raw findings status=%d page=%#v body=%s", rawCollection.Code, rawPage, rawCollection.Body.String())
	}

	failedRawCollection := httptest.NewRecorder()
	mux.ServeHTTP(failedRawCollection, httptest.NewRequest(
		http.MethodGet,
		base+"/raw-findings?query="+url.QueryEscape("deduplication_state = failed ORDER BY created DESC"),
		nil,
	))
	var failedRawPage struct {
		RawFindings []repositoryReviewRawFindingSummary `json:"raw_findings"`
		Total       int                                 `json:"total"`
	}
	if err := json.Unmarshal(failedRawCollection.Body.Bytes(), &failedRawPage); err != nil {
		t.Fatal(err)
	}
	if failedRawCollection.Code != http.StatusOK || failedRawPage.Total != 1 ||
		len(failedRawPage.RawFindings) != 1 || failedRawPage.RawFindings[0].ID != state.RawFindings[2].ID {
		t.Fatalf("filtered raw findings status=%d body=%s", failedRawCollection.Code, failedRawCollection.Body.String())
	}

	wrongCursor := httptest.NewRecorder()
	mux.ServeHTTP(wrongCursor, httptest.NewRequest(
		http.MethodGet,
		base+"/findings?query=ALL&cursor="+url.QueryEscape(rawPage.NextCursor),
		nil,
	))
	if wrongCursor.Code != http.StatusBadRequest ||
		!strings.Contains(wrongCursor.Body.String(), `"code":"invalid_cursor"`) {
		t.Fatalf("wrong raw cursor status=%d body=%s", wrongCursor.Code, wrongCursor.Body.String())
	}

	runFindings := httptest.NewRecorder()
	mux.ServeHTTP(runFindings, httptest.NewRequest(
		http.MethodGet, base+"/run-findings?query=ALL", nil,
	))
	var legacyPage struct {
		Findings []repositoryReviewRunFindingSummary `json:"findings"`
		Total    int                                 `json:"total"`
	}
	if err := json.Unmarshal(runFindings.Body.Bytes(), &legacyPage); err != nil {
		t.Fatal(err)
	}
	if runFindings.Code != http.StatusOK || legacyPage.Total != 0 || len(legacyPage.Findings) != 0 {
		t.Fatalf("run findings status=%d body=%s", runFindings.Code, runFindings.Body.String())
	}

	detail := httptest.NewRecorder()
	mux.ServeHTTP(detail, httptest.NewRequest(
		http.MethodGet, base+"/findings/"+state.DeduplicatedFindings[0].ID, nil,
	))
	if detail.Code != http.StatusOK ||
		!strings.Contains(detail.Body.String(), `"raw_source_total":1`) ||
		!strings.Contains(detail.Body.String(), state.DeduplicatedFindings[0].Evidence) {
		t.Fatalf("deduplicated detail status=%d body=%s", detail.Code, detail.Body.String())
	}
	if len(state.RepositoryFindings) > 0 {
		strict := httptest.NewRecorder()
		mux.ServeHTTP(strict, httptest.NewRequest(
			http.MethodGet, base+"/findings/"+state.RepositoryFindings[0].ID, nil,
		))
		if strict.Code != http.StatusNotFound {
			t.Fatalf("rdf-only detail accepted repository finding: status=%d body=%s", strict.Code, strict.Body.String())
		}
	}

	sources := httptest.NewRecorder()
	mux.ServeHTTP(sources, httptest.NewRequest(
		http.MethodGet,
		base+"/findings/"+state.DeduplicatedFindings[0].ID+"/sources?offset=0&limit=1",
		nil,
	))
	if sources.Code != http.StatusOK ||
		!strings.Contains(sources.Body.String(), state.RawFindings[0].ID) ||
		!strings.Contains(sources.Body.String(), `"model":"provider/review-model"`) ||
		!strings.Contains(sources.Body.String(), `"model_alias":"review-model"`) ||
		!strings.Contains(sources.Body.String(), `"account":"api"`) ||
		strings.Contains(sources.Body.String(), state.RawFindings[0].Evidence) {
		t.Fatalf("raw sources status=%d body=%s", sources.Code, sources.Body.String())
	}

	processing := httptest.NewRecorder()
	mux.ServeHTTP(processing, httptest.NewRequest(
		http.MethodGet,
		base+"/campaigns/"+campaignID+"/findings-processing?offset=0&limit=10",
		nil,
	))
	if processing.Code != http.StatusOK ||
		!strings.Contains(processing.Body.String(), `"raw_total":3`) ||
		!strings.Contains(processing.Body.String(), `"pending":1`) ||
		!strings.Contains(processing.Body.String(), `"failed":1`) ||
		!strings.Contains(processing.Body.String(), state.RawFindings[1].ID) ||
		!strings.Contains(processing.Body.String(), state.RawFindings[2].ID) {
		t.Fatalf("processing status=%d body=%s", processing.Code, processing.Body.String())
	}

	failedDetail := httptest.NewRecorder()
	mux.ServeHTTP(failedDetail, httptest.NewRequest(
		http.MethodGet,
		base+"/campaigns/"+campaignID+"/findings-processing/sources/"+state.RawFindings[2].ID,
		nil,
	))
	if failedDetail.Code != http.StatusOK ||
		!strings.Contains(failedDetail.Body.String(), state.RawFindings[2].Evidence) ||
		!strings.Contains(failedDetail.Body.String(), `"model":"provider/review-model"`) ||
		!strings.Contains(failedDetail.Body.String(), `"model_alias":"review-model"`) ||
		!strings.Contains(failedDetail.Body.String(), `"account":"api"`) ||
		!strings.Contains(failedDetail.Body.String(), `"retryable":true`) {
		t.Fatalf("failed raw detail status=%d body=%s", failedDetail.Code, failedDetail.Body.String())
	}
	canonicalDetail := httptest.NewRecorder()
	mux.ServeHTTP(canonicalDetail, httptest.NewRequest(
		http.MethodGet, base+"/raw-findings/"+state.RawFindings[0].ID, nil,
	))
	var canonicalPayload struct {
		HistoricalDeduplication repoaudit.HistoricalDeduplicationReplay `json:"historical_deduplication"`
	}
	if err := json.Unmarshal(canonicalDetail.Body.Bytes(), &canonicalPayload); err != nil {
		t.Fatal(err)
	}
	if canonicalDetail.Code != http.StatusOK ||
		!strings.Contains(canonicalDetail.Body.String(), state.RawFindings[0].Evidence) ||
		!strings.Contains(canonicalDetail.Body.String(), `"context"`) ||
		!strings.Contains(canonicalDetail.Body.String(), `"finding"`) ||
		canonicalPayload.HistoricalDeduplication.Status != repoaudit.HistoricalDeduplicationFailed ||
		canonicalPayload.HistoricalDeduplication.Error != "historical replay failed" {
		t.Fatalf("canonical raw detail status=%d body=%s", canonicalDetail.Code, canonicalDetail.Body.String())
	}
	aliasDetail := httptest.NewRecorder()
	mux.ServeHTTP(aliasDetail, httptest.NewRequest(
		http.MethodGet, base+"/raw-findings/rfn_api_legacy", nil,
	))
	if aliasDetail.Code != http.StatusOK ||
		!strings.Contains(aliasDetail.Body.String(), `"id":"`+state.RawFindings[0].ID+`"`) {
		t.Fatalf("raw alias detail status=%d body=%s", aliasDetail.Code, aliasDetail.Body.String())
	}

	retryRequest := httptest.NewRequest(
		http.MethodPost,
		base+"/raw-findings/"+state.RawFindings[2].ID+"/retry",
		strings.NewReader(`{}`),
	)
	setRepositoryReviewMutationHeaders(retryRequest)
	retried := httptest.NewRecorder()
	mux.ServeHTTP(retried, retryRequest)
	var retriedPayload struct {
		Source repoaudit.RawReviewFinding `json:"source"`
	}
	if decodeErr := json.Unmarshal(retried.Body.Bytes(), &retriedPayload); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if retried.Code != http.StatusAccepted ||
		retriedPayload.Source.State != repoaudit.RawFindingDeduplicationPending ||
		retriedPayload.Source.Failure != nil {
		t.Fatalf("retry status=%d body=%s", retried.Code, retried.Body.String())
	}
	updated, found, err := repoaudit.NewStore(workspace).Get(state.Repository)
	if err != nil || !found {
		t.Fatalf("updated state found=%v err=%v", found, err)
	}
	var retriedRaw repoaudit.RawReviewFinding
	var retriedJob repoaudit.DeduplicationJob
	for _, raw := range updated.RawFindings {
		if raw.ID == state.RawFindings[2].ID {
			retriedRaw = raw
		}
	}
	for _, job := range updated.DeduplicationJobs {
		if job.RawFindingID == retriedRaw.ID {
			retriedJob = job
		}
	}
	if retriedRaw.State != repoaudit.RawFindingDeduplicationPending ||
		retriedRaw.InsertionOrdinal != 3 || retriedJob.InsertionOrdinal != 4 ||
		updated.NextDeduplicationOrdinal != 5 {
		t.Fatalf(
			"retried raw=%#v job=%#v next=%d",
			retriedRaw, retriedJob, updated.NextDeduplicationOrdinal,
		)
	}
}

func TestHistoricalDeduplicationConflictIsRetryable(t *testing.T) {
	response := httptest.NewRecorder()
	writeRepositoryReviewError(response, repoaudit.ErrHistoricalDeduplicationInProgress)
	if response.Code != http.StatusConflict ||
		!strings.Contains(response.Body.String(), `"code":"historical_deduplication_in_progress"`) {
		t.Fatalf("historical conflict status=%d body=%s", response.Code, response.Body.String())
	}
}

func seedRepositoryReviewDeduplicationAPIState(
	t *testing.T,
	workspace string,
	state repoaudit.RepositoryState,
	campaignID string,
) repoaudit.RepositoryState {
	t.Helper()
	finding := state.Findings[0]
	finding.CommitSHA = strings.Repeat("f", 40)
	contextRecord := state.Contexts[0]
	now := finding.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.Add(-time.Minute)
	snapshot := repoaudit.RepositoryReviewDeduplicationSnapshot{
		ReviewerModel: "review-model", DeduplicationModel: "review-model",
		SimilarityThreshold: 90, CandidateLimit: 4,
	}
	raw := func(
		id string,
		ordinal uint64,
		stateValue repoaudit.RawFindingDeduplicationState,
	) repoaudit.RawReviewFinding {
		created := now.Add(time.Duration(ordinal) * time.Second)
		value := repoaudit.RawReviewFinding{
			ID: id, Version: 1, CampaignID: campaignID,
			AdmissionBucket: "rdb_api_bucket", InsertionOrdinal: ordinal,
			Repository: finding.Repository, CommitSHA: finding.CommitSHA,
			File: finding.File, Line: finding.Line, Severity: finding.Severity,
			Title: finding.Title, Symbol: finding.Symbol, Message: finding.Message,
			Evidence: finding.Evidence, Impact: finding.Impact,
			Validation: finding.Validation, MatchHints: finding.MatchHints,
			FixEffort: finding.FixEffort, ContextID: contextRecord.ID,
			RunID: state.Runs[0].ID, AssignmentID: "assignment-api",
			Model: "provider/review-model", ModelAlias: "review-model", Account: "api",
			Reviewer: "review-model", State: stateValue,
			Disposition: repoaudit.RawFindingDispositionUndecided,
			CreatedAt:   created, UpdatedAt: created,
		}
		value.DiagnosisDigest = repoaudit.RawReviewFindingDiagnosisDigest(value)
		return value
	}
	completed := raw("rrw_api_completed", 1, repoaudit.RawFindingDeduplicationCompleted)
	completed.LegacyFindingID = "rfn_api_legacy"
	completed.DiagnosisDigest = repoaudit.RawReviewFindingDiagnosisDigest(completed)
	completed.Disposition = repoaudit.RawFindingDispositionNew
	completed.DeduplicatedFindingID = finding.ID
	completed.History = []repoaudit.RawFindingHistoryEntry{{
		State: completed.State, Disposition: completed.Disposition,
		DeduplicatedFindingID: finding.ID, Attempt: 1, At: completed.UpdatedAt,
	}}
	pending := raw("rrw_api_pending", 2, repoaudit.RawFindingDeduplicationPending)
	pending.History = []repoaudit.RawFindingHistoryEntry{{
		State: pending.State, Disposition: pending.Disposition, At: pending.UpdatedAt,
	}}
	failed := raw("rrw_api_failed", 3, repoaudit.RawFindingDeduplicationFailed)
	failure := &repoaudit.DeduplicationFailure{
		Code: "provider_failed", Message: "Deduplication failed.", Retryable: true,
		At: failed.UpdatedAt,
	}
	failed.Failure = failure
	failed.History = []repoaudit.RawFindingHistoryEntry{{
		State: failed.State, Disposition: failed.Disposition, Attempt: 3,
		Failure: failure, At: failed.UpdatedAt,
	}}
	state.RawFindings = []repoaudit.RawReviewFinding{completed, pending, failed}
	state.DeduplicatedFindings = []repoaudit.DeduplicatedReviewFinding{{
		ID: finding.ID, Version: 1, CampaignID: campaignID,
		AdmissionBucket: completed.AdmissionBucket, CreationOrdinal: 1,
		DiagnosisDigest: completed.DiagnosisDigest,
		Repository:      finding.Repository, CommitSHA: finding.CommitSHA,
		File: finding.File, Line: finding.Line, Severity: finding.Severity,
		Title: finding.Title, Symbol: finding.Symbol, Message: finding.Message,
		Evidence: finding.Evidence, Impact: finding.Impact,
		Validation: finding.Validation, MatchHints: finding.MatchHints,
		FixEffort: finding.FixEffort, RawSourceIDs: []string{completed.ID},
		History: []repoaudit.DeduplicatedFindingHistoryEntry{{
			Action: "created", RawFindingID: completed.ID, At: completed.UpdatedAt,
		}},
		Status: finding.Status, IssueDraftID: finding.IssueDraftID,
		RepositoryFindingID:     finding.RepositoryFindingID,
		RepositoryMatchState:    finding.RepositoryMatchState,
		TargetBranch:            finding.TargetBranch,
		AdvertisedDefaultBranch: finding.AdvertisedDefaultBranch,
		TargetIsDefault:         finding.TargetIsDefault,
		CreatedAt:               completed.CreatedAt, UpdatedAt: completed.UpdatedAt,
	}}
	state.DeduplicationJobs = []repoaudit.DeduplicationJob{
		{
			ID: "rdj_api_completed", RawFindingID: completed.ID,
			State:           repoaudit.DeduplicationJobCompleted,
			AdmissionBucket: completed.AdmissionBucket, InsertionOrdinal: 1,
			Attempts: 1, ModelSnapshot: snapshot,
			Decision: repoaudit.DeduplicationJudgment{Decision: "new"},
			History: []repoaudit.DeduplicationJobHistoryEntry{{
				State: repoaudit.DeduplicationJobCompleted, Attempt: 1, At: completed.UpdatedAt,
			}},
			CreatedAt: completed.CreatedAt, UpdatedAt: completed.UpdatedAt,
		},
		{
			ID: "rdj_api_pending", RawFindingID: pending.ID,
			State:           repoaudit.DeduplicationJobPending,
			AdmissionBucket: pending.AdmissionBucket, InsertionOrdinal: 2,
			ModelSnapshot: snapshot,
			History: []repoaudit.DeduplicationJobHistoryEntry{{
				State: repoaudit.DeduplicationJobPending, At: pending.UpdatedAt,
			}},
			CreatedAt: pending.CreatedAt, UpdatedAt: pending.UpdatedAt,
		},
		{
			ID: "rdj_api_failed", RawFindingID: failed.ID,
			State:           repoaudit.DeduplicationJobFailed,
			AdmissionBucket: failed.AdmissionBucket, InsertionOrdinal: 3,
			Attempts: 3, ModelSnapshot: snapshot, Failure: failure,
			History: []repoaudit.DeduplicationJobHistoryEntry{{
				State: repoaudit.DeduplicationJobFailed, Attempt: 3,
				Failure: failure, At: failed.UpdatedAt,
			}},
			CreatedAt: failed.CreatedAt, UpdatedAt: failed.UpdatedAt,
		},
	}
	state.NextDeduplicationOrdinal = 4
	state.FindingsProcessing = repoaudit.FindingsProcessingCounters{
		RawTotal: 3, Pending: 1, Failed: 1, Completed: 1, New: 1,
		UpdatedAt: failed.UpdatedAt,
	}
	state.Version++
	state.UpdatedAt = failed.UpdatedAt
	paths, err := filepath.Glob(filepath.Join(workspace, "repository_reviews", "repo_*.json"))
	if err != nil {
		t.Fatal(err)
	}
	statePath := ""
	for _, path := range paths {
		if !strings.HasSuffix(path, ".summary.json") {
			statePath = path
			break
		}
	}
	if statePath == "" {
		t.Fatal("repository review state path is missing")
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if writeErr := os.WriteFile(statePath, data, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	loaded, found, err := repoaudit.NewStore(workspace).Get(state.Repository)
	if err != nil || !found {
		t.Fatalf("seeded deduplication state found=%v err=%v", found, err)
	}
	return loaded
}
