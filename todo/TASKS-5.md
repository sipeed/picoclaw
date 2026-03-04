# TASKS-5: Heartbeat Worktree Management

ハートビートで作られる git worktree の管理 CLI/UI。他トラックへの依存なし。

## 現状の問題

- ハートビートセッションが `.worktrees/heartbeat-YYYYMMDD/` を作るが、人手で管理する手段がない
- `PruneOrphaned()` は起動時にディレクトリを削除するが、**未コミット変更は通知なく消失**する
- active worktree の一覧・点検・手動 merge/dispose ができない

---

## タスク一覧

### 1. `/plan worktrees` コマンド

worktree の一覧表示と操作を提供する CLI コマンド。

**サブコマンド:**
- `/plan worktrees list` — active worktree の一覧 (branch, last commit, status, uncommitted changes の有無)
- `/plan worktrees inspect <name>` — 特定 worktree の詳細 (diff, log)
- `/plan worktrees merge <name>` — main branch に merge を試行
- `/plan worktrees dispose <name>` — worktree を削除 (未コミット変更がある場合は確認)

**対象ファイル:** `pkg/agent/loop.go` (コマンドハンドラ追加), `pkg/git/worktree.go`

---

### 2. PruneOrphaned の安全化

起動時の orphan prune で未コミット変更を保護する。

**やること:**
- prune 前に `git status` で未コミット変更を検出
- 未コミット変更がある場合: auto-commit (メッセージ: "auto-save before prune") してから dispose
- auto-commit 結果をログに出力
- commit 不可の場合 (conflict 等) はスキップしてログ警告

**対象ファイル:** `pkg/git/worktree.go`

---

### 3. Mini App worktree UI

Mini App から worktree を閲覧・操作できる画面。

**やること:**
- `/api/worktrees` エンドポイント (GET: list, POST: merge/dispose)
- Mini App に worktree 一覧パネルを追加
- 各 worktree: branch 名、最終コミット日時、uncommitted changes badge
- merge/dispose ボタン (確認ダイアログ付き)

**対象ファイル:** `pkg/miniapp/handler.go`, `pkg/miniapp/static/index.html`

---

## 完了基準

- `go test ./...` 全パス
- `/plan worktrees list` で active worktree が表示される
- orphan prune 時に未コミット変更が auto-commit で保護される
- Mini App から worktree の一覧と dispose が操作できる
