# PicoClaw Launcher — Wails v2 Migration

## Overview

Desktop GUI for PicoClaw using Wails v2, replacing the previous HTTP server + browser approach.

## Architecture

```
cmd/picoclaw-launcher/
├── main.go           # Wails bootstrap
├── app.go            # App struct with Go↔JS bindings
├── helpers.go        # Gateway process management
├── tray.go           # System tray integration
├── wails.json        # Wails project config
├── frontend/
│   ├── index.html    # Single-file UI (3 tabs)
│   └── wailsjs/      # Auto-generated bindings
└── internal/server/
    ├── setup.go      # Config setup helpers
    └── setup_chat.go # AI chat server
```

## UI: 3-Tab Layout

| Tab | Features |
|-----|----------|
| **Status** | Gateway start/stop/restart, real-time status |
| **Chat** | AI chat interface, first-run setup questionnaire |
| **Settings** | Model/channel/API key config, save/reload |

## Key Design Choices

- **Wails bindings** instead of HTTP: `window.go.main.App.GetConfig()` replaces `fetch('/api/config')`
- **System tray**: minimize-to-tray on close, right-click menu for gateway control
- **First-run setup**: guided overlay when no config exists → AI chat questionnaire
- **Single index.html**: intentional for v1 (no build toolchain), will split if complexity grows

## Build

```bash
go tool wails build -o picoclaw-launcher
# or via Taskfile:
task build-launcher
```

## Dependencies

- Wails v2 (declared as `tool` in go.mod, not linked into picoclaw binary)
- WebView2 (Windows, auto-installed), GTK/WebKit (Linux), WebKit (macOS)
