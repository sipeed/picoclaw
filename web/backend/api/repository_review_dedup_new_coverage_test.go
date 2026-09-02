package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/collectionquery"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/repoaudit"
	"github.com/sipeed/picoclaw/pkg/workflows"
)

func TestRepositoryReviewRawCollectionNewHelperCoverage(t *testing.T) {
	now := time.Date(2026, 8, 30, 19, 0, 0, 0, time.UTC)
	summary := repositoryReviewRawFindingSummary{
		ID: "rrw_fields", Path: "pkg/cache.go", Severity: "high", Title: "stale cache",
		Symbol: "Cache.Load", Model: "model", Reviewer: "reviewer",
		DeduplicationState:    repoaudit.RawFindingDeduplicationCompleted,
		Disposition:           repoaudit.RawFindingDispositionDuplicate,
		DeduplicatedFindingID: "rdf_parent", CreatedAt: now.Add(-time.Hour), UpdatedAt: now,
	}
	contextID := repositoryReviewCollectionCursorContext("raw-findings", "rra_fields", "current")
	options := repositoryReviewRawFindingPageOptions(contextID)
	id, err := options.ID(summary)
	if err != nil || !options.ValidateID(id) {
		t.Fatalf("raw cursor id=%q err=%v", id, err)
	}
	for _, field := range repositoryReviewRawFindingCollectionSchema.Fields {
		if _, ok := options.Resolve(summary, field.Name, now); !ok {
			t.Fatalf("raw field %q was unresolved", field.Name)
		}
	}
	if _, ok := options.Resolve(summary, collectionquery.Field("unknown"), now); ok {
		t.Fatal("unknown raw field resolved")
	}

	started := now.Add(-time.Hour)
	if got := repositoryReviewCurrentCampaignCursorKey(repoaudit.RepositoryReviewAutomation{
		StartedAt: started, CampaignID: "rrc_ignored",
	}); got != started.Format(time.RFC3339Nano) {
		t.Fatalf("started cursor key=%q", got)
	}
	if got := repositoryReviewCurrentCampaignCursorKey(repoaudit.RepositoryReviewAutomation{
		CampaignID: "rrc_current",
	}); got != "rrc_current" {
		t.Fatalf("campaign cursor key=%q", got)
	}
	if got := repositoryReviewCurrentCampaignCursorKey(repoaudit.RepositoryReviewAutomation{}); got != "current" {
		t.Fatalf("default cursor key=%q", got)
	}

	alias := "rfn_legacy"
	findings := []repoaudit.RawReviewFinding{
		{ID: "rrw_z", LegacyFindingID: alias, InsertionOrdinal: 2, CreatedAt: now},
		{ID: "rrw_y", LegacyFindingID: alias, InsertionOrdinal: 1, CreatedAt: now},
		{ID: "rrw_x", DeduplicatedFindingID: alias, InsertionOrdinal: 1, CreatedAt: now.Add(-time.Minute)},
		{ID: "rrw_a", LegacyFindingID: alias, InsertionOrdinal: 1, CreatedAt: now.Add(-time.Minute)},
	}
	selected, found := repositoryReviewRawFindingByAlias(findings, alias)
	if !found || selected.ID != "rrw_a" {
		t.Fatalf("ordered alias=%#v found=%v", selected, found)
	}
	if exact, found := repositoryReviewRawFindingByAlias(findings, "rrw_z"); !found || exact.ID != "rrw_z" {
		t.Fatalf("exact raw alias=%#v found=%v", exact, found)
	}
	if _, found := repositoryReviewRawFindingByAlias(findings, "rrw_missing"); found {
		t.Fatal("nonlegacy missing raw alias resolved")
	}
	if !repositoryReviewRawFindingBefore(findings[1], findings[0]) ||
		!repositoryReviewRawFindingBefore(findings[2], findings[1]) ||
		!repositoryReviewRawFindingBefore(findings[3], findings[2]) {
		t.Fatal("raw ordering branches were inconsistent")
	}
	state := repoaudit.RepositoryState{RawFindings: findings}
	if _, found := repositoryReviewRawFindingByID(state, "missing"); found {
		t.Fatal("missing raw ID resolved")
	}
}

func TestRepositoryReviewRawCollectionNewRouteErrorCoverage(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	state = seedRepositoryReviewDeduplicationAPIState(t, workspace, state, "rrc_new_route_coverage")
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
	base := "/api/repository-reviews/automations/" + automation.ID

	for _, test := range []struct {
		path string
		want int
	}{
		{base + "/raw-findings?other=1", http.StatusBadRequest},
		{"/api/repository-reviews/automations/rra_missing/raw-findings?query=ALL", http.StatusNotFound},
		{base + "/raw-findings?query=ALL&cursor=bad", http.StatusBadRequest},
	} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if response.Code != test.want {
			t.Fatalf("GET %s status=%d want=%d body=%s", test.path, response.Code, test.want, response.Body.String())
		}
	}

	invalidCampaign := httptest.NewRequest(http.MethodGet, base+"/findings-processing", nil)
	invalidCampaign.SetPathValue("automation_id", automation.ID)
	invalidCampaign.SetPathValue("campaign_id", "bad\x00campaign")
	invalidCampaignResponse := httptest.NewRecorder()
	handler.handleGetRepositoryReviewFindingsProcessing(invalidCampaignResponse, invalidCampaign)
	if invalidCampaignResponse.Code != http.StatusBadRequest {
		t.Fatalf(
			"invalid campaign status=%d body=%s",
			invalidCampaignResponse.Code,
			invalidCampaignResponse.Body.String(),
		)
	}

	wrongMember := httptest.NewRecorder()
	mux.ServeHTTP(wrongMember, httptest.NewRequest(
		http.MethodGet,
		base+"/findings/"+state.DeduplicatedFindings[0].ID+"/sources/"+state.RawFindings[1].ID,
		nil,
	))
	if wrongMember.Code != http.StatusNotFound {
		t.Fatalf("wrong member status=%d body=%s", wrongMember.Code, wrongMember.Body.String())
	}

	state.RawFindings[2].LegacyFindingID = "rfn_historical_retry"
	state.RawFindings[2].AssignmentID = "historical-replay"
	state.RawFindings[2].DiagnosisDigest = repoaudit.RawReviewFindingDiagnosisDigest(state.RawFindings[2])
	persistRepositoryReviewAdditionalCoverageState(t, workspace, state)
	historicalRetry := httptest.NewRequest(
		http.MethodPost, base+"/raw-findings/"+state.RawFindings[2].ID+"/retry",
		strings.NewReader(`{}`),
	)
	setRepositoryReviewMutationHeaders(historicalRetry)
	historicalRetryResponse := httptest.NewRecorder()
	mux.ServeHTTP(historicalRetryResponse, historicalRetry)
	if historicalRetryResponse.Code != http.StatusConflict {
		t.Fatalf(
			"historical per-source retry status=%d body=%s",
			historicalRetryResponse.Code,
			historicalRetryResponse.Body.String(),
		)
	}
}

func TestRepositoryReviewLegacyRunFindingAliasResolvesCanonicalRawSource(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	legacyID := "rfn_pruned_legacy_alias"
	state.RawFindings[0].LegacyFindingID = legacyID
	state.RawFindings[0].DiagnosisDigest = repoaudit.RawReviewFindingDiagnosisDigest(state.RawFindings[0])
	state.DeduplicatedFindings[0].DiagnosisDigest = state.RawFindings[0].DiagnosisDigest
	persistRepositoryReviewAdditionalCoverageState(t, workspace, state)
	if _, found := repositoryReviewFindingByID(state, legacyID); found {
		t.Fatalf("legacy projection %q was not pruned", legacyID)
	}
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations/"+automation.ID+"/run-findings/"+legacyID,
		nil,
	))
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"source":{"id":"`+state.RawFindings[0].ID+`"`) {
		t.Fatalf("legacy alias status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewHistoricalRecoveryNewHelperCoverage(t *testing.T) {
	started := time.Date(2026, 8, 30, 20, 0, 0, 0, time.UTC)
	commitSHA := strings.Repeat("a", 40)
	campaignID := repoaudit.NewRepositoryReviewCampaignID()
	automation := repoaudit.RepositoryReviewAutomation{
		CampaignID: campaignID, ResolvedCommitSHA: commitSHA,
		RunIDs: []string{"wr_current", "wr_old"}, StartedAt: started,
	}
	state := repoaudit.RepositoryState{
		CampaignHistory: map[string]string{campaignID: commitSHA},
		CurrentCampaign: &repoaudit.RepositoryReviewCampaignCoverage{
			ID: campaignID, CommitSHA: commitSHA, Exact: true,
			RecoveryDigest:    "sha256:" + strings.Repeat("b", 64),
			AssignmentCatalog: []repoaudit.RepositoryReviewAssignment{{ID: "assignment"}},
		},
		Runs: []repoaudit.ReviewRun{
			{
				ID: "wr_current", CampaignID: campaignID,
				FindingIDs: []string{"rfn_current"}, CompletedAt: started.Add(time.Minute),
			},
			{ID: "wr_old", CampaignID: "wrong", CompletedAt: started.Add(-time.Minute)},
		},
		Findings: []repoaudit.Finding{{ID: "rfn_current", CampaignID: campaignID}},
	}
	if !repositoryReviewHistoricalCampaignRecovered(automation, state) {
		t.Fatal("old out-of-window run invalidated recovered campaign")
	}
	state.Runs[1].CompletedAt = started.Add(time.Minute)
	if repositoryReviewHistoricalCampaignRecovered(automation, state) {
		t.Fatal("selected run with a foreign campaign was accepted")
	}
	state.Runs[1].CampaignID = campaignID
	state.Findings[0].CampaignID = ""
	if repositoryReviewHistoricalCampaignRecovered(automation, state) {
		t.Fatal("untagged selected finding was accepted")
	}

	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := recoverRepositoryReviewHistoricalCampaign(
		canceled, repoaudit.Store{}, "workspace", automation, state,
		workflows.RepositoryReviewModelProfile{},
	); !strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Fatalf("canceled recovery error=%v", err)
	}
	if _, _, err := recoverRepositoryReviewHistoricalCampaign(
		t.Context(), repoaudit.Store{}, "", automation, state,
		workflows.RepositoryReviewModelProfile{},
	); !strings.Contains(err.Error(), errRepositoryReviewHistoricalCampaignRecovery.Error()) {
		t.Fatalf("empty workspace recovery error=%v", err)
	}
	invalidCampaign := automation
	invalidCampaign.CampaignID = "wr_invalid"
	if _, _, err := recoverRepositoryReviewHistoricalCampaign(
		t.Context(), repoaudit.Store{}, t.TempDir(), invalidCampaign, state,
		workflows.RepositoryReviewModelProfile{},
	); !strings.Contains(err.Error(), errRepositoryReviewHistoricalCampaignRecovery.Error()) {
		t.Fatalf("invalid campaign recovery error=%v", err)
	}
	mismatched := automation
	mismatched.CampaignID = ""
	mismatched.Repository = "owner/other"
	if _, _, err := recoverRepositoryReviewHistoricalCampaign(
		t.Context(), repoaudit.Store{}, t.TempDir(), mismatched, state,
		workflows.RepositoryReviewModelProfile{},
	); !strings.Contains(err.Error(), errRepositoryReviewHistoricalCampaignRecovery.Error()) {
		t.Fatalf("invalid preparation recovery error=%v", err)
	}
	unavailableAutomation := repoaudit.RepositoryReviewAutomation{
		ID: "rra_unavailable_recovery", Repository: "owner/unavailable",
		Status: repoaudit.RepositoryReviewAutomationPaused,
	}
	if _, _, err := recoverRepositoryReviewHistoricalCampaign(
		t.Context(), repoaudit.NewStore(t.TempDir()), t.TempDir(), unavailableAutomation,
		repoaudit.RepositoryState{Repository: unavailableAutomation.Repository},
		workflows.RepositoryReviewModelProfile{
			ReviewerModels: []string{"reviewer"}, MaxContentBytes: 1024,
		},
	); !strings.Contains(err.Error(), errRepositoryReviewHistoricalCampaignRecovery.Error()) {
		t.Fatalf("unavailable exact recovery error=%v", err)
	}

	native := repoaudit.RawReviewFinding{ID: "rrw_native", ContextID: "ctx_native", AssignmentID: "native"}
	historical := repoaudit.RawReviewFinding{
		ID: "rrw_historical", ContextID: "ctx_synthetic", AssignmentID: "historical-replay",
		LegacyFindingID: "rfn_legacy",
	}
	projectionState := repoaudit.RepositoryState{
		RawFindings: []repoaudit.RawReviewFinding{native, historical},
		Contexts: []repoaudit.FindingContext{
			{ID: "ctx_native"},
			{ID: "ctx_synthetic", InventoryHash: "historical-replay", ProfileHash: "historical-replay"},
			{ID: "ctx_retained", InventoryHash: "inventory", ProfileHash: "profile"},
		},
		DeduplicatedFindings: []repoaudit.DeduplicatedReviewFinding{
			{ID: "rdf_native", RawSourceIDs: []string{native.ID}},
			{ID: "rdf_historical", RawSourceIDs: []string{"missing", historical.ID}},
		},
		Findings: []repoaudit.Finding{{ID: "rdf_native"}, {ID: "rdf_historical"}, {ID: "rfn_legacy"}},
	}
	projected := repositoryReviewHistoricalRecoveryProjection(projectionState)
	if len(projected.Contexts) != 2 || len(projected.Findings) != 2 ||
		projected.Findings[0].ID != "rdf_native" || projected.Findings[1].ID != "rfn_legacy" {
		t.Fatalf("historical recovery projection=%#v", projected)
	}
}

func TestRepositoryReviewHistoricalRetryNewProfileAndStoreErrors(t *testing.T) {
	t.Run("profile resolution", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		state := seedRepositoryReviewAPIState(t, workspace)
		state.Findings[0].ID = "rfn_profile_error"
		for index := range state.Runs {
			state.Runs[index].FindingIDs = []string{state.Findings[0].ID}
		}
		state.RawFindings = nil
		state.DeduplicatedFindings = nil
		state.DeduplicationJobs = nil
		state.MappingJobs = nil
		state.HistoricalDeduplication = repoaudit.HistoricalDeduplicationReplay{
			Required: true, Status: repoaudit.HistoricalDeduplicationFailed,
			Attempts: 1, UpdatedAt: time.Now().UTC(),
		}
		persistRepositoryReviewAdditionalCoverageState(t, workspace, state)
		automation := seedRepositoryReviewDetailAutomation(
			t, handler, state.Repository, state.Runs[0].ID,
		)
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/repository-reviews/automations/"+automation.ID+"/historical-deduplication/retry",
			strings.NewReader(`{}`),
		)
		setRepositoryReviewMutationHeaders(request)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest ||
			!strings.Contains(response.Body.String(), "invalid_repository_review_automation") {
			t.Fatalf("profile retry status=%d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("configuration changed after ledger read", func(t *testing.T) {
		handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		state := seedRepositoryReviewAPIState(t, workspace)
		state.Findings[0].ID = "rfn_config_error"
		for index := range state.Runs {
			state.Runs[index].FindingIDs = []string{state.Findings[0].ID}
		}
		state.RawFindings = nil
		state.DeduplicatedFindings = nil
		state.DeduplicationJobs = nil
		state.MappingJobs = nil
		state.HistoricalDeduplication = repoaudit.HistoricalDeduplicationReplay{
			Required: true, Status: repoaudit.HistoricalDeduplicationFailed,
			Attempts: 1, UpdatedAt: time.Now().UTC(),
		}
		persistRepositoryReviewAdditionalCoverageState(t, workspace, state)
		automation := seedRepositoryReviewDetailAutomation(
			t, handler, state.Repository, state.Runs[0].ID,
		)
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/repository-reviews/automations/"+automation.ID+"/historical-deduplication/retry",
			strings.NewReader(`{}`),
		)
		setRepositoryReviewMutationHeaders(request)
		response := httptest.NewRecorder()
		done := make(chan struct{})
		handler.repositoryReviewControllerMu.Lock()
		go func() {
			defer close(done)
			mux.ServeHTTP(response, request)
		}()
		waitForRepositoryReviewControllerLock(t)
		if err := os.WriteFile(handler.configPath, []byte(`{`), 0o600); err != nil {
			handler.repositoryReviewControllerMu.Unlock()
			t.Fatal(err)
		}
		handler.repositoryReviewControllerMu.Unlock()
		<-done
		if response.Code != http.StatusBadRequest {
			t.Fatalf("changed config retry status=%d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("runtime profile drift withholds coverage but retains findings", func(t *testing.T) {
		fixture := newRepositoryReviewBackfillFixture(t, 1, repositoryReviewBackfillRunSpec{
			inspected: []int{0}, occurrences: 1,
		})
		state := fixture.state
		state.HistoricalDeduplication = repoaudit.HistoricalDeduplicationReplay{
			Required: true, Status: repoaudit.HistoricalDeduplicationFailed,
			Attempts: 1, UpdatedAt: time.Now().UTC(),
		}
		persistRepositoryReviewAdditionalCoverageState(t, fixture.workspace, state)
		configPath := filepath.Join(fixture.workspace, "historical-recovery-error-config.json")
		cfg := config.DefaultConfig()
		cfg.Agents.Defaults.Workspace = fixture.workspace
		cfg.Agents.Defaults.ModelName = "review-a"
		cfg.Agents.Defaults.AccountRef = fixture.automation.EffectiveAccountRef
		cfg.ModelList = []*config.ModelConfig{{
			ModelName: fixture.automation.EffectiveAccountRef,
			Provider:  "openai", Model: "openai/test", Enabled: true,
			APIKeys: config.SecureStrings{config.NewSecureString("test-api-key")},
		}}
		cfg.ModelAliases = []config.ModelAliasConfig{{Name: "review-a", Model: "gpt-review"}}
		if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
			t.Fatal(saveErr)
		}
		if _, err := resolveRepositoryReviewCampaignProfile(
			t.Context(), configPath, cfg, fixture.automation,
		); err != nil {
			t.Fatalf("coverage recovery profile: %v", err)
		}
		handler := NewHandler(configPath)
		t.Cleanup(handler.Shutdown)
		mux := http.NewServeMux()
		handler.RegisterRoutes(mux)
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/repository-reviews/automations/"+fixture.automation.ID+
				"/historical-deduplication/retry",
			strings.NewReader(`{}`),
		)
		setRepositoryReviewMutationHeaders(request)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusConflict || !strings.Contains(
			response.Body.String(), `"code":"historical_consolidation_restart_required"`,
		) {
			t.Fatalf("drifted recovery status=%d body=%s", response.Code, response.Body.String())
		}
		unchangedAutomation, found, err := fixture.store.GetAutomation(
			t.Context(), fixture.automation.ID,
		)
		if err != nil || !found || unchangedAutomation.CampaignID != fixture.automation.CampaignID ||
			unchangedAutomation.Version != fixture.automation.Version {
			t.Fatalf("drifted recovery mutated automation=%#v found=%v err=%v",
				unchangedAutomation, found, err)
		}
		unchangedState, found, err := fixture.store.Get(state.Repository)
		if err != nil || !found || unchangedState.Version != state.Version ||
			unchangedState.HistoricalDeduplication.Status != repoaudit.HistoricalDeduplicationFailed {
			t.Fatalf("drifted recovery mutated replay=%#v found=%v err=%v",
				unchangedState.HistoricalDeduplication, found, err)
		}
	})

	t.Run("store changed after ledger read", func(t *testing.T) {
		fixture := newRepositoryReviewBackfillFixture(t, 1, repositoryReviewBackfillRunSpec{
			inspected: []int{0}, occurrences: 1,
		})
		resolved := workflows.RepositoryReviewModelProfile{
			Revision: "legacy-automation-profile", AccountRef: fixture.automation.EffectiveAccountRef,
			ReviewerModels:  fixture.automation.ReviewerModels,
			MaxContentBytes: int(fixture.automation.MaxContentBytes),
		}
		final, recovered, err := recoverRepositoryReviewHistoricalCampaign(
			t.Context(), fixture.store, fixture.workspace,
			fixture.automation, fixture.state, resolved,
		)
		if err != nil {
			t.Fatal(err)
		}
		recovered.HistoricalDeduplication = repoaudit.HistoricalDeduplicationReplay{
			Required: true, Status: repoaudit.HistoricalDeduplicationFailed, Attempts: 1,
			UpdatedAt: time.Now().UTC(),
		}
		persistRepositoryReviewAdditionalCoverageState(t, fixture.workspace, recovered)
		configPath := fixture.workspace + "/historical-retry-config.json"
		cfg := config.DefaultConfig()
		cfg.Agents.Defaults.Workspace = fixture.workspace
		if saveErr := config.SaveConfig(configPath, cfg); saveErr != nil {
			t.Fatal(saveErr)
		}
		handler := NewHandler(configPath)
		t.Cleanup(handler.Shutdown)
		mux := http.NewServeMux()
		handler.RegisterRoutes(mux)
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/repository-reviews/automations/"+final.ID+"/historical-deduplication/retry",
			strings.NewReader(`{}`),
		)
		setRepositoryReviewMutationHeaders(request)
		response := httptest.NewRecorder()
		done := make(chan struct{})
		handler.repositoryReviewControllerMu.Lock()
		go func() {
			defer close(done)
			mux.ServeHTTP(response, request)
		}()
		waitForRepositoryReviewControllerLock(t)
		paths, err := filepath.Glob(filepath.Join(
			fixture.workspace, "repository_reviews", "repo_*.json",
		))
		if err != nil {
			handler.repositoryReviewControllerMu.Unlock()
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
			handler.repositoryReviewControllerMu.Unlock()
			t.Fatal("repository state path is missing")
		}
		if err := os.WriteFile(statePath, []byte(`{`), 0o600); err != nil {
			handler.repositoryReviewControllerMu.Unlock()
			t.Fatal(err)
		}
		handler.repositoryReviewControllerMu.Unlock()
		<-done
		if response.Code != http.StatusBadRequest {
			t.Fatalf("changed store retry status=%d body=%s", response.Code, response.Body.String())
		}
	})
}

func waitForRepositoryReviewControllerLock(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	buffer := make([]byte, 1<<20)
	for time.Now().Before(deadline) {
		length := runtime.Stack(buffer, true)
		stack := string(buffer[:length])
		if strings.Contains(stack, "repositoryReviewControllerInstance") &&
			strings.Contains(stack, "handleRetryRepositoryReviewHistoricalDeduplication") {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("historical retry did not reach the controller lock")
}

func TestRepositoryReviewStrictFindingsOffsetErrorCoverage(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	automation := seedRepositoryReviewDetailAutomation(
		t, handler, state.Repository, state.Runs[0].ID,
	)
	base := "/api/repository-reviews/automations/"

	invalidPage := httptest.NewRecorder()
	mux.ServeHTTP(invalidPage, httptest.NewRequest(
		http.MethodGet, base+automation.ID+"/findings?scope=invalid", nil,
	))
	if invalidPage.Code != http.StatusBadRequest {
		t.Fatalf("invalid strict offset page status=%d body=%s", invalidPage.Code, invalidPage.Body.String())
	}

	missing := httptest.NewRecorder()
	mux.ServeHTTP(missing, httptest.NewRequest(
		http.MethodGet, base+"rra_missing/findings?scope=current", nil,
	))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing strict offset page status=%d body=%s", missing.Code, missing.Body.String())
	}

	legacyReport := httptest.NewRecorder()
	mux.ServeHTTP(legacyReport, httptest.NewRequest(
		http.MethodGet, base+automation.ID+"/report?scope=current&offset=999&limit=1", nil,
	))
	if legacyReport.Code != http.StatusOK || !strings.Contains(
		legacyReport.Body.String(), `"repository_finding_offset":0`,
	) {
		t.Fatalf("bounded legacy report status=%d body=%s", legacyReport.Code, legacyReport.Body.String())
	}
}

func TestRepositoryReviewHistoricalRetryGenericRecoveryErrorCoverage(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	state.RawFindings = nil
	state.DeduplicationJobs = nil
	state.DeduplicatedFindings = nil
	state.FindingsProcessing = repoaudit.FindingsProcessingCounters{}
	state.HistoricalDeduplication = repoaudit.HistoricalDeduplicationReplay{
		Required: true, Status: repoaudit.HistoricalDeduplicationFailed,
		Attempts: 1, UpdatedAt: time.Now().UTC(),
	}
	persistRepositoryReviewAdditionalCoverageState(t, workspace, state)
	automation := seedRepositoryReviewDetailAutomation(
		t, handler, state.Repository, state.Runs[0].ID,
	)

	originalResolver := resolveRepositoryReviewHistoricalCampaignProfile
	originalPrepare := prepareRepositoryReviewHistoricalCampaignBackfill
	t.Cleanup(func() {
		resolveRepositoryReviewHistoricalCampaignProfile = originalResolver
		prepareRepositoryReviewHistoricalCampaignBackfill = originalPrepare
	})
	resolveRepositoryReviewHistoricalCampaignProfile = func(
		context.Context,
		string,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
	) (workflows.RepositoryReviewModelProfile, error) {
		return workflows.RepositoryReviewModelProfile{Revision: "coverage"}, nil
	}
	recoveryErr := errors.New("historical recovery persistence failed")
	prepareRepositoryReviewHistoricalCampaignBackfill = func(
		context.Context,
		repoaudit.RepositoryReviewAutomation,
		repoaudit.RepositoryState,
		string,
		repositoryReviewWorkflowRunLoader,
		...workflows.RepositoryReviewModelProfile,
	) (repositoryReviewLegacyCampaignBackfill, error) {
		return repositoryReviewLegacyCampaignBackfill{}, recoveryErr
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+
			"/historical-deduplication/retry",
		strings.NewReader(`{}`),
	)
	setRepositoryReviewMutationHeaders(request)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("generic recovery status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewHistoricalRecoveryFinalUpdateErrorCoverage(t *testing.T) {
	fixture := newRepositoryReviewBackfillFixture(t, 1, repositoryReviewBackfillRunSpec{
		inspected: []int{0}, occurrences: 1,
	})
	resolved := workflows.RepositoryReviewModelProfile{
		Revision:        "legacy-automation-profile",
		AccountRef:      fixture.automation.EffectiveAccountRef,
		ReviewerModels:  fixture.automation.ReviewerModels,
		MaxContentBytes: int(fixture.automation.MaxContentBytes),
	}
	originalUpdate := updateRepositoryReviewHistoricalAutomation
	t.Cleanup(func() { updateRepositoryReviewHistoricalAutomation = originalUpdate })
	updateErr := errors.New("historical final automation update failed")
	updateRepositoryReviewHistoricalAutomation = func(
		context.Context,
		repoaudit.Store,
		repoaudit.RepositoryReviewAutomation,
		repositoryReviewLegacyCampaignBackfill,
		repoaudit.RepositoryState,
	) (repoaudit.RepositoryReviewAutomation, error) {
		return repoaudit.RepositoryReviewAutomation{}, updateErr
	}
	if _, _, err := recoverRepositoryReviewHistoricalCampaign(
		t.Context(), fixture.store, fixture.workspace,
		fixture.automation, fixture.state, resolved,
	); !errors.Is(err, updateErr) {
		t.Fatalf("final historical update error=%v", err)
	}
}

func TestRepositoryReviewHistoricalRecoveryPreparationErrorClassification(t *testing.T) {
	originalPrepare := prepareRepositoryReviewHistoricalCampaignBackfill
	t.Cleanup(func() { prepareRepositoryReviewHistoricalCampaignBackfill = originalPrepare })
	operationalErr := errors.New("historical workflow store failed")
	tests := []struct {
		name       string
		prepareErr error
		recovery   bool
	}{
		{name: "invalid automation", prepareErr: repoaudit.ErrInvalidAutomation, recovery: true},
		{name: "invalid plan", prepareErr: repoaudit.ErrInvalidPlan, recovery: true},
		{name: "missing history", prepareErr: os.ErrNotExist, recovery: true},
		{name: "canceled", prepareErr: context.Canceled},
		{name: "deadline", prepareErr: context.DeadlineExceeded},
		{name: "operational", prepareErr: operationalErr},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prepareRepositoryReviewHistoricalCampaignBackfill = func(
				context.Context,
				repoaudit.RepositoryReviewAutomation,
				repoaudit.RepositoryState,
				string,
				repositoryReviewWorkflowRunLoader,
				...workflows.RepositoryReviewModelProfile,
			) (repositoryReviewLegacyCampaignBackfill, error) {
				return repositoryReviewLegacyCampaignBackfill{}, test.prepareErr
			}
			_, _, err := recoverRepositoryReviewHistoricalCampaign(
				t.Context(), repoaudit.NewStore(t.TempDir()), "workspace",
				repoaudit.RepositoryReviewAutomation{}, repoaudit.RepositoryState{},
				workflows.RepositoryReviewModelProfile{},
			)
			if test.recovery {
				if !errors.Is(err, errRepositoryReviewHistoricalCampaignRecovery) {
					t.Fatalf("preparation error=%v", err)
				}
				return
			}
			if !errors.Is(err, test.prepareErr) ||
				errors.Is(err, errRepositoryReviewHistoricalCampaignRecovery) {
				t.Fatalf("preparation error=%v", err)
			}
		})
	}
}

func TestRepositoryReviewHistoricalRecoveryNewErrorCoverage(t *testing.T) {
	controller := newRepositoryReviewController(nil)
	controller.leasedConfig = &config.Config{}
	controller.leasedStore = repoaudit.NewStore(t.TempDir())
	controller.wakeHistoricalFindingDeduplication()
	controller.cancel()

	resolved := func(fixture repositoryReviewBackfillFixture) workflows.RepositoryReviewModelProfile {
		return workflows.RepositoryReviewModelProfile{
			Revision: "legacy-automation-profile", AccountRef: fixture.automation.EffectiveAccountRef,
			ReviewerModels:  fixture.automation.ReviewerModels,
			MaxContentBytes: int(fixture.automation.MaxContentBytes),
		}
	}

	missingAutomation := newRepositoryReviewBackfillFixture(t, 1, repositoryReviewBackfillRunSpec{
		inspected: []int{0}, occurrences: 1,
	})
	missing := missingAutomation.automation
	missing.ID = "rra_missing_historical_install"
	if _, _, err := recoverRepositoryReviewHistoricalCampaign(
		t.Context(), missingAutomation.store, missingAutomation.workspace,
		missing, missingAutomation.state, resolved(missingAutomation),
	); err == nil || !strings.Contains(strings.ToLower(err.Error()), "not exist") {
		t.Fatalf("missing automation install error=%v", err)
	}

	conflictingState := newRepositoryReviewBackfillFixture(t, 1, repositoryReviewBackfillRunSpec{
		inspected: []int{0}, occurrences: 1,
	})
	if _, err := conflictingState.store.BeginCampaign(t.Context(), repoaudit.BeginCampaignRequest{
		Repository:            conflictingState.state.Repository,
		CampaignID:            repoaudit.NewRepositoryReviewCampaignID(),
		CommitSHA:             conflictingState.automation.ResolvedCommitSHA,
		ExpectedReviewVersion: conflictingState.state.ReviewVersion,
		Exact:                 true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := recoverRepositoryReviewHistoricalCampaign(
		t.Context(), conflictingState.store, conflictingState.workspace,
		conflictingState.automation, conflictingState.state, resolved(conflictingState),
	); err == nil {
		t.Fatal("conflicting durable campaign did not fail apply")
	}
}
