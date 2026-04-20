# PicoClaw Web

`web/` contains the standalone WebUI launcher for PicoClaw.
It is not just a frontend: it is a small launcher service that bundles a React dashboard, exposes a backend API, manages launcher authentication, and starts or attaches to the `picoclaw gateway` process.

![PicoClaw Launcher](./picoclaw-launcher.png)

## What This Directory Provides

- A browser-based chat UI backed by the Pico channel WebSocket proxy.
- A dashboard for models, credentials, channels, agent tools, skills, logs, and runtime settings.
- A launcher process that can auto-open the browser, show a system tray menu, and persist launcher-specific settings.
- A controlled way to start, stop, restart, and inspect the `picoclaw gateway` subprocess.
- A single-binary deployment target where the frontend is embedded into the Go backend.

## Architecture

This directory is a small monorepo:

- `backend/`
  - Go HTTP server and launcher runtime.
  - Serves REST APIs, authentication endpoints, channel helper flows, and the Pico WebSocket reverse proxy.
  - Embeds compiled frontend assets from `backend/dist`.
- `frontend/`
  - Vite + React 19 + TanStack Router SPA.
  - Provides the launcher dashboard and chat UI.

At runtime the launcher and the main PicoClaw engine are separate processes:

1. The launcher starts the web backend on port `18800` by default.
2. The launcher serves the dashboard and handles dashboard authentication.
3. When allowed, it starts or attaches to `picoclaw gateway -E`.
4. The frontend talks only to the launcher backend.
5. The launcher proxies chat traffic to the gateway through `/pico/ws`.

## Dashboard Capabilities

The current frontend exposes these major pages and flows:

- `/`
  - Chat UI with session history, default model selection, and Pico channel messaging.
- `/models`
  - Add, edit, delete, and set the default model.
  - Supports API-key models, OAuth-backed models, and local/CLI-backed models.
- `/credentials`
  - Manage provider credentials.
  - Current built-in flows: OpenAI, Anthropic, and Google Antigravity.
- `/channels/*`
  - Configure supported channels from a shared catalog.
  - Current catalog: `weixin`, `telegram`, `discord`, `slack`, `feishu`, `dingtalk`, `line`, `qq`, `onebot`, `wecom`, `whatsapp`, `whatsapp_native`, `pico`, `maixcam`, `matrix`, `irc`.
  - Includes QR-based binding helpers for WeChat and WeCom.
- `/agent/skills`
  - Browse built-in, global, and workspace skills.
  - Import Markdown skills into the workspace and delete workspace-owned skills.
- `/agent/tools`
  - View tool availability and enable or disable tool switches through config-backed APIs.
- `/config`
  - Edit agent defaults, exec controls, cron controls, heartbeat, device monitoring, launcher networking, and launch-at-login settings.
- `/logs`
  - View the in-memory gateway log buffer and clear it.

The UI currently supports English and Simplified Chinese, plus light and dark themes.

## Runtime Behavior

### Config Resolution

The launcher uses the same PicoClaw config file as the main binary.

- Default app config path: `~/.picoclaw/config.json`
- Override with environment variable: `PICOCLAW_CONFIG`
- Override with a positional CLI argument: `picoclaw-launcher /path/to/config.json`

Launcher-only settings are stored beside that app config:

- File name: `launcher-config.json`
- Default location: `~/.picoclaw/launcher-config.json`

That file currently stores:

- `port`
- `public`
- `allowed_cidrs`

If `-port` or `-public` are passed explicitly, the CLI flag wins for that run.
If they are omitted, stored launcher settings are used.

### First-Run Onboarding

If the target config file does not exist, the launcher tries to bootstrap it automatically by running:

```bash
picoclaw onboard
```

The launcher looks for the main PicoClaw binary in this order:

1. `PICOCLAW_BINARY`
2. A `picoclaw` binary in the same directory as the launcher
3. `picoclaw` from `PATH`

If onboarding or gateway startup cannot find the main binary, set `PICOCLAW_BINARY` explicitly.

### Gateway Management

The launcher manages `picoclaw gateway -E`.

On startup it tries to auto-start or attach to the gateway, but only when startup preconditions pass. In the current code, the main checks are:

- a default model is configured
- the default model entry is valid
- the default model has usable credentials
- local/runtime-probed models are reachable

When a gateway process is started by the launcher, the launcher:

- captures stdout and stderr into an in-memory ring buffer
- tracks transient states such as `starting`, `restarting`, and `stopping`
- marks restart-required when the default model or enabled tool set changed since boot
- ensures the Pico channel is configured before startup

### Launcher Authentication

The dashboard is protected by a launcher access token.

- If `PICOCLAW_LAUNCHER_TOKEN` is set, that token is used.
- Otherwise a random token is generated for each launcher process.
- The browser auto-open URL includes `?token=...` so local launches can sign in automatically.
- Manual login uses `/launcher-login`.
- API clients may also authenticate with `Authorization: Bearer <token>`.

Where users can retrieve the token depends on launch mode:

- Console mode: printed to stdout
- GUI mode: available through the tray menu on supported builds
- GUI mode without stdout:
  - random per-run tokens are written to the launcher log
  - default log path: `~/.picoclaw/logs/launcher.log`
  - if `PICOCLAW_HOME` is set, use `$PICOCLAW_HOME/logs/launcher.log`
  - env-pinned tokens are not reprinted there; the log only notes that `PICOCLAW_LAUNCHER_TOKEN` is in use

### Network Exposure

By default the launcher listens on:

```text
127.0.0.1:18800
```

With `-public` or `public: true`, it listens on all interfaces:

```text
0.0.0.0:18800
```

When public access is enabled:

- the launcher can still protect the dashboard with the access token
- optional `allowed_cidrs` can restrict which client IP ranges may connect
- the gateway host is overridden so remote clients can still use the launcher-managed proxy paths

## Build And Run

### Prerequisites

- Go `1.25+`
- Node.js 20.19+ or 22.13+
- `pnpm`

On macOS, the `web` Makefile enables `CGO_ENABLED=1` so tray-enabled launcher builds work as expected.
On Darwin or FreeBSD without cgo, the launcher falls back to headless mode without a tray.

If you want to prepare the frontend workspace manually, you can still install dependencies yourself:

```bash
cd frontend
pnpm install
```

### Recommended Development Workflow

From the `web/` directory:

```bash
make dev
```

This does three things:

1. Builds `../build/picoclaw` for launcher development.
2. Starts the Go backend with `PICOCLAW_BINARY` pointing at that binary.
3. Starts the Vite frontend dev server.

Use this when you want the full launcher flow during development.

### Run Frontend And Backend Separately

```bash
make dev-frontend
make dev-backend
```

Notes:

- `dev-frontend` runs the Vite server.
- `dev-backend` runs the Go backend only.
- The Vite dev server proxies `/api` to `http://localhost:18800`.
- Chat WebSocket URLs are generated by the backend, so the frontend does not hardcode gateway addresses.
- Running `dev-backend` alone is mainly useful for backend work or when `backend/dist` already contains a built frontend.

### Build The Standalone Launcher Binary

From `web/`:

```bash
make build
```

This:

1. Installs frontend dependencies when needed.
2. Builds the frontend into `backend/dist`.
3. Embeds those assets into the Go backend.
4. Produces `build/picoclaw-launcher`.

Override the output path if needed:

```bash
make build OUTPUT=/tmp/picoclaw-launcher
```

From the repository root you can also use:

```bash
make build-launcher
```

That writes the platform-specific launcher to:

```text
build/picoclaw-launcher-<platform>-<arch>
```

and refreshes the `build/picoclaw-launcher` symlink.

### Frontend-Only Builds

For frontend work there are two useful package scripts:

```bash
cd frontend
pnpm build
pnpm build:backend
```

- `pnpm build` writes a normal Vite build to `frontend/dist`
- `pnpm build:backend` writes the embeddable build to `../backend/dist`

### Run The Built Launcher

Examples:

```bash
./build/picoclaw-launcher
./build/picoclaw-launcher -console
./build/picoclaw-launcher -public
./build/picoclaw-launcher -port 19999 /path/to/config.json
```

Current launcher flags:

- `-port`
- `-public`
- `-no-browser`
- `-lang`
- `-console`

## Make Targets

From `web/`:

```bash
make dev
make dev-frontend
make dev-backend
make build
make build-frontend
make test
make lint
make clean
```

What they do today:

- `make build-frontend`
  - Runs `pnpm install --frozen-lockfile` when dependencies are missing or stale.
  - Builds the embeddable frontend into `backend/dist`.
- `make test`
  - Runs backend Go tests.
  - Runs frontend `pnpm lint`.
- `make lint`
  - Runs backend `go vet`.
  - Runs frontend `pnpm check`.
  - `pnpm check` currently formats files with Prettier and fixes lint issues with ESLint, so this target can modify your working tree.
- `make clean`
  - Removes `frontend/dist`, `backend/dist`, and `build/`, then recreates `backend/dist/.gitkeep`.

## Directory Layout

```text
web/
├── backend/
│   ├── api/             # REST API handlers and launcher runtime endpoints
│   ├── launcherconfig/  # launcher-config.json load/save/validation
│   ├── middleware/      # auth, content type, logging, CIDR allowlist
│   ├── model/           # Go data structures and logic wrappers
│   ├── utils/           # runtime helpers, onboarding, browser launch
│   ├── winres/          # Windows application resources
│   └── dist/            # embedded frontend build output
├── frontend/
│   ├── src/api/         # browser API clients
│   ├── src/components/  # UI pages and shared components
│   ├── src/features/    # feature-specific state, controllers, and protocol helpers
│   ├── src/hooks/       # shared React hooks
│   ├── src/i18n/        # internationalization language packs
│   ├── src/lib/         # generic library utilities
│   ├── src/routes/      # TanStack file routes
│   ├── src/store/       # global state management
│   └── vite.config.ts   # dev server and build config
├── Makefile
└── README.md
```

## Troubleshooting

### You have to sign in again after the launcher restarts

Existing dashboard sessions do not survive launcher restarts.
That is expected: each launcher process generates a new signed session value, so old cookies become invalid.

To make re-login easier, set a stable token:

```bash
export PICOCLAW_LAUNCHER_TOKEN="replace-with-a-long-random-token"
```

Notes:

- a stable token does not preserve the old cookie-based session by itself
- when the launcher opens the browser automatically, it appends `?token=...` and signs in again automatically
- if you reopen the dashboard manually, use the same stable token on `/launcher-login`

### "Start Gateway" stays disabled

The launcher only allows gateway startup when the configured default model is usable.
Check these in the dashboard:

- a default model is selected
- the model has credentials or OAuth state
- local models such as Ollama or vLLM are reachable

### The launcher cannot find `picoclaw`

Set the main binary explicitly:

```bash
export PICOCLAW_BINARY=/absolute/path/to/picoclaw
```

This affects onboarding and gateway subprocess startup.

### The backend starts but the UI is blank in development

Use `make dev` for the normal workflow.
If you run only `make dev-backend`, either run `make dev-frontend` alongside it or build the embedded frontend first with `make build-frontend`.

## Related Docs

- Main project overview: [`../README.md`](../README.md)
- Configuration guide: [`../docs/guides/configuration.md`](../docs/guides/configuration.md)
- Providers: [`../docs/guides/providers.md`](../docs/guides/providers.md)
- Troubleshooting: [`../docs/operations/troubleshooting.md`](../docs/operations/troubleshooting.md)
- Official docs site: [docs.picoclaw.io](https://docs.picoclaw.io)
