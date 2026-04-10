package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

func init() {
	RegisterRegistryProviderBuilder("github", func(_ string, cfg config.SkillRegistryConfig) RegistryProvider {
		privateCfg := githubRegistryPrivateConfig{}
		if err := cfg.DecodeParam(&privateCfg); err != nil {
			slog.Warn("invalid github private config", "error", err)
		}
		return GitHubRegistryConfig{
			Enabled:   cfg.Enabled,
			BaseURL:   cfg.BaseURL,
			AuthToken: cfg.AuthToken.String(),
			Proxy:     privateCfg.Proxy,
		}
	})
}

type githubRegistryPrivateConfig struct {
	Proxy string `json:"proxy"`
}

type GitHubRegistryConfig struct {
	Enabled   bool
	BaseURL   string
	AuthToken string
	Proxy     string
}

type GitHubRegistry struct {
	installer *SkillInstaller
	webBase   string
}

const githubAuthTokenHelp = "configure registries.github.auth_token"

func (c GitHubRegistryConfig) IsEnabled() bool {
	return c.Enabled
}

func (c GitHubRegistryConfig) BuildRegistry() SkillRegistry {
	installer, err := NewSkillInstallerWithBaseURL("", c.BaseURL, c.AuthToken, c.Proxy)
	if err != nil {
		slog.Warn("failed to create github registry installer", "error", err)
		return nil
	}
	return &GitHubRegistry{
		installer: installer,
		webBase:   installer.githubBaseURL,
	}
}

func (r *GitHubRegistry) Name() string {
	return "github"
}

func (r *GitHubRegistry) ResolveInstallDirName(target string) (string, error) {
	return githubInstallDirNameWithBaseURL(target, r.webBase)
}

func (r *GitHubRegistry) SkillURL(target, version string) string {
	defaultRef := strings.TrimSpace(version)
	ref, err := parseGitHubRefWithBaseURL(target, r.webBase, defaultRef)
	if err != nil {
		return ""
	}
	base := strings.TrimRight(r.webBase, "/")
	urlPath := path.Join(ref.Owner, ref.RepoName)
	if ref.Ref == "" {
		return fmt.Sprintf("%s/%s", base, urlPath)
	}
	if ref.SubPath != "" {
		return fmt.Sprintf("%s/%s/tree/%s/%s", base, urlPath, url.PathEscape(ref.Ref), ref.SubPath)
	}
	if ref.Ref != "main" {
		return fmt.Sprintf("%s/%s/tree/%s", base, urlPath, url.PathEscape(ref.Ref))
	}
	return fmt.Sprintf("%s/%s", base, urlPath)
}

type gitHubCodeSearchResponse struct {
	Items []gitHubCodeSearchItem `json:"items"`
}

type gitHubCodeSearchItem struct {
	Path       string  `json:"path"`
	HTMLURL    string  `json:"html_url"`
	Score      float64 `json:"score"`
	Repository struct {
		FullName      string `json:"full_name"`
		Name          string `json:"name"`
		Description   string `json:"description"`
		DefaultBranch string `json:"default_branch"`
	} `json:"repository"`
}

func (r *GitHubRegistry) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5
	}

	u, err := url.Parse(strings.TrimRight(r.installer.githubAPIBaseURL, "/") + "/search/code")
	if err != nil {
		return nil, fmt.Errorf("invalid github api base url: %w", err)
	}
	q := u.Query()
	q.Set("q", fmt.Sprintf("%s filename:SKILL.md", query))
	q.Set("per_page", fmt.Sprintf("%d", limit))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if r.installer.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+r.installer.githubToken)
	}

	resp, err := r.installer.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read github search response: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized && r.installer.githubToken == "" && isGitHubAuthRequiredError(body) {
		slog.Warn("github search requires authentication; returning no results", "help", githubAuthTokenHelp)
		return []SearchResult{}, nil
	}
	if resp.StatusCode == http.StatusForbidden && r.installer.githubToken == "" && isGitHubRateLimitError(body) {
		slog.Warn("github search hit unauthenticated rate limit; returning no results", "help", githubAuthTokenHelp)
		return []SearchResult{}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github search failed: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var parsed gitHubCodeSearchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse github search response: %w", err)
	}

	resultsBySlug := map[string]SearchResult{}
	for _, item := range parsed.Items {
		slug, ok := githubSearchSlug(item)
		if !ok {
			continue
		}
		result := SearchResult{
			Score:        item.Score,
			Slug:         slug,
			DisplayName:  githubSearchDisplayName(item),
			Summary:      strings.TrimSpace(item.Repository.Description),
			Version:      strings.TrimSpace(item.Repository.DefaultBranch),
			RegistryName: r.Name(),
		}
		if existing, exists := resultsBySlug[slug]; exists && existing.Score >= result.Score {
			continue
		}
		resultsBySlug[slug] = result
	}

	results := make([]SearchResult, 0, len(resultsBySlug))
	for _, result := range resultsBySlug {
		results = append(results, result)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Slug < results[j].Slug
		}
		return results[i].Score > results[j].Score
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func isGitHubRateLimitError(body []byte) bool {
	message := strings.ToLower(string(body))
	return strings.Contains(message, "rate limit exceeded")
}

func isGitHubAuthRequiredError(body []byte) bool {
	message := strings.ToLower(string(body))
	return strings.Contains(message, "requires authentication") ||
		strings.Contains(message, "must be authenticated to access the code search api")
}

func githubSearchSlug(item gitHubCodeSearchItem) (string, bool) {
	fullName := strings.TrimSpace(item.Repository.FullName)
	if fullName == "" {
		return "", false
	}
	cleanPath := strings.Trim(strings.TrimSpace(item.Path), "/")
	if cleanPath == "" || filepath.Base(cleanPath) != "SKILL.md" {
		return "", false
	}
	dir := path.Dir(cleanPath)
	if dir == "." || dir == "" {
		return fullName, true
	}
	return fullName + "/" + dir, true
}

func githubSearchDisplayName(item gitHubCodeSearchItem) string {
	cleanPath := strings.Trim(strings.TrimSpace(item.Path), "/")
	if cleanPath != "" {
		dir := path.Dir(cleanPath)
		if dir != "." && dir != "" {
			return path.Base(dir)
		}
	}
	if name := strings.TrimSpace(item.Repository.Name); name != "" {
		return name
	}
	return strings.TrimSpace(item.Repository.FullName)
}

func canonicalGitHubRegistrySlugWithBaseURL(target, githubBaseURL string) (string, error) {
	ref, err := parseGitHubRefWithBaseURL(target, githubBaseURL, "")
	if err != nil {
		return "", err
	}
	slug := path.Join(ref.Owner, ref.RepoName)
	if ref.SubPath != "" {
		slug = path.Join(slug, ref.SubPath)
	}
	return slug, nil
}

func (r *GitHubRegistry) GetSkillMeta(ctx context.Context, target string) (*SkillMeta, error) {
	slug, err := canonicalGitHubRegistrySlugWithBaseURL(target, r.webBase)
	if err != nil {
		return nil, err
	}
	ref, err := parseGitHubRefWithBaseURL(target, r.webBase, "")
	if err != nil {
		return nil, err
	}
	if ref.Ref == "" {
		ref.Ref, err = r.installer.fetchDefaultBranch(ctx, ref.Owner, ref.RepoName)
		if err != nil {
			return nil, err
		}
	}
	return &SkillMeta{
		Slug:          slug,
		DisplayName:   ref.RepoName,
		LatestVersion: ref.Ref,
		RegistryName:  r.Name(),
	}, nil
}

func (r *GitHubRegistry) DownloadAndInstall(
	ctx context.Context,
	target, version, targetDir string,
) (*InstallResult, error) {
	return r.installer.InstallFromGitHubToDir(ctx, target, version, targetDir)
}
