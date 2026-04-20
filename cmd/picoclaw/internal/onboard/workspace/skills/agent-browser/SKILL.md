---
name: agent-browser
description: "Browser automation via agent-browser CLI. Use when the user needs to navigate websites, fill forms, click buttons, take screenshots, extract data, or test web apps."
metadata: {"nanobot":{"emoji":"🌐","requires":{"bins":["agent-browser"]},"install":[{"id":"npm","kind":"npm","package":"agent-browser","global":true,"bins":["agent-browser"],"label":"Install agent-browser (npm)"}]}}
---

# Agent Browser

CLI browser automation via Chrome/Chromium CDP. Install: `npm i -g agent-browser && agent-browser install`.

**Before using this skill**, verify the tool is available by running `which agent-browser`. If the command is not found, tell the user that browser automation requires the `agent-browser` CLI and Chromium, which are only available in the heavy container image. Do not attempt to install it at runtime.

## Core Workflow

1. `agent-browser open <url>` — navigate
2. `agent-browser snapshot -i` — get interactive elements with refs (`@e1`, `@e2`, ...)
3. Interact using refs — `click @e1`, `fill @e2 "text"`
4. Re-snapshot after any navigation or DOM change — refs are invalidated

```bash
agent-browser open https://example.com/form
agent-browser snapshot -i
# @e1 [input] "Email", @e2 [input] "Password", @e3 [button] "Submit"
agent-browser fill @e1 "user@example.com"
agent-browser fill @e2 "secret"
agent-browser click @e3
agent-browser wait --load networkidle
agent-browser snapshot -i
```

Chain commands with `&&` when you don't need intermediate output:
```bash
agent-browser open https://example.com && agent-browser wait --load networkidle && agent-browser snapshot -i
```

## Commands

```bash
# Navigation
agent-browser open <url>
agent-browser close

# Snapshot
agent-browser snapshot -i                # Interactive elements with refs
agent-browser snapshot -s "#selector"    # Scope to CSS selector

# Interaction (use @refs from snapshot)
agent-browser click @e1
agent-browser fill @e2 "text"            # Clear + type
agent-browser type @e2 "text"            # Type without clearing
agent-browser select @e1 "option"
agent-browser check @e1
agent-browser press Enter
agent-browser scroll down 500

# Get info
agent-browser get text @e1
agent-browser get url
agent-browser get title

# Wait
agent-browser wait @e1                   # Wait for element
agent-browser wait --load networkidle    # Wait for network idle
agent-browser wait --url "**/dashboard"  # Wait for URL pattern
agent-browser wait --text "Welcome"      # Wait for text
agent-browser wait 2000                  # Wait ms

# Capture
agent-browser screenshot                 # Screenshot to temp dir
agent-browser screenshot --full          # Full page
agent-browser screenshot --annotate      # With numbered element labels ([N] -> @eN)
agent-browser pdf output.pdf

# Semantic locators (when refs unavailable)
agent-browser find text "Sign In" click
agent-browser find label "Email" fill "user@test.com"
agent-browser find role button click --name "Submit"
```

## Authentication

```bash
# Option 1: Import from user's running Chrome
agent-browser --auto-connect state save ./auth.json
agent-browser --state ./auth.json open https://app.example.com

# Option 2: Persistent profile
agent-browser --profile ~/.myapp open https://app.example.com/login
# ... login once, all future runs are authenticated

# Option 3: Session name (auto-save/restore)
agent-browser --session-name myapp open https://app.example.com/login
# ... login, close, next run state is restored

# Option 4: State file
agent-browser state save auth.json
agent-browser state load auth.json
```

## Iframes

Iframe content is inlined in snapshots. Interact with iframe refs directly — no frame switch needed.

## Parallel Sessions

```bash
agent-browser --session s1 open https://site-a.com
agent-browser --session s2 open https://site-b.com
agent-browser session list
```

## JavaScript Eval

```bash
agent-browser eval 'document.title'

# Complex JS — use --stdin to avoid shell quoting issues
agent-browser eval --stdin <<'EVALEOF'
JSON.stringify(Array.from(document.querySelectorAll("a")).map(a => a.href))
EVALEOF
```

## Cleanup

Always close sessions when done:
```bash
agent-browser close
agent-browser --session s1 close
```
