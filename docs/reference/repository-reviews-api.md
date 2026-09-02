# Repository Reviews API

This is the normative launcher HTTP contract for Repository Reviews. The
feature semantics, durable state machines, and model-output schemas are defined
in [Repository Reviews](../features/repository-reviews.md).

## Protocol And Authentication

- The base prefix is `/api/repository-reviews` and responses are JSON except
  successful profile and automation-configuration deletes, which return
  `204 No Content`. Issue-preview and issue-link deletes return JSON envelopes.
- These routes use the launcher's authenticated, trusted single-operator
  boundary. Repository Reviews adds no tenant or role model. Mutations also
  enforce the launcher's same-origin/replay-header checks.
- Mutation requests use `Content-Type: application/json`, reject unknown fields
  and trailing JSON, and accept no query parameters. The general decoded body
  limit is 8 MiB; issue-link proxy bodies are limited to 32 KiB and the legacy
  repository-owned publish proxy is limited to 16 KiB.
- Standard collections accept exactly one each of `query`, `cursor`, and
  `limit`. `limit` is 1–200; omitted uses the collection default. The response
  contains the item key named below, `total`, optional `next_cursor`,
  `canonical_query`, and `query_schema`. A cursor is opaque and bound to its
  resource, normalized query, page size, repository/campaign/generation
  context, and ordering.
- IDs are opaque. Canonical prefixes are `rrpf_*` profile, `rra_*` automation,
  `rrw_*` raw finding, `rdf_*` deduplicated finding, `rrf_*` repository
  finding, and `rid_*` issue preview. Clients must not derive authority from
  a prefix.

### Registered Route Manifest

This manifest is exhaustive and uses the exact Go `ServeMux` patterns. Route
parity tests compare registration against these entries. The legacy tail route
accepts only the concrete `issue-drafts/{draft_id}/publish` shape described
later.

```text
DELETE /api/repository-reviews/automations/{automation_id}
DELETE /api/repository-reviews/automations/{automation_id}/findings/{finding_id}/issue-link
DELETE /api/repository-reviews/automations/{automation_id}/issues/{draft_id}
DELETE /api/repository-reviews/profiles/{profile_id}
GET /api/repository-reviews
GET /api/repository-reviews/automation-options
GET /api/repository-reviews/automations
GET /api/repository-reviews/automations/{automation_id}
GET /api/repository-reviews/automations/{automation_id}/campaigns/{campaign_id}/findings-processing
GET /api/repository-reviews/automations/{automation_id}/campaigns/{campaign_id}/findings-processing/sources/{source_id}
GET /api/repository-reviews/automations/{automation_id}/commit-options
GET /api/repository-reviews/automations/{automation_id}/file-attributions
GET /api/repository-reviews/automations/{automation_id}/finding-health
GET /api/repository-reviews/automations/{automation_id}/findings
GET /api/repository-reviews/automations/{automation_id}/findings-processing
GET /api/repository-reviews/automations/{automation_id}/findings-processing/sources/{source_id}
GET /api/repository-reviews/automations/{automation_id}/findings/{finding_id}
GET /api/repository-reviews/automations/{automation_id}/findings/{finding_id}/sources
GET /api/repository-reviews/automations/{automation_id}/findings/{finding_id}/sources/{source_id}
GET /api/repository-reviews/automations/{automation_id}/historical-deduplication
GET /api/repository-reviews/automations/{automation_id}/issues
GET /api/repository-reviews/automations/{automation_id}/issues/{draft_id}
GET /api/repository-reviews/automations/{automation_id}/raw-findings
GET /api/repository-reviews/automations/{automation_id}/raw-findings/{source_id}
GET /api/repository-reviews/automations/{automation_id}/report
GET /api/repository-reviews/automations/{automation_id}/repository-findings
GET /api/repository-reviews/automations/{automation_id}/repository-findings/{finding_id}
GET /api/repository-reviews/automations/{automation_id}/run-findings
GET /api/repository-reviews/automations/{automation_id}/run-findings/{finding_id}
GET /api/repository-reviews/profiles
GET /api/repository-reviews/profiles/{profile_id}
GET /api/repository-reviews/{repository_id}
PATCH /api/repository-reviews/automations/{automation_id}
PATCH /api/repository-reviews/automations/{automation_id}/findings/{finding_id}
PATCH /api/repository-reviews/automations/{automation_id}/issues/{draft_id}
PATCH /api/repository-reviews/automations/{automation_id}/repository-findings/{repository_finding_id}
PATCH /api/repository-reviews/profiles/{profile_id}
PATCH /api/repository-reviews/{repository_id}/findings/{finding_id}
PATCH /api/repository-reviews/{repository_id}/issue-drafts/{draft_id}
POST /api/repository-reviews/automations
POST /api/repository-reviews/automations/{automation_id}/campaigns/{campaign_id}/findings-processing/sources/{source_id}/retry
POST /api/repository-reviews/automations/{automation_id}/findings-processing/retry
POST /api/repository-reviews/automations/{automation_id}/findings-processing/sources/{source_id}/retry
POST /api/repository-reviews/automations/{automation_id}/findings/status
POST /api/repository-reviews/automations/{automation_id}/findings/{finding_id}/issue-link
POST /api/repository-reviews/automations/{automation_id}/findings/{finding_id}/issue-link/candidates
POST /api/repository-reviews/automations/{automation_id}/findings/{finding_id}/post
POST /api/repository-reviews/automations/{automation_id}/historical-deduplication/restart
POST /api/repository-reviews/automations/{automation_id}/historical-deduplication/retry
POST /api/repository-reviews/automations/{automation_id}/issues/generations
POST /api/repository-reviews/automations/{automation_id}/issues/publish
POST /api/repository-reviews/automations/{automation_id}/issues/{draft_id}/publish
POST /api/repository-reviews/automations/{automation_id}/issues/{draft_id}/regenerate
POST /api/repository-reviews/automations/{automation_id}/pause
POST /api/repository-reviews/automations/{automation_id}/purge-history
POST /api/repository-reviews/automations/{automation_id}/raw-findings/{source_id}/retry
POST /api/repository-reviews/automations/{automation_id}/repository-findings/validations
POST /api/repository-reviews/automations/{automation_id}/repository-findings/{repository_finding_id}/duplicates
POST /api/repository-reviews/automations/{automation_id}/repository-findings/{repository_finding_id}/sync
POST /api/repository-reviews/automations/{automation_id}/restart
POST /api/repository-reviews/automations/{automation_id}/resume
POST /api/repository-reviews/automations/{automation_id}/start
POST /api/repository-reviews/profiles
POST /api/repository-reviews/{repository_id}/issue-drafts
POST /api/repository-reviews/{repository_id}/{legacy_action...}
```

## Request Bodies

All fields shown are required unless marked optional. Version values are
positive integers except `expected_repository_version`, which is zero for an
assignment with no ledger and otherwise positive. The complete profile field
defaults and bounds are in the feature contract.

| Name | JSON object |
| --- | --- |
| `ProfileCreate` | Writable profile fields: `name`, `review_focus`, `scope_policy`, `reviewer_model`, optional `deduplication_model`, optional `deduplication_similarity_threshold`, optional `deduplication_candidate_limit`, optional `issue_writer_model`, optional `issue_prompt`, optional `account_ref`, `force`, optional `auto_continue`, `max_files_per_run`, `max_content_bytes`, `max_parallel_children`, optional `assignment_timeout_seconds`, and `budget` |
| `ProfileUpdate` | `ProfileCreate` plus `expected_version`; response metadata is forbidden |
| `ProfileDelete` | `{ "expected_version": 7 }` |
| `AutomationCreate` | `{ "repository": "owner/repository", "profile_id": "rrpf_...", "branch": "main" }`; `branch` is optional/blank for the advertised default; display name is materialized from repository/profile |
| `AutomationUpdate` | `AutomationCreate` fields plus `expected_version`; repository remains uniquely assigned |
| `AutomationAction` | `{ "expected_version": 7, "commit_sha": "<optional full SHA>", "run_id": "<optional active run ID>" }`; only action-relevant optional fields are accepted |
| `PurgeOrRemove` | `{ "expected_version": 7, "expected_repository_version": 12, "expected_ledger_fence": "rplf_...", "confirm_repository": "owner/repository" }`; confirmation must equal the displayed normalized identity and the opaque fence must equal the detail capability exactly |
| `RunFindingRetry` | `{ "finding_ids": ["..."] }` with explicit unique occurrence IDs |
| `ProcessingRetry` | `{ "source_ids": ["rrw_..."] }` with 1–200 explicit unique failed raw-source IDs |
| `EmptyMutation` | `{}` for single-source retry, historical retry, or issue synchronization |
| `HistoricalRestart` | `{ "confirmed": true }` |
| `FindingStatus` | `{ "status": "open|dismissed", "expected_version": 3 }`; compatibility occurrence mutation only; `posted` is forbidden |
| `RepositoryLifecycle` | `{ "lifecycle": "open|dismissed", "expected_version": 3 }` |
| `DuplicateDecision` | `{ "candidate_id": "rrf_...", "decision": "distinct|merge", "expected_provisional_version": 3, "expected_candidate_version": 4 }`; candidate version is required for merge |
| `ResolutionChecks` | `{ "repository_finding_ids": ["rrf_..."] }` with 1–50 explicit unique IDs |
| `IssueGeneration` | `{ "generation_id": "rrig_...", "finding_ids": ["<eligible occurrence anchor>"], "instructions_mode": "default|custom", "instructions": "<optional>" }`; 1–200 unique eligible canonical actions, custom instructions at most 16 KiB |
| `IssueEdit` | `{ "title": "...", "body": "...", "labels": ["bug"], "expected_version": 3 }`; title ≤256 bytes, body ≤60 KiB, labels are bounded |
| `IssueRegenerate` | `{ "expected_version": 3 }` |
| `IssueDelete` | `{ "expected_version": 3, "confirmed": true }` |
| `IssuePublish` | `{ "expected_version": 3, "confirmed": true }` |
| `IssueBatchPublish` | `{ "issues": [{ "id": "rid_...", "expected_version": 3 }], "confirmed": true }` with 1–200 explicit unique previews |
| `DirectPost` | `{ "expected_version": 3, "instructions": "<optional>" }`; instructions at most 16 KiB |
| `IssueCandidates` | `{ "expected_version": 3 }` |
| `IssueLink` | `{ "issue_url": "https://github.com/owner/repository/issues/1", "expected_version": 3, "confirmed": true, "replace": false }` |
| `IssueUnlink` | `{ "expected_version": 3, "confirmed": true }` |

## Profiles, Configuration, And Lifecycle Routes

`{aid}` means an automation ID and `{pid}` a profile ID.

| Method and path | Request/query | Success response |
| --- | --- | --- |
| `GET /profiles` | Standard collection | `200 {profiles,total,next_cursor?,canonical_query,query_schema}` |
| `POST /profiles` | `ProfileCreate` | `201 {profile}` |
| `GET /profiles/{pid}` | None | `200 {profile}` |
| `PATCH /profiles/{pid}` | `ProfileUpdate` | `200 {profile}` |
| `DELETE /profiles/{pid}` | `ProfileDelete` | `204` |
| `GET /automations` | Standard collection | `200 {automations,total,next_cursor?,canonical_query,query_schema}`; entries are compact public projections |
| `POST /automations` | `AutomationCreate` | `201 {automation}` |
| `GET /automations/{aid}` | None | `200 {automation,repository?,capabilities}` |
| `PATCH /automations/{aid}` | `AutomationUpdate` | `200 {automation}` |
| `DELETE /automations/{aid}` | `PurgeOrRemove` | `204`; configuration and every resolved repository-review ledger are removed |
| `POST /automations/{aid}/purge-history` | `PurgeOrRemove` | `200 {automation,outcome:"history_purged"}`; automation is fresh and idle and every resolved repository-review ledger is removed |
| `POST /automations/{aid}/start` | `AutomationAction` | `202 {automation,outcome:"started"}` |
| `POST /automations/{aid}/pause` | `AutomationAction` | `202 {automation}` |
| `POST /automations/{aid}/resume` | `AutomationAction` | `202 {automation,outcome:"started"}` |
| `POST /automations/{aid}/restart` | `AutomationAction` | `202 {automation,outcome:"started"}` |
| `GET /automations/{aid}/commit-options` | None | `200 {expected_version,remembered:{sha,short_sha,url?},latest:{sha,short_sha,url?},newer_commit_available}` |
| `GET /automation-options` | None | `200 {models,accounts,limits_error?}`; never returns credentials |

`capabilities` contains existing issue-action fields plus
`can_purge_history`, `can_remove_repository`, ordered `purge_blockers`, and
`purge_summary`. Each blocker is exactly `{code,count,message}`. Summary is
`{repository_version,ledger_fence,raw_findings,deduplicated_findings,repository_findings,issue_previews,external_issue_associations}`.
For a configured assignment without a ledger, the summary contains version and
counts of zero plus a deterministic empty-inventory `ledger_fence`,
`can_purge_history` is `false`, `can_remove_repository` is `true` when otherwise
quiescent, and `purge_blockers` is an empty array.
Capability booleans and blockers are present even when false/empty.
`ledger_fence` binds every configured-identity or authoritative legacy ledger
and its version, so a hidden alias change invalidates stale confirmation.
`retention_unavailable` is the fail-closed blocker when inventory cannot be read.
`external_issue_associations` counts unique stored external issue URLs, including
conflict URLs; deletion never dereferences or mutates them.
Capabilities are advisory: mutations re-evaluate them while holding the store
lock.
Purge/removal never calls GitHub and never deletes profiles, discussion
threads, or generic workflow-run records; those resource owners keep their own
retention policies.

## Collection Query Schemas

Every field is sortable. Enum suggestions are returned in `query_schema`; raw
enum values remain stable beneath UI labels.

| Collection | Fields | Default order |
| --- | --- | --- |
| Profiles | `id,name,account,reviewer,deduplicator,deduplication_threshold,deduplication_candidates,issue_writer,force,auto_continue,files,parallel,version,updated` | `name ASC` |
| Automations | `id,name,repository,branch,status,progress,reviewed,raw_findings,findings,updated` | `updated DESC` |
| File attributions | `path,commit,blob,focus,agent,reviewer,account,model,source,attempts,runs,latest` | `path ASC, focus ASC, reviewer ASC` |
| Findings (`rdf_*`) | `id,repository,title,path,symbol,severity,status,run_status,association,contributors,sources,mapped,created,updated` | `severity DESC, updated DESC` |
| Deprecated Run findings (stored `rfn_*` aliases only) | `id,repository,title,path,symbol,severity,status,run_status,association,contributors,created,updated` | `severity DESC, updated DESC` |
| Raw findings (`rrw_*`) | `id,path,severity,title,symbol,model,reviewer,deduplication_state,disposition,finding,created,updated` | `created DESC` |
| Findings processing | `id,campaign,title,path,symbol,severity,model,reviewer,state,disposition,created,updated` | `updated DESC` |
| Repository findings (`rrf_*`) | `id,repository,title,path,symbol,severity,match,lifecycle,issue,validation,occurrences,commits,created,updated` | `severity DESC, updated DESC` |
| Issue previews | `id,repository,title,generation,state,origin,canonical,publishable,findings,created,updated` | `updated DESC` |

## Evidence And Processing Routes

| Method and path | Request/query | Success response |
| --- | --- | --- |
| `GET /automations/{aid}/file-attributions` | Standard collection | `200 {file_attributions,total,next_cursor?,canonical_query,query_schema}` |
| `GET /automations/{aid}/finding-health` | None | `200 {run_findings,repository_findings,findings_processing,historical_consolidation,updated_at}` |
| `GET /automations/{aid}/findings` | Standard collection; legacy `scope`,`offset`,`limit` remains exclusive | Canonical `200 {automation,repository?,findings,total,next_cursor?,canonical_query,query_schema,capabilities,findings_processing,historical_deduplication}` with completed current-campaign `rdf_*` only; legacy mode returns the documented offset envelope |
| `GET /automations/{aid}/findings/{fid}` | None | `200 {automation,repository,finding,raw_source_total,contexts,repository_finding?,capabilities}` |
| `GET /automations/{aid}/findings/{fid}/sources` | `offset`, `limit` compatibility paging | `200 {automation,repository,finding_id,sources,offset,total,next_offset?}` |
| `GET /automations/{aid}/findings/{fid}/sources/{sid}` | None | Canonical raw-source detail envelope |
| `GET /automations/{aid}/raw-findings` | Standard collection | `200 {automation,repository?,raw_findings,total,next_cursor?,canonical_query,query_schema,findings_processing,historical_deduplication}` |
| `GET /automations/{aid}/raw-findings/{sid}` | None | `200 {automation,repository?,source,context?,finding?,historical_deduplication}` |
| `POST /automations/{aid}/raw-findings/{sid}/retry` | `EmptyMutation` | `202 {automation,repository,source,findings_processing}` |
| `GET /automations/{aid}/findings-processing` | Standard collection; legacy `offset`,`limit`,`state` remains exclusive | `200 {automation,repository?,raw_findings,total,next_cursor?,canonical_query,query_schema,capabilities,findings_processing,historical_consolidation}` |
| `POST /automations/{aid}/findings-processing/retry` | `ProcessingRetry` | `202 {retried_ids,failures,findings_processing,health}` |
| `GET /automations/{aid}/findings-processing/sources/{sid}` | None | Canonical repository-wide raw-source detail envelope |
| `POST /automations/{aid}/findings-processing/sources/{sid}/retry` | `EmptyMutation` | Single-source form of the processing retry response |
| `GET /automations/{aid}/campaigns/{cid}/findings-processing` | Legacy `offset`,`limit`,`state` | `200 {automation,repository,campaign_id,findings_processing,raw_findings,offset,total,next_offset?}` |
| `GET /automations/{aid}/campaigns/{cid}/findings-processing/sources/{sid}` | None | Canonical raw-source detail constrained to the campaign |
| `POST /automations/{aid}/campaigns/{cid}/findings-processing/sources/{sid}/retry` | `EmptyMutation` | `202` single-source retry envelope |
| `GET /automations/{aid}/historical-deduplication` | `offset`,`limit` | `200 {automation,repository,historical_deduplication,batches,raw_findings,offset,total,next_offset?}` |
| `POST /automations/{aid}/historical-deduplication/retry` | `EmptyMutation` | `202 {automation,repository,historical_deduplication}` |
| `POST /automations/{aid}/historical-deduplication/restart` | `HistoricalRestart` | `202 {automation,repository,historical_deduplication}` |

Deprecated compatibility routes are:

| Method and path | Contract |
| --- | --- |
| `GET /automations/{aid}/run-findings` | Standard collection of stored immutable `rfn_*` occurrence aliases; modern `rdf_*` projections are excluded and the route has no issue authority |
| `GET /automations/{aid}/run-findings/{fid}` | One stored `rfn_*` occurrence detail; every non-`rfn_*` ID returns `404` |
| `POST /automations/{aid}/findings/status` | `RunFindingRetry`; returns `202 {automation,repository,findings}` |
| `GET /automations/{aid}/report` | `scope`, `offset`, `limit` page containing legacy run and repository projections |
| `PATCH /automations/{aid}/findings/{fid}` | Accepts `FindingStatus` for wire compatibility but returns `409 stale_repository_review`; immutable occurrences cannot be mutated |

## Repository Findings And Issue Routes

| Method and path | Request/query | Success response |
| --- | --- | --- |
| `GET /automations/{aid}/repository-findings` | Standard collection | `200 {automation,repository?,repository_findings,total,next_cursor?,canonical_query,query_schema,capabilities}` |
| `GET /automations/{aid}/repository-findings/{rfid}` | None | `200 {automation,repository,finding,action_finding,repository_finding,occurrences,possible_duplicate_findings,contexts,issue?,capabilities}` |
| `PATCH /automations/{aid}/repository-findings/{rfid}` | `RepositoryLifecycle` | `200 {automation,repository,repository_finding}` |
| `POST /automations/{aid}/repository-findings/{rfid}/duplicates` | `DuplicateDecision` | `200 {automation,repository,repository_finding}`; a merge also returns the retained identity |
| `POST /automations/{aid}/repository-findings/validations` | `ResolutionChecks` | `202 {automation,repository,validation_jobs}` |
| `POST /automations/{aid}/repository-findings/{rfid}/sync` | `EmptyMutation` | `200 {automation,repository,repository_finding}` |
| `GET /automations/{aid}/issues` | Standard collection plus optional `generation_id`; legacy `offset`,`limit` is compatible | `200 {automation,repository?,issues,total,next_cursor?,canonical_query,query_schema,capabilities}` |
| `POST /automations/{aid}/issues/generations` | `IssueGeneration` | `200 {automation,repository,generation_id,issues,results}`; per-finding failures do not erase successes |
| `GET /automations/{aid}/issues/{did}` | None | `200 {automation,repository,issue,finding?,findings,capabilities}` |
| `PATCH /automations/{aid}/issues/{did}` | `IssueEdit` | `200 {automation,repository,issue}` |
| `DELETE /automations/{aid}/issues/{did}` | `IssueDelete` | `200 {automation,repository,outcome:"deleted"}` |
| `POST /automations/{aid}/issues/{did}/regenerate` | `IssueRegenerate` | `200 {automation,repository,issue,result}`; failed regeneration retains last good content |
| `POST /automations/{aid}/issues/{did}/publish` | `IssuePublish` | `200` posted/reconciled or `202` unknown publication envelope |
| `POST /automations/{aid}/issues/publish` | `IssueBatchPublish` | `200 {automation,repository,results}` with one result per requested preview |
| `POST /automations/{aid}/findings/{fid}/issue-link/candidates` | `IssueCandidates` | `200 {automation,finding,candidates,generator_model,generator_account,repository?,discovered_issue?}` |
| `POST /automations/{aid}/findings/{fid}/issue-link` | `IssueLink` | `200 {automation,repository,finding,issue}` |
| `DELETE /automations/{aid}/findings/{fid}/issue-link` | `IssueUnlink` | `200 {automation,repository,finding}` |
| `POST /automations/{aid}/findings/{fid}/post` | `DirectPost` | Posted/reconciled result over the exact saved generated preview |

Candidate ranking may auto-link only the first result at score ≥95 with at
least four matching anchors and no conflicting anchors, followed by exact
same-repository re-fetch. Every other result is read-only until explicit link.
For these four compatibility-shaped paths, `{fid}` is the eligible immutable
occurrence used to anchor the canonical repository-finding action; it is not an
alternative repository-finding identity.

## Repository-ID Compatibility Routes

These routes are retained for old clients. Automation-owned APIs above are the
canonical UI contract.

| Method and path | Request/query | Success response |
| --- | --- | --- |
| `GET /` | None | `200 {repositories}` containing compact ledger summaries |
| `GET /{repository_id}` | `offset`,`limit`,`draft_offset`,`draft_limit` | `200` allowlisted projection with summary fields plus paged `unsupported`, legacy `findings`, referenced `contexts`, bounded `runs`, and `issue_drafts` |
| `PATCH /{repository_id}/findings/{fid}` | `FindingStatus` | Returns `409 stale_repository_review`; immutable occurrences cannot be mutated |
| `POST /{repository_id}/issue-drafts` | Legacy issue request | Prepared draft plus compact summary |
| `PATCH /{repository_id}/issue-drafts/{did}` | `IssueEdit` | Changed draft plus compact summary |
| `POST /{repository_id}/issue-drafts/{did}/publish` | Legacy publish body | Protected publication/reconciliation result |

The compatibility detail allowlist is exactly the compact summary fields
`schema_version`, `id`, `repository`, `version`, `review_version`,
`last_commit_sha`, `finding_count`, `repository_finding_count`,
`open_finding_count`, `issue_draft_count`, `unsupported_count`,
`reviewed_file_count`, `excluded_file_count`, and `updated_at`, plus the
paged collections named above and their `finding_offset`, `finding_total`,
`next_finding_offset`, `draft_offset`, `draft_total`, and `next_draft_offset`.
It omits files, raw/deduplicated ledgers, jobs, attributions, canonical finding
records, validation state, campaign authority, and internal checkpoints.
Finding pages default to 50 and allow 1–200; draft pages default to 10 and allow
1–20. `unsupported` is capped at 200 sorted paths and `runs` at the latest 50.
Only contexts referenced by the returned finding page are included.

## Errors

Errors are bounded JSON: `{ "code": "stable_code", "message": "safe message" }`.
Collection query errors may also include a byte `position` and query
suggestions. Publication and purge errors may add their safe blocker arrays.
Raw provider errors, prompts, source, credentials, internal paths, and
checkpoint identities never appear.

| Status | Stable codes and meaning |
| --- | --- |
| `400` | `invalid_request`, `invalid_repository_review_profile`, `invalid_repository_review_automation`, `invalid_collection_request`, `invalid_query`, `invalid_cursor`, `invalid_page_limit`, `invalid_generation_id`, `invalid_issue_url`, or another route-specific invalid-input code; validation fails before mutation/effect |
| `404` | `not_found`, `repository_review_profile_not_found`, or `repository_review_history_not_found` |
| `409` | `stale_repository_review`, `stale_repository_review_profile`, `stale_repository_review_automation`, `repository_review_repository_assigned`, `repository_review_profile_assigned`, `repository_review_profile_active`, `repository_review_commit_selection_required`, `repository_review_purge_blocked`, `repository_review_purge_in_progress`, `historical_deduplication_in_progress`, or `historical_consolidation_restart_required` |
| `502` | `invalid_gateway_response` or a safe external operation failure where the external response was invalid |
| `503` | `repository_review_unavailable`, `repository_review_profile_unavailable`, `repository_review_automation_unavailable`, `issue_search_unavailable`, `issue_ranking_unavailable`, `issue_link_unavailable`, `issue_sync_unavailable`, `publication_unavailable`, or another safe dependency-unavailable code |

For `repository_review_purge_blocked`, the response is
`{code,message,purge_blockers}` and the ordered blockers use the same
`{code,count,message}` shape as capabilities. `repository_review_history_not_found`
means purge was requested for a configured assignment without an authoritative
history ledger. Confirmation mismatch and stale automation or repository
versions return `409 stale_repository_review_automation` without mutation.
While a durable primary intent or repository-identity fence exists,
`repository_review_purge_in_progress` rejects affected automation/ledger reads
and mutations until startup recovery completes; clients never receive a
partially purged projection.
