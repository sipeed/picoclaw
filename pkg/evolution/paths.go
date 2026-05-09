package evolution

import (
	"path/filepath"
	"strings"
)

type Paths struct {
	Workspace       string
	RootDir         string
	LearningRecords string
	TaskRecords     string
	PatternRecords  string
	SkillDrafts     string
	ProfilesDir     string
	BackupsDir      string
}

func NewPaths(workspace, override string) Paths {
	root := strings.TrimSpace(override)
	if root == "" {
		root = filepath.Join(workspace, "state", "evolution")
	}

	return Paths{
		Workspace:       workspace,
		RootDir:         root,
		LearningRecords: filepath.Join(root, "learning-records.jsonl"),
		TaskRecords:     filepath.Join(root, "task-records.jsonl"),
		PatternRecords:  filepath.Join(root, "pattern-records.jsonl"),
		SkillDrafts:     filepath.Join(root, "skill-drafts.json"),
		ProfilesDir:     filepath.Join(root, "profiles"),
		BackupsDir:      filepath.Join(root, "backups"),
	}
}
