package evolution

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/skills"
)

type Store struct {
	paths Paths
}

func NewStore(paths Paths) *Store {
	return &Store{paths: paths}
}

var storeFileLocks sync.Map

func (s *Store) AppendLearningRecord(ctx context.Context, record LearningRecord) error {
	return s.appendLearningRecords(ctx, []LearningRecord{record})
}

func (s *Store) AppendLearningRecords(records []LearningRecord) error {
	return s.appendLearningRecords(context.Background(), records)
}

func (s *Store) appendLearningRecords(ctx context.Context, records []LearningRecord) error {
	if len(records) == 0 {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	unlock := lockStoreFile(s.paths.LearningRecords)
	defer unlock()

	if err := os.MkdirAll(filepath.Dir(s.paths.LearningRecords), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(s.paths.LearningRecords, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, record := range records {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := enc.Encode(record); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) LoadLearningRecords() ([]LearningRecord, error) {
	var records []LearningRecord
	if err := decodeJSONLLines(s.paths.LearningRecords, func(line []byte) error {
		var record LearningRecord
		if err := json.Unmarshal(line, &record); err != nil {
			return err
		}
		records = append(records, record)
		return nil
	}); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *Store) SaveDrafts(drafts []SkillDraft) error {
	unlock := lockStoreFile(s.paths.SkillDrafts)
	defer unlock()

	existing, err := s.LoadDrafts()
	if err != nil {
		return err
	}

	indexByKey := make(map[string]int, len(existing))
	for i, draft := range existing {
		indexByKey[draftKey(draft.WorkspaceID, draft.ID)] = i
	}

	for _, draft := range drafts {
		key := draftKey(draft.WorkspaceID, draft.ID)
		if idx, ok := indexByKey[key]; ok {
			existing[idx] = draft
			continue
		}
		indexByKey[key] = len(existing)
		existing = append(existing, draft)
	}

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(s.paths.SkillDrafts, data, 0o644)
}

func (s *Store) LoadDrafts() ([]SkillDraft, error) {
	data, err := os.ReadFile(s.paths.SkillDrafts)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	var drafts []SkillDraft
	if err := json.Unmarshal(data, &drafts); err != nil {
		return nil, err
	}
	return drafts, nil
}

func (s *Store) SaveProfile(profile SkillProfile) error {
	path, err := s.profilePath(profile.WorkspaceID, profile.SkillName)
	if err != nil {
		return err
	}
	unlock := lockStoreFile(path)
	defer unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(path, data, 0o644)
}

func (s *Store) LoadProfile(skillName string) (SkillProfile, error) {
	paths, err := s.profileLookupPaths(skillName)
	if err != nil {
		return SkillProfile{}, err
	}
	for _, path := range paths {
		profile, loadErr := s.loadProfileFromPath(path)
		if errors.Is(loadErr, os.ErrNotExist) {
			continue
		}
		if loadErr != nil {
			return SkillProfile{}, loadErr
		}
		return profile, nil
	}
	return SkillProfile{}, os.ErrNotExist
}

func (s *Store) LoadProfiles() ([]SkillProfile, error) {
	entries, err := os.ReadDir(s.paths.ProfilesDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	profiles := make([]SkillProfile, 0, len(entries))
	for _, entry := range entries {
		entryPath := filepath.Join(s.paths.ProfilesDir, entry.Name())
		if entry.IsDir() {
			nestedProfiles, loadErr := s.loadProfilesFromDir(entryPath)
			if loadErr != nil {
				return nil, loadErr
			}
			profiles = append(profiles, nestedProfiles...)
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		profile, err := s.loadProfileFromPath(entryPath)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}

	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].SkillName != profiles[j].SkillName {
			return profiles[i].SkillName < profiles[j].SkillName
		}
		return profiles[i].WorkspaceID < profiles[j].WorkspaceID
	})
	return profiles, nil
}

func decodeJSONLLines(path string, decode func(line []byte) error) error {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lines [][]byte
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		lines = append(lines, append([]byte(nil), line...))
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	for i, line := range lines {
		if err := decode(line); err != nil {
			if i == len(lines)-1 && isInvalidJSON(err) {
				return nil
			}
			return err
		}
	}
	return nil
}

func draftKey(workspaceID, id string) string {
	return workspaceID + "\x00" + id
}

func isInvalidJSON(err error) bool {
	var syntaxErr *json.SyntaxError
	return errors.As(err, &syntaxErr)
}

func lockStoreFile(path string) func() {
	actual, _ := storeFileLocks.LoadOrStore(path, &sync.Mutex{})
	mu := actual.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func (s *Store) profilePath(workspaceID, skillName string) (string, error) {
	if err := skills.ValidateSkillName(skillName); err != nil {
		return "", err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return filepath.Join(s.paths.ProfilesDir, skillName+".json"), nil
	}
	return filepath.Join(s.paths.ProfilesDir, workspaceScopeDir(workspaceID), skillName+".json"), nil
}

func (s *Store) loadProfilesFromDir(dir string) ([]SkillProfile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	profiles := make([]SkillProfile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		profile, err := s.loadProfileFromPath(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

func (s *Store) loadProfileFromPath(path string) (SkillProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SkillProfile{}, err
	}

	var profile SkillProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return SkillProfile{}, err
	}
	return profile, nil
}

func (s *Store) profileLookupPaths(skillName string) ([]string, error) {
	if err := skills.ValidateSkillName(skillName); err != nil {
		return nil, err
	}

	paths := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	appendPath := func(path string) {
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		paths = append(paths, path)
		seen[path] = struct{}{}
	}

	if workspaceID := strings.TrimSpace(s.paths.Workspace); workspaceID != "" {
		path, err := s.profilePath(workspaceID, skillName)
		if err != nil {
			return nil, err
		}
		appendPath(path)
	}

	legacyPath, err := s.profilePath("", skillName)
	if err != nil {
		return nil, err
	}
	appendPath(legacyPath)

	matches, err := filepath.Glob(filepath.Join(s.paths.ProfilesDir, "*", skillName+".json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	for _, match := range matches {
		appendPath(match)
	}

	return paths, nil
}

func workspaceScopeDir(workspaceID string) string {
	sum := sha1.Sum([]byte(workspaceID))
	base := filepath.Base(filepath.Clean(workspaceID))
	base = sanitizeWorkspaceComponent(base)
	if base == "" || base == "." {
		base = "workspace"
	}
	return base + "-" + hex.EncodeToString(sum[:6])
}

func sanitizeWorkspaceComponent(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
