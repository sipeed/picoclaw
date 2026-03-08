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

	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type GitHubContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}

type SkillInstaller struct {
	workspace string
}

func NewSkillInstaller(workspace string) *SkillInstaller {
	return &SkillInstaller{workspace: workspace}
}

// parseRepoRef 解析 owner/repo/path 格式，默认 main 分支
func parseRepoRef(repo string) (owner, repoName, ref, path string, err error) {
	parts := strings.Split(strings.Trim(repo, "/"), "/")
	if len(parts) < 3 {
		return "", "", "", "", fmt.Errorf("invalid format: owner/repo/path")
	}
	return parts[0], parts[1], "main", strings.Join(parts[2:], "/"), nil
}

func (si *SkillInstaller) downloadFile(ctx context.Context, fileURL, savePath string) error {
	dir := filepath.Dir(savePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", fileURL, nil)

	resp, err := utils.DoRequestWithRetry(client, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	return fileutil.WriteFileAtomic(savePath, body, 0644)
}

func (si *SkillInstaller) downloadDir(ctx context.Context, owner, repo, ref, dirPath, localRoot string) error {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(dirPath), url.PathEscape(ref))
	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", apiURL, nil)

	resp, err := utils.DoRequestWithRetry(client, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API HTTP %d", resp.StatusCode)
	}

	var contents []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return err
	}

	for _, item := range contents {
		relPath := strings.TrimPrefix(strings.TrimPrefix(item.Path, dirPath), "/")
		localPath := filepath.Join(localRoot, relPath)

		switch item.Type {
		case "file":
			if item.DownloadURL != "" {
				if err := si.downloadFile(ctx, item.DownloadURL, localPath); err != nil {
					return fmt.Errorf("download %s: %w", item.Path, err)
				}
			}
		case "dir":
			if err := si.downloadDir(ctx, owner, repo, ref, item.Path, localRoot); err != nil {
				return fmt.Errorf("download dir %s: %w", item.Path, err)
			}
		}
	}
	return nil
}

func (si *SkillInstaller) InstallFromGitHub(ctx context.Context, repo string) error {
	owner, repoName, ref, path, err := parseRepoRef(repo)
	if err != nil {
		return err
	}

	skillName := filepath.Base(path)

	skillDir := filepath.Join(si.workspace, "skills", skillName)
	if _, err := os.Stat(skillDir); err == nil {
		return fmt.Errorf("skill '%s' already exists", skillName)
	}

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}

	if err := si.downloadDir(ctx, owner, repoName, ref, path, skillDir); err != nil {
		os.RemoveAll(skillDir)
		return fmt.Errorf("install skill: %w", err)
	}
	return nil
}

func (si *SkillInstaller) Uninstall(skillName string) error {
	skillDir := filepath.Join(si.workspace, "skills", skillName)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill '%s' not found", skillName)
	}
	return os.RemoveAll(skillDir)
}

// InstallFromRegistry installs a skill from a registry.
func (si *SkillInstaller) InstallFromRegistry(ctx context.Context, registry SkillRegistry, slug string) error {
	targetDir := filepath.Join(si.workspace, "skills", slug)

	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("skill '%s' already exists", slug)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	result, err := registry.DownloadAndInstall(ctx, slug, "", targetDir)
	if err != nil {
		os.RemoveAll(targetDir)
		return err
	}

	if result.IsMalwareBlocked {
		os.RemoveAll(targetDir)
		return fmt.Errorf("skill '%s' is flagged as malicious", slug)
	}

	return nil
}

