package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/utils"
)

const wellKnownPath = "/.well-known/agent-skills/index.json"

type SkillIndex struct {
	Skills []SkillIndexEntry `json:"skills"`
}

type SkillIndexEntry struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ResolvedSource struct {
	Type       string
	URL        string
	SkillIndex *SkillIndex
}

func ResolveSkillSource(ctx context.Context, client *http.Client, input string) (*ResolvedSource, error) {
	input = strings.TrimSpace(input)

	if isLocalPath(input) {
		return &ResolvedSource{Type: "local", URL: input}, nil
	}

	if isJSONURL(input) {
		return &ResolvedSource{Type: "json_url", URL: input}, nil
	}

	if isGitHubShorthand(input) {
		return &ResolvedSource{Type: "github", URL: input}, nil
	}

	if isDomain(input) {
		indexURL := buildWellKnownURL(input)
		skillIndex, err := fetchSkillIndex(ctx, client, indexURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch skill index from %s: %w", indexURL, err)
		}
		return &ResolvedSource{
			Type:       "well_known",
			URL:        indexURL,
			SkillIndex: skillIndex,
		}, nil
	}

	return nil, fmt.Errorf("invalid skill source: %s", input)
}

func isLocalPath(input string) bool {
	return strings.HasPrefix(input, "/") ||
		strings.HasPrefix(input, "./") ||
		strings.HasPrefix(input, "../") ||
		strings.HasPrefix(input, "~")
}

func isJSONURL(input string) bool {
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		return false
	}
	return strings.HasSuffix(strings.ToLower(input), ".json")
}

func isGitHubShorthand(input string) bool {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return false
	}
	if strings.Contains(input, "/") && !strings.Contains(input, "://") {
		parts := strings.Split(input, "/")
		return len(parts) >= 2
	}
	return false
}

func isDomain(input string) bool {
	return strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://")
}

func buildWellKnownURL(domain string) string {
	domain = strings.TrimSuffix(domain, "/")
	return domain + wellKnownPath
}

func fetchSkillIndex(ctx context.Context, client *http.Client, url string) (*SkillIndex, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := utils.DoRequestWithRetry(client, req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("skill index not found (404)")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var index SkillIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %w", err)
	}

	if len(index.Skills) == 0 {
		return nil, fmt.Errorf("no skills found in index")
	}

	return &index, nil
}

func CreateHTTPClient(proxy string, timeout time.Duration) (*http.Client, error) {
	return utils.CreateHTTPClient(proxy, timeout)
}
