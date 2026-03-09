package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTestSkill creates a skill directory with a SKILL.md containing frontmatter.
func createTestSkill(t *testing.T, skillsDir, dirName, name, description, body string) {
	t.Helper()
	dir := filepath.Join(skillsDir, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n\n" + body
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMatchSkillsInMessage_Basic(t *testing.T) {
	tmpDir := setupWorkspace(t, nil)
	defer os.RemoveAll(tmpDir)

	createTestSkill(
		t,
		filepath.Join(tmpDir, "skills"),
		"news-summary",
		"news-summary",
		"Summarize news",
		"Read RSS feeds",
	)
	createTestSkill(t, filepath.Join(tmpDir, "skills"), "weather", "weather", "Get weather info", "Check weather API")

	cb := NewContextBuilder(tmpDir)

	tests := []struct {
		name     string
		message  string
		expected []string
	}{
		{
			name:     "exact skill name",
			message:  "use the news-summary skill",
			expected: []string{"news-summary"},
		},
		{
			name:     "case insensitive",
			message:  "Use the News-Summary skill please",
			expected: []string{"news-summary"},
		},
		{
			name:     "multiple skills",
			message:  "get the weather and run news-summary",
			expected: []string{"news-summary", "weather"},
		},
		{
			name:     "no match",
			message:  "hello how are you",
			expected: nil,
		},
		{
			name:     "empty message",
			message:  "",
			expected: nil,
		},
		{
			name:     "skill name at start",
			message:  "news-summary please run it",
			expected: []string{"news-summary"},
		},
		{
			name:     "skill name at end",
			message:  "please run news-summary",
			expected: []string{"news-summary"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matched := cb.MatchSkillsInMessage(tc.message)
			if tc.expected == nil {
				if len(matched) != 0 {
					t.Errorf("expected no matches, got %v", matched)
				}
				return
			}
			if len(matched) != len(tc.expected) {
				t.Fatalf("expected %d matches %v, got %d: %v", len(tc.expected), tc.expected, len(matched), matched)
			}
			for _, exp := range tc.expected {
				found := false
				for _, m := range matched {
					if m == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected match %q not found in %v", exp, matched)
				}
			}
		})
	}
}

func TestMatchSkillsInMessage_WholeWordBoundary(t *testing.T) {
	tmpDir := setupWorkspace(t, nil)
	defer os.RemoveAll(tmpDir)

	createTestSkill(t, filepath.Join(tmpDir, "skills"), "git", "git", "Git operations", "Run git commands")

	cb := NewContextBuilder(tmpDir)

	tests := []struct {
		name    string
		message string
		match   bool
	}{
		{"standalone", "use git to commit", true},
		{"with punctuation", "run git, then push", true},
		{"in parentheses", "tools (git) are available", true},
		{"partial word github", "check github for updates", false},
		{"partial word digit", "use digit recognition", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matched := cb.MatchSkillsInMessage(tc.message)
			if tc.match && len(matched) == 0 {
				t.Errorf("expected match for %q but got none", tc.message)
			}
			if !tc.match && len(matched) > 0 {
				t.Errorf("expected no match for %q but got %v", tc.message, matched)
			}
		})
	}
}

func TestMatchSkillsInMessage_NilLoader(t *testing.T) {
	cb := &ContextBuilder{skillsLoader: nil}
	matched := cb.MatchSkillsInMessage("use weather skill")
	if len(matched) != 0 {
		t.Errorf("expected no matches with nil loader, got %v", matched)
	}
}

func TestLoadSkillContext(t *testing.T) {
	tmpDir := setupWorkspace(t, nil)
	defer os.RemoveAll(tmpDir)

	createTestSkill(
		t,
		filepath.Join(tmpDir, "skills"),
		"weather",
		"weather",
		"Get weather",
		"Call the weather API with the user's location.",
	)

	cb := NewContextBuilder(tmpDir)

	ctx := cb.LoadSkillContext([]string{"weather"})
	if ctx == "" {
		t.Fatal("expected non-empty skill context")
	}
	if !strings.Contains(ctx, "weather API") {
		t.Errorf("skill context should contain skill body, got: %s", ctx)
	}
}

func TestLoadSkillContext_MetadataNameDiffersFromDir(t *testing.T) {
	tmpDir := setupWorkspace(t, nil)
	defer os.RemoveAll(tmpDir)

	createTestSkill(
		t,
		filepath.Join(tmpDir, "skills"),
		"weather-skill",
		"weather",
		"Get weather",
		"Call the weather API (metadata name test).",
	)

	cb := NewContextBuilder(tmpDir)

	matched := cb.MatchSkillsInMessage("use the weather skill")
	if len(matched) == 0 {
		t.Fatal("expected to match skill by metadata name 'weather'")
	}

	ctx := cb.LoadSkillContext(matched)
	if ctx == "" {
		t.Fatal("expected non-empty skill context when metadata name differs from directory name")
	}
	if !strings.Contains(ctx, "metadata name test") {
		t.Errorf("skill context should contain skill body, got: %s", ctx)
	}
}

func TestLoadSkillContext_NilLoader(t *testing.T) {
	cb := &ContextBuilder{skillsLoader: nil}
	ctx := cb.LoadSkillContext([]string{"weather"})
	if ctx != "" {
		t.Errorf("expected empty context with nil loader, got: %s", ctx)
	}
}

func TestBuildMessages_WithSkillContext(t *testing.T) {
	tmpDir := setupWorkspace(t, nil)
	defer os.RemoveAll(tmpDir)

	createTestSkill(t, filepath.Join(tmpDir, "skills"), "weather", "weather", "Get weather", "Call the weather API.")

	cb := NewContextBuilder(tmpDir)
	skillCtx := cb.LoadSkillContext([]string{"weather"})

	messages := cb.BuildMessages(nil, "", "check weather", nil, "telegram", "123", skillCtx)

	if len(messages) < 2 {
		t.Fatalf("expected at least 2 messages (system + user), got %d", len(messages))
	}

	systemContent := messages[0].Content
	if !strings.Contains(systemContent, "Active Skill Instructions") {
		t.Error("system prompt should contain 'Active Skill Instructions' when skill context is provided")
	}
	if !strings.Contains(systemContent, "weather API") {
		t.Error("system prompt should contain the skill body content")
	}
}

func TestBuildMessages_WithoutSkillContext(t *testing.T) {
	tmpDir := setupWorkspace(t, nil)
	defer os.RemoveAll(tmpDir)

	cb := NewContextBuilder(tmpDir)

	messages := cb.BuildMessages(nil, "", "hello", nil, "telegram", "123")

	if len(messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(messages))
	}

	systemContent := messages[0].Content
	if strings.Contains(systemContent, "Active Skill Instructions") {
		t.Error("system prompt should NOT contain skill instructions when no skill context")
	}
}

func TestBuildMessages_EmptySkillContext(t *testing.T) {
	tmpDir := setupWorkspace(t, nil)
	defer os.RemoveAll(tmpDir)

	cb := NewContextBuilder(tmpDir)

	messages := cb.BuildMessages(nil, "", "hello", nil, "telegram", "123", "")

	systemContent := messages[0].Content
	if strings.Contains(systemContent, "Active Skill Instructions") {
		t.Error("system prompt should NOT contain skill instructions when skill context is empty string")
	}
}

func TestIsSkillNameChar(t *testing.T) {
	for _, c := range "abcxyzABCXYZ0189-" {
		if !isSkillNameChar(byte(c)) {
			t.Errorf("expected %q to be a skill name char", string(c))
		}
	}
	for _, c := range " ._/@!,()[]" {
		if isSkillNameChar(byte(c)) {
			t.Errorf("expected %q to NOT be a skill name char", string(c))
		}
	}
}
