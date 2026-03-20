package skills

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	skillspkg "github.com/sipeed/picoclaw/pkg/skills"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return buf.String()
}

func TestSkillsListCmd_EmptyOutputIncludesSearchRoots(t *testing.T) {
	tmp := t.TempDir()
	workspace := filepath.Join(tmp, "workspace")
	global := filepath.Join(tmp, "global")
	builtin := filepath.Join(tmp, "builtin")
	loader := skillspkg.NewSkillsLoader(workspace, global, builtin)

	output := captureStdout(t, func() {
		skillsListCmd(loader)
	})

	assert.Contains(t, output, "No skills installed.")
	assert.Contains(t, output, "Scanned skill roots:")
	assert.Contains(t, output, filepath.Join(workspace, "skills"))
	assert.Contains(t, output, global)
	assert.Contains(t, output, builtin)
	assert.Contains(t, output, filepath.Join(workspace, "skills", "<skill-name>", "SKILL.md"))
}

func TestSkillsListCmd_ShowsInstalledSkills(t *testing.T) {
	tmp := t.TempDir()
	workspace := filepath.Join(tmp, "workspace")
	skillDir := filepath.Join(workspace, "skills", "weather")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	content := "---\nname: weather\ndescription: Weather lookup\n---\n\n# Weather\n"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644))

	loader := skillspkg.NewSkillsLoader(workspace, filepath.Join(tmp, "global"), filepath.Join(tmp, "builtin"))
	output := captureStdout(t, func() {
		skillsListCmd(loader)
	})

	assert.Contains(t, output, "Installed Skills:")
	assert.Contains(t, output, "weather (workspace)")
	assert.Contains(t, output, "Weather lookup")
	assert.NotContains(t, output, "Scanned skill roots:")
}
