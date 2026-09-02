package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/repoaudit"
	"github.com/sipeed/picoclaw/pkg/workflows"
)

func TestRepositoryReviewAggregateFindingDetailProjectsOccurrencesAndDuplicates(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewGenerationFindings(t, workspace, 4)
	store := repoaudit.NewStore(workspace)
	var first, second repoaudit.RepositoryFinding
	for index, occurrence := range state.Findings {
		var pending repoaudit.RepositoryMappingJob
		for _, job := range state.MappingJobs {
			if job.ReviewFindingID == occurrence.ID {
				pending = job
				break
			}
		}
		claimedState, job, _, claimed, err := store.ClaimMappingJob(
			state.Repository, pending.ID, repoaudit.RepositoryMappingModelSnapshot{},
		)
		if err != nil || !claimed {
			t.Fatalf("claim mapping %d: claimed=%v err=%v", index, claimed, err)
		}
		state = claimedState
		completion := repoaudit.RepositoryMappingCompletion{
			JobID: job.ID, DefaultBranchVerified: true,
		}
		switch index {
		case 0, 1:
			completion.CreateMatchState = repoaudit.RepositoryMatchNew
		case 2:
			completion.RepositoryFindingID = first.ID
		case 3:
			completion.CreateMatchState = repoaudit.RepositoryMatchProvisional
			completion.PossibleDuplicates = []repoaudit.RepositoryFindingPossibleDuplicate{
				{CandidateID: first.ID, Relation: "uncertain", Confidence: .7},
				{CandidateID: second.ID, Relation: "related", Confidence: .6},
			}
		}
		var aggregate repoaudit.RepositoryFinding
		state, aggregate, err = store.CompleteMappingJob(state.Repository, completion)
		if err != nil {
			t.Fatalf("complete mapping %d: %v", index, err)
		}
		if index == 0 {
			first = aggregate
		} else if index == 1 {
			second = aggregate
		}
	}
	laterFile := repoaudit.FileRef{
		Path: "pkg/later.go", BlobSHA: strings.Repeat("f", 40), SizeBytes: 100,
		Category: "code", Mode: "100644",
	}
	plan, err := store.Plan(
		t.Context(), state.Repository, strings.Repeat("9", 40), "inventory-later",
		[]repoaudit.FileRef{laterFile}, true,
	)
	if err != nil {
		t.Fatal(err)
	}
	laterResult, err := store.Record(t.Context(), repoaudit.RecordRequest{
		Plan: plan, RunID: "aggregate-later",
		Observations: []repoaudit.Observation{{
			Model: "review-model", ScopeFiles: []repoaudit.FileRef{laterFile},
			Findings: []repoaudit.FindingCandidate{{
				Severity: "high", Title: "Later occurrence", File: laterFile.Path,
				Evidence: "The immutable source shows the failure.", Impact: "Data is lost.",
				Validation: repoaudit.Validation{Status: "confirmed", Summary: "Confirmed."},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	laterID := laterResult.AcceptedFindingIDs[0]
	laterJob := laterResult.State.MappingJobs[len(laterResult.State.MappingJobs)-1]
	_, claimedLater, _, claimed, err := store.ClaimMappingJob(
		state.Repository, laterJob.ID, repoaudit.RepositoryMappingModelSnapshot{},
	)
	if err != nil || !claimed {
		t.Fatalf("claim later occurrence: claimed=%v err=%v", claimed, err)
	}
	state, _, err = store.CompleteMappingJob(state.Repository, repoaudit.RepositoryMappingCompletion{
		JobID: claimedLater.ID, RepositoryFindingID: first.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if laterID == "" {
		t.Fatal("later occurrence was not recorded")
	}
	automation := seedRepositoryReviewDetailAutomation(
		t, handler, state.Repository, state.Runs[0].ID,
	)
	get := func(id string) *httptest.ResponseRecorder {
		t.Helper()
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(
			http.MethodGet,
			"/api/repository-reviews/automations/"+automation.ID+"/repository-findings/"+id,
			nil,
		))
		return response
	}

	joined := get(first.ID)
	if joined.Code != http.StatusOK ||
		!strings.Contains(joined.Body.String(), `"can_generate":true`) ||
		strings.Count(joined.Body.String(), `"repository_finding_id":"`+first.ID+`"`) < 2 {
		t.Fatalf("joined aggregate detail=%d %s", joined.Code, joined.Body.String())
	}
	provisional := state.RepositoryFindings[len(state.RepositoryFindings)-1]
	preview := get(provisional.ID)
	if preview.Code != http.StatusOK ||
		!strings.Contains(preview.Body.String(), `"can_generate":false`) ||
		!strings.Contains(preview.Body.String(), first.ID) ||
		!strings.Contains(preview.Body.String(), second.ID) {
		t.Fatalf("provisional aggregate detail=%d %s", preview.Code, preview.Body.String())
	}
	missing := get("rrf_missing")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing aggregate detail=%d %s", missing.Code, missing.Body.String())
	}
	missingAutomation := httptest.NewRecorder()
	mux.ServeHTTP(missingAutomation, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations/rra_missing/findings/rrf_missing",
		nil,
	))
	if missingAutomation.Code != http.StatusNotFound {
		t.Fatalf("missing automation detail=%d %s", missingAutomation.Code, missingAutomation.Body.String())
	}
}

func TestRepositoryReviewDirectPostBoundaryOutcomes(t *testing.T) {
	t.Run("request validation", func(t *testing.T) {
		handler, mux, _ := newRepositoryReviewAutomationTestHandler(t)
		t.Cleanup(handler.Shutdown)
		path := "/api/repository-reviews/automations/rra_missing/findings/rf_missing/post"
		requests := []*http.Request{
			httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{`)),
			httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"expected_version":0}`)),
			httptest.NewRequest(
				http.MethodPost,
				path,
				strings.NewReader(`{"expected_version":1,"instructions":"\u0000"}`),
			),
		}
		for _, request := range requests {
			setRepositoryReviewMutationHeaders(request)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("invalid direct post=%d %s", response.Code, response.Body.String())
			}
		}
		crossSite := httptest.NewRequest(
			http.MethodPost, "http://launcher.invalid"+path,
			strings.NewReader(`{"expected_version":1}`),
		)
		crossSite.Header.Set("Content-Type", "application/json")
		crossSite.Header.Set("Sec-Fetch-Site", "cross-site")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, crossSite)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("cross-site direct post=%d %s", response.Code, response.Body.String())
		}

		missing := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost, path, map[string]any{"expected_version": 1},
		)
		if missing.Code != http.StatusNotFound {
			t.Fatalf("missing direct post=%d %s", missing.Code, missing.Body.String())
		}
	})

	t.Run("stale occurrence", func(t *testing.T) {
		_, mux, _, state, automation := newMappedRepositoryReviewDetailFixture(t)
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost, repositoryReviewDirectPostPath(automation.ID, state.Findings[0].ID),
			map[string]any{"expected_version": state.Findings[0].Version + 1},
		)
		if response.Code != http.StatusConflict {
			t.Fatalf("stale direct post=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("generation fails", func(t *testing.T) {
		_, mux, _, state, automation := newMappedRepositoryReviewDetailFixture(t)
		previous := runRepositoryReviewIssueWriter
		t.Cleanup(func() { runRepositoryReviewIssueWriter = previous })
		runRepositoryReviewIssueWriter = func(
			context.Context, *Handler, repoaudit.RepositoryReviewAutomation,
			repoaudit.Finding, []repoaudit.FindingContext, string, string,
		) (repositoryReviewIssueWriterResult, error) {
			return repositoryReviewIssueWriterResult{}, errors.New("provider unavailable")
		}
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost, repositoryReviewDirectPostPath(automation.ID, state.Findings[0].ID),
			map[string]any{"expected_version": state.Findings[0].Version},
		)
		if response.Code != http.StatusBadGateway ||
			!strings.Contains(response.Body.String(), `"code":"generation_failed"`) {
			t.Fatalf("failed generation direct post=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("profile resolution fails", func(t *testing.T) {
		handler, mux, _, state, automation := newMappedRepositoryReviewDetailFixture(t)
		cfg, err := config.LoadConfig(handler.configPath)
		if err != nil {
			t.Fatal(err)
		}
		cfg.Agents.Defaults.AccountRef = ""
		if err := config.SaveConfig(handler.configPath, cfg); err != nil {
			t.Fatal(err)
		}
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost, repositoryReviewDirectPostPath(automation.ID, state.Findings[0].ID),
			map[string]any{"expected_version": state.Findings[0].Version},
		)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("missing profile account direct post=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("generation ID fails", func(t *testing.T) {
		_, mux, _, state, automation := newMappedRepositoryReviewDetailFixture(t)
		previousRandom := readRepositoryReviewIssueGenerationRandom
		t.Cleanup(func() { readRepositoryReviewIssueGenerationRandom = previousRandom })
		readRepositoryReviewIssueGenerationRandom = func([]byte) (int, error) {
			return 0, errors.New("entropy unavailable")
		}
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost, repositoryReviewDirectPostPath(automation.ID, state.Findings[0].ID),
			map[string]any{"expected_version": state.Findings[0].Version},
		)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("generation ID direct post=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("publication fails after custom generation", func(t *testing.T) {
		_, mux, _, state, automation := newMappedRepositoryReviewDetailFixture(t)
		previous := runRepositoryReviewIssueWriter
		t.Cleanup(func() { runRepositoryReviewIssueWriter = previous })
		runRepositoryReviewIssueWriter = successfulRepositoryReviewCoverageWriter
		installEventProxyStubs(t, func(*http.Request, time.Duration) (*http.Response, error) {
			return eventUpstreamResponse(
				http.StatusBadGateway,
				`{"outcome":"failed","code":"provider_failed","message":"publication failed"}`,
			), nil
		})
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost, repositoryReviewDirectPostPath(automation.ID, state.Findings[0].ID),
			map[string]any{
				"expected_version": state.Findings[0].Version,
				"instructions":     "Emphasize the observable impact.",
			},
		)
		if response.Code != http.StatusBadGateway ||
			!strings.Contains(response.Body.String(), `"outcome":"failed"`) {
			t.Fatalf("failed publication direct post=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("ledger disappears after publication", func(t *testing.T) {
		_, mux, workspace, state, automation := newMappedRepositoryReviewDetailFixture(t)
		previous := runRepositoryReviewIssueWriter
		t.Cleanup(func() { runRepositoryReviewIssueWriter = previous })
		runRepositoryReviewIssueWriter = successfulRepositoryReviewCoverageWriter
		installEventProxyStubs(t, func(*http.Request, time.Duration) (*http.Response, error) {
			statePath := filepath.Join(
				workspace, "repository_reviews",
				"repo_"+strings.TrimPrefix(state.ID, "rrp_")+".json",
			)
			_ = os.Remove(statePath)
			_ = os.Remove(strings.TrimSuffix(statePath, ".json") + ".summary.json")
			return eventUpstreamResponse(http.StatusOK, `{"outcome":"posted"}`), nil
		})
		response := repositoryReviewAutomationMutation(
			t, mux, http.MethodPost, repositoryReviewDirectPostPath(automation.ID, state.Findings[0].ID),
			map[string]any{"expected_version": state.Findings[0].Version},
		)
		if response.Code != http.StatusNotFound {
			t.Fatalf("missing final ledger direct post=%d %s", response.Code, response.Body.String())
		}
	})
}

func TestRepositoryReviewAdvertisedDefaultBranchAndGitOutputBoundaries(t *testing.T) {
	if _, err := resolveRepositoryReviewAdvertisedDefaultBranch(
		t.Context(), nil, repoaudit.RepositoryReviewAutomation{},
	); err == nil {
		t.Fatal("nil config default branch resolution succeeded")
	}

	handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	cfg, loadErr := config.LoadConfig(handler.configPath)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	repository := newRepositoryReviewDefaultBranchGitFixture(t)
	cfg.GitWorkspaces.RootDir = t.TempDir()
	branch, branchErr := resolveRepositoryReviewAdvertisedDefaultBranch(
		t.Context(), cfg, repoaudit.RepositoryReviewAutomation{
			ID: "rra_default_branch", Repository: repository,
		},
	)
	if branchErr != nil || branch != "main" {
		t.Fatalf("default branch=%q err=%v", branch, branchErr)
	}
	blockedRoot := filepath.Join(t.TempDir(), "workspace-root-file")
	if err := os.WriteFile(blockedRoot, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	blockedConfig := *cfg
	blockedConfig.GitWorkspaces.RootDir = blockedRoot
	if _, err := resolveRepositoryReviewAdvertisedDefaultBranch(
		t.Context(), &blockedConfig, repoaudit.RepositoryReviewAutomation{
			ID: "rra_blocked_default", Repository: repository,
		},
	); err == nil {
		t.Fatal("blocked workspace root resolved a default branch")
	}
	missingRepositoryConfig := *cfg
	missingRepositoryConfig.GitWorkspaces.RootDir = t.TempDir()
	if _, err := resolveRepositoryReviewAdvertisedDefaultBranch(
		t.Context(), &missingRepositoryConfig, repoaudit.RepositoryReviewAutomation{
			ID: "rra_missing_default", Repository: filepath.Join(t.TempDir(), "missing"),
		},
	); err == nil {
		t.Fatal("missing repository resolved a default branch")
	}
	realGit, lookupErr := exec.LookPath("git")
	if lookupErr != nil {
		t.Fatal(lookupErr)
	}
	wrapperRoot := t.TempDir()
	wrapper := filepath.Join(wrapperRoot, "git")
	wrapperScript := "#!/bin/sh\nif [ \"$1\" = \"symbolic-ref\" ]; then exit 1; fi\nexec \"" +
		realGit + "\" \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(wrapperScript), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", wrapperRoot+string(os.PathListSeparator)+os.Getenv("PATH"))
	noSymbolicRefConfig := *cfg
	noSymbolicRefConfig.GitWorkspaces.RootDir = t.TempDir()
	if _, err := resolveRepositoryReviewAdvertisedDefaultBranch(
		t.Context(), &noSymbolicRefConfig, repoaudit.RepositoryReviewAutomation{
			ID: "rra_no_symbolic_ref", Repository: repository,
		},
	); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("symbolic-ref failure error=%v", err)
	}
	output, gitErr := repositoryReviewGitOutput(
		t.Context(), repository, 3, "git", "rev-parse", "--abbrev-ref", "HEAD",
	)
	if gitErr == nil || string(output) != "mai" {
		t.Fatalf("bounded git output=%q err=%v", output, gitErr)
	}
	if _, err := repositoryReviewGitOutput(
		t.Context(), repository, 10, "git", "rev-parse", "missing-ref",
	); err == nil {
		t.Fatal("failing git command succeeded")
	}
}

func TestRepositoryReviewRemainingDetailProjectionBranches(t *testing.T) {
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewGenerationFindings(t, workspace, 3)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	automation := seedRepositoryReviewDetailAutomation(t, handler, state.Repository, state.Runs[0].ID)
	page := httptest.NewRecorder()
	mux.ServeHTTP(page, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations/"+automation.ID+"/findings?scope=all&offset=50&limit=2",
		nil,
	))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), `"repository_finding_offset":2`) {
		t.Fatalf("bounded aggregate page=%d %s", page.Code, page.Body.String())
	}
	firstPage := httptest.NewRecorder()
	mux.ServeHTTP(firstPage, httptest.NewRequest(
		http.MethodGet,
		"/api/repository-reviews/automations/"+automation.ID+"/findings?scope=all&offset=0&limit=1",
		nil,
	))
	if firstPage.Code != http.StatusOK ||
		!strings.Contains(firstPage.Body.String(), `"next_repository_finding_offset":1`) {
		t.Fatalf("next aggregate page=%d %s", firstPage.Code, firstPage.Body.String())
	}

	orphan := state.Findings[0]
	orphan.RepositoryFindingID = "rrf_missing"
	capabilities := repositoryReviewFindingCapabilities(state, orphan)
	if capabilities.CanGenerate {
		t.Fatalf("orphan aggregate capabilities=%#v", capabilities)
	}
	projection := repositoryReviewFindingDetail(
		repositoryReviewAutomationLedger{Automation: automation, State: state, Found: true}, orphan,
	)
	if _, found := projection["repository_finding"]; found {
		t.Fatalf("orphan detail unexpectedly projected aggregate: %#v", projection)
	}
	if _, found := repositoryReviewAggregateIssueByFinding(state, orphan); found {
		t.Fatal("orphan aggregate issue was projected")
	}

	aggregate := state.RepositoryFindings[0]
	aggregate.Issue.State = repoaudit.RepositoryFindingIssueOpen
	state.RepositoryFindings[0] = aggregate
	associated := state.Findings[0]
	associated.RepositoryFindingID = aggregate.ID
	if _, found := repositoryReviewAggregateIssueByFinding(state, associated); found {
		t.Fatal("aggregate issue without an occurrence draft was projected")
	}
	state.RepositoryFindings[0].ReviewFindingIDs = append(
		[]string{"rf_missing_occurrence"}, state.RepositoryFindings[0].ReviewFindingIDs...,
	)
	if _, found := repositoryReviewAggregateIssueByFinding(state, associated); found {
		t.Fatal("missing occurrence issue was projected")
	}
}

func TestRepositoryReviewCurrentIssueProfileErrorAndFallbackBranches(t *testing.T) {
	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store := repoaudit.NewStore(workspace)
	baseLedger := repositoryReviewAutomationLedger{
		Store: store,
		Automation: repoaudit.RepositoryReviewAutomation{
			IssueWriterModel: "cheap", AccountRef: "api",
		},
	}
	missing := baseLedger
	missing.Automation.ProfileID = "rrpf_missing"
	if _, err := handler.repositoryReviewCurrentIssueProfile(t.Context(), missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing profile error=%v", err)
	}
	profile, createErr := store.CreateProfile(t.Context(), repoaudit.RepositoryReviewProfile{
		Name: "Fallback writer", ReviewFocus: "Find bugs.", ReviewerModel: "cheap",
		IssuePrompt: "Present the diagnosis.", AccountRef: "api", AutoContinue: true,
		MaxFilesPerRun: 4, MaxContentBytes: 65536, MaxParallelChildren: 1,
	})
	if createErr != nil {
		t.Fatal(createErr)
	}
	fallback := baseLedger
	fallback.Automation.ProfileID = profile.ID
	got, resolveErr := handler.repositoryReviewCurrentIssueProfile(t.Context(), fallback)
	if resolveErr != nil || got.Model != profile.ReviewerModel {
		t.Fatalf("fallback writer profile=%#v err=%v", got, resolveErr)
	}

	badConfigPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(badConfigPath, []byte(`{`), 0o600); err != nil {
		t.Fatal(err)
	}
	badHandler := &Handler{configPath: badConfigPath}
	if _, err := badHandler.repositoryReviewCurrentIssueProfile(t.Context(), fallback); err == nil {
		t.Fatal("corrupt config profile resolution succeeded")
	}

	cfg, configErr := config.LoadConfig(handler.configPath)
	if configErr != nil {
		t.Fatal(configErr)
	}
	cfg.ModelAliases[0].DisabledAccounts = []string{"api"}
	if err := config.SaveConfig(handler.configPath, cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := handler.repositoryReviewCurrentIssueProfile(t.Context(), fallback); err == nil {
		t.Fatal("disabled issue writer alias was accepted")
	}
	unknownModelProfile, unknownCreateErr := store.CreateProfile(t.Context(), repoaudit.RepositoryReviewProfile{
		Name: "Unknown writer", ReviewFocus: "Find bugs.", ReviewerModel: "cheap",
		IssueWriterModel: "not-configured", IssuePrompt: "Present it.", AccountRef: "api",
		AutoContinue: true, MaxFilesPerRun: 4, MaxContentBytes: 65536, MaxParallelChildren: 1,
	})
	if unknownCreateErr != nil {
		t.Fatal(unknownCreateErr)
	}
	unknown := baseLedger
	unknown.Automation.ProfileID = unknownModelProfile.ID
	if _, err := handler.repositoryReviewCurrentIssueProfile(t.Context(), unknown); err == nil {
		t.Fatal("unknown issue writer alias was accepted")
	}
	blockedRoot := filepath.Join(t.TempDir(), "profile-store-file")
	if err := os.WriteFile(blockedRoot, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	loadFailure := missing
	loadFailure.Store = repoaudit.NewStore(blockedRoot)
	if _, err := handler.repositoryReviewCurrentIssueProfile(t.Context(), loadFailure); err == nil {
		t.Fatal("profile store failure unexpectedly resolved")
	}
}

func TestRepositoryReviewCurrentIssueProfileRejectsBlankEffectiveAccount(t *testing.T) {
	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store := repoaudit.NewStore(workspace)
	profile, err := store.CreateProfile(t.Context(), repoaudit.RepositoryReviewProfile{
		Name: "Blank account", ReviewFocus: "Find bugs.", ReviewerModel: "cheap",
		IssuePrompt: "Present it.", AutoContinue: true, MaxFilesPerRun: 4,
		MaxContentBytes: 65536, MaxParallelChildren: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Agents.Defaults.AccountRef = ""
	cfg.Agents.Defaults.ModelName = ""
	if err := config.SaveConfig(handler.configPath, cfg); err != nil {
		t.Fatal(err)
	}
	ledger := repositoryReviewAutomationLedger{
		Store: store, Automation: repoaudit.RepositoryReviewAutomation{ProfileID: profile.ID},
	}
	if _, err := handler.repositoryReviewCurrentIssueProfile(t.Context(), ledger); err == nil {
		t.Fatal("blank profile account unexpectedly resolved")
	}
}

func TestRepositoryReviewCurrentIssueProfileRejectsUnknownWriterAlias(t *testing.T) {
	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store := repoaudit.NewStore(workspace)
	profile, err := store.CreateProfile(t.Context(), repoaudit.RepositoryReviewProfile{
		Name: "Unknown writer", ReviewFocus: "Find bugs.", ReviewerModel: "cheap",
		IssueWriterModel: "not-configured", IssuePrompt: "Present it.", AccountRef: "api",
		AutoContinue: true, MaxFilesPerRun: 4, MaxContentBytes: 65536, MaxParallelChildren: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	ledger := repositoryReviewAutomationLedger{
		Store: store, Automation: repoaudit.RepositoryReviewAutomation{ProfileID: profile.ID},
	}
	if _, err := handler.repositoryReviewCurrentIssueProfile(t.Context(), ledger); err == nil {
		t.Fatal("unknown issue writer alias unexpectedly resolved")
	}
}

func TestRepositoryReviewAggregateIssueFallbackProjection(t *testing.T) {
	issue := repoaudit.IssueDraft{
		ID: "rrid_linked", Origin: repoaudit.IssueDraftOriginLinked,
		State: repoaudit.IssueDraftPosted, Canonical: true,
	}
	older := repoaudit.Finding{ID: "rf_old", Status: repoaudit.FindingOpen, IssueDraftID: issue.ID}
	current := repoaudit.Finding{
		ID: "rf_current", Status: repoaudit.FindingOpen, RepositoryFindingID: "rrf_aggregate",
	}
	state := repoaudit.RepositoryState{
		Repository: "owner/repo", Findings: []repoaudit.Finding{older, current},
		IssueDrafts: []repoaudit.IssueDraft{issue},
		RepositoryFindings: []repoaudit.RepositoryFinding{{
			ID: "rrf_aggregate", MatchState: repoaudit.RepositoryMatchKnown,
			Lifecycle:        repoaudit.RepositoryFindingOpen,
			ReviewFindingIDs: []string{"rf_missing", older.ID, current.ID},
			Issue:            repoaudit.RepositoryFindingIssueAssociation{State: repoaudit.RepositoryFindingIssueOpen},
		}},
	}
	projected := repositoryReviewFindingDetail(repositoryReviewAutomationLedger{State: state}, current)
	if projectedIssue, ok := projected["issue"].(repoaudit.IssueDraft); !ok || projectedIssue.ID != issue.ID {
		t.Fatalf("aggregate issue projection=%#v", projected)
	}
	capabilities := repositoryReviewFindingCapabilities(state, current)
	if capabilities.CanGenerate || capabilities.CanUnlinkIssue {
		t.Fatalf("aggregate issue capabilities=%#v", capabilities)
	}
}

func TestRepositoryReviewControllerStartReconcileFailureAndCanceledReconcile(t *testing.T) {
	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	store := repoaudit.NewStore(cfg.WorkspacePath())
	state := seedRepositoryReviewAPIState(t, cfg.WorkspacePath())
	corrupt, found, err := store.Get(state.Repository)
	if err != nil || !found {
		t.Fatal(err)
	}
	corrupt.MappingJobs[0].ReviewFindingID = "rf_missing"
	corrupt.Version++
	corrupt.UpdatedAt = time.Now().UTC()
	data, err := json.Marshal(corrupt)
	if err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(
		workspace, "repository_reviews",
		"repo_"+strings.TrimPrefix(corrupt.ID, "rrp_")+".json",
	)
	if err := os.WriteFile(statePath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	controller := newRepositoryReviewController(handler)
	if err := controller.Start(); err == nil || !strings.Contains(err.Error(), "reconcile") {
		t.Fatalf("controller reconcile error=%v", err)
	}
	hasLease := controller.releaseLease != nil
	if hasLease || controller.ctx.Err() == nil {
		t.Fatalf(
			"failed controller retained lease=%t context=%v",
			hasLease, controller.ctx.Err(),
		)
	}

	canceled := newRepositoryReviewController(handler)
	canceled.leasedStore = store
	canceled.leasedConfig = cfg
	canceled.cancel()
	canceled.reconcile()
}

func TestRepositoryReviewControllerAdmissionUsesAdvertisedDefaultResolver(t *testing.T) {
	for _, test := range []struct {
		name       string
		resolveErr error
	}{
		{name: "failure", resolveErr: errors.New("default unavailable")},
		{name: "success stops after resolution"},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler, _, _ := newRepositoryReviewAutomationTestHandler(t)
			t.Cleanup(handler.Shutdown)
			store, err := handler.repositoryReviewStore()
			if err != nil {
				t.Fatal(err)
			}
			automation := testRepositoryReviewAutomation()
			automation.Repository = newRepositoryReviewDefaultBranchGitFixture(t)
			automation.Ref = "main"
			automation, err = store.CreateAutomation(t.Context(), automation)
			if err != nil {
				t.Fatal(err)
			}
			controller := newRepositoryReviewController(handler)
			t.Cleanup(controller.Stop)
			controller.resolveDefaultBranch = func(
				context.Context, *config.Config, repoaudit.RepositoryReviewAutomation,
			) (string, error) {
				return "main", test.resolveErr
			}
			if test.resolveErr == nil {
				controller.update = func(
					context.Context, repoaudit.Store, string, int64,
					func(*repoaudit.RepositoryReviewAutomation) error,
				) (repoaudit.RepositoryReviewAutomation, error) {
					return repoaudit.RepositoryReviewAutomation{}, errors.New("stop after default resolution")
				}
			}
			_, startErr := controller.startAutomationAtCommit(
				t.Context(), automation.ID, automation.Version, false, "start", "",
			)
			if startErr == nil {
				t.Fatal("admission unexpectedly succeeded")
			}
		})
	}
}

func TestRepositoryReviewControllerRestoresLegacyRememberedCommitForAdmission(t *testing.T) {
	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	store, err := handler.repositoryReviewStore()
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.Repository = newRepositoryReviewDefaultBranchGitFixture(t)
	automation.Ref = "main"
	automation, err = store.CreateAutomation(t.Context(), automation)
	if err != nil {
		t.Fatal(err)
	}
	automation.Ref = "HEAD"
	automation.ResolvedCommitSHA = strings.Repeat("f", 40)
	data, err := json.Marshal(automation)
	if err != nil {
		t.Fatal(err)
	}
	automationPath := filepath.Join(
		workspace, "repository_reviews", "automation_"+automation.ID+".json",
	)
	if err := os.WriteFile(automationPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	controller := newRepositoryReviewController(handler)
	t.Cleanup(controller.Stop)
	controller.resolveCommit = func(
		_ context.Context, _ *config.Config,
		candidate repoaudit.RepositoryReviewAutomation, _ string,
	) (string, error) {
		if candidate.ResolvedCommitSHA != strings.Repeat("f", 40) {
			t.Fatalf("remembered commit was not restored: %#v", candidate)
		}
		return "", errors.New("stop after remembered commit restoration")
	}
	if _, err := controller.startAutomationAtCommit(
		t.Context(), automation.ID, automation.Version, false, "start", "",
	); err == nil {
		t.Fatal("unreachable remembered commit unexpectedly admitted")
	}
}

func TestRepositoryReviewFinishSnapshotsUnmappedCampaign(t *testing.T) {
	handler, _, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	store := repoaudit.NewStore(workspace)
	cfg, err := config.LoadConfig(handler.configPath)
	if err != nil {
		t.Fatal(err)
	}
	automation := testRepositoryReviewAutomation()
	automation.Repository = state.Repository
	automation.Status = repoaudit.RepositoryReviewAutomationRunning
	automation.ActiveRunID = "run-finish-snapshot"
	automation.RunIDs = append([]string{state.Runs[0].ID}, automation.ActiveRunID)
	automation.StartedAt = state.Findings[0].CreatedAt.Add(-time.Minute)
	automation, err = store.CreateAutomation(t.Context(), automation)
	if err != nil {
		t.Fatal(err)
	}
	controller := newRepositoryReviewController(handler)
	controller.active[automation.ID] = &repositoryReviewActiveRun{
		runID: automation.ActiveRunID, store: store, config: cfg,
		reservations: make(map[int]repositoryReviewTaskReservation), guardMu: &sync.Mutex{},
	}
	controller.finishAutomationRun(
		automation.ID, automation.ActiveRunID,
		&workflows.RunResult{Status: workflows.RunStatusFailed}, errors.New("run failed"), false,
		nil,
	)
	updated, found, err := store.Get(state.Repository)
	if err != nil || !found || len(updated.MappingJobs) == 0 ||
		updated.MappingJobs[0].ModelSnapshot.Model == "" {
		t.Fatalf("snapshotted mapping state=%#v found=%v err=%v", updated.MappingJobs, found, err)
	}
}

func newMappedRepositoryReviewDetailFixture(
	t *testing.T,
) (*Handler, *http.ServeMux, string, repoaudit.RepositoryState, repoaudit.RepositoryReviewAutomation) {
	t.Helper()
	handler, mux, workspace := newRepositoryReviewAutomationTestHandler(t)
	t.Cleanup(handler.Shutdown)
	state := seedRepositoryReviewAPIState(t, workspace)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	automation := seedRepositoryReviewDetailAutomation(
		t, handler, state.Repository, state.Runs[0].ID,
	)
	return handler, mux, workspace, state, automation
}

func repositoryReviewDirectPostPath(automationID, findingID string) string {
	return "/api/repository-reviews/automations/" + automationID +
		"/findings/" + findingID + "/post"
}

func successfulRepositoryReviewCoverageWriter(
	context.Context,
	*Handler,
	repoaudit.RepositoryReviewAutomation,
	repoaudit.Finding,
	[]repoaudit.FindingContext,
	string,
	string,
) (repositoryReviewIssueWriterResult, error) {
	return repositoryReviewIssueWriterResult{
		Title: "Generated issue", Body: "Grounded diagnosis and provenance.", Labels: []string{"bug"},
	}, nil
}

func newRepositoryReviewDefaultBranchGitFixture(t *testing.T) string {
	t.Helper()
	repository := t.TempDir()
	git := func(arguments ...string) {
		t.Helper()
		command := exec.Command("git", arguments...)
		command.Dir = repository
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", arguments, err, output)
		}
	}
	git("init", "-b", "main")
	git("config", "user.email", "default-branch@example.test")
	git("config", "user.name", "Default Branch Test")
	if err := os.WriteFile(
		filepath.Join(repository, "service.go"), []byte("package service\n"), 0o600,
	); err != nil {
		t.Fatal(err)
	}
	git("add", "service.go")
	git("commit", "-m", "initial")
	return repository
}
