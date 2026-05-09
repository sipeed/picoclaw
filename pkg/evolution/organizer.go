package evolution

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

type OrganizerOptions struct {
	MinCaseCount   int
	MinSuccessRate float64
	Now            func() time.Time
}

type Organizer struct {
	minCaseCount   int
	minSuccessRate float64
	now            func() time.Time
}

func NewOrganizer(opts OrganizerOptions) *Organizer {
	now := opts.Now
	if now == nil {
		now = time.Now
	}

	minCaseCount := opts.MinCaseCount
	if minCaseCount <= 0 {
		minCaseCount = 3
	}

	minSuccessRate := opts.MinSuccessRate
	if minSuccessRate <= 0 {
		minSuccessRate = 0.7
	}

	return &Organizer{
		minCaseCount:   minCaseCount,
		minSuccessRate: minSuccessRate,
		now:            now,
	}
}

func (o *Organizer) BuildRules(records []LearningRecord) ([]LearningRecord, error) {
	clusters := make(map[string][]LearningRecord)
	keys := make([]string, 0)

	for _, record := range records {
		if !isTaskRecordKind(record.Kind) {
			continue
		}

		key := normalizeRuleKey(record)
		if key == "" {
			continue
		}

		clusterKey := record.WorkspaceID + "\x00" + key
		if _, ok := clusters[clusterKey]; !ok {
			keys = append(keys, clusterKey)
		}
		clusters[clusterKey] = append(clusters[clusterKey], record)
	}

	sort.Strings(keys)

	rules := make([]LearningRecord, 0, len(keys))
	for _, clusterKey := range keys {
		cluster := append([]LearningRecord(nil), clusters[clusterKey]...)
		sortCaseCluster(cluster)

		if len(cluster) < o.minCaseCount {
			continue
		}

		successRate := clusterSuccessRate(cluster)
		if successRate < o.minSuccessRate {
			continue
		}

		ruleKey := clusterKey[strings.Index(clusterKey, "\x00")+1:]
		winningPath := clusterWinningPath(cluster)
		lateAddedSkills, finalSnapshotTrigger := clusterLateAddedSkills(cluster, winningPath)
		matchedSkillNames := append([]string(nil), winningPath...)

		rules = append(rules, LearningRecord{
			ID:                   stableRuleID(cluster[0].WorkspaceID, ruleKey),
			Kind:                 RecordKindPattern,
			WorkspaceID:          cluster[0].WorkspaceID,
			CreatedAt:            o.now(),
			Summary:              buildRuleSummary(cluster, ruleKey, winningPath),
			Source:               map[string]any{"cluster_key": ruleKey},
			Status:               RecordStatus("ready"),
			SourceRecordIDs:      collectRecordIDs(cluster),
			EventCount:           len(cluster),
			SuccessRate:          successRate,
			MaturityScore:        computeMaturityScore(len(cluster), successRate),
			WinningPath:          winningPath,
			LateAddedSkills:      lateAddedSkills,
			FinalSnapshotTrigger: finalSnapshotTrigger,
			MatchedSkillNames:    matchedSkillNames,
		})
	}

	return rules, nil
}

func normalizeRuleKey(record LearningRecord) string {
	if path := preferredRulePath(record); len(path) > 0 {
		return strings.Join(path, " ")
	}
	if path := normalizePath(record.ToolKinds); len(path) > 0 {
		return strings.Join(path, " ")
	}

	tokens := tokenizeForEvolution(record.Summary)
	if len(tokens) == 0 {
		return ""
	}
	if len(tokens) > 6 {
		tokens = tokens[:6]
	}
	return strings.Join(tokens, " ")
}

func preferredRulePath(record LearningRecord) []string {
	if path := normalizeFinalSuccessfulPath(record); len(path) > 0 {
		return path
	}
	if path := normalizePath(record.UsedSkillNames); len(path) > 0 {
		return path
	}
	if path := normalizePath(record.AddedSkillNames); len(path) > 0 {
		return path
	}
	if path := normalizeAttemptedSkills(record); len(path) > 0 {
		return path
	}
	if path := normalizePath(record.ActiveSkillNames); len(path) > 0 {
		return path
	}
	if path := normalizePath(record.MatchedSkillNames); len(path) > 0 {
		return path
	}
	return nil
}

func normalizePath(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeFinalSuccessfulPath(record LearningRecord) []string {
	if record.AttemptTrail == nil {
		return nil
	}
	return normalizePath(record.AttemptTrail.FinalSuccessfulPath)
}

func normalizeAttemptedSkills(record LearningRecord) []string {
	if record.AttemptTrail == nil {
		return nil
	}
	return normalizePath(record.AttemptTrail.AttemptedSkills)
}

func sortCaseCluster(cluster []LearningRecord) {
	sort.Slice(cluster, func(i, j int) bool {
		if !cluster[i].CreatedAt.Equal(cluster[j].CreatedAt) {
			return cluster[i].CreatedAt.Before(cluster[j].CreatedAt)
		}
		return cluster[i].ID < cluster[j].ID
	})
}

func clusterSuccessRate(cluster []LearningRecord) float64 {
	if len(cluster) == 0 {
		return 0
	}

	successes := 0
	for _, record := range cluster {
		if record.Success != nil && *record.Success {
			successes++
		}
	}
	return float64(successes) / float64(len(cluster))
}

func clusterWinningPath(cluster []LearningRecord) []string {
	type pathScore struct {
		path  []string
		count int
	}

	bestKey := ""
	best := pathScore{}
	paths := make(map[string]pathScore)
	order := make([]string, 0)

	for _, record := range cluster {
		path := preferredRulePath(record)
		if len(path) == 0 {
			path = normalizePath(record.ToolKinds)
		}
		if len(path) == 0 {
			continue
		}

		key := strings.Join(path, "\x00")
		score := paths[key]
		if score.path == nil {
			score.path = append([]string(nil), path...)
			order = append(order, key)
		}
		score.count++
		paths[key] = score
	}

	for _, key := range order {
		score := paths[key]
		if score.count > best.count {
			best = score
			bestKey = key
		}
	}

	if bestKey == "" {
		return nil
	}
	return best.path
}

func clusterLateAddedSkills(cluster []LearningRecord, winningPath []string) ([]string, string) {
	type lateAddedScore struct {
		skills  []string
		trigger string
		count   int
	}

	bestKey := ""
	best := lateAddedScore{}
	scores := make(map[string]lateAddedScore)
	order := make([]string, 0)

	for _, record := range cluster {
		skills, trigger := lateAddedSkillsFromRecord(record)
		if len(skills) == 0 {
			continue
		}
		if len(winningPath) > 0 && !pathsEqual(skills, tailAddedWithinWinningPath(winningPath, skills)) {
			continue
		}

		key := trigger + "\x00" + strings.Join(skills, "\x00")
		score := scores[key]
		if score.skills == nil {
			score.skills = append([]string(nil), skills...)
			score.trigger = trigger
			order = append(order, key)
		}
		score.count++
		scores[key] = score
	}

	for _, key := range order {
		score := scores[key]
		if score.count > best.count {
			bestKey = key
			best = score
		}
	}

	if bestKey == "" {
		return nil, ""
	}
	return best.skills, best.trigger
}

func lateAddedSkillsFromRecord(record LearningRecord) ([]string, string) {
	if skills := normalizePath(record.AddedSkillNames); len(skills) > 0 {
		return skills, "loaded_during_task"
	}
	if record.AttemptTrail == nil || len(record.AttemptTrail.SkillContextSnapshots) == 0 {
		return nil, ""
	}

	snapshots := record.AttemptTrail.SkillContextSnapshots
	last := snapshots[len(snapshots)-1]
	if len(last.SkillNames) == 0 {
		return nil, ""
	}
	if len(snapshots) == 1 {
		return nil, strings.TrimSpace(last.Trigger)
	}

	prev := snapshots[len(snapshots)-2]
	prevSet := make(map[string]struct{}, len(prev.SkillNames))
	for _, skill := range normalizePath(prev.SkillNames) {
		prevSet[skill] = struct{}{}
	}

	added := make([]string, 0, len(last.SkillNames))
	for _, skill := range normalizePath(last.SkillNames) {
		if _, ok := prevSet[skill]; ok {
			continue
		}
		added = append(added, skill)
	}
	if len(added) == 0 {
		return nil, strings.TrimSpace(last.Trigger)
	}
	return added, strings.TrimSpace(last.Trigger)
}

func tailAddedWithinWinningPath(winningPath, lateAdded []string) []string {
	if len(winningPath) == 0 || len(lateAdded) == 0 || len(lateAdded) > len(winningPath) {
		return nil
	}
	tail := winningPath[len(winningPath)-len(lateAdded):]
	if !pathsEqual(tail, lateAdded) {
		return nil
	}
	return tail
}

func pathsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func collectRecordIDs(cluster []LearningRecord) []string {
	ids := make([]string, 0, len(cluster))
	for _, record := range cluster {
		ids = append(ids, record.ID)
	}
	return ids
}

func computeMaturityScore(caseCount int, successRate float64) float64 {
	return float64(caseCount) * successRate
}

func stableRuleID(workspaceID, key string) string {
	sum := sha1.Sum([]byte(workspaceID + "\x00" + key))
	return "rule-" + hex.EncodeToString(sum[:6])
}

func buildRuleSummary(cluster []LearningRecord, key string, winningPath []string) string {
	if goal := representativeGoal(cluster); goal != "" && len(winningPath) > 0 {
		return goal + " via " + strings.Join(winningPath, " -> ")
	}
	if goal := representativeGoal(cluster); goal != "" {
		return goal
	}
	if len(winningPath) > 0 {
		return strings.Join(winningPath, " -> ")
	}
	return key
}

func representativeGoal(cluster []LearningRecord) string {
	for _, record := range cluster {
		if goal := strings.TrimSpace(record.UserGoal); goal != "" {
			return goal
		}
	}
	for _, record := range cluster {
		if summary := strings.TrimSpace(record.Summary); summary != "" {
			return summary
		}
	}
	return ""
}
