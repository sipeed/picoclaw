# Repository Bug Finder

PicoClaw's repository bug finder reviews one exact Git commit, checkpoints
validated findings as bounded batches finish, and skips unchanged Git blobs on
later runs. The dashboard keeps the review lifecycle, live findings, finding
evidence, AI-written issue previews, and canonical GitHub issue association on
directly addressable routes.

## Configure A Review

Open **Repository reviews**. Configuration is split into two reusable resource
types:

1. In **Profiles**, use the shared list, table, or grid collection to create the
   review policy on its dedicated editor route. Choose an execution account,
   one passive reviewer alias, and optionally an **Issue writer** alias. A blank
   issue writer means “same as reviewer.” An explicit writer must be available
   on the selected account and cannot use an agentic CLI provider. Profile
   detail and edit links are stable and Browser Back restores the collection
   query, view, and scroll position.
2. Set the review focus and Scope: one or more code types, optional exact
   repository-relative include and exclude folder prefixes, and optional
   guidance that may narrow but never expand the structured boundary.
3. In **Advanced**, set bounded files per batch, content bytes per provider
   request, parallel review workers, automatic continuation, and an optional
   task-admission guard expression. You may also select a passive deduplication
   alias (blank means the reviewer), a similarity threshold (default 90), and a
   candidate limit (default 4; zero promotes each raw finding separately).
   Profiles default to eight review workers.
4. In **Repositories**, assign that profile to one repository and optionally a
   branch. Blank follows the acquired repository's advertised default branch.

Profiles do not contain repository URLs, mutable refs, prices, account lists,
polling intervals, or separate token/cost/quota switches. Repository branch
configuration accepts a branch name only; it rejects `HEAD`, commit hashes,
tags, full refs, revision expressions, URLs, and query or fragment forms.

The execution account appears before reviewer and issue-writer selection
because it constrains both model lists. Blank account follows the runtime
default when a campaign starts. PicoClaw snapshots review execution policy into
the campaign, and every review call uses that frozen account. Each later Draft,
direct Post generation, issue-candidate ranking, or regeneration resolves the
currently assigned profile/account once for that attempt. Generated or
regenerated previews durably store prompt, writer, account, profile ID, and
version; ranking returns the selected generator model/account but remains
read-only unless a qualifying discovered link is stored. Publishing an existing
preview uses its already frozen payload and provenance.

## Run And Observe

The **Repository reviews** collection contains compact review summaries. Open a
summary to reach its detail page; lifecycle controls and complete operational
state do not expand inside the collection.

From review detail you can:

- **Start review**, **Stop safely**, **Continue review**, or **Run again**;
- inspect the exact admitted commit, live workflow stage, bounded-batch
  progress, actual token usage, estimated cost, pause reason, and run history;
- choose the remembered, latest, or a custom full commit SHA when a paused or
  failed campaign's branch has moved; and
- open **Findings**, **Raw findings**, **Findings processing**, or **Issue
  previews** at any time, including before the first batch finishes.

Start resolves the configured branch or remote default to one canonical full
commit SHA and persists it with the workflow run ID before a worker starts.
Automatic batches stay on that commit. **Stop safely** prevents another batch
while allowing already admitted work to checkpoint. **Continue review**
re-enters a paused or failed campaign through the same task guard, preserving
its durable checkpoints, progress, usage, and run history only when both the
assigned profile snapshot and chosen exact commit are unchanged. Choose the
remembered commit to retry unfinished work against the exact source revision
used before the failure. Choosing latest/custom after the branch moves starts a
new campaign, as does a changed profile. **Run again** always begins a new
campaign and resets its campaign progress and accounting; the
repository ledger may still skip matching blob and profile checkpoints unless
the profile uses force mode.

The launcher does not poll quotas or auto-resume a stopped guard. After a
launcher restart, orphaned work becomes an explicit `service_restart` pause and
requires Continue.

Before reviewer calls, PicoClaw inventories the exact commit and classifies the
structured scope. It releases the checkout, asks AI to plan a metadata-only
target filter, reacquires only the pinned commit, and validates the plan
natively against opaque candidate IDs and folder/type boundaries. AI cannot
invent a path, re-include an excluded folder, or select an unchosen code type.

## Purge Or Remove Review Data

Repository-review history is kept indefinitely unless you explicitly remove
it. On an inactive repository detail page, **Purge review history** deletes the
review ledger and resets campaign/runtime state to a fresh idle configuration;
the repository, branch, and assigned profile remain configured. **Remove
repository** always deletes both the assignment and every resolved review
ledger.

Both dialogs show aggregate deletion counts and require typing the exact
displayed normalized repository identity. PicoClaw also checks the current
configuration version, primary ledger version, and opaque composite ledger
fence at confirmation time. If inventory cannot be read, deletion fails closed
with a `retention_unavailable` blocker. If review, deduplication, mapping, a
Resolution check, issue generation/publication, or historical consolidation is
active, the action is blocked with a server-owned reason. Refresh after a
stale-version conflict and review the counts again.

These actions delete PicoClaw Repository Review data only. They do not delete or
modify GitHub issues, do not remove profiles or discussion threads, and do not
delete generic workflow run records; thread- and workflow-owned retention still
applies to those resources. An interrupted purge/removal is completed safely
before repository-review workers start after the next launcher restart.

## Use Live Findings

Repository review has three diagnosis layers:

- **Raw findings** (`rrw_*`) are immutable diagnoses from individual successful
  assignments. They retain the exact commit/blob/context, configured alias,
  concrete model/account, and processing state.
- **Findings** (`rdf_*`) are completed current-campaign diagnoses after raw
  sources have been deduplicated. One finding may represent several ordered raw
  sources while keeping their evidence independently inspectable.
- **Repository findings** (`rrf_*`) join the same causal defect across commits
  and own lifecycle, issue association, and Resolution checks.

**Findings** is always available. Before the first completed deduplication
checkpoint it displays an empty in-progress state rather than substituting raw
or pending evidence. Its typed query/cursor collection supports List, Table,
and Grid and polls only while review or processing work remains active. Open a
finding for its sealed representative diagnosis, raw-source count, commit/blob
provenance, and paged source links.

Open **Raw findings** for current-campaign pending, processing, failed, and
completed `rrw_*` records. Open **Findings processing** for the repository-wide
processing view, including failures from older campaigns. Only failed rows can
be retried; bulk retry accepts 1–200 explicit unique sources and reports ordered
successes plus safe per-source failures without changing immutable diagnosis
content.

Choose **View repository findings** or follow a mapped finding link to inspect
canonical cross-commit aggregates below `/repository-reviews/repositories`.
That list/detail flow owns issue drafting, publication, existing-issue
association, duplicate decisions, Resolution checks, and dismiss/reopen
lifecycle controls. Deprecated Run findings URLs remain compatibility surfaces,
not a separate canonical diagnosis layer.

New review findings also retain causal `match_hints` (component, operation,
failure mode, trigger, violated invariant, outcome, related symbols, source
anchors, and distinguishing facts) so the same defect can be recognized after
a rename or refactor. They include two diagnosis-only effort ranges: **Quick**
containment and **Quality** correction. Each range counts hand-edited additions
plus deletions and is classified by its upper bound: tiny through 10 LOC, small
through 40, medium through 150, large through 500, and refactor above 500 or
for cross-subsystem contract migration. These fields identify and size a
defect; they never contain a fix design, patch, recommendation, or next step.
Older findings display unknown hints and effort and are not re-reviewed merely
to populate them.

Discussion creates a separate reviewing thread seeded with the exact
finding and context provenance. Returning from chat does not generate, link,
or publish an issue; each of those effects still requires an explicit action.

Each Findings, Raw findings, Findings processing, repository-finding, and
issue-preview collection preserves its own query, view, explicit selection,
loaded cursor pages, and in-memory scroll through detail and action routes.
Browser Back restores that state without overwriting the parent collection's
query. Legacy `scope` links still reach the correct campaign or repository
collection; legacy offsets normalize to the first canonical cursor page. The
former **Results** sidebar destination is gone; old
`/repository-reviews/results` links return to the review collection.

## Draft Issue Previews

From a repository's **Repository findings** route, select up to 200 explicit
open, unassociated, non-provisional canonical findings and choose **Draft issue
previews**. Run-finding routes do not expose issue actions. Selection is never
implicit or query-wide. One batch receives one generation ID, creates one
preview per finding, and runs at most four issue-writer calls concurrently
across launcher processes sharing the workspace.
A per-attempt OS lock also prevents two processes from dispatching the same
reservation. A partial failure keeps successful previews
and opens the saved Issue previews route filtered or highlighted by that
generation.

Issue previews use the same List, Table, and Grid collection controls. Open a
preview for its rendered diagnosis and provenance, or use its dedicated Edit
route for version-fenced title, body, and label changes before publication.

Every writer call is private, ephemeral, no-history, no-cache, no-tools, and
structured. A new **Draft issue** or **Post issue** action resolves the currently
assigned profile, then freezes its profile version, issue prompt, issue-writer
alias, and effective account into the saved attempt. The fixed policy permits
grounded diagnosis only. By default, the
writer produces:

- a concise title;
- GitHub-flavored Markdown covering evidence, observable impact, validation
  already performed, and exact location;
- the reviewed commit and blob provenance; and
- a `bug` label.

Optional batch instructions may change presentation, but cannot introduce a
fix, workaround, unsupported fact, invented reproduction, or external lookup.
PicoClaw persists the resolved instructions, mode, generation ID, model/account
provenance, and validated preview. It never stores the raw provider response.

Open a preview to render its Markdown, edit title/body/labels, regenerate it,
delete an unpublished preview, post or reconcile it, or return to its
finding. A failed initial generation remains retryable or deletable. A failed
regeneration keeps the last good preview and its original model/instruction
provenance; the failed attempt remains separately attributable. Deleting an editable or failed
unpublished preview frees its finding for another generation. Posting,
unknown, and posted previews cannot be deleted.

Each finding can have at most one active preview or canonical issue. Retrying
the same generation ID is idempotent, and concurrent attempts cannot reserve
the same finding twice. Older grouped drafts remain visible for history;
conflicting noncanonical legacy drafts are read-only and cannot publish.

## Post To GitHub

For a repository whose acquired `origin` resolves to a canonical
`github.com/owner/repository` identity, select any subset of editable previews
and choose **Post selected**. That explicit click authorizes posting without a
second confirmation dialog. PicoClaw freezes each exact title/body/labels payload
before the protected gateway call. It searches a stable marker before creating
an issue, reports success or a safe failure per preview, and records `unknown`
when a transport outcome may have changed GitHub.

An unknown preview is reconciled by its marker and is never blindly created a
second time. Only successful publication or a validated existing-issue link
sets the canonical association to posted and stores the provider issue ID/URL.
There is no manual **Mark posted** action. An issue created by PicoClaw remains
permanently associated with its finding.

**Post issue** on a finding without a draft generates one saved payload and
immediately posts that exact content. Posting an existing draft never regenerates
or silently edits it.

Local and non-GitHub repositories can still generate, edit, regenerate, and
delete previews, but they expose no post, issue-search, or issue-link action.

## Link An Existing GitHub Issue

From an open, unassociated finding, open **Link existing issue**. You can enter
an issue URL manually or choose **Ask AI to find existing issues**.

Candidate discovery derives bounded GitHub searches from causal hints, stable
symbols, source anchors, path history, and title. The server merges and
deduplicates at most 50 open or closed issues from the same repository. The
current assigned profile's issue-writer model, without tools, ranks and explains
at most 10 candidates. PicoClaw may automatically create a reversible
`discovered` association only for the first-ranked candidate when its score is
at least 95, it has at least four matching causal anchors, and it has no
conflicting anchors. It then re-fetches that exact issue and verifies the exact
repository. A score of 94, three matching anchors, any conflict, or a failed
re-fetch remains available for manual selection without creating a link.

After you select and confirm one issue, PicoClaw re-fetches it and validates
the canonical URL and repository before storing a linked association. A
missing, deleted, malformed, or cross-repository issue is rejected. The same
existing issue may be linked to more than one finding. Unlike issues created by
PicoClaw, a manually linked issue may be unlinked or replaced after explicit
confirmation. A discovered association is reversible in the same way.

## Run A Resolution Check

Issue and code state are shown independently. A closed GitHub issue moves a
repository finding to `resolution_pending`; it does not prove the defect is
fixed. Issue snapshots refresh after 15 minutes when the finding or Resolution
check view opens, before a check, or when you choose **Sync GitHub**. Reopening
the issue returns the finding to `open`.

Choose **Check for fix** for one finding or select up to 50 repository findings
and choose the corresponding Resolution check action. The restart-safe queue
runs at most four validators across launcher processes. PicoClaw considers at
most 200 reachable default-branch commits touching known paths or symbols,
BM25-ranks them, and gives the isolated validator at most eight bounded
diffs/current-source records. The checker can select only those commits, and the
server verifies default-branch ancestry. A confirmed result records the fix
commit and date, check time, and first chronological semantic-version tag
containing it.
Later observation of the same causal defect marks the finding `regressed`
without erasing its earlier resolution history.

## Install And Run From The CLI

Install the built-in workflow once:

```sh
picoclaw workflow install repository-bug-finder
```

Run it with a local checkout or clone URL:

```sh
picoclaw workflow run workflows/repository-bug-finder.yml \
  --inputs '{"repository":"https://github.com/owner/repository.git","ref":"main"}'
```

The default standalone run admits at most 24 pending files, groups related
files into bounded contexts of up to three, and may reduce the effective file
count when several required reviewer aliases are supplied. Inspect
`remainingFiles` in the run output and run the workflow again until it reaches
zero. Unchanged blob SHA/size pairs under the same review profile are not sent
to a model again. A relevant alias, account route, or model configuration
change invalidates that checkpoint.

To challenge every file with several configured reviewer aliases in a direct
workflow run:

```sh
picoclaw workflow run workflows/repository-bug-finder.yml \
  --inputs '{"repository":"https://github.com/owner/repository.git","ref":"main","review_models":"review-a,review-b"}'
```

The profile-backed dashboard deliberately uses one reviewer alias; comparative
testing belongs to **Model review probes**. Direct workflow `review_models`
remains a standalone compatibility input. Every requested alias receives the
same immutable chunks as an independent required reviewer with inherited
fallbacks disabled. Without it, the main agent's configured fallback chain is
used and safe fallback aliases may corroborate findings.

Use passive API-backed reviewer providers. Repository review rejects
`codex-cli` and `claude-cli` aliases because agentic CLIs have local execution
permissions incompatible with the immutable no-tool evidence boundary.

## Bounded Failures

Repository bytes are read only from immutable Git objects into no-tool model
requests. Binary and individually oversized files become visible terminal
unsupported entries. Aggregate-limited or failed files remain retryable and
rotate behind unattempted files. Explicit provider safety/content-filter errors
use bounded request-local handling without marking the provider or account
globally unhealthy; a failed opportunistic corroborator does not block a
successful default fallback chain. Required reviewer failures remain pending.
Each workflow run owns a distinct Git-workspace lease, so concurrent launches
cannot share or release one another's checkout.
