# Branch Structure and Upstream Sync Workflow

This repository is a fork of [sipeed/picoclaw](https://github.com/sipeed/picoclaw).

## Remotes

| Remote   | URL                                          | Purpose             |
|----------|----------------------------------------------|---------------------|
| `origin` | github.com:dimonb/picoclaw.git               | fork (this repo)    |
| `upstream` | github.com:sipeed/picoclaw.git             | upstream source     |

## Branches

### `main`
Fork-maintained main branch based on `upstream/main`, with a small set of
fork-local repository files such as this `AGENTS.md`.
Keep it linear: rebase `main` onto `upstream/main` when upstream moves.
Do not merge `upstream/main` into `main`.
Do not replace `main` with `upstream/main` via `reset --hard`, because that
drops fork-local commits and files.

### `dev`
The integration branch containing all local features on top of upstream.
This is the working branch — build, test, and run from here.
Created by: `git checkout -b dev backup/main-2026-03-17` then `git merge main`.

### `local/0X-*` — Feature Marker Branches
Read-only sha markers that identify where each feature group ends in the `dev` history.
They are never force-pushed; they just point to a specific commit for reference.

| Branch                    | Tip commit | Feature group                                              |
|---------------------------|------------|------------------------------------------------------------|
| `local/01-message-history`| `b7db0c9`  | Store sender first/last name and message author in session |
| `local/02-compaction`     | `c7fde1b`  | Thread-aware compaction, summary keep count, race fixes    |
| `local/03-telegram`       | `89eeb8f`  | Telegram reply routing, forum topics, HTML tag handling    |
| `local/04-cron`           | `f9071ec`  | Cron mode parameter, session binding, missed job startup   |
| `local/05-concurrency`    | `d072cbc`  | Per-session parallel processing, dispatcher key fix        |
| `local/06-media`          | `875a783`  | Persistent media store, resolved workspace path            |
| `local/07-tasktool`       | `4f8e753`  | TaskTool fixes: plan ordering, session key, serialization  |
| `local/08-reaction`       | `c169b64`  | Reaction tool with typing/placeholder + also_reply param   |
| `local/09-providers`      | `d12fa3b`  | OpenAI responses API, provider normalisation               |

### `backup/main-2026-03-17`
Snapshot of the fork's `main` branch taken on 2026-03-17, before the reorganisation.
Kept as a safety net. Do not delete.

---

## Syncing Upstream Changes into `main` and `dev`

Run this whenever `upstream/main` has moved forward:

```bash
# 1. Fetch upstream
git fetch upstream

# 2. Rebase our main onto upstream/main
git checkout main
git rebase upstream/main

# 2.1 Verify fork-local files are still present on main
test -f AGENTS.md

git push --force-with-lease origin main

# 3. Merge refreshed main into dev (resolve conflicts once, then commit)
git checkout dev
git merge main

# 4. Run tests
go test ./...

# 5. Push
git push origin dev
```

### Conflict resolution hints

| File                        | Our change                                  | Strategy                                |
|-----------------------------|---------------------------------------------|-----------------------------------------|
| `pkg/gateway/gateway.go`    | `PersistentFileMediaStoreWithCleanup`       | Keep ours, adopt upstream variable names|
| `pkg/tools/cron.go`         | `allowRemote`, mode-based dispatch          | Merge both feature sets                 |
| `pkg/tools/cron_test.go`    | Additional mode/session tests               | Keep both test suites                   |
| `pkg/agent/loop.go`         | `agentResponse`, `OnDelivered` callback     | Keep ours, pull in upstream additions   |
| `pkg/agent/context.go`      | Extended context keys                       | Keep ours, pull in upstream additions   |

### After a large upstream rebase or merge into `dev`

If a file has deep conflicts, compare with `git diff upstream/main...HEAD -- <file>`
and resolve by applying our semantic change on top of the new upstream version.

If `AGENTS.md` or another fork-local file disappears during sync, restore it
before pushing `main`.

---

## Local-only files (not in upstream)

- `pkg/agent/summarization_prompts.go` — thread-aware compaction prompts
- `pkg/media/store.go` — `PersistentFileMediaStore` with cleanup
- `pkg/tools/reaction.go` — reaction tool
- `AGENTS.md` — this file; it must remain present on `main` after every upstream sync
