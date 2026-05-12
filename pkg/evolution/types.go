package evolution

import "time"

type RecordKind string

const (
	RecordKindTask       RecordKind = "task"
	RecordKindPattern    RecordKind = "pattern"
	legacyRecordKindCase RecordKind = "case"
	legacyRecordKindRule RecordKind = "rule"
	// Deprecated: use RecordKindTask.
	RecordKindCase = RecordKindTask
	// Deprecated: use RecordKindPattern.
	RecordKindRule = RecordKindPattern
)

type RecordStatus string

type DraftType string

const (
	DraftTypeWorkflow DraftType = "workflow"
	DraftTypeShortcut DraftType = "shortcut"
)

type ChangeKind string

const (
	ChangeKindCreate  ChangeKind = "create"
	ChangeKindAppend  ChangeKind = "append"
	ChangeKindReplace ChangeKind = "replace"
	ChangeKindMerge   ChangeKind = "merge"
)

type DraftStatus string

const (
	DraftStatusCandidate   DraftStatus = "candidate"
	DraftStatusQuarantined DraftStatus = "quarantined"
	DraftStatusAccepted    DraftStatus = "accepted"
)

type SkillStatus string

const (
	SkillStatusActive   SkillStatus = "active"
	SkillStatusCold     SkillStatus = "cold"
	SkillStatusArchived SkillStatus = "archived"
	SkillStatusDeleted  SkillStatus = "deleted"
)

type AttemptTrail struct {
	AttemptedSkills       []string               `json:"attempted_skills,omitempty"`
	FinalSuccessfulPath   []string               `json:"final_successful_path,omitempty"`
	SkillContextSnapshots []SkillContextSnapshot `json:"skill_context_snapshots,omitempty"`
}

type SkillContextSnapshot struct {
	Sequence   int      `json:"sequence"`
	Trigger    string   `json:"trigger"`
	SkillNames []string `json:"skill_names,omitempty"`
}

type ToolExecutionRecord struct {
	Name         string   `json:"name"`
	Success      bool     `json:"success"`
	ErrorSummary string   `json:"error_summary,omitempty"`
	SkillNames   []string `json:"skill_names,omitempty"`
}

type LearningRecord struct {
	ID                   string                `json:"id"`
	Kind                 RecordKind            `json:"kind"`
	WorkspaceID          string                `json:"workspace_id"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            *time.Time            `json:"updated_at,omitempty"`
	SessionKey           string                `json:"session_key,omitempty"`
	TaskHash             string                `json:"task_hash,omitempty"`
	Summary              string                `json:"summary"`
	UserGoal             string                `json:"user_goal,omitempty"`
	FinalOutput          string                `json:"final_output,omitempty"`
	Source               map[string]any        `json:"source,omitempty"`
	Status               RecordStatus          `json:"status"`
	Success              *bool                 `json:"success,omitempty"`
	ToolKinds            []string              `json:"tool_kinds,omitempty"`
	ToolExecutions       []ToolExecutionRecord `json:"tool_executions,omitempty"`
	InitialSkillNames    []string              `json:"initial_skill_names,omitempty"`
	AddedSkillNames      []string              `json:"added_skill_names,omitempty"`
	UsedSkillNames       []string              `json:"used_skill_names,omitempty"`
	AllLoadedSkillNames  []string              `json:"all_loaded_skill_names,omitempty"`
	ActiveSkillNames     []string              `json:"active_skill_names,omitempty"`
	AttemptTrail         *AttemptTrail         `json:"attempt_trail,omitempty"`
	Signals              []string              `json:"signals,omitempty"`
	SourceRecordIDs      []string              `json:"source_record_ids,omitempty"`
	TaskRecordIDs        []string              `json:"task_record_ids,omitempty"`
	Label                string                `json:"label,omitempty"`
	ClusterReason        string                `json:"cluster_reason,omitempty"`
	EventCount           int                   `json:"event_count,omitempty"`
	SuccessRate          float64               `json:"success_rate,omitempty"`
	MaturityScore        float64               `json:"maturity_score,omitempty"`
	WinningPath          []string              `json:"winning_path,omitempty"`
	LateAddedSkills      []string              `json:"late_added_skills,omitempty"`
	FinalSnapshotTrigger string                `json:"final_snapshot_trigger,omitempty"`
	MatchedSkillNames    []string              `json:"matched_skill_names,omitempty"`
}

type SkillDraft struct {
	ID                 string      `json:"id"`
	WorkspaceID        string      `json:"workspace_id"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          *time.Time  `json:"updated_at,omitempty"`
	SourceRecordID     string      `json:"source_record_id"`
	TargetSkillName    string      `json:"target_skill_name"`
	MatchedSkillRefs   []string    `json:"matched_skill_refs,omitempty"`
	DraftType          DraftType   `json:"draft_type"`
	ChangeKind         ChangeKind  `json:"change_kind"`
	HumanSummary       string      `json:"human_summary"`
	IntendedUseCases   []string    `json:"intended_use_cases,omitempty"`
	PreferredEntryPath []string    `json:"preferred_entry_path,omitempty"`
	AvoidPatterns      []string    `json:"avoid_patterns,omitempty"`
	BodyOrPatch        string      `json:"body_or_patch"`
	Status             DraftStatus `json:"status"`
	ReviewNotes        []string    `json:"review_notes,omitempty"`
	ScanFindings       []string    `json:"scan_findings,omitempty"`
}

type SkillVersionEntry struct {
	Version        string    `json:"version"`
	Action         string    `json:"action"`
	Timestamp      time.Time `json:"timestamp"`
	DraftID        string    `json:"draft_id,omitempty"`
	Summary        string    `json:"summary"`
	Rollback       bool      `json:"rollback,omitempty"`
	RollbackReason string    `json:"rollback_reason,omitempty"`
}

type SkillProfile struct {
	SkillName          string              `json:"skill_name"`
	WorkspaceID        string              `json:"workspace_id"`
	CurrentVersion     string              `json:"current_version"`
	Status             SkillStatus         `json:"status"`
	Origin             string              `json:"origin"`
	HumanSummary       string              `json:"human_summary"`
	ChangeReason       string              `json:"change_reason,omitempty"`
	IntendedUseCases   []string            `json:"intended_use_cases,omitempty"`
	PreferredEntryPath []string            `json:"preferred_entry_path,omitempty"`
	AvoidPatterns      []string            `json:"avoid_patterns,omitempty"`
	LastUsedAt         time.Time           `json:"last_used_at"`
	UseCount           int                 `json:"use_count"`
	RetentionScore     float64             `json:"retention_score"`
	VersionHistory     []SkillVersionEntry `json:"version_history"`
}
