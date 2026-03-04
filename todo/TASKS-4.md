# TASKS-4: Mini App & Static Serving

Mini App のフロントエンド改善。バックエンドの変更に依存しない。

## 現状

```
pkg/miniapp/static/
  index.html   ← すべての JS/CSS をインライン
  map.js       ← オーケストレーションルーム描画
```

Go 側: `//go:embed static/index.html static/map.js` + 個別ルートで配信。

---

## タスク一覧

### 1. Static file serving の汎用化

個別ルート (`serveIndex`, `serveMapJS`) を `http.FileServer(http.FS(staticFS))` に統合。

**やること:**
- `//go:embed static` でディレクトリごと embed
- `/miniapp/` 以下を `http.FileServer` で一括配信
- 新規ファイル追加時にルート登録が不要になる
- `serveIndex` の `ORCH_ENABLED` 注入は `text/template` に移行

**対象ファイル:** `pkg/miniapp/handler.go`

---

### 2. バンドラ導入 (Vite / esbuild)

inline JS/CSS を外部ファイルに分離し、ビルドステップで bundle する。

**やること:**
- `pkg/miniapp/frontend/` に source を配置
- esbuild (軽量、Go 製) でビルド → `pkg/miniapp/static/dist/` に出力
- Go 側は `//go:embed static/dist` で embed
- `ORCH_ENABLED` はビルド時の環境変数注入 (`define`) で対応
- `Makefile` / `go:generate` でビルドステップを統合

**検討事項:**
- esbuild は Go 製でクロスコンパイル環境に影響しない
- Vite は機能豊富だが Node.js 依存が増える

---

### 3. Log viewer のフロントエンドテスト

`renderLogs()` (index.html 内の inline JS) のテストが皆無。
Fields 表示バグが検出されずに ship された前科あり。

**やること:**
- index.html から JS を外部ファイルに抽出 (タスク 2 と連動)
- `renderLogs()`, `renderFields()` 等の unit test を追加
- DOM 操作のテストは jsdom (vitest) or happy-dom
- CI で `pnpm test` を実行

**最低限のテストケース:**
- メッセージの正常レンダリング
- Fields の表示 (空/1件/複数件)
- サニタイズ (XSS 防止)
- ログのフィルタリング・ページネーション

---

## 完了基準

- 新規 JS/CSS ファイルの追加がルート登録なしで配信される
- `ORCH_ENABLED` の注入がテンプレートまたはビルド時変数で動作
- `renderLogs()` のユニットテストが CI で実行される
- `go build ./...` が変わらず動作 (embed パスの整合性)

## 作業報告 (2026-03-05)

- 実装完了: タスク 1 / 2 / 3
- 主要変更:
  - `pkg/miniapp/miniapp.go`
    - `//go:embed static` に変更
    - `/miniapp/` を `http.FileServer(http.FS(...))` で一括配信
    - `/miniapp` / `/miniapp/index.html` は `html/template` で描画し `OrchEnabled` を注入
    - 旧 `serveMapJS` 個別ルートを削除
  - フロントエンド分離
    - `pkg/miniapp/frontend/src/` に `app.js` / `styles.css` / `map.js` / `logs_view.js` を配置
    - `pkg/miniapp/static/index.html` はテンプレート + 外部アセット参照へ縮小
    - `pkg/miniapp/static/dist/` に `app.js` / `app.css` / `map.js` を生成
  - バンドル導線
    - `pkg/miniapp/frontend/build.mjs` + `package.json` を追加
    - `go generate ./...` で `bun run --cwd frontend build` が実行されるよう接続
  - Log viewer 強化
    - `logs_view.js` へレンダリングロジックを抽出
    - フィルタ/ページネーションを追加
    - `vitest + happy-dom` による unit test を追加 (`logs_view.test.js`)
  - CI 更新 (`.github/workflows/pr.yml`)
    - Bun セットアップを追加
    - frontend の `pnpm test` を追加

- 検証結果:
  - `bun run --cwd pkg/miniapp/frontend build`: 成功
  - `pnpm --dir pkg/miniapp/frontend test`: 成功
  - `go generate ./...`: 成功
  - `go test ./pkg/miniapp/...`: 成功
  - `go test ./...`: 一部既存の環境依存テストが Windows 環境で失敗（TASK-4差分外）
