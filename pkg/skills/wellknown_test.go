package skills

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResolveSkillSource_LocalPath(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	ctx := context.Background()

	tests := []struct {
		input    string
		wantType string
	}{
		{"/path/to/skill", "local"},
		{"./relative/path", "local"},
		{"../parent/path", "local"},
		{"~/home/path", "local"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			resolved, err := ResolveSkillSource(ctx, client, tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resolved.Type != tt.wantType {
				t.Errorf("expected type %s, got %s", tt.wantType, resolved.Type)
			}
		})
	}
}

func TestResolveSkillSource_JSONURL(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	ctx := context.Background()

	tests := []struct {
		input    string
		wantType string
	}{
		{"https://example.com/skill.json", "json_url"},
		{"http://example.com/path/to/skill.JSON", "json_url"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			resolved, err := ResolveSkillSource(ctx, client, tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resolved.Type != tt.wantType {
				t.Errorf("expected type %s, got %s", tt.wantType, resolved.Type)
			}
		})
	}
}

func TestResolveSkillSource_GitHubShorthand(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	ctx := context.Background()

	tests := []struct {
		input    string
		wantType string
	}{
		{"owner/repo", "github"},
		{"owner/repo/path/to/skill", "github"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			resolved, err := ResolveSkillSource(ctx, client, tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resolved.Type != tt.wantType {
				t.Errorf("expected type %s, got %s", tt.wantType, resolved.Type)
			}
		})
	}
}

func TestResolveSkillSource_WellKnown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent-skills/index.json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"skills": [
					{"name": "weather", "url": "https://example.com/weather.json"},
					{"name": "search", "url": "https://github.com/owner/search-skill"}
				]
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := server.Client()
	ctx := context.Background()

	resolved, err := ResolveSkillSource(ctx, client, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Type != "well_known" {
		t.Errorf("expected type well_known, got %s", resolved.Type)
	}

	if resolved.SkillIndex == nil {
		t.Fatal("expected SkillIndex to be non-nil")
	}

	if len(resolved.SkillIndex.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(resolved.SkillIndex.Skills))
	}

	if resolved.SkillIndex.Skills[0].Name != "weather" {
		t.Errorf("expected first skill name 'weather', got %s", resolved.SkillIndex.Skills[0].Name)
	}
}

func TestResolveSkillSource_WellKnown_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := server.Client()
	ctx := context.Background()

	_, err := ResolveSkillSource(ctx, client, server.URL)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestResolveSkillSource_WellKnown_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := server.Client()
	ctx := context.Background()

	_, err := ResolveSkillSource(ctx, client, server.URL)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestResolveSkillSource_WellKnown_EmptySkills(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"skills": []}`))
	}))
	defer server.Close()

	client := server.Client()
	ctx := context.Background()

	_, err := ResolveSkillSource(ctx, client, server.URL)
	if err == nil {
		t.Fatal("expected error for empty skills")
	}
}

func TestBuildWellKnownURL(t *testing.T) {
	tests := []struct {
		domain   string
		expected string
	}{
		{"https://example.com", "https://example.com/.well-known/agent-skills/index.json"},
		{"https://example.com/", "https://example.com/.well-known/agent-skills/index.json"},
		{"http://test.org", "http://test.org/.well-known/agent-skills/index.json"},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := buildWellKnownURL(tt.domain)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestIsLocalPath(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"/absolute/path", true},
		{"./relative/path", true},
		{"../parent/path", true},
		{"~/home/path", true},
		{"https://example.com", false},
		{"owner/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isLocalPath(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsJSONURL(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"https://example.com/skill.json", true},
		{"http://example.com/skill.JSON", true},
		{"https://example.com/skill", false},
		{"owner/repo.json", false},
		{"/local/path.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isJSONURL(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsGitHubShorthand(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"owner/repo", true},
		{"owner/repo/path", true},
		{"https://github.com/owner/repo", false},
		{"single", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isGitHubShorthand(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
