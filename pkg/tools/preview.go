package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HostedPreview describes a published local preview.
type HostedPreview struct {
	Slug         string `json:"slug,omitempty"`
	Root         string `json:"root,omitempty"`
	Entry        string `json:"entry,omitempty"`
	LocalURL     string `json:"local_url,omitempty"`
	TailscaleURL string `json:"tailscale_url,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// PreviewPublisher publishes a local directory on a preview server and returns URLs.
type PreviewPublisher func(root, entry, slug string) (*HostedPreview, error)

// HostPreviewTool publishes a local site/app directory and returns preview URLs.
type HostPreviewTool struct {
	workspace string
	restrict  bool
	publish   PreviewPublisher
}

const recentPreviewsStatePath = "state/recent_previews.json"
const defaultHostedPreviewSlug = "site"

func NewHostPreviewTool(workspace string, restrict bool, publish PreviewPublisher) *HostPreviewTool {
	return &HostPreviewTool{workspace: workspace, restrict: restrict, publish: publish}
}

func (t *HostPreviewTool) Name() string {
	return "host_preview"
}

func (t *HostPreviewTool) Description() string {
	return "Host a local site/app directory on the built-in preview server and return one stable Tailscale/local preview URL for review."
}

func (t *HostPreviewTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to a built site/app directory, or a single entry file such as index.html.",
			},
			"entry": map[string]any{
				"type":        "string",
				"description": "Optional entry file relative to the hosted directory. Defaults to index.html when present, or the file basename when path points to a file.",
			},
			"slug": map[string]any{
				"type":        "string",
				"description": "Optional stable preview slug. Reuse the same slug to keep updating one preview URL across edits.",
			},
		},
		"required": []string{"path"},
	}
}

func (t *HostPreviewTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	_ = ctx
	if t.publish == nil {
		return ErrorResult("preview publishing is not configured")
	}

	rawPath, _ := args["path"].(string)
	if strings.TrimSpace(rawPath) == "" {
		return ErrorResult("path is required")
	}

	resolved, err := validatePath(rawPath, t.workspace, t.restrict)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid path: %v", err))
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return ErrorResult(fmt.Sprintf("path not found: %v", err))
	}

	entry, _ := args["entry"].(string)
	entry = strings.TrimSpace(entry)
	slug, _ := args["slug"].(string)
	slug = strings.TrimSpace(slug)
	if slug == "" {
		slug = defaultHostedPreviewSlug
	}

	root := resolved
	if info.IsDir() {
		if entry != "" {
			candidate := filepath.Join(root, filepath.FromSlash(strings.TrimLeft(entry, "/")))
			entryInfo, statErr := os.Stat(candidate)
			if statErr != nil {
				return ErrorResult(fmt.Sprintf("entry not found: %v", statErr))
			}
			if entryInfo.IsDir() {
				entry = filepath.ToSlash(filepath.Join(entry, "index.html"))
			}
		} else if _, statErr := os.Stat(filepath.Join(root, "index.html")); statErr == nil {
			entry = "index.html"
		}
	} else {
		ext := strings.ToLower(filepath.Ext(resolved))
		if ext != ".html" && ext != ".htm" {
			return ErrorResult("path must be a site directory or an HTML entry file")
		}
		root = filepath.Dir(resolved)
		if entry == "" {
			entry = filepath.Base(resolved)
		}
	}

	preview, err := t.publish(root, entry, slug)
	if err != nil {
		return ErrorResult(err.Error())
	}
	if preview != nil {
		preview.UpdatedAt = time.Now().Format(time.RFC3339)
		if strings.TrimSpace(preview.Slug) == "" {
			preview.Slug = slug
		}
		if saveErr := SaveRecentPreview(t.workspace, preview); saveErr != nil {
			return ErrorResult(fmt.Sprintf("preview published but failed to save state: %v", saveErr))
		}
	}

	lines := []string{fmt.Sprintf("Preview hosted from %s", preview.Root)}
	if preview.Entry != "" {
		lines = append(lines, "Entry: "+preview.Entry)
	}
	if preview.TailscaleURL != "" {
		lines = append(lines, "Tailscale URL: "+preview.TailscaleURL)
	}
	if preview.LocalURL != "" {
		lines = append(lines, "Local URL: "+preview.LocalURL)
	}
	if preview.TailscaleURL == "" && preview.LocalURL == "" {
		return ErrorResult("preview published but no URL was generated")
	}

	content := strings.Join(lines, "\n")
	return &ToolResult{ForLLM: content, ForUser: content}
}

func recentPreviewsFile(workspace string) string {
	return filepath.Join(workspace, recentPreviewsStatePath)
}

func LoadRecentPreviews(workspace string) ([]HostedPreview, error) {
	path := recentPreviewsFile(workspace)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var previews []HostedPreview
	if err := json.Unmarshal(data, &previews); err != nil {
		return nil, err
	}
	return previews, nil
}

func SaveRecentPreview(workspace string, preview *HostedPreview) error {
	if preview == nil {
		return nil
	}
	path := recentPreviewsFile(workspace)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	updated := []HostedPreview{*preview}
	data, err := json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
