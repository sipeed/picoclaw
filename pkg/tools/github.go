package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v60/github"
	"github.com/sipeed/picoclaw/pkg/config"
)

type GitHubTool struct {
	config config.GitHubConfig
	client *github.Client
}

func NewGitHubTool(cfg config.GitHubConfig) *GitHubTool {
	t := &GitHubTool{
		config: cfg,
	}
	if cfg.Enabled {
		if cfg.Token != "" {
			t.client = github.NewClient(nil).WithAuthToken(cfg.Token)
		} else {
			t.client = github.NewClient(nil)
		}
	}
	return t
}

func (t *GitHubTool) Name() string {
	return "github"
}

func (t *GitHubTool) Description() string {
	return "Interact with GitHub to list issues, pull requests, or read files from repositories."
}

func (t *GitHubTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "Action to perform: list_issues, get_pr, read_file, list_repos",
				"enum":        []string{"list_issues", "get_pr", "read_file", "list_repos"},
			},
			"owner": map[string]interface{}{
				"type":        "string",
				"description": "Repository owner (user or organization)",
			},
			"repo": map[string]interface{}{
				"type":        "string",
				"description": "Repository name",
			},
			"number": map[string]interface{}{
				"type":        "integer",
				"description": "Issue or PR number (required for get_pr)",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File path in repository (required for read_file)",
			},
			"count": map[string]interface{}{
				"type":        "integer",
				"description": "Number of items to list (default: 5, max: 20)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *GitHubTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if !t.config.Enabled {
		return ErrorResult("GitHub tool is not enabled in configuration.")
	}
	if t.client == nil {
		return ErrorResult("GitHub client not initialized.")
	}

	action, _ := args["action"].(string)
	owner, _ := args["owner"].(string)
	repo, _ := args["repo"].(string)

	// Defaults
	count := 5
	if c, ok := args["count"].(float64); ok {
		count = int(c)
	}
	if count > 20 {
		count = 20
	}

	switch action {
	case "list_issues":
		if owner == "" || repo == "" {
			return ErrorResult("Owner and repo are required for list_issues.")
		}
		return t.listIssues(ctx, owner, repo, count)
	case "get_pr":
		if owner == "" || repo == "" {
			return ErrorResult("Owner and repo are required for get_pr.")
		}
		number, ok := args["number"].(float64)
		if !ok {
			return ErrorResult("Number is required for get_pr.")
		}
		return t.getPR(ctx, owner, repo, int(number))
	case "read_file":
		if owner == "" || repo == "" {
			return ErrorResult("Owner and repo are required for read_file.")
		}
		path, _ := args["path"].(string)
		if path == "" {
			return ErrorResult("Path is required for read_file.")
		}
		return t.readFile(ctx, owner, repo, path)
	case "list_repos":
		// If owner is provided, list user's repos, else authenticated user's repos
		return t.listRepos(ctx, owner, count)
	default:
		return ErrorResult(fmt.Sprintf("Unknown action: %s", action))
	}
}

func (t *GitHubTool) listIssues(ctx context.Context, owner, repo string, count int) *ToolResult {
	opts := &github.IssueListByRepoOptions{
		State:       "open",
		ListOptions: github.ListOptions{PerPage: count},
	}
	issues, _, err := t.client.Issues.ListByRepo(ctx, owner, repo, opts)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to list issues: %v", err))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Open issues in %s/%s:\n", owner, repo))
	for _, issue := range issues {
		sb.WriteString(fmt.Sprintf("- #%d: %s (by %s)\n", issue.GetNumber(), issue.GetTitle(), issue.User.GetLogin()))
	}

	return &ToolResult{
		ForLLM:  sb.String(),
		ForUser: sb.String(),
	}
}

func (t *GitHubTool) getPR(ctx context.Context, owner, repo string, number int) *ToolResult {
	pr, _, err := t.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to get PR #%d: %v", number, err))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("PR #%d: %s\n", pr.GetNumber(), pr.GetTitle()))
	sb.WriteString(fmt.Sprintf("State: %s\n", pr.GetState()))
	sb.WriteString(fmt.Sprintf("User: %s\n", pr.User.GetLogin()))
	if pr.Body != nil {
		sb.WriteString(fmt.Sprintf("\nBody:\n%s\n", *pr.Body))
	}

	return &ToolResult{
		ForLLM:  sb.String(),
		ForUser: sb.String(),
	}
}

func (t *GitHubTool) readFile(ctx context.Context, owner, repo, path string) *ToolResult {
	content, _, _, err := t.client.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to read file: %v", err))
	}

	decoded, err := content.GetContent()
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to decode file content: %v", err))
	}

	return &ToolResult{
		ForLLM:  decoded,
		ForUser: fmt.Sprintf("Read file %s from %s/%s", path, owner, repo),
	}
}

func (t *GitHubTool) listRepos(ctx context.Context, user string, count int) *ToolResult {
	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: count},
		Sort:        "updated",
	}
	var repos []*github.Repository
	var err error

	if user != "" {
		repos, _, err = t.client.Repositories.List(ctx, user, opts)
	} else {
		// Authenticated user
		repos, _, err = t.client.Repositories.List(ctx, "", opts)
	}

	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to list repos: %v", err))
	}

	var sb strings.Builder
	if user != "" {
		sb.WriteString(fmt.Sprintf("Repositories for %s:\n", user))
	} else {
		sb.WriteString("Your repositories:\n")
	}

	for _, repo := range repos {
		sb.WriteString(fmt.Sprintf("- %s: %s (⭐ %d)\n", repo.GetName(), repo.GetDescription(), repo.GetStargazersCount()))
	}

	return &ToolResult{
		ForLLM:  sb.String(),
		ForUser: sb.String(),
	}
}
