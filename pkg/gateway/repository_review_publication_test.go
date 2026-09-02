package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/repoaudit"
	"github.com/sipeed/picoclaw/pkg/reviews"
	"github.com/sipeed/picoclaw/pkg/workflows"
)

func TestRepositoryReviewAutomationIssueLinkRouteParsingIsExact(t *testing.T) {
	for _, test := range []struct {
		method string
		path   string
		action string
	}{
		{
			method: http.MethodPost,
			path: repositoryReviewPublicationRoute +
				"automations/rra_test/findings/rfn_test/issue-link/candidates",
			action: "candidates",
		},
		{
			method: http.MethodPost,
			path: repositoryReviewPublicationRoute +
				"automations/rra_test/findings/rfn_test/issue-link",
			action: "link",
		},
		{
			method: http.MethodDelete,
			path: repositoryReviewPublicationRoute +
				"automations/rra_test/findings/rfn_test/issue-link",
			action: "link",
		},
	} {
		request := httptest.NewRequest(test.method, test.path, nil)
		operation, ok := repositoryReviewAutomationOperationFromRequest(request)
		if !ok || operation.AutomationID != "rra_test" || operation.FindingID != "rfn_test" ||
			operation.Action != test.action {
			t.Fatalf("operation=%#v ok=%v", operation, ok)
		}
	}
	syncRequest := httptest.NewRequest(
		http.MethodPost,
		repositoryReviewPublicationRoute+
			"automations/rra_test/repository-findings/rrf_test/sync",
		nil,
	)
	syncOperation, ok := repositoryReviewAutomationOperationFromRequest(syncRequest)
	if !ok || syncOperation.AutomationID != "rra_test" ||
		syncOperation.FindingID != "rrf_test" || syncOperation.Action != "sync" {
		t.Fatalf("sync operation=%#v ok=%v", syncOperation, ok)
	}
	for _, target := range []string{
		repositoryReviewPublicationRoute + "automations/rra/findings/rfn/issue-link/extra",
		repositoryReviewPublicationRoute + "automations/rra/findings//issue-link",
		repositoryReviewPublicationRoute + "automations/rra/findings/rfn/issue-link?q=1",
	} {
		if _, ok := repositoryReviewAutomationOperationFromRequest(
			httptest.NewRequest(http.MethodPost, target, nil),
		); ok {
			t.Fatalf("accepted invalid operation route %q", target)
		}
	}
	if _, ok := repositoryReviewAutomationOperationFromRequest(nil); ok {
		t.Fatal("nil operation request was accepted")
	}
	for _, test := range []struct {
		method string
		path   string
	}{
		{
			method: http.MethodGet,
			path: repositoryReviewPublicationRoute +
				"automations/rra/findings/rfn/issue-link/candidates",
		},
		{
			method: http.MethodPut,
			path: repositoryReviewPublicationRoute +
				"automations/rra/findings/rfn/issue-link",
		},
	} {
		if _, ok := repositoryReviewAutomationOperationFromRequest(
			httptest.NewRequest(test.method, test.path, nil),
		); ok {
			t.Fatalf("accepted %s operation route", test.method)
		}
	}
}

func TestDecodeRepositoryReviewGatewayRequestRejectsMissingOversizedAndTrailingBodies(t *testing.T) {
	var destination repositoryReviewIssueCandidateRequest
	if err := decodeRepositoryReviewGatewayRequest(nil, &destination); err == nil {
		t.Fatal("nil request was accepted")
	}
	if err := decodeRepositoryReviewGatewayRequest(&http.Request{}, &destination); err == nil {
		t.Fatal("request without a body was accepted")
	}
	oversized := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"expected_version":1}`))
	oversized.ContentLength = (32 << 10) + 1
	if err := decodeRepositoryReviewGatewayRequest(oversized, &destination); err == nil {
		t.Fatal("oversized request was accepted")
	}
	for _, body := range []string{`{`, `{"expected_version":1}{}`} {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		if err := decodeRepositoryReviewGatewayRequest(request, &destination); err == nil {
			t.Fatalf("invalid body %q was accepted", body)
		}
	}
	request := &http.Request{
		Body: io.NopCloser(strings.NewReader(`{"expected_version":7}`)), ContentLength: -1,
	}
	if err := decodeRepositoryReviewGatewayRequest(request, &destination); err != nil ||
		destination.ExpectedVersion != 7 {
		t.Fatalf("decoded request=%#v err=%v", destination, err)
	}
}

func TestRepositoryReviewIssueCandidateSearchIsDerivedBoundedAndRepositorySafe(t *testing.T) {
	finding := repoaudit.Finding{
		Title: "Concurrent writer loses durable update", Symbol: "Store.Save",
		File: repoaudit.FileRef{Path: "pkg/repoaudit/store.go"},
	}
	queries := repositoryReviewIssueSearchQueries("owner/repo", finding)
	if len(queries) != 3 {
		t.Fatalf("queries=%#v", queries)
	}
	for _, query := range queries {
		if !strings.Contains(query, "repo:owner/repo is:issue") || len(query) > 512 {
			t.Fatalf("unsafe query=%q", query)
		}
	}
	wire := repositoryReviewGitHubIssueCandidateWire{
		Number: json.RawMessage(`12`), ID: json.RawMessage(`99`),
		Title: "Lost update", State: "open",
		HTMLURL: "https://github.com/owner/repo/issues/12",
	}
	candidate, ok := repositoryReviewIssueCandidateFromWire("owner/repo", wire)
	if !ok || candidate.Number != 12 || candidate.ID != "99" {
		t.Fatalf("candidate=%#v ok=%v", candidate, ok)
	}
	wire.HTMLURL = "https://github.com/other/repo/issues/12"
	if _, ok := repositoryReviewIssueCandidateFromWire("owner/repo", wire); ok {
		t.Fatal("cross-repository issue candidate was accepted")
	}
	wire.HTMLURL = "https://github.com/owner/repo/issues/13"
	if _, ok := repositoryReviewIssueCandidateFromWire("owner/repo", wire); ok {
		t.Fatal("issue candidate URL did not match its reported number")
	}
}

func TestRepositoryReviewExistingIssueRequiresExactConfirmedNumber(t *testing.T) {
	_, err := decodeRepositoryReviewExistingIssue(
		[]byte(`{
			"id":99,"number":12,"title":"Mismatched issue",
			"html_url":"https://github.com/owner/repo/issues/13"
		}`),
		"owner/repo",
		12,
	)
	if err == nil {
		t.Fatal("existing issue re-fetch accepted a URL for another issue number")
	}
}

func TestRepositoryReviewIssueAutoLinkPolicyBoundaries(t *testing.T) {
	eligible := repositoryReviewIssueCandidate{
		ID: "eligible", Score: repositoryReviewIssueAutoLinkMinimumScore,
		MatchingAnchors: []string{"mechanism", "trigger", "invariant", "outcome"},
	}
	for _, test := range []struct {
		name       string
		candidates []repositoryReviewIssueCandidate
		want       bool
	}{
		{name: "no candidates"},
		{
			name: "score below threshold",
			candidates: []repositoryReviewIssueCandidate{{
				ID: "low-score", Score: repositoryReviewIssueAutoLinkMinimumScore - 1,
				MatchingAnchors: eligible.MatchingAnchors,
			}},
		},
		{
			name: "three matching anchors",
			candidates: []repositoryReviewIssueCandidate{{
				ID: "few-anchors", Score: repositoryReviewIssueAutoLinkMinimumScore,
				MatchingAnchors: eligible.MatchingAnchors[:3],
			}},
		},
		{
			name: "one conflicting anchor",
			candidates: []repositoryReviewIssueCandidate{{
				ID: "conflict", Score: repositoryReviewIssueAutoLinkMinimumScore,
				MatchingAnchors: eligible.MatchingAnchors, ConflictingAnchors: []string{"different trigger"},
			}},
		},
		{
			name: "eligible candidate is not first",
			candidates: []repositoryReviewIssueCandidate{
				{ID: "first", Score: repositoryReviewIssueAutoLinkMinimumScore - 1}, eligible,
			},
		},
		{name: "exact thresholds", candidates: []repositoryReviewIssueCandidate{eligible}, want: true},
		{
			name: "above thresholds",
			candidates: []repositoryReviewIssueCandidate{{
				ID: "above", Score: 100,
				MatchingAnchors: append(append([]string(nil), eligible.MatchingAnchors...), "location"),
			}},
			want: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate, ok := repositoryReviewIssueAutoLinkCandidate(test.candidates)
			if ok != test.want {
				t.Fatalf("candidate=%#v eligible=%v, want %v", candidate, ok, test.want)
			}
			if len(test.candidates) > 0 && candidate.ID != test.candidates[0].ID {
				t.Fatalf("selected candidate=%q, want first-ranked %q", candidate.ID, test.candidates[0].ID)
			}
		})
	}
}

func TestRepositoryReviewIssueCandidateRankerRequestUsesSnapshotAndPrivateBoundary(t *testing.T) {
	request, err := repositoryReviewIssueCandidateAgentRequest(
		repoaudit.RepositoryReviewAutomation{IssueWriterModel: "writer-snapshot"},
		repoaudit.Finding{
			Title: "Lost update", File: repoaudit.FileRef{Path: "pkg/store.go"},
			Evidence: "A stale write overwrites a committed value.", Impact: "Data is lost.",
		},
		[]repositoryReviewIssueCandidate{{
			ID: "12", Number: 12, Title: "Concurrent save loses data",
			URL: "https://github.com/owner/repo/issues/12", body: "same failure mechanism",
		}},
		"account-snapshot",
	)
	if err != nil || request.Model != "writer-snapshot" ||
		request.AccountRef != "account-snapshot" || !request.EphemeralSession ||
		request.History != "none" || request.Cache != "none" ||
		request.Tools != workflows.AgentToolsNone || !request.PrivateContext ||
		request.IsolatedSystemPrompt != repositoryReviewIssueCandidateSystemPrompt ||
		request.Output == nil || request.Output.Format != "json" ||
		request.Output.Schema["additionalProperties"] != false {
		t.Fatalf("candidate ranker request=%#v err=%v", request, err)
	}
}

func TestRepositoryReviewIssueCandidateRankerRequestPropagatesEncodingFailure(t *testing.T) {
	want := errors.New("encoding unavailable")
	_, err := repositoryReviewIssueCandidateAgentRequestWithMarshal(
		repoaudit.RepositoryReviewAutomation{IssueWriterModel: "writer"},
		repoaudit.Finding{Title: "Finding"},
		[]repositoryReviewIssueCandidate{{ID: "1", Number: 1, Title: "Candidate"}},
		"writer-account",
		func(any) ([]byte, error) { return nil, want },
	)
	if !errors.Is(err, want) {
		t.Fatalf("encoding error=%v", err)
	}
}

func TestRepositoryReviewIssueCandidateRankingDecoderRejectsInvalidInternalShapes(t *testing.T) {
	candidates := []repositoryReviewIssueCandidate{{ID: "1", Number: 1, Title: "Candidate"}}
	tests := []struct {
		name    string
		outputs map[string]any
	}{
		{name: "not validated", outputs: map[string]any{"structured_valid": false}},
		{
			name:    "not JSON encodable",
			outputs: map[string]any{"structured_valid": true, "structured": make(chan int)},
		},
		{
			name: "unknown field",
			outputs: map[string]any{
				"structured_valid": true,
				"structured":       map[string]any{"rankings": []any{}, "extra": true},
			},
		},
		{
			name: "more than ten",
			outputs: map[string]any{
				"structured_valid": true,
				"structured": map[string]any{
					"rankings": func() []any {
						values := make([]any, 11)
						for index := range values {
							values[index] = map[string]any{
								"id": "1", "score": 1, "explanation": "same",
							}
						}
						return values
					}(),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeRepositoryReviewIssueCandidateRankings(
				test.outputs, candidates,
			); err == nil {
				t.Fatal("invalid ranker output was accepted")
			}
		})
	}
}

func TestRepositoryReviewIssueCandidateWriterFailsClosedAfterAliasDrift(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Agents.Defaults.ModelName = "writer"
	cfg.Agents.Defaults.AccountRef = "api"
	cfg.ModelList = []*config.ModelConfig{{
		ModelName: "api", Provider: "codex-cli", Model: "codex-cli/gpt-5", Enabled: true,
	}}
	cfg.ModelAliases = []config.ModelAliasConfig{{
		Name: "writer", Model: "codex-cli/gpt-5",
	}}
	messageBus := bus.NewMessageBus()
	loop := agent.NewAgentLoop(cfg, messageBus, &startupBlockedProvider{reason: "not used"})
	t.Cleanup(func() {
		loop.Stop()
		messageBus.Close()
		loop.Close()
	})
	_, err := repositoryReviewValidatedIssueWriterAccount(
		t.Context(),
		loop,
		repoaudit.RepositoryReviewAutomation{
			IssueWriterModel: "writer", EffectiveAccountRef: "api",
		},
	)
	if err == nil || !strings.Contains(err.Error(), "agentic CLI") {
		t.Fatalf("agentic issue candidate writer was not rejected: %v", err)
	}
}

type repositoryReviewNonResolverAgentRunner struct{}

func (repositoryReviewNonResolverAgentRunner) RunAgent(
	context.Context,
	workflows.AgentRequest,
) (map[string]any, error) {
	return nil, errors.New("not used")
}

func TestRepositoryReviewIssueWriterValidationRequiresProfileResolver(t *testing.T) {
	_, err := repositoryReviewValidateIssueWriterAccount(
		t.Context(), repositoryReviewNonResolverAgentRunner{}, "writer-account", "issue-writer",
	)
	if err == nil || !strings.Contains(err.Error(), "validation is unavailable") {
		t.Fatalf("writer validation error=%v", err)
	}
}

func TestCreateRepositoryReviewIssueUsesFrozenDraftAndStableMarker(t *testing.T) {
	var request workflows.ToolRequest
	provider, err := reviews.NewGitHubProvider(prWorkspaceProviderToolRunnerFunc(func(
		_ context.Context,
		input workflows.ToolRequest,
	) (map[string]any, error) {
		request = input
		return map[string]any{"text": `{"id":12,"html_url":"https://github.com/owner/repo/issues/12"}`}, nil
	}), "")
	if err != nil {
		t.Fatal(err)
	}
	draft := repoaudit.IssueDraft{
		ID: "rid_test", Repository: "owner/repo", Title: "Validated bug",
		Body: "Exact finding body", Labels: []string{"bug", "concurrency"},
	}
	marker := repositoryReviewIssueMarker(draft.ID)
	result, err := createRepositoryReviewIssue(t.Context(), provider, draft.Repository, draft, marker)
	if err != nil || result.ExternalID != "12" || result.ExternalURL != "https://github.com/owner/repo/issues/12" {
		t.Fatalf("create result=%#v err=%v", result, err)
	}
	if request.MCPTool != reviews.GitHubIssueWriteTool || request.Args["title"] != draft.Title ||
		!strings.Contains(request.Args["body"].(string), "<!-- "+marker+" -->") ||
		!reflect.DeepEqual(request.Args["labels"], []any{"bug", "concurrency"}) {
		t.Fatalf("issue request=%#v", request)
	}
}

func TestFindRepositoryReviewIssueRecoversOnlyExactRepositoryMarker(t *testing.T) {
	provider, err := reviews.NewGitHubProvider(prWorkspaceProviderToolRunnerFunc(func(
		_ context.Context,
		input workflows.ToolRequest,
	) (map[string]any, error) {
		if input.MCPTool != reviews.GitHubSearchIssuesTool {
			t.Fatalf("tool=%q", input.MCPTool)
		}
		return map[string]any{"text": `{"items":[` +
			`{"id":9,"html_url":"https://github.com/foreign/repo/issues/9","body":"picoclaw-repository-review:rid_test"},` +
			`{"id":12,"html_url":"https://github.com/owner/repo/issues/12","body":"picoclaw-repository-review:rid_test"}` +
			`]}`}, nil
	}), "")
	if err != nil {
		t.Fatal(err)
	}
	result, found, err := findRepositoryReviewIssue(
		t.Context(), provider, "owner/repo", "picoclaw-repository-review:rid_test",
	)
	if err != nil || !found || result.ExternalID != "12" {
		t.Fatalf("find result=%#v found=%v err=%v", result, found, err)
	}
}

func TestRepositoryReviewPublicationRouteParsingIsExact(t *testing.T) {
	valid := httptest.NewRequest(
		http.MethodPost,
		repositoryReviewPublicationRoute+"rrp_test/issue-drafts/rid_test/publish",
		strings.NewReader(`{"expected_version":1}`),
	)
	repositoryID, draftID, ok := repositoryReviewPublicationRouteIDs(valid)
	if !ok || repositoryID != "rrp_test" || draftID != "rid_test" {
		t.Fatalf("route IDs=%q/%q ok=%v", repositoryID, draftID, ok)
	}
	for _, target := range []string{
		repositoryReviewPublicationRoute + "rrp_test/issue-drafts/rid_test",
		repositoryReviewPublicationRoute + "rrp_test/issue-drafts/rid_test/publish/extra",
		repositoryReviewPublicationRoute + "rrp_test/issue-drafts//publish",
	} {
		request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(`{}`))
		if _, _, accepted := repositoryReviewPublicationRouteIDs(request); accepted {
			t.Fatalf("accepted invalid route %q", target)
		}
	}
}

func TestRepositoryReviewIssueCreateAmbiguityIsStatusAware(t *testing.T) {
	if repositoryReviewIssueCreateAmbiguous(workflows.ErrToolCallNotDispatched) {
		t.Fatal("pre-dispatch error became ambiguous")
	}
	if repositoryReviewIssueCreateAmbiguous(errors.New("API error status: 403 forbidden")) {
		t.Fatal("definitive provider rejection became ambiguous")
	}
	if repositoryReviewIssueCreateAmbiguous(errors.New("GitHub issue create status: 422 validation failed")) {
		t.Fatal("definitive 422 validation rejection became ambiguous")
	}
	if !repositoryReviewIssueCreateAmbiguous(errors.New("connection reset by peer")) {
		t.Fatal("transport error was treated as definite")
	}
	if !repositoryReviewIssueCreateAmbiguous(errors.New("GitHub issue create status: 429 rate limit exceeded")) {
		t.Fatal("post-dispatch rate limit was treated as definite")
	}
	if !repositoryReviewIssueCreateAmbiguous(errors.New("GitHub issue create status: 503 overloaded")) {
		t.Fatal("post-dispatch overload was treated as definite")
	}
}

func TestRepositoryReviewGitHubIdentityRequiresCanonicalDerivedShape(t *testing.T) {
	for _, value := range []string{"owner/repo", "owner/repo.name", "owner/repo_name"} {
		if !validRepositoryReviewGitHubIdentity(value) {
			t.Fatalf("rejected canonical identity %q", value)
		}
	}
	for _, value := range []string{
		"Owner/Repo", "/tmp/repo", "https://github.com/owner/repo", "owner/repo/extra",
		"foo_bar/foo_bar",
	} {
		if validRepositoryReviewGitHubIdentity(value) {
			t.Fatalf("accepted unverified identity %q", value)
		}
	}
}

func TestRepositoryReviewGatewayLedgerIdentityNormalizationIsExact(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "git@github.com:Owner/Repo.git", want: "owner/repo"},
		{input: "ssh://git@github.com/Owner/Repo.git", want: "owner/repo"},
		{input: "https://github.com/Owner/Repo.git", want: "owner/repo"},
		{input: "Owner/Repo.git", want: "owner/repo"},
		{input: "git@example.com:owner/repo.git"},
		{input: "https://example.com/owner/repo"},
		{input: "owner/repo/extra"},
	} {
		if got := repositoryReviewGatewayGitHubIdentity(test.input); got != test.want {
			t.Fatalf("identity(%q)=%q, want %q", test.input, got, test.want)
		}
	}
	if identities := repositoryReviewGatewayLedgerIdentities(" "); identities != nil {
		t.Fatalf("empty identities=%#v", identities)
	}
	absolute := filepath.Join(string(filepath.Separator), "tmp", "repo")
	dirtyAbsolute := filepath.Join(string(filepath.Separator), "tmp", "..", "tmp", "repo")
	if identities := repositoryReviewGatewayLedgerIdentities(
		" " + dirtyAbsolute + " ",
	); !reflect.DeepEqual(identities, []string{absolute}) {
		t.Fatalf("absolute identities=%#v", identities)
	}
	if identities := repositoryReviewGatewayLedgerIdentities(
		"https://github.com/Owner/Repo.git",
	); !reflect.DeepEqual(
		identities,
		[]string{"owner/repo", "https://github.com/Owner/Repo.git"},
	) {
		t.Fatalf("GitHub identities=%#v", identities)
	}
	if identities := repositoryReviewGatewayLedgerIdentities(
		"opaque-repository",
	); !reflect.DeepEqual(identities, []string{"opaque-repository"}) {
		t.Fatalf("opaque identities=%#v", identities)
	}
}

func TestRepositoryReviewIssueSearchDecodingAcceptsOnlySupportedWireShapes(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantCount int
		wantError bool
	}{
		{name: "envelope", raw: `{"items":[{"number":1}]}`, wantCount: 1},
		{name: "null envelope", raw: `{"items":null}`},
		{name: "direct", raw: `[{"number":2}]`, wantCount: 1},
		{name: "invalid envelope items", raw: `{"items":42}`, wantError: true},
		{name: "unsupported object", raw: `{}`, wantError: true},
		{name: "malformed", raw: `{`, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issues, err := decodeRepositoryReviewIssueSearch([]byte(test.raw))
			if (err != nil) != test.wantError || len(issues) != test.wantCount {
				t.Fatalf("issues=%#v err=%v", issues, err)
			}
			if test.name == "null envelope" && issues == nil {
				t.Fatal("null items did not normalize to an empty collection")
			}
		})
	}
}

func TestRepositoryReviewIssueCandidateNormalizationIsBounded(t *testing.T) {
	labels := make([]struct {
		Name string `json:"name"`
	}, 24)
	for index := range labels {
		labels[index].Name = fmt.Sprintf(" label-%02d ", index)
	}
	labels[3].Name = strings.Repeat("x", 51)
	labels[4].Name = " "
	body := strings.Repeat("a", 2047) + "é" + strings.Repeat("z", 20)
	candidate, ok := repositoryReviewIssueCandidateFromWire(
		"owner/repo",
		repositoryReviewGitHubIssueCandidateWire{
			Number: json.RawMessage(`"17"`), Title: "  bounded issue  ",
			URL: "https://github.com/owner/repo/issues/17", State: " OPEN ",
			Body: body, Labels: labels,
		},
	)
	if !ok || candidate.ID != "17" || candidate.Title != "bounded issue" ||
		candidate.State != "open" || len(candidate.Labels) != 20 ||
		!strings.HasSuffix(candidate.body, " [truncated]") ||
		!utf8.ValidString(candidate.body) {
		t.Fatalf("candidate=%#v ok=%v", candidate, ok)
	}

	for _, wire := range []repositoryReviewGitHubIssueCandidateWire{
		{Number: json.RawMessage(`0`), Title: "title", HTMLURL: "https://github.com/owner/repo/issues/0"},
		{Number: json.RawMessage(`1`), Title: " ", HTMLURL: "https://github.com/owner/repo/issues/1"},
		{Number: json.RawMessage(`1`), Title: "title"},
		{Number: json.RawMessage(`not-json`), Title: "title", HTMLURL: "https://github.com/owner/repo/issues/1"},
	} {
		if _, accepted := repositoryReviewIssueCandidateFromWire("owner/repo", wire); accepted {
			t.Fatalf("accepted malformed candidate %#v", wire)
		}
	}
}

func TestRepositoryReviewIssueSearchTermsAndQueriesAreStrictlyBounded(t *testing.T) {
	terms := repositoryReviewIssueSearchTerms(
		"a Alpha ALPHA beta gamma delta epsilon zeta eta theta iota " + strings.Repeat("x", 65),
	)
	if len(terms) != 8 || terms[0] != "Alpha" || terms[7] != "theta" {
		t.Fatalf("terms=%#v", terms)
	}
	pathTerms := repositoryReviewIssueSearchTerms("pkg/repoaudit/store.go")
	if !reflect.DeepEqual(pathTerms, []string{"pkg", "repoaudit", "store.go"}) {
		t.Fatalf("path terms=%#v", pathTerms)
	}
	queries := repositoryReviewIssueSearchQueries("owner/repo", repoaudit.Finding{
		Title: " ", Symbol: "same", File: repoaudit.FileRef{Path: "same"},
	})
	if len(queries) != 2 || !strings.Contains(queries[0], "in:title,body") ||
		!strings.Contains(queries[1], "in:body") {
		t.Fatalf("queries=%#v", queries)
	}
}

func TestSearchRepositoryReviewIssueCandidatesMergesDeduplicatesAndRejectsUnsafeResults(t *testing.T) {
	finding := repoaudit.Finding{
		Title: "Lost update", Symbol: "Store.Save",
		File: repoaudit.FileRef{Path: "pkg/repoaudit/store.go"},
	}
	responses := []string{
		`{"items":[` +
			`{"id":101,"number":1,"title":"first","html_url":"https://github.com/owner/repo/issues/1"},` +
			`{"id":999,"number":9,"title":"foreign","html_url":"https://github.com/other/repo/issues/9"},` +
			`{"id":104,"number":4,"title":"mismatch","html_url":"https://github.com/owner/repo/issues/5"}` +
			`]}`,
		`[{"id":101,"number":1,"title":"duplicate","html_url":"https://github.com/owner/repo/issues/1"},` +
			`{"id":102,"number":2,"title":"second","html_url":"https://github.com/owner/repo/issues/2"}]`,
		`{"items":[{"id":103,"number":3,"title":"third","html_url":"https://github.com/owner/repo/issues/3"}]}`,
	}
	calls := 0
	provider, err := reviews.NewGitHubProvider(prWorkspaceProviderToolRunnerFunc(func(
		_ context.Context,
		request workflows.ToolRequest,
	) (map[string]any, error) {
		if request.MCPTool != reviews.GitHubSearchIssuesTool || request.Args["page"] != 1 ||
			request.Args["perPage"] != 50 || !strings.Contains(request.Args["query"].(string), "repo:owner/repo") {
			t.Fatalf("search request=%#v", request)
		}
		response := responses[calls]
		calls++
		return map[string]any{"text": response}, nil
	}), "")
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := searchRepositoryReviewIssueCandidates(t.Context(), provider, "owner/repo", finding)
	if err != nil || calls != 3 || len(candidates) != 3 || candidates[0].Number != 1 ||
		candidates[1].Number != 2 || candidates[2].Number != 3 {
		t.Fatalf("candidates=%#v calls=%d err=%v", candidates, calls, err)
	}

	for _, test := range []struct {
		name   string
		result map[string]any
		err    error
	}{
		{name: "provider failure", err: errors.New("search unavailable")},
		{name: "invalid response", result: map[string]any{"text": `{"items":42}`}},
	} {
		t.Run(test.name, func(t *testing.T) {
			failing, providerErr := reviews.NewGitHubProvider(prWorkspaceProviderToolRunnerFunc(func(
				context.Context,
				workflows.ToolRequest,
			) (map[string]any, error) {
				return test.result, test.err
			}), "")
			if providerErr != nil {
				t.Fatal(providerErr)
			}
			if _, searchErr := searchRepositoryReviewIssueCandidates(
				t.Context(), failing, "owner/repo", finding,
			); searchErr == nil {
				t.Fatal("search succeeded with an unsafe provider result")
			}
		})
	}
}

func TestSearchRepositoryReviewIssueCandidatesStopsAtFifty(t *testing.T) {
	items := make([]repositoryReviewGitHubIssueCandidateWire, 55)
	for index := range items {
		number := index + 1
		items[index] = repositoryReviewGitHubIssueCandidateWire{
			ID:     json.RawMessage(fmt.Sprintf("%d", 1000+number)),
			Number: json.RawMessage(fmt.Sprintf("%d", number)), Title: fmt.Sprintf("issue %d", number),
			HTMLURL: fmt.Sprintf("https://github.com/owner/repo/issues/%d", number),
		}
	}
	payload, err := json.Marshal(map[string]any{"items": items})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	provider, err := reviews.NewGitHubProvider(prWorkspaceProviderToolRunnerFunc(func(
		context.Context,
		workflows.ToolRequest,
	) (map[string]any, error) {
		calls++
		return map[string]any{"text": string(payload)}, nil
	}), "")
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := searchRepositoryReviewIssueCandidates(
		t.Context(), provider, "owner/repo", repoaudit.Finding{
			Title: "title", Symbol: "symbol", File: repoaudit.FileRef{Path: "path.go"},
		},
	)
	if err != nil || len(candidates) != 50 || calls != 1 || candidates[49].Number != 50 {
		t.Fatalf("candidates=%d calls=%d last=%#v err=%v", len(candidates), calls, candidates[49], err)
	}
}

func TestRepositoryReviewExistingIssueAndExcerptTruncateAtUTF8Boundaries(t *testing.T) {
	body := strings.Repeat("a", (60<<10)-1) + "é" + strings.Repeat("z", 10)
	raw, err := json.Marshal(map[string]any{
		"id": 22, "number": 22, "title": "Existing issue", "body": body,
		"html_url": "https://github.com/owner/repo/issues/22",
	})
	if err != nil {
		t.Fatal(err)
	}
	issue, err := decodeRepositoryReviewExistingIssue(raw, "owner/repo", 22)
	if err != nil || len(issue.Body) > 60<<10 || !utf8.ValidString(issue.Body) ||
		strings.Contains(issue.Body, "é") {
		t.Fatalf("issue body length=%d valid=%v err=%v", len(issue.Body), utf8.ValidString(issue.Body), err)
	}
	if _, err := decodeRepositoryReviewExistingIssue([]byte(`{`), "owner/repo", 22); err == nil {
		t.Fatal("malformed exact issue response was accepted")
	}

	excerpt := repositoryReviewCandidateExcerpt(string(bytes.Repeat([]byte("a"), 2047)) + "é-tail")
	if !utf8.ValidString(excerpt) || !strings.HasSuffix(excerpt, " [truncated]") ||
		strings.Contains(excerpt, "é") {
		t.Fatalf("excerpt=%q", excerpt[len(excerpt)-32:])
	}
}

func TestRepositoryReviewEffectiveAccountUsesImmutableFallbackOrder(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Workspace = t.TempDir()
	cfg.Agents.Defaults.AccountRef = " default-account "
	messageBus := bus.NewMessageBus()
	loop := agent.NewAgentLoop(cfg, messageBus, &startupBlockedProvider{reason: "not used"})
	t.Cleanup(func() {
		loop.Stop()
		messageBus.Close()
		loop.Close()
	})
	if got := repositoryReviewEffectiveAutomationAccount(loop, repoaudit.RepositoryReviewAutomation{
		EffectiveAccountRef: " effective ", AccountRef: "profile",
	}); got != "effective" {
		t.Fatalf("effective account=%q", got)
	}
	if got := repositoryReviewEffectiveAutomationAccount(loop, repoaudit.RepositoryReviewAutomation{
		AccountRef: " profile ",
	}); got != "profile" {
		t.Fatalf("profile account=%q", got)
	}
	if got := repositoryReviewEffectiveAutomationAccount(
		loop,
		repoaudit.RepositoryReviewAutomation{},
	); got != "default-account" {
		t.Fatalf("default account=%q", got)
	}
	if got := repositoryReviewEffectiveAutomationAccount(nil, repoaudit.RepositoryReviewAutomation{}); got != "" {
		t.Fatalf("nil-loop account=%q", got)
	}
}
