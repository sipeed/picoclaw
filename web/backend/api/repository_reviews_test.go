package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/repoaudit"
)

func TestRepositoryReviewRoutesSelectDiscussAndIssueData(t *testing.T) {
	workspace := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatal(err)
	}
	state := seedRepositoryReviewAPIState(t, workspace)
	state = completeRepositoryReviewAPIMappingJobs(t, workspace, state)
	mux := http.NewServeMux()
	NewHandler(configPath).RegisterRoutes(mux)

	list := httptest.NewRecorder()
	mux.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/repository-reviews", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), state.ID) ||
		strings.Contains(list.Body.String(), state.Findings[0].Evidence) {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}

	statusBody, _ := json.Marshal(map[string]any{
		"status": "dismissed", "expected_version": state.Version,
	})
	statusResponse := httptest.NewRecorder()
	statusRequest := httptest.NewRequest(
		http.MethodPatch,
		"/api/repository-reviews/"+state.ID+"/findings/"+state.Findings[0].ID,
		bytes.NewReader(statusBody),
	)
	setRepositoryReviewMutationHeaders(statusRequest)
	mux.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusConflict {
		t.Fatalf("status mutation=%d body=%s", statusResponse.Code, statusResponse.Body.String())
	}

	draftBody, _ := json.Marshal(map[string]any{
		"finding_ids": []string{state.Findings[0].ID}, "expected_version": state.Version,
	})
	draftResponse := httptest.NewRecorder()
	draftRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/repository-reviews/"+state.ID+"/issue-drafts",
		bytes.NewReader(draftBody),
	)
	setRepositoryReviewMutationHeaders(draftRequest)
	mux.ServeHTTP(draftResponse, draftRequest)
	if draftResponse.Code != http.StatusCreated ||
		!strings.Contains(draftResponse.Body.String(), "blob `") ||
		!strings.Contains(draftResponse.Body.String(), state.LastCommitSHA) {
		t.Fatalf("draft status=%d body=%s", draftResponse.Code, draftResponse.Body.String())
	}
}

func TestRepositoryReviewDetailPaginationProjectsOnlyReferencedContext(t *testing.T) {
	canary := repoaudit.NewRepositoryReviewCampaignID()
	internalCanary := "internal-authority-canary"
	firstContext := repoaudit.FindingContext{ID: "context-first", CampaignID: canary}
	secondContext := repoaudit.FindingContext{ID: "context-second", CampaignID: canary}
	drafts := make([]repoaudit.IssueDraft, 12)
	for index := range drafts {
		drafts[index].ID = fmt.Sprintf("draft-%02d", index)
	}
	state := repoaudit.RepositoryState{
		SchemaVersion:           repoaudit.SchemaVersion,
		ID:                      "rrp_compatibility_projection",
		Repository:              "owner/repository",
		Version:                 7,
		ReviewVersion:           9,
		LastCommitSHA:           strings.Repeat("b", 40),
		UpdatedAt:               time.Unix(123, 0).UTC(),
		LastExcludedFiles:       3,
		Files:                   map[string]repoaudit.ReviewedFile{"first.go": {}},
		ReviewAttempts:          map[string]int{internalCanary: 2},
		ReviewAttemptIdentities: map[string]string{internalCanary: "rat_internal"},
		Unsupported: map[string]repoaudit.UnsupportedFile{
			"large.go": {
				FileRef: repoaudit.FileRef{Path: "large.go"}, Reason: "file_too_large",
				ForceCampaignID: canary,
			},
		},
		Findings: []repoaudit.Finding{
			{
				ID: "finding-first", CampaignID: canary, ContextIDs: []string{firstContext.ID},
				Status: repoaudit.FindingOpen,
			},
			{
				ID: "finding-second", CampaignID: canary, ContextIDs: []string{secondContext.ID},
				Status: repoaudit.FindingDismissed,
			},
		},
		RawFindings: []repoaudit.RawReviewFinding{{ID: internalCanary}},
		DeduplicatedFindings: []repoaudit.DeduplicatedReviewFinding{{
			ID: internalCanary, Status: repoaudit.FindingOpen,
		}},
		DeduplicationJobs:        []repoaudit.DeduplicationJob{{ID: internalCanary}},
		NextDeduplicationOrdinal: 23,
		FindingsProcessing:       repoaudit.FindingsProcessingCounters{RawTotal: 1},
		Contexts:                 []repoaudit.FindingContext{firstContext, secondContext},
		Runs:                     make([]repoaudit.ReviewRun, 51),
		FileAttributions: []repoaudit.RepositoryReviewFileAttribution{{
			ID: internalCanary,
		}},
		IssueDrafts:        drafts,
		RepositoryFindings: []repoaudit.RepositoryFinding{{ID: internalCanary}},
		MappingJobs:        []repoaudit.RepositoryMappingJob{{ID: internalCanary}},
		ValidationJobs:     []repoaudit.RepositoryValidationJob{{ID: internalCanary}},
		CurrentCampaign: &repoaudit.RepositoryReviewCampaignCoverage{
			ID: canary, CommitSHA: strings.Repeat("a", 40),
			Paths: map[string]repoaudit.RepositoryReviewCampaignPathCoverage{},
		},
		ActiveReviewRun: &repoaudit.RepositoryReviewActiveRun{
			ID: internalCanary, CampaignID: canary,
		},
		CampaignHistory:         map[string]string{canary: strings.Repeat("a", 40)},
		ActiveForceCampaignID:   internalCanary,
		ActiveForceProfileHash:  internalCanary,
		ActiveForceCommitSHA:    internalCanary,
		HistoricalDeduplication: repoaudit.HistoricalDeduplicationReplay{Required: true, Error: internalCanary},
	}
	for index := range state.Runs {
		state.Runs[index].CampaignID = canary
		state.Runs[index].CheckpointDigests = map[string]string{internalCanary: internalCanary}
		state.Runs[index].CheckpointScopes = map[string][]repoaudit.FileRef{
			internalCanary: {{Path: internalCanary}},
		}
	}

	projected := projectRepositoryReviewDetail(state, repositoryReviewPageRequest{
		FindingOffset: 1, FindingLimit: 1, DraftLimit: 10,
	})
	if projected.FindingOffset != 1 || projected.FindingTotal != 2 ||
		projected.NextFindingOffset != nil || len(projected.Findings) != 1 ||
		projected.Findings[0].ID != "finding-second" || len(projected.Contexts) != 1 ||
		projected.Contexts[0].ID != secondContext.ID ||
		len(projected.Runs) != 50 || projected.DraftTotal != 12 ||
		projected.NextDraftOffset == nil || *projected.NextDraftOffset != 10 ||
		len(projected.IssueDrafts) != 10 || projected.IssueDrafts[0].ID != "draft-02" ||
		projected.IssueDrafts[9].ID != "draft-11" || projected.FindingCount != 1 ||
		projected.RepositoryFindingCount != 1 || projected.OpenFindingCount != 1 ||
		projected.IssueDraftCount != 12 || projected.UnsupportedCount != 1 ||
		projected.ReviewedFileCount != 1 || projected.ExcludedFileCount != 3 {
		t.Fatalf("projected detail=%#v", projected)
	}
	encoded, err := json.Marshal(projected)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), canary) || strings.Contains(string(encoded), internalCanary) {
		t.Fatalf("projected detail exposed campaign authority: %s", encoded)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatal(err)
	}
	wantKeys := map[string]struct{}{
		"schema_version": {}, "id": {}, "repository": {}, "version": {}, "review_version": {},
		"last_commit_sha": {}, "finding_count": {}, "repository_finding_count": {},
		"open_finding_count": {}, "issue_draft_count": {}, "unsupported_count": {},
		"reviewed_file_count": {}, "excluded_file_count": {}, "updated_at": {},
		"unsupported": {}, "findings": {}, "contexts": {}, "runs": {}, "issue_drafts": {},
		"finding_offset": {}, "finding_total": {}, "draft_offset": {}, "draft_total": {},
		"next_draft_offset": {},
	}
	if len(object) != len(wantKeys) {
		t.Fatalf("projected detail keys=%v, want=%v", object, wantKeys)
	}
	for key := range object {
		if _, allowed := wantKeys[key]; !allowed {
			t.Fatalf("projected detail exposed non-compatibility field %q: %s", key, encoded)
		}
	}
	if state.Findings[1].CampaignID != canary || state.Contexts[1].CampaignID != canary ||
		state.Runs[1].CampaignID != canary || state.Runs[1].CheckpointDigests == nil ||
		state.CurrentCampaign == nil || len(state.CampaignHistory) != 1 ||
		len(state.RawFindings) != 1 || len(state.RepositoryFindings) != 1 ||
		state.Unsupported["large.go"].ForceCampaignID != canary {
		t.Fatal("detail projection mutated source campaign authority")
	}
}

func TestRepositoryReviewDetailRejectsAmbiguousOrOversizedPage(t *testing.T) {
	for _, target := range []string{
		"/api/repository-reviews/id?offset=1&offset=2",
		"/api/repository-reviews/id?limit=201",
		"/api/repository-reviews/id?unexpected=true",
	} {
		request := httptest.NewRequest(http.MethodGet, target, nil)
		if _, err := repositoryReviewPage(request); err == nil {
			t.Fatalf("repositoryReviewPage(%q) unexpectedly succeeded", target)
		}
	}
}

func TestRepositoryReviewMutationRejectsStaleVersionAndUnknownFields(t *testing.T) {
	workspace := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatal(err)
	}
	state := seedRepositoryReviewAPIState(t, workspace)
	mux := http.NewServeMux()
	NewHandler(configPath).RegisterRoutes(mux)

	for name, test := range map[string]struct {
		body string
		want int
	}{
		"stale":   {body: `{"status":"dismissed","expected_version":999}`, want: http.StatusConflict},
		"unknown": {body: `{"status":"dismissed","expected_version":1,"secret":"x"}`, want: http.StatusBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(
				http.MethodPatch,
				"/api/repository-reviews/"+state.ID+"/findings/"+state.Findings[0].ID,
				strings.NewReader(test.body),
			)
			setRepositoryReviewMutationHeaders(request)
			mux.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status=%d want=%d body=%s", response.Code, test.want, response.Body.String())
			}
		})
	}
}

func TestRepositoryReviewMutationRejectsCrossSiteAndNonJSONRequests(t *testing.T) {
	workspace := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatal(err)
	}
	state := seedRepositoryReviewAPIState(t, workspace)
	mux := http.NewServeMux()
	NewHandler(configPath).RegisterRoutes(mux)
	for name, test := range map[string][2]string{
		"cross-site": {"application/json", "cross-site"},
		"text":       {"text/plain", "same-origin"},
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(
				http.MethodPatch,
				"http://launcher.local/api/repository-reviews/"+state.ID+"/findings/"+state.Findings[0].ID,
				strings.NewReader(fmt.Sprintf(`{"status":"dismissed","expected_version":%d}`, state.Version)),
			)
			request.Header.Set("Content-Type", test[0])
			request.Header.Set("Sec-Fetch-Site", test[1])
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRepositoryReviewPublishProxiesOnlyExactDraftPayloadToProtectedGateway(t *testing.T) {
	workspace := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = workspace
	if err := config.SaveConfig(configPath, cfg); err != nil {
		t.Fatal(err)
	}
	var captured *http.Request
	var capturedBody string
	installEventProxyStubs(t, func(request *http.Request, _ time.Duration) (*http.Response, error) {
		captured = request
		body, _ := io.ReadAll(request.Body)
		capturedBody = string(body)
		return eventUpstreamResponse(http.StatusOK, `{"repository":{"id":"rrp_test"},"draft":{"id":"rid_test"}}`), nil
	})
	mux := http.NewServeMux()
	NewHandler(configPath).RegisterRoutes(mux)
	request := httptest.NewRequest(
		http.MethodPost,
		"http://launcher.local/api/repository-reviews/rrp_test/issue-drafts/rid_test/publish",
		strings.NewReader(`{"expected_version":3}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("publish status=%d body=%s", response.Code, response.Body.String())
	}
	if captured == nil || captured.Method != http.MethodPost ||
		captured.URL.Path != "/runtime/repository-reviews/rrp_test/issue-drafts/rid_test/publish" ||
		captured.Header.Get("Authorization") != "Bearer gateway-pid-token" ||
		capturedBody != `{"expected_version":3}` {
		t.Fatalf("gateway request=%#v body=%q", captured, capturedBody)
	}
}

func seedRepositoryReviewAPIState(t *testing.T, workspace string) repoaudit.RepositoryState {
	t.Helper()
	store := repoaudit.NewStore(workspace)
	file := repoaudit.FileRef{
		Path: "pkg/service.go", BlobSHA: strings.Repeat("a", 40), SizeBytes: 120,
		Category: "code", Mode: "100644",
	}
	plan, err := store.Plan(
		context.Background(),
		"owner/repo",
		"commit-a",
		"inventory-a",
		[]repoaudit.FileRef{file},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	line := 12
	result, err := store.Record(context.Background(), repoaudit.RecordRequest{
		Plan: plan, RunID: "api-run",
		Observations: []repoaudit.Observation{{
			Model: "provider/review-model", ModelAlias: "review-model", Account: "api",
			ScopeFiles: []repoaudit.FileRef{file},
			Findings: []repoaudit.FindingCandidate{
				{
					Severity: "high",
					Title:    "Lost update",
					File:     file.Path,
					Line:     &line,
					Message:  "The update is not fenced.",
					Evidence: "Two writers overwrite each other.",
					Impact:   "Data is lost.",
					Validation: repoaudit.Validation{
						Status:  "confirmed",
						Summary: "Reproduced",
						Checks:  []string{"race test"},
					},
				},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.State
}

func completeRepositoryReviewAPIMappingJobs(
	t *testing.T,
	workspace string,
	state repoaudit.RepositoryState,
) repoaudit.RepositoryState {
	t.Helper()
	store := repoaudit.NewStore(workspace)
	for _, pending := range state.MappingJobs {
		if pending.State != repoaudit.RepositoryMappingPending {
			continue
		}
		_, job, _, claimed, err := store.ClaimMappingJob(
			state.Repository,
			pending.ID,
			repoaudit.RepositoryMappingModelSnapshot{},
		)
		if err != nil || !claimed {
			t.Fatalf("claim mapping job %q: claimed=%v err=%v", pending.ID, claimed, err)
		}
		state, _, err = store.CompleteMappingJob(state.Repository, repoaudit.RepositoryMappingCompletion{
			JobID:                 job.ID,
			CreateMatchState:      repoaudit.RepositoryMatchNew,
			DefaultBranchVerified: true,
		})
		if err != nil {
			t.Fatalf("complete mapping job %q: %v", job.ID, err)
		}
	}
	return state
}

func setRepositoryReviewMutationHeaders(request *http.Request) {
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
}
