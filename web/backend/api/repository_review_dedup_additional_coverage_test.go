package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/collectionquery"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/repoaudit"
	"github.com/sipeed/picoclaw/pkg/workflows"
)

func TestRepositoryReviewDedupAdditionalProjectionAndPageCoverage(t *testing.T) {
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	line := 17
	summary := repositoryReviewDeduplicatedFindingSummary{
		ID: "rdf_coverage", Repository: "owner/repo", Path: "pkg/core.go", Line: &line,
		Severity: "high", Title: "Lost update", Symbol: "Save", Status: repoaudit.FindingOpen,
		RunFindingStatus: repositoryReviewRunFindingAssociatedExisting, Association: "existing",
		RepositoryFindingID: "rrf_coverage", Contributors: []string{"model-a", "reviewer-a"},
		RawSourceCount: 2, CreatedAt: now.Add(-time.Hour), UpdatedAt: now,
	}
	options := repositoryReviewDeduplicatedFindingPageOptions(
		repositoryReviewCollectionCursorContext("deduplicated-findings", "rra_coverage", "rrc_coverage"),
	)
	id, err := options.ID(summary)
	if err != nil || id == "" || !options.ValidateID(id) {
		t.Fatalf("deduplicated page id=%q err=%v", id, err)
	}
	for _, field := range repositoryReviewDeduplicatedFindingCollectionSchema.Fields {
		if _, ok := options.Resolve(summary, field.Name, now); !ok {
			t.Fatalf("deduplicated field %q was unresolved", field.Name)
		}
	}
	if _, ok := options.Resolve(summary, collectionquery.Field("unknown"), now); ok {
		t.Fatal("unknown deduplicated field resolved")
	}

	for _, request := range []*http.Request{
		nil,
		{URL: nil},
		httptest.NewRequest(http.MethodGet, "/?other=1", nil),
		httptest.NewRequest(http.MethodGet, "/?offset=1&offset=2", nil),
		httptest.NewRequest(http.MethodGet, "/?offset=-1", nil),
		httptest.NewRequest(http.MethodGet, "/?limit=201", nil),
	} {
		if _, _, pageErr := repositoryReviewRawPage(request); pageErr == nil {
			t.Fatalf("raw page accepted request %#v", request)
		}
	}
	offset, limit, err := repositoryReviewRawPage(httptest.NewRequest(http.MethodGet, "/?offset=2&limit=3", nil))
	if err != nil || offset != 2 || limit != 3 {
		t.Fatalf("raw page offset=%d limit=%d err=%v", offset, limit, err)
	}

	invalidProcessing := []string{
		"/?other=1", "/?state=unknown", "/?offset=-1", "/?limit=201", "/?state=pending&state=failed",
	}
	if _, _, _, pageErr := repositoryReviewFindingsProcessingPage(nil); pageErr == nil {
		t.Fatal("nil findings-processing request was accepted")
	}
	for _, target := range invalidProcessing {
		if _, _, _, pageErr := repositoryReviewFindingsProcessingPage(
			httptest.NewRequest(http.MethodGet, target, nil),
		); pageErr == nil {
			t.Fatalf("findings-processing page accepted %q", target)
		}
	}
	for _, state := range []repoaudit.RawFindingDeduplicationState{
		repoaudit.RawFindingDeduplicationPending,
		repoaudit.RawFindingDeduplicationRunning,
		repoaudit.RawFindingDeduplicationFailed,
		repoaudit.RawFindingDeduplicationCompleted,
	} {
		_, _, filter, pageErr := repositoryReviewFindingsProcessingPage(httptest.NewRequest(
			http.MethodGet, "/?state="+string(state), nil,
		))
		if pageErr != nil || filter != string(state) {
			t.Fatalf("findings-processing state=%q filter=%q err=%v", state, filter, pageErr)
		}
	}

	findings := []repoaudit.RawReviewFinding{
		{State: repoaudit.RawFindingDeduplicationPending, UpdatedAt: now.Add(-4 * time.Minute)},
		{State: repoaudit.RawFindingDeduplicationRunning, UpdatedAt: now.Add(-3 * time.Minute)},
		{State: repoaudit.RawFindingDeduplicationFailed, UpdatedAt: now.Add(-2 * time.Minute)},
		{
			State:       repoaudit.RawFindingDeduplicationCompleted,
			Disposition: repoaudit.RawFindingDispositionNew, UpdatedAt: now.Add(-time.Minute),
		},
		{
			State:       repoaudit.RawFindingDeduplicationCompleted,
			Disposition: repoaudit.RawFindingDispositionDuplicate, UpdatedAt: now,
		},
	}
	counters := repositoryReviewFindingsProcessingCounters(findings)
	if counters.RawTotal != 5 || counters.Pending != 1 || counters.Processing != 1 ||
		counters.Failed != 1 || counters.Completed != 2 || counters.New != 1 ||
		counters.Duplicates != 1 || !counters.UpdatedAt.Equal(now) {
		t.Fatalf("findings-processing counters=%#v", counters)
	}
	if !containsRepositoryReviewSourceID([]string{"one", "two"}, "two") ||
		containsRepositoryReviewSourceID([]string{"one"}, "missing") {
		t.Fatal("source membership coverage mismatch")
	}
	contributors := appendUniqueRepositoryReviewContributor(nil, " model-a ")
	contributors = appendUniqueRepositoryReviewContributor(contributors, "model-a")
	contributors = appendUniqueRepositoryReviewContributor(contributors, " ")
	if len(contributors) != 1 || contributors[0] != "model-a" {
		t.Fatalf("contributors=%#v", contributors)
	}
}

func TestRepositoryReviewDedupAdditionalRouteCoverage(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	campaignID := "rrc_additional_coverage"
	state = seedRepositoryReviewDeduplicationAPIState(t, workspace, state, campaignID)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
	base := "/api/repository-reviews/automations/" + automation.ID

	requests := []struct {
		path string
		want int
	}{
		{base + "/findings?query=(", http.StatusBadRequest},
		{base + "/findings/rdf_missing", http.StatusNotFound},
		{base + "/findings/" + state.DeduplicatedFindings[0].ID + "/sources?other=1", http.StatusBadRequest},
		{base + "/findings/rdf_missing/sources", http.StatusNotFound},
		{base + "/findings/" + state.DeduplicatedFindings[0].ID + "/sources/rrw_missing", http.StatusNotFound},
		{base + "/campaigns/wrong/findings-processing/sources/" + state.RawFindings[0].ID, http.StatusNotFound},
		{base + "/campaigns/" + campaignID + "/findings-processing?state=unknown", http.StatusBadRequest},
		{base + "/findings-processing", http.StatusOK},
	}
	for _, test := range requests {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if response.Code != test.want {
			t.Fatalf("GET %s status=%d want=%d body=%s", test.path, response.Code, test.want, response.Body.String())
		}
	}

	detail := httptest.NewRecorder()
	mux.ServeHTTP(detail, httptest.NewRequest(
		http.MethodGet,
		base+"/findings/"+state.DeduplicatedFindings[0].ID+"/sources/"+state.RawFindings[0].ID,
		nil,
	))
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), `"context"`) ||
		!strings.Contains(detail.Body.String(), `"finding"`) {
		t.Fatalf("raw source detail status=%d body=%s", detail.Code, detail.Body.String())
	}

	filtered := httptest.NewRecorder()
	mux.ServeHTTP(filtered, httptest.NewRequest(
		http.MethodGet,
		base+"/campaigns/"+campaignID+"/findings-processing?state=pending&offset=0&limit=1",
		nil,
	))
	if filtered.Code != http.StatusOK || !strings.Contains(filtered.Body.String(), state.RawFindings[1].ID) {
		t.Fatalf("filtered processing status=%d body=%s", filtered.Code, filtered.Body.String())
	}

	for _, target := range []string{
		base + "/campaigns/" + campaignID + "/findings-processing/sources/" + state.RawFindings[1].ID + "/retry?query=1",
		base + "/campaigns/" + campaignID + "/findings-processing/sources/" + state.RawFindings[1].ID + "/retry",
	} {
		request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(`{}`))
		if !strings.Contains(target, "query=1") {
			setRepositoryReviewMutationHeaders(request)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest && response.Code != http.StatusConflict {
			t.Fatalf("retry %s status=%d body=%s", target, response.Code, response.Body.String())
		}
	}
}

func TestRepositoryReviewHistoricalDedupAdditionalRouteCoverage(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ModelList[0].APIKeys = config.SecureStrings{config.NewSecureString("test-api-key")}
	if saveErr := config.SaveConfig(handler.configPath, cfg); saveErr != nil {
		t.Fatal(saveErr)
	}
	state := seedRepositoryReviewAPIState(t, workspace)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
	now := time.Now().UTC()
	state.RawFindings = nil
	state.DeduplicationJobs = nil
	state.DeduplicatedFindings = nil
	state.FindingsProcessing = repoaudit.FindingsProcessingCounters{}
	for index := range state.Findings {
		state.Findings[index].DeduplicationPending = false
		state.Findings[index].CommitSHA = strings.Repeat("a", 40)
	}
	secondHistorical := state.Findings[0]
	secondHistorical.ID = "rfn_historical_additional_second"
	secondHistorical.Fingerprint = "historical-additional-second"
	secondHistorical.Title = "Second historical finding"
	secondHistorical.RepositoryFindingID = ""
	secondHistorical.RepositoryMatchState = ""
	state.Findings = append(state.Findings, secondHistorical)
	state.HistoricalDeduplication = repoaudit.HistoricalDeduplicationReplay{
		Required: true, Status: repoaudit.HistoricalDeduplicationPending, UpdatedAt: now,
	}
	persistRepositoryReviewAdditionalCoverageState(t, workspace, state)
	store := repoaudit.NewStore(workspace)
	snapshot := repoaudit.RepositoryReviewDeduplicationSnapshot{
		ReviewerModel: "cheap", DeduplicationModel: "cheap", AccountRef: "api",
		SimilarityThreshold: 90, CandidateLimit: 4,
	}
	state, _, err = store.FreezeHistoricalDeduplicationReplay(state.Repository, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	state, admission, err := store.AdmitNextHistoricalDeduplicationBatch(state.Repository)
	if err != nil || admission.Admitted == 0 {
		t.Fatalf("historical admission=%#v err=%v", admission, err)
	}
	_, _, err = store.FailHistoricalDeduplicationReplay(state.Repository, "")
	if err != nil {
		t.Fatal(err)
	}
	base := "/api/repository-reviews/automations/" + automation.ID + "/historical-deduplication"

	invalid := httptest.NewRecorder()
	mux.ServeHTTP(invalid, httptest.NewRequest(http.MethodGet, base+"?other=1", nil))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("historical invalid page status=%d body=%s", invalid.Code, invalid.Body.String())
	}
	page := httptest.NewRecorder()
	mux.ServeHTTP(page, httptest.NewRequest(http.MethodGet, base+"?offset=0&limit=1", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), admission.RawFindings[0].ID) ||
		!strings.Contains(page.Body.String(), `"next_offset":1`) {
		t.Fatalf("historical page status=%d body=%s", page.Code, page.Body.String())
	}

	badRetry := httptest.NewRecorder()
	mux.ServeHTTP(badRetry, httptest.NewRequest(http.MethodPost, base+"/retry?query=1", strings.NewReader(`{}`)))
	if badRetry.Code != http.StatusBadRequest {
		t.Fatalf("historical invalid retry status=%d body=%s", badRetry.Code, badRetry.Body.String())
	}
	retryRequest := httptest.NewRequest(http.MethodPost, base+"/retry", strings.NewReader(`{}`))
	setRepositoryReviewMutationHeaders(retryRequest)
	retried := httptest.NewRecorder()
	mux.ServeHTTP(retried, retryRequest)
	if retried.Code != http.StatusConflict || !strings.Contains(
		retried.Body.String(), `"code":"historical_deduplication_campaign_recovery_required"`,
	) {
		t.Fatalf("historical retry status=%d body=%s", retried.Code, retried.Body.String())
	}
	unchanged, found, err := store.Get(state.Repository)
	if err != nil || !found ||
		unchanged.HistoricalDeduplication.Status != repoaudit.HistoricalDeduplicationFailed {
		t.Fatalf("inexact retry mutated replay=%#v found=%v err=%v", unchanged.HistoricalDeduplication, found, err)
	}
}

func TestRepositoryReviewHistoricalRetryRecoversOneCampaignAcrossAutomationRuns(t *testing.T) {
	fixture := newRepositoryReviewBackfillFixture(
		t,
		2,
		repositoryReviewBackfillRunSpec{inspected: []int{0}, occurrences: 1},
		repositoryReviewBackfillRunSpec{inspected: []int{0}, occurrences: 1},
	)
	state := fixture.state
	state.RawFindings = nil
	state.DeduplicationJobs = nil
	state.DeduplicatedFindings = nil
	state.MappingJobs = nil
	state.NextDeduplicationOrdinal = 0
	state.FindingsProcessing = repoaudit.FindingsProcessingCounters{}
	state.HistoricalDeduplication = repoaudit.HistoricalDeduplicationReplay{
		Required: true, Status: repoaudit.HistoricalDeduplicationPending,
		UpdatedAt: time.Now().UTC(),
	}
	state.Version++
	state.UpdatedAt = state.HistoricalDeduplication.UpdatedAt
	persistRepositoryReviewAdditionalCoverageState(t, fixture.workspace, state)
	snapshot := repoaudit.RepositoryReviewDeduplicationSnapshot{
		ReviewerModel: "review-a", DeduplicationModel: "review-a",
		AccountRef:          fixture.automation.EffectiveAccountRef,
		SimilarityThreshold: 90, CandidateLimit: 4,
	}
	state, _, err := fixture.store.FreezeHistoricalDeduplicationReplay(state.Repository, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	state, admission, err := fixture.store.AdmitNextHistoricalDeduplicationBatch(state.Repository)
	if err != nil || admission.Admitted == 0 {
		t.Fatalf("synthetic pre-recovery admission=%#v err=%v", admission, err)
	}
	state, _, err = fixture.store.FailHistoricalDeduplicationReplay(state.Repository, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(repositoryReviewHistoricalRecoveryProjection(state).Contexts) >= len(state.Contexts) {
		t.Fatal("replay-created context was not removed from exact recovery projection")
	}
	resolved := workflows.RepositoryReviewModelProfile{
		Revision:        "legacy-automation-profile",
		AccountRef:      fixture.automation.EffectiveAccountRef,
		ReviewerModels:  fixture.automation.ReviewerModels,
		MaxContentBytes: int(fixture.automation.MaxContentBytes),
	}
	projected := repositoryReviewHistoricalRecoveryProjection(state)
	prepared, prepareErr := prepareRepositoryReviewLegacyCampaignBackfill(
		t.Context(), fixture.automation, projected, repoaudit.NewRepositoryReviewCampaignID(),
		fixture.runStore, resolved,
	)
	if prepareErr != nil || !prepared.Available || !prepared.Exact {
		t.Fatalf("exact pre-retry recovery projection=%#v err=%v", prepared, prepareErr)
	}
	final, recovered, err := recoverRepositoryReviewHistoricalCampaign(
		t.Context(), fixture.store, fixture.workspace, fixture.automation, state, resolved,
	)
	if err != nil || !repoaudit.ValidRepositoryReviewCampaignID(final.CampaignID) ||
		final.CampaignRecoveryPending || recovered.CurrentCampaign == nil ||
		!recovered.CurrentCampaign.Exact || recovered.CurrentCampaign.ID != final.CampaignID ||
		recovered.CampaignHistory[final.CampaignID] != recovered.CurrentCampaign.CommitSHA {
		t.Fatalf("historical campaign recovery final=%#v campaign=%#v err=%v", final, recovered.CurrentCampaign, err)
	}
	for _, run := range recovered.Runs {
		if run.CampaignID != final.CampaignID {
			t.Fatalf("workflow batch retained a split campaign: %#v", recovered.Runs)
		}
	}
	for _, finding := range recovered.Findings {
		if finding.CampaignID != final.CampaignID {
			t.Fatalf("legacy finding retained a split campaign: %#v", recovered.Findings)
		}
	}
	replayed, replayedState, err := recoverRepositoryReviewHistoricalCampaign(
		t.Context(), fixture.store, fixture.workspace, final, recovered, resolved,
	)
	if err != nil || replayed.Version != final.Version || replayedState.Version != recovered.Version {
		t.Fatalf("idempotent campaign recovery automation=%#v state=%#v err=%v", replayed, replayedState, err)
	}
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = fixture.workspace
	if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
		t.Fatal(saveErr)
	}
	handler := NewHandler(configPath)
	t.Cleanup(handler.Shutdown)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	retryPath := "/api/repository-reviews/automations/" + final.ID +
		"/historical-deduplication/retry"
	retry := func() *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, retryPath, strings.NewReader(`{}`))
		setRepositoryReviewMutationHeaders(request)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		return response
	}
	firstRetry := retry()
	if firstRetry.Code != http.StatusConflict || !strings.Contains(
		firstRetry.Body.String(), `"code":"historical_consolidation_restart_required"`,
	) {
		t.Fatalf("recovered retry status=%d body=%s", firstRetry.Code, firstRetry.Body.String())
	}
	restartPath := "/api/repository-reviews/automations/" + final.ID +
		"/historical-deduplication/restart"
	restart := func() *httptest.ResponseRecorder {
		request := httptest.NewRequest(
			http.MethodPost, restartPath, strings.NewReader(`{"confirmed":true}`),
		)
		setRepositoryReviewMutationHeaders(request)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		return response
	}
	firstRestart := restart()
	if firstRestart.Code != http.StatusAccepted || !strings.Contains(
		firstRestart.Body.String(), `"status":"replaying"`,
	) {
		t.Fatalf("recovered restart status=%d body=%s", firstRestart.Code, firstRestart.Body.String())
	}
	afterFirstRestart, _, err := fixture.store.Get(state.Repository)
	if err != nil {
		t.Fatal(err)
	}
	secondRestart := restart()
	afterSecondRestart, _, err := fixture.store.Get(state.Repository)
	if err != nil || secondRestart.Code != http.StatusAccepted ||
		afterSecondRestart.Version != afterFirstRestart.Version {
		t.Fatalf(
			"idempotent API restart status=%d first=%d second=%d err=%v body=%s",
			secondRestart.Code, afterFirstRestart.Version, afterSecondRestart.Version, err,
			secondRestart.Body.String(),
		)
	}
	reset, nextAdmission, err := fixture.store.AdmitNextHistoricalDeduplicationBatch(state.Repository)
	if err != nil || nextAdmission.Admitted != 1 || len(reset.RawFindings) != 2 {
		t.Fatalf("recovered cross-batch admission=%#v raws=%#v err=%v", nextAdmission, reset.RawFindings, err)
	}
	for _, raw := range reset.RawFindings {
		if raw.CampaignID != final.CampaignID {
			t.Fatalf("recovered processing identities remained split: %#v", reset.RawFindings)
		}
	}
}

func TestRepositoryReviewHistoricalCampaignRecoveredRequiresBackfillProof(t *testing.T) {
	started := time.Date(2026, 8, 30, 18, 0, 0, 0, time.UTC)
	commitSHA := strings.Repeat("a", 40)
	campaignID := repoaudit.NewRepositoryReviewCampaignID()
	automation := repoaudit.RepositoryReviewAutomation{
		CampaignID: campaignID, ResolvedCommitSHA: commitSHA,
		RunIDs: []string{"wr_legacy"}, StartedAt: started,
	}
	state := repoaudit.RepositoryState{
		CampaignHistory: map[string]string{campaignID: commitSHA},
		CurrentCampaign: &repoaudit.RepositoryReviewCampaignCoverage{
			ID: campaignID, CommitSHA: commitSHA, Exact: true,
			AssignmentCatalog: []repoaudit.RepositoryReviewAssignment{{ID: "assignment"}},
		},
		Runs: []repoaudit.ReviewRun{{
			ID: "wr_legacy", FindingIDs: []string{"rfn_legacy"},
			CompletedAt: started.Add(time.Minute),
		}},
		Findings: []repoaudit.Finding{{ID: "rfn_legacy"}},
	}
	if repositoryReviewHistoricalCampaignRecovered(automation, state) {
		t.Fatal("native exact campaign bypassed historical recovery without a recovery digest")
	}
	state.CurrentCampaign.RecoveryDigest = "sha256:" + strings.Repeat("b", 64)
	if repositoryReviewHistoricalCampaignRecovered(automation, state) {
		t.Fatal("recovery digest bypassed untagged legacy runs/findings")
	}
	state.Runs[0].CampaignID = campaignID
	state.Findings[0].CampaignID = campaignID
	if !repositoryReviewHistoricalCampaignRecovered(automation, state) {
		t.Fatal("exact backfill proof and tagged legacy records were not recognized")
	}
}

func TestRepositoryReviewDedupAdditionalRouteErrorAndPagingCoverage(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	campaignID := "rrc_additional_route_edges"
	state = seedRepositoryReviewDeduplicationAPIState(t, workspace, state, campaignID)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
	base := "/api/repository-reviews/automations/" + automation.ID

	for _, target := range []string{
		"/api/repository-reviews/automations/rra_missing/findings",
		"/api/repository-reviews/automations/rra_missing/findings/rdf_missing/sources",
		"/api/repository-reviews/automations/rra_missing/findings-processing",
		"/api/repository-reviews/automations/rra_missing/findings-processing/sources/rrw_missing",
		"/api/repository-reviews/automations/rra_missing/historical-deduplication",
		base + "/run-findings/rdf_missing",
	} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusNotFound && response.Code != http.StatusBadRequest {
			t.Fatalf("GET %s status=%d body=%s", target, response.Code, response.Body.String())
		}
	}

	processing := httptest.NewRecorder()
	mux.ServeHTTP(processing, httptest.NewRequest(
		http.MethodGet, base+"/campaigns/"+campaignID+"/findings-processing?limit=1", nil,
	))
	if processing.Code != http.StatusOK ||
		!strings.Contains(processing.Body.String(), `"next_offset":1`) {
		t.Fatalf("processing page status=%d body=%s", processing.Code, processing.Body.String())
	}

	badBody := httptest.NewRequest(
		http.MethodPost,
		base+"/campaigns/"+campaignID+"/findings-processing/sources/"+state.RawFindings[2].ID+"/retry",
		strings.NewReader(`{`),
	)
	setRepositoryReviewMutationHeaders(badBody)
	badBodyResponse := httptest.NewRecorder()
	mux.ServeHTTP(badBodyResponse, badBody)
	if badBodyResponse.Code != http.StatusBadRequest {
		t.Fatalf("raw retry malformed body status=%d body=%s", badBodyResponse.Code, badBodyResponse.Body.String())
	}

	missingSource := httptest.NewRequest(
		http.MethodPost,
		base+"/campaigns/"+campaignID+"/findings-processing/sources/rrw_missing/retry",
		strings.NewReader(`{}`),
	)
	setRepositoryReviewMutationHeaders(missingSource)
	missingSourceResponse := httptest.NewRecorder()
	mux.ServeHTTP(missingSourceResponse, missingSource)
	if missingSourceResponse.Code != http.StatusNotFound {
		t.Fatalf(
			"raw retry missing source status=%d body=%s",
			missingSourceResponse.Code,
			missingSourceResponse.Body.String(),
		)
	}

	wrongFinding := httptest.NewRecorder()
	mux.ServeHTTP(wrongFinding, httptest.NewRequest(
		http.MethodGet,
		base+"/findings/rdf_wrong/sources/"+state.RawFindings[0].ID,
		nil,
	))
	if wrongFinding.Code != http.StatusNotFound {
		t.Fatalf("raw source wrong finding status=%d body=%s", wrongFinding.Code, wrongFinding.Body.String())
	}

	if _, found := repositoryReviewContextByID(state, "missing-context"); found {
		t.Fatal("missing context was found")
	}
	summary := projectRepositoryReviewDeduplicatedFindingSummary(
		repoaudit.DeduplicatedReviewFinding{RawSourceIDs: []string{"missing-raw"}},
		newRepositoryReviewRunFindingStatusIndex(state),
		map[string]repoaudit.RawReviewFinding{},
	)
	if len(summary.Contributors) != 0 {
		t.Fatalf("missing-raw contributors=%#v", summary.Contributors)
	}
}

func TestRepositoryReviewDedupAdditionalLegacyAndSourcePagingCoverage(t *testing.T) {
	t.Run("legacy fallback", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		state := seedRepositoryReviewAPIState(t, workspace)
		state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
		automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
		state.RawFindings = nil
		state.DeduplicationJobs = nil
		state.DeduplicatedFindings = nil
		state.FindingsProcessing = repoaudit.FindingsProcessingCounters{}
		state.NextDeduplicationOrdinal = 0
		persistRepositoryReviewAdditionalCoverageState(t, workspace, state)
		base := "/api/repository-reviews/automations/" + automation.ID
		for _, target := range []struct {
			path string
			want int
		}{
			{base + "/findings", http.StatusOK},
			{base + "/findings/" + state.Findings[0].ID, http.StatusNotFound},
			{base + "/findings?scope=all&offset=2&limit=1", http.StatusOK},
		} {
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target.path, nil))
			if response.Code != target.want {
				t.Fatalf("strict legacy %s status=%d body=%s", target.path, response.Code, response.Body.String())
			}
		}
	})

	t.Run("source page and projection fallback", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		state := seedRepositoryReviewAPIState(t, workspace)
		campaignID := "rrc_source_page_coverage"
		state = seedRepositoryReviewDeduplicationAPIState(t, workspace, state, campaignID)
		automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
		state.DeduplicatedFindings[0].RawSourceIDs = append(
			state.DeduplicatedFindings[0].RawSourceIDs, state.RawFindings[1].ID,
		)
		state.RawFindings[1].CreatedAt = state.RawFindings[0].CreatedAt
		state.RawFindings[2].CampaignID = "rrc_other_campaign"
		state.RawFindings[2].DiagnosisDigest = repoaudit.RawReviewFindingDiagnosisDigest(state.RawFindings[2])
		persistRepositoryReviewAdditionalCoverageState(t, workspace, state)
		base := "/api/repository-reviews/automations/" + automation.ID
		forwarded := httptest.NewRecorder()
		mux.ServeHTTP(forwarded, httptest.NewRequest(
			http.MethodGet, base+"/run-findings/"+state.DeduplicatedFindings[0].ID, nil,
		))
		if forwarded.Code != http.StatusNotFound {
			t.Fatalf("strict legacy occurrence detail status=%d body=%s", forwarded.Code, forwarded.Body.String())
		}
		sources := httptest.NewRecorder()
		mux.ServeHTTP(sources, httptest.NewRequest(
			http.MethodGet,
			base+"/findings/"+state.DeduplicatedFindings[0].ID+"/sources?limit=1",
			nil,
		))
		if sources.Code != http.StatusOK || !strings.Contains(sources.Body.String(), `"next_offset":1`) {
			t.Fatalf("raw source page status=%d body=%s", sources.Code, sources.Body.String())
		}
		processing := httptest.NewRecorder()
		mux.ServeHTTP(processing, httptest.NewRequest(
			http.MethodGet,
			base+"/campaigns/"+campaignID+"/findings-processing?limit=2",
			nil,
		))
		if processing.Code != http.StatusOK {
			t.Fatalf("equal-time processing status=%d body=%s", processing.Code, processing.Body.String())
		}

		state.Findings = nil
		state.MappingJobs = nil
		state.RepositoryFindings = nil
		for index := range state.Runs {
			state.Runs[index].FindingIDs = nil
		}
		persistRepositoryReviewAdditionalCoverageState(t, workspace, state)
		detail := httptest.NewRecorder()
		mux.ServeHTTP(detail, httptest.NewRequest(
			http.MethodGet,
			base+"/findings/"+state.DeduplicatedFindings[0].ID+"/sources/"+state.RawFindings[0].ID,
			nil,
		))
		if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), `"finding"`) {
			t.Fatalf("projection-free raw detail status=%d body=%s", detail.Code, detail.Body.String())
		}
	})
}

func TestRepositoryReviewHistoricalDedupAdditionalErrorCoverage(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
	base := "/api/repository-reviews/automations/" + automation.ID + "/historical-deduplication"
	now := time.Now().UTC()
	state.RawFindings = nil
	state.DeduplicationJobs = nil
	state.DeduplicatedFindings = nil
	state.FindingsProcessing = repoaudit.FindingsProcessingCounters{}
	for index := range state.Findings {
		state.Findings[index].DeduplicationPending = false
		state.Findings[index].CommitSHA = strings.Repeat("c", 40)
	}
	state.HistoricalDeduplication = repoaudit.HistoricalDeduplicationReplay{
		Required: true, Status: repoaudit.HistoricalDeduplicationPending, UpdatedAt: now,
	}
	persistRepositoryReviewAdditionalCoverageState(t, workspace, state)

	for _, test := range []struct {
		target string
		body   string
	}{
		{target: base + "/retry", body: `{`},
		{target: "/api/repository-reviews/automations/rra_missing/historical-deduplication/retry", body: `{}`},
		{target: base + "/retry", body: `{}`},
	} {
		request := httptest.NewRequest(http.MethodPost, test.target, strings.NewReader(test.body))
		setRepositoryReviewMutationHeaders(request)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest && response.Code != http.StatusNotFound &&
			response.Code != http.StatusConflict {
			t.Fatalf("historical retry %s status=%d body=%s", test.target, response.Code, response.Body.String())
		}
	}

	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	controller := newRepositoryReviewController(handler)
	controller.startOnce.Do(func() {})
	controller.leasedStore = store
	controller.leasedConfig = cfg
	t.Cleanup(controller.Stop)
	corruptProfileID := "rrpf_historical_corrupt"
	corruptProfilePath := filepath.Join(workspace, "repository_reviews", "profile_"+corruptProfileID+".json")
	if writeErr := os.WriteFile(corruptProfilePath, []byte(`{`), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	corruptProfileAutomation := testRepositoryReviewAutomation()
	corruptProfileAutomation.ID = "rra_historical_corrupt_profile"
	corruptProfileAutomation.Repository = state.Repository
	corruptProfileAutomation.RunIDs = []string{state.Runs[0].ID}
	corruptProfileAutomation.ProfileID = corruptProfileID
	if advanceErr := controller.advanceHistoricalFindingDeduplication(
		t.Context(), state, []repoaudit.RepositoryReviewAutomation{corruptProfileAutomation},
	); advanceErr == nil {
		t.Fatal("historical replay loaded a corrupt profile")
	}
	if removeErr := os.Remove(corruptProfilePath); removeErr != nil {
		t.Fatal(removeErr)
	}

	badSnapshotController := newRepositoryReviewController(NewHandler(t.TempDir()))
	badSnapshotController.startOnce.Do(func() {})
	badSnapshotController.leasedStore = store
	badSnapshotController.leasedConfig = cfg
	if advanceErr := badSnapshotController.advanceHistoricalFindingDeduplication(
		t.Context(), state, nil,
	); advanceErr == nil {
		t.Fatal("historical replay captured a directory-backed configuration revision")
	}

	if _, _, failErr := store.FailHistoricalDeduplicationReplay(state.Repository, ""); failErr != nil {
		t.Fatal(failErr)
	}
	if advanceErr := controller.advanceHistoricalFindingDeduplication(
		t.Context(), state, nil,
	); advanceErr == nil {
		t.Fatal("historical replay froze over a failed durable replay")
	}
	if _, _, retryErr := store.RetryHistoricalDeduplicationReplay(state.Repository); retryErr != nil {
		t.Fatal(retryErr)
	}
	replayingInput := state
	replayingInput.HistoricalDeduplication.Status = repoaudit.HistoricalDeduplicationReplaying
	if advanceErr := controller.advanceHistoricalFindingDeduplication(
		t.Context(), replayingInput, nil,
	); advanceErr == nil {
		t.Fatal("historical replay admitted against a pending durable replay")
	}
	if advanceErr := controller.advanceHistoricalFindingDeduplication(
		t.Context(), state, nil,
	); advanceErr != nil {
		t.Fatalf("historical fallback automation advance err=%v", advanceErr)
	}
	if _, _, failErr := store.FailHistoricalDeduplicationReplay(state.Repository, ""); failErr != nil {
		t.Fatal(failErr)
	}
	if _, _, retryErr := store.RetryHistoricalDeduplicationReplay(state.Repository); retryErr != nil {
		t.Fatal(retryErr)
	}

	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if processErr := controller.processHistoricalFindingDeduplications(canceled, nil); !errors.Is(
		processErr, context.Canceled,
	) {
		t.Fatalf("canceled historical processing err=%v", processErr)
	}

	missingProfile := state
	missingProfile.HistoricalDeduplication.Status = repoaudit.HistoricalDeduplicationPending
	missingProfile.HistoricalDeduplication.ProfileSnapshot = repoaudit.HistoricalDeduplicationProfileSnapshot{}
	missingProfile.HistoricalDeduplication.UpdatedAt = time.Now().UTC()
	missingProfile.Version++
	missingProfile.UpdatedAt = missingProfile.HistoricalDeduplication.UpdatedAt
	persistRepositoryReviewAdditionalCoverageState(t, workspace, missingProfile)
	profileAutomation := testRepositoryReviewAutomation()
	profileAutomation.ID = "rra_historical_process_error"
	profileAutomation.Repository = state.Repository
	profileAutomation.RunIDs = []string{state.Runs[0].ID}
	profileAutomation.ProfileID = "rrpf_missing"
	if processErr := controller.processHistoricalFindingDeduplications(
		t.Context(), []repoaudit.RepositoryReviewAutomation{profileAutomation},
	); processErr == nil || !strings.Contains(processErr.Error(), "profile was not found") {
		t.Fatalf("historical fatal processing err=%v", processErr)
	}

	// Install a real durable merge with a different lease so the stale input
	// exercises the lease fence rather than the failed-merge resume path.
	persistRepositoryReviewAdditionalCoverageState(t, workspace, missingProfile)
	mergeSnapshot := repoaudit.RepositoryReviewDeduplicationSnapshot{
		ReviewerModel: "cheap", DeduplicationModel: "cheap", AccountRef: "api",
		SimilarityThreshold: 90, CandidateLimit: 0,
	}
	if _, _, freezeErr := store.FreezeHistoricalDeduplicationReplay(
		state.Repository, mergeSnapshot,
	); freezeErr != nil {
		t.Fatal(freezeErr)
	}
	merging, _, _, acquireErr := store.AcquireHistoricalDeduplicationMerge(
		state.Repository, "rhl_durable", nil,
	)
	if acquireErr != nil {
		t.Fatal(acquireErr)
	}
	merging.HistoricalDeduplication.MergeLease.ID = "rhl_missing"
	if advanceErr := controller.advanceHistoricalFindingDeduplication(
		t.Context(), merging, nil,
	); advanceErr == nil {
		t.Fatal("mismatched historical merge unexpectedly completed")
	}
	persistRepositoryReviewAdditionalCoverageState(t, workspace, missingProfile)

	invalidRefProfile := repoaudit.RepositoryReviewProfile{
		ID: "rrpf_historical_invalid_ref", Name: "Historical invalid ref",
		ReviewFocus: "Find bugs.", ReviewerModel: "cheap", AccountRef: "api",
		DeduplicationSimilarityThreshold: 90, DeduplicationCandidateLimit: 4,
		ScopePolicy: repoaudit.RepositoryReviewScopePolicy{CodeTypes: []repoaudit.RepositoryReviewCodeType{
			repoaudit.RepositoryReviewCodeTypeCode,
		}},
		AutoContinue: true, MaxFilesPerRun: 1, MaxContentBytes: 1024,
		MaxParallelChildren: 1, AssignmentTimeoutSeconds: 60,
		BudgetPolicy: repoaudit.RepositoryReviewBudgetPolicy{},
	}
	createdProfile, err := store.CreateProfile(t.Context(), invalidRefProfile)
	if err != nil {
		t.Fatal(err)
	}
	invalidRefAutomation := testRepositoryReviewAutomation()
	invalidRefAutomation.ID = "rra_historical_invalid_ref"
	invalidRefAutomation.Repository = state.Repository
	invalidRefAutomation.RunIDs = []string{state.Runs[0].ID}
	invalidRefAutomation.ProfileID = createdProfile.ID
	invalidRefAutomation.Ref = "bad ref"
	if advanceErr := controller.advanceHistoricalFindingDeduplication(
		t.Context(), state, []repoaudit.RepositoryReviewAutomation{invalidRefAutomation},
	); advanceErr == nil {
		t.Fatal("historical replay materialized an invalid ref")
	}
	if _, _, retryErr := store.RetryHistoricalDeduplicationReplay(state.Repository); retryErr != nil {
		t.Fatal(retryErr)
	}
	_, _, freezeErr := store.FreezeHistoricalDeduplicationReplay(
		state.Repository, mergeSnapshot,
	)
	if freezeErr != nil {
		t.Fatal(freezeErr)
	}
	mergeState, replay, _, acquireErr := store.AcquireHistoricalDeduplicationMerge(
		state.Repository, "rhl_additional_success", nil,
	)
	if acquireErr != nil || replay.Status != repoaudit.HistoricalDeduplicationMerging {
		t.Fatalf("historical merge acquire replay=%#v err=%v", replay, acquireErr)
	}
	if advanceErr := controller.advanceHistoricalFindingDeduplication(
		t.Context(), mergeState, nil,
	); advanceErr != nil {
		t.Fatalf("historical merging advance err=%v", advanceErr)
	}

	controller.startHistoricalFindingDeduplication(nil)
	controller.wg.Wait()
	controller.wakeHistoricalFindingDeduplication()
	controller.wg.Wait()
}

func TestRepositoryReviewDedupAdditionalControllerAndModelCoverage(t *testing.T) {
	if err := (*repositoryReviewController)(nil).processRepositoryFindingDeduplications(t.Context()); err == nil {
		t.Fatal("nil deduplication controller succeeded")
	}
	if repositoryStateHasPendingDeduplication(repoaudit.RepositoryState{}) {
		t.Fatal("empty repository reported pending deduplication")
	}
	if !repositoryStateHasPendingDeduplication(repoaudit.RepositoryState{
		DeduplicationJobs: []repoaudit.DeduplicationJob{{State: repoaudit.DeduplicationJobPending}},
	}) {
		t.Fatal("pending repository was not detected")
	}

	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	state = seedRepositoryReviewDeduplicationAPIState(t, workspace, state, "rrc_controller_coverage")
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Agents.Defaults.ContextWindow = 2
	controller := newRepositoryReviewController(handler)
	controller.startOnce.Do(func() {})
	controller.leasedStore = store
	controller.leasedConfig = cfg
	t.Cleanup(controller.Stop)
	originalProcessor := processRepositoryDeduplicationJobs
	t.Cleanup(func() { processRepositoryDeduplicationJobs = originalProcessor })
	if _, processErr := originalProcessor(
		store, t.Context(), "missing/repository", repoaudit.DeduplicationProcessOptions{},
	); processErr == nil {
		t.Fatal("default deduplication processor accepted a missing repository")
	}

	processorCalls := 0
	processRepositoryDeduplicationJobs = func(
		_ repoaudit.Store,
		_ context.Context,
		repository string,
		options repoaudit.DeduplicationProcessOptions,
	) (repoaudit.DeduplicationProcessResult, error) {
		processorCalls++
		if repository != state.Repository || options.ModelInputCeiling != 1 || options.LeaseDuration != time.Hour {
			t.Fatalf("deduplication options repository=%q options=%#v", repository, options)
		}
		return repoaudit.DeduplicationProcessResult{}, errors.New("injected processor failure")
	}
	controller.wakeRepositoryFindingDeduplication()
	controller.wg.Wait()
	processorCalls = 0
	if processErr := controller.processRepositoryFindingDeduplications(t.Context()); processErr == nil ||
		!strings.Contains(processErr.Error(), "injected processor failure") || processorCalls != 1 {
		t.Fatalf("deduplication processing calls=%d err=%v", processorCalls, processErr)
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if processErr := controller.processRepositoryFindingDeduplications(canceled); !errors.Is(
		processErr,
		context.Canceled,
	) {
		t.Fatalf("canceled deduplication processing err=%v", processErr)
	}

	blockedRoot := filepath.Join(t.TempDir(), "blocked")
	if writeErr := os.WriteFile(blockedRoot, nil, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	blocked := newRepositoryReviewController(handler)
	blocked.leasedConfig = cfg
	blocked.leasedStore = repoaudit.NewStore(blockedRoot)
	if processErr := blocked.processRepositoryFindingDeduplications(t.Context()); processErr == nil {
		t.Fatal("deduplication processing accepted an unreadable store")
	}

	snapshot := repoaudit.RepositoryReviewDeduplicationSnapshot{
		ReviewerModel: "cheap", DeduplicationModel: "cheap", AccountRef: "api",
		SimilarityThreshold: 90, CandidateLimit: 4,
	}
	modelHandler := newRepositoryReviewAIAdjudicationHandler(t, http.StatusOK, `{}`)
	if _, modelErr := runRepositoryReviewDeduplicationModel(
		t.Context(), nil, snapshot, "score", map[string]any{}, "system", map[string]any{},
	); modelErr == nil {
		t.Fatal("nil deduplication handler succeeded")
	}
	badConfigHandler := NewHandler(t.TempDir())
	if _, modelErr := runRepositoryReviewDeduplicationModel(
		t.Context(), badConfigHandler, snapshot, "score", map[string]any{}, "system", map[string]any{},
	); modelErr == nil {
		t.Fatal("directory-backed deduplication configuration succeeded")
	}
	missingModel := snapshot
	missingModel.DeduplicationModel = "missing-model"
	if _, modelErr := runRepositoryReviewDeduplicationModel(
		t.Context(), modelHandler, missingModel, "score", map[string]any{}, "system", map[string]any{},
	); modelErr == nil {
		t.Fatal("missing deduplication model alias resolved")
	}
	if _, modelErr := runRepositoryReviewDeduplicationModel(
		t.Context(), modelHandler, snapshot, "score",
		strings.Repeat("x", repoaudit.DeduplicationMaximumInputBytes+1), "system", map[string]any{},
	); modelErr == nil || !strings.Contains(modelErr.Error(), "exceeds") {
		t.Fatalf("oversized deduplication input err=%v", modelErr)
	}

	originalAgent := runRepositoryDeduplicationAgent
	t.Cleanup(func() { runRepositoryDeduplicationAgent = originalAgent })
	runRepositoryDeduplicationAgent = func(
		context.Context, *webWorkflowRuntimeRunner, workflows.AgentRequest,
	) (map[string]any, error) {
		return nil, errors.New("injected agent failure")
	}
	if _, modelErr := runRepositoryReviewDeduplicationModel(
		t.Context(), modelHandler, snapshot, "score", map[string]any{}, "system", map[string]any{},
	); modelErr == nil || !strings.Contains(modelErr.Error(), "agent failure") {
		t.Fatalf("deduplication agent failure err=%v", modelErr)
	}
	if _, judgeErr := runRepositoryReviewDeduplicationJudgment(
		t.Context(), modelHandler, snapshot, "judge", repoaudit.DeduplicationJudgeRequest{},
	); judgeErr == nil || !strings.Contains(judgeErr.Error(), "agent failure") {
		t.Fatalf("deduplication judge agent failure err=%v", judgeErr)
	}
	runRepositoryDeduplicationAgent = func(
		context.Context, *webWorkflowRuntimeRunner, workflows.AgentRequest,
	) (map[string]any, error) {
		return map[string]any{"structured_valid": false}, nil
	}
	if _, modelErr := runRepositoryReviewDeduplicationModel(
		t.Context(), modelHandler, snapshot, "score", map[string]any{}, "system", map[string]any{},
	); modelErr == nil || !strings.Contains(modelErr.Error(), "invalid structured") {
		t.Fatalf("invalid structured deduplication output err=%v", modelErr)
	}
	runRepositoryDeduplicationAgent = func(
		context.Context, *webWorkflowRuntimeRunner, workflows.AgentRequest,
	) (map[string]any, error) {
		return map[string]any{"structured_valid": true, "structured": func() {}}, nil
	}
	if _, scoreErr := runRepositoryReviewDeduplicationScoring(
		t.Context(), modelHandler, snapshot, "score", repoaudit.DeduplicationScoringRequest{},
	); scoreErr == nil {
		t.Fatal("unencodable structured score succeeded")
	}
	if _, judgeErr := runRepositoryReviewDeduplicationJudgment(
		t.Context(), modelHandler, snapshot, "judge", repoaudit.DeduplicationJudgeRequest{},
	); judgeErr == nil {
		t.Fatal("unencodable structured judgment succeeded")
	}
}

func TestRepositoryReviewHistoricalDedupAdditionalControllerCoverage(t *testing.T) {
	if err := (*repositoryReviewController)(nil).processHistoricalFindingDeduplications(
		t.Context(), nil,
	); err == nil {
		t.Fatal("nil historical controller succeeded")
	}
	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	controller := newRepositoryReviewController(handler)
	controller.startOnce.Do(func() {})
	controller.leasedStore = store
	controller.leasedConfig = cfg
	t.Cleanup(controller.Stop)
	legacy := state
	legacy.RawFindings = nil
	legacy.DeduplicationJobs = nil
	legacy.DeduplicatedFindings = nil
	legacy.FindingsProcessing = repoaudit.FindingsProcessingCounters{}
	for index := range legacy.Findings {
		legacy.Findings[index].DeduplicationPending = false
		legacy.Findings[index].CommitSHA = strings.Repeat("b", 40)
	}
	legacy.HistoricalDeduplication = repoaudit.HistoricalDeduplicationReplay{
		Required: true, Status: repoaudit.HistoricalDeduplicationPending,
		UpdatedAt: time.Now().UTC(),
	}
	persistRepositoryReviewAdditionalCoverageState(t, workspace, legacy)
	replayAutomation := testRepositoryReviewAutomation()
	replayAutomation.ID = "rra_historical_replay_coverage"
	replayAutomation.Repository = legacy.Repository
	replayAutomation.RunIDs = []string{legacy.Runs[0].ID}
	if advanceErr := controller.advanceHistoricalFindingDeduplication(
		t.Context(), legacy, []repoaudit.RepositoryReviewAutomation{replayAutomation},
	); advanceErr != nil {
		t.Fatalf("pending replay advance err=%v", advanceErr)
	}
	failed := state
	failed.HistoricalDeduplication = repoaudit.HistoricalDeduplicationReplay{
		Required: true, Status: repoaudit.HistoricalDeduplicationFailed,
	}
	if advanceErr := controller.advanceHistoricalFindingDeduplication(t.Context(), failed, nil); advanceErr != nil {
		t.Fatalf("failed replay advance err=%v", advanceErr)
	}
	completed := state
	completed.HistoricalDeduplication = repoaudit.HistoricalDeduplicationReplay{
		Status: repoaudit.HistoricalDeduplicationCompleted,
	}
	if advanceErr := controller.advanceHistoricalFindingDeduplication(t.Context(), completed, nil); advanceErr != nil {
		t.Fatalf("completed replay advance err=%v", advanceErr)
	}
	pending := state
	pending.HistoricalDeduplication = repoaudit.HistoricalDeduplicationReplay{
		Required: true, Status: repoaudit.HistoricalDeduplicationPending,
	}
	automation := testRepositoryReviewAutomation()
	automation.ID = "rra_historical_missing_profile"
	automation.Repository = state.Repository
	automation.ProfileID = "rrpf_missing"
	if advanceErr := controller.advanceHistoricalFindingDeduplication(
		t.Context(), pending, []repoaudit.RepositoryReviewAutomation{automation},
	); advanceErr == nil || !strings.Contains(advanceErr.Error(), "profile was not found") {
		t.Fatalf("missing historical profile err=%v", advanceErr)
	}

	blockedRoot := filepath.Join(t.TempDir(), "blocked")
	if writeErr := os.WriteFile(blockedRoot, nil, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	blocked := newRepositoryReviewController(handler)
	blocked.leasedConfig = cfg
	blocked.leasedStore = repoaudit.NewStore(blockedRoot)
	if processErr := blocked.processHistoricalFindingDeduplications(t.Context(), nil); processErr == nil {
		t.Fatal("historical processing accepted an unreadable store")
	}
}

func TestRepositoryReviewHistoricalDedupAdditionalCompletionCoverage(t *testing.T) {
	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	campaignID := "rrc_historical_completion"
	for index := range state.Findings {
		state.Findings[index].DeduplicationPending = false
		state.Findings[index].CommitSHA = strings.Repeat("d", 40)
		state.Findings[index].CampaignID = campaignID
	}
	for index := range state.Contexts {
		state.Contexts[index].CampaignID = campaignID
		state.Contexts[index].CommitSHA = strings.Repeat("d", 40)
	}
	state.Runs = nil
	state.CampaignHistory = map[string]string{campaignID: strings.Repeat("d", 40)}
	state.RawFindings = nil
	state.DeduplicationJobs = nil
	state.DeduplicatedFindings = nil
	state.MappingJobs = nil
	state.FindingsProcessing = repoaudit.FindingsProcessingCounters{}
	state.HistoricalDeduplication = repoaudit.HistoricalDeduplicationReplay{
		Required: true, Status: repoaudit.HistoricalDeduplicationPending,
		UpdatedAt: time.Now().UTC(),
	}
	persistRepositoryReviewAdditionalCoverageState(t, workspace, state)
	store := repoaudit.NewStore(workspace)
	snapshot := repoaudit.RepositoryReviewDeduplicationSnapshot{
		ReviewerModel: "cheap", DeduplicationModel: "cheap", AccountRef: "api",
		SimilarityThreshold: 90, CandidateLimit: 0,
	}
	state, _, err := store.FreezeHistoricalDeduplicationReplay(state.Repository, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	state, admission, err := store.AdmitNextHistoricalDeduplicationBatch(state.Repository)
	if err != nil || admission.Admitted == 0 {
		t.Fatalf("historical completion admission=%#v err=%v", admission, err)
	}
	if _, processErr := store.ProcessPendingDeduplicationJobs(
		t.Context(), state.Repository, repoaudit.DeduplicationProcessOptions{},
	); processErr != nil {
		t.Fatal(processErr)
	}
	state, _, err = store.Get(state.Repository)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	controller := newRepositoryReviewController(handler)
	controller.startOnce.Do(func() {})
	controller.leasedStore = store
	controller.leasedConfig = cfg
	t.Cleanup(controller.Stop)
	if advanceErr := controller.advanceHistoricalFindingDeduplication(
		t.Context(), state, nil,
	); advanceErr != nil {
		t.Fatalf("historical completion advance err=%v", advanceErr)
	}
	completed, found, err := store.Get(state.Repository)
	if err != nil || !found || completed.HistoricalDeduplication.Required ||
		completed.HistoricalDeduplication.Status != repoaudit.HistoricalDeduplicationCompleted {
		t.Fatalf("historical completion state=%#v found=%v err=%v", completed.HistoricalDeduplication, found, err)
	}
}

func TestRepositoryReviewDedupAdditionalProfileCollectionAndControllerCoverage(t *testing.T) {
	profile := repoaudit.RepositoryReviewProfile{
		ReviewerModel: "cheap", DeduplicationSimilarityThreshold: 91,
		DeduplicationCandidateLimit: 7,
	}
	for _, field := range []collectionquery.Field{
		"deduplicator", "deduplication_threshold", "deduplication_candidates",
	} {
		if _, ok := repositoryReviewProfileCollectionField(profile, field); !ok {
			t.Fatalf("profile field %q was unresolved", field)
		}
	}

	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	invalidCreate := repositoryReviewProfileCreateBody("Invalid deduplication", "cheap")
	invalidCreate["deduplication_similarity_threshold"] = 101
	response := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost, "/api/repository-reviews/profiles", invalidCreate,
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid deduplication create status=%d body=%s", response.Code, response.Body.String())
	}
	created := createRepositoryReviewProfileForTest(t, mux, "Deduplication coverage", "cheap")
	invalidUpdate := repositoryReviewProfileBody(created)
	invalidUpdate["expected_version"] = created.Version
	invalidUpdate["deduplication_candidate_limit"] = 21
	response = repositoryReviewAutomationMutation(
		t, mux, http.MethodPatch, "/api/repository-reviews/profiles/"+created.ID, invalidUpdate,
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid deduplication update status=%d body=%s", response.Code, response.Body.String())
	}

	if validationErr := handler.validateRepositoryReviewProfileSelectionWithModels(
		"api", "cheap", "", "missing-model", repoaudit.RepositoryReviewBudgetPolicy{},
	); validationErr == nil {
		t.Fatal("missing explicit deduplication model validated")
	}

	for _, target := range []string{
		"/api/repository-reviews/automations/rra_missing/run-findings",
		"/api/repository-reviews/automations/rra_missing/run-findings/rfn_missing",
	} {
		page := httptest.NewRecorder()
		mux.ServeHTTP(page, httptest.NewRequest(http.MethodGet, target, nil))
		if page.Code != http.StatusNotFound && page.Code != http.StatusBadRequest {
			t.Fatalf("collection error route %s status=%d body=%s", target, page.Code, page.Body.String())
		}
	}
	badQuery := httptest.NewRecorder()
	mux.ServeHTTP(badQuery, httptest.NewRequest(
		http.MethodGet, "/api/repository-reviews/automations/rra_missing/run-findings?query=(", nil,
	))
	if badQuery.Code != http.StatusBadRequest {
		t.Fatalf("run finding invalid query status=%d body=%s", badQuery.Code, badQuery.Body.String())
	}

	controller := newRepositoryReviewController(handler)
	if _, snapshotErr := controller.repositoryReviewDeduplicationSnapshot(
		repoaudit.RepositoryReviewAutomation{ReviewerModels: []string{"cheap"}},
	); snapshotErr != nil {
		t.Fatalf("deduplication snapshot err=%v", snapshotErr)
	}
	badSnapshotController := newRepositoryReviewController(NewHandler(t.TempDir()))
	if _, snapshotErr := badSnapshotController.repositoryReviewDeduplicationSnapshot(
		repoaudit.RepositoryReviewAutomation{ReviewerModels: []string{"cheap"}},
	); snapshotErr == nil {
		t.Fatal("directory-backed snapshot revision succeeded")
	}
	if _, profileErr := (*repositoryReviewController)(nil).resolveRepositoryReviewCampaignProfile(
		t.Context(), nil, repoaudit.RepositoryReviewAutomation{},
	); profileErr == nil {
		t.Fatal("nil campaign profile controller succeeded")
	}

	state := seedRepositoryReviewAPIState(t, workspace)
	reviewStore := repoaudit.NewStore(workspace)
	invalidSnapshotAutomation := testRepositoryReviewAutomation()
	invalidSnapshotAutomation.Repository = state.Repository
	invalidSnapshotAutomation.ReviewerModels = nil
	if _, campaignErr := controller.ensureRepositoryReviewCampaign(
		t.Context(), reviewStore, config.DefaultConfig(), invalidSnapshotAutomation,
		strings.Repeat("e", 40), "start",
	); campaignErr == nil {
		t.Fatal("campaign accepted an invalid deduplication snapshot")
	}
	canceledCampaign, cancelCampaign := context.WithCancel(t.Context())
	cancelCampaign()
	validCampaignAutomation := testRepositoryReviewAutomation()
	validCampaignAutomation.Repository = state.Repository
	if _, campaignErr := controller.ensureRepositoryReviewCampaign(
		canceledCampaign, reviewStore, config.DefaultConfig(), validCampaignAutomation,
		strings.Repeat("f", 40), "start",
	); !errors.Is(campaignErr, context.Canceled) {
		t.Fatalf("canceled campaign begin err=%v", campaignErr)
	}
	outcomeAutomation := repoaudit.RepositoryReviewAutomation{
		Repository: state.Repository, RunIDs: []string{state.Runs[0].ID},
	}
	findingWithIssue := state.Findings[0]
	findingWithIssue.IssueDraftID = "rid_additional_detail"
	issueState := state
	issueState.IssueDrafts = []repoaudit.IssueDraft{{
		ID: findingWithIssue.IssueDraftID, FindingIDs: []string{findingWithIssue.ID},
		State: repoaudit.IssueDraftEditing,
	}}
	detail := repositoryReviewFindingDetail(
		repositoryReviewAutomationLedger{Automation: outcomeAutomation, State: issueState},
		findingWithIssue,
	)
	if detail["issue"] == nil {
		t.Fatal("direct finding issue was not projected")
	}
	if outcome := loadRepositoryReviewOutcome(repoaudit.NewStore(workspace), outcomeAutomation); !outcome.found {
		t.Fatalf("legacy review outcome=%#v", outcome)
	}

	automationError := httptest.NewRecorder()
	writeRepositoryReviewAutomationError(automationError, repoaudit.ErrHistoricalDeduplicationInProgress)
	if automationError.Code != http.StatusConflict ||
		!strings.Contains(automationError.Body.String(), "historical_deduplication_in_progress") {
		t.Fatalf("historical automation error status=%d body=%s", automationError.Code, automationError.Body.String())
	}
}

func persistRepositoryReviewAdditionalCoverageState(
	t *testing.T,
	workspace string,
	state repoaudit.RepositoryState,
) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(workspace, "repository_reviews", "repo_*.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if strings.HasSuffix(path, ".summary.json") {
			continue
		}
		encoded, encodeErr := json.Marshal(state)
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		if writeErr := os.WriteFile(path, encoded, 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
		return
	}
	t.Fatal("repository review state path is missing")
}
