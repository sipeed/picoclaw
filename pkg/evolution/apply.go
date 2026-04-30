package evolution

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/skills"
)

type Applier struct {
	paths Paths
	now   func() time.Time
}

func NewApplier(paths Paths, now func() time.Time) *Applier {
	if now == nil {
		now = time.Now
	}
	return &Applier{
		paths: paths,
		now:   now,
	}
}

func (a *Applier) ApplyDraft(ctx context.Context, workspace string, draft SkillDraft) error {
	rollback, err := a.applyDraftWithRollback(ctx, workspace, draft)
	if err != nil {
		return err
	}
	_ = rollback
	return nil
}

func (a *Applier) applyDraftWithRollback(ctx context.Context, workspace string, draft SkillDraft) (func() error, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if err := skills.ValidateSkillName(draft.TargetSkillName); err != nil {
		return nil, err
	}

	existingBody, backupPath, hadOriginal, err := a.backupCurrentSkill(workspace, draft.TargetSkillName)
	if err != nil {
		return nil, err
	}

	skillDir := filepath.Join(workspace, "skills", draft.TargetSkillName)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return nil, err
	}

	renderedBody, err := renderAppliedBody(draft, existingBody, hadOriginal)
	if err != nil {
		return nil, err
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := fileutil.WriteFileAtomic(skillPath, []byte(renderedBody), 0o644); err != nil {
		return nil, err
	}

	if err := validateAppliedSkillBody(renderedBody); err != nil {
		if rollbackErr := a.rollbackSkill(skillPath, backupPath, hadOriginal); rollbackErr != nil {
			return nil, errorsJoin(err, rollbackErr)
		}
		return nil, err
	}

	return func() error {
		return a.rollbackSkill(skillPath, backupPath, hadOriginal)
	}, nil
}

func (a *Applier) backupCurrentSkill(workspace, skillName string) (currentBody, backupPath string, hadOriginal bool, err error) {
	if err := skills.ValidateSkillName(skillName); err != nil {
		return "", "", false, err
	}

	skillPath := filepath.Join(workspace, "skills", skillName, "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if os.IsNotExist(err) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}

	backupDir := filepath.Join(a.paths.BackupsDir, skillName, a.now().Format("20060102-150405"))
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", "", false, err
	}

	backupPath = filepath.Join(backupDir, "SKILL.md")
	if err := fileutil.WriteFileAtomic(backupPath, data, 0o644); err != nil {
		return "", "", false, err
	}
	return string(data), backupPath, true, nil
}

func (a *Applier) rollbackSkill(skillPath, backupPath string, hadOriginal bool) error {
	if hadOriginal {
		data, err := os.ReadFile(backupPath)
		if err != nil {
			return err
		}
		return fileutil.WriteFileAtomic(skillPath, data, 0o644)
	}
	if err := os.Remove(skillPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	skillDir := filepath.Dir(skillPath)
	if err := os.Remove(skillDir); err != nil && !os.IsNotExist(err) && !isDirNotEmptyError(err) {
		return err
	}
	return nil
}

func isDirNotEmptyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "directory not empty")
}

func validateAppliedSkillBody(body string) error {
	body = strings.TrimSpace(body)
	if !strings.HasPrefix(body, "---\n") {
		return fmt.Errorf("skill frontmatter is required")
	}
	if !strings.Contains(body, "\n# ") {
		return fmt.Errorf("skill heading is required")
	}
	return nil
}

func renderAppliedBody(draft SkillDraft, existingBody string, hadOriginal bool) (string, error) {
	switch draft.ChangeKind {
	case ChangeKindCreate:
		if hadOriginal {
			return "", fmt.Errorf("cannot create skill %q: skill already exists", draft.TargetSkillName)
		}
		return draft.BodyOrPatch, nil
	case ChangeKindReplace:
		if !hadOriginal {
			return "", fmt.Errorf("cannot replace skill %q: skill does not exist", draft.TargetSkillName)
		}
		return draft.BodyOrPatch, nil
	case ChangeKindAppend:
		if !hadOriginal || strings.TrimSpace(existingBody) == "" {
			return draft.BodyOrPatch, nil
		}
		return strings.TrimRight(existingBody, "\n") + "\n\n" + strings.TrimLeft(draft.BodyOrPatch, "\n"), nil
	case ChangeKindMerge:
		if !hadOriginal || strings.TrimSpace(existingBody) == "" {
			return draft.BodyOrPatch, nil
		}
		mergedSection := strings.Join([]string{
			"",
			"## Merged Knowledge",
			strings.TrimSpace(draft.BodyOrPatch),
			"",
		}, "\n")
		return strings.TrimRight(existingBody, "\n") + mergedSection, nil
	default:
		return "", fmt.Errorf("unsupported change_kind %q", draft.ChangeKind)
	}
}

func errorsJoin(errs ...error) error {
	var first error
	for _, err := range errs {
		if err == nil {
			continue
		}
		if first == nil {
			first = err
			continue
		}
		first = fmt.Errorf("%w; %v", first, err)
	}
	return first
}
