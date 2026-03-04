# TASKS-3: Session DAG (SQLite Store)

セッション管理を JSON ファイルベースの線形スライスから SQLite ベースの Turn DAG に移行する。
他トラックへの依存なし — LegacyAdapter で既存コードとの後方互換を維持しながら段階移行。

## 設計原則

1. **セッション内は線形、セッション間が DAG** — per-message DAG は過剰。ターン間の因果は順序で十分。
2. **SQLite single-file backend** — microSD 書き込み最小化、WAL モードでクラッシュ耐性。
3. **サブエージェント報告は user role** — system role の権威性バイアスを回避。conductor が評価・反論できる。
4. **"merge" は特別な操作ではない** — 報告を受けて会話を続ける通常のターン。

## DAG 構造

```
Conductor session:  turn1 → turn2 → turn3 → report(scout-1) → turn4 → report(coder-1) → turn5
                               ↓ fork                          ↑ report
Scout-1 session:              turn1 → turn2 → turn3 ──────────┘
                                        ↓ fork
Coder-1 session:                       turn1 → turn2 → turn3 ──────────┘
```

## SQLite Schema

```sql
CREATE TABLE sessions (
    key          TEXT PRIMARY KEY,
    parent_key   TEXT REFERENCES sessions(key),
    fork_turn_id TEXT,
    status       TEXT NOT NULL DEFAULT 'active',
    label        TEXT NOT NULL DEFAULT '',
    summary      TEXT NOT NULL DEFAULT '',
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL
);

CREATE TABLE turns (
    id          TEXT PRIMARY KEY,
    session_key TEXT NOT NULL REFERENCES sessions(key) ON DELETE CASCADE,
    seq         INTEGER NOT NULL,
    kind        INTEGER NOT NULL DEFAULT 0,
    messages    TEXT NOT NULL,
    origin_key  TEXT,
    summary     TEXT,
    author      TEXT NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    meta        TEXT,
    UNIQUE(session_key, seq)
);

CREATE INDEX idx_turns_session_seq ON turns(session_key, seq);
CREATE INDEX idx_sessions_parent ON sessions(parent_key);
```

## Go Interface

```go
type TurnKind int

const (
    TurnNormal    TurnKind = iota
    TurnReport
    TurnForkPoint
)

type Turn struct {
    ID        string
    Seq       int
    Kind      TurnKind
    Messages  []providers.Message
    OriginKey string
    Summary   string
    Author    string
    CreatedAt time.Time
    Meta      map[string]string
}

type SessionInfo struct {
    Key        string
    ParentKey  string
    ForkTurnID string
    Status     string
    Label      string
    Summary    string
    TurnCount  int
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type SessionStore interface {
    Create(key string, opts *CreateOpts) error
    Get(key string) (*SessionInfo, error)
    List(filter *ListFilter) ([]*SessionInfo, error)
    SetStatus(key, status string) error
    SetSummary(key, summary string) error
    Delete(key string) error
    Children(key string) ([]*SessionInfo, error)

    Append(sessionKey string, turn *Turn) error
    Turns(sessionKey string, sinceSeq int) ([]*Turn, error)
    LastTurn(sessionKey string) (*Turn, error)
    TurnCount(sessionKey string) (int, error)

    Compact(sessionKey string, upToSeq int, summary string) error
    Fork(parentKey, childKey string, opts *CreateOpts) error

    Prune(olderThan time.Duration) (int, error)
    Close() error
}
```

## 高レベルラッパー

```go
type SessionGraph struct {
    store   SessionStore
    buffers sync.Map // sessionKey → *turnBuffer
    views   sync.Map // sessionKey → *cachedView
}

func (g *SessionGraph) Messages(sessionKey string) ([]providers.Message, error)
func (g *SessionGraph) BeginTurn(sessionKey string, kind TurnKind) *TurnWriter

type TurnWriter struct { ... }
func (tw *TurnWriter) Add(msg providers.Message)
func (tw *TurnWriter) SetOrigin(sessionKey string)
func (tw *TurnWriter) Commit() error
func (tw *TurnWriter) Discard()
```

## サブエージェント報告フロー

```
1. conductor が spawn → store.Fork(conductorSession, subagentSession)
2. subagent 実行中 → store.Append(subagentSession, Turn{Kind: TurnNormal, ...})
3. subagent 完了 → store.SetStatus(subagentSession, "completed")
4. conductor 側に report ターン:
     tw := graph.BeginTurn(conductorSession, TurnReport)
     tw.SetOrigin(subagentSession)
     tw.Add(Message{Role: "user", Content: "[scout-1] 調査結果..."})
5. conductor が応答:
     tw.Add(Message{Role: "assistant", Content: "なるほど、JWTで十分..."})
     tw.Commit()
```

---

## タスク一覧

### Phase 0: SQLite SessionStore + LegacyAdapter ✅

既存動作を維持したまま裏側を差し替える。

1. ✅ **SQLite SessionStore 実装**
   - `pkg/session/sqlite.go`: SessionStore interface の SQLite 実装
   - WAL モード、`modernc.org/sqlite` (CGO なし、ARM クロスコンパイル容易)
   - schema migration (CREATE TABLE IF NOT EXISTS)

2. ✅ **LegacyAdapter 実装**
   - `pkg/session/legacy_adapter.go`
   - 既存の `GetHistory` / `SetHistory` / `AddMessage` / `MarkDirty` を SessionStore 経由で実装
   - 既存テスト全パス + テーブル駆動で JSON/SQLite 両方を同一アサーションで検証

3. ✅ **JSON → SQLite lazy migration**
   - `pkg/session/migrate.go`: 起動時に `sessions/*.json` を検出 → SQLite に import → `.json.migrated` にリネーム
   - 個別ファイル失敗はログ警告して続行 (次回起動で再試行)

4. ✅ **AgentLoop 配線**
   - `pkg/agent/instance.go`: `Sessions` 型を `*LegacyAdapter` に変更
   - `loop.go` / `loop_test.go` は変更なし — メソッドシグネチャ同一

---

### Phase 1: Fork/Report ターン導入

5. **Fork 操作**
   - `SessionStore.Fork()` 実装
   - 親セッションに `TurnForkPoint` を追記、子セッションを `parent_key` 付きで作成

6. **Report ターン**
   - `TurnReport` の Append/Turns/Messages 対応
   - `origin_key` でどのセッションの報告かを追跡
   - user role でメッセージ格納 (system role 禁止)

7. **サブエージェントセッション永続化**
   - サブエージェント実行中のターンを SQLite に記録
   - 完了後にセッション status を "completed" に変更

---

### Phase 2: AgentLoop 直接移行

8. **SessionGraph 直接呼び出し**
   - `runAgentLoop()` を `SessionGraph.BeginTurn()` / `TurnWriter` 経由に変更
   - `processSystemMessage()` を Report ターン生成に変更
   - LegacyAdapter 廃止

9. **Compaction**
   - 古いターンの messages を空にして summary で置換
   - context window 管理と連動

10. **セッションライフサイクル**
    - `Delete(key)` + TTL エビクション (Prune)
    - `sessionLocks sync.Map` の GC
    - 起動時の遅延ロード (SQLite なので自然に実現)

---

### Phase 3: UI & Commands

11. **Mini App セッショングラフ可視化**
    - WebSocket で session DAG 構造を配信
    - fork/report 関係をグラフとして描画

12. **CLI コマンド**
    - `/session list` — アクティブセッション一覧
    - `/session fork` — 現在のセッションを fork
    - `/session graph` — DAG 構造をテキスト表示

---

## 完了基準

- `go test ./...` 全パス
- 既存の JSON セッションが SQLite に自動マイグレーション
- Phase 0 完了時点で既存動作に変化なし (LegacyAdapter 透過)
- サブエージェント報告が user role ターンとして記録
- conductor が報告を個別に評価・応答するフロー動作
- microSD 書き込み頻度が既存比で削減
