# Web 模块架构与打包流程

## 项目结构

```
web/
├── frontend/          → React SPA (Vite 构建)
├── backend/           → Go Web 服务器 (嵌入前端产物)
├── Makefile           → 统一构建入口
├── build/             → 最终输出目录
├── picoclaw-launcher.desktop
└── picoclaw-launcher.png
```

## 打包流程

`make build` 分两步执行：

### 1. 构建前端 (`pnpm build:backend`)

```bash
tsc -b && vite build --outDir ../backend/dist --emptyOutDir
```

- TypeScript 编译检查
- Vite 打包，产物直接输出到 `backend/dist/`

前端技术栈：React 19 + TanStack Router + Tailwind CSS v4 + Radix UI (shadcn)

### 2. 编译 Go 后端

```bash
CGO_ENABLED=0 go build -v -tags stdjson \
  -ldflags "-X ...Version=... -X ...GitCommit=... -s -w" \
  -o build/picoclaw-launcher ./backend/
```

- `backend/embed.go` 通过 `//go:embed all:dist` 将前端产物嵌入二进制
- `-s -w` 去掉调试信息，减小体积
- `-ldflags -X` 注入版本号、Git commit、构建时间
- macOS 上 `CGO_ENABLED=1`（systray 依赖），Linux/Windows 上 `CGO_ENABLED=0`

### 最终产物

单个 `build/picoclaw-launcher` 可执行文件，内嵌完整前端 SPA。

## WebSocket 链路

前端通过 WebSocket 与 picoclaw gateway 通信，web 后端作为反向代理中转：

```
前端 WebSocket
  → web 后端反向代理 (GET /pico/ws)
    → picoclaw gateway HTTP server
      → PicoChannel.ServeHTTP (pkg/channels/pico/pico.go:225)
        → handleWebSocket (pico.go:318) — 升级连接
          → authenticate (pico.go:383) — 验证 token
          → readLoop (pico.go:425) — 消息读取循环
          → pingLoop (pico.go:486) — 心跳保活
```

### 关键代码位置

| 组件 | 文件 |
|------|------|
| WebSocket 反向代理 | `web/backend/api/pico.go` |
| Gateway 注册 pico channel | `pkg/gateway/gateway.go` (import) |
| Pico Channel 实现 | `pkg/channels/pico/pico.go` |
| 前端代理配置 | `web/frontend/vite.config.ts` |

### API 端点

| 路由 | 用途 |
|------|------|
| `GET /pico/ws` | WebSocket 代理（转发到 gateway） |
| `POST /api/pico/token` | 生成/刷新 WebSocket 认证 token |
| `POST /api/pico/setup` | 初始化 Pico Channel |

## 开发模式

`make dev` 分别启动前后端，Vite dev server 通过代理转发请求：

```typescript
// vite.config.ts
proxy: {
  "/api": { target: "http://localhost:18800" },
  "/ws":  { target: "ws://localhost:18800", ws: true },
}
```

## 其他命令

```bash
make dev     # 启动前后端开发服务器
make build   # 构建生产包
make test    # 运行测试
make lint    # 代码检查
make clean   # 清理构建产物
```
