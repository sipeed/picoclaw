# C1 - System Context Diagram

PicoClaw as a multi-agent platform within its ecosystem.

```mermaid
C4Context
    title System Context - PicoClaw Multi-Agent Platform

    Person(user, "User", "Interacts via messaging channels or CLI")
    Person(dev, "Developer", "Configures agents, skills, and providers")

    System(picoclaw, "PicoClaw", "Multi-agent AI platform that routes user messages to specialized agents backed by multiple LLM providers")

    System_Ext(discord, "Discord", "Chat platform")
    System_Ext(telegram, "Telegram", "Chat platform")
    System_Ext(slack, "Slack", "Workspace messaging")
    System_Ext(whatsapp, "WhatsApp", "Messaging")
    System_Ext(cli, "CLI", "Direct terminal access")

    System_Ext(openai, "OpenAI API", "GPT models, Codex")
    System_Ext(anthropic, "Anthropic API", "Claude models")
    System_Ext(gemini, "Google Gemini", "Gemini models")
    System_Ext(openrouter, "OpenRouter", "Multi-model gateway")
    System_Ext(groq, "Groq", "Fast inference")
    System_Ext(ollama, "Ollama", "Local LLM")
    System_Ext(claude_cli, "Claude Code CLI", "Subprocess provider")
    System_Ext(codex_cli, "Codex CLI", "Subprocess provider")

    Rel(user, discord, "Sends messages")
    Rel(user, telegram, "Sends messages")
    Rel(user, slack, "Sends messages")
    Rel(user, whatsapp, "Sends messages")
    Rel(user, cli, "Direct input")
    Rel(dev, picoclaw, "Configures via config.json")

    Rel(discord, picoclaw, "Webhook/Bot events")
    Rel(telegram, picoclaw, "Bot API")
    Rel(slack, picoclaw, "Events API")
    Rel(whatsapp, picoclaw, "Webhook")
    Rel(cli, picoclaw, "Stdin/Stdout")

    Rel(picoclaw, openai, "HTTPS/REST")
    Rel(picoclaw, anthropic, "HTTPS/REST")
    Rel(picoclaw, gemini, "HTTPS/REST")
    Rel(picoclaw, openrouter, "HTTPS/REST")
    Rel(picoclaw, groq, "HTTPS/REST")
    Rel(picoclaw, ollama, "HTTP localhost")
    Rel(picoclaw, claude_cli, "Subprocess stdio")
    Rel(picoclaw, codex_cli, "Subprocess stdio")
```

## Key interactions

| Boundary | Protocol | Direction |
|----------|----------|-----------|
| User -> Channels | Platform-native (Discord bot, Telegram bot, etc.) | Inbound |
| Channels -> PicoClaw | Go channel bus (`pkg/bus`) | Internal |
| PicoClaw -> LLM Providers | HTTPS REST / Subprocess stdio | Outbound |
| Developer -> PicoClaw | `~/.picoclaw/config.json` + workspace files | Config |
