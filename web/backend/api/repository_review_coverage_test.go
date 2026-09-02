package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/repoaudit"
	"github.com/sipeed/picoclaw/pkg/workflows"
)

type closeableRepositoryReviewProfileRunner struct {
	*repositoryReviewRecoveryProfileRunner
	closed int
}

func (runner *closeableRepositoryReviewProfileRunner) Close() error {
	runner.closed++
	return nil
}

func TestRepositoryReviewCoverageDetailAndDraftUpdateHandlers(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)

	detail := httptest.NewRecorder()
	mux.ServeHTTP(detail, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/"+state.ID+"?offset=0&limit=1&draft_offset=0&draft_limit=1",
		nil,
	))
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status=%d body=%s", detail.Code, detail.Body.String())
	}
	var projected repositoryReviewDetailResponse
	if err := json.Unmarshal(detail.Body.Bytes(), &projected); err != nil {
		t.Fatal(err)
	}
	if projected.ID != state.ID || projected.FindingTotal != 1 || len(projected.Findings) != 1 ||
		len(projected.Contexts) != 1 {
		t.Fatalf("detail projection=%#v", projected)
	}

	invalidPage := httptest.NewRecorder()
	mux.ServeHTTP(invalidPage, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/"+state.ID+"?limit=0",
		nil,
	))
	if invalidPage.Code != http.StatusBadRequest {
		t.Fatalf("invalid page status=%d body=%s", invalidPage.Code, invalidPage.Body.String())
	}

	missingID := "rrp_" + strings.Repeat("f", 64)
	missing := httptest.NewRecorder()
	mux.ServeHTTP(missing, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/"+missingID,
		nil,
	))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing detail status=%d body=%s", missing.Code, missing.Body.String())
	}
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)

	prepared := repositoryReviewCoverageMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/"+state.ID+"/issue-drafts",
		map[string]any{
			"finding_ids": []string{state.Findings[0].ID},
			"title":       "Initial issue", "body": "Initial body", "labels": []string{"bug"},
			"expected_version": state.Version,
		},
	)
	if prepared.Code != http.StatusCreated {
		t.Fatalf("prepare status=%d body=%s", prepared.Code, prepared.Body.String())
	}
	var preparedResult struct {
		Repository repoaudit.RepositorySummary `json:"repository"`
		Draft      repoaudit.IssueDraft        `json:"draft"`
	}
	if err := json.Unmarshal(prepared.Body.Bytes(), &preparedResult); err != nil {
		t.Fatal(err)
	}

	updated := repositoryReviewCoverageMutation(
		t,
		mux,
		http.MethodPatch,
		"/api/repository-reviews/"+state.ID+"/issue-drafts/"+preparedResult.Draft.ID,
		map[string]any{
			"title": "Updated issue", "body": "Updated body", "labels": []string{"bug", "reviewed"},
			"expected_version": preparedResult.Draft.Version,
		},
	)
	if updated.Code != http.StatusOK {
		t.Fatalf("update draft status=%d body=%s", updated.Code, updated.Body.String())
	}
	var updatedResult struct {
		Repository repoaudit.RepositorySummary `json:"repository"`
		Draft      repoaudit.IssueDraft        `json:"draft"`
	}
	if err := json.Unmarshal(updated.Body.Bytes(), &updatedResult); err != nil {
		t.Fatal(err)
	}
	if updatedResult.Draft.Title != "Updated issue" || updatedResult.Draft.Body != "Updated body" ||
		len(updatedResult.Draft.Labels) != 2 || updatedResult.Draft.Version <= preparedResult.Draft.Version {
		t.Fatalf("updated draft=%#v", updatedResult.Draft)
	}

	missingMutation := repositoryReviewCoverageMutation(
		t,
		mux,
		http.MethodPatch,
		"/api/repository-reviews/"+missingID+"/issue-drafts/"+preparedResult.Draft.ID,
		map[string]any{
			"title": "No repository", "body": "No repository", "expected_version": 1,
		},
	)
	if missingMutation.Code != http.StatusNotFound {
		t.Fatalf("missing mutation status=%d body=%s", missingMutation.Code, missingMutation.Body.String())
	}
}

func TestRepositoryReviewCoverageAutomationOptionsAndAccountProjection(t *testing.T) {
	withPicoclawAuthHome(t)
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automation-options",
		nil,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("options status=%d body=%s", response.Code, response.Body.String())
	}
	var options repositoryReviewAutomationOptionsCoverageResponse
	if err := json.Unmarshal(response.Body.Bytes(), &options); err != nil {
		t.Fatal(err)
	}
	if len(options.Models) != 2 || len(options.Accounts) != 1 ||
		options.Accounts[0].ID != "api" || !options.Accounts[0].Default ||
		!options.Accounts[0].Available {
		t.Fatalf("options=%#v", options)
	}

	used := 25
	accounts := repositoryReviewAccountOptions(nil, codexAccountLimitsResponse{Accounts: []codexAccountLimitAccount{
		{
			ID: "openai:work", Provider: "openai", Email: "work@example.test",
			LimitsStatus: "available", Entries: []codexAccountLimitEntry{{
				Name: "Codex", Status: "available", Window: "weekly", UsedPercent: &used,
				RefreshesAt: "2026-08-27T12:00:00Z",
			}},
		},
		{ID: "github:backup", Provider: "github-copilot", AccountID: "backup-id", CredentialStatus: "missing"},
	}})
	accountByID := make(map[string]repositoryReviewAccountOption, len(accounts))
	for _, account := range accounts {
		accountByID[account.ID] = account
	}
	work := accountByID["credential:openai:work"]
	backup := accountByID["credential:github:backup"]
	if len(accounts) != 2 || !strings.Contains(work.Label, "work@example.test") || len(work.Entries) != 1 ||
		work.Entries[0].RemainingPercent == nil || *work.Entries[0].RemainingPercent != 75 ||
		!strings.Contains(backup.Label, "backup-id") || backup.Status != "missing" {
		t.Fatalf("account options=%#v", accounts)
	}

	missingConfigHandler := NewHandler(t.TempDir())
	missingConfigMux := http.NewServeMux()
	missingConfigHandler.RegisterRoutes(missingConfigMux)
	missingConfigResponse := httptest.NewRecorder()
	missingConfigMux.ServeHTTP(missingConfigResponse, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automation-options",
		nil,
	))
	if missingConfigResponse.Code != http.StatusInternalServerError {
		t.Fatalf(
			"missing config options status=%d body=%s",
			missingConfigResponse.Code,
			missingConfigResponse.Body.String(),
		)
	}
}

type repositoryReviewAutomationOptionsCoverageResponse struct {
	Models   []repositoryReviewModelOption   `json:"models"`
	Accounts []repositoryReviewAccountOption `json:"accounts"`
}

func TestRepositoryReviewCoverageAutomationMutationBranches(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, storeErr := handler.repositoryReviewStore()
	if storeErr != nil {
		t.Fatal(storeErr)
	}
	profile := createRepositoryReviewProfileForTest(t, mux, "Coverage", "cheap")

	createdResponse := repositoryReviewAutomationMutation(t, mux, http.MethodPost,
		"/api/repository-reviews/automations", map[string]any{
			"repository": "owner/coverage",
			"branch":     "main",
			"profile_id": profile.ID,
		})
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create default status=%d body=%s", createdResponse.Code, createdResponse.Body.String())
	}
	var created struct {
		Automation repoaudit.RepositoryReviewAutomation `json:"automation"`
	}
	if err := json.Unmarshal(createdResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Automation.ProfileID != profile.ID ||
		len(created.Automation.ReviewerModels) != 1 ||
		created.Automation.ReviewerModels[0] != "cheap" ||
		!created.Automation.AutoContinue {
		t.Fatalf("defaulted automation=%#v", created.Automation)
	}

	paused, updateErr := store.UpdateAutomation(t.Context(), created.Automation.ID, created.Automation.Version,
		func(candidate *repoaudit.RepositoryReviewAutomation) error {
			candidate.Status = repoaudit.RepositoryReviewAutomationPaused
			candidate.PauseReason = repoaudit.RepositoryReviewPauseManual
			candidate.PauseDetail = "paused before reconfiguration"
			candidate.Progress = repoaudit.RepositoryReviewProgress{TotalBatches: 2, CompletedBatches: 1}
			candidate.Usage = repoaudit.RepositoryReviewTokenUsage{
				PromptTokens:     8,
				CompletionTokens: 2,
				TotalTokens:      10,
			}
			candidate.EstimatedCostUSD = 0.25
			candidate.StartedAt = time.Now().UTC().Add(-time.Minute)
			return nil
		})
	if updateErr != nil {
		t.Fatal(updateErr)
	}
	updateBody := map[string]any{
		"repository":       paused.Repository,
		"branch":           "release/v2",
		"profile_id":       profile.ID,
		"expected_version": paused.Version,
	}
	updatedResponse := repositoryReviewAutomationMutation(t, mux, http.MethodPatch,
		"/api/repository-reviews/automations/"+paused.ID, updateBody)
	if updatedResponse.Code != http.StatusOK {
		t.Fatalf("reconfigure status=%d body=%s", updatedResponse.Code, updatedResponse.Body.String())
	}
	var updated struct {
		Automation repoaudit.RepositoryReviewAutomation `json:"automation"`
	}
	if err := json.Unmarshal(updatedResponse.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Automation.Status != repoaudit.RepositoryReviewAutomationIdle ||
		updated.Automation.Usage.TotalTokens != 0 || updated.Automation.Progress.CompletedBatches != 0 ||
		!updated.Automation.StartedAt.IsZero() {
		t.Fatalf("reconfigured automation=%#v", updated.Automation)
	}

	runningInput := testRepositoryReviewAutomation()
	runningInput.Status = repoaudit.RepositoryReviewAutomationRunning
	runningInput.ActiveRunID = "wr_pause_coverage"
	runningInput.RunIDs = []string{"wr_pause_coverage"}
	runningInput.Progress.TotalBatches = 1
	running, err := store.CreateAutomation(t.Context(), runningInput)
	if err != nil {
		t.Fatal(err)
	}
	running, err = store.UpdateAutomation(
		t.Context(), running.ID, running.Version,
		func(*repoaudit.RepositoryReviewAutomation) error { return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	controller := handler.repositoryReviewControllerInstance()
	controller.mu.Lock()
	controller.active[running.ID] = &repositoryReviewActiveRun{runID: running.ActiveRunID, store: store}
	controller.mu.Unlock()
	handler.StartRepositoryReviewController()
	purgeEligibility, purgeErr := store.RepositoryReviewPurgeEligibilityForAutomation(running)
	if purgeErr != nil {
		t.Fatal(purgeErr)
	}

	deleteActive := repositoryReviewAutomationMutation(t, mux, http.MethodDelete,
		"/api/repository-reviews/automations/"+running.ID,
		map[string]any{
			"expected_version":            running.Version,
			"expected_repository_version": 0,
			"expected_ledger_fence":       purgeEligibility.Summary.LedgerFence,
			"confirm_repository":          running.Repository,
		})
	if deleteActive.Code != http.StatusConflict ||
		!strings.Contains(deleteActive.Body.String(), "repository_review_purge_blocked") ||
		!strings.Contains(deleteActive.Body.String(), "review_active") {
		t.Fatalf("delete active status=%d body=%s", deleteActive.Code, deleteActive.Body.String())
	}

	pausedResponse := repositoryReviewAutomationMutation(t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+running.ID+"/pause",
		map[string]any{"expected_version": running.Version})
	if pausedResponse.Code != http.StatusAccepted {
		t.Fatalf("pause status=%d body=%s", pausedResponse.Code, pausedResponse.Body.String())
	}
	var pauseResult struct {
		Automation repoaudit.RepositoryReviewAutomation `json:"automation"`
	}
	if err := json.Unmarshal(pausedResponse.Body.Bytes(), &pauseResult); err != nil {
		t.Fatal(err)
	}
	if pauseResult.Automation.Status != repoaudit.RepositoryReviewAutomationStopping ||
		pauseResult.Automation.RequestedPauseReason != repoaudit.RepositoryReviewPauseManual {
		t.Fatalf("pause result=%#v", pauseResult.Automation)
	}

	missingDelete := repositoryReviewAutomationMutation(t, mux, http.MethodDelete,
		"/api/repository-reviews/automations/rra_missing",
		map[string]any{
			"expected_version":            1,
			"expected_repository_version": 0,
			"expected_ledger_fence":       "rplf_missing",
			"confirm_repository":          "owner/missing",
		})
	if missingDelete.Code != http.StatusNotFound {
		t.Fatalf("missing delete status=%d body=%s", missingDelete.Code, missingDelete.Body.String())
	}

	badHandler := NewHandler(t.TempDir())
	badMux := http.NewServeMux()
	badHandler.RegisterRoutes(badMux)
	badList := httptest.NewRecorder()
	badMux.ServeHTTP(badList, httptest.NewRequest(http.MethodGet, "/api/repository-reviews/automations", nil))
	if badList.Code != http.StatusInternalServerError {
		t.Fatalf("bad list status=%d body=%s", badList.Code, badList.Body.String())
	}
}

func TestRepositoryReviewSplitCoverageOffsets(t *testing.T) {
	for _, repository := range []string{
		"http://example.com:81/owner/repository",
		"git://example.com:9418/owner/repository",
	} {
		if normalized, err := normalizeRepositoryReviewAutomationRepository(repository); err == nil {
			t.Fatalf("unsafe repository %q normalized to %q", repository, normalized)
		}
	}
	if normalized, err := normalizeRepositoryReviewAutomationRepository(
		"https://[2001:db8::1]/owner/repository",
	); err != nil || normalized != "https://[2001:db8::1]/owner/repository.git" {
		t.Fatalf("IPv6 repository normalization = (%q, %v)", normalized, err)
	}

	validScope := repoaudit.RepositoryReviewScopePolicy{
		CodeTypes:      []repoaudit.RepositoryReviewCodeType{repoaudit.RepositoryReviewCodeTypeCode},
		IncludeFolders: []string{"pkg"},
		ExcludeFolders: []string{"pkg/generated"},
	}
	invalidScope := validScope
	invalidScope.CodeTypes = []repoaudit.RepositoryReviewCodeType{"invalid"}
	if repositoryReviewScopePoliciesEqual(invalidScope, validScope) {
		t.Fatal("invalid scope policies compared equal")
	}
	if slicesEqualRepositoryReviewCodeTypes(
		validScope.CodeTypes,
		append(validScope.CodeTypes, repoaudit.RepositoryReviewCodeTypeTest),
	) {
		t.Fatal("different-length code type slices compared equal")
	}
	if slicesEqualRepositoryReviewCodeTypes(
		validScope.CodeTypes,
		[]repoaudit.RepositoryReviewCodeType{repoaudit.RepositoryReviewCodeTypeTest},
	) {
		t.Fatal("different code type slices compared equal")
	}

	controller := &repositoryReviewController{}
	if _, err := controller.normalizeRepositoryReviewAutomationAdmission(
		t.Context(),
		repoaudit.Store{},
		repoaudit.RepositoryReviewAutomation{Repository: "relative/path/extra"},
	); err == nil {
		t.Fatal("admission accepted an invalid repository")
	}

	automation := repoaudit.RepositoryReviewAutomation{}
	applyRepositoryReviewRunProgress(&automation, &workflows.RunResult{Outputs: map[string]any{
		"scopePlan": map[string]any{
			"commit_sha": strings.Repeat("a", 40), "policy_hash": strings.Repeat("b", 64),
			"hash": strings.Repeat("c", 64), "summary": "Frozen test scope",
		},
		"scopeSelection": map[string]any{
			"include_prefixes": []any{"pkg"}, "exclude_prefixes": []any{},
			"candidate_ids": []any{}, "hotpath_candidate_ids": []any{},
		},
	}}, nil)
	if automation.ScopePlan.CommitSHA != strings.Repeat("a", 40) ||
		automation.ScopeSelection == nil ||
		!reflect.DeepEqual(automation.ScopeSelection.IncludePrefixes, []string{"pkg"}) {
		t.Fatalf("scope plan was not projected: %#v", automation.ScopePlan)
	}
}

func TestRepositoryReviewCoverageControllerHelpersAndOutcome(t *testing.T) {
	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}

	automation := repoaudit.RepositoryReviewAutomation{
		Repository: "owner/repo", ReviewerModels: []string{"review-model"}, RunIDs: []string{"api-run"},
		ModelStats: make(map[string]repoaudit.RepositoryReviewModelStats),
	}
	outcome := loadRepositoryReviewOutcome(store, automation)
	if !outcome.found || outcome.reviewedFiles != 1 || outcome.findings != 1 ||
		outcome.modelFindings["review-model"] != 1 || len(outcome.modelPaths["review-model"]) != 1 {
		t.Fatalf("loaded outcome=%#v state=%#v", outcome, state)
	}
	applyRepositoryReviewOutcome(&automation, outcome)
	if automation.Progress.ReviewedFiles != 1 || automation.Progress.Findings != 1 ||
		automation.ModelStats["review-model"].Findings != 1 ||
		automation.ModelStats["review-model"].ReviewedFiles < 1 {
		t.Fatalf("applied automation=%#v", automation)
	}

	if got := mapStringValues(
		map[string]string{"b": "second", "a": "first"},
	); len(got) != 2 || got[0] != "first" ||
		got[1] != "second" {
		t.Fatalf("map values=%#v", got)
	}
	models := repoaudit.RepositoryReviewAutomation{ReviewerModels: []string{"cheap", "quality"}}
	if got := repositoryReviewExecutionModels(models); len(got) != 1 || got[0] != "cheap" {
		t.Fatalf("single execution models=%#v", got)
	}
	models.CompareModels = true
	if got := repositoryReviewExecutionModels(models); len(got) != 2 {
		t.Fatalf("comparison execution models=%#v", got)
	}

	cfg := config.DefaultConfig()
	cfg.ModelAliases = []config.ModelAliasConfig{{
		Name: "cheap", Model: "openai/gpt-cheap",
		AccountOverrides: map[string]string{"work": "anthropic/claude-cheap"},
	}}
	priced := repoaudit.RepositoryReviewAutomation{
		ReviewerModels: []string{"cheap"},
		ModelPrices: map[string]repoaudit.RepositoryReviewModelPrice{
			"cheap": {InputPricePer1M: 1, OutputPricePer1M: 2},
		},
	}
	index := repositoryReviewAccountingIndex(cfg, priced)
	if index["cheap"].alias != "cheap" || index["gpt-cheap"].alias != "cheap" ||
		index["anthropic/claude-cheap"].alias != "cheap" || !index["*"].known {
		t.Fatalf("accounting index=%#v", index)
	}

	controller := newRepositoryReviewController(handler)
	controller.active["rra_latch"] = &repositoryReviewActiveRun{runID: "wr_latch", store: store}
	latchErr := controller.latchAccountingFailure("rra_latch", "wr_latch", errors.New("disk full"))
	if !errors.Is(latchErr, errRepositoryReviewSafeStop) ||
		controller.active["rra_latch"].pauseReason != repoaudit.RepositoryReviewPauseRunFailed ||
		!strings.Contains(controller.active["rra_latch"].pauseDetail, "disk full") {
		t.Fatalf("latch error=%v active=%#v", latchErr, controller.active["rra_latch"])
	}

	admissionController := newRepositoryReviewController(handler)
	admissionController.active["rra_admit"] = &repositoryReviewActiveRun{runID: "wr_admit"}
	if err := admissionController.admitProviderCall("rra_admit", "wr_admit"); err != nil {
		t.Fatalf("admitted call error=%v", err)
	}
	admissionController.active["rra_admit"].pauseReason = repoaudit.RepositoryReviewPauseManual
	admissionController.active["rra_admit"].pauseDetail = "manual stop"
	if err := admissionController.admitProviderCall(
		"rra_admit",
		"wr_admit",
	); !errors.Is(
		err,
		errRepositoryReviewSafeStop,
	) {
		t.Fatalf("paused admission error=%v", err)
	}
	delete(admissionController.active, "rra_admit")
	if err := admissionController.admitProviderCall(
		"rra_admit",
		"wr_admit",
	); !errors.Is(
		err,
		errRepositoryReviewSafeStop,
	) {
		t.Fatalf("missing admission error=%v", err)
	}
	admissionController.cancel()
	if err := admissionController.admitProviderCall(
		"rra_admit",
		"wr_admit",
	); !errors.Is(
		err,
		errRepositoryReviewSafeStop,
	) {
		t.Fatalf("canceled admission error=%v", err)
	}
	var nilController *repositoryReviewController
	if err := nilController.admitProviderCall("rra_admit", "wr_admit"); !errors.Is(err, errRepositoryReviewSafeStop) {
		t.Fatalf("nil admission error=%v", err)
	}

	if allowed, guardErr := repoaudit.EvaluateRepositoryReviewGuardExpression(
		"spend.total.usd < 1",
		repoaudit.RepositoryReviewGuardEnvironment{SpendTotalUSD: 2, CostKnown: true},
	); guardErr != nil || allowed {
		t.Fatalf("cost guard allowed=%v err=%v", allowed, guardErr)
	}
	if got := repositoryReviewAnySlice([]map[string]any{{"id": 1}}); len(got) != 1 {
		t.Fatalf("map slice=%#v", got)
	}
	if got := repositoryReviewAnySlice("not-a-slice"); got != nil {
		t.Fatalf("invalid slice=%#v", got)
	}

	for name, value := range map[string]any{
		"int": 1, "int64": int64(2), "float64": float64(3), "float32": float32(4), "string": "5",
	} {
		if got := repositoryReviewInt(value); got < 1 || got > 5 {
			t.Fatalf("repositoryReviewInt(%s)=%d", name, got)
		}
	}
	if got := repositoryReviewInt(true); got != 0 {
		t.Fatalf("repositoryReviewInt(bool)=%d", got)
	}
	if got := repositoryReviewRunError(errors.New("provider failed"), nil); got != "provider failed" {
		t.Fatalf("run error=%q", got)
	}
	if got := repositoryReviewRunError(nil, &workflows.RunResult{Error: "result failed"}); got != "result failed" {
		t.Fatalf("result error=%q", got)
	}
	if got := repositoryReviewRunError(nil, nil); got == "" {
		t.Fatal("default run error is empty")
	}
	bounded := repositoryReviewBoundedDetail(strings.Repeat("é", 3000))
	if len(bounded) > 4096 || !utf8.ValidString(bounded) || !strings.HasSuffix(bounded, "...") {
		t.Fatalf("bounded detail bytes=%d valid=%v", len(bounded), utf8.ValidString(bounded))
	}
	if normalizeRepositoryReviewWindow("7d") != "weekly" ||
		normalizeRepositoryReviewWindow("24h") != "daily" ||
		normalizeRepositoryReviewWindow("") != "unknown" ||
		normalizeRepositoryReviewWindow("monthly") != "monthly" {
		t.Fatal("window normalization mismatch")
	}
	if reset, ok := parseRepositoryReviewReset("2026-08-27T12:00:00Z"); !ok || reset.IsZero() {
		t.Fatalf("RFC3339 reset=%s ok=%v", reset, ok)
	}
	if reset, ok := parseRepositoryReviewReset("2026-08-27 12:00:00 UTC"); !ok || reset.IsZero() {
		t.Fatalf("display reset=%s ok=%v", reset, ok)
	}
	if _, ok := parseRepositoryReviewReset("-"); ok {
		t.Fatal("dash reset unexpectedly parsed")
	}
}

func TestRepositoryReviewCoverageFinishAutomationBranches(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	controller := handler.repositoryReviewControllerInstance()

	createRunning := func(runID string, autoContinue bool) repoaudit.RepositoryReviewAutomation {
		t.Helper()
		candidate := testRepositoryReviewAutomation()
		candidate.AutoContinue = autoContinue
		candidate.Status = repoaudit.RepositoryReviewAutomationRunning
		candidate.ActiveRunID = runID
		candidate.RunIDs = []string{runID}
		candidate.Progress.TotalBatches = 1
		created, createErr := store.CreateAutomation(t.Context(), candidate)
		if createErr != nil {
			t.Fatal(createErr)
		}
		controller.mu.Lock()
		controller.active[created.ID] = &repositoryReviewActiveRun{runID: runID, store: store}
		controller.mu.Unlock()
		return created
	}

	checkpointed := createRunning("wr_finish_checkpoint", false)
	controller.finishAutomationRun(checkpointed.ID, checkpointed.ActiveRunID, &workflows.RunResult{
		RunID: checkpointed.ActiveRunID, Status: workflows.RunStatusSucceeded,
		Outputs: map[string]any{"remainingFiles": "invalid", "reviewedFiles": 1},
	}, nil, true, &workflows.Run{Steps: map[string]workflows.StepExecution{
		"find_bugs/record": {
			Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{"run": map[string]any{
				"remaining_files": 2, "reviewed_files": 1, "unsupported_files": 0,
			}},
		},
	}})
	checkpointed, _, err = store.GetAutomation(t.Context(), checkpointed.ID)
	if err != nil || checkpointed.Status != repoaudit.RepositoryReviewAutomationPaused ||
		checkpointed.PauseReason != repoaudit.RepositoryReviewPauseManual ||
		checkpointed.Progress.CompletedBatches != 1 || checkpointed.Progress.RemainingFiles != 2 {
		t.Fatalf("checkpointed finish=%#v err=%v", checkpointed, err)
	}

	failed := createRunning("wr_finish_failed", false)
	controller.finishAutomationRun(
		failed.ID,
		failed.ActiveRunID,
		&workflows.RunResult{RunID: failed.ActiveRunID, Status: workflows.RunStatusFailed, Error: "workflow failed"},
		errors.New("provider failed"),
		false,
		nil,
	)
	failed, _, err = store.GetAutomation(t.Context(), failed.ID)
	if err != nil || failed.Status != repoaudit.RepositoryReviewAutomationFailed ||
		failed.PauseReason != repoaudit.RepositoryReviewPauseRunFailed ||
		failed.Progress.CompletedBatches != 0 || !strings.Contains(failed.PauseDetail, "provider failed") {
		t.Fatalf("failed finish=%#v err=%v", failed, err)
	}

	missingCheckpoint := createRunning("wr_finish_missing", false)
	controller.finishAutomationRun(missingCheckpoint.ID, missingCheckpoint.ActiveRunID, &workflows.RunResult{
		RunID: missingCheckpoint.ActiveRunID, Status: workflows.RunStatusSucceeded,
		Outputs: map[string]any{"remainingFiles": 0},
	}, nil, false, nil)
	missingCheckpoint, _, err = store.GetAutomation(t.Context(), missingCheckpoint.ID)
	if err != nil || missingCheckpoint.Status != repoaudit.RepositoryReviewAutomationFailed ||
		!strings.Contains(missingCheckpoint.PauseDetail, "without a verified durable") {
		t.Fatalf("missing checkpoint finish=%#v err=%v", missingCheckpoint, err)
	}
}

func TestRepositoryReviewFileProgressMadeUsesOnlyResolvedFiles(t *testing.T) {
	base := repoaudit.RepositoryReviewProgress{
		ReviewedFiles: 2, UnsupportedFiles: 1, RemainingFiles: 7, Findings: 3,
	}
	durableRun := func(reviewed, unsupported any) *workflows.Run {
		return &workflows.Run{Steps: map[string]workflows.StepExecution{
			"find_bugs/record": {
				Status: workflows.RunStatusSucceeded,
				Outputs: map[string]any{"run": map[string]any{
					"reviewed_files": reviewed, "unsupported_files": unsupported,
				}},
			},
		}}
	}
	for _, test := range []struct {
		name           string
		after          repoaudit.RepositoryReviewProgress
		persistedRun   *workflows.Run
		outcome        repositoryReviewOutcome
		allowProjected bool
		want           bool
	}{
		{name: "unchanged", after: base},
		{name: "finding only", after: func() repoaudit.RepositoryReviewProgress {
			value := base
			value.Findings++
			return value
		}()},
		{name: "remaining drop", after: func() repoaudit.RepositoryReviewProgress {
			value := base
			value.RemainingFiles--
			return value
		}(), want: true},
		{name: "projected fully completed rise", after: func() repoaudit.RepositoryReviewProgress {
			value := base
			value.ReviewedFiles++
			return value
		}()},
		{name: "test seam projected rise", after: func() repoaudit.RepositoryReviewProgress {
			value := base
			value.ReviewedFiles++
			return value
		}(), allowProjected: true, want: true},
		{name: "durable reviewed files", after: base, persistedRun: durableRun(1, 0), want: true},
		{name: "durable unsupported files", after: base, persistedRun: durableRun(0, 1), want: true},
		{name: "missing durable count", after: base, persistedRun: durableRun(1, nil)},
		{name: "string durable count", after: base, persistedRun: durableRun("1", 0)},
		{name: "negative durable count", after: base, persistedRun: durableRun(-1, 0)},
		{name: "fractional durable count", after: base, persistedRun: durableRun(0.5, 0)},
		{
			name: "above-domain durable count", after: base,
			persistedRun: durableRun(repositoryReviewMaximumFiles+1, 0),
		},
		{name: "ledger reviewed rise", after: base, outcome: repositoryReviewOutcome{
			found: true, reviewedFiles: base.ReviewedFiles + 1,
		}, want: true},
		{name: "ledger unsupported rise", after: func() repoaudit.RepositoryReviewProgress {
			value := base
			value.UnsupportedFiles++
			return value
		}(), outcome: repositoryReviewOutcome{
			found: true, unsupportedFiles: base.UnsupportedFiles + 1,
		}, want: true},
		{name: "exact campaign ledger rise", after: base, outcome: repositoryReviewOutcome{
			found: true, coverageAvailable: true, coverageExact: true,
			reviewedFiles: base.ReviewedFiles + 1,
		}, want: true},
		{name: "inexact campaign lower bound is not operational progress", after: base, outcome: repositoryReviewOutcome{
			found: true, coverageAvailable: true, coverageExact: false,
			reviewedFiles: base.ReviewedFiles + 1,
		}},
		{name: "initial remaining baseline is not progress", after: func() repoaudit.RepositoryReviewProgress {
			value := base
			value.RemainingFiles = 6
			return value
		}()},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := base
			if test.name == "initial remaining baseline is not progress" {
				before.RemainingFiles = 0
			}
			if got := repositoryReviewFileProgressMade(
				before,
				test.after,
				test.persistedRun,
				test.outcome,
				test.allowProjected,
			); got != test.want {
				t.Fatalf("file progress=%v, want %v", got, test.want)
			}
		})
	}
}

func TestRepositoryReviewFinishIgnoresProjectedFileProgress(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	file := repoaudit.FileRef{
		Path: "pkg/projected.go", BlobSHA: strings.Repeat("d", 40), SizeBytes: 80,
		Category: "code", Mode: "100644",
	}
	plan, err := store.Plan(
		t.Context(), "owner/projected-progress", "commit-projected", "inventory-projected",
		[]repoaudit.FileRef{file}, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	line := 9
	runID := "wr_projected_progress"
	recorded, err := store.Record(t.Context(), repoaudit.RecordRequest{
		Plan: plan, RunID: runID, CompletedFiles: []repoaudit.FileRef{},
		Observations: []repoaudit.Observation{{
			Model: "cheap", ScopeFiles: []repoaudit.FileRef{file},
			Findings: []repoaudit.FindingCandidate{{
				Severity: "high", Title: "Finding without completed coverage", File: file.Path,
				Line: &line, Message: "The partial review found a defect.",
				Evidence:   "The failing path is visible in the assigned evidence.",
				Impact:     "The operation fails.",
				Validation: repoaudit.Validation{Status: "confirmed", Summary: "confirmed"},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if recorded.Run.ReviewedFiles != 0 || recorded.Run.RemainingFiles != 1 ||
		len(recorded.AcceptedFindingIDs) != 1 {
		t.Fatalf("partial record=%#v", recorded)
	}

	input := testRepositoryReviewAutomation()
	input.Repository = plan.Repository
	input.Name = "Projected progress"
	input.Status = repoaudit.RepositoryReviewAutomationRunning
	input.ActiveRunID = runID
	input.RunIDs = []string{runID}
	input.AutoContinue = true
	input.Progress = repoaudit.RepositoryReviewProgress{RemainingFiles: 1, TotalBatches: 1}
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	controller := handler.repositoryReviewControllerInstance()
	controller.mu.Lock()
	controller.active[automation.ID] = &repositoryReviewActiveRun{runID: runID, store: store}
	controller.mu.Unlock()
	persisted := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"find_bugs/record": {
			Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{"run": map[string]any{
				"remaining_files": 1, "reviewed_files": 0, "unsupported_files": 0,
			}},
		},
	}}
	result := &workflows.RunResult{
		RunID: runID, Status: workflows.RunStatusSucceeded,
		Outputs: map[string]any{
			"remainingFiles": 1, "reviewedFiles": 1,
			"findingIds": recorded.AcceptedFindingIDs,
		},
	}
	if !repositoryReviewRunCheckpointed(persisted, result) {
		t.Fatal("production-style record was not a verified checkpoint")
	}
	controller.finishAutomationRun(automation.ID, runID, result, nil, true, persisted)

	paused, found, err := store.GetAutomation(t.Context(), automation.ID)
	if err != nil || !found {
		t.Fatalf("paused automation found=%v err=%v", found, err)
	}
	if paused.Status != repoaudit.RepositoryReviewAutomationPaused ||
		paused.PauseReason != repoaudit.RepositoryReviewPauseNoProgress ||
		paused.Progress.CompletedBatches != 1 || paused.Progress.RemainingFiles != 1 ||
		paused.Progress.ReviewedFiles != 0 || paused.Progress.Findings != 1 ||
		paused.ActiveRunID != "" {
		t.Fatalf("projected progress bypassed no-progress pause: %#v", paused)
	}
	if _, active := controller.activeRunSnapshot(automation.ID, runID); active {
		t.Fatal("no-progress checkpoint admitted another batch")
	}
}

func TestRepositoryReviewFinishPausesAfterOneNoProgressCheckpoint(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	finish := func(
		t *testing.T,
		reason repoaudit.RepositoryReviewPauseReason,
	) repoaudit.RepositoryReviewAutomation {
		t.Helper()
		runID := "wr_no_progress_" + strings.ReplaceAll(string(reason), "_", "-")
		if reason == "" {
			runID = "wr_no_progress_none"
		}
		input := testRepositoryReviewAutomation()
		input.Repository = "owner/" + strings.TrimPrefix(runID, "wr_")
		input.Name = runID
		input.Status = repoaudit.RepositoryReviewAutomationRunning
		input.ActiveRunID = runID
		input.RunIDs = []string{runID}
		input.AutoContinue = true
		input.Progress = repoaudit.RepositoryReviewProgress{
			RemainingFiles: 2, TotalBatches: 1,
		}
		automation, createErr := store.CreateAutomation(t.Context(), input)
		if createErr != nil {
			t.Fatal(createErr)
		}
		controller := newRepositoryReviewController(handler)
		controller.active[automation.ID] = &repositoryReviewActiveRun{
			runID: runID, store: store, pauseReason: reason,
			pauseDetail: "Explicit pause wins.",
		}
		controller.finishAutomationRun(
			automation.ID,
			runID,
			&workflows.RunResult{
				Status:  workflows.RunStatusSucceeded,
				Outputs: map[string]any{"remainingFiles": 2, "reviewedFiles": 0},
			},
			nil,
			true,
			&workflows.Run{Steps: map[string]workflows.StepExecution{
				"find_bugs/record": {
					Status:  workflows.RunStatusSucceeded,
					Outputs: map[string]any{"run": map[string]any{"remaining_files": 2}},
				},
			}},
		)
		updated, found, getErr := store.GetAutomation(t.Context(), automation.ID)
		if getErr != nil || !found {
			t.Fatalf("updated automation found=%v err=%v", found, getErr)
		}
		return updated
	}

	paused := finish(t, "")
	if paused.Status != repoaudit.RepositoryReviewAutomationPaused ||
		paused.PauseReason != repoaudit.RepositoryReviewPauseNoProgress ||
		paused.Progress.CompletedBatches != 1 || paused.Progress.RemainingFiles != 2 ||
		!strings.Contains(paused.PauseDetail, "resolved zero files") {
		t.Fatalf("no-progress pause=%#v", paused)
	}
	for _, reason := range []repoaudit.RepositoryReviewPauseReason{
		repoaudit.RepositoryReviewPauseManual,
		repoaudit.RepositoryReviewPauseGuardExpression,
	} {
		paused = finish(t, reason)
		if paused.PauseReason != reason || paused.PauseDetail != "Explicit pause wins." {
			t.Fatalf("explicit %q pause lost precedence: %#v", reason, paused)
		}
	}
}

func TestRepositoryReviewCoverageProgressMonitor(t *testing.T) {
	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	runID := "wr_progress_coverage"
	input := testRepositoryReviewAutomation()
	input.Status = repoaudit.RepositoryReviewAutomationRunning
	input.ActiveRunID = runID
	input.RunIDs = []string{runID}
	input.Progress.TotalBatches = 1
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	workflowStore := workflows.NewFileRunStore(workspace)
	if err := workflowStore.CreateRun(t.Context(), &workflows.Run{
		ID: runID, WorkflowRef: workflows.RepositoryBugFinderWorkflowRef, Status: workflows.RunStatusRunning,
		Steps: map[string]workflows.StepExecution{
			"find_bugs/review": {ID: "review", Status: workflows.RunStatusRunning},
		},
	}); err != nil {
		t.Fatal(err)
	}

	monitorCtx, cancelMonitor := context.WithCancel(t.Context())
	done := make(chan struct{})
	controller := newRepositoryReviewController(handler)
	go func() {
		controller.monitorWorkflowProgress(monitorCtx, store, workflowStore, automation.ID, runID)
		close(done)
	}()
	deadline := time.Now().Add(2500 * time.Millisecond)
	for time.Now().Before(deadline) {
		current, found, getErr := store.GetAutomation(t.Context(), automation.ID)
		if getErr != nil {
			cancelMonitor()
			t.Fatal(getErr)
		}
		if found && current.Progress.Stage == "Reviewing bounded file batch" {
			cancelMonitor()
			<-done
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	cancelMonitor()
	<-done
	t.Fatal("progress monitor did not project the running review step")
}

func repositoryReviewCoverageMutation(
	t *testing.T,
	mux *http.ServeMux,
	method string,
	path string,
	body map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, path, strings.NewReader(string(data)))
	setRepositoryReviewMutationHeaders(request)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func repositoryReviewCoverageRawRequest(
	t *testing.T,
	mux *http.ServeMux,
	method string,
	path string,
	body string,
	validMutationHeaders bool,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if validMutationHeaders {
		setRepositoryReviewMutationHeaders(request)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func TestRepositoryReviewCoverageHandlerRequestFailures(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)

	mutations := []struct {
		method          string
		path            string
		malformedStatus int
	}{
		{method: http.MethodPost, path: "/api/repository-reviews/automations", malformedStatus: http.StatusBadRequest},
		{
			method:          http.MethodPatch,
			path:            "/api/repository-reviews/automations/rra_missing",
			malformedStatus: http.StatusBadRequest,
		},
		{
			method:          http.MethodDelete,
			path:            "/api/repository-reviews/automations/rra_missing",
			malformedStatus: http.StatusBadRequest,
		},
		{
			method:          http.MethodPost,
			path:            "/api/repository-reviews/automations/rra_missing/start",
			malformedStatus: http.StatusBadRequest,
		},
		{
			method:          http.MethodPost,
			path:            "/api/repository-reviews/automations/rra_missing/pause",
			malformedStatus: http.StatusBadRequest,
		},
		{
			method:          http.MethodPatch,
			path:            "/api/repository-reviews/" + state.ID + "/findings/missing",
			malformedStatus: http.StatusBadRequest,
		},
		{
			method:          http.MethodPost,
			path:            "/api/repository-reviews/" + state.ID + "/issue-drafts",
			malformedStatus: http.StatusBadRequest,
		},
		{
			method:          http.MethodPatch,
			path:            "/api/repository-reviews/" + state.ID + "/issue-drafts/missing",
			malformedStatus: http.StatusBadRequest,
		},
	}
	for _, mutation := range mutations {
		withoutHeaders := repositoryReviewCoverageRawRequest(
			t, mux, mutation.method, mutation.path, `{}`, false,
		)
		if withoutHeaders.Code != http.StatusBadRequest {
			t.Fatalf(
				"%s %s without headers = %d %s",
				mutation.method,
				mutation.path,
				withoutHeaders.Code,
				withoutHeaders.Body.String(),
			)
		}
		malformed := repositoryReviewCoverageRawRequest(
			t, mux, mutation.method, mutation.path, `{`, true,
		)
		if malformed.Code != mutation.malformedStatus {
			t.Fatalf("%s %s malformed = %d %s", mutation.method, mutation.path, malformed.Code, malformed.Body.String())
		}
	}

	invalidCreate := repositoryReviewCoverageMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/automations",
		map[string]any{"repository": "", "reviewer_models": []string{}},
	)
	if invalidCreate.Code != http.StatusBadRequest {
		t.Fatalf("invalid create = %d %s", invalidCreate.Code, invalidCreate.Body.String())
	}

	missingFinding := repositoryReviewCoverageMutation(
		t,
		mux,
		http.MethodPatch,
		"/api/repository-reviews/"+state.ID+"/findings/missing",
		map[string]any{"status": "dismissed", "expected_version": state.Version},
	)
	if missingFinding.Code != http.StatusNotFound {
		t.Fatalf("missing finding = %d %s", missingFinding.Code, missingFinding.Body.String())
	}
	missingIssueFinding := repositoryReviewCoverageMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/"+state.ID+"/issue-drafts",
		map[string]any{"finding_ids": []string{"missing"}, "expected_version": state.Version},
	)
	if missingIssueFinding.Code != http.StatusNotFound {
		t.Fatalf("missing issue finding = %d %s", missingIssueFinding.Code, missingIssueFinding.Body.String())
	}
	missingDraft := repositoryReviewCoverageMutation(
		t,
		mux,
		http.MethodPatch,
		"/api/repository-reviews/"+state.ID+"/issue-drafts/missing",
		map[string]any{"title": "title", "body": "body", "expected_version": 1},
	)
	if missingDraft.Code != http.StatusNotFound {
		t.Fatalf("missing draft = %d %s", missingDraft.Code, missingDraft.Body.String())
	}
}

func TestRepositoryReviewCoverageMissingConfigurationHandlers(t *testing.T) {
	handler := NewHandler(t.TempDir())
	t.Cleanup(handler.Shutdown)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	automation := testRepositoryReviewAutomation()
	configBody := automationConfigBody(automation)
	configBody["profile_id"] = "rrpf_missing"
	configBody["expected_version"] = 1

	requests := []struct {
		method string
		path   string
		body   map[string]any
	}{
		{method: http.MethodPost, path: "/api/repository-reviews/automations", body: configBody},
		{method: http.MethodPatch, path: "/api/repository-reviews/automations/rra_missing", body: configBody},
		{
			method: http.MethodDelete,
			path:   "/api/repository-reviews/automations/rra_missing",
			body:   map[string]any{"expected_version": 1},
		},
		{
			method: http.MethodPost,
			path:   "/api/repository-reviews/automations/rra_missing/start",
			body:   map[string]any{"expected_version": 1},
		},
		{
			method: http.MethodPost,
			path:   "/api/repository-reviews/automations/rra_missing/pause",
			body:   map[string]any{"expected_version": 1},
		},
		{
			method: http.MethodPatch,
			path:   "/api/repository-reviews/rrp_missing/findings/missing",
			body:   map[string]any{"status": "dismissed", "expected_version": 1},
		},
		{
			method: http.MethodPost,
			path:   "/api/repository-reviews/rrp_missing/issue-drafts",
			body:   map[string]any{"finding_ids": []string{"missing"}, "expected_version": 1},
		},
		{
			method: http.MethodPatch,
			path:   "/api/repository-reviews/rrp_missing/issue-drafts/missing",
			body:   map[string]any{"title": "title", "body": "body", "expected_version": 1},
		},
	}
	for _, request := range requests {
		response := repositoryReviewCoverageMutation(t, mux, request.method, request.path, request.body)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("%s %s = %d %s", request.method, request.path, response.Code, response.Body.String())
		}
	}

	for _, path := range []string{
		"/api/repository-reviews",
		"/api/repository-reviews/rrp_" + strings.Repeat("a", 64),
	} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("GET %s = %d %s", path, response.Code, response.Body.String())
		}
	}
}

func TestRepositoryReviewCoveragePagingProjectionAndDecodeBoundaries(t *testing.T) {
	if _, pageErr := repositoryReviewPage(nil); pageErr == nil {
		t.Fatal("nil page request was accepted")
	}
	for _, target := range []string{
		"/api/repository-reviews/id?offset=1&limit=2&draft_offset=3&draft_limit=4&extra=5",
		"/api/repository-reviews/id?unknown=1",
		"/api/repository-reviews/id?offset=1&offset=2",
		"/api/repository-reviews/id?offset=nope",
		"/api/repository-reviews/id?limit=201",
		"/api/repository-reviews/id?draft_offset=-1",
		"/api/repository-reviews/id?draft_limit=21",
	} {
		if _, pageErr := repositoryReviewPage(httptest.NewRequest(http.MethodGet, target, nil)); pageErr == nil {
			t.Fatalf("invalid page %q was accepted", target)
		}
	}
	if value, integerErr := repositoryReviewPageInteger("", 7, 10); integerErr != nil || value != 7 {
		t.Fatalf("page fallback=%d err=%v", value, integerErr)
	}

	state := repoaudit.RepositoryState{
		Findings: []repoaudit.Finding{
			{ID: "one", ContextIDs: []string{"ctx-one"}},
			{ID: "two", ContextIDs: []string{"ctx-two"}},
			{ID: "three", ContextIDs: []string{"ctx-three"}},
		},
		Contexts:    []repoaudit.FindingContext{{ID: "ctx-one"}, {ID: "ctx-two"}, {ID: "ctx-three"}},
		Files:       map[string]repoaudit.ReviewedFile{"private": {}},
		Unsupported: make(map[string]repoaudit.UnsupportedFile),
		Runs:        make([]repoaudit.ReviewRun, 51),
		IssueDrafts: []repoaudit.IssueDraft{{ID: "old"}, {ID: "middle"}, {ID: "new"}},
	}
	for index := 0; index < 205; index++ {
		path := "path-" + strconv.Itoa(index)
		state.Unsupported[path] = repoaudit.UnsupportedFile{FileRef: repoaudit.FileRef{Path: path}}
	}
	projected := projectRepositoryReviewDetail(state, repositoryReviewPageRequest{
		FindingOffset: 0, FindingLimit: 1, DraftOffset: 0, DraftLimit: 1,
	})
	if len(projected.Findings) != 1 || len(projected.Contexts) != 1 ||
		len(projected.Unsupported) != 200 || len(projected.Runs) != 50 || len(projected.IssueDrafts) != 1 ||
		projected.NextFindingOffset == nil || projected.NextDraftOffset == nil {
		t.Fatalf("projected detail=%#v", projected)
	}
	clamped := projectRepositoryReviewDetail(state, repositoryReviewPageRequest{
		FindingOffset: 99, FindingLimit: 1, DraftOffset: 99, DraftLimit: 1,
	})
	if clamped.FindingOffset != len(state.Findings) || len(clamped.Findings) != 0 || len(clamped.IssueDrafts) != 0 {
		t.Fatalf("clamped detail=%#v", clamped)
	}

	if decodeErr := decodeRepositoryReviewRequest(nil, &map[string]any{}); decodeErr == nil {
		t.Fatal("nil decode request was accepted")
	}
	trailing := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{} {}`))
	if decodeErr := decodeRepositoryReviewRequest(trailing, &map[string]any{}); decodeErr == nil {
		t.Fatal("trailing JSON was accepted")
	}
	if mutationErr := validateRepositoryReviewMutation(nil); mutationErr == nil {
		t.Fatal("nil mutation was accepted")
	}
}

func TestRepositoryReviewCoverageErrorProjection(t *testing.T) {
	for _, test := range []struct {
		err    error
		status int
	}{
		{err: os.ErrNotExist, status: http.StatusNotFound},
		{err: repoaudit.ErrRepositoryReviewPurgeInProgress, status: http.StatusConflict},
		{err: repoaudit.ErrConflict, status: http.StatusConflict},
		{err: repoaudit.ErrInvalidPlan, status: http.StatusBadRequest},
		{err: errors.New("duplicate input"), status: http.StatusBadRequest},
		{err: errors.New("disk failed"), status: http.StatusInternalServerError},
	} {
		response := httptest.NewRecorder()
		writeRepositoryReviewError(response, test.err)
		if response.Code != test.status {
			t.Fatalf("review error %v = %d", test.err, response.Code)
		}
	}
	for _, test := range []struct {
		err    error
		status int
	}{
		{err: os.ErrNotExist, status: http.StatusNotFound},
		{err: repoaudit.ErrRepositoryReviewPurgeInProgress, status: http.StatusConflict},
		{err: repoaudit.ErrHistoricalDeduplicationRestartRequired, status: http.StatusConflict},
		{err: errRepositoryReviewAutomationBusy, status: http.StatusConflict},
		{err: repoaudit.ErrInvalidAutomation, status: http.StatusBadRequest},
		{err: io.ErrUnexpectedEOF, status: http.StatusBadRequest},
		{err: &json.UnmarshalTypeError{Value: "string", Type: reflect.TypeOf(1)}, status: http.StatusBadRequest},
		{err: errors.New("disk failed"), status: http.StatusInternalServerError},
	} {
		response := httptest.NewRecorder()
		writeRepositoryReviewAutomationError(response, test.err)
		if response.Code != test.status {
			t.Fatalf("automation error %v = %d", test.err, response.Code)
		}
	}
}

func TestRepositoryReviewCoverageAutomationTransitionsAndUtilities(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, storeErr := handler.repositoryReviewStore()
	if storeErr != nil {
		t.Fatal(storeErr)
	}

	runningInput := testRepositoryReviewAutomation()
	runningInput.Status = repoaudit.RepositoryReviewAutomationRunning
	runningInput.ActiveRunID = "run-busy-update"
	runningInput.RunIDs = []string{runningInput.ActiveRunID}
	running, createErr := store.CreateAutomation(t.Context(), runningInput)
	if createErr != nil {
		t.Fatal(createErr)
	}
	busyBody := automationConfigBody(running)
	busyBody["expected_version"] = running.Version
	busy := repositoryReviewCoverageMutation(
		t, mux, http.MethodPatch, "/api/repository-reviews/automations/"+running.ID, busyBody,
	)
	if busy.Code != http.StatusConflict {
		t.Fatalf("busy update = %d %s", busy.Code, busy.Body.String())
	}

	idleInput := testRepositoryReviewAutomation()
	idleInput.AccountLimitSnapshots = []repoaudit.RepositoryReviewAccountLimitSnapshot{{
		AccountID: "account-a", Window: "weekly", CheckedAt: time.Now().UTC(),
	}}
	idle, createErr := store.CreateAutomation(t.Context(), idleInput)
	if createErr != nil {
		t.Fatal(createErr)
	}
	quotaBody := automationConfigBody(idle)
	quotaBody["expected_version"] = idle.Version
	budget := quotaBody["budget"].(repoaudit.RepositoryReviewBudgetPolicy)
	budget.GuardExpression = "account.limits.weekly.known"
	quotaBody["budget"] = budget
	quotaUpdate := repositoryReviewCoverageMutation(
		t, mux, http.MethodPatch, "/api/repository-reviews/automations/"+idle.ID, quotaBody,
	)
	if quotaUpdate.Code != http.StatusOK {
		t.Fatalf("quota-only update = %d %s", quotaUpdate.Code, quotaUpdate.Body.String())
	}
	var quotaResult struct {
		Automation repoaudit.RepositoryReviewAutomation `json:"automation"`
	}
	if decodeErr := json.Unmarshal(quotaUpdate.Body.Bytes(), &quotaResult); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if len(quotaResult.Automation.AccountLimitSnapshots) != 0 {
		t.Fatalf("quota-only automation=%#v", quotaResult.Automation)
	}

	staleDelete := repositoryReviewCoverageMutation(
		t,
		mux,
		http.MethodDelete,
		"/api/repository-reviews/automations/"+idle.ID,
		map[string]any{
			"expected_version":            idle.Version + 10,
			"expected_repository_version": 0,
			"expected_ledger_fence": repositoryReviewPurgeFenceForTest(
				t, workspace, quotaResult.Automation,
			),
			"confirm_repository": idle.Repository,
		},
	)
	if staleDelete.Code != http.StatusConflict {
		t.Fatalf("stale delete = %d %s", staleDelete.Code, staleDelete.Body.String())
	}
	for _, action := range []string{"start", "resume", "restart", "pause"} {
		missing := repositoryReviewCoverageMutation(
			t,
			mux,
			http.MethodPost,
			"/api/repository-reviews/automations/rra_missing/"+action,
			map[string]any{"expected_version": 1},
		)
		if missing.Code != http.StatusNotFound {
			t.Fatalf("missing %s = %d %s", action, missing.Code, missing.Body.String())
		}
	}

	var nilHandler *Handler
	for _, action := range []string{"start", "pause"} {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"expected_version":1}`))
		request.SetPathValue("automation_id", "rra_missing")
		setRepositoryReviewMutationHeaders(request)
		response := httptest.NewRecorder()
		if action == "start" {
			nilHandler.handleRepositoryReviewAutomationStartAction(response, request, false, false)
		} else {
			nilHandler.handlePauseRepositoryReviewAutomation(response, request)
		}
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("nil handler %s = %d %s", action, response.Code, response.Body.String())
		}
	}

	applyRepositoryReviewAutomationRequest(nil, repositoryReviewAutomationConfigRequest{})
	if slicesEqual([]string{"a"}, []string{"b"}) || slicesEqual([]string{"a"}, nil) ||
		!slicesEqual([]string{"a"}, []string{"a"}) {
		t.Fatal("slice equality mismatch")
	}
	if len(repositoryReviewModelOptions(nil)) != 0 ||
		repositoryReviewAliasAvailableForRuntime(nil, config.ModelAliasConfig{Name: "alias"}) ||
		repositoryReviewAliasAvailableForRuntime(config.DefaultConfig(), config.ModelAliasConfig{}) {
		t.Fatal("nil model option boundary mismatch")
	}
	if repositoryReviewRuntimeAccountRefs(nil) != nil {
		t.Fatal("nil runtime account refs were non-nil")
	}
	if _, ok := repositoryReviewAliasPrice(nil, "alias", map[string]bool{}); ok {
		t.Fatal("nil alias pricing succeeded")
	}
	if _, ok := repositoryReviewAliasPrice(config.DefaultConfig(), "", map[string]bool{}); ok {
		t.Fatal("empty alias pricing succeeded")
	}
	if _, ok := repositoryReviewAliasPrice(config.DefaultConfig(), "alias", map[string]bool{"alias": true}); ok {
		t.Fatal("cyclic alias pricing succeeded")
	}

	routerConfig := config.DefaultConfig()
	routerConfig.Agents.Defaults.AccountRef = "review-router"
	routerConfig.AccountRouters = []config.AccountRouterConfig{{
		Name: "review-router", Enabled: true, Entry: "entry",
		Blocks: []config.AccountRouterBlock{
			{
				ID: "entry", Type: config.AccountRouterBlockTypeAccount,
				Account: " account-a ", Fallback: "pool",
			},
			{
				ID: "pool", Type: config.AccountRouterBlockTypeLoadBalance,
				Accounts: []string{"account-b", "account-a", ""},
			},
		},
	}}
	if refs := repositoryReviewRuntimeAccountRefs(
		routerConfig,
	); !reflect.DeepEqual(
		refs,
		[]string{"account-a", "account-b"},
	) {
		t.Fatalf("router refs=%#v", refs)
	}

	usedAbove := 125
	usedBelow := -10
	accounts := repositoryReviewAccountOptions(nil, codexAccountLimitsResponse{Accounts: []codexAccountLimitAccount{
		{ID: "fallback-id", Entries: []codexAccountLimitEntry{{UsedPercent: &usedAbove}}},
		{ID: "account-id", AccountID: "account-label", Entries: []codexAccountLimitEntry{{UsedPercent: &usedBelow}}},
	}})
	accountMap := make(map[string]repositoryReviewAccountOption, len(accounts))
	for _, account := range accounts {
		accountMap[account.ID] = account
	}
	fallback := accountMap["credential:fallback-id"]
	labeled := accountMap["credential:account-id"]
	if !strings.Contains(fallback.Label, "fallback-id") || *fallback.Entries[0].RemainingPercent != 0 ||
		!strings.Contains(labeled.Label, "account-label") || *labeled.Entries[0].RemainingPercent != 100 {
		t.Fatalf("fallback accounts=%#v", accounts)
	}

	root := filepath.Join(workspace, "repository_reviews")
	if removeErr := os.RemoveAll(root); removeErr != nil {
		t.Fatal(removeErr)
	}
	if writeErr := os.WriteFile(root, []byte("not a directory"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	listFailure := httptest.NewRecorder()
	mux.ServeHTTP(listFailure, httptest.NewRequest(http.MethodGet, "/api/repository-reviews/automations", nil))
	if listFailure.Code != http.StatusInternalServerError {
		t.Fatalf("corrupt automation list = %d %s", listFailure.Code, listFailure.Body.String())
	}
}

func TestRepositoryReviewCoveragePublishAndCorruptStoreFailures(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)

	invalid := httptest.NewRecorder()
	handler.handlePublishRepositoryReviewIssue(invalid, nil)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("nil publish = %d %s", invalid.Code, invalid.Body.String())
	}
	wrongMediaRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"expected_version":1}`))
	setRepositoryReviewMutationHeaders(wrongMediaRequest)
	wrongMediaRequest.Header.Set("Content-Type", "text/plain")
	wrongMedia := httptest.NewRecorder()
	handler.handlePublishRepositoryReviewIssue(wrongMedia, wrongMediaRequest)
	if wrongMedia.Code != http.StatusBadRequest {
		t.Fatalf("wrong-media publish = %d %s", wrongMedia.Code, wrongMedia.Body.String())
	}
	nilBodyRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	setRepositoryReviewMutationHeaders(nilBodyRequest)
	nilBodyRequest.Body = nil
	nilBody := httptest.NewRecorder()
	handler.handlePublishRepositoryReviewIssue(nilBody, nilBodyRequest)
	if nilBody.Code != http.StatusBadRequest {
		t.Fatalf("nil-body publish = %d %s", nilBody.Code, nilBody.Body.String())
	}
	emptyRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	emptyRequest.SetPathValue("repository_id", state.ID)
	emptyRequest.SetPathValue("draft_id", "draft")
	setRepositoryReviewMutationHeaders(emptyRequest)
	empty := httptest.NewRecorder()
	handler.handlePublishRepositoryReviewIssue(empty, emptyRequest)
	if empty.Code != http.StatusBadRequest {
		t.Fatalf("empty publish = %d %s", empty.Code, empty.Body.String())
	}
	proxyRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"expected_version":1}`))
	proxyRequest.SetPathValue("repository_id", state.ID)
	proxyRequest.SetPathValue("draft_id", "draft")
	setRepositoryReviewMutationHeaders(proxyRequest)
	proxy := httptest.NewRecorder()
	handler.handlePublishRepositoryReviewIssue(proxy, proxyRequest)
	if proxy.Code < http.StatusBadRequest {
		t.Fatalf("unconfigured publish proxy = %d %s", proxy.Code, proxy.Body.String())
	}

	root := filepath.Join(workspace, "repository_reviews")
	if removeErr := os.RemoveAll(root); removeErr != nil {
		t.Fatal(removeErr)
	}
	if writeErr := os.WriteFile(root, []byte("not a directory"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	for _, path := range []string{
		"/api/repository-reviews",
		"/api/repository-reviews/" + state.ID,
	} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("corrupt GET %s = %d %s", path, response.Code, response.Body.String())
		}
	}
	mutation := repositoryReviewCoverageMutation(
		t,
		mux,
		http.MethodPatch,
		"/api/repository-reviews/"+state.ID+"/findings/"+state.Findings[0].ID,
		map[string]any{"status": "dismissed", "expected_version": state.Version},
	)
	if mutation.Code != http.StatusInternalServerError {
		t.Fatalf("corrupt mutation = %d %s", mutation.Code, mutation.Body.String())
	}
}

func repositoryReviewCoverageRunningAutomation(
	t *testing.T,
	store repoaudit.Store,
	runID string,
	autoContinue bool,
) repoaudit.RepositoryReviewAutomation {
	t.Helper()
	input := testRepositoryReviewAutomation()
	input.Status = repoaudit.RepositoryReviewAutomationRunning
	input.ActiveRunID = runID
	input.RunIDs = []string{runID}
	input.AutoContinue = autoContinue
	input.Progress.TotalBatches = 1
	created, createErr := store.CreateAutomation(t.Context(), input)
	if createErr != nil {
		t.Fatal(createErr)
	}
	return created
}

func TestRepositoryReviewCoverageExecuteAndFinishBoundaries(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, storeErr := handler.repositoryReviewStore()
	if storeErr != nil {
		t.Fatal(storeErr)
	}

	t.Run("missing active run", func(t *testing.T) {
		controller := newRepositoryReviewController(handler)
		controller.wg.Add(1)
		controller.executeAutomation("rra_missing", "run-missing")
	})
	t.Run("missing automation", func(t *testing.T) {
		controller := newRepositoryReviewController(handler)
		controller.active["rra_missing"] = &repositoryReviewActiveRun{runID: "run-missing", store: store}
		controller.wg.Add(1)
		controller.executeAutomation("rra_missing", "run-missing")
		if _, exists := controller.active["rra_missing"]; exists {
			t.Fatal("missing automation active run survived")
		}
	})
	t.Run("runtime config failure", func(t *testing.T) {
		automation := repositoryReviewCoverageRunningAutomation(t, store, "run-runtime-config", false)
		controller := newRepositoryReviewController(handler)
		controller.active[automation.ID] = &repositoryReviewActiveRun{
			runID: automation.ActiveRunID, store: store, config: nil,
		}
		controller.wg.Add(1)
		controller.executeAutomation(automation.ID, automation.ActiveRunID)
		updated, found, getErr := store.GetAutomation(t.Context(), automation.ID)
		if getErr != nil || !found || updated.Status != repoaudit.RepositoryReviewAutomationFailed {
			t.Fatalf("runtime failure automation=%#v found=%v err=%v", updated, found, getErr)
		}
	})
	t.Run("real workflow runtime failure", func(t *testing.T) {
		automation := repositoryReviewCoverageRunningAutomation(t, store, "run-real-runtime", false)
		cfg, loadErr := config.LoadConfig(handler.configPath)
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		controller := newRepositoryReviewController(handler)
		runCtx, cancelRun := context.WithTimeout(context.Background(), 3*time.Second)
		controller.ctx = runCtx
		controller.cancel = cancelRun
		controller.active[automation.ID] = &repositoryReviewActiveRun{
			runID: automation.ActiveRunID, store: store, config: cfg,
		}
		controller.wg.Add(1)
		controller.executeAutomation(automation.ID, automation.ActiveRunID)
		cancelRun()
		updated, found, getErr := store.GetAutomation(t.Context(), automation.ID)
		if getErr != nil || !found || updated.Status == repoaudit.RepositoryReviewAutomationRunning {
			t.Fatalf("real runtime automation=%#v found=%v err=%v", updated, found, getErr)
		}
	})
	t.Run("workflow parse failure", func(t *testing.T) {
		automation := repositoryReviewCoverageRunningAutomation(t, store, "run-parse", false)
		cfg, loadErr := config.LoadConfig(handler.configPath)
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		previousParse := repositoryReviewParseWorkflow
		repositoryReviewParseWorkflow = func([]byte) (*workflows.Workflow, error) {
			return nil, errors.New("injected workflow parse failure")
		}
		t.Cleanup(func() { repositoryReviewParseWorkflow = previousParse })
		controller := newRepositoryReviewController(handler)
		controller.active[automation.ID] = &repositoryReviewActiveRun{
			runID: automation.ActiveRunID, store: store, config: cfg,
		}
		controller.wg.Add(1)
		controller.executeAutomation(automation.ID, automation.ActiveRunID)
		repositoryReviewParseWorkflow = previousParse
		updated, found, getErr := store.GetAutomation(t.Context(), automation.ID)
		if getErr != nil || !found || updated.Status != repoaudit.RepositoryReviewAutomationFailed {
			t.Fatalf("parse failure automation=%#v found=%v err=%v", updated, found, getErr)
		}
	})

	finish := func(
		t *testing.T,
		runID string,
		status repoaudit.RepositoryReviewAutomationStatus,
		autoContinue bool,
		result *workflows.RunResult,
		runErr error,
		checkpointed bool,
		configureActive func(*repositoryReviewActiveRun),
	) repoaudit.RepositoryReviewAutomation {
		t.Helper()
		automation := repositoryReviewCoverageRunningAutomation(t, store, runID, autoContinue)
		if status == repoaudit.RepositoryReviewAutomationStopping {
			var updateErr error
			automation, updateErr = store.UpdateAutomation(
				t.Context(),
				automation.ID,
				automation.Version,
				func(candidate *repoaudit.RepositoryReviewAutomation) error {
					candidate.Status = repoaudit.RepositoryReviewAutomationStopping
					candidate.RequestedPauseReason = repoaudit.RepositoryReviewPauseManual
					return nil
				},
			)
			if updateErr != nil {
				t.Fatal(updateErr)
			}
		}
		controller := newRepositoryReviewController(handler)
		controller.runBatch = func(
			context.Context,
			repoaudit.RepositoryReviewAutomation,
			string,
			workflows.AgentUsageObserver,
		) (*workflows.RunResult, error) {
			return nil, errors.New("test-only runBatch seam")
		}
		active := &repositoryReviewActiveRun{runID: runID, store: store}
		if configureActive != nil {
			configureActive(active)
		}
		controller.active[automation.ID] = active
		if autoContinue {
			controller.stopped = true
		}
		controller.finishAutomationRun(automation.ID, runID, result, runErr, checkpointed, nil)
		updated, found, getErr := store.GetAutomation(t.Context(), automation.ID)
		if getErr != nil || !found {
			t.Fatalf("finished automation found=%v err=%v", found, getErr)
		}
		return updated
	}

	canceled := finish(
		t,
		"run-canceled",
		repoaudit.RepositoryReviewAutomationRunning,
		false,
		nil,
		context.Canceled,
		false,
		nil,
	)
	if canceled.Status != repoaudit.RepositoryReviewAutomationPaused ||
		canceled.PauseReason != repoaudit.RepositoryReviewPauseServiceRestart {
		t.Fatalf("canceled finish=%#v", canceled)
	}
	stopping := finish(
		t,
		"run-stopping",
		repoaudit.RepositoryReviewAutomationStopping,
		false,
		&workflows.RunResult{Status: workflows.RunStatusSucceeded},
		nil,
		true,
		nil,
	)
	if stopping.Status != repoaudit.RepositoryReviewAutomationPaused {
		t.Fatalf("stopping finish=%#v", stopping)
	}
	failed := finish(
		t,
		"run-nil-result",
		repoaudit.RepositoryReviewAutomationRunning,
		false,
		nil,
		nil,
		false,
		nil,
	)
	if failed.Status != repoaudit.RepositoryReviewAutomationFailed {
		t.Fatalf("nil-result finish=%#v", failed)
	}
	completed := finish(
		t,
		"run-complete",
		repoaudit.RepositoryReviewAutomationRunning,
		false,
		&workflows.RunResult{
			Status:  workflows.RunStatusSucceeded,
			Outputs: map[string]any{"remainingFiles": 0, "reviewedFiles": 1},
		},
		nil,
		true,
		nil,
	)
	if completed.Status != repoaudit.RepositoryReviewAutomationCompleted || completed.CompletedAt.IsZero() {
		t.Fatalf("completed finish=%#v", completed)
	}
	continued := finish(
		t,
		"run-continue",
		repoaudit.RepositoryReviewAutomationRunning,
		true,
		&workflows.RunResult{
			Status:  workflows.RunStatusSucceeded,
			Outputs: map[string]any{"remainingFiles": 2, "reviewedFiles": 1},
		},
		nil,
		true,
		nil,
	)
	if continued.Status != repoaudit.RepositoryReviewAutomationIdle {
		t.Fatalf("continued finish=%#v", continued)
	}
}

func TestRepositoryReviewExecutionReachesProviderAdmissionOnLocalRepository(t *testing.T) {
	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	repository := t.TempDir()
	runGit := func(arguments ...string) {
		t.Helper()
		command := exec.Command("git", arguments...)
		command.Dir = repository
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", arguments, err, output)
		}
	}
	runGit("init", "-b", "main")
	runGit("config", "user.email", "review@example.test")
	runGit("config", "user.name", "Repository Review Test")
	if err := os.WriteFile(
		filepath.Join(repository, "service.go"),
		[]byte("package service\n\nfunc Value() int { return 1 }\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	runGit("add", "service.go")
	runGit("commit", "-m", "fixture")

	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	input := testRepositoryReviewAutomation()
	input.Repository = repository
	input.Status = repoaudit.RepositoryReviewAutomationRunning
	input.ActiveRunID = "run-local-provider-admission"
	input.RunIDs = []string{input.ActiveRunID}
	input.Progress.TotalBatches = 1
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	providerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "provider unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(providerServer.Close)
	cfg.ModelList[0].APIBase = providerServer.URL + "/v1"
	cfg.ModelList[0].APIKeys = config.SecureStrings{config.NewSecureString("test-api-key")}
	controller := newRepositoryReviewController(handler)
	runContext, cancelRun := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancelRun()
	controller.ctx = runContext
	controller.active[automation.ID] = &repositoryReviewActiveRun{
		runID: automation.ActiveRunID, store: store, config: cfg,
		reservations: make(map[int]repositoryReviewTaskReservation),
		guardMu:      &sync.Mutex{},
	}
	controller.wg.Add(1)
	controller.executeAutomation(automation.ID, automation.ActiveRunID)

	run, err := workflows.NewFileRunStore(workspace).GetRun(t.Context(), automation.ActiveRunID)
	if err != nil || run == nil {
		t.Fatalf("workflow run=%#v err=%v", run, err)
	}
	reachedPlanner := false
	for stepID := range run.Steps {
		if strings.Contains(stepID, "plan_scope") {
			reachedPlanner = true
			break
		}
	}
	if !reachedPlanner {
		t.Fatalf("workflow did not reach provider-backed scope planning: %#v", run.Steps)
	}
}

func TestRepositoryReviewCommitResolverPinsBranchAndExactCommit(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	repository := t.TempDir()
	git := func(arguments ...string) string {
		t.Helper()
		command := exec.Command("git", arguments...)
		command.Dir = repository
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", arguments, err, output)
		}
		return strings.TrimSpace(string(output))
	}
	git("init", "-b", "main")
	git("config", "user.email", "review@example.test")
	git("config", "user.name", "Repository Review Test")
	if err := os.WriteFile(filepath.Join(repository, "first.go"), []byte("package first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git("add", "first.go")
	git("commit", "-m", "first")
	first := git("rev-parse", "HEAD")

	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.ID = "rra_resolver"
	automation.Repository = repository
	resolved, err := resolveRepositoryReviewAutomationCommit(t.Context(), cfg, automation, "")
	if err != nil || resolved != first {
		t.Fatalf("initial resolved commit = %q, want %q, err=%v", resolved, first, err)
	}

	if writeErr := os.WriteFile(
		filepath.Join(repository, "second.go"),
		[]byte("package second\n"),
		0o600,
	); writeErr != nil {
		t.Fatal(writeErr)
	}
	git("add", "second.go")
	git("commit", "-m", "second")
	latest := git("rev-parse", "HEAD")
	resolved, err = resolveRepositoryReviewAutomationCommit(t.Context(), cfg, automation, "")
	if err != nil || resolved != latest {
		t.Fatalf("latest resolved commit = %q, want %q, err=%v", resolved, latest, err)
	}
	resolved, err = resolveRepositoryReviewAutomationCommit(t.Context(), cfg, automation, first)
	if err != nil || resolved != first {
		t.Fatalf("exact resolved commit = %q, want %q, err=%v", resolved, first, err)
	}
	if _, err = resolveRepositoryReviewAutomationCommit(
		t.Context(), cfg, automation, strings.Repeat("f", 40),
	); err == nil {
		t.Fatal("unreachable exact commit resolved")
	}
	git("checkout", "-b", "feature/review")
	if writeErr := os.WriteFile(
		filepath.Join(repository, "feature.go"),
		[]byte("package feature\n"),
		0o600,
	); writeErr != nil {
		t.Fatal(writeErr)
	}
	git("add", "feature.go")
	git("commit", "-m", "feature")
	feature := git("rev-parse", "HEAD")
	git("checkout", "main")
	automation.Ref = "feature/review"
	resolved, err = resolveRepositoryReviewAutomationCommit(t.Context(), cfg, automation, "")
	if err != nil || resolved != feature {
		t.Fatalf("feature resolved commit = %q, want %q, err=%v", resolved, feature, err)
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		revParse   string
		wantPhrase string
	}{
		{name: "rev parse fails", revParse: "exit 9", wantPhrase: "commit ID"},
		{name: "rev parse is invalid", revParse: "printf 'not-a-commit\\n'", wantPhrase: "noncanonical"},
	} {
		t.Run(test.name, func(t *testing.T) {
			wrapperRoot := t.TempDir()
			wrapper := filepath.Join(wrapperRoot, "git")
			script := "#!/bin/sh\nif [ \"$1\" = \"rev-parse\" ]; then " + test.revParse +
				"; fi\nexec \"" + realGit + "\" \"$@\"\n"
			if writeErr := os.WriteFile(wrapper, []byte(script), 0o700); writeErr != nil {
				t.Fatal(writeErr)
			}
			t.Setenv("PATH", wrapperRoot+string(os.PathListSeparator)+os.Getenv("PATH"))
			candidate := automation
			candidate.ID += "_" + strings.ReplaceAll(test.name, " ", "_")
			resolverConfig := *cfg
			resolverConfig.GitWorkspaces.RootDir = t.TempDir()
			_, resolveErr := resolveRepositoryReviewAutomationCommit(
				t.Context(), &resolverConfig, candidate, feature,
			)
			if resolveErr == nil || !strings.Contains(resolveErr.Error(), test.wantPhrase) {
				t.Fatalf("wrapped resolver error=%v", resolveErr)
			}
		})
	}
}

func TestRepositoryReviewCommitResolutionBoundaryCoverage(t *testing.T) {
	if _, err := resolveRepositoryReviewAutomationCommit(
		t.Context(), nil, repoaudit.RepositoryReviewAutomation{}, "",
	); err == nil {
		t.Fatal("nil configuration resolved a commit")
	}
	invalidRootConfig := config.DefaultConfig()
	invalidRootConfig.Agents.Defaults.Workspace = t.TempDir()
	invalidRoot := filepath.Join(t.TempDir(), "root-file")
	if writeErr := os.WriteFile(invalidRoot, nil, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	invalidRootConfig.GitWorkspaces.RootDir = invalidRoot
	if _, err := resolveRepositoryReviewAutomationCommit(
		t.Context(), invalidRootConfig, testRepositoryReviewAutomation(), "",
	); err == nil {
		t.Fatal("invalid git workspace root initialized")
	}
	validConfig := config.DefaultConfig()
	validConfig.Agents.Defaults.Workspace = t.TempDir()
	validConfig.GitWorkspaces.RootDir = t.TempDir()
	invalidRepository := testRepositoryReviewAutomation()
	invalidRepository.ID = "rra_invalid_repository"
	invalidRepository.Repository = "https://127.0.0.1:1/unavailable.git"
	if _, err := resolveRepositoryReviewAutomationCommit(
		t.Context(), validConfig, invalidRepository, "",
	); err == nil {
		t.Fatal("unavailable repository resolved")
	}

	if _, err := repositoryReviewGitOutput(
		t.Context(), "", 16, "repository-review-command-that-does-not-exist",
	); err == nil {
		t.Fatal("missing command succeeded")
	}
	if _, err := repositoryReviewGitOutput(
		t.Context(), "", 16, "sh", "-c", "exit 7",
	); err == nil {
		t.Fatal("failed command succeeded")
	}
	if _, err := repositoryReviewGitOutput(
		t.Context(), "", 2, "sh", "-c", "printf 12345",
	); err == nil {
		t.Fatal("oversized command output succeeded")
	}
	previousCommandContext := repositoryReviewCommandContext
	previousReadAll := repositoryReviewReadAll
	t.Cleanup(func() {
		repositoryReviewCommandContext = previousCommandContext
		repositoryReviewReadAll = previousReadAll
	})
	repositoryReviewCommandContext = func(context.Context, string, ...string) *exec.Cmd {
		command := exec.Command("sh", "-c", "printf output")
		command.Stdout = io.Discard
		return command
	}
	if _, err := repositoryReviewGitOutput(t.Context(), "", 16, "ignored"); err == nil {
		t.Fatal("command with an occupied stdout pipe succeeded")
	}
	repositoryReviewCommandContext = previousCommandContext
	repositoryReviewReadAll = func(io.Reader) ([]byte, error) {
		return nil, errors.New("injected stdout read failure")
	}
	if _, err := repositoryReviewGitOutput(
		t.Context(), "", 16, "sh", "-c", "printf output",
	); err == nil {
		t.Fatal("stdout read failure was ignored")
	}
	repositoryReviewReadAll = func(reader io.Reader) ([]byte, error) {
		limited, ok := reader.(*io.LimitedReader)
		if !ok {
			t.Fatal("stdout reader is not limited")
		}
		closer, ok := limited.R.(io.Closer)
		if !ok {
			t.Fatal("limited stdout reader is not closable")
		}
		if err := closer.Close(); err != nil {
			t.Fatal(err)
		}
		return []byte{}, nil
	}
	if _, err := repositoryReviewGitOutput(
		t.Context(), "", 16, "sh", "-c", "printf output",
	); err == nil {
		t.Fatal("stdout drain failure was ignored")
	}
	repositoryReviewReadAll = previousReadAll

	commit := strings.Repeat("a", 40)
	automation := repoaudit.RepositoryReviewAutomation{
		ScopePlan: repoaudit.RepositoryReviewScopePlan{CommitSHA: commit},
	}
	if got := repositoryReviewRememberedCommit(automation); got != commit {
		t.Fatalf("scope-plan commit=%q", got)
	}
	if got := repositoryReviewRememberedCommit(repoaudit.RepositoryReviewAutomation{}); got != "" {
		t.Fatalf("empty remembered commit=%q", got)
	}
	if repositoryReviewValidCommitSHA(strings.Repeat("g", 40)) {
		t.Fatal("non-hex commit accepted")
	}
	if got := repositoryReviewExecutionRef(
		repoaudit.RepositoryReviewAutomation{Ref: "main"},
	); got != "refs/heads/main" {
		t.Fatalf("fallback execution ref=%q", got)
	}
	if got := repositoryReviewExecutionRef(repoaudit.RepositoryReviewAutomation{
		ResolvedCommitSHA: commit,
	}); got != commit {
		t.Fatalf("remembered execution ref=%q", got)
	}
}

func TestRepositoryReviewAdmissionCommitBoundaryCoverage(t *testing.T) {
	commitA := strings.Repeat("a", 40)
	commitB := strings.Repeat("b", 40)
	cfg := config.DefaultConfig()
	automation := testRepositoryReviewAutomation()
	automation.ResolvedCommitSHA = commitA

	if _, err := (*repositoryReviewController)(nil).resolveRepositoryReviewAdmissionCommit(
		t.Context(), cfg, automation, "start", "",
	); err == nil {
		t.Fatal("nil resolver admitted a commit")
	}
	controller := &repositoryReviewController{}
	if _, err := controller.resolveRepositoryReviewAdmissionCommit(
		t.Context(), cfg, automation, "resume", "short",
	); err == nil {
		t.Fatal("short commit selection admitted")
	}
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		return "", errors.New("resolve failed")
	}
	if _, err := controller.resolveRepositoryReviewAdmissionCommit(
		t.Context(), cfg, automation, "resume", "",
	); err == nil {
		t.Fatal("resume resolver failure admitted")
	}
	if _, err := controller.resolveRepositoryReviewAdmissionCommit(
		t.Context(), cfg, automation, "resume", commitB,
	); !errors.Is(err, repoaudit.ErrInvalidAutomation) {
		t.Fatalf("selected resolver failure=%v", err)
	}
	if _, err := controller.resolveRepositoryReviewAdmissionCommit(
		t.Context(), cfg, automation, "start", "",
	); err == nil {
		t.Fatal("start resolver failure admitted")
	}
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		return "not-a-commit", nil
	}
	if _, err := controller.resolveRepositoryReviewAdmissionCommit(
		t.Context(), cfg, automation, "resume", "",
	); err == nil {
		t.Fatal("invalid latest commit admitted")
	}
	if _, err := controller.resolveRepositoryReviewAdmissionCommit(
		t.Context(), cfg, automation, "start", "",
	); err == nil {
		t.Fatal("invalid start commit admitted")
	}
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		return commitB, nil
	}
	if _, err := controller.resolveRepositoryReviewAdmissionCommit(
		t.Context(), cfg, automation, "resume", "",
	); !errors.Is(err, errRepositoryReviewCommitSelection) {
		t.Fatalf("moved resume error=%v", err)
	}
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		return commitA, nil
	}
	if got, err := controller.resolveRepositoryReviewAdmissionCommit(
		t.Context(), cfg, automation, "resume", "",
	); err != nil || got != commitA {
		t.Fatalf("same resume commit=%q err=%v", got, err)
	}
	withoutRemembered := automation
	withoutRemembered.ResolvedCommitSHA = ""
	if got, err := controller.resolveRepositoryReviewAdmissionCommit(
		t.Context(), cfg, withoutRemembered, "resume", "",
	); err != nil || got != commitA {
		t.Fatalf("new resume commit=%q err=%v", got, err)
	}
	queued := automation
	queued.Progress.Stage = "next batch queued"
	if got, err := controller.resolveRepositoryReviewAdmissionCommit(
		t.Context(), cfg, queued, "start", "",
	); err != nil || got != commitA {
		t.Fatalf("queued commit=%q err=%v", got, err)
	}
}

func TestRepositoryReviewCommitOptionsBoundaryCoverage(t *testing.T) {
	commit := strings.Repeat("a", 40)
	response := httptest.NewRecorder()
	var nilHandler *Handler
	nilHandler.handleRepositoryReviewAutomationCommitOptions(
		response,
		httptest.NewRequest(
			http.MethodGet,
			"/api/repository-reviews/automations/rra_missing/commit-options",
			nil,
		),
	)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("nil handler commit options status=%d", response.Code)
	}
	setup := func(
		t *testing.T,
		status repoaudit.RepositoryReviewAutomationStatus,
		remembered string,
	) (*Handler, repoaudit.Store, repoaudit.RepositoryReviewAutomation, *repositoryReviewController, string) {
		t.Helper()
		handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		store, err := handler.repositoryReviewStore()
		if err != nil {
			t.Fatal(err)
		}
		automation := testRepositoryReviewAutomation()
		testID := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
		automation.ID = "rra_commit_options_" + strings.ToLower(testID)
		automation.Status = status
		automation.ResolvedCommitSHA = remembered
		if status == repoaudit.RepositoryReviewAutomationPaused {
			automation.PauseReason = repoaudit.RepositoryReviewPauseManual
			automation.PauseDetail = "paused"
		}
		automation, err = store.CreateAutomation(t.Context(), automation)
		if err != nil {
			t.Fatal(err)
		}
		controller := repositoryReviewCoverageLeasedController(t, handler, store)
		controller.resolveCommit = func(
			context.Context,
			*config.Config,
			repoaudit.RepositoryReviewAutomation,
			string,
		) (string, error) {
			return commit, nil
		}
		t.Cleanup(controller.cancel)
		return handler, store, automation, controller, workspace
	}

	if _, _, _, err := (*repositoryReviewController)(nil).repositoryReviewCommitOptions(
		t.Context(), "rra_missing",
	); err == nil {
		t.Fatal("nil commit-options controller succeeded")
	}
	if _, _, _, err := newRepositoryReviewController(nil).repositoryReviewCommitOptions(
		t.Context(), "rra_missing",
	); err == nil {
		t.Fatal("unavailable commit-options controller succeeded")
	}

	t.Run("stopped", func(t *testing.T) {
		_, _, automation, controller, _ := setup(
			t, repoaudit.RepositoryReviewAutomationPaused, commit,
		)
		controller.cancel()
		if _, _, _, err := controller.repositoryReviewCommitOptions(
			t.Context(), automation.ID,
		); !errors.Is(err, context.Canceled) {
			t.Fatalf("stopped options error=%v", err)
		}
	})
	t.Run("missing", func(t *testing.T) {
		handler, _, _, _, _ := setup(
			t, repoaudit.RepositoryReviewAutomationPaused, commit,
		)
		store, err := handler.repositoryReviewStore()
		if err != nil {
			t.Fatal(err)
		}
		controller := repositoryReviewCoverageLeasedController(t, handler, store)
		controller.resolveCommit = func(
			context.Context,
			*config.Config,
			repoaudit.RepositoryReviewAutomation,
			string,
		) (string, error) {
			return commit, nil
		}
		if _, _, _, err := controller.repositoryReviewCommitOptions(
			t.Context(), "rra_not_found",
		); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("missing options error=%v", err)
		}
	})
	t.Run("invalid status", func(t *testing.T) {
		_, _, automation, controller, _ := setup(
			t, repoaudit.RepositoryReviewAutomationIdle, "",
		)
		if _, _, _, err := controller.repositoryReviewCommitOptions(
			t.Context(), automation.ID,
		); !errors.Is(err, errRepositoryReviewInvalidTransition) {
			t.Fatalf("idle options error=%v", err)
		}
	})
	t.Run("invalid legacy branch", func(t *testing.T) {
		_, store, automation, controller, _ := setup(
			t, repoaudit.RepositoryReviewAutomationPaused, commit,
		)
		automation, err := store.UpdateAutomation(
			t.Context(), automation.ID, automation.Version,
			func(candidate *repoaudit.RepositoryReviewAutomation) error {
				candidate.Ref = "not a branch"
				return nil
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := controller.repositoryReviewCommitOptions(
			t.Context(), automation.ID,
		); err == nil {
			t.Fatal("invalid legacy branch returned no error")
		}
	})
	t.Run("resolver errors", func(t *testing.T) {
		_, _, automation, controller, _ := setup(
			t, repoaudit.RepositoryReviewAutomationPaused, commit,
		)
		controller.resolveCommit = func(
			context.Context,
			*config.Config,
			repoaudit.RepositoryReviewAutomation,
			string,
		) (string, error) {
			return "", errors.New("resolver failed")
		}
		if _, _, _, err := controller.repositoryReviewCommitOptions(
			t.Context(), automation.ID,
		); err == nil {
			t.Fatal("resolver failure returned no error")
		}
		controller.resolveCommit = func(
			context.Context,
			*config.Config,
			repoaudit.RepositoryReviewAutomation,
			string,
		) (string, error) {
			return "invalid", nil
		}
		if _, _, _, err := controller.repositoryReviewCommitOptions(
			t.Context(), automation.ID,
		); err == nil {
			t.Fatal("invalid resolver commit returned no error")
		}
	})
	t.Run("adopts latest", func(t *testing.T) {
		_, _, automation, controller, _ := setup(
			t, repoaudit.RepositoryReviewAutomationPaused, "",
		)
		_, remembered, latest, err := controller.repositoryReviewCommitOptions(
			t.Context(), automation.ID,
		)
		if err != nil || remembered != commit || latest != commit {
			t.Fatalf("adopted options remembered=%q latest=%q err=%v", remembered, latest, err)
		}
	})
	t.Run("configuration changes", func(t *testing.T) {
		handler, _, automation, controller, _ := setup(
			t, repoaudit.RepositoryReviewAutomationPaused, commit,
		)
		if writeErr := os.WriteFile(handler.configPath, []byte("{"), 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
		if _, _, _, err := controller.repositoryReviewCommitOptions(
			t.Context(), automation.ID,
		); err == nil {
			t.Fatal("corrupt current configuration returned no error")
		}
	})
	t.Run("state corrupts during resolution", func(t *testing.T) {
		_, _, automation, controller, workspace := setup(
			t, repoaudit.RepositoryReviewAutomationPaused, commit,
		)
		statePath := filepath.Join(
			workspace,
			"repository_reviews",
			"automation_"+automation.ID+".json",
		)
		controller.resolveCommit = func(
			context.Context,
			*config.Config,
			repoaudit.RepositoryReviewAutomation,
			string,
		) (string, error) {
			return commit, os.WriteFile(statePath, []byte("{"), 0o600)
		}
		if _, _, _, err := controller.repositoryReviewCommitOptions(
			t.Context(), automation.ID,
		); err == nil {
			t.Fatal("mid-resolution corruption returned no error")
		}
	})
	t.Run("state changes", func(t *testing.T) {
		_, store, automation, controller, workspace := setup(
			t, repoaudit.RepositoryReviewAutomationPaused, commit,
		)
		controller.resolveCommit = func(
			ctx context.Context,
			_ *config.Config,
			_ repoaudit.RepositoryReviewAutomation,
			_ string,
		) (string, error) {
			_, updateErr := store.UpdateAutomation(ctx, automation.ID, automation.Version, func(
				candidate *repoaudit.RepositoryReviewAutomation,
			) error {
				candidate.PauseDetail = "changed"
				return nil
			})
			return commit, updateErr
		}
		if _, _, _, err := controller.repositoryReviewCommitOptions(
			t.Context(), automation.ID,
		); !errors.Is(err, repoaudit.ErrConflict) {
			t.Fatalf("changed options error=%v", err)
		}

		current, found, err := store.GetAutomation(t.Context(), automation.ID)
		if err != nil || !found {
			t.Fatalf("changed automation found=%v err=%v", found, err)
		}
		controller.resolveCommit = func(
			ctx context.Context,
			_ *config.Config,
			_ repoaudit.RepositoryReviewAutomation,
			_ string,
		) (string, error) {
			return commit, store.DeleteAutomation(ctx, current.ID, current.Version)
		}
		if _, _, _, err := controller.repositoryReviewCommitOptions(
			t.Context(), current.ID,
		); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("deleted options error=%v", err)
		}

		statePath := filepath.Join(
			workspace,
			"repository_reviews",
			"automation_"+automation.ID+".json",
		)
		if writeErr := os.WriteFile(statePath, []byte("{"), 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
		if _, _, _, err := controller.repositoryReviewCommitOptions(
			t.Context(), automation.ID,
		); err == nil {
			t.Fatal("corrupt options state returned no error")
		}
	})
}

func TestRepositoryReviewPauseBoundaryCoverage(t *testing.T) {
	if err := applyRepositoryReviewPauseTransition(nil, 1, ""); !errors.Is(err, errRepositoryReviewInvalidTransition) {
		t.Fatalf("nil pause transition error=%v", err)
	}
	transition := testRepositoryReviewAutomation()
	transition.Version = 2
	transition.Status = repoaudit.RepositoryReviewAutomationRunning
	transition.ActiveRunID = "wr_transition"
	transition.RunIDs = []string{transition.ActiveRunID}
	if err := applyRepositoryReviewPauseTransition(&transition, 1, ""); !errors.Is(err, repoaudit.ErrConflict) {
		t.Fatalf("stale pause transition error=%v", err)
	}
	if err := applyRepositoryReviewPauseTransition(&transition, 2, "wr_other"); !errors.Is(err, repoaudit.ErrConflict) {
		t.Fatalf("wrong-run pause transition error=%v", err)
	}
	for _, status := range []repoaudit.RepositoryReviewAutomationStatus{
		repoaudit.RepositoryReviewAutomationStopping,
		repoaudit.RepositoryReviewAutomationPaused,
		repoaudit.RepositoryReviewAutomationCompleted,
		repoaudit.RepositoryReviewAutomationFailed,
	} {
		candidate := transition
		candidate.Status = status
		candidate.ActiveRunID = ""
		candidate.RunIDs = []string{"wr_transition"}
		if err := applyRepositoryReviewPauseTransition(
			&candidate, 2, "wr_transition",
		); !errors.Is(err, errRepositoryReviewPauseSettled) {
			t.Fatalf("settled status %q error=%v", status, err)
		}
	}
	invalid := transition
	invalid.Status = repoaudit.RepositoryReviewAutomationStatus("unknown")
	if err := applyRepositoryReviewPauseTransition(
		&invalid, 2, "wr_transition",
	); !errors.Is(err, errRepositoryReviewInvalidTransition) {
		t.Fatalf("invalid pause transition error=%v", err)
	}
	idleTransition := transition
	idleTransition.Status = repoaudit.RepositoryReviewAutomationIdle
	idleTransition.ActiveRunID = ""
	idleTransition.RunIDs = []string{"wr_transition"}
	idleTransition.AutoContinue = true
	idleTransition.Progress.Stage = "not queued"
	if err := applyRepositoryReviewPauseTransition(
		&idleTransition, 2, "wr_transition",
	); !errors.Is(err, errRepositoryReviewInvalidTransition) {
		t.Fatalf("unqueued pause transition error=%v", err)
	}
	idleTransition.Progress.Stage = "next batch queued"
	if err := applyRepositoryReviewPauseTransition(&idleTransition, 2, "wr_transition"); err != nil ||
		idleTransition.Status != repoaudit.RepositoryReviewAutomationPaused {
		t.Fatalf("queued pause transition=%#v error=%v", idleTransition, err)
	}

	if repositoryReviewPauseRunMatches(repoaudit.RepositoryReviewAutomation{}, "") {
		t.Fatal("empty pause run matched")
	}
	if repositoryReviewPauseRunMatches(
		repoaudit.RepositoryReviewAutomation{ActiveRunID: "wr_active"}, "wr_other",
	) {
		t.Fatal("different active run matched")
	}
	if !repositoryReviewPauseRunMatches(
		repoaudit.RepositoryReviewAutomation{RunIDs: []string{"wr_done"}}, "wr_done",
	) {
		t.Fatal("latest completed run did not match")
	}
	if repositoryReviewPauseRunMatches(
		repoaudit.RepositoryReviewAutomation{RunIDs: []string{"wr_done"}}, "wr_other",
	) {
		t.Fatal("different completed run matched")
	}

	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	controller := repositoryReviewCoverageLeasedController(t, handler, store)
	if _, pauseErr := controller.pauseAutomationForRun(
		t.Context(), "rra_not_found", 1, "wr_missing",
	); !errors.Is(pauseErr, os.ErrNotExist) {
		t.Fatalf("missing pause error=%v", pauseErr)
	}

	runningInput := testRepositoryReviewAutomation()
	runningInput.ID = "rra_pause_boundaries"
	runningInput.Status = repoaudit.RepositoryReviewAutomationRunning
	runningInput.ActiveRunID = "wr_pause_boundaries"
	runningInput.RunIDs = []string{runningInput.ActiveRunID}
	running, err := store.CreateAutomation(t.Context(), runningInput)
	if err != nil {
		t.Fatal(err)
	}
	running, err = store.UpdateAutomation(
		t.Context(), running.ID, running.Version,
		func(*repoaudit.RepositoryReviewAutomation) error { return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []struct {
		name    string
		version int64
		runID   string
	}{
		{name: "zero version", version: 0, runID: running.ActiveRunID},
		{name: "future version", version: running.Version + 1, runID: running.ActiveRunID},
		{name: "stale without run", version: running.Version - 1},
		{name: "long run", version: running.Version, runID: strings.Repeat("x", 1025)},
		{name: "wrong run", version: running.Version, runID: "wr_other"},
	} {
		t.Run(request.name, func(t *testing.T) {
			if _, pauseErr := controller.pauseAutomationForRun(
				t.Context(), running.ID, request.version, request.runID,
			); !errors.Is(pauseErr, repoaudit.ErrConflict) {
				t.Fatalf("pause error=%v", pauseErr)
			}
		})
	}

	idleInput := testRepositoryReviewAutomation()
	idleInput.ID = "rra_pause_idle_boundary"
	idleInput.AutoContinue = false
	idle, err := store.CreateAutomation(t.Context(), idleInput)
	if err != nil {
		t.Fatal(err)
	}
	if _, pauseErr := controller.pauseAutomationForRun(
		t.Context(), idle.ID, idle.Version, "",
	); !errors.Is(pauseErr, errRepositoryReviewInvalidTransition) {
		t.Fatalf("idle pause error=%v", pauseErr)
	}

	pausedInput := testRepositoryReviewAutomation()
	pausedInput.ID = "rra_pause_settled_boundary"
	pausedInput.Status = repoaudit.RepositoryReviewAutomationPaused
	pausedInput.PauseReason = repoaudit.RepositoryReviewPauseManual
	pausedInput.PauseDetail = "paused"
	paused, err := store.CreateAutomation(t.Context(), pausedInput)
	if err != nil {
		t.Fatal(err)
	}
	if got, pauseErr := controller.pauseAutomationForRun(
		t.Context(), paused.ID, paused.Version, "",
	); pauseErr != nil || got.Status != repoaudit.RepositoryReviewAutomationPaused {
		t.Fatalf("settled pause=%#v err=%v", got, pauseErr)
	}
	if got, loadErr := loadSettledRepositoryReviewPause(
		t.Context(), store, paused.ID,
	); loadErr != nil || got.ID != paused.ID {
		t.Fatalf("loaded settled pause=%#v err=%v", got, loadErr)
	}
	if _, loadErr := loadSettledRepositoryReviewPause(
		t.Context(), store, "rra_settled_missing",
	); !errors.Is(loadErr, os.ErrNotExist) {
		t.Fatalf("missing settled pause error=%v", loadErr)
	}

	corruptInput := testRepositoryReviewAutomation()
	corruptInput.ID = "rra_pause_corrupt_boundary"
	corrupt, err := store.CreateAutomation(t.Context(), corruptInput)
	if err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(
		workspace,
		"repository_reviews",
		"automation_"+corrupt.ID+".json",
	)
	if writeErr := os.WriteFile(statePath, []byte("{"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	if _, pauseErr := controller.pauseAutomationForRun(
		t.Context(), corrupt.ID, corrupt.Version, "",
	); pauseErr == nil {
		t.Fatal("corrupt pause state returned no error")
	}
	if _, loadErr := loadSettledRepositoryReviewPause(
		t.Context(), store, corrupt.ID,
	); loadErr == nil {
		t.Fatal("corrupt settled pause state returned no error")
	}

	controller.cancel()
	if _, pauseErr := controller.pauseAutomationForRun(
		t.Context(), running.ID, running.Version, running.ActiveRunID,
	); !errors.Is(pauseErr, context.Canceled) {
		t.Fatalf("stopped pause error=%v", pauseErr)
	}
}

func TestRepositoryReviewPausedTaskAdmissionBoundaryCoverage(t *testing.T) {
	controller := &repositoryReviewController{
		ctx: context.Background(),
		active: map[string]*repositoryReviewActiveRun{
			"rra_paused_task": {
				runID:        "wr_paused_task",
				pauseReason:  repoaudit.RepositoryReviewPauseManual,
				pauseDetail:  "operator paused this task",
				guardMu:      &sync.Mutex{},
				reservations: make(map[int]repositoryReviewTaskReservation),
			},
		},
	}
	err := controller.observeRepositoryReviewTask(
		"rra_paused_task",
		"wr_paused_task",
		workflows.ManagedChildActivity{Phase: workflows.ManagedChildStarted, Index: 1},
	)
	if !errors.Is(err, errRepositoryReviewSafeStop) ||
		!strings.Contains(err.Error(), "operator paused this task") {
		t.Fatalf("paused task admission error=%v", err)
	}

	guard := &sync.Mutex{}
	guard.Lock()
	controller.active["rra_mid_admission_pause"] = &repositoryReviewActiveRun{
		runID:        "wr_mid_admission_pause",
		guardMu:      guard,
		reservations: make(map[int]repositoryReviewTaskReservation),
	}
	result := make(chan error, 1)
	started := make(chan struct{})
	go func() {
		close(started)
		result <- controller.observeRepositoryReviewTask(
			"rra_mid_admission_pause",
			"wr_mid_admission_pause",
			workflows.ManagedChildActivity{
				Phase: workflows.ManagedChildStarted,
				Index: 2,
			},
		)
	}()
	<-started
	time.Sleep(10 * time.Millisecond)
	controller.mu.Lock()
	controller.active["rra_mid_admission_pause"].pauseReason = repoaudit.RepositoryReviewPauseManual
	controller.active["rra_mid_admission_pause"].pauseDetail = "paused while waiting for admission"
	controller.mu.Unlock()
	guard.Unlock()
	select {
	case err := <-result:
		if !errors.Is(err, errRepositoryReviewSafeStop) ||
			!strings.Contains(err.Error(), "paused while waiting for admission") {
			t.Fatalf("mid-admission pause error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("mid-admission pause did not finish")
	}
}

func TestRepositoryReviewExecutionObserverBoundaryCoverage(t *testing.T) {
	controller := &repositoryReviewController{}
	observer := controller.repositoryReviewManagedChildObserver("rra_observer", "wr_observer")
	if err := observer(workflows.ManagedChildActivityEvent{
		RunID: "wr_other", StepID: "review",
	}); err != nil {
		t.Fatalf("other-run observer error=%v", err)
	}
	if err := observer(workflows.ManagedChildActivityEvent{
		RunID: "wr_observer", StepID: "other",
	}); err != nil {
		t.Fatalf("other-step observer error=%v", err)
	}
	if err := observer(workflows.ManagedChildActivityEvent{
		RunID: "wr_observer", StepID: "review",
		ManagedChildActivity: workflows.ManagedChildActivity{
			Phase: workflows.ManagedChildActivityPhase("unknown"),
		},
	}); err != nil {
		t.Fatalf("ignored activity observer error=%v", err)
	}
	commitA := strings.Repeat("a", 40)
	commitB := strings.Repeat("b", 40)
	automation := repoaudit.RepositoryReviewAutomation{ResolvedCommitSHA: commitA}
	result := &workflows.RunResult{Outputs: map[string]any{"commit": commitB}}
	runErr := repositoryReviewJoinCommitError(automation, result, nil)
	if runErr == nil || !strings.Contains(runErr.Error(), commitA) {
		t.Fatalf("joined commit error=%v", runErr)
	}
	original := errors.New("run failed")
	if joined := repositoryReviewJoinCommitError(automation, nil, original); !errors.Is(joined, original) {
		t.Fatalf("joined run error=%v", joined)
	}
	selection := repoaudit.RepositoryReviewScopeSelection{IncludePrefixes: []string{"pkg"}}
	plan := repoaudit.RepositoryReviewScopePlan{
		CommitSHA: commitA, PolicyHash: strings.Repeat("c", 64), Hash: strings.Repeat("d", 64),
		Summary: "frozen scope",
	}
	automation.ScopeSelection = &selection
	automation.ScopePlan = plan
	matchingScope := &workflows.RunResult{Outputs: map[string]any{
		"scopeSelection": repositoryReviewWorkflowObject(selection),
		"scopePlan":      repositoryReviewWorkflowObject(plan),
	}}
	if err := repositoryReviewValidateExecutionScope(automation, matchingScope); err != nil {
		t.Fatalf("matching frozen scope error=%v", err)
	}
	persistedScope := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"scope": {
			Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{
				"scopeSelection": repositoryReviewWorkflowObject(selection),
				"scopePlan":      repositoryReviewWorkflowObject(plan),
			},
		},
		"review": {Status: workflows.RunStatusFailed, Error: "later review failed"},
	}}
	if err := repositoryReviewValidatePersistedScope(automation, persistedScope); err != nil {
		t.Fatalf("persisted frozen scope after later failure error=%v", err)
	}
	matchingScope.Outputs["scopeSelection"] = map[string]any{
		"include_prefixes": []any{"other"},
	}
	if err := repositoryReviewValidateExecutionScope(automation, matchingScope); err == nil ||
		!strings.Contains(err.Error(), "changed") {
		t.Fatalf("changed frozen scope error=%v", err)
	}
	if err := repositoryReviewValidateExecutionScope(
		automation,
		&workflows.RunResult{Outputs: map[string]any{}},
	); err == nil || !strings.Contains(err.Error(), "omitted") {
		t.Fatalf("omitted frozen scope error=%v", err)
	}
	if err := repositoryReviewValidateExecutionScope(automation, nil); err == nil ||
		!strings.Contains(err.Error(), "omitted") {
		t.Fatalf("nil frozen scope result error=%v", err)
	}
	if err := repositoryReviewValidateExecutionScope(
		automation, &workflows.RunResult{},
	); err == nil || !strings.Contains(err.Error(), "omitted") {
		t.Fatalf("nil frozen scope outputs error=%v", err)
	}
	if got := repositoryReviewWorkflowObject(nil); len(got) != 0 {
		t.Fatalf("nil workflow object=%#v", got)
	}
	if got := repositoryReviewWorkflowObject(make(chan int)); len(got) != 0 {
		t.Fatalf("unserializable workflow object=%#v", got)
	}
	var nilSelection *repoaudit.RepositoryReviewScopeSelection
	if got := repositoryReviewWorkflowObject(nilSelection); len(got) != 0 {
		t.Fatalf("null workflow object=%#v", got)
	}
	if got := repositoryReviewWorkflowObject("scalar"); len(got) != 0 {
		t.Fatalf("scalar workflow object=%#v", got)
	}
	if planned, selectionInput, planInput := repositoryReviewWorkflowScopeInputs(
		repoaudit.RepositoryReviewAutomation{},
	); planned || len(selectionInput) != 0 || len(planInput) != 0 {
		t.Fatalf("empty workflow scope inputs=(%v, %#v, %#v)", planned, selectionInput, planInput)
	}
	if planned, selectionInput, planInput := repositoryReviewWorkflowScopeInputs(automation); !planned ||
		selectionInput["include_prefixes"] == nil || planInput["hash"] != plan.Hash {
		t.Fatalf("frozen workflow scope inputs=(%v, %#v, %#v)", planned, selectionInput, planInput)
	}
	if err := repositoryReviewValidatePersistedScope(
		repoaudit.RepositoryReviewAutomation{}, nil,
	); err != nil {
		t.Fatalf("unplanned persisted scope error=%v", err)
	}
	if err := repositoryReviewValidatePersistedScope(
		automation, &workflows.Run{},
	); err == nil || !strings.Contains(err.Error(), "omitted") {
		t.Fatalf("missing persisted scope error=%v", err)
	}
	invalidOutputs := &workflows.RunResult{Outputs: map[string]any{
		"scopeSelection": make(chan int), "scopePlan": repositoryReviewWorkflowObject(plan),
	}}
	if err := repositoryReviewValidateExecutionScope(automation, invalidOutputs); err == nil ||
		!strings.Contains(err.Error(), "invalid") {
		t.Fatalf("invalid frozen scope outputs error=%v", err)
	}
}

func TestRepositoryReviewPersistsValidatedScopeBeforeReviewWork(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, storeErr := handler.repositoryReviewStore()
	if storeErr != nil {
		t.Fatal(storeErr)
	}
	commit := strings.Repeat("a", 40)
	input := testRepositoryReviewAutomation()
	input.Status = repoaudit.RepositoryReviewAutomationRunning
	input.ActiveRunID = "wr_scope_freeze"
	input.RunIDs = []string{input.ActiveRunID}
	input.ResolvedCommitSHA = commit
	input.Progress.TotalBatches = 1
	automation, createErr := store.CreateAutomation(t.Context(), input)
	if createErr != nil {
		t.Fatal(createErr)
	}
	selection := repoaudit.RepositoryReviewScopeSelection{IncludePrefixes: []string{"pkg"}}
	plan := repoaudit.RepositoryReviewScopePlan{
		CommitSHA: commit, PolicyHash: strings.Repeat("b", 64), Hash: strings.Repeat("c", 64),
		Summary: "Frozen target scope", Rationale: "Production package",
		Warnings: []string{},
		Counts: repoaudit.RepositoryReviewScopePlanCounts{
			TotalFiles: 3, CodeTypeFiles: 3, IncludeFiles: 2, SelectedFiles: 2,
		},
	}
	run := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"scope": {
			Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{
				"scopeSelection": repositoryReviewWorkflowObject(selection),
				"scopePlan":      repositoryReviewWorkflowObject(plan),
			},
		},
	}}
	controller := newRepositoryReviewController(handler)
	observer := controller.repositoryReviewStepObserver(
		automation.ID, automation.ActiveRunID, store, nil,
	)
	if err := observer(workflows.StepActivityEvent{RunID: "other", StepID: "scope_files"}); err != nil {
		t.Fatalf("other-run scope observer error=%v", err)
	}
	if err := observer(workflows.StepActivityEvent{
		RunID: automation.ActiveRunID, StepID: "scope_files",
	}); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("nil-store scope observer error=%v", err)
	}
	missingWorkflowStore := workflows.NewFileRunStore(t.TempDir())
	observer = controller.repositoryReviewStepObserver(
		automation.ID, automation.ActiveRunID, store, missingWorkflowStore,
	)
	if err := observer(workflows.StepActivityEvent{
		RunID: automation.ActiveRunID, StepID: "scope_files",
	}); err == nil || !strings.Contains(err.Error(), "load") {
		t.Fatalf("missing-run scope observer error=%v", err)
	}
	if err := controller.persistRepositoryReviewFrozenScope(
		store, automation.ID, automation.ActiveRunID, &workflows.Run{},
	); err == nil || !strings.Contains(err.Error(), "not durably validated") {
		t.Fatalf("unvalidated durable scope error=%v", err)
	}
	omitted := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"scope": {Status: workflows.RunStatusSucceeded},
	}}
	if err := controller.persistRepositoryReviewFrozenScope(
		store, automation.ID, automation.ActiveRunID, omitted,
	); err == nil || !strings.Contains(err.Error(), "omitted") {
		t.Fatalf("omitted durable scope error=%v", err)
	}
	workflowStore := workflows.NewFileRunStore(t.TempDir())
	run.ID = automation.ActiveRunID
	run.Status = workflows.RunStatusRunning
	run.CreatedAt = time.Now().UTC()
	run.UpdatedAt = run.CreatedAt
	if err := workflowStore.CreateRun(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	observer = controller.repositoryReviewStepObserver(
		automation.ID, automation.ActiveRunID, store, workflowStore,
	)
	if err := observer(workflows.StepActivityEvent{
		RunID: automation.ActiveRunID, StepID: "scope_files",
	}); err != nil {
		t.Fatalf("durable scope observer error=%v", err)
	}
	if err := controller.persistRepositoryReviewFrozenScope(
		store, automation.ID, automation.ActiveRunID, run,
	); err != nil {
		t.Fatal(err)
	}
	updated, found, getErr := store.GetAutomation(t.Context(), automation.ID)
	if getErr != nil || !found || updated.ScopeSelection == nil ||
		!repositoryReviewScopeSelectionsEqual(*updated.ScopeSelection, selection) ||
		!repositoryReviewScopePlansEqual(updated.ScopePlan, plan) {
		t.Fatalf("persisted frozen scope=%#v found=%v err=%v", updated, found, getErr)
	}
	if err := controller.persistRepositoryReviewFrozenScope(
		store, automation.ID, automation.ActiveRunID, run,
	); err != nil {
		t.Fatalf("idempotent frozen scope error=%v", err)
	}
	changed := plan
	changed.Hash = strings.Repeat("d", 64)
	run.Steps["scope"] = workflows.StepExecution{
		Status: workflows.RunStatusSucceeded,
		Outputs: map[string]any{
			"scopeSelection": repositoryReviewWorkflowObject(selection),
			"scopePlan":      repositoryReviewWorkflowObject(changed),
		},
	}
	if err := controller.persistRepositoryReviewFrozenScope(
		store, automation.ID, automation.ActiveRunID, run,
	); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("changed durable scope error=%v", err)
	}
	if err := controller.persistRepositoryReviewFrozenScope(
		store, automation.ID, "wr_other", run,
	); !errors.Is(err, errRepositoryReviewSafeStop) {
		t.Fatalf("stale durable scope error=%v", err)
	}
	mismatchInput := testRepositoryReviewAutomation()
	mismatchInput.Repository = "owner/frozen-scope-mismatch"
	mismatchInput.Name = "Frozen scope mismatch"
	mismatchInput.Status = repoaudit.RepositoryReviewAutomationRunning
	mismatchInput.ActiveRunID = "wr_scope_mismatch"
	mismatchInput.RunIDs = []string{mismatchInput.ActiveRunID}
	mismatchInput.ResolvedCommitSHA = strings.Repeat("f", 40)
	mismatchInput.Progress.TotalBatches = 1
	mismatch, mismatchErr := store.CreateAutomation(t.Context(), mismatchInput)
	if mismatchErr != nil {
		t.Fatal(mismatchErr)
	}
	if err := controller.persistRepositoryReviewFrozenScope(
		store, mismatch.ID, mismatch.ActiveRunID, run,
	); err == nil || !strings.Contains(err.Error(), "admitted commit") {
		t.Fatalf("commit-mismatched durable scope error=%v", err)
	}
}

func TestRepositoryReviewCoverageRunAndProgressHelpers(t *testing.T) {
	if repositoryReviewWorkflowStage(nil) != "" || repositoryReviewRunStep(nil, "review").ID != "" {
		t.Fatal("nil workflow helpers returned state")
	}
	queued := &workflows.Run{Steps: map[string]workflows.StepExecution{}}
	if repositoryReviewWorkflowStage(queued) != "queued" {
		t.Fatalf("queued stage=%q", repositoryReviewWorkflowStage(queued))
	}
	running := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"review": {ID: "review", Status: workflows.RunStatusRunning},
	}}
	if repositoryReviewWorkflowStage(running) != "Reviewing bounded file batch" ||
		repositoryReviewRunStep(running, "review").ID != "review" {
		t.Fatalf(
			"running stage=%q step=%#v",
			repositoryReviewWorkflowStage(running),
			repositoryReviewRunStep(running, "review"),
		)
	}
	succeeded := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"find_bugs/record": {
			ID: "record", Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{"run": map[string]any{
				"id": "saved", "remaining_files": 0,
			}},
		},
	}}
	result := &workflows.RunResult{Status: workflows.RunStatusSucceeded}
	if !repositoryReviewRunCheckpointed(succeeded, result) ||
		repositoryReviewRunCheckpointed(nil, result) || repositoryReviewRunCheckpointed(succeeded, nil) {
		t.Fatal("record checkpoint detection mismatch")
	}
	noPending := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"plan": {Status: workflows.RunStatusSucceeded, Outputs: map[string]any{"pending_count": 0}},
		"result": {
			Status:  workflows.RunStatusSucceeded,
			Outputs: map[string]any{"run": map[string]any{"remaining_files": 0}},
		},
	}}
	result.Outputs = map[string]any{"remainingFiles": 0}
	if !repositoryReviewRunCheckpointed(noPending, result) {
		t.Fatal("no-op checkpoint was not recognized")
	}

	automation := repoaudit.RepositoryReviewAutomation{MaxFilesPerRun: 2}
	applyRepositoryReviewRunProgress(nil, result, nil)
	applyRepositoryReviewRunProgress(&automation, nil, nil)
	applyRepositoryReviewRunProgress(&automation, &workflows.RunResult{Outputs: map[string]any{
		"remaining_files": 3, "reviewed_files": 2,
	}}, nil)
	if automation.Progress.RemainingFiles != 3 || automation.Progress.ReviewedFiles != 2 ||
		automation.Progress.TotalBatches != 2 {
		t.Fatalf("snake-case progress=%#v", automation.Progress)
	}

	if outcome := loadRepositoryReviewOutcome(
		repoaudit.NewStore(t.TempDir()),
		repoaudit.RepositoryReviewAutomation{Repository: "owner/missing"},
	); outcome.found {
		t.Fatalf("missing outcome=%#v", outcome)
	}
	applyRepositoryReviewOutcome(nil, repositoryReviewOutcome{found: true})
	applyRepositoryReviewOutcome(&automation, repositoryReviewOutcome{})
	addRepositoryReviewModelPaths(nil, "alias", []string{"path"})
	addRepositoryReviewModelPaths(&automation, "", []string{"path"})
	addRepositoryReviewModelPaths(&automation, "alias", nil)

	controller := newRepositoryReviewController(nil)
	if _, updateErr := controller.updateLatest(
		t.Context(),
		repoaudit.NewStore(t.TempDir()),
		"rra_missing",
		func(*repoaudit.RepositoryReviewAutomation) error {
			return nil
		},
	); updateErr == nil {
		t.Fatal("missing updateLatest automation was accepted")
	}
	var nilController *repositoryReviewController
	if nilController.clock().IsZero() {
		t.Fatal("nil controller clock returned zero")
	}
}

func TestRepositoryReviewRemainingFilesUsesStrictPrecedence(t *testing.T) {
	durableRun := func(value any) *workflows.Run {
		return &workflows.Run{Steps: map[string]workflows.StepExecution{
			"find_bugs/record": {
				Status:  workflows.RunStatusSucceeded,
				Outputs: map[string]any{"run": map[string]any{"remaining_files": value}},
			},
		}}
	}
	tests := []struct {
		name   string
		result *workflows.RunResult
		run    *workflows.Run
		want   int
		ok     bool
	}{
		{
			name: "contradictory camel zero and durable positive fails",
			result: &workflows.RunResult{Outputs: map[string]any{
				"remainingFiles": 0, "remaining_files": 4,
			}},
			run: durableRun(5),
		},
		{
			name: "contradictory top level aliases fail even when durable matches camel",
			result: &workflows.RunResult{Outputs: map[string]any{
				"remainingFiles": 0, "remaining_files": 5,
			}},
			run: durableRun(0),
		},
		{
			name: "contradictory valid snake and durable fails",
			result: &workflows.RunResult{Outputs: map[string]any{
				"remainingFiles": "0", "remaining_files": float64(2),
			}},
			run: durableRun(5),
		},
		{
			name: "valid top level agrees with durable",
			result: &workflows.RunResult{Outputs: map[string]any{
				"remainingFiles": "invalid", "remaining_files": float64(5),
			}},
			run: durableRun(5), want: 5, ok: true,
		},
		{
			name: "invalid top level aliases fall through to durable",
			result: &workflows.RunResult{Outputs: map[string]any{
				"remainingFiles": -1, "remaining_files": 1.5,
			}},
			run: durableRun(float64(3)), want: 3, ok: true,
		},
		{
			name:   "missing top level falls through to durable zero",
			result: &workflows.RunResult{Outputs: map[string]any{}},
			run:    durableRun(0), ok: true,
		},
		{
			name:   "missing everywhere",
			result: &workflows.RunResult{Outputs: map[string]any{}},
			run: &workflows.Run{Steps: map[string]workflows.StepExecution{
				"find_bugs/record": {
					Status:  workflows.RunStatusSucceeded,
					Outputs: map[string]any{"run": map[string]any{"id": "saved"}},
				},
			}},
		},
		{
			name:   "malformed durable remaining",
			result: &workflows.RunResult{Outputs: map[string]any{}},
			run:    durableRun("3"),
		},
		{
			name:   "negative durable remaining",
			result: &workflows.RunResult{Outputs: map[string]any{}},
			run:    durableRun(-3),
		},
		{
			name:   "nonintegral durable remaining",
			result: &workflows.RunResult{Outputs: map[string]any{}},
			run:    durableRun(3.5),
		},
		{
			name:   "failed record is not trusted",
			result: &workflows.RunResult{Outputs: map[string]any{}},
			run: &workflows.Run{Steps: map[string]workflows.StepExecution{
				"find_bugs/record": {
					Status:  workflows.RunStatusFailed,
					Outputs: map[string]any{"run": map[string]any{"remaining_files": 0}},
				},
			}},
		},
		{
			name:   "durable camel alias is not trusted",
			result: &workflows.RunResult{Outputs: map[string]any{}},
			run: &workflows.Run{Steps: map[string]workflows.StepExecution{
				"find_bugs/record": {
					Status:  workflows.RunStatusSucceeded,
					Outputs: map[string]any{"run": map[string]any{"remainingFiles": 0}},
				},
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := repositoryReviewRemainingFiles(test.result, test.run)
			if ok != test.ok || got != test.want {
				t.Fatalf("remaining=(%d, %v), want (%d, %v)", got, ok, test.want, test.ok)
			}
		})
	}
}

func TestRepositoryReviewNonnegativeIntRejectsAmbiguousNumbers(t *testing.T) {
	valid := []struct {
		value any
		want  int
	}{
		{value: 0},
		{value: int8(1), want: 1},
		{value: int16(1), want: 1},
		{value: int32(1), want: 1},
		{value: int64(1), want: 1},
		{value: uint(2), want: 2},
		{value: uint8(2), want: 2},
		{value: uint16(2), want: 2},
		{value: uint32(2), want: 2},
		{value: uint64(2), want: 2},
		{value: float32(3), want: 3},
		{value: float64(4), want: 4},
		{value: json.Number("5"), want: 5},
		{value: repositoryReviewMaximumFiles, want: repositoryReviewMaximumFiles},
	}
	for _, test := range valid {
		if got, ok := repositoryReviewNonnegativeInt(test.value); !ok || got != test.want {
			t.Fatalf("valid value %#v parsed as (%d, %v)", test.value, got, ok)
		}
	}
	for _, value := range []any{
		nil, true, "0", int(-1), int64(-2), float32(-3), float64(-4),
		float32(1.5), float64(2.5), json.Number("3.0"), json.Number("bad"),
		repositoryReviewMaximumFiles + 1, uint64(^uint(0)), json.Number("100001"),
	} {
		if got, ok := repositoryReviewNonnegativeInt(value); ok {
			t.Fatalf("invalid value %#v parsed as %d", value, got)
		}
	}
	remaining, found, conflict := repositoryReviewTopLevelRemainingFilesDetailed(nil)
	if remaining != 0 || found || conflict {
		t.Fatalf("nil result remaining=(%d, %v, %v)", remaining, found, conflict)
	}
	if remaining, found := repositoryReviewDurableResultRemainingFiles(&workflows.Run{
		Steps: map[string]workflows.StepExecution{
			"result": {
				Status:  workflows.RunStatusSucceeded,
				Outputs: map[string]any{"run": true},
			},
		},
	}); remaining != 0 || found {
		t.Fatalf("non-map durable result remaining=(%d, %v)", remaining, found)
	}
}

func TestRepositoryReviewDurableRecordFileCountsRejectsNonMapRun(t *testing.T) {
	reviewed, unsupported, found := repositoryReviewDurableRecordFileCounts(&workflows.Run{
		Steps: map[string]workflows.StepExecution{
			"record": {
				Status:  workflows.RunStatusSucceeded,
				Outputs: map[string]any{"run": true},
			},
		},
	})
	if reviewed != 0 || unsupported != 0 || found {
		t.Fatalf("non-map durable file counts = (%d, %d, %v)", reviewed, unsupported, found)
	}
}

func TestApplyRepositoryReviewRunProgressDoesNotOverwriteWithoutValidRemaining(t *testing.T) {
	automation := repoaudit.RepositoryReviewAutomation{
		MaxFilesPerRun: 2,
		Progress: repoaudit.RepositoryReviewProgress{
			RemainingFiles: 9,
		},
	}
	applyRepositoryReviewRunProgress(
		&automation,
		&workflows.RunResult{Outputs: map[string]any{"remainingFiles": "missing"}},
		nil,
	)
	if automation.Progress.RemainingFiles != 9 {
		t.Fatalf("malformed remaining overwrote progress: %#v", automation.Progress)
	}
	durable := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"find_bugs/record": {
			Status:  workflows.RunStatusSucceeded,
			Outputs: map[string]any{"run": map[string]any{"remaining_files": 4}},
		},
	}}
	applyRepositoryReviewRunProgress(
		&automation,
		&workflows.RunResult{Outputs: map[string]any{"remainingFiles": false}},
		durable,
	)
	if automation.Progress.RemainingFiles != 4 {
		t.Fatalf("durable fallback was not applied: %#v", automation.Progress)
	}
	applyRepositoryReviewRunProgress(
		&automation,
		&workflows.RunResult{Outputs: map[string]any{
			"remainingFiles": 0, "remaining_files": 2,
		}},
		durable,
	)
	if automation.Progress.RemainingFiles != 4 {
		t.Fatalf("contradictory projected zero overwrote durable progress: %#v", automation.Progress)
	}
}

func TestRepositoryReviewShouldRecoverLegacyCampaignOnlyOnResume(t *testing.T) {
	legacy := repoaudit.RepositoryReviewAutomation{RunIDs: []string{"wr_legacy"}, StartedAt: time.Now()}
	if !repositoryReviewShouldRecoverLegacyCampaign(legacy, "resume") {
		t.Fatal("legacy resume did not request campaign recovery")
	}
	legacy.CampaignID = repoaudit.NewRepositoryReviewCampaignID()
	if repositoryReviewShouldRecoverLegacyCampaign(legacy, "resume") ||
		repositoryReviewShouldRecoverLegacyCampaign(repoaudit.RepositoryReviewAutomation{}, "start") {
		t.Fatal("campaign recovery escaped legacy resume boundary")
	}
	legacy.CampaignRecoveryPending = true
	if !repositoryReviewShouldRecoverLegacyCampaign(legacy, "resume") {
		t.Fatal("torn legacy recovery marker was not resumable")
	}
	legacy.CampaignID = ""
	legacy.CampaignRecoveryPending = false
	legacy.Progress.Stage = "next batch queued"
	if !repositoryReviewShouldRecoverLegacyCampaign(legacy, "start") {
		t.Fatal("legacy automatic handoff did not request campaign recovery")
	}
}

func TestRepositoryReviewCampaignRecoveryAdmissionBoundaries(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	commit := strings.Repeat("a", 40)
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		return commit, nil
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	input := testRepositoryReviewAutomation()
	input.Status = repoaudit.RepositoryReviewAutomationFailed
	input.PauseReason = repoaudit.RepositoryReviewPauseRunFailed
	input.ResolvedCommitSHA = commit
	input.RunIDs = []string{"wr_legacy_boundary"}
	input.StartedAt = time.Now().Add(-time.Hour)
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}

	controller.recoverCampaign = nil
	if _, startErr := controller.startAutomation(
		t.Context(), automation.ID, automation.Version, false, "resume",
	); startErr == nil || !strings.Contains(startErr.Error(), "recovery is unavailable") {
		t.Fatalf("missing recovery adapter error = %v", startErr)
	}
	automation, found, err := store.GetAutomation(t.Context(), automation.ID)
	if err != nil || !found {
		t.Fatalf("reload automation found=%v err=%v", found, err)
	}

	previousRunners := newWorkflowRuntimeRunners
	t.Cleanup(func() { newWorkflowRuntimeRunners = previousRunners })
	newWorkflowRuntimeRunners = func(string) workflowRuntimeRunners {
		return workflowRuntimeRunners{Agents: fakeWorkflowRuntimeRunner{}}
	}
	controller.recoverCampaign = controller.recoverLegacyRepositoryReviewCampaign
	if _, startErr := controller.startAutomation(
		t.Context(), automation.ID, automation.Version, false, "resume",
	); startErr == nil || !strings.Contains(startErr.Error(), "profile-aware runtime") {
		t.Fatalf("non-profile recovery runtime error = %v", startErr)
	}

	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	closeable := &closeableRepositoryReviewProfileRunner{
		repositoryReviewRecoveryProfileRunner: &repositoryReviewRecoveryProfileRunner{
			profile: workflows.RepositoryReviewModelProfile{
				Revision: "sha256:profile", ReviewerModels: []string{"cheap"},
				MaxContentBytes: 65536,
			},
		},
	}
	newWorkflowRuntimeRunners = func(string) workflowRuntimeRunners {
		return workflowRuntimeRunners{Agents: closeable}
	}
	profile, err := resolveRepositoryReviewCampaignProfile(
		t.Context(), handler.configPath, cfg, automation,
	)
	if err != nil || profile.Revision != "sha256:profile" || closeable.closed != 1 {
		t.Fatalf("closeable profile=%#v closed=%d err=%v", profile, closeable.closed, err)
	}
	resetRepositoryReviewCampaignProgress(nil)
}

func TestRepositoryReviewCampaignAdmissionReadsLedgerAndFencesFinalCAS(t *testing.T) {
	t.Run("ledger read failure", func(t *testing.T) {
		handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		store, err := handler.repositoryReviewStore()
		if err != nil {
			t.Fatal(err)
		}
		commit := strings.Repeat("b", 40)
		input := testRepositoryReviewAutomation()
		if _, beginErr := store.BeginCampaign(t.Context(), repoaudit.BeginCampaignRequest{
			Repository: repoaudit.CanonicalRepositoryIdentity(input.Repository),
			CampaignID: repoaudit.NewRepositoryReviewCampaignID(),
			CommitSHA:  commit, ExpectedReviewVersion: 0, Exact: true,
		}); beginErr != nil {
			t.Fatal(beginErr)
		}
		paths, err := filepath.Glob(filepath.Join(workspace, "repository_reviews", "repo_*.json"))
		if err != nil {
			t.Fatal(err)
		}
		statePath := ""
		for _, candidate := range paths {
			if !strings.HasSuffix(candidate, ".summary.json") {
				statePath = candidate
				break
			}
		}
		if statePath == "" {
			t.Fatal("repository state path was not created")
		}
		if writeErr := os.WriteFile(statePath, []byte("{"), 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
		automation, err := store.CreateAutomation(t.Context(), input)
		if err != nil {
			t.Fatal(err)
		}
		controller := handler.repositoryReviewControllerInstance()
		controller.resolveCommit = func(
			context.Context,
			*config.Config,
			repoaudit.RepositoryReviewAutomation,
			string,
		) (string, error) {
			return commit, nil
		}
		if _, startErr := controller.startAutomation(
			t.Context(), automation.ID, automation.Version, false, "start",
		); startErr == nil || !strings.Contains(startErr.Error(), "unexpected end of JSON") {
			t.Fatalf("corrupt repository ledger error=%v", startErr)
		}
	})

	t.Run("final admission mismatch", func(t *testing.T) {
		handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		store, err := handler.repositoryReviewStore()
		if err != nil {
			t.Fatal(err)
		}
		automation, err := store.CreateAutomation(t.Context(), testRepositoryReviewAutomation())
		if err != nil {
			t.Fatal(err)
		}
		controller := handler.repositoryReviewControllerInstance()
		commit := strings.Repeat("d", 40)
		controller.resolveCommit = func(
			context.Context,
			*config.Config,
			repoaudit.RepositoryReviewAutomation,
			string,
		) (string, error) {
			return commit, nil
		}
		calls := 0
		controller.update = func(
			ctx context.Context,
			candidateStore repoaudit.Store,
			id string,
			expectedVersion int64,
			mutate func(*repoaudit.RepositoryReviewAutomation) error,
		) (repoaudit.RepositoryReviewAutomation, error) {
			calls++
			if calls != 3 {
				return updateRepositoryReviewAutomation(
					ctx, candidateStore, id, expectedVersion, mutate,
				)
			}
			candidate, found, getErr := candidateStore.GetAutomation(ctx, id)
			if getErr != nil || !found {
				return repoaudit.RepositoryReviewAutomation{}, getErr
			}
			candidate.CampaignID = repoaudit.NewRepositoryReviewCampaignID()
			return repoaudit.RepositoryReviewAutomation{}, mutate(&candidate)
		}
		if _, startErr := controller.startAutomation(
			t.Context(), automation.ID, automation.Version, false, "start",
		); !errors.Is(startErr, repoaudit.ErrConflict) {
			t.Fatalf("final admission mismatch error = %v", startErr)
		}
	})

	t.Run("authorization failure preserves newer campaign", func(t *testing.T) {
		handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		store, err := handler.repositoryReviewStore()
		if err != nil {
			t.Fatal(err)
		}
		automation, err := store.CreateAutomation(t.Context(), testRepositoryReviewAutomation())
		if err != nil {
			t.Fatal(err)
		}
		controller := handler.repositoryReviewControllerInstance()
		commit := strings.Repeat("f", 40)
		controller.resolveCommit = func(
			context.Context,
			*config.Config,
			repoaudit.RepositoryReviewAutomation,
			string,
		) (string, error) {
			return commit, nil
		}
		calls := 0
		var newerCampaignID string
		controller.update = func(
			ctx context.Context,
			candidateStore repoaudit.Store,
			id string,
			expectedVersion int64,
			mutate func(*repoaudit.RepositoryReviewAutomation) error,
		) (repoaudit.RepositoryReviewAutomation, error) {
			calls++
			updated, updateErr := updateRepositoryReviewAutomation(
				ctx, candidateStore, id, expectedVersion, mutate,
			)
			if updateErr != nil || calls != 2 {
				return updated, updateErr
			}
			identity := repoaudit.CanonicalRepositoryIdentity(updated.Repository)
			ledger, _, getErr := candidateStore.Get(identity)
			if getErr != nil {
				return repoaudit.RepositoryReviewAutomation{}, getErr
			}
			if _, beginErr := candidateStore.BeginCampaign(ctx, repoaudit.BeginCampaignRequest{
				Repository: identity, CampaignID: updated.CampaignID,
				CommitSHA: strings.Repeat("e", 40), ExpectedReviewVersion: ledger.ReviewVersion,
				Exact: true,
			}); beginErr != nil {
				return repoaudit.RepositoryReviewAutomation{}, beginErr
			}
			newerCampaignID = repoaudit.NewRepositoryReviewCampaignID()
			if _, replaceErr := candidateStore.UpdateAutomation(
				ctx, id, updated.Version,
				func(candidate *repoaudit.RepositoryReviewAutomation) error {
					candidate.CampaignID = newerCampaignID
					return nil
				},
			); replaceErr != nil {
				return repoaudit.RepositoryReviewAutomation{}, replaceErr
			}
			return updated, nil
		}
		if _, startErr := controller.startAutomation(
			t.Context(), automation.ID, automation.Version, false, "start",
		); !errors.Is(startErr, repoaudit.ErrConflict) {
			t.Fatalf("campaign authorization error = %v", startErr)
		}
		current, found, err := store.GetAutomation(t.Context(), automation.ID)
		if err != nil || !found || current.CampaignID != newerCampaignID ||
			current.Status == repoaudit.RepositoryReviewAutomationFailed {
			t.Fatalf("newer campaign=%#v found=%v err=%v", current, found, err)
		}
	})
}

func TestRepositoryReviewCampaignAdmissionResetsBudgetAndHonorsStoppedController(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	input := testRepositoryReviewAutomation()
	input.Usage = repoaudit.RepositoryReviewTokenUsage{
		PromptTokens: 4, CompletionTokens: 1, TotalTokens: 5,
	}
	input.EstimatedCostUSD = 0.25
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	controller := handler.repositoryReviewControllerInstance()
	controller.runBatch = func(
		ctx context.Context,
		_ repoaudit.RepositoryReviewAutomation,
		_ string,
		_ workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	started, err := controller.startAutomation(
		t.Context(), automation.ID, automation.Version, true, "start",
	)
	if err != nil || started.Usage.TotalTokens != 0 || started.EstimatedCostUSD != 0 {
		t.Fatalf("budget reset admission=%#v err=%v", started, err)
	}

	stoppedController := newRepositoryReviewController(handler)
	stoppedController.stopped = true
	stoppedController.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		return strings.Repeat("a", 40), nil
	}
	second := testRepositoryReviewAutomation()
	second.Repository = "https://github.com/acme/stopped.git"
	second, err = store.CreateAutomation(t.Context(), second)
	if err != nil {
		t.Fatal(err)
	}
	if _, startErr := stoppedController.startAutomation(
		t.Context(), second.ID, second.Version, false, "start",
	); !errors.Is(startErr, context.Canceled) {
		t.Fatalf("stopped campaign admission error=%v", startErr)
	}
}

func TestApplyRepositoryReviewOutcomeUsesExactCampaignMetrics(t *testing.T) {
	automation := repoaudit.RepositoryReviewAutomation{Progress: repoaudit.RepositoryReviewProgress{
		ReviewedFiles: 9, RemainingFiles: 9, UnsupportedFiles: 9,
	}}
	applyRepositoryReviewOutcome(&automation, repositoryReviewOutcome{
		found: true, coverageAvailable: true, coverageExact: true,
		selectedFiles: 5, inspectedFiles: 4, reviewedFiles: 2, remainingFiles: 2,
		unsupportedFiles: 1, findings: 7, findingAggregates: 3, pendingFindingMappings: 2,
		modelFindings: map[string]int{}, modelPaths: map[string][]string{},
	})
	progress := automation.Progress
	if !progress.CoverageAvailable || !progress.CoverageExact || progress.SelectedFiles != 5 ||
		progress.InspectedFiles != 4 || progress.ReviewedFiles != 2 || progress.RemainingFiles != 2 ||
		progress.UnsupportedFiles != 1 || progress.Findings != 7 || progress.FindingAggregates != 3 ||
		progress.PendingFindingMappings != 2 {
		t.Fatalf("exact campaign progress=%#v", progress)
	}
}

func TestRepositoryReviewCampaignOutcomeUsesTaggedCoverageAndModelContexts(t *testing.T) {
	campaignID := repoaudit.NewRepositoryReviewCampaignID()
	files := []repoaudit.FileRef{
		{Path: "a.go", BlobSHA: strings.Repeat("a", 40), SizeBytes: 1},
		{Path: "b.go", BlobSHA: strings.Repeat("b", 40), SizeBytes: 2},
		{Path: "c.bin", BlobSHA: strings.Repeat("c", 40), SizeBytes: 3},
	}
	state := repoaudit.RepositoryState{
		CurrentCampaign: &repoaudit.RepositoryReviewCampaignCoverage{
			ID: campaignID, CommitSHA: strings.Repeat("d", 40),
			InventoryHash: "inventory", ProfileHash: "profile", SelectedFiles: 4, Exact: true,
			Paths: map[string]repoaudit.RepositoryReviewCampaignPathCoverage{
				files[0].Path: {Inspected: true, Completed: true},
				files[1].Path: {Inspected: true},
				files[2].Path: {Unsupported: true},
			},
		},
		Contexts: []repoaudit.FindingContext{
			{
				ID: "ctx-a", CampaignID: campaignID, Model: "review-b", ModelAlias: "review-a",
				Account: "account-a", Reviewer: "review-b", Files: []repoaudit.FileRef{files[0]},
			},
			{
				ID: "ctx-b", CampaignID: campaignID, Model: "provider/model-b", ModelAlias: "review-b",
				Account: "account-b", Files: []repoaudit.FileRef{files[1]},
			},
			{ID: "ctx-old", CampaignID: repoaudit.NewRepositoryReviewCampaignID(), Reviewer: "review-a"},
		},
		Findings: []repoaudit.Finding{
			{
				ID: "finding-a", CampaignID: campaignID, RepositoryFindingID: "aggregate-a",
				Observations: []repoaudit.FindingObservation{{
					ContextID: "ctx-a", Model: "review-b", ModelAlias: "review-a",
					Account: "account-a", Reviewer: "review-b",
				}},
			},
			{
				ID: "finding-b", CampaignID: campaignID,
				Observations: []repoaudit.FindingObservation{{
					ContextID: "ctx-b", Model: "provider/model-b", ModelAlias: "review-b", Account: "account-b",
				}},
			},
			{ID: "finding-old", CampaignID: repoaudit.NewRepositoryReviewCampaignID()},
		},
	}
	automation := repoaudit.RepositoryReviewAutomation{
		CampaignID: campaignID, ReviewerModels: []string{"review-a", "review-b", "review-none"},
		ModelStats: make(map[string]repoaudit.RepositoryReviewModelStats),
		Progress: repoaudit.RepositoryReviewProgress{
			ReviewedFiles: 9, RemainingFiles: 9, UnsupportedFiles: 9,
		},
	}
	outcome := loadRepositoryReviewCampaignOutcome(state, automation)
	if !outcome.found || !outcome.coverageAvailable || !outcome.coverageExact ||
		outcome.selectedFiles != 4 || outcome.inspectedFiles != 2 || outcome.reviewedFiles != 1 ||
		outcome.remainingFiles != 2 || outcome.unsupportedFiles != 1 || outcome.rawFindings != 2 ||
		outcome.deduplicatedFindings != 0 || outcome.findings != 0 ||
		outcome.findingAggregates != 0 || outcome.pendingFindingMappings != 0 ||
		outcome.modelFindings["review-a"] != 1 || outcome.modelFindings["review-b"] != 1 ||
		outcome.modelFindings["review-none"] != 0 ||
		!reflect.DeepEqual(outcome.modelPaths["review-a"], []string{"a.go"}) ||
		!reflect.DeepEqual(outcome.modelPaths["review-b"], []string{"b.go"}) {
		t.Fatalf("campaign outcome=%#v", outcome)
	}
	applyRepositoryReviewLiveMetrics(&automation, state)
	if !automation.Progress.CoverageExact || automation.Progress.SelectedFiles != 4 ||
		automation.Progress.InspectedFiles != 2 || automation.Progress.ReviewedFiles != 1 ||
		automation.Progress.RemainingFiles != 2 || automation.Progress.UnsupportedFiles != 1 ||
		automation.Progress.RawFindings != 2 || automation.Progress.DeduplicatedFindings != 0 ||
		automation.Progress.Findings != 0 || automation.Progress.FindingAggregates != 0 ||
		automation.Progress.PendingFindingMappings != 0 ||
		automation.ModelStats["review-a"].Findings != 1 ||
		automation.ModelStats["review-b"].ReviewedFiles != 1 ||
		automation.ModelCoverageSketches["review-b"] == "" {
		t.Fatalf("campaign live progress=%#v stats=%#v", automation.Progress, automation.ModelStats)
	}

	state.CurrentCampaign.Exact = false
	automation.Progress.ReviewedFiles = 8
	automation.Progress.RemainingFiles = 7
	automation.Progress.UnsupportedFiles = 6
	applyRepositoryReviewLiveMetrics(&automation, state)
	if automation.Progress.CoverageExact || automation.Progress.ReviewedFiles != 8 ||
		automation.Progress.RemainingFiles != 7 || automation.Progress.UnsupportedFiles != 6 ||
		automation.Progress.SelectedFiles != 4 || automation.Progress.InspectedFiles != 2 {
		t.Fatalf("inexact campaign overwrote operational progress=%#v", automation.Progress)
	}
	applyRepositoryReviewLiveMetrics(nil, state)
	applyRepositoryReviewOutcome(nil, outcome)
	unchanged := automation
	applyRepositoryReviewOutcome(&unchanged, repositoryReviewOutcome{})
	if !reflect.DeepEqual(unchanged, automation) {
		t.Fatal("empty outcome mutated automation")
	}
}

func TestRepositoryReviewPreparedCampaignUsesLegacyMembershipUntilCoverageBinds(t *testing.T) {
	campaignID := repoaudit.NewRepositoryReviewCampaignID()
	startedAt := time.Now().Add(-time.Hour)
	automation := repoaudit.RepositoryReviewAutomation{
		CampaignID: campaignID, CampaignRecoveryPending: true,
		RunIDs: []string{"wr_legacy"}, StartedAt: startedAt,
	}
	state := repoaudit.RepositoryState{
		CurrentCampaign: &repoaudit.RepositoryReviewCampaignCoverage{
			ID: campaignID, CommitSHA: strings.Repeat("a", 40),
			Paths: map[string]repoaudit.RepositoryReviewCampaignPathCoverage{},
		},
		Runs: []repoaudit.ReviewRun{{
			ID: "wr_legacy", FindingIDs: []string{"finding"}, CompletedAt: startedAt.Add(time.Minute),
		}},
		Findings: []repoaudit.Finding{{ID: "finding"}},
	}
	if findings := repositoryReviewCurrentFindings(automation, state); len(findings) != 1 {
		t.Fatalf("prepared campaign findings=%#v", findings)
	}
	applyRepositoryReviewLiveMetrics(&automation, state)
	if automation.Progress.RawFindings != 1 || automation.Progress.DeduplicatedFindings != 0 ||
		automation.Progress.Findings != 0 || automation.Progress.CoverageAvailable {
		t.Fatalf("prepared campaign progress=%#v", automation.Progress)
	}
	fresh := automation
	fresh.CampaignID = repoaudit.NewRepositoryReviewCampaignID()
	fresh.CampaignRecoveryPending = false
	fresh.Progress.Findings = 99
	if findings := repositoryReviewCurrentFindings(fresh, state); len(findings) != 0 {
		t.Fatalf("fresh unbound campaign resurrected legacy findings=%#v", findings)
	}
	applyRepositoryReviewLiveMetrics(&fresh, state)
	if fresh.Progress.Findings != 0 || fresh.Progress.CoverageAvailable || fresh.Progress.CoverageExact {
		t.Fatalf("fresh unbound campaign progress=%#v", fresh.Progress)
	}
	state.CurrentCampaign.InventoryHash = strings.Repeat("b", 64)
	state.CurrentCampaign.ProfileHash = strings.Repeat("c", 64)
	state.Findings[0].CampaignID = campaignID
	automation.CampaignRecoveryPending = false
	if findings := repositoryReviewCurrentFindings(automation, state); len(findings) != 1 ||
		findings[0].CampaignID != campaignID {
		t.Fatalf("bound campaign findings=%#v", findings)
	}
}

func TestRepositoryReviewCoverageQuotaRemainingBranches(t *testing.T) {
	if got := normalizeRepositoryReviewWindow("7d"); got != "weekly" {
		t.Fatalf("7d window=%q", got)
	}
	if got := normalizeRepositoryReviewWindow("24h"); got != "daily" {
		t.Fatalf("24h window=%q", got)
	}
	reset := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	if parsed, ok := parseRepositoryReviewReset(reset.Format(time.RFC3339)); !ok || !parsed.Equal(reset) {
		t.Fatalf("reset parsed=%s ok=%v", parsed, ok)
	}
}

func TestRepositoryReviewCoverageSharedAPIBoundaries(t *testing.T) {
	cache := &systemVersionCache{}
	started, available := cache.waitOrStart(nil)
	if !started || !available {
		t.Fatalf("nil-context version cache admission = (%v, %v)", started, available)
	}
	cache.finishResolve(systemVersionResponse{}, 0, false)
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if started, available := cache.waitOrStart(canceled); started || available {
		t.Fatalf("canceled version cache admission = (%v, %v)", started, available)
	}

	handler := &Handler{}
	response := httptest.NewRecorder()
	handler.handleUpdate(
		response,
		httptest.NewRequest(http.MethodGet, "/api/update", nil),
	)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET update status = %d", response.Code)
	}

	malformed := httptest.NewRecorder()
	handler.handleUpdate(
		malformed,
		httptest.NewRequest(http.MethodPost, "/api/update", strings.NewReader("{")),
	)
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("malformed update status = %d body = %s", malformed.Code, malformed.Body.String())
	}
}

func TestRepositoryReviewCoverageWorkflowReconciliationProjection(t *testing.T) {
	testedAt := time.Date(2020, time.August, 31, 12, 0, 0, 0, time.UTC)
	lastTest := &workflows.WorkflowDevelopmentTest{
		RunID: "wr_previous", Status: workflows.RunStatusRunning,
		Error: "private diagnostic", TestedAt: testedAt,
	}
	session := &workflows.WorkflowDevelopmentSession{
		Status:   workflows.WorkflowDevelopmentStatusTesting,
		LastTest: lastTest,
	}

	projected := projectWorkflowDevelopmentReconciliationFailure(session, "wr_terminal")
	if projected == session || projected.LastTest == lastTest ||
		projected.Status != workflows.WorkflowDevelopmentStatusEditing ||
		projected.LastTest == nil || projected.LastTest.RunID != "wr_terminal" ||
		projected.LastTest.Status != "reconciliation_failed" ||
		projected.LastTest.Error == "" || !projected.LastTest.TestedAt.After(testedAt) {
		t.Fatalf("reconciliation projection=%#v", projected)
	}
	if session.Status != workflows.WorkflowDevelopmentStatusTesting ||
		lastTest.RunID != "wr_previous" || lastTest.Status != workflows.RunStatusRunning ||
		lastTest.Error != "private diagnostic" || !lastTest.TestedAt.Equal(testedAt) {
		t.Fatalf("reconciliation projection mutated source session=%#v", session)
	}
}

func TestRepositoryReviewCoverageManagedChildOutcomeBranches(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, storeErr := handler.repositoryReviewStore()
	if storeErr != nil {
		t.Fatal(storeErr)
	}
	controller := newRepositoryReviewController(handler)
	controller.recordManagedChildOutcomes("missing", "run", nil, nil)
	controller.recordManagedChildOutcomes(
		"missing",
		"run",
		&workflows.Run{Steps: map[string]workflows.StepExecution{"review": {Outputs: map[string]any{}}}},
		nil,
	)

	automation := repositoryReviewCoverageRunningAutomation(t, store, "run-managed-branches", false)
	controller.active[automation.ID] = &repositoryReviewActiveRun{runID: automation.ActiveRunID, store: store}
	index := map[string]repositoryReviewAccountingModel{
		"selected-model": {alias: "cheap", known: true},
		"fallback":       {alias: "cheap", known: true},
		"":               {alias: "cheap", known: true},
	}
	children := []any{
		"not-a-map",
		map[string]any{"model": map[string]any{"selected": "missing"}, "admitted": true, "valid": true},
		map[string]any{"model": map[string]any{"selected": "selected-model"}, "admitted": false},
		map[string]any{
			"model":    map[string]any{"requested": nil, "selected": "selected-model"},
			"admitted": true, "valid": false,
		},
		map[string]any{
			"model":    map[string]any{"requested": "selected-model"},
			"admitted": true, "valid": true, "run_error": errors.New("child failed"),
		},
		map[string]any{
			"model":    map[string]any{"requested": "selected-model"},
			"admitted": true, "valid": true,
			"scope":      []any{map[string]any{"path": "a.go"}, map[string]any{"path": ""}, "bad-scope"},
			"structured": map[string]any{"reviewed_files": []any{"a.go", "outside.go"}},
		},
	}
	run := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"review": {Outputs: map[string]any{"managed_children": children}},
	}}
	controller.recordManagedChildOutcomes(automation.ID, automation.ActiveRunID, run, index)
	updated, found, getErr := store.GetAutomation(t.Context(), automation.ID)
	if getErr != nil || !found {
		t.Fatalf("managed automation found=%v err=%v", found, getErr)
	}
	stats := updated.ModelStats["cheap"]
	if stats.Failures != 2 || stats.Requests < stats.Failures || stats.ReviewedFiles < 1 {
		t.Fatalf("managed child stats=%#v automation=%#v", stats, updated)
	}

	delete(controller.active, automation.ID)
	controller.recordManagedChildOutcomes(automation.ID, automation.ActiveRunID, run, index)
	controller.requestSafeStop("missing", "run", repoaudit.RepositoryReviewPauseManual, "stop")
}

func TestRepositoryReviewCoverageReconcileBranches(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, storeErr := handler.repositoryReviewStore()
	if storeErr != nil {
		t.Fatal(storeErr)
	}
	cfg, loadErr := config.LoadConfig(handler.configPath)
	if loadErr != nil {
		t.Fatal(loadErr)
	}

	nilConfigController := newRepositoryReviewController(handler)
	nilConfigController.reconcile()

	stale := repositoryReviewCoverageRunningAutomation(t, store, "run-stale-reconcile", false)
	controller := newRepositoryReviewController(handler)
	controller.leasedStore = store
	controller.leasedConfig = cfg
	controller.now = func() time.Time { return time.Now().UTC() }
	controller.reconcile()
	stale, found, getErr := store.GetAutomation(t.Context(), stale.ID)
	if getErr != nil || !found || stale.Status != repoaudit.RepositoryReviewAutomationPaused ||
		stale.PauseReason != repoaudit.RepositoryReviewPauseServiceRestart {
		t.Fatalf("stale reconcile=%#v found=%v err=%v", stale, found, getErr)
	}

	stoppingInput := testRepositoryReviewAutomation()
	stoppingInput.Status = repoaudit.RepositoryReviewAutomationStopping
	stoppingInput.ActiveRunID = "run-stopping-reconcile"
	stoppingInput.RunIDs = []string{stoppingInput.ActiveRunID}
	stoppingInput.RequestedPauseReason = repoaudit.RepositoryReviewPauseManual
	stoppingInput.RequestedPauseDetail = "requested manual pause"
	stopping, createErr := store.CreateAutomation(t.Context(), stoppingInput)
	if createErr != nil {
		t.Fatal(createErr)
	}
	controller.reconcile()
	stopping, found, getErr = store.GetAutomation(t.Context(), stopping.ID)
	if getErr != nil || !found || stopping.Status != repoaudit.RepositoryReviewAutomationPaused ||
		stopping.PauseReason != repoaudit.RepositoryReviewPauseManual {
		t.Fatalf("stopping reconcile=%#v found=%v err=%v", stopping, found, getErr)
	}

	quotaInput := testRepositoryReviewAutomation()
	quotaInput.Status = repoaudit.RepositoryReviewAutomationRunning
	quotaInput.ActiveRunID = "run-quota-reconcile"
	quotaInput.RunIDs = []string{quotaInput.ActiveRunID}
	quotaInput.BudgetPolicy.GuardExpression = "account.limits.weekly.remaining_percent > 10"
	quota, createErr := store.CreateAutomation(t.Context(), quotaInput)
	if createErr != nil {
		t.Fatal(createErr)
	}
	controller.active[quota.ID] = &repositoryReviewActiveRun{
		runID: quota.ActiveRunID, store: store, config: cfg,
		reservations: make(map[int]repositoryReviewTaskReservation),
	}
	controller.probe = func(context.Context) (codexAccountLimitsResponse, error) {
		return codexAccountLimitsResponse{Accounts: []codexAccountLimitAccount{{
			ID: "work", Entries: []codexAccountLimitEntry{{
				Status: "available", Window: "weekly", UsedPercent: ptrInt(100),
			}},
		}}}, nil
	}
	_ = controller.observeRepositoryReviewTask(
		quota.ID, quota.ActiveRunID,
		workflows.ManagedChildActivity{Phase: workflows.ManagedChildStarted, Index: 1},
	)
	quota, found, getErr = store.GetAutomation(t.Context(), quota.ID)
	if getErr != nil || !found || quota.Status != repoaudit.RepositoryReviewAutomationStopping ||
		quota.RequestedPauseReason != repoaudit.RepositoryReviewPauseGuardExpression {
		t.Fatalf("quota reconcile=%#v found=%v err=%v", quota, found, getErr)
	}

	badWorkspace := t.TempDir()
	if writeErr := os.WriteFile(
		filepath.Join(badWorkspace, "repository_reviews"),
		[]byte("not a directory"),
		0o600,
	); writeErr != nil {
		t.Fatal(writeErr)
	}
	badController := newRepositoryReviewController(handler)
	badController.leasedConfig = cfg
	badController.leasedStore = repoaudit.NewStore(badWorkspace)
	badController.reconcile()
}

func TestRepositoryReviewCompatibilityFindingDispatchCoverage(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	automation := seedRepositoryReviewDetailAutomation(
		t, handler, state.Repository, state.Runs[0].ID,
	)

	direct := func(automationID, findingID string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.SetPathValue("automation_id", automationID)
		request.SetPathValue("finding_id", findingID)
		response := httptest.NewRecorder()
		handler.handleGetRepositoryReviewAutomationFinding(response, request)
		return response
	}
	if response := direct("rra_missing", "rfn_missing"); response.Code != http.StatusNotFound {
		t.Fatalf("missing compatibility automation=%d %s", response.Code, response.Body.String())
	}
	if response := direct(automation.ID, state.DeduplicatedFindings[0].ID); response.Code != http.StatusOK {
		t.Fatalf("deduplicated compatibility detail=%d %s", response.Code, response.Body.String())
	}
	if response := direct(automation.ID, "unknown_finding"); response.Code != http.StatusNotFound {
		t.Fatalf("unknown compatibility finding=%d %s", response.Code, response.Body.String())
	}

	missingAlias := httptest.NewRecorder()
	mux.ServeHTTP(missingAlias, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations/"+automation.ID+"/run-findings/rfn_missing",
		nil,
	))
	if missingAlias.Code != http.StatusNotFound {
		t.Fatalf("missing legacy alias=%d %s", missingAlias.Code, missingAlias.Body.String())
	}

	canonical := httptest.NewRecorder()
	mux.ServeHTTP(canonical, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations/"+automation.ID+"/findings/"+
			state.DeduplicatedFindings[0].ID,
		nil,
	))
	if canonical.Code != http.StatusOK ||
		!strings.Contains(canonical.Body.String(), `"repository_finding"`) {
		t.Fatalf("mapped deduplicated detail=%d %s", canonical.Code, canonical.Body.String())
	}
}

func TestRepositoryReviewCoverageControllerLifecycleBoundaries(t *testing.T) {
	var nilHandler *Handler
	nilHandler.StartRepositoryReviewController()
	nilHandler.stopRepositoryReviewController()
	if nilHandler.repositoryReviewControllerInstance() != nil {
		t.Fatal("nil handler created a controller")
	}

	var nilController *repositoryReviewController
	if startErr := nilController.Start(); startErr == nil {
		t.Fatal("nil controller started")
	}
	nilController.Stop()
	if _, _, storeErr := nilController.store(); storeErr == nil {
		t.Fatal("nil controller returned a store")
	}

	badHandler := NewHandler(t.TempDir())
	badHandler.StartRepositoryReviewController()
	badHandler.stopRepositoryReviewController()

	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := newRepositoryReviewController(handler)
	controller.stopped = true
	if startErr := controller.Start(); !errors.Is(startErr, context.Canceled) {
		t.Fatalf("stopped controller start error=%v", startErr)
	}
	if _, configErr := controller.currentLeasedConfiguration(); configErr == nil {
		t.Fatal("controller without lease returned config")
	}

	leased, loadErr := config.LoadConfig(handler.configPath)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	changed := *leased
	changed.Agents.Defaults.Workspace = t.TempDir()
	controller = newRepositoryReviewController(handler)
	controller.leasedConfig = &changed
	if _, configErr := controller.currentLeasedConfiguration(); configErr == nil ||
		!strings.Contains(configErr.Error(), "workspace changed") {
		t.Fatalf("changed workspace config error=%v", configErr)
	}

	canceledController := newRepositoryReviewController(handler)
	canceledController.cancel()
	canceledController.wg.Add(1)
	canceledController.monitor()
}

func TestRepositoryReviewCoverageFinishRunFailedPause(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, storeErr := handler.repositoryReviewStore()
	if storeErr != nil {
		t.Fatal(storeErr)
	}
	automation := repositoryReviewCoverageRunningAutomation(t, store, "run-accounting-failed", false)
	controller := newRepositoryReviewController(handler)
	controller.active[automation.ID] = &repositoryReviewActiveRun{
		runID: automation.ActiveRunID, store: store,
		pauseReason: repoaudit.RepositoryReviewPauseRunFailed,
		pauseDetail: "usage accounting failed",
	}
	controller.finishAutomationRun(
		automation.ID,
		automation.ActiveRunID,
		&workflows.RunResult{Status: workflows.RunStatusSucceeded},
		nil,
		true,
		nil,
	)
	updated, found, getErr := store.GetAutomation(t.Context(), automation.ID)
	if getErr != nil || !found || updated.Status != repoaudit.RepositoryReviewAutomationFailed ||
		!strings.Contains(updated.PauseDetail, "accounting") {
		t.Fatalf("accounting-failed finish=%#v found=%v err=%v", updated, found, getErr)
	}
}

func repositoryReviewCoverageLeasedController(
	t *testing.T,
	handler *Handler,
	store repoaudit.Store,
) *repositoryReviewController {
	t.Helper()
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	controller := newRepositoryReviewController(handler)
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		return strings.Repeat("a", 40), nil
	}
	controller.startOnce.Do(func() {})
	controller.leasedStore = store
	controller.leasedConfig = cfg
	t.Cleanup(controller.Stop)
	return controller
}

func TestRepositoryReviewCoverageModelOptionEdgeBranches(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.AccountRef = "review-router"
	cfg.AccountRouters = []config.AccountRouterConfig{{
		Name: "review-router", Enabled: true, Entry: "pool",
		Blocks: []config.AccountRouterBlock{{
			ID:       "pool",
			Type:     config.AccountRouterBlockTypeLoadBalance,
			Accounts: []string{"api", "missing", "dynamic"},
		}},
	}}
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "api", Provider: "openai", Model: "openai/base", Enabled: true,
			InputPricePerMTok: 1.5, OutputPricePerMTok: 6,
		},
		{ModelName: "dynamic", Provider: config.ModelRouterProvider, Enabled: true},
	}
	cfg.ModelAliases = []config.ModelAliasConfig{
		{Name: "plain", Model: "plain-model"},
		{Name: "disabled", Model: "openai/disabled", DisabledAccounts: []string{"api"}},
	}
	options := repositoryReviewModelOptions(cfg)
	byAlias := make(map[string]repositoryReviewModelOption, len(options))
	for _, option := range options {
		byAlias[option.Alias] = option
	}
	if plain := byAlias["plain"]; !plain.Available || !plain.PriceKnown || plain.Provider != "openai" {
		t.Fatalf("plain option=%#v", plain)
	}
	if disabled := byAlias["disabled"]; disabled.Available || disabled.BlockedReason == "" {
		t.Fatalf("disabled option=%#v", disabled)
	}

	embedded := config.DefaultConfig()
	embedded.Agents.Defaults.AccountRef = "embedded-router"
	embedded.ModelList = []*config.ModelConfig{{
		ModelName: "embedded-router", Enabled: true,
		Router: &config.AccountRouterConfig{Entry: "entry", Blocks: []config.AccountRouterBlock{{
			ID: "entry", Type: config.AccountRouterBlockTypeAccount, Account: "direct",
		}}},
	}}
	if refs := repositoryReviewRuntimeAccountRefs(embedded); !reflect.DeepEqual(refs, []string{"direct"}) {
		t.Fatalf("embedded router refs=%#v", refs)
	}

	blankIndex := repositoryReviewAccountingIndex(
		&config.Config{ModelAliases: []config.ModelAliasConfig{{Name: "blank", Model: "  "}}},
		repoaudit.RepositoryReviewAutomation{ReviewerModels: []string{"blank"}},
	)
	if _, exists := blankIndex["blank"]; !exists {
		t.Fatalf("blank accounting index=%#v", blankIndex)
	}
}

func TestRepositoryReviewCoverageOptionsExposeCredentialLoadFailure(t *testing.T) {
	authHome := t.TempDir()
	t.Setenv("PICOCLAW_HOME", authHome)
	if err := os.WriteFile(filepath.Join(authHome, "auth.json"), []byte(`{`), 0o600); err != nil {
		t.Fatal(err)
	}
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automation-options",
		nil,
	))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "limits_error") {
		t.Fatalf("options credential failure=%d %s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewCoverageControllerTransitionEdges(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	createIdle := func(t *testing.T) repoaudit.RepositoryReviewAutomation {
		t.Helper()
		created, createErr := store.CreateAutomation(t.Context(), testRepositoryReviewAutomation())
		if createErr != nil {
			t.Fatal(createErr)
		}
		return created
	}

	controller := repositoryReviewCoverageLeasedController(t, handler, store)
	idle := createIdle(t)
	directCanceled := repositoryReviewCoverageLeasedController(t, handler, store)
	directCanceled.cancel()
	if _, startErr := directCanceled.startAutomation(
		t.Context(), idle.ID, idle.Version, false, "start",
	); !errors.Is(startErr, context.Canceled) {
		t.Fatalf("direct canceled start error=%v", startErr)
	}
	if _, startErr := controller.startAutomation(
		t.Context(),
		idle.ID,
		idle.Version+1,
		false,
		"start",
	); !errors.Is(
		startErr,
		repoaudit.ErrConflict,
	) {
		t.Fatalf("stale start error=%v", startErr)
	}
	if _, startErr := controller.startAutomation(
		t.Context(),
		idle.ID,
		idle.Version,
		false,
		"invalid",
	); !errors.Is(
		startErr,
		errRepositoryReviewInvalidTransition,
	) {
		t.Fatalf("invalid action error=%v", startErr)
	}
	controller.active[idle.ID] = &repositoryReviewActiveRun{runID: "already-active", store: store}
	if _, startErr := controller.startAutomation(
		t.Context(),
		idle.ID,
		idle.Version,
		false,
		"start",
	); !errors.Is(
		startErr,
		errRepositoryReviewAutomationBusy,
	) {
		t.Fatalf("locally active start error=%v", startErr)
	}
	delete(controller.active, idle.ID)

	resetFailure := repositoryReviewCoverageLeasedController(t, handler, store)
	resetFailure.update = func(
		context.Context,
		repoaudit.Store,
		string,
		int64,
		func(*repoaudit.RepositoryReviewAutomation) error,
	) (repoaudit.RepositoryReviewAutomation, error) {
		return repoaudit.RepositoryReviewAutomation{}, errors.New("reset persistence failed")
	}
	if _, startErr := resetFailure.startAutomation(
		t.Context(), idle.ID, idle.Version, true, "start",
	); startErr == nil || !strings.Contains(startErr.Error(), "reset persistence") {
		t.Fatalf("reset persistence error=%v", startErr)
	}

	transitionFailure := repositoryReviewCoverageLeasedController(t, handler, store)
	transitionFailure.update = func(
		context.Context,
		repoaudit.Store,
		string,
		int64,
		func(*repoaudit.RepositoryReviewAutomation) error,
	) (repoaudit.RepositoryReviewAutomation, error) {
		return repoaudit.RepositoryReviewAutomation{}, errors.New("transition persistence failed")
	}
	if _, startErr := transitionFailure.startAutomation(
		t.Context(), idle.ID, idle.Version, false, "start",
	); startErr == nil || !strings.Contains(startErr.Error(), "transition persistence") {
		t.Fatalf("transition persistence error=%v", startErr)
	}

	canceled := createIdle(t)
	cancelController := repositoryReviewCoverageLeasedController(t, handler, store)
	cancelController.cancel()
	if _, startErr := cancelController.startAutomation(
		t.Context(), canceled.ID, canceled.Version, false, "start",
	); !errors.Is(startErr, context.Canceled) {
		t.Fatalf("canceled admission error=%v", startErr)
	}

	raced := createIdle(t)
	raceController := repositoryReviewCoverageLeasedController(t, handler, store)
	raceController.mu.Lock()
	raceController.active[raced.ID] = &repositoryReviewActiveRun{runID: "won-race", store: store}
	raceController.mu.Unlock()
	if _, startErr := raceController.startAutomation(
		t.Context(),
		raced.ID,
		raced.Version,
		false,
		"start",
	); !errors.Is(
		startErr,
		errRepositoryReviewAutomationBusy,
	) {
		t.Fatalf("raced admission error=%v", startErr)
	}

	stoppingInput := testRepositoryReviewAutomation()
	stoppingInput.Status = repoaudit.RepositoryReviewAutomationStopping
	stoppingInput.ActiveRunID = "run-already-stopping"
	stoppingInput.RunIDs = []string{stoppingInput.ActiveRunID}
	stoppingInput.RequestedPauseReason = repoaudit.RepositoryReviewPauseManual
	stopping, err := store.CreateAutomation(t.Context(), stoppingInput)
	if err != nil {
		t.Fatal(err)
	}
	pauseController := repositoryReviewCoverageLeasedController(t, handler, store)
	paused, pauseErr := pauseController.pauseAutomation(t.Context(), stopping.ID, stopping.Version)
	if pauseErr != nil || paused.Status != repoaudit.RepositoryReviewAutomationStopping {
		t.Fatalf("repeat pause=%#v err=%v", paused, pauseErr)
	}

	canceledPause := repositoryReviewCoverageLeasedController(t, handler, store)
	canceledPause.cancel()
	if _, pauseErr := canceledPause.pauseAutomation(
		t.Context(),
		stopping.ID,
		stopping.Version,
	); !errors.Is(
		pauseErr,
		context.Canceled,
	) {
		t.Fatalf("canceled pause error=%v", pauseErr)
	}
}

func TestRepositoryReviewCoverageConfigurationAndStoreErrors(t *testing.T) {
	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	controller := repositoryReviewCoverageLeasedController(t, handler, store)
	valid, createErr := store.CreateAutomation(t.Context(), testRepositoryReviewAutomation())
	if createErr != nil {
		t.Fatal(createErr)
	}
	if _, startErr := controller.startAutomation(t.Context(), "invalid", 1, false, "start"); startErr == nil {
		t.Fatal("invalid automation ID started")
	}

	if err := os.WriteFile(handler.configPath, []byte(`{`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, configErr := controller.currentLeasedConfiguration(); configErr == nil {
		t.Fatal("corrupt leased configuration loaded")
	}
	if _, startErr := controller.startAutomation(
		t.Context(), valid.ID, valid.Version, false, "start",
	); startErr == nil {
		t.Fatal("automation started with corrupt current configuration")
	}

	badWorkspace := t.TempDir()
	badStore := repoaudit.NewStore(badWorkspace)
	bad := repositoryReviewCoverageRunningAutomation(t, badStore, "run-corrupt-usage", false)
	badPath := filepath.Join(
		badWorkspace,
		"repository_reviews",
		"automation_"+bad.ID+".json",
	)
	if err := os.WriteFile(badPath, []byte(`{`), 0o600); err != nil {
		t.Fatal(err)
	}
	accountingController := newRepositoryReviewController(nil)
	accountingController.active[bad.ID] = &repositoryReviewActiveRun{
		runID: bad.ActiveRunID, store: badStore,
	}
	usageErr := accountingController.recordUsage(
		bad.ID,
		bad.ActiveRunID,
		workflows.AgentUsage{PromptTokens: 1, TotalTokens: 1},
		nil,
	)
	if !errors.Is(usageErr, errRepositoryReviewSafeStop) ||
		accountingController.active[bad.ID].pauseReason != repoaudit.RepositoryReviewPauseRunFailed {
		t.Fatalf("corrupt accounting error=%v active=%#v", usageErr, accountingController.active[bad.ID])
	}

	_ = workspace
}

func TestRepositoryReviewCoverageAccountingAndRetryEdges(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	controller := newRepositoryReviewController(handler)
	if usageErr := controller.recordUsage(
		"rra_missing", "run-missing", workflows.AgentUsage{TotalTokens: 1}, nil,
	); !errors.Is(usageErr, errRepositoryReviewSafeStop) {
		t.Fatalf("missing active accounting error=%v", usageErr)
	}

	mismatch := repositoryReviewCoverageRunningAutomation(t, store, "run-persisted", false)
	controller.active[mismatch.ID] = &repositoryReviewActiveRun{runID: "run-observer", store: store}
	if usageErr := controller.recordUsage(
		mismatch.ID,
		"run-observer",
		workflows.AgentUsage{PromptTokens: 4, TotalTokens: 4},
		map[string]repositoryReviewAccountingModel{"": {alias: "cheap", known: true}},
	); usageErr != nil {
		t.Fatalf("mismatched run accounting error=%v", usageErr)
	}
	unchanged, found, getErr := store.GetAutomation(t.Context(), mismatch.ID)
	if getErr != nil || !found || unchanged.Usage.TotalTokens != 0 {
		t.Fatalf("mismatched run persisted=%#v found=%v err=%v", unchanged, found, getErr)
	}

	conflicted, createErr := store.CreateAutomation(t.Context(), testRepositoryReviewAutomation())
	if createErr != nil {
		t.Fatal(createErr)
	}
	attempts := 0
	if _, retryErr := controller.updateLatest(
		t.Context(),
		store,
		conflicted.ID,
		func(*repoaudit.RepositoryReviewAutomation) error {
			attempts++
			return repoaudit.ErrConflict
		},
	); !errors.Is(retryErr, repoaudit.ErrConflict) ||
		attempts != 12 {
		t.Fatalf("conflict retries=%d err=%v", attempts, retryErr)
	}

	invalidOnly := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"review": {Outputs: map[string]any{"managed_children": []any{"invalid-child"}}},
	}}
	controller.recordManagedChildOutcomes("missing", "run", invalidOnly, nil)
	missingAccounting := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"review": {Outputs: map[string]any{"managed_children": []any{map[string]any{
			"model": map[string]any{"selected": "unpriced"}, "admitted": true, "valid": true,
		}}}},
	}}
	controller.recordManagedChildOutcomes("missing", "run", missingAccounting, nil)

	childMismatch := repositoryReviewCoverageRunningAutomation(t, store, "run-child-persisted", false)
	controller.active[childMismatch.ID] = &repositoryReviewActiveRun{runID: "run-child-observer", store: store}
	validChild := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"review": {Outputs: map[string]any{"managed_children": []any{map[string]any{
			"model": map[string]any{"selected": "selected"}, "admitted": true, "valid": false,
		}}}},
	}}
	controller.recordManagedChildOutcomes(
		childMismatch.ID,
		"run-child-observer",
		validChild,
		map[string]repositoryReviewAccountingModel{"selected": {alias: "cheap", known: true}},
	)
	childUnchanged, found, getErr := store.GetAutomation(t.Context(), childMismatch.ID)
	if getErr != nil || !found || childUnchanged.ModelStats["cheap"].Failures != 0 {
		t.Fatalf("mismatched child outcome=%#v found=%v err=%v", childUnchanged, found, getErr)
	}
}

func TestRepositoryReviewCoverageOutcomeSelectionEdges(t *testing.T) {
	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	seed := seedRepositoryReviewAPIState(t, workspace)
	store := repoaudit.NewStore(workspace)

	future := loadRepositoryReviewOutcome(store, repoaudit.RepositoryReviewAutomation{
		Repository: seed.Repository,
		RunIDs:     []string{"api-run"},
		StartedAt:  time.Now().UTC().Add(time.Hour),
	})
	if future.found {
		t.Fatalf("future campaign outcome=%#v", future)
	}

	code := repoaudit.FileRef{
		Path: "pkg/edge.go", BlobSHA: strings.Repeat("b", 40), SizeBytes: 80,
		Category: "code", Mode: "100644",
	}
	binary := repoaudit.FileRef{
		Path: "assets/edge.bin", BlobSHA: strings.Repeat("c", 40), SizeBytes: 16,
		Category: "binary", Mode: "100644",
	}
	plan, planErr := store.Plan(
		t.Context(), seed.Repository, "commit-edge", "inventory-edge", []repoaudit.FileRef{code, binary}, false,
	)
	if planErr != nil {
		t.Fatal(planErr)
	}
	line := 7
	recorded, recordErr := store.Record(t.Context(), repoaudit.RecordRequest{
		Plan: plan, RunID: "edge-run",
		UnsupportedFiles: []repoaudit.UnsupportedFile{{FileRef: binary, Reason: "binary fixture"}},
		Observations: []repoaudit.Observation{{
			Model: "edge-model", Reviewer: "edge-reviewer", ScopeFiles: []repoaudit.FileRef{code},
			Findings: []repoaudit.FindingCandidate{{
				Severity: "medium", Title: "Edge finding", File: code.Path, Line: &line,
				Evidence: "edge evidence", Impact: "edge impact",
				Validation: repoaudit.Validation{Status: "confirmed", Summary: "confirmed"},
			}},
		}},
	})
	if recordErr != nil {
		t.Fatal(recordErr)
	}
	outcome := loadRepositoryReviewOutcome(store, repoaudit.RepositoryReviewAutomation{
		Repository: seed.Repository, RunIDs: []string{"edge-run"}, ReviewerModels: []string{"other-model"},
	})
	if !outcome.found || outcome.unsupportedFiles != 1 || outcome.findings != 1 ||
		outcome.modelFindings["other-model"] != 0 {
		t.Fatalf("selected outcome=%#v state=%#v", outcome, recorded.State)
	}
	completeRepositoryReviewAPIMappingJobs(t, workspace, recorded.State)
	mapped := loadRepositoryReviewOutcome(store, repoaudit.RepositoryReviewAutomation{
		Repository: seed.Repository, RunIDs: []string{"edge-run"},
	})
	if mapped.findingAggregates != 1 || mapped.pendingFindingMappings != 0 {
		t.Fatalf("mapped outcome=%#v", mapped)
	}

	automation := repoaudit.RepositoryReviewAutomation{
		ModelStats:            make(map[string]repoaudit.RepositoryReviewModelStats),
		ModelCoverageSketches: make(map[string]string),
	}
	addRepositoryReviewModelPaths(&automation, "edge", []string{" ", "pkg/edge.go"})
	if automation.ModelStats["edge"].ReviewedFiles != 1 {
		t.Fatalf("blank-path sketch stats=%#v", automation.ModelStats["edge"])
	}
}

func TestRepositoryReviewCoverageQuotaProbeAndReconcileErrors(t *testing.T) {
	withPicoclawAuthHome(t)
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	controller := repositoryReviewCoverageLeasedController(t, handler, store)
	controller.probe = nil
	quotaInput := testRepositoryReviewAutomation()
	if snapshots, _, quotaErr := controller.repositoryReviewGuardAccountLimits(
		t.Context(), controller.leasedConfig, quotaInput,
	); quotaErr == nil && len(snapshots) == 0 {
		t.Fatalf("default empty quota snapshots=%#v err=%v", snapshots, quotaErr)
	}

	runningInput := testRepositoryReviewAutomation()
	runningInput.Status = repoaudit.RepositoryReviewAutomationRunning
	runningInput.ActiveRunID = "run-probe-error"
	runningInput.RunIDs = []string{runningInput.ActiveRunID}
	runningInput.BudgetPolicy.GuardExpression = "account.limits.weekly.remaining_percent > 10"
	running, createErr := store.CreateAutomation(t.Context(), runningInput)
	if createErr != nil {
		t.Fatal(createErr)
	}
	controller.active[running.ID] = &repositoryReviewActiveRun{
		runID: running.ActiveRunID, store: store, config: controller.leasedConfig,
		reservations: make(map[int]repositoryReviewTaskReservation),
	}
	controller.probe = func(context.Context) (codexAccountLimitsResponse, error) {
		return codexAccountLimitsResponse{}, errors.New("telemetry offline")
	}
	_ = controller.observeRepositoryReviewTask(
		running.ID, running.ActiveRunID,
		workflows.ManagedChildActivity{Phase: workflows.ManagedChildStarted, Index: 1},
	)
	updated, found, getErr := store.GetAutomation(t.Context(), running.ID)
	if getErr != nil || !found || updated.Status != repoaudit.RepositoryReviewAutomationStopping ||
		updated.RequestedPauseReason != repoaudit.RepositoryReviewPauseGuardExpression {
		t.Fatalf("probe-error reconcile=%#v found=%v err=%v", updated, found, getErr)
	}

	canceledInput, createErr := store.CreateAutomation(t.Context(), testRepositoryReviewAutomation())
	if createErr != nil {
		t.Fatal(createErr)
	}
	canceledReconcile := repositoryReviewCoverageLeasedController(t, handler, store)
	canceledReconcile.now = func() time.Time {
		canceledReconcile.cancel()
		return time.Now().UTC()
	}
	canceledReconcile.reconcile()
	canceledInput, found, getErr = store.GetAutomation(context.Background(), canceledInput.ID)
	if getErr != nil || !found || canceledInput.Status != repoaudit.RepositoryReviewAutomationIdle {
		t.Fatalf("canceled reconcile automation=%#v found=%v err=%v", canceledInput, found, getErr)
	}
}

func TestRepositoryReviewCoverageWorkflowObservers(t *testing.T) {
	if limitsError := repositoryReviewLimitsError(
		codexAccountLimitsResponse{Error: "partial telemetry"}, nil,
	); limitsError != "partial telemetry" {
		t.Fatalf("projected limits error=%q", limitsError)
	}
	if limitsError := repositoryReviewLimitsError(codexAccountLimitsResponse{}, nil); limitsError != "" {
		t.Fatalf("empty limits error=%q", limitsError)
	}
	if reason, detail, pause := repositoryReviewFinalPause(
		"", "", repoaudit.RepositoryReviewAutomationStopping,
	); !pause || reason != repoaudit.RepositoryReviewPauseManual || detail == "" {
		t.Fatalf("stopping fallback reason=%q detail=%q pause=%v", reason, detail, pause)
	}
	if _, _, pause := repositoryReviewFinalPause(
		"", "", repoaudit.RepositoryReviewAutomationRunning,
	); pause {
		t.Fatal("running automation received a final pause")
	}

	usageCalls := 0
	wantUsageErr := errors.New("usage stopped")
	usageObserver := repositoryReviewAgentUsageObserver("run-target", func(usage workflows.AgentUsage) error {
		usageCalls++
		if usage.TotalTokens != 7 {
			t.Fatalf("usage=%#v", usage)
		}
		return wantUsageErr
	})
	if observeErr := usageObserver(workflows.AgentUsageEvent{
		RunID: "run-other", Usage: workflows.AgentUsage{TotalTokens: 3},
	}); observeErr != nil || usageCalls != 0 {
		t.Fatalf("unrelated usage calls=%d err=%v", usageCalls, observeErr)
	}
	if observeErr := usageObserver(workflows.AgentUsageEvent{
		RunID: "run-target", Usage: workflows.AgentUsage{TotalTokens: 7},
	}); !errors.Is(observeErr, wantUsageErr) || usageCalls != 1 {
		t.Fatalf("target usage calls=%d err=%v", usageCalls, observeErr)
	}

	admissionCalls := 0
	wantAdmissionErr := errors.New("admission stopped")
	admissionObserver := repositoryReviewAgentCallAdmissionObserver("run-target", func() error {
		admissionCalls++
		return wantAdmissionErr
	})
	if admitErr := admissionObserver(
		workflows.AgentCallAdmissionEvent{RunID: "run-other"},
	); admitErr != nil ||
		admissionCalls != 0 {
		t.Fatalf("unrelated admission calls=%d err=%v", admissionCalls, admitErr)
	}
	if admitErr := admissionObserver(
		workflows.AgentCallAdmissionEvent{RunID: "run-target"},
	); !errors.Is(admitErr, wantAdmissionErr) ||
		admissionCalls != 1 {
		t.Fatalf("target admission calls=%d err=%v", admissionCalls, admitErr)
	}
}

func TestRepositoryReviewCoverageProgressMonitorMissingAndDuplicateStage(t *testing.T) {
	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation := repositoryReviewCoverageRunningAutomation(t, store, "run-progress-edges", false)
	workflowStore := workflows.NewFileRunStore(workspace)
	controller := newRepositoryReviewController(handler)
	controller.progressEvery = 2 * time.Millisecond
	monitorCtx, cancelMonitor := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		controller.monitorWorkflowProgress(
			monitorCtx, store, workflowStore, automation.ID, automation.ActiveRunID,
		)
		close(done)
	}()
	time.Sleep(10 * time.Millisecond)
	if createErr := workflowStore.CreateRun(t.Context(), &workflows.Run{
		ID: automation.ActiveRunID, WorkflowRef: workflows.RepositoryBugFinderWorkflowRef,
		Status: workflows.RunStatusRunning,
		Steps: map[string]workflows.StepExecution{
			"review": {ID: "review", Status: workflows.RunStatusSucceeded},
		},
	}); createErr != nil {
		cancelMonitor()
		<-done
		t.Fatal(createErr)
	}
	deadline := time.Now().Add(time.Second)
	for {
		current, found, getErr := store.GetAutomation(t.Context(), automation.ID)
		if getErr != nil {
			cancelMonitor()
			<-done
			t.Fatal(getErr)
		}
		if found && current.Progress.Stage == "Reviewing bounded file batch" {
			break
		}
		if time.Now().After(deadline) {
			cancelMonitor()
			<-done
			t.Fatalf("progress stage never updated: %#v", current.Progress)
		}
		time.Sleep(2 * time.Millisecond)
	}
	time.Sleep(10 * time.Millisecond)
	cancelMonitor()
	<-done

	succeeded := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"record": {Status: workflows.RunStatusSucceeded},
	}}
	if stage := repositoryReviewWorkflowStage(succeeded); stage != "Checkpointing findings" {
		t.Fatalf("succeeded stage=%q", stage)
	}
}

func TestRepositoryReviewCoverageFinishMismatchedRunAndControllerTiming(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	mismatch := repositoryReviewCoverageRunningAutomation(t, store, "run-finish-persisted", false)
	controller := newRepositoryReviewController(handler)
	controller.active[mismatch.ID] = &repositoryReviewActiveRun{runID: "run-finish-observer", store: store}
	controller.finishAutomationRun(
		mismatch.ID,
		"run-finish-observer",
		&workflows.RunResult{Status: workflows.RunStatusSucceeded},
		nil,
		true,
		nil,
	)
	mismatch, found, getErr := store.GetAutomation(t.Context(), mismatch.ID)
	if getErr != nil || !found || mismatch.ActiveRunID != "run-finish-persisted" {
		t.Fatalf("mismatched finish=%#v found=%v err=%v", mismatch, found, getErr)
	}

	timedStop := newRepositoryReviewController(nil)
	timedStop.stopTimeout = time.Millisecond
	timedStop.wg.Add(1)
	timedStop.Stop()
	timedStop.wg.Done()

	monitor := repositoryReviewCoverageLeasedController(t, handler, store)
	monitor.monitorEvery = 2 * time.Millisecond
	monitor.wg.Add(1)
	monitorDone := make(chan struct{})
	go func() {
		monitor.monitor()
		close(monitorDone)
	}()
	time.Sleep(10 * time.Millisecond)
	monitor.cancel()
	<-monitorDone
}

func TestRepositoryReviewBackgroundWorkerLifecycleFencing(t *testing.T) {
	controller := newRepositoryReviewController(nil)
	controller.stopTimeout = time.Second
	var workerMu sync.Mutex
	if !controller.admitBackgroundWorker(&workerMu) {
		t.Fatal("background worker was not admitted")
	}
	workerDone := make(chan struct{})
	go func() {
		defer controller.wg.Done()
		defer workerMu.Unlock()
		<-controller.ctx.Done()
		close(workerDone)
	}()
	controller.Stop()
	select {
	case <-workerDone:
	case <-time.After(time.Second):
		t.Fatal("Stop returned without settling the admitted worker")
	}
	if controller.admitBackgroundWorker(&workerMu) {
		t.Fatal("worker was admitted after Stop")
	}
	if !workerMu.TryLock() {
		t.Fatal("settled worker mutex remained locked")
	}
	workerMu.Unlock()

	delayed := newRepositoryReviewController(nil)
	delayed.lifecycleMu.Lock()
	started := make(chan struct{})
	admitted := make(chan bool, 1)
	go func() {
		close(started)
		admitted <- delayed.registerBackgroundWorker()
	}()
	<-started
	delayed.stopped = true
	delayed.cancel()
	delayed.lifecycleMu.Unlock()
	if <-admitted {
		t.Fatal("worker crossed a closed lifecycle admission fence")
	}
}

func TestRepositoryReviewCoverageAccountSelectionAndPricingBoundaries(t *testing.T) {
	if repositoryReviewAccountAvailable(nil, "", codexAccountLimitAccount{}, false) ||
		repositoryReviewAccountAvailable(nil, "direct", codexAccountLimitAccount{}, false) {
		t.Fatal("invalid account option was available")
	}
	if repositoryReviewAliasAvailableForRuntime(
		config.DefaultConfig(),
		config.ModelAliasConfig{Name: "review"},
		" ",
	) {
		t.Fatal("blank additional account made alias available")
	}
	if refs := repositoryReviewAccountRefsForSelection(nil, "account"); refs != nil {
		t.Fatalf("nil configuration refs=%#v", refs)
	}
	if refs := repositoryReviewAccountRefsForSelection(config.DefaultConfig(), ""); refs != nil {
		t.Fatalf("empty account refs=%#v", refs)
	}
	if refs := repositoryReviewRuntimeAccountRefs(config.DefaultConfig()); len(refs) != 0 {
		t.Fatalf("default configuration runtime refs=%#v", refs)
	}
	if repositoryReviewAliasAvailableForAccount(nil, "review", "account") ||
		repositoryReviewAliasAvailableForAccount(config.DefaultConfig(), "", "account") ||
		repositoryReviewAliasAvailableForAccount(config.DefaultConfig(), "review", "") {
		t.Fatal("invalid account/model selection was available")
	}
	if repositoryReviewAliasUsesAgenticCLIOnAccount(nil, "review", "account") {
		t.Fatal("nil configuration reported an agentic CLI")
	}

	embedded := config.DefaultConfig()
	embedded.Agents.Defaults.AccountRef = "embedded-router"
	embedded.ModelList = []*config.ModelConfig{
		{
			ModelName: "embedded-router", Enabled: true,
			Router: &config.AccountRouterConfig{Entry: "entry", Blocks: []config.AccountRouterBlock{{
				ID: "entry", Type: config.AccountRouterBlockTypeAccount, Account: "concrete",
			}}},
		},
		{ModelName: "concrete", Provider: "openai", Model: "openai/review", Enabled: true},
	}
	embedded.ModelAliases = []config.ModelAliasConfig{{Name: "review", Model: "openai/review"}}
	if refs := repositoryReviewAccountRefsForSelection(embedded, "embedded-router"); !reflect.DeepEqual(
		refs,
		[]string{"concrete"},
	) {
		t.Fatalf("embedded selection refs=%#v", refs)
	}

	directCLI := config.DefaultConfig()
	directCLI.Agents.Defaults.AccountRef = "api"
	directCLI.ModelAliases = []config.ModelAliasConfig{{Name: "unsafe", Model: "codex-cli/codex"}}
	directCLI.ModelList = []*config.ModelConfig{{
		ModelName: "api", Provider: "openai", Model: "openai/review", Enabled: true,
	}}
	if !repositoryReviewAliasUsesAgenticCLIOnAccount(directCLI, "unsafe", "api") {
		t.Fatal("agentic provider encoded in the alias was not rejected")
	}
	providerCLI := config.DefaultConfig()
	providerCLI.Agents.Defaults.AccountRef = "cli-account"
	providerCLI.ModelAliases = []config.ModelAliasConfig{{Name: "unsafe", Model: "review"}}
	providerCLI.ModelList = []*config.ModelConfig{{
		ModelName: "cli-account", Provider: "claude-cli", Model: "review", Enabled: true,
	}}
	if !repositoryReviewAliasUsesAgenticCLIOnAccount(providerCLI, "unsafe", "cli-account") {
		t.Fatal("agentic provider encoded in the account was not rejected")
	}

	if price, ok := repositoryReviewAliasPriceForAccount(nil, "review", "api", nil); price != nil || ok {
		t.Fatalf("nil account price=(%#v,%v)", price, ok)
	}
	if price, ok := repositoryReviewAliasPriceForAccount(
		config.DefaultConfig(), "review", "api", map[string]bool{"review": true},
	); price != nil || ok {
		t.Fatalf("recursive account price=(%#v,%v)", price, ok)
	}
	if price, ok := repositoryReviewAliasPriceForAccount(
		config.DefaultConfig(), "review", "", make(map[string]bool),
	); price != nil || ok {
		t.Fatalf("empty-route account price=(%#v,%v)", price, ok)
	}

	missing := config.DefaultConfig()
	missing.Agents.Defaults.AccountRef = "missing"
	missing.ModelAliases = []config.ModelAliasConfig{{Name: "review", Model: "openai/review"}}
	if price, ok := repositoryReviewAliasPriceForAccount(
		missing, "review", "missing", make(map[string]bool),
	); price != nil || ok {
		t.Fatalf("missing concrete account price=(%#v,%v)", price, ok)
	}

	subscription := config.DefaultConfig()
	subscription.Agents.Defaults.AccountRef = "subscription"
	subscription.ModelAliases = []config.ModelAliasConfig{
		{Name: "review", Model: "openai/subscription"},
		{Name: "equivalent", Model: "openai/equivalent"},
	}
	subscription.ModelList = []*config.ModelConfig{{
		ModelName: "subscription", Provider: "openai", Model: "openai/subscription", Enabled: true,
		Subscription: true, SubscriptionEquivalentModel: "equivalent",
	}}
	if price, ok := repositoryReviewAliasPriceForAccount(
		subscription, "review", "subscription", make(map[string]bool),
	); price != nil || ok {
		t.Fatalf("unpriced subscription fallback=(%#v,%v)", price, ok)
	}
	credentialPricing := config.DefaultConfig()
	credentialPricing.Agents.Defaults.AccountRef = "subscription-metadata"
	credentialPricing.ModelAliases = []config.ModelAliasConfig{
		{Name: "review", Model: "openai/subscription"},
		{Name: "equivalent", Model: "openai/equivalent"},
	}
	credentialPricing.ModelList = []*config.ModelConfig{
		{
			ModelName: "subscription-metadata", Provider: "openai", Model: "openai/subscription", Enabled: true,
			Subscription: true, SubscriptionEquivalentModel: "equivalent",
		},
		{
			ModelName: "equivalent-metadata", Provider: "openai", Model: "openai/equivalent", Enabled: true,
			InputPricePerMTok: 2, OutputPricePerMTok: 8,
		},
	}
	credentialRef := config.AccountRouterCredentialAccountPrefix + "openai:work"
	if price, ok := repositoryReviewAliasPriceForAccount(
		credentialPricing, "review", credentialRef, make(map[string]bool),
	); !ok || price.InputPricePerMTok != 2 || price.OutputPricePerMTok != 8 {
		t.Fatalf("credential subscription price=(%#v,%v)", price, ok)
	}
	credentialPricing.ModelList = nil
	if price, ok := repositoryReviewAliasPriceForAccount(
		credentialPricing, "review", credentialRef, make(map[string]bool),
	); price != nil || ok {
		t.Fatalf("missing credential fallback price=(%#v,%v)", price, ok)
	}

	selectable := config.DefaultConfig()
	selectable.Agents.Defaults.AccountRef = " account "
	selectable.ModelList = []*config.ModelConfig{
		nil,
		{ModelName: "account", Provider: "openai", Enabled: true},
		{ModelName: "", Provider: "openai", Enabled: true},
	}
	selectable.AccountRouters = []config.AccountRouterConfig{
		{Name: "account", Enabled: true},
		{Name: "", Enabled: true},
	}
	if refs := repositoryReviewSelectableAccountRefs(selectable); !reflect.DeepEqual(refs, []string{"account"}) {
		t.Fatalf("deduplicated selectable refs=%#v", refs)
	}

	limits := codexAccountLimitsResponse{Accounts: []codexAccountLimitAccount{
		{ID: "work", Provider: "openai", Email: "first@example.test"},
		{ID: "work", Provider: "openai", Email: "second@example.test"},
	}}
	accounts := repositoryReviewAccountOptions(selectable, limits)
	seen := make(map[string]bool)
	for _, account := range accounts {
		if seen[account.ID] {
			t.Fatalf("duplicate account option %#v", accounts)
		}
		seen[account.ID] = true
	}
	if len(accounts) != 2 || !accounts[0].Default {
		t.Fatalf("account option ordering=%#v", accounts)
	}
}

func TestRepositoryReviewCoverageUnchangedStartedAutomationKeepsPriceSnapshot(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	input := testRepositoryReviewAutomation()
	input.StartedAt = time.Now().UTC().Add(-time.Minute)
	input.ModelPrices = map[string]repoaudit.RepositoryReviewModelPrice{
		"cheap": {InputPricePer1M: 7, OutputPricePer1M: 9},
	}
	created, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	body := automationConfigBody(created)
	body["expected_version"] = created.Version
	response := repositoryReviewCoverageMutation(
		t, mux, http.MethodPatch, "/api/repository-reviews/automations/"+created.ID, body,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("unchanged update=%d %s", response.Code, response.Body.String())
	}
	var result struct {
		Automation repoaudit.RepositoryReviewAutomation `json:"automation"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Automation.ModelPrices["cheap"].InputPricePer1M != 7 {
		t.Fatalf("price snapshot=%#v", result.Automation.ModelPrices)
	}
}

func TestRepositoryReviewCoverageTaskAdmissionAndLimitBoundaries(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	controller := newRepositoryReviewController(handler)

	if taskErr := controller.observeRepositoryReviewTask(
		"missing", "run", workflows.ManagedChildActivity{Phase: workflows.ManagedChildActivityPhase("other")},
	); taskErr != nil {
		t.Fatalf("non-admission activity error=%v", taskErr)
	}
	if taskErr := controller.observeRepositoryReviewTask(
		"missing", "run", workflows.ManagedChildActivity{Phase: workflows.ManagedChildStarted},
	); !errors.Is(taskErr, errRepositoryReviewSafeStop) {
		t.Fatalf("missing active admission error=%v", taskErr)
	}

	paused := repositoryReviewCoverageRunningAutomation(t, store, "run-paused-admission", false)
	controller.active[paused.ID] = &repositoryReviewActiveRun{
		runID: paused.ActiveRunID, store: store,
		pauseReason: repoaudit.RepositoryReviewPauseManual, pauseDetail: "already paused",
	}
	if taskErr := controller.observeRepositoryReviewTask(
		paused.ID, paused.ActiveRunID, workflows.ManagedChildActivity{Phase: workflows.ManagedChildStarted},
	); !errors.Is(taskErr, errRepositoryReviewSafeStop) || !strings.Contains(taskErr.Error(), "already paused") {
		t.Fatalf("paused admission error=%v", taskErr)
	}

	emptyGuard := repositoryReviewCoverageRunningAutomation(t, store, "run-empty-guard", false)
	controller.active[emptyGuard.ID] = &repositoryReviewActiveRun{
		runID: emptyGuard.ActiveRunID, store: store, reservations: make(map[int]repositoryReviewTaskReservation),
	}
	if taskErr := controller.observeRepositoryReviewTask(
		emptyGuard.ID, emptyGuard.ActiveRunID,
		workflows.ManagedChildActivity{Phase: workflows.ManagedChildStarted, Index: 1},
	); taskErr != nil {
		t.Fatalf("empty guard admission error=%v", taskErr)
	}

	missingState := &repositoryReviewActiveRun{
		runID: "run-missing-state", store: store, guardMu: &sync.Mutex{},
		reservations: make(map[int]repositoryReviewTaskReservation),
	}
	controller.active["rra_missing_state"] = missingState
	if taskErr := controller.observeRepositoryReviewTask(
		"rra_missing_state", missingState.runID,
		workflows.ManagedChildActivity{Phase: workflows.ManagedChildStarted},
	); !errors.Is(taskErr, errRepositoryReviewSafeStop) || !strings.Contains(taskErr.Error(), "unavailable") {
		t.Fatalf("missing durable state admission error=%v", taskErr)
	}

	addRepositoryReviewGuardReservation(nil, repositoryReviewTaskReservation{TotalTokens: 10})
	if repositoryReviewAutomationPriceKnown(repoaudit.RepositoryReviewAutomation{}) {
		t.Fatal("automation without models had known pricing")
	}

	if snapshots, known, limitErr := controller.repositoryReviewGuardAccountLimits(
		t.Context(), nil, repoaudit.RepositoryReviewAutomation{},
	); limitErr == nil || known || snapshots != nil {
		t.Fatalf("unavailable account snapshots=%#v known=%v err=%v", snapshots, known, limitErr)
	}

	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	used := 40
	controller.probe = func(context.Context) (codexAccountLimitsResponse, error) {
		return codexAccountLimitsResponse{Accounts: []codexAccountLimitAccount{{
			ID: "api", LimitsError: "temporarily unavailable",
		}}}, nil
	}
	snapshots, known, err := controller.repositoryReviewGuardAccountLimits(
		t.Context(), cfg, testRepositoryReviewAutomation(),
	)
	if err != nil || known || len(snapshots) != 1 ||
		!strings.Contains(snapshots[0].Detail, "temporarily") {
		t.Fatalf("empty telemetry snapshots=%#v known=%v err=%v", snapshots, known, err)
	}

	reset := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	controller.probe = func(context.Context) (codexAccountLimitsResponse, error) {
		return codexAccountLimitsResponse{Accounts: []codexAccountLimitAccount{{
			ID: "api", Entries: []codexAccountLimitEntry{
				{Status: "limit_reached", Window: "weekly", RefreshesAt: reset.Format(time.RFC3339)},
				{Status: "available", Window: "daily", UsedPercent: &used},
				{Status: "available", Window: "monthly"},
			},
		}}}, nil
	}
	snapshots, known, err = controller.repositoryReviewGuardAccountLimits(
		t.Context(), cfg, testRepositoryReviewAutomation(),
	)
	if err != nil || known || len(snapshots) != 3 || snapshots[0].RemainingPercent == nil ||
		*snapshots[0].RemainingPercent != 0 || !snapshots[0].ResetsAt.Equal(reset) ||
		snapshots[1].RemainingPercent == nil || *snapshots[1].RemainingPercent != 60 {
		t.Fatalf("mixed telemetry snapshots=%#v known=%v err=%v", snapshots, known, err)
	}

	controller.probe = func(context.Context) (codexAccountLimitsResponse, error) {
		return codexAccountLimitsResponse{Error: "partial response"}, nil
	}
	if _, known, err = controller.repositoryReviewGuardAccountLimits(
		t.Context(), cfg, testRepositoryReviewAutomation(),
	); err == nil || known || !strings.Contains(err.Error(), "partial response") {
		t.Fatalf("response error known=%v err=%v", known, err)
	}
	controller.probe = func(context.Context) (codexAccountLimitsResponse, error) {
		return codexAccountLimitsResponse{}, errors.New("probe failed")
	}
	if _, known, err = controller.repositoryReviewGuardAccountLimits(
		t.Context(), cfg, testRepositoryReviewAutomation(),
	); err == nil || known || !strings.Contains(err.Error(), "probe failed") {
		t.Fatalf("probe error known=%v err=%v", known, err)
	}

	if ids := repositoryReviewTelemetryIDsForAccountRef(
		nil,
		config.AccountRouterCredentialAccountPrefix+"OpenAI:Work",
	); !reflect.DeepEqual(ids, []string{"openai:work"}) {
		t.Fatalf("credential telemetry ids=%#v", ids)
	}

	dispatchInput := testRepositoryReviewAutomation()
	dispatchInput.BudgetPolicy.GuardExpression = "account.limits.known"
	dispatchInput.Status = repoaudit.RepositoryReviewAutomationRunning
	dispatchInput.ActiveRunID = "run-dispatch-pause"
	dispatchInput.RunIDs = []string{dispatchInput.ActiveRunID}
	dispatch, err := store.CreateAutomation(t.Context(), dispatchInput)
	if err != nil {
		t.Fatal(err)
	}
	probeStarted := make(chan struct{})
	releaseProbe := make(chan struct{})
	var releaseProbeOnce sync.Once
	releaseProbeCall := func() { releaseProbeOnce.Do(func() { close(releaseProbe) }) }
	t.Cleanup(releaseProbeCall)
	controller.probe = func(context.Context) (codexAccountLimitsResponse, error) {
		close(probeStarted)
		<-releaseProbe
		return codexAccountLimitsResponse{Accounts: []codexAccountLimitAccount{{
			ID: "api", Entries: []codexAccountLimitEntry{{
				Status: "available", Window: "weekly", UsedPercent: &used,
			}},
		}}}, nil
	}
	controller.active[dispatch.ID] = &repositoryReviewActiveRun{
		runID: dispatch.ActiveRunID, store: store, config: cfg, guardMu: &sync.Mutex{},
		reservations: make(map[int]repositoryReviewTaskReservation),
	}
	dispatchResult := make(chan error, 1)
	go func() {
		dispatchResult <- controller.observeRepositoryReviewTask(
			dispatch.ID, dispatch.ActiveRunID,
			workflows.ManagedChildActivity{Phase: workflows.ManagedChildStarted, Index: 7},
		)
	}()
	select {
	case <-probeStarted:
	case err := <-dispatchResult:
		t.Fatalf("admission returned before probing limits: %v", err)
	case <-time.After(time.Second):
		t.Fatal("admission did not probe account limits")
	}
	controller.mu.Lock()
	controller.active[dispatch.ID].pauseReason = repoaudit.RepositoryReviewPauseManual
	controller.active[dispatch.ID].pauseDetail = "paused before dispatch"
	controller.mu.Unlock()
	releaseProbeCall()
	select {
	case err := <-dispatchResult:
		if !errors.Is(err, errRepositoryReviewSafeStop) ||
			!strings.Contains(err.Error(), "paused before dispatch") {
			t.Fatalf("pre-dispatch pause error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("admission did not return after the account-limit probe was released")
	}

	canceled := newRepositoryReviewController(handler)
	canceled.leasedStore = store
	canceled.leasedConfig = cfg
	canceled.cancel()
	canceled.reconcile()
}

func TestRepositoryReviewCoverageStartAdmissionLateFailures(t *testing.T) {
	t.Run("invalid selected account", func(t *testing.T) {
		handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		store, err := handler.repositoryReviewStore()
		if err != nil {
			t.Fatal(err)
		}
		input := testRepositoryReviewAutomation()
		input.AccountRef = "missing-account"
		created, err := store.CreateAutomation(t.Context(), input)
		if err != nil {
			t.Fatal(err)
		}
		controller := repositoryReviewCoverageLeasedController(t, handler, store)
		if _, err := controller.startAutomation(
			t.Context(), created.ID, created.Version, false, "start",
		); !errors.Is(err, repoaudit.ErrInvalidAutomation) || !strings.Contains(err.Error(), "account_ref") {
			t.Fatalf("invalid account start error=%v", err)
		}
	})

	t.Run("unavailable reviewer alias", func(t *testing.T) {
		handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		store, err := handler.repositoryReviewStore()
		if err != nil {
			t.Fatal(err)
		}
		input := testRepositoryReviewAutomation()
		input.ReviewerModels = []string{"missing-alias"}
		created, err := store.CreateAutomation(t.Context(), input)
		if err != nil {
			t.Fatal(err)
		}
		controller := repositoryReviewCoverageLeasedController(t, handler, store)
		if _, err := controller.startAutomation(
			t.Context(), created.ID, created.Version, false, "start",
		); !errors.Is(err, repoaudit.ErrInvalidAutomation) || !strings.Contains(err.Error(), "unavailable") {
			t.Fatalf("unavailable alias start error=%v", err)
		}
	})

	t.Run("missing central spend price", func(t *testing.T) {
		handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		store, err := handler.repositoryReviewStore()
		if err != nil {
			t.Fatal(err)
		}
		input := testRepositoryReviewAutomation()
		input.BudgetPolicy.GuardExpression = "spend.total.usd < 10"
		created, err := store.CreateAutomation(t.Context(), input)
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := config.LoadConfig(handler.configPath)
		if err != nil {
			t.Fatal(err)
		}
		cfg.ModelList[0].InputPricePerMTok = 0
		cfg.ModelList[0].OutputPricePerMTok = 0
		if err := config.SaveConfig(handler.configPath, cfg); err != nil {
			t.Fatal(err)
		}
		controller := repositoryReviewCoverageLeasedController(t, handler, store)
		if _, err := controller.startAutomation(
			t.Context(), created.ID, created.Version, false, "start",
		); !errors.Is(err, repoaudit.ErrInvalidAutomation) || !strings.Contains(err.Error(), "spend.total") {
			t.Fatalf("unpriced start error=%v", err)
		}
	})

	stagedFailure := func(t *testing.T, mode string, reset bool, want error) {
		t.Helper()
		handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		store, err := handler.repositoryReviewStore()
		if err != nil {
			t.Fatal(err)
		}
		created, err := store.CreateAutomation(t.Context(), testRepositoryReviewAutomation())
		if err != nil {
			t.Fatal(err)
		}
		controller := repositoryReviewCoverageLeasedController(t, handler, store)
		calls := 0
		controller.update = func(
			ctx context.Context,
			actualStore repoaudit.Store,
			id string,
			version int64,
			mutate func(*repoaudit.RepositoryReviewAutomation) error,
		) (repoaudit.RepositoryReviewAutomation, error) {
			calls++
			updated, updateErr := actualStore.UpdateAutomation(ctx, id, version, mutate)
			if calls != 1 || updateErr != nil {
				return updated, updateErr
			}
			switch mode {
			case "reset failure", "transition failure":
				controller.update = func(
					context.Context, repoaudit.Store, string, int64,
					func(*repoaudit.RepositoryReviewAutomation) error,
				) (repoaudit.RepositoryReviewAutomation, error) {
					return repoaudit.RepositoryReviewAutomation{}, want
				}
			case "cancel after pricing":
				controller.cancel()
			case "active race":
				controller.mu.Lock()
				controller.active[id] = &repositoryReviewActiveRun{runID: "racing-run", store: actualStore}
				controller.mu.Unlock()
			}
			return updated, nil
		}
		_, startErr := controller.startAutomation(
			t.Context(), created.ID, created.Version, reset, "start",
		)
		switch mode {
		case "reset failure", "transition failure":
			if !errors.Is(startErr, want) {
				t.Fatalf("%s error=%v", mode, startErr)
			}
		case "cancel after pricing":
			if !errors.Is(startErr, context.Canceled) {
				t.Fatalf("canceled error=%v", startErr)
			}
		case "active race":
			if !errors.Is(startErr, errRepositoryReviewAutomationBusy) {
				t.Fatalf("active race error=%v", startErr)
			}
		}
	}
	t.Run("reset persistence after pricing", func(t *testing.T) {
		stagedFailure(t, "reset failure", true, errors.New("reset after price failed"))
	})
	t.Run("transition persistence after pricing", func(t *testing.T) {
		stagedFailure(t, "transition failure", false, errors.New("transition after price failed"))
	})
	t.Run("canceled after pricing", func(t *testing.T) {
		stagedFailure(t, "cancel after pricing", false, nil)
	})
	t.Run("second active check", func(t *testing.T) {
		stagedFailure(t, "active race", false, nil)
	})

	if model := repositoryReviewPlannerModel(repoaudit.RepositoryReviewAutomation{}); model != "" {
		t.Fatalf("empty planner model=%q", model)
	}
}
