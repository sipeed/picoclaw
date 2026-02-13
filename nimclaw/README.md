# NimClaw 🦞
Ultra-Efficient AI Assistant in Nim

NimClaw is a complete, high-performance clone of PicoClaw.

## Features
- Independent implementations of all channels (Telegram, Discord, QQ, Feishu, DingTalk, WhatsApp, MaixCam).
- Powerful toolset: filesystem, shell, web, cron, spawn.
- <10MB RAM footprint.
- Zero heavy dependencies for channels.

## Usage
1. `nimble install -y jsony cligen ws regex`
2. `nim c -d:release src/nimclaw.nim`
3. `./src/nimclaw onboard`
4. `./src/nimclaw agent`
