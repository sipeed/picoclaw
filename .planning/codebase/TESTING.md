# Testing Patterns

**Analysis Date:** 2026-04-10

## Test Framework

**Runner:** Go 标准 `testing` 包。

**Assertion libraries:**
- `github.com/stretchr/testify/assert` -- 非致命断言
- `github.com/stretchr/testify/require` -- 致命断言

**无外部测试框架** -- 纯 `testing` + `testify`。

## 运行测试

```bash
make test                                    # 运行所有测试
make check                                   # deps + fmt + vet + test
go test -run TestName -v ./pkg/session/      # 运行单个测试
go test -bench=. -benchmem -run='^$' ./...  # 仅运行 benchmark
cd web && make test                          # Web backend 测试
go test -tags goolm,stdjson ./...           # CI 测试命令
```

**必需的 Build tags:** `goolm,stdjson`

## 测试文件组织

- **位置:** 与源码同目录同包（非 `_test` 包），`.golangci.yaml` 第25行禁用了 `testpackage`
- **示例:** `pkg/seahorse/store_test.go` (`package seahorse`), `cmd/picoclaw/internal/auth/command_test.go` (`package auth`)
- **命名:** `*_test.go` 后缀
- **数量:** 约 240 个测试文件

## 测试结构模式

**表驱动测试** (标准模式):
```go
func TestShouldEnableLauncherFileLogging(t *testing.T) {
    tests := []struct {
        name          string
        enableConsole bool
        debug         bool
        want          bool
    }{
        {name: "gui mode", enableConsole: false, debug: false, want: true},
        {name: "console mode", enableConsole: true, debug: false, want: false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := fn(tt.enableConsole, tt.debug); got != tt.want {
                t.Fatalf("... = %t, want %t", got, tt.want)
            }
        })
    }
}
```

**Cobra 命令测试** (`cmd/picoclaw/internal/*/command_test.go`):
```go
func TestNewAuthCommand(t *testing.T) {
    cmd := NewAuthCommand()
    require.NotNil(t, cmd)
    assert.Equal(t, "auth", cmd.Use)
    allowedCommands := []string{"login", "logout", "status", "models", "weixin", "wecom"}
    subcommands := cmd.Commands()
    assert.Len(t, subcommands, len(allowedCommands))
    for _, subcmd := range subcommands {
        found := slices.Contains(allowedCommands, subcmd.Name())
        assert.True(t, found, "unexpected subcommand %q", subcmd.Name())
    }
}
```

**HTTP 处理器测试** (`web/backend/api/*_test.go`):
```go
mux := http.NewServeMux()
RegisterLauncherAuthRoutes(mux, LauncherAuthRouteOpts{DashboardToken: tok, SessionCookie: sess})
t.Run("status_unauthenticated", func(t *testing.T) {
    rec := httptest.NewRecorder()
    mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/status", nil))
    if rec.Code != http.StatusOK { t.Fatalf("status code = %d", rec.Code) }
})
```

**数据库测试** (`pkg/seahorse/store_test.go`):
```go
func openTestStore(t *testing.T) *Store {
    t.Helper()
    db := openTestDB(t)
    if err := runSchema(db); err != nil { t.Fatalf("migration: %v", err) }
    return &Store{db: db}
}
```

## Mock 策略

**手写 mock 结构体**，无 gomock/mockgen/testify/mock。

**`pkg/tools/subagent_tool_test.go`:**
```go
type MockLLMProvider struct { lastOptions map[string]any }
func (m *MockLLMProvider) Chat(ctx, messages, tools, model, options) (*providers.LLMResponse, error) {
    m.lastOptions = options
    return &providers.LLMResponse{Content: "Task completed"}, nil
}
```

**回调函数 mock** (`pkg/seahorse/short_compaction_test.go`):
```go
var mockCompleteFn CompleteFn = func(ctx, prompt, opts) (string, error) {
    return "Mock summary of the conversation segment.", nil
}
```

**Mock 对象:** LLM providers, CompleteFn, 时间 (`l.now = func() time.Time { return t0 }`)
**不 Mock 对象:** 数据库 (使用 in-memory SQLite), HTTP handlers (`httptest`), 文件系统 (`t.TempDir()`)

## Fixtures 和测试数据

- 内存数据库: `openTestDB(t)` + `runSchema(db)`, 每个测试独立
- 环境变量: `t.Setenv()` 自动清理
- Testdata: `pkg/channels/telegram/testdata/md2_all_formats.txt`
- 测试常量: 硬编码在测试文件中

## Benchmark 测试

**位置:** `cmd/membench/` 和 `pkg/seahorse/short_bench_test.go`

**模式:**
```go
func newBenchStore(b *testing.B) (*Store, func()) {
    b.Helper()
    db, err := sql.Open("sqlite", ":memory:")
    if err != nil { b.Fatalf("open test db: %v", err) }
    if err := runSchema(db); err != nil { db.Close(); b.Fatalf("migration: %v", err) }
    return &Store{db: db}, func() { db.Close() }
}
func BenchmarkIngest_SingleMessage(b *testing.B) {
    s, cleanup := newBenchStore(b); defer cleanup()
    b.ResetTimer()
    for i := 0; i < b.N; i++ { s.AddMessage(ctx, convID, "user", "Test", 15) }
}
```

**运行:** `go test -bench=. -benchmem -run='^$' ./pkg/seahorse/`

## CI 配置

**PR workflow** (`.github/workflows/pr.yml`):
- Lint: `golangci-lint-action@v9` (v2.10.1), `--build-tags=goolm,stdjson`
- 安全检查: `govulncheck`
- 测试: `go test -tags goolm,stdjson ./...`
- 所有任务在 `ubuntu-latest` 上运行

**无覆盖率报告** -- CI 中没有 codecov 或 `-coverprofile`

## 通用模式

- `t.Helper()` 在测试工具函数中
- `t.TempDir()` 用于临时目录
- `t.Fatalf()` 致命错误, `t.Errorf()` 断言失败
- OS 特定跳过: `t.Skip("user environment variables only apply on Linux")`
- 并发测试: 直接操作互斥锁 (`manager.mu.Lock()`)
- 错误测试: 检查 `err == nil` 或 `result.IsError`

---

*Testing analysis: 2026-04-10*
