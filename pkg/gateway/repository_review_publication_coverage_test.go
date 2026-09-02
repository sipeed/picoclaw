package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/repoaudit"
	"github.com/sipeed/picoclaw/pkg/reviews"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/workflows"
)

func TestRepositoryReviewPublicationRouteLifecycle(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Tools.MCP.Enabled = false
	messageBus := bus.NewMessageBus()
	loop := agent.NewAgentLoop(cfg, messageBus, &startupBlockedProvider{reason: "not used"})
	t.Cleanup(func() {
		loop.Stop()
		messageBus.Close()
		loop.Close()
	})

	manager, err := channels.NewManager(cfg, messageBus, nil)
	if err != nil {
		t.Fatal(err)
	}
	healthServer := health.NewServer("127.0.0.1", 0, "publication-token")
	manager.SetupHTTPServer("127.0.0.1:0", healthServer)
	running := &services{
		ChannelManager: manager,
		HealthServer:   healthServer,
		authToken:      "publication-token",
	}
	if err := prepareRepositoryReviewPublicationRoute(running, loop); err != nil {
		t.Fatal(err)
	}
	if running.repositoryReviewPublicationHandler == nil || running.repositoryReviewPublicationRelease == nil {
		t.Fatal("publication route was not installed")
	}
	if err := prepareRepositoryReviewPublicationRoute(running, loop); err != nil {
		t.Fatalf("refresh installed route: %v", err)
	}
	running.repositoryReviewPublicationHandler = nil
	if err := prepareRepositoryReviewPublicationRoute(running, loop); err == nil {
		t.Fatal("accepted an installed route without handler state")
	}
	releaseRepositoryReviewPublicationRoute(running)
	if running.repositoryReviewPublicationRelease != nil || running.repositoryReviewPublicationHandler != nil {
		t.Fatal("publication route state survived release")
	}
	releaseRepositoryReviewPublicationRoute(running)
	releaseRepositoryReviewPublicationRoute(nil)
}

func TestRepositoryReviewPublicationRouteRejectsUnsafeRuntime(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	messageBus := bus.NewMessageBus()
	loop := agent.NewAgentLoop(cfg, messageBus, &startupBlockedProvider{reason: "not used"})
	t.Cleanup(func() {
		loop.Stop()
		messageBus.Close()
		loop.Close()
	})
	manager, err := channels.NewManager(cfg, messageBus, nil)
	if err != nil {
		t.Fatal(err)
	}
	manager.SetupHTTPServer("127.0.0.1:0", health.NewServer("127.0.0.1", 0, "other"))

	for _, running := range []*services{
		nil,
		{},
		{ChannelManager: manager},
		{ChannelManager: manager, HealthServer: health.NewServer("127.0.0.1", 0, "other"), authToken: "wrong"},
	} {
		if routeErr := prepareRepositoryReviewPublicationRoute(running, loop); routeErr == nil {
			t.Fatalf("prepareRepositoryReviewPublicationRoute(%#v) succeeded", running)
		}
	}
	if nilLoopErr := prepareRepositoryReviewPublicationRoute(
		&services{ChannelManager: manager},
		nil,
	); nilLoopErr == nil {
		t.Fatal("accepted a nil agent loop")
	}

	release, err := manager.RegisterHTTPRoute(repositoryReviewPublicationRoute, http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	running := &services{
		ChannelManager: manager,
		HealthServer:   health.NewServer("127.0.0.1", 0, "token"),
		authToken:      "token",
	}
	if err := prepareRepositoryReviewPublicationRoute(running, loop); err == nil {
		t.Fatal("duplicate route registration succeeded")
	}
}

func TestRepositoryReviewPublicationHandlerRejectsMalformedRequests(t *testing.T) {
	handler := newRepositoryReviewPublicationHandler(nil)
	validPath := repositoryReviewPublicationRoute + "rrp_missing/issue-drafts/rid_missing/publish"
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		status int
		code   string
	}{
		{
			name:   "route",
			method: http.MethodGet,
			path:   validPath,
			body:   `{}`,
			status: http.StatusNotFound,
			code:   "not_found",
		},
		{
			name:   "syntax",
			method: http.MethodPost,
			path:   validPath,
			body:   `{`,
			status: http.StatusBadRequest,
			code:   "invalid_request",
		},
		{
			name:   "version",
			method: http.MethodPost,
			path:   validPath,
			body:   `{"expected_version":0}`,
			status: http.StatusBadRequest,
			code:   "invalid_request",
		},
		{
			name:   "unknown",
			method: http.MethodPost,
			path:   validPath,
			body:   `{"expected_version":1,"extra":true}`,
			status: http.StatusBadRequest,
			code:   "invalid_request",
		},
		{
			name:   "trailing",
			method: http.MethodPost,
			path:   validPath,
			body:   `{"expected_version":1}{}`,
			status: http.StatusBadRequest,
			code:   "invalid_request",
		},
		{
			name:   "runtime",
			method: http.MethodPost,
			path:   validPath,
			body:   `{"expected_version":1}`,
			status: http.StatusServiceUnavailable,
			code:   "publication_unavailable",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			if response.Header().Get("Cache-Control") != "no-store" ||
				response.Header().Get("X-Content-Type-Options") != "nosniff" ||
				response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
				t.Fatalf("response headers = %#v", response.Header())
			}
		})
	}
}

func TestRepositoryReviewPublicationHandlerReportsMissingLedger(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	messageBus := bus.NewMessageBus()
	loop := agent.NewAgentLoop(cfg, messageBus, &startupBlockedProvider{reason: "not used"})
	t.Cleanup(func() {
		loop.Stop()
		messageBus.Close()
		loop.Close()
	})
	handler := newRepositoryReviewPublicationHandler(loop)
	request := httptest.NewRequest(
		http.MethodPost,
		repositoryReviewPublicationRoute+"rrp_missing/issue-drafts/rid_missing/publish",
		strings.NewReader(`{"expected_version":1}`),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), `"code":"not_found"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewPublicationHandlerRejectsMissingToolRuntime(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Tools.MCP.Enabled = false
	messageBus := bus.NewMessageBus()
	loop := agent.NewAgentLoop(cfg, messageBus, &startupBlockedProvider{reason: "not used"})
	t.Cleanup(func() {
		loop.Stop()
		messageBus.Close()
		loop.Close()
	})
	store, state, draft := repositoryReviewPublicationTestDraft(t, cfg.WorkspacePath(), "owner/repo")
	_ = store
	defaultAgent := loop.GetRegistry().GetDefaultAgent()
	toolRegistry := defaultAgent.Tools
	defaultAgent.Tools = nil
	t.Cleanup(func() { defaultAgent.Tools = toolRegistry })

	request := httptest.NewRequest(
		http.MethodPost,
		repositoryReviewPublicationRoute+state.ID+"/issue-drafts/"+draft.ID+"/publish",
		strings.NewReader(`{"expected_version":1}`),
	)
	response := httptest.NewRecorder()
	newRepositoryReviewPublicationHandler(loop).ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable ||
		!strings.Contains(response.Body.String(), `"code":"publication_unavailable"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewPublicationHandlerReportsClaimPersistenceFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can bypass directory permission checks")
	}
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Tools.MCP.Enabled = false
	messageBus := bus.NewMessageBus()
	loop := agent.NewAgentLoop(cfg, messageBus, &startupBlockedProvider{reason: "not used"})
	t.Cleanup(func() {
		loop.Stop()
		messageBus.Close()
		loop.Close()
	})
	store, state, draft := repositoryReviewPublicationTestDraft(t, cfg.WorkspacePath(), "owner/repo")
	root := filepath.Join(cfg.WorkspacePath(), "repository_reviews")
	if err := os.Chmod(root, 0o500); err != nil {
		t.Fatal(err)
	}
	restore := func() { _ = os.Chmod(root, 0o700) }
	t.Cleanup(restore)

	request := httptest.NewRequest(
		http.MethodPost,
		repositoryReviewPublicationRoute+state.ID+"/issue-drafts/"+draft.ID+"/publish",
		strings.NewReader(`{"expected_version":`+strconv.FormatInt(draft.Version, 10)+`}`),
	)
	response := httptest.NewRecorder()
	newRepositoryReviewPublicationHandler(loop).ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable ||
		!strings.Contains(response.Body.String(), `"code":"publication_unavailable"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	restore()
	current, found, err := store.GetByID(state.ID)
	if err != nil || !found || len(current.IssueDrafts) != 1 ||
		current.IssueDrafts[0].State != repoaudit.IssueDraftEditing {
		t.Fatalf("claim failure state=%#v found=%v err=%v", current.IssueDrafts, found, err)
	}
}

func TestRepositoryReviewPublicationHandlerCoversDurableDraftStates(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		prepare    func(t *testing.T, store repoaudit.Store, state repoaudit.RepositoryState, draft repoaudit.IssueDraft) (string, int64)
		status     int
	}{
		{
			name:       "missing draft",
			repository: "owner/repo",
			prepare: func(_ *testing.T, _ repoaudit.Store, _ repoaudit.RepositoryState, draft repoaudit.IssueDraft) (string, int64) {
				return "rid_missing", draft.Version
			},
			status: http.StatusNotFound,
		},
		{
			name:       "stale draft",
			repository: "owner/repo",
			prepare: func(_ *testing.T, _ repoaudit.Store, _ repoaudit.RepositoryState, draft repoaudit.IssueDraft) (string, int64) {
				return draft.ID, draft.Version + 1
			},
			status: http.StatusConflict,
		},
		{
			name:       "posted draft",
			repository: "owner/repo",
			prepare: func(t *testing.T, store repoaudit.Store, _ repoaudit.RepositoryState, draft repoaudit.IssueDraft) (string, int64) {
				_, claimed, didClaim, err := store.ClaimIssueDraftPublication("owner/repo", draft.ID, draft.Version)
				if err != nil || !didClaim {
					t.Fatalf("claim draft: claimed=%v err=%v", didClaim, err)
				}
				_, posted, err := store.SetIssueDraftPublication(
					"owner/repo", draft.ID, claimed.Version, repoaudit.IssueDraftPosted,
					"12", "https://github.com/owner/repo/issues/12",
				)
				if err != nil {
					t.Fatal(err)
				}
				return posted.ID, posted.Version
			},
			status: http.StatusOK,
		},
		{
			name:       "invalid GitHub identity",
			repository: "/tmp/repo",
			prepare: func(_ *testing.T, _ repoaudit.Store, _ repoaudit.RepositoryState, draft repoaudit.IssueDraft) (string, int64) {
				return draft.ID, draft.Version
			},
			status: http.StatusBadRequest,
		},
		{
			name:       "provider unavailable",
			repository: "owner/repo",
			prepare: func(_ *testing.T, _ repoaudit.Store, _ repoaudit.RepositoryState, draft repoaudit.IssueDraft) (string, int64) {
				return draft.ID, draft.Version
			},
			status: http.StatusServiceUnavailable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Agents.Defaults.Workspace = t.TempDir()
			cfg.Tools.MCP.Enabled = false
			messageBus := bus.NewMessageBus()
			loop := agent.NewAgentLoop(cfg, messageBus, &startupBlockedProvider{reason: "not used"})
			t.Cleanup(func() {
				loop.Stop()
				messageBus.Close()
				loop.Close()
			})

			store, state, draft := repositoryReviewPublicationTestDraft(t, cfg.WorkspacePath(), test.repository)
			draftID, expectedVersion := test.prepare(t, store, state, draft)
			request := httptest.NewRequest(
				http.MethodPost,
				repositoryReviewPublicationRoute+state.ID+"/issue-drafts/"+draftID+"/publish",
				strings.NewReader(`{"expected_version":`+strconv.FormatInt(expectedVersion, 10)+`}`),
			)
			response := httptest.NewRecorder()
			newRepositoryReviewPublicationHandler(loop).ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("response = %d %s, want %d", response.Code, response.Body.String(), test.status)
			}
		})
	}
}

func TestRepositoryReviewPublicationHandlerRejectsNoncanonicalLegacyConflict(t *testing.T) {
	workspace := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	messageBus := bus.NewMessageBus()
	loop := agent.NewAgentLoop(cfg, messageBus, &startupBlockedProvider{reason: "not used"})
	t.Cleanup(func() {
		loop.Stop()
		messageBus.Close()
		loop.Close()
	})
	_, state, draft := repositoryReviewPublicationTestDraft(t, workspace, "owner/repo")

	root := filepath.Join(workspace, "repository_reviews")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var statePath string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") &&
			!strings.HasSuffix(entry.Name(), ".summary.json") {
			statePath = filepath.Join(root, entry.Name())
			break
		}
	}
	if statePath == "" {
		t.Fatal("authoritative repository-review state file was not found")
	}
	raw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	var persisted repoaudit.RepositoryState
	if unmarshalErr := json.Unmarshal(raw, &persisted); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	draft.State = repoaudit.IssueDraftPosted
	draft.ExternalID = "41"
	draft.ExternalURL = "https://github.com/owner/repo/issues/41"
	for index := range persisted.IssueDrafts {
		if persisted.IssueDrafts[index].ID == draft.ID {
			persisted.IssueDrafts[index] = draft
		}
	}
	for index := range persisted.Findings {
		persisted.Findings[index].Status = repoaudit.FindingPosted
	}
	newer := draft
	newer.ID = "rid_newer_legacy_conflict"
	newer.Canonical = false
	newer.ExternalID = "42"
	newer.ExternalURL = "https://github.com/owner/repo/issues/42"
	newer.CreatedAt = draft.CreatedAt.Add(1)
	newer.UpdatedAt = draft.UpdatedAt.Add(1)
	persisted.IssueDrafts = append(persisted.IssueDrafts, newer)
	raw, err = json.Marshal(persisted)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		repositoryReviewPublicationRoute+state.ID+"/issue-drafts/"+draft.ID+"/publish",
		strings.NewReader(`{"expected_version":`+strconv.FormatInt(draft.Version, 10)+`}`),
	)
	response := httptest.NewRecorder()
	newRepositoryReviewPublicationHandler(loop).ServeHTTP(response, request)
	if response.Code != http.StatusConflict ||
		!strings.Contains(response.Body.String(), `"code":"preview_not_canonical"`) {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
}

func repositoryReviewPublicationTestDraft(
	t *testing.T,
	workspace string,
	repository string,
) (repoaudit.Store, repoaudit.RepositoryState, repoaudit.IssueDraft) {
	t.Helper()
	store := repoaudit.NewStore(workspace)
	file := repoaudit.FileRef{
		Path: "service.go", BlobSHA: strings.Repeat("a", 40), SizeBytes: 10,
		Category: "code", Mode: "100644",
	}
	plan, err := store.Plan(t.Context(), repository, "commit-a", "inventory-a", []repoaudit.FileRef{file}, false)
	if err != nil {
		t.Fatal(err)
	}
	recorded, err := store.Record(t.Context(), repoaudit.RecordRequest{
		Plan: plan, RunID: "publication-run",
		Observations: []repoaudit.Observation{{
			Model: "review-a", ScopeFiles: []repoaudit.FileRef{file},
			Findings: []repoaudit.FindingCandidate{{
				Severity: "high", Title: "Lost update", File: file.Path,
				Evidence: "unfenced write", Impact: "data loss",
				Validation: repoaudit.Validation{Status: "confirmed", Summary: "reproduced"},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	mappedState := completeRepositoryReviewPublicationTestMapping(
		t,
		store,
		recorded.State,
		recorded.State.Findings[0].ID,
	)
	state, draft, err := store.PrepareIssue(repoaudit.IssueDraftRequest{
		Repository: repository, FindingIDs: []string{mappedState.Findings[0].ID},
		ExpectedVersion: mappedState.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	return store, state, draft
}

func completeRepositoryReviewPublicationTestMapping(
	t *testing.T,
	store repoaudit.Store,
	state repoaudit.RepositoryState,
	findingID string,
) repoaudit.RepositoryState {
	t.Helper()
	for index := range state.MappingJobs {
		job := state.MappingJobs[index]
		if job.ReviewFindingID != findingID {
			continue
		}
		claimedState, claimedJob, _, claimed, err := store.ClaimMappingJob(
			state.Repository,
			job.ID,
			repoaudit.RepositoryMappingModelSnapshot{},
		)
		if err != nil || !claimed {
			t.Fatalf("claim mapping job for finding %q: claimed=%v err=%v", findingID, claimed, err)
		}
		mappedState, _, err := store.CompleteMappingJob(
			claimedState.Repository,
			repoaudit.RepositoryMappingCompletion{
				JobID:                 claimedJob.ID,
				CreateMatchState:      repoaudit.RepositoryMatchNew,
				DefaultBranchVerified: true,
			},
		)
		if err != nil {
			t.Fatalf("complete mapping job for finding %q: %v", findingID, err)
		}
		return mappedState
	}
	t.Fatalf("mapping job for finding %q is missing", findingID)
	return repoaudit.RepositoryState{}
}

func repositoryReviewIssueLinkTestFixture(
	t *testing.T,
	workspace string,
	repository string,
) (repoaudit.Store, repoaudit.RepositoryState, repoaudit.Finding, repoaudit.RepositoryReviewAutomation) {
	t.Helper()
	store, state, draft := repositoryReviewPublicationTestDraft(t, workspace, repository)
	state, err := store.DeleteIssueDraft(state.Repository, draft.ID, draft.Version)
	if err != nil {
		t.Fatal(err)
	}
	finding := state.Findings[0]
	automation, err := store.CreateAutomation(t.Context(), repoaudit.RepositoryReviewAutomation{
		ID:   "rra_issue_link_fixture",
		Name: "Issue-link test", Repository: state.Repository,
		Target: "all", ReviewFocus: "Find bugs.", ReviewerModels: []string{"issue-writer"},
		IssueWriterModel: "issue-writer", AccountRef: "writer-account",
		EffectiveAccountRef: "writer-account", MaxFilesPerRun: 1, MaxContentBytes: 1024,
		MaxParallelChildren: 1, Status: repoaudit.RepositoryReviewAutomationIdle,
		RunIDs: []string{"publication-run"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return store, state, finding, automation
}

type repositoryReviewPublicationMCPManager struct {
	searchText   string
	createText   string
	searchErr    error
	createErr    error
	beforeReturn func(string)
}

type repositoryReviewRankingProvider struct {
	responses []string
	err       error
	calls     int
	models    []string
	tools     []int
}

func repositoryReviewRankingTestLoop(
	t *testing.T,
	provider *repositoryReviewRankingProvider,
) *agent.AgentLoop {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("ranker provider path=%q", r.URL.Path)
		}
		defer r.Body.Close()
		var request struct {
			Model string `json:"model"`
			Tools []any  `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode ranker request: %v", err)
		}
		provider.models = append(provider.models, request.Model)
		provider.tools = append(provider.tools, len(request.Tools))
		provider.calls++
		if provider.err != nil {
			http.Error(w, provider.err.Error(), http.StatusServiceUnavailable)
			return
		}
		response := ""
		if len(provider.responses) > 0 {
			response = provider.responses[min(provider.calls-1, len(provider.responses)-1)]
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{"content": response}, "finish_reason": "stop",
			}},
		}); err != nil {
			t.Fatalf("encode ranker response: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Agents.Defaults.AccountRef = "writer-account"
	cfg.Agents.Defaults.ModelName = "issue-writer"
	cfg.ModelAliases = []config.ModelAliasConfig{{Name: "issue-writer", Model: "writer-v1"}}
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "writer-account", Provider: "openai", Model: "writer-v1",
		APIBase: server.URL, APIKeys: config.SimpleSecureStrings("test-key"), Enabled: true,
	}}
	messageBus := bus.NewMessageBus()
	loop := agent.NewAgentLoop(cfg, messageBus, &startupBlockedProvider{reason: "bootstrap provider unused"})
	t.Cleanup(func() {
		loop.Stop()
		messageBus.Close()
		loop.Close()
	})
	return loop
}

func TestRepositoryReviewIssueCandidateRankingAcceptsOnlyGroundedUniqueResults(t *testing.T) {
	provider := &repositoryReviewRankingProvider{responses: []string{
		`{"rankings":[` +
			`{"id":"2","score":91,"explanation":"Same stable symbol and path."},` +
			`{"id":"1","score":73.5,"explanation":"Same failure mechanism."}` +
			`]}`,
	}}
	loop := repositoryReviewRankingTestLoop(t, provider)
	candidates := []repositoryReviewIssueCandidate{
		{ID: "1", Number: 1, Title: "First", body: "first body"},
		{ID: "2", Number: 2, Title: "Second", body: "second body"},
	}
	ranked, err := rankRepositoryReviewIssueCandidates(
		t.Context(), loop,
		repoaudit.RepositoryReviewAutomation{IssueWriterModel: "issue-writer"},
		repoaudit.Finding{Title: "Finding", Symbol: "Store.Save", File: repoaudit.FileRef{Path: "store.go"}},
		candidates,
		"writer-account",
	)
	if err != nil || len(ranked) != 2 || ranked[0].ID != "2" || ranked[0].Score != 91 ||
		ranked[1].ID != "1" || ranked[1].Explanation != "Same failure mechanism." {
		t.Fatalf("ranked=%#v err=%v", ranked, err)
	}
	if provider.calls != 1 || len(provider.tools) != 1 || provider.tools[0] != 0 {
		t.Fatalf("provider calls=%d models=%#v tools=%#v", provider.calls, provider.models, provider.tools)
	}
}

func TestRepositoryReviewIssueCandidateRankingRejectsUngroundedOrMalformedResults(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		provider  error
		wantError string
	}{
		{
			name: "unknown candidate",
			response: `{"rankings":[` +
				`{"id":"99","score":80,"explanation":"Not in the bounded input."}` +
				`]}`,
			wantError: "invalid structured output",
		},
		{
			name: "duplicate candidate",
			response: `{"rankings":[` +
				`{"id":"1","score":80,"explanation":"First."},` +
				`{"id":"1","score":70,"explanation":"Duplicate."}` +
				`]}`,
			wantError: "duplicate candidate",
		},
		{
			name: "score above parser bound",
			response: `{"rankings":[` +
				`{"id":"1","score":101,"explanation":"Out of range."}` +
				`]}`,
			wantError: "invalid structured output",
		},
		{name: "invalid JSON", response: `not-json`, wantError: "structured output invalid"},
		{
			name: "schema violation", response: `{"rankings":[{"id":"1","score":80,"explanation":""}]}`,
			wantError: "structured output invalid",
		},
		{name: "provider failure", provider: errors.New("ranker offline"), wantError: "API request failed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &repositoryReviewRankingProvider{
				responses: []string{test.response}, err: test.provider,
			}
			loop := repositoryReviewRankingTestLoop(t, provider)
			_, err := rankRepositoryReviewIssueCandidates(
				t.Context(), loop,
				repoaudit.RepositoryReviewAutomation{IssueWriterModel: "issue-writer"},
				repoaudit.Finding{Title: "Finding"},
				[]repositoryReviewIssueCandidate{{ID: "1", Number: 1, Title: "Candidate"}},
				"writer-account",
			)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("rank error=%v, want %q", err, test.wantError)
			}
		})
	}
}

func TestRepositoryReviewIssueCandidateRankingRejectsOversizedInputBeforeDispatch(t *testing.T) {
	candidates := make([]repositoryReviewIssueCandidate, 600)
	for index := range candidates {
		candidates[index] = repositoryReviewIssueCandidate{
			ID: strconv.Itoa(index + 1), Number: int64(index + 1),
			Title: "candidate", body: strings.Repeat("x", 2048),
		}
	}
	_, err := rankRepositoryReviewIssueCandidates(
		t.Context(), nil,
		repoaudit.RepositoryReviewAutomation{IssueWriterModel: "issue-writer"},
		repoaudit.Finding{Title: "Finding"}, candidates, "writer-account",
	)
	if err == nil || !strings.Contains(err.Error(), "exceeds its safe bound") {
		t.Fatalf("oversized rank input error=%v", err)
	}
}

func TestRepositoryReviewValidatedWriterAccountRequiresCompletePassiveSnapshot(t *testing.T) {
	if _, err := repositoryReviewValidatedIssueWriterAccount(
		t.Context(), nil, repoaudit.RepositoryReviewAutomation{},
	); err == nil {
		t.Fatal("nil writer runtime was accepted")
	}

	loop := repositoryReviewRankingTestLoop(t, &repositoryReviewRankingProvider{})
	if _, err := repositoryReviewValidatedIssueWriterAccount(
		t.Context(), loop,
		repoaudit.RepositoryReviewAutomation{EffectiveAccountRef: "writer-account"},
	); err == nil {
		t.Fatal("writer snapshot without a model was accepted")
	}
	emptyConfig := config.DefaultConfig()
	emptyConfig.Agents.Defaults.Workspace = t.TempDir()
	emptyConfig.Agents.Defaults.AccountRef = ""
	emptyBus := bus.NewMessageBus()
	emptyLoop := agent.NewAgentLoop(
		emptyConfig, emptyBus, &startupBlockedProvider{reason: "not used"},
	)
	t.Cleanup(func() {
		emptyLoop.Stop()
		emptyBus.Close()
		emptyLoop.Close()
	})
	if _, err := repositoryReviewValidatedIssueWriterAccount(
		t.Context(), emptyLoop,
		repoaudit.RepositoryReviewAutomation{IssueWriterModel: "issue-writer"},
	); err == nil {
		t.Fatal("writer snapshot without an account was accepted")
	}
	account, err := repositoryReviewValidatedIssueWriterAccount(
		t.Context(), loop,
		repoaudit.RepositoryReviewAutomation{
			IssueWriterModel: "issue-writer", EffectiveAccountRef: "writer-account",
		},
	)
	if err != nil || account != "writer-account" {
		t.Fatalf("validated account=%q err=%v", account, err)
	}
}

func TestRepositoryReviewProtectedIssueLinkRefetchesAndPersistsValidatedIssue(t *testing.T) {
	workspace := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	messageBus := bus.NewMessageBus()
	loop := agent.NewAgentLoop(cfg, messageBus, &startupBlockedProvider{reason: "not used"})
	t.Cleanup(func() {
		loop.Stop()
		messageBus.Close()
		loop.Close()
	})
	manager := &repositoryReviewPublicationMCPManager{searchText: `{
		"id":99,"number":12,"title":"Existing lost-update report",
		"body":"The same write loses data.","state":"closed",
		"html_url":"https://github.com/owner/repo/issues/12",
		"labels":[{"name":"bug"}]
	}`}
	loop.RegisterTool(tools.NewMCPTool(manager, reviews.DefaultGitHubMCPServer, &sdkmcp.Tool{
		Name:        reviews.GitHubIssueReadTool,
		InputSchema: map[string]any{"type": "object", "additionalProperties": true},
	}))
	store, state, draft := repositoryReviewPublicationTestDraft(t, workspace, "owner/repo")
	state, err := store.DeleteIssueDraft(state.Repository, draft.ID, draft.Version)
	if err != nil {
		t.Fatal(err)
	}
	finding := state.Findings[0]
	automation, err := store.CreateAutomation(t.Context(), repoaudit.RepositoryReviewAutomation{
		ID: "rra_link_test", Name: "Link test", Repository: state.Repository,
		Target: "all", ReviewFocus: "Find bugs.", ReviewerModels: []string{"review"},
		IssueWriterModel: "writer", MaxFilesPerRun: 1, MaxContentBytes: 1024,
		MaxParallelChildren: 1, Status: repoaudit.RepositoryReviewAutomationIdle,
		RunIDs: []string{"publication-run"},
	})
	if err != nil {
		t.Fatal(err)
	}
	crossRepository := httptest.NewRequest(
		http.MethodPost,
		repositoryReviewPublicationRoute+"automations/"+automation.ID+"/findings/"+
			finding.ID+"/issue-link",
		strings.NewReader(`{"issue_url":"https://github.com/other/repo/issues/12","expected_version":`+
			strconv.FormatInt(finding.Version, 10)+`,"confirmed":true}`),
	)
	crossResponse := httptest.NewRecorder()
	newRepositoryReviewPublicationHandler(loop).ServeHTTP(crossResponse, crossRepository)
	if crossResponse.Code != http.StatusBadRequest {
		t.Fatalf("cross-repository link response=%d %s", crossResponse.Code, crossResponse.Body.String())
	}
	request := httptest.NewRequest(
		http.MethodPost,
		repositoryReviewPublicationRoute+"automations/"+automation.ID+"/findings/"+
			finding.ID+"/issue-link",
		strings.NewReader(`{"issue_url":"https://github.com/owner/repo/issues/12","expected_version":`+
			strconv.FormatInt(finding.Version, 10)+`,"confirmed":true}`),
	)
	response := httptest.NewRecorder()
	newRepositoryReviewPublicationHandler(loop).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"origin":"linked"`) ||
		!strings.Contains(response.Body.String(), `"external_url":"https://github.com/owner/repo/issues/12"`) {
		t.Fatalf("link response=%d %s", response.Code, response.Body.String())
	}
	persisted, found, err := store.Get(state.Repository)
	if err != nil || !found || persisted.Findings[0].Status != repoaudit.FindingPosted ||
		len(persisted.IssueDrafts) != 1 ||
		persisted.IssueDrafts[0].Origin != repoaudit.IssueDraftOriginLinked {
		t.Fatalf("persisted link=%#v found=%v err=%v", persisted, found, err)
	}
	manager.searchText = `{
		"id":100,"number":13,"title":"Replacement lost-update report",
		"body":"An even closer existing report.","state":"open",
		"html_url":"https://github.com/owner/repo/issues/13",
		"labels":[{"name":"bug"}]
	}`
	replaceRequest := httptest.NewRequest(
		http.MethodPost,
		repositoryReviewPublicationRoute+"automations/"+automation.ID+"/findings/"+
			finding.ID+"/issue-link",
		strings.NewReader(`{"issue_url":"https://github.com/owner/repo/issues/13","expected_version":`+
			strconv.FormatInt(persisted.Findings[0].Version, 10)+`,"confirmed":true,"replace":true}`),
	)
	replaceResponse := httptest.NewRecorder()
	newRepositoryReviewPublicationHandler(loop).ServeHTTP(replaceResponse, replaceRequest)
	replaced, found, err := store.Get(state.Repository)
	if replaceResponse.Code != http.StatusOK || err != nil || !found ||
		len(replaced.IssueDrafts) != 1 || replaced.IssueDrafts[0].ExternalID != "100" ||
		replaced.IssueDrafts[0].ExternalURL != "https://github.com/owner/repo/issues/13" {
		t.Fatalf(
			"replace response=%d %s state=%#v found=%v err=%v",
			replaceResponse.Code, replaceResponse.Body.String(), replaced, found, err,
		)
	}
	unlinkRequest := httptest.NewRequest(
		http.MethodDelete,
		repositoryReviewPublicationRoute+"automations/"+automation.ID+"/findings/"+
			finding.ID+"/issue-link",
		strings.NewReader(`{"expected_version":`+
			strconv.FormatInt(replaced.Findings[0].Version, 10)+`,"confirmed":true}`),
	)
	unlinkResponse := httptest.NewRecorder()
	newRepositoryReviewPublicationHandler(loop).ServeHTTP(unlinkResponse, unlinkRequest)
	unlinked, found, err := store.Get(state.Repository)
	if unlinkResponse.Code != http.StatusOK || err != nil || !found ||
		unlinked.Findings[0].Status != repoaudit.FindingOpen ||
		unlinked.Findings[0].IssueDraftID != "" || len(unlinked.IssueDrafts) != 0 {
		t.Fatalf(
			"unlink response=%d %s state=%#v found=%v err=%v",
			unlinkResponse.Code, unlinkResponse.Body.String(), unlinked, found, err,
		)
	}
}

func TestRepositoryReviewIssueLinkAndUnlinkRejectInvalidOrStaleRequests(t *testing.T) {
	workspace := t.TempDir()
	store, state, finding, automation := repositoryReviewIssueLinkTestFixture(
		t, workspace, "owner/repo",
	)
	handler := newRepositoryReviewPublicationHandler(nil)

	for _, test := range []struct {
		name   string
		body   string
		status int
		code   string
	}{
		{name: "malformed", body: `{`, status: http.StatusBadRequest, code: "invalid_request"},
		{
			name:   "confirmation required",
			body:   `{"issue_url":"https://github.com/owner/repo/issues/12","expected_version":1}`,
			status: http.StatusBadRequest, code: "invalid_request",
		},
		{
			name:   "stale",
			body:   `{"issue_url":"https://github.com/owner/repo/issues/12","expected_version":99,"confirmed":true}`,
			status: http.StatusConflict, code: "stale_repository_review",
		},
		{
			name: "invalid URL",
			body: `{"issue_url":"http://github.com/owner/repo/issues/12","expected_version":` +
				strconv.FormatInt(finding.Version, 10) + `,"confirmed":true}`,
			status: http.StatusBadRequest, code: "invalid_issue_url",
		},
		{
			name: "tool runtime unavailable",
			body: `{"issue_url":"https://github.com/owner/repo/issues/12","expected_version":` +
				strconv.FormatInt(finding.Version, 10) + `,"confirmed":true}`,
			status: http.StatusServiceUnavailable, code: "issue_link_unavailable",
		},
	} {
		t.Run("link "+test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			handler.serveRepositoryReviewIssueLink(
				response, request, nil, store, automation, state, finding,
			)
			if response.Code != test.status ||
				!strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response=%d %s", response.Code, response.Body.String())
			}
		})
	}

	for _, test := range []struct {
		name   string
		body   string
		status int
		code   string
	}{
		{name: "malformed", body: `{`, status: http.StatusBadRequest, code: "invalid_request"},
		{name: "confirmation required", body: `{"expected_version":1}`, status: http.StatusBadRequest, code: "invalid_request"},
		{name: "stale", body: `{"expected_version":99,"confirmed":true}`, status: http.StatusConflict, code: "stale_repository_review"},
		{
			name:   "not linked",
			body:   `{"expected_version":` + strconv.FormatInt(finding.Version, 10) + `,"confirmed":true}`,
			status: http.StatusConflict, code: "stale_repository_review",
		},
	} {
		t.Run("unlink "+test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodDelete, "/", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			handler.serveRepositoryReviewIssueUnlink(
				response, request, store, automation, state, finding,
			)
			if response.Code != test.status ||
				!strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response=%d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRepositoryReviewIssueLinkReportsProviderAndPersistenceFailuresSafely(t *testing.T) {
	for _, test := range []struct {
		name       string
		issueText  string
		issueErr   error
		poisonRoot bool
		status     int
		code       string
	}{
		{
			name: "provider failure", issueErr: errors.New("GitHub unavailable"),
			status: http.StatusServiceUnavailable, code: "issue_link_unavailable",
		},
		{
			name: "invalid response",
			issueText: `{"id":99,"number":12,"title":"Wrong number",` +
				`"html_url":"https://github.com/owner/repo/issues/13"}`,
			status: http.StatusBadGateway, code: "invalid_gateway_response",
		},
		{
			name: "persistence failure",
			issueText: `{"id":99,"number":12,"title":"Existing issue",` +
				`"html_url":"https://github.com/owner/repo/issues/12"}`,
			poisonRoot: true, status: http.StatusServiceUnavailable, code: "publication_unavailable",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			loop := repositoryReviewRankingTestLoop(t, &repositoryReviewRankingProvider{})
			workspace := loop.GetConfig().WorkspacePath()
			manager := &repositoryReviewPublicationMCPManager{
				searchText: test.issueText, searchErr: test.issueErr,
			}
			if test.poisonRoot {
				manager.beforeReturn = func(toolName string) {
					if toolName != reviews.GitHubIssueReadTool {
						return
					}
					root := filepath.Join(workspace, "repository_reviews")
					if err := os.RemoveAll(root); err != nil {
						t.Errorf("remove store root: %v", err)
						return
					}
					if err := os.WriteFile(root, []byte("not a directory"), 0o600); err != nil {
						t.Errorf("replace store root: %v", err)
					}
				}
			}
			loop.RegisterTool(tools.NewMCPTool(manager, reviews.DefaultGitHubMCPServer, &sdkmcp.Tool{
				Name:        reviews.GitHubIssueReadTool,
				InputSchema: map[string]any{"type": "object", "additionalProperties": true},
			}))
			store, state, finding, automation := repositoryReviewIssueLinkTestFixture(
				t, workspace, "owner/repo",
			)
			request := httptest.NewRequest(
				http.MethodPost, "/",
				strings.NewReader(`{"issue_url":"https://github.com/owner/repo/issues/12",`+
					`"expected_version":`+strconv.FormatInt(finding.Version, 10)+`,"confirmed":true}`),
			)
			response := httptest.NewRecorder()
			newRepositoryReviewPublicationHandler(loop).serveRepositoryReviewIssueLink(
				response, request, loop, store, automation, state, finding,
			)
			if response.Code != test.status ||
				!strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response=%d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRepositoryReviewIssueLinkReportsConcurrentFindingMutation(t *testing.T) {
	loop := repositoryReviewRankingTestLoop(t, &repositoryReviewRankingProvider{})
	manager := &repositoryReviewPublicationMCPManager{searchText: `{
		"id":99,"number":12,"title":"Existing issue",
		"html_url":"https://github.com/owner/repo/issues/12"
	}`}
	loop.RegisterTool(tools.NewMCPTool(manager, reviews.DefaultGitHubMCPServer, &sdkmcp.Tool{
		Name:        reviews.GitHubIssueReadTool,
		InputSchema: map[string]any{"type": "object", "additionalProperties": true},
	}))
	store, state, finding, automation := repositoryReviewIssueLinkTestFixture(
		t, loop.GetConfig().WorkspacePath(), "owner/repo",
	)
	if _, err := store.SetFindingStatus(
		state.Repository, finding.ID, repoaudit.FindingDismissed, state.Version,
	); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(
		http.MethodPost, "/",
		strings.NewReader(`{"issue_url":"https://github.com/owner/repo/issues/12",`+
			`"expected_version":`+strconv.FormatInt(finding.Version, 10)+`,"confirmed":true}`),
	)
	response := httptest.NewRecorder()
	newRepositoryReviewPublicationHandler(loop).serveRepositoryReviewIssueLink(
		response, request, loop, store, automation, state, finding,
	)
	if response.Code != http.StatusConflict ||
		!strings.Contains(response.Body.String(), `"code":"stale_repository_review"`) {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
}

func TestRepositoryReviewProtectedIssueCandidateDiscoverySearchesThenRanks(t *testing.T) {
	rankingProvider := &repositoryReviewRankingProvider{responses: []string{
		`{"rankings":[{"id":"99","score":94,"explanation":"Same write and data-loss mechanism."}]}`,
	}}
	loop := repositoryReviewRankingTestLoop(t, rankingProvider)
	workspace := loop.GetConfig().WorkspacePath()
	manager := &repositoryReviewPublicationMCPManager{searchText: `{"items":[{
		"id":99,"number":12,"title":"Existing lost-update report",
		"body":"The same write loses data.","state":"closed",
		"html_url":"https://github.com/owner/repo/issues/12",
		"labels":[{"name":"bug"}]
	}]}`}
	loop.RegisterTool(tools.NewMCPTool(manager, reviews.DefaultGitHubMCPServer, &sdkmcp.Tool{
		Name:        reviews.GitHubSearchIssuesTool,
		InputSchema: map[string]any{"type": "object", "additionalProperties": true},
	}))
	store, state, draft := repositoryReviewPublicationTestDraft(t, workspace, "owner/repo")
	state, err := store.DeleteIssueDraft(state.Repository, draft.ID, draft.Version)
	if err != nil {
		t.Fatal(err)
	}
	finding := state.Findings[0]
	automation, err := store.CreateAutomation(t.Context(), repoaudit.RepositoryReviewAutomation{
		ID: "rra_candidate_test", Name: "Candidate test", Repository: state.Repository,
		Target: "all", ReviewFocus: "Find bugs.", ReviewerModels: []string{"issue-writer"},
		IssueWriterModel: "issue-writer", AccountRef: "writer-account",
		EffectiveAccountRef: "writer-account", MaxFilesPerRun: 1, MaxContentBytes: 1024,
		MaxParallelChildren: 1, Status: repoaudit.RepositoryReviewAutomationIdle,
		RunIDs: []string{"publication-run"},
	})
	if err != nil {
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
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"score":94`) ||
		!strings.Contains(response.Body.String(), `"generator_model":"issue-writer"`) ||
		!strings.Contains(response.Body.String(), `"generator_account":"writer-account"`) {
		t.Fatalf("candidate response=%d %s", response.Code, response.Body.String())
	}
	if rankingProvider.calls != 1 || rankingProvider.tools[0] != 0 {
		t.Fatalf("ranking calls=%d tools=%#v", rankingProvider.calls, rankingProvider.tools)
	}
}

func TestRepositoryReviewAutomationOperationRejectsMissingOrUnsafeState(t *testing.T) {
	t.Run("runtime unavailable", func(t *testing.T) {
		request := httptest.NewRequest(
			http.MethodPost,
			repositoryReviewPublicationRoute+"automations/missing/findings/missing/issue-link",
			strings.NewReader(`{}`),
		)
		response := httptest.NewRecorder()
		newRepositoryReviewPublicationHandler(nil).ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable ||
			!strings.Contains(response.Body.String(), `"code":"repository_review_unavailable"`) {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("automation missing", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Agents.Defaults.Workspace = t.TempDir()
		messageBus := bus.NewMessageBus()
		loop := agent.NewAgentLoop(cfg, messageBus, &startupBlockedProvider{reason: "not used"})
		t.Cleanup(func() {
			loop.Stop()
			messageBus.Close()
			loop.Close()
		})
		request := httptest.NewRequest(
			http.MethodPost,
			repositoryReviewPublicationRoute+"automations/missing/findings/missing/issue-link",
			strings.NewReader(`{}`),
		)
		response := httptest.NewRecorder()
		newRepositoryReviewPublicationHandler(loop).ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("automation ledger missing", func(t *testing.T) {
		loop := repositoryReviewRankingTestLoop(t, &repositoryReviewRankingProvider{})
		store := repoaudit.NewStore(loop.GetConfig().WorkspacePath())
		automation, err := store.CreateAutomation(t.Context(), repoaudit.RepositoryReviewAutomation{
			ID: "rra_missing_ledger", Name: "Missing ledger", Repository: "owner/missing",
			Target: "all", ReviewFocus: "Find bugs.", ReviewerModels: []string{"issue-writer"},
			IssueWriterModel: "issue-writer", AccountRef: "writer-account",
			EffectiveAccountRef: "writer-account", MaxFilesPerRun: 1, MaxContentBytes: 1024,
			MaxParallelChildren: 1, Status: repoaudit.RepositoryReviewAutomationIdle,
		})
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(
			http.MethodPost,
			repositoryReviewPublicationRoute+"automations/"+automation.ID+
				"/findings/missing/issue-link",
			strings.NewReader(`{}`),
		)
		response := httptest.NewRecorder()
		newRepositoryReviewPublicationHandler(loop).ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("finding missing", func(t *testing.T) {
		loop := repositoryReviewRankingTestLoop(t, &repositoryReviewRankingProvider{})
		_, _, _, automation := repositoryReviewIssueLinkTestFixture(
			t, loop.GetConfig().WorkspacePath(), "owner/repo",
		)
		request := httptest.NewRequest(
			http.MethodPost,
			repositoryReviewPublicationRoute+"automations/"+automation.ID+
				"/findings/missing/issue-link",
			strings.NewReader(`{}`),
		)
		response := httptest.NewRecorder()
		newRepositoryReviewPublicationHandler(loop).ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("repository not linkable", func(t *testing.T) {
		loop := repositoryReviewRankingTestLoop(t, &repositoryReviewRankingProvider{})
		_, _, finding, automation := repositoryReviewIssueLinkTestFixture(
			t, loop.GetConfig().WorkspacePath(), "/tmp/local-repository",
		)
		request := httptest.NewRequest(
			http.MethodPost,
			repositoryReviewPublicationRoute+"automations/"+automation.ID+"/findings/"+
				finding.ID+"/issue-link",
			strings.NewReader(`{}`),
		)
		response := httptest.NewRecorder()
		newRepositoryReviewPublicationHandler(loop).ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest ||
			!strings.Contains(response.Body.String(), `"code":"repository_not_linkable"`) {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("unknown operation", func(t *testing.T) {
		loop := repositoryReviewRankingTestLoop(t, &repositoryReviewRankingProvider{})
		_, _, finding, automation := repositoryReviewIssueLinkTestFixture(
			t, loop.GetConfig().WorkspacePath(), "owner/unknown-operation",
		)
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
		response := httptest.NewRecorder()
		newRepositoryReviewPublicationHandler(loop).serveRepositoryReviewAutomationOperation(
			response, request,
			repositoryReviewAutomationOperation{
				AutomationID: automation.ID, FindingID: finding.ID, Action: "unknown",
			},
		)
		if response.Code != http.StatusNotFound {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
	})
}

func TestRepositoryReviewAutomationStateFallsBackOnlyToUnambiguousRunMembership(t *testing.T) {
	t.Run("direct identity", func(t *testing.T) {
		workspace := t.TempDir()
		store, expected, _ := repositoryReviewPublicationTestDraft(t, workspace, "owner/repo")
		state, found, err := repositoryReviewAutomationState(
			store,
			repoaudit.RepositoryReviewAutomation{Repository: "https://github.com/Owner/Repo.git"},
		)
		if err != nil || !found || state.ID != expected.ID {
			t.Fatalf("state=%#v found=%v err=%v", state, found, err)
		}
	})

	t.Run("no run membership", func(t *testing.T) {
		store := repoaudit.NewStore(t.TempDir())
		state, found, err := repositoryReviewAutomationState(
			store, repoaudit.RepositoryReviewAutomation{Repository: "missing/repo"},
		)
		if err != nil || found || state.ID != "" {
			t.Fatalf("state=%#v found=%v err=%v", state, found, err)
		}
	})

	t.Run("unique run fallback", func(t *testing.T) {
		workspace := t.TempDir()
		store, expected, _ := repositoryReviewPublicationTestDraft(t, workspace, "owner/actual")
		state, found, err := repositoryReviewAutomationState(
			store,
			repoaudit.RepositoryReviewAutomation{
				Repository: "missing/repo", RunIDs: []string{"publication-run", "unrelated"},
			},
		)
		if err != nil || !found || state.ID != expected.ID {
			t.Fatalf("state=%#v found=%v err=%v", state, found, err)
		}
	})

	t.Run("unmatched run", func(t *testing.T) {
		workspace := t.TempDir()
		store, _, _ := repositoryReviewPublicationTestDraft(t, workspace, "owner/actual")
		state, found, err := repositoryReviewAutomationState(
			store,
			repoaudit.RepositoryReviewAutomation{
				Repository: "missing/repo", RunIDs: []string{"other-run"},
			},
		)
		if err != nil || found || state.ID != "" {
			t.Fatalf("state=%#v found=%v err=%v", state, found, err)
		}
	})

	t.Run("ambiguous run", func(t *testing.T) {
		workspace := t.TempDir()
		store, _, _ := repositoryReviewPublicationTestDraft(t, workspace, "owner/first")
		_, _, _ = repositoryReviewPublicationTestDraft(t, workspace, "owner/second")
		_, found, err := repositoryReviewAutomationState(
			store,
			repoaudit.RepositoryReviewAutomation{
				Repository: "missing/repo", RunIDs: []string{"publication-run"},
			},
		)
		if err == nil || found || !strings.Contains(err.Error(), "ambiguous") {
			t.Fatalf("found=%v err=%v", found, err)
		}
	})

	t.Run("ledger list failure", func(t *testing.T) {
		workspace := t.TempDir()
		_, _, _ = repositoryReviewPublicationTestDraft(t, workspace, "owner/corrupt")
		root := filepath.Join(workspace, "repository_reviews")
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		statePath := ""
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") &&
				!strings.HasSuffix(entry.Name(), ".summary.json") {
				statePath = filepath.Join(root, entry.Name())
				break
			}
		}
		if statePath == "" {
			t.Fatal("authoritative state file was not found")
		}
		if writeErr := os.WriteFile(statePath, []byte(`{`), 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
		_, found, err := repositoryReviewAutomationState(
			repoaudit.NewStore(workspace),
			repoaudit.RepositoryReviewAutomation{
				Repository: "missing/repo", RunIDs: []string{"publication-run"},
			},
		)
		if err == nil || found {
			t.Fatalf("found=%v err=%v", found, err)
		}
	})
}

func TestRepositoryReviewCandidateEndpointRejectsInvalidStaleAndUnavailableRequests(t *testing.T) {
	state := repoaudit.RepositoryState{Repository: "owner/repo"}
	baseFinding := repoaudit.Finding{ID: "rfn_test", Version: 3, Status: repoaudit.FindingOpen}
	automation := repoaudit.RepositoryReviewAutomation{
		IssueWriterModel: "issue-writer", EffectiveAccountRef: "writer-account",
	}
	handler := newRepositoryReviewPublicationHandler(nil)
	for _, test := range []struct {
		name    string
		body    string
		finding repoaudit.Finding
		status  int
		code    string
	}{
		{name: "malformed", body: `{`, finding: baseFinding, status: http.StatusBadRequest, code: "invalid_request"},
		{name: "unknown field", body: `{"expected_version":3,"extra":true}`, finding: baseFinding, status: http.StatusBadRequest, code: "invalid_request"},
		{name: "stale", body: `{"expected_version":2}`, finding: baseFinding, status: http.StatusConflict, code: "stale_repository_review"},
		{
			name: "not open", body: `{"expected_version":3}`,
			finding: func() repoaudit.Finding {
				finding := baseFinding
				finding.Status = repoaudit.FindingDismissed
				return finding
			}(),
			status: http.StatusConflict, code: "stale_repository_review",
		},
		{
			name: "associated", body: `{"expected_version":3}`,
			finding: func() repoaudit.Finding {
				finding := baseFinding
				finding.IssueDraftID = "rid_existing"
				return finding
			}(),
			status: http.StatusConflict, code: "stale_repository_review",
		},
		{name: "writer unavailable", body: `{"expected_version":3}`, finding: baseFinding, status: http.StatusServiceUnavailable, code: "issue_ranking_unavailable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			handler.serveRepositoryReviewIssueCandidates(
				response, request, nil, automation, state, test.finding,
			)
			if response.Code != test.status ||
				!strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response=%d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRepositoryReviewCandidateEndpointReportsSearchAndRankingFailuresSafely(t *testing.T) {
	for _, test := range []struct {
		name       string
		searchText string
		searchErr  error
		ranking    string
		code       string
	}{
		{
			name: "search failure", searchErr: errors.New("GitHub unavailable"),
			ranking: `{"rankings":[]}`, code: "issue_search_unavailable",
		},
		{
			name:       "invalid search response",
			searchText: `{"items":42}`, ranking: `{"rankings":[]}`,
			code: "issue_search_unavailable",
		},
		{
			name: "ranking failure",
			searchText: `{"items":[{"id":12,"number":12,"title":"Candidate",` +
				`"html_url":"https://github.com/owner/repo/issues/12"}]}`,
			ranking: `not-json`, code: "issue_ranking_unavailable",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			loop := repositoryReviewRankingTestLoop(t, &repositoryReviewRankingProvider{
				responses: []string{test.ranking},
			})
			manager := &repositoryReviewPublicationMCPManager{
				searchText: test.searchText, searchErr: test.searchErr,
			}
			loop.RegisterTool(tools.NewMCPTool(manager, reviews.DefaultGitHubMCPServer, &sdkmcp.Tool{
				Name:        reviews.GitHubSearchIssuesTool,
				InputSchema: map[string]any{"type": "object", "additionalProperties": true},
			}))
			_, _, finding, automation := repositoryReviewIssueLinkTestFixture(
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
			if response.Code != http.StatusServiceUnavailable ||
				!strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response=%d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRepositoryReviewGatewayHandlersFailClosedWhenDependenciesCannotInitialize(t *testing.T) {
	t.Run("candidate tool runner", func(t *testing.T) {
		loop := repositoryReviewRankingTestLoop(t, &repositoryReviewRankingProvider{})
		_, _, finding, automation := repositoryReviewIssueLinkTestFixture(
			t, loop.GetConfig().WorkspacePath(), "owner/repo",
		)
		handler := newRepositoryReviewPublicationHandler(loop)
		handler.newToolRunner = func(*agent.AgentLoop, string) (workflows.ToolRunner, error) {
			return nil, errors.New("tool runtime unavailable")
		}
		request := httptest.NewRequest(
			http.MethodPost,
			repositoryReviewPublicationRoute+"automations/"+automation.ID+"/findings/"+
				finding.ID+"/issue-link/candidates",
			strings.NewReader(`{"expected_version":`+strconv.FormatInt(finding.Version, 10)+`}`),
		)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable ||
			!strings.Contains(response.Body.String(), `"code":"issue_search_unavailable"`) {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("candidate provider", func(t *testing.T) {
		loop := repositoryReviewRankingTestLoop(t, &repositoryReviewRankingProvider{})
		_, _, finding, automation := repositoryReviewIssueLinkTestFixture(
			t, loop.GetConfig().WorkspacePath(), "owner/repo",
		)
		handler := newRepositoryReviewPublicationHandler(loop)
		handler.newGitHubProvider = func(
			workflows.ToolRunner,
			string,
		) (*reviews.GitHubProvider, error) {
			return nil, errors.New("GitHub provider unavailable")
		}
		request := httptest.NewRequest(
			http.MethodPost,
			repositoryReviewPublicationRoute+"automations/"+automation.ID+"/findings/"+
				finding.ID+"/issue-link/candidates",
			strings.NewReader(`{"expected_version":`+strconv.FormatInt(finding.Version, 10)+`}`),
		)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable ||
			!strings.Contains(response.Body.String(), `"code":"issue_search_unavailable"`) {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("link provider", func(t *testing.T) {
		loop := repositoryReviewRankingTestLoop(t, &repositoryReviewRankingProvider{})
		store, state, finding, automation := repositoryReviewIssueLinkTestFixture(
			t, loop.GetConfig().WorkspacePath(), "owner/repo",
		)
		handler := newRepositoryReviewPublicationHandler(loop)
		handler.newGitHubProvider = func(
			workflows.ToolRunner,
			string,
		) (*reviews.GitHubProvider, error) {
			return nil, errors.New("GitHub provider unavailable")
		}
		request := httptest.NewRequest(
			http.MethodPost, "/",
			strings.NewReader(`{"issue_url":"https://github.com/owner/repo/issues/12",`+
				`"expected_version":`+strconv.FormatInt(finding.Version, 10)+`,"confirmed":true}`),
		)
		response := httptest.NewRecorder()
		handler.serveRepositoryReviewIssueLink(
			response, request, loop, store, automation, state, finding,
		)
		if response.Code != http.StatusServiceUnavailable ||
			!strings.Contains(response.Body.String(), `"code":"issue_link_unavailable"`) {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
	})

	t.Run("publication provider", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Agents.Defaults.Workspace = t.TempDir()
		messageBus := bus.NewMessageBus()
		loop := agent.NewAgentLoop(cfg, messageBus, &startupBlockedProvider{reason: "not used"})
		t.Cleanup(func() {
			loop.Stop()
			messageBus.Close()
			loop.Close()
		})
		_, state, draft := repositoryReviewPublicationTestDraft(
			t, cfg.WorkspacePath(), "owner/repo",
		)
		handler := newRepositoryReviewPublicationHandler(loop)
		handler.newGitHubProvider = func(
			workflows.ToolRunner,
			string,
		) (*reviews.GitHubProvider, error) {
			return nil, errors.New("GitHub provider unavailable")
		}
		request := httptest.NewRequest(
			http.MethodPost,
			repositoryReviewPublicationRoute+state.ID+"/issue-drafts/"+draft.ID+"/publish",
			strings.NewReader(`{"expected_version":`+strconv.FormatInt(draft.Version, 10)+`}`),
		)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable ||
			!strings.Contains(response.Body.String(), `"code":"publication_unavailable"`) {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
	})
}

func (manager *repositoryReviewPublicationMCPManager) CallTool(
	_ context.Context,
	_, toolName string,
	_ map[string]any,
) (*sdkmcp.CallToolResult, error) {
	text, err := manager.searchText, manager.searchErr
	if toolName == reviews.GitHubIssueWriteTool {
		text, err = manager.createText, manager.createErr
	}
	if manager.beforeReturn != nil {
		manager.beforeReturn(toolName)
	}
	if err != nil {
		return nil, err
	}
	return &sdkmcp.CallToolResult{Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: text}}}, nil
}

func TestRepositoryReviewPublicationHandlerReportsPersistenceFailures(t *testing.T) {
	tests := []struct {
		name         string
		initialState repoaudit.IssueDraftState
		searchText   func(repoaudit.IssueDraft) string
		createText   string
		createErr    error
		poisonTool   string
	}{
		{
			name: "recovered issue update", poisonTool: reviews.GitHubSearchIssuesTool,
			searchText: func(draft repoaudit.IssueDraft) string {
				return `{"items":[{"id":31,"html_url":"https://github.com/owner/repo/issues/31","body":"` +
					repositoryReviewIssueMarker(draft.ID) + `"}]}`
			},
		},
		{
			name: "publishing transition", initialState: repoaudit.IssueDraftPublishing,
			poisonTool: reviews.GitHubSearchIssuesTool,
			searchText: func(repoaudit.IssueDraft) string { return `{"items":[]}` },
		},
		{
			name: "ambiguous transition", poisonTool: reviews.GitHubIssueWriteTool,
			searchText: func(repoaudit.IssueDraft) string { return `{"items":[]}` },
			createErr:  errors.New("connection reset by peer"),
		},
		{
			name: "definite failure reset", poisonTool: reviews.GitHubIssueWriteTool,
			searchText: func(repoaudit.IssueDraft) string { return `{"items":[]}` },
			createErr:  errors.New("HTTP status: 422 validation failed"),
		},
		{
			name: "posted transition", poisonTool: reviews.GitHubIssueWriteTool,
			searchText: func(repoaudit.IssueDraft) string { return `{"items":[]}` },
			createText: `{"id":32,"html_url":"https://github.com/owner/repo/issues/32"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace := t.TempDir()
			cfg := config.DefaultConfig()
			cfg.Agents.Defaults.Workspace = workspace
			cfg.Tools.MCP.Enabled = false
			messageBus := bus.NewMessageBus()
			loop := agent.NewAgentLoop(cfg, messageBus, &startupBlockedProvider{reason: "not used"})
			t.Cleanup(func() {
				loop.Stop()
				messageBus.Close()
				loop.Close()
			})
			manager := &repositoryReviewPublicationMCPManager{
				createText: test.createText, createErr: test.createErr,
			}
			loop.RegisterTool(tools.NewMCPTool(manager, reviews.DefaultGitHubMCPServer, &sdkmcp.Tool{
				Name:        reviews.GitHubSearchIssuesTool,
				InputSchema: map[string]any{"type": "object", "additionalProperties": true},
			}))
			loop.RegisterTool(tools.NewMCPTool(manager, reviews.DefaultGitHubMCPServer, &sdkmcp.Tool{
				Name:        reviews.GitHubIssueWriteTool,
				InputSchema: map[string]any{"type": "object", "additionalProperties": true},
			}))
			store, state, draft := repositoryReviewPublicationTestDraft(t, workspace, "owner/repo")
			if test.initialState == repoaudit.IssueDraftPublishing {
				_, claimed, didClaim, err := store.ClaimIssueDraftPublication("owner/repo", draft.ID, draft.Version)
				if err != nil || !didClaim {
					t.Fatalf("claim draft: claimed=%v err=%v", didClaim, err)
				}
				draft = claimed
			}
			manager.searchText = test.searchText(draft)
			manager.beforeReturn = func(toolName string) {
				if toolName != test.poisonTool {
					return
				}
				root := filepath.Join(workspace, "repository_reviews")
				if err := os.RemoveAll(root); err != nil {
					t.Errorf("remove store root: %v", err)
					return
				}
				if err := os.WriteFile(root, []byte("not a directory"), 0o600); err != nil {
					t.Errorf("replace store root: %v", err)
				}
			}
			request := httptest.NewRequest(
				http.MethodPost,
				repositoryReviewPublicationRoute+state.ID+"/issue-drafts/"+draft.ID+"/publish",
				strings.NewReader(`{"expected_version":`+strconv.FormatInt(draft.Version, 10)+`}`),
			)
			response := httptest.NewRecorder()
			newRepositoryReviewPublicationHandler(loop).ServeHTTP(response, request)
			if response.Code != http.StatusServiceUnavailable {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRepositoryReviewPublicationHandlerPublishesAndRecovers(t *testing.T) {
	tests := []struct {
		name         string
		configure    func(manager *repositoryReviewPublicationMCPManager, draft repoaudit.IssueDraft)
		initialState repoaudit.IssueDraftState
		wantStatus   int
		wantDraft    repoaudit.IssueDraftState
		wantOutcome  string
		wantExternal string
	}{
		{
			name: "recover existing issue",
			configure: func(manager *repositoryReviewPublicationMCPManager, draft repoaudit.IssueDraft) {
				manager.searchText = `{"items":[{"id":21,"html_url":"https://github.com/owner/repo/issues/21","body":"` +
					repositoryReviewIssueMarker(
						draft.ID,
					) + `"}]}`
			},
			wantStatus: http.StatusOK, wantDraft: repoaudit.IssueDraftPosted,
			wantExternal: "21",
		},
		{
			name: "create issue",
			configure: func(manager *repositoryReviewPublicationMCPManager, _ repoaudit.IssueDraft) {
				manager.searchText = `{"items":[]}`
				manager.createText = `{"id":22,"html_url":"https://github.com/owner/repo/issues/22"}`
			},
			wantStatus: http.StatusOK, wantDraft: repoaudit.IssueDraftPosted,
			wantExternal: "22",
		},
		{
			name: "already publishing becomes unknown",
			configure: func(manager *repositoryReviewPublicationMCPManager, _ repoaudit.IssueDraft) {
				manager.searchText = `{"items":[]}`
			},
			initialState: repoaudit.IssueDraftPublishing,
			wantStatus:   http.StatusAccepted, wantDraft: repoaudit.IssueDraftUnknown,
			wantOutcome: "unknown",
		},
		{
			name: "already unknown remains unknown",
			configure: func(manager *repositoryReviewPublicationMCPManager, _ repoaudit.IssueDraft) {
				manager.searchText = `{"items":[]}`
			},
			initialState: repoaudit.IssueDraftUnknown,
			wantStatus:   http.StatusAccepted, wantDraft: repoaudit.IssueDraftUnknown,
			wantOutcome: "unknown",
		},
		{
			name: "ambiguous create",
			configure: func(manager *repositoryReviewPublicationMCPManager, _ repoaudit.IssueDraft) {
				manager.searchText = `{"items":[]}`
				manager.createErr = errors.New("connection reset by peer")
			},
			wantStatus: http.StatusAccepted, wantDraft: repoaudit.IssueDraftUnknown,
			wantOutcome: "unknown",
		},
		{
			name: "definite create failure",
			configure: func(manager *repositoryReviewPublicationMCPManager, _ repoaudit.IssueDraft) {
				manager.searchText = `{"items":[]}`
				manager.createErr = errors.New("HTTP status: 422 validation failed")
			},
			wantStatus: http.StatusServiceUnavailable, wantDraft: repoaudit.IssueDraftEditing,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Agents.Defaults.Workspace = t.TempDir()
			cfg.Tools.MCP.Enabled = false
			messageBus := bus.NewMessageBus()
			loop := agent.NewAgentLoop(cfg, messageBus, &startupBlockedProvider{reason: "not used"})
			t.Cleanup(func() {
				loop.Stop()
				messageBus.Close()
				loop.Close()
			})
			manager := &repositoryReviewPublicationMCPManager{}
			loop.RegisterTool(tools.NewMCPTool(manager, reviews.DefaultGitHubMCPServer, &sdkmcp.Tool{
				Name: reviews.GitHubSearchIssuesTool,
				InputSchema: map[string]any{
					"type": "object", "additionalProperties": true,
				},
			}))
			loop.RegisterTool(tools.NewMCPTool(manager, reviews.DefaultGitHubMCPServer, &sdkmcp.Tool{
				Name: reviews.GitHubIssueWriteTool,
				InputSchema: map[string]any{
					"type": "object", "additionalProperties": true,
				},
			}))

			store, state, draft := repositoryReviewPublicationTestDraft(t, cfg.WorkspacePath(), "owner/repo")
			if test.initialState != "" {
				_, claimed, didClaim, err := store.ClaimIssueDraftPublication("owner/repo", draft.ID, draft.Version)
				if err != nil || !didClaim {
					t.Fatalf("claim draft: claimed=%v err=%v", didClaim, err)
				}
				draft = claimed
				if test.initialState == repoaudit.IssueDraftUnknown {
					_, draft, err = store.SetIssueDraftPublication(
						"owner/repo", draft.ID, draft.Version, repoaudit.IssueDraftUnknown, "", "",
					)
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			test.configure(manager, draft)
			request := httptest.NewRequest(
				http.MethodPost,
				repositoryReviewPublicationRoute+state.ID+"/issue-drafts/"+draft.ID+"/publish",
				strings.NewReader(`{"expected_version":`+strconv.FormatInt(draft.Version, 10)+`}`),
			)
			response := httptest.NewRecorder()
			newRepositoryReviewPublicationHandler(loop).ServeHTTP(response, request)
			if response.Code != test.wantStatus ||
				(test.wantOutcome != "" && !strings.Contains(response.Body.String(), `"outcome":"`+test.wantOutcome+`"`)) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			persisted, found, err := store.GetByID(state.ID)
			if err != nil || !found {
				t.Fatalf("persisted state found=%v err=%v", found, err)
			}
			persistedDraft, found := repositoryReviewDraft(persisted, draft.ID)
			if !found || persistedDraft.State != test.wantDraft ||
				(test.wantExternal != "" && persistedDraft.ExternalID != test.wantExternal) {
				t.Fatalf("persisted draft = %#v", persistedDraft)
			}
		})
	}
}

func TestRepositoryReviewPublicationHelpersCoverBoundaryResponses(t *testing.T) {
	state := repoaudit.RepositoryState{IssueDrafts: []repoaudit.IssueDraft{{ID: "rid_match"}}}
	if draft, found := repositoryReviewDraft(state, "rid_match"); !found || draft.ID != "rid_match" {
		t.Fatalf("draft = %#v, found=%v", draft, found)
	}
	if _, found := repositoryReviewDraft(state, "rid_missing"); found {
		t.Fatal("missing draft was found")
	}

	for _, test := range []struct {
		err    error
		found  bool
		status int
	}{
		{err: nil, found: false, status: http.StatusNotFound},
		{err: os.ErrNotExist, found: true, status: http.StatusNotFound},
		{err: repoaudit.ErrConflict, found: true, status: http.StatusConflict},
		{err: repoaudit.ErrRepositoryReviewPurgeInProgress, found: true, status: http.StatusConflict},
		{
			err: repoaudit.ErrHistoricalDeduplicationInProgress, found: true,
			status: http.StatusConflict,
		},
		{err: errors.New("disk unavailable"), found: true, status: http.StatusServiceUnavailable},
	} {
		response := httptest.NewRecorder()
		writeRepositoryReviewPublicationStoreError(response, test.err, test.found)
		if response.Code != test.status {
			t.Fatalf("write store error status=%d, want %d", response.Code, test.status)
		}
	}

	invalidRequests := []*http.Request{
		nil,
		httptest.NewRequest(http.MethodGet, repositoryReviewPublicationRoute+"r/issue-drafts/d/publish", nil),
		httptest.NewRequest(http.MethodPost, repositoryReviewPublicationRoute+"r/issue-drafts/d/publish?q=1", nil),
		httptest.NewRequest(http.MethodPost, repositoryReviewPublicationRoute+"r/other/d/publish", nil),
	}
	for _, request := range invalidRequests {
		if _, _, ok := repositoryReviewPublicationRouteIDs(request); ok {
			t.Fatalf("accepted route %#v", request)
		}
	}

	for _, repository := range []string{
		strings.Repeat("a", 101) + "/repo",
		"owner/.",
		"owner/..",
		"owner/repo!",
	} {
		if validRepositoryReviewGitHubIdentity(repository) {
			t.Fatalf("accepted invalid GitHub identity %q", repository)
		}
	}
	if repositoryReviewIssueCreateAmbiguous(
		errors.Join(reviews.ErrWorkspaceProviderCallNotDispatched, errors.New("invalid request")),
	) {
		t.Fatal("pre-dispatch workspace provider error became ambiguous")
	}
}

func TestRepositoryReviewProviderHelpersRejectMalformedProviderResults(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		result     map[string]any
		err        error
	}{
		{name: "identity", repository: "invalid", result: map[string]any{"text": `{}`}},
		{name: "dispatch", repository: "owner/repo", err: errors.New("write failed")},
		{name: "json", repository: "owner/repo", result: map[string]any{"text": `not-json`}},
		{
			name:       "foreign",
			repository: "owner/repo",
			result:     map[string]any{"text": `{"id":1,"html_url":"https://github.com/other/repo/issues/1"}`},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider, err := reviews.NewGitHubProvider(prWorkspaceProviderToolRunnerFunc(func(
				_ context.Context,
				_ workflows.ToolRequest,
			) (map[string]any, error) {
				return test.result, test.err
			}), "")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := createRepositoryReviewIssue(
				t.Context(), provider, test.repository, repoaudit.IssueDraft{Title: "title"}, "marker",
			); err == nil {
				t.Fatal("createRepositoryReviewIssue() succeeded")
			}
		})
	}
}

func TestFindRepositoryReviewIssueHandlesProviderWireShapes(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		err   error
		found bool
		fail  bool
	}{
		{name: "provider", err: errors.New("search failed"), fail: true},
		// A scalar is valid provider JSON but neither accepted issue-search wire
		// shape, so the publication boundary must reject both envelope and direct
		// array decoding paths.
		{name: "invalid", text: `42`, fail: true},
		{
			name:  "direct",
			text:  `[{"id":2,"html_url":"https://github.com/owner/repo/issues/2","body":"marker"}]`,
			found: true,
		},
		{name: "empty", text: `{"items":[]}`},
		{
			name: "foreign",
			text: `{"items":[{"id":2,"html_url":"https://github.com/other/repo/issues/2","body":"marker"},{"id":3,"html_url":"https://github.com/owner/repo/issues/3","body":"other"}]}`,
		},
		{
			name: "multiple",
			text: `{"items":[{"id":2,"html_url":"https://github.com/owner/repo/issues/2","body":"marker"},{"id":3,"html_url":"https://github.com/owner/repo/issues/3","body":"marker"}]}`,
			fail: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider, err := reviews.NewGitHubProvider(prWorkspaceProviderToolRunnerFunc(func(
				_ context.Context,
				_ workflows.ToolRequest,
			) (map[string]any, error) {
				return map[string]any{"text": test.text}, test.err
			}), "")
			if err != nil {
				t.Fatal(err)
			}
			_, found, findErr := findRepositoryReviewIssue(t.Context(), provider, "owner/repo", "marker")
			if found != test.found || (findErr != nil) != test.fail {
				t.Fatalf("found=%v err=%v", found, findErr)
			}
		})
	}
}
