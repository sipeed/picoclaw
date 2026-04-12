package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/utils"
)

// GitHubContent represents a file or directory in GitHub API response
type GitHubContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"` // "file" or "dir"
	DownloadURL string `json:"download_url"`
	URL         string `json:"url"` // API URL for subdirectories
}

// GitHubRef represents a parsed GitHub reference
type GitHubRef struct {
	Owner    string // Repository owner
	RepoName string // Repository name
	Ref      string // Git reference (branch, tag, or commit)
	SubPath  string // Path within the repository
}

type SkillInstaller struct {
	workspace   string
	client      *http.Client
	githubToken string
	proxy       string
}

// NewSkillInstaller creates a new skill installer.
// proxy is an optional HTTP/HTTPS/SOCKS5 proxy URL for downloading skills.
func NewSkillInstaller(workspace, githubToken, proxy string) (*SkillInstaller, error) {
	client, err := utils.CreateHTTPClient(proxy, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &SkillInstaller{
		workspace:   workspace,
		client:      client,
		githubToken: githubToken,
		proxy:       proxy,
	}, nil
}

// parseGitHubRef parses a GitHub reference.
// Supports: "owner/repo", "owner/repo/path", or full URL like "https://github.com/owner/repo/tree/ref/path"
func parseGitHubRef(repo string) (GitHubRef, error) {
	repo = strings.TrimSpace(repo)

	// Handle full URL
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") {
		u, err := url.Parse(repo)
		if err != nil {
			return GitHubRef{}, fmt.Errorf("invalid URL: %w", err)
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 {
			return GitHubRef{}, fmt.Errorf("invalid GitHub URL")
		}
		ref := GitHubRef{
			Owner:    parts[0],
			RepoName: parts[1],
			Ref:      "main",
		}
		// Look for /tree/ or /blob/ in the path
		for i := 2; i < len(parts); i++ {
			if parts[i] == "tree" || parts[i] == "blob" {
				if i+1 < len(parts) {
					ref.Ref = parts[i+1]
					ref.SubPath = strings.Join(parts[i+2:], "/")
				}
				break
			}
		}
		return ref, nil
	}

	// Handle shorthand format
	parts := strings.Split(strings.Trim(repo, "/"), "/")
	if len(parts) < 2 {
		return GitHubRef{}, fmt.Errorf("invalid format %q: expected 'owner/repo'", repo)
	}
	ref := GitHubRef{
		Owner:    parts[0],
		RepoName: parts[1],
		Ref:      "main",
	}
	if len(parts) > 2 {
		ref.SubPath = strings.Join(parts[2:], "/")
	}
	return ref, nil
}

func (si *SkillInstaller) InstallFromGitHub(ctx context.Context, repo string) error {
	ref, err := parseGitHubRef(repo)
	if err != nil {
		return err
	}

	skillName := ref.RepoName
	if ref.SubPath != "" {
		skillName = filepath.Base(ref.SubPath)
	}
	skillDirectory := filepath.Join(si.workspace, "skills", skillName)

	if _, err := os.Stat(skillDirectory); err == nil {
		return fmt.Errorf("skill '%s' already exists", skillName)
	}

	// Build GitHub API URL
	apiPath := path.Join(ref.Owner, ref.RepoName, "contents")
	if ref.SubPath != "" {
		apiPath = path.Join(apiPath, ref.SubPath)
	}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s?ref=%s", apiPath, ref.Ref)

	if err := si.getGithubDirAllFiles(ctx, apiURL, skillDirectory, true); err != nil {
		// Fallback to raw download
		return si.downloadRaw(ctx, ref.Owner, ref.RepoName, ref.Ref, ref.SubPath, skillDirectory)
	}

	if _, err := os.Stat(filepath.Join(skillDirectory, "SKILL.md")); err != nil {
		return fmt.Errorf("SKILL.md not found in repository")
	}
	return nil
}

// downloadDir recursively downloads a directory from GitHub API
// isRoot: true if this is the skill root directory (only download SKILL.md at root)
func (si *SkillInstaller) getGithubDirAllFiles(ctx context.Context, apiURL, localDir string, isRoot bool) error {
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return err
	}
	if si.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+si.githubToken)
	}

	resp, err := utils.DoRequestWithRetry(si.client, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var items []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return err
	}

	for _, item := range items {
		localPath := filepath.Join(localDir, item.Name)

		switch item.Type {
		case "file":
			if !shouldDownload(item.Name, isRoot) {
				continue
			}
			if err := si.downloadFile(ctx, item.DownloadURL, localPath); err != nil {
				return fmt.Errorf("download %s: %w", item.Name, err)
			}
		case "dir":
			if !isSkillDirectory(item.Name) {
				continue
			}
			if err := si.getGithubDirAllFiles(ctx, item.URL, localPath, false); err != nil {
				return err
			}
		}
	}
	return nil
}

// downloadRaw is a fallback that downloads just SKILL.md from raw.githubusercontent.com
func (si *SkillInstaller) downloadRaw(ctx context.Context, owner, repo, ref, subPath, localDir string) error {
	urlPath := path.Join(owner, repo, ref)
	if subPath != "" {
		urlPath = path.Join(urlPath, subPath)
	}
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/SKILL.md", urlPath)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Use chunked download to temporary file.
	tmpPath, err := utils.DownloadToFile(ctx, si.client, req, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch skill: %w", err)
	}
	defer os.Remove(tmpPath)

	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	localPath := filepath.Join(localDir, "SKILL.md")

	// Atomic move from temp to final location.
	if err := os.Rename(tmpPath, localPath); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	return os.Chmod(localPath, 0o600)
}

func (si *SkillInstaller) downloadFile(ctx context.Context, url, localPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	// Use chunked download to temporary file, then move atomically to target.
	tmpPath, err := utils.DownloadToFile(ctx, si.client, req, 0)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}

	// Atomic move from temp to final location.
	if err := os.Rename(tmpPath, localPath); err != nil {
		return fmt.Errorf("failed to move downloaded file: %w", err)
	}

	return os.Chmod(localPath, 0o600)
}

// shouldDownload determines if a file should be downloaded
// root: true if we're at the skill root directory
func shouldDownload(name string, root bool) bool {
	if root {
		return name == "SKILL.md"
	}
	return true
}

// isSkillDir checks if a directory is a standard skill resource directory
func isSkillDirectory(name string) bool {
	switch name {
	case "scripts", "references", "assets", "templates", "docs":
		return true
	}
	return false
}

func (si *SkillInstaller) Uninstall(skillName string) error {
	parts := strings.Split(skillName, "/")
	var finalSkillName string
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			finalSkillName = parts[i]
			break
		}
	}
	if finalSkillName == "" {
		finalSkillName = skillName
	}

	skillDir := filepath.Join(si.workspace, "skills", finalSkillName)

	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill '%s' not found (processed as '%s')", skillName, finalSkillName)
	}

	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("failed to remove skill '%s': %w", finalSkillName, err)
	}

	return nil
}
