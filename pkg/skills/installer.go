package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/utils"
)

type SkillInstaller struct {
	workspace string
	// baseURL for GitHub raw content; empty means https://raw.githubusercontent.com
	baseURL string
}

type AvailableSkill struct {
	Name        string   `json:"name"`
	Repository  string   `json:"repository"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
}

func NewSkillInstaller(workspace string) *SkillInstaller {
	return &SkillInstaller{workspace: workspace}
}

// NewSkillInstallerWithBase returns an installer that uses baseURL for raw content (e.g. httptest.Server.URL). Used for testing.
func NewSkillInstallerWithBase(workspace, baseURL string) *SkillInstaller {
	return &SkillInstaller{workspace: workspace, baseURL: baseURL}
}

const defaultBranch = "main"

// ParseInstallSpec parses "owner/repo" or "owner/repo@branch" into repo and branch.
// When @ is absent, branch is "" meaning "use repo default branch" (fetched from GitHub API).
// Repo must contain at least one "/".
func ParseInstallSpec(spec string) (repo, branch string, err error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", "", fmt.Errorf("empty install spec")
	}
	idx := strings.LastIndex(spec, "@")
	if idx >= 0 {
		repo = strings.TrimSpace(spec[:idx])
		branch = strings.TrimSpace(spec[idx+1:])
		if branch == "" {
			return "", "", fmt.Errorf("branch name after @ is empty")
		}
	} else {
		repo = spec
		branch = "" // empty = resolve default branch via API
	}
	if repo == "" || !strings.Contains(repo, "/") {
		return "", "", fmt.Errorf("repo must be owner/repo (got %q)", spec)
	}
	return repo, branch, nil
}

// githubRepoResponse is the minimal GitHub API response for GET /repos/owner/repo.
type githubRepoResponse struct {
	DefaultBranch string `json:"default_branch"`
}

// githubTreeEntry is one entry from GET /repos/owner/repo/git/trees/branch?recursive=1.
type githubTreeEntry struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
}

// githubTreeResponse is the response for Git Trees API.
type githubTreeResponse struct {
	SHA        string           `json:"sha"`
	Tree       []githubTreeEntry `json:"tree"`
	Truncated  bool             `json:"truncated"`
}

// fetchDefaultBranch returns the default branch for repo "owner/repo" via GitHub API.
// On failure (network, 404, rate limit) returns defaultBranch "main" and nil error so install can still be tried.
func fetchDefaultBranch(ctx context.Context, repo string) (string, error) {
	apiURL := "https://api.github.com/repos/" + repo
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return defaultBranch, nil
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := client.Do(req)
	if err != nil {
		return defaultBranch, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return defaultBranch, nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return defaultBranch, nil
	}
	var v githubRepoResponse
	if err := json.Unmarshal(body, &v); err != nil {
		return defaultBranch, nil
	}
	if v.DefaultBranch == "" {
		return defaultBranch, nil
	}
	return v.DefaultBranch, nil
}

// fetchTree returns blob paths under the given path prefix. prefix "" means repo root (all files).
// Branch must be resolved (e.g. main or from fetchDefaultBranch).
func fetchTree(ctx context.Context, repo, branch, pathPrefix string) ([]string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/git/trees/%s?recursive=1", repo, branch)
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch tree: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tr githubTreeResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, err
	}
	if tr.Truncated {
		return nil, fmt.Errorf("repository tree is too large (truncated)")
	}
	var paths []string
	for _, e := range tr.Tree {
		if e.Type != "blob" {
			continue
		}
		if pathPrefix == "" {
			paths = append(paths, e.Path)
			continue
		}
		prefix := pathPrefix + "/"
		if e.Path == pathPrefix || strings.HasPrefix(e.Path, prefix) {
			paths = append(paths, e.Path)
		}
	}
	return paths, nil
}

// validateSubpath ensures subpath is safe (no ".." or absolute path).
// Multi-segment paths like "skills/kanban-ai" are allowed.
func validateSubpath(subpath string) error {
	subpath = strings.TrimSpace(subpath)
	if subpath == "" {
		return nil
	}
	if strings.Contains(subpath, "..") {
		return fmt.Errorf("subpath must not contain ..")
	}
	cleaned := filepath.Clean(subpath)
	if cleaned != subpath || filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "..") {
		return fmt.Errorf("invalid subpath %q", subpath)
	}
	return nil
}

// InstallFromGitHubEx installs a skill from GitHub. repo is "owner/repo". When branch is empty,
// the repo's default branch is fetched from GitHub API; subpath is optional (e.g. "skills/kanban-ai").
// If force is true, an existing skill directory is removed before install. Returns the installed skill name.
func (si *SkillInstaller) InstallFromGitHubEx(ctx context.Context, repo, branch, subpath string, force bool) (skillName string, err error) {
	repo = strings.TrimSpace(repo)
	branch = strings.TrimSpace(branch)
	if branch == "" {
		if si.baseURL == "" {
			branch, _ = fetchDefaultBranch(ctx, repo)
		}
		if branch == "" {
			branch = defaultBranch
		}
	}
	if err := validateSubpath(subpath); err != nil {
		return "", err
	}

	if subpath != "" {
		skillName = filepath.Base(filepath.Clean(subpath))
	} else {
		skillName = filepath.Base(repo)
	}
	skillDir := filepath.Join(si.workspace, "skills", skillName)

	if _, err := os.Stat(skillDir); err == nil {
		if !force {
			return "", fmt.Errorf("skill '%s' already exists (use reinstall to overwrite)", skillName)
		}
		if err := os.RemoveAll(skillDir); err != nil {
			return "", fmt.Errorf("failed to remove existing skill: %w", err)
		}
	}

	base := "https://raw.githubusercontent.com/" + repo + "/" + branch
	if si.baseURL != "" {
		base = strings.TrimSuffix(si.baseURL, "/") + "/" + repo + "/" + branch
	}

	// Test mode (baseURL set): single-file install (SKILL.md only) for backward-compatible tests.
	if si.baseURL != "" {
		return si.installSingleFile(ctx, base, repo, branch, subpath, skillDir, skillName, force)
	}

	// Production: install entire directory via GitHub Trees API.
	paths, err := fetchTree(ctx, repo, branch, subpath)
	if err != nil {
		return "", err
	}
	skillMD := "SKILL.md"
	if subpath != "" {
		skillMD = subpath + "/SKILL.md"
	}
	hasSKILL := false
	for _, p := range paths {
		if p == skillMD {
			hasSKILL = true
			break
		}
	}
	if !hasSKILL {
		return "", fmt.Errorf("SKILL.md not found at %s (check branch and path)", base+"/"+skillMD)
	}

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create skill directory: %w", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	for _, p := range paths {
		rawURL := base + "/" + p
		req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create request for %s: %w", p, err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to fetch %s: %w", p, err)
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			return "", fmt.Errorf("failed to fetch %s: HTTP %d", p, resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", p, err)
		}
		localPath := p
		if subpath != "" {
			localPath = p[len(subpath)+1:]
		}
		dest := filepath.Join(skillDir, filepath.FromSlash(localPath))
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return "", fmt.Errorf("failed to create directory for %s: %w", p, err)
		}
		if err := os.WriteFile(dest, body, 0644); err != nil {
			return "", fmt.Errorf("failed to write %s: %w", p, err)
		}
	}

	return skillName, nil
}

// installSingleFile installs only SKILL.md (used when baseURL is set, e.g. tests).
func (si *SkillInstaller) installSingleFile(ctx context.Context, base, repo, branch, subpath, skillDir, skillName string, force bool) (string, error) {
	var rawURL string
	if subpath != "" {
		rawURL = base + "/" + subpath + "/SKILL.md"
	} else {
		rawURL = base + "/SKILL.md"
	}
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch skill: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			return "", fmt.Errorf("SKILL.md not found at %s (check branch and path)", rawURL)
		}
		return "", fmt.Errorf("failed to fetch skill: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create skill directory: %w", err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, body, 0644); err != nil {
		return "", fmt.Errorf("failed to write skill file: %w", err)
	}
	return skillName, nil
}

// InstallFromGitHub installs a skill from GitHub using a single spec string.
// Spec can be "owner/repo" or "owner/repo@branch". Branch defaults to "main".
// For monorepo subpath, use InstallFromGitHubEx.
func (si *SkillInstaller) InstallFromGitHub(ctx context.Context, spec string) error {
	repo, branch, err := ParseInstallSpec(spec)
	if err != nil {
		return err
	}
	_, err = si.InstallFromGitHubEx(ctx, repo, branch, "", false)
	return err
}

const maxArchiveDownloadBytes = 50 * 1024 * 1024 // 50MB

// archiveExt returns the archive extension from a path or URL (e.g. ".zip", ".tar.gz", ".tgz"), or "".
func archiveExt(pathOrURL string) string {
	base := pathOrURL
	if u, err := url.Parse(pathOrURL); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		base = filepath.Base(u.Path)
	} else {
		base = filepath.Base(pathOrURL)
	}
	base = strings.ToLower(base)
	if strings.HasSuffix(base, ".tar.gz") {
		return ".tar.gz"
	}
	if strings.HasSuffix(base, ".tgz") {
		return ".tgz"
	}
	if strings.HasSuffix(base, ".zip") {
		return ".zip"
	}
	return ""
}

// skillNameFromArchivePath derives skill name from path or URL by stripping archive extension.
func skillNameFromArchivePath(pathOrURL string) string {
	base := pathOrURL
	if u, err := url.Parse(pathOrURL); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		base = filepath.Base(u.Path)
	} else {
		base = filepath.Base(pathOrURL)
	}
	base = strings.TrimSuffix(strings.ToLower(base), ".tar.gz")
	base = strings.TrimSuffix(base, ".tgz")
	base = strings.TrimSuffix(base, ".zip")
	return base
}

// InstallFromArchive installs a skill from a .zip, .tar.gz, or .tgz file (local path or http(s) URL).
// If skillName is empty, it is derived from the archive path/URL. If force is true, existing skill dir is removed first.
func (si *SkillInstaller) InstallFromArchive(ctx context.Context, archivePathOrURL, skillName string, force bool) (string, error) {
	archivePathOrURL = strings.TrimSpace(archivePathOrURL)
	if archivePathOrURL == "" {
		return "", fmt.Errorf("empty archive path or URL")
	}

	ext := archiveExt(archivePathOrURL)
	if ext == "" {
		return "", fmt.Errorf("unsupported archive format (need .zip, .tar.gz, or .tgz)")
	}

	var localPath string
	isURL := strings.HasPrefix(strings.ToLower(archivePathOrURL), "http://") || strings.HasPrefix(strings.ToLower(archivePathOrURL), "https://")
	if isURL {
		req, err := http.NewRequestWithContext(ctx, "GET", archivePathOrURL, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}
		client := &http.Client{Timeout: 60 * time.Second}
		tmpPath, err := utils.DownloadToFile(ctx, client, req, maxArchiveDownloadBytes)
		if err != nil {
			return "", fmt.Errorf("download failed: %w", err)
		}
		defer os.Remove(tmpPath)
		localPath = tmpPath
	} else {
		if _, err := os.Stat(archivePathOrURL); err != nil {
			return "", fmt.Errorf("archive file not found: %w", err)
		}
		localPath = archivePathOrURL
	}

	if skillName == "" {
		skillName = skillNameFromArchivePath(archivePathOrURL)
	}
	if skillName == "" {
		return "", fmt.Errorf("could not derive skill name from path or URL")
	}

	skillDir := filepath.Join(si.workspace, "skills", skillName)
	if _, err := os.Stat(skillDir); err == nil {
		if !force {
			return "", fmt.Errorf("skill '%s' already exists (use reinstall to overwrite)", skillName)
		}
		if err := os.RemoveAll(skillDir); err != nil {
			return "", fmt.Errorf("failed to remove existing skill: %w", err)
		}
	}

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create skill directory: %w", err)
	}

	switch ext {
	case ".zip":
		if err := utils.ExtractZipFile(localPath, skillDir); err != nil {
			os.RemoveAll(skillDir)
			return "", err
		}
	case ".tar.gz", ".tgz":
		if err := utils.ExtractTarGzFile(localPath, skillDir); err != nil {
			os.RemoveAll(skillDir)
			return "", err
		}
	default:
		os.RemoveAll(skillDir)
		return "", fmt.Errorf("unsupported archive format: %s", ext)
	}

	// Normalize: if skillDir contains only one directory and SKILL.md is not at root, move contents up.
	if err := normalizeSingleRootDir(skillDir); err != nil {
		os.RemoveAll(skillDir)
		return "", err
	}

	skillMD := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillMD); err != nil {
		os.RemoveAll(skillDir)
		return "", fmt.Errorf("SKILL.md not found in archive (invalid skill package)")
	}

	return skillName, nil
}

// normalizeSingleRootDir moves the contents of a single top-level directory up into skillDir and removes that directory.
func normalizeSingleRootDir(skillDir string) error {
	entries, err := os.ReadDir(skillDir)
	if err != nil {
		return err
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		return nil
	}
	skillMD := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillMD); err == nil {
		return nil
	}
	inner := filepath.Join(skillDir, entries[0].Name())
	innerEntries, err := os.ReadDir(inner)
	if err != nil {
		return err
	}
	for _, e := range innerEntries {
		src := filepath.Join(inner, e.Name())
		dst := filepath.Join(skillDir, e.Name())
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("move %q to skill root: %w", e.Name(), err)
		}
	}
	return os.Remove(inner)
}

func (si *SkillInstaller) Uninstall(skillName string) error {
	skillDir := filepath.Join(si.workspace, "skills", skillName)

	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill '%s' not found", skillName)
	}

	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("failed to remove skill: %w", err)
	}

	return nil
}

func (si *SkillInstaller) ListAvailableSkills(ctx context.Context) ([]AvailableSkill, error) {
	url := "https://raw.githubusercontent.com/sipeed/picoclaw-skills/main/skills.json"

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skills list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch skills list: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var skills []AvailableSkill
	if err := json.Unmarshal(body, &skills); err != nil {
		return nil, fmt.Errorf("failed to parse skills list: %w", err)
	}

	return skills, nil
}
