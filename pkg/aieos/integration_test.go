package aieos_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sipeed/picoclaw/pkg/aieos"
	"github.com/stretchr/testify/require"
)

// TestIntegration_FullPipeline tests the complete pipeline:
// load real aieos.json → validate → render to prompt text
func TestIntegration_FullPipeline(t *testing.T) {
	// Use the actual workspace/aieos.json template
	repoRoot := findRepoRoot(t)
	templatePath := filepath.Join(repoRoot, "workspace", "aieos.json")

	profile, err := aieos.LoadProfile(templatePath)
	require.NoError(t, err, "should load the real workspace/aieos.json template")

	require.Equal(t, "1.1", profile.Version)
	require.Equal(t, "PicoClaw", profile.Identity.Name)
	require.NotNil(t, profile.Psychology)
	require.NotEmpty(t, profile.Capabilities)

	// Render to prompt
	prompt := aieos.RenderToPrompt(profile)

	fmt.Println("=== AIEOS System Prompt Output ===")
	fmt.Println(prompt)
	fmt.Println("=== END ===")

	require.Contains(t, prompt, "You are **PicoClaw**")
	require.Contains(t, prompt, "## Personality")
	require.Contains(t, prompt, "curious and open")    // openness 0.8
	require.Contains(t, prompt, "organized, thorough") // conscientiousness 0.9
	require.Contains(t, prompt, "calm, stable")        // neuroticism 0.1
	require.Contains(t, prompt, "## Capabilities")
	require.Contains(t, prompt, "web_search")
	require.Contains(t, prompt, "## Communication Style")
	require.Contains(t, prompt, "moderately detailed") // verbosity 0.3 (mid range)
	require.Contains(t, prompt, "## Values & Goals")
	require.Contains(t, prompt, "accuracy")
	require.Contains(t, prompt, "## Boundaries")
	require.Contains(t, prompt, "no_harmful_content")
}

// TestIntegration_FallbackWhenNoAIEOS verifies that an empty workspace has no profile
func TestIntegration_FallbackWhenNoAIEOS(t *testing.T) {
	emptyDir := t.TempDir()
	require.False(t, aieos.ProfileExists(emptyDir))
}

// TestIntegration_FallbackOnInvalidJSON verifies graceful failure
func TestIntegration_FallbackOnInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aieos.json")
	require.NoError(t, os.WriteFile(path, []byte(`{invalid`), 0644))

	require.True(t, aieos.ProfileExists(dir))

	_, err := aieos.LoadProfile(path)
	require.Error(t, err, "should fail on invalid JSON")
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (go.mod)")
		}
		dir = parent
	}
}
