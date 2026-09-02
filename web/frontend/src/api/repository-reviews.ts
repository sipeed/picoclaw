import type {
  CollectionListRequest,
  CollectionPageMetadata,
} from "@/api/collection"
import { collectionListURL, collectionRequest } from "@/api/collection"
import { launcherFetch } from "@/api/http"

export type RepositoryReviewFindingStatus = "open" | "dismissed" | "posted"
export type RepositoryReviewIssueDraftState =
  | "generating"
  | "failed"
  | "editing"
  | "publishing"
  | "posted"
  | "unknown"
export type RepositoryReviewIssueDraftOrigin =
  | "ai_generated"
  | "linked"
  | "discovered"
  | "legacy"
export type RepositoryReviewPublishBlockerCode =
  | "repository_not_github"
  | "preview_not_canonical"
  | "origin_not_publishable"
  | "state_not_publishable"
  | "finding_missing"
  | "finding_status_unresolved"
  | "duplicate_review_required"
  | "issue_association_conflict"
  | "historical_merge_in_progress"
  | "finding_not_publishable"
export type RepositoryReviewPurgeBlockerCode =
  | "review_active"
  | "finding_processing_active"
  | "resolution_check_active"
  | "issue_generation_active"
  | "publication_active"
  | "historical_consolidation_active"
  | "retention_unavailable"
export type RepositoryReviewIssueInstructionsMode = "default" | "custom"
export type RepositoryReviewFindingsScope = "current" | "all"
/** @deprecated Use RepositoryReviewFindingsScope. */
export type RepositoryReviewReportScope = RepositoryReviewFindingsScope
export type RepositoryReviewMatchState = "new" | "known" | "provisional"
export type RepositoryReviewRunFindingStatusState =
  | "pending"
  | "processing"
  | "failed"
  | "associated_new"
  | "associated_existing"
  | "needs_review"
export type RepositoryFindingLifecycle =
  | "open"
  | "resolution_pending"
  | "resolved"
  | "regressed"
  | "dismissed"
export type RepositoryFindingIssueState =
  | "none"
  | "draft"
  | "open"
  | "closed"
  | "unknown"
export type RepositoryFindingValidationState =
  | "not_requested"
  | "pending"
  | "running"
  | "confirmed"
  | "not_fixed"
  | "inconclusive"
  | "failed"
export type RepositoryMappingJobState = "pending" | "running" | "completed"
export type RepositoryReviewDeduplicationState =
  | "pending"
  | "running"
  | "completed"
  | "failed"
export type RepositoryReviewRawFindingDisposition =
  | "undecided"
  | "new"
  | "duplicate"
export type RepositoryReviewHistoricalDeduplicationStatus =
  | "pending"
  | "replaying"
  | "merging"
  | "failed"
  | "completed"
export type RepositoryReviewHistoricalConsolidationStatus =
  | "not_required"
  | RepositoryReviewHistoricalDeduplicationStatus
export type RepositoryReviewFixEffortClass =
  | "tiny"
  | "small"
  | "medium"
  | "large"
  | "refactor"

export const repositoryReviewDefaultIssuePrompt =
  "Present the confirmed diagnosis concisely. Include evidence, impact, validation already performed, the exact location, and commit/blob provenance. Do not include a fix or advice."

export interface RepositoryReviewMatchHints {
  component: string
  operation: string
  failure_mode: string
  trigger: string
  violated_invariant: string
  observable_outcome: string
  related_symbols: string[]
  source_anchors: string[]
  distinguishing_facts: string[]
}

export interface RepositoryReviewFixEffortEstimate {
  loc_min: number
  loc_max: number
  class: RepositoryReviewFixEffortClass | string
  rationale: string
}

export interface RepositoryReviewFixEffort {
  quick: RepositoryReviewFixEffortEstimate
  quality: RepositoryReviewFixEffortEstimate
}

export interface RepositoryReviewFileRef {
  path: string
  blob_sha: string
  size_bytes: number
  category?: string
  mode?: string
}

export interface RepositoryReviewedFile extends RepositoryReviewFileRef {
  commit_sha: string
  profile_hash: string
  run_id: string
  reviewed_at: string
}

export interface RepositoryUnsupportedFile extends RepositoryReviewFileRef {
  commit_sha: string
  profile_hash: string
  reason: string
  updated_at: string
}

export interface RepositoryReviewValidation {
  status: string
  summary: string
  checks?: string[]
}

export interface RepositoryReviewFindingContext {
  id: string
  repository: string
  commit_sha: string
  inventory_hash: string
  profile_hash?: string
  run_id: string
  model: string
  model_alias?: string
  account?: string
  reviewer?: string
  files: RepositoryReviewFileRef[]
  raw_digest?: string
  created_at: string
}

export interface RepositoryReviewFinding {
  id: string
  fingerprint: string
  repository: string
  commit_sha: string
  file: RepositoryReviewFileRef
  line?: number
  severity: string
  title: string
  symbol?: string
  message?: string
  evidence: string
  impact: string
  validation: RepositoryReviewValidation
  match_hints?: RepositoryReviewMatchHints
  fix_effort?: RepositoryReviewFixEffort
  context_ids: string[]
  models: string[]
  observation_count: number
  observations?: RepositoryReviewFindingObservation[]
  status: RepositoryReviewFindingStatus
  issue_draft_id?: string
  repository_finding_id?: string
  repository_match_state?: RepositoryReviewMatchState
  run_finding_status?: RepositoryReviewRunFindingStatusState
  target_branch?: string
  advertised_default_branch?: string
  target_is_default?: boolean
  version: number
  created_at: string
  updated_at: string
  raw_source_ids?: string[]
  raw_source_total?: number
}

export interface RepositoryReviewRawFindingHistoryEntry {
  state: RepositoryReviewDeduplicationState
  disposition: RepositoryReviewRawFindingDisposition
  deduplicated_finding_id?: string
  attempt?: number
  failure?: RepositoryReviewDeduplicationFailure
  at: string
}

export interface RepositoryReviewDeduplicationFailure {
  code: string
  message: string
  retryable: boolean
  at: string
}

export interface RepositoryReviewRawFinding {
  id: string
  version?: number
  campaign_id?: string
  admission_bucket?: string
  insertion_ordinal?: number
  diagnosis_digest?: string
  legacy_finding_id?: string
  repository?: string
  commit_sha?: string
  file?: RepositoryReviewFileRef
  path?: string
  line?: number
  severity: string
  title: string
  symbol?: string
  message?: string
  evidence?: string
  impact?: string
  validation?: RepositoryReviewValidation
  match_hints?: RepositoryReviewMatchHints
  fix_effort?: RepositoryReviewFixEffort
  context_id?: string
  run_id?: string
  assignment_id?: string
  model: string
  model_alias?: string
  account?: string
  reviewer?: string
  deduplication_state: RepositoryReviewDeduplicationState
  disposition: RepositoryReviewRawFindingDisposition
  deduplicated_finding_id?: string
  history?: RepositoryReviewRawFindingHistoryEntry[]
  failure?: RepositoryReviewDeduplicationFailure
  created_at: string
  updated_at: string
}

export interface RepositoryReviewHistoricalDeduplication {
  required: boolean
  status?: RepositoryReviewHistoricalDeduplicationStatus
  attempts?: number
  error?: string
  updated_at?: string
}

export interface RepositoryReviewFindingsProcessingCounters {
  raw_total: number
  pending: number
  processing: number
  failed: number
  completed: number
  new: number
  duplicates: number
  updated_at?: string
}

export interface RepositoryReviewFindingHealthRunFindings {
  total: number
  pending: number
  processing: number
  failed: number
  needs_review: number
  associated_new: number
  associated_existing: number
  unrepresented: number
}

export interface RepositoryReviewFindingHealthRepositoryFindings {
  total: number
  provisional: number
  validation_failed: number
  issue_conflicts: number
}

export interface RepositoryReviewFindingHealthProcessing {
  total: number
  pending: number
  processing: number
  failed: number
  completed: number
}

export interface RepositoryReviewHistoricalConsolidation {
  required: boolean
  status: RepositoryReviewHistoricalConsolidationStatus
  retryable: boolean
}

export interface RepositoryReviewFindingHealth {
  run_findings: RepositoryReviewFindingHealthRunFindings
  repository_findings: RepositoryReviewFindingHealthRepositoryFindings
  findings_processing: RepositoryReviewFindingHealthProcessing
  historical_consolidation: RepositoryReviewHistoricalConsolidation
  updated_at: string
}

export interface RepositoryReviewRawFindingsPage {
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  campaign_id?: string
  finding_id?: string
  findings_processing?: RepositoryReviewFindingsProcessingCounters
  sources?: RepositoryReviewRawFinding[]
  raw_findings?: RepositoryReviewRawFinding[]
  offset: number
  total: number
  next_offset?: number
}

export interface RepositoryReviewRawFindingDetail {
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  source: RepositoryReviewRawFinding
  context?: RepositoryReviewFindingContext
  finding?: RepositoryReviewFinding
  findings_processing?: RepositoryReviewFindingsProcessingCounters
  historical_deduplication?: RepositoryReviewHistoricalDeduplication
}

export interface RepositoryReviewRawFindingsCollectionPage extends CollectionPageMetadata {
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  raw_findings: RepositoryReviewRawFinding[]
  findings_processing?: RepositoryReviewFindingsProcessingCounters
  historical_deduplication?: RepositoryReviewHistoricalDeduplication
}

export interface RepositoryReviewFindingsProcessingCollectionPage extends CollectionPageMetadata {
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  sources: RepositoryReviewRawFinding[]
  findings_processing: RepositoryReviewFindingHealthProcessing
  historical_consolidation?: RepositoryReviewHistoricalConsolidation
  capabilities?: RepositoryReviewCapabilities
}

export interface RepositoryReviewFindingsProcessingDetail {
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  source: RepositoryReviewRawFinding
  context?: RepositoryReviewFindingContext
  finding?: RepositoryReviewFinding
  repository_finding?: RepositoryFinding
  findings_processing?: RepositoryReviewFindingHealthProcessing
  historical_consolidation?: RepositoryReviewHistoricalConsolidation
  health?: RepositoryReviewFindingHealth
}

export interface RepositoryReviewFindingsProcessingRetryFailure {
  source_id: string
  code: string
  message: string
}

export interface RepositoryReviewFindingsProcessingRetryResponse {
  retried_ids: string[]
  failures: RepositoryReviewFindingsProcessingRetryFailure[]
  findings_processing: RepositoryReviewFindingHealthProcessing
  health: RepositoryReviewFindingHealth
}

export interface RepositoryReviewFindingObservation {
  context_id: string
  model: string
  model_alias?: string
  account?: string
  reviewer?: string
  severity: string
  title: string
  symbol?: string
  line?: number
  message?: string
  evidence: string
  impact: string
  validation: RepositoryReviewValidation
  match_hints?: RepositoryReviewMatchHints
  fix_effort?: RepositoryReviewFixEffort
}

export interface RepositoryReviewRun {
  id: string
  plan_id: string
  commit_sha: string
  inventory_hash: string
  reviewed_files: number
  unreviewed_files: number
  unsupported_files: number
  remaining_files: number
  unreviewed_paths?: string[]
  unsupported_paths?: string[]
  skipped_files: number
  excluded_files?: number
  accepted_findings: number
  rejected_findings: number
  models: string[]
  target_branch?: string
  advertised_default_branch?: string
  target_is_default?: boolean
  completed_at: string
}

export interface RepositoryReviewIssueDraft {
  id: string
  repository: string
  finding_ids: string[]
  origin?: RepositoryReviewIssueDraftOrigin
  generation_id?: string
  resolved_instructions?: string
  instructions_mode?: RepositoryReviewIssueInstructionsMode
  generator_model?: string
  generator_account?: string
  generator_profile_id?: string
  generator_profile_version?: number
  attempt_generation_id?: string
  attempt_resolved_instructions?: string
  attempt_instructions_mode?: RepositoryReviewIssueInstructionsMode
  attempt_generator_model?: string
  attempt_generator_account?: string
  attempt_generator_profile_id?: string
  attempt_generator_profile_version?: number
  generation_error?: string
  canonical?: boolean
  read_only?: boolean
  publishable?: boolean
  deletable?: boolean
  regeneratable?: boolean
  unlinkable?: boolean
  conflict_reason?: string
  title: string
  body: string
  labels?: string[]
  state: RepositoryReviewIssueDraftState
  external_id?: string
  external_url?: string
  external_state?: "open" | "closed" | string
  version: number
  created_at: string
  updated_at: string
}

export interface RepositoryReviewPublishBlocker {
  code: RepositoryReviewPublishBlockerCode
  count: number
  message: string
}

export interface RepositoryReviewPurgeBlocker {
  code: RepositoryReviewPurgeBlockerCode
  count: number
  message: string
}

export interface RepositoryReviewPurgeSummary {
  repository_version: number
  ledger_fence: string
  raw_findings: number
  deduplicated_findings: number
  repository_findings: number
  issue_previews: number
  external_issue_associations: number
}

export interface RepositoryReviewCapabilities {
  github?: boolean
  can_generate?: boolean
  can_publish?: boolean
  can_search_issues?: boolean
  can_link_issue?: boolean
  can_unlink_issue?: boolean
  can_replace_issue?: boolean
  can_edit?: boolean
  can_delete?: boolean
  can_regenerate?: boolean
  read_only_reason?: string
  publish_blockers?: RepositoryReviewPublishBlocker[]
  can_purge_history?: boolean
  can_remove_repository?: boolean
  purge_blockers?: RepositoryReviewPurgeBlocker[]
  purge_summary?: RepositoryReviewPurgeSummary
}

export interface RepositoryReviewFindingDetail {
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  finding: RepositoryReviewFinding
  action_finding?: RepositoryReviewFinding
  contexts: RepositoryReviewFindingContext[]
  issue?: RepositoryReviewIssueDraft
  repository_finding?: RepositoryFinding
  occurrences?: RepositoryReviewFinding[]
  possible_duplicate_findings?: RepositoryFinding[]
  raw_source_total?: number
  capabilities?: RepositoryReviewCapabilities
}

export interface RepositoryFindingPathSymbol {
  review_finding_id: string
  commit_sha: string
  path: string
  symbol?: string
  default_branch_verified?: boolean
  observed_at: string
}

export interface RepositoryFindingPossibleDuplicate {
  candidate_id: string
  relation: string
  confidence: number
  matching_anchors?: string[]
  conflicting_anchors?: string[]
  explanation?: string
  created_at: string
}

export interface RepositoryFindingIssueAssociation {
  external_id?: string
  url?: string
  origin?: RepositoryReviewIssueDraftOrigin
  state: RepositoryFindingIssueState
  title?: string
  snapshot_at?: string
  conflict?: boolean
  conflict_urls?: string[]
}

export interface RepositoryValidationFailure {
  code: string
  message: string
  retryable: boolean
  at: string
}

export interface RepositoryFindingResolution {
  outcome: RepositoryFindingValidationState
  fix_commit_sha?: string
  fix_commit_time?: string
  validated_at: string
  first_containing_tag?: string
  summary?: string
  failure?: RepositoryValidationFailure
}

export interface RepositoryFinding {
  id: string
  repository: string
  canonical_title: string
  canonical_severity: string
  match_hints?: RepositoryReviewMatchHints
  fix_effort?: RepositoryReviewFixEffort
  review_finding_ids: string[]
  occurrence_count?: number
  found_commits: string[]
  found_commit_count?: number
  path_symbol_history: RepositoryFindingPathSymbol[]
  match_state: RepositoryReviewMatchState
  lifecycle: RepositoryFindingLifecycle
  issue: RepositoryFindingIssueAssociation
  possible_duplicates?: RepositoryFindingPossibleDuplicate[]
  validation_state: RepositoryFindingValidationState
  fix_commit_sha?: string
  fix_commit_time?: string
  first_containing_tag?: string
  resolution_history?: RepositoryFindingResolution[]
  version: number
  created_at: string
  updated_at: string
}

export interface RepositoryMappingModelSnapshot {
  profile_id?: string
  profile_version?: number
  prompt?: string
  model?: string
  account?: string
}

export interface RepositoryMappingAdjudication {
  decision: string
  candidate_id?: string
  confidence: number
  matching_anchors?: string[]
  conflicting_anchors?: string[]
  conflict_fields?: string[]
  explanation?: string
}

export interface RepositoryMappingJob {
  id: string
  review_finding_id: string
  state: RepositoryMappingJobState
  repository_finding_id?: string
  model_snapshot?: RepositoryMappingModelSnapshot
  adjudication?: RepositoryMappingAdjudication
  attempts: number
  error?: string
  reserved_at?: string
  created_at: string
  updated_at: string
}

export interface RepositoryValidationJob {
  id: string
  repository_finding_id: string
  state: RepositoryFindingValidationState
  model_snapshot?: RepositoryMappingModelSnapshot
  candidate_commits?: string[]
  attempts: number
  error?: string
  failure?: RepositoryValidationFailure
  reserved_at?: string
  created_at: string
  updated_at: string
}

export interface RepositoryFindingMutationResponse {
  automation?: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  repository_finding?: RepositoryFinding
  validation_jobs?: RepositoryValidationJob[]
}

export interface RepositoryReviewRunFindingStatusMutationResponse {
  automation?: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  findings: RepositoryReviewRunFindingStatusProjection[]
}

export interface RepositoryReviewRunFindingStatusProjection {
  id: string
  run_finding_status: RepositoryReviewRunFindingStatusState
}

export interface RepositoryReviewFindingsPage {
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  findings: RepositoryReviewFinding[]
  repository_findings: RepositoryFinding[]
  contexts?: RepositoryReviewFindingContext[]
  scope: RepositoryReviewFindingsScope
  offset: number
  total: number
  next_offset?: number
  repository_finding_total: number
  repository_finding_offset?: number
  next_repository_finding_offset?: number
  capabilities?: RepositoryReviewCapabilities
}

export interface RepositoryReviewRunFindingSummary {
  id: string
  repository: string
  path: string
  line?: number
  severity: string
  title: string
  symbol?: string
  status: RepositoryReviewFindingStatus
  run_finding_status: RepositoryReviewRunFindingStatusState
  association: "unassociated" | "new" | "existing" | "needs_review"
  repository_finding_id?: string
  contributors: string[]
  raw_source_count?: number
  created_at: string
  updated_at: string
}

export interface RepositoryReviewRepositoryFindingIssueSummary {
  url?: string
  state: RepositoryFindingIssueState
  snapshot_at?: string
  conflict?: boolean
}

export interface RepositoryReviewRepositoryFindingSummary {
  id: string
  repository: string
  canonical_title: string
  canonical_severity: string
  path?: string
  symbol?: string
  match_state: RepositoryReviewMatchState
  lifecycle: RepositoryFindingLifecycle
  issue: RepositoryReviewRepositoryFindingIssueSummary
  validation_state: RepositoryFindingValidationState
  occurrence_count: number
  found_commit_count: number
  created_at: string
  updated_at: string
}

export interface RepositoryReviewIssueSummary {
  id: string
  repository: string
  finding_count: number
  origin: RepositoryReviewIssueDraftOrigin
  generation_id?: string
  canonical: boolean
  publishable: boolean
  publish_blockers: RepositoryReviewPublishBlocker[]
  title: string
  state: RepositoryReviewIssueDraftState
  version: number
  created_at: string
  updated_at: string
}

export interface RepositoryReviewRunFindingsCollectionPage extends CollectionPageMetadata {
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  findings: RepositoryReviewRunFindingSummary[]
  findings_processing?: RepositoryReviewFindingsProcessingCounters
  historical_deduplication?: RepositoryReviewHistoricalDeduplication
  capabilities?: RepositoryReviewCapabilities
}

export interface RepositoryReviewRepositoryFindingsCollectionPage extends CollectionPageMetadata {
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  repository_findings: RepositoryReviewRepositoryFindingSummary[]
  capabilities?: RepositoryReviewCapabilities
}

export interface RepositoryReviewIssuesCollectionPage extends CollectionPageMetadata {
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  issues: RepositoryReviewIssueSummary[]
  generation_id?: string
  capabilities?: RepositoryReviewCapabilities
}

export type RepositoryReviewRepositoryFindingDetail =
  RepositoryReviewFindingDetail

/** @deprecated Use RepositoryReviewFindingsPage. */
export type RepositoryReviewReportPage = RepositoryReviewFindingsPage

export interface RepositoryReviewIssuePage {
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  issues: RepositoryReviewIssueDraft[]
  offset: number
  total: number
  next_offset?: number
  generation_id?: string
  capabilities?: RepositoryReviewCapabilities
}

export interface RepositoryReviewIssueDetail {
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  issue: RepositoryReviewIssueDraft
  finding?: RepositoryReviewFinding
  capabilities?: RepositoryReviewCapabilities
}

export interface RepositoryReviewMutationResult {
  id?: string
  draft_id?: string
  state?: RepositoryReviewIssueDraftState
  outcome?: "posted" | "failed" | "unknown" | "deleted" | "linked"
  success?: boolean
  code?: string
  message?: string
  external_url?: string
}

export interface RepositoryReviewIssueMutationResponse {
  automation?: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  issue?: RepositoryReviewIssueDraft
  draft?: RepositoryReviewIssueDraft
  finding?: RepositoryReviewFinding
  generation_id?: string
  issues?: RepositoryReviewIssueDraft[]
  results?: RepositoryReviewMutationResult[]
  result?: RepositoryReviewMutationResult
}

export interface RepositoryReviewIssueCandidate {
  id?: string
  number: number
  title: string
  url: string
  state?: "open" | "closed" | string
  labels?: string[]
  score?: number
  rank?: number
  explanation?: string
  matching_anchors?: string[]
  conflicting_anchors?: string[]
}

export interface RepositoryReviewIssueCandidatesResponse {
  automation?: RepositoryReviewAutomation
  finding?: RepositoryReviewFinding
  candidates: RepositoryReviewIssueCandidate[]
  generator_model?: string
  generator_account?: string
  discovered_issue?: RepositoryReviewIssueDraft
}

export interface RepositoryReviewSummary {
  schema_version: number
  id: string
  repository: string
  version: number
  review_version: number
  last_commit_sha?: string
  finding_count?: number
  repository_finding_count?: number
  open_finding_count?: number
  issue_draft_count?: number
  unsupported_count?: number
  reviewed_file_count?: number
  excluded_file_count?: number
  updated_at: string
}

export interface RepositoryReviewState extends RepositoryReviewSummary {
  files: Record<string, RepositoryReviewedFile>
  unsupported: Record<string, RepositoryUnsupportedFile>
  findings: RepositoryReviewFinding[]
  contexts: RepositoryReviewFindingContext[]
  runs: RepositoryReviewRun[]
  issue_drafts: RepositoryReviewIssueDraft[]
  repository_findings: RepositoryFinding[]
  mapping_jobs: RepositoryMappingJob[]
  validation_jobs: RepositoryValidationJob[]
  finding_offset?: number
  finding_total?: number
  next_finding_offset?: number
  draft_offset?: number
  draft_total?: number
  next_draft_offset?: number
}

export interface RepositoryReviewPage {
  repositories: RepositoryReviewSummary[]
}

export interface RepositoryReviewIssueDraftResult {
  repository: RepositoryReviewSummary
  draft: RepositoryReviewIssueDraft
  outcome?: "unknown"
}

export interface RepositoryReviewFindingResult {
  repository: RepositoryReviewSummary
  finding: RepositoryReviewFinding
}

export type RepositoryReviewAutomationStatus =
  | "idle"
  | "running"
  | "stopping"
  | "paused"
  | "completed"
  | "failed"

export type RepositoryReviewPauseReason =
  | "manual"
  | "token_budget"
  | "cost_budget"
  | "account_limit"
  | "guard_expression"
  | "no_progress"
  | "run_failed"
  | "service_restart"

export interface ReviewModelPrice {
  input_price_per_1m: number
  output_price_per_1m: number
}

export interface ReviewModelOption extends ReviewModelPrice {
  alias: string
  resolved_model: string
  provider: string
  available: boolean
  price_known: boolean
  blocked_reason?: string
  subscription?: boolean
  equivalent_model?: string
}

export interface ReviewAccountLimitEntry {
  window?: string
  label?: string
  name?: string
  remaining_percent?: number
  used_percent?: number
  reset_at?: string
  refreshes_at?: string
  refreshed_at?: string
  status?: string
}

export interface ReviewAccountOption {
  id: string
  provider?: string
  label: string
  status: string
  available?: boolean
  default?: boolean
  models?: string[]
  writer_models?: string[]
  entries: ReviewAccountLimitEntry[]
}

export interface RepositoryReviewAutomationBudget {
  guard_expression: string
}

export interface RepositoryReviewTokenUsage {
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  cached_tokens: number
}

export type RepositoryReviewFocusID =
  | "correctness_state"
  | "security_trust"
  | "concurrency_recovery"
  | "integration_validation"

export interface RepositoryReviewFileAttribution {
  id: string
  path: string
  commit_sha: string
  blob_sha: string
  focus_id: RepositoryReviewFocusID
  root_agent_id?: string
  reviewer_identity?: string
  account?: string
  model?: string
  source: "legacy" | "live" | "mixed"
  sources: string[]
  attempts: number
  run_ids: string[]
  run_count: number
  latest_completed_at: string
}

export interface RepositoryReviewAssignmentFocusProgress {
  total: number
  completed: number
  pending: number
  active: number
}

export interface RepositoryReviewAssignmentProgress extends RepositoryReviewAssignmentFocusProgress {
  by_focus: Record<
    RepositoryReviewFocusID,
    RepositoryReviewAssignmentFocusProgress
  >
}

export interface RepositoryReviewAutomationProgress {
  stage: string
  completed_batches: number
  total_batches: number
  coverage_available: boolean
  coverage_exact: boolean
  selected_files: number
  inspected_files: number
  reviewed_files: number
  remaining_files: number
  unsupported_files: number
  raw_findings?: number
  deduplicated_findings?: number
  /** @deprecated Use deduplicated_findings. */
  findings: number
  assignment_progress: RepositoryReviewAssignmentProgress
  scope_frozen?: boolean
  finding_aggregates: number
  unaggregated_findings: number
}

export type RepositoryReviewCodeType =
  | "hotpath-code"
  | "code"
  | "test"
  | "bench-test"

export interface RepositoryReviewScopePolicy {
  code_types: RepositoryReviewCodeType[]
  include_folders: string[]
  exclude_folders: string[]
  free_text: string
}

export interface RepositoryReviewScopePlanCounts {
  total_files: number
  code_type_files: number
  include_files: number
  excluded_files: number
  selected_files: number
}

export interface RepositoryReviewScopePlan {
  commit_sha: string
  policy_hash: string
  hash: string
  summary: string
  rationale?: string
  warnings: string[]
  counts: RepositoryReviewScopePlanCounts
}

export interface RepositoryReviewModelStats extends RepositoryReviewTokenUsage {
  model: string
  estimated_cost_usd: number
  requests: number
  failures: number
  reviewed_files: number
  findings: number
  latency_ms: number
}

export interface RepositoryReviewAccountSnapshot extends ReviewAccountOption {
  refreshed_at?: string
  error?: string
}

export interface RepositoryReviewAutomationConfig {
  name: string
  repository: string
  ref: string
  target: string
  account_ref: string
  effective_account_ref?: string
  review_focus: string
  scope_policy: RepositoryReviewScopePolicy
  reviewer_models: string[]
  issue_writer_model?: string
  compare_models: boolean
  force: boolean
  max_files_per_run: number
  max_content_bytes: number
  max_parallel_children: number
  assignment_timeout_seconds: number
  auto_continue: boolean
  model_prices: Record<string, ReviewModelPrice>
  budget: RepositoryReviewAutomationBudget
}

export interface RepositoryReviewProfileConfig {
  name: string
  account_ref: string
  review_focus: string
  issue_prompt: string
  scope_policy: RepositoryReviewScopePolicy
  reviewer_model: string
  deduplication_model?: string
  deduplication_similarity_threshold?: number
  deduplication_candidate_limit?: number
  issue_writer_model?: string
  force: boolean
  auto_continue: boolean
  max_files_per_run: number
  max_content_bytes: number
  max_parallel_children: number
  assignment_timeout_seconds: number
  budget: RepositoryReviewAutomationBudget
}

export interface RepositoryReviewProfile extends RepositoryReviewProfileConfig {
  schema_version?: number
  id: string
  version: number
  created_at: string
  updated_at: string
}

export interface RepositoryReviewRepositoryConfigInput {
  repository: string
  branch: string
  profile_id: string
}

export interface RepositoryReviewAutomation extends RepositoryReviewAutomationConfig {
  id: string
  version: number
  profile_id: string
  profile_version: number
  branch: string
  status: RepositoryReviewAutomationStatus
  pause_reason?: RepositoryReviewPauseReason
  pause_detail?: string
  active_run_id?: string
  run_ids: string[]
  usage: RepositoryReviewTokenUsage
  estimated_cost_usd: number
  progress: RepositoryReviewAutomationProgress
  model_stats: RepositoryReviewModelStats[]
  account_limits: RepositoryReviewAccountSnapshot[]
  scope_plan?: RepositoryReviewScopePlan
  resolved_commit_sha?: string
  resolved_target_branch?: string
  advertised_default_branch?: string
  target_is_default?: boolean
  started_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
}

export interface RepositoryReviewAutomationOptions {
  models: ReviewModelOption[]
  accounts: ReviewAccountOption[]
  limits_error?: string
}

export interface RepositoryReviewAutomationPage extends CollectionPageMetadata {
  automations: RepositoryReviewAutomation[]
}

export interface RepositoryReviewAutomationDetail {
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  capabilities: RepositoryReviewCapabilities
}

export interface RepositoryReviewHistoryMutationInput {
  expected_version: number
  expected_repository_version: number
  expected_ledger_fence: string
  confirm_repository: string
}

export interface RepositoryReviewPurgeHistoryResponse {
  automation: RepositoryReviewAutomation
  outcome: "history_purged"
}

export interface RepositoryReviewFileAttributionPage extends CollectionPageMetadata {
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  file_attributions: RepositoryReviewFileAttribution[]
}

export interface RepositoryReviewProfilePage extends CollectionPageMetadata {
  profiles: RepositoryReviewProfile[]
}

export interface RepositoryReviewCommitOption {
  sha: string
  short_sha: string
  url?: string
}

export interface RepositoryReviewCommitOptions {
  expected_version: number
  remembered: RepositoryReviewCommitOption
  latest: RepositoryReviewCommitOption
  newer_commit_available: boolean
}

export class RepositoryReviewAPIError extends Error {
  readonly status: number
  readonly code?: string

  constructor(status: number, message: string, code?: string) {
    super(message)
    this.name = "RepositoryReviewAPIError"
    this.status = status
    this.code = code
  }
}

const apiRoot = "/api/repository-reviews"

export async function listRepositoryReviews(
  signal?: AbortSignal,
): Promise<RepositoryReviewPage> {
  const page = await requestJSON<RepositoryReviewPage>(
    apiRoot,
    undefined,
    signal,
  )
  return {
    repositories: (page.repositories ?? []).map(
      normalizeRepositoryReviewSummary,
    ),
  }
}

export async function listRepositoryReviewAutomations(
  signal?: AbortSignal,
): Promise<{ automations: RepositoryReviewAutomation[] }> {
  const page = await requestJSON<{
    automations?: RepositoryReviewAutomation[]
  }>(`${apiRoot}/automations`, undefined, signal)
  return {
    automations: (page.automations ?? []).map(normalizeAutomation),
  }
}

export async function listRepositoryReviewAutomationsPage(
  input: CollectionListRequest = {},
  signal?: AbortSignal,
): Promise<RepositoryReviewAutomationPage> {
  const page = await requestJSON<Partial<RepositoryReviewAutomationPage>>(
    collectionListURL(`${apiRoot}/automations`, input),
    undefined,
    signal,
  )
  return {
    automations: (page.automations ?? []).map(normalizeAutomation),
    total: page.total ?? 0,
    next_cursor: page.next_cursor ?? "",
    canonical_query: page.canonical_query ?? "",
    query_schema: page.query_schema ?? { fields: [] },
  }
}

export async function listRepositoryReviewProfiles(
  signal?: AbortSignal,
): Promise<{ profiles: RepositoryReviewProfile[] }> {
  const page = await requestJSON<{ profiles?: RepositoryReviewProfile[] }>(
    `${apiRoot}/profiles`,
    undefined,
    signal,
  )
  return { profiles: (page.profiles ?? []).map(normalizeProfile) }
}

export async function listRepositoryReviewProfilesPage(
  input: CollectionListRequest = {},
  signal?: AbortSignal,
): Promise<RepositoryReviewProfilePage> {
  const page = await requestJSON<Partial<RepositoryReviewProfilePage>>(
    collectionListURL(`${apiRoot}/profiles`, input),
    undefined,
    signal,
  )
  return {
    profiles: (page.profiles ?? []).map(normalizeProfile),
    total: page.total ?? 0,
    next_cursor: page.next_cursor ?? "",
    canonical_query: page.canonical_query ?? "",
    query_schema: page.query_schema ?? { fields: [] },
  }
}

export async function getRepositoryReviewProfile(
  profileID: string,
  signal?: AbortSignal,
): Promise<RepositoryReviewProfile> {
  return profileFromMutation(
    await requestJSON<RepositoryReviewProfile | ProfileMutationResult>(
      profilePath(profileID),
      undefined,
      signal,
    ),
  )
}

export async function createRepositoryReviewProfile(
  input: RepositoryReviewProfileConfig,
  signal?: AbortSignal,
): Promise<RepositoryReviewProfile> {
  return profileFromMutation(
    await requestJSON<RepositoryReviewProfile | ProfileMutationResult>(
      `${apiRoot}/profiles`,
      jsonMutation("POST", repositoryReviewProfileConfigPayload(input)),
      signal,
    ),
  )
}

export async function updateRepositoryReviewProfile(
  profileID: string,
  input: RepositoryReviewProfileConfig & { expected_version: number },
  signal?: AbortSignal,
): Promise<RepositoryReviewProfile> {
  return profileFromMutation(
    await requestJSON<RepositoryReviewProfile | ProfileMutationResult>(
      profilePath(profileID),
      jsonMutation("PATCH", {
        ...repositoryReviewProfileConfigPayload(input),
        expected_version: input.expected_version,
      }),
      signal,
    ),
  )
}

export async function deleteRepositoryReviewProfile(
  profileID: string,
  input: { expected_version: number },
  signal?: AbortSignal,
): Promise<void> {
  await requestVoid(
    profilePath(profileID),
    jsonMutation("DELETE", input),
    signal,
  )
}

export async function getRepositoryReviewAutomationOptions(
  signal?: AbortSignal,
): Promise<RepositoryReviewAutomationOptions> {
  const options = await requestJSON<RepositoryReviewAutomationOptions>(
    `${apiRoot}/automation-options`,
    undefined,
    signal,
  )
  return {
    models: (options.models ?? []).map((model) => ({
      ...model,
      available: model.available ?? false,
      price_known: model.price_known ?? false,
      input_price_per_1m: model.input_price_per_1m ?? 0,
      output_price_per_1m: model.output_price_per_1m ?? 0,
    })),
    accounts: (options.accounts ?? []).map(normalizeAccount),
    ...(options.limits_error ? { limits_error: options.limits_error } : {}),
  }
}

export async function getRepositoryReviewCommitOptions(
  automationID: string,
  signal?: AbortSignal,
): Promise<RepositoryReviewCommitOptions> {
  return requestJSON<RepositoryReviewCommitOptions>(
    `${automationPath(automationID)}/commit-options`,
    undefined,
    signal,
  )
}

export async function createRepositoryReviewAutomation(
  input: RepositoryReviewRepositoryConfigInput,
  signal?: AbortSignal,
): Promise<RepositoryReviewAutomation> {
  return automationFromMutation(
    await requestJSON<RepositoryReviewAutomation | AutomationMutationResult>(
      `${apiRoot}/automations`,
      jsonMutation("POST", input),
      signal,
    ),
  )
}

export async function updateRepositoryReviewAutomation(
  automationID: string,
  input: RepositoryReviewRepositoryConfigInput & { expected_version: number },
  signal?: AbortSignal,
): Promise<RepositoryReviewAutomation> {
  return automationFromMutation(
    await requestJSON<RepositoryReviewAutomation | AutomationMutationResult>(
      automationPath(automationID),
      jsonMutation("PATCH", input),
      signal,
    ),
  )
}

export async function deleteRepositoryReviewAutomation(
  automationID: string,
  input: RepositoryReviewHistoryMutationInput,
  signal?: AbortSignal,
): Promise<void> {
  await requestVoid(
    automationPath(automationID),
    jsonMutation("DELETE", input),
    signal,
  )
}

export async function purgeRepositoryReviewAutomationHistory(
  automationID: string,
  input: RepositoryReviewHistoryMutationInput,
  signal?: AbortSignal,
): Promise<RepositoryReviewPurgeHistoryResponse> {
  const value = await requestJSON<RepositoryReviewPurgeHistoryResponse>(
    `${automationPath(automationID)}/purge-history`,
    jsonMutation("POST", input),
    signal,
  )
  return {
    automation: normalizeAutomation(value.automation),
    outcome: value.outcome,
  }
}

export async function startRepositoryReviewAutomation(
  automationID: string,
  input: { expected_version: number },
  signal?: AbortSignal,
): Promise<RepositoryReviewAutomation> {
  return mutateAutomationAction(automationID, "start", input, signal)
}

export async function pauseRepositoryReviewAutomation(
  automationID: string,
  input: { expected_version: number; run_id?: string },
  signal?: AbortSignal,
): Promise<RepositoryReviewAutomation> {
  return mutateAutomationAction(automationID, "pause", input, signal)
}

export async function resumeRepositoryReviewAutomation(
  automationID: string,
  input: { expected_version: number; commit_sha?: string },
  signal?: AbortSignal,
): Promise<RepositoryReviewAutomation> {
  return mutateAutomationAction(automationID, "resume", input, signal)
}

export async function restartRepositoryReviewAutomation(
  automationID: string,
  input: { expected_version: number },
  signal?: AbortSignal,
): Promise<RepositoryReviewAutomation> {
  return mutateAutomationAction(automationID, "restart", input, signal)
}

export async function getRepositoryReviewAutomation(
  automationID: string,
  signal?: AbortSignal,
): Promise<RepositoryReviewAutomation> {
  const value = await requestJSON<
    RepositoryReviewAutomation | AutomationMutationResult
  >(automationPath(automationID), undefined, signal)
  return automationFromMutation(value)
}

export async function getRepositoryReviewAutomationDetail(
  automationID: string,
  signal?: AbortSignal,
): Promise<RepositoryReviewAutomationDetail> {
  const value = await requestJSON<RepositoryReviewAutomationDetail>(
    automationPath(automationID),
    undefined,
    signal,
  )
  return {
    automation: normalizeAutomation(value.automation),
    repository: value.repository
      ? normalizeRepositoryReviewSummary(value.repository)
      : undefined,
    capabilities: normalizeCapabilities(value.capabilities) ?? {},
  }
}

export async function listRepositoryReviewAutomationFileAttributionsPage(
  automationID: string,
  input: CollectionListRequest = {},
  signal?: AbortSignal,
): Promise<RepositoryReviewFileAttributionPage> {
  const collectionInput = {
    ...input,
    query:
      input.query?.trim() || "ALL ORDER BY path ASC, focus ASC, reviewer ASC",
  }
  const page = await collectionRequest<
    Partial<RepositoryReviewFileAttributionPage>
  >(
    collectionListURL(
      `${automationPath(automationID)}/file-attributions`,
      collectionInput,
    ),
    undefined,
    signal,
  )
  return {
    automation: normalizeAutomation(page.automation!),
    repository: page.repository
      ? normalizeRepositoryReviewSummary(page.repository)
      : undefined,
    file_attributions: (page.file_attributions ?? []).map(
      normalizeFileAttribution,
    ),
    total: page.total ?? 0,
    next_cursor: page.next_cursor ?? "",
    canonical_query: page.canonical_query ?? "",
    query_schema: page.query_schema ?? { fields: [] },
  }
}

export async function getRepositoryReviewAutomationFindings(
  automationID: string,
  input: {
    scope?: RepositoryReviewFindingsScope
    offset?: number
    limit?: number
  } = {},
  signal?: AbortSignal,
): Promise<RepositoryReviewFindingsPage> {
  const params = new URLSearchParams()
  params.set("scope", input.scope ?? "current")
  if (input.offset) params.set("offset", String(input.offset))
  if (input.limit) params.set("limit", String(input.limit))
  const value = await requestJSON<
    Partial<RepositoryReviewFindingsPage> & {
      finding_total?: number
      next_finding_offset?: number
    }
  >(
    `${automationPath(automationID)}/report?${params.toString()}`,
    undefined,
    signal,
  )
  return {
    ...value,
    automation: normalizeAutomation(value.automation!),
    repository: value.repository
      ? normalizeRepositoryReviewSummary(value.repository)
      : undefined,
    findings: (value.findings ?? []).map(normalizeFinding),
    repository_findings: (value.repository_findings ?? []).map(
      normalizeRepositoryFinding,
    ),
    contexts: (value.contexts ?? []).map(normalizeFindingContext),
    scope: value.scope === "all" ? "all" : "current",
    offset: value.offset ?? input.offset ?? 0,
    total: value.total ?? value.finding_total ?? value.findings?.length ?? 0,
    next_offset: value.next_offset ?? value.next_finding_offset,
    repository_finding_total:
      value.repository_finding_total ?? value.repository_findings?.length ?? 0,
    repository_finding_offset: value.repository_finding_offset ?? 0,
    next_repository_finding_offset: value.next_repository_finding_offset,
    capabilities: normalizeCapabilities(value.capabilities),
  }
}

export async function listRepositoryReviewAutomationFindingsPage(
  automationID: string,
  input: CollectionListRequest = {},
  signal?: AbortSignal,
): Promise<RepositoryReviewRunFindingsCollectionPage> {
  const collectionInput = {
    ...input,
    query: input.query?.trim() || "ALL ORDER BY severity DESC, updated DESC",
  }
  const page = await collectionRequest<
    Partial<RepositoryReviewRunFindingsCollectionPage>
  >(
    collectionListURL(
      `${automationPath(automationID)}/findings`,
      collectionInput,
    ),
    undefined,
    signal,
  )
  return {
    automation: normalizeAutomation(page.automation!),
    repository: page.repository
      ? normalizeRepositoryReviewSummary(page.repository)
      : undefined,
    findings: (page.findings ?? []).map(normalizeRunFindingSummary),
    total: page.total ?? 0,
    next_cursor: page.next_cursor ?? "",
    canonical_query: page.canonical_query ?? "",
    query_schema: page.query_schema ?? { fields: [] },
    findings_processing: page.findings_processing
      ? normalizeFindingsProcessingCounters(page.findings_processing)
      : undefined,
    historical_deduplication: page.historical_deduplication
      ? normalizeHistoricalDeduplication(page.historical_deduplication)
      : undefined,
    capabilities: normalizeCapabilities(page.capabilities),
  }
}

export async function listRepositoryReviewAutomationRawFindingsPage(
  automationID: string,
  input: CollectionListRequest = {},
  signal?: AbortSignal,
): Promise<RepositoryReviewRawFindingsCollectionPage> {
  const collectionInput = {
    ...input,
    query: input.query?.trim() || "ALL ORDER BY created DESC",
  }
  const page = await collectionRequest<
    Partial<RepositoryReviewRawFindingsCollectionPage>
  >(
    collectionListURL(
      `${automationPath(automationID)}/raw-findings`,
      collectionInput,
    ),
    undefined,
    signal,
  )
  return {
    automation: normalizeAutomation(page.automation!),
    repository: page.repository
      ? normalizeRepositoryReviewSummary(page.repository)
      : undefined,
    raw_findings: (page.raw_findings ?? []).map(normalizeRawFinding),
    total: page.total ?? 0,
    next_cursor: page.next_cursor ?? "",
    canonical_query: page.canonical_query ?? "",
    query_schema: page.query_schema ?? { fields: [] },
    findings_processing: page.findings_processing
      ? normalizeFindingsProcessingCounters(page.findings_processing)
      : undefined,
    historical_deduplication: page.historical_deduplication
      ? normalizeHistoricalDeduplication(page.historical_deduplication)
      : undefined,
  }
}

export async function getRepositoryReviewFindingHealth(
  automationID: string,
  signal?: AbortSignal,
): Promise<RepositoryReviewFindingHealth> {
  const value = await requestJSON<Partial<RepositoryReviewFindingHealth>>(
    `${automationPath(automationID)}/finding-health`,
    undefined,
    signal,
  )
  return normalizeFindingHealth(value)
}

export async function listRepositoryReviewFindingsProcessingPage(
  automationID: string,
  input: CollectionListRequest = {},
  signal?: AbortSignal,
): Promise<RepositoryReviewFindingsProcessingCollectionPage> {
  const collectionInput = {
    ...input,
    query: input.query?.trim() || "ALL ORDER BY updated DESC",
  }
  const page = await collectionRequest<
    Partial<RepositoryReviewFindingsProcessingCollectionPage> & {
      raw_findings?: RepositoryReviewRawFinding[]
    }
  >(
    collectionListURL(
      `${automationPath(automationID)}/findings-processing`,
      collectionInput,
    ),
    undefined,
    signal,
  )
  return {
    automation: normalizeAutomation(page.automation!),
    repository: page.repository
      ? normalizeRepositoryReviewSummary(page.repository)
      : undefined,
    sources: (page.sources ?? page.raw_findings ?? []).map(normalizeRawFinding),
    total: page.total ?? 0,
    next_cursor: page.next_cursor ?? "",
    canonical_query: page.canonical_query ?? "",
    query_schema: page.query_schema ?? { fields: [] },
    findings_processing: normalizeFindingHealthProcessing(
      page.findings_processing,
    ),
    historical_consolidation: page.historical_consolidation
      ? normalizeHistoricalConsolidation(page.historical_consolidation)
      : undefined,
    capabilities: normalizeCapabilities(page.capabilities),
  }
}

export async function listRepositoryReviewAutomationRepositoryFindingsPage(
  automationID: string,
  input: CollectionListRequest = {},
  signal?: AbortSignal,
): Promise<RepositoryReviewRepositoryFindingsCollectionPage> {
  const collectionInput = {
    ...input,
    query: input.query?.trim() || "ALL ORDER BY severity DESC, updated DESC",
  }
  const page = await collectionRequest<
    Partial<RepositoryReviewRepositoryFindingsCollectionPage>
  >(
    collectionListURL(
      `${automationPath(automationID)}/repository-findings`,
      collectionInput,
    ),
    undefined,
    signal,
  )
  return {
    automation: normalizeAutomation(page.automation!),
    repository: page.repository
      ? normalizeRepositoryReviewSummary(page.repository)
      : undefined,
    repository_findings: (page.repository_findings ?? []).map(
      normalizeRepositoryFindingSummary,
    ),
    total: page.total ?? 0,
    next_cursor: page.next_cursor ?? "",
    canonical_query: page.canonical_query ?? "",
    query_schema: page.query_schema ?? { fields: [] },
    capabilities: normalizeCapabilities(page.capabilities),
  }
}

export async function getRepositoryReviewAutomationRepositoryFinding(
  automationID: string,
  repositoryFindingID: string,
  signal?: AbortSignal,
): Promise<RepositoryReviewRepositoryFindingDetail> {
  const value = await requestJSON<
    Partial<RepositoryReviewFindingDetail> & {
      draft?: RepositoryReviewIssueDraft
    }
  >(
    `${automationPath(automationID)}/repository-findings/${encodeURIComponent(repositoryFindingID)}`,
    undefined,
    signal,
  )
  return normalizeFindingDetail(value)
}

/** @deprecated Use getRepositoryReviewAutomationFindings. */
export async function getRepositoryReviewAutomationReport(
  automationID: string,
  input: {
    scope?: RepositoryReviewReportScope
    offset?: number
    limit?: number
  } = {},
  signal?: AbortSignal,
): Promise<RepositoryReviewFindingsPage> {
  return getRepositoryReviewAutomationFindings(automationID, input, signal)
}

export async function getRepositoryReviewAutomationFinding(
  automationID: string,
  findingID: string,
  signal?: AbortSignal,
): Promise<RepositoryReviewFindingDetail> {
  const value = await requestJSON<
    Partial<RepositoryReviewFindingDetail> & {
      draft?: RepositoryReviewIssueDraft
    }
  >(automationFindingPath(automationID, findingID), undefined, signal)
  return normalizeFindingDetail(value)
}

/** @deprecated Use only to resolve legacy rfn_* occurrence bookmarks. */
export async function getRepositoryReviewAutomationRunFinding(
  automationID: string,
  findingID: string,
  signal?: AbortSignal,
): Promise<RepositoryReviewFindingDetail> {
  const value = await requestJSON<
    Partial<RepositoryReviewFindingDetail> & {
      draft?: RepositoryReviewIssueDraft
    }
  >(
    `${automationPath(automationID)}/run-findings/${encodeURIComponent(findingID)}`,
    undefined,
    signal,
  )
  return normalizeFindingDetail(value)
}

export async function listRepositoryReviewFindingRawSources(
  automationID: string,
  findingID: string,
  input: { offset?: number; limit?: number } = {},
  signal?: AbortSignal,
): Promise<RepositoryReviewRawFindingsPage> {
  const params = new URLSearchParams()
  if (input.offset) params.set("offset", String(input.offset))
  if (input.limit) params.set("limit", String(input.limit))
  const value = await requestJSON<RepositoryReviewRawFindingsPage>(
    `${automationFindingPath(automationID, findingID)}/sources?${params.toString()}`,
    undefined,
    signal,
  )
  return {
    ...value,
    automation: normalizeAutomation(value.automation),
    repository: value.repository
      ? normalizeRepositoryReviewSummary(value.repository)
      : undefined,
    sources: (value.sources ?? []).map(normalizeRawFinding),
    offset: value.offset ?? 0,
    total: value.total ?? value.sources?.length ?? 0,
  }
}

export async function getRepositoryReviewRawSource(
  automationID: string,
  sourceID: string,
  signal?: AbortSignal,
): Promise<RepositoryReviewRawFindingDetail> {
  const value = await requestJSON<RepositoryReviewRawFindingDetail>(
    `${automationPath(automationID)}/raw-findings/${encodeURIComponent(sourceID)}`,
    undefined,
    signal,
  )
  return {
    ...value,
    automation: normalizeAutomation(value.automation),
    repository: value.repository
      ? normalizeRepositoryReviewSummary(value.repository)
      : undefined,
    source: normalizeRawFinding(value.source),
    finding: value.finding ? normalizeFinding(value.finding) : undefined,
    context: value.context ? normalizeFindingContext(value.context) : undefined,
    findings_processing: value.findings_processing
      ? normalizeFindingsProcessingCounters(value.findings_processing)
      : undefined,
    historical_deduplication: value.historical_deduplication
      ? normalizeHistoricalDeduplication(value.historical_deduplication)
      : undefined,
  }
}

export async function getRepositoryReviewFindingsProcessingSource(
  automationID: string,
  sourceID: string,
  signal?: AbortSignal,
): Promise<RepositoryReviewFindingsProcessingDetail> {
  const value = await requestJSON<RepositoryReviewFindingsProcessingDetail>(
    `${automationPath(automationID)}/findings-processing/sources/${encodeURIComponent(sourceID)}`,
    undefined,
    signal,
  )
  return normalizeFindingsProcessingDetail(value)
}

export async function getRepositoryReviewFindingsProcessing(
  automationID: string,
  input: {
    offset?: number
    limit?: number
    state?: RepositoryReviewDeduplicationState
  } = {},
  signal?: AbortSignal,
): Promise<RepositoryReviewRawFindingsPage> {
  const params = new URLSearchParams()
  if (input.offset) params.set("offset", String(input.offset))
  if (input.limit) params.set("limit", String(input.limit))
  if (input.state) params.set("state", input.state)
  const value = await requestJSON<RepositoryReviewRawFindingsPage>(
    `${automationPath(automationID)}/findings-processing?${params.toString()}`,
    undefined,
    signal,
  )
  return {
    ...value,
    automation: normalizeAutomation(value.automation),
    repository: value.repository
      ? normalizeRepositoryReviewSummary(value.repository)
      : undefined,
    raw_findings: (value.raw_findings ?? []).map(normalizeRawFinding),
    findings_processing: value.findings_processing
      ? normalizeFindingsProcessingCounters(value.findings_processing)
      : undefined,
    offset: value.offset ?? 0,
    total: value.total ?? value.raw_findings?.length ?? 0,
  }
}

export async function retryRepositoryReviewRawSource(
  automationID: string,
  sourceID: string,
  signal?: AbortSignal,
): Promise<RepositoryReviewRawFindingDetail> {
  const value = await requestJSON<RepositoryReviewRawFindingDetail>(
    `${automationPath(automationID)}/raw-findings/${encodeURIComponent(sourceID)}/retry`,
    jsonMutation("POST", {}),
    signal,
  )
  return {
    ...value,
    automation: normalizeAutomation(value.automation),
    repository: value.repository
      ? normalizeRepositoryReviewSummary(value.repository)
      : undefined,
    source: normalizeRawFinding(value.source),
    context: value.context ? normalizeFindingContext(value.context) : undefined,
    finding: value.finding ? normalizeFinding(value.finding) : undefined,
    findings_processing: value.findings_processing
      ? normalizeFindingsProcessingCounters(value.findings_processing)
      : undefined,
  }
}

export async function retryRepositoryReviewFindingsProcessingSource(
  automationID: string,
  sourceID: string,
  signal?: AbortSignal,
): Promise<RepositoryReviewFindingsProcessingDetail> {
  const value = await requestJSON<RepositoryReviewFindingsProcessingDetail>(
    `${automationPath(automationID)}/findings-processing/sources/${encodeURIComponent(sourceID)}/retry`,
    jsonMutation("POST", {}),
    signal,
  )
  return normalizeFindingsProcessingDetail(value)
}

export async function retryRepositoryReviewFindingsProcessingSources(
  automationID: string,
  sourceIDs: string[],
  signal?: AbortSignal,
): Promise<RepositoryReviewFindingsProcessingRetryResponse> {
  const value =
    await requestJSON<RepositoryReviewFindingsProcessingRetryResponse>(
      `${automationPath(automationID)}/findings-processing/retry`,
      jsonMutation("POST", { source_ids: sourceIDs }),
      signal,
    )
  return {
    retried_ids: value.retried_ids ?? [],
    failures: (value.failures ?? []).map((failure) => ({
      source_id: failure.source_id ?? "",
      code: failure.code ?? "retry_failed",
      message: failure.message ?? "This finding could not be retried.",
    })),
    findings_processing: normalizeFindingHealthProcessing(
      value.findings_processing,
    ),
    health: normalizeFindingHealth(value.health),
  }
}

export async function retryRepositoryReviewHistoricalDeduplication(
  automationID: string,
  signal?: AbortSignal,
): Promise<{
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  historical_deduplication: RepositoryReviewHistoricalDeduplication
}> {
  const value = await requestJSON<{
    automation: RepositoryReviewAutomation
    repository?: RepositoryReviewSummary
    historical_deduplication: RepositoryReviewHistoricalDeduplication
  }>(
    `${automationPath(automationID)}/historical-deduplication/retry`,
    jsonMutation("POST", {}),
    signal,
  )
  return {
    automation: normalizeAutomation(value.automation),
    repository: value.repository
      ? normalizeRepositoryReviewSummary(value.repository)
      : undefined,
    historical_deduplication: normalizeHistoricalDeduplication(
      value.historical_deduplication,
    ),
  }
}

export async function restartRepositoryReviewHistoricalDeduplication(
  automationID: string,
  signal?: AbortSignal,
): Promise<{
  automation: RepositoryReviewAutomation
  repository?: RepositoryReviewSummary
  historical_deduplication: RepositoryReviewHistoricalDeduplication
}> {
  const value = await requestJSON<{
    automation: RepositoryReviewAutomation
    repository?: RepositoryReviewSummary
    historical_deduplication: RepositoryReviewHistoricalDeduplication
  }>(
    `${automationPath(automationID)}/historical-deduplication/restart`,
    jsonMutation("POST", { confirmed: true }),
    signal,
  )
  return {
    automation: normalizeAutomation(value.automation),
    repository: value.repository
      ? normalizeRepositoryReviewSummary(value.repository)
      : undefined,
    historical_deduplication: normalizeHistoricalDeduplication(
      value.historical_deduplication,
    ),
  }
}

export async function retryRepositoryReviewRunFindingStatuses(
  automationID: string,
  findingIDs: string[],
  signal?: AbortSignal,
): Promise<RepositoryReviewRunFindingStatusMutationResponse> {
  const value = await requestJSON<
    Partial<RepositoryReviewRunFindingStatusMutationResponse>
  >(
    `${automationPath(automationID)}/findings/status`,
    jsonMutation("POST", { finding_ids: findingIDs }),
    signal,
  )
  return {
    automation: value.automation
      ? normalizeAutomation(value.automation)
      : undefined,
    repository: value.repository
      ? normalizeRepositoryReviewSummary(value.repository)
      : undefined,
    findings: (value.findings ?? []).map((finding) => ({
      id: finding.id,
      run_finding_status:
        normalizeRunFindingStatus(finding.run_finding_status) ?? "pending",
    })),
  }
}

export async function resolveRepositoryReviewPossibleDuplicate(
  automationID: string,
  repositoryFindingID: string,
  input: {
    candidate_id: string
    decision: "merge" | "distinct"
    expected_provisional_version: number
    expected_candidate_version?: number
  },
  signal?: AbortSignal,
): Promise<RepositoryFindingMutationResponse> {
  const value = await requestJSON<RepositoryFindingMutationResponse>(
    `${automationPath(automationID)}/repository-findings/${encodeURIComponent(repositoryFindingID)}/duplicates`,
    jsonMutation("POST", input),
    signal,
  )
  return normalizeRepositoryFindingMutation(value)
}

export async function updateRepositoryReviewFindingLifecycle(
  automationID: string,
  repositoryFindingID: string,
  input: {
    lifecycle: "open" | "dismissed"
    expected_version: number
  },
  signal?: AbortSignal,
): Promise<RepositoryFindingMutationResponse> {
  const value = await requestJSON<RepositoryFindingMutationResponse>(
    `${automationPath(automationID)}/repository-findings/${encodeURIComponent(repositoryFindingID)}`,
    jsonMutation("PATCH", input),
    signal,
  )
  return normalizeRepositoryFindingMutation(value)
}

export async function postRepositoryReviewFinding(
  automationID: string,
  findingID: string,
  input: { expected_version: number; instructions?: string },
  signal?: AbortSignal,
): Promise<RepositoryReviewIssueMutationResponse> {
  const value = await requestJSON<RepositoryReviewIssueMutationResponse>(
    `${automationFindingPath(automationID, findingID)}/post`,
    jsonMutation("POST", input),
    signal,
  )
  return normalizeIssueMutationResponse(value)
}

export async function reserveRepositoryReviewValidations(
  automationID: string,
  repositoryFindingIDs: string[],
  signal?: AbortSignal,
): Promise<RepositoryFindingMutationResponse> {
  const value = await requestJSON<RepositoryFindingMutationResponse>(
    `${automationPath(automationID)}/repository-findings/validations`,
    jsonMutation("POST", { repository_finding_ids: repositoryFindingIDs }),
    signal,
  )
  return normalizeRepositoryFindingMutation(value)
}

export async function syncRepositoryReviewFinding(
  automationID: string,
  repositoryFindingID: string,
  signal?: AbortSignal,
): Promise<RepositoryFindingMutationResponse> {
  const value = await requestJSON<RepositoryFindingMutationResponse>(
    `${automationPath(automationID)}/repository-findings/${encodeURIComponent(repositoryFindingID)}/sync`,
    jsonMutation("POST", {}),
    signal,
  )
  return normalizeRepositoryFindingMutation(value)
}

export async function updateRepositoryReviewAutomationFinding(
  automationID: string,
  findingID: string,
  input: {
    status: Exclude<RepositoryReviewFindingStatus, "posted">
    expected_version: number
  },
  signal?: AbortSignal,
): Promise<RepositoryReviewFindingDetail> {
  const value = await requestJSON<
    Partial<RepositoryReviewFindingDetail> & {
      draft?: RepositoryReviewIssueDraft
    }
  >(
    automationFindingPath(automationID, findingID),
    jsonMutation("PATCH", input),
    signal,
  )
  return normalizeFindingDetail(value)
}

export async function listRepositoryReviewAutomationIssues(
  automationID: string,
  input: { generation_id?: string; offset?: number; limit?: number } = {},
  signal?: AbortSignal,
): Promise<RepositoryReviewIssuePage> {
  const params = new URLSearchParams()
  if (input.generation_id) params.set("generation_id", input.generation_id)
  if (input.offset) params.set("offset", String(input.offset))
  if (input.limit) params.set("limit", String(input.limit))
  const query = params.size > 0 ? `?${params.toString()}` : ""
  const value = await requestJSON<
    Partial<RepositoryReviewIssuePage> & {
      drafts?: RepositoryReviewIssueDraft[]
      issue_total?: number
      next_issue_offset?: number
    }
  >(`${automationPath(automationID)}/issues${query}`, undefined, signal)
  return {
    ...value,
    automation: normalizeAutomation(value.automation!),
    repository: value.repository
      ? normalizeRepositoryReviewSummary(value.repository)
      : undefined,
    issues: (value.issues ?? value.drafts ?? []).map(normalizeIssueDraft),
    offset: value.offset ?? input.offset ?? 0,
    total:
      value.total ??
      value.issue_total ??
      value.issues?.length ??
      value.drafts?.length ??
      0,
    next_offset: value.next_offset ?? value.next_issue_offset,
    generation_id: value.generation_id ?? input.generation_id,
    capabilities: normalizeCapabilities(value.capabilities),
  }
}

export async function listRepositoryReviewAutomationIssuesPage(
  automationID: string,
  input: CollectionListRequest & { generation_id?: string } = {},
  signal?: AbortSignal,
): Promise<RepositoryReviewIssuesCollectionPage> {
  const collectionInput = {
    ...input,
    query: input.query?.trim() || "ALL ORDER BY updated DESC",
  }
  let path = collectionListURL(
    `${automationPath(automationID)}/issues`,
    collectionInput,
  )
  if (input.generation_id) {
    path += `${path.includes("?") ? "&" : "?"}generation_id=${encodeURIComponent(input.generation_id)}`
  }
  const page = await collectionRequest<
    Partial<RepositoryReviewIssuesCollectionPage>
  >(path, undefined, signal)
  return {
    automation: normalizeAutomation(page.automation!),
    repository: page.repository
      ? normalizeRepositoryReviewSummary(page.repository)
      : undefined,
    issues: (page.issues ?? []).map(normalizeIssueSummary),
    total: page.total ?? 0,
    next_cursor: page.next_cursor ?? "",
    canonical_query: page.canonical_query ?? "",
    query_schema: page.query_schema ?? { fields: [] },
    generation_id: page.generation_id ?? input.generation_id,
    capabilities: normalizeCapabilities(page.capabilities),
  }
}

export async function generateRepositoryReviewIssues(
  automationID: string,
  input: {
    generation_id: string
    finding_ids: string[]
    instructions_mode: RepositoryReviewIssueInstructionsMode
    instructions?: string
  },
  signal?: AbortSignal,
): Promise<RepositoryReviewIssueMutationResponse> {
  return normalizeIssueMutationResponse(
    await requestJSON<RepositoryReviewIssueMutationResponse>(
      `${automationPath(automationID)}/issues/generations`,
      jsonMutation("POST", input),
      signal,
    ),
  )
}

export async function publishRepositoryReviewIssues(
  automationID: string,
  input: {
    issues: Array<{ id: string; expected_version: number }>
    confirmed: true
  },
  signal?: AbortSignal,
): Promise<RepositoryReviewIssueMutationResponse> {
  return normalizeIssueMutationResponse(
    await requestJSON<RepositoryReviewIssueMutationResponse>(
      `${automationPath(automationID)}/issues/publish`,
      jsonMutation("POST", input),
      signal,
    ),
  )
}

export async function getRepositoryReviewAutomationIssue(
  automationID: string,
  draftID: string,
  signal?: AbortSignal,
): Promise<RepositoryReviewIssueDetail> {
  const value = await requestJSON<
    Partial<RepositoryReviewIssueDetail> & {
      draft?: RepositoryReviewIssueDraft
    }
  >(automationIssuePath(automationID, draftID), undefined, signal)
  return normalizeIssueDetail(value)
}

export async function updateRepositoryReviewAutomationIssue(
  automationID: string,
  draftID: string,
  input: {
    title: string
    body: string
    labels: string[]
    expected_version: number
  },
  signal?: AbortSignal,
): Promise<RepositoryReviewIssueDetail> {
  const value = await requestJSON<
    Partial<RepositoryReviewIssueDetail> & {
      draft?: RepositoryReviewIssueDraft
    }
  >(
    automationIssuePath(automationID, draftID),
    jsonMutation("PATCH", input),
    signal,
  )
  return normalizeIssueDetail(value)
}

export async function deleteRepositoryReviewAutomationIssue(
  automationID: string,
  draftID: string,
  input: { expected_version: number; confirmed: true },
  signal?: AbortSignal,
): Promise<RepositoryReviewIssueMutationResponse> {
  return normalizeIssueMutationResponse(
    await requestJSON<RepositoryReviewIssueMutationResponse>(
      automationIssuePath(automationID, draftID),
      jsonMutation("DELETE", input),
      signal,
    ),
  )
}

export async function regenerateRepositoryReviewAutomationIssue(
  automationID: string,
  draftID: string,
  input: { expected_version: number },
  signal?: AbortSignal,
): Promise<RepositoryReviewIssueDetail> {
  const value = await requestJSON<
    Partial<RepositoryReviewIssueDetail> & {
      draft?: RepositoryReviewIssueDraft
    }
  >(
    `${automationIssuePath(automationID, draftID)}/regenerate`,
    jsonMutation("POST", input),
    signal,
  )
  return normalizeIssueDetail(value)
}

export async function publishRepositoryReviewAutomationIssue(
  automationID: string,
  draftID: string,
  input: { expected_version: number; confirmed: true },
  signal?: AbortSignal,
): Promise<RepositoryReviewIssueDetail> {
  const value = await requestJSON<
    Partial<RepositoryReviewIssueDetail> & {
      draft?: RepositoryReviewIssueDraft
    }
  >(
    `${automationIssuePath(automationID, draftID)}/publish`,
    jsonMutation("POST", input),
    signal,
  )
  return normalizeIssueDetail(value)
}

export async function findRepositoryReviewIssueCandidates(
  automationID: string,
  findingID: string,
  input: { expected_version: number },
  signal?: AbortSignal,
): Promise<RepositoryReviewIssueCandidatesResponse> {
  const value = await requestJSON<RepositoryReviewIssueCandidatesResponse>(
    `${automationFindingPath(automationID, findingID)}/issue-link/candidates`,
    jsonMutation("POST", input),
    signal,
  )
  return {
    ...value,
    automation: value.automation
      ? normalizeAutomation(value.automation)
      : undefined,
    finding: value.finding ? normalizeFinding(value.finding) : undefined,
    candidates: (value.candidates ?? []).map((candidate) => ({
      ...candidate,
      labels: candidate.labels ?? [],
      matching_anchors: candidate.matching_anchors ?? [],
      conflicting_anchors: candidate.conflicting_anchors ?? [],
    })),
    discovered_issue: value.discovered_issue
      ? normalizeIssueDraft(value.discovered_issue)
      : undefined,
  }
}

export async function linkRepositoryReviewIssue(
  automationID: string,
  findingID: string,
  input: {
    issue_url: string
    expected_version: number
    confirmed: true
    replace?: boolean
  },
  signal?: AbortSignal,
): Promise<RepositoryReviewFindingDetail> {
  const value = await requestJSON<
    Partial<RepositoryReviewFindingDetail> & {
      draft?: RepositoryReviewIssueDraft
    }
  >(
    `${automationFindingPath(automationID, findingID)}/issue-link`,
    jsonMutation("POST", input),
    signal,
  )
  return normalizeFindingDetail(value)
}

export async function unlinkRepositoryReviewIssue(
  automationID: string,
  findingID: string,
  input: { expected_version: number; confirmed: true },
  signal?: AbortSignal,
): Promise<RepositoryReviewFindingDetail> {
  const value = await requestJSON<
    Partial<RepositoryReviewFindingDetail> & {
      draft?: RepositoryReviewIssueDraft
    }
  >(
    `${automationFindingPath(automationID, findingID)}/issue-link`,
    jsonMutation("DELETE", input),
    signal,
  )
  return normalizeFindingDetail(value)
}

export async function getRepositoryReview(
  repositoryID: string,
  signal?: AbortSignal,
  options?: {
    offset?: number
    limit?: number
    draftOffset?: number
    draftLimit?: number
  },
): Promise<RepositoryReviewState> {
  const params = new URLSearchParams()
  if (options?.offset) params.set("offset", String(options.offset))
  if (options?.limit) params.set("limit", String(options.limit))
  if (options?.draftOffset)
    params.set("draft_offset", String(options.draftOffset))
  if (options?.draftLimit) params.set("draft_limit", String(options.draftLimit))
  const query = params.size > 0 ? `?${params.toString()}` : ""
  return normalizeRepositoryReviewState(
    await requestJSON<RepositoryReviewState>(
      repositoryPath(repositoryID) + query,
      undefined,
      signal,
    ),
  )
}

export async function updateRepositoryReviewFinding(
  repositoryID: string,
  findingID: string,
  input: {
    status: RepositoryReviewFindingStatus
    expected_version: number
  },
  signal?: AbortSignal,
): Promise<RepositoryReviewFindingResult> {
  const result = await requestJSON<RepositoryReviewFindingResult>(
    `${repositoryPath(repositoryID)}/findings/${encodeURIComponent(findingID)}`,
    jsonMutation("PATCH", input),
    signal,
  )
  return {
    repository: normalizeRepositoryReviewSummary(result.repository),
    finding: normalizeFinding(result.finding),
  }
}

export async function createRepositoryReviewIssueDraft(
  repositoryID: string,
  input: {
    finding_ids: string[]
    title?: string
    body?: string
    labels?: string[]
    expected_version: number
  },
  signal?: AbortSignal,
): Promise<RepositoryReviewIssueDraftResult> {
  return normalizeIssueDraftResult(
    await requestJSON<RepositoryReviewIssueDraftResult>(
      `${repositoryPath(repositoryID)}/issue-drafts`,
      jsonMutation("POST", input),
      signal,
    ),
  )
}

export async function updateRepositoryReviewIssueDraft(
  repositoryID: string,
  draftID: string,
  input: {
    title: string
    body: string
    labels: string[]
    expected_version: number
  },
  signal?: AbortSignal,
): Promise<RepositoryReviewIssueDraftResult> {
  return normalizeIssueDraftResult(
    await requestJSON<RepositoryReviewIssueDraftResult>(
      `${repositoryPath(repositoryID)}/issue-drafts/${encodeURIComponent(draftID)}`,
      jsonMutation("PATCH", input),
      signal,
    ),
  )
}

export async function publishRepositoryReviewIssueDraft(
  repositoryID: string,
  draftID: string,
  input: { expected_version: number },
  signal?: AbortSignal,
): Promise<RepositoryReviewIssueDraftResult> {
  return normalizeIssueDraftResult(
    await requestJSON<RepositoryReviewIssueDraftResult>(
      `${repositoryPath(repositoryID)}/issue-drafts/${encodeURIComponent(draftID)}/publish`,
      jsonMutation("POST", input),
      signal,
    ),
  )
}

function normalizeIssueDraftResult(
  value: RepositoryReviewIssueDraftResult,
): RepositoryReviewIssueDraftResult {
  return {
    repository: normalizeRepositoryReviewSummary(value.repository),
    draft: normalizeIssueDraft(value.draft),
    ...(value.outcome ? { outcome: value.outcome } : {}),
  }
}

function normalizeRepositoryReviewState(
  value: RepositoryReviewState,
): RepositoryReviewState {
  return {
    ...normalizeRepositoryReviewSummary(value),
    files: value.files ?? {},
    unsupported: value.unsupported ?? {},
    findings: (value.findings ?? []).map(normalizeFinding),
    contexts: (value.contexts ?? []).map((context) => ({
      ...context,
      files: context.files ?? [],
    })),
    runs: (value.runs ?? []).map((run) => ({
      ...run,
      remaining_files: run.remaining_files ?? run.unreviewed_files ?? 0,
      unsupported_files: run.unsupported_files ?? 0,
      unreviewed_paths: run.unreviewed_paths ?? [],
      unsupported_paths: run.unsupported_paths ?? [],
      models: run.models ?? [],
    })),
    issue_drafts: (value.issue_drafts ?? []).map(normalizeIssueDraft),
    repository_findings: (value.repository_findings ?? []).map(
      normalizeRepositoryFinding,
    ),
    mapping_jobs: (value.mapping_jobs ?? []).map(normalizeRepositoryMappingJob),
    validation_jobs: (value.validation_jobs ?? []).map(
      normalizeRepositoryValidationJob,
    ),
  }
}

function normalizeRepositoryReviewSummary(
  value: RepositoryReviewSummary,
): RepositoryReviewSummary {
  return { ...value, review_version: value.review_version ?? 0 }
}

function normalizeFileAttribution(
  attribution: RepositoryReviewFileAttribution,
): RepositoryReviewFileAttribution {
  return {
    id: attribution.id ?? "",
    path: attribution.path ?? "",
    commit_sha: attribution.commit_sha ?? "",
    blob_sha: attribution.blob_sha ?? "",
    focus_id: attribution.focus_id,
    ...(attribution.root_agent_id
      ? { root_agent_id: attribution.root_agent_id }
      : {}),
    ...(attribution.reviewer_identity
      ? { reviewer_identity: attribution.reviewer_identity }
      : {}),
    ...(attribution.account ? { account: attribution.account } : {}),
    ...(attribution.model ? { model: attribution.model } : {}),
    source: attribution.source ?? "legacy",
    sources: attribution.sources ?? [],
    attempts: attribution.attempts ?? 0,
    run_ids: attribution.run_ids ?? [],
    run_count: attribution.run_count ?? attribution.run_ids?.length ?? 0,
    latest_completed_at: attribution.latest_completed_at ?? "",
  }
}

function normalizeRunFindingSummary(
  finding: RepositoryReviewRunFindingSummary,
): RepositoryReviewRunFindingSummary {
  return {
    id: finding.id,
    repository: finding.repository,
    path: finding.path,
    ...(finding.line == null ? {} : { line: finding.line }),
    severity: finding.severity,
    title: finding.title,
    ...(finding.symbol ? { symbol: finding.symbol } : {}),
    status: finding.status,
    run_finding_status: finding.run_finding_status,
    association: finding.association,
    ...(finding.repository_finding_id
      ? { repository_finding_id: finding.repository_finding_id }
      : {}),
    contributors: finding.contributors ?? [],
    ...(finding.raw_source_count == null
      ? {}
      : { raw_source_count: finding.raw_source_count }),
    created_at: finding.created_at,
    updated_at: finding.updated_at,
  }
}

function normalizeRawFinding(
  finding: RepositoryReviewRawFinding,
): RepositoryReviewRawFinding {
  return {
    ...finding,
    path: finding.path || finding.file?.path,
    validation: finding.validation
      ? {
          ...finding.validation,
          checks: finding.validation.checks ?? [],
        }
      : undefined,
    match_hints: finding.match_hints
      ? normalizeMatchHints(finding.match_hints)
      : undefined,
    fix_effort: finding.fix_effort
      ? normalizeFixEffort(finding.fix_effort)
      : undefined,
    history: (finding.history ?? []).map((entry) => ({ ...entry })),
  }
}

function normalizeFindingsProcessingDetail(
  value: RepositoryReviewFindingsProcessingDetail,
): RepositoryReviewFindingsProcessingDetail {
  return {
    ...value,
    automation: normalizeAutomation(value.automation),
    repository: value.repository
      ? normalizeRepositoryReviewSummary(value.repository)
      : undefined,
    source: normalizeRawFinding(value.source),
    context: value.context ? normalizeFindingContext(value.context) : undefined,
    finding: value.finding ? normalizeFinding(value.finding) : undefined,
    repository_finding: value.repository_finding
      ? normalizeRepositoryFinding(value.repository_finding)
      : undefined,
    findings_processing: value.findings_processing
      ? normalizeFindingHealthProcessing(value.findings_processing)
      : undefined,
    historical_consolidation: value.historical_consolidation
      ? normalizeHistoricalConsolidation(value.historical_consolidation)
      : undefined,
    health: value.health ? normalizeFindingHealth(value.health) : undefined,
  }
}

function normalizeFindingHealth(
  value: Partial<RepositoryReviewFindingHealth>,
): RepositoryReviewFindingHealth {
  const run = value.run_findings
  const repository = value.repository_findings
  return {
    run_findings: {
      total: run?.total ?? 0,
      pending: run?.pending ?? 0,
      processing: run?.processing ?? 0,
      failed: run?.failed ?? 0,
      needs_review: run?.needs_review ?? 0,
      associated_new: run?.associated_new ?? 0,
      associated_existing: run?.associated_existing ?? 0,
      unrepresented: run?.unrepresented ?? 0,
    },
    repository_findings: {
      total: repository?.total ?? 0,
      provisional: repository?.provisional ?? 0,
      validation_failed: repository?.validation_failed ?? 0,
      issue_conflicts: repository?.issue_conflicts ?? 0,
    },
    findings_processing: normalizeFindingHealthProcessing(
      value.findings_processing,
    ),
    historical_consolidation: normalizeHistoricalConsolidation(
      value.historical_consolidation,
    ),
    updated_at: value.updated_at ?? "",
  }
}

function normalizeFindingHealthProcessing(
  counters?: Partial<RepositoryReviewFindingHealthProcessing>,
): RepositoryReviewFindingHealthProcessing {
  return {
    total: counters?.total ?? 0,
    pending: counters?.pending ?? 0,
    processing: counters?.processing ?? 0,
    failed: counters?.failed ?? 0,
    completed: counters?.completed ?? 0,
  }
}

function normalizeHistoricalConsolidation(
  value?: Partial<RepositoryReviewHistoricalConsolidation>,
): RepositoryReviewHistoricalConsolidation {
  const validStatuses = new Set<RepositoryReviewHistoricalConsolidationStatus>([
    "not_required",
    "pending",
    "replaying",
    "merging",
    "failed",
    "completed",
  ])
  const required = value?.required ?? false
  const status = validStatuses.has(
    value?.status as RepositoryReviewHistoricalConsolidationStatus,
  )
    ? (value?.status as RepositoryReviewHistoricalConsolidationStatus)
    : required
      ? "pending"
      : "not_required"
  return {
    required,
    status,
    retryable: value?.retryable ?? false,
  }
}

function normalizeFindingsProcessingCounters(
  counters: RepositoryReviewFindingsProcessingCounters,
): RepositoryReviewFindingsProcessingCounters {
  return {
    raw_total: counters.raw_total ?? 0,
    pending: counters.pending ?? 0,
    processing: counters.processing ?? 0,
    failed: counters.failed ?? 0,
    completed: counters.completed ?? 0,
    new: counters.new ?? 0,
    duplicates: counters.duplicates ?? 0,
    ...(counters.updated_at ? { updated_at: counters.updated_at } : {}),
  }
}

function normalizeHistoricalDeduplication(
  replay: RepositoryReviewHistoricalDeduplication,
): RepositoryReviewHistoricalDeduplication {
  const validStatus = new Set<RepositoryReviewHistoricalDeduplicationStatus>([
    "pending",
    "replaying",
    "merging",
    "failed",
    "completed",
  ]).has(replay.status as RepositoryReviewHistoricalDeduplicationStatus)
    ? replay.status
    : undefined
  return {
    required: replay.required ?? false,
    ...(validStatus ? { status: validStatus } : {}),
    ...(replay.attempts == null ? {} : { attempts: replay.attempts }),
    ...(replay.error ? { error: replay.error } : {}),
    ...(replay.updated_at ? { updated_at: replay.updated_at } : {}),
  }
}

function normalizeRepositoryFindingSummary(
  finding: RepositoryReviewRepositoryFindingSummary,
): RepositoryReviewRepositoryFindingSummary {
  return {
    id: finding.id,
    repository: finding.repository,
    canonical_title: finding.canonical_title,
    canonical_severity: finding.canonical_severity,
    ...(finding.path ? { path: finding.path } : {}),
    ...(finding.symbol ? { symbol: finding.symbol } : {}),
    match_state: finding.match_state,
    lifecycle: finding.lifecycle,
    issue: {
      ...(finding.issue?.url ? { url: finding.issue.url } : {}),
      state: finding.issue?.state ?? "none",
      ...(finding.issue?.snapshot_at
        ? { snapshot_at: finding.issue.snapshot_at }
        : {}),
      ...(finding.issue?.conflict ? { conflict: true } : {}),
    },
    validation_state: finding.validation_state,
    occurrence_count: finding.occurrence_count ?? 0,
    found_commit_count: finding.found_commit_count ?? 0,
    created_at: finding.created_at,
    updated_at: finding.updated_at,
  }
}

function normalizeIssueSummary(
  issue: RepositoryReviewIssueSummary,
): RepositoryReviewIssueSummary {
  return {
    id: issue.id,
    repository: issue.repository,
    finding_count: issue.finding_count ?? 0,
    origin: issue.origin || "legacy",
    ...(issue.generation_id ? { generation_id: issue.generation_id } : {}),
    canonical: issue.canonical ?? false,
    publishable: issue.publishable ?? false,
    publish_blockers: normalizePublishBlockers(issue.publish_blockers),
    title: issue.title,
    state: issue.state,
    version: issue.version,
    created_at: issue.created_at,
    updated_at: issue.updated_at,
  }
}

function normalizeFinding(
  finding: RepositoryReviewFinding,
): RepositoryReviewFinding {
  return {
    ...finding,
    context_ids: finding.context_ids ?? [],
    models: finding.models ?? [],
    observations: (finding.observations ?? []).map((observation) => ({
      ...observation,
      match_hints: observation.match_hints
        ? normalizeMatchHints(observation.match_hints)
        : undefined,
      fix_effort: observation.fix_effort
        ? normalizeFixEffort(observation.fix_effort)
        : undefined,
      validation: {
        ...observation.validation,
        checks: observation.validation?.checks ?? [],
      },
    })),
    match_hints: finding.match_hints
      ? normalizeMatchHints(finding.match_hints)
      : undefined,
    fix_effort: finding.fix_effort
      ? normalizeFixEffort(finding.fix_effort)
      : undefined,
    validation: {
      ...finding.validation,
      checks: finding.validation?.checks ?? [],
    },
    run_finding_status: normalizeRunFindingStatus(finding.run_finding_status),
  }
}

function normalizeRunFindingStatus(
  status: RepositoryReviewRunFindingStatusState | undefined,
): RepositoryReviewRunFindingStatusState | undefined {
  return new Set<RepositoryReviewRunFindingStatusState>([
    "pending",
    "processing",
    "failed",
    "associated_new",
    "associated_existing",
    "needs_review",
  ]).has(status as RepositoryReviewRunFindingStatusState)
    ? status
    : undefined
}

function normalizeFindingContext(
  context: RepositoryReviewFindingContext,
): RepositoryReviewFindingContext {
  return { ...context, files: context.files ?? [] }
}

function normalizeIssueDraft(
  draft: RepositoryReviewIssueDraft,
): RepositoryReviewIssueDraft {
  return {
    ...draft,
    finding_ids: draft.finding_ids ?? [],
    labels: draft.labels ?? [],
  }
}

function normalizeMatchHints(
  hints: RepositoryReviewMatchHints,
): RepositoryReviewMatchHints {
  return {
    ...hints,
    component: hints.component ?? "",
    operation: hints.operation ?? "",
    failure_mode: hints.failure_mode ?? "",
    trigger: hints.trigger ?? "",
    violated_invariant: hints.violated_invariant ?? "",
    observable_outcome: hints.observable_outcome ?? "",
    related_symbols: hints.related_symbols ?? [],
    source_anchors: hints.source_anchors ?? [],
    distinguishing_facts: hints.distinguishing_facts ?? [],
  }
}

function normalizeFixEffort(
  effort: RepositoryReviewFixEffort,
): RepositoryReviewFixEffort {
  return {
    quick: normalizeFixEffortEstimate(effort.quick),
    quality: normalizeFixEffortEstimate(effort.quality),
  }
}

function normalizeFixEffortEstimate(
  estimate: RepositoryReviewFixEffortEstimate | undefined,
): RepositoryReviewFixEffortEstimate {
  return {
    loc_min: estimate?.loc_min ?? 0,
    loc_max: estimate?.loc_max ?? 0,
    class: estimate?.class ?? "",
    rationale: estimate?.rationale ?? "",
  }
}

function normalizeRepositoryFinding(
  finding: RepositoryFinding,
): RepositoryFinding {
  return {
    ...finding,
    review_finding_ids: finding.review_finding_ids ?? [],
    found_commits: finding.found_commits ?? [],
    path_symbol_history: finding.path_symbol_history ?? [],
    match_hints: finding.match_hints
      ? normalizeMatchHints(finding.match_hints)
      : undefined,
    fix_effort: finding.fix_effort
      ? normalizeFixEffort(finding.fix_effort)
      : undefined,
    issue: {
      ...finding.issue,
      state: finding.issue?.state ?? "none",
      conflict_urls: finding.issue?.conflict_urls ?? [],
    },
    possible_duplicates: (finding.possible_duplicates ?? []).map(
      (candidate) => ({
        ...candidate,
        matching_anchors: candidate.matching_anchors ?? [],
        conflicting_anchors: candidate.conflicting_anchors ?? [],
      }),
    ),
    resolution_history: finding.resolution_history ?? [],
  }
}

function normalizeRepositoryMappingJob(
  job: RepositoryMappingJob,
): RepositoryMappingJob {
  return {
    ...job,
    adjudication: job.adjudication
      ? {
          ...job.adjudication,
          matching_anchors: job.adjudication.matching_anchors ?? [],
          conflicting_anchors: job.adjudication.conflicting_anchors ?? [],
        }
      : undefined,
  }
}

function normalizeRepositoryValidationJob(
  job: RepositoryValidationJob,
): RepositoryValidationJob {
  return { ...job, candidate_commits: job.candidate_commits ?? [] }
}

function normalizeFindingDetail(
  value: Partial<RepositoryReviewFindingDetail> & {
    draft?: RepositoryReviewIssueDraft
  },
): RepositoryReviewFindingDetail {
  return {
    ...value,
    automation: normalizeAutomation(value.automation!),
    repository: value.repository
      ? normalizeRepositoryReviewSummary(value.repository)
      : undefined,
    finding: normalizeFinding(value.finding!),
    action_finding: value.action_finding
      ? normalizeFinding(value.action_finding)
      : undefined,
    contexts: (value.contexts ?? []).map(normalizeFindingContext),
    repository_finding: value.repository_finding
      ? normalizeRepositoryFinding(value.repository_finding)
      : undefined,
    occurrences: value.occurrences?.map(normalizeFinding),
    possible_duplicate_findings: value.possible_duplicate_findings?.map(
      normalizeRepositoryFinding,
    ),
    raw_source_total:
      value.raw_source_total ?? value.finding?.raw_source_total ?? 0,
    issue: value.issue
      ? normalizeIssueDraft(value.issue)
      : value.draft
        ? normalizeIssueDraft(value.draft)
        : undefined,
    capabilities: normalizeCapabilities(value.capabilities),
  }
}

function normalizeIssueDetail(
  value: Partial<RepositoryReviewIssueDetail> & {
    draft?: RepositoryReviewIssueDraft
  },
): RepositoryReviewIssueDetail {
  return {
    ...value,
    automation: normalizeAutomation(value.automation!),
    repository: value.repository
      ? normalizeRepositoryReviewSummary(value.repository)
      : undefined,
    issue: normalizeIssueDraft(value.issue ?? value.draft!),
    finding: value.finding ? normalizeFinding(value.finding) : undefined,
    capabilities: normalizeCapabilities(value.capabilities),
  }
}

function normalizeCapabilities(
  capabilities: RepositoryReviewCapabilities | undefined,
): RepositoryReviewCapabilities | undefined {
  if (!capabilities) return undefined
  return {
    ...capabilities,
    publish_blockers: normalizePublishBlockers(capabilities.publish_blockers),
    purge_blockers: normalizePurgeBlockers(capabilities.purge_blockers),
    purge_summary: capabilities.purge_summary
      ? {
          repository_version:
            capabilities.purge_summary.repository_version ?? 0,
          ledger_fence: capabilities.purge_summary.ledger_fence ?? "",
          raw_findings: capabilities.purge_summary.raw_findings ?? 0,
          deduplicated_findings:
            capabilities.purge_summary.deduplicated_findings ?? 0,
          repository_findings:
            capabilities.purge_summary.repository_findings ?? 0,
          issue_previews: capabilities.purge_summary.issue_previews ?? 0,
          external_issue_associations:
            capabilities.purge_summary.external_issue_associations ?? 0,
        }
      : undefined,
  }
}

function normalizePurgeBlockers(
  blockers: RepositoryReviewPurgeBlocker[] | null | undefined,
): RepositoryReviewPurgeBlocker[] {
  return (blockers ?? []).map((blocker) => ({
    code: blocker.code,
    count: blocker.count ?? 0,
    message: blocker.message ?? "",
  }))
}

function normalizePublishBlockers(
  blockers: RepositoryReviewPublishBlocker[] | null | undefined,
): RepositoryReviewPublishBlocker[] {
  return (blockers ?? []).map((blocker) => ({
    code: blocker.code,
    count: blocker.count ?? 0,
    message: blocker.message ?? "",
  }))
}

function normalizeIssueMutationResponse(
  value: RepositoryReviewIssueMutationResponse,
): RepositoryReviewIssueMutationResponse {
  return {
    ...value,
    automation: value.automation
      ? normalizeAutomation(value.automation)
      : undefined,
    repository: value.repository
      ? normalizeRepositoryReviewSummary(value.repository)
      : undefined,
    issue: value.issue ? normalizeIssueDraft(value.issue) : undefined,
    draft: value.draft ? normalizeIssueDraft(value.draft) : undefined,
    finding: value.finding ? normalizeFinding(value.finding) : undefined,
    issues: value.issues?.map(normalizeIssueDraft),
    results: value.results ?? [],
  }
}

function normalizeRepositoryFindingMutation(
  value: RepositoryFindingMutationResponse,
): RepositoryFindingMutationResponse {
  return {
    ...value,
    automation: value.automation
      ? normalizeAutomation(value.automation)
      : undefined,
    repository: value.repository
      ? normalizeRepositoryReviewSummary(value.repository)
      : undefined,
    repository_finding: value.repository_finding
      ? normalizeRepositoryFinding(value.repository_finding)
      : undefined,
    validation_jobs: value.validation_jobs ?? [],
  }
}

interface AutomationMutationResult {
  automation: RepositoryReviewAutomation
}

interface ProfileMutationResult {
  profile: RepositoryReviewProfile
}

async function mutateAutomationAction(
  automationID: string,
  action: "start" | "pause" | "resume" | "restart",
  input: { expected_version: number; commit_sha?: string; run_id?: string },
  signal?: AbortSignal,
): Promise<RepositoryReviewAutomation> {
  return automationFromMutation(
    await requestJSON<RepositoryReviewAutomation | AutomationMutationResult>(
      `${automationPath(automationID)}/${action}`,
      jsonMutation("POST", input),
      signal,
    ),
  )
}

function automationFromMutation(
  value: RepositoryReviewAutomation | AutomationMutationResult,
): RepositoryReviewAutomation {
  return normalizeAutomation("automation" in value ? value.automation : value)
}

function profileFromMutation(
  value: RepositoryReviewProfile | ProfileMutationResult,
): RepositoryReviewProfile {
  return normalizeProfile("profile" in value ? value.profile : value)
}

function normalizeProfile(
  profile: RepositoryReviewProfile,
): RepositoryReviewProfile {
  return {
    ...profile,
    name: profile.name ?? "Review profile",
    account_ref: profile.account_ref ?? "",
    review_focus: profile.review_focus ?? "",
    issue_prompt:
      profile.issue_prompt?.trim() || repositoryReviewDefaultIssuePrompt,
    reviewer_model: profile.reviewer_model ?? "",
    deduplication_model: profile.deduplication_model ?? "",
    deduplication_similarity_threshold:
      profile.deduplication_similarity_threshold ?? 90,
    deduplication_candidate_limit: profile.deduplication_candidate_limit ?? 4,
    issue_writer_model: profile.issue_writer_model ?? "",
    force: profile.force ?? false,
    auto_continue: profile.auto_continue ?? true,
    max_files_per_run: profile.max_files_per_run ?? 24,
    max_content_bytes: profile.max_content_bytes ?? 524_288,
    max_parallel_children: profile.max_parallel_children ?? 8,
    assignment_timeout_seconds: profile.assignment_timeout_seconds ?? 3_600,
    scope_policy: {
      code_types:
        profile.scope_policy?.code_types?.length > 0
          ? profile.scope_policy.code_types
          : ["hotpath-code", "code"],
      include_folders: profile.scope_policy?.include_folders ?? [],
      exclude_folders: profile.scope_policy?.exclude_folders ?? [],
      free_text: profile.scope_policy?.free_text ?? "",
    },
    budget: normalizeBudget(profile.budget),
  }
}

function normalizeAutomation(
  automation: RepositoryReviewAutomation,
): RepositoryReviewAutomation {
  return {
    ...automation,
    name: automation.name ?? automation.repository ?? "Repository review",
    repository: automation.repository ?? "",
    profile_id: automation.profile_id ?? "",
    profile_version: automation.profile_version ?? 0,
    pause_reason: normalizePauseReason(automation.pause_reason),
    branch: automation.branch ?? automation.ref ?? "",
    ref: automation.branch ?? automation.ref ?? "",
    target: automation.target ?? "all",
    account_ref: automation.account_ref ?? "",
    effective_account_ref: automation.effective_account_ref ?? "",
    review_focus: automation.review_focus ?? "",
    scope_policy: {
      code_types:
        automation.scope_policy?.code_types?.length > 0
          ? automation.scope_policy.code_types
          : ["hotpath-code", "code"],
      include_folders: automation.scope_policy?.include_folders ?? [],
      exclude_folders: automation.scope_policy?.exclude_folders ?? [],
      free_text: automation.scope_policy?.free_text ?? "",
    },
    reviewer_models: automation.reviewer_models ?? [],
    issue_writer_model:
      automation.issue_writer_model ?? automation.reviewer_models?.[0] ?? "",
    compare_models: automation.compare_models ?? false,
    force: automation.force ?? false,
    max_files_per_run: automation.max_files_per_run ?? 24,
    max_content_bytes: automation.max_content_bytes ?? 524_288,
    max_parallel_children: automation.max_parallel_children ?? 8,
    assignment_timeout_seconds: automation.assignment_timeout_seconds ?? 3_600,
    auto_continue: automation.auto_continue ?? true,
    run_ids: automation.run_ids ?? [],
    model_prices: automation.model_prices ?? {},
    budget: normalizeBudget(automation.budget),
    usage: normalizeUsage(automation.usage),
    estimated_cost_usd: automation.estimated_cost_usd ?? 0,
    progress: {
      stage: automation.progress?.stage ?? "waiting",
      completed_batches: automation.progress?.completed_batches ?? 0,
      total_batches: automation.progress?.total_batches ?? 0,
      coverage_available: automation.progress?.coverage_available ?? false,
      coverage_exact: automation.progress?.coverage_exact ?? false,
      selected_files: automation.progress?.selected_files ?? 0,
      inspected_files: automation.progress?.inspected_files ?? 0,
      reviewed_files: automation.progress?.reviewed_files ?? 0,
      remaining_files: automation.progress?.remaining_files ?? 0,
      unsupported_files: automation.progress?.unsupported_files ?? 0,
      raw_findings: automation.progress?.raw_findings ?? 0,
      deduplicated_findings:
        automation.progress?.deduplicated_findings ??
        automation.progress?.findings ??
        0,
      findings:
        automation.progress?.findings ??
        automation.progress?.deduplicated_findings ??
        0,
      assignment_progress: normalizeAssignmentProgress(
        automation.progress?.assignment_progress,
      ),
      scope_frozen: automation.progress?.scope_frozen ?? false,
      finding_aggregates: automation.progress?.finding_aggregates ?? 0,
      unaggregated_findings: automation.progress?.unaggregated_findings ?? 0,
    },
    model_stats: normalizeModelStats(automation.model_stats),
    account_limits: normalizeAccountSnapshots(automation.account_limits),
    scope_plan: automation.scope_plan
      ? {
          ...automation.scope_plan,
          warnings: automation.scope_plan.warnings ?? [],
          counts: {
            total_files: automation.scope_plan.counts?.total_files ?? 0,
            code_type_files: automation.scope_plan.counts?.code_type_files ?? 0,
            include_files: automation.scope_plan.counts?.include_files ?? 0,
            excluded_files: automation.scope_plan.counts?.excluded_files ?? 0,
            selected_files: automation.scope_plan.counts?.selected_files ?? 0,
          },
        }
      : undefined,
    started_at: normalizeOptionalTimestamp(automation.started_at),
    completed_at: normalizeOptionalTimestamp(automation.completed_at),
  }
}

function normalizeAssignmentProgress(
  progress: RepositoryReviewAssignmentProgress | undefined,
): RepositoryReviewAssignmentProgress {
  const counts = (value: RepositoryReviewAssignmentFocusProgress | undefined) =>
    ({
      total: value?.total ?? 0,
      completed: value?.completed ?? 0,
      pending: value?.pending ?? 0,
      active: value?.active ?? 0,
    }) satisfies RepositoryReviewAssignmentFocusProgress
  return {
    ...counts(progress),
    by_focus: {
      correctness_state: counts(progress?.by_focus?.correctness_state),
      security_trust: counts(progress?.by_focus?.security_trust),
      concurrency_recovery: counts(progress?.by_focus?.concurrency_recovery),
      integration_validation: counts(
        progress?.by_focus?.integration_validation,
      ),
    },
  }
}

function normalizePauseReason(
  value: RepositoryReviewPauseReason | undefined,
): RepositoryReviewPauseReason | undefined {
  switch (value) {
    case "manual":
    case "token_budget":
    case "cost_budget":
    case "account_limit":
    case "guard_expression":
    case "no_progress":
    case "run_failed":
    case "service_restart":
      return value
    default:
      return undefined
  }
}

function normalizeBudget(
  budget?: Partial<RepositoryReviewAutomationBudget>,
): RepositoryReviewAutomationBudget {
  return {
    guard_expression: budget?.guard_expression ?? "",
  }
}

function normalizeUsage(
  usage?: Partial<RepositoryReviewTokenUsage>,
): RepositoryReviewTokenUsage {
  return {
    prompt_tokens: usage?.prompt_tokens ?? 0,
    completion_tokens: usage?.completion_tokens ?? 0,
    total_tokens: usage?.total_tokens ?? 0,
    cached_tokens: usage?.cached_tokens ?? 0,
  }
}

function normalizeModelStats(value: unknown): RepositoryReviewModelStats[] {
  const rows: Array<[string, Record<string, unknown>]> = Array.isArray(value)
    ? value.flatMap((candidate) => {
        const record = isRecord(candidate) ? candidate : undefined
        return record && typeof record.model === "string"
          ? [[record.model, record]]
          : []
      })
    : isRecord(value)
      ? Object.entries(value).flatMap(([model, candidate]) =>
          isRecord(candidate) ? [[model, candidate]] : [],
        )
      : []
  return rows.map(([model, stats]) => {
    const nestedTokens = isRecord(stats.tokens) ? stats.tokens : undefined
    const usage = normalizeUsage(
      (nestedTokens ?? stats) as Partial<RepositoryReviewTokenUsage>,
    )
    return {
      ...usage,
      model,
      estimated_cost_usd: numberValue(stats.estimated_cost_usd),
      requests: numberValue(stats.requests),
      failures: numberValue(stats.failures),
      reviewed_files: numberValue(stats.reviewed_files),
      findings: numberValue(stats.findings),
      latency_ms: numberValue(stats.latency_ms ?? stats.latency_millis),
    }
  })
}

function normalizeAccount<T extends ReviewAccountOption>(account: T): T {
  return {
    ...account,
    available: account.available ?? false,
    models: Array.isArray(account.models) ? account.models : [],
    writer_models: Array.isArray(account.writer_models)
      ? account.writer_models
      : Array.isArray(account.models)
        ? account.models
        : [],
    entries: (account.entries ?? []).map((entry) => ({
      ...entry,
      label: entry.label ?? entry.name,
      reset_at: normalizeOptionalTimestamp(
        entry.reset_at ?? entry.refreshes_at,
      ),
      refreshed_at: normalizeOptionalTimestamp(entry.refreshed_at),
    })),
  }
}

function normalizeAccountSnapshots(
  value: unknown,
): RepositoryReviewAccountSnapshot[] {
  if (!Array.isArray(value)) return []
  const grouped = new Map<string, RepositoryReviewAccountSnapshot>()
  for (const candidate of value) {
    if (!isRecord(candidate)) continue
    if (typeof candidate.account_id === "string") {
      const id = candidate.account_id
      const existing = grouped.get(id) ?? {
        id,
        provider: "",
        label: id,
        status: "available",
        entries: [],
      }
      const remaining =
        typeof candidate.remaining_percent === "number"
          ? candidate.remaining_percent
          : undefined
      existing.entries.push({
        window:
          typeof candidate.window === "string" ? candidate.window : "default",
        label:
          typeof candidate.name === "string" && candidate.name
            ? candidate.name
            : undefined,
        remaining_percent: remaining,
        status:
          typeof candidate.detail === "string" && candidate.detail
            ? candidate.detail
            : remaining === undefined
              ? "unknown"
              : "available",
        reset_at:
          typeof candidate.resets_at === "string"
            ? normalizeOptionalTimestamp(candidate.resets_at)
            : undefined,
        refreshed_at:
          typeof candidate.checked_at === "string"
            ? normalizeOptionalTimestamp(candidate.checked_at)
            : undefined,
      })
      existing.refreshed_at =
        typeof candidate.checked_at === "string"
          ? normalizeOptionalTimestamp(candidate.checked_at)
          : existing.refreshed_at
      if (remaining === undefined) existing.status = "unknown"
      grouped.set(id, existing)
      continue
    }
    if (typeof candidate.id !== "string") continue
    grouped.set(
      candidate.id,
      normalizeAccount(candidate as unknown as RepositoryReviewAccountSnapshot),
    )
  }
  return [...grouped.values()]
}

function numberValue(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0
}

function normalizeOptionalTimestamp(
  value: string | undefined,
): string | undefined {
  if (!value) return undefined
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) || parsed.getUTCFullYear() <= 1
    ? undefined
    : value
}

function repositoryPath(repositoryID: string): string {
  return `${apiRoot}/${encodeURIComponent(repositoryID)}`
}

function automationPath(automationID: string): string {
  return `${apiRoot}/automations/${encodeURIComponent(automationID)}`
}

function automationFindingPath(
  automationID: string,
  findingID: string,
): string {
  return `${automationPath(automationID)}/findings/${encodeURIComponent(findingID)}`
}

function automationIssuePath(automationID: string, draftID: string): string {
  return `${automationPath(automationID)}/issues/${encodeURIComponent(draftID)}`
}

function profilePath(profileID: string): string {
  return `${apiRoot}/profiles/${encodeURIComponent(profileID)}`
}

function repositoryReviewProfileConfigPayload(
  input: RepositoryReviewProfileConfig,
): RepositoryReviewProfileConfig {
  return {
    name: input.name,
    account_ref: input.account_ref,
    review_focus: input.review_focus,
    issue_prompt: input.issue_prompt,
    scope_policy: input.scope_policy,
    reviewer_model: input.reviewer_model,
    ...(input.deduplication_model !== undefined
      ? { deduplication_model: input.deduplication_model }
      : {}),
    ...(input.deduplication_similarity_threshold !== undefined
      ? {
          deduplication_similarity_threshold:
            input.deduplication_similarity_threshold,
        }
      : {}),
    ...(input.deduplication_candidate_limit !== undefined
      ? { deduplication_candidate_limit: input.deduplication_candidate_limit }
      : {}),
    ...(input.issue_writer_model !== undefined
      ? { issue_writer_model: input.issue_writer_model }
      : {}),
    force: input.force,
    auto_continue: input.auto_continue,
    max_files_per_run: input.max_files_per_run,
    max_content_bytes: input.max_content_bytes,
    max_parallel_children: input.max_parallel_children,
    assignment_timeout_seconds: input.assignment_timeout_seconds,
    budget: input.budget,
  }
}

function jsonMutation(
  method: "POST" | "PATCH" | "DELETE",
  body: unknown,
): RequestInit {
  return {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  }
}

async function requestVoid(
  path: string,
  init?: RequestInit,
  signal?: AbortSignal,
): Promise<void> {
  const response = await launcherFetch(path, { ...init, signal })
  if (response.ok) return
  const contentType = response.headers.get("Content-Type")
  const isJSON =
    contentType?.split(";", 1)[0]?.trim().toLowerCase() === "application/json"
  let payload: unknown
  if (isJSON) {
    try {
      payload = await response.json()
    } catch {
      payload = undefined
    }
  }
  const record = isRecord(payload) ? payload : undefined
  throw new RepositoryReviewAPIError(
    response.status,
    typeof record?.message === "string"
      ? record.message
      : "Repository review request failed.",
    typeof record?.code === "string" ? record.code : undefined,
  )
}

async function requestJSON<T>(
  path: string,
  init?: RequestInit,
  signal?: AbortSignal,
): Promise<T> {
  const response = await launcherFetch(path, { ...init, signal })
  const contentType = response.headers.get("Content-Type")
  const isJSON =
    contentType?.split(";", 1)[0]?.trim().toLowerCase() === "application/json"
  let payload: unknown
  if (isJSON) {
    try {
      payload = await response.json()
    } catch {
      throw new RepositoryReviewAPIError(
        response.ok ? 502 : response.status,
        "Repository review returned malformed JSON.",
        "malformed_response",
      )
    }
  }
  if (!response.ok) {
    const record = isRecord(payload) ? payload : undefined
    throw new RepositoryReviewAPIError(
      response.status,
      typeof record?.message === "string"
        ? record.message
        : "Repository review request failed.",
      typeof record?.code === "string" ? record.code : undefined,
    )
  }
  if (!isJSON || !isRecord(payload)) {
    throw new RepositoryReviewAPIError(
      502,
      "Repository review returned a malformed response.",
      "malformed_response",
    )
  }
  return payload as T
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}
