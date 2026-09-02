package api

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestRepositoryReviewLauncherRoutesRemainDocumented(t *testing.T) {
	t.Parallel()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve repository review route test path")
	}
	apiRoot := filepath.Dir(source)
	repositoryRoot := filepath.Clean(filepath.Join(apiRoot, "..", "..", ".."))
	documentation, err := os.ReadFile(filepath.Join(
		repositoryRoot, "docs", "reference", "repository-reviews-api.md",
	))
	if err != nil {
		t.Fatal(err)
	}
	routePattern := regexp.MustCompile(
		`"((?:GET|POST|PATCH|DELETE) /api/repository-reviews[^"\s]+)"`,
	)
	registered := make(map[string]struct{})
	for _, name := range []string{
		"repository_review_automations.go",
		"repository_review_profiles.go",
		"repository_reviews.go",
	} {
		data, readErr := os.ReadFile(filepath.Join(apiRoot, name))
		if readErr != nil {
			t.Fatal(readErr)
		}
		for _, match := range routePattern.FindAllStringSubmatch(string(data), -1) {
			registered[normalizeRepositoryReviewDocumentedRoute(match[1])] = struct{}{}
		}
	}
	for route := range registered {
		if !strings.Contains(string(documentation), "`"+route+"`") {
			t.Errorf("registered repository-review route %q is absent from docs/reference/repository-reviews-api.md", route)
		}
	}
}

func normalizeRepositoryReviewDocumentedRoute(route string) string {
	route = strings.Replace(route, "/api/repository-reviews", "", 1)
	replacements := []struct{ from, to string }{
		{"{automation_id}", "{aid}"},
		{"{profile_id}", "{pid}"},
		{"{repository_finding_id}", "{rfid}"},
		{"{finding_id}", "{fid}"},
		{"{draft_id}", "{did}"},
		{"{source_id}", "{sid}"},
		{"{campaign_id}", "{cid}"},
	}
	for _, replacement := range replacements {
		route = strings.ReplaceAll(route, replacement.from, replacement.to)
	}
	route = strings.ReplaceAll(route, "/repository-findings/{fid}", "/repository-findings/{rfid}")
	if strings.HasSuffix(route, "/{repository_id}/{legacy_action...}") {
		return "POST /{repository_id}/issue-drafts/{did}/publish"
	}
	return route
}
