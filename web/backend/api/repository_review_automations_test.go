package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/collectionquery"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/repoaudit"
	"github.com/sipeed/picoclaw/pkg/workflows"
)

type repositoryReviewRecoveryProfileRunner struct {
	profile workflows.RepositoryReviewModelProfile
}

func (runner *repositoryReviewRecoveryProfileRunner) RunAgent(
	context.Context,
	workflows.AgentRequest,
) (map[string]any, error) {
	return nil, errors.New("unexpected agent call")
}

func (runner *repositoryReviewRecoveryProfileRunner) ResolveRepositoryReviewProfile(
	context.Context,
	string,
	string,
	[]string,
) (workflows.RepositoryReviewModelProfile, error) {
	return runner.profile, nil
}

func TestRepositoryReviewAutomationRoutesCreateUpdateListAndDelete(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	profile := createRepositoryReviewProfileForTest(t, mux, "Core pre-review", "cheap")

	create := repositoryReviewAutomationMutation(t, mux, http.MethodPost,
		"/api/repository-reviews/automations", map[string]any{
			"repository": "https://github.com/acme/core.git",
			"branch":     "main",
			"profile_id": profile.ID,
		})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		Automation repoaudit.RepositoryReviewAutomation `json:"automation"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Automation.ID == "" || created.Automation.Status != repoaudit.RepositoryReviewAutomationIdle ||
		created.Automation.ProfileID != profile.ID ||
		created.Automation.ProfileVersion != profile.Version ||
		created.Automation.Ref != "main" || created.Automation.Target != "all" ||
		len(created.Automation.ReviewerModels) != 1 ||
		created.Automation.ReviewerModels[0] != profile.ReviewerModel ||
		created.Automation.CompareModels {
		t.Fatalf("created automation=%#v", created.Automation)
	}
	statePath := filepath.Join(workspace, "repository_reviews", "automation_"+created.Automation.ID+".json")
	if info, err := os.Stat(statePath); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("automation file info=%v err=%v", info, err)
	}

	list := httptest.NewRecorder()
	mux.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/repository-reviews/automations", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), created.Automation.ID) {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}

	updateBody := map[string]any{
		"repository":       created.Automation.Repository,
		"branch":           "release/v2",
		"profile_id":       profile.ID,
		"expected_version": created.Automation.Version,
	}
	update := repositoryReviewAutomationMutation(t, mux, http.MethodPatch,
		"/api/repository-reviews/automations/"+created.Automation.ID, updateBody)
	if update.Code != http.StatusOK || !strings.Contains(update.Body.String(), "release/v2") {
		t.Fatalf("update status=%d body=%s", update.Code, update.Body.String())
	}
	var changed struct {
		Automation repoaudit.RepositoryReviewAutomation `json:"automation"`
	}
	if err := json.Unmarshal(update.Body.Bytes(), &changed); err != nil {
		t.Fatal(err)
	}

	stale := repositoryReviewAutomationMutation(t, mux, http.MethodPatch,
		"/api/repository-reviews/automations/"+created.Automation.ID, updateBody)
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale update status=%d body=%s", stale.Code, stale.Body.String())
	}
	deleted := repositoryReviewAutomationMutation(t, mux, http.MethodDelete,
		"/api/repository-reviews/automations/"+created.Automation.ID,
		map[string]any{
			"expected_version":            changed.Automation.Version,
			"expected_repository_version": 0,
			"expected_ledger_fence":       repositoryReviewPurgeFenceForTest(t, workspace, changed.Automation),
			"confirm_repository":          changed.Automation.Repository,
		})
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
}

func TestRepositoryReviewAutomationHistoryPurgeContract(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := completeRepositoryReviewAPIMappingJobs(
		t, workspace, seedRepositoryReviewAPIState(t, workspace),
	)
	store := repoaudit.NewStore(workspace)
	automationInput := testRepositoryReviewAutomation()
	automationInput.Repository = state.Repository
	automation, err := store.CreateAutomation(t.Context(), automationInput)
	if err != nil {
		t.Fatal(err)
	}

	detail := httptest.NewRecorder()
	mux.ServeHTTP(detail, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations/"+automation.ID,
		nil,
	))
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status=%d body=%s", detail.Code, detail.Body.String())
	}
	if !strings.Contains(detail.Body.String(), `"can_purge_history":true`) ||
		!strings.Contains(detail.Body.String(), `"can_remove_repository":true`) ||
		!strings.Contains(detail.Body.String(), `"purge_blockers":[]`) {
		t.Fatalf("detail omitted authoritative purge capabilities: %s", detail.Body.String())
	}
	var projected struct {
		Repository   repoaudit.RepositorySummary `json:"repository"`
		Capabilities struct {
			CanPurgeHistory     bool                                     `json:"can_purge_history"`
			CanRemoveRepository bool                                     `json:"can_remove_repository"`
			PurgeBlockers       []repoaudit.RepositoryReviewPurgeBlocker `json:"purge_blockers"`
			PurgeSummary        repoaudit.RepositoryReviewPurgeSummary   `json:"purge_summary"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(detail.Body.Bytes(), &projected); err != nil {
		t.Fatal(err)
	}
	if projected.Repository.Version != state.Version ||
		!projected.Capabilities.CanPurgeHistory || !projected.Capabilities.CanRemoveRepository ||
		len(projected.Capabilities.PurgeBlockers) != 0 ||
		projected.Capabilities.PurgeSummary.RepositoryVersion != state.Version {
		t.Fatalf("purge detail = %#v", projected)
	}

	stale := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/purge-history",
		map[string]any{
			"expected_version":            automation.Version,
			"expected_repository_version": state.Version + 1,
			"expected_ledger_fence":       projected.Capabilities.PurgeSummary.LedgerFence,
			"confirm_repository":          automation.Repository,
		},
	)
	if stale.Code != http.StatusConflict || !strings.Contains(stale.Body.String(), "stale_repository_review_automation") {
		t.Fatalf("stale purge status=%d body=%s", stale.Code, stale.Body.String())
	}

	purged := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/purge-history",
		map[string]any{
			"expected_version":            automation.Version,
			"expected_repository_version": state.Version,
			"expected_ledger_fence":       projected.Capabilities.PurgeSummary.LedgerFence,
			"confirm_repository":          automation.Repository,
		},
	)
	if purged.Code != http.StatusOK || !strings.Contains(purged.Body.String(), `"outcome":"history_purged"`) {
		t.Fatalf("purge status=%d body=%s", purged.Code, purged.Body.String())
	}
	if _, found, getErr := store.Get(state.Repository); getErr != nil || found {
		t.Fatalf("purged ledger found=%v err=%v", found, getErr)
	}
	reset, found, getErr := store.GetAutomation(t.Context(), automation.ID)
	if getErr != nil || !found || reset.Status != repoaudit.RepositoryReviewAutomationIdle ||
		len(reset.RunIDs) != 0 || reset.Progress.Findings != 0 || reset.Version != automation.Version+1 {
		t.Fatalf("reset automation=%#v found=%v err=%v", reset, found, getErr)
	}
	after := httptest.NewRecorder()
	mux.ServeHTTP(after, httptest.NewRequest(
		http.MethodGet, "/api/repository-reviews/automations/"+automation.ID, nil,
	))
	if after.Code != http.StatusOK ||
		!strings.Contains(after.Body.String(), `"can_purge_history":false`) ||
		!strings.Contains(after.Body.String(), `"can_remove_repository":true`) ||
		!strings.Contains(after.Body.String(), `"purge_blockers":[]`) ||
		!strings.Contains(after.Body.String(), `"repository_version":0`) {
		t.Fatalf("post-purge capabilities status=%d body=%s", after.Code, after.Body.String())
	}
}

func repositoryReviewPurgeFenceForTest(
	t *testing.T,
	workspace string,
	automation repoaudit.RepositoryReviewAutomation,
) string {
	t.Helper()
	eligibility, err := repoaudit.NewStore(workspace).RepositoryReviewPurgeEligibilityForAutomation(automation)
	if err != nil || eligibility.Summary.LedgerFence == "" {
		t.Fatalf("purge eligibility=%#v err=%v", eligibility, err)
	}
	return eligibility.Summary.LedgerFence
}

func TestRepositoryReviewEffectiveWorkflowTimeout(t *testing.T) {
	for _, test := range []struct {
		name       string
		configured time.Duration
		want       time.Duration
	}{
		{name: "unset", want: 65 * time.Minute},
		{name: "default", configured: 5 * time.Minute, want: 65 * time.Minute},
		{name: "longer configured", configured: 6 * time.Hour, want: 6 * time.Hour},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := repositoryReviewEffectiveWorkflowTimeout(test.configured); got != test.want {
				t.Fatalf("effective timeout = %s, want %s", got, test.want)
			}
		})
	}
	if got := repositoryReviewEffectiveWorkflowTimeoutForAssignment(5*time.Minute, 90*60); got != 95*time.Minute {
		t.Fatalf("custom assignment timeout = %s, want 95m", got)
	}
	if got := repositoryReviewEffectiveWorkflowTimeoutForAssignment(2*time.Hour, 90*60); got != 2*time.Hour {
		t.Fatalf("longer workflow timeout = %s, want 2h", got)
	}
}

func TestRepositoryReviewCampaignUsesCanonicalRemoteLedgerIdentity(t *testing.T) {
	store := repoaudit.NewStore(t.TempDir())
	automation := testRepositoryReviewAutomation()
	automation.ID = "rra_remote_ledger"
	automation.Repository = "https://github.com/Acme/Core.git"
	created, err := store.CreateAutomation(t.Context(), automation)
	if err != nil {
		t.Fatal(err)
	}
	commit := strings.Repeat("a", 40)
	prepared, err := (&repositoryReviewController{}).ensureRepositoryReviewCampaign(
		t.Context(), store, &config.Config{}, created, commit, "start",
	)
	if err != nil || prepared.CampaignID == "" {
		t.Fatalf("prepared remote campaign = %#v, %v", prepared, err)
	}
	canonical, found, err := store.Get("acme/core")
	if err != nil || !found || canonical.CurrentCampaign == nil ||
		canonical.CurrentCampaign.ID != prepared.CampaignID {
		t.Fatalf("canonical ledger = %#v found=%v err=%v", canonical.CurrentCampaign, found, err)
	}
	if _, found, err := store.Get(automation.Repository); err != nil || found {
		t.Fatalf("raw remote ledger unexpectedly exists: found=%v err=%v", found, err)
	}
}

func TestRepositoryReviewAutomationCollectionQueryAndPaging(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	firstInput := testRepositoryReviewAutomation()
	firstInput.ID = "rra_collection_first"
	firstInput.Repository = "https://github.com/acme/first.git"
	firstInput.Name = "First review"
	firstInput.Status = repoaudit.RepositoryReviewAutomationPaused
	firstInput.PauseReason = repoaudit.RepositoryReviewPauseManual
	firstInput.PauseDetail = "paused"
	firstInput.Progress.ReviewedFiles = 2
	firstInput.Progress.RemainingFiles = 8
	first, err := store.CreateAutomation(t.Context(), firstInput)
	if err != nil {
		t.Fatal(err)
	}
	secondInput := testRepositoryReviewAutomation()
	secondInput.ID = "rra_collection_second"
	secondInput.Repository = "https://github.com/acme/second.git"
	secondInput.Name = "Second review"
	secondInput.Status = repoaudit.RepositoryReviewAutomationPaused
	secondInput.PauseReason = repoaudit.RepositoryReviewPauseManual
	secondInput.PauseDetail = "paused"
	secondInput.Progress.ReviewedFiles = 8
	secondInput.Progress.RemainingFiles = 2
	second, err := store.CreateAutomation(t.Context(), secondInput)
	if err != nil {
		t.Fatal(err)
	}

	query := url.QueryEscape("status = paused ORDER BY name ASC")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations?query="+query+"&limit=1",
		nil,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("collection status=%d body=%s", response.Code, response.Body.String())
	}
	var page struct {
		Automations []repoaudit.RepositoryReviewAutomation `json:"automations"`
		Total       int                                    `json:"total"`
		NextCursor  string                                 `json:"next_cursor"`
		Canonical   string                                 `json:"canonical_query"`
		QuerySchema json.RawMessage                        `json:"query_schema"`
	}
	if decodeErr := json.Unmarshal(response.Body.Bytes(), &page); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if page.Total != 2 || len(page.Automations) != 1 || page.Automations[0].ID != first.ID ||
		page.NextCursor == "" || page.Canonical != `status = "paused" ORDER BY name ASC` ||
		!json.Valid(page.QuerySchema) {
		t.Fatalf("collection page=%#v first=%s second=%s", page, first.ID, second.ID)
	}

	progressQuery := url.QueryEscape("progress >= 50 ORDER BY progress DESC")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations?query="+progressQuery,
		nil,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("progress collection status=%d body=%s", response.Code, response.Body.String())
	}
	if decodeErr := json.Unmarshal(response.Body.Bytes(), &page); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if page.Total != 1 || len(page.Automations) != 1 || page.Automations[0].ID != second.ID {
		t.Fatalf("file-progress collection page=%#v", page)
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations?query=status%20%3D%20missing",
		nil,
	))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid collection status=%d body=%s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations?query="+query+"&cursor=invalid",
		nil,
	))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid cursor status=%d body=%s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations/rra_absent/commit-options",
		nil,
	))
	if response.Code != http.StatusNotFound {
		t.Fatalf("missing commit options status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewAutomationCollectionFields(t *testing.T) {
	automation := testRepositoryReviewAutomation()
	automation.ID = "rra_collection_fields"
	automation.Name = "Collection fields"
	automation.Status = repoaudit.RepositoryReviewAutomationRunning
	automation.Progress.CompletedBatches = 1
	automation.Progress.TotalBatches = 4
	automation.Progress.ReviewedFiles = 7
	automation.Progress.RemainingFiles = 21
	automation.Progress.RawFindings = 5
	automation.Progress.DeduplicatedFindings = 2
	automation.Progress.Findings = 2
	// A scope plan written by an older launcher is not authoritative without
	// the durable internal selection that freezes it.
	automation.ScopePlan = repoaudit.RepositoryReviewScopePlan{
		Counts: repoaudit.RepositoryReviewScopePlanCounts{SelectedFiles: 100},
	}
	automation.UpdatedAt = time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	for _, field := range []collectionquery.Field{
		"id", "name", "repository", "branch", "status", "progress", "reviewed", "raw_findings",
		"findings", "updated",
	} {
		if _, ok := repositoryReviewAutomationCollectionField(automation, field); !ok {
			t.Fatalf("field %q was not resolved", field)
		}
	}
	if value, ok := repositoryReviewAutomationCollectionField(automation, "progress"); !ok || value.Number != 25 {
		t.Fatalf("progress field=%#v ok=%v", value, ok)
	}
	if value, ok := repositoryReviewAutomationCollectionField(automation, "raw_findings"); !ok ||
		value.Number != 5 {
		t.Fatalf("raw findings field=%#v ok=%v", value, ok)
	}
	automation.Progress = repoaudit.RepositoryReviewProgress{}
	if value, ok := repositoryReviewAutomationCollectionField(automation, "progress"); !ok || value.Number != 0 {
		t.Fatalf("zero progress field=%#v ok=%v", value, ok)
	}
	if _, ok := repositoryReviewAutomationCollectionField(automation, "missing"); ok {
		t.Fatal("unknown collection field resolved")
	}
}

func TestRepositoryReviewGitHubCommitURLBoundaries(t *testing.T) {
	commit := strings.Repeat("a", 40)
	if got := repositoryReviewGitHubCommitURL("owner/repo", "invalid"); got != "" {
		t.Fatalf("invalid commit URL=%q", got)
	}
	if got := repositoryReviewGitHubCommitURL(
		"git@github.com:owner/repo.git", commit,
	); got != "https://github.com/owner/repo/commit/"+commit {
		t.Fatalf("SCP commit URL=%q", got)
	}
	for _, repository := range []string{
		"git@gitlab.com:owner/repo.git",
		"relative-repository",
		"https://github.com/owner",
		"https://github.com//repo",
		"https://github.com/owner/.git",
	} {
		if got := repositoryReviewGitHubCommitURL(repository, commit); got != "" {
			t.Fatalf("repository %q commit URL=%q", repository, got)
		}
	}
}

func TestRepositoryReviewAutomationRoutesRejectInvalidStateTransitionsAndBodies(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, storeErr := handler.repositoryReviewStore()
	if storeErr != nil {
		t.Fatal(storeErr)
	}
	idle, err := store.CreateAutomation(t.Context(), testRepositoryReviewAutomation())
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"pause", "resume", "restart"} {
		response := repositoryReviewAutomationMutation(t, mux, http.MethodPost,
			"/api/repository-reviews/automations/"+idle.ID+"/"+action,
			map[string]any{"expected_version": idle.Version})
		if response.Code != http.StatusConflict {
			t.Fatalf("idle %s status=%d body=%s", action, response.Code, response.Body.String())
		}
	}
	pausedInput := testRepositoryReviewAutomation()
	pausedInput.Status = repoaudit.RepositoryReviewAutomationPaused
	pausedInput.PauseReason = repoaudit.RepositoryReviewPauseManual
	pausedInput.PauseDetail = "paused"
	paused, err := store.CreateAutomation(t.Context(), pausedInput)
	if err != nil {
		t.Fatal(err)
	}
	response := repositoryReviewAutomationMutation(t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+paused.ID+"/start",
		map[string]any{"expected_version": paused.Version})
	if response.Code != http.StatusConflict {
		t.Fatalf("paused start status=%d body=%s", response.Code, response.Body.String())
	}

	for _, body := range []string{"", `{`, `{"expected_version":"wrong"}`, `{} {}`} {
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/repository-reviews/automations/"+idle.ID+"/start",
			strings.NewReader(body),
		)
		setRepositoryReviewMutationHeaders(request)
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("body %q status=%d response=%s", body, recorder.Code, recorder.Body.String())
		}
	}
	invalidScope := automationConfigBody(testRepositoryReviewAutomation())
	invalidScope["scope_policy"] = map[string]any{
		"code_types": []string{"code"}, "include_folders": []string{"../outside"},
	}
	response = repositoryReviewAutomationMutation(
		t, mux, http.MethodPost, "/api/repository-reviews/automations", invalidScope,
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unsafe scope status=%d body=%s", response.Code, response.Body.String())
	}
	forgedPlan := automationConfigBody(testRepositoryReviewAutomation())
	forgedPlan["scope_plan"] = map[string]any{"summary": "client supplied"}
	response = repositoryReviewAutomationMutation(
		t, mux, http.MethodPost, "/api/repository-reviews/automations", forgedPlan,
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("client scope plan status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewAutomationScopeChangeClearsCommitPlan(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation, err := store.CreateAutomation(t.Context(), testRepositoryReviewAutomation())
	if err != nil {
		t.Fatal(err)
	}
	automation, err = store.UpdateAutomation(
		t.Context(), automation.ID, automation.Version,
		func(candidate *repoaudit.RepositoryReviewAutomation) error {
			candidate.ScopePlan = repoaudit.RepositoryReviewScopePlan{
				CommitSHA: strings.Repeat("a", 40), PolicyHash: strings.Repeat("b", 64),
				Hash: strings.Repeat("c", 64), Summary: "production files selected",
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	equivalentBody := automationConfigBody(automation)
	equivalentBody["scope_policy"] = map[string]any{
		"code_types":      []string{" CODE ", "hotpath-code"},
		"include_folders": []string{}, "exclude_folders": []string{},
	}
	equivalentBody["expected_version"] = automation.Version
	equivalentResponse := repositoryReviewAutomationMutation(
		t, mux, http.MethodPatch, "/api/repository-reviews/automations/"+automation.ID,
		equivalentBody,
	)
	if equivalentResponse.Code != http.StatusOK {
		t.Fatalf("equivalent scope update status=%d body=%s", equivalentResponse.Code, equivalentResponse.Body.String())
	}
	var equivalent struct {
		Automation repoaudit.RepositoryReviewAutomation `json:"automation"`
	}
	if err := json.Unmarshal(equivalentResponse.Body.Bytes(), &equivalent); err != nil {
		t.Fatal(err)
	}
	if equivalent.Automation.ScopePlan.Hash == "" {
		t.Fatalf("equivalent normalized scope cleared plan: %#v", equivalent.Automation.ScopePlan)
	}
	automation = equivalent.Automation
	body := automationConfigBody(automation)
	policy := automation.ScopePolicy
	policy.FreeText = "Focus on storage boundaries."
	body["scope_policy"] = policy
	body["expected_version"] = automation.Version
	response := repositoryReviewAutomationMutation(
		t, mux, http.MethodPatch, "/api/repository-reviews/automations/"+automation.ID, body,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("scope update status=%d body=%s", response.Code, response.Body.String())
	}
	var result struct {
		Automation repoaudit.RepositoryReviewAutomation `json:"automation"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Automation.ScopePolicy.FreeText != "Focus on storage boundaries." ||
		result.Automation.ScopePlan.CommitSHA != "" || result.Automation.ScopePlan.Hash != "" ||
		len(result.Automation.ScopePlan.Warnings) != 0 {
		t.Fatalf("scope update = %#v", result.Automation)
	}
}

func TestRepositoryReviewAutomationStartPersistsTokenBudgetPause(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.Status = repoaudit.RepositoryReviewAutomationRunning
	automation.ActiveRunID = "wr_guard"
	automation.RunIDs = []string{"wr_guard"}
	automation.BudgetPolicy.GuardExpression = "spent.tokens.total < 100"
	automation.Usage = repoaudit.RepositoryReviewTokenUsage{
		PromptTokens: 80, CompletionTokens: 20, TotalTokens: 100,
	}
	automation, err = store.CreateAutomation(t.Context(), automation)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	controller.mu.Lock()
	controller.active[automation.ID] = &repositoryReviewActiveRun{
		runID: "wr_guard", store: store, config: cfg,
		reservations: make(map[int]repositoryReviewTaskReservation),
	}
	controller.mu.Unlock()
	guardErr := controller.observeRepositoryReviewTask(
		automation.ID, "wr_guard", workflows.ManagedChildActivity{
			Phase: workflows.ManagedChildStarted, Index: 1, EstimatedPromptTokens: 1,
		},
	)
	if !errors.Is(guardErr, errRepositoryReviewSafeStop) {
		t.Fatalf("guard error=%v", guardErr)
	}
	updated, _, err := store.GetAutomation(t.Context(), automation.ID)
	if err != nil || updated.Status != repoaudit.RepositoryReviewAutomationStopping ||
		updated.RequestedPauseReason != repoaudit.RepositoryReviewPauseGuardExpression {
		t.Fatalf("guarded automation=%#v err=%v", updated, err)
	}
}

func TestRepositoryReviewAutomationUsageTriggersSafeCheckpointStopAndComparison(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.BudgetPolicy.GuardExpression = "spent.tokens.total < 100"
	automation.ModelPrices = map[string]repoaudit.RepositoryReviewModelPrice{
		"cheap": {InputPricePer1M: 1, OutputPricePer1M: 4},
	}
	automation.Status = repoaudit.RepositoryReviewAutomationRunning
	automation.ActiveRunID = "wr_usage"
	automation.RunIDs = []string{"wr_usage"}
	automation.Progress.TotalBatches = 1
	automation, createErr := store.CreateAutomation(t.Context(), automation)
	if createErr != nil {
		t.Fatal(createErr)
	}
	controller.mu.Lock()
	controller.active[automation.ID] = &repositoryReviewActiveRun{runID: "wr_usage", store: store}
	controller.mu.Unlock()
	stopErr := controller.recordUsage(automation.ID, "wr_usage", workflows.AgentUsage{
		Model: "cheap", PromptTokens: 80, CompletionTokens: 25, TotalTokens: 105, CachedTokens: 10,
	}, repositoryReviewAccountingIndex(nil, automation))
	if stopErr != nil {
		t.Fatalf("recordUsage error=%v", stopErr)
	}

	updated, found, err := store.GetAutomation(t.Context(), automation.ID)
	if err != nil || !found {
		t.Fatalf("GetAutomation found=%v err=%v", found, err)
	}
	stats := updated.ModelStats["cheap"]
	if updated.Status != repoaudit.RepositoryReviewAutomationRunning ||
		updated.Usage.TotalTokens != 105 || stats.Requests != 1 || stats.Tokens.CachedTokens != 10 ||
		math.Abs(stats.EstimatedCostUSD-0.00018) > 0.0000001 {
		t.Fatalf("usage automation=%#v stats=%#v", updated, stats)
	}
	controller.mu.Lock()
	active := controller.active[automation.ID]
	controller.mu.Unlock()
	if active == nil || active.pauseReason != "" {
		t.Fatalf("active stop=%#v", active)
	}
	if err := controller.observeRepositoryReviewTask(
		automation.ID, "wr_usage", workflows.ManagedChildActivity{
			Phase: workflows.ManagedChildStarted, Index: 1, EstimatedPromptTokens: 1,
		},
	); !errors.Is(err, errRepositoryReviewSafeStop) {
		t.Fatalf("next task guard error=%v", err)
	}
}

func TestRepositoryReviewTaskAdmissionReservesConcurrentWorkerUsage(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.Status = repoaudit.RepositoryReviewAutomationRunning
	automation.ActiveRunID = "wr_reservations"
	automation.RunIDs = []string{automation.ActiveRunID}
	automation.BudgetPolicy.GuardExpression = "spent.tokens.total < 2500"
	automation, err = store.CreateAutomation(t.Context(), automation)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	controller.mu.Lock()
	controller.active[automation.ID] = &repositoryReviewActiveRun{
		runID: automation.ActiveRunID, store: store, config: cfg,
		reservations: make(map[int]repositoryReviewTaskReservation),
	}
	controller.mu.Unlock()
	activity := workflows.ManagedChildActivity{
		Phase: workflows.ManagedChildStarted, EstimatedPromptTokens: 1_000,
		EstimatedOutputTokens: 500, PriceKnown: true,
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for index := 1; index <= 2; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			candidate := activity
			candidate.Index = index
			results <- controller.observeRepositoryReviewTask(
				automation.ID, automation.ActiveRunID, candidate,
			)
		}(index)
	}
	close(start)
	wait.Wait()
	close(results)
	admitted, denied := 0, 0
	for result := range results {
		if result == nil {
			admitted++
		} else if errors.Is(result, errRepositoryReviewSafeStop) {
			denied++
		} else {
			t.Fatalf("unexpected task admission error=%v", result)
		}
	}
	if admitted != 1 || denied != 1 {
		t.Fatalf("concurrent admissions admitted=%d denied=%d", admitted, denied)
	}
	controller.mu.Lock()
	reservations := len(controller.active[automation.ID].reservations)
	admittedIndex := 0
	for index := range controller.active[automation.ID].reservations {
		admittedIndex = index
	}
	controller.mu.Unlock()
	if reservations != 1 {
		t.Fatalf("in-flight reservations=%d, want one admitted task", reservations)
	}
	if err := controller.admitProviderCall(automation.ID, automation.ActiveRunID); err != nil {
		t.Fatalf("guard pause interrupted an already admitted task: %v", err)
	}
	if err := controller.observeRepositoryReviewTask(
		automation.ID, automation.ActiveRunID,
		workflows.ManagedChildActivity{Phase: workflows.ManagedChildCompleted, Index: admittedIndex, Success: true},
	); err != nil {
		t.Fatal(err)
	}
	controller.mu.Lock()
	reservations = len(controller.active[automation.ID].reservations)
	controller.mu.Unlock()
	if reservations != 0 {
		t.Fatalf("completed task retained %d reservations", reservations)
	}
}

func TestRepositoryReviewUnmappedModelStillConsumesGlobalTokenBudget(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.Status = repoaudit.RepositoryReviewAutomationRunning
	automation.ActiveRunID = "wr_unmapped"
	automation.RunIDs = []string{"wr_unmapped"}
	automation.Progress.TotalBatches = 1
	automation.BudgetPolicy.GuardExpression = "spent.tokens.total < 10"
	automation, err = store.CreateAutomation(t.Context(), automation)
	if err != nil {
		t.Fatal(err)
	}
	controller.mu.Lock()
	controller.active[automation.ID] = &repositoryReviewActiveRun{runID: "wr_unmapped", store: store}
	controller.mu.Unlock()
	stopErr := controller.recordUsage(automation.ID, "wr_unmapped", workflows.AgentUsage{
		Model: "unexpected-fallback", PromptTokens: 7, CompletionTokens: 3, TotalTokens: 10,
	}, map[string]repositoryReviewAccountingModel{})
	if stopErr != nil {
		t.Fatalf("recordUsage error=%v", stopErr)
	}
	updated, _, err := store.GetAutomation(t.Context(), automation.ID)
	if err != nil || updated.Usage.TotalTokens != 10 ||
		updated.Status != repoaudit.RepositoryReviewAutomationRunning {
		t.Fatalf("unmapped usage=%#v err=%v", updated, err)
	}
}

func TestRepositoryReviewGuardReservationUsesConservativeAutomationPrice(t *testing.T) {
	automation := testRepositoryReviewAutomation()
	automation.ModelPrices = map[string]repoaudit.RepositoryReviewModelPrice{
		"cheap": {InputPricePer1M: 10, OutputPricePer1M: 20},
	}
	reservation := repositoryReviewGuardReservation(
		automation,
		workflows.ManagedChildActivity{
			ModelAlias: "cheap", EstimatedPromptTokens: 1_000, EstimatedOutputTokens: 500,
			EstimatedCostUSD: 0.000001, PriceKnown: true,
		},
	)
	if !reservation.CostKnown || math.Abs(reservation.CostUSD-0.02) > 0.0000001 {
		t.Fatalf("reservation=%#v, want conservative snapshot cost", reservation)
	}
}

func TestRepositoryReviewModelOutcomeUsesRequestedReviewerAndAcknowledgedZeroFindingCoverage(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.Status = repoaudit.RepositoryReviewAutomationRunning
	automation.ActiveRunID = "wr_outcome"
	automation.RunIDs = []string{"wr_outcome"}
	automation.Progress.TotalBatches = 1
	automation, err = store.CreateAutomation(t.Context(), automation)
	if err != nil {
		t.Fatal(err)
	}
	controller.mu.Lock()
	controller.active[automation.ID] = &repositoryReviewActiveRun{runID: "wr_outcome", store: store}
	controller.mu.Unlock()
	run := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"find_bugs/review": {
			ID: "review", Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{
				"managed_children": []map[string]any{{
					"admitted": true, "valid": true,
					"model": map[string]any{"requested": "cheap", "selected": "fallback-model"},
					"scope": []any{map[string]any{"path": "a.go"}},
					"structured": map[string]any{
						"reviewedFiles": []any{"a.go"}, "findings": []any{},
					},
				}},
			},
		},
	}}
	controller.recordManagedChildOutcomes(
		automation.ID, "wr_outcome", run, repositoryReviewAccountingIndex(nil, automation),
	)
	controller.recordManagedChildOutcomes(
		automation.ID, "wr_outcome", run, repositoryReviewAccountingIndex(nil, automation),
	)
	updated, _, err := store.GetAutomation(t.Context(), automation.ID)
	if err != nil {
		t.Fatal(err)
	}
	stats := updated.ModelStats["cheap"]
	if stats.ReviewedFiles != 1 || stats.Findings != 0 || stats.Failures != 0 {
		t.Fatalf("requested reviewer stats=%#v", stats)
	}
	if updated.ModelCoverageSketches["cheap"] == "" {
		t.Fatal("durable model coverage sketch was not stored")
	}
	updated.ScopeSelection = &repoaudit.RepositoryReviewScopeSelection{IncludePrefixes: []string{"pkg"}}
	updated.CampaignID = repoaudit.NewRepositoryReviewCampaignID()
	updated.CampaignRecoveryPending = true
	if projected := projectRepositoryReviewAutomation(updated); projected.ModelCoverageSketches != nil ||
		projected.ScopeSelection != nil || projected.CampaignID != "" ||
		projected.CampaignRecoveryPending ||
		!projected.Progress.ScopeFrozen {
		t.Fatalf(
			"API projection state: sketches=%#v selection=%#v frozen=%v",
			projected.ModelCoverageSketches, projected.ScopeSelection,
			projected.Progress.ScopeFrozen,
		)
	}
	updated.ScopeSelection = nil
	updated.Progress.ScopeFrozen = true
	if projected := projectRepositoryReviewAutomation(updated); projected.Progress.ScopeFrozen {
		t.Fatal("API projection trusted a non-durable scope_frozen marker")
	}
}

func TestRepositoryReviewAutomationStartAutoContinuesBoundedBatches(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	controller.probe = func(context.Context) (codexAccountLimitsResponse, error) {
		return codexAccountLimitsResponse{}, nil
	}
	var calls atomic.Int32
	controller.runBatch = func(
		_ context.Context,
		automation repoaudit.RepositoryReviewAutomation,
		runID string,
		observe workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		call := calls.Add(1)
		if automation.ID == "" || runID == "" {
			t.Fatal("fake batch received empty automation or run identity")
		}
		if err := observe(workflows.AgentUsage{
			Model: "cheap", PromptTokens: 40, CompletionTokens: 10, TotalTokens: 50,
		}); err != nil {
			return nil, err
		}
		remaining := 2
		reviewed := 1
		if call == 1 {
			cfg, err := config.LoadConfig(handler.configPath)
			if err != nil {
				return nil, err
			}
			cfg.ModelList[0].InputPricePerMTok = 9
			cfg.ModelList[0].OutputPricePerMTok = 13
			if err := config.SaveConfig(handler.configPath, cfg); err != nil {
				return nil, err
			}
		}
		if call == 2 {
			remaining = 0
			reviewed = 2
		}
		return &workflows.RunResult{
			RunID: runID, Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{"remainingFiles": remaining, "reviewedFiles": reviewed},
		}, nil
	}

	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.ModelPrices = map[string]repoaudit.RepositoryReviewModelPrice{
		"cheap": {InputPricePer1M: 7, OutputPricePer1M: 11},
	}
	automation, err = store.CreateAutomation(t.Context(), automation)
	if err != nil {
		t.Fatal(err)
	}
	response := repositoryReviewAutomationMutation(t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/start",
		map[string]any{"expected_version": automation.Version})
	if response.Code != http.StatusAccepted {
		t.Fatalf("start status=%d body=%s", response.Code, response.Body.String())
	}

	deadline := time.Now().Add(3 * time.Second)
	var completed repoaudit.RepositoryReviewAutomation
	for time.Now().Before(deadline) {
		current, found, getErr := store.GetAutomation(t.Context(), automation.ID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if found && current.Status == repoaudit.RepositoryReviewAutomationCompleted {
			completed = current
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if completed.ID == "" {
		t.Fatalf("automation did not complete; batches=%d", calls.Load())
	}
	stats := completed.ModelStats["cheap"]
	price := completed.ModelPrices["cheap"]
	if calls.Load() != 2 || len(completed.RunIDs) != 2 ||
		completed.Progress.CompletedBatches != 2 || completed.Progress.RemainingFiles != 0 ||
		completed.EffectiveAccountRef != "api" ||
		completed.Usage.TotalTokens != 100 || stats.Requests != 2 ||
		math.Abs(completed.EstimatedCostUSD-0.00055) > 0.0000001 ||
		price.InputPricePer1M != 9 || price.OutputPricePer1M != 13 {
		t.Fatalf("completed=%#v stats=%#v calls=%d", completed, stats, calls.Load())
	}
}

func TestRepositoryReviewAutomationNoProgressStopsAutoContinuationAfterOneBatch(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	controller.probe = func(context.Context) (codexAccountLimitsResponse, error) {
		return codexAccountLimitsResponse{}, nil
	}
	var calls atomic.Int32
	controller.runBatch = func(
		_ context.Context,
		_ repoaudit.RepositoryReviewAutomation,
		runID string,
		_ workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		call := calls.Add(1)
		remaining := 2
		reviewed := 0
		if call == 2 {
			remaining = 0
			reviewed = 1
		}
		return &workflows.RunResult{
			RunID: runID, Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{
				"remainingFiles": remaining, "reviewedFiles": reviewed,
			},
		}, nil
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation, err := store.CreateAutomation(t.Context(), testRepositoryReviewAutomation())
	if err != nil {
		t.Fatal(err)
	}
	response := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/start",
		map[string]any{"expected_version": automation.Version},
	)
	if response.Code != http.StatusAccepted {
		t.Fatalf("start status=%d body=%s", response.Code, response.Body.String())
	}
	paused := waitForRepositoryReviewAutomationStatus(
		t,
		store,
		automation.ID,
		repoaudit.RepositoryReviewAutomationPaused,
	)
	if calls.Load() != 1 || len(paused.RunIDs) != 1 ||
		paused.PauseReason != repoaudit.RepositoryReviewPauseNoProgress ||
		paused.Progress.CompletedBatches != 1 || paused.Progress.RemainingFiles != 2 {
		t.Fatalf("no-progress automation=%#v calls=%d", paused, calls.Load())
	}
	time.Sleep(50 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatalf("no-progress auto-continuation launched %d batches", calls.Load())
	}
	response = repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/resume",
		map[string]any{"expected_version": paused.Version},
	)
	if response.Code != http.StatusAccepted {
		t.Fatalf("resume status=%d body=%s", response.Code, response.Body.String())
	}
	completed := waitForRepositoryReviewAutomationStatus(
		t,
		store,
		automation.ID,
		repoaudit.RepositoryReviewAutomationCompleted,
	)
	if calls.Load() != 2 || completed.Progress.CompletedBatches != 2 {
		t.Fatalf("resumed no-progress automation=%#v calls=%d", completed, calls.Load())
	}
}

func TestRepositoryReviewAutomationStartPersistsResolvedCommitBeforeBatch(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	resolvedCommit := strings.Repeat("1", 40)
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		return resolvedCommit, nil
	}
	type batchObservation struct {
		argument    repoaudit.RepositoryReviewAutomation
		persisted   repoaudit.RepositoryReviewAutomation
		ledger      repoaudit.RepositoryState
		ledgerFound bool
		found       bool
		err         error
	}
	observed := make(chan batchObservation, 1)
	release := make(chan struct{})
	defer close(release)
	controller.runBatch = func(
		_ context.Context,
		automation repoaudit.RepositoryReviewAutomation,
		runID string,
		_ workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		persisted, found, getErr := store.GetAutomation(context.Background(), automation.ID)
		ledger, ledgerFound, ledgerErr := store.Get(
			repoaudit.CanonicalRepositoryIdentity(automation.Repository),
		)
		if getErr == nil {
			getErr = ledgerErr
		}
		observed <- batchObservation{
			argument: automation, persisted: persisted, ledger: ledger,
			ledgerFound: ledgerFound, found: found, err: getErr,
		}
		<-release
		return &workflows.RunResult{
			RunID: runID, Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{"commit": resolvedCommit, "remainingFiles": 0},
		}, nil
	}

	automation, err := store.CreateAutomation(t.Context(), testRepositoryReviewAutomation())
	if err != nil {
		t.Fatal(err)
	}
	response := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/start",
		map[string]any{"expected_version": automation.Version},
	)
	if response.Code != http.StatusAccepted {
		t.Fatalf("start status=%d body=%s", response.Code, response.Body.String())
	}
	var started struct {
		Automation repoaudit.RepositoryReviewAutomation `json:"automation"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &started); err != nil {
		t.Fatal(err)
	}
	if started.Automation.Status != repoaudit.RepositoryReviewAutomationRunning ||
		started.Automation.ResolvedCommitSHA != resolvedCommit ||
		started.Automation.ActiveRunID == "" {
		t.Fatalf("started automation=%#v", started.Automation)
	}
	select {
	case observation := <-observed:
		if observation.err != nil || !observation.found || !observation.ledgerFound {
			t.Fatalf("persisted automation found=%v ledger=%v err=%v",
				observation.found, observation.ledgerFound, observation.err)
		}
		if observation.argument.ResolvedCommitSHA != resolvedCommit ||
			observation.persisted.ResolvedCommitSHA != resolvedCommit ||
			observation.argument.ActiveRunID == "" ||
			observation.argument.ActiveRunID != observation.persisted.ActiveRunID ||
			observation.persisted.Status != repoaudit.RepositoryReviewAutomationRunning ||
			observation.argument.CampaignID == "" ||
			observation.argument.CampaignID != observation.persisted.CampaignID ||
			observation.ledger.CurrentCampaign == nil ||
			observation.ledger.CurrentCampaign.ID != observation.argument.CampaignID ||
			observation.ledger.CurrentCampaign.CommitSHA != resolvedCommit {
			t.Fatalf(
				"batch argument=%#v persisted=%#v",
				observation.argument,
				observation.persisted,
			)
		}
	case <-time.After(time.Second):
		t.Fatal("repository review batch did not observe admitted commit")
	}
}

func TestRepositoryReviewAutomationStartAfterConfigurationResetCreatesCampaignDespiteHistory(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	commit := strings.Repeat("8", 40)
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		return commit, nil
	}
	started := make(chan repoaudit.RepositoryReviewAutomation, 1)
	controller.runBatch = func(
		_ context.Context,
		automation repoaudit.RepositoryReviewAutomation,
		runID string,
		_ workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		started <- automation
		return &workflows.RunResult{
			RunID: runID, Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{"commit": commit, "remainingFiles": 0},
		}, nil
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	input := testRepositoryReviewAutomation()
	input.Repository = "https://github.com/acme/core.git"
	input.RunIDs = []string{"wr_historical_before_config_change"}
	input.CampaignID = ""
	input.StartedAt = time.Time{}
	input.Progress = repoaudit.RepositoryReviewProgress{}
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := controller.startAutomation(
		t.Context(), automation.ID, automation.Version, false, "start",
	)
	if err != nil || resumed.CampaignID == "" || resumed.ActiveRunID == "" {
		t.Fatalf("reset start=%#v err=%v", resumed, err)
	}
	select {
	case observed := <-started:
		ledger, found, ledgerErr := store.Get(repoaudit.CanonicalRepositoryIdentity(observed.Repository))
		if ledgerErr != nil || !found || ledger.CurrentCampaign == nil ||
			ledger.CurrentCampaign.ID != observed.CampaignID || observed.CampaignID == "" {
			t.Fatalf("reset start batch=%#v ledger=%#v found=%v err=%v",
				observed, ledger.CurrentCampaign, found, ledgerErr)
		}
	case <-time.After(time.Second):
		t.Fatal("configuration-reset start did not launch")
	}
}

func TestRepositoryReviewAutomationAuthorizationFailureAppendsNoRun(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	resolvedCommit := strings.Repeat("4", 40)
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		return resolvedCommit, nil
	}
	var batchCalls atomic.Int32
	controller.runBatch = func(
		context.Context,
		repoaudit.RepositoryReviewAutomation,
		string,
		workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		batchCalls.Add(1)
		return nil, errors.New("batch must not start")
	}
	originalUpdate := controller.update
	poisoned := false
	controller.update = func(
		ctx context.Context,
		candidateStore repoaudit.Store,
		id string,
		expectedVersion int64,
		mutate func(*repoaudit.RepositoryReviewAutomation) error,
	) (repoaudit.RepositoryReviewAutomation, error) {
		updated, updateErr := originalUpdate(
			ctx, candidateStore, id, expectedVersion, mutate,
		)
		if updateErr == nil && !poisoned && updated.CampaignID != "" && updated.ActiveRunID == "" {
			poisoned = true
			_, updateErr = candidateStore.BeginCampaign(ctx, repoaudit.BeginCampaignRequest{
				Repository: repoaudit.CanonicalRepositoryIdentity(updated.Repository),
				CampaignID: updated.CampaignID, CommitSHA: strings.Repeat("5", 40),
				ExpectedReviewVersion: 0, Exact: true,
			})
		}
		return updated, updateErr
	}
	automation, err := store.CreateAutomation(t.Context(), testRepositoryReviewAutomation())
	if err != nil {
		t.Fatal(err)
	}
	if _, startErr := controller.startAutomation(
		t.Context(), automation.ID, automation.Version, false, "start",
	); !errors.Is(startErr, repoaudit.ErrConflict) {
		t.Fatalf("authorization error = %v", startErr)
	}
	current, found, err := store.GetAutomation(t.Context(), automation.ID)
	if err != nil || !found || current.Status != repoaudit.RepositoryReviewAutomationFailed ||
		current.CampaignID == "" || current.ActiveRunID != "" || len(current.RunIDs) != 0 ||
		batchCalls.Load() != 0 {
		t.Fatalf("authorization failure state=%#v found=%v err=%v batches=%d",
			current, found, err, batchCalls.Load())
	}
}

func TestRepositoryReviewAutomationLegacyRecoveryCannotFallThroughWithoutScope(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	previousRunners := newWorkflowRuntimeRunners
	t.Cleanup(func() { newWorkflowRuntimeRunners = previousRunners })
	profileRunner := &repositoryReviewRecoveryProfileRunner{profile: workflows.RepositoryReviewModelProfile{
		Revision: "sha256:resolved-profile", AccountRef: "resolved-account",
		ReviewerModels:         []string{"fallback-a", "fallback-b"},
		IncludeDefaultReviewer: true, MaxContentBytes: 282624,
	}}
	newWorkflowRuntimeRunners = func(string) workflowRuntimeRunners {
		return workflowRuntimeRunners{Agents: profileRunner}
	}
	commit := strings.Repeat("6", 40)
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		return commit, nil
	}
	var recoveryCalls atomic.Int32
	controller.recoverCampaign = func(
		_ context.Context,
		_ repoaudit.Store,
		_ string,
		automation repoaudit.RepositoryReviewAutomation,
		resolvedCommit string,
		resolved workflows.RepositoryReviewModelProfile,
	) (repoaudit.RepositoryReviewAutomation, error) {
		recoveryCalls.Add(1)
		effectiveMax, boundErr := workflows.RepositoryBugFinderEffectiveMaxContentBytes(
			automation.MaxContentBytes, resolved.MaxContentBytes,
		)
		required, requiredErr := workflows.RepositoryBugFinderRequiredAssignments(
			resolved.ReviewerModels, resolved.IncludeDefaultReviewer,
		)
		if boundErr != nil || requiredErr != nil || effectiveMax != 282624 || required != 4 ||
			resolved.AccountRef != "resolved-account" {
			t.Fatalf("resolved recovery profile=%#v max=%d required=%d bound_err=%v required_err=%v",
				resolved, effectiveMax, required, boundErr, requiredErr)
		}
		if automation.CampaignID != "" || automation.ActiveRunID != "" || resolvedCommit != commit {
			t.Fatalf("legacy recovery preflight saw mutated automation: %#v", automation)
		}
		return automation, nil
	}
	var batchCalls atomic.Int32
	controller.runBatch = func(
		context.Context,
		repoaudit.RepositoryReviewAutomation,
		string,
		workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		batchCalls.Add(1)
		return nil, errors.New("batch must not start")
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	input := testRepositoryReviewAutomation()
	input.Repository = "https://github.com/acme/core.git"
	input.Status = repoaudit.RepositoryReviewAutomationFailed
	input.PauseReason = repoaudit.RepositoryReviewPauseRunFailed
	input.ResolvedCommitSHA = commit
	input.RunIDs = []string{"wr_legacy"}
	input.StartedAt = time.Now().Add(-time.Hour)
	input.MaxContentBytes = 524288
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	if _, startErr := controller.startAutomation(
		t.Context(), automation.ID, automation.Version, false, "resume",
	); startErr == nil || !strings.Contains(startErr.Error(), "invalid installed authority") {
		t.Fatalf("legacy recovery error = %v", startErr)
	}
	current, found, err := store.GetAutomation(t.Context(), automation.ID)
	if err != nil || !found || current.CampaignID != "" || current.ActiveRunID != "" ||
		!reflect.DeepEqual(current.RunIDs, input.RunIDs) || recoveryCalls.Load() != 1 ||
		batchCalls.Load() != 0 {
		t.Fatalf("prepared legacy recovery=%#v found=%v err=%v recovery=%d batch=%d",
			current, found, err, recoveryCalls.Load(), batchCalls.Load())
	}
}

func TestRepositoryReviewAutomationLegacyAutomaticHandoffUsesRealRecoveryAdapter(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	if controller.recoverCampaign == nil {
		t.Fatal("production legacy recovery adapter is not wired")
	}
	previousRunners := newWorkflowRuntimeRunners
	t.Cleanup(func() { newWorkflowRuntimeRunners = previousRunners })
	newWorkflowRuntimeRunners = func(string) workflowRuntimeRunners {
		return workflowRuntimeRunners{Agents: &repositoryReviewRecoveryProfileRunner{
			profile: workflows.RepositoryReviewModelProfile{
				Revision: "sha256:resolved-profile", AccountRef: "resolved-account",
				ReviewerModels: []string{"cheap"}, MaxContentBytes: 65536,
			},
		}}
	}
	commit := strings.Repeat("9", 40)
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		return commit, nil
	}
	var batchCalls atomic.Int32
	controller.runBatch = func(
		context.Context,
		repoaudit.RepositoryReviewAutomation,
		string,
		workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		batchCalls.Add(1)
		return nil, errors.New("batch must not start")
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	input := testRepositoryReviewAutomation()
	input.Repository = "https://github.com/acme/core.git"
	input.Status = repoaudit.RepositoryReviewAutomationIdle
	input.Progress.Stage = "next batch queued"
	input.Progress.TotalBatches = 2
	input.Progress.CompletedBatches = 1
	input.ResolvedCommitSHA = commit
	input.RunIDs = []string{"wr_legacy"}
	input.StartedAt = time.Now().Add(-time.Hour)
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	if _, startErr := controller.startAutomation(
		t.Context(), automation.ID, automation.Version, false, "start",
	); !errors.Is(startErr, os.ErrNotExist) {
		t.Fatalf("automatic legacy handoff error = %v", startErr)
	}
	current, found, err := store.GetAutomation(t.Context(), automation.ID)
	if err != nil || !found || current.ActiveRunID != "" || current.CampaignID != "" ||
		!reflect.DeepEqual(current.RunIDs, input.RunIDs) || batchCalls.Load() != 0 {
		t.Fatalf("legacy handoff=%#v found=%v err=%v batches=%d",
			current, found, err, batchCalls.Load())
	}
}

func TestRepositoryReviewAutomationChangedCommitOrProfileResetStartsCleanCampaign(t *testing.T) {
	for _, test := range []struct {
		name          string
		startedAt     time.Time
		remembered    string
		selection     string
		startedAction func(
			*repositoryReviewController,
			repoaudit.RepositoryReviewAutomation,
		) (repoaudit.RepositoryReviewAutomation, error)
	}{
		{
			name:      "changed commit",
			startedAt: time.Now().Add(-time.Hour), remembered: strings.Repeat("a", 40),
			selection: strings.Repeat("b", 40),
			startedAction: func(
				controller *repositoryReviewController,
				automation repoaudit.RepositoryReviewAutomation,
			) (repoaudit.RepositoryReviewAutomation, error) {
				return controller.startAutomationAtCommit(
					t.Context(), automation.ID, automation.Version, false, "resume", strings.Repeat("b", 40),
				)
			},
		},
		{
			name:       "profile reset",
			remembered: strings.Repeat("c", 40), selection: strings.Repeat("c", 40),
			startedAction: func(
				controller *repositoryReviewController,
				automation repoaudit.RepositoryReviewAutomation,
			) (repoaudit.RepositoryReviewAutomation, error) {
				return controller.startAutomation(
					t.Context(), automation.ID, automation.Version, false, "resume",
				)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
			t.Cleanup(handler.Shutdown)
			controller := handler.repositoryReviewControllerInstance()
			controller.resolveCommit = func(
				_ context.Context,
				_ *config.Config,
				_ repoaudit.RepositoryReviewAutomation,
				selection string,
			) (string, error) {
				if selection != "" {
					return selection, nil
				}
				return test.selection, nil
			}
			var recoveryCalls atomic.Int32
			controller.recoverCampaign = func(
				context.Context,
				repoaudit.Store,
				string,
				repoaudit.RepositoryReviewAutomation,
				string,
				workflows.RepositoryReviewModelProfile,
			) (repoaudit.RepositoryReviewAutomation, error) {
				recoveryCalls.Add(1)
				return repoaudit.RepositoryReviewAutomation{}, errors.New("legacy recovery must not run")
			}
			started := make(chan repoaudit.RepositoryReviewAutomation, 1)
			controller.runBatch = func(
				_ context.Context,
				automation repoaudit.RepositoryReviewAutomation,
				runID string,
				_ workflows.AgentUsageObserver,
			) (*workflows.RunResult, error) {
				started <- automation
				return &workflows.RunResult{
					RunID: runID, Status: workflows.RunStatusSucceeded,
					Outputs: map[string]any{"commit": test.selection, "remainingFiles": 0},
				}, nil
			}
			store, err := handler.repositoryReviewStore()
			if err != nil {
				t.Fatal(err)
			}
			input := testRepositoryReviewAutomation()
			input.Repository = "https://github.com/acme/core.git"
			input.Status = repoaudit.RepositoryReviewAutomationFailed
			input.PauseReason = repoaudit.RepositoryReviewPauseRunFailed
			input.ResolvedCommitSHA = test.remembered
			input.RunIDs = []string{"wr_before_reset"}
			input.StartedAt = test.startedAt
			input.Progress = repoaudit.RepositoryReviewProgress{
				CompletedBatches: 2, TotalBatches: 3, ReviewedFiles: 7, RemainingFiles: 3,
			}
			automation, err := store.CreateAutomation(t.Context(), input)
			if err != nil {
				t.Fatal(err)
			}
			resumed, err := test.startedAction(controller, automation)
			if err != nil || resumed.CampaignID == "" || resumed.ActiveRunID == "" ||
				recoveryCalls.Load() != 0 || resumed.Progress.CompletedBatches != 0 ||
				resumed.Progress.ReviewedFiles != 0 {
				t.Fatalf("clean resumed campaign=%#v recovery=%d err=%v",
					resumed, recoveryCalls.Load(), err)
			}
			select {
			case observed := <-started:
				if observed.CampaignID != resumed.CampaignID || observed.ResolvedCommitSHA != test.selection {
					t.Fatalf("clean campaign batch=%#v", observed)
				}
			case <-time.After(time.Second):
				t.Fatal("clean campaign did not start")
			}
		})
	}
}

func TestRepositoryReviewAutomationLegacyRecoveryReplaysAfterScopeCASConflict(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	previousRunners := newWorkflowRuntimeRunners
	t.Cleanup(func() { newWorkflowRuntimeRunners = previousRunners })
	profileRunner := &repositoryReviewRecoveryProfileRunner{profile: workflows.RepositoryReviewModelProfile{
		Revision: "sha256:resolved-profile", AccountRef: "resolved-account",
		ReviewerModels: []string{"cheap"}, MaxContentBytes: 65536,
	}}
	newWorkflowRuntimeRunners = func(string) workflowRuntimeRunners {
		return workflowRuntimeRunners{Agents: profileRunner}
	}
	commit := strings.Repeat("7", 40)
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
	var recoveryCalls atomic.Int32
	scopePlan := repoaudit.RepositoryReviewScopePlan{
		CommitSHA: commit, PolicyHash: strings.Repeat("a", 64),
		Hash: strings.Repeat("b", 64), Summary: "Recovered legacy campaign scope",
	}
	scopeSelection := repoaudit.RepositoryReviewScopeSelection{IncludePrefixes: []string{"pkg"}}
	controller.recoverCampaign = func(
		ctx context.Context,
		candidateStore repoaudit.Store,
		_ string,
		automation repoaudit.RepositoryReviewAutomation,
		resolvedCommit string,
		_ workflows.RepositoryReviewModelProfile,
	) (repoaudit.RepositoryReviewAutomation, error) {
		call := recoveryCalls.Add(1)
		if automation.CampaignID == "" {
			campaignID := repoaudit.NewRepositoryReviewCampaignID()
			installed, installErr := candidateStore.UpdateAutomation(
				ctx, automation.ID, automation.Version,
				func(candidate *repoaudit.RepositoryReviewAutomation) error {
					candidate.CampaignID = campaignID
					candidate.CampaignRecoveryPending = true
					candidate.ScopePlan = scopePlan
					candidate.ResolvedCommitSHA = resolvedCommit
					selection := scopeSelection
					candidate.ScopeSelection = &selection
					return nil
				},
			)
			if installErr != nil {
				return repoaudit.RepositoryReviewAutomation{}, installErr
			}
			automation = installed
		}
		identity := repoaudit.CanonicalRepositoryIdentity(automation.Repository)
		state, _, getErr := candidateStore.Get(identity)
		if getErr != nil {
			return repoaudit.RepositoryReviewAutomation{}, getErr
		}
		if _, beginErr := candidateStore.BeginCampaign(ctx, repoaudit.BeginCampaignRequest{
			Repository: identity, CampaignID: automation.CampaignID,
			CommitSHA: commit, ExpectedReviewVersion: state.ReviewVersion, Exact: false,
		}); beginErr != nil {
			return repoaudit.RepositoryReviewAutomation{}, beginErr
		}
		if call == 1 {
			return repoaudit.RepositoryReviewAutomation{}, repoaudit.ErrConflict
		}
		return candidateStore.UpdateAutomation(
			ctx, automation.ID, automation.Version,
			func(candidate *repoaudit.RepositoryReviewAutomation) error {
				if candidate.CampaignID != automation.CampaignID || candidate.ScopeSelection == nil ||
					candidate.ScopePlan.Hash != scopePlan.Hash {
					return repoaudit.ErrConflict
				}
				candidate.CampaignRecoveryPending = false
				return nil
			},
		)
	}
	started := make(chan repoaudit.RepositoryReviewAutomation, 1)
	controller.runBatch = func(
		_ context.Context,
		automation repoaudit.RepositoryReviewAutomation,
		runID string,
		_ workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		started <- automation
		return &workflows.RunResult{
			RunID: runID, Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{"commit": commit, "remainingFiles": 0},
		}, nil
	}
	input := testRepositoryReviewAutomation()
	input.Repository = "https://github.com/acme/core.git"
	input.Status = repoaudit.RepositoryReviewAutomationFailed
	input.PauseReason = repoaudit.RepositoryReviewPauseRunFailed
	input.ResolvedCommitSHA = ""
	input.ScopePlan = repoaudit.RepositoryReviewScopePlan{
		CommitSHA: commit, PolicyHash: strings.Repeat("d", 64),
		Hash: strings.Repeat("e", 64), Summary: "Legacy remembered scope",
	}
	input.RunIDs = []string{"wr_legacy"}
	input.StartedAt = time.Now().Add(-time.Hour)
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	if _, startErr := controller.startAutomation(
		t.Context(), automation.ID, automation.Version, false, "resume",
	); !errors.Is(startErr, repoaudit.ErrConflict) {
		t.Fatalf("injected scope CAS error = %v", startErr)
	}
	prepared, found, err := store.GetAutomation(t.Context(), automation.ID)
	if err != nil || !found || prepared.CampaignID == "" || !prepared.CampaignRecoveryPending ||
		prepared.ActiveRunID != "" ||
		!reflect.DeepEqual(prepared.RunIDs, input.RunIDs) {
		t.Fatalf("prepared automation=%#v found=%v err=%v", prepared, found, err)
	}
	resumed, err := controller.startAutomation(
		t.Context(), prepared.ID, prepared.Version, false, "resume",
	)
	if err != nil || resumed.ActiveRunID == "" || resumed.CampaignRecoveryPending ||
		resumed.CampaignID != prepared.CampaignID ||
		recoveryCalls.Load() != 2 {
		t.Fatalf("resumed=%#v recovery_calls=%d err=%v", resumed, recoveryCalls.Load(), err)
	}
	select {
	case observed := <-started:
		if observed.CampaignID != prepared.CampaignID || observed.ScopeSelection == nil ||
			observed.ScopePlan.Hash == "" {
			t.Fatalf("replayed recovery batch=%#v", observed)
		}
	case <-time.After(time.Second):
		t.Fatal("replayed legacy recovery did not start")
	}
}

func TestRepositoryReviewAutomationAutoContinueReusesResolvedCommit(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	rememberedCommit := strings.Repeat("2", 40)
	newerCommit := strings.Repeat("3", 40)
	var resolveCalls atomic.Int32
	controller.resolveCommit = func(
		_ context.Context,
		_ *config.Config,
		_ repoaudit.RepositoryReviewAutomation,
		revision string,
	) (string, error) {
		if revision != "" {
			return strings.ToLower(strings.TrimSpace(revision)), nil
		}
		if resolveCalls.Add(1) == 1 {
			return rememberedCommit, nil
		}
		return newerCommit, nil
	}
	seenCommits := make(chan string, 2)
	seenCampaigns := make(chan string, 2)
	var batches atomic.Int32
	controller.runBatch = func(
		_ context.Context,
		automation repoaudit.RepositoryReviewAutomation,
		runID string,
		_ workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		seenCommits <- automation.ResolvedCommitSHA
		seenCampaigns <- automation.CampaignID
		remaining := 1
		if batches.Add(1) == 2 {
			remaining = 0
		}
		return &workflows.RunResult{
			RunID: runID, Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{
				"commit": rememberedCommit, "remainingFiles": remaining, "reviewedFiles": 1,
			},
		}, nil
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation, err := store.CreateAutomation(t.Context(), testRepositoryReviewAutomation())
	if err != nil {
		t.Fatal(err)
	}
	response := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/start",
		map[string]any{"expected_version": automation.Version},
	)
	if response.Code != http.StatusAccepted {
		t.Fatalf("start status=%d body=%s", response.Code, response.Body.String())
	}
	completed := waitForRepositoryReviewAutomationStatus(
		t,
		store,
		automation.ID,
		repoaudit.RepositoryReviewAutomationCompleted,
	)
	close(seenCommits)
	close(seenCampaigns)
	var observed []string
	for commit := range seenCommits {
		observed = append(observed, commit)
	}
	var observedCampaigns []string
	for campaignID := range seenCampaigns {
		observedCampaigns = append(observedCampaigns, campaignID)
	}
	if !reflect.DeepEqual(observed, []string{rememberedCommit, rememberedCommit}) ||
		len(observedCampaigns) != 2 || observedCampaigns[0] == "" ||
		observedCampaigns[0] != observedCampaigns[1] ||
		resolveCalls.Load() != 1 || batches.Load() != 2 ||
		completed.ResolvedCommitSHA != rememberedCommit {
		t.Fatalf(
			"observed commits=%#v resolver calls=%d batches=%d completed=%#v",
			observed,
			resolveCalls.Load(),
			batches.Load(),
			completed,
		)
	}
}

func TestRepositoryReviewAutomationCommitOptionsExposeRememberedAndLatest(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	rememberedCommit := strings.Repeat("4", 40)
	latestCommit := strings.Repeat("5", 40)
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		return latestCommit, nil
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	input := testRepositoryReviewAutomation()
	input.Status = repoaudit.RepositoryReviewAutomationPaused
	input.PauseReason = repoaudit.RepositoryReviewPauseManual
	input.PauseDetail = "paused between batches"
	input.ResolvedCommitSHA = rememberedCommit
	input.ScopePlan = repoaudit.RepositoryReviewScopePlan{
		CommitSHA: rememberedCommit, PolicyHash: strings.Repeat("a", 64),
		Hash: strings.Repeat("b", 64), Summary: "Frozen campaign scope",
	}
	input.ScopeSelection = &repoaudit.RepositoryReviewScopeSelection{
		IncludePrefixes: []string{"pkg"},
	}
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodGet,
			"/api/repository-reviews/automations/"+automation.ID+"/commit-options",
			nil,
		),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("commit options status=%d body=%s", response.Code, response.Body.String())
	}
	var options repositoryReviewCommitOptionsResponse
	if decodeErr := json.Unmarshal(response.Body.Bytes(), &options); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if options.ExpectedVersion != automation.Version || !options.NewerCommitAvailable ||
		options.Remembered.SHA != rememberedCommit ||
		options.Remembered.ShortSHA != rememberedCommit[:8] ||
		options.Remembered.URL != "https://github.com/acme/core/commit/"+rememberedCommit ||
		options.Latest.SHA != latestCommit || options.Latest.ShortSHA != latestCommit[:8] ||
		options.Latest.URL != "https://github.com/acme/core/commit/"+latestCommit {
		t.Fatalf("commit options=%#v", options)
	}
}

func TestRepositoryReviewAutomationCommitOptionsAcceptFailed(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	rememberedCommit := strings.Repeat("4", 40)
	latestCommit := strings.Repeat("5", 40)
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		return latestCommit, nil
	}
	var batchCalls atomic.Int32
	controller.runBatch = func(
		context.Context,
		repoaudit.RepositoryReviewAutomation,
		string,
		workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		batchCalls.Add(1)
		return nil, errors.New("unexpected batch")
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	input := testRepositoryReviewAutomation()
	input.Status = repoaudit.RepositoryReviewAutomationFailed
	input.PauseReason = repoaudit.RepositoryReviewPauseRunFailed
	input.PauseDetail = "temporary provider outage"
	input.ResolvedCommitSHA = rememberedCommit
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	mux.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodGet,
			"/api/repository-reviews/automations/"+automation.ID+"/commit-options",
			nil,
		),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("failed commit options status=%d body=%s", response.Code, response.Body.String())
	}
	var options repositoryReviewCommitOptionsResponse
	if decodeErr := json.Unmarshal(response.Body.Bytes(), &options); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if options.ExpectedVersion != automation.Version || !options.NewerCommitAvailable ||
		options.Remembered.SHA != rememberedCommit || options.Latest.SHA != latestCommit {
		t.Fatalf("failed commit options=%#v", options)
	}

	resume := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/resume",
		map[string]any{"expected_version": options.ExpectedVersion},
	)
	if resume.Code != http.StatusConflict ||
		!strings.Contains(resume.Body.String(), "repository_review_commit_selection_required") {
		t.Fatalf("failed resume status=%d body=%s", resume.Code, resume.Body.String())
	}
	current, found, err := store.GetAutomation(t.Context(), automation.ID)
	if err != nil || !found || current.Status != repoaudit.RepositoryReviewAutomationFailed ||
		current.Version != automation.Version || current.ResolvedCommitSHA != rememberedCommit ||
		batchCalls.Load() != 0 {
		t.Fatalf(
			"failed campaign after unfenced resume=%#v found=%v err=%v batches=%d",
			current,
			found,
			err,
			batchCalls.Load(),
		)
	}
}

func TestRepositoryReviewAutomationCommitOptionsNormalizeLegacyTarget(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	latestCommit := strings.Repeat("5", 40)
	var resolvedTarget repoaudit.RepositoryReviewAutomation
	controller.resolveCommit = func(
		_ context.Context,
		_ *config.Config,
		automation repoaudit.RepositoryReviewAutomation,
		_ string,
	) (string, error) {
		resolvedTarget = automation
		return latestCommit, nil
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	input := testRepositoryReviewAutomation()
	input.Repository = "owner/repo"
	input.Ref = "HEAD"
	input.Status = repoaudit.RepositoryReviewAutomationPaused
	input.PauseReason = repoaudit.RepositoryReviewPauseServiceRestart
	input.PauseDetail = "legacy run paused after restart"
	input.ResolvedCommitSHA = strings.Repeat("4", 40)
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodGet,
			"/api/repository-reviews/automations/"+automation.ID+"/commit-options",
			nil,
		),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("legacy commit options status=%d body=%s", response.Code, response.Body.String())
	}
	if resolvedTarget.Repository != "https://github.com/owner/repo.git" || resolvedTarget.Ref != "" {
		t.Fatalf("legacy resolution target=%#v", resolvedTarget)
	}
	var options repositoryReviewCommitOptionsResponse
	if err := json.Unmarshal(response.Body.Bytes(), &options); err != nil {
		t.Fatal(err)
	}
	if options.Remembered.URL != "https://github.com/owner/repo/commit/"+input.ResolvedCommitSHA ||
		options.Latest.URL != "https://github.com/owner/repo/commit/"+latestCommit {
		t.Fatalf("legacy commit options=%#v", options)
	}
}

func TestRepositoryReviewAutomationResumeRechecksTipAfterCommitOptions(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	rememberedCommit := strings.Repeat("6", 40)
	latestCommit := strings.Repeat("7", 40)
	var resolveCalls atomic.Int32
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		if resolveCalls.Add(1) == 1 {
			return rememberedCommit, nil
		}
		return latestCommit, nil
	}
	var batchCalls atomic.Int32
	controller.runBatch = func(
		context.Context,
		repoaudit.RepositoryReviewAutomation,
		string,
		workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		batchCalls.Add(1)
		return nil, errors.New("unexpected batch")
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	input := testRepositoryReviewAutomation()
	input.Status = repoaudit.RepositoryReviewAutomationPaused
	input.PauseReason = repoaudit.RepositoryReviewPauseManual
	input.PauseDetail = "paused between batches"
	input.ResolvedCommitSHA = rememberedCommit
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	optionsResponse := httptest.NewRecorder()
	mux.ServeHTTP(
		optionsResponse,
		httptest.NewRequest(
			http.MethodGet,
			"/api/repository-reviews/automations/"+automation.ID+"/commit-options",
			nil,
		),
	)
	if optionsResponse.Code != http.StatusOK {
		t.Fatalf("commit options status=%d body=%s", optionsResponse.Code, optionsResponse.Body.String())
	}
	var options repositoryReviewCommitOptionsResponse
	if unmarshalErr := json.Unmarshal(optionsResponse.Body.Bytes(), &options); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if options.NewerCommitAvailable || options.Latest.SHA != rememberedCommit {
		t.Fatalf("initial commit options=%#v", options)
	}
	response := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/resume",
		map[string]any{"expected_version": automation.Version},
	)
	if response.Code != http.StatusConflict ||
		!strings.Contains(response.Body.String(), "repository_review_commit_selection_required") {
		t.Fatalf("resume status=%d body=%s", response.Code, response.Body.String())
	}
	current, found, err := store.GetAutomation(t.Context(), automation.ID)
	if err != nil || !found || current.Status != repoaudit.RepositoryReviewAutomationPaused ||
		current.ResolvedCommitSHA != rememberedCommit || current.Version != automation.Version ||
		batchCalls.Load() != 0 || resolveCalls.Load() != 2 {
		t.Fatalf("current=%#v found=%v err=%v batch calls=%d", current, found, err, batchCalls.Load())
	}
}

func TestRepositoryReviewAutomationResumePersistsExactCommitSelection(t *testing.T) {
	rememberedCommit := strings.Repeat("8", 40)
	latestCommit := strings.Repeat("9", 40)
	customCommit := strings.Repeat("b", 40)
	for _, testCase := range []struct {
		name      string
		selection string
	}{
		{name: "remembered", selection: rememberedCommit},
		{name: "latest", selection: latestCommit},
		{name: "custom", selection: customCommit},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
			t.Cleanup(handler.Shutdown)
			controller := handler.repositoryReviewControllerInstance()
			controller.resolveCommit = func(
				_ context.Context,
				_ *config.Config,
				_ repoaudit.RepositoryReviewAutomation,
				revision string,
			) (string, error) {
				if revision == "" {
					return latestCommit, nil
				}
				return strings.ToLower(strings.TrimSpace(revision)), nil
			}
			seen := make(chan repoaudit.RepositoryReviewAutomation, 1)
			controller.runBatch = func(
				_ context.Context,
				automation repoaudit.RepositoryReviewAutomation,
				runID string,
				_ workflows.AgentUsageObserver,
			) (*workflows.RunResult, error) {
				seen <- automation
				return &workflows.RunResult{
					RunID: runID, Status: workflows.RunStatusSucceeded,
					Outputs: map[string]any{
						"commit": testCase.selection, "remainingFiles": 0,
					},
				}, nil
			}
			store, err := handler.repositoryReviewStore()
			if err != nil {
				t.Fatal(err)
			}
			input := testRepositoryReviewAutomation()
			input.Status = repoaudit.RepositoryReviewAutomationPaused
			input.PauseReason = repoaudit.RepositoryReviewPauseManual
			input.PauseDetail = "paused between batches"
			input.ResolvedCommitSHA = rememberedCommit
			automation, err := store.CreateAutomation(t.Context(), input)
			if err != nil {
				t.Fatal(err)
			}
			response := repositoryReviewAutomationMutation(
				t,
				mux,
				http.MethodPost,
				"/api/repository-reviews/automations/"+automation.ID+"/resume",
				map[string]any{
					"expected_version": automation.Version,
					"commit_sha":       testCase.selection,
				},
			)
			if response.Code != http.StatusAccepted {
				t.Fatalf("resume status=%d body=%s", response.Code, response.Body.String())
			}
			var resumed struct {
				Automation repoaudit.RepositoryReviewAutomation `json:"automation"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &resumed); err != nil {
				t.Fatal(err)
			}
			if resumed.Automation.ResolvedCommitSHA != testCase.selection ||
				resumed.Automation.Status != repoaudit.RepositoryReviewAutomationRunning {
				t.Fatalf("resumed automation=%#v", resumed.Automation)
			}
			select {
			case observed := <-seen:
				if observed.ResolvedCommitSHA != testCase.selection {
					t.Fatalf("batch automation=%#v", observed)
				}
			case <-time.After(time.Second):
				t.Fatal("resumed batch did not start")
			}
			completed := waitForRepositoryReviewAutomationStatus(
				t,
				store,
				automation.ID,
				repoaudit.RepositoryReviewAutomationCompleted,
			)
			if completed.ResolvedCommitSHA != testCase.selection {
				t.Fatalf("completed automation=%#v", completed)
			}
		})
	}
}

func TestRepositoryReviewAutomationResumeFailedPreservesCampaignState(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	rememberedCommit := strings.Repeat("c", 40)
	type batchObservation struct {
		automation repoaudit.RepositoryReviewAutomation
		runID      string
	}
	batchStarted := make(chan batchObservation, 1)
	releaseBatch := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseBatch) }) }
	defer release()
	controller.runBatch = func(
		_ context.Context,
		automation repoaudit.RepositoryReviewAutomation,
		runID string,
		_ workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		batchStarted <- batchObservation{automation: automation, runID: runID}
		<-releaseBatch
		return &workflows.RunResult{
			RunID: runID, Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{
				"commit": rememberedCommit, "remainingFiles": 0,
				"scopeSelection": repositoryReviewWorkflowObject(automation.ScopeSelection),
				"scopePlan":      repositoryReviewWorkflowObject(automation.ScopePlan),
			},
		}, nil
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	startedAt := time.Date(2026, 8, 25, 14, 30, 0, 0, time.UTC)
	usage := repoaudit.RepositoryReviewTokenUsage{
		PromptTokens: 120, CompletionTokens: 30, CachedTokens: 20, TotalTokens: 150,
	}
	progress := repoaudit.RepositoryReviewProgress{
		Stage: "failed", CompletedBatches: 2, TotalBatches: 4,
		ReviewedFiles: 7, RemainingFiles: 3, UnsupportedFiles: 1, Findings: 2,
	}
	previousRunIDs := []string{"wr_completed_batch", "wr_failed_batch"}
	input := testRepositoryReviewAutomation()
	input.Repository = "https://github.com/acme/core.git"
	input.Status = repoaudit.RepositoryReviewAutomationFailed
	input.PauseReason = repoaudit.RepositoryReviewPauseRunFailed
	input.PauseDetail = "temporary provider outage"
	input.ResolvedCommitSHA = rememberedCommit
	input.ScopePlan = repoaudit.RepositoryReviewScopePlan{
		CommitSHA: rememberedCommit, PolicyHash: strings.Repeat("a", 64),
		Hash: strings.Repeat("b", 64), Summary: "Frozen campaign scope",
	}
	input.ScopeSelection = &repoaudit.RepositoryReviewScopeSelection{
		IncludePrefixes: []string{"pkg"},
	}
	input.RunIDs = append([]string(nil), previousRunIDs...)
	input.Usage = usage
	input.EstimatedCostUSD = 0.0042
	input.Progress = progress
	input.StartedAt = startedAt
	input.CampaignID = repoaudit.NewRepositoryReviewCampaignID()
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.BeginCampaign(t.Context(), repoaudit.BeginCampaignRequest{
		Repository: repoaudit.CanonicalRepositoryIdentity(input.Repository),
		CampaignID: input.CampaignID, CommitSHA: rememberedCommit,
		ExpectedReviewVersion: 0, Exact: true,
	}); err != nil {
		t.Fatal(err)
	}

	response := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/resume",
		map[string]any{
			"expected_version": automation.Version,
			"commit_sha":       rememberedCommit,
		},
	)
	if response.Code != http.StatusAccepted {
		t.Fatalf("failed resume status=%d body=%s", response.Code, response.Body.String())
	}
	var resumed struct {
		Automation repoaudit.RepositoryReviewAutomation `json:"automation"`
	}
	if decodeErr := json.Unmarshal(response.Body.Bytes(), &resumed); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	newRunID := resumed.Automation.ActiveRunID
	expectedRunIDs := append(append([]string(nil), previousRunIDs...), newRunID)
	expectedQueuedProgress := progress
	expectedQueuedProgress.Stage = "queued"
	expectedQueuedProgress.RawFindings = 0
	expectedQueuedProgress.DeduplicatedFindings = 0
	expectedQueuedProgress.Findings = 0
	expectedQueuedProgress.FindingAggregates = 0
	expectedQueuedProgress.PendingFindingMappings = 0
	expectedProjectedProgress := expectedQueuedProgress
	expectedProjectedProgress.ScopeFrozen = true
	expectedBatchProgress := expectedQueuedProgress
	expectedBatchProgress.DeduplicatedFindings = progress.Findings
	expectedBatchProgress.Findings = progress.Findings
	if resumed.Automation.Status != repoaudit.RepositoryReviewAutomationRunning || newRunID == "" ||
		!reflect.DeepEqual(resumed.Automation.RunIDs, expectedRunIDs) ||
		resumed.Automation.Usage != usage ||
		math.Abs(resumed.Automation.EstimatedCostUSD-input.EstimatedCostUSD) > 0.0000001 ||
		resumed.Automation.Progress != expectedProjectedProgress ||
		!resumed.Automation.StartedAt.Equal(startedAt) {
		t.Fatalf("resumed failed campaign=%#v", resumed.Automation)
	}
	select {
	case observed := <-batchStarted:
		if observed.runID != newRunID || observed.automation.Usage != usage ||
			observed.automation.CampaignID != input.CampaignID ||
			observed.automation.Progress != expectedBatchProgress ||
			!observed.automation.StartedAt.Equal(startedAt) ||
			!reflect.DeepEqual(observed.automation.RunIDs, expectedRunIDs) ||
			!reflect.DeepEqual(observed.automation.ScopeSelection, automation.ScopeSelection) ||
			!reflect.DeepEqual(observed.automation.ScopePlan, automation.ScopePlan) {
			t.Fatalf("resumed batch=%#v run_id=%q", observed.automation, observed.runID)
		}
	case <-time.After(time.Second):
		t.Fatal("resumed failed batch did not start")
	}

	release()
	completed := waitForRepositoryReviewAutomationStatus(
		t,
		store,
		automation.ID,
		repoaudit.RepositoryReviewAutomationCompleted,
	)
	if completed.Usage != usage ||
		math.Abs(completed.EstimatedCostUSD-input.EstimatedCostUSD) > 0.0000001 ||
		completed.Progress.CompletedBatches != progress.CompletedBatches+1 ||
		completed.Progress.ReviewedFiles != progress.ReviewedFiles ||
		completed.Progress.RemainingFiles != 0 ||
		completed.Progress.UnsupportedFiles != progress.UnsupportedFiles ||
		completed.Progress.RawFindings != 0 ||
		completed.Progress.DeduplicatedFindings != progress.Findings ||
		completed.Progress.Findings != progress.Findings ||
		!completed.StartedAt.Equal(startedAt) ||
		!reflect.DeepEqual(completed.RunIDs, expectedRunIDs) ||
		!reflect.DeepEqual(completed.ScopeSelection, automation.ScopeSelection) ||
		!reflect.DeepEqual(completed.ScopePlan, automation.ScopePlan) {
		t.Fatalf("completed resumed campaign=%#v", completed)
	}
}

func TestRepositoryReviewAutomationResumeRejectsMalformedCustomCommit(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	var resolveCalls atomic.Int32
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		resolveCalls.Add(1)
		return strings.Repeat("c", 40), nil
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	input := testRepositoryReviewAutomation()
	input.Status = repoaudit.RepositoryReviewAutomationPaused
	input.PauseReason = repoaudit.RepositoryReviewPauseManual
	input.PauseDetail = "paused between batches"
	input.ResolvedCommitSHA = strings.Repeat("d", 40)
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	response := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/resume",
		map[string]any{
			"expected_version": automation.Version,
			"commit_sha":       "not-a-full-commit",
		},
	)
	if response.Code != http.StatusBadRequest ||
		!strings.Contains(response.Body.String(), "invalid_repository_review_automation") {
		t.Fatalf("malformed resume status=%d body=%s", response.Code, response.Body.String())
	}
	current, found, err := store.GetAutomation(t.Context(), automation.ID)
	if err != nil || !found || current.Status != repoaudit.RepositoryReviewAutomationPaused ||
		current.ResolvedCommitSHA != input.ResolvedCommitSHA || current.Version != automation.Version ||
		resolveCalls.Load() != 0 {
		t.Fatalf("current=%#v found=%v err=%v resolver calls=%d", current, found, err, resolveCalls.Load())
	}
}

func TestRepositoryReviewAutomationPauseQueuedHandoff(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	input := testRepositoryReviewAutomation()
	input.Status = repoaudit.RepositoryReviewAutomationIdle
	input.Progress.Stage = "next batch queued"
	input.Progress.CompletedBatches = 1
	input.Progress.TotalBatches = 2
	input.Progress.RemainingFiles = 1
	input.ResolvedCommitSHA = strings.Repeat("e", 40)
	automation, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	response := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/pause",
		map[string]any{"expected_version": automation.Version},
	)
	if response.Code != http.StatusAccepted {
		t.Fatalf("pause queued handoff status=%d body=%s", response.Code, response.Body.String())
	}
	var paused struct {
		Automation repoaudit.RepositoryReviewAutomation `json:"automation"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &paused); err != nil {
		t.Fatal(err)
	}
	if paused.Automation.Status != repoaudit.RepositoryReviewAutomationPaused ||
		paused.Automation.PauseReason != repoaudit.RepositoryReviewPauseManual ||
		paused.Automation.Progress.Stage != "paused" ||
		paused.Automation.ResolvedCommitSHA != input.ResolvedCommitSHA ||
		paused.Automation.ActiveRunID != "" {
		t.Fatalf("paused queued handoff=%#v", paused.Automation)
	}
}

func TestRepositoryReviewAutomationPauseAcceptsStaleRunningVersion(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	input := testRepositoryReviewAutomation()
	input.Status = repoaudit.RepositoryReviewAutomationRunning
	input.ActiveRunID = "wr_stale_pause"
	input.RunIDs = []string{input.ActiveRunID}
	input.Progress.Stage = "Reviewing bounded file batch"
	input.Progress.TotalBatches = 1
	input.ResolvedCommitSHA = strings.Repeat("f", 40)
	created, err := store.CreateAutomation(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	current, err := store.UpdateAutomation(
		t.Context(),
		created.ID,
		created.Version,
		func(candidate *repoaudit.RepositoryReviewAutomation) error {
			candidate.Usage = repoaudit.RepositoryReviewTokenUsage{
				PromptTokens: 4, CompletionTokens: 1, TotalTokens: 5,
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	controller.mu.Lock()
	controller.active[current.ID] = &repositoryReviewActiveRun{
		runID: current.ActiveRunID,
		store: store,
	}
	controller.mu.Unlock()
	delayed := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/automations/"+current.ID+"/pause",
		map[string]any{
			"expected_version": created.Version,
			"run_id":           "wr_prior_campaign",
		},
	)
	if delayed.Code != http.StatusConflict {
		t.Fatalf("prior-run pause status=%d body=%s", delayed.Code, delayed.Body.String())
	}
	response := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/automations/"+current.ID+"/pause",
		map[string]any{
			"expected_version": created.Version,
			"run_id":           current.ActiveRunID,
		},
	)
	if response.Code != http.StatusAccepted {
		t.Fatalf("stale pause status=%d body=%s", response.Code, response.Body.String())
	}
	var paused struct {
		Automation repoaudit.RepositoryReviewAutomation `json:"automation"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &paused); err != nil {
		t.Fatal(err)
	}
	if paused.Automation.Status != repoaudit.RepositoryReviewAutomationStopping ||
		paused.Automation.RequestedPauseReason != repoaudit.RepositoryReviewPauseManual ||
		paused.Automation.ResolvedCommitSHA != input.ResolvedCommitSHA ||
		paused.Automation.Version <= current.Version {
		t.Fatalf("stale pause result=%#v current version=%d", paused.Automation, current.Version)
	}
	controller.mu.Lock()
	active := controller.active[current.ID]
	controller.mu.Unlock()
	if active == nil || active.pauseReason != repoaudit.RepositoryReviewPauseManual {
		t.Fatalf("active pause latch=%#v", active)
	}
}

func TestRepositoryReviewAutomationPauseLatchDoesNotTransferToNewRun(t *testing.T) {
	controller := newRepositoryReviewController(nil)
	controller.active["rra_pause_race"] = &repositoryReviewActiveRun{runID: "wr_new"}

	controller.latchManualPause("rra_pause_race", "wr_old")
	if active := controller.active["rra_pause_race"]; active.pauseReason != "" || active.pauseDetail != "" {
		t.Fatalf("old pause transferred to new run: %#v", active)
	}

	controller.latchManualPause("rra_pause_race", "wr_new")
	active := controller.active["rra_pause_race"]
	if active.pauseReason != repoaudit.RepositoryReviewPauseManual ||
		!strings.Contains(active.pauseDetail, "safe checkpoint") {
		t.Fatalf("matching pause was not latched: %#v", active)
	}
}

func TestRepositoryReviewAutomationFailsOnWorkflowCommitMismatch(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	rememberedCommit := strings.Repeat("a", 40)
	workflowCommit := strings.Repeat("b", 40)
	controller.resolveCommit = func(
		context.Context,
		*config.Config,
		repoaudit.RepositoryReviewAutomation,
		string,
	) (string, error) {
		return rememberedCommit, nil
	}
	controller.runBatch = func(
		_ context.Context,
		_ repoaudit.RepositoryReviewAutomation,
		runID string,
		_ workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		return &workflows.RunResult{
			RunID: runID, Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{"commit": workflowCommit, "remainingFiles": 0},
		}, nil
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation, err := store.CreateAutomation(t.Context(), testRepositoryReviewAutomation())
	if err != nil {
		t.Fatal(err)
	}
	response := repositoryReviewAutomationMutation(
		t,
		mux,
		http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/start",
		map[string]any{"expected_version": automation.Version},
	)
	if response.Code != http.StatusAccepted {
		t.Fatalf("start status=%d body=%s", response.Code, response.Body.String())
	}
	failed := waitForRepositoryReviewAutomationStatus(
		t,
		store,
		automation.ID,
		repoaudit.RepositoryReviewAutomationFailed,
	)
	if failed.ResolvedCommitSHA != rememberedCommit ||
		failed.Progress.CompletedBatches != 0 ||
		failed.PauseReason != repoaudit.RepositoryReviewPauseRunFailed ||
		!strings.Contains(failed.PauseDetail, "workflow resolved commit") ||
		!strings.Contains(failed.PauseDetail, rememberedCommit) ||
		!strings.Contains(failed.PauseDetail, workflowCommit) {
		t.Fatalf("commit mismatch failure=%#v", failed)
	}
}

func TestRepositoryReviewOrdinaryResumeRetainsLegacyAccountingSnapshot(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	seenPrice := make(chan repoaudit.RepositoryReviewModelPrice, 1)
	controller.runBatch = func(
		_ context.Context,
		automation repoaudit.RepositoryReviewAutomation,
		runID string,
		observe workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		seenPrice <- automation.ModelPrices["cheap"]
		if err := observe(workflows.AgentUsage{
			Model: "cheap", Reviewer: "cheap",
			PromptTokens: 10, CompletionTokens: 2, TotalTokens: 12,
		}); err != nil {
			return nil, err
		}
		return &workflows.RunResult{
			RunID: runID, Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{"remainingFiles": 0},
		}, nil
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.Status = repoaudit.RepositoryReviewAutomationPaused
	automation.PauseReason = repoaudit.RepositoryReviewPauseManual
	automation.PauseDetail = "legacy campaign paused"
	automation.ModelPrices = map[string]repoaudit.RepositoryReviewModelPrice{
		"cheap": {InputPricePer1M: 7, OutputPricePer1M: 11},
	}
	automation, err = store.CreateAutomation(t.Context(), automation)
	if err != nil {
		t.Fatal(err)
	}
	resumed := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/resume",
		map[string]any{"expected_version": automation.Version},
	)
	if resumed.Code != http.StatusAccepted {
		t.Fatalf("resume status=%d body=%s", resumed.Code, resumed.Body.String())
	}
	select {
	case price := <-seenPrice:
		if price.InputPricePer1M != 1 || price.OutputPricePer1M != 2 {
			t.Fatalf("ordinary resume did not refresh central pricing: %#v", price)
		}
	case <-time.After(time.Second):
		t.Fatal("resumed batch did not start")
	}
	completed := waitForRepositoryReviewAutomationStatus(
		t, store, automation.ID, repoaudit.RepositoryReviewAutomationCompleted,
	)
	if math.Abs(completed.EstimatedCostUSD-0.000014) > 0.0000001 {
		t.Fatalf("refreshed snapshot cost=%v", completed.EstimatedCostUSD)
	}
}

func TestRepositoryReviewGuardPauseRequiresExplicitResume(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	var calls atomic.Int32
	controller.runBatch = func(
		_ context.Context,
		_ repoaudit.RepositoryReviewAutomation,
		runID string,
		_ workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		calls.Add(1)
		return &workflows.RunResult{
			RunID: runID, Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{"remainingFiles": 0},
		}, nil
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.Status = repoaudit.RepositoryReviewAutomationPaused
	automation.PauseReason = repoaudit.RepositoryReviewPauseGuardExpression
	automation.PauseDetail = "task admission guard was false"
	automation.BudgetPolicy.GuardExpression = "account.limits.weekly.remaining_percent >= 50"
	automation, err = store.CreateAutomation(t.Context(), automation)
	if err != nil {
		t.Fatal(err)
	}

	if startErr := controller.Start(); startErr != nil {
		t.Fatal(startErr)
	}
	controller.reconcile()
	time.Sleep(30 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatalf("guard-paused review auto-resumed %d times", calls.Load())
	}
	current, _, err := store.GetAutomation(t.Context(), automation.ID)
	if err != nil || current.Status != repoaudit.RepositoryReviewAutomationPaused {
		t.Fatalf("guard pause changed without explicit resume: %#v err=%v", current, err)
	}
}

func TestRepositoryReviewRestartReconciliationPreservesManualStopIntent(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.Status = repoaudit.RepositoryReviewAutomationStopping
	automation.ActiveRunID = "wr_manual_stop"
	automation.RunIDs = []string{"wr_manual_stop"}
	automation.RequestedPauseReason = repoaudit.RepositoryReviewPauseManual
	automation.RequestedPauseDetail = "operator requested a safe stop"
	automation, err = store.CreateAutomation(t.Context(), automation)
	if err != nil {
		t.Fatal(err)
	}
	controller.leasedStore = store
	controller.leasedConfig = cfg
	controller.reconcile()
	updated, found, err := store.GetAutomation(t.Context(), automation.ID)
	if err != nil || !found {
		t.Fatalf("GetAutomation found=%v err=%v", found, err)
	}
	if updated.Status != repoaudit.RepositoryReviewAutomationPaused ||
		updated.PauseReason != repoaudit.RepositoryReviewPauseManual ||
		updated.PauseDetail != "operator requested a safe stop" ||
		updated.ActiveRunID != "" || len(updated.RunIDs) != 1 {
		t.Fatalf("reconciled manual stop=%#v", updated)
	}
}

func TestRepositoryReviewBudgetResetKeepsLifetimeComparisonWithoutResurrectingGuardCost(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.Status = repoaudit.RepositoryReviewAutomationPaused
	automation.PauseReason = repoaudit.RepositoryReviewPauseGuardExpression
	automation.PauseDetail = "guard is false"
	automation, err = store.CreateAutomation(t.Context(), automation)
	if err != nil {
		t.Fatal(err)
	}
	response := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/resume",
		map[string]any{"expected_version": automation.Version, "reset_budget": true},
	)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "unknown field") {
		t.Fatalf("legacy reset status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewAutomationRejectsCallerCampaignRecoveryAuthority(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	response := repositoryReviewAutomationMutation(
		t, mux, http.MethodPost, "/api/repository-reviews/automations",
		map[string]any{"campaign_recovery_pending": true},
	)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "unknown field") {
		t.Fatalf("caller recovery authority status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewAutomationManualPauseResumeAndRestart(t *testing.T) {
	handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	controller.runBatch = func(
		_ context.Context,
		_ repoaudit.RepositoryReviewAutomation,
		runID string,
		observe workflows.AgentUsageObserver,
	) (*workflows.RunResult, error) {
		if err := observe(workflows.AgentUsage{
			Model: "cheap", PromptTokens: 8, CompletionTokens: 2, TotalTokens: 10,
		}); err != nil {
			return nil, err
		}
		return &workflows.RunResult{
			RunID: runID, Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{"remainingFiles": 0, "reviewedFiles": 1},
		}, nil
	}
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	initial := testRepositoryReviewAutomation()
	initial.Status = repoaudit.RepositoryReviewAutomationPaused
	initial.PauseReason = repoaudit.RepositoryReviewPauseManual
	initial.PauseDetail = "Paused manually."
	automation, err := store.CreateAutomation(t.Context(), initial)
	if err != nil {
		t.Fatal(err)
	}

	resumed := repositoryReviewAutomationMutation(t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/resume",
		map[string]any{"expected_version": automation.Version})
	if resumed.Code != http.StatusAccepted {
		t.Fatalf("resume status=%d body=%s", resumed.Code, resumed.Body.String())
	}
	completed := waitForRepositoryReviewAutomationStatus(
		t, store, automation.ID, repoaudit.RepositoryReviewAutomationCompleted,
	)
	if completed.Usage.TotalTokens != 10 || completed.Progress.CompletedBatches != 1 {
		t.Fatalf("resumed completion=%#v", completed)
	}

	restarted := repositoryReviewAutomationMutation(t, mux, http.MethodPost,
		"/api/repository-reviews/automations/"+automation.ID+"/restart",
		map[string]any{"expected_version": completed.Version})
	if restarted.Code != http.StatusAccepted {
		t.Fatalf("restart status=%d body=%s", restarted.Code, restarted.Body.String())
	}
	recompleted := waitForRepositoryReviewAutomationStatus(
		t, store, automation.ID, repoaudit.RepositoryReviewAutomationCompleted,
	)
	if len(recompleted.RunIDs) != 2 || recompleted.Usage.TotalTokens != 10 ||
		recompleted.Progress.CompletedBatches != 1 {
		t.Fatalf("restarted completion=%#v", recompleted)
	}
}

func TestEvaluateRepositoryReviewQuotaAcrossAccountsAndWindows(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	controller := handler.repositoryReviewControllerInstance()
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	usedWeekly, usedDaily := 80, 5
	controller.probe = func(context.Context) (codexAccountLimitsResponse, error) {
		return codexAccountLimitsResponse{Accounts: []codexAccountLimitAccount{{
			ID: "api", Entries: []codexAccountLimitEntry{
				{Name: "Codex", Status: "available", Window: "weekly", UsedPercent: &usedWeekly},
				{Name: "Codex", Status: "available", Window: "daily", UsedPercent: &usedDaily},
			},
		}}}, nil
	}
	snapshots, known, err := controller.repositoryReviewGuardAccountLimits(
		t.Context(), cfg, testRepositoryReviewAutomation(),
	)
	if err != nil || !known || len(snapshots) != 2 {
		t.Fatalf("guard snapshots=%#v known=%v err=%v", snapshots, known, err)
	}
	allowed, err := repoaudit.EvaluateRepositoryReviewGuardExpression(
		"account.limits.weekly.remaining_percent >= 25",
		repoaudit.RepositoryReviewGuardEnvironment{
			AccountLimitsKnown: true, AccountLimitSnapshots: snapshots,
		},
	)
	if err != nil || allowed {
		t.Fatalf("weekly guard allowed=%v err=%v", allowed, err)
	}
}

func TestRepositoryReviewModelOptionsExposePriceAndBlockAgenticCLI(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "cheap"
	cfg.Agents.Defaults.AccountRef = "api"
	cfg.ModelAliases = []config.ModelAliasConfig{
		{Name: "cheap", Model: "openai/gpt-cheap"},
		{Name: "unsafe", Model: "codex-cli/codex"},
	}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "api", Provider: "openai", Model: "openai/gpt-cheap", Enabled: true,
		InputPricePerMTok: 0.2, OutputPricePerMTok: 0.8,
	}}
	options := repositoryReviewModelOptions(cfg)
	if len(options) != 2 || options[0].Alias != "cheap" || !options[0].Default ||
		!options[0].PriceKnown || options[0].InputPricePer1M != 0.2 ||
		options[1].Alias != "unsafe" || options[1].Available || options[1].BlockedReason == "" {
		t.Fatalf("options=%#v", options)
	}
}

func TestRepositoryReviewModelOptionsSupportCredentialOnlyAccountRouter(t *testing.T) {
	withPicoclawAuthHome(t)
	for _, credentialID := range []string{"openai:work", "openai:backup", "openai:overflow"} {
		setOpenAIAuthCredential(
			t,
			credentialID,
			"access-token-"+credentialID,
			"refresh-token-"+credentialID,
			"account-"+credentialID,
			credentialID+"@example.test",
		)
	}
	if err := auth.SetCredential("openai:expired", &auth.AuthCredential{
		AccessToken: "expired-access-token",
		Provider:    "openai",
		AuthMethod:  "oauth",
		ExpiresAt:   time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := auth.SetCredential("github-copilot:gh-copilot", &auth.AuthCredential{
		AccessToken: "ghp_invalid-copilot-token",
		Provider:    "github-copilot",
		AuthMethod:  "token",
	}); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "review"
	cfg.Agents.Defaults.AccountRef = "review-router"
	credentialRefs := []string{
		"credential:github-copilot:gh-copilot",
		"credential:openai:work",
		"credential:openai:backup",
		"credential:openai:overflow",
	}
	cfg.ModelAliases = []config.ModelAliasConfig{
		{
			Name: "review", Model: "gpt-review",
			AccountOverrides: map[string]string{
				"credential:github-copilot:gh-copilot": "gpt-review-copilot",
			},
		},
		{
			Name: "partial", Model: "gpt-partial",
			DisabledAccounts: []string{"credential:openai:backup"},
		},
		{
			Name: "disabled", Model: "gpt-disabled",
			DisabledAccounts: credentialRefs,
		},
		{Name: "unsafe", Model: "codex-cli/codex"},
	}
	cfg.ModelList = nil
	cfg.AccountRouters = []config.AccountRouterConfig{{
		Name: "review-router", Enabled: true, Entry: "copilot",
		Blocks: []config.AccountRouterBlock{
			{
				ID: "copilot", Type: config.AccountRouterBlockTypeAccount,
				Account: "credential:github-copilot:gh-copilot", Fallback: "openai",
			},
			{
				ID: "openai", Type: config.AccountRouterBlockTypeLoadBalance,
				Accounts: []string{
					"credential:openai:work",
					"credential:openai:backup",
					"credential:openai:overflow",
				},
			},
		},
	}}

	accounts := repositoryReviewAccountOptions(cfg, codexAccountLimitsResponse{Accounts: []codexAccountLimitAccount{
		{
			ID: "github-copilot:gh-copilot", Provider: "github-copilot",
			CredentialStatus: "invalid", LimitsStatus: "unavailable",
		},
		{
			ID: "openai:work", Provider: "openai",
			CredentialStatus: "available", LimitsStatus: "available",
		},
		{
			ID: "openai:backup", Provider: "openai",
			CredentialStatus: "available", LimitsStatus: "error", LimitsError: "telemetry_offline",
		},
		{
			ID: "openai:overflow", Provider: "openai",
			CredentialStatus: "available", LimitsStatus: "available",
		},
		{
			ID: "openai:expired", Provider: "openai",
			CredentialStatus: "available", LimitsStatus: "error", LimitsError: "token_expired",
		},
	}})
	availableRefs := make([]string, 0, len(accounts))
	for _, account := range accounts {
		if account.Available {
			availableRefs = append(availableRefs, account.ID)
		}
	}
	options := repositoryReviewModelOptions(cfg, availableRefs...)
	byAlias := make(map[string]repositoryReviewModelOption, len(options))
	for _, option := range options {
		byAlias[option.Alias] = option
	}
	if review := byAlias["review"]; !review.Available || !review.Default || review.BlockedReason != "" {
		t.Fatalf("credential-router review option=%#v", review)
	}
	if partial := byAlias["partial"]; !partial.Available || partial.BlockedReason != "" {
		t.Fatalf("directly selectable partial option=%#v", partial)
	}
	for _, alias := range []string{"disabled", "unsafe"} {
		if option := byAlias[alias]; option.Available || option.BlockedReason == "" {
			t.Fatalf("blocked credential-router option %q=%#v", alias, option)
		}
	}

	if len(accounts) != 6 || accounts[0].ID != "review-router" || !accounts[0].Default ||
		!accounts[0].Available {
		t.Fatalf("credential-router accounts=%#v", accounts)
	}
	accountByID := make(map[string]repositoryReviewAccountOption, len(accounts))
	for _, account := range accounts {
		accountByID[account.ID] = account
		for _, alias := range account.Models {
			if account.Available && !byAlias[alias].Available {
				t.Fatalf("account %q exposes globally unavailable alias %q", account.ID, alias)
			}
		}
	}
	if accountByID["credential:github-copilot:gh-copilot"].Available ||
		accountByID["credential:openai:expired"].Available ||
		!accountByID["credential:openai:work"].Available ||
		!accountByID["credential:openai:backup"].Available ||
		accountByID["credential:openai:backup"].Status != "error" {
		t.Fatalf("credential availability=%#v", accountByID)
	}
	if !reflect.DeepEqual(accountByID["review-router"].Models, []string{"review"}) ||
		!reflect.DeepEqual(
			accountByID["credential:openai:work"].Models,
			[]string{"partial", "review"},
		) ||
		!reflect.DeepEqual(
			accountByID["credential:openai:backup"].Models,
			[]string{"review"},
		) {
		t.Fatalf("credential model projection=%#v", accountByID)
	}
}

func TestRepositoryReviewModelOptionsRejectPartiallyPricedAccountRoute(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.AccountRef = "review-router"
	cfg.ModelAliases = []config.ModelAliasConfig{{
		Name: "review", Model: "openai/review",
	}}
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "priced", Provider: "openai", Model: "openai/review", Enabled: true,
			InputPricePerMTok: 1, OutputPricePerMTok: 4,
		},
		{ModelName: "unpriced", Provider: "openai", Model: "openai/review", Enabled: true},
	}
	cfg.AccountRouters = []config.AccountRouterConfig{{
		Name: "review-router", Enabled: true, Entry: "accounts",
		Blocks: []config.AccountRouterBlock{{
			ID: "accounts", Type: config.AccountRouterBlockTypeLoadBalance,
			Accounts: []string{"priced", "unpriced"},
		}},
	}}

	options := repositoryReviewModelOptions(cfg)
	if len(options) != 1 || !options[0].Available || options[0].PriceKnown {
		t.Fatalf("partially priced option=%#v", options)
	}
	automation := testRepositoryReviewAutomation()
	automation.ReviewerModels = []string{"review"}
	automation.BudgetPolicy.GuardExpression = "spend.total.usd < 10"
	if err := repositoryReviewRefreshAccountingSnapshot(cfg, &automation); err == nil {
		t.Fatal("partially priced route admitted a USD budget")
	}
}

func TestRepositoryReviewPricingIgnoresUnreachableAccountRouterBlocks(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.AccountRef = "review-router"
	cfg.ModelAliases = []config.ModelAliasConfig{{Name: "review", Model: "openai/review"}}
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "priced", Provider: "openai", Model: "openai/review", Enabled: true,
			InputPricePerMTok: 1, OutputPricePerMTok: 4,
		},
		{ModelName: "orphan-unpriced", Provider: "openai", Model: "openai/review", Enabled: true},
	}
	cfg.AccountRouters = []config.AccountRouterConfig{{
		Name: "review-router", Enabled: true, Entry: "entry",
		Blocks: []config.AccountRouterBlock{
			{ID: "entry", Type: config.AccountRouterBlockTypeAccount, Account: "priced"},
			{ID: "orphan", Type: config.AccountRouterBlockTypeAccount, Account: "orphan-unpriced"},
		},
	}}

	if refs := repositoryReviewRuntimeAccountRefs(cfg); !reflect.DeepEqual(refs, []string{"priced"}) {
		t.Fatalf("reachable account refs=%#v", refs)
	}
	options := repositoryReviewModelOptions(cfg)
	if len(options) != 1 || !options[0].PriceKnown || options[0].InputPricePer1M != 1 {
		t.Fatalf("orphan block affected pricing=%#v", options)
	}
}

func TestRepositoryReviewCentralPricingHelperBoundaries(t *testing.T) {
	t.Run("configuration and snapshot errors", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "invalid.json")
		if err := os.WriteFile(configPath, []byte("{"), 0o600); err != nil {
			t.Fatal(err)
		}
		handler := &Handler{configPath: configPath}
		automation := testRepositoryReviewAutomation()
		if err := handler.refreshRepositoryReviewAccountingSnapshot(&automation); err == nil {
			t.Fatal("invalid central configuration produced an accounting snapshot")
		}
		if err := handler.validateRepositoryReviewProfileSelection(
			"", "cheap",
			repoaudit.RepositoryReviewBudgetPolicy{GuardExpression: "spend.total.usd < 1"},
		); err == nil {
			t.Fatal("invalid central configuration admitted a profile cost budget")
		}
		if err := repositoryReviewRefreshAccountingSnapshot(nil, nil); !errors.Is(
			err,
			repoaudit.ErrInvalidAutomation,
		) {
			t.Fatalf("nil automation pricing error=%v", err)
		}
		unknown := testRepositoryReviewAutomation()
		unknown.ReviewerModels = []string{"missing"}
		unknown.ModelPrices = map[string]repoaudit.RepositoryReviewModelPrice{
			"missing": {InputPricePer1M: 99, OutputPricePer1M: 99},
		}
		if err := repositoryReviewRefreshAccountingSnapshot(nil, &unknown); err != nil ||
			len(unknown.ModelPrices) != 0 {
			t.Fatalf("unknown central pricing snapshot=%#v error=%v", unknown.ModelPrices, err)
		}
	})

	t.Run("reachable router graph", func(t *testing.T) {
		if refs := repositoryReviewReachableAccountRouterRefs(nil); refs != nil {
			t.Fatalf("nil router refs=%#v", refs)
		}
		router := &config.AccountRouterConfig{
			Entry: " branch ",
			Blocks: []config.AccountRouterBlock{
				{ID: "", Type: config.AccountRouterBlockTypeAccount, Account: "ignored"},
				{
					ID: "branch", Type: config.AccountRouterBlockTypeBranch,
					Then: "direct", Else: "missing", Fallback: "branch",
				},
				{
					ID: "direct", Type: config.AccountRouterBlockTypeAccount,
					Account: " account-a ", Fallback: "pool",
				},
				{
					ID: "pool", Type: config.AccountRouterBlockTypeLoadBalance,
					Accounts: []string{"", "account-a", "account-b"},
				},
			},
		}
		if refs := repositoryReviewReachableAccountRouterRefs(router); !reflect.DeepEqual(
			refs,
			[]string{"account-a", "account-b"},
		) {
			t.Fatalf("reachable router refs=%#v", refs)
		}
	})

	t.Run("equivalent alias recursion", func(t *testing.T) {
		if price, found := repositoryReviewEquivalentAliasPrice(nil, "root", nil); price != nil || found {
			t.Fatalf("nil equivalent pricing=(%#v,%v)", price, found)
		}
		cfg := config.DefaultConfig()
		cfg.ModelAliases = []config.ModelAliasConfig{
			{Name: "root", Model: "openai/root"},
			{Name: "middle", Model: "openai/middle"},
			{Name: "leaf", Model: "openai/leaf"},
		}
		cfg.ModelList = []*config.ModelConfig{
			nil,
			{ModelName: "disabled", Provider: "openai", Model: "openai/disabled"},
			{
				ModelName: "account-router", Enabled: true,
				Router: &config.AccountRouterConfig{Name: "account-router"},
			},
			{
				ModelName: "model-router", Enabled: true,
				ModelRouter: &config.ModelRouterConfig{Name: "model-router"},
			},
			{
				ModelName: "subscription-middle", Provider: "openai", Model: "openai/root",
				Enabled: true, Subscription: true, SubscriptionEquivalentModel: "middle",
			},
			{
				ModelName: "subscription-leaf", Provider: "openai", Model: "openai/middle",
				Enabled: true, Subscription: true, SubscriptionEquivalentModel: "leaf",
			},
			{
				ModelName: "priced", Provider: "openai", Model: "openai/leaf", Enabled: true,
				InputPricePerMTok: 1.5, OutputPricePerMTok: 6,
			},
		}
		price, found := repositoryReviewEquivalentAliasPrice(
			cfg,
			"root",
			make(map[string]bool),
		)
		if !found || price.InputPricePerMTok != 1.5 || price.OutputPricePerMTok != 6 {
			t.Fatalf("recursive equivalent pricing=(%#v,%v)", price, found)
		}
		missingPrice, missingFound := repositoryReviewEquivalentAliasPrice(
			cfg,
			"missing",
			make(map[string]bool),
		)
		if missingPrice == nil || missingFound {
			t.Fatalf("missing equivalent alias pricing=(%#v,%v)", missingPrice, missingFound)
		}
		if price, found := repositoryReviewEquivalentAliasPrice(
			cfg,
			"root",
			map[string]bool{"root": true},
		); price != nil || found {
			t.Fatalf("recursive guard pricing=(%#v,%v)", price, found)
		}
	})
}

func TestRepositoryReviewModelOptionsInheritSubscriptionPriceAndRejectUnsafeOverride(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.AccountRef = "review-router"
	cfg.ModelAliases = []config.ModelAliasConfig{
		{Name: "subscription-review", Model: "openai/subscription", AccountOverrides: map[string]string{
			"unsafe": "codex-cli/codex",
		}},
		{Name: "metered-review", Model: "openai/metered"},
	}
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "subscription", Provider: "openai", Model: "openai/subscription", Enabled: true,
			Subscription: true, SubscriptionEquivalentModel: "metered-review",
		},
		{
			ModelName: "metered", Provider: "openai", Model: "openai/metered", Enabled: true,
			InputPricePerMTok: 1.25, OutputPricePerMTok: 5,
		},
		{ModelName: "unsafe", Provider: "codex-cli", Model: "codex-cli/codex", Enabled: true},
	}
	cfg.AccountRouters = []config.AccountRouterConfig{{
		Name: "review-router", Enabled: true, Entry: "accounts",
		Blocks: []config.AccountRouterBlock{{
			ID: "accounts", Type: config.AccountRouterBlockTypeLoadBalance,
			Accounts: []string{"subscription", "metered", "unsafe"},
		}},
	}}
	options := repositoryReviewModelOptions(cfg)
	byAlias := make(map[string]repositoryReviewModelOption, len(options))
	for _, option := range options {
		byAlias[option.Alias] = option
	}
	subscription := byAlias["subscription-review"]
	if subscription.Available || subscription.BlockedReason == "" || subscription.PriceKnown {
		t.Fatalf("subscription option=%#v", subscription)
	}
	cfg.ModelAliases[0].AccountOverrides = nil
	cfg.AccountRouters[0].Blocks[0].Accounts = []string{"subscription", "metered"}
	options = repositoryReviewModelOptions(cfg)
	byAlias = make(map[string]repositoryReviewModelOption, len(options))
	for _, option := range options {
		byAlias[option.Alias] = option
	}
	subscription = byAlias["subscription-review"]
	if !subscription.Available || !subscription.PriceKnown ||
		subscription.InputPricePer1M != 1.25 || subscription.OutputPricePer1M != 5 ||
		!subscription.Subscription || subscription.EquivalentModel != "metered-review" {
		t.Fatalf("safe subscription option=%#v", subscription)
	}
}

func TestRepositoryReviewWorkflowProjectionUsesQualifiedStepIDs(t *testing.T) {
	run := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"find_bugs/plan":   {ID: "plan", Status: workflows.RunStatusSucceeded},
		"find_bugs/review": {ID: "review", Status: workflows.RunStatusRunning},
	}}
	if got := repositoryReviewWorkflowStage(run); got != "Reviewing bounded file batch" {
		t.Fatalf("stage=%q", got)
	}
	if step := repositoryReviewRunStep(run, "review"); step.ID != "review" {
		t.Fatalf("qualified step=%#v", step)
	}
}

func TestRepositoryReviewCheckpointRequiresDurableRecordOrVerifiedNoop(t *testing.T) {
	result := &workflows.RunResult{
		Status: workflows.RunStatusSucceeded, Outputs: map[string]any{"remainingFiles": 0},
	}
	recordFailure := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"find_bugs/plan":   {Status: workflows.RunStatusSucceeded, Outputs: map[string]any{"pendingCount": 1}},
		"find_bugs/record": {Status: workflows.RunStatusSucceeded, Error: "disk write failed"},
		"find_bugs/result": {Status: workflows.RunStatusSucceeded},
	}}
	if repositoryReviewRunCheckpointed(recordFailure, result) {
		t.Fatal("continued record failure counted as a durable checkpoint")
	}
	recordSuccess := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"find_bugs/record": {
			Status: workflows.RunStatusSucceeded,
			Outputs: map[string]any{"run": map[string]any{
				"id": "wr_checkpoint", "remaining_files": 0,
			}},
		},
	}}
	if !repositoryReviewRunCheckpointed(recordSuccess, result) {
		t.Fatal("durable record was not counted as a checkpoint")
	}
	for name, value := range map[string]any{
		"missing":        nil,
		"malformed":      "0",
		"negative":       -1,
		"nonintegral":    0.5,
		"wrong alias":    map[string]any{"remainingFiles": 0},
		"non-map record": true,
	} {
		recorded := map[string]any{"remaining_files": value}
		if mapped, ok := value.(map[string]any); ok {
			recorded = mapped
		}
		var durable any = recorded
		if name == "non-map record" {
			durable = value
		}
		if name == "missing" {
			recorded = map[string]any{"id": "wr_checkpoint"}
			durable = recorded
		}
		run := &workflows.Run{Steps: map[string]workflows.StepExecution{
			"find_bugs/record": {
				Status:  workflows.RunStatusSucceeded,
				Outputs: map[string]any{"run": durable},
			},
		}}
		if repositoryReviewRunCheckpointed(run, result) {
			t.Fatalf("%s durable count was counted as a checkpoint", name)
		}
	}
	noop := &workflows.Run{Steps: map[string]workflows.StepExecution{
		"find_bugs/plan": {Status: workflows.RunStatusSucceeded, Outputs: map[string]any{"pendingCount": 0}},
		"find_bugs/result": {
			Status:  workflows.RunStatusSucceeded,
			Outputs: map[string]any{"run": map[string]any{"remaining_files": 0}},
		},
	}}
	if !repositoryReviewRunCheckpointed(noop, result) {
		t.Fatal("verified no-op result was not counted as a checkpoint")
	}
	noopWith := func(pending, remaining map[string]any) bool {
		persistedRun := make(map[string]any)
		if value, valid := repositoryReviewOutputNonnegativeInt(
			remaining, "remainingFiles", "remaining_files",
		); valid {
			persistedRun["remaining_files"] = value
		} else if value, exists := remaining["remainingFiles"]; exists {
			persistedRun["remaining_files"] = value
		} else if value, exists := remaining["remaining_files"]; exists {
			persistedRun["remaining_files"] = value
		}
		return repositoryReviewRunCheckpointed(
			&workflows.Run{Steps: map[string]workflows.StepExecution{
				"find_bugs/plan": {
					Status: workflows.RunStatusSucceeded, Outputs: pending,
				},
				"find_bugs/result": {
					Status:  workflows.RunStatusSucceeded,
					Outputs: map[string]any{"run": persistedRun},
				},
			}},
			&workflows.RunResult{Status: workflows.RunStatusSucceeded, Outputs: remaining},
		)
	}
	for name, values := range map[string]struct {
		pending   map[string]any
		remaining map[string]any
	}{
		"missing pending": {
			pending: map[string]any{}, remaining: map[string]any{"remainingFiles": 0},
		},
		"malformed pending": {
			pending: map[string]any{"pendingCount": "0"}, remaining: map[string]any{"remainingFiles": 0},
		},
		"negative pending": {
			pending: map[string]any{"pendingCount": -1}, remaining: map[string]any{"remainingFiles": 0},
		},
		"missing remaining": {
			pending: map[string]any{"pendingCount": 0}, remaining: map[string]any{},
		},
		"malformed remaining": {
			pending: map[string]any{"pendingCount": 0}, remaining: map[string]any{"remainingFiles": "0"},
		},
		"negative remaining": {
			pending: map[string]any{"pendingCount": 0}, remaining: map[string]any{"remainingFiles": -1},
		},
		"nonintegral remaining": {
			pending: map[string]any{"pendingCount": 0}, remaining: map[string]any{"remainingFiles": 0.5},
		},
		"positive remaining": {
			pending: map[string]any{"pendingCount": 0}, remaining: map[string]any{"remainingFiles": 1},
		},
	} {
		if noopWith(values.pending, values.remaining) {
			t.Fatalf("%s was counted as an explicit no-op checkpoint", name)
		}
	}
	if !noopWith(
		map[string]any{"pendingCount": "invalid", "pending_count": float64(0)},
		map[string]any{"remainingFiles": false, "remaining_files": float64(0)},
	) {
		t.Fatal("valid snake-case zero aliases were not counted as an explicit no-op checkpoint")
	}
	for _, contradiction := range []struct {
		durable int
		project int
	}{{durable: 5, project: 0}, {durable: 0, project: 5}} {
		contradictory := &workflows.Run{Steps: map[string]workflows.StepExecution{
			"find_bugs/record": {
				Status: workflows.RunStatusSucceeded,
				Outputs: map[string]any{"run": map[string]any{
					"remaining_files": contradiction.durable,
				}},
			},
		}}
		if repositoryReviewRunCheckpointed(
			contradictory,
			&workflows.RunResult{
				Status:  workflows.RunStatusSucceeded,
				Outputs: map[string]any{"remainingFiles": contradiction.project},
			},
		) {
			t.Fatalf("contradictory durable=%d projected=%d checkpointed", contradiction.durable, contradiction.project)
		}
	}
	for _, contradiction := range []struct {
		durable int
		project int
	}{{durable: 5, project: 0}, {durable: 0, project: 5}} {
		contradictoryNoop := &workflows.Run{Steps: map[string]workflows.StepExecution{
			"find_bugs/plan": {
				Status:  workflows.RunStatusSucceeded,
				Outputs: map[string]any{"pendingCount": 0},
			},
			"find_bugs/result": {
				Status: workflows.RunStatusSucceeded,
				Outputs: map[string]any{"run": map[string]any{
					"remaining_files": contradiction.durable,
				}},
			},
		}}
		if repositoryReviewRunCheckpointed(
			contradictoryNoop,
			&workflows.RunResult{
				Status:  workflows.RunStatusSucceeded,
				Outputs: map[string]any{"remainingFiles": contradiction.project},
			},
		) {
			t.Fatalf(
				"contradictory no-op durable=%d projected=%d checkpointed",
				contradiction.durable, contradiction.project,
			)
		}
	}
	for _, contradiction := range []struct {
		record int
		result int
	}{{record: 5, result: 0}, {record: 0, result: 5}} {
		contradictoryPersisted := &workflows.Run{Steps: map[string]workflows.StepExecution{
			"find_bugs/record": {
				Status: workflows.RunStatusSucceeded,
				Outputs: map[string]any{"run": map[string]any{
					"remaining_files": contradiction.record,
				}},
			},
			"find_bugs/result": {
				Status: workflows.RunStatusSucceeded,
				Outputs: map[string]any{"run": map[string]any{
					"remaining_files": contradiction.result,
				}},
			},
		}}
		if repositoryReviewRunCheckpointed(
			contradictoryPersisted,
			&workflows.RunResult{
				Status:  workflows.RunStatusSucceeded,
				Outputs: map[string]any{"remainingFiles": contradiction.record},
			},
		) {
			t.Fatalf(
				"contradictory persisted record=%d result=%d checkpointed",
				contradiction.record, contradiction.result,
			)
		}
	}
	if noopWith(
		map[string]any{"pendingCount": 0, "pending_count": 1},
		map[string]any{"remainingFiles": 0},
	) {
		t.Fatal("contradictory pending aliases were counted as a no-op checkpoint")
	}
}

func TestRepositoryReviewAutomationControllerLeaseIsWorkspaceSingleton(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	first := handler.repositoryReviewControllerInstance()
	if err := first.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(first.Stop)
	secondHandler := NewHandler(handler.configPath)
	second := secondHandler.repositoryReviewControllerInstance()
	if err := second.Start(); !errors.Is(err, repoaudit.ErrAutomationControllerLocked) {
		t.Fatalf("second controller Start() error=%v", err)
	}
}

func TestRepositoryReviewAutomationStopCancelsBlockedQuotaAdmission(t *testing.T) {
	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	controller := handler.repositoryReviewControllerInstance()
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.Status = repoaudit.RepositoryReviewAutomationRunning
	automation.ActiveRunID = "wr_guard_probe"
	automation.RunIDs = []string{"wr_guard_probe"}
	automation.BudgetPolicy.GuardExpression = "account.limits.known"
	automation, err = store.CreateAutomation(t.Context(), automation)
	if err != nil {
		t.Fatal(err)
	}
	probeStarted := make(chan struct{})
	controller.probe = func(ctx context.Context) (codexAccountLimitsResponse, error) {
		close(probeStarted)
		<-ctx.Done()
		return codexAccountLimitsResponse{}, ctx.Err()
	}
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	controller.mu.Lock()
	controller.active[automation.ID] = &repositoryReviewActiveRun{
		runID: automation.ActiveRunID, store: store, config: cfg,
		reservations: make(map[int]repositoryReviewTaskReservation),
	}
	controller.mu.Unlock()
	admissionDone := make(chan error, 1)
	go func() {
		admissionDone <- controller.observeRepositoryReviewTask(
			automation.ID, automation.ActiveRunID, workflows.ManagedChildActivity{
				Phase: workflows.ManagedChildStarted, Index: 1,
			},
		)
	}()
	select {
	case <-probeStarted:
	case <-time.After(time.Second):
		t.Fatal("quota probe did not start")
	}
	stoppedAt := time.Now()
	controller.Stop()
	if time.Since(stoppedAt) > time.Second {
		t.Fatalf("controller Stop took %s", time.Since(stoppedAt))
	}
	select {
	case admissionErr := <-admissionDone:
		if !errors.Is(admissionErr, errRepositoryReviewSafeStop) {
			t.Fatalf("task admission error=%v", admissionErr)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked admission did not exit")
	}
	current, _, getErr := store.GetAutomation(t.Context(), automation.ID)
	if getErr != nil || current.Status != repoaudit.RepositoryReviewAutomationStopping ||
		current.RequestedPauseReason != repoaudit.RepositoryReviewPauseGuardExpression {
		t.Fatalf("post-stop automation=%#v err=%v", current, getErr)
	}
}

func newRepositoryReviewAutomationTestHandler(t *testing.T) (*Handler, *http.ServeMux, string) {
	t.Helper()
	workspace := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	cfg.Agents.Defaults.ModelName = "cheap"
	cfg.Agents.Defaults.AccountRef = "api"
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "api", Provider: "openai", Model: "openai/test", Enabled: true,
		InputPricePerMTok: 1, OutputPricePerMTok: 2,
	}}
	cfg.ModelAliases = []config.ModelAliasConfig{
		{Name: "cheap", Model: "gpt-cheap"},
		{Name: "quality", Model: "gpt-quality"},
	}
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(configPath)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	handler.repositoryReviewControllerInstance().resolveCommit = func(
		_ context.Context,
		_ *config.Config,
		_ repoaudit.RepositoryReviewAutomation,
		revision string,
	) (string, error) {
		if repositoryReviewValidCommitSelection(revision) {
			return strings.ToLower(strings.TrimSpace(revision)), nil
		}
		return strings.Repeat("a", 40), nil
	}
	return handler, mux, workspace
}

func testRepositoryReviewAutomation() repoaudit.RepositoryReviewAutomation {
	return repoaudit.RepositoryReviewAutomation{
		Name: "Test review", Repository: "https://github.com/acme/core.git",
		Ref: "main", Target: "all", ReviewFocus: "Find correctness bugs.",
		ReviewerModels: []string{"cheap"}, AutoContinue: true,
		MaxFilesPerRun: 4, MaxContentBytes: 65536, MaxParallelChildren: 1,
		AssignmentTimeoutSeconds: 3_600,
		EstimatedOutputTokens:    900,
		BudgetPolicy:             repoaudit.RepositoryReviewBudgetPolicy{},
		Status:                   repoaudit.RepositoryReviewAutomationIdle,
	}
}

func repositoryReviewAutomationMutation(
	t *testing.T,
	mux *http.ServeMux,
	method string,
	path string,
	body map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(encoded))
	setRepositoryReviewMutationHeaders(request)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func automationConfigBody(automation repoaudit.RepositoryReviewAutomation) map[string]any {
	autoContinue := automation.AutoContinue
	return map[string]any{
		"name": automation.Name, "repository": automation.Repository, "ref": automation.Ref,
		"target": automation.Target, "review_focus": automation.ReviewFocus,
		"scope_policy":    automation.ScopePolicy,
		"reviewer_models": automation.ReviewerModels, "compare_models": automation.CompareModels,
		"force":         automation.Force,
		"auto_continue": autoContinue, "max_files_per_run": automation.MaxFilesPerRun,
		"max_content_bytes":          automation.MaxContentBytes,
		"max_parallel_children":      automation.MaxParallelChildren,
		"assignment_timeout_seconds": automation.AssignmentTimeoutSeconds,
		"budget":                     automation.BudgetPolicy,
	}
}

func waitForRepositoryReviewAutomationStatus(
	t *testing.T,
	store repoaudit.Store,
	id string,
	status repoaudit.RepositoryReviewAutomationStatus,
) repoaudit.RepositoryReviewAutomation {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		automation, found, err := store.GetAutomation(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		if found && automation.Status == status {
			return automation
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("automation %s did not reach %s", id, status)
	return repoaudit.RepositoryReviewAutomation{}
}

func ptrInt(value int) *int { return &value }
