# Repository Reviews

## Feature ID

`FR-REPOREVIEW`

## Behavior Summary

Repository reviews provide a durable pre-review control plane and retain
validated bug findings independently from pull-request workspaces. Reusable
named review profiles own review behavior, one reviewer model, an optional issue
writer model, scope, real provider-call work bounds, and guardrails. A separate
repository configuration assigns exactly one profile to one normalized
repository and may select only an optional branch; blank follows the
repository's advertised default branch. Repository assignments, profiles, and
the compact review collection have separate launcher flows. Opening a compact
review summary provides lifecycle controls, live progress, usage, run history,
commit selection, and always-available Findings and Issue previews actions. The
Findings surface remains readable while work is active and exposes review-finding
checkpoints as they complete instead of waiting for a terminal result screen.
Model comparison is a separate repository model probe that reads and freezes an
evaluation-relevant profile snapshot at admission but never writes the profile,
review campaign, or finding ledger. An actual review exposes live stage and
cumulative consumption, pauses safely after the current checkpoint, resumes a
paused or failed campaign without resetting it only when both the frozen profile
snapshot and exact commit are unchanged, restarts into a new campaign, and
automatically continues through bounded batches. Each profile
selects one execution account (blank follows the runtime default) and may define
one bounded task-admission expression evaluated whenever a worker takes its
next task. At Start/Resume/Restart the chosen default is resolved and stored as
the campaign's effective account. Review calls and their completed-run
provenance always use that campaign-frozen account. Later Draft, direct Post
generation, regeneration, and candidate-ranking actions instead resolve the
profile currently assigned to the repository, resolve its effective account at
action admission, and freeze the profile/account/model revision for that
individual attempt. Generation/regeneration persists that snapshot on the
preview; ranking returns its generator model/account but is otherwise read-only.
Publishing an existing preview keeps the provenance and payload already frozen
on that preview.
Start resolves the configured branch or remote default to one exact commit and
persists that commit in the same transition that creates the active workflow
run. Every automatic batch stays on that commit. When Continue observes a newer
branch tip, the run flow offers the remembered commit, the latest commit, or a
caller-supplied exact commit; remembered/latest choices show short GitHub-linked
commit IDs when the normalized repository is hosted on GitHub.
Each validated model diagnosis is retained first as an immutable raw finding
bound to an exact repository commit, primary file blob and size, opaque AI
context snapshot, workflow assignment, and contributing model. Campaign-level
deduplication promotes completed raw records into diagnosis-sealed findings;
each promoted finding keeps its ordered raw-source provenance. A separate
canonical repository finding joins the same causal defect across commits while
retaining every occurrence, path and symbol history, lifecycle transition,
issue association, and validated resolution. A finding has at most one
canonical issue-preview or issue association. The review-owned Findings surface
lists only completed deduplicated findings, exposes their raw-source counts and
representative diagnosis for discussion, and links to both paged raw evidence
and the canonical repository finding. Pending or failed raw records remain
inspectable through Raw findings and processing counters, never through the
main Findings collection. Repository-finding lifecycle, issue drafting,
publication, and existing-issue association remain available only from the
Repositories workspace.
Findings, Raw findings, canonical repository findings, and issue previews use
the shared server-query collection controller with List, Table, and Grid views.
Each nested collection keeps its own canonical query, explicit selection, view
preference, cursor paging, and restored scroll position independently from its
parent review or repository collection. The primary list and detail adapters
use the strict automation `/findings` resource; `/run-findings` remains a
deprecated compatibility resource for occurrence clients. Its detail route
accepts only a stored legacy `rfn_*` occurrence alias; any other resource ID
returns `404` instead of falling through to a broader detail dispatcher. The
deprecated collection likewise excludes modern `rdf_*` projections.
Model findings, issue previews, and candidate rankings are diagnosis-only: they
may describe a confirmed defect and its grounded evidence, but never a fix,
recommendation, remediation, workaround, patch, or test change.
The same model-output boundary applies to the generic code-review template and
PR-workspace review/completion findings; any later implementation or
user-directed discussion is a separate process rather than a field smuggled
through the finding.

New model-probe reports also retain a bounded diagnosis-only claim ledger for
each model: path, title, evidence, impact, supported/unsupported judge
disposition, and rationale. The report exposes those claims in a collapsed
drill-down beside its charts and textual analysis. It retains no prompts,
repository source payloads, fixes, or provider payloads; historical
aggregate-only reports remain readable.

### Product Contract

Repository Reviews is designed for one trusted operator of a launcher
workspace. Its outcome is an auditable diagnosis of concrete defects in one
exact Git commit, durable evidence that can be resumed without repeating
confirmed work, and explicit issue/lifecycle actions over the resulting
repository ledger. It does not implement tenant isolation or an organizational
approval system.

The deterministic support envelope is:

- a Git repository identity that the launcher can acquire and an inventory of
  at most 100,000 files;
- the structured source categories `hotpath-code`, `code`, `test`, and
  `bench-test`; binary and individually oversized files are terminally marked
  unsupported, while aggregate-limited or failed work remains retryable;
- profile batches of 1–100,000 distinct files, provider evidence groups of at
  most 524,288 bytes, 1–64 review workers, and 60–86,400 second
  minute-aligned assignment deadlines; and
- schema, identity, provenance, concurrency, privacy, resource-bound, retry,
  and external-side-effect safety as release-blocking acceptance properties.

Repository languages are supported through the inventory categories and model
evidence boundary rather than a language-specific completeness guarantee.
False-positive rate, recall, semantic deduplication quality, and model
usefulness are measured through Repository Model Evaluations; they are not
nondeterministic merge gates and the feature promises no fixed quality
percentage for a model or language.
Time to first evidence is checkpoint-based rather than a provider-latency SLA:
committed raw evidence is readable immediately, and a completed `rdf_*` becomes
visible in Findings as soon as its deduplication checkpoint commits, without
waiting for the campaign to finish.

Explicit non-goals are multi-user RBAC, scheduled or push-triggered reviews,
notifications, ledger export, automatic retention expiry, code modification,
fix generation, and automatic guard recovery. Retention is indefinite until a
trusted operator explicitly purges history or removes the repository
assignment.

### Canonical Resource Glossary

| Term | Stable ID | Canonical meaning | Primary automation resource |
| --- | --- | --- | --- |
| Raw finding | `rrw_*` | One immutable confirmed diagnosis emitted by one successful assignment, with exact context/checkpoint/model/account provenance and deduplication processing state. | `/raw-findings`; also `/findings-processing` when queried from the whole repository ledger |
| Deduplicated finding | `rdf_*` | One diagnosis-sealed, current-campaign aggregate of one or more ordered raw findings judged to describe the same defect. | `/findings`; nested `/findings/{id}/sources` pages its `rrw_*` provenance |
| Repository finding | `rrf_*` | The canonical cross-commit identity for one causal defect, including all occurrences, path/symbol history, lifecycle, issue association, and resolution history. | `/repository-findings` |
| Legacy occurrence alias | `rfn_*` | A compatibility identity that resolves to its admitted `rrw_*` raw source; it is never a fourth canonical finding layer. | Deprecated `/run-findings` and canonical raw-detail redirects |

Unqualified **Findings** and the strict automation `/findings` API always mean
completed current-campaign `rdf_*` records. **Raw findings** means campaign
evidence, including pending or failed processing. **Findings processing** means
the canonical repository-wide processing view over `rrw_*` records. **Repository
findings** means cross-commit `rrf_*` records. Deprecated `/run-findings`
preserves occurrence clients only and must not define new UI semantics.

## Reconstruction Notes

- Similarity target: recreate a launcher-owned repository pre-review control
  plane backed by bounded workflow batches and an independent immutable finding
  ledger, not a browser-only wrapper around the generic workflow form.
- Core types/functions: `repoaudit.Store`, `RepositoryReviewProfile`,
  `RepositoryReviewAutomation`, `Finding`, `IssueDraft`, the repository review
  controller, findings/issue-generation API handlers, protected publication and
  existing-issue provider boundaries, workflow `review.repository` native
  functions, the built-in repository-bug-finder template, and the collection,
  detail, findings, finding, and issue-preview routes.
- Runtime ordering: validate and persist a reusable profile; atomically assign
  it to one branch-bound repository configuration; materialize the current
  profile version and execution account; reserve one durable workflow run;
  immediately before each managed worker takes a task, load exact cumulative
  usage, reserve projected in-flight usage, refresh the selected account's
  referenced limit telemetry, and evaluate the guard expression; then execute
  and account provider calls and verify the record/no-op checkpoint. A preview
  request then reserves each still-open unassociated finding under one
  generation ID before making a fixed isolated issue-writer call, persists only
  validated structured output and safe provenance, and publishes only an
  explicitly confirmed subset. Recovery reconciles orphaned review state,
  generation reservations, and ambiguous publications without repeating an
  unsafe external effect.
- Non-obvious constraints: repository bytes remain immutable no-tool evidence;
  one workspace-wide controller lease prevents competing launchers; eight
  workers may run concurrently, while atomic projected-usage reservations stop
  simultaneous task pickups from all passing the same token/cost predicate;
  internal coverage sketches and raw provider responses never enter API
  responses or durable state; new previews are one finding each while legacy
  grouped drafts remain visible; issue publication and existing-issue
  validation use protected GitHub boundaries.
- Candidate-scope failures expose bounded shape-only diagnostics: a
  Git-object-shaped planner value is distinguished from an unknown frozen
  candidate missing from the rebuilt commit-bound catalog, while the
  model-provided value is never echoed into durable failure state. The review
  detail expands the legacy bare unknown-candidate failure into an actionable
  explanation and keeps Continue available from saved review state.

## Requirements

| ID                  | Level | Requirement                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      | Rationale                                                                                                                                                                                                                                                             |
| ------------------- | ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `FR-REPOREVIEW-001` | MUST  | Repository review state stores each validated finding with its commit SHA, primary file path/blob SHA/size, validation result, model contributors, observation count, and opaque context IDs; each referenced context stores its review profile hash and complete file-ref set.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  | Findings must be recoverable and attributable without guessing which source revision an AI saw.                                                                                                                                                                       |
| `FR-REPOREVIEW-002` | MUST  | `GET /api/repository-reviews` returns compact repository summaries. `GET /api/repository-reviews/{id}` independently paginates findings and issue drafts, returns only contexts referenced by that finding page, bounds run/unsupported projections, and omits internal checkpoint maps. Review occurrences are immutable; issue-draft and repository-finding lifecycle mutations use their documented version fence and return only the changed object plus a compact repository summary.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       | The frontend must recover large ledgers, reach older drafts, and reject stale edits without returning the whole internal state.                                                                                                                                       |
| `FR-REPOREVIEW-003` | MUST  | `/repository-reviews` is a standard compact collection of review summaries; each row and `/repository-reviews/:id` detail expose separate current-campaign Findings and Raw findings counts and stable links. The shared detail shell owns Start, safe Stop, Continue, Run again, live stage/progress/usage, commit selection, and bounded run history. `/repository-reviews/:id/findings` lists completed deduplicated findings, `/repository-reviews/:id/findings/:findingId` loads one representative sealed diagnosis with paged raw sources, and `/repository-reviews/:id/raw-findings` plus `/:sourceId` retain all raw evidence and processing failures. Canonical cross-commit aggregate lists and details remain directly loadable only below `/repository-reviews/repositories/:id/findings`. The review detail keeps Findings, Raw findings, and Issue previews enabled before, during, and after a run. | Raw evidence, deduplicated campaign findings, and repository lifecycle are different resources; stable route ownership prevents one from silently substituting another. |
| `FR-REPOREVIEW-004` | MUST  | The user can select one or many deduplicated review findings. Discuss creates one `reviewing` thread tagged with repository/review/finding/context identity and sends one bounded self-contained message per selected representative diagnosis; collectively those messages contain every selected finding, its raw-source count, and, for every representative retained context, its opaque ID, profile hash, model, and complete path/blob/size manifest before the thread opens. Every raw source remains independently inspectable rather than being copied into the discussion prompt. | Discussion must stay attached to the sealed representative diagnosis while preserving the cardinality and independent auditability of its raw evidence. |
| `FR-REPOREVIEW-005` | MUST  | Issue generation authority appears only on canonical repository-finding list and detail routes. It accepts at most 200 explicitly selected open, unassociated, non-provisional repository findings, anchors each action to one eligible immutable occurrence, reserves one canonical preview slot per action under a caller-visible generation ID, and runs at most four isolated issue-writer requests concurrently across launcher processes sharing the workspace. Run-finding list and detail routes never render Draft, Post, link, or discovery controls. Each successful request creates one durable preview for exactly one canonical finding action with a concise generated title, GitHub-flavored Markdown body, and labels; persists its resolved default or custom presentation instructions, mode, generator model/account, and generation ID; and never persists the raw provider response. Custom batch instructions may change presentation but cannot request fixes or unsupported facts. Completion or partial failure navigates to `/repository-reviews/:id/issues` filtered or highlighted by that generation ID. `/repository-reviews/:id/issues/:draftId` is a read-only rendered preview with provenance, finding links, regeneration, deletion, and publication actions; `/repository-reviews/:id/issues/:draftId/edit` version-fences title/body/label editing. A failed regeneration preserves the last good preview and its authoring provenance while retaining separate safe provenance for the failed attempt.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | An issue represents canonical repository state, while the immutable occurrence remains evidence rather than publication authority.                                                                                                                                    |
| `FR-REPOREVIEW-006` | MUST  | For a repository identity derived from the acquired checkout's exact GitHub origin as a normalized safe `owner/repo`, the user may publish any explicitly selected subset of publishable previews. One authoritative publication-eligibility evaluation is reused by issue-detail capabilities, issue-collection `publishable`, API and protected-gateway preflight, and the atomic store claim. It returns `can_publish` plus an ordered safe `publish_blockers` projection whose entries contain only `code`, aggregate `count`, and displayable `message`. Stable blocker codes are `repository_not_github`, `preview_not_canonical`, `origin_not_publishable`, `state_not_publishable`, `finding_missing`, `finding_status_unresolved`, `duplicate_review_required`, `issue_association_conflict`, `historical_merge_in_progress`, and fallback `finding_not_publishable`; grouped previews aggregate every affected finding instead of hiding later blockers. Eligibility requires a canonical AI-generated or legacy preview in `editing`, `publishing`, or `unknown`, publishable linked-finding state, no unresolved duplicate decision or issue conflict, and no active historical merge. `publishing` and `unknown` remain eligible only for idempotent reconciliation; `posted` opens its validated external issue instead of offering Post. Each initial publication durably changes that preview from `editing` to `publishing` before the protected gateway call, freezes the exact title/body/labels, searches its stable marker before create, and reports per-preview success, safe failure, or `unknown` reconciliation state. A recovered marker completes publication; an absent marker for `publishing` or `unknown` never causes an unsafe duplicate create. Only successful publication or a separately validated existing-issue link stores the provider issue ID/URL and changes the canonical association to `posted`; there is no untracked Mark posted transition. Created issues remain permanently associated. Local or non-GitHub repositories retain previews but expose no publish, search, or link action. | External publication must use a reviewable durable payload, one consistent eligibility decision, a verified bounded destination, and idempotent recovery rather than allowing a capability projection or gateway preflight to disagree with the final store guard. |
| `FR-REPOREVIEW-007` | MUST  | Review checkpoint CAS uses a dedicated `review_version`, while repository-finding lifecycle, issue drafting, and publication use aggregate/draft versions. Immutable review occurrences have no lifecycle mutation. UI mutations during a long AI run therefore merge with later occurrences; only a newer review checkpoint invalidates an older review plan. Repository writes use OS locks on Unix and Windows.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               | A user discussing or drafting an issue must not make completed AI work fail, and launcher/gateway processes must not overwrite one another.                                                                                                                           |
| `FR-REPOREVIEW-008` | MUST  | A reusable versioned review profile can be created before any repository assignment, ledger, finding, or model probe. It stores a name, focus, exactly one safe reviewer alias, one optional issue-writer alias governed by `FR-REPOREVIEW-018`, one optional execution `account_ref` (blank means the runtime default), force mode, bounded files-per-batch/content-bytes-per-batch/parallel settings, automatic batch continuation, scope, and one bounded `guard_expression`. For an actual repository review, the files value bounds both pending batch admission and the related-file group sent to a provider child, while the content value bounds that real provider-call group; immutable hydration retains its independent safe per-file ceiling. A model probe may read the latest profile and freeze its ID/version/name, effective account, reviewer, focus/scope, file/content maxima, and parallelism, but it does not assign or mutate the profile and does not inherit actual-review-only force, continuation, task-guard, or issue-publication authority. The profile never stores model prices, account lists, polling intervals, output-token estimates, or separate token/cost/quota threshold controls. Profile create/update/delete uses cross-process locking, atomic `0600` files, validation, and version CAS; legacy budget fields migrate to an equivalent guard expression on durable load, deletion is rejected while assigned, and mutation is rejected while an assigned review is running or stopping. Profile mutation clients send only writable configuration fields, plus `expected_version` for updates, and never echo response metadata such as `schema_version`, ID, version, or timestamps.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            | Review behavior should be reusable across production review and controlled model experiments while account routing, mutation, and actual-review admission authority remain explicit.                                                                                  |
| `FR-REPOREVIEW-009` | MUST  | A separate versioned repository configuration assigns one profile to one normalized repository and an optional branch. A repository has at most one configuration; one profile may serve many repositories. The authenticated run API and `/repository-reviews/:id` detail flow expose Start, safe Stop/Pause, Continue/Resume, Restart/Run again, and Delete; the collection remains summary-only. Starting snapshots the latest assigned profile version, resolves the configured branch or advertised default to a canonical full commit SHA, and atomically persists that SHA with the durable workflow run ID before work begins. Every automatic continuation batch receives that exact SHA. A paused or failed continuation resolves the current branch tip; when it differs from the remembered SHA, execution requires an explicit remembered/latest/custom full-SHA choice. Resume appends a workflow run to the same campaign and preserves its campaign start, run history, durable checkpoints, cumulative usage/cost, and progress only when both the assigned profile snapshot and chosen exact commit are unchanged; a changed profile, changed commit, or explicit Restart starts a new campaign and resets campaign progress and accounting. The detail UI automatically shows remembered and latest short IDs with canonical GitHub commit links when available, plus the custom option. Active state exposes the remembered SHA, running/stopping/paused/completed/failed status, current workflow stage, file-based resolved/total progress, bounded-batch telemetry, run history, timestamps, and a structured pause reason/detail. Frozen-scope progress uses selected files as its total and remaining files to derive a clamped resolved count; legacy progress uses fully reviewed plus unsupported files, never finding occurrences or completed batches. A verified checkpoint with remaining work that resolves zero files pauses immediately with `no_progress`, preserves the completed-batch telemetry, and requires explicit Resume before another attempt. Safe stop tolerates progress-version drift only while its submitted workflow run ID still matches, stops a queued automatic handoff, prevents new admission, and lets the current batch record its durable checkpoint. | Repository assignment, profile policy, exact source identity, and execution lifecycle require distinct authority and concurrency fences. |
| `FR-REPOREVIEW-010` | MUST  | Every non-nil provider response reports actual model-attributed prompt, completion, cached, and total token usage, including fallbacks and structured repairs. At every Start, Resume, or Restart the server refreshes conservative prices for the selected account/model route and stores only numeric per-alias accounting snapshots on the automation. The controller durably aggregates actual usage and estimated USD. At task pickup it atomically includes all in-flight projected prompt/output tokens and price-known cost reservations in `spent.tokens.*` and `spend.total.*`; completion releases the reservation while provider-reported usage remains. Unknown price makes monetary guard fields unknown rather than zero.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         | Concurrent workers must not all pass the same guard from stale counters, and accounting must remain honest without editable price metadata or reset controls.                                                                                                         |
| `FR-REPOREVIEW-011` | MUST  | The task-admission guard is an optional, bounded, JQL-like boolean expression with case-insensitive `AND`, `OR`, `NOT`, parentheses, numeric/boolean/string literals, and `=`, `==`, `!=`, `<`, `<=`, `>`, `>=`. Its only variable families are `account.limits.*`, `spent.tokens.*`, and `spend.total.*`. It is parsed at profile mutation/load and evaluated exactly once after a managed worker claims its next task but before provider dispatch. Account-limit fields come only from the profile's selected account (router members aggregate conservatively). A false, unknown, malformed, or telemetry-error result denies that task and latches a safe `guard_expression` pause; already admitted tasks may finish. There is no account-target list, background quota polling interval, or automatic guard recovery. The profile editor autocompletes valid fields (including common and current selected-account windows), operators, keywords, and literals at the caret; its full expression reference opens from the help control beside the field label.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            | One expression can combine account health, real consumption, and cost while preserving a clear execution-account boundary and deterministic fail-closed admission.                                                                                                    |
| `FR-REPOREVIEW-012` | MUST  | The profile UI presents safe reviewer aliases and execution-account routes but exposes no model-price fields or price override controls. Account-router availability expands every reachable concrete or credential-backed member instead of treating the router ID as a concrete account. Individually selectable credential accounts expose execution availability separately from limit-telemetry status: invalid or expired credentials are disabled, while a telemetry-only error does not disable an otherwise executable credential. The UI fails closed unless an alias is globally safe and explicitly compatible with the selected available account, preserves a compatible model when the account changes, selects the first safe replacement otherwise, invalidates cached options after a failed refresh, and explains globally blocked, account-incompatible, stale, missing, and empty option states. Pricing authority remains in central model/account configuration. Runtime retains request/failure counts, actual tokens, estimated USD, compact approximate reviewed-file coverage, and finding yield without claiming unknown price as zero. Comparative model testing, scoring, context-size ceilings, and cached-weighted token efficiency remain exclusively in the separate model-review probe domain. That domain may list compatible safe profiles and snapshot one at Run, but it never starts or mutates an actual review, profile assignment, or finding ledger. Agentic CLI providers remain unavailable for immutable repository review.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       | Operators need honest centrally governed actual-run economics without conflating exploratory model comparison with production findings.                                                                                                                               |
| `FR-REPOREVIEW-013` | MUST  | On launcher startup, the durable controller reconciles an automation left running without a local executor, marks the orphaned workflow run canceled, and records a `service_restart` pause. Resume is explicit and re-evaluates the task guard at the next worker pickup. Controller shutdown cancels in-process execution and never creates a new batch after its context is closed.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           | Process restarts must not leave phantom running state, duplicate work, or an implicit polling/restart loop.                                                                                                                                                           |
| `FR-REPOREVIEW-014` | MUST  | Each review profile persists a bounded scope policy with one or more selectable inventory code types (`hotpath-code`, `code`, `test`, `bench-test`), canonical repository-relative include and exclude folder prefixes, and optional free-text guidance. The default selects normal production code (`hotpath-code` and `code`); exclusions always win. A generated repository preflight is commit-bound and persists bounded policy/plan hashes, summary, rationale, warnings, aggregate file counts, and a server-internal canonical planner selection. During initial planning only, native validation strictly decodes the planner's exact and hotpath candidate arrays, canonicalizes their IDs, retains known commit-bound IDs, discards syntactically valid unknown IDs, and records only their bounded counts in a native warning whose prefix is reserved from planner output. If no known exact ID remains, selection falls back to the trusted normalized prefixes and hard scope; malformed, duplicate, or wrong-type IDs still fail. The sanitized selection is the only selection frozen, and new plan hashes bind its warnings; legacy warning-unbound hashes remain readable only when they contain no reserved native warning. Automatic continuation revalidates and reuses that exact selection and plan without calling the planner again, and treats any unknown frozen ID or changed native warning as corruption, so checkpoint identity and the authoritative file universe remain stable across the campaign. Changing the commit, assigned profile, or execution scope clears the frozen selection and plan.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           | Operators must be able to express reusable, reproducible review intent without allowing unsafe paths, unbounded manifests, stale commit summaries, or an exclusion to be silently re-included.                                                                        |
| `FR-REPOREVIEW-015` | MUST  | Repository configuration accepts only an optional branch name. Blank resolves through the acquired repository's advertised default branch; `HEAD`, commit hashes, tags, full refs, revision expressions, URLs, query/fragment forms, and invalid Git branch names are rejected before admission. The launcher exposes Review runs, Repositories, and Profiles destinations; it removes the separate Results navigation item, redirects legacy `/repository-reviews/results` URLs to the review collection while preserving valid `q`/`view`, and places run occurrences under Review runs while repository aggregates and their issue/lifecycle actions use nested routes below Repositories. Repositories and Profiles are independent standard server-query/paged list-table-grid collections with stable `/repositories/new`, `/repositories/:id`, `/repositories/:id/edit`, `/profiles/new`, `/profiles/:profileID`, and `/profiles/:profileID/edit` routes that preserve collection state. Every repository item and configuration detail exposes its repository-owned findings route. A legacy run-findings or report URL with `scope=all` redirects to the repository-finding subtree while preserving valid query and view state and normalizing offset paging to the first cursor page. Basic profile identity, execution account, reviewer, issue writer, issue prompt, focus, and Scope are immediately visible in the profile editor; account precedes both model selectors because it constrains their availability. Sizing and task-admission guard remain in Advanced, collapsed by default, and preserve their draft while closed. New profiles default to eight configurable parallel review workers; adding a guard does not silently serialize them because admission uses atomic in-flight reservations. Internal output-token estimation is server-owned and absent from profile/API controls.                                                                                                                                                                                                                                                                                                                                              | Repository review should follow a predictable branch, keep core review intent visible, and give configuration, run evidence, and canonical repository state unambiguous navigation ownership.                                                                         |
| `FR-REPOREVIEW-016` | MUST  | Review, issue-writing, and issue-candidate-ranking calls use a fixed diagnosis-only system policy that profile focus, scope guidance, custom presentation instructions, repository content, candidate issue content, model output, and other user-controlled text cannot override. Every issue-writing and ranking request is private, ephemeral, no-history, no-cache, no-tools, and structured. Finding output contains only a factual summary, exact reviewed paths, confirmed findings, and evidence limitations. Each finding and issue preview contains only severity or label, a concise title, smallest stable symbol, exact path and optional line, factual failure mechanism/trigger, source-grounded evidence, observable impact, validation already performed, and exact commit/blob provenance. It never supplies or implies a fix, recommendation, remediation, mitigation, workaround, patch, replacement code, design/configuration/test change, next-step advice, or unsupported fact. Structured schemas and native decoders reject extra fields, including legacy `recommendation` and top-level `tests`; only validated projections and safe bounded errors are persisted, never raw provider responses. Model probes apply the same policy, and diagnostic utility means locatable, reproducible, verifiable, and prioritizable without rewarding remediation.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | Finding and issue quality must be comparable independently from a model's ability to invent patches, and no editable or repository-supplied text may weaken the reporting boundary.                                                                                   |
| `FR-REPOREVIEW-017` | MUST  | `/repository-reviews/:id/findings` and its strict automation API are directly loadable while review is idle, active, paused, failed, or complete; they show completed current-campaign `rdf_*` findings only, render an empty in-progress state before the first completed checkpoint, poll only while relevant work is active, and offer selection, discussion, detail, raw-source navigation, and canonical repository-finding navigation. Findings, Raw findings, Findings processing, repository findings, and issue previews are independent List/Table/Grid collections backed by typed queries and query-bound cursors. The repository-finding collection identifies each item as `N occurrences across M commits`; table columns are Identity, Severity, Occurrences, Finding state, Issue, Resolution check, and Updated. It omits normal `new`/`known` badges and shows only Duplicate review, Issue conflict, and Fix check failed attention. Lifecycle remains independently visible. Human labels never rewrite raw `match`, `issue`, or `validation` values. Selection covers explicit loaded items only and respects action bounds. `/report` redirects to the appropriate canonical collection. Each child collection preserves its own query, view, selection, cursor position, and scroll through Browser Back. | Operators need unambiguous finding layers and concise lifecycle information while every decision, conflict, or retry remains visible. |
| `FR-REPOREVIEW-018` | MUST  | A review profile stores optional `issue_writer_model` and editable `issue_prompt`. Blank writer means the reviewer; an explicit writer must be a passive-provider alias available on the selected account. Profile mutation validates reviewer and writer against the same effective account without permitting agentic CLI providers. Review execution snapshots its admitted policy, while Draft, Post, and candidate ranking resolve the currently assigned profile at action time and freeze profile ID/version, resolved prompt, writer, and account into the reservation or generated draft. A retry reuses that frozen attempt provenance.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                | Issue text and matching need reproducible provenance while intentional new actions follow the operator's current assigned profile.                                                                                                                                    |
| `FR-REPOREVIEW-019` | MUST  | Each finding stores at most one canonical issue-draft reference. Each draft stores origin (`ai_generated`, `linked`, `discovered`, or `legacy`), one finding for new previews, generation ID, resolved instructions and mode, generator model/account, safe generation error, version, external identity, and state `generating`, `failed`, `editing`, `publishing`, `unknown`, or `posted`. Only an open, unassociated finding may reserve generation or linking; the reservation occurs before the isolated call and retry with the same generation ID is idempotent. A workspace-wide per-attempt OS lock prevents two launcher processes from dispatching the same durable reservation and releases automatically after process failure. Failed initial generations remain retryable and deletable; deleting an `editing` or `failed` unpublished preview clears its finding association, while `publishing`, `unknown`, and `posted` previews cannot be deleted. Regeneration stores active-attempt provenance separately, promotes it only with successful output, and preserves the last good preview and its provenance on failure. Created issues cannot be unlinked; manually linked or discovered issues may be unlinked or replaced only after explicit confirmation, and one existing issue may be linked to multiple findings. Durable load deterministically backfills legacy associations by preferring `posted`, then `publishing`/`unknown`, then the newest `editing` draft; grouped and duplicate legacy drafts remain visible, while noncanonical conflicts are read-only and cannot publish. Repository-finding issue labels are No issue or preview for `none`, Saved preview · Not on GitHub for `draft`, GitHub issue open for `open`, GitHub issue closed for `closed`, and GitHub issue state unknown for `unknown`. An internal preview uses a document icon and Review preview action. A GitHub icon or Open GitHub issue action appears only when the record has a validated external GitHub URL; an internal preview never borrows GitHub identity merely because it may later be posted. These labels and icons are presentation only and preserve raw issue/draft origin and state values. | A canonical association and explicit state machine prevent duplicate generation/publication while presentation makes the local-preview versus external-GitHub boundary unmistakable. |
| `FR-REPOREVIEW-020` | MUST  | For a canonical GitHub repository, the link route accepts a same-repository issue URL and offers discovery. Searches use causal hints, symbols, anchors, path history, and title, merge/deduplicate at most 50 open or closed issues, and rank at most 10. Automatic discovery uses the exact threshold and re-fetch policy in `FR-REPOREVIEW-026`; every other candidate requires manual selection. Manual links are `linked`; malformed, deleted, or cross-repository results are rejected. Local and non-GitHub repositories expose no discovery or link action. | Bounded ranking can reduce duplicate issues without accepting an unverified or irreversible association. |
| `FR-REPOREVIEW-021` | MUST  | Newly generated review findings include closed, required `match_hints` and `fix_effort` objects. Match hints identify component, operation, failure mode, trigger, violated invariant, observable outcome, directly participating stable symbols, source identifiers/fields/keys/literals, and facts that distinguish nearby defects; they describe identity rather than remediation. Quick containment and best-quality correction estimates contain bounded minimum/maximum changed LOC, a class consistent with the documented LOC bands, and diagnosis-only rationale. Existing findings migrate with unknown hints and effort and are never re-reviewed solely for enrichment.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | Cross-commit identity needs causal evidence and honest sizing without allowing the finder to smuggle in a patch design.                                                                                                                                               |
| `FR-REPOREVIEW-022` | MUST  | The repository ledger atomically stores immutable review-finding occurrences, stable `rrf_` repository findings, and stable internal association jobs derived from review-finding IDs. Startup creates exactly one job for every unassociated occurrence, returns orphaned `running` jobs to `pending`, and atomically commits association plus job completion. Automatic processing stops after three failed attempts instead of dispatching indefinitely. An authenticated, bounded selection of at most 200 occurrence IDs may reset eligible failed work and request another processing cycle without mutating occurrence evidence or manually choosing an aggregate; pending or active work cannot be reset early. Reprocessing is idempotent. A repository finding stores canonical diagnosis/hints, occurrence and unique commit IDs, path/symbol history, known/new/provisional match state, lifecycle, issue and possible-duplicate associations, resolution history, version, and timestamps.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          | Crash recovery needs durable association work, while a broken model response must not cause an unbounded provider-call loop hidden from the operator.                                                                                                                 |
| `FR-REPOREVIEW-023` | MUST  | Internal association uses exact same-commit fingerprints, deterministic prefilter/conflict checks and thresholds, then an in-memory BM25 index over title and causal identity fields to retrieve at most ten candidates. Remaining ambiguity is adjudicated by an isolated private no-history/no-cache/no-tools reviewer request using opaque candidate IDs and the job's profile/model/account snapshot. Every adjudicated conflict has one closed field classification: `severity`, `title_wording`, `fix_effort`, and `lifecycle_status` are non-blocking, while causal identity fields, location/path, symbol, evidence, impact, validation content, and fallback `other` are blocking. Unknown, malformed, misaligned, or legacy missing classifications fail closed as blocking. Only `same` at confidence at least 0.90 with no blocking conflict may auto-associate during initial mapping; low-confidence `same`, any blocking conflict, and `uncertain` create a provisional finding with visible possible duplicates, while `related` never merges. Existing provisional findings are not reevaluated. Provisional repository findings cannot draft, link, discover, or post issues until the user resolves them on their repository-owned detail route. The provisional decision and Issue sections appear before long occurrence provenance and resolution history. Each duplicate-candidate panel shows the candidate title, latest path/symbol location, severity, exact occurrence count, exact distinct-commit count, and the retained duplicate-decision evidence: relation/confidence, explanation, and matching or conflicting anchors. When a blocking conflict makes a finding provisional, every conflict's original text, including non-blocking conflicts, remains in original order in the existing Conflicting anchors display without classification badges, grouping, labels, filtering, or style differences. View candidate opens the candidate's stable repository-finding detail for full diagnosis. The negative decision is labeled Keep separate, preserving the raw `distinct` request value. Merge with candidate always opens an explicit confirmation before the version-fenced merge request; a provisional finding is never merged by displayed confidence or by rendering the panel. | Renames and refactors should match while defects sharing only a file, function, or symptom remain distinct, and every destructive identity merge remains an informed, explicit user decision. |
| `FR-REPOREVIEW-024` | MUST  | Every occurrence persists the resolved target branch, advertised default branch, and `target_is_default`. A default-branch occurrence may create or join a repository finding. A non-default occurrence may join an already-known finding but cannot create one. Legacy occurrences use the same queue, with reachability from the current default branch verified before creation. A new occurrence after confirmed resolution changes lifecycle to `regressed` while retaining prior resolution history.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       | Canonical repository state must describe defects on the repository's shipped line without discarding useful corroboration from other branches.                                                                                                                        |
| `FR-REPOREVIEW-025` | MUST  | Profile schema stores an editable bounded `issue_prompt`; legacy profiles migrate to the built-in default. The private issue-writer policy remains server-owned. Draft and direct Post resolve the currently assigned profile, and persist profile ID/version, resolved prompt, model, and account on the generation reservation/draft. Draft creates an editable preview with optional one-off presentation instructions. Post publishes saved draft content unchanged, or, when no draft exists, generates and posts immediately from the current profile. The explicit Post click is publication authority and requires no second confirmation. Existing publication marker, `publishing`, `unknown`, and reconciliation protections remain unchanged.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        | Presentation policy should be reusable and attributable while explicit user publication remains safe and free of stale-profile surprises.                                                                                                                             |
| `FR-REPOREVIEW-026` | MUST  | GitHub discovery derives bounded searches from causal hints, stable symbols, anchors, path history, and title. Only the first ranked candidate with score ≥95, at least four matching anchors, and zero conflicting anchors may auto-link, and only after exact same-repository re-fetch; the stored `discovered` origin is reversible. Lower-confidence candidates remain selectable and a same-repository issue URL may be entered manually. Issue snapshots refresh after a 15-minute TTL when finding or Resolution check UI opens, before a Resolution check, or on explicit Sync; there is no background polling. Closed issues move lifecycle to `resolution_pending`, reopened issues return it to `open`, and closure alone never proves resolution. | Issue state and code resolution are independent facts, and automatic discovery must be both verified and reversible. |
| `FR-REPOREVIEW-027` | MUST  | Resolution validation uses a restart-safe queue for one or up to 50 selected repository findings, with at most four concurrent validators. The server gathers at most 200 default-branch commits touching known paths/symbols since the last occurrence, BM25-ranks them, and supplies at most eight bounded diffs/current-source records. The isolated validator may select only supplied commit IDs; the server verifies ancestry. A confirmed result records fix commit/time, validation time, and the first chronological semantic-version tag containing it; other outcomes are not-fixed, inconclusive, or failed. Repository-level UI copy calls this process Resolution check, never Validation. Raw validation states render as Not checked, Queued, Checking, Fix confirmed, Still present, Inconclusive, and Check failed for `not_requested`, `pending`, `running`, `confirmed`, `not_fixed`, `inconclusive`, and `failed`. The state-aware action is Check for fix when a check may be requested, Retry fix check after `failed`, disabled Fix check queued while `pending`, or disabled Checking for fix… while `running`. Lifecycle, issue state, resolution-check state, found commits, fix commit/date, and first containing version remain independent. The immutable occurrence diagnosis `validation.status`, `validation.summary`, and `validation.checks` retain their Validation / Validation already performed meaning and are never relabeled as a resolution check. API field names, query field `validation`, and enum values remain unchanged. | A closed issue is not proof of a fix; validation must be bounded and reproducible while operators can distinguish original diagnosis validation from a later check for resolution. |
| `FR-REPOREVIEW-028` | MUST  | Repository identity resolution is centralized in `pkg/repoaudit` and shared by controller, API, and gateway for GitHub URL, SSH/SCP, owner/repository shorthand, absolute local path, and run-ID fallback. Automation progress exposes current-campaign `raw_findings` and `deduplicated_findings`; deprecated `findings` is an exact alias of the deduplicated count, and `raw_findings` is a sortable review-collection field. Canonical deduplicated-finding, raw-finding, repository-finding, and issue-preview UI/API routes use separate automation-scoped query/cursor collection resources. `/report` remains a compatibility surface and legacy `scope=all` UI URLs redirect while preserving valid query and view state. Legacy positive UI offsets normalize to the first canonical cursor page; `scope` and `offset` are removed with history replacement. User-visible copy uses Findings, Raw findings, and repository findings; it never uses Report, review report, durable findings, or internal association-job terminology. | All entry points must count and display the same current campaign while route and copy make raw evidence, deduplicated review findings, and cross-commit aggregates unmistakable. |
| `FR-REPOREVIEW-029` | MUST  | The ledger has one controller-authorized opaque current campaign installed under review-version CAS and bound to the exact commit, inventory, frozen profile, full-FileRef scope digest, selected-file count, and required-assignment count. Native checkpoints derive monotonic inspected/completed/unsupported coverage only from matching same-child acknowledgements and provenance; callers cannot project credit or reuse a campaign ID as authority. Recovery promotes evidence to exact only from digest-valid matching records, otherwise reports unknown. Private campaign/scope/checkpoint identities are omitted from API/gateway projections and all envelopes are bounded. | Durable metrics must distinguish partial inspection from fully completed files without trusting caller counts or reconstructed guesses. |
| `FR-REPOREVIEW-030` | MUST  | Each campaign freezes an ordered catalog over `correctness_state`, `security_trust`, `concurrency_recovery`, and `integration_validation`, binding stable assignment identity to reviewer, prompt revision, and profile hash. Planning emits only missing file-assignment pairs within the profile batch bound; a durable active-run reservation precedes dispatch. Confirmed attribution is append-only, exact, idempotent, and separately reports assignment-pair progress from fully reviewed files. Legacy credit is reusable only when every recovered campaign/commit/inventory/profile/reviewer/file fact matches durable authority. | Confirmed focus/reviewer work must survive sibling failure and restart without fabricating credit or repeating a proven pair. |
| `FR-REPOREVIEW-031` | MUST  | Before a successful child callback, every confirmed diagnosis is atomically stored as immutable `rrw_*` evidence with context, checkpoint, campaign/path/blob/symbol bucket, insertion order, concrete model/configured alias/account provenance, counters, and one durable deduplication job. Historical ambiguity remains empty rather than invented. Bounded private FIFO deduplication creates a diagnosis-verbatim `rdf_*` or attaches ordered raw provenance after rechecking its lease/universe/version fences. Three failures leave safe readable evidence and explicit retry authority. | Concurrent workers must not publish duplicate campaign diagnoses or hide raw evidence needed to audit the decision. |
| `FR-REPOREVIEW-032` | MUST  | Migration admits historical campaigns oldest-first under a frozen assigned profile and explicit `setup`, `processing`, or `merge` checkpoints while live ingestion continues. Compatible Resume preserves completed raw/`rdf_*` identities and resets only failed work; missing exact evidence fails closed, and profile/campaign/bucket drift returns structured restart-required conflict without mutation. Only confirmed Restart resets the affected dependency closure. Atomic merge waits for issue/mapping/Resolution-check work to quiesce, retains the earliest repository identity and all compatible histories, and records issue conflicts without GitHub effects. | Historical consolidation must resume safely and restart only incompatible work without stopping live review or causing external effects. |
| `FR-REPOREVIEW-033` | MUST  | Each automation exposes one health projection: run counts use selected-campaign membership; repository and processing counts use the canonical ledger; `unrepresented` is the direct pending+processing+failed sum. Historical state is normalized to `not_required|pending|replaying|merging|failed|completed` with explicit retryability. Findings processing is a typed cursor collection defaulting to `ALL ORDER BY updated DESC`, exposes safe source/detail state, allows selection only for failed sources, and polls only while work is active. Review and repository surfaces separate Run findings, Repository findings, and Findings processing and never render unavailable health as zero. | Operators need one honest coverage view and a bounded path to every failure. |

The following atomic rows are authoritative for the high-risk behavior formerly
combined in `FR-REPOREVIEW-017` and `FR-REPOREVIEW-029` through
`FR-REPOREVIEW-033`. Those older IDs retain their collection-presentation,
checkpoint-foundation, ingestion, replay, and health compatibility guarantees;
when language overlaps, the trigger/output/state/failure split below controls.
`FR-REPOREVIEW-017` is limited to collection navigation and presentation and
cannot change strict `/findings` from `rdf_*` to occurrence records;
deprecated occurrence semantics belong to `/run-findings` only.
`FR-REPOREVIEW-029` is limited to checkpoint/campaign authority,
`FR-REPOREVIEW-030` to assignment attribution, `FR-REPOREVIEW-031` to raw
admission/native deduplication, `FR-REPOREVIEW-032` to historical replay, and
`FR-REPOREVIEW-033` to health/processing presentation and retry.

| ID | Level | Trigger/Input | Required Output | State Mutation | Failure/Edge | Rationale |
| --- | --- | --- | --- | --- | --- | --- |
| `FR-REPOREVIEW-034` | MUST | A client reads Findings, Raw findings, Findings processing, repository findings, or a legacy occurrence URL. | The server and UI use the `rrw_*` → `rdf_*` → `rrf_*` taxonomy and `/findings` returns completed current-campaign `rdf_*` records only. | Reads do not promote or remap evidence; `rfn_*` aliases resolve to canonical `rrw_*` identities. | Pending/failed raw records never fall through into `/findings`; repository-wide processing and cross-commit aggregates never masquerade as campaign findings; `/run-findings` lists only stored `rfn_*` aliases and its detail returns `404` for any non-alias ID. | Every diagnosis layer must have one stable meaning. |
| `FR-REPOREVIEW-035` | MUST | Start, Resume/Continue, automatic continuation, or Restart selects a profile snapshot and exact commit. | Resume preserves the campaign only when both the frozen profile snapshot and exact commit are unchanged; automatic continuation preserves both; changed profile, changed commit, or Restart creates a new campaign. | A preserved campaign keeps its ID, start, checkpoints, progress, usage, cost, and history; a new campaign resets campaign-scoped runtime/accounting while retaining repository history. | A moved branch requires explicit remembered/latest/custom commit choice. No action may silently apply prior checkpoint authority to a different profile or commit. | Campaign identity is the reuse boundary. |
| `FR-REPOREVIEW-036` | MUST | A managed assignment is reserved, dispatched, acknowledged, completed, retried, recovered, or times out. | Exact file/focus/reviewer attribution and per-focus assignment progress are returned without exposing masks, reservations, digests, or source bytes. | Valid same-child acknowledgements atomically persist assignment credit, raw evidence, checkpoint digest, and append-only attribution; duplicate identical callbacks are idempotent. | Malformed, timed-out, canceled, unknown, conflicting, or unacknowledged work receives no credit; recovery retries only missing or unknown pairs. | File completion must derive from proven assignment work. |
| `FR-REPOREVIEW-037` | MUST | Raw evidence is admitted, deduplicated, replayed from history, retried, or merged. | Immutable `rrw_*` evidence remains readable; completed work yields diagnosis-verbatim `rdf_*` findings with ordered provenance; compatible replay resumes checkpoints and incompatible replay requires explicit confirmed restart. | Jobs, leases, candidate snapshots, replay phases, and merge state are bounded and version-fenced; completed unrelated buckets remain unchanged. | Three native-processing failures retain a failed raw record; replay never invents campaign/account/model provenance, performs GitHub actions, or rewinds unrelated completed work. | Deduplication and migration must not hide or corrupt evidence. |
| `FR-REPOREVIEW-038` | MUST | A client reads finding health/processing or retries failed processing. | Health reports campaign-scoped run counts and repository-scoped aggregate/processing counts; processing defaults to `ALL ORDER BY updated DESC`; bulk retry accepts 1–200 unique failed `rrw_*` IDs and returns ordered successes plus safe per-ID failures. | Only eligible failed jobs are reset; successful and ineligible records remain unchanged. | Malformed/duplicate selection fails before mutation, cursor and legacy offset modes cannot mix, and polling stops at terminal state without granting retry authority. | Operators need an exact, bounded recovery path. |
| `FR-REPOREVIEW-039` | MUST | A review call starts or a later Draft, direct Post generation, regeneration, or candidate-ranking attempt starts. | Review calls report campaign-frozen profile/account/model provenance. Each later model-mediated issue attempt resolves the currently assigned profile/account once; generation/regeneration returns durable preview provenance, while ranking returns its generator model/account. | A generation reservation stores profile ID/version, prompt, requested alias, concrete model/account, and attempt identity without rewriting prior provenance; ranking is read-only unless a qualifying discovered link is stored; publication keeps the saved preview payload/provenance. | A later default-account/profile change never rewrites a campaign, admitted attempt, or saved preview; unavailable/incompatible current policy fails before a model call. Manual link/sync uses validated GitHub identity and does not invent model provenance. | Review authority and later model-mediated issue authority have different admission times. |
| `FR-REPOREVIEW-040` | MUST | Mapping, issue synchronization, a resolution check, regression observation, or manual dismiss/reopen requests a repository-finding lifecycle change. | The lifecycle follows the transition table in this document and is displayed independently from issue and resolution-check state. | Every accepted transition is finding-version-fenced and appended to durable issue/resolution/occurrence history as applicable. | Provisional findings, active checks, stale versions, disallowed manual transitions, and dismissal with an issue association fail without mutation. | Lifecycle must be explicit rather than inferred from one external signal. |
| `FR-REPOREVIEW-041` | MUST | A trusted operator purges history or removes a repository assignment with the expected automation version, primary repository version, opaque composite ledger fence, and exact normalized repository confirmation. | Purge returns a fresh idle configuration and removes every resolved repository-review ledger; removal deletes both configuration and all resolved ledgers. GitHub issues, profiles, discussion threads, and generic workflow-run records are untouched. | A `0600` restart-safe purge intent records the complete sorted ledger target/version set, serializes reset/removal, and is reconciled before workers start; history is otherwise retained indefinitely. | Any active review, deduplication, mapping, Resolution check, issue generation/publication, or historical consolidation blocks the action. Unreadable inventory fails closed with `retention_unavailable`; stale versions or composite fence, mismatched confirmation, unsafe intent files, or interruption fail/recover without resurrecting state. | Retention must be explicit, destructive, and recoverable across process loss. |
| `FR-REPOREVIEW-042` | MUST | A launcher client invokes any Repository Review HTTP route. | The authenticated single-operator API uses the routes, request bodies, response envelopes, limits, and safe error codes in the Repository Reviews API reference. | Mutations require `Content-Type: application/json`, accept no query parameters, and use the documented automation/finding/repository/ledger version fences; reads are side-effect free except documented TTL refresh endpoints. | Missing or invalid mutation media type, mutation unknown fields/query parameters, collection extra parameters, malformed IDs/cursors, oversized pages/selections, stale fences, and unavailable dependencies return bounded JSON errors without secrets or raw provider/repository data. | The public wire contract must be reconstructable and testable. |
| `FR-REPOREVIEW-043` | MUST | Release acceptance or product-scope review evaluates Repository Reviews. | Deterministic tests gate schema, exact-commit/campaign identity, provenance, privacy, resource limits, restart safety, accessibility, and external side-effect idempotency. Model evaluations report usefulness separately. | No product state changes merely because a quality evaluation runs. | No merge gate promises a model/language false-positive, recall, or deduplication percentage. RBAC, scheduling/on-push triggers, notifications, export, automatic expiry, and code fixes remain out of scope. | Product success and deterministic conformance must not be conflated. |

## Data And State Model

| State                        | Shape And Location                                                          | Contract                                                                                                                                                                                                                                                                                                                                                                                    |
| ---------------------------- | --------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Review profile               | `workspace/repository_reviews/profile_rrpf_*.json`                          | Reusable versioned `0600` behavior, one reviewer alias, optional issue-writer and deduplication aliases, editable issue prompt, deduplication threshold/candidate limit, optional execution account, scope, sizing, and one guard expression without repository or lifecycle state. Blank writer or deduplicator means reviewer.                                                                 |
| Repository configuration/run | `workspace/repository_reviews/automation_rra_*.json`                        | Unique normalized repository/profile assignment, optional branch, exact admitted `resolved_commit_sha`, campaign start and durable run IDs, materialized profile/account/reviewer/writer version, lifecycle, cumulative usage/cost, bounded run history, latest selected-account limit snapshots, and internal approximate coverage sketch.                                                 |
| Repository ledger            | `workspace/repository_reviews/repo_*.json` plus summary sidecar             | Exact commit/blob checkpoints, immutable raw and deduplicated review findings, deduplication/mapping/validation jobs, canonical repository findings, observations, contexts, completed review runs, replay/merge state, lifecycle/resolution history, canonical finding-to-draft references, and issue previews/drafts including safe provenance and errors.                              |
| Current campaign coverage    | `current_campaign` and `campaign_history` in the repository ledger          | Controller-authorized single-use identity; exact commit/inventory/profile/scope/required-assignment binding; a frozen deduplication profile/account/model snapshot; bounded monotonic per-path inspected, completed, and unsupported state; verified run/context/finding provenance; and a bounded recovery digest. An absent or inexact coverage record is unknown, never an inferred zero. |
| Issue-generation reservation | Canonical finding reference plus an issue draft in the repository ledger    | Durable reservation written before an isolated call. It binds one finding to one generation ID and survives retry/restart without retaining provider history or raw output. Per-attempt and four-slot OS locks make dispatch idempotent and bounded across launcher processes sharing the workspace.                                                                                        |
| Workflow run                 | `workspace/workflow_runs/<run-id>/`                                         | Generic durable job/step state and events; automation stores only bounded identities/progress and requires a verified `record` output or authoritative no-op checkpoint.                                                                                                                                                                                                                    |
| Controller lease             | `workspace/repository_reviews.controller.lock`                              | Non-blocking workspace-wide OS lock held for the launcher controller lifetime; never returned by an API.                                                                                                                                                                                                                                                                                    |
| Task guard                   | `budget.guard_expression`, `usage`, `estimated_cost_usd`                    | Resume preserves counters; Restart starts a new campaign epoch. In-flight reservations are process-local because their provider tasks cannot survive a launcher restart. Legacy threshold fields migrate to the expression on durable load.                                                                                                                                                 |
| Actual-model statistics      | `model_stats`, `model_coverage_sketches`                                    | Per-campaign request/failure/usage/cost/latency/finding counts and fixed-size approximate unique coverage for the assigned reviewer model. Sketches are persistence-only and removed from API projections. Issue-writer usage is attributed to its snapshotted model/account and does not alter review coverage.                                                                            |
| Scope policy and preflight   | `scope_policy` in the review profile and `scope_plan` in the repository run | Reusable code-type/folder/free-text intent plus the latest bounded commit SHA, policy hash, plan hash, explanation, warnings, and counts. It does not contain the selected-file manifest.                                                                                                                                                                                                   |

### Review Profile Fields And Bounds

Profile create/update accepts only the writable fields below. Server-assigned
schema version, ID, version, and timestamps are response metadata; update adds
`expected_version`. At most 10,000 profiles may exist in one workspace.

| JSON field | Default/inheritance | Valid value |
| --- | --- | --- |
| `name` | None | Required non-empty text, at most 256 bytes |
| `review_focus` | None | Required non-empty diagnosis focus, at most 65,536 bytes |
| `scope_policy.code_types` | `hotpath-code`, `code` | One or more unique values from `hotpath-code`, `code`, `test`, `bench-test` |
| `scope_policy.include_folders`, `exclude_folders` | Empty | At most 64 unique canonical relative prefixes in each list; each is at most 1,024 bytes; exclusions win |
| `scope_policy.free_text` | Empty | Optional narrowing guidance, at most 16,384 bytes |
| `account_ref` | Empty means the runtime default resolved at action admission | Empty or one selectable execution account reference, at most 256 bytes |
| `reviewer_model` | None | Exactly one required passive executable alias, at most 256 bytes |
| `deduplication_model` | Empty means `reviewer_model` | Empty or one passive executable alias, at most 256 bytes |
| `deduplication_similarity_threshold` | `90` | Integer 0–100 |
| `deduplication_candidate_limit` | `4` | Integer 0–20; zero disables model deduplication and promotes each raw finding separately |
| `issue_writer_model` | Empty means `reviewer_model` | Empty or one passive executable alias, at most 256 bytes |
| `issue_prompt` | Server diagnosis-only presentation prompt | Non-empty presentation instructions, at most 16,384 bytes |
| `force` | `false` | Boolean; forces campaign file work without changing immutable evidence rules |
| `auto_continue` | `true` | Boolean |
| `max_files_per_run` | `24` | Integer 1–100,000; bounds both batch admission and distinct files in a provider group |
| `max_content_bytes` | `524288` | Integer 1–524,288; bounds aggregate provider evidence, independently of the immutable per-file ceiling |
| `max_parallel_children` | `8` | Integer 1–64 |
| `assignment_timeout_seconds` | `3600` | Minute-aligned integer 60–86,400; the workflow reserves at least five additional minutes for setup/cleanup |
| `budget.guard_expression` | Empty permits work | At most 4,096 bytes, 256 tokens, and 16 nesting levels; must produce a boolean |

### Retention State

Repository-review history is retained indefinitely. `purge-history` resets the
automation to a fresh idle runtime but preserves its repository/profile/branch
configuration. Removing an automation always deletes the configuration and its
resolved authoritative repository ledgers; there is no retained orphan-ledger
mode.
Neither action deletes profiles or discussion threads, calls GitHub, changes
posted/linked/discovered issues, or deletes generic
`workspace/workflow_runs/<run-id>/` records. Threads and Workflows retain those
resources under their own policies.

Automation detail returns `can_purge_history`, `can_remove_repository`, ordered
`purge_blockers` (`code`, `count`, safe `message` only), and a `purge_summary`
with `repository_version`, opaque composite `ledger_fence`,
raw/deduplicated/repository-finding counts, issue-preview count, and unique
stored external issue URL count including conflicts. The projection is produced
calculation used by API preflight and the locked store mutation; it is advisory
rather than a substitute for automation, primary-ledger, and composite-ledger
fences. The composite fence binds every configured-identity or authoritative
fallback ledger and version, preventing a changed hidden alias from being
deleted by stale confirmation. When inventory cannot be read, both destructive
capabilities fail closed with one `retention_unavailable` blocker and no
authority to mutate. An assignment with no ledger has zero numeric version and
counts plus the deterministic empty-inventory fence, cannot purge history, and
can still be removed when otherwise quiescent; a direct purge returns `404
repository_review_history_not_found`.

Repository detail shows aggregate counts and server-provided blockers before
either destructive action and requires typing the exact displayed normalized
repository identity. Its confirmation states that local review data is
permanent, GitHub issues are unchanged, and threads/workflow runs retain their
own policies. A stale-version conflict refetches current capabilities/counts;
successful history purge refreshes the same detail and successful removal
returns to the repository collection.

Both destructive actions are serialized by one validated `0600` primary
automation intent plus `0600` repository-identity fence copies. They contain
only the configured and resolved normalized targets, operation, expected
versions, current phase, and creation time. Phases are `prepared`,
`automation_committing`, `automation_applied`, `ledger_committing`, and
`ledger_removed`; each advances only after its durable effect. Startup completes
an interrupted reset/removal before starting repository-review workers, then
removes the marker. Summary sidecars are removed consistently with their
authoritative ledger and can never resurrect purged content.
While any primary intent or configured/resolved-identity fence remains,
affected automation/ledger reads and mutations return the retryable
`repository_review_purge_in_progress` conflict instead of partial state.

## Diagnosis-Only Finding Contract

Each assignment model returns one closed top-level object. All four top-level
fields are required, unknown fields are rejected, and findings are ordered by
severity priority. Each `reviewedFiles` path must be an exact readable path the
assignment actually inspected and may occur once. Each `residualRisks` item may
identify only unavailable/unread evidence, never advice. A confirmed finding
uses the item shape below; only `line` is optional.

```json
{
  "summary": "Reviewed the assigned persistence path and confirmed one defect.",
  "reviewedFiles": ["service.go"],
  "findings": [
    {
      "severity": "high",
      "title": "Concurrent Save can silently lose a committed update",
      "symbol": "Save",
      "file": "service.go",
      "line": 83,
      "message": "When two callers enter Save from the same stored version, each can complete successfully while the later write replaces the earlier update.",
      "evidence": "Save reads the current version before serialization, and the final write path has no version check. Both executions can derive version N and commit different values as N+1.",
      "impact": "A caller receives success even though its committed state can disappear, causing silent data loss under concurrent writes.",
      "validation": {
        "status": "confirmed",
        "summary": "The two-writer interleaving was traced through the read and commit paths, and no later conflict check was present in the assigned evidence.",
        "checks": [
          "Both calls can observe the same starting version",
          "Both final writes accept state derived from that version",
          "No surrounding lock serializes the complete read-modify-write sequence"
        ]
      },
      "match_hints": {
        "component": "state persistence",
        "operation": "commit a serialized state update from a previously read version",
        "failure_mode": "the later write replaces another successful update derived from the same version",
        "trigger": "two Save calls read the same stored version before either final write",
        "violated_invariant": "every successful concurrent update remains represented in committed state",
        "observable_outcome": "a caller's successfully committed state disappears",
        "related_symbols": ["Save"],
        "source_anchors": ["version"],
        "distinguishing_facts": [
          "both callers report success",
          "requires both writes to derive from one stored version"
        ]
      },
      "fix_effort": {
        "quick": {
          "loc_min": 5,
          "loc_max": 20,
          "class": "small",
          "rationale": "The immediate containment is localized to the commit boundary."
        },
        "quality": {
          "loc_min": 30,
          "loc_max": 100,
          "class": "medium",
          "rationale": "The ownership invariant and concurrency validation span related state and test units."
        }
      }
    }
  ],
  "residualRisks": []
}
```

The server, not the model, adds finding IDs, commit/blob identity, model and
reviewer identity, consensus, context IDs, status, versions, and timestamps.
`validation.checks` records analysis already performed; it is not a test plan.
Severity is exactly `critical`, `high`, `medium`, or `low`.
`related_symbols`, `source_anchors`, and `distinguishing_facts` contain at most
32 items apiece. Validation checks contain at most 128 items. A review with no
confirmed defect returns the same envelope with an empty `findings` array.

Both `quick` and `quality` effort ranges count additions plus deletions in
hand-edited source, tests, configuration, and migrations; they exclude
generated, vendor, and documentation-only lines. Each range is 1–1,000,000 LOC
with `loc_min <= loc_max`; the quality minimum and maximum cannot be smaller
than the corresponding quick values. Class is derived from `loc_max`: `tiny`
through 10, `small` through 40, `medium` through 150, `large` through 500, and
`refactor` above 500. `refactor` is also allowed below that threshold only when
its rationale explicitly identifies a cross-subsystem architectural or contract
migration. Rationale describes sizing evidence only and never a fix design.

## Issue Preview And Association Contract

Every new preview is generated from one immutable finding projection. The
fixed private system policy is server-owned; the user payload contains only the
finding diagnosis, validation already performed, exact location and
commit/blob/context provenance, plus bounded presentation instructions. The
default instructions request:

- a concise defect title;
- GitHub-flavored Markdown sections for evidence, observable impact,
  validation already performed, and exact location;
- the reviewed commit and primary blob provenance; and
- the `bug` label.

Presentation instructions may reorder, condense, or rename those sections, but
cannot request a fix, unsupported claim, invented reproduction, or external
lookup. Draft and Post resolve the currently assigned profile, then snapshot its
prompt, writer alias, account, profile ID, and version into the reservation. The
request has no session history, cache, tools, hooks, MCP initialization, or durable raw
response, and accepts only the bounded structured title/body/labels schema.
Candidate ranking uses the same isolation and policy over server-fetched issue
metadata.

Automatic discovery is allowed only for the first ranked candidate when its
integer score is at least 95, it reports at least four matching causal anchors,
and it reports zero conflicting anchors. The server must then re-fetch that
exact issue and validate its canonical identity and same normalized repository
before storing a reversible `discovered` association. Score 94, three anchors,
any conflict, a failed re-fetch, or a repository mismatch remains an unlinked
candidate for explicit operator selection.

The durable issue states have these meanings:

| State        | Contract                                                                                                                                                                                         |
| ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `generating` | The finding is reserved under a generation ID before the provider call. Repeating that generation ID returns or completes the same reservation.                                                  |
| `failed`     | Initial generation produced no valid preview. A bounded safe error is retained; retry or deletion is allowed.                                                                                    |
| `editing`    | A valid unpublished preview exists. Version-fenced editing, regeneration, deletion, or publication is allowed. Failed regeneration retains this last good content and records only a safe error. |
| `publishing` | Exact title/body/labels are frozen before the external call. Editing, regeneration, deletion, linking, and a second create are forbidden.                                                        |
| `unknown`    | The external effect may have occurred. Only marker-based reconciliation may advance it; deletion or blind publication is forbidden.                                                              |
| `posted`     | A created issue or re-fetched validated existing issue is canonical. Created associations are permanent; confirmed unlink/replace is available only for `linked` or `discovered` origins.        |

Origins are `ai_generated`, `linked`, `discovered`, or `legacy`. New previews contain one
finding ID. On durable load, a legacy finding's canonical association is chosen
deterministically by preferring `posted`, then `publishing` or `unknown`, then
the newest `editing` draft. Grouped drafts and duplicate noncanonical records
remain visible with their original membership; conflicting records are
read-only and cannot publish. The same manually linked GitHub issue may be the
validated canonical association for more than one finding.

## Repository-Finding Presentation Contract

Human-facing labels are a view projection over the stable API and query
contract. Clients continue sending and filtering the raw values in this table;
persisted records, mutation bodies, and collection query suggestions do not use
the labels.

| Domain | Raw value | Human label |
| --- | --- | --- |
| Occurrence grouping | `new` | Unique so far |
| Occurrence grouping | `known` | Matched existing finding |
| Occurrence grouping | `provisional` | Needs duplicate review |
| Issue | `none` | No issue or preview |
| Issue | `draft` | Saved preview · Not on GitHub |
| Issue | `open` | GitHub issue open |
| Issue | `closed` | GitHub issue closed |
| Issue | `unknown` | GitHub issue state unknown |
| Resolution check | `not_requested` | Not checked |
| Resolution check | `pending` | Queued |
| Resolution check | `running` | Checking |
| Resolution check | `confirmed` | Fix confirmed |
| Resolution check | `not_fixed` | Still present |
| Resolution check | `inconclusive` | Inconclusive |
| Resolution check | `failed` | Check failed |
| Issue preview | `editing` | Unpublished preview · Not on GitHub |
| Issue preview | `publishing` | Posting to GitHub |
| Issue preview | `unknown` | Publication needs reconciliation |
| Issue preview | `posted` | Posted to GitHub |

The raw repository-finding query fields remain `match`, `issue`, and
`validation`. The occurrence diagnosis object remains `validation` and its
status, summary, and checks describe evidence validation already performed at
finding creation; only the later repository-level resolution-check workflow is
presented as Resolution check.

### Repository-Finding Lifecycle

Lifecycle is version-fenced and independent from occurrence grouping, issue
state, and resolution-check state. Only the transitions below are legal; an
event not listed leaves lifecycle unchanged.

| From | Trigger | To | Additional conditions |
| --- | --- | --- | --- |
| — | First canonical mapping | `open` | New repository finding |
| `open` | Validated associated GitHub issue becomes closed | `resolution_pending` | Closure alone does not prove a fix |
| `open`, `resolution_pending`, `resolved`, `dismissed`, or `regressed` | Validated associated GitHub issue becomes open | `open` | Reopening restores actionable state |
| `resolution_pending` | Reversible issue association is removed | `open` | Created issues remain non-reversible |
| Any non-provisional, non-dismissed actionable state | Resolution check confirms an ancestry-verified supplied commit | `resolved` | Stores fix commit/time and first containing semantic-version tag |
| `resolved` or `resolution_pending` | Resolution check returns `not_fixed` | `open` | `inconclusive` and `failed` do not change lifecycle |
| `resolved` | A later default-branch occurrence is verified after the confirmed resolution | `regressed` | Preserves earlier resolution history and clears current check to `not_requested` |
| `open` or `regressed` | Operator dismisses | `dismissed` | Finding is non-provisional, has no issue association, has no queued/running check, and version matches |
| `dismissed` | Operator reopens | `open` | Version must match; issue and check rules still apply |

Issue closure does not move `resolved`, `dismissed`, or `regressed` to
`resolution_pending`. Manual mutation can request only `dismissed` or `open`;
`resolution_pending`, `resolved`, and `regressed` are evidence-driven. Merge
and historical-consolidation policy preserves the most actionable compatible
state and records incompatible issue URLs as a conflict rather than inventing a
transition.

Repository-finding list identity is `N occurrences across M commits`. The table
columns are Identity, Severity, Occurrences, Finding state, Issue, Resolution
check, and Updated. List, Table, and Grid suppress normal `new`/`known` badges
and show only the attention badges Duplicate review, Issue conflict, and Fix
check failed. The Occurrence grouping value remains Needs duplicate review,
and the Resolution check value remains Check failed. Lifecycle is always
rendered independently rather than folded into an attention badge. Across
List, Table, Grid, and the mobile Table fallback, repository-finding titles and
metadata use the identity block's available width; responsive truncation or
wrapping is not replaced by a fixed maximum text width.

On detail, Occurrence grouping replaces a raw match-state heading. A
provisional decision panel and the Issue section precede long provenance,
occurrence, and resolution histories. A possible-duplicate card joins its
judgment to a compact candidate projection containing title, latest location,
severity, occurrence and commit counts, duplicate-decision evidence, and View
candidate. Keep separate submits the existing `distinct` decision. Merge with
candidate requires confirmation before dispatch. Internal previews use a
document icon and Review preview; the GitHub icon is reserved for records with
a validated external GitHub URL.

Each newly failed resolution check records an allowlisted, API-safe failure
with a stable code, displayable message, retryability, and timestamp on both
the validation job and its resolution-history entry. Raw provider errors,
prompts, source content, credentials, and repository paths never enter that
failure. Repository-finding detail renders the latest failed attempt as a
visible Fix check failed alert beside the resolution-check facts and retains
Retry fix check. A legacy failed attempt without structured diagnostics says
that no failure details were recorded; it never invents a cause from an empty
candidate set or other circumstantial state. Historical failure details remain
hidden whenever the current resolution-check state is not `failed`.

Finding health is a separate read model, not a replacement for raw or
deduplicated collections. Run findings use the selected automation's campaign
membership. Repository findings and Findings processing use the canonical
repository ledger so an older failed source remains reachable. The review
detail presents Run findings, Repository findings, Raw findings, Findings
processing, and Issue previews as distinct destinations. Its Repository
findings and Unrepresented run findings metrics come from health; unavailable
health never renders as a false zero. A stopped or failed review alert precedes
all cards and metrics.

The Findings processing collection defaults to `ALL ORDER BY updated DESC` and
supports List, Table, and Grid. Identity contains the immutable diagnosis;
collection facts include path, model/reviewer, processing state, disposition,
severity, and update time. Processing states render pending as Queued, running
as Processing, failed as Failed, and completed as Completed. Only failed rows
can enter the explicit selection. Retry selected retains explicit safe failures
for partial outcomes, and the direct detail exposes the same diagnosis,
provenance, failure, linked finding identities, and individual Retry action.
The historical-consolidation notice offers Retry only when the server marks the
failed state retryable; pending, replaying, and merging states poll, but the UI
never turns polling into automatic retry authority.

## Task-Admission Guard Contract

An empty expression permits every task. A non-empty expression must evaluate to
the boolean value `true`; `false` or a final unknown value pauses admission.
Expressions are limited to 4,096 bytes, 256 tokens, and 16 nesting levels.
`*` documents a field family and is not literal wildcard syntax.

Available fields:

- `spent.tokens.prompt`, `.completion`, `.cached`, `.total`
- `spend.total.usd`
- `account.limits.known`, `.exhausted_known`, `.exhausted`, `.any`
- `account.limits.any.known`, `.observed`, `.remaining_percent`,
  `.used_percent`, `.minimum_remaining_percent`, `.maximum_used_percent`
- `account.limits.<window>.known`, `.observed`, `.remaining_percent`,
  `.used_percent`, `.minimum_remaining_percent`, `.maximum_used_percent`, where
  common window names normalize to values such as `daily` and `weekly`

For example:

```text
account.limits.weekly.known and
account.limits.weekly.remaining_percent >= 10 and
spent.tokens.total < 500000 and
spend.total.usd < 25
```

The expression is evaluated only for the profile's execution account. For an
account router, reachable concrete members aggregate conservatively: minimum
remaining percentage and maximum used percentage. A missing price makes
`spend.total.usd` unknown; missing or partial limit telemetry makes affected
numeric limit fields unknown. `.known` fields let a profile state its intended
policy explicitly without a separate fail-open/fail-closed toggle.

Legacy threshold fields are durably rewritten into a bounded expression on
load. Legacy account IDs selected telemetry rather than execution credentials,
so any account-targeted legacy policy migrates fail-closed until an operator
selects the single execution account required by the new contract.

## Surface Ownership

<!-- prettier-ignore -->
Owns: CODE pkg/repoaudit/**
Owns: TEST pkg/repoaudit/**
Owns: CODE cmd/repository-review-attribution-backfill/**
Owns: TEST cmd/repository-review-attribution-backfill/**
Owns: CODE pkg/gateway/repository_review_*.go
Owns: TEST pkg/gateway/repository_review_*
Owns: CODE web/backend/api/repository_review*.go
Owns: TEST web/backend/api/repository_review*_test.go
Owns: CODE web/frontend/src/api/repository-reviews.ts
Owns: TEST web/frontend/src/api/repository-reviews.test.ts
Owns: CODE web/frontend/src/components/repository-reviews/**
Owns: TEST web/frontend/src/components/repository-reviews/**
Owns: CODE web/frontend/src/routes/repository-reviews*.tsx
Owns: TEST web/frontend/src/routes/-repository-reviews*.test.tsx
Owns: HTTP * /api/repository-reviews*
Owns: UI /repository-reviews*

Profile state is stored in `workspace/repository_reviews/profile_rrpf_*.json`;
repository configuration and runtime state remains in
`workspace/repository_reviews/automation_rra_*.json`. Authenticated profile CRUD
uses `/api/repository-reviews/profiles*`. Repository configuration and run
lifecycle use `/api/repository-reviews/automations*`, with `start`, `pause`,
`resume`, and `restart` action subresources, a per-run `commit-options`
preflight for remembered/latest selection, plus
`GET /api/repository-reviews/automation-options` for safe model/account choices.
Automation-owned detail, run-finding, repository-finding, issue-generation,
preview, and existing-issue routes resolve the repository ledger through that
assignment and are the canonical UI API. Their collection reads use typed
query/cursor envelopes and compact summaries; ID-addressed reads retain the
complete detail projections. Repository-ID ledger APIs and offset-based list
reads remain compatibility surfaces. The protected gateway retains authority for GitHub create, marker
reconciliation, search, and exact-issue re-fetch; the issue-writer model has no
provider tools.
Profileless automation files written by older versions remain readable for
recovery and have legacy `HEAD`/target values sanitized at admission. New HTTP
creation always requires `profile_id`; the split UI does not expose legacy
profileless editing or multi-model actual-review configuration.

The shared application sidebar and thread/chat controller remain auxiliary
interfaces owned by their existing feature specifications.

## Auxiliary Interfaces

| Type         | Surface                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       | Contract                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       | Requirement IDs                                                                                         |
| ------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| HTTP         | `GET/POST/PATCH/DELETE /api/repository-reviews/profiles*`                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | List/create/update/delete reusable CAS-fenced profiles, including one execution account, one reviewer, optional issue writer, bounded scope/sizing, and one guard expression, but no caller-supplied pricing or polling controls. Assigned deletion and active-assignment mutation fail closed.                                                                                                                                                                                                                                                | `FR-REPOREVIEW-008`, `FR-REPOREVIEW-011`, `FR-REPOREVIEW-012`, `FR-REPOREVIEW-014`, `FR-REPOREVIEW-018` |
| HTTP         | `GET/POST/PATCH/DELETE /api/repository-reviews/automations*`                                                                                                                                                                                                                                                                                                                                                                                                                                                                  | List/create/update/delete unique repository/profile assignments; independently load one automation; resolve remembered/latest commit options; and invoke start, pause, resume, or restart transitions against an exact admitted SHA. Internal sketches and credentials are never projected; bounded scope preflight summaries and frozen reviewer/writer provenance are returned.                                                                                                                                                              | `FR-REPOREVIEW-009`, `FR-REPOREVIEW-013`, `FR-REPOREVIEW-015`, `FR-REPOREVIEW-018`                      |
| HTTP         | `GET /api/repository-reviews/automation-options`                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | Return safe passive aliases and selectable runtime account refs with bounded current limit summaries and explicit execution availability, never credential material. Empty profile `account_ref` remains the explicit Default account option.                                                                                                                                                                                                                                                                                                  | `FR-REPOREVIEW-011`, `FR-REPOREVIEW-012`, `FR-REPOREVIEW-018`                                           |
| HTTP | `GET /api/repository-reviews/automations/{automation_id}/file-attributions?query=...&cursor=...&limit=...` | Query successful file/focus/agent/reviewer/model attribution with a query-bound cursor. Private assignment masks, reservations, checkpoint digests, and raw evidence are omitted. | `FR-REPOREVIEW-030`, `FR-REPOREVIEW-036`, `FR-REPOREVIEW-042` |
| HTTP | `POST /api/repository-reviews/automations/{automation_id}/purge-history`; destructive `DELETE /api/repository-reviews/automations/{automation_id}` | Require `expected_version`, `expected_repository_version`, opaque `expected_ledger_fence`, and exact normalized `confirm_repository`. Purge preserves a fresh idle configuration; Delete removes configuration and every resolved ledger. Both use the same authoritative blockers/capabilities and restart-safe intent, and never change GitHub issues, profiles, discussion threads, or generic workflow runs. | `FR-REPOREVIEW-041`, `FR-REPOREVIEW-042` |
| HTTP         | Deprecated `GET /api/repository-reviews/automations/{automation_id}/run-findings?query=...&cursor=...&limit=...`; deprecated `GET .../run-findings/{finding_id}`; `POST .../findings/status`; compatibility `.../report` | Preserve stored `rfn_*` occurrence clients and bounded repository-mapping status retry without using those resources in the primary UI. The collection excludes modern `rdf_*` projections, and the detail path returns `404` for every non-`rfn_*` ID. Legacy `scope`/`offset` report reads remain compatibility behavior. | `FR-REPOREVIEW-003`, `FR-REPOREVIEW-004`, `FR-REPOREVIEW-017`, `FR-REPOREVIEW-022`–`FR-REPOREVIEW-028` |
| HTTP         | `GET /api/repository-reviews/automations/{automation_id}/findings?query=...&cursor=...&limit=...`; `GET .../findings/{finding_id}`; nested `.../sources`; `GET .../raw-findings?query=...&cursor=...&limit=...`; `GET .../raw-findings/{source_id}`; `POST .../raw-findings/{source_id}/retry`; `POST .../historical-deduplication/retry`; `POST .../historical-deduplication/restart` with `{ "confirmed": true }` | Strictly query completed current-campaign deduplicated findings with source counts, processing counters, and historical status; load representative diagnosis and paged sources; independently query or inspect every current-campaign raw record, resolve legacy aliases, retry one failed native source, Resume a compatible failed historical replay from its checkpoints, or explicitly restart only an incompatible dependency closure after the structured restart-required conflict and confirmation. | `FR-REPOREVIEW-031`, `FR-REPOREVIEW-032` |
| HTTP         | `GET /api/repository-reviews/automations/{automation_id}/finding-health`; `GET .../findings-processing?query=...&cursor=...&limit=...`; `GET .../findings-processing/sources/{source_id}`; `POST .../findings-processing/sources/{source_id}/retry`; `POST .../findings-processing/retry` | Project selected-automation run coverage plus canonical-ledger repository/processing health; query every canonical processing source with typed cursor binding while preserving legacy offset/state mode; inspect one source; and retry one or 1–200 unique failed sources with explicit ordered partial outcomes, updated counters, and health. | `FR-REPOREVIEW-031`–`FR-REPOREVIEW-033` |
| HTTP | `GET .../campaigns/{campaign_id}/findings-processing`; `GET/POST .../campaigns/{campaign_id}/findings-processing/sources/{source_id}[/retry]`; `GET .../historical-deduplication` | Preserve campaign-scoped processing compatibility, canonical source detail/retry, and a bounded historical-consolidation status projection. Primary UI uses canonical repository-wide processing and explicit historical retry/restart resources. | `FR-REPOREVIEW-032`, `FR-REPOREVIEW-037`, `FR-REPOREVIEW-038` |
| HTTP         | `GET /api/repository-reviews/automations/{automation_id}/repository-findings?query=...&cursor=...&limit=...`; `GET /api/repository-reviews/automations/{automation_id}/repository-findings/{finding_id}`                                                                                                                                                                                                                                                                                                                      | Query and cursor-page compact canonical repository-finding summaries with exact occurrence/commit counts, lifecycle, issue, and resolution-check state independently from run occurrences. Detail loads the complete aggregate and occurrence histories plus compact possible-duplicate candidates with title, latest location, severity, exact counts, evidence, and version for explicit resolution. Raw API/query enum names and values remain stable beneath human labels. | `FR-REPOREVIEW-005`, `FR-REPOREVIEW-017`, `FR-REPOREVIEW-022`–`FR-REPOREVIEW-028`                       |
| HTTP | `PATCH .../repository-findings/{repository_finding_id}`; `POST .../repository-findings/{repository_finding_id}/duplicates`; `POST .../repository-findings/validations`; `POST .../repository-findings/{repository_finding_id}/sync` | Version-fence manual dismiss/reopen, resolve a provisional duplicate as distinct or confirmed merge, queue 1–50 resolution checks, or explicitly synchronize one canonical GitHub issue snapshot. | `FR-REPOREVIEW-022`–`FR-REPOREVIEW-027`, `FR-REPOREVIEW-040`, `FR-REPOREVIEW-042` |
| HTTP         | `POST /api/repository-reviews/automations/{automation_id}/issues/generations`; `GET /api/repository-reviews/automations/{automation_id}/issues?query=...&cursor=...&limit=...&generation_id=...`; `GET/PATCH/DELETE /api/repository-reviews/automations/{automation_id}/issues/{draft_id}`; `POST .../issues/{draft_id}/regenerate`; `POST .../issues/publish`; `POST .../issues/{draft_id}/publish`                                                                                                                          | Reserve and generate at most 200 explicit one-finding previews under one generation ID with four-call concurrency; query/cursor-page compact durable previews with generation-bound cursors and authoritative `publishable`/`publish_blockers`; load one preview with the same `can_publish`/blocker capability projection; version-fence edit, regeneration, or conditional deletion; and publish only an eligible confirmed subset or one eligible preview with per-preview outcomes. Blocked selections fail before gateway/provider initialization. Legacy offset reads remain compatible. | `FR-REPOREVIEW-005`, `FR-REPOREVIEW-006`, `FR-REPOREVIEW-016`, `FR-REPOREVIEW-018`, `FR-REPOREVIEW-019` |
| HTTP         | `POST /api/repository-reviews/automations/{automation_id}/findings/{finding_id}/issue-link/candidates`; `POST/DELETE /api/repository-reviews/automations/{automation_id}/findings/{finding_id}/issue-link`                                                                                                                                                                                                                                                                                                                    | Search and AI-rank bounded server-derived same-repository GitHub candidates, then re-fetch and version-fence an explicitly confirmed link, unlink, or replacement according to association origin.                                                                                                                                                                                                                                                                                                                                             | `FR-REPOREVIEW-016`, `FR-REPOREVIEW-018`–`FR-REPOREVIEW-020`                                            |
| HTTP | Compatibility `PATCH .../findings/{finding_id}`; `POST .../findings/{finding_id}/post` | Reject direct immutable-occurrence status mutation with a conflict while preserving direct generate-then-publish behavior. Canonical issue authority remains repository-finding based and direct Post re-checks eligibility under the ledger lock. | `FR-REPOREVIEW-016`, `FR-REPOREVIEW-019`, `FR-REPOREVIEW-039`, `FR-REPOREVIEW-042` |
| HTTP         | `GET/PATCH/POST /api/repository-reviews/{repository_id}/**`                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | Compatibility APIs page repository ledgers and retain issue-draft operations; direct review-finding status mutation is rejected because occurrences are immutable. The routed UI uses automation-owned endpoints.                                                                                                                                                                                                                                                                                                                              | `FR-REPOREVIEW-001`, `FR-REPOREVIEW-002`                                                                |
| Gateway HTTP | `/runtime/repository-reviews/<repo>/issue-drafts/<draft>/publish`; `POST /runtime/repository-reviews/automations/{automation_id}/findings/{finding_id}/issue-link/candidates`; `POST/DELETE .../issue-link`; `POST .../repository-findings/{id}/sync`                                                                                                                                                                                                                                                                         | Idempotent publication/reconciliation, bounded same-repository candidate discovery, exact issue re-fetch, reversible discovery/manual associations, and explicit issue-state synchronization for canonical GitHub identities.                                                                                                                                                                                                                                                                                                                  | `FR-REPOREVIEW-006`, `FR-REPOREVIEW-019`, `FR-REPOREVIEW-020`, `FR-REPOREVIEW-026`                      |
| UI           | `/repository-reviews`; `/:id`; `/:id/findings`; `/:id/findings/:findingId`; `/:id/raw-findings`; `/:id/raw-findings/:sourceId`; `/:id/findings-processing`; `/:id/findings-processing/:sourceId`; compatibility redirect `/:id/report`; `/:id/issues`; `/:id/issues/:draftId`; `/:id/issues/:draftId/edit`; `/repository-reviews/repositories`; `/repositories/new`; `/repositories/:id`; `/repositories/:id/edit`; `/repositories/:id/findings`; `/repositories/:id/findings/:findingId`; `/repositories/:id/findings/:findingId/link-issue`; `/repository-reviews/profiles`; `/profiles/new`; `/profiles/:profileID`; `/profiles/:profileID/edit` | Standard compact review, deduplicated-finding, raw-finding, processing-recovery, issue-preview, repository-configuration, repository-finding, and profile collections with List/Table/Grid views and dedicated stable detail/editor routes. Review detail leads with a stopped/failed alert, then separates Run findings, Repository findings, Raw findings, Findings processing, and Issue previews and uses health for repository/unrepresented metrics. Findings processing exposes every canonical source, failed-only selection, bounded partial bulk retry, safe detail provenance, and individual retry while raw findings remains current-campaign evidence. Repository findings emits one health-scoped incomplete-coverage notice; duplicate/conflict/failed-check attention stays row-level. Repository-finding identity is `N occurrences across M commits`; its table is Identity, Severity, Occurrences, Finding state, Issue, Resolution check, and Updated. Detail places duplicate decisions and Issue before histories, uses candidate cards with View candidate, Keep separate, and confirmed Merge with candidate, and derives state-aware resolution-check actions. Internal previews use a document icon and Review preview; GitHub iconography/actions require a validated external URL. Failed historical replay exposes checkpoint Resume and is never retried by polling. Only the structured restart-required conflict reveals **Restart incompatible work**, whose confirmation distinguishes reprocessed affected buckets from preserved unrelated work; every other failure remains on the normal error path. Each nested collection preserves its own query, view, selection, cursor paging, and scroll state. Legacy `rfn_*` bookmarks redirect to canonical raw detail; Results, report, `scope`, and offset UI state normalize or redirect to canonical cursor surfaces. | `FR-REPOREVIEW-003`–`FR-REPOREVIEW-006`, `FR-REPOREVIEW-009`, `FR-REPOREVIEW-015`–`FR-REPOREVIEW-033` |

The normative [Repository Reviews API reference](../reference/repository-reviews-api.md)
enumerates every registered launcher route, request/response envelope, bound,
authentication rule, and safe error. Route-registration parity tests keep that
reference synchronized with the server.

## Algorithms And Ordering

1. Durable ledger load normalizes missing collections and backfills canonical
   finding/draft references deterministically. It never deletes a legacy
   grouped or duplicate draft; noncanonical conflicts are projected read-only.
2. Profile create/update normalizes one reviewer, an optional issue writer,
   work bounds, scope, and guard policy under the shared review-store lock. A
   blank writer inherits the reviewer; an explicit writer must be passive and
   executable on the selected effective account. Code types are canonicalized;
   folder prefixes must be exact safe repository-relative paths. CAS mismatch
   fails without partial mutation.
3. Repository configuration create/update normalizes repository identity and an
   optional branch under the catalog lock, atomically rejects a second
   configuration for the same repository, and binds one existing profile.
4. Start/resume/restart checks the action-specific source state, controller
   lifecycle, workspace lease, current configuration, assigned profile version,
   selected account/reviewer/writer compatibility, and central pricing required
   by any `spend.total.*` field. It snapshots the effective account, reviewer,
   writer, profile version, exact commit, and an opaque campaign ID while the
   automation remains inactive. A changed profile or commit and explicit
   Restart create a new ID and clear campaign counters; automatic continuation
   and same-commit Resume preserve the existing ID, counters, checkpoints,
   start time, and run history.
5. Admission authorizes that ID against the canonical repository ledger and
   exact commit under review-version CAS before registering an active worker or
   appending a run ID. It then atomically persists `running`, run history, and
   the canonical full `resolved_commit_sha`; the workflow receives only that
   exact SHA. Automatic handoff reuses it. Resume compares it with the freshly
   resolved tip and requires an exact old/latest/custom selection when they
   differ.
6. The workflow acquires a fresh checkout, inventories exact Git blobs, plans
   only changed/profile/account-invalidated work, freezes bounded source
   evidence, and releases the checkout before model calls. Scope planning,
   managed reviewers, fallbacks, and structured repairs all use the profile's
   frozen effective account. The controller gives the built-in review workflow
   at least 15 minutes while preserving any longer configured workflow timeout,
   so required review children retain time to finish before the durable
   checkpoint cleanup reserve. On the initial plan, native validation removes
   only well-formed candidate IDs that are absent from the commit-bound catalog,
   retains known IDs, emits bounded counts without exposing those IDs, and uses
   trusted prefixes and hard scope when no exact ID remains. The first validated
   scope step durably freezes that sanitized canonical selection and plan before
   review work starts; continuation batches revalidate them strictly against the
   same commit and hard policy without another planner call. Campaign planning
   is admitted only for the built-in workflow, and native code derives the
   immutable required-assignment denominator from four trusted task assignments
   multiplied by required resolved reviewers; model output cannot choose it.
   The immutable profile hash binds requested aliases, resolved model-graph
   revision, effective reviewer cohort, default-chain classification, effective
   account, and the resolver-clamped content bound, so routing drift conflicts
   instead of reusing incompatible checkpoints.
7. When a worker dequeues one managed child, the controller serializes only the
   admission decision, adds that child's projected prompt/output tokens and
   known cost to current in-flight reservations, refreshes referenced limit
   telemetry for the selected account, and evaluates the expression. A denial
   latches a safe stop and prevents new children; already admitted children may
   finish. Every provider response records requested reviewer, actual model,
   actual token usage, cost, and latency. Completion releases the projection;
   accounting persistence failure fails closed.
8. Record consumes every actual managed child, including failed, invalid,
   zero-finding, and optional children. Trusted runtime ordinals create stable
   assignment IDs; exact FileRefs are canonicalized; successful same-child
   acknowledgements produce the inspected union; only every required
   assignment acknowledging a file produces completion. Aggregate-limit files
   produce explicit failed required evidence and remain pending, while binary
   and file-too-large files remain terminal unsupported.
9. A completed batch counts only after the qualified record step persisted a
   run with an explicit valid nonnegative integer `remaining_files`, or an
   authoritative no-op has both an explicit zero pending count and an explicit
   valid zero persisted-result remaining count. A successful record's persisted
   `remaining_files` is authoritative; a valid top-level `remainingFiles` or
   `remaining_files` projection must agree with persisted record/result evidence.
   Missing, malformed, negative, fractional, contradictory, and above-domain
   values never imply zero and never overwrite prior progress. The controller
   then merges campaign-scoped ledger outcomes. A batch makes file progress only
   when a verified remaining count falls, its durable record reports reviewed or
   unsupported files, or campaign-scoped ledger counters rise; projected
   top-level counters and finding occurrences alone never qualify. If
   remaining work is positive and the verified batch resolves zero files, the
   controller applies exact ID-first campaign metrics independent of bounded
   run history, preserves completed-batch telemetry, pauses with `no_progress`,
   and does not admit another batch until explicit Resume. Existing explicit
   manual, guard, failure, or service pause reasons take precedence.
   Legacy campaign recovery is background/admission work and an explicit
   prerequisite for resuming failed historical consolidation: for an inactive
   automation it scans at most 1,000 retained workflow runs, accepts inspection
   only from validated same-child acknowledgements (including zero-finding
   responses), and installs one recovered campaign, manifest, coverage, and
   provenance through automation/campaign/review-version CAS. Across current
   runtime profile drift, only exact required file-attribution evidence whose
   recovered run, semantic focus/reviewer, commit, inventory, run profile,
   completion time, and file identity all match that installed authority may
   seed reusable assignment credit. Independently envelope-validated finding
   provenance remains eligible for replay. Missing, corrupt, truncated,
   inconsistent, or bound-limited history
   fails closed without installing a campaign, changing raw provenance, or
   hiding legacy findings; automation Resume and historical-consolidation Resume
   remain unavailable until exact provenance can be recovered, and this evidence
   failure never offers incompatible-work restart.
10. The monitor only reconciles orphaned active runs after launcher failure; it
   does not poll account limits or auto-resume a guard pause. Explicit Resume
   causes the next worker pickup to fetch current telemetry and evaluate again.
11. New campaign raw findings, contexts, and model projections join by campaign
    ID rather than the retained run list. Exact metrics distinguish inspected
    files, fully reviewed files, remaining and unsupported files, raw findings,
    deduplicated findings, distinct repository aggregates, and pending mappings.
    During replay the raw count unions retained legacy occurrences with admitted
    records by legacy identity, so admitting a source changes processing state
    without changing the total. The repository-owned projection reads the
    complete canonical aggregate ledger. Polling observes newly committed
    checkpoints, deduplication, and mapping; terminal replay failure stops
    polling and requires an explicit retry that may consume model usage.
12. Batch generation validates at most 200 explicit finding IDs, rejects every
    non-open or associated finding, creates one generation ID, and reserves
    each canonical association under the ledger lock before dispatch. A
    workspace-wide four-slot OS semaphore admits at most four writer calls
    concurrently, while a per-attempt lock prevents duplicate cross-process
    dispatch. Retrying the same generation ID reuses completed results and
    pending reservations.
13. Each new Draft/Post action resolves and freezes the current assigned profile,
    issue prompt, writer, and account with the fixed private structured policy.
    Validated output replaces a `generating` reservation
    with `editing`; a failed initial call stores `failed` and a safe error. On
    regeneration, active-attempt provenance is stored separately and promoted
    with validated output atomically, while a failure retains the prior good
    title/body/labels and their original provenance.
13. Candidate discovery derives bounded searches from causal hints, symbols,
    anchors, path history, and title, fetches and deduplicates at most 50
    same-repository GitHub issues, and passes at most 10 ranked candidates to
    the isolated writer. Automatic discovery considers only the first ranked
    candidate and requires score ≥95, at least four matching anchors, and zero
    conflicting anchors. Manual and qualifying discovered selection both
    re-fetch the exact issue and validate repository, URL, and external
    identity; discovered associations remain reversible.
14. Issue publication evaluates the same complete blocker set for capabilities,
    collection summaries, API/gateway preflight, and the locked store claim.
    Only an eligible initial request freezes its selected preview in
    `publishing`, searches its stable marker, creates when safe, and stores the
    per-preview `posted`, safe failure, or `unknown` outcome before returning.
    An `unknown` preview can advance only through reconciliation; it is never
    blindly created again.
15. Startup reconciles internal association jobs, then CPU matching performs
    exact, deterministic-threshold, and BM25 stages. Ambiguity snapshots the
    current reviewer/profile/account before isolated opaque-ID adjudication.
    Association and job completion share one write; automatic failure stops after
    three attempts, a bounded explicit status retry starts a fresh attempt window,
    and provisional decisions are CAS-fenced. The UI joins each retained
    judgment to its compact candidate projection, renders Keep separate for the
    raw `distinct` decision, and requires explicit confirmation before Merge
    with candidate. Displayed confidence never dispatches a provisional merge.
16. Resolution validation claims at most 50 selected items through four
    workspace-wide slots, retrieves at most 200 reachable default-branch commits
    touching path/symbol history, BM25-ranks eight bounded evidence records, and
    accepts only a supplied ancestry-verified fix commit before recording its
    first chronological semantic-version tag. Presentation maps the stable
    states to Resolution check labels and Check for fix, Retry fix check, Fix
    check queued, or Checking for fix… actions without renaming the API/query
    `validation` field or occurrence diagnosis Validation. A failed attempt
    persists only its allowlisted safe failure classification, and detail shows
    that reason without exposing raw model or repository diagnostics.
17. Purge/removal preflight computes one authoritative capability and blocker
    projection from the automation and repository ledger. After matching both
    version fences and exact normalized repository confirmation, the locked
    mutation writes the `0600` intent/fences, rechecks quiescence, then either
    resets the automation and removes its ledger/summary sidecar or removes both
    configuration and ledger. It advances and durably rewrites the intent only
    after each corresponding configuration/ledger phase completes; repeated
    application recognizes already-applied effects. Startup performs the same
    idempotent reconciliation before workers may claim repository-review work.
    No purge phase initializes a GitHub/provider client or deletes a generic
    workflow-run directory.

## Cross-Feature Behavior

- Workflows owns generic run persistence, DAG execution, events, cancellation,
  immutable repository-bug-finder YAML, and agent call admission plumbing.
- Agent execution optimization owns managed splitting, independent reviewer
  assignment, usage aggregation, and structured repair behavior.
- Account routing/model configuration owns passive alias resolution, effective
  account compatibility, and conservative provider/account price metadata;
  GitHub Copilot supplies measured or conservative usage when its transports
  omit it.
- Git workspaces owns fresh lease acquisition, immutable Git-object reads, and
  checkout release. Repository review never retains mutable workspace access in
  a model call.
- Repository model evaluations owns the separate model-review probe lifecycle,
  read-only profile materialization, frozen comparison corpus, deterministic
  provider-call sizing sweep, blinded judging, and quality/efficiency ranking.
  A probe freezes the admitted profile version and never changes a review
  profile, assignment, repository run, or finding ledger.
- Agent/workflow isolation owns the private ephemeral no-history/no-cache/no-tool
  structured request boundary. Repository review supplies the fixed
  diagnosis-only policy, immutable projection, selected writer/account, and
  output schema.
- Launcher authentication and same-origin mutation guards protect every control,
  generation, link, and publication mutation. Runtime events/workflow pages
  remain optional detailed observation surfaces.
- Threads owns durable discussion creation; the protected GitHub provider owns
  issue search, exact issue re-fetch, and external create calls. Repository
  review supplies bounded server-derived queries, exact provenance, association
  state, and explicit confirmation to those features.

## Failure And Edge Cases

- Duplicate normalized repository assignments, missing or assigned profiles,
  credentialed or query/fragment-bearing repository URLs, unsafe paths,
  symlinked roots/files/locks, oversized ledgers, invalid reviewer or writer
  aliases, unavailable account/model pairs, unsafe agentic CLI aliases, invalid
  prices required by a monetary guard, and out-of-range work/guard values fail
  before execution. Blank writer always resolves to the reviewer; it never
  means a later runtime default model.
- Scope policies reject unknown/duplicate code types, absolute, parent-relative,
  non-canonical, duplicate, or over-limit folder prefixes, and oversized free
  text. Include folders narrow category matches and excludes always win. The
  commit-bound summary is invalidated when repository/branch/profile/account/scope changes.
- Branch configuration rejects detached commit or tag targets and every unsafe
  or ambiguous ref form. Internal workflow checkpoints may still reacquire the
  exact commit resolved from the admitted branch.
- A guard that references `spend.total.usd` requires a positive central price
  for every reachable selected-account route. Unknown price is unknown, never
  zero/free.
- Parallel workers remain configurable up to 64. Prompt/output/cost projections
  are reserved at task pickup, so later workers see admitted in-flight work
  before evaluating the same expression. Provider-reported usage remains after
  a task releases its projection.
- Unknown or partial selected-account telemetry makes affected numeric fields
  unknown; final unknown denies work. Known `limit_reached`/exhausted entries
  normalize to zero remaining. Multiple entries in one window aggregate to the
  most conservative value.
- Manual and guard pause intent survives a launcher restart and is never
  auto-resumed. Orphaned running work becomes `service_restart`; explicit
  Resume re-enters through the same task-admission boundary.
- Safe stop accepts a stale observation version only while the submitted run ID
  still identifies the active run or latest queued handoff, and can turn that
  queued handoff directly into `paused`. A delayed command cannot stop a later
  campaign, and stop never changes an initial idle configuration into a paused
  run.
- Missing, malformed, ambiguous, or unreachable custom commits fail before a
  worker starts. A workflow commit that differs from `resolved_commit_sha`
  fails the batch and cannot count as a verified checkpoint.
- A workflow that reports success without a verified durable record/no-op
  checkpoint becomes `failed`. A successful record without its durable
  nonnegative integer remaining count and a no-op without both explicit zeros
  are unverified. Conflicting top-level and persisted counts also fail closed,
  so incomplete or ambiguous work is never counted as a batch.
- User-visible progress is file-based rather than batch-based. Frozen campaigns
  show resolved files against the selected scope; legacy campaigns use fully
  reviewed plus unsupported files against those counters plus remaining files.
  A non-completed campaign with no durable file counters shows zero even when it
  has batch telemetry or finding occurrences, while a completed all-prechecked
  campaign reports 100 percent. Fully reviewed files require acknowledgement by
  every required reviewer, so findings from partial inspection can exist before
  a file contributes to resolved progress.
- A failed campaign remains explicitly resumable through the commit-selection
  fence. Resume appends a workflow run to that campaign and preserves its
  counters and durable checkpoints only when the frozen profile and chosen
  exact commit are unchanged; a changed profile or latest/custom commit starts
  a new campaign. Run again remains the explicit new-campaign reset. Neither
  action retries a provider call from an uncertain midpoint.
- Accepted findings/coverage are scoped to the automation campaign. Approximate
  model coverage is monotonic and explicitly labeled; internal sketches are
  excluded from frequently polled API responses.
- A review run with no checkpointed finding is a valid empty in-progress
  Findings state. A finding outside the automation's recorded run IDs and campaign start
  never leaks into current scope, even when it shares a repository or was
  updated during the campaign. The repository-finding collection remains bounded
  and paged on its repository-owned route.
- Empty, implicit, duplicate, unknown, dismissed, posted, or already-associated
  finding selections cannot start generation or linking. More than 200 explicit
  IDs fail before any reservation. Concurrent requests have at most one winner
  for each canonical finding association.
- A malformed or schema-invalid writer response becomes a safe `failed`
  generation without retaining provider text. Partial batch failure does not
  roll back successful previews. Failed regeneration never erases the last
  valid preview. Only `editing` and failed unpublished previews are deletable,
  and deletion atomically clears the finding reference.
- Legacy grouped drafts remain readable even though new drafts are one finding
  each. Their safe blocker projection counts every unresolved finding and
  duplicate decision; a noncanonical legacy conflict cannot be edited into a
  publishable record, and backfill ordering is deterministic across repeated
  loads.
- Candidate discovery bounds each derived query and its merged results. Missing,
  malformed, deleted, or cross-repository selected issues fail final re-fetch;
  a nonqualifying ranking never creates a link, and the same valid existing
  issue may be linked to several findings.
- Publication transport ambiguity becomes `unknown`; retries reconcile the
  stable marker and never blindly create a duplicate issue. Per-preview safe
  failures do not hide successes in the same selected subset. `posted` is never
  accepted as any direct review-finding status mutation.
- Local and non-GitHub repositories may generate, edit, regenerate, and delete
  previews but cannot search, link, publish, or construct a GitHub composer
  handoff.
- Purge and removal are blocked while review, deduplication, repository mapping,
  resolution checking, issue generation/publication, or historical
  consolidation is active. A stale automation/ledger version, incorrect typed
  repository identity, unsafe intent marker, or incompatible on-disk target
  fails before deletion. External issue associations are warnings and counts,
  not blockers, because no external issue is changed. An interrupted phase is
  completed idempotently on startup before review workers start.

## Acceptance Evidence

| Requirement IDs     | Evidence                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `FR-REPOREVIEW-001` | [pkg/repoaudit/store_test.go](../../pkg/repoaudit/store_test.go), [pkg/repoaudit/ensemble_test.go](../../pkg/repoaudit/ensemble_test.go)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `FR-REPOREVIEW-002` | [web/backend/api/repository_reviews_test.go](../../web/backend/api/repository_reviews_test.go), [web/frontend/src/api/repository-reviews.test.ts](../../web/frontend/src/api/repository-reviews.test.ts)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `FR-REPOREVIEW-003` | [web/backend/api/repository_review_details_test.go](../../web/backend/api/repository_review_details_test.go), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx), [web/frontend/src/routes/-repository-reviews-route.test.tsx](../../web/frontend/src/routes/-repository-reviews-route.test.tsx)                                                                                                                                                                                                                                                                                                                                                                                               |
| `FR-REPOREVIEW-004` | [web/backend/api/repository_review_details_test.go](../../web/backend/api/repository_review_details_test.go), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx), [web/frontend/src/components/repository-reviews/repository-review-actions.ts](../../web/frontend/src/components/repository-reviews/repository-review-actions.ts)                                                                                                                                                                                                                                                                                                                                                             |
| `FR-REPOREVIEW-005` | [pkg/repoaudit/issues_test.go](../../pkg/repoaudit/issues_test.go), [web/backend/api/repository_review_details_test.go](../../web/backend/api/repository_review_details_test.go), [web/backend/api/repository_review_collections_test.go](../../web/backend/api/repository_review_collections_test.go), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx)                                                                                                                                                                                                                                                                                                                                     |
| `FR-REPOREVIEW-006` | [pkg/repoaudit/publication_test.go](../../pkg/repoaudit/publication_test.go), [pkg/gateway/repository_review_publication_eligibility_test.go](../../pkg/gateway/repository_review_publication_eligibility_test.go), [pkg/gateway/repository_review_publication_test.go](../../pkg/gateway/repository_review_publication_test.go), [web/backend/api/repository_review_publication_eligibility_test.go](../../web/backend/api/repository_review_publication_eligibility_test.go), [web/backend/api/repository_review_details_test.go](../../web/backend/api/repository_review_details_test.go), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx) |
| `FR-REPOREVIEW-007` | [pkg/repoaudit/store_test.go](../../pkg/repoaudit/store_test.go), [pkg/repoaudit/ensemble_test.go](../../pkg/repoaudit/ensemble_test.go)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `FR-REPOREVIEW-008` | [pkg/repoaudit/profile_test.go](../../pkg/repoaudit/profile_test.go), [pkg/repoaudit/control_test.go](../../pkg/repoaudit/control_test.go), [web/backend/api/repository_review_profiles_test.go](../../web/backend/api/repository_review_profiles_test.go), [web/frontend/src/api/repository-reviews.test.ts](../../web/frontend/src/api/repository-reviews.test.ts), [web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx)                                                                                                                                                                                                                                                                     |
| `FR-REPOREVIEW-009` | [web/backend/api/repository_review_automations_test.go](../../web/backend/api/repository_review_automations_test.go), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx), [web/frontend/src/components/repository-reviews/repository-review-runs-page.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-runs-page.test.tsx), [web/frontend/src/components/repository-reviews/repository-review-repositories-page.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-repositories-page.test.tsx)                                                                                                                                   |
| `FR-REPOREVIEW-010` | [pkg/agent/workflow_managed_ensemble_test.go](../../pkg/agent/workflow_managed_ensemble_test.go), [pkg/agent/workflow_runtime_test.go](../../pkg/agent/workflow_runtime_test.go), [web/backend/api/repository_review_automations_test.go](../../web/backend/api/repository_review_automations_test.go)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `FR-REPOREVIEW-011` | [pkg/repoaudit/guard_expression_test.go](../../pkg/repoaudit/guard_expression_test.go), [pkg/repoaudit/guard_migration_test.go](../../pkg/repoaudit/guard_migration_test.go), [web/backend/api/repository_review_automations_test.go](../../web/backend/api/repository_review_automations_test.go), [web/backend/api/codex_account_limits_test.go](../../web/backend/api/codex_account_limits_test.go), [web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx)                                                                                                                                                                                                                                   |
| `FR-REPOREVIEW-012` | [pkg/providers/cli/github_copilot_provider_test.go](../../pkg/providers/cli/github_copilot_provider_test.go), [web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx), [web/frontend/src/components/model-evaluations/model-evaluations-page.test.tsx](../../web/frontend/src/components/model-evaluations/model-evaluations-page.test.tsx)                                                                                                                                                                                                                                                                                                                                                       |
| `FR-REPOREVIEW-013` | [web/backend/api/repository_review_automations_test.go](../../web/backend/api/repository_review_automations_test.go), [pkg/repoaudit/control_test.go](../../pkg/repoaudit/control_test.go)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `FR-REPOREVIEW-014` | [pkg/repoaudit/scope_policy_test.go](../../pkg/repoaudit/scope_policy_test.go), [pkg/workflows/repository_model_evaluation_native_test.go](../../pkg/workflows/repository_model_evaluation_native_test.go), [web/backend/api/repository_review_automations_test.go](../../web/backend/api/repository_review_automations_test.go), [web/frontend/src/api/repository-reviews.test.ts](../../web/frontend/src/api/repository-reviews.test.ts), [web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx)                                                                                                                                                                                               |
| `FR-REPOREVIEW-015` | [pkg/repoaudit/profile_test.go](../../pkg/repoaudit/profile_test.go), [web/backend/api/repository_review_automations_test.go](../../web/backend/api/repository_review_automations_test.go), [web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx), [web/frontend/src/components/repository-reviews/repository-review-repositories-page.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-repositories-page.test.tsx), [web/frontend/src/routes/-repository-reviews-route.test.tsx](../../web/frontend/src/routes/-repository-reviews-route.test.tsx), [web/frontend/src/components/app-sidebar.test.tsx](../../web/frontend/src/components/app-sidebar.test.tsx) |
| `FR-REPOREVIEW-016` | [pkg/workflows/repository_bug_finder_workflow_test.go](../../pkg/workflows/repository_bug_finder_workflow_test.go), [pkg/workflows/repository_review_native_test.go](../../pkg/workflows/repository_review_native_test.go), [web/backend/api/repository_review_details_test.go](../../web/backend/api/repository_review_details_test.go), [pkg/agent/workflow_runtime_test.go](../../pkg/agent/workflow_runtime_test.go)                                                                                                                                                                                                                                                                                                                                                                                                           |
| `FR-REPOREVIEW-017` | [web/backend/api/repository_review_details_test.go](../../web/backend/api/repository_review_details_test.go), [web/backend/api/repository_review_collections_test.go](../../web/backend/api/repository_review_collections_test.go), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx), [web/frontend/src/routes/-repository-reviews-route.test.tsx](../../web/frontend/src/routes/-repository-reviews-route.test.tsx), [web/frontend/tests/ui-smoke.spec.ts](../../web/frontend/tests/ui-smoke.spec.ts), [web/frontend/tests/collection-visual.spec.ts](../../web/frontend/tests/collection-visual.spec.ts) |
| `FR-REPOREVIEW-018` | [pkg/repoaudit/profile_test.go](../../pkg/repoaudit/profile_test.go), [pkg/repoaudit/control_test.go](../../pkg/repoaudit/control_test.go), [web/backend/api/repository_review_profiles_test.go](../../web/backend/api/repository_review_profiles_test.go), [web/backend/api/repository_review_details_test.go](../../web/backend/api/repository_review_details_test.go), [web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx)                                                                                                                                                                                                                                                                 |
| `FR-REPOREVIEW-019` | [pkg/repoaudit/issues_test.go](../../pkg/repoaudit/issues_test.go), [web/backend/api/repository_review_details_test.go](../../web/backend/api/repository_review_details_test.go), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx), [web/frontend/tests/collection-visual.spec.ts](../../web/frontend/tests/collection-visual.spec.ts) |
| `FR-REPOREVIEW-020` | [pkg/gateway/repository_review_publication_test.go](../../pkg/gateway/repository_review_publication_test.go), [web/backend/api/repository_review_details_test.go](../../web/backend/api/repository_review_details_test.go), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx)                                                                                                                                                                                                                                                                                                                                                                                                                 |
| `FR-REPOREVIEW-021` | [pkg/repoaudit/coverage_boundaries_test.go](../../pkg/repoaudit/coverage_boundaries_test.go), [pkg/workflows/repository_bug_finder_workflow_test.go](../../pkg/workflows/repository_bug_finder_workflow_test.go), [pkg/workflows/repository_review_native_test.go](../../pkg/workflows/repository_review_native_test.go)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `FR-REPOREVIEW-022` | [pkg/repoaudit/lifecycle_test.go](../../pkg/repoaudit/lifecycle_test.go), [pkg/repoaudit/mapping_worker_test.go](../../pkg/repoaudit/mapping_worker_test.go)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `FR-REPOREVIEW-023` | [pkg/repoaudit/matching_test.go](../../pkg/repoaudit/matching_test.go), [pkg/repoaudit/mapping_worker_test.go](../../pkg/repoaudit/mapping_worker_test.go), [web/backend/api/repository_review_mapping_validation_coverage_test.go](../../web/backend/api/repository_review_mapping_validation_coverage_test.go), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx), [web/frontend/tests/collection-visual.spec.ts](../../web/frontend/tests/collection-visual.spec.ts) |
| `FR-REPOREVIEW-024` | [pkg/repoaudit/lifecycle_test.go](../../pkg/repoaudit/lifecycle_test.go), [pkg/workflows/repository_review_native_test.go](../../pkg/workflows/repository_review_native_test.go)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `FR-REPOREVIEW-025` | [pkg/repoaudit/profile_test.go](../../pkg/repoaudit/profile_test.go), [pkg/repoaudit/issues_test.go](../../pkg/repoaudit/issues_test.go), [web/backend/api/repository_review_details_test.go](../../web/backend/api/repository_review_details_test.go), [web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx)                                                                                                                                                                                                                                                                                                                                                                                   |
| `FR-REPOREVIEW-026` | [pkg/repoaudit/lifecycle_test.go](../../pkg/repoaudit/lifecycle_test.go), [pkg/gateway/repository_review_publication_test.go](../../pkg/gateway/repository_review_publication_test.go), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx)                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| `FR-REPOREVIEW-027` | [pkg/repoaudit/lifecycle_test.go](../../pkg/repoaudit/lifecycle_test.go), [pkg/repoaudit/validation_worker_test.go](../../pkg/repoaudit/validation_worker_test.go), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx), [web/frontend/tests/collection-visual.spec.ts](../../web/frontend/tests/collection-visual.spec.ts) |
| `FR-REPOREVIEW-028` | [pkg/repoaudit/identity_test.go](../../pkg/repoaudit/identity_test.go), [pkg/repoaudit/campaign_test.go](../../pkg/repoaudit/campaign_test.go), [web/backend/api/repository_review_details_test.go](../../web/backend/api/repository_review_details_test.go), [web/frontend/src/routes/-repository-reviews-route.test.tsx](../../web/frontend/src/routes/-repository-reviews-route.test.tsx)                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `FR-REPOREVIEW-029` | [pkg/repoaudit/campaign_foundation_test.go](../../pkg/repoaudit/campaign_foundation_test.go), [pkg/repoaudit/store_test.go](../../pkg/repoaudit/store_test.go), [pkg/repoaudit/coverage_boundaries_test.go](../../pkg/repoaudit/coverage_boundaries_test.go), [pkg/workflows/repository_review_campaign_write_test.go](../../pkg/workflows/repository_review_campaign_write_test.go), [pkg/workflows/repository_review_campaign_privacy_test.go](../../pkg/workflows/repository_review_campaign_privacy_test.go), [web/backend/api/repository_review_campaign_backfill_test.go](../../web/backend/api/repository_review_campaign_backfill_test.go), [web/backend/api/repository_review_automations_test.go](../../web/backend/api/repository_review_automations_test.go), [web/backend/api/repository_review_coverage_test.go](../../web/backend/api/repository_review_coverage_test.go), [web/backend/api/repository_reviews_test.go](../../web/backend/api/repository_reviews_test.go), [pkg/gateway/repository_review_issue_sync_coverage_test.go](../../pkg/gateway/repository_review_issue_sync_coverage_test.go) |
| `FR-REPOREVIEW-030` | [pkg/repoaudit/assignment_run_test.go](../../pkg/repoaudit/assignment_run_test.go), [pkg/repoaudit/assignment_timeout_test.go](../../pkg/repoaudit/assignment_timeout_test.go), [pkg/repoaudit/file_attribution_test.go](../../pkg/repoaudit/file_attribution_test.go), [pkg/agent/workflow_managed_ensemble_test.go](../../pkg/agent/workflow_managed_ensemble_test.go), [web/backend/api/repository_review_campaign_backfill_test.go](../../web/backend/api/repository_review_campaign_backfill_test.go), [web/backend/api/repository_review_file_attribution_backfill_test.go](../../web/backend/api/repository_review_file_attribution_backfill_test.go), [web/backend/api/repository_review_file_attributions_test.go](../../web/backend/api/repository_review_file_attributions_test.go), [web/backend/api/repository_review_automations_test.go](../../web/backend/api/repository_review_automations_test.go), [web/frontend/src/api/repository-reviews.test.ts](../../web/frontend/src/api/repository-reviews.test.ts), [web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-profiles-page.test.tsx), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx) |
| `FR-REPOREVIEW-031` | [pkg/repoaudit/campaign_test.go](../../pkg/repoaudit/campaign_test.go), [pkg/repoaudit/deduplication_schema_test.go](../../pkg/repoaudit/deduplication_schema_test.go), [pkg/repoaudit/deduplication_engine_test.go](../../pkg/repoaudit/deduplication_engine_test.go), [pkg/repoaudit/deduplication_worker_test.go](../../pkg/repoaudit/deduplication_worker_test.go), [pkg/repoaudit/lifecycle_test.go](../../pkg/repoaudit/lifecycle_test.go), [web/backend/api/repository_review_dedup_test.go](../../web/backend/api/repository_review_dedup_test.go), [web/backend/api/repository_review_dedup_additional_coverage_test.go](../../web/backend/api/repository_review_dedup_additional_coverage_test.go), [web/frontend/src/api/repository-reviews.test.ts](../../web/frontend/src/api/repository-reviews.test.ts), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx), [web/frontend/src/routes/-repository-reviews-route.test.tsx](../../web/frontend/src/routes/-repository-reviews-route.test.tsx), [web/frontend/tests/ui-smoke.spec.ts](../../web/frontend/tests/ui-smoke.spec.ts) |
| `FR-REPOREVIEW-032` | [pkg/repoaudit/historical_replay_test.go](../../pkg/repoaudit/historical_replay_test.go), [pkg/repoaudit/historical_replay_additional_coverage_test.go](../../pkg/repoaudit/historical_replay_additional_coverage_test.go), [web/backend/api/repository_review_campaign_backfill_test.go](../../web/backend/api/repository_review_campaign_backfill_test.go), [web/backend/api/repository_review_dedup_test.go](../../web/backend/api/repository_review_dedup_test.go), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx) |
| `FR-REPOREVIEW-033` | [pkg/repoaudit/deduplication_worker_test.go](../../pkg/repoaudit/deduplication_worker_test.go), [web/backend/api/repository_review_dedup_new_coverage_test.go](../../web/backend/api/repository_review_dedup_new_coverage_test.go), [web/backend/api/repository_review_health_test.go](../../web/backend/api/repository_review_health_test.go), [web/frontend/src/api/repository-reviews.test.ts](../../web/frontend/src/api/repository-reviews.test.ts), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx), [web/frontend/src/routes/-repository-reviews-route.test.tsx](../../web/frontend/src/routes/-repository-reviews-route.test.tsx), [web/frontend/tests/ui-smoke.spec.ts](../../web/frontend/tests/ui-smoke.spec.ts), [web/frontend/tests/collection-visual.spec.ts](../../web/frontend/tests/collection-visual.spec.ts) |
| `FR-REPOREVIEW-034` | [pkg/repoaudit/deduplication_schema_test.go](../../pkg/repoaudit/deduplication_schema_test.go), [web/backend/api/repository_review_dedup_test.go](../../web/backend/api/repository_review_dedup_test.go), [web/backend/api/repository_review_collections_test.go](../../web/backend/api/repository_review_collections_test.go), [web/frontend/src/api/repository-reviews.test.ts](../../web/frontend/src/api/repository-reviews.test.ts) |
| `FR-REPOREVIEW-035` | [web/backend/api/repository_review_automations_test.go](../../web/backend/api/repository_review_automations_test.go), [web/backend/api/repository_review_assignment_coverage_test.go](../../web/backend/api/repository_review_assignment_coverage_test.go), [pkg/repoaudit/campaign_foundation_test.go](../../pkg/repoaudit/campaign_foundation_test.go) |
| `FR-REPOREVIEW-036` | [pkg/repoaudit/assignment_run_test.go](../../pkg/repoaudit/assignment_run_test.go), [pkg/repoaudit/file_attribution_test.go](../../pkg/repoaudit/file_attribution_test.go), [web/backend/api/repository_review_file_attributions_test.go](../../web/backend/api/repository_review_file_attributions_test.go) |
| `FR-REPOREVIEW-037` | [pkg/repoaudit/deduplication_worker_test.go](../../pkg/repoaudit/deduplication_worker_test.go), [pkg/repoaudit/historical_replay_test.go](../../pkg/repoaudit/historical_replay_test.go), [web/backend/api/repository_review_dedup_additional_coverage_test.go](../../web/backend/api/repository_review_dedup_additional_coverage_test.go), [web/backend/api/repository_review_historical_restart_test.go](../../web/backend/api/repository_review_historical_restart_test.go) |
| `FR-REPOREVIEW-038` | [web/backend/api/repository_review_health_test.go](../../web/backend/api/repository_review_health_test.go), [web/backend/api/repository_review_processing_test.go](../../web/backend/api/repository_review_processing_test.go), [web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-routed-pages.test.tsx) |
| `FR-REPOREVIEW-039` | [pkg/repoaudit/profile_test.go](../../pkg/repoaudit/profile_test.go), [pkg/repoaudit/issues_test.go](../../pkg/repoaudit/issues_test.go), [web/backend/api/repository_review_details_test.go](../../web/backend/api/repository_review_details_test.go), [pkg/gateway/repository_review_publication_test.go](../../pkg/gateway/repository_review_publication_test.go) |
| `FR-REPOREVIEW-040` | [pkg/repoaudit/lifecycle_test.go](../../pkg/repoaudit/lifecycle_test.go), [pkg/repoaudit/validation_worker_test.go](../../pkg/repoaudit/validation_worker_test.go), [web/backend/api/repository_review_mapping_validation_coverage_test.go](../../web/backend/api/repository_review_mapping_validation_coverage_test.go) |
| `FR-REPOREVIEW-041` | [pkg/repoaudit/purge_test.go](../../pkg/repoaudit/purge_test.go), [web/backend/api/repository_review_automations_test.go](../../web/backend/api/repository_review_automations_test.go), [web/backend/api/repository_review_coverage_test.go](../../web/backend/api/repository_review_coverage_test.go), [web/frontend/src/components/repository-reviews/repository-review-repositories-page.test.tsx](../../web/frontend/src/components/repository-reviews/repository-review-repositories-page.test.tsx) |
| `FR-REPOREVIEW-042` | [web/backend/api/repository_review_details_test.go](../../web/backend/api/repository_review_details_test.go), [web/backend/api/repository_review_collections_test.go](../../web/backend/api/repository_review_collections_test.go), [web/backend/api/repository_review_route_docs_test.go](../../web/backend/api/repository_review_route_docs_test.go), [web/backend/api/repository_reviews_test.go](../../web/backend/api/repository_reviews_test.go), [docs/reference/repository-reviews-api.md](../reference/repository-reviews-api.md) |
| `FR-REPOREVIEW-043` | [pkg/workflows/repository_bug_finder_workflow_test.go](../../pkg/workflows/repository_bug_finder_workflow_test.go), [pkg/workflows/repository_review_native_test.go](../../pkg/workflows/repository_review_native_test.go), [web/frontend/tests/ui-smoke.spec.ts](../../web/frontend/tests/ui-smoke.spec.ts), [web/frontend/tests/collection-visual.spec.ts](../../web/frontend/tests/collection-visual.spec.ts) |

## Implementation Anchors

- [pkg/repoaudit/control.go](../../pkg/repoaudit/control.go)
- [pkg/repoaudit/assignment.go](../../pkg/repoaudit/assignment.go)
- [pkg/repoaudit/assignment_run.go](../../pkg/repoaudit/assignment_run.go)
- [pkg/repoaudit/file_attribution.go](../../pkg/repoaudit/file_attribution.go)
- [pkg/repoaudit/deduplication_model.go](../../pkg/repoaudit/deduplication_model.go)
- [pkg/repoaudit/deduplication_engine.go](../../pkg/repoaudit/deduplication_engine.go)
- [pkg/repoaudit/deduplication_worker.go](../../pkg/repoaudit/deduplication_worker.go)
- [pkg/repoaudit/historical_replay.go](../../pkg/repoaudit/historical_replay.go)
- [pkg/repoaudit/guard_expression.go](../../pkg/repoaudit/guard_expression.go)
- [pkg/repoaudit/profile.go](../../pkg/repoaudit/profile.go)
- [pkg/repoaudit/issues.go](../../pkg/repoaudit/issues.go)
- [pkg/repoaudit/publication.go](../../pkg/repoaudit/publication.go)
- [pkg/repoaudit/purge.go](../../pkg/repoaudit/purge.go)
- [pkg/repoaudit/scope_policy.go](../../pkg/repoaudit/scope_policy.go)
- [pkg/repoaudit/store.go](../../pkg/repoaudit/store.go)
- [pkg/repoaudit/identity.go](../../pkg/repoaudit/identity.go)
- [pkg/repoaudit/matching.go](../../pkg/repoaudit/matching.go)
- [pkg/repoaudit/mapping_worker.go](../../pkg/repoaudit/mapping_worker.go)
- [pkg/repoaudit/lifecycle.go](../../pkg/repoaudit/lifecycle.go)
- [pkg/repoaudit/validation_worker.go](../../pkg/repoaudit/validation_worker.go)
- [pkg/workflows/templates.go](../../pkg/workflows/templates.go)
- [pkg/workflows/native_functions.go](../../pkg/workflows/native_functions.go)
- [pkg/agent/workflow_managed.go](../../pkg/agent/workflow_managed.go)
- [web/backend/api/repository_review_controller.go](../../web/backend/api/repository_review_controller.go)
- [web/backend/api/repository_review_automations.go](../../web/backend/api/repository_review_automations.go)
- [web/backend/api/repository_review_details.go](../../web/backend/api/repository_review_details.go)
- [web/backend/api/repository_review_dedup.go](../../web/backend/api/repository_review_dedup.go)
- [web/backend/api/repository_review_deduplication.go](../../web/backend/api/repository_review_deduplication.go)
- [web/backend/api/repository_review_historical_deduplication.go](../../web/backend/api/repository_review_historical_deduplication.go)
- [web/backend/api/repository_review_file_attribution_backfill.go](../../web/backend/api/repository_review_file_attribution_backfill.go)
- [web/backend/api/repository_review_file_attributions.go](../../web/backend/api/repository_review_file_attributions.go)
- [web/backend/api/repository_review_collections.go](../../web/backend/api/repository_review_collections.go)
- [web/backend/api/repository_review_gateway.go](../../web/backend/api/repository_review_gateway.go)
- [web/backend/api/repository_review_mapping.go](../../web/backend/api/repository_review_mapping.go)
- [web/backend/api/repository_review_validation.go](../../web/backend/api/repository_review_validation.go)
- [web/backend/api/repository_review_lifecycle.go](../../web/backend/api/repository_review_lifecycle.go)
- [web/backend/api/repository_reviews.go](../../web/backend/api/repository_reviews.go)
- [Repository Reviews API reference](../reference/repository-reviews-api.md)
- [pkg/gateway/repository_review_publication.go](../../pkg/gateway/repository_review_publication.go)
- [pkg/gateway/repository_review_issue_links.go](../../pkg/gateway/repository_review_issue_links.go)
- [web/frontend/src/components/repository-reviews/repository-review-profiles-page.tsx](../../web/frontend/src/components/repository-reviews/repository-review-profiles-page.tsx)
- [web/frontend/src/components/repository-reviews/repository-review-repositories-page.tsx](../../web/frontend/src/components/repository-reviews/repository-review-repositories-page.tsx)
- [web/frontend/src/components/repository-reviews/repository-review-detail-page.tsx](../../web/frontend/src/components/repository-reviews/repository-review-detail-page.tsx)
- [web/frontend/src/components/repository-reviews/repository-review-findings-page.tsx](../../web/frontend/src/components/repository-reviews/repository-review-findings-page.tsx)
- [web/frontend/src/components/repository-reviews/repository-review-finding-page.tsx](../../web/frontend/src/components/repository-reviews/repository-review-finding-page.tsx)
- [web/frontend/src/components/repository-reviews/repository-review-link-issue-page.tsx](../../web/frontend/src/components/repository-reviews/repository-review-link-issue-page.tsx)
- [web/frontend/src/components/repository-reviews/repository-review-issues-page.tsx](../../web/frontend/src/components/repository-reviews/repository-review-issues-page.tsx)
- [web/frontend/src/components/repository-reviews/repository-review-issue-page.tsx](../../web/frontend/src/components/repository-reviews/repository-review-issue-page.tsx)
