package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/utils"
)

// IndexRegistry discovers skills from any index.json URL.
type IndexRegistry struct {
	name                string
	indexURL            string
	extraHeader         string
	authorizationHeader string
	agentHeader         string
	allowedPrefixes     []string
	urlMappings         map[string]string
	symlinkLocal        bool
	httpClient          *http.Client
}

// NewIndexRegistry creates a new Index registry.
func NewIndexRegistry(name string, config IndexRegistryConfig) *IndexRegistry {
	return &IndexRegistry{
		name:                name,
		indexURL:            config.IndexURL,
		extraHeader:         config.ExtraHeader,
		authorizationHeader: config.AuthorizationHeader,
		agentHeader:         config.AgentHeader,
		allowedPrefixes:     config.AllowedPrefixes,
		urlMappings:         config.URLMappings,
		symlinkLocal:        config.SymlinkLocal,
		httpClient:          http.DefaultClient,
	}
}

// Name returns the registry name.
func (r *IndexRegistry) Name() string {
	return r.name
}

// isURLAllowed checks if the URL matches any of the allowed prefixes.
// If no prefixes are configured, all URLs are allowed.
func (r *IndexRegistry) isURLAllowed(url string) bool {
	if len(r.allowedPrefixes) == 0 {
		return true
	}
	for _, prefix := range r.allowedPrefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}
	return false
}

// mapURL maps a URL to another URL if a mapping exists.
// This allows redirecting to local files (file://) or alternative sources (forks).
func (r *IndexRegistry) mapURL(url string) string {
	if len(r.urlMappings) == 0 {
		return url
	}
	for prefix, replacement := range r.urlMappings {
		if strings.HasPrefix(url, prefix) {
			return strings.Replace(url, prefix, replacement, 1)
		}
	}
	return url
}

// Search searches for skills in the Index registry.
func (r *IndexRegistry) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	index, err := r.getSkillIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get skill index: %w", err)
	}

	var results []SearchResult
	queryLower := strings.ToLower(query)
	for _, skill := range index.Skills {
		if query == "" || strings.Contains(strings.ToLower(skill.Slug), queryLower) {
			results = append(results, SearchResult{
				Score:        1.0,
				Slug:         skill.Slug,
				DisplayName:  skill.Name,
				Summary:      skill.Description,
				RegistryName: r.name,
			})
			if len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

// GetSkillMeta retrieves metadata for a specific skill.
func (r *IndexRegistry) GetSkillMeta(ctx context.Context, slug string) (*SkillMeta, error) {
	index, err := r.getSkillIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get skill index: %w", err)
	}

	for _, skill := range index.Skills {
		if skill.Slug == slug {
			return &SkillMeta{
				Slug:         skill.Slug,
				DisplayName:  skill.Name,
				Summary:      skill.Description,
				RegistryName: r.name,
			}, nil
		}
	}

	return nil, fmt.Errorf("skill not found: %s", slug)
}

// DownloadAndInstall downloads and installs a skill from GitHub.
func (r *IndexRegistry) DownloadAndInstall(
	ctx context.Context, slug, version, targetDir string,
) (*InstallResult, error) {
	index, err := r.getSkillIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get skill index: %w", err)
	}

	var skillDef SkillDefinition
	var found bool
	for _, skill := range index.Skills {
		if skill.Slug == slug {
			skillDef = skill
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("skill not found: %s", slug)
	}

	// Get download URL
	if skillDef.DownloadURL == "" {
		return nil, fmt.Errorf("no download URL for skill: %s", slug)
	}

	// Apply URL mapping for development (local files or forks)
	mappedURL := r.mapURL(skillDef.DownloadURL)

	// If mapped URL is a local file:// directory and symlink_local enabled,
	// symlink the entire directory instead of downloading individual files
	if r.symlinkLocal && strings.HasPrefix(mappedURL, "file://") {
		localPath := strings.TrimPrefix(mappedURL, "file://")
		info, err := os.Stat(localPath)
		if err == nil && info.IsDir() {
			if err := r.copyLocalPath(localPath, targetDir); err != nil {
				return nil, err
			}
			return &InstallResult{
				Summary: fmt.Sprintf("Symlinked skill from %s", mappedURL),
			}, nil
		}
	}

	// If files list provided, fetch each file from download_url + file path
	// Otherwise, fetch directly from download_url
	if len(skillDef.Files) > 0 {
		for _, filePath := range skillDef.Files {
			url := strings.TrimRight(mappedURL, "/") + "/" + filePath
			if err := r.downloadFromURL(ctx, url, targetDir); err != nil {
				return nil, fmt.Errorf("failed to download %s: %w", filePath, err)
			}
		}
	} else {
		if err := r.downloadFromURL(ctx, mappedURL, targetDir); err != nil {
			return nil, fmt.Errorf("failed to download skill: %w", err)
		}
	}

	return &InstallResult{
		Summary: fmt.Sprintf("Installed skill from %s", mappedURL),
	}, nil
}

// downloadFromURL fetches from download_url and handles both file content and directory listings.
func (r *IndexRegistry) downloadFromURL(ctx context.Context, url, targetDir string) error {
	// Apply URL mapping (for local files or alternative sources)
	url = r.mapURL(url)

	// Handle file:// URLs for local development
	if strings.HasPrefix(url, "file://") {
		localPath := strings.TrimPrefix(url, "file://")
		return r.copyLocalPath(localPath, targetDir)
	}

	// Validate URL against allowed prefixes
	if !r.isURLAllowed(url) {
		return fmt.Errorf("URL not allowed by registry restrictions: %s", url)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: %s", resp.Status)
	}

	contentType := resp.Header.Get("Content-Type")
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Check if it's a ZIP archive
	if strings.HasSuffix(url, ".zip") || strings.Contains(contentType, "zip") {
		tmpPath, tmpErr := os.CreateTemp("", "skill-*.zip")
		if tmpErr != nil {
			return fmt.Errorf("failed to create temp file: %w", tmpErr)
		}
		defer os.Remove(tmpPath.Name())

		if _, err := tmpPath.Write(data); err != nil {
			tmpPath.Close()
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		tmpPath.Close()

		// Extract to temp dir first, then strip root component
		tmpDir, err := os.MkdirTemp("", "skill-extract-*")
		if err != nil {
			return fmt.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		if err := utils.ExtractZipFile(tmpPath.Name(), tmpDir); err != nil {
			return err
		}

		// Strip first path component (common in archives)
		return r.stripRootAndMove(tmpDir, targetDir)
	}

	// Check if it's JSON (directory listing) or raw content
	if strings.Contains(contentType, "application/json") || strings.HasPrefix(strings.TrimSpace(string(data)), "[") {
		// It's a JSON listing - parse and fetch each item
		return r.parseAndDownloadListing(ctx, data, targetDir)
	}

	// It's raw content - save directly to targetDir
	return r.saveFile(data, targetDir, filepath.Base(url))
}

// parseAndDownloadListing parses JSON listing and downloads each file.
func (r *IndexRegistry) parseAndDownloadListing(ctx context.Context, data []byte, targetDir string) error {
	// Try parsing as array first
	var items []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"download_url"`
		Type        string `json:"type"`
	}

	if err := json.Unmarshal(data, &items); err != nil {
		// Try single object
		var item struct {
			Name        string `json:"name"`
			DownloadURL string `json:"download_url"`
			Type        string `json:"type"`
		}
		if err := json.Unmarshal(data, &item); err != nil {
			return fmt.Errorf("failed to parse listing: %w", err)
		}
		if item.Name != "" {
			items = []struct {
				Name        string `json:"name"`
				DownloadURL string `json:"download_url"`
				Type        string `json:"type"`
			}{item}
		}
	}

	for _, item := range items {
		if item.Type == "dir" || item.DownloadURL == "" {
			subDir := filepath.Join(targetDir, item.Name)
			if err := os.MkdirAll(subDir, 0o755); err != nil {
				return err
			}
			if err := r.downloadFromURL(ctx, item.DownloadURL, subDir); err != nil {
				return err
			}
		} else {
			// Download file
			if err := r.downloadFile(ctx, item.DownloadURL, targetDir, item.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

// downloadFile downloads a single file.
func (r *IndexRegistry) downloadFile(ctx context.Context, url, targetDir, filename string) error {
	// Apply URL mapping
	url = r.mapURL(url)

	// Handle file:// URLs for local development
	if strings.HasPrefix(url, "file://") {
		localPath := strings.TrimPrefix(url, "file://")
		return r.copyLocalPath(localPath, targetDir)
	}

	if !r.isURLAllowed(url) {
		return fmt.Errorf("URL not allowed by registry restrictions: %s", url)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return r.saveFile(data, targetDir, filename)
}

// saveFile saves content to a file in targetDir.
func (r *IndexRegistry) saveFile(data []byte, targetDir, filename string) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}

	path := filepath.Join(targetDir, filename)
	return os.WriteFile(path, data, 0o644)
}

// copyLocalPath copies a local file or directory to targetDir.
// If symlinkLocal is true, creates a symlink to the local directory instead of copying.
func (r *IndexRegistry) copyLocalPath(localPath, targetDir string) error {
	// If symlink enabled, create a symlink to the local directory
	if r.symlinkLocal {
		absPath, err := filepath.Abs(localPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		// Remove targetDir if it exists (we're creating a symlink, not a directory)
		os.RemoveAll(targetDir)
		return os.Symlink(absPath, targetDir)
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to stat local path: %w", err)
	}

	if info.IsDir() {
		return filepath.Walk(localPath, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			relPath, err := filepath.Rel(localPath, path)
			if err != nil {
				return err
			}
			if relPath == "." {
				return nil
			}
			destPath := filepath.Join(targetDir, relPath)
			if fi.IsDir() {
				return os.MkdirAll(destPath, 0o755)
			}
			return copyFile(path, destPath)
		})
	}

	// Single file
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}
	return copyFile(localPath, filepath.Join(targetDir, filepath.Base(localPath)))
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// stripRootAndMove moves files from srcDir to targetDir, stripping the first path component.
func (r *IndexRegistry) stripRootAndMove(srcDir, targetDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	// If there's a single directory, use it as the root
	var rootDir string
	for _, entry := range entries {
		if entry.IsDir() {
			rootDir = entry.Name()
			break
		}
	}

	if rootDir == "" {
		// No subdirectory, just move everything up
	} else {
		srcDir = filepath.Join(srcDir, rootDir)
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		destPath := filepath.Join(targetDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		return os.Rename(path, destPath)
	})
}

// getSkillIndex fetches the skill index from the configured URL.
func (r *IndexRegistry) getSkillIndex(ctx context.Context) (*SkillIndex, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", r.indexURL, nil)
	if err != nil {
		return nil, err
	}

	if r.extraHeader != "" {
		parts := strings.SplitN(r.extraHeader, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
	if r.authorizationHeader != "" {
		req.Header.Set("Authorization", r.authorizationHeader)
	}
	if r.agentHeader != "" {
		req.Header.Set("User-Agent", r.agentHeader)
	} else {
		req.Header.Set("User-Agent", "picoclaw")
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch skills index from %s: HTTP %d", r.indexURL, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var index SkillIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse skills index from %s: %w", r.indexURL, err)
	}

	if len(index.Skills) == 0 {
		return nil, fmt.Errorf("no skills found in index from %s (check format/version)", r.indexURL)
	}

	// Validate at least one skill has a download URL
	hasValidSkill := false
	for _, s := range index.Skills {
		if s.DownloadURL != "" {
			hasValidSkill = true
			break
		}
	}
	if !hasValidSkill {
		return nil, fmt.Errorf("index from %s has no skills with download_url (check format)", r.indexURL)
	}

	return &index, nil
}
