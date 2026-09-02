package gateway

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sipeed/picoclaw/pkg/repoaudit"
	"github.com/sipeed/picoclaw/pkg/reviews"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/workflows"
)

func TestRepositoryReviewGatewayAutomationProjectionHidesInternalScopeState(t *testing.T) {
	projected := projectRepositoryReviewGatewayAutomation(repoaudit.RepositoryReviewAutomation{
		CampaignID:              repoaudit.NewRepositoryReviewCampaignID(),
		CampaignRecoveryPending: true,
		ModelCoverageSketches:   map[string]string{"review": "internal"},
		ScopeSelection: &repoaudit.RepositoryReviewScopeSelection{
			IncludePrefixes: []string{"pkg"},
		},
	})
	if projected.ModelCoverageSketches != nil || projected.ScopeSelection != nil ||
		projected.CampaignID != "" || projected.CampaignRecoveryPending ||
		!projected.Progress.ScopeFrozen {
		t.Fatalf("gateway projection exposed internal state: %#v", projected)
	}
	projected = projectRepositoryReviewGatewayAutomation(repoaudit.RepositoryReviewAutomation{
		Progress: repoaudit.RepositoryReviewProgress{ScopeFrozen: true},
	})
	if projected.Progress.ScopeFrozen {
		t.Fatal("gateway projection trusted a non-durable scope_frozen marker")
	}
}

func TestRepositoryReviewGatewayFindingProjectionHidesCampaignAuthority(t *testing.T) {
	canary := repoaudit.NewRepositoryReviewCampaignID()
	finding := repoaudit.Finding{ID: "finding", CampaignID: canary, ContextIDs: []string{"context"}}
	state := repoaudit.RepositoryState{Contexts: []repoaudit.FindingContext{{
		ID: "context", CampaignID: canary,
	}}}
	projected := projectRepositoryReviewGatewayFinding(finding)
	contexts := repositoryReviewGatewayFindingContexts(state, finding)
	if projected.CampaignID != "" || len(contexts) != 1 || contexts[0].CampaignID != "" {
		t.Fatalf("gateway campaign projection finding=%#v contexts=%#v", projected, contexts)
	}
	if finding.CampaignID != canary || state.Contexts[0].CampaignID != canary {
		t.Fatal("gateway projection mutated stored campaign authority")
	}
}

type repositoryReviewIssueSyncMCPManager struct {
	responses map[string]string
}

func (manager *repositoryReviewIssueSyncMCPManager) CallTool(
	_ context.Context,
	_, toolName string,
	_ map[string]any,
) (*sdkmcp.CallToolResult, error) {
	return &sdkmcp.CallToolResult{Content: []sdkmcp.Content{
		&sdkmcp.TextContent{Text: manager.responses[toolName]},
	}}, nil
}

func repositoryReviewLinkedAggregateFixture(
	t *testing.T,
	issueState string,
) (
	repoaudit.Store,
	repoaudit.RepositoryState,
	repoaudit.RepositoryFinding,
	repoaudit.RepositoryReviewAutomation,
) {
	t.Helper()
	workspace := t.TempDir()
	store, state, finding, automation := repositoryReviewIssueLinkTestFixture(
		t, workspace, "owner/repo",
	)
	updated, _, err := store.LinkExistingIssue(repoaudit.ExistingIssueLink{
		Repository: state.Repository, FindingID: finding.ID,
		ExpectedFindingVersion: finding.Version,
		ExternalID:             "99",
		ExternalURL:            "https://github.com/owner/repo/issues/12",
		Title:                  "Existing waiter issue",
		Body:                   "The waiter remains blocked.",
		State:                  issueState,
		Origin:                 repoaudit.IssueDraftOriginLinked,
		Confirmed:              true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.RepositoryFindings) != 1 {
		t.Fatalf("repository findings=%#v", updated.RepositoryFindings)
	}
	return store, updated, updated.RepositoryFindings[0], automation
}

func TestRepositoryReviewFindingSyncProtectedRouteRefreshesIssueSnapshot(t *testing.T) {
	loop := repositoryReviewRankingTestLoop(t, &repositoryReviewRankingProvider{})
	store, state, occurrence, automation := repositoryReviewIssueLinkTestFixture(
		t, loop.GetConfig().WorkspacePath(), "owner/repo",
	)
	state, _, err := store.LinkExistingIssue(repoaudit.ExistingIssueLink{
		Repository: state.Repository, FindingID: occurrence.ID,
		ExpectedFindingVersion: occurrence.Version,
		ExternalID:             "99",
		ExternalURL:            "https://github.com/owner/repo/issues/12",
		Title:                  "Old title",
		State:                  "open",
		Origin:                 repoaudit.IssueDraftOriginLinked,
		Confirmed:              true,
	})
	if err != nil {
		t.Fatal(err)
	}
	aggregate := state.RepositoryFindings[0]
	manager := &repositoryReviewPublicationMCPManager{searchText: `{
		"id":99,"number":12,"title":"Refreshed waiter issue","state":"closed",
		"html_url":"https://github.com/owner/repo/issues/12"
	}`}
	loop.RegisterTool(tools.NewMCPTool(manager, reviews.DefaultGitHubMCPServer, &sdkmcp.Tool{
		Name: reviews.GitHubIssueReadTool,
		InputSchema: map[string]any{
			"type": "object", "additionalProperties": true,
		},
	}))
	request := httptest.NewRequest(
		http.MethodPost,
		repositoryReviewPublicationRoute+"automations/"+automation.ID+
			"/repository-findings/"+aggregate.ID+"/sync",
		strings.NewReader(`{"expected_version":`+strconv.FormatInt(aggregate.Version, 10)+`}`),
	)
	response := httptest.NewRecorder()
	newRepositoryReviewPublicationHandler(loop).ServeHTTP(response, request)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"state":"closed"`) ||
		!strings.Contains(response.Body.String(), `"title":"Refreshed waiter issue"`) {
		t.Fatalf("sync response=%d %s", response.Code, response.Body.String())
	}
	persisted, found, err := store.Get(state.Repository)
	if err != nil || !found || persisted.RepositoryFindings[0].Issue.State != repoaudit.RepositoryFindingIssueClosed ||
		persisted.RepositoryFindings[0].Lifecycle != repoaudit.RepositoryFindingResolutionPending {
		t.Fatalf("persisted=%#v found=%v err=%v", persisted.RepositoryFindings, found, err)
	}
}

func TestRepositoryReviewFindingSyncFailureFences(t *testing.T) {
	handler := newRepositoryReviewPublicationHandler(nil)
	store, state, aggregate, automation := repositoryReviewLinkedAggregateFixture(t, "open")
	for _, test := range []struct {
		name      string
		body      string
		state     repoaudit.RepositoryState
		findingID string
		status    int
		code      string
	}{
		{
			name: "malformed", body: `{`, state: state, findingID: aggregate.ID,
			status: http.StatusBadRequest, code: "invalid_request",
		},
		{
			name: "zero version", body: `{"expected_version":0}`, state: state, findingID: aggregate.ID,
			status: http.StatusBadRequest, code: "invalid_request",
		},
		{
			name: "missing aggregate", body: `{"expected_version":1}`, state: state, findingID: "rrf_missing",
			status: http.StatusNotFound, code: "not_found",
		},
		{
			name: "stale", body: `{"expected_version":999}`, state: state, findingID: aggregate.ID,
			status: http.StatusConflict, code: "stale_repository_review",
		},
		{
			name: "tool runtime unavailable", body: `{"expected_version":` + strconv.FormatInt(aggregate.Version, 10) + `}`,
			state: state, findingID: aggregate.ID,
			status: http.StatusServiceUnavailable, code: "issue_sync_unavailable",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			handler.serveRepositoryReviewFindingSync(
				response, request, nil, store, automation, test.state, test.findingID,
			)
			if response.Code != test.status ||
				!strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response=%d %s", response.Code, response.Body.String())
			}
		})
	}

	provisional := state
	provisional.RepositoryFindings = append(
		[]repoaudit.RepositoryFinding(nil), state.RepositoryFindings...,
	)
	provisional.RepositoryFindings[0].MatchState = repoaudit.RepositoryMatchProvisional
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(
		`{"expected_version":`+strconv.FormatInt(aggregate.Version, 10)+`}`,
	))
	response := httptest.NewRecorder()
	handler.serveRepositoryReviewFindingSync(
		response, request, nil, store, automation, provisional, aggregate.ID,
	)
	if response.Code != http.StatusConflict {
		t.Fatalf("provisional response=%d %s", response.Code, response.Body.String())
	}

	invalidURL := state
	invalidURL.RepositoryFindings = append(
		[]repoaudit.RepositoryFinding(nil), state.RepositoryFindings...,
	)
	invalidURL.RepositoryFindings[0].Issue.URL = "https://github.com/other/repo/issues/12"
	request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(
		`{"expected_version":`+strconv.FormatInt(aggregate.Version, 10)+`}`,
	))
	response = httptest.NewRecorder()
	handler.serveRepositoryReviewFindingSync(
		response, request, nil, store, automation, invalidURL, aggregate.ID,
	)
	if response.Code != http.StatusBadRequest ||
		!strings.Contains(response.Body.String(), `"code":"invalid_issue_url"`) {
		t.Fatalf("invalid URL response=%d %s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewFindingSyncProviderFailuresAndUnknownSnapshot(t *testing.T) {
	for _, test := range []struct {
		name        string
		issueText   string
		issueErr    error
		providerErr error
		status      int
		code        string
		wantUnknown bool
	}{
		{
			name: "read failure becomes unknown", issueErr: errors.New("GitHub unavailable"),
			status: http.StatusOK, wantUnknown: true,
		},
		{
			name: "malformed issue", issueText: `{"number":12}`,
			status: http.StatusBadGateway, code: "invalid_gateway_response",
		},
		{
			name: "provider unavailable", providerErr: errors.New("provider unavailable"),
			status: http.StatusServiceUnavailable, code: "issue_sync_unavailable",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			loop := repositoryReviewRankingTestLoop(t, &repositoryReviewRankingProvider{})
			store, state, finding, automation := repositoryReviewIssueLinkTestFixture(
				t, loop.GetConfig().WorkspacePath(), "owner/repo",
			)
			state, _, err := store.LinkExistingIssue(repoaudit.ExistingIssueLink{
				Repository: state.Repository, FindingID: finding.ID,
				ExpectedFindingVersion: finding.Version,
				ExternalID:             "99", ExternalURL: "https://github.com/owner/repo/issues/12",
				Title: "Old issue", State: "open", Origin: repoaudit.IssueDraftOriginLinked,
				Confirmed: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			aggregate := state.RepositoryFindings[0]
			manager := &repositoryReviewPublicationMCPManager{
				searchText: test.issueText, searchErr: test.issueErr,
			}
			loop.RegisterTool(tools.NewMCPTool(manager, reviews.DefaultGitHubMCPServer, &sdkmcp.Tool{
				Name:        reviews.GitHubIssueReadTool,
				InputSchema: map[string]any{"type": "object", "additionalProperties": true},
			}))
			handler := newRepositoryReviewPublicationHandler(loop)
			if test.providerErr != nil {
				handler.newGitHubProvider = func(workflows.ToolRunner, string) (*reviews.GitHubProvider, error) {
					return nil, test.providerErr
				}
			}
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(
				`{"expected_version":`+strconv.FormatInt(aggregate.Version, 10)+`}`,
			))
			response := httptest.NewRecorder()
			handler.serveRepositoryReviewFindingSync(
				response, request, loop, store, automation, state, aggregate.ID,
			)
			if response.Code != test.status || test.code != "" &&
				!strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response=%d %s", response.Code, response.Body.String())
			}
			if test.wantUnknown && !strings.Contains(
				response.Body.String(), `"state":"unknown"`,
			) {
				t.Fatalf("unknown response=%d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRepositoryReviewFindingSyncPersistenceFailuresAreFenced(t *testing.T) {
	for _, test := range []struct {
		name      string
		issueText string
		issueErr  error
	}{
		{name: "unknown snapshot write", issueErr: errors.New("read unavailable")},
		{name: "refreshed snapshot write", issueText: `{
			"id":99,"number":12,"title":"Current issue","state":"open",
			"html_url":"https://github.com/owner/repo/issues/12"
		}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			loop := repositoryReviewRankingTestLoop(t, &repositoryReviewRankingProvider{})
			workspace := loop.GetConfig().WorkspacePath()
			store, state, occurrence, automation := repositoryReviewIssueLinkTestFixture(
				t, workspace, "owner/repo",
			)
			state, _, err := store.LinkExistingIssue(repoaudit.ExistingIssueLink{
				Repository: state.Repository, FindingID: occurrence.ID,
				ExpectedFindingVersion: occurrence.Version,
				ExternalID:             "99", ExternalURL: "https://github.com/owner/repo/issues/12",
				Title: "Old issue", State: "open", Origin: repoaudit.IssueDraftOriginLinked,
				Confirmed: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			aggregate := state.RepositoryFindings[0]
			manager := &repositoryReviewPublicationMCPManager{
				searchText: test.issueText, searchErr: test.issueErr,
				beforeReturn: func(string) {
					root := filepath.Join(workspace, "repository_reviews")
					if err := os.RemoveAll(root); err != nil {
						t.Errorf("remove ledger root: %v", err)
						return
					}
					if err := os.WriteFile(root, []byte("not a directory"), 0o600); err != nil {
						t.Errorf("poison ledger root: %v", err)
					}
				},
			}
			loop.RegisterTool(tools.NewMCPTool(manager, reviews.DefaultGitHubMCPServer, &sdkmcp.Tool{
				Name:        reviews.GitHubIssueReadTool,
				InputSchema: map[string]any{"type": "object", "additionalProperties": true},
			}))
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(
				`{"expected_version":`+strconv.FormatInt(aggregate.Version, 10)+`}`,
			))
			response := httptest.NewRecorder()
			newRepositoryReviewPublicationHandler(loop).serveRepositoryReviewFindingSync(
				response, request, loop, store, automation, state, aggregate.ID,
			)
			if response.Code != http.StatusServiceUnavailable ||
				!strings.Contains(response.Body.String(), `"code":"issue_sync_unavailable"`) &&
					!strings.Contains(response.Body.String(), `"code":"publication_unavailable"`) {
				t.Fatalf("response=%d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRepositoryReviewIssueDiscoveryAutoLinksOnlyRefetchedGroundedIssue(t *testing.T) {
	ranking := &repositoryReviewRankingProvider{responses: []string{
		`{"rankings":[{"id":"99","score":95,"explanation":"Exact causal identity.",` +
			`"matching_anchors":["operation","trigger","invariant","outcome"],` +
			`"conflicting_anchors":[]}]}`,
	}}
	loop := repositoryReviewRankingTestLoop(t, ranking)
	manager := &repositoryReviewIssueSyncMCPManager{responses: map[string]string{
		reviews.GitHubSearchIssuesTool: `{"items":[{
			"id":99,"number":12,"title":"Existing waiter issue","body":"same causal path",
			"state":"open","html_url":"https://github.com/owner/repo/issues/12"
		}]}`,
		reviews.GitHubIssueReadTool: `{
			"id":99,"number":12,"title":"Existing waiter issue","body":"same causal path",
			"state":"open","html_url":"https://github.com/owner/repo/issues/12"
		}`,
	}}
	for _, tool := range []string{reviews.GitHubSearchIssuesTool, reviews.GitHubIssueReadTool} {
		loop.RegisterTool(tools.NewMCPTool(manager, reviews.DefaultGitHubMCPServer, &sdkmcp.Tool{
			Name: tool, InputSchema: map[string]any{"type": "object", "additionalProperties": true},
		}))
	}
	_, state, finding, automation := repositoryReviewIssueLinkTestFixture(
		t, loop.GetConfig().WorkspacePath(), "owner/repo",
	)
	request := httptest.NewRequest(
		http.MethodPost,
		repositoryReviewPublicationRoute+"automations/"+automation.ID+"/findings/"+
			finding.ID+"/issue-link/candidates",
		strings.NewReader(`{"expected_version":`+strconv.FormatInt(finding.Version, 10)+`}`),
	)
	response := httptest.NewRecorder()
	newRepositoryReviewPublicationHandler(loop).ServeHTTP(response, request)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"origin":"discovered"`) ||
		!strings.Contains(response.Body.String(), `"discovered_issue"`) ||
		!strings.Contains(response.Body.String(), state.Repository) {
		t.Fatalf("discovery response=%d %s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewIssueDiscoveryDoesNotLinkFailedRefetch(t *testing.T) {
	ranking := &repositoryReviewRankingProvider{responses: []string{
		`{"rankings":[{"id":"99","score":95,"explanation":"Exact causal identity.",` +
			`"matching_anchors":["operation","trigger","invariant","outcome"],` +
			`"conflicting_anchors":[]}]}`,
	}}
	loop := repositoryReviewRankingTestLoop(t, ranking)
	manager := &repositoryReviewIssueSyncMCPManager{responses: map[string]string{
		reviews.GitHubSearchIssuesTool: `{"items":[{
			"id":99,"number":12,"title":"Existing waiter issue","body":"same causal path",
			"state":"open","html_url":"https://github.com/owner/repo/issues/12"
		}]}`,
		// The exact issue re-fetch is authoritative. A mismatched issue number
		// must leave the ranked candidate visible without creating a link.
		reviews.GitHubIssueReadTool: `{
			"id":99,"number":13,"title":"Different issue","body":"different causal path",
			"state":"open","html_url":"https://github.com/owner/repo/issues/13"
		}`,
	}}
	for _, tool := range []string{reviews.GitHubSearchIssuesTool, reviews.GitHubIssueReadTool} {
		loop.RegisterTool(tools.NewMCPTool(manager, reviews.DefaultGitHubMCPServer, &sdkmcp.Tool{
			Name: tool, InputSchema: map[string]any{"type": "object", "additionalProperties": true},
		}))
	}
	store, state, finding, automation := repositoryReviewIssueLinkTestFixture(
		t, loop.GetConfig().WorkspacePath(), "owner/repo",
	)
	request := httptest.NewRequest(
		http.MethodPost,
		repositoryReviewPublicationRoute+"automations/"+automation.ID+"/findings/"+
			finding.ID+"/issue-link/candidates",
		strings.NewReader(`{"expected_version":`+strconv.FormatInt(finding.Version, 10)+`}`),
	)
	response := httptest.NewRecorder()
	newRepositoryReviewPublicationHandler(loop).ServeHTTP(response, request)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"candidates"`) ||
		strings.Contains(response.Body.String(), `"discovered_issue"`) {
		t.Fatalf("discovery response=%d %s", response.Code, response.Body.String())
	}
	persisted, found, err := store.Get(state.Repository)
	if err != nil || !found || len(persisted.IssueDrafts) != 0 ||
		len(persisted.Findings) != 1 || persisted.Findings[0].IssueDraftID != "" {
		t.Fatalf("persisted=%#v found=%v err=%v", persisted, found, err)
	}
}

func TestRepositoryReviewIssueLinkRemainingPureBranches(t *testing.T) {
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet,
			repositoryReviewPublicationRoute+"automations/a/repository-findings/f/sync", nil),
		httptest.NewRequest(http.MethodPost,
			repositoryReviewPublicationRoute+"automations/a/repository-findings/f/not-sync", nil),
	} {
		if operation, ok := repositoryReviewAutomationOperationFromRequest(request); ok {
			t.Fatalf("unexpected operation=%#v for %s %s", operation, request.Method, request.URL)
		}
	}

	base := repoaudit.Finding{ID: "rvf", File: repoaudit.FileRef{Path: "original.go"}}
	if got := repositoryReviewEnrichedIssueSearchFinding(
		repoaudit.RepositoryState{}, base,
	); got.File.Path != "original.go" {
		t.Fatalf("unassociated finding=%#v", got)
	}
	base.RepositoryFindingID = "rrf_missing"
	if got := repositoryReviewEnrichedIssueSearchFinding(
		repoaudit.RepositoryState{}, base,
	); got.File.Path != "original.go" {
		t.Fatalf("missing aggregate finding=%#v", got)
	}
	base.RepositoryFindingID = "rrf_found"
	state := repoaudit.RepositoryState{RepositoryFindings: []repoaudit.RepositoryFinding{{
		ID: "rrf_found", MatchHints: repoaudit.MatchHints{Component: "scheduler"},
		PathSymbolHistory: []repoaudit.RepositoryFindingPathSymbol{
			{Path: ""}, {Path: "old.go"}, {Path: "old.go"}, {Path: "new.go"},
		},
	}}}
	enriched := repositoryReviewEnrichedIssueSearchFinding(state, base)
	if enriched.MatchHints.Component != "scheduler" || enriched.File.Path != "old.go new.go" {
		t.Fatalf("enriched=%#v", enriched)
	}

	_, err := decodeRepositoryReviewIssueCandidateRankings(map[string]any{
		"structured_valid": true,
		"structured": map[string]any{"rankings": []map[string]any{{
			"id": "1", "score": 50, "explanation": "candidate",
			"matching_anchors": []string{""},
		}}},
	}, []repositoryReviewIssueCandidate{{ID: "1", Number: 1, Title: "candidate"}})
	if err == nil {
		t.Fatal("blank ranking anchor was accepted")
	}
}

func TestRepositoryReviewCandidateRouteFailsClosedForMissingCurrentProfile(t *testing.T) {
	loop := repositoryReviewRankingTestLoop(t, &repositoryReviewRankingProvider{})
	store, _, finding, automation := repositoryReviewIssueLinkTestFixture(
		t, loop.GetConfig().WorkspacePath(), "owner/repo",
	)
	profile, err := store.CreateProfile(t.Context(), repoaudit.RepositoryReviewProfile{
		ID: "rrpf_kttutlpoaklekkcrod5fqpz3qw", Name: "Soon missing",
		ReviewFocus: "Find concrete bugs.",
		ScopePolicy: repoaudit.RepositoryReviewScopePolicy{
			CodeTypes: []repoaudit.RepositoryReviewCodeType{repoaudit.RepositoryReviewCodeTypeCode},
		},
		ReviewerModel: "issue-writer", AutoContinue: true, MaxFilesPerRun: 12,
		MaxContentBytes: 64 << 10, MaxParallelChildren: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	materialized, err := repoaudit.MaterializeRepositoryReviewAutomation(profile, automation)
	if err != nil {
		t.Fatal(err)
	}
	automation, err = store.UpdateAutomation(
		t.Context(), automation.ID, automation.Version,
		func(candidate *repoaudit.RepositoryReviewAutomation) error {
			*candidate = materialized
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(
		loop.GetConfig().WorkspacePath(), "repository_reviews", "profile_"+profile.ID+".json",
	)
	if err := os.Remove(profilePath); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		repositoryReviewPublicationRoute+"automations/"+automation.ID+"/findings/"+
			finding.ID+"/issue-link/candidates",
		strings.NewReader(`{"expected_version":`+strconv.FormatInt(finding.Version, 10)+`}`),
	)
	response := httptest.NewRecorder()
	newRepositoryReviewPublicationHandler(loop).ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable ||
		!strings.Contains(response.Body.String(), `"code":"issue_ranking_unavailable"`) {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewCurrentIssueWriterAutomationProfileBranches(t *testing.T) {
	store := repoaudit.NewStore(t.TempDir())
	automation := repoaudit.RepositoryReviewAutomation{IssueWriterModel: "unchanged"}
	current, err := repositoryReviewCurrentIssueWriterAutomation(t.Context(), store, automation)
	if err != nil || current.IssueWriterModel != "unchanged" {
		t.Fatalf("unprofiled current=%#v err=%v", current, err)
	}
	if _, lookupErr := repositoryReviewCurrentIssueWriterAutomation(
		t.Context(), store, repoaudit.RepositoryReviewAutomation{ProfileID: "rrpf_missing"},
	); lookupErr == nil {
		t.Fatal("missing current profile was accepted")
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, lookupErr := repositoryReviewCurrentIssueWriterAutomation(
		canceled, store, repoaudit.RepositoryReviewAutomation{ProfileID: "rrpf_missing"},
	); !errors.Is(lookupErr, context.Canceled) {
		t.Fatalf("canceled profile lookup error=%v", lookupErr)
	}
	profile, err := store.CreateProfile(t.Context(), repoaudit.RepositoryReviewProfile{
		ID: "rrpf_gateway_current", Name: "Current", ReviewFocus: "Find concrete bugs.",
		ScopePolicy: repoaudit.RepositoryReviewScopePolicy{
			CodeTypes: []repoaudit.RepositoryReviewCodeType{repoaudit.RepositoryReviewCodeTypeCode},
		},
		ReviewerModel: "review-a", AutoContinue: true, MaxFilesPerRun: 12,
		MaxContentBytes: 64 << 10, MaxParallelChildren: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err = repositoryReviewCurrentIssueWriterAutomation(
		t.Context(), store, repoaudit.RepositoryReviewAutomation{
			ProfileID: profile.ID, Ref: "main", EffectiveAccountRef: "old-account",
		},
	)
	if err != nil || current.ProfileVersion != profile.Version ||
		current.IssueWriterModel != "review-a" || current.EffectiveAccountRef != "" {
		t.Fatalf("profile current=%#v err=%v", current, err)
	}
	if _, err := repositoryReviewCurrentIssueWriterAutomation(
		t.Context(), store, repoaudit.RepositoryReviewAutomation{
			ProfileID: profile.ID, Ref: "bad branch value",
		},
	); err == nil {
		t.Fatal("invalid automation materialization was accepted")
	}
}
