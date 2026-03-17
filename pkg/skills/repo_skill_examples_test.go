package skills

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepoPythonScriptSkillMetadataIsValid(t *testing.T) {
	sl := &SkillsLoader{}
	skillPath := filepath.Join("..", "..", "workspace", "skills", "python-script", "SKILL.md")

	meta := sl.getSkillMetadata(skillPath)
	require.NotNil(t, meta)
	require.Equal(t, "python-script", meta.Name)
	require.NotEmpty(t, meta.Description)

	info := SkillInfo{
		Name:        meta.Name,
		Description: meta.Description,
	}
	require.NoError(t, info.validate())
}
